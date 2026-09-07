package render

import (
	"testing"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestUIDrawRecorderIgnoresTypedNilScene(t *testing.T) {
	recorder := newUIDrawRecorder(320, 240, 1)
	defer recorder.close()

	recorder.ReplayScene((*scene.Scene)(nil))
	if got := len(recorder.list().ops); got != 0 {
		t.Fatalf("recorded operations = %d, want 0", got)
	}
}

func TestUIDrawRecorderReplaysStyledText(t *testing.T) {
	recorder := newUIDrawRecorder(320, 240, 1)
	defer recorder.close()

	bounds := geometry.NewRect(4, 8, 80, 16)
	style := widget.TextStyle{
		FontFamily: "Inter",
		FontSize:   11,
		Color:      widget.ColorBlack,
		Align:      widget.TextAlignRight,
	}
	recorder.DrawStyledText("Account", bounds, style)

	dst := &uitest.MockCanvas{}
	recorder.list().replay(dst)
	if len(dst.StyledTexts) != 1 {
		t.Fatalf("styled text calls = %d, want 1", len(dst.StyledTexts))
	}
	call := dst.StyledTexts[0]
	if call.Text != "Account" || call.Bounds != bounds || call.Style != style {
		t.Fatalf("styled text call = %+v, want text=%q bounds=%+v style=%+v", call, "Account", bounds, style)
	}
}

func TestScaledImageCanvasForwardsStyledText(t *testing.T) {
	dst := &uitest.MockCanvas{}
	canvas := scaledImageCanvas{Canvas: dst, scale: 1.5}
	style := widget.TextStyle{FontFamily: "Inter", FontSize: 11, Bold: true}

	canvas.DrawStyledText("Display", geometry.NewRect(0, 0, 80, 16), style)
	if len(dst.StyledTexts) != 1 {
		t.Fatalf("styled text calls = %d, want 1", len(dst.StyledTexts))
	}
	if got := canvas.MeasureStyledText("Display", style); got <= 0 {
		t.Fatalf("styled text width = %v, want > 0", got)
	}
}
