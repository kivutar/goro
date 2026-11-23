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

func TestAssumptioOnlyGlowsBodyLayer(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &System{}
	start := time.Now()
	m.AddAt(ctx, Assumptio, 300, start)
	for _, group := range []int{0, 1, 2} {
		frame := render.NewFrame(800, 600)
		billboard := &scene.Billboard{Image: render.NewImage(12, 24), AnchorX: 6, AnchorY: 24, LayerGroup: group}
		entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 12, WorldY: 20}
		m.DrawActor(ctx, frame, testutil.Projection(800, 600, 12, 20, 0), entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start, 0)
		count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
		want := 1
		if group == 0 {
			want = 2
		}
		if count != want {
			t.Fatalf("layer %d: got %d draws, want %d", group, count, want)
		}
	}
}

func TestPriestRealAssets(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 30.5, 20.5, 0)
	for _, id := range []int{
		TurnUndead, Revive, Resurrection, Assumptio2,
		Impositio, Suffragium, Aspersio, Benedictio,
		Kyrie, Magnificat, Gloria, LexDivina,
		Recovery, SlowPoison, LexAeterna, Magnus,
		SafetyWall, BottomSanc, BottomMagnus, BottomBasilica,
	} {
		m := &System{}
		spec, _ := Lookup(id)
		for _, sound := range spec.Sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d: %v", id, err)
			}
		}
		m.AddAt(ctx, id, 300, start)
		drew := false
		for _, tick := range []int{0, 15, 50, 60, 90} {
			frame := render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			value := reflect.ValueOf(frame).Elem()
			drew = drew || value.FieldByName("commands").Len()+value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() > 0
		}
		if !drew || len(m.textureMiss) != 0 {
			t.Fatalf("effect %d drew nothing or has missing textures: %v", id, m.textureMiss)
		}
	}
	if _, err := ctx.Resources.ReadFile("data/wav/effect/바실리카.wav"); err != nil {
		t.Fatal(err)
	}
}

func TestTurnUndeadPhases(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/alpha_down.tga|blackkey=true":   render.NewImage(16, 16),
		"__effectfile_effect/alpha_center.tga|blackkey=true": render.NewImage(16, 16),
		"__effectfile_effect/ring_liner.tga|blackkey=true":   render.NewImage(16, 16),
	}}
	spec, _ := Lookup(TurnUndead)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, tc := range []struct{ tick, screenDraws, worldDraws int }{
		{-1, 0, 0}, {0, 2, 0}, {19, 2, 0}, {20, 0, 0}, {49, 0, 0}, {50, 0, 1}, {65, 0, 1}, {80, 0, 0},
	} {
		frame := render.NewFrame(800, 600)
		m.drawTurnUndeadEffect(frame, ctx, projection, spec.Components[0], Instance{Starts: start}, 10, 20, 3, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		value := reflect.ValueOf(frame).Elem()
		world := value.FieldByName("worldCommands")
		if value.FieldByName("commands").Len() != tc.screenDraws || world.Len() != tc.worldDraws {
			t.Fatalf("tick %d: wrong Turn Undead phase", tc.tick)
		}
		if world.Len() > 0 {
			vertices := world.Index(0).FieldByName("Vertices")
			for i := 0; i < vertices.Len(); i++ {
				if math.Abs(vertices.Index(i).FieldByName("Y").Float()-4.4) > 1e-5 {
					t.Fatal("Turn Undead ring must lie horizontally above the caster's feet")
				}
			}
		}
	}
}

func TestPriestGroundGeometryAndBlending(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		id      int
		half    float64
		sides   int
		alpha   float64
		blend   render.Blend
		texture string
	}{
		{BottomSanc, .5, 4, 120.0 / 255, render.BlendSourceOver, "effect/alpha_down.tga"},
		{BottomMagnus, 1, 4, 30.0 / 255, render.BlendLighter, "effect/ring_red.tga"},
		{BottomBasilica, 2.96, 24, 65.0 / 255, render.BlendLighter, "effect/alpha_down.tga"},
	} {
		m := &System{textures: map[string]*render.Image{
			"__effectfile_" + tc.texture + "|blackkey=true": render.NewImage(16, 16),
		}}
		spec, _ := Lookup(tc.id)
		frame := render.NewFrame(800, 600)
		m.drawGroundWallsEffect(frame, ctx, spec.Components[0], Instance{EffectID: tc.id, Starts: start},
			10, 20, 3, start.Add(89*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if cmds.Len() != 1 {
			t.Fatalf("effect %d: expected one batch", tc.id)
		}
		cmd := cmds.Index(0)
		vertices := cmd.FieldByName("Vertices")
		if vertices.Len() != tc.sides*4 || render.Blend(cmd.FieldByName("Options").FieldByName("Blend").Int()) != tc.blend {
			t.Fatalf("effect %d: wrong face count (%d) or blend", tc.id, vertices.Len()/4)
		}
		minY := math.Inf(1)
		maxRadius := 0.0
		for i := 0; i < vertices.Len(); i++ {
			v := vertices.Index(i)
			x, y, z := v.FieldByName("X").Float(), v.FieldByName("Y").Float(), v.FieldByName("Z").Float()
			maxRadius = math.Max(maxRadius, math.Max(math.Abs(x-10), math.Abs(z-20)))
			minY = math.Min(minY, y)
		}
		if math.Abs(maxRadius-tc.half) > 1e-5 || math.Abs(minY-3.2) > 1e-5 {
			t.Fatalf("effect %d: wrong footprint or elevation (%g, %g)", tc.id, maxRadius, minY)
		}
		if math.Abs(vertices.Index(0).FieldByName("ColorA").Float()-tc.alpha) > 1e-5 {
			t.Fatalf("effect %d: wrong wall opacity", tc.id)
		}
	}
}
