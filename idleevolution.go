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
// Idle Evolution Audio
// ---------------------------------------------------------------------------

var (
	ieBGMPlayer   *audio.Player
	ieSFXClick    []byte
	ieSFXEvolve   []byte
	ieSFXUpgrade  []byte
	ieAudioReady  bool
)

func ieInitAudio() {
	if ieAudioReady {
		return
	}
	ieAudioReady = true

	// --- BGM: Evolving ambient soundscape (~70 BPM) ---
	// Ethereal sine pads with slow-moving harmonies, cosmic feel
	// Low drone pad in E minor — very slow, spacey
	padNotes := [][2]int{
		// A section - spacey Em drones
		{40, 8}, {0, 2}, {43, 8}, {0, 2},
		{45, 8}, {0, 2}, {43, 6}, {40, 4},
		{38, 8}, {0, 2}, {40, 6}, {43, 4},
		{47, 8}, {0, 2}, {45, 6}, {0, 4},
		{40, 6}, {38, 4}, {36, 6}, {0, 4},
		{43, 8}, {0, 2}, {47, 8}, {0, 2},
		{48, 6}, {47, 4}, {45, 6}, {0, 4},
		{40, 8}, {0, 4}, {43, 4}, {40, 4},
		// B section - Am drift, deeper evolution
		{45, 8}, {0, 2}, {48, 8}, {0, 2},
		{43, 8}, {0, 2}, {40, 6}, {38, 4},
		{36, 8}, {0, 2}, {38, 6}, {40, 4},
		{43, 8}, {0, 2}, {45, 6}, {0, 4},
		// C section - Cmaj resolution, emergence
		{48, 8}, {0, 2}, {52, 8}, {0, 2},
		{48, 6}, {47, 4}, {45, 6}, {0, 4},
		{43, 8}, {0, 2}, {40, 8}, {0, 2},
		{40, 8}, {0, 4}, {43, 4}, {40, 4},
	}
	padPCM := generateTrack(padNotes, sineWave, 70, 0.28)

	// High ethereal overtones — sine harmonics floating above
	overNotes := [][2]int{
		// A section overtones
		{0, 6}, {76, 4}, {0, 4}, {79, 4}, {0, 2},
		{83, 6}, {0, 4}, {81, 4}, {0, 6},
		{0, 4}, {79, 4}, {76, 4}, {0, 8},
		{74, 4}, {0, 2}, {76, 6}, {0, 4}, {79, 2}, {0, 2},
		{0, 8}, {83, 4}, {81, 2}, {79, 2}, {0, 4},
		{76, 6}, {0, 6}, {74, 4}, {0, 4},
		{72, 4}, {0, 2}, {74, 4}, {76, 6}, {0, 4},
		{79, 4}, {0, 4}, {76, 4}, {0, 4}, {72, 4},
		// B section - Am overtones, slower
		{0, 8}, {69, 4}, {0, 4}, {72, 4}, {0, 2},
		{76, 6}, {0, 6}, {74, 4}, {0, 4},
		{0, 4}, {72, 4}, {0, 4}, {69, 4}, {0, 4},
		{76, 4}, {0, 2}, {79, 6}, {0, 4}, {76, 2}, {0, 2},
		// C section - brighter Cmaj
		{0, 6}, {84, 4}, {0, 4}, {79, 4}, {0, 2},
		{76, 6}, {0, 4}, {74, 4}, {0, 6},
		{0, 4}, {72, 4}, {76, 4}, {0, 8},
		{79, 4}, {0, 4}, {76, 4}, {0, 4}, {72, 4},
	}
	overPCM := generateTrack(overNotes, sineWave, 70, 0.08)

	// Very subtle percussion — almost like distant heartbeat
	drumPattern := "K.............h.........K.............h.........K...........h...........K.............h.......s..."
	drumPCM := generateDrumPattern(drumPattern, 70, 0.08)

	bgmMix := mixBuffers(padPCM, overPCM)
	for len(drumPCM) < len(bgmMix) {
		drumPCM = append(drumPCM, drumPCM...)
	}
	drumPCM = drumPCM[:len(bgmMix)]
	bgmMix = mixBuffers(bgmMix, drumPCM)

	ieBGMPlayer = loopPlayer(bgmMix, 0.12)

	// --- SFX: Click (organic blip) ---
	{
		dur := 0.06
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			// Organic blip: sine with quick pitch drop
			freq := 600 * math.Exp(-t*20)
			env := math.Exp(-t * 40)
			val := sineWave(t, freq) * env * 0.35
			writeSample(buf, i*4, val)
		}
		ieSFXClick = buf
	}

	// --- SFX: Evolve (dramatic ascending chord) ---
	{
		// E minor chord ascending: E4 G4 B4 E5 G5 — wide sweep
		notes := [][2]int{{64, 2}, {67, 2}, {71, 2}, {76, 2}, {79, 4}}
		ieSFXEvolve = generateTrack(notes, sineWave, 300, 0.22)
	}

	// --- SFX: Upgrade (confirmation tone) ---
	{
		dur := 0.15
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			env := envelope(float64(i)/float64(samples), 0.05, 0.4, 0.55)
			// Two-tone confirmation: E5 then G5
			val := 0.0
			if t < 0.07 {
				val = sineWave(t, noteFreq(76)) * env * 0.3
			} else {
				val = sineWave(t, noteFreq(79)) * env * 0.3
			}
			writeSample(buf, i*4, val)
		}
		ieSFXUpgrade = buf
	}
}

// Screen dimensions
const (
	ieScreenW = 500
	ieScreenH = 450
)

