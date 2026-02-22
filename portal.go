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

// ── Screen & tile constants ─────────────────────────────────────────────────
const (
	ptScreenW = 480
	ptScreenH = 360
	ptTileS   = 24
	ptGridW   = 20 // tiles across
	ptGridH   = 15 // tiles down
)

// ── Physics constants ───────────────────────────────────────────────────────
const (
	ptGravity   = 0.4
	ptJumpVel   = -7.0
	ptMoveSpeed = 3.0
	ptFriction  = 0.8
	ptMaxFallV  = 12.0
)

// ── Tile types ──────────────────────────────────────────────────────────────
const (
	ptAir       = 0
	ptWall      = 1 // portal-able solid
	ptNoPortal  = 2 // solid, no portals
	ptExit      = 3
	ptSpawn     = 4
	ptButton    = 5
	ptDoor      = 6 // opened by button
	ptSpike     = 7
	ptCube      = 8
	ptMovingPlat = 9
)

// ── Portal direction (which face of tile the portal is on) ──────────────────
const (
	ptDirUp    = 0
	ptDirDown  = 1
	ptDirLeft  = 2
	ptDirRight = 3
)

// ── Portal struct ───────────────────────────────────────────────────────────
type ptPortalPair struct {
	bx, by  int // blue tile coords
	bdir    int // blue facing direction
	bactive bool
	ox, oy  int // orange tile coords
	odir    int
	oactive bool
}

// ── Cube entity ─────────────────────────────────────────────────────────────
type ptCubeEntity struct {
	x, y   float64
	vx, vy float64
	w, h   float64
	onGnd  bool
}

// ── Portal (the game) ───────────────────────────────────────────────────────
type Portal struct {
	goBack bool

	level    int
	deaths   int
	ticks    int
	levelWon bool
	winTimer int

	// player state
	px, py   float64 // position (top-left of bounding box)
	pvx, pvy float64 // velocity
	pw, ph   float64 // size
	onGround bool
	dead     bool
	deadTimer int

	// portals
	portals ptPortalPair

	// level data
	grid     [ptGridH][ptGridW]int
	origGrid [ptGridH][ptGridW]int // for restart
	spawnX, spawnY float64

	// cubes
	cubes []ptCubeEntity

	// buttons/doors state
	buttonPressed bool
	doorOpen      bool

	// audio
	bgm        *audio.Player
	sfxPortal  []byte
	sfxTeleport []byte
	sfxJump    []byte
	sfxDeath   []byte
	sfxWin     []byte

	// portal cooldown to avoid re-teleport
	teleportCooldown int
}

// ── Level data ──────────────────────────────────────────────────────────────
// 0=air 1=wall 2=noPortal 3=exit 4=spawn 5=button 6=door 7=spike 8=cube
var ptLevels = [][ptGridH][ptGridW]int{
	// Level 1: Tutorial — simple gap, portal across
	{
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 2},
		{2, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	},
	// Level 2: Vertical — portal from floor up to high platform
	{
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	},
	// Level 3: Momentum — fall from height into floor portal, launch out of wall portal
	// Left: tall shaft, player spawns at top. Portal-able floor at bottom.
	// Right: high platform with exit. Wall between is portal-able (type 1).
	// Solution: place blue portal on floor at bottom of shaft, orange on right-facing
	// wall of the divider. Walk off ledge, fall into blue, launch out orange across gap.
	{
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		{2, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	},
	// Level 4: Button + door
	{
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 3, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 1, 1, 1, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 8, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 4, 0, 0, 0, 0, 0, 0, 5, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	},
	// Level 5: Spikes + precise portal placement
	{
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	},
	// Level 6: Double portal momentum with gap and ceiling
	{
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 2},
		{2, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 7, 7, 7, 7, 7, 7, 7, 7, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	},
	// Level 7: Cube puzzle — push cube onto button to hold door open
	{
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 3, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 1, 1, 1, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 0, 0, 8, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{2, 4, 0, 0, 0, 0, 0, 5, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	},
	// Level 8: Complex — multiple mechanics combined
	{
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 3, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 4, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 8, 0, 0, 5, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 7, 7, 7, 7, 7, 7, 7, 7, 7, 1, 7, 7, 7, 7, 7, 7, 7, 0, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	},
}

// ── Tutorial text per level ─────────────────────────────────────────────────
var ptTutorials = []string{
	"LClick=Blue portal, RClick=Orange. Portal across the gap!",
	"Place portals on floor+wall to reach the high platform.",
	"Fall into a floor portal to launch out a wall portal!",
	"Push the cube onto the button to open the door. Use portals!",
	"Avoid spikes! Place portals on the walls above to fly over.",
	"Use momentum: fall far, portal at bottom, exit from wall.",
	"Push cube onto button, then portal through the door.",
	"Combine everything: cube, button, spikes, momentum!",
}

// ── Audio generation ────────────────────────────────────────────────────────

func ptGenBGM() []byte {
	bpm := 90.0
	// Ambient sci-fi pad — long sine tones with slow modulation
	// C minor ambient: C3, Eb3, G3, Bb3, C4, Eb4 ...
	padNotes := [][2]int{
		{48, 4}, {48, 4}, {51, 4}, {51, 4},
		{55, 4}, {55, 4}, {51, 4}, {51, 4},
		{48, 4}, {48, 4}, {53, 4}, {53, 4},
		{55, 4}, {55, 4}, {58, 4}, {58, 4},
		{60, 4}, {60, 4}, {58, 4}, {58, 4},
		{55, 4}, {55, 4}, {53, 4}, {53, 4},
	}
	pad := generateTrack(padNotes, sineWave, bpm, 0.12)

	// Higher melodic line
	melNotes := [][2]int{
		{72, 2}, {0, 2}, {75, 2}, {0, 2}, {72, 2}, {0, 2}, {67, 2}, {0, 2},
		{63, 2}, {0, 2}, {67, 2}, {0, 2}, {72, 2}, {0, 6},
		{75, 2}, {0, 2}, {72, 2}, {0, 2}, {67, 2}, {0, 2}, {63, 2}, {0, 2},
		{60, 2}, {0, 2}, {63, 2}, {0, 2}, {67, 2}, {0, 6},
		{72, 2}, {0, 2}, {75, 2}, {0, 2}, {79, 2}, {0, 2}, {75, 2}, {0, 2},
		{72, 2}, {0, 2}, {67, 2}, {0, 2}, {63, 2}, {0, 6},
	}
	mel := generateTrack(melNotes, triangleWave, bpm, 0.06)

	return mixBuffers(pad, mel)
}

func ptGenPortalSFX() []byte {
	dur := 0.3
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		frac := t / dur
		// Sci-fi whoosh: sweeping frequency
		freq := 800.0 + 1200.0*(1.0-frac)
		val := sineWave(t, freq)*0.3 + noise()*0.05
		env := (1.0 - frac) * (1.0 - frac)
		writeSample(buf, i*4, val*env*0.5)
	}
	return buf
}

func ptGenTeleportSFX() []byte {
	dur := 0.35
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		frac := t / dur
		// Spatial warp: dual oscillating frequencies
		f1 := 200.0 + 600.0*math.Sin(frac*math.Pi*4)
		f2 := 400.0 + 300.0*math.Cos(frac*math.Pi*6)
		val := sineWave(t, f1)*0.25 + triangleWave(t, f2)*0.15
		env := math.Sin(frac * math.Pi)
		writeSample(buf, i*4, val*env*0.5)
	}
	return buf
}

