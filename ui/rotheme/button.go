package rotheme

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

const (
	ButtonRadius        float32 = 6
	ButtonPaddingX      float32 = 8
	ButtonPaddingY      float32 = 5.5
	LargeButtonPaddingY float32 = 8.5
)

func Button(label string, onClick func()) *primitives.BoxWidget {
	return ButtonDisabled(label, false, onClick)
}

func ButtonDisabled(label string, disabled bool, onClick func()) *primitives.BoxWidget {
	return buttonWithPadding(label, disabled, ButtonPaddingY, onClick)
}

func LargeButton(label string, onClick func()) *primitives.BoxWidget {
	return LargeButtonDisabled(label, false, onClick)
}

func LargeButtonDisabled(label string, disabled bool, onClick func()) *primitives.BoxWidget {
	return buttonWithPadding(label, disabled, LargeButtonPaddingY, onClick)
}

func buttonWithPadding(label string, disabled bool, paddingY float32, onClick func()) *primitives.BoxWidget {
	opts := []button.Option{
		button.TextOpt(label),
		button.SizeOpt(button.Small),
		button.PainterOpt(ButtonPainter{}),
		button.RoundedOpt(ButtonRadius),
		button.Disabled(disabled),
	}
	if onClick != nil {
		opts = append(opts, button.OnClick(onClick))
	}
	return primitives.Box(
		button.New(opts...).PaddingXY(ButtonPaddingX, paddingY),
	).
		CrossAlign(primitives.CrossAxisStretch).
		Height(Default.Typography.TextSize + paddingY*2)
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
	if radius > 0 {
		canvas.PushClipRoundRect(bounds, radius)
		defer canvas.PopClip()
	}
	for i := 0; i < height; i++ {
		t := float32(i) / float32(height-1)
		y := bounds.Min.Y + float32(i)
		canvas.DrawRect(geometry.NewRect(bounds.Min.X, y, bounds.Width(), 1), top.Lerp(bottom, t))
	}
}
