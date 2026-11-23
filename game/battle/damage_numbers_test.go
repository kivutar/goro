package battle

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/world"
)

func TestDamageNumberFollowsClassicArc(t *testing.T) {
	// CMsgEffect MET_NUMBER at tick 30: drift 0.8 * (30/3 + 3) client units,
	// 15 - 8 + 30*(2 - 1) - 4/3 units up, zoom at its 1.2 floor, alpha 148.
	dx, dy, up, zoom, alpha := damageFloaterPlacement(FloaterNormal, 0, 30)
	if math.Abs(dx-0.8*13/5) > 1e-9 || math.Abs(dy+0.8*13/5) > 1e-9 {
		t.Fatalf("drift = %.3f,%.3f", dx, dy)
	}
	if want := (7 + 30 - 4.0/3) / 5; math.Abs(up-want) > 1e-9 {
		t.Fatalf("up = %.3f, want %.3f", up, want)
	}
	if zoom != 1.2 || math.Abs(alpha-(250-3.4*30)/255) > 1e-9 {
		t.Fatalf("zoom/alpha = %.3f/%.3f", zoom, alpha)
	}
	// It starts five times larger and falls back below its start by tick 70.
	if _, _, _, start, _ := damageFloaterPlacement(FloaterNormal, 0, 0); start != 5 {
		t.Fatalf("initial zoom = %.2f, want 5", start)
	}
	if _, _, end, _, _ := damageFloaterPlacement(FloaterNormal, 0, 70); end >= (7-4.0/3)/5 {
		t.Fatalf("number did not fall back: up %.3f", end)
	}
}

func TestDamageNumberDriftFollowsTargetFacing(t *testing.T) {
	// Facing is stored as dir*45+180 degrees; each quadrant drifts diagonally.
	want := map[int][2]float64{0: {1, -1}, 2: {1, 1}, 4: {-1, 1}, 6: {-1, -1}}
	for dir, sign := range want {
		x, z := damageFloaterDrift(dir, 1)
		if x != sign[0] || z != sign[1] {
			t.Fatalf("dir %d drift = %v,%v, want %v", dir, x, z, sign)
		}
	}
}

func TestDamageFloaterLifetimesUseClassicTicks(t *testing.T) {
	for kind, ticks := range map[FloaterKind]int{FloaterNormal: 70, FloaterCritical: 70, FloaterMiss: 80, FloaterCombo: 120, FloaterRecoveryHP: 120} {
		if got := DamageFloaterDuration(kind); got != time.Duration(ticks)*24*time.Millisecond {
			t.Fatalf("kind %d lifetime = %s, want %d ticks of 24ms", kind, got, ticks)
		}
	}
}

func TestComboTotalGrowsQuicklyAndRises(t *testing.T) {
	if z := damageFloaterComboZoom(0); z != .5 {
		t.Fatalf("initial combo zoom = %.2f, want 0.5", z)
	}
	if z := damageFloaterComboZoom(10); z != 3 {
		t.Fatalf("combo zoom after 240ms = %.2f, want 3", z)
	}
	_, _, up, _, alpha := damageFloaterPlacement(FloaterCombo, 0, 100)
	if math.Abs(up-(30+10)/5.0) > 1e-9 || math.Abs(alpha-(250-80)/255.0) > 1e-9 {
		t.Fatalf("combo up/alpha = %.3f/%.3f", up, alpha)
	}
}

func TestDamageNumbersAreWhiteAndMissesRedOnPlayers(t *testing.T) {
	hit := network.ActorActionNotify{Damage: 10, Action: 0}
	if _, kind, c := actionDamageFloater(hit, world.Actor{Job: 0}, true); kind != FloaterNormal || c != DamageFloaterWhite {
		t.Fatalf("damage on a player = %v %v, want white", kind, c)
	}
	miss := network.ActorActionNotify{Action: 0}
	if _, _, c := actionDamageFloater(miss, world.Actor{Job: 0}, true); c != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("miss on a player = %v, want red", c)
	}
	if _, _, c := actionDamageFloater(miss, world.Actor{Job: 1002}, true); c != DamageFloaterWhite {
		t.Fatalf("miss on a monster = %v, want white", c)
	}
}

func TestNewComboTotalReplacesThePreviousOne(t *testing.T) {
	start := time.Unix(10, 0)
	m := &System{DamageFloaters: []Floater{
		{ActorID: 1, Kind: FloaterCombo, Starts: start, Expires: start.Add(time.Second)},
		{ActorID: 1, Kind: FloaterCombo, Starts: start.Add(200 * time.Millisecond), Expires: start.Add(time.Second)},
	}}
	if m.damageFloaterReplaced(m.DamageFloaters[0], start.Add(100*time.Millisecond)) {
		t.Fatal("first total hidden before the next one started")
	}
	if !m.damageFloaterReplaced(m.DamageFloaters[0], start.Add(300*time.Millisecond)) {
		t.Fatal("first total still shown after the next one started")
	}
}

func TestSiegeHidesCombatDamageButKeepsMissAndRecovery(t *testing.T) {
	for _, kind := range []FloaterKind{FloaterNormal, FloaterCritical, FloaterCombo} {
		if !damageFloaterHiddenInSiege(kind) {
			t.Fatalf("combat floater %d remained visible in siege", kind)
		}
	}
	for _, kind := range []FloaterKind{FloaterMiss, FloaterRecoveryHP, FloaterRecoverySP} {
		if damageFloaterHiddenInSiege(kind) {
			t.Fatalf("non-combat floater %d was hidden in siege", kind)
		}
	}
}
