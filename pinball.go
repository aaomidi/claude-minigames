package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	pbW = 300
	pbH = 520
)

// Physics
const (
	pbBallR    = 6.0
	pbGrav     = 0.30
	pbMaxSpd   = 14.0
	pbSub      = 5    // substeps
	pbDamp     = 0.55 // wall bounce damping
	pbBumpKick = 5.5
)

// Flipper
const (
	pbFLen   = 52.0
	pbFThick = 7.0
	pbFRest  = 0.50  // rest angle (tip down)
	pbFUp    = -0.40 // active angle (tip up)
	pbFSpd   = 0.28
	pbFY     = 455.0
	pbFLX    = 80.0  // left flipper pivot X
	pbFRX    = 220.0 // right flipper pivot X
)

type pbSt int

const (
	pbStLaunch pbSt = iota
	pbStPlay
	pbStLost
	pbStOver
)

type pbV2 struct{ x, y float64 }

type pbBump struct {
	p      pbV2
	r      float64
	pts    int
	flash  int
}

type pbFlip struct {
	piv pbV2
	dir float64 // +1=left, -1=right
	ang float64
}

func (f *pbFlip) tip() pbV2 {
	return pbV2{
		f.piv.x + math.Cos(f.ang)*pbFLen*f.dir,
		f.piv.y + math.Sin(f.ang)*pbFLen,
	}
}

type pbSeg struct {
	a, b pbV2
	t    float64 // thickness
}

// Ball trail position for fading trail effect
type pbTrailPos struct {
	x, y float64
}

// Bumper ripple effect
type pbRipple struct {
	x, y   float64
	radius float64
	life   int
}

// Background sparkle
type pbSparkle struct {
	x, y  float64
	phase float64
}

type Pinball struct {
	goBack   bool
	state    pbSt
	ball     pbV2
	vel      pbV2
	bumps    []pbBump
	lf, rf   pbFlip
	walls    []pbSeg
	score    int
	lives    int
	pow      int // launch power (frames held)
	msg      string
	msgT     int
	ticks    int

	// Visual: ball trail
	trail    [10]pbTrailPos
	trailIdx int

	// Visual: bumper ripples
	ripples  []pbRipple

	// Visual: background sparkles (static positions)
	sparkles [20]pbSparkle

	// Visual: previous flipper angles for motion blur
	lfPrevAng float64
	rfPrevAng float64

	// Audio
	bgm        *audio.Player
	sfxBoing   []byte
	sfxFlip    []byte
	sfxLaunch  []byte
	audioInit  bool
	lfWasUp    bool // track flipper state changes for SFX
	rfWasUp    bool
}

func newPinball() *Pinball {
	p := &Pinball{}
	p.pbInit()
	return p
}

func (p *Pinball) pbInit() {
	p.score = 0
	p.lives = 3
	p.msg = ""
	p.msgT = 0

	p.lf = pbFlip{pbV2{pbFLX, pbFY}, 1, pbFRest}
	p.rf = pbFlip{pbV2{pbFRX, pbFY}, -1, pbFRest}

	w := 10.0 // wall inset
	p.bumps = []pbBump{
		{pbV2{100, 110}, 18, 200, 0},
		{pbV2{200, 110}, 18, 200, 0},
		{pbV2{150, 80}, 15, 300, 0},
		{pbV2{80, 210}, 16, 100, 0},
		{pbV2{220, 210}, 16, 100, 0},
		{pbV2{150, 195}, 20, 250, 0},
		{pbV2{115, 310}, 13, 75, 0},
		{pbV2{185, 310}, 13, 75, 0},
	}

	// Gutter walls only — no slingshots (they create dead zones)
	p.walls = []pbSeg{
		{pbV2{w, 380}, pbV2{pbFLX, pbFY}, 5},
		{pbV2{pbW - w, 380}, pbV2{pbFRX, pbFY}, 5},
	}

	// Initialize background sparkles
	for i := range p.sparkles {
		p.sparkles[i] = pbSparkle{
			x:     20 + rand.Float64()*float64(pbW-40),
			y:     20 + rand.Float64()*float64(pbH-40),
			phase: rand.Float64() * math.Pi * 2,
		}
	}

	p.pbResetBall()
}

