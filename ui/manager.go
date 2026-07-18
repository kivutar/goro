package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
)

type Manager struct {
	app      client.UIApp
	root     *overlayRoot
	overlays []widget.Widget
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
	m.overlays = append(m.overlays, root)
	m.apply()
}

func (m *Manager) RemoveOverlay(root widget.Widget) {
	if m == nil || root == nil {
		return
	}
	for i, child := range m.overlays {
		if child == root {
			m.overlays = append(m.overlays[:i], m.overlays[i+1:]...)
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
	m.apply()
}

func (m *Manager) PointerBlocked(x, y int) bool {
	return m != nil && m.root != nil && m.root.PointerBlocked(geometry.Pt(float32(x), float32(y)))
}

func (m *Manager) apply() {
	m.root = newOverlayRoot(m.overlays)
	if m.app != nil && m.root != nil {
		m.app.SetUIRoot(m.root)
		disableRootRepaintBoundary(m.root)
		m.root.SetNeedsRedraw(true)
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
	children []widget.Widget
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
	for _, child := range r.children {
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
		child.Event(ctx, e)
		return true
	}
	return false
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

func (r *overlayRoot) Children() []widget.Widget {
	if len(r.children) == 0 {
		return nil
	}
	children := make([]widget.Widget, len(r.children))
	copy(children, r.children)
	return children
}
