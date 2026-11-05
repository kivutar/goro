package maps_test

import (
	"strings"
	"testing"

	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/res"
)

func TestWalkPathFindsShortRouteThroughObstacles(t *testing.T) {
	// This 17-step route exhausted the old 150-node search limit.
	rows := strings.Fields(`
#....#............#.....
...##.#..........#.#....
#...#............#.#....
....#.......#.......#..#
..#..................###
.....#.#.##..#.#...#....
.#........#....#...#.#..
.............#..#..##..#
#.....S.#..#...#G.......
.#...#...#..####.....#..
##.##.......#..#.#...###
##........##..#.#..#.#..
...#......#.#...........
#.....#...#....#.#.#....
...#.......###..........
.#.#.#.........#...#.#.#
...#.#..........#.......
#.......#...............
.#.....#.##..##.##....#.
.......#.........#......
#.#...#..#.#..........#.
...####.....#.#.....#...
..##.......##.#...#.....
.#...#.........###...#.#`)
	gat := testutil.FlatWalkableGAT(len(rows[0]), len(rows))
	for y, row := range rows {
		if len(row) != gat.Width {
			t.Fatal("inconsistent fixture width")
		}
		for x, cell := range row {
			if cell == '#' {
				gat.Cells[y*gat.Width+x].Type = res.GATTypeNone
			}
		}
	}
	path, ok := maps.FindWalkPath(gat, 6, 8, 16, 8)
	if !ok || len(path) != 18 {
		t.Fatalf("short route was not found: %v", path)
	}
}

func TestWalkPathDoesNotInventRouteThroughWall(t *testing.T) {
	gat := testutil.FlatWalkableGAT(128, 128)
	for y := 0; y < gat.Height; y++ {
		gat.Cells[y*gat.Width+64].Type = res.GATTypeNone
	}
	if path := maps.WalkPath(gat, 60, 64, 68, 64); len(path) != 0 {
		t.Fatalf("unreachable target produced a path: %v", path)
	}
}

func TestWalkPathCanLeaveBlockedStartWithoutCuttingCorners(t *testing.T) {
	gat := testutil.FlatWalkableGAT(8, 8)
	gat.Cells[2*gat.Width+2].Type = res.GATTypeNone
	gat.Cells[2*gat.Width+3].Type = res.GATTypeNone
	path, ok := maps.FindWalkPath(gat, 2, 2, 4, 2)
	if !ok {
		t.Fatal("could not leave a cell blocked after the actor entered it")
	}
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		if !gat.Walkable(b.X, b.Y) || (a.X != b.X && a.Y != b.Y && (!gat.Walkable(a.X, b.Y) || !gat.Walkable(b.X, a.Y))) {
			t.Fatalf("path crosses a blocked cell or corner: %v", path)
		}
	}
}
