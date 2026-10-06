package ui

import (
	"testing"

	"github.com/kivutar/goro/ui/rotheme"
)

func TestMercenaryInfoWindowHeightFitsDetails(t *testing.T) {
	const barBlockH = homunculusInfoRowH + homunculusInfoBarH + homunculusInfoBarGap
	requiredDetailsH := 4*homunculusInfoRowH + 2*barBlockH + 2*int(rotheme.ProgressBarHeight) + 7*homunculusInfoRowGap
	requiredBodyH := homunculusInfoContentPad*2 + requiredDetailsH
	availableBodyH := mercenaryInfoWindowH - ROWindowTitleHeight - ROWindowFooterHeight
	if availableBodyH < requiredBodyH {
		t.Fatalf("body height = %d, want at least %d", availableBodyH, requiredBodyH)
	}
}
