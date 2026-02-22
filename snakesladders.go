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
	slCellSize  = 50
	slBoardSize = 10
	slBoardW    = slBoardSize * slCellSize // 500
	slBoardH    = slBoardSize * slCellSize // 500
	slUIHeight  = 60
	slScreenW   = slBoardW       // 500
	slScreenH   = slBoardH + slUIHeight // 560
)

// Game phases.
type slPhase int

const (
	slPlayerTurn slPhase = iota
	slPlayerRolled
	slPlayerMoving
	slPlayerSnakeOrLadder
	slAIDelay
	slAIRolled
	slAIMoving
	slAISnakeOrLadder
	slGameOver
)

// A snake or ladder link: from -> to.
type slLink struct {
	from int
	to   int
}

// Predefined snakes (head -> tail, from > to).
var slSnakes = []slLink{
	{16, 6},
	{47, 26},
	{49, 11},
	{56, 53},
	{62, 19},
	{64, 60},
	{87, 24},
	{93, 73},
	{95, 75},
	{98, 78},
}

// Predefined ladders (bottom -> top, from < to).
var slLadders = []slLink{
	{1, 38},
	{4, 14},
	{9, 31},
	{21, 42},
	{28, 84},
	{36, 44},
	{51, 67},
	{71, 91},
	{80, 100},
}

// ---------------------------------------------------------------------------
// Audio — Snakes & Ladders
// ---------------------------------------------------------------------------

// slGenerateBGM creates a whimsical, carnival-like board-game melody.
// Triangle wave, C-major, ~100 bpm, bouncy rhythm.
func slGenerateBGM() []byte {
	// Playful melody — C major pentatonic carnival feel
	// MIDI notes: C5=72, D5=74, E5=76, G5=79, A5=81, C6=84
	melody := [][2]int{
		// Phrase 1 -- bouncy opening
		{72, 1}, {76, 1}, {79, 1}, {76, 1},
		{84, 2}, {81, 1}, {79, 1},
		{76, 1}, {74, 1}, {72, 2},
		{0, 1}, {74, 1}, {76, 1}, {79, 1},
		// Phrase 2 -- ascending carnival run
		{72, 1}, {74, 1}, {76, 1}, {79, 1},
		{81, 2}, {84, 2},
		{81, 1}, {79, 1}, {76, 1}, {74, 1},
		{72, 2}, {0, 2},
		// Phrase 3 -- playful call and response
		{84, 1}, {81, 1}, {79, 2},
		{76, 1}, {74, 1}, {72, 2},
		{74, 2}, {76, 2},
		{79, 1}, {81, 1}, {84, 2},
		// Phrase 4 -- ending with flourish
		{79, 1}, {76, 1}, {79, 1}, {81, 1},
		{84, 2}, {0, 1}, {81, 1},
		{79, 1}, {76, 1}, {74, 1}, {72, 1},
		{72, 2}, {0, 2},
		// B section -- F major detour, new melody
		{77, 1}, {76, 1}, {74, 1}, {72, 1},
		{74, 2}, {77, 2},
		{79, 1}, {77, 1}, {74, 1}, {72, 1},
		{69, 2}, {72, 2},
		{74, 1}, {77, 1}, {79, 1}, {81, 1},
		{84, 2}, {81, 1}, {79, 1},
		{77, 1}, {74, 1}, {72, 2},
		{0, 2},
		// B' -- Am surprise, playful tension
		{69, 1}, {72, 1}, {76, 1}, {79, 1},
		{81, 2}, {79, 1}, {76, 1},
		{74, 1}, {72, 1}, {69, 2},
		{67, 1}, {69, 1}, {72, 2},
		{76, 1}, {79, 1}, {81, 1}, {84, 1},
		{86, 2}, {84, 1}, {81, 1},
		{79, 1}, {76, 1}, {74, 1}, {72, 1},
		{72, 2}, {0, 2},
		// Return A -- recap first phrase
		{72, 1}, {76, 1}, {79, 1}, {76, 1},
		{84, 2}, {81, 1}, {79, 1},
		{76, 1}, {74, 1}, {72, 2},
		{0, 1}, {74, 1}, {76, 1}, {79, 1},
		{72, 1}, {74, 1}, {76, 1}, {79, 1},
		{84, 2}, {81, 1}, {79, 1},
		{76, 1}, {74, 1}, {72, 2},
		{72, 2}, {0, 2},
	}
	// Bouncy bass line in low register
	bass := [][2]int{
		// A section
		{48, 2}, {48, 2}, {55, 2}, {55, 2},
		{52, 2}, {52, 2}, {48, 2}, {48, 2},
		{48, 2}, {48, 2}, {55, 2}, {55, 2},
		{53, 2}, {53, 2}, {48, 2}, {48, 2},
		{48, 2}, {55, 2}, {52, 2}, {53, 2},
		{48, 2}, {55, 2}, {52, 2}, {48, 2},
		{55, 2}, {53, 2}, {48, 2}, {52, 2},
		{55, 2}, {53, 2}, {48, 2}, {48, 2},
		// B section -- F major bass
		{53, 2}, {53, 2}, {48, 2}, {48, 2},
		{53, 2}, {55, 2}, {48, 2}, {48, 2},
		{53, 2}, {53, 2}, {55, 2}, {55, 2},
		{53, 2}, {48, 2}, {48, 2}, {48, 2},
		// B' -- Am bass
		{45, 2}, {45, 2}, {48, 2}, {48, 2},
		{52, 2}, {52, 2}, {45, 2}, {45, 2},
		{48, 2}, {52, 2}, {55, 2}, {55, 2},
		{48, 2}, {48, 2}, {48, 2}, {48, 2},
		// Return
		{48, 2}, {48, 2}, {55, 2}, {55, 2},
		{52, 2}, {52, 2}, {48, 2}, {48, 2},
		{48, 2}, {48, 2}, {55, 2}, {55, 2},
		{53, 2}, {53, 2}, {48, 2}, {48, 2},
	}
	drums := generateDrumPattern("K.h.s.h.K.h.s.h.K.h.s.h.K.h.shshK.h.s.hSK.h.sKh.K.h.s.h.K.hxKshs", 100, 0.25)
	melodyPCM := generateTrack(melody, triangleWave, 100, 0.30)
	bassPCM := generateTrack(bass, squareWave, 100, 0.12)
	mixed := mixBuffers(melodyPCM, bassPCM)
	// Mix in drums — repeat drums to match melody length
	for len(drums) < len(mixed) {
		drums = append(drums, drums...)
	}
	drums = drums[:len(mixed)]
	mixed = mixBuffers(mixed, drums)
	return mixed
}

