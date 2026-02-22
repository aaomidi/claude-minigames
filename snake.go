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
	snGridW    = 20
	snGridH    = 20
	snCellSize = 20
	snTopBar   = 40
	snScreenW  = snGridW * snCellSize
	snScreenH  = snGridH*snCellSize + snTopBar
)

// Directions
const (
	snDirUp    = 0
	snDirDown  = 1
	snDirLeft  = 2
	snDirRight = 3
)

// Base tick rate (frames per move). Decreases as snake grows.
const snBaseSpeed = 10

type snPoint struct {
	x, y int
}

type Snake struct {
	body      []snPoint
	dir       int
	nextDir   int
	food      snPoint
	score     int
	ticks     int
	speed     int
	gameOver  bool
	goBack    bool
	frameTick int // increments every frame for animation

	// Audio
	bgm       *audio.Player
	sfxChomp  []byte
	sfxCrash  []byte
	audioInit bool
}

func newSnake() *Snake {
	s := &Snake{}
	s.initSnAudio()
	s.init()
	return s
}

func (s *Snake) initSnAudio() {
	// --- BGM: Upbeat bouncy chiptune (~140 BPM), C major, energetic ---
	// Fast arpeggios and a catchy 8-bit melody
	// C4=60, D4=62, E4=64, F4=65, G4=67, A4=69, B4=71, C5=72
	melody := [][2]int{
		// A section - bouncy C major
		{72, 1}, {71, 1}, {69, 1}, {67, 1}, {69, 2}, {72, 2},
		{74, 1}, {72, 1}, {69, 1}, {67, 1}, {65, 2}, {67, 2},
		{64, 1}, {67, 1}, {69, 1}, {72, 1}, {74, 2}, {76, 2},
		{74, 1}, {72, 1}, {69, 2}, {67, 2}, {0, 2},
		// A' - varied
		{67, 1}, {69, 1}, {72, 1}, {74, 1}, {76, 2}, {74, 2},
		{72, 1}, {69, 1}, {67, 1}, {65, 1}, {64, 2}, {67, 2},
		{69, 1}, {72, 1}, {76, 1}, {74, 1}, {72, 2}, {69, 2},
		{67, 1}, {64, 1}, {65, 1}, {67, 1}, {72, 2}, {0, 2},
		// B section - shift to F major, new groove
		{65, 1}, {69, 1}, {72, 1}, {77, 1}, {76, 2}, {74, 2},
		{72, 1}, {74, 1}, {76, 1}, {77, 1}, {79, 2}, {77, 2},
		{76, 1}, {74, 1}, {72, 1}, {69, 1}, {65, 2}, {67, 2},
		{69, 1}, {72, 1}, {74, 2}, {0, 2}, {72, 2},
		// B' - builds higher
		{77, 1}, {79, 1}, {81, 1}, {79, 1}, {77, 2}, {76, 2},
		{74, 1}, {76, 1}, {77, 1}, {79, 1}, {81, 2}, {79, 2},
		{77, 1}, {76, 1}, {74, 1}, {72, 1}, {69, 2}, {67, 2},
		{65, 1}, {67, 1}, {69, 2}, {72, 2}, {0, 2},
		// C section - Am breakdown, surprise minor
		{69, 2}, {72, 2}, {76, 1}, {74, 1}, {72, 2},
		{69, 1}, {67, 1}, {64, 2}, {60, 2}, {0, 2},
		{64, 1}, {67, 1}, {69, 1}, {72, 1}, {71, 2}, {69, 2},
		{67, 1}, {64, 1}, {60, 1}, {64, 1}, {67, 2}, {0, 2},
		// Return to A - triumphant recap
		{72, 1}, {74, 1}, {76, 1}, {79, 1}, {81, 2}, {79, 2},
		{76, 1}, {74, 1}, {72, 1}, {69, 1}, {67, 2}, {69, 2},
		{72, 1}, {76, 1}, {79, 1}, {76, 1}, {74, 2}, {72, 2},
		{69, 1}, {67, 1}, {65, 1}, {67, 1}, {72, 4}, {0, 2},
	}
	melodyPCM := generateTrack(melody, squareWave, 140, 0.30)

	// Bouncy bass line with arpeggiated feel
	bass := [][2]int{
		// A section
		{48, 2}, {48, 1}, {55, 1}, {48, 2}, {55, 1}, {48, 1},
		{53, 2}, {53, 1}, {48, 1}, {53, 2}, {48, 1}, {53, 1},
		{48, 2}, {52, 1}, {55, 1}, {48, 2}, {55, 1}, {52, 1},
		{55, 2}, {53, 2}, {48, 2}, {55, 2},
		{55, 2}, {48, 1}, {55, 1}, {53, 2}, {48, 1}, {53, 1},
		{48, 2}, {52, 1}, {48, 1}, {55, 2}, {53, 1}, {55, 1},
		{48, 2}, {48, 1}, {52, 1}, {55, 2}, {48, 1}, {55, 1},
		{53, 2}, {55, 2}, {48, 2}, {48, 2},
		// B section - F major bass
		{53, 2}, {53, 1}, {48, 1}, {53, 2}, {48, 1}, {53, 1},
		{53, 2}, {55, 1}, {53, 1}, {48, 2}, {55, 1}, {53, 1},
		{53, 2}, {48, 1}, {53, 1}, {55, 2}, {48, 1}, {53, 1},
		{53, 2}, {55, 2}, {48, 2}, {48, 2},
		{53, 2}, {55, 1}, {53, 1}, {50, 2}, {48, 1}, {50, 1},
		{53, 2}, {55, 1}, {53, 1}, {48, 2}, {50, 1}, {48, 1},
		{53, 2}, {48, 1}, {50, 1}, {55, 2}, {53, 1}, {55, 1},
		{48, 2}, {50, 2}, {48, 2}, {48, 2},
		// C section - Am bass
		{45, 2}, {45, 1}, {48, 1}, {52, 2}, {48, 1}, {45, 1},
		{45, 2}, {43, 2}, {40, 2}, {45, 2},
		{45, 2}, {48, 1}, {45, 1}, {43, 2}, {40, 1}, {43, 1},
		{45, 2}, {43, 2}, {48, 2}, {48, 2},
		// Return
		{48, 2}, {48, 1}, {55, 1}, {48, 2}, {55, 1}, {48, 1},
		{53, 2}, {48, 1}, {55, 1}, {48, 2}, {55, 1}, {48, 1},
		{48, 2}, {52, 1}, {55, 1}, {53, 2}, {48, 1}, {55, 1},
		{48, 2}, {55, 2}, {48, 2}, {48, 2},
	}
	bassPCM := generateTrack(bass, triangleWave, 140, 0.25)

	// Fast drum pattern - driving beat
	drums := generateDrumPattern("X.h.s.h.X.h.s.h.X.h.s.h.X.h.shshX.hSs.h.X.h.s.hSXxh.s.h.Xsh.Kxhs", 140, 0.30)

	mixed := mixBuffers(melodyPCM, bassPCM)
	mixed = mixBuffers(mixed, drums)
	s.bgm = loopPlayer(mixed, 0.20)

	// --- SFX: Chomp (short rising chirp) ---
	chompDur := 0.1
	chompBuf := pcmStereo(chompDur)
	for i := 0; i < len(chompBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / chompDur
		// Rising pitch from C5 to C6
		freq := noteFreq(72) + (noteFreq(84)-noteFreq(72))*tFrac
		val := squareWave(t, freq) * envelope(tFrac, 0.01, 0.4, 0.59) * 0.5
		writeSample(chompBuf, i*4, val)
	}
	s.sfxChomp = chompBuf

	// --- SFX: Crash (descending noise burst) ---
	crashDur := 0.45
	crashBuf := pcmStereo(crashDur)
	for i := 0; i < len(crashBuf)/4; i++ {
		t := float64(i) / float64(sampleRate)
		tFrac := t / crashDur
		// Descending square tone mixed with noise
		freq := noteFreq(72) * math.Pow(0.2, tFrac)
		val := (squareWave(t, freq)*0.4 + noise()*0.6) * envelope(tFrac, 0.01, 0.2, 0.79) * 0.6
		writeSample(crashBuf, i*4, val)
	}
	s.sfxCrash = crashBuf
}

