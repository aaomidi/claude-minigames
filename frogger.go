package main

import (
	"fmt"
	"image/color"
	"math"

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
	fgCellSize = 32
	fgCols     = 15
	fgRows     = 15
	fgScreenW  = fgCols * fgCellSize // 480
	fgScreenH  = fgRows * fgCellSize // 480
)

// Row types describe what each row of the map is.
const (
	fgRowGoal   = 0 // goal row (top)
	fgRowSafe   = 1 // safe zone
	fgRowWater  = 2 // water with logs
	fgRowRoad   = 3 // road with cars
	fgRowStart  = 4 // starting safe zone (bottom)
	fgRowMiddle = 5 // mid-field safe zone
)

// Timings
const (
	fgMoveCooldown  = 6        // ticks between moves
	fgDeathPause    = 40       // ticks to show death before respawn
	fgWinPause      = 50       // ticks to show win before next level
	fgBonusTimerMax = 60 * 30  // 30 seconds at 60 FPS for time bonus
	fgGoalSlots     = 5        // number of goal positions across the top
)

// Game states
const (
	fgStatePlaying = 0
	fgStateDying   = 1
	fgStateWin     = 2
	fgStateOver    = 3
)

// ---------------------------------------------------------------------------
// Lane configuration
// ---------------------------------------------------------------------------

// fgLaneConf describes a lane (row) in the game world.
type fgLaneConf struct {
	rowType  int     // fgRowGoal, fgRowSafe, etc.
	speed    float64 // positive = right, negative = left
	objLen   int     // length of vehicle/log in cells
	gap      int     // gap between objects in cells
	colorIdx int     // index into a palette for the object color
}

// fgBaseLanes defines rows 0..14 from top to bottom.
//
//	Row 0  = goal
//	Row 1  = safe (water-side bank)
//	Rows 2-6 = water/log lanes
//	Row 7  = safe (mid)
//	Rows 8-12 = road/car lanes
//	Row 13 = safe (shoulder)
//	Row 14 = start
var fgBaseLanes = [fgRows]fgLaneConf{
	{fgRowGoal, 0, 0, 0, 0},     // 0  goal
	{fgRowSafe, 0, 0, 0, 0},     // 1  safe bank
	{fgRowWater, -1.0, 4, 4, 0}, // 2  long logs left
	{fgRowWater, 0.8, 3, 5, 1},  // 3  medium logs right
	{fgRowWater, -1.2, 5, 3, 2}, // 4  long logs left
	{fgRowWater, 0.6, 3, 6, 0},  // 5  short logs right
	{fgRowWater, -0.9, 4, 4, 1}, // 6  medium logs left
	{fgRowMiddle, 0, 0, 0, 0},   // 7  safe middle
	{fgRowRoad, 1.0, 2, 5, 0},   // 8  cars right
	{fgRowRoad, -1.4, 1, 4, 1},  // 9  cars left
	{fgRowRoad, 0.7, 3, 4, 2},   // 10 trucks right
	{fgRowRoad, -1.1, 1, 5, 3},  // 11 cars left
	{fgRowRoad, 0.9, 2, 4, 0},   // 12 cars right
	{fgRowSafe, 0, 0, 0, 0},     // 13 safe shoulder
	{fgRowStart, 0, 0, 0, 0},    // 14 start
}

// ---------------------------------------------------------------------------
// Colors
// ---------------------------------------------------------------------------

var (
	fgColorGrass    = color.RGBA{0x3a, 0x7d, 0x28, 0xff}
	fgColorGoal     = color.RGBA{0x2e, 0x8b, 0x2e, 0xff}
	fgColorRoad     = color.RGBA{0x55, 0x55, 0x60, 0xff}
	fgColorRoadLine = color.RGBA{0xcc, 0xcc, 0x44, 0x80}
	fgColorWater    = color.RGBA{0x1e, 0x70, 0xc0, 0xff}
	fgColorSidewalk = color.RGBA{0x88, 0x88, 0x78, 0xff}
	fgColorFrog     = color.RGBA{0x30, 0xd0, 0x30, 0xff}
	fgColorFrogEye  = color.RGBA{0xff, 0xff, 0xff, 0xff}
	fgColorFrogPup  = color.RGBA{0x10, 0x10, 0x10, 0xff}
	fgColorDead     = color.RGBA{0xd0, 0x30, 0x30, 0xff}
	fgColorHUD      = color.RGBA{0x20, 0x20, 0x20, 0xcc}
)