// Evolution stages
const (
	ieStageSoup       = 0
	ieStageSingleCell = 1
	ieStageMulti      = 2
	ieStageFish       = 3
	ieStageAmphibian  = 4
	ieStageReptile    = 5
	ieStageMammal     = 6
	ieStagePrimate    = 7
	ieStageHuman      = 8
	ieStageSpace      = 9
	ieStageCount      = 10
)

// DNA thresholds for each stage
var ieStageThresholds = [ieStageCount]float64{
	0,       // Soup
	100,     // Single Cell
	1e3,     // Multicellular
	1e4,     // Fish
	1e5,     // Amphibian
	1e6,     // Reptile
	1e7,     // Mammal
	1e8,     // Primate
	1e9,     // Human
	1e11,    // Space Age
}

var ieStageNames = [ieStageCount]string{
	"Primordial Soup",
	"Single Cell",
	"Multicellular",
	"Fish",
	"Amphibian",
	"Reptile",
	"Mammal",
	"Primate",
	"Human",
	"Space Age",
}

// Background colors per stage (R, G, B)
var ieStageBgColors = [ieStageCount][3]uint8{
	{10, 30, 50},    // dark blue-green
	{15, 40, 55},    // slightly lighter
	{20, 50, 45},    // more green
	{15, 45, 70},    // ocean blue
	{25, 55, 40},    // swampy green
	{45, 50, 30},    // earthy
	{40, 55, 35},    // forest green
	{50, 45, 35},    // brown-ish
	{35, 35, 55},    // urban blue
	{10, 10, 30},    // deep space
}

// Click DNA per stage
var ieClickDNA = [ieStageCount]float64{
	1, 3, 10, 30, 100, 500, 2000, 8000, 50000, 500000,
}

// Auto-gen DNA per second per stage (base, before multipliers)
var ieAutoBase = [ieStageCount]float64{
	0, 1, 5, 25, 150, 800, 5000, 30000, 200000, 2000000,
}

// Mutation upgrades
const ieMutationCount = 8

type ieMutation struct {
	name     string
	baseCost float64
	multi    float64 // multiplier to DNA production
	stage    int     // stage required to unlock
}

var ieMutations = [ieMutationCount]ieMutation{
	{"Stronger Membrane", 50, 1.5, ieStageSingleCell},
	{"Efficient Mitosis", 500, 2.0, ieStageMulti},
	{"Gill Enhancement", 5000, 2.5, ieStageFish},
	{"Lung Capacity", 50000, 3.0, ieStageAmphibian},
	{"Thick Scales", 500000, 3.5, ieStageReptile},
	{"Warm Blood", 5e6, 4.0, ieStageMammal},
	{"Tool Use", 5e7, 5.0, ieStagePrimate},
	{"Written Language", 5e8, 8.0, ieStageHuman},
}

// Generator types
const ieGenCount = 5

type ieGenerator struct {
	name     string
	baseCost float64
	baseDPS  float64 // DNA per second per unit
	stage    int     // stage required to unlock
}

var ieGenerators = [ieGenCount]ieGenerator{
	{"RNA Strand", 20, 0.5, ieStageSingleCell},
	{"Enzyme", 200, 3.0, ieStageMulti},
	{"Organ", 3000, 20.0, ieStageFish},
	{"Brain Cell", 80000, 150.0, ieStageReptile},
	{"Laboratory", 5e6, 2000.0, ieStageHuman},
}

type ieParticle struct {
	x, y   float64
	vx, vy float64
	life   int
	r, g, b uint8
}

// IdleEvolution is the main game state.
type IdleEvolution struct {
	goBack bool

	dna         float64
	totalDNA    float64
	stage       int
	tickCount   int
	timePlayed  int // in seconds (approx, ticks/60)

	// click feedback
	clickAnim   int
	clickX      int
	clickY      int

	// stage transition flash
	flashTimer  int

	// mutations purchased count per mutation
	mutBought   [ieMutationCount]int

	// generators owned count per generator
	genOwned    [ieGenCount]int

	// scroll offset for upgrades panel
	scrollY     int

	// particles for visual flair
	particles   []ieParticle

	// creature bob animation
	bobPhase    float64

	// win state
	won         bool
}

func newIdleEvolution() *IdleEvolution {
	g := &IdleEvolution{}
	ieInitAudio()
	if ieBGMPlayer != nil && !ieBGMPlayer.IsPlaying() {
		ieBGMPlayer.Play()
	}
	return g
}

func (g *IdleEvolution) WantsBack() bool {
	return g.goBack
}

func (g *IdleEvolution) Layout(w, h int) (int, int) {
	return ieScreenW, ieScreenH
}

// ---------- helpers ----------

func ieFormatNum(n float64) string {
	if n < 0 {
		return "0"
	}
	switch {
	case n >= 1e12:
		return fmt.Sprintf("%.1fT", n/1e12)
	case n >= 1e9:
		return fmt.Sprintf("%.1fB", n/1e9)
	case n >= 1e6:
		return fmt.Sprintf("%.1fM", n/1e6)
	case n >= 1e3:
		return fmt.Sprintf("%.1fK", n/1e3)
	default:
		return fmt.Sprintf("%.0f", n)
	}
}

func ieLerpU8(a, b uint8, t float64) uint8 {
	return uint8(float64(a)*(1-t) + float64(b)*t)
}

func ieMutCost(idx int, bought int) float64 {
	return ieMutations[idx].baseCost * math.Pow(3.5, float64(bought))
}

func ieGenCost(idx int, owned int) float64 {
	return ieGenerators[idx].baseCost * math.Pow(1.15, float64(owned))
}

func (g *IdleEvolution) ieTotalMultiplier() float64 {
	m := 1.0
	for i := 0; i < ieMutationCount; i++ {
		for j := 0; j < g.mutBought[i]; j++ {
			m *= ieMutations[i].multi
		}
	}
	return m
}

