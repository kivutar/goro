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

func TestSafetyWallRisesAndRipples(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	texture := render.NewImage(32, 32)
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/alpha_down.tga|blackkey=true": texture,
	}}
	spec, _ := Lookup(SafetyWall)
	effect := Instance{EffectID: SafetyWall, Starts: start}
	for _, tick := range []int{-1, 0, 14, 89, 150, 3000, 3117} {
		frame := render.NewFrame(800, 600)
		m.drawSafetyWall(frame, ctx, spec.Components[1], effect, scene.Point3{}, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick < 0 || tick == 3117 {
			if cmds.Len() != 0 {
				t.Fatalf("tick %d: wall visible outside its lifetime", tick)
			}
			continue
		}
		if cmds.Len() != 1 {
			t.Fatalf("tick %d: got %d draw calls, want one band mesh", tick, cmds.Len())
		}
		cmd := cmds.Index(0)
		vertices := cmd.FieldByName("Vertices")
		if vertices.Len() != 3*20*4 || !cmd.FieldByName("Options").FieldByName("DepthTest").Bool() ||
			render.Blend(cmd.FieldByName("Options").FieldByName("Blend").Int()) != render.BlendLighter {
			t.Fatal("Safety Wall must draw three additive, depth-tested bands")
		}
		alpha := 75.0
		if tick == 0 {
			alpha = 5
		}
		for channel, want := range map[string]float64{"ColorR": 1, "ColorG": 89.0 / 255, "ColorB": 182.0 / 255, "ColorA": alpha / 255} {
			if got := vertices.Index(0).FieldByName(channel).Float(); math.Abs(got-want) > 1e-6 {
				t.Fatalf("tick %d: %s=%g, want %g", tick, channel, got, want)
			}
		}
		for band, radius := range []float64{.72, .66, .60} {
			low, high := math.Inf(1), 0.0
			for i := band * 80; i < (band+1)*80; i++ {
				v := vertices.Index(i)
				x, y, z := v.FieldByName("X").Float(), v.FieldByName("Y").Float(), v.FieldByName("Z").Float()
				if math.Abs(math.Hypot(x, z)-radius) > 1e-6 {
					t.Fatalf("band %d: radius=%g, want %g", band, math.Hypot(x, z), radius)
				}
				if i%4 < 2 {
					low, high = min(low, y), max(high, y)
				} else if y != 0 {
					t.Fatal("wall lifted off the ground")
				}
			}
			if tick == 0 && high > .15 || tick >= 89 && high < 6 || low >= high*.75 {
				t.Fatalf("tick %d band %d: wall did not rise/ripple: heights %g..%g", tick, band, low, high)
			}
		}
	}
}

func TestSafetyWallStartupDoesNotRepeat(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &System{}
	str := m.LoadWorldEffectSTR(ctx.Resources, "safetywall", "")
	if str == nil || str.MaxKey <= 0 || str.FPS <= 0 {
		t.Fatal("Safety Wall animation missing")
	}
	start := time.Unix(100, 0)
	m.AddCellLifetime(ctx, SafetyWall, 999, 20, 20, start, time.Minute, true)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for tick := 1; tick < min(30, str.MaxKey); tick++ {
		at := start.Add(time.Duration(tick) * time.Second / time.Duration(str.FPS))
		startup := render.NewFrame(800, 600)
		m.Draw(startup, ctx, projection, at, false)
		if reflect.ValueOf(startup).Elem().FieldByName("worldCommands").Len() <= 1 {
			continue
		}
		later := render.NewFrame(800, 600)
		m.Draw(later, ctx, projection, at.Add(time.Duration(str.MaxKey)*time.Second/time.Duration(str.FPS)), false)
		if got := reflect.ValueOf(later).Elem().FieldByName("worldCommands").Len(); got != 1 {
			t.Fatalf("got %d draw calls after startup; only the wall bands should remain", got)
		}
		m.Remove(SafetyWall, 999)
		if len(m.Active) != 0 {
			t.Fatal("wall survived removal of its skill unit")
		}
		return
	}
	t.Fatal("Safety Wall startup animation produced no geometry")
}

func TestSightFlamesStayAtOrbitHeight(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	view := testutil.EffectRenderTestSprite()
	view.Billboards[scene.SpriteBillboardKey{}].AnchorY = 4
	m := &System{EffectViews: map[string]*scene.SpriteView{"sight": view}}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, id := range []int{Sight, SightBlaster} {
		spec, _ := Lookup(id)
		for _, tick := range []int{0, 12, 60, 600} {
			frame := render.NewFrame(800, 600)
			m.drawSightEffect(frame, ctx, projection, spec.Components[0],
				Instance{EffectID: id, Starts: start}, scene.Point3{Y: 2}, start.Add(time.Duration(tick)*db.EffectFrameDuration))
			cmds := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if cmds.Len() == 0 {
				t.Fatalf("effect %d tick %d: no flames", id, tick)
			}
			for i := 0; i < cmds.Len(); i++ {
				y := cmds.Index(i).FieldByName("Center").Index(1).Float()
				if math.Abs(y-6) > 1e-5 {
					t.Fatalf("effect %d tick %d: flame height=%g, want 6", id, tick, y)
				}
			}
		}
	}
}

