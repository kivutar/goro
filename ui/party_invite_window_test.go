package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
)

func TestPartyInvitePromptPresentationAndLimit(t *testing.T) {
	var window PartyInviteWindow
	// World updates closed windows before their first opening too.
	window.Update(Context{})
	window.Open(Context{})
	if window.width != partyInviteW || window.height != 140 {
		t.Fatalf("size = %dx%d, want 286x140", window.width, window.height)
	}
	if window.title != "Party Invitation" || window.label != "Player Name" || window.placeholder != "Player Name" {
		t.Fatal("party prompt labels changed")
	}
	if !window.inputField.IsFocused() {
		t.Fatal("party prompt did not focus its input")
	}
	wc := widget.NewContext()
	window.content.Layout(wc, geometry.Loose(geometry.Sz(800, 600)))
	if got := window.content.(interface{ Bounds() geometry.Rect }).Bounds().Width(); got != partyInviteW {
		t.Fatalf("content width = %g, want %d", got, partyInviteW)
	}
	for range 24 {
		window.inputField.Event(wc, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'a', event.ModNone))
	}
	if window.value != strings.Repeat("a", 23) {
		t.Fatalf("name = %q, want 23 characters", window.value)
	}

	var generic TextPromptWindow
	generic.Open(Context{}, "Title", "Label", "Placeholder", 79)
	if generic.width != textPromptW {
		t.Fatalf("generic prompt width = %d, want %d", generic.width, textPromptW)
	}
}

func TestPartyInvitePromptActionsAndReopen(t *testing.T) {
	for _, action := range []string{"OK", "Cancel", "Close", "Escape", "Enter"} {
		t.Run(action, func(t *testing.T) {
			ctx, manager, app := newWindowInstanceTest()
			var window PartyInviteWindow
			window.Open(ctx)
			for _, r := range " Alice " {
				window.inputField.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, event.ModNone))
			}
			app.Frame()
			app.Window().DrawTo(&uitest.MockCanvas{})
			switch action {
			case "OK", "Cancel", "Close":
				buttons := collectEscapeMenuButtons(window.content)
				if len(buttons) != 3 {
					t.Fatalf("buttons = %d, want close, OK and Cancel", len(buttons))
				}
				index := 0
				if action == "OK" {
					index = 1
				} else if action == "Cancel" {
					index = 2
				}
				point := buttons[index].ScreenBounds().Center()
				app.Window().HandleEvent(uitest.Click(point.X, point.Y))
				app.Window().HandleEvent(uitest.Release(point.X, point.Y))
			case "Escape":
				ctx.Input.SetKey(input.KeyEscape, true)
				window.Update(ctx)
			case "Enter":
				window.inputField.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
			}
			if window.IsOpen() || len(manager.overlays) != 0 {
				t.Fatal("party prompt stayed open or published")
			}
			want := ""
			if action == "OK" || action == "Enter" {
				want = "Alice"
			}
			if got := window.PopAction(); got != want {
				t.Fatalf("action = %q, want %q", got, want)
			}
			if got := window.PopAction(); got != "" {
				t.Fatalf("action repeated: %q", got)
			}
			window.Open(ctx)
			if window.value != "" || window.inputField.Text() != "" || !window.inputField.IsFocused() || window.width != partyInviteW {
				t.Fatal("reopening did not restore an empty focused party prompt")
			}
			window.value = "  "
			window.submit(ctx)
			if !window.IsOpen() || window.PopAction() != "" {
				t.Fatal("blank name submitted")
			}
		})
	}
}

func TestPartyInvitePromptRebindTargetsCurrentWindow(t *testing.T) {
	ctx := Context{UIManager: NewManager(), ScreenW: 800, ScreenH: 600}
	var original PartyInviteWindow
	original.Open(ctx)
	original.inputField.SetText("Alice")
	original.value = "Alice"
	original.setPosition(ctx, 45, 67)
	next := original // World transfers persistent windows when changing maps.
	next.Rebind(ctx)
	if next.inputField == original.inputField || next.inputField.Text() != "Alice" || !next.inputField.IsFocused() {
		t.Fatal("rebind did not retain a focused name in a new input widget")
	}
	if next.x != 45 || next.y != 67 || next.width != partyInviteW {
		t.Fatal("rebind changed the party prompt frame")
	}
	next.inputField.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'x', event.ModNone))
	next.inputField.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if got := next.PopAction(); got != "Alicex" {
		t.Fatalf("rebound action = %q, want Alicex", got)
	}
	if next.IsOpen() || !original.IsOpen() || original.PopAction() != "" || original.value != "Alice" {
		t.Fatal("rebound callbacks modified the previous world window")
	}
	if len(ctx.UIManager.(*Manager).overlays) != 0 {
		t.Fatal("rebound window left a stale overlay")
	}
}
