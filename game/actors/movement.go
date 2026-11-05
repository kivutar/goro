package actors

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	worldstate "github.com/kivutar/goro/world"
)

func UpsertActor(ctx client.Context, actor worldstate.Actor) {
	UpsertActorAt(ctx, actor, time.Now())
}

func UpsertActorAt(ctx client.Context, actor worldstate.Actor, now time.Time) {
	if ctx.World == nil {
		return
	}
	existing, hasExisting := ctx.World.Actors[actor.ID]
	// Network movement entries have no local start time yet. Updates copied
	// from a live actor already have one, even after the walk has finished;
	// changing direction, appearance or status must not replay that path.
	if actor.Moving && actor.MoveStarted.IsZero() {
		if actor.FromX == 0 && actor.FromY == 0 {
			if hasExisting {
				actor.FromX = existing.X
				actor.FromY = existing.Y
			}
		}
		if actor.ToX == 0 && actor.ToY == 0 {
			actor.ToX = actor.X
			actor.ToY = actor.Y
		}
		if actor.Speed <= 0 && hasExisting && existing.Speed > 0 {
			actor.Speed = existing.Speed
		}
		if len(actor.MovePath) == 0 {
			actor.MovePath = maps.WalkPath(ctx.World.GAT, actor.FromX, actor.FromY, actor.ToX, actor.ToY)
			if len(actor.MovePath) == 0 {
				// The map may have changed (for example an Ice Wall). Do not
				// draw a straight line through it when the announced route fails.
				actor.X, actor.Y = actor.FromX, actor.FromY
				if hasExisting {
					actor.X, actor.Y = scene.ActorCurrentCell(existing, now)
				}
				ClearActorMovement(&actor)
				ctx.World.UpsertActor(actor)
				return
			}
		}
		fastForward := time.Duration(0)
		if actor.HasMoveStartTick && ctx.Session != nil {
			if elapsed, ok := ctx.Session.ElapsedSinceServerTick(actor.MoveStartTick, now); ok {
				fastForward = elapsed
			}
		}
		speed := scene.ActorMoveSpeed(actor)
		startX, startY, hasMoveStart, offset := MovementStart(actor.MovePath, existing, now, fastForward, speed, actor.FromX, actor.FromY)
		actor.MoveStarted = now.Add(-offset)
		actor.MoveSpeed = speed
		actor.MoveDuration = ActorMovementDurationFromWithSpeed(actor.MovePath, actor.FromX, actor.FromY, actor.ToX, actor.ToY, speed, startX, startY, hasMoveStart)
		actor.MoveStartX = startX
		actor.MoveStartY = startY
		actor.HasMoveStart = hasMoveStart
		if hasExisting && scene.ActorIsMovingAt(existing, now) {
			actor.WalkDistance = MovementWalkDistanceOffset(actor.MovePath, existing, now, offset, speed, startX, startY, hasMoveStart)
		}
	}
	ctx.World.UpsertActor(actor)
}

func ApplyWarpPosition(ctx client.Context, x, y int) {
	dir := ctx.World.Dir
	if ctx.Session.PlayerDir != 0 {
		dir = ctx.Session.PlayerDir
	}
	ctx.Session.PlayerX = x
	ctx.Session.PlayerY = y
	ctx.World.SetPlayerPosition(x, y, dir)
}

func ApplyActorSetPosition(ctx client.Context, position network.ActorSetPosition) {
	if ctx.World == nil {
		return
	}
	if scene.IsLocalActor(ctx, position.ID) {
		ctx.World.SetPlayerPosition(position.X, position.Y, ctx.World.Dir)
		ctx.Session.PlayerX = position.X
		ctx.Session.PlayerY = position.Y
		return
	}
	// Position updates can also refer to skill units; they never spawn actors.
	if _, ok := ctx.World.Actors[position.ID]; !ok {
		return
	}
	UpsertActor(ctx, worldstate.Actor{
		ID: position.ID,
		X:  position.X,
		Y:  position.Y,
	})
}

func ApplyActorJumpPosition(ctx client.Context, position network.ActorJumpPosition) {
	if ctx.World == nil {
		return
	}
	if scene.IsLocalActor(ctx, position.ID) {
		ctx.World.SetPlayerPosition(position.X, position.Y, ctx.World.Dir)
		ctx.Session.PlayerX = position.X
		ctx.Session.PlayerY = position.Y
		return
	}
	if _, ok := ctx.World.Actors[position.ID]; !ok {
		return
	}
	UpsertActor(ctx, worldstate.Actor{
		ID: position.ID,
		X:  position.X,
		Y:  position.Y,
	})
}

