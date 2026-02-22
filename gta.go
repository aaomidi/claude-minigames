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

// --------------- constants ---------------

const (
	gtScreenW = 600
	gtScreenH = 480
	gtTileS   = 16
	gtWorldT  = 60 // world is 60x60 tiles
	gtWorldPx = gtWorldT * gtTileS
)

// tile types
const (
	gtTileGrass    = 0
	gtTileRoad     = 1
	gtTileSidewalk = 2
	gtTileBuilding = 3
	gtTileWater    = 4
	gtTilePark     = 5
	gtTileBridge   = 6
	gtTileSpecial  = 7 // gas station, hospital, police, safehouse
)

// car types
const (
	gtCarSedan   = 0
	gtCarSports  = 1
	gtCarTruck   = 2
	gtCarPolice  = 3
	gtCarTaxi    = 4
)

// mission states
const (
	gtMissionNone      = 0
	gtMissionActive    = 1
	gtMissionComplete  = 2
)

// --------------- types ---------------

type gtVec2 struct {
	x, y float64
}

type gtBuilding struct {
	x, y, w, h int // tile coords
	col         color.RGBA
	special     int // 0=normal,1=gas,2=hospital,3=police,4=safehouse
}

type gtCar struct {
	x, y      float64 // world pixel position (center)
	angle     float64 // radians
	vx, vy    float64 // velocity
	speed     float64 // current forward speed
	carType   int
	maxSpeed  float64
	accel     float64
	handling  float64
	hp        float64
	maxHP     float64
	col       color.RGBA
	w, h      float64 // dimensions
	occupied  bool
	smoking   bool
	parked    bool
}

type gtPedestrian struct {
	x, y       float64
	vx, vy     float64
	col        color.RGBA
	alive      bool
	fleeing    bool
	fleeTimer  int
	walkTimer  int
}

type gtCop struct {
	x, y      float64
	vx, vy    float64
	speed     float64
	inCar     bool
	carIdx    int
	alive     bool
}

type gtMission struct {
	name      string
	desc      string
	state     int // none/active/complete
	targetX   float64
	targetY   float64
	markerX   float64
	markerY   float64
	reward    int
	timer     int
	targetNPC int
	carIdx    int
}

type gtParticle struct {
	x, y     float64
	vx, vy   float64
	life     int
	col      color.RGBA
}

type GTA struct {
	goBack bool
	tick   int

	// world
	tiles     [gtWorldT][gtWorldT]int
	buildings []gtBuilding

	// player on foot
	px, py      float64 // position
	pAngle      float64
	pSpeed      float64
	inCar       bool
	carIdx      int
	health      float64
	maxHealth   float64
	stamina     float64
	maxStamina  float64
	sprinting   bool

	// vehicles
	cars []gtCar

	// NPCs
	peds []gtPedestrian
	cops []gtCop

	// wanted
	wantedStars   int
	wantedDecay   int // ticks until star decay
	busted        bool
	bustedTimer   int

	// missions
	missions      []gtMission
	activeMission int // index, -1 = none
	cash          int

	// day/night
	timeOfDay  float64 // 0-1, 0.5 = noon
	timeFast   bool
	nightAlpha float64

	// camera
	camX, camY float64

	// particles
	particles []gtParticle

	// special locations
	policeX, policeY float64
	hospitalX, hospitalY float64
	safehouseX, safehouseY float64

	// audio
	bgmPlayer   *audio.Player
	enginePlayer *audio.Player
	sirenPlayer  *audio.Player
	sfxPunch     []byte
	sfxCrash     []byte
	sfxCash      []byte
	sfxSiren     []byte

	// punch cooldown
	punchCool int

	// message
	msgText  string
	msgTimer int
}

// --------------- constructor ---------------

func newGTA() *GTA {
	g := &GTA{
		health:     100,
		maxHealth:  100,
		stamina:    100,
		maxStamina: 100,
		timeOfDay:  0.35, // morning
		activeMission: -1,
	}

	g.generateWorld()
	g.spawnPlayer()
	g.spawnCars()
	g.spawnPedestrians()
	g.createMissions()
	g.initAudio()

	return g
}

// --------------- world generation ---------------

func (g *GTA) generateWorld() {
	// fill with grass
	for y := 0; y < gtWorldT; y++ {
		for x := 0; x < gtWorldT; x++ {
			g.tiles[y][x] = gtTileGrass
		}
	}

	// Water on right edge (harbor)
	for y := 0; y < gtWorldT; y++ {
		for x := 54; x < gtWorldT; x++ {
			g.tiles[y][x] = gtTileWater
		}
	}

	// Main road network - horizontal roads
	hRoads := []int{5, 15, 25, 35, 45}
	for _, ry := range hRoads {
		for x := 0; x < 54; x++ {
			g.tiles[ry][x] = gtTileRoad
			g.tiles[ry+1][x] = gtTileRoad
			// sidewalks
			if ry > 0 && g.tiles[ry-1][x] != gtTileRoad {
				g.tiles[ry-1][x] = gtTileSidewalk
			}
			if ry+2 < gtWorldT && g.tiles[ry+2][x] != gtTileRoad {
				g.tiles[ry+2][x] = gtTileSidewalk
			}
		}
	}

	// Vertical roads
	vRoads := []int{5, 15, 25, 35, 45}
	for _, rx := range vRoads {
		for y := 0; y < gtWorldT; y++ {
			g.tiles[y][rx] = gtTileRoad
			g.tiles[y][rx+1] = gtTileRoad
			// sidewalks
			if rx > 0 && g.tiles[y][rx-1] != gtTileRoad && g.tiles[y][rx-1] != gtTileSidewalk {
				g.tiles[y][rx-1] = gtTileSidewalk
			}
			if rx+2 < gtWorldT && g.tiles[y][rx+2] != gtTileRoad && g.tiles[y][rx+2] != gtTileSidewalk {
				g.tiles[y][rx+2] = gtTileSidewalk
			}
		}
	}

	// Bridge over water at y=25
	for x := 50; x < gtWorldT; x++ {
		g.tiles[25][x] = gtTileBridge
		g.tiles[26][x] = gtTileBridge
	}

	// Park area (top-left quadrant)
	for y := 7; y < 14; y++ {
		for x := 7; x < 14; x++ {
			if g.tiles[y][x] == gtTileGrass {
				g.tiles[y][x] = gtTilePark
			}
		}
	}

	// Generate buildings in city blocks
	blockDefs := [][4]int{
		// {startX, startY, endX, endY} for each block between roads
		{7, 17, 14, 24},
		{17, 7, 24, 14},
		{17, 17, 24, 24},
		{27, 7, 34, 14},
		{27, 17, 34, 24},
		{37, 7, 44, 14},
		{37, 17, 44, 24},
		{7, 27, 14, 34},
		{17, 27, 24, 34},
		{27, 27, 34, 34},
		{37, 27, 44, 34},
		{7, 37, 14, 44},
		{17, 37, 24, 44},
		{27, 37, 34, 44},
		{37, 37, 44, 44},
		{47, 7, 53, 14},
		{47, 17, 53, 24},
		{47, 27, 53, 34},
		{47, 37, 53, 44},
		{7, 47, 14, 53},
		{17, 47, 24, 53},
		{27, 47, 34, 53},
		{37, 47, 44, 53},
		{47, 47, 53, 53},
	}

	bldgColors := []color.RGBA{
		{0x8B, 0x73, 0x55, 0xFF}, // brown
		{0xD2, 0xC4, 0xA0, 0xFF}, // beige
		{0x88, 0x88, 0x99, 0xFF}, // gray
		{0x66, 0x77, 0x99, 0xFF}, // blue-gray
		{0x99, 0x77, 0x66, 0xFF}, // terracotta
		{0xAA, 0x99, 0x88, 0xFF}, // tan
	}

	for _, bd := range blockDefs {
		sx, sy, ex, ey := bd[0], bd[1], bd[2], bd[3]
		bw := ex - sx
		bh := ey - sy
		if bw < 3 || bh < 3 {
			continue
		}
		// Fill block tiles as building
		for y := sy; y < ey; y++ {
			for x := sx; x < ex; x++ {
				if g.tiles[y][x] == gtTileGrass || g.tiles[y][x] == gtTilePark {
					g.tiles[y][x] = gtTileBuilding
				}
			}
		}
		// Sub-divide into 1-3 buildings
		numBldgs := 1 + rand.Intn(3)
		if bw < 4 {
			numBldgs = 1
		}
		cx := sx
		for bi := 0; bi < numBldgs && cx < ex-1; bi++ {
			remW := ex - cx
			var w int
			if bi == numBldgs-1 {
				w = remW
			} else {
				if remW <= 3 {
					w = remW
				} else {
					w = 2 + rand.Intn(remW-2)
					if w > remW-2 {
						w = remW - 2
					}
				}
			}
			col := bldgColors[rand.Intn(len(bldgColors))]
			g.buildings = append(g.buildings, gtBuilding{
				x: cx, y: sy, w: w, h: bh,
				col: col,
			})
			cx += w
		}
	}

	// Special buildings
	// Police station at block near (17,17)
	g.addSpecialBuilding(17, 17, 3, 3, 3, color.RGBA{0x33, 0x33, 0xAA, 0xFF})
	g.policeX = float64(18)*gtTileS + 8
	g.policeY = float64(20)*gtTileS + 8

	// Hospital at block near (27,7)
	g.addSpecialBuilding(27, 8, 3, 3, 2, color.RGBA{0xFF, 0x44, 0x44, 0xFF})
	g.hospitalX = float64(28)*gtTileS + 8
	g.hospitalY = float64(11)*gtTileS + 8

	// Safehouse at (7,37)
	g.addSpecialBuilding(8, 38, 3, 3, 4, color.RGBA{0x44, 0xAA, 0x44, 0xFF})
	g.safehouseX = float64(9)*gtTileS + 8
	g.safehouseY = float64(41)*gtTileS + 8

	// Gas station at (37,7)
	g.addSpecialBuilding(38, 8, 3, 2, 1, color.RGBA{0xDD, 0xDD, 0x33, 0xFF})
}

