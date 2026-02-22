package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"

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
	gaScreenW = 480
	gaScreenH = 500
)

// Rarity tiers
const (
	gaRarityN = iota
	gaRarityR
	gaRaritySR
	gaRaritySSR
	gaRarityUR
)

var gaRarityNames = []string{"N", "R", "SR", "SSR", "UR"}
var gaRarityRates = []float64{0.50, 0.30, 0.15, 0.04, 0.01}

// Pull costs
const (
	gaSingleCost = 100
	gaMultiCost   = 900
	gaMultiCount  = 10
)

// Pity thresholds
const (
	gaSSRPity = 50
	gaURPity  = 100
)

// Sell values by rarity
var gaSellValues = []int{5, 15, 50, 200, 500}

// Battle gem rewards
const (
	gaWinGemsMin  = 10
	gaWinGemsMax  = 30
	gaLoseGems    = 5
	gaBattleXP    = 20
)

// Elements
const (
	gaElemFire = iota
	gaElemWater
	gaElemEarth
	gaElemLight
	gaElemDark
)

var gaElemNames = []string{"Fire", "Water", "Earth", "Light", "Dark"}
var gaElemColors = []color.RGBA{
	{0xE0, 0x40, 0x30, 0xFF}, // fire red
	{0x30, 0x80, 0xE0, 0xFF}, // water blue
	{0x30, 0xA0, 0x30, 0xFF}, // earth green
	{0xE0, 0xD0, 0x30, 0xFF}, // light yellow
	{0x90, 0x30, 0xC0, 0xFF}, // dark purple
}

// Rarity border colors
var gaRarityColors = []color.RGBA{
	{0x88, 0x88, 0x88, 0xFF}, // N gray
	{0x30, 0x60, 0xE0, 0xFF}, // R blue
	{0xA0, 0x30, 0xD0, 0xFF}, // SR purple
	{0xE0, 0xB0, 0x10, 0xFF}, // SSR gold
	{0xFF, 0x40, 0x80, 0xFF}, // UR rainbow (placeholder, animated)
}

// Screen states
const (
	gaScreenPull = iota
	gaScreenCollection
	gaScreenTeam
	gaScreenBattle
)

// Sort modes
const (
	gaSortRarity = iota
	gaSortElement
	gaSortLevel
	gaSortModes
)

var gaSortNames = []string{"Rarity", "Element", "Level"}

// ---------------------------------------------------------------------------
// Character definition
// ---------------------------------------------------------------------------

type gaCharDef struct {
	name    string
	rarity  int
	element int
	baseATK int
	baseDEF int
	baseHP  int
	bodyClr color.RGBA
}

var gaAllChars = []gaCharDef{
	// N rarity (5)
	{"Slime", gaRarityN, gaElemWater, 10, 12, 18, color.RGBA{0x60, 0xC0, 0x60, 0xFF}},
	{"Goblin", gaRarityN, gaElemEarth, 14, 10, 16, color.RGBA{0x50, 0xA0, 0x30, 0xFF}},
	{"Bat", gaRarityN, gaElemDark, 13, 8, 14, color.RGBA{0x50, 0x30, 0x60, 0xFF}},
	{"Rat", gaRarityN, gaElemEarth, 11, 11, 15, color.RGBA{0x90, 0x80, 0x70, 0xFF}},
	{"Mushroom", gaRarityN, gaElemEarth, 8, 15, 20, color.RGBA{0xC0, 0x80, 0x60, 0xFF}},

	// R rarity (5)
	{"Knight", gaRarityR, gaElemLight, 28, 32, 35, color.RGBA{0xC0, 0xC0, 0xD0, 0xFF}},
	{"Archer", gaRarityR, gaElemFire, 35, 20, 30, color.RGBA{0x40, 0x80, 0x30, 0xFF}},
	{"Mage", gaRarityR, gaElemWater, 38, 18, 25, color.RGBA{0x40, 0x40, 0xC0, 0xFF}},
	{"Healer", gaRarityR, gaElemLight, 15, 25, 40, color.RGBA{0xF0, 0xE0, 0xB0, 0xFF}},
	{"Thief", gaRarityR, gaElemDark, 33, 22, 28, color.RGBA{0x30, 0x30, 0x40, 0xFF}},

	// SR rarity (5)
	{"Dragon Knight", gaRaritySR, gaElemFire, 55, 50, 65, color.RGBA{0xD0, 0x40, 0x20, 0xFF}},
	{"Ice Queen", gaRaritySR, gaElemWater, 60, 45, 55, color.RGBA{0x80, 0xD0, 0xF0, 0xFF}},
	{"Shadow Assassin", gaRaritySR, gaElemDark, 68, 35, 50, color.RGBA{0x20, 0x10, 0x30, 0xFF}},
	{"Holy Paladin", gaRaritySR, gaElemLight, 45, 60, 70, color.RGBA{0xF0, 0xE8, 0xD0, 0xFF}},
	{"Forest Sage", gaRaritySR, gaElemEarth, 50, 55, 60, color.RGBA{0x20, 0x80, 0x40, 0xFF}},

	// SSR rarity (4)
	{"Phoenix Lord", gaRaritySSR, gaElemFire, 85, 70, 90, color.RGBA{0xF0, 0x60, 0x10, 0xFF}},
	{"Void Empress", gaRaritySSR, gaElemDark, 95, 65, 80, color.RGBA{0x40, 0x10, 0x60, 0xFF}},
	{"Thunder God", gaRaritySSR, gaElemLight, 90, 75, 85, color.RGBA{0xF0, 0xE0, 0x30, 0xFF}},
	{"Crystal Dragon", gaRaritySSR, gaElemEarth, 80, 85, 95, color.RGBA{0x60, 0xE0, 0xE0, 0xFF}},

	// UR rarity (2)
	{"Celestial Arbiter", gaRarityUR, gaElemLight, 130, 120, 140, color.RGBA{0xFF, 0xF0, 0xD0, 0xFF}},
	{"Primordial Chaos", gaRarityUR, gaElemDark, 145, 110, 135, color.RGBA{0x30, 0x00, 0x50, 0xFF}},
}

// ---------------------------------------------------------------------------
// Owned character instance
// ---------------------------------------------------------------------------

type gaOwnedChar struct {
	defIdx int // index into gaAllChars
	stars  int // 0 = base, each dupe +1
	level  int // 1+
	xp     int
}

func (c *gaOwnedChar) def() gaCharDef { return gaAllChars[c.defIdx] }

func (c *gaOwnedChar) statMult() float64 {
	lvlMult := 1.0 + float64(c.level-1)*0.05
	starMult := 1.0 + float64(c.stars)*0.10
	return lvlMult * starMult
}

func (c *gaOwnedChar) atk() int { return int(float64(c.def().baseATK) * c.statMult()) }
func (c *gaOwnedChar) def2() int { return int(float64(c.def().baseDEF) * c.statMult()) }
func (c *gaOwnedChar) hp() int  { return int(float64(c.def().baseHP) * c.statMult()) }

func (c *gaOwnedChar) xpToNext() int { return 50 + c.level*30 }

func (c *gaOwnedChar) addXP(amount int) bool {
	c.xp += amount
	leveled := false
	for c.xp >= c.xpToNext() {
		c.xp -= c.xpToNext()
		c.level++
		leveled = true
	}
	return leveled
}

// ---------------------------------------------------------------------------
// Battle unit (enemy or your char in battle)
// ---------------------------------------------------------------------------

type gaBattleUnit struct {
	name    string
	rarity  int
	element int
	atk     int
	def     int
	maxHP   int
	curHP   int
	bodyClr color.RGBA
}

// ---------------------------------------------------------------------------
// Pull animation state
// ---------------------------------------------------------------------------

type gaPullAnim struct {
	active    bool
	tick      int
	phase     int // 0=darken, 1=orb descend, 2=orb glow, 3=reveal, 4=show card
	charIdx   int
	rarity    int
	multiMode bool
	multiList []int // char indices for 10-pull
	multiPos  int   // which card we're showing in multi
	done      bool
}

// ---------------------------------------------------------------------------
// Battle animation state
// ---------------------------------------------------------------------------

type gaBattleAnim struct {
	active     bool
	tick       int
	phase      int // 0=intro, 1=combat rounds, 2=result
	round      int
	won        bool
	gemsEarned int
	xpEarned   int
	yourTeam   []gaBattleUnit
	enemyTeam  []gaBattleUnit
	hitFlash   int
	hitTarget  int // 0-7, 0-3=your, 4-7=enemy
	logLines   []string
}

// ---------------------------------------------------------------------------
// Particle for sparkle effects
// ---------------------------------------------------------------------------

type gaParticle struct {
	x, y   float64
	vx, vy float64
	life   int
	maxL   int
	clr    color.RGBA
}

// ---------------------------------------------------------------------------
// Audio
// ---------------------------------------------------------------------------

var (
	gaBGMPlayer   *audio.Player
	gaSFXPull     []byte
	gaSFXRevealN  []byte
	gaSFXRevealR  []byte
	gaSFXRevealSR []byte
	gaSFXRevealSSR []byte
	gaSFXRevealUR []byte
	gaSFXSell     []byte
	gaSFXHit      []byte
	gaSFXLevelUp  []byte
	gaSFXVictory  []byte
	gaSFXDefeat   []byte
	gaAudioReady  bool
)

