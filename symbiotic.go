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
	sgGridW    = 12
	sgGridH    = 12
	sgCellSize = 34
	sgMargin   = 16
	sgHudH     = 70
	sgScreenW  = sgMargin*2 + sgGridW*sgCellSize
	sgScreenH  = sgHudH + sgMargin + sgGridH*sgCellSize + sgMargin
)

// Cell types
const (
	sgEmpty    = 0
	sgLight    = 1
	sgDark     = 2
	sgNeutral  = 3
	sgWall     = 4
	sgSymbiote = 5 // symbiosis tile: decouple movement for N turns
	sgTrailA   = 6 // safe trail left by organism A
	sgTrailB   = 7 // safe trail left by organism B
)

type sgPhase int

const (
	sgPlaying sgPhase = iota
	sgWon
	sgDead
	sgLevelIntro
)

type SymbioticGrid struct {
	goBack bool

	grid     [sgGridH][sgGridW]int
	ax, ay   int // organism A position
	bx, by   int // organism B position
	aSize    int // how much A has grown
	bSize    int // how much B has grown
	aHP      int // A health (starts at 5, toxic cells reduce it)
	bHP      int // B health
	targetA  int // growth target for A
	targetB  int // growth target for B

	phase    sgPhase
	level    int
	moves    int
	decouple int // turns of decoupled movement remaining
	moveA    bool // when decoupled, which organism to move (toggled with Tab)

	introTimer int
	msg        string
	msgTimer   int

	// Visual: tick counter for animations
	sgTicks int

	// Audio
	bgm          *audio.Player
	sfxAbsorb    []byte
	sfxToxic     []byte
	sfxSymbiosis []byte
	sfxWin       []byte
	sfxDeath     []byte
	audioInit    bool
}

func newSymbioticGrid() *SymbioticGrid {
	g := &SymbioticGrid{}
	g.loadLevel(1)
	return g
}

// ---------------------------------------------------------------------------
// Audio initialization
// ---------------------------------------------------------------------------

