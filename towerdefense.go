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
	twGridW    = 20
	twGridH    = 15
	twCellSize = 30
	twUIBarH   = 50
	twScreenW  = twGridW * twCellSize  // 600
	twScreenH  = twGridH*twCellSize + twUIBarH // 500
)

// Tower type IDs
const (
	twTowerArrow  = 0
	twTowerCannon = 1
	twTowerIce    = 2
)

// Game states
const (
	twStatePlaying  = 0
	twStateGameOver = 1
)

// ---------------------------------------------------------------------------
// Path definition — a winding snake pattern across the grid
// ---------------------------------------------------------------------------

// twPathNodes defines the waypoints (grid coords) enemies follow.
// The path snakes from left to right across the map in a winding pattern.
var twPathNodes = []twPoint{
	{0, 2},
	{6, 2},
	{6, 5},
	{1, 5},
	{1, 8},
	{6, 8},
	{6, 12},
	{11, 12},
	{11, 8},
	{14, 8},
	{14, 4},
	{18, 4},
	{18, 10},
	{15, 10},
	{15, 13},
	{19, 13},
}

// twBuildPathCells generates the set of grid cells occupied by the path.
func twBuildPathCells() map[twPoint]bool {
	cells := make(map[twPoint]bool)
	for i := 0; i < len(twPathNodes)-1; i++ {
		a := twPathNodes[i]
		b := twPathNodes[i+1]
		if a.x == b.x {
			// vertical segment
			minY, maxY := a.y, b.y
			if minY > maxY {
				minY, maxY = maxY, minY
			}
			for y := minY; y <= maxY; y++ {
				cells[twPoint{a.x, y}] = true
			}
		} else {
			// horizontal segment
			minX, maxX := a.x, b.x
			if minX > maxX {
				minX, maxX = maxX, minX
			}
			for x := minX; x <= maxX; x++ {
				cells[twPoint{x, a.y}] = true
			}
		}
	}
	return cells
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type twPoint struct {
	x, y int
}

// --- Enemy ---

type twEnemyType struct {
	speed    float64 // pixels per tick
	hp       int
	reward   int
	clr      color.RGBA
	size     float64 // half-size of the square
}

var twEnemyTypes = []twEnemyType{
	// 0: normal
	{speed: 1.0, hp: 30, reward: 5, clr: color.RGBA{0xE0, 0x40, 0x40, 0xFF}, size: 6},
	// 1: fast/weak
	{speed: 1.8, hp: 15, reward: 4, clr: color.RGBA{0xFF, 0xA0, 0x00, 0xFF}, size: 5},
	// 2: slow/tanky
	{speed: 0.6, hp: 80, reward: 10, clr: color.RGBA{0x80, 0x00, 0x80, 0xFF}, size: 8},
	// 3: boss
	{speed: 0.4, hp: 200, reward: 25, clr: color.RGBA{0xFF, 0x00, 0xFF, 0xFF}, size: 10},
}

type twEnemy struct {
	alive      bool
	x, y       float64 // pixel position
	hp         int
	maxHP      int
	speed      float64
	baseSpeed  float64
	reward     int
	clr        color.RGBA
	size       float64
	nodeIndex  int     // index of the next path node to reach
	slowTimer  int     // ticks remaining of slow effect
}

// --- Tower ---

type twTowerDef struct {
	name   string
	cost   int
	damage int
	rng    float64 // range in pixels
	rate   int     // ticks between shots
	splash float64 // splash radius (0 = single target)
	slow   int     // slow duration in ticks (0 = none)
	clr    color.RGBA
}

var twTowerDefs = []twTowerDef{
	// Arrow: cheap, fast, low damage
	{name: "Arrow", cost: 25, damage: 8, rng: 100, rate: 15, splash: 0, slow: 0, clr: color.RGBA{0x00, 0xC0, 0x00, 0xFF}},
	// Cannon: medium, slow, area damage
	{name: "Cannon", cost: 50, damage: 25, rng: 90, rate: 45, splash: 40, slow: 0, clr: color.RGBA{0xD0, 0x20, 0x20, 0xFF}},
	// Ice: medium, slows enemies
	{name: "Ice", cost: 40, damage: 5, rng: 100, rate: 30, splash: 0, slow: 90, clr: color.RGBA{0x40, 0x80, 0xFF, 0xFF}},
}

type twTower struct {
	gx, gy    int // grid position
	towerType int
	cooldown  int
}

// --- Projectile ---

type twProjectile struct {
	alive     bool
	x, y      float64
	targetIdx int // index into enemies slice (-1 if dead)
	speed     float64
	damage    int
	splash    float64
	slow      int
	clr       color.RGBA
}

// --- Wave definition ---

type twWaveEntry struct {
	enemyType int
	count     int
}

type twWaveDef struct {
	entries []twWaveEntry
	hpMul   float64 // multiplier for enemy HP
}

func twBuildWaves() []twWaveDef {
	return []twWaveDef{
		// Wave 1
		{entries: []twWaveEntry{{0, 5}}, hpMul: 1.0},
		// Wave 2
		{entries: []twWaveEntry{{0, 8}}, hpMul: 1.0},
		// Wave 3
		{entries: []twWaveEntry{{1, 6}, {0, 4}}, hpMul: 1.0},
		// Wave 4
		{entries: []twWaveEntry{{0, 8}, {1, 4}}, hpMul: 1.2},
		// Wave 5
		{entries: []twWaveEntry{{2, 3}, {0, 5}}, hpMul: 1.2},
		// Wave 6
		{entries: []twWaveEntry{{1, 10}, {0, 5}}, hpMul: 1.3},
		// Wave 7
		{entries: []twWaveEntry{{2, 5}, {1, 5}}, hpMul: 1.3},
		// Wave 8
		{entries: []twWaveEntry{{0, 12}, {2, 3}}, hpMul: 1.5},
		// Wave 9
		{entries: []twWaveEntry{{1, 8}, {2, 5}}, hpMul: 1.5},
		// Wave 10 — first boss
		{entries: []twWaveEntry{{3, 1}, {0, 10}}, hpMul: 1.5},
		// Wave 11
		{entries: []twWaveEntry{{0, 15}, {1, 8}}, hpMul: 1.8},
		// Wave 12
		{entries: []twWaveEntry{{2, 8}, {1, 6}}, hpMul: 2.0},
		// Wave 13
		{entries: []twWaveEntry{{0, 12}, {2, 6}, {1, 6}}, hpMul: 2.0},
		// Wave 14
		{entries: []twWaveEntry{{1, 12}, {2, 6}, {3, 1}}, hpMul: 2.2},
		// Wave 15
		{entries: []twWaveEntry{{3, 2}, {2, 8}, {1, 8}}, hpMul: 2.5},
		// Wave 16
		{entries: []twWaveEntry{{0, 20}, {2, 8}, {3, 2}}, hpMul: 3.0},
	}
}

// ---------------------------------------------------------------------------
// Audio — Tower Defense
// ---------------------------------------------------------------------------

// twGenerateBGM creates epic strategic battle-preparation music.
// Minor key (A minor / D minor), ~110 bpm, square wave lead + triangle bass.
func twGenerateBGM() []byte {
	// Epic melody in A minor — dramatic, building tension
	// A3=57, B3=59, C4=60, D4=62, E4=64, F4=65, G4=67, A4=69
	melody := [][2]int{
		// Phrase 1 -- ominous march
		{69, 2}, {67, 1}, {65, 1}, {64, 2}, {62, 2},
		{60, 1}, {62, 1}, {64, 2}, {0, 2},
		{69, 1}, {72, 1}, {71, 2}, {69, 1}, {67, 1},
		{65, 2}, {64, 2},
		// Phrase 2 -- rising tension
		{60, 1}, {62, 1}, {64, 1}, {65, 1},
		{67, 2}, {69, 2},
		{72, 2}, {71, 1}, {69, 1},
		{67, 1}, {65, 1}, {64, 2},
		// Phrase 3 -- dramatic peak
		{69, 2}, {72, 2}, {76, 2}, {74, 1}, {72, 1},
		{69, 1}, {67, 1}, {65, 1}, {64, 1},
		{62, 2}, {64, 2},
		{65, 1}, {67, 1}, {69, 2},
		// Phrase 4 -- resolving descent
		{72, 1}, {71, 1}, {69, 2},
		{67, 1}, {65, 1}, {64, 2},
		{62, 1}, {60, 1}, {57, 2},
		{57, 2}, {0, 2},
		// B section -- Dm modulation, heroic countermelody
		{62, 2}, {65, 1}, {69, 1}, {72, 2}, {74, 2},
		{72, 1}, {69, 1}, {67, 2}, {65, 2},
		{62, 1}, {65, 1}, {67, 1}, {69, 1},
		{72, 2}, {74, 2},
		{76, 2}, {74, 1}, {72, 1}, {69, 2}, {67, 2},
		{65, 1}, {62, 1}, {60, 2}, {0, 2},
		// B' -- builds to higher climax
		{69, 1}, {72, 1}, {76, 1}, {79, 1},
		{81, 2}, {79, 1}, {76, 1},
		{74, 1}, {72, 1}, {69, 2}, {67, 2},
		{65, 1}, {67, 1}, {69, 2},
		{72, 1}, {74, 1}, {76, 2},
		{74, 1}, {72, 1}, {69, 2},
		{67, 1}, {65, 1}, {64, 2},
		{62, 2}, {0, 2},
		// Return A -- darker restatement
		{69, 2}, {67, 1}, {65, 1}, {64, 2}, {62, 2},
		{60, 1}, {62, 1}, {64, 2}, {0, 2},
		{69, 1}, {72, 1}, {71, 2}, {69, 1}, {67, 1},
		{65, 2}, {64, 2}, {57, 4}, {0, 2},
	}
	// Dark bass line — power fifths in low register
	bass := [][2]int{
		// A section
		{45, 2}, {45, 2}, {41, 2}, {41, 2},
		{36, 2}, {38, 2}, {40, 2}, {40, 2},
		{45, 2}, {45, 2}, {43, 2}, {43, 2},
		{41, 2}, {41, 2}, {40, 2}, {40, 2},
		{36, 2}, {38, 2}, {40, 2}, {41, 2},
		{43, 2}, {45, 2}, {43, 2}, {40, 2},
		{45, 2}, {48, 2}, {52, 2}, {48, 2},
		{45, 2}, {43, 2}, {41, 2}, {40, 2},
		{45, 2}, {43, 2}, {41, 2}, {40, 2},
		{38, 2}, {36, 2}, {33, 2}, {33, 2},
		// B section -- Dm bass with movement
		{38, 2}, {38, 2}, {41, 2}, {45, 2},
		{43, 2}, {41, 2}, {38, 2}, {38, 2},
		{38, 2}, {41, 2}, {43, 2}, {45, 2},
		{48, 2}, {45, 2}, {43, 2}, {40, 2},
		// B' bass
		{45, 2}, {48, 2}, {52, 2}, {45, 2},
		{43, 2}, {41, 2}, {38, 2}, {40, 2},
		{45, 2}, {43, 2}, {41, 2}, {40, 2},
		{38, 2}, {36, 2}, {38, 2}, {38, 2},
		// Return
		{45, 2}, {45, 2}, {41, 2}, {41, 2},
		{36, 2}, {38, 2}, {40, 2}, {40, 2},
		{45, 2}, {45, 2}, {43, 2}, {43, 2},
		{41, 2}, {38, 2}, {33, 2}, {33, 2},
	}
	drums := generateDrumPattern("K.h.s.h.K.h.sKh.K.h.s.h.KshsKshSK.h.s.hSK.h.sKhSKxh.s.h.K.h.sKhs", 110, 0.30)
	melodyPCM := generateTrack(melody, squareWave, 110, 0.22)
	bassPCM := generateTrack(bass, triangleWave, 110, 0.15)
	mixed := mixBuffers(melodyPCM, bassPCM)
	// Extend drums to match
	for len(drums) < len(mixed) {
		drums = append(drums, drums...)
	}
	drums = drums[:len(mixed)]
	mixed = mixBuffers(mixed, drums)
	return mixed
}

// twGenerateSFXPlace creates a short confirmation beep (tower placed).
func twGenerateSFXPlace() []byte {
	dur := 0.12
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(samples)
		env := envelope(tFrac, 0.05, 0.4, 0.55)
		val := triangleWave(t, noteFreq(72))*0.5 + squareWave(t, noteFreq(79))*0.3
		val *= env * 0.40
		writeSample(buf, i*4, val)
	}
	return buf
}

