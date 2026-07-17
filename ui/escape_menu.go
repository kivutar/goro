package ui

import (
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	escapeMenuWidth  = 252
	escapeMenuHeight = 200
	escapeMenuPad    = 16
	escapeMenuGap    = 8
)

type EscapeMenu struct {
	Window
	action        EscapeMenuAction
	pending       bool
	pendingAction EscapeMenuAction
	ctx           client.Context
}

type EscapeMenuAction int

const (
	EscapeMenuActionNone EscapeMenuAction = iota
	EscapeMenuActionCharacterSelect
	EscapeMenuActionSettings
	EscapeMenuActionCancel
	EscapeMenuActionExit
)

func (m *EscapeMenu) Toggle(ctx client.Context) {
	m.ctx = ctx
	m.EnsureWindow(escapeMenuWidth, escapeMenuHeight)
	if m.IsOpen() {
		m.Window.Close()
		m.Publish(ctx)
		return
	}
	m.action = EscapeMenuActionNone
	m.pending = false
	m.pendingAction = EscapeMenuActionNone
	m.CloseOnEsc = true
	m.Window.Open(ctx, m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *EscapeMenu) Update(ctx client.Context) bool {
	m.ctx = ctx
	if m.action != EscapeMenuActionNone {
		return true
	}
	if ctx.Input == nil {
		return false
	}
	if ctx.Input.JustPressed(input.KeyEscape) {
		m.Toggle(ctx)
		return true
	}
	if !m.IsOpen() {
		return false
	}
	m.openWindow(ctx)
	if m.Window.Update(ctx) {
		if !m.IsOpen() {
			m.Publish(ctx)
		}
		return true
	}
	return true
}

func (m *EscapeMenu) RequestCharacterSelect(ctx client.Context) {
	m.pending = true
	m.pendingAction = EscapeMenuActionCharacterSelect
	if ctx.Network == nil {
		m.pending = false
		m.pendingAction = EscapeMenuActionNone
		m.refresh(ctx)
		return
	}
	if err := ctx.Network.SendRestart(network.RestartTypeCharSelect); err != nil {
		m.pending = false
		m.pendingAction = EscapeMenuActionNone
		glog.Warnf("escape menu character select failed: %v", err)
		m.refresh(ctx)
		return
	}
	m.refresh(ctx)
}

func (m *EscapeMenu) RequestQuitGame(ctx client.Context) {
	m.pending = true
	m.pendingAction = EscapeMenuActionExit
	if ctx.Network == nil {
		m.pending = false
		m.pendingAction = EscapeMenuActionNone
		m.refresh(ctx)
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
		return
	}
	if err := ctx.Network.SendQuitGame(); err != nil {
		m.pending = false
		m.pendingAction = EscapeMenuActionNone
		glog.Warnf("escape menu quit failed: %v", err)
		m.refresh(ctx)
		return
	}
	m.refresh(ctx)
}

func (m *EscapeMenu) ApplyRestartAck(ack network.RestartAck) bool {
	if !m.pending || m.pendingAction != EscapeMenuActionCharacterSelect {
		return false
	}
	if ack.Allowed {
		m.refresh(m.ctx)
		return true
	}
	m.pending = false
	m.pendingAction = EscapeMenuActionNone
	m.EnsureWindow(escapeMenuWidth, escapeMenuHeight)
	m.Window.Open(m.ctx, m.widgetTree(m.ctx))
	m.refresh(m.ctx)
	return false
}

func (m *EscapeMenu) ApplyQuitGameAck(ack network.QuitGameAck) bool {
	if !m.pending || m.pendingAction != EscapeMenuActionExit {
		return false
	}
	if ack.Allowed {
		m.refresh(m.ctx)
		return true
	}
	m.pending = false
	m.pendingAction = EscapeMenuActionNone
	m.EnsureWindow(escapeMenuWidth, escapeMenuHeight)
	m.Window.Open(m.ctx, m.widgetTree(m.ctx))
	m.refresh(m.ctx)
	return true
}

func (m *EscapeMenu) ConsumeAction() EscapeMenuAction {
	action := m.action
	m.action = EscapeMenuActionNone
	return action
}

func (m *EscapeMenu) Pending() bool {
	return m.pending
}

func (m *EscapeMenu) Action() EscapeMenuAction {
	return m.action
}

func (m *EscapeMenu) openWindow(ctx client.Context) {
	m.EnsureWindow(escapeMenuWidth, escapeMenuHeight)
	if !m.IsOpen() {
		return
	}
	if m.content == nil {
		m.SetContent(m.widgetTree(ctx))
	}
	m.Publish(ctx)
}

func (m *EscapeMenu) refresh(ctx client.Context) {
	m.EnsureWindow(escapeMenuWidth, escapeMenuHeight)
	if !m.IsOpen() {
		return
	}
	m.SetContent(m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *EscapeMenu) widgetTree(ctx client.Context) widget.Widget {
	return Win(
		Title("Menu"),
		CloseButton(false),
		Size(escapeMenuWidth, escapeMenuHeight),
		Content(
			primitives.Box(
				rotheme.LargeButtonDisabled("Character Select", m.pending, func() {
					m.action = EscapeMenuActionCharacterSelect
					m.refresh(ctx)
				}),
				rotheme.LargeButtonDisabled("Settings", m.pending, func() {
					m.action = EscapeMenuActionSettings
					m.Window.Close()
				}),
				rotheme.LargeButtonDisabled("Cancel", m.pending, func() {
					m.Window.Close()
				}),
				rotheme.LargeButtonDisabled("Exit to Windows", m.pending, func() {
					m.RequestQuitGame(ctx)
				}),
			).
				Padding(escapeMenuPad).
				Gap(escapeMenuGap).
				CrossAlign(primitives.CrossAxisStretch),
		),
	)
}
