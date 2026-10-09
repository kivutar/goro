package config

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAnisotropySettingsRoundTrip(t *testing.T) {
	isolateUserConfig(t)
	path := filepath.Join(t.TempDir(), "goro.ini")
	if err := os.WriteFile(path, []byte("[render]\nanisotropy = 4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	defaults, err := LoadConfig(nil)
	if err != nil || defaults.Render.Anisotropy != 8 {
		t.Fatalf("default anisotropy=%d error=%v", defaults.Render.Anisotropy, err)
	}
	cfg, err := LoadConfig([]string{"--config", path})
	if err != nil || cfg.Render.Anisotropy != 4 {
		t.Fatalf("INI anisotropy=%d error=%v", cfg.Render.Anisotropy, err)
	}
	for _, level := range []int{0, 2, 4, 8, 16} {
		if _, err := cfg.SaveUserSettings(UserSettings{Anisotropy: level}); err != nil {
			t.Fatal(err)
		}
		saved, err := LoadConfig([]string{"--config", path})
		if err != nil || saved.Render.Anisotropy != level {
			t.Fatalf("saved anisotropy=%d, want %d; error=%v", saved.Render.Anisotropy, level, err)
		}
		overridden, err := LoadConfig([]string{"--config", path, "--anisotropy=2"})
		if err != nil || overridden.Render.Anisotropy != 2 {
			t.Fatalf("CLI anisotropy=%d error=%v", overridden.Render.Anisotropy, err)
		}
	}
	for _, level := range []int{-1, 1, 3, 17, 65536} {
		if _, err := LoadConfig([]string{"--anisotropy", strconv.Itoa(level)}); err == nil {
			t.Fatalf("accepted invalid level %d", level)
		}
		if _, err := cfg.SaveUserSettings(UserSettings{Anisotropy: level}); err == nil {
			t.Fatalf("saved invalid level %d", level)
		}
	}
}
