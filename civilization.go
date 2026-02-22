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

// ── Screen constants ───────────────────────────────────────────────────────
const (
	cvScreenW = 640
	cvScreenH = 480
	cvMapW    = 40
	cvMapH    = 30
	cvTileS   = 16
	cvMinimapW = 80
	cvMinimapH = 60
	cvTopBarH  = 24
	cvBotBarH  = 48
	cvMaxCivs  = 3 // player + 2 AI
	cvFogRange = 2
)

// ── Terrain ────────────────────────────────────────────────────────────────
const (
	cvTerrGrass = iota
	cvTerrPlains
	cvTerrForest
	cvTerrHills
	cvTerrMountain
	cvTerrWater
	cvTerrDesert
	cvTerrTundra
)

var cvTerrNames = [8]string{"Grassland", "Plains", "Forest", "Hills", "Mountain", "Water", "Desert", "Tundra"}

var cvTerrColors = [8]color.RGBA{
	{0x60, 0xc0, 0x40, 0xff}, // grass
	{0xd0, 0xc0, 0x60, 0xff}, // plains
	{0x20, 0x80, 0x20, 0xff}, // forest
	{0x90, 0x70, 0x40, 0xff}, // hills
	{0x90, 0x90, 0x90, 0xff}, // mountain
	{0x30, 0x60, 0xc0, 0xff}, // water
	{0xd0, 0xc0, 0x80, 0xff}, // desert
	{0xd8, 0xd8, 0xe8, 0xff}, // tundra
}

// food, prod, science per terrain
var cvTerrYield = [8][3]int{
	{2, 0, 0}, // grass
	{1, 1, 0}, // plains
	{1, 2, 0}, // forest
	{1, 3, 0}, // hills
	{0, 0, 0}, // mountain
	{0, 0, 0}, // water
	{0, 1, 0}, // desert
	{1, 0, 0}, // tundra
}

// ── Tech tree ──────────────────────────────────────────────────────────────
const (
	cvTechAgriculture = iota
	cvTechMining
	cvTechWriting
	cvTechIronWorking
	cvTechHorseback
	cvTechEngineering
	cvTechMathematics
	cvTechNavigation
	cvTechCount
)

type cvTechDef struct {
	name string
	cost int
}

var cvTechs = [cvTechCount]cvTechDef{
	{"Agriculture", 20},
	{"Mining", 30},
	{"Writing", 40},
	{"Iron Working", 60},
	{"Horseback Riding", 60},
	{"Engineering", 80},
	{"Mathematics", 100},
	{"Navigation", 120},
}

// ── Unit types ─────────────────────────────────────────────────────────────
const (
	cvUnitSettler = iota
	cvUnitWarrior
	cvUnitArcher
	cvUnitSwordsman
	cvUnitHorseman
	cvUnitCatapult
)

type cvUnitDef struct {
	name     string
	cost     int
	strength int
	rangeAt  int
	moves    int
	reqTech  int // -1 = none
}

var cvUnitDefs = [6]cvUnitDef{
	{"Settler", 30, 0, 0, 2, -1},
	{"Warrior", 15, 5, 0, 2, -1},
	{"Archer", 20, 4, 2, 2, -1},
	{"Swordsman", 25, 7, 0, 2, cvTechIronWorking},
	{"Horseman", 25, 6, 0, 3, cvTechHorseback},
	{"Catapult", 35, 8, 3, 1, cvTechEngineering},
}

// ── Building types ─────────────────────────────────────────────────────────
const (
	cvBuildGranary = iota
	cvBuildBarracks
	cvBuildLibrary
	cvBuildWalls
	cvBuildWorkshop
	cvBuildCount
)

type cvBuildDef struct {
	name    string
	cost    int
	reqTech int
}

var cvBuildDefs = [cvBuildCount]cvBuildDef{
	{"Granary", 20, cvTechAgriculture},
	{"Barracks", 25, -1},
	{"Library", 30, cvTechWriting},
	{"Walls", 35, cvTechEngineering},
	{"Workshop", 40, cvTechMining},
}

// ── Civ colors ─────────────────────────────────────────────────────────────
var cvCivColors = [cvMaxCivs]color.RGBA{
	{0x40, 0x80, 0xff, 0xff}, // player - blue
	{0xff, 0x40, 0x40, 0xff}, // AI 1 - red
	{0xff, 0xd0, 0x20, 0xff}, // AI 2 - yellow
}

var cvCivNames = [cvMaxCivs]string{"Player", "Red Empire", "Gandhi's India"}

// Gandhi's aggression: starts at 1, democracy (Mathematics) subtracts 2 → underflows to 253
var cvCivBaseAggression = [cvMaxCivs]int{0, 5, 1} // Gandhi starts peaceful

// ── Fog states ─────────────────────────────────────────────────────────────
const (
	cvFogUnseen  = 0
	cvFogRevealed = 1
	cvFogVisible  = 2
)

// ── Data structures ────────────────────────────────────────────────────────

type cvTile struct {
	terrain int
}

type cvUnit struct {
	unitType    int
	civ         int
	x, y        int
	hp          int
	movesLeft   int
	dead        bool
	autoExplore bool
	// animation: interpolated movement
	animFromX, animFromY float64
	animProgress         float64 // 0 = at old pos, 1 = at new pos
}

type cvCity struct {
	name       string
	civ        int
	x, y       int
	population int
	food       int
	prodAccum  int
	buildings  [cvBuildCount]bool
	producing  int // -1=none, 0..5=unit, 100..104=building
	isCapital  bool
}

type cvCivData struct {
	alive        bool
	techs        [cvTechCount]bool
	researchTarget int
	scienceAccum int
	totalFood    int
	totalProd    int
	totalScience int
	fog          [cvMapW][cvMapH]int
}

// ── Notification ──────────────────────────────────────────────────────────

type cvNotification struct {
	text  string
	timer int
	col   color.RGBA
}

// ── Combat popup ──────────────────────────────────────────────────────────

type cvCombatPopup struct {
	text  string
	x, y  int
	timer int
	col   color.RGBA
}

// ── Main struct ────────────────────────────────────────────────────────────

type Civilization struct {
	goBack  bool
	tiles   [cvMapW][cvMapH]cvTile
	units   []*cvUnit
	cities  []*cvCity
	civs    [cvMaxCivs]cvCivData
	turn    int
	camX    int
	camY    int

	selectedUnit int // index or -1
	selectedCity int // index or -1
	showTechTree bool
	showBuild    bool
	moveHighlight [][2]int

	gameOver    bool
	winner      int
	victoryType string

	// audio
	bgmStarted  bool
	sfxCombat   []byte
	sfxCity     []byte
	sfxTech     []byte
	sfxTurn     []byte
	sfxMove     []byte
	audioReady  bool

	// era-based BGM
	bgmAncient    *audio.Player
	bgmMedieval   *audio.Player
	bgmIndustrial *audio.Player
	bgmNuclear    *audio.Player
	currentBGM    *audio.Player
	currentEra    int

	// Gandhi nuclear event
	gandhiNukeMsg     bool
	gandhiNukeMsgTick int

	// city naming
	cityCounter [cvMaxCivs]int

	// frame tick for animations
	tick int

	// UX: notifications
	notifications []cvNotification

	// UX: combat popups
	combatPopups []cvCombatPopup

	// UX: click feedback
	clickFeedbackX, clickFeedbackY int
	clickFeedbackTimer             int

	// UX: unmoved units warning
	endTurnWarning     bool
	endTurnWarningTick int

	// UX: tutorial overlay for first turn
	tutorialShown   bool
	tutorialDismiss bool

	// UX: event announcements (centered screen text)
	announcement     string
	announcementTick int
}

var cvCityNamePool = []string{
	"Rome", "Athens", "Sparta", "Thebes", "Memphis", "Babylon",
	"Carthage", "Troy", "Persepolis", "Alexandria", "Antioch",
	"Damascus", "Jerusalem", "Corinth", "Syracuse", "Tyre",
	"Nineveh", "Ur", "Susa", "Ecbatana", "Sardis", "Miletus",
	"Marathon", "Delphi", "Olympia", "Knossos", "Mycenae",
	"Sidon", "Byblos", "Gaza",
}

// ── Constructor ────────────────────────────────────────────────────────────

func newCivilization() *Civilization {
	g := &Civilization{
		selectedUnit: -1,
		selectedCity: -1,
		turn:         1,
		winner:       -1,
	}
	g.generateMap()
	g.initCivs()
	g.updateAllFog()
	g.initAudio()

	// Welcome notifications
	g.addNotification("Your civilization begins! Explore and expand.", color.RGBA{0xc0, 0xa0, 0x40, 0xff})
	g.addNotification("Use your Settler to found a second city.", color.RGBA{0x40, 0xff, 0x40, 0xff})
	g.addNotification("Use your Warrior to explore the map.", color.RGBA{0x40, 0xc0, 0xff, 0xff})
	return g
}

// ── Map generation ─────────────────────────────────────────────────────────

func (g *Civilization) generateMap() {
	// Step 1: random base
	for x := 0; x < cvMapW; x++ {
		for y := 0; y < cvMapH; y++ {
			r := rand.Intn(100)
			switch {
			case r < 25:
				g.tiles[x][y].terrain = cvTerrGrass
			case r < 45:
				g.tiles[x][y].terrain = cvTerrPlains
			case r < 60:
				g.tiles[x][y].terrain = cvTerrForest
			case r < 72:
				g.tiles[x][y].terrain = cvTerrHills
			case r < 80:
				g.tiles[x][y].terrain = cvTerrDesert
			case r < 88:
				g.tiles[x][y].terrain = cvTerrTundra
			case r < 94:
				g.tiles[x][y].terrain = cvTerrWater
			default:
				g.tiles[x][y].terrain = cvTerrMountain
			}
		}
	}

	// Step 2: smooth with neighbor averaging (3 passes)
	for pass := 0; pass < 3; pass++ {
		var tmp [cvMapW][cvMapH]int
		for x := 0; x < cvMapW; x++ {
			for y := 0; y < cvMapH; y++ {
				counts := [8]int{}
				counts[g.tiles[x][y].terrain]++
				for dx := -1; dx <= 1; dx++ {
					for dy := -1; dy <= 1; dy++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := x+dx, y+dy
						if nx >= 0 && nx < cvMapW && ny >= 0 && ny < cvMapH {
							counts[g.tiles[nx][ny].terrain]++
						}
					}
				}
				best := g.tiles[x][y].terrain
				bestCount := 0
				for t := 0; t < 8; t++ {
					if counts[t] > bestCount {
						bestCount = counts[t]
						best = t
					}
				}
				tmp[x][y] = best
			}
		}
		for x := 0; x < cvMapW; x++ {
			for y := 0; y < cvMapH; y++ {
				g.tiles[x][y].terrain = tmp[x][y]
			}
		}
	}

	// Step 3: place mountain ranges
	for i := 0; i < 3; i++ {
		sx := rand.Intn(cvMapW)
		sy := rand.Intn(cvMapH)
		length := 4 + rand.Intn(6)
		dx := rand.Intn(3) - 1
		dy := rand.Intn(3) - 1
		if dx == 0 && dy == 0 {
			dx = 1
		}
		for j := 0; j < length; j++ {
			px := sx + dx*j
			py := sy + dy*j
			if px >= 0 && px < cvMapW && py >= 0 && py < cvMapH {
				g.tiles[px][py].terrain = cvTerrMountain
			}
		}
	}

	// Step 4: place water bodies (lakes)
	for i := 0; i < 4; i++ {
		cx := 3 + rand.Intn(cvMapW-6)
		cy := 3 + rand.Intn(cvMapH-6)
		radius := 2 + rand.Intn(2)
		for dx := -radius; dx <= radius; dx++ {
			for dy := -radius; dy <= radius; dy++ {
				if dx*dx+dy*dy <= radius*radius {
					px := cx + dx
					py := cy + dy
					if px >= 0 && px < cvMapW && py >= 0 && py < cvMapH {
						g.tiles[px][py].terrain = cvTerrWater
					}
				}
			}
		}
	}

	// Ensure starting positions are habitable
	starts := [][2]int{{5, 5}, {cvMapW - 6, cvMapH - 6}, {cvMapW - 6, 5}}
	for _, s := range starts {
		for dx := -2; dx <= 2; dx++ {
			for dy := -2; dy <= 2; dy++ {
				px := s[0] + dx
				py := s[1] + dy
				if px >= 0 && px < cvMapW && py >= 0 && py < cvMapH {
					t := g.tiles[px][py].terrain
					if t == cvTerrWater || t == cvTerrMountain {
						g.tiles[px][py].terrain = cvTerrGrass
					}
				}
			}
		}
	}
}

// ── Civ initialization ────────────────────────────────────────────────────

func (g *Civilization) initCivs() {
	starts := [][2]int{{5, 5}, {cvMapW - 6, cvMapH - 6}, {cvMapW - 6, 5}}

	for i := 0; i < cvMaxCivs; i++ {
		g.civs[i].alive = true
		g.civs[i].researchTarget = -1
		g.civs[i].techs[cvTechAgriculture] = true

		sx, sy := starts[i][0], starts[i][1]

		// founding city
		cityName := g.nextCityName(i)
		city := &cvCity{
			name:       cityName,
			civ:        i,
			x:          sx,
			y:          sy,
			population: 1,
			producing:  -1,
			isCapital:  true,
		}
		g.cities = append(g.cities, city)

		// starting warrior
		wx, wy := sx+1, sy
		if wx >= cvMapW {
			wx = sx - 1
		}
		warrior := &cvUnit{
			unitType:     cvUnitWarrior,
			civ:          i,
			x:            wx,
			y:            wy,
			hp:           10,
			movesLeft:    cvUnitDefs[cvUnitWarrior].moves,
			animProgress: 1.0,
		}
		g.units = append(g.units, warrior)

		// starting settler for expansion (everyone gets one)
		setX, setY := sx-1, sy+1
		if setX < 0 {
			setX = sx + 2
		}
		if setY >= cvMapH {
			setY = sy - 1
		}
		// Make sure settler doesn't land on impassable terrain
		if g.tiles[setX][setY].terrain == cvTerrWater || g.tiles[setX][setY].terrain == cvTerrMountain {
			setX = sx
			setY = sy + 1
			if setY >= cvMapH {
				setY = sy - 1
			}
		}
		settler := &cvUnit{
			unitType:     cvUnitSettler,
			civ:          i,
			x:            setX,
			y:            setY,
			hp:           10,
			movesLeft:    cvUnitDefs[cvUnitSettler].moves,
			animProgress: 1.0,
		}
		g.units = append(g.units, settler)

		// Pick first research
		g.cvPickNextResearch(i)
	}
}

func (g *Civilization) nextCityName(civ int) string {
	idx := g.cityCounter[civ]
	g.cityCounter[civ]++
	base := civ * 10
	nameIdx := base + idx
	if nameIdx >= len(cvCityNamePool) {
		nameIdx = idx % len(cvCityNamePool)
	}
	return cvCityNamePool[nameIdx]
}

// ── Audio init ─────────────────────────────────────────────────────────────

