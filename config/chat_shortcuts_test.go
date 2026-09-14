package config

import (
	"os"
	"strings"
	"testing"
)

func TestChatShortcutsDefaultsAndSlots(t *testing.T) {
	cfg := defaultConfig()
	if cfg.ChatShortcuts[0] != "/!" || cfg.ChatShortcuts[9] != "/..." {
		t.Fatalf("default shortcuts = %q", cfg.ChatShortcuts)
	}
	if err := applyINI(&cfg, strings.NewReader("[chat_shortcuts]\n1 = /sit\n0 =\n")); err != nil {
		t.Fatal(err)
	}
	if cfg.ChatShortcuts[0] != "/sit" || cfg.ChatShortcuts[9] != "" || cfg.ChatShortcuts[1] != "/?" {
		t.Fatalf("loaded shortcuts = %q", cfg.ChatShortcuts)
	}
	for _, key := range []string{"-1", "10", "abc"} {
		if err := applyConfigValue(&cfg, "chatshortcuts", key, "/sit"); err == nil {
			t.Fatalf("accepted invalid slot %q", key)
		}
	}
}

func TestChatShortcutsSavePreservesOtherSettings(t *testing.T) {
	isolateUserConfig(t)
	t.Setenv("APPDATA", t.TempDir())
	if _, err := SaveLoginID("Tester", true); err != nil {
		t.Fatal(err)
	}
	commands := defaultChatShortcuts()
	commands[0] = `hello #;="world"`
	commands[3] = "/w Alice hello"
	commands[9] = ""
	path, err := SaveChatShortcuts(commands)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SaveUserSettings(UserSettings{VSync: true, BGMVolume: 0.5, SFXVolume: 0.5}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadChatShortcuts(defaultChatShortcuts())
	if err != nil || got != commands {
		t.Fatalf("reload = %q, %v; want %q", got, err, commands)
	}
	cfg := defaultConfig()
	if err := applyINIFile(&cfg, path, true); err != nil {
		t.Fatal(err)
	}
	if cfg.Login.SavedUsername != "Tester" || !cfg.Render.VSync {
		t.Fatalf("other settings were lost: username=%q vsync=%v", cfg.Login.SavedUsername, cfg.Render.VSync)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	commands[0] = "hello\n[login]\nusername = injected"
	if _, err := SaveChatShortcuts(commands); err == nil {
		t.Fatal("accepted multiline shortcut")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("invalid save changed config")
	}
}
