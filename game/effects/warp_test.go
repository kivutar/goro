package effects

import (
	"math"
	"testing"
	"time"
)

func TestWarpFunnelRingsShrinkAndRestart(t *testing.T) {
	// Prim3DCasting4: 0.05 per frame, restart at 10 with alpha reset.
	if d, a := warpFunnelRingState(5, 0); math.Abs(d-4.95) > 1e-9 || a != 1 {
		t.Fatalf("first step = %.3f alpha %d", d, a)
	}
	if d, a := warpFunnelRingState(5, 98); math.Abs(d-0.05) > 1e-9 || a != 70 {
		t.Fatalf("before restart = %.3f alpha %d", d, a)
	}
	if d, a := warpFunnelRingState(5, 99); d != 10 || a != 1 {
		t.Fatalf("restart = %.3f alpha %d, want 10 and 1", d, a)
	}
	if d, _ := warpFunnelRingState(5, 99+200); d != 10 {
		t.Fatalf("second restart = %.3f, want a 200-frame period", d)
	}
}

func TestWarpParticlesOrbitRiseAndFade(t *testing.T) {
	effect := Instance{EffectID: WarpZone2, ActorID: 1, Starts: time.Unix(10, 0)}
	for index := 0; index < 20; index++ {
		start, alpha := warpParticleAt(effect, index, 0)
		if alpha != 254 {
			t.Fatalf("particle %d starts with alpha %.1f", index, alpha)
		}
		end, endAlpha := warpParticleAt(effect, index, warpParticleLife-1)
		if end.Y >= start.Y || endAlpha > 20 {
			t.Fatalf("particle %d: start %+v end %+v alpha %.1f, want risen and faded", index, start, end, endAlpha)
		}
		if r := math.Hypot(start.X, start.Z); r > 2*5.25+0.1 {
			t.Fatalf("particle %d starts %.2f units away", index, r)
		}
	}
}
