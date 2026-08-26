package game

import (
	"math"
	"testing"
	"time"
)

func TestDamageFloaterPlacementUsesRobrowserRightMotion(t *testing.T) {
	dx, dy, zLift, scale, alpha := damageFloaterPlacement(damageFloaterNormal, 0.5)
	if math.Abs(dx-2) > 0.001 {
		t.Fatalf("dx = %.3f, want robr right drift 2.000", dx)
	}
	if math.Abs(dy) > 0.001 {
		t.Fatalf("dy = %.3f, want no Y drift for robr right motion", dy)
	}
	wantZ := 2 + math.Sin(-math.Pi/2+math.Pi*(0.5+0.5*1.5))*5
	if math.Abs(zLift-wantZ) > 0.001 {
		t.Fatalf("zLift = %.3f, want %.3f", zLift, wantZ)
	}
	if math.Abs(scale-2) > 0.001 || math.Abs(alpha-0.5) > 0.001 {
		t.Fatalf("scale/alpha = %.3f/%.3f, want 2.000/0.500", scale, alpha)
	}
}

func TestDamageFloaterPlacementUsesRobrowserComboMotion(t *testing.T) {
	dx, dy, zLift, scale, alpha := damageFloaterPlacement(damageFloaterCombo, 0.1)
	if dx != 0 || dy != 0 {
		t.Fatalf("combo drift = %.3f,%.3f, want stationary over target", dx, dy)
	}
	if math.Abs(zLift-7.1) > 0.001 {
		t.Fatalf("combo zLift = %.3f, want 7.100", zLift)
	}
	if math.Abs(scale-3.5) > 0.001 || math.Abs(alpha-0.9) > 0.001 {
		t.Fatalf("combo scale/alpha = %.3f/%.3f, want 3.500/0.900", scale, alpha)
	}
}

func TestDamageFloaterPlacementUsesRobrowserRecoveryAndMissMotion(t *testing.T) {
	_, _, healZ, healScale, _ := damageFloaterPlacement(damageFloaterRecoveryHP, 0.6)
	if math.Abs(healZ-3) > 0.001 || math.Abs(healScale-0.8) > 0.001 {
		t.Fatalf("heal z/scale = %.3f/%.3f, want 3.000/0.800", healZ, healScale)
	}

	_, _, missZ, missScale, _ := damageFloaterPlacement(damageFloaterMiss, 0.5)
	if math.Abs(missZ-7) > 0.001 || math.Abs(missScale-0.5) > 0.001 {
		t.Fatalf("miss z/scale = %.3f/%.3f, want 7.000/0.500", missZ, missScale)
	}
}

func TestDamageFloaterProgressUsesAnimationDuration(t *testing.T) {
	start := time.Unix(10, 0)
	floater := damageFloater{
		starts:   start,
		expires:  start.Add(200 * time.Millisecond),
		duration: 3000 * time.Millisecond,
	}
	if got := damageFloaterProgress(floater, start.Add(1500*time.Millisecond)); math.Abs(got-0.5) > 0.001 {
		t.Fatalf("progress = %.3f, want 0.500", got)
	}
}

func TestSiegeHidesCombatDamageButKeepsMissAndRecovery(t *testing.T) {
	for _, kind := range []damageFloaterKind{damageFloaterNormal, damageFloaterCritical, damageFloaterIncoming, damageFloaterCombo} {
		if !damageFloaterHiddenInSiege(kind) {
			t.Fatalf("combat floater %d remained visible in siege", kind)
		}
	}
	for _, kind := range []damageFloaterKind{damageFloaterMiss, damageFloaterRecoveryHP, damageFloaterRecoverySP} {
		if damageFloaterHiddenInSiege(kind) {
			t.Fatalf("non-combat floater %d was hidden in siege", kind)
		}
	}
}