func ptGenJumpSFX() []byte {
	dur := 0.12
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		frac := t / dur
		freq := 150.0 + 100.0*frac
		val := sineWave(t, freq) * (1.0 - frac)
		writeSample(buf, i*4, val*0.3)
	}
	return buf
}

func ptGenDeathSFX() []byte {
	dur := 0.4
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		frac := t / dur
		val := squareWave(t, 80.0)*0.3 + noise()*0.2
		env := 1.0 - frac
		writeSample(buf, i*4, val*env*0.4)
	}
	return buf
}

func ptGenWinSFX() []byte {
	dur := 0.5
	buf := pcmStereo(dur)
	samples := len(buf) / 4
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		frac := t / dur
		// Rising chime
		freq := 523.0
		if frac > 0.25 {
			freq = 659.0
		}
		if frac > 0.5 {
			freq = 784.0
		}
		if frac > 0.75 {
			freq = 1047.0
		}
		val := sineWave(t, freq)*0.3 + sineWave(t, freq*2)*0.1
		env := math.Sin(frac * math.Pi)
		writeSample(buf, i*4, val*env*0.5)
	}
	return buf
}

// ── Constructor ─────────────────────────────────────────────────────────────

func newPortal() *Portal {
	p := &Portal{
		pw: 12,
		ph: 20,
	}

	// Generate audio
	p.sfxPortal = ptGenPortalSFX()
	p.sfxTeleport = ptGenTeleportSFX()
	p.sfxJump = ptGenJumpSFX()
	p.sfxDeath = ptGenDeathSFX()
	p.sfxWin = ptGenWinSFX()

	bgmPCM := ptGenBGM()
	p.bgm = loopPlayer(bgmPCM, 0.25)
	p.bgm.Play()

	p.loadLevel(0)
	return p
}

// ── Level loading ───────────────────────────────────────────────────────────

func (p *Portal) loadLevel(idx int) {
	if idx >= len(ptLevels) {
		idx = 0 // wrap around
	}
	p.level = idx
	p.grid = ptLevels[idx]
	p.origGrid = ptLevels[idx]
	p.dead = false
	p.deadTimer = 0
	p.levelWon = false
	p.winTimer = 0
	p.pvx = 0
	p.pvy = 0
	p.onGround = false
	p.portals = ptPortalPair{}
	p.cubes = nil
	p.buttonPressed = false
	p.doorOpen = false
	p.teleportCooldown = 0

	// Find spawn and cubes
	for r := 0; r < ptGridH; r++ {
		for c := 0; c < ptGridW; c++ {
			switch p.grid[r][c] {
			case ptSpawn:
				p.spawnX = float64(c*ptTileS) + float64(ptTileS)/2 - p.pw/2
				p.spawnY = float64(r*ptTileS) + float64(ptTileS) - p.ph
				p.px = p.spawnX
				p.py = p.spawnY
				p.grid[r][c] = ptAir
			case ptCube:
				p.cubes = append(p.cubes, ptCubeEntity{
					x: float64(c*ptTileS) + 2,
					y: float64(r*ptTileS) + 4,
					w: 20, h: 20,
				})
				p.grid[r][c] = ptAir
			}
		}
	}
}

