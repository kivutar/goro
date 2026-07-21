package ui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	homunculusContextMenuWidth = 120
	homunculusContextMenuRowH  = 28
)

type HomunculusContextActionKind uint8

const (
	HomunculusContextActionNone HomunculusContextActionKind = iota
	HomunculusContextActionInfo
	HomunculusContextActionFeed
	HomunculusContextActionToggleAssist
)

type HomunculusContextAction struct {
	Kind HomunculusContextActionKind
}

type HomunculusContextMenu struct {
	Window
	aggressive bool
	action     HomunculusContextAction
}

func (m *HomunculusContextMenu) Open(ctx Context, x, y int, aggressive bool) {
	m.EnsureWindow(homunculusContextMenuWidth, m.height())
	m.titleHeight = 0
	m.ctx = ctx
	m.aggressive = aggressive
	screenW, screenH := ctx.ScreenSize()
	x = clampWindowInt(x, windowScreenMargin, maxInt(windowScreenMargin, screenW-homunculusContextMenuWidth-windowScreenMargin))
	y = clampWindowInt(y, windowScreenMargin, maxInt(windowScreenMargin, screenH-m.height()-windowScreenMargin))
	m.OpenAt(x, y, m.widgetTree())
	m.Publish(ctx)
}

func (m *HomunculusContextMenu) Update(ctx Context) bool {
	m.EnsureWindow(homunculusContextMenuWidth, m.height())
	m.titleHeight = 0
	m.ctx = ctx
	if !m.IsOpen() {
		return false
	}
	if ctx.Input != nil {
		inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, m.x, m.y, homunculusContextMenuWidth, m.height())
		if ctx.Input.JustPressed(input.KeyEscape) || (!inside && (ctx.Input.MouseJustPressed(input.MouseButtonLeft) || ctx.Input.MouseJustPressed(input.MouseButtonRight))) {
			m.Close()
			return true
		}
	}
	consumed := m.Window.Update(ctx)
	m.Publish(ctx)
	return consumed
}

func (m *HomunculusContextMenu) PopAction() HomunculusContextAction {
	action := m.action
	m.action = HomunculusContextAction{}
	return action
}

func (m *HomunculusContextMenu) widgetTree() widget.Widget {
	assistLabel := "Assist"
	if m.aggressive {
		assistLabel = "Stand By"
	}
	return Win(
		TitleBar(false),
		Radius(0),
		Size(homunculusContextMenuWidth, float32(m.height())),
		Content(
			primitives.Box(
				m.button("View Status", HomunculusContextActionInfo),
				m.button("Feed", HomunculusContextActionFeed),
				m.button(assistLabel, HomunculusContextActionToggleAssist),
			),
		),
	)
}

func (m *HomunculusContextMenu) button(label string, action HomunculusContextActionKind) widget.Widget {
	return rotheme.Button(label, func() {
		m.action = HomunculusContextAction{Kind: action}
		m.Close()
	}).
		Width(homunculusContextMenuWidth).
		Height(homunculusContextMenuRowH)
}

func (m *HomunculusContextMenu) height() int {
	return homunculusContextMenuRowH * 3
}
