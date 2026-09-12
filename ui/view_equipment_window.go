package ui

import (
	"image"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

type ViewEquipmentWindow struct {
	Window
	title    string
	items    []session.InventoryItem
	preview  image.Image
	itemInfo *ItemWindows
	icons    map[equipmentItemIconKey]image.Image
	iconMiss map[equipmentItemIconKey]struct{}
}

func (w *ViewEquipmentWindow) Open(ctx Context, view network.ViewedEquipment, assets AssetProvider) {
	w.EnsureWindow(equipmentWindowWidth, equipmentWindowHeight-ROWindowFooterHeight)
	w.title = view.Name
	if w.title == "" {
		w.title = "Equipment"
	}
	w.items = w.items[:0]
	for _, item := range view.Items {
		w.items = append(w.items, session.InventoryItem{
			Index:      item.Index,
			ItemID:     item.ItemID,
			Type:       item.Type,
			Location:   item.Location,
			Identified: item.Identified,
			Amount:     int(item.Amount),
			Equip:      true,
			Equipped:   item.Equipped,
			Damaged:    item.Damaged,
			Refine:     item.Refine,
		})
	}
	w.preview = nil
	if assets != nil {
		w.preview = assets.EquipmentPreviewImageForCharacter(ctx, viewedEquipmentCharacter(view, ctx.Resources), view.Sex, equipmentCenterColW, equipmentPreviewImageH)
	}
	w.Window.Open(ctx, w.widgetTree(ctx, nil))
	w.Publish(ctx)
}

func (w *ViewEquipmentWindow) Update(ctx Context, itemInfo *ItemWindows) bool {
	w.EnsureWindow(equipmentWindowWidth, equipmentWindowHeight-ROWindowFooterHeight)
	if !w.IsOpen() {
		return false
	}
	if itemInfo != w.itemInfo {
		w.itemInfo = itemInfo
		w.SetContent(w.widgetTree(ctx, itemInfo))
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *ViewEquipmentWindow) widgetTree(ctx Context, itemInfo *ItemWindows) widget.Widget {
	return Win(
		Title(w.title),
		CloseButton(true),
		OnClose(func() {
			w.Close()
			w.Publish(ctx)
		}),
		Size(equipmentWindowWidth, equipmentWindowHeight-ROWindowFooterHeight),
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
					).
						Width(equipmentCenterColW).
						Height(equipmentCenterColH).
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
	)
}

func (w *ViewEquipmentWindow) slotWidget(ctx Context, itemInfo *ItemWindows, slot equipmentSlotDef, width int) widget.Widget {
	item, hasItem := w.itemForSlot(slot.location)
	return newEquipmentSlotWidget(equipmentSlotWidgetConfig{
		slot:    slot,
		item:    item,
		icon:    w.itemIconImage(ctx.Resources, item),
		hasItem: hasItem,
		width:   width,
		res:     ctx.Resources,
		onRightClick: func(item session.InventoryItem) {
			if itemInfo == nil {
				return
			}
			x, y := 0, 0
			if ctx.Input != nil {
				x, y = ctx.Input.MouseX, ctx.Input.MouseY
			}
			itemInfo.openItem(ctx, item, x, y)
		},
	})
}

func (w *ViewEquipmentWindow) itemForSlot(location uint16) (session.InventoryItem, bool) {
	for _, item := range w.items {
		if item.Location&location != 0 {
			return item, true
		}
	}
	return session.InventoryItem{}, false
}

func (w *ViewEquipmentWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
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

func (w *ViewEquipmentWindow) markIconMiss(key equipmentItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[equipmentItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func viewedEquipmentCharacter(view network.ViewedEquipment, manager *res.Manager) session.Character {
	character := session.Character{
		Name:      view.Name,
		Job:       int16(view.Job),
		Hair:      int16(view.Head),
		HairColor: uint8(view.HeadPalette),
		HeadPal:   int16(view.HeadPalette),
		BodyPal:   int16(view.BodyPalette),
		HeadTop:   int16(view.HeadTop),
		HeadMid:   int16(view.HeadMid),
		HeadLow:   int16(view.HeadLow),
	}
	for _, item := range view.Items {
		if !item.Equipped {
			continue
		}
		switch {
		case item.Location&db.EquipWeapon != 0:
			character.Weapon = int16(res.PlayerWeaponViewID(manager, int(item.ItemID)))
		case item.Location&db.EquipShield != 0:
			character.Shield = int16(item.ItemID)
		}
	}
	return character
}
