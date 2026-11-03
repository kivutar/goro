package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func TestAdrenalineParticlesStayAtOriginWhileRaysFollowCaster(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	start := time.Unix(100, 0)
	m := &System{
		EffectViews: map[string]*scene.SpriteView{"particle1": testutil.EffectRenderTestSprite()},
		textures: map[string]*render.Image{
			"__effectfile_effect/alpha_center.tga|blackkey=true": render.NewImage(32, 32),
		},
	}
	m.AddAt(ctx, HasteUp, 2000000, start)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, x := range []int{12, 14} {
		ctx.World.Player.X = x // Move before the first draw, too.
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		particles, rays := value.FieldByName("worldBillboards"), value.FieldByName("commands")
		if particles.Len() != 4 || rays.Len() != 1 {
			t.Fatal("Adrenaline Rush needs four orbiting particles and one batch of rays")
		}
		var center scene.Point3
		for i := 0; i < particles.Len(); i++ {
			p := particles.Index(i).FieldByName("Center")
			center.X += p.Index(0).Float() / 4
			center.Y += p.Index(1).Float() / 4
			center.Z += p.Index(2).Float() / 4
		}
		if math.Abs(center.X-10.5) > 1e-5 || math.Abs(center.Z-20.5) > 1e-5 || math.Abs(center.Y-(.07+2.562/5)) > 1e-5 {
			t.Fatalf("orbit moved away from its casting position: %+v", center)
		}
		anchor, _ := projection.ProjectPoint(float64(x)+.5, 20.5, .07)
		ray := rays.Index(0).FieldByName("Vertices").Index(0)
		if math.Abs(ray.FieldByName("DstX").Float()-float64(anchor.X)) > 1e-4 || math.Abs(ray.FieldByName("DstY").Float()-float64(anchor.Y-40)) > 1e-4 {
			t.Fatal("screen-space rays stopped following the caster")
		}
	}
}
