package game

import (
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func chatShortcutTestContext(t *testing.T) client.Context {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	return client.Context{Input: input.NewState(), Session: session.New(), ScreenW: 800, ScreenH: 600}
}

func TestChatShortcutWindowToggleAndBlocking(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	m := &WorldMode{}
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	ctx.Input.SetKeyCode(gpucontext.KeyM, true)
	if !m.SuppressShortcutText(ctx, gpucontext.KeyM) || !m.chatShortcutFromInput(ctx) || !m.ui.chatShortcuts.IsOpen() {
		t.Fatal("Alt+M did not open shortcut list")
	}
	if !m.ui.keyboardInputBlocked(ctx) {
		t.Fatal("shortcut editor did not block game hotkeys")
	}
	if !m.SuppressShortcutText(ctx, gpucontext.KeyM) || m.chatShortcutFromInput(ctx) || !m.ui.chatShortcuts.IsOpen() {
		t.Fatal("held Alt+M retriggered or leaked text")
	}
	ctx.Input.SetKeyCode(gpucontext.KeyM, false)
	ctx.Input.SetKeyCode(gpucontext.KeyM, true)
	if !m.chatShortcutFromInput(ctx) || m.ui.chatShortcuts.IsOpen() {
		t.Fatal("Alt+M did not close shortcut list")
	}
}

func TestChatShortcutTextOnlyInterceptsAvailableBindings(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	m := &WorldMode{}
	ctx.Config.ChatShortcuts[0] = "/ns"
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	ctx.Input.SetKeyCode(gpucontext.Key1, true)
	if !m.SuppressShortcutText(ctx, gpucontext.Key1) {
		t.Fatal("bound Alt+1 text leaked")
	}
	if !ctx.Input.KeyCodeJustPressed(gpucontext.Key1) || ctx.Session.NoShift {
		t.Fatal("text filter consumed or executed a shortcut")
	}
	if m.SuppressShortcutText(ctx, gpucontext.Key2) {
		t.Fatal("unbound Option text was swallowed")
	}
	for _, key := range []gpucontext.Key{gpucontext.KeyL, gpucontext.KeyG} {
		if !m.SuppressShortcutText(ctx, key) {
			t.Fatalf("Alt+%v text leaked", key)
		}
	}
	for _, modifier := range []gpucontext.Key{gpucontext.KeyLeftControl, gpucontext.KeyRightAlt, gpucontext.KeyLeftShift, gpucontext.KeyLeftSuper, gpucontext.KeyRightSuper} {
		ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
		ctx.Input.SetKeyCode(modifier, true)
		if m.SuppressShortcutText(ctx, gpucontext.Key1) {
			t.Fatalf("text with %v was swallowed", modifier)
		}
		ctx.Input.SetKeyCode(modifier, false)
	}
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	m.ui.settingsWindow.OpenWindow(ctx)
	if m.SuppressShortcutText(ctx, gpucontext.Key1) {
		t.Fatal("shortcut text was swallowed while a form blocked shortcuts")
	}
	if m.SuppressShortcutText(ctx, gpucontext.KeyM) {
		t.Fatal("Alt+M was swallowed while a form blocked shortcuts")
	}
	login := &Manager{mode: NewLoginMode()}
	if login.SuppressShortcutText(ctx, gpucontext.KeyM) {
		t.Fatal("login intercepted a world shortcut")
	}
}

func TestConsoleSubmitsWhileShortcutEditorIsOpen(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.World = worldstate.New()
	ctx.Network = network.NewClient(20080910, false)
	t.Cleanup(func() { ctx.Network.Close() })
	manager := &worldModeTestUIManager{}
	ctx.UIManager = manager
	mode := NewWorldMode()
	mode.ui.console.Publish(ctx)
	var field *textfield.Widget
	var find func(widget.Widget)
	find = func(w widget.Widget) {
		if f, ok := w.(*textfield.Widget); ok {
			field = f
		}
		for _, child := range w.Children() {
			find(child)
		}
	}
	for _, root := range manager.overlays {
		find(root)
	}
	if field == nil {
		t.Fatal("no console field")
	}
	mode.ui.chatShortcuts.Toggle(ctx, &mode.ui.console, nil)
	mode.ui.chatShortcuts.SetAutoPosition(100, 100)
	ctx.Input.SetMousePosition(120, 150) // Hovering the editor must not swallow console submissions.
	field.SetText("/ns")
	field.SetFocused(true)
	field.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	ctx.Input.SetKey(input.KeyEnter, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !ctx.Session.NoShift || !mode.ui.chatShortcuts.IsOpen() {
		t.Fatal("console command did not execute with shortcut editor open")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKey(input.KeyEnter, false)
	field.SetFocused(true)
	ctx.Input.SetKey(input.KeyEscape, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.console.Active() || !mode.ui.chatShortcuts.IsOpen() {
		t.Fatal("Escape did not leave console input before closing the editor")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKey(input.KeyEscape, false)
	ctx.Input.SetKey(input.KeyEscape, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.chatShortcuts.IsOpen() || mode.ui.escapeMenu.IsOpen() {
		t.Fatal("Escape did not close the editor")
	}
	if !ctx.Session.NoShift {
		t.Fatal("closing shortcut editor resubmitted a console command")
	}
}

func TestChatShortcutAllPhysicalSlotsSendOnce(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	conn, server := newBotTestConnection(t, 20080910)
	ctx.Network = conn
	ctx.Session.Selected.Name = "Tester"
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	ctx.Config.ChatShortcuts = config.ChatShortcuts{"hello", "@where", "%party", "$guild", "/w Alice hello", "/!", "/ns", "/nc", "", "last slot"}
	m := &WorldMode{}
	for _, key := range chatShortcutKeys {
		ctx.Input.SetKeyCode(key, true)
		if !m.chatShortcutFromInput(ctx) {
			t.Fatalf("key %v was not consumed", key)
		}
		if m.chatShortcutFromInput(ctx) {
			t.Fatalf("held key %v retriggered", key)
		}
		ctx.Input.SetKeyCode(key, false)
	}
	for _, key := range []input.Key{input.Key1, input.Key2, input.Key3, input.Key4, input.Key5, input.Key6, input.Key7, input.Key8, input.Key9} {
		if ctx.Input.JustPressed(key) {
			t.Fatal("legacy skill-bar edge was not consumed")
		}
	}
	if !ctx.Session.NoShift || !ctx.Session.NoCtrl {
		t.Fatal("local console commands were not executed")
	}
	want := network.BuildGlobalChatPacketForClientDate("Tester", "hello", 20080910)
	want = append(want, network.BuildGlobalChatPacketForClientDate("Tester", "@where", 20080910)...)
	want = append(want, network.BuildPartyMessagePacket("Tester : party")...)
	want = append(want, network.BuildGuildMessagePacket("Tester : guild")...)
	want = append(want, network.BuildWhisperPacket("Alice", "hello")...)
	want = append(want, network.BuildEmotionPacket(0)...)
	want = append(want, network.BuildGlobalChatPacketForClientDate("Tester", "last slot", 20080910)...)
	readBotTestPackets(t, server, want)
	assertNoBotTestPacket(t, server, func() error { return nil })
}

func TestChatShortcutIgnoresOtherModifiersAndModal(t *testing.T) {
	for _, modifier := range []gpucontext.Key{gpucontext.KeyRightAlt, gpucontext.KeyLeftControl, gpucontext.KeyLeftShift, gpucontext.KeyLeftSuper, gpucontext.KeyRightSuper} {
		t.Run(modifier.String(), func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
			ctx.Input.SetKeyCode(modifier, true)
			ctx.Input.SetKeyCode(gpucontext.Key1, true)
			ctx.Config.ChatShortcuts[0] = "/ns"
			m := &WorldMode{}
			if m.chatShortcutFromInput(ctx) || ctx.Session.NoShift {
				t.Fatal("modified key executed a shortcut")
			}
			ctx.Input.SetKeyCode(gpucontext.KeyL, true)
			ctx.Input.SetKeyCode(gpucontext.KeyG, true)
			if m.toggleEmoteWindowFromInput(ctx) || m.toggleGuildWindowFromInput(ctx) {
				t.Fatal("modified key executed a window shortcut")
			}
		})
	}
	ctx := chatShortcutTestContext(t)
	ctx.Config.ChatShortcuts[0] = "/ns"
	m := &WorldMode{}
	m.ui.settingsWindow.OpenWindow(ctx)
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	ctx.Input.SetKeyCode(gpucontext.Key1, true)
	m.chatShortcutFromInput(ctx)
	if ctx.Session.NoShift {
		t.Fatal("shortcut executed through settings")
	}
	ctx.Input.SetKeyCode(gpucontext.KeyM, true)
	if m.chatShortcutFromInput(ctx) || m.ui.chatShortcuts.IsOpen() {
		t.Fatal("shortcut editor opened through settings")
	}
}

func TestChatShortcutRetainedAcrossMapChange(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.ChatShortcuts[0] = "/sit"
	m := &WorldMode{}
	m.ui.chatShortcuts.Toggle(ctx, &m.ui.console, nil)
	next := m.nextWorldMode()
	next.rebindPersistentUI(ctx)
	if !next.ui.chatShortcuts.IsOpen() || next.ui.chatShortcuts.Command(ctx, 0) != "/sit" {
		t.Fatal("map change lost shortcut editor")
	}
}
