package rotheme

import "testing"

func TestLabelUsesBlueBoldStyle(t *testing.T) {
	style := Label("Account").Style()

	if !style.Bold {
		t.Fatal("label should request native bold text")
	}
	if style.FontFamily != "" {
		t.Fatalf("label font family = %q, want gogpu/ui's built-in weighted family", style.FontFamily)
	}
	if style.Color != Default.Colors.LabelText {
		t.Fatalf("label color = %+v, want %+v", style.Color, Default.Colors.LabelText)
	}
}