func StopLocalPlayerMovementAt(ctx client.Context, at time.Time) {
	if ctx.World == nil || !scene.ActorIsMovingAt(ctx.World.Player, at) {
		return
	}
	x, y := scene.ActorRenderPosition(ctx.World.Player, at)
	ctx.World.Player.X = int(math.Round(x))
	ctx.World.Player.Y = int(math.Round(y))
	ClearActorMovement(&ctx.World.Player)
	if ctx.Session != nil {
		ctx.Session.PlayerX = ctx.World.Player.X
		ctx.Session.PlayerY = ctx.World.Player.Y
	}
}

func ClearActorMovement(actor *worldstate.Actor) {
	actor.Moving = false
	actor.FromX = actor.X
	actor.FromY = actor.Y
	actor.ToX = actor.X
	actor.ToY = actor.Y
	actor.MovePath = nil
	actor.HasMoveStart = false
	actor.MoveStartX = 0
	actor.MoveStartY = 0
	actor.WalkDistance = 0
	actor.MoveSpeed = 0
}

func actorRenderWalkDistance(actor worldstate.Actor, now time.Time) float64 {
	if !scene.ActorIsMovingAt(actor, now) || actor.MoveDuration <= 0 {
		return 0
	}
	base := actor.WalkDistance
	elapsed := now.Sub(actor.MoveStarted)
	if elapsed < 0 {
		elapsed = 0
	}
	if len(actor.MovePath) >= 2 {
		return base + pathWalkDistanceWithSpeed(actor.MovePath, elapsed, scene.ActorWalkSpeed(actor), actor.MoveStartX, actor.MoveStartY, actor.HasMoveStart)
	}
	t := float64(elapsed) / float64(actor.MoveDuration)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return base + math.Hypot(float64(actor.ToX-actor.FromX), float64(actor.ToY-actor.FromY))*t
}

func ActorMovementDurationFromWithSpeed(path []worldstate.WalkStep, fromX, fromY, toX, toY int, speedMS int, startX, startY float64, hasMoveStart bool) time.Duration {
	if len(path) >= 2 {
		total := time.Duration(0)
		for i := 1; i < len(path); i++ {
			segmentStartX, segmentStartY := scene.PathSegmentStart(path, i, startX, startY, hasMoveStart)
			total += scene.MovementSegmentDurationFloatWithSpeed(float64(path[i].X)-segmentStartX, float64(path[i].Y)-segmentStartY, speedMS)
		}
		if total > 0 {
			return total
		}
	}
	dx := scene.AbsInt(toX - fromX)
	dy := scene.AbsInt(toY - fromY)
	if dx == 0 && dy == 0 {
		return movementSegmentDurationWithSpeed(0, 0, speedMS)
	}
	return movementSegmentDurationWithSpeed(dx, dy, speedMS)
}

func DirectionFromDelta(fromX, fromY, toX, toY int, fallback int) int {
	return scene.DirectionFromFloatDelta(float64(fromX), float64(fromY), float64(toX), float64(toY), scene.NormalizeDirectionIndex(fallback))
}

func DirectMovementPath(fromX, fromY, toX, toY int) []worldstate.WalkStep {
	if fromX == toX && fromY == toY {
		return []worldstate.WalkStep{{X: fromX, Y: fromY}}
	}
	return []worldstate.WalkStep{{X: fromX, Y: fromY}, {X: toX, Y: toY}}
}

func MovementStart(path []worldstate.WalkStep, oldPlayer worldstate.Actor, now time.Time, fastForward time.Duration, speedMS, fromX, fromY int) (float64, float64, bool, time.Duration) {
	startX := float64(fromX)
	startY := float64(fromY)
	duration := time.Duration(0)
	if len(path) > 0 {
		duration = actorMovementDurationWithSpeed(path, path[0].X, path[0].Y, path[len(path)-1].X, path[len(path)-1].Y, speedMS)
	}
	if duration > 0 && fastForward > duration*4 {
		fastForward = 0
	}
	offset := clampMovementOffset(fastForward, duration)
	if scene.ActorIsMovingAt(oldPlayer, now) && len(path) >= 2 {
		x, y := scene.ActorRenderPosition(oldPlayer, now)
		if currentOffset, ok := pathElapsedAtPositionWithSpeed(path, x, y, speedMS); ok {
			if currentOffset > offset {
				offset = clampMovementOffset(currentOffset, duration)
			}
			return startX, startY, false, offset
		}
		if math.Hypot(x-float64(fromX), y-float64(fromY)) <= 1.25 {
			return x, y, true, 0
		}
		if nearMovementStartLine(path, x, y, 3.0, 0.25) {
			return x, y, true, 0
		}
	}
	return startX, startY, false, offset
}

