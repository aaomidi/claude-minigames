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

// ─── Constants ───

const (
	frScreenW = 640
	frScreenH = 480
	frTileW   = 28
	frTileH   = 28
	frMapW    = 24 // tiles wide
	frMapH    = 20 // tiles tall
	frHotbarSlots = 8
	frMaxEnergy   = 100
	frDaysPerSeason = 28
)

// ─── Seasons ───

const (
	frSpring = iota
	frSummer
	frFall
	frWinter
)

var frSeasonNames = []string{"Spring", "Summer", "Fall", "Winter"}

// ─── Time of day phases ───

const (
	frMorning = iota
	frAfternoon
	frEvening
	frNight
)

var frPhaseNames = []string{"Morning", "Afternoon", "Evening", "Night"}

// ─── Tile types ───

const (
	frTileGrass   = iota
	frTileDirt          // tilled
	frTileWatered       // watered tilled
	frTilePlanted       // has a crop
	frTileRock
	frTileTree
	frTileFence
	frTilePath
	frTileWater
	frTileBuilding
	frTileShipBin
	frTileShop
	frTileDoor
)

// ─── Direction ───

const (
	frDirDown = iota
	frDirUp
	frDirLeft
	frDirRight
)

// ─── Crop definitions ───

type frCropDef struct {
	name     string
	season   int
	growDays int
	sellVal  int
	regrows  bool
	seedCost int
}

var frCrops = []frCropDef{
	{"Turnip", frSpring, 4, 40, false, 20},
	{"Potato", frSpring, 6, 80, false, 40},
	{"Tomato", frSummer, 8, 120, true, 60},
	{"Corn", frSummer, 10, 150, true, 80},
	{"Pumpkin", frFall, 12, 300, false, 150},
	{"Starfruit", frSummer, 14, 800, false, 400},
}

// ─── Item types ───

const (
	frItemNone = iota
	frItemHoe
	frItemWateringCan
	frItemFishingRod
	frItemTurnipSeed
	frItemPotatoSeed
	frItemTomatoSeed
	frItemCornSeed
	frItemPumpkinSeed
	frItemStarfruitSeed
	frItemTurnip
	frItemPotato
	frItemTomato
	frItemCorn
	frItemPumpkin
	frItemStarfruit
	frItemFishSmall
	frItemFishMed
	frItemFishLarge
)

func frItemName(id int) string {
	switch id {
	case frItemHoe:
		return "Hoe"
	case frItemWateringCan:
		return "Water Can"
	case frItemFishingRod:
		return "Rod"
	case frItemTurnipSeed:
		return "TurnipSd"
	case frItemPotatoSeed:
		return "PotatoSd"
	case frItemTomatoSeed:
		return "TomatoSd"
	case frItemCornSeed:
		return "CornSd"
	case frItemPumpkinSeed:
		return "PumpknSd"
	case frItemStarfruitSeed:
		return "StarfSd"
	case frItemTurnip:
		return "Turnip"
	case frItemPotato:
		return "Potato"
	case frItemTomato:
		return "Tomato"
	case frItemCorn:
		return "Corn"
	case frItemPumpkin:
		return "Pumpkin"
	case frItemStarfruit:
		return "Starfrt"
	case frItemFishSmall:
		return "SmFish"
	case frItemFishMed:
		return "MdFish"
	case frItemFishLarge:
		return "LgFish"
	}
	return ""
}

func frItemSellPrice(id int) int {
	switch id {
	case frItemTurnip:
		return 40
	case frItemPotato:
		return 80
	case frItemTomato:
		return 120
	case frItemCorn:
		return 150
	case frItemPumpkin:
		return 300
	case frItemStarfruit:
		return 800
	case frItemFishSmall:
		return 20
	case frItemFishMed:
		return 80
	case frItemFishLarge:
		return 200
	}
	return 0
}

func frItemColor(id int) color.RGBA {
	switch id {
	case frItemHoe:
		return color.RGBA{0x8B, 0x45, 0x13, 0xff}
	case frItemWateringCan:
		return color.RGBA{0x40, 0x80, 0xc0, 0xff}
	case frItemFishingRod:
		return color.RGBA{0xa0, 0x70, 0x40, 0xff}
	case frItemTurnipSeed, frItemPotatoSeed, frItemTomatoSeed, frItemCornSeed, frItemPumpkinSeed, frItemStarfruitSeed:
		return color.RGBA{0x80, 0xa0, 0x30, 0xff}
	case frItemTurnip:
		return color.RGBA{0xff, 0xff, 0xff, 0xff}
	case frItemPotato:
		return color.RGBA{0xc0, 0xa0, 0x60, 0xff}
	case frItemTomato:
		return color.RGBA{0xff, 0x30, 0x30, 0xff}
	case frItemCorn:
		return color.RGBA{0xff, 0xe0, 0x40, 0xff}
	case frItemPumpkin:
		return color.RGBA{0xff, 0x80, 0x00, 0xff}
	case frItemStarfruit:
		return color.RGBA{0xff, 0xff, 0x00, 0xff}
	case frItemFishSmall:
		return color.RGBA{0x60, 0x90, 0xc0, 0xff}
	case frItemFishMed:
		return color.RGBA{0x40, 0x70, 0xb0, 0xff}
	case frItemFishLarge:
		return color.RGBA{0x20, 0x50, 0xa0, 0xff}
	}
	return color.RGBA{0x80, 0x80, 0x80, 0xff}
}

func frSeedToCropIdx(seedItem int) int {
	switch seedItem {
	case frItemTurnipSeed:
		return 0
	case frItemPotatoSeed:
		return 1
	case frItemTomatoSeed:
		return 2
	case frItemCornSeed:
		return 3
	case frItemPumpkinSeed:
		return 4
	case frItemStarfruitSeed:
		return 5
	}
	return -1
}

func frCropIdxToHarvest(idx int) int {
	switch idx {
	case 0:
		return frItemTurnip
	case 1:
		return frItemPotato
	case 2:
		return frItemTomato
	case 3:
		return frItemCorn
	case 4:
		return frItemPumpkin
	case 5:
		return frItemStarfruit
	}
	return frItemNone
}

func frCropIdxToSeed(idx int) int {
	return frItemTurnipSeed + idx
}

// ─── Crop instance on the map ───

type frCropInst struct {
	cropIdx   int
	dayPlanted int
	growDay   int // days of growth accumulated
	watered   bool
	dead      bool
}

// ─── Inventory slot ───

type frInvSlot struct {
	item  int
	count int
}

// ─── NPC ───

type frNPC struct {
	name      string
	x, y      float64
	pathIdx   int
	path      [][2]int
	speed     float64
	dialogues []string
	dlgIdx    int
	hearts    int
	talkCD    int
}

// ─── Fishing state ───

const (
	frFishIdle = iota
	frFishCasting
	frFishReeling
	frFishCaught
	frFishMiss
)

// ─── Shop state ───

const (
	frShopClosed = iota
	frShopOpen
)

// ─── Message popup ───

type frMessage struct {
	text   string
	frames int
}

// ─── The main game struct ───

