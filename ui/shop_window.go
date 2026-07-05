package ui

import (
	"fmt"
	"image"
	"log"

	"github.com/gogpu/ui/core/datatable"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	shopBuyListWindowW   = 420
	shopBuyListWindowH   = 416
	shopBuyCartWindowW   = 420
	shopBuyCartWindowH   = 184
	shopBuyCartFooterH   = 42
	shopDataTableHeaderH = 36
	shopWindowTitleH     = 28
	shopWindowPad        = 10
	shopBuyRowH          = 31

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
	shopButtonColor = ButtonColor
)

type ShopWindow struct {
	dealOpen        bool
	dealNPCID       uint32
	open            bool
	mode            int
	x               int
	y               int
	sellable        map[uint16]network.ShopSellItem
	cart            []shopSellCartItem
	buyItems        []network.ShopBuyItem
	buyCart         []shopBuyCartItem
	buyWindow       WindowState
	buyCartWindow   WindowState
	buySelectedRow  int
	buyScrollY      state.Signal[float32]
	buyCartScrollY  state.Signal[float32]
	buyPressRow     int
	buyPressCart    bool
	buyPressX       int
	buyPressY       int
	buyDraggingItem bool
	buyIcons        map[shopItemIconKey]image.Image
	buyIconMiss     map[shopItemIconKey]struct{}
	status          string
	statusGood      bool
	closePacketSent bool
}

type shopSellCartItem struct {
	item   session.InventoryItem
	over   uint32
	amount uint16
	max    uint16
}

type shopBuyCartItem struct {
	item   network.ShopBuyItem
	amount uint16
}

type shopTableRow struct {
	name   string
	price  string
	amount string
	icon   image.Image
}

type shopItemIconKey struct {
	itemID     uint16
	identified bool
}

func (w *ShopWindow) OpenDeal(selection network.ShopDealSelection) {
	w.dealOpen = true
	w.dealNPCID = selection.NPCID
}

func (w *ShopWindow) OpenSell(list []network.ShopSellItem, ctx Context) {
	w.dealOpen = false
	w.open = true
	w.mode = shopModeSell
	w.ensureBuyPosition(ctx)
	w.sellable = make(map[uint16]network.ShopSellItem, len(list))
	for _, item := range list {
		w.sellable[item.Index] = item
	}
	w.cart = nil
	w.buyItems = nil
	w.buyCart = nil
	w.buySelectedRow = -1
	w.buyPressRow = -1
	w.buyPressCart = false
	w.buyDraggingItem = false
	w.ensureBuyScrollSignal().Set(0)
	w.ensureBuyCartScrollSignal().Set(0)
	w.status = "Drag items here to sell"
	w.statusGood = true
	w.closePacketSent = false
	w.openBuyWindow(ctx)
}

func (w *ShopWindow) OpenBuy(list []network.ShopBuyItem, ctx Context) {
	w.dealOpen = false
	w.open = true
	w.mode = shopModeBuy
	w.ensureBuyPosition(ctx)
	w.buyItems = append(w.buyItems[:0], list...)
	w.buyCart = nil
	w.buySelectedRow = -1
	w.buyPressRow = -1
	w.buyPressCart = false
	w.buyDraggingItem = false
	w.ensureBuyScrollSignal().Set(0)
	w.ensureBuyCartScrollSignal().Set(0)
	w.sellable = nil
	w.cart = nil
	w.status = "Select items to buy"
	w.statusGood = true
	w.closePacketSent = false
	w.openBuyWindow(ctx)
}

func (w *ShopWindow) ApplyResult(ctx Context, result network.ShopResult) {
	if !result.Sell {
		if result.Result == 0 {
			w.status = "Deal completed"
			w.statusGood = true
			w.open = false
			w.mode = shopModeNone
			w.buyCart = nil
			w.buyItems = nil
			w.closePacketSent = true
			w.closeBuyWindows(ctx)
			return
		}
		w.status = fmt.Sprintf("Buy failed result=%d", result.Result)
		w.statusGood = result.Result == 0
		w.refreshBuyWindow(ctx)
		return
	}
	if result.Result == 0 {
		w.status = "Deal completed"
		w.statusGood = true
		w.open = false
		w.mode = shopModeNone
		w.cart = nil
		w.sellable = nil
		w.closePacketSent = true
		w.closeBuyWindows(ctx)
		return
	}
	w.status = "Sell failed"
	w.statusGood = false
}

