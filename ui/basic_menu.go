package ui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	basicMenuX         = windowScreenMargin
	basicMenuY         = characterWindowY + characterWindowHeight + basicMenuFollowGap
	basicMenuFollowGap = 6
	basicMenuCols      = 4
	basicMenuRows      = 2
	basicMenuButtonW   = 72
	basicMenuButtonH   = 24
	basicMenuGapX      = 6
	basicMenuGapY      = 5
	basicMenuPad       = 8
)

type BasicMenu struct {
	Window
	content   widget.Widget
	callbacks BasicMenuCallbacks
}

type BasicMenuCallbacks struct {
	OnStatus func()
	OnOption func()
	OnItems  func()
	OnEquip  func()
	OnSkill  func()
	OnMap    func()
	OnComm   func()
	OnFriend func()
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

func (m *BasicMenu) Update(ctx client.Context, callbacks BasicMenuCallbacks) bool {
	m.callbacks = callbacks
	width, height := basicMenuSize()
	if m.EnsureWindow(width, height) {
		m.titleHeight = 0
		m.CloseOnEsc = false
	}
	if !m.IsOpen() {
		m.OpenAt(basicMenuX, basicMenuY, m.widgetTree())
	} else if m.content == nil {
		m.SetContent(m.widgetTree())
	}
	consumed := m.Window.Update(ctx)
	m.Publish(ctx)
	return consumed
}

func (m *BasicMenu) Rebind(ctx client.Context, callbacks BasicMenuCallbacks) {
	m.callbacks = callbacks
	width, height := basicMenuSize()
	if m.EnsureWindow(width, height) {
		m.titleHeight = 0
		m.CloseOnEsc = false
	}
	m.content = nil
	if !m.IsOpen() {
		return
	}
	m.SetContent(m.widgetTree())
	m.Publish(ctx)
}

// FollowCharacterWindow keeps the menu attached below the character window
// while leaving both as independent overlays for input and redraw purposes.
func (m *BasicMenu) FollowCharacterWindow(ctx client.Context, character *CharacterWindow) {
	if character == nil || !character.IsOpen() {
		return
	}
	width, height := basicMenuSize()
	if m.EnsureWindow(width, height) {
		m.titleHeight = 0
		m.CloseOnEsc = false
	}
	x := character.x
	y := character.y + character.height + basicMenuFollowGap
	if character.dragLayer {
		m.followDragPosition(ctx, x, y)
		return
	}
	m.endFollowDrag(ctx)
	if m.positioned && m.x == x && m.y == y {
		return
	}
	m.ctx = ctx
	m.positioned = true
	m.setPosition(ctx, x, y)
	if m.IsOpen() {
		m.Publish(ctx)
	}
}

func (m *BasicMenu) followDragPosition(ctx client.Context, x, y int) {
	m.ctx = ctx
	m.positioned = true
	m.x, m.y = x, y
	if overlay := m.positionedOverlay(); overlay != nil {
		overlay.setFrameQuiet(x, y, m.width, m.height)
		if !overlay.hidden {
			overlay.hidden = true
			damage := overlay.markFrameDirty()
			invalidateWindowRect(ctx, damage)
		}
		return
	}
	m.placed = nil
}

func (m *BasicMenu) endFollowDrag(ctx client.Context) {
	overlay := m.positionedOverlay()
	if overlay == nil || !overlay.hidden {
		return
	}
	overlay.hidden = false
	damage := overlay.markFrameDirty()
	invalidateWindowRect(ctx, damage)
}

func basicMenuBounds() (int, int, int, int) {
	w, h := basicMenuSize()
	return basicMenuX, basicMenuY, w, h
}

func basicMenuSize() (int, int) {
	w := basicMenuPad*2 + basicMenuCols*basicMenuButtonW + (basicMenuCols-1)*basicMenuGapX
	h := basicMenuPad*2 + basicMenuRows*basicMenuButtonH + (basicMenuRows-1)*basicMenuGapY
	return w, h
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
			key := button.key
			label := button.label
			buttons = append(buttons,
				rotheme.Button(label, func() {
					m.invoke(key)
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
	width, height := basicMenuSize()
	m.content = Win(
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

func (m *BasicMenu) invoke(key string) {
	var callback func()
	switch key {
	case "status":
		callback = m.callbacks.OnStatus
	case "option":
		callback = m.callbacks.OnOption
	case "items":
		callback = m.callbacks.OnItems
	case "equip":
		callback = m.callbacks.OnEquip
	case "skill":
		callback = m.callbacks.OnSkill
	case "map":
		callback = m.callbacks.OnMap
	case "comm":
		callback = m.callbacks.OnComm
	case "friend":
		callback = m.callbacks.OnFriend
	}
	if callback != nil {
		callback()
	}
}