func (p *Pinball) pbResetBall() {
	// Ball starts above center of playfield — simple, reliable
	p.ball = pbV2{float64(pbW) / 2, 400}
	p.vel = pbV2{0, 0}
	p.state = pbStLaunch
	p.pow = 0
}

// ---------------------------------------------------------------------------
// Audio initialization
// ---------------------------------------------------------------------------

func (p *Pinball) pbInitAudio() {
	if p.audioInit {
		return
	}
	p.audioInit = true

	// --- BGM: Funky groovy arcade music, ~130bpm, major key, bouncy ---
	// Sine+triangle melody - fun bouncy major key
	melody := [][2]int{
		// A section - bouncy G major groove
		{67, 1}, {71, 1}, {74, 1}, {76, 1},
		{79, 2}, {76, 1}, {74, 1},
		{71, 1}, {74, 1}, {76, 1}, {79, 1},
		{81, 1}, {79, 1}, {76, 2},
		{74, 1}, {72, 1}, {71, 1}, {69, 1},
		{67, 2}, {69, 1}, {71, 1},
		{74, 1}, {76, 1}, {74, 1}, {71, 1},
		{67, 2}, {0, 2},
		// A' - higher energy response
		{79, 1}, {81, 1}, {83, 1}, {86, 1},
		{83, 2}, {81, 1}, {79, 1},
		{76, 1}, {79, 1}, {81, 1}, {83, 1},
		{86, 2}, {83, 2},
		{81, 1}, {79, 1}, {76, 1}, {74, 1},
		{71, 2}, {74, 1}, {76, 1},
		{79, 1}, {76, 1}, {74, 1}, {71, 1},
		{67, 2}, {0, 2},
		// B section - C major bridge, wide intervals
		{72, 1}, {76, 1}, {79, 1}, {84, 1},
		{81, 2}, {79, 1}, {76, 1},
		{72, 1}, {74, 1}, {76, 1}, {79, 1},
		{81, 2}, {84, 2},
		{81, 1}, {79, 1}, {76, 1}, {74, 1},
		{72, 2}, {71, 1}, {69, 1},
		{67, 1}, {71, 1}, {74, 1}, {79, 1},
		{76, 2}, {0, 2},
		// C section - Em surprise, funky
		{64, 1}, {67, 1}, {71, 1}, {76, 1},
		{74, 2}, {71, 1}, {67, 1},
		{64, 1}, {67, 1}, {69, 1}, {71, 1},
		{74, 1}, {76, 1}, {79, 2},
		{76, 1}, {74, 1}, {71, 1}, {67, 1},
		{69, 2}, {71, 1}, {74, 1},
		{76, 1}, {74, 1}, {71, 1}, {67, 1},
		{67, 2}, {0, 2},
	}
	melodyPCM := generateTrack(melody, sineWave, 130, 0.30)

	// Triangle bass - funky groove
	bass := [][2]int{
		// A section
		{43, 1}, {0, 1}, {43, 1}, {47, 1},
		{48, 1}, {0, 1}, {48, 1}, {43, 1},
		{43, 1}, {0, 1}, {47, 1}, {43, 1},
		{45, 1}, {0, 1}, {45, 1}, {43, 1},
		{38, 1}, {0, 1}, {43, 1}, {0, 1},
		{43, 2}, {45, 1}, {47, 1},
		{50, 1}, {48, 1}, {47, 1}, {43, 1},
		{43, 2}, {0, 2},
		// A' bass
		{43, 1}, {0, 1}, {47, 1}, {50, 1},
		{48, 1}, {0, 1}, {48, 1}, {47, 1},
		{43, 1}, {0, 1}, {50, 1}, {47, 1},
		{45, 1}, {0, 1}, {47, 1}, {43, 1},
		{43, 1}, {0, 1}, {45, 1}, {0, 1},
		{47, 2}, {45, 1}, {43, 1},
		{50, 1}, {48, 1}, {47, 1}, {45, 1},
		{43, 2}, {0, 2},
		// B section - C major bass
		{48, 1}, {0, 1}, {48, 1}, {52, 1},
		{48, 1}, {0, 1}, {43, 1}, {48, 1},
		{48, 1}, {0, 1}, {52, 1}, {48, 1},
		{45, 1}, {0, 1}, {47, 1}, {48, 1},
		{48, 1}, {0, 1}, {43, 1}, {0, 1},
		{48, 2}, {47, 1}, {45, 1},
		{43, 1}, {47, 1}, {50, 1}, {43, 1},
		{43, 2}, {0, 2},
		// C section - Em bass
		{40, 1}, {0, 1}, {40, 1}, {43, 1},
		{40, 1}, {0, 1}, {43, 1}, {40, 1},
		{40, 1}, {0, 1}, {45, 1}, {43, 1},
		{40, 1}, {43, 1}, {45, 1}, {43, 1},
		{40, 1}, {0, 1}, {43, 1}, {0, 1},
		{45, 2}, {43, 1}, {40, 1},
		{43, 1}, {40, 1}, {38, 1}, {43, 1},
		{43, 2}, {0, 2},
	}
	bassPCM := generateTrack(bass, triangleWave, 130, 0.25)

	// Square arpeggio for richness
	arp := [][2]int{
		// A section arp
		{79, 1}, {83, 1}, {86, 1}, {83, 1},
		{84, 1}, {88, 1}, {84, 1}, {81, 1},
		{79, 1}, {83, 1}, {86, 1}, {83, 1},
		{81, 1}, {84, 1}, {81, 1}, {79, 1},
		{74, 1}, {79, 1}, {74, 1}, {71, 1},
		{67, 2}, {71, 1}, {74, 1},
		{76, 1}, {79, 1}, {76, 1}, {74, 1},
		{71, 2}, {0, 2},
		// A' arp - higher register
		{86, 1}, {91, 1}, {86, 1}, {83, 1},
		{88, 1}, {91, 1}, {88, 1}, {84, 1},
		{86, 1}, {91, 1}, {86, 1}, {83, 1},
		{84, 1}, {88, 1}, {84, 1}, {81, 1},
		{79, 1}, {83, 1}, {79, 1}, {76, 1},
		{74, 2}, {76, 1}, {79, 1},
		{81, 1}, {83, 1}, {81, 1}, {79, 1},
		{76, 2}, {0, 2},
		// B section arp - C major
		{84, 1}, {88, 1}, {91, 1}, {88, 1},
		{84, 1}, {81, 1}, {84, 1}, {88, 1},
		{84, 1}, {86, 1}, {88, 1}, {86, 1},
		{84, 1}, {81, 1}, {79, 1}, {84, 1},
		{84, 1}, {79, 1}, {76, 1}, {79, 1},
		{76, 2}, {74, 1}, {76, 1},
		{79, 1}, {83, 1}, {79, 1}, {76, 1},
		{74, 2}, {0, 2},
		// C section arp - Em
		{76, 1}, {79, 1}, {83, 1}, {79, 1},
		{76, 1}, {74, 1}, {76, 1}, {79, 1},
		{76, 1}, {79, 1}, {81, 1}, {79, 1},
		{76, 1}, {79, 1}, {83, 1}, {86, 1},
		{83, 1}, {79, 1}, {76, 1}, {79, 1},
		{81, 2}, {79, 1}, {76, 1},
		{74, 1}, {76, 1}, {74, 1}, {71, 1},
		{67, 2}, {0, 2},
	}
	arpPCM := generateTrack(arp, squareWave, 130, 0.12)

	// Groovy drum pattern
	drums := generateDrumPattern("K.hxK.hSK.hxK.hSKxhxKshSKxhxKshSK.hxKxhSK.hxKshSKshxKxhSK.hxKshS", 130, 0.25)

	bgmPCM := mixBuffers(melodyPCM, bassPCM)
	bgmPCM = mixBuffers(bgmPCM, arpPCM)
	if len(drums) <= len(bgmPCM) {
		bgmPCM = mixBuffers(bgmPCM, drums)
	}
	p.bgm = loopPlayer(bgmPCM, 0.18)

	// --- SFX: Boing (short rising ping on bumper hit) ---
	boingBuf := pcmStereo(0.08)
	for i := 0; i < len(boingBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(len(boingBuf)/4)
		freq := 600.0 + 1200.0*tFrac // fast rising pitch
		env := (1.0 - tFrac)
		val := sineWave(t, freq)*0.6*env + triangleWave(t, freq*2)*0.2*env
		writeSample(boingBuf, i*4, val)
	}
	p.sfxBoing = boingBuf

	// --- SFX: Flip (quick mechanical click) ---
	flipBuf := pcmStereo(0.04)
	for i := 0; i < len(flipBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(len(flipBuf)/4)
		val := (noise()*0.3 + squareWave(t, 300)*0.5) * math.Exp(-t*60) * (1.0 - tFrac)
		writeSample(flipBuf, i*4, val)
	}
	p.sfxFlip = flipBuf

	// --- SFX: Launch (rising tone whoosh) ---
	launchBuf := pcmStereo(0.2)
	for i := 0; i < len(launchBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(len(launchBuf)/4)
		freq := 200.0 + 800.0*tFrac*tFrac // accelerating rise
		env := envelope(tFrac, 0.05, 0.5, 0.45)
		val := (sawWave(t, freq)*0.4 + sineWave(t, freq*0.5)*0.3) * env * 0.6
		writeSample(launchBuf, i*4, val)
	}
	p.sfxLaunch = launchBuf

	p.bgm.Play()
}

