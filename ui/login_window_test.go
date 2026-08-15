package ui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

func TestLoginWindowLabelsFillRightAlignedColumn(t *testing.T) {
	width, height := loginWindowSize()
	window := &LoginWindow{layout: loginWindowLayout{W: width, H: height}}
	tree := window.widgetTree()
	tree.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(float32(width), float32(height))))

	windowChildren := tree.Children()
	if len(windowChildren) < 2 || len(windowChildren[1].Children()) != 1 {
		t.Fatal("login window content tree is incomplete")
	}
	rows := windowChildren[1].Children()[0].Children()
	if len(rows) != 2 {
		t.Fatalf("login form rows = %d, want Account and Password", len(rows))
	}

	for i, want := range []string{"Account", "Password"} {
		rowChildren := rows[i].Children()
		if len(rowChildren) != 2 || len(rowChildren[0].Children()) != 1 {
			t.Fatalf("%s row does not contain a label and field", want)
		}
		labelSlot := rowChildren[0]
		label, ok := labelSlot.Children()[0].(*primitives.TextWidget)
		if !ok {
			t.Fatalf("%s label = %T, want text", want, labelSlot.Children()[0])
		}
		if label.Content() != want {
			t.Fatalf("login label = %q, want %q", label.Content(), want)
		}
		if label.Style().Align != widget.TextAlignRight {
			t.Fatalf("%s alignment = %v, want right", want, label.Style().Align)
		}

		slotBounds := labelSlot.(interface{ Bounds() geometry.Rect }).Bounds()
		labelBounds := label.Bounds()
		if labelBounds.Min.X != 0 || labelBounds.Width() != slotBounds.Width() {
			t.Fatalf("%s label bounds = %v, want full %.1fpx column width", want, labelBounds, slotBounds.Width())
		}
	}
}
