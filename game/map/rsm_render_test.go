package maps

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestCalculateRSMBoundsSeparatesMainNodeFromModel(t *testing.T) {
	rsm := &res.RSM{
		MainNodeName: "root",
		Nodes: []res.RSMNode{
			{
				Name:   "root",
				Matrix: identityRSMMatrix(),
				Scale:  res.RSMVector3{X: 1, Y: 1, Z: 1},
				Vertices: []res.RSMVector3{
					{X: -1, Y: 0, Z: -1},
					{X: 1, Y: 2, Z: 1},
				},
			},
			{
				Name:       "child",
				ParentName: "root",
				Matrix:     identityRSMMatrix(),
				Position:   res.RSMVector3{X: 100, Y: 0, Z: 0},
				Scale:      res.RSMVector3{X: 1, Y: 1, Z: 1},
				Vertices: []res.RSMVector3{
					{X: 0, Y: 0, Z: 0},
					{X: 2, Y: 3, Z: 2},
				},
			},
		},
	}

	bounds := calculateRSMBoundsForNodes(rsm, []int{0, 1})
	if bounds.main.min.X != -1 || bounds.main.max.X != 1 {
		t.Fatalf("main bounds x = %.1f..%.1f, want -1..1", bounds.main.min.X, bounds.main.max.X)
	}
	if bounds.model.min.X != -1 || bounds.model.max.X != 102 {
		t.Fatalf("model bounds x = %.1f..%.1f, want -1..102", bounds.model.min.X, bounds.model.max.X)
	}
}

func TestRSMPlacementContextIncludesSelectedNodeAncestors(t *testing.T) {
	rsm := &res.RSM{
		MainNodeName: "root",
		Nodes: []res.RSMNode{
			{
				Name:   "root",
				Matrix: identityRSMMatrix(),
				Scale:  res.RSMVector3{X: 1, Y: 1, Z: 1},
				Vertices: []res.RSMVector3{
					{X: -1, Y: 0, Z: -1},
					{X: 1, Y: 2, Z: 1},
				},
			},
			{
				Name:       "child",
				ParentName: "root",
				Matrix:     identityRSMMatrix(),
				Position:   res.RSMVector3{X: 100},
				Scale:      res.RSMVector3{X: 1, Y: 1, Z: 1},
				Vertices: []res.RSMVector3{
					{X: 0, Y: 0, Z: 0},
					{X: 2, Y: 3, Z: 2},
				},
			},
			{
				Name:       "sibling",
				ParentName: "root",
				Matrix:     identityRSMMatrix(),
				Scale:      res.RSMVector3{X: 1, Y: 1, Z: 1},
				Vertices: []res.RSMVector3{
					{X: 10, Y: 0, Z: 0},
					{X: 11, Y: 1, Z: 1},
				},
			},
		},
	}

	world := &System{}
	context, ok := world.rsmPlacementContext(rsm, &res.RSW{}, ModelPlacement{
		Model: res.RSWModel{
			NodeName: "child",
			Scale:    res.RSWVector3{X: 1, Y: 1, Z: 1},
		},
	})
	if !ok {
		t.Fatal("placement context was not built")
	}
	if len(context.nodeIndices) != 2 || context.nodeIndices[0] != 0 || context.nodeIndices[1] != 1 {
		t.Fatalf("node indices = %#v, want selected node plus ancestor", context.nodeIndices)
	}
}

func TestRSMDrawOptionsLayerHorizontalSurfaces(t *testing.T) {
	horizontal := modelWorldTriangle{
		verts: [3]scene.Point3{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 0, Z: 1},
		},
	}
	vertical := modelWorldTriangle{
		verts: [3]scene.Point3{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 1, Z: 0},
		},
	}

	early := rsmDrawOptionsForTriangle(render.FilterLinear, render.AddressClampToEdge, horizontal, 3)
	later := rsmDrawOptionsForTriangle(render.FilterLinear, render.AddressClampToEdge, horizontal, 4)
	wall := rsmDrawOptionsForTriangle(render.FilterLinear, render.AddressClampToEdge, vertical, 4)

	if early.DepthBias <= rsmModelDepthBias {
		t.Fatalf("horizontal RSM depth bias = %.10f, want above base %.10f", early.DepthBias, rsmModelDepthBias)
	}
	if later.DepthBias <= early.DepthBias {
		t.Fatalf("later horizontal RSM depth bias = %.10f, want above earlier %.10f", later.DepthBias, early.DepthBias)
	}
	if wall.DepthBias != rsmModelDepthBias {
		t.Fatalf("vertical RSM depth bias = %.10f, want base %.10f", wall.DepthBias, rsmModelDepthBias)
	}
	capped := rsmDrawOptionsForTriangle(render.FilterLinear, render.AddressClampToEdge, horizontal, 10000)
	if capped.DepthBias >= rsmModelDepthBias*1.25 {
		t.Fatalf("capped horizontal RSM depth bias = %.10f, want below actor-occluding bias %.10f", capped.DepthBias, rsmModelDepthBias*1.25)
	}
	wrapped := rsmDrawOptionsForTriangle(render.FilterLinear, render.AddressClampToEdge, horizontal, 4+rsmHorizontalDepthBiasLayers)
	if wrapped.DepthBias != later.DepthBias {
		t.Fatalf("wrapped horizontal RSM depth bias = %.10f, want layer repeat %.10f", wrapped.DepthBias, later.DepthBias)
	}
}

