package ui

import (
	"image/color"
	"strings"
	"time"

	"github.com/kivutar/goro/render"
)

const (
	announcementLife     = 20 * time.Second
	announcementTop      = 40
	announcementMaxWidth = 500
	announcementPadding  = 10
	announcementFontSize = 12
	// UI text labels include two pixels of transparent canvas padding. Starting
	// them here centers the visible glyphs in the banner's visual padding.
	announcementTextTop = 2
)

type AnnouncementStyle struct {
	Y        int
	FontSize int
	Bold     bool
}

type Announcement struct {
	text      string
	color     color.RGBA
	shownAt   time.Time
	y         int
	fontSize  int
	bold      bool
	lineWidth int
	lines     []string
}

func (a *Announcement) Show(text string, messageColor color.RGBA, style AnnouncementStyle, now time.Time) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	if messageColor.A == 0 {
		messageColor.A = 255
	}
	if style.Y <= 0 {
		style.Y = announcementTop
	}
	if style.FontSize <= 0 {
		style.FontSize = announcementFontSize
	} else {
		style.FontSize = maxInt(8, minInt(style.FontSize, 32))
	}
	a.text = text
	a.color = messageColor
	a.shownAt = now
	a.y = style.Y
	a.fontSize = style.FontSize
	a.bold = style.Bold
	a.lineWidth = 0
	a.lines = nil
}

func (a *Announcement) Visible(now time.Time) bool {
	if a == nil || a.text == "" || a.shownAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	return now.Sub(a.shownAt) < announcementLife
}

func (a *Announcement) Draw(screen *render.Frame, now time.Time) {
	if screen == nil || !a.Visible(now) {
		return
	}
	screenW := screen.Bounds().Dx()
	maxWidth := minInt(announcementMaxWidth, screenW-2*announcementPadding)
	if maxWidth <= 0 {
		return
	}
	if a.lines == nil || a.lineWidth != maxWidth {
		a.lines = wrapAnnouncementText(a.text, maxWidth, a.fontSize)
		a.lineWidth = maxWidth
	}
	if len(a.lines) == 0 {
		return
	}
	textWidth := 0
	for _, line := range a.lines {
		width := announcementTextWidth(line, a.fontSize)
		textWidth = maxInt(textWidth, width)
	}
	boxWidth := minInt(maxWidth+2*announcementPadding, textWidth+2*announcementPadding)
	lineHeight := a.fontSize + 5
	boxHeight := 10 + len(a.lines)*lineHeight
	boxX := (screenW - boxWidth) / 2
	boxY := maxInt(0, minInt(a.y, maxInt(0, screen.Bounds().Dy()-boxHeight)))
	render.DrawRect(screen, float64(boxX), float64(boxY), float64(boxWidth), float64(boxHeight), color.RGBA{A: 128})
	centerX := float64(screenW) / 2
	for i, line := range a.lines {
		render.DrawCenteredUITextAtSize(screen, line, centerX, float64(boxY+announcementTextTop+i*lineHeight), a.color, float32(a.fontSize), a.bold)
	}
}

func wrapAnnouncementText(text string, maxWidth, fontSize int) []string {
	if maxWidth <= 0 {
		return nil
	}
	var lines []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			candidate := line + " " + word
			if announcementTextWidth(candidate, fontSize) <= maxWidth {
				line = candidate
				continue
			}
			lines = append(lines, line)
			line = word
		}
		lines = append(lines, line)
	}
	return lines
}

func announcementTextWidth(text string, fontSize int) int {
	width, _ := render.BitmapTextSize(text)
	if fontSize <= 0 || fontSize == announcementFontSize {
		return width
	}
	return (width*fontSize + announcementFontSize - 1) / announcementFontSize
}
