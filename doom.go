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

// ── Screen constants ────────────────────────────────────────────────────────
const (
	dmScreenW = 480
	dmScreenH = 360
	dmFOV     = math.Pi / 3 // 60 degrees
	dmMapW    = 24
	dmMapH    = 24
)

// ── Movement / gameplay constants ───────────────────────────────────────────
const (
	dmMoveSpeed     = 0.05
	dmRotSpeed      = 0.03
	dmMouseSens     = 0.003
	dmMaxHP         = 100
	dmMaxAmmo       = 99
	dmShootCooldown = 12
	dmShootRange    = 16.0
	dmShootAngle    = 0.1 // radians tolerance for hitting
	dmMuzzleTicks   = 4
	dmHurtTicks     = 20
	dmMsgTicks      = 90
	dmMinimapSize   = 80
	dmMinimapX      = dmScreenW - dmMinimapSize - 6
	dmMinimapY      = 6
	dmCollisionR    = 0.2
)

// ── Wall types ──────────────────────────────────────────────────────────────
// 0 = empty, 1 = gray stone, 2 = red brick, 3 = blue steel, 4 = green moss
// 5+ reserved for boundaries

// ── Map definition ──────────────────────────────────────────────────────────
var dmWorldMap = [dmMapH][dmMapW]int{
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 2, 2, 2, 0, 0, 3, 3, 3, 3, 0, 0, 0, 4, 4, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 2, 0, 2, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 2, 0, 2, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 4, 4, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 4, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 4, 0, 1},
	{1, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 0, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 3, 0, 0, 2, 2, 2, 2, 2, 0, 0, 1},
	{1, 0, 0, 2, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 1},
	{1, 0, 0, 2, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 1},
	{1, 0, 0, 2, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 4, 4, 4, 4, 0, 0, 0, 0, 0, 2, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 4, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 1, 0, 0, 1, 1, 1, 3, 0, 0, 4, 0, 0, 4, 0, 0, 2, 0, 2, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 4, 0, 4, 0, 0, 2, 0, 2, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 3, 3, 0, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 3, 0, 0, 0, 3, 0, 0, 0, 0, 2, 0, 2, 0, 0, 4, 4, 4, 4, 4, 4, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

// ── Enemy types ─────────────────────────────────────────────────────────────
const (
	dmEnemyImp     = 0
	dmEnemySoldier = 1
	dmEnemyDemon   = 2
)

type dmEnemy struct {
	x, y      float64
	hp        int
	maxHP     int
	eType     int
	speed     float64
	attackDmg int
	attackRng float64
	attackCD  int // cooldown timer
	alive     bool
	deathTick int
	seePlayer bool
}

func dmNewEnemy(eType int, x, y float64) dmEnemy {
	e := dmEnemy{x: x, y: y, eType: eType, alive: true}
	switch eType {
	case dmEnemyImp:
		e.hp, e.maxHP = 30, 30
		e.speed = 0.015
		e.attackDmg = 8
		e.attackRng = 1.5
	case dmEnemySoldier:
		e.hp, e.maxHP = 50, 50
		e.speed = 0.025
		e.attackDmg = 12
		e.attackRng = 8.0
	case dmEnemyDemon:
		e.hp, e.maxHP = 100, 100
		e.speed = 0.04
		e.attackDmg = 20
		e.attackRng = 1.8
	}
	return e
}

func dmEnemyColor(eType int) color.RGBA {
	switch eType {
	case dmEnemyImp:
		return color.RGBA{220, 50, 50, 255}
	case dmEnemySoldier:
		return color.RGBA{50, 200, 50, 255}
	case dmEnemyDemon:
		return color.RGBA{180, 50, 220, 255}
	}
	return color.RGBA{255, 255, 255, 255}
}

func dmEnemyName(eType int) string {
	switch eType {
	case dmEnemyImp:
		return "IMP"
	case dmEnemySoldier:
		return "SOLDIER"
	case dmEnemyDemon:
		return "DEMON"
	}
	return "ENEMY"
}

// ── Pickup types ────────────────────────────────────────────────────────────
const (
	dmPickupHealth = 0
	dmPickupAmmo   = 1
)

type dmPickup struct {
	x, y    float64
	pType   int
	active  bool
}

// ── Game state ──────────────────────────────────────────────────────────────
type Doom struct {
	goBack bool

	// Player state
	posX, posY float64
	angle      float64
	dirX, dirY float64
	hp         int
	ammo       int
	score      int
	alive      bool

	// Timing / effects
	shootCD    int
	muzzleTick int
	hurtTick   int
	hurtDir    float64 // angle to attacker relative to player facing
	msgText    string
	msgTimer   int
	prevMouseX int
	mouseInit  bool

	// Entities
	enemies       []dmEnemy
	pickups       []dmPickup
	bloodParticles []dmBloodParticle

	// Z-buffer for sprite rendering
	zBuf [dmScreenW]float64

	// Audio
	bgm        *audio.Player
	sfxShoot   []byte
	sfxDeath   []byte
	sfxPickup  []byte
	sfxHurt    []byte

	// Game over
	gameOver   bool
	gameOverT  int

	// Tick counter
	ticks int
}

// ── Wall colors ─────────────────────────────────────────────────────────────
func dmWallColor(wallType int, side int, dist float64) color.RGBA {
	var base color.RGBA
	switch wallType {
	case 1: // gray stone
		base = color.RGBA{160, 160, 160, 255}
	case 2: // red brick
		base = color.RGBA{180, 60, 60, 255}
	case 3: // blue steel
		base = color.RGBA{60, 80, 180, 255}
	case 4: // green moss
		base = color.RGBA{60, 160, 60, 255}
	default:
		base = color.RGBA{200, 200, 200, 255}
	}

	// Side shading: east-west walls slightly darker
	sideMul := 1.0
	if side == 1 {
		sideMul = 0.75
	}

	// Distance darkening
	distMul := 1.0 / (1.0 + dist*0.08)
	if distMul < 0.1 {
		distMul = 0.1
	}

	mul := sideMul * distMul
	return color.RGBA{
		uint8(float64(base.R) * mul),
		uint8(float64(base.G) * mul),
		uint8(float64(base.B) * mul),
		255,
	}
}

// ── Constructor ─────────────────────────────────────────────────────────────
func newDoom() *Doom {
	d := &Doom{}
	d.initGame()
	d.initAudio()
	return d
}

func (d *Doom) initGame() {
	d.posX = 2.5
	d.posY = 2.5
	d.angle = 0.0
	d.dirX = math.Cos(d.angle)
	d.dirY = math.Sin(d.angle)
	d.hp = dmMaxHP
	d.ammo = 50
	d.score = 0
	d.alive = true
	d.gameOver = false
	d.gameOverT = 0
	d.shootCD = 0
	d.muzzleTick = 0
	d.hurtTick = 0
	d.msgText = ""
	d.msgTimer = 0
	d.mouseInit = false
	d.ticks = 0

	// Spawn enemies
	d.enemies = []dmEnemy{
		dmNewEnemy(dmEnemyImp, 5.5, 5.5),
		dmNewEnemy(dmEnemyImp, 10.5, 3.5),
		dmNewEnemy(dmEnemySoldier, 14.5, 5.5),
		dmNewEnemy(dmEnemySoldier, 8.5, 12.5),
		dmNewEnemy(dmEnemyDemon, 18.5, 7.5),
		dmNewEnemy(dmEnemyImp, 4.5, 14.5),
		dmNewEnemy(dmEnemySoldier, 12.5, 17.5),
		dmNewEnemy(dmEnemyDemon, 4.5, 21.5),
		dmNewEnemy(dmEnemyImp, 18.5, 17.5),
		dmNewEnemy(dmEnemySoldier, 21.5, 13.5),
		dmNewEnemy(dmEnemyDemon, 13.5, 13.5),
		dmNewEnemy(dmEnemyImp, 20.5, 20.5),
	}

	// Spawn pickups
	d.pickups = []dmPickup{
		{x: 6.5, y: 1.5, pType: dmPickupHealth, active: true},
		{x: 1.5, y: 10.5, pType: dmPickupHealth, active: true},
		{x: 22.5, y: 1.5, pType: dmPickupAmmo, active: true},
		{x: 15.5, y: 15.5, pType: dmPickupAmmo, active: true},
		{x: 10.5, y: 10.5, pType: dmPickupHealth, active: true},
		{x: 3.5, y: 18.5, pType: dmPickupAmmo, active: true},
		{x: 22.5, y: 22.5, pType: dmPickupHealth, active: true},
		{x: 18.5, y: 13.5, pType: dmPickupAmmo, active: true},
	}
}

// ── Audio generation ────────────────────────────────────────────────────────
func (d *Doom) initAudio() {
	// BGM: Dark metal riff in E minor, ~120 BPM, 80+ beats
	// E2=40, B2=47, G2=43, A2=45, D3=50, E3=52
	riffNotes := [][2]int{
		// Heavy E minor power chord riff
		{28, 2}, {0, 1}, {28, 1}, {28, 2}, {0, 1}, {31, 1}, // E1 riff
		{33, 2}, {0, 1}, {33, 1}, {31, 2}, {0, 1}, {28, 1}, // A1-G1
		{28, 2}, {0, 1}, {35, 1}, {33, 2}, {0, 1}, {31, 1}, // E1 up to B1
		{28, 4}, {0, 2}, {28, 1}, {0, 1},                   // sustain
		// Second phrase - lower and heavier
		{26, 2}, {0, 1}, {28, 1}, {31, 2}, {0, 1}, {33, 1},
		{35, 2}, {0, 1}, {33, 1}, {31, 2}, {0, 1}, {28, 1},
		{26, 2}, {0, 1}, {28, 1}, {26, 2}, {0, 1}, {24, 1},
		{28, 4}, {0, 2}, {28, 1}, {0, 1},
		// Third phrase - chromatic tension
		{28, 1}, {29, 1}, {28, 1}, {0, 1}, {31, 1}, {32, 1}, {31, 1}, {0, 1},
		{33, 1}, {34, 1}, {33, 1}, {0, 1}, {35, 2}, {33, 2},
		// Resolution
		{31, 2}, {28, 2}, {0, 1}, {28, 1}, {28, 1}, {0, 1},
	}
	riffPCM := generateTrack(riffNotes, squareWave, 120, 0.25)

	drumPattern := "K.h.s.h.K.h.s.h.K.h.s.h.KKh.s.h." +
		"K.h.s.h.K.h.s.h.K.h.s.h.K.hSs.h." +
		"K.h.s.h.K.h.s.hSK.h.s.h.K.hSs.h." +
		"K.h.s.h.K.h.s.h.K.K.s.h.KKhSs.h."
	drumPCM := generateDrumPattern(drumPattern, 120, 0.3)

	bgmPCM := mixBuffers(riffPCM, drumPCM)
	d.bgm = loopPlayer(bgmPCM, 0.35)
	d.bgm.Play()

	// SFX: gunshot - sharp noise burst
	{
		dur := 0.15
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := t / dur
			val := noise() * math.Exp(-t*30) * 0.8
			val += squareWave(t, 120) * math.Exp(-t*20) * 0.3
			val *= (1.0 - tFrac)
			writeSample(buf, i*4, val)
		}
		d.sfxShoot = buf
	}

	// SFX: enemy death - descending tone
	{
		dur := 0.4
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := t / dur
			freq := 400.0 - 300.0*tFrac
			val := squareWave(t, freq) * 0.4 * (1.0 - tFrac)
			val += noise() * 0.1 * (1.0 - tFrac)
			writeSample(buf, i*4, val)
		}
		d.sfxDeath = buf
	}

	// SFX: pickup - rising ding
	{
		dur := 0.25
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := t / dur
			freq := 600.0 + 400.0*tFrac
			val := sineWave(t, freq) * 0.4 * (1.0 - tFrac*0.7)
			writeSample(buf, i*4, val)
		}
		d.sfxPickup = buf
	}

	// SFX: player hurt - low thud
	{
		dur := 0.2
		buf := pcmStereo(dur)
		samples := len(buf) / 4
		for i := 0; i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := t / dur
			freq := 80.0 * math.Exp(-t*10)
			val := sineWave(t, freq)*0.6*(1.0-tFrac) + noise()*0.15*(1.0-tFrac)
			writeSample(buf, i*4, val)
		}
		d.sfxHurt = buf
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────
func dmIsWall(mx, my int) bool {
	if mx < 0 || mx >= dmMapW || my < 0 || my >= dmMapH {
		return true
	}
	return dmWorldMap[my][mx] != 0
}

