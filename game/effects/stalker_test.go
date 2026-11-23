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

func TestStalkerBodyFlashes(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	img := render.NewImage(12, 24)
	img.Fill(color.White)
	for _, tc := range []struct {
		id, tick, shade int
		double, light   bool
	}{{392, 0, 100, true, false}, {392, 3, 250, true, false}, {392, 30, 255, false, false}, {496, 9, 255, false, false}, {496, 10, 250, false, true}, {496, 11, 255, false, false}, {496, 30, 250, false, true}, {496, 31, 255, false, false}} {
		for layer := 0; layer < 3; layer++ {
			m := &System{}
			m.AddAt(ctx, tc.id, 300, start)
			frame := render.NewFrame(800, 600)
			entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
			billboard := &scene.Billboard{Image: img, AnchorX: 6, AnchorY: 24, LayerGroup: layer}
			m.DrawActor(ctx, frame, testutil.Projection(800, 600, 20.5, 20.5, 0), entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), 0)
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			count := 1
			if tc.double && layer == 0 {
				count = 2
			}
			if draws.Len() != count {
				t.Fatalf("effect %d tick %d layer %d: body copies %d", tc.id, tc.tick, layer, draws.Len())
			}
			draw := draws.Index(count - 1)
			blend := render.BlendSourceOver
			if tc.light && layer == 0 {
				blend = render.BlendLighter
			}
			if draw.FieldByName("Options").FieldByName("Blend").Int() != int64(blend) || math.Abs(draw.FieldByName("ColorR").Float()-float64(tc.shade)/255) > .00001 {
				t.Fatalf("effect %d tick %d layer %d: wrong tint/blend", tc.id, tc.tick, layer)
			}
		}
	}
}
