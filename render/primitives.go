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

func DrawRect(dst *Frame, x, y, w, h float64, c color.Color) {
	if dst == nil || w <= 0 || h <= 0 {
		return
	}
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	x0, y0 := snapScreenPoint(dst, x, y)
	x1, y1 := snapScreenPoint(dst, x+w, y+h)
	w, h = x1-x0, y1-y0
	if w <= 0 || h <= 0 {
		return
	}
	drawSolidQuad(dst, x0, y0, w, h, rgba)
}

func DrawImageRect(dst *Image, x, y, w, h float64, c color.Color) {
	if dst == nil || dst.pix == nil || w <= 0 || h <= 0 {
		return
	}
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
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

func DrawUIRect(dst *Frame, x, y, w, h float64, c color.RGBA) {
	if dst == nil || w <= 0 || h <= 0 {
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

func DrawUISpeechBubble(dst *Frame, text string, centerX, bottomY, maxWidth float64) {
	if dst == nil || text == "" {
		return
	}
	if maxWidth <= 0 {
		maxWidth = 220
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

func DrawUITooltip(dst *Frame, text string, centerX, belowY, aboveY float64) {
	DrawUITooltipBox(dst, text, centerX, belowY, aboveY, 0, 1)
}

func DrawUITooltipBox(dst *Frame, text string, centerX, belowY, aboveY, maxWidth float64, maxLines int) {
	if dst == nil || text == "" {
		return
	}
	dst.uiTextBoxes = append(dst.uiTextBoxes, UITextBoxCommand{
		Text:     text,
		X:        centerX,
		Y:        belowY,
		AltY:     aboveY,
		Anchor:   UITextBoxAnchorTooltipCenter,
		MaxWidth: maxWidth,
		MaxLines: maxLines,
	})
}

func DrawLine(dst *Frame, x0, y0, x1, y1 float64, c color.Color) {
	if dst == nil {
		return
	}
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0)))
	if steps <= 0 {
		drawSolidQuad(dst, math.Round(x0), math.Round(y0), 1, 1, rgba)
		return
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		drawSolidQuad(dst, math.Round(x0+(x1-x0)*t), math.Round(y0+(y1-y0)*t), 1, 1, rgba)
	}
}

func DrawBitmapTextAt(dst *Frame, text string, x, y int) {
	DrawBitmapTextAtColor(dst, text, x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

func DrawBitmapTextAtColor(dst *Frame, text string, x, y int, c color.RGBA) {
	if dst == nil || text == "" {
		return
	}
	img := cachedBitmapTextColor(text, c)
	var opts DrawImageOptions
	sx, sy := snapScreenPoint(dst, float64(x), float64(y))
	opts.GeoM.Translate(sx, sy)
	opts.Filter = FilterNearest
	dst.DrawImage(img, &opts)
}

func DrawImageBitmapTextAtColor(dst *Image, text string, x, y int, c color.RGBA) {
	if dst == nil || dst.pix == nil || text == "" {
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

func cachedBitmapTextColor(text string, c color.RGBA) *Image {
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
	DrawImageBitmapTextAtColor(img, text, 0, y, c)
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

func BitmapTextSize(text string) (int, int) {
	return debugTextSize(text)
}

func BitmapTextTopForCenter(containerH int) int {
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

func DrawOutlinedTextAt(dst *Frame, text string, x, y int, foreground, outline color.RGBA) {
	if dst == nil || text == "" {
		return
	}
	img := OutlinedTextImage(text, foreground, outline)
	var opts DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	opts.Filter = FilterNearest
	dst.DrawImage(img, &opts)
}

func DrawUIOutlinedTextAt(dst *Frame, text string, x, y float64, foreground, outline color.RGBA) {
	drawOrQueueUITextLabel(dst, text, x, y, foreground, outline, false, true, 12)
}

func DrawCenteredUIOutlinedTextAt(dst *Frame, text string, centerX, y float64, foreground, outline color.RGBA) {
	drawOrQueueUITextLabel(dst, text, centerX, y, foreground, outline, true, true, 12)
}

func DrawCenteredUIOutlinedTextAtSize(dst *Frame, text string, centerX, y float64, foreground, outline color.RGBA, size float32, bold bool) {
	drawOrQueueUITextLabel(dst, text, centerX, y, foreground, outline, true, bold, size)
}

func DrawActorUILabels(dst *Frame, labels []string, emblem *Image, centerX, y float64, foreground, outline color.RGBA) {
	if dst == nil || len(labels) == 0 {
		return
	}
	copied := make([]string, 0, len(labels))
	for _, label := range labels {
		if label != "" {
			copied = append(copied, label)
		}
	}
	if len(copied) == 0 {
		return
	}
	dst.uiActorLabels = append(dst.uiActorLabels, UIActorLabelCommand{
		Labels:     copied,
		Emblem:     emblem,
		CenterX:    centerX,
		Y:          y,
		Foreground: foreground,
		Outline:    outline,
		Size:       12,
	})
}

func DrawUITextAt(dst *Frame, text string, x, y float64, foreground color.RGBA) {
	drawOrQueueUITextLabel(dst, text, x, y, foreground, color.RGBA{}, false, false, 12)
}

func DrawCenteredUITextAt(dst *Frame, text string, centerX, y float64, foreground color.RGBA) {
	drawOrQueueUITextLabel(dst, text, centerX, y, foreground, color.RGBA{}, true, false, 12)
}

func drawOrQueueUITextLabel(dst *Frame, text string, x, y float64, foreground, outline color.RGBA, centered, bold bool, size float32) {
	if dst == nil || text == "" {
		return
	}
	dst.uiTextLabels = append(dst.uiTextLabels, UITextLabelCommand{
		Text:       text,
		X:          x,
		Y:          y,
		Foreground: foreground,
		Outline:    outline,
		Centered:   centered,
		Bold:       bold,
		Size:       size,
	})
}

func snapScreenPoint(dst *Frame, x, y float64) (float64, float64) {
	if dst == nil {
		return x, y
	}
	return snapScreenValue(x, float64(dst.screenScaleX)), snapScreenValue(y, float64(dst.screenScaleY))
}

func SnapScreenPoint(dst *Frame, x, y float64) (float64, float64) {
	return snapScreenPoint(dst, x, y)
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

func drawSolidQuad(dst *Frame, x, y, w, h float64, c color.RGBA) {
	if dst == nil {
		return
	}
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
