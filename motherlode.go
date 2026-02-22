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
// Constants
// ---------------------------------------------------------------------------

const (
	mlScreenW = 400
	mlScreenH = 500

	mlCellSize = 20
	mlGridCols = mlScreenW / mlCellSize // 20
	mlGridRows = 80                     // total underground rows

	mlSurfaceH   = 80 // pixels of surface area at top
	mlPlayerW    = 16
	mlPlayerH    = 18
	mlDigTicks   = 8  // base ticks to dig dirt
	mlRockDigMul = 3  // rock takes this many times longer

	mlFuelCostMove = 0.15
	mlFuelCostDig  = 0.4
	mlFuelCostFly  = 0.25

	mlStartFuel  = 100.0
	mlStartCargo = 50.0
)

// ---------------------------------------------------------------------------
// Cell types
// ---------------------------------------------------------------------------

const (
	mlCellEmpty   = 0
	mlCellDirt    = 1
	mlCellRock    = 2
	mlCellCoal    = 3
	mlCellCopper  = 4
	mlCellSilver  = 5
	mlCellGold    = 6
	mlCellDiamond = 7
	mlCellRuby    = 8
)

type mlMineral struct {
	name   string
	value  int
	weight int
	clr    color.RGBA
}

var mlMinerals = map[int]mlMineral{
	mlCellCoal:    {"Coal", 5, 2, color.RGBA{0x44, 0x44, 0x44, 0xff}},
	mlCellCopper:  {"Copper", 20, 3, color.RGBA{0xdd, 0x88, 0x33, 0xff}},
	mlCellSilver:  {"Silver", 50, 4, color.RGBA{0xcc, 0xcc, 0xdd, 0xff}},
	mlCellGold:    {"Gold", 100, 5, color.RGBA{0xff, 0xdd, 0x33, 0xff}},
	mlCellDiamond: {"Diamond", 500, 6, color.RGBA{0x66, 0xee, 0xff, 0xff}},
	mlCellRuby:    {"Ruby", 1000, 7, color.RGBA{0xee, 0x22, 0x44, 0xff}},
}

var mlDirtColor = color.RGBA{0x8b, 0x5a, 0x2b, 0xff}
var mlSkyColor = color.RGBA{0x55, 0xbb, 0xff, 0xff}
var mlGrassColor = color.RGBA{0x44, 0x99, 0x33, 0xff}
var mlShopColor = color.RGBA{0xaa, 0x66, 0x33, 0xff}
var mlShopRoofColor = color.RGBA{0xcc, 0x33, 0x33, 0xff}
var mlBgUnderground = color.RGBA{0x3b, 0x25, 0x15, 0xff}
var mlPlayerColor = color.RGBA{0xee, 0xee, 0x33, 0xff}
var mlDrillColor = color.RGBA{0xaa, 0xaa, 0xaa, 0xff}
var mlTrackColor = color.RGBA{0x33, 0x33, 0x33, 0xff}
var mlCabinColor = color.RGBA{0x33, 0x99, 0xdd, 0xff}
var mlExhaustColor = color.RGBA{0x66, 0x66, 0x66, 0x99}

// ---------------------------------------------------------------------------
// Shop upgrades
// ---------------------------------------------------------------------------

const (
	mlUpgFuel  = 0
	mlUpgCargo = 1
	mlUpgDrill = 2
	mlUpgHull  = 3
	mlUpgCount = 4
)

type mlUpgrade struct {
	name     string
	maxLevel int
	costs    []int
	desc     string
}

var mlUpgrades = [mlUpgCount]mlUpgrade{
	{"Fuel Tank", 5, []int{200, 500, 1200, 3000, 8000}, "More max fuel"},
	{"Cargo Hold", 5, []int{150, 400, 1000, 2500, 6000}, "More cargo capacity"},
	{"Drill", 5, []int{300, 700, 1500, 4000, 10000}, "Dig faster"},
	{"Hull", 5, []int{250, 600, 1300, 3500, 9000}, "Dig deeper safely"},
}

// ---------------------------------------------------------------------------
// Game state
// ---------------------------------------------------------------------------

const (
	mlStatePlaying = 0
	mlStateShop    = 1
)

type mlInventoryItem struct {
	cellType int
	count    int
}

// ---------------------------------------------------------------------------
// Audio assets
// ---------------------------------------------------------------------------

var (
	mlBGMPlayer    *audio.Player
	mlSFXDig       []byte
	mlSFXMineral   []byte
	mlSFXSell      []byte
	mlSFXUpgrade   []byte
	mlAudioReady   bool
)