func (g *Civilization) initAudio() {
	// ── Era-based BGM ──

	// ANCIENT (~70bpm): Slow, tribal. Deep sine bass drone, pentatonic triangle melody
	// A minor pentatonic: A3=57, C4=60, D4=62, E4=64, G4=67
	{
		bpm := 70.0
		melody := [][2]int{
			{57, 4}, {60, 2}, {62, 2}, {64, 4}, {62, 2}, {60, 2},
			{57, 4}, {0, 2}, {67, 2}, {64, 4}, {62, 4},
			{60, 2}, {62, 2}, {64, 2}, {67, 2}, {64, 4}, {60, 4},
			{57, 4}, {60, 2}, {57, 2}, {0, 4}, {62, 2}, {64, 2},
			{67, 4}, {64, 2}, {62, 2}, {60, 4}, {57, 4},
			{0, 2}, {60, 2}, {62, 4}, {57, 4}, {0, 4},
		}
		pad := [][2]int{
			{45, 8}, {45, 8}, {48, 8}, {45, 8},
			{48, 8}, {45, 8}, {43, 8}, {45, 8},
			{45, 8}, {48, 8}, {45, 8}, {43, 8},
		}
		drums := "K...h...K...h...K...h.h.K...h..." +
			"K...h...K...h...K.h.h...K...h..."  +
			"K...h...K...h...K...h.h.K...h..."
		melPCM := generateTrack(melody, triangleWave, bpm, 0.20)
		padPCM := generateTrack(pad, sineWave, bpm, 0.12)
		drumPCM := generateDrumPattern(drums, bpm, 0.18)
		mix := mixBuffers(melPCM, padPCM)
		mix = mixBuffers(mix, drumPCM)
		g.bgmAncient = loopPlayer(mix, 0.3)
	}

	// MEDIEVAL (~85bpm): Regal fanfare. Square wave melody in C major. Steady bass march.
	{
		bpm := 85.0
		melody := [][2]int{
			{60, 2}, {64, 2}, {67, 2}, {72, 2}, {71, 2}, {72, 4}, {0, 2},
			{72, 2}, {74, 2}, {76, 2}, {72, 2}, {71, 2}, {67, 4}, {0, 2},
			{60, 2}, {64, 2}, {67, 2}, {71, 2}, {72, 4}, {74, 2}, {72, 2},
			{71, 2}, {67, 2}, {64, 2}, {60, 2}, {60, 4}, {0, 4},
			{67, 2}, {69, 2}, {71, 2}, {72, 4}, {74, 2}, {72, 2}, {71, 2},
			{67, 2}, {64, 2}, {67, 4}, {60, 4}, {0, 4},
		}
		bass := [][2]int{
			{48, 4}, {48, 4}, {43, 4}, {43, 4},
			{48, 4}, {45, 4}, {43, 4}, {48, 4},
			{48, 4}, {43, 4}, {45, 4}, {48, 4},
			{43, 4}, {45, 4}, {47, 4}, {48, 4},
		}
		drums := "K..hS..hK..hS..hK..hS..hK..hS..h" +
			"K..hS..hK..hS..hK..hS..hK..hS..h" +
			"K..hS..hK..hS..hK..hS..hK..hS..h"
		melPCM := generateTrack(melody, squareWave, bpm, 0.18)
		bassPCM := generateTrack(bass, sineWave, bpm, 0.15)
		drumPCM := generateDrumPattern(drums, bpm, 0.22)
		mix := mixBuffers(melPCM, bassPCM)
		mix = mixBuffers(mix, drumPCM)
		g.bgmMedieval = loopPlayer(mix, 0.3)
	}

	// INDUSTRIAL (~100bpm): Mechanical, tense. Saw wave lead in D minor. Complex drums.
	// D minor: D4=62, E4=64, F4=65, G4=67, A4=69, Bb4=70, C5=72
	{
		bpm := 100.0
		melody := [][2]int{
			{62, 2}, {65, 2}, {69, 2}, {67, 2}, {65, 2}, {62, 2}, {0, 2}, {70, 2},
			{69, 2}, {67, 2}, {65, 2}, {62, 2}, {64, 4}, {62, 2}, {0, 2},
			{62, 1}, {65, 1}, {69, 2}, {72, 2}, {70, 2}, {69, 2}, {67, 2}, {65, 2}, {62, 2},
			{65, 2}, {67, 2}, {69, 2}, {70, 4}, {69, 2}, {67, 2},
			{62, 2}, {0, 2}, {65, 2}, {67, 2}, {69, 4}, {0, 4},
		}
		bass := [][2]int{
			{38, 2}, {38, 2}, {38, 2}, {38, 2}, {41, 2}, {41, 2}, {41, 2}, {41, 2},
			{43, 2}, {43, 2}, {43, 2}, {43, 2}, {38, 2}, {38, 2}, {38, 2}, {38, 2},
			{36, 2}, {36, 2}, {38, 2}, {38, 2}, {41, 2}, {41, 2}, {43, 2}, {43, 2},
		}
		drums := "K.hSK.hSK.hSKhhS" +
			"K.hSK.hSKhhSK.hS" +
			"K.hSKhhSK.hSK.hS" +
			"KhhSK.hSK.hSKhhS"
		melPCM := generateTrack(melody, sawWave, bpm, 0.16)
		bassPCM := generateTrack(bass, squareWave, bpm, 0.14)
		drumPCM := generateDrumPattern(drums, bpm, 0.25)
		mix := mixBuffers(melPCM, bassPCM)
		mix = mixBuffers(mix, drumPCM)
		g.bgmIndustrial = loopPlayer(mix, 0.3)
	}

	// NUCLEAR/GANDHI (~120bpm): AGGRESSIVE. Heavy power chords, fast drums, dissonance.
	// E2=40, B2=47 (power fifth), plus dissonant Bb2=46, F2=41
	{
		bpm := 120.0
		melody := [][2]int{
			{40, 2}, {47, 2}, {40, 2}, {47, 2}, {41, 2}, {46, 2}, {40, 2}, {47, 2},
			{40, 1}, {47, 1}, {40, 1}, {47, 1}, {41, 2}, {46, 2}, {40, 2}, {0, 2},
			{40, 2}, {47, 2}, {46, 2}, {41, 2}, {40, 2}, {47, 2}, {40, 2}, {47, 2},
			{41, 1}, {46, 1}, {41, 1}, {46, 1}, {40, 2}, {47, 2}, {40, 4},
			{40, 2}, {47, 2}, {40, 2}, {41, 2}, {46, 2}, {47, 2}, {40, 2}, {0, 2},
			{40, 1}, {41, 1}, {46, 1}, {47, 1}, {40, 2}, {47, 2}, {40, 4},
		}
		bass := [][2]int{
			{28, 2}, {28, 2}, {28, 2}, {28, 2}, {29, 2}, {29, 2}, {28, 2}, {28, 2},
			{28, 1}, {28, 1}, {28, 1}, {28, 1}, {29, 2}, {29, 2}, {28, 2}, {28, 2},
			{28, 2}, {28, 2}, {29, 2}, {29, 2}, {28, 2}, {28, 2}, {28, 2}, {28, 2},
			{29, 1}, {29, 1}, {29, 1}, {29, 1}, {28, 2}, {28, 2}, {28, 4},
		}
		drums := "KShSKShSKShSKShS" +
			"KSKSKShSKShSKSKS" +
			"KShSKSKSKShSKShS" +
			"KSKSKSKSKShSKSKS" +
			"KShSKShSKSKSKShS" +
			"KSKSKShSKSKSKSKS"
		melPCM := generateTrack(melody, squareWave, bpm, 0.22)
		bassPCM := generateTrack(bass, squareWave, bpm, 0.18)
		drumPCM := generateDrumPattern(drums, bpm, 0.30)
		mix := mixBuffers(melPCM, bassPCM)
		mix = mixBuffers(mix, drumPCM)
		g.bgmNuclear = loopPlayer(mix, 0.35)
	}

	// Start with Ancient era
	g.currentEra = 0
	g.currentBGM = g.bgmAncient
	g.bgmAncient.Play()

	// SFX
	g.sfxCombat = g.genSFXCombat()
	g.sfxCity = g.genSFXCity()
	g.sfxTech = g.genSFXTech()
	g.sfxTurn = g.genSFXTurn()
	g.sfxMove = g.genSFXMove()
	g.audioReady = true
}

// cvCountResearchedTechs counts how many techs a civ has researched
func (g *Civilization) cvCountResearchedTechs(civ int) int {
	count := 0
	for i := 0; i < cvTechCount; i++ {
		if g.civs[civ].techs[i] {
			count++
		}
	}
	return count
}

// cvDetermineEra returns the era index based on tech count
func (g *Civilization) cvDetermineEra() int {
	techCount := g.cvCountResearchedTechs(0)

	// Gandhi nuclear override: if Gandhi (civ 2) has Mathematics AND aggression > 200
	if g.civs[2].alive && g.civs[2].techs[cvTechMathematics] && g.cvAggression(2) > 200 {
		return 3 // force nuclear era
	}

	switch {
	case techCount <= 2:
		return 0 // Ancient
	case techCount <= 4:
		return 1 // Medieval
	case techCount <= 6:
		return 2 // Industrial
	default:
		return 3 // Nuclear
	}
}

// cvSwitchBGM switches background music when era changes
func (g *Civilization) cvSwitchBGM() {
	era := g.cvDetermineEra()
	if era == g.currentEra && g.currentBGM != nil {
		return
	}

	// Pause current BGM
	if g.currentBGM != nil {
		g.currentBGM.Pause()
	}

	g.currentEra = era
	switch era {
	case 0:
		g.currentBGM = g.bgmAncient
	case 1:
		g.currentBGM = g.bgmMedieval
	case 2:
		g.currentBGM = g.bgmIndustrial
	case 3:
		g.currentBGM = g.bgmNuclear
	}

	if g.currentBGM != nil {
		g.currentBGM.Play()
	}

	// Gandhi nuclear message (one-time)
	if era == 3 && g.civs[2].alive && g.civs[2].techs[cvTechMathematics] && g.cvAggression(2) > 200 && !g.gandhiNukeMsg {
		g.gandhiNukeMsg = true
		g.gandhiNukeMsgTick = g.tick
		g.addNotification("Gandhi has gone NUCLEAR!", color.RGBA{0xff, 0x20, 0x20, 0xff})
	}
}

func (g *Civilization) genSFXCombat() []byte {
	// Proper clash with metallic overtones
	buf := pcmStereo(0.45)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(samples)

		// Impact hit (noise burst)
		impact := noise() * 0.6 * math.Exp(-t*12)
		// Metallic ring (inharmonic partials)
		metal := sineWave(t, 1200)*0.15*math.Exp(-t*8) +
			sineWave(t, 2340)*0.10*math.Exp(-t*10) +
			sineWave(t, 3750)*0.08*math.Exp(-t*14)
		// Low thud
		thud := sineWave(t, 80*math.Exp(-t*6)) * 0.3 * math.Exp(-t*5)
		// Scrape (descending noise-based)
		scrape := noise() * 0.12 * math.Exp(-t*6) * sineWave(t, 600-t*800)

		val := (impact + metal + thud + scrape) * (1 - tFrac*0.3) * 0.5
		writeSample(buf, i*4, val)
	}
	return buf
}

func (g *Civilization) genSFXCity() []byte {
	// Triumphant 4-note fanfare: C5, E5, G5, C6
	fanfareNotes := []float64{523.25, 659.25, 783.99, 1046.50}
	noteDur := 0.2
	totalDur := noteDur * float64(len(fanfareNotes))
	buf := pcmStereo(totalDur + 0.3) // extra tail for reverb
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		val := 0.0
		for ni, freq := range fanfareNotes {
			noteStart := float64(ni) * noteDur
			if t >= noteStart {
				nt := t - noteStart
				ntFrac := nt / (totalDur + 0.3 - noteStart)
				if ntFrac > 1 {
					ntFrac = 1
				}
				env := envelope(ntFrac, 0.03, 0.4, 0.57)
				val += (triangleWave(t, freq)*0.25 + sineWave(t, freq)*0.15 + sineWave(t, freq*2)*0.05) * env
			}
		}
		val *= 0.45
		writeSample(buf, i*4, val)
	}
	return buf
}

func (g *Civilization) genSFXTech() []byte {
	// Ascending chime sequence: E5, G5, B5, E6
	chimeNotes := []float64{659.25, 783.99, 987.77, 1318.51}
	noteDur := 0.12
	totalDur := noteDur*float64(len(chimeNotes)) + 0.3
	buf := pcmStereo(totalDur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		val := 0.0
		for ni, freq := range chimeNotes {
			noteStart := float64(ni) * noteDur
			if t >= noteStart {
				nt := t - noteStart
				env := math.Exp(-nt * 4)
				val += (sineWave(t, freq)*0.3 + sineWave(t, freq*2.01)*0.08) * env
			}
		}
		val *= 0.4
		writeSample(buf, i*4, val)
	}
	return buf
}

func (g *Civilization) genSFXTurn() []byte {
	buf := pcmStereo(0.1)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(samples)
		val := squareWave(t, 600) * (1 - tFrac) * 0.15
		writeSample(buf, i*4, val)
	}
	return buf
}

func (g *Civilization) genSFXMove() []byte {
	buf := pcmStereo(0.08)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(samples)
		val := noise() * (1 - tFrac) * 0.1
		_ = t
		writeSample(buf, i*4, val)
	}
	return buf
}

// ── Fog of war ─────────────────────────────────────────────────────────────

func (g *Civilization) updateAllFog() {
	for c := 0; c < cvMaxCivs; c++ {
		g.updateFogForCiv(c)
	}
}

func (g *Civilization) updateFogForCiv(civ int) {
	// Demote all visible to revealed
	for x := 0; x < cvMapW; x++ {
		for y := 0; y < cvMapH; y++ {
			if g.civs[civ].fog[x][y] == cvFogVisible {
				g.civs[civ].fog[x][y] = cvFogRevealed
			}
		}
	}

	// Mark tiles near units/cities as visible
	for _, u := range g.units {
		if u.dead || u.civ != civ {
			continue
		}
		g.revealAround(civ, u.x, u.y, cvFogRange)
	}
	for _, c := range g.cities {
		if c.civ != civ {
			continue
		}
		g.revealAround(civ, c.x, c.y, cvFogRange)
	}
}

func (g *Civilization) revealAround(civ, cx, cy, r int) {
	for dx := -r; dx <= r; dx++ {
		for dy := -r; dy <= r; dy++ {
			nx := cx + dx
			ny := cy + dy
			if nx >= 0 && nx < cvMapW && ny >= 0 && ny < cvMapH {
				g.civs[civ].fog[nx][ny] = cvFogVisible
			}
		}
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func cvIsPassable(terrain int, hasNav bool) bool {
	if terrain == cvTerrMountain {
		return false
	}
	if terrain == cvTerrWater && !hasNav {
		return false
	}
	return true
}

func (g *Civilization) unitAt(x, y int) *cvUnit {
	for _, u := range g.units {
		if !u.dead && u.x == x && u.y == y {
			return u
		}
	}
	return nil
}

func (g *Civilization) unitIndexAt(x, y int) int {
	for i, u := range g.units {
		if !u.dead && u.x == x && u.y == y {
			return i
		}
	}
	return -1
}

func (g *Civilization) cityAt(x, y int) *cvCity {
	for _, c := range g.cities {
		if c.x == x && c.y == y {
			return c
		}
	}
	return nil
}

func (g *Civilization) cityIndexAt(x, y int) int {
	for i, c := range g.cities {
		if c.x == x && c.y == y {
			return i
		}
	}
	return -1
}

func (g *Civilization) countCities(civ int) int {
	n := 0
	for _, c := range g.cities {
		if c.civ == civ {
			n++
		}
	}
	return n
}

func (g *Civilization) countUnits(civ int) int {
	n := 0
	for _, u := range g.units {
		if !u.dead && u.civ == civ {
			n++
		}
	}
	return n
}

func (g *Civilization) getCivTotals(civ int) (food, prod, sci int) {
	for _, c := range g.cities {
		if c.civ != civ {
			continue
		}
		f, p, s := g.cityYield(c)
		food += f
		prod += p
		sci += s
	}
	return
}

func (g *Civilization) cityYield(c *cvCity) (food, prod, sci int) {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			nx := c.x + dx
			ny := c.y + dy
			if nx >= 0 && nx < cvMapW && ny >= 0 && ny < cvMapH {
				t := g.tiles[nx][ny].terrain
				food += cvTerrYield[t][0]
				prod += cvTerrYield[t][1]
				sci += cvTerrYield[t][2]

				// Mining bonus
				if g.civs[c.civ].techs[cvTechMining] && t == cvTerrHills {
					prod++
				}
				// Mountain adjacency science
				if t == cvTerrMountain && (dx != 0 || dy != 0) {
					sci++
				}
			}
		}
	}

	// Population multiplier
	food += c.population
	prod += c.population / 2
	sci += c.population / 3

	// Building bonuses
	if c.buildings[cvBuildGranary] {
		food += 2
	}
	if c.buildings[cvBuildLibrary] {
		sci += 3
	}
	if c.buildings[cvBuildWorkshop] {
		prod += 3
	}

	// Mathematics bonus
	if g.civs[c.civ].techs[cvTechMathematics] {
		sci = sci * 3 / 2
	}

	// Ensure minimum 1 production
	if prod < 1 {
		prod = 1
	}

	return
}

func (g *Civilization) cityDefense(c *cvCity) int {
	def := 10
	if c.buildings[cvBuildWalls] {
		def += 5
	}
	if g.tiles[c.x][c.y].terrain == cvTerrHills {
		def += 3
	}
	return def
}

// ── Notification system ────────────────────────────────────────────────────

func (g *Civilization) addNotification(text string, col color.RGBA) {
	g.notifications = append(g.notifications, cvNotification{
		text:  text,
		timer: 180, // ~3 seconds at 60fps
		col:   col,
	})
	// Keep only last 6
	if len(g.notifications) > 6 {
		g.notifications = g.notifications[len(g.notifications)-6:]
	}
}

