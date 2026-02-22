package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	msRows     = 16
	msCols     = 16
	msMines    = 40
	msCellSize = 32
	msTopBar   = 48
)

const (
	msScreenW = msCols * msCellSize
	msScreenH = msRows*msCellSize + msTopBar
)

type msCellState int

const (
	msHidden msCellState = iota
	msRevealed
	msFlagged
)

type msCell struct {
	mine   bool
	state  msCellState
	nearby int
}

type msStatus int

const (
	msPlaying msStatus = iota
	msWon
	msLost
)

type Minesweeper struct {
	board      [msRows][msCols]msCell
	status     msStatus
	startTime  time.Time
	elapsed    time.Duration
	minesLeft  int
	firstClick bool
	botActive  bool
	botTick    int
	goBack     bool
	ticks      int

	// Screen shake for mine explosion
	shakeFrames int
	shakeX      float64
	shakeY      float64

	// Audio
	bgm       *audio.Player
	sfxClick  []byte
	sfxBoom   []byte
	sfxWin    []byte
	audioInit bool
}

func newMinesweeper() *Minesweeper {
	m := &Minesweeper{
		minesLeft:  msMines,
		firstClick: true,
	}
	m.initMsAudio()
	return m
}

func (g *Minesweeper) initMsAudio() {
	// --- BGM: Tense, mysterious minor-key melody (~90 BPM) ---
	// Triangle wave melody - E minor / dorian, suspenseful and brooding
	// MIDI notes: E3=52, F#3=54, G3=55, A3=57, B3=59, C4=60, D4=62, E4=64
	melody := [][2]int{
		// A section - brooding Em theme
		{52, 2}, {55, 2}, {59, 1}, {57, 1}, {55, 2},
		{54, 2}, {57, 2}, {54, 1}, {52, 1}, {50, 2},
		{52, 2}, {55, 1}, {57, 1}, {59, 2}, {62, 2},
		{60, 1}, {59, 1}, {57, 2}, {55, 2}, {52, 2},
		// A' variation - different ending
		{55, 2}, {59, 2}, {57, 1}, {55, 1}, {52, 2},
		{50, 2}, {52, 1}, {55, 1}, {57, 2}, {59, 2},
		{62, 1}, {60, 1}, {59, 2}, {57, 2}, {0, 2},
		{52, 2}, {55, 2}, {57, 1}, {55, 1}, {52, 4},
		// B section - shift to G major, lighter tension release
		{55, 2}, {59, 2}, {62, 2}, {64, 2},
		{66, 1}, {64, 1}, {62, 2}, {59, 2}, {0, 2},
		{62, 2}, {66, 1}, {64, 1}, {62, 1}, {59, 1}, {55, 2},
		{57, 1}, {59, 1}, {62, 2}, {64, 4},
		// B' - builds higher
		{66, 2}, {64, 1}, {62, 1}, {59, 2}, {55, 2},
		{57, 2}, {59, 1}, {62, 1}, {64, 2}, {66, 2},
		{67, 2}, {66, 1}, {64, 1}, {62, 2}, {0, 2},
		{59, 1}, {57, 1}, {55, 2}, {52, 4},
		// C section - chromatic suspense, Am feel
		{57, 2}, {56, 2}, {55, 2}, {54, 2},
		{52, 1}, {55, 1}, {57, 2}, {60, 2}, {0, 2},
		{64, 2}, {63, 1}, {62, 1}, {60, 2}, {57, 2},
		{55, 1}, {54, 1}, {52, 2}, {50, 2}, {0, 2},
		// Return to A - darker restatement
		{52, 2}, {55, 2}, {59, 1}, {57, 1}, {55, 2},
		{54, 2}, {57, 2}, {54, 1}, {52, 1}, {50, 2},
		{52, 1}, {55, 1}, {57, 1}, {59, 1}, {62, 2}, {64, 2},
		{62, 1}, {60, 1}, {59, 2}, {57, 2}, {55, 2}, {52, 4},
	}
	melodyPCM := generateTrack(melody, triangleWave, 90, 0.35)

	// Square wave bass - slow root notes
	bass := [][2]int{
		// A section bass
		{40, 4}, {40, 4}, {42, 4}, {38, 4},
		{40, 4}, {43, 4}, {36, 4}, {40, 4},
		{43, 4}, {40, 4}, {38, 4}, {40, 4},
		{43, 4}, {38, 4}, {40, 4}, {40, 4},
		// B section bass - moves to G
		{43, 4}, {43, 4}, {47, 4}, {43, 4},
		{45, 4}, {43, 4}, {47, 4}, {43, 4},
		{43, 4}, {47, 4}, {45, 4}, {43, 4},
		{40, 4}, {38, 4}, {40, 4}, {40, 4},
		// C section bass - chromatic descent
		{45, 4}, {44, 4}, {43, 4}, {42, 4},
		{40, 4}, {43, 4}, {45, 4}, {40, 4},
		{36, 4}, {35, 4}, {33, 4}, {38, 4},
		// Return bass
		{40, 4}, {40, 4}, {42, 4}, {38, 4},
		{40, 4}, {43, 4}, {36, 4}, {40, 4}, {40, 4},
	}
	bassPCM := generateTrack(bass, squareWave, 90, 0.25)

	// Drum pattern - sparse, tense ticking
	drums := generateDrumPattern("h...h...X...h...h...h...X.h.h...h.....h.X...h.s.h...h...X.s.h...", 90, 0.25)

	mixed := mixBuffers(melodyPCM, bassPCM)
	mixed = mixBuffers(mixed, drums)
	g.bgm = loopPlayer(mixed, 0.18)

	// --- SFX: Click (short pluck) ---
	clickDur := 0.08
	clickBuf := pcmStereo(clickDur)
	for i := 0; i < len(clickBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / clickDur
		freq := 1200.0 + 400.0*(1.0-tFrac)
		val := squareWave(t, freq) * math.Exp(-t*50) * 0.5
		writeSample(clickBuf, i*4, val)
	}
	g.sfxClick = clickBuf

	// --- SFX: Boom (mine explosion) ---
	boomDur := 0.5
	boomBuf := pcmStereo(boomDur)
	for i := 0; i < len(boomBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / boomDur
		freq := 60.0 * math.Exp(-t*4)
		val := (noise()*0.6 + sineWave(t, freq)*0.4) * envelope(tFrac, 0.01, 0.15, 0.84) * 0.7
		writeSample(boomBuf, i*4, val)
	}
	g.sfxBoom = boomBuf

	// --- SFX: Win (ascending ding arpeggio) ---
	winDur := 0.6
	winBuf := pcmStereo(winDur)
	winNotes := []float64{
		noteFreq(72), // C5
		noteFreq(76), // E5
		noteFreq(79), // G5
		noteFreq(84), // C6
	}
	noteDur := winDur / float64(len(winNotes))
	for i := 0; i < len(winBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		noteIdx := int(t / noteDur)
		if noteIdx >= len(winNotes) {
			noteIdx = len(winNotes) - 1
		}
		localT := t - float64(noteIdx)*noteDur
		localFrac := localT / noteDur
		freq := winNotes[noteIdx]
		val := triangleWave(t, freq) * envelope(localFrac, 0.02, 0.5, 0.48) * 0.5
		writeSample(winBuf, i*4, val)
	}
	g.sfxWin = winBuf
}

