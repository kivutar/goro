package ui

import (
	"testing"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func TestEquippedItemForSlotUsesEquippedWearLocation(t *testing.T) {
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 1, ItemID: 1201, Location: db.EquipWeapon, Equip: true},
				{Index: 2, ItemID: 2101, Location: db.EquipShield, WearLocation: db.EquipShield, Equip: true, Equipped: true},
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

func TestAccessoryWindowAndEquipSelectionUseOccupiedSlots(t *testing.T) {
	for _, worn := range []uint16{db.EquipAccessory1, db.EquipAccessory2} {
		free := (db.EquipAccessory1 | db.EquipAccessory2) &^ worn
		ring := session.InventoryItem{Index: 11, ItemID: 2601, Type: db.ItemTypeArmor,
			Location: db.EquipAccessory1 | db.EquipAccessory2, WearLocation: worn, Equip: true, Equipped: true}
		s := &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{ring}}}
		if item, ok := equippedItemForSlot(s, worn); !ok || item.Index != ring.Index {
			t.Fatalf("missing Ring in worn slot 0x%04X", worn)
		}
		if _, ok := equippedItemForSlot(s, free); ok {
			t.Fatalf("Ring duplicated in free slot 0x%04X", free)
		}
		next := session.InventoryItem{Index: 12, ItemID: 2602, Type: db.ItemTypeArmor,
			Location: db.EquipAccessory1 | db.EquipAccessory2, Equip: true}
		if got := inventoryItemEquipLocationForSession(s, next); got != free {
			t.Fatalf("next accessory chooses 0x%04X, want free slot 0x%04X", got, free)
		}
		before := equipmentSnapshot(s)
		s.Inventory.Items[0].WearLocation = free
		if equipmentSnapshot(s) == before {
			t.Fatal("moving the Ring did not invalidate the equipment window")
		}
	}
}

func TestEquipmentWindowKeepsGenuineMultipleSlotEquipment(t *testing.T) {
	item := session.InventoryItem{Index: 11, ItemID: 1151, Location: db.EquipWeapon | db.EquipShield,
		WearLocation: db.EquipWeapon | db.EquipShield, Equip: true, Equipped: true}
	s := &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{item}}}
	for _, slot := range []uint16{db.EquipWeapon, db.EquipShield} {
		if got, ok := equippedItemForSlot(s, slot); !ok || got.Index != item.Index {
			t.Fatalf("two-handed item missing from slot 0x%04X", slot)
		}
	}
}