func TestRSMDrawOptionsForVerticesMatchesTriangleHelper(t *testing.T) {
	triangles := []modelWorldTriangle{
		{
			verts: [3]scene.Point3{
				{X: 0, Y: 0, Z: 0},
				{X: 1, Y: 0, Z: 0},
				{X: 0, Y: 0, Z: 1},
			},
		},
		{
			verts: [3]scene.Point3{
				{X: 0, Y: 0, Z: 0},
				{X: 1, Y: 0, Z: 0},
				{X: 0, Y: 1, Z: 0},
			},
		},
	}
	for _, tri := range triangles {
		fromTri := rsmDrawOptionsForTriangle(render.FilterLinear, render.AddressClampToEdge, tri, 7)
		fromVerts := rsmDrawOptionsForVertices(render.FilterLinear, render.AddressClampToEdge, tri.verts[0], tri.verts[1], tri.verts[2], 7)
		if *fromTri != fromVerts {
			t.Fatalf("draw options mismatch: triangle=%#v vertices=%#v", *fromTri, fromVerts)
		}
	}
}

func TestAnimatedRSMPlacementClearsScratchBatches(t *testing.T) {
	rsm := &res.RSM{
		Nodes: []res.RSMNode{
			{
				Name:   "root",
				Matrix: identityRSMMatrix(),
				Scale:  res.RSMVector3{X: 1, Y: 1, Z: 1},
				Vertices: []res.RSMVector3{
					{X: 0, Y: 0, Z: 0},
					{X: 1, Y: 0, Z: 0},
					{X: 0, Y: 1, Z: 0},
				},
				Faces: []res.RSMFace{
					{VertexIndices: [3]uint16{0, 1, 2}},
				},
			},
		},
	}
	world := &System{}
	screen := render.NewFrame(320, 240)
	screen.BeginFrame()

	world.DrawAnimatedRSMPlacement(screen, nil, &res.RSW{}, nil, rsm, ModelPlacement{
		Index: 3,
		Model: res.RSWModel{Scale: res.RSWVector3{X: 1, Y: 1, Z: 1}},
	}, 0)

	if world.WhitePixel == nil {
		t.Fatal("animated RSM draw did not use the fallback texture")
	}
	if len(world.rsmAnimScratch.batches) != 0 {
		t.Fatalf("scratch batches length = %d, want 0", len(world.rsmAnimScratch.batches))
	}
	if cap(world.rsmAnimScratch.batches) == 0 {
		t.Fatal("scratch batches were not reused")
	}
	for i, batch := range world.rsmAnimScratch.batches[:cap(world.rsmAnimScratch.batches)] {
		if batch.Screen != nil || batch.Texture != nil || batch.Verts != nil || batch.Indices != nil {
			t.Fatalf("scratch batch %d retained frame data: %#v", i, batch)
		}
	}
	if cap(world.rsmAnimScratch.worldVerts) < 3 {
		t.Fatalf("scratch world verts capacity = %d, want at least 3", cap(world.rsmAnimScratch.worldVerts))
	}
}

func TestRSMInstanceMatrixUsesParsedRSWModelY(t *testing.T) {
	rsm := &res.RSM{}
	placement := res.RSWModel{
		Position: res.RSWVector3{X: 10, Y: -7, Z: 20},
		Scale:    res.RSWVector3{X: 1, Y: 1, Z: 1},
	}

	matrix := buildRSMInstanceMatrix(rsm, placement, 110, 220, modelBounds{})
	point := scene.Mat4TransformPoint(matrix, scene.Point3{})
	if point.Y != -7 {
		t.Fatalf("instance y = %.1f, want -7", point.Y)
	}
}

