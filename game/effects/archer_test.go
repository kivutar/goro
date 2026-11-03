package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestArrowCapturesFlightEndpointsAndStaysOpaque(t *testing.T) {
	for _, targetDisappears := range []bool{false, true} {
		ctx := testutil.ArcherEffectContext()
		ctx.Resources = &res.Manager{}
		ctx.Session.NoEffects = true
		view := testutil.EffectRenderTestSprite()
		view.Billboards[scene.SpriteBillboardKey{IgnoreLayerAngles: true}] = view.Billboards[scene.SpriteBillboardKey{}]
		m := &System{EffectViews: map[string]*scene.SpriteView{`data\sprite\몬스터\skel_archer_arrow`: view}}
		release := time.Unix(100, 0)
		if !m.AddBetween(ctx, ArrowShot, 300, 2000000, release) {
			t.Fatal("arrows must remain visible with effects disabled")
		}
		projection := testutil.Projection(800, 600, 15, 20, 0)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, release.Add(-time.Millisecond), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() != 0 {
			t.Fatal("arrow appeared before release")
		}
		// The source is fixed at creation, but the target is sampled at release.
		ctx.World.Player.X = 12
		target := ctx.World.Actors[300]
		target.X = 24
		ctx.World.Actors[300] = target
		endX := 24.5
		if targetDisappears {
			delete(ctx.World.Actors, 300)
			endX = 20.5
		}
		for _, elapsed := range []time.Duration{0, 96 * time.Millisecond, 191 * time.Millisecond, 192 * time.Millisecond} {
			frame = render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, release.Add(elapsed), false)
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if elapsed == 192*time.Millisecond {
				if draws.Len() != 0 {
					t.Fatal("arrow remained visible after arrival")
				}
				continue
			}
			if draws.Len() != 1 {
				t.Fatalf("elapsed %s: got %d arrows, want one", elapsed, draws.Len())
			}
			draw := draws.Index(0)
			center := draw.FieldByName("Center")
			x := 10.5 + (endX-10.5)*float64(elapsed)/float64(192*time.Millisecond)
			if math.Abs(center.Index(0).Float()-x) > 1e-5 || math.Abs(center.Index(1).Float()-2.07) > 1e-5 || math.Abs(center.Index(2).Float()-20.5) > 1e-5 {
				t.Fatalf("elapsed %s: unexpected flight position %v", elapsed, center)
			}
			if draw.FieldByName("ColorA").Float() != 1 {
				t.Fatal("arrow must remain opaque throughout its flight")
			}
			ctx.World.Player.X = 15
			target.X = 30
			ctx.World.Actors[300] = target
		}
	}
}

func TestArrowShowerRemainsVisibleWithEffectsDisabled(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Session.NoEffects = true
	m := &System{}
	m.AddArrowShower(ctx, 2000000, 20, 20, false, time.Unix(100, 0))
	if len(m.Active) != 1 || m.Active[0].ArrowTargets == nil {
		t.Fatal("effect toggle must not remove Arrow Shower's projectiles")
	}
}
