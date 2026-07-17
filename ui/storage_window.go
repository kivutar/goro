package ui

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"sort"
	"time"

	"github.com/gogpu/ui/core/datatable"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	storageWindowWidth   = 312
	storageWindowTitleH  = ROWindowTitleHeight
	storageWindowFooterH = 38
	storageTableHeaderH  = 36
	storageRowH          = 32
	storageRows          = 9
	storageWindowHeight  = storageWindowTitleH + storageTableHeaderH + storageRows*storageRowH + storageWindowFooterH
)

type StorageWindow struct {
	Window
	scrollY       state.Signal[float32]
	selectedRow   int
	snapshot      string
	itemInfo      *ItemInfoWindow
	lastClickItem uint16
	lastClickAt   time.Time
	dragItem      session.InventoryItem
	dragActive    bool
	dragFrom      time.Time
	icons         map[storageItemIconKey]image.Image
	iconMiss      map[storageItemIconKey]struct{}
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
	w.selectedRow = -1
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
		FooterHeight(storageWindowFooterH),
		FooterPadding(10),
		Content(
			primitives.Box(w.storageTableWidget(ctx)).
				Height(storageTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.HBox(
				rotheme.Text(w.storageCountText(ctx.Session)),
				primitives.Expanded(primitives.Box()),
			).
				CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *StorageWindow) storageTableWidget(ctx Context) *datatable.Widget {
	items := sortedStorageItems(ctx.Session)
	rows := w.storageRows(ctx, items)
	return datatable.New(
		datatable.Columns([]datatable.Column{
			{Key: "item", Title: "Item", Width: 236},
			{Key: "amount", Title: "Qty", Width: 76, Align: widget.TextAlignRight},
		}),
		datatable.RowCount(len(rows)),
		datatable.RowHeight(storageRowH),
		datatable.ScrollYSignal(w.ensureScrollSignal()),
		datatable.SelectionModeOpt(datatable.SelectionSingle),
		datatable.SelectedRow(w.selectedRow),
		datatable.PainterOpt(storageTablePainter{icons: w.storageItemIcons(ctx, items)}),
		datatable.CellValue(func(row int, col string) string {
			if row < 0 || row >= len(rows) {
				return ""
			}
			switch col {
			case "item":
				return rows[row].name
			case "amount":
				return rows[row].amount
			default:
				return ""
			}
		}),
		datatable.OnRowSelect(func(row int) {
			if row >= 0 && row < len(rows) {
				w.selectedRow = row
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
	items := sortedStorageItems(s)
	if w.selectedRow >= len(items) {
		w.selectedRow = -1
	}
	scroll := w.ensureScrollSignal()
	maxScroll := float32(maxInt(0, len(items)-storageRows) * storageRowH)
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

func (w *StorageWindow) storageSnapshot(s *session.Session) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%d/%d:%v", s.Storage.Amount, s.Storage.MaxAmount, sortedStorageItems(s))
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

func (w *StorageWindow) itemAt(s *session.Session, mx, my int) (session.InventoryItem, int, bool) {
	tableX := w.x
	tableY := w.y + storageWindowTitleH
	row, ok := storageTableRowAt(mx, my, tableX, tableY, storageWindowWidth, int(storageTableHeight()), len(sortedStorageItems(s)), w.ensureScrollSignal().Get())
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

func storageTableHeight() float32 {
	return storageTableHeaderH + storageRows*storageRowH
}

func storageTableRowAt(mx, my, tableX, tableY, tableW, tableH, rowCount int, scrollY float32) (int, bool) {
	if !pointInRect(mx, my, tableX, tableY+storageTableHeaderH, tableW, tableH-storageTableHeaderH) {
		return 0, false
	}
	localY := float32(my-tableY) - storageTableHeaderH + scrollY
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

type storageTablePainter struct {
	datatable.DefaultPainter
	icons []image.Image
}

func (p storageTablePainter) PaintHeader(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderPaintState) {
	if bounds.IsEmpty() {
		return
	}
	canvas.DrawRect(bounds, rotheme.Default.Colors.PanelBody)
}

func (p storageTablePainter) PaintHeaderCell(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderCellPaintState) {
}

func (p storageTablePainter) PaintRow(canvas widget.Canvas, s datatable.RowPaintState) {
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

func (p storageTablePainter) PaintCell(canvas widget.Canvas, s datatable.CellPaintState) {
	color := rotheme.Default.Colors.Text
	if s.ColIndex == 1 {
		color = rotheme.Default.Colors.MutedText
	}
	textBounds := geometry.NewRect(s.Bounds.Min.X+4, s.Bounds.Min.Y+4, s.Bounds.Width()-8, s.Bounds.Height()-8)
	if s.ColIndex == 0 && s.RowIndex >= 0 && s.RowIndex < len(p.icons) && p.icons[s.RowIndex] != nil {
		icon := p.icons[s.RowIndex]
		iconBounds := icon.Bounds()
		iconW := float32(iconBounds.Dx())
		iconH := float32(iconBounds.Dy())
		canvas.DrawImage(icon, geometry.Pt(s.Bounds.Min.X+6, s.Bounds.Min.Y+(s.Bounds.Height()-iconH)/2))
		textBounds = geometry.NewRect(s.Bounds.Min.X+iconW+12, s.Bounds.Min.Y+4, s.Bounds.Width()-iconW-16, s.Bounds.Height()-8)
	}
	rotheme.DrawText(canvas, s.Value, textBounds, rotheme.Default.Typography.TextSize, color, false, s.Align)
}

func (p storageTablePainter) PaintEmptyState(canvas widget.Canvas, bounds geometry.Rect) {
	rotheme.DrawText(canvas, "No items", bounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignCenter)
}
