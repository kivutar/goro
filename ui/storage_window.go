package ui

import (
	"fmt"
	"image"
	"log"
	"sort"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
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
	storageRowH          = 32
	storageRows          = 9
	storageWindowHeight  = storageWindowTitleH + storageRows*storageRowH + storageWindowFooterH
)

type StorageWindow struct {
	window        WindowState
	scroll        int
	snapshot      string
	itemInfo      *ItemInfoWindow
	lastClickItem uint16
	lastClickAt   time.Time
	icons         map[storageItemIconKey]image.Image
	iconMiss      map[storageItemIconKey]struct{}
}

type storageItemIconKey struct {
	itemID     uint16
	identified bool
}

func (w *StorageWindow) SetOpen(open bool) {
	w.ensureWindow()
	if !open {
		w.window.Close()
	}
}

func (w *StorageWindow) OpenWindow(ctx Context) {
	w.ensureWindow()
	w.ClampScroll(ctx.Session)
	w.snapshot = w.storageSnapshot(ctx.Session)
	x, y := storageDefaultPosition(ctx)
	if !w.window.IsOpen() {
		w.window.OpenAt(x, y, w.widgetTree(ctx, nil))
	} else {
		w.window.SetAutoPosition(x, y)
		w.window.SetContent(w.widgetTree(ctx, w.itemInfo))
	}
	w.Publish(ctx)
}

func (w *StorageWindow) Update(ctx Context, itemInfo *ItemInfoWindow) bool {
	w.ensureWindow()
	if !w.window.IsOpen() || ctx.Input == nil {
		return false
	}
	if ctx.Session == nil || !ctx.Session.Storage.Open {
		w.window.Close()
		w.Publish(ctx)
		return false
	}
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.close(ctx)
		w.Publish(ctx)
		return true
	}
	w.ClampScroll(ctx.Session)
	snapshot := w.storageSnapshot(ctx.Session)
	if snapshot != w.snapshot || itemInfo != w.itemInfo {
		w.snapshot = snapshot
		w.itemInfo = itemInfo
		w.window.SetContent(w.widgetTree(ctx, itemInfo))
	}
	consumed := w.window.Update(ctx)
	if !w.window.IsOpen() {
		w.Publish(ctx)
		return consumed
	}
	w.Publish(ctx)
	return consumed
}

func (w *StorageWindow) Draw(screen *render.Image, ctx Context, assets AssetRenderer) {
	w.Publish(ctx)
}

func (w *StorageWindow) Publish(ctx Context) {
	w.ensureWindow()
	if !w.window.IsOpen() {
		w.window.Unpublish(ctx)
		return
	}
	w.window.Publish(ctx)
}

func (w *StorageWindow) AcceptInventoryDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	w.ensureWindow()
	if !w.window.IsOpen() || !pointInRect(mx, my, w.window.x, w.window.y, storageWindowWidth, storageWindowHeight) {
		return false
	}
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		log.Printf("storage deposit failed: not connected")
		return true
	}
	if err := ctx.Network.SendMoveToStorage(item.Index, amount); err != nil {
		log.Printf("storage deposit failed: %v", err)
		return true
	}
	log.Printf("storage deposit requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
	return true
}

func (w *StorageWindow) ensureWindow() {
	if w.window.width == 0 {
		w.window = NewWindowState(storageWindowWidth, storageWindowHeight)
		w.window.SetCloseOnEscape(false)
	}
}

