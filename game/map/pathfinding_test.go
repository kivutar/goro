package maps_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/world"
)

func TestWalkPathMatchesClassicRoutes(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		blocked       map[world.WalkStep]bool
		want          []world.WalkStep
	}{
		{
			name: "Payon dungeon obstacle", width: 10, height: 14,
			blocked: map[world.WalkStep]bool{{X: 4, Y: 6}: true, {X: 5, Y: 6}: true, {X: 4, Y: 7}: true, {X: 5, Y: 7}: true},
			// Both routes reach the goal, but going east of this obstacle
			// puts the rendered character over four cells from the server.
			want: []world.WalkStep{{X: 7, Y: 2}, {X: 6, Y: 3}, {X: 5, Y: 4}, {X: 4, Y: 5}, {X: 3, Y: 5}, {X: 2, Y: 6}, {X: 2, Y: 7}, {X: 2, Y: 8}, {X: 2, Y: 9}, {X: 2, Y: 10}, {X: 2, Y: 11}},
		},
		{
			name: "equal cost detours", width: 3, height: 3,
			blocked: map[world.WalkStep]bool{{X: 1, Y: 1}: true},
			want:    []world.WalkStep{{X: 0, Y: 1}, {X: 0, Y: 2}, {X: 1, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 1}},
		},
		{
			name: "improved visited cell", width: 8, height: 8,
			blocked: map[world.WalkStep]bool{
				{X: 1, Y: 0}: true, {X: 2, Y: 0}: true, {X: 4, Y: 0}: true, {X: 6, Y: 0}: true, {X: 7, Y: 0}: true,
				{X: 1, Y: 1}: true, {X: 4, Y: 1}: true, {X: 5, Y: 3}: true, {X: 7, Y: 3}: true,
				{X: 3, Y: 4}: true, {X: 4, Y: 5}: true,
				{X: 2, Y: 6}: true, {X: 5, Y: 6}: true, {X: 6, Y: 6}: true, {X: 4, Y: 7}: true, {X: 5, Y: 7}: true,
			},
			want: []world.WalkStep{{X: 2, Y: 7}, {X: 1, Y: 7}, {X: 1, Y: 6}, {X: 1, Y: 5}, {X: 1, Y: 4}, {X: 2, Y: 3}, {X: 3, Y: 3}, {X: 4, Y: 3}, {X: 4, Y: 4}, {X: 5, Y: 4}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gat := testutil.PathGAT(tc.width, tc.height, tc.blocked)
			from, to := tc.want[0], tc.want[len(tc.want)-1]
			got, ok := maps.FindWalkPath(gat, from.X, from.Y, to.X, to.Y)
			if !ok || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("route = %v, want %v", got, tc.want)
			}
		})
	}
}

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
	want := []world.WalkStep{{X: 6, Y: 8}, {X: 7, Y: 7}, {X: 8, Y: 7}, {X: 9, Y: 7}, {X: 10, Y: 7}, {X: 11, Y: 7}, {X: 12, Y: 6}, {X: 13, Y: 6}, {X: 14, Y: 6}, {X: 14, Y: 5}, {X: 14, Y: 4}, {X: 15, Y: 4}, {X: 16, Y: 4}, {X: 16, Y: 5}, {X: 17, Y: 6}, {X: 17, Y: 7}, {X: 17, Y: 8}, {X: 16, Y: 8}}
	if !ok || !reflect.DeepEqual(path, want) {
		t.Fatalf("route = %v, want %v", path, want)
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