func dmCanSee(x0, y0, x1, y1 float64) bool {
	dx := x1 - x0
	dy := y1 - y0
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 0.01 {
		return true
	}
	steps := int(dist * 20)
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		cx := x0 + dx*t
		cy := y0 + dy*t
		mx := int(cx)
		my := int(cy)
		if mx < 0 || mx >= dmMapW || my < 0 || my >= dmMapH {
			return false
		}
		if dmWorldMap[my][mx] != 0 {
			return false
		}
	}
	return true
}

func dmClamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func dmNormAngle(a float64) float64 {
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	return a
}

func (d *Doom) showMsg(msg string) {
	d.msgText = msg
	d.msgTimer = dmMsgTicks
}

func (d *Doom) spawnBlood(wx, wy float64, count int) {
	for i := 0; i < count; i++ {
		angle := float64(i) / float64(count) * 2 * math.Pi
		speed := 0.01 + float64(i%3)*0.005
		d.bloodParticles = append(d.bloodParticles, dmBloodParticle{
			x: wx, y: wy,
			vx:      math.Cos(angle) * speed,
			vy:      math.Sin(angle) * speed,
			life:    15 + (i % 10),
			maxLife: 15 + (i % 10),
		})
	}
}

// ── Update ──────────────────────────────────────────────────────────────────
func (d *Doom) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if d.bgm != nil {
			d.bgm.Pause()
		}
		d.goBack = true
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if d.bgm != nil {
			d.bgm.Pause()
		}
		d.initGame()
		d.initAudio()
		return nil
	}

	d.ticks++
	if d.msgTimer > 0 {
		d.msgTimer--
	}
	if d.hurtTick > 0 {
		d.hurtTick--
	}
	if d.muzzleTick > 0 {
		d.muzzleTick--
	}
	if d.shootCD > 0 {
		d.shootCD--
	}

	// Game over state
	if d.gameOver {
		d.gameOverT++
		return nil
	}

	// ── Player movement ─────────────────────────────────────────────────
	// Mouse look
	mx, _ := ebiten.CursorPosition()
	if !d.mouseInit {
		d.prevMouseX = mx
		d.mouseInit = true
	}
	mouseDX := mx - d.prevMouseX
	d.prevMouseX = mx
	if mouseDX != 0 {
		d.angle += float64(mouseDX) * dmMouseSens
	}

	// Keyboard rotation
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		d.angle -= dmRotSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		d.angle += dmRotSpeed
	}

	d.dirX = math.Cos(d.angle)
	d.dirY = math.Sin(d.angle)

	// Movement with collision
	moveX := 0.0
	moveY := 0.0
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		moveX += d.dirX * dmMoveSpeed
		moveY += d.dirY * dmMoveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		moveX -= d.dirX * dmMoveSpeed
		moveY -= d.dirY * dmMoveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		// Strafe left: perpendicular to direction
		moveX += d.dirY * dmMoveSpeed
		moveY -= d.dirX * dmMoveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		// Strafe right
		moveX -= d.dirY * dmMoveSpeed
		moveY += d.dirX * dmMoveSpeed
	}

	// Apply collision detection (check x and y separately for wall sliding)
	newX := d.posX + moveX
	newY := d.posY + moveY
	if !dmIsWall(int(newX+dmCollisionR*sign(moveX)), int(d.posY)) &&
		!dmIsWall(int(newX-dmCollisionR*sign(-moveX)), int(d.posY)) {
		d.posX = newX
	}
	if !dmIsWall(int(d.posX), int(newY+dmCollisionR*sign(moveY))) &&
		!dmIsWall(int(d.posX), int(newY-dmCollisionR*sign(-moveY))) {
		d.posY = newY
	}

	// ── Shooting ────────────────────────────────────────────────────────
	if ebiten.IsKeyPressed(ebiten.KeySpace) && d.shootCD <= 0 && d.ammo > 0 {
		d.ammo--
		d.shootCD = dmShootCooldown
		d.muzzleTick = dmMuzzleTicks
		playSFX(d.sfxShoot, 0.5)

		// Find closest enemy near crosshair
		bestDist := math.MaxFloat64
		bestIdx := -1
		for i := range d.enemies {
			e := &d.enemies[i]
			if !e.alive {
				continue
			}
			dx := e.x - d.posX
			dy := e.y - d.posY
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > dmShootRange {
				continue
			}
			// Angle to enemy
			enemyAngle := math.Atan2(dy, dx)
			angleDiff := dmNormAngle(enemyAngle - d.angle)
			if math.Abs(angleDiff) < dmShootAngle && dist < bestDist {
				// Also check line of sight
				if dmCanSee(d.posX, d.posY, e.x, e.y) {
					bestDist = dist
					bestIdx = i
				}
			}
		}

		if bestIdx >= 0 {
			e := &d.enemies[bestIdx]
			// Damage based on distance (more damage close up)
			dmg := int(50.0 / (1.0 + bestDist*0.5))
			if dmg < 5 {
				dmg = 5
			}
			e.hp -= dmg

			// Spawn blood particles at enemy position
			d.spawnBlood(e.x, e.y, 8)

			if e.hp <= 0 {
				e.alive = false
				e.deathTick = 30
				pts := (e.maxHP / 10) * 10
				d.score += pts
				playSFX(d.sfxDeath, 0.5)
				d.showMsg(fmt.Sprintf("%s KILLED! +%d", dmEnemyName(e.eType), pts))
				// Extra blood burst on kill
				d.spawnBlood(e.x, e.y, 20)
			}
		}
	}

	// ── Pickup collection ───────────────────────────────────────────────
	for i := range d.pickups {
		p := &d.pickups[i]
		if !p.active {
			continue
		}
		dx := p.x - d.posX
		dy := p.y - d.posY
		if dx*dx+dy*dy < 0.5 {
			p.active = false
			playSFX(d.sfxPickup, 0.5)
			switch p.pType {
			case dmPickupHealth:
				d.hp += 25
				if d.hp > dmMaxHP {
					d.hp = dmMaxHP
				}
				d.showMsg("HEALTH +25")
			case dmPickupAmmo:
				d.ammo += 10
				if d.ammo > dmMaxAmmo {
					d.ammo = dmMaxAmmo
				}
				d.showMsg("AMMO +10")
			}
		}
	}

	// ── Update blood particles ──────────────────────────────────────────
	alive := d.bloodParticles[:0]
	for i := range d.bloodParticles {
		bp := &d.bloodParticles[i]
		bp.life--
		bp.x += bp.vx
		bp.y += bp.vy
		bp.vx *= 0.95
		bp.vy *= 0.95
		if bp.life > 0 {
			alive = append(alive, *bp)
		}
	}
	d.bloodParticles = alive

	// ── Enemy AI ────────────────────────────────────────────────────────
	for i := range d.enemies {
		e := &d.enemies[i]
		if !e.alive {
			if e.deathTick > 0 {
				e.deathTick--
			}
			continue
		}

		// Attack cooldown
		if e.attackCD > 0 {
			e.attackCD--
		}

		// Check line of sight to player
		e.seePlayer = dmCanSee(e.x, e.y, d.posX, d.posY)

		if e.seePlayer {
			dx := d.posX - e.x
			dy := d.posY - e.y
			dist := math.Sqrt(dx*dx + dy*dy)

			// Move toward player if not in attack range
			if dist > e.attackRng*0.8 {
				ndx := dx / dist * e.speed
				ndy := dy / dist * e.speed
				nx := e.x + ndx
				ny := e.y + ndy
				if !dmIsWall(int(nx), int(ny)) {
					e.x = nx
					e.y = ny
				} else if !dmIsWall(int(nx), int(e.y)) {
					e.x = nx
				} else if !dmIsWall(int(e.x), int(ny)) {
					e.y = ny
				}
			}

			// Attack player
			if dist < e.attackRng && e.attackCD <= 0 {
				e.attackCD = 60 // 1 second cooldown at 60fps
				d.hp -= e.attackDmg
				d.hurtTick = dmHurtTicks
				// Store direction to attacker for damage indicator
				d.hurtDir = math.Atan2(e.y-d.posY, e.x-d.posX) - d.angle
				playSFX(d.sfxHurt, 0.5)
				if d.hp <= 0 {
					d.hp = 0
					d.alive = false
					d.gameOver = true
					d.gameOverT = 0
				} else if d.hp < 25 {
					d.showMsg("LOW HEALTH!")
				}
			}
		}
	}

	return nil
}