func (w *StorageWindow) widgetTree(ctx Context, itemInfo *ItemInfoWindow) widget.Widget {
	return Window(
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
			primitives.Box(
				newStorageListWidget(storageListConfig{
					items:   visibleStorageItems(sortedStorageItems(ctx.Session), w.scroll),
					icons:   w.visibleItemIcons(ctx),
					names:   w.visibleItemNames(ctx),
					total:   len(sortedStorageItems(ctx.Session)),
					scroll:  w.scroll,
					onWheel: func(delta float32) { w.scrollBy(delta, ctx.Session); w.refresh(ctx, itemInfo) },
					onPress: func(item session.InventoryItem) { w.activateItem(ctx, item) },
					onRightClick: func(item session.InventoryItem, mx, my int) {
						if itemInfo != nil {
							itemInfo.openItem(ctx, item, mx, my)
						}
					},
				}),
			).
				Height(storageRows*storageRowH).
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

func (w *StorageWindow) refresh(ctx Context, itemInfo *ItemInfoWindow) {
	w.ClampScroll(ctx.Session)
	w.snapshot = w.storageSnapshot(ctx.Session)
	w.itemInfo = itemInfo
	w.window.SetContent(w.widgetTree(ctx, itemInfo))
	w.Publish(ctx)
}

func (w *StorageWindow) activateItem(ctx Context, item session.InventoryItem) {
	now := time.Now()
	if w.lastClickItem == item.Index && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.withdraw(ctx, item)
		w.lastClickItem = 0
		w.refresh(ctx, w.itemInfo)
		return
	}
	w.lastClickItem = item.Index
	w.lastClickAt = now
}

func (w *StorageWindow) close(ctx Context) {
	if ctx.Network != nil {
		if err := ctx.Network.SendCloseStorage(); err != nil {
			log.Printf("storage close failed: %v", err)
			return
		}
	}
	w.window.Close()
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
		log.Printf("storage withdraw failed: not connected")
		return
	}
	if err := ctx.Network.SendMoveFromStorage(item.Index, amount); err != nil {
		log.Printf("storage withdraw failed: %v", err)
		return
	}
	log.Printf("storage withdraw requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
}

func (w *StorageWindow) scrollBy(wheelY float32, s *session.Session) {
	if wheelY > 0 {
		w.scroll--
	} else if wheelY < 0 {
		w.scroll++
	}
	w.ClampScroll(s)
}

func (w *StorageWindow) ClampScroll(s *session.Session) {
	maxScroll := maxInt(0, len(sortedStorageItems(s))-storageRows)
	w.scroll = clampWindowInt(w.scroll, 0, maxScroll)
}

func (w *StorageWindow) visibleItemIcons(ctx Context) []image.Image {
	items := visibleStorageItems(sortedStorageItems(ctx.Session), w.scroll)
	icons := make([]image.Image, len(items))
	for i, item := range items {
		icons[i] = w.itemIconImage(ctx.Resources, item)
	}
	return icons
}

func (w *StorageWindow) visibleItemNames(ctx Context) []string {
	items := visibleStorageItems(sortedStorageItems(ctx.Session), w.scroll)
	names := make([]string, len(items))
	for i, item := range items {
		name := inventoryItemDisplayName(ctx.Resources, item)
		if item.Refine > 0 {
			name = fmt.Sprintf("+%d %s", item.Refine, name)
		}
		names[i] = name
	}
	return names
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

func storageDefaultPosition(ctx Context) (int, int) {
	width, _ := ctx.ScreenSize()
	return maxInt(8, width-storageWindowWidth-24), 118
}

type storageListConfig struct {
	items        []session.InventoryItem
	icons        []image.Image
	names        []string
	total        int
	scroll       int
	onWheel      func(float32)
	onPress      func(session.InventoryItem)
	onRightClick func(session.InventoryItem, int, int)
}

type storageListWidget struct {
	widget.WidgetBase
	cfg     storageListConfig
	hovered int
}

func newStorageListWidget(cfg storageListConfig) *storageListWidget {
	w := &storageListWidget{cfg: cfg, hovered: -1}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *storageListWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(storageWindowWidth, storageRows*storageRowH))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *storageListWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	canvas.DrawRect(bounds, rotheme.Default.Colors.PanelBody)
	if len(w.cfg.items) == 0 {
		rotheme.DrawText(canvas, "No items", bounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignCenter)
		w.drawScrollBar(canvas)
		return
	}
	for row, item := range w.cfg.items {
		if row >= storageRows {
			break
		}
		rowBounds := w.rowBounds(row)
		fill := widget.RGBA8(246, 249, 253, 255)
		if row%2 == 1 {
			fill = rotheme.Default.Colors.PanelBody
		}
		if row == w.hovered {
			fill = rotheme.Default.Colors.ButtonHover
		}
		canvas.DrawRect(rowBounds, fill)
		if row > 0 {
			canvas.DrawRect(geometry.NewRect(rowBounds.Min.X, rowBounds.Min.Y, rowBounds.Width()-8, 1), widget.RGBA8(216, 224, 232, 160))
		}
		if row < len(w.cfg.icons) && w.cfg.icons[row] != nil {
			icon := w.cfg.icons[row]
			iconBounds := icon.Bounds()
			canvas.DrawImage(icon, geometry.Pt(rowBounds.Min.X+6, rowBounds.Min.Y+(rowBounds.Height()-float32(iconBounds.Dy()))/2))
		}
		name := fmt.Sprintf("item %d", item.ItemID)
		if row < len(w.cfg.names) && w.cfg.names[row] != "" {
			name = w.cfg.names[row]
		}
		rotheme.DrawText(canvas, name, geometry.NewRect(rowBounds.Min.X+inventoryIconSize+12, rowBounds.Min.Y+4, rowBounds.Width()-inventoryIconSize-70, rowBounds.Height()-8), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignLeft)
		if item.Amount > 1 {
			rotheme.DrawText(canvas, fmt.Sprintf("x%d", item.Amount), geometry.NewRect(rowBounds.Max.X-58, rowBounds.Min.Y+4, 42, rowBounds.Height()-8), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignRight)
		}
	}
	w.drawScrollBar(canvas)
}

func (w *storageListWidget) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.WheelEvent:
		if w.cfg.onWheel != nil {
			w.cfg.onWheel(ev.DeltaY())
		}
		return true
	case *event.MouseEvent:
		row := w.indexAt(ev.Position)
		switch ev.MouseType {
		case event.MouseEnter, event.MouseMove:
			w.hovered = row
			if row >= 0 && row < len(w.cfg.items) {
				ctx.SetCursor(widget.CursorPointer)
			} else {
				ctx.SetCursor(widget.CursorDefault)
			}
			w.SetNeedsRedraw(true)
			return true
		case event.MouseLeave:
			w.hovered = -1
			ctx.SetCursor(widget.CursorDefault)
			w.SetNeedsRedraw(true)
			return false
		case event.MousePress:
			if row < 0 || row >= len(w.cfg.items) {
				return true
			}
			item := w.cfg.items[row]
			switch ev.Button {
			case event.ButtonLeft:
				if w.cfg.onPress != nil {
					w.cfg.onPress(item)
				}
			case event.ButtonRight:
				if w.cfg.onRightClick != nil {
					w.cfg.onRightClick(item, int(ev.GlobalPosition.X), int(ev.GlobalPosition.Y))
				}
			}
			return true
		}
	}
	return true
}

