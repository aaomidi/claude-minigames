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

// Board dimensions
const (
	bsGridSize = 10
	bsCellSize = 30
	bsGap      = 50  // gap between the two grids
	bsMarginX  = 20  // left margin
	bsMarginY  = 70  // top margin (room for header)
	bsLabelPad = 16  // space for row/col labels
)

const (
	bsScreenW = bsMarginX*2 + bsLabelPad*2 + bsGridSize*bsCellSize*2 + bsGap
	bsScreenH = bsMarginY + bsLabelPad + bsGridSize*bsCellSize + 50
)

// Ship definitions
type shipDef struct {
	name string
	size int
}

var bsShips = []shipDef{
	{"Carrier", 5},
	{"Battleship", 4},
	{"Cruiser", 3},
	{"Submarine", 3},
	{"Destroyer", 2},
}

// Cell markers
type bsMark int

const (
	bsEmpty bsMark = iota
	bsShip         // has a ship segment
	bsMiss         // shot and missed
	bsHit          // shot and hit
	bsSunk         // part of a sunk ship
)

type bsPhase int

const (
	bsPlacing bsPhase = iota
	bsPlayerTurn
	bsAITurn
	bsGameOver
)

type bsShipPlacement struct {
	r, c       int
	horizontal bool
	size       int
	name       string
}

type Battleship struct {
	playerBoard [bsGridSize][bsGridSize]bsMark
	aiBoard     [bsGridSize][bsGridSize]bsMark
	// what the player can see of the AI board (fog of war)
	playerView [bsGridSize][bsGridSize]bsMark

	playerShips []bsShipPlacement
	aiShips     []bsShipPlacement

	phase       bsPhase
	placingIdx  int  // which ship we're placing next
	placeHoriz  bool // current placement orientation
	message     string
	goBack      bool

	// AI state — fair play, no peeking at player board
	aiHits       []bsPos        // unresolved hits (not yet part of a sunk ship)
	aiTried      [bsGridSize][bsGridSize]bool
	aiLastResult bsMark
	aiDelay      int // frames to wait before AI shoots (visual pacing)

	playerSunk int
	aiSunk     int
	ticks      int

	// Audio
	bgm         *audio.Player
	sfxSplash   []byte
	sfxExplosion []byte
	sfxSinking  []byte
	audioInit   bool
}

type bsPos struct{ r, c int }

func newBattleship() *Battleship {
	b := &Battleship{
		phase:   bsPlacing,
		message: fmt.Sprintf("Place your %s (%d)  |  R to rotate  |  Click to place", bsShips[0].name, bsShips[0].size),
	}
	b.initBsAudio()
	return b
}

