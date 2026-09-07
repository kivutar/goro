package render

import (
	"fmt"
	"image"
	"math"
	"sync"
	"time"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/geometry"
	uirender "github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
)

type asyncUIRasterizer struct {
	jobs     chan uiRasterJob
	done     chan uiRasterResult
	quit     chan struct{}
	stopOnce sync.Once
}

type uiRasterJob struct {
	list uiDrawList
}

type uiRasterResult struct {
	generation uint64
	width      int
	height     int
	scale      float64
	image      *Image
	rasterDur  time.Duration
	canvasDur  time.Duration
	flushDur   time.Duration
	imageDur   time.Duration
	err        error
}

type uiRasterState struct {
	dc     *gg.Context
	width  int
	height int
	scale  float64
}

func newAsyncUIRasterizer() *asyncUIRasterizer {
	r := &asyncUIRasterizer{
		jobs: make(chan uiRasterJob, 1),
		done: make(chan uiRasterResult, 1),
		quit: make(chan struct{}),
	}
	go r.run()
	return r
}

func (r *asyncUIRasterizer) submit(job uiRasterJob) bool {
	if r == nil {
		return false
	}
	select {
	case r.jobs <- job:
		return true
	default:
		return false
	}
}

func (r *asyncUIRasterizer) stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.quit)
	})
}

func (r *asyncUIRasterizer) run() {
	var state uiRasterState
	defer state.close()
	for {
		select {
		case <-r.quit:
			return
		case job := <-r.jobs:
			result := state.rasterize(job)
			select {
			case r.done <- result:
			case <-r.quit:
				return
			}
		}
	}
}

func (s *uiRasterState) close() {
	if s.dc != nil {
		_ = s.dc.Close()
		s.dc = nil
	}
}

func (s *uiRasterState) ensure(width, height int, scale float64) {
	if scale <= 0 {
		scale = 1
	}
	if s.dc != nil && s.width == width && s.height == height && sameUIScale(s.scale, scale) {
		return
	}
	s.close()
	opts := []gg.ContextOption(nil)
	if !sameUIScale(scale, 1) {
		opts = append(opts, gg.WithDeviceScale(scale))
	}
	s.dc = gg.NewContext(width, height, opts...)
	s.width = width
	s.height = height
	s.scale = scale
}

func (s *uiRasterState) rasterize(job uiRasterJob) uiRasterResult {
	list := job.list
	result := uiRasterResult{
		generation: list.generation,
		width:      list.width,
		height:     list.height,
		scale:      list.scale,
	}
	start := time.Now()
	if list.width <= 0 || list.height <= 0 {
		result.err = fmt.Errorf("invalid ui raster size %dx%d", list.width, list.height)
		return result
	}
	s.ensure(list.width, list.height, list.scale)
	if s.dc == nil {
		result.err = fmt.Errorf("create ui raster context")
		return result
	}

	canvasStart := time.Now()
	baseCanvas := uirender.NewCanvas(s.dc, list.width, list.height)
	canvas := widget.Canvas(scaledImageCanvas{Canvas: baseCanvas, scale: float32(list.scale)})
	if textMode, ok := baseCanvas.(widget.TextModeController); ok {
		textMode.SetTextMode(widget.TextModeVector)
		defer textMode.SetTextMode(widget.TextModeAuto)
	}
	list.replay(canvas)
	result.canvasDur = time.Since(canvasStart)

	flushStart := time.Now()
	if err := s.dc.FlushGPU(); err != nil {
		result.err = fmt.Errorf("flush async ui raster: %w", err)
		return result
	}
	result.flushDur = time.Since(flushStart)

	imageStart := time.Now()
	result.image = imageFromGGContext(s.dc, nil)
	result.imageDur = time.Since(imageStart)
	result.rasterDur = time.Since(start)
	return result
}