type FarmRPG struct {
	goBack bool

	// Map
	tiles    [frMapW][frMapH]int
	crops    [frMapW][frMapH]*frCropInst

	// Player
	px, py     float64 // pixel position
	pdir       int
	moveCD     int

	// Camera
	camX, camY float64

	// Time
	day     int
	season  int
	year    int
	phase   int // time of day
	raining bool

	// Economy
	money int

	// Energy
	energy int

	// Inventory
	hotbar   [frHotbarSlots]frInvSlot
	selected int

	// NPCs
	npcs []*frNPC

	// Fishing
	fishState  int
	fishBar    float64 // 0.0 - 1.0
	fishDir    int     // 1 or -1
	fishGreen  float64 // center of green zone
	fishGreenW float64
	fishTimer  int

	// Shop
	shopState  int
	shopCursor int

	// Messages
	messages []frMessage

	// Stats
	totalSold   int
	cropsGrown  int
	fishCaught  int

	// Audio
	bgmPlayer *audio.Player
	sfxTill   []byte
	sfxWater  []byte
	sfxHarvest []byte
	sfxSell   []byte
	sfxFish   []byte

	// Tick
	tick int
}

// ─── Constructor ───

func newFarmRPG() *FarmRPG {
	g := &FarmRPG{
		money:    500,
		energy:   frMaxEnergy,
		day:      1,
		season:   frSpring,
		year:     1,
		phase:    frMorning,
		selected: 0,
	}

	g.initMap()
	g.initPlayer()
	g.initInventory()
	g.initNPCs()
	g.initAudio()

	return g
}

// ─── Map initialization ───

func (g *FarmRPG) initMap() {
	// Fill with grass
	for x := 0; x < frMapW; x++ {
		for y := 0; y < frMapH; y++ {
			g.tiles[x][y] = frTileGrass
		}
	}

	// Fence border
	for x := 0; x < frMapW; x++ {
		g.tiles[x][0] = frTileFence
		g.tiles[x][frMapH-1] = frTileFence
	}
	for y := 0; y < frMapH; y++ {
		g.tiles[0][y] = frTileFence
		g.tiles[frMapW-1][y] = frTileFence
	}

	// House (top-left, 3x3)
	for x := 1; x <= 3; x++ {
		for y := 1; y <= 3; y++ {
			g.tiles[x][y] = frTileBuilding
		}
	}
	g.tiles[2][3] = frTileDoor

	// Shop (top-right, 2x3)
	for x := frMapW - 4; x <= frMapW-3; x++ {
		for y := 1; y <= 3; y++ {
			g.tiles[x][y] = frTileShop
		}
	}
	g.tiles[frMapW-4][3] = frTileDoor

	// Shipping bin (left edge)
	g.tiles[1][8] = frTileShipBin

	// Pond/water (right edge, 5x4)
	for x := frMapW - 6; x <= frMapW-2; x++ {
		for y := 8; y <= 12; y++ {
			g.tiles[x][y] = frTileWater
		}
	}

	// Paths
	for x := 2; x < frMapW-2; x++ {
		g.tiles[x][5] = frTilePath
	}
	for y := 3; y < frMapH-1; y++ {
		g.tiles[5][y] = frTilePath
	}
	for y := 3; y <= 8; y++ {
		g.tiles[1][y] = frTilePath
	}

	// Scattered rocks and trees
	rockPositions := [][2]int{{8, 2}, {12, 3}, {15, 2}, {7, 14}, {10, 16}, {3, 15}}
	for _, rp := range rockPositions {
		if g.tiles[rp[0]][rp[1]] == frTileGrass {
			g.tiles[rp[0]][rp[1]] = frTileRock
		}
	}
	treePositions := [][2]int{{3, 6}, {3, 10}, {14, 4}, {16, 15}, {11, 17}, {8, 18}}
	for _, tp := range treePositions {
		if g.tiles[tp[0]][tp[1]] == frTileGrass {
			g.tiles[tp[0]][tp[1]] = frTileTree
		}
	}
}

func (g *FarmRPG) initPlayer() {
	g.px = 3 * frTileW
	g.py = 4 * frTileW
	g.pdir = frDirDown
}

func (g *FarmRPG) initInventory() {
	g.hotbar[0] = frInvSlot{frItemHoe, 1}
	g.hotbar[1] = frInvSlot{frItemWateringCan, 1}
	g.hotbar[2] = frInvSlot{frItemFishingRod, 1}
	g.hotbar[3] = frInvSlot{frItemTurnipSeed, 12}
}

func (g *FarmRPG) initNPCs() {
	g.npcs = []*frNPC{
		{
			name: "Lily",
			x: 10 * frTileW, y: 5 * frTileW,
			path: [][2]int{{10, 5}, {14, 5}, {14, 7}, {10, 7}, {10, 5}},
			speed: 0.5,
			dialogues: []string{
				"Beautiful day, isn't it?",
				"I love the smell of fresh soil!",
				"Have you tried growing Starfruit?",
			},
		},
		{
			name: "Gus",
			x: 7 * frTileW, y: 12 * frTileW,
			path: [][2]int{{7, 12}, {7, 16}, {12, 16}, {12, 12}, {7, 12}},
			speed: 0.4,
			dialogues: []string{
				"The fish are biting today!",
				"I heard the big ones come out at night.",
				"Try fishing when it rains for rare catches!",
			},
		},
		{
			name: "Mae",
			x: float64(frMapW-5) * frTileW, y: 5 * frTileW,
			path: [][2]int{{frMapW - 5, 5}, {frMapW - 5, 7}, {frMapW - 7, 7}, {frMapW - 7, 5}, {frMapW - 5, 5}},
			speed: 0.3,
			dialogues: []string{
				"Welcome to the shop!",
				"We just got Starfruit seeds in stock!",
				"Plant crops in the right season or they die!",
			},
		},
	}
}

func (g *FarmRPG) initAudio() {
	// BGM: Peaceful pastoral melody
	melody := [][2]int{
		// Phrase 1: gentle ascending motif in C major
		{60, 2}, {64, 2}, {67, 2}, {72, 2},
		{71, 1}, {69, 1}, {67, 2}, {0, 2},
		{65, 2}, {67, 2}, {69, 2}, {67, 2},
		{64, 2}, {60, 2}, {0, 2}, {0, 2},
		// Phrase 2: warm counter-melody
		{67, 2}, {69, 2}, {71, 2}, {72, 2},
		{74, 1}, {72, 1}, {71, 2}, {69, 2},
		{67, 2}, {65, 2}, {64, 2}, {0, 2},
		{60, 2}, {62, 2}, {64, 4},
		// Phrase 3: rising hope
		{65, 2}, {67, 2}, {69, 1}, {71, 1}, {72, 2}, {74, 2},
		{72, 2}, {69, 2}, {67, 4},
		{65, 2}, {64, 2}, {62, 2}, {60, 2},
		{0, 2}, {0, 2},
		// Phrase 4: peaceful resolution
		{64, 2}, {67, 2}, {72, 3}, {71, 1},
		{69, 2}, {67, 2}, {65, 2}, {64, 2},
		{62, 2}, {64, 2}, {65, 2}, {67, 2},
		{69, 2}, {67, 2}, {64, 2}, {60, 4},
		// Phrase 5: gentle variation
		{72, 2}, {71, 1}, {72, 1}, {74, 2}, {72, 2},
		{69, 2}, {67, 2}, {65, 2}, {64, 2},
		{62, 2}, {60, 2}, {0, 2}, {0, 2},
		{60, 2}, {64, 2}, {67, 2}, {72, 4},
	}

	harm := [][2]int{
		// Sustained chords underneath
		{48, 8}, {48, 8}, {45, 8}, {48, 8},
		{43, 8}, {45, 8}, {48, 8}, {48, 4}, {50, 4},
		{48, 8}, {45, 8}, {43, 8}, {48, 8},
		{48, 8}, {45, 8}, {43, 8}, {48, 8},
		{45, 8}, {43, 8}, {48, 8}, {48, 4}, {0, 4},
	}

	melPCM := generateTrack(melody, sineWave, 80, 0.25)
	harmPCM := generateTrack(harm, triangleWave, 80, 0.12)

	// Pad harmony to melody length
	if len(harmPCM) < len(melPCM) {
		extended := make([]byte, len(melPCM))
		for i := 0; i < len(melPCM); i++ {
			extended[i] = harmPCM[i%len(harmPCM)]
		}
		harmPCM = extended
	}
	if len(melPCM) < len(harmPCM) {
		harmPCM = harmPCM[:len(melPCM)]
	}

	bgm := mixBuffers(melPCM, harmPCM)
	g.bgmPlayer = loopPlayer(bgm, 0.3)
	g.bgmPlayer.Play()

	// SFX
	g.sfxTill = g.genSFXTill()
	g.sfxWater = g.genSFXWater()
	g.sfxHarvest = g.genSFXHarvest()
	g.sfxSell = g.genSFXSell()
	g.sfxFish = g.genSFXFish()
}