func mlInitAudio() {
	if mlAudioReady {
		return
	}
	mlAudioReady = true

	// --- BGM: Underground mining ambiance (~80 BPM) ---
	// Deep triangle bass line in C minor, slow and cavernous
	bassNotes := [][2]int{
		// A section - cavernous Cm
		{36, 4}, {0, 2}, {39, 2}, {36, 4}, {0, 2}, {43, 2},
		{34, 4}, {0, 2}, {38, 2}, {34, 4}, {0, 2}, {41, 2},
		{31, 4}, {0, 2}, {36, 2}, {38, 3}, {36, 3}, {34, 2},
		{33, 4}, {0, 2}, {36, 2}, {33, 4}, {0, 4},
		{36, 3}, {0, 1}, {39, 2}, {41, 2}, {43, 4}, {0, 2}, {39, 2},
		{34, 4}, {0, 2}, {36, 2}, {31, 4}, {0, 2}, {34, 2},
		{36, 6}, {0, 2}, {33, 6}, {0, 2},
		{31, 4}, {34, 4}, {36, 4}, {0, 4},
		// B section - deeper, Fm feel
		{29, 4}, {0, 2}, {33, 2}, {34, 4}, {0, 2}, {36, 2},
		{29, 4}, {0, 2}, {31, 2}, {33, 4}, {0, 4},
		{34, 3}, {0, 1}, {36, 2}, {38, 2}, {39, 4}, {0, 2}, {36, 2},
		{31, 4}, {0, 2}, {29, 2}, {27, 4}, {0, 4},
		// C section - Eb major brightness
		{39, 4}, {0, 2}, {43, 2}, {39, 4}, {0, 2}, {46, 2},
		{41, 4}, {0, 2}, {43, 2}, {41, 4}, {0, 4},
		{36, 3}, {0, 1}, {39, 2}, {43, 2}, {36, 4}, {0, 2}, {34, 2},
		{31, 4}, {34, 4}, {36, 4}, {0, 4},
	}
	bassPCM := generateTrack(bassNotes, triangleWave, 80, 0.35)

	// High sine pings — mysterious cave echoes in C minor pentatonic
	pingNotes := [][2]int{
		// A section pings
		{0, 8}, {84, 1}, {0, 7},
		{0, 6}, {87, 1}, {0, 5}, {79, 1}, {0, 3},
		{0, 10}, {91, 1}, {0, 5},
		{0, 4}, {82, 1}, {0, 7}, {86, 1}, {0, 3},
		{0, 12}, {84, 1}, {0, 3},
		{0, 6}, {79, 1}, {0, 3}, {82, 1}, {0, 5},
		{0, 8}, {87, 1}, {0, 7},
		{0, 4}, {91, 1}, {0, 3}, {84, 1}, {0, 3}, {79, 1}, {0, 3},
		// B section pings - darker, Fm colors
		{0, 6}, {80, 1}, {0, 5}, {84, 1}, {0, 3},
		{0, 10}, {87, 1}, {0, 5},
		{0, 4}, {77, 1}, {0, 7}, {80, 1}, {0, 3},
		{0, 8}, {84, 1}, {0, 3}, {77, 1}, {0, 3},
		// C section pings - brighter Eb
		{0, 6}, {87, 1}, {0, 5}, {91, 1}, {0, 3},
		{0, 8}, {84, 1}, {0, 7},
		{0, 4}, {79, 1}, {0, 7}, {84, 1}, {0, 3},
		{0, 6}, {91, 1}, {0, 3}, {87, 1}, {0, 3}, {84, 1}, {0, 3},
	}
	pingPCM := generateTrack(pingNotes, sineWave, 80, 0.12)

	// Subtle drum rumble
	drumPattern := "K...........h...K.......h...K...K.......h.......h...K.........h...K...h.......K.s.K...........h...K.......h.s.K..."
	drumPCM := generateDrumPattern(drumPattern, 80, 0.20)

	// Mix bass and pings, then mix with drums
	bgmMix := mixBuffers(bassPCM, pingPCM)
	// Extend drum pattern to match bgm length by repeating
	for len(drumPCM) < len(bgmMix) {
		drumPCM = append(drumPCM, drumPCM...)
	}
	drumPCM = drumPCM[:len(bgmMix)]
	bgmMix = mixBuffers(bgmMix, drumPCM)

	mlBGMPlayer = loopPlayer(bgmMix, 0.12)

	// --- SFX: Dig (short crunch/noise burst) ---
	{
		dur := 0.08
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := float64(i) / float64(samples)
			env := math.Exp(-t * 50) // fast decay
			val := noise() * env * 0.5
			// add a low thump
			val += sineWave(t, 120*math.Exp(-t*30)) * env * 0.3
			_ = tFrac
			writeSample(buf, i*4, val)
		}
		mlSFXDig = buf
	}

	// --- SFX: Mineral found (sparkle arpeggio) ---
	{
		// Quick ascending arpeggio: C5 E5 G5 C6
		notes := [][2]int{{72, 1}, {76, 1}, {79, 1}, {84, 2}}
		mlSFXMineral = generateTrack(notes, sineWave, 480, 0.25)
	}

	// --- SFX: Sell (cash register ding) ---
	{
		dur := 0.25
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			env := math.Exp(-t * 8)
			// two metallic tones
			val := sineWave(t, 2200) * env * 0.3
			val += sineWave(t, 3300) * env * 0.2
			val += sineWave(t, 4400) * env * 0.1
			writeSample(buf, i*4, val)
		}
		mlSFXSell = buf
	}

	// --- SFX: Upgrade (ascending level-up tone) ---
	{
		// Ascending: C4 E4 G4 C5 with square wave
		notes := [][2]int{{60, 2}, {64, 2}, {67, 2}, {72, 4}}
		mlSFXUpgrade = generateTrack(notes, squareWave, 300, 0.15)
	}
}

// Dig particle for debris effect
type mlDigParticle struct {
	x, y   float64
	vx, vy float64
	life   int
}

