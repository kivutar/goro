package effects

import (
	"image/color"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

func TestAuraPreferenceUpdatesVisibleActorsWithoutChangingOtherEffects(t *testing.T) {
	ctx := testutil.Level99AuraTestContext(99, false)
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, Level: 99, HasLevel: true}
	now := time.Unix(10, 0)
	m := &System{}
	m.AddCellLifetime(ctx, Heal, 300, 12, 20, now, time.Minute, false)
	for _, step := range []struct {
		simple, less bool
		wantFull     bool
	}{
		{false, false, true},
		{true, false, false},
		{false, false, true},
		{false, true, false},
		{true, true, false},
	} {
		ctx.Session.SimpleAura = step.simple
		ctx.Session.LessEffects = step.less
		m.SyncLevel99AuraEffects(ctx, now)
		for _, actorID := range []uint32{2000000, 300} {
			for _, effectID := range level99AuraAllEffects {
				want := effectID == Level99Bubble
				if step.wantFull {
					want = effectID != Level99Bubble
				}
				if got := m.HasActive(effectID, actorID, now); got != want {
					t.Fatalf("simple=%t less=%t actor=%d effect=%d: active=%t, want %t", step.simple, step.less, actorID, effectID, got, want)
				}
			}
		}
		if !m.HasActive(Heal, 300, now) {
			t.Fatal("changing aura preference removed a skill effect")
		}
	}
}

func TestLevel99AuraEffectsFollowClassicJobRules(t *testing.T) {
	ctx := testutil.Level99AuraTestContext(99, false)
	cases := []struct {
		job          int
		simple, want []int
	}{
		{0, []int{Level99Bubble}, Level99AuraNormalEffects},
		{23, []int{Level99Aura1}, Level99AuraNormalEffects},
		{4008, []int{Level99Aura1}, Level99AuraNormalEffects},
		{1324, nil, nil},
		{1640, []int{Level99Aura1}, []int{Level99Aura1}},
		{1700, []int{Level99Aura1}, Level99AuraNormalEffects},
	}
	for _, tc := range cases {
		ctx.Session.SimpleAura = false
		if got := level99AuraEffectIDs(ctx, tc.job); !slices.Equal(got, tc.want) {
			t.Fatalf("job %d full aura = %v, want %v", tc.job, got, tc.want)
		}
		ctx.Session.SimpleAura = true
		if got := level99AuraEffectIDs(ctx, tc.job); !slices.Equal(got, tc.simple) {
			t.Fatalf("job %d simple aura = %v, want %v", tc.job, got, tc.simple)
		}
	}
}

func TestLevel99BubblesRiseDriftAndRestart(t *testing.T) {
	effect := Instance{EffectID: Level99Aura1, ActorID: 2000000, Starts: time.Unix(10, 0)}
	s := newLevel99BubbleState(effect)
	if len(s.groups) != 8 {
		t.Fatalf("groups = %d, want two primitives of four", len(s.groups))
	}
	for _, g := range s.groups {
		for _, p := range g.particles {
			if p.X != 0 || p.Z != 0 || p.Y < 0 || p.Y >= 100 {
				t.Fatalf("initial particle = %+v, want at the feet, delayed by up to 100 units", p)
			}
		}
	}
	// Particles move only while risen (negative Y) and restart past -30.
	maxSpread, minY := 0.0, 0.0
	for frame := 0; frame < 2000; frame++ {
		s.advance(effect, frame)
		for _, g := range s.groups {
			for _, p := range g.particles {
				if p.Y >= 0 && (p.X != 0 || p.Z != 0) {
					t.Fatalf("frame %d: hidden particle drifted: %+v", frame, p)
				}
				minY = math.Min(minY, p.Y)
				maxSpread = math.Max(maxSpread, math.Hypot(p.X, p.Z))
			}
		}
	}
	if minY < -30.2 {
		t.Fatalf("particle rose to %.2f, want restart after -30", minY)
	}
	// The random walk carries bubbles several cells away (a cell is 5 units).
	if maxSpread < 10 {
		t.Fatalf("maximum drift = %.2f client units, want a wide spread", maxSpread)
	}
}

func TestLevel99AuraBandMatchesClassicGeometry(t *testing.T) {
	texture := render.NewImage(8, 8)
	var vertices []render.Vertex3D
	var indices []uint16
	tint := color.RGBA{R: 100, G: 100, B: 255, A: 180}
	appendLevel99AuraBand(&vertices, &indices, texture, scene.Point3{}, 0, 200, 1, tint)
	// 22 points 15 degrees apart over 315 degrees give 21 quads.
	if len(vertices) != 21*4 || len(indices) != 21*6 {
		t.Fatalf("vertices=%d indices=%d, want 21 quads", len(vertices), len(indices))
	}
	base := vertices[0]
	if r := math.Hypot(float64(base.X), float64(base.Z)); math.Abs(r-3.9/5) > 1e-6 || base.Y != 0 {
		t.Fatalf("band 0 base radius=%.4f y=%.4f, want 3.9 client units on the ground", r, base.Y)
	}
	// The middle segment reaches the full 15-unit height at 55 degrees.
	middle := vertices[9*4+2]
	if math.Abs(float64(middle.Y)-15*math.Sin(55*math.Pi/180)/5) > 1e-4 {
		t.Fatalf("middle top height = %.4f", middle.Y)
	}
}
