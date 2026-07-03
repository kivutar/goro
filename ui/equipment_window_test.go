package ui

import (
	"testing"

	"github.com/kivutar/goro/session"
)

func TestEquippedItemForSlotUsesEquippedWearLocation(t *testing.T) {
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 1, ItemID: 1201, Location: equipLocationWeapon, Equip: true},
				{Index: 2, ItemID: 2101, Location: equipLocationShield, Equip: true, Equipped: true},
			},
		},
	}

	item, ok := equippedItemForSlot(s, equipLocationShield)
	if !ok {
		t.Fatal("expected shield slot item")
	}
	if item.Index != 2 || item.ItemID != 2101 {
		t.Fatalf("slot item = %+v, want equipped shield", item)
	}
	if _, ok := equippedItemForSlot(s, equipLocationWeapon); ok {
		t.Fatal("unequipped weapon should not be shown in equipment slot")
	}
}

func TestEquipmentSlotByLocationFindsFirstMatchingSlot(t *testing.T) {
	slot, ok := equipmentSlotByLocation(equipLocationWeapon | equipLocationShield)
	if !ok {
		t.Fatal("expected slot")
	}
	if slot.location != equipLocationWeapon {
		t.Fatalf("slot location = 0x%04X, want weapon first", slot.location)
	}
}

func TestEquipmentSlotShowsAmountOnlyForAmmo(t *testing.T) {
	if !equipmentSlotShowsAmount(equipmentSlotDef{location: equipLocationAmmo}, session.InventoryItem{Amount: 120}) {
		t.Fatal("equipped ammo should show stack amount")
	}
	if equipmentSlotShowsAmount(equipmentSlotDef{location: equipLocationAmmo}, session.InventoryItem{}) {
		t.Fatal("empty ammo amount should not be shown")
	}
	if equipmentSlotShowsAmount(equipmentSlotDef{location: equipLocationWeapon}, session.InventoryItem{Amount: 1}) {
		t.Fatal("weapon amount should not be shown")
	}
}

func TestEquipmentWindowOpensCentered(t *testing.T) {
	window := EquipmentWindow{}
	window.Toggle(Context{ScreenW: 1280, ScreenH: 720})

	if !window.window.IsOpen() {
		t.Fatal("equipment window did not open")
	}
	if window.window.x != (1280-equipmentWindowWidth)/2 || window.window.y != (720-equipmentWindowHeight)/2 {
		t.Fatalf("equipment position = %d,%d, want centered", window.window.x, window.window.y)
	}
}
