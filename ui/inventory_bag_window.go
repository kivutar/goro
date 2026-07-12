package ui

import (
	"fmt"
	"image"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	inventoryBagTabW    = 64
	inventoryBagTabH    = 32
	inventoryBagCell    = 32
	inventoryBagIcon    = 24
	inventoryBagCols    = 8
	inventoryBagRows    = 5
	inventoryBagTabOver = 1
	inventoryBagWidth   = inventoryBagTabW + inventoryBagCols*inventoryBagCell + 2
	inventoryBagHeight  = ROWindowTitleHeight + inventoryBagRows*inventoryBagCell + 2
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
	scroll        int
	snapshot      string
	itemInfo      *ItemInfoWindow
	cart          *CartWindow
	lastClickItem uint16
	lastClickAt   time.Time
	dragItem      session.InventoryItem
	dragActive    bool
	dragFrom      time.Time
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
		w.Window.Close()
		w.Publish(ctx)
		return
	}
	w.selectFirstNonEmptyTab(ctx.Session)
	w.ClampScroll(ctx.Session)
	w.snapshot = w.inventorySnapshot(ctx.Session)
	x, y := inventoryBagDefaultPosition(ctx)
	w.OpenAt(x, y, w.widgetTree(ctx, nil, nil))
	w.Publish(ctx)
}

func (w *InventoryBagWindow) Update(ctx Context, shortcuts *ShortcutBar, storage *StorageWindow, cart *CartWindow, trade *TradeWindow, itemInfo *ItemInfoWindow) bool {
	w.EnsureWindow(inventoryBagWidth, inventoryBagHeight)
	if !w.IsOpen() || ctx.Input == nil {
		w.hideTooltip()
		return false
	}
	cartChanged := cart != w.cart
	w.cart = cart
	if w.UpdateDrag(ctx, shortcuts, storage, cart, trade) {
		return true
	}
	w.ClampScroll(ctx.Session)
	snapshot := w.inventorySnapshot(ctx.Session)
	if snapshot != w.snapshot || itemInfo != w.itemInfo || cartChanged {
		w.snapshot = snapshot
		w.itemInfo = itemInfo
		w.SetContent(w.widgetTree(ctx, itemInfo, w.cart))
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

func (w *InventoryBagWindow) UpdateDrag(ctx Context, shortcuts *ShortcutBar, storage *StorageWindow, cart *CartWindow, trade *TradeWindow) bool {
	if !w.dragActive || ctx.Input == nil {
		return false
	}
	if ctx.Input.MouseJustReleased(render.MouseButtonLeft) || !ctx.Input.MousePressed(render.MouseButtonLeft) {
		item := w.dragItem
		w.dragActive = false
		w.dragItem = session.InventoryItem{}
		if storage != nil && storage.AcceptInventoryDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if cart != nil && cart.AcceptInventoryDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if trade != nil && trade.AcceptInventoryDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if shortcuts != nil && shortcuts.AcceptItemDrop(ctx, item, ctx.Input.MouseX, ctx.Input.MouseY) {
			return true
		}
		if !w.pointInside(ctx.Input.MouseX, ctx.Input.MouseY) {
			if err := dropInventoryItem(ctx, item); err != nil {
				log.Printf("inventory drop failed: %v", err)
				return true
			}
			log.Printf("inventory drop requested index=%d item=%d amount=%d", item.Index, item.ItemID, inventoryDropAmount(item))
			return true
		}
		return true
	}
	return true
}

func (w *InventoryBagWindow) Draw(screen *render.Image, ctx Context, assets AssetProvider) {
	w.Publish(ctx)
}

func (w *InventoryBagWindow) DrawTooltip(screen *render.Image) {
	if w.dragActive {
		return
	}
	w.tooltip.Draw(screen)
}

func (w *InventoryBagWindow) DrawDragGhost(screen *render.Image, ctx Context, assets AssetProvider) {
	if !w.dragActive || screen == nil || ctx.Input == nil || assets == nil {
		return
	}
	if time.Since(w.dragFrom) <= 80*time.Millisecond {
		return
	}
	assets.DrawInventoryItemIcon(screen, ctx.Resources, w.dragItem, ctx.Input.MouseX-inventoryIconSize/2, ctx.Input.MouseY-inventoryIconSize/2)
}

func (w *InventoryBagWindow) Rebind(ctx Context, itemInfo *ItemInfoWindow, cart *CartWindow) {
	w.EnsureWindow(inventoryBagWidth, inventoryBagHeight)
	if !w.IsOpen() {
		return
	}
	w.cart = cart
	w.refresh(ctx, itemInfo)
}

func (w *InventoryBagWindow) widgetTree(ctx Context, itemInfo *ItemInfoWindow, cart *CartWindow) widget.Widget {
	return Win(
		Title("Inventory"),
		CloseButton(true),
		OnClose(func() {
			w.Window.Close()
			w.Publish(ctx)
		}),
		Size(inventoryBagWidth, inventoryBagHeight),
		Content(
			primitives.HBox(
				w.tabColumn(ctx, cart),
				newInventoryGridWidget(inventoryGridConfig{
					items:   w.visibleItems(ctx.Session),
					icons:   w.visibleItemIcons(ctx),
					total:   len(w.tabItems(ctx.Session)),
					scroll:  w.scroll,
					onWheel: func(delta float32) { w.scrollBy(delta, ctx.Session); w.refresh(ctx, itemInfo) },
					onPress: func(item session.InventoryItem) { w.startItemDragOrActivate(ctx, item) },
					onHover: func(item session.InventoryItem) { w.showTooltip(ctx, item) },
					onLeave: func() { w.hideTooltip() },
					onRightClick: func(item session.InventoryItem, mx, my int) {
						w.hideTooltip()
						w.dragActive = false
						w.dragItem = session.InventoryItem{}
						if itemInfo != nil {
							itemInfo.openItem(ctx, item, mx, my)
						}
					},
				}),
			).
				Gap(0),
		),
	)
}

func (w *InventoryBagWindow) tabColumn(ctx Context, cart *CartWindow) widget.Widget {
	tabs := make([]widget.Widget, 0, len(inventoryBagTabs))
	for _, tab := range inventoryBagTabs {
		tab := tab
		tabs = append(tabs, newTabWidget(tabWidgetConfig{
			label:      tab.label,
			active:     tab.tab == w.tab,
			width:      inventoryBagTabW + inventoryBagTabOver*2,
			height:     inventoryBagTabH,
			blendEdge:  tabBlendRight,
			blendInset: inventoryBagTabOver,
			onClick: func() {
				w.hideTooltip()
				w.tab = tab.tab
				w.scroll = 0
				w.lastClickItem = 0
				w.refresh(ctx, w.itemInfo)
			},
		}))
	}
	if inventoryBagHasCart(ctx) {
		tabs = append(tabs,
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Cart", func() {
				if cart != nil {
					cart.Toggle(ctx)
				}
			}).
				Width(inventoryBagTabW).
				Height(24),
		)
	}
	return primitives.Box(tabs...).
		Width(inventoryBagTabW + inventoryBagTabOver*2).
		Height(inventoryBagRows * inventoryBagCell).
		Gap(-inventoryBagTabOver)
}