func (p *Portal) restartLevel() {
	p.loadLevel(p.level)
}

// ── Tile collision helpers ──────────────────────────────────────────────────

func (p *Portal) ptIsSolid(gx, gy int) bool {
	if gx < 0 || gx >= ptGridW || gy < 0 || gy >= ptGridH {
		return true
	}
	t := p.grid[gy][gx]
	switch t {
	case ptWall, ptNoPortal:
		return true
	case ptDoor:
		return !p.doorOpen
	}
	return false
}

func (p *Portal) ptIsPortalable(gx, gy int) bool {
	if gx < 0 || gx >= ptGridW || gy < 0 || gy >= ptGridH {
		return false
	}
	return p.grid[gy][gx] == ptWall
}

func (p *Portal) ptIsDangerous(gx, gy int) bool {
	if gx < 0 || gx >= ptGridW || gy < 0 || gy >= ptGridH {
		return false
	}
	return p.grid[gy][gx] == ptSpike
}

func (p *Portal) ptIsExit(gx, gy int) bool {
	if gx < 0 || gx >= ptGridW || gy < 0 || gy >= ptGridH {
		return false
	}
	return p.grid[gy][gx] == ptExit
}

func (p *Portal) ptIsButton(gx, gy int) bool {
	if gx < 0 || gx >= ptGridW || gy < 0 || gy >= ptGridH {
		return false
	}
	return p.grid[gy][gx] == ptButton
}

// ── AABB collision resolve ──────────────────────────────────────────────────

// moveAndCollide moves an entity with collision, returns new position and whether
// entity is on ground. Applies to both player and cubes.
func (p *Portal) moveAndCollide(x, y, w, h, vx, vy float64) (float64, float64, float64, float64, bool) {
	onGnd := false

	// Horizontal movement
	x += vx
	// Check horizontal collision
	leftTile := int(math.Floor(x / ptTileS))
	rightTile := int(math.Floor((x + w - 0.01) / ptTileS))
	topTile := int(math.Floor(y / ptTileS))
	bottomTile := int(math.Floor((y + h - 0.01) / ptTileS))

	for ty := topTile; ty <= bottomTile; ty++ {
		for tx := leftTile; tx <= rightTile; tx++ {
			if p.ptIsSolid(tx, ty) {
				if vx > 0 {
					x = float64(tx*ptTileS) - w
					vx = 0
				} else if vx < 0 {
					x = float64((tx+1)*ptTileS)
					vx = 0
				}
			}
		}
	}

	// Vertical movement
	y += vy
	leftTile = int(math.Floor(x / ptTileS))
	rightTile = int(math.Floor((x + w - 0.01) / ptTileS))
	topTile = int(math.Floor(y / ptTileS))
	bottomTile = int(math.Floor((y + h - 0.01) / ptTileS))

	for ty := topTile; ty <= bottomTile; ty++ {
		for tx := leftTile; tx <= rightTile; tx++ {
			if p.ptIsSolid(tx, ty) {
				if vy > 0 {
					y = float64(ty*ptTileS) - h
					vy = 0
					onGnd = true
				} else if vy < 0 {
					y = float64((ty+1)*ptTileS)
					vy = 0
				}
			}
		}
	}

	return x, y, vx, vy, onGnd
}

// ── Portal placement via raycasting ─────────────────────────────────────────

// ptRaycast: from pixel origin (ox,oy) in direction (dx,dy), find the first
// solid tile face. Returns tile coords and which face was hit.
func (p *Portal) ptRaycast(ox, oy, dx, dy float64) (int, int, int, bool) {
	// Normalize direction
	mag := math.Sqrt(dx*dx + dy*dy)
	if mag < 0.001 {
		return 0, 0, 0, false
	}
	dx /= mag
	dy /= mag

	// Step along ray
	step := 1.0
	for dist := 0.0; dist < 600.0; dist += step {
		rx := ox + dx*dist
		ry := oy + dy*dist
		gx := int(math.Floor(rx / ptTileS))
		gy := int(math.Floor(ry / ptTileS))

		if gx < 0 || gx >= ptGridW || gy < 0 || gy >= ptGridH {
			return 0, 0, 0, false
		}

		if p.ptIsSolid(gx, gy) || p.ptIsDangerous(gx, gy) {
			// Don't allow portal on non-portalable surfaces
			if !p.ptIsPortalable(gx, gy) {
				return 0, 0, 0, false
			}

			// Determine which face was hit by checking the previous position
			prx := ox + dx*(dist-step)
			pry := oy + dy*(dist-step)
			pgx := int(math.Floor(prx / ptTileS))
			pgy := int(math.Floor(pry / ptTileS))

			dir := ptDirUp
			if pgy < gy {
				dir = ptDirUp // coming from above, portal faces up
			} else if pgy > gy {
				dir = ptDirDown // coming from below, portal faces down
			} else if pgx < gx {
				dir = ptDirLeft // coming from left, portal faces left
			} else if pgx > gx {
				dir = ptDirRight // coming from right, portal faces right
			} else {
				// Same tile — use sub-pixel position to determine face
				localX := rx - float64(gx*ptTileS)
				localY := ry - float64(gy*ptTileS)
				halfT := float64(ptTileS) / 2.0
				// Check which edge is closest
				dTop := localY
				dBot := float64(ptTileS) - localY
				dLeft := localX
				dRight := float64(ptTileS) - localX
				_ = halfT
				minD := dTop
				dir = ptDirUp
				if dBot < minD {
					minD = dBot
					dir = ptDirDown
				}
				if dLeft < minD {
					minD = dLeft
					dir = ptDirLeft
				}
				if dRight < minD {
					dir = ptDirRight
				}
			}
			return gx, gy, dir, true
		}
	}
	return 0, 0, 0, false
}

