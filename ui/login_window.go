package ui

import (
	"github.com/gogpu/ui/core/checkbox"
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
	KeepID   bool

	Window
	layout    loginWindowLayout
	callbacks LoginWindowCallbacks
	user      *textfield.Widget
	password  *textfield.Widget
	keep      *checkbox.Widget
}

const (
	loginWindowFormTopPad   = 18
	loginWindowFormLeftPad  = 16
	loginWindowFormRightPad = 8
	loginWindowFieldGap     = 11
	loginWindowLabelW       = 56
	loginWindowLabelGap     = 12
	loginWindowFieldH       = 22
	loginWindowKeepW        = 64
	loginWindowKeepGap      = 12
)

func NewLoginWindow(ctx client.Context, username, password string, keepID bool, callbacks LoginWindowCallbacks) *LoginWindow {
	layout := loginWindowLayoutForContext(ctx)
	w := &LoginWindow{
		Username:  username,
		Password:  password,
		KeepID:    keepID,
		layout:    layout,
		callbacks: callbacks,
	}
	w.Window = NewWindow(layout.W, layout.H)
	w.OpenAt(layout.X, layout.Y, w.widgetTree())
	w.restoreFocus(ctx)
	return w
}

func (w *LoginWindow) SetContext(ctx client.Context) {
	if w == nil {
		return
	}
	layout := loginWindowLayoutForContext(ctx)
	sameLayout := loginWindowLayoutEqual(w.layout, layout)
	w.layout = layout
	w.SetAutoPosition(layout.X, layout.Y)
	w.SetSize(layout.W, layout.H)
	if sameLayout {
		return
	}
	w.SetContent(w.widgetTree())
	w.restoreFocus(ctx)
}

func (w *LoginWindow) Update(ctx client.Context) bool {
	if w == nil {
		return false
	}
	return w.Window.Update(ctx)
}

func (w *LoginWindow) restoreFocus(ctx client.Context) {
	if wc := windowWidgetContext(ctx); wc != nil {
		// Register the initial/rebuilt field with the focus manager too, so
		// Tab advances from it instead of selecting it a second time.
		if w.user.IsFocused() {
			wc.RequestFocus(w.user)
		} else if w.password.IsFocused() {
			wc.RequestFocus(w.password)
		} else if w.keep.IsFocused() {
			wc.RequestFocus(w.keep)
		}
	}
}

func (w *LoginWindow) widgetTree() widget.Widget {
	submit := func() {
		if w.callbacks.OnSubmit != nil {
			w.callbacks.OnSubmit()
		}
	}
	userFocused, passwordFocused := w.fieldFocus()
	keepFocused := w.keep != nil && w.keep.IsFocused()
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
	w.keep = rotheme.Checkbox(
		checkbox.LabelOpt("Keep"),
		checkbox.Checked(w.KeepID),
		checkbox.OnToggle(func(keep bool) { w.KeepID = keep }),
	)
	w.keep.SetFocused(keepFocused)
	labelW := float32(loginWindowLabelW)
	fieldW := float32(w.layout.W - loginWindowFormLeftPad - loginWindowFormRightPad - loginWindowLabelW - loginWindowLabelGap - loginWindowKeepW - loginWindowKeepGap)
	fieldH := float32(loginWindowFieldH)
	return Win(
		Title("Login"),
		CloseButton(false),
		Size(float32(w.layout.W), float32(w.layout.H)),
		Content(
			primitives.HBox(
				// Keep the text fields together so Tab still moves from Account to Password.
				primitives.Box(
					primitives.HBox(
						primitives.Box(
							rotheme.Label("Account").
								Align(widget.TextAlignRight).
								LineHeight(fieldH/rotheme.Default.Typography.TextSize),
						).
							CrossAlign(primitives.CrossAxisStretch).
							Width(labelW).
							Height(fieldH),
						primitives.Box(user).
							Width(fieldW).
							Height(fieldH),
					).
						CrossAlign(primitives.CrossAxisCenter).
						Gap(loginWindowLabelGap),
					primitives.HBox(
						primitives.Box(
							rotheme.Label("Password").
								Align(widget.TextAlignRight).
								LineHeight(fieldH/rotheme.Default.Typography.TextSize),
						).
							CrossAlign(primitives.CrossAxisStretch).
							Width(labelW).
							Height(fieldH),
						primitives.Box(password).
							Width(fieldW).
							Height(fieldH),
					).
						CrossAlign(primitives.CrossAxisCenter).
						Gap(loginWindowLabelGap),
				).Gap(loginWindowFieldGap),
				primitives.Box(w.keep).Width(loginWindowKeepW).Height(fieldH),
			).
				PaddingTop(loginWindowFormTopPad).
				PaddingLeft(loginWindowFormLeftPad).
				PaddingRight(loginWindowFormRightPad).
				CrossAlign(primitives.CrossAxisStart).
				Gap(loginWindowKeepGap),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Login", submit),
		),
	)
}

func (w *LoginWindow) fieldFocus() (bool, bool) {
	if w.user == nil && w.password == nil {
		return w.Username == "", w.Username != ""
	}
	userFocused := w.user != nil && w.user.IsFocused()
	passwordFocused := w.password != nil && w.password.IsFocused()
	if !userFocused && !passwordFocused && (w.keep == nil || !w.keep.IsFocused()) {
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
	return 304, ROWindowTitleHeight + loginWindowFormTopPad + loginWindowFieldH*2 + loginWindowFieldGap + 16 + ROWindowFooterHeight
}
