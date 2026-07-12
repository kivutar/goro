package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/gogpu/ui/core/checkbox"
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
	equipmentWindowWidth   = 300
	equipmentWindowHeight  = 236
	equipmentFooterH       = 36
	equipmentWindowPad     = 10
	equipmentLeftColW      = 112
	equipmentCenterColW    = 56
	equipmentRightColW     = 112
	equipmentRowH          = 24
	equipmentPreviewImageH = 106
)

type EquipmentWindow struct {
	Window
	snapshot    string
	itemInfo    *ItemInfoWindow
	cart        *CartWindow
	hasCart     bool
	preview     image.Image
	tooltipItem session.InventoryItem
	tooltipOpen bool
	icons       map[equipmentItemIconKey]image.Image
	iconMiss    map[equipmentItemIconKey]struct{}
}

type equipmentItemIconKey struct {
	itemID     uint16
	identified bool
}

type equipmentSlotDef struct {
	label    string
	location uint16
	side     equipmentSlotSide
	row      int
}

type equipmentSlotSide int

const (
	equipmentSlotLeft equipmentSlotSide = iota
	equipmentSlotRight
	equipmentSlotCenter
)

var (
	equipmentSlotHeadTop    = equipmentSlotDef{label: "Head Top", location: db.EquipHeadTop, side: equipmentSlotLeft, row: 0}
	equipmentSlotHeadMid    = equipmentSlotDef{label: "Head Mid", location: db.EquipHeadMid, side: equipmentSlotRight, row: 0}
	equipmentSlotHeadLow    = equipmentSlotDef{label: "Head Low", location: db.EquipHeadBottom, side: equipmentSlotLeft, row: 1}
	equipmentSlotArmor      = equipmentSlotDef{label: "Armor", location: db.EquipArmor, side: equipmentSlotRight, row: 1}
	equipmentSlotWeapon     = equipmentSlotDef{label: "Weapon", location: db.EquipWeapon, side: equipmentSlotLeft, row: 2}
	equipmentSlotShield     = equipmentSlotDef{label: "Shield", location: db.EquipShield, side: equipmentSlotRight, row: 2}
	equipmentSlotGarment    = equipmentSlotDef{label: "Garment", location: db.EquipGarment, side: equipmentSlotLeft, row: 3}
	equipmentSlotShoes      = equipmentSlotDef{label: "Shoes", location: db.EquipShoes, side: equipmentSlotRight, row: 3}
	equipmentSlotAccessory  = equipmentSlotDef{label: "Accessory", location: db.EquipAccessory1, side: equipmentSlotLeft, row: 4}
	equipmentSlotAccessory2 = equipmentSlotDef{label: "Accessory", location: db.EquipAccessory2, side: equipmentSlotRight, row: 4}
	equipmentSlotAmmo       = equipmentSlotDef{label: "Ammo", location: db.EquipAmmo, side: equipmentSlotCenter, row: 1}

	equipmentSlots = []equipmentSlotDef{
		equipmentSlotHeadTop,
		equipmentSlotHeadMid,
		equipmentSlotHeadLow,
		equipmentSlotArmor,
		equipmentSlotWeapon,
		equipmentSlotShield,
		equipmentSlotGarment,
		equipmentSlotShoes,
		equipmentSlotAccessory,
		equipmentSlotAccessory2,
		equipmentSlotAmmo,
	}
)

func (w *EquipmentWindow) Toggle(ctx Context) {
	w.EnsureWindow(equipmentWindowWidth, equipmentWindowHeight)
	if w.IsOpen() {
		w.hideTooltip()
		w.Window.Close()
		w.Publish(ctx)
		return
	}
	w.snapshot = equipmentSnapshot(ctx.Session)
	w.hasCart = inventoryBagHasCart(ctx)
	w.preview = nil
	w.Window.Open(ctx, w.widgetTree(ctx, nil, nil))
	w.Publish(ctx)
}

