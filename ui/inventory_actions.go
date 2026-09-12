package ui

import (
	"fmt"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/session"
)

// UseInventoryItem requests use of a consumable inventory entry.
func UseInventoryItem(ctx Context, item session.InventoryItem) error {
	if ctx.PlayerHasEffectState(db.EffectStateHide) {
		return fmt.Errorf("cannot use items while hiding")
	}
	if ctx.Session != nil && ctx.Session.Dead {
		return fmt.Errorf("player is dead")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if !db.ItemTypeIsUsable(item.Type) {
		return fmt.Errorf("item cannot be used")
	}
	target := uint32(0)
	if ctx.Session != nil {
		target = ctx.Session.AccountID
		if target == 0 {
			target = ctx.Session.CharID
		}
	}
	if target == 0 {
		return fmt.Errorf("missing player id")
	}
	return ctx.Network.SendUseInventoryItem(item.Index, target)
}

func dropInventoryItem(ctx Context, item session.InventoryItem, amount uint16) error {
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	return ctx.Network.SendDropInventoryItem(item.Index, inventoryDropAmount(item, amount))
}

func equipInventoryItem(ctx Context, item session.InventoryItem) {
	if ctx.Network == nil {
		glog.Warnf("inventory equip failed: not connected")
		return
	}
	if item.Equipped {
		if err := ctx.Network.SendTakeoffEquip(item.Index); err != nil {
			glog.Warnf("inventory unequip failed: %v", err)
		}
		return
	}
	if item.Type == db.ItemTypePetArmor {
		if err := ctx.Network.SendWearEquip(item.Index, 0); err != nil {
			glog.Warnf("inventory pet armor equip failed: %v", err)
		}
		return
	}
	location := inventoryItemEquipLocationForSession(ctx.Session, item)
	if location == 0 {
		glog.Warnf("inventory equip failed: missing equip location item=%d index=%d", item.ItemID, item.Index)
		return
	}
	if err := ctx.Network.SendWearEquip(item.Index, location); err != nil {
		glog.Warnf("inventory equip failed: %v", err)
	}
}

func inventoryItemEquipLocationForSession(s *session.Session, item session.InventoryItem) uint16 {
	location := inventoryItemEquipLocation(item)
	if location&(db.EquipAccessory1|db.EquipAccessory2) != db.EquipAccessory1|db.EquipAccessory2 {
		return location
	}
	if _, ok := equippedItemForSlot(s, db.EquipAccessory1); !ok {
		return db.EquipAccessory1
	}
	if _, ok := equippedItemForSlot(s, db.EquipAccessory2); !ok {
		return db.EquipAccessory2
	}
	return db.EquipAccessory1
}

func inventoryItemTypeStackable(itemType uint8) bool {
	switch itemType {
	case db.ItemTypeWeapon, db.ItemTypeArmor, db.ItemTypePetEgg, db.ItemTypePetArmor, db.ItemTypeShadowGear:
		return false
	default:
		return true
	}
}

func inventoryDropMaxAmount(item session.InventoryItem) uint16 {
	if !inventoryItemTypeStackable(item.Type) || item.Amount <= 1 {
		return 1
	}
	if item.Amount > int(^uint16(0)) {
		return ^uint16(0)
	}
	return uint16(item.Amount)
}

func inventoryDropAmount(item session.InventoryItem, requested uint16) uint16 {
	return clampAmount(requested, inventoryDropMaxAmount(item))
}

func currentInventoryDropItem(s *session.Session, dragged session.InventoryItem) (session.InventoryItem, bool) {
	if s == nil {
		return dragged, true
	}
	for _, item := range s.Inventory.Items {
		if item.Index == dragged.Index && item.ItemID == dragged.ItemID {
			return item, true
		}
	}
	return session.InventoryItem{}, false
}
