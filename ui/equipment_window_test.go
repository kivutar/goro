package ui

import (
	"testing"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
)

func TestEquippedItemForSlotUsesEquippedWearLocation(t *testing.T) {
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 1, ItemID: 1201, Location: db.EquipWeapon, Equip: true},
				{Index: 2, ItemID: 2101, Location: db.EquipShield, Equip: true, Equipped: true},
			},
		},
	}

	item, ok := equippedItemForSlot(s, db.EquipShield)
	if !ok {
		t.Fatal("expected shield slot item")
	}
	if item.Index != 2 || item.ItemID != 2101 {
		t.Fatalf("slot item = %+v, want equipped shield", item)
	}
	if _, ok := equippedItemForSlot(s, db.EquipWeapon); ok {
		t.Fatal("unequipped weapon should not be shown in equipment slot")
	}
}

func TestEquipmentSlotByLocationFindsFirstMatchingSlot(t *testing.T) {
	slot, ok := equipmentSlotByLocation(db.EquipWeapon | db.EquipShield)
	if !ok {
		t.Fatal("expected slot")
	}
	if slot.location != db.EquipWeapon {
		t.Fatalf("slot location = 0x%04X, want weapon first", slot.location)
	}
}

func TestEquipmentSlotShowsAmountOnlyForAmmo(t *testing.T) {
	if !equipmentSlotShowsAmount(equipmentSlotDef{location: db.EquipAmmo}, session.InventoryItem{Amount: 120}) {
		t.Fatal("equipped ammo should show stack amount")
	}
	if equipmentSlotShowsAmount(equipmentSlotDef{location: db.EquipAmmo}, session.InventoryItem{}) {
		t.Fatal("empty ammo amount should not be shown")
	}
	if equipmentSlotShowsAmount(equipmentSlotDef{location: db.EquipWeapon}, session.InventoryItem{Amount: 1}) {
		t.Fatal("weapon amount should not be shown")
	}
}

func TestEquipmentWindowOpensCentered(t *testing.T) {
	window := EquipmentWindow{}
	window.Toggle(Context{ScreenW: 1280, ScreenH: 720})

	if !window.Window.IsOpen() {
		t.Fatal("equipment window did not open")
	}
	if window.Window.x != (1280-equipmentWindowWidth)/2 || window.Window.y != (720-equipmentWindowHeight)/2 {
		t.Fatalf("equipment position = %d,%d, want centered", window.Window.x, window.Window.y)
	}
}

func TestEquipmentWindowTooltipTracksHoveredItem(t *testing.T) {
	window := EquipmentWindow{}
	item := session.InventoryItem{Index: 3, ItemID: 1201, Identified: true}

	window.showTooltip(item)
	if !window.tooltipOpen {
		t.Fatal("tooltip should be open")
	}
	if window.tooltipItem.ItemID != item.ItemID || window.tooltipItem.Index != item.Index {
		t.Fatalf("tooltip item = %+v, want %+v", window.tooltipItem, item)
	}

	window.hideTooltip()
	if window.tooltipOpen || window.tooltipItem.ItemID != 0 {
		t.Fatalf("tooltip not cleared: open=%t item=%+v", window.tooltipOpen, window.tooltipItem)
	}
}
