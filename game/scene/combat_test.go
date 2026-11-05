package scene_test

import (
	"testing"
	"time"

	"github.com/kivutar/goro/game/scene"
)

func TestAttackRetryDueUsesReferenceInterval(t *testing.T) {
	now := time.Unix(100, 0)
	if !scene.AttackRetryDue(time.Time{}, now) {
		t.Fatal("zero last attack should be due")
	}
	if scene.AttackRetryDue(now.Add(-scene.AttackRetryInterval+time.Millisecond), now) {
		t.Fatal("attack should not retry before the interval")
	}
	if !scene.AttackRetryDue(now.Add(-scene.AttackRetryInterval), now) {
		t.Fatal("attack should retry at the interval")
	}
}