func (g *FarmRPG) genSFXTill() []byte {
	buf := pcmStereo(0.15)
	for i := 0; i < len(buf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		env := math.Exp(-t * 20)
		val := noise() * env * 0.4
		writeSample(buf, i*4, val)
	}
	return buf
}

func (g *FarmRPG) genSFXWater() []byte {
	buf := pcmStereo(0.2)
	for i := 0; i < len(buf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		env := math.Exp(-t * 10)
		val := (noise()*0.3 + sineWave(t, 400+noise()*200)*0.4) * env * 0.35
		writeSample(buf, i*4, val)
	}
	return buf
}

func (g *FarmRPG) genSFXHarvest() []byte {
	buf := pcmStereo(0.12)
	for i := 0; i < len(buf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		env := math.Exp(-t * 25)
		val := sineWave(t, 600+t*2000) * env * 0.4
		writeSample(buf, i*4, val)
	}
	return buf
}

func (g *FarmRPG) genSFXSell() []byte {
	buf := pcmStereo(0.25)
	for i := 0; i < len(buf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		env := math.Exp(-t * 8)
		val := (sineWave(t, 800)*0.3 + sineWave(t, 1200)*0.2 + sineWave(t, 1600)*0.15) * env * 0.35
		writeSample(buf, i*4, val)
	}
	return buf
}

func (g *FarmRPG) genSFXFish() []byte {
	buf := pcmStereo(0.08)
	for i := 0; i < len(buf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		env := math.Exp(-t * 30)
		val := sineWave(t, 1000) * env * 0.4
		writeSample(buf, i*4, val)
	}
	return buf
}

// ─── Helpers ───

func (g *FarmRPG) playerTile() (int, int) {
	return int((g.px + float64(frTileW)/2) / float64(frTileW)),
		int((g.py + float64(frTileH)/2) / float64(frTileH))
}

func (g *FarmRPG) facingTile() (int, int) {
	px, py := g.playerTile()
	switch g.pdir {
	case frDirUp:
		py--
	case frDirDown:
		py++
	case frDirLeft:
		px--
	case frDirRight:
		px++
	}
	return px, py
}

func (g *FarmRPG) inBounds(x, y int) bool {
	return x >= 0 && x < frMapW && y >= 0 && y < frMapH
}

func (g *FarmRPG) canWalk(x, y int) bool {
	if !g.inBounds(x, y) {
		return false
	}
	t := g.tiles[x][y]
	return t != frTileFence && t != frTileBuilding && t != frTileWater &&
		t != frTileRock && t != frTileTree && t != frTileShop
}

func (g *FarmRPG) addMessage(text string) {
	g.messages = append(g.messages, frMessage{text, 120})
}

func (g *FarmRPG) addItem(item, count int) bool {
	// Try to stack first
	for i := 0; i < frHotbarSlots; i++ {
		if g.hotbar[i].item == item {
			g.hotbar[i].count += count
			return true
		}
	}
	// Find empty slot
	for i := 0; i < frHotbarSlots; i++ {
		if g.hotbar[i].item == frItemNone {
			g.hotbar[i] = frInvSlot{item, count}
			return true
		}
	}
	g.addMessage("Inventory full!")
	return false
}

func (g *FarmRPG) removeItem(slot int, count int) {
	if slot < 0 || slot >= frHotbarSlots {
		return
	}
	g.hotbar[slot].count -= count
	if g.hotbar[slot].count <= 0 {
		g.hotbar[slot] = frInvSlot{frItemNone, 0}
	}
}

func (g *FarmRPG) hasItem(item int) (int, bool) {
	for i := 0; i < frHotbarSlots; i++ {
		if g.hotbar[i].item == item && g.hotbar[i].count > 0 {
			return i, true
		}
	}
	return -1, false
}

func (g *FarmRPG) isAdjacentTo(tileType int) bool {
	px, py := g.playerTile()
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := px+dx, py+dy
			if g.inBounds(nx, ny) && g.tiles[nx][ny] == tileType {
				return true
			}
		}
	}
	return false
}

func (g *FarmRPG) totalDays() int {
	return (g.year-1)*4*frDaysPerSeason + g.season*frDaysPerSeason + g.day
}

// ─── Update ───

func (g *FarmRPG) Update() error {
	g.tick++

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.shopState == frShopOpen {
			g.shopState = frShopClosed
		} else if g.fishState != frFishIdle {
			g.fishState = frFishIdle
		} else {
			if g.bgmPlayer != nil {
				g.bgmPlayer.Pause()
			}
			g.goBack = true
		}
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if g.bgmPlayer != nil {
			g.bgmPlayer.Pause()
		}
		*g = *newFarmRPG()
		return nil
	}

	// Update messages
	for i := len(g.messages) - 1; i >= 0; i-- {
		g.messages[i].frames--
		if g.messages[i].frames <= 0 {
			g.messages = append(g.messages[:i], g.messages[i+1:]...)
		}
	}

	// Shop open? Handle shop
	if g.shopState == frShopOpen {
		g.updateShop()
		return nil
	}

	// Fishing active? Handle fishing
	if g.fishState != frFishIdle {
		g.updateFishing()
		return nil
	}

	// Player movement
	g.updateMovement()

	// Hotbar selection
	for i := 0; i < frHotbarSlots; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			g.selected = i
		}
	}

	// Actions
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		g.actionTill()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.actionPlant()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.actionWaterKey()
	}
	// W key for watering when not moving
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		g.actionWaterKey()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.actionHarvest()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.actionSell()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.actionBuy()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		g.actionFish()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.actionTalk()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.advanceDay()
	}

	// NPC updates
	g.updateNPCs()

	// Camera follow
	g.updateCamera()

	return nil
}

func (g *FarmRPG) updateMovement() {
	if g.moveCD > 0 {
		g.moveCD--
		return
	}

	dx, dy := 0.0, 0.0
	speed := 2.0

	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		dy = -speed
		g.pdir = frDirUp
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		// Only move down if S isn't for selling (S moves AND sells contextually)
		// Actually we use S for sell only on press, movement uses arrows + WASD
		dy = speed
		g.pdir = frDirDown
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dx = -speed
		g.pdir = frDirLeft
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dx = speed
		g.pdir = frDirRight
	}

	// Normalize diagonal
	if dx != 0 && dy != 0 {
		dx *= 0.707
		dy *= 0.707
	}

	// Try X
	newPX := g.px + dx
	ntx := int((newPX + float64(frTileW)/2) / float64(frTileW))
	nty := int((g.py + float64(frTileH)/2) / float64(frTileH))
	if g.canWalk(ntx, nty) {
		g.px = newPX
	}

	// Try Y
	newPY := g.py + dy
	ntx = int((g.px + float64(frTileW)/2) / float64(frTileW))
	nty = int((newPY + float64(frTileH)/2) / float64(frTileH))
	if g.canWalk(ntx, nty) {
		g.py = newPY
	}

	// Clamp
	if g.px < float64(frTileW) {
		g.px = float64(frTileW)
	}
	if g.py < float64(frTileH) {
		g.py = float64(frTileH)
	}
	maxPX := float64((frMapW - 2) * frTileW)
	maxPY := float64((frMapH - 2) * frTileH)
	if g.px > maxPX {
		g.px = maxPX
	}
	if g.py > maxPY {
		g.py = maxPY
	}
}

