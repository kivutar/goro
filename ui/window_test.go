package ui

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestFooterStretchesContent(t *testing.T) {
	button := primitives.Box().Width(30).Height(10)
	row := primitives.HBox(
		primitives.Expanded(primitives.Box()),
		button,
	)
	window := Win(
		TitleBar(false),
		Size(200, 80),
		Footer(row),
	)

	window.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(200, 80)))

	if got := row.Bounds().Width(); got != 180 {
		t.Fatalf("footer row width = %.1f, want 180.0", got)
	}
	if got := button.Bounds().Min.X; got != 150 {
		t.Fatalf("right footer button x = %.1f, want 150.0", got)
	}
}

func TestFooterCreatesEmptyFooterBand(t *testing.T) {
	window := Win(
		TitleBar(false),
		Size(200, 80),
		Footer(primitives.Box()),
	)

	window.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(200, 80)))

	children := window.Children()
	if len(children) != 1 {
		t.Fatalf("window children = %d, want 1 footer", len(children))
	}
	footer := children[0]
	if got := footer.(interface{ Bounds() geometry.Rect }).Bounds().Width(); got != 200 {
		t.Fatalf("footer width = %.1f, want 200.0", got)
	}
	footerChildren := footer.Children()
	if len(footerChildren) != 2 {
		t.Fatalf("footer children = %d, want divider and body", len(footerChildren))
	}
	body := footerChildren[1]
	if got := body.(interface{ Bounds() geometry.Rect }).Bounds().Width(); got != 200 {
		t.Fatalf("footer body width = %.1f, want 200.0", got)
	}
}

func TestWindowTitleButtonsAreLaidOutBesideClose(t *testing.T) {
	controls := windowTitleButtons(
		[]windowTitleButton{{kind: rotheme.IconButtonMinus}},
		true,
		nil,
	)
	width := float32(windowTitleButtonSize*2 + windowTitleButtonGap)
	controls.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(width, ROWindowTitleHeight)))

	children := controls.Children()
	if len(children) != 2 {
		t.Fatalf("title controls = %d, want minus and close", len(children))
	}
	if got := children[0].(interface{ Bounds() geometry.Rect }).Bounds().Min.X; got != 0 {
		t.Fatalf("minus button x = %.1f, want 0", got)
	}
	if got := children[1].(interface{ Bounds() geometry.Rect }).Bounds().Min.X; got != float32(windowTitleButtonSize+windowTitleButtonGap) {
		t.Fatalf("close button x = %.1f, want %d", got, windowTitleButtonSize+windowTitleButtonGap)
	}
}

func TestWindowTitleButtonHitCoversAllConfiguredButtons(t *testing.T) {
	window := NewWindow(100, 80)
	window.OpenAt(10, 20, primitives.Box())
	window.setTitleButtonCount(2)

	controlsWidth := windowTitleButtonSize*2 + windowTitleButtonGap
	left := window.x + window.width - windowTitleButtonPadR - controlsWidth
	top := window.y + (window.titleHeight-windowTitleButtonSize)/2
	if !window.titleButtonHit(left, top) {
		t.Fatal("left title button was treated as draggable title bar")
	}
	if !window.titleButtonHit(left+controlsWidth-1, top+windowTitleButtonSize-1) {
		t.Fatal("right title button was treated as draggable title bar")
	}
	if window.titleButtonHit(left-1, top) {
		t.Fatal("title bar beside the buttons was treated as a title button")
	}
}

func TestWindowFullRedrawPublishIsIdempotent(t *testing.T) {
	manager := &escapeMenuTestUIManager{}
	ctx := client.Context{UIManager: manager, ScreenW: 800, ScreenH: 600}
	window := NewWindow(100, 80)
	window.SetFullRedraw(true)
	window.OpenAt(10, 20, primitives.Box())
	window.Publish(ctx)
	if len(manager.overlays) != 1 {
		t.Fatalf("published overlays = %d, want 1", len(manager.overlays))
	}

	root := manager.overlays[0]
	widget.ClearRedrawInTree(root)
	redraw, ok := root.(interface{ NeedsRedraw() bool })
	if !ok {
		t.Fatal("published root does not expose redraw state")
	}
	if redraw.NeedsRedraw() {
		t.Fatal("published root stayed dirty after clear")
	}

	window.Publish(ctx)
	if redraw.NeedsRedraw() {
		t.Fatal("unchanged full redraw publish dirtied the published root")
	}
	if manager.overlays[0] != root {
		t.Fatal("unchanged full redraw publish replaced the published root")
	}

	window.SetContent(primitives.Box())
	window.Publish(ctx)
	if manager.overlays[0] != root {
		t.Fatal("content change replaced the published root")
	}
	if !redraw.NeedsRedraw() {
		t.Fatal("content replacement did not dirty the published root")
	}
}

