package weather

import (
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
)

func TestSceneFogFromMapUsesOriginalViewPlanes(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "fogparametertable.txt"), []byte("tur_dun02#0.2#0.4#4a6976#0.3#"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}

	fog := FogFromMap(manager, "tur_dun02", config.FogConfig{Enabled: true})
	if !fog.Enabled {
		t.Fatal("expected fog")
	}
	if math.Abs(fog.Near-61.6) > 1e-9 || math.Abs(fog.Far-121.2) > 1e-9 {
		t.Fatalf("unexpected scaled distances: near=%v far=%v", fog.Near, fog.Far)
	}
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	if got := projection.RenderCameraWithFog(fog).Fog.Strength; got != 1 {
		t.Fatalf("fog strength = %.2f, want full reference-client strength", got)
	}
	if disabled := FogFromMap(manager, "tur_dun02", config.FogConfig{Enabled: false}); disabled.Enabled {
		t.Fatal("disabled fog still enabled on a foggy map")
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

	fog := FogFromMap(manager, "einbroch.rsw", config.FogConfig{Enabled: true})
	if !fog.Enabled {
		t.Fatal("expected fog")
	}
	if fog.Color != (color.RGBA{R: 252, G: 171, B: 143, A: 255}) {
		t.Fatalf("einbroch fog color = %#v, want EF_CLOUD4 tint", fog.Color)
	}
}

func TestSceneFogMixColorPreservesAlpha(t *testing.T) {
	fog := scene.SceneFog{
		Enabled: true,
		Near:    10,
		Far:     20,
		Color:   color.RGBA{R: 200, G: 100, B: 50, A: 255},
	}
	base := color.RGBA{R: 100, G: 100, B: 100, A: 180}
	if got := fog.MixColor(base, 5); got != base {
		t.Fatalf("near color changed: %#v", got)
	}
	if got := fog.MixColor(base, 20); got != (color.RGBA{R: 200, G: 100, B: 50, A: 180}) {
		t.Fatalf("far color mismatch: %#v", got)
	}
	if got := fog.MixColor(base, 15); got != (color.RGBA{R: 150, G: 100, B: 74, A: 180}) {
		t.Fatalf("mid color mismatch: %#v", got)
	}
}
