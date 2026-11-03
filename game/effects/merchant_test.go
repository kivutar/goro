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

func TestMammoniteCoinInitialTrail(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	start := time.Unix(100, 0)
	m := &System{EffectViews: map[string]*scene.SpriteView{"동전": testutil.EffectRenderTestSprite()}}
	m.AddAt(ctx, Mammonite, 300, start)
	effect := m.Active[0]
	actor := ctx.World.Actors[300]
	actor.X += 10
	ctx.World.Actors[300] = actor
	spec, _ := Lookup(Mammonite)
	for _, tick := range []int{19, 20, 21} {
		frame := render.NewFrame(800, 600)
		m.DrawMammoniteCoins(frame, ctx, testutil.Projection(800, 600, 11, 20, 0), spec.Components[1], effect, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if tick == 19 {
			if draws.Len() != 0 {
				t.Fatal("coins appeared before tick 20")
			}
			continue
		}
		if draws.Len() != 10 {
			t.Fatalf("tick %d: expected five coins with afterimages, got %d draws", tick, draws.Len())
		}
		for i := 0; i < 5; i++ {
			want := effect.ActorOrigin
			if tick == 21 {
				want = effect.CoinTrails[i][0]
			}
			draw := draws.Index(i * 2)
			center := draw.FieldByName("Center")
			if math.Abs(center.Index(0).Float()-want.X) > 1e-5 || math.Abs(center.Index(1).Float()-want.Y) > 1e-5 || math.Abs(center.Index(2).Float()-want.Z) > 1e-5 {
				t.Fatal("coin afterimage must start at the effect origin, then follow the previous position")
			}
			if math.Abs(draw.FieldByName("ColorA").Float()-127.0/255) > 1e-6 {
				t.Fatal("coin afterimage must retain half opacity")
			}
		}
	}
}