func (w *EquipmentWindow) Update(ctx Context, itemInfo *ItemInfoWindow, cart *CartWindow, assets AssetProvider) bool {
	w.EnsureWindow(equipmentWindowWidth, equipmentWindowHeight)
	if !w.IsOpen() {
		w.hideTooltip()
		return false
	}
	snapshot := equipmentSnapshot(ctx.Session)
	needsPreview := w.preview == nil && assets != nil
	hasCart := inventoryBagHasCart(ctx)
	if snapshot != w.snapshot || itemInfo != w.itemInfo || cart != w.cart || hasCart != w.hasCart || needsPreview {
		w.hideTooltip()
		w.snapshot = snapshot
		w.itemInfo = itemInfo
		w.cart = cart
		w.hasCart = hasCart
		if assets != nil {
			w.preview = assets.EquipmentPreviewImage(ctx, equipmentCenterColW, equipmentPreviewImageH)
		}
		w.SetContent(w.widgetTree(ctx, itemInfo, cart))
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

func (w *EquipmentWindow) Rebind(ctx Context, itemInfo *ItemInfoWindow, cart *CartWindow, assets AssetProvider) {
	w.EnsureWindow(equipmentWindowWidth, equipmentWindowHeight)
	if !w.IsOpen() {
		return
	}
	w.snapshot = equipmentSnapshot(ctx.Session)
	w.itemInfo = itemInfo
	w.cart = cart
	w.hasCart = inventoryBagHasCart(ctx)
	if assets != nil {
		w.preview = assets.EquipmentPreviewImage(ctx, equipmentCenterColW, equipmentPreviewImageH)
	}
	w.SetContent(w.widgetTree(ctx, itemInfo, cart))
	w.Publish(ctx)
}

func (w *EquipmentWindow) widgetTree(ctx Context, itemInfo *ItemInfoWindow, cart *CartWindow) widget.Widget {
	return Win(
		Title("Equipment"),
		CloseButton(true),
		OnClose(func() {
			w.Window.Close()
			w.Publish(ctx)
		}),
		Size(equipmentWindowWidth, equipmentWindowHeight),
		FooterHeight(equipmentFooterH),
		Content(
			primitives.Box(
				primitives.HBox(
					primitives.Box(
						w.slotWidget(ctx, itemInfo, equipmentSlotHeadTop, equipmentLeftColW),
						w.slotWidget(ctx, itemInfo, equipmentSlotHeadLow, equipmentLeftColW),
						w.slotWidget(ctx, itemInfo, equipmentSlotWeapon, equipmentLeftColW),
						w.slotWidget(ctx, itemInfo, equipmentSlotGarment, equipmentLeftColW),
						w.slotWidget(ctx, itemInfo, equipmentSlotAccessory, equipmentLeftColW),
					).
						Width(equipmentLeftColW).
						Gap(0),

					primitives.Box(
						newStaticImageWidget(w.preview, equipmentCenterColW, equipmentPreviewImageH),
						w.slotWidget(ctx, itemInfo, equipmentSlotAmmo, equipmentCenterColW),
						w.cartButtonWidget(ctx, cart),
					).
						Width(equipmentCenterColW).
						Height(152).
						Gap(0),

					primitives.Box(
						w.slotWidget(ctx, itemInfo, equipmentSlotHeadMid, equipmentRightColW),
						w.slotWidget(ctx, itemInfo, equipmentSlotArmor, equipmentRightColW),
						w.slotWidget(ctx, itemInfo, equipmentSlotShield, equipmentRightColW),
						w.slotWidget(ctx, itemInfo, equipmentSlotShoes, equipmentRightColW),
						w.slotWidget(ctx, itemInfo, equipmentSlotAccessory2, equipmentRightColW),
					).
						Width(equipmentRightColW).
						Gap(0),
				).
					Gap(0),
			).
				Padding(equipmentWindowPad),
		),
		Footer(
			primitives.HBox(
				primitives.Box(
					rotheme.Checkbox(
						checkbox.Checked(ctx.Session != nil && ctx.Session.ShowEquip),
						checkbox.LabelOpt("Show Equip"),
						checkbox.OnToggle(func(enabled bool) {
							if ctx.Session != nil {
								ctx.Session.ShowEquip = enabled
							}
							if ctx.Network != nil {
								_ = ctx.Network.SendShowEquipConfig(enabled)
							}
							w.SetContent(w.widgetTree(ctx, itemInfo, cart))
							w.Publish(ctx)
						}),
					),
				).
					Width(120).
					Height(20),
				primitives.Expanded(primitives.Box()),
			).
				CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *EquipmentWindow) cartButtonWidget(ctx Context, cart *CartWindow) widget.Widget {
	if !inventoryBagHasCart(ctx) {
		return primitives.Box().
			Width(equipmentCenterColW).
			Height(22)
	}
	return rotheme.Button("Cart", func() {
		if cart != nil {
			cart.Toggle(ctx)
		}
	}).
		Width(equipmentCenterColW).
		Height(22)
}

func (w *EquipmentWindow) slotWidget(ctx Context, itemInfo *ItemInfoWindow, slot equipmentSlotDef, width int) widget.Widget {
	if !equipmentSlotVisible(ctx.Session, slot) {
		return primitives.Box().
			Width(float32(width)).
			Height(equipmentRowH)
	}
	item, hasItem := equippedItemForSlot(ctx.Session, slot.location)
	return newEquipmentSlotWidget(equipmentSlotWidgetConfig{
		slot:    slot,
		item:    item,
		icon:    w.itemIconImage(ctx.Resources, item),
		hasItem: hasItem,
		width:   width,
		res:     ctx.Resources,
		onClick: func(item session.InventoryItem) {
			w.hideTooltip()
			w.activateItem(ctx, item)
		},
		onHover: func(item session.InventoryItem) { w.showTooltip(item) },
		onLeave: func() { w.hideTooltip() },
		onRightClick: func(item session.InventoryItem) {
			w.hideTooltip()
			if itemInfo != nil {
				x, y := 0, 0
				if ctx.Input != nil {
					x, y = ctx.Input.MouseX, ctx.Input.MouseY
				}
				itemInfo.openItem(ctx, item, x, y)
			}
		},
	})
}

func (w *EquipmentWindow) DrawTooltip(screen *render.Image, ctx Context) {
	if !w.tooltipOpen || screen == nil || ctx.Input == nil {
		return
	}
	drawInventoryItemTooltip(screen, ctx, w.tooltipItem)
}

func (w *EquipmentWindow) showTooltip(item session.InventoryItem) {
	if item.ItemID == 0 {
		w.hideTooltip()
		return
	}
	w.tooltipItem = item
	w.tooltipOpen = true
}

func (w *EquipmentWindow) hideTooltip() {
	w.tooltipItem = session.InventoryItem{}
	w.tooltipOpen = false
}

func (w *EquipmentWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
	if manager == nil || item.ItemID == 0 {
		return nil
	}
	key := equipmentItemIconKey{itemID: item.ItemID, identified: item.Identified}
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
		w.icons = make(map[equipmentItemIconKey]image.Image)
	}
	w.icons[key] = img
	return img
}

func (w *EquipmentWindow) markIconMiss(key equipmentItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[equipmentItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

type staticImageWidget struct {
	widget.WidgetBase
	image  image.Image
	width  int
	height int
}

func newStaticImageWidget(img image.Image, width, height int) *staticImageWidget {
	w := &staticImageWidget{
		image:  img,
		width:  width,
		height: height,
	}
	w.SetVisible(true)
	w.SetEnabled(false)
	return w
}

func (w *staticImageWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(float32(w.width), float32(w.height)))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *staticImageWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() || w.image == nil {
		return
	}
	canvas.DrawImage(w.image, w.Bounds().Min)
}

func (w *staticImageWidget) Event(ctx widget.Context, e event.Event) bool {
	return false
}

func (w *EquipmentWindow) activateItem(ctx Context, item session.InventoryItem) {
	if ctx.Network == nil {
		return
	}
	if err := ctx.Network.SendTakeoffEquip(item.Index); err != nil {
		return
	}
}

func equipmentSnapshot(s *session.Session) string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "job=%d;", selectedCharacter(s).Job)
	fmt.Fprintf(&b, "showEquip=%t;", s.ShowEquip)
	for _, item := range s.Inventory.Items {
		if !item.Equip || !item.Equipped {
			continue
		}
		fmt.Fprintf(&b, "%d:%d:%d:%d:%d;", item.Index, item.ItemID, item.Location, item.Amount, item.Refine)
	}
	return b.String()
}

type equipmentSlotWidgetConfig struct {
	slot         equipmentSlotDef
	item         session.InventoryItem
	icon         image.Image
	hasItem      bool
	width        int
	res          *res.Manager
	onClick      func(session.InventoryItem)
	onHover      func(session.InventoryItem)
	onLeave      func()
	onRightClick func(session.InventoryItem)
}

type equipmentSlotWidget struct {
	widget.WidgetBase
	cfg     equipmentSlotWidgetConfig
	hovered bool
}

func newEquipmentSlotWidget(cfg equipmentSlotWidgetConfig) *equipmentSlotWidget {
	w := &equipmentSlotWidget{cfg: cfg}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *equipmentSlotWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(float32(w.cfg.width), equipmentRowH))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *equipmentSlotWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	line := rotheme.Default.Colors.WindowBorder
	if w.hovered && w.cfg.hasItem {
		line = rotheme.Default.Colors.ButtonBorder
	}
	if w.cfg.slot.side != equipmentSlotCenter {
		canvas.DrawLine(
			geometry.Pt(bounds.Min.X, bounds.Max.Y-1),
			geometry.Pt(bounds.Max.X, bounds.Max.Y-1),
			line,
			1,
		)
	}

	color := rotheme.Default.Colors.MutedText
	text := w.cfg.slot.label
	if w.cfg.hasItem {
		color = rotheme.Default.Colors.Text
		text = inventoryItemDisplayName(w.cfg.res, w.cfg.item)
		if w.cfg.slot.side == equipmentSlotCenter && w.cfg.icon != nil {
			text = ""
		}
		if w.cfg.item.Refine > 0 {
			text = "+" + formatHUDNumber(int64(w.cfg.item.Refine)) + " " + text
		}
		if equipmentSlotShowsAmount(w.cfg.slot, w.cfg.item) {
			if text == "" {
				text = fmt.Sprintf("%d", w.cfg.item.Amount)
			} else {
				text = fmt.Sprintf("%s %d", text, w.cfg.item.Amount)
			}
		}
	}
	text = trimRunes(text, equipmentSlotTextLimit(w.cfg.slot))
	textBounds := equipmentSlotTextBounds(bounds, w.cfg.slot, w.cfg.icon)
	if w.cfg.icon != nil {
		canvas.DrawImage(w.cfg.icon, equipmentSlotIconPosition(bounds, w.cfg.slot, w.cfg.icon))
	}
	align := equipmentSlotTextAlign(w.cfg.slot)
	if w.cfg.slot.side == equipmentSlotCenter && w.cfg.icon != nil {
		align = widget.TextAlignRight
	}
	rotheme.DrawText(
		canvas,
		text,
		textBounds,
		rotheme.Default.Typography.TextSize,
		color,
		false,
		align,
	)
}

func (w *equipmentSlotWidget) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove:
		w.hovered = true
		if w.cfg.hasItem {
			if w.cfg.onHover != nil {
				w.cfg.onHover(w.cfg.item)
			}
			ctx.SetCursor(widget.CursorPointer)
		} else if w.cfg.onLeave != nil {
			w.cfg.onLeave()
		}
		return w.cfg.hasItem
	case event.MouseLeave:
		w.hovered = false
		if w.cfg.onLeave != nil {
			w.cfg.onLeave()
		}
		ctx.SetCursor(widget.CursorDefault)
		return false
	case event.MousePress:
		if !w.cfg.hasItem {
			return true
		}
		switch mouse.Button {
		case event.ButtonLeft:
			if w.cfg.onClick != nil {
				w.cfg.onClick(w.cfg.item)
			}
			return true
		case event.ButtonRight:
			if w.cfg.onRightClick != nil {
				w.cfg.onRightClick(w.cfg.item)
			}
			return true
		}
	}
	return w.cfg.hasItem
}

