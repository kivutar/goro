package ui

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"strconv"
	"strings"
	"time"

	"github.com/gogpu/ui/core/datatable"
	"github.com/gogpu/ui/core/textfield"
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
	vendingWindowW      = 420
	vendingSetupRows    = 7
	vendingBuyRows      = 7
	vendingCartRows     = 4
	vendingFooterH      = 42
	vendingNameH        = 30
	vendingWindowGap    = 20
	vendingDefaultPrice = 1
)

const (
	vendingModeNone = iota
	vendingModeSetup
	vendingModeBuy
	vendingModeOwn
)

type VendingWindow struct {
	mode        int
	maxItems    int
	ownerAID    uint32
	shopName    string
	leftWindow  Window
	rightWindow Window

	setupItems []vendingSetupItem
	buyItems   []network.VendingItem
	buyCart    []vendingBuyCartItem
	ownItems   []network.VendingItem

	selectedSetup int
	selectedBuy   int
	selectedCart  int
	priceInput    string
	nameField     *textfield.Widget
	priceField    *textfield.Widget

	leftScroll  state.Signal[float32]
	rightScroll state.Signal[float32]

	pressRow      int
	pressRight    bool
	pressX        int
	pressY        int
	draggingItem  bool
	lastClickAt   time.Time
	lastClickRow  int
	lastClickSide bool

	icons    map[shopItemIconKey]image.Image
	iconMiss map[shopItemIconKey]struct{}
}

type vendingSetupItem struct {
	item   session.InventoryItem
	amount uint16
	price  uint32
}

type vendingBuyCartItem struct {
	item   network.VendingItem
	amount uint16
}

func (w *VendingWindow) OpenSetup(ctx Context, req network.VendingOpenRequest) {
	w.mode = vendingModeSetup
	w.maxItems = int(req.MaxItems)
	if w.maxItems <= 0 {
		w.maxItems = 1
	}
	w.ownerAID = 0
	w.setupItems = nil
	w.buyItems = nil
	w.buyCart = nil
	w.ownItems = nil
	w.selectedSetup = -1
	w.selectedBuy = -1
	w.selectedCart = -1
	w.shopName = ""
	w.priceInput = ""
	w.nameField = nil
	w.priceField = nil
	w.ensurePosition(ctx)
	w.refresh(ctx)
}

func (w *VendingWindow) OpenBuy(ctx Context, list network.VendingItemList) {
	w.mode = vendingModeBuy
	w.ownerAID = list.OwnerAID
	w.buyItems = append(w.buyItems[:0], list.Items...)
	w.buyCart = nil
	w.setupItems = nil
	w.ownItems = nil
	w.selectedBuy = -1
	w.selectedCart = -1
	w.ensurePosition(ctx)
	w.refresh(ctx)
}

func (w *VendingWindow) ApplyOwnList(ctx Context, list network.VendingItemList) {
	w.mode = vendingModeOwn
	w.ownerAID = list.OwnerAID
	w.ownItems = append(w.ownItems[:0], list.Items...)
	w.buyItems = nil
	w.buyCart = nil
	w.setupItems = nil
	w.closeRight(ctx)
	w.refresh(ctx)
}

func (w *VendingWindow) ApplyPurchaseResult(ctx Context, result network.VendingPurchaseResult) {
	if result.Result == 0 {
		w.mode = vendingModeNone
		w.buyItems = nil
		w.buyCart = nil
		w.closeBoth(ctx)
		return
	}
	glog.Warnf("vending purchase failed index=%d amount=%d result=%d", result.Index, result.Amount, result.Result)
	w.refresh(ctx)
}

func (w *VendingWindow) ApplySoldItem(ctx Context, sold network.VendingSoldItem) {
	for i := range w.ownItems {
		if w.ownItems[i].Index != sold.Index {
			continue
		}
		if w.ownItems[i].Amount > sold.Amount {
			w.ownItems[i].Amount -= sold.Amount
		} else {
			w.ownItems = append(w.ownItems[:i], w.ownItems[i+1:]...)
		}
		break
	}
	w.refresh(ctx)
}

