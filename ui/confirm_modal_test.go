package ui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

func TestConfirmModalFooterAfterReuse(t *testing.T) {
	manager := NewManager()
	ctx := client.Context{ScreenW: 800, ScreenH: 600, UIManager: manager}
	var modal ConfirmModal
	for _, tt := range []struct {
		name    string
		message string
		alert   bool
		height  int
	}{
		{"return", "Return this mail and its attachments to the sender?", false, 126},
		{"delete_after_return", "Delete this message permanently?", false, 126},
		{"alert", "Disconnected from Server.", true, 126},
		{"long_alert", "First line\nSecond line\nThird line", true, 140},
		{"login_failed_after_long_alert", "Incorrect Password.", true, 126},
		{"delete_after_alert", "Delete this message permanently?", false, 126},
		{"return_after_delete", "Return this mail and its attachments to the sender?", false, 126},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.alert {
				modal.OpenAlert(ctx, tt.name, tt.message, nil)
			} else {
				modal.Open(ctx, tt.name, tt.message, nil, nil)
			}
			defer modal.Close(ctx)
			wc := widget.NewContext()
			manager.root.Layout(wc, geometry.Tight(geometry.Sz(800, 600)))
			manager.root.Draw(wc, &uitest.MockCanvas{})

			if modal.height != tt.height {
				t.Errorf("reused window height = %d, want %d", modal.height, tt.height)
			}
			children := modal.content.Children()
			footer := children[len(children)-1].(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
			want := geometry.NewRect(float32(modal.x), float32(modal.y+modal.height-ROWindowFooterHeight), float32(modal.width), ROWindowFooterHeight)
			if footer != want {
				t.Errorf("footer = %v, want bottom band %v", footer, want)
			}
		})
	}
}

func TestConfirmModalOpenAlertEscapeConfirms(t *testing.T) {
	var modal ConfirmModal
	inputState := input.NewState()
	confirmed := false
	ctx := client.Context{
		Input:   inputState,
		ScreenW: 800,
		ScreenH: 600,
	}
	modal.OpenAlert(ctx, "Disconnected", "Disconnected from Server.", func() {
		confirmed = true
	})
	inputState.EndFrame()

	inputState.SetKey(input.KeyEscape, true)
	if !modal.Update(ctx) {
		t.Fatal("alert did not consume escape")
	}
	if modal.IsOpen() {
		t.Fatal("alert remained open after escape")
	}
	if !confirmed {
		t.Fatal("escape did not confirm alert")
	}
}

func TestConfirmModalIgnoresOpeningFrameKeys(t *testing.T) {
	for _, alert := range []bool{false, true} {
		for _, key := range []input.Key{input.KeyEnter, input.KeyEscape} {
			var modal ConfirmModal
			state := input.NewState()
			ctx := client.Context{Input: state, ScreenW: 800, ScreenH: 600}
			called := false
			callback := func() { called = true }
			if alert {
				modal.OpenAlert(ctx, "Alert", "Connection failed.", callback)
			} else {
				modal.Open(ctx, "Confirm", "Are you sure?", callback, callback)
			}
			state.SetKey(key, true)
			modal.Update(ctx)
			if called || !modal.IsOpen() {
				t.Fatalf("opening key %v dismissed modal (alert=%t)", key, alert)
			}
			state.EndFrame()
			modal.Update(ctx)
			if called || !modal.IsOpen() {
				t.Fatalf("held key %v dismissed modal (alert=%t)", key, alert)
			}
			state.EndFrame()
			state.SetKey(key, false)
			state.SetKey(key, true)
			modal.Update(ctx)
			if !called || modal.IsOpen() {
				t.Fatalf("new key %v did not dismiss modal (alert=%t)", key, alert)
			}
		}
	}
}

func TestConfirmModalReservesTwoLinesForShortPrompt(t *testing.T) {
	for _, message := range []string{"", "Expel Alice from the party?", "First line\nSecond line"} {
		t.Run(message, func(t *testing.T) {
			var modal ConfirmModal
			modal.Open(client.Context{ScreenW: 800, ScreenH: 600}, "Confirm", message, nil, nil)

			if got := modal.messageMaxLines(); got != 2 {
				t.Fatalf("reserved message lines = %d, want 2", got)
			}
			want := ROWindowTitleHeight + smallPromptContentH + smallPromptLineH + ROWindowFooterHeight
			if modal.height != want {
				t.Fatalf("modal height = %d, want %d", modal.height, want)
			}
		})
	}
}

func TestConfirmModalKeepsRoomForWrappedPrompt(t *testing.T) {
	var oneLine ConfirmModal
	oneLine.Open(client.Context{ScreenW: 800, ScreenH: 600}, "Confirm", "Expel Alice from the party?", nil, nil)

	var wrapped ConfirmModal
	wrapped.Open(client.Context{ScreenW: 800, ScreenH: 600}, "Confirm", "Would you like to invite Some Very Long Character Name to join your party?", nil, nil)

	if wrapped.messageMaxLines() != 2 || wrapped.height != oneLine.height {
		t.Fatalf("wrapped prompt = %d lines, height %d; want 2 lines with the same height %d as a one-line prompt", wrapped.messageMaxLines(), wrapped.height, oneLine.height)
	}
}

func TestConfirmModalAlertUsesSameSizing(t *testing.T) {
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	for _, message := range []string{
		"",
		"Incorrect Password.",
		"First line\nSecond line",
		"Your connection is terminated because your IP doesn't match the authorized IP from the account server.",
	} {
		t.Run(message, func(t *testing.T) {
			var confirmation, alert ConfirmModal
			confirmation.Open(ctx, "Confirm", message, nil, nil)
			alert.OpenAlert(ctx, "Alert", message, nil)

			if got, want := alert.messageMaxLines(), confirmation.messageMaxLines(); got != want {
				t.Errorf("alert reserved lines = %d, want %d like confirmation", got, want)
			}
			if alert.height != confirmation.height {
				t.Errorf("alert height = %d, want %d like confirmation", alert.height, confirmation.height)
			}
		})
	}
}

func TestConfirmModalShowsCompleteStarPlaceWarning(t *testing.T) {
	const message = "You cannot change a map's designation once it is designated. Are you sure that you want to designate this map?"
	var modal ConfirmModal
	modal.Open(client.Context{ScreenW: 800, ScreenH: 600}, "Feeling the Sun, Moon and Stars", message, nil, nil)

	if got := modal.messageMaxLines(); got != 3 {
		t.Fatalf("warning lines = %d, want 3", got)
	}
	wantHeight := ROWindowTitleHeight + smallPromptContentH + 2*smallPromptLineH + ROWindowFooterHeight
	if modal.height != wantHeight {
		t.Fatalf("warning height = %d, want %d", modal.height, wantHeight)
	}
}

func TestSmallPromptLinesWrapLongDisconnectMessage(t *testing.T) {
	lines := smallPromptLines("You have been forced to disconnect by the Game Master Team.", smallPromptDefaultLines)
	if len(lines) != smallPromptDefaultLines {
		t.Fatalf("line count = %d, want %d", len(lines), smallPromptDefaultLines)
	}
	if lines[0] == "" || lines[1] == "" {
		t.Fatalf("message was not wrapped into visible rows: %#v", lines)
	}
}