func (g *SymbioticGrid) sgInitAudio() {
	if g.audioInit {
		return
	}
	g.audioInit = true

	// --- BGM: Mysterious, ethereal puzzle music (~85 BPM) ---
	// Two intertwined melodies representing the two organisms.
	// Minor key (E minor) with modal interchange to C major / A dorian.

	// Melody A (sine wave) -- organism A's voice, contemplative
	melodyA := [][2]int{
		// A section -- E minor, floating
		{76, 4}, {0, 2}, {79, 2}, {83, 4}, {0, 2}, {81, 2},
		{79, 4}, {0, 2}, {76, 2}, {74, 4}, {0, 4},
		{76, 3}, {0, 1}, {79, 2}, {81, 2}, {83, 4}, {0, 2}, {79, 2},
		{76, 4}, {0, 2}, {74, 2}, {72, 4}, {0, 4},
		{71, 4}, {0, 2}, {74, 2}, {76, 4}, {0, 2}, {79, 2},
		{83, 6}, {0, 2}, {81, 6}, {0, 2},
		{79, 4}, {76, 4}, {74, 4}, {0, 4},
		// B section -- A dorian shift, warmer
		{69, 4}, {0, 2}, {72, 2}, {74, 4}, {0, 2}, {76, 2},
		{79, 4}, {0, 2}, {76, 2}, {74, 4}, {0, 4},
		{72, 3}, {0, 1}, {74, 2}, {76, 2}, {79, 4}, {0, 2}, {76, 2},
		{74, 4}, {0, 2}, {72, 2}, {69, 4}, {0, 4},
		// C section -- tension build, chromatic touches
		{76, 4}, {0, 2}, {75, 2}, {74, 4}, {0, 2}, {72, 2},
		{71, 4}, {0, 2}, {74, 2}, {76, 6}, {0, 2},
		{79, 4}, {0, 2}, {83, 2}, {81, 4}, {0, 2}, {79, 2},
		{76, 4}, {74, 4}, {72, 4}, {0, 4},
	}
	melodyAPCM := generateTrack(melodyA, sineWave, 85, 0.20)

	// Melody B (triangle wave) -- organism B's voice, darker countermelody
	melodyB := [][2]int{
		// A section -- lower register, mysterious
		{0, 8}, {64, 4}, {0, 2}, {67, 2},
		{71, 4}, {0, 2}, {69, 2}, {67, 4}, {0, 4},
		{0, 4}, {64, 3}, {0, 1}, {67, 2}, {69, 2},
		{71, 4}, {0, 2}, {67, 2}, {64, 4}, {0, 4},
		{0, 4}, {62, 4}, {0, 2}, {64, 2},
		{67, 6}, {0, 2}, {69, 6}, {0, 2},
		{67, 4}, {64, 4}, {62, 4}, {0, 4},
		// B section -- moves with A
		{0, 4}, {57, 4}, {0, 2}, {60, 2},
		{64, 4}, {0, 2}, {62, 2}, {60, 4}, {0, 4},
		{0, 4}, {60, 3}, {0, 1}, {62, 2}, {64, 2},
		{67, 4}, {0, 2}, {64, 2}, {60, 4}, {0, 4},
		// C section -- chromatic tension
		{0, 4}, {64, 4}, {0, 2}, {63, 2},
		{62, 4}, {0, 2}, {64, 2}, {67, 6}, {0, 2},
		{0, 4}, {67, 4}, {0, 2}, {71, 2},
		{69, 4}, {67, 4}, {64, 4}, {0, 4},
	}
	melodyBPCM := generateTrack(melodyB, triangleWave, 85, 0.15)

	// Bass -- deep sine, Em pedal tones
	bass := [][2]int{
		{40, 8}, {0, 4}, {43, 4}, {40, 8}, {0, 4}, {47, 4},
		{40, 6}, {0, 2}, {43, 4}, {45, 4}, {47, 4}, {0, 4},
		{40, 8}, {0, 4}, {43, 4}, {40, 8}, {0, 8},
		// B section bass
		{45, 8}, {0, 4}, {43, 4}, {40, 8}, {0, 4}, {38, 4},
		{36, 6}, {0, 2}, {38, 4}, {40, 4}, {43, 4}, {0, 4},
		// C section bass
		{40, 8}, {0, 4}, {39, 4}, {38, 8}, {0, 4}, {40, 4},
		{43, 8}, {0, 4}, {40, 4}, {40, 8}, {0, 8},
	}
	bassPCM := generateTrack(bass, sineWave, 85, 0.30)

	// Very sparse, meditative percussion
	drums := generateDrumPattern(
		"K.............h.........K...........h..........."+
			"K.......h...........K...............h...........", 85, 0.12)

	// Mix all layers
	bgmPCM := mixBuffers(melodyAPCM, melodyBPCM)
	bgmPCM = mixBuffers(bgmPCM, bassPCM)
	for len(drums) < len(bgmPCM) {
		drums = append(drums, drums...)
	}
	drums = drums[:len(bgmPCM)]
	bgmPCM = mixBuffers(bgmPCM, drums)
	g.bgm = loopPlayer(bgmPCM, 0.15)

	// --- SFX: Absorb (soft rising tone on feeding) ---
	{
		dur := 0.15
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := float64(i) / float64(samples)
			freq := noteFreq(72) + (noteFreq(84)-noteFreq(72))*tFrac
			env := envelope(tFrac, 0.05, 0.5, 0.45)
			val := sineWave(t, freq)*0.4*env + triangleWave(t, freq*1.5)*0.2*env
			writeSample(buf, i*4, val)
		}
		g.sfxAbsorb = buf
	}

	// --- SFX: Toxic (harsh buzz on taking damage) ---
	{
		dur := 0.2
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := float64(i) / float64(samples)
			env := envelope(tFrac, 0.01, 0.3, 0.69)
			val := squareWave(t, 90)*0.3*env + noise()*0.4*math.Exp(-t*10)
			val += sineWave(t, 55)*0.2*env
			writeSample(buf, i*4, val)
		}
		g.sfxToxic = buf
	}

	// --- SFX: Symbiosis (magical shimmer) ---
	{
		dur := 0.35
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := float64(i) / float64(samples)
			env := envelope(tFrac, 0.05, 0.4, 0.55)
			// Layered harmonics creating shimmer
			val := sineWave(t, noteFreq(84))*0.25*env +
				sineWave(t, noteFreq(88))*0.2*env +
				sineWave(t, noteFreq(91))*0.15*env +
				triangleWave(t, noteFreq(79))*0.1*env
			// Add gentle sparkle noise
			val += noise() * math.Exp(-t*20) * 0.05
			writeSample(buf, i*4, val)
		}
		g.sfxSymbiosis = buf
	}

	// --- SFX: Win (triumphant ascending fanfare) ---
	{
		notes := [][2]int{
			{64, 2}, {67, 2}, {71, 2}, {76, 2}, {79, 2}, {83, 4},
		}
		g.sfxWin = generateTrack(notes, sineWave, 300, 0.30)
	}

	// --- SFX: Death (sad descending tone) ---
	{
		dur := 0.5
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := float64(i) / float64(samples)
			// Descending from E5 down to E3
			freq := noteFreq(76) * math.Pow(0.25, tFrac)
			env := envelope(tFrac, 0.02, 0.4, 0.58)
			val := sineWave(t, freq)*0.3*env + triangleWave(t, freq*0.5)*0.2*env
			writeSample(buf, i*4, val)
		}
		g.sfxDeath = buf
	}
}