func (w *VendingWindow) Update(ctx Context, itemInfo *ItemInfoWindow) bool {
	if ctx.Input == nil || w.mode == vendingModeNone {
		return false
	}
	if w.leftWindow.Update(ctx) {
		w.leftWindow.Publish(ctx)
	}
	if w.rightWindow.Update(ctx) {
		w.rightWindow.Publish(ctx)
	}
	if w.handlePointer(ctx, itemInfo) {
		return true
	}
	inside := w.inside(ctx.Input.MouseX, ctx.Input.MouseY)
	w.leftWindow.Publish(ctx)
	w.rightWindow.Publish(ctx)
	return inside
}

func (w *VendingWindow) Draw(screen *render.Frame, ctx Context, assets AssetProvider) {
	if w.mode == vendingModeNone {
		return
	}
	w.leftWindow.Publish(ctx)
	w.rightWindow.Publish(ctx)
}

func (w *VendingWindow) DrawDragGhost(screen *render.Frame, ctx Context, assets AssetProvider) {
	if !w.draggingItem || screen == nil || ctx.Input == nil || assets == nil {
		return
	}
	item, ok := w.dragItem(ctx)
	if !ok {
		return
	}
	assets.DrawInventoryItemIcon(screen, ctx.Resources, item, ctx.Input.MouseX-inventoryIconSize/2, ctx.Input.MouseY-inventoryIconSize/2)
}

func (w *VendingWindow) ensurePosition(ctx Context) {
	w.ensureWindows()
	screenW, screenH := ctx.ScreenSize()
	totalW := vendingWindowW*2 + vendingWindowGap
	leftX := maxInt(8, (screenW-totalW)/2)
	leftY := maxInt(8, (screenH-w.leftHeight())/2)
	w.leftWindow.SetAutoPosition(leftX, leftY)
	w.rightWindow.SetAutoPosition(leftX+vendingWindowW+vendingWindowGap, leftY)
}

func (w *VendingWindow) ensureWindows() {
	if w.leftWindow.width == 0 {
		w.leftWindow = NewWindow(vendingWindowW, w.leftHeight())
	}
	w.leftWindow.SetSize(vendingWindowW, w.leftHeight())
	if w.rightWindow.width == 0 {
		w.rightWindow = NewWindow(vendingWindowW, w.rightHeight())
	}
	w.rightWindow.SetSize(vendingWindowW, w.rightHeight())
}

func (w *VendingWindow) refresh(ctx Context) {
	w.ensureWindows()
	switch w.mode {
	case vendingModeSetup:
		w.leftWindow.OpenAt(w.leftWindow.x, w.leftWindow.y, w.availableCartTree(ctx))
		w.rightWindow.OpenAt(w.rightWindow.x, w.rightWindow.y, w.setupTree(ctx))
	case vendingModeBuy:
		w.leftWindow.OpenAt(w.leftWindow.x, w.leftWindow.y, w.vendorItemsTree(ctx))
		w.rightWindow.OpenAt(w.rightWindow.x, w.rightWindow.y, w.buyCartTree(ctx))
	case vendingModeOwn:
		w.leftWindow.OpenAt(w.leftWindow.x, w.leftWindow.y, w.ownShopTree(ctx))
		w.closeRight(ctx)
	}
	w.leftWindow.Publish(ctx)
	w.rightWindow.Publish(ctx)
}

func (w *VendingWindow) closeBoth(ctx Context) {
	w.closeRight(ctx)
	if w.leftWindow.width != 0 {
		w.leftWindow.Close()
		w.leftWindow.Unpublish(ctx)
	}
}

func (w *VendingWindow) closeRight(ctx Context) {
	if w.rightWindow.width != 0 {
		w.rightWindow.Close()
		w.rightWindow.Unpublish(ctx)
	}
}

func (w *VendingWindow) availableCartTree(ctx Context) widget.Widget {
	return Win(
		Title("Available Items for Vending"),
		CloseButton(true),
		OnClose(func() { w.cancel(ctx) }),
		Size(vendingWindowW, float32(w.leftHeight())),
		FooterHeight(vendingFooterH),
		Content(primitives.Box(w.cartItemsTable(ctx)).Height(float32(shopTableHeight(vendingSetupRows)))),
	)
}