func (b *Battleship) initBsAudio() {
	// --- BGM: Naval martial theme, minor key (~100 BPM) ---
	// Square wave melody - D minor martial march
	// D3=50, E3=52, F3=53, G3=55, A3=57, Bb3=58, C4=60, D4=62, F4=65
	melody := [][2]int{
		// A section - martial Dm march
		{62, 2}, {62, 1}, {60, 1}, {58, 2}, {57, 2},
		{55, 2}, {57, 1}, {58, 1}, {60, 2}, {57, 2},
		{62, 2}, {65, 2}, {62, 1}, {60, 1}, {58, 2},
		{57, 1}, {55, 1}, {53, 2}, {50, 2}, {0, 2},
		// A' - varied response
		{50, 2}, {53, 2}, {55, 1}, {57, 1}, {58, 2},
		{60, 2}, {62, 1}, {60, 1}, {58, 2}, {57, 2},
		{55, 2}, {57, 2}, {58, 1}, {60, 1}, {62, 2},
		{65, 1}, {62, 1}, {60, 2}, {58, 2}, {50, 2},
		// B section - Gm relative, broader nautical feel
		{55, 2}, {58, 2}, {62, 1}, {65, 1}, {67, 2},
		{65, 1}, {62, 1}, {60, 2}, {58, 2}, {0, 2},
		{67, 2}, {65, 1}, {62, 1}, {60, 2}, {58, 2},
		{55, 2}, {53, 1}, {55, 1}, {58, 2}, {62, 2},
		// B' - climax phrase, trumpet call
		{62, 1}, {65, 1}, {67, 2}, {69, 2}, {70, 2},
		{69, 1}, {67, 1}, {65, 2}, {62, 2}, {0, 2},
		{67, 1}, {69, 1}, {70, 1}, {69, 1}, {67, 2}, {65, 2},
		{62, 1}, {60, 1}, {58, 2}, {57, 4},
		// C section - suspense, Bbmaj modulation
		{58, 2}, {60, 2}, {62, 1}, {0, 1}, {65, 2},
		{67, 1}, {65, 1}, {62, 2}, {60, 2}, {58, 2},
		{53, 2}, {55, 2}, {58, 1}, {60, 1}, {62, 2},
		{0, 2}, {60, 1}, {58, 1}, {55, 2}, {50, 2},
		// Return A - final statement
		{62, 2}, {62, 1}, {60, 1}, {58, 2}, {57, 2},
		{55, 2}, {57, 1}, {58, 1}, {60, 2}, {57, 2},
		{62, 2}, {65, 2}, {62, 1}, {60, 1}, {58, 2},
		{57, 1}, {55, 1}, {53, 2}, {50, 4}, {0, 2},
	}
	melodyPCM := generateTrack(melody, squareWave, 100, 0.30)

	// Bass line - root notes marching
	bass := [][2]int{
		// A section
		{38, 4}, {38, 4}, {43, 4}, {45, 4},
		{38, 4}, {41, 4}, {45, 4}, {38, 4},
		{38, 4}, {41, 4}, {43, 4}, {45, 4},
		{43, 4}, {45, 4}, {41, 4}, {38, 4},
		// B section - Gm bass
		{43, 4}, {46, 4}, {43, 4}, {41, 4},
		{43, 4}, {41, 4}, {38, 4}, {43, 4},
		{43, 4}, {45, 4}, {46, 4}, {45, 4},
		{43, 4}, {41, 4}, {38, 4}, {45, 4},
		// C section - Bb bass
		{46, 4}, {48, 4}, {43, 4}, {41, 4},
		{38, 4}, {41, 4}, {43, 4}, {38, 4},
		// Return
		{38, 4}, {38, 4}, {43, 4}, {45, 4},
		{38, 4}, {41, 4}, {45, 4}, {38, 4}, {38, 4},
	}
	bassPCM := generateTrack(bass, sawWave, 100, 0.18)

	// Military drum pattern - march rhythm
	drums := generateDrumPattern("X.s.h.s.X.s.h.X.X.s.h.s.X.s.hshsX.s.h.s.X.s.hSh.X.s.h.s.XshsXshs", 100, 0.30)

	mixed := mixBuffers(melodyPCM, bassPCM)
	mixed = mixBuffers(mixed, drums)
	b.bgm = loopPlayer(mixed, 0.20)

	// --- SFX: Splash (miss - short noise burst filtered) ---
	splashDur := 0.25
	splashBuf := pcmStereo(splashDur)
	for i := 0; i < len(splashBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / splashDur
		val := noise() * envelope(tFrac, 0.01, 0.1, 0.89) * 0.4
		// Add some low rumble
		val += sineWave(t, 120) * math.Exp(-t*12) * 0.3
		writeSample(splashBuf, i*4, val)
	}
	b.sfxSplash = splashBuf

	// --- SFX: Explosion (hit - low frequency boom) ---
	explDur := 0.4
	explBuf := pcmStereo(explDur)
	for i := 0; i < len(explBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / explDur
		freq := 50.0 * math.Exp(-t*3)
		val := (sineWave(t, freq)*0.5 + noise()*0.5) * envelope(tFrac, 0.005, 0.2, 0.795) * 0.7
		writeSample(explBuf, i*4, val)
	}
	b.sfxExplosion = explBuf

	// --- SFX: Sinking (descending tone) ---
	sinkDur := 0.7
	sinkBuf := pcmStereo(sinkDur)
	for i := 0; i < len(sinkBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / sinkDur
		// Descending from A4 to D2 over duration
		freq := noteFreq(69) * math.Pow(0.25, tFrac)
		val := squareWave(t, freq) * envelope(tFrac, 0.02, 0.5, 0.48) * 0.5
		// Add some noise texture
		val += noise() * math.Exp(-t*6) * 0.2
		writeSample(sinkBuf, i*4, val)
	}
	b.sfxSinking = sinkBuf
}

func (b *Battleship) WantsBack() bool { return b.goBack }

// --- Ship placement ---

