package ui

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"sort"
	"time"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	cartGridCols     = 7
	cartGridRows     = 4
	cartGridCell     = inventoryBagCell
	cartGridW        = cartGridCols * cartGridCell
	cartGridViewW    = cartGridW + ROScrollbarGutter
	cartGridViewH    = cartGridRows * cartGridCell
	cartGridLeftPad  = 40
	cartWindowWidth  = cartGridLeftPad + cartGridViewW
	cartWindowHeight = ROWindowTitleHeight + cartGridViewH + ROWindowFooterHeight
)

type CartWindow struct {
	Window
	scrollY       state.Signal[float32]
	snapshot      uint64
	itemInfo      *ItemWindows
	lastClickItem uint16
	lastClickAt   time.Time
	dragItem      session.InventoryItem
	dragActive    bool
	dragFrom      time.Time
	tooltip       tooltipState
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
		w.hideTooltip()
		w.Window.Close()
	}
}

func (w *CartWindow) OpenWindow(ctx Context) {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	w.ClampScroll(ctx.Session)
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

func (w *CartWindow) Update(ctx Context, inventory *InventoryBagWindow, storage *StorageWindow, itemInfo *ItemWindows) bool {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	if !w.IsOpen() || ctx.Input == nil {
		w.hideTooltip()
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
	consumed := w.Window.Update(ctx)
	if !w.IsOpen() {
		w.hideTooltip()
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

func (w *CartWindow) DrawTooltip(ctx Context, screen *render.Frame) {
	if w.dragActive {
		return
	}
	w.tooltip.Draw(ctx, screen)
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

func (w *CartWindow) widgetTree(ctx Context, itemInfo *ItemWindows) widget.Widget {
	items := sortedCartItems(ctx.Session)
	grid := newInventoryGridWidget(inventoryGridConfig{
		items:     items,
		icons:     w.cartItemIcons(ctx, items),
		amounts:   inventoryGridAmountLabels(items),
		cols:      cartGridCols,
		minRows:   cartGridRows,
		cellSize:  cartGridCell,
		viewWidth: cartGridViewW,
		onPress: func(item session.InventoryItem) {
			w.startItemDragOrWithdraw(ctx, item)
		},
		onHover: func(item session.InventoryItem) {
			w.showTooltip(ctx, item)
		},
		onLeave: func() {
			w.hideTooltip()
		},
		onRightClick: func(item session.InventoryItem, mx, my int) {
			w.hideTooltip()
			w.dragActive = false
			w.dragItem = session.InventoryItem{}
			if itemInfo != nil {
				itemInfo.openItem(ctx, item, mx, my)
			}
		},
	})
	scroll := scrollview.New(
		grid,
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
		scrollview.ScrollYSignal(w.ensureScrollSignal()),
		scrollview.ScrollStep(cartGridCell),
	)
	return Win(
		Title("Pushcart"),
		CloseButton(true),
		OnClose(func() {
			w.close(ctx)
			w.Publish(ctx)
		}),
		Size(cartWindowWidth, cartWindowHeight),
		Content(
			primitives.HBox(
				primitives.Box().
					Width(cartGridLeftPad).
					Height(cartGridViewH).
					Background(rotheme.Default.Colors.WindowBody),
				primitives.Box(scroll).
					Width(cartGridViewW).
					Height(cartGridViewH),
			).
				Gap(0).
				Background(rotheme.Default.Colors.WindowBody),
		),
		Footer(
			rotheme.Text(w.cartCountText(ctx.Session)),
			primitives.Expanded(primitives.Box()),
			rotheme.Text(w.cartWeightText(ctx.Session)),
		),
	)
}

func (w *CartWindow) startItemDragOrWithdraw(ctx Context, item session.InventoryItem) {
	w.hideTooltip()
	now := time.Now()
	if w.lastClickItem == item.Index && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.withdraw(ctx, item)
		w.lastClickItem = 0
		w.refresh(ctx, w.itemInfo)
		return
	}
	w.lastClickItem = item.Index
	w.lastClickAt = now
	w.dragItem = item
	w.dragActive = true
	w.dragFrom = now
	w.refresh(ctx, w.itemInfo)
}

func (w *CartWindow) close(ctx Context) {
	w.Window.Close()
	w.dragActive = false
	w.hideTooltip()
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

func (w *CartWindow) refresh(ctx Context, itemInfo *ItemWindows) {
	w.EnsureWindow(cartWindowWidth, cartWindowHeight)
	w.ClampScroll(ctx.Session)
	w.snapshot = w.cartSnapshot(ctx.Session)
	w.itemInfo = itemInfo
	w.SetContent(w.widgetTree(ctx, itemInfo))
	w.Publish(ctx)
}

func (w *CartWindow) Refresh(ctx Context, itemInfo *ItemWindows) {
	w.refresh(ctx, itemInfo)
}

func (w *CartWindow) Rebind(ctx Context, itemInfo *ItemWindows) {
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
	scroll := w.ensureScrollSignal()
	contentHeight := inventoryGridTotalRows(itemCount, cartGridCols, cartGridRows) * cartGridCell
	maxScroll := float32(maxInt(0, contentHeight-cartGridViewH))
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
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

func (w *CartWindow) showTooltip(ctx Context, item session.InventoryItem) {
	if item.ItemID == 0 || w.dragActive || ctx.Input == nil {
		w.hideTooltip()
		return
	}
	text := fmt.Sprintf("%s: %d ea", inventoryItemDisplayName(ctx.Resources, item), maxInt(1, item.Amount))
	w.tooltip.Show(ctx, text, ctx.Input.MouseX, ctx.Input.MouseY+18, ctx.Input.MouseY-6)
}

func (w *CartWindow) hideTooltip() {
	w.tooltip.Hide()
}

func cartDefaultPosition(ctx Context) (int, int) {
	width, _ := ctx.ScreenSize()
	return maxInt(windowScreenMargin, width-cartWindowWidth-windowScreenMargin), 118
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