// ── Portal teleportation ────────────────────────────────────────────────────

// ptExitPos returns the pixel position and velocity for exiting a portal
func ptExitPos(tileX, tileY, dir int, pw, ph, speed float64) (float64, float64, float64, float64) {
	// Position player just outside the portal tile
	cx := float64(tileX*ptTileS) + float64(ptTileS)/2
	cy := float64(tileY*ptTileS) + float64(ptTileS)/2

	var ex, ey, evx, evy float64

	// Place the player clearly outside the portal tile to avoid moveAndCollide
	// detecting a collision on the next frame and zeroing velocity.
	const exitGap = 4.0

	switch dir {
	case ptDirUp:
		// Portal faces up — exit above the tile
		ex = cx - pw/2
		ey = float64(tileY*ptTileS) - ph - exitGap
		evx = 0
		evy = -speed
	case ptDirDown:
		// Portal faces down — exit below the tile
		ex = cx - pw/2
		ey = float64((tileY+1)*ptTileS) + exitGap
		evx = 0
		evy = speed
	case ptDirLeft:
		// Portal faces left — exit to the left of the tile
		ex = float64(tileX*ptTileS) - pw - exitGap
		ey = cy - ph/2
		evx = -speed
		evy = 0
	case ptDirRight:
		// Portal faces right — exit to the right of the tile
		ex = float64((tileX+1)*ptTileS) + exitGap
		ey = cy - ph/2
		evx = speed
		evy = 0
	}
	return ex, ey, evx, evy
}

func (p *Portal) ptTryTeleport() {
	if p.teleportCooldown > 0 || !p.portals.bactive || !p.portals.oactive {
		return
	}

	pcx := p.px + p.pw/2
	pcy := p.py + p.ph/2

	speed := math.Sqrt(p.pvx*p.pvx + p.pvy*p.pvy)
	if speed < 2.0 {
		speed = 2.0 // tiny nudge so player doesn't get stuck in portal
	}
	// Momentum conservation: "speedy thing goes in, speedy thing comes out."
	// Add a small energy boost to compensate for gravity/friction that will
	// eat velocity between sequential portals in a chain.
	speed *= 1.08

	// Check if player overlaps blue portal
	if p.ptPlayerOverlapsPortal(pcx, pcy, p.portals.bx, p.portals.by, p.portals.bdir) {
		ex, ey, evx, evy := ptExitPos(p.portals.ox, p.portals.oy, p.portals.odir, p.pw, p.ph, speed)
		p.px = ex
		p.py = ey
		p.pvx = evx
		p.pvy = evy
		p.onGround = false // exiting a portal means you're airborne
		p.teleportCooldown = 6
		playSFX(p.sfxTeleport, 0.5)
		return
	}

	// Check if player overlaps orange portal
	if p.ptPlayerOverlapsPortal(pcx, pcy, p.portals.ox, p.portals.oy, p.portals.odir) {
		ex, ey, evx, evy := ptExitPos(p.portals.bx, p.portals.by, p.portals.bdir, p.pw, p.ph, speed)
		p.px = ex
		p.py = ey
		p.pvx = evx
		p.pvy = evy
		p.onGround = false // exiting a portal means you're airborne
		p.teleportCooldown = 6
		playSFX(p.sfxTeleport, 0.5)
		return
	}
}

func (p *Portal) ptPlayerOverlapsPortal(pcx, pcy float64, tx, ty, dir int) bool {
	// Portal zone: a thin region on the face of the tile
	var zx, zy, zw, zh float64

	switch dir {
	case ptDirUp:
		zx = float64(tx*ptTileS) + 2
		zy = float64(ty*ptTileS) - 4
		zw = float64(ptTileS) - 4
		zh = 8
	case ptDirDown:
		zx = float64(tx*ptTileS) + 2
		zy = float64((ty+1)*ptTileS) - 4
		zw = float64(ptTileS) - 4
		zh = 8
	case ptDirLeft:
		zx = float64(tx*ptTileS) - 4
		zy = float64(ty*ptTileS) + 2
		zw = 8
		zh = float64(ptTileS) - 4
	case ptDirRight:
		zx = float64((tx+1)*ptTileS) - 4
		zy = float64(ty*ptTileS) + 2
		zw = 8
		zh = float64(ptTileS) - 4
	}

	// Check if player center is in portal zone
	hw := p.pw / 2
	hh := p.ph / 2
	return pcx-hw < zx+zw && pcx+hw > zx && pcy-hh < zy+zh && pcy+hh > zy
}

// ── Update ──────────────────────────────────────────────────────────────────

