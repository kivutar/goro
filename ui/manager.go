package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
)

type Manager struct {
	app        client.UIApp
	root       *overlayRoot
	overlays   []widget.Widget
	foreground []widget.Widget
}

func NewManager() *Manager {
	return &Manager{root: newOverlayRoot(nil)}
}

func (m *Manager) SetUIApp(app client.UIApp) {
	if m == nil || m.app == app {
		return
	}
	m.app = app
	m.apply()
}

func (m *Manager) AddOverlay(root widget.Widget) {
	if m == nil || root == nil {
		return
	}
	for _, child := range m.overlays {
		if child == root {
			return
		}
	}
	disableRootRepaintBoundary(root)
	insert := len(m.overlays) - len(m.foreground)
	m.overlays = append(m.overlays, nil)
	copy(m.overlays[insert+1:], m.overlays[insert:])
	m.overlays[insert] = root
	m.apply()
}

// AddForegroundOverlay publishes an overlay above ordinary windows. New and
// raised windows remain below it until the foreground overlay is removed.
func (m *Manager) AddForegroundOverlay(root widget.Widget) {
	if m == nil || root == nil {
		return
	}
	for _, child := range m.overlays {
		if child == root {
			return
		}
	}
	disableRootRepaintBoundary(root)
	m.overlays = append(m.overlays, root)
	m.foreground = append(m.foreground, root)
	m.apply()
}

func (m *Manager) RemoveOverlay(root widget.Widget) {
	if m == nil || root == nil {
		return
	}
	for i, child := range m.overlays {
		if child == root {
			m.overlays = append(m.overlays[:i], m.overlays[i+1:]...)
			for j, foreground := range m.foreground {
				if foreground == root {
					m.foreground = append(m.foreground[:j], m.foreground[j+1:]...)
					break
				}
			}
			m.apply()
			return
		}
	}
}

func (m *Manager) Clear() {
	if m == nil || len(m.overlays) == 0 {
		return
	}
	m.overlays = nil
	m.foreground = nil
	m.apply()
}

func (m *Manager) PointerBlocked(x, y int) bool {
	return m != nil && m.root != nil && m.root.PointerBlocked(geometry.Pt(float32(x), float32(y)))
}

// OverlayAt returns the same topmost hit target used for pointer dispatch.
// Drag-and-drop destinations use it to avoid accepting items through a window.
func (m *Manager) OverlayAt(x, y int) widget.Widget {
	if m == nil || m.root == nil {
		return nil
	}
	position := geometry.Pt(float32(x), float32(y))
	for i := len(m.root.children) - 1; i >= 0; i-- {
		if child := m.root.children[i]; widgetCoversPoint(child, position) {
			return child
		}
	}
	return nil
}

func (m *Manager) RaiseOverlay(root widget.Widget) {
	m.raiseOverlay(root)
}

func (m *Manager) apply() {
	m.root = newOverlayRoot(m.overlays)
	m.root.onActivate = m.raiseOverlay
	if m.app != nil && m.root != nil {
		m.app.SetUIRoot(m.root)
		disableRootRepaintBoundary(m.root)
		m.root.SetNeedsRedraw(true)
	}
}

func (m *Manager) raiseOverlay(overlay widget.Widget) {
	if m == nil || overlay == nil || len(m.overlays) < 2 {
		return
	}
	top := len(m.overlays) - len(m.foreground) - 1
	if top < 0 {
		return
	}
	for i, child := range m.overlays {
		if child != overlay || i >= top {
			continue
		}
		copy(m.overlays[i:], m.overlays[i+1:top+1])
		m.overlays[top] = overlay
		if m.root != nil {
			// Keep the active root so pointer capture and keyboard focus survive.
			m.root.children = append(m.root.children[:0], m.overlays...)
			m.root.SetNeedsRedraw(true)
		}
		return
	}
}

func disableRootRepaintBoundary(root widget.Widget) {
	type boundarySetter interface{ SetRepaintBoundary(bool) }
	if rb, ok := root.(boundarySetter); ok {
		// The renderer already caches the complete UI image between dirty frames.
		// Drawing the root directly keeps rounded clipping on the normal canvas;
		// gogpu/ui's scene recorder currently degrades rounded clips to rectangles.
		rb.SetRepaintBoundary(false)
	}
}

type overlayRoot struct {
	widget.WidgetBase
	children   []widget.Widget
	onActivate func(widget.Widget)
}

