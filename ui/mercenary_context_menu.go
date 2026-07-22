package ui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	mercenaryContextMenuWidth = 120
	mercenaryContextMenuRowH  = 28
)

type MercenaryContextActionKind uint8

const (
	MercenaryContextActionNone MercenaryContextActionKind = iota
	MercenaryContextActionInfo
	MercenaryContextActionToggleAssist
)

type MercenaryContextAction struct {
	Kind MercenaryContextActionKind
}

type MercenaryContextMenu struct {
	Window
	aggressive bool
	action     MercenaryContextAction
}

func (m *MercenaryContextMenu) Open(ctx Context, x, y int, aggressive bool) {
	m.EnsureWindow(mercenaryContextMenuWidth, m.height())
	m.titleHeight = 0
	m.ctx = ctx
	m.aggressive = aggressive
	screenW, screenH := ctx.ScreenSize()
	x = clampWindowInt(x, windowScreenMargin, maxInt(windowScreenMargin, screenW-mercenaryContextMenuWidth-windowScreenMargin))
	y = clampWindowInt(y, windowScreenMargin, maxInt(windowScreenMargin, screenH-m.height()-windowScreenMargin))
	m.OpenAt(x, y, m.widgetTree())
	m.Publish(ctx)
}

func (m *MercenaryContextMenu) Update(ctx Context) bool {
	m.EnsureWindow(mercenaryContextMenuWidth, m.height())
	m.titleHeight = 0
	m.ctx = ctx
	if !m.IsOpen() {
		return false
	}
	if ctx.Input != nil {
		inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, m.x, m.y, mercenaryContextMenuWidth, m.height())
		if ctx.Input.JustPressed(input.KeyEscape) || (!inside && (ctx.Input.MouseJustPressed(input.MouseButtonLeft) || ctx.Input.MouseJustPressed(input.MouseButtonRight))) {
			m.Close()
			return true
		}
	}
	consumed := m.Window.Update(ctx)
	m.Publish(ctx)
	return consumed
}

func (m *MercenaryContextMenu) PopAction() MercenaryContextAction {
	action := m.action
	m.action = MercenaryContextAction{}
	return action
}

func (m *MercenaryContextMenu) widgetTree() widget.Widget {
	assistLabel := "Assist"
	if m.aggressive {
		assistLabel = "Stand By"
	}
	return Win(
		TitleBar(false),
		Radius(0),
		Size(mercenaryContextMenuWidth, float32(m.height())),
		Content(
			primitives.Box(
				m.button("View Status", MercenaryContextActionInfo),
				m.button(assistLabel, MercenaryContextActionToggleAssist),
			),
		),
	)
}

func (m *MercenaryContextMenu) button(label string, action MercenaryContextActionKind) widget.Widget {
	return rotheme.Button(label, func() {
		m.action = MercenaryContextAction{Kind: action}
		m.Close()
	}).
		Width(mercenaryContextMenuWidth).
		Height(mercenaryContextMenuRowH)
}

func (m *MercenaryContextMenu) height() int {
	return mercenaryContextMenuRowH * 2
}