func (g *IdleEvolution) ieGenDPS() float64 {
	total := 0.0
	for i := 0; i < ieGenCount; i++ {
		total += float64(g.genOwned[i]) * ieGenerators[i].baseDPS
	}
	return total
}

func (g *IdleEvolution) ieDNAPerSecond() float64 {
	base := ieAutoBase[g.stage] + g.ieGenDPS()
	return base * g.ieTotalMultiplier()
}

func (g *IdleEvolution) ieMutsPurchased() int {
	total := 0
	for i := 0; i < ieMutationCount; i++ {
		total += g.mutBought[i]
	}
	return total
}

func (g *IdleEvolution) ieCurrentStage() int {
	s := 0
	for i := ieStageCount - 1; i >= 0; i-- {
		if g.totalDNA >= ieStageThresholds[i] {
			s = i
			break
		}
	}
	return s
}

func (g *IdleEvolution) ieNextThreshold() float64 {
	if g.stage >= ieStageCount-1 {
		return ieStageThresholds[ieStageCount-1]
	}
	return ieStageThresholds[g.stage+1]
}

func (g *IdleEvolution) ieStageProgress() float64 {
	if g.stage >= ieStageCount-1 {
		return 1.0
	}
	cur := ieStageThresholds[g.stage]
	nxt := ieStageThresholds[g.stage+1]
	if nxt <= cur {
		return 1.0
	}
	p := (g.totalDNA - cur) / (nxt - cur)
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	return p
}

func (g *IdleEvolution) ieSpawnParticles(x, y float64, count int, r, g2, b uint8) {
	for i := 0; i < count; i++ {
		g.particles = append(g.particles, ieParticle{
			x: x, y: y,
			vx: (rand.Float64() - 0.5) * 4,
			vy: (rand.Float64() - 0.5) * 4,
			life: 20 + rand.Intn(30),
			r: r, g: g2, b: b,
		})
	}
}

// ---------- Update ----------

func (g *IdleEvolution) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if ieBGMPlayer != nil && ieBGMPlayer.IsPlaying() {
			ieBGMPlayer.Pause()
		}
		g.goBack = true
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		*g = IdleEvolution{}
		if ieBGMPlayer != nil && !ieBGMPlayer.IsPlaying() {
			ieBGMPlayer.Play()
		}
		return nil
	}

	// Cheats: M = 1M DNA, B = 1B DNA, G = 10x current DNA
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.dna += 1_000_000
		g.totalDNA += 1_000_000
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.dna += 1_000_000_000
		g.totalDNA += 1_000_000_000
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		add := g.dna * 9
		g.dna += add
		g.totalDNA += add
	}

	if g.won {
		return nil
	}

	g.tickCount++
	if g.tickCount%60 == 0 {
		g.timePlayed++
	}

	g.bobPhase += 0.05

	// Auto-generate DNA
	dps := g.ieDNAPerSecond()
	earned := dps / 60.0
	g.dna += earned
	g.totalDNA += earned

	// Check stage advancement
	newStage := g.ieCurrentStage()
	if newStage > g.stage {
		g.stage = newStage
		g.flashTimer = 30
		// big particle burst for evolution
		g.ieSpawnParticles(250, 180, 60, 255, 255, 100)
		if ieSFXEvolve != nil {
			playSFX(ieSFXEvolve, 0.35)
		}
		if g.stage == ieStageSpace {
			g.won = true
		}
	}

	// Click / Space to earn DNA
	clicked := false
	clickX, clickY := 250, 180

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		// Check if click is in the creature area (central area)
		if mx >= 150 && mx <= 350 && my >= 80 && my <= 280 {
			clicked = true
			clickX = mx
			clickY = my
		}
		// Check upgrade buttons (right panel)
		g.ieHandleUpgradeClick(mx, my)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		clicked = true
	}

	if clicked {
		amount := ieClickDNA[g.stage] * g.ieTotalMultiplier()
		g.dna += amount
		g.totalDNA += amount
		g.clickAnim = 15
		g.clickX = clickX
		g.clickY = clickY
		g.ieSpawnParticles(float64(clickX), float64(clickY), 8, 100, 255, 150)
		if ieSFXClick != nil {
			playSFX(ieSFXClick, 0.20)
		}
	}

	if g.clickAnim > 0 {
		g.clickAnim--
	}
	if g.flashTimer > 0 {
		g.flashTimer--
	}

	// scroll upgrade panel
	_, wy := ebiten.Wheel()
	g.scrollY -= int(wy * 20)
	if g.scrollY < 0 {
		g.scrollY = 0
	}
	maxScroll := 600 // approximate max scroll
	if g.scrollY > maxScroll {
		g.scrollY = maxScroll
	}

	// Update particles
	alive := g.particles[:0]
	for i := range g.particles {
		p := &g.particles[i]
		p.life--
		if p.life <= 0 {
			continue
		}
		p.x += p.vx
		p.y += p.vy
		p.vy += 0.05 // slight gravity
		alive = append(alive, *p)
	}
	g.particles = alive

	return nil
}

