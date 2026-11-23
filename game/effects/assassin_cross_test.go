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

func TestAssassinCrossBodyLayers(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	image := render.NewImage(12, 24)
	image.Fill(color.White)
	for _, tc := range []struct {
		effect, tick int
		purple       bool
	}{
		{361, 0, true}, {361, 10, true}, {361, 11, false},
		{409, 0, true}, {409, 11, false},
		{493, 0, true}, {493, 1, false}, {493, 80, true}, {493, 82, false},
	} {
		for layer := 0; layer < 3; layer++ {
			m := &System{}
			m.AddAt(ctx, tc.effect, 300, start)
			frame := render.NewFrame(800, 600)
			entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
			billboard := &scene.Billboard{Image: image, AnchorX: 6, AnchorY: 24, LayerGroup: layer}
			m.DrawActor(ctx, frame, testutil.Projection(800, 600, 20.5, 20.5, 0), entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), 0)
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			blend, green := render.BlendSourceOver, 1.0
			if tc.purple {
				green = 0
				if layer == 0 {
					blend = render.BlendLighter
				}
			}
			if draws.Len() != 1 || draws.Index(0).FieldByName("Options").FieldByName("Blend").Int() != int64(blend) || math.Abs(draws.Index(0).FieldByName("ColorG").Float()-green) > .00001 {
				t.Fatalf("effect %d tick %d layer %d: wrong tint or blend", tc.effect, tc.tick, layer)
			}
		}
	}
}
