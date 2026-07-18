package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

type WindowOption func(*windowConfig)

type windowConfig struct {
	title       string
	closeButton bool
	content     widget.Widget
	footer      widget.Widget
	onClose     func()
	width       float32
	height      float32
	titleBar    bool
	radius      float32
	background  *widget.Color
}

const (
	ROWindowTitleHeight   = 28
	ROWindowFooterHeight  = 42
	ROWindowFooterPadding = 10
	ROWindowFooterGap     = 8
)

func Win(options ...WindowOption) widget.Widget {
	cfg := windowConfig{
		width:    300,
		height:   240,
		titleBar: true,
		radius:   WindowRadius,
	}
	for _, option := range options {
		option(&cfg)
	}

	children := make([]widget.Widget, 0, 3)
	if cfg.titleBar {
		titleContent := primitives.HBox(
			rotheme.Title(cfg.title),
			primitives.Expanded(primitives.Box()),
			windowCloseButton(cfg.closeButton, cfg.onClose),
		).
			CrossAlign(primitives.CrossAxisCenter).
			PaddingLeft(12).
			PaddingRight(7).
			Height(ROWindowTitleHeight)
		children = append(children, roTitleBar(titleContent))
	}
	if cfg.content != nil {
		children = append(children, primitives.Expanded(cfg.content))
	}
	if cfg.footer != nil {
		footerBody := primitives.Box(
			primitives.Expanded(primitives.Box()),
			cfg.footer,
			primitives.Expanded(primitives.Box()),
		).
			CrossAlign(primitives.CrossAxisStretch).
			PaddingXY(ROWindowFooterPadding, 0).
			Height(ROWindowFooterHeight - 1).
			Background(rotheme.Default.Colors.WindowFooter)
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
		).
			CrossAlign(primitives.CrossAxisStretch)
		children = append(children,
			footer,
		)
	}

	background := rotheme.Default.Colors.WindowBody
	if cfg.background != nil {
		background = *cfg.background
	}
	return primitives.Box(children...).
		CrossAlign(primitives.CrossAxisStretch).
		Width(cfg.width).
		Height(cfg.height).
		Background(background).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder).
		Rounded(cfg.radius)
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

func Footer(children ...widget.Widget) WindowOption {
	return func(cfg *windowConfig) {
		if len(children) == 0 {
			cfg.footer = primitives.Box()
			return
		}
		cfg.footer = primitives.HBox(children...).
			Gap(ROWindowFooterGap).
			CrossAlign(primitives.CrossAxisCenter)
	}
}

func footerLabel(text string) widget.Widget {
	return primitives.Box(
		rotheme.Text(text),
	).
		Height(rotheme.Default.Typography.TextSize + rotheme.ButtonPaddingY*2).
		PaddingTop(rotheme.ButtonPaddingY)
}

func Size(width, height float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.width = width
		cfg.height = height
	}
}

func TitleBar(enabled bool) WindowOption {
	return func(cfg *windowConfig) {
		cfg.titleBar = enabled
	}
}

func Radius(radius float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.radius = radius
	}
}

