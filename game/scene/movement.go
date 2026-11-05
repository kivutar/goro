package scene

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	worldstate "github.com/kivutar/goro/world"
)

func ActorIsMovingAt(actor worldstate.Actor, now time.Time) bool {
	return actor.Moving && !actor.MoveStarted.IsZero() && now.Before(actor.MoveStarted.Add(actor.MoveDuration))
}

func ActorRenderPosition(actor worldstate.Actor, now time.Time) (float64, float64) {
	if !ActorIsMovingAt(actor, now) || actor.MoveDuration <= 0 {
		return float64(actor.X), float64(actor.Y)
	}
	elapsed := now.Sub(actor.MoveStarted)
	if len(actor.MovePath) >= 2 {
		return RenderPathPositionWithSpeed(actor.MovePath, elapsed, ActorWalkSpeed(actor), actor.MoveStartX, actor.MoveStartY, actor.HasMoveStart)
	}
	t := float64(elapsed) / float64(actor.MoveDuration)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	x := float64(actor.FromX) + float64(actor.ToX-actor.FromX)*t
	y := float64(actor.FromY) + float64(actor.ToY-actor.FromY)*t
	return x, y
}

func ActorRenderDirection(actor worldstate.Actor, now time.Time) int {
	if !ActorIsMovingAt(actor, now) || len(actor.MovePath) < 2 {
		return actor.Dir
	}
	elapsed := now.Sub(actor.MoveStarted)
	fromX, fromY, toX, toY := RenderPathSegmentWithSpeed(actor.MovePath, elapsed, ActorWalkSpeed(actor), actor.MoveStartX, actor.MoveStartY, actor.HasMoveStart)
	if math.Abs(fromX-toX) < 0.001 && math.Abs(fromY-toY) < 0.001 {
		return actor.Dir
	}
	return DirectionFromFloatDelta(fromX, fromY, toX, toY, actor.Dir)
}

const DefaultMoveSpeedMS = 150

func ActorWalkSpeed(actor worldstate.Actor) int {
	if actor.MoveSpeed > 0 {
		return actor.MoveSpeed
	}
	return ActorMoveSpeed(actor)
}

func ActorMoveSpeed(actor worldstate.Actor) int {
	if actor.Speed > 0 {
		return actor.Speed
	}
	return DefaultMoveSpeedMS
}

func MovementSegmentDurationFloatWithSpeed(dx, dy float64, speedMS int) time.Duration {
	if speedMS <= 0 {
		speedMS = DefaultMoveSpeedMS
	}
	dist := math.Hypot(dx, dy)
	if dist <= 0 {
		return 0
	}
	return time.Duration(math.Round(dist*float64(speedMS))) * time.Millisecond
}

func RenderPathPositionWithSpeed(path []worldstate.WalkStep, elapsed time.Duration, speedMS int, startX, startY float64, hasMoveStart bool) (float64, float64) {
	if len(path) == 0 {
		return 0, 0
	}
	if len(path) == 1 {
		return float64(path[0].X), float64(path[0].Y)
	}
	if elapsed < 0 {
		elapsed = 0
	}
	remaining := elapsed
	for i := 1; i < len(path); i++ {
		fromX, fromY := PathSegmentStart(path, i, startX, startY, hasMoveStart)
		toX := float64(path[i].X)
		toY := float64(path[i].Y)
		duration := MovementSegmentDurationFloatWithSpeed(toX-fromX, toY-fromY, speedMS)
		if duration <= 0 {
			continue
		}
		if remaining <= duration || i == len(path)-1 {
			t := float64(remaining) / float64(duration)
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			return fromX + (toX-fromX)*t, fromY + (toY-fromY)*t
		}
		remaining -= duration
	}
	last := path[len(path)-1]
	return float64(last.X), float64(last.Y)
}

func RenderPathSegmentWithSpeed(path []worldstate.WalkStep, elapsed time.Duration, speedMS int, startX, startY float64, hasMoveStart bool) (float64, float64, float64, float64) {
	if len(path) == 0 {
		return 0, 0, 0, 0
	}
	if len(path) == 1 {
		x := float64(path[0].X)
		y := float64(path[0].Y)
		return x, y, x, y
	}
	if elapsed <= 0 {
		fromX, fromY := PathSegmentStart(path, 1, startX, startY, hasMoveStart)
		return fromX, fromY, float64(path[1].X), float64(path[1].Y)
	}
	remaining := elapsed
	for i := 1; i < len(path); i++ {
		fromX, fromY := PathSegmentStart(path, i, startX, startY, hasMoveStart)
		toX := float64(path[i].X)
		toY := float64(path[i].Y)
		duration := MovementSegmentDurationFloatWithSpeed(toX-fromX, toY-fromY, speedMS)
		if remaining < duration || i == len(path)-1 {
			return fromX, fromY, toX, toY
		}
		remaining -= duration
	}
	last := path[len(path)-1]
	x := float64(last.X)
	y := float64(last.Y)
	return x, y, x, y
}

func PathSegmentStart(path []worldstate.WalkStep, segmentIndex int, startX, startY float64, hasMoveStart bool) (float64, float64) {
	if segmentIndex == 1 && hasMoveStart {
		return startX, startY
	}
	from := path[segmentIndex-1]
	return float64(from.X), float64(from.Y)
}

func DirectionFromFloatDelta(fromX, fromY, toX, toY float64, fallback int) int {
	dx := toX - fromX
	dy := toY - fromY
	if math.Abs(dx) < 0.001 && math.Abs(dy) < 0.001 {
		if fallback < 0 {
			return 0
		}
		return fallback & 7
	}
	actionDir := int(math.Round(-math.Atan2(dy, dx)/(math.Pi/4)+6)) & 7
	return (4 - actionDir) & 7
}

func TerrainHeightAtRenderPoint(world *worldstate.World, x, y float64) float64 {
	return TerrainHeightAt(world, x-0.5, y-0.5)
}

func CurrentPlayerCell(ctx client.Context, now time.Time) (int, int) {
	if ctx.World == nil {
		return 0, 0
	}
	return ActorCurrentCell(ctx.World.Player, now)
}

func ActorCurrentCell(actor worldstate.Actor, now time.Time) (int, int) {
	x, y := ActorRenderPosition(actor, now)
	return int(math.Round(x)), int(math.Round(y))
}

func WalkTargetInBounds(ctx client.Context, x, y int) bool {
	if x < 0 || y < 0 {
		return false
	}
	if ctx.World.GAT != nil {
		return x < ctx.World.GAT.Width && y < ctx.World.GAT.Height
	}
	if ctx.World.GND != nil {
		return x < ctx.World.GND.Width && y < ctx.World.GND.Height
	}
	return x <= 1023 && y <= 1023
}