func (g *GTA) addSpecialBuilding(x, y, w, h, special int, col color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			if y+dy < gtWorldT && x+dx < gtWorldT {
				g.tiles[y+dy][x+dx] = gtTileSpecial
			}
		}
	}
	g.buildings = append(g.buildings, gtBuilding{
		x: x, y: y, w: w, h: h,
		col:     col,
		special: special,
	})
}

func (g *GTA) spawnPlayer() {
	// Spawn on a road near safehouse
	g.px = float64(9) * gtTileS
	g.py = float64(36) * gtTileS
	g.pAngle = 0
}

func (g *GTA) spawnCars() {
	type carDef struct {
		carType  int
		maxSpeed float64
		accel    float64
		handling float64
		hp       float64
		w, h     float64
		col      color.RGBA
	}
	defs := []carDef{
		{gtCarSedan, 3.5, 0.06, 0.04, 100, 24, 12, color.RGBA{0x44, 0x66, 0xCC, 0xFF}},
		{gtCarSports, 5.0, 0.10, 0.035, 80, 26, 11, color.RGBA{0xDD, 0x22, 0x22, 0xFF}},
		{gtCarTruck, 2.5, 0.04, 0.05, 150, 30, 14, color.RGBA{0x88, 0x88, 0x44, 0xFF}},
		{gtCarPolice, 4.5, 0.08, 0.04, 120, 25, 12, color.RGBA{0x22, 0x22, 0x88, 0xFF}},
		{gtCarTaxi, 3.2, 0.05, 0.04, 100, 24, 12, color.RGBA{0xEE, 0xCC, 0x22, 0xFF}},
	}

	// Place cars along roads
	roadSpots := [][2]float64{}
	for y := 0; y < gtWorldT; y++ {
		for x := 0; x < gtWorldT; x++ {
			if g.tiles[y][x] == gtTileRoad {
				roadSpots = append(roadSpots, [2]float64{
					float64(x)*gtTileS + gtTileS/2,
					float64(y)*gtTileS + gtTileS/2,
				})
			}
		}
	}

	// Place 20 cars
	for i := 0; i < 20; i++ {
		spot := roadSpots[rand.Intn(len(roadSpots))]
		d := defs[rand.Intn(len(defs))]
		angle := float64(rand.Intn(4)) * math.Pi / 2
		c := gtCar{
			x:        spot[0],
			y:        spot[1],
			angle:    angle,
			carType:  d.carType,
			maxSpeed: d.maxSpeed,
			accel:    d.accel,
			handling: d.handling,
			hp:       d.hp,
			maxHP:    d.hp,
			w:        d.w,
			h:        d.h,
			col:      d.col,
			parked:   true,
		}
		g.cars = append(g.cars, c)
	}
}

func (g *GTA) spawnPedestrians() {
	pedColors := []color.RGBA{
		{0xDD, 0x88, 0x44, 0xFF},
		{0x44, 0x88, 0xDD, 0xFF},
		{0xDD, 0x44, 0x88, 0xFF},
		{0x88, 0xDD, 0x44, 0xFF},
		{0xAA, 0x66, 0xAA, 0xFF},
		{0x66, 0xAA, 0xAA, 0xFF},
	}

	for i := 0; i < 20; i++ {
		// Find sidewalk spot
		for attempts := 0; attempts < 100; attempts++ {
			tx := rand.Intn(gtWorldT)
			ty := rand.Intn(gtWorldT)
			if g.tiles[ty][tx] == gtTileSidewalk || g.tiles[ty][tx] == gtTileRoad {
				g.peds = append(g.peds, gtPedestrian{
					x:         float64(tx)*gtTileS + gtTileS/2,
					y:         float64(ty)*gtTileS + gtTileS/2,
					col:       pedColors[rand.Intn(len(pedColors))],
					alive:     true,
					walkTimer: rand.Intn(120),
				})
				break
			}
		}
	}
}

func (g *GTA) createMissions() {
	g.missions = []gtMission{
		{
			name:    "Drive to the Docks",
			desc:    "Drive to the dock warehouse",
			state:   gtMissionNone,
			markerX: float64(16) * gtTileS,
			markerY: float64(46) * gtTileS,
			targetX: float64(52) * gtTileS,
			targetY: float64(26) * gtTileS,
			reward:  500,
		},
		{
			name:    "Eliminate the Target",
			desc:    "Find and punch the target",
			state:   gtMissionNone,
			markerX: float64(26) * gtTileS,
			markerY: float64(36) * gtTileS,
			targetX: float64(36) * gtTileS,
			targetY: float64(16) * gtTileS,
			reward:  750,
			targetNPC: -1,
		},
		{
			name:    "Escape the Cops",
			desc:    "Survive 3 stars for 30 seconds",
			state:   gtMissionNone,
			markerX: float64(36) * gtTileS,
			markerY: float64(26) * gtTileS,
			targetX: 0,
			targetY: 0,
			reward:  1000,
			timer:   30 * 60, // 30 seconds at 60fps
		},
		{
			name:    "Deliver the Car",
			desc:    "Drive the marked car to the garage",
			state:   gtMissionNone,
			markerX: float64(46) * gtTileS,
			markerY: float64(16) * gtTileS,
			targetX: float64(9) * gtTileS,
			targetY: float64(38) * gtTileS,
			reward:  600,
			carIdx:  0,
		},
	}
}

// --------------- audio ---------------