func (b *Battleship) canPlace(board *[bsGridSize][bsGridSize]bsMark, r, c, size int, horiz bool) bool {
	for i := 0; i < size; i++ {
		cr, cc := r, c
		if horiz {
			cc += i
		} else {
			cr += i
		}
		if cr < 0 || cr >= bsGridSize || cc < 0 || cc >= bsGridSize {
			return false
		}
		if board[cr][cc] != bsEmpty {
			return false
		}
	}
	return true
}

func (b *Battleship) placeShip(board *[bsGridSize][bsGridSize]bsMark, ships *[]bsShipPlacement, r, c, size int, horiz bool, name string) {
	for i := 0; i < size; i++ {
		cr, cc := r, c
		if horiz {
			cc += i
		} else {
			cr += i
		}
		board[cr][cc] = bsShip
	}
	*ships = append(*ships, bsShipPlacement{r, c, horiz, size, name})
}

func (b *Battleship) placeAIShips() {
	for _, s := range bsShips {
		for {
			horiz := rand.Intn(2) == 0
			r := rand.Intn(bsGridSize)
			c := rand.Intn(bsGridSize)
			if b.canPlace(&b.aiBoard, r, c, s.size, horiz) {
				b.placeShip(&b.aiBoard, &b.aiShips, r, c, s.size, horiz, s.name)
				break
			}
		}
	}
}

// --- Game logic ---

func (b *Battleship) playerShoot(r, c int) {
	if b.playerView[r][c] != bsEmpty {
		return // already shot here
	}
	if b.aiBoard[r][c] == bsShip {
		b.playerView[r][c] = bsHit
		b.aiBoard[r][c] = bsHit
		if b.checkSunk(r, c, &b.aiBoard, &b.aiShips, &b.playerView) {
			b.playerSunk++
			b.message = "You sunk a ship!"
			playSFX(b.sfxSinking, 0.45)
		} else {
			b.message = "Hit!"
			playSFX(b.sfxExplosion, 0.40)
		}
	} else {
		b.playerView[r][c] = bsMiss
		b.message = "Miss."
		playSFX(b.sfxSplash, 0.35)
	}

	if b.playerSunk == len(bsShips) {
		b.phase = bsGameOver
		b.message = "YOU WIN! All enemy ships destroyed! Press R to restart."
		return
	}

	b.phase = bsAITurn
	b.aiDelay = 30 // half-second pause before AI shoots
}

func (b *Battleship) aiShoot() {
	r, c := b.aiPickTarget()
	b.aiTried[r][c] = true

	if b.playerBoard[r][c] == bsShip {
		b.playerBoard[r][c] = bsHit
		b.aiHits = append(b.aiHits, bsPos{r, c})
		if b.checkSunk(r, c, &b.playerBoard, &b.playerShips, nil) {
			b.aiSunk++
			// remove sunk ship hits from aiHits
			b.removeSunkFromHits(&b.playerBoard)
			b.message = "AI sunk your ship!"
			playSFX(b.sfxSinking, 0.45)
		} else {
			b.message = "AI hit your ship!"
			playSFX(b.sfxExplosion, 0.40)
		}
	} else {
		b.playerBoard[r][c] = bsMiss
		b.message = "AI missed."
		playSFX(b.sfxSplash, 0.35)
	}

	if b.aiSunk == len(bsShips) {
		b.phase = bsGameOver
		b.message = "AI WINS. All your ships destroyed! Press R to restart."
		return
	}

	b.phase = bsPlayerTurn
}

// checkSunk checks if the ship at (r,c) is fully hit. If so, marks all cells as sunk.
func (b *Battleship) checkSunk(r, c int, board *[bsGridSize][bsGridSize]bsMark, ships *[]bsShipPlacement, view *[bsGridSize][bsGridSize]bsMark) bool {
	for _, s := range *ships {
		// check if (r,c) belongs to this ship
		belongs := false
		allHit := true
		cells := make([]bsPos, s.size)
		for i := 0; i < s.size; i++ {
			cr, cc := s.r, s.c
			if s.horizontal {
				cc += i
			} else {
				cr += i
			}
			cells[i] = bsPos{cr, cc}
			if cr == r && cc == c {
				belongs = true
			}
			if board[cr][cc] != bsHit {
				allHit = false
			}
		}
		if belongs && allHit {
			for _, p := range cells {
				board[p.r][p.c] = bsSunk
				if view != nil {
					view[p.r][p.c] = bsSunk
				}
			}
			return true
		}
	}
	return false
}

