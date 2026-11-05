package scene

import (
	"time"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/world"
)

const AttackRetryInterval = 1200 * time.Millisecond
const ChaseRetryInterval = 450 * time.Millisecond

func AttackRetryDue(last time.Time, now time.Time) bool {
	return last.IsZero() || now.Sub(last) >= AttackRetryInterval
}

type CellRangePredicate func(sourceX, sourceY, targetX, targetY, attackRange int) bool

func MovingActorDestinationWithinRange(actor world.Actor, targetX, targetY, attackRange int, inRange CellRangePredicate) bool {
	if !actor.Moving || inRange == nil {
		return false
	}
	return inRange(actor.X, actor.Y, targetX, targetY, attackRange)
}

func ActionVisualHitCount(action network.ActorActionNotify) int {
	if action.HitCount == 0 {
		return 1
	}
	return MaxInt(1, int(action.HitCount))
}
