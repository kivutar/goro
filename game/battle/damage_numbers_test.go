package battle

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
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
	if m.damageFloaterReplaced(0, start.Add(100*time.Millisecond)) {
		t.Fatal("first total hidden before the next one started")
	}
	if !m.damageFloaterReplaced(0, start.Add(300*time.Millisecond)) {
		t.Fatal("first total still shown after the next one started")
	}
	// Totals queued for the same instant: the later one wins.
	m.DamageFloaters[1].Starts = start
	if !m.damageFloaterReplaced(0, start) || m.damageFloaterReplaced(1, start) {
		t.Fatal("same-start totals should keep only the last queued")
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

func TestActionDamageHitsFollowClassicScheduling(t *testing.T) {
	ms := time.Millisecond
	tests := []struct {
		name   string
		action network.ActorActionNotify
		want   []damageHit
	}{
		{"single", network.ActorActionNotify{Damage: 50, HitCount: 1}, []damageHit{{0, 50, 50, damageHitSingle}}},
		{"critical", network.ActorActionNotify{Damage: 50, HitCount: 1, Action: actionAttackCritical}, []damageHit{{0, 50, 50, damageHitCritical}}},
		{"count ignored without multiple action", network.ActorActionNotify{Damage: 50, HitCount: 2}, []damageHit{{0, 50, 50, damageHitSingle}}},
		{"double attack floors each hit", network.ActorActionNotify{Damage: 101, HitCount: 2, Action: actionAttackMultiple}, []damageHit{
			{0, 50, 50, damageHitMany}, {200 * ms, 50, 100, damageHitMany},
		}},
		{"dual wield", network.ActorActionNotify{Damage: 40, LeftDamage: 15, HitCount: 1}, []damageHit{
			{0, 40, 40, damageHitSingle}, {350 * ms, 15, 55, damageHitSingle},
		}},
		{"dual wield double attack", network.ActorActionNotify{Damage: 40, LeftDamage: 15, HitCount: 2, Action: actionAttackMultiple}, []damageHit{
			{0, 20, 20, damageHitMany}, {100 * ms, 20, 40, damageHitMany}, {350 * ms, 15, 55, damageHitMany},
		}},
		{"bash", network.ActorActionNotify{SkillID: db.SkillSMBash, Damage: 300, HitCount: 1, Action: actionSkill}, []damageHit{{0, 300, 300, damageHitMany2}}},
		{"turn kick", network.ActorActionNotify{SkillID: db.SkillTKTurnkick, Damage: 90, HitCount: 1, Action: actionSkill}, []damageHit{{300 * ms, 90, 90, damageHitMany2}}},
		{"multi-hit skill sums floored hits", network.ActorActionNotify{SkillID: db.SkillKNPierce, Damage: 10, HitCount: 3, Action: actionSkill}, []damageHit{
			{0, 3, 3, damageHitMany}, {280 * ms, 3, 6, damageHitMany}, {560 * ms, 3, 9, damageHitMany},
		}},
		{"tiger fist shows a total", network.ActorActionNotify{SkillID: db.SkillChTigerfist, Damage: 70, HitCount: 1, Action: actionSkill}, []damageHit{{0, 70, 70, damageHitMany}}},
		{"splash shows one hit", network.ActorActionNotify{SkillID: db.SkillKNBowlingbash, Damage: 90, HitCount: 3, Action: actionSplash}, []damageHit{{0, 30, 30, damageHitSingle}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := actionDamageHits(tc.action); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("hits = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestBlueFlashPlacement(t *testing.T) {
	_, _, up, zoom, alpha := damageFloaterPlacement(FloaterBlue, 0, 10)
	if math.Abs(up-31.0/5) > 1e-9 || zoom != 2 || math.Abs(alpha-210.0/255) > 1e-9 {
		t.Fatalf("blue flash = up %v zoom %v alpha %v", up, zoom, alpha)
	}
	if _, _, _, _, alpha := damageFloaterPlacement(FloaterBlue, 0, 120); alpha != 0 {
		t.Fatalf("blue flash alpha at end = %v, want 0", alpha)
	}
}
