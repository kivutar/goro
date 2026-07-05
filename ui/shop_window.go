package ui

import (
	"fmt"
	"image"
	"log"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	shopWindowWidth  = 360
	shopWindowHeight = 276
	shopWindowTitleH = 28
	shopWindowPad    = 10
	shopCartRowH     = 28
	shopBuyRowH      = 31
	shopListH        = 194
	shopFooterGap    = 8
	shopButtonH      = 24
	shopRowButtonGap = 2
	shopRowRightPad  = 3

	shopDealWidth  = 244
	shopDealHeight = 108
)

const (
	shopModeNone = iota
	shopModeBuy
	shopModeSell
)

var (
	shopTitleColor  = TitleTextColor
	shopTextColor   = TextColor
	shopMutedColor  = MutedTextColor
	shopGoodColor   = GoodTextColor
	shopErrorColor  = ErrorTextColor
	shopButtonColor = ButtonColor
	shopHoverColor  = ButtonHoverColor
	shopDropColor   = PanelHoverColor
)

type ShopWindow struct {
	dealOpen        bool
	dealNPCID       uint32
	open            bool
	mode            int
	x               int
	y               int
	positioned      bool
	dragging        bool
	dragDX          int
	dragDY          int
	sellable        map[uint16]network.ShopSellItem
	cart            []shopSellCartItem
	buyItems        []network.ShopBuyItem
	buyCart         []shopBuyCartItem
	buyScroll       int
	buyWindow       WindowState
	buyItemInfo     *ItemInfoWindow
	buyIcons        map[shopItemIconKey]image.Image
	buyIconMiss     map[shopItemIconKey]struct{}
	status          string
	statusGood      bool
	statusAt        time.Time
	closePacketSent bool
}

type shopSellCartItem struct {
	item    session.InventoryItem
	price   uint32
	over    uint32
	amount  uint16
	max     uint16
	addedAt time.Time
}

type shopBuyCartItem struct {
	item   network.ShopBuyItem
	amount uint16
}

type shopItemIconKey struct {
	itemID     uint16
	identified bool
}

func (w *ShopWindow) OpenDeal(selection network.ShopDealSelection, ctx Context) {
	w.dealOpen = true
	w.dealNPCID = selection.NPCID
	w.ensureSellPosition(ctx)
}

func (w *ShopWindow) OpenSell(list []network.ShopSellItem, ctx Context) {
	w.dealOpen = false
	w.open = true
	w.mode = shopModeSell
	w.ensureSellPosition(ctx)
	w.sellable = make(map[uint16]network.ShopSellItem, len(list))
	for _, item := range list {
		w.sellable[item.Index] = item
	}
	w.cart = nil
	w.buyItems = nil
	w.buyCart = nil
	w.buyScroll = 0
	w.status = "Drag items here to sell"
	w.statusGood = true
	w.statusAt = time.Now()
	w.closePacketSent = false
}

func (w *ShopWindow) OpenBuy(list []network.ShopBuyItem, ctx Context) {
	w.dealOpen = false
	w.open = true
	w.mode = shopModeBuy
	w.ensureSellPosition(ctx)
	w.buyItems = append(w.buyItems[:0], list...)
	w.buyCart = nil
	w.buyScroll = 0
	w.sellable = nil
	w.cart = nil
	w.status = "Select items to buy"
	w.statusGood = true
	w.statusAt = time.Now()
	w.closePacketSent = false
	w.openBuyWindow(ctx)
}

func (w *ShopWindow) ApplyResult(ctx Context, result network.ShopResult) {
	if !result.Sell {
		if result.Result == 0 {
			w.status = "Deal completed"
			w.statusGood = true
			w.statusAt = time.Now()
			w.open = false
			w.mode = shopModeNone
			w.buyCart = nil
			w.buyItems = nil
			w.closePacketSent = true
			w.closeBuyWindow(ctx)
			return
		}
		w.status = fmt.Sprintf("Buy failed result=%d", result.Result)
		w.statusGood = result.Result == 0
		w.statusAt = time.Now()
		w.refreshBuyWindow(ctx)
		return
	}
	if result.Result == 0 {
		w.status = "Deal completed"
		w.statusGood = true
		w.statusAt = time.Now()
		w.open = false
		w.mode = shopModeNone
		w.cart = nil
		w.sellable = nil
		w.closePacketSent = true
		return
	}
	w.status = "Sell failed"
	w.statusGood = false
	w.statusAt = time.Now()
}

