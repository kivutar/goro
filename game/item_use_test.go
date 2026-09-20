package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func itemUseTestContext() client.Context {
	return client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150004,
			Inventory: session.Inventory{Items: []session.InventoryItem{
				{Index: 12, ItemID: 604, Type: db.ItemTypeUsable, Amount: 4, Identified: true},
			}},
			Hotkeys: session.Hotkeys{
				Loaded:  true,
				Version: 1,
				Slots:   []session.HotkeySlot{{Type: network.HotkeyTypeItem, ID: 604}},
			},
		},
		World: worldstate.New(),
	}
}

// rAthena broadcasts successful ZC_USE_ITEM_ACK2 packets to nearby players.
func itemUseAckTestPacket(aid uint32, itemID, amount uint16, result byte) network.Packet {
	data := make([]byte, 13)
	binary.LittleEndian.PutUint16(data[0:2], 0x01C8)
	binary.LittleEndian.PutUint16(data[2:4], 12)
	binary.LittleEndian.PutUint16(data[4:6], itemID)
	binary.LittleEndian.PutUint32(data[6:10], aid)
	binary.LittleEndian.PutUint16(data[10:12], amount)
	data[12] = result
	return network.Packet{ID: 0x01C8, Data: data}
}

func TestNearbyItemUseDoesNotChangeOwnInventory(t *testing.T) {
	for _, tc := range []struct {
		name   string
		itemID uint16
		amount uint16
		charID uint32
	}{
		{"last dead branch", 604, 0, 150004},
		{"larger dead branch stack", 604, 9, 150004},
		{"potion at same inventory index", 501, 0, 150004},
		{"nearby account matches own character ID", 604, 0, 2000001},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := itemUseTestContext()
			ctx.Session.CharID = tc.charID
			ctx.World.UpsertActor(worldstate.Actor{ID: 2000001, X: 10, Y: 20})
			mode := &WorldMode{}
			mode.ui.shortcutBar.SyncFromSession(ctx)
			mode.handleNetworkPacket(ctx, itemUseAckTestPacket(2000001, tc.itemID, tc.amount, 1), time.Now())
			if items := ctx.Session.Inventory.Items; len(items) != 1 || items[0].Amount != 4 {
				t.Fatalf("nearby item use changed own inventory: %+v", items)
			}
			if slot := ctx.Session.Hotkeys.Slots[0]; slot.ID != 604 {
				t.Fatalf("nearby item use cleared own shortcut: %+v", slot)
			}
			if tc.itemID == 501 {
				if len(mode.worldEffects) != 1 || mode.worldEffects[0].actorID != 2000001 {
					t.Fatalf("nearby potion effect missing or attached to wrong actor: %+v", mode.worldEffects)
				}
			}
		})
	}
}

func TestApplyUseItemAckIgnoresOtherPlayer(t *testing.T) {
	ctx := itemUseTestContext()
	// A character ID can overlap another player's account ID. Only the
	// account ID identifies whose inventory this acknowledgement updates.
	ctx.Session.CharID = 2000001
	applyUseItemAck(ctx, network.UseItemAck{AID: 2000001, Index: 12, ItemID: 604, Amount: 0, Result: 1})
	if items := ctx.Session.Inventory.Items; len(items) != 1 || items[0].Amount != 4 {
		t.Fatalf("nearby item use changed own inventory: %+v", items)
	}
}

func TestUnconsumedItemUsePreservesInventoryWithoutFalseError(t *testing.T) {
	// rAthena pc_useitem sends result=0 both for rejected uses and for
	// successful rental/reusable item uses. The ACK cannot distinguish them.
	for _, tc := range []struct {
		name   string
		itemID uint16
	}{
		{"rejected dead branch", 604},
		{"successful rental potion", 501},
		{"successful reins of mount", 12622},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, packet := range []network.Packet{
				itemUseAckTestPacket(2000000, tc.itemID, 0, 0),
				{ID: 0x00A8, Data: []byte{0xA8, 0, 12, 0, 0, 0, 0}},
			} {
				ctx := itemUseTestContext()
				ctx.Session.Inventory.Items[0].ItemID = tc.itemID
				ctx.Session.Hotkeys.Slots[0].ID = uint32(tc.itemID)
				mode := &WorldMode{}
				mode.ui.shortcutBar.SyncFromSession(ctx)
				mode.handleNetworkPacket(ctx, packet, time.Now())
				if items := ctx.Session.Inventory.Items; len(items) != 1 || items[0].ItemID != tc.itemID || items[0].Amount != 4 {
					t.Fatalf("packet 0x%04X: unconsumed item changed inventory: %+v", packet.ID, items)
				}
				if slot := ctx.Session.Hotkeys.Slots[0]; slot.ID != uint32(tc.itemID) {
					t.Fatalf("packet 0x%04X: unconsumed item cleared shortcut: %+v", packet.ID, slot)
				}
				if messages := mode.ui.console.Messages(); len(messages) != 0 {
					t.Fatalf("packet 0x%04X: ambiguous ACK produced a false error: %+v", packet.ID, messages)
				}
			}
		})
	}
}

func TestDeadBranchUseAndMonsterSpawn2008(t *testing.T) {
	ctx := itemUseTestContext()
	mode := &WorldMode{}
	mode.ui.shortcutBar.SyncFromSession(ctx)
	ack := itemUseAckTestPacket(2000000, 604, 3, 1)
	// rAthena packet_spawn_unit2: monsters use 0x007C, with job at offset
	// 21 and position at offset 37 (different from the idle entry 0x0078).
	spawn := make([]byte, 42)
	binary.LittleEndian.PutUint16(spawn[0:2], 0x007C)
	spawn[2] = 5 // monster
	binary.LittleEndian.PutUint32(spawn[3:7], 110000001)
	binary.LittleEndian.PutUint16(spawn[7:9], 400)
	binary.LittleEndian.PutUint16(spawn[21:23], 1002) // Poring
	spawn[37], spawn[38], spawn[39] = 25, 6, 66       // x=100, y=100, dir=2
	packets, err := network.NewFramer(network.PacketLengths2008()).Push(append(ack.Data, spawn...))
	if err != nil || len(packets) != 2 {
		t.Fatalf("frame item use and spawn: packets=%+v err=%v", packets, err)
	}
	for _, packet := range packets {
		mode.handleNetworkPacket(ctx, packet, time.Now())
	}
	if got := ctx.Session.Inventory.Items[0].Amount; got != 3 {
		t.Fatalf("dead branch count = %d, want 3", got)
	}
	actor, ok := ctx.World.Actors[110000001]
	if !ok || actor.Job != 1002 || actor.X != 100 || actor.Y != 100 || !actorHasMobObjectType(actor) {
		t.Fatalf("spawned monster missing or malformed: %+v", actor)
	}
	if messages := mode.ui.console.Messages(); len(messages) != 0 {
		t.Fatalf("successful use produced an error: %+v", messages)
	}
}
