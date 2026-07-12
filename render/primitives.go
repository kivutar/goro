package render

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/go-fonts/dejavu/dejavusans"
	"github.com/go-fonts/dejavu/dejavusansbold"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var debugTextCache = make(map[string]*Image)
var outlinedTextCache = make(map[string]*Image)
var textFontMu sync.Mutex
var debugTextFace font.Face = basicfont.Face7x13
var debugTextLineHeight = 14
var debugTextBaseline = 13
var debugTextFixedWidth = 7
var outlinedTextFace font.Face

func init() {
	regular, err := parseOpenTypeFace(dejavusans.TTF, 11)
	if err != nil {
		return
	}
	bold, _ := parseOpenTypeFace(dejavusansbold.TTF, 12)
	textFontMu.Lock()
	defer textFontMu.Unlock()
	debugTextFace = noKernFace{Face: regular}
	debugTextFixedWidth = 0
	debugTextLineHeight, debugTextBaseline = textFaceLineMetrics(regular)
	if debugTextLineHeight < 14 {
		debugTextLineHeight = 14
	}
	if bold != nil {
		outlinedTextFace = noKernFace{Face: bold}
	}
}

type noKernFace struct {
	font.Face
}

func (f noKernFace) Kern(r0, r1 rune) fixed.Int26_6 {
	return 0
}

// SetUIFont installs the client UI font used for ordinary window text and,
// when provided, the bold face used by outlined actor names. It is intended
// for RO clients that ship System/Font/SCDream4.otf and SCDream6.otf.
func SetUIFont(regular, bold []byte) error {
	regularFace, err := parseOpenTypeFace(regular, 11)
	if err != nil {
		return err
	}
	var boldFace font.Face
	if len(bold) > 0 {
		boldFace, err = parseOpenTypeFace(bold, 12)
		if err != nil {
			return err
		}
	}
	textFontMu.Lock()
	defer textFontMu.Unlock()
	debugTextFace = regularFace
	debugTextFixedWidth = 0
	debugTextLineHeight, debugTextBaseline = textFaceLineMetrics(regularFace)
	if debugTextLineHeight < 14 {
		debugTextLineHeight = 14
	}
	if boldFace != nil {
		outlinedTextFace = boldFace
	}
	debugTextCache = make(map[string]*Image)
	outlinedTextCache = make(map[string]*Image)
	return nil
}

func DrawRect(dst *Image, x, y, w, h float64, c color.Color) {
	if dst == nil || dst.pix == nil || w <= 0 || h <= 0 {
		return
	}
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	if dst.screen {
		x0, y0 := snapScreenPoint(dst, x, y)
		x1, y1 := snapScreenPoint(dst, x+w, y+h)
		w, h = x1-x0, y1-y0
		if w <= 0 || h <= 0 {
			return
		}
		x, y = x0, y0
		drawSolidQuad(dst, x, y, w, h, rgba)
		return
	}
	x0 := clampInt(int(math.Floor(x)), 0, dst.pix.Bounds().Dx())
	y0 := clampInt(int(math.Floor(y)), 0, dst.pix.Bounds().Dy())
	x1 := clampInt(int(math.Ceil(x+w)), 0, dst.pix.Bounds().Dx())
	y1 := clampInt(int(math.Ceil(y+h)), 0, dst.pix.Bounds().Dy())
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			dst.blendPixel(xx, yy, rgba, BlendSourceOver)
		}
	}
}

func DrawUIRect(dst *Image, x, y, w, h float64, c color.RGBA) {
	if dst == nil || w <= 0 || h <= 0 {
		return
	}
	if !dst.screen {
		DrawRect(dst, x, y, w, h, c)
		return
	}
	dst.uiRects = append(dst.uiRects, UIRectCommand{
		X:     x,
		Y:     y,
		W:     w,
		H:     h,
		Color: c,
	})
}

func DrawUISpeechBubble(dst *Image, text string, centerX, bottomY, maxWidth float64) {
	if dst == nil || text == "" {
		return
	}
	if maxWidth <= 0 {
		maxWidth = 220
	}
	if !dst.screen {
		return
	}
	dst.uiTextBoxes = append(dst.uiTextBoxes, UITextBoxCommand{
		Text:     text,
		X:        centerX,
		Y:        bottomY,
		Anchor:   UITextBoxAnchorBottomCenter,
		MaxWidth: maxWidth,
		MaxLines: 4,
	})
}

func DrawUITooltip(dst *Image, text string, centerX, belowY, aboveY float64) {
	if dst == nil || text == "" || !dst.screen {
		return
	}
	dst.uiTextBoxes = append(dst.uiTextBoxes, UITextBoxCommand{
		Text:   text,
		X:      centerX,
		Y:      belowY,
		AltY:   aboveY,
		Anchor: UITextBoxAnchorTooltipCenter,
	})
}