func TestWindowDragReusesPublishedOverlay(t *testing.T) {
	manager := &escapeMenuTestUIManager{}
	app := &windowDragTestApp{}
	inputState := input.NewState()
	ctx := client.Context{
		Input:     inputState,
		UIApp:     app,
		UIManager: manager,
		ScreenW:   800,
		ScreenH:   600,
	}
	content := newWindowDragEventRecorder()
	window := NewWindow(100, 80)
	window.OpenAt(10, 20, content)
	window.Publish(ctx)
	if len(manager.overlays) != 1 {
		t.Fatalf("published overlays = %d, want 1", len(manager.overlays))
	}
	root := manager.overlays[0]
	overlay := root.(*positionedOverlay)

	inputState.SetMousePosition(20, 25)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !window.Update(ctx) {
		t.Fatal("title press was not consumed")
	}
	window.Publish(ctx)
	if manager.overlays[0] != root {
		t.Fatal("drag start replaced the published overlay")
	}
	if app.beginToken != root {
		t.Fatal("drag start did not capture the published overlay")
	}
	if app.beginRect != geometry.NewRect(10, 20, 100, 80) {
		t.Fatalf("drag capture rect = %v, want x=10 y=20 w=100 h=80", app.beginRect)
	}
	if !overlay.hidden {
		t.Fatal("drag start did not hide the overlay from the UI canvas")
	}
	if children := overlay.Children(); len(children) != 0 {
		t.Fatalf("hidden overlay exposed %d dirty children", len(children))
	}
	mouse := event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, geometry.Pt(20, 25), geometry.Pt(20, 25), event.ModNone)
	if overlay.Event(widget.NewContext(), mouse) {
		t.Fatal("hidden overlay consumed a mouse event")
	}
	if content.events != 0 {
		t.Fatalf("hidden overlay forwarded %d events to content", content.events)
	}
	widget.ClearRedrawInTree(root)
	overlay.clearDamage()
	app.rects = nil

	inputState.EndFrame()
	inputState.SetMousePosition(50, 60)
	if !window.Update(ctx) {
		t.Fatal("drag move was not consumed")
	}
	window.Publish(ctx)
	if manager.overlays[0] != root {
		t.Fatal("drag move replaced the published overlay")
	}
	if window.x != 40 || window.y != 55 {
		t.Fatalf("window position = %d,%d; want 40,55", window.x, window.y)
	}
	if bounds := overlay.Bounds(); bounds != geometry.NewRect(40, 55, 100, 80) {
		t.Fatalf("overlay bounds = %v, want x=40 y=55 w=100 h=80", bounds)
	}
	if screenBounds := overlay.ScreenBounds(); screenBounds != geometry.NewRect(40, 55, 100, 80) {
		t.Fatalf("overlay screen bounds = %v, want x=40 y=55 w=100 h=80", screenBounds)
	}
	if len(app.moves) != 1 || app.moves[0] != geometry.NewRect(40, 55, 100, 80) {
		t.Fatalf("drag layer moves = %v, want final x=40 y=55 w=100 h=80", app.moves)
	}
	if overlay.NeedsRedraw() {
		t.Fatal("drag move dirtied the UI overlay")
	}
	if len(app.rects) != 0 {
		t.Fatalf("drag move invalidated UI rects: got %d calls, want 0", len(app.rects))
	}

	content.SetNeedsRedraw(true)
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)
	if !window.Update(ctx) {
		t.Fatal("drag release was not consumed")
	}
	if app.endToken != root {
		t.Fatal("drag release did not begin the drag-layer handoff")
	}
	if app.cancelToken != nil {
		t.Fatal("ordinary drag release cancelled the drag layer")
	}
	if overlay.hidden {
		t.Fatal("drag release did not restore the overlay")
	}
	if !overlay.NeedsRedraw() {
		t.Fatal("drag release did not redraw the window at its final position")
	}
	if !content.NeedsRedraw() {
		t.Fatal("drag release discarded content dirty state")
	}
	if len(app.rects) != 1 {
		t.Fatalf("drag release invalidates = %d, want 1", len(app.rects))
	}
	if last := app.rects[0]; last != geometry.NewRect(38, 53, 104, 84) {
		t.Fatalf("drag release dirty rect = %v, want x=38 y=53 w=104 h=84", last)
	}
	if children := overlay.Children(); len(children) != 0 {
		t.Fatalf("damaged overlay exposed %d dirty children before repaint", len(children))
	}
	overlay.clearDamage()
	if children := overlay.Children(); len(children) != 1 {
		t.Fatalf("clean overlay children = %d, want 1", len(children))
	}
}