func (g *GTA) initAudio() {
	// BGM - funky hip-hop groove ~100bpm
	bpm := 100.0

	// Bass line - square wave funky groove
	bassNotes := [][2]int{
		{36, 2}, {0, 1}, {36, 1}, {38, 2}, {0, 1}, {38, 1},
		{41, 2}, {0, 1}, {43, 1}, {41, 2}, {0, 1}, {38, 1},
		{36, 2}, {0, 1}, {36, 1}, {38, 2}, {0, 1}, {41, 1},
		{43, 2}, {0, 1}, {41, 1}, {38, 2}, {0, 1}, {36, 1},
		{36, 2}, {36, 1}, {0, 1}, {38, 2}, {38, 1}, {0, 1},
		{41, 2}, {43, 1}, {41, 1}, {38, 2}, {0, 1}, {36, 1},
		{33, 2}, {0, 1}, {33, 1}, {36, 2}, {0, 1}, {38, 1},
		{41, 2}, {0, 1}, {38, 1}, {36, 2}, {0, 2},
	}
	bassTrack := generateTrack(bassNotes, squareWave, bpm, 0.25)

	// Melody - triangle wave
	melodyNotes := [][2]int{
		{60, 1}, {63, 1}, {65, 2}, {0, 1}, {67, 1}, {65, 1}, {63, 1},
		{60, 2}, {0, 1}, {58, 1}, {60, 2}, {0, 2},
		{60, 1}, {63, 1}, {65, 1}, {67, 1}, {70, 2}, {0, 1}, {67, 1},
		{65, 2}, {63, 1}, {60, 1}, {58, 2}, {0, 2},
		{60, 1}, {0, 1}, {63, 1}, {0, 1}, {65, 2}, {67, 1}, {65, 1},
		{63, 2}, {60, 1}, {58, 1}, {60, 2}, {0, 2},
		{58, 1}, {60, 1}, {63, 2}, {65, 1}, {0, 1}, {63, 1}, {60, 1},
		{58, 2}, {0, 1}, {60, 1}, {63, 2}, {0, 2},
	}
	melodyTrack := generateTrack(melodyNotes, triangleWave, bpm, 0.15)

	// Drums
	drumPattern := "K.h.S.h.K.h.S.h.K.h.S.h.K.K.S.h." +
		"K.h.S.h.K.h.S.h.K.h.S.h.K.h.S.K." +
		"K.h.S.h.K.K.S.h.K.h.S.h.K.h.S.h." +
		"K.h.S.h.K.h.S.K.K.h.S.h.K.K.S.h."
	drumTrack := generateDrumPattern(drumPattern, bpm, 0.3)

	// Mix all tracks - match shortest length
	minLen := len(bassTrack)
	if len(melodyTrack) < minLen {
		minLen = len(melodyTrack)
	}
	if len(drumTrack) < minLen {
		minLen = len(drumTrack)
	}
	bassTrack = bassTrack[:minLen]
	melodyTrack = melodyTrack[:minLen]
	drumTrack = drumTrack[:minLen]

	bgm := mixBuffers(bassTrack, melodyTrack)
	bgm = mixBuffers(bgm, drumTrack)

	g.bgmPlayer = loopPlayer(bgm, 0.3)
	g.bgmPlayer.Play()

	// Engine loop
	engBuf := pcmStereo(0.5)
	for i := 0; i < len(engBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		val := sineWave(t, 80) * 0.2
		val += sineWave(t, 120) * 0.1
		val += noise() * 0.05
		writeSample(engBuf, i*4, val)
	}
	g.enginePlayer = loopPlayer(engBuf, 0.0)
	g.enginePlayer.Play()

	// Siren
	sirenBuf := pcmStereo(1.0)
	for i := 0; i < len(sirenBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		freq := 600.0 + 200.0*math.Sin(2*math.Pi*2*t)
		val := sineWave(t, freq) * 0.3
		writeSample(sirenBuf, i*4, val)
	}
	g.sfxSiren = sirenBuf
	g.sirenPlayer = loopPlayer(sirenBuf, 0.0)
	g.sirenPlayer.Play()

	// Punch SFX
	punchBuf := pcmStereo(0.15)
	for i := 0; i < len(punchBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / 0.15
		val := (noise()*0.5 + sineWave(t, 100)*0.5) * (1 - tFrac) * 0.5
		writeSample(punchBuf, i*4, val)
	}
	g.sfxPunch = punchBuf

	// Crash SFX
	crashBuf := pcmStereo(0.3)
	for i := 0; i < len(crashBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / 0.3
		val := noise() * (1 - tFrac) * 0.6
		val += sineWave(t, 60*(1-tFrac*0.5)) * (1 - tFrac) * 0.3
		writeSample(crashBuf, i*4, val)
	}
	g.sfxCrash = crashBuf

	// Cash pickup
	cashBuf := pcmStereo(0.2)
	for i := 0; i < len(cashBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / 0.2
		val := sineWave(t, 880+440*tFrac) * (1 - tFrac) * 0.4
		writeSample(cashBuf, i*4, val)
	}
	g.sfxCash = cashBuf
}

// --------------- helpers ---------------

func gtDist(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

func gtClamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func gtLerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func gtTileAt(x, y float64) (int, int) {
	tx := int(x) / gtTileS
	ty := int(y) / gtTileS
	if tx < 0 {
		tx = 0
	}
	if ty < 0 {
		ty = 0
	}
	if tx >= gtWorldT {
		tx = gtWorldT - 1
	}
	if ty >= gtWorldT {
		ty = gtWorldT - 1
	}
	return tx, ty
}

func gtNormAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

func (g *GTA) isSolid(wx, wy float64) bool {
	tx, ty := gtTileAt(wx, wy)
	t := g.tiles[ty][tx]
	return t == gtTileBuilding || t == gtTileWater || t == gtTileSpecial
}

func (g *GTA) isRoad(wx, wy float64) bool {
	tx, ty := gtTileAt(wx, wy)
	t := g.tiles[ty][tx]
	return t == gtTileRoad || t == gtTileBridge
}

// --------------- update ---------------

func (g *GTA) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.bgmPlayer != nil {
			g.bgmPlayer.Pause()
		}
		if g.enginePlayer != nil {
			g.enginePlayer.Pause()
		}
		if g.sirenPlayer != nil {
			g.sirenPlayer.Pause()
		}
		g.goBack = true
		return nil
	}

	g.tick++

	// Time of day
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		g.timeFast = !g.timeFast
	}
	timeSpeed := 0.00005
	if g.timeFast {
		timeSpeed = 0.002
	}
	g.timeOfDay += timeSpeed
	if g.timeOfDay > 1.0 {
		g.timeOfDay -= 1.0
	}
	// Calculate night darkness
	// 0.0 = midnight, 0.25 = sunrise, 0.5 = noon, 0.75 = sunset, 1.0 = midnight
	dayDist := math.Abs(g.timeOfDay - 0.5)
	g.nightAlpha = gtClamp((dayDist-0.2)*3.0, 0, 0.7)

	// Busted state
	if g.busted {
		g.bustedTimer--
		if g.bustedTimer <= 0 {
			g.busted = false
			g.wantedStars = 0
			g.px = g.policeX
			g.py = g.policeY
			if g.inCar {
				g.exitCar()
			}
			g.health = g.maxHealth
			g.cash -= 200
			if g.cash < 0 {
				g.cash = 0
			}
			g.showMessage("BUSTED! Lost $200")
		}
		return nil
	}

	// Messages
	if g.msgTimer > 0 {
		g.msgTimer--
	}

	if g.inCar {
		g.updateDriving()
	} else {
		g.updateOnFoot()
	}

	g.updatePedestrians()
	g.updateCops()
	g.updateWanted()
	g.updateMissions()
	g.updateParticles()
	g.updateCamera()
	g.updateAudioState()

	// Punch cooldown
	if g.punchCool > 0 {
		g.punchCool--
	}

	return nil
}

func (g *GTA) updateOnFoot() {
	speed := 1.8
	g.sprinting = false
	if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		if g.stamina > 0 {
			speed = 3.5
			g.sprinting = true
			g.stamina -= 0.3
			if g.stamina < 0 {
				g.stamina = 0
			}
		}
	} else {
		g.stamina += 0.15
		if g.stamina > g.maxStamina {
			g.stamina = g.maxStamina
		}
	}

	dx, dy := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		dy = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		dy = 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		dx = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		dx = 1
	}

	if dx != 0 || dy != 0 {
		l := math.Sqrt(dx*dx + dy*dy)
		dx /= l
		dy /= l
		g.pAngle = math.Atan2(dy, dx)

		nx := g.px + dx*speed
		ny := g.py + dy*speed

		// Collision with buildings
		if !g.isSolid(nx, g.py) {
			g.px = nx
		}
		if !g.isSolid(g.px, ny) {
			g.py = ny
		}

		// World bounds
		g.px = gtClamp(g.px, 4, gtWorldPx-4)
		g.py = gtClamp(g.py, 4, gtWorldPx-4)
	}

	// Enter car (E)
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		bestDist := 40.0
		bestIdx := -1
		for i, c := range g.cars {
			if c.occupied {
				continue
			}
			d := gtDist(g.px, g.py, c.x, c.y)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		if bestIdx >= 0 {
			g.enterCar(bestIdx)
		}
	}

	// Punch (F)
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && g.punchCool <= 0 {
		g.punchCool = 20
		playSFX(g.sfxPunch, 0.5)

		// Hit pedestrians
		punchX := g.px + math.Cos(g.pAngle)*16
		punchY := g.py + math.Sin(g.pAngle)*16
		for i := range g.peds {
			if !g.peds[i].alive {
				continue
			}
			d := gtDist(punchX, punchY, g.peds[i].x, g.peds[i].y)
			if d < 18 {
				g.peds[i].alive = false
				// Ragdoll - fly away
				angle := math.Atan2(g.peds[i].y-g.py, g.peds[i].x-g.px)
				g.peds[i].vx = math.Cos(angle) * 5
				g.peds[i].vy = math.Sin(angle) * 5
				g.addWantedStar()

				// Check mission target
				if g.activeMission >= 0 && g.activeMission < len(g.missions) {
					m := &g.missions[g.activeMission]
					if m.name == "Eliminate the Target" && m.targetNPC == i {
						m.state = gtMissionComplete
						g.cash += m.reward
						playSFX(g.sfxCash, 0.6)
						g.showMessage(fmt.Sprintf("Mission Complete! +$%d", m.reward))
						g.activeMission = -1
					}
				}

				// Particles
				for p := 0; p < 5; p++ {
					g.particles = append(g.particles, gtParticle{
						x: g.peds[i].x, y: g.peds[i].y,
						vx: (rand.Float64() - 0.5) * 3,
						vy: (rand.Float64() - 0.5) * 3,
						life: 20 + rand.Intn(20),
						col: color.RGBA{0xFF, 0x00, 0x00, 0xFF},
					})
				}
			}
		}

		// Hit cops
		for i := range g.cops {
			if !g.cops[i].alive {
				continue
			}
			d := gtDist(punchX, punchY, g.cops[i].x, g.cops[i].y)
			if d < 18 {
				g.cops[i].alive = false
				g.addWantedStar()
				g.addWantedStar()
			}
		}
	}
}

