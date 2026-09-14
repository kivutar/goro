package ui

import (
	"fmt"
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	chatShortcutsW = 280
	chatShortcutsH = ROWindowTitleHeight + 10*28 + 20 + ROWindowFooterHeight
)

type ChatShortcutsWindow struct {
	Window
	commands config.ChatShortcuts
	loaded   bool
	fields   [10]*textfield.Widget
	selected int
	console  *ChatConsole
	onView   func()
}

func (w *ChatShortcutsWindow) load(ctx Context) {
	if w.loaded {
		return
	}
	w.loaded = true
	var err error
	w.commands, err = config.LoadChatShortcuts(ctx.Config.ChatShortcuts)
	if err != nil {
		glog.Warnf("load chat shortcuts: %v", err)
	}
}

func (w *ChatShortcutsWindow) Command(ctx Context, slot int) string {
	w.load(ctx)
	if slot < 0 || slot >= len(w.commands) {
		return ""
	}
	return w.commands[slot]
}

func (w *ChatShortcutsWindow) Toggle(ctx Context, console *ChatConsole, onView func()) {
	if w.IsOpen() {
		w.Close()
		return
	}
	w.load(ctx)
	w.EnsureWindow(chatShortcutsW, chatShortcutsH)
	w.ctx, w.console, w.onView = ctx, console, onView
	w.Window.Open(ctx, w.widgetTree())
	w.Publish(ctx)
}

func (w *ChatShortcutsWindow) Update(ctx Context) bool {
	if !w.IsOpen() {
		return false
	}
	w.ctx = ctx
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *ChatShortcutsWindow) UpdateKeyboardInput(ctx Context) bool {
	if !w.IsOpen() || ctx.Input == nil {
		return false
	}
	editing := false
	for slot, field := range w.fields {
		if field != nil && field.IsFocused() {
			w.selected = slot
			editing = true
		}
	}
	if !editing {
		return false
	}
	if w.escapePressed(ctx) {
		w.Close()
		return true
	}
	return ctx.Input.JustPressed(input.KeyEnter) || ctx.Input.JustPressed(input.KeyArrowUp) || ctx.Input.JustPressed(input.KeyArrowDown)
}

func (w *ChatShortcutsWindow) KeyboardShortcutsBlocked() bool {
	if w.IsOpen() {
		for _, field := range w.fields {
			if field != nil && field.IsFocused() {
				return true
			}
		}
	}
	return false
}

func (w *ChatShortcutsWindow) Rebind(ctx Context, console *ChatConsole, onView func()) {
	w.ctx, w.console, w.onView = ctx, console, onView
	if w.IsOpen() {
		w.RebindContent(ctx, w.widgetTree())
	}
}

// SelectEmotion lets the existing emote picker edit the last focused slot.
// Returning false keeps the picker's ordinary console behavior when closed.
func (w *ChatShortcutsWindow) SelectEmotion(command string) bool {
	if !w.IsOpen() {
		return false
	}
	w.setCommand(w.selected, command)
	return true
}

func (w *ChatShortcutsWindow) setCommand(slot int, command string) {
	command = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == 0 {
			return -1
		}
		return r
	}, command)
	if field := w.fields[slot]; field != nil && field.Text() != command {
		field.SetText(command)
	}
	if w.commands[slot] == command {
		return
	}
	w.commands[slot] = command
	if _, err := config.SaveChatShortcuts(w.commands); err != nil {
		glog.Warnf("save chat shortcuts: %v", err)
		if w.console != nil {
			w.console.AddErrorMessage("Could not save chat shortcuts: %s", err)
		}
	}
}

func (w *ChatShortcutsWindow) widgetTree() widget.Widget {
	rows := make([]widget.Widget, 0, len(w.commands))
	for slot, command := range w.commands {
		w.fields[slot] = rotheme.TextField(command, textfield.TypeText,
			func(value string) { w.selected = slot; w.setCommand(slot, value) }, nil,
			textfield.MaxLength(consoleMaxInput),
		)
		rows = append(rows, primitives.HBox(
			primitives.Box(rotheme.Label(fmt.Sprintf("Alt + %d", (slot+1)%10))).Width(54),
			primitives.Expanded(w.fields[slot]),
		).Height(24).Gap(8).CrossAlign(primitives.CrossAxisCenter))
	}
	return Win(
		Title("Shortcut List"), CloseButton(true), OnClose(w.Close),
		Size(chatShortcutsW, chatShortcutsH),
		Content(primitives.Box(rows...).Padding(12).Gap(4)),
		Footer(primitives.Expanded(primitives.Box()), rotheme.Button("View", func() {
			if w.onView != nil {
				w.onView()
			}
		})),
	)
}
