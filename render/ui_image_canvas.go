package render

import (
	"image"
	"image/draw"
	"math"
	"reflect"
	"sync"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	xdraw "golang.org/x/image/draw"
)

type scaledImageCanvas struct {
	widget.Canvas
	scale float32
}

func (c scaledImageCanvas) DrawImage(img image.Image, at geometry.Point) {
	if c.Canvas == nil || img == nil {
		return
	}
	bounds := img.Bounds()
	logicalW, logicalH := bounds.Dx(), bounds.Dy()
	if logicalW <= 0 || logicalH <= 0 || c.scale <= 1 {
		c.Canvas.DrawImage(img, at)
		return
	}
	scImg := scaledCanvasSceneImage(img, logicalW, logicalH, c.scale)
	if scImg == nil || scImg.Width <= 0 || scImg.Height <= 0 {
		c.Canvas.DrawImage(img, at)
		return
	}
	at = snapCanvasImagePoint(at, c.Canvas.TransformOffset(), c.scale)
	s := scene.NewScene()
	s.DrawImage(scImg, scene.NewAffine(
		float32(logicalW)/float32(scImg.Width), 0, at.X,
		0, float32(logicalH)/float32(scImg.Height), at.Y,
	))
	c.Canvas.ReplayScene(s)
}

func snapCanvasImagePoint(at, offset geometry.Point, scale float32) geometry.Point {
	if scale <= 0 || math.IsNaN(float64(scale)) || math.IsInf(float64(scale), 0) {
		return at
	}
	physicalScale := float64(scale)
	at.X = float32(math.Round(float64(at.X+offset.X)*physicalScale)/physicalScale) - offset.X
	at.Y = float32(math.Round(float64(at.Y+offset.Y)*physicalScale)/physicalScale) - offset.Y
	return at
}

func (c scaledImageCanvas) FillSVGPath(svgData string, viewBox float32, bounds geometry.Rect, color widget.Color) {
	if filler, ok := c.Canvas.(widget.SVGFiller); ok {
		filler.FillSVGPath(svgData, viewBox, bounds, color)
	}
}

func (c scaledImageCanvas) DrawStyledText(text string, bounds geometry.Rect, style widget.TextStyle) {
	if styled, ok := c.Canvas.(widget.StyledTextDrawer); ok {
		styled.DrawStyledText(text, bounds, style)
		return
	}
	c.Canvas.DrawText(text, bounds, style.FontSize, style.Color, style.Bold, style.Align)
}

func (c scaledImageCanvas) MeasureStyledText(text string, style widget.TextStyle) float32 {
	if styled, ok := c.Canvas.(widget.StyledTextDrawer); ok {
		return styled.MeasureStyledText(text, style)
	}
	return c.Canvas.MeasureText(text, style.FontSize, style.Bold)
}

type scaledCanvasImageKey struct {
	ptr                          uintptr
	srcMinX, srcMinY, srcW, srcH int
	dstW, dstH                   int
}

var scaledCanvasImageCache = struct {
	sync.Mutex
	images map[scaledCanvasImageKey]*scene.Image
}{images: make(map[scaledCanvasImageKey]*scene.Image)}

// scaledImageRasterizer is implemented by CPU-generated images that can
// rasterize themselves directly at the display's physical pixel size. This
// avoids enlarging a low-resolution logical image on HiDPI displays.
type scaledImageRasterizer interface {
	RasterizeForScale(scale float32, width, height int) image.Image
}

func scaledCanvasSceneImage(img image.Image, logicalW, logicalH int, scale float32) *scene.Image {
	if logicalW <= 0 || logicalH <= 0 {
		return nil
	}
	dstW := scaledCanvasPhysicalSize(logicalW, scale)
	dstH := scaledCanvasPhysicalSize(logicalH, scale)
	nativeScale := false
	if rasterizer, ok := img.(scaledImageRasterizer); ok {
		if rasterized := rasterizer.RasterizeForScale(scale, dstW, dstH); rasterized != nil && rasterized.Bounds().Dx() == dstW && rasterized.Bounds().Dy() == dstH {
			img = rasterized
			nativeScale = true
		}
	}
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil
	}
	key := scaledCanvasImageKey{
		ptr:     imagePointer(img),
		srcMinX: bounds.Min.X,
		srcMinY: bounds.Min.Y,
		srcW:    srcW,
		srcH:    srcH,
		dstW:    dstW,
		dstH:    dstH,
	}
	if key.ptr != 0 {
		scaledCanvasImageCache.Lock()
		cached := scaledCanvasImageCache.images[key]
		scaledCanvasImageCache.Unlock()
		if cached != nil {
			return cached
		}
	}

	rgba := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	if nativeScale {
		draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)
	} else {
		xdraw.ApproxBiLinear.Scale(rgba, rgba.Bounds(), img, bounds, xdraw.Src, nil)
	}
	scImg := &scene.Image{Width: dstW, Height: dstH, Data: rgba.Pix}
	if key.ptr != 0 {
		scaledCanvasImageCache.Lock()
		scaledCanvasImageCache.images[key] = scImg
		scaledCanvasImageCache.Unlock()
	}
	return scImg
}

func scaledCanvasPhysicalSize(logical int, scale float32) int {
	if logical <= 0 {
		return 0
	}
	size := int(math.Round(float64(logical) * float64(scale)))
	if size < 1 {
		return 1
	}
	return size
}

func imagePointer(img image.Image) uintptr {
	v := reflect.ValueOf(img)
	if v.Kind() != reflect.Pointer && v.Kind() != reflect.UnsafePointer {
		return 0
	}
	if v.IsNil() {
		return 0
	}
	return v.Pointer()
}
