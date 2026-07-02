// Package rotheme centralizes the Ragnarok-style gogpu/ui theme values.
package rotheme

import (
	"image/color"

	"github.com/gogpu/ui/theme"
	"github.com/gogpu/ui/widget"
)

type Colors struct {
	WindowBody   widget.Color
	WindowTitle  widget.Color
	WindowBorder widget.Color
	WindowFooter widget.Color
	PanelBody    widget.Color
	Text         widget.Color
	TitleText    widget.Color
	MutedText    widget.Color
}

type Typography struct {
	FontFamily string
	TextSize   float32
}

type Theme struct {
	Colors     Colors
	Typography Typography
}

var Default = Theme{
	Colors: Colors{
		WindowBody:   fromRGBA(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
		WindowTitle:  fromRGBA(color.RGBA{R: 184, G: 214, B: 242, A: 255}),
		WindowBorder: fromRGBA(color.RGBA{R: 118, G: 160, B: 206, A: 255}),
		WindowFooter: fromRGBA(color.RGBA{R: 244, G: 246, B: 248, A: 255}),
		PanelBody:    fromRGBA(color.RGBA{R: 250, G: 252, B: 255, A: 255}),
		Text:         fromRGBA(color.RGBA{R: 38, G: 48, B: 58, A: 255}),
		TitleText:    fromRGBA(color.RGBA{R: 22, G: 54, B: 88, A: 255}),
		MutedText:    fromRGBA(color.RGBA{R: 98, G: 112, B: 126, A: 255}),
	},
	Typography: Typography{
		FontFamily: "Inter",
		TextSize:   11,
	},
}

func (t Theme) AsTheme() *theme.Theme {
	base := theme.New("Ragnarok", theme.ModeLight)
	base.Colors.Background = t.Colors.WindowBody
	base.Colors.Surface = t.Colors.WindowBody
	base.Colors.SurfaceVariant = t.Colors.PanelBody
	base.Colors.Primary = t.Colors.WindowBorder
	base.Colors.PrimaryLight = t.Colors.WindowTitle
	base.Colors.PrimaryDark = t.Colors.TitleText
	base.Colors.OnPrimary = t.Colors.TitleText
	base.Colors.OnBackground = t.Colors.Text
	base.Colors.OnSurface = t.Colors.Text
	base.Colors.Divider = t.Colors.WindowBorder
	base.Colors.Outline = t.Colors.WindowBorder
	base.Typography = base.Typography.WithFontFamily(t.Typography.FontFamily)
	base.Typography.BodyMedium = base.Typography.BodyMedium.WithSize(t.Typography.TextSize)
	base.Typography.TitleSmall = base.Typography.TitleSmall.WithSize(t.Typography.TextSize)
	base.Typography.LabelLarge = base.Typography.LabelLarge.WithSize(t.Typography.TextSize)
	return base
}

func (t Theme) IsDark() bool {
	return false
}

func (t Theme) OnSurface() widget.Color {
	return t.Colors.Text
}

func fromRGBA(c color.RGBA) widget.Color {
	return widget.RGBA8(c.R, c.G, c.B, c.A)
}
