package ui

import (
	"fmt"
	"image"
	"sort"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	storageWindowWidth  = 312
	storageWindowTitleH = ROWindowTitleHeight
	storageRowH         = 32
	storageRows         = 9
	storageTableViewH   = storageRows * storageRowH
	storageWindowHeight = storageWindowTitleH + storageTableViewH + ROWindowFooterHeight
)

type StorageWindow struct {
	Window
	scrollY           state.Signal[float32]
	selectedRow       int
	selectedRowSignal state.Signal[int]
	snapshot          uint64
	itemInfo          *ItemInfoWindow
	lastClickItem     uint16
	lastClickAt       time.Time
	dragItem          session.InventoryItem
	dragActive        bool
	dragFrom          time.Time
	icons             map[storageItemIconKey]image.Image
	iconMiss          map[storageItemIconKey]struct{}
}

type storageItemIconKey struct {
	itemID     uint16
	identified bool
}

type storageTableRow struct {
	name   string
	amount string
}

func (w *StorageWindow) SetOpen(open bool) {
	w.EnsureWindow(storageWindowWidth, storageWindowHeight)
	if !open {
		w.Window.Close()
	}
}

func (w *StorageWindow) OpenWindow(ctx Context) {
	w.EnsureWindow(storageWindowWidth, storageWindowHeight)
	w.ClampScroll(ctx.Session)
	w.setSelectedRow(-1)
	w.snapshot = w.storageSnapshot(ctx.Session)
	x, y := storageDefaultPosition(ctx)
	if !w.IsOpen() {
		w.OpenAt(x, y, w.widgetTree(ctx, nil))
	} else {
		w.SetAutoPosition(x, y)
		w.SetContent(w.widgetTree(ctx, w.itemInfo))
	}
	w.Publish(ctx)
}

func (w *StorageWindow) Update(ctx Context, inventory *InventoryBagWindow, cart *CartWindow, itemInfo *ItemInfoWindow) bool {
	w.EnsureWindow(storageWindowWidth, storageWindowHeight)
	if !w.IsOpen() || ctx.Input == nil {
		return false
	}
	if ctx.Session == nil || !ctx.Session.Storage.Open {
		w.Window.Close()
		w.dragActive = false
		w.Publish(ctx)
		return false
	}
	if w.UpdateDrag(ctx, inventory, cart) {
		return true
	}
	if ctx.Input.JustPressed(input.KeyEscape) {
		w.close(ctx)
		w.Publish(ctx)
		return true
	}
	w.ClampScroll(ctx.Session)
	snapshot := w.storageSnapshot(ctx.Session)
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

func (w *StorageWindow) UpdateDrag(ctx Context, inventory *InventoryBagWindow, cart *CartWindow) bool {
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
		if cart != nil && cart.AcceptStorageDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		return true
	}
	return true
}

func (w *StorageWindow) Draw(screen *render.Frame, ctx Context, assets AssetProvider) {
	w.Publish(ctx)
}

func (w *StorageWindow) DrawDragGhost(screen *render.Frame, ctx Context, assets AssetProvider) {
	if w.dragActive && screen != nil && ctx.Input != nil && assets != nil && time.Since(w.dragFrom) > 80*time.Millisecond {
		assets.DrawInventoryItemIcon(screen, ctx.Resources, w.dragItem, ctx.Input.MouseX-inventoryIconSize/2, ctx.Input.MouseY-inventoryIconSize/2)
	}
}

func (w *StorageWindow) AcceptInventoryDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	w.EnsureWindow(storageWindowWidth, storageWindowHeight)
	if !w.IsOpen() || !pointInRect(mx, my, w.x, w.y, storageWindowWidth, storageWindowHeight) {
		return false
	}
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		glog.Warnf("storage deposit failed: not connected")
		return true
	}
	if err := ctx.Network.SendMoveToStorage(item.Index, amount); err != nil {
		glog.Warnf("storage deposit failed: %v", err)
		return true
	}
	glog.Debugf("storage deposit requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
	return true
}

