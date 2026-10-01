package game

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestHitRingOriginalAnimation(t *testing.T) {
	spec, _ := worldEffectSpecForID(effectHit1)
	ring := spec.components[1]
	for _, tt := range []struct {
		frame           int
		distance, alpha float64
	}{
		{-1, 0, 0},
		{0, 0.133, 254.0 / 255},
		{4, 0.595, 254.0 / 255},
		{5, 0.693, 203.2 / 255},
		{8, 0.945, 50.8 / 255},
		{9, 1.015, 0},
	} {
		distance, alpha := hitRingAnimation(ring, time.Duration(tt.frame)*ring.frameDelay)
		if math.Abs(distance-tt.distance) > 1e-6 || math.Abs(alpha-tt.alpha) > 1e-6 {
			t.Errorf("frame %d: distance/alpha = %g/%g, want %g/%g", tt.frame, distance, alpha, tt.distance, tt.alpha)
		}
	}
	if _, alpha := hitRingAnimation(ring, ring.duration); alpha != 0 {
		t.Fatal("ring should expire before the 300ms sparks")
	}
}

func TestHitRingGeometryAndCapturedFacing(t *testing.T) {
	spec, _ := worldEffectSpecForID(effectHit1)
	ring := spec.components[1]
	starts := time.Unix(100, 0)
	for _, tt := range []struct {
		dir  int
		axis modelPoint3
	}{
		{0, modelPoint3{z: 1}},
		{2, modelPoint3{x: -1}},
		{4, modelPoint3{z: -1}},
		{6, modelPoint3{x: 1}},
	} {
		world := worldstate.New()
		world.Actors[42] = worldstate.Actor{ID: 42, Dir: tt.dir}
		ctx := client.Context{World: world, Resources: &res.Manager{}}
		mode := &WorldMode{textures: map[string]*render.Image{"__effect_ring_blue": render.NewImage(64, 64)}}
		if !mode.addWorldEffectAt(ctx, effectHit1, 42, starts) {
			t.Fatal("failed to add regular hit")
		}
		mode.drawWorldEffects(render.NewFrame(800, 600), ctx, sceneProjection{}, starts)
		// Turning after the hit must not rotate the cylinder mid-flight.
		world.Actors[42] = worldstate.Actor{ID: 42, Dir: (tt.dir + 2) % 8}
		frame := render.NewFrame(800, 600)
		mode.drawFuncEffect(frame, ctx, sceneProjection{}, mode.worldEffects[0], ring, 1, 10, 20, 3, 0, starts)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 {
			t.Fatalf("direction %d: got %d draws, want a cylinder", tt.dir, commands.Len())
		}
		command := commands.Index(0)
		vertices := command.FieldByName("Vertices")
		if vertices.Len() != 22 || command.FieldByName("Indices").Len() != 60 {
			t.Fatal("expected ten cylinder faces")
		}
		point := func(i int) modelPoint3 {
			v := vertices.Index(i)
			return modelPoint3{x: v.FieldByName("X").Float(), y: v.FieldByName("Y").Float(), z: v.FieldByName("Z").Float()}
		}
		var bottom, top modelPoint3
		for i := 0; i < 10; i++ {
			bottom = add3(bottom, mul3(point(2*i), 0.1))
			top = add3(top, mul3(point(2*i+1), 0.1))
		}
		// The origin, like the facing, stays at the captured impact position.
		wantBottom := add3(modelPoint3{x: .5, y: 2.07, z: .5}, mul3(tt.axis, 0.133))
		wantTop := add3(wantBottom, mul3(tt.axis, 0.7))
		if !modelPointNear(bottom, wantBottom, 1e-5) || !modelPointNear(top, wantTop, 1e-5) {
			t.Fatalf("direction %d: centers = %+v/%+v, want %+v/%+v", tt.dir, bottom, top, wantBottom, wantTop)
		}
		radius := sub3(point(0), bottom)
		if !modelPointNear(sub3(point(1), top), mul3(radius, 2), 1e-5) || math.Abs(dot3(radius, radius)-1) > 1e-5 {
			t.Fatal("expected aligned inner/outer radii of 1/2 cells")
		}
		if got := vertices.Index(20).FieldByName("SrcX").Float(); got != 160 {
			t.Fatalf("final U = %g, want 2.5 texture repeats", got)
		}
		options := command.FieldByName("Options")
		if render.Blend(options.FieldByName("Blend").Int()) != render.BlendSourceOver || !options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() {
			t.Fatal("ring must alpha blend, test world depth, and not write depth")
		}
	}
}

func TestHitRingCapturesFacingAtImpact(t *testing.T) {
	world := worldstate.New()
	world.Actors[42] = worldstate.Actor{ID: 42, Dir: 0}
	ctx := client.Context{World: world}
	mode := &WorldMode{}
	starts := time.Unix(100, 0)
	mode.addWorldEffectAt(ctx, effectHit1, 42, starts)
	draw := func(at time.Time) {
		mode.drawWorldEffects(render.NewFrame(800, 600), ctx, sceneProjection{}, at)
	}
	draw(starts.Add(-time.Millisecond))
	if mode.worldEffects[0].hasActorTransform {
		t.Fatal("facing captured before the scheduled impact")
	}
	world.Actors[42] = worldstate.Actor{ID: 42, Dir: 2, X: 12, Y: 14}
	draw(starts)
	if effect := mode.worldEffects[0]; !effect.hasActorTransform || effect.actorDirection != 2 {
		t.Fatalf("impact facing = %d, want current target direction 2", effect.actorDirection)
	}
	wantOrigin := modelPoint3{x: 12.5, y: .07, z: 14.5}
	if mode.worldEffects[0].actorOrigin != wantOrigin {
		t.Fatal("origin captured before the scheduled impact")
	}
	world.Actors[42] = worldstate.Actor{ID: 42, Dir: 4, X: 30, Y: 40}
	draw(starts.Add(50 * time.Millisecond))
	if effect := mode.worldEffects[0]; effect.actorDirection != 2 || effect.actorOrigin != wantOrigin {
		t.Fatal("ring moved or turned with the actor after impact")
	}
}

func TestHitRingDrawsWithOldROData(t *testing.T) {
	manager := realDataManager(t)
	world := worldstate.New()
	world.Actors[42] = worldstate.Actor{ID: 42, X: 10, Y: 20}
	ctx := client.Context{World: world, Resources: manager}
	mode := &WorldMode{}
	starts := time.Unix(100, 0)
	mode.addWorldEffectAt(ctx, effectHit1, 42, starts)
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 0)
	for _, elapsed := range []time.Duration{0, 100 * time.Millisecond, 200 * time.Millisecond} {
		frame := render.NewFrame(800, 600)
		mode.drawWorldEffects(frame, ctx, projection, starts.Add(elapsed))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		rings := 0
		for i := 0; i < commands.Len(); i++ {
			if commands.Index(i).FieldByName("Vertices").Len() == 22 {
				rings++
			}
		}
		if want := elapsed < 150*time.Millisecond; (rings == 1) != want {
			t.Fatalf("elapsed %s: rings = %d, want visible=%t", elapsed, rings, want)
		}
	}
}
