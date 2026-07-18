package ui

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"sort"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	cartWindowWidth  = storageWindowWidth
	cartTableHeaderH = 36
	cartRows         = storageRows
	cartWindowHeight = ROWindowTitleHeight + cartTableHeaderH + cartRows*storageRowH + ROWindowFooterHeight
)

type CartWindow struct {
	Window
	scrollY       state.Signal[float32]
	selectedRow   int
	snapshot      uint64
	itemInfo      *ItemInfoWindow
	lastClickItem uint16
	lastClickAt   time.Time
	dragItem      session.InventoryItem
	dragActive    bool
	dragFrom      time.Time
	icons         map[storageItemIconKey]image.Image
	iconMiss      map[storageItemIconKey]struct{}
}

func (w *CartWindow) Toggle(ctx Context) {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	if w.IsOpen() {
		w.close(ctx)
		w.Publish(ctx)
		return
	}
	w.OpenWindow(ctx)
}

func (w *CartWindow) SetOpen(open bool) {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	if !open {
		w.Window.Close()
	}
}

func (w *CartWindow) OpenWindow(ctx Context) {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	w.ClampScroll(ctx.Session)
	w.selectedRow = -1
	w.snapshot = w.cartSnapshot(ctx.Session)
	x, y := cartDefaultPosition(ctx)
	if !w.IsOpen() {
		w.OpenAt(x, y, w.widgetTree(ctx, nil))
	} else {
		w.SetAutoPosition(x, y)
		w.SetContent(w.widgetTree(ctx, w.itemInfo))
	}
	if ctx.Session != nil {
		ctx.Session.Cart.Open = true
	}
	w.Publish(ctx)
}

func (w *CartWindow) Update(ctx Context, inventory *InventoryBagWindow, storage *StorageWindow, itemInfo *ItemInfoWindow) bool {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	if !w.IsOpen() || ctx.Input == nil {
		return false
	}
	if !inventoryBagHasCart(ctx) {
		w.close(ctx)
		w.Publish(ctx)
		return false
	}
	if w.UpdateDrag(ctx, inventory, storage) {
		return true
	}
	w.ClampScroll(ctx.Session)
	snapshot := w.cartSnapshot(ctx.Session)
	if snapshot != w.snapshot || itemInfo != w.itemInfo {
		w.snapshot = snapshot
		w.itemInfo = itemInfo
		w.SetContent(w.widgetTree(ctx, itemInfo))
	}
	if w.handlePointer(ctx, itemInfo) {
		return true
	}
	consumed := w.Window.Update(ctx)
	if !w.IsOpen() {
		w.Publish(ctx)
		return consumed
	}
	w.Publish(ctx)
	return consumed
}

func (w *CartWindow) UpdateDrag(ctx Context, inventory *InventoryBagWindow, storage *StorageWindow) bool {
	if !w.dragActive || ctx.Input == nil {
		return false
	}
	if ctx.Input.MouseJustReleased(input.MouseButtonLeft) || !ctx.Input.MousePressed(input.MouseButtonLeft) {
		item := w.dragItem
		w.dragActive = false
		w.dragItem = session.InventoryItem{}
		if inventory != nil && inventory.AcceptStorageDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			w.withdraw(ctx, item)
			return true
		}
		if storage != nil && storage.AcceptCartDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		return true
	}
	return true
}

func (w *CartWindow) Draw(screen *render.Frame, ctx Context, assets AssetProvider) {
	w.Publish(ctx)
}

func (w *CartWindow) DrawDragGhost(screen *render.Frame, ctx Context, assets AssetProvider) {
	if w.dragActive && screen != nil && ctx.Input != nil && assets != nil && time.Since(w.dragFrom) > 80*time.Millisecond {
		assets.DrawInventoryItemIcon(screen, ctx.Resources, w.dragItem, ctx.Input.MouseX-inventoryIconSize/2, ctx.Input.MouseY-inventoryIconSize/2)
	}
}

func (w *CartWindow) AcceptInventoryDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	if !w.IsOpen() || !pointInRect(mx, my, w.x, w.y, cartWindowWidth, cartWindowHeight) {
		return false
	}
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		glog.Warnf("cart deposit failed: not connected")
		return true
	}
	if err := ctx.Network.SendMoveToCart(item.Index, amount); err != nil {
		glog.Warnf("cart deposit failed: %v", err)
		return true
	}
	glog.Debugf("cart deposit requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
	return true
}

func (w *CartWindow) AcceptStorageDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	if !w.IsOpen() || !pointInRect(mx, my, w.x, w.y, cartWindowWidth, cartWindowHeight) {
		return false
	}
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		glog.Warnf("storage to cart failed: not connected")
		return true
	}
	if err := ctx.Network.SendMoveStorageToCart(item.Index, amount); err != nil {
		glog.Warnf("storage to cart failed: %v", err)
		return true
	}
	glog.Debugf("storage to cart requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
	return true
}