func (g *GTA) enterCar(idx int) {
	g.inCar = true
	g.carIdx = idx
	g.cars[idx].occupied = true
	g.cars[idx].parked = false
	// Stealing a car gives wanted star (unless it's already yours)
	if g.cars[idx].carType != gtCarTaxi { // Taxis are free
		g.addWantedStar()
	}
}

func (g *GTA) exitCar() {
	if g.carIdx >= 0 && g.carIdx < len(g.cars) {
		c := &g.cars[g.carIdx]
		c.occupied = false
		c.speed = 0
		c.vx = 0
		c.vy = 0
		// Place player beside car
		g.px = c.x + math.Cos(c.angle+math.Pi/2)*20
		g.py = c.y + math.Sin(c.angle+math.Pi/2)*20
		if g.isSolid(g.px, g.py) {
			g.px = c.x + math.Cos(c.angle-math.Pi/2)*20
			g.py = c.y + math.Sin(c.angle-math.Pi/2)*20
		}
	}
	g.inCar = false
	g.carIdx = -1
}

func (g *GTA) updateDriving() {
	if g.carIdx < 0 || g.carIdx >= len(g.cars) {
		g.inCar = false
		return
	}
	c := &g.cars[g.carIdx]

	// Exit car (E)
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		if math.Abs(c.speed) < 1.5 {
			g.exitCar()
			return
		}
	}

	// Acceleration
	accelInput := 0.0
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		accelInput = 1.0
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		accelInput = -0.6 // reverse is slower
	}

	// Steering
	steerInput := 0.0
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		steerInput = -1.0
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		steerInput = 1.0
	}

	// Handbrake
	handbrake := ebiten.IsKeyPressed(ebiten.KeySpace)

	// Apply acceleration
	c.speed += accelInput * c.accel
	c.speed = gtClamp(c.speed, -c.maxSpeed*0.4, c.maxSpeed)

	// Friction/drag
	if accelInput == 0 {
		c.speed *= 0.98
	}
	if handbrake {
		c.speed *= 0.95
	}
	if math.Abs(c.speed) < 0.01 {
		c.speed = 0
	}

	// Turning - faster turning at low speed, slower at high speed
	if math.Abs(c.speed) > 0.1 {
		turnRate := c.handling
		// Turn more responsive at moderate speeds
		speedFactor := 1.0 / (1.0 + math.Abs(c.speed)*0.15)
		// But need some speed to turn
		speedMin := math.Min(math.Abs(c.speed)/1.5, 1.0)
		turnAmount := steerInput * turnRate * speedFactor * speedMin

		if handbrake && math.Abs(c.speed) > 1.0 {
			// Drift! More turning, less grip
			turnAmount *= 2.5
		}

		// Reverse steering
		if c.speed < 0 {
			turnAmount = -turnAmount
		}

		c.angle += turnAmount
		c.angle = gtNormAngle(c.angle)
	}

	// Calculate velocity from speed and angle
	targetVX := math.Cos(c.angle) * c.speed
	targetVY := math.Sin(c.angle) * c.speed

	if handbrake && math.Abs(c.speed) > 1.0 {
		// Drift: velocity doesn't immediately match facing direction
		c.vx = gtLerp(c.vx, targetVX, 0.05)
		c.vy = gtLerp(c.vy, targetVY, 0.05)
	} else {
		// Normal: velocity mostly matches facing
		c.vx = gtLerp(c.vx, targetVX, 0.15)
		c.vy = gtLerp(c.vy, targetVY, 0.15)
	}

	// Move
	newX := c.x + c.vx
	newY := c.y + c.vy

	// Collision with buildings
	collided := false
	hw, hh := c.w/2+2, c.h/2+2
	corners := [4][2]float64{
		{newX - hw, newY - hh},
		{newX + hw, newY - hh},
		{newX - hw, newY + hh},
		{newX + hw, newY + hh},
	}
	for _, cr := range corners {
		if g.isSolid(cr[0], cr[1]) {
			collided = true
			break
		}
	}

	if collided {
		// Bounce and damage
		c.speed *= -0.3
		c.vx *= -0.3
		c.vy *= -0.3
		c.hp -= math.Abs(c.speed) * 5
		if c.hp < 0 {
			c.hp = 0
		}
		playSFX(g.sfxCrash, 0.4)
		// Particles
		for p := 0; p < 3; p++ {
			g.particles = append(g.particles, gtParticle{
				x: c.x, y: c.y,
				vx: (rand.Float64() - 0.5) * 4,
				vy: (rand.Float64() - 0.5) * 4,
				life: 15 + rand.Intn(10),
				col: color.RGBA{0xFF, 0xAA, 0x00, 0xFF},
			})
		}
	} else {
		c.x = newX
		c.y = newY
	}

	// World bounds
	c.x = gtClamp(c.x, c.w, gtWorldPx-c.w)
	c.y = gtClamp(c.y, c.h, gtWorldPx-c.h)

	// Smoking when damaged
	c.smoking = c.hp < c.maxHP*0.3

	// Hit pedestrians
	for i := range g.peds {
		if !g.peds[i].alive {
			continue
		}
		d := gtDist(c.x, c.y, g.peds[i].x, g.peds[i].y)
		if d < 15 && math.Abs(c.speed) > 0.5 {
			g.peds[i].alive = false
			angle := math.Atan2(g.peds[i].y-c.y, g.peds[i].x-c.x)
			g.peds[i].vx = math.Cos(angle) * 8
			g.peds[i].vy = math.Sin(angle) * 8
			g.addWantedStar()
			for p := 0; p < 5; p++ {
				g.particles = append(g.particles, gtParticle{
					x: g.peds[i].x, y: g.peds[i].y,
					vx: (rand.Float64() - 0.5) * 4,
					vy: (rand.Float64() - 0.5) * 4,
					life: 20 + rand.Intn(15),
					col: color.RGBA{0xFF, 0x00, 0x00, 0xFF},
				})
			}
		}
	}

	// Update player position to car position
	g.px = c.x
	g.py = c.y
	g.pAngle = c.angle
}

func (g *GTA) updatePedestrians() {
	for i := range g.peds {
		p := &g.peds[i]
		if !p.alive {
			// Dead ped slides
			p.x += p.vx
			p.y += p.vy
			p.vx *= 0.9
			p.vy *= 0.9
			continue
		}

		// Check if player is nearby and causing trouble
		distToPlayer := gtDist(p.x, p.y, g.px, g.py)
		if g.wantedStars > 0 && distToPlayer < 80 {
			p.fleeing = true
			p.fleeTimer = 120
		}

		if p.fleeing && p.fleeTimer > 0 {
			p.fleeTimer--
			// Run away from player
			angle := math.Atan2(p.y-g.py, p.x-g.px)
			p.vx = math.Cos(angle) * 2.5
			p.vy = math.Sin(angle) * 2.5
			if p.fleeTimer <= 0 {
				p.fleeing = false
			}
		} else {
			// Random walk
			p.walkTimer--
			if p.walkTimer <= 0 {
				p.walkTimer = 60 + rand.Intn(120)
				angle := rand.Float64() * 2 * math.Pi
				spd := 0.3 + rand.Float64()*0.5
				p.vx = math.Cos(angle) * spd
				p.vy = math.Sin(angle) * spd
			}
		}

		nx := p.x + p.vx
		ny := p.y + p.vy
		if !g.isSolid(nx, ny) && nx > 4 && nx < gtWorldPx-4 && ny > 4 && ny < gtWorldPx-4 {
			p.x = nx
			p.y = ny
		} else {
			p.vx = -p.vx
			p.vy = -p.vy
		}
	}
}

func (g *GTA) updateCops() {
	// Spawn/despawn cops based on wanted level
	desiredCops := g.wantedStars
	if desiredCops > 5 {
		desiredCops = 5
	}

	// Remove excess cops
	for len(g.cops) > desiredCops {
		g.cops = g.cops[:len(g.cops)-1]
	}

	// Spawn new cops if needed
	for len(g.cops) < desiredCops {
		// Spawn cop at edge of screen near player
		angle := rand.Float64() * 2 * math.Pi
		dist := 250.0 + rand.Float64()*100
		cx := g.px + math.Cos(angle)*dist
		cy := g.py + math.Sin(angle)*dist
		cx = gtClamp(cx, 20, gtWorldPx-20)
		cy = gtClamp(cy, 20, gtWorldPx-20)
		g.cops = append(g.cops, gtCop{
			x:     cx,
			y:     cy,
			speed: 2.0 + float64(g.wantedStars)*0.3,
			alive: true,
		})
	}

	// Update cop behavior
	for i := range g.cops {
		cop := &g.cops[i]
		if !cop.alive {
			continue
		}

		// Chase player
		angle := math.Atan2(g.py-cop.y, g.px-cop.x)
		cop.vx = math.Cos(angle) * cop.speed
		cop.vy = math.Sin(angle) * cop.speed

		nx := cop.x + cop.vx
		ny := cop.y + cop.vy
		if !g.isSolid(nx, ny) {
			cop.x = nx
			cop.y = ny
		} else {
			// Try to navigate around
			cop.x += cop.vx * 0.5
			cop.y += cop.vy * 0.5
			if g.isSolid(cop.x, cop.y) {
				cop.x -= cop.vx * 0.5
				cop.y -= cop.vy * 0.5
				// Try perpendicular
				cop.x += cop.vy * 0.5
				cop.y -= cop.vx * 0.5
			}
		}

		// Bust player if close and player is on foot
		if !g.inCar {
			d := gtDist(cop.x, cop.y, g.px, g.py)
			if d < 14 {
				g.busted = true
				g.bustedTimer = 120
				g.showMessage("BUSTED!")
			}
		}
	}
}

