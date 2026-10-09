package config

import (
	"path/filepath"
	"testing"
)

func TestBloomDefaultsAndSavedOverride(t *testing.T) {
	isolateUserConfig(t)
	cfg, err := LoadConfig(nil)
	if err != nil || cfg.Render.Bloom {
		t.Fatalf("Bloom should default off: config=%+v err=%v", cfg.Render, err)
	}
	cfg.ConfigPath = filepath.Join(t.TempDir(), "goro.ini")
	for _, enabled := range []bool{false, true} {
		if _, err := cfg.SaveUserSettings(UserSettings{Bloom: enabled}); err != nil {
			t.Fatal(err)
		}
		saved, err := LoadConfig([]string{"--config", cfg.ConfigPath})
		if err != nil || saved.Render.Bloom != enabled {
			t.Fatalf("saved Bloom=%t, want %t; error=%v", saved.Render.Bloom, enabled, err)
		}
		overridden, err := LoadConfig([]string{"--config", cfg.ConfigPath, "--bloom=false"})
		if err != nil || overridden.Render.Bloom {
			t.Fatalf("CLI Bloom=%t, want false; error=%v", overridden.Render.Bloom, err)
		}
	}
}
