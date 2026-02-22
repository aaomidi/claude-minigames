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
	rsScreenW = 560
	rsScreenH = 480
	rsTileS   = 16
	rsMapW    = 40
	rsMapH    = 30
	rsViewW   = 384 // game viewport width
	rsViewH   = 480
	rsPanelW  = 176 // right panel
	rsMiniS   = 100 // minimap size
	rsInvCols = 7
	rsInvRows = 4
	rsInvSlot = 22
)

// Tile types
const (
	rsGrass = iota
	rsDirt
	rsWater
	rsWall
	rsTree
	rsRock
	rsDoor
	rsFloor
	rsFire
	rsShopTile
	rsBankTile
	rsQuestTile
)

// Item IDs
const (
	rsNoItem = iota
	rsCoins
	rsLogs
	rsOre
	rsRawFish
	rsCookedFish
	rsBones
	rsFeathers
	rsFur
	rsBronzeSword
	rsDragonScale
)

// Skill IDs
const (
	rsAttack = iota
	rsStrength
	rsDefence
	rsHitpoints
	rsWoodcutting
	rsMining
	rsFishing
	rsCooking
	rsSkillCount
)

var rsSkillNames = [rsSkillCount]string{
	"Attack", "Strength", "Defence", "Hitpoints",
	"Woodcutting", "Mining", "Fishing", "Cooking",
}

var rsItemNames = []string{
	"", "Coins", "Logs", "Ore", "Raw Fish",
	"Cooked Fish", "Bones", "Feathers", "Fur", "Bronze Sword", "Dragon Scale",
}

var rsItemColors = []color.RGBA{
	{}, {0xff, 0xd7, 0x00, 0xff}, {0x8b, 0x45, 0x13, 0xff}, {0x80, 0x80, 0x90, 0xff},
	{0x40, 0x80, 0xc0, 0xff}, {0xc0, 0x80, 0x40, 0xff}, {0xd0, 0xd0, 0xd0, 0xff},
	{0xff, 0xff, 0xe0, 0xff}, {0xa0, 0x70, 0x40, 0xff}, {0xb0, 0x60, 0x20, 0xff},
	{0x60, 0xd0, 0x60, 0xff},
}

// Monster definitions
type rsMonsterDef struct {
	name   string
	level  int
	maxHP  int
	atk    int
	def    int
	str    int
	aggro  bool
	color  color.RGBA
	loot   [][2]int // {itemID, chance%}
}

var rsMonsters = []rsMonsterDef{
	{"Chicken", 1, 4, 1, 1, 1, false, color.RGBA{0xff, 0xff, 0x80, 0xff},
		[][2]int{{rsFeathers, 100}, {rsBones, 80}, {rsCoins, 50}}},
	{"Goblin", 5, 12, 4, 3, 5, true, color.RGBA{0x40, 0xa0, 0x40, 0xff},
		[][2]int{{rsCoins, 100}, {rsBones, 80}}},
	{"Wolf", 10, 20, 7, 5, 8, true, color.RGBA{0x70, 0x70, 0x70, 0xff},
		[][2]int{{rsBones, 100}, {rsFur, 60}, {rsCoins, 50}}},
	{"Skeleton", 25, 50, 15, 12, 18, true, color.RGBA{0xd0, 0xd0, 0xc0, 0xff},
		[][2]int{{rsBones, 100}, {rsCoins, 90}}},
	{"Dragon", 50, 120, 30, 25, 35, true, color.RGBA{0xd0, 0x30, 0x30, 0xff},
		[][2]int{{rsDragonScale, 100}, {rsCoins, 100}, {rsBones, 100}}},
}

// NPC types
const (
	rsNPCShop = iota
	rsNPCBank
	rsNPCQuest
	rsNPCCook
)

type rsNPC struct {
	x, y     int
	npcType  int
	name     string
	color    color.RGBA
	dialogue []string
	dialIdx  int
}

type rsMonster struct {
	defIdx    int
	x, y      float64
	hp        int
	maxHP     int
	respawnT  int
	dead      bool
	atkTimer  int
}

type rsInvItem struct {
	id    int
	count int
}

type rsSkillData struct {
	xp    int
	level int
}

type rsState int

const (
	rsStatePlaying rsState = iota
	rsStateShop
	rsStateBank
	rsStateSkills
	rsStateQuest
	rsStateFishing
	rsStateCooking
)

type Runescape struct {
	goBack bool

	tileMap  [rsMapH][rsMapW]int
	px, py   float64 // player world position in pixels
	destX, destY float64
	moving   bool
	camX, camY float64

	skills   [rsSkillCount]rsSkillData
	inv      [28]rsInvItem
	bank     [28]rsInvItem

	monsters []rsMonster
	npcs     []rsNPC
	target   int // monster index being fought, -1 if none
	combatT  int

	state    rsState
	shopSel  int

	// Skilling state
	skillTimer int
	skillType  int // which skill is being used

	// Quest
	questDone    bool
	questTurnedIn bool

	chatLog  []string
	ticks    int

	// Audio
	bgm       *audio.Player
	sfxHit    []byte
	sfxLvl    []byte
	sfxChop   []byte
	sfxMine   []byte
	sfxFish   []byte
	sfxPickup []byte
	audioInit bool
}

func newRunescape() *Runescape {
	g := &Runescape{target: -1}
	g.initMap()
	g.initPlayer()
	g.initMonsters()
	g.initNPCs()
	g.initRsAudio()
	g.addChat("Welcome to RuneScape! Click to move.")
	return g
}

func (g *Runescape) WantsBack() bool            { return g.goBack }
func (g *Runescape) Layout(w, h int) (int, int) { return rsScreenW, rsScreenH }

// --- Map generation ---

func (g *Runescape) initMap() {
	// Fill with grass
	for r := 0; r < rsMapH; r++ {
		for c := 0; c < rsMapW; c++ {
			g.tileMap[r][c] = rsGrass
		}
	}
	// Town center (15-25, 12-18) - stone floor with walls
	for r := 12; r <= 18; r++ {
		for c := 15; c <= 25; c++ {
			g.tileMap[r][c] = rsFloor
		}
	}
	// Town walls
	for c := 15; c <= 25; c++ {
		g.tileMap[11][c] = rsWall
		g.tileMap[19][c] = rsWall
	}
	for r := 11; r <= 19; r++ {
		g.tileMap[r][14] = rsWall
		g.tileMap[r][26] = rsWall
	}
	// Doors
	g.tileMap[11][20] = rsDoor
	g.tileMap[19][20] = rsDoor
	g.tileMap[15][14] = rsDoor
	g.tileMap[15][26] = rsDoor

	// Paths (dirt)
	for c := 0; c < rsMapW; c++ {
		g.tileMap[15][c] = rsDirt
		g.tileMap[16][c] = rsDirt
	}
	for r := 0; r < rsMapH; r++ {
		g.tileMap[r][20] = rsDirt
		g.tileMap[r][21] = rsDirt
	}

	// Shop tile
	g.tileMap[13][17] = rsShopTile
	g.tileMap[13][18] = rsShopTile
	// Bank tile
	g.tileMap[13][22] = rsBankTile
	g.tileMap[13][23] = rsBankTile
	// Quest giver
	g.tileMap[17][17] = rsQuestTile
	// Fire for cooking
	g.tileMap[17][23] = rsFire

	// Forest (north, 0-10 rows)
	for r := 0; r < 10; r++ {
		for c := 0; c < 15; c++ {
			if g.tileMap[r][c] == rsGrass && rand.Float64() < 0.3 {
				g.tileMap[r][c] = rsTree
			}
		}
	}

	// Mine (east, cols 28-38, rows 5-15)
	for r := 5; r < 15; r++ {
		for c := 28; c < 38; c++ {
			if g.tileMap[r][c] == rsGrass && rand.Float64() < 0.25 {
				g.tileMap[r][c] = rsRock
			}
		}
	}

	// Water (south, rows 22-28, cols 5-18)
	for r := 22; r < 28; r++ {
		for c := 5; c < 18; c++ {
			if g.tileMap[r][c] == rsGrass || g.tileMap[r][c] == rsDirt {
				g.tileMap[r][c] = rsWater
			}
		}
	}

	// Dungeon (west, rows 5-12, cols 0-10) — stone floor with walls
	for r := 5; r < 12; r++ {
		for c := 0; c < 10; c++ {
			g.tileMap[r][c] = rsFloor
		}
	}
	for c := 0; c < 10; c++ {
		g.tileMap[4][c] = rsWall
		g.tileMap[12][c] = rsWall
	}
	for r := 4; r <= 12; r++ {
		g.tileMap[r][10] = rsWall
	}
	g.tileMap[8][10] = rsDoor // dungeon entrance
}

