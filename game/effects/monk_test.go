package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestMonkAttackBlendModes(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = &res.Manager{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	for _, tc := range []struct {
		id, tick int
		blends   []render.Blend
	}{
		{TeiHit1, 45, []render.Blend{render.BlendLighter}},
		{TeiHit1X, 45, []render.Blend{render.BlendLighter}},
		{HitLine, 30, []render.Blend{render.BlendLighter}},
		{TripleAttack, 10, []render.Blend{render.BlendSourceOver, render.BlendLighter}},
		{AbsorbSpirits, 30, []render.Blend{render.BlendLighter}},
		{Gumgang2, 30, []render.Blend{render.BlendSourceOver}},
		{db.SkillEffectSteelBodyRing, 30, []render.Blend{render.BlendSourceOver}},
	} {
		spec, _ := Lookup(tc.id)
		m := &System{textures: map[string]*render.Image{
			"__effectfile_" + spec.Components[0].TextureFile + "|blackkey=true": render.NewImage(8, 8),
		}}
		m.AddBetween(ctx, tc.id, 300, 2000000, start)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != len(tc.blends) {
			t.Fatalf("effect %d: got %d draws, want %d", tc.id, commands.Len(), len(tc.blends))
		}
		for i, want := range tc.blends {
			got := render.Blend(commands.Index(i).FieldByName("Options").FieldByName("Blend").Int())
			if got != want {
				t.Fatalf("effect %d layer %d: blend %d, want %d", tc.id, i, got, want)
			}
		}
	}
}

func TestMonkAttackEffectsRenderWithClientAssets(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	starts := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	for _, tc := range []struct{ id, tick int }{
		{TeiHit1, 45}, {TeiHit1X, 45}, {TeiHit2, 49},
		{TripleAttack, 10}, {HitLine, 30}, {Tanji, 24},
		{AbsorbSpirits, 30}, {Gumgang2, 30}, {db.SkillEffectSteelBodyRing, 30},
	} {
		m := &System{}
		if !m.AddBetween(ctx, tc.id, 300, 2000000, starts) {
			t.Fatalf("missing effect %d", tc.id)
		}
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		if value.FieldByName("worldBillboards").Len()+value.FieldByName("worldCommands").Len() == 0 || len(m.textureMiss) != 0 {
			t.Fatalf("effect %d drew nothing or has missing assets: %v", tc.id, m.textureMiss)
		}
	}
}

func TestInvestigateSpriteRendersWithOldROAssets(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &System{}
	starts := time.Now()
	m.AddBetween(ctx, Chimto, 300, 2000000, starts)
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	for _, elapsed := range []time.Duration{0, 100 * time.Millisecond, 200 * time.Millisecond} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts.Add(elapsed), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("Investigate has no sprite at %s", elapsed)
		}
	}
}

func TestGumgangFlashesMatchOriginalUpdateLoop(t *testing.T) {
	for i, pulse := range []int{70, 130, 190, 250} {
		counter, cycle := 10, 0
		for frame := 1; frame <= 200; frame++ {
			counter++
			if counter < 10 {
				pulse += 25
			} else {
				pulse -= 12
				if pulse < 12 {
					counter, pulse = 0, 0
					cycle++
				}
			}
			alpha, gotCycle := gumgangFlash(frame, i)
			if math.Abs(alpha-float64(pulse)/255) > 1e-9 || gotCycle != cycle {
				t.Fatalf("flash %d frame %d: alpha=%g cycle=%d, want %d/255 cycle=%d", i, frame, alpha, gotCycle, pulse, cycle)
			}
		}
	}
}

func TestThrowSpiritSphereFliesFromCasterAndDoesNotReturn(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	target := ctx.World.Actors[300]
	target.X = 20
	ctx.World.Actors[300] = target
	makeFlight := func() *shieldFlight {
		m := &System{}
		m.AddBetween(ctx, Tanji, 300, 2000000, time.Unix(100, 0))
		effect := m.Active[0]
		if effect.ActorID != 2000000 || effect.TargetID != 300 {
			t.Fatalf("reversed sphere ownership: %+v", effect)
		}
		return effect.Shield
	}
	flight := makeFlight()
	origin := flight.Images[0].Position
	flight.Advance(0, origin)
	if flight.Images[0].Position != origin || flight.Images[0].Alpha != 5 {
		t.Fatal("sphere moved during its 15-tick startup")
	}
	flight.Advance(10, origin)
	if math.Abs(flight.Images[0].Position.X-origin.X-4) > 1e-9 || flight.Images[0].Alpha != 255 {
		t.Fatalf("sphere at process 10 = %+v", flight.Images)
	}
	for i := 1; i < 4; i++ {
		if flight.Images[i].Alpha <= 0 || flight.Images[i].Position.X >= flight.Images[i-1].Position.X {
			t.Fatalf("sphere trail missing or ahead of projectile: %+v", flight.Images)
		}
	}
	flight.Advance(26, origin)
	if flight.Phase != 2 || flight.Images[0].Position.X < 20.5 {
		t.Fatalf("sphere did not pass the target: %+v", flight)
	}
	x := flight.Images[0].Position.X
	flight.Advance(60, origin)
	for _, image := range flight.Images {
		if image.Alpha != 0 || image.Position.X <= x {
			t.Fatalf("sphere returned or failed to fade: %+v", flight)
		}
	}
	// Skipping draws must preserve the native tick simulation.
	skipped := makeFlight()
	skipped.Advance(60, origin)
	if *flight != *skipped {
		t.Fatal("sphere flight depends on rendering rate")
	}
}

