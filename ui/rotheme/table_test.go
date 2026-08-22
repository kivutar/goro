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

	if len(canvas.Texts) != 1 {
		t.Fatalf("native label draws = %d, want one head cell", len(canvas.Texts))
	}
	head := canvas.Texts[0]
	if head.Text != "STR" || head.Color != Default.Colors.LabelText || !head.Bold {
		t.Fatalf("head cell draw = %q/%+v/%t, want blue bold STR", head.Text, head.Color, head.Bold)
	}
	if len(canvas.StyledTexts) != 1 || canvas.StyledTexts[0].Text != "9" {
		t.Fatalf("regular cell draws = %+v, want one value cell", canvas.StyledTexts)
	}
}
