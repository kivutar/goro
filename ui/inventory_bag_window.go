package ui

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	inventoryBagTabW    = 32
	inventoryBagTabH    = 44
	inventoryBagCell    = 32
	inventoryBagIcon    = 24
	inventoryBagCols    = 8
	inventoryBagRows    = 5
	inventoryBagTabOver = 1
	inventoryBagTabRail = inventoryBagTabW + inventoryBagTabOver*2
	inventoryBagGridW   = inventoryBagCols * inventoryBagCell
	inventoryBagWidth   = characterWindowWidth
	inventoryBagViewW   = inventoryBagWidth - inventoryBagTabRail - verticalTabDividerW
	inventoryBagViewH   = inventoryBagRows * inventoryBagCell
	inventoryBagHeight  = ROWindowTitleHeight + inventoryBagViewH

	inventoryBagCellShadowInset  float32 = 3
	inventoryBagCellShadowRadius float32 = 3
	inventoryBagCellShadowAlpha  float32 = 0.4
	inventoryBagCellHoverAlpha   float32 = 0.75
)

const (
	inventoryBagTabItem = iota
	inventoryBagTabEquip
	inventoryBagTabEtc
)

const inventoryBagCartOptionMask uint32 = 0x00000008 | 0x00000080 | 0x00000100 | 0x00000200 | 0x00000400

var inventoryBagTabs = []struct {
	label string
	tab   int
}{
	{label: "Item", tab: inventoryBagTabItem},
	{label: "Equip", tab: inventoryBagTabEquip},
	{label: "Etc", tab: inventoryBagTabEtc},
}

type InventoryBagWindow struct {
	Window
	tab           int
	scrollY       state.Signal[float32]
	snapshot      string
	itemInfo      *ItemInfoWindow
	lastClickItem uint16
	lastClickAt   time.Time
	dragItem      session.InventoryItem
	dragActive    bool
	dragFrom      time.Time
	amountPrompt  amountPrompt
	pendingCard   uint16
	tooltip       tooltipState
	icons         map[inventoryBagIconKey]image.Image
	iconMiss      map[inventoryBagIconKey]struct{}
}

type inventoryBagIconKey struct {
	itemID     uint16
	identified bool
}

func (w *InventoryBagWindow) Toggle(ctx Context) {
	w.EnsureWindow(inventoryBagWidth, inventoryBagHeight)
	if w.IsOpen() {
		w.hideTooltip()
		w.amountPrompt.Close(ctx)
		w.Window.Close()
		w.Publish(ctx)
		return
	}
	w.selectFirstNonEmptyTab(ctx.Session)
	w.ClampScroll(ctx.Session)
	w.snapshot = w.inventorySnapshot(ctx.Session)
	x, y := inventoryBagDefaultPosition(ctx)
	w.OpenAt(x, y, w.widgetTree(ctx, nil))
	w.Publish(ctx)
}