func (w *StorageWindow) AcceptCartDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	w.EnsureWindow(storageWindowWidth, storageWindowHeight)
	if !w.IsOpen() || !pointInRect(mx, my, w.x, w.y, storageWindowWidth, storageWindowHeight) {
		return false
	}
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		glog.Warnf("cart to storage failed: not connected")
		return true
	}
	if err := ctx.Network.SendMoveCartToStorage(item.Index, amount); err != nil {
		glog.Warnf("cart to storage failed: %v", err)
		return true
	}
	glog.Debugf("cart to storage requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
	return true
}

func (w *StorageWindow) widgetTree(ctx Context, itemInfo *ItemInfoWindow) widget.Widget {
	return Win(
		Title("Storage"),
		CloseButton(true),
		OnClose(func() {
			w.close(ctx)
			w.Publish(ctx)
		}),
		Size(storageWindowWidth, storageWindowHeight),
		Content(
			primitives.Box(w.storageTableWidget(ctx)).
				Height(storageTableViewHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			rotheme.Text(w.storageCountText(ctx.Session)),
			primitives.Expanded(primitives.Box()),
		),
	)
}

func (w *StorageWindow) storageTableWidget(ctx Context) *rotheme.TableViewWidget {
	items := sortedStorageItems(ctx.Session)
	rows := w.storageRows(ctx, items)
	icons := w.storageItemIcons(ctx, items)
	return rotheme.TableView(
		rotheme.TableViewColumns(storageTableColumns),
		rotheme.TableViewRowCount(len(rows)),
		rotheme.TableViewRowHeight(storageRowH),
		rotheme.TableViewShowHeader(false),
		rotheme.TableViewEmptyText("No items"),
		rotheme.TableViewScrollYSignal(w.ensureScrollSignal()),
		rotheme.TableViewSelectedRow(w.ensureSelectedRowSignal()),
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
				w.setSelectedRow(row)
			}
		}),
	)
}

func (w *StorageWindow) refresh(ctx Context, itemInfo *ItemInfoWindow) {
	w.ClampScroll(ctx.Session)
	w.snapshot = w.storageSnapshot(ctx.Session)
	w.itemInfo = itemInfo
	w.SetContent(w.widgetTree(ctx, itemInfo))
	w.Publish(ctx)
}

func (w *StorageWindow) Refresh(ctx Context, itemInfo *ItemInfoWindow) {
	w.refresh(ctx, itemInfo)
}

