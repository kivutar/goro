package rotheme

import (
	"github.com/gogpu/ui/core/listview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

const SelectListRowPadX float32 = 6

type SelectListPainter struct {
	EmptyText string
}

type selectListRowWidget struct {
	widget.WidgetBase
	label  string
	color  widget.Color
	height float32
}

func SelectListRow(label string, enabled bool, height float32) widget.Widget {
	color := Default.Colors.Text
	if !enabled {
		color = Default.Colors.MutedText
	}
	row := &selectListRowWidget{
		label:  label,
		color:  color,
		height: height,
	}
	row.SetVisible(true)
	row.SetEnabled(enabled)
	return row
}

func (w *selectListRowWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.BiggestFinite(0, w.height)
	size.Height = constraints.ConstrainHeight(w.height)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *selectListRowWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	textWidth := bounds.Width() - 2*SelectListRowPadX
	if textWidth < 0 {
		textWidth = 0
	}
	textBounds := geometry.NewRect(
		bounds.Min.X+SelectListRowPadX,
		bounds.Min.Y,
		textWidth,
		bounds.Height(),
	)
	DrawText(canvas, w.label, textBounds, Default.Typography.TextSize, w.color, false, widget.TextAlignLeft)
}

func (w *selectListRowWidget) Children() []widget.Widget {
	return nil
}

func (w *selectListRowWidget) Event(widget.Context, event.Event) bool {
	return false
}

func (SelectListPainter) PaintDivider(widget.Canvas, listview.DividerState) {}

func (p SelectListPainter) PaintEmptyState(canvas widget.Canvas, bounds geometry.Rect) {
	DrawText(canvas, p.EmptyText, bounds, Default.Typography.TextSize, Default.Colors.MutedText, false, widget.TextAlignCenter)
}

func (SelectListPainter) PaintItemBackground(canvas widget.Canvas, state listview.ItemPaintState) {
	fill := widget.RGBA8(246, 249, 253, 255)
	if state.Index%2 == 1 {
		fill = Default.Colors.PanelBody
	}
	canvas.DrawRect(state.Bounds, fill)
}

func (SelectListPainter) PaintSelection(canvas widget.Canvas, state listview.ItemPaintState) {
	if state.Selected {
		canvas.DrawRect(state.Bounds, Default.Colors.ButtonDown)
	}
}

var _ listview.Painter = SelectListPainter{}
