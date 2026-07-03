package ui

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

func TestSettingsWindowEscapeCloses(t *testing.T) {
	var window SettingsWindow
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	window.OpenWindow(ctx)

	inputState.SetKey(input.KeyEscape, true)
	if !window.Update(ctx) {
		t.Fatal("settings window did not consume escape")
	}
	if window.IsOpen() {
		t.Fatal("settings window stayed open after escape")
	}
}
