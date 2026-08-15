package ui

import (
	"fmt"
	"unicode/utf8"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	storagePasswordWindowW  = 326
	storagePasswordContentH = 116
	storagePasswordWindowH  = ROWindowTitleHeight + storagePasswordContentH + ROWindowFooterHeight
	storagePasswordLabelW   = 104
	storagePasswordFieldH   = 24
	storagePasswordMinLen   = 4
	storagePasswordMaxLen   = 8
)

type StoragePasswordMode uint8

const (
	StoragePasswordModeEnter StoragePasswordMode = iota
	StoragePasswordModeSet
	StoragePasswordModeLocked
)

type StoragePasswordActionKind uint8

const (
	StoragePasswordActionNone StoragePasswordActionKind = iota
	StoragePasswordActionCheck
	StoragePasswordActionChange
	StoragePasswordActionCancel
)

type StoragePasswordAction struct {
	Kind     StoragePasswordActionKind
	Password string
}

type StoragePasswordWindow struct {
	Window
	mode          StoragePasswordMode
	message       string
	password      string
	confirmation  string
	passwordField *textfield.Widget
	confirmField  *textfield.Widget
	waiting       bool
	action        StoragePasswordAction
}

func (w *StoragePasswordWindow) OpenPrompt(ctx Context, mode StoragePasswordMode, message string) {
	w.EnsureWindow(storagePasswordWindowW, storagePasswordWindowH)
	w.mode = mode
	w.message = message
	w.password = ""
	w.confirmation = ""
	w.passwordField = nil
	w.confirmField = nil
	w.waiting = false
	w.action = StoragePasswordAction{}
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.focusFirstField()
	w.Publish(ctx)
}

func (w *StoragePasswordWindow) Update(ctx Context) bool {
	w.EnsureWindow(storagePasswordWindowW, storagePasswordWindowH)
	if !w.IsOpen() {
		return false
	}
	if ctx.Input != nil {
		if ctx.Input.JustPressed(input.KeyEscape) {
			w.cancel(ctx)
			return true
		}
		if ctx.Input.JustPressed(input.KeyEnter) && w.focusedField() {
			w.submit(ctx)
			return true
		}
	}
	w.Window.Update(ctx)
	w.Publish(ctx)
	// Storage authentication is modal: pointer input outside the window must
	// not reach the map while the server is waiting for a reply.
	return true
}

func (w *StoragePasswordWindow) PopAction() StoragePasswordAction {
	action := w.action
	w.action = StoragePasswordAction{}
	return action
}

func (w *StoragePasswordWindow) ShowMessage(ctx Context, message string) {
	if !w.IsOpen() {
		return
	}
	w.message = message
	w.waiting = false
	w.password = ""
	w.confirmation = ""
	w.rebuild(ctx)
}

func (w *StoragePasswordWindow) CloseFromServer(ctx Context) {
	w.action = StoragePasswordAction{}
	w.Window.Close()
	w.Publish(ctx)
}

func (w *StoragePasswordWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Storage Password"),
		CloseButton(true),
		OnClose(func() { w.cancel(ctx) }),
		Size(storagePasswordWindowW, storagePasswordWindowH),
		Content(w.content(ctx)),
		Footer(w.footer(ctx)...),
	)
}

func (w *StoragePasswordWindow) content(ctx Context) widget.Widget {
	children := make([]widget.Widget, 0, 3)
	switch w.mode {
	case StoragePasswordModeSet:
		children = append(children,
			w.fieldRow("New Password", w.passwordInput(ctx)),
			w.fieldRow("Confirm", w.confirmInput(ctx)),
		)
	case StoragePasswordModeEnter:
		children = append(children, w.fieldRow("Password", w.passwordInput(ctx)))
	}
	children = append(children, primitives.Expanded(primitives.Box()))
	if w.message != "" {
		children = append(children,
			primitives.Box(rotheme.Text(w.message).MaxLines(2)).Height(30),
		)
	}
	if w.mode == StoragePasswordModeLocked {
		children = append(children, primitives.Expanded(primitives.Box()))
	}
	return primitives.Box(children...).
		PaddingXY(14, 12).
		Gap(8).
		CrossAlign(primitives.CrossAxisStretch)
}

