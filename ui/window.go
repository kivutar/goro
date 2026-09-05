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
	title        string
	closeButton  bool
	titleButtons []windowTitleButton
	content      widget.Widget
	footer       widget.Widget
	onClose      func()
	width        float32
	height       float32
	titleBar     bool
	radius       float32
	background   *widget.Color
}

type windowTitleButton struct {
	kind    rotheme.IconButtonKind
	onClick func()
}

const (
	ROWindowTitleHeight   = 28
	ROWindowFooterHeight  = 42
	ROWindowFooterPadding = 10
	ROWindowFooterGap     = 8
	windowScreenMargin    = 8
	windowDirtyPadding    = 2
	windowTitleButtonSize = 17
	windowTitleButtonGap  = 2
	windowTitleButtonPadR = 7
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
			windowTitleButtons(cfg.titleButtons, cfg.closeButton, cfg.onClose),
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

func TitleButton(kind rotheme.IconButtonKind, onClick func()) WindowOption {
	return func(cfg *windowConfig) {
		cfg.titleButtons = append(cfg.titleButtons, windowTitleButton{
			kind:    kind,
			onClick: onClick,
		})
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

func windowTitleButtons(buttons []windowTitleButton, closeButton bool, onClose func()) widget.Widget {
	children := make([]widget.Widget, 0, len(buttons)+1)
	for _, button := range buttons {
		children = append(children, windowTitleIconButton(button.kind, button.onClick))
	}
	children = append(children, windowCloseButton(closeButton, onClose))
	return primitives.HBox(children...).
		Gap(windowTitleButtonGap).
		CrossAlign(primitives.CrossAxisCenter)
}

func windowCloseButton(enabled bool, onClose func()) widget.Widget {
	if !enabled {
		return primitives.Box().Width(windowTitleButtonSize).Height(ROWindowTitleHeight)
	}
	return windowTitleIconButton(rotheme.IconButtonClose, onClose)
}

func windowTitleIconButton(kind rotheme.IconButtonKind, onClick func()) widget.Widget {
	return primitives.Box(
		primitives.Expanded(primitives.Box()),
		rotheme.IconButton(kind, onClick),
		primitives.Expanded(primitives.Box()),
	).
		Width(windowTitleButtonSize).
		Height(ROWindowTitleHeight)
}

type Window struct {
	open         bool
	x            int
	y            int
	width        int
	height       int
	titleHeight  int
	positioned   bool
	userMoved    bool
	dragging     bool
	dragLayer    bool
	dragDX       int
	dragDY       int
	dragBottom   int
	titleButtons int
	content      widget.Widget
	placed       widget.Widget
	published    widget.Widget
	opacity      float32
	background   *widget.Color
	fullRedraw   bool
	CloseOnEsc   bool
	ctx          client.Context
}

func (w *Window) EnsureWindow(width, height int) bool {
	if w.width != 0 {
		return false
	}
	*w = NewWindow(width, height)
	return true
}

func (w *Window) Close() {
	w.cancelDragLayer(w.ctx)
	w.setOpacity(1)
	w.open = false
	w.dragging = false
	w.content = nil
	w.placed = nil
	w.Publish(w.ctx)
}

func NewWindow(width, height int) Window {
	return Window{
		width:        width,
		height:       height,
		titleHeight:  ROWindowTitleHeight,
		titleButtons: 1,
		opacity:      1,
		CloseOnEsc:   true,
	}
}

func (w *Window) setTitleButtonCount(count int) {
	w.titleButtons = maxInt(0, count)
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
	w.cancelDragLayer(w.ctx)
	w.content = content
	if overlay := w.positionedOverlay(); overlay != nil {
		damage := overlay.setChild(content, windowWidgetContext(w.ctx))
		w.setOpacity(1)
		invalidateWindowLayout(w.ctx)
		invalidateWindowRect(w.ctx, damage)
		return
	}
	w.placed = nil
	w.setOpacity(1)
}

func (w *Window) RebindContent(ctx client.Context, content widget.Widget) {
	if w == nil || !w.open {
		return
	}
	w.Unpublish(ctx)
	w.placed = nil
	w.ctx = ctx
	w.SetContent(content)
	w.Publish(ctx)
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

func (w *Window) Raise(ctx client.Context) {
	if w == nil || w.published == nil || ctx.UIManager == nil {
		return
	}
	if manager, ok := ctx.UIManager.(interface{ RaiseOverlay(widget.Widget) }); ok {
		manager.RaiseOverlay(w.published)
	}
}

func (w *Window) Unpublish(ctx client.Context) {
	if w == nil || ctx.UIManager == nil || w.published == nil {
		return
	}
	w.cancelDragLayer(ctx)
	clearPositionedOverlayDamage(w.published)
	ctx.UIManager.RemoveOverlay(w.published)
	w.published = nil
}

func (w *Window) SetSize(width, height int) {
	if w.width == width && w.height == height {
		return
	}
	w.cancelDragLayer(w.ctx)
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
	w.positioned = true
	w.setPosition(w.ctx, x, y)
	return true
}

func (w *Window) Update(ctx client.Context) bool {
	w.ctx = ctx
	if !w.open || ctx.Input == nil {
		return false
	}
	// The world updates windows in a stable order, not z-order. While another
	// window owns the renderer's drag capture, yielding here prevents an
	// overlapped window from consuming the frame before the owner is updated.
	if !w.dragging && windowDragActive(ctx) {
		return false
	}
	w.ensurePosition(ctx)
	screenW, screenH := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(input.MouseButtonLeft) {
			x := clampWindowInt(ctx.Input.MouseX-w.dragDX, windowScreenMargin, maxInt(windowScreenMargin, screenW-w.width-windowScreenMargin))
			dragHeight := w.height + maxInt(0, w.dragBottom)
			y := clampWindowInt(ctx.Input.MouseY-w.dragDY, windowScreenMargin, maxInt(windowScreenMargin, screenH-dragHeight-windowScreenMargin))
			w.setDragPosition(ctx, x, y)
			return true
		}
		w.dragging = false
		w.endDragLayer(ctx)
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
		if w.titleButtonHit(ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		w.dragging = true
		w.userMoved = true
		w.dragDX = ctx.Input.MouseX - w.x
		w.dragDY = ctx.Input.MouseY - w.y
		w.beginDragLayer(ctx)
		return true
	}
	return true
}

func (w *Window) ensurePosition(ctx client.Context) {
	if w.positioned {
		return
	}
	screenW, screenH := ctx.ScreenSize()
	w.x = maxInt(windowScreenMargin, (screenW-w.width)/2)
	w.y = maxInt(windowScreenMargin, (screenH-w.height)/2)
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
		if overlay := w.positionedOverlay(); overlay != nil {
			damage := overlay.markFrameDirty()
			invalidateWindowRect(w.ctx, damage)
		} else if w.placed != nil {
			markNeedsRedraw(w.placed)
			invalidateWindowRect(w.ctx, windowFrameRect(w.x, w.y, w.width, w.height))
		}
	}
}

func (w *Window) Widget() widget.Widget {
	if w == nil || !w.open || w.content == nil {
		return nil
	}
	if w.placed == nil {
		w.placed = positionedWidget(w.content, w.x, w.y, w.width, w.height)
		if overlay := w.positionedOverlay(); overlay != nil {
			overlay.raiseOnPress = true
		}
	} else if overlay, ok := w.placed.(*positionedOverlay); ok {
		overlay.setFrame(w.x, w.y, w.width, w.height)
	}
	return w.placed
}

func (w *Window) setPosition(ctx client.Context, x, y int) {
	if w.x == x && w.y == y {
		return
	}
	oldX, oldY := w.x, w.y
	w.x, w.y = x, y
	if overlay := w.positionedOverlay(); overlay != nil {
		damage := overlay.setFrame(x, y, w.width, w.height)
		invalidateWindowRect(ctx, damage)
		return
	}
	w.placed = nil
	damage := windowFrameRect(oldX, oldY, w.width, w.height).Union(windowFrameRect(x, y, w.width, w.height)).Expand(windowDirtyPadding)
	invalidateWindowRect(ctx, damage)
}

func (w *Window) setDragPosition(ctx client.Context, x, y int) {
	if !w.dragLayer {
		w.setPosition(ctx, x, y)
		return
	}
	if w.x == x && w.y == y {
		return
	}
	overlay := w.positionedOverlay()
	if overlay == nil {
		w.dragLayer = false
		w.setPosition(ctx, x, y)
		return
	}
	w.x, w.y = x, y
	overlay.setFrameQuiet(x, y, w.width, w.height)
	if app, ok := ctx.UIApp.(windowDragLayerUIApp); ok {
		app.MoveWindowDragLayer(overlay, w.dragLayerRect())
	}
}

func (w *Window) dragLayerRect() geometry.Rect {
	return windowFrameRect(w.x, w.y, w.width, w.height+maxInt(0, w.dragBottom))
}

func (w *Window) positionedOverlay() *positionedOverlay {
	overlay, _ := w.placed.(*positionedOverlay)
	return overlay
}

func (w *Window) titleButtonHit(x, y int) bool {
	if w.titleHeight <= 0 || w.width <= 0 || w.titleButtons == 0 {
		return false
	}
	controlsWidth := w.titleButtons*windowTitleButtonSize + (w.titleButtons-1)*windowTitleButtonGap
	left := w.x + w.width - windowTitleButtonPadR - controlsWidth
	top := w.y + (w.titleHeight-windowTitleButtonSize)/2
	return pointInRect(x, y, left, top, controlsWidth, windowTitleButtonSize)
}

func (w *Window) beginDragLayer(ctx client.Context) {
	if w.dragLayer {
		return
	}
	overlay := w.positionedOverlay()
	if overlay == nil {
		if root := w.Widget(); root != nil {
			overlay, _ = root.(*positionedOverlay)
		}
	}
	if overlay == nil {
		return
	}
	app, ok := ctx.UIApp.(windowDragLayerUIApp)
	if !ok || !app.BeginWindowDragLayer(overlay, w.dragLayerRect()) {
		return
	}
	w.dragLayer = true
	overlay.hidden = true
	damage := overlay.markFrameDirty()
	invalidateWindowRect(ctx, damage)
}

func (w *Window) endDragLayer(ctx client.Context) {
	w.finishDragLayer(ctx, false)
}

func (w *Window) cancelDragLayer(ctx client.Context) {
	w.finishDragLayer(ctx, true)
}

func (w *Window) finishDragLayer(ctx client.Context, cancel bool) {
	if !w.dragLayer {
		return
	}
	overlay := w.positionedOverlay()
	if overlay == nil {
		w.dragLayer = false
		return
	}
	if app, ok := ctx.UIApp.(windowDragLayerUIApp); ok {
		if cancel {
			app.CancelWindowDragLayer(overlay)
		} else {
			app.EndWindowDragLayer(overlay)
		}
	}
	w.dragLayer = false
	overlay.hidden = false
	damage := overlay.markFrameDirty()
	invalidateWindowRect(ctx, damage)
}

type rectInvalidatingUIApp interface {
	InvalidateRect(geometry.Rect)
}

type layoutInvalidatingUIApp interface {
	InvalidateLayout()
}

type widgetContextUIApp interface {
	WidgetContext() widget.Context
}

type windowDragLayerUIApp interface {
	BeginWindowDragLayer(token any, rect geometry.Rect) bool
	MoveWindowDragLayer(token any, rect geometry.Rect)
	EndWindowDragLayer(token any)
	CancelWindowDragLayer(token any)
}

type windowDragStateUIApp interface {
	WindowDragActive() bool
}

func windowDragActive(ctx client.Context) bool {
	app, ok := ctx.UIApp.(windowDragStateUIApp)
	return ok && app.WindowDragActive()
}

func invalidateWindowRect(ctx client.Context, rect geometry.Rect) {
	if ctx.UIApp == nil || rect.IsEmpty() {
		return
	}
	if app, ok := ctx.UIApp.(rectInvalidatingUIApp); ok {
		app.InvalidateRect(rect)
		return
	}
	ctx.UIApp.Invalidate()
}

func invalidateWindowLayout(ctx client.Context) {
	if ctx.UIApp == nil {
		return
	}
	if app, ok := ctx.UIApp.(layoutInvalidatingUIApp); ok {
		app.InvalidateLayout()
		return
	}
	ctx.UIApp.Invalidate()
}

func windowWidgetContext(ctx client.Context) widget.Context {
	if app, ok := ctx.UIApp.(widgetContextUIApp); ok {
		return app.WidgetContext()
	}
	return nil
}

func windowFrameRect(x, y, width, height int) geometry.Rect {
	return geometry.NewRect(float32(x), float32(y), float32(width), float32(height))
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
	w.SetBounds(windowFrameRect(x, y, width, height))
	if setter, ok := content.(interface{ SetParent(widget.Widget) }); ok {
		setter.SetParent(w)
	}
	return w
}

func clearPositionedOverlayDamage(root widget.Widget) {
	overlay, ok := root.(*positionedOverlay)
	if !ok {
		return
	}
	overlay.clearDamage()
	overlay.hidden = false
}

type positionedOverlay struct {
	widget.WidgetBase
	child         widget.Widget
	x, y          int
	width, height int
	damage        geometry.Rect
	hasDamage     bool
	hidden        bool
	raiseOnPress  bool
}

func (w *positionedOverlay) setFrame(x, y, width, height int) geometry.Rect {
	if w.x == x && w.y == y && w.width == width && w.height == height {
		return geometry.Rect{}
	}
	oldFrame := w.frameRect()
	w.x, w.y = x, y
	w.width, w.height = width, height
	newFrame := w.frameRect()
	w.SetBounds(newFrame)
	w.SetScreenOrigin(newFrame.Min)
	return w.markDamage(oldFrame.Union(newFrame))
}

func (w *positionedOverlay) setFrameQuiet(x, y, width, height int) {
	w.x, w.y = x, y
	w.width, w.height = width, height
	frame := w.frameRect()
	w.SetBounds(frame)
	w.SetScreenOrigin(frame.Min)
}

func (w *positionedOverlay) setChild(child widget.Widget, ctx widget.Context) geometry.Rect {
	if w.child == child {
		return w.markFrameDirty()
	}
	if w.child != nil {
		widget.UnmountTree(w.child)
	}
	w.child = child
	if child != nil {
		if setter, ok := child.(interface{ SetParent(widget.Widget) }); ok {
			setter.SetParent(w)
		}
		if ctx != nil {
			widget.MountTree(child, ctx)
		}
	}
	return w.markFrameDirty()
}

func (w *positionedOverlay) markFrameDirty() geometry.Rect {
	return w.markDamage(w.frameRect())
}

func (w *positionedOverlay) markDamage(rect geometry.Rect) geometry.Rect {
	if rect.IsEmpty() {
		return geometry.Rect{}
	}
	rect = rect.Expand(windowDirtyPadding)
	if w.hasDamage && !w.damage.IsEmpty() {
		w.damage = w.damage.Union(rect)
	} else {
		w.damage = rect
	}
	w.hasDamage = true
	w.MarkRedrawLocal()
	return rect
}

func (w *positionedOverlay) frameRect() geometry.Rect {
	return windowFrameRect(w.x, w.y, w.width, w.height)
}

func (w *positionedOverlay) ScreenBounds() geometry.Rect {
	if w.hasDamage && !w.damage.IsEmpty() {
		return w.damage
	}
	return w.WidgetBase.ScreenBounds()
}

func (w *positionedOverlay) clearDamage() {
	w.damage = geometry.Rect{}
	w.hasDamage = false
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
	defer w.clearDamage()
	if !w.IsVisible() || w.hidden || w.child == nil {
		return
	}
	canvas.PushTransform(w.Bounds().Min)
	widget.StampScreenOrigin(w.child, canvas)
	if w.hasDamage {
		widget.ClearRedrawInTree(w.child)
	}
	widget.DrawChild(w.child, ctx, canvas)
	canvas.PopTransform()
}

func (w *positionedOverlay) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() || w.hidden || w.child == nil {
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
	if w.hidden || w.hasDamage || w.child == nil {
		return nil
	}
	return []widget.Widget{w.child}
}

func centeredWindowRect(ctx client.Context, width, height int) (int, int, int, int) {
	screenW, screenH := ctx.ScreenSize()
	x := (screenW - width) / 2
	y := (screenH - height) / 2
	if x < windowScreenMargin {
		x = windowScreenMargin
	}
	if y < windowScreenMargin {
		y = windowScreenMargin
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