func (w *VendingWindow) setupTree(ctx Context) widget.Widget {
	name := w.nameField
	if name == nil {
		name = rotheme.TextField(w.shopName, textfield.TypeText, func(value string) {
			w.shopName = value
		}, nil, textfield.MaxLength(36))
		w.nameField = name
	}
	price := w.priceField
	if price == nil {
		price = rotheme.TextField(w.priceInput, textfield.TypeText, func(value string) {
			w.setSelectedSetupPrice(value)
		}, nil, textfield.MaxLength(10))
		w.priceField = price
	}
	return Win(
		Title("Vending"),
		CloseButton(true),
		OnClose(func() { w.cancel(ctx) }),
		Size(vendingWindowW, float32(w.rightHeight())),
		FooterHeight(vendingFooterH),
		Content(
			primitives.Box(
				primitives.HBox(
					primitives.Box(rotheme.Text("Name")).Width(54).Height(vendingNameH),
					primitives.Box(name).Height(22).Width(vendingWindowW-72),
				).CrossAlign(primitives.CrossAxisCenter),
				primitives.Box(w.setupItemsTable(ctx)).Height(float32(shopTableHeight(w.setupRows()))),
			).PaddingXY(8, 4).Gap(4),
		),
		Footer(
			primitives.HBox(
				rotheme.Text("Price"),
				primitives.Box(price).Width(90).Height(22),
				primitives.Expanded(primitives.Box()),
				rotheme.ButtonDisabledFn("OK", func() bool {
					return len(w.setupItems) == 0 || w.currentShopName() == ""
				}, func() { w.submitOpen(ctx) }),
				rotheme.Button("Cancel", func() { w.cancel(ctx) }),
			).Gap(8).CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *VendingWindow) vendorItemsTree(ctx Context) widget.Widget {
	return Win(
		Title("Vending Items"),
		CloseButton(true),
		OnClose(func() { w.cancel(ctx) }),
		Size(vendingWindowW, float32(w.leftHeight())),
		FooterHeight(vendingFooterH),
		Content(primitives.Box(w.vendorItemsTable(ctx)).Height(float32(shopTableHeight(vendingBuyRows)))),
	)
}

func (w *VendingWindow) buyCartTree(ctx Context) widget.Widget {
	return Win(
		Title("Buying Items"),
		CloseButton(true),
		OnClose(func() { w.cancel(ctx) }),
		Size(vendingWindowW, float32(w.rightHeight())),
		FooterHeight(vendingFooterH),
		Content(primitives.Box(w.buyCartTable(ctx)).Height(float32(shopTableHeight(vendingCartRows)))),
		Footer(
			primitives.HBox(
				rotheme.Text(fmt.Sprintf("Total: %s Zeny", formatHUDNumber(w.buyTotal()))),
				primitives.Expanded(primitives.Box()),
				rotheme.ButtonDisabled("Buy", len(w.buyCart) == 0, func() { w.submitBuy(ctx) }),
				rotheme.Button("Cancel", func() { w.cancel(ctx) }),
			).Gap(8).CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *VendingWindow) ownShopTree(ctx Context) widget.Widget {
	return Win(
		Title("Vending"),
		CloseButton(true),
		OnClose(func() { w.closeOwnStore(ctx) }),
		Size(vendingWindowW, float32(w.leftHeight())),
		FooterHeight(vendingFooterH),
		Content(primitives.Box(w.ownItemsTable(ctx)).Height(float32(shopTableHeight(vendingBuyRows)))),
		Footer(
			primitives.HBox(
				rotheme.Text(fmt.Sprintf("Items: %d", len(w.ownItems))),
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Close", func() { w.closeOwnStore(ctx) }),
			).Gap(8).CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *VendingWindow) cartItemsTable(ctx Context) *datatable.Widget {
	return w.table(w.cartRows(ctx), w.ensureLeftScroll(), -1)
}

func (w *VendingWindow) setupItemsTable(ctx Context) *datatable.Widget {
	return w.table(w.setupRowsData(ctx), w.ensureRightScroll(), w.selectedSetup)
}

func (w *VendingWindow) vendorItemsTable(ctx Context) *datatable.Widget {
	return w.table(w.vendorRows(ctx), w.ensureLeftScroll(), w.selectedBuy)
}

func (w *VendingWindow) buyCartTable(ctx Context) *datatable.Widget {
	return w.table(w.buyCartRows(ctx), w.ensureRightScroll(), w.selectedCart)
}

func (w *VendingWindow) ownItemsTable(ctx Context) *datatable.Widget {
	return w.table(w.ownRows(ctx), w.ensureLeftScroll(), -1)
}

func (w *VendingWindow) table(rows []shopTableRow, scroll state.Signal[float32], selectedRow int) *datatable.Widget {
	return datatable.New(
		datatable.Columns([]datatable.Column{
			{Key: "item", Title: "Item", Width: 250},
			{Key: "price", Title: "Price", Width: 104, Align: widget.TextAlignRight},
			{Key: "amount", Title: "Qty", Width: 66, Align: widget.TextAlignCenter},
		}),
		datatable.RowCount(len(rows)),
		datatable.RowHeight(shopRowH),
		datatable.ScrollYSignal(scroll),
		datatable.SelectionModeOpt(datatable.SelectionSingle),
		datatable.SelectedRow(selectedRow),
		datatable.PainterOpt(shopBuyTablePainter{icons: rowIcons(rows)}),
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
			default:
				return ""
			}
		}),
	)
}

func rowIcons(rows []shopTableRow) []image.Image {
	icons := make([]image.Image, len(rows))
	for i, row := range rows {
		if row.icon != nil {
			icons[i] = row.icon
		}
	}
	return icons
}

func (w *VendingWindow) cartRows(ctx Context) []shopTableRow {
	items := vendingCartItems(ctx.Session)
	rows := make([]shopTableRow, len(items))
	for i, item := range items {
		rows[i] = shopTableRow{
			name:   inventoryItemDisplayName(ctx.Resources, item),
			amount: fmt.Sprintf("%d", item.Amount),
			price:  "",
			icon:   w.itemIconImage(ctx.Resources, item),
		}
	}
	return rows
}

func (w *VendingWindow) setupRowsData(ctx Context) []shopTableRow {
	rows := make([]shopTableRow, len(w.setupItems))
	for i, item := range w.setupItems {
		rows[i] = shopTableRow{
			name:   inventoryItemDisplayName(ctx.Resources, item.item),
			amount: fmt.Sprintf("%d", item.amount),
			price:  formatHUDNumber(int64(item.price)) + " Z",
			icon:   w.itemIconImage(ctx.Resources, item.item),
		}
	}
	return rows
}

func (w *VendingWindow) vendorRows(ctx Context) []shopTableRow {
	rows := make([]shopTableRow, len(w.buyItems))
	for i, item := range w.buyItems {
		rows[i] = shopTableRow{
			name:   vendingItemName(ctx.Resources, item),
			amount: fmt.Sprintf("%d", item.Amount),
			price:  formatHUDNumber(int64(item.Price)) + " Z",
			icon:   w.vendingItemIconImage(ctx.Resources, item),
		}
	}
	return rows
}

func (w *VendingWindow) buyCartRows(ctx Context) []shopTableRow {
	rows := make([]shopTableRow, len(w.buyCart))
	for i, item := range w.buyCart {
		rows[i] = shopTableRow{
			name:   vendingItemName(ctx.Resources, item.item),
			amount: fmt.Sprintf("%d", item.amount),
			price:  formatHUDNumber(int64(item.item.Price)*int64(item.amount)) + " Z",
			icon:   w.vendingItemIconImage(ctx.Resources, item.item),
		}
	}
	return rows
}

func (w *VendingWindow) ownRows(ctx Context) []shopTableRow {
	rows := make([]shopTableRow, len(w.ownItems))
	for i, item := range w.ownItems {
		rows[i] = shopTableRow{
			name:   vendingItemName(ctx.Resources, item),
			amount: fmt.Sprintf("%d", item.Amount),
			price:  formatHUDNumber(int64(item.Price)) + " Z",
			icon:   w.vendingItemIconImage(ctx.Resources, item),
		}
	}
	return rows
}

func (w *VendingWindow) addCartRow(ctx Context, row int) {
	items := vendingCartItems(ctx.Session)
	if row < 0 || row >= len(items) || len(w.setupItems) >= w.maxItems {
		return
	}
	item := items[row]
	for i := range w.setupItems {
		if w.setupItems[i].item.Index == item.Index {
			return
		}
	}
	amount := uint16(1)
	if item.Amount > 0 && item.Amount < int(^uint16(0)) {
		amount = uint16(item.Amount)
	}
	w.setupItems = append(w.setupItems, vendingSetupItem{item: item, amount: amount, price: vendingDefaultPrice})
	w.selectedSetup = len(w.setupItems) - 1
	w.syncPriceInput()
	w.refresh(ctx)
}

func (w *VendingWindow) addVendorRow(row int) {
	if row < 0 || row >= len(w.buyItems) {
		return
	}
	item := w.buyItems[row]
	for i := range w.buyCart {
		if w.buyCart[i].item.Index != item.Index {
			continue
		}
		if w.buyCart[i].amount < item.Amount {
			w.buyCart[i].amount++
		}
		return
	}
	w.buyCart = append(w.buyCart, vendingBuyCartItem{item: item, amount: 1})
}

func (w *VendingWindow) removeBuyCartRow(row int) {
	if row < 0 || row >= len(w.buyCart) {
		return
	}
	w.buyCart = append(w.buyCart[:row], w.buyCart[row+1:]...)
}

func (w *VendingWindow) removeSetupRow(row int) {
	if row < 0 || row >= len(w.setupItems) {
		return
	}
	w.setupItems = append(w.setupItems[:row], w.setupItems[row+1:]...)
	if w.selectedSetup >= len(w.setupItems) {
		w.selectedSetup = len(w.setupItems) - 1
	}
	w.syncPriceInput()
}

func (w *VendingWindow) setSelectedSetupPrice(value string) {
	w.priceInput = value
	if w.selectedSetup < 0 || w.selectedSetup >= len(w.setupItems) {
		return
	}
	price, err := strconv.ParseUint(value, 10, 32)
	if err != nil || price == 0 {
		return
	}
	w.setupItems[w.selectedSetup].price = uint32(price)
}

func (w *VendingWindow) syncPriceInput() {
	if w.selectedSetup < 0 || w.selectedSetup >= len(w.setupItems) {
		w.priceInput = ""
		w.priceField = nil
		return
	}
	w.priceInput = strconv.FormatUint(uint64(w.setupItems[w.selectedSetup].price), 10)
	w.priceField = nil
}

func (w *VendingWindow) handlePointer(ctx Context, itemInfo *ItemInfoWindow) bool {
	if ctx.Input == nil {
		return false
	}
	if ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		if item, ok := w.itemAt(ctx, ctx.Input.MouseX, ctx.Input.MouseY); ok && itemInfo != nil {
			itemInfo.openItem(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY)
			return true
		}
	}
	if ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		w.pressX, w.pressY = ctx.Input.MouseX, ctx.Input.MouseY
		w.pressRow = -1
		w.pressRight = false
		if row, ok := w.leftRowAt(ctx, ctx.Input.MouseX, ctx.Input.MouseY); ok {
			w.pressRow = row
			w.pressRight = false
			return true
		}
		if row, ok := w.rightRowAt(ctx.Input.MouseX, ctx.Input.MouseY); ok {
			w.pressRow = row
			w.pressRight = true
			return true
		}
	}
	if ctx.Input.MousePressed(input.MouseButtonLeft) && w.pressRow >= 0 {
		if absShopWindowInt(ctx.Input.MouseX-w.pressX) > 4 || absShopWindowInt(ctx.Input.MouseY-w.pressY) > 4 {
			w.draggingItem = true
			return true
		}
	}
	if ctx.Input.MouseJustReleased(input.MouseButtonLeft) && w.pressRow >= 0 {
		row, right, dragging := w.pressRow, w.pressRight, w.draggingItem
		w.pressRow = -1
		w.draggingItem = false
		if dragging {
			w.handleDrop(ctx, row, right)
			return true
		}
		if w.isDoubleClick(row, right) {
			if right {
				if w.mode == vendingModeSetup {
					w.removeSetupRow(row)
				} else if w.mode == vendingModeBuy {
					w.removeBuyCartRow(row)
				}
			} else {
				if w.mode == vendingModeSetup {
					w.addCartRow(ctx, row)
				} else if w.mode == vendingModeBuy {
					w.addVendorRow(row)
				}
			}
			w.refresh(ctx)
			return true
		}
		w.selectRow(ctx, row, right)
		w.rememberClick(row, right)
		return true
	}
	return false
}

func (w *VendingWindow) selectRow(ctx Context, row int, right bool) {
	switch w.mode {
	case vendingModeSetup:
		if right {
			w.selectedSetup = row
			w.syncPriceInput()
			w.refresh(ctx)
		}
	case vendingModeBuy:
		if right {
			w.selectedCart = row
		} else {
			w.selectedBuy = row
		}
		w.refresh(ctx)
	}
}

func (w *VendingWindow) handleDrop(ctx Context, row int, fromRight bool) {
	if fromRight {
		if w.leftDropTarget(ctx.Input.MouseX, ctx.Input.MouseY) {
			if w.mode == vendingModeSetup {
				w.removeSetupRow(row)
			} else if w.mode == vendingModeBuy {
				w.removeBuyCartRow(row)
			}
			w.refresh(ctx)
		}
		return
	}
	if w.rightDropTarget(ctx.Input.MouseX, ctx.Input.MouseY) {
		if w.mode == vendingModeSetup {
			w.addCartRow(ctx, row)
		} else if w.mode == vendingModeBuy {
			w.addVendorRow(row)
			w.refresh(ctx)
		}
	}
}

func (w *VendingWindow) itemAt(ctx Context, mx, my int) (session.InventoryItem, bool) {
	if row, ok := w.leftRowAt(ctx, mx, my); ok {
		switch w.mode {
		case vendingModeSetup:
			items := vendingCartItems(ctx.Session)
			if row >= 0 && row < len(items) {
				return items[row], true
			}
		case vendingModeBuy:
			if row >= 0 && row < len(w.buyItems) {
				return session.InventoryItem{ItemID: w.buyItems[row].ItemID, Type: w.buyItems[row].Type, Amount: int(w.buyItems[row].Amount), Identified: w.buyItems[row].Identified, Damaged: w.buyItems[row].Damaged, Refine: w.buyItems[row].Refine}, true
			}
		case vendingModeOwn:
			if row >= 0 && row < len(w.ownItems) {
				return session.InventoryItem{ItemID: w.ownItems[row].ItemID, Type: w.ownItems[row].Type, Amount: int(w.ownItems[row].Amount), Identified: w.ownItems[row].Identified, Damaged: w.ownItems[row].Damaged, Refine: w.ownItems[row].Refine}, true
			}
		}
	}
	if row, ok := w.rightRowAt(mx, my); ok {
		switch w.mode {
		case vendingModeSetup:
			if row >= 0 && row < len(w.setupItems) {
				return w.setupItems[row].item, true
			}
		case vendingModeBuy:
			if row >= 0 && row < len(w.buyCart) {
				item := w.buyCart[row].item
				return session.InventoryItem{ItemID: item.ItemID, Type: item.Type, Amount: int(item.Amount), Identified: item.Identified, Damaged: item.Damaged, Refine: item.Refine}, true
			}
		}
	}
	return session.InventoryItem{}, false
}

func (w *VendingWindow) dragItem(ctx Context) (session.InventoryItem, bool) {
	if w.pressRight {
		if w.mode == vendingModeSetup && w.pressRow >= 0 && w.pressRow < len(w.setupItems) {
			return w.setupItems[w.pressRow].item, true
		}
		if w.mode == vendingModeBuy && w.pressRow >= 0 && w.pressRow < len(w.buyCart) {
			item := w.buyCart[w.pressRow].item
			return session.InventoryItem{ItemID: item.ItemID, Type: item.Type, Amount: int(item.Amount), Identified: item.Identified}, true
		}
		return session.InventoryItem{}, false
	}
	if w.mode == vendingModeSetup {
		items := vendingCartItems(ctx.Session)
		if w.pressRow >= 0 && w.pressRow < len(items) {
			return items[w.pressRow], true
		}
	}
	if w.mode == vendingModeBuy && w.pressRow >= 0 && w.pressRow < len(w.buyItems) {
		item := w.buyItems[w.pressRow]
		return session.InventoryItem{ItemID: item.ItemID, Type: item.Type, Amount: int(item.Amount), Identified: item.Identified}, true
	}
	return session.InventoryItem{}, false
}

func (w *VendingWindow) leftRowAt(ctx Context, mx, my int) (int, bool) {
	tableX, tableY, tableW, tableH := w.leftTableBounds()
	rows := vendingBuyRows
	switch w.mode {
	case vendingModeSetup:
		rows = len(vendingCartItems(ctx.Session))
	case vendingModeBuy:
		rows = len(w.buyItems)
	case vendingModeOwn:
		rows = len(w.ownItems)
	}
	return tableRowAt(mx, my, tableX, tableY, tableW, tableH, rows, shopRowH, w.ensureLeftScroll().Get())
}

func (w *VendingWindow) rightRowAt(mx, my int) (int, bool) {
	tableX, tableY, tableW, tableH := w.rightTableBounds()
	rowCount := len(w.setupItems)
	if w.mode == vendingModeBuy {
		rowCount = len(w.buyCart)
	}
	return tableRowAt(mx, my, tableX, tableY, tableW, tableH, rowCount, shopRowH, w.ensureRightScroll().Get())
}

func (w *VendingWindow) leftDropTarget(mx, my int) bool {
	return pointInRect(mx, my, w.leftWindow.x, w.leftWindow.y, vendingWindowW, w.leftHeight())
}

func (w *VendingWindow) rightDropTarget(mx, my int) bool {
	return pointInRect(mx, my, w.rightWindow.x, w.rightWindow.y, vendingWindowW, w.rightHeight())
}

func (w *VendingWindow) inside(mx, my int) bool {
	return pointInRect(mx, my, w.leftWindow.x, w.leftWindow.y, vendingWindowW, w.leftHeight()) ||
		pointInRect(mx, my, w.rightWindow.x, w.rightWindow.y, vendingWindowW, w.rightHeight())
}

func (w *VendingWindow) leftTableBounds() (int, int, int, int) {
	return w.leftWindow.x, w.leftWindow.y + ROWindowTitleHeight, vendingWindowW, shopTableHeight(vendingSetupRows)
}

func (w *VendingWindow) rightTableBounds() (int, int, int, int) {
	y := w.rightWindow.y + ROWindowTitleHeight
	h := shopTableHeight(w.setupRows())
	if w.mode == vendingModeSetup {
		y += vendingNameH + 8
	}
	return w.rightWindow.x, y, vendingWindowW, h
}

func (w *VendingWindow) leftHeight() int {
	return ROWindowTitleHeight + shopTableHeight(vendingSetupRows) + vendingFooterH
}

func (w *VendingWindow) rightHeight() int {
	return ROWindowTitleHeight + w.setupContentHeight() + vendingFooterH
}

func (w *VendingWindow) setupContentHeight() int {
	if w.mode == vendingModeSetup {
		return vendingNameH + 8 + shopTableHeight(w.setupRows())
	}
	return shopTableHeight(vendingCartRows)
}

func (w *VendingWindow) setupRows() int {
	rows := w.maxItems
	if rows <= 0 {
		rows = vendingCartRows
	}
	if rows > vendingSetupRows {
		rows = vendingSetupRows
	}
	return rows
}

func (w *VendingWindow) submitOpen(ctx Context) {
	shopName := w.currentShopName()
	if ctx.Network == nil || len(w.setupItems) == 0 || shopName == "" {
		return
	}
	w.shopName = shopName
	items := make([]network.VendingOpenItem, 0, len(w.setupItems))
	for _, item := range w.setupItems {
		glog.Debugf("vending open item cart_index=%d item=%d amount=%d price=%d identified=%t damaged=%t", item.item.Index, item.item.ItemID, item.amount, item.price, item.item.Identified, item.item.Damaged)
		items = append(items, network.VendingOpenItem{Index: item.item.Index, Amount: item.amount, Price: item.price})
	}
	if err := ctx.Network.SendOpenVendingStore(shopName, items); err != nil {
		glog.Warnf("open vending failed: %v", err)
	}
	w.mode = vendingModeNone
	w.closeBoth(ctx)
}

func (w *VendingWindow) currentShopName() string {
	if w.nameField != nil {
		return strings.TrimSpace(w.nameField.Text())
	}
	return strings.TrimSpace(w.shopName)
}

func (w *VendingWindow) submitBuy(ctx Context) {
	if ctx.Network == nil || w.ownerAID == 0 || len(w.buyCart) == 0 {
		return
	}
	items := make([]network.VendingPurchaseItem, 0, len(w.buyCart))
	for _, item := range w.buyCart {
		items = append(items, network.VendingPurchaseItem{Index: item.item.Index, Amount: item.amount})
	}
	if err := ctx.Network.SendVendingPurchase(w.ownerAID, items); err != nil {
		glog.Warnf("vending purchase failed: %v", err)
		return
	}
	w.mode = vendingModeNone
	w.buyCart = nil
	w.closeBoth(ctx)
}

func (w *VendingWindow) closeOwnStore(ctx Context) {
	if ctx.Network != nil {
		if err := ctx.Network.SendCloseVendingStore(); err != nil {
			glog.Warnf("close vending failed: %v", err)
		}
	}
	w.mode = vendingModeNone
	w.closeBoth(ctx)
}

func (w *VendingWindow) cancel(ctx Context) {
	if w.mode == vendingModeSetup && ctx.Network != nil {
		if err := ctx.Network.SendCancelVendingStoreOpen(); err != nil {
			glog.Warnf("cancel vending setup failed: %v", err)
		}
	}
	w.mode = vendingModeNone
	w.closeBoth(ctx)
}

func (w *VendingWindow) buyTotal() int64 {
	var total int64
	for _, item := range w.buyCart {
		total += int64(item.item.Price) * int64(item.amount)
	}
	return total
}

func (w *VendingWindow) rememberClick(row int, right bool) {
	w.lastClickRow = row
	w.lastClickSide = right
	w.lastClickAt = time.Now()
}

func (w *VendingWindow) isDoubleClick(row int, right bool) bool {
	return w.lastClickRow == row && w.lastClickSide == right && time.Since(w.lastClickAt) <= 360*time.Millisecond
}

func (w *VendingWindow) ensureLeftScroll() state.Signal[float32] {
	if w.leftScroll == nil {
		w.leftScroll = state.NewSignal[float32](0)
	}
	return w.leftScroll
}

func (w *VendingWindow) ensureRightScroll() state.Signal[float32] {
	if w.rightScroll == nil {
		w.rightScroll = state.NewSignal[float32](0)
	}
	return w.rightScroll
}

func (w *VendingWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
	if manager == nil || item.ItemID == 0 {
		return nil
	}
	key := shopItemIconKey{itemID: item.ItemID, identified: item.Identified}
	if w.icons != nil {
		if img := w.icons[key]; img != nil {
			return img
		}
	}
	if _, missed := w.iconMiss[key]; missed {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(item.ItemID), item.Identified)
	if !ok {
		w.markIconMiss(key)
		return nil
	}
	img, _, err := res.LoadImage(manager, res.ItemIconTextureCandidates(resourceName))
	if err != nil {
		w.markIconMiss(key)
		return nil
	}
	if w.icons == nil {
		w.icons = make(map[shopItemIconKey]image.Image)
	}
	w.icons[key] = img
	return img
}

func (w *VendingWindow) vendingItemIconImage(manager *res.Manager, item network.VendingItem) image.Image {
	return w.itemIconImage(manager, session.InventoryItem{ItemID: item.ItemID, Identified: item.Identified, Type: item.Type, Amount: int(item.Amount), Damaged: item.Damaged, Refine: item.Refine})
}

func (w *VendingWindow) markIconMiss(key shopItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[shopItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func vendingItemName(manager *res.Manager, item network.VendingItem) string {
	return inventoryItemDisplayName(manager, session.InventoryItem{ItemID: item.ItemID, Identified: item.Identified, Type: item.Type, Amount: int(item.Amount), Damaged: item.Damaged, Refine: item.Refine})
}

func vendingCartItems(s *session.Session) []session.InventoryItem {
	items := sortedCartItems(s)
	out := items[:0]
	for _, item := range items {
		if item.ItemID == 0 || item.Amount <= 0 || !item.Identified || item.Damaged {
			continue
		}
		out = append(out, item)
	}
	return out
}