func (p *Pinball) WantsBack() bool            { return p.goBack }
func (p *Pinball) Layout(w, h int) (int, int) { return pbW, pbH }

func (p *Pinball) Update() error {
	p.pbInitAudio()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if p.bgm != nil {
			p.bgm.Pause()
		}
		p.goBack = true
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		bgm := p.bgm
		sfxBoing := p.sfxBoing
		sfxFlip := p.sfxFlip
		sfxLaunch := p.sfxLaunch
		p.pbInit()
		p.bgm = bgm
		p.sfxBoing = sfxBoing
		p.sfxFlip = sfxFlip
		p.sfxLaunch = sfxLaunch
		p.audioInit = true
		return nil
	}
	if p.msgT > 0 {
		p.msgT--
	}

	p.ticks++

	// Update ripples
	aliveRipples := p.ripples[:0]
	for i := range p.ripples {
		p.ripples[i].life--
		p.ripples[i].radius += 2.5
		if p.ripples[i].life > 0 {
			aliveRipples = append(aliveRipples, p.ripples[i])
		}
	}
	p.ripples = aliveRipples

	// Store previous flipper angles for motion blur
	p.lfPrevAng = p.lf.ang
	p.rfPrevAng = p.rf.ang

	switch p.state {
	case pbStLaunch:
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			p.pow++
			if p.pow > 80 {
				p.pow = 80
			}
		} else if p.pow > 0 {
			t := float64(p.pow) / 80.0
			spd := 8.0 + t*6.0
			p.vel = pbV2{(t - 0.5) * 3, -spd}
			p.state = pbStPlay
			p.pow = 0
			if p.sfxLaunch != nil {
				playSFX(p.sfxLaunch, 0.40)
			}
		}
	case pbStPlay:
		p.pbStep()
	case pbStLost:
		if p.msgT <= 0 {
			p.pbResetBall()
		}
	}
	return nil
}