func (g *GTA) updateWanted() {
	if g.wantedStars > 0 {
		g.wantedDecay++
		if g.wantedDecay >= 600 { // 10 seconds
			g.wantedStars--
			g.wantedDecay = 0
			if g.wantedStars < 0 {
				g.wantedStars = 0
			}
		}
	} else {
		g.wantedDecay = 0
	}
}

func (g *GTA) addWantedStar() {
	g.wantedStars++
	if g.wantedStars > 5 {
		g.wantedStars = 5
	}
	g.wantedDecay = 0
}

func (g *GTA) updateMissions() {
	// Check if player walks into a mission marker
	if g.activeMission < 0 {
		for i := range g.missions {
			m := &g.missions[i]
			if m.state != gtMissionNone {
				continue
			}
			d := gtDist(g.px, g.py, m.markerX, m.markerY)
			if d < 20 {
				g.activeMission = i
				m.state = gtMissionActive
				g.showMessage("Mission: " + m.name)

				// Special setup
				if m.name == "Eliminate the Target" {
					// Spawn target NPC
					m.targetNPC = len(g.peds)
					g.peds = append(g.peds, gtPedestrian{
						x:     m.targetX,
						y:     m.targetY,
						col:   color.RGBA{0xFF, 0x00, 0x00, 0xFF},
						alive: true,
					})
				}
				if m.name == "Escape the Cops" {
					g.wantedStars = 3
				}
			}
		}
	}

	// Update active mission
	if g.activeMission >= 0 && g.activeMission < len(g.missions) {
		m := &g.missions[g.activeMission]
		if m.state == gtMissionActive {
			switch m.name {
			case "Drive to the Docks":
				d := gtDist(g.px, g.py, m.targetX, m.targetY)
				if d < 30 {
					m.state = gtMissionComplete
					g.cash += m.reward
					playSFX(g.sfxCash, 0.6)
					g.showMessage(fmt.Sprintf("Mission Complete! +$%d", m.reward))
					g.activeMission = -1
				}
			case "Escape the Cops":
				m.timer--
				if m.timer <= 0 {
					m.state = gtMissionComplete
					g.cash += m.reward
					playSFX(g.sfxCash, 0.6)
					g.showMessage(fmt.Sprintf("Mission Complete! +$%d", m.reward))
					g.wantedStars = 0
					g.activeMission = -1
				}
			case "Deliver the Car":
				if g.inCar && g.carIdx == m.carIdx {
					d := gtDist(g.px, g.py, m.targetX, m.targetY)
					if d < 30 {
						m.state = gtMissionComplete
						g.cash += m.reward
						playSFX(g.sfxCash, 0.6)
						g.showMessage(fmt.Sprintf("Mission Complete! +$%d", m.reward))
						g.exitCar()
						g.activeMission = -1
					}
				}
			}
		}
	}
}

func (g *GTA) updateParticles() {
	alive := g.particles[:0]
	for i := range g.particles {
		p := &g.particles[i]
		p.x += p.vx
		p.y += p.vy
		p.vx *= 0.95
		p.vy *= 0.95
		p.life--
		if p.life > 0 {
			alive = append(alive, *p)
		}
	}
	g.particles = alive
}

func (g *GTA) updateCamera() {
	// Smooth camera follow
	targetX := g.px - gtScreenW/2
	targetY := g.py - gtScreenH/2
	g.camX = gtLerp(g.camX, targetX, 0.08)
	g.camY = gtLerp(g.camY, targetY, 0.08)
	// Clamp camera
	g.camX = gtClamp(g.camX, 0, gtWorldPx-gtScreenW)
	g.camY = gtClamp(g.camY, 0, gtWorldPx-gtScreenH)
}

func (g *GTA) updateAudioState() {
	if g.inCar && g.carIdx >= 0 && g.carIdx < len(g.cars) {
		c := &g.cars[g.carIdx]
		vol := math.Abs(c.speed) / c.maxSpeed * 0.3
		g.enginePlayer.SetVolume(gtClamp(vol, 0.02, 0.3))
	} else {
		g.enginePlayer.SetVolume(0)
	}

	if g.wantedStars > 0 {
		g.sirenPlayer.SetVolume(float64(g.wantedStars) * 0.06)
	} else {
		g.sirenPlayer.SetVolume(0)
	}
}

func (g *GTA) showMessage(msg string) {
	g.msgText = msg
	g.msgTimer = 180
}

// --------------- draw ---------------

func (g *GTA) Draw(screen *ebiten.Image) {
	// Sky color based on time of day
	skyR := uint8(gtLerp(100, 20, g.nightAlpha))
	skyG := uint8(gtLerp(140, 20, g.nightAlpha))
	skyB := uint8(gtLerp(180, 40, g.nightAlpha))
	screen.Fill(color.RGBA{skyR, skyG, skyB, 0xFF})

	g.drawWorld(screen)
	g.drawCars(screen)
	g.drawPedestrians(screen)
	g.drawCops(screen)
	if !g.inCar {
		g.drawPlayer(screen)
	}
	g.drawParticles(screen)
	g.drawMissionMarkers(screen)

	// Night overlay
	if g.nightAlpha > 0.05 {
		a := uint8(g.nightAlpha * 180)
		vector.DrawFilledRect(screen, 0, 0, gtScreenW, gtScreenH,
			color.RGBA{0x05, 0x05, 0x15, a}, false)
	}

	g.drawHUD(screen)
	g.drawMinimap(screen)

	// Busted overlay
	if g.busted {
		vector.DrawFilledRect(screen, 0, 0, gtScreenW, gtScreenH,
			color.RGBA{0xFF, 0x00, 0x00, 0x44}, false)
		ebitenutil.DebugPrintAt(screen, "BUSTED!", gtScreenW/2-30, gtScreenH/2-10)
	}
}