func (w *ShopWindow) Update(ctx Context, itemInfo *ItemInfoWindow) bool {
	if ctx.Input == nil {
		return false
	}
	if w.dealOpen {
		return w.updateDeal(ctx)
	}
	if !w.open {
		return false
	}
	w.ensureSellPosition(ctx)
	if w.mode == shopModeBuy {
		return w.updateBuyWindow(ctx, itemInfo)
	}
	width, height := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(render.MouseButtonLeft) {
			w.x = clampShopWindowInt(ctx.Input.MouseX-w.dragDX, 8, maxInt(8, width-shopWindowWidth-8))
			w.y = clampShopWindowInt(ctx.Input.MouseY-w.dragDY, 8, maxInt(8, height-shopWindowHeight-8))
			return true
		}
		w.dragging = false
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.cancel(ctx)
		return true
	}
	inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, shopWindowWidth, shopWindowHeight)
	if inside && ctx.Input.WheelY != 0 && w.mode == shopModeBuy {
		w.scrollBuyBy(ctx.Input.WheelY)
		return true
	}
	if ctx.Input.MouseJustPressed(render.MouseButtonRight) {
		mx, my := ctx.Input.MouseX, ctx.Input.MouseY
		if !inside {
			return false
		}
		if itemInfo != nil {
			if item, ok := w.itemAt(mx, my); ok {
				itemInfo.openItem(ctx, item, mx, my)
				return true
			}
		}
		return true
	}
	if !ctx.Input.MouseJustPressed(render.MouseButtonLeft) {
		return inside
	}
	mx, my := ctx.Input.MouseX, ctx.Input.MouseY
	if !inside {
		return false
	}
	cx, cy, cw, ch := w.closeBounds()
	if pointInRect(mx, my, cx, cy, cw, ch) {
		w.cancel(ctx)
		return true
	}
	if pointInRect(mx, my, w.x, w.y, shopWindowWidth, shopWindowTitleH) {
		w.dragging = true
		w.dragDX = mx - w.x
		w.dragDY = my - w.y
		return true
	}
	if w.mode == shopModeBuy {
		if w.handleBuyClick(mx, my) {
			return true
		}
	}
	sx, sy, sw, sh := w.sellButtonBounds()
	if pointInRect(mx, my, sx, sy, sw, sh) {
		w.submit(ctx)
		return true
	}
	bx, by, bw, bh := w.cancelButtonBounds()
	if pointInRect(mx, my, bx, by, bw, bh) {
		w.cancel(ctx)
		return true
	}
	if w.mode == shopModeSell {
		for i := range w.cart {
			if w.handleCartButton(ctx, i, mx, my) {
				return true
			}
		}
	}
	return true
}

func (w *ShopWindow) updateDeal(ctx Context) bool {
	width, height := ctx.ScreenSize()
	x := (width - shopDealWidth) / 2
	y := (height - shopDealHeight) * 2 / 3
	if !ctx.Input.MouseJustPressed(render.MouseButtonLeft) {
		return pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, x, y, shopDealWidth, shopDealHeight)
	}
	mx, my := ctx.Input.MouseX, ctx.Input.MouseY
	if !pointInRect(mx, my, x, y, shopDealWidth, shopDealHeight) {
		return false
	}
	if pointInRect(mx, my, x+18, y+64, 60, 24) {
		w.sendDealSelection(ctx, 0)
		return true
	}
	if pointInRect(mx, my, x+92, y+64, 60, 24) {
		w.sendDealSelection(ctx, 1)
		return true
	}
	if pointInRect(mx, my, x+166, y+64, 60, 24) {
		w.dealOpen = false
		return true
	}
	return true
}

func (w *ShopWindow) Draw(screen *render.Image, ctx Context, assets AssetRenderer) {
	if screen == nil {
		return
	}
	if w.dealOpen {
		w.drawDeal(screen, ctx)
	}
	if !w.open {
		return
	}
	w.ensureSellPosition(ctx)
	if w.mode == shopModeBuy && ctx.UIManager != nil {
		w.buyWindow.Publish(ctx)
		return
	}
	x, y := w.x, w.y
	DrawTitledWindowFrame(screen, x, y, shopWindowWidth, shopWindowHeight, shopWindowTitleH)
	title := "Sell Items"
	if w.mode == shopModeBuy {
		title = "Buy Items"
	}
	DrawWindowTitle(screen, x, y, shopWindowTitleH, shopWindowPad, title, shopTitleColor)
	cx, cy, cw, ch := w.closeBounds()
	DrawCloseButton(screen, cx, cy, cw, ch, shopButtonColor, shopTextColor)

	if w.mode == shopModeBuy {
		w.drawBuyRows(screen, ctx, assets)
	} else {
		dx, dy, dw, dh := w.dropBounds()
		fill := PanelBodyColor
		if ctx.Input != nil && pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, dx, dy, dw, dh) {
			fill = shopDropColor
		}
		DrawSurface(screen, dx, dy, dw, dh, fill, WindowBorderColor)
		if len(w.cart) == 0 {
			label := "Drop inventory items here"
			labelW, labelH := render.DebugTextSize(label)
			render.DebugPrintAtColor(screen, label, dx+(dw-labelW)/2, dy+(dh-labelH)/2, shopMutedColor)
		} else {
			for i, item := range w.visibleCartItems() {
				w.drawCartRow(screen, ctx, assets, i, item)
			}
		}
	}
	sx, sy, sw, sh := w.sellButtonBounds()
	render.DebugPrintAtColor(screen, fmt.Sprintf("Total: %s z", formatHUDNumber(int64(w.total()))), x+shopWindowPad, sy+6, shopTextColor)
	actionLabel := "Sell"
	enabled := len(w.cart) > 0
	if w.mode == shopModeBuy {
		actionLabel = "Buy"
		enabled = len(w.buyCart) > 0
	}
	w.drawButton(screen, sx, sy, sw, sh, actionLabel, enabled)
	bx, by, bw, bh := w.cancelButtonBounds()
	w.drawButton(screen, bx, by, bw, bh, "Cancel", true)
}

