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

// ---------------------------------------------------------------------------
// Idle Clicker Audio
// ---------------------------------------------------------------------------

var (
	icBGMPlayer    *audio.Player
	icSFXClick     []byte
	icSFXPurchase  []byte
	icSFXPrestige  []byte
	icAudioReady   bool
)

func icInitAudio() {
	if icAudioReady {
		return
	}
	icAudioReady = true

	// --- BGM: Chill lo-fi chiptune (~90 BPM) ---
	// Dreamy major key melody in G major, relaxing sine wave
	melodyNotes := [][2]int{
		// A section - dreamy G major
		{67, 2}, {71, 2}, {74, 4},
		{72, 2}, {74, 1}, {72, 1}, {71, 4},
		{67, 2}, {69, 2}, {71, 2}, {74, 2},
		{76, 4}, {74, 2}, {0, 2},
		{79, 2}, {78, 2}, {76, 4},
		{74, 2}, {71, 2}, {72, 2}, {74, 2},
		{71, 3}, {69, 1}, {67, 4},
		{0, 2}, {66, 2}, {67, 4},
		// A' - gentle variation
		{71, 2}, {74, 2}, {76, 2}, {78, 2},
		{79, 4}, {76, 2}, {74, 2},
		{72, 2}, {71, 2}, {69, 4},
		{67, 2}, {69, 1}, {71, 1}, {72, 4},
		{74, 2}, {0, 2}, {71, 2}, {69, 2},
		{67, 4}, {0, 2}, {66, 1}, {64, 1},
		{67, 6}, {0, 2},
		{71, 2}, {72, 2}, {74, 2}, {76, 2},
		// B section - Em contemplative bridge
		{76, 4}, {74, 2}, {71, 2},
		{69, 2}, {67, 2}, {64, 4},
		{67, 2}, {69, 1}, {71, 1}, {74, 4},
		{76, 2}, {74, 2}, {71, 2}, {0, 2},
		{64, 2}, {67, 2}, {71, 2}, {74, 2},
		{76, 4}, {78, 2}, {76, 2},
		{74, 2}, {72, 2}, {71, 4},
		{69, 2}, {67, 4}, {0, 2},
		// C section - C major warmth, optimistic
		{72, 2}, {74, 2}, {76, 4},
		{79, 2}, {76, 2}, {74, 2}, {72, 2},
		{74, 4}, {76, 2}, {74, 2},
		{72, 2}, {71, 2}, {69, 4},
		{67, 2}, {71, 2}, {74, 2}, {79, 2},
		{78, 4}, {76, 2}, {0, 2},
		{74, 2}, {71, 2}, {67, 4},
		{67, 4}, {0, 4},
	}
	melodyPCM := generateTrack(melodyNotes, sineWave, 90, 0.22)

	// Gentle triangle arpeggio accompaniment
	arpNotes := [][2]int{
		// A section arps
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{57, 1}, {60, 1}, {64, 1}, {69, 1}, {64, 1}, {60, 1}, {57, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {66, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{52, 1}, {55, 1}, {59, 1}, {64, 1}, {59, 1}, {55, 1}, {52, 1}, {0, 1},
		{53, 1}, {57, 1}, {60, 1}, {65, 1}, {60, 1}, {57, 1}, {53, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{50, 1}, {55, 1}, {59, 1}, {62, 1}, {59, 1}, {55, 1}, {50, 1}, {0, 1},
		{52, 1}, {57, 1}, {60, 1}, {64, 1}, {60, 1}, {57, 1}, {52, 1}, {0, 1},
		// A' arps
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{57, 1}, {60, 1}, {64, 1}, {69, 1}, {64, 1}, {60, 1}, {57, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {66, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{52, 1}, {55, 1}, {59, 1}, {64, 1}, {59, 1}, {55, 1}, {52, 1}, {0, 1},
		{50, 1}, {55, 1}, {59, 1}, {62, 1}, {59, 1}, {55, 1}, {50, 1}, {0, 1},
		{48, 1}, {52, 1}, {55, 1}, {60, 1}, {55, 1}, {52, 1}, {48, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		// B section arps - Em
		{52, 1}, {55, 1}, {59, 1}, {64, 1}, {59, 1}, {55, 1}, {52, 1}, {0, 1},
		{52, 1}, {55, 1}, {59, 1}, {64, 1}, {59, 1}, {55, 1}, {52, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{53, 1}, {57, 1}, {60, 1}, {65, 1}, {60, 1}, {57, 1}, {53, 1}, {0, 1},
		{50, 1}, {55, 1}, {59, 1}, {62, 1}, {59, 1}, {55, 1}, {50, 1}, {0, 1},
		{53, 1}, {57, 1}, {60, 1}, {65, 1}, {60, 1}, {57, 1}, {53, 1}, {0, 1},
		{52, 1}, {55, 1}, {59, 1}, {64, 1}, {59, 1}, {55, 1}, {52, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		// C section arps - C major
		{48, 1}, {52, 1}, {55, 1}, {60, 1}, {55, 1}, {52, 1}, {48, 1}, {0, 1},
		{48, 1}, {52, 1}, {55, 1}, {60, 1}, {55, 1}, {52, 1}, {48, 1}, {0, 1},
		{50, 1}, {55, 1}, {57, 1}, {62, 1}, {57, 1}, {55, 1}, {50, 1}, {0, 1},
		{48, 1}, {52, 1}, {55, 1}, {60, 1}, {55, 1}, {52, 1}, {48, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{53, 1}, {57, 1}, {60, 1}, {65, 1}, {60, 1}, {57, 1}, {53, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
		{55, 1}, {59, 1}, {62, 1}, {67, 1}, {62, 1}, {59, 1}, {55, 1}, {0, 1},
	}
	arpPCM := generateTrack(arpNotes, triangleWave, 90, 0.10)

	// Light drums
	drumPattern := "K...h...K.h.h...K...h.h.K.h.h.h.K...h...K.h.h.h.K...h.s.K.h.h..."
	drumPCM := generateDrumPattern(drumPattern, 90, 0.10)

	bgmMix := mixBuffers(melodyPCM, arpPCM)
	for len(drumPCM) < len(bgmMix) {
		drumPCM = append(drumPCM, drumPCM...)
	}
	drumPCM = drumPCM[:len(bgmMix)]
	bgmMix = mixBuffers(bgmMix, drumPCM)

	icBGMPlayer = loopPlayer(bgmMix, 0.10)

	// --- SFX: Click (soft tap) ---
	{
		dur := 0.05
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			env := math.Exp(-t * 80)
			val := sineWave(t, 800) * env * 0.3
			val += noise() * env * 0.1
			writeSample(buf, i*4, val)
		}
		icSFXClick = buf
	}

	// --- SFX: Purchase (cash register) ---
	{
		dur := 0.20
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			env := math.Exp(-t * 10)
			val := sineWave(t, 2000) * env * 0.25
			val += sineWave(t, 3000) * env * 0.15
			val += noise() * math.Exp(-t*40) * 0.1
			writeSample(buf, i*4, val)
		}
		icSFXPurchase = buf
	}

	// --- SFX: Prestige (dramatic ascending fanfare) ---
	{
		// G4 B4 D5 G5 B5 — ascending major arpeggio with square wave
		notes := [][2]int{{67, 2}, {71, 2}, {74, 2}, {79, 2}, {83, 4}}
		icSFXPrestige = generateTrack(notes, squareWave, 360, 0.18)
	}
}

const (
	icScreenW = 450
	icScreenH = 500
)

// Generator definitions
type icGeneratorDef struct {
	name     string
	baseCost float64
	baseRate float64 // coins per second
	color    color.RGBA
}

var icGenerators = []icGeneratorDef{
	{"Cursor", 15, 1, color.RGBA{0x6c, 0xb4, 0xee, 0xff}},
	{"Grandma", 100, 5, color.RGBA{0xee, 0x9a, 0x6c, 0xff}},
	{"Farm", 500, 20, color.RGBA{0x6c, 0xee, 0x6c, 0xff}},
	{"Factory", 3000, 100, color.RGBA{0xee, 0xee, 0x6c, 0xff}},
	{"Bank", 15000, 500, color.RGBA{0xc0, 0x6c, 0xee, 0xff}},
	{"Temple", 100000, 2500, color.RGBA{0xee, 0x6c, 0xc0, 0xff}},
	{"Wizard Tower", 1000000, 15000, color.RGBA{0x6c, 0xee, 0xee, 0xff}},
}

// Floating text animation for click feedback
type icFloatText struct {
	x, y  float64
	text  string
	life  int
	color color.RGBA
}

// Pulse ring for click feedback
type icPulseRing struct {
	x, y   float64
	radius float64
	life   int
	maxLife int
}

// IdleClicker is the main game struct
type IdleClicker struct {
	goBack bool

	coins       float64
	totalCoins  float64 // lifetime coins earned (for prestige tracking)
	clickValue  float64
	owned       [7]int // count of each generator owned
	prestigeLvl int
	multiplier  float64

	floats     []icFloatText
	clickPulse float64 // pulse animation for coin counter
	btnPulse   float64 // pulse for the big button

	scrollY    int // scroll offset for generator list
	maxScrollY int

	frameTick int

	// Visual: expanding pulse rings on click
	pulseRings []icPulseRing
}

func newIdleClicker() *IdleClicker {
	g := &IdleClicker{}
	g.init()
	return g
}

func (g *IdleClicker) init() {
	g.coins = 0
	g.totalCoins = 0
	g.clickValue = 1
	g.owned = [7]int{}
	g.prestigeLvl = 0
	g.multiplier = 1.0
	g.floats = nil
	g.clickPulse = 0
	g.btnPulse = 0
	g.scrollY = 0
	g.frameTick = 0

	// Start audio
	icInitAudio()
	if icBGMPlayer != nil && !icBGMPlayer.IsPlaying() {
		icBGMPlayer.Play()
	}
}

func (g *IdleClicker) restart() {
	prestige := g.prestigeLvl
	mult := g.multiplier
	g.init()
	g.prestigeLvl = prestige
	g.multiplier = mult
}

// icGenCost calculates the current cost of a generator given how many are owned
func icGenCost(basePrice float64, owned int) float64 {
	return math.Floor(basePrice * math.Pow(1.15, float64(owned)))
}

// icFormatNum formats a large number with K/M/B/T suffixes
func icFormatNum(n float64) string {
	if n < 0 {
		return "0"
	}
	if n < 1000 {
		return fmt.Sprintf("%.0f", math.Floor(n))
	}
	if n < 1e6 {
		return fmt.Sprintf("%.1fK", n/1e3)
	}
	if n < 1e9 {
		return fmt.Sprintf("%.1fM", n/1e6)
	}
	if n < 1e12 {
		return fmt.Sprintf("%.1fB", n/1e9)
	}
	return fmt.Sprintf("%.1fT", n/1e12)
}

// icProductionRate returns the total coins-per-second from all generators
func (g *IdleClicker) icProductionRate() float64 {
	total := 0.0
	for i, def := range icGenerators {
		total += def.baseRate * float64(g.owned[i])
	}
	return total * g.multiplier
}

func (g *IdleClicker) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if icBGMPlayer != nil && icBGMPlayer.IsPlaying() {
			icBGMPlayer.Pause()
		}
		g.goBack = true
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.restart()
		return nil
	}

	// Cheats: M = 1M coins, B = 1B coins, G = 10x current coins
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.coins += 1_000_000
		g.totalCoins += 1_000_000
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.coins += 1_000_000_000
		g.totalCoins += 1_000_000_000
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		add := g.coins * 9
		g.coins += add
		g.totalCoins += add
	}

	g.frameTick++

	// --- Auto-production (every frame, scaled by 1/60) ---
	rate := g.icProductionRate()
	earned := rate / 60.0
	g.coins += earned
	g.totalCoins += earned

	// --- Pulse decay ---
	if g.clickPulse > 0 {
		g.clickPulse -= 0.03
		if g.clickPulse < 0 {
			g.clickPulse = 0
		}
	}
	if g.btnPulse > 0 {
		g.btnPulse -= 0.04
		if g.btnPulse < 0 {
			g.btnPulse = 0
		}
	}

	// --- Floating text animation ---
	alive := g.floats[:0]
	for i := range g.floats {
		g.floats[i].life--
		g.floats[i].y -= 1.2
		if g.floats[i].life > 0 {
			alive = append(alive, g.floats[i])
		}
	}
	g.floats = alive

	// --- Big button click / Space ---
	bigBtnX, bigBtnY := float64(icScreenW/2-60), 70.0
	bigBtnW, bigBtnH := 120.0, 60.0
	clicked := false

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if float64(mx) >= bigBtnX && float64(mx) <= bigBtnX+bigBtnW &&
			float64(my) >= bigBtnY && float64(my) <= bigBtnY+bigBtnH {
			clicked = true
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		clicked = true
	}

	if clicked {
		gain := g.clickValue * g.multiplier
		g.coins += gain
		g.totalCoins += gain
		g.clickPulse = 1.0
		g.btnPulse = 1.0
		if icSFXClick != nil {
			playSFX(icSFXClick, 0.20)
		}
		// Spawn floating text
		fx := bigBtnX + bigBtnW/2 + float64(rand.Intn(40)-20)
		fy := bigBtnY - 5
		g.floats = append(g.floats, icFloatText{
			x: fx, y: fy,
			text: "+" + icFormatNum(gain),
			life: 50,
			color: color.RGBA{0xff, 0xd7, 0x00, 0xff},
		})
		// Spawn expanding pulse ring
		g.pulseRings = append(g.pulseRings, icPulseRing{
			x: bigBtnX + bigBtnW/2,
			y: bigBtnY + bigBtnH/2,
			radius: 20,
			life: 25,
			maxLife: 25,
		})
	}

	// Update pulse rings
	aliveRings := g.pulseRings[:0]
	for i := range g.pulseRings {
		g.pulseRings[i].life--
		g.pulseRings[i].radius += 3.0
		if g.pulseRings[i].life > 0 {
			aliveRings = append(aliveRings, g.pulseRings[i])
		}
	}
	g.pulseRings = aliveRings

	// --- Scroll for generator list ---
	_, wy := ebiten.Wheel()
	if wy != 0 {
		g.scrollY -= int(wy * 20)
		if g.scrollY < 0 {
			g.scrollY = 0
		}
		if g.scrollY > g.maxScrollY {
			g.scrollY = g.maxScrollY
		}
	}

	// --- Buy generators (click or number key) ---
	mx, my := ebiten.CursorPosition()
	for i := range icGenerators {
		cost := icGenCost(icGenerators[i].baseCost, g.owned[i])

		// Number key purchase (1-7)
		keyBuy := false
		switch i {
		case 0:
			keyBuy = inpututil.IsKeyJustPressed(ebiten.Key1)
		case 1:
			keyBuy = inpututil.IsKeyJustPressed(ebiten.Key2)
		case 2:
			keyBuy = inpututil.IsKeyJustPressed(ebiten.Key3)
		case 3:
			keyBuy = inpututil.IsKeyJustPressed(ebiten.Key4)
		case 4:
			keyBuy = inpututil.IsKeyJustPressed(ebiten.Key5)
		case 5:
			keyBuy = inpututil.IsKeyJustPressed(ebiten.Key6)
		case 6:
			keyBuy = inpututil.IsKeyJustPressed(ebiten.Key7)
		}

		// Click purchase
		clickBuy := false
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			bx := 15
			by := 170 + i*45 - g.scrollY
			bw := icScreenW - 30
			bh := 40
			if mx >= bx && mx < bx+bw && my >= by && my < by+bh && by >= 155 && by+bh <= icScreenH-35 {
				clickBuy = true
			}
		}

		if (keyBuy || clickBuy) && g.coins >= cost {
			g.coins -= cost
			g.owned[i]++
			if icSFXPurchase != nil {
				playSFX(icSFXPurchase, 0.25)
			}
		}
	}

	// --- Prestige (P key) ---
	if inpututil.IsKeyJustPressed(ebiten.KeyP) && g.totalCoins >= 1e6 {
		if icSFXPrestige != nil {
			playSFX(icSFXPrestige, 0.30)
		}
		g.prestigeLvl++
		g.multiplier *= 2.0
		oldPrestige := g.prestigeLvl
		oldMult := g.multiplier
		g.init()
		g.prestigeLvl = oldPrestige
		g.multiplier = oldMult
	}

	// Update max scroll
	totalListH := len(icGenerators)*45 + 10
	visibleH := icScreenH - 170 - 35
	g.maxScrollY = totalListH - visibleH
	if g.maxScrollY < 0 {
		g.maxScrollY = 0
	}
	if g.scrollY > g.maxScrollY {
		g.scrollY = g.maxScrollY
	}

	return nil
}

func (g *IdleClicker) Draw(screen *ebiten.Image) {
	// Background
	screen.Fill(color.RGBA{0x1a, 0x1a, 0x2e, 0xff})

	// --- Coins display at top ---
	pulseScale := 1.0 + g.clickPulse*0.15
	coinStr := icFormatNum(g.coins)
	rateStr := icFormatNum(g.icProductionRate())

	// Coin counter background with pulse
	headerH := float32(55)
	headerColor := color.RGBA{0x16, 0x21, 0x3e, 0xff}
	vector.FillRect(screen, 0, 0, icScreenW, headerH, headerColor, false)

	// Pulsing gold bar under header
	barBright := uint8(0x40 + int(g.clickPulse*0x40))
	vector.FillRect(screen, 0, headerH-3, icScreenW, 3, color.RGBA{0xda, 0xa5, 0x20, barBright}, false)

	// Draw coin amount with scale effect (simulated via text offset)
	coinLabel := fmt.Sprintf("Coins: %s", coinStr)
	cx := icScreenW/2 - len(coinLabel)*3
	cy := 8
	if pulseScale > 1.01 {
		cy -= int((pulseScale - 1.0) * 10)
	}
	// Large-ish text for coins (we use DebugPrint scaled)
	ebitenutil.DebugPrintAt(screen, coinLabel, cx, cy)

	// Per-second rate
	rateLabel := fmt.Sprintf("per second: %s  |  click: %s  |  x%.0f mult", rateStr, icFormatNum(g.clickValue*g.multiplier), g.multiplier)
	ebitenutil.DebugPrintAt(screen, rateLabel, icScreenW/2-len(rateLabel)*3, 28)

	// Prestige info
	if g.prestigeLvl > 0 {
		pStr := fmt.Sprintf("Prestige: %d", g.prestigeLvl)
		ebitenutil.DebugPrintAt(screen, pStr, icScreenW/2-len(pStr)*3, 42)
	} else if g.totalCoins >= 1e6 {
		pStr := "Press P to Prestige! (2x mult, reset progress)"
		ebitenutil.DebugPrintAt(screen, pStr, icScreenW/2-len(pStr)*3, 42)
	}

	// --- Big click button ---
	btnX := float32(icScreenW/2 - 60)
	btnY := float32(70)
	btnW := float32(120)
	btnH := float32(60)

	// Draw expanding pulse rings behind the button
	for _, ring := range g.pulseRings {
		progress := 1.0 - float64(ring.life)/float64(ring.maxLife)
		alpha := uint8((1.0 - progress) * 180)
		vector.StrokeCircle(screen, float32(ring.x), float32(ring.y), float32(ring.radius), 2,
			color.RGBA{0xff, 0xd7, 0x00, alpha}, false)
	}

	// Button shadow
	vector.FillRect(screen, btnX+3, btnY+3, btnW, btnH, color.RGBA{0x00, 0x00, 0x00, 0x80}, false)

	// Button body with pulse
	btnR := uint8(min(255, 0xda+int(g.btnPulse*40)))
	btnGc := uint8(min(255, 0xa5+int(g.btnPulse*40)))
	btnBc := uint8(0x20)
	vector.FillRect(screen, btnX, btnY, btnW, btnH, color.RGBA{btnR, btnGc, btnBc, 0xff}, false)
	vector.StrokeRect(screen, btnX, btnY, btnW, btnH, 2, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)

	// Subtle idle pulse glow around button
	idlePulse := float32(math.Sin(float64(g.frameTick)*0.06) * 0.5 + 0.5)
	glowAlpha := uint8(20 + idlePulse*30)
	vector.StrokeRect(screen, btnX-2, btnY-2, btnW+4, btnH+4, 1, color.RGBA{0xff, 0xd7, 0x00, glowAlpha}, false)

	// Button text
	clickLabel := "CLICK!"
	ebitenutil.DebugPrintAt(screen, clickLabel, int(btnX)+60-len(clickLabel)*3, int(btnY)+14)
	spaceHint := "[Space]"
	ebitenutil.DebugPrintAt(screen, spaceHint, int(btnX)+60-len(spaceHint)*3, int(btnY)+32)

	// --- Floating texts ---
	for _, ft := range g.floats {
		alpha := float64(ft.life) / 50.0
		if alpha > 1 {
			alpha = 1
		}
		c := ft.color
		c.A = uint8(alpha * 255)
		ebitenutil.DebugPrintAt(screen, ft.text, int(ft.x)-len(ft.text)*3, int(ft.y))
	}

	// --- Separator line ---
	vector.FillRect(screen, 10, 148, icScreenW-20, 1, color.RGBA{0x44, 0x44, 0x66, 0xff}, false)

	// --- Generator list header ---
	ebitenutil.DebugPrintAt(screen, "GENERATORS  (scroll or press 1-7 to buy)", 15, 152)

	// --- Generator list (clipped area) ---
	// We draw generators from y=170 to y=icScreenH-35
	listTop := 170
	listBot := icScreenH - 35

	mx, my := ebiten.CursorPosition()

	for i, def := range icGenerators {
		by := listTop + i*45 - g.scrollY
		bx := 15
		bw := icScreenW - 30
		bh := 40

		// Skip if out of visible area
		if by+bh < listTop || by > listBot {
			continue
		}

		cost := icGenCost(def.baseCost, g.owned[i])
		canAfford := g.coins >= cost

		// Background color
		bgColor := color.RGBA{0x20, 0x20, 0x38, 0xff}
		hover := mx >= bx && mx < bx+bw && my >= by && my < by+bh && my >= listTop && my < listBot
		if hover {
			bgColor = color.RGBA{0x30, 0x30, 0x50, 0xff}
		}
		if !canAfford {
			bgColor = color.RGBA{0x18, 0x18, 0x28, 0xff}
		}

		vector.FillRect(screen, float32(bx), float32(by), float32(bw), float32(bh), bgColor, false)

		// Shimmer effect on affordable generators
		if canAfford {
			shimmerPhase := float64(g.frameTick)*0.05 + float64(i)*1.2
			shimmerX := float32(bx) + float32(math.Mod(shimmerPhase*60, float64(bw+40))) - 20
			if shimmerX >= float32(bx) && shimmerX <= float32(bx+bw-20) {
				shimmerAlpha := uint8(30)
				vector.FillRect(screen, shimmerX, float32(by), 20, float32(bh),
					color.RGBA{0xff, 0xff, 0xff, shimmerAlpha}, false)
			}
		}

		// Left color accent bar
		accentColor := def.color
		if !canAfford {
			accentColor = color.RGBA{accentColor.R / 2, accentColor.G / 2, accentColor.B / 2, 0xff}
		}
		vector.FillRect(screen, float32(bx), float32(by), 4, float32(bh), accentColor, false)

		// Animated production indicator dots (small moving dots along the accent bar)
		if g.owned[i] > 0 {
			dotCount := g.owned[i]
			if dotCount > 5 {
				dotCount = 5
			}
			for d := 0; d < dotCount; d++ {
				dotPhase := float64(g.frameTick)*0.04 + float64(d)*0.7
				dotY := float32(by) + float32(math.Mod(dotPhase*float64(bh), float64(bh)))
				vector.FillCircle(screen, float32(bx)+2, dotY, 1.5,
					color.RGBA{def.color.R, def.color.G, def.color.B, 0xaa}, false)
			}
		}

		// Border
		borderColor := def.color
		if !canAfford {
			borderColor = color.RGBA{0x33, 0x33, 0x44, 0xff}
		}
		vector.StrokeRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 1, borderColor, false)

		// Generator number key hint
		keyHint := fmt.Sprintf("[%d]", i+1)
		ebitenutil.DebugPrintAt(screen, keyHint, bx+8, by+4)

		// Generator name and owned count
		nameStr := fmt.Sprintf("%s x%d", def.name, g.owned[i])
		ebitenutil.DebugPrintAt(screen, nameStr, bx+35, by+4)

		// Production rate
		prodStr := fmt.Sprintf("+%s/sec", icFormatNum(def.baseRate*g.multiplier))
		ebitenutil.DebugPrintAt(screen, prodStr, bx+35, by+20)

		// Cost (right side)
		costStr := fmt.Sprintf("Cost: %s", icFormatNum(cost))
		costX := bx + bw - len(costStr)*6 - 10
		ebitenutil.DebugPrintAt(screen, costStr, costX, by+12)

		// Production total for this generator
		if g.owned[i] > 0 {
			totalProd := fmt.Sprintf("(total: %s/s)", icFormatNum(def.baseRate*float64(g.owned[i])*g.multiplier))
			ebitenutil.DebugPrintAt(screen, totalProd, bx+160, by+20)
		}
	}

	// --- Bottom bar ---
	vector.FillRect(screen, 0, float32(icScreenH-30), icScreenW, 30, color.RGBA{0x16, 0x21, 0x3e, 0xff}, false)
	vector.FillRect(screen, 0, float32(icScreenH-30), icScreenW, 1, color.RGBA{0x44, 0x44, 0x66, 0xff}, false)

	bottomStr := "ESC:Back R:Restart M:1M B:1B G:10x"
	if g.totalCoins >= 1e6 {
		bottomStr += "  P: Prestige"
	}
	totalProdStr := fmt.Sprintf("Total: %s/sec", icFormatNum(g.icProductionRate()))
	ebitenutil.DebugPrintAt(screen, bottomStr, 10, icScreenH-22)
	ebitenutil.DebugPrintAt(screen, totalProdStr, icScreenW-len(totalProdStr)*6-10, icScreenH-22)
}

func (g *IdleClicker) Layout(w, h int) (int, int) {
	return icScreenW, icScreenH
}

func (g *IdleClicker) WantsBack() bool {
	return g.goBack
}