func (g *GTA) drawWorld(screen *ebiten.Image) {
	// Calculate visible tile range
	startTX := int(g.camX) / gtTileS
	startTY := int(g.camY) / gtTileS
	endTX := startTX + gtScreenW/gtTileS + 2
	endTY := startTY + gtScreenH/gtTileS + 2

	if startTX < 0 {
		startTX = 0
	}
	if startTY < 0 {
		startTY = 0
	}
	if endTX > gtWorldT {
		endTX = gtWorldT
	}
	if endTY > gtWorldT {
		endTY = gtWorldT
	}

	for ty := startTY; ty < endTY; ty++ {
		for tx := startTX; tx < endTX; tx++ {
			sx := float32(float64(tx*gtTileS) - g.camX)
			sy := float32(float64(ty*gtTileS) - g.camY)

			var col color.RGBA
			switch g.tiles[ty][tx] {
			case gtTileGrass:
				col = color.RGBA{0x44, 0x88, 0x33, 0xFF}
			case gtTileRoad:
				col = color.RGBA{0x44, 0x44, 0x44, 0xFF}
			case gtTileSidewalk:
				col = color.RGBA{0x99, 0x99, 0x88, 0xFF}
			case gtTileBuilding:
				col = color.RGBA{0x77, 0x66, 0x55, 0xFF}
			case gtTileWater:
				waveOff := math.Sin(float64(g.tick)/30.0+float64(tx)*0.5) * 10
				col = color.RGBA{0x22, 0x44, uint8(0xAA + int(waveOff)), 0xFF}
			case gtTilePark:
				col = color.RGBA{0x33, 0x99, 0x33, 0xFF}
			case gtTileBridge:
				col = color.RGBA{0x88, 0x88, 0x77, 0xFF}
			case gtTileSpecial:
				col = color.RGBA{0x88, 0x77, 0x66, 0xFF}
			}

			vector.DrawFilledRect(screen, sx, sy, gtTileS, gtTileS, col, false)

			// Road markings
			if g.tiles[ty][tx] == gtTileRoad {
				// Dashed center lines
				if tx%4 < 2 {
					// Check if this is a horizontal road
					isHRoad := false
					for _, hr := range []int{5, 15, 25, 35, 45} {
						if ty == hr || ty == hr+1 {
							isHRoad = true
							break
						}
					}
					if isHRoad {
						vector.DrawFilledRect(screen, sx+2, sy+float32(gtTileS)/2-1, float32(gtTileS)-4, 2,
							color.RGBA{0xFF, 0xFF, 0xFF, 0x88}, false)
					}
				}
				if ty%4 < 2 {
					isVRoad := false
					for _, vr := range []int{5, 15, 25, 35, 45} {
						if tx == vr || tx == vr+1 {
							isVRoad = true
							break
						}
					}
					if isVRoad {
						vector.DrawFilledRect(screen, sx+float32(gtTileS)/2-1, sy+2, 2, float32(gtTileS)-4,
							color.RGBA{0xFF, 0xFF, 0xFF, 0x88}, false)
					}
				}
			}
		}
	}

	// Draw buildings with outlines
	for _, b := range g.buildings {
		bx := float32(float64(b.x*gtTileS) - g.camX)
		by := float32(float64(b.y*gtTileS) - g.camY)
		bw := float32(b.w * gtTileS)
		bh := float32(b.h * gtTileS)

		// Skip if off screen
		if bx+bw < 0 || bx > gtScreenW || by+bh < 0 || by > gtScreenH {
			continue
		}

		vector.DrawFilledRect(screen, bx+1, by+1, bw-2, bh-2, b.col, false)
		// Darker outline
		outCol := color.RGBA{
			uint8(float64(b.col.R) * 0.6),
			uint8(float64(b.col.G) * 0.6),
			uint8(float64(b.col.B) * 0.6),
			0xFF,
		}
		vector.StrokeRect(screen, bx, by, bw, bh, 1.5, outCol, false)

		// Windows for regular buildings
		if b.special == 0 && bw > 20 && bh > 20 {
			winCol := color.RGBA{0xDD, 0xDD, 0x88, 0x88}
			if g.nightAlpha > 0.3 {
				winCol = color.RGBA{0xFF, 0xFF, 0x88, 0xCC}
			}
			for wy := by + 4; wy < by+bh-6; wy += 8 {
				for wx := bx + 4; wx < bx+bw-6; wx += 8 {
					vector.DrawFilledRect(screen, wx, wy, 4, 4, winCol, false)
				}
			}
		}

		// Special building labels
		if b.special > 0 {
			var label string
			switch b.special {
			case 1:
				label = "GAS"
			case 2:
				label = "H" // Hospital
			case 3:
				label = "PD" // Police
			case 4:
				label = "SAFE"
			}
			ebitenutil.DebugPrintAt(screen, label, int(bx+2), int(by+2))
		}
	}

	// Park trees
	for ty := startTY; ty < endTY; ty++ {
		for tx := startTX; tx < endTX; tx++ {
			if g.tiles[ty][tx] == gtTilePark && (tx+ty)%3 == 0 {
				sx := float32(float64(tx*gtTileS) - g.camX + gtTileS/2)
				sy := float32(float64(ty*gtTileS) - g.camY + gtTileS/2)
				// Tree trunk
				vector.DrawFilledRect(screen, sx-1, sy, 3, 5,
					color.RGBA{0x66, 0x44, 0x22, 0xFF}, false)
				// Canopy
				vector.DrawFilledCircle(screen, sx, sy-2, 5,
					color.RGBA{0x22, 0x77, 0x22, 0xFF}, false)
			}
		}
	}
}

func (g *GTA) drawCars(screen *ebiten.Image) {
	for i := range g.cars {
		c := &g.cars[i]
		cx := c.x - g.camX
		cy := c.y - g.camY

		// Skip if off screen
		if cx < -40 || cx > gtScreenW+40 || cy < -40 || cy > gtScreenH+40 {
			continue
		}

		g.drawCarSprite(screen, float32(cx), float32(cy), c)
	}
}

func (g *GTA) drawCarSprite(screen *ebiten.Image, sx, sy float32, c *gtCar) {
	// Draw rotated car rectangle using corners
	hw := float64(c.w) / 2
	hh := float64(c.h) / 2
	cos := math.Cos(c.angle)
	sin := math.Sin(c.angle)

	// Four corners of the car body
	corners := [4][2]float32{
		{float32(float64(sx) + (-hw)*cos - (-hh)*sin), float32(float64(sy) + (-hw)*sin + (-hh)*cos)},
		{float32(float64(sx) + hw*cos - (-hh)*sin), float32(float64(sy) + hw*sin + (-hh)*cos)},
		{float32(float64(sx) + hw*cos - hh*sin), float32(float64(sy) + hw*sin + hh*cos)},
		{float32(float64(sx) + (-hw)*cos - hh*sin), float32(float64(sy) + (-hw)*sin + hh*cos)},
	}

	// Car body - use triangles
	bodyCol := c.col
	if c.smoking {
		// Flicker when smoking
		if g.tick%10 < 5 {
			bodyCol = color.RGBA{0x44, 0x44, 0x44, 0xFF}
		}
	}

	// Draw car body as two triangles
	vs := [4]ebiten.Vertex{}
	for j := 0; j < 4; j++ {
		vs[j] = ebiten.Vertex{
			DstX:   corners[j][0],
			DstY:   corners[j][1],
			SrcX:   0,
			SrcY:   0,
			ColorR: float32(bodyCol.R) / 255,
			ColorG: float32(bodyCol.G) / 255,
			ColorB: float32(bodyCol.B) / 255,
			ColorA: 1,
		}
	}
	indices := []uint16{0, 1, 2, 0, 2, 3}
	screen.DrawTriangles(vs[:], indices, whitePixel(), &ebiten.DrawTrianglesOptions{})

	// Windshield (front 30% of car)
	windshieldOffset := hw * 0.4
	ww := hw * 0.3
	wh := hh * 0.7
	wcx := float64(sx) + windshieldOffset*cos
	wcy := float64(sy) + windshieldOffset*sin
	wCorners := [4][2]float32{
		{float32(wcx + (-ww)*cos - (-wh)*sin), float32(wcy + (-ww)*sin + (-wh)*cos)},
		{float32(wcx + ww*cos - (-wh)*sin), float32(wcy + ww*sin + (-wh)*cos)},
		{float32(wcx + ww*cos - wh*sin), float32(wcy + ww*sin + wh*cos)},
		{float32(wcx + (-ww)*cos - wh*sin), float32(wcy + (-ww)*sin + wh*cos)},
	}
	wsCol := color.RGBA{0x88, 0xBB, 0xDD, 0xCC}
	var wvs [4]ebiten.Vertex
	for j := 0; j < 4; j++ {
		wvs[j] = ebiten.Vertex{
			DstX:   wCorners[j][0],
			DstY:   wCorners[j][1],
			SrcX:   0,
			SrcY:   0,
			ColorR: float32(wsCol.R) / 255,
			ColorG: float32(wsCol.G) / 255,
			ColorB: float32(wsCol.B) / 255,
			ColorA: float32(wsCol.A) / 255,
		}
	}
	screen.DrawTriangles(wvs[:], indices, whitePixel(), &ebiten.DrawTrianglesOptions{})

	// Wheels (small dark rectangles at corners)
	wheelPositions := [][2]float64{
		{-hw * 0.7, -hh - 1}, {hw * 0.7, -hh - 1},
		{-hw * 0.7, hh + 1}, {hw * 0.7, hh + 1},
	}
	for _, wp := range wheelPositions {
		wx := float32(float64(sx) + wp[0]*cos - wp[1]*sin)
		wy := float32(float64(sy) + wp[0]*sin + wp[1]*cos)
		vector.DrawFilledRect(screen, wx-2, wy-1, 4, 3, color.RGBA{0x22, 0x22, 0x22, 0xFF}, false)
	}

	// Headlights at night
	if g.nightAlpha > 0.2 {
		hlOff := hw + 1
		for _, side := range []float64{-hh * 0.5, hh * 0.5} {
			hlx := float32(float64(sx) + hlOff*cos - side*sin)
			hly := float32(float64(sy) + hlOff*sin + side*cos)
			vector.DrawFilledCircle(screen, hlx, hly, 2,
				color.RGBA{0xFF, 0xFF, 0xAA, 0xCC}, false)
		}
		// Light beam
		if c.occupied {
			beamLen := 40.0
			for _, side := range []float64{-hh * 0.3, hh * 0.3} {
				hlx1 := float64(sx) + hlOff*cos - side*sin
				hly1 := float64(sy) + hlOff*sin + side*cos
				hlx2 := hlx1 + beamLen*cos
				hly2 := hly1 + beamLen*sin
				vector.StrokeLine(screen, float32(hlx1), float32(hly1),
					float32(hlx2), float32(hly2), 3,
					color.RGBA{0xFF, 0xFF, 0x88, 0x22}, false)
			}
		}
	}

	// Police car lights
	if c.carType == gtCarPolice && c.occupied {
		if g.tick%20 < 10 {
			lx := float32(float64(sx) - hh*0.3*sin)
			ly := float32(float64(sy) + hh*0.3*cos)
			vector.DrawFilledCircle(screen, lx, ly, 3, color.RGBA{0xFF, 0x00, 0x00, 0xFF}, false)
		} else {
			lx := float32(float64(sx) + hh*0.3*sin)
			ly := float32(float64(sy) - hh*0.3*cos)
			vector.DrawFilledCircle(screen, lx, ly, 3, color.RGBA{0x00, 0x00, 0xFF, 0xFF}, false)
		}
	}

	// Smoking effect
	if c.smoking && g.tick%6 < 3 {
		smokeX := float32(float64(sx) - hw*cos)
		smokeY := float32(float64(sy) - hw*sin)
		vector.DrawFilledCircle(screen, smokeX, smokeY, 4+float32(rand.Intn(3)),
			color.RGBA{0x88, 0x88, 0x88, 0x88}, false)
	}
}