// twGenerateSFXShoot creates a quick zap sound (projectile fired).
func twGenerateSFXShoot() []byte {
	dur := 0.08
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(samples)
		// Rising frequency zap
		freq := 400.0 + 1200.0*tFrac
		env := 1.0 - tFrac
		val := squareWave(t, freq) * env * 0.25
		writeSample(buf, i*4, val)
	}
	return buf
}

// twGenerateSFXEnemyDeath creates a short pop (enemy destroyed).
func twGenerateSFXEnemyDeath() []byte {
	dur := 0.15
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(samples)
		// Descending pop with noise burst
		freq := 600.0 * math.Exp(-tFrac*3)
		env := (1.0 - tFrac) * (1.0 - tFrac)
		val := (sineWave(t, freq)*0.6 + noise()*0.4) * env * 0.30
		writeSample(buf, i*4, val)
	}
	return buf
}

// twGenerateSFXWaveStart creates an alarm sound (wave starting).
func twGenerateSFXWaveStart() []byte {
	// Two-tone alarm: alternating high-low
	notes := [][2]int{
		{76, 1}, {72, 1}, {76, 1}, {72, 1},
		{79, 2},
	}
	return generateTrack(notes, squareWave, 360, 0.35)
}

// ---------------------------------------------------------------------------
// TowerDefense — main game struct
// ---------------------------------------------------------------------------