func equipmentSlotIconPosition(bounds geometry.Rect, slot equipmentSlotDef, icon image.Image) geometry.Point {
	iconBounds := icon.Bounds()
	width := float32(iconBounds.Dx())
	height := float32(iconBounds.Dy())
	y := bounds.Min.Y + (bounds.Height()-height)/2
	switch slot.side {
	case equipmentSlotRight:
		return geometry.Pt(bounds.Max.X-width-4, y)
	case equipmentSlotCenter:
		return geometry.Pt(bounds.Min.X+(bounds.Width()-width)/2, y)
	default:
		return geometry.Pt(bounds.Min.X+4, y)
	}
}

func equipmentSlotTextBounds(bounds geometry.Rect, slot equipmentSlotDef, icon image.Image) geometry.Rect {
	if icon == nil {
		return geometry.NewRect(bounds.Min.X+4, bounds.Min.Y+4, bounds.Width()-8, bounds.Height()-8)
	}
	iconWidth := float32(icon.Bounds().Dx())
	switch slot.side {
	case equipmentSlotRight:
		return geometry.NewRect(bounds.Min.X+4, bounds.Min.Y+4, bounds.Width()-iconWidth-12, bounds.Height()-8)
	case equipmentSlotCenter:
		return geometry.NewRect(bounds.Min.X+4, bounds.Min.Y+8, bounds.Width()-8, bounds.Height()-8)
	default:
		return geometry.NewRect(bounds.Min.X+iconWidth+8, bounds.Min.Y+4, bounds.Width()-iconWidth-12, bounds.Height()-8)
	}
}