func (g *Runescape) initPlayer() {
	g.px = 20 * rsTileS
	g.py = 15 * rsTileS
	g.destX = g.px
	g.destY = g.py
	for i := range g.skills {
		g.skills[i] = rsSkillData{xp: 0, level: 1}
	}
	g.skills[rsHitpoints] = rsSkillData{xp: 0, level: 10}
	// Starting inventory
	g.inv[0] = rsInvItem{rsBronzeSword, 1}
	g.inv[1] = rsInvItem{rsCoins, 50}
}

func (g *Runescape) initMonsters() {
	g.monsters = nil
	// Chickens in town outskirts
	for i := 0; i < 4; i++ {
		g.monsters = append(g.monsters, rsMonster{
			defIdx: 0, x: float64(27+rand.Intn(5)) * rsTileS, y: float64(14+rand.Intn(4)) * rsTileS,
			hp: rsMonsters[0].maxHP, maxHP: rsMonsters[0].maxHP,
		})
	}
	// Goblins near forest edge
	for i := 0; i < 3; i++ {
		g.monsters = append(g.monsters, rsMonster{
			defIdx: 1, x: float64(12+rand.Intn(6)) * rsTileS, y: float64(10+rand.Intn(2)) * rsTileS,
			hp: rsMonsters[1].maxHP, maxHP: rsMonsters[1].maxHP,
		})
	}
	// Wolves in forest
	for i := 0; i < 3; i++ {
		g.monsters = append(g.monsters, rsMonster{
			defIdx: 2, x: float64(2+rand.Intn(10)) * rsTileS, y: float64(1+rand.Intn(6)) * rsTileS,
			hp: rsMonsters[2].maxHP, maxHP: rsMonsters[2].maxHP,
		})
	}
	// Skeletons in dungeon
	for i := 0; i < 3; i++ {
		g.monsters = append(g.monsters, rsMonster{
			defIdx: 3, x: float64(2+rand.Intn(6)) * rsTileS, y: float64(6+rand.Intn(4)) * rsTileS,
			hp: rsMonsters[3].maxHP, maxHP: rsMonsters[3].maxHP,
		})
	}
	// Dragon deep in dungeon
	g.monsters = append(g.monsters, rsMonster{
		defIdx: 4, x: 5 * rsTileS, y: 8 * rsTileS,
		hp: rsMonsters[4].maxHP, maxHP: rsMonsters[4].maxHP,
	})
}

func (g *Runescape) initNPCs() {
	g.npcs = []rsNPC{
		{17, 13, rsNPCShop, "Shopkeeper", color.RGBA{0xff, 0xff, 0x00, 0xff},
			[]string{"Welcome to my shop!", "Buy or sell items.", "Press B near me to trade."}, 0},
		{22, 13, rsNPCBank, "Banker", color.RGBA{0x40, 0x40, 0xc0, 0xff},
			[]string{"Welcome to the bank.", "Your items are safe here.", "Press K near me to bank."}, 0},
		{17, 17, rsNPCQuest, "Quest Giver", color.RGBA{0xff, 0x80, 0x00, 0xff},
			[]string{"I'm hungry! Bring me 5 cooked fish.", "You'll be rewarded handsomely!", "Press Q near me."}, 0},
		{23, 17, rsNPCCook, "Cook", color.RGBA{0xff, 0xff, 0xff, 0xff},
			[]string{"Use raw fish on the fire to cook!", "Stand near the fire and press C.", "Higher cooking = less burning."}, 0},
	}
}

// --- XP and leveling ---

func rsXPForLevel(lvl int) int {
	return lvl * lvl * 10
}

func rsLevelFromXP(xp int) int {
	lvl := 1
	for rsXPForLevel(lvl+1) <= xp {
		lvl++
	}
	return lvl
}

func (g *Runescape) addXP(skill, amount int) {
	g.skills[skill].xp += amount
	newLvl := rsLevelFromXP(g.skills[skill].xp)
	if newLvl > g.skills[skill].level {
		g.skills[skill].level = newLvl
		g.addChat(fmt.Sprintf("Level up! %s is now %d!", rsSkillNames[skill], newLvl))
		playSFX(g.sfxLvl, 0.4)
	}
}

func (g *Runescape) combatLevel() int {
	a := g.skills[rsAttack].level
	s := g.skills[rsStrength].level
	d := g.skills[rsDefence].level
	h := g.skills[rsHitpoints].level
	return (a + s + d + h) / 4
}

func (g *Runescape) maxHP() int {
	return g.skills[rsHitpoints].level + 10
}

// --- Inventory ---

func (g *Runescape) addItem(id, count int) bool {
	// Stack with existing
	for i := range g.inv {
		if g.inv[i].id == id {
			g.inv[i].count += count
			return true
		}
	}
	// Find empty slot
	for i := range g.inv {
		if g.inv[i].id == rsNoItem {
			g.inv[i] = rsInvItem{id, count}
			return true
		}
	}
	g.addChat("Inventory full!")
	return false
}

func (g *Runescape) removeItem(id, count int) bool {
	for i := range g.inv {
		if g.inv[i].id == id && g.inv[i].count >= count {
			g.inv[i].count -= count
			if g.inv[i].count <= 0 {
				g.inv[i] = rsInvItem{}
			}
			return true
		}
	}
	return false
}

func (g *Runescape) itemCount(id int) int {
	for _, it := range g.inv {
		if it.id == id {
			return it.count
		}
	}
	return 0
}

func (g *Runescape) addChat(msg string) {
	g.chatLog = append(g.chatLog, msg)
	if len(g.chatLog) > 50 {
		g.chatLog = g.chatLog[len(g.chatLog)-50:]
	}
}

// --- Audio ---