func TestViewedEquipmentUsesOccupiedAccessorySlot(t *testing.T) {
	w := &ViewEquipmentWindow{}
	w.Open(Context{ScreenW: 800, ScreenH: 600}, network.ViewedEquipment{
		Name: "Alice",
		Items: []network.InventoryItem{{Index: 11, ItemID: 2601, Type: db.ItemTypeArmor,
			Location: db.EquipAccessory1 | db.EquipAccessory2, WearLocation: db.EquipAccessory2,
			Amount: 1, Equip: true, Equipped: true}},
	}, nil)
	if _, ok := w.itemForSlot(db.EquipAccessory1); ok {
		t.Fatal("viewed Ring duplicated in the empty slot")
	}
	if item, ok := w.itemForSlot(db.EquipAccessory2); !ok || item.ItemID != 2601 {
		t.Fatal("viewed Ring missing from its occupied slot")
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

func TestEquipmentWindowHeightHasNoCartContentRow(t *testing.T) {
	want := ROWindowTitleHeight + equipmentWindowPad*2 + equipmentPreviewImageH + equipmentRowH + ROWindowFooterHeight
	if equipmentWindowHeight != want {
		t.Fatalf("equipment window height = %d, want content-fitting height %d", equipmentWindowHeight, want)
	}
}

func TestEquipmentWindowCartActionsAreInFooter(t *testing.T) {
	window := &EquipmentWindow{}
	withoutCart := window.footerWidgets(Context{Session: &session.Session{}}, nil, nil)
	if len(withoutCart) != 2 {
		t.Fatalf("footer children without cart = %d, want checkbox and spacer", len(withoutCart))
	}
	withCart := window.footerWidgets(Context{Session: &session.Session{Cart: session.Cart{MaxAmount: 1}}}, nil, &CartWindow{})
	if len(withCart) != 4 {
		t.Fatalf("footer children with cart = %d, want checkbox, spacer, Cart, and Cart Off", len(withCart))
	}
}

func TestEquipmentWindowPecoActionIsInFooter(t *testing.T) {
	window := &EquipmentWindow{}
	ctx := Context{Session: &session.Session{
		Selected: session.Character{ID: 1, Job: db.JobKnight, Option: db.EffectStateRiding},
	}}

	children := window.footerWidgets(ctx, nil, nil)
	if len(children) != 3 {
		t.Fatalf("footer children with Peco = %d, want checkbox, spacer, and Peco Off", len(children))
	}
	if got := equipmentRemoveOptionLabel(ctx); got != "Peco Off" {
		t.Fatalf("remove option label = %q, want Peco Off", got)
	}
}

func TestEquipmentWindowPecoStatePrefersCurrentWorldState(t *testing.T) {
	ctx := Context{
		Session: &session.Session{
			Selected: session.Character{ID: 1, Job: db.JobKnight, Option: db.EffectStateRiding},
		},
		World: &world.World{Player: world.Actor{ID: 1, HasState: true}},
	}

	if equipmentHasPeco(ctx) {
		t.Fatal("current unmounted world state should override stale selected-character state")
	}
}

func TestEquipmentWindowTooltipTracksHoveredItem(t *testing.T) {
	window := EquipmentWindow{}
	item := session.InventoryItem{Index: 3, ItemID: 1201, Identified: true}
	ctx := Context{Input: input.NewState(), ScreenW: 800, ScreenH: 600, UIManager: NewManager()}

	window.showTooltip(ctx, item)
	if !window.tooltip.Open() {
		t.Fatal("tooltip should be open")
	}
	if got := window.tooltip.Text(); got != "item 1201" {
		t.Fatalf("tooltip text = %q", got)
	}

	window.hideTooltip()
	if window.tooltip.Open() {
		t.Fatal("tooltip not cleared")
	}
}

func TestInventoryEquipLocationChoosesFreeAccessorySlot(t *testing.T) {
	item := session.InventoryItem{Index: 7, ItemID: 2601, Type: db.ItemTypeArmor, Location: db.EquipAccessory1 | db.EquipAccessory2, Equip: true}
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 2601, Type: db.ItemTypeArmor, Location: db.EquipAccessory1, WearLocation: db.EquipAccessory1, Equip: true, Equipped: true},
			},
		},
	}

	if got := inventoryItemEquipLocationForSession(s, item); got != db.EquipAccessory2 {
		t.Fatalf("location = 0x%04X, want second accessory slot 0x%04X", got, db.EquipAccessory2)
	}
}

func TestInventoryEquipLocationKeepsAccessoryMaskWhenBothSlotsFull(t *testing.T) {
	item := session.InventoryItem{Index: 7, ItemID: 2601, Type: db.ItemTypeArmor, Location: db.EquipAccessory1 | db.EquipAccessory2, Equip: true}
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 2601, Type: db.ItemTypeArmor, Location: db.EquipAccessory1, WearLocation: db.EquipAccessory1, Equip: true, Equipped: true},
				{Index: 4, ItemID: 2602, Type: db.ItemTypeArmor, Location: db.EquipAccessory2, WearLocation: db.EquipAccessory2, Equip: true, Equipped: true},
			},
		},
	}

	if got := inventoryItemEquipLocationForSession(s, item); got != db.EquipAccessory1 {
		t.Fatalf("location = 0x%04X, want first accessory replacement 0x%04X", got, db.EquipAccessory1)
	}
}

func TestEquipmentWindowConsumesInventoryDrop(t *testing.T) {
	window := EquipmentWindow{}
	window.EnsureWindow(equipmentWindowWidth, equipmentWindowHeight)
	window.Window.OpenAt(32, 32, nil)

	ok := window.AcceptInventoryDrop(Context{}, session.InventoryItem{Index: 7, ItemID: 2601, Type: db.ItemTypeArmor, Equip: true}, 40, 40)
	if !ok {
		t.Fatal("drop over equipment window was not consumed")
	}
}

func TestEquipmentWindowSingleClickDoesNotUnequip(t *testing.T) {
	window := EquipmentWindow{}
	item := session.InventoryItem{Index: 7, ItemID: 2601, Type: db.ItemTypeArmor, Equip: true, Equipped: true}

	window.activateItem(Context{}, item)
	if window.lastClickItem != item.Index {
		t.Fatalf("first click lastClickItem = %d, want %d", window.lastClickItem, item.Index)
	}

	window.activateItem(Context{}, item)
	if window.lastClickItem != 0 {
		t.Fatalf("second click lastClickItem = %d, want reset after double click", window.lastClickItem)
	}
}
