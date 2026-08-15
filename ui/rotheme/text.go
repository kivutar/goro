package rotheme

import "github.com/gogpu/ui/primitives"

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