func gaInitAudio() {
	if gaAudioReady {
		return
	}
	gaAudioReady = true

	// BGM: Sparkly magical anticipation theme ~110 BPM
	melodyNotes := [][2]int{
		// Intro sparkle (4 bars)
		{72, 1}, {76, 1}, {79, 1}, {84, 1}, {83, 2}, {79, 2},
		{76, 1}, {79, 1}, {83, 1}, {88, 1}, {86, 2}, {84, 2},
		{72, 1}, {74, 1}, {76, 1}, {79, 1}, {83, 4},
		{84, 2}, {83, 1}, {81, 1}, {79, 2}, {0, 2},
		// A section - mystical building (8 bars)
		{60, 2}, {64, 2}, {67, 2}, {72, 2},
		{71, 2}, {72, 1}, {74, 1}, {76, 4},
		{74, 2}, {72, 2}, {71, 2}, {69, 2},
		{67, 4}, {64, 2}, {0, 2},
		{72, 1}, {74, 1}, {76, 2}, {79, 2}, {83, 2},
		{84, 4}, {83, 2}, {81, 2},
		{79, 2}, {76, 2}, {74, 1}, {76, 1}, {79, 2},
		{84, 4}, {0, 2}, {83, 1}, {84, 1},
		// B section - tension build (8 bars)
		{86, 2}, {84, 2}, {83, 2}, {84, 2},
		{86, 4}, {88, 2}, {86, 2},
		{84, 2}, {83, 1}, {81, 1}, {79, 4},
		{76, 2}, {79, 2}, {83, 2}, {0, 2},
		{72, 1}, {76, 1}, {79, 1}, {83, 1}, {84, 2}, {86, 2},
		{88, 4}, {86, 2}, {84, 2},
		{83, 2}, {79, 2}, {76, 2}, {72, 2},
		{74, 2}, {76, 4}, {0, 2},
		// C section - resolution/sparkle (8 bars)
		{84, 2}, {88, 2}, {91, 4},
		{89, 2}, {88, 2}, {86, 2}, {84, 2},
		{83, 2}, {84, 2}, {86, 4},
		{88, 2}, {86, 1}, {84, 1}, {83, 2}, {0, 2},
		{79, 1}, {83, 1}, {84, 1}, {86, 1}, {88, 2}, {91, 2},
		{93, 4}, {91, 2}, {88, 2},
		{86, 2}, {84, 2}, {83, 2}, {79, 2},
		{84, 6}, {0, 2},
	}
	melodyPCM := generateTrack(melodyNotes, sineWave, 110, 0.18)

	// Sparkly arpeggios
	arpNotes := [][2]int{
		{48, 1}, {52, 1}, {55, 1}, {60, 1}, {55, 1}, {52, 1}, {48, 1}, {52, 1},
		{50, 1}, {53, 1}, {57, 1}, {62, 1}, {57, 1}, {53, 1}, {50, 1}, {53, 1},
		{48, 1}, {52, 1}, {55, 1}, {60, 1}, {55, 1}, {52, 1}, {48, 1}, {52, 1},
		{45, 1}, {48, 1}, {52, 1}, {57, 1}, {52, 1}, {48, 1}, {45, 1}, {48, 1},
		// repeat variations
		{48, 1}, {55, 1}, {60, 1}, {64, 1}, {60, 1}, {55, 1}, {48, 1}, {55, 1},
		{50, 1}, {57, 1}, {62, 1}, {65, 1}, {62, 1}, {57, 1}, {50, 1}, {57, 1},
		{52, 1}, {55, 1}, {60, 1}, {64, 1}, {60, 1}, {55, 1}, {52, 1}, {55, 1},
		{48, 1}, {52, 1}, {57, 1}, {60, 1}, {57, 1}, {52, 1}, {48, 1}, {52, 1},
		{48, 1}, {52, 1}, {55, 1}, {60, 1}, {55, 1}, {52, 1}, {48, 1}, {52, 1},
		{53, 1}, {57, 1}, {60, 1}, {65, 1}, {60, 1}, {57, 1}, {53, 1}, {57, 1},
		{48, 1}, {55, 1}, {60, 1}, {67, 1}, {60, 1}, {55, 1}, {48, 1}, {55, 1},
		{45, 1}, {52, 1}, {57, 1}, {64, 1}, {57, 1}, {52, 1}, {45, 1}, {52, 1},
	}
	arpPCM := generateTrack(arpNotes, triangleWave, 110, 0.10)
	drumPCM := generateDrumPattern("K.h.s.h.K.h.s.hxK.h.s.h.K.hxs.hx", 110, 0.12)

	bgm := mixBuffers(melodyPCM, arpPCM)
	if len(drumPCM) <= len(bgm) {
		bgm = mixBuffers(bgm, drumPCM)
	}
	gaBGMPlayer = loopPlayer(bgm, 0.5)

	// SFX: Pull whoosh - ascending shimmer
	{
		dur := 0.6
		buf := pcmStereo(dur)
		for i := 0; i < len(buf)/4; i++ {
			t := float64(i) / float64(sampleRate)
			frac := t / dur
			freq := 400 + 1200*frac
			val := sineWave(t, freq)*0.3 + sineWave(t, freq*2.01)*0.15
			val *= (1 - frac) * 0.5
			writeSample(buf, i*4, val)
		}
		gaSFXPull = buf
	}

	// reveal chords by rarity
	gaRevealChord := func(notes []float64, dur float64, vol float64) []byte {
		buf := pcmStereo(dur)
		for i := 0; i < len(buf)/4; i++ {
			t := float64(i) / float64(sampleRate)
			frac := t / dur
			val := 0.0
			for _, f := range notes {
				val += sineWave(t, f) / float64(len(notes))
			}
			env := math.Exp(-frac * 3)
			writeSample(buf, i*4, val*env*vol)
		}
		return buf
	}
	gaSFXRevealN = gaRevealChord([]float64{262}, 0.3, 0.25)
	gaSFXRevealR = gaRevealChord([]float64{330, 392}, 0.4, 0.3)
	gaSFXRevealSR = gaRevealChord([]float64{330, 415, 523}, 0.5, 0.35)
	gaSFXRevealSSR = gaRevealChord([]float64{330, 415, 523, 659}, 0.7, 0.4)
	gaSFXRevealUR = gaRevealChord([]float64{262, 330, 392, 523, 659, 784}, 1.0, 0.5)

	// Sell clink
	{
		dur := 0.2
		buf := pcmStereo(dur)
		for i := 0; i < len(buf)/4; i++ {
			t := float64(i) / float64(sampleRate)
			frac := t / dur
			val := sineWave(t, 1800)*0.3 + sineWave(t, 2400)*0.2
			val *= math.Exp(-frac * 8)
			writeSample(buf, i*4, val)
		}
		gaSFXSell = buf
	}

	// Hit thud
	{
		dur := 0.15
		buf := pcmStereo(dur)
		for i := 0; i < len(buf)/4; i++ {
			t := float64(i) / float64(sampleRate)
			frac := t / dur
			freq := 100 * math.Exp(-t*15)
			val := sineWave(t, freq)*0.5 + noise()*0.2
			val *= (1 - frac)
			writeSample(buf, i*4, val)
		}
		gaSFXHit = buf
	}

	// Level up fanfare
	{
		notes := [][2]int{{72, 1}, {76, 1}, {79, 1}, {84, 2}}
		gaSFXLevelUp = generateTrack(notes, sineWave, 200, 0.3)
	}

	// Victory
	{
		notes := [][2]int{{60, 1}, {64, 1}, {67, 1}, {72, 1}, {76, 2}, {79, 2}, {84, 4}}
		gaSFXVictory = generateTrack(notes, sineWave, 160, 0.3)
	}

	// Defeat
	{
		notes := [][2]int{{60, 2}, {58, 2}, {55, 2}, {53, 4}}
		gaSFXDefeat = generateTrack(notes, sineWave, 100, 0.25)
	}
}

// ---------------------------------------------------------------------------
// Gacha - main struct
// ---------------------------------------------------------------------------

type Gacha struct {
	goBack bool

	gems      int
	pullCount int // total pulls done
	ssrPity   int // pulls since last SSR
	urPity    int // pulls since last UR

	collection []*gaOwnedChar // all owned
	team       [4]int         // indices into collection, -1 = empty

	screen    int // current screen
	sortMode  int
	scrollY   int // collection scroll
	teamSlot  int // which slot we're picking for

	pullAnim   gaPullAnim
	battleAnim gaBattleAnim
	particles  []gaParticle

	dailyClaimed bool
	tick         int

	// flash overlay
	flashAlpha float64
	shakeX     float64
	shakeY     float64
}

func newGacha() *Gacha {
	gaInitAudio()
	if gaBGMPlayer != nil && !gaBGMPlayer.IsPlaying() {
		gaBGMPlayer.Rewind()
		gaBGMPlayer.Play()
	}

	g := &Gacha{
		gems:   1000,
		screen: gaScreenPull,
	}
	for i := range g.team {
		g.team[i] = -1
	}
	// Give starter characters: a Slime and a Knight
	g.addCharacter(0) // Slime
	g.addCharacter(5) // Knight
	g.team[0] = 0
	g.team[1] = 1
	return g
}

// ---------------------------------------------------------------------------
// Character management
// ---------------------------------------------------------------------------

func (g *Gacha) addCharacter(defIdx int) {
	// Check if we already own this character
	for _, c := range g.collection {
		if c.defIdx == defIdx {
			c.stars++
			return
		}
	}
	g.collection = append(g.collection, &gaOwnedChar{
		defIdx: defIdx,
		stars:  0,
		level:  1,
		xp:     0,
	})
}

func (g *Gacha) rollRarity() int {
	// Check pity
	if g.urPity >= gaURPity-1 {
		return gaRarityUR
	}
	if g.ssrPity >= gaSSRPity-1 {
		return gaRaritySSR
	}
	r := rand.Float64()
	cum := 0.0
	for i, rate := range gaRarityRates {
		cum += rate
		if r < cum {
			return i
		}
	}
	return gaRarityN
}