func (w *ShopWindow) Update(ctx Context) bool {
	if ctx.Input == nil {
		return false
	}
	if w.dealOpen {
		return w.updateDeal(ctx)
	}
	if !w.open {
		return false
	}
	if w.mode == shopModeBuy || w.mode == shopModeSell {
		return w.updateBuyWindow(ctx)
	}
	return false
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
	if w.mode == shopModeBuy || w.mode == shopModeSell {
		w.buyWindow.Publish(ctx)
		w.buyCartWindow.Publish(ctx)
		return
	}
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
	if (w.mode == shopModeBuy || w.mode == shopModeSell) && (pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.buyWindow.x, w.buyWindow.y, shopBuyListWindowW, shopBuyListWindowH) ||
		pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.buyCartWindow.x, w.buyCartWindow.y, shopBuyCartWindowW, shopBuyCartWindowH)) {
		return CursorActionClick, true
	}
	return 0, false
}

func (w *ShopWindow) ensureBuyWindow() {
	if w.buyWindow.width == 0 {
		w.buyWindow = NewWindowState(shopBuyListWindowW, shopBuyListWindowH)
		w.buyWindow.SetCloseOnEscape(false)
	}
	if w.buyCartWindow.width == 0 {
		w.buyCartWindow = NewWindowState(shopBuyCartWindowW, shopBuyCartWindowH)
		w.buyCartWindow.SetCloseOnEscape(false)
	}
}

func (w *ShopWindow) openBuyWindow(ctx Context) {
	w.ensureBuyWindow()
	w.buyWindow.OpenAt(w.x, w.y, w.buyListWidgetTree(ctx))
	w.buyCartWindow.OpenAt(w.x+shopBuyListWindowW+20, w.y, w.buyCartWidgetTree(ctx))
	w.buyWindow.Publish(ctx)
	w.buyCartWindow.Publish(ctx)
}

func (w *ShopWindow) refreshBuyWindow(ctx Context) {
	if (w.mode != shopModeBuy && w.mode != shopModeSell) || !w.open {
		w.closeBuyWindows(ctx)
		return
	}
	w.ensureBuyWindow()
	if !w.buyWindow.IsOpen() || !w.buyCartWindow.IsOpen() {
		w.openBuyWindow(ctx)
		return
	}
	w.buyWindow.SetContent(w.buyListWidgetTree(ctx))
	w.buyCartWindow.SetContent(w.buyCartWidgetTree(ctx))
	w.buyWindow.Publish(ctx)
	w.buyCartWindow.Publish(ctx)
}

func (w *ShopWindow) closeBuyWindows(ctx Context) {
	if w.buyWindow.width != 0 {
		w.buyWindow.Close()
		w.buyWindow.Unpublish(ctx)
	}
	if w.buyCartWindow.width != 0 {
		w.buyCartWindow.Close()
		w.buyCartWindow.Unpublish(ctx)
	}
}

func (w *ShopWindow) buyListWidgetTree(ctx Context) widget.Widget {
	title := "Shop Items"
	if w.mode == shopModeSell {
		title = "Sell Items"
	}
	return Window(
		Title(title),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
		}),
		Size(shopBuyListWindowW, shopBuyListWindowH),
		Content(
			primitives.Box(
				primitives.Box(w.buyTableWidget(ctx)).
					Height(shopBuyListWindowH-ROWindowTitleHeight).
					Background(rotheme.Default.Colors.PanelBody),
			).Gap(0),
		),
	)
}

