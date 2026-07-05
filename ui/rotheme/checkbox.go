package rotheme

import (
	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

const (
	CheckboxSize   float32 = 17
	CheckboxRadius float32 = 6
	CheckboxGap    float32 = 8
)

func Checkbox(opts ...checkbox.Option) *checkbox.Widget {
	opts = append(opts, checkbox.PainterOpt(CheckboxPainter{}))
	return checkbox.New(opts...).Padding(0)
}

type CheckboxPainter struct{}

func (CheckboxPainter) PaintCheckbox(canvas widget.Canvas, state checkbox.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}

	box := checkboxBox(state.Bounds)
	bg := Default.Colors.Button
	border := Default.Colors.ButtonBorder
	text := Default.Colors.Text
	if state.Hovered {
		bg = Default.Colors.ButtonHover
	}
	if state.Pressed {
		bg = Default.Colors.ButtonDown
	}
	if state.Disabled {
		bg = Default.Colors.Disabled
		text = Default.Colors.MutedText
		border = Default.Colors.Disabled
	}
	if state.Background != nil {
		bg = *state.Background
	}

	drawButtonGradient(canvas, box, bg, CheckboxRadius)
	canvas.StrokeRoundRect(box, border, CheckboxRadius, 1)

	if state.Checked {
		drawCheckboxV(canvas, box, text)
	} else if state.Indeterminate {
		drawCheckboxDash(canvas, box, text)
	}
	if state.Focused && !state.Disabled {
		canvas.StrokeRoundRect(box, Default.Colors.InputFocus, CheckboxRadius, 1)
	}
	if state.Label != "" {
		DrawText(canvas, state.Label, checkboxLabel(state.Bounds), Default.Typography.TextSize, text, false, widget.TextAlignLeft)
	}
}

func checkboxBox(bounds geometry.Rect) geometry.Rect {
	y := bounds.Min.Y + (bounds.Height()-CheckboxSize)/2
	return geometry.NewRect(bounds.Min.X, y, CheckboxSize, CheckboxSize)
}

func checkboxLabel(bounds geometry.Rect) geometry.Rect {
	x := bounds.Min.X + CheckboxSize + CheckboxGap
	return geometry.NewRect(x, bounds.Min.Y, bounds.Width()-(x-bounds.Min.X), bounds.Height())
}

func drawCheckboxV(canvas widget.Canvas, box geometry.Rect, color widget.Color) {
	x := box.Min.X
	y := box.Min.Y
	canvas.DrawLine(geometry.Pt(x+4, y+6), geometry.Pt(x+8, y+11), color, 1)
	canvas.DrawLine(geometry.Pt(x+8, y+11), geometry.Pt(x+13, y+5), color, 1)
}

func drawCheckboxDash(canvas widget.Canvas, box geometry.Rect, color widget.Color) {
	y := box.Min.Y + box.Height()/2
	canvas.DrawLine(geometry.Pt(box.Min.X+4, y), geometry.Pt(box.Max.X-4, y), color, 1)
}

var _ checkbox.Painter = CheckboxPainter{}