func DrawLine(dst *Image, x0, y0, x1, y1 float64, c color.Color) {
	if dst == nil || dst.pix == nil {
		return
	}
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	if dst.screen {
		steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0)))
		if steps <= 0 {
			drawSolidQuad(dst, math.Round(x0), math.Round(y0), 1, 1, rgba)
			return
		}
		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			drawSolidQuad(dst, math.Round(x0+(x1-x0)*t), math.Round(y0+(y1-y0)*t), 1, 1, rgba)
		}
		return
	}
	dx := x1 - x0
	dy := y1 - y0
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if steps <= 0 {
		dst.blendPixel(clampInt(int(math.Round(x0)), 0, dst.pix.Bounds().Dx()-1), clampInt(int(math.Round(y0)), 0, dst.pix.Bounds().Dy()-1), rgba, BlendSourceOver)
		return
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(x0 + dx*t))
		y := int(math.Round(y0 + dy*t))
		if imageContains(dst, x, y) {
			dst.blendPixel(x, y, rgba, BlendSourceOver)
		}
	}
}

func DebugPrintAt(dst *Image, text string, x, y int) {
	DebugPrintAtColor(dst, text, x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

func DebugPrintAtColor(dst *Image, text string, x, y int, c color.RGBA) {
	if dst == nil || dst.pix == nil || text == "" {
		return
	}
	if dst.screen {
		img := cachedDebugTextColor(text, c)
		var opts DrawImageOptions
		x, y := snapScreenPoint(dst, float64(x), float64(y))
		opts.GeoM.Translate(x, y)
		opts.Filter = FilterNearest
		dst.DrawImage(img, &opts)
		return
	}
	face, _, baseline, _ := debugTextFont()
	d := &font.Drawer{
		Dst:  dst.pix,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y+baseline),
	}
	d.DrawString(text)
}

func cachedDebugTextColor(text string, c color.RGBA) *Image {
	key := fmt.Sprintf("%02x%02x%02x%02x:%s", c.R, c.G, c.B, c.A, text)
	if img := debugTextCache[key]; img != nil {
		return img
	}
	w, h := debugTextSize(text)
	if w < 1 {
		w = 1
	}
	img := NewImage(w, h)
	_, _, _, fixedWidth := debugTextFont()
	y := 0
	if fixedWidth > 0 {
		y = -1
	}
	DebugPrintAtColor(img, text, 0, y, c)
	debugTextCache[key] = img
	if len(debugTextCache) > 512 {
		for key := range debugTextCache {
			delete(debugTextCache, key)
			if len(debugTextCache) <= 384 {
				break
			}
		}
	}
	return img
}

func debugTextSize(text string) (int, int) {
	face, lineHeight, _, fixedWidth := debugTextFont()
	if fixedWidth > 0 {
		return len([]rune(text)) * fixedWidth, lineHeight
	}
	width := font.MeasureString(face, text).Ceil() + 2
	if width < 1 {
		width = 1
	}
	return width, lineHeight
}

func DebugTextSize(text string) (int, int) {
	return debugTextSize(text)
}

func DebugTextTopForCenter(containerH int) int {
	face, lineHeight, baseline, _ := debugTextFont()
	metrics := face.Metrics()
	textH := (metrics.Ascent + metrics.Descent).Ceil()
	ascent := metrics.Ascent.Ceil()
	if textH <= 0 || ascent <= 0 {
		return (containerH - lineHeight) / 2
	}
	return (containerH-textH)/2 + ascent - baseline
}

func debugTextFont() (font.Face, int, int, int) {
	textFontMu.Lock()
	defer textFontMu.Unlock()
	return debugTextFace, debugTextLineHeight, debugTextBaseline, debugTextFixedWidth
}

func OutlinedTextImage(text string, foreground, outline color.RGBA) *Image {
	key := fmt.Sprintf("%02x%02x%02x%02x:%02x%02x%02x%02x:%s", foreground.R, foreground.G, foreground.B, foreground.A, outline.R, outline.G, outline.B, outline.A, text)
	if img := outlinedTextCache[key]; img != nil {
		return img
	}
	face := roNameTextFace()
	width := (font.MeasureString(face, text).Ceil()) + 4
	if width < 1 {
		width = 1
	}
	baseline := 2 + face.Metrics().Ascent.Ceil()
	height := 4 + (face.Metrics().Ascent + face.Metrics().Descent).Ceil()
	if height < 16 {
		height = 16
	}
	img := NewImage(width, height)
	drawTextWithFace(img, text, 2, baseline-1, face, outline)
	drawTextWithFace(img, text, 2, baseline+1, face, outline)
	drawTextWithFace(img, text, 1, baseline, face, outline)
	drawTextWithFace(img, text, 3, baseline, face, outline)
	drawTextWithFace(img, text, 2, baseline, face, foreground)
	outlinedTextCache[key] = img
	if len(outlinedTextCache) > 512 {
		for key := range outlinedTextCache {
			delete(outlinedTextCache, key)
			if len(outlinedTextCache) <= 384 {
				break
			}
		}
	}
	return img
}

func roNameTextFace() font.Face {
	textFontMu.Lock()
	defer textFontMu.Unlock()
	if outlinedTextFace != nil {
		return outlinedTextFace
	}
	if face, err := parseOpenTypeFace(dejavusansbold.TTF, 12); err == nil {
		outlinedTextFace = noKernFace{Face: face}
		return outlinedTextFace
	}
	outlinedTextFace = basicfont.Face7x13
	return outlinedTextFace
}

func parseOpenTypeFace(data []byte, size float64) (font.Face, error) {
	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	return face, nil
}

func textFaceLineMetrics(face font.Face) (int, int) {
	metrics := face.Metrics()
	lineHeight := metrics.Height.Ceil()
	if lineHeight <= 0 {
		lineHeight = (metrics.Ascent + metrics.Descent).Ceil()
	}
	if lineHeight <= 0 {
		lineHeight = 14
	}
	baseline := metrics.Ascent.Ceil()
	if baseline <= 0 {
		baseline = lineHeight - 1
	}
	return lineHeight, baseline
}

func DrawOutlinedTextAt(dst *Image, text string, x, y int, foreground, outline color.RGBA) {
	if dst == nil || text == "" {
		return
	}
	img := OutlinedTextImage(text, foreground, outline)
	var opts DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	opts.Filter = FilterNearest
	dst.DrawImage(img, &opts)
}

func DrawUIOutlinedTextAt(dst *Image, text string, x, y float64, foreground, outline color.RGBA) {
	drawOrQueueUIOutlinedText(dst, text, x, y, foreground, outline, false)
}

func DrawCenteredUIOutlinedTextAt(dst *Image, text string, centerX, y float64, foreground, outline color.RGBA) {
	drawOrQueueUIOutlinedText(dst, text, centerX, y, foreground, outline, true)
}

func drawOrQueueUIOutlinedText(dst *Image, text string, x, y float64, foreground, outline color.RGBA, centered bool) {
	if dst == nil || text == "" {
		return
	}
	if dst.screen {
		dst.uiTextLabels = append(dst.uiTextLabels, UITextLabelCommand{
			Text:       text,
			X:          x,
			Y:          y,
			Foreground: foreground,
			Outline:    outline,
			Centered:   centered,
			Bold:       true,
			Size:       12,
		})
		return
	}
	img := OutlinedTextImage(text, foreground, outline)
	if img == nil {
		return
	}
	if centered {
		x -= float64(img.Bounds().Dx()) / 2
	}
	x, y = snapScreenPoint(dst, x, y)
	var opts DrawImageOptions
	opts.GeoM.Translate(x, y)
	opts.Filter = FilterNearest
	dst.DrawImage(img, &opts)
}

func snapScreenPoint(dst *Image, x, y float64) (float64, float64) {
	if dst == nil || !dst.screen {
		return x, y
	}
	return snapScreenValue(x, float64(dst.screenScaleX)), snapScreenValue(y, float64(dst.screenScaleY))
}

func snapScreenValue(v, scale float64) float64 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return math.Round(v)
	}
	return math.Round(v*scale) / scale
}

