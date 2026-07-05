package ui

import (
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
		PaddingLeft(12).
		PaddingRight(7).
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

func windowBodyColor(opacity float32) widget.Color {
	color := rotheme.Default.Colors.WindowBody
	color.A *= opacity
	return color
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
		return primitives.Box().Width(17).Height(ROWindowTitleHeight)
	}
	return primitives.Box(
		primitives.Expanded(primitives.Box()),
		rotheme.IconButton(rotheme.IconButtonClose, onClose),
		primitives.Expanded(primitives.Box()),
	).
		Width(17).
		Height(ROWindowTitleHeight)
}

type WindowState struct {
	open        bool
	x           int
	y           int
	width       int
	height      int
	titleHeight int
	positioned  bool
	userMoved   bool
	dragging    bool
	dragDX      int
	dragDY      int
	content     widget.Widget
	placed      widget.Widget
	published   widget.Widget
	opacity     float32
	closeOnEsc  bool
}

const grabbedWindowOpacity = 0.95

func NewWindowState(width, height int) WindowState {
	return WindowState{
		width:       width,
		height:      height,
		titleHeight: ROWindowTitleHeight,
		opacity:     1,
		closeOnEsc:  true,
	}
}

func (w *WindowState) Open(ctx client.Context, content widget.Widget) {
	w.open = true
	w.ensurePosition(ctx)
	w.SetContent(content)
}

func (w *WindowState) OpenAt(x, y int, content widget.Widget) {
	w.open = true
	w.x = x
	w.y = y
	w.positioned = true
	w.SetContent(content)
}

func (w *WindowState) Close() {
	w.setOpacity(1)
	w.open = false
	w.dragging = false
	w.content = nil
	w.placed = nil
}

func (w *WindowState) IsOpen() bool {
	return w.open
}

func (w *WindowState) SetContent(content widget.Widget) {
	w.content = content
	w.placed = nil
	if w.dragging {
		w.setOpacity(grabbedWindowOpacity)
	} else {
		w.setOpacity(1)
	}
}

func (w *WindowState) Publish(ctx client.Context) {
	if w == nil || ctx.UIManager == nil {
		return
	}
	if !w.open || w.content == nil {
		w.Unpublish(ctx)
		return
	}
	root := w.Widget()
	if root == nil || root == w.published {
		return
	}
	w.Unpublish(ctx)
	w.published = root
	ctx.UIManager.AddOverlay(root)
}

func (w *WindowState) Unpublish(ctx client.Context) {
	if w == nil || ctx.UIManager == nil || w.published == nil {
		return
	}
	ctx.UIManager.RemoveOverlay(w.published)
	w.published = nil
}

func (w *WindowState) SetSize(width, height int) {
	if w.width == width && w.height == height {
		return
	}
	w.width = width
	w.height = height
	w.placed = nil
}

func (w *WindowState) SetAutoPosition(x, y int) bool {
	if w.userMoved {
		return false
	}
	if w.x == x && w.y == y && w.positioned {
		return false
	}
	w.x = x
	w.y = y
	w.positioned = true
	w.placed = nil
	return true
}

func (w *WindowState) SetCloseOnEscape(enabled bool) {
	w.closeOnEsc = enabled
}

func (w *WindowState) Update(ctx client.Context) bool {
	if !w.open || ctx.Input == nil {
		return false
	}
	w.ensurePosition(ctx)
	screenW, screenH := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(render.MouseButtonLeft) {
			w.x = clampWindowInt(ctx.Input.MouseX-w.dragDX, 8, maxInt(8, screenW-w.width-8))
			w.y = clampWindowInt(ctx.Input.MouseY-w.dragDY, 8, maxInt(8, screenH-w.height-8))
			w.placed = nil
			return true
		}
		w.dragging = false
		w.setOpacity(1)
		return true
	}
	if w.closeOnEsc && ctx.Input.JustPressed(render.KeyEscape) {
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
		w.userMoved = true
		w.dragDX = ctx.Input.MouseX - w.x
		w.dragDY = ctx.Input.MouseY - w.y
		w.setOpacity(grabbedWindowOpacity)
		return true
	}
	return true
}

func (w *WindowState) ensurePosition(ctx client.Context) {
	if w.positioned {
		return
	}
	screenW, screenH := ctx.ScreenSize()
	w.x = maxInt(8, (screenW-w.width)/2)
	w.y = maxInt(8, (screenH-w.height)/2)
	w.positioned = true
	w.placed = nil
}

func (w *WindowState) setOpacity(opacity float32) {
	changed := w.opacity != opacity
	w.opacity = opacity
	if box, ok := w.content.(*primitives.BoxWidget); ok {
		box.Background(windowBodyColor(opacity))
		box.SetNeedsRedraw(true)
	}
	if changed {
		w.placed = nil
	} else if box, ok := w.placed.(*primitives.BoxWidget); ok {
		box.SetNeedsRedraw(true)
	}
}

func (w *WindowState) Widget() widget.Widget {
	if w == nil || !w.open || w.content == nil {
		return nil
	}
	if w.placed == nil {
		w.placed = primitives.Box(w.content).
			PaddingLeft(float32(w.x)).
			PaddingTop(float32(w.y)).
			Width(float32(w.x + w.width)).
			Height(float32(w.y + w.height))
	}
	return w.placed
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