func (g *Minesweeper) WantsBack() bool { return g.goBack }

func (g *Minesweeper) placeMines(safeR, safeC int) {
	placed := 0
	for placed < msMines {
		r := rand.Intn(msRows)
		c := rand.Intn(msCols)
		if abs(r-safeR) <= 1 && abs(c-safeC) <= 1 {
			continue
		}
		if g.board[r][c].mine {
			continue
		}
		g.board[r][c].mine = true
		placed++
	}
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			if g.board[r][c].mine {
				continue
			}
			count := 0
			g.forNeighbors(r, c, func(nr, nc int) {
				if g.board[nr][nc].mine {
					count++
				}
			})
			g.board[r][c].nearby = count
		}
	}
}

func (g *Minesweeper) forNeighbors(r, c int, fn func(int, int)) {
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			nr, nc := r+dr, c+dc
			if nr >= 0 && nr < msRows && nc >= 0 && nc < msCols {
				fn(nr, nc)
			}
		}
	}
}

func (g *Minesweeper) reveal(r, c int) {
	if r < 0 || r >= msRows || c < 0 || c >= msCols {
		return
	}
	cell := &g.board[r][c]
	if cell.state != msHidden {
		return
	}
	cell.state = msRevealed
	if cell.mine {
		g.status = msLost
		g.revealAll()
		return
	}
	if cell.nearby == 0 {
		g.forNeighbors(r, c, func(nr, nc int) {
			g.reveal(nr, nc)
		})
	}
}

