package ui

import (
	"fmt"

	"github.com/gogpu/ui/core/progressbar"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	vitalsLabelWidth   float32 = 22
	vitalsPercentWidth float32 = 34
	vitalsGap          float32 = 6
)

// vitalsRow is shared by the player, homunculus and mercenary info windows.
func vitalsRow(label string, current, maxValue int, width float32) widget.Widget {
	ratio := ratioInt(current, maxValue)
	values := fmt.Sprintf("%d / %d", current, maxValue)
	percent := 0
	if maxValue > 0 {
		// Multiply before dividing so exact percentages such as 29/100
		// do not round down to 28 through floating-point ratio rounding.
		percent = int(float64(min(max(current, 0), maxValue)) * 100 / float64(maxValue))
	}
	return primitives.HBox(
		vitalsText(label, vitalsLabelWidth, primitives.TextAlignStart),
		primitives.Expanded(rotheme.VitalBar(
			progressbar.Value(ratio),
			progressbar.ShowLabel(true),
			progressbar.FormatLabelFn(func(float64) string { return values }),
		)),
		vitalsText(fmt.Sprintf("%d%%", percent), vitalsPercentWidth, primitives.TextAlignEnd),
	).Width(width).Gap(vitalsGap).CrossAlign(primitives.CrossAxisCenter)
}

func vitalsText(text string, width float32, align primitives.TextAlign) widget.Widget {
	return primitives.Box(
		rotheme.Text(text).Align(align).
			LineHeight(rotheme.ProgressBarHeight / rotheme.Default.Typography.TextSize),
	).Width(width).CrossAlign(primitives.CrossAxisStretch)
}
