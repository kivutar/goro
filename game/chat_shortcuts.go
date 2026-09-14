package game

import (
	"strings"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

// SuppressShortcutText filters only active physical shortcuts before their
// accompanying text reaches editors. Actions still run in frame input handling.
func (m *WorldMode) SuppressShortcutText(ctx client.Context, code input.KeyCode) bool {
	if !plainAltDown(ctx.Input) || code == gpucontext.KeyUnknown {
		return false
	}
	if !m.serverProgress.started.IsZero() || m.mapFade.phase == mapFadeOut || m.mapFade.phase == mapFadeHold || m.mapFade.phase == mapFadePrewarm {
		return false
	}
	if code == gpucontext.KeyM {
		return m.ui.chatShortcuts.IsOpen() || !m.ui.nonConsoleKeyboardInputBlocked(ctx)
	}
	if m.ui.nonConsoleKeyboardInputBlocked(ctx) {
		return false
	}
	if code == gpucontext.KeyL || code == gpucontext.KeyG {
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
		if !m.ui.chatShortcuts.IsOpen() && m.ui.nonConsoleKeyboardInputBlocked(ctx) {
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
		if !m.ui.nonConsoleKeyboardInputBlocked(ctx) {
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