func (g *Minesweeper) chordReveal(r, c int) {
	cell := &g.board[r][c]
	if cell.state != msRevealed || cell.nearby == 0 {
		return
	}
	flagCount := 0
	g.forNeighbors(r, c, func(nr, nc int) {
		if g.board[nr][nc].state == msFlagged {
			flagCount++
		}
	})
	if flagCount == cell.nearby {
		g.forNeighbors(r, c, func(nr, nc int) {
			if g.board[nr][nc].state == msHidden {
				g.reveal(nr, nc)
			}
		})
	}
}

func (g *Minesweeper) revealAll() {
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			g.board[r][c].state = msRevealed
		}
	}
}

func (g *Minesweeper) checkWin() {
	if g.status != msPlaying {
		return
	}
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			if !g.board[r][c].mine && g.board[r][c].state != msRevealed {
				return
			}
		}
	}
	g.status = msWon
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			if g.board[r][c].mine {
				g.board[r][c].state = msFlagged
			}
		}
	}
	g.minesLeft = 0
}

func (g *Minesweeper) Update() error {
	// Start BGM on first frame
	if !g.audioInit {
		g.audioInit = true
		if g.bgm != nil && !g.bgm.IsPlaying() {
			g.bgm.Play()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.bgm != nil && g.bgm.IsPlaying() {
			g.bgm.Pause()
		}
		g.goBack = true
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		oldBgm := g.bgm
		*g = *newMinesweeper()
		// Reuse existing bgm player if already playing
		if oldBgm != nil {
			oldBgm.Pause()
		}
		g.audioInit = false
		return nil
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		smileyX := msScreenW/2 - 16
		if mx >= smileyX && mx < smileyX+32 && my >= 8 && my < 40 {
			oldBgm := g.bgm
			*g = *newMinesweeper()
			if oldBgm != nil {
				oldBgm.Pause()
			}
			g.audioInit = false
			return nil
		}
		botX := msScreenW/2 + 28
		if mx >= botX && mx < botX+48 && my >= 8 && my < 40 {
			g.botActive = !g.botActive
		}
	}

	g.ticks++

	// Update screen shake
	if g.shakeFrames > 0 {
		g.shakeFrames--
		intensity := float64(g.shakeFrames) * 0.5
		g.shakeX = (rand.Float64()*2 - 1) * intensity
		g.shakeY = (rand.Float64()*2 - 1) * intensity
	} else {
		g.shakeX = 0
		g.shakeY = 0
	}

	if g.status != msPlaying {
		g.botActive = false
		return nil
	}

	if g.botActive {
		g.botTick++
		if g.botTick >= 8 {
			g.botTick = 0
			g.msBotStep()
		}
	}

	if !g.firstClick {
		g.elapsed = time.Since(g.startTime)
		if g.elapsed > 999*time.Second {
			g.elapsed = 999 * time.Second
		}
	}

	mx, my := ebiten.CursorPosition()
	c := mx / msCellSize
	r := (my - msTopBar) / msCellSize
	inBounds := r >= 0 && r < msRows && c >= 0 && c < msCols && my >= msTopBar

	if inBounds && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.firstClick {
			g.placeMines(r, c)
			g.firstClick = false
			g.startTime = time.Now()
		}
		prevStatus := g.status
		if g.board[r][c].state == msHidden {
			g.reveal(r, c)
			g.checkWin()
			// Play appropriate SFX based on result
			if g.status == msLost {
				playSFX(g.sfxBoom, 0.45)
				g.shakeFrames = 20
			} else if g.status == msWon {
				playSFX(g.sfxWin, 0.40)
			} else if prevStatus == msPlaying {
				playSFX(g.sfxClick, 0.35)
			}
		} else if g.board[r][c].state == msRevealed {
			g.chordReveal(r, c)
			g.checkWin()
			if g.status == msLost {
				playSFX(g.sfxBoom, 0.45)
				g.shakeFrames = 20
			} else if g.status == msWon {
				playSFX(g.sfxWin, 0.40)
			} else if prevStatus == msPlaying {
				playSFX(g.sfxClick, 0.35)
			}
		}
	}

	if inBounds && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		cell := &g.board[r][c]
		switch cell.state {
		case msHidden:
			cell.state = msFlagged
			g.minesLeft--
		case msFlagged:
			cell.state = msHidden
			g.minesLeft++
		}
	}

	return nil
}