func (w *InventoryBagWindow) refresh(ctx Context, itemInfo *ItemInfoWindow) {
	w.hideTooltip()
	w.ClampScroll(ctx.Session)
	w.snapshot = w.inventorySnapshot(ctx.Session)
	w.itemInfo = itemInfo
	w.SetContent(w.widgetTree(ctx, itemInfo, w.cart))
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
	x := clampWindowInt(menuX, 8, maxInt(8, width-inventoryBagWidth-8))
	y := clampWindowInt(menuY+menuH+8, 8, maxInt(8, height-inventoryBagHeight-8))
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

func (w *InventoryBagWindow) visibleItemIcons(ctx Context) []image.Image {
	items := w.visibleItems(ctx.Session)
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
	if inventoryItemIsEquipment(item) {
		if ctx.Network == nil {
			log.Printf("inventory equip failed: not connected")
			return
		}
		if item.Equipped {
			if err := ctx.Network.SendTakeoffEquip(item.Index); err != nil {
				log.Printf("inventory unequip failed: %v", err)
			}
			return
		}
		location := inventoryItemEquipLocation(item)
		if location == 0 {
			log.Printf("inventory equip failed: missing equip location item=%d index=%d", item.ItemID, item.Index)
			return
		}
		if err := ctx.Network.SendWearEquip(item.Index, location); err != nil {
			log.Printf("inventory equip failed: %v", err)
		}
		return
	}
	if !inventoryItemIsUsable(item) {
		log.Printf("inventory use skipped: item cannot be used index=%d item=%d type=%d", item.Index, item.ItemID, item.Type)
		return
	}
	if err := useInventoryItem(ctx, item); err != nil {
		log.Printf("inventory use failed: %v", err)
		return
	}
	log.Printf("inventory use requested index=%d item=%d type=%d", item.Index, item.ItemID, item.Type)
}

func (w *InventoryBagWindow) scrollBy(wheelY float32, s *session.Session) {
	if wheelY > 0 {
		w.scroll--
	} else if wheelY < 0 {
		w.scroll++
	}
	w.ClampScroll(s)
}

func (w *InventoryBagWindow) ClampScroll(s *session.Session) {
	maxScroll := maxInt(0, (len(w.tabItems(s))+inventoryBagCols-1)/inventoryBagCols-inventoryBagRows)
	w.scroll = clampWindowInt(w.scroll, 0, maxScroll)
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
			w.scroll = 0
			return
		}
	}
	w.tab = original
}