var fgCarColors = []color.RGBA{
	{0xd0, 0x30, 0x30, 0xff}, // red
	{0xe0, 0xb0, 0x20, 0xff}, // yellow
	{0x30, 0x60, 0xd0, 0xff}, // blue
	{0xd0, 0x70, 0x20, 0xff}, // orange
}

var fgLogColors = []color.RGBA{
	{0x8b, 0x5a, 0x2b, 0xff}, // brown
	{0x6e, 0x4b, 0x20, 0xff}, // dark brown
	{0xa0, 0x6e, 0x3a, 0xff}, // light brown
}

// ---------------------------------------------------------------------------
// Obstacle
// ---------------------------------------------------------------------------

type fgObstacle struct {
	x     float64 // pixel x position (left edge)
	row   int     // which row
	len   int     // length in cells
	speed float64 // pixels per tick
	clr   color.RGBA
}

// ---------------------------------------------------------------------------
// Frogger struct
// ---------------------------------------------------------------------------

type Frogger struct {
	goBack bool

	// Player position in grid coords
	frogCol int
	frogRow int

	// Movement cooldown
	moveTick int

	// Game state
	state     int
	stateTick int
	frameTick int // animation tick, incremented every frame

	// Lives & score
	lives int
	score int
	level int

	// Time bonus tracking
	bonusTimer int

	// Goals reached (5 goal slots across the top)
	goalSlots [fgGoalSlots]bool
	goalCount int

	// Obstacles (cars on road, logs on water)
	obstacles []fgObstacle

	// Lane configurations (adjusted per level for difficulty)
	lanes [fgRows]fgLaneConf

	// Best (highest = smallest row number) row reached this life (for progressive scoring)
	bestRow int

	// Audio
	bgm       *audio.Player
	sfxHop    []byte
	sfxSplat  []byte
	sfxDing   []byte
	audioInit bool
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func newFrogger() *Frogger {
	f := &Frogger{
		lives: 3,
		level: 1,
	}
	f.setupLevel()
	return f
}

// ---------------------------------------------------------------------------
// Level setup
// ---------------------------------------------------------------------------

func (f *Frogger) setupLevel() {
	// Copy base lanes and scale speed by difficulty
	speedMul := 1.0 + float64(f.level-1)*0.15
	for i := 0; i < fgRows; i++ {
		f.lanes[i] = fgBaseLanes[i]
		f.lanes[i].speed *= speedMul
	}

	// Generate obstacles for each lane
	f.obstacles = f.obstacles[:0]
	for row := 0; row < fgRows; row++ {
		lane := f.lanes[row]
		if lane.rowType != fgRowRoad && lane.rowType != fgRowWater {
			continue
		}
		f.fgSpawnLane(row, lane)
	}

	f.fgResetFrog()
}

func (f *Frogger) fgSpawnLane(row int, lane fgLaneConf) {
	stride := lane.objLen + lane.gap // cells per repeating unit
	if stride <= 0 {
		return
	}
	numObjs := (fgCols / stride) + 3 // enough to wrap seamlessly
	for i := 0; i < numObjs; i++ {
		ox := float64(i*stride*fgCellSize) - float64(fgCellSize)
		var clr color.RGBA
		if lane.rowType == fgRowRoad {
			clr = fgCarColors[lane.colorIdx%len(fgCarColors)]
		} else {
			clr = fgLogColors[lane.colorIdx%len(fgLogColors)]
		}
		f.obstacles = append(f.obstacles, fgObstacle{
			x:     ox,
			row:   row,
			len:   lane.objLen,
			speed: lane.speed,
			clr:   clr,
		})
	}
}

func (f *Frogger) fgResetFrog() {
	f.frogCol = fgCols / 2
	f.frogRow = fgRows - 1
	f.bestRow = fgRows - 1
	f.bonusTimer = fgBonusTimerMax
	f.state = fgStatePlaying
	f.stateTick = 0
	f.moveTick = 0
}

// ---------------------------------------------------------------------------
// Audio initialization
// ---------------------------------------------------------------------------

func (f *Frogger) fgInitAudio() {
	if f.audioInit {
		return
	}
	f.audioInit = true

	// --- BGM: Quirky hopping melody, ~120bpm ---
	// Triangle wave melody - playful C major staccato hopping feel
	melody := [][2]int{
		// A section - playful hopping
		{72, 1}, {74, 1}, {76, 1}, {79, 1},
		{77, 1}, {76, 1}, {74, 2},
		{72, 1}, {76, 1}, {79, 1}, {84, 1},
		{81, 1}, {79, 1}, {76, 2},
		{67, 1}, {72, 1}, {74, 1}, {76, 1},
		{79, 2}, {76, 1}, {74, 1},
		{72, 1}, {71, 1}, {72, 2},
		{0, 2},
		// A' - chirpy variation
		{79, 1}, {76, 1}, {74, 1}, {72, 1},
		{74, 1}, {76, 1}, {79, 2},
		{84, 1}, {81, 1}, {79, 1}, {76, 1},
		{74, 2}, {72, 2},
		{76, 1}, {79, 1}, {81, 1}, {84, 1},
		{86, 2}, {84, 1}, {81, 1},
		{79, 1}, {76, 1}, {74, 2},
		{72, 2}, {0, 2},
		// B section - Dm feel, danger zone
		{74, 1}, {77, 1}, {81, 1}, {77, 1},
		{74, 2}, {72, 1}, {0, 1},
		{69, 1}, {72, 1}, {74, 1}, {77, 1},
		{79, 2}, {77, 2},
		{74, 1}, {72, 1}, {69, 1}, {67, 1},
		{65, 2}, {67, 1}, {69, 1},
		{72, 1}, {74, 1}, {72, 2},
		{0, 2},
		// C section - triumphant crossing
		{72, 1}, {76, 1}, {79, 1}, {84, 1},
		{86, 2}, {84, 1}, {81, 1},
		{79, 1}, {84, 1}, {86, 1}, {88, 1},
		{86, 2}, {84, 2},
		{81, 1}, {79, 1}, {76, 1}, {74, 1},
		{72, 2}, {0, 1}, {71, 1},
		{72, 1}, {74, 1}, {76, 1}, {79, 1},
		{84, 2}, {0, 2},
	}
	melodyPCM := generateTrack(melody, triangleWave, 120, 0.35)

	// Square wave bass line - bouncy accompaniment
	bass := [][2]int{
		// A section
		{48, 2}, {48, 1}, {0, 1}, {53, 2}, {53, 1}, {0, 1},
		{48, 2}, {48, 1}, {0, 1}, {55, 2}, {53, 1}, {0, 1},
		{43, 2}, {48, 1}, {0, 1}, {55, 2}, {0, 1}, {53, 1},
		{48, 2}, {47, 1}, {48, 1}, {0, 2},
		// A' bass
		{55, 2}, {48, 1}, {0, 1}, {53, 2}, {55, 1}, {0, 1},
		{48, 2}, {52, 1}, {0, 1}, {55, 2}, {53, 1}, {0, 1},
		{48, 2}, {55, 1}, {0, 1}, {53, 2}, {48, 1}, {0, 1},
		{48, 2}, {47, 1}, {48, 1}, {0, 2},
		// B section - Dm bass
		{50, 2}, {50, 1}, {0, 1}, {45, 2}, {50, 1}, {0, 1},
		{50, 2}, {48, 1}, {0, 1}, {53, 2}, {50, 1}, {0, 1},
		{50, 2}, {45, 1}, {0, 1}, {43, 2}, {0, 1}, {45, 1},
		{48, 2}, {47, 1}, {48, 1}, {0, 2},
		// C section bass
		{48, 2}, {48, 1}, {0, 1}, {55, 2}, {55, 1}, {0, 1},
		{48, 2}, {52, 1}, {0, 1}, {55, 2}, {53, 1}, {0, 1},
		{48, 2}, {43, 1}, {0, 1}, {48, 2}, {0, 1}, {47, 1},
		{48, 2}, {48, 1}, {48, 1}, {0, 2},
	}
	bassPCM := generateTrack(bass, squareWave, 120, 0.20)

	// Drum pattern - hopping rhythm
	drums := generateDrumPattern("K.h.K.h.K.h.K.h.Kxh.Kxh.K.hxK.h.K.h.K.hSK.h.KshSKxh.K.h.K.hxKshs", 120, 0.30)

	bgmPCM := mixBuffers(melodyPCM, bassPCM)
	if len(drums) <= len(bgmPCM) {
		bgmPCM = mixBuffers(bgmPCM, drums)
	}
	f.bgm = loopPlayer(bgmPCM, 0.20)

	// --- SFX: Hop (short chirpy blip) ---
	hopBuf := pcmStereo(0.06)
	for i := 0; i < len(hopBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(len(hopBuf)/4)
		freq := 800.0 + 400.0*tFrac // rising pitch
		val := triangleWave(t, freq) * (1.0 - tFrac) * 0.5
		writeSample(hopBuf, i*4, val)
	}
	f.sfxHop = hopBuf

	// --- SFX: Splat (noise burst with low rumble) ---
	splatBuf := pcmStereo(0.25)
	for i := 0; i < len(splatBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(len(splatBuf)/4)
		n := noise() * math.Exp(-t*12) * 0.7
		low := sineWave(t, 80.0*math.Exp(-t*8)) * math.Exp(-t*6) * 0.5
		val := (n + low) * (1.0 - tFrac*0.5)
		writeSample(splatBuf, i*4, val)
	}
	f.sfxSplat = splatBuf

	// --- SFX: Ding (bright success tone) ---
	dingBuf := pcmStereo(0.3)
	for i := 0; i < len(dingBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := float64(i) / float64(len(dingBuf)/4)
		env := envelope(tFrac, 0.01, 0.3, 0.69)
		val := (sineWave(t, noteFreq(84))*0.5 + sineWave(t, noteFreq(88))*0.3 + sineWave(t, noteFreq(91))*0.2) * env * 0.5
		writeSample(dingBuf, i*4, val)
	}
	f.sfxDing = dingBuf

	f.bgm.Play()
}

// ---------------------------------------------------------------------------
// SubGame interface
// ---------------------------------------------------------------------------

func (f *Frogger) WantsBack() bool {
	return f.goBack
}

func (f *Frogger) Layout(_, _ int) (int, int) {
	return fgScreenW, fgScreenH
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (f *Frogger) Update() error {
	f.fgInitAudio()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if f.bgm != nil {
			f.bgm.Pause()
		}
		f.goBack = true
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		bgm := f.bgm
		sfxHop := f.sfxHop
		sfxSplat := f.sfxSplat
		sfxDing := f.sfxDing
		goBack := f.goBack
		*f = *newFrogger()
		f.goBack = goBack
		f.bgm = bgm
		f.sfxHop = sfxHop
		f.sfxSplat = sfxSplat
		f.sfxDing = sfxDing
		f.audioInit = true
		return nil
	}

	f.frameTick++

	switch f.state {
	case fgStatePlaying:
		f.fgUpdatePlaying()
	case fgStateDying:
		f.stateTick++
		if f.stateTick >= fgDeathPause {
			if f.lives <= 0 {
				f.state = fgStateOver
				f.stateTick = 0
			} else {
				f.fgResetFrog()
			}
		}
	case fgStateWin:
		f.stateTick++
		if f.stateTick >= fgWinPause {
			f.level++
			f.goalSlots = [fgGoalSlots]bool{}
			f.goalCount = 0
			f.setupLevel()
		}
	case fgStateOver:
		// Wait for R (handled above)
	}

	return nil
}

func (f *Frogger) fgUpdatePlaying() {
	// Move all obstacles
	f.fgMoveObstacles()

	// Tick down bonus timer
	if f.bonusTimer > 0 {
		f.bonusTimer--
	}

	// Tick down move cooldown
	if f.moveTick > 0 {
		f.moveTick--
	}

	// Handle arrow key input (discrete grid movement)
	if f.moveTick == 0 {
		moved := false
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
			if f.frogRow > 0 {
				f.frogRow--
				moved = true
			}
		} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
			if f.frogRow < fgRows-1 {
				f.frogRow++
				moved = true
			}
		} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
			if f.frogCol > 0 {
				f.frogCol--
				moved = true
			}
		} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
			if f.frogCol < fgCols-1 {
				f.frogCol++
				moved = true
			}
		}
		if moved {
			f.moveTick = fgMoveCooldown
			if f.sfxHop != nil {
				playSFX(f.sfxHop, 0.35)
			}
			// Award points for forward progress
			if f.frogRow < f.bestRow {
				f.score += 10
				f.bestRow = f.frogRow
			}
		}
	}

	// Check if frog reached the goal row
	if f.frogRow == 0 {
		f.fgHandleGoal()
		return
	}

	// Collision / water check
	f.fgCheckCollisions()
}