type Motherlode struct {
	goBack bool

	grid [mlGridRows][mlGridCols]int

	// Player position in pixel coords. x is horizontal, y is vertical.
	// y=0 is the surface level (top of underground grid).
	// Negative y means above surface.
	px, py float64

	// Digging state
	digging    bool
	digTimer   int
	digTargetR int
	digTargetC int

	// Camera offset (vertical scroll)
	camY float64

	// Resources
	fuel    float64
	maxFuel float64
	cargo   []mlInventoryItem
	cargoWt int
	maxWt   int
	money   int

	// Upgrade levels (0 = base)
	upgLevels [mlUpgCount]int

	// Drill speed multiplier
	drillMul float64

	// Max safe depth (row) based on hull upgrade
	maxSafeDepth int

	// State
	state    int
	shopSel  int
	msgText  string
	msgTimer int

	// Tick counter for animations
	ticks int

	// Visual: dig debris particles
	digParticles []mlDigParticle
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func newMotherlode() *Motherlode {
	m := &Motherlode{}
	m.mlInit()
	return m
}

func (m *Motherlode) mlInit() {
	m.goBack = false
	m.state = mlStatePlaying
	m.shopSel = 0
	m.money = 0
	m.ticks = 0

	m.upgLevels = [mlUpgCount]int{0, 0, 0, 0}
	m.mlApplyUpgrades()

	// Start on surface
	m.px = float64(mlScreenW/2 - mlPlayerW/2)
	m.py = -float64(mlPlayerH) // above ground
	m.camY = 0
	m.digging = false
	m.cargo = nil
	m.cargoWt = 0

	m.msgText = ""
	m.msgTimer = 0

	m.mlGenerateGrid()

	// Start audio
	mlInitAudio()
	if mlBGMPlayer != nil && !mlBGMPlayer.IsPlaying() {
		mlBGMPlayer.Play()
	}
}

func (m *Motherlode) mlApplyUpgrades() {
	m.maxFuel = mlStartFuel + float64(m.upgLevels[mlUpgFuel])*40
	m.fuel = m.maxFuel
	m.maxWt = int(mlStartCargo) + m.upgLevels[mlUpgCargo]*30
	m.drillMul = 1.0 + float64(m.upgLevels[mlUpgDrill])*0.5
	m.maxSafeDepth = 20 + m.upgLevels[mlUpgHull]*15
}

// ---------------------------------------------------------------------------
// Grid generation
// ---------------------------------------------------------------------------

func (m *Motherlode) mlGenerateGrid() {
	for r := 0; r < mlGridRows; r++ {
		for c := 0; c < mlGridCols; c++ {
			// Deeper = more rock
			rockChance := 0.05 + float64(r)*0.012
			if rockChance > 0.50 {
				rockChance = 0.50
			}

			if rand.Float64() < rockChance {
				m.grid[r][c] = mlCellRock
			} else {
				m.grid[r][c] = mlCellDirt
			}

			// Place minerals based on depth
			mineral := m.mlRollMineral(r)
			if mineral != 0 {
				m.grid[r][c] = mineral
			}
		}
	}
}

func (m *Motherlode) mlRollMineral(row int) int {
	depth := float64(row)

	// Coal: common, starts at row 2
	if depth >= 2 && rand.Float64() < 0.12 {
		return mlCellCoal
	}
	// Copper: moderate, starts at row 5
	if depth >= 5 && rand.Float64() < 0.07 {
		return mlCellCopper
	}
	// Silver: less common, starts at row 12
	if depth >= 12 && rand.Float64() < 0.05 {
		return mlCellSilver
	}
	// Gold: uncommon, starts at row 20
	if depth >= 20 && rand.Float64() < 0.035 {
		return mlCellGold
	}
	// Diamond: rare, starts at row 35
	if depth >= 35 && rand.Float64() < 0.02 {
		return mlCellDiamond
	}
	// Ruby: very rare, starts at row 55
	if depth >= 55 && rand.Float64() < 0.012 {
		return mlCellRuby
	}
	return 0
}

// ---------------------------------------------------------------------------
// SubGame interface
// ---------------------------------------------------------------------------

func (m *Motherlode) WantsBack() bool            { return m.goBack }
func (m *Motherlode) Layout(w, h int) (int, int)  { return mlScreenW, mlScreenH }

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m *Motherlode) Update() error {
	m.ticks++

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if m.state == mlStateShop {
			m.state = mlStatePlaying
			return nil
		}
		if mlBGMPlayer != nil && mlBGMPlayer.IsPlaying() {
			mlBGMPlayer.Pause()
		}
		m.goBack = true
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		m.mlInit()
		return nil
	}

	if m.msgTimer > 0 {
		m.msgTimer--
	}

	// Update dig debris particles
	alive := m.digParticles[:0]
	for i := range m.digParticles {
		p := &m.digParticles[i]
		p.life--
		if p.life <= 0 {
			continue
		}
		p.x += p.vx
		p.y += p.vy
		p.vy += 0.15 // gravity
		alive = append(alive, *p)
	}
	m.digParticles = alive

	if m.state == mlStateShop {
		m.mlUpdateShop()
		return nil
	}

	m.mlUpdatePlaying()
	return nil
}

func (m *Motherlode) mlUpdateShop() {
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		m.state = mlStatePlaying
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		m.shopSel--
		if m.shopSel < 0 {
			m.shopSel = mlUpgCount - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		m.shopSel = (m.shopSel + 1) % mlUpgCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		m.mlBuyUpgrade(m.shopSel)
	}
}

func (m *Motherlode) mlBuyUpgrade(idx int) {
	upg := mlUpgrades[idx]
	lvl := m.upgLevels[idx]
	if lvl >= upg.maxLevel {
		m.mlSetMsg("Already maxed!")
		return
	}
	cost := upg.costs[lvl]
	if m.money < cost {
		m.mlSetMsg("Not enough money!")
		return
	}
	m.money -= cost
	m.upgLevels[idx]++
	m.mlApplyUpgrades()
	m.fuel = m.maxFuel // refuel on upgrade
	if mlSFXUpgrade != nil {
		playSFX(mlSFXUpgrade, 0.30)
	}
	m.mlSetMsg(fmt.Sprintf("Upgraded %s to Lv%d!", upg.name, m.upgLevels[idx]))
}