// twParticle represents a small visual explosion particle.
type twParticle struct {
	x, y   float64
	vx, vy float64
	life   int
	clr    color.RGBA
}

type TowerDefense struct {
	goBack bool

	state      int
	pathCells  map[twPoint]bool

	towers      []twTower
	enemies     []twEnemy
	projectiles []twProjectile
	particles   []twParticle

	gold       int
	lives      int
	score      int
	wave       int // 0-indexed current wave
	waveActive bool
	waveDefs   []twWaveDef

	// wave spawning state
	spawnQueue    []int // enemy type indices to spawn
	spawnTimer    int
	spawnInterval int

	selectedTower int // 0,1,2

	ticks int

	// Audio
	bgmPlayer       *audio.Player
	sfxPlace         []byte
	sfxShoot         []byte
	sfxEnemyDeath    []byte
	sfxWaveStart     []byte
}

func newTowerDefense() *TowerDefense {
	td := &TowerDefense{}
	td.init()
	return td
}

func (td *TowerDefense) init() {
	td.state = twStatePlaying
	td.pathCells = twBuildPathCells()
	td.towers = nil
	td.enemies = nil
	td.projectiles = nil
	td.gold = 100
	td.lives = 20
	td.score = 0
	td.wave = 0
	td.waveActive = false
	td.waveDefs = twBuildWaves()
	td.spawnQueue = nil
	td.spawnTimer = 0
	td.spawnInterval = 20
	td.selectedTower = twTowerArrow
	td.ticks = 0
	td.goBack = false

	// Initialise audio (only generate PCM once)
	if td.sfxPlace == nil {
		td.sfxPlace = twGenerateSFXPlace()
		td.sfxShoot = twGenerateSFXShoot()
		td.sfxEnemyDeath = twGenerateSFXEnemyDeath()
		td.sfxWaveStart = twGenerateSFXWaveStart()
	}
	if td.bgmPlayer == nil {
		td.bgmPlayer = loopPlayer(twGenerateBGM(), 0.18)
	}
	if !td.bgmPlayer.IsPlaying() {
		td.bgmPlayer.Play()
	}
}