func (g *Runescape) initRsAudio() {
	melody := [][2]int{
		// A — pastoral C major theme
		{60, 2}, {64, 2}, {67, 2}, {72, 2},
		{71, 1}, {69, 1}, {67, 2}, {64, 2},
		{62, 2}, {64, 1}, {67, 1}, {69, 2}, {67, 2},
		{64, 1}, {62, 1}, {60, 2}, {0, 2},
		// A' — variation
		{67, 2}, {69, 2}, {71, 1}, {72, 1}, {74, 2},
		{72, 1}, {71, 1}, {69, 2}, {67, 2},
		{64, 2}, {67, 1}, {69, 1}, {71, 2}, {72, 2},
		{69, 1}, {67, 1}, {64, 2}, {60, 4},
		// B — Am section, adventure
		{69, 2}, {72, 2}, {76, 2}, {74, 1}, {72, 1},
		{69, 2}, {67, 2}, {64, 2}, {0, 2},
		{72, 1}, {74, 1}, {76, 2}, {79, 2},
		{76, 1}, {74, 1}, {72, 2}, {69, 4},
		// C — return, triumphant
		{60, 2}, {64, 2}, {67, 1}, {69, 1}, {72, 2},
		{74, 2}, {76, 2}, {74, 1}, {72, 1},
		{69, 2}, {67, 2}, {64, 2}, {60, 4}, {0, 2},
	}
	bass := [][2]int{
		{48, 4}, {48, 4}, {52, 4}, {48, 4},
		{50, 4}, {52, 4}, {48, 4}, {48, 4},
		{55, 4}, {52, 4}, {48, 4}, {50, 4},
		{52, 4}, {55, 4}, {48, 4}, {48, 4},
		{45, 4}, {48, 4}, {52, 4}, {48, 4},
		{45, 4}, {48, 4}, {52, 4}, {45, 4},
		{48, 4}, {52, 4}, {55, 4}, {48, 4},
		{52, 4}, {55, 4}, {48, 4}, {48, 4},
	}
	drums := generateDrumPattern("K...h...s...h...K...h...s.h.h...", 95, 0.20)
	mel := generateTrack(melody, triangleWave, 95, 0.30)
	bas := generateTrack(bass, squareWave, 95, 0.15)
	mix := mixBuffers(mel, bas)
	for len(drums) < len(mix) {
		drums = append(drums, drums...)
	}
	drums = drums[:len(mix)]
	mix = mixBuffers(mix, drums)
	g.bgm = loopPlayer(mix, 0.18)

	// SFX
	g.sfxHit = generateTrack([][2]int{{48, 1}}, squareWave, 600, 0.4)
	g.sfxLvl = generateTrack([][2]int{{72, 1}, {76, 1}, {79, 1}, {84, 2}}, triangleWave, 400, 0.4)
	g.sfxChop = generateTrack([][2]int{{55, 1}}, squareWave, 500, 0.3)
	g.sfxMine = generateTrack([][2]int{{65, 1}}, triangleWave, 500, 0.3)
	g.sfxFish = generateTrack([][2]int{{72, 1}, {74, 1}}, sineWave, 400, 0.3)
	g.sfxPickup = generateTrack([][2]int{{76, 1}, {79, 1}}, sineWave, 500, 0.3)
}

// --- Update ---

func (g *Runescape) Update() error {
	if !g.audioInit {
		g.audioInit = true
		if g.bgm != nil {
			g.bgm.Play()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.state != rsStatePlaying {
			g.state = rsStatePlaying
			return nil
		}
		if g.bgm != nil {
			g.bgm.Pause()
		}
		g.goBack = true
		return nil
	}

	g.ticks++

	// Respawn dead monsters
	for i := range g.monsters {
		if g.monsters[i].dead {
			g.monsters[i].respawnT--
			if g.monsters[i].respawnT <= 0 {
				g.monsters[i].dead = false
				g.monsters[i].hp = g.monsters[i].maxHP
			}
		}
	}

	switch g.state {
	case rsStatePlaying:
		g.updatePlaying()
	case rsStateSkills:
		if inpututil.IsKeyJustPressed(ebiten.KeyS) {
			g.state = rsStatePlaying
		}
	case rsStateShop:
		g.updateShop()
	case rsStateFishing:
		g.updateFishing()
	case rsStateCooking:
		g.updateCooking()
	}

	// Camera follow
	g.camX += (g.px - float64(rsViewW)/2 - g.camX) * 0.1
	g.camY += (g.py - float64(rsViewH)/2 - g.camY) * 0.1
	maxCamX := float64(rsMapW*rsTileS) - float64(rsViewW)
	maxCamY := float64(rsMapH*rsTileS) - float64(rsViewH)
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

	return nil
}

func (g *Runescape) updatePlaying() {
	// Toggle skills
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.state = rsStateSkills
		return
	}

	// Skilling progress
	if g.skillTimer > 0 {
		g.skillTimer--
		if g.skillTimer <= 0 {
			g.finishSkilling()
		}
		return
	}

	// Click to move / interact
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if mx < rsViewW {
			// Clicked in game viewport
			worldX := float64(mx) + g.camX
			worldY := float64(my) + g.camY
			tileC := int(worldX) / rsTileS
			tileR := int(worldY) / rsTileS

			// Check if clicked on a monster
			clicked := false
			for i := range g.monsters {
				if g.monsters[i].dead {
					continue
				}
				dx := worldX - g.monsters[i].x
				dy := worldY - g.monsters[i].y
				if dx*dx+dy*dy < 20*20 {
					g.target = i
					g.destX = g.monsters[i].x
					g.destY = g.monsters[i].y
					g.moving = true
					clicked = true
					break
				}
			}

			// Check if clicked on interactable tile
			if !clicked && tileR >= 0 && tileR < rsMapH && tileC >= 0 && tileC < rsMapW {
				tile := g.tileMap[tileR][tileC]
				switch tile {
				case rsTree:
					g.destX = float64(tileC*rsTileS + rsTileS/2)
					g.destY = float64(tileR*rsTileS + rsTileS/2)
					g.moving = true
					g.skillType = rsWoodcutting
				case rsRock:
					g.destX = float64(tileC*rsTileS + rsTileS/2)
					g.destY = float64(tileR*rsTileS + rsTileS/2)
					g.moving = true
					g.skillType = rsMining
				case rsWater:
					g.destX = float64(tileC*rsTileS + rsTileS/2)
					g.destY = float64(tileR*rsTileS + rsTileS/2)
					g.moving = true
					g.skillType = rsFishing
				default:
					g.destX = worldX
					g.destY = worldY
					g.moving = true
					g.target = -1
					g.skillType = -1
				}
				clicked = true
			}

			if !clicked {
				g.destX = worldX
				g.destY = worldY
				g.moving = true
				g.target = -1
			}
		}
	}

	// Keyboard shortcuts near NPCs/objects
	pr, pc := int(g.py)/rsTileS, int(g.px)/rsTileS
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if g.nearTile(rsShopTile) {
			g.state = rsStateShop
			g.shopSel = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if g.nearTile(rsFire) {
			g.startCooking()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		if g.nearTile(rsQuestTile) {
			g.handleQuest()
		}
	}
	_ = pr
	_ = pc

	// NPC talk with E
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		for i := range g.npcs {
			dx := g.px - float64(g.npcs[i].x*rsTileS)
			dy := g.py - float64(g.npcs[i].y*rsTileS)
			if dx*dx+dy*dy < 32*32 {
				npc := &g.npcs[i]
				g.addChat(fmt.Sprintf("%s: %s", npc.name, npc.dialogue[npc.dialIdx]))
				npc.dialIdx = (npc.dialIdx + 1) % len(npc.dialogue)
			}
		}
	}

	// Movement
	if g.moving {
		dx := g.destX - g.px
		dy := g.destY - g.py
		dist := math.Sqrt(dx*dx + dy*dy)
		speed := 2.0
		if dist < speed {
			g.px = g.destX
			g.py = g.destY
			g.moving = false
			g.onArrived()
		} else {
			nx := dx / dist * speed
			ny := dy / dist * speed
			newX := g.px + nx
			newY := g.py + ny
			tc := int(newX) / rsTileS
			tr := int(newY) / rsTileS
			if tc >= 0 && tc < rsMapW && tr >= 0 && tr < rsMapH {
				tile := g.tileMap[tr][tc]
				if tile != rsWall && tile != rsWater {
					g.px = newX
					g.py = newY
				} else {
					g.moving = false
				}
			}
		}
	}

	// Combat
	if g.target >= 0 && g.target < len(g.monsters) && !g.monsters[g.target].dead {
		m := &g.monsters[g.target]
		dx := g.px - m.x
		dy := g.py - m.y
		if dx*dx+dy*dy < 24*24 {
			g.moving = false
			g.combatT++
			if g.combatT >= 40 {
				g.combatT = 0
				g.doCombatRound()
			}
		}
	}

	// Aggressive monsters
	for i := range g.monsters {
		if g.monsters[i].dead {
			continue
		}
		def := rsMonsters[g.monsters[i].defIdx]
		if !def.aggro {
			continue
		}
		dx := g.px - g.monsters[i].x
		dy := g.py - g.monsters[i].y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 80 && dist > 20 {
			g.monsters[i].x += dx / dist * 0.8
			g.monsters[i].y += dy / dist * 0.8
		}
		if dist < 24 && g.target != i {
			g.target = i
			g.combatT = 0
		}
	}
}

