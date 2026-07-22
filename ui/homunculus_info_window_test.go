package ui

import (
	"testing"

	"github.com/kivutar/goro/session"
)

func TestHomunculusIntimacyTextMatchesRobrowserThresholds(t *testing.T) {
	tests := []struct {
		value int
		want  string
	}{
		{0, "Awkward"},
		{99, "Awkward"},
		{100, "Shy"},
		{249, "Shy"},
		{250, "Neutral"},
		{599, "Neutral"},
		{600, "Cordial"},
		{899, "Cordial"},
		{900, "Loyal"},
	}
	for _, tt := range tests {
		if got := HomunculusIntimacyText(tt.value); got != tt.want {
			t.Fatalf("HomunculusIntimacyText(%d) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestHomunculusASPDDisplayMatchesRobrowserFormula(t *testing.T) {
	if got := HomunculusASPDDisplay(1560); got != 44 {
		t.Fatalf("ASPD display = %d, want 44", got)
	}
	if got := HomunculusASPDDisplay(0); got != 0 {
		t.Fatalf("empty ASPD display = %d, want 0", got)
	}
}

func TestHomunculusRenameUsesModifiedFlagThreshold(t *testing.T) {
	if !homunculusCanRename(session.Companion{Flags: 4}) {
		t.Fatal("flags below 5 should allow rename")
	}
	if homunculusCanRename(session.Companion{Flags: 5}) {
		t.Fatal("flags 5 and above should disable rename")
	}
}

func TestHomunculusInfoWindowHeightFitsDetails(t *testing.T) {
	const barBlockH = homunculusInfoRowH + homunculusInfoBarH + homunculusInfoBarGap
	requiredDetailsH := homunculusInfoNameH + homunculusInfoRowH + 4*barBlockH + 2*homunculusInfoRowH + 7*homunculusInfoRowGap
	requiredBodyH := homunculusInfoContentPad*2 + requiredDetailsH
	availableBodyH := homunculusInfoWindowH - ROWindowTitleHeight - ROWindowFooterHeight
	if availableBodyH < requiredBodyH {
		t.Fatalf("body height = %d, want at least %d", availableBodyH, requiredBodyH)
	}
}

func TestHomunculusInfoWindowColumnsFitContent(t *testing.T) {
	assertHomunculusInfoWidth(t, "content columns", homunculusInfoLeftW+homunculusInfoColumnGap+homunculusInfoRightW, homunculusInfoContentW)
	assertHomunculusInfoWidth(t, "detail row", homunculusInfoInfoLabelW+homunculusInfoInfoGap+homunculusInfoInfoValueW, homunculusInfoRightW)
	assertHomunculusInfoWidth(t, "name row", homunculusInfoInfoLabelW+homunculusInfoNameGap+homunculusInfoNameInputW+homunculusInfoNameGap+homunculusInfoNameButtonW, homunculusInfoRightW)
	assertHomunculusInfoWidth(t, "bar row", homunculusInfoBarPadX+homunculusInfoBarW, homunculusInfoRightW)
}

func assertHomunculusInfoWidth(t *testing.T, name string, got, want float32) {
	t.Helper()
	if got != want {
		t.Fatalf("%s width = %v, want %v", name, got, want)
	}
}

func TestHomunculusEXPUsesUint64(t *testing.T) {
	if got := formatHomunculusEXPBarText(5_000_000_000, 10_000_000_000); got != "50.0%" {
		t.Fatalf("exp text = %q, want 50.0%%", got)
	}
	if got := ratioUint64(5_000_000_000, 10_000_000_000); got != 0.5 {
		t.Fatalf("exp ratio = %f, want 0.5", got)
	}
}

func TestHungerBarFillColorMatchesOverlayThreshold(t *testing.T) {
	if got := HungerBarFillColor(24, 100); got != HungerBarLowColor {
		t.Fatalf("24%% hunger color = %#v, want low hunger yellow %#v", got, HungerBarLowColor)
	}
	if got := HungerBarFillColor(25, 100); got != HungerBarColor {
		t.Fatalf("25%% hunger color = %#v, want normal hunger color %#v", got, HungerBarColor)
	}
}
