package ui

import (
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/ui/rotheme"
)

type LoginWindowDrawOptions struct {
	X, Y, W, H    int
	FooterH       int
	FormTopPad    int
	FieldGap      int
	FieldLeft     int
	FieldRightPad int
	FieldH        int
	Username      string
	Password      string
}

type LoginWindowCallbacks struct {
	OnSubmit func()
}

type LoginWindow struct {
	Username string
	Password string

	opts      LoginWindowDrawOptions
	callbacks LoginWindowCallbacks
	window    WindowState
	user      *textfield.Widget
	password  *textfield.Widget
}

func NewLoginWindow(opts LoginWindowDrawOptions, callbacks LoginWindowCallbacks) *LoginWindow {
	w := &LoginWindow{
		Username:  opts.Username,
		Password:  opts.Password,
		opts:      opts,
		callbacks: callbacks,
		window:    NewWindowState(opts.W, opts.H),
	}
	w.window.OpenAt(opts.X, opts.Y, w.widgetTree())
	return w
}

func (w *LoginWindow) SetOptions(opts LoginWindowDrawOptions) {
	if w == nil {
		return
	}
	sameLayout := loginWindowLayoutEqual(w.opts, opts)
	w.opts = opts
	w.window.SetAutoPosition(opts.X, opts.Y)
	w.window.SetSize(opts.W, opts.H)
	if sameLayout {
		return
	}
	w.rebuild()
}

func (w *LoginWindow) Widget() widget.Widget {
	if w == nil {
		return nil
	}
	return w.window.Widget()
}

func (w *LoginWindow) Update(ctx client.Context) bool {
	if w == nil {
		return false
	}
	return w.window.Update(ctx)
}

func (w *LoginWindow) rebuild() {
	userFocused, passwordFocused := w.fieldFocus()
	w.window.SetContent(w.widgetTree())
	if w.user != nil {
		w.user.SetFocused(userFocused)
	}
	if w.password != nil {
		w.password.SetFocused(passwordFocused)
	}
}

func (w *LoginWindow) widgetTree() widget.Widget {
	submit := func() {
		if w.callbacks.OnSubmit != nil {
			w.callbacks.OnSubmit()
		}
	}
	userFocused, passwordFocused := w.fieldFocus()
	username, passwordValue := w.fieldValues()
	user := rotheme.TextField(
		username,
		textfield.TypeText,
		func(v string) {
			w.Username = v
		},
		func(string) { submit() },
	)
	user.SetFocused(userFocused)
	password := rotheme.TextField(
		passwordValue,
		textfield.TypePassword,
		func(v string) {
			w.Password = v
		},
		func(string) { submit() },
	)
	password.SetFocused(passwordFocused)
	w.user = user
	w.password = password
	labelW := float32(w.opts.FieldLeft - 36)
	fieldW := float32(w.opts.W - w.opts.FieldLeft - w.opts.FieldRightPad)
	fieldH := float32(w.opts.FieldH)
	buttonW := float32(ButtonLabelWidth("Login"))
	return Window(
		Title("Login"),
		CloseButton(false),
		Size(float32(w.opts.W), float32(w.opts.H)),
		FooterHeight(float32(w.opts.FooterH)),
		FooterPadding(float32(w.opts.FieldRightPad)),
		Content(
			primitives.Box(
				primitives.HBox(
					primitives.Box(
						rotheme.Text("Account").
							LineHeight(fieldH/rotheme.Default.Typography.TextSize),
					).
						Width(labelW).
						Height(fieldH),
					primitives.Box(user).
						Width(fieldW).
						Height(fieldH),
				).
					CrossAlign(primitives.CrossAxisCenter).
					Gap(12),
				primitives.HBox(
					primitives.Box(
						rotheme.Text("Password").
							LineHeight(fieldH/rotheme.Default.Typography.TextSize),
					).
						Width(labelW).
						Height(fieldH),
					primitives.Box(password).
						Width(fieldW).
						Height(fieldH),
				).
					CrossAlign(primitives.CrossAxisCenter).
					Gap(12),
			).
				PaddingTop(float32(w.opts.FormTopPad)).
				PaddingLeft(24).
				PaddingRight(float32(w.opts.FieldRightPad)).
				Gap(float32(w.opts.FieldGap)),
		),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Login", submit).
					Width(buttonW),
			),
		),
	)
}

func (w *LoginWindow) fieldFocus() (bool, bool) {
	if w.user == nil && w.password == nil {
		return true, false
	}
	userFocused := w.user != nil && w.user.IsFocused()
	passwordFocused := w.password != nil && w.password.IsFocused()
	if !userFocused && !passwordFocused {
		return true, false
	}
	return userFocused, passwordFocused
}

func (w *LoginWindow) fieldValues() (string, string) {
	username, password := w.Username, w.Password
	if w.user != nil {
		username = w.user.Text()
	}
	if w.password != nil {
		password = w.password.Text()
	}
	return username, password
}

func loginWindowLayoutEqual(a, b LoginWindowDrawOptions) bool {
	return a.W == b.W &&
		a.H == b.H &&
		a.FooterH == b.FooterH &&
		a.FormTopPad == b.FormTopPad &&
		a.FieldGap == b.FieldGap &&
		a.FieldLeft == b.FieldLeft &&
		a.FieldRightPad == b.FieldRightPad &&
		a.FieldH == b.FieldH
}