func MovementWalkDistanceOffset(path []worldstate.WalkStep, oldPlayer worldstate.Actor, now time.Time, offset time.Duration, speedMS int, startX, startY float64, hasMoveStart bool) float64 {
	if !scene.ActorIsMovingAt(oldPlayer, now) {
		return 0
	}
	previous := actorRenderWalkDistance(oldPlayer, now)
	current := pathWalkDistanceWithSpeed(path, offset, speedMS, startX, startY, hasMoveStart)
	return previous - current
}

func pathWalkDistanceWithSpeed(path []worldstate.WalkStep, elapsed time.Duration, speedMS int, startX, startY float64, hasMoveStart bool) float64 {
	if len(path) < 2 || elapsed <= 0 {
		return 0
	}
	remaining := elapsed
	total := 0.0
	for i := 1; i < len(path); i++ {
		fromX, fromY := scene.PathSegmentStart(path, i, startX, startY, hasMoveStart)
		toX := float64(path[i].X)
		toY := float64(path[i].Y)
		segmentDistance := math.Hypot(toX-fromX, toY-fromY)
		duration := scene.MovementSegmentDurationFloatWithSpeed(toX-fromX, toY-fromY, speedMS)
		if duration <= 0 {
			total += segmentDistance
			continue
		}
		if remaining >= duration {
			total += segmentDistance
			remaining -= duration
			continue
		}
		t := float64(remaining) / float64(duration)
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		total += segmentDistance * t
		break
	}
	return total
}

func movementSegmentDurationWithSpeed(dx, dy int, speedMS int) time.Duration {
	if speedMS <= 0 {
		speedMS = scene.DefaultMoveSpeedMS
	}
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx == 0 && dy == 0 {
		return time.Duration(speedMS) * time.Millisecond
	}
	if dx != 0 && dy != 0 && dx == dy {
		return time.Duration(math.Round(float64(dx)*float64(speedMS)*math.Sqrt2)) * time.Millisecond
	}
	steps := dx
	if dy > steps {
		steps = dy
	}
	return time.Duration(steps*speedMS) * time.Millisecond
}

func actorMovementDurationWithSpeed(path []worldstate.WalkStep, fromX, fromY, toX, toY int, speedMS int) time.Duration {
	return ActorMovementDurationFromWithSpeed(path, fromX, fromY, toX, toY, speedMS, 0, 0, false)
}

func clampMovementOffset(offset, duration time.Duration) time.Duration {
	if offset < 0 {
		return 0
	}
	if duration > 0 && offset > duration {
		return duration
	}
	return offset
}

func pathElapsedAtPositionWithSpeed(path []worldstate.WalkStep, x, y float64, speedMS int) (time.Duration, bool) {
	if len(path) < 2 {
		return 0, false
	}
	bestDistance := math.Inf(1)
	bestElapsed := time.Duration(0)
	elapsedBefore := time.Duration(0)
	for i := 1; i < len(path); i++ {
		from := path[i-1]
		to := path[i]
		dx := float64(to.X - from.X)
		dy := float64(to.Y - from.Y)
		segmentDuration := movementSegmentDurationWithSpeed(to.X-from.X, to.Y-from.Y, speedMS)
		segmentLengthSq := dx*dx + dy*dy
		if segmentLengthSq == 0 {
			elapsedBefore += segmentDuration
			continue
		}
		t := ((x-float64(from.X))*dx + (y-float64(from.Y))*dy) / segmentLengthSq
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		px := float64(from.X) + dx*t
		py := float64(from.Y) + dy*t
		distance := math.Hypot(x-px, y-py)
		if distance < bestDistance {
			bestDistance = distance
			bestElapsed = elapsedBefore + time.Duration(float64(segmentDuration)*t)
		}
		elapsedBefore += segmentDuration
	}
	if bestDistance > 0.05 {
		return 0, false
	}
	return bestElapsed, true
}

func nearMovementStartLine(path []worldstate.WalkStep, x, y float64, maxDistance, maxLateral float64) bool {
	if len(path) < 2 {
		return false
	}
	from := path[0]
	to := path[1]
	fromX := float64(from.X)
	fromY := float64(from.Y)
	dx := float64(to.X - from.X)
	dy := float64(to.Y - from.Y)
	lengthSq := dx*dx + dy*dy
	if lengthSq == 0 {
		return false
	}
	if math.Hypot(x-fromX, y-fromY) > maxDistance {
		return false
	}
	t := ((x-fromX)*dx + (y-fromY)*dy) / lengthSq
	if t > 0.25 {
		return false
	}
	px := fromX + dx*t
	py := fromY + dy*t
	return math.Hypot(x-px, y-py) <= maxLateral
}
