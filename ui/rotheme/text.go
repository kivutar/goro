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

func SectionLabel(content string) *primitives.TextWidget {
	return Title(content).
		Bold()
}
