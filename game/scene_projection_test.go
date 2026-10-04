package game

import (
	"math"
	"testing"

	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func newSceneProjectionForTarget(width, height int, targetX, targetY, targetZ float64) sceneProjection {
	return newSceneProjectionForTargetYaw(width, height, targetX, targetY, targetZ, sceneCameraYaw())
}

func newSceneProjectionForTargetYaw(width, height int, targetX, targetY, targetZ, yaw float64) sceneProjection {
	return newSceneProjectionForTargetYawZoom(width, height, targetX, targetY, targetZ, yaw, sceneCameraZoom())
}

func newSceneProjectionForTargetYawZoom(width, height int, targetX, targetY, targetZ, yaw, zoom float64) sceneProjection {
	return newSceneProjectionForTargetYawPitchZoom(width, height, targetX, targetY, targetZ, yaw, sceneCameraPitch(), zoom)
}

func TestBilinearHeight(t *testing.T) {
	heights := [4]float32{0, 10, 20, 30}
	if got := bilinearHeight(heights, 0.5, 0.5); got != 15 {
		t.Fatalf("height = %v, want 15", got)
	}
}

func TestTerrainHeightPrefersGAT(t *testing.T) {
	world := &worldstate.World{
		GAT: &res.GAT{
			Width:  1,
			Height: 1,
			Cells:  []res.GATCell{{Heights: [4]float32{10, 20, 30, 40}}},
		},
		GND: &res.GND{
			Width:  1,
			Height: 1,
			Cells:  []res.GNDCell{{Heights: [4]float32{1, 1, 1, 1}}},
		},
	}
	if got := terrainHeightAt(world, 0, 0); got != 25 {
		t.Fatalf("terrain height = %v, want GAT center height 25", got)
	}
}

func TestCameraProjectionCentersPlayerCell(t *testing.T) {
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 5)
	point := projection.Project(10.5, 20.5, 5)
	if math.Abs(float64(point.x)-400) > 0.01 || math.Abs(float64(point.y)-300) > 0.01 {
		t.Fatalf("projected point = %.1f, %.1f, want 400, 300", point.x, point.y)
	}
}

func TestSceneCameraDefaultZoomUsesGameplayScale(t *testing.T) {
	if got := sceneCameraZoom() * 0.5; got != 62.5 {
		t.Fatalf("default camera distance = %.1f, want reference client default 62.5", got)
	}
}

func TestCameraProjectionUsesRequestedPitch(t *testing.T) {
	low := newSceneProjectionForTargetYawPitchZoom(800, 600, 10.5, 20.5, 5, 0, defaultCameraMinPitch, 125)
	high := newSceneProjectionForTargetYawPitchZoom(800, 600, 10.5, 20.5, 5, 0, defaultCameraMaxPitch, 125)
	lowEye, _, _, _ := low.cameraBasis()
	highEye, _, _, _ := high.cameraBasis()

	if low.cameraPitch != defaultCameraMinPitch || high.cameraPitch != defaultCameraMaxPitch {
		t.Fatalf("projection pitches = %.1f and %.1f, want %.1f and %.1f", low.cameraPitch, high.cameraPitch, defaultCameraMinPitch, defaultCameraMaxPitch)
	}
	if highEye.y <= lowEye.y {
		t.Fatalf("high-pitch eye y = %.2f, want above low-pitch eye y %.2f", highEye.y, lowEye.y)
	}
}

func TestCameraProjectionMapsPositiveXToScreenRight(t *testing.T) {
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 5)
	center := projection.Project(10.5, 20.5, 5)
	right := projection.Project(11.5, 20.5, 5)
	if right.x <= center.x {
		t.Fatalf("positive world X projected to %.1f, center %.1f; want screen right", right.x, center.x)
	}
}

func TestCameraProjectionMovesHigherWorldPointUpScreen(t *testing.T) {
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 5)
	base := projection.Project(10.5, 20.5, 5)
	higher := projection.Project(10.5, 20.5, 15)
	if higher.y >= base.y {
		t.Fatalf("higher point y = %.1f, base y = %.1f, want higher point above base", higher.y, base.y)
	}
}

func TestBillboardBasisStaysScreenAlignedOffCenter(t *testing.T) {
	projection := newSceneProjectionForTargetYawZoom(800, 600, 10.5, 20.5, 5, 0, 150)
	worldX, worldY, worldZ := 18.5, 20.5, 5.0
	right, up, unitsPerPixel, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		t.Fatal("missing billboard basis")
	}

	center := modelPoint3{x: worldX, y: worldZ, z: worldY}
	project := func(p modelPoint3) screenPoint {
		return projection.Project(p.x, p.z, p.y)
	}
	left := project(add3(center, mul3(right, -32*unitsPerPixel)))
	rightPoint := project(add3(center, mul3(right, 32*unitsPerPixel)))
	top := project(add3(center, mul3(up, 64*unitsPerPixel)))
	bottom := project(add3(center, mul3(up, -64*unitsPerPixel)))
	centerPoint := project(center)

	if math.Abs(float64(left.y-rightPoint.y)) > 0.25 {
		t.Fatalf("billboard horizontal edge y mismatch: left %.3f right %.3f", left.y, rightPoint.y)
	}
	if math.Abs(float64(top.x-centerPoint.x)) > 0.25 || math.Abs(float64(bottom.x-centerPoint.x)) > 0.25 {
		t.Fatalf("billboard vertical edge x mismatch: top %.3f center %.3f bottom %.3f", top.x, centerPoint.x, bottom.x)
	}
	if got := math.Abs(float64(rightPoint.x - left.x)); math.Abs(got-64) > 0.5 {
		t.Fatalf("billboard projected width = %.3f, want 64", got)
	}
	if got := math.Abs(float64(bottom.y - top.y)); math.Abs(got-128) > 0.5 {
		t.Fatalf("billboard projected height = %.3f, want 128", got)
	}
}

func TestCameraDepthOrdersFarBeforeNear(t *testing.T) {
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 5)
	near := projection.Depth(10.5, 20.5, 5)
	far := projection.Depth(10.5, 30.5, 5)
	if far <= near {
		t.Fatalf("far depth %.2f must be greater than near depth %.2f for painter sorting", far, near)
	}
}