func (g *FarmRPG) updateCamera() {
	targetX := g.px - float64(frScreenW)/2 + float64(frTileW)/2
	targetY := g.py - float64(frScreenH)/2 + float64(frTileH)/2 + 20

	g.camX += (targetX - g.camX) * 0.1
	g.camY += (targetY - g.camY) * 0.1

	// Clamp camera
	maxCamX := float64(frMapW*frTileW) - float64(frScreenW)
	maxCamY := float64(frMapH*frTileH) - float64(frScreenH) + 60
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
}

// ─── Actions ───

func (g *FarmRPG) actionTill() {
	if g.energy < 5 {
		g.addMessage("Too tired to till!")
		return
	}
	fx, fy := g.facingTile()
	if !g.inBounds(fx, fy) {
		return
	}
	if g.tiles[fx][fy] == frTileGrass {
		g.tiles[fx][fy] = frTileDirt
		g.energy -= 5
		playSFX(g.sfxTill, 0.5)
		g.addMessage("Tilled the soil!")
	}
}

func (g *FarmRPG) actionPlant() {
	fx, fy := g.facingTile()
	if !g.inBounds(fx, fy) {
		return
	}
	if g.tiles[fx][fy] != frTileDirt && g.tiles[fx][fy] != frTileWatered {
		g.addMessage("Need tilled soil to plant!")
		return
	}

	// Find seed in selected slot or any slot
	slot := g.selected
	seedItem := g.hotbar[slot].item
	cropIdx := frSeedToCropIdx(seedItem)
	if cropIdx < 0 {
		// Search all slots for a seed
		for i := 0; i < frHotbarSlots; i++ {
			ci := frSeedToCropIdx(g.hotbar[i].item)
			if ci >= 0 && g.hotbar[i].count > 0 {
				slot = i
				seedItem = g.hotbar[i].item
				cropIdx = ci
				break
			}
		}
	}
	if cropIdx < 0 {
		g.addMessage("No seeds to plant!")
		return
	}
	if g.hotbar[slot].count <= 0 {
		g.addMessage("No seeds left!")
		return
	}

	// Check season
	crop := frCrops[cropIdx]
	if crop.season != g.season {
		g.addMessage(fmt.Sprintf("%s is a %s crop!", crop.name, frSeasonNames[crop.season]))
		return
	}

	g.crops[fx][fy] = &frCropInst{
		cropIdx:    cropIdx,
		dayPlanted: g.totalDays(),
		growDay:    0,
		watered:    g.tiles[fx][fy] == frTileWatered,
	}
	g.tiles[fx][fy] = frTilePlanted
	g.removeItem(slot, 1)
	g.addMessage(fmt.Sprintf("Planted %s!", crop.name))
}

func (g *FarmRPG) actionWaterKey() {
	if g.energy < 3 {
		g.addMessage("Too tired to water!")
		return
	}
	fx, fy := g.facingTile()
	if !g.inBounds(fx, fy) {
		return
	}
	tile := g.tiles[fx][fy]
	if tile == frTileDirt {
		g.tiles[fx][fy] = frTileWatered
		g.energy -= 3
		playSFX(g.sfxWater, 0.5)
		g.addMessage("Watered the soil!")
	} else if tile == frTilePlanted {
		if c := g.crops[fx][fy]; c != nil {
			c.watered = true
			g.energy -= 3
			playSFX(g.sfxWater, 0.5)
			g.addMessage("Watered the crop!")
		}
	}
}

func (g *FarmRPG) actionHarvest() {
	if g.energy < 2 {
		g.addMessage("Too tired to harvest!")
		return
	}
	fx, fy := g.facingTile()
	if !g.inBounds(fx, fy) {
		return
	}
	if g.tiles[fx][fy] != frTilePlanted {
		return
	}
	c := g.crops[fx][fy]
	if c == nil {
		return
	}
	crop := frCrops[c.cropIdx]
	if c.dead {
		g.tiles[fx][fy] = frTileDirt
		g.crops[fx][fy] = nil
		g.addMessage("Cleared dead crop.")
		return
	}
	if c.growDay < crop.growDays {
		g.addMessage(fmt.Sprintf("%s needs %d more days.", crop.name, crop.growDays-c.growDay))
		return
	}

	harvestItem := frCropIdxToHarvest(c.cropIdx)
	if g.addItem(harvestItem, 1) {
		g.energy -= 2
		g.cropsGrown++
		playSFX(g.sfxHarvest, 0.5)
		g.addMessage(fmt.Sprintf("Harvested %s!", crop.name))

		if crop.regrows {
			c.growDay = crop.growDays / 2 // regrow from halfway
			c.watered = false
		} else {
			g.tiles[fx][fy] = frTileDirt
			g.crops[fx][fy] = nil
		}
	}
}

func (g *FarmRPG) actionSell() {
	nearBin := g.isAdjacentTo(frTileShipBin)
	if !nearBin {
		px, py := g.playerTile()
		if g.inBounds(px, py) && g.tiles[px][py] == frTileShipBin {
			nearBin = true
		}
	}
	if !nearBin {
		return // silently fail - S is also move-down
	}

	slot := g.selected
	item := g.hotbar[slot].item
	price := frItemSellPrice(item)
	if price <= 0 {
		g.addMessage("Can't sell that!")
		return
	}

	count := g.hotbar[slot].count
	total := price * count
	g.money += total
	g.totalSold += total
	g.removeItem(slot, count)
	playSFX(g.sfxSell, 0.5)
	g.addMessage(fmt.Sprintf("Sold %d %s for $%d!", count, frItemName(item), total))
}

func (g *FarmRPG) actionBuy() {
	// Check if near shop
	px, py := g.playerTile()
	nearShop := false
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			nx, ny := px+dx, py+dy
			if g.inBounds(nx, ny) && (g.tiles[nx][ny] == frTileShop || g.tiles[nx][ny] == frTileDoor) {
				nearShop = true
			}
		}
	}
	if !nearShop {
		g.addMessage("Stand near the shop!")
		return
	}
	g.shopState = frShopOpen
	g.shopCursor = 0
}

func (g *FarmRPG) actionFish() {
	if !g.isAdjacentTo(frTileWater) {
		g.addMessage("Stand near water to fish!")
		return
	}
	if _, ok := g.hasItem(frItemFishingRod); !ok {
		g.addMessage("You need a fishing rod!")
		return
	}
	g.fishState = frFishCasting
	g.fishBar = 0
	g.fishDir = 1
	g.fishGreen = 0.3 + rand.Float64()*0.4
	g.fishGreenW = 0.15 + rand.Float64()*0.1
	if g.raining {
		g.fishGreenW += 0.05 // Easier when raining
	}
	if g.phase == frNight {
		g.fishGreenW -= 0.03 // Harder at night, but better rewards
	}
	g.fishTimer = 180
	g.addMessage("Casting line... Press SPACE in the green zone!")
}