func (b *Battleship) removeSunkFromHits(board *[bsGridSize][bsGridSize]bsMark) {
	filtered := b.aiHits[:0]
	for _, h := range b.aiHits {
		if board[h.r][h.c] == bsHit {
			filtered = append(filtered, h)
		}
	}
	b.aiHits = filtered
}

// --- AI target selection (probability density + hunt/target) ---

func (b *Battleship) aiPickTarget() (int, int) {
	// Target mode: if we have unresolved hits, exploit them
	if len(b.aiHits) > 0 {
		if pos, ok := b.aiTargetMode(); ok {
			return pos.r, pos.c
		}
	}

	// Hunt mode: probability density map
	return b.aiHuntMode()
}

func (b *Battleship) aiTargetMode() (bsPos, bool) {
	// If we have 2+ collinear hits, try extending the line
	if len(b.aiHits) >= 2 {
		if pos, ok := b.aiExtendLine(); ok {
			return pos, true
		}
	}

	// Try adjacent cells of each hit
	dirs := []bsPos{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	// prioritize by probability
	type candidate struct {
		pos  bsPos
		prob float64
	}
	var candidates []candidate

	probs := b.aiCalcProbs()

	for _, h := range b.aiHits {
		for _, d := range dirs {
			nr, nc := h.r+d.r, h.c+d.c
			if nr >= 0 && nr < bsGridSize && nc >= 0 && nc < bsGridSize && !b.aiTried[nr][nc] {
				candidates = append(candidates, candidate{bsPos{nr, nc}, probs[nr][nc]})
			}
		}
	}

	if len(candidates) > 0 {
		// pick highest probability candidate
		best := candidates[0]
		for _, c := range candidates[1:] {
			if c.prob > best.prob {
				best = c
			}
		}
		return best.pos, true
	}

	return bsPos{}, false
}

func (b *Battleship) aiExtendLine() (bsPos, bool) {
	// find direction of hits
	if len(b.aiHits) < 2 {
		return bsPos{}, false
	}

	// check if hits are in a line
	h0 := b.aiHits[0]
	horizontal := true
	vertical := true
	for _, h := range b.aiHits[1:] {
		if h.r != h0.r {
			horizontal = false
		}
		if h.c != h0.c {
			vertical = false
		}
	}

	if !horizontal && !vertical {
		return bsPos{}, false
	}

	if horizontal {
		// find min/max col
		minC, maxC := h0.c, h0.c
		for _, h := range b.aiHits {
			if h.c < minC {
				minC = h.c
			}
			if h.c > maxC {
				maxC = h.c
			}
		}
		r := h0.r
		// try extending right
		if maxC+1 < bsGridSize && !b.aiTried[r][maxC+1] {
			return bsPos{r, maxC + 1}, true
		}
		// try extending left
		if minC-1 >= 0 && !b.aiTried[r][minC-1] {
			return bsPos{r, minC - 1}, true
		}
	} else {
		minR, maxR := h0.r, h0.r
		for _, h := range b.aiHits {
			if h.r < minR {
				minR = h.r
			}
			if h.r > maxR {
				maxR = h.r
			}
		}
		c := h0.c
		if maxR+1 < bsGridSize && !b.aiTried[maxR+1][c] {
			return bsPos{maxR + 1, c}, true
		}
		if minR-1 >= 0 && !b.aiTried[minR-1][c] {
			return bsPos{minR - 1, c}, true
		}
	}

	return bsPos{}, false
}

func (b *Battleship) aiHuntMode() (int, int) {
	probs := b.aiCalcProbs()

	bestProb := -1.0
	var bestCells []bsPos

	for r := 0; r < bsGridSize; r++ {
		for c := 0; c < bsGridSize; c++ {
			if b.aiTried[r][c] {
				continue
			}
			if probs[r][c] > bestProb {
				bestProb = probs[r][c]
				bestCells = []bsPos{{r, c}}
			} else if probs[r][c] == bestProb {
				bestCells = append(bestCells, bsPos{r, c})
			}
		}
	}

	if len(bestCells) > 0 {
		pick := bestCells[rand.Intn(len(bestCells))]
		return pick.r, pick.c
	}

	// fallback: random untried cell
	for {
		r := rand.Intn(bsGridSize)
		c := rand.Intn(bsGridSize)
		if !b.aiTried[r][c] {
			return r, c
		}
	}
}

// aiCalcProbs builds a probability density map based on remaining ship sizes.
// For each unsunk ship, try every valid placement and count how many times each cell is covered.
func (b *Battleship) aiCalcProbs() [bsGridSize][bsGridSize]float64 {
	var probs [bsGridSize][bsGridSize]float64

	// figure out which ship sizes remain (unsunk)
	sunkSizes := make(map[int]int)
	for _, s := range b.playerShips {
		allSunk := true
		for i := 0; i < s.size; i++ {
			cr, cc := s.r, s.c
			if s.horizontal {
				cc += i
			} else {
				cr += i
			}
			if b.playerBoard[cr][cc] != bsSunk {
				allSunk = false
				break
			}
		}
		if allSunk {
			sunkSizes[s.size]++
		}
	}

	remaining := []int{}
	shipSizeCounts := make(map[int]int)
	for _, s := range bsShips {
		shipSizeCounts[s.size]++
	}
	for _, s := range bsShips {
		needed := shipSizeCounts[s.size] - sunkSizes[s.size]
		if needed > 0 {
			remaining = append(remaining, s.size)
			sunkSizes[s.size]++ // avoid double-counting
		}
	}

	for _, size := range remaining {
		for horiz := 0; horiz <= 1; horiz++ {
			for r := 0; r < bsGridSize; r++ {
				for c := 0; c < bsGridSize; c++ {
					valid := true
					hasHit := false
					cells := make([]bsPos, size)
					for i := 0; i < size; i++ {
						cr, cc := r, c
						if horiz == 1 {
							cc += i
						} else {
							cr += i
						}
						if cr >= bsGridSize || cc >= bsGridSize {
							valid = false
							break
						}
						mark := b.playerBoard[cr][cc]
						if mark == bsMiss || mark == bsSunk {
							valid = false
							break
						}
						if mark == bsHit {
							hasHit = true
						}
						cells[i] = bsPos{cr, cc}
					}
					if !valid {
						continue
					}
					// weight: placements that overlap existing hits are much more likely
					weight := 1.0
					if hasHit {
						weight = 20.0
					}
					for _, p := range cells {
						if !b.aiTried[p.r][p.c] {
							probs[p.r][p.c] += weight
						}
					}
				}
			}
		}
	}

	return probs
}

// --- Update ---

func (b *Battleship) Update() error {
	// Start BGM on first frame
	if !b.audioInit {
		b.audioInit = true
		if b.bgm != nil && !b.bgm.IsPlaying() {
			b.bgm.Play()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if b.bgm != nil && b.bgm.IsPlaying() {
			b.bgm.Pause()
		}
		b.goBack = true
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) && b.phase == bsGameOver {
		oldBgm := b.bgm
		*b = *newBattleship()
		if oldBgm != nil {
			oldBgm.Pause()
		}
		b.audioInit = false
		return nil
	}

	b.ticks++

	switch b.phase {
	case bsPlacing:
		b.updatePlacing()
	case bsPlayerTurn:
		b.updatePlayerTurn()
	case bsAITurn:
		b.aiDelay--
		if b.aiDelay <= 0 {
			b.aiShoot()
		}
	}

	return nil
}

func (b *Battleship) updatePlacing() {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		b.placeHoriz = !b.placeHoriz
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		r, c, ok := b.screenToPlayerGrid(mx, my)
		if !ok {
			return
		}
		ship := bsShips[b.placingIdx]
		if b.canPlace(&b.playerBoard, r, c, ship.size, b.placeHoriz) {
			b.placeShip(&b.playerBoard, &b.playerShips, r, c, ship.size, b.placeHoriz, ship.name)
			b.placingIdx++
			if b.placingIdx >= len(bsShips) {
				// done placing, set up AI
				b.placeAIShips()
				b.phase = bsPlayerTurn
				b.message = "Your turn! Click the right grid to fire."
			} else {
				next := bsShips[b.placingIdx]
				b.message = fmt.Sprintf("Place your %s (%d)  |  R to rotate  |  Click to place", next.name, next.size)
			}
		}
	}
}

func (b *Battleship) updatePlayerTurn() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		r, c, ok := b.screenToAIGrid(mx, my)
		if !ok {
			return
		}
		if b.playerView[r][c] != bsEmpty {
			return // already shot
		}
		b.playerShoot(r, c)
	}
}

