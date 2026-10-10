package config

import (
	"path/filepath"
	"testing"
)

func TestTextureUpscalingDefaultSavedSettingAndCLIOverride(t *testing.T) {
	isolateUserConfig(t)
	cfg, err := LoadConfig(nil)
	if err != nil || cfg.Render.TextureUpscaling {
		t.Fatalf("upscaling must default off: render=%+v err=%v", cfg.Render, err)
	}
	cfg.ConfigPath = filepath.Join(t.TempDir(), "goro.ini")
	for _, enabled := range []bool{true, false} {
		if _, err := cfg.SaveUserSettings(UserSettings{TextureUpscaling: enabled, Anisotropy: 8}); err != nil {
			t.Fatal(err)
		}
		saved, err := LoadConfig([]string{"--config", cfg.ConfigPath})
		if err != nil || saved.Render.TextureUpscaling != enabled || saved.Render.Anisotropy != 8 {
			t.Fatalf("upscaling did not round trip: render=%+v err=%v", saved.Render, err)
		}
	}
	overridden, err := LoadConfig([]string{"--config", cfg.ConfigPath, "--texture-upscaling"})
	if err != nil || !overridden.Render.TextureUpscaling {
		t.Fatalf("CLI override ignored: render=%+v err=%v", overridden.Render, err)
	}
}
