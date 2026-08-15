package rotheme

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

const (
	TableRowHeight = float32(14)
	TableGap       = float32(1)
	TableCellPadX  = float32(4)
)

type TableCell struct {
	Text  string
	Width float32
	Align widget.TextAlign
	Head  bool
}

type TableRow []TableCell

type TableOption func(*tableWidget)

type tableWidget struct {
	widget.WidgetBase
	rows      []TableRow
	rowHeight float32
	gap       float32
	cellPadX  float32
	headBG    widget.Color
	cellBG    widget.Color
}

func Table(rows []TableRow, opts ...TableOption) widget.Widget {
	w := &tableWidget{
		rows:      rows,
		rowHeight: TableRowHeight,
		gap:       TableGap,
		cellPadX:  TableCellPadX,
		headBG:    Default.Colors.ButtonHover,
		cellBG:    Default.Colors.WindowFooter,
	}
	for _, opt := range opts {
		opt(w)
	}
	w.SetVisible(true)
	w.SetEnabled(false)
	return w
}

func TableRowHeightOpt(height float32) TableOption {
	return func(w *tableWidget) { w.rowHeight = height }
}

func TableColors(headBG, cellBG widget.Color) TableOption {
	return func(w *tableWidget) {
		w.headBG = headBG
		w.cellBG = cellBG
	}
}

func (w *tableWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(w.width(), w.height()))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *tableWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	for rowIndex, row := range w.rows {
		x := bounds.Min.X
		y := bounds.Min.Y + float32(rowIndex)*(w.rowHeight+w.gap)
		for _, cell := range row {
			bg := w.cellBG
			if cell.Head {
				bg = w.headBG
			}
			rect := geometry.NewRect(x, y, cell.Width, w.rowHeight)
			canvas.DrawRect(rect, bg)
			textRect := geometry.NewRect(rect.Min.X+w.cellPadX, rect.Min.Y, rect.Width()-2*w.cellPadX, rect.Height())
			if cell.Head {
				DrawLabel(canvas, cell.Text, textRect, cell.Align)
			} else {
				DrawText(canvas, cell.Text, textRect, Default.Typography.TextSize, Default.Colors.Text, false, cell.Align)
			}
			x += cell.Width + w.gap
		}
	}
}

func (w *tableWidget) Event(widget.Context, event.Event) bool {
	return false
}

func (w *tableWidget) Children() []widget.Widget {
	return nil
}

func (w *tableWidget) width() float32 {
	width := float32(0)
	for _, row := range w.rows {
		rowWidth := float32(0)
		for i, cell := range row {
			rowWidth += cell.Width
			if i > 0 {
				rowWidth += w.gap
			}
		}
		if rowWidth > width {
			width = rowWidth
		}
	}
	return width
}

func (w *tableWidget) height() float32 {
	if len(w.rows) == 0 {
		return 0
	}
	return float32(len(w.rows))*w.rowHeight + float32(len(w.rows)-1)*w.gap
}