func newOverlayRoot(children []widget.Widget) *overlayRoot {
	root := &overlayRoot{children: append([]widget.Widget(nil), children...)}
	root.SetVisible(true)
	root.SetEnabled(true)
	root.SetNeedsRedraw(true)
	return root
}

func (r *overlayRoot) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Biggest()
	if size.Width <= 0 || size.Height <= 0 {
		size = constraints.Constrain(geometry.Sz(1, 1))
	}
	r.SetBounds(geometry.FromPointSize(r.Position(), size))
	for _, child := range r.children {
		child.Layout(ctx, geometry.Loose(size))
	}
	return size
}

func (r *overlayRoot) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !r.IsVisible() {
		return
	}
	clip := canvas.ClipBounds()
	for _, child := range r.children {
		if !widgetIntersectsRect(child, clip) {
			continue
		}
		widget.StampScreenOrigin(child, canvas)
		widget.DrawChild(child, ctx, canvas)
	}
}

func (r *overlayRoot) Event(ctx widget.Context, e event.Event) bool {
	if !r.IsVisible() || !r.IsEnabled() {
		return false
	}
	if mouse, ok := e.(*event.MouseEvent); ok {
		return r.dispatchPositionedEvent(ctx, e, mouse.Position)
	}
	if wheel, ok := e.(*event.WheelEvent); ok {
		return r.dispatchPositionedEvent(ctx, e, wheel.Position)
	}
	for i := len(r.children) - 1; i >= 0; i-- {
		if r.children[i].Event(ctx, e) {
			return true
		}
	}
	return false
}

func (r *overlayRoot) dispatchPositionedEvent(ctx widget.Context, e event.Event, position geometry.Point) bool {
	for i := len(r.children) - 1; i >= 0; i-- {
		child := r.children[i]
		if !widgetCoversPoint(child, position) {
			continue
		}
		if mouse, ok := e.(*event.MouseEvent); ok && mouse.IsPress() && overlayRaisesOnPress(child) && r.onActivate != nil {
			r.onActivate(child)
		}
		child.Event(ctx, e)
		return true
	}
	return false
}

func overlayRaisesOnPress(overlay widget.Widget) bool {
	positioned, ok := overlay.(*positionedOverlay)
	return ok && positioned.raiseOnPress
}

func (r *overlayRoot) PointerBlocked(position geometry.Point) bool {
	if r == nil || !r.IsVisible() || !r.IsEnabled() {
		return false
	}
	for i := len(r.children) - 1; i >= 0; i-- {
		if widgetCoversPoint(r.children[i], position) {
			return true
		}
	}
	return false
}

// IsUIRootEmpty lets the render bridge discard a previously published UI
// image immediately when the last overlay is removed. This avoids retaining a
// stale asynchronous frame while the empty root is rasterized.
func (r *overlayRoot) IsUIRootEmpty() bool {
	return r == nil || len(r.children) == 0
}

func widgetCoversPoint(child widget.Widget, position geometry.Point) bool {
	if child == nil {
		return false
	}
	if visible, ok := child.(interface{ IsVisible() bool }); ok && !visible.IsVisible() {
		return false
	}
	if enabled, ok := child.(interface{ IsEnabled() bool }); ok && !enabled.IsEnabled() {
		return false
	}
	if bounds, ok := child.(interface{ Bounds() geometry.Rect }); ok {
		return bounds.Bounds().Contains(position)
	}
	if box, ok := child.(*primitives.BoxWidget); ok {
		return box.Bounds().Contains(position)
	}
	return false
}

func widgetIntersectsRect(child widget.Widget, rect geometry.Rect) bool {
	if child == nil {
		return false
	}
	if rect.IsEmpty() {
		return false
	}
	if visible, ok := child.(interface{ IsVisible() bool }); ok && !visible.IsVisible() {
		return false
	}
	if bounds, ok := child.(interface{ Bounds() geometry.Rect }); ok {
		return bounds.Bounds().Intersects(rect)
	}
	if box, ok := child.(*primitives.BoxWidget); ok {
		return box.Bounds().Intersects(rect)
	}
	return true
}

func (r *overlayRoot) Children() []widget.Widget {
	if len(r.children) == 0 {
		return nil
	}
	children := make([]widget.Widget, len(r.children))
	copy(children, r.children)
	return children
}
