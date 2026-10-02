package ui

import (
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/uitest"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

func TestServiceWindowKeyboardAndControllerNavigation(t *testing.T) {
	app := uiapp.New()
	bridge := loginWindowTestApp{basicMenuTestApp{app: app}}
	manager := NewManager()
	manager.SetUIApp(bridge)
	ctx := client.Context{ScreenW: 800, ScreenH: 600, UIApp: bridge, UIManager: manager}
	selected := -1
	w := NewServiceWindow(ctx, []string{"1", "2", "3", "4", "5", "6", "7", "8"}, ServiceWindowOptions{}, ServiceWindowCallbacks{
		OnSelect: func(index int) { selected = index },
	})
	w.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if app.Window().Context().FocusedWidget() != w.list {
		t.Fatal("server list lacks initial keyboard focus")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyDown, 0, event.ModNone))
	if w.SelectedIndex() != 1 {
		t.Fatal("keyboard did not navigate server list")
	}
	for range 10 {
		w.Control("down")
	}
	if w.SelectedIndex() != 7 || w.scrollY.Get() <= 0 {
		t.Fatal("controller selection did not clamp and scroll into view")
	}
	w.Control("confirm")
	if selected != 7 {
		t.Fatalf("confirmed %d, want 7", selected)
	}
}

func TestServiceWindowClampsInitialSelection(t *testing.T) {
	window := NewServiceWindow(client.Context{ScreenW: 1280, ScreenH: 720}, []string{"Local", "Internet"}, ServiceWindowOptions{Selected: 8}, ServiceWindowCallbacks{})

	if got := window.SelectedIndex(); got != 0 {
		t.Fatalf("selected service = %d, want 0", got)
	}
}

func TestServiceWindowUsesStageTitle(t *testing.T) {
	window := NewServiceWindow(client.Context{ScreenW: 1280, ScreenH: 720}, []string{"Local"}, ServiceWindowOptions{Title: "Server"}, ServiceWindowCallbacks{})

	if window.title != "Server" {
		t.Fatalf("service window title = %q, want Server", window.title)
	}
}

func TestServiceWindowConfirmUsesCurrentSelection(t *testing.T) {
	selected := -1
	window := NewServiceWindow(client.Context{ScreenW: 1280, ScreenH: 720}, []string{"Local", "Internet"}, ServiceWindowOptions{Selected: 1}, ServiceWindowCallbacks{
		OnSelect: func(index int) {
			selected = index
		},
	})

	window.confirm()

	if selected != 1 {
		t.Fatalf("confirmed service = %d, want 1", selected)
	}
}

func TestServiceWindowCannotConfirmEmptyList(t *testing.T) {
	called := false
	window := NewServiceWindow(client.Context{ScreenW: 1280, ScreenH: 720}, nil, ServiceWindowOptions{}, ServiceWindowCallbacks{
		OnSelect: func(int) {
			called = true
		},
	})

	window.confirm()

	if called {
		t.Fatal("empty service list was confirmed")
	}
}

func TestServiceWindowEnterConfirmsCurrentSelection(t *testing.T) {
	selected := -1
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 1280, ScreenH: 720}
	window := NewServiceWindow(ctx, []string{"Local", "Internet"}, ServiceWindowOptions{Selected: 1}, ServiceWindowCallbacks{
		OnSelect: func(index int) {
			selected = index
		},
	})
	inputState.SetKey(input.KeyEnter, true)

	if !window.Update(ctx) {
		t.Fatal("Enter was not consumed")
	}
	if selected != 1 {
		t.Fatalf("confirmed service = %d, want 1", selected)
	}
}