func Background(color widget.Color) WindowOption {
	return func(cfg *windowConfig) {
		cfg.background = &color
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

type Window struct {
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
	background  *widget.Color
	fullRedraw  bool
	CloseOnEsc  bool
	ctx         client.Context
}

func (w *Window) EnsureWindow(width, height int) bool {
	if w.width != 0 {
		return false
	}
	*w = NewWindow(width, height)
	return true
}

func (w *Window) Close() {
	w.setOpacity(1)
	w.open = false
	w.dragging = false
	w.content = nil
	w.placed = nil
	w.Publish(w.ctx)
}

const grabbedWindowOpacity = 0.95

func NewWindow(width, height int) Window {
	return Window{
		width:       width,
		height:      height,
		titleHeight: ROWindowTitleHeight,
		opacity:     1,
		CloseOnEsc:  true,
	}
}

func (w *Window) Open(ctx client.Context, content widget.Widget) {
	w.ctx = ctx
	w.open = true
	w.ensurePosition(ctx)
	w.SetContent(content)
}

func (w *Window) OpenAt(x, y int, content widget.Widget) {
	w.open = true
	w.x = x
	w.y = y
	w.positioned = true
	w.SetContent(content)
}

func (w *Window) IsOpen() bool {
	return w.open
}

func (w *Window) SetContent(content widget.Widget) {
	w.content = content
	w.placed = nil
	if w.dragging {
		w.setOpacity(grabbedWindowOpacity)
	} else {
		w.setOpacity(1)
	}
}

func (w *Window) Publish(ctx client.Context) {
	if w == nil || ctx.UIManager == nil {
		return
	}
	w.ctx = ctx
	if !w.open || w.content == nil {
		w.Unpublish(ctx)
		return
	}
	root := w.Widget()
	if root == nil {
		return
	}
	if root == w.published {
		return
	}
	if w.fullRedraw {
		markNeedsRedraw(root)
	}
	w.Unpublish(ctx)
	w.published = root
	ctx.UIManager.AddOverlay(root)
}

func (w *Window) Unpublish(ctx client.Context) {
	if w == nil || ctx.UIManager == nil || w.published == nil {
		return
	}
	ctx.UIManager.RemoveOverlay(w.published)
	w.published = nil
}

func (w *Window) SetSize(width, height int) {
	if w.width == width && w.height == height {
		return
	}
	w.width = width
	w.height = height
	w.placed = nil
}

func (w *Window) SetBackground(color widget.Color) {
	w.background = &color
	w.setOpacity(w.opacity)
}

func (w *Window) SetFullRedraw(enabled bool) {
	w.fullRedraw = enabled
}

func (w *Window) SetAutoPosition(x, y int) bool {
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

func (w *Window) Update(ctx client.Context) bool {
	w.ctx = ctx
	if !w.open || ctx.Input == nil {
		return false
	}
	w.ensurePosition(ctx)
	screenW, screenH := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(input.MouseButtonLeft) {
			w.x = clampWindowInt(ctx.Input.MouseX-w.dragDX, 8, maxInt(8, screenW-w.width-8))
			w.y = clampWindowInt(ctx.Input.MouseY-w.dragDY, 8, maxInt(8, screenH-w.height-8))
			w.placed = nil
			return true
		}
		w.dragging = false
		w.setOpacity(1)
		return true
	}
	if w.CloseOnEsc && ctx.Input.JustPressed(input.KeyEscape) {
		w.Close()
		return true
	}
	inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, w.width, w.height)
	if !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
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

func (w *Window) ensurePosition(ctx client.Context) {
	if w.positioned {
		return
	}
	screenW, screenH := ctx.ScreenSize()
	w.x = maxInt(8, (screenW-w.width)/2)
	w.y = maxInt(8, (screenH-w.height)/2)
	w.positioned = true
	w.placed = nil
}

func (w *Window) setOpacity(opacity float32) {
	changed := w.opacity != opacity
	w.opacity = opacity
	if w.titleHeight <= 0 {
		return
	}
	if box, ok := w.content.(*primitives.BoxWidget); ok {
		background := windowBodyColor(opacity)
		if w.background != nil {
			background = *w.background
			background.A *= opacity
		}
		box.Background(background)
		box.SetNeedsRedraw(true)
	}
	if changed {
		w.placed = nil
	} else if box, ok := w.placed.(*primitives.BoxWidget); ok {
		box.SetNeedsRedraw(true)
	}
}

func (w *Window) Widget() widget.Widget {
	if w == nil || !w.open || w.content == nil {
		return nil
	}
	if w.placed == nil {
		w.placed = positionedWidget(w.content, w.x, w.y, w.width, w.height)
	}
	return w.placed
}

func markNeedsRedraw(root widget.Widget) {
	type redrawSetter interface {
		SetNeedsRedraw(bool)
	}
	if redraw, ok := root.(redrawSetter); ok {
		redraw.SetNeedsRedraw(true)
	}
}

func positionedWidget(content widget.Widget, x, y, width, height int) widget.Widget {
	w := &positionedOverlay{
		child:  content,
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

type positionedOverlay struct {
	widget.WidgetBase
	child         widget.Widget
	x, y          int
	width, height int
}

func (w *positionedOverlay) Layout(ctx widget.Context, _ geometry.Constraints) geometry.Size {
	size := geometry.Sz(float32(w.width), float32(w.height))
	w.SetBounds(geometry.NewRect(float32(w.x), float32(w.y), size.Width, size.Height))
	if w.child != nil {
		w.child.Layout(ctx, geometry.Tight(size))
		if setter, ok := w.child.(interface{ SetBounds(geometry.Rect) }); ok {
			setter.SetBounds(geometry.FromPointSize(geometry.Point{}, size))
		}
	}
	return size
}

func (w *positionedOverlay) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() || w.child == nil {
		return
	}
	canvas.PushTransform(w.Bounds().Min)
	widget.StampScreenOrigin(w.child, canvas)
	widget.DrawChild(w.child, ctx, canvas)
	canvas.PopTransform()
}

func (w *positionedOverlay) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() || w.child == nil {
		return false
	}
	switch ev := e.(type) {
	case *event.MouseEvent:
		if !w.Bounds().Contains(ev.Position) {
			return false
		}
		local := *ev
		local.Position = ev.Position.Sub(w.Bounds().Min)
		w.child.Event(ctx, &local)
		return true
	case *event.WheelEvent:
		if !w.Bounds().Contains(ev.Position) {
			return false
		}
		local := *ev
		local.Position = ev.Position.Sub(w.Bounds().Min)
		w.child.Event(ctx, &local)
		return true
	default:
		return w.child.Event(ctx, e)
	}
}

func (w *positionedOverlay) Children() []widget.Widget {
	if w.child == nil {
		return nil
	}
	return []widget.Widget{w.child}
}

func centeredWindowRect(ctx client.Context, width, height int) (int, int, int, int) {
	screenW, screenH := ctx.ScreenSize()
	x := (screenW - width) / 2
	y := (screenH - height) / 2
	if x < 8 {
		x = 8
	}
	if y < 8 {
		y = 8
	}
	return x, y, width, height
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