func (w *ShopWindow) drawDeal(screen *render.Image, ctx Context) {
	width, height := ctx.ScreenSize()
	x := (width - shopDealWidth) / 2
	y := (height - shopDealHeight) * 2 / 3
	DrawTitledWindowFrame(screen, x, y, shopDealWidth, shopDealHeight, shopWindowTitleH)
	DrawWindowTitle(screen, x, y, shopWindowTitleH, shopWindowPad, "Shop", shopTitleColor)
	prompt := "What do you want to do?"
	promptW, _ := render.DebugTextSize(prompt)
	render.DebugPrintAtColor(screen, prompt, x+(shopDealWidth-promptW)/2, y+42, shopTextColor)
	w.drawButton(screen, x+18, y+64, 60, 24, "Buy", true)
	w.drawButton(screen, x+92, y+64, 60, 24, "Sell", true)
	w.drawButton(screen, x+166, y+64, 60, 24, "Cancel", true)
}

func (w *ShopWindow) CursorAction(ctx Context) (int, bool) {
	if ctx.Input == nil {
		return 0, false
	}
	if w.dealOpen {
		width, height := ctx.ScreenSize()
		x := (width - shopDealWidth) / 2
		y := (height - shopDealHeight) * 2 / 3
		if pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, x, y, shopDealWidth, shopDealHeight) {
			return CursorActionClick, true
		}
	}
	if w.open && pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, shopWindowWidth, shopWindowHeight) {
		return CursorActionClick, true
	}
	return 0, false
}

func (w *ShopWindow) ensureBuyWindow() {
	if w.buyWindow.width == 0 {
		w.buyWindow = NewWindowState(shopWindowWidth, shopWindowHeight)
		w.buyWindow.SetCloseOnEscape(false)
	}
}

func (w *ShopWindow) openBuyWindow(ctx Context) {
	w.ensureBuyWindow()
	w.buyWindow.OpenAt(w.x, w.y, w.buyWidgetTree(ctx))
	w.buyWindow.Publish(ctx)
}

func (w *ShopWindow) refreshBuyWindow(ctx Context) {
	if w.mode != shopModeBuy || !w.open {
		w.closeBuyWindow(ctx)
		return
	}
	w.ensureBuyWindow()
	if !w.buyWindow.IsOpen() {
		w.openBuyWindow(ctx)
		return
	}
	w.buyWindow.SetContent(w.buyWidgetTree(ctx))
	w.buyWindow.Publish(ctx)
}

func (w *ShopWindow) closeBuyWindow(ctx Context) {
	if w.buyWindow.width == 0 {
		return
	}
	w.buyWindow.Close()
	w.buyWindow.Unpublish(ctx)
}

func (w *ShopWindow) buyWidgetTree(ctx Context) widget.Widget {
	rows := make([]widget.Widget, 0, visibleBuyRows())
	for _, item := range w.visibleBuyItems() {
		item := item
		amount := w.buyAmount(item.ItemID)
		rows = append(rows, newShopBuyRowWidget(shopBuyRowConfig{
			item:   item,
			name:   inventoryItemDisplayName(ctx.Resources, session.InventoryItem{ItemID: item.ItemID, Type: item.Type, Identified: true}),
			price:  shopBuyItemPrice(item),
			amount: amount,
			icon:   w.shopItemIconImage(ctx.Resources, item.ItemID),
			onAdd: func() {
				w.addBuyItem(item)
				w.refreshBuyWindow(ctx)
			},
			onRemove: func() {
				w.decrementBuyItem(item.ItemID)
				w.refreshBuyWindow(ctx)
			},
			onInfo: func(mx, my int) {
				if w.buyItemInfo != nil {
					w.buyItemInfo.openItem(ctx, session.InventoryItem{ItemID: item.ItemID, Type: item.Type, Identified: true, Amount: 1}, mx, my)
				}
			},
		}))
	}
	for len(rows) < visibleBuyRows() {
		rows = append(rows, primitives.Box().Height(shopBuyRowH-3))
	}
	statusColor := rotheme.Default.Colors.MutedText
	if w.status != "" && !w.statusGood {
		statusColor = widget.RGBA8(176, 42, 42, 255)
	}
	return Window(
		Title("Buy Items"),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
		}),
		Size(shopWindowWidth, shopWindowHeight),
		FooterHeight(42),
		FooterPadding(10),
		Content(
			primitives.Box(
				primitives.Box(
					primitives.HBox(
						primitives.Box(rows...).
							Width(shopWindowWidth-shopWindowPad*2-22).
							Height(shopListH-10).
							Gap(0),
						primitives.Box(
							rotheme.IconButtonDisabled(rotheme.IconButtonUp, !w.canScrollBuy(-1), func() {
								w.scrollBuyBy(1)
								w.refreshBuyWindow(ctx)
							}),
							primitives.Expanded(primitives.Box()),
							rotheme.IconButtonDisabled(rotheme.IconButtonDown, !w.canScrollBuy(1), func() {
								w.scrollBuyBy(-1)
								w.refreshBuyWindow(ctx)
							}),
						).
							Width(17).
							Height(shopListH-10),
					).
						Gap(3),
				).
					Padding(5).
					Height(shopListH).
					Background(rotheme.Default.Colors.PanelBody).
					BorderStyle(1, rotheme.Default.Colors.WindowBorder),
				rotheme.Text(w.status).
					Color(statusColor),
			).Padding(shopWindowPad),
		),
		Footer(
			primitives.HBox(
				rotheme.Text(fmt.Sprintf("Total: %s z", formatHUDNumber(w.total()))),
				primitives.Expanded(primitives.Box()),
				rotheme.ButtonDisabled("Buy", len(w.buyCart) == 0, func() {
					w.submitBuy(ctx)
					w.refreshBuyWindow(ctx)
				}).Width(58),
				rotheme.Button("Cancel", func() {
					w.cancel(ctx)
				}).Width(62),
			).
				CrossAlign(primitives.CrossAxisCenter).
				Gap(8),
		),
	)
}