func (w *StorageWindow) handlePointer(ctx Context, itemInfo *ItemInfoWindow) bool {
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
	w.setSelectedRow(row)
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

func (w *StorageWindow) close(ctx Context) {
	if ctx.Network != nil {
		if err := ctx.Network.SendCloseStorage(); err != nil {
			glog.Warnf("storage close failed: %v", err)
			return
		}
	}
	w.Window.Close()
	w.dragActive = false
	if ctx.Session != nil {
		ctx.Session.Storage.Open = false
	}
}

func (w *StorageWindow) withdraw(ctx Context, item session.InventoryItem) {
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		glog.Warnf("storage withdraw failed: not connected")
		return
	}
	if err := ctx.Network.SendMoveFromStorage(item.Index, amount); err != nil {
		glog.Warnf("storage withdraw failed: %v", err)
		return
	}
	glog.Debugf("storage withdraw requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
}

func (w *StorageWindow) ClampScroll(s *session.Session) {
	itemCount := 0
	if s != nil {
		itemCount = len(s.Storage.Items)
	}
	if w.selectedRow >= itemCount {
		w.setSelectedRow(-1)
	}
	scroll := w.ensureScrollSignal()
	maxScroll := float32(maxInt(0, itemCount-storageRows) * storageRowH)
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *StorageWindow) storageRows(ctx Context, items []session.InventoryItem) []storageTableRow {
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

func (w *StorageWindow) storageItemIcons(ctx Context, items []session.InventoryItem) []image.Image {
	icons := make([]image.Image, len(items))
	for i, item := range items {
		icons[i] = w.itemIconImage(ctx.Resources, item)
	}
	return icons
}

func (w *StorageWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
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

func (w *StorageWindow) markIconMiss(key storageItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[storageItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func (w *StorageWindow) storageSnapshot(s *session.Session) uint64 {
	if s == nil {
		return 0
	}
	hash := storageSnapshotMix(storageSnapshotSeed, uint64(s.Storage.Amount))
	hash = storageSnapshotMix(hash, uint64(s.Storage.MaxAmount))
	hash = storageSnapshotMix(hash, uint64(len(s.Storage.Items)))
	for _, item := range s.Storage.Items {
		hash = storageSnapshotItem(hash, item)
	}
	return hash
}

const storageSnapshotSeed = uint64(1469598103934665603)

func storageSnapshotMix(hash, value uint64) uint64 {
	hash ^= value + 0x9e3779b97f4a7c15 + (hash << 6) + (hash >> 2)
	return hash
}

func storageSnapshotItem(hash uint64, item session.InventoryItem) uint64 {
	hash = storageSnapshotMix(hash, uint64(item.Index))
	hash = storageSnapshotMix(hash, uint64(item.ItemID))
	hash = storageSnapshotMix(hash, uint64(item.Type))
	hash = storageSnapshotMix(hash, uint64(item.Location))
	hash = storageSnapshotMix(hash, storageBoolSnapshot(item.Identified))
	hash = storageSnapshotMix(hash, uint64(item.Amount))
	hash = storageSnapshotMix(hash, storageBoolSnapshot(item.Equip))
	hash = storageSnapshotMix(hash, storageBoolSnapshot(item.Equipped))
	hash = storageSnapshotMix(hash, storageBoolSnapshot(item.Damaged))
	hash = storageSnapshotMix(hash, uint64(item.Refine))
	for _, card := range item.Cards {
		hash = storageSnapshotMix(hash, uint64(card))
	}
	return hash
}

func storageBoolSnapshot(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}

func (w *StorageWindow) storageCountText(s *session.Session) string {
	if s == nil {
		return "Num: 0/0"
	}
	return fmt.Sprintf("Num: %d/%d", s.Storage.Amount, s.Storage.MaxAmount)
}

func (w *StorageWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func (w *StorageWindow) ensureSelectedRowSignal() state.Signal[int] {
	if w.selectedRowSignal == nil {
		w.selectedRowSignal = state.NewSignal[int](w.selectedRow)
	}
	return w.selectedRowSignal
}

func (w *StorageWindow) setSelectedRow(row int) {
	w.selectedRow = row
	w.ensureSelectedRowSignal().Set(row)
}

func (w *StorageWindow) itemAt(s *session.Session, mx, my int) (session.InventoryItem, int, bool) {
	tableX := w.x
	tableY := w.y + storageWindowTitleH
	row, ok := storageTableViewRowAt(mx, my, tableX, tableY, storageWindowWidth, int(storageTableViewHeight()), len(sortedStorageItems(s)), w.ensureScrollSignal().Get())
	if !ok {
		return session.InventoryItem{}, 0, false
	}
	items := sortedStorageItems(s)
	if row < 0 || row >= len(items) {
		return session.InventoryItem{}, 0, false
	}
	return items[row], row, true
}

func storageDefaultPosition(ctx Context) (int, int) {
	width, _ := ctx.ScreenSize()
	return maxInt(8, width-storageWindowWidth-24), 118
}

func storageTableViewHeight() float32 {
	return storageTableViewH
}

var storageTableColumns = []rotheme.TableViewColumn{
	{Key: "item", Title: "Item", Flex: 1, MinWidth: 120},
	{Key: "amount", Title: "Qty", Width: 76, Align: widget.TextAlignRight},
}

func storageTableViewRowAt(mx, my, tableX, tableY, tableW, tableH, rowCount int, scrollY float32) (int, bool) {
	if !pointInRect(mx, my, tableX, tableY, scrollbarSafeIntWidth(tableW), tableH) {
		return 0, false
	}
	localY := float32(my-tableY) + scrollY
	row := int(localY / float32(storageRowH))
	if row < 0 || row >= rowCount {
		return 0, false
	}
	return row, true
}

func sortedStorageItems(s *session.Session) []session.InventoryItem {
	if s == nil || len(s.Storage.Items) == 0 {
		return nil
	}
	items := append([]session.InventoryItem(nil), s.Storage.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}