// ---------------------------------------------------------------------------
// SubGame interface
// ---------------------------------------------------------------------------

func (td *TowerDefense) WantsBack() bool {
	return td.goBack
}

func (td *TowerDefense) Layout(w, h int) (int, int) {
	return twScreenW, twScreenH
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (td *TowerDefense) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if td.bgmPlayer != nil {
			td.bgmPlayer.Pause()
		}
		td.goBack = true
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		td.init()
		return nil
	}

	// Tower type selection
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		td.selectedTower = twTowerArrow
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		td.selectedTower = twTowerCannon
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		td.selectedTower = twTowerIce
	}

	if td.state == twStateGameOver {
		return nil
	}

	td.ticks++

	// Start wave with Space or button click
	if !td.waveActive && td.wave < len(td.waveDefs) {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			td.startWave()
		}
		// Check "NEXT WAVE" button click
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			if td.twInNextWaveButton(mx, my) {
				td.startWave()
			}
		}
	}

	// Place tower on click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && td.state == twStatePlaying {
		mx, my := ebiten.CursorPosition()
		gx, gy := mx/twCellSize, my/twCellSize
		if gx >= 0 && gx < twGridW && gy >= 0 && gy < twGridH {
			td.tryPlaceTower(gx, gy)
		}
	}

	// Spawn enemies from queue
	td.updateSpawning()

	// Move enemies
	td.updateEnemies()

	// Towers shoot
	td.updateTowers()

	// Move projectiles
	td.updateProjectiles()

	// Update particles
	td.updateParticles()

	// Check wave complete
	td.checkWaveComplete()

	return nil
}

// ---------------------------------------------------------------------------
// Wave management
// ---------------------------------------------------------------------------

func (td *TowerDefense) startWave() {
	if td.wave >= len(td.waveDefs) {
		return
	}
	playSFX(td.sfxWaveStart, 0.40)
	td.waveActive = true
	wd := td.waveDefs[td.wave]
	td.spawnQueue = nil
	for _, entry := range wd.entries {
		for j := 0; j < entry.count; j++ {
			td.spawnQueue = append(td.spawnQueue, entry.enemyType)
		}
	}
	// Shuffle the spawn queue a bit for variety
	for i := range td.spawnQueue {
		j := rand.Intn(len(td.spawnQueue))
		td.spawnQueue[i], td.spawnQueue[j] = td.spawnQueue[j], td.spawnQueue[i]
	}
	td.spawnTimer = 0
}

func (td *TowerDefense) updateSpawning() {
	if len(td.spawnQueue) == 0 {
		return
	}
	td.spawnTimer++
	if td.spawnTimer >= td.spawnInterval {
		td.spawnTimer = 0
		eType := td.spawnQueue[0]
		td.spawnQueue = td.spawnQueue[1:]
		td.spawnEnemy(eType)
	}
}

