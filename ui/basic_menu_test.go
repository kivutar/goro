package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

type inertOverlay struct {
	widget.WidgetBase
}

func newInertOverlay() *inertOverlay {
	w := &inertOverlay{}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *inertOverlay) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.BiggestFinite(1, 1)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *inertOverlay) Draw(widget.Context, widget.Canvas) {}

func (w *inertOverlay) Event(widget.Context, event.Event) bool {
	return false
}

func (w *inertOverlay) Children() []widget.Widget {
	return nil
}

type basicMenuTestApp struct {
	app *uiapp.App
}

func (a basicMenuTestApp) SetUIRoot(root widget.Widget) {
	a.app.SetRoot(root)
}

func (a basicMenuTestApp) Frame() {
	a.app.Frame()
}

func (a basicMenuTestApp) Cursor() widget.CursorType {
	return a.app.Window().Context().Cursor()
}

func (a basicMenuTestApp) HoveredWidget() widget.Widget {
	return a.app.Window().HoveredWidget()
}

func TestBasicMenuRowsUsePointerCursor(t *testing.T) {
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	ctx := client.Context{
		Input:     input.NewState(),
		Session:   &session.Session{Selected: session.Character{Name: "Kivutar"}},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}
	var character CharacterWindow
	var menu BasicMenu
	character.Update(ctx)
	menu.Update(ctx)
	manager.AddOverlay(positionedWidget(newInertOverlay(), 600, 16, 188, 206))

	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})

	firstRow := geometry.Pt(
		float32(basicMenuX+basicMenuPad+basicMenuButtonW/2),
		float32(basicMenuY+basicMenuPad+basicMenuButtonH/2),
	)
	secondRow := geometry.Pt(
		firstRow.X,
		float32(basicMenuY+basicMenuPad+basicMenuButtonH+basicMenuGapY+basicMenuButtonH/2),
	)

	app.Window().HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0, firstRow, firstRow, event.ModNone))
	if got := app.Window().Context().Cursor(); got != widget.CursorPointer {
		t.Fatalf("first row cursor = %v, want pointer, hovered=%T", got, app.Window().HoveredWidget())
	}

	app.Window().HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0, secondRow, secondRow, event.ModNone))
	if got := app.Window().Context().Cursor(); got != widget.CursorPointer {
		t.Fatalf("second row cursor = %v, want pointer, hovered=%T", got, app.Window().HoveredWidget())
	}
}
