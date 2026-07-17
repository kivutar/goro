package ui

import (
	"image/color"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
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
	Window
	pending DeathModalAction
	ctx     client.Context
}

type DeathModalAction int

const (
	DeathModalActionNone DeathModalAction = iota
	DeathModalActionSavePoint
	DeathModalActionCharSelect
	DeathModalActionExit
)

func (m *DeathModal) OpenDeath() {
	m.pending = DeathModalActionNone
	m.EnsureWindow(deathModalWidth, deathModalHeight)
	m.Window.open = true
}

func (m *DeathModal) Reset() {
	m.Window.Close()
	m.Publish(m.ctx)
	*m = DeathModal{}
}

func (m *DeathModal) ClearIfAlive(ctx client.Context) {
	if !m.IsOpen() || ctx.Session == nil {
		return
	}
	if ctx.Session.Vitals.HP > 0 || ctx.Session.Selected.HP > 0 {
		m.Reset()
	}
}

func (m *DeathModal) ApplyRestartAck(ack network.RestartAck) bool {
	if !m.IsOpen() || m.pending != DeathModalActionCharSelect {
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

func (m *DeathModal) ApplyQuitGameAck(ack network.QuitGameAck) bool {
	if !m.IsOpen() || m.pending != DeathModalActionExit {
		return false
	}
	if ack.Allowed {
		m.refresh(m.ctx)
		return true
	}
	m.pending = DeathModalActionNone
	m.refresh(m.ctx)
	return true
}

func (m *DeathModal) Update(ctx client.Context) bool {
	m.ctx = ctx
	if !m.IsOpen() {
		return false
	}
	m.openWindow(ctx)
	if m.Window.Update(ctx) {
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
		glog.Warnf("death modal respawn failed: %v", err)
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
		glog.Warnf("death modal character select failed: %v", err)
	}
	m.refresh(ctx)
}

func (m *DeathModal) ExitToWindows(ctx client.Context) {
	m.pending = DeathModalActionExit
	if ctx.Network == nil {
		m.pending = DeathModalActionNone
		m.refresh(ctx)
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
		return
	}
	if err := ctx.Network.SendQuitGame(); err != nil {
		m.pending = DeathModalActionNone
		glog.Warnf("death modal quit failed: %v", err)
	}
	m.refresh(ctx)
}

func (m *DeathModal) Draw(screen *render.Frame, ctx client.Context, width, height int) {
	if !m.IsOpen() || screen == nil {
		return
	}
	DrawSurface(screen, 0, 0, width, height, color.RGBA{A: 112}, color.RGBA{})
}

func (m *DeathModal) PendingAction() DeathModalAction {
	return m.pending
}

func (m *DeathModal) openWindow(ctx client.Context) {
	m.EnsureWindow(deathModalWidth, deathModalHeight)
	if m.content == nil {
		m.Window.Open(ctx, m.widgetTree(ctx))
	}
	m.Publish(ctx)
}

func (m *DeathModal) refresh(ctx client.Context) {
	if !m.IsOpen() {
		return
	}
	m.SetContent(m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *DeathModal) widgetTree(ctx client.Context) widget.Widget {
	return Win(
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