func (g *IdleEvolution) ieHandleUpgradeClick(mx, my int) {
	// Upgrades panel is on the right side: x=[310..490], y=[80..440]
	panelX := 310
	panelY := 80
	panelW := 180

	// Generators section
	yOff := panelY - g.scrollY
	yOff += 18 // header

	for i := 0; i < ieGenCount; i++ {
		if ieGenerators[i].stage > g.stage {
			continue
		}
		btnX := panelX
		btnY := yOff
		btnH := 34
		if mx >= btnX && mx < btnX+panelW && my >= btnY && my < btnY+btnH && my >= panelY && my < 440 {
			cost := ieGenCost(i, g.genOwned[i])
			if g.dna >= cost {
				g.dna -= cost
				g.genOwned[i]++
				if ieSFXUpgrade != nil {
					playSFX(ieSFXUpgrade, 0.25)
				}
			}
		}
		yOff += 36
	}

	yOff += 18 // "Mutations" header

	for i := 0; i < ieMutationCount; i++ {
		if ieMutations[i].stage > g.stage {
			continue
		}
		btnX := panelX
		btnY := yOff
		btnH := 34
		if mx >= btnX && mx < btnX+panelW && my >= btnY && my < btnY+btnH && my >= panelY && my < 440 {
			cost := ieMutCost(i, g.mutBought[i])
			if g.dna >= cost {
				g.dna -= cost
				g.mutBought[i]++
				if ieSFXUpgrade != nil {
					playSFX(ieSFXUpgrade, 0.25)
				}
			}
		}
		yOff += 36
	}
}

// ---------- Draw ----------

func (g *IdleEvolution) Draw(screen *ebiten.Image) {
	// Background gradient
	g.ieDrawBackground(screen)

	// Stage flash overlay — dramatic multi-layer flash
	if g.flashTimer > 0 {
		progress := float64(g.flashTimer) / 30.0
		// Core white flash
		alpha := uint8(progress * 200)
		vector.FillRect(screen, 0, 0, ieScreenW, ieScreenH, color.RGBA{255, 255, 200, alpha}, false)
		// Expanding ring from center
		ringR := float32((1.0 - progress) * 300)
		ringAlpha := uint8(progress * 150)
		vector.StrokeCircle(screen, ieScreenW/2, 180, ringR, 3,
			color.RGBA{255, 255, 100, ringAlpha}, false)
		vector.StrokeCircle(screen, ieScreenW/2, 180, ringR*0.7, 2,
			color.RGBA{255, 200, 50, ringAlpha}, false)
	}

	if g.won {
		g.ieDrawWinScreen(screen)
		return
	}

	// Header: stage name
	stageName := ieStageNames[g.stage]
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Stage %d: %s", g.stage+1, stageName), 10, 8)

	// DNA counter and rate
	dps := g.ieDNAPerSecond()
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DNA: %s  (+%s/s)", ieFormatNum(g.dna), ieFormatNum(dps)), 10, 24)

	// Click info
	clickAmt := ieClickDNA[g.stage] * g.ieTotalMultiplier()
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Click: +%s DNA  [Space/Click]", ieFormatNum(clickAmt)), 10, 40)

	// Progress bar to next stage
	g.ieDrawProgressBar(screen)

	// DNA helix visualization on the left side
	g.ieDrawDNAHelix(screen)

	// Ambient particles matching current stage around creature area
	g.ieDrawStageParticles(screen)

	// Creature display area
	g.ieDrawCreature(screen)

	// Click animation popup
	if g.clickAnim > 0 {
		amt := ieClickDNA[g.stage] * g.ieTotalMultiplier()
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("+%s", ieFormatNum(amt)),
			g.clickX-15, g.clickY-20-g.clickAnim)
	}

	// Particles
	for _, p := range g.particles {
		alpha := uint8(float64(p.life) / 50.0 * 255)
		if alpha > 255 {
			alpha = 255
		}
		vector.FillRect(screen, float32(p.x)-1, float32(p.y)-1, 3, 3,
			color.RGBA{p.r, p.g, p.b, alpha}, false)
	}

	// Upgrade panel (right side)
	g.ieDrawUpgradePanel(screen)

	// Stats (bottom)
	g.ieDrawStats(screen)
}

func (g *IdleEvolution) ieDrawBackground(screen *ebiten.Image) {
	// Gradient between current stage color and next stage color
	cur := ieStageBgColors[g.stage]
	nxtIdx := g.stage + 1
	if nxtIdx >= ieStageCount {
		nxtIdx = g.stage
	}
	nxt := ieStageBgColors[nxtIdx]
	t := g.ieStageProgress()

	r := ieLerpU8(cur[0], nxt[0], t)
	gc := ieLerpU8(cur[1], nxt[1], t)
	b := ieLerpU8(cur[2], nxt[2], t)

	// Breathing effect: background slowly shifts brightness
	breathe := math.Sin(float64(g.tickCount)*0.015) * 8

	// Draw vertical gradient with breathing variation
	for y := 0; y < ieScreenH; y += 4 {
		frac := float64(y) / float64(ieScreenH)
		dr := uint8(math.Max(0, math.Min(255, float64(r)-frac*20+breathe)))
		dg := uint8(math.Max(0, math.Min(255, float64(gc)-frac*15+breathe*0.7)))
		db := uint8(math.Max(0, math.Min(255, float64(b)+frac*20+breathe*0.5)))
		vector.FillRect(screen, 0, float32(y), ieScreenW, 4,
			color.RGBA{dr, dg, db, 255}, false)
	}

	// Ambient bubbles / stars based on stage
	g.ieDrawAmbient(screen)
}

func (g *IdleEvolution) ieDrawAmbient(screen *ebiten.Image) {
	// Simple animated background elements
	t := float64(g.tickCount)

	if g.stage <= ieStageMulti {
		// Bubbles in primordial soup
		for i := 0; i < 15; i++ {
			fi := float64(i)
			bx := float32(math.Mod(fi*37.3+t*0.3, float64(ieScreenW)))
			by := float32(math.Mod(fi*53.7-t*0.5+500, float64(ieScreenH)))
			r := float32(2 + math.Sin(fi*2.1)*1.5)
			vector.StrokeCircle(screen, bx, by, r, 1,
				color.RGBA{80, 120, 150, 40}, false)
		}
	} else if g.stage <= ieStageFish {
		// Water ripples
		for i := 0; i < 10; i++ {
			fi := float64(i)
			bx := float32(math.Mod(fi*47.3+t*0.2, float64(ieScreenW)))
			by := float32(100 + math.Sin(t*0.02+fi)*50)
			vector.StrokeLine(screen, bx, by, bx+20, by+float32(math.Sin(t*0.05+fi)*5),
				1, color.RGBA{60, 100, 140, 30}, false)
		}
	} else if g.stage >= ieStageHuman {
		// Stars
		for i := 0; i < 30; i++ {
			fi := float64(i)
			sx := float32(math.Mod(fi*67.3+13, float64(ieScreenW)))
			sy := float32(math.Mod(fi*43.7+7, float64(ieScreenH)))
			bright := uint8(100 + int(math.Sin(t*0.03+fi*1.7)*50))
			vector.FillRect(screen, sx, sy, 2, 2,
				color.RGBA{bright, bright, bright + 30, 200}, false)
		}
	}
}