func (g *Gacha) rollCharacter() int {
	rarity := g.rollRarity()
	g.pullCount++
	g.ssrPity++
	g.urPity++
	if rarity >= gaRaritySSR {
		g.ssrPity = 0
	}
	if rarity >= gaRarityUR {
		g.urPity = 0
	}

	// Pick random character of this rarity
	candidates := []int{}
	for i, ch := range gaAllChars {
		if ch.rarity == rarity {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return 0
	}
	return candidates[rand.Intn(len(candidates))]
}

func (g *Gacha) uniqueCount() int {
	seen := map[int]bool{}
	for _, c := range g.collection {
		seen[c.defIdx] = true
	}
	return len(seen)
}

func (g *Gacha) teamPower() int {
	total := 0
	for _, idx := range g.team {
		if idx >= 0 && idx < len(g.collection) {
			c := g.collection[idx]
			total += c.atk() + c.def2() + c.hp()
		}
	}
	return total
}

func (g *Gacha) teamSize() int {
	n := 0
	for _, idx := range g.team {
		if idx >= 0 && idx < len(g.collection) {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Sort collection
// ---------------------------------------------------------------------------

func (g *Gacha) sortCollection() {
	// We need to remap team indices
	oldMap := map[*gaOwnedChar]bool{}
	teamChars := [4]*gaOwnedChar{}
	for i, idx := range g.team {
		if idx >= 0 && idx < len(g.collection) {
			teamChars[i] = g.collection[idx]
			oldMap[g.collection[idx]] = true
		}
	}

	switch g.sortMode {
	case gaSortRarity:
		sort.SliceStable(g.collection, func(i, j int) bool {
			return g.collection[i].def().rarity > g.collection[j].def().rarity
		})
	case gaSortElement:
		sort.SliceStable(g.collection, func(i, j int) bool {
			return g.collection[i].def().element < g.collection[j].def().element
		})
	case gaSortLevel:
		sort.SliceStable(g.collection, func(i, j int) bool {
			return g.collection[i].level > g.collection[j].level
		})
	}

	// Remap team indices
	for slot := 0; slot < 4; slot++ {
		g.team[slot] = -1
		if teamChars[slot] != nil {
			for j, c := range g.collection {
				if c == teamChars[slot] {
					g.team[slot] = j
					break
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Spawn particles
// ---------------------------------------------------------------------------

func (g *Gacha) spawnSparkles(cx, cy float64, count int, clr color.RGBA) {
	for i := 0; i < count; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := 0.5 + rand.Float64()*2.5
		life := 20 + rand.Intn(40)
		g.particles = append(g.particles, gaParticle{
			x: cx, y: cy,
			vx: math.Cos(angle) * speed,
			vy: math.Sin(angle) * speed,
			life: life, maxL: life,
			clr: clr,
		})
	}
}

// ---------------------------------------------------------------------------
// Start pull animation
// ---------------------------------------------------------------------------

func (g *Gacha) startSinglePull() {
	if g.gems < gaSingleCost {
		return
	}
	if g.pullAnim.active || g.battleAnim.active {
		return
	}
	g.gems -= gaSingleCost
	idx := g.rollCharacter()
	g.pullAnim = gaPullAnim{
		active:    true,
		tick:      0,
		phase:     0,
		charIdx:   idx,
		rarity:    gaAllChars[idx].rarity,
		multiMode: false,
	}
	playSFX(gaSFXPull, 0.5)
}

func (g *Gacha) startMultiPull() {
	if g.gems < gaMultiCost {
		return
	}
	if g.pullAnim.active || g.battleAnim.active {
		return
	}
	g.gems -= gaMultiCost
	chars := make([]int, gaMultiCount)
	hasGuarantee := false
	for i := 0; i < gaMultiCount; i++ {
		chars[i] = g.rollCharacter()
		if gaAllChars[chars[i]].rarity >= gaRaritySR {
			hasGuarantee = true
		}
	}
	// Guarantee at least SR in 10-pull
	if !hasGuarantee {
		// Replace last pull with a random SR+
		srCandidates := []int{}
		for i, ch := range gaAllChars {
			if ch.rarity >= gaRaritySR {
				srCandidates = append(srCandidates, i)
			}
		}
		if len(srCandidates) > 0 {
			chars[gaMultiCount-1] = srCandidates[rand.Intn(len(srCandidates))]
		}
	}
	g.pullAnim = gaPullAnim{
		active:    true,
		tick:      0,
		phase:     0,
		charIdx:   chars[0],
		rarity:    gaAllChars[chars[0]].rarity,
		multiMode: true,
		multiList: chars,
		multiPos:  0,
	}
	playSFX(gaSFXPull, 0.5)
}

// ---------------------------------------------------------------------------
// Start battle
// ---------------------------------------------------------------------------

func (g *Gacha) startBattle() {
	if g.battleAnim.active || g.pullAnim.active {
		return
	}
	if g.teamSize() == 0 {
		return
	}

	ba := gaBattleAnim{
		active: true,
		tick:   0,
		phase:  0,
	}

	// Build your team
	for _, idx := range g.team {
		if idx >= 0 && idx < len(g.collection) {
			c := g.collection[idx]
			ba.yourTeam = append(ba.yourTeam, gaBattleUnit{
				name:    c.def().name,
				rarity:  c.def().rarity,
				element: c.def().element,
				atk:     c.atk(),
				def:     c.def2(),
				maxHP:   c.hp() * 3,
				curHP:   c.hp() * 3,
				bodyClr: c.def().bodyClr,
			})
		}
	}

	// Generate enemy team scaling with power
	power := g.teamPower()
	enemyPower := 0
	for enemyPower < power*70/100 && len(ba.enemyTeam) < 4 {
		idx := rand.Intn(len(gaAllChars))
		ch := gaAllChars[idx]
		scale := 0.6 + rand.Float64()*0.8
		u := gaBattleUnit{
			name:    ch.name,
			rarity:  ch.rarity,
			element: ch.element,
			atk:     int(float64(ch.baseATK) * scale),
			def:     int(float64(ch.baseDEF) * scale),
			maxHP:   int(float64(ch.baseHP) * scale * 3),
			curHP:   int(float64(ch.baseHP) * scale * 3),
			bodyClr: ch.bodyClr,
		}
		ba.enemyTeam = append(ba.enemyTeam, u)
		enemyPower += u.atk + u.def + u.maxHP/3
	}
	if len(ba.enemyTeam) == 0 {
		ch := gaAllChars[0]
		ba.enemyTeam = append(ba.enemyTeam, gaBattleUnit{
			name: ch.name, rarity: ch.rarity, element: ch.element,
			atk: ch.baseATK, def: ch.baseDEF,
			maxHP: ch.baseHP * 3, curHP: ch.baseHP * 3,
			bodyClr: ch.bodyClr,
		})
	}

	g.battleAnim = ba
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (g *Gacha) Update() error {
	g.tick++

	// ESC
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.pullAnim.active || g.battleAnim.active {
			// ignore during animations
		} else {
			if gaBGMPlayer != nil {
				gaBGMPlayer.Pause()
			}
			g.goBack = true
			return nil
		}
	}

	// R restart
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if !g.pullAnim.active && !g.battleAnim.active {
			*g = *newGacha()
			return nil
		}
	}

	// Update particles
	alive := g.particles[:0]
	for i := range g.particles {
		p := &g.particles[i]
		p.x += p.vx
		p.y += p.vy
		p.vy += 0.03
		p.life--
		if p.life > 0 {
			alive = append(alive, *p)
		}
	}
	g.particles = alive

	// Decay flash/shake
	g.flashAlpha *= 0.92
	g.shakeX *= 0.85
	g.shakeY *= 0.85

	// Update pull animation
	if g.pullAnim.active {
		g.updatePullAnim()
		return nil
	}

	// Update battle animation
	if g.battleAnim.active {
		g.updateBattleAnim()
		return nil
	}

	// Navigation: arrow keys or tab for screens
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.screen = (g.screen + 1) % 4
		g.scrollY = 0
		g.teamSlot = -1
	}

	// Number keys for quick nav
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) && g.screen == gaScreenPull {
		g.startSinglePull()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit0) && g.screen == gaScreenPull {
		g.startMultiPull()
	}

	// D for daily
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if !g.dailyClaimed {
			g.dailyClaimed = true
			g.gems += 100
		}
	}

	// B for battle
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.screen = gaScreenBattle
		g.startBattle()
	}

	// T for team
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		g.screen = gaScreenTeam
		g.teamSlot = -1
	}

	// C for collection
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.screen = gaScreenCollection
		g.scrollY = 0
	}

	// P for pull screen
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.screen = gaScreenPull
	}

	// S for sort (in collection)
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && g.screen == gaScreenCollection {
		g.sortMode = (g.sortMode + 1) % gaSortModes
		g.sortCollection()
	}

	// Mouse for click interactions
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		g.handleClick(mx, my)
	}

	// Scroll in collection
	if g.screen == gaScreenCollection {
		_, wy := ebiten.Wheel()
		g.scrollY -= int(wy * 20)
		if g.scrollY < 0 {
			g.scrollY = 0
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
			g.scrollY += 30
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
			g.scrollY -= 30
			if g.scrollY < 0 {
				g.scrollY = 0
			}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Handle click
// ---------------------------------------------------------------------------

func (g *Gacha) handleClick(mx, my int) {
	// Bottom nav buttons
	if my >= gaScreenH-35 {
		bw := gaScreenW / 4
		idx := mx / bw
		if idx >= 0 && idx < 4 {
			g.screen = idx
			g.scrollY = 0
			g.teamSlot = -1
		}
		return
	}

	// Pull screen buttons
	if g.screen == gaScreenPull {
		// Single pull button area
		if mx >= 60 && mx <= 220 && my >= 400 && my <= 435 {
			g.startSinglePull()
		}
		// Multi pull button area
		if mx >= 260 && mx <= 420 && my >= 400 && my <= 435 {
			g.startMultiPull()
		}
		// Daily bonus
		if mx >= 170 && mx <= 310 && my >= 445 && my <= 470 {
			if !g.dailyClaimed {
				g.dailyClaimed = true
				g.gems += 100
			}
		}
		return
	}

	// Collection screen - sell
	if g.screen == gaScreenCollection {
		if len(g.collection) == 0 {
			return
		}
		// Cards are laid out in a grid
		cols := 4
		cardW := 105
		cardH := 130
		padX := 10
		padY := 8
		startY := 60 - g.scrollY
		startX := 10

		for i, c := range g.collection {
			col := i % cols
			row := i / cols
			cx := startX + col*(cardW+padX)
			cy := startY + row*(cardH+padY)

			if mx >= cx && mx < cx+cardW && my >= cy && my < cy+cardH {
				// Check if this is in team - can't sell team members
				inTeam := false
				for _, ti := range g.team {
					if ti == i {
						inTeam = true
						break
					}
				}
				if !inTeam {
					// Sell
					sellVal := gaSellValues[c.def().rarity]
					g.gems += sellVal
					// Remove from collection, fix team indices
					g.collection = append(g.collection[:i], g.collection[i+1:]...)
					for slot := 0; slot < 4; slot++ {
						if g.team[slot] > i {
							g.team[slot]--
						} else if g.team[slot] == i {
							g.team[slot] = -1
						}
					}
					playSFX(gaSFXSell, 0.6)
				}
				return
			}
		}
		return
	}

	// Team screen
	if g.screen == gaScreenTeam {
		// Team slots at top
		slotW := 100
		slotH := 130
		slotStartX := 20
		slotStartY := 50
		slotPad := 10
		for i := 0; i < 4; i++ {
			sx := slotStartX + i*(slotW+slotPad)
			if mx >= sx && mx < sx+slotW && my >= slotStartY && my < slotStartY+slotH {
				if g.teamSlot == i {
					g.teamSlot = -1
				} else {
					g.teamSlot = i
				}
				return
			}
		}
		// Collection below for picking
		if g.teamSlot >= 0 {
			cols := 4
			cardW := 105
			cardH := 100
			padX := 10
			padY := 8
			startY := 210
			startX := 10
			for i := range g.collection {
				col := i % cols
				row := i / cols
				cx := startX + col*(cardW+padX)
				cy := startY + row*(cardH+padY) - g.scrollY
				if mx >= cx && mx < cx+cardW && my >= cy && my < cy+cardH {
					// Check if already in team
					alreadySlot := -1
					for s, ti := range g.team {
						if ti == i {
							alreadySlot = s
							break
						}
					}
					if alreadySlot >= 0 {
						// Swap
						g.team[alreadySlot] = g.team[g.teamSlot]
					}
					g.team[g.teamSlot] = i
					g.teamSlot = -1
					return
				}
			}
		}
		return
	}

	// Battle screen - just press B again or click to start
	if g.screen == gaScreenBattle {
		if !g.battleAnim.active {
			if mx >= 160 && mx <= 320 && my >= 380 && my <= 420 {
				g.startBattle()
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Pull Animation Update
// ---------------------------------------------------------------------------

func (g *Gacha) updatePullAnim() {
	pa := &g.pullAnim
	pa.tick++

	currentRarity := gaAllChars[pa.charIdx].rarity
	if pa.multiMode {
		currentRarity = gaAllChars[pa.multiList[pa.multiPos]].rarity
		pa.charIdx = pa.multiList[pa.multiPos]
	}

	// Phase timing depends on rarity
	revealDelay := 30
	if currentRarity >= gaRaritySR {
		revealDelay = 50
	}
	if currentRarity >= gaRaritySSR {
		revealDelay = 70
	}
	if currentRarity >= gaRarityUR {
		revealDelay = 90
	}

	switch pa.phase {
	case 0: // darken
		if pa.tick > 20 {
			pa.phase = 1
			pa.tick = 0
		}
	case 1: // orb descend
		if pa.tick > 40 {
			pa.phase = 2
			pa.tick = 0
		}
	case 2: // orb glow/color reveal
		if pa.tick > revealDelay {
			pa.phase = 3
			pa.tick = 0
			// Spawn particles based on rarity
			cx := float64(gaScreenW / 2)
			cy := float64(gaScreenH/2 - 20)
			clr := gaRarityColors[currentRarity]
			count := 10 + currentRarity*15
			g.spawnSparkles(cx, cy, count, clr)
			// Flash and shake for high rarity
			if currentRarity >= gaRaritySSR {
				g.flashAlpha = 0.6 + float64(currentRarity-gaRaritySSR)*0.2
				g.shakeX = float64(3 + (currentRarity-gaRaritySSR)*4)
				g.shakeY = float64(2 + (currentRarity-gaRaritySSR)*3)
			}
			// Play reveal SFX
			switch currentRarity {
			case gaRarityN:
				playSFX(gaSFXRevealN, 0.5)
			case gaRarityR:
				playSFX(gaSFXRevealR, 0.5)
			case gaRaritySR:
				playSFX(gaSFXRevealSR, 0.6)
			case gaRaritySSR:
				playSFX(gaSFXRevealSSR, 0.7)
			case gaRarityUR:
				playSFX(gaSFXRevealUR, 0.9)
			}
		}
	case 3: // reveal card
		if pa.tick > 60 {
			pa.phase = 4
			pa.tick = 0
			// Add character to collection
			g.addCharacter(pa.charIdx)
		}
	case 4: // show card, wait for next or done
		waitTime := 60
		if pa.multiMode {
			waitTime = 30
		}
		if pa.tick > waitTime || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if pa.multiMode && pa.multiPos < len(pa.multiList)-1 {
				pa.multiPos++
				pa.charIdx = pa.multiList[pa.multiPos]
				pa.phase = 1
				pa.tick = 0
				playSFX(gaSFXPull, 0.3)
			} else {
				pa.active = false
				pa.done = true
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Battle Animation Update
// ---------------------------------------------------------------------------

func (g *Gacha) updateBattleAnim() {
	ba := &g.battleAnim
	ba.tick++

	switch ba.phase {
	case 0: // intro
		if ba.tick > 40 {
			ba.phase = 1
			ba.tick = 0
			ba.round = 0
		}
	case 1: // combat rounds
		// Every 30 ticks, do a round
		if ba.tick%30 == 0 {
			ba.round++
			// Each alive unit attacks a random alive enemy
			gaDoAttacks := func(attackers, defenders []gaBattleUnit, label string) {
				for i := range attackers {
					if attackers[i].curHP <= 0 {
						continue
					}
					// Find alive defender
					targets := []int{}
					for j := range defenders {
						if defenders[j].curHP > 0 {
							targets = append(targets, j)
						}
					}
					if len(targets) == 0 {
						continue
					}
					t := targets[rand.Intn(len(targets))]
					dmg := attackers[i].atk - defenders[t].def/2 + rand.Intn(10)
					if dmg < 1 {
						dmg = 1
					}
					defenders[t].curHP -= dmg
					if defenders[t].curHP < 0 {
						defenders[t].curHP = 0
					}
					if len(ba.logLines) > 6 {
						ba.logLines = ba.logLines[1:]
					}
					ba.logLines = append(ba.logLines,
						fmt.Sprintf("%s %s->%s -%d", label, attackers[i].name, defenders[t].name, dmg))
				}
			}

			gaDoAttacks(ba.yourTeam, ba.enemyTeam, ">>")
			gaDoAttacks(ba.enemyTeam, ba.yourTeam, "<<")
			ba.hitFlash = 8
			playSFX(gaSFXHit, 0.4)

			// Check victory/defeat
			yourAlive := false
			for _, u := range ba.yourTeam {
				if u.curHP > 0 {
					yourAlive = true
				}
			}
			enemyAlive := false
			for _, u := range ba.enemyTeam {
				if u.curHP > 0 {
					enemyAlive = true
				}
			}

			if !yourAlive || !enemyAlive || ba.round >= 20 {
				ba.phase = 2
				ba.tick = 0
				ba.won = !yourAlive == false && (!enemyAlive || ba.round >= 20 && yourAlive)

				// Determine winner more carefully
				yourHP := 0
				enemyHP := 0
				for _, u := range ba.yourTeam {
					if u.curHP > 0 {
						yourHP += u.curHP
					}
				}
				for _, u := range ba.enemyTeam {
					if u.curHP > 0 {
						enemyHP += u.curHP
					}
				}
				ba.won = yourHP >= enemyHP

				if ba.won {
					ba.gemsEarned = gaWinGemsMin + rand.Intn(gaWinGemsMax-gaWinGemsMin+1)
					ba.xpEarned = gaBattleXP
					g.gems += ba.gemsEarned
					// Give XP to team
					for _, idx := range g.team {
						if idx >= 0 && idx < len(g.collection) {
							if g.collection[idx].addXP(ba.xpEarned) {
								playSFX(gaSFXLevelUp, 0.5)
							}
						}
					}
					playSFX(gaSFXVictory, 0.5)
				} else {
					ba.gemsEarned = gaLoseGems
					g.gems += gaLoseGems
					playSFX(gaSFXDefeat, 0.5)
				}
			}
		}
	case 2: // result
		if ba.tick > 120 || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			ba.active = false
		}
	}

	if ba.hitFlash > 0 {
		ba.hitFlash--
	}
}

// ---------------------------------------------------------------------------
// Draw
// ---------------------------------------------------------------------------

func (g *Gacha) Draw(screen *ebiten.Image) {
	// Apply shake
	if g.shakeX != 0 || g.shakeY != 0 {
		// We'll draw to screen offset by drawing everything translated
	}

	screen.Fill(color.RGBA{0x1a, 0x1a, 0x2e, 0xFF})

	// Pull animation overlay
	if g.pullAnim.active {
		g.drawPullAnim(screen)
		g.drawParticles(screen)
		g.drawFlash(screen)
		return
	}

	// Battle overlay
	if g.battleAnim.active {
		g.drawBattleAnim(screen)
		g.drawParticles(screen)
		return
	}

	// Top bar
	g.drawTopBar(screen)

	// Main content
	switch g.screen {
	case gaScreenPull:
		g.drawPullScreen(screen)
	case gaScreenCollection:
		g.drawCollectionScreen(screen)
	case gaScreenTeam:
		g.drawTeamScreen(screen)
	case gaScreenBattle:
		g.drawBattleScreen(screen)
	}

	// Bottom nav
	g.drawBottomNav(screen)

	// Particles
	g.drawParticles(screen)

	// Flash
	g.drawFlash(screen)
}

// ---------------------------------------------------------------------------
// Draw top bar
// ---------------------------------------------------------------------------

func (g *Gacha) drawTopBar(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, gaScreenW, 30, color.RGBA{0x10, 0x10, 0x20, 0xFF}, false)

	// Gem icon (small diamond shape)
	vector.FillRect(screen, 6, 10, 10, 10, color.RGBA{0x40, 0xD0, 0xF0, 0xFF}, false)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Gems:%d", g.gems), 20, 7)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Pulls:%d", g.pullCount), 120, 7)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Pity SSR:%d/50 UR:%d/100", g.ssrPity, g.urPity), 220, 7)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d/%d chars", g.uniqueCount(), len(gaAllChars)), 420, 7)
}

// ---------------------------------------------------------------------------
// Draw bottom nav
// ---------------------------------------------------------------------------

func (g *Gacha) drawBottomNav(screen *ebiten.Image) {
	labels := []string{"Pull(P)", "Box(C)", "Team(T)", "Battle(B)"}
	bw := float32(gaScreenW / 4)
	y := float32(gaScreenH - 35)

	for i, lbl := range labels {
		x := float32(i) * bw
		bg := color.RGBA{0x30, 0x30, 0x50, 0xFF}
		if i == g.screen {
			bg = color.RGBA{0x50, 0x40, 0x80, 0xFF}
		}
		vector.FillRect(screen, x, y, bw-1, 34, bg, false)
		vector.StrokeRect(screen, x, y, bw-1, 34, 1, color.RGBA{0x60, 0x60, 0x90, 0xFF}, false)
		ebitenutil.DebugPrintAt(screen, lbl, int(x)+int(bw)/2-len(lbl)*3, int(y)+10)
	}
}

// ---------------------------------------------------------------------------
// Draw pull screen
// ---------------------------------------------------------------------------

func (g *Gacha) drawPullScreen(screen *ebiten.Image) {
	// Banner art area
	vector.FillRect(screen, 20, 40, gaScreenW-40, 180, color.RGBA{0x20, 0x15, 0x40, 0xFF}, false)

	// Banner border glow
	t := float64(g.tick) * 0.05
	r := uint8(128 + 80*math.Sin(t))
	gr := uint8(100 + 60*math.Sin(t*1.3))
	b := uint8(180 + 60*math.Sin(t*0.7))
	vector.StrokeRect(screen, 20, 40, gaScreenW-40, 180, 2, color.RGBA{r, gr, b, 0xFF}, false)

	// Banner text
	ebitenutil.DebugPrintAt(screen, "* * * SUMMON BANNER * * *", gaScreenW/2-78, 60)
	ebitenutil.DebugPrintAt(screen, "Featured: All Heroes!", gaScreenW/2-65, 85)

	// Rate display
	ebitenutil.DebugPrintAt(screen, "Rates:", 50, 115)
	rates := []string{"N:50%", "R:30%", "SR:15%", "SSR:4%", "UR:1%"}
	for i, rt := range rates {
		clr := gaRarityColors[i]
		x := 50 + i*80
		// Draw colored label
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), 135)
		op.ColorScale.SetR(float32(clr.R) / 255)
		op.ColorScale.SetG(float32(clr.G) / 255)
		op.ColorScale.SetB(float32(clr.B) / 255)
		img := ebiten.NewImage(60, 16)
		ebitenutil.DebugPrint(img, rt)
		screen.DrawImage(img, op)
	}

	// Featured character showcase (animated)
	showcaseIdx := (g.tick / 120) % len(gaAllChars)
	featured := gaAllChars[showcaseIdx]
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Spotlight: %s (%s)", featured.name, gaRarityNames[featured.rarity]), 50, 170)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Element: %s  ATK:%d DEF:%d HP:%d",
		gaElemNames[featured.element], featured.baseATK, featured.baseDEF, featured.baseHP), 50, 190)

	// Pull info
	ebitenutil.DebugPrintAt(screen, "10-pull guarantees SR or better!", gaScreenW/2-96, 240)

	// Pity info
	if g.ssrPity >= 40 {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SSR PITY CLOSE! %d/50", g.ssrPity), gaScreenW/2-60, 265)
	}
	if g.urPity >= 85 {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("UR PITY CLOSE! %d/100", g.urPity), gaScreenW/2-55, 280)
	}

	// Collection progress
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Collection: %d / %d", g.uniqueCount(), len(gaAllChars)), gaScreenW/2-55, 310)

	// Progress bar
	pct := float32(g.uniqueCount()) / float32(len(gaAllChars))
	vector.FillRect(screen, 120, 330, 240, 12, color.RGBA{0x30, 0x30, 0x40, 0xFF}, false)
	vector.FillRect(screen, 120, 330, 240*pct, 12, color.RGBA{0x50, 0xC0, 0x50, 0xFF}, false)
	vector.StrokeRect(screen, 120, 330, 240, 12, 1, color.RGBA{0x60, 0x60, 0x80, 0xFF}, false)

	// Single pull button
	canSingle := g.gems >= gaSingleCost
	sClr := color.RGBA{0x40, 0x60, 0xA0, 0xFF}
	if !canSingle {
		sClr = color.RGBA{0x30, 0x30, 0x40, 0xFF}
	}
	vector.FillRect(screen, 60, 400, 160, 35, sClr, false)
	vector.StrokeRect(screen, 60, 400, 160, 35, 2, color.RGBA{0x60, 0x80, 0xC0, 0xFF}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("1-Pull [1] %dG", gaSingleCost), 80, 411)

	// Multi pull button
	canMulti := g.gems >= gaMultiCost
	mClr := color.RGBA{0xA0, 0x50, 0x30, 0xFF}
	if !canMulti {
		mClr = color.RGBA{0x30, 0x30, 0x40, 0xFF}
	}
	vector.FillRect(screen, 260, 400, 160, 35, mClr, false)
	vector.StrokeRect(screen, 260, 400, 160, 35, 2, color.RGBA{0xC0, 0x70, 0x40, 0xFF}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("10-Pull [0] %dG", gaMultiCost), 275, 411)

	// Daily bonus button
	dClr := color.RGBA{0x20, 0x80, 0x30, 0xFF}
	dTxt := "Daily +100G [D]"
	if g.dailyClaimed {
		dClr = color.RGBA{0x30, 0x30, 0x40, 0xFF}
		dTxt = "Daily Claimed"
	}
	vector.FillRect(screen, 170, 445, 140, 25, dClr, false)
	vector.StrokeRect(screen, 170, 445, 140, 25, 1, color.RGBA{0x40, 0xA0, 0x50, 0xFF}, false)
	ebitenutil.DebugPrintAt(screen, dTxt, 185, 450)
}

