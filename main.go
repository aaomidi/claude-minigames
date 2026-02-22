package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// SubGame is implemented by each game in the collection.
type SubGame interface {
	Update() error
	Draw(screen *ebiten.Image)
	Layout(w, h int) (int, int)
	WantsBack() bool
}

// Launcher is the top-level ebiten.Game that manages the menu and active game.
type Launcher struct {
	active SubGame
}

type menuItem struct {
	label string
	make  func() SubGame
}

var menuItems = []menuItem{
	{"Mine Sweep", func() SubGame { return newMinesweeper() }},
	{"Naval Strike", func() SubGame { return newBattleship() }},
	{"Snek", func() SubGame { return newSnake() }},
	{"Road Hopper", func() SubGame { return newFrogger() }},
	{"Traffic Dodge", func() SubGame { return newTrafficDodge() }},
	{"Silver Ball", func() SubGame { return newPinball() }},
	{"Serpents & Steps", func() SubGame { return newSnakesLadders() }},
	{"Tower Defense", func() SubGame { return newTowerDefense() }},
	{"Deep Miner", func() SubGame { return newMotherlode() }},
	{"Idle Clicker", func() SubGame { return newIdleClicker() }},
	{"Idle Evolution", func() SubGame { return newIdleEvolution() }},
	{"Symbiotic Grid", func() SubGame { return newSymbioticGrid() }},
	{"Harvest Valley", func() SubGame { return newFarmRPG() }},
	{"Hellshot", func() SubGame { return newDoom() }},
	{"Warp Gate", func() SubGame { return newPortal() }},
	{"Hero Summon", func() SubGame { return newGacha() }},
	{"Realm Quest", func() SubGame { return newRunescape() }},
	{"Crime City", func() SubGame { return newGTA() }},
	{"Epoch", func() SubGame { return newCivilization() }},
}

const (
	menuCols   = 2
	menuBtnW   = 190
	menuBtnH   = 38
	menuPadX   = 20
	menuPadY   = 12
	menuStartX = 20
	menuStartY = 55
	menuW      = menuStartX*2 + menuCols*menuBtnW + (menuCols-1)*menuPadX
	menuRows   = 10 // ceil(19 items / 2 cols)
	menuH      = menuStartY + menuRows*(menuBtnH+menuPadY) + 40
)

func menuBtnPos(i int) (int, int) {
	col := i % menuCols
	row := i / menuCols
	bx := menuStartX + col*(menuBtnW+menuPadX)
	by := menuStartY + row*(menuBtnH+menuPadY)
	return bx, by
}

func (l *Launcher) Update() error {
	if l.active != nil {
		if l.active.WantsBack() {
			l.active = nil
			ebiten.SetWindowSize(menuW*2, menuH*2)
			ebiten.SetWindowTitle("Game Collection")
			return nil
		}
		return l.active.Update()
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for i, item := range menuItems {
			bx, by := menuBtnPos(i)
			if mx >= bx && mx < bx+menuBtnW && my >= by && my < by+menuBtnH {
				g := item.make()
				l.active = g
				sw, sh := g.Layout(0, 0)
				ebiten.SetWindowSize(sw*2, sh*2)
				ebiten.SetWindowTitle(item.label)
				return nil
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	return nil
}

func (l *Launcher) Draw(screen *ebiten.Image) {
	if l.active != nil {
		l.active.Draw(screen)
		return
	}

	screen.Fill(color.RGBA{0x2a, 0x2a, 0x3a, 0xff})

	ebitenutil.DebugPrintAt(screen, "GAME COLLECTION", menuW/2-48, 18)

	mx, my := ebiten.CursorPosition()
	for i, item := range menuItems {
		bx, by := menuBtnPos(i)
		fbx, fby := float32(bx), float32(by)

		bg := color.RGBA{0x40, 0x40, 0x60, 0xff}
		if mx >= bx && mx < bx+menuBtnW && my >= by && my < by+menuBtnH {
			bg = color.RGBA{0x50, 0x50, 0x80, 0xff}
		}
		vector.FillRect(screen, fbx, fby, menuBtnW, menuBtnH, bg, false)
		vector.StrokeRect(screen, fbx, fby, menuBtnW, menuBtnH, 2, color.RGBA{0x80, 0x80, 0xa0, 0xff}, false)

		tx := bx + menuBtnW/2 - len(item.label)*3
		ebitenutil.DebugPrintAt(screen, item.label, tx, by+11)
	}

	ebitenutil.DebugPrintAt(screen, "Click to play  |  ESC to quit", menuW/2-90, menuH-22)
}

func (l *Launcher) Layout(outsideWidth, outsideHeight int) (int, int) {
	if l.active != nil {
		return l.active.Layout(outsideWidth, outsideHeight)
	}
	return menuW, menuH
}

func main() {
	ebiten.SetWindowSize(menuW*2, menuH*2)
	ebiten.SetWindowTitle("Game Collection")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(&Launcher{}); err != nil {
		log.Fatal(err)
	}
}