func (p *Portal) Update() error {
	p.ticks++

	// ESC: go back
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.goBack = true
		if p.bgm != nil {
			p.bgm.Pause()
		}
		return nil
	}

	// R: restart level
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		p.restartLevel()
		return nil
	}
	// N: skip to next level
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		next := p.level + 1
		if next >= len(ptLevels) {
			next = 0
		}
		p.loadLevel(next)
		return nil
	}

	// Level won animation
	if p.levelWon {
		p.winTimer++
		if p.winTimer > 60 {
			if p.level+1 < len(ptLevels) {
				p.loadLevel(p.level + 1)
			} else {
				p.loadLevel(0) // wrap
			}
		}
		return nil
	}

	// Death animation
	if p.dead {
		p.deadTimer++
		if p.deadTimer > 40 {
			p.deaths++
			p.restartLevel()
		}
		return nil
	}

	// ── Portal placement ──
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p.ptPlacePortal(true) // blue
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		p.ptPlacePortal(false) // orange
	}

	// ── Player input ──
	moveX := 0.0
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		moveX = -ptMoveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		moveX = ptMoveSpeed
	}

	wantJump := inpututil.IsKeyJustPressed(ebiten.KeyW) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace)

	// Apply horizontal input
	if p.onGround {
		// When on the ground, only override velocity if player is providing input
		// OR if current speed is at/below normal move speed. This preserves portal
		// momentum when landing at high speed.
		if moveX != 0 {
			absVx := math.Abs(p.pvx)
			if absVx <= ptMoveSpeed {
				p.pvx = moveX
			} else {
				// Blend input with existing momentum so player has some control
				p.pvx += moveX * 0.15
			}
		}
	} else {
		// Air control (reduced) — only clamp if player is actively pushing AND
		// the velocity is below normal move speed. Never clamp high portal speeds.
		if moveX != 0 {
			p.pvx += moveX * 0.15
			// Only clamp toward move speed if we aren't carrying portal momentum
			absVx := math.Abs(p.pvx)
			if absVx < ptMoveSpeed {
				if p.pvx > ptMoveSpeed {
					p.pvx = ptMoveSpeed
				}
				if p.pvx < -ptMoveSpeed {
					p.pvx = -ptMoveSpeed
				}
			}
		}
	}

	// Jump
	if wantJump && p.onGround {
		p.pvy = ptJumpVel
		p.onGround = false
		playSFX(p.sfxJump, 0.3)
	}

	// Gravity
	p.pvy += ptGravity
	if p.pvy > ptMaxFallV {
		p.pvy = ptMaxFallV
	}

	// Ground friction — only apply if cooldown has fully expired, so we don't
	// kill portal momentum the frame after landing from a fling.
	if p.onGround && moveX == 0 && p.teleportCooldown == 0 {
		p.pvx *= ptFriction
		if p.pvx > -0.1 && p.pvx < 0.1 {
			p.pvx = 0
		}
	}

	// Teleport cooldown
	if p.teleportCooldown > 0 {
		p.teleportCooldown--
	}

	// ── Portal teleportation ──
	// IMPORTANT: Check teleport BEFORE moveAndCollide. When the player falls
	// into a floor portal, we need to capture their full velocity before
	// collision resolution would zero it against the solid portal tile.
	// We tentatively apply velocity to position for the overlap check, then
	// undo it if no teleport occurred, letting moveAndCollide handle it.
	tentX := p.px + p.pvx
	tentY := p.py + p.pvy
	oldPx, oldPy := p.px, p.py
	p.px = tentX
	p.py = tentY
	p.ptTryTeleport()
	if p.px == tentX && p.py == tentY {
		// No teleport happened — restore position for normal moveAndCollide
		p.px = oldPx
		p.py = oldPy
	} else {
		// Teleport happened — skip moveAndCollide this frame to avoid
		// the collision resolver interfering with the new exit position.
		goto afterCollide
	}

	{
		// Move and collide
		var newOnGnd bool
		p.px, p.py, p.pvx, p.pvy, newOnGnd = p.moveAndCollide(p.px, p.py, p.pw, p.ph, p.pvx, p.pvy)
		p.onGround = newOnGnd
	}
afterCollide:

	// ── Update cubes ──
	p.ptUpdateCubes()

	// ── Check button state ──
	p.ptCheckButtons()

	// ── Check hazards ──
	p.ptCheckHazards()

	// ── Check exit ──
	p.ptCheckExit()

	// Clamp player to screen
	if p.px < 0 {
		p.px = 0
		p.pvx = 0
	}
	if p.px+p.pw > float64(ptScreenW) {
		p.px = float64(ptScreenW) - p.pw
		p.pvx = 0
	}
	if p.py > float64(ptScreenH)+50 {
		// Fell off bottom
		p.dead = true
		playSFX(p.sfxDeath, 0.4)
	}

	return nil
}

func (p *Portal) ptPlacePortal(blue bool) {
	mx, my := ebiten.CursorPosition()
	// Ray from player center to mouse
	pcx := p.px + p.pw/2
	pcy := p.py + p.ph/2
	dx := float64(mx) - pcx
	dy := float64(my) - pcy

	gx, gy, dir, hit := p.ptRaycast(pcx, pcy, dx, dy)
	if !hit {
		return
	}

	if blue {
		p.portals.bx = gx
		p.portals.by = gy
		p.portals.bdir = dir
		p.portals.bactive = true
	} else {
		p.portals.ox = gx
		p.portals.oy = gy
		p.portals.odir = dir
		p.portals.oactive = true
	}
	playSFX(p.sfxPortal, 0.4)
}

