package ui

import (
	"image"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

type opacityWidget struct {
	widget.WidgetBase
	child   widget.Widget
	opacity float32
}

func withOpacity(child widget.Widget, opacity float32) widget.Widget {
	if child == nil || opacity >= 1 {
		return child
	}
	if opacity < 0 {
		opacity = 0
	}
	w := &opacityWidget{
		child:   child,
		opacity: opacity,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *opacityWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	if w.child == nil {
		return geometry.Size{}
	}
	size := w.child.Layout(ctx, constraints)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *opacityWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() || w.child == nil {
		return
	}
	widget.DrawChild(w.child, ctx, alphaCanvas{Canvas: canvas, alpha: w.opacity})
}

func (w *opacityWidget) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() || w.child == nil {
		return false
	}
	return w.child.Event(ctx, e)
}

func (w *opacityWidget) Children() []widget.Widget {
	if w.child == nil {
		return nil
	}
	return []widget.Widget{w.child}
}

type alphaCanvas struct {
	widget.Canvas
	alpha float32
}

func (c alphaCanvas) color(color widget.Color) widget.Color {
	color.A *= c.alpha
	return color
}

func (c alphaCanvas) Clear(color widget.Color) {
	c.Canvas.Clear(c.color(color))
}

func (c alphaCanvas) DrawRect(r geometry.Rect, color widget.Color) {
	c.Canvas.DrawRect(r, c.color(color))
}

func (c alphaCanvas) FillRectDirect(r geometry.Rect, color widget.Color) {
	c.Canvas.FillRectDirect(r, c.color(color))
}

func (c alphaCanvas) StrokeRect(r geometry.Rect, color widget.Color, strokeWidth float32) {
	c.Canvas.StrokeRect(r, c.color(color), strokeWidth)
}

func (c alphaCanvas) DrawRoundRect(r geometry.Rect, color widget.Color, radius float32) {
	c.Canvas.DrawRoundRect(r, c.color(color), radius)
}

func (c alphaCanvas) StrokeRoundRect(r geometry.Rect, color widget.Color, radius float32, strokeWidth float32) {
	c.Canvas.StrokeRoundRect(r, c.color(color), radius, strokeWidth)
}

func (c alphaCanvas) DrawCircle(center geometry.Point, radius float32, color widget.Color) {
	c.Canvas.DrawCircle(center, radius, c.color(color))
}

func (c alphaCanvas) StrokeCircle(center geometry.Point, radius float32, color widget.Color, strokeWidth float32) {
	c.Canvas.StrokeCircle(center, radius, c.color(color), strokeWidth)
}

func (c alphaCanvas) StrokeArc(center geometry.Point, radius float32, startAngle, sweepAngle float64, color widget.Color, strokeWidth float32) {
	c.Canvas.StrokeArc(center, radius, startAngle, sweepAngle, c.color(color), strokeWidth)
}

func (c alphaCanvas) DrawLine(from, to geometry.Point, color widget.Color, strokeWidth float32) {
	c.Canvas.DrawLine(from, to, c.color(color), strokeWidth)
}

func (c alphaCanvas) DrawText(text string, bounds geometry.Rect, fontSize float32, color widget.Color, bold bool, align widget.TextAlign) {
	c.Canvas.DrawText(text, bounds, fontSize, c.color(color), bold, align)
}

func (c alphaCanvas) DrawImage(img image.Image, at geometry.Point) {
	c.Canvas.DrawImage(img, at)
}

func (c alphaCanvas) ReplayScene(s *scene.Scene) {
	c.Canvas.ReplayScene(s)
}

func (c alphaCanvas) DrawStyledText(text string, bounds geometry.Rect, style widget.TextStyle) {
	style.Color = c.color(style.Color)
	if styled, ok := c.Canvas.(widget.StyledTextDrawer); ok {
		styled.DrawStyledText(text, bounds, style)
		return
	}
	c.DrawText(text, bounds, style.FontSize, style.Color, style.Bold, style.Align)
}

func (c alphaCanvas) MeasureStyledText(text string, style widget.TextStyle) float32 {
	if styled, ok := c.Canvas.(widget.StyledTextDrawer); ok {
		return styled.MeasureStyledText(text, style)
	}
	return c.MeasureText(text, style.FontSize, style.Bold)
}

func (c alphaCanvas) StrokeArcStyled(center geometry.Point, radius float32, startAngle, sweepAngle float64, color widget.Color, strokeWidth float32, lineCap widget.LineCap) {
	if stroker, ok := c.Canvas.(widget.ArcStroker); ok {
		stroker.StrokeArcStyled(center, radius, startAngle, sweepAngle, c.color(color), strokeWidth, lineCap)
		return
	}
	c.StrokeArc(center, radius, startAngle, sweepAngle, color, strokeWidth)
}

func (c alphaCanvas) FillSVGPath(svgData string, viewBox float32, bounds geometry.Rect, color widget.Color) {
	if filler, ok := c.Canvas.(widget.SVGFiller); ok {
		filler.FillSVGPath(svgData, viewBox, bounds, c.color(color))
	}
}

func (c alphaCanvas) RenderSVG(svgXML []byte, bounds geometry.Rect, color widget.Color) {
	if renderer, ok := c.Canvas.(widget.SVGRenderer); ok {
		renderer.RenderSVG(svgXML, bounds, c.color(color))
	}
}

func (c alphaCanvas) SetTextMode(mode widget.TextMode) {
	if controller, ok := c.Canvas.(widget.TextModeController); ok {
		controller.SetTextMode(mode)
	}
}

func (c alphaCanvas) TextMode() widget.TextMode {
	if controller, ok := c.Canvas.(widget.TextModeController); ok {
		return controller.TextMode()
	}
	return widget.TextModeAuto
}

func (c alphaCanvas) IsBoundaryRecording() bool {
	if recorder, ok := c.Canvas.(widget.BoundaryRecorder); ok {
		return recorder.IsBoundaryRecording()
	}
	return false
}

var _ widget.Canvas = alphaCanvas{}
var _ widget.StyledTextDrawer = alphaCanvas{}
var _ widget.ArcStroker = alphaCanvas{}
var _ widget.SVGFiller = alphaCanvas{}
var _ widget.SVGRenderer = alphaCanvas{}
var _ widget.TextModeController = alphaCanvas{}
var _ widget.BoundaryRecorder = alphaCanvas{}
