package rotheme

import (
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

func Text(content string) *primitives.TextWidget {
	return primitives.Text(content).
		FontFamily(Default.Typography.FontFamily).
		FontSize(Default.Typography.TextSize).
		Color(Default.Colors.Text)
}

func Title(content string) *primitives.TextWidget {
	return Text(content).
		Color(Default.Colors.TitleText)
}

func Label(content string) *primitives.TextWidget {
	// gogpu/ui's built-in family registers regular and bold as distinct weights.
	// Goro's custom DejaVu faces are registered as separate families.
	return primitives.Text(content).
		FontSize(Default.Typography.TextSize).
		Color(Default.Colors.LabelText).
		Bold()
}

// DrawLabel renders the semantic label style on a canvas.
func DrawLabel(canvas widget.Canvas, content string, bounds geometry.Rect, align widget.TextAlign) {
	if content == "" {
		return
	}
	canvas.DrawText(content, bounds, Default.Typography.TextSize, Default.Colors.LabelText, true, align)
}
