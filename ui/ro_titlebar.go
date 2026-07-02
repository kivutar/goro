package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

type roTitleBarWidget struct {
	widget.WidgetBase
	child  widget.Widget
	height float32
}

func roTitleBar(child widget.Widget, height float32) *roTitleBarWidget {
	w := &roTitleBarWidget{
		child:  child,
		height: height,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *roTitleBarWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	if w.height > 0 {
		constraints = constraints.TightenHeight(w.height)
	}
	size := w.child.Layout(ctx, constraints)
	if w.height > 0 {
		size.Height = w.height
	}
	size = constraints.Constrain(size)
	w.child.(interface{ SetBounds(geometry.Rect) }).SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *roTitleBarWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	drawTitleBarGradient(canvas, bounds)

	canvas.PushTransform(bounds.Min)
	widget.StampScreenOrigin(w.child, canvas)
	widget.DrawChild(w.child, ctx, canvas)
	canvas.PopTransform()
}

func (w *roTitleBarWidget) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	switch ev := e.(type) {
	case *event.MouseEvent:
		local := *ev
		local.Position = ev.Position.Sub(w.Bounds().Min)
		return w.child.Event(ctx, &local)
	case *event.WheelEvent:
		local := *ev
		local.Position = ev.Position.Sub(w.Bounds().Min)
		return w.child.Event(ctx, &local)
	default:
		return w.child.Event(ctx, e)
	}
}

func (w *roTitleBarWidget) Children() []widget.Widget {
	return []widget.Widget{w.child}
}

func drawTitleBarGradient(canvas widget.Canvas, bounds geometry.Rect) {
	if bounds.IsEmpty() {
		return
	}
	top := rotheme.Default.Colors.WindowTitleTop
	bottom := rotheme.Default.Colors.WindowTitle
	height := int(bounds.Height())
	if height <= 1 {
		canvas.DrawRect(bounds, bottom)
		return
	}
	for i := 0; i < height; i++ {
		t := float32(i) / float32(height-1)
		y := bounds.Min.Y + float32(i)
		canvas.DrawRect(geometry.NewRect(bounds.Min.X, y, bounds.Width(), 1), top.Lerp(bottom, t))
	}
}