// slGenerateSFXDice creates a short rattling noise burst (dice roll).
func slGenerateSFXDice() []byte {
	dur := 0.25
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(samples)
		// Rapid clicks mixed with noise, fading out
		click := squareWave(t, 300+200*math.Sin(t*80)) * 0.3
		n := noise() * 0.5
		env := 1.0 - tFrac
		val := (click + n) * env * 0.4
		writeSample(buf, i*4, val)
	}
	return buf
}

// slGenerateSFXClimb creates an ascending arpeggio (ladder).
func slGenerateSFXClimb() []byte {
	// Fast ascending arpeggio: C5-E5-G5-C6
	notes := [][2]int{{72, 1}, {76, 1}, {79, 1}, {84, 2}}
	return generateTrack(notes, triangleWave, 480, 0.50)
}

// slGenerateSFXSlide creates a descending tone (snake).
func slGenerateSFXSlide() []byte {
	dur := 0.4
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(samples)
		// Descending frequency sweep from 800 Hz down to 150 Hz
		freq := 800.0 - 650.0*tFrac
		env := envelope(tFrac, 0.02, 0.5, 0.48)
		val := sawWave(t, freq) * env * 0.35
		writeSample(buf, i*4, val)
	}
	return buf
}

// slGenerateSFXWin creates a win fanfare.
func slGenerateSFXWin() []byte {
	// Triumphant fanfare: C5-E5-G5 hold, then C6 hold
	notes := [][2]int{
		{72, 2}, {76, 2}, {79, 2}, {84, 4},
		{0, 1}, {84, 1}, {86, 1}, {88, 3},
	}
	return generateTrack(notes, triangleWave, 200, 0.40)
}