func (m *Motherlode) mlSetMsg(txt string) {
	m.msgText = txt
	m.msgTimer = 120
}

func (m *Motherlode) mlUpdatePlaying() {
	// Check if player is on the surface
	onSurface := m.py <= -float64(mlPlayerH)+2

	// Open shop
	if onSurface && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		m.state = mlStateShop
		return
	}

	// If currently digging, continue
	if m.digging {
		m.digTimer--
		if m.digTimer <= 0 {
			m.mlFinishDig()
		}
		return
	}

	// Movement
	moveSpeed := 2.0

	if m.fuel <= 0 {
		// Out of fuel: can only sit
		if onSurface {
			m.mlSellCargo()
		}
		return
	}

	// Left/Right movement
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		m.px -= moveSpeed
		if m.px < 0 {
			m.px = 0
		}
		m.fuel -= mlFuelCostMove
		if m.fuel < 0 {
			m.fuel = 0
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		m.px += moveSpeed
		if m.px > float64(mlScreenW-mlPlayerW) {
			m.px = float64(mlScreenW - mlPlayerW)
		}
		m.fuel -= mlFuelCostMove
		if m.fuel < 0 {
			m.fuel = 0
		}
	}

	// Dig down
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		if onSurface {
			// Move from surface into ground
			m.py += moveSpeed
			m.fuel -= mlFuelCostMove
			if m.fuel < 0 {
				m.fuel = 0
			}
		} else {
			// Try to dig the cell below
			m.mlTryDig()
		}
	}

	// Fly up
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		if onSurface {
			// Already on surface, sell
			m.mlSellCargo()
		} else {
			m.mlTryFlyUp()
		}
	}

	// Apply gravity when underground (if not pressing up and cell below is empty)
	if !onSurface && !ebiten.IsKeyPressed(ebiten.KeyArrowUp) && !ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		m.mlApplyGravity()
	}

	// Clamp vertical position
	if m.py < -float64(mlPlayerH) {
		m.py = -float64(mlPlayerH)
	}
	maxPy := float64((mlGridRows - 1) * mlCellSize)
	if m.py > maxPy {
		m.py = maxPy
	}

	// Depth damage check
	playerRow := int(m.py+float64(mlPlayerH)) / mlCellSize
	if playerRow > m.maxSafeDepth {
		// Take hull damage - lose extra fuel
		m.fuel -= 0.3
		if m.fuel < 0 {
			m.fuel = 0
		}
	}

	// Update camera
	m.mlUpdateCamera()
}

func (m *Motherlode) mlApplyGravity() {
	// Check cell directly below player
	bottomY := m.py + float64(mlPlayerH) + 2
	if bottomY < 0 {
		return // above ground
	}
	row := int(bottomY) / mlCellSize
	col := int(m.px+float64(mlPlayerW)/2) / mlCellSize
	if col < 0 {
		col = 0
	}
	if col >= mlGridCols {
		col = mlGridCols - 1
	}

	if row >= 0 && row < mlGridRows {
		if m.grid[row][col] == mlCellEmpty {
			m.py += 1.5 // fall
		}
	}
}

func (m *Motherlode) mlTryDig() {
	// Find the cell below the player center
	centerX := m.px + float64(mlPlayerW)/2
	bottomY := m.py + float64(mlPlayerH)

	col := int(centerX) / mlCellSize
	row := int(bottomY) / mlCellSize

	if col < 0 || col >= mlGridCols || row < 0 || row >= mlGridRows {
		return
	}

	cell := m.grid[row][col]
	if cell == mlCellEmpty {
		// Empty cell below, just move down
		m.py += 2
		m.fuel -= mlFuelCostMove
		if m.fuel < 0 {
			m.fuel = 0
		}
		return
	}

	// Start digging
	baseTicks := mlDigTicks
	if cell == mlCellRock {
		baseTicks *= mlRockDigMul
	}
	// Apply drill upgrade
	ticks := int(float64(baseTicks) / m.drillMul)
	if ticks < 2 {
		ticks = 2
	}

	m.digging = true
	m.digTimer = ticks
	m.digTargetR = row
	m.digTargetC = col
	if mlSFXDig != nil {
		playSFX(mlSFXDig, 0.25)
	}
	m.fuel -= mlFuelCostDig
	if m.fuel < 0 {
		m.fuel = 0
	}
}

func (m *Motherlode) mlFinishDig() {
	m.digging = false
	r := m.digTargetR
	c := m.digTargetC
	if r < 0 || r >= mlGridRows || c < 0 || c >= mlGridCols {
		return
	}

	cell := m.grid[r][c]

	// Check if it is a mineral
	if mineral, ok := mlMinerals[cell]; ok {
		if m.cargoWt+mineral.weight <= m.maxWt {
			m.mlAddCargo(cell)
			if mlSFXMineral != nil {
				playSFX(mlSFXMineral, 0.30)
			}
			m.mlSetMsg(fmt.Sprintf("Found %s! ($%d)", mineral.name, mineral.value))
		} else {
			m.mlSetMsg("Cargo full! Return to surface.")
			// Still dig the cell but don't collect
		}
	}

	// Spawn dig debris particles
	debrisX := float64(c*mlCellSize) + float64(mlCellSize)/2
	debrisY := float64(r*mlCellSize) + float64(mlCellSize)/2
	for i := 0; i < 8; i++ {
		m.digParticles = append(m.digParticles, mlDigParticle{
			x:    debrisX,
			y:    debrisY,
			vx:   (rand.Float64() - 0.5) * 3,
			vy:   (rand.Float64() - 0.5) * 3 - 1,
			life: 15 + rand.Intn(20),
		})
	}

	m.grid[r][c] = mlCellEmpty
	// Move player down into the cleared cell
	m.py = float64(r*mlCellSize) - float64(mlPlayerH) + float64(mlCellSize)/2
}

