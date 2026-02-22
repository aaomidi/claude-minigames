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

// Screen dimensions
const (
	tdScreenW = 300
	tdScreenH = 500
)

// Road geometry
const (
	tdLaneCount = 4
	tdRoadLeft  = 30
	tdRoadRight = 270
	tdRoadW     = tdRoadRight - tdRoadLeft
	tdLaneW     = tdRoadW / tdLaneCount
	tdLineW     = 2
	tdLineDash  = 30
	tdLineGap   = 20
	tdLineLen   = tdLineDash + tdLineGap
	tdShoulderW = 4
)

// Player car dimensions
const (
	tdCarW = 30
	tdCarH = 50
)

// Enemy types
const (
	tdEnemyCar     = 0
	tdEnemyTruck   = 1
	tdEnemyBarrier = 2
)

// Game timing
const (
	tdInitialSpeed     = 3.0
	tdMaxSpeed         = 12.0
	tdSpeedIncrement   = 0.003
	tdEnemySpawnMin    = 30
	tdEnemySpawnMax    = 60
	tdBarrierChance    = 15 // percent
	tdTruckChance      = 25 // percent
	tdPlayerSpeed      = 4.0
)

// Game states
const (
	tdPlaying  = 0
	tdGameOver = 1
)

// tdEnemy represents an oncoming vehicle or obstacle.
type tdEnemy struct {
	x, y   float64
	w, h   float64
	lane   int
	kind   int
	colorR uint8
	colorG uint8
	colorB uint8
}

// TrafficDodge is the top-down traffic dodging game.
type TrafficDodge struct {
	goBack bool

	state    int
	score    int
	hiScore  int
	speed    float64
	scrollY  float64 // for road line animation
	playerX  float64
	playerY  float64

	enemies       []tdEnemy
	spawnCooldown int

	ticks int

	// Audio
	bgm       *audio.Player
	sfxCrash  []byte
	audioInit bool
}

func newTrafficDodge() *TrafficDodge {
	g := &TrafficDodge{}
	g.tdReset()
	return g
}

func (g *TrafficDodge) tdReset() {
	g.state = tdPlaying
	g.score = 0
	g.speed = tdInitialSpeed
	g.scrollY = 0
	g.playerX = float64(tdRoadLeft) + float64(tdLaneW)*1.5 - float64(tdCarW)/2
	g.playerY = float64(tdScreenH) - tdCarH - 20
	g.enemies = g.enemies[:0]
	g.spawnCooldown = 60
	g.ticks = 0
}

// ---------------------------------------------------------------------------
// Audio initialization
// ---------------------------------------------------------------------------

