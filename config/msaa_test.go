package config

import (
	"path/filepath"
	"testing"
)

func TestMSAADefaultsAndSavedOverride(t *testing.T) {
	isolateUserConfig(t)
	cfg, err := LoadConfig(nil)
	if err != nil || cfg.Render.MSAA {
		t.Fatalf("MSAA should default off: config=%+v err=%v", cfg.Render, err)
	}
	cfg.ConfigPath = filepath.Join(t.TempDir(), "goro.ini")
	for _, enabled := range []bool{false, true} {
		if _, err := cfg.SaveUserSettings(UserSettings{MSAA: enabled}); err != nil {
			t.Fatal(err)
		}
		saved, err := LoadConfig([]string{"--config", cfg.ConfigPath})
		if err != nil || saved.Render.MSAA != enabled {
			t.Fatalf("saved MSAA=%t, want %t; error=%v", saved.Render.MSAA, enabled, err)
		}
		overridden, err := LoadConfig([]string{"--config", cfg.ConfigPath, "--msaa=false"})
		if err != nil || overridden.Render.MSAA {
			t.Fatalf("CLI MSAA=%t, want false; error=%v", overridden.Render.MSAA, err)
		}
	}
}
