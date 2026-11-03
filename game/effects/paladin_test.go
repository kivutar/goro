package effects

import (
	"image/color"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestPressureTracksOnlyDuringItsFall(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = nil
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 30, 20, 50, 20
	ctx.World.Actors[300] = actor
	projection := testutil.Projection(800, 600, 20, 20, 0)
	wantX := scene.CellCenter(30 + 10*(31*db.EffectFrameDuration).Seconds())
	for _, ticks := range [][]int{{0, 30, 31, 33}, {0, 30, 33}, {33}} {
		m := &System{}
		m.AddBetween(ctx, Pressure, 300, 2000000, start)
		for _, tick := range ticks {
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.Draw(render.NewFrame(800, 600), ctx, projection, now, false)
		}
		flight := m.Active[0].pressure
		if flight.tick < 32 || math.Abs(flight.origin.X-wantX) > 1e-6 {
			t.Fatalf("frames %v: landing X=%f, want %f at the final falling tick", ticks, flight.origin.X, wantX)
		}
		for i, particle := range flight.images {
			at := time.Duration(min(33-i, 31)) * db.EffectFrameDuration
			if want := scene.CellCenter(30 + 10*at.Seconds()); math.Abs(particle.Position.X-want) > 1e-6 {
				t.Fatalf("frames %v: afterimage %d at X=%f, want %f", ticks, i, particle.Position.X, want)
			}
		}
		// The victim keeps moving, but the landed cross must stay put.
		origin := flight.origin
		m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(time.Second), false)
		if flight.origin != origin {
			t.Fatal("landed cross followed the victim")
		}
	}
}

func TestPressureRetainsTrailWhenVictimStops(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 30, 20, 50, 20
	ctx.World.Actors[300] = actor
	m := &System{}
	m.AddBetween(ctx, Pressure, 300, 2000000, start)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(20*db.EffectFrameDuration), false)
	previous := m.Active[0].pressure.images
	actor.Moving, actor.X, actor.Y = false, 50, 20
	ctx.World.Actors[300] = actor
	now := start.Add(21 * db.EffectFrameDuration)
	ctx.Session.NoEffects = true
	m.Draw(render.NewFrame(800, 600), ctx, projection, now, true)
	flight := m.Active[0].pressure
	if flight.images[0].Position.X != scene.CellCenter(50) {
		t.Fatal("falling cross did not follow the victim")
	}
	for i := 1; i < len(flight.images); i++ {
		if flight.images[i] != previous[i-1] {
			t.Fatalf("afterimage %d lost its previous position, rotation or alpha", i)
		}
	}
	before := *flight
	ctx.Session.NoEffects = false
	m.Draw(render.NewFrame(800, 600), ctx, projection, now, false)
	if *flight != before {
		t.Fatal("drawing the same tick twice advanced Pressure")
	}
}

func TestPaladinBodyEffects(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	texture := render.NewImage(12, 24)
	texture.Fill(color.White)
	billboard := &scene.Billboard{Image: texture, AnchorX: 6, AnchorY: 24}
	entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 30.5, WorldY: 20.5}
	for _, tc := range []struct {
		effect, tick, group, count int
		disabled                   bool
		blend                      render.Blend
		tint                       color.RGBA
	}{
		{Bash3D, 19, 0, 1, false, render.BlendSourceOver, color.RGBA{255, 255, 255, 255}},
		{Bash3D, 20, 0, 2, false, render.BlendSourceOver, color.RGBA{255, 200, 200, 255}},
		{Bash3D, 40, 0, 2, false, render.BlendSourceOver, color.RGBA{255, 200, 200, 255}},
		{Bash3D, 41, 0, 1, false, render.BlendSourceOver, color.RGBA{255, 255, 255, 255}},
		{Bash3D, 20, 1, 1, false, render.BlendSourceOver, color.RGBA{255, 200, 200, 255}},
		{Bash3D, 20, 2, 1, false, render.BlendSourceOver, color.RGBA{255, 200, 200, 255}},
		{Bash3D, 20, 0, 1, true, render.BlendSourceOver, color.RGBA{255, 255, 255, 255}},
		{Shrink, 4, 0, 1, false, render.BlendSourceOver, color.RGBA{255, 255, 255, 255}},
		{Shrink, 6, 0, 1, false, render.BlendLighter, color.RGBA{250, 250, 100, 255}},
		{Shrink, 7, 0, 1, false, render.BlendSourceOver, color.RGBA{255, 255, 255, 255}},
		{Shrink, 24, 0, 1, false, render.BlendLighter, color.RGBA{250, 250, 100, 255}},
		{Shrink, 25, 0, 1, false, render.BlendSourceOver, color.RGBA{255, 255, 255, 255}},
		{Shrink, 26, 0, 1, false, render.BlendSourceOver, color.RGBA{255, 255, 255, 255}},
		{Shrink, 6, 1, 1, false, render.BlendSourceOver, color.RGBA{250, 250, 100, 255}},
		{Shrink, 6, 2, 1, false, render.BlendSourceOver, color.RGBA{250, 250, 100, 255}},
		{Shrink, 6, 0, 1, true, render.BlendSourceOver, color.RGBA{255, 255, 255, 255}},
	} {
		m := &System{}
		ctx.Session.NoEffects = tc.disabled
		m.AddAt(ctx, tc.effect, 300, start)
		frame := render.NewFrame(800, 600)
		billboard.LayerGroup = tc.group
		m.DrawActor(ctx, frame, testutil.Projection(800, 600, 30.5, 20.5, 0), entry, billboard, 1, 1, 1,
			color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), 0)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.count {
			t.Fatalf("%+v: got %d draws", tc, draws.Len())
		}
		body := draws.Index(draws.Len() - 1)
		if body.FieldByName("Options").FieldByName("Blend").Int() != int64(tc.blend) ||
			math.Abs(body.FieldByName("ColorR").Float()-float64(tc.tint.R)/255) > .0001 ||
			math.Abs(body.FieldByName("ColorG").Float()-float64(tc.tint.G)/255) > .0001 ||
			math.Abs(body.FieldByName("ColorB").Float()-float64(tc.tint.B)/255) > .0001 {
			t.Fatalf("%+v: incorrect tint or blend", tc)
		}
		if tc.count == 2 && draws.Index(0).FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendLighter) {
			t.Fatal("Sacrifice's body glow must be additive")
		}
	}
}