func (p *Pinball) pbStep() {
	lAct := ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyZ)
	rAct := ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyM)

	// Flipper SFX on activation edge
	if lAct && !p.lfWasUp && p.sfxFlip != nil {
		playSFX(p.sfxFlip, 0.40)
	}
	if rAct && !p.rfWasUp && p.sfxFlip != nil {
		playSFX(p.sfxFlip, 0.40)
	}
	p.lfWasUp = lAct
	p.rfWasUp = rAct

	pbAnim(&p.lf, lAct)
	pbAnim(&p.rf, rAct)

	dt := 1.0 / float64(pbSub)
	for i := 0; i < pbSub; i++ {
		p.vel.y += pbGrav * dt
		if p.vel.x > pbMaxSpd {
			p.vel.x = pbMaxSpd
		}
		if p.vel.x < -pbMaxSpd {
			p.vel.x = -pbMaxSpd
		}
		if p.vel.y > pbMaxSpd {
			p.vel.y = pbMaxSpd
		}
		if p.vel.y < -pbMaxSpd {
			p.vel.y = -pbMaxSpd
		}
		p.ball.x += p.vel.x * dt
		p.ball.y += p.vel.y * dt

		// Hard boundary walls
		lw := 10.0 + pbBallR
		rw := float64(pbW) - 10.0 - pbBallR
		tw := 10.0 + pbBallR
		if p.ball.x < lw {
			p.ball.x = lw
			p.vel.x = math.Abs(p.vel.x) * pbDamp
		}
		if p.ball.x > rw {
			p.ball.x = rw
			p.vel.x = -math.Abs(p.vel.x) * pbDamp
		}
		if p.ball.y < tw {
			p.ball.y = tw
			p.vel.y = math.Abs(p.vel.y) * pbDamp
		}

		// Wall segments
		for _, w := range p.walls {
			p.pbSegHit(w)
		}

		// Bumpers
		for j := range p.bumps {
			p.pbBumpHit(&p.bumps[j])
		}

		// Flippers
		p.pbFlipHit(&p.lf, lAct)
		p.pbFlipHit(&p.rf, rAct)
	}

	// Record ball trail position
	if p.ticks%2 == 0 {
		p.trail[p.trailIdx] = pbTrailPos{p.ball.x, p.ball.y}
		p.trailIdx = (p.trailIdx + 1) % len(p.trail)
	}

	for j := range p.bumps {
		if p.bumps[j].flash > 0 {
			p.bumps[j].flash--
		}
	}

	// Drain
	if p.ball.y > float64(pbH)+pbBallR {
		p.lives--
		if p.lives <= 0 {
			p.state = pbStOver
			p.msg = "GAME OVER"
			p.msgT = 999999
		} else {
			p.state = pbStLost
			p.msg = fmt.Sprintf("Ball lost! %d left", p.lives)
			p.msgT = 60
		}
	}
}

