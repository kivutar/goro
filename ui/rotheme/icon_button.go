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
)

var iconButtonSegments = map[IconButtonKind][][4]float32{
	IconButtonClose: {{-1, -1, 1, 1}, {1, -1, -1, 1}},
	IconButtonPlus:  {{-1, 0, 1, 0}, {0, -1, 0, 1}},
	IconButtonMinus: {{-1, 0, 1, 0}},
	IconButtonLeft:  {{1, -1, -1, 0}, {-1, 0, 1, 1}},
	IconButtonRight: {{-1, -1, 1, 0}, {1, 0, -1, 1}},
	IconButtonUp:    {{-1, 1, 0, -1}, {0, -1, 1, 1}},
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

func drawIconGlyph(canvas widget.Canvas, bounds geometry.Rect, kind IconButtonKind, color widget.Color) {
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