// ---------------------------------------------------------------------------
// Draw pull animation
// ---------------------------------------------------------------------------

func (g *Gacha) drawPullAnim(screen *ebiten.Image) {
	pa := &g.pullAnim

	currentRarity := gaAllChars[pa.charIdx].rarity

	// Dark overlay
	alpha := uint8(180)
	if pa.phase == 0 {
		alpha = uint8(float64(pa.tick) / 20.0 * 180)
	}
	vector.FillRect(screen, 0, 0, gaScreenW, gaScreenH, color.RGBA{0, 0, 0, alpha}, false)

	// Multi-pull progress
	if pa.multiMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Pull %d / %d", pa.multiPos+1, len(pa.multiList)), gaScreenW/2-35, 15)
	}

	cx := float32(gaScreenW / 2)
	cy := float32(gaScreenH/2 - 20)

	switch pa.phase {
	case 0: // darkening
		// Nothing extra
	case 1: // orb descending
		orbY := float32(0) + float32(pa.tick)/40.0*(cy-30)
		orbSize := float32(8 + pa.tick/5)

		// Orb starts white then transitions to rarity color
		frac := float32(pa.tick) / 40.0
		rc := gaRarityColors[currentRarity]
		orbClr := color.RGBA{
			uint8(float32(255)*(1-frac) + float32(rc.R)*frac),
			uint8(float32(255)*(1-frac) + float32(rc.G)*frac),
			uint8(float32(255)*(1-frac) + float32(rc.B)*frac),
			0xFF,
		}
		vector.DrawFilledCircle(screen, cx, orbY, orbSize, orbClr, false)
		// Glow
		for i := 0; i < 3; i++ {
			glowA := uint8(40 - i*12)
			vector.DrawFilledCircle(screen, cx, orbY, orbSize+float32(i+1)*4, color.RGBA{orbClr.R, orbClr.G, orbClr.B, glowA}, false)
		}

	case 2: // orb glowing, building anticipation
		orbSize := float32(12 + pa.tick/3)
		rc := gaRarityColors[currentRarity]

		// UR gets rainbow cycling
		if currentRarity == gaRarityUR {
			t := float64(pa.tick) * 0.15
			rc = color.RGBA{
				uint8(128 + 127*math.Sin(t)),
				uint8(128 + 127*math.Sin(t+2.094)),
				uint8(128 + 127*math.Sin(t+4.189)),
				0xFF,
			}
		}

		// Pulsing
		pulse := float32(1.0 + 0.15*math.Sin(float64(pa.tick)*0.3))
		s := orbSize * pulse

		// Inner orb
		vector.DrawFilledCircle(screen, cx, cy, s, rc, false)

		// Expanding rings
		for ring := 0; ring < 4; ring++ {
			ringR := s + float32(ring+1)*8 + float32(pa.tick%20)
			ringA := uint8(60 - ring*15)
			if ringA > 60 {
				ringA = 0
			}
			vector.StrokeCircle(screen, cx, cy, ringR, 1.5, color.RGBA{rc.R, rc.G, rc.B, ringA}, false)
		}

		// Spawn sparkles continuously for high rarity
		if currentRarity >= gaRaritySR && pa.tick%3 == 0 {
			g.spawnSparkles(float64(cx), float64(cy), 2+currentRarity, rc)
		}

		// Rarity text hint
		if pa.tick > 15 {
			rarityName := gaRarityNames[currentRarity]
			ebitenutil.DebugPrintAt(screen, rarityName+"!", int(cx)-len(rarityName)*3, int(cy)+40)
		}

	case 3, 4: // reveal / show card
		g.drawCharCard(screen, pa.charIdx, int(cx)-55, int(cy)-60, 110, 140, true)

		// Text below
		ch := gaAllChars[pa.charIdx]
		ebitenutil.DebugPrintAt(screen, gaRarityNames[ch.rarity]+" - "+ch.name,
			int(cx)-len(ch.name+gaRarityNames[ch.rarity])*3-6, int(cy)+90)
		ebitenutil.DebugPrintAt(screen,
			fmt.Sprintf("ATK:%d DEF:%d HP:%d", ch.baseATK, ch.baseDEF, ch.baseHP),
			int(cx)-60, int(cy)+105)

		if pa.phase == 4 {
			ebitenutil.DebugPrintAt(screen, "Click / Space to continue", int(cx)-75, int(cy)+130)
		}

		// Continuous sparkles for SSR+
		if currentRarity >= gaRaritySSR && pa.tick%5 == 0 {
			clr := gaRarityColors[currentRarity]
			if currentRarity == gaRarityUR {
				t := float64(g.tick) * 0.1
				clr = color.RGBA{
					uint8(128 + 127*math.Sin(t)),
					uint8(128 + 127*math.Sin(t+2.094)),
					uint8(128 + 127*math.Sin(t+4.189)),
					0xFF,
				}
			}
			g.spawnSparkles(float64(cx)+float64(rand.Intn(120)-60),
				float64(cy)+float64(rand.Intn(140)-70), 3, clr)
		}
	}
}