func (g *Civilization) addCombatPopup(text string, x, y int, col color.RGBA) {
	g.combatPopups = append(g.combatPopups, cvCombatPopup{
		text:  text,
		x:     x,
		y:     y,
		timer: 90, // ~1.5 seconds
		col:   col,
	})
}

func (g *Civilization) showAnnouncement(text string) {
	g.announcement = text
	g.announcementTick = g.tick
}

func (g *Civilization) updateNotifications() {
	var alive []cvNotification
	for _, n := range g.notifications {
		n.timer--
		if n.timer > 0 {
			alive = append(alive, n)
		}
	}
	g.notifications = alive

	var alivePopups []cvCombatPopup
	for _, p := range g.combatPopups {
		p.timer--
		if p.timer > 0 {
			alivePopups = append(alivePopups, p)
		}
	}
	g.combatPopups = alivePopups
}

// ── Auto-explore ──────────────────────────────────────────────────────────

func (g *Civilization) autoExploreUnit(u *cvUnit) {
	if u.dead || u.movesLeft <= 0 || u.civ != 0 {
		return
	}

	// Check if enemy is visible nearby - stop auto-explore
	for _, eu := range g.units {
		if eu.dead || eu.civ == 0 {
			continue
		}
		if g.civs[0].fog[eu.x][eu.y] == cvFogVisible {
			dist := abs(u.x-eu.x) + abs(u.y-eu.y)
			if dist <= 5 {
				u.autoExplore = false
				g.addNotification(fmt.Sprintf("%s spotted enemy! Auto-explore stopped.", cvUnitDefs[u.unitType].name), color.RGBA{0xff, 0x80, 0x40, 0xff})
				return
			}
		}
	}

	// Find nearest unseen tile
	bestDist := 999
	bestX, bestY := -1, -1
	for x := 0; x < cvMapW; x += 2 {
		for y := 0; y < cvMapH; y += 2 {
			if g.civs[0].fog[x][y] == cvFogUnseen {
				dist := abs(u.x-x) + abs(u.y-y)
				if dist < bestDist {
					bestDist = dist
					bestX = x
					bestY = y
				}
			}
		}
	}
	if bestX >= 0 {
		g.aiMoveToward(u, bestX, bestY)
		g.updateFogForCiv(0)
	} else {
		u.autoExplore = false
		g.addNotification("Map fully explored!", color.RGBA{0x40, 0xff, 0x40, 0xff})
	}
}

// ── Next unmoved unit ─────────────────────────────────────────────────────

func (g *Civilization) selectNextUnmovedUnit() {
	startIdx := 0
	if g.selectedUnit >= 0 {
		startIdx = g.selectedUnit + 1
	}
	n := len(g.units)
	for i := 0; i < n; i++ {
		idx := (startIdx + i) % n
		u := g.units[idx]
		if !u.dead && u.civ == 0 && u.movesLeft > 0 && !u.autoExplore {
			g.selectedUnit = idx
			g.selectedCity = -1
			g.showBuild = false
			g.moveHighlight = g.computeValidMoves(u)
			// Center camera on unit
			g.camX = u.x*cvTileS - cvScreenW/2
			g.camY = u.y*cvTileS - cvScreenH/2
			return
		}
	}
	// No unmoved units found
	g.selectedUnit = -1
	g.moveHighlight = nil
}

func (g *Civilization) countUnmovedPlayerUnits() int {
	n := 0
	for _, u := range g.units {
		if !u.dead && u.civ == 0 && u.movesLeft > 0 && !u.autoExplore {
			n++
		}
	}
	return n
}

// ── Unit animations ───────────────────────────────────────────────────────

func (g *Civilization) updateAnimations() {
	for _, u := range g.units {
		if u.animProgress < 1.0 {
			u.animProgress += 0.15 // smooth movement over ~7 frames
			if u.animProgress > 1.0 {
				u.animProgress = 1.0
			}
		}
	}
}

func (g *Civilization) startMoveAnim(u *cvUnit, fromX, fromY int) {
	u.animFromX = float64(fromX)
	u.animFromY = float64(fromY)
	u.animProgress = 0.0
}

// ── Movement computation ───────────────────────────────────────────────────

func (g *Civilization) computeValidMoves(u *cvUnit) [][2]int {
	if u.dead || u.movesLeft <= 0 {
		return nil
	}
	hasNav := g.civs[u.civ].techs[cvTechNavigation]
	moves := u.movesLeft
	visited := map[[2]int]bool{{u.x, u.y}: true}
	frontier := [][2]int{{u.x, u.y}}
	var result [][2]int

	for step := 0; step < moves; step++ {
		var next [][2]int
		for _, pos := range frontier {
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					// Only cardinal + diagonal
					nx := pos[0] + dx
					ny := pos[1] + dy
					if nx < 0 || nx >= cvMapW || ny < 0 || ny >= cvMapH {
						continue
					}
					key := [2]int{nx, ny}
					if visited[key] {
						continue
					}
					if !cvIsPassable(g.tiles[nx][ny].terrain, hasNav) {
						continue
					}
					// Can't move through enemy units
					eu := g.unitAt(nx, ny)
					if eu != nil && eu.civ != u.civ {
						// Can target for attack but not pass through
						visited[key] = true
						result = append(result, key)
						continue
					}
					// Can't stack own units
					if eu != nil && eu.civ == u.civ {
						continue
					}
					visited[key] = true
					next = append(next, key)
					result = append(result, key)
				}
			}
		}
		frontier = next
	}
	return result
}

func (g *Civilization) canMoveTo(moves [][2]int, x, y int) bool {
	for _, m := range moves {
		if m[0] == x && m[1] == y {
			return true
		}
	}
	return false
}

// ── Combat ─────────────────────────────────────────────────────────────────

func (g *Civilization) doCombat(attacker, defender *cvUnit) {
	atkStr := float64(cvUnitDefs[attacker.unitType].strength)
	defStr := float64(cvUnitDefs[defender.unitType].strength)

	// Barracks bonus
	for _, c := range g.cities {
		if c.civ == attacker.civ && c.buildings[cvBuildBarracks] {
			atkStr += 1
			break
		}
	}
	for _, c := range g.cities {
		if c.civ == defender.civ && c.buildings[cvBuildBarracks] {
			defStr += 1
			break
		}
	}

	// Hills defense bonus
	if g.tiles[defender.x][defender.y].terrain == cvTerrHills {
		defStr += 2
	}

	// Attack damage
	atkDmg := atkStr*(0.5+rand.Float64()*0.5) - defStr*0.3
	if atkDmg < 1 {
		atkDmg = 1
	}

	// Defender retaliation
	defDmg := defStr*0.5*(0.5+rand.Float64()*0.5) - atkStr*0.15
	if defDmg < 0 {
		defDmg = 0
	}

	defender.hp -= int(atkDmg)
	attacker.hp -= int(defDmg)

	// Combat feedback popups
	atkName := cvUnitDefs[attacker.unitType].name
	defName := cvUnitDefs[defender.unitType].name
	g.addCombatPopup(fmt.Sprintf("%s deals %d dmg!", atkName, int(atkDmg)), defender.x, defender.y, color.RGBA{0xff, 0x60, 0x20, 0xff})
	if int(defDmg) > 0 {
		g.addCombatPopup(fmt.Sprintf("%s retaliates %d!", defName, int(defDmg)), attacker.x, attacker.y, color.RGBA{0xff, 0x40, 0x40, 0xff})
	}

	if defender.hp <= 0 {
		defender.dead = true
		if attacker.civ == 0 {
			g.addNotification(fmt.Sprintf("Your %s killed enemy %s!", atkName, defName), color.RGBA{0x40, 0xff, 0x40, 0xff})
		} else if defender.civ == 0 {
			g.addNotification(fmt.Sprintf("Your %s was killed by %s!", defName, atkName), color.RGBA{0xff, 0x40, 0x40, 0xff})
		}
	}
	if attacker.hp <= 0 {
		attacker.dead = true
		if attacker.civ == 0 {
			g.addNotification(fmt.Sprintf("Your %s was killed attacking %s!", atkName, defName), color.RGBA{0xff, 0x40, 0x40, 0xff})
		}
	}

	// If defender died and attacker alive, attacker moves to defender pos
	if defender.dead && !attacker.dead {
		attacker.x = defender.x
		attacker.y = defender.y
	}

	attacker.movesLeft = 0

	if g.audioReady {
		playSFX(g.sfxCombat, 0.4)
	}
}

func (g *Civilization) attackCity(attacker *cvUnit, city *cvCity) {
	atkStr := float64(cvUnitDefs[attacker.unitType].strength)
	defStr := float64(g.cityDefense(city))

	dmg := atkStr*(0.5+rand.Float64()*0.5) - defStr*0.2
	if dmg < 1 {
		dmg = 1
	}

	retDmg := defStr * 0.3 * (0.5 + rand.Float64()*0.5)
	attacker.hp -= int(retDmg)
	if attacker.hp <= 0 {
		attacker.dead = true
	}

	city.population -= int(dmg) / 3
	if city.population <= 0 {
		// City captured
		oldCiv := city.civ
		city.civ = attacker.civ
		city.population = 1
		city.producing = -1
		city.prodAccum = 0
		if !attacker.dead {
			attacker.x = city.x
			attacker.y = city.y
		}
		if attacker.civ == 0 {
			g.addNotification(fmt.Sprintf("You captured %s from %s!", city.name, cvCivNames[oldCiv]), color.RGBA{0xff, 0xd7, 0x00, 0xff})
			g.showAnnouncement(fmt.Sprintf("%s has been captured!", city.name))
		} else if oldCiv == 0 {
			g.addNotification(fmt.Sprintf("%s captured your city %s!", cvCivNames[attacker.civ], city.name), color.RGBA{0xff, 0x20, 0x20, 0xff})
			g.showAnnouncement(fmt.Sprintf("%s has fallen!", city.name))
		}
	}

	attacker.movesLeft = 0

	if g.audioReady {
		playSFX(g.sfxCombat, 0.4)
	}
}

// ── Research ───────────────────────────────────────────────────────────────

func (g *Civilization) cvPickNextResearch(civ int) {
	best := -1
	bestCost := 999999
	for i := 0; i < cvTechCount; i++ {
		if !g.civs[civ].techs[i] && cvTechs[i].cost < bestCost {
			bestCost = cvTechs[i].cost
			best = i
		}
	}
	g.civs[civ].researchTarget = best
}

func (g *Civilization) allTechsResearched(civ int) bool {
	for i := 0; i < cvTechCount; i++ {
		if !g.civs[civ].techs[i] {
			return false
		}
	}
	return true
}

// ── Turn processing ────────────────────────────────────────────────────────

func (g *Civilization) processTurnForCiv(civ int) {
	if !g.civs[civ].alive {
		return
	}

	// Reset unit moves
	for _, u := range g.units {
		if !u.dead && u.civ == civ {
			u.movesLeft = cvUnitDefs[u.unitType].moves
		}
	}

	// City processing
	for _, c := range g.cities {
		if c.civ != civ {
			continue
		}
		food, prod, sci := g.cityYield(c)

		// Food accumulation and growth
		c.food += food
		growthThreshold := 10 + c.population*5
		if c.food >= growthThreshold {
			c.food -= growthThreshold
			c.population++
			if civ == 0 {
				g.addNotification(fmt.Sprintf("%s grows to pop %d!", c.name, c.population), color.RGBA{0x40, 0xc0, 0x40, 0xff})
				g.addCombatPopup("+1 POP", c.x, c.y, color.RGBA{0x40, 0xff, 0x40, 0xff})
			}
		}

		// Production
		if c.producing >= 0 {
			c.prodAccum += prod
			if c.producing < 100 {
				// Producing a unit
				ut := c.producing
				if ut >= 0 && ut < 6 {
					if c.prodAccum >= cvUnitDefs[ut].cost {
						c.prodAccum = 0
						completedName := cvUnitDefs[ut].name
						c.producing = -1
						// Spawn unit adjacent to city
						sx, sy := g.findSpawnSpot(c)
						newUnit := &cvUnit{
							unitType:     ut,
							civ:          civ,
							x:            sx,
							y:            sy,
							hp:           10,
							movesLeft:    0,
							animProgress: 1.0,
						}
						g.units = append(g.units, newUnit)
						if civ == 0 {
							g.addNotification(fmt.Sprintf("%s completed in %s!", completedName, c.name), color.RGBA{0x40, 0xc0, 0xff, 0xff})
							g.showAnnouncement(fmt.Sprintf("%s completed!", completedName))
							if g.audioReady {
								playSFX(g.sfxCity, 0.3)
							}
						}
					}
				}
			} else {
				// Producing a building
				bi := c.producing - 100
				if bi >= 0 && bi < cvBuildCount {
					if c.prodAccum >= cvBuildDefs[bi].cost {
						c.prodAccum = 0
						completedName := cvBuildDefs[bi].name
						c.producing = -1
						c.buildings[bi] = true
						if civ == 0 {
							g.addNotification(fmt.Sprintf("%s built in %s!", completedName, c.name), color.RGBA{0x40, 0xff, 0xc0, 0xff})
							g.showAnnouncement(fmt.Sprintf("%s built!", completedName))
							if g.audioReady {
								playSFX(g.sfxCity, 0.3)
							}
						}
					}
				}
			}
		}

		// Science
		g.civs[civ].scienceAccum += sci

		g.civs[civ].totalFood = food
		g.civs[civ].totalProd = prod
		g.civs[civ].totalScience = sci
	}

	// Tech research
	rt := g.civs[civ].researchTarget
	if rt >= 0 && rt < cvTechCount {
		if g.civs[civ].scienceAccum >= cvTechs[rt].cost {
			g.civs[civ].scienceAccum -= cvTechs[rt].cost
			g.civs[civ].techs[rt] = true
			if civ == 0 {
				g.addNotification(fmt.Sprintf("Researched: %s!", cvTechs[rt].name), color.RGBA{0x60, 0xc0, 0xff, 0xff})
				g.showAnnouncement(fmt.Sprintf("Technology Discovered: %s", cvTechs[rt].name))
				if g.audioReady {
					playSFX(g.sfxTech, 0.4)
				}
			} else {
				g.addNotification(fmt.Sprintf("%s researched %s", cvCivNames[civ], cvTechs[rt].name), color.RGBA{0x80, 0x80, 0x80, 0xff})
			}
			g.cvPickNextResearch(civ)
		}
	} else {
		g.cvPickNextResearch(civ)
	}

	// Heal units in friendly territory
	for _, u := range g.units {
		if u.dead || u.civ != civ {
			continue
		}
		if u.hp < 10 {
			// Check if in friendly territory (near own city)
			for _, c := range g.cities {
				if c.civ == civ && abs(u.x-c.x) <= 2 && abs(u.y-c.y) <= 2 {
					u.hp++
					break
				}
			}
		}
	}

	// Update fog
	g.updateFogForCiv(civ)
}

func (g *Civilization) findSpawnSpot(c *cvCity) (int, int) {
	hasNav := g.civs[c.civ].techs[cvTechNavigation]
	// Try adjacent tiles
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx := c.x + dx
			ny := c.y + dy
			if nx >= 0 && nx < cvMapW && ny >= 0 && ny < cvMapH {
				if cvIsPassable(g.tiles[nx][ny].terrain, hasNav) && g.unitAt(nx, ny) == nil {
					return nx, ny
				}
			}
		}
	}
	return c.x, c.y // fallback
}

// ── AI ─────────────────────────────────────────────────────────────────────

// cvAggression returns effective aggression for a civ.
// Gandhi (civ 2): base aggression 1. Mathematics subtracts 2 → underflows to 253.
func (g *Civilization) cvAggression(civ int) int {
	aggro := cvCivBaseAggression[civ]
	// The classic bug: "democracy" (Mathematics) reduces aggression by 2
	if civ == 2 && g.civs[civ].techs[cvTechMathematics] {
		aggro -= 2
		if aggro < 0 {
			aggro = 253 // unsigned underflow! Gandhi goes nuclear
		}
	}
	return aggro
}

