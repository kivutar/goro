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

func TestAcidDemonstrationCapturesEachBottleLaunch(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 30, 20, 50, 20
	projection := testutil.Projection(800, 600, 20, 20, 0)
	spec, _ := Lookup(ThrowItem4)
	for _, ticks := range [][]int{{0, 4, 5, 9, 10}, {10}} {
		ctx.World.Actors[300] = actor
		ctx.World.Player.X = 10
		ctx.Session.NoEffects = false
		m := &System{textures: map[string]*render.Image{}}
		for _, component := range spec.Components {
			m.textures["__effectfile_"+component.TextureFile+"|blackkey=false"] = render.NewImage(8, 8)
		}
		m.AddBetween(ctx, ThrowItem4, 2000000, 300, start)
		// A moving target must not steer bottles already aimed at the cast position.
		ctx.World.Player.X = 70
		ctx.Session.NoEffects = true
		for _, tick := range ticks {
			m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			want := 0
			if tick >= 5 {
				want++
			}
			if tick >= 10 {
				want++
			}
			if m.Active[0].acidBottles.count != want {
				t.Fatalf("tick %d: bottles must launch on ticks five and ten, even while hidden", tick)
			}
		}
		flight := *m.Active[0].acidBottles
		for i, origin := range flight.origins {
			want := scene.CellCenter(30 + 10*(time.Duration(5+5*i)*db.EffectFrameDuration).Seconds())
			if math.Abs(origin.X-want) > 1e-6 {
				t.Fatalf("frames %v: bottle %d launched at X=%f, want %f", ticks, i, origin.X, want)
			}
		}
		stopped := actor
		stopped.Moving, stopped.X = false, 60
		ctx.World.Actors[300] = stopped
		ctx.Session.NoEffects = false
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		if *m.Active[0].acidBottles != flight {
			t.Fatal("launched bottles moved with the caster")
		}
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != 8 {
			t.Fatal("both bottles must retain their three afterimages")
		}
		for i, origin := range flight.origins {
			progress := float64(16-5*i) / 25
			wantX := origin.X + (10.5-origin.X)*progress
			if got := draws.Index(4 * i).FieldByName("Center").Index(0).Float(); math.Abs(got-wantX) > 1e-5 {
				t.Fatalf("bottle %d at X=%f, want %f towards the original target", i, got, wantX)
			}
		}
	}
}