func (g *Runescape) nearTile(tileType int) bool {
	pr, pc := int(g.py)/rsTileS, int(g.px)/rsTileS
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			r, c := pr+dr, pc+dc
			if r >= 0 && r < rsMapH && c >= 0 && c < rsMapW {
				if g.tileMap[r][c] == tileType {
					return true
				}
			}
		}
	}
	return false
}

func (g *Runescape) onArrived() {
	if g.skillType == rsWoodcutting {
		g.startWoodcutting()
	} else if g.skillType == rsMining {
		g.startMining()
	} else if g.skillType == rsFishing {
		g.startFishingAction()
	}
	g.skillType = -1
}

func (g *Runescape) startWoodcutting() {
	g.addChat("You swing your axe...")
	g.skillTimer = 60
	g.skillType = rsWoodcutting
}

func (g *Runescape) startMining() {
	g.addChat("You swing your pickaxe...")
	g.skillTimer = 70
	g.skillType = rsMining
}

func (g *Runescape) startFishingAction() {
	g.addChat("You cast your net...")
	g.skillTimer = 80
	g.skillType = rsFishing
}

func (g *Runescape) startCooking() {
	if g.itemCount(rsRawFish) <= 0 {
		g.addChat("You have no raw fish to cook!")
		return
	}
	g.addChat("You cook a fish...")
	g.skillTimer = 30
	g.skillType = rsCooking
	g.state = rsStateCooking
}

func (g *Runescape) finishSkilling() {
	switch g.skillType {
	case rsWoodcutting:
		g.addItem(rsLogs, 1)
		g.addXP(rsWoodcutting, 25)
		g.addChat("You get some logs.")
		playSFX(g.sfxChop, 0.35)
	case rsMining:
		g.addItem(rsOre, 1)
		g.addXP(rsMining, 30)
		g.addChat("You mine some ore.")
		playSFX(g.sfxMine, 0.35)
	case rsFishing:
		g.addItem(rsRawFish, 1)
		g.addXP(rsFishing, 20)
		g.addChat("You catch a fish!")
		playSFX(g.sfxFish, 0.35)
	case rsCooking:
		if g.removeItem(rsRawFish, 1) {
			burnChance := 60 - g.skills[rsCooking].level*3
			if burnChance < 5 {
				burnChance = 5
			}
			if rand.Intn(100) < burnChance {
				g.addChat("You accidentally burn the fish!")
			} else {
				g.addItem(rsCookedFish, 1)
				g.addXP(rsCooking, 30)
				g.addChat("You cook a fish.")
			}
		}
		g.state = rsStatePlaying
	}
	g.skillType = -1
}

func (g *Runescape) doCombatRound() {
	if g.target < 0 || g.target >= len(g.monsters) {
		return
	}
	m := &g.monsters[g.target]
	def := rsMonsters[m.defIdx]

	// Player attacks monster
	atkRoll := rand.Intn(g.skills[rsAttack].level+5) + 1
	defRoll := rand.Intn(def.def+5) + 1
	if atkRoll > defRoll {
		maxHit := g.skills[rsStrength].level/2 + 2
		dmg := rand.Intn(maxHit) + 1
		m.hp -= dmg
		g.addChat(fmt.Sprintf("You hit the %s for %d damage.", def.name, dmg))
		playSFX(g.sfxHit, 0.3)
		g.addXP(rsAttack, dmg*2)
		g.addXP(rsStrength, dmg*2)
	} else {
		g.addChat(fmt.Sprintf("You miss the %s.", def.name))
	}

	if m.hp <= 0 {
		m.dead = true
		m.respawnT = 300
		g.target = -1
		g.addChat(fmt.Sprintf("You killed the %s!", def.name))
		g.addXP(rsHitpoints, def.maxHP)
		// Loot
		for _, loot := range def.loot {
			if rand.Intn(100) < loot[1] {
				amt := 1
				if loot[0] == rsCoins {
					amt = def.level*3 + rand.Intn(def.level*5+1)
				}
				g.addItem(loot[0], amt)
				g.addChat(fmt.Sprintf("You received: %s x%d", rsItemNames[loot[0]], amt))
				playSFX(g.sfxPickup, 0.3)
			}
		}
		return
	}

	// Monster attacks player
	mAtkRoll := rand.Intn(def.atk+5) + 1
	pDefRoll := rand.Intn(g.skills[rsDefence].level+5) + 1
	if mAtkRoll > pDefRoll {
		maxHit := def.str/2 + 1
		dmg := rand.Intn(maxHit) + 1
		g.skills[rsHitpoints].xp -= dmg // lose HP as XP for simplicity
		if g.skills[rsHitpoints].xp < 0 {
			g.skills[rsHitpoints].xp = 0
		}
		g.addXP(rsDefence, dmg)
		g.addChat(fmt.Sprintf("The %s hits you for %d.", def.name, dmg))
		// Check death
		currentHP := g.skills[rsHitpoints].level + 10 - (rsXPForLevel(g.skills[rsHitpoints].level) - g.skills[rsHitpoints].xp)
		if currentHP <= 0 {
			g.addChat("Oh no! You have died!")
			g.addChat("You respawn in town.")
			g.px = 20 * rsTileS
			g.py = 15 * rsTileS
			g.skills[rsHitpoints].xp = rsXPForLevel(g.skills[rsHitpoints].level)
			g.target = -1
		}
	}
}