// --- Bot solver ---

func (g *Minesweeper) msBotStep() {
	if g.status != msPlaying {
		g.botActive = false
		return
	}
	if g.firstClick {
		r, c := msRows/2, msCols/2
		g.placeMines(r, c)
		g.firstClick = false
		g.startTime = time.Now()
		g.reveal(r, c)
		g.checkWin()
		return
	}
	if g.msBotSolve() {
		return
	}
	g.msBotGuess()
}

type msCoord struct{ r, c int }

func (g *Minesweeper) msBotSolve() bool {
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			cell := g.board[r][c]
			if cell.state != msRevealed || cell.nearby == 0 {
				continue
			}
			hiddenCount, flagCount := 0, 0
			g.forNeighbors(r, c, func(nr, nc int) {
				switch g.board[nr][nc].state {
				case msHidden:
					hiddenCount++
				case msFlagged:
					flagCount++
				}
			})
			unflagged := cell.nearby - flagCount
			if unflagged > 0 && unflagged == hiddenCount {
				g.forNeighbors(r, c, func(nr, nc int) {
					if g.board[nr][nc].state == msHidden {
						g.board[nr][nc].state = msFlagged
						g.minesLeft--
					}
				})
				return true
			}
			if unflagged == 0 && hiddenCount > 0 {
				g.forNeighbors(r, c, func(nr, nc int) {
					if g.board[nr][nc].state == msHidden {
						g.reveal(nr, nc)
					}
				})
				g.checkWin()
				return true
			}
		}
	}

	// subset analysis
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			cellA := g.board[r][c]
			if cellA.state != msRevealed || cellA.nearby == 0 {
				continue
			}
			var hiddenA []msCoord
			flagA := 0
			g.forNeighbors(r, c, func(nr, nc int) {
				switch g.board[nr][nc].state {
				case msHidden:
					hiddenA = append(hiddenA, msCoord{nr, nc})
				case msFlagged:
					flagA++
				}
			})
			minesA := cellA.nearby - flagA
			if len(hiddenA) == 0 || minesA <= 0 {
				continue
			}
			g.forNeighbors(r, c, func(nr, nc int) {
				cellB := g.board[nr][nc]
				if cellB.state != msRevealed || cellB.nearby == 0 {
					return
				}
				var hiddenB []msCoord
				flagB := 0
				g.forNeighbors(nr, nc, func(nnr, nnc int) {
					switch g.board[nnr][nnc].state {
					case msHidden:
						hiddenB = append(hiddenB, msCoord{nnr, nnc})
					case msFlagged:
						flagB++
					}
				})
				minesB := cellB.nearby - flagB
				if len(hiddenB) == 0 || minesB <= 0 {
					return
				}
				if msIsSubset(hiddenA, hiddenB) && len(hiddenA) < len(hiddenB) {
					extra := minesB - minesA
					diff := msDiff(hiddenB, hiddenA)
					if extra == len(diff) {
						for _, p := range diff {
							if g.board[p.r][p.c].state == msHidden {
								g.board[p.r][p.c].state = msFlagged
								g.minesLeft--
							}
						}
					} else if extra == 0 {
						for _, p := range diff {
							if g.board[p.r][p.c].state == msHidden {
								g.reveal(p.r, p.c)
							}
						}
						g.checkWin()
					}
				}
			})
		}
	}
	return false
}

