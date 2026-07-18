package rotheme

import (
	"image"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

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
