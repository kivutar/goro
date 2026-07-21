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