func msIsSubset(a, b []msCoord) bool {
	set := make(map[msCoord]bool, len(b))
	for _, p := range b {
		set[p] = true
	}
	for _, p := range a {
		if !set[p] {
			return false
		}
	}
	return true
}

func msDiff(b, a []msCoord) []msCoord {
	set := make(map[msCoord]bool, len(a))
	for _, p := range a {
		set[p] = true
	}
	var result []msCoord
	for _, p := range b {
		if !set[p] {
			result = append(result, p)
		}
	}
	return result
}

func (g *Minesweeper) msBotGuess() {
	var probs [msRows][msCols]float64
	var counts [msRows][msCols]int
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			probs[r][c] = -1
		}
	}
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			cell := g.board[r][c]
			if cell.state != msRevealed || cell.nearby == 0 {
				continue
			}
			hiddenN, flagN := 0, 0
			g.forNeighbors(r, c, func(nr, nc int) {
				switch g.board[nr][nc].state {
				case msHidden:
					hiddenN++
				case msFlagged:
					flagN++
				}
			})
			remaining := cell.nearby - flagN
			if hiddenN == 0 || remaining <= 0 {
				continue
			}
			prob := float64(remaining) / float64(hiddenN)
			g.forNeighbors(r, c, func(nr, nc int) {
				if g.board[nr][nc].state == msHidden {
					if probs[nr][nc] < 0 {
						probs[nr][nc] = 0
					}
					probs[nr][nc] += prob
					counts[nr][nc]++
				}
			})
		}
	}
	bestR, bestC := -1, -1
	bestProb := 999.0
	var noInfo []msCoord
	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			if g.board[r][c].state != msHidden {
				continue
			}
			if counts[r][c] > 0 {
				avg := probs[r][c] / float64(counts[r][c])
				if avg < bestProb {
					bestProb = avg
					bestR = r
					bestC = c
				}
			} else {
				noInfo = append(noInfo, msCoord{r, c})
			}
		}
	}
	if bestR >= 0 && (bestProb < 0.3 || len(noInfo) == 0) {
		g.reveal(bestR, bestC)
		g.checkWin()
		return
	}
	if len(noInfo) > 0 {
		pick := noInfo[rand.Intn(len(noInfo))]
		g.reveal(pick.r, pick.c)
		g.checkWin()
		return
	}
	if bestR >= 0 {
		g.reveal(bestR, bestC)
		g.checkWin()
	}
}

// --- Drawing ---

var msNumberColors = [9]color.RGBA{
	{0, 0, 0, 255},
	{0, 0, 255, 255},
	{0, 130, 0, 255},
	{255, 0, 0, 255},
	{0, 0, 128, 255},
	{128, 0, 0, 255},
	{0, 128, 128, 255},
	{0, 0, 0, 255},
	{128, 128, 128, 255},
}

func (g *Minesweeper) Draw(screen *ebiten.Image) {
	// Apply screen shake via an offscreen buffer
	target := screen
	if g.shakeFrames > 0 {
		target = ebiten.NewImage(msScreenW, msScreenH)
	}
	target.Fill(color.RGBA{0xc0, 0xc0, 0xc0, 0xff})
	g.drawTopBar(target)
	g.drawBoard(target)
	if g.shakeFrames > 0 {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(g.shakeX, g.shakeY)
		screen.DrawImage(target, op)
	}
}

