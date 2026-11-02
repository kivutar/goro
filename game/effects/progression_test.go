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
	"github.com/kivutar/goro/res"
)

func TestSuperAngelLayers(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = &res.Manager{}
	ctx.Session.NoEffects = true // The original keeps the level-up angel visible.
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	m := &System{EffectViews: map[string]*scene.SpriteView{}}
	spec, _ := Lookup(Angel2)
	for _, component := range spec.Components[:4] {
		view := testutil.EffectRenderTestSprite()
		view.Act.Actions[0].Animations = make([]res.ACTAnimation, 30)
		for motion := 0; motion < 30; motion++ {
			view.Billboards[scene.SpriteBillboardKey{Motion: motion}] = &scene.Billboard{Image: render.NewImage(8+motion, 8)}
		}
		m.EffectViews[component.SpriteFile] = view
	}
	m.AddAt(ctx, Angel2, ctx.World.Player.ID, start)
	for _, tc := range []struct{ tick, layers, bodyAlpha, motion int }{
		{0, 4, 5, 0}, {3, 5, 20, 0}, {18, 10, 95, 0},
		{29, 10, 150, 0}, {32, 10, 165, 1}, {160, 1, 1, 29}, {161, 0, 0, 29},
	} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.layers {
			t.Fatalf("tick %d: got %d angel layers, want %d", tc.tick, draws.Len(), tc.layers)
		}
		if tc.layers == 0 {
			continue
		}
		body := draws.Index(0)
		if math.Abs(body.FieldByName("ColorA").Float()-float64(tc.bodyAlpha)/255) > 1e-6 || body.FieldByName("Width").Float() != float64(8+tc.motion) {
			t.Fatalf("tick %d: wrong angel fade or motion (must hold first and final frames)", tc.tick)
		}
		for i := 0; i < draws.Len(); i++ {
			if render.Blend(draws.Index(i).FieldByName("Options").FieldByName("Blend").Int()) != render.BlendLighter {
				t.Fatal("angel, wings and feathers must all blend additively")
			}
		}
	}
}

func TestLevelUpBurstTiming(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = &res.Manager{}
	start := time.Unix(100, 0)
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/ring_blue.tga|blackkey=true":   render.NewImage(32, 32),
		"__effectfile_effect/magic_green.tga|blackkey=true": render.NewImage(32, 32),
	}}
	for _, tc := range []struct{ id, tick, groups int }{
		{Angel2, 64, 0}, {Angel2, 65, 1}, {Angel2, 66, 2}, {Angel2, 93, 0},
		{JobLvUp50, 4, 0}, {JobLvUp50, 5, 2}, {JobLvUp50, 32, 0},
	} {
		spec, _ := Lookup(tc.id)
		component := spec.Components[len(spec.Components)-1]
		frame := render.NewFrame(800, 600)
		m.drawLevelUpBurst(frame, ctx, component, Instance{EffectID: tc.id, Starts: start}, 0, 0, 0, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if draws.Len() != tc.groups {
			t.Fatalf("effect %d tick %d: got %d bursts, want %d", tc.id, tc.tick, draws.Len(), tc.groups)
		}
		for i := 0; i < draws.Len(); i++ {
			if draws.Index(i).FieldByName("Vertices").Len() != 48 {
				t.Fatal("each burst must have four blades with three slices")
			}
		}
	}
}

func TestSuperNoviceRealAssets(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	for _, id := range []int{Angel2, JobLvUp50, FirstAid} {
		m := &System{}
		spec, _ := Lookup(id)
		for _, component := range spec.Components {
			if component.SpriteFile != "" && m.EffectSpriteView(ctx.Resources, component.SpriteFile) == nil {
				t.Fatalf("effect %d missing sprite %s", id, component.SpriteFile)
			}
		}
		for _, sound := range spec.Sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
		m.AddAt(ctx, id, ctx.World.Player.ID, start)
		for _, tick := range []int{0, 19, 32, 66, 100} {
			frame := render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			value := reflect.ValueOf(frame).Elem()
			if tick == 19 && value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() == 0 {
				t.Fatalf("effect %d drew nothing", id)
			}
		}
		if len(m.textureMiss)+len(m.EffectViewMiss) != 0 {
			t.Fatalf("effect %d has missing assets: %v %v", id, m.textureMiss, m.EffectViewMiss)
		}
	}
}