func sign(x float64) float64 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

// ── DDA Raycasting ──────────────────────────────────────────────────────────
type dmRayResult struct {
	dist     float64
	wallType int
	side     int // 0 = x-side (N/S wall face), 1 = y-side (E/W wall face)
	wallX    float64 // where on the wall the ray hit (0.0-1.0)
}

func dmCastRay(posX, posY, rayDirX, rayDirY float64) dmRayResult {
	// Which map cell we're in
	mapX := int(posX)
	mapY := int(posY)

	// Length of ray from one x/y side to next x/y side
	deltaDistX := math.MaxFloat64
	if rayDirX != 0 {
		deltaDistX = math.Abs(1.0 / rayDirX)
	}
	deltaDistY := math.MaxFloat64
	if rayDirY != 0 {
		deltaDistY = math.Abs(1.0 / rayDirY)
	}

	// Step direction and initial side distance
	var stepX, stepY int
	var sideDistX, sideDistY float64

	if rayDirX < 0 {
		stepX = -1
		sideDistX = (posX - float64(mapX)) * deltaDistX
	} else {
		stepX = 1
		sideDistX = (float64(mapX) + 1.0 - posX) * deltaDistX
	}
	if rayDirY < 0 {
		stepY = -1
		sideDistY = (posY - float64(mapY)) * deltaDistY
	} else {
		stepY = 1
		sideDistY = (float64(mapY) + 1.0 - posY) * deltaDistY
	}

	// DDA loop
	side := 0
	for steps := 0; steps < 100; steps++ {
		if sideDistX < sideDistY {
			sideDistX += deltaDistX
			mapX += stepX
			side = 0
		} else {
			sideDistY += deltaDistY
			mapY += stepY
			side = 1
		}

		if mapX < 0 || mapX >= dmMapW || mapY < 0 || mapY >= dmMapH {
			break
		}

		if dmWorldMap[mapY][mapX] > 0 {
			// Calculate perpendicular distance to avoid fisheye
			var perpDist float64
			if side == 0 {
				perpDist = (float64(mapX) - posX + float64(1-stepX)/2.0) / rayDirX
			} else {
				perpDist = (float64(mapY) - posY + float64(1-stepY)/2.0) / rayDirY
			}
			if perpDist < 0.001 {
				perpDist = 0.001
			}

			// Calculate wallX (where exactly the wall was hit)
			var wallX float64
			if side == 0 {
				wallX = posY + perpDist*rayDirY
			} else {
				wallX = posX + perpDist*rayDirX
			}
			wallX -= math.Floor(wallX)

			return dmRayResult{
				dist:     perpDist,
				wallType: dmWorldMap[mapY][mapX],
				side:     side,
				wallX:    wallX,
			}
		}
	}

	return dmRayResult{dist: 100.0, wallType: 0, side: 0}
}

// ── Blood particle system ───────────────────────────────────────────────────
type dmBloodParticle struct {
	x, y    float64 // world position
	vx, vy  float64 // velocity
	life    int     // remaining ticks
	maxLife int
}

// ── Sprite rendering data ───────────────────────────────────────────────────
type dmSpriteInfo struct {
	screenX    int
	dist       float64
	halfW      int
	drawStartY int
	drawEndY   int
	colR, colG, colB uint8
	isPickup     bool
	eType        int  // enemy type (imp/soldier/demon)
	pickupType   int  // pickup type (health/ammo)
	isDying      bool // death animation in progress
	deathTick    int  // death animation frame
	brightness   float64 // distance-based brightness multiplier
	spriteHeight int  // full sprite height in pixels
}

func (d *Doom) buildSprites() []dmSpriteInfo {
	sprites := []dmSpriteInfo{}

	// Camera plane (perpendicular to direction)
	planeX := -d.dirY * math.Tan(dmFOV/2)
	planeY := d.dirX * math.Tan(dmFOV/2)

	// Enemies
	for i := range d.enemies {
		e := &d.enemies[i]
		if !e.alive && e.deathTick <= 0 {
			continue
		}
		dx := e.x - d.posX
		dy := e.y - d.posY
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.1 {
			continue
		}

		// Transform sprite with inverse camera matrix
		invDet := 1.0 / (planeX*d.dirY - d.dirX*planeY)
		transformX := invDet * (d.dirY*dx - d.dirX*dy)
		transformY := invDet * (-planeY*dx + planeX*dy)

		if transformY <= 0.1 {
			continue // behind camera
		}

		screenX := int(float64(dmScreenW)/2.0 * (1.0 + transformX/transformY))

		spriteH := int(float64(dmScreenH) / transformY)
		if spriteH < 1 {
			spriteH = 1
		}
		if spriteH > dmScreenH*2 {
			spriteH = dmScreenH * 2
		}

		drawStartY := dmScreenH/2 - spriteH/2
		drawEndY := dmScreenH/2 + spriteH/2

		col := dmEnemyColor(e.eType)

		// If dying, flash white
		if !e.alive && e.deathTick > 0 {
			flash := float64(e.deathTick) / 30.0
			col.R = uint8(dmClamp(float64(col.R)+flash*200, 0, 255))
			col.G = uint8(dmClamp(float64(col.G)+flash*200, 0, 255))
			col.B = uint8(dmClamp(float64(col.B)+flash*200, 0, 255))
		}

		// Distance darkening
		distMul := 1.0 / (1.0 + transformY*0.08)
		if distMul < 0.15 {
			distMul = 0.15
		}

		halfW := spriteH / 4
		if halfW < 1 {
			halfW = 1
		}

		isDying := !e.alive && e.deathTick > 0
		sprites = append(sprites, dmSpriteInfo{
			screenX:      screenX,
			dist:         transformY,
			halfW:        halfW,
			drawStartY:   drawStartY,
			drawEndY:     drawEndY,
			colR:         uint8(float64(col.R) * distMul),
			colG:         uint8(float64(col.G) * distMul),
			colB:         uint8(float64(col.B) * distMul),
			eType:        e.eType,
			isDying:      isDying,
			deathTick:    e.deathTick,
			brightness:   distMul,
			spriteHeight: spriteH,
		})
	}

	// Pickups
	for i := range d.pickups {
		p := &d.pickups[i]
		if !p.active {
			continue
		}
		dx := p.x - d.posX
		dy := p.y - d.posY
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.1 {
			continue
		}

		invDet := 1.0 / (planeX*d.dirY - d.dirX*planeY)
		transformX := invDet * (d.dirY*dx - d.dirX*dy)
		transformY := invDet * (-planeY*dx + planeX*dy)

		if transformY <= 0.1 {
			continue
		}

		screenX := int(float64(dmScreenW)/2.0 * (1.0 + transformX/transformY))

		spriteH := int(float64(dmScreenH) / transformY / 3) // pickups are smaller
		if spriteH < 1 {
			spriteH = 1
		}

		drawStartY := dmScreenH/2 + spriteH/2 - spriteH
		drawEndY := dmScreenH/2 + spriteH/2

		var col color.RGBA
		switch p.pType {
		case dmPickupHealth:
			col = color.RGBA{0, 220, 0, 255}
		case dmPickupAmmo:
			col = color.RGBA{220, 220, 0, 255}
		}

		distMul := 1.0 / (1.0 + transformY*0.08)
		if distMul < 0.15 {
			distMul = 0.15
		}

		halfW := spriteH / 2
		if halfW < 1 {
			halfW = 1
		}

		sprites = append(sprites, dmSpriteInfo{
			screenX:      screenX,
			dist:         transformY,
			halfW:        halfW,
			drawStartY:   drawStartY,
			drawEndY:     drawEndY,
			colR:         uint8(float64(col.R) * distMul),
			colG:         uint8(float64(col.G) * distMul),
			colB:         uint8(float64(col.B) * distMul),
			isPickup:     true,
			pickupType:   p.pType,
			brightness:   distMul,
			spriteHeight: spriteH,
		})
	}

	// Sort by distance (farthest first for painter's algorithm won't be used;
	// instead we use z-buffer checking per column)
	for i := 0; i < len(sprites); i++ {
		for j := i + 1; j < len(sprites); j++ {
			if sprites[j].dist > sprites[i].dist {
				sprites[i], sprites[j] = sprites[j], sprites[i]
			}
		}
	}

	return sprites
}