func (m *Motherlode) mlAddCargo(cellType int) {
	mineral := mlMinerals[cellType]
	for i := range m.cargo {
		if m.cargo[i].cellType == cellType {
			m.cargo[i].count++
			m.cargoWt += mineral.weight
			return
		}
	}
	m.cargo = append(m.cargo, mlInventoryItem{cellType, 1})
	m.cargoWt += mineral.weight
}

func (m *Motherlode) mlTryFlyUp() {
	// Check cell above player center
	centerX := m.px + float64(mlPlayerW)/2
	topY := m.py - 2

	if topY < 0 {
		// Flying to surface
		m.py -= 3
		m.fuel -= mlFuelCostFly
		if m.fuel < 0 {
			m.fuel = 0
		}
		if m.py <= -float64(mlPlayerH) {
			m.py = -float64(mlPlayerH)
			m.mlSellCargo()
		}
		return
	}

	col := int(centerX) / mlCellSize
	row := int(topY) / mlCellSize

	if col < 0 {
		col = 0
	}
	if col >= mlGridCols {
		col = mlGridCols - 1
	}

	if row >= 0 && row < mlGridRows {
		if m.grid[row][col] != mlCellEmpty {
			// Blocked by solid cell
			return
		}
	}

	m.py -= 3
	m.fuel -= mlFuelCostFly
	if m.fuel < 0 {
		m.fuel = 0
	}
}

func (m *Motherlode) mlSellCargo() {
	if len(m.cargo) == 0 {
		return
	}
	totalValue := 0
	for _, item := range m.cargo {
		mineral := mlMinerals[item.cellType]
		totalValue += mineral.value * item.count
	}
	m.money += totalValue
	m.cargo = nil
	m.cargoWt = 0
	m.fuel = m.maxFuel // refuel at surface
	if totalValue > 0 {
		if mlSFXSell != nil {
			playSFX(mlSFXSell, 0.30)
		}
		m.mlSetMsg(fmt.Sprintf("Sold minerals for $%d!", totalValue))
	}
}

func (m *Motherlode) mlUpdateCamera() {
	// Camera target: keep player centered vertically
	targetCamY := m.py - float64(mlScreenH)/2 + float64(mlPlayerH)/2
	if targetCamY < -float64(mlSurfaceH) {
		targetCamY = -float64(mlSurfaceH)
	}
	maxCam := float64(mlGridRows*mlCellSize) - float64(mlScreenH)
	if targetCamY > maxCam {
		targetCamY = maxCam
	}

	// Smooth camera
	m.camY += (targetCamY - m.camY) * 0.12
}

// ---------------------------------------------------------------------------
// Draw
// ---------------------------------------------------------------------------

func (m *Motherlode) Draw(screen *ebiten.Image) {
	screen.Fill(mlBgUnderground)

	// Parallax: draw a darker background layer that scrolls slower
	m.mlDrawParallax(screen)

	m.mlDrawSky(screen)
	m.mlDrawGrid(screen)
	m.mlDrawDigParticles(screen)
	m.mlDrawPlayer(screen)
	m.mlDrawHUD(screen)

	if m.state == mlStateShop {
		m.mlDrawShop(screen)
	}

	if m.msgTimer > 0 {
		m.mlDrawMsg(screen)
	}
}

func (m *Motherlode) mlDrawParallax(screen *ebiten.Image) {
	// Slow-moving darker rock layer behind the main grid
	parallaxCamY := m.camY * 0.4
	layerColor := color.RGBA{0x25, 0x18, 0x0d, 0xff}
	for i := 0; i < 25; i++ {
		fi := float64(i)
		rx := float32(math.Mod(fi*73.3+17, float64(mlScreenW)))
		ry := float32(math.Mod(fi*97.7+31, float64(mlGridRows*mlCellSize)) - parallaxCamY)
		if ry < -30 || ry > float32(mlScreenH)+30 {
			continue
		}
		w := float32(15 + math.Mod(fi*13.7, 25))
		h := float32(10 + math.Mod(fi*11.3, 20))
		vector.FillRect(screen, rx, ry, w, h, layerColor, false)
	}
}

func (m *Motherlode) mlDrawDigParticles(screen *ebiten.Image) {
	for _, p := range m.digParticles {
		sx := float32(p.x) - float32(0)
		sy := float32(p.y-m.camY) - float32(0)
		if sy < 0 || sy > float32(mlScreenH) {
			continue
		}
		alpha := uint8(float64(p.life) / 35.0 * 255)
		if alpha > 255 {
			alpha = 255
		}
		sz := float32(1 + p.life/10)
		vector.FillRect(screen, sx, sy, sz, sz, color.RGBA{0x8b, 0x5a, 0x2b, alpha}, false)
	}
}

