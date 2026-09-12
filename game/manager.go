package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

type Mode interface {
	Name() string
	// Enter may redirect to another mode if this one cannot be entered.
	Enter(client.Context) Mode
	Update(client.Context) (Mode, error)
	Draw(client.Context, *render.Frame)
}

type overlayMode interface {
	DrawOverlay(client.Context, *render.Frame)
}

type uiOverlayMode interface {
	DrawUIOverlay(client.Context, *render.Frame)
}

type frameSubmittedMode interface {
	FrameSubmitted()
}

type Manager struct {
	ctx  client.Context
	mode Mode
}

func NewManager(ctx client.Context, mode Mode) *Manager {
	m := &Manager{ctx: ctx}
	m.enter(mode)
	return m
}

func (m *Manager) enter(mode Mode) {
	for mode != nil {
		m.mode = mode
		mode = mode.Enter(m.ctx)
	}
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
		m.enter(next)
	}
	return nil
}

func (m *Manager) Draw(screen *render.Frame) {
	if m.mode != nil {
		m.mode.Draw(m.ctx, screen)
	}
}

func (m *Manager) DrawOverlay(screen *render.Frame) {
	if mode, ok := m.mode.(overlayMode); ok {
		mode.DrawOverlay(m.ctx, screen)
	}
}

func (m *Manager) DrawUIOverlay(screen *render.Frame) {
	if mode, ok := m.mode.(uiOverlayMode); ok {
		mode.DrawUIOverlay(m.ctx, screen)
	}
}

func (m *Manager) FrameSubmitted() {
	if mode, ok := m.mode.(frameSubmittedMode); ok {
		mode.FrameSubmitted()
	}
}
