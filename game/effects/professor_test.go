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

func TestProfessorBodyLayers(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	image := render.NewImage(12, 24)
	image.Fill(color.White)
	for _, tc := range []struct {
		effect, tick, draws int
		tint                color.RGBA
		blend               render.Blend
	}{
		{383, 49, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{383, 50, 2, color.RGBA{150, 150, 250, 255}, render.BlendSourceOver},
		{383, 80, 2, color.RGBA{90, 90, 250, 255}, render.BlendSourceOver},
		{385, 40, 1, color.RGBA{200, 150, 50, 255}, render.BlendLighter},
		{385, 71, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{505, 10, 1, color.RGBA{250, 250, 100, 255}, render.BlendLighter},
		{505, 11, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{505, 36, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{521, 10, 2, color.RGBA{255, 0, 0, 255}, render.BlendLighter},
		{521, 11, 2, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{521, 51, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
	} {
		for layer := 0; layer < 3; layer++ {
			m := &System{}
			m.AddAt(ctx, tc.effect, 300, start)
			frame := render.NewFrame(800, 600)
			entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
			billboard := &scene.Billboard{Image: image, AnchorX: 6, AnchorY: 24, LayerGroup: layer}
			m.DrawActor(ctx, frame, testutil.Projection(800, 600, 20.5, 20.5, 0), entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), 0)
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			want, blend := tc.draws, tc.blend
			if layer != 0 {
				want, blend = 1, render.BlendSourceOver
			}
			if draws.Len() != want || draws.Index(draws.Len()-1).FieldByName("Options").FieldByName("Blend").Int() != int64(blend) {
				t.Fatalf("wrong body/weapon rendering for %+v layer %d", tc, layer)
			}
			for i, channel := range []string{"ColorR", "ColorG", "ColorB"} {
				want := []uint8{tc.tint.R, tc.tint.G, tc.tint.B}[i]
				if math.Abs(draws.Index(draws.Len()-1).FieldByName(channel).Float()-float64(want)/255) > .00001 {
					t.Fatalf("wrong %s tint for %+v layer %d", channel, tc, layer)
				}
			}
		}
	}
}