func (td *TowerDefense) spawnEnemy(eType int) {
	et := twEnemyTypes[eType]
	hpMul := td.waveDefs[td.wave].hpMul
	hp := int(float64(et.hp) * hpMul)
	if hp < 1 {
		hp = 1
	}
	startNode := twPathNodes[0]
	e := twEnemy{
		alive:     true,
		x:         float64(startNode.x*twCellSize + twCellSize/2),
		y:         float64(startNode.y*twCellSize + twCellSize/2),
		hp:        hp,
		maxHP:     hp,
		speed:     et.speed,
		baseSpeed: et.speed,
		reward:    et.reward,
		clr:       et.clr,
		size:      et.size,
		nodeIndex: 1,
		slowTimer: 0,
	}
	td.enemies = append(td.enemies, e)
}

func (td *TowerDefense) checkWaveComplete() {
	if !td.waveActive {
		return
	}
	if len(td.spawnQueue) > 0 {
		return
	}
	for i := range td.enemies {
		if td.enemies[i].alive {
			return
		}
	}
	// All enemies dead or leaked, wave done
	td.waveActive = false
	td.wave++
	td.enemies = nil
	td.projectiles = nil
}

// ---------------------------------------------------------------------------
// Enemy movement
// ---------------------------------------------------------------------------

func (td *TowerDefense) updateEnemies() {
	for i := range td.enemies {
		e := &td.enemies[i]
		if !e.alive {
			continue
		}

		// Handle slow effect
		if e.slowTimer > 0 {
			e.slowTimer--
			e.speed = e.baseSpeed * 0.4
		} else {
			e.speed = e.baseSpeed
		}

		if e.nodeIndex >= len(twPathNodes) {
			// Reached the end
			e.alive = false
			td.lives--
			if td.lives <= 0 {
				td.lives = 0
				td.state = twStateGameOver
			}
			continue
		}

		target := twPathNodes[e.nodeIndex]
		tx := float64(target.x*twCellSize + twCellSize/2)
		ty := float64(target.y*twCellSize + twCellSize/2)
		dx := tx - e.x
		dy := ty - e.y
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist < e.speed+1 {
			e.x = tx
			e.y = ty
			e.nodeIndex++
		} else {
			e.x += (dx / dist) * e.speed
			e.y += (dy / dist) * e.speed
		}
	}
}

// ---------------------------------------------------------------------------
// Tower logic
// ---------------------------------------------------------------------------

func (td *TowerDefense) tryPlaceTower(gx, gy int) {
	p := twPoint{gx, gy}
	// Can't place on path
	if td.pathCells[p] {
		return
	}
	// Can't place on existing tower
	for _, t := range td.towers {
		if t.gx == gx && t.gy == gy {
			return
		}
	}
	def := twTowerDefs[td.selectedTower]
	if td.gold < def.cost {
		return
	}
	td.gold -= def.cost
	td.towers = append(td.towers, twTower{
		gx:        gx,
		gy:        gy,
		towerType: td.selectedTower,
		cooldown:  0,
	})
	playSFX(td.sfxPlace, 0.40)
}

func (td *TowerDefense) updateTowers() {
	for i := range td.towers {
		t := &td.towers[i]
		if t.cooldown > 0 {
			t.cooldown--
			continue
		}
		def := twTowerDefs[t.towerType]
		cx := float64(t.gx*twCellSize + twCellSize/2)
		cy := float64(t.gy*twCellSize + twCellSize/2)

		// Find nearest enemy in range
		bestIdx := -1
		bestDist := math.MaxFloat64
		for ei := range td.enemies {
			e := &td.enemies[ei]
			if !e.alive {
				continue
			}
			dx := e.x - cx
			dy := e.y - cy
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= def.rng && d < bestDist {
				bestDist = d
				bestIdx = ei
			}
		}

		if bestIdx >= 0 {
			t.cooldown = def.rate
			proj := twProjectile{
				alive:     true,
				x:         cx,
				y:         cy,
				targetIdx: bestIdx,
				speed:     4.0,
				damage:    def.damage,
				splash:    def.splash,
				slow:      def.slow,
				clr:       def.clr,
			}
			td.projectiles = append(td.projectiles, proj)
			playSFX(td.sfxShoot, 0.15)
		}
	}
}

// ---------------------------------------------------------------------------
// Projectile logic
// ---------------------------------------------------------------------------

