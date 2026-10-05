package render

import (
	"bytes"
	"fmt"
	"image"
	"testing"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestUIPartialButtonRepaintMatchesFull(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 4.0 / 3, 1.5, 1.75, 2, 2.3} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			var state uiRasterState
			t.Cleanup(state.close)
			rasterize := func(damage *geometry.Rect) *image.RGBA {
				recorder := newUIDrawRecorder(180, 100, scale)
				defer recorder.close()
				if damage == nil {
					recorder.Clear(widget.ColorTransparent)
				} else {
					recorder.FillRectDirect(*damage, widget.ColorTransparent)
					recorder.PushClip(*damage)
				}
				recorder.DrawRoundRect(geometry.NewRect(5, 5, 170, 90), widget.ColorWhite, 6)
				rotheme.ButtonPainter{}.PaintButton(recorder, button.PaintState{
					Bounds: geometry.NewRect(20, 20, 80, 22), Text: "Hover",
				})
				if damage != nil {
					recorder.PopClip()
				}
				result := state.rasterize(uiRasterJob{list: recorder.list()})
				if result.err != nil {
					t.Fatal(result.err)
				}
				return result.image.pix
			}
			want := rasterize(nil)
			// Cut through the gradient, rounded clip, border and shadow, as
			// happens when a different widget or overlapping window repaints.
			for _, y := range []float32{19.5, 20, 21, 41, 42, 42.5, 43, 43.5} {
				damage := geometry.NewRect(65.5, y, 28, 18).Expand(1)
				got := rasterize(&damage)
				if !bytes.Equal(got.Pix, want.Pix) {
					for py := range got.Bounds().Dy() {
						for px := range got.Bounds().Dx() {
							if a, b := got.RGBAAt(px, py), want.RGBAAt(px, py); a != b {
								t.Fatalf("damage %v: pixel (%d,%d) = %v, full repaint = %v", damage, px, py, a, b)
							}
						}
					}
				}
			}
		})
	}
}
