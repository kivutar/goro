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
	escapeMenuWidth        = 252
	escapeMenuHeight       = 200
	escapeMenuReviveHeight = 238
	escapeMenuPad          = 16
	escapeMenuGap          = 8
)

type EscapeMenu struct {
	Window
	action        EscapeMenuAction
	pending       bool
	pendingAction EscapeMenuAction
	deathMode     bool
	ctx           client.Context
}

type EscapeMenuAction int

const (
	EscapeMenuActionNone EscapeMenuAction = iota
	EscapeMenuActionAutoRevive
	EscapeMenuActionSavePoint
	EscapeMenuActionCharacterSelect
	EscapeMenuActionSettings
	EscapeMenuActionCancel
	EscapeMenuActionExit
)

func (m *EscapeMenu) Toggle(ctx client.Context) {
	m.ctx = ctx
	m.ensureSize(ctx)
	if m.IsOpen() {
		m.Window.Close()
		m.Publish(ctx)
		return
	}
	if !m.deathMode {
		m.action = EscapeMenuActionNone
		m.pending = false
		m.pendingAction = EscapeMenuActionNone
	}
	m.CloseOnEsc = true
	m.Window.Open(ctx, m.widgetTree(ctx))
	m.Publish(ctx)
}

// OpenDeath switches the regular escape menu to its death variant and opens it.
// The death mode remains active if the player later hides the window with Escape.
func (m *EscapeMenu) OpenDeath(ctx client.Context) {
	m.ctx = ctx
	m.deathMode = true
	m.ensureSize(ctx)
	m.action = EscapeMenuActionNone
	m.pending = false
	m.pendingAction = EscapeMenuActionNone
	m.CloseOnEsc = true
	if m.IsOpen() {
		m.SetContent(m.widgetTree(ctx))
	} else {
		m.Window.Open(ctx, m.widgetTree(ctx))
	}
	m.Publish(ctx)
}

// ResetDeath returns the escape menu to its regular mode after resurrection or
// a map change. It also closes a death menu that the player left visible.
func (m *EscapeMenu) ResetDeath(ctx client.Context) {
	if !m.deathMode {
		return
	}
	m.ctx = ctx
	m.Window.Close()
	m.Publish(ctx)
	m.deathMode = false
	m.SetSize(escapeMenuWidth, escapeMenuHeight)
	m.action = EscapeMenuActionNone
	m.pending = false
	m.pendingAction = EscapeMenuActionNone
}

func (m *EscapeMenu) DeathMode() bool {
	return m.deathMode
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
	return !m.deathMode
}

func (m *EscapeMenu) ReturnToSavePoint(ctx client.Context) {
	m.pending = true
	m.pendingAction = EscapeMenuActionSavePoint
	if ctx.Network == nil {
		m.pending = false
		m.pendingAction = EscapeMenuActionNone
		m.refresh(ctx)
		return
	}
	if err := ctx.Network.SendRestart(network.RestartTypeRespawn); err != nil {
		m.pending = false
		m.pendingAction = EscapeMenuActionNone
		glog.Warnf("escape menu respawn failed: %v", err)
	}
	m.refresh(ctx)
}

func (m *EscapeMenu) RequestAutoRevive(ctx client.Context) {
	if err := client.RequestAutoRevive(ctx); err != nil {
		glog.Warnf("escape menu auto-revive failed: %v", err)
	}
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

func (m *EscapeMenu) PendingAction() EscapeMenuAction {
	return m.pendingAction
}

func (m *EscapeMenu) Action() EscapeMenuAction {
	return m.action
}

func (m *EscapeMenu) openWindow(ctx client.Context) {
	m.ensureSize(ctx)
	if !m.IsOpen() {
		return
	}
	if m.content == nil {
		m.SetContent(m.widgetTree(ctx))
	}
	m.Publish(ctx)
}

func (m *EscapeMenu) refresh(ctx client.Context) {
	m.ensureSize(ctx)
	if !m.IsOpen() {
		return
	}
	m.SetContent(m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *EscapeMenu) ensureSize(ctx client.Context) {
	height := m.menuHeight(ctx)
	if !m.EnsureWindow(escapeMenuWidth, height) {
		m.SetSize(escapeMenuWidth, height)
	}
}

func (m *EscapeMenu) menuHeight(ctx client.Context) int {
	if m.deathMode && client.AutoReviveAvailable(ctx) {
		return escapeMenuReviveHeight
	}
	return escapeMenuHeight
}

func (m *EscapeMenu) widgetTree(ctx client.Context) widget.Widget {
	if m.deathMode {
		return m.deathWidgetTree(ctx)
	}
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
				rotheme.LargeButton("Settings", func() {
					m.action = EscapeMenuActionSettings
					m.Window.Close()
				}),
				rotheme.LargeButton("Cancel", func() {
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

func (m *EscapeMenu) deathWidgetTree(ctx client.Context) widget.Widget {
	buttons := make([]widget.Widget, 0, 5)
	if client.AutoReviveAvailable(ctx) {
		buttons = append(buttons, rotheme.LargeButtonDisabled("Return to Life", m.pending, func() {
			m.action = EscapeMenuActionAutoRevive
			m.refresh(ctx)
		}))
	}
	buttons = append(buttons,
		rotheme.LargeButtonDisabled("Return to Save Point", m.pending, func() {
			m.action = EscapeMenuActionSavePoint
			m.refresh(ctx)
		}),
		rotheme.LargeButtonDisabled("Character Select", m.pending, func() {
			m.action = EscapeMenuActionCharacterSelect
			m.refresh(ctx)
		}),
		rotheme.LargeButtonDisabled("Exit to Windows", m.pending, func() {
			m.RequestQuitGame(ctx)
		}),
		rotheme.LargeButton("Cancel", func() {
			m.Window.Close()
		}),
	)
	return Win(
		Title("Menu"),
		CloseButton(false),
		Size(escapeMenuWidth, float32(m.menuHeight(ctx))),
		Content(
			primitives.Box(buttons...).
				Padding(escapeMenuPad).
				Gap(escapeMenuGap).
				CrossAlign(primitives.CrossAxisStretch),
		),
	)
}
