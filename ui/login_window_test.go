package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
)

type loginWindowTestApp struct{ basicMenuTestApp }

func (a loginWindowTestApp) WidgetContext() widget.Context { return a.app.Window().Context() }

func TestLoginWindowInitialFocusAndTabNavigation(t *testing.T) {
	app := uiapp.New()
	bridge := loginWindowTestApp{basicMenuTestApp{app: app}}
	manager := NewManager()
	manager.SetUIApp(bridge)
	ctx := client.Context{ScreenW: 800, ScreenH: 600, UIApp: bridge, UIManager: manager}
	window := NewLoginWindow(ctx, "", "", false, LoginWindowCallbacks{})
	window.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if app.Window().Context().FocusedWidget() != window.user {
		t.Error("Account's initial focus was not registered with the UI context")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'a', event.ModNone))
	if window.Username != "a" {
		t.Fatal("Account did not accept typing before the first Tab")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModNone))
	if !window.password.IsFocused() || window.user.IsFocused() || app.Window().Context().FocusedWidget() != window.password {
		t.Fatal("first Tab did not move from Account to Password")
	}
	// Publishing/updating an unchanged form must not reset the chosen field.
	window.SetContext(ctx)
	window.Publish(ctx)
	if app.Window().Context().FocusedWidget() != window.password {
		t.Fatal("updating the login window stole focus from Password")
	}
	// Resizing rebuilds the fields: the context must follow the new instance.
	oldPassword := window.password
	ctx.ScreenW = 1024
	window.SetContext(ctx)
	window.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if window.password == oldPassword || app.Window().Context().FocusedWidget() != window.password || oldPassword.IsFocused() {
		t.Fatal("resizing did not transfer focus to the rebuilt Password field")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModShift))
	if !window.user.IsFocused() || window.password.IsFocused() || app.Window().Context().FocusedWidget() != window.user {
		t.Fatal("Shift+Tab did not return to Account after resizing")
	}
}

func TestLoginWindowLabelsFillRightAlignedColumn(t *testing.T) {
	width, height := loginWindowSize()
	window := &LoginWindow{layout: loginWindowLayout{W: width, H: height}}
	tree := window.widgetTree()
	tree.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(float32(width), float32(height))))

	windowChildren := tree.Children()
	if len(windowChildren) < 2 || len(windowChildren[1].Children()) != 1 {
		t.Fatal("login window content tree is incomplete")
	}
	rows := windowChildren[1].Children()[0].Children()[0].Children()
	if len(rows) != 2 {
		t.Fatalf("login form rows = %d, want Account and Password", len(rows))
	}

	for i, want := range []string{"Account", "Password"} {
		rowChildren := rows[i].Children()
		if len(rowChildren) != 2 || len(rowChildren[0].Children()) != 1 {
			t.Fatalf("%s row does not contain a label and field", want)
		}
		labelSlot := rowChildren[0]
		label, ok := labelSlot.Children()[0].(*primitives.TextWidget)
		if !ok {
			t.Fatalf("%s label = %T, want text", want, labelSlot.Children()[0])
		}
		if label.Content() != want {
			t.Fatalf("login label = %q, want %q", label.Content(), want)
		}
		if label.Style().Align != widget.TextAlignRight {
			t.Fatalf("%s alignment = %v, want right", want, label.Style().Align)
		}

		slotBounds := labelSlot.(interface{ Bounds() geometry.Rect }).Bounds()
		labelBounds := label.Bounds()
		if labelBounds.Min.X != 0 || labelBounds.Width() != slotBounds.Width() {
			t.Fatalf("%s label bounds = %v, want full %.1fpx column width", want, labelBounds, slotBounds.Width())
		}
	}
}

func TestLoginWindowKeepToggleSurvivesResize(t *testing.T) {
	app := uiapp.New()
	bridge := loginWindowTestApp{basicMenuTestApp{app: app}}
	manager := NewManager()
	manager.SetUIApp(bridge)
	ctx := client.Context{ScreenW: 800, ScreenH: 600, UIApp: bridge, UIManager: manager}
	submitted := false
	window := NewLoginWindow(ctx, "remembered-id", "", true, LoginWindowCallbacks{
		OnSubmit: func() { submitted = true },
	})
	window.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if app.Window().Context().FocusedWidget() != window.password {
		t.Fatal("prefilled ID did not focus Password")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModNone))
	if app.Window().Context().FocusedWidget() != window.keep {
		t.Fatal("Keep was not reachable after Password")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeySpace, 0, event.ModNone))
	app.HandleEvent(event.NewKeyEvent(event.KeyRelease, event.KeySpace, 0, event.ModNone))
	if window.KeepID || submitted {
		t.Fatal("Keep did not toggle independently of login submission")
	}
	ctx.ScreenW = 1024
	window.SetContext(ctx)
	window.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if window.KeepID || window.Username != "remembered-id" || app.Window().Context().FocusedWidget() != window.keep {
		t.Fatal("resize changed the ID, Keep choice, or checkbox focus")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeySpace, 0, event.ModNone))
	app.HandleEvent(event.NewKeyEvent(event.KeyRelease, event.KeySpace, 0, event.ModNone))
	if !window.KeepID {
		t.Fatal("rebuilt checkbox did not retain its unchecked state")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModShift))
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if !submitted {
		t.Fatal("returning to Password did not allow login submission")
	}
}