func (f *Frogger) fgMoveObstacles() {
	wrapW := float64(fgScreenW)
	for i := range f.obstacles {
		o := &f.obstacles[i]
		o.x += o.speed
		objW := float64(o.len * fgCellSize)
		// Wrap: if moving right and fully off the right edge, teleport to left
		if o.speed > 0 && o.x > wrapW+float64(fgCellSize) {
			o.x = -objW - float64(fgCellSize)
		} else if o.speed < 0 && o.x+objW < -float64(fgCellSize) {
			o.x = wrapW + float64(fgCellSize)
		}
	}
}

func (f *Frogger) fgHandleGoal() {
	// Determine which of the 5 goal slots the frog landed in
	slotW := fgCols / fgGoalSlots
	slot := f.frogCol / slotW
	if slot < 0 {
		slot = 0
	}
	if slot >= fgGoalSlots {
		slot = fgGoalSlots - 1
	}

	if f.goalSlots[slot] {
		// Slot already filled -- death
		f.fgDie()
		return
	}

	f.goalSlots[slot] = true
	f.goalCount++
	if f.sfxDing != nil {
		playSFX(f.sfxDing, 0.45)
	}

	// Points for reaching goal + time bonus
	f.score += 50
	f.score += f.bonusTimer / 60

	if f.goalCount >= fgGoalSlots {
		// All slots filled -- level complete!
		f.score += 1000
		f.state = fgStateWin
		f.stateTick = 0
	} else {
		f.fgResetFrog()
	}
}