func (w *InventoryBagWindow) visibleItems(s *session.Session) []session.InventoryItem {
	items := w.tabItems(s)
	start := w.scroll * inventoryBagCols
	if start < 0 {
		start = 0
	}
	if start >= len(items) {
		return nil
	}
	end := minInt(len(items), start+inventoryBagCols*inventoryBagRows)
	return items[start:end]
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
	fmt.Fprintf(&b, "tab=%d;scroll=%d;", w.tab, w.scroll)
	for _, item := range sortedInventoryItems(s) {
		fmt.Fprintf(&b, "%d:%d:%d:%d:%d:%t:%t;", item.Index, item.ItemID, item.Type, item.Amount, item.Location, item.Equip, item.Equipped)
	}
	return b.String()
}

type tabBlendEdge int

const (
	tabBlendNone tabBlendEdge = iota
	tabBlendRight
	tabBlendBottom
)

type tabWidgetConfig struct {
	label      string
	active     bool
	width      int
	height     int
	blendEdge  tabBlendEdge
	blendInset int
	onClick    func()
}

type tabWidget struct {
	widget.WidgetBase
	cfg     tabWidgetConfig
	hovered bool
}

func newTabWidget(cfg tabWidgetConfig) *tabWidget {
	w := &tabWidget{cfg: cfg}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *tabWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(float32(w.cfg.width), float32(w.cfg.height)))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *tabWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	fill := rotheme.Default.Colors.Button
	if w.cfg.active {
		fill = rotheme.Default.Colors.WindowBody
	} else if w.hovered {
		fill = rotheme.Default.Colors.ButtonHover
	}
	canvas.DrawRect(bounds, fill)
	canvas.StrokeRect(bounds, rotheme.Default.Colors.WindowBorder, 1)
	if w.cfg.active {
		inset := float32(w.cfg.blendInset)
		switch w.cfg.blendEdge {
		case tabBlendRight:
			canvas.DrawRect(geometry.NewRect(bounds.Max.X-1, bounds.Min.Y+inset, 1, bounds.Height()-inset*2), rotheme.Default.Colors.WindowBody)
		case tabBlendBottom:
			canvas.DrawRect(geometry.NewRect(bounds.Min.X+inset, bounds.Max.Y-1, bounds.Width()-inset*2, 1), rotheme.Default.Colors.WindowBody)
		}
	}
	rotheme.DrawText(canvas, w.cfg.label, bounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignCenter)
}

func (w *tabWidget) Event(ctx widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter:
		w.hovered = true
		ctx.SetCursor(widget.CursorPointer)
		return true
	case event.MouseLeave:
		w.hovered = false
		ctx.SetCursor(widget.CursorDefault)
		return false
	case event.MousePress:
		if mouse.Button == event.ButtonLeft && w.cfg.onClick != nil {
			w.cfg.onClick()
			return true
		}
	}
	return true
}