func (g *GTA) drawPlayer(screen *ebiten.Image) {
	sx := float32(g.px - g.camX)
	sy := float32(g.py - g.camY)

	// Shadow
	vector.DrawFilledCircle(screen, sx+1, sy+2, 5,
		color.RGBA{0x00, 0x00, 0x00, 0x44}, false)

	// Body
	vector.DrawFilledCircle(screen, sx, sy, 5,
		color.RGBA{0xFF, 0xCC, 0x88, 0xFF}, false)

	// Direction indicator (small triangle showing facing)
	dirX := float32(math.Cos(g.pAngle)) * 7
	dirY := float32(math.Sin(g.pAngle)) * 7
	vector.StrokeLine(screen, sx, sy, sx+dirX, sy+dirY, 2,
		color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}, false)

	// Shirt color
	vector.DrawFilledRect(screen, sx-3, sy-2, 6, 5,
		color.RGBA{0x33, 0x33, 0xCC, 0xFF}, false)

	// Sprinting effect
	if g.sprinting {
		vector.StrokeLine(screen, sx-dirX*0.5, sy-dirY*0.5,
			sx-dirX*1.5, sy-dirY*1.5, 1,
			color.RGBA{0xFF, 0xFF, 0xFF, 0x44}, false)
	}

	// Punch animation
	if g.punchCool > 15 {
		fistX := sx + float32(math.Cos(g.pAngle))*12
		fistY := sy + float32(math.Sin(g.pAngle))*12
		vector.DrawFilledCircle(screen, fistX, fistY, 3,
			color.RGBA{0xFF, 0xCC, 0x88, 0xFF}, false)
	}
}

func (g *GTA) drawPedestrians(screen *ebiten.Image) {
	for i := range g.peds {
		p := &g.peds[i]
		sx := float32(p.x - g.camX)
		sy := float32(p.y - g.camY)

		if sx < -10 || sx > gtScreenW+10 || sy < -10 || sy > gtScreenH+10 {
			continue
		}

		if !p.alive {
			// Dead ped - lying down
			vector.DrawFilledRect(screen, sx-4, sy-2, 8, 4,
				color.RGBA{p.col.R / 2, p.col.G / 2, p.col.B / 2, 0xFF}, false)
			continue
		}

		// Shadow
		vector.DrawFilledCircle(screen, sx+1, sy+1, 3,
			color.RGBA{0x00, 0x00, 0x00, 0x33}, false)

		// Body
		vector.DrawFilledCircle(screen, sx, sy, 4, p.col, false)

		// Head
		vector.DrawFilledCircle(screen, sx, sy-3, 2,
			color.RGBA{0xFF, 0xDD, 0xBB, 0xFF}, false)

		// Fleeing indicator
		if p.fleeing {
			ebitenutil.DebugPrintAt(screen, "!", int(sx-2), int(sy-14))
		}
	}

	// Draw mission target indicator
	if g.activeMission >= 0 && g.activeMission < len(g.missions) {
		m := &g.missions[g.activeMission]
		if m.name == "Eliminate the Target" && m.targetNPC >= 0 && m.targetNPC < len(g.peds) {
			p := &g.peds[m.targetNPC]
			if p.alive {
				sx := float32(p.x - g.camX)
				sy := float32(p.y - g.camY)
				// Red arrow above target
				vector.StrokeLine(screen, sx, sy-20, sx, sy-12, 2,
					color.RGBA{0xFF, 0x00, 0x00, 0xFF}, false)
				vector.StrokeLine(screen, sx-4, sy-16, sx, sy-12, 2,
					color.RGBA{0xFF, 0x00, 0x00, 0xFF}, false)
				vector.StrokeLine(screen, sx+4, sy-16, sx, sy-12, 2,
					color.RGBA{0xFF, 0x00, 0x00, 0xFF}, false)
			}
		}
	}
}

func (g *GTA) drawCops(screen *ebiten.Image) {
	for i := range g.cops {
		cop := &g.cops[i]
		if !cop.alive {
			continue
		}
		sx := float32(cop.x - g.camX)
		sy := float32(cop.y - g.camY)

		if sx < -10 || sx > gtScreenW+10 || sy < -10 || sy > gtScreenH+10 {
			continue
		}

		// Shadow
		vector.DrawFilledCircle(screen, sx+1, sy+1, 4,
			color.RGBA{0x00, 0x00, 0x00, 0x33}, false)

		// Blue uniform body
		vector.DrawFilledCircle(screen, sx, sy, 5,
			color.RGBA{0x22, 0x22, 0xBB, 0xFF}, false)

		// Head
		vector.DrawFilledCircle(screen, sx, sy-4, 3,
			color.RGBA{0xFF, 0xDD, 0xBB, 0xFF}, false)

		// Hat
		vector.DrawFilledRect(screen, sx-3, sy-7, 6, 2,
			color.RGBA{0x11, 0x11, 0x66, 0xFF}, false)

		// Badge
		vector.DrawFilledCircle(screen, sx, sy-1, 1.5,
			color.RGBA{0xFF, 0xFF, 0x00, 0xFF}, false)
	}
}

func (g *GTA) drawParticles(screen *ebiten.Image) {
	for _, p := range g.particles {
		sx := float32(p.x - g.camX)
		sy := float32(p.y - g.camY)
		alpha := uint8(float64(p.life) / 40.0 * 255)
		if alpha > 255 {
			alpha = 255
		}
		col := color.RGBA{p.col.R, p.col.G, p.col.B, alpha}
		vector.DrawFilledCircle(screen, sx, sy, 2, col, false)
	}
}

func (g *GTA) drawMissionMarkers(screen *ebiten.Image) {
	// Draw inactive mission markers
	for i := range g.missions {
		m := &g.missions[i]
		if m.state != gtMissionNone {
			continue
		}
		sx := float32(m.markerX - g.camX)
		sy := float32(m.markerY - g.camY)
		if sx < -20 || sx > gtScreenW+20 || sy < -20 || sy > gtScreenH+20 {
			continue
		}
		// Pulsing yellow circle
		pulse := float32(3 + 2*math.Sin(float64(g.tick)*0.1))
		vector.DrawFilledCircle(screen, sx, sy, pulse+5,
			color.RGBA{0xFF, 0xFF, 0x00, 0x88}, false)
		vector.StrokeCircle(screen, sx, sy, pulse+5, 1.5,
			color.RGBA{0xFF, 0xFF, 0x00, 0xFF}, false)
		ebitenutil.DebugPrintAt(screen, "M", int(sx-3), int(sy-5))
	}

	// Draw active mission target
	if g.activeMission >= 0 && g.activeMission < len(g.missions) {
		m := &g.missions[g.activeMission]
		if m.state == gtMissionActive && m.targetX > 0 && m.targetY > 0 {
			sx := float32(m.targetX - g.camX)
			sy := float32(m.targetY - g.camY)
			if sx >= -20 && sx <= gtScreenW+20 && sy >= -20 && sy <= gtScreenH+20 {
				pulse := float32(3 + 2*math.Sin(float64(g.tick)*0.15))
				vector.StrokeCircle(screen, sx, sy, pulse+8, 2,
					color.RGBA{0xFF, 0x44, 0x44, 0xFF}, false)
			}
		}
	}
}

