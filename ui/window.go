package ui

import (
	"github.com/gogpu/ui/offscreen"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

type WindowOption func(*windowConfig)

type windowConfig struct {
	title         string
	closeButton   bool
	content       widget.Widget
	footer        widget.Widget
	onClose       func()
	width         float32
	height        float32
	footerPadding float32
	footerHeight  float32
}

const ROWindowTitleHeight = 28

func Window(options ...WindowOption) widget.Widget {
	cfg := windowConfig{
		width:         300,
		height:        240,
		footerPadding: 8,
	}
	for _, option := range options {
		option(&cfg)
	}

	titleContent := primitives.HBox(
		rotheme.Title(cfg.title),
		primitives.Expanded(primitives.Box()),
		windowCloseButton(cfg.closeButton, cfg.onClose),
	).
		CrossAlign(primitives.CrossAxisCenter).
		PaddingXY(14, 0).
		Height(ROWindowTitleHeight)

	children := []widget.Widget{
		roTitleBar(titleContent),
	}
	if cfg.content != nil {
		children = append(children, primitives.Expanded(cfg.content))
	}
	if cfg.footer != nil {
		footerBody := primitives.Box(cfg.footer).
			Padding(cfg.footerPadding).
			Background(rotheme.Default.Colors.WindowFooter)
		if cfg.footerHeight > 0 {
			footerBody = primitives.Box(
				primitives.Expanded(primitives.Box()),
				cfg.footer,
				primitives.Expanded(primitives.Box()),
			).
				PaddingXY(cfg.footerPadding, 0).
				Height(cfg.footerHeight - 1).
				Background(rotheme.Default.Colors.WindowFooter)
		}
		footer := primitives.Box(
			primitives.HBox(
				primitives.Expanded(
					primitives.Box().
						Height(1).
						Background(rotheme.Default.Colors.FooterLine),
				),
			).
				Height(1),
			footerBody,
		)
		children = append(children,
			footer,
		)
	}

	return primitives.Box(children...).
		Width(cfg.width).
		Height(cfg.height).
		Background(rotheme.Default.Colors.WindowBody).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder).
		Rounded(WindowRadius)
}

func Title(title string) WindowOption {
	return func(cfg *windowConfig) {
		cfg.title = title
	}
}

func CloseButton(enabled bool) WindowOption {
	return func(cfg *windowConfig) {
		cfg.closeButton = enabled
	}
}

func OnClose(onClose func()) WindowOption {
	return func(cfg *windowConfig) {
		cfg.onClose = onClose
	}
}

func Content(content widget.Widget) WindowOption {
	return func(cfg *windowConfig) {
		cfg.content = content
	}
}

func Footer(footer widget.Widget) WindowOption {
	return func(cfg *windowConfig) {
		cfg.footer = footer
	}
}

func FooterPadding(padding float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.footerPadding = padding
	}
}

func FooterHeight(height float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.footerHeight = height
	}
}

func Size(width, height float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.width = width
		cfg.height = height
	}
}

func windowCloseButton(enabled bool, onClose func()) widget.Widget {
	if !enabled {
		return primitives.Box().Width(17).Height(17)
	}
	return primitives.Box(
		rotheme.Button("x", onClose).
			Width(17).
			Height(17),
	).
		Width(17).
		Height(17)
}

type WindowState struct {
	open        bool
	x           int
	y           int
	width       int
	height      int
	titleHeight int
	positioned  bool
	dragging    bool
	dragDX      int
	dragDY      int
	uiApp       client.UIApp
	root        widget.Widget
}

func NewWindowState(width, height int) WindowState {
	return WindowState{
		width:       width,
		height:      height,
		titleHeight: ROWindowTitleHeight,
	}
}

func (w *WindowState) Open(ctx client.Context, root widget.Widget) {
	w.open = true
	w.ensurePosition(ctx)
	w.SetRoot(root)
}

func (w *WindowState) Close() {
	w.open = false
	w.root = nil
	if w.uiApp != nil {
		w.uiApp.SetRoot(primitives.Box())
	}
}

func (w *WindowState) IsOpen() bool {
	return w.open
}

func (w *WindowState) SetRoot(root widget.Widget) {
	w.root = root
	w.setAppRoot()
}

func (w *WindowState) Update(ctx client.Context) bool {
	if !w.open || ctx.Input == nil {
		return false
	}
	w.SetUIApp(ctx.UIApp)
	w.ensurePosition(ctx)
	screenW, screenH := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(render.MouseButtonLeft) {
			w.x = clampWindowInt(ctx.Input.MouseX-w.dragDX, 8, maxInt(8, screenW-w.width-8))
			w.y = clampWindowInt(ctx.Input.MouseY-w.dragDY, 8, maxInt(8, screenH-w.height-8))
			w.setAppRoot()
			return true
		}
		w.dragging = false
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.Close()
		return true
	}
	inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, w.width, w.height)
	if !ctx.Input.MouseJustPressed(render.MouseButtonLeft) {
		return inside
	}
	if !inside {
		return false
	}
	if pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, w.width, w.titleHeight) {
		w.dragging = true
		w.dragDX = ctx.Input.MouseX - w.x
		w.dragDY = ctx.Input.MouseY - w.y
		return true
	}
	return true
}

func (w *WindowState) Draw(screen *render.Image) {
	if !w.open || screen == nil || w.root == nil {
		return
	}
	r := offscreen.NewRenderer(w.width, w.height, offscreen.WithTheme(rotheme.Default.AsTheme()))
	r.Render(w.root)
	img := r.Image()
	if img == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(w.x), float64(w.y))
	opts.Filter = render.FilterNearest
	screen.DrawImage(render.NewImageFromImage(img), &opts)
}

func (w *WindowState) SetUIApp(uiApp client.UIApp) {
	if w == nil || w.uiApp == uiApp {
		return
	}
	w.uiApp = uiApp
	w.setAppRoot()
}

func (w *WindowState) ensurePosition(ctx client.Context) {
	if w.positioned {
		return
	}
	screenW, screenH := ctx.ScreenSize()
	w.x = maxInt(8, (screenW-w.width)/2)
	w.y = maxInt(8, (screenH-w.height)/2)
	w.positioned = true
}

func (w *WindowState) setAppRoot() {
	if w.uiApp == nil || w.root == nil {
		return
	}
	w.uiApp.SetRoot(
		primitives.Box(w.root).
			PaddingLeft(float32(w.x)).
			PaddingTop(float32(w.y)).
			Width(float32(w.x + w.width)).
			Height(float32(w.y + w.height)),
	)
}

func clampWindowInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
