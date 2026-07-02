package ui

import (
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/offscreen"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

type LoginWindowDrawOptions struct {
	X, Y, W, H       int
	TitleH           int
	FooterH          int
	FormTopPad       int
	FieldGap         int
	FieldLeft        int
	FieldRightPad    int
	FieldH           int
	Username         string
	Password         string
	FocusUser        bool
	FocusPassword    bool
	LoginButtonHover bool
}

func DrawLoginWindow(screen *render.Image, opts LoginWindowDrawOptions) {
	if screen == nil {
		return
	}
	if window := renderLoginWindowTree(opts); window != nil {
		var drawOpts render.DrawImageOptions
		drawOpts.GeoM.Translate(float64(opts.X), float64(opts.Y))
		screen.DrawImage(window, &drawOpts)
	}
}

func renderLoginWindowTree(opts LoginWindowDrawOptions) *render.Image {
	renderer := offscreen.NewRenderer(opts.W, opts.H, offscreen.WithTheme(rotheme.Default.AsTheme()))
	renderer.Render(loginWindowTree(opts))
	img := renderer.Image()
	if img == nil {
		return nil
	}
	return render.NewImageFromImage(img)
}

func loginWindowTree(opts LoginWindowDrawOptions) widget.Widget {
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
	buttonColor := ButtonColor
	if local.LoginButtonHover {
		buttonColor = ButtonHoverColor
	}
	loginButton := roButton("Login").
		SetBackground(widget.RGBA8(buttonColor.R, buttonColor.G, buttonColor.B, buttonColor.A)).
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
				formRow("Account", roTextField(local.Username, textfield.TypeText, local.FocusUser).Width(fieldW)),
				formRow("Password", roTextField(local.Password, textfield.TypePassword, local.FocusPassword).Width(fieldW)),
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