// ── Draw ────────────────────────────────────────────────────────────────────
func (d *Doom) Draw(screen *ebiten.Image) {
	// ── Ceiling & Floor ─────────────────────────────────────────────────
	// Draw as two rectangles
	ceilCol := color.RGBA{15, 15, 45, 255}
	floorCol := color.RGBA{50, 50, 50, 255}
	vector.FillRect(screen, 0, 0, dmScreenW, dmScreenH/2, ceilCol, false)
	vector.FillRect(screen, 0, float32(dmScreenH/2), dmScreenW, dmScreenH/2, floorCol, false)

	// Distance-based floor/ceiling shading - draw gradient bands
	for row := 0; row < dmScreenH/2; row++ {
		// Distance from horizon
		distFromCenter := float64(dmScreenH/2-row) / float64(dmScreenH/2)
		if distFromCenter < 0.01 {
			distFromCenter = 0.01
		}
		shade := distFromCenter * 0.7
		if shade > 1 {
			shade = 1
		}
		// Ceiling gradient
		cr := uint8(15.0 * shade)
		cg := uint8(15.0 * shade)
		cb := uint8(45.0 * shade)
		vector.FillRect(screen, 0, float32(row), dmScreenW, 1, color.RGBA{cr, cg, cb, 255}, false)
		// Floor gradient (mirror)
		fr := uint8(50.0 * shade)
		fg := uint8(50.0 * shade)
		fb := uint8(50.0 * shade)
		vector.FillRect(screen, 0, float32(dmScreenH-1-row), dmScreenW, 1, color.RGBA{fr, fg, fb, 255}, false)
	}

	// ── Raycasting walls ────────────────────────────────────────────────
	for x := 0; x < dmScreenW; x++ {
		// Calculate ray direction for this column
		cameraX := 2.0*float64(x)/float64(dmScreenW) - 1.0 // -1 to +1
		rayDirX := d.dirX + (-d.dirY)*math.Tan(dmFOV/2)*cameraX
		rayDirY := d.dirY + d.dirX*math.Tan(dmFOV/2)*cameraX

		result := dmCastRay(d.posX, d.posY, rayDirX, rayDirY)
		d.zBuf[x] = result.dist

		if result.wallType == 0 {
			continue
		}

		// Calculate wall strip height
		lineHeight := int(float64(dmScreenH) / result.dist)
		if lineHeight > dmScreenH*2 {
			lineHeight = dmScreenH * 2
		}

		drawStart := dmScreenH/2 - lineHeight/2
		drawEnd := dmScreenH/2 + lineHeight/2
		if drawStart < 0 {
			drawStart = 0
		}
		if drawEnd > dmScreenH {
			drawEnd = dmScreenH
		}

		col := dmWallColor(result.wallType, result.side, result.dist)
		vector.FillRect(screen, float32(x), float32(drawStart), 1, float32(drawEnd-drawStart), col, false)

		// ── Wall texture patterns ──────────────────────────────────
		texDarken := color.RGBA{0, 0, 0, 35}
		switch result.wallType {
		case 1: // gray stone: horizontal mortar lines
			brickH := lineHeight / 8
			if brickH >= 3 {
				for row := drawStart; row < drawEnd; row++ {
					relY := row - (dmScreenH/2 - lineHeight/2)
					if relY >= 0 && relY%brickH == 0 {
						vector.FillRect(screen, float32(x), float32(row), 1, 1, texDarken, false)
					}
				}
				// Vertical offset mortar for staggered brick pattern
				halfBrick := brickH / 2
				if halfBrick > 0 {
					wallPixel := int(result.wallX * float64(lineHeight))
					if wallPixel%(brickH*2) < brickH {
						for row := drawStart; row < drawEnd; row++ {
							relY := row - (dmScreenH/2 - lineHeight/2)
							if relY >= 0 && relY%brickH == halfBrick {
								vector.FillRect(screen, float32(x), float32(row), 1, 1, texDarken, false)
							}
						}
					}
				}
			}
		case 2: // red brick: horizontal brick lines with mortar
			brickH := lineHeight / 6
			if brickH >= 3 {
				for row := drawStart; row < drawEnd; row++ {
					relY := row - (dmScreenH/2 - lineHeight/2)
					if relY >= 0 && relY%brickH < 1 {
						vector.FillRect(screen, float32(x), float32(row), 1, 1,
							color.RGBA{0, 0, 0, 50}, false)
					}
				}
			}
		case 3: // blue steel: vertical rivet lines
			wallPixel := int(result.wallX * 16)
			if wallPixel%4 == 0 {
				for row := drawStart; row < drawEnd; row++ {
					relY := row - (dmScreenH/2 - lineHeight/2)
					if relY >= 0 && (relY*8/lineHeight)%2 == 0 {
						vector.FillRect(screen, float32(x), float32(row), 1, 1, texDarken, false)
					}
				}
			}
		case 4: // green moss: vertical vine lines
			wallPixel := int(result.wallX * 12)
			if wallPixel%3 == 0 {
				for row := drawStart; row < drawEnd; row++ {
					vector.FillRect(screen, float32(x), float32(row), 1, 1,
						color.RGBA{0, 40, 0, 30}, false)
				}
			}
		}
	}

	// ── Sprites ─────────────────────────────────────────────────────────
	sprites := d.buildSprites()
	for _, sp := range sprites {
		if sp.screenX+sp.halfW < 0 || sp.screenX-sp.halfW >= dmScreenW {
			continue
		}

		// Check at least center column passes z-buffer
		centerVisible := false
		for cx := sp.screenX - sp.halfW; cx < sp.screenX+sp.halfW; cx++ {
			if cx >= 0 && cx < dmScreenW && sp.dist < d.zBuf[cx] {
				centerVisible = true
				break
			}
		}
		if !centerVisible {
			continue
		}

		if sp.isDying {
			// Death splatter animation
			dmDrawDeathSplatter(screen, d, sp)
		} else if sp.isPickup {
			dmDrawPickupSprite(screen, d, sp)
		} else {
			switch sp.eType {
			case dmEnemyImp:
				dmDrawImpSprite(screen, d, sp)
			case dmEnemySoldier:
				dmDrawSoldierSprite(screen, d, sp)
			case dmEnemyDemon:
				dmDrawDemonSprite(screen, d, sp)
			}
		}
	}

	// ── Blood particles ──────────────────────────────────────────────
	d.drawBloodParticles(screen)

	// ── Muzzle flash ────────────────────────────────────────────────────
	if d.muzzleTick > 0 {
		alpha := uint8(float64(d.muzzleTick) / float64(dmMuzzleTicks) * 100)
		vector.FillRect(screen, 0, 0, dmScreenW, dmScreenH,
			color.RGBA{255, 255, 200, alpha}, false)
	}

	// ── Hurt overlay + directional damage indicator ─────────────────────
	if d.hurtTick > 0 {
		alpha := uint8(float64(d.hurtTick) / float64(dmHurtTicks) * 80)
		vector.FillRect(screen, 0, 0, dmScreenW, dmScreenH,
			color.RGBA{255, 0, 0, alpha}, false)
		// Draw directional arrow showing where damage came from
		indAlpha := uint8(float64(d.hurtTick) / float64(dmHurtTicks) * 200)
		cx, cy := float32(dmScreenW/2), float32(dmScreenH/2)
		r := float32(70)
		ax := cx + r*float32(math.Cos(d.hurtDir))
		ay := cy + r*float32(math.Sin(d.hurtDir))
		vector.FillCircle(screen, ax, ay, 8, color.RGBA{255, 40, 40, indAlpha}, true)
		vector.StrokeLine(screen, cx+float32(math.Cos(d.hurtDir))*40, cy+float32(math.Sin(d.hurtDir))*40,
			ax, ay, 3, color.RGBA{255, 80, 80, indAlpha}, true)
	}

	// ── Vignette effect (darken screen edges) ──────────────────────────
	vignetteStrips := 20
	for i := 0; i < vignetteStrips; i++ {
		// How far from edge (0 = edge, 1 = center)
		frac := float64(i) / float64(vignetteStrips)
		alpha := uint8((1.0 - frac) * 60)
		// Top
		vector.FillRect(screen, 0, float32(i), dmScreenW, 1, color.RGBA{0, 0, 0, alpha}, false)
		// Bottom
		vector.FillRect(screen, 0, float32(dmScreenH-1-i), dmScreenW, 1, color.RGBA{0, 0, 0, alpha}, false)
		// Left
		vector.FillRect(screen, float32(i), 0, 1, dmScreenH, color.RGBA{0, 0, 0, alpha}, false)
		// Right
		vector.FillRect(screen, float32(dmScreenW-1-i), 0, 1, dmScreenH, color.RGBA{0, 0, 0, alpha}, false)
	}

	// ── HUD ─────────────────────────────────────────────────────────────
	d.drawHUD(screen)

	// ── Game Over Screen ────────────────────────────────────────────────
	if d.gameOver {
		alpha := uint8(dmClamp(float64(d.gameOverT)*3, 0, 180))
		vector.FillRect(screen, 0, 0, dmScreenW, dmScreenH,
			color.RGBA{80, 0, 0, alpha}, false)

		if d.gameOverT > 30 {
			ebitenutil.DebugPrintAt(screen, "YOU DIED", dmScreenW/2-30, dmScreenH/2-20)
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SCORE: %d", d.score), dmScreenW/2-35, dmScreenH/2)
			ebitenutil.DebugPrintAt(screen, "Press R to restart", dmScreenW/2-55, dmScreenH/2+20)
		}
	}
}