func (g *TrafficDodge) tdInitAudio() {
	if g.audioInit {
		return
	}
	g.audioInit = true

	// --- BGM: High-energy driving music, ~160bpm, minor key, chase feel ---
	// Saw wave lead - urgent minor key melody
	lead := [][2]int{
		// A section - urgent Am chase
		{69, 1}, {72, 1}, {74, 1}, {76, 1},
		{77, 2}, {76, 1}, {74, 1},
		{72, 1}, {69, 1}, {67, 1}, {65, 1},
		{64, 2}, {0, 2},
		{69, 1}, {76, 1}, {77, 1}, {76, 1},
		{74, 1}, {72, 1}, {69, 2},
		{67, 1}, {69, 1}, {72, 1}, {74, 1},
		{76, 2}, {77, 1}, {76, 1},
		// B section - Dm shift, faster runs
		{74, 1}, {77, 1}, {74, 1}, {72, 1},
		{69, 1}, {72, 1}, {74, 2},
		{77, 1}, {79, 1}, {81, 1}, {79, 1},
		{77, 2}, {74, 2},
		{72, 1}, {74, 1}, {77, 1}, {79, 1},
		{81, 1}, {79, 1}, {77, 2},
		{74, 1}, {72, 1}, {69, 1}, {67, 1},
		{65, 2}, {0, 2},
		// C section - climax, pedal-to-metal
		{69, 1}, {72, 1}, {76, 1}, {79, 1},
		{81, 2}, {79, 1}, {76, 1},
		{77, 1}, {76, 1}, {74, 1}, {72, 1},
		{69, 2}, {0, 1}, {67, 1},
		{69, 1}, {72, 1}, {74, 1}, {76, 1},
		{77, 1}, {79, 1}, {81, 2},
		{79, 1}, {77, 1}, {76, 1}, {74, 1},
		{72, 2}, {69, 2},
		// Return A - tighter
		{69, 1}, {72, 1}, {74, 1}, {76, 1},
		{77, 2}, {76, 1}, {74, 1},
		{72, 1}, {69, 1}, {67, 1}, {65, 1},
		{64, 2}, {69, 2},
	}
	leadPCM := generateTrack(lead, sawWave, 160, 0.25)

	// Square wave bass - pulsing driving bass
	bass := [][2]int{
		// A section
		{45, 1}, {45, 1}, {0, 1}, {45, 1},
		{41, 1}, {41, 1}, {0, 1}, {41, 1},
		{45, 1}, {45, 1}, {0, 1}, {45, 1},
		{40, 1}, {40, 1}, {0, 1}, {40, 1},
		{45, 1}, {0, 1}, {45, 1}, {45, 1},
		{41, 1}, {0, 1}, {41, 1}, {41, 1},
		{43, 1}, {43, 1}, {0, 1}, {43, 1},
		{40, 1}, {0, 1}, {40, 2},
		// B section - Dm bass
		{38, 1}, {38, 1}, {0, 1}, {38, 1},
		{41, 1}, {41, 1}, {0, 1}, {41, 1},
		{38, 1}, {0, 1}, {38, 1}, {41, 1},
		{43, 1}, {43, 1}, {0, 1}, {43, 1},
		{38, 1}, {38, 1}, {0, 1}, {41, 1},
		{43, 1}, {0, 1}, {43, 1}, {45, 1},
		{41, 1}, {41, 1}, {0, 1}, {38, 1},
		{40, 1}, {0, 1}, {40, 2},
		// C section - higher energy
		{45, 1}, {45, 1}, {0, 1}, {48, 1},
		{45, 1}, {0, 1}, {43, 1}, {41, 1},
		{45, 1}, {45, 1}, {0, 1}, {45, 1},
		{43, 1}, {0, 1}, {41, 1}, {40, 1},
		{45, 1}, {0, 1}, {45, 1}, {48, 1},
		{45, 1}, {43, 1}, {0, 1}, {45, 1},
		{41, 1}, {0, 1}, {40, 1}, {38, 1},
		{40, 1}, {0, 1}, {45, 2},
		// Return
		{45, 1}, {45, 1}, {0, 1}, {45, 1},
		{41, 1}, {41, 1}, {0, 1}, {41, 1},
		{45, 1}, {45, 1}, {0, 1}, {45, 1},
		{40, 1}, {0, 1}, {45, 2},
	}
	bassPCM := generateTrack(bass, squareWave, 160, 0.22)

	// Fast driving drum pattern - 16ths
	drums := generateDrumPattern("K.hxK.hxK.hxKxhxKshxKshxKshxKxhSK.hxKshxK.hxKxhSKxhxKshSKshxKshS", 160, 0.28)

	bgmPCM := mixBuffers(leadPCM, bassPCM)
	if len(drums) <= len(bgmPCM) {
		bgmPCM = mixBuffers(bgmPCM, drums)
	}
	g.bgm = loopPlayer(bgmPCM, 0.18)

	// --- SFX: Crash (big noise burst + low boom) ---
	crashBuf := pcmStereo(0.45)
	for i := 0; i < len(crashBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(len(crashBuf)/4)
		// Noise burst
		n := noise() * math.Exp(-t*8) * 0.8
		// Low boom
		boom := sineWave(t, 50.0*math.Exp(-t*5)) * math.Exp(-t*4) * 0.7
		// Mid crunch
		crunch := squareWave(t, 120.0*math.Exp(-t*10)) * math.Exp(-t*10) * 0.3
		val := (n + boom + crunch) * (1.0 - tFrac*0.3)
		writeSample(crashBuf, i*4, val)
	}
	g.sfxCrash = crashBuf

	g.bgm.Play()
}

