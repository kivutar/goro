package ui

import (
	"image/color"
	"math"
	"time"

	uiwidget "github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
)

type surfaceKey struct {
	w        int
	h        int
	top      color.RGBA
	bottom   color.RGBA
	border   color.RGBA
	radius   float32
	gradient bool
}

var surfaceCache = map[surfaceKey]*render.Image{}

const IconButtonSize = 17

const HungerBarLowThreshold = 0.25

var (
	WindowRadius      = float32(6)
	ButtonRadius      = float32(6)
	ButtonPaddingX    = 14
	WindowBodyColor   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	WindowTitleTop    = color.RGBA{R: 214, G: 232, B: 250, A: 255}
	WindowTitleColor  = color.RGBA{R: 184, G: 214, B: 242, A: 255}
	WindowBorderColor = color.RGBA{R: 118, G: 160, B: 206, A: 255}
	WindowFooterColor = color.RGBA{R: 244, G: 246, B: 248, A: 255}
	PanelBodyColor    = color.RGBA{R: 250, G: 252, B: 255, A: 255}
	PanelAltColor     = color.RGBA{R: 236, G: 244, B: 252, A: 255}
	PanelHoverColor   = color.RGBA{R: 222, G: 236, B: 250, A: 255}
	DisabledColor     = color.RGBA{R: 226, G: 230, B: 235, A: 255}
	TextColor         = color.RGBA{R: 38, G: 48, B: 58, A: 255}
	MutedTextColor    = color.RGBA{R: 98, G: 112, B: 126, A: 255}
	TitleTextColor    = color.RGBA{R: 22, G: 54, B: 88, A: 255}
	GoodTextColor     = color.RGBA{R: 34, G: 142, B: 72, A: 255}
	ErrorTextColor    = color.RGBA{R: 204, G: 48, B: 48, A: 255}
	ButtonColor       = color.RGBA{R: 236, G: 244, B: 252, A: 255}
	ButtonHoverColor  = color.RGBA{R: 218, G: 235, B: 250, A: 255}
	ButtonDownColor   = color.RGBA{R: 198, G: 222, B: 245, A: 255}
	ButtonBorderColor = color.RGBA{R: 138, G: 174, B: 214, A: 255}
	PlayerHPBarColor  = color.RGBA{R: 16, G: 239, B: 33, A: 255}
	PlayerSPBarColor  = color.RGBA{R: 24, G: 99, B: 222, A: 255}
	HungerBarColor    = color.RGBA{R: 255, G: 231, B: 231, A: 255}
	HungerBarLowColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	SeparatorColor    = color.RGBA{R: 160, G: 190, B: 222, A: 190}
	FooterLineColor   = color.RGBA{R: 174, G: 180, B: 188, A: 255}
	SelectionColor    = color.RGBA{R: 206, G: 226, B: 248, A: 255}
	SelectionBorder   = color.RGBA{R: 82, G: 138, B: 200, A: 255}
)

func HungerBarFillColor(current, maxValue int) color.RGBA {
	ratio := 0.0
	if maxValue > 0 {
		ratio = clampUnit(float64(current) / float64(maxValue))
	}
	if ratio < HungerBarLowThreshold {
		return HungerBarLowColor
	}
	return HungerBarColor
}

func DrawWindowFrame(screen *render.Frame, x, y, w, h int) {
	DrawRoundedSurface(screen, x, y, w, h, WindowBodyColor, WindowBorderColor, WindowRadius)
}

func DrawTitledWindowFrame(screen *render.Frame, x, y, w, h, titleH int) {
	DrawWindowFrame(screen, x, y, w, h)
	DrawTitleBar(screen, x, y, w, titleH)
}

func DrawWindowTitle(screen *render.Frame, x, y, titleH, pad int, title string, text color.RGBA) {
	DrawTitleTextAt(screen, x+pad, y, titleH, title, text)
}

func DrawTitleTextAt(screen *render.Frame, x, y, titleH int, title string, text color.RGBA) {
	ty := y + render.BitmapTextTopForCenter(titleH)
	if titleH > 1 {
		ty = y + 1 + render.BitmapTextTopForCenter(titleH-1)
	}
	render.DrawBitmapTextAtColor(screen, title, x, ty, text)
}