func (td *TowerDefense) updateProjectiles() {
	for i := range td.projectiles {
		p := &td.projectiles[i]
		if !p.alive {
			continue
		}

		// Find target position
		var tx, ty float64
		hasTarget := false
		if p.targetIdx >= 0 && p.targetIdx < len(td.enemies) && td.enemies[p.targetIdx].alive {
			tx = td.enemies[p.targetIdx].x
			ty = td.enemies[p.targetIdx].y
			hasTarget = true
		} else {
			// Try to retarget nearest enemy
			bestIdx := -1
			bestDist := math.MaxFloat64
			for ei := range td.enemies {
				if !td.enemies[ei].alive {
					continue
				}
				dx := td.enemies[ei].x - p.x
				dy := td.enemies[ei].y - p.y
				d := math.Sqrt(dx*dx + dy*dy)
				if d < bestDist {
					bestDist = d
					bestIdx = ei
				}
			}
			if bestIdx >= 0 {
				p.targetIdx = bestIdx
				tx = td.enemies[bestIdx].x
				ty = td.enemies[bestIdx].y
				hasTarget = true
			}
		}

		if !hasTarget {
			p.alive = false
			continue
		}

		dx := tx - p.x
		dy := ty - p.y
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist < p.speed+2 {
			// Hit!
			td.applyProjectileHit(p)
			p.alive = false
		} else {
			p.x += (dx / dist) * p.speed
			p.y += (dy / dist) * p.speed
		}
	}
}

func (td *TowerDefense) applyProjectileHit(p *twProjectile) {
	if p.splash > 0 {
		// Area damage
		for ei := range td.enemies {
			e := &td.enemies[ei]
			if !e.alive {
				continue
			}
			dx := e.x - p.x
			dy := e.y - p.y
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= p.splash {
				td.damageEnemy(ei, p.damage, p.slow)
			}
		}
	} else {
		// Single target
		if p.targetIdx >= 0 && p.targetIdx < len(td.enemies) && td.enemies[p.targetIdx].alive {
			td.damageEnemy(p.targetIdx, p.damage, p.slow)
		}
	}
}

func (td *TowerDefense) damageEnemy(idx int, damage int, slow int) {
	e := &td.enemies[idx]
	e.hp -= damage
	if slow > 0 && e.slowTimer < slow {
		e.slowTimer = slow
	}
	if e.hp <= 0 {
		// Spawn death particles
		for i := 0; i < 8; i++ {
			angle := float64(i) * math.Pi / 4
			td.particles = append(td.particles, twParticle{
				x: e.x, y: e.y,
				vx: math.Cos(angle) * (1.5 + rand.Float64()),
				vy: math.Sin(angle) * (1.5 + rand.Float64()),
				life: 15 + rand.Intn(10),
				clr:  e.clr,
			})
		}
		e.alive = false
		td.gold += e.reward
		td.score += e.reward
		playSFX(td.sfxEnemyDeath, 0.30)
	}
}

func (td *TowerDefense) updateParticles() {
	alive := td.particles[:0]
	for i := range td.particles {
		p := &td.particles[i]
		p.x += p.vx
		p.y += p.vy
		p.life--
		if p.life > 0 {
			alive = append(alive, *p)
		}
	}
	td.particles = alive
}

// ---------------------------------------------------------------------------
// UI helpers
// ---------------------------------------------------------------------------

func (td *TowerDefense) twInNextWaveButton(mx, my int) bool {
	bx, by, bw, bh := 480, twGridH*twCellSize+8, 110, 34
	return mx >= bx && mx < bx+bw && my >= by && my < by+bh
}

// ---------------------------------------------------------------------------
// Draw
// ---------------------------------------------------------------------------

func (td *TowerDefense) Draw(screen *ebiten.Image) {
	// Background: grass green
	screen.Fill(color.RGBA{0x3A, 0x7D, 0x28, 0xFF})

	td.drawPath(screen)
	td.drawTowers(screen)
	td.drawEnemies(screen)
	td.drawProjectiles(screen)
	td.drawParticles(screen)
	td.drawHoverRange(screen)
	td.drawUI(screen)

	if td.state == twStateGameOver {
		td.drawGameOver(screen)
	}
}

func (td *TowerDefense) drawPath(screen *ebiten.Image) {
	pathColor := color.RGBA{0xC8, 0xA8, 0x6E, 0xFF}      // tan/brown dirt
	pathBorder := color.RGBA{0xA0, 0x80, 0x50, 0xFF}      // darker border

	for p := range td.pathCells {
		fx := float32(p.x * twCellSize)
		fy := float32(p.y * twCellSize)
		vector.FillRect(screen, fx, fy, float32(twCellSize), float32(twCellSize), pathColor, false)
	}
	// Draw subtle grid lines on the path for texture
	for p := range td.pathCells {
		fx := float32(p.x * twCellSize)
		fy := float32(p.y * twCellSize)
		vector.StrokeRect(screen, fx, fy, float32(twCellSize), float32(twCellSize), 0.5, pathBorder, false)
	}
}