func (f *Frogger) fgCheckCollisions() {
	lane := f.lanes[f.frogRow]
	frogPx := float64(f.frogCol * fgCellSize)

	// Use a slightly inset hitbox for the frog
	frogLeft := frogPx + 2
	frogRight := frogPx + float64(fgCellSize) - 2

	switch lane.rowType {
	case fgRowRoad:
		// Colliding with any car/truck = death
		for _, o := range f.obstacles {
			if o.row != f.frogRow {
				continue
			}
			objLeft := o.x
			objRight := o.x + float64(o.len*fgCellSize)
			if frogRight > objLeft && frogLeft < objRight {
				f.fgDie()
				return
			}
		}

	case fgRowWater:
		// Frog must be standing on a log; otherwise drowns
		onLog := false
		for _, o := range f.obstacles {
			if o.row != f.frogRow {
				continue
			}
			objLeft := o.x + 4
			objRight := o.x + float64(o.len*fgCellSize) - 4
			if frogRight > objLeft && frogLeft < objRight {
				onLog = true
				// Ride with the log
				newPx := frogPx + o.speed
				f.frogCol = int(math.Round(newPx / float64(fgCellSize)))
				if f.frogCol < 0 || f.frogCol >= fgCols {
					f.fgDie()
					return
				}
				break
			}
		}
		if !onLog {
			f.fgDie()
			return
		}
	}
}