func TestRSMInstanceMatrixMirrorsVerticalBasisRotations(t *testing.T) {
	rsm := &res.RSM{}
	tests := []struct {
		name      string
		rotation  res.RSWVector3
		point     scene.Point3
		wantPoint scene.Point3
	}{
		{
			name:      "pitch",
			rotation:  res.RSWVector3{X: 90},
			point:     scene.Point3{Y: 1},
			wantPoint: scene.Point3{Z: -1},
		},
		{
			name:      "yaw",
			rotation:  res.RSWVector3{Y: 90},
			point:     scene.Point3{X: 1},
			wantPoint: scene.Point3{Z: -1},
		},
		{
			name:      "roll",
			rotation:  res.RSWVector3{Z: 90},
			point:     scene.Point3{X: 1},
			wantPoint: scene.Point3{Y: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matrix := buildRSMInstanceMatrix(rsm, res.RSWModel{
				Rotation: tt.rotation,
				Scale:    res.RSWVector3{X: 1, Y: 1, Z: 1},
			}, 0, 0, modelBounds{})
			got := scene.Mat4TransformPoint(matrix, tt.point)
			if math.Abs(got.X-tt.wantPoint.X) > 0.0001 || math.Abs(got.Y-tt.wantPoint.Y) > 0.0001 || math.Abs(got.Z-tt.wantPoint.Z) > 0.0001 {
				t.Fatalf("point = (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)", got.X, got.Y, got.Z, tt.wantPoint.X, tt.wantPoint.Y, tt.wantPoint.Z)
			}
		})
	}
}

func testRSMFaceLightModel(shadeType int32) *res.RSM {
	return &res.RSM{
		VersionMajor: 1,
		VersionMinor: 4,
		ShadeType:    shadeType,
		MainNodeName: "root",
		Nodes: []res.RSMNode{{
			Name:     "root",
			Matrix:   identityRSMMatrix(),
			Scale:    res.RSMVector3{X: 1, Y: 1, Z: 1},
			Vertices: []res.RSMVector3{{}, {X: 1}, {Z: 1}},
			TextureVertices: []res.RSMTextureVertex{
				{}, {}, {}, {Color: [4]float32{1, 1, 1, 1}},
			},
			Faces: []res.RSMFace{
				{VertexIndices: [3]uint16{0, 1, 2}},
				{VertexIndices: [3]uint16{0, 2, 1}},
				{VertexIndices: [3]uint16{0, 1, 2}, TextureVertexIndices: [3]uint16{3, 3, 3}},
			},
		}},
	}
}

func testRSMFaceLights(t *testing.T, rsm *res.RSM, gnd *res.GND) [][3]color.RGBA {
	t.Helper()
	// Light from straight above: ambient 0.2, diffuse 0.5.
	rsw := &res.RSW{Light: res.RSWLight{Diffuse: [3]float32{0.5, 0.5, 0.5}, Ambient: [3]float32{0.2, 0.2, 0.2}}}
	m := &System{}
	visible := ModelPlacement{Model: res.RSWModel{Scale: res.RSWVector3{X: 1, Y: 1, Z: 1}}, BaseX: 4, BaseY: 4}
	context, ok := m.rsmPlacementContext(rsm, rsw, visible)
	if !ok {
		t.Fatal("no placement context")
	}
	return m.rsmPlacementFaceLights(rsm, &rsm.Nodes[0], context, visible, gnd)
}

func TestRSMFaceLightsShadeNoneIsUnlit(t *testing.T) {
	lights := testRSMFaceLights(t, testRSMFaceLightModel(rsmShadeNone), nil)
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for i, face := range lights {
		for k, got := range face {
			if got != white {
				t.Fatalf("face %d corner %d = %+v, want unlit white", i, k, got)
			}
		}
	}
}

func TestRSMFaceLightsFlatShadingLightsFrontFacesOnly(t *testing.T) {
	lights := testRSMFaceLights(t, testRSMFaceLightModel(rsmShadeFlat), nil)
	litLevel, ambientLevel := 0.7, 0.2
	lit, ambient := uint8(litLevel*255), uint8(ambientLevel*255)
	front, back := lights[0][0].R, lights[1][0].R
	if !(front == lit && back == ambient) && !(front == ambient && back == lit) {
		t.Fatalf("opposite faces = %d and %d, want %d and %d", front, back, lit, ambient)
	}
	for _, face := range lights[:2] {
		if face[0] != face[1] || face[0] != face[2] {
			t.Fatalf("flat face corners differ: %+v", face)
		}
	}
	// Coloured texture vertices use a 128 scale (C3dNode::UpdateVertexColor).
	want := uint8(ambientLevel * 128)
	if front == lit {
		want = uint8(litLevel * 128)
	}
	if got := lights[2][0].R; got != want {
		t.Fatalf("coloured texture vertex = %d, want %d", got, want)
	}
}

