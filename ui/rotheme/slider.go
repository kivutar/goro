package rotheme

import (
	"github.com/gogpu/ui/core/slider"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

const (
	SliderTrackHeight float32 = 6
	SliderThumbSize   float32 = 16
)

func Slider(opts ...slider.Option) *slider.Widget {
	opts = append(opts, slider.PainterOpt(SliderPainter{}))
	return slider.New(opts...).Padding(0)
}

type SliderPainter struct{}

func (SliderPainter) PaintSlider(canvas widget.Canvas, state slider.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}
	if state.Orientation == slider.Vertical {
		paintVerticalSlider(canvas, state)
		return
	}
	paintHorizontalSlider(canvas, state)
}

func paintHorizontalSlider(canvas widget.Canvas, state slider.PaintState) {
	bounds := state.Bounds
	trackY := bounds.Min.Y + bounds.Height()/2
	trackLeft := bounds.Min.X + SliderThumbSize/2
	trackRight := bounds.Max.X - SliderThumbSize/2
	trackWidth := trackRight - trackLeft
	if trackWidth <= 0 {
		return
	}

	thumbX := trackLeft + state.Progress*trackWidth
	track := geometry.NewRect(trackLeft, trackY-SliderTrackHeight/2, trackWidth, SliderTrackHeight)
	paintHorizontalSliderTrack(canvas, track, thumbX-trackLeft, state.Disabled)
	paintSliderThumb(canvas, geometry.Pt(thumbX, trackY), state)
}

func paintVerticalSlider(canvas widget.Canvas, state slider.PaintState) {
	bounds := state.Bounds
	trackX := bounds.Min.X + bounds.Width()/2
	trackTop := bounds.Min.Y + SliderThumbSize/2
	trackBottom := bounds.Max.Y - SliderThumbSize/2
	trackHeight := trackBottom - trackTop
	if trackHeight <= 0 {
		return
	}

	thumbY := trackBottom - state.Progress*trackHeight
	track := geometry.NewRect(trackX-SliderTrackHeight/2, trackTop, SliderTrackHeight, trackHeight)
	paintVerticalSliderTrack(canvas, track, thumbY, trackBottom-thumbY, state.Disabled)
	paintSliderThumb(canvas, geometry.Pt(trackX, thumbY), state)
}

func paintHorizontalSliderTrack(canvas widget.Canvas, track geometry.Rect, filled float32, disabled bool) {
	back := widget.RGBA8(224, 232, 242, 255)
	fill := Default.Colors.WindowBorder
	border := Default.Colors.WindowBorder
	if disabled {
		fill = Default.Colors.Disabled
		border = Default.Colors.Disabled
	}

	canvas.DrawRect(track, back)
	if filled > 0 {
		if filled > track.Width() {
			filled = track.Width()
		}
		canvas.DrawRect(geometry.NewRect(track.Min.X, track.Min.Y, filled, track.Height()), fill)
	}
	canvas.StrokeRect(track, border, 1)
}

func paintVerticalSliderTrack(canvas widget.Canvas, track geometry.Rect, fillTop, filled float32, disabled bool) {
	back := widget.RGBA8(224, 232, 242, 255)
	fill := Default.Colors.WindowBorder
	border := Default.Colors.WindowBorder
	if disabled {
		fill = Default.Colors.Disabled
		border = Default.Colors.Disabled
	}

	canvas.DrawRect(track, back)
	if filled > 0 {
		if filled > track.Height() {
			filled = track.Height()
		}
		canvas.DrawRect(geometry.NewRect(track.Min.X, fillTop, track.Width(), filled), fill)
	}
	canvas.StrokeRect(track, border, 1)
}

func paintSliderThumb(canvas widget.Canvas, center geometry.Point, state slider.PaintState) {
	color := Default.Colors.Button
	if state.Hovered {
		color = Default.Colors.ButtonHover
	}
	if state.Dragging {
		color = Default.Colors.ButtonDown
	}
	if state.Disabled {
		color = Default.Colors.Disabled
	}

	radius := SliderThumbSize / 2
	bounds := geometry.NewRect(center.X-radius, center.Y-radius, SliderThumbSize, SliderThumbSize)
	drawButtonGradient(canvas, bounds, color, radius)
	canvas.StrokeRoundRect(bounds, Default.Colors.ButtonBorder, radius, 1)
	if state.Focused && !state.Disabled {
		canvas.StrokeRoundRect(bounds, Default.Colors.InputFocus, radius, 1)
	}
}

var _ slider.Painter = SliderPainter{}