func (m *Motherlode) mlDrawSky(screen *ebiten.Image) {
	// Sky area is above the grid (y < 0 in world coords)
	skyTop := -float64(mlSurfaceH) - m.camY
	skyBottom := -m.camY

	if skyBottom < 0 {
		return // sky not visible
	}
	if skyTop < 0 {
		skyTop = 0
	}
	if skyBottom > float64(mlScreenH) {
		skyBottom = float64(mlScreenH)
	}

	// Draw sky background
	vector.FillRect(screen, 0, float32(skyTop), float32(mlScreenW), float32(skyBottom-skyTop), mlSkyColor, false)

	// Moving clouds
	t := float64(m.ticks)
	for i := 0; i < 5; i++ {
		fi := float64(i)
		cx := float32(math.Mod(fi*97+t*0.3*(0.5+fi*0.15), float64(mlScreenW+80))) - 40
		cy := float32(skyTop + fi*12 + 5)
		if cy < float32(skyTop) || cy > float32(skyBottom) {
			continue
		}
		cw := float32(30 + fi*8)
		ch := float32(8 + fi*2)
		cloudAlpha := uint8(50 + i*15)
		vector.FillCircle(screen, cx, cy, ch, color.RGBA{0xff, 0xff, 0xff, cloudAlpha}, false)
		vector.FillCircle(screen, cx+cw*0.3, cy-ch*0.3, ch*0.8, color.RGBA{0xff, 0xff, 0xff, cloudAlpha}, false)
		vector.FillCircle(screen, cx-cw*0.3, cy+ch*0.15, ch*0.7, color.RGBA{0xff, 0xff, 0xff, cloudAlpha}, false)
	}

	// Draw grass strip at ground level
	grassY := -m.camY - 8
	if grassY >= 0 && grassY < float64(mlScreenH) {
		vector.FillRect(screen, 0, float32(grassY), float32(mlScreenW), 8, mlGrassColor, false)
	}

	// Draw shop building
	shopX := float32(280)
	shopW := float32(60)
	shopH := float32(40)
	shopY := float32(-m.camY) - shopH - 8

	if shopY+shopH > 0 && shopY < float32(mlScreenH) {
		// Building body
		vector.FillRect(screen, shopX, shopY, shopW, shopH, mlShopColor, false)
		// Roof
		var path vector.Path
		path.MoveTo(shopX-5, shopY)
		path.LineTo(shopX+shopW/2, shopY-20)
		path.LineTo(shopX+shopW+5, shopY)
		path.Close()
		vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].ColorR = float32(mlShopRoofColor.R) / 255
			vs[i].ColorG = float32(mlShopRoofColor.G) / 255
			vs[i].ColorB = float32(mlShopRoofColor.B) / 255
			vs[i].ColorA = float32(mlShopRoofColor.A) / 255
		}
		screen.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{})
		// Door
		vector.FillRect(screen, shopX+22, shopY+20, 16, 20, color.RGBA{0x55, 0x33, 0x11, 0xff}, false)
		// Sign
		ebitenutil.DebugPrintAt(screen, "SHOP", int(shopX)+15, int(shopY)+5)
	}

	// Draw surface ground (brown soil at top)
	soilTop := -m.camY
	if soilTop >= 0 && soilTop < float64(mlScreenH) {
		vector.FillRect(screen, 0, float32(soilTop), float32(mlScreenW), 2, color.RGBA{0x66, 0x44, 0x22, 0xff}, false)
	}
}

func (m *Motherlode) mlDrawGrid(screen *ebiten.Image) {
	// Determine visible rows
	startRow := int(m.camY) / mlCellSize
	if startRow < 0 {
		startRow = 0
	}
	endRow := startRow + mlScreenH/mlCellSize + 2
	if endRow > mlGridRows {
		endRow = mlGridRows
	}

	for r := startRow; r < endRow; r++ {
		for c := 0; c < mlGridCols; c++ {
			cell := m.grid[r][c]
			if cell == mlCellEmpty {
				continue
			}

			sx := float32(c * mlCellSize)
			sy := float32(float64(r*mlCellSize) - m.camY)

			if sy+float32(mlCellSize) < 0 || sy > float32(mlScreenH) {
				continue
			}

			var clr color.RGBA
			switch cell {
			case mlCellDirt:
				// Vary dirt color slightly by position
				shade := byte(0x8b - (r+c)%3*8)
				clr = color.RGBA{shade, byte(int(shade) * 2 / 3), byte(int(shade) / 3), 0xff}
			case mlCellRock:
				shade := byte(0x88 - (r*3+c*7)%20)
				clr = color.RGBA{shade, shade, shade, 0xff}
			default:
				if mineral, ok := mlMinerals[cell]; ok {
					clr = mineral.clr
				} else {
					clr = mlDirtColor
				}
			}

			vector.FillRect(screen, sx, sy, float32(mlCellSize), float32(mlCellSize), clr, false)

			// Draw mineral sparkle for precious minerals — animated star shapes
			if cell >= mlCellSilver {
				phase := float64(m.ticks)/8.0 + float64(r*7+c*13)
				sparkleAlpha := uint8(0x66 + int(math.Sin(phase)*0x66))
				sparkleClr := color.RGBA{0xff, 0xff, 0xff, sparkleAlpha}
				scx := sx + float32(mlCellSize)/2
				scy := sy + float32(mlCellSize)/2
				// Animated cross/star shape
				starSize := float32(2 + math.Sin(phase)*1.5)
				vector.FillRect(screen, scx-starSize, scy-0.5, starSize*2, 1, sparkleClr, false)
				vector.FillRect(screen, scx-0.5, scy-starSize, 1, starSize*2, sparkleClr, false)
				// Diagonal arms for diamond/ruby (more precious = more sparkle)
				if cell >= mlCellDiamond {
					ds := starSize * 0.7
					vector.StrokeLine(screen, scx-ds, scy-ds, scx+ds, scy+ds, 0.8, sparkleClr, false)
					vector.StrokeLine(screen, scx+ds, scy-ds, scx-ds, scy+ds, 0.8, sparkleClr, false)
				}
				// Second sparkle offset for extra richness
				if (m.ticks/12+r+c)%4 == 0 {
					offX := float32(3 + (r*3+c*5)%6)
					offY := float32(2 + (r*5+c*3)%6)
					vector.FillRect(screen, sx+offX, sy+offY, 2, 1, color.RGBA{0xff, 0xff, 0xff, 0x60}, false)
					vector.FillRect(screen, sx+offX+0.5, sy+offY-0.5, 1, 2, color.RGBA{0xff, 0xff, 0xff, 0x60}, false)
				}
			}

			// Subtle grid lines
			vector.StrokeRect(screen, sx, sy, float32(mlCellSize), float32(mlCellSize), 0.5,
				color.RGBA{0x00, 0x00, 0x00, 0x22}, false)
		}
	}
}

