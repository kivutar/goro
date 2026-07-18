package rotheme

import (
	"image"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

func TableTextCell(text string, width, height float32, align widget.TextAlign) widget.Widget {
	return TableTextCellColor(text, width, height, align, Default.Colors.Text)
}

func TableTextCellColor(text string, width, height float32, align widget.TextAlign, color widget.Color) widget.Widget {
	return newTableCell(text, nil, width, height, align, color)
}

func TableIconTextCell(icon image.Image, text string, width, height float32) widget.Widget {
	return newTableCell(text, icon, width, height, widget.TextAlignLeft, Default.Colors.Text)
}

type tableCellWidget struct {
	widget.WidgetBase
	text   string
	icon   image.Image
	width  float32
	height float32
	align  widget.TextAlign
	color  widget.Color
}

func newTableCell(text string, icon image.Image, width, height float32, align widget.TextAlign, color widget.Color) *tableCellWidget {
	w := &tableCellWidget{
		text:   text,
		icon:   icon,
		width:  width,
		height: height,
		align:  align,
		color:  color,
	}
	w.SetVisible(true)
	w.SetEnabled(false)
	return w
}

func (w *tableCellWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(w.width, w.height))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *tableCellWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	textBounds := bounds
	if w.icon != nil {
		iconBounds := w.icon.Bounds()
		iconW := float32(iconBounds.Dx())
		iconH := float32(iconBounds.Dy())
		iconY := bounds.Min.Y + (bounds.Height()-iconH)/2
		canvas.DrawImage(w.icon, geometry.Pt(bounds.Min.X+6, iconY))
		textBounds.Min.X += iconW + 12
	}
	textBounds.Min.X += TableCellPadX
	textBounds.Max.X -= TableCellPadX
	if textBounds.Width() < 0 {
		textBounds.Max.X = textBounds.Min.X
	}
	textHeight := Default.Typography.TextSize * 1.2
	if textHeight > textBounds.Height() {
		textHeight = textBounds.Height()
	}
	textY := textBounds.Min.Y + (textBounds.Height()-textHeight)/2
	DrawText(canvas, w.text, geometry.NewRect(textBounds.Min.X, textY, textBounds.Width(), textHeight), Default.Typography.TextSize, w.color, false, w.align)
}

func (w *tableCellWidget) Event(widget.Context, event.Event) bool {
	return false
}

func (w *tableCellWidget) Children() []widget.Widget {
	return nil
}
