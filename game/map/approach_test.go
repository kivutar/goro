package maps_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestAttackTargetWithinRangeUsesMeleeAdjacency(t *testing.T) {
	if !maps.AttackTargetWithinRange(10, 20, 11, 21, 1) {
		t.Fatal("diagonal adjacent target should be in melee range")
	}
	if maps.AttackTargetWithinRange(10, 20, 12, 20, 1) {
		t.Fatal("two cells away should be out of melee range")
	}
	if !maps.AttackTargetWithinRange(10, 20, 15, 20, 5) {
		t.Fatal("five cells away should be in bow range")
	}
}

func TestNormalAttackTargetWithinRangeRejectsRangedCorner(t *testing.T) {
	if !maps.NormalAttackTargetWithinRange(10, 20, 11, 21, 1) {
		t.Fatal("normal melee attack should keep diagonal adjacency")
	}
	if !maps.AttackTargetWithinRange(124, 70, 129, 75, 5) {
		t.Fatal("client skill-style range should still include the square corner")
	}
	if maps.NormalAttackTargetWithinRange(124, 70, 129, 75, 5) {
		t.Fatal("normal ranged attack should reject the server-failing square corner")
	}
	if !maps.NormalAttackTargetWithinRange(125, 72, 129, 75, 5) {
		t.Fatal("normal ranged attack should allow a non-corner cell at the same range")
	}
}

func TestNormalAttackRangeUsesMovingActorCurrentCell(t *testing.T) {
	now := time.Now()
	target := worldstate.Actor{
		ID:           300,
		X:            30,
		Y:            10,
		FromX:        20,
		FromY:        10,
		ToX:          30,
		ToY:          10,
		Moving:       true,
		MoveStarted:  now,
		MoveDuration: 10 * time.Second,
		MovePath: []worldstate.WalkStep{
			{X: 20, Y: 10},
			{X: 30, Y: 10},
		},
	}

	targetX, targetY := scene.ActorCurrentCell(target, now)
	if !maps.NormalAttackTargetWithinRange(11, 10, targetX, targetY, 9) {
		t.Fatal("moving target should be in range at its current rendered cell")
	}
	if maps.AttackTargetWithinRange(11, 10, target.X, target.Y, 9) {
		t.Fatal("test target final destination should be out of range")
	}
}

func TestAttackApproachCellChoosesClosestWalkableNeighbor(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 112, Y: 302}
	world.GAT = &res.GAT{
		Width:  200,
		Height: 400,
		Cells:  make([]res.GATCell, 200*400),
	}
	for i := range world.GAT.Cells {
		world.GAT.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
	ctx := client.Context{World: world}
	actor := worldstate.Actor{ID: 300, X: 116, Y: 303}

	x, y, ok := maps.NormalAttackApproachCellFromTarget(ctx, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, 1)
	if !ok {
		t.Fatal("expected approach cell")
	}
	if x != 114 || y != 303 {
		t.Fatalf("approach = %d,%d, want 114,303", x, y)
	}

	world.GAT.Cells[303*world.GAT.Width+114] = res.GATCell{}
	x, y, ok = maps.NormalAttackApproachCellFromTarget(ctx, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, 1)
	if !ok {
		t.Fatal("expected fallback approach cell")
	}
	if x == 114 && y == 303 {
		t.Fatalf("blocked approach cell was selected")
	}
}

func TestAttackApproachCellUsesRangedAttackRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 112, Y: 302}
	world.GAT = &res.GAT{
		Width:  200,
		Height: 400,
		Cells:  make([]res.GATCell, 200*400),
	}
	for i := range world.GAT.Cells {
		world.GAT.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
	ctx := client.Context{World: world}
	actor := worldstate.Actor{ID: 300, X: 120, Y: 302}

	x, y, ok := maps.NormalAttackApproachCellFromTarget(ctx, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, 5)
	if !ok {
		t.Fatal("expected ranged approach cell")
	}
	targetDistance := scene.MaxInt(scene.AbsInt(x-actor.X), scene.AbsInt(y-actor.Y))
	if !maps.NormalAttackTargetWithinRange(x, y, actor.X, actor.Y, 5) {
		t.Fatalf("ranged approach = %d,%d, distance %d from target, want within client range 5", x, y, targetDistance)
	}
	if targetDistance <= 1 {
		t.Fatalf("ranged approach = %d,%d, want not adjacent contact", x, y)
	}
}

