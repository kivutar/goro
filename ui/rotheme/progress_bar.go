package rotheme

import (
	"github.com/gogpu/ui/core/progressbar"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

const ProgressBarHeight float32 = 12

// ProgressBar uses the button gradient and reflection, with room for a label.
func ProgressBar(opts ...progressbar.Option) *progressbar.Widget {
	opts = append([]progressbar.Option{
		progressbar.Height(ProgressBarHeight),
		progressbar.Radius(ButtonRadius),
		progressbar.PainterOpt(ProgressBarPainter{}),
	}, opts...)
	return progressbar.New(opts...).Padding(0)
}

// VitalBar uses the classic HP/SP warning color below 25%.
// HighPriest's UIBarGraphPlayerHp/Sp and roBrowser use the same threshold.
func VitalBar(opts ...progressbar.Option) *progressbar.Widget {
	opts = append(opts, progressbar.PainterOpt(ProgressBarPainter{LowThreshold: 0.25}))
	return ProgressBar(opts...)
}

type ProgressBarPainter struct {
	LowThreshold float64 // Zero disables the low-value warning.
}

func (p ProgressBarPainter) PaintProgressBar(canvas widget.Canvas, state progressbar.PaintState) {
	height := min(state.BarHeight, state.Bounds.Height())
	if state.Bounds.IsEmpty() || height <= 0 {
		return
	}
	bounds := geometry.NewRect(state.Bounds.Min.X, state.Bounds.Center().Y-height/2, state.Bounds.Width(), height)
	radius := min(max(float32(0), state.Radius), height/2)
	fill, border, text := lighterColor(Default.Colors.WindowBorder, 2), Default.Colors.ButtonBorder, Default.Colors.Text
	if state.Value > 0 && state.Value < p.LowThreshold {
		fill = lighterColor(widget.Hex(0xCE768A), 2)
		border = widget.Hex(0xD68AA2)
	}
	if state.Disabled {
		fill, border, text = Default.Colors.Disabled, Default.Colors.FooterLine, Default.Colors.MutedText
	}

	drawButtonShadow(canvas, bounds, radius)
	canvas.DrawRoundRect(bounds, Default.Colors.WindowFooter, radius)
	canvas.PushClipRoundRect(bounds, radius)
	if state.Value > 0 {
		canvas.PushClip(geometry.NewRect(bounds.Min.X, bounds.Min.Y, bounds.Width()*float32(state.Value), height))
		// Keep the gradient image the full bar width so changing the value
		// reuses the same cached image instead of allocating one per fill width.
		DrawVerticalGradient(canvas, bounds, fill, buttonGradientLight(fill))
		canvas.PopClip()
	}
	drawButtonReflect(canvas, bounds, radius)
	canvas.PopClip()
	canvas.StrokeRoundRect(bounds, border, radius, 1)
	if state.ShowLabel {
		DrawText(canvas, state.Label, bounds, Default.Typography.TextSize, text, false, widget.TextAlignCenter)
	}
}

var _ progressbar.Painter = ProgressBarPainter{}
