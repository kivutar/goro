package maps

import (
	"math"
	"sort"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
)

func AttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange int) bool {
	return scene.MaxInt(scene.AbsInt(playerX-targetX), scene.AbsInt(playerY-targetY)) <= scene.MaxInt(1, attackRange)
}

func NormalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange int) bool {
	dx := float64(playerX - targetX)
	dy := float64(playerY - targetY)
	return math.Hypot(dx, dy) < float64(scene.MaxInt(1, attackRange))+1.1
}

func NormalAttackApproachCellFromTarget(ctx client.Context, sourceX, sourceY, targetX, targetY int, attackRange int) (int, int, bool) {
	return RangedAttackApproachCellFromTargetMatching(ctx, sourceX, sourceY, targetX, targetY, attackRange, NormalAttackTargetWithinRange)
}

func AttackApproachCellFromTarget(ctx client.Context, sourceX, sourceY, targetX, targetY int, attackRange int) (int, int, bool) {
	attackRange = scene.MaxInt(1, attackRange)
	if attackRange > 1 {
		if x, y, ok := RangedAttackApproachCellFromTarget(ctx, sourceX, sourceY, targetX, targetY, attackRange); ok {
			return x, y, true
		}
	}
	bestX, bestY := 0, 0
	bestDistance := math.Inf(1)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			x := targetX + dx
			y := targetY + dy
			if !scene.WalkTargetInBounds(ctx, x, y) {
				continue
			}
			if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			if !WalkTargetReachable(ctx, sourceX, sourceY, x, y) {
				continue
			}
			distance := math.Hypot(float64(x-sourceX), float64(y-sourceY))
			if distance < bestDistance {
				bestDistance = distance
				bestX = x
				bestY = y
			}
		}
	}
	return bestX, bestY, bestDistance < math.Inf(1)
}

func RangedAttackApproachCellFromTarget(ctx client.Context, sourceX, sourceY, targetX, targetY int, attackRange int) (int, int, bool) {
	return RangedAttackApproachCellFromTargetMatching(ctx, sourceX, sourceY, targetX, targetY, attackRange, AttackTargetWithinRange)
}

func RangedAttackApproachCellFromTargetMatching(ctx client.Context, sourceX, sourceY, targetX, targetY int, attackRange int, inRange scene.CellRangePredicate) (int, int, bool) {
	if ctx.World == nil {
		return 0, 0, false
	}
	attackRange = scene.MaxInt(1, attackRange)
	source := PathPoint{X: sourceX, Y: sourceY}
	type candidate struct {
		x        int
		y        int
		walkCost int
	}
	searchRange := attackRange + 1
	candidates := make([]candidate, 0, (searchRange*2+1)*(searchRange*2+1))
	for dy := -searchRange; dy <= searchRange; dy++ {
		for dx := -searchRange; dx <= searchRange; dx++ {
			ringDistance := scene.MaxInt(scene.AbsInt(dx), scene.AbsInt(dy))
			if ringDistance == 0 || ringDistance > searchRange {
				continue
			}
			x := targetX + dx
			y := targetY + dy
			if inRange != nil && !inRange(x, y, targetX, targetY, attackRange) {
				continue
			}
			if !scene.WalkTargetInBounds(ctx, x, y) {
				continue
			}
			if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			// Rank by estimated walking cost, not square range distance: an
			// unnecessary diagonal step must not tie with a straight step.
			candidates = append(candidates, candidate{
				x:        x,
				y:        y,
				walkCost: PathHeuristic(source, PathPoint{X: x, Y: y}),
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.walkCost != right.walkCost {
			return left.walkCost < right.walkCost
		}
		if left.y != right.y {
			return left.y < right.y
		}
		return left.x < right.x
	})
	for _, candidate := range candidates {
		if !WalkTargetReachable(ctx, sourceX, sourceY, candidate.x, candidate.y) {
			continue
		}
		return candidate.x, candidate.y, true
	}
	return 0, 0, false
}

func WalkTargetReachable(ctx client.Context, sourceX, sourceY, targetX, targetY int) bool {
	if ctx.World == nil || ctx.World.GAT == nil {
		return true
	}
	if !ctx.World.GAT.InBounds(sourceX, sourceY) || !ctx.World.GAT.InBounds(targetX, targetY) {
		return false
	}
	if !ctx.World.GAT.Walkable(targetX, targetY) {
		return false
	}
	if sourceX == targetX && sourceY == targetY {
		return true
	}
	_, ok := FindWalkPath(ctx.World.GAT, sourceX, sourceY, targetX, targetY)
	return ok
}
