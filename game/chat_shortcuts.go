package game

import (
	"strings"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/input"
)

// PrepareTextInput routes ordinary typing to classic chat, or filters the text
// accompanying active shortcuts. Shortcut actions still run in frame input.
func (m *WorldMode) PrepareTextInput(ctx client.Context, code input.KeyCode) bool {
	if m.uiInputSuspended() {
		return false
	}
	if m.suppressShortcutText(ctx, code) {
		return true
	}
	if !m.ui.nonConsoleKeyboardInputBlocked() {
		return m.ui.console.PrepareTextInput(ctx, code)
	}
	return false
}

func (m *WorldMode) PrepareKeyInput(ctx client.Context, code input.KeyCode, mods gpucontext.Modifiers) {
	if !m.uiInputSuspended() && !m.ui.nonConsoleKeyboardInputBlocked() {
		m.ui.console.PrepareKeyInput(ctx, code, mods)
	}
}

func (m *WorldMode) uiInputSuspended() bool {
	return m.loading != nil || !m.npcSystem.Progress.Started.IsZero() || m.mapFade.Phase == maps.FadeOut || m.mapFade.Phase == maps.FadeHold || m.mapFade.Phase == maps.FadePrewarm
}

func (m *WorldMode) suppressShortcutText(ctx client.Context, code input.KeyCode) bool {
	if m.windowShortcutAllowed(ctx, code) {
		return true
	}
	if code == gpucontext.KeyGrave && plainCtrlDown(ctx.Input) && (m.ui.worldMap.IsOpen() || !m.ui.nonConsoleKeyboardInputBlocked()) {
		return true
	}
	if !plainAltDown(ctx.Input) || code == gpucontext.KeyUnknown {
		return false
	}
	if code == gpucontext.KeyM {
		return m.ui.chatShortcuts.IsOpen() || !m.ui.nonConsoleKeyboardInputBlocked()
	}
	if m.ui.nonConsoleKeyboardInputBlocked() {
		return false
	}
	if code == gpucontext.KeyL || code == gpucontext.KeyG || code == gpucontext.KeyU {
		return true
	}
	for slot, key := range chatShortcutKeys {
		if code == key {
			return strings.TrimSpace(m.ui.chatShortcuts.Command(ctx, slot)) != ""
		}
	}
	return false
}

// These are physical keys: Alt+number works on the AZERTY number row too,
// without Shift. Right Alt (AltGr) must remain available for typing.
var chatShortcutKeys = [...]input.KeyCode{
	gpucontext.Key1, gpucontext.Key2, gpucontext.Key3, gpucontext.Key4, gpucontext.Key5,
	gpucontext.Key6, gpucontext.Key7, gpucontext.Key8, gpucontext.Key9, gpucontext.Key0,
}

func (m *WorldMode) chatShortcutFromInput(ctx client.Context) bool {
	in := ctx.Input
	if !plainAltDown(in) {
		return false
	}
	if in.KeyCodeJustPressed(gpucontext.KeyM) {
		if !m.ui.chatShortcuts.IsOpen() && m.ui.nonConsoleKeyboardInputBlocked() {
			return false
		}
		in.ConsumeKeyCodePress(gpucontext.KeyM)
		m.ui.emoteWindow.OnSelect = m.ui.chatShortcuts.SelectEmotion
		m.ui.chatShortcuts.Toggle(ctx, &m.ui.console, func() {
			m.ui.emoteWindow.OpenWindow(ctx, &m.ui.console)
		})
		return true
	}
	for slot, key := range chatShortcutKeys {
		if !in.KeyCodeJustPressed(key) {
			continue
		}
		// Even an empty/blocked binding must not fall through to a skill slot.
		in.ConsumeKeyCodePress(key)
		if !m.ui.nonConsoleKeyboardInputBlocked() {
			m.ui.console.SendText(ctx, m.ui.chatShortcuts.Command(ctx, slot))
		}
		return true
	}
	return false
}

func plainAltDown(in *input.State) bool {
	return in != nil && in.Pressed(input.KeyAlt) && !in.Pressed(input.KeyCtrl) &&
		!in.Pressed(input.KeyShift) && !in.KeyCodeDown(gpucontext.KeyRightAlt) &&
		!in.KeyCodeDown(gpucontext.KeyLeftSuper) && !in.KeyCodeDown(gpucontext.KeyRightSuper)
}

func plainCtrlDown(in *input.State) bool {
	return in != nil && in.Pressed(input.KeyCtrl) && !in.Pressed(input.KeyAlt) &&
		!in.Pressed(input.KeyShift) && !in.KeyCodeDown(gpucontext.KeyRightAlt) &&
		!in.KeyCodeDown(gpucontext.KeyLeftSuper) && !in.KeyCodeDown(gpucontext.KeyRightSuper)
}