func (w *storageListWidget) rowBounds(row int) geometry.Rect {
	bounds := w.Bounds()
	return geometry.NewRect(bounds.Min.X, bounds.Min.Y+float32(row*storageRowH), bounds.Width(), storageRowH)
}

func (w *storageListWidget) indexAt(point geometry.Point) int {
	local := point.Sub(w.Bounds().Min)
	if local.X < 0 || local.Y < 0 || local.X >= w.Bounds().Width() || local.Y >= storageRows*storageRowH {
		return -1
	}
	row := int(local.Y) / storageRowH
	if row < 0 || row >= storageRows {
		return -1
	}
	return row
}

func (w *storageListWidget) drawScrollBar(canvas widget.Canvas) {
	if w.cfg.total <= storageRows {
		return
	}
	bounds := w.Bounds()
	trackX := bounds.Max.X - 7
	trackH := storageRows*storageRowH - 4
	canvas.DrawRect(geometry.NewRect(trackX, bounds.Min.Y+2, 4, float32(trackH)), rotheme.Default.Colors.Button)
	maxScroll := maxInt(1, w.cfg.total-storageRows)
	thumbH := maxInt(18, trackH*storageRows/w.cfg.total)
	thumbTravel := trackH - thumbH
	thumbY := bounds.Min.Y + 2 + float32(thumbTravel*w.cfg.scroll/maxScroll)
	canvas.DrawRect(geometry.NewRect(trackX, thumbY, 4, float32(thumbH)), rotheme.Default.Colors.MutedText)
}

func visibleStorageItems(items []session.InventoryItem, scroll int) []session.InventoryItem {
	if scroll < 0 {
		scroll = 0
	}
	if scroll >= len(items) {
		return nil
	}
	end := minInt(len(items), scroll+storageRows)
	return items[scroll:end]
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