func (w *InventoryBagWindow) Update(ctx Context, shortcuts *ShortcutBar, storage *StorageWindow, cart *CartWindow, trade *TradeWindow, equipment *EquipmentWindow, itemInfo *ItemInfoWindow, dropTargets ...InventoryDropTarget) bool {
	w.EnsureWindow(inventoryBagWidth, inventoryBagHeight)
	if !w.IsOpen() || ctx.Input == nil {
		w.hideTooltip()
		return false
	}
	if w.UpdateDrag(ctx, shortcuts, storage, cart, trade, equipment, dropTargets...) {
		return true
	}
	w.ClampScroll(ctx.Session)
	snapshot := w.inventorySnapshot(ctx.Session)
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

type InventoryDropTarget interface {
	AcceptInventoryDrop(Context, session.InventoryItem, int, int) bool
}

func (w *InventoryBagWindow) UpdateDrag(ctx Context, shortcuts *ShortcutBar, storage *StorageWindow, cart *CartWindow, trade *TradeWindow, equipment *EquipmentWindow, dropTargets ...InventoryDropTarget) bool {
	if w.UpdateDropPrompt(ctx) {
		return true
	}
	if !w.dragActive || ctx.Input == nil {
		return false
	}
	if ctx.Input.MouseJustReleased(input.MouseButtonLeft) || !ctx.Input.MousePressed(input.MouseButtonLeft) {
		item := w.dragItem
		w.dragActive = false
		w.dragItem = session.InventoryItem{}
		for _, target := range dropTargets {
			if target != nil && target.AcceptInventoryDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
				return true
			}
		}
		if storage != nil && storage.AcceptInventoryDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if cart != nil && cart.AcceptInventoryDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if trade != nil && trade.AcceptInventoryDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if equipment != nil && equipment.AcceptInventoryDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if shortcuts != nil && shortcuts.AcceptItemDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if manager, ok := ctx.UIManager.(interface{ PointerBlocked(int, int) bool }); ok && manager.PointerBlocked(ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if !w.pointInside(ctx.Input.MouseX, ctx.Input.MouseY) {
			w.requestDrop(ctx, item)
			return true
		}
		return true
	}
	return true
}

func (w *InventoryBagWindow) UpdateDropPrompt(ctx Context) bool {
	return w != nil && w.amountPrompt.Update(ctx)
}

func (w *InventoryBagWindow) requestDrop(ctx Context, item session.InventoryItem) {
	maxAmount := inventoryDropMaxAmount(item)
	if inventoryItemTypeStackable(item.Type) && maxAmount > 1 {
		w.amountPrompt.Open(ctx, "Enter amount to drop", maxAmount, maxAmount, func(amount uint16) {
			w.sendDrop(ctx, item, amount)
		})
		return
	}
	w.sendDrop(ctx, item, 1)
}

func (w *InventoryBagWindow) sendDrop(ctx Context, item session.InventoryItem, requested uint16) {
	current, ok := currentInventoryDropItem(ctx.Session, item)
	if !ok {
		glog.Warnf("inventory drop cancelled: item no longer available index=%d item=%d", item.Index, item.ItemID)
		return
	}
	item = current
	amount := inventoryDropAmount(item, requested)
	if err := dropInventoryItem(ctx, item, amount); err != nil {
		glog.Warnf("inventory drop failed: %v", err)
		return
	}
	glog.Debugf("inventory drop requested index=%d item=%d amount=%d", item.Index, item.ItemID, amount)
}

func (w *InventoryBagWindow) KeyboardShortcutsBlocked() bool {
	return w != nil && w.amountPrompt.IsOpen()
}

func (w *InventoryBagWindow) Draw(screen *render.Frame, ctx Context, assets AssetProvider) {
	w.Publish(ctx)
}

func (w *InventoryBagWindow) DrawTooltip(ctx Context, screen *render.Frame) {
	if w.dragActive {
		return
	}
	w.tooltip.Draw(ctx, screen)
}

func (w *InventoryBagWindow) DrawDragGhost(screen *render.Frame, ctx Context, assets AssetProvider) {
	if !w.dragActive || screen == nil || ctx.Input == nil || assets == nil {
		return
	}
	if time.Since(w.dragFrom) <= 80*time.Millisecond {
		return
	}
	assets.DrawInventoryItemIcon(screen, ctx.Resources, w.dragItem, ctx.Input.MouseX-inventoryIconSize/2, ctx.Input.MouseY-inventoryIconSize/2)
}

func (w *InventoryBagWindow) Rebind(ctx Context, itemInfo *ItemInfoWindow) {
	w.EnsureWindow(inventoryBagWidth, inventoryBagHeight)
	if !w.IsOpen() {
		return
	}
	w.refresh(ctx, itemInfo)
}

func (w *InventoryBagWindow) PendingCardIndex() uint16 {
	return w.pendingCard
}

func (w *InventoryBagWindow) widgetTree(ctx Context, itemInfo *ItemInfoWindow) widget.Widget {
	items := w.tabItems(ctx.Session)
	grid := newInventoryGridWidget(inventoryGridConfig{
		items:     items,
		icons:     w.itemIcons(ctx, items),
		amounts:   inventoryGridAmountLabels(items),
		viewWidth: inventoryBagViewW,
		onPress:   func(item session.InventoryItem) { w.startItemDragOrActivate(ctx, item) },
		onHover:   func(item session.InventoryItem) { w.showTooltip(ctx, item) },
		onLeave:   func() { w.hideTooltip() },
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
		scrollview.ScrollStep(inventoryBagCell),
	)
	return Win(
		Title("Inventory"),
		CloseButton(true),
		OnClose(func() {
			w.amountPrompt.Close(ctx)
			w.Window.Close()
			w.Publish(ctx)
		}),
		Size(inventoryBagWidth, inventoryBagHeight),
		Content(
			verticalTabFrame(
				w.tabColumn(ctx),
				primitives.Box(scroll).
					Width(inventoryBagViewW).
					Height(inventoryBagViewH),
			),
		),
	)
}

func (w *InventoryBagWindow) tabColumn(ctx Context) widget.Widget {
	tabs := make([]widget.Widget, 0, len(inventoryBagTabs))
	for _, tab := range inventoryBagTabs {
		tab := tab
		tabs = append(tabs, newTabWidget(tabWidgetConfig{
			label:         tab.label,
			labelRotation: rotheme.TextRotationCounterClockwise,
			active:        tab.tab == w.tab,
			width:         inventoryBagTabW + inventoryBagTabOver*2,
			height:        inventoryBagTabH,
			onClick: func() {
				w.hideTooltip()
				w.tab = tab.tab
				w.ensureScrollSignal().Set(0)
				w.lastClickItem = 0
				w.refresh(ctx, w.itemInfo)
			},
		}))
	}
	return primitives.Box(tabs...).
		Width(inventoryBagTabRail).
		Height(inventoryBagViewH).
		Gap(-inventoryBagTabOver)
}

func (w *InventoryBagWindow) refresh(ctx Context, itemInfo *ItemInfoWindow) {
	w.hideTooltip()
	w.ClampScroll(ctx.Session)
	w.snapshot = w.inventorySnapshot(ctx.Session)
	w.itemInfo = itemInfo
	w.SetContent(w.widgetTree(ctx, itemInfo))
	w.Publish(ctx)
}

func inventoryBagHasCart(ctx Context) bool {
	if ctx.World != nil && ctx.World.Player.HasCartState {
		return ctx.World.Player.HasCart
	}
	if selectedCharacter(ctx.Session).Option&inventoryBagCartOptionMask != 0 {
		return true
	}
	return ctx.Session != nil && (ctx.Session.Cart.MaxAmount > 0 || len(ctx.Session.Cart.Items) > 0)
}

func (w *InventoryBagWindow) startItemDragOrActivate(ctx Context, item session.InventoryItem) {
	w.hideTooltip()
	now := time.Now()
	if w.lastClickItem == item.Index && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.dragActive = false
		w.dragItem = session.InventoryItem{}
		w.activateItem(ctx, item)
		w.lastClickItem = 0
		w.refresh(ctx, w.itemInfo)
		return
	}
	w.dragItem = item
	w.dragActive = true
	w.dragFrom = now
	w.lastClickItem = item.Index
	w.lastClickAt = now
}

func (w *InventoryBagWindow) showTooltip(ctx Context, item session.InventoryItem) {
	if item.ItemID == 0 || w.dragActive || ctx.Input == nil {
		w.hideTooltip()
		return
	}
	text := inventoryItemDisplayName(ctx.Resources, item)
	w.tooltip.Show(ctx, text, ctx.Input.MouseX, ctx.Input.MouseY+18, ctx.Input.MouseY-6)
}

func (w *InventoryBagWindow) hideTooltip() {
	w.tooltip.Hide()
}

func (w *InventoryBagWindow) pointInside(x, y int) bool {
	return pointInRect(x, y, w.x, w.y, inventoryBagWidth, inventoryBagHeight)
}

func (w *InventoryBagWindow) AcceptStorageDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	w.EnsureWindow(inventoryBagWidth, inventoryBagHeight)
	return w.IsOpen() && w.pointInside(mx, my)
}

