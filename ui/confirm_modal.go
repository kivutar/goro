package ui

import (
	"github.com/kivutar/goro/input"
	"strings"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	smallPromptWidth        = 286
	smallPromptHeight       = 128
	smallPromptContentH     = 42
	smallPromptSidePad      = 12
	smallPromptLineH        = 14
	smallPromptDefaultLines = 2
	smallPromptMaxLines     = 4
	smallPromptLineMaxRunes = 42
)

type ConfirmModal struct {
	Window
	title       string
	message     string
	onOK        func()
	onCancel    func()
	okOnly      bool
	ctx         client.Context
	openedFrame uint64
}

func (m *ConfirmModal) Open(ctx client.Context, title, message string, onOK, onCancel func()) {
	m.title = title
	m.message = message
	m.onOK = onOK
	m.onCancel = onCancel
	m.okOnly = false
	m.openPrompt(ctx)
}

func (m *ConfirmModal) OpenAlert(ctx client.Context, title, message string, onOK func()) {
	m.title = title
	m.message = message
	m.onOK = onOK
	m.onCancel = nil
	m.okOnly = true
	m.openPrompt(ctx)
}

func (m *ConfirmModal) openPrompt(ctx client.Context) {
	m.ctx = ctx
	m.openedFrame = 0
	if ctx.Input != nil {
		m.openedFrame = ctx.Input.FrameID()
	}
	m.EnsureWindow(smallPromptWidth, m.promptHeight())
	m.CloseOnEsc = false // Escape is handled here, including opening-frame suppression.
	m.SetSize(smallPromptWidth, m.promptHeight())
	m.Window.Open(ctx, m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *ConfirmModal) Update(ctx client.Context) bool {
	m.ctx = ctx
	if !m.Window.IsOpen() {
		return false
	}
	// Widget callbacks can open this modal before the triggering key reaches
	// input.State. Only accept keyboard confirmation in a later input frame.
	if ctx.Input != nil && ctx.Input.FrameID() != m.openedFrame {
		if ctx.Input.JustPressed(input.KeyEscape) {
			if m.okOnly {
				m.Confirm(ctx)
			} else {
				m.Cancel(ctx)
			}
			return true
		}
		if ctx.Input.JustPressed(input.KeyEnter) {
			m.Confirm(ctx)
			return true
		}
	}
	m.openWindow(ctx)
	if m.Window.Update(ctx) {
		m.Publish(ctx)
		return true
	}
	m.Publish(ctx)
	return true
}

func (m *ConfirmModal) Confirm(ctx client.Context) {
	m.Close(ctx)
	if m.onOK != nil {
		m.onOK()
	}
}

func (m *ConfirmModal) Cancel(ctx client.Context) {
	m.Close(ctx)
	if m.onCancel != nil {
		m.onCancel()
	}
}

func (m *ConfirmModal) Close(ctx client.Context) {
	m.Window.Close()
	m.Publish(ctx)
}

func (m *ConfirmModal) openWindow(ctx client.Context) {
	m.EnsureWindow(smallPromptWidth, m.promptHeight())
	if m.content == nil {
		m.Window.Open(ctx, m.widgetTree(ctx))
	}
}

func (m *ConfirmModal) promptHeight() int {
	return ROWindowTitleHeight + smallPromptContentH + smallPromptLineH*(m.messageMaxLines()-1) + ROWindowFooterHeight
}

func (m *ConfirmModal) messageMaxLines() int {
	return max(smallPromptDefaultLines, smallPromptVisibleLineCount(m.message, smallPromptMaxLines))
}

func (m *ConfirmModal) widgetTree(ctx client.Context) widget.Widget {
	footer := []widget.Widget{
		primitives.Expanded(primitives.Box()),
		rotheme.Button("OK", func() {
			m.Confirm(ctx)
		}),
		rotheme.Button("Cancel", func() {
			m.Cancel(ctx)
		}),
	}
	if m.okOnly {
		footer = []widget.Widget{
			primitives.Expanded(primitives.Box()),
			rotheme.Button("OK", func() {
				m.Confirm(ctx)
			}),
		}
	}
	return Win(
		Title(m.title),
		CloseButton(false),
		Size(smallPromptWidth, float32(m.promptHeight())),
		Content(smallPromptContent(m.message, m.messageMaxLines())),
		Footer(footer...),
	)
}

func smallPromptContent(message string, maxLines int) widget.Widget {
	return primitives.Box(
		primitives.Expanded(primitives.Box()),
		smallPromptMessage(message, maxLines),
		primitives.Expanded(primitives.Box()),
	).
		PaddingLeft(smallPromptSidePad).
		PaddingRight(smallPromptSidePad)
}

func smallPromptMessage(message string, maxLines int) widget.Widget {
	lines := smallPromptLines(message, maxLines)
	rows := make([]widget.Widget, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, smallPromptLine(line))
	}
	return primitives.Box(rows...).Height(float32(smallPromptLineH * maxLines))
}

func smallPromptLines(message string, maxLines int) []string {
	if maxLines < 1 {
		maxLines = 1
	}
	var lines []string
	for _, paragraph := range strings.Split(message, "\n") {
		wrapped := wrapSmallPromptLine(paragraph)
		lines = append(lines, wrapped...)
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for len(lines) < maxLines {
		lines = append(lines, "")
	}
	return lines
}

func smallPromptVisibleLineCount(message string, maxLines int) int {
	lines := smallPromptLines(message, maxLines)
	count := len(lines)
	for count > 1 && lines[count-1] == "" {
		count--
	}
	return count
}

func wrapSmallPromptLine(line string) []string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := ""
	for _, word := range words {
		if current == "" {
			current = word
			continue
		}
		if len([]rune(current))+1+len([]rune(word)) <= smallPromptLineMaxRunes {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func smallPromptLine(line string) widget.Widget {
	if line == "" {
		return primitives.Box().Height(smallPromptLineH)
	}
	return primitives.Box(
		rotheme.Text(line).
			MaxLines(1),
	).
		Height(smallPromptLineH)
}
