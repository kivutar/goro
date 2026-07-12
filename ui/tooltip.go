package ui

import (
	"strings"

	"github.com/kivutar/goro/render"
)

type tooltipState struct {
	text    string
	centerX int
	belowY  int
	aboveY  int
	open    bool
}

func (t *tooltipState) Show(ctx Context, text string, centerX, belowY, aboveY int) {
	text = strings.TrimSpace(text)
	if text == "" {
		t.Hide()
		return
	}
	t.text = text
	t.centerX = centerX
	t.belowY = belowY
	t.aboveY = aboveY
	t.open = true
}

func (t *tooltipState) Hide() {
	if t == nil {
		return
	}
	t.text = ""
	t.open = false
}

func (t *tooltipState) Draw(screen *render.Image) {
	if t == nil || !t.open {
		return
	}
	render.DrawUITooltip(screen, t.text, float64(t.centerX), float64(t.belowY), float64(t.aboveY))
}

func (t *tooltipState) Open() bool {
	return t != nil && t.open
}

func (t *tooltipState) Text() string {
	if t == nil {
		return ""
	}
	return t.text
}