func (g *IdleEvolution) ieDrawDNAHelix(screen *ebiten.Image) {
	// Rotating double helix using sine waves along the left edge
	helixX := float32(30)
	t := float64(g.tickCount) * 0.04
	for i := 0; i < 40; i++ {
		fi := float64(i)
		y := float32(85 + i*5)
		if y > 285 {
			break
		}
		phase := t + fi*0.35
		// Strand A
		ax := helixX + float32(math.Sin(phase)*12)
		// Strand B (opposite phase)
		bx := helixX + float32(math.Sin(phase+math.Pi)*12)

		depthA := float32(math.Cos(phase)*0.5 + 0.5)
		depthB := float32(math.Cos(phase+math.Pi)*0.5 + 0.5)

		alphaA := uint8(80 + depthA*120)
		alphaB := uint8(80 + depthB*120)

		// Draw connecting rung (every 4th position)
		if i%4 == 0 {
			vector.StrokeLine(screen, ax, y, bx, y, 0.8,
				color.RGBA{60, 180, 100, 50}, false)
		}

		// Draw strand dots
		rA := float32(1.5 + depthA*1.5)
		rB := float32(1.5 + depthB*1.5)
		vector.FillCircle(screen, ax, y, rA, color.RGBA{80, 220, 120, alphaA}, false)
		vector.FillCircle(screen, bx, y, rB, color.RGBA{120, 80, 220, alphaB}, false)
	}
}

func (g *IdleEvolution) ieDrawStageParticles(screen *ebiten.Image) {
	// Ambient particles around the creature matching the current stage
	t := float64(g.tickCount)
	var pr, pg, pb uint8
	switch {
	case g.stage <= ieStageMulti:
		pr, pg, pb = 60, 180, 80 // green organic
	case g.stage <= ieStageFish:
		pr, pg, pb = 60, 130, 200 // blue aquatic
	case g.stage <= ieStageAmphibian:
		pr, pg, pb = 50, 160, 50 // green land
	case g.stage <= ieStageReptile:
		pr, pg, pb = 80, 120, 40 // earthy
	case g.stage <= ieStageMammal:
		pr, pg, pb = 140, 100, 60 // warm brown
	case g.stage <= ieStagePrimate:
		pr, pg, pb = 120, 80, 50 // darker brown
	default:
		pr, pg, pb = 100, 150, 255 // tech blue
	}

	for i := 0; i < 12; i++ {
		fi := float64(i)
		px := float32(110 + math.Sin(t*0.02+fi*1.1)*80)
		py := float32(120 + math.Cos(t*0.018+fi*0.9)*70)
		alpha := uint8(30 + math.Sin(t*0.03+fi*2.0)*20)
		r := float32(1 + math.Sin(t*0.04+fi)*0.8)
		vector.FillCircle(screen, px, py, r, color.RGBA{pr, pg, pb, alpha}, false)
	}
}

func (g *IdleEvolution) ieDrawProgressBar(screen *ebiten.Image) {
	barX := float32(10)
	barY := float32(56)
	barW := float32(290)
	barH := float32(14)

	// Background
	vector.FillRect(screen, barX, barY, barW, barH, color.RGBA{20, 20, 30, 200}, false)
	vector.StrokeRect(screen, barX, barY, barW, barH, 1, color.RGBA{100, 100, 120, 255}, false)

	// Fill
	progress := g.ieStageProgress()
	fillW := barW * float32(progress)
	vector.FillRect(screen, barX+1, barY+1, fillW-2, barH-2, color.RGBA{50, 200, 100, 220}, false)

	// Label
	if g.stage < ieStageCount-1 {
		nextName := ieStageNames[g.stage+1]
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Next: %s (%s DNA)", nextName, ieFormatNum(g.ieNextThreshold())),
			int(barX), int(barY+barH+2))
	} else {
		ebitenutil.DebugPrintAt(screen, "FULLY EVOLVED!", int(barX), int(barY+barH+2))
	}
}