// --- Coordinate conversion ---

func (b *Battleship) playerGridOrigin() (int, int) {
	return bsMarginX + bsLabelPad, bsMarginY + bsLabelPad
}

func (b *Battleship) aiGridOrigin() (int, int) {
	ox := bsMarginX + bsLabelPad + bsGridSize*bsCellSize + bsGap + bsLabelPad
	return ox, bsMarginY + bsLabelPad
}

func (b *Battleship) screenToPlayerGrid(mx, my int) (int, int, bool) {
	ox, oy := b.playerGridOrigin()
	c := (mx - ox) / bsCellSize
	r := (my - oy) / bsCellSize
	if r >= 0 && r < bsGridSize && c >= 0 && c < bsGridSize && mx >= ox && my >= oy {
		return r, c, true
	}
	return 0, 0, false
}

func (b *Battleship) screenToAIGrid(mx, my int) (int, int, bool) {
	ox, oy := b.aiGridOrigin()
	c := (mx - ox) / bsCellSize
	r := (my - oy) / bsCellSize
	if r >= 0 && r < bsGridSize && c >= 0 && c < bsGridSize && mx >= ox && my >= oy {
		return r, c, true
	}
	return 0, 0, false
}

// --- Drawing ---

func (b *Battleship) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x18, 0x28, 0x48, 0xff})

	// header
	ebitenutil.DebugPrintAt(screen, "BATTLESHIP", bsScreenW/2-30, 8)
	ebitenutil.DebugPrintAt(screen, b.message, 10, 28)

	// score
	scoreStr := fmt.Sprintf("You sunk: %d/%d    AI sunk: %d/%d    ESC=menu  R=restart",
		b.playerSunk, len(bsShips), b.aiSunk, len(bsShips))
	ebitenutil.DebugPrintAt(screen, scoreStr, 10, 46)

	// labels
	pox, poy := b.playerGridOrigin()
	aox, aoy := b.aiGridOrigin()
	ebitenutil.DebugPrintAt(screen, "YOUR FLEET", pox+bsGridSize*bsCellSize/2-30, poy-bsLabelPad-4)
	ebitenutil.DebugPrintAt(screen, "ENEMY WATERS", aox+bsGridSize*bsCellSize/2-36, aoy-bsLabelPad-4)

	b.drawGrid(screen, pox, poy, &b.playerBoard, true)
	b.drawGrid(screen, aox, aoy, &b.playerView, false)

	// during placement, show ghost preview
	if b.phase == bsPlacing && b.placingIdx < len(bsShips) {
		b.drawPlacementGhost(screen)
	}
}