func pbAnim(f *pbFlip, act bool) {
	if act {
		f.ang -= pbFSpd
		if f.ang < pbFUp {
			f.ang = pbFUp
		}
	} else {
		f.ang += pbFSpd * 0.5
		if f.ang > pbFRest {
			f.ang = pbFRest
		}
	}
}

func (p *Pinball) pbSegHit(w pbSeg) {
	dx, dy := w.b.x-w.a.x, w.b.y-w.a.y
	lenSq := dx*dx + dy*dy
	if lenSq < 0.01 {
		return
	}
	t := ((p.ball.x-w.a.x)*dx + (p.ball.y-w.a.y)*dy) / lenSq
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cx, cy := w.a.x+t*dx, w.a.y+t*dy
	nx, ny := p.ball.x-cx, p.ball.y-cy
	dist := math.Sqrt(nx*nx + ny*ny)
	minD := pbBallR + w.t*0.5
	if dist < minD && dist > 0.01 {
		nx /= dist
		ny /= dist
		p.ball.x = cx + nx*minD
		p.ball.y = cy + ny*minD
		dot := p.vel.x*nx + p.vel.y*ny
		if dot < 0 {
			p.vel.x -= 2 * dot * nx
			p.vel.y -= 2 * dot * ny
			p.vel.x *= 0.85
			p.vel.y *= 0.85
		}
	}
}

