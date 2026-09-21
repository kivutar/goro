package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func accessoryListPacket(firstSlot, secondSlot uint16) network.Packet {
	data := make([]byte, 4+2*26)
	binary.LittleEndian.PutUint16(data[0:2], 0x02D0)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	for i, worn := range []uint16{firstSlot, secondSlot} {
		entry := data[4+i*26:]
		binary.LittleEndian.PutUint16(entry[0:2], uint16(11+i))
		binary.LittleEndian.PutUint16(entry[2:4], uint16(2601+i))
		entry[4], entry[5] = db.ItemTypeArmor, 1
		binary.LittleEndian.PutUint16(entry[6:8], db.EquipAccessory1|db.EquipAccessory2)
		binary.LittleEndian.PutUint16(entry[8:10], worn)
	}
	return network.Packet{ID: 0x02D0, Data: data}
}

func TestAccessorySlotsSurviveEquipReloadAndUnequip(t *testing.T) {
	ctx := client.Context{Session: &session.Session{}}
	mode := &WorldMode{}
	apply := func(packet network.Packet) {
		t.Helper()
		mode.handleNetworkPacket(ctx, packet, time.Now())
	}
	check := func(index, worn uint16) {
		t.Helper()
		item, ok := findSessionInventoryItem(ctx.Session, index)
		if !ok || !item.Equip || item.Location != db.EquipAccessory1|db.EquipAccessory2 || item.WearLocation != worn || item.Equipped != (worn != 0) {
			t.Fatalf("index %d: %+v, want both slots allowed and wear 0x%04X", index, item, worn)
		}
	}
	apply(accessoryListPacket(0, db.EquipAccessory2))
	apply(network.Packet{ID: 0x00AA, Data: []byte{0xAA, 0, 11, 0, 8, 0, 1}})
	check(11, db.EquipAccessory1)
	check(12, db.EquipAccessory2)

	// A failed attempt must not move either accessory.
	apply(network.Packet{ID: 0x00AA, Data: []byte{0xAA, 0, 12, 0, 8, 0, 0}})
	check(11, db.EquipAccessory1)
	check(12, db.EquipAccessory2)

	// Login/map-load inventory lists carry both allowed slots and the worn slot.
	apply(accessoryListPacket(db.EquipAccessory1, db.EquipAccessory2))
	check(11, db.EquipAccessory1)
	check(12, db.EquipAccessory2)

	apply(network.Packet{ID: 0x00AC, Data: []byte{0xAC, 0, 11, 0, 8, 0, 1}})
	check(11, 0)
	check(12, db.EquipAccessory2)

	// The Ring may now use the other slot, replacing only its occupant.
	apply(network.Packet{ID: 0x00AA, Data: []byte{0xAA, 0, 11, 0, 0x80, 0, 1}})
	check(11, db.EquipAccessory2)
	check(12, 0)
}

func TestEquipmentAppearanceUsesOccupiedHand(t *testing.T) {
	ctx := client.Context{Session: &session.Session{}}
	applyInventoryItemList(ctx, []network.InventoryItem{
		{Index: 11, ItemID: 1201, Type: db.ItemTypeWeapon, Location: db.EquipWeapon | db.EquipShield, WearLocation: db.EquipShield, Equip: true, Equipped: true},
		{Index: 12, ItemID: 1202, Type: db.ItemTypeWeapon, Location: db.EquipWeapon | db.EquipShield, WearLocation: db.EquipWeapon, Equip: true, Equipped: true},
	})
	if ctx.Session.Selected.Weapon != 1202 || ctx.Session.Selected.Shield != 1201 {
		t.Fatalf("weapon=%d shield=%d, want 1202/1201", ctx.Session.Selected.Weapon, ctx.Session.Selected.Shield)
	}
}

func coupleStatusPacket(statusID uint32, base, bonus int32) network.Packet {
	data := make([]byte, 14)
	binary.LittleEndian.PutUint16(data[0:2], 0x0141)
	binary.LittleEndian.PutUint32(data[2:6], statusID)
	binary.LittleEndian.PutUint32(data[6:10], uint32(base))
	binary.LittleEndian.PutUint32(data[10:14], uint32(bonus))
	return network.Packet{ID: 0x0141, Data: data}
}

func TestPrimaryStatBonusesUpdateFromServer(t *testing.T) {
	ctx := client.Context{Session: &session.Session{}}
	mode := &WorldMode{}
	for _, bonus := range []int32{2, 0, -3} {
		for i := uint32(0); i < 6; i++ {
			mode.handleNetworkPacket(ctx, coupleStatusPacket(uint32(network.StatusStr)+i, 10+int32(i), bonus), time.Now())
		}
		stats := ctx.Session.Stats
		bases := [6]int{stats.Str, stats.Agi, stats.Vit, stats.Int, stats.Dex, stats.Luk}
		bonuses := [6]int{stats.StrBonus, stats.AgiBonus, stats.VitBonus, stats.IntBonus, stats.DexBonus, stats.LukBonus}
		selected := ctx.Session.Selected
		selectedBases := [6]uint8{selected.Str, selected.Agi, selected.Vit, selected.Int, selected.Dex, selected.Luk}
		for i := range bases {
			if bases[i] != 10+i || bonuses[i] != int(bonus) || int(selectedBases[i]) != bases[i] {
				t.Fatalf("stat %d: base=%d bonus=%d selected=%d, want %d/%d/%d", i, bases[i], bonuses[i], selectedBases[i], 10+i, bonus, 10+i)
			}
		}
	}
	// The ID is 32 bits on the wire; an unknown ID must not alias STR.
	before := ctx.Session.Stats
	mode.handleNetworkPacket(ctx, coupleStatusPacket(0x10000|uint32(network.StatusStr), 99, 99), time.Now())
	if ctx.Session.Stats != before {
		t.Fatal("unknown status ID changed primary stats")
	}
	// A base-only status snapshot carries no bonus values.
	data := make([]byte, 44)
	binary.LittleEndian.PutUint16(data[0:2], 0x00BD)
	for i := 0; i < 6; i++ {
		data[4+i*2] = byte(10 + i)
	}
	mode.handleNetworkPacket(ctx, network.Packet{ID: 0x00BD, Data: data}, time.Now())
	if ctx.Session.Stats != before {
		t.Fatal("status snapshot cleared bonuses")
	}
}

func TestLoginAcceptsPrimaryStatBonuses(t *testing.T) {
	ctx := client.Context{Session: &session.Session{}}
	mode := &LoginMode{}
	if !mode.applyLoginParameterChange(ctx, coupleStatusPacket(uint32(network.StatusStr), 10, 2)) {
		t.Fatal("login did not handle ZC_COUPLESTATUS")
	}
	if ctx.Session.Stats.Str != 10 || ctx.Session.Stats.StrBonus != 2 {
		t.Fatalf("login stats: %+v", ctx.Session.Stats)
	}
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusStr, Value: 11})
	if ctx.Session.Stats.Str != 11 || ctx.Session.Stats.StrBonus != 2 {
		t.Fatalf("base-only update changed the bonus: %+v", ctx.Session.Stats)
	}
}
