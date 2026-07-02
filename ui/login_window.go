package ui

import (
	"strings"

	"github.com/gogpu/ui/core/button"
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

	userX, userY, userW, userH := LoginWindowFieldRect(opts, 0)
	passX, passY, passW, passH := LoginWindowFieldRect(opts, 1)
	DrawTextInput(screen, userX, userY, userW, userH, opts.Username, opts.FocusUser)
	DrawTextInput(screen, passX, passY, passW, passH, strings.Repeat("*", len([]rune(opts.Password))), opts.FocusPassword)
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
	inputFrame := func() widget.Widget {
		return primitives.Box().
			Width(fieldW).
			Height(fieldH).
			Background(rotheme.Default.Colors.WindowBody).
			BorderStyle(1, rotheme.Default.Colors.WindowBorder)
	}
	formRow := func(label string) widget.Widget {
		return primitives.HBox(
			primitives.Box(rotheme.Text(label)).
				Width(labelW),
			inputFrame(),
		).
			CrossAlign(primitives.CrossAxisCenter).
			Gap(12)
	}
	buttonColor := ButtonColor
	if local.LoginButtonHover {
		buttonColor = ButtonHoverColor
	}
	buttonOptions := []button.Option{
		button.TextOpt("Login"),
		button.SizeOpt(button.Small),
		button.VariantOpt(button.Filled),
		button.BackgroundOpt(widget.RGBA8(buttonColor.R, buttonColor.G, buttonColor.B, buttonColor.A)),
	}
	footer := primitives.HBox(
		primitives.Expanded(primitives.Box()),
		primitives.Box(
			button.New(buttonOptions...),
		).Width(buttonW),
	)
	return Window(
		Title("Login"),
		CloseButton(false),
		Size(float32(local.W), float32(local.H)),
		TitleHeight(float32(local.TitleH)),
		FooterPadding(5),
		Content(
			primitives.Box(
				formRow("Account"),
				formRow("Password"),
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
	buttonH := 24
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
