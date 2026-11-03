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

func TestChemicalProtectionOriginAndFade(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/shockwave_c.bmp|blackkey=true": render.NewImage(8, 8),
	}}
	m.AddAt(ctx, ChemicalProt, 300, start)
	origin := m.Active[0].ActorOrigin
	actor := ctx.World.Actors[300]
	actor.X += 10
	ctx.World.Actors[300] = actor
	for _, tc := range []struct{ tick, alpha int }{{0, 18}, {9, 180}, {19, 180}, {20, 165}, {30, 15}, {31, 0}} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, testutil.Projection(800, 600, 30.5, 20.5, 0), start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tc.alpha == 0 {
			if draws.Len() != 0 {
				t.Fatal("protection rays survived their fade")
			}
			continue
		}
		if draws.Len() != 1 || draws.Index(0).FieldByName("Vertices").Len() != 64 {
			t.Fatal("protection must draw eight rays with a white core and a blue edge")
		}
		vertices := draws.Index(0).FieldByName("Vertices")
		for i := 0; i < vertices.Len(); i += 4 {
			center := vertices.Index(i)
			if math.Abs(center.FieldByName("X").Float()-origin.X) > 1e-5 || math.Abs(center.FieldByName("Y").Float()-origin.Y) > 1e-5 {
				t.Fatal("protection rays must keep their original ground-level center")
			}
		}
		if math.Abs(vertices.Index(4).FieldByName("ColorA").Float()-float64(tc.alpha)/255) > 1e-6 {
			t.Fatalf("tick %d: wrong protection fade", tc.tick)
		}
	}
}

func TestDemonstrationAnimationStartsOnSecondMotion(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	spec, _ := Lookup(Demonstration)
	view := testutil.EffectRenderTestSprite()
	view.Act.Actions[0].Animations = make([]res.ACTAnimation, 4)
	for motion := 0; motion < 4; motion++ {
		view.Billboards[scene.SpriteBillboardKey{Motion: motion}] = &scene.Billboard{Image: render.NewImage(8+motion, 8)}
	}
	m := &System{EffectViews: map[string]*scene.SpriteView{spec.Components[0].SpriteFile: view}}
	m.AddCellLifetime(ctx, Demonstration, 500, 30, 20, start, 0, true)
	for _, tc := range []struct{ tick, motion int }{{0, 1}, {3, 1}, {4, 2}, {8, 3}, {12, 0}, {16, 1}} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, testutil.Projection(800, 600, 30.5, 20.5, 0), start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != 1 {
			t.Fatalf("tick %d: missing Demonstration fire", tc.tick)
		}
		draw := draws.Index(0)
		if draw.FieldByName("Width").Float() != float64(8+tc.motion) || math.Abs(draw.FieldByName("ColorA").Float()-200.0/255) > 1e-6 {
			t.Fatalf("tick %d: width=%f alpha=%f, want motion %d at opacity 200/255", tc.tick, draw.FieldByName("Width").Float(), draw.FieldByName("ColorA").Float(), tc.motion)
		}
		axis := draw.FieldByName("RightAxis")
		scale := math.Sqrt(math.Pow(axis.Index(0).Float(), 2) + math.Pow(axis.Index(1).Float(), 2) + math.Pow(axis.Index(2).Float(), 2))
		if math.Abs(scale-1.2*scene.PixelRatio) > 1e-6 {
			t.Fatal("Demonstration lost its native sprite scale")
		}
	}
}

func TestAlchemistAndCreatorRealAssets(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, id := range []int{ThrowItem, ThrowItem2, ThrowItem4, ThrowItem5, ChemicalProt, Demonstration, PharmacyOK, PharmacyFail, CoupleCasting, Twilight1, Twilight2, Twilight3, Slim, Slim2, Slim3, AcidDemon, ItemFast3, BeginSpell6} {
		m := &System{}
		m.AddBetween(ctx, id, 300, 2000000, start)
		m.Active[0].ItemLevel = 4
		spec, _ := Lookup(id)
		for _, sound := range spec.Sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d sound: %v", id, err)
			}
		}
		drew := false
		for _, tick := range []int{0, 5, 10, 20, 30, 60, 100} {
			frame := render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			v := reflect.ValueOf(frame).Elem()
			drew = drew || v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() > 0
		}
		if !drew || len(m.textureMiss)+len(m.EffectViewMiss) != 0 {
			t.Fatalf("effect %d: drew=%v, missing textures=%v sprites=%v", id, drew, m.textureMiss, m.EffectViewMiss)
		}
	}
}
