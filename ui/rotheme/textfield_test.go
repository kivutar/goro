package rotheme

import (
	"testing"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestTextFieldInsetShadowMeetsTopAndSideBorders(t *testing.T) {
	bounds := geometry.NewRect(3, 5, 80, 22)
	canvas := &uitest.MockCanvas{}

	TextFieldPainter{}.PaintTextField(canvas, &textfield.PaintState{Bounds: bounds})

	const shadowRows = 4
	if len(canvas.Rects) != 1+shadowRows {
		t.Fatalf("textfield rectangles = %d, want background and %d shadow rows", len(canvas.Rects), shadowRows)
	}
	for row, call := range canvas.Rects[1:] {
		want := geometry.NewRect(bounds.Min.X, bounds.Min.Y+float32(row), bounds.Width(), 1)
		if call.Bounds != want {
			t.Fatalf("shadow row %d bounds = %v, want %v", row, call.Bounds, want)
		}
	}
	if len(canvas.StrokeRects) != 1 || canvas.StrokeRects[0].Bounds != bounds {
		t.Fatalf("textfield border strokes = %v, want one around %v", canvas.StrokeRects, bounds)
	}
}
