package ui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	basicMenuX       = 16
	basicMenuY       = 16 + 158 + 6
	basicMenuCols    = 4
	basicMenuRows    = 2
	basicMenuButtonW = 72
	basicMenuButtonH = 24
	basicMenuGapX    = 6
	basicMenuGapY    = 5
	basicMenuPad     = 8
)

type BasicMenu struct {
	window     WindowState
	content    widget.Widget
	lastAction string
}

type basicMenuButton struct {
	key   string
	label string
}

var basicMenuButtons = []basicMenuButton{
	{key: "status", label: "Status"},
	{key: "option", label: "Option"},
	{key: "items", label: "Items"},
	{key: "equip", label: "Equip"},
	{key: "skill", label: "Skill"},
	{key: "map", label: "Map"},
	{key: "comm", label: "Comm"},
	{key: "friend", label: "Friend"},
}

func (m *BasicMenu) Update(ctx client.Context) bool {
	m.ensureWindow()
	if !m.window.IsOpen() {
		m.window.OpenAt(basicMenuX, basicMenuY, m.widgetTree())
	}
	consumed := m.window.Update(ctx)
	m.window.Publish(ctx)
	return consumed
}

func basicMenuBounds() (int, int, int, int) {
	w := basicMenuPad*2 + basicMenuCols*basicMenuButtonW + (basicMenuCols-1)*basicMenuGapX
	h := basicMenuPad*2 + basicMenuRows*basicMenuButtonH + (basicMenuRows-1)*basicMenuGapY
	return basicMenuX, basicMenuY, w, h
}

func (m *BasicMenu) ensureWindow() {
	if m.window.width != 0 {
		return
	}
	_, _, width, height := basicMenuBounds()
	m.window = NewWindowState(width, height)
	m.window.titleHeight = 0
	m.window.SetCloseOnEscape(false)
}

func (m *BasicMenu) widgetTree() widget.Widget {
	if m.content != nil {
		return m.content
	}
	rows := make([]widget.Widget, 0, basicMenuRows)
	for row := 0; row < basicMenuRows; row++ {
		buttons := make([]widget.Widget, 0, basicMenuCols)
		for col := 0; col < basicMenuCols; col++ {
			button := basicMenuButtons[row*basicMenuCols+col]
			buttons = append(buttons,
				rotheme.Button(button.label, func() {
					m.lastAction = button.key
				}).
					Width(basicMenuButtonW).
					Height(basicMenuButtonH),
			)
		}
		rows = append(rows,
			primitives.HBox(buttons...).
				Gap(basicMenuGapX).
				CrossAlign(primitives.CrossAxisStretch),
		)
	}
	_, _, width, height := basicMenuBounds()
	m.content = Window(
		TitleBar(false),
		Size(float32(width), float32(height)),
		Content(
			primitives.Box(rows...).
				Padding(basicMenuPad).
				Gap(basicMenuGapY).
				CrossAlign(primitives.CrossAxisStretch),
		),
	)
	return m.content
}

func (m *BasicMenu) PopAction() string {
	action := m.lastAction
	m.lastAction = ""
	return action
}
