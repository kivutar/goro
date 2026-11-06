package maps

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/world"
)

func TestRSWEffectWorldPositionUsesMapOffset(t *testing.T) {
	gnd := &res.GND{Width: 300, Height: 200}
	effect := res.RSWEffect{Position: res.RSWVector3{X: 12, Y: -3, Z: 34}}
	x, y, z := rswEffectWorldPosition(gnd, effect)
	if x != 312 || y != 234 || z != -3 {
		t.Fatalf("rswEffectWorldPosition = %.1f, %.1f, %.1f; want 312, 234, -3", x, y, z)
	}
}

func TestRSWEmitterUsesNativeIntervalsAndVisibility(t *testing.T) {
	now := time.Unix(10, 0)
	source := res.RSWEffect{ID: effects.Smoke, Delay: 40}
	emitter := rswEffectEmitter{}
	emitter.emit(source, 0, now, false)
	if emitter.interval != 40*db.EffectFrameDuration {
		t.Fatalf("interval = %s", emitter.interval)
	}
	if delay := emitter.next.Sub(now); delay < 0 || delay >= 24*db.EffectFrameDuration {
		t.Fatalf("startup delay = %s", delay)
	}
	if !emitter.emit(source, 0, emitter.next.Add(-time.Nanosecond), true).IsZero() {
		t.Fatal("emitted early")
	}
	if emitter.emit(source, 0, emitter.next, true).IsZero() {
		t.Fatal("did not emit on schedule")
	}
	due := emitter.next
	if !emitter.emit(source, 0, due, false).IsZero() {
		t.Fatal("emitted out of range")
	}
	resumed := due.Add(time.Hour)
	if emitter.emit(source, 0, resumed, true).IsZero() || !emitter.emit(source, 0, resumed, true).IsZero() {
		t.Fatal("did not resume with exactly one emission")
	}
	if emitter.next != resumed.Add(emitter.interval) {
		t.Fatal("replaying missed emissions")
	}
}

func TestRSWFireflyEmittersHaveIndependentPeriods(t *testing.T) {
	now := time.Unix(10, 0)
	source := res.RSWEffect{ID: effects.Firefly, Delay: 500}
	periods := map[time.Duration]bool{}
	for index := 0; index < 32; index++ {
		emitter := rswEffectEmitter{}
		emitter.emit(source, index, now, false)
		if emitter.interval < 475*db.EffectFrameDuration || emitter.interval >= 525*db.EffectFrameDuration {
			t.Fatalf("period = %s", emitter.interval)
		}
		if delay := emitter.next.Sub(now); delay < 0 || delay > 525*db.EffectFrameDuration {
			t.Fatalf("phase = %s", delay)
		}
		periods[emitter.interval] = true
	}
	if len(periods) < 2 {
		t.Fatal("fireflies are synchronized")
	}
}

func TestMapEffectComponentUsesWorldSizedSpriteWithoutChangingPlacement(t *testing.T) {
	component := effects.Component{
		Kind:       effects.Component3D,
		SpriteFile: "torch_01",
		PosX:       0.1,
		PosY:       -0.2,
		PosZ:       0.8,
	}
	got := MapEffectComponent(component)
	if !got.WorldSizedSprite {
		t.Fatalf("mapEffectComponent did not mark sprite as world sized: %+v", got)
	}
	if got.PosX != 0.1 || got.PosY != -0.2 || got.PosZ != 0.8 {
		t.Fatalf("mapEffectComponent = %+v", got)
	}
}

func TestReferenceRSWMapEffectIDsHaveSpecs(t *testing.T) {
	for _, effectID := range []int{
		effects.Smoke,
		effects.Firefly,
		effects.Torch,
		effects.Bubble,
		effects.DragonSmoke,
		effects.Banjjakii,
		effects.MapPillar,
		effects.MapPillar2,
		effects.MapPillar3,
		effects.MapPillar4,
		effects.TorchRed,
		effects.TorchGreen,
		effects.TorchPurple,
		effects.MapGhost,
		effects.Glow1,
		effects.Glow2,
		effects.Glow4,
		effects.BubbleDrop,
		effects.Rainbow,
	} {
		if _, ok := effects.Lookup(effectID); !ok {
			t.Fatalf("missing RSW map effect spec %d", effectID)
		}
	}
}