func (w *ShopWindow) buyCartWidgetTree(ctx Context) widget.Widget {
	title := "Buying Items"
	action := "Buy"
	disabled := len(w.buyCart) == 0
	if w.mode == shopModeSell {
		title = "Selling Items"
		action = "Sell"
		disabled = len(w.cart) == 0
	}
	return Window(
		Title(title),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
		}),
		Size(shopBuyCartWindowW, shopBuyCartWindowH),
		FooterHeight(shopBuyCartFooterH),
		FooterPadding(10),
		Content(
			primitives.Box(w.buyCartTableWidget(ctx)).
				Height(shopBuyCartWindowH-ROWindowTitleHeight-shopBuyCartFooterH).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.HBox(
				rotheme.Text(fmt.Sprintf("Total: %s z", formatHUDNumber(w.total()))),
				primitives.Expanded(primitives.Box()),
				rotheme.ButtonDisabled(action, disabled, func() {
					w.submit(ctx)
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

func (w *ShopWindow) buyTableWidget(ctx Context) *datatable.Widget {
	rows := w.buyTableRows(ctx)
	if w.mode == shopModeSell {
		rows = w.sellTableRows(ctx)
	}
	return w.shopTableWidget(rows, false, w.ensureBuyScrollSignal(), true, func(row int) {
		w.buySelectedRow = row
		w.refreshBuyWindow(ctx)
	})
}

func (w *ShopWindow) buyCartTableWidget(ctx Context) *datatable.Widget {
	rows := w.buyCartTableRows(ctx)
	if w.mode == shopModeSell {
		rows = w.sellCartTableRows(ctx)
	}
	return w.shopTableWidget(rows, true, w.ensureBuyCartScrollSignal(), false, nil)
}

func (w *ShopWindow) shopTableWidget(rows []shopTableRow, amountColumn bool, scroll state.Signal[float32], selectable bool, onSelect func(int)) *datatable.Widget {
	columns := []datatable.Column{
		{Key: "item", Title: "Item", Width: 296},
		{Key: "price", Title: "Price", Width: 124, Align: widget.TextAlignRight},
	}
	if amountColumn {
		columns = []datatable.Column{
			{Key: "item", Title: "Item", Width: 250},
			{Key: "price", Title: "Price", Width: 104, Align: widget.TextAlignRight},
			{Key: "amount", Title: "Qty", Width: 66, Align: widget.TextAlignCenter},
		}
	}
	icons := make([]image.Image, len(rows))
	for i, row := range rows {
		icons[i] = row.icon
	}
	options := []datatable.Option{
		datatable.Columns(columns),
		datatable.RowCount(len(rows)),
		datatable.RowHeight(shopBuyRowH - 3),
		datatable.ScrollYSignal(scroll),
		datatable.PainterOpt(shopBuyTablePainter{
			icons: icons,
		}),
		datatable.CellValue(func(row int, col string) string {
			if row < 0 || row >= len(rows) {
				return ""
			}
			switch col {
			case "item":
				return rows[row].name
			case "price":
				return rows[row].price
			case "amount":
				return rows[row].amount
			}
			return ""
		}),
	}
	if selectable {
		options = append(options,
			datatable.SelectionModeOpt(datatable.SelectionSingle),
			datatable.SelectedRow(w.buySelectedRow),
			datatable.OnRowSelect(func(row int) {
				if row < 0 || row >= len(rows) {
					return
				}
				if onSelect != nil {
					onSelect(row)
				}
			}),
		)
	}
	return datatable.New(options...)
}

func (w *ShopWindow) buyTableRows(ctx Context) []shopTableRow {
	rows := make([]shopTableRow, len(w.buyItems))
	for i, item := range w.buyItems {
		rows[i] = shopTableRow{
			name:  inventoryItemDisplayName(ctx.Resources, session.InventoryItem{ItemID: item.ItemID, Type: item.Type, Identified: true}),
			price: formatHUDNumber(int64(shopBuyItemPrice(item))) + " z",
			icon:  w.shopItemIconImage(ctx.Resources, item.ItemID),
		}
	}
	return rows
}

func (w *ShopWindow) buyCartTableRows(ctx Context) []shopTableRow {
	rows := make([]shopTableRow, len(w.buyCart))
	for i, item := range w.buyCart {
		rows[i] = shopTableRow{
			name:   inventoryItemDisplayName(ctx.Resources, session.InventoryItem{ItemID: item.item.ItemID, Type: item.item.Type, Identified: true}),
			price:  formatHUDNumber(int64(shopBuyItemPrice(item.item)) * int64(item.amount)),
			amount: fmt.Sprintf("x%d", item.amount),
			icon:   w.shopItemIconImage(ctx.Resources, item.item.ItemID),
		}
	}
	return rows
}

func (w *ShopWindow) sellTableRows(ctx Context) []shopTableRow {
	items := w.sellAvailableItems(ctx)
	rows := make([]shopTableRow, len(items))
	for i, item := range items {
		sell, ok := w.sellable[item.Index]
		if !ok {
			continue
		}
		rows[i] = shopTableRow{
			name:  inventoryItemDisplayName(ctx.Resources, item),
			price: formatHUDNumber(int64(shopSellItemPrice(sell))) + " z",
			icon:  w.shopItemIconImage(ctx.Resources, item.ItemID),
		}
	}
	return rows
}

func (w *ShopWindow) sellCartTableRows(ctx Context) []shopTableRow {
	rows := make([]shopTableRow, len(w.cart))
	for i, item := range w.cart {
		rows[i] = shopTableRow{
			name:   inventoryItemDisplayName(ctx.Resources, item.item),
			price:  formatHUDNumber(int64(item.over) * int64(item.amount)),
			amount: fmt.Sprintf("x%d", item.amount),
			icon:   w.shopItemIconImage(ctx.Resources, item.item.ItemID),
		}
	}
	return rows
}

func (w *ShopWindow) ensureBuyScrollSignal() state.Signal[float32] {
	if w.buyScrollY == nil {
		w.buyScrollY = state.NewSignal[float32](0)
	}
	return w.buyScrollY
}

func (w *ShopWindow) ensureBuyCartScrollSignal() state.Signal[float32] {
	if w.buyCartScrollY == nil {
		w.buyCartScrollY = state.NewSignal[float32](0)
	}
	return w.buyCartScrollY
}

func (w *ShopWindow) updateBuyWindow(ctx Context) bool {
	w.ensureBuyWindow()
	if !w.buyWindow.IsOpen() || !w.buyCartWindow.IsOpen() {
		w.openBuyWindow(ctx)
	}
	w.x, w.y = w.buyWindow.x, w.buyWindow.y
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.cancel(ctx)
		return true
	}
	if w.handleBuyPointer(ctx) {
		return true
	}
	inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.buyWindow.x, w.buyWindow.y, shopBuyListWindowW, shopBuyListWindowH) ||
		pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.buyCartWindow.x, w.buyCartWindow.y, shopBuyCartWindowW, shopBuyCartWindowH)
	consumed := w.buyWindow.Update(ctx)
	if w.buyCartWindow.Update(ctx) {
		consumed = true
	}
	w.x, w.y = w.buyWindow.x, w.buyWindow.y
	if !w.buyWindow.IsOpen() || !w.buyCartWindow.IsOpen() {
		w.cancel(ctx)
		return true
	}
	w.buyWindow.Publish(ctx)
	w.buyCartWindow.Publish(ctx)
	return consumed || inside
}

func (w *ShopWindow) handleBuyPointer(ctx Context) bool {
	if ctx.Input.MouseJustPressed(render.MouseButtonLeft) {
		if row, ok := w.buyShopRowAt(ctx, ctx.Input.MouseX, ctx.Input.MouseY); ok {
			w.buySelectedRow = row
			w.buyPressRow = row
			w.buyPressCart = false
			w.buyPressX = ctx.Input.MouseX
			w.buyPressY = ctx.Input.MouseY
			w.buyDraggingItem = false
			w.refreshBuyWindow(ctx)
			return true
		}
		if row, ok := w.buyCartRowAt(ctx.Input.MouseX, ctx.Input.MouseY); ok {
			w.buyPressRow = row
			w.buyPressCart = true
			w.buyPressX = ctx.Input.MouseX
			w.buyPressY = ctx.Input.MouseY
			w.buyDraggingItem = false
			w.refreshBuyWindow(ctx)
			return true
		}
	}
	if w.buyPressRow >= 0 && ctx.Input.MousePressed(render.MouseButtonLeft) {
		if absShopWindowInt(ctx.Input.MouseX-w.buyPressX) > 4 || absShopWindowInt(ctx.Input.MouseY-w.buyPressY) > 4 {
			w.buyDraggingItem = true
			return true
		}
	}
	if w.buyPressRow >= 0 && ctx.Input.MouseJustReleased(render.MouseButtonLeft) {
		row := w.buyPressRow
		fromCart := w.buyPressCart
		dragging := w.buyDraggingItem
		w.buyPressRow = -1
		w.buyPressCart = false
		w.buyDraggingItem = false
		if fromCart {
			if w.mode == shopModeSell {
				if row < 0 || row >= len(w.cart) {
					return true
				}
				if dragging && w.buyShopDropTarget(ctx.Input.MouseX, ctx.Input.MouseY) {
					w.decrementSellCartRow(row)
					w.refreshBuyWindow(ctx)
				}
				return true
			}
			if row < 0 || row >= len(w.buyCart) {
				return true
			}
			if dragging && w.buyShopDropTarget(ctx.Input.MouseX, ctx.Input.MouseY) {
				w.decrementBuyCartRow(row)
				w.refreshBuyWindow(ctx)
			}
			return true
		}
		if w.mode == shopModeSell {
			items := w.sellAvailableItems(ctx)
			if row < 0 || row >= len(items) {
				return true
			}
			if dragging && w.buyCartDropTarget(ctx.Input.MouseX, ctx.Input.MouseY) {
				if sell, ok := w.sellable[items[row].Index]; ok {
					w.addCartItem(items[row], sell)
					w.refreshBuyWindow(ctx)
				}
			}
			return true
		}
		if row < 0 || row >= len(w.buyItems) {
			return true
		}
		if dragging && w.buyCartDropTarget(ctx.Input.MouseX, ctx.Input.MouseY) {
			w.addBuyItem(w.buyItems[row])
			w.refreshBuyWindow(ctx)
			return true
		}
		return true
	}
	return false
}

func (w *ShopWindow) buyShopRowAt(ctx Context, mx, my int) (int, bool) {
	tableX, tableY, tableW, tableH := w.buyShopTableBounds()
	rowCount := len(w.buyItems)
	if w.mode == shopModeSell {
		rowCount = len(w.sellAvailableItems(ctx))
	}
	return tableRowAt(mx, my, tableX, tableY, tableW, tableH, rowCount, shopBuyRowH-3, w.ensureBuyScrollSignal().Get())
}

func (w *ShopWindow) buyCartRowAt(mx, my int) (int, bool) {
	tableX, tableY, tableW, tableH := w.buyCartTableBounds()
	rowCount := len(w.buyCart)
	if w.mode == shopModeSell {
		rowCount = len(w.cart)
	}
	return tableRowAt(mx, my, tableX, tableY, tableW, tableH, rowCount, shopBuyRowH-3, w.ensureBuyCartScrollSignal().Get())
}

func (w *ShopWindow) buyCartDropTarget(mx, my int) bool {
	return pointInRect(mx, my, w.buyCartWindow.x, w.buyCartWindow.y, shopBuyCartWindowW, shopBuyCartWindowH)
}

func (w *ShopWindow) buyShopDropTarget(mx, my int) bool {
	return pointInRect(mx, my, w.buyWindow.x, w.buyWindow.y, shopBuyListWindowW, shopBuyListWindowH)
}

func (w *ShopWindow) buyShopTableBounds() (int, int, int, int) {
	return w.buyWindow.x, w.buyWindow.y + ROWindowTitleHeight, shopBuyListWindowW, shopBuyListWindowH - ROWindowTitleHeight
}

func (w *ShopWindow) buyCartTableBounds() (int, int, int, int) {
	return w.buyCartWindow.x, w.buyCartWindow.y + ROWindowTitleHeight, shopBuyCartWindowW, shopBuyCartWindowH - ROWindowTitleHeight - shopBuyCartFooterH
}

func tableRowAt(mx, my, tableX, tableY, tableW, tableH, rowCount, rowHeight int, scrollY float32) (int, bool) {
	if !pointInRect(mx, my, tableX, tableY+shopDataTableHeaderH, tableW, tableH-shopDataTableHeaderH) {
		return 0, false
	}
	localY := float32(my-tableY) - shopDataTableHeaderH + scrollY
	row := int(localY / float32(rowHeight))
	if row < 0 || row >= rowCount {
		return 0, false
	}
	return row, true
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

func shopSellItemPrice(item network.ShopSellItem) uint32 {
	if item.OverchargePrice != 0 {
		return item.OverchargePrice
	}
	return item.Price
}

func (w *ShopWindow) sellAvailableItems(ctx Context) []session.InventoryItem {
	if ctx.Session == nil || len(w.sellable) == 0 {
		return nil
	}
	items := make([]session.InventoryItem, 0, len(w.sellable))
	for _, item := range ctx.Session.Inventory.Items {
		if _, ok := w.sellable[item.Index]; ok {
			items = append(items, item)
		}
	}
	return items
}

type shopBuyTablePainter struct {
	datatable.DefaultPainter
	icons []image.Image
}

func (p shopBuyTablePainter) PaintHeader(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderPaintState) {
	if bounds.IsEmpty() {
		return
	}
	canvas.DrawRect(bounds, rotheme.Default.Colors.PanelBody)
}

func (p shopBuyTablePainter) PaintHeaderCell(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderCellPaintState) {
}

func (p shopBuyTablePainter) PaintRow(canvas widget.Canvas, s datatable.RowPaintState) {
	fill := widget.RGBA8(246, 249, 253, 255)
	if s.RowIndex%2 == 1 {
		fill = rotheme.Default.Colors.PanelBody
	}
	if s.Hovered {
		fill = rotheme.Default.Colors.ButtonHover
	}
	if s.Selected {
		fill = rotheme.Default.Colors.ButtonDown
	}
	canvas.DrawRect(s.Bounds, fill)
}

func (p shopBuyTablePainter) PaintCell(canvas widget.Canvas, s datatable.CellPaintState) {
	color := rotheme.Default.Colors.Text
	if s.ColIndex == 1 {
		color = rotheme.Default.Colors.MutedText
	}
	textBounds := geometry.NewRect(s.Bounds.Min.X+4, s.Bounds.Min.Y+4, s.Bounds.Width()-8, s.Bounds.Height()-8)
	if s.ColIndex == 0 {
		if s.RowIndex >= 0 && s.RowIndex < len(p.icons) && p.icons[s.RowIndex] != nil {
			icon := p.icons[s.RowIndex]
			iconBounds := icon.Bounds()
			iconW := float32(iconBounds.Dx())
			iconH := float32(iconBounds.Dy())
			canvas.DrawImage(icon, geometry.Pt(s.Bounds.Min.X+4, s.Bounds.Min.Y+(s.Bounds.Height()-iconH)/2))
			textBounds = geometry.NewRect(s.Bounds.Min.X+iconW+9, s.Bounds.Min.Y+4, s.Bounds.Width()-iconW-13, s.Bounds.Height()-8)
		}
	}
	if s.ColIndex == 2 && s.Value != "" {
		color = widget.RGBA8(54, 128, 76, 255)
	}
	rotheme.DrawText(canvas, s.Value, textBounds, rotheme.Default.Typography.TextSize, color, false, s.Align)
}

func (p shopBuyTablePainter) PaintEmptyState(canvas widget.Canvas, bounds geometry.Rect) {
	rotheme.DrawText(canvas, "No items", bounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignCenter)
}

func (w *ShopWindow) addCartItem(item session.InventoryItem, sell network.ShopSellItem) {
	maxAmount := uint16(maxInt(1, item.Amount))
	for i := range w.cart {
		if w.cart[i].item.Index == item.Index {
			if w.cart[i].amount < w.cart[i].max {
				w.cart[i].amount++
			}
			return
		}
	}
	w.cart = append(w.cart, shopSellCartItem{
		item:   item,
		over:   sell.OverchargePrice,
		amount: 1,
		max:    maxAmount,
	})
}

func (w *ShopWindow) addBuyItem(item network.ShopBuyItem) {
	for i := range w.buyCart {
		if w.buyCart[i].item.ItemID == item.ItemID {
			w.buyCart[i].amount++
			return
		}
	}
	w.buyCart = append(w.buyCart, shopBuyCartItem{item: item, amount: 1})
}

func (w *ShopWindow) submit(ctx Context) {
	if w.mode == shopModeBuy {
		w.submitBuy(ctx)
		return
	}
	if len(w.cart) == 0 {
		w.status = "No items selected"
		w.statusGood = false
		return
	}
	if ctx.Network == nil {
		w.status = "Not connected"
		w.statusGood = false
		return
	}
	items := make([]network.SellRequestItem, 0, len(w.cart))
	for _, item := range w.cart {
		items = append(items, network.SellRequestItem{Index: item.item.Index, Amount: item.amount})
	}
	if err := ctx.Network.SendShopSellItems(items); err != nil {
		w.status = err.Error()
		w.statusGood = false
		return
	}
	w.closePacketSent = true
	w.status = "Sell requested"
	w.statusGood = true
}

func (w *ShopWindow) submitBuy(ctx Context) {
	if len(w.buyCart) == 0 {
		w.status = "No items selected"
		w.statusGood = false
		return
	}
	if ctx.Session != nil && w.total() > ctx.Session.Inventory.Zeny {
		w.status = "Not enough zeny"
		w.statusGood = false
		return
	}
	if ctx.Network == nil {
		w.status = "Not connected"
		w.statusGood = false
		return
	}
	items := make([]network.BuyRequestItem, 0, len(w.buyCart))
	for _, item := range w.buyCart {
		items = append(items, network.BuyRequestItem{ItemID: item.item.ItemID, Amount: item.amount})
	}
	if err := ctx.Network.SendShopBuyItems(items); err != nil {
		w.status = err.Error()
		w.statusGood = false
		return
	}
	w.closePacketSent = true
	w.status = "Buy requested"
	w.statusGood = true
}

func (w *ShopWindow) decrementBuyCartRow(row int) {
	if row < 0 || row >= len(w.buyCart) {
		return
	}
	if w.buyCart[row].amount > 1 {
		w.buyCart[row].amount--
		return
	}
	w.buyCart = append(w.buyCart[:row], w.buyCart[row+1:]...)
}

func (w *ShopWindow) decrementSellCartRow(row int) {
	if row < 0 || row >= len(w.cart) {
		return
	}
	if w.cart[row].amount > 1 {
		w.cart[row].amount--
		return
	}
	w.cart = append(w.cart[:row], w.cart[row+1:]...)
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
	w.closeBuyWindows(ctx)
}

func (w *ShopWindow) sendDealSelection(ctx Context, dealType uint8) {
	if ctx.Network == nil {
		w.status = "Not connected"
		w.statusGood = false
		return
	}
	if err := ctx.Network.SendShopDealSelection(w.dealNPCID, dealType); err != nil {
		w.status = err.Error()
		w.statusGood = false
		return
	}
	w.dealOpen = false
	if dealType == 1 {
		w.status = "Waiting for sell list"
	} else {
		w.status = "Buy requested"
	}
	w.statusGood = true
}

func (w *ShopWindow) ensureBuyPosition(ctx Context) {
	width, _ := ctx.ScreenSize()
	totalWidth := shopBuyListWindowW + 20 + shopBuyCartWindowW
	w.x = maxInt(8, (width-totalWidth)/2)
	w.y = 120
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

func (w *ShopWindow) total() int64 {
	var total int64
	if w.mode == shopModeBuy {
		for _, item := range w.buyCart {
			total += int64(shopBuyItemPrice(item.item)) * int64(item.amount)
		}
		return total
	}
	for _, item := range w.cart {
		total += int64(item.over) * int64(item.amount)
	}
	return total
}

func absShopWindowInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
