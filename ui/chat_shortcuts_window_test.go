package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func shortcutWindowTestContext(t *testing.T) Context {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	return Context{Input: input.NewState(), ScreenW: 800, ScreenH: 600, Config: config.Config{ChatShortcuts: config.ChatShortcuts{"/!"}}}
}

func TestChatShortcutEditorSavesAndClears(t *testing.T) {
	ctx := shortcutWindowTestContext(t)
	w := &ChatShortcutsWindow{}
	w.Toggle(ctx, nil, nil)
	tree := w.content
	field := w.fields[9]
	field.SetFocused(true)
	for _, r := range "/sit" {
		field.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, event.ModNone))
	}
	if w.Command(ctx, 9) != "/sit" {
		t.Fatal("typing did not update slot 0")
	}
	w.Update(ctx)
	if tree != w.content {
		t.Fatal("editor rebuilt on update")
	}
	fresh := &ChatShortcutsWindow{}
	if fresh.Command(ctx, 9) != "/sit" {
		t.Fatal("fresh world did not read saved bindings")
	}
	for range 4 {
		field.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyBackspace, 0, event.ModNone))
	}
	w.Close()
	fresh = &ChatShortcutsWindow{}
	if fresh.Command(ctx, 9) != "" {
		t.Fatal("cleared binding was not saved")
	}
	if fresh.Command(ctx, -1) != "" || fresh.Command(ctx, 10) != "" {
		t.Fatal("invalid slot was not ignored")
	}
}

func TestChatShortcutEmotePickerEditsSelectedSlot(t *testing.T) {
	ctx := shortcutWindowTestContext(t)
	w := &ChatShortcutsWindow{}
	console := &ChatConsole{input: "draft"}
	w.Toggle(ctx, console, nil)
	w.fields[6].SetFocused(true)
	w.UpdateKeyboardInput(ctx)
	emotes := &EmoteWindow{OnSelect: w.SelectEmotion}
	emote := db.Emotion{Command: "ho"}
	emotes.selectEmotion(ctx, console, emote)
	emotes.playEmotion(ctx, console, emote)
	if w.Command(ctx, 6) != "/ho" || w.fields[6].Text() != "/ho" {
		t.Fatal("picker did not edit selected slot")
	}
	if console.input != "draft" || console.active || len(console.messages) != 0 {
		t.Fatal("picker submitted to console instead of binding")
	}
	w.Close()
	emotes.selectEmotion(ctx, console, emote)
	if console.input != "/ho" || !console.active {
		t.Fatal("closed editor still intercepted picker")
	}
}

func TestChatShortcutEditorRebindsCallbacksAfterMapChange(t *testing.T) {
	ctx := shortcutWindowTestContext(t)
	original := &ChatShortcutsWindow{}
	original.Toggle(ctx, nil, nil)
	next := *original
	next.Rebind(ctx, nil, nil)
	next.fields[9].SetFocused(true)
	next.fields[9].Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'x', event.ModNone))
	if next.Command(ctx, 9) != "x" || original.Command(ctx, 9) != "" {
		t.Fatal("rebound editor still updates old world")
	}
	next.setCommand(9, "hello\r\nworld\x00")
	if next.fields[9].Text() != "helloworld" || next.Command(ctx, 9) != "helloworld" {
		t.Fatal("editor displays a different command than it saves")
	}
}

func TestChatShortcutViewAndEscape(t *testing.T) {
	ctx := shortcutWindowTestContext(t)
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	ctx.UIManager = manager
	w := &ChatShortcutsWindow{}
	views := 0
	w.Toggle(ctx, nil, func() { views++ })
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	point := geometry.Pt(float32(w.x+chatShortcutsW-35), float32(w.y+chatShortcutsH-ROWindowFooterHeight/2))
	app.Window().HandleEvent(uitest.Click(point.X, point.Y))
	app.Window().HandleEvent(uitest.Release(point.X, point.Y))
	if views != 1 {
		t.Fatalf("View callback count = %d", views)
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	if !w.Update(ctx) || w.IsOpen() {
		t.Fatal("Escape did not close editor")
	}
}

func TestChatShortcutEditorOnlyBlocksFocusedInput(t *testing.T) {
	ctx := shortcutWindowTestContext(t)
	w := &ChatShortcutsWindow{}
	w.Toggle(ctx, nil, nil)
	ctx.Input.SetKey(input.KeyEnter, true)
	if w.KeyboardShortcutsBlocked() || w.UpdateKeyboardInput(ctx) {
		t.Fatal("unfocused editor blocked console")
	}
	w.fields[0].SetFocused(true)
	if !w.KeyboardShortcutsBlocked() || !w.UpdateKeyboardInput(ctx) {
		t.Fatal("focused editor leaked Enter to console")
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	if !w.UpdateKeyboardInput(ctx) || w.IsOpen() || w.fields[0].IsFocused() {
		t.Fatal("Escape did not close focused editor and release focus")
	}
}

func TestConsoleSendTextPreservesDraftAndSelection(t *testing.T) {
	for _, command := range []string{"/ns", "/nc", "/!", "/sit", "hello", "/w Alice hello", ""} {
		t.Run(command, func(t *testing.T) {
			console := &ChatConsole{input: "unfinished draft", active: true, history: []string{"previous"}}
			field := console.inputWidget()
			field.SetFocused(true)
			field.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyLeft, 0, event.ModShift))
			start, end := field.Selection()
			cursor := field.CursorPosition()
			console.SendText(Context{Session: session.New()}, command)
			nextStart, nextEnd := field.Selection()
			if console.input != "unfinished draft" || field.Text() != console.input || !field.IsFocused() || !console.active || field.CursorPosition() != cursor || start != nextStart || end != nextEnd {
				t.Fatal("shortcut changed console draft, selection, or focus")
			}
			if len(console.history) != 1 || console.history[0] != "previous" {
				t.Fatal("shortcut changed console history")
			}
		})
	}
}

func TestShortcutBarDoesNotConsumeAltNumber(t *testing.T) {
	ctx := shortcutWindowTestContext(t)
	ctx.Input.SetKey(input.KeyAlt, true)
	ctx.Input.SetKey(input.Key1, true)
	ctx.Input.SetMousePosition(0, 500)
	bar := &ShortcutBar{}
	if bar.Update(ctx, nil) {
		t.Fatal("Alt+1 triggered a skill-bar slot")
	}
}
