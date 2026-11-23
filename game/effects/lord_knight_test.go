package effects

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func TestLordKnightBodyAndWeaponRendering(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	actor := ctx.World.Actors[300]
	start := time.Unix(100, 0)
	texture := render.NewImage(12, 24)
	texture.Fill(color.White)
	billboard := &scene.Billboard{Image: texture, AnchorX: 6, AnchorY: 24}
	for _, tc := range []struct {
		effect, frame, group, count int
		aura                        bool
		blend                       render.Blend
		green, blue                 float64
	}{
		{RedBody, 0, 0, 1, false, render.BlendLighter, 100.0 / 255, 100.0 / 255},
		{RedBody, 0, 1, 1, false, render.BlendSourceOver, 100.0 / 255, 100.0 / 255},
		{RedBody, 0, 2, 1, false, render.BlendSourceOver, 100.0 / 255, 100.0 / 255},
		{0, 0, 1, 1, true, render.BlendLighter, 1, 1},
		{0, 0, 0, 1, true, render.BlendSourceOver, 1, 1},
		{0, 0, 2, 1, true, render.BlendSourceOver, 1, 1},
		{PierceBody, 6, 0, 2, false, render.BlendLighter, 250.0 / 255, 100.0 / 255},
		{PierceBody, 6, 1, 1, false, render.BlendSourceOver, 250.0 / 255, 100.0 / 255},
		{PierceBody, 6, 2, 1, false, render.BlendSourceOver, 250.0 / 255, 100.0 / 255},
		{PierceBody, 7, 0, 2, false, render.BlendSourceOver, 1, 1},
		{PierceBody, 141, 0, 1, false, render.BlendSourceOver, 1, 1},
		{Bash3D3, 20, 0, 2, false, render.BlendSourceOver, 1, 200.0 / 255},
		{Bash3D3, 20, 1, 1, false, render.BlendSourceOver, 1, 200.0 / 255},
		{Bash3D3, 20, 2, 1, false, render.BlendSourceOver, 1, 200.0 / 255},
		{Bash3D4, 20, 0, 2, false, render.BlendSourceOver, 1, 1},
		{Bash3D4, 20, 1, 1, false, render.BlendSourceOver, 1, 1},
		{Bash3D4, 20, 2, 1, false, render.BlendSourceOver, 1, 1},
		{LKConcentration, 0, 0, 1, false, render.BlendSourceOver, 1, 160.0 / 255},
		{LKConcentration, 0, 1, 1, false, render.BlendSourceOver, 1, 160.0 / 255},
		{LKConcentration, 0, 2, 1, false, render.BlendSourceOver, 1, 160.0 / 255},
	} {
		m := &System{}
		m.AddAt(ctx, tc.effect, 300, start)
		actor.Opt3State = 0
		if tc.aura {
			actor.Opt3State = db.Opt3Aurablade
		}
		entry := scene.ActorDrawEntry{Actor: actor, WorldX: 30.5, WorldY: 20.5}
		frame := render.NewFrame(800, 600)
		billboard.LayerGroup = tc.group
		m.DrawActor(ctx, frame, testutil.Projection(800, 600, 30.5, 20.5, 0), entry, billboard, 1, 1, scene.FullLight,
			color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration), 0)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.count {
			t.Fatalf("%+v: got %d draws", tc, draws.Len())
		}
		body := draws.Index(draws.Len() - 1)
		if body.FieldByName("Options").FieldByName("Blend").Int() != int64(tc.blend) ||
			math.Abs(body.FieldByName("ColorG").Float()-tc.green) > .0001 ||
			math.Abs(body.FieldByName("ColorB").Float()-tc.blue) > .0001 {
			t.Fatalf("%+v: incorrect tint or blend", tc)
		}
	}
}