func (g *TrafficDodge) WantsBack() bool { return g.goBack }

func (g *TrafficDodge) Layout(w, h int) (int, int) {
	return tdScreenW, tdScreenH
}

// Update handles input and game logic each tick.
func (g *TrafficDodge) Update() error {
	g.tdInitAudio()

	// ESC returns to menu
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.bgm != nil {
			g.bgm.Pause()
		}
		g.goBack = true
		return nil
	}

	// R restarts
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		bgm := g.bgm
		sfxCrash := g.sfxCrash
		hiScore := g.hiScore
		g.tdReset()
		g.bgm = bgm
		g.sfxCrash = sfxCrash
		g.audioInit = true
		g.hiScore = hiScore
		return nil
	}

	if g.state == tdGameOver {
		return nil
	}

	g.ticks++

	// --- Player movement ---
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.playerX -= tdPlayerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.playerX += tdPlayerSpeed
	}

	// Clamp player to road
	if g.playerX < float64(tdRoadLeft)+4 {
		g.playerX = float64(tdRoadLeft) + 4
	}
	if g.playerX+float64(tdCarW) > float64(tdRoadRight)-4 {
		g.playerX = float64(tdRoadRight) - 4 - float64(tdCarW)
	}

	// --- Scroll road lines ---
	g.scrollY += g.speed
	if g.scrollY >= float64(tdLineLen) {
		g.scrollY -= float64(tdLineLen)
	}

	// --- Speed increases over time ---
	if g.speed < tdMaxSpeed {
		g.speed += tdSpeedIncrement
	}

	// --- Score ---
	g.score += int(g.speed)

	// --- Spawn enemies ---
	g.spawnCooldown--
	if g.spawnCooldown <= 0 {
		g.tdSpawnEnemy()
		minC := tdEnemySpawnMin - g.ticks/200
		if minC < 10 {
			minC = 10
		}
		maxC := tdEnemySpawnMax - g.ticks/150
		if maxC < minC+5 {
			maxC = minC + 5
		}
		g.spawnCooldown = minC + rand.Intn(maxC-minC)
	}

	// --- Move enemies ---
	alive := g.enemies[:0]
	for i := range g.enemies {
		e := &g.enemies[i]
		e.y += g.speed + 1
		if e.y < float64(tdScreenH)+100 {
			alive = append(alive, *e)
		}
	}
	g.enemies = alive

	// --- Collision detection ---
	px := g.playerX
	py := g.playerY
	pw := float64(tdCarW)
	ph := float64(tdCarH)
	for _, e := range g.enemies {
		if tdRectsOverlap(px, py, pw, ph, e.x, e.y, e.w, e.h) {
			g.state = tdGameOver
			if g.score > g.hiScore {
				g.hiScore = g.score
			}
			if g.sfxCrash != nil {
				playSFX(g.sfxCrash, 0.50)
			}
			break
		}
	}

	return nil
}

// tdSpawnEnemy creates a new enemy in a random lane.
func (g *TrafficDodge) tdSpawnEnemy() {
	lane := rand.Intn(tdLaneCount)
	laneCenter := float64(tdRoadLeft) + float64(lane)*float64(tdLaneW) + float64(tdLaneW)/2

	roll := rand.Intn(100)
	var e tdEnemy
	e.lane = lane

	if roll < tdBarrierChance {
		// Barrier / construction cone
		e.kind = tdEnemyBarrier
		e.w = float64(tdLaneW) - 8
		e.h = 16
		e.colorR = 255
		e.colorG = 165
		e.colorB = 0
	} else if roll < tdBarrierChance+tdTruckChance {
		// Truck
		e.kind = tdEnemyTruck
		e.w = 34
		e.h = 70
		e.colorR = tdRandColor()
		e.colorG = tdRandColor()
		e.colorB = tdRandColor()
	} else {
		// Car
		e.kind = tdEnemyCar
		e.w = 28
		e.h = 48
		e.colorR = tdRandColor()
		e.colorG = tdRandColor()
		e.colorB = tdRandColor()
	}

	e.x = laneCenter - e.w/2
	e.y = -e.h - float64(rand.Intn(40))

	// Avoid spawning directly on top of another enemy in the same lane
	for _, ex := range g.enemies {
		if ex.lane == lane && ex.y < 80 {
			return // skip this spawn
		}
	}

	g.enemies = append(g.enemies, e)
}

