package rotheme

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

const ButtonRadius float32 = 6

func Button(label string, onClick func()) *button.Widget {
	opts := []button.Option{
		button.TextOpt(label),
		button.SizeOpt(button.Small),
		button.PainterOpt(ButtonPainter{}),
		button.RoundedOpt(ButtonRadius),
	}
	if onClick != nil {
		opts = append(opts, button.OnClick(onClick))
	}
	return button.New(opts...).PaddingXY(8, 1)
}

type ButtonPainter struct{}

func (ButtonPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}
	bg := Default.Colors.Button
	if state.Background != nil {
		bg = *state.Background
	}
	if state.Hovered {
		bg = Default.Colors.ButtonHover
	}
	if state.Pressed {
		bg = Default.Colors.ButtonDown
	}
	if state.Disabled {
		bg = Default.Colors.Disabled
	}
	radius := ButtonRadius
	if state.Radius != nil {
		radius = *state.Radius
	}
	drawButtonGradient(canvas, state.Bounds, bg, radius)
	canvas.StrokeRoundRect(state.Bounds, Default.Colors.ButtonBorder, radius, 1)

	text := Default.Colors.Text
	if state.Disabled {
		text = Default.Colors.MutedText
	}
	DrawText(canvas, state.Text, state.Bounds, Default.Typography.TextSize, text, false, widget.TextAlignCenter)
}

func drawButtonGradient(canvas widget.Canvas, bounds geometry.Rect, bottom widget.Color, radius float32) {
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
