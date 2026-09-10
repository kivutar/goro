package render

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestTextBannerUsesThemeFontAndPreservesServerColor(t *testing.T) {
	frame := NewFrame(800, 600)
	cyan := color.RGBA{G: 255, B: 255, A: 255}
	DrawUITextBanner(frame, "Voting Event", 400, 40, 520, cyan, 0, false)
	if len(frame.uiTextBoxes) != 1 || len(frame.uiTextLabels) != 0 {
		t.Fatal("banner text and background should be queued as one measured box")
	}
	style := frame.uiTextBoxes[0].style
	if style.size != rotheme.Default.Typography.TextSize || style.bold {
		t.Fatalf("banner font = %.1f bold=%t, want regular console font", style.size, style.bold)
	}
	if style.foreground != widget.RGBA8(0, 255, 255, 255) || style.background != widget.RGBA8(0, 0, 0, 128) {
		t.Fatalf("banner colors = %v / %v", style.foreground, style.background)
	}
	if style.align != widget.TextAlignCenter {
		t.Fatal("banner lines are not centered")
	}
}

func TestTextBannerLayoutUsesDrawFontMetrics(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 2} {
		for _, bold := range []bool{false, true} {
			for _, size := range []float32{11, 12, 20, 32} {
				t.Run(fmt.Sprintf("scale_%g_bold_%t_size_%g", scale, bold, size), func(t *testing.T) {
					// The recorder measures through the real UI font implementation,
					// exactly as the console does when recording asynchronous UI.
					canvas := newUIDrawRecorder(800, 600, scale)
					defer canvas.close()
					canvas.setTextMode(widget.TextModeVector)
					style := bannerOverlayTextBoxStyle(size, bold, color.RGBA{B: 255, A: 255}, 520)
					for _, text := range []string{
						"The Voting Event is now open! Visit the event NPC in Prontera and vote for your favorite event to receive rewards.",
						strings.Repeat("WideWWW", 30),
						"First line\r\nSecond line with accented letters: été, récompenses.",
					} {
						lines := overlayTextBoxLines(canvas, text, style)
						width, height := overlayTextBoxSize(canvas, lines, style)
						if len(lines) < 2 || width > 520 || height <= int(style.padY*2) {
							t.Fatalf("layout = %dx%d, lines=%q", width, height, lines)
						}
						if strings.Join(strings.Fields(strings.Join(lines, "")), "") != strings.Join(strings.Fields(text), "") {
							t.Fatalf("wrapping lost text: %q", lines)
						}
						for _, line := range lines {
							bounds := geometry.NewRect(style.padX, style.padY, float32(width)-style.padX*2, style.lineH)
							rotheme.DrawText(canvas, line, bounds, style.size, style.foreground, style.bold, style.align)
							if measured := rotheme.MeasureText(canvas, line, style.size, style.bold); measured > bounds.Width() {
								t.Fatalf("line width %.2f exceeds padded box width %.2f: %q", measured, bounds.Width(), line)
							}
						}
					}
				})
			}
		}
	}
}

func TestTextBannerClampsVerticalPosition(t *testing.T) {
	frame := NewFrame(800, 600)
	box := UITextBoxCommand{X: 400, Y: 590, Anchor: UITextBoxAnchorTopCenter}
	x, y := uiTextBoxPosition(frame, box, cachedOverlayImage{width: 520, height: 50})
	if x != 140 || y != 550 {
		t.Fatalf("banner position = %g,%g, want 140,550", x, y)
	}
}
