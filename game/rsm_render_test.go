package game

import (
	"image/color"
	"math"
	"testing"
	"time"

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
	if bounds.main.min.x != -1 || bounds.main.max.x != 1 {
		t.Fatalf("main bounds x = %.1f..%.1f, want -1..1", bounds.main.min.x, bounds.main.max.x)
	}
	if bounds.model.min.x != -1 || bounds.model.max.x != 102 {
		t.Fatalf("model bounds x = %.1f..%.1f, want -1..102", bounds.model.min.x, bounds.model.max.x)
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

	world := &WorldMode{}
	context, ok := world.rsmPlacementContext(rsm, &res.RSW{}, visibleRSMPlacement{
		model: res.RSWModel{
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
		verts: [3]modelPoint3{
			{x: 0, y: 0, z: 0},
			{x: 1, y: 0, z: 0},
			{x: 0, y: 0, z: 1},
		},
	}
	vertical := modelWorldTriangle{
		verts: [3]modelPoint3{
			{x: 0, y: 0, z: 0},
			{x: 1, y: 0, z: 0},
			{x: 0, y: 1, z: 0},
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
			verts: [3]modelPoint3{
				{x: 0, y: 0, z: 0},
				{x: 1, y: 0, z: 0},
				{x: 0, y: 0, z: 1},
			},
		},
		{
			verts: [3]modelPoint3{
				{x: 0, y: 0, z: 0},
				{x: 1, y: 0, z: 0},
				{x: 0, y: 1, z: 0},
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
	world := &WorldMode{}
	screen := render.NewFrame(320, 240)
	screen.BeginFrame()

	world.drawAnimatedRSMPlacement(screen, nil, &res.RSW{}, rsm, visibleRSMPlacement{
		index: 3,
		model: res.RSWModel{Scale: res.RSWVector3{X: 1, Y: 1, Z: 1}},
	}, 0)

	if world.whitePixel == nil {
		t.Fatal("animated RSM draw did not use the fallback texture")
	}
	if len(world.rsmAnimScratch.batches) != 0 {
		t.Fatalf("scratch batches length = %d, want 0", len(world.rsmAnimScratch.batches))
	}
	if cap(world.rsmAnimScratch.batches) == 0 {
		t.Fatal("scratch batches were not reused")
	}
	for i, batch := range world.rsmAnimScratch.batches[:cap(world.rsmAnimScratch.batches)] {
		if batch.screen != nil || batch.texture != nil || batch.verts != nil || batch.indices != nil {
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
	point := mat4TransformPoint(matrix, modelPoint3{})
	if point.y != -7 {
		t.Fatalf("instance y = %.1f, want -7", point.y)
	}
}

func TestRSMInstanceMatrixMirrorsVerticalBasisRotations(t *testing.T) {
	rsm := &res.RSM{}
	tests := []struct {
		name      string
		rotation  res.RSWVector3
		point     modelPoint3
		wantPoint modelPoint3
	}{
		{
			name:      "pitch",
			rotation:  res.RSWVector3{X: 90},
			point:     modelPoint3{y: 1},
			wantPoint: modelPoint3{z: -1},
		},
		{
			name:      "yaw",
			rotation:  res.RSWVector3{Y: 90},
			point:     modelPoint3{x: 1},
			wantPoint: modelPoint3{z: -1},
		},
		{
			name:      "roll",
			rotation:  res.RSWVector3{Z: 90},
			point:     modelPoint3{x: 1},
			wantPoint: modelPoint3{y: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matrix := buildRSMInstanceMatrix(rsm, res.RSWModel{
				Rotation: tt.rotation,
				Scale:    res.RSWVector3{X: 1, Y: 1, Z: 1},
			}, 0, 0, modelBounds{})
			got := mat4TransformPoint(matrix, tt.point)
			if math.Abs(got.x-tt.wantPoint.x) > 0.0001 || math.Abs(got.y-tt.wantPoint.y) > 0.0001 || math.Abs(got.z-tt.wantPoint.z) > 0.0001 {
				t.Fatalf("point = (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)", got.x, got.y, got.z, tt.wantPoint.x, tt.wantPoint.y, tt.wantPoint.z)
			}
		})
	}
}

func TestRSMFaceColorShadeNoneUsesDefaultModelNormal(t *testing.T) {
	lighting := sceneLighting{
		direction: modelPoint3{y: -1},
		diffuse:   modelPoint3{x: 1, y: 1, z: 1},
		env:       modelPoint3{x: 1, y: 1, z: 1},
	}
	triangleNormalWouldFaceAway := [3]modelPoint3{
		{},
		{x: 1},
		{z: -1},
	}

	got := rsmFaceColor(&res.RSM{ShadeType: 0}, "model.bmp",
		triangleNormalWouldFaceAway[0],
		triangleNormalWouldFaceAway[1],
		triangleNormalWouldFaceAway[2],
		lighting,
	)
	want := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if got != want {
		t.Fatalf("shade-none face color = %#v, want %#v", got, want)
	}

	flat := rsmFaceColor(&res.RSM{ShadeType: 1}, "model.bmp",
		triangleNormalWouldFaceAway[0],
		triangleNormalWouldFaceAway[1],
		triangleNormalWouldFaceAway[2],
		lighting,
	)
	if flat == want {
		t.Fatalf("flat-shaded face unexpectedly matched shade-none color %#v", flat)
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

	frame, animated := rsmAnimationFrame(rsm, res.RSWModel{AnimType: 2}, time.UnixMilli(1015))
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

	frame, animated := rsmAnimationFrame(rsm, res.RSWModel{AnimType: 2, AnimSpeed: 0.7}, time.UnixMilli(160))
	if !animated {
		t.Fatal("model should be animated")
	}
	if frame != 160 {
		t.Fatalf("animation frame = %d, want 160", frame)
	}

	frame, animated = rsmAnimationFrame(rsm, res.RSWModel{AnimType: 2, AnimSpeed: 0.7}, time.UnixMilli(1000))
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

	frame, animated := rsmAnimationFrame(rsm, res.RSWModel{AnimType: 0}, time.UnixMilli(14))
	if !animated {
		t.Fatal("model should be animated")
	}
	if frame != 2 {
		t.Fatalf("animation frame = %d, want 2", frame)
	}
}

func TestAnimatedRSMNodeMatricesRetainsOnlyLatestFramePerModel(t *testing.T) {
	rsm := &res.RSM{Nodes: []res.RSMNode{{Name: "root"}}}
	mode := &WorldMode{}

	first := mode.animatedRSMNodeMatrices(rsm, 1)
	first["cache-marker"] = mat4Identity()
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

func TestRSMFaceColorUsesModelAlpha(t *testing.T) {
	got := rsmFaceColor(&res.RSM{ShadeType: 0, Alpha: 0.5}, "model.bmp",
		modelPoint3{x: 0, y: 0, z: 0},
		modelPoint3{x: 1, y: 0, z: 0},
		modelPoint3{x: 0, y: 0, z: 1},
		sceneLighting{})
	if got.A < 126 || got.A > 129 {
		t.Fatalf("alpha = %d, want about 128", got.A)
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
	got := mat4TransformPoint(matrix, modelPoint3{x: 1})
	want := modelPoint3{x: 7}
	if math.Abs(got.x-want.x) > 0.0001 || math.Abs(got.y-want.y) > 0.0001 || math.Abs(got.z-want.z) > 0.0001 {
		t.Fatalf("animated point = (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)", got.x, got.y, got.z, want.x, want.y, want.z)
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
	got := mat4TransformPoint(matrix, modelPoint3{x: 1})
	want := modelPoint3{y: 1}
	if math.Abs(got.x-want.x) > 0.0001 || math.Abs(got.y-want.y) > 0.0001 || math.Abs(got.z-want.z) > 0.0001 {
		t.Fatalf("animated rotation point = (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)", got.x, got.y, got.z, want.x, want.y, want.z)
	}
}

func identityRSMMatrix() [9]float32 {
	return [9]float32{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}
}