func (m *Motherlode) mlDrawPlayer(screen *ebiten.Image) {
	sx := float32(m.px)
	sy := float32(m.py - m.camY)

	if sy+float32(mlPlayerH) < 0 || sy > float32(mlScreenH) {
		return
	}

	pw := float32(mlPlayerW)
	ph := float32(mlPlayerH)

	// Tracks/wheels
	vector.FillRect(screen, sx, sy+ph-4, pw, 4, mlTrackColor, false)
	vector.FillRect(screen, sx+1, sy+ph-3, 3, 2, color.RGBA{0x66, 0x66, 0x66, 0xff}, false)
	vector.FillRect(screen, sx+6, sy+ph-3, 3, 2, color.RGBA{0x66, 0x66, 0x66, 0xff}, false)
	vector.FillRect(screen, sx+11, sy+ph-3, 3, 2, color.RGBA{0x66, 0x66, 0x66, 0xff}, false)

	// Body
	vector.FillRect(screen, sx+1, sy+4, pw-2, ph-8, mlPlayerColor, false)

	// Cabin
	vector.FillRect(screen, sx+3, sy+1, pw-6, 8, mlCabinColor, false)

	// Cabin window highlight
	vector.FillRect(screen, sx+5, sy+2, 3, 3, color.RGBA{0xaa, 0xdd, 0xff, 0xff}, false)

	// Drill bit at bottom center
	drillX := sx + pw/2 - 2
	drillY := sy + ph - 2
	vector.FillRect(screen, drillX, drillY, 4, 4, mlDrillColor, false)
	// Drill tip
	vector.FillRect(screen, drillX+1, drillY+3, 2, 2, color.RGBA{0xdd, 0xdd, 0xdd, 0xff}, false)

	// If digging, show digging animation with sparks
	if m.digging {
		// Shake effect
		shakeX := float32((m.ticks%4 - 2))
		vector.FillRect(screen, sx+pw/2-1+shakeX, sy+ph+2, 2, 4, mlDrillColor, false)
		// Debris particles
		for i := 0; i < 3; i++ {
			px := sx + pw/2 + float32(((m.ticks*7+i*13)%12)-6)
			py := sy + ph + float32(((m.ticks*5+i*9)%8))
			vector.FillRect(screen, px, py, 2, 2, mlDirtColor, false)
		}
		// Animated sparks flying off the drill
		for i := 0; i < 5; i++ {
			sparkPhase := float64(m.ticks*11+i*17) / 7.0
			sparkX := sx + pw/2 + float32(math.Sin(sparkPhase)*8)
			sparkY := sy + ph + float32(math.Cos(sparkPhase*1.3)*6) + 2
			sparkAlpha := uint8(180 - (m.ticks*30+i*50)%180)
			sparkClr := color.RGBA{0xff, 0xcc, 0x33, sparkAlpha}
			vector.FillRect(screen, sparkX, sparkY, 1.5, 1.5, sparkClr, false)
		}
	}

	// If flying, show exhaust
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && m.py > -float64(mlPlayerH) {
		for i := 0; i < 3; i++ {
			ex := sx + pw/2 - 3 + float32(((m.ticks*3+i*7)%6))
			ey := sy + ph + 2 + float32(i*3)
			vector.FillRect(screen, ex, ey, 3, 3, mlExhaustColor, false)
		}
	}
}

