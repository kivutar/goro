package ui

import (
	"image/color"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	roUIButtonHeight = 22
	roUIButtonRadius = 3
)

type roButtonWidget struct {
	widget.WidgetBase
	inner *button.Widget
}

type roButtonPainter struct{}

func (roButtonPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}
	bg := uiColor(ButtonColor)
	if state.Background != nil {
		bg = *state.Background
	}
	if state.Hovered {
		bg = uiColor(ButtonHoverColor)
	}
	if state.Pressed {
		bg = uiColor(ButtonDownColor)
	}
	if state.Disabled {
		bg = uiColor(DisabledColor)
	}
	radius := float32(roUIButtonRadius)
	if state.Radius != nil {
		radius = *state.Radius
	}
	drawRoundedButtonGradient(canvas, state.Bounds, bg, radius)
	canvas.StrokeRoundRect(state.Bounds, uiColor(ButtonBorderColor), radius, 1)
	text := uiColor(TextColor)
	if state.Disabled {
		text = uiColor(MutedTextColor)
	}
	rotheme.DrawText(canvas, state.Text, state.Bounds, rotheme.Default.Typography.TextSize, text, false, widget.TextAlignCenter)
}

func drawRoundedButtonGradient(canvas widget.Canvas, bounds geometry.Rect, bottom widget.Color, radius float32) {
	top := bottom.Lerp(widget.RGBA(1, 1, 1, bottom.A), 0.42)
	height := int(bounds.Height())
	if height <= 1 {
		canvas.DrawRoundRect(bounds, bottom, radius)
		return
	}
	for i := 0; i < height; i++ {
		t := float32(i) / float32(height-1)
		y := bounds.Min.Y + float32(i)
		canvas.DrawRect(geometry.NewRect(bounds.Min.X, y, bounds.Width(), 1), top.Lerp(bottom, t))
	}
}

func roButton(label string) *roButtonWidget {
	inner := button.New(
		button.TextOpt(label),
		button.SizeOpt(button.Small),
		button.PainterOpt(roButtonPainter{}),
		button.RoundedOpt(roUIButtonRadius),
	).PaddingXY(8, 1)
	w := &roButtonWidget{inner: inner}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *roButtonWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := w.inner.Layout(ctx, constraints)
	size.Height = roUIButtonHeight
	size = constraints.Constrain(size)
	w.inner.SetBounds(geometry.FromPointSize(w.Position(), size))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *roButtonWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	w.inner.SetBounds(w.Bounds())
	w.inner.Draw(ctx, canvas)
}

func (w *roButtonWidget) Event(ctx widget.Context, e event.Event) bool {
	return w.inner.Event(ctx, e)
}

func (w *roButtonWidget) Children() []widget.Widget {
	return nil
}

func (w *roButtonWidget) IsFocusable() bool {
	return w.inner.IsFocusable()
}

func (w *roButtonWidget) SetFocused(focused bool) {
	w.WidgetBase.SetFocused(focused)
	w.inner.SetFocused(focused)
}

func (w *roButtonWidget) IsFocused() bool {
	return w.inner.IsFocused()
}

func (w *roButtonWidget) Mount(ctx widget.Context) {
	w.inner.Mount(ctx)
}

func (w *roButtonWidget) Unmount() {
	w.inner.Unmount()
}

func (w *roButtonWidget) PaddingXY(x, y float32) *roButtonWidget {
	w.inner.PaddingXY(x, y)
	return w
}

func (w *roButtonWidget) SetBackground(c widget.Color) *roButtonWidget {
	w.inner.SetBackground(c)
	return w
}

func (w *roButtonWidget) MinWidth(v float32) *roButtonWidget {
	w.inner.MinWidth(v)
	return w
}

func uiColor(c color.RGBA) widget.Color {
	return widget.RGBA8(c.R, c.G, c.B, c.A)
}