func (f *Frogger) fgDie() {
	f.lives--
	f.state = fgStateDying
	f.stateTick = 0
	if f.sfxSplat != nil {
		playSFX(f.sfxSplat, 0.45)
	}
}

// ---------------------------------------------------------------------------
// Draw
// ---------------------------------------------------------------------------

func (f *Frogger) Draw(screen *ebiten.Image) {
	f.fgDrawBackground(screen)
	f.fgDrawGoalSlots(screen)
	f.fgDrawObstacles(screen)
	if f.state != fgStateOver {
		f.fgDrawFrog(screen)
	}
	f.fgDrawHUD(screen)
	f.fgDrawOverlay(screen)
}

// fgDrawBackground paints each row according to its lane type and adds details.
func (f *Frogger) fgDrawBackground(screen *ebiten.Image) {
	for row := 0; row < fgRows; row++ {
		ry := float32(row * fgCellSize)
		lane := f.lanes[row]
		var bg color.RGBA
		switch lane.rowType {
		case fgRowGoal:
			bg = fgColorGoal
		case fgRowSafe, fgRowStart:
			bg = fgColorGrass
		case fgRowMiddle:
			bg = fgColorSidewalk
		case fgRowWater:
			bg = fgColorWater
		case fgRowRoad:
			bg = fgColorRoad
		default:
			bg = fgColorGrass
		}
		vector.FillRect(screen, 0, ry, float32(fgScreenW), float32(fgCellSize), bg, false)

		// Dashed centre line on road lanes
		if lane.rowType == fgRowRoad {
			for c := 0; c < fgCols; c++ {
				dx := float32(c*fgCellSize + fgCellSize/4)
				dy := ry + float32(fgCellSize)/2 - 1
				vector.FillRect(screen, dx, dy, float32(fgCellSize/2), 2, fgColorRoadLine, false)
			}
		}

		// Animated wave shimmer on water lanes
		if lane.rowType == fgRowWater {
			for c := 0; c < fgCols; c++ {
				waveOff := math.Sin(float64(f.frameTick)*0.08 + float64(c)*0.7 + float64(row)*1.3)
				shimmerAlpha := uint8(40 + int(waveOff*30))
				wx := float32(c * fgCellSize)
				// Animated wave line that moves vertically
				waveY := ry + float32(fgCellSize)/2 + float32(waveOff*3)
				vector.StrokeLine(screen, wx+2, waveY, wx+float32(fgCellSize)-2, waveY, 1, color.RGBA{0x60, 0xb0, 0xee, shimmerAlpha}, false)
				// Second wave line offset
				waveY2 := ry + float32(fgCellSize)*0.75 + float32(math.Sin(float64(f.frameTick)*0.08+float64(c)*0.5+1.5)*2)
				vector.StrokeLine(screen, wx+4, waveY2, wx+float32(fgCellSize)-4, waveY2, 0.8, color.RGBA{0x80, 0xc0, 0xf0, uint8(30 + int(waveOff*20))}, false)
			}
		}
	}
}