func TestMapPillarEffectSpecsMatchReferenceVariants(t *testing.T) {
	type variant struct {
		name        string
		id          int
		texture     string
		radiusStart float64
		alpha       float64
		tint        color.RGBA
	}
	for _, tc := range []variant{
		{"EF_MAPPILLAR", effects.MapPillar, "ring_blue", 2, 50.0 / 255.0, color.RGBA{R: 100, G: 100, B: 255, A: 255}},
		{"EF_MAPPILLAR2", effects.MapPillar2, "ring_blue", 11, 70.0 / 255.0, color.RGBA{R: 100, G: 100, B: 255, A: 255}},
		{"EF_MAPPILLAR3", effects.MapPillar3, "magic_green", 2, 50.0 / 255.0, color.RGBA{R: 100, G: 255, B: 100, A: 255}},
		{"EF_MAPPILLAR4", effects.MapPillar4, "ring_red", 2, 50.0 / 255.0, color.RGBA{R: 255, G: 100, B: 100, A: 255}},
	} {
		spec, ok := effects.Lookup(tc.id)
		if !ok || spec.Duration != 999*db.EffectFrameDuration || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != effects.ComponentFUNC || component.FuncName != "MapPillar" || component.FuncAdapter != effects.EffectFuncMapPillar || component.TextureName != tc.texture || component.Duration != 999*db.EffectFrameDuration {
			t.Fatalf("%s component timing/texture = %+v", tc.name, component)
		}
		if component.AlphaMax != tc.alpha || component.Color != tc.tint || component.BlendMode != 2 || !component.BlendAdditive || component.AttachedEntity {
			t.Fatalf("%s component alpha/color/blend = %+v", tc.name, component)
		}
		if component.BottomSize != tc.radiusStart*0.2 || component.Height != 24 {
			t.Fatalf("%s component geometry = %+v", tc.name, component)
		}
	}
}

func TestMapPillarBandHeightMatchesReferenceLifecycle(t *testing.T) {
	maxHeight := 24.0
	if got := effects.MapPillarBandHeight(199, maxHeight); got != 0 {
		t.Fatalf("height before grow = %.3f, want 0", got)
	}
	if got := effects.MapPillarBandHeight(245, maxHeight); math.Abs(got-maxHeight*math.Sin(45*math.Pi/180)) > 0.0001 {
		t.Fatalf("height mid grow = %.3f", got)
	}
	if got := effects.MapPillarBandHeight(291, maxHeight); got != maxHeight {
		t.Fatalf("height after grow = %.3f, want %.3f", got, maxHeight)
	}
	if got := effects.MapPillarBandHeight(800, maxHeight); math.Abs(got-23.8) > 0.0001 {
		t.Fatalf("height shrink start = %.3f, want 23.8", got)
	}
	if got := effects.MapPillarBandHeight(919, maxHeight); got != 0 {
		t.Fatalf("height after shrink = %.3f, want 0", got)
	}
}

func TestRSWEmitterKeepsCadenceWithLateFrames(t *testing.T) {
	now := time.Unix(10, 0)
	source := res.RSWEffect{ID: effects.Smoke, Delay: 40}
	e := rswEffectEmitter{}
	e.emit(source, 0, now, false)
	first := e.next
	for i := 0; i < 100; i++ {
		due := first.Add(time.Duration(i) * 40 * db.EffectFrameDuration)
		if got := e.emit(source, 0, due.Add(10*time.Millisecond), true); got != due {
			t.Fatalf("emission %d drifted: %s, want %s", i, got, due)
		}
	}
}