func (p *Portal) ptUpdateCubes() {
	for i := range p.cubes {
		c := &p.cubes[i]
		// Gravity
		c.vy += ptGravity
		if c.vy > ptMaxFallV {
			c.vy = ptMaxFallV
		}

		// Player pushes cube horizontally
		if p.ptBoxOverlap(p.px, p.py, p.pw, p.ph, c.x, c.y, c.w, c.h) {
			if p.pvx > 0 && p.px+p.pw/2 < c.x+c.w/2 {
				c.vx = p.pvx
			} else if p.pvx < 0 && p.px+p.pw/2 > c.x+c.w/2 {
				c.vx = p.pvx
			}
		}

		// Friction on cube
		if c.onGnd {
			c.vx *= 0.7
			if c.vx > -0.1 && c.vx < 0.1 {
				c.vx = 0
			}
		}

		c.x, c.y, c.vx, c.vy, c.onGnd = p.moveAndCollide(c.x, c.y, c.w, c.h, c.vx, c.vy)
	}
}

func (p *Portal) ptBoxOverlap(ax, ay, aw, ah, bx, by, bw, bh float64) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

func (p *Portal) ptCheckButtons() {
	p.buttonPressed = false
	for r := 0; r < ptGridH; r++ {
		for c := 0; c < ptGridW; c++ {
			if p.grid[r][c] == ptButton {
				bx := float64(c * ptTileS)
				by := float64(r*ptTileS) + float64(ptTileS) - 6
				bw := float64(ptTileS)
				bh := 6.0

				// Player on button
				if p.ptBoxOverlap(p.px, p.py, p.pw, p.ph, bx, by, bw, bh) {
					p.buttonPressed = true
				}
				// Cube on button
				for _, cube := range p.cubes {
					if p.ptBoxOverlap(cube.x, cube.y, cube.w, cube.h, bx, by, bw, bh) {
						p.buttonPressed = true
					}
				}
			}
		}
	}
	p.doorOpen = p.buttonPressed
}

func (p *Portal) ptCheckHazards() {
	// Check player feet/body tiles for spikes
	leftTile := int(math.Floor(p.px / ptTileS))
	rightTile := int(math.Floor((p.px + p.pw - 0.01) / ptTileS))
	topTile := int(math.Floor(p.py / ptTileS))
	bottomTile := int(math.Floor((p.py + p.ph - 0.01) / ptTileS))

	for ty := topTile; ty <= bottomTile; ty++ {
		for tx := leftTile; tx <= rightTile; tx++ {
			if p.ptIsDangerous(tx, ty) {
				p.dead = true
				playSFX(p.sfxDeath, 0.4)
				return
			}
		}
	}
}

func (p *Portal) ptCheckExit() {
	leftTile := int(math.Floor(p.px / ptTileS))
	rightTile := int(math.Floor((p.px + p.pw - 0.01) / ptTileS))
	topTile := int(math.Floor(p.py / ptTileS))
	bottomTile := int(math.Floor((p.py + p.ph - 0.01) / ptTileS))

	for ty := topTile; ty <= bottomTile; ty++ {
		for tx := leftTile; tx <= rightTile; tx++ {
			if p.ptIsExit(tx, ty) {
				p.levelWon = true
				playSFX(p.sfxWin, 0.5)
				return
			}
		}
	}
}

// ── Draw ────────────────────────────────────────────────────────────────────

func (p *Portal) Draw(screen *ebiten.Image) {
	// Dark background
	screen.Fill(color.RGBA{0x10, 0x10, 0x18, 0xff})

	// Draw tiles
	p.ptDrawTiles(screen)

	// Draw portals
	p.ptDrawPortals(screen)

	// Draw cubes
	p.ptDrawCubes(screen)

	// Draw player
	p.ptDrawPlayer(screen)

	// Draw crosshair and aim line
	p.ptDrawCrosshair(screen)

	// Draw HUD
	p.ptDrawHUD(screen)

	// Level won overlay
	if p.levelWon {
		alpha := p.winTimer * 4
		if alpha > 200 {
			alpha = 200
		}
		vector.FillRect(screen, 0, 0, ptScreenW, ptScreenH, color.RGBA{0, 0, 0, uint8(alpha)}, false)
		if p.level+1 < len(ptLevels) {
			ebitenutil.DebugPrintAt(screen, "LEVEL COMPLETE!", ptScreenW/2-50, ptScreenH/2-10)
		} else {
			ebitenutil.DebugPrintAt(screen, "ALL LEVELS COMPLETE! Restarting...", ptScreenW/2-100, ptScreenH/2-10)
		}
	}

	// Death overlay
	if p.dead {
		alpha := p.deadTimer * 6
		if alpha > 180 {
			alpha = 180
		}
		vector.FillRect(screen, 0, 0, ptScreenW, ptScreenH, color.RGBA{0x80, 0, 0, uint8(alpha)}, false)
		ebitenutil.DebugPrintAt(screen, "DEATH", ptScreenW/2-16, ptScreenH/2-6)
	}
}