func (p *Pinball) pbBumpHit(b *pbBump) {
	dx := p.ball.x - b.p.x
	dy := p.ball.y - b.p.y
	dist := math.Sqrt(dx*dx + dy*dy)
	minD := pbBallR + b.r
	if dist < minD && dist > 0.01 {
		nx, ny := dx/dist, dy/dist
		p.ball.x = b.p.x + nx*minD
		p.ball.y = b.p.y + ny*minD
		dot := p.vel.x*nx + p.vel.y*ny
		p.vel.x -= 2 * dot * nx
		p.vel.y -= 2 * dot * ny
		sp := math.Sqrt(p.vel.x*p.vel.x + p.vel.y*p.vel.y)
		if sp < pbBumpKick {
			sc := pbBumpKick / (sp + 0.001)
			p.vel.x *= sc
			p.vel.y *= sc
		}
		p.score += b.pts
		b.flash = 10
		// Spawn ripple effect
		p.ripples = append(p.ripples, pbRipple{
			x: b.p.x, y: b.p.y,
			radius: b.r + 3,
			life: 15,
		})
		if p.sfxBoing != nil {
			playSFX(p.sfxBoing, 0.40)
		}
	}
}

func (p *Pinball) pbFlipHit(f *pbFlip, act bool) {
	pv, tp := f.piv, f.tip()
	dx, dy := tp.x-pv.x, tp.y-pv.y
	lenSq := dx*dx + dy*dy
	if lenSq < 0.01 {
		return
	}
	t := ((p.ball.x-pv.x)*dx + (p.ball.y-pv.y)*dy) / lenSq
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cx, cy := pv.x+t*dx, pv.y+t*dy
	nx, ny := p.ball.x-cx, p.ball.y-cy
	dist := math.Sqrt(nx*nx + ny*ny)
	minD := pbBallR + pbFThick
	if dist < minD && dist > 0.01 {
		nx /= dist
		ny /= dist
		p.ball.x = cx + nx*minD
		p.ball.y = cy + ny*minD

		// reflect
		dot := p.vel.x*nx + p.vel.y*ny
		if dot < 0 {
			p.vel.x -= 2 * dot * nx
			p.vel.y -= 2 * dot * ny
		}

		// active flipper kick
		if act && f.ang > pbFUp+0.1 {
			kick := 8.0 + t*4.0
			p.vel.y = -kick
			p.vel.x += kick * 0.25 * f.dir
		} else {
			p.vel.x *= 0.65
			p.vel.y *= 0.65
		}
	}
}

// --- Draw ---

func (p *Pinball) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x0a, 0x12, 0x24, 0xff})
	p.dSparkles(screen)
	p.dTable(screen)
	p.dBumps(screen)
	p.dRipples(screen)
	p.dFlips(screen)
	p.dBallTrail(screen)
	p.dBall(screen)
	p.dHUD(screen)
}

func (p *Pinball) dSparkles(screen *ebiten.Image) {
	t := float64(p.ticks)
	for _, s := range p.sparkles {
		bright := math.Sin(t*0.04+s.phase)*0.5 + 0.5
		alpha := uint8(bright * 80)
		sz := float32(0.8 + bright*1.2)
		vector.FillRect(screen, float32(s.x)-sz/2, float32(s.y)-sz/2, sz, sz,
			color.RGBA{0x80, 0x90, 0xc0, alpha}, false)
		// Cross sparkle for brighter ones
		if bright > 0.7 {
			vector.FillRect(screen, float32(s.x)-sz, float32(s.y)-0.3, sz*2, 0.6,
				color.RGBA{0x80, 0x90, 0xc0, alpha / 2}, false)
			vector.FillRect(screen, float32(s.x)-0.3, float32(s.y)-sz, 0.6, sz*2,
				color.RGBA{0x80, 0x90, 0xc0, alpha / 2}, false)
		}
	}
}