func TestRSWEffectsOverlapExpireAndReset(t *testing.T) {
	ctx := client.Context{World: world.New()}
	ctx.World.GND = &res.GND{}
	ctx.World.RSW = &res.RSW{Effects: []res.RSWEffect{{ID: effects.Torch, Delay: 125, Param: [4]float32{.6, 3}}}}
	m, fx := &System{}, &effects.System{}
	frame := render.NewFrame(800, 600)
	draw := func(now time.Time) {
		frame.BeginFrame()
		m.DrawRSWEffects(frame, ctx, scene.Projection{}, now, func() *effects.System { return fx })
	}
	now := time.Unix(10, 0)
	draw(now)
	start := now
	if len(m.rswEffects[0].active) == 0 {
		start = m.rswEffects[0].next
		draw(start)
	}
	first := m.rswEffects[0].active[0]
	if first.Params != ctx.World.RSW.Effects[0].Param || first.Duration != 250*db.EffectFrameDuration {
		t.Fatalf("torch parameters/lifetime lost: %+v", first)
	}
	draw(start.Add(125 * db.EffectFrameDuration))
	if len(m.rswEffects[0].active) != 2 {
		t.Fatal("new emission replaced the previous torch")
	}
	draw(start.Add(250 * db.EffectFrameDuration))
	if active := m.rswEffects[0].active; len(active) != 2 || active[0].Starts == start {
		t.Fatalf("expired emission retained: %+v", active)
	}
	ctx.Config.Gameplay.NoEffects = true
	draw(now.Add(time.Minute))
	if m.rswEffects != nil || m.rswEffectSource != nil {
		t.Fatal("disabled effects retained emitters")
	}
	ctx.Config.Gameplay.NoEffects = false
	draw(now.Add(2 * time.Minute))
	ctx.World.RSW = &res.RSW{}
	draw(now.Add(3 * time.Minute))
	if len(m.rswEffects) != 0 {
		t.Fatal("old map's emitters survived a map change")
	}
}

func TestRSWRandomSTRUsesItsOwnLifetime(t *testing.T) {
	ctx := client.Context{World: world.New(), Resources: testutil.RealDataManager(t)}
	ctx.World.GND = &res.GND{}
	ctx.World.RSW = &res.RSW{Effects: []res.RSWEffect{{ID: effects.Pong, Delay: 300}}}
	fx := &effects.System{}
	spec, _ := effects.Lookup(effects.Pong)
	frame := render.NewFrame(800, 600)
	projection := testutil.Projection(800, 600, 0, 0, 0)
	seen := make(map[string]bool)
	for i := 0; i < 32; i++ {
		m := &System{}
		draw := func(at time.Time) {
			frame.BeginFrame()
			m.DrawRSWEffects(frame, ctx, projection, at, func() *effects.System { return fx })
		}
		draw(time.Unix(int64(10+i), 0))
		if len(m.rswEffects[0].active) == 0 {
			draw(m.rswEffects[0].next)
		}
		effect := m.rswEffects[0].active[0]
		file := effects.ResolveEffectSTRFile(spec.Components[0], effect, false)
		seen[file] = true
		str := fx.LoadWorldEffectSTR(ctx.Resources, file, "")
		if str == nil {
			t.Fatalf("missing %s", file)
		}
		want := time.Duration(float64(str.MaxKey)/float64(str.FPS)*float64(time.Second)) + 100*time.Millisecond
		if effect.Duration != want || effect.Expires.Sub(effect.Starts) != want {
			t.Fatalf("%s lifetime = %s, want %s", file, effect.Duration, want)
		}
		draw(effect.Expires.Add(-time.Nanosecond))
		if len(m.rswEffects[0].active) != 1 {
			t.Fatalf("%s expired prematurely", file)
		}
		draw(effect.Expires)
		if len(m.rswEffects[0].active) != 0 {
			t.Fatalf("%s did not expire", file)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("did not exercise all variants: %v", seen)
	}
}