func (p *Portal) ptDrawTiles(screen *ebiten.Image) {
	for r := 0; r < ptGridH; r++ {
		for c := 0; c < ptGridW; c++ {
			t := p.grid[r][c]
			fx := float32(c * ptTileS)
			fy := float32(r * ptTileS)
			fs := float32(ptTileS)

			switch t {
			case ptWall:
				// Gray, portal-able
				vector.FillRect(screen, fx, fy, fs, fs, color.RGBA{0x60, 0x60, 0x70, 0xff}, false)
				vector.StrokeRect(screen, fx, fy, fs, fs, 1, color.RGBA{0x50, 0x50, 0x58, 0xff}, false)
			case ptNoPortal:
				// Darker, no portals
				vector.FillRect(screen, fx, fy, fs, fs, color.RGBA{0x30, 0x30, 0x38, 0xff}, false)
				vector.StrokeRect(screen, fx, fy, fs, fs, 1, color.RGBA{0x28, 0x28, 0x30, 0xff}, false)
				// Draw X pattern to indicate no-portal
				vector.StrokeLine(screen, fx+2, fy+2, fx+fs-2, fy+fs-2, 1, color.RGBA{0x40, 0x20, 0x20, 0xff}, false)
				vector.StrokeLine(screen, fx+fs-2, fy+2, fx+2, fy+fs-2, 1, color.RGBA{0x40, 0x20, 0x20, 0xff}, false)
			case ptExit:
				// Green door
				vector.FillRect(screen, fx+4, fy, fs-8, fs, color.RGBA{0x00, 0x80, 0x00, 0xff}, false)
				// Glow
				glowAlpha := uint8(80 + 40*math.Sin(float64(p.ticks)*0.1))
				vector.FillRect(screen, fx+2, fy-2, fs-4, fs+4, color.RGBA{0x00, 0xff, 0x00, glowAlpha}, false)
				// Door handle
				vector.FillCircle(screen, fx+fs-8, fy+fs/2, 2, color.RGBA{0xff, 0xff, 0x00, 0xff}, false)
			case ptButton:
				// Yellow pressure plate
				pressed := p.buttonPressed
				plateH := float32(4)
				if pressed {
					plateH = 2
				}
				vector.FillRect(screen, fx+2, fy+fs-plateH, fs-4, plateH, color.RGBA{0xff, 0xcc, 0x00, 0xff}, false)
				vector.StrokeRect(screen, fx+2, fy+fs-plateH, fs-4, plateH, 1, color.RGBA{0xcc, 0x99, 0x00, 0xff}, false)
			case ptDoor:
				if p.doorOpen {
					// Open door — draw faint outline
					vector.StrokeRect(screen, fx, fy, fs, fs, 1, color.RGBA{0x40, 0x60, 0x40, 0x60}, false)
				} else {
					// Closed door — dark red
					vector.FillRect(screen, fx, fy, fs, fs, color.RGBA{0x60, 0x30, 0x30, 0xff}, false)
					vector.StrokeRect(screen, fx, fy, fs, fs, 1, color.RGBA{0x80, 0x40, 0x40, 0xff}, false)
				}
			case ptSpike:
				// Red spikes
				for i := 0; i < 4; i++ {
					sx := fx + float32(i*6) + 1
					// Triangle spike
					var path vector.Path
					path.MoveTo(sx, fy+fs)
					path.LineTo(sx+3, fy+4)
					path.LineTo(sx+6, fy+fs)
					path.Close()

					vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
					for vi := range vs {
						vs[vi].ColorR = 0.9
						vs[vi].ColorG = 0.15
						vs[vi].ColorB = 0.15
						vs[vi].ColorA = 1.0
					}
					screen.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{})
				}
			}
		}
	}
}

func (p *Portal) ptDrawPortals(screen *ebiten.Image) {
	if p.portals.bactive {
		p.ptDrawOnePortal(screen, p.portals.bx, p.portals.by, p.portals.bdir,
			color.RGBA{0x00, 0x80, 0xff, 0xff}, color.RGBA{0x00, 0x40, 0xff, 0x60})
	}
	if p.portals.oactive {
		p.ptDrawOnePortal(screen, p.portals.ox, p.portals.oy, p.portals.odir,
			color.RGBA{0xff, 0x80, 0x00, 0xff}, color.RGBA{0xff, 0x40, 0x00, 0x60})
	}
}

func (p *Portal) ptDrawOnePortal(screen *ebiten.Image, tx, ty, dir int, col, glow color.RGBA) {
	cx := float32(tx*ptTileS) + float32(ptTileS)/2
	cy := float32(ty*ptTileS) + float32(ptTileS)/2

	// Pulsing glow
	pulse := float32(0.5 + 0.5*math.Sin(float64(p.ticks)*0.15))
	_ = pulse

	switch dir {
	case ptDirUp:
		// Horizontal oval on top face
		oy := float32(ty * ptTileS)
		vector.FillRect(screen, cx-10, oy-3, 20, 6, glow, false)
		vector.FillRect(screen, cx-8, oy-2, 16, 4, col, false)
	case ptDirDown:
		// Horizontal oval on bottom face
		oy := float32((ty + 1) * ptTileS)
		vector.FillRect(screen, cx-10, oy-3, 20, 6, glow, false)
		vector.FillRect(screen, cx-8, oy-2, 16, 4, col, false)
	case ptDirLeft:
		// Vertical oval on left face
		ox := float32(tx * ptTileS)
		vector.FillRect(screen, ox-3, cy-10, 6, 20, glow, false)
		vector.FillRect(screen, ox-2, cy-8, 4, 16, col, false)
	case ptDirRight:
		// Vertical oval on right face
		ox := float32((tx + 1) * ptTileS)
		vector.FillRect(screen, ox-3, cy-10, 6, 20, glow, false)
		vector.FillRect(screen, ox-2, cy-8, 4, 16, col, false)
	}
}