func drawTextWithFace(dst *Image, text string, x, y int, face font.Face, c color.RGBA) {
	if dst == nil || dst.pix == nil || text == "" {
		return
	}
	d := &font.Drawer{
		Dst:  dst.pix,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func imageContains(dst *Image, x, y int) bool {
	b := dst.pix.Bounds()
	return x >= b.Min.X && y >= b.Min.Y && x < b.Max.X && y < b.Max.Y
}

func drawSolidQuad(dst *Image, x, y, w, h float64, c color.RGBA) {
	white := WhiteImage()
	r := float32(c.R) / 255
	g := float32(c.G) / 255
	b := float32(c.B) / 255
	a := float32(c.A) / 255
	vertices := []Vertex{
		{DstX: float32(x), DstY: float32(y), SrcX: 0, SrcY: 0, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
		{DstX: float32(x + w), DstY: float32(y), SrcX: 1, SrcY: 0, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
		{DstX: float32(x), DstY: float32(y + h), SrcX: 0, SrcY: 1, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
		{DstX: float32(x + w), DstY: float32(y + h), SrcX: 1, SrcY: 1, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
	}
	dst.DrawTrianglesOwned(vertices, quadIndices012213, white, &DrawTrianglesOptions{Filter: FilterNearest, Address: AddressUnsafe})
}
