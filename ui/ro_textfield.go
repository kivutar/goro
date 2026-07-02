package ui

import (
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

type roTextFieldWidget struct {
	widget.WidgetBase
	inner *textfield.Widget
	width float32
}

type roTextFieldPainter struct{}

func (roTextFieldPainter) PaintTextField(canvas widget.Canvas, state textfield.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}
	border := uiColor(ButtonBorderColor)
	if state.Focused {
		border = uiColor(SelectionBorder)
	}
	canvas.DrawRect(state.Bounds, rotheme.Default.Colors.WindowBody)
	canvas.StrokeRect(state.Bounds, border, 1)

	content := state.Bounds
	content.Min.X += 6
	content.Max.X -= 6
	displayText := state.Text
	if state.InputType == textfield.TypePassword {
		displayText = strings.Repeat("*", len([]rune(state.Text)))
	}
	canvas.PushClip(content)
	canvas.DrawText(displayText, content, rotheme.Default.Typography.TextSize, uiColor(TextColor), false, widget.TextAlignLeft)
	if state.Focused {
		cursorX := content.Min.X + canvas.MeasureText(displayText, rotheme.Default.Typography.TextSize, false)
		if cursorX > content.Max.X {
			cursorX = content.Max.X
		}
		canvas.DrawLine(
			geometry.Pt(cursorX, state.Bounds.Min.Y+4),
			geometry.Pt(cursorX, state.Bounds.Max.Y-4),
			uiColor(TextColor),
			1,
		)
	}
	canvas.PopClip()
}

func roTextField(value string, inputType textfield.InputType, focused bool) *roTextFieldWidget {
	inner := textfield.New(
		textfield.InitialValue(value),
		textfield.InputTypeOpt(inputType),
		textfield.PainterOpt(roTextFieldPainter{}),
	)
	inner.SetFocused(focused)
	w := &roTextFieldWidget{inner: inner}
	w.SetVisible(true)
	w.SetEnabled(true)
	w.SetFocused(focused)
	return w
}

func (w *roTextFieldWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := w.inner.Layout(ctx, constraints)
	if w.width > 0 {
		size.Width = w.width
	}
	size.Height = constraints.ConstrainHeight(roUITextFieldHeight)
	size = constraints.Constrain(size)
	w.inner.SetBounds(geometry.FromPointSize(w.Position(), size))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *roTextFieldWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	w.inner.SetBounds(w.Bounds())
	w.inner.Draw(ctx, canvas)
}

func (w *roTextFieldWidget) Event(ctx widget.Context, e event.Event) bool {
	return w.inner.Event(ctx, e)
}

func (w *roTextFieldWidget) Children() []widget.Widget {
	return nil
}

func (w *roTextFieldWidget) IsFocusable() bool {
	return w.inner.IsFocusable()
}

func (w *roTextFieldWidget) SetFocused(focused bool) {
	w.WidgetBase.SetFocused(focused)
	w.inner.SetFocused(focused)
}

func (w *roTextFieldWidget) IsFocused() bool {
	return w.inner.IsFocused()
}

func (w *roTextFieldWidget) Mount(ctx widget.Context) {
	w.inner.Mount(ctx)
}

func (w *roTextFieldWidget) Unmount() {
	w.inner.Unmount()
}

func (w *roTextFieldWidget) Width(width float32) *roTextFieldWidget {
	w.width = width
	return w
}

const roUITextFieldHeight = 22