func inventoryBagDefaultPosition(ctx Context) (int, int) {
	width, height := ctx.ScreenSize()
	menuX, menuY, _, menuH := basicMenuBounds()
	x := clampWindowInt(menuX, windowScreenMargin, maxInt(windowScreenMargin, width-inventoryBagWidth-windowScreenMargin))
	y := clampWindowInt(menuY+menuH+8, windowScreenMargin, maxInt(windowScreenMargin, height-inventoryBagHeight-windowScreenMargin))
	return x, y
}

func (w *InventoryBagWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
	if manager == nil || item.ItemID == 0 {
		return nil
	}
	key := inventoryBagIconKey{itemID: item.ItemID, identified: item.Identified}
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
		w.icons = make(map[inventoryBagIconKey]image.Image)
	}
	w.icons[key] = img
	return img
}

func (w *InventoryBagWindow) itemIcons(ctx Context, items []session.InventoryItem) []image.Image {
	icons := make([]image.Image, len(items))
	for i, item := range items {
		icons[i] = w.itemIconImage(ctx.Resources, item)
	}
	return icons
}

func (w *InventoryBagWindow) markIconMiss(key inventoryBagIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[inventoryBagIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func (w *InventoryBagWindow) activateItem(ctx Context, item session.InventoryItem) {
	if item.Type == db.ItemTypeCard {
		if ctx.Network == nil {
			glog.Warnf("card composition failed: not connected")
			return
		}
		candidates := cardCompositionCandidateItems(ctx, item)
		if len(candidates) == 0 {
			glog.Debugf("card composition has no local compatible equipment card_index=%d item=%d location=0x%04X", item.Index, item.ItemID, item.Location)
			logCardCompositionRejectedItems(ctx, item)
		} else {
			glog.Debugf("card composition local compatible equipment card_index=%d item=%d indexes=%v", item.Index, item.ItemID, inventoryItemIndexes(candidates))
		}
		w.pendingCard = item.Index
		if err := ctx.Network.SendItemCompositionList(item.Index); err != nil {
			glog.Warnf("card composition list failed: %v", err)
			return
		}
		glog.Debugf("card composition list requested index=%d item=%d", item.Index, item.ItemID)
		return
	}
	if inventoryItemCanEquip(item) {
		equipInventoryItem(ctx, item)
		return
	}
	if !db.ItemTypeIsUsable(item.Type) {
		glog.Debugf("inventory use skipped: item cannot be used index=%d item=%d type=%d", item.Index, item.ItemID, item.Type)
		return
	}
	if err := UseInventoryItem(ctx, item); err != nil {
		glog.Warnf("inventory use failed: %v", err)
		return
	}
	glog.Debugf("inventory use requested index=%d item=%d type=%d", item.Index, item.ItemID, item.Type)
}

func (w *InventoryBagWindow) ClampScroll(s *session.Session) {
	scroll := w.ensureScrollSignal()
	contentHeight := inventoryBagTotalRows(len(w.tabItems(s))) * inventoryBagCell
	maxScroll := float32(maxInt(0, contentHeight-inventoryBagViewH))
	value := scroll.Get()
	switch {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *InventoryBagWindow) selectFirstNonEmptyTab(s *session.Session) {
	if len(w.tabItems(s)) > 0 {
		return
	}
	original := w.tab
	for _, tab := range inventoryBagTabs {
		if tab.tab == w.tab {
			continue
		}
		w.tab = tab.tab
		if len(w.tabItems(s)) > 0 {
			w.ensureScrollSignal().Set(0)
			return
		}
	}
	w.tab = original
}

func (w *InventoryBagWindow) tabItems(s *session.Session) []session.InventoryItem {
	items := sortedInventoryItems(s)
	if len(items) == 0 {
		return nil
	}
	filtered := items[:0]
	for _, item := range items {
		if inventoryItemTab(item) == w.tab {
			filtered = append(filtered, item)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Index < filtered[j].Index
	})
	return filtered
}

func (w *InventoryBagWindow) inventorySnapshot(s *session.Session) string {
	var b strings.Builder
	fmt.Fprintf(&b, "tab=%d;", w.tab)
	for _, item := range sortedInventoryItems(s) {
		fmt.Fprintf(&b, "%d:%d:%d:%d:%d:%t:%t;", item.Index, item.ItemID, item.Type, item.Amount, item.Location, item.Equip, item.Equipped)
	}
	return b.String()
}

func (w *InventoryBagWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func inventoryBagTotalRows(itemCount int) int {
	return inventoryGridTotalRows(itemCount, inventoryBagCols, inventoryBagRows)
}

func inventoryGridTotalRows(itemCount, cols, minRows int) int {
	if cols <= 0 {
		cols = 1
	}
	rows := (itemCount + cols - 1) / cols
	if rows < minRows {
		return minRows
	}
	return rows
}

func inventoryGridAmountLabels(items []session.InventoryItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		if item.Amount > 1 {
			labels[i] = strconv.Itoa(item.Amount)
		}
	}
	return labels
}

type inventoryGridConfig struct {
	items        []session.InventoryItem
	icons        []image.Image
	amounts      []string
	cols         int
	minRows      int
	cellSize     int
	viewWidth    int
	onPress      func(session.InventoryItem)
	onHover      func(session.InventoryItem)
	onLeave      func()
	onRightClick func(session.InventoryItem, int, int)
}

type inventoryGridWidget struct {
	widget.WidgetBase
	cfg     inventoryGridConfig
	hovered int
}

func newInventoryGridWidget(cfg inventoryGridConfig) *inventoryGridWidget {
	w := &inventoryGridWidget{cfg: cfg, hovered: -1}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *inventoryGridWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(float32(w.viewWidth()), float32(w.totalRows()*w.cellSize())))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *inventoryGridWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	canvas.DrawRect(bounds, rotheme.Default.Colors.WindowBody)
	startRow, endRow := w.visibleRows(canvas)
	cols := w.cols()
	for row := startRow; row < endRow; row++ {
		for col := 0; col < cols; col++ {
			index := row*cols + col
			cell := w.cellBounds(index)
			drawInventoryGridCellShadow(canvas, cell, index == w.hovered)
		}
	}
	startIndex := startRow * cols
	endIndex := minInt(len(w.cfg.items), endRow*cols)
	for i := startIndex; i < endIndex; i++ {
		item := w.cfg.items[i]
		cell := w.cellBounds(i)
		if i < len(w.cfg.icons) && w.cfg.icons[i] != nil {
			canvas.DrawImage(w.cfg.icons[i], geometry.Pt(cell.Min.X+4, cell.Min.Y+4))
		}
		if item.Amount > 1 {
			amount := ""
			if i < len(w.cfg.amounts) {
				amount = w.cfg.amounts[i]
			}
			if amount == "" {
				amount = strconv.Itoa(item.Amount)
			}
			rotheme.DrawText(canvas, amount, geometry.NewRect(cell.Max.X-18, cell.Max.Y-15, 16, 12), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignRight)
		}
		if item.Equipped {
			rotheme.DrawText(canvas, "E", geometry.NewRect(cell.Min.X+2, cell.Min.Y+2, 12, 12), rotheme.Default.Typography.TextSize, widget.RGBA8(54, 128, 76, 255), false, widget.TextAlignLeft)
		}
	}
}

func drawInventoryGridCellShadow(canvas widget.Canvas, cell geometry.Rect, hovered bool) {
	width := cell.Width() - inventoryBagCellShadowInset*2
	height := (cell.Height() - inventoryBagCellShadowInset*2) / 2
	if width <= 0 || height <= 0 {
		return
	}
	shadow := geometry.NewRect(
		cell.Min.X+inventoryBagCellShadowInset,
		cell.Max.Y-inventoryBagCellShadowInset-height,
		width,
		height,
	)
	color := rotheme.Default.Colors.ButtonHover
	color.A = inventoryBagCellShadowAlpha
	if hovered {
		color.A = inventoryBagCellHoverAlpha
	}
	canvas.DrawRoundRect(shadow, color, min(inventoryBagCellShadowRadius, height/2))
}

func (w *inventoryGridWidget) visibleRows(canvas widget.Canvas) (int, int) {
	rows := w.totalRows()
	if rows <= 0 {
		return 0, 0
	}
	clip := canvas.ClipBounds()
	if clip.IsEmpty() {
		return 0, 0
	}
	offset := canvas.TransformOffset()
	cellSize := w.cellSize()
	localTop := clip.Min.Y - offset.Y
	localBottom := clip.Max.Y - offset.Y
	if localBottom <= 0 || localTop >= float32(rows*cellSize) {
		return 0, 0
	}
	start := maxInt(0, int(localTop)/cellSize)
	end := minInt(rows, int(localBottom)/cellSize+1)
	if end < start {
		end = start
	}
	return start, end
}

func (w *inventoryGridWidget) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		index := w.indexAt(ev.Position)
		switch ev.MouseType {
		case event.MouseEnter, event.MouseMove:
			w.hovered = index
			if index >= 0 && index < len(w.cfg.items) {
				if w.cfg.onHover != nil {
					w.cfg.onHover(w.cfg.items[index])
				}
				ctx.SetCursor(widget.CursorPointer)
			} else {
				if w.cfg.onLeave != nil {
					w.cfg.onLeave()
				}
				ctx.SetCursor(widget.CursorDefault)
			}
			return true
		case event.MouseLeave:
			w.hovered = -1
			if w.cfg.onLeave != nil {
				w.cfg.onLeave()
			}
			ctx.SetCursor(widget.CursorDefault)
			return false
		case event.MousePress:
			if index < 0 || index >= len(w.cfg.items) {
				return true
			}
			item := w.cfg.items[index]
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
		return true
	}
	return false
}

func (w *inventoryGridWidget) cellBounds(index int) geometry.Rect {
	bounds := w.Bounds()
	cellSize := w.cellSize()
	col := index % w.cols()
	row := index / w.cols()
	return geometry.NewRect(
		bounds.Min.X+float32(col*cellSize),
		bounds.Min.Y+float32(row*cellSize),
		float32(cellSize-1),
		float32(cellSize-1),
	)
}

func (w *inventoryGridWidget) indexAt(point geometry.Point) int {
	local := point.Sub(w.Bounds().Min)
	cellSize := w.cellSize()
	if local.X < 0 || local.Y < 0 || local.X >= float32(w.gridWidth()) || local.Y >= float32(w.totalRows()*cellSize) {
		return -1
	}
	col := int(local.X) / cellSize
	row := int(local.Y) / cellSize
	index := row*w.cols() + col
	if index < 0 || index >= len(w.cfg.items) {
		return -1
	}
	return index
}

func (w *inventoryGridWidget) cols() int {
	if w.cfg.cols > 0 {
		return w.cfg.cols
	}
	return inventoryBagCols
}

func (w *inventoryGridWidget) minRows() int {
	if w.cfg.minRows > 0 {
		return w.cfg.minRows
	}
	return inventoryBagRows
}

func (w *inventoryGridWidget) cellSize() int {
	if w.cfg.cellSize > 0 {
		return w.cfg.cellSize
	}
	return inventoryBagCell
}

func (w *inventoryGridWidget) gridWidth() int {
	return w.cols() * w.cellSize()
}

func (w *inventoryGridWidget) viewWidth() int {
	if w.cfg.viewWidth > 0 {
		return w.cfg.viewWidth
	}
	return w.gridWidth() + ROScrollbarGutter
}

func (w *inventoryGridWidget) totalRows() int {
	return inventoryGridTotalRows(len(w.cfg.items), w.cols(), w.minRows())
}

func inventoryItemTab(item session.InventoryItem) int {
	if inventoryItemUsesEquipmentTab(item) {
		return inventoryBagTabEquip
	}
	if db.ItemTypeIsUsable(item.Type) {
		return inventoryBagTabItem
	}
	return inventoryBagTabEtc
}

func inventoryItemCanEquip(item session.InventoryItem) bool {
	if item.Type == db.ItemTypeCard {
		return false
	}
	return item.Equip || inventoryItemTypeCanEquip(item.Type)
}

func inventoryItemUsesEquipmentTab(item session.InventoryItem) bool {
	switch item.Type {
	case db.ItemTypeCard, db.ItemTypeAmmo:
		return false
	default:
		return item.Equip || inventoryItemTypeUsesEquipmentTab(item.Type)
	}
}

func inventoryItemTypeCanEquip(itemType uint8) bool {
	switch itemType {
	case db.ItemTypeArmor, db.ItemTypeWeapon, db.ItemTypePetEgg, db.ItemTypePetArmor, db.ItemTypeAmmo, db.ItemTypeShadowGear:
		return true
	default:
		return false
	}
}

func inventoryItemTypeUsesEquipmentTab(itemType uint8) bool {
	switch itemType {
	case db.ItemTypeArmor, db.ItemTypeWeapon, db.ItemTypePetEgg, db.ItemTypePetArmor, db.ItemTypeShadowGear:
		return true
	default:
		return false
	}
}

func cardCompositionCandidateItems(ctx Context, card session.InventoryItem) []session.InventoryItem {
	if ctx.Session == nil || ctx.Resources == nil || card.Type != db.ItemTypeCard || card.Location == 0 {
		return nil
	}
	candidates := make([]session.InventoryItem, 0, len(ctx.Session.Inventory.Items))
	for _, item := range ctx.Session.Inventory.Items {
		if ok, _ := cardCompositionCanTarget(ctx, card, item); !ok {
			continue
		}
		candidates = append(candidates, item)
	}
	return candidates
}

func cardCompositionCanTarget(ctx Context, card, item session.InventoryItem) (bool, string) {
	switch item.Type {
	case db.ItemTypeWeapon, db.ItemTypeArmor:
	default:
		return false, "not weapon or armor"
	}
	if !inventoryItemCanShowSlots(item) {
		return false, "special card slots"
	}
	if !item.Identified {
		return false, "unidentified"
	}
	if item.Location&card.Location == 0 {
		return false, "incompatible location"
	}
	slotCount, ok := ctx.Resources.ItemSlotCount(int(item.ItemID))
	if !ok || slotCount <= 0 {
		return false, "no slots in resource db"
	}
	for i := 0; i < slotCount && i < len(item.Cards); i++ {
		if item.Cards[i] == 0 {
			if item.Equipped {
				return false, "equipped"
			}
			return true, ""
		}
	}
	return false, "no empty slot"
}

func logCardCompositionRejectedItems(ctx Context, card session.InventoryItem) {
	if ctx.Session == nil || ctx.Resources == nil {
		return
	}
	for _, item := range ctx.Session.Inventory.Items {
		switch item.Type {
		case db.ItemTypeWeapon, db.ItemTypeArmor:
		default:
			continue
		}
		if ok, reason := cardCompositionCanTarget(ctx, card, item); !ok {
			glog.Warnf("card composition rejected equipment index=%d item=%d type=%d location=0x%04X identified=%v equipped=%v cards=%v reason=%q", item.Index, item.ItemID, item.Type, item.Location, item.Identified, item.Equipped, item.Cards, reason)
		}
	}
}

func inventoryItemIndexes(items []session.InventoryItem) []uint16 {
	indexes := make([]uint16, len(items))
	for i, item := range items {
		indexes[i] = item.Index
	}
	return indexes
}

func inventoryItemEquipLocation(item session.InventoryItem) uint16 {
	if item.Location != 0 {
		return item.Location
	}
	if item.Type == db.ItemTypeAmmo {
		return db.EquipAmmo
	}
	return 0
}