func (f *Frogger) fgDrawGoalSlots(screen *ebiten.Image) {
	slotPx := fgScreenW / fgGoalSlots
	for i := 0; i < fgGoalSlots; i++ {
		sx := float32(i*slotPx) + float32(slotPx)/2 - float32(fgCellSize)/2
		sy := float32(2)
		inset := float32(4)

		if f.goalSlots[i] {
			// Occupied: draw small frog icon
			vector.FillRect(screen, sx+inset, sy+inset, float32(fgCellSize)-inset*2, float32(fgCellSize)-inset*2, fgColorFrog, false)
			vector.StrokeRect(screen, sx+inset, sy+inset, float32(fgCellSize)-inset*2, float32(fgCellSize)-inset*2, 1, color.RGBA{0x20, 0x80, 0x20, 0xff}, false)
		} else {
			// Empty slot outline
			vector.StrokeRect(screen, sx+inset, sy+inset, float32(fgCellSize)-inset*2, float32(fgCellSize)-inset*2, 1, color.RGBA{0xff, 0xff, 0xff, 0x80}, false)
		}
	}
}

func (f *Frogger) fgDrawObstacles(screen *ebiten.Image) {
	for _, o := range f.obstacles {
		oy := float32(o.row * fgCellSize)
		ox := float32(o.x)
		ow := float32(o.len * fgCellSize)
		oh := float32(fgCellSize)

		lane := f.lanes[o.row]
		if lane.rowType == fgRowWater {
			f.fgDrawLog(screen, ox, oy, ow, oh, o.clr)
		} else {
			f.fgDrawVehicle(screen, ox, oy, ow, oh, o.clr, lane.speed)
		}
	}
}

func (f *Frogger) fgDrawLog(screen *ebiten.Image, x, y, w, h float32, clr color.RGBA) {
	r := h / 2
	// Central rectangle
	vector.FillRect(screen, x+r, y+2, w-h, h-4, clr, false)
	// Rounded caps
	vector.FillCircle(screen, x+r, y+h/2, r-2, clr, false)
	vector.FillCircle(screen, x+w-r, y+h/2, r-2, clr, false)
	// Wood grain lines
	dr := uint8(20)
	dg := uint8(15)
	db := uint8(10)
	if clr.R < dr {
		dr = clr.R
	}
	if clr.G < dg {
		dg = clr.G
	}
	if clr.B < db {
		db = clr.B
	}
	darker := color.RGBA{clr.R - dr, clr.G - dg, clr.B - db, 0xff}
	for lx := x + 10; lx < x+w-10; lx += 14 {
		vector.StrokeLine(screen, lx, y+6, lx, y+h-6, 1, darker, false)
	}
}

