package ui

import (
	"github.com/kivutar/goro/input"
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	textPromptW        = 306
	textPromptContentH = 70
)

type TextPromptAction struct {
	Text      string
	Submitted bool
}

type TextPromptWindow struct {
	Window
	title       string
	label       string
	placeholder string
	value       string
	maxLength   int
	inputField  *textfield.Widget
	action      TextPromptAction
}

func (w *TextPromptWindow) Open(ctx Context, title, label, placeholder string, maxLength int) {
	w.EnsureWindow(textPromptW, ROWindowTitleHeight+textPromptContentH+ROWindowFooterHeight)
	w.ctx = ctx
	w.title = title
	w.label = label
	w.placeholder = placeholder
	w.value = ""
	w.maxLength = maxLength
	w.inputField = nil
	w.action = TextPromptAction{}
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.focusInput()
	w.Publish(ctx)
}

func (w *TextPromptWindow) Update(ctx Context) bool {
	w.EnsureWindow(textPromptW, ROWindowTitleHeight+textPromptContentH+ROWindowFooterHeight)
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	if w.submitFromFocusedEnter(ctx) {
		w.Publish(ctx)
		return true
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *TextPromptWindow) Rebind(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.inputField = nil
	w.SetContent(w.widgetTree(ctx))
	w.focusInput()
	w.Publish(ctx)
}

func (w *TextPromptWindow) PopAction() TextPromptAction {
	action := w.action
	w.action = TextPromptAction{}
	return action
}

func (w *TextPromptWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title(w.title),
		CloseButton(true),
		OnClose(w.Close),
		Size(textPromptW, ROWindowTitleHeight+textPromptContentH+ROWindowFooterHeight),
		Content(
			primitives.Box(
				rotheme.Label(w.label),
				primitives.Box(w.input(ctx)).
					Height(24).
					CrossAlign(primitives.CrossAxisStretch),
			).
				Padding(14).
				Gap(8),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("OK", func() {
				w.submit(ctx)
			}),
			rotheme.Button("Cancel", w.Close),
		),
	)
}

func (w *TextPromptWindow) input(ctx Context) *textfield.Widget {
	if w.inputField != nil {
		return w.inputField
	}
	options := []textfield.Option{
		textfield.Placeholder(w.placeholder),
	}
	if w.maxLength > 0 {
		options = append(options, textfield.MaxLength(w.maxLength))
	}
	w.inputField = rotheme.TextField(
		w.value,
		textfield.TypeText,
		func(value string) {
			w.value = value
		},
		func(string) {
			w.submit(ctx)
		},
		options...,
	)
	return w.inputField
}

func (w *TextPromptWindow) submit(ctx Context) {
	text := strings.TrimSpace(w.value)
	if text == "" {
		return
	}
	w.action = TextPromptAction{Text: text, Submitted: true}
	w.Close()
	w.Publish(ctx)
}

func (w *TextPromptWindow) submitFromFocusedEnter(ctx Context) bool {
	if ctx.Input == nil || w.inputField == nil || !w.inputField.IsFocused() {
		return false
	}
	if !ctx.Input.JustPressed(input.KeyEnter) {
		return false
	}
	w.submit(ctx)
	return true
}

func (w *TextPromptWindow) focusInput() {
	if w.inputField != nil {
		w.inputField.SetFocused(true)
	}
}
