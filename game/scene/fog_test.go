package scene_test

import (
	"image/color"
	"math"
	"testing"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
)

func TestSceneFogUsesViewDepth(t *testing.T) {
	projection := testutil.ProjectionYawZoom(1280, 720, 10.5, 20.5, 0, 0, scene.DefaultSceneCameraZoom)
	// Black-to-white fog exposes the original's byte-sized visibility directly.
	fog := scene.SceneFog{Enabled: true, Near: 32.5, Far: 92.5, Color: color.RGBA{255, 255, 255, 255}}
	_, forward, right, _ := projection.CameraBasis()
	center := scene.Point3{X: 10.5, Z: 20.5}
	verts := [4]scene.Point3{
		center,
		scene.Add3(center, scene.Mul3(right, 10)),
		scene.Add3(center, scene.Mul3(forward, 15)),
		scene.Add3(center, scene.Mul3(forward, -15)),
	}
	base := color.RGBA{A: 180}
	got := fog.MixVertexTints(projection, verts, [4]color.RGBA{base, base, base, base})
	for i, shade := range []uint8{128, 128, 192, 64} {
		want := color.RGBA{R: shade, G: shade, B: shade, A: base.A}
		if got[i] != want {
			t.Fatalf("vertex %d fog = %+v, want %+v", i, got[i], want)
		}
	}
	// Changing clip-space Z must not affect fog depth.
	projection.ViewProjection[14] += 20
	if changed := fog.MixVertexTints(projection, verts, [4]color.RGBA{base, base, base, base}); changed != got {
		t.Fatalf("clipping-plane change affected fog: %+v -> %+v", got, changed)
	}
}

func TestSceneFogMatchesOriginalLinearRamp(t *testing.T) {
	fog := scene.SceneFog{Enabled: true, Near: 10, Far: 20, Color: color.RGBA{255, 255, 255, 255}}
	base := color.RGBA{A: 180}
	for _, tt := range []struct {
		depth float64
		shade uint8
	}{
		{5, 0}, {10, 0}, {12.5, 64}, {15, 128}, {17.5, 192}, {20, 255}, {25, 255},
	} {
		want := color.RGBA{R: tt.shade, G: tt.shade, B: tt.shade, A: base.A}
		if got := fog.MixColor(base, tt.depth); got != want {
			t.Errorf("fog at depth %v = %+v, want %+v", tt.depth, got, want)
		}
	}
	for _, depth := range []float64{math.NaN(), math.Inf(-1), math.Inf(1)} {
		if got := fog.MixColor(base, depth); got != base {
			t.Errorf("invalid depth %v changed color to %+v", depth, got)
		}
	}
}