// ---------------------------------------------------------------------------
// Draw character card
// ---------------------------------------------------------------------------

func (g *Gacha) drawCharCard(screen *ebiten.Image, defIdx, x, y, w, h int, detailed bool) {
	ch := gaAllChars[defIdx]

	// Card background
	vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h),
		color.RGBA{0x20, 0x20, 0x35, 0xFF}, false)

	// Rarity border
	borderClr := gaRarityColors[ch.rarity]
	borderW := float32(2)

	// SSR glow
	if ch.rarity == gaRaritySSR {
		t := float64(g.tick) * 0.08
		glow := uint8(200 + 55*math.Sin(t))
		borderClr = color.RGBA{glow, uint8(float64(glow) * 0.7), 0x10, 0xFF}
		borderW = 3
	}
	// UR rainbow
	if ch.rarity == gaRarityUR {
		t := float64(g.tick) * 0.12
		borderClr = color.RGBA{
			uint8(128 + 127*math.Sin(t)),
			uint8(128 + 127*math.Sin(t+2.094)),
			uint8(128 + 127*math.Sin(t+4.189)),
			0xFF,
		}
		borderW = 3
	}
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), borderW, borderClr, false)

	// Element color strip at top
	elemClr := gaElemColors[ch.element]
	vector.FillRect(screen, float32(x+2), float32(y+2), float32(w-4), 14, elemClr, false)
	ebitenutil.DebugPrintAt(screen, gaElemNames[ch.element], x+4, y+3)

	// Character portrait - unique shapes per type
	cx := float32(x + w/2)
	cy := float32(y + 20 + (h-40)/3)
	g.drawCharPortrait(screen, ch, cx, cy, float32(w)/4)

	// Name
	nameY := y + h - 38
	if detailed {
		nameY = y + h - 50
	}
	ebitenutil.DebugPrintAt(screen, ch.name, x+w/2-len(ch.name)*3, nameY)

	// Stars for owned chars
	owned := g.findOwned(defIdx)
	if owned != nil {
		starStr := ""
		starsToShow := owned.stars + 1
		if starsToShow > 5 {
			starsToShow = 5
		}
		for i := 0; i < starsToShow; i++ {
			starStr += "*"
		}
		if owned.stars > 4 {
			starStr += fmt.Sprintf("+%d", owned.stars-4)
		}
		ebitenutil.DebugPrintAt(screen, starStr, x+w/2-len(starStr)*3, nameY+12)

		if detailed {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Lv%d", owned.level), x+4, nameY+12)
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("A:%d D:%d H:%d", owned.atk(), owned.def2(), owned.hp()),
				x+4, nameY+24)
		}
	}

	// Rarity label
	ebitenutil.DebugPrintAt(screen, gaRarityNames[ch.rarity], x+w-18, y+18)
}

