package ui

import (
	"fmt"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	escapeMenuWidth  = 252
	escapeMenuHeight = 200
	escapeMenuPad    = 16
	escapeMenuGap    = 8
)

var (
	escapeMenuDisabledColor = DisabledColor
)

type EscapeMenu struct {
	open    bool
	action  EscapeMenuAction
	pending bool
	status  string
	window  WindowState
	ctx     client.Context
}

type EscapeMenuAction int

const (
	EscapeMenuActionNone EscapeMenuAction = iota
	EscapeMenuActionCharacterSelect
	EscapeMenuActionSettings
	EscapeMenuActionCancel
	EscapeMenuActionExit
)

func (m *EscapeMenu) OpenMenu() {
	m.open = true
	m.action = EscapeMenuActionNone
	m.pending = false
	m.status = ""
}

func (m *EscapeMenu) Update(ctx client.Context) bool {
	m.ctx = ctx
	if m.action != EscapeMenuActionNone {
		return true
	}
	if ctx.Input == nil {
		return false
	}
	if !m.open {
		if ctx.Input.JustPressed(render.KeyEscape) {
			m.OpenMenu()
			m.openWindow(ctx)
			return true
		}
		return false
	}
	m.openWindow(ctx)
	if m.window.Update(ctx) {
		if !m.window.IsOpen() {
			m.open = false
			m.Publish(ctx)
		}
		return true
	}
	return true
}

func (m *EscapeMenu) RequestCharacterSelect(ctx client.Context) {
	m.pending = true
	m.status = "Requesting character select..."
	if ctx.Network == nil {
		m.pending = false
		m.status = "Character select failed: not connected"
		m.refresh(ctx)
		return
	}
	if err := ctx.Network.SendRestart(network.RestartTypeCharSelect); err != nil {
		m.pending = false
		m.status = fmt.Sprintf("Character select failed: %v", err)
		m.refresh(ctx)
		return
	}
	m.refresh(ctx)
}

func (m *EscapeMenu) ApplyRestartAck(ack network.RestartAck) bool {
	if !m.open || !m.pending {
		return false
	}
	if ack.Allowed {
		m.status = "Returning to character select..."
		m.refresh(m.ctx)
		return true
	}
	m.pending = false
	m.status = "Please wait before changing characters."
	m.refresh(m.ctx)
	return false
}

func (m *EscapeMenu) ConsumeAction() EscapeMenuAction {
	action := m.action
	m.action = EscapeMenuActionNone
	return action
}

func (m *EscapeMenu) Draw(screen *render.Image, ctx client.Context, width, height int) {}

func (m *EscapeMenu) CursorAction(ctx client.Context) (int, bool) {
	if !m.open {
		return 0, false
	}
	return CursorActionDefault, true
}

func (m *EscapeMenu) IsOpen() bool {
	return m.open
}

func (m *EscapeMenu) Pending() bool {
	return m.pending
}

func (m *EscapeMenu) Status() string {
	return m.status
}

func (m *EscapeMenu) Action() EscapeMenuAction {
	return m.action
}

func (m *EscapeMenu) ensureWindow() {
	if m.window.width == 0 {
		m.window = NewWindowState(escapeMenuWidth, escapeMenuHeight)
	}
}

func (m *EscapeMenu) openWindow(ctx client.Context) {
	m.ensureWindow()
	if !m.window.IsOpen() {
		m.window.Open(ctx, m.widgetTree(ctx))
	}
	m.Publish(ctx)
}

func (m *EscapeMenu) Publish(ctx client.Context) {
	m.ensureWindow()
	if ctx.UIManager == nil {
		return
	}
	if !m.open || !m.window.IsOpen() {
		ctx.UIManager.Clear()
		return
	}
	ctx.UIManager.SetRoot(m.window.Widget())
}

func (m *EscapeMenu) refresh(ctx client.Context) {
	m.ensureWindow()
	if !m.open || !m.window.IsOpen() {
		return
	}
	m.window.SetRoot(m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *EscapeMenu) close(ctx client.Context) {
	m.open = false
	m.window.Close()
	m.Publish(ctx)
}

func (m *EscapeMenu) widgetTree(ctx client.Context) widget.Widget {
	return Window(
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
					m.close(ctx)
				}),
				rotheme.LargeButtonDisabled("Cancel", m.pending, func() {
					m.close(ctx)
				}),
				rotheme.LargeButton("Exit to Windows", func() {
					m.close(ctx)
					if ctx.RequestQuit != nil {
						ctx.RequestQuit()
					}
				}),
			).
				Padding(escapeMenuPad).
				Gap(escapeMenuGap).
				CrossAlign(primitives.CrossAxisStretch).
				Background(rotheme.Default.Colors.WindowBody),
		),
	)
}