func TestRSMFaceLightsSmoothShadingAveragesSmoothingGroup(t *testing.T) {
	rsm := testRSMFaceLightModel(rsmShadeSmooth)
	node := &rsm.Nodes[0]
	node.Vertices = []res.RSMVector3{{}, {X: 1}, {Z: 1}, {X: 1, Y: 1, Z: 1}}
	node.Faces = []res.RSMFace{
		{VertexIndices: [3]uint16{0, 1, 2}},
		{VertexIndices: [3]uint16{1, 3, 2}},
	}
	lights := testRSMFaceLights(t, rsm, nil)
	if lights[0][0] == lights[0][1] {
		t.Fatalf("shared vertex took the flat face colour: %+v", lights[0])
	}
	if lights[0][1] != lights[1][0] || lights[0][2] != lights[1][2] {
		t.Fatalf("shared vertices differ between faces: %+v %+v", lights[0], lights[1])
	}
}

func TestRSMFaceLightsAddGroundCellColour(t *testing.T) {
	var lightmap res.GNDLightmap
	lightmap.Color[4][4] = color.RGBA{R: 20, G: 200, B: 0, A: 255}
	gnd := &res.GND{Width: 8, Height: 8, Lightmaps: []res.GNDLightmap{lightmap}, Surfaces: []res.GNDSurface{{LightmapID: 0}}, Cells: make([]res.GNDCell, 64)}
	for i := range gnd.Cells {
		gnd.Cells[i].Front, gnd.Cells[i].Right = -1, -1
	}
	unlit := testRSMFaceLights(t, testRSMFaceLightModel(rsmShadeNone), gnd)
	if got := unlit[0][0]; got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("unlit model with ground colour = %+v, want white", got)
	}
	plain := testRSMFaceLights(t, testRSMFaceLightModel(rsmShadeFlat), nil)
	tinted := testRSMFaceLights(t, testRSMFaceLightModel(rsmShadeFlat), gnd)
	for k := range tinted[1] {
		want := color.RGBA{R: plain[1][k].R + 40, G: 255, B: plain[1][k].B, A: 255}
		if tinted[1][k] != want {
			t.Fatalf("corner %d = %+v, want %+v", k, tinted[1][k], want)
		}
	}
}

func TestRSMAnimationFrameLoopsOnMillisecondTick(t *testing.T) {
	rsm := &res.RSM{
		AnimLength: 12,
		Nodes: []res.RSMNode{
			{
				Name: "root",
				PositionKeyframes: []res.RSMPositionKeyframe{
					{Frame: 0},
					{Frame: 12, Pos: res.RSMVector3{X: 12}},
				},
			},
		},
	}

	frame, animated := RSMAnimationFrame(rsm, res.RSWModel{AnimType: 2}, time.UnixMilli(1015))
	if !animated {
		t.Fatal("model should be animated")
	}
	if frame != 7 {
		t.Fatalf("animation frame = %d, want 7", frame)
	}
}

func TestRSMAnimationFrameIgnoresRSWAnimationSpeedLikeRobr(t *testing.T) {
	rsm := &res.RSM{
		AnimLength: 6400,
		Nodes: []res.RSMNode{
			{
				Name: "root",
				RotationKeyframes: []res.RSMRotationKeyframe{
					{Frame: 0, Quaternion: [4]float32{0, 0, 0, 1}},
					{Frame: 160, Quaternion: [4]float32{0, 0, 1, 0}},
				},
			},
		},
	}

	frame, animated := RSMAnimationFrame(rsm, res.RSWModel{AnimType: 2, AnimSpeed: 0.7}, time.UnixMilli(160))
	if !animated {
		t.Fatal("model should be animated")
	}
	if frame != 160 {
		t.Fatalf("animation frame = %d, want 160", frame)
	}

	frame, animated = RSMAnimationFrame(rsm, res.RSWModel{AnimType: 2, AnimSpeed: 0.7}, time.UnixMilli(1000))
	if !animated {
		t.Fatal("model should be animated")
	}
	if frame != 1000 {
		t.Fatalf("animation frame = %d, want 1000", frame)
	}
}

