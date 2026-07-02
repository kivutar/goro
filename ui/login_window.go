package ui

import (
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/offscreen"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const loginWindowButtonHeight = 22

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
	uiApp     client.UIApp
	root      widget.Widget
	user      *textfield.Widget
	password  *textfield.Widget
}

func NewLoginWindow(opts LoginWindowDrawOptions, callbacks LoginWindowCallbacks) *LoginWindow {
	w := &LoginWindow{
		Username:  opts.Username,
		Password:  opts.Password,
		opts:      opts,
		callbacks: callbacks,
	}
	w.rebuild()
	return w
}

func (w *LoginWindow) SetUIApp(uiApp client.UIApp) {
	if w == nil || w.uiApp == uiApp {
		return
	}
	w.uiApp = uiApp
	w.setAppRoot()
}

func (w *LoginWindow) SetOptions(opts LoginWindowDrawOptions) {
	if w == nil {
		return
	}
	if w.opts == opts {
		return
	}
	if w.opts.W != opts.W || w.opts.H != opts.H || w.opts.FieldLeft != opts.FieldLeft || w.opts.FieldRightPad != opts.FieldRightPad {
		w.opts = opts
		w.rebuild()
		return
	}
	w.opts = opts
	w.setAppRoot()
}

func (w *LoginWindow) Draw(screen *render.Image) {
	if w == nil || screen == nil {
		return
	}
	root := w.drawTree()
	renderer := offscreen.NewRenderer(w.opts.W, w.opts.H, offscreen.WithTheme(rotheme.Default.AsTheme()))
	renderer.Render(root)
	img := renderer.Image()
	if img == nil {
		return
	}
	var drawOpts render.DrawImageOptions
	drawOpts.GeoM.Translate(float64(w.opts.X), float64(w.opts.Y))
	screen.DrawImage(render.NewImageFromImage(img), &drawOpts)
}

func (w *LoginWindow) drawTree() widget.Widget {
	user := rotheme.TextField(w.Username, textfield.TypeText, nil, nil)
	user.SetFocused(w.user != nil && w.user.IsFocused())
	password := rotheme.TextField(w.Password, textfield.TypePassword, nil, nil)
	password.SetFocused(w.password != nil && w.password.IsFocused())
	return loginWindowTreeWithFields(w.opts, user, password, nil)
}

func (w *LoginWindow) rebuild() {
	w.user = nil
	w.password = nil
	w.root = w.widgetTree()
	w.setAppRoot()
}

func (w *LoginWindow) setAppRoot() {
	if w.uiApp == nil || w.root == nil {
		return
	}
	w.uiApp.SetRoot(
		primitives.Box(w.root).
			PaddingLeft(float32(w.opts.X)).
			PaddingTop(float32(w.opts.Y)).
			Width(float32(w.opts.X + w.opts.W)).
			Height(float32(w.opts.Y + w.opts.H)),
	)
}

func (w *LoginWindow) widgetTree() widget.Widget {
	submit := func(string) {
		if w.callbacks.OnSubmit != nil {
			w.callbacks.OnSubmit()
		}
	}
	w.user = rotheme.TextField(
		w.Username,
		textfield.TypeText,
		func(v string) {
			w.Username = v
		},
		submit,
	)
	w.user.SetFocused(true)
	w.password = rotheme.TextField(
		w.Password,
		textfield.TypePassword,
		func(v string) {
			w.Password = v
		},
		submit,
	)
	return loginWindowTreeWithFields(w.opts, w.user, w.password, func() {
		if w.callbacks.OnSubmit != nil {
			w.callbacks.OnSubmit()
		}
	})
}

func loginWindowTreeWithFields(opts LoginWindowDrawOptions, userField, passField widget.Widget, onLogin func()) widget.Widget {
	local := opts
	local.X = 0
	local.Y = 0
	labelW := float32(local.FieldLeft - 36)
	fieldW := float32(local.W - local.FieldLeft - local.FieldRightPad)
	fieldH := float32(local.FieldH)
	buttonW := float32(ButtonLabelWidth("Login"))
	labelText := func(label string) widget.Widget {
		return rotheme.Text(label).
			LineHeight(fieldH / rotheme.Default.Typography.TextSize)
	}
	formRow := func(label string, field widget.Widget) widget.Widget {
		return primitives.HBox(
			primitives.Box(labelText(label)).
				Width(labelW).
				Height(fieldH),
			primitives.Box(field).
				Width(fieldW).
				Height(fieldH),
		).
			CrossAlign(primitives.CrossAxisCenter).
			Gap(12)
	}
	loginButton := primitives.Box(
		rotheme.Button("Login", onLogin).
			MinWidth(buttonW),
	).
		Width(buttonW).
		Height(loginWindowButtonHeight)
	footer := primitives.HBox(
		primitives.Expanded(primitives.Box()),
		loginButton,
	)
	return Window(
		Title("Login"),
		CloseButton(false),
		Size(float32(local.W), float32(local.H)),
		FooterHeight(float32(local.FooterH)),
		FooterPadding(float32(local.FieldRightPad)),
		Content(
			primitives.Box(
				formRow("Account", userField),
				formRow("Password", passField),
			).
				PaddingTop(float32(local.FormTopPad)).
				PaddingLeft(24).
				PaddingRight(float32(local.FieldRightPad)).
				Gap(float32(local.FieldGap)).
				Background(rotheme.Default.Colors.WindowBody),
		),
		Footer(footer),
	)
}

func LoginWindowButtonRect(opts LoginWindowDrawOptions) (int, int, int, int) {
	fieldX, _, fieldW, _ := LoginWindowFieldRect(opts, 0)
	buttonW := ButtonLabelWidth("Login")
	_, footerY, _, footerH := LoginWindowFooterRect(opts)
	buttonH := loginWindowButtonHeight
	return fieldX + fieldW - buttonW, footerY + (footerH-buttonH)/2, buttonW, buttonH
}

func LoginWindowFooterRect(opts LoginWindowDrawOptions) (int, int, int, int) {
	return opts.X, opts.Y + opts.H - opts.FooterH, opts.W, opts.FooterH
}

func LoginWindowFieldRect(opts LoginWindowDrawOptions, row int) (int, int, int, int) {
	fieldX := opts.X + opts.FieldLeft
	fieldY := opts.Y + ROWindowTitleHeight + opts.FormTopPad + row*(opts.FieldH+opts.FieldGap)
	fieldW := opts.W - opts.FieldLeft - opts.FieldRightPad
	return fieldX, fieldY, fieldW, opts.FieldH
}

func LoginWindowLabelX(fieldX int, label string) int {
	return fieldX - 12 - len([]rune(label))*7
}

func LoginWindowLabelY(fieldY, fieldH int) int {
	return fieldY + maxInt(0, (fieldH-14)/2)
}