func TestShieldChainCapturesEachSpawnPosition(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 30, 20, 50, 20
	ctx.World.Actors[300] = actor
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, ticks := range [][]int{{0, 3, 6, 9, 12}, {12}} {
		m := &System{textures: map[string]*render.Image{
			"__effectfile_effect/shield_boomerang.bmp|blackkey=true": render.NewImage(1, 1),
		}}
		m.AddBetween(ctx, ShieldBoomerang3, 300, 2000000, start)
		ctx.Session.NoEffects = true
		for _, tick := range ticks {
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.Draw(render.NewFrame(800, 600), ctx, projection, now, false)
		}
		effect := m.Active[0]
		if effect.ShieldChain.Count != 5 {
			t.Fatalf("frames %v: spawned %d shields, want five", ticks, effect.ShieldChain.Count)
		}
		for i, origin := range effect.ShieldChain.Origins {
			wantX := scene.CellCenter(30 + 10*(time.Duration(3*i)*db.EffectFrameDuration).Seconds())
			particle := shieldChainImage(effect, origin, i, 15)
			if math.Abs(particle.Position.X-wantX) > 1e-6 {
				t.Fatalf("frames %v: shield %d crosses X=%f, want spawn position %f", ticks, i, particle.Position.X, wantX)
			}
		}
		origins := effect.ShieldChain.Origins
		stopped := actor
		stopped.Moving, stopped.X, stopped.Y = false, 60, 20
		ctx.World.Actors[300] = stopped
		ctx.Session.NoEffects = false
		m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(30*db.EffectFrameDuration), false)
		if effect.ShieldChain.Origins != origins {
			t.Fatal("existing shield trajectories moved with the victim")
		}
		ctx.World.Actors[300] = actor
	}
}

func TestPaladinRealAssetsAndRendering(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to validate original Paladin assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	ctx := testutil.KnightEffectContext()
	ctx.Resources = manager
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, id := range []int{Guard, ShieldCharge, ShieldBoomer, ReflectShield, HolyCross, GrandCross, Devotion, Providence, CrusaderDef, SpearQuicken, Pressure, Bash3D, BottomGospel, GospelGround, ShieldBoomerang3, Shrink, LineLink} {
		m := &System{}
		spec, ok := Lookup(id)
		if !ok {
			t.Fatalf("missing effect %d", id)
		}
		for _, component := range spec.Components {
			for _, path := range append([]string{component.TextureFile}, component.TextureFiles...) {
				if path != "" && m.EffectFileTexture(manager, path) == nil {
					t.Fatalf("effect %d missing %s", id, path)
				}
			}
			if component.Kind == ComponentFUNC && component.FuncAdapter == EffectFuncUnknown {
				t.Fatalf("effect %d has no renderer for %s", id, component.FuncName)
			}
			if component.StrFile != "" {
				str := m.LoadWorldEffectSTR(manager, component.StrFile, component.TexturePath)
				if str == nil {
					t.Fatalf("missing STR %s", component.StrFile)
				}
				for _, layer := range str.Layers {
					for _, path := range layer.Textures {
						if m.strEffectTexture(manager, path) == nil {
							t.Fatalf("missing STR texture %s", path)
						}
					}
				}
			}
		}
		for _, sound := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d sound: %v", id, err)
			}
		}
		m.AddBetween(ctx, id, 300, 2000000, start)
		drew := false
		for _, tick := range []int{0, 6, 15, 30, 40, 60, 100, 190} {
			frame := render.NewFrame(800, 600)
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.Draw(frame, ctx, projection, now, true)
			m.Draw(frame, ctx, projection, now, false)
			v := reflect.ValueOf(frame).Elem()
			drew = drew || v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() > 0
		}
		if len(spec.Components) > 0 && !drew {
			t.Fatalf("effect %d submitted no geometry", id)
		}
	}
}