func (s *Snake) init() {
	// Start snake in the center, length 3, moving right
	cx, cy := snGridW/2, snGridH/2
	s.body = []snPoint{
		{cx, cy},
		{cx - 1, cy},
		{cx - 2, cy},
	}
	s.dir = snDirRight
	s.nextDir = snDirRight
	s.score = 0
	s.ticks = 0
	s.speed = snBaseSpeed
	s.gameOver = false
	s.spawnFood()
}

func (s *Snake) spawnFood() {
	for {
		fx := rand.Intn(snGridW)
		fy := rand.Intn(snGridH)
		onSnake := false
		for _, p := range s.body {
			if p.x == fx && p.y == fy {
				onSnake = true
				break
			}
		}
		if !onSnake {
			s.food = snPoint{fx, fy}
			return
		}
	}
}

func (s *Snake) WantsBack() bool {
	return s.goBack
}

func (s *Snake) Update() error {
	// Start BGM on first frame
	if !s.audioInit {
		s.audioInit = true
		if s.bgm != nil && !s.bgm.IsPlaying() {
			s.bgm.Play()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if s.bgm != nil && s.bgm.IsPlaying() {
			s.bgm.Pause()
		}
		s.goBack = true
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		goBack := s.goBack
		bgm := s.bgm
		sfxChomp := s.sfxChomp
		sfxCrash := s.sfxCrash
		audioInit := s.audioInit
		s.init()
		s.goBack = goBack
		s.bgm = bgm
		s.sfxChomp = sfxChomp
		s.sfxCrash = sfxCrash
		s.audioInit = audioInit
		return nil
	}

	s.frameTick++

	if s.gameOver {
		return nil
	}

	// Read direction input. Prevent 180-degree reversal.
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if s.dir != snDirDown {
			s.nextDir = snDirUp
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if s.dir != snDirUp {
			s.nextDir = snDirDown
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		if s.dir != snDirRight {
			s.nextDir = snDirLeft
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if s.dir != snDirLeft {
			s.nextDir = snDirRight
		}
	}

	s.ticks++
	if s.ticks < s.speed {
		return nil
	}
	s.ticks = 0

	// Apply direction
	s.dir = s.nextDir

	// Compute new head position
	head := s.body[0]
	var newHead snPoint
	switch s.dir {
	case snDirUp:
		newHead = snPoint{head.x, head.y - 1}
	case snDirDown:
		newHead = snPoint{head.x, head.y + 1}
	case snDirLeft:
		newHead = snPoint{head.x - 1, head.y}
	case snDirRight:
		newHead = snPoint{head.x + 1, head.y}
	}

	// Wall collision
	if newHead.x < 0 || newHead.x >= snGridW || newHead.y < 0 || newHead.y >= snGridH {
		s.gameOver = true
		playSFX(s.sfxCrash, 0.45)
		return nil
	}

	// Self collision (check against all body segments except the tail,
	// which will be removed unless we just ate food)
	for _, p := range s.body {
		if p.x == newHead.x && p.y == newHead.y {
			s.gameOver = true
			playSFX(s.sfxCrash, 0.45)
			return nil
		}
	}

	// Move: prepend new head
	s.body = append([]snPoint{newHead}, s.body...)

	// Check food
	if newHead.x == s.food.x && newHead.y == s.food.y {
		s.score++
		s.spawnFood()
		playSFX(s.sfxChomp, 0.40)
		// Increase speed every 5 points, down to a minimum of 3 frames
		s.speed = snBaseSpeed - s.score/5
		if s.speed < 3 {
			s.speed = 3
		}
	} else {
		// Remove tail
		s.body = s.body[:len(s.body)-1]
	}

	return nil
}

func (s *Snake) Draw(screen *ebiten.Image) {
	// Dark background
	screen.Fill(color.RGBA{0x1a, 0x1a, 0x2e, 0xff})

	// Draw grid area background
	vector.FillRect(screen, 0, float32(snTopBar), float32(snScreenW), float32(snGridH*snCellSize),
		color.RGBA{0x10, 0x10, 0x20, 0xff}, false)

	// Draw subtle grid pattern on background (checkerboard)
	for gx := 0; gx < snGridW; gx++ {
		for gy := 0; gy < snGridH; gy++ {
			if (gx+gy)%2 == 0 {
				fx := float32(gx * snCellSize)
				fy := float32(gy*snCellSize + snTopBar)
				vector.FillRect(screen, fx, fy, float32(snCellSize), float32(snCellSize),
					color.RGBA{0x18, 0x18, 0x2c, 0xff}, false)
			}
		}
	}

	// Draw grid lines (subtle)
	gridLineColor := color.RGBA{0x20, 0x20, 0x35, 0xff}
	for x := 0; x <= snGridW; x++ {
		fx := float32(x * snCellSize)
		vector.StrokeLine(screen, fx, float32(snTopBar), fx, float32(snScreenH), 1, gridLineColor, false)
	}
	for y := 0; y <= snGridH; y++ {
		fy := float32(y*snCellSize + snTopBar)
		vector.StrokeLine(screen, 0, fy, float32(snScreenW), fy, 1, gridLineColor, false)
	}

	// Draw glow trail behind snake (faint afterimage of recent body positions)
	if len(s.body) > 1 {
		for i := len(s.body) - 1; i >= 1; i-- {
			frac := float64(i) / float64(len(s.body))
			alpha := uint8(30 * (1.0 - frac))
			if alpha < 5 {
				continue
			}
			p := s.body[i]
			gpx := float32(p.x*snCellSize) + float32(snCellSize)/2
			gpy := float32(p.y*snCellSize+snTopBar) + float32(snCellSize)/2
			vector.FillCircle(screen, gpx, gpy, float32(snCellSize)/2+2, color.RGBA{0x40, 0xe0, 0x40, alpha}, false)
		}
	}

	// Draw food with pulse/breathe (size oscillation)
	foodPulse := float32(math.Sin(float64(s.frameTick)*0.12) * 2)
	foodX := float32(s.food.x*snCellSize) + 2 - foodPulse/2
	foodY := float32(s.food.y*snCellSize+snTopBar) + 2 - foodPulse/2
	foodSize := float32(snCellSize) - 4 + foodPulse
	// Food glow
	foodGlowAlpha := uint8(40 + int(math.Sin(float64(s.frameTick)*0.12)*25))
	foodCx := float32(s.food.x*snCellSize) + float32(snCellSize)/2
	foodCy := float32(s.food.y*snCellSize+snTopBar) + float32(snCellSize)/2
	vector.FillCircle(screen, foodCx, foodCy, float32(snCellSize)/2+3, color.RGBA{0xe0, 0x30, 0x30, foodGlowAlpha}, false)
	vector.FillRect(screen, foodX, foodY, foodSize, foodSize, color.RGBA{0xe0, 0x30, 0x30, 0xff}, false)

	// Draw snake with gradient body (head brightest, fades toward tail)
	for i := len(s.body) - 1; i >= 0; i-- {
		p := s.body[i]
		px := float32(p.x*snCellSize) + 1
		py := float32(p.y*snCellSize+snTopBar) + 1
		sz := float32(snCellSize) - 2

		var c color.RGBA
		if i == 0 {
			// Head: brightest green
			c = color.RGBA{0x40, 0xe0, 0x40, 0xff}
		} else {
			// Gradient: fade from bright to darker toward tail
			frac := float64(i) / float64(len(s.body))
			g8 := uint8(0xe0 - int(frac*float64(0xe0-0x50)))
			r8 := uint8(0x30 - int(frac*float64(0x30-0x10)))
			b8 := uint8(0x30 - int(frac*float64(0x30-0x10)))
			c = color.RGBA{r8, g8, b8, 0xff}
		}
		vector.FillRect(screen, px, py, sz, sz, c, false)
	}

	// Draw eyes on snake head that face the movement direction
	if len(s.body) > 0 {
		head := s.body[0]
		hx := float32(head.x*snCellSize) + float32(snCellSize)/2
		hy := float32(head.y*snCellSize+snTopBar) + float32(snCellSize)/2
		eyeR := float32(3)
		pupilR := float32(1.5)
		var e1x, e1y, e2x, e2y float32
		var pdx, pdy float32
		switch s.dir {
		case snDirUp:
			e1x, e1y = hx-4, hy-3
			e2x, e2y = hx+4, hy-3
			pdx, pdy = 0, -1
		case snDirDown:
			e1x, e1y = hx-4, hy+3
			e2x, e2y = hx+4, hy+3
			pdx, pdy = 0, 1
		case snDirLeft:
			e1x, e1y = hx-3, hy-4
			e2x, e2y = hx-3, hy+4
			pdx, pdy = -1, 0
		case snDirRight:
			e1x, e1y = hx+3, hy-4
			e2x, e2y = hx+3, hy+4
			pdx, pdy = 1, 0
		}
		vector.FillCircle(screen, e1x, e1y, eyeR, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
		vector.FillCircle(screen, e2x, e2y, eyeR, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)
		vector.FillCircle(screen, e1x+pdx*1.5, e1y+pdy*1.5, pupilR, color.RGBA{0x10, 0x10, 0x10, 0xff}, false)
		vector.FillCircle(screen, e2x+pdx*1.5, e2y+pdy*1.5, pupilR, color.RGBA{0x10, 0x10, 0x10, 0xff}, false)
	}

	// Draw score in top bar
	scoreStr := fmt.Sprintf("Score: %d", s.score)
	ebitenutil.DebugPrintAt(screen, scoreStr, 8, 8)
	controlsStr := "Arrow/WASD  ESC=menu  R=restart"
	ebitenutil.DebugPrintAt(screen, controlsStr, 8, 24)

	// Game over overlay
	if s.gameOver {
		// Semi-transparent overlay
		vector.FillRect(screen, 0, 0, float32(snScreenW), float32(snScreenH),
			color.RGBA{0x00, 0x00, 0x00, 0xc0}, false)

		msg := "GAME OVER"
		msgX := snScreenW/2 - len(msg)*3
		msgY := snScreenH/2 - 20
		ebitenutil.DebugPrintAt(screen, msg, msgX, msgY)

		finalScore := fmt.Sprintf("Score: %d", s.score)
		fsX := snScreenW/2 - len(finalScore)*3
		ebitenutil.DebugPrintAt(screen, finalScore, fsX, msgY+20)

		restart := "Press R to restart"
		rsX := snScreenW/2 - len(restart)*3
		ebitenutil.DebugPrintAt(screen, restart, rsX, msgY+40)
	}
}

func (s *Snake) Layout(w, h int) (int, int) {
	return snScreenW, snScreenH
}