func (td *TowerDefense) drawTowers(screen *ebiten.Image) {
	for _, t := range td.towers {
		def := twTowerDefs[t.towerType]
		cx := float32(t.gx*twCellSize + twCellSize/2)
		cy := float32(t.gy*twCellSize + twCellSize/2)

		// Pulsing range circle for all towers
		rangePulse := float32(math.Sin(float64(td.ticks)*0.05+float64(t.gx+t.gy)) * 4)
		rangeR := float32(def.rng) + rangePulse
		rangeAlpha := uint8(15 + int(math.Sin(float64(td.ticks)*0.05+float64(t.gx))*10))
		vector.StrokeCircle(screen, cx, cy, rangeR, 0.5, color.RGBA{def.clr.R, def.clr.G, def.clr.B, rangeAlpha}, false)

		// Glow effect when tower is firing (cooldown > 0 means it just fired)
		if t.cooldown > def.rate/2 {
			glowAlpha := uint8(float64(t.cooldown) / float64(def.rate) * 80)
			vector.FillCircle(screen, cx, cy, float32(twCellSize/2+4), color.RGBA{def.clr.R, def.clr.G, def.clr.B, glowAlpha}, false)
		}

		// Draw tower as colored circle
		vector.DrawFilledCircle(screen, cx, cy, float32(twCellSize/2-2), def.clr, false)
		// Draw a darker ring
		darkerClr := color.RGBA{def.clr.R / 2, def.clr.G / 2, def.clr.B / 2, 0xFF}
		vector.StrokeCircle(screen, cx, cy, float32(twCellSize/2-2), 2, darkerClr, false)
		// Small inner dot for style
		vector.DrawFilledCircle(screen, cx, cy, 3, color.RGBA{0xFF, 0xFF, 0xFF, 0xAA}, false)
	}
}

func (td *TowerDefense) drawEnemies(screen *ebiten.Image) {
	for i := range td.enemies {
		e := &td.enemies[i]
		if !e.alive {
			continue
		}
		// Draw enemy as colored square
		sz := float32(e.size)
		ex := float32(e.x) - sz
		ey := float32(e.y) - sz
		clr := e.clr
		if e.slowTimer > 0 {
			// Tint blue-ish when slowed
			clr = color.RGBA{
				uint8((int(clr.R) + 0x40) / 2),
				uint8((int(clr.G) + 0x80) / 2),
				uint8((int(clr.B) + 0xFF) / 2),
				0xFF,
			}
		}
		vector.FillRect(screen, ex, ey, sz*2, sz*2, clr, false)
		// HP bar
		if e.hp < e.maxHP {
			barW := sz * 2
			barH := float32(3)
			barY := ey - barH - 2
			// Background (red)
			vector.FillRect(screen, ex, barY, barW, barH, color.RGBA{0xCC, 0x00, 0x00, 0xFF}, false)
			// Foreground (green)
			hpFrac := float32(e.hp) / float32(e.maxHP)
			vector.FillRect(screen, ex, barY, barW*hpFrac, barH, color.RGBA{0x00, 0xCC, 0x00, 0xFF}, false)
		}
	}
}

func (td *TowerDefense) drawProjectiles(screen *ebiten.Image) {
	for i := range td.projectiles {
		p := &td.projectiles[i]
		if !p.alive {
			continue
		}
		// Fading trail behind projectile (draw faded copies behind the projectile)
		if p.targetIdx >= 0 && p.targetIdx < len(td.enemies) && td.enemies[p.targetIdx].alive {
			tx := td.enemies[p.targetIdx].x
			ty := td.enemies[p.targetIdx].y
			dx := tx - p.x
			dy := ty - p.y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 0 {
				nx := -dx / dist
				ny := -dy / dist
				for t := 1; t <= 3; t++ {
					trailX := float32(p.x + nx*float64(t)*3)
					trailY := float32(p.y + ny*float64(t)*3)
					trailAlpha := uint8(150 - t*40)
					trailR := float32(3 - t)
					if trailR < 1 {
						trailR = 1
					}
					vector.DrawFilledCircle(screen, trailX, trailY, trailR, color.RGBA{p.clr.R, p.clr.G, p.clr.B, trailAlpha}, false)
				}
			}
		}
		vector.DrawFilledCircle(screen, float32(p.x), float32(p.y), 3, p.clr, false)
		// Bright core
		vector.DrawFilledCircle(screen, float32(p.x), float32(p.y), 1.5, color.RGBA{0xff, 0xff, 0xff, 0xaa}, false)
	}
}

func (td *TowerDefense) drawParticles(screen *ebiten.Image) {
	for _, p := range td.particles {
		alpha := uint8(float64(p.life) / 25.0 * 255)
		if alpha > 255 {
			alpha = 255
		}
		sz := float32(p.life) / 25.0 * 4
		if sz < 1 {
			sz = 1
		}
		c := color.RGBA{p.clr.R, p.clr.G, p.clr.B, alpha}
		vector.FillRect(screen, float32(p.x)-sz/2, float32(p.y)-sz/2, sz, sz, c, false)
	}
}