func TestNormalAttackApproachCellAvoidsRejectedRangedCorner(t *testing.T) {
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(200, 100)
	ctx := client.Context{World: world}

	x, y, ok := maps.NormalAttackApproachCellFromTarget(ctx, 123, 71, 129, 75, 5)
	if !ok {
		t.Fatal("expected ranged normal attack approach cell")
	}
	if x == 124 && y == 70 {
		t.Fatalf("normal attack approach = %d,%d, want to avoid server-rejected range corner", x, y)
	}
	if !maps.NormalAttackTargetWithinRange(x, y, 129, 75, 5) {
		t.Fatalf("normal attack approach = %d,%d, want server-compatible range", x, y)
	}
	if !maps.WalkTargetReachable(ctx, 123, 71, x, y) {
		t.Fatalf("normal attack approach = %d,%d, want reachable path", x, y)
	}
}

func TestRangedAttackApproachCellRequiresReachablePath(t *testing.T) {
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(12, 5)
	for y := 0; y < world.GAT.Height; y++ {
		world.GAT.SetCellRawType(4, y, 1)
	}
	ctx := client.Context{World: world}

	if x, y, ok := maps.RangedAttackApproachCellFromTarget(ctx, 1, 2, 8, 2, 3); ok {
		t.Fatalf("ranged approach = %d,%d, want no unreachable chase cell", x, y)
	}
}

func TestSquareRangeApproachChoosesNearestCellInEveryDirection(t *testing.T) {
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	ctx := client.Context{World: world}
	const sourceX, sourceY, skillRange = 20, 20, 9
	for _, dx := range []int{-11, -2, 0, 2, 11} {
		for _, dy := range []int{-11, -2, 0, 2, 11} {
			if scene.MaxInt(scene.AbsInt(dx), scene.AbsInt(dy)) <= skillRange {
				continue
			}
			t.Run(fmt.Sprintf("offset_%d_%d", dx, dy), func(t *testing.T) {
				targetX, targetY := sourceX+dx, sourceY+dy
				x, y, ok := maps.AttackApproachCellFromTarget(ctx, sourceX, sourceY, targetX, targetY, skillRange)
				// On open ground, only move along axes that are out of range.
				wantX := scene.ClampInt(sourceX, targetX-skillRange, targetX+skillRange)
				wantY := scene.ClampInt(sourceY, targetY-skillRange, targetY+skillRange)
				if !ok || x != wantX || y != wantY {
					t.Fatalf("approach = %d,%d ok=%t, want nearest in-range cell %d,%d", x, y, ok, wantX, wantY)
				}
			})
		}
	}
}

func TestRangedAttackApproachCellSkipsUnreachablePreferredCell(t *testing.T) {
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(12, 5)
	for _, cell := range [][2]int{
		{4, 1}, {4, 2}, {4, 3},
		{5, 1}, {5, 3},
		{6, 1}, {6, 2}, {6, 3},
	} {
		world.GAT.SetCellRawType(cell[0], cell[1], 1)
	}
	ctx := client.Context{World: world}

	x, y, ok := maps.RangedAttackApproachCellFromTarget(ctx, 1, 2, 8, 2, 3)
	if !ok {
		t.Fatal("expected reachable fallback approach cell")
	}
	if x != 5 || y != 0 {
		t.Fatalf("ranged approach = %d,%d, want reachable fallback 5,0", x, y)
	}
}
