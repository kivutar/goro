package rotheme

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestLabelUsesBlueBoldStyle(t *testing.T) {
	style := Label("Account").Style()

	if !style.Bold {
		t.Fatal("label should request native bold text")
	}
	if style.FontFamily != "" {
		t.Fatalf("label font family = %q, want gogpu/ui's built-in weighted family", style.FontFamily)
	}
	if style.Color != Default.Colors.LabelText {
		t.Fatalf("label color = %+v, want %+v", style.Color, Default.Colors.LabelText)
	}
}

func TestDrawLabelUsesBlueNativeBoldStyle(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	DrawLabel(canvas, "STR", geometry.NewRect(1, 2, 30, 18), widget.TextAlignCenter)

	if len(canvas.Texts) != 1 {
		t.Fatalf("native label draws = %d, want 1", len(canvas.Texts))
	}
	draw := canvas.Texts[0]
	if draw.Color != Default.Colors.LabelText || !draw.Bold {
		t.Fatalf("label draw color/bold = %+v/%t, want %+v/true", draw.Color, draw.Bold, Default.Colors.LabelText)
	}
	if draw.FontSize != Default.Typography.TextSize || draw.Align != widget.TextAlignCenter {
		t.Fatalf("label draw size/alignment = %.1f/%v", draw.FontSize, draw.Align)
	}
	if len(canvas.StyledTexts) != 0 {
		t.Fatal("label draw unexpectedly used the custom DejaVu family")
	}
}
