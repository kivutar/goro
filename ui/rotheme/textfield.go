package rotheme

import (
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

func TextField(value string, inputType textfield.InputType, onChange func(string), onSubmit func(string), extra ...textfield.Option) *textfield.Widget {
	opts := []textfield.Option{
		textfield.InitialValue(value),
		textfield.InputTypeOpt(inputType),
		textfield.OnChange(onChange),
		textfield.OnSubmit(onSubmit),
		textfield.PainterOpt(TextFieldPainter{}),
	}
	opts = append(opts, extra...)
	return textfield.New(opts...)
}

type TextFieldPainter struct{}

func (TextFieldPainter) PaintTextField(canvas widget.Canvas, state textfield.PaintState) {
	border := Default.Colors.InputBorder
	if state.Focused {
		border = Default.Colors.InputFocus
	}
	canvas.DrawRect(state.Bounds, Default.Colors.PanelBody)
	canvas.StrokeRect(state.Bounds, border, 1)

	content := geometry.NewRect(state.Bounds.Min.X+6, state.Bounds.Min.Y+4, state.Bounds.Width()-12, state.Bounds.Height()-8)
	text := state.Text
	if state.InputType == textfield.TypePassword {
		text = strings.Repeat("*", len([]rune(text)))
	}
	DrawText(canvas, text, content, Default.Typography.TextSize, Default.Colors.Text, false, widget.TextAlignLeft)
	if state.Focused && state.SelectStart == state.SelectEnd {
		runes := []rune(text)
		cursor := state.CursorPos
		if cursor > len(runes) {
			cursor = len(runes)
		}
		x := min(content.Max.X, content.Min.X+MeasureText(canvas, string(runes[:cursor]), Default.Typography.TextSize, false))
		canvas.DrawLine(geometry.Pt(x, state.Bounds.Min.Y+4), geometry.Pt(x, state.Bounds.Max.Y-4), Default.Colors.Text, 1)
	}
}
