package ui

import (
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/offscreen"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

type LoginWindowDrawOptions struct {
	X, Y, W, H    int
	TitleH        int
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
	OnUsernameChange func(string)
	OnPasswordChange func(string)
	OnSubmit         func()
}

type LoginWindow struct {
	opts      LoginWindowDrawOptions
	callbacks LoginWindowCallbacks
	ctx       *widget.ContextImpl
	root      widget.Widget
	user      *roTextFieldWidget
	password  *roTextFieldWidget
	lastMouse geometry.Point
	hasMouse  bool
}

func NewLoginWindow(opts LoginWindowDrawOptions, callbacks LoginWindowCallbacks) *LoginWindow {
	w := &LoginWindow{
		opts:      opts,
		callbacks: callbacks,
		ctx:       widget.NewContext(),
	}
	w.ctx.SetThemeProvider(rotheme.Default.AsTheme())
	w.rebuild()
	return w
}

func (w *LoginWindow) SetOptions(opts LoginWindowDrawOptions) {
	if w == nil {
		return
	}
	if w.opts.W != opts.W || w.opts.H != opts.H || w.opts.FieldLeft != opts.FieldLeft || w.opts.FieldRightPad != opts.FieldRightPad {
		w.opts = opts
		w.rebuild()
		return
	}
	w.opts = opts
}

func (w *LoginWindow) Update(state *input.State) {
	if w == nil || state == nil || w.root == nil {
		return
	}
	w.layout()
	local := geometry.Pt(float32(state.MouseX-w.opts.X), float32(state.MouseY-w.opts.Y))
	global := geometry.Pt(float32(state.MouseX), float32(state.MouseY))
	if !w.hasMouse || w.lastMouse != local {
		if w.hasMouse {
			w.root.Event(w.ctx, event.NewMouseEvent(event.MouseLeave, event.ButtonNone, 0, w.lastMouse, w.lastMouse, event.ModNone))
		}
		w.root.Event(w.ctx, event.NewMouseEvent(event.MouseEnter, event.ButtonNone, 0, local, global, event.ModNone))
		w.lastMouse = local
		w.hasMouse = true
	}
	if state.MouseJustPressed(input.MouseButtonLeft) {
		w.root.Event(w.ctx, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, local, global, event.ModNone))
	}
	if state.MouseJustReleased(input.MouseButtonLeft) {
		w.root.Event(w.ctx, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, local, global, event.ModNone))
	}
	if state.JustPressed(input.KeyTab) {
		w.focusNext()
	}
	if state.JustPressed(input.KeyBackspace) {
		w.root.Event(w.ctx, event.NewKeyEvent(event.KeyPress, event.KeyBackspace, 0, event.ModNone))
	}
	if state.JustPressed(input.KeyEnter) {
		w.root.Event(w.ctx, event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	}
	for _, r := range state.TextInput() {
		w.root.Event(w.ctx, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, event.ModNone))
	}
}

func (w *LoginWindow) Draw(screen *render.Image) {
	if w == nil || screen == nil || w.root == nil {
		return
	}
	w.layout()
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
	w.layout()
	w.ctx.RequestFocus(w.user)
}

func (w *LoginWindow) layout() {
	w.ctx.SetWindowSize(geometry.Sz(float32(w.opts.W), float32(w.opts.H)))
	w.root.Layout(w.ctx, geometry.Loose(geometry.Sz(float32(w.opts.W), float32(w.opts.H))))
}

func (w *LoginWindow) focusNext() {
	if w.ctx.IsFocused(w.user) {
		w.ctx.RequestFocus(w.password)
		return
	}
	w.ctx.RequestFocus(w.user)
}

func (w *LoginWindow) widgetTree() widget.Widget {
	submit := func(string) {
		if w.callbacks.OnSubmit != nil {
			w.callbacks.OnSubmit()
		}
	}
	w.user = roTextFieldAction(w.opts.Username, textfield.TypeText, true, func(v string) {
		if w.callbacks.OnUsernameChange != nil {
			w.callbacks.OnUsernameChange(v)
		}
	}, submit)
	w.password = roTextFieldAction(w.opts.Password, textfield.TypePassword, false, func(v string) {
		if w.callbacks.OnPasswordChange != nil {
			w.callbacks.OnPasswordChange(v)
		}
	}, submit)
	return loginWindowTreeWithFields(w.opts, w.user, w.password, func() {
		if w.callbacks.OnSubmit != nil {
			w.callbacks.OnSubmit()
		}
	})
}

func loginWindowTreeWithFields(opts LoginWindowDrawOptions, userField, passField *roTextFieldWidget, onLogin func()) widget.Widget {
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
			field,
		).
			CrossAlign(primitives.CrossAxisCenter).
			Gap(12)
	}
	loginButton := roButtonAction("Login", onLogin).
		SetBackground(widget.RGBA8(ButtonColor.R, ButtonColor.G, ButtonColor.B, ButtonColor.A)).
		MinWidth(buttonW)
	footer := primitives.HBox(
		primitives.Expanded(primitives.Box()),
		loginButton,
	)
	return Window(
		Title("Login"),
		CloseButton(false),
		Size(float32(local.W), float32(local.H)),
		TitleHeight(float32(local.TitleH)),
		FooterHeight(float32(local.FooterH)),
		FooterPadding(float32(local.FieldRightPad)),
		Content(
			primitives.Box(
				formRow("Account", userField.Width(fieldW)),
				formRow("Password", passField.Width(fieldW)),
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
	buttonH := roUIButtonHeight
	return fieldX + fieldW - buttonW, footerY + (footerH-buttonH)/2, buttonW, buttonH
}

func LoginWindowFooterRect(opts LoginWindowDrawOptions) (int, int, int, int) {
	return opts.X, opts.Y + opts.H - opts.FooterH, opts.W, opts.FooterH
}

func LoginWindowFieldRect(opts LoginWindowDrawOptions, row int) (int, int, int, int) {
	fieldX := opts.X + opts.FieldLeft
	fieldY := opts.Y + opts.TitleH + opts.FormTopPad + row*(opts.FieldH+opts.FieldGap)
	fieldW := opts.W - opts.FieldLeft - opts.FieldRightPad
	return fieldX, fieldY, fieldW, opts.FieldH
}

func LoginWindowLabelX(fieldX int, label string) int {
	return fieldX - 12 - len([]rune(label))*7
}

func LoginWindowLabelY(fieldY, fieldH int) int {
	return fieldY + maxInt(0, (fieldH-14)/2)
}