func (g *Gacha) findOwned(defIdx int) *gaOwnedChar {
	for _, c := range g.collection {
		if c.defIdx == defIdx {
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Draw character portrait shapes
// ---------------------------------------------------------------------------

func (g *Gacha) drawCharPortrait(screen *ebiten.Image, ch gaCharDef, cx, cy, size float32) {
	clr := ch.bodyClr

	switch ch.name {
	case "Slime":
		// Round blob
		vector.DrawFilledCircle(screen, cx, cy, size, clr, false)
		vector.DrawFilledCircle(screen, cx-size*0.3, cy-size*0.2, size*0.15, color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}, false)
		vector.DrawFilledCircle(screen, cx+size*0.3, cy-size*0.2, size*0.15, color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}, false)
	case "Goblin":
		// Triangle-ish head + ears
		vector.DrawFilledCircle(screen, cx, cy, size*0.8, clr, false)
		vector.FillRect(screen, cx-size*0.9, cy-size*0.8, size*0.4, size*0.4, clr, false)
		vector.FillRect(screen, cx+size*0.5, cy-size*0.8, size*0.4, size*0.4, clr, false)
	case "Bat":
		// Wings
		vector.DrawFilledCircle(screen, cx, cy, size*0.5, clr, false)
		vector.FillRect(screen, cx-size*1.2, cy-size*0.3, size, size*0.5, clr, false)
		vector.FillRect(screen, cx+size*0.2, cy-size*0.3, size, size*0.5, clr, false)
	case "Rat":
		vector.DrawFilledCircle(screen, cx, cy, size*0.7, clr, false)
		vector.FillRect(screen, cx+size*0.4, cy-size*0.1, size*0.5, size*0.15, clr, false) // tail
		vector.DrawFilledCircle(screen, cx-size*0.3, cy-size*0.6, size*0.25, clr, false) // ear
	case "Mushroom":
		// Cap and stem
		vector.DrawFilledCircle(screen, cx, cy-size*0.3, size*0.8, color.RGBA{0xE0, 0x40, 0x30, 0xFF}, false)
		vector.FillRect(screen, cx-size*0.25, cy, size*0.5, size*0.7, clr, false)
	case "Knight":
		// Shield + helmet
		vector.FillRect(screen, cx-size*0.5, cy-size*0.7, size, size*0.9, clr, false)
		vector.FillRect(screen, cx-size*0.6, cy-size*0.3, size*1.2, size*0.3, color.RGBA{0xA0, 0xA0, 0xB0, 0xFF}, false) // visor
		vector.FillRect(screen, cx-size*0.1, cy-size*1.0, size*0.2, size*0.4, color.RGBA{0xA0, 0xA0, 0xB0, 0xFF}, false) // plume
	case "Archer":
		// Bow shape
		vector.FillRect(screen, cx-size*0.1, cy-size, size*0.2, size*2, color.RGBA{0x80, 0x50, 0x20, 0xFF}, false) // bow
		vector.FillRect(screen, cx-size*0.5, cy, size*1.2, size*0.1, color.RGBA{0x60, 0x40, 0x20, 0xFF}, false) // arrow
		vector.DrawFilledCircle(screen, cx, cy-size*0.5, size*0.35, clr, false) // head
	case "Mage":
		// Hat + robe
		vector.DrawFilledCircle(screen, cx, cy+size*0.2, size*0.6, clr, false)
		vector.FillRect(screen, cx-size*0.5, cy-size*0.5, size, size*0.5, color.RGBA{0x20, 0x20, 0x80, 0xFF}, false)
		vector.DrawFilledCircle(screen, cx, cy-size*0.7, size*0.15, color.RGBA{0xFF, 0xE0, 0x40, 0xFF}, false) // star
	case "Healer":
		// Cross + circle
		vector.DrawFilledCircle(screen, cx, cy, size*0.7, clr, false)
		vector.FillRect(screen, cx-size*0.1, cy-size*0.5, size*0.2, size, color.RGBA{0xE0, 0x30, 0x30, 0xFF}, false)
		vector.FillRect(screen, cx-size*0.5, cy-size*0.1, size, size*0.2, color.RGBA{0xE0, 0x30, 0x30, 0xFF}, false)
	case "Thief":
		// Mask + dagger
		vector.DrawFilledCircle(screen, cx, cy-size*0.2, size*0.5, clr, false)
		vector.FillRect(screen, cx-size*0.6, cy-size*0.35, size*1.2, size*0.2, color.RGBA{0x10, 0x10, 0x20, 0xFF}, false) // mask
		vector.FillRect(screen, cx+size*0.3, cy, size*0.15, size*0.8, color.RGBA{0xC0, 0xC0, 0xC0, 0xFF}, false) // dagger
	case "Dragon Knight":
		// Armored + wings
		vector.FillRect(screen, cx-size*0.4, cy-size*0.6, size*0.8, size*1.2, clr, false)
		vector.FillRect(screen, cx-size*1.0, cy-size*0.5, size*0.5, size*0.6, color.RGBA{0x80, 0x20, 0x10, 0xFF}, false) // wing
		vector.FillRect(screen, cx+size*0.5, cy-size*0.5, size*0.5, size*0.6, color.RGBA{0x80, 0x20, 0x10, 0xFF}, false) // wing
	case "Ice Queen":
		// Crown + flowing shape
		vector.DrawFilledCircle(screen, cx, cy, size*0.7, clr, false)
		vector.FillRect(screen, cx-size*0.5, cy-size*0.9, size*0.2, size*0.35, color.RGBA{0x60, 0xD0, 0xF0, 0xFF}, false)
		vector.FillRect(screen, cx-size*0.1, cy-size*1.0, size*0.2, size*0.45, color.RGBA{0x60, 0xD0, 0xF0, 0xFF}, false)
		vector.FillRect(screen, cx+size*0.3, cy-size*0.9, size*0.2, size*0.35, color.RGBA{0x60, 0xD0, 0xF0, 0xFF}, false)
	case "Shadow Assassin":
		// Dark figure with blade
		vector.DrawFilledCircle(screen, cx, cy-size*0.3, size*0.4, clr, false) // head
		vector.FillRect(screen, cx-size*0.3, cy, size*0.6, size*0.8, clr, false)
		vector.FillRect(screen, cx+size*0.2, cy-size*0.5, size*0.1, size, color.RGBA{0xC0, 0xC0, 0xD0, 0xFF}, false)
	case "Holy Paladin":
		// Shield with cross
		vector.FillRect(screen, cx-size*0.6, cy-size*0.7, size*1.2, size*1.4, clr, false)
		vector.FillRect(screen, cx-size*0.08, cy-size*0.5, size*0.16, size*0.8, color.RGBA{0xFF, 0xD7, 0x00, 0xFF}, false)
		vector.FillRect(screen, cx-size*0.35, cy-size*0.15, size*0.7, size*0.16, color.RGBA{0xFF, 0xD7, 0x00, 0xFF}, false)
	case "Forest Sage":
		// Staff + robe
		vector.FillRect(screen, cx-size*0.3, cy-size*0.3, size*0.6, size, clr, false)
		vector.FillRect(screen, cx-size*0.6, cy-size*0.8, size*0.12, size*1.5, color.RGBA{0x60, 0x40, 0x20, 0xFF}, false) // staff
		vector.DrawFilledCircle(screen, cx-size*0.55, cy-size*0.9, size*0.2, color.RGBA{0x40, 0xE0, 0x40, 0xFF}, false)   // orb
	case "Phoenix Lord":
		// Fire bird
		vector.DrawFilledCircle(screen, cx, cy, size*0.6, clr, false)
		vector.FillRect(screen, cx-size*1.0, cy-size*0.2, size*0.7, size*0.4, color.RGBA{0xF0, 0x80, 0x10, 0xFF}, false) // wing
		vector.FillRect(screen, cx+size*0.3, cy-size*0.2, size*0.7, size*0.4, color.RGBA{0xF0, 0x80, 0x10, 0xFF}, false)
		vector.FillRect(screen, cx-size*0.15, cy-size*1.0, size*0.1, size*0.5, color.RGBA{0xFF, 0x40, 0x10, 0xFF}, false) // flame
		vector.FillRect(screen, cx+size*0.05, cy-size*0.9, size*0.1, size*0.4, color.RGBA{0xFF, 0x60, 0x10, 0xFF}, false)
	case "Void Empress":
		// Dark with void circles
		vector.DrawFilledCircle(screen, cx, cy, size*0.8, clr, false)
		vector.StrokeCircle(screen, cx, cy, size*1.0, 2, color.RGBA{0x80, 0x20, 0xC0, 0xFF}, false)
		vector.StrokeCircle(screen, cx, cy, size*1.2, 1, color.RGBA{0x60, 0x10, 0xA0, 0xFF}, false)
		vector.DrawFilledCircle(screen, cx, cy, size*0.25, color.RGBA{0x10, 0x00, 0x20, 0xFF}, false) // void core
	case "Thunder God":
		// Lightning bolt shape
		vector.FillRect(screen, cx-size*0.4, cy-size*0.8, size*0.6, size*0.4, clr, false) // upper
		vector.FillRect(screen, cx-size*0.2, cy-size*0.4, size*0.6, size*0.4, clr, false) // mid
		vector.FillRect(screen, cx-size*0.4, cy, size*0.6, size*0.5, clr, false)           // lower
		// Lightning spark
		vector.FillRect(screen, cx-size*0.05, cy-size*1.0, size*0.1, size*2.0, color.RGBA{0xFF, 0xFF, 0x40, 0xA0}, false)
	case "Crystal Dragon":
		// Dragon with crystal body
		vector.DrawFilledCircle(screen, cx, cy, size*0.7, clr, false)
		vector.FillRect(screen, cx+size*0.4, cy-size*0.1, size*0.6, size*0.15, clr, false) // tail
		vector.FillRect(screen, cx-size*0.1, cy-size*0.9, size*0.15, size*0.3, color.RGBA{0x80, 0xFF, 0xFF, 0xFF}, false) // horn
		vector.FillRect(screen, cx+size*0.1, cy-size*0.85, size*0.12, size*0.25, color.RGBA{0x80, 0xFF, 0xFF, 0xFF}, false)
	case "Celestial Arbiter":
		// Angelic with halo and wings
		vector.DrawFilledCircle(screen, cx, cy, size*0.5, clr, false)
		vector.StrokeCircle(screen, cx, cy-size*0.7, size*0.3, 2, color.RGBA{0xFF, 0xFF, 0x80, 0xFF}, false) // halo
		vector.FillRect(screen, cx-size*1.2, cy-size*0.3, size*0.6, size*0.7, color.RGBA{0xFF, 0xF0, 0xD0, 0xA0}, false) // wing
		vector.FillRect(screen, cx+size*0.6, cy-size*0.3, size*0.6, size*0.7, color.RGBA{0xFF, 0xF0, 0xD0, 0xA0}, false)
	case "Primordial Chaos":
		// Swirling void with multiple orbits
		vector.DrawFilledCircle(screen, cx, cy, size*0.6, clr, false)
		t := float64(g.tick) * 0.08
		for i := 0; i < 5; i++ {
			angle := t + float64(i)*math.Pi*2/5
			ox := cx + float32(math.Cos(angle)*float64(size)*0.9)
			oy := cy + float32(math.Sin(angle)*float64(size)*0.9)
			oc := color.RGBA{
				uint8(128 + 127*math.Sin(t+float64(i))),
				uint8(50 + 50*math.Sin(t*1.3+float64(i))),
				uint8(180 + 75*math.Sin(t*0.7+float64(i))),
				0xCC,
			}
			vector.DrawFilledCircle(screen, ox, oy, size*0.15, oc, false)
		}
	default:
		// Generic circle
		vector.DrawFilledCircle(screen, cx, cy, size*0.7, clr, false)
	}
}

// ---------------------------------------------------------------------------
// Draw collection screen
// ---------------------------------------------------------------------------

func (g *Gacha) drawCollectionScreen(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Collection (%d owned, %d unique) Sort:%s [S]",
		len(g.collection), g.uniqueCount(), gaSortNames[g.sortMode]), 10, 35)
	ebitenutil.DebugPrintAt(screen, "Click a card to sell (team members protected)", 10, 47)

	if len(g.collection) == 0 {
		ebitenutil.DebugPrintAt(screen, "No characters! Go pull some!", gaScreenW/2-80, gaScreenH/2)
		return
	}

	cols := 4
	cardW := 105
	cardH := 130
	padX := 10
	padY := 8
	startY := 60 - g.scrollY
	startX := 10

	for i, c := range g.collection {
		col := i % cols
		row := i / cols
		cx := startX + col*(cardW+padX)
		cy := startY + row*(cardH+padY)

		if cy+cardH < 30 || cy > gaScreenH-35 {
			continue
		}

		// Highlight team members
		inTeam := false
		for _, ti := range g.team {
			if ti == i {
				inTeam = true
				break
			}
		}
		if inTeam {
			vector.FillRect(screen, float32(cx-2), float32(cy-2), float32(cardW+4), float32(cardH+4),
				color.RGBA{0x40, 0x60, 0x40, 0xFF}, false)
		}

		g.drawCharCard(screen, c.defIdx, cx, cy, cardW, cardH, true)

		// Sell value label
		if !inTeam {
			sellV := gaSellValues[c.def().rarity]
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Sell:%dG", sellV), cx+2, cy+cardH-12)
		} else {
			ebitenutil.DebugPrintAt(screen, "TEAM", cx+cardW-28, cy+cardH-12)
		}
	}

	// Scroll indicator
	maxScroll := len(g.collection)/cols*(cardH+padY) - (gaScreenH - 120)
	if maxScroll > 0 {
		scrollFrac := float32(g.scrollY) / float32(maxScroll)
		barH := float32(gaScreenH-90) * float32(gaScreenH-90) / float32(maxScroll+gaScreenH-90)
		barY := float32(30) + scrollFrac*float32(gaScreenH-90-barH)
		vector.FillRect(screen, gaScreenW-8, barY, 6, barH, color.RGBA{0x60, 0x60, 0x80, 0xA0}, false)
	}
}

