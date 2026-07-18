package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

type countingOverlay struct {
	widget.WidgetBase
	events int
}

func newCountingOverlay() *countingOverlay {
	w := &countingOverlay{}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *countingOverlay) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.BiggestFinite(1, 1)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *countingOverlay) Draw(widget.Context, widget.Canvas) {}

func (w *countingOverlay) Event(widget.Context, event.Event) bool {
	w.events++
	return false
}

func (w *countingOverlay) Children() []widget.Widget { return nil }

type lifecycleOverlay struct {
	widget.WidgetBase
	mounts   int
	unmounts int
}

func newLifecycleOverlay() *lifecycleOverlay {
	w := &lifecycleOverlay{}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *lifecycleOverlay) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.BiggestFinite(1, 1)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *lifecycleOverlay) Draw(widget.Context, widget.Canvas) {}

func (w *lifecycleOverlay) Event(widget.Context, event.Event) bool { return false }

func (w *lifecycleOverlay) Children() []widget.Widget { return nil }

func (w *lifecycleOverlay) Mount(widget.Context) { w.mounts++ }

func (w *lifecycleOverlay) Unmount() { w.unmounts++ }

type countingManagerApp struct {
	app         *uiapp.App
	roots       int
	invalidates int
}

func newCountingManagerApp() *countingManagerApp {
	return &countingManagerApp{app: uiapp.New()}
}

func (a *countingManagerApp) SetUIRoot(root widget.Widget) {
	a.roots++
	a.app.SetRoot(root)
}

func (a *countingManagerApp) Frame() {
	a.app.Frame()
}

func (a *countingManagerApp) Invalidate() {
	a.invalidates++
	if a.app.Window() != nil && a.app.Window().Context() != nil {
		a.app.Window().Context().Invalidate()
	}
}

func (a *countingManagerApp) Cursor() widget.CursorType {
	return a.app.Window().Context().Cursor()
}

func (a *countingManagerApp) HoveredWidget() widget.Widget {
	return a.app.Window().HoveredWidget()
}

func TestManagerPointerBlockedUsesOverlayBounds(t *testing.T) {
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	manager.AddOverlay(positionedWidget(newInertOverlay(), 100, 120, 80, 40))
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})

	if !manager.PointerBlocked(120, 130) {
		t.Fatal("pointer over overlay was not blocked")
	}
	if manager.PointerBlocked(90, 130) {
		t.Fatal("pointer outside overlay was blocked")
	}
}

func TestTopOverlayBlocksLowerOverlayEvents(t *testing.T) {
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	lower := newCountingOverlay()
	top := newCountingOverlay()
	manager.AddOverlay(positionedWidget(lower, 100, 100, 80, 80))
	manager.AddOverlay(positionedWidget(top, 120, 120, 80, 80))
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})

	point := geometry.Pt(130, 130)
	app.Window().HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, 0, point, point, event.ModNone))
	if top.events != 1 {
		t.Fatalf("top overlay events = %d, want 1", top.events)
	}
	if lower.events != 0 {
		t.Fatalf("lower overlay events = %d, want 0", lower.events)
	}
}

func TestManagerRemoveOverlayUnmountsDetachedChild(t *testing.T) {
	app := newCountingManagerApp()
	manager := NewManager()
	manager.SetUIApp(app)
	overlay := newLifecycleOverlay()
	manager.AddOverlay(overlay)

	if !overlay.IsMounted() {
		t.Fatal("overlay was not mounted")
	}
	if overlay.mounts == 0 {
		t.Fatal("overlay Mount was not called")
	}
	unmountsBeforeRemove := overlay.unmounts

	manager.RemoveOverlay(overlay)

	if overlay.IsMounted() {
		t.Fatal("removed overlay is still mounted")
	}
	if overlay.unmounts != unmountsBeforeRemove+1 {
		t.Fatalf("removed overlay unmounts = %d, want %d", overlay.unmounts, unmountsBeforeRemove+1)
	}
}

func TestManagerClearUnmountsDetachedChildren(t *testing.T) {
	app := newCountingManagerApp()
	manager := NewManager()
	manager.SetUIApp(app)
	first := newLifecycleOverlay()
	second := newLifecycleOverlay()
	manager.AddOverlay(first)
	manager.AddOverlay(second)

	firstUnmounts := first.unmounts
	secondUnmounts := second.unmounts

	manager.Clear()

	if first.IsMounted() || second.IsMounted() {
		t.Fatalf("cleared overlays still mounted: first=%v second=%v", first.IsMounted(), second.IsMounted())
	}
	if first.unmounts != firstUnmounts+1 || second.unmounts != secondUnmounts+1 {
		t.Fatalf("cleared overlays unmounts: first=%d second=%d, want %d and %d", first.unmounts, second.unmounts, firstUnmounts+1, secondUnmounts+1)
	}
}
