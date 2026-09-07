package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
)

type countingOverlay struct {
	widget.WidgetBase
	events  int
	presses int
	draws   int
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

func (w *countingOverlay) Draw(widget.Context, widget.Canvas) { w.draws++ }

func (w *countingOverlay) Event(_ widget.Context, e event.Event) bool {
	if mouse, ok := e.(*event.MouseEvent); ok && mouse.IsPress() {
		w.presses++
	}
	w.events++
	return false
}

func (w *countingOverlay) Children() []widget.Widget { return nil }

func TestOverlayRootReportsWhetherItIsEmpty(t *testing.T) {
	if root := newOverlayRoot(nil); !root.IsUIRootEmpty() {
		t.Fatal("root without overlays did not report itself empty")
	}
	if root := newOverlayRoot([]widget.Widget{newCountingOverlay()}); root.IsUIRootEmpty() {
		t.Fatal("root with an overlay reported itself empty")
	}
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
	if top.presses != 1 {
		t.Fatalf("top overlay presses = %d, want 1", top.presses)
	}
	if lower.presses != 0 {
		t.Fatalf("lower overlay presses = %d, want 0", lower.presses)
	}
}

func TestClickingWindowRaisesItAboveOtherWindows(t *testing.T) {
	manager := NewManager()
	ctx := client.Context{UIManager: manager}
	lowerContent := newCountingOverlay()
	topContent := newCountingOverlay()
	lower := NewWindow(100, 80)
	top := NewWindow(100, 80)
	lower.OpenAt(100, 100, lowerContent)
	top.OpenAt(150, 100, topContent)
	lower.Publish(ctx)
	top.Publish(ctx)
	root := manager.root
	root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(300, 240)))
	widget.ClearRedrawInTree(root)

	lowerOnly := geometry.Pt(125, 125)
	root.Event(widget.NewContext(), event.NewMouseEvent(event.MousePress, event.ButtonLeft, 0, lowerOnly, lowerOnly, event.ModNone))
	if manager.root != root {
		t.Fatal("raising a window replaced the UI root")
	}
	if got := manager.overlays[len(manager.overlays)-1]; got != lower.published {
		t.Fatal("clicked window was not raised in the manager order")
	}
	if got := root.children[len(root.children)-1]; got != lower.published {
		t.Fatal("clicked window was not raised in the active root")
	}
	if !root.NeedsRedraw() {
		t.Fatal("raising a window did not invalidate the overlay root")
	}

	overlap := geometry.Pt(175, 125)
	root.Event(widget.NewContext(), event.NewMouseEvent(event.MousePress, event.ButtonLeft, 0, overlap, overlap, event.ModNone))
	if lowerContent.events != 2 {
		t.Fatalf("raised window events = %d, want 2", lowerContent.events)
	}
	if topContent.events != 0 {
		t.Fatalf("covered window events = %d, want 0", topContent.events)
	}
}

func TestClickingPlainOverlayDoesNotRaiseIt(t *testing.T) {
	manager := NewManager()
	lower := positionedWidget(newCountingOverlay(), 100, 100, 100, 80)
	top := positionedWidget(newCountingOverlay(), 150, 100, 100, 80)
	manager.AddOverlay(lower)
	manager.AddOverlay(top)
	root := manager.root
	root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(300, 240)))

	point := geometry.Pt(125, 125)
	root.Event(widget.NewContext(), event.NewMouseEvent(event.MousePress, event.ButtonLeft, 0, point, point, event.ModNone))
	if got := manager.overlays[len(manager.overlays)-1]; got != top {
		t.Fatal("plain overlay changed stacking order after a click")
	}
}

func TestForegroundOverlayStaysAboveNewWindows(t *testing.T) {
	manager := NewManager()
	foregroundContent := newCountingOverlay()
	foreground := positionedWidget(foregroundContent, 100, 100, 80, 80)
	lower := positionedWidget(newCountingOverlay(), 100, 100, 80, 80)
	window := positionedWidget(newCountingOverlay(), 100, 100, 80, 80)

	manager.AddForegroundOverlay(foreground)
	manager.AddOverlay(lower)
	manager.AddOverlay(window)
	manager.RaiseOverlay(lower)
	if len(manager.overlays) != 3 || manager.overlays[0] != window || manager.overlays[1] != lower || manager.overlays[2] != foreground {
		t.Fatalf("overlay order = %+v", manager.overlays)
	}

	manager.root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(300, 240)))
	point := geometry.Pt(120, 120)
	manager.root.Event(widget.NewContext(), event.NewMouseEvent(event.MousePress, event.ButtonLeft, 0, point, point, event.ModNone))
	if foregroundContent.events != 1 {
		t.Fatalf("foreground events = %d, want 1", foregroundContent.events)
	}
}

func TestOverlayRootDrawSkipsChildrenOutsideClip(t *testing.T) {
	left := newCountingOverlay()
	right := newCountingOverlay()
	root := newOverlayRoot([]widget.Widget{
		positionedWidget(left, 10, 20, 40, 30),
		positionedWidget(right, 200, 20, 40, 30),
	})
	root.Layout(nil, geometry.Tight(geometry.Sz(300, 100)))

	canvas := clippedOverlayCanvas{
		MockCanvas: uitest.MockCanvas{},
		clip:       geometry.NewRect(190, 0, 80, 80),
	}
	root.Draw(nil, &canvas)

	if left.draws != 0 {
		t.Fatalf("left overlay draws = %d, want 0", left.draws)
	}
	if right.draws != 1 {
		t.Fatalf("right overlay draws = %d, want 1", right.draws)
	}
}

type clippedOverlayCanvas struct {
	uitest.MockCanvas
	clip geometry.Rect
}

func (c *clippedOverlayCanvas) ClipBounds() geometry.Rect {
	return c.clip
}
