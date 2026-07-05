package rotheme

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

const IconButtonSize float32 = 17

type IconButtonKind int

const (
	IconButtonClose IconButtonKind = iota
	IconButtonPlus
	IconButtonMinus
	IconButtonLeft
	IconButtonRight
	IconButtonUp
	IconButtonDown
)

var iconButtonSegments = map[IconButtonKind][][4]float32{
	IconButtonClose: {{-1, -1, 1, 1}, {1, -1, -1, 1}},
	IconButtonPlus:  {{-1, 0, 1, 0}, {0, -1, 0, 1}},
	IconButtonMinus: {{-1, 0, 1, 0}},
}

func IconButton(kind IconButtonKind, onClick func()) *primitives.BoxWidget {
	return IconButtonDisabled(kind, false, onClick)
}

func IconButtonDisabled(kind IconButtonKind, disabled bool, onClick func()) *primitives.BoxWidget {
	opts := []button.Option{
		button.TextOpt(""),
		button.SizeOpt(button.Small),
		button.PainterOpt(IconButtonPainter{Kind: kind}),
		button.RoundedOpt(ButtonRadius),
		button.Disabled(disabled),
	}
	if !disabled && onClick != nil {
		opts = append(opts, button.OnClick(onClick))
	}
	return primitives.Box(button.New(opts...).PaddingXY(0, 0).MinWidth(IconButtonSize)).
		Width(IconButtonSize).
		Height(IconButtonSize)
}

type IconButtonPainter struct {
	Kind IconButtonKind
}

func (p IconButtonPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	ButtonPainter{}.PaintButton(canvas, state)
	color := Default.Colors.Text
	if state.Disabled {
		color = Default.Colors.MutedText
	}
	drawIconGlyph(canvas, state.Bounds, p.Kind, color)
}

func DrawIconButton(canvas widget.Canvas, bounds geometry.Rect, kind IconButtonKind, hovered, disabled bool) {
	bg := Default.Colors.Button
	color := Default.Colors.Text
	border := Default.Colors.ButtonBorder
	if hovered {
		bg = Default.Colors.ButtonHover
	}
	if disabled {
		bg = Default.Colors.Disabled
		color = Default.Colors.MutedText
		border = Default.Colors.Disabled
	}
	drawButtonGradient(canvas, bounds, bg, ButtonRadius)
	canvas.StrokeRoundRect(bounds, border, ButtonRadius, 1)
	drawIconGlyph(canvas, bounds, kind, color)
}

func drawIconGlyph(canvas widget.Canvas, bounds geometry.Rect, kind IconButtonKind, color widget.Color) {
	switch kind {
	case IconButtonLeft:
		drawIconChevron(canvas, bounds, color, geometry.Pt(11, 4), geometry.Pt(5, 8), geometry.Pt(11, 12))
		return
	case IconButtonRight:
		drawIconChevron(canvas, bounds, color, geometry.Pt(5, 4), geometry.Pt(11, 8), geometry.Pt(5, 12))
		return
	case IconButtonUp:
		drawIconChevron(canvas, bounds, color, geometry.Pt(4, 11), geometry.Pt(8, 5), geometry.Pt(12, 11))
		return
	case IconButtonDown:
		drawIconChevron(canvas, bounds, color, geometry.Pt(4, 5), geometry.Pt(8, 11), geometry.Pt(12, 5))
		return
	}

	size := bounds.Width()
	if bounds.Height() < size {
		size = bounds.Height()
	}
	icon := int(size) / 2
	if icon < 6 {
		icon = int(size) - 6
	}
	if icon%2 == 0 {
		icon--
	}
	if icon < 2 {
		return
	}
	midX := float32(int(bounds.Min.X + bounds.Width()/2))
	midY := float32(int(bounds.Min.Y + bounds.Height()/2))
	half := float32(icon / 2)
	for _, s := range iconButtonSegments[kind] {
		canvas.DrawLine(
			geometry.Pt(midX+s[0]*half, midY+s[1]*half),
			geometry.Pt(midX+s[2]*half, midY+s[3]*half),
			color,
			1,
		)
	}
}

func drawIconChevron(canvas widget.Canvas, bounds geometry.Rect, color widget.Color, p1, p2, p3 geometry.Point) {
	size := bounds.Width()
	if bounds.Height() < size {
		size = bounds.Height()
	}
	scale := size / IconButtonSize
	offsetX := bounds.Min.X + (bounds.Width()-IconButtonSize*scale)/2
	offsetY := bounds.Min.Y + (bounds.Height()-IconButtonSize*scale)/2
	a := geometry.Pt(offsetX+p1.X*scale, offsetY+p1.Y*scale)
	b := geometry.Pt(offsetX+p2.X*scale, offsetY+p2.Y*scale)
	c := geometry.Pt(offsetX+p3.X*scale, offsetY+p3.Y*scale)
	canvas.DrawLine(a, b, color, 1)
	canvas.DrawLine(b, c, color, 1)
}
