package game

import (
	"sync/atomic"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
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
	ctx           client.Context
	mode          Mode
	login         atomic.Bool
	script        *luaScript
	scriptBlocked [input.GamepadButtonCount]bool
}

func NewManager(ctx client.Context, mode Mode) *Manager {
	m := &Manager{ctx: ctx}
	m.enter(mode)
	return m
}

// Close releases the active mode after the render loop stops, outside any Lua callback.
func (m *Manager) Close() {
	m.closeScript()
	if leaving, ok := m.mode.(interface{ Leave() }); ok {
		leaving.Leave()
	}
	m.mode = nil
	m.login.Store(false)
}

func (m *Manager) enter(mode Mode) {
	for mode != nil {
		m.closeScript()
		if m.ctx.Input != nil {
			for button := input.GamepadButton(0); button < input.GamepadButtonCount; button++ {
				m.scriptBlocked[button] = m.ctx.Input.GamepadDown(button)
			}
		}
		if leaving, ok := m.mode.(interface{ Leave() }); ok {
			leaving.Leave()
		}
		m.mode = mode
		_, login := mode.(*LoginMode)
		m.login.Store(login)
		mode = mode.Enter(m.ctx)
	}
}

// InLogin can be queried by platform UI threads without reading the live mode.
func (m *Manager) InLogin() bool {
	return m.login.Load()
}

func (m *Manager) UpdateContext(ctx client.Context) {
	m.ctx = ctx
}

func (m *Manager) HandleGamepadInput(ctx client.Context, dt float64) input.GamepadCapture {
	for button := range m.scriptBlocked {
		if ctx.Input == nil || !ctx.Input.GamepadDown(input.GamepadButton(button)) || ctx.Input.GamepadJustReleased(input.GamepadButton(button)) {
			m.scriptBlocked[button] = false
		}
	}
	m.syncScript(ctx)
	return m.script.handleGamepad(ctx, dt)
}

func (m *Manager) HandleKeyPress(ctx client.Context, code input.KeyCode) {
	m.syncScript(ctx)
	if handler, ok := m.mode.(interface {
		HandleKeyPress(client.Context, input.KeyCode)
	}); ok {
		handler.HandleKeyPress(ctx, code)
	}
}

func (m *Manager) PrepareTextInput(ctx client.Context, code input.KeyCode) bool {
	if filter, ok := m.mode.(interface {
		PrepareTextInput(client.Context, input.KeyCode) bool
	}); ok {
		return filter.PrepareTextInput(ctx, code)
	}
	return false
}

func (m *Manager) PrepareKeyInput(ctx client.Context, code input.KeyCode, mods gpucontext.Modifiers) {
	if preparer, ok := m.mode.(interface {
		PrepareKeyInput(client.Context, input.KeyCode, gpucontext.Modifiers)
	}); ok {
		preparer.PrepareKeyInput(ctx, code, mods)
	}
}

func (m *Manager) Update() error {
	if m.mode == nil {
		return nil
	}

	m.syncScript(m.ctx)
	// Gameplay input and ticks retain WorldMode's modal/death ordering.
	if m.script != nil && m.script.mode == nil {
		if err := m.script.inputFrame(false); err != nil {
			m.script.fail("input", err)
		}
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