func (g *Runescape) handleQuest() {
	if g.questTurnedIn {
		g.addChat("Quest Giver: Thanks again for the fish!")
		return
	}
	if g.itemCount(rsCookedFish) >= 5 {
		g.removeItem(rsCookedFish, 5)
		g.addItem(rsCoins, 500)
		g.questDone = true
		g.questTurnedIn = true
		g.addChat("Quest Giver: Thank you! Here's 500 coins!")
		g.addXP(rsCooking, 200)
	} else {
		have := g.itemCount(rsCookedFish)
		g.addChat(fmt.Sprintf("Quest Giver: Bring me 5 cooked fish! (%d/5)", have))
	}
}

func (g *Runescape) updateShop() {
	if inpututil.IsKeyJustPressed(ebiten.KeyB) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = rsStatePlaying
		return
	}
	// Sell with S — sell first sellable item
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		for i := range g.inv {
			it := g.inv[i]
			if it.id == rsNoItem || it.id == rsBronzeSword || it.id == rsCoins {
				continue
			}
			price := g.sellPrice(it.id)
			g.removeItem(it.id, 1)
			g.addItem(rsCoins, price)
			g.addChat(fmt.Sprintf("Sold %s for %d coins.", rsItemNames[it.id], price))
			break
		}
	}
}

func (g *Runescape) sellPrice(id int) int {
	switch id {
	case rsLogs:
		return 5
	case rsOre:
		return 10
	case rsRawFish:
		return 8
	case rsCookedFish:
		return 15
	case rsBones:
		return 3
	case rsFeathers:
		return 2
	case rsFur:
		return 20
	case rsDragonScale:
		return 500
	}
	return 1
}

func (g *Runescape) updateFishing() {
	g.skillTimer--
	if g.skillTimer <= 0 {
		g.finishSkilling()
		g.state = rsStatePlaying
	}
}

func (g *Runescape) updateCooking() {
	g.skillTimer--
	if g.skillTimer <= 0 {
		g.finishSkilling()
	}
}

// --- Draw ---

func (g *Runescape) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x10, 0x10, 0x10, 0xff})
	g.drawViewport(screen)
	g.drawRightPanel(screen)
	g.drawChatLog(screen)
}