func (w *ShopWindow) canScrollBuy(delta int) bool {
	maxScroll := maxInt(0, len(w.buyItems)-visibleBuyRows())
	next := w.buyScroll + delta
	return next >= 0 && next <= maxScroll
}

func (w *ShopWindow) updateBuyWindow(ctx Context, itemInfo *ItemInfoWindow) bool {
	w.ensureBuyWindow()
	w.buyItemInfo = itemInfo
	if !w.buyWindow.IsOpen() {
		w.openBuyWindow(ctx)
	}
	w.x, w.y = w.buyWindow.x, w.buyWindow.y
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.cancel(ctx)
		return true
	}
	inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, shopWindowWidth, shopWindowHeight)
	if inside && ctx.Input.WheelY != 0 {
		w.scrollBuyBy(ctx.Input.WheelY)
		w.refreshBuyWindow(ctx)
		return true
	}
	if inside && ctx.Input.MouseJustPressed(render.MouseButtonRight) {
		if itemInfo != nil {
			if item, ok := w.itemAt(ctx.Input.MouseX, ctx.Input.MouseY); ok {
				itemInfo.openItem(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY)
			}
		}
		return true
	}
	consumed := w.buyWindow.Update(ctx)
	w.x, w.y = w.buyWindow.x, w.buyWindow.y
	if !w.buyWindow.IsOpen() {
		w.cancel(ctx)
		return true
	}
	w.buyWindow.Publish(ctx)
	return consumed || inside
}

func (w *ShopWindow) shopItemIconImage(manager *res.Manager, itemID uint16) image.Image {
	if manager == nil || itemID == 0 {
		return nil
	}
	key := shopItemIconKey{itemID: itemID, identified: true}
	if w.buyIcons != nil {
		if img := w.buyIcons[key]; img != nil {
			return img
		}
	}
	if _, ok := w.buyIconMiss[key]; ok {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(itemID), true)
	if !ok {
		w.markShopIconMiss(key)
		return nil
	}
	img, _, err := res.LoadImage(manager, res.ItemIconTextureCandidates(resourceName))
	if err != nil {
		w.markShopIconMiss(key)
		return nil
	}
	if w.buyIcons == nil {
		w.buyIcons = make(map[shopItemIconKey]image.Image)
	}
	w.buyIcons[key] = img
	return img
}

func (w *ShopWindow) markShopIconMiss(key shopItemIconKey) {
	if w.buyIconMiss == nil {
		w.buyIconMiss = make(map[shopItemIconKey]struct{})
	}
	w.buyIconMiss[key] = struct{}{}
}

func shopBuyItemPrice(item network.ShopBuyItem) uint32 {
	if item.DiscountPrice != 0 {
		return item.DiscountPrice
	}
	return item.Price
}

type shopBuyRowConfig struct {
	item     network.ShopBuyItem
	name     string
	price    uint32
	amount   uint16
	icon     image.Image
	onAdd    func()
	onRemove func()
	onInfo   func(mx, my int)
}

type shopBuyRowWidget struct {
	widget.WidgetBase
	cfg     shopBuyRowConfig
	hovered bool
}