func (g *Minesweeper) drawTopBar(screen *ebiten.Image) {
	vector.FillRect(screen, 8, 8, 56, 32, color.RGBA{0x20, 0x20, 0x20, 0xff}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(" %03d", g.minesLeft), 12, 16)

	vector.FillRect(screen, float32(msScreenW-64), 8, 56, 32, color.RGBA{0x20, 0x20, 0x20, 0xff}, false)
	secs := int(g.elapsed.Seconds())
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(" %03d", secs), msScreenW-60, 16)

	sx := float32(msScreenW/2 - 16)
	vector.FillRect(screen, sx, 8, 32, 32, color.RGBA{0xe0, 0xe0, 0x00, 0xff}, false)
	vector.StrokeRect(screen, sx, 8, 32, 32, 2, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
	face := ":)"
	if g.status == msLost {
		face = "X("
	} else if g.status == msWon {
		face = "B)"
	}
	ebitenutil.DebugPrintAt(screen, face, int(sx)+8, 16)

	botX := float32(msScreenW/2 + 28)
	botColor := color.RGBA{0xa0, 0xa0, 0xa0, 0xff}
	if g.botActive {
		botColor = color.RGBA{0x40, 0xb0, 0x40, 0xff}
	}
	vector.FillRect(screen, botX, 8, 48, 32, botColor, false)
	vector.StrokeRect(screen, botX, 8, 48, 32, 2, color.RGBA{0x60, 0x60, 0x60, 0xff}, false)
	ebitenutil.DebugPrintAt(screen, " BOT", int(botX)+4, 16)
}