// SnakesLadders is the main game struct.
type SnakesLadders struct {
	goBack bool

	phase     slPhase
	playerPos int // 0 means not on board yet, 1-100
	aiPos     int
	dieResult int
	message   string

	// Animation state
	animFrom    int
	animTo      int
	animCurrent int
	animTick    int

	// AI delay counter
	aiDelayTick int

	// Lookup map: square -> destination (for snakes and ladders combined)
	links map[int]int

	// Audio
	bgmPlayer  *audio.Player
	sfxDice    []byte
	sfxClimb   []byte
	sfxSlide   []byte
	sfxWin     []byte
}

func newSnakesLadders() *SnakesLadders {
	g := &SnakesLadders{
		phase:   slPlayerTurn,
		message: "Your turn! Roll the die.",
	}
	g.links = make(map[int]int)
	for _, s := range slSnakes {
		g.links[s.from] = s.to
	}
	for _, l := range slLadders {
		g.links[l.from] = l.to
	}

	// Initialise audio
	g.sfxDice = slGenerateSFXDice()
	g.sfxClimb = slGenerateSFXClimb()
	g.sfxSlide = slGenerateSFXSlide()
	g.sfxWin = slGenerateSFXWin()
	g.bgmPlayer = loopPlayer(slGenerateBGM(), 0.18)
	g.bgmPlayer.Play()

	return g
}

func (g *SnakesLadders) WantsBack() bool { return g.goBack }

func (g *SnakesLadders) Layout(w, h int) (int, int) {
	return slScreenW, slScreenH
}

// slSquareToPixel converts a board square (1-100) to pixel coordinates (center of cell).
func slSquareToPixel(sq int) (float32, float32) {
	if sq < 1 {
		sq = 1
	}
	if sq > 100 {
		sq = 100
	}
	idx := sq - 1
	row := idx / slBoardSize // 0 = bottom row
	col := idx % slBoardSize
	// Even rows (0,2,4,...) go left-to-right, odd rows go right-to-left.
	if row%2 == 1 {
		col = slBoardSize - 1 - col
	}
	// Row 0 is the bottom of the screen (board area).
	pixelX := float32(col*slCellSize) + float32(slCellSize)/2
	pixelY := float32((slBoardSize-1-row)*slCellSize) + float32(slCellSize)/2
	return pixelX, pixelY
}

func (g *SnakesLadders) slReset() {
	g.phase = slPlayerTurn
	g.playerPos = 0
	g.aiPos = 0
	g.dieResult = 0
	g.message = "Your turn! Roll the die."
	g.animTick = 0
	g.aiDelayTick = 0
}