func TestWindowCloseCancelsDragLayer(t *testing.T) {
	manager := &escapeMenuTestUIManager{}
	app := &windowDragTestApp{}
	inputState := input.NewState()
	ctx := client.Context{
		Input:     inputState,
		UIApp:     app,
		UIManager: manager,
		ScreenW:   800,
		ScreenH:   600,
	}
	window := NewWindow(100, 80)
	window.OpenAt(10, 20, primitives.Box())
	window.Publish(ctx)
	root := manager.overlays[0]

	inputState.SetMousePosition(20, 25)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	window.Update(ctx)
	window.Close()

	if app.cancelToken != root {
		t.Fatal("closing a dragged window did not cancel its drag layer")
	}
	if app.endToken != nil {
		t.Fatal("closing a dragged window started a release handoff")
	}
	if window.dragLayer || window.dragging {
		t.Fatal("closed window retained drag state")
	}
}

func TestWindowDragContinuesAcrossEarlierUpdatedWindow(t *testing.T) {
	app := &windowDragTestApp{}
	inputState := input.NewState()
	ctx := client.Context{
		Input:   inputState,
		UIApp:   app,
		ScreenW: 800,
		ScreenH: 600,
	}
	dragged := NewWindow(100, 80)
	dragged.OpenAt(10, 20, primitives.Box())
	blocker := NewWindow(120, 90)
	blocker.OpenAt(50, 50, primitives.Box())

	inputState.SetMousePosition(20, 25)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !dragged.Update(ctx) {
		t.Fatal("drag start was not consumed")
	}

	inputState.EndFrame()
	inputState.SetMousePosition(80, 65) // Inside the other window.
	if blocker.Update(ctx) {
		t.Fatal("overlapped window consumed a frame owned by the active drag")
	}
	if !dragged.Update(ctx) {
		t.Fatal("active window did not continue dragging across overlap")
	}
	if dragged.x != 70 || dragged.y != 60 {
		t.Fatalf("dragged window position = %d,%d; want 70,60", dragged.x, dragged.y)
	}

	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)
	if !dragged.Update(ctx) || app.WindowDragActive() {
		t.Fatal("drag release did not relinquish shared drag capture")
	}
}

func TestDamagedPositionedOverlayClearsPreexistingChildDirty(t *testing.T) {
	child := newWindowDragEventRecorder()
	overlay := positionedWidget(child, 10, 20, 100, 80).(*positionedOverlay)
	child.SetNeedsRedraw(true)
	overlay.markFrameDirty()

	overlay.Draw(widget.NewContext(), &uitest.MockCanvas{})

	if child.NeedsRedraw() {
		t.Fatal("damaged overlay left preexisting child redraw dirty for another frame")
	}
}

func TestDamagedPositionedOverlayKeepsChildDirtyRaisedDuringDraw(t *testing.T) {
	child := newWindowDragEventRecorder()
	child.dirtyDuringDraw = true
	overlay := positionedWidget(child, 10, 20, 100, 80).(*positionedOverlay)
	child.SetNeedsRedraw(true)
	overlay.markFrameDirty()

	overlay.Draw(widget.NewContext(), &uitest.MockCanvas{})

	if !child.NeedsRedraw() {
		t.Fatal("damaged overlay cleared redraw raised by child Draw")
	}
}

