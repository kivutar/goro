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

func TestAmbientSpriteLifetimesAndAssets(t *testing.T) {
	for _, tc := range []struct {
		id, frames int
		file       string
	}{
		{Smoke, 50, "굴뚝연기"}, {DragonSmoke, 50, "굴뚝연기"}, {Torch, 250, "torch_01"}, {Firefly, 140, "particle1"}, {Banjjakii, 125, "크리스마스"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != time.Duration(tc.frames)*db.EffectFrameDuration || len(spec.Components) != 1 {
			t.Fatalf("effect %d lifetime/components", tc.id)
		}
		c := spec.Components[0]
		if c.FuncAdapter != effectFuncAmbientSprite || c.SpriteFile != tc.file {
			t.Fatalf("effect %d renderer/asset", tc.id)
		}
	}
}

func TestTorchUsesMapSizeAndAnimationSpeed(t *testing.T) {
	effect := Instance{EffectID: Torch, Params: [4]float32{.6, 3}}
	p, action, motion := ambientSpriteAt(effect, 0, 8)
	if math.Abs(p.size-.6) > 1e-6 || p.Position != (scene.Point3{Y: 2}) || p.Angle != 0 || action != 0 || motion != 3 {
		t.Fatalf("torch = %+v action=%d motion=%d", p, action, motion)
	}
	effect.Params = [4]float32{}
	p, _, motion = ambientSpriteAt(effect, 0, 8)
	if p.size != 1 || motion != 9 {
		t.Fatalf("default torch = %+v motion=%d", p, motion)
	}
}

func TestTorchDrawsSeparateLayersAfterActorSilhouettes(t *testing.T) {
	view := testutil.EffectRenderTestSprite()
	view.Images = map[scene.SpriteFrameKey]*render.Image{{}: render.NewImage(2, 2)}
	view.Act.Actions[0].Animations[0].Layers = []res.ACTLayer{
		{ScaleX: 8, ScaleY: 8, Angle: 169, Color: [4]float32{1, 1, 1, .12}},
		{ScaleX: 2, ScaleY: 2, Color: [4]float32{1, 1, 1, .94}},
		{Index: -1},
	}
	m := &System{EffectViews: map[string]*scene.SpriteView{"torch_01": view}}
	start := time.Unix(100, 0)
	spec, _ := Lookup(Torch)
	frame := render.NewFrame(800, 600)
	m.drawAmbientSprite(frame, testutil.KnightEffectContext(), testutil.Projection(800, 600, 0, 0, 0),
		spec.Components[0], Instance{EffectID: Torch, Starts: start}, scene.Point3{}, start)
	commands := reflect.ValueOf(frame).Elem()
	quads := commands.FieldByName("worldCommands")
	if quads.Len() != 2 || commands.FieldByName("worldBillboards").Len() != 0 {
		t.Fatal("torch must draw halo and flame as separate layers")
	}
	for i, alpha := range []float32{.12, .94} {
		cmd := quads.Index(i)
		options := cmd.FieldByName("Options")
		if !options.FieldByName("AfterBillboards").Bool() || !options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() {
			t.Fatal("torch must test actor silhouettes without writing transparent depth")
		}
		if render.Filter(options.FieldByName("Filter").Int()) != render.FilterLinear ||
			float32(cmd.FieldByName("Vertices").Index(0).FieldByName("ColorA").Float()) != alpha {
			t.Fatal("lost smooth filtering, layer order or ACT opacity")
		}
	}
}

func TestTorchLayerProjectionAndDepth(t *testing.T) {
	// A 128px flame clip from torch_01: the classic ACT loader rounds its
	// top-left corner to (-67, -100), then applies the 2x clip scale.
	layer := res.ACTLayer{X: -7, Y: -73, ScaleX: 2, ScaleY: 2, Color: [4]float32{1, .8, .3, .94}}
	position := scene.Point3{X: 33, Y: 1.2, Z: 117}
	for _, pitch := range []float64{205, 230, 245} {
		for _, yaw := range []float64{0, 90, 225} {
			for _, height := range []int{480, 1080} {
				projection := scene.NewProjection(height*4/3, height, 30, 114, 0, yaw, pitch, 140)
				vertices := torchLayerVertices(projection, position, layer, 128, 128, 1)
				if len(vertices) != 4 {
					t.Fatal("missing torch quad")
				}
				anchor := projection.Project(position.X, position.Z, position.Y)
				_, _, unitsPerPixel, _ := projection.BillboardBasis(position.X, position.Z, position.Y)
				pixels := 1 / (35 * unitsPerPixel)
				depth := projection.Depth(position.X, position.Z, position.Y) * 5
				for i, corner := range [4][2]float64{{-134, -200}, {122, -200}, {122, 56}, {-134, 56}} {
					v := vertices[i]
					point := projection.Project(float64(v.X), float64(v.Z), float64(v.Y))
					wantX, wantY := float64(anchor.X)+corner[0]*pixels, float64(anchor.Y)+corner[1]*pixels
					if math.Abs(float64(point.X)-wantX) > .005 || math.Abs(float64(point.Y)-wantY) > .005 {
						t.Fatalf("torch shifted at pitch/yaw %v/%v: %v; want %v,%v", pitch, yaw, point, wantX, wantY)
					}
					slope := math.Tan(scene.DegreesToRadians(pitch-180)) * .16 * 480 / float64(height)
					oow := 1/(depth+corner[1]*pixels*slope) + .00001
					gotDepth := projection.Depth(float64(v.DepthX), float64(v.DepthZ), float64(v.DepthY))
					if math.Abs(gotDepth-1/oow/5) > .0001 {
						t.Fatalf("torch depth = %v; want %v", gotDepth, 1/oow/5)
					}
					if v.ColorA != layer.Color[3] || v.ColorG != layer.Color[1] {
						t.Fatal("lost ACT tint")
					}
				}
			}
		}
	}
}

func TestTorchHaloRotationKeepsUnrotatedDepthRows(t *testing.T) {
	projection := scene.NewProjection(1280, 960, 0, 0, 0, 45, 230, 140)
	layer := res.ACTLayer{X: -11, Y: -56, ScaleX: 7.46, ScaleY: 7.46, Color: [4]float32{1, 1, 1, .12}}
	position := scene.Point3{Y: 2}
	plain := torchLayerVertices(projection, position, layer, 64, 64, .6)
	layer.Angle, layer.Mirror = 169, true
	rotated := torchLayerVertices(projection, position, layer, 64, 64, .6)
	for i, v := range rotated {
		before := plain[i]
		d1 := projection.Depth(float64(before.DepthX), float64(before.DepthZ), float64(before.DepthY))
		d2 := projection.Depth(float64(v.DepthX), float64(v.DepthZ), float64(v.DepthY))
		if math.Abs(d1-d2) > .0001 {
			t.Fatalf("rotation changed clip row depth: %v -> %v", d1, d2)
		}
		if v.X == before.X && v.Y == before.Y && v.Z == before.Z {
			t.Fatal("halo rotation was ignored")
		}
	}
}

func TestTorchDepthDoesNotDistortTextureAtCloseZoom(t *testing.T) {
	projection := scene.NewProjection(1280, 960, 0, 0, 0, 90, 245, 65)
	position := scene.Point3{Y: 1.2}
	layer := res.ACTLayer{X: -11, Y: -56, ScaleX: 8.9, ScaleY: 8.9, Angle: 169}
	vertices := torchLayerVertices(projection, position, layer, 64, 64, 1)
	if len(vertices) != 4 {
		t.Fatal("halo disappeared at close zoom")
	}
	for _, v := range vertices {
		visibleDepth := projection.Depth(float64(v.X), float64(v.Z), float64(v.Y))
		anchorDepth := projection.Depth(position.X, position.Z, position.Y)
		if math.Abs(visibleDepth-anchorDepth) > .0001 {
			t.Fatal("depth correction changed texture interpolation")
		}
		if d := projection.Depth(float64(v.DepthX), float64(v.DepthZ), float64(v.DepthY)); d < .99 || !scene.IsFinite(d) {
			t.Fatalf("invalid halo depth: %v", d)
		}
	}
}

func TestSmokeRisesWithoutGrowingAndFadesAtEnd(t *testing.T) {
	e := Instance{EffectID: Smoke, Starts: time.Unix(1, 0)}
	first, _, _ := ambientSpriteAt(e, 0, 0)
	last, _, _ := ambientSpriteAt(e, 0, 49)
	if first.size != 1.5 || last.size != first.size || first.Position.Y < 1.8 || last.Position.Y <= first.Position.Y {
		t.Fatalf("puff: first=%+v last=%+v", first, last)
	}
	if first.Alpha < .99 || last.Alpha != 0 {
		t.Fatalf("puff fade: %v -> %v", first.Alpha, last.Alpha)
	}
	if math.Abs(last.Angle-scene.DegreesToRadians(37.5)) > 1e-9 {
		t.Fatalf("roll=%v", last.Angle)
	}
}

func TestFireflyTwinkleAndMovement(t *testing.T) {
	path := newFireflyPath(Instance{EffectID: Firefly, Starts: time.Unix(3, 0)})
	if path[0].size < .3 || path[0].size >= 1 {
		t.Fatalf("size=%v", path[0].size)
	}
	bright := false
	for frame, p := range path {
		maximum := 80.0 / 255
		if frame >= 40 && frame < 100 {
			maximum = 200.0 / 255
		}
		if p.Alpha < 0 || p.Alpha > maximum {
			t.Fatalf("alpha at %d = %v", frame, p.Alpha)
		}
		if p.Alpha > 120.0/255 {
			bright = true
		}
		if frame > 0 {
			delta := scene.Sub3(p.Position, path[frame-1].Position)
			if scene.Dot3(delta, delta) > .1*.1 {
				t.Fatalf("movement jumps at %d", frame)
			}
		}
	}
	if !bright || path[0].Position == path[139].Position {
		t.Fatal("firefly does not wander and brighten")
	}
}

func TestColoredMapTorchesUseNativeTexturesAndTiming(t *testing.T) {
	for _, tc := range []struct {
		id          int
		first, last string
	}{
		{TorchRed, "effect/torch_red01.bmp", "effect/torch_red13.bmp"},
		{TorchGreen, "effect/torch_green01.bmp", "effect/torch_green13.bmp"},
		{TorchPurple, "effect/torch_violet01.bmp", "effect/torch_violet13.bmp"},
	} {
		spec, _ := Lookup(tc.id)
		c := spec.Components[0]
		if c.FuncAdapter != effectFuncMapTexture || len(c.TextureFiles) != 13 || c.TextureFiles[0] != tc.first || c.TextureFiles[12] != tc.last {
			t.Fatalf("wrong texture sequence for %d", tc.id)
		}
		e := Instance{EffectID: tc.id, Params: [4]float32{9}}
		size, height, angle, alpha, frame := mapTextureFrame(e, 4)
		if math.Abs(size-9*math.Sqrt2/5) > 1e-9 || height != 2 || angle != 0 || alpha != 130 || frame != 0 {
			t.Fatalf("torch geometry or first frame: %v %v %v %v %v", size, height, angle, alpha, frame)
		}
		_, _, _, _, frame = mapTextureFrame(e, 5)
		if frame != 1 {
			t.Fatal("frame does not advance after six ticks")
		}
		_, _, _, _, frame = mapTextureFrame(e, 77)
		if frame != 0 {
			t.Fatal("texture sequence does not wrap")
		}
	}
}

func TestGlowUsesBillboardRadiusAndSlowPulse(t *testing.T) {
	e := Instance{EffectID: Glow2, Params: [4]float32{70}}
	size, height, _, alpha, _ := mapTextureFrame(e, 44)
	if math.Abs(size-70*1.05*math.Sqrt2/5) > 1e-9 || height != 0 || alpha != 50 {
		t.Fatalf("glow=%v %v %v", size, height, alpha)
	}
}

func TestMapParticleUpdatesIndependentOfRenderRate(t *testing.T) {
	for _, id := range []int{Glow4, MapGhost} {
		e := Instance{EffectID: id, Starts: time.Unix(3, 0)}
		frequent, slow := newMapParticleState(e), newMapParticleState(e)
		for frame := 0; frame <= 180; frame++ {
			frequent.advance(e, frame)
		}
		for frame := 0; frame <= 180; frame += 3 {
			slow.advance(e, frame)
		}
		for i := range frequent.groups {
			if frequent.groups[i] != slow.groups[i] {
				t.Fatalf("effect %d depends on drawing frequency", id)
			}
		}
	}
}

func TestBubbleDropFallsAndKeepsItsShortTrail(t *testing.T) {
	e := Instance{EffectID: BubbleDrop}
	start, end := bubbleDropAt(e, 0), bubbleDropAt(e, 49)
	if start.Position.Y != 9.8 || math.Abs(end.Position.Y-.2) > 1e-9 || end.Alpha != 190.0/255 {
		t.Fatalf("drop=%+v -> %+v", start, end)
	}
	if p := bubbleDropAt(Instance{Params: [4]float32{4.5}}, 3); math.Abs(p.Position.Y-.3) > 1e-9 || p.Alpha != 70.0/255 {
		t.Fatalf("fractional height stop tick: %+v", p)
	}
	if p := bubbleDropAt(e, 68); p.Alpha != 0 {
		t.Fatalf("drop did not fade: %+v", p)
	}
}

func TestEnvironmentAssetsAndRendering(t *testing.T) {
	manager := testutil.RealDataManager(t)
	ctx := testutil.KnightEffectContext()
	ctx.Resources = manager
	m := &System{}
	now := time.Unix(20, 0)
	projection := testutil.Projection(800, 600, 0, 0, 0)
	for _, tc := range []struct{ id, tick int }{
		{Smoke, 20}, {DragonSmoke, 20}, {Torch, 8}, {Banjjakii, 100}, {Firefly, 40},
		{TorchRed, 20}, {TorchGreen, 20}, {TorchPurple, 20}, {Glow1, 30}, {Glow2, 30},
		{Glow4, 180}, {MapGhost, 120}, {MapPillar, 360}, {MapPillar2, 360}, {MapPillar3, 360}, {MapPillar4, 360},
		{BubbleDrop, 20}, {Rainbow, 200},
	} {
		spec, _ := Lookup(tc.id)
		effect := Instance{EffectID: tc.id, Starts: now.Add(-time.Duration(tc.tick) * db.EffectFrameDuration)}
		m.PrepareEnvironmentEffect(&effect, projection)
		at := now
		if tc.id == MapGhost {
			at = effect.Starts.Add(time.Duration((effect.mapParticles.groups[0].position.Y+20)/.6) * db.EffectFrameDuration)
		}
		frame := render.NewFrame(800, 600)
		for i, c := range spec.Components {
			if c.SpriteFile != "" && m.EffectSpriteView(manager, c.SpriteFile) == nil {
				t.Fatalf("missing sprite %s", c.SpriteFile)
			}
			files := append([]string{}, c.TextureFiles...)
			if c.TextureFile != "" {
				files = append(files, c.TextureFile)
			}
			for _, file := range files {
				if m.EffectFileTexture(manager, file) == nil {
					t.Fatalf("missing texture %s", file)
				}
			}
			duration := m.WorldEffectResolvedComponentDuration(ctx, spec, c, effect)
			m.DrawWorldEffectComponent(frame, ctx, projection, effect, c, i, 0, 0, 0, float64(at.Sub(effect.Starts))/float64(duration), duration, at)
		}
		commands := reflect.ValueOf(frame).Elem()
		if commands.FieldByName("worldBillboards").Len()+commands.FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d emitted no geometry at tick %d", tc.id, tc.tick)
		}
	}
}

func TestBubbleDropUsesOneBluePassPerTrail(t *testing.T) {
	m := &System{textures: map[string]*render.Image{"__effectfile_effect/썬더스톰파티클.tga|blackkey=true": render.NewImage(2, 2)}}
	spec, _ := Lookup(BubbleDrop)
	now := time.Unix(20, 0)
	effect := Instance{EffectID: BubbleDrop, Starts: now.Add(-20 * db.EffectFrameDuration)}
	frame := render.NewFrame(800, 600)
	m.drawBubbleDrop(frame, testutil.KnightEffectContext(), testutil.Projection(800, 600, 0, 0, 0), spec.Components[0], effect, scene.Point3{}, now)
	draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if draws.Len() != 4 {
		t.Fatalf("got %d draws, want one pass for the drop and each of its 3 trails", draws.Len())
	}
	for i, alpha := range []float64{200, 100, 75, 50} {
		draw := draws.Index(i)
		for name, value := range map[string]float64{"ColorR": 130, "ColorG": 130, "ColorB": 250, "ColorA": alpha} {
			if math.Abs(draw.FieldByName(name).Float()-value/255) > .00001 {
				t.Fatalf("wrong %s at trail %d", name, i)
			}
		}
	}
}