func (g *Civilization) runAI(civ int) {
	if !g.civs[civ].alive {
		return
	}

	// Process turn (resource accumulation)
	g.processTurnForCiv(civ)

	aggro := g.cvAggression(civ)
	gandhiNuke := civ == 2 && aggro > 200 // Gandhi has gone nuclear

	// AI city production decisions
	for _, c := range g.cities {
		if c.civ != civ || c.producing >= 0 {
			continue
		}
		// Decide what to produce
		threatened := g.isCityThreatened(c)
		numCities := g.countCities(civ)
		numUnits := g.countUnits(civ)

		// Gandhi post-underflow: produce military nonstop
		if gandhiNuke {
			if g.civs[civ].techs[cvTechEngineering] {
				c.producing = cvUnitCatapult // "nukes" (catapults are the best we have)
			} else if g.civs[civ].techs[cvTechIronWorking] {
				c.producing = cvUnitSwordsman
			} else {
				c.producing = cvUnitWarrior
			}
			continue
		}

		if threatened || numUnits < numCities*2 || aggro > 5 {
			// Produce military
			if g.civs[civ].techs[cvTechIronWorking] {
				c.producing = cvUnitSwordsman
			} else {
				c.producing = cvUnitWarrior
			}
		} else if numCities < 3 {
			c.producing = cvUnitSettler
		} else {
			// Build a building
			built := false
			for bi := 0; bi < cvBuildCount; bi++ {
				if !c.buildings[bi] {
					req := cvBuildDefs[bi].reqTech
					if req < 0 || g.civs[civ].techs[req] {
						c.producing = 100 + bi
						built = true
						break
					}
				}
			}
			if !built {
				c.producing = cvUnitWarrior
			}
		}
	}

	// AI unit movement
	for _, u := range g.units {
		if u.dead || u.civ != civ || u.movesLeft <= 0 {
			continue
		}

		if u.unitType == cvUnitSettler {
			g.aiMoveSettler(u, civ)
		} else {
			g.aiMoveMilitary(u, civ)
		}
	}
}

func (g *Civilization) isCityThreatened(c *cvCity) bool {
	for _, u := range g.units {
		if u.dead || u.civ == c.civ {
			continue
		}
		dist := abs(u.x-c.x) + abs(u.y-c.y)
		if dist <= 4 {
			return true
		}
	}
	return false
}

func (g *Civilization) aiMoveSettler(u *cvUnit, civ int) {
	// Find a good location for a new city (far from existing cities)
	if g.countCities(civ) >= 5 {
		return // enough cities
	}

	bestX, bestY := -1, -1
	bestScore := -1

	hasNav := g.civs[civ].techs[cvTechNavigation]

	for x := 2; x < cvMapW-2; x += 3 {
		for y := 2; y < cvMapH-2; y += 3 {
			if !cvIsPassable(g.tiles[x][y].terrain, hasNav) {
				continue
			}
			if g.cityAt(x, y) != nil {
				continue
			}

			// Check if visible to this AI
			if g.civs[civ].fog[x][y] == cvFogUnseen {
				continue
			}

			// Score based on distance from own cities + terrain quality
			minCityDist := 999
			for _, c := range g.cities {
				dist := abs(x-c.x) + abs(y-c.y)
				if dist < minCityDist {
					minCityDist = dist
				}
			}
			if minCityDist < 5 {
				continue // too close
			}

			score := 0
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					nx := x + dx
					ny := y + dy
					if nx >= 0 && nx < cvMapW && ny >= 0 && ny < cvMapH {
						t := g.tiles[nx][ny].terrain
						score += cvTerrYield[t][0] + cvTerrYield[t][1]
					}
				}
			}

			if score > bestScore {
				bestScore = score
				bestX = x
				bestY = y
			}
		}
	}

	if bestX >= 0 {
		// Move toward target
		dist := abs(u.x-bestX) + abs(u.y-bestY)
		if dist <= 1 {
			// Found city here
			g.foundCity(u)
			return
		}
		g.aiMoveToward(u, bestX, bestY)
	}
}

func (g *Civilization) aiMoveMilitary(u *cvUnit, civ int) {
	// Find nearest enemy
	nearDist := 999
	nearX, nearY := -1, -1

	for _, eu := range g.units {
		if eu.dead || eu.civ == civ {
			continue
		}
		// Only target things we can see
		if g.civs[civ].fog[eu.x][eu.y] != cvFogVisible {
			continue
		}
		dist := abs(u.x-eu.x) + abs(u.y-eu.y)
		if dist < nearDist {
			nearDist = dist
			nearX = eu.x
			nearY = eu.y
		}
	}

	// Also consider enemy cities
	for _, ec := range g.cities {
		if ec.civ == civ {
			continue
		}
		if g.civs[civ].fog[ec.x][ec.y] != cvFogVisible {
			continue
		}
		dist := abs(u.x-ec.x) + abs(u.y-ec.y)
		if dist < nearDist {
			nearDist = dist
			nearX = ec.x
			nearY = ec.y
		}
	}

	if nearX < 0 {
		// No visible enemy, explore
		g.aiExplore(u, civ)
		return
	}

	// If adjacent to enemy, attack
	if nearDist <= 1 {
		target := g.unitAt(nearX, nearY)
		if target != nil && target.civ != civ {
			g.doCombat(u, target)
			return
		}
		targetCity := g.cityAt(nearX, nearY)
		if targetCity != nil && targetCity.civ != civ {
			g.attackCity(u, targetCity)
			return
		}
	}

	// Move toward enemy
	g.aiMoveToward(u, nearX, nearY)

	// After moving, check for attack opportunity
	if !u.dead && u.movesLeft > 0 {
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx := u.x + dx
				ny := u.y + dy
				if nx < 0 || nx >= cvMapW || ny < 0 || ny >= cvMapH {
					continue
				}
				target := g.unitAt(nx, ny)
				if target != nil && target.civ != civ && !target.dead {
					g.doCombat(u, target)
					return
				}
			}
		}
	}
}

func (g *Civilization) aiExplore(u *cvUnit, civ int) {
	// Move toward nearest unseen tile
	bestDist := 999
	bestX, bestY := -1, -1
	for x := 0; x < cvMapW; x += 2 {
		for y := 0; y < cvMapH; y += 2 {
			if g.civs[civ].fog[x][y] == cvFogUnseen {
				dist := abs(u.x-x) + abs(u.y-y)
				if dist < bestDist {
					bestDist = dist
					bestX = x
					bestY = y
				}
			}
		}
	}
	if bestX >= 0 {
		g.aiMoveToward(u, bestX, bestY)
	}
}

func (g *Civilization) aiMoveToward(u *cvUnit, tx, ty int) {
	hasNav := g.civs[u.civ].techs[cvTechNavigation]
	for u.movesLeft > 0 {
		bestDx, bestDy := 0, 0
		bestDist := abs(u.x-tx) + abs(u.y-ty)
		found := false

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx := u.x + dx
				ny := u.y + dy
				if nx < 0 || nx >= cvMapW || ny < 0 || ny >= cvMapH {
					continue
				}
				if !cvIsPassable(g.tiles[nx][ny].terrain, hasNav) {
					continue
				}
				eu := g.unitAt(nx, ny)
				if eu != nil {
					continue
				}
				dist := abs(nx-tx) + abs(ny-ty)
				if dist < bestDist {
					bestDist = dist
					bestDx = dx
					bestDy = dy
					found = true
				}
			}
		}

		if !found {
			break
		}
		u.x += bestDx
		u.y += bestDy
		u.movesLeft--
	}
}

// ── Found city ─────────────────────────────────────────────────────────────

func (g *Civilization) foundCity(settler *cvUnit) {
	if settler.unitType != cvUnitSettler {
		return
	}
	if g.cityAt(settler.x, settler.y) != nil {
		return
	}
	t := g.tiles[settler.x][settler.y].terrain
	if t == cvTerrWater || t == cvTerrMountain {
		return
	}

	cityName := g.nextCityName(settler.civ)
	city := &cvCity{
		name:       cityName,
		civ:        settler.civ,
		x:          settler.x,
		y:          settler.y,
		population: 1,
		producing:  -1,
		isCapital:  false,
	}
	g.cities = append(g.cities, city)
	settler.dead = true

	if settler.civ == 0 {
		g.addNotification(fmt.Sprintf("Founded %s!", cityName), color.RGBA{0xff, 0xd7, 0x00, 0xff})
		g.showAnnouncement(fmt.Sprintf("City Founded: %s", cityName))
		if g.audioReady {
			playSFX(g.sfxCity, 0.4)
		}
	} else {
		g.addNotification(fmt.Sprintf("%s founded %s", cvCivNames[settler.civ], cityName), color.RGBA{0x80, 0x80, 0x80, 0xff})
		if settler.civ == 0 && g.audioReady {
			playSFX(g.sfxCity, 0.4)
		}
	}
}

// ── Victory checking ───────────────────────────────────────────────────────

func (g *Civilization) checkVictory() {
	if g.gameOver {
		return
	}

	// Domination: check if all enemy capitals are captured
	for civ := 0; civ < cvMaxCivs; civ++ {
		if !g.civs[civ].alive {
			continue
		}
		ownsAllCapitals := true
		for _, c := range g.cities {
			if c.isCapital && c.civ != civ {
				ownsAllCapitals = false
				break
			}
		}
		if ownsAllCapitals {
			g.gameOver = true
			g.winner = civ
			g.victoryType = "Domination"
			return
		}
	}

	// Science victory
	for civ := 0; civ < cvMaxCivs; civ++ {
		if g.allTechsResearched(civ) {
			g.gameOver = true
			g.winner = civ
			g.victoryType = "Science"
			return
		}
	}

	// Check if any civ has no cities (eliminated)
	for civ := 0; civ < cvMaxCivs; civ++ {
		if !g.civs[civ].alive {
			continue
		}
		if g.countCities(civ) == 0 && g.countUnits(civ) == 0 {
			g.civs[civ].alive = false
		}
	}
}

// ── Update ─────────────────────────────────────────────────────────────────

func (g *Civilization) Update() error {
	g.tick++

	// Update era-based BGM
	if g.audioReady {
		g.cvSwitchBGM()
	}

	// Update animations
	g.updateAnimations()
	g.updateNotifications()

	// Click feedback timer
	if g.clickFeedbackTimer > 0 {
		g.clickFeedbackTimer--
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.showTechTree {
			g.showTechTree = false
		} else if g.showBuild {
			g.showBuild = false
		} else if g.endTurnWarning {
			g.endTurnWarning = false
		} else {
			if g.currentBGM != nil {
				g.currentBGM.Pause()
			}
			g.goBack = true
		}
		return nil
	}

	if g.gameOver {
		return nil
	}

	// Dismiss tutorial on any key/click
	if !g.tutorialDismiss && g.turn == 1 {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
			inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
			inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			g.tutorialDismiss = true
		}
	}

	// Scroll map with smooth acceleration
	scrollSpeed := 3
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.camX -= scrollSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camX += scrollSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.camY -= scrollSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camY += scrollSpeed
	}

	// Mouse edge scrolling
	mx, my := ebiten.CursorPosition()
	edgeMargin := 4
	edgeSpeed := 3
	if mx <= edgeMargin && mx >= 0 {
		g.camX -= edgeSpeed
	}
	if mx >= cvScreenW-edgeMargin && mx <= cvScreenW {
		g.camX += edgeSpeed
	}
	if my <= edgeMargin && my >= 0 {
		g.camY -= edgeSpeed
	}
	if my >= cvScreenH-edgeMargin && my <= cvScreenH {
		g.camY += edgeSpeed
	}

	// Clamp camera
	maxCamX := cvMapW*cvTileS - cvScreenW
	maxCamY := cvMapH*cvTileS - cvScreenH + cvTopBarH + cvBotBarH
	if g.camX < 0 {
		g.camX = 0
	}
	if g.camY < 0 {
		g.camY = 0
	}
	if g.camX > maxCamX {
		g.camX = maxCamX
	}
	if g.camY > maxCamY {
		g.camY = maxCamY
	}

	// Toggle tech tree
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		g.showTechTree = !g.showTechTree
		g.showBuild = false
	}

	// Toggle build menu
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if g.selectedCity >= 0 {
			g.showBuild = !g.showBuild
			g.showTechTree = false
		}
	}

	// Found city with settler
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if g.selectedUnit >= 0 && g.selectedUnit < len(g.units) {
			u := g.units[g.selectedUnit]
			if !u.dead && u.civ == 0 && u.unitType == cvUnitSettler {
				g.foundCity(u)
				g.selectedUnit = -1
				g.moveHighlight = nil
				g.updateFogForCiv(0)
			}
		}
	}

	// Auto-explore toggle (X key)
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		if g.selectedUnit >= 0 && g.selectedUnit < len(g.units) {
			u := g.units[g.selectedUnit]
			if !u.dead && u.civ == 0 && u.unitType != cvUnitSettler {
				u.autoExplore = !u.autoExplore
				if u.autoExplore {
					g.addNotification(fmt.Sprintf("%s set to auto-explore", cvUnitDefs[u.unitType].name), color.RGBA{0xc0, 0xc0, 0x40, 0xff})
					g.selectedUnit = -1
					g.moveHighlight = nil
				} else {
					g.addNotification(fmt.Sprintf("%s stopped auto-explore", cvUnitDefs[u.unitType].name), color.RGBA{0xc0, 0xc0, 0x40, 0xff})
				}
			}
		}
	}

	// TAB: cycle to next unmoved unit
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.selectNextUnmovedUnit()
	}

	// Right-click to deselect
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.selectedUnit = -1
		g.selectedCity = -1
		g.moveHighlight = nil
		g.showBuild = false
		g.endTurnWarning = false
	}

	// End turn with unmoved unit warning
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || g.checkEndTurnButton() {
		unmoved := g.countUnmovedPlayerUnits()
		if g.endTurnWarning || unmoved == 0 {
			g.endTurnWarning = false
			g.endPlayerTurn()
		} else {
			g.endTurnWarning = true
			g.endTurnWarningTick = g.tick
		}
	}

	// Mouse click handling
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.endTurnWarning = false
		g.handleClick()
	}

	return nil
}

func (g *Civilization) checkEndTurnButton() bool {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	mx, my := ebiten.CursorPosition()
	// End turn button at bottom-right
	bx := cvScreenW - 90
	by := cvScreenH - cvBotBarH + 8
	return mx >= bx && mx <= bx+80 && my >= by && my <= by+30
}

func (g *Civilization) endPlayerTurn() {
	if g.audioReady {
		playSFX(g.sfxTurn, 0.3)
	}

	// Run auto-exploring units before processing turn
	for _, u := range g.units {
		if !u.dead && u.civ == 0 && u.autoExplore {
			g.autoExploreUnit(u)
		}
	}

	// Process player's cities/resources
	g.processTurnForCiv(0)

	// AI turns
	for civ := 1; civ < cvMaxCivs; civ++ {
		g.runAI(civ)
	}

	g.turn++
	g.selectedUnit = -1
	g.selectedCity = -1
	g.moveHighlight = nil
	g.showBuild = false

	// Reset player unit moves
	for _, u := range g.units {
		if !u.dead && u.civ == 0 {
			u.movesLeft = cvUnitDefs[u.unitType].moves
		}
	}

	g.updateAllFog()
	g.checkVictory()

	// Clean up dead units
	var alive []*cvUnit
	for _, u := range g.units {
		if !u.dead {
			alive = append(alive, u)
		}
	}
	g.units = alive

	// Fix selectedUnit index if needed
	g.selectedUnit = -1

	// Turn notification
	g.addNotification(fmt.Sprintf("--- Turn %d ---", g.turn), color.RGBA{0xc0, 0xa0, 0x40, 0xff})
}