func (w *CartWindow) widgetTree(ctx Context, itemInfo *ItemInfoWindow) widget.Widget {
	return Win(
		Title("Pushcart"),
		CloseButton(true),
		OnClose(func() {
			w.close(ctx)
			w.Publish(ctx)
		}),
		Size(cartWindowWidth, cartWindowHeight),
		Content(
			primitives.Box(w.cartTableWidget(ctx)).
				Height(cartTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			rotheme.Text(w.cartCountText(ctx.Session)),
			primitives.Expanded(primitives.Box()),
			rotheme.Text(w.cartWeightText(ctx.Session)),
		),
	)
}

func (w *CartWindow) cartTableWidget(ctx Context) *rotheme.TableViewWidget {
	items := sortedCartItems(ctx.Session)
	rows := w.cartRows(ctx, items)
	icons := w.cartItemIcons(ctx, items)
	return rotheme.TableView(
		rotheme.TableViewColumns(storageTableColumns),
		rotheme.TableViewRowCount(len(rows)),
		rotheme.TableViewRowHeight(storageRowH),
		rotheme.TableViewHeaderHeight(cartTableHeaderH),
		rotheme.TableViewEmptyText("No items"),
		rotheme.TableViewScrollYSignal(w.ensureScrollSignal()),
		rotheme.TableViewSelectedRow(state.NewSignal[int](w.selectedRow)),
		rotheme.TableViewInvalidateHover(false),
		rotheme.TableViewDispatchHoverToCells(false),
		rotheme.TableViewBuildSimpleCell(func(cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
			if cell.Row < 0 || cell.Row >= len(rows) {
				return rotheme.TableViewSimpleCell{Hidden: true}
			}
			switch cell.Column.Key {
			case "item":
				var icon image.Image
				if cell.Row < len(icons) {
					icon = icons[cell.Row]
				}
				return rotheme.TableViewSimpleCell{Icon: icon, Text: rows[cell.Row].name}
			case "amount":
				return rotheme.TableViewSimpleCell{
					Text:  rows[cell.Row].amount,
					Align: widget.TextAlignRight,
					Color: rotheme.Default.Colors.MutedText,
				}
			default:
				return rotheme.TableViewSimpleCell{Hidden: true}
			}
		}),
		rotheme.TableViewOnRowClick(func(row int) {
			if row >= 0 && row < len(rows) {
				w.selectedRow = row
			}
		}),
	)
}

func (w *CartWindow) handlePointer(ctx Context, itemInfo *ItemInfoWindow) bool {
	if ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		item, _, ok := w.itemAt(ctx.Session, ctx.Input.MouseX, ctx.Input.MouseY)
		if !ok {
			return false
		}
		if itemInfo != nil {
			itemInfo.openItem(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY)
		}
		return true
	}
	if !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return false
	}
	item, row, ok := w.itemAt(ctx.Session, ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok {
		return false
	}
	w.selectedRow = row
	now := time.Now()
	if w.lastClickItem == item.Index && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.withdraw(ctx, item)
		w.lastClickItem = 0
		w.refresh(ctx, itemInfo)
		return true
	}
	w.lastClickItem = item.Index
	w.lastClickAt = now
	w.dragItem = item
	w.dragActive = true
	w.dragFrom = now
	w.refresh(ctx, itemInfo)
	return true
}

func (w *CartWindow) close(ctx Context) {
	w.Window.Close()
	w.dragActive = false
	if ctx.Session != nil {
		ctx.Session.Cart.Open = false
	}
}

func (w *CartWindow) withdraw(ctx Context, item session.InventoryItem) {
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		glog.Warnf("cart withdraw failed: not connected")
		return
	}
	if err := ctx.Network.SendMoveFromCart(item.Index, amount); err != nil {
		glog.Warnf("cart withdraw failed: %v", err)
		return
	}
	glog.Debugf("cart withdraw requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
}

func (w *CartWindow) refresh(ctx Context, itemInfo *ItemInfoWindow) {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	w.ClampScroll(ctx.Session)
	w.snapshot = w.cartSnapshot(ctx.Session)
	w.itemInfo = itemInfo
	w.SetContent(w.widgetTree(ctx, itemInfo))
	w.Publish(ctx)
}

func (w *CartWindow) Refresh(ctx Context, itemInfo *ItemInfoWindow) {
	w.refresh(ctx, itemInfo)
}