// Draw renders the game.
func (g *TrafficDodge) Draw(screen *ebiten.Image) {
	// Background (grass)
	screen.Fill(color.RGBA{0x3a, 0x6e, 0x2a, 0xff})

	// Road surface
	vector.FillRect(screen, float32(tdRoadLeft), 0, float32(tdRoadW), float32(tdScreenH),
		color.RGBA{0x55, 0x55, 0x55, 0xff}, false)

	// Shoulder lines (solid yellow)
	vector.FillRect(screen, float32(tdRoadLeft), 0, tdShoulderW, float32(tdScreenH),
		color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
	vector.FillRect(screen, float32(tdRoadRight)-tdShoulderW, 0, tdShoulderW, float32(tdScreenH),
		color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)

	// Lane divider lines (dashed white)
	for lane := 1; lane < tdLaneCount; lane++ {
		lx := float32(tdRoadLeft) + float32(lane)*float32(tdLaneW) - tdLineW/2
		g.tdDrawDashedLine(screen, lx, g.scrollY)
	}

	// Draw enemies
	for _, e := range g.enemies {
		g.tdDrawEnemy(screen, &e)
	}

	// Draw player car
	g.tdDrawPlayerCar(screen)

	// HUD
	scoreStr := fmt.Sprintf("Score: %d", g.score/100)
	ebitenutil.DebugPrintAt(screen, scoreStr, 5, 5)
	speedStr := fmt.Sprintf("Speed: %.0f", g.speed)
	ebitenutil.DebugPrintAt(screen, speedStr, tdScreenW-90, 5)

	if g.state == tdGameOver {
		g.tdDrawGameOver(screen)
	}
}

// tdDrawDashedLine draws a vertical dashed lane marking.
func (g *TrafficDodge) tdDrawDashedLine(screen *ebiten.Image, x float32, scroll float64) {
	offset := float32(scroll)
	startY := -float32(tdLineDash) + (offset - float32(int(offset/float32(tdLineLen)))*float32(tdLineLen))

	for y := startY; y < float32(tdScreenH); y += float32(tdLineLen) {
		top := y
		bot := y + float32(tdLineDash)
		if top < 0 {
			top = 0
		}
		if bot > float32(tdScreenH) {
			bot = float32(tdScreenH)
		}
		if bot > top {
			vector.FillRect(screen, x, top, tdLineW, bot-top,
				color.RGBA{0xff, 0xff, 0xff, 0xcc}, false)
		}
	}
}

// tdDrawPlayerCar draws the player's car as a set of rectangles.
func (g *TrafficDodge) tdDrawPlayerCar(screen *ebiten.Image) {
	px := float32(g.playerX)
	py := float32(g.playerY)
	cw := float32(tdCarW)
	ch := float32(tdCarH)

	// Car body (blue)
	vector.FillRect(screen, px, py, cw, ch, color.RGBA{0x22, 0x66, 0xee, 0xff}, false)

	// Windshield (darker blue)
	wsY := py + 6
	wsH := float32(12)
	wsMargin := float32(4)
	vector.FillRect(screen, px+wsMargin, wsY, cw-wsMargin*2, wsH,
		color.RGBA{0x88, 0xcc, 0xff, 0xff}, false)

	// Rear window
	rwY := py + ch - 16
	vector.FillRect(screen, px+wsMargin, rwY, cw-wsMargin*2, float32(10),
		color.RGBA{0x88, 0xcc, 0xff, 0xff}, false)

	// Wheels (dark gray rectangles)
	wheelW := float32(5)
	wheelH := float32(10)
	// Front left
	vector.FillRect(screen, px-2, py+8, wheelW, wheelH, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
	// Front right
	vector.FillRect(screen, px+cw-3, py+8, wheelW, wheelH, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
	// Rear left
	vector.FillRect(screen, px-2, py+ch-18, wheelW, wheelH, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
	// Rear right
	vector.FillRect(screen, px+cw-3, py+ch-18, wheelW, wheelH, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)

	// Headlights (yellow)
	hlSize := float32(4)
	vector.FillRect(screen, px+3, py, hlSize, hlSize, color.RGBA{0xff, 0xff, 0x44, 0xff}, false)
	vector.FillRect(screen, px+cw-3-hlSize, py, hlSize, hlSize, color.RGBA{0xff, 0xff, 0x44, 0xff}, false)

	// Taillights (red)
	vector.FillRect(screen, px+3, py+ch-4, hlSize, hlSize, color.RGBA{0xff, 0x22, 0x22, 0xff}, false)
	vector.FillRect(screen, px+cw-3-hlSize, py+ch-4, hlSize, hlSize, color.RGBA{0xff, 0x22, 0x22, 0xff}, false)
}

// tdDrawEnemy draws an enemy vehicle or barrier.
func (g *TrafficDodge) tdDrawEnemy(screen *ebiten.Image, e *tdEnemy) {
	ex := float32(e.x)
	ey := float32(e.y)
	ew := float32(e.w)
	eh := float32(e.h)

	bodyColor := color.RGBA{e.colorR, e.colorG, e.colorB, 0xff}

	switch e.kind {
	case tdEnemyBarrier:
		// Barrier: orange/white striped block
		vector.FillRect(screen, ex, ey, ew, eh, color.RGBA{0xff, 0xa5, 0x00, 0xff}, false)
		// Diagonal stripe effect: two white stripes
		stripeW := ew / 4
		vector.FillRect(screen, ex+stripeW, ey, stripeW, eh,
			color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
		vector.FillRect(screen, ex+stripeW*3, ey, stripeW, eh,
			color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
		// Black outline
		vector.StrokeRect(screen, ex, ey, ew, eh, 1, color.RGBA{0x00, 0x00, 0x00, 0xff}, false)

	case tdEnemyTruck:
		// Truck body
		vector.FillRect(screen, ex, ey, ew, eh, bodyColor, false)

		// Cab (front section, darker)
		cabH := float32(18)
		cabColor := color.RGBA{
			tdDarken(e.colorR),
			tdDarken(e.colorG),
			tdDarken(e.colorB),
			0xff,
		}
		vector.FillRect(screen, ex, ey+eh-cabH, ew, cabH, cabColor, false)

		// Cab windshield
		margin := float32(5)
		vector.FillRect(screen, ex+margin, ey+eh-cabH+3, ew-margin*2, float32(10),
			color.RGBA{0x88, 0xcc, 0xff, 0xdd}, false)

		// Wheels
		ww := float32(5)
		wh := float32(10)
		vector.FillRect(screen, ex-2, ey+10, ww, wh, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
		vector.FillRect(screen, ex+ew-3, ey+10, ww, wh, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
		vector.FillRect(screen, ex-2, ey+eh-14, ww, wh, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
		vector.FillRect(screen, ex+ew-3, ey+eh-14, ww, wh, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)

		// Taillights at top (they are oncoming, so tail faces us at bottom)
		vector.FillRect(screen, ex+2, ey, 4, 4, color.RGBA{0xff, 0x22, 0x22, 0xff}, false)
		vector.FillRect(screen, ex+ew-6, ey, 4, 4, color.RGBA{0xff, 0x22, 0x22, 0xff}, false)

	default:
		// Car body
		vector.FillRect(screen, ex, ey, ew, eh, bodyColor, false)

		// Windshield (they face us, so windshield at bottom)
		margin := float32(4)
		vector.FillRect(screen, ex+margin, ey+eh-16, ew-margin*2, float32(12),
			color.RGBA{0x88, 0xcc, 0xff, 0xdd}, false)

		// Rear window (top of car since they are oncoming)
		vector.FillRect(screen, ex+margin, ey+5, ew-margin*2, float32(10),
			color.RGBA{0x88, 0xcc, 0xff, 0xdd}, false)

		// Wheels
		ww := float32(5)
		wh := float32(10)
		vector.FillRect(screen, ex-2, ey+8, ww, wh, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
		vector.FillRect(screen, ex+ew-3, ey+8, ww, wh, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
		vector.FillRect(screen, ex-2, ey+eh-18, ww, wh, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)
		vector.FillRect(screen, ex+ew-3, ey+eh-18, ww, wh, color.RGBA{0x22, 0x22, 0x22, 0xff}, false)

		// Headlights (at bottom since they approach)
		vector.FillRect(screen, ex+2, ey+eh-4, 4, 4, color.RGBA{0xff, 0xff, 0x44, 0xff}, false)
		vector.FillRect(screen, ex+ew-6, ey+eh-4, 4, 4, color.RGBA{0xff, 0xff, 0x44, 0xff}, false)

		// Taillights (at top)
		vector.FillRect(screen, ex+2, ey, 4, 4, color.RGBA{0xff, 0x22, 0x22, 0xff}, false)
		vector.FillRect(screen, ex+ew-6, ey, 4, 4, color.RGBA{0xff, 0x22, 0x22, 0xff}, false)
	}
}

// tdDrawGameOver renders the game over overlay.
func (g *TrafficDodge) tdDrawGameOver(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.FillRect(screen, 0, 0, float32(tdScreenW), float32(tdScreenH),
		color.RGBA{0x00, 0x00, 0x00, 0xaa}, false)

	// Crash indicator at player position
	cx := float32(g.playerX) + float32(tdCarW)/2
	cy := float32(g.playerY) + float32(tdCarH)/2
	for r := float32(30); r > 5; r -= 5 {
		alpha := uint8(100 + int(r)*4)
		vector.StrokeCircle(screen, cx, cy, r, 2,
			color.RGBA{0xff, 0x66, 0x00, alpha}, false)
	}

	// Game Over text box
	boxW := float32(220)
	boxH := float32(130)
	boxX := float32(tdScreenW)/2 - boxW/2
	boxY := float32(140)
	vector.FillRect(screen, boxX, boxY, boxW, boxH,
		color.RGBA{0x20, 0x20, 0x30, 0xee}, false)
	vector.StrokeRect(screen, boxX, boxY, boxW, boxH, 2,
		color.RGBA{0xff, 0x44, 0x44, 0xff}, false)

	ebitenutil.DebugPrintAt(screen, "GAME OVER", int(boxX)+75, int(boxY)+12)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d", g.score/100), int(boxX)+75, int(boxY)+36)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Best:  %d", g.hiScore/100), int(boxX)+75, int(boxY)+56)
	ebitenutil.DebugPrintAt(screen, "R - Restart", int(boxX)+72, int(boxY)+84)
	ebitenutil.DebugPrintAt(screen, "ESC - Menu", int(boxX)+75, int(boxY)+104)
}

// --- Helper functions ---

// tdRectsOverlap returns true if two axis-aligned rectangles overlap.
func tdRectsOverlap(ax, ay, aw, ah, bx, by, bw, bh float64) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

// tdRandColor returns a random color component in a vivid range.
func tdRandColor() uint8 {
	return uint8(80 + rand.Intn(176))
}

// tdDarken returns a darkened version of a color component.
func tdDarken(c uint8) uint8 {
	d := int(c) - 50
	if d < 0 {
		d = 0
	}
	return uint8(d)
}