// ── HUD Drawing ─────────────────────────────────────────────────────────────
func (d *Doom) drawHUD(screen *ebiten.Image) {
	// ── Score (top center) ──────────────────────────────────────────────
	scoreStr := fmt.Sprintf("SCORE: %d", d.score)
	ebitenutil.DebugPrintAt(screen, scoreStr, dmScreenW/2-len(scoreStr)*3, 4)

	// ── Health bar (bottom left) ────────────────────────────────────────
	hpBarX := float32(10)
	hpBarY := float32(dmScreenH - 30)
	hpBarW := float32(120)
	hpBarH := float32(16)

	// Background
	vector.FillRect(screen, hpBarX-1, hpBarY-1, hpBarW+2, hpBarH+2,
		color.RGBA{40, 40, 40, 200}, false)
	// HP fill
	hpFrac := float32(d.hp) / float32(dmMaxHP)
	hpCol := color.RGBA{0, 200, 0, 255}
	if hpFrac < 0.5 {
		hpCol = color.RGBA{200, 200, 0, 255}
	}
	if hpFrac < 0.25 {
		hpCol = color.RGBA{200, 0, 0, 255}
	}
	vector.FillRect(screen, hpBarX, hpBarY, hpBarW*hpFrac, hpBarH, hpCol, false)
	vector.StrokeRect(screen, hpBarX, hpBarY, hpBarW, hpBarH, 1,
		color.RGBA{200, 200, 200, 200}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HP: %d", d.hp), int(hpBarX)+2, int(hpBarY)+2)

	// ── Ammo counter (bottom right) ─────────────────────────────────────
	ammoStr := fmt.Sprintf("AMMO: %d", d.ammo)
	ammoX := dmScreenW - 90
	ammoY := dmScreenH - 30
	vector.FillRect(screen, float32(ammoX-4), float32(ammoY-2), 85, 20,
		color.RGBA{40, 40, 40, 200}, false)
	ebitenutil.DebugPrintAt(screen, ammoStr, ammoX, ammoY+2)

	// ── Crosshair ───────────────────────────────────────────────────────
	cx := float32(dmScreenW / 2)
	cy := float32(dmScreenH / 2)
	crossCol := color.RGBA{255, 255, 255, 180}
	vector.StrokeLine(screen, cx-8, cy, cx-3, cy, 1.5, crossCol, false)
	vector.StrokeLine(screen, cx+3, cy, cx+8, cy, 1.5, crossCol, false)
	vector.StrokeLine(screen, cx, cy-8, cx, cy-3, 1.5, crossCol, false)
	vector.StrokeLine(screen, cx, cy+3, cx, cy+8, 1.5, crossCol, false)

	// ── Gun graphic (bottom center) ─────────────────────────────────────
	gunX := float32(dmScreenW/2 - 24)
	gunY := float32(dmScreenH - 75)

	// Gun recoil when shooting
	if d.muzzleTick > 0 {
		gunY += float32(d.muzzleTick) * 2
	}

	// Gun shadow on ground
	vector.FillRect(screen, gunX+6, gunY+62, 36, 8, color.RGBA{20, 20, 20, 80}, false)

	// Main receiver body (dark steel)
	vector.FillRect(screen, gunX+4, gunY+6, 40, 38, color.RGBA{55, 55, 65, 255}, false)
	vector.FillRect(screen, gunX+6, gunY+8, 36, 34, color.RGBA{70, 70, 82, 255}, false)
	// Highlight strip on receiver
	vector.FillRect(screen, gunX+6, gunY+8, 36, 2, color.RGBA{95, 95, 110, 255}, false)

	// Barrel (long, thin)
	vector.FillRect(screen, gunX+16, gunY-24, 16, 32, color.RGBA{45, 45, 55, 255}, false)
	vector.StrokeRect(screen, gunX+16, gunY-24, 16, 32, 1, color.RGBA{80, 80, 95, 255}, false)
	// Barrel bore (dark hole at top)
	vector.FillRect(screen, gunX+20, gunY-24, 8, 3, color.RGBA{15, 15, 20, 255}, false)
	// Barrel bands/rings
	vector.FillRect(screen, gunX+16, gunY-10, 16, 2, color.RGBA{85, 85, 100, 255}, false)
	vector.FillRect(screen, gunX+16, gunY-2, 16, 2, color.RGBA{85, 85, 100, 255}, false)

	// Front sight (small post on top of barrel)
	vector.FillRect(screen, gunX+22, gunY-28, 4, 5, color.RGBA{60, 60, 70, 255}, false)
	vector.FillRect(screen, gunX+23, gunY-28, 2, 2, color.RGBA{255, 100, 50, 255}, false) // red dot

	// Rear sight
	vector.FillRect(screen, gunX+18, gunY+2, 4, 5, color.RGBA{60, 60, 70, 255}, false)
	vector.FillRect(screen, gunX+26, gunY+2, 4, 5, color.RGBA{60, 60, 70, 255}, false)

	// Trigger guard
	vector.FillRect(screen, gunX+14, gunY+38, 20, 3, color.RGBA{50, 50, 60, 255}, false)
	vector.FillRect(screen, gunX+14, gunY+38, 2, 12, color.RGBA{50, 50, 60, 255}, false)
	vector.FillRect(screen, gunX+32, gunY+38, 2, 12, color.RGBA{50, 50, 60, 255}, false)
	// Trigger
	vector.FillRect(screen, gunX+22, gunY+40, 4, 8, color.RGBA{40, 40, 50, 255}, false)

	// Grip (wood texture)
	vector.FillRect(screen, gunX+10, gunY+44, 28, 28, color.RGBA{100, 65, 35, 255}, false)
	vector.StrokeRect(screen, gunX+10, gunY+44, 28, 28, 1, color.RGBA{75, 50, 25, 255}, false)
	// Grip texture lines
	for gy := 0; gy < 5; gy++ {
		ly := gunY + 48 + float32(gy)*5
		vector.FillRect(screen, gunX+12, ly, 24, 1, color.RGBA{85, 55, 28, 255}, false)
	}
	// Grip checkering (cross-hatch pattern via dots)
	for gy := 0; gy < 4; gy++ {
		for gx := 0; gx < 4; gx++ {
			dotX := gunX + 14 + float32(gx)*6
			dotY := gunY + 50 + float32(gy)*5
			vector.FillRect(screen, dotX, dotY, 2, 2, color.RGBA{110, 72, 40, 255}, false)
		}
	}

	// Magazine base plate (bottom of grip)
	vector.FillRect(screen, gunX+12, gunY+70, 24, 4, color.RGBA{60, 60, 70, 255}, false)

	// Ejection port (side detail)
	vector.FillRect(screen, gunX+30, gunY+14, 10, 8, color.RGBA{40, 40, 50, 255}, false)
	vector.StrokeRect(screen, gunX+30, gunY+14, 10, 8, 1, color.RGBA{90, 90, 100, 255}, false)

	// Hammer
	vector.FillRect(screen, gunX+2, gunY+12, 5, 10, color.RGBA{60, 60, 70, 255}, false)

	// Slide serrations (lines on side)
	for i := 0; i < 5; i++ {
		sx := gunX + 6 + float32(i)*4
		vector.FillRect(screen, sx, gunY+28, 1, 10, color.RGBA{55, 55, 65, 255}, false)
	}

	// Muzzle flash from barrel
	if d.muzzleTick > 2 {
		// Multi-layer flash
		vector.FillRect(screen, gunX+10, gunY-42, 28, 20, color.RGBA{255, 200, 50, 150}, false)
		vector.FillRect(screen, gunX+14, gunY-38, 20, 14, color.RGBA{255, 255, 100, 200}, false)
		vector.FillRect(screen, gunX+18, gunY-34, 12, 8, color.RGBA{255, 255, 220, 255}, false)
	} else if d.muzzleTick > 0 {
		// Fading flash
		vector.FillRect(screen, gunX+14, gunY-30, 20, 8, color.RGBA{255, 150, 50, 100}, false)
	}

	// ── Minimap (top right) ─────────────────────────────────────────────
	mmBg := color.RGBA{0, 0, 0, 160}
	vector.FillRect(screen, float32(dmMinimapX-1), float32(dmMinimapY-1),
		float32(dmMinimapSize+2), float32(dmMinimapSize+2), mmBg, false)

	cellW := float32(dmMinimapSize) / float32(dmMapW)
	cellH := float32(dmMinimapSize) / float32(dmMapH)

	for my := 0; my < dmMapH; my++ {
		for mx := 0; mx < dmMapW; mx++ {
			if dmWorldMap[my][mx] > 0 {
				var wc color.RGBA
				switch dmWorldMap[my][mx] {
				case 1:
					wc = color.RGBA{120, 120, 120, 255}
				case 2:
					wc = color.RGBA{150, 50, 50, 255}
				case 3:
					wc = color.RGBA{50, 60, 150, 255}
				case 4:
					wc = color.RGBA{50, 120, 50, 255}
				}
				vector.FillRect(screen,
					float32(dmMinimapX)+float32(mx)*cellW,
					float32(dmMinimapY)+float32(my)*cellH,
					cellW, cellH, wc, false)
			}
		}
	}

	// Player on minimap
	px := float32(dmMinimapX) + float32(d.posX)*cellW
	py := float32(dmMinimapY) + float32(d.posY)*cellH
	vector.FillRect(screen, px-1, py-1, 3, 3, color.RGBA{255, 255, 0, 255}, false)

	// Player direction line
	lineLen := float32(6)
	vector.StrokeLine(screen, px, py,
		px+lineLen*float32(d.dirX), py+lineLen*float32(d.dirY),
		1, color.RGBA{255, 255, 0, 200}, false)

	// Enemies on minimap
	for i := range d.enemies {
		e := &d.enemies[i]
		if !e.alive {
			continue
		}
		ex := float32(dmMinimapX) + float32(e.x)*cellW
		ey := float32(dmMinimapY) + float32(e.y)*cellH
		ec := dmEnemyColor(e.eType)
		ec.A = 200
		vector.FillRect(screen, ex-1, ey-1, 2, 2, ec, false)
	}

	// Pickups on minimap
	for i := range d.pickups {
		p := &d.pickups[i]
		if !p.active {
			continue
		}
		ppx := float32(dmMinimapX) + float32(p.x)*cellW
		ppy := float32(dmMinimapY) + float32(p.y)*cellH
		pc := color.RGBA{0, 200, 0, 200}
		if p.pType == dmPickupAmmo {
			pc = color.RGBA{200, 200, 0, 200}
		}
		vector.FillRect(screen, ppx-1, ppy-1, 2, 2, pc, false)
	}

	// Minimap border
	vector.StrokeRect(screen, float32(dmMinimapX-1), float32(dmMinimapY-1),
		float32(dmMinimapSize+2), float32(dmMinimapSize+2), 1,
		color.RGBA{100, 100, 100, 200}, false)

	// ── Message area ────────────────────────────────────────────────────
	if d.msgTimer > 0 && d.msgText != "" {
		alpha := uint8(255)
		if d.msgTimer < 30 {
			alpha = uint8(float64(d.msgTimer) / 30.0 * 255)
		}
		msgX := dmScreenW/2 - len(d.msgText)*3
		msgY := dmScreenH/2 + 50
		vector.FillRect(screen, float32(msgX-4), float32(msgY-2), float32(len(d.msgText)*6+8), 18,
			color.RGBA{0, 0, 0, alpha / 2}, false)
		// We can't easily set alpha on debug text, but darken background works
		ebitenutil.DebugPrintAt(screen, d.msgText, msgX, msgY)
	}

	// ── Controls hint ───────────────────────────────────────────────────
	ebitenutil.DebugPrintAt(screen, "WASD:Move Arrow/Mouse:Look SPACE:Shoot R:Restart ESC:Back", 4, 4)

	// Enemy count
	aliveCount := 0
	for i := range d.enemies {
		if d.enemies[i].alive {
			aliveCount++
		}
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Enemies: %d", aliveCount), 4, dmScreenH-14)

	// FPS
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS: %.0f", ebiten.ActualFPS()), dmScreenW-60, dmScreenH-14)
}

// ── Sprite Drawing Helpers ──────────────────────────────────────────────────

// dmSpritePixel draws a single "pixel" of a sprite if it passes the z-buffer check.
// px,py are screen coordinates; size is the pixel size.
func dmSpritePixel(screen *ebiten.Image, d *Doom, dist float64, px, py, size int, col color.RGBA) {
	if size < 1 {
		size = 1
	}
	for dx := 0; dx < size; dx++ {
		sx := px + dx
		if sx < 0 || sx >= dmScreenW {
			continue
		}
		if dist >= d.zBuf[sx] {
			continue
		}
		for dy := 0; dy < size; dy++ {
			sy := py + dy
			if sy < 0 || sy >= dmScreenH {
				continue
			}
			vector.FillRect(screen, float32(sx), float32(sy), 1, 1, col, false)
		}
	}
}

// dmSpriteRect draws a filled rectangle as part of a sprite with z-buffer checking.
func dmSpriteRect(screen *ebiten.Image, d *Doom, dist float64, px, py, w, h int, col color.RGBA) {
	for dx := 0; dx < w; dx++ {
		sx := px + dx
		if sx < 0 || sx >= dmScreenW {
			continue
		}
		if dist >= d.zBuf[sx] {
			continue
		}
		vector.FillRect(screen, float32(sx), float32(py), 1, float32(h), col, false)
	}
}

// dmSpriteCircle draws a filled circle as part of a sprite with z-buffer checking.
func dmSpriteCircle(screen *ebiten.Image, d *Doom, dist float64, cx, cy, radius int, col color.RGBA) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				sx := cx + dx
				sy := cy + dy
				if sx < 0 || sx >= dmScreenW || sy < 0 || sy >= dmScreenH {
					continue
				}
				if dist >= d.zBuf[sx] {
					continue
				}
				vector.FillRect(screen, float32(sx), float32(sy), 1, 1, col, false)
			}
		}
	}
}