func (g *SnakesLadders) Update() error {
	// ESC -> go back to menu.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.bgmPlayer != nil {
			g.bgmPlayer.Pause()
		}
		g.goBack = true
		return nil
	}
	// R -> restart.
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.slReset()
		return nil
	}

	switch g.phase {
	case slPlayerTurn:
		rolled := false
		// Space or click ROLL button.
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			rolled = true
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			// ROLL button region.
			bx, by, bw, bh := slRollButtonRect()
			if mx >= bx && mx < bx+bw && my >= by && my < by+bh {
				rolled = true
			}
		}
		if rolled {
			playSFX(g.sfxDice, 0.40)
			g.dieResult = rand.Intn(6) + 1
			g.message = fmt.Sprintf("You rolled %d!", g.dieResult)
			target := g.playerPos + g.dieResult
			if target > 100 {
				// Can't move past 100; stay in place.
				g.message = fmt.Sprintf("You rolled %d - too high! Stay.", g.dieResult)
				g.phase = slAIDelay
				g.aiDelayTick = 0
			} else {
				g.animFrom = g.playerPos
				g.animTo = target
				g.animCurrent = g.playerPos
				g.animTick = 0
				g.phase = slPlayerMoving
			}
		}

	case slPlayerMoving:
		g.animTick++
		if g.animTick >= 8 { // move one step every 8 frames
			g.animTick = 0
			if g.animCurrent < g.animTo {
				g.animCurrent++
			}
			g.playerPos = g.animCurrent
			if g.animCurrent >= g.animTo {
				// Arrived at target.
				if g.playerPos >= 100 {
					g.playerPos = 100
					g.message = "YOU WIN! Press R to restart."
					g.phase = slGameOver
					playSFX(g.sfxWin, 0.45)
					return nil
				}
				// Check snake or ladder.
				if dest, ok := g.links[g.playerPos]; ok {
					g.phase = slPlayerSnakeOrLadder
					g.animFrom = g.playerPos
					g.animTo = dest
					g.animCurrent = g.playerPos
					g.animTick = 0
					if dest < g.playerPos {
						g.message = fmt.Sprintf("Snake! %d -> %d", g.playerPos, dest)
						playSFX(g.sfxSlide, 0.40)
					} else {
						g.message = fmt.Sprintf("Ladder! %d -> %d", g.playerPos, dest)
						playSFX(g.sfxClimb, 0.40)
					}
				} else {
					// No link; go to AI turn.
					g.phase = slAIDelay
					g.aiDelayTick = 0
				}
			}
		}

	case slPlayerSnakeOrLadder:
		g.animTick++
		if g.animTick >= 4 {
			g.animTick = 0
			if g.animFrom < g.animTo {
				// Ladder: move up.
				g.animCurrent++
				if g.animCurrent > g.animTo {
					g.animCurrent = g.animTo
				}
			} else {
				// Snake: move down.
				g.animCurrent--
				if g.animCurrent < g.animTo {
					g.animCurrent = g.animTo
				}
			}
			g.playerPos = g.animCurrent
			if g.animCurrent == g.animTo {
				if g.playerPos >= 100 {
					g.playerPos = 100
					g.message = "YOU WIN! Press R to restart."
					g.phase = slGameOver
					playSFX(g.sfxWin, 0.45)
					return nil
				}
				g.phase = slAIDelay
				g.aiDelayTick = 0
			}
		}

	case slAIDelay:
		g.aiDelayTick++
		if g.aiDelayTick >= 45 { // ~0.75 seconds at 60fps
			playSFX(g.sfxDice, 0.35)
			g.dieResult = rand.Intn(6) + 1
			g.message = fmt.Sprintf("AI rolled %d!", g.dieResult)
			target := g.aiPos + g.dieResult
			if target > 100 {
				g.message = fmt.Sprintf("AI rolled %d - too high! Stay.", g.dieResult)
				g.phase = slPlayerTurn
				g.message += " Your turn!"
			} else {
				g.animFrom = g.aiPos
				g.animTo = target
				g.animCurrent = g.aiPos
				g.animTick = 0
				g.phase = slAIMoving
			}
		}

	case slAIMoving:
		g.animTick++
		if g.animTick >= 8 {
			g.animTick = 0
			if g.animCurrent < g.animTo {
				g.animCurrent++
			}
			g.aiPos = g.animCurrent
			if g.animCurrent >= g.animTo {
				if g.aiPos >= 100 {
					g.aiPos = 100
					g.message = "AI WINS! Press R to restart."
					g.phase = slGameOver
					return nil
				}
				if dest, ok := g.links[g.aiPos]; ok {
					g.phase = slAISnakeOrLadder
					g.animFrom = g.aiPos
					g.animTo = dest
					g.animCurrent = g.aiPos
					g.animTick = 0
					if dest < g.aiPos {
						g.message = fmt.Sprintf("AI hit snake! %d -> %d", g.aiPos, dest)
						playSFX(g.sfxSlide, 0.35)
					} else {
						g.message = fmt.Sprintf("AI hit ladder! %d -> %d", g.aiPos, dest)
						playSFX(g.sfxClimb, 0.35)
					}
				} else {
					g.phase = slPlayerTurn
					g.message = "Your turn! Roll the die."
				}
			}
		}

	case slAISnakeOrLadder:
		g.animTick++
		if g.animTick >= 4 {
			g.animTick = 0
			if g.animFrom < g.animTo {
				g.animCurrent++
				if g.animCurrent > g.animTo {
					g.animCurrent = g.animTo
				}
			} else {
				g.animCurrent--
				if g.animCurrent < g.animTo {
					g.animCurrent = g.animTo
				}
			}
			g.aiPos = g.animCurrent
			if g.animCurrent == g.animTo {
				if g.aiPos >= 100 {
					g.aiPos = 100
					g.message = "AI WINS! Press R to restart."
					g.phase = slGameOver
					return nil
				}
				g.phase = slPlayerTurn
				g.message = "Your turn! Roll the die."
			}
		}

	case slGameOver:
		// Nothing to update; wait for R or ESC.
	}

	return nil
}

