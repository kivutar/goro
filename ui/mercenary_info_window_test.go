package ui

import "testing"

func TestMercenaryInfoWindowHeightFitsDetails(t *testing.T) {
	const barBlockH = homunculusInfoRowH + homunculusInfoBarH + homunculusInfoBarGap
	requiredDetailsH := 4*homunculusInfoRowH + 4*barBlockH + 7*homunculusInfoRowGap
	requiredBodyH := homunculusInfoContentPad*2 + requiredDetailsH
	availableBodyH := mercenaryInfoWindowH - ROWindowTitleHeight - ROWindowFooterHeight
	if availableBodyH < requiredBodyH {
		t.Fatalf("body height = %d, want at least %d", availableBodyH, requiredBodyH)
	}
}
