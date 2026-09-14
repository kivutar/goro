package game

import (
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
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
	if !m.chatShortcutFromInput(ctx) || !m.ui.chatShortcuts.IsOpen() {
		t.Fatal("Alt+M did not open shortcut list")
	}
	if !m.ui.keyboardInputBlocked(ctx) {
		t.Fatal("shortcut editor did not block game hotkeys")
	}
	if m.chatShortcutFromInput(ctx) {
		t.Fatal("held Alt+M retriggered")
	}
	ctx.Input.SetKeyCode(gpucontext.KeyM, false)
	ctx.Input.SetKeyCode(gpucontext.KeyM, true)
	if !m.chatShortcutFromInput(ctx) || m.ui.chatShortcuts.IsOpen() {
		t.Fatal("Alt+M did not close shortcut list")
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
	for _, modifier := range []gpucontext.Key{gpucontext.KeyRightAlt, gpucontext.KeyLeftControl, gpucontext.KeyLeftShift} {
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
