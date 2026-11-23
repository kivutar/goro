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

func TestChampionBodyFlashWindows(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	start := time.Unix(100, 0)
	img := render.NewImage(12, 24)
	img.Fill(color.White)
	for _, tc := range []struct {
		id, first, last int
		tint            color.RGBA
		double          bool
	}{
		{376, 10, 20, color.RGBA{255, 120, 50, 255}, true},
		{512, 20, 100, color.RGBA{255, 255, 150, 255}, true},
		{377, 5, 35, color.RGBA{255, 255, 255, 255}, false},
		{402, 5, 65, color.RGBA{250, 250, 200, 255}, false},
	} {
		for _, tick := range []int{tc.first - 1, tc.first, tc.last, tc.last + 1} {
			for layer := 0; layer < 3; layer++ {
				m := &System{}
				m.AddAt(ctx, tc.id, 300, start)
				frame := render.NewFrame(800, 600)
				active := tick >= tc.first && tick <= tc.last
				tint, blend, count := (color.RGBA{255, 255, 255, 255}), render.BlendSourceOver, 1
				if active {
					tint = tc.tint
					if layer == 0 {
						if tc.double {
							count = 2
						} else {
							blend = render.BlendLighter
						}
					}
				}
				m.DrawActor(ctx, frame, testutil.Projection(800, 600, 12, 20, 0), scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 12.5, WorldY: 20.5}, &scene.Billboard{Image: img, AnchorX: 6, AnchorY: 24, LayerGroup: layer}, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tick)*db.EffectFrameDuration), 0)
				draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
				if draws.Len() != count {
					t.Fatalf("effect %d tick %d layer %d: wrong body copies", tc.id, tick, layer)
				}
				draw := draws.Index(count - 1)
				for name, want := range map[string]uint8{"ColorR": tint.R, "ColorG": tint.G, "ColorB": tint.B, "ColorA": 255} {
					if math.Abs(draw.FieldByName(name).Float()-float64(want)/255) > .00001 {
						t.Fatalf("effect %d tick %d: wrong %s", tc.id, tick, name)
					}
				}
				if draw.FieldByName("Options").FieldByName("Blend").Int() != int64(blend) {
					t.Fatal("wrong body flash blend")
				}
			}
		}
	}
}