func (w *StoragePasswordWindow) footer(ctx Context) []widget.Widget {
	children := []widget.Widget{primitives.Expanded(primitives.Box())}
	if w.mode == StoragePasswordModeLocked {
		return append(children, rotheme.Button("Close", func() { w.cancel(ctx) }))
	}
	return append(children,
		rotheme.ButtonDisabled("OK", w.waiting, func() { w.submit(ctx) }),
		rotheme.Button("Cancel", func() { w.cancel(ctx) }),
	)
}

func (w *StoragePasswordWindow) fieldRow(label string, field *textfield.Widget) widget.Widget {
	return primitives.HBox(
		primitives.Box(rotheme.Label(label)).
			Width(storagePasswordLabelW).
			Height(storagePasswordFieldH),
		primitives.Expanded(
			primitives.Box(field).
				Height(storagePasswordFieldH),
		),
	).
		Gap(8).
		CrossAlign(primitives.CrossAxisCenter).
		Height(storagePasswordFieldH)
}

func (w *StoragePasswordWindow) passwordInput(ctx Context) *textfield.Widget {
	if w.passwordField == nil {
		w.passwordField = rotheme.TextField(
			w.password,
			textfield.TypePassword,
			func(value string) { w.password = value },
			func(string) { w.submit(ctx) },
			textfield.MaxLength(storagePasswordMaxLen),
			textfield.Disabled(w.waiting),
		)
	}
	return w.passwordField
}

func (w *StoragePasswordWindow) confirmInput(ctx Context) *textfield.Widget {
	if w.confirmField == nil {
		w.confirmField = rotheme.TextField(
			w.confirmation,
			textfield.TypePassword,
			func(value string) { w.confirmation = value },
			func(string) { w.submit(ctx) },
			textfield.MaxLength(storagePasswordMaxLen),
			textfield.Disabled(w.waiting),
		)
	}
	return w.confirmField
}

func (w *StoragePasswordWindow) submit(ctx Context) {
	if w.waiting || w.mode == StoragePasswordModeLocked {
		return
	}
	length := utf8.RuneCountInString(w.password)
	if length < storagePasswordMinLen || length > storagePasswordMaxLen {
		w.message = fmt.Sprintf("Use %d to %d characters.", storagePasswordMinLen, storagePasswordMaxLen)
		w.rebuild(ctx)
		return
	}
	if w.mode == StoragePasswordModeSet && w.password != w.confirmation {
		w.message = "The passwords do not match."
		w.confirmation = ""
		w.rebuild(ctx)
		return
	}
	kind := StoragePasswordActionCheck
	message := "Checking password..."
	if w.mode == StoragePasswordModeSet {
		kind = StoragePasswordActionChange
		message = "Changing password..."
	}
	w.action = StoragePasswordAction{Kind: kind, Password: w.password}
	w.password = ""
	w.confirmation = ""
	w.waiting = true
	w.message = message
	w.rebuild(ctx)
}

func (w *StoragePasswordWindow) cancel(ctx Context) {
	w.action = StoragePasswordAction{Kind: StoragePasswordActionCancel}
	w.Window.Close()
	w.Publish(ctx)
}

func (w *StoragePasswordWindow) rebuild(ctx Context) {
	w.passwordField = nil
	w.confirmField = nil
	w.SetContent(w.widgetTree(ctx))
	w.focusFirstField()
	w.Publish(ctx)
}

func (w *StoragePasswordWindow) focusFirstField() {
	if w.passwordField != nil && !w.waiting {
		w.passwordField.SetFocused(true)
	}
}

func (w *StoragePasswordWindow) focusedField() bool {
	return w.passwordField != nil && w.passwordField.IsFocused() ||
		w.confirmField != nil && w.confirmField.IsFocused()
}