// slRollButtonRect returns x, y, w, h for the ROLL button.
func slRollButtonRect() (int, int, int, int) {
	return slScreenW - 80, slBoardH + 10, 70, 40
}

func (g *SnakesLadders) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x30, 0x30, 0x30, 0xff})

	// Draw the board.
	g.slDrawBoard(screen)

	// Draw ladders (behind pieces).
	g.slDrawLadders(screen)

	// Draw snakes (behind pieces).
	g.slDrawSnakes(screen)

	// Draw square numbers.
	g.slDrawSquareNumbers(screen)

	// Draw pieces.
	g.slDrawPieces(screen)

	// Draw UI panel.
	g.slDrawUI(screen)
}

func (g *SnakesLadders) slDrawBoard(screen *ebiten.Image) {
	// Two alternating colors for the checkerboard.
	colA := color.RGBA{0xFB, 0xE8, 0xC8, 0xff} // cream
	colB := color.RGBA{0xC8, 0xD8, 0xE8, 0xff} // light blue

	for sq := 1; sq <= 100; sq++ {
		idx := sq - 1
		row := idx / slBoardSize
		col := idx % slBoardSize
		if row%2 == 1 {
			col = slBoardSize - 1 - col
		}

		px := float32(col * slCellSize)
		py := float32((slBoardSize - 1 - row) * slCellSize)

		c := colA
		if (row+col)%2 == 1 {
			c = colB
		}
		vector.FillRect(screen, px, py, float32(slCellSize), float32(slCellSize), c, false)
		vector.StrokeRect(screen, px, py, float32(slCellSize), float32(slCellSize), 1, color.RGBA{0x88, 0x88, 0x88, 0xff}, false)
	}
}

func (g *SnakesLadders) slDrawSquareNumbers(screen *ebiten.Image) {
	for sq := 1; sq <= 100; sq++ {
		idx := sq - 1
		row := idx / slBoardSize
		col := idx % slBoardSize
		if row%2 == 1 {
			col = slBoardSize - 1 - col
		}
		px := col*slCellSize + 2
		py := (slBoardSize-1-row)*slCellSize + 2
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", sq), px, py)
	}
}

func (g *SnakesLadders) slDrawSnakes(screen *ebiten.Image) {
	snakeColor := color.RGBA{0xE0, 0x30, 0x30, 0xCC}
	snakeHeadColor := color.RGBA{0xFF, 0x00, 0x00, 0xFF}
	for _, s := range slSnakes {
		x1, y1 := slSquareToPixel(s.from)
		x2, y2 := slSquareToPixel(s.to)
		// Draw thick line for snake body.
		vector.StrokeLine(screen, x1, y1, x2, y2, 4, snakeColor, false)
		// Small circle at head.
		vector.FillCircle(screen, x1, y1, 6, snakeHeadColor, false)
		// Small circle at tail.
		vector.FillCircle(screen, x2, y2, 4, snakeColor, false)
	}
}

