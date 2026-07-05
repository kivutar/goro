package ui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	confirmModalWidth   = 286
	confirmModalHeight  = 128
	confirmModalFooterH = 42
)

type ConfirmModal struct {
	open     bool
	title    string
	message  string
	onOK     func()
	onCancel func()
	window   WindowState
	ctx      client.Context
}

func (m *ConfirmModal) Open(ctx client.Context, title, message string, onOK, onCancel func()) {
	m.open = true
	m.title = title
	m.message = message
	m.onOK = onOK
	m.onCancel = onCancel
	m.ctx = ctx
	m.ensureWindow()
	m.window.Open(ctx, m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *ConfirmModal) Update(ctx client.Context) bool {
	m.ctx = ctx
	if !m.open {
		return false
	}
	if ctx.Input != nil {
		if ctx.Input.JustPressed(render.KeyEscape) {
			m.Cancel(ctx)
			return true
		}
		if ctx.Input.JustPressed(render.KeyEnter) {
			m.Confirm(ctx)
			return true
		}
	}
	m.openWindow(ctx)
	if m.window.Update(ctx) {
		m.Publish(ctx)
		return true
	}
	m.Publish(ctx)
	return true
}

func (m *ConfirmModal) IsOpen() bool {
	return m.open
}

func (m *ConfirmModal) Confirm(ctx client.Context) {
	m.close(ctx)
	if m.onOK != nil {
		m.onOK()
	}
}

func (m *ConfirmModal) Cancel(ctx client.Context) {
	m.close(ctx)
	if m.onCancel != nil {
		m.onCancel()
	}
}

func (m *ConfirmModal) Close(ctx client.Context) {
	m.close(ctx)
}

func (m *ConfirmModal) Publish(ctx client.Context) {
	if ctx.UIManager == nil {
		return
	}
	m.window.Publish(ctx)
}

func (m *ConfirmModal) ensureWindow() {
	if m.window.width != 0 {
		return
	}
	m.window = NewWindowState(confirmModalWidth, confirmModalHeight)
	m.window.SetCloseOnEscape(false)
}

func (m *ConfirmModal) openWindow(ctx client.Context) {
	m.ensureWindow()
	if !m.window.IsOpen() {
		m.window.Open(ctx, m.widgetTree(ctx))
	}
}

func (m *ConfirmModal) close(ctx client.Context) {
	m.open = false
	m.window.Close()
	m.Publish(ctx)
}

func (m *ConfirmModal) widgetTree(ctx client.Context) widget.Widget {
	okW := float32(ButtonLabelWidth("OK"))
	cancelW := float32(ButtonLabelWidth("Cancel"))
	return Window(
		Title(m.title),
		CloseButton(false),
		Size(confirmModalWidth, confirmModalHeight),
		FooterHeight(confirmModalFooterH),
		FooterPadding(18),
		Content(
			primitives.Box(
				rotheme.Text(m.message),
			).
				PaddingTop(22).
				PaddingLeft(28).
				PaddingRight(28),
		),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("OK", func() {
					m.Confirm(ctx)
				}).
					Width(okW),
				rotheme.Button("Cancel", func() {
					m.Cancel(ctx)
				}).
					Width(cancelW),
			).
				Gap(8),
		),
	)
}
