package ui

import (
	"image/color"
	"log"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	deathModalWidth  = 312
	deathModalHeight = 190
	deathModalPad    = 16
	deathModalGap    = 8
)

type DeathModal struct {
	open    bool
	pending DeathModalAction
	window  WindowState
	ctx     client.Context
}

type DeathModalAction int

const (
	DeathModalActionNone DeathModalAction = iota
	DeathModalActionSavePoint
	DeathModalActionCharSelect
)

func (m *DeathModal) OpenDeath() {
	m.open = true
	m.pending = DeathModalActionNone
}

func (m *DeathModal) Reset() {
	if m.window.IsOpen() {
		m.window.Close()
		m.Publish(m.ctx)
	}
	*m = DeathModal{}
}

func (m *DeathModal) ClearIfAlive(ctx client.Context) {
	if !m.open || ctx.Session == nil {
		return
	}
	if ctx.Session.Vitals.HP > 0 || ctx.Session.Selected.HP > 0 {
		m.Reset()
	}
}

func (m *DeathModal) ApplyRestartAck(ack network.RestartAck) bool {
	if !m.open || m.pending != DeathModalActionCharSelect {
		return false
	}
	if ack.Allowed {
		m.refresh(m.ctx)
		return true
	}
	m.pending = DeathModalActionNone
	m.refresh(m.ctx)
	return false
}

func (m *DeathModal) Update(ctx client.Context) bool {
	m.ctx = ctx
	if !m.open {
		return false
	}
	m.openWindow(ctx)
	if m.window.Update(ctx) {
		m.Publish(ctx)
		return true
	}
	m.Publish(ctx)
	return true
}

func (m *DeathModal) ReturnToSavePoint(ctx client.Context) {
	m.pending = DeathModalActionSavePoint
	if ctx.Network == nil {
		m.pending = DeathModalActionNone
		m.refresh(ctx)
		return
	}
	if err := ctx.Network.SendRestart(network.RestartTypeRespawn); err != nil {
		m.pending = DeathModalActionNone
		log.Printf("death modal respawn failed: %v", err)
	}
	m.refresh(ctx)
}

func (m *DeathModal) RequestCharacterSelect(ctx client.Context) {
	m.pending = DeathModalActionCharSelect
	if ctx.Network == nil {
		m.pending = DeathModalActionNone
		m.refresh(ctx)
		return
	}
	if err := ctx.Network.SendRestart(network.RestartTypeCharSelect); err != nil {
		m.pending = DeathModalActionNone
		log.Printf("death modal character select failed: %v", err)
	}
	m.refresh(ctx)
}

func (m *DeathModal) ExitToWindows(ctx client.Context) {
	m.open = false
	m.window.Close()
	m.Publish(ctx)
	if ctx.RequestQuit != nil {
		ctx.RequestQuit()
	}
}

func (m *DeathModal) Draw(screen *render.Image, ctx client.Context, width, height int) {
	if !m.open || screen == nil {
		return
	}
	DrawSurface(screen, 0, 0, width, height, color.RGBA{A: 112}, color.RGBA{})
}

func (m *DeathModal) IsOpen() bool {
	return m.open
}

func (m *DeathModal) PendingAction() DeathModalAction {
	return m.pending
}

func (m *DeathModal) ensureWindow(ctx client.Context) {
	if m.window.width != 0 {
		return
	}
	m.window = NewWindowState(deathModalWidth, deathModalHeight)
	m.window.SetCloseOnEscape(false)
}

func (m *DeathModal) openWindow(ctx client.Context) {
	m.ensureWindow(ctx)
	if !m.window.IsOpen() {
		m.window.Open(ctx, m.widgetTree(ctx))
	}
	m.Publish(ctx)
}

func (m *DeathModal) Publish(ctx client.Context) {
	if ctx.UIManager == nil {
		return
	}
	if !m.open || !m.window.IsOpen() {
		ctx.UIManager.Clear()
		return
	}
	ctx.UIManager.SetRoot(m.window.Widget())
}

func (m *DeathModal) refresh(ctx client.Context) {
	if !m.open || !m.window.IsOpen() {
		return
	}
	m.window.SetRoot(m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *DeathModal) widgetTree(ctx client.Context) widget.Widget {
	return Window(
		Title("You have died"),
		CloseButton(false),
		Size(deathModalWidth, deathModalHeight),
		Content(
			primitives.Box(
				rotheme.Text("Choose what to do next."),
				rotheme.LargeButtonDisabled("Return to Save Point", m.pending != DeathModalActionNone, func() {
					m.ReturnToSavePoint(ctx)
				}),
				rotheme.LargeButtonDisabled("Character Select", m.pending != DeathModalActionNone, func() {
					m.RequestCharacterSelect(ctx)
				}),
				rotheme.LargeButton("Exit to Windows", func() {
					m.ExitToWindows(ctx)
				}),
			).
				Padding(deathModalPad).
				Gap(deathModalGap).
				CrossAlign(primitives.CrossAxisStretch),
		),
	)
}
