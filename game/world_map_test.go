package game

import (
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

func TestWorldMapShortcutPhysicalKeyAndModifiers(t *testing.T) {
	for _, tc := range []struct {
		name                string
		altGr, shift, modal bool
		want                bool
	}{
		{"plain", false, false, false, true}, {"AltGr", true, false, false, false},
		{"shift", false, true, false, false}, {"modal", false, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := client.Context{Input: input.NewState()}
			ctx.Input.SetKey(input.KeyCtrl, true)
			ctx.Input.SetKeyCode(gpucontext.KeyGrave, true)
			ctx.Input.SetKeyCode(gpucontext.KeyRightAlt, tc.altGr)
			ctx.Input.SetKey(input.KeyShift, tc.shift)
			m := &WorldMode{}
			if tc.modal {
				m.ui.escapeMenu.Window.Open(ctx, nil)
			}
			if got := m.suppressShortcutText(ctx, gpucontext.KeyGrave); got != tc.want {
				t.Fatalf("suppress = %v", got)
			}
			if got := m.toggleWorldMapFromInput(ctx); got != tc.want {
				t.Fatalf("toggle = %v", got)
			}
			if tc.want && ctx.Input.KeyCodeJustPressed(gpucontext.KeyGrave) {
				t.Fatal("shortcut not consumed")
			}
		})
	}
}

func TestOpenWorldMapBlocksGameplayButKeepsItsToggle(t *testing.T) {
	ctx := client.Context{Input: input.NewState()}
	m := &WorldMode{}
	m.ui.worldMap.Window.Open(ctx, nil)
	if !m.ui.nonConsoleKeyboardInputBlocked() {
		t.Fatal("atlas did not block gameplay shortcuts and the Escape menu")
	}
	if !m.nextWorldMode().ui.worldMap.IsOpen() {
		t.Fatal("map transition did not carry the atlas")
	}
	ctx.Input.SetKey(input.KeyCtrl, true)
	ctx.Input.SetKeyCode(gpucontext.KeyGrave, true)
	if !m.suppressShortcutText(ctx, gpucontext.KeyGrave) || !m.toggleWorldMapFromInput(ctx) || m.ui.worldMap.IsOpen() {
		t.Fatal("open atlas blocked its own close shortcut")
	}
}
