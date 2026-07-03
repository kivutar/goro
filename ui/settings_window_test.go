package ui

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

func TestSettingsWindowEscapeCloses(t *testing.T) {
	var window SettingsWindow
	window.open = true
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}

	inputState.SetKey(input.KeyEscape, true)
	if !window.Update(ctx) {
		t.Fatal("settings window did not consume escape")
	}
	if window.open {
		t.Fatal("settings window stayed open after escape")
	}
}

func TestSettingsBoolText(t *testing.T) {
	if settingsBoolText(true) != "on" || settingsBoolText(false) != "off" {
		t.Fatalf("unexpected settings bool text")
	}
}