func TestMageBoltGeometryAndCadence(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	texture := render.NewImage(32, 32)
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/icearrow.tga|blackkey=true":  texture,
		"__effectfile_effect/ring_blue.tga|blackkey=true": texture,
	}}
	effect := Instance{EffectID: ColdBolt, Starts: start, HitCount: 3}
	spec, _ := Lookup(ColdBolt)
	firstHeight := 0.0
	for _, tick := range []int{11, 12, 22, 32} {
		frame := render.NewFrame(800, 600)
		m.drawMageBolt(frame, ctx, projection, spec.Components[0], effect, scene.Point3{}, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		want := 0
		if tick >= 12 {
			want = (tick-12)/10 + 1
		}
		if cmds.Len() != want {
			t.Fatalf("tick %d: got %d bolts, want %d", tick, cmds.Len(), want)
		}
		for i := 0; i < cmds.Len(); i++ {
			cmd := cmds.Index(i)
			vertices := cmd.FieldByName("Vertices")
			if vertices.Len() != 8 || !cmd.FieldByName("Options").FieldByName("DepthTest").Bool() ||
				render.Blend(cmd.FieldByName("Options").FieldByName("Blend").Int()) != render.BlendLighter {
				t.Fatal("bolt must be two crossed quads, with additive blending and depth testing")
			}
			height := 0.0
			for j := 0; j < vertices.Len(); j++ {
				height += vertices.Index(j).FieldByName("Y").Float() / float64(vertices.Len())
			}
			if i == 0 && tick == 12 {
				firstHeight = height
				if height < 11 || height > 12 {
					t.Fatal("bolt must start 60 native units above the target")
				}
			} else if i == 0 && height >= firstHeight-4 {
				t.Fatal("bolt did not descend between emission ticks")
			}
		}
	}
	effect.HitCount = 1
	frame := render.NewFrame(800, 600)
	m.drawMageBolt(frame, ctx, projection, spec.Components[0], effect, scene.Point3{}, start.Add(60*db.EffectFrameDuration))
	cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	foundRing := false
	for i := 0; i < cmds.Len(); i++ {
		cmd := cmds.Index(i)
		if cmd.FieldByName("Vertices").Len() == 22 {
			foundRing = true
			if render.Address(cmd.FieldByName("Options").FieldByName("Address").Int()) != render.AddressRepeat {
				t.Fatal("impact ring must repeat its texture around the whole circle")
			}
		}
	}
	if !foundRing {
		t.Fatal("bolt impact ring is missing")
	}
}

func TestMageSoulStrikeFlight(t *testing.T) {
	effect := Instance{ActorOrigin: scene.Point3{X: 10}, ProjectileDirection: scene.Point3{X: 20}}
	a, b := splineProjectilePosition(effect, 1, 90), splineProjectilePosition(effect, 20, 90)
	if a.Z <= 0 || b.Z <= a.Z || b.X <= a.X {
		t.Fatal("Soul Strike must curve outward while moving toward the target")
	}
	end := splineProjectilePosition(effect, 40, 90)
	if math.Abs(end.X-30.7) > 1e-6 || math.Abs(end.Z-.4) > 1e-6 {
		t.Fatalf("unexpected native spline endpoint: %+v", end)
	}
}

func TestMageBoltLandsOnRaisedTerrain(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	effect := Instance{Starts: time.Unix(100, 0)}
	origin := scene.Point3{X: 10, Z: 10}
	start, direction, flatLanding := mageBoltFlight(ctx, effect, origin, 0)
	ctx.World.GAT = &res.GAT{Width: 20, Height: 20, Cells: make([]res.GATCell, 400)}
	for i := range ctx.World.GAT.Cells {
		ctx.World.GAT.Cells[i].Heights = [4]float32{4, 4, 4, 4}
	}
	_, _, raisedLanding := mageBoltFlight(ctx, effect, origin, 0)
	before := scene.Add3(origin, scene.Add3(start, scene.Mul3(direction, .55*float64(raisedLanding))))
	after := scene.Add3(before, scene.Mul3(direction, .55))
	if raisedLanding >= flatLanding || before.Y < 4 || after.Y >= 4 {
		t.Fatalf("bolt missed raised terrain: flat=%d raised=%d height=%g -> %g", flatLanding, raisedLanding, before.Y, after.Y)
	}
}