func (g *FarmRPG) actionTalk() {
	px, py := g.playerTile()
	for _, npc := range g.npcs {
		nx := int(npc.x / float64(frTileW))
		ny := int(npc.y / float64(frTileH))
		dx := px - nx
		dy := py - ny
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		dist := dx + dy
		if dist <= 2 && npc.talkCD <= 0 {
			// Check if giving a gift (selected slot has a crop)
			selItem := g.hotbar[g.selected].item
			giftPrice := frItemSellPrice(selItem)
			if giftPrice > 0 && g.hotbar[g.selected].count > 0 {
				npc.hearts++
				g.removeItem(g.selected, 1)
				if npc.hearts >= 10 {
					g.addMessage(fmt.Sprintf("%s: I love you! (Max hearts)", npc.name))
				} else {
					g.addMessage(fmt.Sprintf("%s loves your gift! Hearts: %d/10", npc.name, npc.hearts))
				}
			} else {
				g.addMessage(fmt.Sprintf("%s: %s", npc.name, npc.dialogues[npc.dlgIdx]))
				npc.dlgIdx = (npc.dlgIdx + 1) % len(npc.dialogues)
			}
			npc.talkCD = 30
			return
		}
	}
	g.addMessage("Nobody to talk to here.")
}

// ─── Day advancement ───

func (g *FarmRPG) advanceDay() {
	g.day++
	if g.day > frDaysPerSeason {
		g.day = 1
		g.season++
		if g.season > frWinter {
			g.season = frSpring
			g.year++
		}
		g.addMessage(fmt.Sprintf("A new season: %s!", frSeasonNames[g.season]))

		// Kill crops from wrong season
		for x := 0; x < frMapW; x++ {
			for y := 0; y < frMapH; y++ {
				if g.tiles[x][y] == frTilePlanted && g.crops[x][y] != nil {
					crop := frCrops[g.crops[x][y].cropIdx]
					if crop.season != g.season {
						g.crops[x][y].dead = true
					}
				}
			}
		}
	}

	// Advance time of day
	g.phase = frMorning

	// Energy restored
	g.energy = frMaxEnergy

	// Rain chance
	g.raining = rand.Float64() < 0.2

	// Grow crops
	for x := 0; x < frMapW; x++ {
		for y := 0; y < frMapH; y++ {
			if g.tiles[x][y] == frTilePlanted && g.crops[x][y] != nil {
				c := g.crops[x][y]
				if c.dead {
					continue
				}
				if g.raining {
					c.watered = true
				}
				if c.watered {
					c.growDay++
				}
				c.watered = false // reset for next day
			}
			// Reset watered soil to tilled
			if g.tiles[x][y] == frTileWatered {
				g.tiles[x][y] = frTileDirt
			}
		}
	}

	// Advance phase through the day
	g.phase = frMorning

	if g.raining {
		g.addMessage(fmt.Sprintf("Day %d, Year %d - %s - Raining! All crops watered.", g.day, g.year, frSeasonNames[g.season]))
	} else {
		g.addMessage(fmt.Sprintf("Day %d, Year %d - %s - A new day!", g.day, g.year, frSeasonNames[g.season]))
	}

	// Advance phase based on energy spent / tick
	// Actually we'll let the tick auto-advance phases
}

// ─── Shop update ───