func (g *IdleEvolution) ieDrawCreature(screen *ebiten.Image) {
	cx := float32(175)
	cy := float32(190)
	bob := float32(math.Sin(g.bobPhase) * 5)

	// Creature area border
	vector.StrokeRect(screen, 100, 85, 200, 200, 1, color.RGBA{80, 80, 100, 100}, false)

	switch g.stage {
	case ieStageSoup:
		// Swirling dots
		for i := 0; i < 8; i++ {
			fi := float64(i)
			angle := g.bobPhase + fi*0.8
			dx := float32(math.Cos(angle) * 30)
			dy := float32(math.Sin(angle) * 30)
			vector.FillCircle(screen, cx+dx, cy+bob+dy, 3, color.RGBA{100, 180, 100, 180}, false)
		}
		// central blob
		vector.FillCircle(screen, cx, cy+bob, 12, color.RGBA{80, 150, 80, 200}, false)

	case ieStageSingleCell:
		// Cell with nucleus
		vector.FillCircle(screen, cx, cy+bob, 25, color.RGBA{60, 180, 80, 200}, false)
		vector.StrokeCircle(screen, cx, cy+bob, 25, 2, color.RGBA{40, 140, 60, 255}, false)
		vector.FillCircle(screen, cx, cy+bob, 8, color.RGBA{40, 100, 40, 255}, false)
		// membrane dots
		for i := 0; i < 6; i++ {
			a := g.bobPhase*0.5 + float64(i)*math.Pi/3
			dx := float32(math.Cos(a) * 25)
			dy := float32(math.Sin(a) * 25)
			vector.FillCircle(screen, cx+dx, cy+bob+dy, 2, color.RGBA{100, 220, 100, 200}, false)
		}

	case ieStageMulti:
		// Cluster of cells
		positions := [][2]float32{{0, 0}, {-18, -12}, {18, -12}, {-18, 12}, {18, 12}, {0, -22}, {0, 22}}
		for _, p := range positions {
			vector.FillCircle(screen, cx+p[0], cy+bob+p[1], 11, color.RGBA{50, 170, 90, 200}, false)
			vector.StrokeCircle(screen, cx+p[0], cy+bob+p[1], 11, 1, color.RGBA{30, 120, 60, 255}, false)
		}

	case ieStageFish:
		// Fish body (ellipse-ish with vector calls)
		// Body
		vector.FillCircle(screen, cx, cy+bob, 22, color.RGBA{60, 130, 200, 230}, false)
		vector.FillCircle(screen, cx+15, cy+bob, 18, color.RGBA{60, 130, 200, 230}, false)
		vector.FillCircle(screen, cx-15, cy+bob, 18, color.RGBA{60, 130, 200, 230}, false)
		// Tail
		vector.FillRect(screen, cx-38, cy+bob-12, 15, 24, color.RGBA{50, 110, 180, 200}, false)
		// Eye
		vector.FillCircle(screen, cx+22, cy+bob-5, 4, color.RGBA{255, 255, 255, 255}, false)
		vector.FillCircle(screen, cx+23, cy+bob-5, 2, color.RGBA{0, 0, 0, 255}, false)
		// Fin
		vector.FillRect(screen, cx-5, cy+bob-25, 15, 8, color.RGBA{40, 100, 160, 180}, false)

	case ieStageAmphibian:
		// Frog-like creature
		// Body
		vector.FillCircle(screen, cx, cy+bob, 22, color.RGBA{50, 160, 50, 230}, false)
		// Head
		vector.FillCircle(screen, cx+25, cy+bob-5, 14, color.RGBA{60, 170, 60, 230}, false)
		// Eyes
		vector.FillCircle(screen, cx+30, cy+bob-16, 6, color.RGBA{80, 200, 80, 255}, false)
		vector.FillCircle(screen, cx+30, cy+bob-16, 3, color.RGBA{0, 0, 0, 255}, false)
		vector.FillCircle(screen, cx+22, cy+bob-16, 6, color.RGBA{80, 200, 80, 255}, false)
		vector.FillCircle(screen, cx+22, cy+bob-16, 3, color.RGBA{0, 0, 0, 255}, false)
		// Legs
		vector.FillRect(screen, cx-20, cy+bob+15, 8, 18, color.RGBA{40, 140, 40, 200}, false)
		vector.FillRect(screen, cx+10, cy+bob+15, 8, 18, color.RGBA{40, 140, 40, 200}, false)

	case ieStageReptile:
		// Lizard
		// Body
		vector.FillRect(screen, cx-30, cy+bob-8, 60, 16, color.RGBA{80, 120, 40, 230}, false)
		// Head
		vector.FillCircle(screen, cx+35, cy+bob, 12, color.RGBA{90, 130, 50, 230}, false)
		// Eye
		vector.FillCircle(screen, cx+40, cy+bob-4, 3, color.RGBA{200, 200, 0, 255}, false)
		vector.FillCircle(screen, cx+41, cy+bob-4, 1, color.RGBA{0, 0, 0, 255}, false)
		// Tail
		vector.FillRect(screen, cx-55, cy+bob-4, 28, 8, color.RGBA{70, 110, 35, 200}, false)
		// Legs
		vector.FillRect(screen, cx-15, cy+bob+8, 6, 14, color.RGBA{70, 110, 35, 200}, false)
		vector.FillRect(screen, cx+15, cy+bob+8, 6, 14, color.RGBA{70, 110, 35, 200}, false)
		// Spines
		for i := 0; i < 5; i++ {
			sx := cx - 20 + float32(i)*10
			vector.FillRect(screen, sx, cy+bob-12, 4, 5, color.RGBA{100, 140, 50, 180}, false)
		}

	case ieStageMammal:
		// Dog/wolf-like
		// Body
		vector.FillCircle(screen, cx, cy+bob, 22, color.RGBA{140, 100, 60, 230}, false)
		vector.FillCircle(screen, cx-15, cy+bob, 18, color.RGBA{140, 100, 60, 230}, false)
		// Head
		vector.FillCircle(screen, cx+28, cy+bob-10, 14, color.RGBA{150, 110, 70, 230}, false)
		// Ears
		vector.FillRect(screen, cx+20, cy+bob-28, 6, 10, color.RGBA{130, 90, 50, 220}, false)
		vector.FillRect(screen, cx+32, cy+bob-28, 6, 10, color.RGBA{130, 90, 50, 220}, false)
		// Eye
		vector.FillCircle(screen, cx+32, cy+bob-12, 3, color.RGBA{30, 30, 30, 255}, false)
		// Nose
		vector.FillCircle(screen, cx+40, cy+bob-6, 2, color.RGBA{20, 20, 20, 255}, false)
		// Legs
		vector.FillRect(screen, cx-20, cy+bob+14, 6, 16, color.RGBA{130, 90, 50, 200}, false)
		vector.FillRect(screen, cx-8, cy+bob+14, 6, 16, color.RGBA{130, 90, 50, 200}, false)
		vector.FillRect(screen, cx+8, cy+bob+14, 6, 16, color.RGBA{130, 90, 50, 200}, false)
		vector.FillRect(screen, cx+20, cy+bob+14, 6, 16, color.RGBA{130, 90, 50, 200}, false)
		// Tail
		vector.FillRect(screen, cx-35, cy+bob-15, 8, 5, color.RGBA{140, 100, 60, 200}, false)

	case ieStagePrimate:
		// Ape-like (upright)
		// Body
		vector.FillCircle(screen, cx, cy+bob+5, 20, color.RGBA{120, 80, 50, 230}, false)
		// Head
		vector.FillCircle(screen, cx, cy+bob-22, 15, color.RGBA{130, 90, 55, 230}, false)
		// Face
		vector.FillCircle(screen, cx, cy+bob-18, 8, color.RGBA{160, 120, 80, 200}, false)
		// Eyes
		vector.FillCircle(screen, cx-5, cy+bob-24, 3, color.RGBA{255, 255, 255, 255}, false)
		vector.FillCircle(screen, cx-5, cy+bob-24, 1, color.RGBA{0, 0, 0, 255}, false)
		vector.FillCircle(screen, cx+5, cy+bob-24, 3, color.RGBA{255, 255, 255, 255}, false)
		vector.FillCircle(screen, cx+5, cy+bob-24, 1, color.RGBA{0, 0, 0, 255}, false)
		// Arms
		vector.FillRect(screen, cx-28, cy+bob-5, 10, 6, color.RGBA{110, 70, 40, 200}, false)
		vector.FillRect(screen, cx+18, cy+bob-5, 10, 6, color.RGBA{110, 70, 40, 200}, false)
		// Legs
		vector.FillRect(screen, cx-10, cy+bob+20, 8, 16, color.RGBA{110, 70, 40, 200}, false)
		vector.FillRect(screen, cx+4, cy+bob+20, 8, 16, color.RGBA{110, 70, 40, 200}, false)

	case ieStageHuman:
		// Stick figure style human
		headY := cy + bob - 30
		// Head
		vector.FillCircle(screen, cx, headY, 12, color.RGBA{220, 180, 140, 255}, false)
		// Eyes
		vector.FillCircle(screen, cx-4, headY-2, 2, color.RGBA{50, 50, 50, 255}, false)
		vector.FillCircle(screen, cx+4, headY-2, 2, color.RGBA{50, 50, 50, 255}, false)
		// Mouth
		vector.StrokeLine(screen, cx-3, headY+4, cx+3, headY+4, 1, color.RGBA{50, 50, 50, 255}, false)
		// Body
		vector.FillRect(screen, cx-8, cy+bob-18, 16, 28, color.RGBA{50, 80, 150, 230}, false)
		// Arms
		armWave := float32(math.Sin(g.bobPhase*2) * 8)
		vector.StrokeLine(screen, cx-8, cy+bob-14, cx-25, cy+bob-5+armWave, 3, color.RGBA{220, 180, 140, 255}, false)
		vector.StrokeLine(screen, cx+8, cy+bob-14, cx+25, cy+bob-5-armWave, 3, color.RGBA{220, 180, 140, 255}, false)
		// Legs
		vector.StrokeLine(screen, cx-4, cy+bob+10, cx-10, cy+bob+30, 3, color.RGBA{80, 80, 80, 255}, false)
		vector.StrokeLine(screen, cx+4, cy+bob+10, cx+10, cy+bob+30, 3, color.RGBA{80, 80, 80, 255}, false)
		// Tool in hand
		vector.StrokeLine(screen, cx+25, cy+bob-5-armWave, cx+30, cy+bob-20-armWave, 2, color.RGBA{180, 140, 60, 255}, false)

	case ieStageSpace:
		// Rocket / spaceship
		// Fuselage
		vector.FillRect(screen, cx-8, cy+bob-35, 16, 50, color.RGBA{200, 200, 210, 230}, false)
		// Nose cone
		vector.FillCircle(screen, cx, cy+bob-35, 8, color.RGBA{220, 50, 50, 255}, false)
		// Window
		vector.FillCircle(screen, cx, cy+bob-18, 5, color.RGBA{100, 180, 255, 255}, false)
		// Fins
		vector.FillRect(screen, cx-20, cy+bob+5, 14, 12, color.RGBA{180, 180, 190, 200}, false)
		vector.FillRect(screen, cx+6, cy+bob+5, 14, 12, color.RGBA{180, 180, 190, 200}, false)
		// Flame
		flicker := float32(rand.Float64() * 8)
		vector.FillRect(screen, cx-5, cy+bob+15, 10, 15+flicker, color.RGBA{255, 150, 30, 220}, false)
		vector.FillRect(screen, cx-3, cy+bob+18, 6, 12+flicker, color.RGBA{255, 255, 50, 200}, false)
	}
}

