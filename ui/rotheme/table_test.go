package rotheme

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestTableHeadCellsUseLabelStyle(t *testing.T) {
	table := Table([]TableRow{{
		{Text: "STR", Width: 40, Align: widget.TextAlignLeft, Head: true},
		{Text: "9", Width: 40, Align: widget.TextAlignRight},
	}}).(*tableWidget)
	table.Layout(widget.NewContext(), geometry.Loose(geometry.Sz(100, 30)))
	canvas := &uitest.MockCanvas{}
	table.Draw(widget.NewContext(), canvas)

	if len(canvas.StyledTexts) != 2 {
		t.Fatalf("styled text draws = %d, want head and value cells", len(canvas.StyledTexts))
	}
	head := canvas.StyledTexts[0]
	if head.Text != "STR" || head.Style.Color != Default.Colors.LabelText || head.Style.FontFamily != Default.Typography.FontFamily || !head.Style.Bold {
		t.Fatalf("head cell draw = %q/%+v, want blue themed bold STR", head.Text, head.Style)
	}
	if canvas.StyledTexts[1].Text != "9" || canvas.StyledTexts[1].Style.FontFamily != Default.Typography.FontFamily {
		t.Fatalf("regular cell draws = %+v, want one value cell", canvas.StyledTexts)
	}
	if len(canvas.Texts) != 0 {
		t.Fatal("table unexpectedly used the built-in font family")
	}
}
