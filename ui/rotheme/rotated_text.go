package rotheme

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"github.com/gogpu/ui/fontdata"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type TextRotation uint8

const (
	TextRotationNone TextRotation = iota
	TextRotationCounterClockwise
	TextRotationClockwise
)

type rotatedTextImageKey struct {
	text     string
	size     float32
	color    widget.Color
	bold     bool
	rotation TextRotation
}

type scaledRotatedTextImageKey struct {
	scale         uint32
	width, height int
}

type scaledRotatedTextImage struct {
	*image.RGBA
	key rotatedTextImageKey

	sync.Mutex
	images map[scaledRotatedTextImageKey]image.Image
}

var rotatedTextImages = struct {
	sync.Mutex
	images map[rotatedTextImageKey]image.Image
}{images: make(map[rotatedTextImageKey]image.Image)}

// DrawRotatedText draws centered text at a right-angle rotation. Horizontal
// text continues through DrawText so it retains the canvas's native shaping.
func DrawRotatedText(canvas widget.Canvas, text string, bounds geometry.Rect, size float32, textColor widget.Color, bold bool, rotation TextRotation) {
	if rotation == TextRotationNone {
		DrawText(canvas, text, bounds, size, textColor, bold, widget.TextAlignCenter)
		return
	}
	if text == "" {
		return
	}
	img := rotatedTextImage(text, size, textColor, bold, rotation)
	if img == nil {
		return
	}
	imageBounds := img.Bounds()
	x := bounds.Min.X + (bounds.Width()-float32(imageBounds.Dx()))/2
	y := bounds.Min.Y + (bounds.Height()-float32(imageBounds.Dy()))/2
	canvas.DrawImage(img, geometry.Pt(
		float32(math.Round(float64(x))),
		float32(math.Round(float64(y))),
	))
}

func rotatedTextImage(text string, size float32, textColor widget.Color, bold bool, rotation TextRotation) image.Image {
	key := rotatedTextImageKey{text: text, size: size, color: textColor, bold: bold, rotation: rotation}
	rotatedTextImages.Lock()
	defer rotatedTextImages.Unlock()
	if cached := rotatedTextImages.images[key]; cached != nil {
		return cached
	}

	base := rasterizeRotatedText(key, 1)
	if base == nil {
		return nil
	}
	rotated := &scaledRotatedTextImage{
		RGBA:   base,
		key:    key,
		images: make(map[scaledRotatedTextImageKey]image.Image),
	}
	rotatedTextImages.images[key] = rotated
	if len(rotatedTextImages.images) > 128 {
		for oldKey := range rotatedTextImages.images {
			delete(rotatedTextImages.images, oldKey)
			if len(rotatedTextImages.images) <= 96 {
				break
			}
		}
	}
	return rotated
}

// RasterizeForScale supplies the renderer with an image produced directly at
// the display scale. width and height are the exact physical target size.
func (img *scaledRotatedTextImage) RasterizeForScale(scale float32, width, height int) image.Image {
	if img == nil || width <= 0 || height <= 0 {
		return nil
	}
	if scale <= 1 && width == img.Bounds().Dx() && height == img.Bounds().Dy() {
		return img.RGBA
	}
	key := scaledRotatedTextImageKey{scale: math.Float32bits(scale), width: width, height: height}
	img.Lock()
	defer img.Unlock()
	if cached := img.images[key]; cached != nil {
		return cached
	}

	rasterized := rasterizeRotatedText(img.key, scale)
	if rasterized == nil {
		return nil
	}
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	x := (width - rasterized.Bounds().Dx()) / 2
	y := (height - rasterized.Bounds().Dy()) / 2
	destination := image.Rect(x, y, x+rasterized.Bounds().Dx(), y+rasterized.Bounds().Dy()).Intersect(target.Bounds())
	source := image.Pt(max(0, -x), max(0, -y))
	draw.Draw(target, destination, rasterized, source, draw.Src)
	img.images[key] = target
	return target
}

func rasterizeRotatedText(key rotatedTextImageKey, scale float32) *image.RGBA {
	if scale <= 0 {
		scale = 1
	}
	fontData := fontdata.InterRegular
	if key.bold {
		fontData = fontdata.InterBold
	}
	parsed, err := opentype.Parse(fontData)
	if err != nil {
		return nil
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    float64(key.size * scale),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil
	}
	defer face.Close()

	padding := max(1, int(math.Round(float64(scale))))
	metrics := face.Metrics()
	sourceWidth := font.MeasureString(face, key.text).Ceil() + padding*2
	sourceHeight := (metrics.Ascent + metrics.Descent).Ceil() + padding*2
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return nil
	}
	r, g, b, a := key.color.RGBA8()
	source := image.NewRGBA(image.Rect(0, 0, sourceWidth, sourceHeight))
	drawer := font.Drawer{
		Dst:  source,
		Src:  image.NewUniform(color.RGBA{R: r, G: g, B: b, A: a}),
		Face: face,
		Dot:  fixed.P(padding, padding+metrics.Ascent.Ceil()),
	}
	drawer.DrawString(key.text)
	return rotateRightAngle(source, key.rotation)
}

func rotateRightAngle(source *image.RGBA, rotation TextRotation) *image.RGBA {
	if source == nil || (rotation != TextRotationCounterClockwise && rotation != TextRotationClockwise) {
		return nil
	}
	sourceBounds := source.Bounds()
	width, height := sourceBounds.Dx(), sourceBounds.Dy()
	destination := image.NewRGBA(image.Rect(0, 0, height, width))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := source.RGBAAt(sourceBounds.Min.X+x, sourceBounds.Min.Y+y)
			if rotation == TextRotationCounterClockwise {
				destination.SetRGBA(y, width-1-x, pixel)
			} else {
				destination.SetRGBA(height-1-y, x, pixel)
			}
		}
	}
	return destination
}
