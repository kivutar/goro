package ui

import (
	"path/filepath"
	"testing"

	"github.com/gogpu/gpucontext"
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestSettingsTabsFitSmallWindowAndKeepSelectionAfterToggle(t *testing.T) {
	app := uiapp.New(uiapp.WithWindowProvider(gpucontext.NullWindowProvider{W: 640, H: 480}))
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	ctx := client.Context{
		Input: input.NewState(), Session: session.New(), UIManager: manager, UIApp: basicMenuTestApp{app: app},
		ScreenW: 640, ScreenH: 480,
		Config: config.Config{ConfigPath: filepath.Join(t.TempDir(), "goro.ini"), Render: config.RenderConfig{Anisotropy: 8}},
	}
	var window SettingsWindow
	window.OpenWindow(ctx)
	draw := func() map[string]geometry.Rect {
		app.Frame()
		canvas := &settingsLabelCanvas{labels: make(map[string]geometry.Rect)}
		app.Window().DrawTo(canvas)
		return canvas.labels
	}
	labels := draw()
	anisotropic, hasLabel := labels["Anisotropic filtering"]
	button, hasButton := labels[anisotropyLabel(settingsAnisotropy(ctx))]
	if !hasLabel || !hasButton {
		t.Fatal("anisotropic label or button is missing")
	}
	if anisotropic.Center().Y != button.Center().Y {
		t.Fatalf("anisotropic vertical centers differ: label=%g button=%g", anisotropic.Center().Y, button.Center().Y)
	}
	upscaling, hasUpscaling := labels["Upscaling (Next map)"]
	upscaleButton, hasUpscaleButton := labels["Off"]
	if !hasUpscaling || !hasUpscaleButton || upscaling.Center().Y != upscaleButton.Center().Y {
		t.Fatal("texture upscaling control is missing or misaligned")
	}
	click := func(label string) {
		t.Helper()
		bounds, ok := draw()[label]
		if !ok {
			t.Fatalf("control %q is not drawn", label)
		}
		p := bounds.Center()
		app.Window().HandleEvent(uitest.Click(p.X, p.Y))
		app.Window().HandleEvent(uitest.Release(p.X, p.Y))
	}
	for _, section := range []struct{ tab, lastControl string }{
		{"Display", "Ambient occlusion"},
		{"Sound", "SFX Vol"},
		{"Gameplay", "Keyboard & joypad controls"},
	} {
		click(section.tab)
		labels := draw()
		if _, ok := labels[section.lastControl]; !ok {
			t.Fatalf("%s: last control %q is missing", section.tab, section.lastControl)
		}
		for label, bounds := range labels {
			if bounds.Min.X < 0 || bounds.Min.Y < 0 || bounds.Max.X > 640 || bounds.Max.Y > 480 {
				t.Fatalf("%s: control %q outside 640x480: %v", section.tab, label, bounds)
			}
		}
	}
	click("No Shift")
	if !ctx.Session.NoShift || window.tab != settingsTabGameplay {
		t.Fatal("toggle did not apply or changed the selected tab")
	}
	if _, ok := draw()["Keyboard & joypad controls"]; !ok {
		t.Fatal("refresh replaced the selected section")
	}
}

// Record screen coordinates, including the nested layout transforms.
type settingsLabelCanvas struct {
	uitest.MockCanvas
	labels map[string]geometry.Rect
}

func (c *settingsLabelCanvas) DrawStyledText(text string, bounds geometry.Rect, _ widget.TextStyle) {
	c.labels[text] = bounds.Translate(c.TransformOffset())
}

func (c *settingsLabelCanvas) DrawText(text string, bounds geometry.Rect, _ float32, _ widget.Color, _ bool, _ widget.TextAlign) {
	c.labels[text] = bounds.Translate(c.TransformOffset())
}

func TestSettingsWindowEscapeCloses(t *testing.T) {
	var window SettingsWindow
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	window.OpenWindow(ctx)

	inputState.SetKey(input.KeyEscape, true)
	if !window.Update(ctx) {
		t.Fatal("settings window did not consume escape")
	}
	if window.IsOpen() {
		t.Fatal("settings window stayed open after escape")
	}
}

func TestSettingsWindowCloseButtonCloses(t *testing.T) {
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}
	var window SettingsWindow
	window.OpenWindow(ctx)

	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	x, y, _, _ := centeredWindowRect(ctx, settingsWindowW, settingsWindowH)
	buttonX := float32(x + settingsWindowW - 16)
	buttonY := float32(y + ROWindowTitleHeight/2)
	inputState.SetMousePosition(int(buttonX), int(buttonY))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !window.Update(ctx) {
		t.Fatal("settings window did not consume close-button press")
	}
	if window.dragging || window.dragLayer {
		t.Fatal("close-button press started a window drag")
	}

	app.Window().HandleEvent(uitest.Click(buttonX, buttonY))
	app.Window().HandleEvent(uitest.Release(buttonX, buttonY))

	if window.IsOpen() {
		t.Fatal("settings window stayed open after close button")
	}
}
