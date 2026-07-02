// Package rotheme centralizes the Ragnarok-style gogpu/ui theme values.
package rotheme

import (
	"image/color"

	"github.com/go-fonts/dejavu/dejavusans"
	"github.com/go-fonts/dejavu/dejavusansbold"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/plugin"
	"github.com/gogpu/ui/theme"
	"github.com/gogpu/ui/widget"
)

const (
	dejavuFamily     = "GoroDejaVuSans"
	dejavuBoldFamily = "GoroDejaVuSansBold"
)

type Colors struct {
	WindowBody     widget.Color
	WindowTitleTop widget.Color
	WindowTitle    widget.Color
	WindowBorder   widget.Color
	WindowFooter   widget.Color
	PanelBody      widget.Color
	Text           widget.Color
	TitleText      widget.Color
	MutedText      widget.Color
}

type Typography struct {
	FontFamily     string
	BoldFontFamily string
	TextSize       float32
}

type Theme struct {
	Colors     Colors
	Typography Typography
}

var Default = Theme{
	Colors: Colors{
		WindowBody:     fromRGBA(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
		WindowTitleTop: fromRGBA(color.RGBA{R: 214, G: 232, B: 250, A: 255}),
		WindowTitle:    fromRGBA(color.RGBA{R: 184, G: 214, B: 242, A: 255}),
		WindowBorder:   fromRGBA(color.RGBA{R: 118, G: 160, B: 206, A: 255}),
		WindowFooter:   fromRGBA(color.RGBA{R: 244, G: 246, B: 248, A: 255}),
		PanelBody:      fromRGBA(color.RGBA{R: 250, G: 252, B: 255, A: 255}),
		Text:           fromRGBA(color.RGBA{R: 38, G: 48, B: 58, A: 255}),
		TitleText:      fromRGBA(color.RGBA{R: 22, G: 54, B: 88, A: 255}),
		MutedText:      fromRGBA(color.RGBA{R: 98, G: 112, B: 126, A: 255}),
	},
	Typography: Typography{
		FontFamily:     dejavuFamily,
		BoldFontFamily: dejavuBoldFamily,
		TextSize:       11,
	},
}

func init() {
	ctx := plugin.NewDefaultPluginContext()
	_ = ctx.Assets.LoadFont(dejavuFamily, dejavusans.TTF)
	_ = ctx.Assets.LoadFont(dejavuBoldFamily, dejavusansbold.TTF)
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

func DrawText(canvas widget.Canvas, text string, bounds geometry.Rect, size float32, color widget.Color, bold bool, align widget.TextAlign) {
	if text == "" {
		return
	}
	family := Default.Typography.FontFamily
	if bold && Default.Typography.BoldFontFamily != "" {
		family = Default.Typography.BoldFontFamily
		bold = false
	}
	if styled, ok := canvas.(widget.StyledTextDrawer); ok {
		styled.DrawStyledText(text, bounds, widget.TextStyle{
			FontFamily: family,
			FontSize:   size,
			Bold:       bold,
			Color:      color,
			Align:      align,
		})
		return
	}
	canvas.DrawText(text, bounds, size, color, bold, align)
}

func MeasureText(canvas widget.Canvas, text string, size float32, bold bool) float32 {
	if text == "" {
		return 0
	}
	family := Default.Typography.FontFamily
	if bold && Default.Typography.BoldFontFamily != "" {
		family = Default.Typography.BoldFontFamily
		bold = false
	}
	if styled, ok := canvas.(widget.StyledTextDrawer); ok {
		return styled.MeasureStyledText(text, widget.TextStyle{
			FontFamily: family,
			FontSize:   size,
			Bold:       bold,
		})
	}
	return canvas.MeasureText(text, size, bold)
}

func fromRGBA(c color.RGBA) widget.Color {
	return widget.RGBA8(c.R, c.G, c.B, c.A)
}
