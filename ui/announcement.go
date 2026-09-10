package ui

import (
	"image/color"
	"strings"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	announcementLife     = 20 * time.Second
	announcementTop      = 40
	announcementMaxWidth = 500
	announcementPadding  = 10
)

type AnnouncementStyle struct {
	Y        int
	FontSize int
	Bold     bool
}

type Announcement struct {
	text     string
	color    color.RGBA
	shownAt  time.Time
	y        int
	fontSize int
	bold     bool
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
		style.FontSize = int(rotheme.Default.Typography.TextSize)
	} else {
		style.FontSize = maxInt(8, minInt(style.FontSize, 32))
	}
	a.text = text
	a.color = messageColor
	a.shownAt = now
	a.y = style.Y
	a.fontSize = style.FontSize
	a.bold = style.Bold
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
	maxWidth := minInt(announcementMaxWidth+2*announcementPadding, screenW)
	render.DrawUITextBanner(screen, a.text, float64(screenW)/2, float64(a.y), float64(maxWidth), a.color, float32(a.fontSize), a.bold)
}
