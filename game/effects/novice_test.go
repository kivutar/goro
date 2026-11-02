package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestFirstAidGlowTimingAndPosition(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = &res.Manager{}
	start := time.Unix(100, 0)
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/pikapika2.bmp|blackkey=true": render.NewImage(32, 32),
	}}
	m.AddAt(ctx, FirstAid, ctx.World.Player.ID, start)
	origin := m.Active[0].ActorOrigin
	ctx.World.Player.X += 10
	for _, tc := range []struct{ frame, alpha int }{
		{-1, 0}, {0, 4}, {19, 80}, {99, 80}, {100, 79}, {178, 1}, {179, 0},
	} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, testutil.Projection(800, 600, origin.X, origin.Z, 0), start.Add(time.Duration(tc.frame)*db.EffectFrameDuration), false)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if tc.alpha == 0 {
			if draws.Len() != 0 {
				t.Fatalf("tick %d: glow outside its visible lifetime", tc.frame)
			}
			continue
		}
		if draws.Len() != 1 {
			t.Fatalf("tick %d: got %d glows", tc.frame, draws.Len())
		}
		draw := draws.Index(0)
		center := draw.FieldByName("Center")
		if math.Abs(center.Index(0).Float()-origin.X) > 1e-5 || math.Abs(center.Index(1).Float()-origin.Y-2.4) > 1e-5 {
			t.Fatal("First Aid must stay at its cast position, 12 native units above the ground")
		}
		for field, value := range map[string]int{"ColorR": 150, "ColorG": 150, "ColorB": 250, "ColorA": tc.alpha} {
			if math.Abs(draw.FieldByName(field).Float()-float64(value)/255) > 1e-6 {
				t.Fatalf("tick %d: incorrect %s", tc.frame, field)
			}
		}
		if render.Blend(draw.FieldByName("Options").FieldByName("Blend").Int()) != render.BlendLighter {
			t.Fatal("First Aid must use additive blending")
		}
	}
}