func DrawTitleBar(screen *render.Frame, x, y, w, titleH int) {
	if titleH <= 1 {
		return
	}
	barH := titleH - 1
	for row := 0; row < barH; row++ {
		t := 0.0
		if barH > 1 {
			t = float64(row) / float64(barH-1)
		}
		c := LerpColor(WindowTitleTop, WindowTitleColor, t)
		for col := 1; col < w-1; col++ {
			if roundedTopSampleInside(float64(col)+0.5, float64(row+1)+0.5, float64(w), float64(WindowRadius), 1) {
				render.DrawRect(screen, float64(x+col), float64(y+1+row), 1, 1, c)
			}
		}
	}
	render.DrawRect(screen, float64(x+1), float64(y+titleH), float64(w-2), 1, SeparatorColor)
}

func DrawWindowFooter(screen *render.Frame, x, y, w, h, footerH int) {
	if screen == nil || w <= 2 || h <= 2 || footerH <= 0 {
		return
	}
	footerY := y + h - footerH
	if footerY <= y {
		footerY = y + 1
	}
	bottom := y + h - 1
	if footerY >= bottom {
		return
	}
	drawFooterRows(screen, x, y, w, h, footerY, bottom)
}

func drawFooterRows(screen *render.Frame, x, y, w, h, footerY, bottom int) {
	for row := footerY; row < bottom; row++ {
		inset := footerInnerRowInset(w, h, row-y)
		width := w - inset*2
		if width <= 0 {
			continue
		}
		rowColor := WindowFooterColor
		if row == footerY {
			rowColor = FooterLineColor
		}
		render.DrawRect(screen, float64(x+inset), float64(row), float64(width), 1, rowColor)
	}
}

func footerInnerRowInset(w, h, localY int) int {
	inset := 1
	radius := float64(WindowRadius) - 1
	if radius <= 0 {
		return inset
	}
	bottomCenter := float64(h) - 1 - radius
	sampleY := float64(localY) + 0.5
	if sampleY <= bottomCenter {
		return inset
	}
	dy := sampleY - bottomCenter
	if dy >= radius {
		return w / 2
	}
	dx := math.Sqrt(radius*radius - dy*dy)
	inset = int(math.Ceil(1 + radius - dx))
	if inset < 1 {
		return 1
	}
	if inset > w/2 {
		return w / 2
	}
	return inset
}

func LerpColor(a, b color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t + 0.5),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t + 0.5),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t + 0.5),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t + 0.5),
	}
}

func DrawPanelSurface(screen *render.Frame, x, y, w, h int, bg color.RGBA) {
	DrawSurface(screen, x, y, w, h, bg, WindowBorderColor)
}

func DrawButtonSurface(screen *render.Frame, x, y, w, h int, bg color.RGBA) {
	DrawRoundedGradientSurface(screen, x, y, w, h, Lighten(bg, 0.42), bg, ButtonBorderColor, ButtonRadius)
}

func DrawButtonLabel(screen *render.Frame, x, y, w, h int, label string, bg, text color.RGBA) {
	DrawButtonSurface(screen, x, y, w, h, bg)
	DrawCenteredText(screen, x, y, w, h, label, text)
}

func ButtonLabelWidth(label string) int {
	textW, _ := render.BitmapTextSize(label)
	return textW + ButtonPaddingX*2
}

func DrawCloseButton(screen *render.Frame, x, y, w, h int, bg, line color.RGBA) {
	DrawButtonSurface(screen, x, y, w, h, bg)
	DrawCloseGlyph(screen, x, y, w, h, line)
}

func DrawCloseGlyph(screen *render.Frame, x, y, w, h int, line color.RGBA) {
	icon := minInt(w, h) / 2
	if icon < 6 {
		icon = minInt(w, h) - 6
	}
	if icon%2 == 0 {
		icon--
	}
	if icon < 2 {
		return
	}
	midX := x + w/2
	midY := y + h/2
	half := icon / 2
	left := midX - half
	right := midX + half
	top := midY - half
	bottom := midY + half
	render.DrawLine(screen, float64(left), float64(top), float64(right), float64(bottom), line)
	render.DrawLine(screen, float64(right), float64(top), float64(left), float64(bottom), line)
}

func DrawPlusButton(screen *render.Frame, x, y int, bg, line color.RGBA) {
	DrawIconButton(screen, x, y, IconButtonSize, IconButtonSize, bg, line, true)
}

