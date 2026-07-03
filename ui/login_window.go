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
	w.opts = opts
	w.rebuild()
}

func (w *LoginWindow) Draw(screen *render.Image) {
	if w == nil || screen == nil {
		return
	}
	if w.root == nil {
		w.rebuild()
	}
	renderer := offscreen.NewRenderer(w.opts.W, w.opts.H, offscreen.WithTheme(rotheme.Default.AsTheme()))
	renderer.Render(w.root)
	img := renderer.Image()
	if img == nil {
		return
	}
	var drawOpts render.DrawImageOptions
	drawOpts.GeoM.Translate(float64(w.opts.X), float64(w.opts.Y))
	screen.DrawImage(render.NewImageFromImage(img), &drawOpts)
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
	submit := func() {
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
		func(string) { submit() },
	)
	w.user.SetFocused(true)
	w.password = rotheme.TextField(
		w.Password,
		textfield.TypePassword,
		func(v string) {
			w.Password = v
		},
		func(string) { submit() },
	)
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
					primitives.Box(w.user).
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
					primitives.Box(w.password).
						Width(fieldW).
						Height(fieldH),
				).
					CrossAlign(primitives.CrossAxisCenter).
					Gap(12),
			).
				PaddingTop(float32(w.opts.FormTopPad)).
				PaddingLeft(24).
				PaddingRight(float32(w.opts.FieldRightPad)).
				Gap(float32(w.opts.FieldGap)).
				Background(rotheme.Default.Colors.WindowBody),
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
