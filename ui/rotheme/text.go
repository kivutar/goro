package rotheme

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
)

type TextWidget struct {
	widget.WidgetBase
	content    string
	color      widget.Color
	bold       bool
	fontSize   float32
	lineHeight float32
}

var (
	classicTextMu    sync.Mutex
	classicTextCache = map[string]image.Image{}
)

func Text(content string) *TextWidget {
	t := &TextWidget{
		content:    content,
		color:      Default.Colors.Text,
		fontSize:   Default.Typography.TextSize,
		lineHeight: 1.2,
	}
	t.SetVisible(true)
	t.SetEnabled(true)
	return t
}

func Title(content string) *TextWidget {
	return Text(content).Color(Default.Colors.TitleText)
}

func SectionLabel(content string) *TextWidget {
	return Title(content).Bold()
}

func (t *TextWidget) FontSize(size float32) *TextWidget {
	t.fontSize = size
	return t
}

func (t *TextWidget) Color(c widget.Color) *TextWidget {
	t.color = c
	return t
}

func (t *TextWidget) Bold() *TextWidget {
	t.bold = true
	return t
}

func (t *TextWidget) FontFamily(_ string) *TextWidget {
	return t
}

func (t *TextWidget) LineHeight(v float32) *TextWidget {
	t.lineHeight = v
	return t
}

func (t *TextWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	w, h := render.DebugTextSize(t.content)
	if t.lineHeight > 0 {
		h = maxInt(h, int(math.Ceil(float64(t.fontSize*t.lineHeight))))
	}
	size := constraints.Constrain(geometry.Sz(float32(w), float32(h)))
	t.SetBounds(geometry.FromPointSize(t.Position(), size))
	return size
}

func (t *TextWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	if !t.IsVisible() || t.content == "" {
		return
	}
	bounds := t.Bounds()
	if bounds.IsEmpty() {
		return
	}
	if t.bold {
		DrawText(canvas, t.content, bounds, t.fontSize, t.color, true, widget.TextAlignLeft)
		return
	}
	img := classicTextImage(t.content, widgetToRGBA(t.color))
	if img == nil {
		return
	}
	imageBounds := img.Bounds()
	x := float32(math.Round(float64(bounds.Min.X)))
	y := float32(math.Round(float64(bounds.Min.Y + (bounds.Height()-float32(imageBounds.Dy()))*0.5)))
	canvas.DrawImage(img, geometry.Pt(x, y))
}

func (t *TextWidget) Event(widget.Context, event.Event) bool {
	return false
}

func (t *TextWidget) Children() []widget.Widget {
	return nil
}

func classicTextImage(text string, c color.RGBA) image.Image {
	key := fmt.Sprintf("%02x%02x%02x%02x:%s", c.R, c.G, c.B, c.A, text)
	classicTextMu.Lock()
	defer classicTextMu.Unlock()
	if img := classicTextCache[key]; img != nil {
		return img
	}
	w, h := render.DebugTextSize(text)
	img := render.NewImage(w, h)
	render.DebugPrintAtColor(img, text, 0, 0, c)
	rgba := image.Image(img.RGBA())
	classicTextCache[key] = rgba
	if len(classicTextCache) > 512 {
		classicTextCache = map[string]image.Image{}
	}
	return rgba
}

func widgetToRGBA(c widget.Color) color.RGBA {
	return color.RGBA{
		R: uint8(clamp01(c.R)*255 + 0.5),
		G: uint8(clamp01(c.G)*255 + 0.5),
		B: uint8(clamp01(c.B)*255 + 0.5),
		A: uint8(clamp01(c.A)*255 + 0.5),
	}
}

func clamp01(v float32) float32 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
