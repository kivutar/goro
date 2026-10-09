package config

import (
	"path/filepath"
	"testing"
)

func TestSpriteSmoothingDefaultsAndSavedOverride(t *testing.T) {
	isolateUserConfig(t)
	cfg, err := LoadConfig(nil)
	if err != nil || !cfg.Render.SmoothSprites {
		t.Fatalf("sprite smoothing should default on: config=%+v err=%v", cfg.Render, err)
	}
	cfg.ConfigPath = filepath.Join(t.TempDir(), "goro.ini")
	for _, enabled := range []bool{false, true} {
		if _, err := cfg.SaveUserSettings(UserSettings{SmoothSprites: enabled}); err != nil {
			t.Fatal(err)
		}
		saved, err := LoadConfig([]string{"--config", cfg.ConfigPath})
		if err != nil || saved.Render.SmoothSprites != enabled {
			t.Fatalf("saved smoothing=%t, want %t; error=%v", saved.Render.SmoothSprites, enabled, err)
		}
		overridden, err := LoadConfig([]string{"--config", cfg.ConfigPath, "--smooth-sprites=false"})
		if err != nil || overridden.Render.SmoothSprites {
			t.Fatalf("CLI smoothing=%t, want false; error=%v", overridden.Render.SmoothSprites, err)
		}
	}
}