func newShopBuyRowWidget(cfg shopBuyRowConfig) *shopBuyRowWidget {
	w := &shopBuyRowWidget{cfg: cfg}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *shopBuyRowWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(shopWindowWidth-shopWindowPad*2-22, shopBuyRowH-3))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *shopBuyRowWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	fill := widget.RGBA8(246, 249, 253, 255)
	if w.hovered {
		fill = rotheme.Default.Colors.ButtonHover
	}
	canvas.DrawRect(bounds, fill)
	canvas.StrokeRect(bounds, rotheme.Default.Colors.WindowBorder, 1)
	if w.cfg.icon != nil {
		iconBounds := w.cfg.icon.Bounds()
		iconH := float32(iconBounds.Dy())
		canvas.DrawImage(w.cfg.icon, geometry.Pt(bounds.Min.X+4, bounds.Min.Y+(bounds.Height()-iconH)/2))
	}
	nameBounds := geometry.NewRect(bounds.Min.X+32, bounds.Min.Y+4, 128, bounds.Height()-8)
	priceBounds := geometry.NewRect(bounds.Min.X+168, bounds.Min.Y+4, 58, bounds.Height()-8)
	amountBounds := geometry.NewRect(bounds.Min.X+226, bounds.Min.Y+4, 40, bounds.Height()-8)
	rotheme.DrawText(canvas, trimRunes(w.cfg.name, 20), nameBounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignLeft)
	rotheme.DrawText(canvas, formatHUDNumber(int64(w.cfg.price)), priceBounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignRight)
	if w.cfg.amount > 0 {
		rotheme.DrawText(canvas, fmt.Sprintf("x%d", w.cfg.amount), amountBounds, rotheme.Default.Typography.TextSize, widget.RGBA8(54, 128, 76, 255), false, widget.TextAlignLeft)
	}
	minus := geometry.NewRect(bounds.Max.X-shopRowRightPad-IconButtonSize*2-shopRowButtonGap, bounds.Min.Y+6, IconButtonSize, IconButtonSize)
	plus := geometry.NewRect(bounds.Max.X-shopRowRightPad-IconButtonSize, bounds.Min.Y+6, IconButtonSize, IconButtonSize)
	rotheme.DrawIconButton(canvas, minus, rotheme.IconButtonMinus, false, w.cfg.amount == 0)
	rotheme.DrawIconButton(canvas, plus, rotheme.IconButtonPlus, false, false)
}

func (w *shopBuyRowWidget) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter:
		w.hovered = true
		ctx.SetCursor(widget.CursorPointer)
		w.SetNeedsRedraw(true)
		return true
	case event.MouseLeave:
		w.hovered = false
		ctx.SetCursor(widget.CursorDefault)
		w.SetNeedsRedraw(true)
		return false
	case event.MousePress:
		switch mouse.Button {
		case event.ButtonLeft:
			if w.minusButtonBounds().Contains(mouse.Position) {
				if w.cfg.amount > 0 && w.cfg.onRemove != nil {
					w.cfg.onRemove()
				}
				return true
			}
			if w.plusButtonBounds().Contains(mouse.Position) || w.bodyBounds().Contains(mouse.Position) {
				if w.cfg.onAdd != nil {
					w.cfg.onAdd()
				}
				return true
			}
		case event.ButtonRight:
			if w.bodyBounds().Contains(mouse.Position) && w.cfg.onInfo != nil {
				w.cfg.onInfo(int(mouse.GlobalPosition.X), int(mouse.GlobalPosition.Y))
				return true
			}
		}
	}
	return true
}

func (w *shopBuyRowWidget) bodyBounds() geometry.Rect {
	bounds := w.Bounds()
	local := geometry.NewRect(0, 0, bounds.Width()-58, bounds.Height())
	return local
}

func (w *shopBuyRowWidget) minusButtonBounds() geometry.Rect {
	bounds := w.Bounds()
	return geometry.NewRect(bounds.Width()-shopRowRightPad-IconButtonSize*2-shopRowButtonGap, 6, IconButtonSize, IconButtonSize)
}

func (w *shopBuyRowWidget) plusButtonBounds() geometry.Rect {
	bounds := w.Bounds()
	return geometry.NewRect(bounds.Width()-shopRowRightPad-IconButtonSize, 6, IconButtonSize, IconButtonSize)
}

func (w *ShopWindow) AcceptInventoryDrop(ctx Context, item session.InventoryItem, mouseX, mouseY int) bool {
	dx, dy, dw, dh := w.dropBounds()
	if !w.open || w.mode != shopModeSell || !pointInRect(mouseX, mouseY, dx, dy, dw, dh) {
		return false
	}
	sell, ok := w.sellable[item.Index]
	if !ok {
		w.status = "That item cannot be sold"
		w.statusGood = false
		w.statusAt = time.Now()
		return true
	}
	w.addCartItem(item, sell)
	return true
}

func (w *ShopWindow) addCartItem(item session.InventoryItem, sell network.ShopSellItem) {
	maxAmount := uint16(maxInt(1, item.Amount))
	for i := range w.cart {
		if w.cart[i].item.Index == item.Index {
			if w.cart[i].amount < w.cart[i].max {
				w.cart[i].amount++
			}
			w.status = "Item added"
			w.statusGood = true
			w.statusAt = time.Now()
			return
		}
	}
	w.cart = append(w.cart, shopSellCartItem{
		item:    item,
		price:   sell.Price,
		over:    sell.OverchargePrice,
		amount:  1,
		max:     maxAmount,
		addedAt: time.Now(),
	})
	w.status = "Item added"
	w.statusGood = true
	w.statusAt = time.Now()
}