func TestMonkParticlePhasesAndGeometry(t *testing.T) {
	effect := Instance{Starts: time.Unix(100, 0), ProjectileDirection: scene.Point3{X: 1}}
	for _, id := range []int{TeiHit1, TeiHit1X} {
		effect.EffectID = id
		for i := 0; i < 96; i++ {
			for _, tc := range []struct{ age, alpha int }{{10, 0}, {11, 15}, {20, 150}, {21, 145}, {50, 0}} {
				frame := tc.age + 9 + HitEffectRandom(effect, i*3, 25)
				corners, tint := monkRay(effect, i, frame)
				if int(tint.A) != tc.alpha {
					t.Fatalf("ray %d age %d alpha=%d, want %d", i, tc.age, tint.A, tc.alpha)
				}
				for _, p := range corners {
					if !scene.IsFinite(p.X) || !scene.IsFinite(p.Y) || !scene.IsFinite(p.Z) {
						t.Fatal("non-finite ray geometry")
					}
				}
			}
		}
	}
	effect.EffectID = TripleAttack
	for i := 0; i < 8; i++ {
		for _, tc := range []struct{ age, alpha int }{{0, 0}, {1, 8}, {5, 40}, {6, 36}, {15, 0}} {
			frame := tc.age - 1 + i/4 + HitEffectRandom(effect, i*6, 20)
			from, to, alpha := tripleAttackLine(effect, i, frame)
			if alpha != tc.alpha || to.X <= from.X || to.Y != from.Y || from.Y < 1.6 || from.Y >= 2.4 {
				t.Fatalf("triple attack line %d age %d: %v -> %v, alpha %d", i, tc.age, from, to, alpha)
			}
		}
	}
	previous := 0.0
	for age := 1; age < 100; age++ {
		p := hitLinePoint(age, 10, 0)
		if p.X < previous || p.Y < -1e-9 || p.Z != 0 {
			t.Fatalf("Combo Finish trail doubled back or went underground: %v", p)
		}
		previous = p.X
	}
}

func TestMonkParticleCounts(t *testing.T) {
	for _, tc := range []struct {
		id, count int
		adapter   EffectFuncAdapter
	}{
		{TeiHit1, 80, effectFuncMonkRays},
		{TeiHit1X, 96, effectFuncMonkRays},
		{TeiHit2, 20, effectFuncImpactParticles},
		{BackStab, 80, effectFuncImpactParticles},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 || spec.Components[0].FuncAdapter != tc.adapter || spec.Components[0].Duplicate != tc.count {
			t.Fatalf("effect %d must render %d native particles using adapter %d", tc.id, tc.count, tc.adapter)
		}
		if tc.id != BackStab && len(spec.Sfx) != 0 {
			t.Fatalf("impact effect %d must not add a skill activation sound", tc.id)
		}
	}
}

func TestMonkBandsMatchOriginalPhases(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = &res.Manager{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	for _, tc := range []struct {
		id, tick, alpha int
		radii           [4]float64
	}{
		{AbsorbSpirits, 14, 75, [4]float64{.92, 1, 1, 1.04}},
		{AbsorbSpirits, 39, 73, [4]float64{.92, 1, 1, 1.04}},
		{AbsorbSpirits, 77, 0, [4]float64{}},
		{Gumgang2, 0, 20, [4]float64{.22, .42, .62, .82}},
		{Gumgang2, 9, 200, [4]float64{.4, .6, .8, 1}},
		{Gumgang3, 9, 100, [4]float64{.8, 1, 1.2, 1.4}},
		{Gumgang2, 109, 0, [4]float64{}},
		{db.SkillEffectSteelBodyRing, 9, 200, [4]float64{.4, .6, .8, 1}},
	} {
		spec, _ := Lookup(tc.id)
		m := &System{textures: map[string]*render.Image{
			"__effectfile_" + spec.Components[0].TextureFile + "|blackkey=true": render.NewImage(8, 8),
		}}
		ctx.World.Player.X = 10
		m.AddAt(ctx, tc.id, 2000000, start)
		// All four bands must follow the caster, including after movement.
		ctx.World.Player.X = 15
		origin := EffectWorldAnchor(ctx, m.Active[0], start)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tc.alpha == 0 {
			if commands.Len() != 0 {
				t.Fatalf("effect %d tick %d did not fade out", tc.id, tc.tick)
			}
			continue
		}
		if commands.Len() != 1 {
			t.Fatalf("effect %d tick %d: draws=%d, want four bands in one draw", tc.id, tc.tick, commands.Len())
		}
		vertices := commands.Index(0).FieldByName("Vertices")
		if vertices.Len() != 320 {
			t.Fatalf("effect %d: vertices=%d, want four 20-segment bands", tc.id, vertices.Len())
		}
		for i, radius := range tc.radii {
			v := vertices.Index(i*80 + 3) // bottom of the first segment
			x, z := v.FieldByName("X").Float()-origin.X, v.FieldByName("Z").Float()-origin.Z
			if math.Abs(math.Hypot(x, z)-radius) > 1e-5 || math.Abs(v.FieldByName("Y").Float()-origin.Y) > 1e-5 {
				t.Fatalf("effect %d tick %d band %d: bottom is misplaced", tc.id, tc.tick, i)
			}
			if math.Abs(v.FieldByName("ColorA").Float()-float64(tc.alpha)/255) > 1e-6 {
				t.Fatalf("effect %d tick %d: incorrect fade", tc.id, tc.tick)
			}
		}
	}
}