func (g *Civilization) handleClick() {
	mx, my := ebiten.CursorPosition()

	// Click feedback
	g.clickFeedbackX = mx
	g.clickFeedbackY = my
	g.clickFeedbackTimer = 12

	// Check tech tree overlay clicks
	if g.showTechTree {
		g.handleTechTreeClick(mx, my)
		return
	}

	// Check build menu clicks
	if g.showBuild {
		g.handleBuildClick(mx, my)
		return
	}

	// Minimap click-to-jump
	mmX := cvScreenW - cvMinimapW - 4
	mmY := cvScreenH - cvBotBarH - cvMinimapH - 4
	if mx >= mmX && mx <= mmX+cvMinimapW && my >= mmY && my <= mmY+cvMinimapH {
		// Convert minimap coords to map coords
		relX := float64(mx-mmX) / float64(cvMinimapW)
		relY := float64(my-mmY) / float64(cvMinimapH)
		g.camX = int(relX*float64(cvMapW*cvTileS)) - cvScreenW/2
		g.camY = int(relY*float64(cvMapH*cvTileS)) - cvScreenH/2
		return
	}

	// Don't process map clicks on UI areas
	if my < cvTopBarH || my > cvScreenH-cvBotBarH {
		return
	}

	// Convert screen to tile coords
	tileX := (mx + g.camX) / cvTileS
	tileY := (my - cvTopBarH + g.camY) / cvTileS

	if tileX < 0 || tileX >= cvMapW || tileY < 0 || tileY >= cvMapH {
		return
	}

	// Check if clicking on a unit we want to move to (for selected unit)
	if g.selectedUnit >= 0 && g.selectedUnit < len(g.units) {
		su := g.units[g.selectedUnit]
		if !su.dead && su.civ == 0 {
			// Check if clicking enemy to attack
			targetUnit := g.unitAt(tileX, tileY)
			if targetUnit != nil && targetUnit.civ != 0 && g.canMoveTo(g.moveHighlight, tileX, tileY) {
				// Attack with ranged or melee
				rangeDist := abs(su.x-tileX) + abs(su.y-tileY)
				unitRange := cvUnitDefs[su.unitType].rangeAt
				if unitRange > 0 && rangeDist <= unitRange {
					// Ranged attack (no movement, no retaliation)
					atkStr := float64(cvUnitDefs[su.unitType].strength)
					dmg := atkStr * (0.5 + rand.Float64()*0.5)
					if dmg < 1 {
						dmg = 1
					}
					targetUnit.hp -= int(dmg)
					if targetUnit.hp <= 0 {
						targetUnit.dead = true
					}
					su.movesLeft = 0
					if g.audioReady {
						playSFX(g.sfxCombat, 0.4)
					}
				} else if rangeDist <= 1 {
					g.doCombat(su, targetUnit)
				}
				g.selectedUnit = -1
				g.moveHighlight = nil
				g.updateFogForCiv(0)
				return
			}

			// Check if clicking enemy city to attack
			targetCity := g.cityAt(tileX, tileY)
			if targetCity != nil && targetCity.civ != 0 {
				dist := abs(su.x-tileX) + abs(su.y-tileY)
				if dist <= 1 {
					g.attackCity(su, targetCity)
					g.selectedUnit = -1
					g.moveHighlight = nil
					g.updateFogForCiv(0)
					return
				}
			}

			// Try to move
			if g.canMoveTo(g.moveHighlight, tileX, tileY) {
				oldX, oldY := su.x, su.y
				su.x = tileX
				su.y = tileY
				su.movesLeft--
				if su.movesLeft < 0 {
					su.movesLeft = 0
				}
				g.startMoveAnim(su, oldX, oldY)
				if g.audioReady {
					playSFX(g.sfxMove, 0.2)
				}
				g.updateFogForCiv(0)
				if su.movesLeft > 0 {
					g.moveHighlight = g.computeValidMoves(su)
				} else {
					g.selectedUnit = -1
					g.moveHighlight = nil
				}
				return
			}
		}
	}

	// Try to select own unit at tile
	idx := g.unitIndexAt(tileX, tileY)
	if idx >= 0 && g.units[idx].civ == 0 {
		g.selectedUnit = idx
		g.selectedCity = -1
		g.showBuild = false
		g.moveHighlight = g.computeValidMoves(g.units[idx])
		return
	}

	// Try to select own city at tile
	ci := g.cityIndexAt(tileX, tileY)
	if ci >= 0 && g.cities[ci].civ == 0 {
		g.selectedCity = ci
		g.selectedUnit = -1
		g.moveHighlight = nil
		g.showBuild = true
		return
	}

	// Deselect
	g.selectedUnit = -1
	g.selectedCity = -1
	g.moveHighlight = nil
	g.showBuild = false
}

func (g *Civilization) handleTechTreeClick(mx, my int) {
	// Tech tree overlay - click a tech to research it
	startX := 80
	startY := 72
	techH := 35
	for i := 0; i < cvTechCount; i++ {
		ty := startY + i*techH
		if mx >= startX && mx <= startX+300 && my >= ty && my <= ty+28 {
			if !g.civs[0].techs[i] {
				g.civs[0].researchTarget = i
				g.showTechTree = false
			}
			return
		}
	}
	// Click outside to close
	if mx < 50 || mx > 430 || my < 40 || my > startY+cvTechCount*techH+30 {
		g.showTechTree = false
	}
}

func (g *Civilization) handleBuildClick(mx, my int) {
	if g.selectedCity < 0 || g.selectedCity >= len(g.cities) {
		g.showBuild = false
		return
	}
	c := g.cities[g.selectedCity]
	if c.civ != 0 {
		g.showBuild = false
		return
	}

	startX := cvScreenW - 190
	startY := 50
	itemH := 24
	itemIdx := 0

	// Units
	for i := 0; i < 6; i++ {
		req := cvUnitDefs[i].reqTech
		if req >= 0 && !g.civs[0].techs[req] {
			continue
		}
		ty := startY + 12 + itemIdx*itemH
		if mx >= startX && mx <= startX+170 && my >= ty && my <= ty+20 {
			c.producing = i
			c.prodAccum = 0
			g.showBuild = false
			return
		}
		itemIdx++
	}

	// Buildings
	for i := 0; i < cvBuildCount; i++ {
		if c.buildings[i] {
			continue
		}
		req := cvBuildDefs[i].reqTech
		if req >= 0 && !g.civs[0].techs[req] {
			continue
		}
		ty := startY + 12 + itemIdx*itemH
		if mx >= startX && mx <= startX+170 && my >= ty && my <= ty+20 {
			c.producing = 100 + i
			c.prodAccum = 0
			g.showBuild = false
			return
		}
		itemIdx++
	}
}

// ── Draw ───────────────────────────────────────────────────────────────────

func (g *Civilization) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x10, 0x10, 0x18, 0xff})

	g.drawMap(screen)
	g.drawTopBar(screen)
	g.drawBottomBar(screen)
	g.drawMinimap(screen)

	if g.showTechTree {
		g.drawTechTree(screen)
	}
	if g.showBuild {
		g.drawBuildMenu(screen)
	}

	// Click feedback ring
	if g.clickFeedbackTimer > 0 {
		alpha := uint8(float64(g.clickFeedbackTimer) / 12.0 * 200)
		radius := float32(12 - g.clickFeedbackTimer)
		vector.StrokeCircle(screen, float32(g.clickFeedbackX), float32(g.clickFeedbackY), radius, 1.5, color.RGBA{0xff, 0xff, 0xff, alpha}, false)
	}

	// Combat popups (floating text)
	for _, p := range g.combatPopups {
		sx := float32(p.x*cvTileS-g.camX) + float32(cvTileS)/2
		sy := float32(p.y*cvTileS-g.camY+cvTopBarH) - float32(90-p.timer)/3
		alpha := uint8(0xff)
		if p.timer < 30 {
			alpha = uint8(float64(p.timer) / 30.0 * 255)
		}
		_ = alpha
		ebitenutil.DebugPrintAt(screen, p.text, int(sx)-len(p.text)*3, int(sy))
	}

	// Announcement (centered screen, large text area)
	if g.announcement != "" && g.tick-g.announcementTick < 120 {
		elapsed := g.tick - g.announcementTick
		alpha := uint8(0xff)
		if elapsed > 80 {
			alpha = uint8(float64(120-elapsed) / 40.0 * 255)
		}
		annW := float32(len(g.announcement)*7 + 40)
		annX := float32(cvScreenW)/2 - annW/2
		annY := float32(60)
		vector.DrawFilledRect(screen, annX, annY, annW, 28, color.RGBA{0x10, 0x10, 0x30, alpha}, false)
		vector.StrokeRect(screen, annX, annY, annW, 28, 1.5, color.RGBA{0xc0, 0xa0, 0x40, alpha}, false)
		ebitenutil.DebugPrintAt(screen, g.announcement, int(annX)+20, int(annY)+8)
	}

	// Notification log (bottom-left, stacked)
	g.drawNotifications(screen)

	// Unmoved units warning
	if g.endTurnWarning {
		unmoved := g.countUnmovedPlayerUnits()
		warnText := fmt.Sprintf("You have %d unmoved units. Press SPACE again to end turn.", unmoved)
		wW := float32(len(warnText)*6 + 30)
		wX := float32(cvScreenW)/2 - wW/2
		wY := float32(cvScreenH)/2 - 20
		vector.DrawFilledRect(screen, wX, wY, wW, 36, color.RGBA{0x60, 0x40, 0x10, 0xf0}, false)
		vector.StrokeRect(screen, wX, wY, wW, 36, 2, color.RGBA{0xff, 0xc0, 0x40, 0xff}, false)
		ebitenutil.DebugPrintAt(screen, warnText, int(wX)+15, int(wY)+12)
	}

	// Tutorial overlay (first turn only)
	if g.turn == 1 && !g.tutorialDismiss {
		g.drawTutorial(screen)
	}

	// Contextual controls display
	g.drawContextualHints(screen)

	if g.gameOver {
		g.drawVictoryScreen(screen)
	}
}