func (g *IdleEvolution) ieDrawUpgradePanel(screen *ebiten.Image) {
	panelX := float32(305)
	panelY := float32(80)
	panelW := float32(190)
	panelH := float32(360)

	// Panel background
	vector.FillRect(screen, panelX, panelY, panelW, panelH, color.RGBA{15, 15, 25, 220}, false)
	vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 1, color.RGBA{80, 80, 100, 200}, false)

	yOff := int(panelY) - g.scrollY

	// --- Generators ---
	ebitenutil.DebugPrintAt(screen, "== Generators ==", int(panelX)+4, yOff)
	yOff += 18

	for i := 0; i < ieGenCount; i++ {
		if ieGenerators[i].stage > g.stage {
			continue
		}
		if yOff >= int(panelY)-10 && yOff < int(panelY+panelH)-10 {
			cost := ieGenCost(i, g.genOwned[i])
			canBuy := g.dna >= cost

			// Button background
			btnColor := color.RGBA{30, 30, 50, 220}
			if canBuy {
				btnColor = color.RGBA{30, 60, 40, 220}
			}
			vector.FillRect(screen, panelX+2, float32(yOff), panelW-4, 32, btnColor, false)
			vector.StrokeRect(screen, panelX+2, float32(yOff), panelW-4, 32, 1,
				color.RGBA{70, 70, 90, 200}, false)

			ebitenutil.DebugPrintAt(screen,
				fmt.Sprintf("%s x%d", ieGenerators[i].name, g.genOwned[i]),
				int(panelX)+6, yOff+2)
			ebitenutil.DebugPrintAt(screen,
				fmt.Sprintf("Cost: %s  +%s/s", ieFormatNum(cost), ieFormatNum(ieGenerators[i].baseDPS*g.ieTotalMultiplier())),
				int(panelX)+6, yOff+16)
		}
		yOff += 36
	}

	yOff += 8
	// --- Mutations ---
	ebitenutil.DebugPrintAt(screen, "== Mutations ==", int(panelX)+4, yOff)
	yOff += 18

	for i := 0; i < ieMutationCount; i++ {
		if ieMutations[i].stage > g.stage {
			continue
		}
		if yOff >= int(panelY)-10 && yOff < int(panelY+panelH)-10 {
			cost := ieMutCost(i, g.mutBought[i])
			canBuy := g.dna >= cost

			btnColor := color.RGBA{30, 30, 50, 220}
			if canBuy {
				btnColor = color.RGBA{50, 40, 30, 220}
			}
			vector.FillRect(screen, panelX+2, float32(yOff), panelW-4, 32, btnColor, false)
			vector.StrokeRect(screen, panelX+2, float32(yOff), panelW-4, 32, 1,
				color.RGBA{70, 70, 90, 200}, false)

			ebitenutil.DebugPrintAt(screen,
				fmt.Sprintf("%s (x%.1f) Lv%d", ieMutations[i].name, ieMutations[i].multi, g.mutBought[i]),
				int(panelX)+6, yOff+2)
			ebitenutil.DebugPrintAt(screen,
				fmt.Sprintf("Cost: %s", ieFormatNum(cost)),
				int(panelX)+6, yOff+16)
		}
		yOff += 36
	}
}