// dmSpriteLine draws a line as part of a sprite with z-buffer checking.
func dmSpriteLine(screen *ebiten.Image, d *Doom, dist float64, x1, y1, x2, y2, thickness int, col color.RGBA) {
	dx := x2 - x1
	dy := y2 - y1
	steps := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))
	if steps == 0 {
		dmSpritePixel(screen, d, dist, x1, y1, thickness, col)
		return
	}
	xInc := float64(dx) / float64(steps)
	yInc := float64(dy) / float64(steps)
	fx := float64(x1)
	fy := float64(y1)
	for i := 0; i <= steps; i++ {
		dmSpritePixel(screen, d, dist, int(fx), int(fy), thickness, col)
		fx += xInc
		fy += yInc
	}
}

// dmSpriteTriangle draws a filled triangle as part of a sprite.
func dmSpriteTriangle(screen *ebiten.Image, d *Doom, dist float64, x1, y1, x2, y2, x3, y3 int, col color.RGBA) {
	// Find bounding box
	minX := x1
	if x2 < minX { minX = x2 }
	if x3 < minX { minX = x3 }
	maxX := x1
	if x2 > maxX { maxX = x2 }
	if x3 > maxX { maxX = x3 }
	minY := y1
	if y2 < minY { minY = y2 }
	if y3 < minY { minY = y3 }
	maxY := y1
	if y2 > maxY { maxY = y2 }
	if y3 > maxY { maxY = y3 }

	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			// Barycentric coordinate check
			d1 := (px-x2)*(y1-y2) - (x1-x2)*(py-y2)
			d2 := (px-x3)*(y2-y3) - (x2-x3)*(py-y3)
			d3 := (px-x1)*(y3-y1) - (x3-x1)*(py-y1)
			hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
			hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)
			if !(hasNeg && hasPos) {
				if px >= 0 && px < dmScreenW && py >= 0 && py < dmScreenH {
					if dist < d.zBuf[px] {
						vector.FillRect(screen, float32(px), float32(py), 1, 1, col, false)
					}
				}
			}
		}
	}
}

// applyBrightness multiplies a color's RGB by brightness factor.
func applyBrightness(r, g, b uint8, brightness float64) (uint8, uint8, uint8) {
	return uint8(float64(r) * brightness),
		uint8(float64(g) * brightness),
		uint8(float64(b) * brightness)
}

// ── Imp Sprite Drawing ─────────────────────────────────────────────────────
func dmDrawImpSprite(screen *ebiten.Image, d *Doom, sp dmSpriteInfo) {
	cx := sp.screenX  // center X
	h := sp.spriteHeight
	top := sp.drawStartY
	br := sp.brightness
	dist := sp.dist

	// Scale factor based on sprite height
	s := float64(h) / 80.0 // normalize: assume 80px is "normal" height

	// Colors
	rr, rg, rb := applyBrightness(200, 40, 30, br)   // body red
	dr, dg, db := applyBrightness(150, 25, 20, br)    // dark red
	hr, hg, hb := applyBrightness(180, 35, 25, br)    // horn color
	er, eg, eb := applyBrightness(255, 255, 50, br)    // eye yellow
	mr, mg, mb := applyBrightness(80, 10, 10, br)      // mouth dark

	bodyCol := color.RGBA{rr, rg, rb, 255}
	darkCol := color.RGBA{dr, dg, db, 255}
	hornCol := color.RGBA{hr, hg, hb, 255}
	eyeCol := color.RGBA{er, eg, eb, 255}
	mouthCol := color.RGBA{mr, mg, mb, 255}

	// Head position: top 25% of sprite
	headCY := top + int(12*s)
	headR := int(math.Max(6*s, 2))

	// Head
	dmSpriteCircle(screen, d, dist, cx, headCY, headR, bodyCol)

	// Horns (two small triangles on head)
	hornH := int(math.Max(6*s, 2))
	hornW := int(math.Max(3*s, 1))
	// Left horn
	dmSpriteTriangle(screen, d, dist,
		cx-headR+1, headCY-headR/2,
		cx-headR-hornW, headCY-headR-hornH,
		cx-headR+hornW+1, headCY-headR/2,
		hornCol)
	// Right horn
	dmSpriteTriangle(screen, d, dist,
		cx+headR-1, headCY-headR/2,
		cx+headR+hornW, headCY-headR-hornH,
		cx+headR-hornW-1, headCY-headR/2,
		hornCol)

	// Eyes (two glowing dots)
	eyeOff := int(math.Max(3*s, 1))
	eyeR := int(math.Max(1.5*s, 1))
	dmSpriteCircle(screen, d, dist, cx-eyeOff, headCY-1, eyeR, eyeCol)
	dmSpriteCircle(screen, d, dist, cx+eyeOff, headCY-1, eyeR, eyeCol)

	// Mouth slit
	mouthW := int(math.Max(4*s, 2))
	mouthY := headCY + int(math.Max(3*s, 1))
	dmSpriteRect(screen, d, dist, cx-mouthW/2, mouthY, mouthW, int(math.Max(2*s, 1)), mouthCol)

	// Neck
	neckY := headCY + headR
	neckW := int(math.Max(4*s, 2))
	neckH := int(math.Max(3*s, 1))
	dmSpriteRect(screen, d, dist, cx-neckW/2, neckY, neckW, neckH, darkCol)

	// Body (trapezoid shape - wider at shoulders)
	bodyTop := neckY + neckH
	bodyH := int(math.Max(22*s, 6))
	shoulderW := int(math.Max(16*s, 4))
	waistW := int(math.Max(10*s, 3))
	for row := 0; row < bodyH; row++ {
		frac := float64(row) / float64(bodyH)
		w := int(float64(shoulderW)*(1-frac) + float64(waistW)*frac)
		py := bodyTop + row
		bc := bodyCol
		if row < 3 {
			bc = darkCol // shoulder darkness
		}
		dmSpriteRect(screen, d, dist, cx-w/2, py, w, 1, bc)
	}

	// Arms reaching out (angled lines from shoulders)
	armLen := int(math.Max(14*s, 4))
	armThick := int(math.Max(2*s, 1))
	// Left arm reaching outward and forward
	dmSpriteLine(screen, d, dist,
		cx-shoulderW/2, bodyTop+2,
		cx-shoulderW/2-armLen, bodyTop+armLen/2,
		armThick, bodyCol)
	// Claw on left arm
	clawLen := int(math.Max(4*s, 1))
	dmSpriteLine(screen, d, dist,
		cx-shoulderW/2-armLen, bodyTop+armLen/2,
		cx-shoulderW/2-armLen-clawLen, bodyTop+armLen/2-clawLen,
		1, hornCol)
	dmSpriteLine(screen, d, dist,
		cx-shoulderW/2-armLen, bodyTop+armLen/2,
		cx-shoulderW/2-armLen-clawLen, bodyTop+armLen/2+clawLen/2,
		1, hornCol)

	// Right arm reaching outward and forward
	dmSpriteLine(screen, d, dist,
		cx+shoulderW/2, bodyTop+2,
		cx+shoulderW/2+armLen, bodyTop+armLen/2,
		armThick, bodyCol)
	// Claw on right arm
	dmSpriteLine(screen, d, dist,
		cx+shoulderW/2+armLen, bodyTop+armLen/2,
		cx+shoulderW/2+armLen+clawLen, bodyTop+armLen/2-clawLen,
		1, hornCol)
	dmSpriteLine(screen, d, dist,
		cx+shoulderW/2+armLen, bodyTop+armLen/2,
		cx+shoulderW/2+armLen+clawLen, bodyTop+armLen/2+clawLen/2,
		1, hornCol)

	// Legs
	legTop := bodyTop + bodyH
	legH := int(math.Max(18*s, 4))
	legW := int(math.Max(4*s, 2))
	legGap := int(math.Max(3*s, 1))
	// Left leg
	dmSpriteRect(screen, d, dist, cx-legGap-legW, legTop, legW, legH, darkCol)
	// Right leg
	dmSpriteRect(screen, d, dist, cx+legGap, legTop, legW, legH, darkCol)
	// Feet
	feetW := int(math.Max(5*s, 2))
	feetH := int(math.Max(3*s, 1))
	dmSpriteRect(screen, d, dist, cx-legGap-feetW, legTop+legH, feetW, feetH, darkCol)
	dmSpriteRect(screen, d, dist, cx+legGap, legTop+legH, feetW, feetH, darkCol)
}

