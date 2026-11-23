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

func TestHighWizardBodyLayers(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	image := render.NewImage(12, 24)
	image.Fill(color.White)
	for _, tc := range []struct {
		effect, tick, layer, draws int
		blend                      render.Blend
	}{
		{382, 0, 0, 1, render.BlendSourceOver}, {382, 0, 1, 2, render.BlendSourceOver},
		{382, 1, 1, 1, render.BlendSourceOver}, {382, 0, 2, 1, render.BlendSourceOver},
		{380, 30, 0, 1, render.BlendLighter}, {380, 30, 1, 1, render.BlendSourceOver}, {380, 61, 0, 1, render.BlendSourceOver},
		{378, 54, 0, 1, render.BlendSourceOver}, {378, 55, 0, 1, render.BlendLighter},
		{378, 65, 0, 1, render.BlendLighter}, {378, 66, 0, 1, render.BlendSourceOver}, {378, 55, 1, 1, render.BlendSourceOver},
	} {
		m := &System{}
		m.AddAt(ctx, tc.effect, 300, start)
		frame := render.NewFrame(800, 600)
		entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
		billboard := &scene.Billboard{Image: image, AnchorX: 6, AnchorY: 24, LayerGroup: tc.layer}
		m.DrawActor(ctx, frame, testutil.Projection(800, 600, 20.5, 20.5, 0), entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), 0)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.draws || draws.Index(draws.Len()-1).FieldByName("Options").FieldByName("Blend").Int() != int64(tc.blend) {
			t.Fatalf("wrong body/weapon rendering: %+v", tc)
		}
		if tc.draws == 2 && draws.Index(0).FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendLighter) {
			t.Fatal("weapon sparkle must add a separate additive draw")
		}
		if tc.draws == 2 && draws.Index(0).FieldByName("Options").FieldByName("DepthWrite").Bool() {
			t.Fatal("the additional weapon sparkle must not write depth")
		}
	}
}

func TestHighWizardGravitationNativeMotion(t *testing.T) {
	position, speed, accel := 0.0, 1.0, .01
	for frame := 0; frame < 2000; frame++ {
		if frame == 550 || frame < 550 && frame%6 == 2 {
			speed, accel = -1.2, 0
		} else if frame < 550 && frame%6 == 5 {
			speed, accel = 1.18, .01
		}
		speed += accel
		position += speed
		if math.Abs(gravitationTravel(frame+1)-position) > .000001 {
			t.Fatalf("spike diverged from Prim3DQuadHorn2 at frame %d", frame)
		}
	}
}
