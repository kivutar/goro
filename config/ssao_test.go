package config

import (
	"path/filepath"
	"testing"
)

func TestSSAODefaultAndSavedOverride(t *testing.T) {
	isolateUserConfig(t)
	cfg, err := LoadConfig(nil)
	if err != nil || cfg.Render.SSAO {
		t.Fatalf("SSAO must default off: enabled=%t err=%v", cfg.Render.SSAO, err)
	}
	cfg.ConfigPath = filepath.Join(t.TempDir(), "goro.ini")
	if _, err := cfg.SaveUserSettings(UserSettings{SSAO: true, Bloom: true, MSAA: true}); err != nil {
		t.Fatal(err)
	}
	saved, err := LoadConfig([]string{"--config", cfg.ConfigPath})
	if err != nil || !saved.Render.SSAO || !saved.Render.Bloom || !saved.Render.MSAA {
		t.Fatalf("SSAO/Bloom/MSAA did not round trip: render=%+v err=%v", saved.Render, err)
	}
	overridden, err := LoadConfig([]string{"--config", cfg.ConfigPath, "--ssao=false"})
	if err != nil || overridden.Render.SSAO {
		t.Fatalf("CLI override ignored: enabled=%t err=%v", overridden.Render.SSAO, err)
	}
}