// ── Soldier Sprite Drawing ─────────────────────────────────────────────────
func dmDrawSoldierSprite(screen *ebiten.Image, d *Doom, sp dmSpriteInfo) {
	cx := sp.screenX
	h := sp.spriteHeight
	top := sp.drawStartY
	br := sp.brightness
	dist := sp.dist

	s := float64(h) / 80.0

	// Colors
	gr, gg, gb := applyBrightness(50, 140, 50, br)     // green uniform
	ar, ag, ab := applyBrightness(70, 160, 70, br)      // armor plate (lighter green)
	hr, hg, hb := applyBrightness(60, 100, 60, br)      // helmet (dark green)
	vr, vg, vb := applyBrightness(20, 20, 20, br)       // visor
	sr, sg, sb := applyBrightness(120, 100, 80, br)      // skin
	wr, wg, wb := applyBrightness(50, 50, 55, br)       // weapon metal
	br2, bg2, bb := applyBrightness(80, 55, 35, br)     // boots

	uniformCol := color.RGBA{gr, gg, gb, 255}
	armorCol := color.RGBA{ar, ag, ab, 255}
	helmetCol := color.RGBA{hr, hg, hb, 255}
	visorCol := color.RGBA{vr, vg, vb, 255}
	skinCol := color.RGBA{sr, sg, sb, 255}
	weaponCol := color.RGBA{wr, wg, wb, 255}
	bootCol := color.RGBA{br2, bg2, bb, 255}

	// Helmet (half-circle on top)
	headCY := top + int(12*s)
	headR := int(math.Max(7*s, 2))
	dmSpriteCircle(screen, d, dist, cx, headCY, headR, helmetCol)
	// Flatten bottom of helmet for face area
	faceH := int(math.Max(4*s, 2))
	faceW := int(math.Max(10*s, 3))
	dmSpriteRect(screen, d, dist, cx-faceW/2, headCY, faceW, faceH, skinCol)

	// Helmet rim
	rimW := int(math.Max(16*s, 4))
	rimH := int(math.Max(2*s, 1))
	dmSpriteRect(screen, d, dist, cx-rimW/2, headCY-2, rimW, rimH, helmetCol)

	// Visor slit (dark horizontal line across helmet)
	visorW := int(math.Max(10*s, 3))
	visorY := headCY - int(math.Max(1*s, 1))
	dmSpriteRect(screen, d, dist, cx-visorW/2, visorY, visorW, int(math.Max(2*s, 1)), visorCol)

	// Neck
	neckY := headCY + faceH
	neckW := int(math.Max(4*s, 2))
	neckH := int(math.Max(2*s, 1))
	dmSpriteRect(screen, d, dist, cx-neckW/2, neckY, neckW, neckH, skinCol)

	// Body with armor plate
	bodyTop := neckY + neckH
	bodyH := int(math.Max(22*s, 6))
	bodyW := int(math.Max(16*s, 4))
	dmSpriteRect(screen, d, dist, cx-bodyW/2, bodyTop, bodyW, bodyH, uniformCol)

	// Armor chest plate (lighter rectangle on chest)
	plateW := int(math.Max(10*s, 3))
	plateH := int(math.Max(12*s, 3))
	dmSpriteRect(screen, d, dist, cx-plateW/2, bodyTop+2, plateW, plateH, armorCol)

	// Belt
	beltY := bodyTop + bodyH - int(math.Max(3*s, 1))
	beltW := int(math.Max(18*s, 4))
	beltH := int(math.Max(3*s, 1))
	dmSpriteRect(screen, d, dist, cx-beltW/2, beltY, beltW, beltH, bootCol)
	// Belt buckle
	buckleW := int(math.Max(4*s, 1))
	dmSpriteRect(screen, d, dist, cx-buckleW/2, beltY, buckleW, beltH, color.RGBA{
		uint8(float64(180) * br), uint8(float64(160) * br), uint8(float64(50) * br), 255,
	})

	// Arms holding weapon
	armThick := int(math.Max(3*s, 1))
	armY := bodyTop + int(6*s)
	// Left arm extends out to hold weapon
	dmSpriteRect(screen, d, dist, cx-bodyW/2-int(8*s), armY, int(8*s), armThick, uniformCol)
	// Right arm extends out to hold weapon
	dmSpriteRect(screen, d, dist, cx+bodyW/2, armY, int(8*s), armThick, uniformCol)
	// Hands
	handR := int(math.Max(2*s, 1))
	dmSpriteCircle(screen, d, dist, cx-bodyW/2-int(8*s), armY+armThick/2, handR, skinCol)
	dmSpriteCircle(screen, d, dist, cx+bodyW/2+int(8*s), armY+armThick/2, handR, skinCol)

	// Weapon (horizontal rifle across arms)
	gunLen := int(math.Max(30*s, 6))
	gunH := int(math.Max(3*s, 1))
	gunY := armY - 1
	dmSpriteRect(screen, d, dist, cx-gunLen/2, gunY, gunLen, gunH, weaponCol)
	// Rifle stock
	stockW := int(math.Max(6*s, 2))
	stockH := int(math.Max(5*s, 2))
	dmSpriteRect(screen, d, dist, cx+gunLen/2-stockW, gunY, stockW, stockH, bootCol)
	// Rifle barrel tip
	tipW := int(math.Max(4*s, 1))
	dmSpriteRect(screen, d, dist, cx-gunLen/2-tipW, gunY, tipW, gunH-1, weaponCol)

	// Legs in stance (slightly apart)
	legTop := bodyTop + bodyH
	legH := int(math.Max(18*s, 4))
	legW := int(math.Max(5*s, 2))
	legGap := int(math.Max(2*s, 1))
	// Left leg
	dmSpriteRect(screen, d, dist, cx-legGap-legW, legTop, legW, legH, uniformCol)
	// Right leg (slightly forward)
	dmSpriteRect(screen, d, dist, cx+legGap, legTop, legW, legH, uniformCol)

	// Boots
	bootH := int(math.Max(4*s, 1))
	bootW := int(math.Max(6*s, 2))
	dmSpriteRect(screen, d, dist, cx-legGap-bootW, legTop+legH-bootH, bootW, bootH, bootCol)
	dmSpriteRect(screen, d, dist, cx+legGap, legTop+legH-bootH, bootW, bootH, bootCol)
}

// ── Demon Sprite Drawing ───────────────────────────────────────────────────
func dmDrawDemonSprite(screen *ebiten.Image, d *Doom, sp dmSpriteInfo) {
	cx := sp.screenX
	h := sp.spriteHeight
	top := sp.drawStartY
	br := sp.brightness
	dist := sp.dist

	s := float64(h) / 80.0

	// Colors
	pr, pg, pb := applyBrightness(160, 50, 190, br)    // purple body
	dr, dg, db := applyBrightness(120, 30, 150, br)    // dark purple
	er, eg, eb := applyBrightness(255, 80, 80, br)     // red eyes
	mr, mg, mb := applyBrightness(40, 10, 50, br)      // dark mouth
	spr, spg, spb := applyBrightness(140, 40, 170, br) // spike color

	bodyCol := color.RGBA{pr, pg, pb, 255}
	darkCol := color.RGBA{dr, dg, db, 255}
	eyeCol := color.RGBA{er, eg, eb, 255}
	mouthCol := color.RGBA{mr, mg, mb, 255}
	spikeCol := color.RGBA{spr, spg, spb, 255}

	// Head (wider, more bestial)
	headCY := top + int(14*s)
	headRX := int(math.Max(9*s, 3))
	headRY := int(math.Max(7*s, 2))
	// Elongated head (ellipse approximation)
	for dy := -headRY; dy <= headRY; dy++ {
		for dx := -headRX; dx <= headRX; dx++ {
			fx := float64(dx) / float64(headRX)
			fy := float64(dy) / float64(headRY)
			if fx*fx+fy*fy <= 1.0 {
				px := cx + dx
				py := headCY + dy
				if px >= 0 && px < dmScreenW && py >= 0 && py < dmScreenH && dist < d.zBuf[px] {
					vector.FillRect(screen, float32(px), float32(py), 1, 1, bodyCol, false)
				}
			}
		}
	}

	// Big red eyes
	eyeOff := int(math.Max(4*s, 1))
	eyeR := int(math.Max(2.5*s, 1))
	dmSpriteCircle(screen, d, dist, cx-eyeOff, headCY-int(2*s), eyeR, eyeCol)
	dmSpriteCircle(screen, d, dist, cx+eyeOff, headCY-int(2*s), eyeR, eyeCol)
	// Eye pupils
	pupR := int(math.Max(1*s, 1))
	dmSpriteCircle(screen, d, dist, cx-eyeOff, headCY-int(2*s), pupR, color.RGBA{mr, mg, mb, 255})
	dmSpriteCircle(screen, d, dist, cx+eyeOff, headCY-int(2*s), pupR, color.RGBA{mr, mg, mb, 255})

	// Big jaw/mouth (dark opening)
	jawW := int(math.Max(12*s, 3))
	jawH := int(math.Max(5*s, 2))
	jawY := headCY + int(2*s)
	dmSpriteRect(screen, d, dist, cx-jawW/2, jawY, jawW, jawH, mouthCol)
	// Teeth (small white triangles in mouth)
	teethCount := int(math.Max(4*s, 2))
	teethW := jawW / teethCount
	if teethW < 1 { teethW = 1 }
	for t := 0; t < teethCount; t++ {
		tx := cx - jawW/2 + t*teethW + teethW/2
		ty := jawY
		th := int(math.Max(2*s, 1))
		tr, tg, tb := applyBrightness(220, 220, 200, br)
		dmSpriteTriangle(screen, d, dist,
			tx-teethW/2, ty, tx+teethW/2, ty, tx, ty+th,
			color.RGBA{tr, tg, tb, 255})
	}

	// Massive body (wide shoulders, slightly narrower waist)
	bodyTop := headCY + headRY + int(2*s)
	bodyH := int(math.Max(28*s, 8))
	shoulderW := int(math.Max(26*s, 6))
	waistW := int(math.Max(18*s, 4))
	for row := 0; row < bodyH; row++ {
		frac := float64(row) / float64(bodyH)
		w := int(float64(shoulderW)*(1-frac*0.5) + float64(waistW)*frac*0.5)
		py := bodyTop + row
		bc := bodyCol
		if row%int(math.Max(4*s, 2)) < int(math.Max(1*s, 1)) {
			bc = darkCol // muscle striping
		}
		dmSpriteRect(screen, d, dist, cx-w/2, py, w, 1, bc)
	}

	// Spikes on shoulders/back
	spikeH := int(math.Max(10*s, 3))
	spikeW := int(math.Max(3*s, 1))
	// Left shoulder spikes
	dmSpriteTriangle(screen, d, dist,
		cx-shoulderW/2-2, bodyTop+2,
		cx-shoulderW/2+spikeW, bodyTop+2,
		cx-shoulderW/2, bodyTop-spikeH,
		spikeCol)
	dmSpriteTriangle(screen, d, dist,
		cx-shoulderW/2+3, bodyTop,
		cx-shoulderW/2+3+spikeW, bodyTop,
		cx-shoulderW/2+4, bodyTop-spikeH+3,
		spikeCol)
	// Right shoulder spikes
	dmSpriteTriangle(screen, d, dist,
		cx+shoulderW/2-spikeW, bodyTop+2,
		cx+shoulderW/2+2, bodyTop+2,
		cx+shoulderW/2, bodyTop-spikeH,
		spikeCol)
	dmSpriteTriangle(screen, d, dist,
		cx+shoulderW/2-3-spikeW, bodyTop,
		cx+shoulderW/2-3, bodyTop,
		cx+shoulderW/2-4, bodyTop-spikeH+3,
		spikeCol)

	// Thick arms
	armLen := int(math.Max(12*s, 3))
	armThick := int(math.Max(5*s, 2))
	armY := bodyTop + int(4*s)
	// Left arm
	dmSpriteRect(screen, d, dist, cx-shoulderW/2-armLen, armY, armLen, armThick, bodyCol)
	// Fist
	fistR := int(math.Max(3*s, 1))
	dmSpriteCircle(screen, d, dist, cx-shoulderW/2-armLen, armY+armThick/2, fistR, darkCol)
	// Right arm
	dmSpriteRect(screen, d, dist, cx+shoulderW/2, armY, armLen, armThick, bodyCol)
	dmSpriteCircle(screen, d, dist, cx+shoulderW/2+armLen, armY+armThick/2, fistR, darkCol)

	// Thick legs
	legTop := bodyTop + bodyH
	legH := int(math.Max(16*s, 4))
	legW := int(math.Max(7*s, 2))
	legGap := int(math.Max(2*s, 1))
	dmSpriteRect(screen, d, dist, cx-legGap-legW, legTop, legW, legH, darkCol)
	dmSpriteRect(screen, d, dist, cx+legGap, legTop, legW, legH, darkCol)
	// Hooves/feet
	footW := int(math.Max(9*s, 2))
	footH := int(math.Max(4*s, 1))
	dmSpriteRect(screen, d, dist, cx-legGap-footW, legTop+legH, footW, footH, darkCol)
	dmSpriteRect(screen, d, dist, cx+legGap, legTop+legH, footW, footH, darkCol)

	// Tail (line extending from lower back)
	tailStartY := bodyTop + bodyH - int(4*s)
	tailEndY := bodyTop + bodyH + int(6*s)
	tailEndX := cx + shoulderW/2 + int(12*s)
	tailThick := int(math.Max(2*s, 1))
	dmSpriteLine(screen, d, dist, cx+waistW/2, tailStartY, tailEndX, tailEndY, tailThick, darkCol)
	// Tail tip (small triangle)
	tipSize := int(math.Max(3*s, 1))
	dmSpriteTriangle(screen, d, dist,
		tailEndX, tailEndY-tipSize,
		tailEndX+tipSize*2, tailEndY,
		tailEndX, tailEndY+tipSize,
		spikeCol)
}