func (w *CartWindow) Rebind(ctx Context, itemInfo *ItemInfoWindow) {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	if !w.IsOpen() {
		return
	}
	w.refresh(ctx, itemInfo)
}

func (w *CartWindow) ClampScroll(s *session.Session) {
	itemCount := 0
	if s != nil {
		itemCount = len(s.Cart.Items)
	}
	if w.selectedRow >= itemCount {
		w.selectedRow = -1
	}
	scroll := w.ensureScrollSignal()
	maxScroll := float32(maxInt(0, itemCount-cartRows) * storageRowH)
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *CartWindow) cartRows(ctx Context, items []session.InventoryItem) []storageTableRow {
	rows := make([]storageTableRow, len(items))
	for i, item := range items {
		name := inventoryItemDisplayName(ctx.Resources, item)
		if item.Refine > 0 {
			name = fmt.Sprintf("+%d %s", item.Refine, name)
		}
		rows[i] = storageTableRow{
			name:   name,
			amount: fmt.Sprintf("x%d", maxInt(1, item.Amount)),
		}
	}
	return rows
}

func (w *CartWindow) cartItemIcons(ctx Context, items []session.InventoryItem) []image.Image {
	icons := make([]image.Image, len(items))
	for i, item := range items {
		icons[i] = w.itemIconImage(ctx.Resources, item)
	}
	return icons
}

func (w *CartWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
	if manager == nil || item.ItemID == 0 {
		return nil
	}
	key := storageItemIconKey{itemID: item.ItemID, identified: item.Identified}
	if w.icons != nil {
		if img := w.icons[key]; img != nil {
			return img
		}
	}
	if _, ok := w.iconMiss[key]; ok {
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
		w.icons = make(map[storageItemIconKey]image.Image)
	}
	w.icons[key] = img
	return img
}

func (w *CartWindow) markIconMiss(key storageItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[storageItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func (w *CartWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func (w *CartWindow) itemAt(s *session.Session, mx, my int) (session.InventoryItem, int, bool) {
	tableX := w.x
	tableY := w.y + ROWindowTitleHeight
	row, ok := cartTableRowAt(mx, my, tableX, tableY, cartWindowWidth, int(cartTableHeight()), len(sortedCartItems(s)), w.ensureScrollSignal().Get())
	if !ok {
		return session.InventoryItem{}, 0, false
	}
	items := sortedCartItems(s)
	if row < 0 || row >= len(items) {
		return session.InventoryItem{}, 0, false
	}
	return items[row], row, true
}

func (w *CartWindow) cartSnapshot(s *session.Session) uint64 {
	if s == nil {
		return 0
	}
	hash := storageSnapshotMix(storageSnapshotSeed, uint64(s.Cart.Amount))
	hash = storageSnapshotMix(hash, uint64(s.Cart.MaxAmount))
	hash = storageSnapshotMix(hash, uint64(s.Cart.Weight))
	hash = storageSnapshotMix(hash, uint64(s.Cart.MaxWeight))
	hash = storageSnapshotMix(hash, uint64(len(s.Cart.Items)))
	for _, item := range s.Cart.Items {
		hash = storageSnapshotItem(hash, item)
	}
	return hash
}

func (w *CartWindow) cartCountText(s *session.Session) string {
	if s == nil {
		return "Num: 0/0"
	}
	return fmt.Sprintf("Num: %d/%d", s.Cart.Amount, s.Cart.MaxAmount)
}

func (w *CartWindow) cartWeightText(s *session.Session) string {
	if s == nil || s.Cart.MaxWeight <= 0 {
		return "Weight: 0/0"
	}
	return fmt.Sprintf("Weight: %.1f/%.1f", float64(s.Cart.Weight)/10, float64(s.Cart.MaxWeight)/10)
}

func cartTableHeight() float32 {
	return cartTableHeaderH + cartRows*storageRowH
}

func cartTableRowAt(mx, my, tableX, tableY, tableW, tableH, rowCount int, scrollY float32) (int, bool) {
	if !pointInRect(mx, my, tableX, tableY+cartTableHeaderH, scrollbarSafeIntWidth(tableW), tableH-cartTableHeaderH) {
		return 0, false
	}
	localY := float32(my-tableY) - cartTableHeaderH + scrollY
	row := int(localY / float32(storageRowH))
	if row < 0 || row >= rowCount {
		return 0, false
	}
	return row, true
}

func cartDefaultPosition(ctx Context) (int, int) {
	width, _ := ctx.ScreenSize()
	return maxInt(8, width-cartWindowWidth-24), 118
}

func sortedCartItems(s *session.Session) []session.InventoryItem {
	if s == nil || len(s.Cart.Items) == 0 {
		return nil
	}
	items := append([]session.InventoryItem(nil), s.Cart.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}
