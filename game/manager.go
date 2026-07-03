package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

type Mode interface {
	Name() string
	Enter(client.Context)
	Update(client.Context) (Mode, error)
	Draw(client.Context, *render.Image)
}

type overlayMode interface {
	DrawOverlay(client.Context, *render.Image)
}

type Manager struct {
	ctx  client.Context
	mode Mode
}

func NewManager(ctx client.Context, mode Mode) *Manager {
	m := &Manager{ctx: ctx, mode: mode}
	if m.mode != nil {
		m.mode.Enter(ctx)
	}
	return m
}

func (m *Manager) UpdateContext(ctx client.Context) {
	m.ctx = ctx
}

func (m *Manager) Update() error {
	if m.mode == nil {
		return nil
	}

	next, err := m.mode.Update(m.ctx)
	if err != nil {
		return err
	}
	if next != nil {
		m.mode = next
		m.mode.Enter(m.ctx)
	}
	return nil
}

func (m *Manager) Draw(screen *render.Image) {
	if m.mode != nil {
		m.mode.Draw(m.ctx, screen)
	}
}

func (m *Manager) DrawOverlay(screen *render.Image) {
	if mode, ok := m.mode.(overlayMode); ok {
		mode.DrawOverlay(m.ctx, screen)
	}
}