func DrawMinusButton(screen *render.Frame, x, y int, bg, line color.RGBA) {
	DrawIconButton(screen, x, y, IconButtonSize, IconButtonSize, bg, line, false)
}

func DrawCheckboxButton(screen *render.Frame, x, y int, bg, line color.RGBA, checked bool) {
	DrawButtonSurface(screen, x, y, IconButtonSize, IconButtonSize, bg)
	if !checked {
		return
	}
	left := x + 4
	midX := x + 7
	right := x + 13
	midY := y + 10
	top := y + 5
	bottom := y + 12
	render.DrawLine(screen, float64(left), float64(midY), float64(midX), float64(bottom), line)
	render.DrawLine(screen, float64(midX), float64(bottom), float64(right), float64(top), line)
}

func DrawIconButton(screen *render.Frame, x, y, w, h int, bg, line color.RGBA, vertical bool) {
	DrawButtonSurface(screen, x, y, w, h, bg)
	icon := minInt(w, h) / 2
	if icon < 6 {
		icon = minInt(w, h) - 6
	}
	if icon%2 == 0 {
		icon--
	}
	if icon < 2 {
		return
	}
	midX := x + w/2
	midY := y + h/2
	half := icon / 2
	left := midX - half
	right := midX + half
	render.DrawLine(screen, float64(left), float64(midY), float64(right), float64(midY), line)
	if vertical {
		top := midY - half
		bottom := midY + half
		render.DrawLine(screen, float64(midX), float64(top), float64(midX), float64(bottom), line)
	}
}

func DrawCenteredText(screen *render.Frame, x, y, w, h int, label string, text color.RGBA) {
	textW, textH := render.BitmapTextSize(label)
	tx := x + (w-textW)/2
	ty := y + (h-textH)/2
	render.DrawBitmapTextAtColor(screen, label, tx, ty, text)
}

func DrawRowSurface(screen *render.Frame, x, y, w, h int, bg color.RGBA) {
	DrawSurface(screen, x, y, w, h, bg, color.RGBA{})
}

func DrawTextBoxSurface(screen *render.Frame, x, y, w, h int, bg, border color.RGBA) {
	DrawSurface(screen, x, y, w, h, bg, border)
	shadowRows := minInt(4, h-2)
	for row := 0; row < shadowRows; row++ {
		alpha := uint8(34 - row*7)
		render.DrawRect(screen, float64(x+1), float64(y+1+row), float64(w-2), 1, color.RGBA{R: 82, G: 108, B: 138, A: alpha})
	}
}

func DrawBlinkingCaret(screen *render.Frame, x, y, h int, c color.RGBA) {
	if screen == nil || h <= 0 || time.Now().UnixMilli()/500%2 != 0 {
		return
	}
	top := y + 4
	caretH := h - 8
	if caretH < 8 {
		top = y + 2
		caretH = maxInt(4, h-4)
	}
	render.DrawRect(screen, float64(x), float64(top), 1, float64(caretH), c)
}

type imageDrawTarget interface {
	DrawImage(*render.Image, *render.DrawImageOptions)
}

func DrawSurface(screen imageDrawTarget, x, y, w, h int, bg, border color.RGBA) {
	DrawRoundedSurface(screen, x, y, w, h, bg, border, 0)
}

func DrawRoundedSurface(screen imageDrawTarget, x, y, w, h int, bg, border color.RGBA, radius float32) {
	DrawRoundedGradientSurface(screen, x, y, w, h, bg, bg, border, radius)
}

func DrawRoundedGradientSurface(screen imageDrawTarget, x, y, w, h int, top, bottom, border color.RGBA, radius float32) {
	if screen == nil || w <= 0 || h <= 0 {
		return
	}
	img := cachedSurface(w, h, top, bottom, border, radius)
	if img == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	opts.Filter = render.FilterNearest
	screen.DrawImage(img, &opts)
}

func Lighten(c color.RGBA, amount float64) color.RGBA {
	return LerpColor(c, color.RGBA{R: 255, G: 255, B: 255, A: c.A}, clampUnit(amount))
}