func (g *Civilization) drawMap(screen *ebiten.Image) {
	viewX := g.camX
	viewY := g.camY
	startTX := viewX / cvTileS
	startTY := viewY / cvTileS
	endTX := startTX + cvScreenW/cvTileS + 2
	endTY := startTY + (cvScreenH-cvTopBarH-cvBotBarH)/cvTileS + 2

	if endTX > cvMapW {
		endTX = cvMapW
	}
	if endTY > cvMapH {
		endTY = cvMapH
	}

	// Draw terrain
	for tx := startTX; tx < endTX; tx++ {
		for ty := startTY; ty < endTY; ty++ {
			if tx < 0 || tx >= cvMapW || ty < 0 || ty >= cvMapH {
				continue
			}
			fog := g.civs[0].fog[tx][ty]
			if fog == cvFogUnseen {
				// Fully black
				sx := float32(tx*cvTileS - viewX)
				sy := float32(ty*cvTileS - viewY + cvTopBarH)
				vector.DrawFilledRect(screen, sx, sy, float32(cvTileS), float32(cvTileS), color.RGBA{0x05, 0x05, 0x08, 0xff}, false)
				continue
			}

			sx := float32(tx*cvTileS - viewX)
			sy := float32(ty*cvTileS - viewY + cvTopBarH)

			// Position-seeded color variation for richer terrain
			seed := uint32(tx*3571 + ty*2137)
			colorVar := int8((seed % 21) - 10) // -10 to +10

			tCol := cvTerrColors[g.tiles[tx][ty].terrain]
			tCol.R = uint8(clampInt(int(tCol.R)+int(colorVar), 0, 255))
			tCol.G = uint8(clampInt(int(tCol.G)+int(colorVar), 0, 255))
			tCol.B = uint8(clampInt(int(tCol.B)+int(colorVar/2), 0, 255))

			if fog == cvFogRevealed {
				// Dark blue-tinted overlay (not just gray)
				tCol.R = uint8(clampInt(int(tCol.R)/3, 0, 255))
				tCol.G = uint8(clampInt(int(tCol.G)/3, 0, 255))
				tCol.B = uint8(clampInt(int(tCol.B)/3+0x15, 0, 255))
			}

			vector.DrawFilledRect(screen, sx, sy, float32(cvTileS), float32(cvTileS), tCol, false)

			// Terrain decorations for all visible tiles (and dimmed for revealed)
			g.drawTerrainDecoration(screen, tx, ty, sx, sy)

			// Fog of war edge glow at boundary of visibility
			if fog == cvFogVisible {
				for dx := -1; dx <= 1; dx++ {
					for dy := -1; dy <= 1; dy++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := tx+dx, ty+dy
						if nx >= 0 && nx < cvMapW && ny >= 0 && ny < cvMapH {
							if g.civs[0].fog[nx][ny] != cvFogVisible {
								// Draw a subtle glow/gradient at the edge
								edgeX := sx + float32(dx+1)*float32(cvTileS)/3
								edgeY := sy + float32(dy+1)*float32(cvTileS)/3
								vector.DrawFilledCircle(screen, edgeX, edgeY, 3, color.RGBA{0x40, 0x50, 0x80, 0x20}, false)
							}
						}
					}
				}
			}
		}
	}

	// Draw movement highlights with green for empty tiles, red for enemies
	for _, m := range g.moveHighlight {
		fog := g.civs[0].fog[m[0]][m[1]]
		if fog != cvFogVisible {
			continue
		}
		sx := float32(m[0]*cvTileS - viewX)
		sy := float32(m[1]*cvTileS - viewY + cvTopBarH)

		eu := g.unitAt(m[0], m[1])
		ec := g.cityAt(m[0], m[1])
		if eu != nil && eu.civ != 0 {
			// Enemy unit - red highlight
			vector.DrawFilledRect(screen, sx, sy, float32(cvTileS), float32(cvTileS), color.RGBA{0xff, 0x30, 0x30, 0x40}, false)
			vector.StrokeRect(screen, sx, sy, float32(cvTileS), float32(cvTileS), 1, color.RGBA{0xff, 0x40, 0x40, 0x80}, false)
		} else if ec != nil && ec.civ != 0 {
			// Enemy city - red highlight
			vector.DrawFilledRect(screen, sx, sy, float32(cvTileS), float32(cvTileS), color.RGBA{0xff, 0x30, 0x30, 0x40}, false)
			vector.StrokeRect(screen, sx, sy, float32(cvTileS), float32(cvTileS), 1, color.RGBA{0xff, 0x40, 0x40, 0x80}, false)
		} else {
			// Valid move - green highlight
			vector.DrawFilledRect(screen, sx, sy, float32(cvTileS), float32(cvTileS), color.RGBA{0x40, 0xff, 0x40, 0x28}, false)
			vector.StrokeRect(screen, sx, sy, float32(cvTileS), float32(cvTileS), 0.5, color.RGBA{0x40, 0xff, 0x40, 0x50}, false)
		}
	}

	// Draw cities
	for _, c := range g.cities {
		fog := g.civs[0].fog[c.x][c.y]
		if fog == cvFogUnseen {
			continue
		}

		sx := float32(c.x*cvTileS - viewX)
		sy := float32(c.y*cvTileS - viewY + cvTopBarH)
		col := cvCivColors[c.civ]
		ts := float32(cvTileS)

		if fog == cvFogRevealed {
			// Show city but dimmed
			dimCol := color.RGBA{col.R / 3, col.G / 3, col.B / 3, 0xff}
			vector.DrawFilledRect(screen, sx+3, sy+4, 5, 6, dimCol, false)
			vector.DrawFilledRect(screen, sx+9, sy+5, 4, 5, dimCol, false)
			continue
		}

		// Glow effect around city in civ color
		glowAlpha := uint8(0x18 + 0x08*float32(math.Sin(float64(g.tick)*0.05)))
		glowCol := color.RGBA{col.R, col.G, col.B, glowAlpha}
		vector.DrawFilledCircle(screen, sx+ts/2, sy+ts/2, ts*0.7, glowCol, false)

		// Cluster of buildings (3-4 small rectangles)
		bldgCol := color.RGBA{
			uint8(clampInt(int(col.R)+0x30, 0, 255)),
			uint8(clampInt(int(col.G)+0x30, 0, 255)),
			uint8(clampInt(int(col.B)+0x30, 0, 255)),
			0xff,
		}
		darkCol := color.RGBA{col.R / 2, col.G / 2, col.B / 2, 0xff}

		// Building 1 (tall left)
		vector.DrawFilledRect(screen, sx+2, sy+3, 4, 8, bldgCol, false)
		vector.StrokeRect(screen, sx+2, sy+3, 4, 8, 0.5, darkCol, false)
		// Building 2 (short center)
		vector.DrawFilledRect(screen, sx+6, sy+6, 4, 5, col, false)
		vector.StrokeRect(screen, sx+6, sy+6, 4, 5, 0.5, darkCol, false)
		// Building 3 (medium right)
		vector.DrawFilledRect(screen, sx+10, sy+4, 3, 7, bldgCol, false)
		vector.StrokeRect(screen, sx+10, sy+4, 3, 7, 0.5, darkCol, false)
		// Building 4 (small front)
		if c.population > 2 {
			vector.DrawFilledRect(screen, sx+5, sy+9, 3, 3, col, false)
		}

		// Walls visible if Walls building is built
		if c.buildings[cvBuildWalls] {
			wallCol := color.RGBA{0xa0, 0x90, 0x70, 0xff}
			vector.StrokeRect(screen, sx, sy+1, ts, ts-2, 1.5, wallCol, false)
			// Battlements (small notches on top)
			for wx := sx; wx < sx+ts; wx += 3 {
				vector.DrawFilledRect(screen, wx, sy, 2, 2, wallCol, false)
			}
		}

		// Population number drawn larger and clearer (with shadow)
		popStr := fmt.Sprintf("%d", c.population)
		ebitenutil.DebugPrintAt(screen, popStr, int(sx)+1, int(sy)-8)
		// Bright foreground
		ebitenutil.DebugPrintAt(screen, popStr, int(sx), int(sy)-9)

		// Capital city gets a star/crown symbol
		if c.isCapital {
			starCol := color.RGBA{0xff, 0xd7, 0x00, 0xff}
			// Star: small diamond on top
			scx := sx + ts/2
			scy := sy - 3
			vector.StrokeLine(screen, scx, scy-3, scx-2, scy, 1, starCol, false)
			vector.StrokeLine(screen, scx, scy-3, scx+2, scy, 1, starCol, false)
			vector.StrokeLine(screen, scx-3, scy-1, scx+3, scy-1, 1, starCol, false)
			vector.StrokeLine(screen, scx-2, scy, scx-1, scy+2, 0.7, starCol, false)
			vector.StrokeLine(screen, scx+2, scy, scx+1, scy+2, 0.7, starCol, false)
		}

		// City name + production info
		cityLabel := c.name
		if c.producing >= 0 {
			var prodName string
			var cost int
			if c.producing < 100 && c.producing >= 0 && c.producing < 6 {
				prodName = cvUnitDefs[c.producing].name
				cost = cvUnitDefs[c.producing].cost
			} else if c.producing >= 100 {
				bi := c.producing - 100
				if bi >= 0 && bi < cvBuildCount {
					prodName = cvBuildDefs[bi].name
					cost = cvBuildDefs[bi].cost
				}
			}
			if prodName != "" && cost > 0 {
				_, cityProd, _ := g.cityYield(c)
				remaining := cost - c.prodAccum
				turnsLeft := (remaining + cityProd - 1) / cityProd
				if turnsLeft < 1 {
					turnsLeft = 1
				}
				cityLabel = fmt.Sprintf("%s [%s %dt]", c.name, prodName, turnsLeft)
			}
		}
		ebitenutil.DebugPrintAt(screen, cityLabel, int(sx)-2, int(sy)+int(ts)+1)

		// Production progress bar
		if c.producing >= 0 {
			var cost int
			if c.producing < 100 {
				if c.producing >= 0 && c.producing < 6 {
					cost = cvUnitDefs[c.producing].cost
				}
			} else {
				bi := c.producing - 100
				if bi >= 0 && bi < cvBuildCount {
					cost = cvBuildDefs[bi].cost
				}
			}
			if cost > 0 {
				progress := float32(c.prodAccum) / float32(cost)
				if progress > 1 {
					progress = 1
				}
				barW := ts - 2
				vector.DrawFilledRect(screen, sx+1, sy+ts-3, barW, 2, color.RGBA{0x40, 0x20, 0x20, 0xff}, false)
				vector.DrawFilledRect(screen, sx+1, sy+ts-3, barW*progress, 2, color.RGBA{0x40, 0xff, 0x40, 0xff}, false)
			}
		}
	}

	// Draw units
	for i, u := range g.units {
		if u.dead {
			continue
		}
		fog := g.civs[0].fog[u.x][u.y]
		if fog != cvFogVisible && u.civ != 0 {
			continue
		}

		// Animated position interpolation
		drawX := float64(u.x)
		drawY := float64(u.y)
		if u.animProgress < 1.0 {
			drawX = u.animFromX + (float64(u.x)-u.animFromX)*u.animProgress
			drawY = u.animFromY + (float64(u.y)-u.animFromY)*u.animProgress
		}

		sx := float32(drawX*float64(cvTileS)) - float32(viewX)
		sy := float32(drawY*float64(cvTileS)) - float32(viewY) + float32(cvTopBarH)
		ts := float32(cvTileS)
		col := cvCivColors[u.civ]

		cx := sx + ts/2
		cy := sy + ts/2

		// Colored border ring matching civ
		vector.StrokeCircle(screen, cx, cy, 6.5, 1.2, col, false)
		// VERY obvious pulsing highlight when selected
		if i == g.selectedUnit {
			pulseVal := math.Sin(float64(g.tick) * 0.12)
			pulseAlpha := uint8(0x80 + 0x7f*pulseVal)
			pulseRadius := float32(8.0 + 1.5*pulseVal)
			vector.StrokeCircle(screen, cx, cy, pulseRadius, 2.5, color.RGBA{0xff, 0xff, 0x40, pulseAlpha}, false)
			// Outer glow ring
			glowAlpha := uint8(0x30 + 0x20*pulseVal)
			vector.StrokeCircle(screen, cx, cy, pulseRadius+2, 1.0, color.RGBA{0xff, 0xff, 0xff, glowAlpha}, false)
		}

		// AUTO label for auto-exploring units
		if u.autoExplore {
			ebitenutil.DebugPrintAt(screen, "AUTO", int(sx)-2, int(sy)-10)
		}

		// Draw unit shape based on type
		switch u.unitType {
		case cvUnitSettler:
			// Wagon shape: rectangle body + circle wheels
			vector.DrawFilledRect(screen, cx-5, cy-3, 10, 5, col, false)
			vector.StrokeRect(screen, cx-5, cy-3, 10, 5, 0.5, color.RGBA{col.R / 2, col.G / 2, col.B / 2, 0xff}, false)
			// Wheels
			vector.DrawFilledCircle(screen, cx-3, cy+3, 1.5, color.RGBA{0x60, 0x50, 0x30, 0xff}, false)
			vector.DrawFilledCircle(screen, cx+3, cy+3, 1.5, color.RGBA{0x60, 0x50, 0x30, 0xff}, false)
			// Canvas top
			vector.StrokeLine(screen, cx-4, cy-3, cx, cy-5, 0.8, color.RGBA{0xe0, 0xd0, 0xb0, 0xff}, false)
			vector.StrokeLine(screen, cx, cy-5, cx+4, cy-3, 0.8, color.RGBA{0xe0, 0xd0, 0xb0, 0xff}, false)

		case cvUnitWarrior:
			// Shield shape: small rectangle + circle on top
			vector.DrawFilledRect(screen, cx-3, cy-1, 6, 6, col, false)
			vector.DrawFilledCircle(screen, cx, cy-2, 3, col, false)
			// Shield boss (center dot)
			vector.DrawFilledCircle(screen, cx, cy+1, 1, color.RGBA{0xe0, 0xd0, 0x80, 0xff}, false)

		case cvUnitArcher:
			// Arrow/triangle shape pointing up
			cvDrawTriangle(screen, cx, cy-5, cx-4, cy+4, cx+4, cy+4, col)
			// Arrow shaft
			vector.StrokeLine(screen, cx, cy-5, cx, cy+4, 0.8, color.RGBA{0xe0, 0xd0, 0x80, 0xff}, false)
			// Fletching
			vector.StrokeLine(screen, cx-2, cy+2, cx, cy+4, 0.6, color.RGBA{0xff, 0xff, 0xff, 0xc0}, false)
			vector.StrokeLine(screen, cx+2, cy+2, cx, cy+4, 0.6, color.RGBA{0xff, 0xff, 0xff, 0xc0}, false)

		case cvUnitSwordsman:
			// Cross/sword shape
			// Blade (vertical)
			vector.DrawFilledRect(screen, cx-1, cy-5, 2, 9, color.RGBA{0xc0, 0xc8, 0xd0, 0xff}, false)
			// Crossguard (horizontal)
			vector.DrawFilledRect(screen, cx-4, cy-1, 8, 2, col, false)
			// Handle
			vector.DrawFilledRect(screen, cx-1, cy+4, 2, 2, color.RGBA{0x80, 0x60, 0x30, 0xff}, false)
			// Pommel
			vector.DrawFilledCircle(screen, cx, cy+6, 1, color.RGBA{0xe0, 0xd0, 0x40, 0xff}, false)

		case cvUnitHorseman:
			// Wider rectangle (horse body) + head
			vector.DrawFilledRect(screen, cx-6, cy-2, 10, 5, col, false)
			// Horse head (small rect extending forward)
			vector.DrawFilledRect(screen, cx+4, cy-4, 3, 4, col, false)
			// Legs
			vector.StrokeLine(screen, cx-4, cy+3, cx-4, cy+5, 1, col, false)
			vector.StrokeLine(screen, cx+2, cy+3, cx+2, cy+5, 1, col, false)
			// Eye
			vector.DrawFilledCircle(screen, cx+5.5, cy-3, 0.5, color.White, false)

		case cvUnitCatapult:
			// Triangle on rectangle (siege weapon)
			// Base platform
			vector.DrawFilledRect(screen, cx-5, cy+1, 10, 4, color.RGBA{0x70, 0x50, 0x30, 0xff}, false)
			// Arm (triangle above)
			cvDrawTriangle(screen, cx, cy-5, cx-4, cy+1, cx+4, cy+1, col)
			// Wheels
			vector.DrawFilledCircle(screen, cx-4, cy+5, 1.5, color.RGBA{0x50, 0x40, 0x20, 0xff}, false)
			vector.DrawFilledCircle(screen, cx+4, cy+5, 1.5, color.RGBA{0x50, 0x40, 0x20, 0xff}, false)
			// Projectile
			vector.DrawFilledCircle(screen, cx, cy-4, 1, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
		}

		// HP bar below unit (always shown, red/green)
		barW := ts - 2
		hpFrac := float32(u.hp) / 10.0
		barY := sy + ts + 1
		// Background (red)
		vector.DrawFilledRect(screen, sx+1, barY, barW, 2, color.RGBA{0xb0, 0x20, 0x20, 0xff}, false)
		// Foreground (green)
		greenVal := uint8(0xc0)
		if hpFrac < 0.5 {
			greenVal = uint8(float32(0xc0) * hpFrac * 2)
		}
		redVal := uint8(0)
		if hpFrac < 0.5 {
			redVal = uint8(float32(0xc0) * (1 - hpFrac*2))
		}
		vector.DrawFilledRect(screen, sx+1, barY, barW*hpFrac, 2, color.RGBA{redVal, greenVal, 0, 0xff}, false)

		// Movement dots below HP bar showing remaining moves
		dotY := barY + 3
		maxMoves := cvUnitDefs[u.unitType].moves
		for mi := 0; mi < maxMoves; mi++ {
			dotX := sx + 2 + float32(mi)*4
			dotCol := color.RGBA{0x40, 0x40, 0x40, 0xff}
			if mi < u.movesLeft {
				dotCol = color.RGBA{0xff, 0xff, 0x60, 0xff}
			}
			vector.DrawFilledCircle(screen, dotX+1, dotY+1, 1.2, dotCol, false)
		}
	}

	// Grid lines (subtle)
	for tx := startTX; tx <= endTX; tx++ {
		sx := float32(tx*cvTileS - viewX)
		vector.StrokeLine(screen, sx, float32(cvTopBarH), sx, float32(cvScreenH-cvBotBarH), 0.5, color.RGBA{0, 0, 0, 0x30}, false)
	}
	for ty := startTY; ty <= endTY; ty++ {
		sy := float32(ty*cvTileS - viewY + cvTopBarH)
		vector.StrokeLine(screen, 0, sy, float32(cvScreenW), sy, 0.5, color.RGBA{0, 0, 0, 0x30}, false)
	}
}

func cvDrawTriangle(screen *ebiten.Image, x1, y1, x2, y2, x3, y3 float32, col color.RGBA) {
	// Simple filled triangle using lines
	// Sort by y
	if y1 > y2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	if y1 > y3 {
		x1, y1, x3, y3 = x3, y3, x1, y1
	}
	if y2 > y3 {
		x2, y2, x3, y3 = x3, y3, x2, y2
	}

	totalH := y3 - y1
	if totalH < 1 {
		return
	}

	for y := y1; y <= y3; y++ {
		var lx, rx float32
		if y < y2 {
			segH := y2 - y1
			if segH < 1 {
				segH = 1
			}
			t := (y - y1) / segH
			lx = x1 + (x2-x1)*t
		} else {
			segH := y3 - y2
			if segH < 1 {
				segH = 1
			}
			t := (y - y2) / segH
			lx = x2 + (x3-x2)*t
		}
		t2 := (y - y1) / totalH
		rx = x1 + (x3-x1)*t2
		if lx > rx {
			lx, rx = rx, lx
		}
		vector.StrokeLine(screen, lx, y, rx, y, 1, col, false)
	}
}

func (g *Civilization) drawTerrainDecoration(screen *ebiten.Image, tx, ty int, sx, sy float32) {
	terrain := g.tiles[tx][ty].terrain
	// Seed position-based randomness for consistent per-tile variation
	seed := uint32(tx*7919 + ty*6271)
	seedRand := func() float32 {
		seed = seed*1103515245 + 12345
		return float32(seed%1000) / 1000.0
	}
	ts := float32(cvTileS)
	anim := float64(g.tick)

	switch terrain {
	case cvTerrGrass:
		// Grass tufts - small lines in varied green shades
		for i := 0; i < 4; i++ {
			gx := sx + seedRand()*ts
			gy := sy + seedRand()*ts
			gh := 2 + seedRand()*3
			greenVar := uint8(0x30 + seedRand()*0x40)
			tufCol := color.RGBA{0x20 + uint8(seedRand()*0x30), 0x80 + greenVar, 0x20, 0xff}
			vector.StrokeLine(screen, gx, gy, gx+seedRand()*2-1, gy-gh, 0.7, tufCol, false)
		}

	case cvTerrPlains:
		// Wheat-colored grain dots
		for i := 0; i < 5; i++ {
			px := sx + seedRand()*ts
			py := sy + seedRand()*ts
			wheatCol := color.RGBA{0xc0 + uint8(seedRand()*0x20), 0xa0 + uint8(seedRand()*0x20), 0x40 + uint8(seedRand()*0x10), 0xff}
			vector.DrawFilledCircle(screen, px, py, 0.5+seedRand()*0.5, wheatCol, false)
		}

	case cvTerrForest:
		// Dark green base already drawn; add 2-3 tree circles of varying shades + shadows
		for i := 0; i < 3; i++ {
			cx := sx + 3 + seedRand()*(ts-6)
			cy := sy + 3 + seedRand()*(ts-6)
			r := 2 + seedRand()*2
			shade := uint8(0x18 + seedRand()*0x30)
			treeCol := color.RGBA{shade / 2, 0x40 + shade, shade / 3, 0xff}
			// Shadow beneath tree
			vector.DrawFilledCircle(screen, cx+1, cy+1, r, color.RGBA{0x08, 0x20, 0x08, 0x80}, false)
			// Tree crown
			vector.DrawFilledCircle(screen, cx, cy, r, treeCol, false)
		}
		// Tree trunk hint (tiny dark line)
		vector.StrokeLine(screen, sx+ts/2, sy+ts-3, sx+ts/2, sy+ts-1, 1, color.RGBA{0x40, 0x30, 0x10, 0xff}, false)

	case cvTerrHills:
		// Brown gradient: lighter at top, darker at bottom + contour lines
		for row := float32(0); row < ts; row += 2 {
			frac := row / ts
			r := uint8(float32(0xa0) - frac*0x30)
			green := uint8(float32(0x78) - frac*0x20)
			blue := uint8(float32(0x40) - frac*0x10)
			gradCol := color.RGBA{r, green, blue, 0x60}
			vector.DrawFilledRect(screen, sx, sy+row, ts, 2, gradCol, false)
		}
		// Contour lines
		cx := sx + ts/2
		vector.StrokeLine(screen, sx+2, sy+ts*0.7, cx, sy+ts*0.3, 0.8, color.RGBA{0x60, 0x48, 0x28, 0xc0}, false)
		vector.StrokeLine(screen, cx, sy+ts*0.3, sx+ts-2, sy+ts*0.7, 0.8, color.RGBA{0x60, 0x48, 0x28, 0xc0}, false)
		// Second smaller contour
		vector.StrokeLine(screen, sx+4, sy+ts*0.55, cx, sy+ts*0.2, 0.5, color.RGBA{0x70, 0x58, 0x38, 0x80}, false)
		vector.StrokeLine(screen, cx, sy+ts*0.2, sx+ts-4, sy+ts*0.55, 0.5, color.RGBA{0x70, 0x58, 0x38, 0x80}, false)

	case cvTerrMountain:
		// Gray base with white snow cap at top, dark rock at bottom
		cx := sx + ts/2
		// Dark rock at bottom
		vector.DrawFilledRect(screen, sx, sy+ts*0.6, ts, ts*0.4, color.RGBA{0x50, 0x50, 0x55, 0xa0}, false)
		// Mountain silhouette
		cvDrawTriangle(screen, cx, sy+1, sx+1, sy+ts-1, sx+ts-1, sy+ts-1, color.RGBA{0x78, 0x78, 0x80, 0xff})
		// Snow cap (top third)
		cvDrawTriangle(screen, cx, sy+1, cx-3, sy+ts*0.35, cx+3, sy+ts*0.35, color.RGBA{0xf0, 0xf0, 0xff, 0xff})
		// Rock texture lines
		vector.StrokeLine(screen, cx-2, sy+ts*0.5, cx+3, sy+ts*0.6, 0.5, color.RGBA{0x40, 0x40, 0x48, 0x80}, false)

	case cvTerrWater:
		// Animated waves using sine, blue color oscillation, foam at edges
		wavePhase := float32(math.Sin(anim*0.05)) * 2
		blueShift := uint8(0x10 + 0x08*float32(math.Sin(anim*0.03+float64(tx))))
		for row := float32(0); row < ts; row += 2 {
			for col := float32(0); col < ts; col += 2 {
				wy := float32(math.Sin(float64(col+sx+wavePhase)*0.4+float64(row)*0.3+anim*0.06)) * 1.2
				waterCol := color.RGBA{0x20 + uint8(wy*8), 0x50 + uint8(wy*12), 0xb0 + blueShift + uint8(wy*10), 0xff}
				vector.DrawFilledRect(screen, sx+col, sy+row+wy, 2, 2, waterCol, false)
			}
		}
		// Foam at edges near land
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx, ny := tx+dx, ty+dy
				if nx >= 0 && nx < cvMapW && ny >= 0 && ny < cvMapH && g.tiles[nx][ny].terrain != cvTerrWater {
					// Draw foam on the edge facing land
					foamX := sx + float32(dx+1)*ts/3
					foamY := sy + float32(dy+1)*ts/3
					foamAlpha := uint8(0x60 + 0x20*float32(math.Sin(anim*0.08+float64(dx+dy))))
					vector.DrawFilledCircle(screen, foamX, foamY, 1.2, color.RGBA{0xd0, 0xe0, 0xf0, foamAlpha}, false)
				}
			}
		}

	case cvTerrDesert:
		// Sandy with darker spots + heat shimmer effect
		for i := 0; i < 3; i++ {
			dx := seedRand() * ts
			dy := seedRand() * ts
			vector.DrawFilledCircle(screen, sx+dx, sy+dy, 1+seedRand(), color.RGBA{0xb0, 0xa0, 0x60, 0x60}, false)
		}
		// Heat shimmer: subtle vertical offset dots
		if g.tick%3 == 0 {
			shimX := sx + seedRand()*ts
			shimY := sy + float32(math.Sin(anim*0.1+float64(tx)))*1.5 + ts/2
			vector.DrawFilledCircle(screen, shimX, shimY, 0.5, color.RGBA{0xe0, 0xd0, 0x90, 0x30}, false)
		}

	case cvTerrTundra:
		// White-ish with blue tint, small snowflake dots
		for i := 0; i < 4; i++ {
			fx := sx + seedRand()*ts
			fy := sy + seedRand()*ts
			snowCol := color.RGBA{0xd0 + uint8(seedRand()*0x20), 0xd0 + uint8(seedRand()*0x20), 0xe8, 0xb0}
			r := 0.5 + seedRand()*0.8
			vector.DrawFilledCircle(screen, fx, fy, r, snowCol, false)
		}
		// Blue tint overlay hint
		vector.DrawFilledRect(screen, sx, sy, ts, ts, color.RGBA{0x80, 0x90, 0xc0, 0x18}, false)
	}
}