func (g *SymbioticGrid) WantsBack() bool            { return g.goBack }
func (g *SymbioticGrid) Layout(w, h int) (int, int) { return sgScreenW, sgScreenH }

// --- Level generation ---

func (g *SymbioticGrid) loadLevel(lvl int) {
	g.level = lvl
	g.phase = sgLevelIntro
	g.introTimer = 90
	g.moves = 0
	g.decouple = 0
	g.moveA = true
	g.aSize = 0
	g.bSize = 0
	g.aHP = 5 + lvl/3
	g.bHP = 5 + lvl/3
	g.msg = ""
	g.msgTimer = 0

	// Difficulty scaling
	g.targetA = 3 + lvl
	g.targetB = 3 + lvl
	if g.targetA > 12 {
		g.targetA = 12
	}
	if g.targetB > 12 {
		g.targetB = 12
	}

	// Clear grid
	for r := 0; r < sgGridH; r++ {
		for c := 0; c < sgGridW; c++ {
			g.grid[r][c] = sgEmpty
		}
	}

	// Place walls (more walls at higher levels)
	wallCount := 8 + lvl*3
	if wallCount > 35 {
		wallCount = 35
	}
	for i := 0; i < wallCount; i++ {
		r, c := rand.Intn(sgGridH), rand.Intn(sgGridW)
		g.grid[r][c] = sgWall
	}

	// Place light cells
	lightCount := g.targetA + 3 + lvl
	for i := 0; i < lightCount; i++ {
		g.placeRandom(sgLight)
	}

	// Place dark cells
	darkCount := g.targetB + 3 + lvl
	for i := 0; i < darkCount; i++ {
		g.placeRandom(sgDark)
	}

	// Place neutral cells (safe for both)
	neutralCount := 4 + lvl/2
	for i := 0; i < neutralCount; i++ {
		g.placeRandom(sgNeutral)
	}

	// Place symbiosis tiles (decouple movement)
	symbCount := 1 + lvl/3
	if symbCount > 4 {
		symbCount = 4
	}
	for i := 0; i < symbCount; i++ {
		g.placeRandom(sgSymbiote)
	}

	// Place organism A (top-left area)
	g.ax, g.ay = 1, 1
	g.grid[g.ay][g.ax] = sgEmpty

	// Place organism B (bottom-right area) — B moves mirrored, so it starts opposite
	g.bx, g.by = sgGridW-2, sgGridH-2
	g.grid[g.by][g.bx] = sgEmpty

	// Clear immediate surroundings
	for _, d := range [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}} {
		nr, nc := g.ay+d[1], g.ax+d[0]
		if nr >= 0 && nr < sgGridH && nc >= 0 && nc < sgGridW {
			if g.grid[nr][nc] == sgWall {
				g.grid[nr][nc] = sgEmpty
			}
		}
		nr, nc = g.by+d[1], g.bx+d[0]
		if nr >= 0 && nr < sgGridH && nc >= 0 && nc < sgGridW {
			if g.grid[nr][nc] == sgWall {
				g.grid[nr][nc] = sgEmpty
			}
		}
	}
}