func sameUIScale(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

type uiDrawList struct {
	generation uint64
	width      int
	height     int
	scale      float64
	ops        []uiDrawOp
}

type uiDrawOp func(widget.Canvas)

func (l uiDrawList) replay(canvas widget.Canvas) {
	for _, op := range l.ops {
		if op != nil {
			op(canvas)
		}
	}
}

type uiDrawRecorder struct {
	width          int
	height         int
	scale          float64
	ops            []uiDrawOp
	measureDC      *gg.Context
	measureCanvas  widget.Canvas
	clipStack      []geometry.Rect
	currentClip    geometry.Rect
	transformStack []geometry.Point
	currentOffset  geometry.Point
	textMode       widget.TextMode
}

func newUIDrawRecorder(width, height int, scale float64) *uiDrawRecorder {
	if scale <= 0 {
		scale = 1
	}
	opts := []gg.ContextOption(nil)
	if !sameUIScale(scale, 1) {
		opts = append(opts, gg.WithDeviceScale(scale))
	}
	dc := gg.NewContext(1, 1, opts...)
	return &uiDrawRecorder{
		width:          width,
		height:         height,
		scale:          scale,
		ops:            make([]uiDrawOp, 0, 256),
		measureDC:      dc,
		measureCanvas:  uirender.NewCanvas(dc, 1, 1),
		clipStack:      make([]geometry.Rect, 0, 8),
		currentClip:    geometry.NewRect(0, 0, float32(width), float32(height)),
		transformStack: make([]geometry.Point, 0, 8),
		textMode:       widget.TextModeAuto,
	}
}

func (c *uiDrawRecorder) close() {
	if c != nil && c.measureDC != nil {
		_ = c.measureDC.Close()
		c.measureDC = nil
		c.measureCanvas = nil
	}
}

func (c *uiDrawRecorder) list() uiDrawList {
	if c == nil {
		return uiDrawList{}
	}
	ops := make([]uiDrawOp, len(c.ops))
	copy(ops, c.ops)
	return uiDrawList{width: c.width, height: c.height, scale: c.scale, ops: ops}
}

func (c *uiDrawRecorder) append(op uiDrawOp) {
	c.ops = append(c.ops, op)
}

func (c *uiDrawRecorder) Clear(color widget.Color) {
	c.append(func(dst widget.Canvas) {
		dst.Clear(color)
	})
}

func (c *uiDrawRecorder) DrawRect(r geometry.Rect, color widget.Color) {
	c.append(func(dst widget.Canvas) {
		dst.DrawRect(r, color)
	})
}

func (c *uiDrawRecorder) FillRectDirect(r geometry.Rect, color widget.Color) {
	c.append(func(dst widget.Canvas) {
		dst.FillRectDirect(r, color)
	})
}

func (c *uiDrawRecorder) StrokeRect(r geometry.Rect, color widget.Color, strokeWidth float32) {
	c.append(func(dst widget.Canvas) {
		dst.StrokeRect(r, color, strokeWidth)
	})
}

func (c *uiDrawRecorder) DrawRoundRect(r geometry.Rect, color widget.Color, radius float32) {
	c.append(func(dst widget.Canvas) {
		dst.DrawRoundRect(r, color, radius)
	})
}

func (c *uiDrawRecorder) StrokeRoundRect(r geometry.Rect, color widget.Color, radius float32, strokeWidth float32) {
	c.append(func(dst widget.Canvas) {
		dst.StrokeRoundRect(r, color, radius, strokeWidth)
	})
}

func (c *uiDrawRecorder) DrawCircle(center geometry.Point, radius float32, color widget.Color) {
	c.append(func(dst widget.Canvas) {
		dst.DrawCircle(center, radius, color)
	})
}

func (c *uiDrawRecorder) StrokeCircle(center geometry.Point, radius float32, color widget.Color, strokeWidth float32) {
	c.append(func(dst widget.Canvas) {
		dst.StrokeCircle(center, radius, color, strokeWidth)
	})
}

func (c *uiDrawRecorder) StrokeArc(center geometry.Point, radius float32, startAngle, sweepAngle float64, color widget.Color, strokeWidth float32) {
	c.append(func(dst widget.Canvas) {
		dst.StrokeArc(center, radius, startAngle, sweepAngle, color, strokeWidth)
	})
}

func (c *uiDrawRecorder) DrawLine(from, to geometry.Point, color widget.Color, strokeWidth float32) {
	c.append(func(dst widget.Canvas) {
		dst.DrawLine(from, to, color, strokeWidth)
	})
}

func (c *uiDrawRecorder) DrawText(text string, bounds geometry.Rect, fontSize float32, color widget.Color, bold bool, align widget.TextAlign) {
	c.append(func(dst widget.Canvas) {
		dst.DrawText(text, bounds, fontSize, color, bold, align)
	})
}

func (c *uiDrawRecorder) MeasureText(text string, fontSize float32, bold bool) float32 {
	if c.measureCanvas == nil {
		return 0
	}
	if textMode, ok := c.measureCanvas.(widget.TextModeController); ok {
		textMode.SetTextMode(c.textMode)
	}
	return c.measureCanvas.MeasureText(text, fontSize, bold)
}

func (c *uiDrawRecorder) DrawStyledText(text string, bounds geometry.Rect, style widget.TextStyle) {
	c.append(func(dst widget.Canvas) {
		if styled, ok := dst.(widget.StyledTextDrawer); ok {
			styled.DrawStyledText(text, bounds, style)
			return
		}
		dst.DrawText(text, bounds, style.FontSize, style.Color, style.Bold, style.Align)
	})
}

func (c *uiDrawRecorder) MeasureStyledText(text string, style widget.TextStyle) float32 {
	if c.measureCanvas == nil {
		return 0
	}
	if styled, ok := c.measureCanvas.(widget.StyledTextDrawer); ok {
		return styled.MeasureStyledText(text, style)
	}
	return c.measureCanvas.MeasureText(text, style.FontSize, style.Bold)
}

func (c *uiDrawRecorder) DrawImage(img image.Image, at geometry.Point) {
	c.append(func(dst widget.Canvas) {
		dst.DrawImage(img, at)
	})
}

func (c *uiDrawRecorder) PushClip(r geometry.Rect) {
	transformed := r.Translate(c.currentOffset)
	c.clipStack = append(c.clipStack, c.currentClip)
	c.currentClip = c.currentClip.Intersection(transformed)
	c.append(func(dst widget.Canvas) {
		dst.PushClip(r)
	})
}

func (c *uiDrawRecorder) PushClipRoundRect(r geometry.Rect, radius float32) {
	transformed := r.Translate(c.currentOffset)
	c.clipStack = append(c.clipStack, c.currentClip)
	c.currentClip = c.currentClip.Intersection(transformed)
	c.append(func(dst widget.Canvas) {
		dst.PushClipRoundRect(r, radius)
	})
}

func (c *uiDrawRecorder) PopClip() {
	if len(c.clipStack) == 0 {
		return
	}
	last := len(c.clipStack) - 1
	c.currentClip = c.clipStack[last]
	c.clipStack = c.clipStack[:last]
	c.append(func(dst widget.Canvas) {
		dst.PopClip()
	})
}

func (c *uiDrawRecorder) PushTransform(offset geometry.Point) {
	c.transformStack = append(c.transformStack, c.currentOffset)
	c.currentOffset = c.currentOffset.Add(offset)
	c.append(func(dst widget.Canvas) {
		dst.PushTransform(offset)
	})
}

func (c *uiDrawRecorder) PopTransform() {
	if len(c.transformStack) == 0 {
		return
	}
	last := len(c.transformStack) - 1
	c.currentOffset = c.transformStack[last]
	c.transformStack = c.transformStack[:last]
	c.append(func(dst widget.Canvas) {
		dst.PopTransform()
	})
}

func (c *uiDrawRecorder) TransformOffset() geometry.Point {
	return c.currentOffset
}

func (c *uiDrawRecorder) ScreenOriginBase() geometry.Point {
	return geometry.Point{}
}

func (c *uiDrawRecorder) ClipBounds() geometry.Rect {
	return c.currentClip
}

func (c *uiDrawRecorder) ReplayScene(cache widget.SceneCache) {
	s, ok := cache.(*scene.Scene)
	if !ok || s == nil || s.IsEmpty() {
		return
	}
	snapshot := scene.NewScene()
	snapshot.Append(s)
	c.append(func(dst widget.Canvas) {
		dst.ReplayScene(snapshot)
	})
}

func (c *uiDrawRecorder) FillSVGPath(svgData string, viewBox float32, bounds geometry.Rect, color widget.Color) {
	c.append(func(dst widget.Canvas) {
		if filler, ok := dst.(widget.SVGFiller); ok {
			filler.FillSVGPath(svgData, viewBox, bounds, color)
		}
	})
}

func (c *uiDrawRecorder) setTextMode(mode widget.TextMode) {
	c.textMode = mode
	if textMode, ok := c.measureCanvas.(widget.TextModeController); ok {
		textMode.SetTextMode(mode)
	}
}

func imageFromGGContext(dc *gg.Context, dstImage *Image) *Image {
	if dc == nil || dc.ResizeTarget() == nil {
		return dstImage
	}
	return updateRGBAImage(dc.ResizeTarget().ImageView(), dstImage)
}

func updateRGBAImage(src *image.RGBA, dstImage *Image) *Image {
	if src == nil {
		return dstImage
	}
	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	if dstImage == nil || dstImage.pix == nil || dstImage.Bounds().Dx() != width || dstImage.Bounds().Dy() != height {
		dstImage = NewImage(width, height)
	}
	dst := dstImage.pix
	if src.Stride == width*4 && dst.Stride == width*4 {
		copy(dst.Pix, src.Pix)
	} else {
		for y := 0; y < height; y++ {
			copy(dst.Pix[y*dst.Stride:y*dst.Stride+width*4], src.Pix[y*src.Stride:y*src.Stride+width*4])
		}
	}
	dstImage.version++
	return dstImage
}

var (
	_ widget.Canvas           = (*uiDrawRecorder)(nil)
	_ widget.StyledTextDrawer = (*uiDrawRecorder)(nil)
	_ widget.SVGFiller        = (*uiDrawRecorder)(nil)
)