func cachedSurface(w, h int, top, bottom, border color.RGBA, radius float32) *render.Image {
	key := surfaceKey{w: w, h: h, top: top, bottom: bottom, border: border, radius: radius, gradient: top != bottom}
	if img, ok := surfaceCache[key]; ok {
		return img
	}
	img := render.NewImage(w, h)
	drawCachedSurface(img, w, h, top, bottom, border, float64(radius))
	if len(surfaceCache) > 256 {
		surfaceCache = map[surfaceKey]*render.Image{}
	}
	surfaceCache[key] = img
	return img
}

func drawCachedSurface(img *render.Image, w, h int, top, bottom, border color.RGBA, radius float64) {
	if img == nil || w <= 0 || h <= 0 {
		return
	}
	maxRadius := float64(minInt(w, h)) * 0.5
	if radius > maxRadius {
		radius = maxRadius
	}
	const samples = 4
	for py := 0; py < h; py++ {
		t := 0.0
		if h > 1 {
			t = float64(py) / float64(h-1)
		}
		fill := LerpColor(top, bottom, t)
		for px := 0; px < w; px++ {
			var outer, inner float64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					x := float64(px) + (float64(sx)+0.5)/samples
					y := float64(py) + (float64(sy)+0.5)/samples
					if roundedSampleInside(x, y, float64(w), float64(h), radius, 0) {
						outer++
					}
					if roundedSampleInside(x, y, float64(w), float64(h), radius, 1) {
						inner++
					}
				}
			}
			outer /= samples * samples
			inner /= samples * samples
			if border.A != 0 {
				drawSurfacePixel(img, px, py, border, math.Max(0, outer-inner))
				drawSurfacePixel(img, px, py, fill, inner)
			} else {
				drawSurfacePixel(img, px, py, fill, outer)
			}
		}
	}
}

func roundedSampleInside(x, y, w, h, radius float64, inset float64) bool {
	if x < inset || y < inset || x >= w-inset || y >= h-inset {
		return false
	}
	r := radius - inset
	if r <= 0 {
		return true
	}
	left := inset + r
	right := w - inset - r
	top := inset + r
	bottom := h - inset - r
	cx := clampFloat(x, left, right)
	cy := clampFloat(y, top, bottom)
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= r*r
}

func roundedTopSampleInside(x, y, w, radius float64, inset float64) bool {
	if x < inset || y < inset || x >= w-inset {
		return false
	}
	r := radius - inset
	if r <= 0 {
		return true
	}
	left := inset + r
	right := w - inset - r
	top := inset + r
	if y >= top {
		return true
	}
	cx := clampFloat(x, left, right)
	dx := x - cx
	dy := y - top
	return dx*dx+dy*dy <= r*r
}

func drawSurfacePixel(img *render.Image, x, y int, c color.RGBA, coverage float64) {
	if c.A == 0 || coverage <= 0 {
		return
	}
	if coverage > 1 {
		coverage = 1
	}
	c.A = uint8(float64(c.A)*coverage + 0.5)
	if c.A == 0 {
		return
	}
	blendStraightAlphaPixel(img, x, y, c)
}

func blendStraightAlphaPixel(img *render.Image, x, y int, src color.RGBA) {
	rgba := img.RGBA()
	if rgba == nil {
		return
	}
	bounds := rgba.Bounds()
	if x < 0 || y < 0 || x >= bounds.Dx() || y >= bounds.Dy() {
		return
	}
	off := rgba.PixOffset(x+bounds.Min.X, y+bounds.Min.Y)
	sa := float64(src.A) / 255
	da := float64(rgba.Pix[off+3]) / 255
	outA := sa + da*(1-sa)
	if outA <= 0 {
		rgba.Pix[off], rgba.Pix[off+1], rgba.Pix[off+2], rgba.Pix[off+3] = 0, 0, 0, 0
		return
	}
	dr := float64(rgba.Pix[off])
	dg := float64(rgba.Pix[off+1])
	db := float64(rgba.Pix[off+2])
	rgba.Pix[off] = uint8((float64(src.R)*sa+dr*da*(1-sa))/outA + 0.5)
	rgba.Pix[off+1] = uint8((float64(src.G)*sa+dg*da*(1-sa))/outA + 0.5)
	rgba.Pix[off+2] = uint8((float64(src.B)*sa+db*da*(1-sa))/outA + 0.5)
	rgba.Pix[off+3] = uint8(outA*255 + 0.5)
}

func Color(c color.RGBA) uiwidget.Color {
	return uiwidget.RGBA8(c.R, c.G, c.B, c.A)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