func (p *Pinball) dRipples(screen *ebiten.Image) {
	for _, r := range p.ripples {
		progress := 1.0 - float64(r.life)/15.0
		alpha := uint8((1.0 - progress) * 150)
		vector.StrokeCircle(screen, float32(r.x), float32(r.y), float32(r.radius), 1.5,
			color.RGBA{0xff, 0xd0, 0x30, alpha}, false)
		// Second inner ripple
		if r.radius > 5 {
			vector.StrokeCircle(screen, float32(r.x), float32(r.y), float32(r.radius)*0.6, 1,
				color.RGBA{0xff, 0xff, 0x50, alpha / 2}, false)
		}
	}
}

func (p *Pinball) dBallTrail(screen *ebiten.Image) {
	if p.state == pbStOver {
		return
	}
	// Draw last N positions as progressively smaller/fainter circles
	for i := 0; i < len(p.trail); i++ {
		idx := (p.trailIdx - 1 - i + len(p.trail)) % len(p.trail)
		pos := p.trail[idx]
		if pos.x == 0 && pos.y == 0 {
			continue
		}
		age := float64(i) / float64(len(p.trail))
		alpha := uint8((1.0 - age) * 80)
		radius := float32(pbBallR * (1.0 - age*0.7))
		vector.FillCircle(screen, float32(pos.x), float32(pos.y), radius,
			color.RGBA{0xc0, 0xc0, 0xd0, alpha}, true)
	}
}

func (p *Pinball) dTable(screen *ebiten.Image) {
	wc := color.RGBA{0x45, 0x48, 0x60, 0xff}
	vector.FillRect(screen, 0, 0, 10, pbH, wc, false)           // left
	vector.FillRect(screen, float32(pbW)-10, 0, 10, pbH, wc, false) // right
	vector.FillRect(screen, 0, 0, pbW, 10, wc, false)            // top

	// Playfield border
	vector.StrokeRect(screen, 11, 11, float32(pbW)-22, float32(pbH)-11, 1, color.RGBA{0x50, 0x30, 0x80, 0xff}, false)

	// Wall segments
	for _, w := range p.walls {
		vector.StrokeLine(screen, float32(w.a.x), float32(w.a.y), float32(w.b.x), float32(w.b.y), float32(w.t), color.RGBA{0x60, 0x60, 0x90, 0xff}, true)
	}

	// Drain zone hint
	lt := p.lf.tip()
	rt := p.rf.tip()
	vector.FillRect(screen, float32(lt.x), float32(pbFY+12), float32(rt.x-lt.x), 6, color.RGBA{0x35, 0x10, 0x10, 0xff}, false)

	// Decorative dots
	for row := 0; row < 3; row++ {
		for col := 0; col < 5; col++ {
			vector.FillCircle(screen, float32(40+col*55), float32(150+row*55), 1.5, color.RGBA{0x20, 0x28, 0x40, 0xff}, true)
		}
	}
}

func (p *Pinball) dBumps(screen *ebiten.Image) {
	for _, b := range p.bumps {
		bx, by, br := float32(b.p.x), float32(b.p.y), float32(b.r)
		if b.flash > 0 {
			vector.FillCircle(screen, bx, by, br+3, color.RGBA{0xff, 0xff, 0x50, 0x50}, true)
			vector.FillCircle(screen, bx, by, br, color.RGBA{0xff, 0xd0, 0x30, 0xff}, true)
		} else {
			vector.FillCircle(screen, bx, by, br, color.RGBA{0xd0, 0x30, 0x50, 0xff}, true)
		}
		vector.StrokeCircle(screen, bx, by, br, 2, color.RGBA{0xff, 0x70, 0x80, 0xff}, true)
		vector.StrokeCircle(screen, bx, by, br*0.4, 1, color.RGBA{0xff, 0xb0, 0xb0, 0x60}, true)
		lbl := fmt.Sprintf("%d", b.pts)
		ebitenutil.DebugPrintAt(screen, lbl, int(bx)-len(lbl)*3, int(by)-5)
	}
}

func (p *Pinball) dFlips(screen *ebiten.Image) {
	dFlipWithBlur(screen, &p.lf, p.lfPrevAng)
	dFlipWithBlur(screen, &p.rf, p.rfPrevAng)
}

