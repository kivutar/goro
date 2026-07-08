package ui

import (
	"strings"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	friendContextMenuWidth  = 118
	friendContextMenuHeight = 28
)

type FriendContextMenu struct {
	window WindowState
	ctx    Context
	name   string
	action string
}

func (m *FriendContextMenu) Open(ctx Context, x, y int, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	m.ensureWindow()
	m.ctx = ctx
	m.name = name
	screenW, screenH := ctx.ScreenSize()
	x = clampWindowInt(x, 8, maxInt(8, screenW-friendContextMenuWidth-8))
	y = clampWindowInt(y, 8, maxInt(8, screenH-friendContextMenuHeight-8))
	m.window.OpenAt(x, y, m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *FriendContextMenu) Update(ctx Context) bool {
	m.ensureWindow()
	m.ctx = ctx
	if !m.window.IsOpen() {
		return false
	}
	if ctx.Input != nil {
		inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, m.window.x, m.window.y, friendContextMenuWidth, friendContextMenuHeight)
		if ctx.Input.JustPressed(render.KeyEscape) || (!inside && (ctx.Input.MouseJustPressed(render.MouseButtonLeft) || ctx.Input.MouseJustPressed(render.MouseButtonRight))) {
			m.Close()
			return true
		}
	}
	consumed := m.window.Update(ctx)
	m.Publish(ctx)
	return consumed
}

func (m *FriendContextMenu) Close() {
	m.ensureWindow()
	m.window.Close()
	m.Publish(m.ctx)
}

func (m *FriendContextMenu) Publish(ctx Context) {
	m.ensureWindow()
	m.window.Publish(ctx)
}

func (m *FriendContextMenu) PopAddFriendName() string {
	name := m.action
	m.action = ""
	return name
}

func (m *FriendContextMenu) ensureWindow() {
	if m.window.width != 0 {
		return
	}
	m.window = NewWindowState(friendContextMenuWidth, friendContextMenuHeight)
	m.window.titleHeight = 0
}

func (m *FriendContextMenu) widgetTree(ctx Context) widget.Widget {
	return Window(
		TitleBar(false),
		Radius(0),
		Size(friendContextMenuWidth, friendContextMenuHeight),
		Content(
			primitives.Box(
				rotheme.Button("Add Friend", func() {
					m.action = m.name
					m.Close()
				}).
					Width(friendContextMenuWidth).
					Height(friendContextMenuHeight),
			),
		),
	)
}
