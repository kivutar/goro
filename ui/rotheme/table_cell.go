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

type TableViewSimpleCell struct {
	Text               string
	Icon               image.Image
	Align              widget.TextAlign
	Color              widget.Color
	Hidden             bool
	IconButton         IconButtonKind
	HasIconButton      bool
	IconButtonDisabled bool
}

func (c TableViewSimpleCell) simpleCell() tableViewSimpleCell {
	color := c.Color
	if color.IsTransparent() {
		color = Default.Colors.Text
	}
	return tableViewSimpleCell{
		text:               c.Text,
		icon:               c.Icon,
		align:              c.Align,
		color:              color,
		visible:            !c.Hidden,
		iconButton:         c.IconButton,
		hasIconButton:      c.HasIconButton,
		iconButtonDisabled: c.IconButtonDisabled,
	}
}

func TableViewIconButtonCell(kind IconButtonKind, disabled bool) TableViewSimpleCell {
	return TableViewSimpleCell{
		IconButton:         kind,
		HasIconButton:      true,
		IconButtonDisabled: disabled,
	}
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
	w.simpleCell().draw(canvas, w.Bounds())
}

func (w *tableCellWidget) simpleCell() tableViewSimpleCell {
	return tableViewSimpleCell{
		text:    w.text,
		icon:    w.icon,
		align:   w.align,
		color:   w.color,
		visible: w.IsVisible(),
	}
}

type tableViewSimpleCell struct {
	text               string
	icon               image.Image
	align              widget.TextAlign
	color              widget.Color
	visible            bool
	iconButton         IconButtonKind
	hasIconButton      bool
	iconButtonDisabled bool
}

func asTableViewSimpleCell(w widget.Widget) (tableViewSimpleCell, bool) {
	cell, ok := w.(*tableCellWidget)
	if !ok {
		return tableViewSimpleCell{}, false
	}
	return cell.simpleCell(), true
}

func (c tableViewSimpleCell) draw(canvas widget.Canvas, bounds geometry.Rect) {
	if !c.visible {
		return
	}
	if c.hasIconButton {
		size := IconButtonSize
		if size > bounds.Width() {
			size = bounds.Width()
		}
		if size > bounds.Height() {
			size = bounds.Height()
		}
		buttonBounds := geometry.NewRect(
			bounds.Min.X+(bounds.Width()-size)/2,
			bounds.Min.Y+(bounds.Height()-size)/2,
			size,
			size,
		)
		DrawIconButton(canvas, buttonBounds, c.iconButton, false, c.iconButtonDisabled)
		return
	}
	textBounds := bounds
	if c.icon != nil {
		iconBounds := c.icon.Bounds()
		iconW := float32(iconBounds.Dx())
		iconH := float32(iconBounds.Dy())
		iconY := bounds.Min.Y + (bounds.Height()-iconH)/2
		canvas.DrawImage(c.icon, geometry.Pt(bounds.Min.X+6, iconY))
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
	DrawText(canvas, c.text, geometry.NewRect(textBounds.Min.X, textY, textBounds.Width(), textHeight), Default.Typography.TextSize, c.color, false, c.align)
}

func (w *tableCellWidget) Event(widget.Context, event.Event) bool {
	return false
}

func (w *tableCellWidget) Children() []widget.Widget {
	return nil
}