func dFlipWithBlur(screen *ebiten.Image, f *pbFlip, prevAng float64) {
	pv, tp := f.piv, f.tip()

	// Motion blur: if angle changed significantly, draw ghost at previous position
	angDiff := math.Abs(f.ang - prevAng)
	if angDiff > 0.05 {
		// Draw 2 ghost copies at interpolated positions
		for gi := 1; gi <= 2; gi++ {
			frac := float64(gi) / 3.0
			ghostAng := prevAng + (f.ang-prevAng)*frac
			ghostTipX := f.piv.x + math.Cos(ghostAng)*pbFLen*f.dir
			ghostTipY := f.piv.y + math.Sin(ghostAng)*pbFLen
			alpha := uint8(30 + gi*15)
			ghostClr := color.RGBA{0x20, 0xa0, 0xf0, alpha}
			vector.StrokeLine(screen, float32(pv.x), float32(pv.y),
				float32(ghostTipX), float32(ghostTipY),
				float32(pbFThick*2.5), ghostClr, true)
		}
	}

	fc := color.RGBA{0x20, 0xa0, 0xf0, 0xff}
	vector.StrokeLine(screen, float32(pv.x), float32(pv.y), float32(tp.x), float32(tp.y), float32(pbFThick*2.5), fc, true)
	vector.FillCircle(screen, float32(pv.x), float32(pv.y), float32(pbFThick+2), color.RGBA{0x40, 0xc0, 0xff, 0xff}, true)
	vector.FillCircle(screen, float32(tp.x), float32(tp.y), float32(pbFThick-1), fc, true)
}

func (p *Pinball) dBall(screen *ebiten.Image) {
	if p.state == pbStOver {
		return
	}
	bx, by := float32(p.ball.x), float32(p.ball.y)
	// shadow
	vector.FillCircle(screen, bx+1.5, by+1.5, pbBallR, color.RGBA{0, 0, 0, 0x40}, true)
	// ball
	vector.FillCircle(screen, bx, by, pbBallR, color.RGBA{0xe0, 0xe0, 0xe0, 0xff}, true)
	// sheen
	vector.FillCircle(screen, bx-1.5, by-1.5, pbBallR*0.3, color.White, true)

	// Launch power indicator
	if p.state == pbStLaunch && p.pow > 0 {
		frac := float32(p.pow) / 80.0
		barW := frac * 60
		r := uint8(60 + int(frac*195))
		g := uint8(180 - int(frac*160))
		vector.FillRect(screen, bx-30, by+15, barW, 6, color.RGBA{r, g, 0x20, 0xc0}, false)
		vector.StrokeRect(screen, bx-30, by+15, 60, 6, 1, color.RGBA{0x60, 0x60, 0x80, 0xff}, false)
	}
}

func (p *Pinball) dHUD(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SCORE: %d", p.score), 14, 14)
	for i := 0; i < p.lives; i++ {
		vector.FillCircle(screen, float32(pbW-25-i*15), 19, 4, color.RGBA{0xd0, 0xd0, 0xd0, 0xff}, true)
	}

	switch p.state {
	case pbStLaunch:
		ebitenutil.DebugPrintAt(screen, "Hold SPACE to launch", 75, pbH-15)
	case pbStOver:
		vector.FillRect(screen, 0, 0, pbW, pbH, color.RGBA{0, 0, 0, 0xa0}, false)
		ebitenutil.DebugPrintAt(screen, "GAME OVER", pbW/2-30, pbH/2-25)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d", p.score), pbW/2-30, pbH/2-5)
		ebitenutil.DebugPrintAt(screen, "R=restart  ESC=menu", pbW/2-55, pbH/2+15)
	}

	if p.msgT > 0 && p.state == pbStLost {
		ebitenutil.DebugPrintAt(screen, p.msg, pbW/2-55, pbH/2)
	}

	ebitenutil.DebugPrintAt(screen, "Z/Left  M/Right", 14, pbH-15)
}

func pbClamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