func (f *Frogger) fgDrawVehicle(screen *ebiten.Image, x, y, w, h float32, clr color.RGBA, speed float64) {
	m := float32(3)
	// Body
	vector.FillRect(screen, x+m, y+m, w-m*2, h-m*2, clr, false)

	// Outline
	dr := int(clr.R) - 60
	dg := int(clr.G) - 60
	db := int(clr.B) - 60
	if dr < 0 {
		dr = 0
	}
	if dg < 0 {
		dg = 0
	}
	if db < 0 {
		db = 0
	}
	darker := color.RGBA{uint8(dr), uint8(dg), uint8(db), 0xff}
	vector.StrokeRect(screen, x+m, y+m, w-m*2, h-m*2, 1.5, darker, false)

	// Windshield on the leading edge
	wsClr := color.RGBA{0xaa, 0xdd, 0xff, 0xcc}
	if speed > 0 {
		vector.FillRect(screen, x+w-m-8, y+m+4, 5, h-m*2-8, wsClr, false)
	} else {
		vector.FillRect(screen, x+m+3, y+m+4, 5, h-m*2-8, wsClr, false)
	}

	// Wheels (4 small dark rectangles)
	wClr := color.RGBA{0x20, 0x20, 0x20, 0xff}
	vector.FillRect(screen, x+m+2, y+1, 6, 3, wClr, false)
	vector.FillRect(screen, x+w-m-8, y+1, 6, 3, wClr, false)
	vector.FillRect(screen, x+m+2, y+h-4, 6, 3, wClr, false)
	vector.FillRect(screen, x+w-m-8, y+h-4, 6, 3, wClr, false)

	// Headlights / tail-lights with glow
	if speed > 0 {
		// Headlights on right side (front) with glow
		vector.FillCircle(screen, x+w-m, y+m+4, 4, color.RGBA{0xff, 0xff, 0x80, 0x30}, false)
		vector.FillCircle(screen, x+w-m, y+h-m-4, 4, color.RGBA{0xff, 0xff, 0x80, 0x30}, false)
		vector.FillRect(screen, x+w-m-2, y+m+2, 3, 4, color.RGBA{0xff, 0xff, 0x80, 0xff}, false)
		vector.FillRect(screen, x+w-m-2, y+h-m-6, 3, 4, color.RGBA{0xff, 0xff, 0x80, 0xff}, false)
		// Tail lights
		vector.FillRect(screen, x+m, y+m+2, 2, 4, color.RGBA{0xff, 0x40, 0x40, 0xff}, false)
		vector.FillRect(screen, x+m, y+h-m-6, 2, 4, color.RGBA{0xff, 0x40, 0x40, 0xff}, false)
	} else {
		// Headlights on left side (front) with glow
		vector.FillCircle(screen, x+m, y+m+4, 4, color.RGBA{0xff, 0xff, 0x80, 0x30}, false)
		vector.FillCircle(screen, x+m, y+h-m-4, 4, color.RGBA{0xff, 0xff, 0x80, 0x30}, false)
		vector.FillRect(screen, x+m, y+m+2, 3, 4, color.RGBA{0xff, 0xff, 0x80, 0xff}, false)
		vector.FillRect(screen, x+m, y+h-m-6, 3, 4, color.RGBA{0xff, 0xff, 0x80, 0xff}, false)
		// Tail lights
		vector.FillRect(screen, x+w-m-2, y+m+2, 2, 4, color.RGBA{0xff, 0x40, 0x40, 0xff}, false)
		vector.FillRect(screen, x+w-m-2, y+h-m-6, 2, 4, color.RGBA{0xff, 0x40, 0x40, 0xff}, false)
	}

	// Tire marks / wake behind vehicles (faint)
	if speed > 0 {
		vector.FillRect(screen, x-4, y+2, 4, 2, color.RGBA{0x30, 0x30, 0x30, 0x40}, false)
		vector.FillRect(screen, x-4, y+h-4, 4, 2, color.RGBA{0x30, 0x30, 0x30, 0x40}, false)
	} else {
		vector.FillRect(screen, x+w, y+2, 4, 2, color.RGBA{0x30, 0x30, 0x30, 0x40}, false)
		vector.FillRect(screen, x+w, y+h-4, 4, 2, color.RGBA{0x30, 0x30, 0x30, 0x40}, false)
	}
}

func (f *Frogger) fgDrawFrog(screen *ebiten.Image) {
	fx := float32(f.frogCol*fgCellSize) + float32(fgCellSize)/2
	fy := float32(f.frogRow*fgCellSize) + float32(fgCellSize)/2

	bodyClr := fgColorFrog
	borderClr := color.RGBA{0x20, 0x90, 0x20, 0xff}
	if f.state == fgStateDying && (f.stateTick/4)%2 == 0 {
		bodyClr = fgColorDead
		borderClr = color.RGBA{0x90, 0x20, 0x20, 0xff}
	}

	bodyR := float32(fgCellSize)/2 - 3

	// Squish/stretch animation when jumping (during move cooldown)
	scaleX := float32(1.0)
	scaleY := float32(1.0)
	if f.moveTick > 0 {
		// Stretch in direction of movement, squish perpendicular
		jumpFrac := float64(f.moveTick) / float64(fgMoveCooldown)
		squish := float32(math.Sin(jumpFrac*math.Pi) * 0.25)
		scaleX = 1.0 + squish
		scaleY = 1.0 - squish
	}

	scaledRx := bodyR * scaleX
	scaledRy := bodyR * scaleY

	// Legs (4 small circles at corners, drawn first so body covers the join)
	legR := float32(4)
	legOffX := (bodyR - 2) * scaleX
	legOffY := (bodyR - 2) * scaleY
	vector.FillCircle(screen, fx-legOffX, fy+legOffY, legR, bodyClr, false)
	vector.FillCircle(screen, fx+legOffX, fy+legOffY, legR, bodyClr, false)
	vector.FillCircle(screen, fx-legOffX, fy-legOffY, legR, bodyClr, false)
	vector.FillCircle(screen, fx+legOffX, fy-legOffY, legR, bodyClr, false)

	// Body (use ellipse approximation via slightly scaled circle - draw as oval using rects)
	// Since there's no DrawEllipse, approximate with the average radius
	avgR := (scaledRx + scaledRy) / 2
	vector.FillCircle(screen, fx, fy, avgR, bodyClr, false)
	vector.StrokeCircle(screen, fx, fy, avgR, 1.5, borderClr, false)

	// Eyes
	eyeSpread := float32(5) * scaleX
	eyeUp := float32(5) * scaleY
	eyeR := float32(4)
	pupilR := float32(2)
	// Left
	vector.FillCircle(screen, fx-eyeSpread, fy-eyeUp, eyeR, fgColorFrogEye, false)
	vector.FillCircle(screen, fx-eyeSpread, fy-eyeUp, pupilR, fgColorFrogPup, false)
	// Right
	vector.FillCircle(screen, fx+eyeSpread, fy-eyeUp, eyeR, fgColorFrogEye, false)
	vector.FillCircle(screen, fx+eyeSpread, fy-eyeUp, pupilR, fgColorFrogPup, false)
}

