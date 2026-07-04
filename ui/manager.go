package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
)

type Manager struct {
	app  client.UIApp
	root *overlayRoot
}

func NewManager() *Manager {
	return &Manager{root: newOverlayRoot()}
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
	m.root.Add(root)
	disableRootRepaintBoundary(root)
	m.apply()
}

func (m *Manager) RemoveOverlay(root widget.Widget) {
	if m == nil || root == nil {
		return
	}
	m.root.Remove(root)
	m.apply()
}

func (m *Manager) Clear() {
	if m == nil {
		return
	}
	m.root.Clear()
	m.apply()
}

func (m *Manager) apply() {
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

func newOverlayRoot() *overlayRoot {
	root := &overlayRoot{}
	root.SetVisible(true)
	root.SetEnabled(true)
	return root
}

func (r *overlayRoot) Add(root widget.Widget) {
	if root == nil {
		return
	}
	for _, child := range r.children {
		if child == root {
			return
		}
	}
	r.children = append(r.children, root)
	r.SetNeedsRedraw(true)
}

func (r *overlayRoot) Remove(root widget.Widget) {
	for i, child := range r.children {
		if child == root {
			r.children = append(r.children[:i], r.children[i+1:]...)
			r.SetNeedsRedraw(true)
			return
		}
	}
}

func (r *overlayRoot) Clear() {
	if len(r.children) == 0 {
		return
	}
	r.children = nil
	r.SetNeedsRedraw(true)
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
		if bounds, ok := child.(interface{ Bounds() geometry.Rect }); ok && !bounds.Bounds().Contains(position) {
			continue
		}
		if child.Event(ctx, e) {
			return true
		}
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