func (g *SnakesLadders) slDrawLadders(screen *ebiten.Image) {
	ladderColor := color.RGBA{0x30, 0xB0, 0x30, 0xCC}
	rungColor := color.RGBA{0x20, 0x90, 0x20, 0xCC}
	for _, l := range slLadders {
		x1, y1 := slSquareToPixel(l.from)
		x2, y2 := slSquareToPixel(l.to)

		// Two side rails offset from center.
		offX := float32(5)
		// Left rail.
		vector.StrokeLine(screen, x1-offX, y1, x2-offX, y2, 3, ladderColor, false)
		// Right rail.
		vector.StrokeLine(screen, x1+offX, y1, x2+offX, y2, 3, ladderColor, false)

		// Draw rungs along the ladder.
		dx := x2 - x1
		dy := y2 - y1
		steps := 5
		for i := 1; i < steps; i++ {
			t := float32(i) / float32(steps)
			rx := x1 + dx*t
			ry := y1 + dy*t
			vector.StrokeLine(screen, rx-offX, ry, rx+offX, ry, 2, rungColor, false)
		}
	}
}

func (g *SnakesLadders) slDrawPieces(screen *ebiten.Image) {
	playerColor := color.RGBA{0x30, 0x60, 0xFF, 0xFF}
	aiColor := color.RGBA{0xFF, 0x40, 0x40, 0xFF}

	// Determine display positions. During animation, use animCurrent.
	playerDisplay := g.playerPos
	aiDisplay := g.aiPos

	if g.phase == slPlayerMoving || g.phase == slPlayerSnakeOrLadder {
		playerDisplay = g.animCurrent
	}
	if g.phase == slAIMoving || g.phase == slAISnakeOrLadder {
		aiDisplay = g.animCurrent
	}

	// Draw player piece (blue).
	if playerDisplay > 0 {
		px, py := slSquareToPixel(playerDisplay)
		// Offset slightly left if both on same square.
		if playerDisplay == aiDisplay {
			px -= 8
		}
		vector.FillCircle(screen, px, py, 10, playerColor, false)
		vector.StrokeCircle(screen, px, py, 10, 2, color.White, false)
	}

	// Draw AI piece (red).
	if aiDisplay > 0 {
		px, py := slSquareToPixel(aiDisplay)
		if playerDisplay == aiDisplay {
			px += 8
		}
		vector.FillCircle(screen, px, py, 10, aiColor, false)
		vector.StrokeCircle(screen, px, py, 10, 2, color.White, false)
	}
}

func (g *SnakesLadders) slDrawUI(screen *ebiten.Image) {
	// Dark panel below the board.
	vector.FillRect(screen, 0, float32(slBoardH), float32(slScreenW), float32(slUIHeight), color.RGBA{0x20, 0x20, 0x30, 0xff}, false)

	// Status message.
	ebitenutil.DebugPrintAt(screen, g.message, 10, slBoardH+8)

	// Positions.
	posText := fmt.Sprintf("You: %d  AI: %d", g.playerPos, g.aiPos)
	ebitenutil.DebugPrintAt(screen, posText, 10, slBoardH+24)

	// Die result.
	if g.dieResult > 0 {
		dieText := fmt.Sprintf("Die: %d", g.dieResult)
		ebitenutil.DebugPrintAt(screen, dieText, 10, slBoardH+40)
	}

	// ROLL button (only during player's turn).
	bx, by, bw, bh := slRollButtonRect()
	btnColor := color.RGBA{0x50, 0x50, 0x70, 0xff}
	if g.phase == slPlayerTurn {
		mx, my := ebiten.CursorPosition()
		if mx >= bx && mx < bx+bw && my >= by && my < by+bh {
			btnColor = color.RGBA{0x70, 0x70, 0xA0, 0xff}
		}
	} else {
		btnColor = color.RGBA{0x40, 0x40, 0x50, 0xff}
	}
	vector.FillRect(screen, float32(bx), float32(by), float32(bw), float32(bh), btnColor, false)
	vector.StrokeRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 2, color.RGBA{0x90, 0x90, 0xB0, 0xff}, false)
	ebitenutil.DebugPrintAt(screen, "ROLL", bx+20, by+12)

	// Controls hint.
	ebitenutil.DebugPrintAt(screen, "Space/Click:Roll R:Restart ESC:Menu", 120, slBoardH+40)
}