func (g *Runescape) drawViewport(screen *ebiten.Image) {
	t := float64(g.ticks)

	for r := 0; r < rsMapH; r++ {
		for c := 0; c < rsMapW; c++ {
			sx := float32(float64(c*rsTileS) - g.camX)
			sy := float32(float64(r*rsTileS) - g.camY)
			if sx+rsTileS < 0 || sx > float32(rsViewW) || sy+rsTileS < 0 || sy > float32(rsViewH) {
				continue
			}
			tile := g.tileMap[r][c]
			tc := g.tileColor(tile)
			// Grass variation
			if tile == rsGrass {
				shade := byte((r*7 + c*13) % 15)
				tc.G += shade
			}
			vector.FillRect(screen, sx, sy, rsTileS, rsTileS, tc, false)

			switch tile {
			case rsTree:
				// Shadow
				vector.FillCircle(screen, sx+9, sy+12, 5, color.RGBA{0x15, 0x40, 0x10, 0x60}, false)
				// Trunk
				vector.FillRect(screen, sx+6, sy+8, 4, 8, color.RGBA{0x6b, 0x3a, 0x0a, 0xff}, false)
				vector.FillRect(screen, sx+7, sy+9, 2, 6, color.RGBA{0x80, 0x50, 0x20, 0xff}, false)
				// Canopy layers
				vector.FillCircle(screen, sx+8, sy+5, 7, color.RGBA{0x18, 0x70, 0x18, 0xff}, false)
				vector.FillCircle(screen, sx+6, sy+4, 4, color.RGBA{0x25, 0x90, 0x25, 0xff}, false)
				vector.FillCircle(screen, sx+10, sy+3, 3, color.RGBA{0x30, 0xa0, 0x20, 0xff}, false)
				// Leaf highlights
				sway := float32(math.Sin(t*0.03+float64(c*5))) * 0.5
				vector.FillCircle(screen, sx+5+sway, sy+3, 1.5, color.RGBA{0x40, 0xb0, 0x30, 0xff}, false)

			case rsRock:
				// Rock body with highlights
				vector.FillCircle(screen, sx+8, sy+9, 6, color.RGBA{0x60, 0x60, 0x60, 0xff}, false)
				vector.FillCircle(screen, sx+7, sy+7, 5, color.RGBA{0x78, 0x78, 0x78, 0xff}, false)
				vector.FillCircle(screen, sx+5, sy+5, 2, color.RGBA{0x95, 0x95, 0x95, 0xff}, false)
				// Ore sparkle
				if (g.ticks/15+r+c)%7 == 0 {
					vector.FillCircle(screen, sx+10, sy+6, 1, color.RGBA{0xff, 0xff, 0xcc, 0xff}, false)
				}

			case rsFire:
				// Animated fire
				flicker := float32(math.Sin(t*0.15)) * 2
				vector.FillCircle(screen, sx+8, sy+10, 4, color.RGBA{0xc0, 0x30, 0x00, 0xff}, false)
				vector.FillCircle(screen, sx+8, sy+7+flicker, 5, color.RGBA{0xff, 0x80, 0x00, 0xff}, false)
				vector.FillCircle(screen, sx+8, sy+5+flicker, 3, color.RGBA{0xff, 0xd0, 0x00, 0xff}, false)
				vector.FillCircle(screen, sx+8, sy+3+flicker, 1.5, color.RGBA{0xff, 0xff, 0x80, 0xff}, false)
				// Glow
				vector.FillCircle(screen, sx+8, sy+8, 10, color.RGBA{0xff, 0x60, 0x00, 0x18}, false)

			case rsWater:
				// Animated waves
				wave := float32(math.Sin(t*0.04+float64(c)*0.8+float64(r)*0.5)) * 0.1
				wc := color.RGBA{
					uint8(0x20 + int(wave*30)),
					uint8(0x50 + int(math.Abs(float64(wave))*40)),
					uint8(0xb0 + int(wave*20)),
					0xff,
				}
				vector.FillRect(screen, sx, sy, rsTileS, rsTileS, wc, false)
				// Wave lines
				wy := sy + 4 + float32(math.Sin(t*0.06+float64(c)))*2
				vector.StrokeLine(screen, sx+1, wy, sx+rsTileS-1, wy, 0.5, color.RGBA{0x40, 0x70, 0xd0, 0x60}, false)
				wy2 := sy + 10 + float32(math.Sin(t*0.06+float64(c)+2))*2
				vector.StrokeLine(screen, sx+2, wy2, sx+rsTileS-2, wy2, 0.5, color.RGBA{0x40, 0x70, 0xd0, 0x40}, false)

			case rsWall:
				// Brick pattern
				vector.StrokeLine(screen, sx, sy+rsTileS/2, sx+rsTileS, sy+rsTileS/2, 0.5, color.RGBA{0x45, 0x45, 0x45, 0xff}, false)
				vector.StrokeLine(screen, sx+rsTileS/2, sy, sx+rsTileS/2, sy+rsTileS/2, 0.5, color.RGBA{0x45, 0x45, 0x45, 0xff}, false)
				vector.StrokeLine(screen, sx+rsTileS/4, sy+rsTileS/2, sx+rsTileS/4, sy+rsTileS, 0.5, color.RGBA{0x45, 0x45, 0x45, 0xff}, false)

			case rsShopTile:
				// Shop sign
				vector.FillRect(screen, sx+2, sy+1, 12, 8, color.RGBA{0x8b, 0x60, 0x20, 0xff}, false)
				ebitenutil.DebugPrintAt(screen, "$", int(sx)+5, int(sy))

			case rsBankTile:
				vector.FillRect(screen, sx+2, sy+1, 12, 8, color.RGBA{0x20, 0x20, 0x80, 0xff}, false)
				ebitenutil.DebugPrintAt(screen, "B", int(sx)+5, int(sy))

			case rsQuestTile:
				// Pulsing quest marker
				pulse := float32(math.Sin(t*0.08))*0.3 + 0.7
				qc := color.RGBA{uint8(float32(0xff) * pulse), uint8(float32(0xc0) * pulse), 0x00, 0xff}
				vector.FillCircle(screen, sx+8, sy+5, 5, qc, false)
				ebitenutil.DebugPrintAt(screen, "!", int(sx)+5, int(sy)-1)

			case rsDoor:
				// Wooden door with handle
				vector.FillRect(screen, sx+3, sy+1, 10, 14, color.RGBA{0x8b, 0x60, 0x30, 0xff}, false)
				vector.FillRect(screen, sx+4, sy+2, 8, 5, color.RGBA{0x7a, 0x50, 0x25, 0xff}, false)
				vector.FillCircle(screen, sx+10, sy+9, 1, color.RGBA{0xd0, 0xb0, 0x40, 0xff}, false)
			}
		}
	}

	// NPCs — draw as proper characters
	for _, npc := range g.npcs {
		sx := float32(float64(npc.x*rsTileS) - g.camX)
		sy := float32(float64(npc.y*rsTileS) - g.camY)
		cx, cy := sx+8, sy+8
		// Shadow
		vector.FillCircle(screen, cx+1, cy+7, 5, color.RGBA{0, 0, 0, 0x30}, false)
		// Body
		vector.FillRect(screen, cx-4, cy-1, 8, 10, npc.color, false)
		// Head
		vector.FillCircle(screen, cx, cy-4, 4, color.RGBA{0xff, 0xd0, 0x90, 0xff}, false)
		// Eyes
		vector.FillCircle(screen, cx-1.5, cy-5, 0.8, color.RGBA{0x20, 0x20, 0x20, 0xff}, false)
		vector.FillCircle(screen, cx+1.5, cy-5, 0.8, color.RGBA{0x20, 0x20, 0x20, 0xff}, false)
		// Name tag with background
		nameW := len(npc.name) * 6
		vector.FillRect(screen, cx-float32(nameW)/2-2, cy-16, float32(nameW+4), 12, color.RGBA{0, 0, 0, 0x80}, false)
		ebitenutil.DebugPrintAt(screen, npc.name, int(cx)-nameW/2, int(cy)-15)
	}

	// Monsters — draw as proper creatures
	for _, m := range g.monsters {
		if m.dead {
			continue
		}
		def := rsMonsters[m.defIdx]
		sx := float32(m.x - g.camX)
		sy := float32(m.y - g.camY)

		// Shadow
		vector.FillCircle(screen, sx+1, sy+6, 5, color.RGBA{0, 0, 0, 0x30}, false)

		switch m.defIdx {
		case 0: // Chicken — small body with beak
			vector.FillCircle(screen, sx, sy, 5, color.RGBA{0xff, 0xff, 0x80, 0xff}, false)
			vector.FillCircle(screen, sx+3, sy-2, 2, color.RGBA{0xff, 0xff, 0xa0, 0xff}, false)
			vector.FillRect(screen, sx+4, sy-2, 3, 2, color.RGBA{0xff, 0xa0, 0x00, 0xff}, false) // beak
			vector.FillCircle(screen, sx+3, sy-3, 0.8, color.RGBA{0x20, 0x20, 0x20, 0xff}, false) // eye
			vector.FillRect(screen, sx+1, sy-6, 2, 3, color.RGBA{0xff, 0x30, 0x30, 0xff}, false) // comb

		case 1: // Goblin — green with pointy ears
			vector.FillRect(screen, sx-4, sy-2, 8, 10, color.RGBA{0x30, 0x80, 0x30, 0xff}, false)
			vector.FillCircle(screen, sx, sy-5, 4, color.RGBA{0x40, 0xa0, 0x40, 0xff}, false)
			vector.FillRect(screen, sx-6, sy-7, 3, 3, color.RGBA{0x40, 0xa0, 0x40, 0xff}, false) // ear
			vector.FillRect(screen, sx+3, sy-7, 3, 3, color.RGBA{0x40, 0xa0, 0x40, 0xff}, false) // ear
			vector.FillCircle(screen, sx-1.5, sy-5.5, 1, color.RGBA{0xff, 0x30, 0x00, 0xff}, false) // eye
			vector.FillCircle(screen, sx+1.5, sy-5.5, 1, color.RGBA{0xff, 0x30, 0x00, 0xff}, false)

		case 2: // Wolf — gray with snout
			vector.FillRect(screen, sx-5, sy-2, 10, 7, color.RGBA{0x60, 0x60, 0x60, 0xff}, false)
			vector.FillCircle(screen, sx+4, sy-3, 3, color.RGBA{0x70, 0x70, 0x70, 0xff}, false) // head
			vector.FillRect(screen, sx+6, sy-3, 4, 2, color.RGBA{0x50, 0x50, 0x50, 0xff}, false) // snout
			vector.FillRect(screen, sx+3, sy-7, 2, 3, color.RGBA{0x60, 0x60, 0x60, 0xff}, false) // ear
			vector.FillCircle(screen, sx+5, sy-4, 0.8, color.RGBA{0xff, 0xff, 0x00, 0xff}, false) // eye
			vector.FillRect(screen, sx-6, sy, 2, 4, color.RGBA{0x50, 0x50, 0x50, 0xff}, false) // tail

		case 3: // Skeleton — white bones
			vector.FillRect(screen, sx-2, sy-2, 4, 10, color.RGBA{0xd0, 0xd0, 0xc0, 0xff}, false) // spine
			vector.FillCircle(screen, sx, sy-5, 4, color.RGBA{0xd0, 0xd0, 0xc0, 0xff}, false) // skull
			vector.FillCircle(screen, sx-1.5, sy-5.5, 1.2, color.RGBA{0x20, 0x20, 0x20, 0xff}, false) // eye socket
			vector.FillCircle(screen, sx+1.5, sy-5.5, 1.2, color.RGBA{0x20, 0x20, 0x20, 0xff}, false)
			vector.FillRect(screen, sx-5, sy-1, 3, 2, color.RGBA{0xc0, 0xc0, 0xb0, 0xff}, false) // arm
			vector.FillRect(screen, sx+2, sy-1, 3, 2, color.RGBA{0xc0, 0xc0, 0xb0, 0xff}, false)

		case 4: // Dragon — big red with wings
			vector.FillRect(screen, sx-6, sy-3, 12, 10, color.RGBA{0xc0, 0x20, 0x20, 0xff}, false) // body
			vector.FillCircle(screen, sx, sy-7, 5, color.RGBA{0xd0, 0x30, 0x20, 0xff}, false) // head
			// Wings
			vector.StrokeLine(screen, sx-6, sy-3, sx-12, sy-10, 2, color.RGBA{0xa0, 0x20, 0x20, 0xff}, false)
			vector.StrokeLine(screen, sx-12, sy-10, sx-6, sy-1, 1, color.RGBA{0x80, 0x20, 0x20, 0xff}, false)
			vector.StrokeLine(screen, sx+6, sy-3, sx+12, sy-10, 2, color.RGBA{0xa0, 0x20, 0x20, 0xff}, false)
			vector.StrokeLine(screen, sx+12, sy-10, sx+6, sy-1, 1, color.RGBA{0x80, 0x20, 0x20, 0xff}, false)
			// Eyes
			vector.FillCircle(screen, sx-2, sy-8, 1.2, color.RGBA{0xff, 0xff, 0x00, 0xff}, false)
			vector.FillCircle(screen, sx+2, sy-8, 1.2, color.RGBA{0xff, 0xff, 0x00, 0xff}, false)
			// Fire breath animation
			if g.ticks%60 < 30 {
				vector.FillCircle(screen, sx, sy-13, 2, color.RGBA{0xff, 0x80, 0x00, 0x80}, false)
			}

		default:
			vector.FillCircle(screen, sx, sy, 7, def.color, false)
		}

		// HP bar
		barW := float32(18)
		hpFrac := float32(m.hp) / float32(m.maxHP)
		barY := sy - 14
		if m.defIdx == 4 {
			barY = sy - 18
		}
		vector.FillRect(screen, sx-barW/2, barY, barW, 4, color.RGBA{0x20, 0x00, 0x00, 0xcc}, false)
		hpColor := color.RGBA{0x00, 0xc0, 0x00, 0xff}
		if hpFrac < 0.3 {
			hpColor = color.RGBA{0xc0, 0x00, 0x00, 0xff}
		} else if hpFrac < 0.6 {
			hpColor = color.RGBA{0xc0, 0xc0, 0x00, 0xff}
		}
		vector.FillRect(screen, sx-barW/2, barY, barW*hpFrac, 4, hpColor, false)
		vector.StrokeRect(screen, sx-barW/2, barY, barW, 4, 0.5, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
		// Level
		lvlStr := fmt.Sprintf("Lv%d %s", def.level, def.name)
		ebitenutil.DebugPrintAt(screen, lvlStr, int(sx)-len(lvlStr)*3, int(barY)-12)
	}

	// Player — proper character sprite
	px := float32(g.px - g.camX)
	py := float32(g.py - g.camY)
	// Shadow
	vector.FillCircle(screen, px+1, py+7, 6, color.RGBA{0, 0, 0, 0x30}, false)
	// Legs
	vector.FillRect(screen, px-3, py+3, 3, 6, color.RGBA{0x30, 0x30, 0x80, 0xff}, false)
	vector.FillRect(screen, px+1, py+3, 3, 6, color.RGBA{0x30, 0x30, 0x80, 0xff}, false)
	// Body (blue tunic)
	vector.FillRect(screen, px-4, py-3, 9, 8, color.RGBA{0x30, 0x80, 0xd0, 0xff}, false)
	vector.FillRect(screen, px-3, py-2, 7, 6, color.RGBA{0x40, 0x90, 0xe0, 0xff}, false)
	// Belt
	vector.FillRect(screen, px-4, py+2, 9, 2, color.RGBA{0x80, 0x60, 0x20, 0xff}, false)
	// Arms
	vector.FillRect(screen, px-6, py-2, 3, 6, color.RGBA{0xff, 0xd0, 0x90, 0xff}, false)
	vector.FillRect(screen, px+4, py-2, 3, 6, color.RGBA{0xff, 0xd0, 0x90, 0xff}, false)
	// Head
	vector.FillCircle(screen, px, py-6, 5, color.RGBA{0xff, 0xd0, 0x90, 0xff}, false)
	// Hair
	vector.FillCircle(screen, px, py-9, 4, color.RGBA{0x60, 0x30, 0x10, 0xff}, false)
	// Eyes
	vector.FillCircle(screen, px-2, py-6.5, 1, color.RGBA{0x20, 0x20, 0x20, 0xff}, false)
	vector.FillCircle(screen, px+2, py-6.5, 1, color.RGBA{0x20, 0x20, 0x20, 0xff}, false)
	// Mouth
	vector.FillRect(screen, px-1, py-4, 2, 1, color.RGBA{0xc0, 0x60, 0x50, 0xff}, false)
	// Name tag
	vector.FillRect(screen, px-18, py-20, 36, 12, color.RGBA{0, 0, 0, 0x80}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Lv%d Player", g.combatLevel()), int(px)-17, int(py)-19)

	// Skilling progress bar
	if g.skillTimer > 0 {
		barW := float32(40)
		frac := float32(g.skillTimer) / 80.0
		vector.FillRect(screen, px-20, py-28, barW, 6, color.RGBA{0x20, 0x20, 0x20, 0xcc}, false)
		vector.FillRect(screen, px-20, py-28, barW*(1-frac), 6, color.RGBA{0x40, 0xc0, 0x40, 0xff}, false)
		vector.StrokeRect(screen, px-20, py-28, barW, 6, 0.5, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
	}

	// Minimap
	mmX := float32(rsViewW - rsMiniS - 5)
	mmY := float32(5)
	vector.FillRect(screen, mmX, mmY, rsMiniS, float32(rsMiniS*rsMapH/rsMapW), color.RGBA{0, 0, 0, 0xaa}, false)
	scale := float32(rsMiniS) / float32(rsMapW)
	for r := 0; r < rsMapH; r++ {
		for c := 0; c < rsMapW; c++ {
			tc := g.tileColor(g.tileMap[r][c])
			tc.A = 0xcc
			vector.FillRect(screen, mmX+float32(c)*scale, mmY+float32(r)*scale, scale, scale, tc, false)
		}
	}
	// Player dot on minimap
	ppx := mmX + float32(g.px/float64(rsTileS))*scale
	ppy := mmY + float32(g.py/float64(rsTileS))*scale
	vector.FillCircle(screen, ppx, ppy, 2, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
}

func (g *Runescape) tileColor(tile int) color.RGBA {
	switch tile {
	case rsGrass:
		return color.RGBA{0x3a, 0x7d, 0x28, 0xff}
	case rsDirt:
		return color.RGBA{0x8b, 0x73, 0x55, 0xff}
	case rsWater:
		return color.RGBA{0x20, 0x50, 0xb0, 0xff}
	case rsWall:
		return color.RGBA{0x55, 0x55, 0x55, 0xff}
	case rsFloor:
		return color.RGBA{0x70, 0x65, 0x55, 0xff}
	case rsDoor:
		return color.RGBA{0x8b, 0x60, 0x30, 0xff}
	case rsFire:
		return color.RGBA{0x70, 0x65, 0x55, 0xff}
	case rsShopTile:
		return color.RGBA{0x70, 0x65, 0x55, 0xff}
	case rsBankTile:
		return color.RGBA{0x70, 0x65, 0x55, 0xff}
	case rsQuestTile:
		return color.RGBA{0x70, 0x65, 0x55, 0xff}
	default:
		return color.RGBA{0x3a, 0x7d, 0x28, 0xff}
	}
}

func (g *Runescape) drawRightPanel(screen *ebiten.Image) {
	px := float32(rsViewW)
	vector.FillRect(screen, px, 0, rsPanelW, rsScreenH, color.RGBA{0x20, 0x1a, 0x10, 0xff}, false)
	vector.StrokeLine(screen, px, 0, px, rsScreenH, 1, color.RGBA{0x40, 0x35, 0x20, 0xff}, false)

	// Combat level
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Combat Lv: %d", g.combatLevel()), int(px)+5, 5)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HP: %d/%d", g.currentHP(), g.maxHP()), int(px)+5, 18)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Coins: %d", g.itemCount(rsCoins)), int(px)+5, 31)

	// HP bar
	hpFrac := float32(g.currentHP()) / float32(g.maxHP())
	vector.FillRect(screen, px+5, 44, rsPanelW-10, 8, color.RGBA{0x40, 0x00, 0x00, 0xff}, false)
	vector.FillRect(screen, px+5, 44, (rsPanelW-10)*hpFrac, 8, color.RGBA{0x00, 0xb0, 0x00, 0xff}, false)

	// Inventory
	ebitenutil.DebugPrintAt(screen, "Inventory:", int(px)+5, 58)
	for i := 0; i < 28; i++ {
		col := i % rsInvCols
		row := i / rsInvCols
		ix := px + 5 + float32(col*rsInvSlot)
		iy := float32(72 + row*rsInvSlot)
		vector.FillRect(screen, ix, iy, float32(rsInvSlot-2), float32(rsInvSlot-2), color.RGBA{0x30, 0x28, 0x18, 0xff}, false)
		vector.StrokeRect(screen, ix, iy, float32(rsInvSlot-2), float32(rsInvSlot-2), 1, color.RGBA{0x50, 0x45, 0x30, 0xff}, false)
		if g.inv[i].id != rsNoItem {
			ic := rsItemColors[g.inv[i].id]
			vector.FillRect(screen, ix+3, iy+3, float32(rsInvSlot-8), float32(rsInvSlot-8), ic, false)
			if g.inv[i].count > 1 {
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", g.inv[i].count), int(ix)+1, int(iy)+1)
			}
		}
	}

	// Skills summary
	skillY := 72 + (rsInvRows+1)*rsInvSlot
	ebitenutil.DebugPrintAt(screen, "Skills (S):", int(px)+5, skillY)
	for i := 0; i < rsSkillCount; i++ {
		y := skillY + 14 + i*12
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s: %d", rsSkillNames[i][:3], g.skills[i].level), int(px)+5, y)
	}

	// Controls
	ctrlY := rsScreenH - 80
	ebitenutil.DebugPrintAt(screen, "Click=Move/Attack", int(px)+5, ctrlY)
	ebitenutil.DebugPrintAt(screen, "E=Talk B=Shop", int(px)+5, ctrlY+13)
	ebitenutil.DebugPrintAt(screen, "C=Cook Q=Quest", int(px)+5, ctrlY+26)
	ebitenutil.DebugPrintAt(screen, "S=Skills ESC=Back", int(px)+5, ctrlY+39)

	// Skills panel overlay
	if g.state == rsStateSkills {
		g.drawSkillsPanel(screen)
	}
	// Shop overlay
	if g.state == rsStateShop {
		g.drawShopPanel(screen)
	}
}

func (g *Runescape) drawSkillsPanel(screen *ebiten.Image) {
	vector.FillRect(screen, 50, 50, 300, 300, color.RGBA{0x10, 0x10, 0x10, 0xee}, false)
	vector.StrokeRect(screen, 50, 50, 300, 300, 2, color.RGBA{0xc0, 0xa0, 0x40, 0xff}, false)
	ebitenutil.DebugPrintAt(screen, "=== SKILLS ===", 150, 60)
	totalLvl := 0
	for i := 0; i < rsSkillCount; i++ {
		s := g.skills[i]
		totalLvl += s.level
		y := 80 + i*28
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%-12s Lv %-3d  XP: %d / %d",
			rsSkillNames[i], s.level, s.xp, rsXPForLevel(s.level+1)), 70, y)
		// XP bar
		frac := float32(s.xp-rsXPForLevel(s.level)) / float32(rsXPForLevel(s.level+1)-rsXPForLevel(s.level))
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		vector.FillRect(screen, 70, float32(y+12), 200, 5, color.RGBA{0x30, 0x30, 0x30, 0xff}, false)
		vector.FillRect(screen, 70, float32(y+12), 200*frac, 5, color.RGBA{0x40, 0xb0, 0x40, 0xff}, false)
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Total Level: %d", totalLvl), 150, 310)
	ebitenutil.DebugPrintAt(screen, "Press S to close", 150, 330)
}

func (g *Runescape) drawShopPanel(screen *ebiten.Image) {
	vector.FillRect(screen, 40, 60, 320, 200, color.RGBA{0x10, 0x10, 0x10, 0xee}, false)
	vector.StrokeRect(screen, 40, 60, 320, 200, 2, color.RGBA{0xff, 0xd7, 0x00, 0xff}, false)
	ebitenutil.DebugPrintAt(screen, "=== SHOP ===", 150, 70)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Your coins: %d", g.itemCount(rsCoins)), 60, 90)
	ebitenutil.DebugPrintAt(screen, "Press S to sell items from inventory", 60, 110)
	ebitenutil.DebugPrintAt(screen, "Sell prices:", 60, 130)
	ebitenutil.DebugPrintAt(screen, "Logs=5  Ore=10  Raw Fish=8", 60, 145)
	ebitenutil.DebugPrintAt(screen, "Cooked Fish=15  Bones=3  Fur=20", 60, 160)
	ebitenutil.DebugPrintAt(screen, "Dragon Scale=500", 60, 175)
	ebitenutil.DebugPrintAt(screen, "B or ESC to close", 150, 240)
}

func (g *Runescape) drawChatLog(screen *ebiten.Image) {
	chatH := 60
	chatY := rsScreenH - chatH
	vector.FillRect(screen, 0, float32(chatY), rsViewW, float32(chatH), color.RGBA{0, 0, 0, 0xcc}, false)
	lines := g.chatLog
	if len(lines) > 4 {
		lines = lines[len(lines)-4:]
	}
	for i, line := range lines {
		ebitenutil.DebugPrintAt(screen, line, 5, chatY+2+i*14)
	}
}

func (g *Runescape) currentHP() int {
	hp := g.maxHP()
	// Simplified: HP = maxHP minus damage taken this session
	// We track HP via the hitpoints XP in a simplified way
	return hp // simplified for this version
}
