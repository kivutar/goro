package ui

import (
	"image/color"
	"testing"
	"time"

	"github.com/kivutar/goro/render"
)

func TestAnnouncementLifecycleAndReplacement(t *testing.T) {
	now := time.Unix(100, 0)
	var announcement Announcement
	announcement.Show("First", color.RGBA{R: 255}, AnnouncementStyle{}, now)
	if !announcement.Visible(now.Add(announcementLife-time.Millisecond)) || announcement.Visible(now.Add(announcementLife)) {
		t.Fatal("announcement did not observe its 20 second lifetime")
	}
	announcement.Show("Second", color.RGBA{G: 255}, AnnouncementStyle{Y: 75, FontSize: 20, Bold: true}, now.Add(time.Second))
	if announcement.text != "Second" || announcement.y != 75 || announcement.fontSize != 20 || !announcement.bold || announcement.color.G != 255 || !announcement.Visible(now.Add(announcementLife)) {
		t.Fatalf("replacement announcement = %+v", announcement)
	}
}

func TestWrapAnnouncementText(t *testing.T) {
	maxWidth, _ := render.BitmapTextSize("alpha beta")
	lines := wrapAnnouncementText("alpha beta gamma", maxWidth, announcementFontSize)
	if len(lines) != 2 || lines[0] != "alpha beta" || lines[1] != "gamma" {
		t.Fatalf("wrapped lines = %#v", lines)
	}
}

func TestAnnouncementDefaultsAndBoundsLegacyFontSize(t *testing.T) {
	now := time.Unix(100, 0)
	var announcement Announcement
	announcement.Show("Default", color.RGBA{}, AnnouncementStyle{}, now)
	if announcement.fontSize != announcementFontSize || announcement.y != announcementTop {
		t.Fatalf("default announcement = %+v", announcement)
	}
	announcement.Show("Large", color.RGBA{}, AnnouncementStyle{FontSize: 200}, now)
	if announcement.fontSize != 32 {
		t.Fatalf("bounded font size = %d, want 32", announcement.fontSize)
	}
}