// ---------------------------------------------------------------------------
// Draw team screen
// ---------------------------------------------------------------------------

func (g *Gacha) drawTeamScreen(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Team (Power: %d)", g.teamPower()), 10, 35)

	// Team slots
	slotW := 100
	slotH := 130
	slotStartX := 20
	slotStartY := 50
	slotPad := 10

	for i := 0; i < 4; i++ {
		sx := slotStartX + i*(slotW+slotPad)
		sy := slotStartY

		// Slot bg
		bg := color.RGBA{0x20, 0x20, 0x35, 0xFF}
		if i == g.teamSlot {
			bg = color.RGBA{0x40, 0x30, 0x60, 0xFF}
		}
		vector.FillRect(screen, float32(sx), float32(sy), float32(slotW), float32(slotH), bg, false)
		vector.StrokeRect(screen, float32(sx), float32(sy), float32(slotW), float32(slotH), 2,
			color.RGBA{0x60, 0x60, 0x90, 0xFF}, false)

		idx := g.team[i]
		if idx >= 0 && idx < len(g.collection) {
			g.drawCharCard(screen, g.collection[idx].defIdx, sx, sy, slotW, slotH, true)
		} else {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Slot %d", i+1), sx+20, sy+55)
			ebitenutil.DebugPrintAt(screen, "Empty", sx+28, sy+70)
		}
	}

	ebitenutil.DebugPrintAt(screen, "Click slot then click character below", 10, 195)

	// Available characters
	if g.teamSlot >= 0 {
		cols := 4
		cardW := 105
		cardH := 100
		padX := 10
		padY := 8
		startY := 210
		startX := 10

		for i, c := range g.collection {
			col := i % cols
			row := i / cols
			cx := startX + col*(cardW+padX)
			cy := startY + row*(cardH+padY) - g.scrollY

			if cy+cardH < 30 || cy > gaScreenH-35 {
				continue
			}

			// Highlight if already in team
			inTeam := false
			for _, ti := range g.team {
				if ti == i {
					inTeam = true
					break
				}
			}
			if inTeam {
				vector.FillRect(screen, float32(cx-1), float32(cy-1), float32(cardW+2), float32(cardH+2),
					color.RGBA{0x40, 0x60, 0x40, 0xFF}, false)
			}

			g.drawCharCard(screen, c.defIdx, cx, cy, cardW, cardH, false)
		}
	}
}