func (g *IdleEvolution) ieDrawStats(screen *ebiten.Image) {
	y := ieScreenH - 22
	mins := g.timePlayed / 60
	secs := g.timePlayed % 60

	ebitenutil.DebugPrintAt(screen,
		fmt.Sprintf("Total:%s Time:%d:%02d Muts:%d ESC:Back M:1M B:1B G:10x",
			ieFormatNum(g.totalDNA), mins, secs, g.ieMutsPurchased()),
		5, y)
}

func (g *IdleEvolution) ieDrawWinScreen(screen *ebiten.Image) {
	// Starfield background already drawn

	// Big congratulations
	ebitenutil.DebugPrintAt(screen, "=================================", 100, 100)
	ebitenutil.DebugPrintAt(screen, "   YOU REACHED THE SPACE AGE!", 110, 120)
	ebitenutil.DebugPrintAt(screen, "   EVOLUTION COMPLETE!", 130, 140)
	ebitenutil.DebugPrintAt(screen, "=================================", 100, 160)

	// Draw rocket
	cx := float32(250)
	cy := float32(250)
	bob := float32(math.Sin(g.bobPhase) * 8)

	// Fuselage
	vector.FillRect(screen, cx-12, cy+bob-45, 24, 65, color.RGBA{200, 200, 220, 255}, false)
	// Nose
	vector.FillCircle(screen, cx, cy+bob-45, 12, color.RGBA{230, 50, 50, 255}, false)
	// Window
	vector.FillCircle(screen, cx, cy+bob-25, 7, color.RGBA{80, 170, 255, 255}, false)
	// Fins
	vector.FillRect(screen, cx-28, cy+bob+8, 18, 16, color.RGBA{180, 180, 200, 230}, false)
	vector.FillRect(screen, cx+10, cy+bob+8, 18, 16, color.RGBA{180, 180, 200, 230}, false)
	// Flame
	flicker := float32(rand.Float64() * 12)
	vector.FillRect(screen, cx-8, cy+bob+20, 16, 22+flicker, color.RGBA{255, 140, 20, 240}, false)
	vector.FillRect(screen, cx-4, cy+bob+25, 8, 18+flicker, color.RGBA{255, 255, 40, 220}, false)

	// Stars around rocket
	for i := 0; i < 40; i++ {
		fi := float64(i)
		sx := float32(math.Mod(fi*71.3+17, float64(ieScreenW)))
		sy := float32(math.Mod(fi*47.7+11, float64(ieScreenH)))
		twinkle := uint8(150 + int(math.Sin(g.bobPhase+fi*1.3)*80))
		vector.FillRect(screen, sx, sy, 2, 2, color.RGBA{twinkle, twinkle, twinkle + 20, 255}, false)
	}

	mins := g.timePlayed / 60
	secs := g.timePlayed % 60
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Total DNA earned: %s", ieFormatNum(g.totalDNA)), 150, 340)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Time: %d:%02d  Mutations: %d", mins, secs, g.ieMutsPurchased()), 150, 358)
	ebitenutil.DebugPrintAt(screen, "Press R to restart  |  ESC to go back", 120, 400)

	g.bobPhase += 0.02 // extra animation speed for win screen
}
