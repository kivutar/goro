package ui

import (
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/ui/rotheme"
)

type loginWindowLayout struct {
	X, Y, W, H int
}

type LoginWindowCallbacks struct {
	OnSubmit func()
}

type LoginWindow struct {
	Username string
	Password string

	layout    loginWindowLayout
	callbacks LoginWindowCallbacks
	window    WindowState
	user      *textfield.Widget
	password  *textfield.Widget
}

const (
	loginWindowFooterH       = 42
	loginWindowFormTopPad    = 18
	loginWindowFieldGap      = 11
	loginWindowFieldLeft     = 92
	loginWindowFieldRightPad = 20
	loginWindowFieldH        = 22
)

func NewLoginWindow(ctx client.Context, username, password string, callbacks LoginWindowCallbacks) *LoginWindow {
	layout := loginWindowLayoutForContext(ctx)
	w := &LoginWindow{
		Username:  username,
		Password:  password,
		layout:    layout,
		callbacks: callbacks,
		window:    NewWindowState(layout.W, layout.H),
	}
	w.window.OpenAt(layout.X, layout.Y, w.widgetTree())
	return w
}

func (w *LoginWindow) SetContext(ctx client.Context) {
	if w == nil {
		return
	}
	layout := loginWindowLayoutForContext(ctx)
	sameLayout := loginWindowLayoutEqual(w.layout, layout)
	w.layout = layout
	w.window.SetAutoPosition(layout.X, layout.Y)
	w.window.SetSize(layout.W, layout.H)
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
	labelW := float32(loginWindowFieldLeft - 36)
	fieldW := float32(w.layout.W - loginWindowFieldLeft - loginWindowFieldRightPad)
	fieldH := float32(loginWindowFieldH)
	buttonW := float32(ButtonLabelWidth("Login"))
	return Window(
		Title("Login"),
		CloseButton(false),
		Size(float32(w.layout.W), float32(w.layout.H)),
		FooterHeight(loginWindowFooterH),
		FooterPadding(loginWindowFieldRightPad),
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
				PaddingTop(loginWindowFormTopPad).
				PaddingLeft(24).
				PaddingRight(loginWindowFieldRightPad).
				Gap(loginWindowFieldGap),
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

func loginWindowLayoutEqual(a, b loginWindowLayout) bool {
	return a.W == b.W &&
		a.H == b.H &&
		a.X == b.X &&
		a.Y == b.Y
}

func loginWindowLayoutForContext(ctx client.Context) loginWindowLayout {
	width, height := ctx.ScreenSize()
	w, h := loginWindowSize()
	x := (width - w) / 2
	y := (height*2)/3 - h/2
	if y < 48 {
		y = (height - h) / 2
	}
	if x < 8 {
		x = 8
	}
	if y < 8 {
		y = 8
	}
	return loginWindowLayout{X: x, Y: y, W: w, H: h}
}

func loginWindowSize() (int, int) {
	return 304, ROWindowTitleHeight + loginWindowFormTopPad + loginWindowFieldH*2 + loginWindowFieldGap + 16 + loginWindowFooterH
}
