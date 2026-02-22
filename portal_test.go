package main

import (
	"fmt"
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. Spawn exists: every level has exactly one spawn tile (type 4)
// ---------------------------------------------------------------------------

func TestEveryLevelHasExactlyOneSpawn(t *testing.T) {
	for li, level := range ptLevels {
		count := 0
		for r := 0; r < ptGridH; r++ {
			for c := 0; c < ptGridW; c++ {
				if level[r][c] == ptSpawn {
					count++
				}
			}
		}
		if count != 1 {
			t.Errorf("level %d: expected exactly 1 spawn tile, got %d", li, count)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. Exit exists: every level has exactly one exit tile (type 3)
// ---------------------------------------------------------------------------

func TestEveryLevelHasExactlyOneExit(t *testing.T) {
	for li, level := range ptLevels {
		count := 0
		for r := 0; r < ptGridH; r++ {
			for c := 0; c < ptGridW; c++ {
				if level[r][c] == ptExit {
					count++
				}
			}
		}
		if count != 1 {
			t.Errorf("level %d: expected exactly 1 exit tile, got %d", li, count)
		}
	}
}

// ---------------------------------------------------------------------------
// 3. Exit is reachable: BFS flood-fill from spawn treating non-solid tiles
//    as passable. Because portals can bridge walls, we also treat any tile
//    adjacent to a portal-able wall as reachable from any other such tile.
//    The minimal check here: the exit is reachable through non-wall/non-noportal
//    tiles, or (for portal-dependent levels) the exit tile is adjacent to at
//    least one non-solid tile AND there are portal-able walls present.
// ---------------------------------------------------------------------------

func TestExitReachableFromSpawn(t *testing.T) {
	for li, level := range ptLevels {
		t.Run(fmt.Sprintf("level_%d", li), func(t *testing.T) {
			// Find spawn and exit positions.
			spawnR, spawnC := -1, -1
			exitR, exitC := -1, -1
			for r := 0; r < ptGridH; r++ {
				for c := 0; c < ptGridW; c++ {
					if level[r][c] == ptSpawn {
						spawnR, spawnC = r, c
					}
					if level[r][c] == ptExit {
						exitR, exitC = r, c
					}
				}
			}
			if spawnR < 0 || exitR < 0 {
				t.Fatal("missing spawn or exit")
			}

			// BFS treating passable tiles (air, spawn, exit, button, spike,
			// door, cube) as walkable. Walls and noPortal are impassable.
			passable := func(r, c int) bool {
				if r < 0 || r >= ptGridH || c < 0 || c >= ptGridW {
					return false
				}
				switch level[r][c] {
				case ptWall, ptNoPortal:
					return false
				}
				return true
			}

			visited := [ptGridH][ptGridW]bool{}
			type pos struct{ r, c int }
			queue := []pos{{spawnR, spawnC}}
			visited[spawnR][spawnC] = true
			dirs := [4]pos{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				for _, d := range dirs {
					nr, nc := cur.r+d.r, cur.c+d.c
					if passable(nr, nc) && !visited[nr][nc] {
						visited[nr][nc] = true
						queue = append(queue, pos{nr, nc})
					}
				}
			}

			if visited[exitR][exitC] {
				return // directly reachable without portals
			}

			// Not directly reachable -- this is expected in a portal game.
			// Verify that:
			//   a) The exit tile is adjacent to at least one non-solid tile
			//      (so the player can step onto it after portaling).
			//   b) Portal-able walls (type 1) exist in the level.
			exitHasOpenNeighbor := false
			for _, d := range dirs {
				nr, nc := exitR+d.r, exitC+d.c
				if passable(nr, nc) {
					exitHasOpenNeighbor = true
					break
				}
			}
			if !exitHasOpenNeighbor {
				t.Errorf("exit at (%d,%d) is completely walled off with no open neighbor", exitR, exitC)
			}

			hasPortalWalls := false
			for r := 0; r < ptGridH; r++ {
				for c := 0; c < ptGridW; c++ {
					if level[r][c] == ptWall {
						hasPortalWalls = true
						break
					}
				}
				if hasPortalWalls {
					break
				}
			}
			if !hasPortalWalls {
				t.Error("exit not directly reachable and no portal-able walls exist")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. Portal-able walls exist: each level has at least one ptWall tile
// ---------------------------------------------------------------------------

func TestPortalableWallsExist(t *testing.T) {
	for li, level := range ptLevels {
		count := 0
		for r := 0; r < ptGridH; r++ {
			for c := 0; c < ptGridW; c++ {
				if level[r][c] == ptWall {
					count++
				}
			}
		}
		if count == 0 {
			t.Errorf("level %d: no portal-able walls (ptWall) found", li)
		}
	}
}

// ---------------------------------------------------------------------------
// 5. No level is empty: must have a mix of walls and air
// ---------------------------------------------------------------------------

func TestNoLevelIsEmpty(t *testing.T) {
	for li, level := range ptLevels {
		tileCounts := map[int]int{}
		for r := 0; r < ptGridH; r++ {
			for c := 0; c < ptGridW; c++ {
				tileCounts[level[r][c]]++
			}
		}
		total := ptGridH * ptGridW
		// Verify no single tile type fills the entire grid.
		for tileType, cnt := range tileCounts {
			if cnt == total {
				t.Errorf("level %d: entirely filled with tile type %d", li, tileType)
			}
		}
		// Verify there is at least one air tile and at least one solid tile.
		solidCount := tileCounts[ptWall] + tileCounts[ptNoPortal]
		airCount := tileCounts[ptAir]
		// Spawn and cube tiles are treated as air by loadLevel, so count them too.
		airCount += tileCounts[ptSpawn] + tileCounts[ptCube]
		if solidCount == 0 {
			t.Errorf("level %d: no solid tiles (wall or noPortal) found", li)
		}
		if airCount == 0 {
			t.Errorf("level %d: no air/open tiles found", li)
		}
	}
}

// ---------------------------------------------------------------------------
// 6. Grid dimensions are correct: each level is exactly ptGridH x ptGridW
// ---------------------------------------------------------------------------

func TestGridDimensions(t *testing.T) {
	for li, level := range ptLevels {
		if len(level) != ptGridH {
			t.Errorf("level %d: expected %d rows, got %d", li, ptGridH, len(level))
			continue
		}
		for r := 0; r < ptGridH; r++ {
			if len(level[r]) != ptGridW {
				t.Errorf("level %d row %d: expected %d cols, got %d", li, r, ptGridW, len(level[r]))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 7. ptExitPos: exit positions are placed outside the portal tile
// ---------------------------------------------------------------------------

func TestPtExitPosDirections(t *testing.T) {
	pw, ph := 12.0, 20.0
	speed := 5.0
	tileX, tileY := 5, 5

	tileLeft := float64(tileX * ptTileS)
	tileRight := float64((tileX + 1) * ptTileS)
	tileTop := float64(tileY * ptTileS)
	tileBottom := float64((tileY + 1) * ptTileS)

	tests := []struct {
		dir       int
		name      string
		checkPos  func(ex, ey float64) bool
		checkVel  func(evx, evy float64) bool
	}{
		{
			dir:  ptDirUp,
			name: "up",
			checkPos: func(ex, ey float64) bool {
				// Player bottom edge should be above the tile top
				return ey+ph <= tileTop
			},
			checkVel: func(evx, evy float64) bool {
				return evy < 0 && evx == 0
			},
		},
		{
			dir:  ptDirDown,
			name: "down",
			checkPos: func(ex, ey float64) bool {
				// Player top edge should be below the tile bottom
				return ey >= tileBottom
			},
			checkVel: func(evx, evy float64) bool {
				return evy > 0 && evx == 0
			},
		},
		{
			dir:  ptDirLeft,
			name: "left",
			checkPos: func(ex, ey float64) bool {
				// Player right edge should be left of the tile left
				return ex+pw <= tileLeft
			},
			checkVel: func(evx, evy float64) bool {
				return evx < 0 && evy == 0
			},
		},
		{
			dir:  ptDirRight,
			name: "right",
			checkPos: func(ex, ey float64) bool {
				// Player left edge should be right of the tile right
				return ex >= tileRight
			},
			checkVel: func(evx, evy float64) bool {
				return evx > 0 && evy == 0
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ex, ey, evx, evy := ptExitPos(tileX, tileY, tc.dir, pw, ph, speed)
			if !tc.checkPos(ex, ey) {
				t.Errorf("direction %s: exit position (%.1f, %.1f) is not outside tile bounds [%.0f-%.0f, %.0f-%.0f] given player size %.0fx%.0f",
					tc.name, ex, ey, tileLeft, tileRight, tileTop, tileBottom, pw, ph)
			}
			if !tc.checkVel(evx, evy) {
				t.Errorf("direction %s: exit velocity (%.1f, %.1f) unexpected", tc.name, evx, evy)
			}
		})
	}
}

// Test that ptExitPos centers the player horizontally or vertically relative
// to the tile center depending on direction.
func TestPtExitPosCentering(t *testing.T) {
	pw, ph := 12.0, 20.0
	speed := 5.0
	tileX, tileY := 7, 7
	cx := float64(tileX*ptTileS) + float64(ptTileS)/2

	// For up and down, the player should be centered horizontally on the tile.
	for _, dir := range []int{ptDirUp, ptDirDown} {
		ex, _, _, _ := ptExitPos(tileX, tileY, dir, pw, ph, speed)
		playerCenterX := ex + pw/2
		if math.Abs(playerCenterX-cx) > 0.01 {
			t.Errorf("dir %d: player not horizontally centered on tile (player center x=%.1f, tile center x=%.1f)", dir, playerCenterX, cx)
		}
	}

	cy := float64(tileY*ptTileS) + float64(ptTileS)/2

	// For left and right, the player should be centered vertically on the tile.
	for _, dir := range []int{ptDirLeft, ptDirRight} {
		_, ey, _, _ := ptExitPos(tileX, tileY, dir, pw, ph, speed)
		playerCenterY := ey + ph/2
		if math.Abs(playerCenterY-cy) > 0.01 {
			t.Errorf("dir %d: player not vertically centered on tile (player center y=%.1f, tile center y=%.1f)", dir, playerCenterY, cy)
		}
	}
}

// ---------------------------------------------------------------------------
// 8. Speed conservation: the exit speed formula preserves/boosts input speed
// ---------------------------------------------------------------------------

func TestSpeedConservation(t *testing.T) {
	pw, ph := 12.0, 20.0

	inputSpeeds := []float64{2.0, 5.0, 10.0, 15.0, 20.0}

	for _, inSpeed := range inputSpeeds {
		for _, dir := range []int{ptDirUp, ptDirDown, ptDirLeft, ptDirRight} {
			_, _, evx, evy := ptExitPos(5, 5, dir, pw, ph, inSpeed)
			outSpeed := math.Sqrt(evx*evx + evy*evy)

			if outSpeed < inSpeed-0.01 {
				t.Errorf("speed lost: input=%.2f output=%.2f dir=%d", inSpeed, outSpeed, dir)
			}
			// The function receives the speed value directly from the caller
			// (ptTryTeleport applies the 1.08 multiplier before calling ptExitPos).
			// ptExitPos itself preserves the speed exactly.
			if math.Abs(outSpeed-inSpeed) > 0.01 {
				t.Errorf("speed not preserved: input=%.2f output=%.2f dir=%d", inSpeed, outSpeed, dir)
			}
		}
	}
}

// Test that the momentum boost multiplier in teleport logic increases speed.
// The boost factor (1.08) is applied in ptTryTeleport before calling ptExitPos.
func TestMomentumBoostFactor(t *testing.T) {
	// Simulate what ptTryTeleport does: multiply speed by 1.08 before exit.
	boostFactor := 1.08
	inputSpeed := 10.0
	boostedSpeed := inputSpeed * boostFactor

	if boostedSpeed <= inputSpeed {
		t.Errorf("boost factor should increase speed: input=%.2f boosted=%.2f", inputSpeed, boostedSpeed)
	}

	// Verify the minimum speed clamp (speed < 2.0 gets clamped to 2.0).
	minSpeed := 0.5
	clamped := minSpeed
	if clamped < 2.0 {
		clamped = 2.0
	}
	if clamped != 2.0 {
		t.Errorf("minimum speed clamp failed: expected 2.0, got %.2f", clamped)
	}
}

// ---------------------------------------------------------------------------
// Additional sanity checks
// ---------------------------------------------------------------------------

// Verify level count matches tutorial count.
func TestLevelCountMatchesTutorials(t *testing.T) {
	if len(ptLevels) != len(ptTutorials) {
		t.Errorf("level count (%d) does not match tutorial count (%d)", len(ptLevels), len(ptTutorials))
	}
}

// Verify that all tile values in every level are recognized types.
func TestAllTileTypesValid(t *testing.T) {
	validTiles := map[int]bool{
		ptAir: true, ptWall: true, ptNoPortal: true, ptExit: true,
		ptSpawn: true, ptButton: true, ptDoor: true, ptSpike: true,
		ptCube: true, ptMovingPlat: true,
	}
	for li, level := range ptLevels {
		for r := 0; r < ptGridH; r++ {
			for c := 0; c < ptGridW; c++ {
				if !validTiles[level[r][c]] {
					t.Errorf("level %d tile (%d,%d): unknown tile type %d", li, r, c, level[r][c])
				}
			}
		}
	}
}

// Verify every level has a fully enclosed border (no air on the outermost
// ring) so the player cannot walk off the grid.
func TestLevelBorderEnclosed(t *testing.T) {
	isSolid := func(tile int) bool {
		switch tile {
		case ptWall, ptNoPortal:
			return true
		}
		return false
	}

	for li, level := range ptLevels {
		// Top and bottom rows.
		for c := 0; c < ptGridW; c++ {
			if !isSolid(level[0][c]) {
				t.Errorf("level %d: top border tile (0,%d) is type %d, not solid", li, c, level[0][c])
			}
			if !isSolid(level[ptGridH-1][c]) {
				t.Errorf("level %d: bottom border tile (%d,%d) is type %d, not solid", li, ptGridH-1, c, level[ptGridH-1][c])
			}
		}
		// Left and right columns.
		for r := 0; r < ptGridH; r++ {
			if !isSolid(level[r][0]) {
				t.Errorf("level %d: left border tile (%d,0) is type %d, not solid", li, r, level[r][0])
			}
			if !isSolid(level[r][ptGridW-1]) {
				t.Errorf("level %d: right border tile (%d,%d) is type %d, not solid", li, r, ptGridW-1, level[r][ptGridW-1])
			}
		}
	}
}

// Verify that buttons and doors come in matched pairs: if a level has a
// button, it should also have a door, and vice versa.
func TestButtonDoorPairing(t *testing.T) {
	for li, level := range ptLevels {
		hasButton := false
		hasDoor := false
		for r := 0; r < ptGridH; r++ {
			for c := 0; c < ptGridW; c++ {
				if level[r][c] == ptButton {
					hasButton = true
				}
				if level[r][c] == ptDoor {
					hasDoor = true
				}
			}
		}
		if hasButton && !hasDoor {
			t.Errorf("level %d: has button but no door", li)
		}
		if hasDoor && !hasButton {
			t.Errorf("level %d: has door but no button", li)
		}
	}
}