func (g *Minesweeper) drawBoard(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	hoverC := mx / msCellSize
	hoverR := (my - msTopBar) / msCellSize

	for r := 0; r < msRows; r++ {
		for c := 0; c < msCols; c++ {
			x := float32(c * msCellSize)
			y := float32(r*msCellSize + msTopBar)
			cell := g.board[r][c]

			switch cell.state {
			case msHidden:
				vector.FillRect(screen, x, y, msCellSize, msCellSize, color.RGBA{0xc0, 0xc0, 0xc0, 0xff}, false)
				// Gradient-like 3D depth: lighter top-left quadrant, darker bottom-right
				vector.FillRect(screen, x, y, msCellSize/2, msCellSize/2, color.RGBA{0xff, 0xff, 0xff, 0x18}, false)
				vector.FillRect(screen, x+msCellSize/2, y+msCellSize/2, msCellSize/2, msCellSize/2, color.RGBA{0x00, 0x00, 0x00, 0x18}, false)
				vector.FillRect(screen, x, y, msCellSize, 2, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
				vector.FillRect(screen, x, y, 2, msCellSize, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
				vector.FillRect(screen, x, y+msCellSize-2, msCellSize, 2, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
				vector.FillRect(screen, x+msCellSize-2, y, 2, msCellSize, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
				if g.status == msPlaying && r == hoverR && c == hoverC && my >= msTopBar {
					vector.FillRect(screen, x+2, y+2, msCellSize-4, msCellSize-4, color.RGBA{0xd8, 0xd8, 0xd8, 0x60}, false)
				}

			case msFlagged:
				vector.FillRect(screen, x, y, msCellSize, msCellSize, color.RGBA{0xc0, 0xc0, 0xc0, 0xff}, false)
				// Gradient-like 3D depth on flagged cells too
				vector.FillRect(screen, x, y, msCellSize/2, msCellSize/2, color.RGBA{0xff, 0xff, 0xff, 0x18}, false)
				vector.FillRect(screen, x+msCellSize/2, y+msCellSize/2, msCellSize/2, msCellSize/2, color.RGBA{0x00, 0x00, 0x00, 0x18}, false)
				vector.FillRect(screen, x, y, msCellSize, 2, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
				vector.FillRect(screen, x, y, 2, msCellSize, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
				vector.FillRect(screen, x, y+msCellSize-2, msCellSize, 2, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
				vector.FillRect(screen, x+msCellSize-2, y, 2, msCellSize, color.RGBA{0x80, 0x80, 0x80, 0xff}, false)
				// Pulsing glow effect on flagged cells
				pulse := math.Sin(float64(g.ticks)*0.1+float64(r+c)*0.5)*0.5 + 0.5
				glowAlpha := uint8(30 + int(pulse*50))
				vector.FillRect(screen, x+2, y+2, msCellSize-4, msCellSize-4, color.RGBA{0xff, 0x40, 0x40, glowAlpha}, false)
				g.drawFlag(screen, x, y)

			case msRevealed:
				vector.FillRect(screen, x, y, msCellSize, msCellSize, color.RGBA{0xd0, 0xd0, 0xd0, 0xff}, false)
				vector.StrokeRect(screen, x, y, msCellSize, msCellSize, 1, color.RGBA{0xa0, 0xa0, 0xa0, 0xff}, false)
				if cell.mine {
					g.drawMine(screen, x, y)
				} else if cell.nearby > 0 {
					g.drawNumber(screen, x, y, cell.nearby)
				}
			}
		}
	}
}

func (g *Minesweeper) drawMine(screen *ebiten.Image, x, y float32) {
	cx := x + msCellSize/2
	cy := y + msCellSize/2
	r := float32(10)
	vector.FillCircle(screen, cx, cy, r, color.RGBA{0x20, 0x20, 0x20, 0xff}, true)
	vector.StrokeLine(screen, cx-r-3, cy, cx+r+3, cy, 2, color.RGBA{0x20, 0x20, 0x20, 0xff}, true)
	vector.StrokeLine(screen, cx, cy-r-3, cx, cy+r+3, 2, color.RGBA{0x20, 0x20, 0x20, 0xff}, true)
	vector.StrokeLine(screen, cx-r-1, cy-r-1, cx+r+1, cy+r+1, 2, color.RGBA{0x20, 0x20, 0x20, 0xff}, true)
	vector.StrokeLine(screen, cx+r+1, cy-r-1, cx-r-1, cy+r+1, 2, color.RGBA{0x20, 0x20, 0x20, 0xff}, true)
	vector.FillCircle(screen, cx-3, cy-3, 3, color.RGBA{0xff, 0xff, 0xff, 0xff}, true)
}

func (g *Minesweeper) drawFlag(screen *ebiten.Image, x, y float32) {
	px := x + msCellSize/2
	vector.StrokeLine(screen, px, y+6, px, y+msCellSize-8, 2, color.RGBA{0x20, 0x20, 0x20, 0xff}, true)
	var path vector.Path
	path.MoveTo(px, y+6)
	path.LineTo(px+10, y+12)
	path.LineTo(px, y+18)
	path.Close()
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = 1.0
		vs[i].ColorG = 0.0
		vs[i].ColorB = 0.0
		vs[i].ColorA = 1.0
	}
	screen.DrawTriangles(vs, is, whitePixel(), nil)
	vector.StrokeLine(screen, px-5, y+msCellSize-8, px+5, y+msCellSize-8, 2, color.RGBA{0x20, 0x20, 0x20, 0xff}, true)
}

var _msWhitePixel *ebiten.Image

func whitePixel() *ebiten.Image {
	if _msWhitePixel == nil {
		_msWhitePixel = ebiten.NewImage(3, 3)
		_msWhitePixel.Fill(color.White)
	}
	return _msWhitePixel
}

var msNumCache [9]*ebiten.Image

func msGetNumImage(n int) *ebiten.Image {
	if msNumCache[n] == nil {
		img := ebiten.NewImage(12, 16)
		ebitenutil.DebugPrint(img, fmt.Sprintf("%d", n))
		msNumCache[n] = img
	}
	return msNumCache[n]
}

func (g *Minesweeper) drawNumber(screen *ebiten.Image, x, y float32, n int) {
	col := msNumberColors[n]
	tx := int(x) + msCellSize/2 - 3
	ty := int(y) + msCellSize/2 - 8
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(tx), float64(ty))
	op.ColorScale.SetR(float32(col.R) / 255)
	op.ColorScale.SetG(float32(col.G) / 255)
	op.ColorScale.SetB(float32(col.B) / 255)
	op.ColorScale.SetA(1)
	screen.DrawImage(msGetNumImage(n), op)
}

func (g *Minesweeper) Layout(w, h int) (int, int) {
	return msScreenW, msScreenH
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