func (td *TowerDefense) drawHoverRange(screen *ebiten.Image) {
	if td.state != twStatePlaying {
		return
	}
	mx, my := ebiten.CursorPosition()
	gx, gy := mx/twCellSize, my/twCellSize
	if gx < 0 || gx >= twGridW || gy < 0 || gy >= twGridH {
		return
	}
	// Only show if this cell is empty (no path, no tower)
	p := twPoint{gx, gy}
	if td.pathCells[p] {
		return
	}
	for _, t := range td.towers {
		if t.gx == gx && t.gy == gy {
			return
		}
	}
	def := twTowerDefs[td.selectedTower]
	cx := float32(gx*twCellSize + twCellSize/2)
	cy := float32(gy*twCellSize + twCellSize/2)

	// Draw range circle
	rangeClr := color.RGBA{0xFF, 0xFF, 0xFF, 0x30}
	vector.DrawFilledCircle(screen, cx, cy, float32(def.rng), rangeClr, false)
	vector.StrokeCircle(screen, cx, cy, float32(def.rng), 1, color.RGBA{0xFF, 0xFF, 0xFF, 0x80}, false)

	// Ghost tower preview
	ghostClr := color.RGBA{def.clr.R, def.clr.G, def.clr.B, 0x80}
	vector.DrawFilledCircle(screen, cx, cy, float32(twCellSize/2-2), ghostClr, false)
}

func (td *TowerDefense) drawUI(screen *ebiten.Image) {
	// UI bar background
	barY := float32(twGridH * twCellSize)
	vector.FillRect(screen, 0, barY, float32(twScreenW), float32(twUIBarH), color.RGBA{0x20, 0x20, 0x30, 0xFF}, false)
	vector.StrokeRect(screen, 0, barY, float32(twScreenW), float32(twUIBarH), 1, color.RGBA{0x60, 0x60, 0x80, 0xFF}, false)

	uiY := int(barY) + 5

	// Gold, lives, score, wave
	info := fmt.Sprintf("Gold: %d  Lives: %d  Score: %d  Wave: %d/%d",
		td.gold, td.lives, td.score, td.wave+1, len(td.waveDefs))
	ebitenutil.DebugPrintAt(screen, info, 10, uiY)

	// Tower selection
	towerInfo := fmt.Sprintf("[1]Arrow($%d) [2]Cannon($%d) [3]Ice($%d)  Sel: %s",
		twTowerDefs[0].cost, twTowerDefs[1].cost, twTowerDefs[2].cost, twTowerDefs[td.selectedTower].name)
	ebitenutil.DebugPrintAt(screen, towerInfo, 10, uiY+16)

	// Highlight selected tower indicator
	selColors := []color.RGBA{
		twTowerDefs[0].clr,
		twTowerDefs[1].clr,
		twTowerDefs[2].clr,
	}
	for i := 0; i < 3; i++ {
		dotX := float32(10 + i*120)
		dotY := float32(uiY + 34)
		if i == td.selectedTower {
			vector.DrawFilledCircle(screen, dotX+3, dotY+3, 5, selColors[i], false)
		} else {
			vector.DrawFilledCircle(screen, dotX+3, dotY+3, 3, color.RGBA{selColors[i].R, selColors[i].G, selColors[i].B, 0x80}, false)
		}
	}

	// "NEXT WAVE" button (if wave not active and waves remain)
	if !td.waveActive && td.wave < len(td.waveDefs) && td.state == twStatePlaying {
		bx, by := float32(480), barY+8
		bw, bh := float32(110), float32(34)
		mx, my := ebiten.CursorPosition()
		btnClr := color.RGBA{0x40, 0x80, 0x40, 0xFF}
		if mx >= int(bx) && mx < int(bx+bw) && my >= int(by) && my < int(by+bh) {
			btnClr = color.RGBA{0x50, 0xA0, 0x50, 0xFF}
		}
		vector.FillRect(screen, bx, by, bw, bh, btnClr, false)
		vector.StrokeRect(screen, bx, by, bw, bh, 2, color.RGBA{0x80, 0xC0, 0x80, 0xFF}, false)
		ebitenutil.DebugPrintAt(screen, "NEXT WAVE", int(bx)+20, int(by)+10)
	}

	// All waves complete
	if td.wave >= len(td.waveDefs) && td.state == twStatePlaying {
		ebitenutil.DebugPrintAt(screen, "ALL WAVES CLEARED! YOU WIN!", 200, uiY+34)
	}
}

func (td *TowerDefense) drawGameOver(screen *ebiten.Image) {
	// Dim overlay
	vector.FillRect(screen, 0, 0, float32(twScreenW), float32(twScreenH), color.RGBA{0, 0, 0, 0xA0}, false)
	ebitenutil.DebugPrintAt(screen, "GAME OVER", twScreenW/2-30, twScreenH/2-20)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d  Wave: %d", td.score, td.wave+1), twScreenW/2-60, twScreenH/2)
	ebitenutil.DebugPrintAt(screen, "R to restart  |  ESC for menu", twScreenW/2-90, twScreenH/2+20)
}
