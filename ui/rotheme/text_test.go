package rotheme

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestLabelUsesBlueThemedBoldFace(t *testing.T) {
	style := Label("Account").Style()

	if style.FontFamily != Default.Typography.FontFamily || !style.Bold {
		t.Fatalf("label font = family %q, bold %t; want %q bold", style.FontFamily, style.Bold, Default.Typography.FontFamily)
	}
	if style.Color != Default.Colors.LabelText {
		t.Fatalf("label color = %+v, want %+v", style.Color, Default.Colors.LabelText)
	}
}

func TestDrawLabelUsesBlueThemedBoldFace(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	DrawLabel(canvas, "STR", geometry.NewRect(1, 2, 30, 18), widget.TextAlignCenter)

	if len(canvas.StyledTexts) != 1 {
		t.Fatalf("styled label draws = %d, want 1", len(canvas.StyledTexts))
	}
	draw := canvas.StyledTexts[0]
	if draw.Style.Color != Default.Colors.LabelText || draw.Style.FontFamily != Default.Typography.FontFamily || !draw.Style.Bold {
		t.Fatalf("label draw style = %+v, want blue %q bold text", draw.Style, Default.Typography.FontFamily)
	}
	if draw.Style.FontSize != Default.Typography.TextSize || draw.Style.Align != widget.TextAlignCenter {
		t.Fatalf("label draw size/alignment = %.1f/%v", draw.Style.FontSize, draw.Style.Align)
	}
	if len(canvas.Texts) != 0 {
		t.Fatal("label draw unexpectedly used the built-in font family")
	}
}