func (g *SymbioticGrid) placeRandom(cellType int) {
	for attempts := 0; attempts < 100; attempts++ {
		r, c := rand.Intn(sgGridH), rand.Intn(sgGridW)
		if g.grid[r][c] == sgEmpty && !(r == 1 && c == 1) && !(r == sgGridH-2 && c == sgGridW-2) {
			g.grid[r][c] = cellType
			return
		}
	}
}

// --- Update ---

func (g *SymbioticGrid) Update() error {
	// Initialize and start audio
	g.sgInitAudio()
	if g.bgm != nil && !g.bgm.IsPlaying() {
		g.bgm.Play()
	}

	g.sgTicks++

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.bgm != nil && g.bgm.IsPlaying() {
			g.bgm.Pause()
		}
		g.goBack = true
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.loadLevel(g.level)
		return nil
	}
	if g.msgTimer > 0 {
		g.msgTimer--
	}

	switch g.phase {
	case sgLevelIntro:
		g.introTimer--
		if g.introTimer <= 0 {
			g.phase = sgPlaying
		}
		// Skip intro on any key
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.phase = sgPlaying
		}
	case sgPlaying:
		g.updatePlaying()
	case sgWon:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.loadLevel(g.level + 1)
		}
	case sgDead:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.loadLevel(g.level)
		}
	}
	return nil
}

func (g *SymbioticGrid) updatePlaying() {
	// Toggle which organism to move when decoupled
	if g.decouple > 0 && inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.moveA = !g.moveA
	}

	dx, dy := 0, 0
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		dy = -1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		dy = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		dx = -1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		dx = 1
	}

	if dx == 0 && dy == 0 {
		return
	}

	if g.decouple > 0 {
		// Move only one organism
		if g.moveA {
			g.tryMoveA(dx, dy)
		} else {
			g.tryMoveB(dx, dy)
		}
		g.decouple--
		if g.decouple == 0 {
			g.setMsg("Symbiosis ended — linked again!")
		}
	} else {
		// Normal: A moves in input direction, B moves mirrored (inverted X, same Y)
		g.tryMoveA(dx, dy)
		g.tryMoveB(-dx, dy) // mirror X
	}

	g.moves++
	g.checkWin()
	g.checkDeath()
}

func (g *SymbioticGrid) tryMoveA(dx, dy int) {
	nx, ny := g.ax+dx, g.ay+dy
	if nx < 0 || nx >= sgGridW || ny < 0 || ny >= sgGridH {
		return
	}
	cell := g.grid[ny][nx]
	if cell == sgWall {
		return
	}

	// Leave trail at old position
	g.grid[g.ay][g.ax] = sgTrailA

	g.ax, g.ay = nx, ny
	g.processLandingA(cell)
}

func (g *SymbioticGrid) tryMoveB(dx, dy int) {
	nx, ny := g.bx+dx, g.by+dy
	if nx < 0 || nx >= sgGridW || ny < 0 || ny >= sgGridH {
		return
	}
	cell := g.grid[ny][nx]
	if cell == sgWall {
		return
	}

	g.grid[g.by][g.bx] = sgTrailB

	g.bx, g.by = nx, ny
	g.processLandingB(cell)
}