// ── Death Splatter Drawing ─────────────────────────────────────────────────
func dmDrawDeathSplatter(screen *ebiten.Image, d *Doom, sp dmSpriteInfo) {
	cx := sp.screenX
	h := sp.spriteHeight
	midY := (sp.drawStartY + sp.drawEndY) / 2
	dist := sp.dist
	br := sp.brightness

	// Death animation progress (30 -> 0)
	progress := 1.0 - float64(sp.deathTick)/30.0 // 0 -> 1

	s := float64(h) / 80.0

	// Flash white at start, then red splatter
	if progress < 0.2 {
		// Bright flash
		flashBr := 1.0 - progress*5
		fr := uint8(dmClamp(255*flashBr*br, 0, 255))
		fg := uint8(dmClamp(255*flashBr*br, 0, 255))
		fb := uint8(dmClamp(255*flashBr*br, 0, 255))
		flashCol := color.RGBA{fr, fg, fb, 255}
		flashR := int(math.Max(12*s, 3))
		dmSpriteCircle(screen, d, dist, cx, midY, flashR, flashCol)
	} else {
		// Red blood splatter expanding outward
		splatCount := int(8 * s)
		if splatCount < 4 { splatCount = 4 }
		for i := 0; i < splatCount; i++ {
			angle := float64(i) * (2 * math.Pi / float64(splatCount))
			radius := progress * 20 * s
			px := cx + int(math.Cos(angle)*radius)
			py := midY + int(math.Sin(angle)*radius*0.7) // squished vertically
			splatR := int(math.Max(3*s*(1-progress*0.5), 1))

			alpha := uint8(dmClamp(255*(1-progress)*br, 0, 255))
			rr := uint8(dmClamp(200*br, 0, 255))
			dmSpriteCircle(screen, d, dist, px, py, splatR,
				color.RGBA{rr, 0, 0, alpha})
		}
		// Central blood pool
		poolR := int(math.Max(6*s*progress, 1))
		poolAlpha := uint8(dmClamp(200*(1-progress)*br, 0, 255))
		rr := uint8(dmClamp(150*br, 0, 255))
		dmSpriteCircle(screen, d, dist, cx, midY, poolR,
			color.RGBA{rr, 0, 0, poolAlpha})
	}
}

// ── Pickup Sprite Drawing ──────────────────────────────────────────────────
func dmDrawPickupSprite(screen *ebiten.Image, d *Doom, sp dmSpriteInfo) {
	cx := sp.screenX
	h := sp.spriteHeight
	midY := (sp.drawStartY + sp.drawEndY) / 2
	dist := sp.dist
	br := sp.brightness

	s := float64(h) / 30.0 // pickups are smaller

	switch sp.pickupType {
	case dmPickupHealth:
		// Draw a white cross/plus sign on green background
		bgR := int(math.Max(8*s, 2))
		// Green circle background
		gr, gg, gb := applyBrightness(0, 180, 0, br)
		dmSpriteCircle(screen, d, dist, cx, midY, bgR, color.RGBA{gr, gg, gb, 255})
		// White cross
		crossW := int(math.Max(3*s, 1))
		crossH := int(math.Max(10*s, 3))
		wr, wg, wb := applyBrightness(255, 255, 255, br)
		crossCol := color.RGBA{wr, wg, wb, 255}
		// Vertical bar
		dmSpriteRect(screen, d, dist, cx-crossW/2, midY-crossH/2, crossW, crossH, crossCol)
		// Horizontal bar
		dmSpriteRect(screen, d, dist, cx-crossH/2, midY-crossW/2, crossH, crossW, crossCol)

	case dmPickupAmmo:
		// Draw bullet shapes
		bgR := int(math.Max(8*s, 2))
		// Dark background circle
		yr, yg, yb := applyBrightness(60, 60, 40, br)
		dmSpriteCircle(screen, d, dist, cx, midY, bgR, color.RGBA{yr, yg, yb, 255})

		// Three bullet shapes
		bulletW := int(math.Max(2*s, 1))
		bulletH := int(math.Max(6*s, 2))
		tipH := int(math.Max(2*s, 1))
		for b := -1; b <= 1; b++ {
			bx := cx + int(float64(b)*3*s)
			by := midY - bulletH/2
			// Bullet casing (brass/gold)
			br2, bg2, bb := applyBrightness(200, 170, 50, br)
			dmSpriteRect(screen, d, dist, bx-bulletW/2, by+tipH, bulletW, bulletH-tipH, color.RGBA{br2, bg2, bb, 255})
			// Bullet tip (copper/darker)
			tr, tg, tb := applyBrightness(180, 120, 60, br)
			dmSpriteTriangle(screen, d, dist,
				bx-bulletW/2, by+tipH,
				bx+bulletW/2, by+tipH,
				bx, by,
				color.RGBA{tr, tg, tb, 255})
		}
	}
}

// ── Blood Particle Drawing ─────────────────────────────────────────────────
func (d *Doom) drawBloodParticles(screen *ebiten.Image) {
	planeX := -d.dirY * math.Tan(dmFOV/2)
	planeY := d.dirX * math.Tan(dmFOV/2)

	for i := range d.bloodParticles {
		bp := &d.bloodParticles[i]
		dx := bp.x - d.posX
		dy := bp.y - d.posY

		invDet := 1.0 / (planeX*d.dirY - d.dirX*planeY)
		transformX := invDet * (d.dirY*dx - d.dirX*dy)
		transformY := invDet * (-planeY*dx + planeX*dy)

		if transformY <= 0.1 {
			continue
		}

		screenX := int(float64(dmScreenW)/2.0 * (1.0 + transformX/transformY))
		screenY := dmScreenH/2 + int(5.0/transformY) // slightly above center

		if screenX < 0 || screenX >= dmScreenW || screenY < 0 || screenY >= dmScreenH {
			continue
		}

		if transformY >= d.zBuf[screenX] {
			continue
		}

		lifeFrac := float64(bp.life) / float64(bp.maxLife)
		alpha := uint8(dmClamp(lifeFrac*255, 0, 255))
		size := int(math.Max(2.0/transformY, 1))
		if size > 4 {
			size = 4
		}

		col := color.RGBA{
			uint8(dmClamp(180+lifeFrac*75, 0, 255)),
			0, 0, alpha,
		}
		vector.FillRect(screen, float32(screenX), float32(screenY), float32(size), float32(size), col, false)
	}
}

func (d *Doom) Layout(w, h int) (int, int) {
	return dmScreenW, dmScreenH
}

func (d *Doom) WantsBack() bool {
	return d.goBack
}