func (p *Portal) ptDrawCubes(screen *ebiten.Image) {
	for _, c := range p.cubes {
		fx := float32(c.x)
		fy := float32(c.y)
		fw := float32(c.w)
		fh := float32(c.h)
		// Brown cube
		vector.FillRect(screen, fx, fy, fw, fh, color.RGBA{0x8B, 0x5A, 0x2B, 0xff}, false)
		vector.StrokeRect(screen, fx, fy, fw, fh, 1, color.RGBA{0x6B, 0x3A, 0x0B, 0xff}, false)
		// Cross decoration
		vector.StrokeLine(screen, fx+2, fy+2, fx+fw-2, fy+fh-2, 1, color.RGBA{0xA0, 0x70, 0x40, 0xff}, false)
		vector.StrokeLine(screen, fx+fw-2, fy+2, fx+2, fy+fh-2, 1, color.RGBA{0xA0, 0x70, 0x40, 0xff}, false)
	}
}

func (p *Portal) ptDrawPlayer(screen *ebiten.Image) {
	if p.dead {
		return
	}
	fx := float32(p.px)
	fy := float32(p.py)
	fw := float32(p.pw)
	fh := float32(p.ph)

	// Body — cyan/white stick figure area
	vector.FillRect(screen, fx, fy, fw, fh, color.RGBA{0x00, 0xdd, 0xdd, 0xff}, false)
	vector.StrokeRect(screen, fx, fy, fw, fh, 1, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)

	// Head (circle above body)
	headR := float32(4)
	headX := fx + fw/2
	headY := fy + headR + 1
	vector.FillCircle(screen, headX, headY, headR, color.RGBA{0xff, 0xff, 0xff, 0xff}, false)

	// Eyes
	vector.FillCircle(screen, headX-2, headY-1, 1, color.RGBA{0x00, 0x00, 0x00, 0xff}, false)
	vector.FillCircle(screen, headX+2, headY-1, 1, color.RGBA{0x00, 0x00, 0x00, 0xff}, false)
}

func (p *Portal) ptDrawCrosshair(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	fmx := float32(mx)
	fmy := float32(my)

	// Crosshair
	chSize := float32(6)
	chCol := color.RGBA{0xff, 0xff, 0xff, 0xaa}
	vector.StrokeLine(screen, fmx-chSize, fmy, fmx+chSize, fmy, 1, chCol, false)
	vector.StrokeLine(screen, fmx, fmy-chSize, fmx, fmy+chSize, 1, chCol, false)
	vector.StrokeCircle(screen, fmx, fmy, chSize-1, 1, chCol, false)

	// Dotted aim line from player to cursor
	pcx := float32(p.px + p.pw/2)
	pcy := float32(p.py + p.ph/2)
	dx := fmx - pcx
	dy := fmy - pcy
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist > 1 {
		ndx := dx / dist
		ndy := dy / dist
		for d := float32(10); d < dist && d < 300; d += 8 {
			dotx := pcx + ndx*d
			doty := pcy + ndy*d
			vector.FillRect(screen, dotx-1, doty-1, 2, 2, color.RGBA{0xff, 0xff, 0xff, 0x30}, false)
		}
	}
}

func (p *Portal) ptDrawHUD(screen *ebiten.Image) {
	// Level number
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Level %d/%d", p.level+1, len(ptLevels)), 4, 2)

	// Deaths
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Deaths: %d", p.deaths), ptScreenW-80, 2)

	// Portal status
	bStatus := "---"
	if p.portals.bactive {
		bStatus = "SET"
	}
	oStatus := "---"
	if p.portals.oactive {
		oStatus = "SET"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Blue: %s  Orange: %s", bStatus, oStatus), 4, 16)

	// Tutorial text
	if p.level < len(ptTutorials) {
		txt := ptTutorials[p.level]
		// Wrap long text
		if len(txt) > 70 {
			ebitenutil.DebugPrintAt(screen, txt[:70], 4, ptScreenH-28)
			ebitenutil.DebugPrintAt(screen, txt[70:], 4, ptScreenH-14)
		} else {
			ebitenutil.DebugPrintAt(screen, txt, 4, ptScreenH-14)
		}
	}

	// Controls hint
	ebitenutil.DebugPrintAt(screen, "WASD:Move  Space:Jump  LClick:Blue  RClick:Orange  R:Restart  N:Skip  ESC:Menu", 4, ptScreenH-28)
}

// ── Layout ──────────────────────────────────────────────────────────────────

func (p *Portal) Layout(w, h int) (int, int) {
	return ptScreenW, ptScreenH
}

// ── WantsBack ───────────────────────────────────────────────────────────────

func (p *Portal) WantsBack() bool {
	return p.goBack
}