func (m *Motherlode) mlDrawHUD(screen *ebiten.Image) {
	// Background panel
	vector.FillRect(screen, 0, 0, float32(mlScreenW), 58, color.RGBA{0x00, 0x00, 0x00, 0xcc}, false)

	// Fuel bar
	ebitenutil.DebugPrintAt(screen, "FUEL", 5, 2)
	fuelPct := m.fuel / m.maxFuel
	if fuelPct < 0 {
		fuelPct = 0
	}
	barW := float32(90)
	vector.FillRect(screen, 40, 3, barW, 10, color.RGBA{0x44, 0x44, 0x44, 0xff}, false)
	fuelClr := color.RGBA{0x33, 0xcc, 0x33, 0xff}
	if fuelPct < 0.25 {
		fuelClr = color.RGBA{0xdd, 0x33, 0x33, 0xff}
	} else if fuelPct < 0.5 {
		fuelClr = color.RGBA{0xdd, 0xdd, 0x33, 0xff}
	}
	vector.FillRect(screen, 40, 3, barW*float32(fuelPct), 10, fuelClr, false)
	vector.StrokeRect(screen, 40, 3, barW, 10, 1, color.RGBA{0xaa, 0xaa, 0xaa, 0xff}, false)

	// Cargo bar
	ebitenutil.DebugPrintAt(screen, "CARGO", 5, 16)
	cargoPct := float32(0)
	if m.maxWt > 0 {
		cargoPct = float32(m.cargoWt) / float32(m.maxWt)
	}
	if cargoPct > 1 {
		cargoPct = 1
	}
	vector.FillRect(screen, 50, 17, barW-10, 10, color.RGBA{0x44, 0x44, 0x44, 0xff}, false)
	cargoClr := color.RGBA{0x33, 0x99, 0xcc, 0xff}
	if cargoPct > 0.8 {
		cargoClr = color.RGBA{0xdd, 0x66, 0x33, 0xff}
	}
	vector.FillRect(screen, 50, 17, (barW-10)*cargoPct, 10, cargoClr, false)
	vector.StrokeRect(screen, 50, 17, barW-10, 10, 1, color.RGBA{0xaa, 0xaa, 0xaa, 0xff}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d/%d", m.cargoWt, m.maxWt), 135, 16)

	// Money
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("$%d", m.money), 5, 30)

	// Depth
	depth := int(m.py+float64(mlPlayerH)) / mlCellSize
	if depth < 0 {
		depth = 0
	}
	depthFt := depth * 5 // 5 feet per cell for flavor
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Depth: %dft", depthFt), 5, 44)

	// Right side: cargo contents summary
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Fuel: %.0f/%.0f", m.fuel, m.maxFuel), 200, 2)

	// Controls reminder at bottom
	onSurface := m.py <= -float64(mlPlayerH)+2
	if onSurface {
		ebitenutil.DebugPrintAt(screen, "[S]hop  [Arrows]Move  [R]estart", 50, mlScreenH-14)
	} else {
		ebitenutil.DebugPrintAt(screen, "[UP]Fly  [DOWN]Dig  [L/R]Move", 60, mlScreenH-14)
	}

	// Depth warning
	playerRow := int(m.py+float64(mlPlayerH)) / mlCellSize
	if playerRow > m.maxSafeDepth {
		if m.ticks%40 < 20 {
			ebitenutil.DebugPrintAt(screen, "!! HULL DAMAGE - TOO DEEP !!", 90, 60)
		}
	}
}

func (m *Motherlode) mlDrawShop(screen *ebiten.Image) {
	// Overlay
	vector.FillRect(screen, 0, 0, float32(mlScreenW), float32(mlScreenH), color.RGBA{0x00, 0x00, 0x00, 0xbb}, false)

	// Shop panel
	panelX := float32(30)
	panelY := float32(40)
	panelW := float32(340)
	panelH := float32(400)
	vector.FillRect(screen, panelX, panelY, panelW, panelH, color.RGBA{0x33, 0x22, 0x11, 0xff}, false)
	vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 2, color.RGBA{0xcc, 0x99, 0x44, 0xff}, false)

	ebitenutil.DebugPrintAt(screen, "=== UPGRADE SHOP ===", 130, 55)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Money: $%d", m.money), 140, 75)

	for i := 0; i < mlUpgCount; i++ {
		upg := mlUpgrades[i]
		lvl := m.upgLevels[i]
		y := 100 + i*75

		// Selection highlight
		if i == m.shopSel {
			vector.FillRect(screen, panelX+5, float32(y-3), panelW-10, 68, color.RGBA{0x55, 0x44, 0x22, 0xff}, false)
			vector.StrokeRect(screen, panelX+5, float32(y-3), panelW-10, 68, 1, color.RGBA{0xff, 0xcc, 0x44, 0xff}, false)
		}

		// Name and level
		lvlStr := fmt.Sprintf("%s  [Lv %d/%d]", upg.name, lvl, upg.maxLevel)
		ebitenutil.DebugPrintAt(screen, lvlStr, 50, y)

		// Description
		ebitenutil.DebugPrintAt(screen, upg.desc, 50, y+16)

		// Cost or maxed
		if lvl >= upg.maxLevel {
			ebitenutil.DebugPrintAt(screen, "MAXED", 50, y+32)
		} else {
			cost := upg.costs[lvl]
			costStr := fmt.Sprintf("Cost: $%d", cost)
			clr := "  "
			if m.money < cost {
				clr = " (can't afford)"
			}
			ebitenutil.DebugPrintAt(screen, costStr+clr, 50, y+32)
		}

		// Level pips
		for p := 0; p < upg.maxLevel; p++ {
			px := float32(50 + p*20)
			py := float32(y + 48)
			if p < lvl {
				vector.FillRect(screen, px, py, 14, 8, color.RGBA{0x44, 0xcc, 0x44, 0xff}, false)
			} else {
				vector.FillRect(screen, px, py, 14, 8, color.RGBA{0x44, 0x44, 0x44, 0xff}, false)
			}
			vector.StrokeRect(screen, px, py, 14, 8, 1, color.RGBA{0xaa, 0xaa, 0xaa, 0xff}, false)
		}
	}

	ebitenutil.DebugPrintAt(screen, "[UP/DOWN] Select  [ENTER] Buy  [S/ESC] Close", 48, int(panelY+panelH-25))
}

func (m *Motherlode) mlDrawMsg(screen *ebiten.Image) {
	if m.msgText == "" {
		return
	}
	msgW := len(m.msgText)*6 + 20
	msgX := (mlScreenW - msgW) / 2
	msgY := 65

	alpha := byte(0xff)
	if m.msgTimer < 30 {
		alpha = byte(m.msgTimer * 255 / 30)
	}
	vector.FillRect(screen, float32(msgX), float32(msgY), float32(msgW), 20,
		color.RGBA{0x00, 0x00, 0x00, alpha}, false)
	if alpha > 128 {
		ebitenutil.DebugPrintAt(screen, m.msgText, msgX+10, msgY+4)
	}
}