func equipmentSlotTextAlign(slot equipmentSlotDef) widget.TextAlign {
	if slot.side == equipmentSlotRight {
		return widget.TextAlignRight
	}
	if slot.side == equipmentSlotCenter {
		return widget.TextAlignCenter
	}
	return widget.TextAlignLeft
}

func equipmentSlotTextLimit(slot equipmentSlotDef) int {
	if slot.side == equipmentSlotCenter {
		return 8
	}
	return 14
}

func equipmentSlotShowsAmount(slot equipmentSlotDef, item session.InventoryItem) bool {
	return slot.location == db.EquipAmmo && item.Amount > 0
}

func equippedItemForSlot(s *session.Session, location uint16) (session.InventoryItem, bool) {
	if s == nil || location == 0 {
		return session.InventoryItem{}, false
	}
	for _, item := range s.Inventory.Items {
		if !item.Equip || !item.Equipped || item.Location&location == 0 {
			continue
		}
		return item, true
	}
	return session.InventoryItem{}, false
}

func equipmentSlotVisible(s *session.Session, slot equipmentSlotDef) bool {
	if slot.location != db.EquipAmmo {
		return true
	}
	if s == nil {
		return false
	}
	return jobSupportsAmmo(int(selectedCharacter(s).Job))
}

func jobSupportsAmmo(job int) bool {
	switch job {
	case 3, 11, 19, 20, 24,
		4004, 4012, 4020, 4021,
		4026, 4034, 4042, 4043,
		4056, 4068, 4069:
		return true
	default:
		return false
	}
}

func equipmentSlotByLocation(location uint16) (equipmentSlotDef, bool) {
	for _, slot := range equipmentSlots {
		if location&slot.location != 0 {
			return slot, true
		}
	}
	return equipmentSlotDef{}, false
}