type inventoryGridConfig struct {
	items        []session.InventoryItem
	icons        []image.Image
	total        int
	scroll       int
	onWheel      func(float32)
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
	size := constraints.Constrain(geometry.Sz(inventoryBagCols*inventoryBagCell, inventoryBagRows*inventoryBagCell))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *inventoryGridWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	canvas.DrawRect(bounds, rotheme.Default.Colors.WindowBody)
	canvas.StrokeRect(bounds, rotheme.Default.Colors.WindowBorder, 1)
	for row := 0; row < inventoryBagRows; row++ {
		for col := 0; col < inventoryBagCols; col++ {
			index := row*inventoryBagCols + col
			cell := w.cellBounds(index)
			fill := widget.RGBA8(255, 255, 250, 64)
			if index == w.hovered {
				fill = widget.RGBA8(118, 150, 204, 92)
			}
			canvas.DrawRect(cell, fill)
			canvas.StrokeRect(cell, widget.RGBA8(216, 224, 232, 160), 1)
		}
	}
	for i, item := range w.cfg.items {
		if i >= inventoryBagCols*inventoryBagRows {
			break
		}
		cell := w.cellBounds(i)
		if i < len(w.cfg.icons) && w.cfg.icons[i] != nil {
			canvas.DrawImage(w.cfg.icons[i], geometry.Pt(cell.Min.X+4, cell.Min.Y+4))
		}
		if item.Amount > 1 {
			rotheme.DrawText(canvas, fmt.Sprintf("%d", item.Amount), geometry.NewRect(cell.Max.X-18, cell.Max.Y-15, 16, 12), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignRight)
		}
		if item.Equipped {
			rotheme.DrawText(canvas, "E", geometry.NewRect(cell.Min.X+2, cell.Min.Y+2, 12, 12), rotheme.Default.Typography.TextSize, widget.RGBA8(54, 128, 76, 255), false, widget.TextAlignLeft)
		}
	}
	w.drawScrollBar(canvas)
}

func (w *inventoryGridWidget) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.WheelEvent:
		if w.cfg.onWheel != nil {
			w.cfg.onWheel(ev.DeltaY())
		}
		return true
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
	}
	return true
}

func (w *inventoryGridWidget) cellBounds(index int) geometry.Rect {
	bounds := w.Bounds()
	col := index % inventoryBagCols
	row := index / inventoryBagCols
	return geometry.NewRect(
		bounds.Min.X+float32(col*inventoryBagCell),
		bounds.Min.Y+float32(row*inventoryBagCell),
		inventoryBagCell-1,
		inventoryBagCell-1,
	)
}

func (w *inventoryGridWidget) indexAt(point geometry.Point) int {
	local := point.Sub(w.Bounds().Min)
	if local.X < 0 || local.Y < 0 || local.X >= inventoryBagCols*inventoryBagCell || local.Y >= inventoryBagRows*inventoryBagCell {
		return -1
	}
	col := int(local.X) / inventoryBagCell
	row := int(local.Y) / inventoryBagCell
	index := row*inventoryBagCols + col
	if index < 0 || index >= inventoryBagCols*inventoryBagRows {
		return -1
	}
	return index
}

func (w *inventoryGridWidget) drawScrollBar(canvas widget.Canvas) {
	total := w.cfg.total
	if total <= inventoryBagCols*inventoryBagRows {
		return
	}
	bounds := w.Bounds()
	trackX := bounds.Max.X - 5
	canvas.DrawRect(geometry.NewRect(trackX, bounds.Min.Y+1, 4, bounds.Height()-2), rotheme.Default.Colors.Button)
	totalRows := (total + inventoryBagCols - 1) / inventoryBagCols
	maxScroll := maxInt(1, totalRows-inventoryBagRows)
	thumbH := maxInt(18, int(bounds.Height())*inventoryBagRows/totalRows)
	thumbTravel := int(bounds.Height()) - 2 - thumbH
	thumbY := bounds.Min.Y + 1 + float32(thumbTravel*w.cfg.scroll/maxScroll)
	canvas.DrawRect(geometry.NewRect(trackX, thumbY, 4, float32(thumbH)), rotheme.Default.Colors.MutedText)
}

func inventoryItemTab(item session.InventoryItem) int {
	if inventoryItemIsEquipment(item) {
		return inventoryBagTabEquip
	}
	if inventoryItemIsUsable(item) {
		return inventoryBagTabItem
	}
	return inventoryBagTabEtc
}

func inventoryItemIsEquipment(item session.InventoryItem) bool {
	return item.Equip || inventoryItemTypeIsEquipment(item.Type)
}

func inventoryItemTypeIsEquipment(itemType uint8) bool {
	switch itemType {
	case db.ItemTypeArmor, db.ItemTypeWeapon, db.ItemTypePetEgg, db.ItemTypePetArmor, db.ItemTypeAmmo, db.ItemTypeShadowGear:
		return true
	default:
		return false
	}
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

func inventoryItemIsUsable(item session.InventoryItem) bool {
	switch item.Type {
	case db.ItemTypeHealing, db.ItemTypeUsable, db.ItemTypeDelayConsume, db.ItemTypeCash:
		return true
	default:
		return false
	}
}