func (g *Civilization) drawTopBar(screen *ebiten.Image) {
	// Dark semi-transparent background
	vector.DrawFilledRect(screen, 0, 0, float32(cvScreenW), float32(cvTopBarH), color.RGBA{0x12, 0x12, 0x1e, 0xf0}, false)
	// Gold accent line at bottom
	vector.DrawFilledRect(screen, 0, float32(cvTopBarH-1), float32(cvScreenW), 1, color.RGBA{0xc0, 0xa0, 0x40, 0xff}, false)

	food, prod, sci := g.getCivTotals(0)

	// Era names for display
	eraNames := [4]string{"Ancient", "Medieval", "Industrial", "Nuclear"}
	eraStr := eraNames[g.currentEra]

	turnStr := fmt.Sprintf("TURN %d | %s Era", g.turn, eraStr)
	ebitenutil.DebugPrintAt(screen, turnStr, 8, 6)
	// Highlight turn number with gold underline
	vector.DrawFilledRect(screen, 8, 17, float32(len(turnStr)*6), 1, color.RGBA{0xc0, 0xa0, 0x40, 0x60}, false)

	// Icons before resource text
	resStr := fmt.Sprintf("F:%d  P:%d  S:%d", food, prod, sci)
	ebitenutil.DebugPrintAt(screen, resStr, 180, 6)

	// Food/Prod/Sci colored indicators
	vector.DrawFilledCircle(screen, 176, 11, 2, color.RGBA{0x40, 0xc0, 0x40, 0xff}, false) // food green
	vector.DrawFilledCircle(screen, 212, 11, 2, color.RGBA{0xc0, 0x80, 0x20, 0xff}, false) // prod orange
	vector.DrawFilledCircle(screen, 248, 11, 2, color.RGBA{0x40, 0x80, 0xff, 0xff}, false) // sci blue

	// Current research
	rt := g.civs[0].researchTarget
	if rt >= 0 && rt < cvTechCount {
		cost := cvTechs[rt].cost
		progress := g.civs[0].scienceAccum
		if progress > cost {
			progress = cost
		}
		techStr := fmt.Sprintf("Research: %s (%d/%d)", cvTechs[rt].name, progress, cost)
		ebitenutil.DebugPrintAt(screen, techStr, 310, 6)

		// Progress bar with border
		barX := float32(520)
		barY := float32(7)
		barW := float32(100)
		barH := float32(10)
		vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{0x20, 0x20, 0x30, 0xff}, false)
		vector.StrokeRect(screen, barX, barY, barW, barH, 0.5, color.RGBA{0x60, 0x60, 0x80, 0xff}, false)
		if cost > 0 {
			frac := float32(progress) / float32(cost)
			if frac > 1 {
				frac = 1
			}
			vector.DrawFilledRect(screen, barX+1, barY+1, (barW-2)*frac, barH-2, color.RGBA{0x30, 0x90, 0xff, 0xff}, false)
		}
	}
}

func (g *Civilization) drawBottomBar(screen *ebiten.Image) {
	barY := float32(cvScreenH - cvBotBarH)
	// Dark semi-transparent background
	vector.DrawFilledRect(screen, 0, barY, float32(cvScreenW), float32(cvBotBarH), color.RGBA{0x12, 0x12, 0x1e, 0xf0}, false)
	// Gold accent line at top
	vector.DrawFilledRect(screen, 0, barY, float32(cvScreenW), 1, color.RGBA{0xc0, 0xa0, 0x40, 0xff}, false)

	// Selected unit info
	if g.selectedUnit >= 0 && g.selectedUnit < len(g.units) {
		u := g.units[g.selectedUnit]
		if !u.dead {
			def := cvUnitDefs[u.unitType]
			// Unit type icon (colored circle)
			vector.DrawFilledCircle(screen, 12, barY+14, 5, cvCivColors[u.civ], false)

			info := fmt.Sprintf("%s  HP:%d/10  Str:%d  Moves:%d/%d",
				def.name, u.hp, def.strength, u.movesLeft, def.moves)
			ebitenutil.DebugPrintAt(screen, info, 22, int(barY)+8)

			// HP mini-bar in bottom bar
			hpBarX := float32(22)
			hpBarY := barY + 22
			hpBarW := float32(80)
			hpFrac := float32(u.hp) / 10.0
			vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarW, 3, color.RGBA{0x60, 0x20, 0x20, 0xff}, false)
			vector.DrawFilledRect(screen, hpBarX, hpBarY, hpBarW*hpFrac, 3, color.RGBA{0x20, 0xc0, 0x20, 0xff}, false)

			actions := ""
			if u.unitType == cvUnitSettler {
				actions = "[F] Found City"
			} else {
				actions = "[X] Auto-Explore"
				if u.autoExplore {
					actions = "[X] Stop Auto  (AUTO)"
				}
			}
			if def.rangeAt > 0 {
				actions += fmt.Sprintf("  Range:%d", def.rangeAt)
			}
			ebitenutil.DebugPrintAt(screen, actions, 22, int(barY)+28)
		}
	}

	// Selected city info
	if g.selectedCity >= 0 && g.selectedCity < len(g.cities) {
		c := g.cities[g.selectedCity]
		food, prod, sci := g.cityYield(c)

		// City icon
		vector.DrawFilledRect(screen, 8, barY+8, 8, 10, cvCivColors[c.civ], false)
		if c.isCapital {
			vector.DrawFilledCircle(screen, 12, barY+6, 2, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
		}

		info := fmt.Sprintf("%s  Pop:%d  F:%d P:%d S:%d",
			c.name, c.population, food, prod, sci)
		ebitenutil.DebugPrintAt(screen, info, 22, int(barY)+8)

		prodStr := "Producing: None [B]"
		if c.producing >= 0 {
			if c.producing < 100 {
				if c.producing >= 0 && c.producing < 6 {
					cost := cvUnitDefs[c.producing].cost
					prodStr = fmt.Sprintf(">> %s (%d/%d)", cvUnitDefs[c.producing].name, c.prodAccum, cost)
				}
			} else {
				bi := c.producing - 100
				if bi >= 0 && bi < cvBuildCount {
					cost := cvBuildDefs[bi].cost
					prodStr = fmt.Sprintf(">> %s (%d/%d)", cvBuildDefs[bi].name, c.prodAccum, cost)
				}
			}
		}
		ebitenutil.DebugPrintAt(screen, prodStr, 22, int(barY)+24)
	}

	// End turn button with polished style
	bx := float32(cvScreenW - 95)
	by := barY + 8
	unmoved := g.countUnmovedPlayerUnits()
	btnBgCol := color.RGBA{0x28, 0x50, 0x28, 0xff}
	btnBorderCol := color.RGBA{0x80, 0xc0, 0x60, 0xff}
	btnLabel := "END TURN"
	if unmoved > 0 && !g.endTurnWarning {
		// Dimmer when units remain
		btnBgCol = color.RGBA{0x30, 0x30, 0x20, 0xff}
		btnBorderCol = color.RGBA{0x80, 0x80, 0x40, 0xff}
	}
	if g.endTurnWarning {
		// Pulsing orange when warning
		pulseVal := math.Sin(float64(g.tick) * 0.15)
		btnBgCol = color.RGBA{0x60, 0x40, 0x10, 0xff}
		btnBorderCol = color.RGBA{0xff, 0xc0, uint8(0x40 + 0x20*pulseVal), 0xff}
		btnLabel = "CONFIRM?"
	}
	// Button glow
	vector.DrawFilledRect(screen, bx-1, by-1, 87, 34, color.RGBA{btnBgCol.R / 2, btnBgCol.G / 2, btnBgCol.B / 2, 0x40}, false)
	vector.DrawFilledRect(screen, bx, by, 85, 32, btnBgCol, false)
	vector.StrokeRect(screen, bx, by, 85, 32, 1.5, btnBorderCol, false)
	// Gold inner accent
	vector.StrokeRect(screen, bx+2, by+2, 81, 28, 0.5, color.RGBA{0xc0, 0xa0, 0x40, 0x60}, false)
	ebitenutil.DebugPrintAt(screen, btnLabel, int(bx)+14, int(by)+10)

	// Controls hint at bottom
	ebitenutil.DebugPrintAt(screen, "[T]Tech [B]Build [TAB]Next Unit [X]Auto-Explore [SPACE]End Turn", 110, int(barY)+38)

	// Gandhi nuclear warning message
	if g.gandhiNukeMsg && g.tick-g.gandhiNukeMsgTick < 360 {
		msgAlpha := uint8(0xff)
		elapsed := g.tick - g.gandhiNukeMsgTick
		if elapsed > 300 {
			msgAlpha = uint8(float64(0xff) * float64(360-elapsed) / 60.0)
		}
		// Red warning banner
		vector.DrawFilledRect(screen, 40, 200, 560, 40, color.RGBA{0x80, 0x10, 0x10, msgAlpha}, false)
		vector.StrokeRect(screen, 40, 200, 560, 40, 2, color.RGBA{0xff, 0x40, 0x40, msgAlpha}, false)
		ebitenutil.DebugPrintAt(screen, "GANDHI HAS DISCOVERED DEMOCRACY.", 60, 206)
		ebitenutil.DebugPrintAt(screen, "OUR WORDS ARE BACKED WITH NUCLEAR WEAPONS.", 60, 220)
	}
}

func (g *Civilization) drawMinimap(screen *ebiten.Image) {
	mmX := float32(cvScreenW - cvMinimapW - 4)
	mmY := float32(cvScreenH - cvBotBarH - cvMinimapH - 4)

	// Background with border
	vector.DrawFilledRect(screen, mmX-2, mmY-2, float32(cvMinimapW)+4, float32(cvMinimapH)+4, color.RGBA{0x08, 0x08, 0x10, 0xff}, false)
	// Gold accent border
	vector.StrokeRect(screen, mmX-2, mmY-2, float32(cvMinimapW)+4, float32(cvMinimapH)+4, 1.5, color.RGBA{0xc0, 0xa0, 0x40, 0xff}, false)

	scaleX := float32(cvMinimapW) / float32(cvMapW)
	scaleY := float32(cvMinimapH) / float32(cvMapH)

	// Draw terrain
	for tx := 0; tx < cvMapW; tx++ {
		for ty := 0; ty < cvMapH; ty++ {
			fog := g.civs[0].fog[tx][ty]
			if fog == cvFogUnseen {
				continue
			}
			col := cvTerrColors[g.tiles[tx][ty].terrain]
			if fog == cvFogRevealed {
				col.R /= 3
				col.G /= 3
				col.B = uint8(clampInt(int(col.B)/3+0x10, 0, 255))
			}
			px := mmX + float32(tx)*scaleX
			py := mmY + float32(ty)*scaleY
			vector.DrawFilledRect(screen, px, py, scaleX+1, scaleY+1, col, false)
		}
	}

	// Draw cities on minimap as larger colored dots
	for _, c := range g.cities {
		fog := g.civs[0].fog[c.x][c.y]
		if fog == cvFogUnseen {
			continue
		}
		px := mmX + float32(c.x)*scaleX + scaleX/2
		py := mmY + float32(c.y)*scaleY + scaleY/2
		dotSize := float32(2.5)
		if c.isCapital {
			dotSize = 3.5
		}
		vector.DrawFilledCircle(screen, px, py, dotSize, cvCivColors[c.civ], false)
	}

	// Draw units on minimap as tiny dots
	for _, u := range g.units {
		if u.dead {
			continue
		}
		fog := g.civs[0].fog[u.x][u.y]
		if fog != cvFogVisible && u.civ != 0 {
			continue
		}
		px := mmX + float32(u.x)*scaleX + scaleX/2
		py := mmY + float32(u.y)*scaleY + scaleY/2
		vector.DrawFilledCircle(screen, px, py, 1, cvCivColors[u.civ], false)
	}

	// Camera viewport shown as white rectangle
	vpX := mmX + float32(g.camX)/float32(cvMapW*cvTileS)*float32(cvMinimapW)
	vpY := mmY + float32(g.camY)/float32(cvMapH*cvTileS)*float32(cvMinimapH)
	vpW := float32(cvScreenW) / float32(cvMapW*cvTileS) * float32(cvMinimapW)
	vpH := float32(cvScreenH-cvTopBarH-cvBotBarH) / float32(cvMapH*cvTileS) * float32(cvMinimapH)
	vector.StrokeRect(screen, vpX, vpY, vpW, vpH, 1, color.RGBA{0xff, 0xff, 0xff, 0xc0}, false)
}