func (b *Battleship) drawGrid(screen *ebiten.Image, ox, oy int, board *[bsGridSize][bsGridSize]bsMark, showShips bool) {
	// column letters
	for c := 0; c < bsGridSize; c++ {
		ebitenutil.DebugPrintAt(screen, string(rune('A'+c)), ox+c*bsCellSize+bsCellSize/2-3, oy-14)
	}
	// row numbers
	for r := 0; r < bsGridSize; r++ {
		label := fmt.Sprintf("%d", r+1)
		if r+1 < 10 {
			label = " " + label
		}
		ebitenutil.DebugPrintAt(screen, label, ox-16, oy+r*bsCellSize+bsCellSize/2-8)
	}

	mx, my := ebiten.CursorPosition()

	for r := 0; r < bsGridSize; r++ {
		for c := 0; c < bsGridSize; c++ {
			x := float32(ox + c*bsCellSize)
			y := float32(oy + r*bsCellSize)
			w := float32(bsCellSize)

			mark := board[r][c]

			// cell background with wave animation on water cells
			bg := color.RGBA{0x20, 0x50, 0x80, 0xff} // ocean blue
			// Animated wave oscillation for empty water cells
			if mark == bsEmpty || (mark == bsShip && !showShips) {
				waveVal := math.Sin(float64(b.ticks)*0.06 + float64(r)*0.8 + float64(c)*0.5)
				waveShift := int(waveVal * 12)
				bgB := int(bg.B) + waveShift
				bgG := int(bg.G) + waveShift/2
				if bgB > 255 {
					bgB = 255
				}
				if bgB < 0 {
					bgB = 0
				}
				if bgG > 255 {
					bgG = 255
				}
				if bgG < 0 {
					bgG = 0
				}
				bg = color.RGBA{bg.R, uint8(bgG), uint8(bgB), 0xff}
			}
			switch mark {
			case bsShip:
				if showShips {
					bg = color.RGBA{0x60, 0x60, 0x60, 0xff} // gray ship
				}
			case bsMiss:
				bg = color.RGBA{0x30, 0x40, 0x60, 0xff} // darker
			case bsHit:
				bg = color.RGBA{0xc0, 0x40, 0x20, 0xff} // red
			case bsSunk:
				// Sinking animation: darken further with pulsing
				sinkPulse := math.Sin(float64(b.ticks)*0.08+float64(r+c)*0.3)*0.15 + 0.85
				bg = color.RGBA{uint8(float64(0x80) * sinkPulse), uint8(float64(0x20) * sinkPulse), uint8(float64(0x10) * sinkPulse), 0xff}
			}

			vector.FillRect(screen, x, y, w, w, bg, false)
			vector.StrokeRect(screen, x, y, w, w, 1, color.RGBA{0x40, 0x70, 0xa0, 0xff}, false)

			// markers
			cx := x + w/2
			cy := y + w/2
			switch mark {
			case bsMiss:
				// Ripple circles on misses (expanding concentric rings)
				vector.FillCircle(screen, cx, cy, 4, color.RGBA{0x80, 0x90, 0xa0, 0xff}, true)
				ripple1 := float32(math.Sin(float64(b.ticks)*0.1+float64(r+c))*2 + 8)
				ripple2 := float32(math.Sin(float64(b.ticks)*0.1+float64(r+c)+1.5)*2 + 12)
				vector.StrokeCircle(screen, cx, cy, ripple1, 0.8, color.RGBA{0x80, 0xb0, 0xd0, 0x50}, false)
				vector.StrokeCircle(screen, cx, cy, ripple2, 0.5, color.RGBA{0x80, 0xb0, 0xd0, 0x30}, false)
			case bsHit:
				// Explosion particles on hits
				vector.StrokeLine(screen, x+6, y+6, x+w-6, y+w-6, 2, color.White, true)
				vector.StrokeLine(screen, x+w-6, y+6, x+6, y+w-6, 2, color.White, true)
				// Animated spark particles
				for sp := 0; sp < 4; sp++ {
					angle := float64(sp)*math.Pi/2 + float64(b.ticks)*0.15
					sparkDist := float32(math.Sin(float64(b.ticks)*0.2+float64(sp))*3 + 6)
					spx := cx + float32(math.Cos(angle))*sparkDist
					spy := cy + float32(math.Sin(angle))*sparkDist
					vector.FillCircle(screen, spx, spy, 1.5, color.RGBA{0xff, 0xcc, 0x44, 0xbb}, false)
				}
			case bsSunk:
				// filled X with slight pulsing glow
				vector.StrokeLine(screen, x+4, y+4, x+w-4, y+w-4, 3, color.RGBA{0xff, 0x80, 0x80, 0xff}, true)
				vector.StrokeLine(screen, x+w-4, y+4, x+4, y+w-4, 3, color.RGBA{0xff, 0x80, 0x80, 0xff}, true)
			}

			// hover highlight on enemy grid when it's player's turn
			if !showShips && b.phase == bsPlayerTurn && mark == bsEmpty {
				if mx >= int(x) && mx < int(x+w) && my >= int(y) && my < int(y+w) {
					vector.FillRect(screen, x, y, w, w, color.RGBA{0xff, 0xff, 0xff, 0x30}, false)
				}
			}
		}
	}
}

func (b *Battleship) drawPlacementGhost(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	r, c, ok := b.screenToPlayerGrid(mx, my)
	if !ok {
		return
	}
	ship := bsShips[b.placingIdx]
	valid := b.canPlace(&b.playerBoard, r, c, ship.size, b.placeHoriz)
	ox, oy := b.playerGridOrigin()

	ghostColor := color.RGBA{0x40, 0xff, 0x40, 0x60}
	if !valid {
		ghostColor = color.RGBA{0xff, 0x40, 0x40, 0x60}
	}

	for i := 0; i < ship.size; i++ {
		cr, cc := r, c
		if b.placeHoriz {
			cc += i
		} else {
			cr += i
		}
		if cr >= bsGridSize || cc >= bsGridSize {
			break
		}
		x := float32(ox + cc*bsCellSize)
		y := float32(oy + cr*bsCellSize)
		vector.FillRect(screen, x, y, bsCellSize, bsCellSize, ghostColor, false)
	}
}

func (b *Battleship) Layout(w, h int) (int, int) {
	return bsScreenW, bsScreenH
}
