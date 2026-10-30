package app

import (
	"path/filepath"
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/input"
	gameui "github.com/kivutar/goro/ui"
)

func TestFullscreenShortcut(t *testing.T) {
	for _, alt := range []input.KeyCode{gpucontext.KeyLeftAlt, gpucontext.KeyRightAlt} {
		g := &Game{input: input.NewState(), runtime: newRuntimeSettings(config.Config{})}
		g.input.SetKeyCode(alt, true)
		for _, want := range []bool{true, false, true} {
			g.input.SetKeyCode(gpucontext.KeyEnter, true)
			g.HandleKeyPress(gpucontext.KeyEnter)
			if g.RuntimeFullscreen() != want {
				t.Fatalf("fullscreen=%v, want %v", g.RuntimeFullscreen(), want)
			}
			if !g.input.KeyCodeConsumed(gpucontext.KeyEnter) || g.input.JustPressed(input.KeyEnter) {
				t.Fatal("fullscreen shortcut left Enter available to chat or dialogs")
			}
			g.input.SetKeyCode(gpucontext.KeyEnter, false)
			g.input.EndFrame()
		}
		g.input.SetKeyCode(alt, false)
		g.input.SetKeyCode(gpucontext.KeyEnter, true)
		g.HandleKeyPress(gpucontext.KeyEnter)
		if !g.RuntimeFullscreen() || g.input.KeyCodeConsumed(gpucontext.KeyEnter) {
			t.Fatal("plain Enter changed fullscreen or was consumed")
		}
	}
}

func TestFullscreenCheckboxFollowsShortcut(t *testing.T) {
	g := &Game{input: input.NewState(), runtime: newRuntimeSettings(config.Config{})}
	ctx := client.Context{
		Runtime: g.runtime, Input: g.input, ScreenW: 800, ScreenH: 600,
		Config: config.Config{ConfigPath: filepath.Join(t.TempDir(), "goro.ini")},
	}
	var window gameui.SettingsWindow
	window.OpenWindow(ctx)
	var findCheckbox func(widget.Widget) *checkbox.Widget
	findCheckbox = func(w widget.Widget) *checkbox.Widget {
		if cb, ok := w.(*checkbox.Widget); ok {
			return cb
		}
		for _, child := range w.Children() {
			if cb := findCheckbox(child); cb != nil {
				return cb
			}
		}
		return nil
	}
	cb := findCheckbox(window.Widget()) // Fullscreen is the first display control.
	if cb == nil {
		t.Fatal("fullscreen checkbox is missing")
	}
	cb.SetBounds(geometry.NewRect(0, 0, 200, 24))
	wctx := widget.NewContext()
	canvas := &fullscreenCheckboxCanvas{}
	cb.Draw(wctx, canvas)
	if canvas.lines != 0 {
		t.Fatal("windowed mode drew a fullscreen checkmark")
	}
	g.input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	g.input.SetKeyCode(gpucontext.KeyEnter, true)
	g.HandleKeyPress(gpucontext.KeyEnter)
	cb.Draw(wctx, canvas)
	if canvas.lines != 2 {
		t.Fatal("the existing checkbox did not draw the fullscreen checkmark after Alt+Enter")
	}
	cb.Event(wctx, uitest.Click(8, 8))
	cb.Event(wctx, uitest.Release(8, 8))
	if g.RuntimeFullscreen() {
		t.Fatal("clicking the checked box did not leave fullscreen")
	}
}

type fullscreenCheckboxCanvas struct {
	uitest.MockCanvas
	lines int
}

func (c *fullscreenCheckboxCanvas) DrawLine(_, _ geometry.Point, _ widget.Color, _ float32) {
	c.lines++
}
