package game

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/res"
)

func TestSceneFogFromMapUsesReferenceClientScaleAndFullStrength(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "fogparametertable.txt"), []byte("prontera#0.5#1.5#ffffff#0.3#"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}

	fog := sceneFogFromMap(manager, "prontera", config.FogConfig{Enabled: true})
	if !fog.enabled {
		t.Fatal("expected fog")
	}
	if fog.near != 120 || fog.far != 360 {
		t.Fatalf("unexpected scaled distances: near=%v far=%v", fog.near, fog.far)
	}
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 0)
	if got := projection.RenderCameraWithFog(fog).Fog.Strength; got != 1 {
		t.Fatalf("fog strength = %.2f, want full reference-client strength", got)
	}
}

func TestSceneFogFromMapUsesEinbrochWeatherTint(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "fogparametertable.txt"), []byte("einbroch.rsw#0.4#0.9#660000#0.5#"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}

	fog := sceneFogFromMap(manager, "einbroch.rsw", config.FogConfig{Enabled: true})
	if !fog.enabled {
		t.Fatal("expected fog")
	}
	if fog.color != (color.RGBA{R: 252, G: 171, B: 143, A: 255}) {
		t.Fatalf("einbroch fog color = %#v, want EF_CLOUD4 tint", fog.color)
	}
}

func TestSceneFogMixColorSmoothstepsToFogColor(t *testing.T) {
	fog := sceneFog{
		enabled: true,
		near:    10,
		far:     20,
		color:   color.RGBA{R: 200, G: 100, B: 50, A: 255},
	}
	base := color.RGBA{R: 100, G: 100, B: 100, A: 180}
	if got := fog.mixColor(base, 5); got != base {
		t.Fatalf("near color changed: %#v", got)
	}
	if got := fog.mixColor(base, 20); got != (color.RGBA{R: 200, G: 100, B: 50, A: 180}) {
		t.Fatalf("far color mismatch: %#v", got)
	}
	if got := fog.mixColor(base, 15); got != (color.RGBA{R: 150, G: 100, B: 75, A: 180}) {
		t.Fatalf("mid color mismatch: %#v", got)
	}
}