func (g *FarmRPG) updateShop() {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.shopCursor--
		if g.shopCursor < 0 {
			g.shopCursor = len(frCrops) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.shopCursor++
		if g.shopCursor >= len(frCrops) {
			g.shopCursor = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		crop := frCrops[g.shopCursor]
		if g.money >= crop.seedCost {
			seedItem := frCropIdxToSeed(g.shopCursor)
			if g.addItem(seedItem, 5) {
				g.money -= crop.seedCost
				g.addMessage(fmt.Sprintf("Bought 5 %s seeds for $%d!", crop.name, crop.seedCost))
			}
		} else {
			g.addMessage("Not enough money!")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.shopState = frShopClosed
	}
}

// ─── Fishing update ───

func (g *FarmRPG) updateFishing() {
	switch g.fishState {
	case frFishCasting:
		// Bar moves back and forth
		speed := 0.025
		if g.phase == frNight {
			speed = 0.03 // faster at night
		}
		g.fishBar += float64(g.fishDir) * speed
		if g.fishBar >= 1.0 {
			g.fishBar = 1.0
			g.fishDir = -1
		}
		if g.fishBar <= 0 {
			g.fishBar = 0
			g.fishDir = 1
		}

		g.fishTimer--
		if g.fishTimer <= 0 {
			g.fishState = frFishMiss
			g.fishTimer = 60
			g.addMessage("The fish got away...")
			return
		}

		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			playSFX(g.sfxFish, 0.5)
			// Check if in green zone
			dist := math.Abs(g.fishBar - g.fishGreen)
			if dist < g.fishGreenW/2 {
				g.fishState = frFishCaught
				g.fishTimer = 60

				// Determine fish quality
				accuracy := 1.0 - (dist / (g.fishGreenW / 2))
				var fishItem int
				if accuracy > 0.8 {
					fishItem = frItemFishLarge
					g.addMessage("Caught a LARGE fish! ($200)")
				} else if accuracy > 0.4 {
					fishItem = frItemFishMed
					g.addMessage("Caught a medium fish! ($80)")
				} else {
					fishItem = frItemFishSmall
					g.addMessage("Caught a small fish! ($20)")
				}
				g.addItem(fishItem, 1)
				g.fishCaught++
			} else {
				g.fishState = frFishMiss
				g.fishTimer = 60
				g.addMessage("Missed! The fish got away...")
			}
		}

	case frFishCaught, frFishMiss:
		g.fishTimer--
		if g.fishTimer <= 0 {
			g.fishState = frFishIdle
		}
	}
}

// ─── NPC update ───

func (g *FarmRPG) updateNPCs() {
	for _, npc := range g.npcs {
		if npc.talkCD > 0 {
			npc.talkCD--
		}

		if len(npc.path) < 2 {
			continue
		}

		target := npc.path[npc.pathIdx]
		tx := float64(target[0]) * float64(frTileW)
		ty := float64(target[1]) * float64(frTileH)

		dx := tx - npc.x
		dy := ty - npc.y

		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 2 {
			npc.pathIdx = (npc.pathIdx + 1) % len(npc.path)
		} else {
			npc.x += (dx / dist) * npc.speed
			npc.y += (dy / dist) * npc.speed
		}
	}
}

// ─── Draw ───

func (g *FarmRPG) Draw(screen *ebiten.Image) {
	// Sky color based on time of day and phase
	skyColor := g.getSkyColor()
	screen.Fill(skyColor)

	// Draw tiles
	g.drawTiles(screen)

	// Draw crops
	g.drawCrops(screen)

	// Draw NPCs
	g.drawNPCs(screen)

	// Draw player
	g.drawPlayer(screen)

	// Draw HUD
	g.drawHUD(screen)

	// Draw messages
	g.drawMessages(screen)

	// Draw overlays
	if g.shopState == frShopOpen {
		g.drawShop(screen)
	}
	if g.fishState != frFishIdle {
		g.drawFishing(screen)
	}

	// Phase auto-advance visual
	if g.tick%300 == 0 && g.phase < frNight {
		g.phase++
	}
}

func (g *FarmRPG) getSkyColor() color.RGBA {
	seasonMod := [4]color.RGBA{
		{0x87, 0xCE, 0xEB, 0xff}, // Spring - light blue
		{0x55, 0xB0, 0xF0, 0xff}, // Summer - bright blue
		{0xC0, 0x90, 0x60, 0xff}, // Fall - amber
		{0xA0, 0xB0, 0xC8, 0xff}, // Winter - grey blue
	}

	base := seasonMod[g.season]

	// Darken based on phase
	switch g.phase {
	case frMorning:
		return base
	case frAfternoon:
		return color.RGBA{
			uint8(float64(base.R) * 0.9),
			uint8(float64(base.G) * 0.9),
			uint8(float64(base.B) * 0.95),
			0xff,
		}
	case frEvening:
		return color.RGBA{
			uint8(float64(base.R) * 0.6),
			uint8(float64(base.G) * 0.5),
			uint8(float64(base.B) * 0.6),
			0xff,
		}
	case frNight:
		return color.RGBA{0x15, 0x15, 0x30, 0xff}
	}
	return base
}

func (g *FarmRPG) grassColor() color.RGBA {
	switch g.season {
	case frSpring:
		return color.RGBA{0x4a, 0xb0, 0x4a, 0xff}
	case frSummer:
		return color.RGBA{0x3a, 0x90, 0x3a, 0xff}
	case frFall:
		return color.RGBA{0xb0, 0x80, 0x30, 0xff}
	case frWinter:
		return color.RGBA{0xe0, 0xe8, 0xf0, 0xff}
	}
	return color.RGBA{0x4a, 0xb0, 0x4a, 0xff}
}

func (g *FarmRPG) drawTiles(screen *ebiten.Image) {
	startTX := int(g.camX) / frTileW
	startTY := int(g.camY) / frTileH
	endTX := startTX + frScreenW/frTileW + 2
	endTY := startTY + frScreenH/frTileH + 2

	if startTX < 0 {
		startTX = 0
	}
	if startTY < 0 {
		startTY = 0
	}
	if endTX > frMapW {
		endTX = frMapW
	}
	if endTY > frMapH {
		endTY = frMapH
	}

	gc := g.grassColor()

	for tx := startTX; tx < endTX; tx++ {
		for ty := startTY; ty < endTY; ty++ {
			sx := float32(float64(tx*frTileW) - g.camX)
			sy := float32(float64(ty*frTileH) - g.camY)
			fw := float32(frTileW)
			fh := float32(frTileH)

			tile := g.tiles[tx][ty]

			var col color.RGBA
			switch tile {
			case frTileGrass:
				col = gc
			case frTileDirt:
				col = color.RGBA{0x8B, 0x6B, 0x3B, 0xff}
			case frTileWatered:
				col = color.RGBA{0x5B, 0x4B, 0x2B, 0xff}
			case frTilePlanted:
				col = color.RGBA{0x6B, 0x5B, 0x2B, 0xff}
			case frTileRock:
				col = color.RGBA{0x80, 0x80, 0x80, 0xff}
			case frTileTree:
				col = gc // base is grass
			case frTileFence:
				col = color.RGBA{0x8B, 0x73, 0x55, 0xff}
			case frTilePath:
				col = color.RGBA{0xc0, 0xb0, 0x90, 0xff}
			case frTileWater:
				// Animate water
				wb := 0.7 + 0.1*math.Sin(float64(g.tick)/20.0+float64(tx)*0.5)
				col = color.RGBA{0x30, 0x60, uint8(200 * wb), 0xff}
			case frTileBuilding:
				col = color.RGBA{0xA0, 0x60, 0x30, 0xff}
			case frTileShipBin:
				col = color.RGBA{0xC0, 0x40, 0x40, 0xff}
			case frTileShop:
				col = color.RGBA{0x60, 0x80, 0xA0, 0xff}
			case frTileDoor:
				col = color.RGBA{0x50, 0x30, 0x10, 0xff}
			}

			vector.FillRect(screen, sx, sy, fw, fh, col, false)

			// Tile outlines for grid visibility
			outlineCol := color.RGBA{0, 0, 0, 30}
			vector.StrokeRect(screen, sx, sy, fw, fh, 0.5, outlineCol, false)

			// Special decorations
			switch tile {
			case frTileRock:
				// Draw rock highlights
				vector.FillRect(screen, sx+4, sy+4, fw-8, fh-8, color.RGBA{0x99, 0x99, 0x99, 0xff}, false)
				vector.FillRect(screen, sx+8, sy+6, 6, 4, color.RGBA{0xbb, 0xbb, 0xbb, 0xff}, false)
			case frTileTree:
				// Trunk
				vector.FillRect(screen, sx+10, sy+14, 8, 14, color.RGBA{0x6B, 0x3B, 0x1B, 0xff}, false)
				// Canopy
				treeCol := gc
				treeCol.R = uint8(math.Min(255, float64(treeCol.R)*1.2))
				treeCol.G = uint8(math.Min(255, float64(treeCol.G)*1.3))
				vector.FillRect(screen, sx+3, sy+2, 22, 14, treeCol, false)
				vector.FillRect(screen, sx+6, sy-2, 16, 8, treeCol, false)
			case frTileFence:
				vector.FillRect(screen, sx+2, sy+10, fw-4, 4, color.RGBA{0xA0, 0x83, 0x65, 0xff}, false)
				vector.FillRect(screen, sx+6, sy+2, 3, 24, color.RGBA{0xA0, 0x83, 0x65, 0xff}, false)
				vector.FillRect(screen, sx+18, sy+2, 3, 24, color.RGBA{0xA0, 0x83, 0x65, 0xff}, false)
			case frTileBuilding:
				// Bricks
				if tx%2 == 0 {
					vector.FillRect(screen, sx+2, sy+2, fw-4, fh/2-2, color.RGBA{0xB0, 0x70, 0x40, 0xff}, false)
				}
			case frTileShipBin:
				// Draw an 'S' label
				ebitenutil.DebugPrintAt(screen, "BIN", int(sx)+4, int(sy)+8)
			case frTileShop:
				if ty == 1 {
					ebitenutil.DebugPrintAt(screen, "SHOP", int(sx)-2, int(sy)+8)
				}
			case frTileDoor:
				vector.FillRect(screen, sx+8, sy+4, 12, 20, color.RGBA{0x30, 0x18, 0x08, 0xff}, false)
			}
		}
	}

	// Rain effect
	if g.raining {
		for i := 0; i < 60; i++ {
			rx := rand.Float32() * float32(frScreenW)
			ry := rand.Float32() * float32(frScreenH)
			vector.StrokeLine(screen, rx, ry, rx-2, ry+8, 1, color.RGBA{0x80, 0x80, 0xff, 0x60}, false)
		}
	}
}

func (g *FarmRPG) drawCrops(screen *ebiten.Image) {
	for x := 0; x < frMapW; x++ {
		for y := 0; y < frMapH; y++ {
			if g.tiles[x][y] != frTilePlanted || g.crops[x][y] == nil {
				continue
			}

			c := g.crops[x][y]
			crop := frCrops[c.cropIdx]
			sx := float32(float64(x*frTileW) - g.camX)
			sy := float32(float64(y*frTileH) - g.camY)

			if sx < -frTileW || sx > float32(frScreenW) || sy < -frTileH || sy > float32(frScreenH) {
				continue
			}

			if c.dead {
				// Dead crop - brown withered
				vector.FillRect(screen, sx+10, sy+16, 8, 10, color.RGBA{0x60, 0x40, 0x20, 0xff}, false)
				vector.FillRect(screen, sx+8, sy+14, 12, 4, color.RGBA{0x50, 0x30, 0x10, 0xff}, false)
				continue
			}

			// Growth progress
			progress := float64(c.growDay) / float64(crop.growDays)
			if progress > 1 {
				progress = 1
			}

			// Draw crop based on growth stage
			h := 4 + int(progress*18)
			w := 4 + int(progress*10)

			var cropCol color.RGBA
			switch c.cropIdx {
			case 0: // Turnip
				cropCol = color.RGBA{0xe0, 0xe0, 0xff, 0xff}
			case 1: // Potato
				cropCol = color.RGBA{0x80, 0xc0, 0x40, 0xff}
			case 2: // Tomato
				if progress >= 1 {
					cropCol = color.RGBA{0xff, 0x30, 0x30, 0xff}
				} else {
					cropCol = color.RGBA{0x40, 0xa0, 0x40, 0xff}
				}
			case 3: // Corn
				cropCol = color.RGBA{0x80, 0xc0, 0x30, 0xff}
				if progress >= 1 {
					cropCol = color.RGBA{0xff, 0xe0, 0x40, 0xff}
				}
			case 4: // Pumpkin
				if progress >= 1 {
					cropCol = color.RGBA{0xff, 0x80, 0x00, 0xff}
				} else {
					cropCol = color.RGBA{0x40, 0x90, 0x40, 0xff}
				}
			case 5: // Starfruit
				if progress >= 1 {
					cropCol = color.RGBA{0xff, 0xff, 0x00, 0xff}
				} else {
					cropCol = color.RGBA{0x60, 0xb0, 0x60, 0xff}
				}
			}

			// Stem
			stemH := h
			vector.FillRect(screen, sx+13, sy+float32(frTileH-stemH), 2, float32(stemH), color.RGBA{0x30, 0x80, 0x30, 0xff}, false)

			// Leaves / fruit
			if progress >= 0.3 {
				vector.FillRect(screen, sx+float32(14-w/2), sy+float32(frTileH-h), float32(w), float32(h/2), cropCol, false)
			}
			if progress >= 0.7 {
				vector.FillRect(screen, sx+float32(14-w/2-1), sy+float32(frTileH-h-2), float32(w+2), float32(h/2+2), cropCol, false)
			}

			// Mature sparkle
			if progress >= 1.0 && g.tick%30 < 15 {
				sparkle := color.RGBA{0xff, 0xff, 0x80, 0xc0}
				vector.FillRect(screen, sx+12, sy+float32(frTileH-h-4), 4, 4, sparkle, false)
			}

			// Watered indicator
			if c.watered {
				vector.FillRect(screen, sx+2, sy+float32(frTileH-4), 4, 3, color.RGBA{0x40, 0x60, 0xff, 0xaa}, false)
			}
		}
	}
}

func (g *FarmRPG) drawNPCs(screen *ebiten.Image) {
	for _, npc := range g.npcs {
		sx := float32(npc.x - g.camX)
		sy := float32(npc.y - g.camY)

		if sx < -frTileW || sx > float32(frScreenW) || sy < -frTileH || sy > float32(frScreenH) {
			continue
		}

		// Body
		var bodyCol color.RGBA
		switch npc.name {
		case "Lily":
			bodyCol = color.RGBA{0xff, 0x80, 0xc0, 0xff}
		case "Gus":
			bodyCol = color.RGBA{0x60, 0x80, 0xff, 0xff}
		case "Mae":
			bodyCol = color.RGBA{0xff, 0xc0, 0x40, 0xff}
		}

		// Shadow
		vector.FillRect(screen, sx+4, sy+22, 20, 6, color.RGBA{0, 0, 0, 0x40}, false)

		// Body rectangle
		vector.FillRect(screen, sx+8, sy+10, 12, 14, bodyCol, false)

		// Head
		vector.FillRect(screen, sx+7, sy+2, 14, 10, color.RGBA{0xff, 0xd0, 0xa0, 0xff}, false)

		// Eyes
		vector.FillRect(screen, sx+10, sy+5, 2, 2, color.RGBA{0, 0, 0, 0xff}, false)
		vector.FillRect(screen, sx+16, sy+5, 2, 2, color.RGBA{0, 0, 0, 0xff}, false)

		// Name tag
		ebitenutil.DebugPrintAt(screen, npc.name, int(sx)+2, int(sy)-8)

		// Hearts
		heartStr := ""
		for i := 0; i < npc.hearts && i < 10; i++ {
			heartStr += "*"
		}
		if heartStr != "" {
			ebitenutil.DebugPrintAt(screen, heartStr, int(sx), int(sy)-16)
		}
	}
}

func (g *FarmRPG) drawPlayer(screen *ebiten.Image) {
	sx := float32(g.px - g.camX)
	sy := float32(g.py - g.camY)

	// Shadow
	vector.FillRect(screen, sx+4, sy+22, 20, 6, color.RGBA{0, 0, 0, 0x40}, false)

	// Body
	vector.FillRect(screen, sx+8, sy+10, 12, 14, color.RGBA{0x40, 0x60, 0xc0, 0xff}, false)

	// Head
	vector.FillRect(screen, sx+7, sy+2, 14, 10, color.RGBA{0xff, 0xd0, 0xa0, 0xff}, false)

	// Hair
	vector.FillRect(screen, sx+6, sy+1, 16, 4, color.RGBA{0x60, 0x30, 0x10, 0xff}, false)

	// Eyes based on direction
	switch g.pdir {
	case frDirDown:
		vector.FillRect(screen, sx+10, sy+6, 2, 3, color.RGBA{0x20, 0x20, 0x60, 0xff}, false)
		vector.FillRect(screen, sx+16, sy+6, 2, 3, color.RGBA{0x20, 0x20, 0x60, 0xff}, false)
	case frDirUp:
		// Show back of head
		vector.FillRect(screen, sx+6, sy+1, 16, 8, color.RGBA{0x60, 0x30, 0x10, 0xff}, false)
	case frDirLeft:
		vector.FillRect(screen, sx+8, sy+6, 2, 3, color.RGBA{0x20, 0x20, 0x60, 0xff}, false)
	case frDirRight:
		vector.FillRect(screen, sx+18, sy+6, 2, 3, color.RGBA{0x20, 0x20, 0x60, 0xff}, false)
	}

	// Tool indicator near player
	selItem := g.hotbar[g.selected].item
	if selItem != frItemNone {
		toolCol := frItemColor(selItem)
		vector.FillRect(screen, sx+22, sy+12, 6, 6, toolCol, false)
	}
}

func (g *FarmRPG) drawHUD(screen *ebiten.Image) {
	// Top HUD bar
	vector.FillRect(screen, 0, 0, float32(frScreenW), 44, color.RGBA{0x20, 0x20, 0x30, 0xd0}, false)

	// Date & time
	dateStr := fmt.Sprintf("Y%d %s Day %d - %s", g.year, frSeasonNames[g.season], g.day, frPhaseNames[g.phase])
	ebitenutil.DebugPrintAt(screen, dateStr, 8, 4)

	// Money
	moneyStr := fmt.Sprintf("$%d", g.money)
	ebitenutil.DebugPrintAt(screen, moneyStr, 8, 18)

	// Weather
	weatherStr := "Sunny"
	if g.raining {
		weatherStr = "Rainy"
	}
	ebitenutil.DebugPrintAt(screen, weatherStr, 200, 4)

	// Energy bar
	ebitenutil.DebugPrintAt(screen, "Energy:", 300, 4)
	barX := float32(360)
	barY := float32(4)
	barW := float32(100)
	barH := float32(10)
	vector.FillRect(screen, barX, barY, barW, barH, color.RGBA{0x40, 0x40, 0x40, 0xff}, false)
	energyFrac := float32(g.energy) / float32(frMaxEnergy)
	energyCol := color.RGBA{0x40, 0xc0, 0x40, 0xff}
	if energyFrac < 0.3 {
		energyCol = color.RGBA{0xc0, 0x40, 0x40, 0xff}
	} else if energyFrac < 0.6 {
		energyCol = color.RGBA{0xc0, 0xc0, 0x40, 0xff}
	}
	vector.FillRect(screen, barX, barY, barW*energyFrac, barH, energyCol, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d/%d", g.energy, frMaxEnergy), int(barX)+30, 4)

	// Stats
	statsStr := fmt.Sprintf("Sold:$%d Crops:%d Fish:%d", g.totalSold, g.cropsGrown, g.fishCaught)
	ebitenutil.DebugPrintAt(screen, statsStr, 300, 18)

	// Controls reminder
	controlStr := "T:Till P:Plant Q:Water H:Harvest S:Sell B:Buy F:Fish E:Talk N:Sleep"
	ebitenutil.DebugPrintAt(screen, controlStr, 8, 32)

	// Hotbar at bottom
	hotbarY := float32(frScreenH - 42)
	hotbarStartX := float32(frScreenW/2) - float32(frHotbarSlots*36)/2

	vector.FillRect(screen, hotbarStartX-4, hotbarY-4, float32(frHotbarSlots*36)+8, 40, color.RGBA{0x20, 0x20, 0x30, 0xd0}, false)

	for i := 0; i < frHotbarSlots; i++ {
		slotX := hotbarStartX + float32(i*36)
		slotY := hotbarY

		// Slot background
		bgCol := color.RGBA{0x40, 0x40, 0x50, 0xff}
		if i == g.selected {
			bgCol = color.RGBA{0x80, 0x80, 0xa0, 0xff}
		}
		vector.FillRect(screen, slotX, slotY, 32, 32, bgCol, false)
		vector.StrokeRect(screen, slotX, slotY, 32, 32, 1, color.RGBA{0xa0, 0xa0, 0xb0, 0xff}, false)

		slot := g.hotbar[i]
		if slot.item != frItemNone {
			// Item icon
			ic := frItemColor(slot.item)
			vector.FillRect(screen, slotX+4, slotY+4, 24, 18, ic, false)

			// Item name (abbreviated)
			name := frItemName(slot.item)
			if len(name) > 5 {
				name = name[:5]
			}
			ebitenutil.DebugPrintAt(screen, name, int(slotX)+2, int(slotY)+4)

			// Count
			if slot.count > 1 {
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", slot.count), int(slotX)+20, int(slotY)+22)
			}
		}

		// Slot number
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", i+1), int(slotX)+1, int(slotY)+22)
	}
}

func (g *FarmRPG) drawMessages(screen *ebiten.Image) {
	y := frScreenH - 60
	for i := len(g.messages) - 1; i >= 0; i-- {
		msg := g.messages[i]
		alpha := uint8(255)
		if msg.frames < 30 {
			alpha = uint8(255 * msg.frames / 30)
		}
		bgCol := color.RGBA{0x10, 0x10, 0x20, alpha - alpha/4}
		textY := y - (len(g.messages)-1-i)*16

		if textY < 50 {
			continue
		}

		vector.FillRect(screen, 8, float32(textY-2), float32(len(msg.text)*7+8), 14, bgCol, false)
		ebitenutil.DebugPrintAt(screen, msg.text, 12, textY)
	}
}

func (g *FarmRPG) drawShop(screen *ebiten.Image) {
	// Shop overlay
	ox := float32(frScreenW/2 - 150)
	oy := float32(60)
	ow := float32(300)
	oh := float32(320)

	vector.FillRect(screen, ox, oy, ow, oh, color.RGBA{0x20, 0x30, 0x40, 0xf0}, false)
	vector.StrokeRect(screen, ox, oy, ow, oh, 2, color.RGBA{0x80, 0xa0, 0xc0, 0xff}, false)

	ebitenutil.DebugPrintAt(screen, "=== SEED SHOP ===", int(ox)+80, int(oy)+8)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Your Money: $%d", g.money), int(ox)+20, int(oy)+28)

	for i, crop := range frCrops {
		ly := int(oy) + 50 + i*38

		bgCol := color.RGBA{0x30, 0x40, 0x50, 0xff}
		if i == g.shopCursor {
			bgCol = color.RGBA{0x50, 0x60, 0x80, 0xff}
		}
		vector.FillRect(screen, ox+10, float32(ly), ow-20, 34, bgCol, false)

		seasonTag := frSeasonNames[crop.season]
		info := fmt.Sprintf("%s (%s) - $%d/5 seeds", crop.name, seasonTag, crop.seedCost)
		ebitenutil.DebugPrintAt(screen, info, int(ox)+18, ly+4)

		detail := fmt.Sprintf("  Grows: %d days | Sells: $%d", crop.growDays, crop.sellVal)
		if crop.regrows {
			detail += " [Regrows!]"
		}
		ebitenutil.DebugPrintAt(screen, detail, int(ox)+18, ly+18)
	}

	ebitenutil.DebugPrintAt(screen, "UP/DOWN: Select  ENTER: Buy  ESC: Close", int(ox)+20, int(oy)+int(oh)-20)
}

func (g *FarmRPG) drawFishing(screen *ebiten.Image) {
	// Fishing overlay
	ox := float32(frScreenW/2 - 120)
	oy := float32(frScreenH/2 - 50)

	vector.FillRect(screen, ox, oy, 240, 100, color.RGBA{0x10, 0x20, 0x40, 0xf0}, false)
	vector.StrokeRect(screen, ox, oy, 240, 100, 2, color.RGBA{0x40, 0x80, 0xc0, 0xff}, false)

	switch g.fishState {
	case frFishCasting:
		ebitenutil.DebugPrintAt(screen, "== FISHING ==", int(ox)+70, int(oy)+8)
		ebitenutil.DebugPrintAt(screen, "Press SPACE in the green zone!", int(ox)+20, int(oy)+24)

		// Bar background
		barX := ox + 20
		barY := oy + 50
		barW := float32(200)
		barH := float32(20)

		vector.FillRect(screen, barX, barY, barW, barH, color.RGBA{0x40, 0x40, 0x40, 0xff}, false)

		// Green zone
		greenStart := float32(g.fishGreen-g.fishGreenW/2) * barW
		greenW := float32(g.fishGreenW) * barW
		vector.FillRect(screen, barX+greenStart, barY, greenW, barH, color.RGBA{0x40, 0xc0, 0x40, 0xff}, false)

		// Moving indicator
		indX := barX + float32(g.fishBar)*barW
		vector.FillRect(screen, indX-2, barY-4, 4, barH+8, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)

		// Timer
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Time: %d", g.fishTimer/30), int(ox)+100, int(oy)+78)

	case frFishCaught:
		ebitenutil.DebugPrintAt(screen, "CAUGHT!", int(ox)+90, int(oy)+40)

	case frFishMiss:
		ebitenutil.DebugPrintAt(screen, "Missed...", int(ox)+80, int(oy)+40)
	}
}

// ─── Interface ───

func (g *FarmRPG) Layout(w, h int) (int, int) {
	return frScreenW, frScreenH
}

func (g *FarmRPG) WantsBack() bool {
	return g.goBack
}