// ---------------------------------------------------------------------------
// Draw battle screen (idle state)
// ---------------------------------------------------------------------------

func (g *Gacha) drawBattleScreen(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "Battle Arena", gaScreenW/2-36, 35)

	// Team preview
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Your Team (Power: %d)", g.teamPower()), 10, 60)
	for i := 0; i < 4; i++ {
		x := 10 + i*115
		y := 80
		idx := g.team[i]
		if idx >= 0 && idx < len(g.collection) {
			c := g.collection[idx]
			g.drawCharCard(screen, c.defIdx, x, y, 105, 120, true)
		} else {
			vector.FillRect(screen, float32(x), float32(y), 105, 120, color.RGBA{0x20, 0x20, 0x30, 0xFF}, false)
			vector.StrokeRect(screen, float32(x), float32(y), 105, 120, 1, color.RGBA{0x40, 0x40, 0x50, 0xFF}, false)
			ebitenutil.DebugPrintAt(screen, "Empty", x+30, y+50)
		}
	}

	ebitenutil.DebugPrintAt(screen, "Win: 10-30 gems + XP | Lose: 5 gems", gaScreenW/2-110, 220)
	ebitenutil.DebugPrintAt(screen, "Enemy scales with your power", gaScreenW/2-85, 240)

	// Battle info
	ebitenutil.DebugPrintAt(screen, "Element advantage: Fire>Earth>Water>Fire", 10, 280)
	ebitenutil.DebugPrintAt(screen, "Light <-> Dark (mutual advantage)", 10, 296)

	// Recent battle stats
	ebitenutil.DebugPrintAt(screen, "Characters gain XP from battles", 10, 330)
	ebitenutil.DebugPrintAt(screen, "Level up to increase stats!", 10, 346)

	// Battle button
	if g.teamSize() > 0 {
		vector.FillRect(screen, 160, 380, 160, 40, color.RGBA{0xA0, 0x30, 0x30, 0xFF}, false)
		vector.StrokeRect(screen, 160, 380, 160, 40, 2, color.RGBA{0xD0, 0x50, 0x50, 0xFF}, false)
		ebitenutil.DebugPrintAt(screen, "BATTLE! [B]", 195, 393)
	} else {
		ebitenutil.DebugPrintAt(screen, "Add characters to team first!", gaScreenW/2-90, 390)
	}
}

// ---------------------------------------------------------------------------
// Draw battle animation
// ---------------------------------------------------------------------------

func (g *Gacha) drawBattleAnim(screen *ebiten.Image) {
	ba := &g.battleAnim

	screen.Fill(color.RGBA{0x12, 0x12, 0x20, 0xFF})

	ebitenutil.DebugPrintAt(screen, "=== BATTLE ===", gaScreenW/2-42, 5)

	// Draw your team (left side top)
	ebitenutil.DebugPrintAt(screen, "YOUR TEAM", 10, 25)
	for i, u := range ba.yourTeam {
		x := float32(10 + i*115)
		y := float32(40)
		w := float32(105)
		h := float32(80)

		bg := color.RGBA{0x20, 0x30, 0x20, 0xFF}
		if u.curHP <= 0 {
			bg = color.RGBA{0x30, 0x15, 0x15, 0xFF}
		}
		if ba.hitFlash > 0 && ba.hitFlash%2 == 0 && u.curHP > 0 {
			bg = color.RGBA{0x40, 0x40, 0x30, 0xFF}
		}
		vector.FillRect(screen, x, y, w, h, bg, false)
		vector.StrokeRect(screen, x, y, w, h, 1, gaRarityColors[u.rarity], false)

		// HP bar
		hpFrac := float32(u.curHP) / float32(u.maxHP)
		if hpFrac < 0 {
			hpFrac = 0
		}
		barClr := color.RGBA{0x30, 0xC0, 0x30, 0xFF}
		if hpFrac < 0.5 {
			barClr = color.RGBA{0xC0, 0xC0, 0x30, 0xFF}
		}
		if hpFrac < 0.25 {
			barClr = color.RGBA{0xC0, 0x30, 0x30, 0xFF}
		}
		vector.FillRect(screen, x+2, y+h-10, (w-4)*hpFrac, 8, barClr, false)
		vector.StrokeRect(screen, x+2, y+h-10, w-4, 8, 1, color.RGBA{0x60, 0x60, 0x60, 0xFF}, false)

		ebitenutil.DebugPrintAt(screen, u.name, int(x)+4, int(y)+4)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HP:%d/%d", u.curHP, u.maxHP), int(x)+4, int(y)+18)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("A:%d D:%d", u.atk, u.def), int(x)+4, int(y)+32)

		// Element dot
		vector.DrawFilledCircle(screen, x+w-10, y+10, 5, gaElemColors[u.element], false)

		// Portrait mini
		g.drawCharPortrait(screen, gaCharDef{name: u.name, bodyClr: u.bodyClr}, x+w/2, y+55, 10)
	}

	// Draw enemy team (right side bottom)
	ebitenutil.DebugPrintAt(screen, "ENEMY TEAM", 10, 130)
	for i, u := range ba.enemyTeam {
		x := float32(10 + i*115)
		y := float32(145)
		w := float32(105)
		h := float32(80)

		bg := color.RGBA{0x30, 0x20, 0x20, 0xFF}
		if u.curHP <= 0 {
			bg = color.RGBA{0x20, 0x15, 0x15, 0xFF}
		}
		if ba.hitFlash > 0 && ba.hitFlash%2 == 0 && u.curHP > 0 {
			bg = color.RGBA{0x40, 0x30, 0x30, 0xFF}
		}
		vector.FillRect(screen, x, y, w, h, bg, false)
		vector.StrokeRect(screen, x, y, w, h, 1, color.RGBA{0xC0, 0x40, 0x40, 0xFF}, false)

		// HP bar
		hpFrac := float32(u.curHP) / float32(u.maxHP)
		if hpFrac < 0 {
			hpFrac = 0
		}
		barClr := color.RGBA{0xC0, 0x30, 0x30, 0xFF}
		vector.FillRect(screen, x+2, y+h-10, (w-4)*hpFrac, 8, barClr, false)
		vector.StrokeRect(screen, x+2, y+h-10, w-4, 8, 1, color.RGBA{0x60, 0x60, 0x60, 0xFF}, false)

		ebitenutil.DebugPrintAt(screen, u.name, int(x)+4, int(y)+4)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HP:%d/%d", u.curHP, u.maxHP), int(x)+4, int(y)+18)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("A:%d D:%d", u.atk, u.def), int(x)+4, int(y)+32)

		vector.DrawFilledCircle(screen, x+w-10, y+10, 5, gaElemColors[u.element], false)
		g.drawCharPortrait(screen, gaCharDef{name: u.name, bodyClr: u.bodyClr}, x+w/2, y+55, 10)
	}

	// Battle log
	vector.FillRect(screen, 5, 235, gaScreenW-10, 145, color.RGBA{0x10, 0x10, 0x18, 0xFF}, false)
	vector.StrokeRect(screen, 5, 235, gaScreenW-10, 145, 1, color.RGBA{0x40, 0x40, 0x60, 0xFF}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Round %d", ba.round), gaScreenW/2-25, 238)
	for i, line := range ba.logLines {
		ebitenutil.DebugPrintAt(screen, line, 12, 254+i*16)
	}

	// VS indicator
	if ba.phase == 0 {
		ebitenutil.DebugPrintAt(screen, "VS", gaScreenW/2-6, 125)
	}

	// Result
	if ba.phase == 2 {
		resultBg := color.RGBA{0x10, 0x40, 0x10, 0xD0}
		resultTxt := "VICTORY!"
		if !ba.won {
			resultBg = color.RGBA{0x40, 0x10, 0x10, 0xD0}
			resultTxt = "DEFEAT..."
		}
		vector.FillRect(screen, 100, 400, 280, 80, resultBg, false)
		vector.StrokeRect(screen, 100, 400, 280, 80, 2, color.RGBA{0xFF, 0xFF, 0xFF, 0x80}, false)
		ebitenutil.DebugPrintAt(screen, resultTxt, gaScreenW/2-24, 410)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("+%d gems", ba.gemsEarned), gaScreenW/2-24, 430)
		if ba.won {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("+%d XP to team", ba.xpEarned), gaScreenW/2-40, 448)
		}
		ebitenutil.DebugPrintAt(screen, "Click/Space to continue", gaScreenW/2-65, 465)
	}
}

// ---------------------------------------------------------------------------
// Draw particles
// ---------------------------------------------------------------------------

func (g *Gacha) drawParticles(screen *ebiten.Image) {
	for _, p := range g.particles {
		alpha := float64(p.life) / float64(p.maxL)
		sz := float32(1 + 3*alpha)
		clr := color.RGBA{p.clr.R, p.clr.G, p.clr.B, uint8(255 * alpha)}
		vector.DrawFilledCircle(screen, float32(p.x), float32(p.y), sz, clr, false)
	}
}

// ---------------------------------------------------------------------------
// Draw flash overlay
// ---------------------------------------------------------------------------

func (g *Gacha) drawFlash(screen *ebiten.Image) {
	if g.flashAlpha > 0.01 {
		a := uint8(g.flashAlpha * 255)
		vector.FillRect(screen, 0, 0, gaScreenW, gaScreenH, color.RGBA{0xFF, 0xFF, 0xFF, a}, false)
	}
}

// ---------------------------------------------------------------------------
// Layout & WantsBack
// ---------------------------------------------------------------------------

func (g *Gacha) Layout(w, h int) (int, int) {
	return gaScreenW, gaScreenH
}

func (g *Gacha) WantsBack() bool {
	return g.goBack
}