func TestRSMAnimationFrameAnimatesKeyframedModelsWithoutRSWAnimType(t *testing.T) {
	rsm := &res.RSM{
		AnimLength: 12,
		Nodes: []res.RSMNode{
			{
				Name: "root",
				PositionKeyframes: []res.RSMPositionKeyframe{
					{Frame: 0},
					{Frame: 12, Pos: res.RSMVector3{X: 12}},
				},
			},
		},
	}

	frame, animated := RSMAnimationFrame(rsm, res.RSWModel{AnimType: 0}, time.UnixMilli(14))
	if !animated {
		t.Fatal("model should be animated")
	}
	if frame != 2 {
		t.Fatalf("animation frame = %d, want 2", frame)
	}
}

func TestAnimatedRSMNodeMatricesRetainsOnlyLatestFramePerModel(t *testing.T) {
	rsm := &res.RSM{Nodes: []res.RSMNode{{Name: "root"}}}
	mode := &System{}

	first := mode.animatedRSMNodeMatrices(rsm, 1)
	first["cache-marker"] = scene.Mat4Identity()
	if _, ok := mode.animatedRSMNodeMatrices(rsm, 1)["cache-marker"]; !ok {
		t.Fatal("same animation frame did not reuse cached node matrices")
	}

	for frame := 2; frame <= 1000; frame++ {
		mode.animatedRSMNodeMatrices(rsm, frame)
	}
	if got := len(mode.rsmAnimNodes); got != 1 {
		t.Fatalf("animation cache contains %d entries for one model, want 1", got)
	}
	if cached := mode.rsmAnimNodes[rsm]; cached.frame != 1000 {
		t.Fatalf("cached animation frame = %d, want 1000", cached.frame)
	}
}

func TestRSMFaceLightsUseModelAlpha(t *testing.T) {
	rsm := testRSMFaceLightModel(rsmShadeNone)
	rsm.Alpha = 0.5
	if got := testRSMFaceLights(t, rsm, nil)[0][0].A; got < 126 || got > 129 {
		t.Fatalf("alpha = %d, want about 128", got)
	}
}

func TestBuildRSMNodeMatricesSamplesAnimatedPositionAndScale(t *testing.T) {
	rsm := &res.RSM{Nodes: []res.RSMNode{
		{
			Name:   "root",
			Matrix: identityRSMMatrix(),
			PositionKeyframes: []res.RSMPositionKeyframe{
				{Frame: 0, Pos: res.RSMVector3{}},
				{Frame: 10, Pos: res.RSMVector3{X: 10}},
			},
			ScaleKeyframes: []res.RSMScaleKeyframe{
				{Frame: 0, Scale: res.RSMVector3{X: 1, Y: 1, Z: 1}},
				{Frame: 10, Scale: res.RSMVector3{X: 3, Y: 3, Z: 3}},
			},
		},
	}}

	matrix := buildRSMNodeMatrices(rsm, 5)["root"]
	got := scene.Mat4TransformPoint(matrix, scene.Point3{X: 1})
	want := scene.Point3{X: 7}
	if math.Abs(got.X-want.X) > 0.0001 || math.Abs(got.Y-want.Y) > 0.0001 || math.Abs(got.Z-want.Z) > 0.0001 {
		t.Fatalf("animated point = (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)", got.X, got.Y, got.Z, want.X, want.Y, want.Z)
	}
}

func TestBuildRSMNodeMatricesSlerpsAnimatedRotation(t *testing.T) {
	rsm := &res.RSM{Nodes: []res.RSMNode{
		{
			Name:   "root",
			Matrix: identityRSMMatrix(),
			Scale:  res.RSMVector3{X: 1, Y: 1, Z: 1},
			RotationKeyframes: []res.RSMRotationKeyframe{
				{Frame: 0, Quaternion: [4]float32{0, 0, 0, 1}},
				{Frame: 10, Quaternion: [4]float32{0, 0, 1, 0}},
			},
		},
	}}

	matrix := buildRSMNodeMatrices(rsm, 5)["root"]
	got := scene.Mat4TransformPoint(matrix, scene.Point3{X: 1})
	want := scene.Point3{Y: 1}
	if math.Abs(got.X-want.X) > 0.0001 || math.Abs(got.Y-want.Y) > 0.0001 || math.Abs(got.Z-want.Z) > 0.0001 {
		t.Fatalf("animated rotation point = (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)", got.X, got.Y, got.Z, want.X, want.Y, want.Z)
	}
}

func identityRSMMatrix() [9]float32 {
	return [9]float32{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}
}