func (g *SymbioticGrid) processLandingA(cell int) {
	switch cell {
	case sgLight:
		g.aSize++
		g.setMsg(fmt.Sprintf("A absorbs light! (%d/%d)", g.aSize, g.targetA))
		if g.sfxAbsorb != nil {
			playSFX(g.sfxAbsorb, 0.35)
		}
	case sgDark:
		g.aHP--
		g.setMsg(fmt.Sprintf("Dark is toxic to A! HP:%d", g.aHP))
		if g.sfxToxic != nil {
			playSFX(g.sfxToxic, 0.40)
		}
	case sgSymbiote:
		g.decouple += 4
		g.setMsg("Symbiosis! Tab to switch, 4 free moves")
		if g.sfxSymbiosis != nil {
			playSFX(g.sfxSymbiosis, 0.40)
		}
	case sgTrailB:
		// B's trail is safe for A
	}
	g.grid[g.ay][g.ax] = sgEmpty // consumed
}

func (g *SymbioticGrid) processLandingB(cell int) {
	switch cell {
	case sgDark:
		g.bSize++
		g.setMsg(fmt.Sprintf("B absorbs dark! (%d/%d)", g.bSize, g.targetB))
		if g.sfxAbsorb != nil {
			playSFX(g.sfxAbsorb, 0.35)
		}
	case sgLight:
		g.bHP--
		g.setMsg(fmt.Sprintf("Light is toxic to B! HP:%d", g.bHP))
		if g.sfxToxic != nil {
			playSFX(g.sfxToxic, 0.40)
		}
	case sgSymbiote:
		g.decouple += 4
		g.setMsg("Symbiosis! Tab to switch, 4 free moves")
		if g.sfxSymbiosis != nil {
			playSFX(g.sfxSymbiosis, 0.40)
		}
	case sgTrailA:
		// A's trail is safe for B
	}
	g.grid[g.by][g.bx] = sgEmpty
}

func (g *SymbioticGrid) checkWin() {
	if g.aSize >= g.targetA && g.bSize >= g.targetB {
		g.phase = sgWon
		if g.sfxWin != nil {
			playSFX(g.sfxWin, 0.40)
		}
	}
}

func (g *SymbioticGrid) checkDeath() {
	if g.aHP <= 0 || g.bHP <= 0 {
		g.phase = sgDead
		if g.aHP <= 0 {
			g.msg = "Organism A died from toxicity!"
		} else {
			g.msg = "Organism B died from toxicity!"
		}
		if g.sfxDeath != nil {
			playSFX(g.sfxDeath, 0.45)
		}
	}
}

func (g *SymbioticGrid) setMsg(s string) {
	g.msg = s
	g.msgTimer = 90
}

// --- Drawing ---

func (g *SymbioticGrid) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x12, 0x14, 0x20, 0xff})
	g.drawHUD(screen)
	g.drawGrid(screen)
	g.drawOrganisms(screen)
	g.drawOverlays(screen)
}

