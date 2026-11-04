package effects

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestWinkAndPangVoiceSpriteTiming(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = &res.Manager{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, id := range []int{Wink, FVoice} {
		spec, _ := Lookup(id)
		view := testutil.EffectRenderTestSprite()
		view.Act.Actions[0].Animations = make([]res.ACTAnimation, 4)
		for motion := 0; motion < 4; motion++ {
			view.Billboards[scene.SpriteBillboardKey{Motion: motion}] = &scene.Billboard{Image: render.NewImage(8+motion, 8)}
		}
		m := &System{EffectViews: map[string]*scene.SpriteView{spec.Components[0].SpriteFile: view}}
		m.AddBetween(ctx, id, 300, 2000000, start)
		for _, tc := range []struct{ tick, motion int }{{0, 1}, {5, 1}, {6, 2}, {11, 2}, {12, 3}, {17, 3}, {18, -1}} {
			frame := render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if tc.motion < 0 {
				if draws.Len() != 0 {
					t.Fatalf("effect %d survived the end of its ACT", id)
				}
				continue
			}
			if draws.Len() != 1 || draws.Index(0).FieldByName("Width").Float() != float64(8+tc.motion) {
				t.Fatalf("effect %d tick %d: expected motion %d", id, tc.tick, tc.motion)
			}
		}
	}
}