func (w *ShopWindow) addBuyItem(item network.ShopBuyItem) {
	for i := range w.buyCart {
		if w.buyCart[i].item.ItemID == item.ItemID {
			w.buyCart[i].amount++
			w.status = "Item added"
			w.statusGood = true
			w.statusAt = time.Now()
			return
		}
	}
	w.buyCart = append(w.buyCart, shopBuyCartItem{item: item, amount: 1})
	w.status = "Item added"
	w.statusGood = true
	w.statusAt = time.Now()
}

func (w *ShopWindow) submit(ctx Context) {
	if w.mode == shopModeBuy {
		w.submitBuy(ctx)
		return
	}
	if len(w.cart) == 0 {
		w.status = "No items selected"
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	if ctx.Network == nil {
		w.status = "Not connected"
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	items := make([]network.SellRequestItem, 0, len(w.cart))
	for _, item := range w.cart {
		items = append(items, network.SellRequestItem{Index: item.item.Index, Amount: item.amount})
	}
	if err := ctx.Network.SendShopSellItems(items); err != nil {
		w.status = err.Error()
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	w.closePacketSent = true
	w.status = "Sell requested"
	w.statusGood = true
	w.statusAt = time.Now()
}

func (w *ShopWindow) submitBuy(ctx Context) {
	if len(w.buyCart) == 0 {
		w.status = "No items selected"
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	if ctx.Session != nil && w.total() > ctx.Session.Inventory.Zeny {
		w.status = "Not enough zeny"
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	if ctx.Network == nil {
		w.status = "Not connected"
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	items := make([]network.BuyRequestItem, 0, len(w.buyCart))
	for _, item := range w.buyCart {
		items = append(items, network.BuyRequestItem{ItemID: item.item.ItemID, Amount: item.amount})
	}
	if err := ctx.Network.SendShopBuyItems(items); err != nil {
		w.status = err.Error()
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	w.closePacketSent = true
	w.status = "Buy requested"
	w.statusGood = true
	w.statusAt = time.Now()
}

func (w *ShopWindow) handleBuyClick(mx, my int) bool {
	for row, item := range w.visibleBuyItems() {
		itemIndex := w.buyScroll + row
		x, y, width, _ := w.buyRowBounds(row)
		minus := shopRowButtonBounds(x, y+6, width, 1)
		plus := shopRowButtonBounds(x, y+6, width, 0)
		switch {
		case pointInRect(mx, my, minus[0], minus[1], minus[2], minus[3]):
			w.decrementBuyItem(item.ItemID)
			return true
		case pointInRect(mx, my, plus[0], plus[1], plus[2], plus[3]):
			w.addBuyItem(item)
			return true
		case pointInRect(mx, my, x, y, width-58, shopBuyRowH-3):
			w.addBuyItem(w.buyItems[itemIndex])
			return true
		}
	}
	return false
}

func (w *ShopWindow) itemAt(mx, my int) (session.InventoryItem, bool) {
	if w.mode == shopModeBuy {
		for row, item := range w.visibleBuyItems() {
			x, y, width, _ := w.buyRowBounds(row)
			if !pointInRect(mx, my, x, y, width-58, shopBuyRowH-3) {
				continue
			}
			return session.InventoryItem{ItemID: item.ItemID, Type: item.Type, Identified: true, Amount: 1}, true
		}
		return session.InventoryItem{}, false
	}
	if w.mode != shopModeSell {
		return session.InventoryItem{}, false
	}
	for row, cartItem := range w.visibleCartItems() {
		x, y, width, height := w.cartRowBounds(row)
		if pointInRect(mx, my, x, y, width, height) {
			item := cartItem.item
			item.Amount = int(cartItem.amount)
			return item, true
		}
	}
	return session.InventoryItem{}, false
}

func (w *ShopWindow) decrementBuyItem(itemID uint16) {
	for i := range w.buyCart {
		if w.buyCart[i].item.ItemID != itemID {
			continue
		}
		if w.buyCart[i].amount > 1 {
			w.buyCart[i].amount--
		} else {
			w.buyCart = append(w.buyCart[:i], w.buyCart[i+1:]...)
		}
		return
	}
}

func (w *ShopWindow) cancel(ctx Context) {
	if w.open && !w.closePacketSent && ctx.Network != nil {
		if w.mode == shopModeBuy {
			if err := ctx.Network.SendShopBuyItems(nil); err != nil {
				log.Printf("send empty buy list on shop close failed: %v", err)
			}
		} else {
			if err := ctx.Network.SendShopSellItems(nil); err != nil {
				log.Printf("send empty sell list on shop close failed: %v", err)
			}
		}
	}
	w.open = false
	w.dealOpen = false
	w.mode = shopModeNone
	w.cart = nil
	w.sellable = nil
	w.buyItems = nil
	w.buyCart = nil
	w.closePacketSent = true
	w.closeBuyWindow(ctx)
}

func (w *ShopWindow) sendDealSelection(ctx Context, dealType uint8) {
	if ctx.Network == nil {
		w.status = "Not connected"
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	if err := ctx.Network.SendShopDealSelection(w.dealNPCID, dealType); err != nil {
		w.status = err.Error()
		w.statusGood = false
		w.statusAt = time.Now()
		return
	}
	w.dealOpen = false
	if dealType == 1 {
		w.status = "Waiting for sell list"
	} else {
		w.status = "Buy requested"
	}
	w.statusGood = true
	w.statusAt = time.Now()
}

func (w *ShopWindow) ensureSellPosition(ctx Context) {
	if w.positioned {
		return
	}
	width, _ := ctx.ScreenSize()
	w.x = maxInt(8, (width-shopWindowWidth)/2)
	w.y = 120
	w.positioned = true
}

func (w *ShopWindow) closeBounds() (int, int, int, int) {
	return w.x + shopWindowWidth - 24, w.y + 7, IconButtonSize, IconButtonSize
}

func (w *ShopWindow) dropBounds() (int, int, int, int) {
	return w.x + shopWindowPad, w.y + shopWindowTitleH + 12, shopWindowWidth - shopWindowPad*2, shopListH
}

func (w *ShopWindow) sellButtonBounds() (int, int, int, int) {
	_, dropY, _, dropH := w.dropBounds()
	return w.x + shopWindowWidth - 146, dropY + dropH + shopFooterGap, 58, shopButtonH
}

func (w *ShopWindow) cancelButtonBounds() (int, int, int, int) {
	_, dropY, _, dropH := w.dropBounds()
	return w.x + shopWindowWidth - 80, dropY + dropH + shopFooterGap, 62, shopButtonH
}

func (w *ShopWindow) cartRowBounds(row int) (int, int, int, int) {
	x, y, width, _ := w.dropBounds()
	return x + 5, y + 5 + row*shopCartRowH, width - 10, shopCartRowH - 3
}

func (w *ShopWindow) buyRowBounds(row int) (int, int, int, int) {
	x, y, width, _ := w.dropBounds()
	return x + 5, y + 5 + row*shopBuyRowH, width - 18, shopBuyRowH - 3
}

func (w *ShopWindow) visibleBuyItems() []network.ShopBuyItem {
	if w.buyScroll < 0 {
		w.buyScroll = 0
	}
	if w.buyScroll >= len(w.buyItems) {
		return nil
	}
	end := minInt(len(w.buyItems), w.buyScroll+visibleBuyRows())
	return w.buyItems[w.buyScroll:end]
}

func visibleBuyRows() int {
	return 6
}

func (w *ShopWindow) scrollBuyBy(wheelY float64) {
	if wheelY > 0 {
		w.buyScroll--
	} else if wheelY < 0 {
		w.buyScroll++
	}
	maxScroll := maxInt(0, len(w.buyItems)-visibleBuyRows())
	if w.buyScroll < 0 {
		w.buyScroll = 0
	}
	if w.buyScroll > maxScroll {
		w.buyScroll = maxScroll
	}
}

func (w *ShopWindow) visibleCartItems() []shopSellCartItem {
	visible := minInt(len(w.cart), 6)
	return w.cart[:visible]
}

func (w *ShopWindow) drawCartRow(screen *render.Image, ctx Context, assets AssetRenderer, row int, item shopSellCartItem) {
	x, y, width, height := w.cartRowBounds(row)
	DrawSurface(screen, x, y, width, height, PanelAltColor, WindowBorderColor)
	if assets != nil {
		assets.DrawInventoryItemIcon(screen, ctx.Resources, item.item, x+3, y+1)
	}
	name := inventoryItemDisplayName(ctx.Resources, item.item)
	render.DebugPrintAtColor(screen, trimRunes(name, 18), x+inventoryIconSize+10, y+5, shopTextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("x%d", item.amount), x+170, y+5, shopMutedColor)
	render.DebugPrintAtColor(screen, formatHUDNumber(int64(item.over)*int64(item.amount)), x+210, y+5, shopMutedColor)
	w.drawTinyButtonAt(screen, shopRowButtonBounds(x, y+4, width, 2), "-", true)
	w.drawTinyButtonAt(screen, shopRowButtonBounds(x, y+4, width, 1), "+", item.amount < item.max)
	w.drawTinyButtonAt(screen, shopRowButtonBounds(x, y+4, width, 0), "x", true)
}

func (w *ShopWindow) drawBuyRows(screen *render.Image, ctx Context, assets AssetRenderer) {
	dx, dy, dw, dh := w.dropBounds()
	DrawSurface(screen, dx, dy, dw, dh, PanelBodyColor, WindowBorderColor)
	if len(w.buyItems) == 0 {
		render.DebugPrintAtColor(screen, "No items", dx+112, dy+72, shopMutedColor)
		return
	}
	mx, my := -1, -1
	if ctx.Input != nil {
		mx, my = ctx.Input.MouseX, ctx.Input.MouseY
	}
	for row, item := range w.visibleBuyItems() {
		x, y, width, height := w.buyRowBounds(row)
		fill := PanelAltColor
		if pointInRect(mx, my, x, y, width, height) {
			fill = shopHoverColor
		}
		DrawSurface(screen, x, y, width, height, fill, WindowBorderColor)
		if assets != nil {
			assets.DrawInventoryItemIcon(screen, ctx.Resources, session.InventoryItem{ItemID: item.ItemID, Identified: true, Amount: 1}, x+3, y+2)
		}
		name := inventoryItemDisplayName(ctx.Resources, session.InventoryItem{ItemID: item.ItemID, Identified: true})
		price := int64(item.DiscountPrice)
		if price == 0 {
			price = int64(item.Price)
		}
		amount := w.buyAmount(item.ItemID)
		render.DebugPrintAtColor(screen, trimRunes(name, 20), x+32, y+5, shopTextColor)
		render.DebugPrintAtColor(screen, formatHUDNumber(price), x+176, y+5, shopMutedColor)
		if amount > 0 {
			render.DebugPrintAtColor(screen, fmt.Sprintf("x%d", amount), x+244, y+5, shopGoodColor)
		}
		w.drawTinyButtonAt(screen, shopRowButtonBounds(x, y+6, width, 1), "-", amount > 0)
		w.drawTinyButtonAt(screen, shopRowButtonBounds(x, y+6, width, 0), "+", true)
	}
	w.drawBuyScrollBar(screen)
}

func (w *ShopWindow) drawBuyScrollBar(screen *render.Image) {
	total := len(w.buyItems)
	visible := visibleBuyRows()
	if total <= visible {
		return
	}
	dx, dy, dw, _ := w.dropBounds()
	trackX := dx + dw - 8
	trackY := dy + 5
	trackH := visible*shopBuyRowH - 3
	render.DrawRect(screen, float64(trackX), float64(trackY), 4, float64(trackH), PanelAltColor)
	maxScroll := maxInt(1, total-visible)
	thumbH := maxInt(18, trackH*visible/total)
	thumbTravel := trackH - thumbH
	thumbY := trackY + thumbTravel*w.buyScroll/maxScroll
	render.DrawRect(screen, float64(trackX), float64(thumbY), 4, float64(thumbH), shopMutedColor)
}

func (w *ShopWindow) buyAmount(itemID uint16) uint16 {
	for _, item := range w.buyCart {
		if item.item.ItemID == itemID {
			return item.amount
		}
	}
	return 0
}

func (w *ShopWindow) handleCartButton(ctx Context, row int, mx, my int) bool {
	if row >= len(w.cart) {
		return false
	}
	x, y, width, _ := w.cartRowBounds(row)
	minus := shopRowButtonBounds(x, y+4, width, 2)
	plus := shopRowButtonBounds(x, y+4, width, 1)
	remove := shopRowButtonBounds(x, y+4, width, 0)
	switch {
	case pointInRect(mx, my, minus[0], minus[1], minus[2], minus[3]):
		if w.cart[row].amount > 1 {
			w.cart[row].amount--
		} else {
			w.cart = append(w.cart[:row], w.cart[row+1:]...)
		}
		return true
	case pointInRect(mx, my, plus[0], plus[1], plus[2], plus[3]):
		if w.cart[row].amount < w.cart[row].max {
			w.cart[row].amount++
		}
		return true
	case pointInRect(mx, my, remove[0], remove[1], remove[2], remove[3]):
		w.cart = append(w.cart[:row], w.cart[row+1:]...)
		return true
	default:
		return false
	}
}

func (w *ShopWindow) drawButton(screen *render.Image, x, y, width, height int, label string, enabled bool) {
	fill := shopButtonColor
	text := shopTextColor
	if !enabled {
		fill = DisabledColor
		text = shopMutedColor
	}
	DrawButtonLabel(screen, x, y, width, height, label, fill, text)
}

func (w *ShopWindow) drawTinyButton(screen *render.Image, x, y int, label string, enabled bool) {
	fill := shopButtonColor
	text := shopTextColor
	if !enabled {
		fill = DisabledColor
		text = shopMutedColor
	}
	switch label {
	case "+":
		DrawPlusButton(screen, x, y, fill, text)
	case "-":
		DrawMinusButton(screen, x, y, fill, text)
	case "x":
		DrawButtonSurface(screen, x, y, IconButtonSize, IconButtonSize, fill)
		DrawCloseGlyph(screen, x, y, IconButtonSize, IconButtonSize, text)
	default:
		w.drawButton(screen, x, y, IconButtonSize, IconButtonSize, label, enabled)
	}
}

func (w *ShopWindow) drawTinyButtonAt(screen *render.Image, bounds [4]int, label string, enabled bool) {
	w.drawTinyButton(screen, bounds[0], bounds[1], label, enabled)
}

func shopRowButtonBounds(rowX, y, rowWidth, fromRight int) [4]int {
	x := rowX + rowWidth - shopRowRightPad - IconButtonSize - fromRight*(IconButtonSize+shopRowButtonGap)
	return [4]int{x, y, IconButtonSize, IconButtonSize}
}

func (w *ShopWindow) total() int64 {
	var total int64
	if w.mode == shopModeBuy {
		for _, item := range w.buyCart {
			price := item.item.DiscountPrice
			if price == 0 {
				price = item.item.Price
			}
			total += int64(price) * int64(item.amount)
		}
		return total
	}
	for _, item := range w.cart {
		total += int64(item.over) * int64(item.amount)
	}
	return total
}

func clampShopWindowInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