func (g *GTA) drawHUD(screen *ebiten.Image) {
	// Health bar (top left)
	vector.DrawFilledRect(screen, 10, 10, 102, 12, color.RGBA{0x00, 0x00, 0x00, 0x88}, false)
	healthPct := float32(g.health / g.maxHealth)
	hCol := color.RGBA{0x00, 0xCC, 0x00, 0xFF}
	if healthPct < 0.3 {
		hCol = color.RGBA{0xFF, 0x00, 0x00, 0xFF}
	} else if healthPct < 0.6 {
		hCol = color.RGBA{0xFF, 0xAA, 0x00, 0xFF}
	}
	vector.DrawFilledRect(screen, 11, 11, healthPct*100, 10, hCol, false)
	ebitenutil.DebugPrintAt(screen, "HP", 115, 8)

	// Stamina bar (below health)
	vector.DrawFilledRect(screen, 10, 24, 102, 8, color.RGBA{0x00, 0x00, 0x00, 0x88}, false)
	stamPct := float32(g.stamina / g.maxStamina)
	vector.DrawFilledRect(screen, 11, 25, stamPct*100, 6,
		color.RGBA{0x44, 0x88, 0xFF, 0xFF}, false)

	// Car HP bar (when in vehicle)
	if g.inCar && g.carIdx >= 0 && g.carIdx < len(g.cars) {
		c := &g.cars[g.carIdx]
		vector.DrawFilledRect(screen, 10, 34, 102, 8, color.RGBA{0x00, 0x00, 0x00, 0x88}, false)
		carHPPct := float32(c.hp / c.maxHP)
		carCol := color.RGBA{0x88, 0x88, 0xFF, 0xFF}
		if carHPPct < 0.3 {
			carCol = color.RGBA{0xFF, 0x44, 0x44, 0xFF}
		}
		vector.DrawFilledRect(screen, 11, 35, carHPPct*100, 6, carCol, false)
		ebitenutil.DebugPrintAt(screen, "CAR", 115, 32)

		// Speed display
		speed := math.Abs(g.cars[g.carIdx].speed)
		mph := int(speed * 30) // fake mph conversion
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d MPH", mph), 10, 44)
	}

	// Wanted stars (top right)
	for i := 0; i < 5; i++ {
		sx := float32(gtScreenW - 120 + i*22)
		sy := float32(10)
		if i < g.wantedStars {
			g.drawStar(screen, sx, sy+6, 8, color.RGBA{0xFF, 0xDD, 0x00, 0xFF})
		} else {
			g.drawStar(screen, sx, sy+6, 8, color.RGBA{0x44, 0x44, 0x44, 0x88})
		}
	}

	// Cash
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("$%d", g.cash), gtScreenW-120, 26)

	// Weapon indicator
	weaponTxt := "FIST"
	ebitenutil.DebugPrintAt(screen, weaponTxt, gtScreenW-50, 40)

	// Mission text (top center)
	if g.activeMission >= 0 && g.activeMission < len(g.missions) {
		m := &g.missions[g.activeMission]
		if m.state == gtMissionActive {
			txt := m.desc
			if m.name == "Escape the Cops" {
				secs := m.timer / 60
				txt = fmt.Sprintf("%s (%ds)", txt, secs)
			}
			tw := len(txt) * 6
			vector.DrawFilledRect(screen, float32(gtScreenW/2-tw/2-4), 2,
				float32(tw+8), 16, color.RGBA{0x00, 0x00, 0x00, 0xAA}, false)
			ebitenutil.DebugPrintAt(screen, txt, gtScreenW/2-tw/2, 4)
		}
	}

	// Message display
	if g.msgTimer > 0 {
		tw := len(g.msgText) * 6
		vector.DrawFilledRect(screen, float32(gtScreenW/2-tw/2-4), 60,
			float32(tw+8), 16, color.RGBA{0x00, 0x00, 0x00, 0xAA}, false)
		ebitenutil.DebugPrintAt(screen, g.msgText, gtScreenW/2-tw/2, 62)
	}

	// Day/night indicator
	timeStr := "DAY"
	if g.nightAlpha > 0.3 {
		timeStr = "NIGHT"
	}
	if g.timeFast {
		timeStr += " [>>]"
	}
	ebitenutil.DebugPrintAt(screen, timeStr, gtScreenW-60, gtScreenH-16)

	// Controls help
	ebitenutil.DebugPrintAt(screen, "WASD:Move E:Car F:Punch Shift:Run Space:Brake T:Time", 120, gtScreenH-16)
}

func (g *GTA) drawStar(screen *ebiten.Image, cx, cy, r float32, col color.RGBA) {
	// 5-pointed star
	for i := 0; i < 5; i++ {
		a1 := float64(i)*2*math.Pi/5 - math.Pi/2
		a2 := float64(i+1)*2*math.Pi/5 - math.Pi/2
		amid := (a1 + a2) / 2

		x1 := cx + float32(math.Cos(a1))*r
		y1 := cy + float32(math.Sin(a1))*r
		x2 := cx + float32(math.Cos(amid))*r*0.4
		y2 := cy + float32(math.Sin(amid))*r*0.4
		x3 := cx + float32(math.Cos(a2))*r
		y3 := cy + float32(math.Sin(a2))*r

		// Triangle from center to outer points
		vs := [3]ebiten.Vertex{
			{DstX: cx, DstY: cy, SrcX: 0, SrcY: 0,
				ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
			{DstX: x1, DstY: y1, SrcX: 0, SrcY: 0,
				ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
			{DstX: x2, DstY: y2, SrcX: 0, SrcY: 0,
				ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
		}
		screen.DrawTriangles(vs[:], []uint16{0, 1, 2}, whitePixel(), &ebiten.DrawTrianglesOptions{})

		vs2 := [3]ebiten.Vertex{
			{DstX: cx, DstY: cy, SrcX: 0, SrcY: 0,
				ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
			{DstX: x2, DstY: y2, SrcX: 0, SrcY: 0,
				ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
			{DstX: x3, DstY: y3, SrcX: 0, SrcY: 0,
				ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
		}
		screen.DrawTriangles(vs2[:], []uint16{0, 1, 2}, whitePixel(), &ebiten.DrawTrianglesOptions{})
	}
}

func (g *GTA) drawMinimap(screen *ebiten.Image) {
	mmX := float32(10)
	mmY := float32(gtScreenH - 120)
	mmW := float32(100)
	mmH := float32(100)

	// Background
	vector.DrawFilledRect(screen, mmX, mmY, mmW, mmH, color.RGBA{0x00, 0x00, 0x00, 0xCC}, false)
	vector.StrokeRect(screen, mmX, mmY, mmW, mmH, 1.5, color.RGBA{0x88, 0x88, 0x88, 0xFF}, false)

	// Scale: world px -> minimap px
	scaleX := float64(mmW) / float64(gtWorldPx)
	scaleY := float64(mmH) / float64(gtWorldPx)

	// Draw tiles (sampled)
	for ty := 0; ty < gtWorldT; ty += 2 {
		for tx := 0; tx < gtWorldT; tx += 2 {
			mx := mmX + float32(float64(tx*gtTileS)*scaleX)
			my := mmY + float32(float64(ty*gtTileS)*scaleY)
			mw := float32(float64(gtTileS*2) * scaleX)
			mh := float32(float64(gtTileS*2) * scaleY)

			var col color.RGBA
			switch g.tiles[ty][tx] {
			case gtTileGrass, gtTilePark:
				col = color.RGBA{0x22, 0x55, 0x22, 0xFF}
			case gtTileRoad, gtTileBridge:
				col = color.RGBA{0x55, 0x55, 0x55, 0xFF}
			case gtTileSidewalk:
				col = color.RGBA{0x66, 0x66, 0x55, 0xFF}
			case gtTileBuilding, gtTileSpecial:
				col = color.RGBA{0x66, 0x55, 0x44, 0xFF}
			case gtTileWater:
				col = color.RGBA{0x22, 0x33, 0x77, 0xFF}
			default:
				continue
			}
			vector.DrawFilledRect(screen, mx, my, mw, mh, col, false)
		}
	}

	// Mission markers on minimap
	for i := range g.missions {
		m := &g.missions[i]
		if m.state == gtMissionNone {
			mx := mmX + float32(m.markerX*scaleX)
			my := mmY + float32(m.markerY*scaleY)
			vector.DrawFilledCircle(screen, mx, my, 2,
				color.RGBA{0xFF, 0xFF, 0x00, 0xFF}, false)
		}
	}

	// Active mission target on minimap
	if g.activeMission >= 0 && g.activeMission < len(g.missions) {
		m := &g.missions[g.activeMission]
		if m.state == gtMissionActive && m.targetX > 0 && m.targetY > 0 {
			mx := mmX + float32(m.targetX*scaleX)
			my := mmY + float32(m.targetY*scaleY)
			vector.DrawFilledCircle(screen, mx, my, 3,
				color.RGBA{0xFF, 0x44, 0x44, 0xFF}, false)
		}
	}

	// Player on minimap
	px := mmX + float32(g.px*scaleX)
	py := mmY + float32(g.py*scaleY)
	vector.DrawFilledCircle(screen, px, py, 2.5,
		color.RGBA{0x00, 0xFF, 0x00, 0xFF}, false)

	// Player direction indicator
	dirLen := float32(5)
	dx := float32(math.Cos(g.pAngle)) * dirLen
	dy := float32(math.Sin(g.pAngle)) * dirLen
	vector.StrokeLine(screen, px, py, px+dx, py+dy, 1.5,
		color.RGBA{0x00, 0xFF, 0x00, 0xFF}, false)

	// Cops on minimap
	for _, cop := range g.cops {
		if !cop.alive {
			continue
		}
		cx := mmX + float32(cop.x*scaleX)
		cy := mmY + float32(cop.y*scaleY)
		vector.DrawFilledCircle(screen, cx, cy, 1.5,
			color.RGBA{0x44, 0x44, 0xFF, 0xFF}, false)
	}
}

// --------------- interface ---------------

func (g *GTA) Layout(w, h int) (int, int) {
	return gtScreenW, gtScreenH
}

func (g *GTA) WantsBack() bool {
	return g.goBack
}