func (g *SymbioticGrid) drawHUD(screen *ebiten.Image) {
	// Level + moves
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Level %d   Moves: %d", g.level, g.moves), 10, 6)

	// Organism A stats
	aColor := color.RGBA{0xff, 0xe0, 0x40, 0xff}
	vector.FillRect(screen, 10, 22, 10, 10, aColor, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("A: %d/%d  HP:%d", g.aSize, g.targetA, g.aHP), 24, 22)

	// Organism B stats
	bColor := color.RGBA{0x60, 0x40, 0xff, 0xff}
	vector.FillRect(screen, 10, 38, 10, 10, bColor, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("B: %d/%d  HP:%d", g.bSize, g.targetB, g.bHP), 24, 38)

	// Decouple indicator
	if g.decouple > 0 {
		which := "A"
		if !g.moveA {
			which = "B"
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SYMBIOSIS: %d moves (moving %s, Tab=switch)", g.decouple, which), 180, 6)
	}

	// Message
	if g.msgTimer > 0 && g.msg != "" {
		ebitenutil.DebugPrintAt(screen, g.msg, 10, 54)
	}

	// Controls
	ebitenutil.DebugPrintAt(screen, "Arrows=move  R=restart  ESC=menu", sgScreenW-210, 54)
}

func (g *SymbioticGrid) drawGrid(screen *ebiten.Image) {
	ox := float32(sgMargin)
	oy := float32(sgHudH)

	for r := 0; r < sgGridH; r++ {
		for c := 0; c < sgGridW; c++ {
			x := ox + float32(c*sgCellSize)
			y := oy + float32(r*sgCellSize)
			w := float32(sgCellSize)

			cell := g.grid[r][c]
			bg := color.RGBA{0x1a, 0x1c, 0x2c, 0xff} // empty

			switch cell {
			case sgLight:
				bg = color.RGBA{0xff, 0xf0, 0x80, 0xff} // warm yellow
			case sgDark:
				bg = color.RGBA{0x30, 0x20, 0x60, 0xff} // deep purple
			case sgNeutral:
				bg = color.RGBA{0x40, 0x60, 0x40, 0xff} // green-gray
			case sgWall:
				bg = color.RGBA{0x50, 0x50, 0x50, 0xff} // gray
			case sgSymbiote:
				// Pulsing magenta
				pulse := byte(0xa0 + (g.moves*17+r*7+c*13)%40)
				bg = color.RGBA{pulse, 0x30, pulse, 0xff}
			case sgTrailA:
				bg = color.RGBA{0x3a, 0x34, 0x18, 0xff} // faint gold
			case sgTrailB:
				bg = color.RGBA{0x18, 0x14, 0x3a, 0xff} // faint blue
			}

			vector.FillRect(screen, x, y, w, w, bg, false)

			// Cell decorations
			t := float64(g.sgTicks)
			switch cell {
			case sgLight:
				// Sun with radiating rays
				cx, cy := x+w/2, y+w/2
				vector.FillCircle(screen, cx, cy, 6, color.RGBA{0xff, 0xff, 0xc0, 0xff}, true)
				// Radiating light rays
				for ri := 0; ri < 8; ri++ {
					angle := float64(ri)*math.Pi/4 + t*0.03
					rayLen := float32(8 + math.Sin(t*0.05+float64(ri)*0.8)*3)
					rx := cx + float32(math.Cos(angle))*rayLen
					ry := cy + float32(math.Sin(angle))*rayLen
					vector.StrokeLine(screen, cx+float32(math.Cos(angle))*7, cy+float32(math.Sin(angle))*7,
						rx, ry, 0.8, color.RGBA{0xff, 0xff, 0x80, 0x80}, false)
				}
			case sgDark:
				// Moon crescent with swirling dots
				cx, cy := x+w/2, y+w/2
				vector.FillCircle(screen, cx, cy, 6, color.RGBA{0x60, 0x50, 0xa0, 0xff}, true)
				vector.FillCircle(screen, cx+3, cy-2, 5, bg, true)
				// Swirling dots around the dark cell
				for di := 0; di < 4; di++ {
					angle := t*0.06 + float64(di)*math.Pi/2
					swR := float32(10 + math.Sin(t*0.04+float64(di))*2)
					dx := cx + float32(math.Cos(angle))*swR
					dy := cy + float32(math.Sin(angle))*swR
					vector.FillCircle(screen, dx, dy, 1.2, color.RGBA{0x80, 0x60, 0xc0, 0x80}, false)
				}
			case sgWall:
				// Brick pattern
				vector.StrokeLine(screen, x+2, y+w/2, x+w-2, y+w/2, 1, color.RGBA{0x40, 0x40, 0x40, 0xff}, false)
				vector.StrokeLine(screen, x+w/2, y+2, x+w/2, y+w/2, 1, color.RGBA{0x40, 0x40, 0x40, 0xff}, false)
			case sgSymbiote:
				// DNA helix symbol with pulse
				pulseAlpha := uint8(0xc0 + int(math.Sin(t*0.08)*0x30))
				vector.FillCircle(screen, x+w/2, y+w/2, 5, color.RGBA{pulseAlpha, 0x30, pulseAlpha, 0xff}, true)
				ebitenutil.DebugPrintAt(screen, "~", int(x)+13, int(y)+9)
			case sgNeutral:
				vector.FillCircle(screen, x+w/2, y+w/2, 3, color.RGBA{0x60, 0x80, 0x60, 0xff}, true)
			case sgTrailA:
				// Fading glow effect for trail A
				glowAlpha := uint8(0x20 + int(math.Sin(t*0.04+float64(r*3+c*5))*0x10))
				vector.FillCircle(screen, x+w/2, y+w/2, 8, color.RGBA{0xff, 0xe0, 0x40, glowAlpha}, false)
			case sgTrailB:
				// Fading glow effect for trail B
				glowAlpha := uint8(0x20 + int(math.Sin(t*0.04+float64(r*5+c*3))*0x10))
				vector.FillCircle(screen, x+w/2, y+w/2, 8, color.RGBA{0x60, 0x40, 0xff, glowAlpha}, false)
			}

			// Grid line
			vector.StrokeRect(screen, x, y, w, w, 1, color.RGBA{0x25, 0x28, 0x38, 0xff}, false)
		}
	}
}

func (g *SymbioticGrid) drawOrganisms(screen *ebiten.Image) {
	ox := float32(sgMargin)
	oy := float32(sgHudH)
	t := float64(g.sgTicks)

	// Breathing size oscillation
	breatheA := float32(math.Sin(t*0.08) * 1.5)
	breatheB := float32(math.Sin(t*0.08+math.Pi*0.5) * 1.5)

	// Organism A — light creature (golden)
	ax := ox + float32(g.ax*sgCellSize) + float32(sgCellSize)/2
	ay := oy + float32(g.ay*sgCellSize) + float32(sgCellSize)/2
	sizeA := float32(8+g.aSize) + breatheA
	if sizeA > float32(sgCellSize)/2-2 {
		sizeA = float32(sgCellSize)/2 - 2
	}
	if sizeA < 6 {
		sizeA = 6
	}
	// Glow (pulsing)
	glowAlphaA := uint8(0x20 + int(math.Sin(t*0.06)*0x10))
	vector.FillCircle(screen, ax, ay, sizeA+5, color.RGBA{0xff, 0xe0, 0x40, glowAlphaA}, true)
	// Body
	vector.FillCircle(screen, ax, ay, sizeA, color.RGBA{0xff, 0xe0, 0x40, 0xff}, true)
	vector.StrokeCircle(screen, ax, ay, sizeA, 1.5, color.RGBA{0xff, 0xff, 0x80, 0xff}, true)
	// Eye
	vector.FillCircle(screen, ax+2, ay-2, 2, color.RGBA{0x40, 0x20, 0x00, 0xff}, true)
	// Label
	ebitenutil.DebugPrintAt(screen, "A", int(ax)-3, int(ay)-6)

	// Organism B — dark creature (indigo)
	bx := ox + float32(g.bx*sgCellSize) + float32(sgCellSize)/2
	by := oy + float32(g.by*sgCellSize) + float32(sgCellSize)/2
	sizeB := float32(8+g.bSize) + breatheB
	if sizeB > float32(sgCellSize)/2-2 {
		sizeB = float32(sgCellSize)/2 - 2
	}
	if sizeB < 6 {
		sizeB = 6
	}
	// Glow (pulsing)
	glowAlphaB := uint8(0x20 + int(math.Sin(t*0.06+1.5)*0x10))
	vector.FillCircle(screen, bx, by, sizeB+5, color.RGBA{0x60, 0x40, 0xff, glowAlphaB}, true)
	// Body
	vector.FillCircle(screen, bx, by, sizeB, color.RGBA{0x60, 0x40, 0xff, 0xff}, true)
	vector.StrokeCircle(screen, bx, by, sizeB, 1.5, color.RGBA{0x90, 0x70, 0xff, 0xff}, true)
	// Eye
	vector.FillCircle(screen, bx-2, by-2, 2, color.RGBA{0xc0, 0xc0, 0xff, 0xff}, true)
	// Label
	ebitenutil.DebugPrintAt(screen, "B", int(bx)-3, int(by)-6)

	// Connection line between organisms when linked (not decoupled)
	if g.decouple == 0 && g.phase == sgPlaying {
		// Faint pulsing line connecting A and B
		lineAlpha := uint8(0x25 + int(math.Sin(t*0.05)*0x15))
		// Draw segmented line for visual interest
		steps := 12
		for i := 0; i < steps; i++ {
			frac := float64(i) / float64(steps)
			fracNext := float64(i+1) / float64(steps)
			x1 := ax + (bx-ax)*float32(frac)
			y1 := ay + (by-ay)*float32(frac)
			x2 := ax + (bx-ax)*float32(fracNext)
			y2 := ay + (by-ay)*float32(fracNext)
			// Add slight wave to the line
			wave := float32(math.Sin(t*0.06+frac*math.Pi*2) * 4)
			perpX := -(by - ay) / float32(math.Sqrt(float64((bx-ax)*(bx-ax)+(by-ay)*(by-ay)))+0.01)
			perpY := (bx - ax) / float32(math.Sqrt(float64((bx-ax)*(bx-ax)+(by-ay)*(by-ay)))+0.01)
			vector.StrokeLine(screen, x1+perpX*wave, y1+perpY*wave, x2+perpX*wave, y2+perpY*wave,
				1, color.RGBA{0xb0, 0x90, 0xff, lineAlpha}, false)
		}
	}

	// Active indicator when decoupled
	if g.decouple > 0 {
		activeX, activeY := ax, ay
		activeSize := sizeA
		if !g.moveA {
			activeX, activeY = bx, by
			activeSize = sizeB
		}
		// Pulsing selection ring
		ringPulse := float32(math.Sin(t*0.12) * 2)
		vector.StrokeCircle(screen, activeX, activeY, activeSize+5+ringPulse, 1.5, color.RGBA{0xff, 0xff, 0xff, 0x80}, true)
	}
}

func (g *SymbioticGrid) drawOverlays(screen *ebiten.Image) {
	switch g.phase {
	case sgLevelIntro:
		vector.FillRect(screen, 0, 0, sgScreenW, sgScreenH, color.RGBA{0, 0, 0, 0xaa}, false)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("LEVEL %d", g.level), sgScreenW/2-24, sgScreenH/2-30)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Feed A %d light cells", g.targetA), sgScreenW/2-60, sgScreenH/2-10)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Feed B %d dark cells", g.targetB), sgScreenW/2-60, sgScreenH/2+6)
		ebitenutil.DebugPrintAt(screen, "A=normal move  B=mirrored X", sgScreenW/2-84, sgScreenH/2+26)
		ebitenutil.DebugPrintAt(screen, "Light=food for A, toxic for B", sgScreenW/2-90, sgScreenH/2+42)
		ebitenutil.DebugPrintAt(screen, "Dark=food for B, toxic for A", sgScreenW/2-90, sgScreenH/2+58)
		ebitenutil.DebugPrintAt(screen, "Press SPACE to start", sgScreenW/2-60, sgScreenH/2+82)

	case sgWon:
		vector.FillRect(screen, 0, 0, sgScreenW, sgScreenH, color.RGBA{0x10, 0x30, 0x10, 0xcc}, false)
		ebitenutil.DebugPrintAt(screen, "LEVEL COMPLETE!", sgScreenW/2-48, sgScreenH/2-20)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Moves: %d", g.moves), sgScreenW/2-30, sgScreenH/2)
		ebitenutil.DebugPrintAt(screen, "SPACE = next level", sgScreenW/2-54, sgScreenH/2+20)

	case sgDead:
		vector.FillRect(screen, 0, 0, sgScreenW, sgScreenH, color.RGBA{0x30, 0x10, 0x10, 0xcc}, false)
		ebitenutil.DebugPrintAt(screen, g.msg, sgScreenW/2-70, sgScreenH/2-20)
		ebitenutil.DebugPrintAt(screen, "SPACE = retry  R = restart", sgScreenW/2-78, sgScreenH/2+10)
	}
}