func (f *Frogger) fgDrawHUD(screen *ebiten.Image) {
	// Semi-transparent bar at the bottom
	barH := float32(18)
	barY := float32(fgScreenH) - barH
	vector.FillRect(screen, 0, barY, float32(fgScreenW), barH, fgColorHUD, false)

	livesStr := fmt.Sprintf("Lives: %d", f.lives)
	ebitenutil.DebugPrintAt(screen, livesStr, 4, int(barY)+2)

	scoreStr := fmt.Sprintf("Score: %d", f.score)
	ebitenutil.DebugPrintAt(screen, scoreStr, 100, int(barY)+2)

	levelStr := fmt.Sprintf("Lv: %d", f.level)
	ebitenutil.DebugPrintAt(screen, levelStr, 230, int(barY)+2)

	// Time-bonus bar
	tbW := float32(100)
	tbH := float32(8)
	tbX := float32(fgScreenW) - tbW - 8
	tbY := barY + 5
	vector.StrokeRect(screen, tbX, tbY, tbW, tbH, 1, color.RGBA{0xff, 0xff, 0xff, 0x80}, false)
	frac := float32(f.bonusTimer) / float32(fgBonusTimerMax)
	if frac < 0 {
		frac = 0
	}
	tbClr := color.RGBA{0x40, 0xd0, 0x40, 0xff}
	if frac < 0.3 {
		tbClr = color.RGBA{0xd0, 0x40, 0x40, 0xff}
	} else if frac < 0.6 {
		tbClr = color.RGBA{0xd0, 0xc0, 0x20, 0xff}
	}
	vector.FillRect(screen, tbX+1, tbY+1, (tbW-2)*frac, tbH-2, tbClr, false)
}

func (f *Frogger) fgDrawOverlay(screen *ebiten.Image) {
	cx := fgScreenW / 2
	cy := fgScreenH / 2

	switch f.state {
	case fgStateDying:
		msg := "SPLAT!"
		ebitenutil.DebugPrintAt(screen, msg, cx-len(msg)*3, cy-20)

	case fgStateWin:
		boxW := float32(fgScreenW) / 2
		boxH := float32(40)
		boxX := (float32(fgScreenW) - boxW) / 2
		boxY := float32(cy) - boxH/2
		vector.FillRect(screen, boxX, boxY, boxW, boxH, fgColorHUD, false)
		vector.StrokeRect(screen, boxX, boxY, boxW, boxH, 1, color.RGBA{0xff, 0xff, 0xff, 0x60}, false)
		msg := fmt.Sprintf("LEVEL %d CLEAR!", f.level)
		ebitenutil.DebugPrintAt(screen, msg, cx-len(msg)*3, cy-6)

	case fgStateOver:
		boxW := float32(fgScreenW)*0.6 + 20
		boxH := float32(70)
		boxX := (float32(fgScreenW) - boxW) / 2
		boxY := float32(cy) - boxH/2
		vector.FillRect(screen, boxX, boxY, boxW, boxH, fgColorHUD, false)
		vector.StrokeRect(screen, boxX, boxY, boxW, boxH, 1, color.RGBA{0xff, 0x40, 0x40, 0x80}, false)
		msg1 := "GAME OVER"
		msg2 := fmt.Sprintf("Final Score: %d", f.score)
		msg3 := "Press R to restart  |  ESC for menu"
		ebitenutil.DebugPrintAt(screen, msg1, cx-len(msg1)*3, cy-28)
		ebitenutil.DebugPrintAt(screen, msg2, cx-len(msg2)*3, cy-10)
		ebitenutil.DebugPrintAt(screen, msg3, cx-len(msg3)*3, cy+10)
	}
}