func (g *Civilization) drawTechTree(screen *ebiten.Image) {
	panelW := float32(380)
	panelH := float32(cvTechCount*35 + 70)

	// Dark overlay behind panel
	vector.DrawFilledRect(screen, 0, 0, float32(cvScreenW), float32(cvScreenH), color.RGBA{0, 0, 0, 0x80}, false)

	// Panel background
	vector.DrawFilledRect(screen, 50, 40, panelW, panelH, color.RGBA{0x0c, 0x0c, 0x18, 0xf0}, false)
	// Gold border
	vector.StrokeRect(screen, 50, 40, panelW, panelH, 2, color.RGBA{0xc0, 0xa0, 0x40, 0xff}, false)
	// Inner border
	vector.StrokeRect(screen, 52, 42, panelW-4, panelH-4, 0.5, color.RGBA{0x60, 0x50, 0x30, 0x80}, false)

	// Title
	ebitenutil.DebugPrintAt(screen, "TECHNOLOGY TREE (click to research)", 70, 50)
	// Title underline
	vector.DrawFilledRect(screen, 70, 62, 300, 1, color.RGBA{0xc0, 0xa0, 0x40, 0x80}, false)

	startY := 72
	techH := 35

	// Draw connection lines between techs
	for i := 1; i < cvTechCount; i++ {
		ty := startY + i*techH
		prevTy := startY + (i-1)*techH
		lineCol := color.RGBA{0x40, 0x40, 0x60, 0x80}
		if g.civs[0].techs[i-1] {
			lineCol = color.RGBA{0x40, 0x80, 0x40, 0xc0}
		}
		vector.StrokeLine(screen, 70, float32(prevTy+28), 70, float32(ty), 1, lineCol, false)
	}

	for i := 0; i < cvTechCount; i++ {
		ty := startY + i*techH
		tech := cvTechs[i]

		bgCol := color.RGBA{0x20, 0x20, 0x30, 0xff}
		borderCol := color.RGBA{0x40, 0x40, 0x60, 0xff}
		textSuffix := ""

		if g.civs[0].techs[i] {
			// Researched: green
			bgCol = color.RGBA{0x18, 0x50, 0x18, 0xff}
			borderCol = color.RGBA{0x40, 0xa0, 0x40, 0xff}
			textSuffix = " [DONE]"
		} else if g.civs[0].researchTarget == i {
			// Currently researching: bright blue
			bgCol = color.RGBA{0x20, 0x30, 0x60, 0xff}
			borderCol = color.RGBA{0x50, 0x70, 0xd0, 0xff}
			textSuffix = fmt.Sprintf(" [%d/%d]", g.civs[0].scienceAccum, tech.cost)
		} else {
			// Available or not: dimmed
			// Check if any prereq logic would dim it further
			borderCol = color.RGBA{0x50, 0x50, 0x70, 0xff}
		}

		// Hover highlight (check mouse position)
		mx, my := ebiten.CursorPosition()
		if mx >= 80 && mx <= 380 && my >= ty && my <= ty+28 && !g.civs[0].techs[i] {
			bgCol.R = uint8(clampInt(int(bgCol.R)+0x15, 0, 255))
			bgCol.G = uint8(clampInt(int(bgCol.G)+0x15, 0, 255))
			bgCol.B = uint8(clampInt(int(bgCol.B)+0x15, 0, 255))
		}

		vector.DrawFilledRect(screen, 80, float32(ty), 300, 28, bgCol, false)
		vector.StrokeRect(screen, 80, float32(ty), 300, 28, 1, borderCol, false)

		// Connection dot on left
		dotCol := color.RGBA{0x60, 0x60, 0x80, 0xff}
		if g.civs[0].techs[i] {
			dotCol = color.RGBA{0x40, 0xc0, 0x40, 0xff}
		}
		vector.DrawFilledCircle(screen, 70, float32(ty+14), 3, dotCol, false)

		// Cost display with affordability indicator
		affordable := tech.cost <= g.civs[0].scienceAccum || g.civs[0].techs[i]
		costIndicator := ""
		if affordable && !g.civs[0].techs[i] {
			costIndicator = " *"
		}

		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s (cost %d)%s%s", tech.name, tech.cost, textSuffix, costIndicator), 88, ty+8)
	}
}

func (g *Civilization) drawBuildMenu(screen *ebiten.Image) {
	if g.selectedCity < 0 || g.selectedCity >= len(g.cities) {
		return
	}
	c := g.cities[g.selectedCity]

	startX := cvScreenW - 190
	startY := 50
	_, cityProd, _ := g.cityYield(c)

	// Count items
	itemCount := 0
	for i := 0; i < 6; i++ {
		req := cvUnitDefs[i].reqTech
		if req >= 0 && !g.civs[0].techs[req] {
			continue
		}
		itemCount++
	}
	for i := 0; i < cvBuildCount; i++ {
		if c.buildings[i] {
			continue
		}
		req := cvBuildDefs[i].reqTech
		if req >= 0 && !g.civs[0].techs[req] {
			continue
		}
		itemCount++
	}

	panelH := float32(itemCount*24 + 40)
	// Panel background
	vector.DrawFilledRect(screen, float32(startX-8), float32(startY-15), 190, panelH, color.RGBA{0x0c, 0x0c, 0x18, 0xf0}, false)
	// Gold border
	vector.StrokeRect(screen, float32(startX-8), float32(startY-15), 190, panelH, 1.5, color.RGBA{0xc0, 0xa0, 0x40, 0xff}, false)
	vector.StrokeRect(screen, float32(startX-6), float32(startY-13), 186, panelH-4, 0.5, color.RGBA{0x60, 0x50, 0x30, 0x60}, false)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("BUILD - %s", c.name), startX, startY-8)
	vector.DrawFilledRect(screen, float32(startX), float32(startY+4), 170, 1, color.RGBA{0xc0, 0xa0, 0x40, 0x60}, false)

	mx, my := ebiten.CursorPosition()
	itemIdx := 0

	// Units
	for i := 0; i < 6; i++ {
		req := cvUnitDefs[i].reqTech
		if req >= 0 && !g.civs[0].techs[req] {
			continue
		}
		ty := startY + 12 + itemIdx*24
		bgCol := color.RGBA{0x20, 0x20, 0x30, 0xff}
		borderCol := color.RGBA{0x40, 0x40, 0x60, 0xff}
		if c.producing == i {
			bgCol = color.RGBA{0x30, 0x30, 0x70, 0xff}
			borderCol = color.RGBA{0x60, 0x60, 0xd0, 0xff}
		}
		// Hover highlight
		if mx >= startX && mx <= startX+170 && my >= ty && my <= ty+20 {
			bgCol.R = uint8(clampInt(int(bgCol.R)+0x18, 0, 255))
			bgCol.G = uint8(clampInt(int(bgCol.G)+0x18, 0, 255))
			bgCol.B = uint8(clampInt(int(bgCol.B)+0x18, 0, 255))
		}
		vector.DrawFilledRect(screen, float32(startX), float32(ty), 170, 20, bgCol, false)
		vector.StrokeRect(screen, float32(startX), float32(ty), 170, 20, 0.5, borderCol, false)

		// Cost in green/red based on turns to build
		costStr := fmt.Sprintf("%d", cvUnitDefs[i].cost)
		turnsNeeded := (cvUnitDefs[i].cost + cityProd - 1) / cityProd
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s  %s (%dt)", cvUnitDefs[i].name, costStr, turnsNeeded), startX+4, ty+4)
		itemIdx++
	}

	// Buildings separator
	if itemIdx > 0 {
		sepY := startY + 12 + itemIdx*24 - 4
		vector.DrawFilledRect(screen, float32(startX+10), float32(sepY), 150, 1, color.RGBA{0x60, 0x60, 0x80, 0x80}, false)
	}

	// Buildings
	for i := 0; i < cvBuildCount; i++ {
		if c.buildings[i] {
			continue
		}
		req := cvBuildDefs[i].reqTech
		if req >= 0 && !g.civs[0].techs[req] {
			continue
		}
		ty := startY + 12 + itemIdx*24
		bgCol := color.RGBA{0x20, 0x30, 0x20, 0xff}
		borderCol := color.RGBA{0x40, 0x60, 0x40, 0xff}
		if c.producing == 100+i {
			bgCol = color.RGBA{0x30, 0x60, 0x30, 0xff}
			borderCol = color.RGBA{0x50, 0xa0, 0x50, 0xff}
		}
		// Hover highlight
		if mx >= startX && mx <= startX+170 && my >= ty && my <= ty+20 {
			bgCol.R = uint8(clampInt(int(bgCol.R)+0x18, 0, 255))
			bgCol.G = uint8(clampInt(int(bgCol.G)+0x18, 0, 255))
			bgCol.B = uint8(clampInt(int(bgCol.B)+0x18, 0, 255))
		}
		vector.DrawFilledRect(screen, float32(startX), float32(ty), 170, 20, bgCol, false)
		vector.StrokeRect(screen, float32(startX), float32(ty), 170, 20, 0.5, borderCol, false)

		costStr := fmt.Sprintf("%d", cvBuildDefs[i].cost)
		turnsNeeded := (cvBuildDefs[i].cost + cityProd - 1) / cityProd
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s  %s (%dt)", cvBuildDefs[i].name, costStr, turnsNeeded), startX+4, ty+4)
		itemIdx++
	}
}

func (g *Civilization) drawVictoryScreen(screen *ebiten.Image) {
	// Full-screen dramatic overlay
	vector.DrawFilledRect(screen, 0, 0, float32(cvScreenW), float32(cvScreenH), color.RGBA{0x05, 0x05, 0x10, 0xd0}, false)

	// Particle effects - falling golden sparkles
	for i := 0; i < 30; i++ {
		seed := uint32(i*7919 + g.tick*131)
		px := float32(seed%uint32(cvScreenW))
		seed = seed*1103515245 + 12345
		baseY := float32(seed % uint32(cvScreenH))
		// Animate downward
		py := float32(math.Mod(float64(baseY)+float64(g.tick)*0.5+float64(i*17), float64(cvScreenH)))
		sparkAlpha := uint8(0x40 + (seed%0x80))
		sparkSize := float32(0.5 + float64(seed%3)*0.5)
		sparkCol := color.RGBA{0xff, 0xd7, 0x00, sparkAlpha}
		if g.winner != 0 {
			sparkCol = color.RGBA{0xff, 0x30, 0x30, sparkAlpha} // red particles for defeat
		}
		vector.DrawFilledCircle(screen, px, py, sparkSize, sparkCol, false)
	}

	// Central panel
	panelX := float32(80)
	panelY := float32(120)
	panelW := float32(480)
	panelH := float32(240)
	vector.DrawFilledRect(screen, panelX, panelY, panelW, panelH, color.RGBA{0x10, 0x10, 0x20, 0xf8}, false)

	// Double gold border
	borderCol := color.RGBA{0xff, 0xd7, 0x00, 0xff}
	if g.winner != 0 {
		borderCol = color.RGBA{0xff, 0x40, 0x40, 0xff}
	}
	vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 3, borderCol, false)
	vector.StrokeRect(screen, panelX+4, panelY+4, panelW-8, panelH-8, 1, color.RGBA{borderCol.R, borderCol.G, borderCol.B, 0x80}, false)

	// Corner decorations
	for _, corner := range [][2]float32{{panelX, panelY}, {panelX + panelW, panelY}, {panelX, panelY + panelH}, {panelX + panelW, panelY + panelH}} {
		vector.DrawFilledCircle(screen, corner[0], corner[1], 5, borderCol, false)
	}

	winnerName := cvCivNames[g.winner]
	if g.winner == 0 {
		// Victory text with glow
		ebitenutil.DebugPrintAt(screen, "=== VICTORY ===", 250, int(panelY)+20)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("You achieved a %s victory!", g.victoryType), 180, int(panelY)+50)
	} else {
		ebitenutil.DebugPrintAt(screen, "=== DEFEAT ===", 255, int(panelY)+20)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s achieved a %s victory!", winnerName, g.victoryType), 140, int(panelY)+50)
	}

	// Divider line
	vector.DrawFilledRect(screen, panelX+30, panelY+70, panelW-60, 1, color.RGBA{0xc0, 0xa0, 0x40, 0x80}, false)

	// Stats
	statsX := 200
	statsY := int(panelY) + 85
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Turn:   %d", g.turn), statsX, statsY)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Cities: %d", g.countCities(0)), statsX, statsY+20)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Units:  %d", g.countUnits(0)), statsX, statsY+40)

	techCount := 0
	for i := 0; i < cvTechCount; i++ {
		if g.civs[0].techs[i] {
			techCount++
		}
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Techs:  %d/%d", techCount, cvTechCount), statsX, statsY+60)

	// Score calculation
	score := g.turn*-1 + g.countCities(0)*100 + g.countUnits(0)*20 + techCount*50
	if g.winner == 0 {
		score += 500
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score:  %d", score), statsX+180, statsY+20)

	// Divider
	vector.DrawFilledRect(screen, panelX+30, panelY+170, panelW-60, 1, color.RGBA{0xc0, 0xa0, 0x40, 0x80}, false)

	// Return prompt with pulsing
	promptAlpha := uint8(0x80 + 0x7f*float64(math.Sin(float64(g.tick)*0.08)))
	_ = promptAlpha
	ebitenutil.DebugPrintAt(screen, "Press ESC to return to menu", 220, int(panelY)+185)
}

// ── Notification drawing ───────────────────────────────────────────────────

func (g *Civilization) drawNotifications(screen *ebiten.Image) {
	count := len(g.notifications)
	if count == 0 {
		return
	}
	// Show last 4 notifications stacked in bottom-left
	startIdx := 0
	if count > 4 {
		startIdx = count - 4
	}
	baseY := cvScreenH - cvBotBarH - 12
	for i := startIdx; i < count; i++ {
		n := g.notifications[i]
		slot := i - startIdx
		ny := baseY - (count-startIdx-1-slot)*14

		alpha := uint8(0xff)
		if n.timer < 40 {
			alpha = uint8(float64(n.timer) / 40.0 * 255)
		}

		// Background
		textW := float32(len(n.text)*6 + 12)
		vector.DrawFilledRect(screen, 4, float32(ny)-2, textW, 14, color.RGBA{0x08, 0x08, 0x14, alpha / 2}, false)

		// Colored left bar
		barCol := n.col
		barCol.A = alpha
		vector.DrawFilledRect(screen, 4, float32(ny)-2, 2, 14, barCol, false)

		_ = alpha
		ebitenutil.DebugPrintAt(screen, n.text, 10, ny)
	}
}

// ── Tutorial overlay ──────────────────────────────────────────────────────

func (g *Civilization) drawTutorial(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(cvScreenW), float32(cvScreenH), color.RGBA{0, 0, 0, 0x60}, false)

	// Tutorial panel
	panelW := float32(380)
	panelH := float32(180)
	panelX := float32(cvScreenW)/2 - panelW/2
	panelY := float32(cvScreenH)/2 - panelH/2

	vector.DrawFilledRect(screen, panelX, panelY, panelW, panelH, color.RGBA{0x10, 0x10, 0x28, 0xf8}, false)
	vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 2, color.RGBA{0xc0, 0xa0, 0x40, 0xff}, false)

	lines := []string{
		"WELCOME TO CIVILIZATION",
		"",
		"Click a unit to select it, then click a",
		"green tile to move. Red tiles are enemies.",
		"",
		"Your Settler can found cities: select it",
		"and press F. Your Warrior explores.",
		"",
		"[TAB] Next unit   [T] Tech tree",
		"[B] Build menu    [SPACE] End turn",
		"[X] Auto-explore  Right-click: deselect",
		"",
		"Click anywhere to begin!",
	}

	for i, line := range lines {
		ebitenutil.DebugPrintAt(screen, line, int(panelX)+20, int(panelY)+12+i*13)
	}
}

// ── Contextual hints ──────────────────────────────────────────────────────

func (g *Civilization) drawContextualHints(screen *ebiten.Image) {
	if g.gameOver || g.showTechTree || g.showBuild {
		return
	}
	if g.turn == 1 && !g.tutorialDismiss {
		return
	}

	// Show context-sensitive hints near the center-top
	hint := ""
	if g.selectedUnit >= 0 && g.selectedUnit < len(g.units) {
		u := g.units[g.selectedUnit]
		if !u.dead && u.civ == 0 {
			if u.unitType == cvUnitSettler {
				hint = "Click green tile to move | [F] Found City | [TAB] Next Unit"
			} else if u.movesLeft > 0 {
				hint = "Click green tile to move | Click red to attack | [X] Auto-Explore | [TAB] Next"
			} else {
				hint = "No moves left | [TAB] Next Unit | [SPACE] End Turn"
			}
		}
	} else if g.selectedCity >= 0 {
		hint = "[B] Build Menu | Click map to deselect | [SPACE] End Turn"
	} else {
		unmoved := g.countUnmovedPlayerUnits()
		if unmoved > 0 {
			hint = fmt.Sprintf("[TAB] Select next unit (%d unmoved) | [SPACE] End Turn", unmoved)
		} else {
			hint = "[SPACE] End Turn | Click a unit or city to select"
		}
	}

	if hint != "" {
		hintW := float32(len(hint)*6 + 20)
		hintX := float32(cvScreenW)/2 - hintW/2
		hintY := float32(cvTopBarH + 4)
		vector.DrawFilledRect(screen, hintX, hintY, hintW, 14, color.RGBA{0x08, 0x08, 0x18, 0xc0}, false)
		ebitenutil.DebugPrintAt(screen, hint, int(hintX)+10, int(hintY)+2)
	}
}

// ── Layout and WantsBack ───────────────────────────────────────────────────

func (g *Civilization) Layout(w, h int) (int, int) {
	return cvScreenW, cvScreenH
}

func (g *Civilization) WantsBack() bool {
	return g.goBack
}
