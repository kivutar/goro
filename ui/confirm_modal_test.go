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
		{"delete_after_return", "Delete this message permanently?", false, 112},
		{"alert", "Disconnected from Server.", true, 156},
		{"delete_after_alert", "Delete this message permanently?", false, 112},
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

func TestConfirmModalUsesCompactHeightForOneLinePrompt(t *testing.T) {
	var modal ConfirmModal
	modal.Open(client.Context{ScreenW: 800, ScreenH: 600}, "Expel Party Member", "Expel Alice from the party?", nil, nil)

	want := ROWindowTitleHeight + smallPromptContentH + ROWindowFooterHeight
	if modal.height != want {
		t.Fatalf("modal height = %d, want %d", modal.height, want)
	}
}

func TestConfirmModalKeepsRoomForWrappedPrompt(t *testing.T) {
	var oneLine ConfirmModal
	oneLine.Open(client.Context{ScreenW: 800, ScreenH: 600}, "Confirm", "Expel Alice from the party?", nil, nil)

	var wrapped ConfirmModal
	wrapped.Open(client.Context{ScreenW: 800, ScreenH: 600}, "Confirm", "Would you like to invite Some Very Long Character Name to join your party?", nil, nil)

	if wrapped.height <= oneLine.height {
		t.Fatalf("wrapped height = %d, want greater than one-line height %d", wrapped.height, oneLine.height)
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
	lines := smallPromptLines("You have been forced to disconnect by the Game Master Team.", alertPromptMaxLines)
	if len(lines) != alertPromptMaxLines {
		t.Fatalf("line count = %d, want %d", len(lines), alertPromptMaxLines)
	}
	if lines[0] == "" || lines[1] == "" {
		t.Fatalf("message was not wrapped into visible rows: %#v", lines)
	}
}