func TestScreenEdgeAnchorsUseWindowMargin(t *testing.T) {
	ctx := client.Context{ScreenW: 800, ScreenH: 600}

	if characterWindowX != windowScreenMargin || characterWindowY != windowScreenMargin {
		t.Fatalf("character window position = %d,%d; want %d,%d", characterWindowX, characterWindowY, windowScreenMargin, windowScreenMargin)
	}
	if x, y, _, _ := basicMenuBounds(); x != windowScreenMargin || y != windowScreenMargin+characterWindowHeight+6 {
		t.Fatalf("basic menu position = %d,%d; want x=%d y=%d", x, y, windowScreenMargin, windowScreenMargin+characterWindowHeight+6)
	}
	if x, y, _, _ := MinimapBounds(ctx.ScreenW, ctx.ScreenH); x != ctx.ScreenW-minimapWidth-windowScreenMargin || y != windowScreenMargin {
		t.Fatalf("minimap position = %d,%d; want x=%d y=%d", x, y, ctx.ScreenW-minimapWidth-windowScreenMargin, windowScreenMargin)
	}
	if x, y, _, _ := consoleBounds(ctx.ScreenW, ctx.ScreenH); x != windowScreenMargin || y != ctx.ScreenH-consoleHeight-windowScreenMargin {
		t.Fatalf("console position = %d,%d; want x=%d y=%d", x, y, windowScreenMargin, ctx.ScreenH-consoleHeight-windowScreenMargin)
	}
	if x, _ := storageDefaultPosition(ctx); x != ctx.ScreenW-storageWindowWidth-windowScreenMargin {
		t.Fatalf("storage x = %d; want %d", x, ctx.ScreenW-storageWindowWidth-windowScreenMargin)
	}
	if x, _ := cartDefaultPosition(ctx); x != ctx.ScreenW-cartWindowWidth-windowScreenMargin {
		t.Fatalf("cart x = %d; want %d", x, ctx.ScreenW-cartWindowWidth-windowScreenMargin)
	}
}

type windowDragTestApp struct {
	rects       []geometry.Rect
	beginToken  any
	beginRect   geometry.Rect
	moves       []geometry.Rect
	endToken    any
	cancelToken any
	active      bool
}

func (a *windowDragTestApp) SetUIRoot(widget.Widget) {}
func (a *windowDragTestApp) Frame()                  {}
func (a *windowDragTestApp) Invalidate()             {}
func (a *windowDragTestApp) InvalidateRect(rect geometry.Rect) {
	a.rects = append(a.rects, rect)
}
func (a *windowDragTestApp) Cursor() widget.CursorType {
	return widget.CursorDefault
}
func (a *windowDragTestApp) HoveredWidget() widget.Widget {
	return nil
}
func (a *windowDragTestApp) BeginWindowDragLayer(token any, rect geometry.Rect) bool {
	a.beginToken = token
	a.beginRect = rect
	a.active = true
	return true
}
func (a *windowDragTestApp) MoveWindowDragLayer(token any, rect geometry.Rect) {
	a.moves = append(a.moves, rect)
}
func (a *windowDragTestApp) EndWindowDragLayer(token any) {
	a.endToken = token
	a.active = false
}
func (a *windowDragTestApp) CancelWindowDragLayer(token any) {
	a.cancelToken = token
	a.active = false
}
func (a *windowDragTestApp) WindowDragActive() bool { return a.active }

type windowDragEventRecorder struct {
	widget.WidgetBase
	events          int
	dirtyDuringDraw bool
}

func newWindowDragEventRecorder() *windowDragEventRecorder {
	w := &windowDragEventRecorder{}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *windowDragEventRecorder) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(10, 10))
	w.SetBounds(geometry.FromPointSize(geometry.Point{}, size))
	return size
}

func (w *windowDragEventRecorder) Draw(widget.Context, widget.Canvas) {
	if w.dirtyDuringDraw {
		w.SetNeedsRedraw(true)
	}
}

func (w *windowDragEventRecorder) Event(widget.Context, event.Event) bool {
	w.events++
	return true
}
