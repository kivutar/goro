package network

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestParseFloorItemEntryExistingItem(t *testing.T) {
	data := make([]byte, 17)
	binary.LittleEndian.PutUint16(data[0:2], 0x009D)
	binary.LittleEndian.PutUint32(data[2:6], 1001)
	binary.LittleEndian.PutUint16(data[6:8], 909)
	data[8] = 1
	binary.LittleEndian.PutUint16(data[9:11], 150)
	binary.LittleEndian.PutUint16(data[11:13], 160)
	binary.LittleEndian.PutUint16(data[13:15], 3)
	data[15] = 4
	data[16] = 8

	item, ok, err := ParseFloorItemEntry(Packet{ID: 0x009D, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not parsed")
	}
	if item.ID != 1001 || item.ItemID != 909 || !item.Identified || item.X != 150 || item.Y != 160 || item.Amount != 3 || item.SubX != 4 || item.SubY != 8 || item.Falling {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestParseFloorItemEntryFallingItem(t *testing.T) {
	data := make([]byte, 17)
	binary.LittleEndian.PutUint16(data[0:2], 0x009E)
	binary.LittleEndian.PutUint32(data[2:6], 2002)
	binary.LittleEndian.PutUint16(data[6:8], 512)
	data[8] = 0
	binary.LittleEndian.PutUint16(data[9:11], 30)
	binary.LittleEndian.PutUint16(data[11:13], 40)
	data[13] = 5
	data[14] = 7
	binary.LittleEndian.PutUint16(data[15:17], 9)

	item, ok, err := ParseFloorItemEntry(Packet{ID: 0x009E, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not parsed")
	}
	if item.ID != 2002 || item.ItemID != 512 || item.Identified || item.X != 30 || item.Y != 40 || item.Amount != 9 || item.SubX != 5 || item.SubY != 7 || !item.Falling {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestParseFloorItemDisappear(t *testing.T) {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[0:2], 0x00A1)
	binary.LittleEndian.PutUint32(data[2:6], 3003)

	disappear, ok, err := ParseFloorItemDisappear(Packet{ID: 0x00A1, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || disappear.ID != 3003 {
		t.Fatalf("unexpected disappear: ok=%v value=%+v", ok, disappear)
	}
}

func TestParseItemPickupAckLegacy(t *testing.T) {
	data := make([]byte, 23)
	binary.LittleEndian.PutUint16(data[0:2], 0x00A0)
	binary.LittleEndian.PutUint16(data[2:4], 12)
	binary.LittleEndian.PutUint16(data[4:6], 3)
	binary.LittleEndian.PutUint16(data[6:8], 938)
	data[8] = 1
	binary.LittleEndian.PutUint16(data[19:21], 0x0002)
	data[21] = 5
	data[22] = 0

	ack, ok, err := ParseItemPickupAck(Packet{ID: 0x00A0, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not parsed")
	}
	if ack.Index != 12 || ack.Amount != 3 || ack.ItemID != 938 || ack.Location != 0x0002 || ack.Type != 5 || !ack.Identified || ack.Result != 0 {
		t.Fatalf("unexpected ack: %+v", ack)
	}
}

func TestParseItemPickupAckExtendedVariants(t *testing.T) {
	for _, tc := range []struct {
		id             uint16
		length         int
		locationOffset int
		locationSize   int
		typeOffset     int
		resultOffset   int
	}{
		{id: 0x029A, length: 27, locationOffset: 19, locationSize: 2, typeOffset: 21, resultOffset: 22},
		{id: 0x02D4, length: 29, locationOffset: 19, locationSize: 2, typeOffset: 21, resultOffset: 22},
		{id: 0x0990, length: 31, locationOffset: 19, locationSize: 4, typeOffset: 23, resultOffset: 24},
		{id: 0x0A0C, length: 56, locationOffset: 19, locationSize: 4, typeOffset: 23, resultOffset: 24},
		{id: 0x0A37, length: 59, locationOffset: 19, locationSize: 4, typeOffset: 23, resultOffset: 24},
	} {
		t.Run(fmt.Sprintf("0x%04X", tc.id), func(t *testing.T) {
			data := make([]byte, tc.length)
			binary.LittleEndian.PutUint16(data[0:2], tc.id)
			binary.LittleEndian.PutUint16(data[2:4], 31)
			binary.LittleEndian.PutUint16(data[4:6], 2)
			binary.LittleEndian.PutUint16(data[6:8], 909)
			data[8] = 1
			if tc.locationSize == 4 {
				binary.LittleEndian.PutUint32(data[tc.locationOffset:tc.locationOffset+4], 0x00000020)
			} else {
				binary.LittleEndian.PutUint16(data[tc.locationOffset:tc.locationOffset+2], 0x0020)
			}
			data[tc.typeOffset] = 5
			data[tc.resultOffset] = 0

			ack, ok, err := ParseItemPickupAck(Packet{ID: tc.id, Data: data})
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("not parsed")
			}
			if ack.Index != 31 || ack.Amount != 2 || ack.ItemID != 909 || ack.Location != 0x0020 || ack.Type != 5 || !ack.Identified || ack.Result != 0 {
				t.Fatalf("unexpected ack: %+v", ack)
			}
		})
	}
}

func TestParseUseItemAckLegacy(t *testing.T) {
	data := make([]byte, 7)
	binary.LittleEndian.PutUint16(data[0:2], 0x00A8)
	binary.LittleEndian.PutUint16(data[2:4], 12)
	binary.LittleEndian.PutUint16(data[4:6], 3)
	data[6] = 1

	ack, ok, err := ParseUseItemAck(Packet{ID: 0x00A8, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not parsed")
	}
	if ack.Index != 12 || ack.Amount != 3 || ack.Result != 1 {
		t.Fatalf("unexpected use ack: %+v", ack)
	}
}

func TestParseUseItemAck2(t *testing.T) {
	data := make([]byte, 13)
	binary.LittleEndian.PutUint16(data[0:2], 0x01C8)
	binary.LittleEndian.PutUint16(data[2:4], 12)
	binary.LittleEndian.PutUint16(data[4:6], 512)
	binary.LittleEndian.PutUint32(data[6:10], 0x11223344)
	binary.LittleEndian.PutUint16(data[10:12], 3)
	data[12] = 1

	ack, ok, err := ParseUseItemAck(Packet{ID: 0x01C8, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not parsed")
	}
	if ack.Index != 12 || ack.ItemID != 512 || ack.AID != 0x11223344 || ack.Amount != 3 || ack.Result != 1 {
		t.Fatalf("unexpected use ack: %+v", ack)
	}
}

func TestBuildItemPickupPacket(t *testing.T) {
	packet := BuildItemPickupPacket(0x11223344)
	if len(packet) != 6 {
		t.Fatalf("len = %d, want 6", len(packet))
	}
	if got := ID(packet); got != 0x009F {
		t.Fatalf("opcode = 0x%04X, want 0x009F", got)
	}
	if got := binary.LittleEndian.Uint32(packet[2:6]); got != 0x11223344 {
		t.Fatalf("gid = 0x%08X", got)
	}
}

func TestBuildItemPickupPacketForClientDate20080910(t *testing.T) {
	packet := BuildItemPickupPacketForClientDate(0x11223344, 20080910)
	if len(packet) != 8 {
		t.Fatalf("len = %d, want 8", len(packet))
	}
	if got := ID(packet); got != 0x00F5 {
		t.Fatalf("opcode = 0x%04X, want 0x00F5", got)
	}
	if got := binary.LittleEndian.Uint32(packet[4:8]); got != 0x11223344 {
		t.Fatalf("gid = 0x%08X", got)
	}
	if padding := binary.LittleEndian.Uint16(packet[2:4]); padding != 0 {
		t.Fatalf("padding = 0x%04X, want 0", padding)
	}
}

func TestBuildItemPickupPacketForClientDateShuffledBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		clientDate int
		opcode     uint16
		length     int
		offset     int
	}{
		{name: "legacy", clientDate: 20040712, opcode: 0x009F, length: 6, offset: 2},
		{name: "20040713", clientDate: 20040713, opcode: 0x009F, length: 10, offset: 6},
		{name: "20040726", clientDate: 20040726, opcode: 0x0094, length: 10, offset: 6},
		{name: "20070212", clientDate: 20070212, opcode: 0x00F5, length: 8, offset: 4},
		{name: "20101124", clientDate: 20101124, opcode: 0x0362, length: 6, offset: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			packet := BuildItemPickupPacketForClientDate(0x11223344, tc.clientDate)
			if len(packet) != tc.length {
				t.Fatalf("len = %d, want %d", len(packet), tc.length)
			}
			if got := ID(packet); got != tc.opcode {
				t.Fatalf("opcode = 0x%04X, want 0x%04X", got, tc.opcode)
			}
			if got := binary.LittleEndian.Uint32(packet[tc.offset : tc.offset+4]); got != 0x11223344 {
				t.Fatalf("gid = 0x%08X", got)
			}
		})
	}
}

func TestBuildUseAndEquipItemPackets(t *testing.T) {
	use := BuildUseInventoryItemPacketForClientDate(7, 0x11223344, 20080910)
	if len(use) != 8 || ID(use) != 0x0439 {
		t.Fatalf("unexpected use packet header: % X", use)
	}
	if got := binary.LittleEndian.Uint16(use[2:4]); got != 7 {
		t.Fatalf("use item index = %d, want 7", got)
	}
	if got := binary.LittleEndian.Uint32(use[4:8]); got != 0x11223344 {
		t.Fatalf("use target = 0x%08X", got)
	}

	useLegacy := BuildUseInventoryItemPacketForClientDate(7, 0x11223344, 20070212)
	if len(useLegacy) != 14 || ID(useLegacy) != 0x009F {
		t.Fatalf("unexpected legacy use packet header: % X", useLegacy)
	}
	if got := binary.LittleEndian.Uint16(useLegacy[4:6]); got != 7 {
		t.Fatalf("legacy use item index = %d, want 7", got)
	}
	if got := binary.LittleEndian.Uint32(useLegacy[10:14]); got != 0x11223344 {
		t.Fatalf("legacy use target = 0x%08X", got)
	}

	use2 := BuildUseInventoryItemPacketForClientDate(7, 0x11223344, 20180307)
	if len(use2) != 8 || ID(use2) != 0x0439 || binary.LittleEndian.Uint16(use2[2:4]) != 7 || binary.LittleEndian.Uint32(use2[4:8]) != 0x11223344 {
		t.Fatalf("unexpected modern use packet: % X", use2)
	}

	equip := BuildWearEquipPacket(11, 0x0002)
	if len(equip) != 6 || ID(equip) != 0x00A9 || binary.LittleEndian.Uint16(equip[2:4]) != 11 || binary.LittleEndian.Uint16(equip[4:6]) != 0x0002 {
		t.Fatalf("unexpected equip packet: % X", equip)
	}

	takeoff := BuildTakeoffEquipPacket(11)
	if len(takeoff) != 4 || ID(takeoff) != 0x00AB || binary.LittleEndian.Uint16(takeoff[2:4]) != 11 {
		t.Fatalf("unexpected takeoff packet: % X", takeoff)
	}
}

func TestBuildDropInventoryItemPacket(t *testing.T) {
	packet := BuildDropInventoryItemPacket(9, 2)
	if len(packet) != 6 || ID(packet) != PacketCZItemThrow {
		t.Fatalf("unexpected drop packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 9 {
		t.Fatalf("drop item index = %d, want 9", got)
	}
	if got := binary.LittleEndian.Uint16(packet[4:6]); got != 2 {
		t.Fatalf("drop amount = %d, want 2", got)
	}

	zero := BuildDropInventoryItemPacket(9, 0)
	if got := binary.LittleEndian.Uint16(zero[4:6]); got != 1 {
		t.Fatalf("zero drop amount = %d, want 1", got)
	}
}

func TestBuildDropInventoryItemPacketFor20080910(t *testing.T) {
	packet := BuildDropInventoryItemPacketForClientDate(15, 1, 20080910)
	if len(packet) != 10 || ID(packet) != 0x0116 {
		t.Fatalf("unexpected 20080910 drop packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint16(packet[5:7]); got != 15 {
		t.Fatalf("20080910 drop index = %d, want 15", got)
	}
	if got := binary.LittleEndian.Uint16(packet[8:10]); got != 1 {
		t.Fatalf("20080910 drop amount = %d, want 1", got)
	}
}

func TestParseItemIdentifyListAndAck(t *testing.T) {
	listData := make([]byte, 8)
	binary.LittleEndian.PutUint16(listData[0:2], 0x0177)
	binary.LittleEndian.PutUint16(listData[2:4], uint16(len(listData)))
	binary.LittleEndian.PutUint16(listData[4:6], 7)
	binary.LittleEndian.PutUint16(listData[6:8], 9)

	list, ok, err := ParseItemIdentifyList(Packet{ID: 0x0177, Data: listData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(list.Indexes) != 2 || list.Indexes[0] != 7 || list.Indexes[1] != 9 {
		t.Fatalf("unexpected identify list ok=%v value=%+v", ok, list)
	}

	ackData := make([]byte, 5)
	binary.LittleEndian.PutUint16(ackData[0:2], 0x0179)
	binary.LittleEndian.PutUint16(ackData[2:4], 9)
	ackData[4] = 0
	ack, ok, err := ParseItemIdentifyAck(Packet{ID: 0x0179, Data: ackData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || ack.Index != 9 || !ack.Success {
		t.Fatalf("unexpected identify ack ok=%v value=%+v", ok, ack)
	}
}

func TestParseItemCompositionListAndAck(t *testing.T) {
	listData := make([]byte, 8)
	binary.LittleEndian.PutUint16(listData[0:2], 0x017B)
	binary.LittleEndian.PutUint16(listData[2:4], uint16(len(listData)))
	binary.LittleEndian.PutUint16(listData[4:6], 11)
	binary.LittleEndian.PutUint16(listData[6:8], 12)

	list, ok, err := ParseItemCompositionList(Packet{ID: 0x017B, Data: listData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(list.Indexes) != 2 || list.Indexes[0] != 11 || list.Indexes[1] != 12 {
		t.Fatalf("composition list ok=%v value=%+v", ok, list)
	}

	ackData := make([]byte, 7)
	binary.LittleEndian.PutUint16(ackData[0:2], 0x017D)
	binary.LittleEndian.PutUint16(ackData[2:4], 11)
	binary.LittleEndian.PutUint16(ackData[4:6], 7)
	ackData[6] = 0
	ack, ok, err := ParseItemCompositionAck(Packet{ID: 0x017D, Data: ackData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || ack.EquipIndex != 11 || ack.CardIndex != 7 || !ack.Success {
		t.Fatalf("composition ack ok=%v value=%+v", ok, ack)
	}
}

func TestParseMakingArrowList(t *testing.T) {
	listData := make([]byte, 8)
	binary.LittleEndian.PutUint16(listData[0:2], 0x01AD)
	binary.LittleEndian.PutUint16(listData[2:4], uint16(len(listData)))
	binary.LittleEndian.PutUint16(listData[4:6], 909)
	binary.LittleEndian.PutUint16(listData[6:8], 914)

	list, ok, err := ParseMakingArrowList(Packet{ID: 0x01AD, Data: listData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(list.ItemIDs) != 2 || list.ItemIDs[0] != 909 || list.ItemIDs[1] != 914 {
		t.Fatalf("making arrow list ok=%v value=%+v", ok, list)
	}
}

func TestParseMakableItemList(t *testing.T) {
	listData := make([]byte, 20)
	binary.LittleEndian.PutUint16(listData[0:2], PacketZCMakableItemList)
	binary.LittleEndian.PutUint16(listData[2:4], uint16(len(listData)))
	binary.LittleEndian.PutUint16(listData[4:6], 501)
	binary.LittleEndian.PutUint16(listData[6:8], 1000)
	binary.LittleEndian.PutUint16(listData[8:10], 990)
	binary.LittleEndian.PutUint16(listData[10:12], 5)
	binary.LittleEndian.PutUint16(listData[12:14], 713)
	binary.LittleEndian.PutUint16(listData[14:16], 0)
	binary.LittleEndian.PutUint16(listData[16:18], 0)
	binary.LittleEndian.PutUint16(listData[18:20], 0)

	list, ok, err := ParseMakableItemList(Packet{ID: PacketZCMakableItemList, Data: listData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(list.Items) != 2 {
		t.Fatalf("makable item list ok=%v value=%+v", ok, list)
	}
	if list.Items[0].ItemID != 501 || list.Items[0].Material != [3]uint16{1000, 990, 5} {
		t.Fatalf("first makable item = %+v", list.Items[0])
	}
	if list.Items[1].ItemID != 713 || list.Items[1].Material != [3]uint16{} {
		t.Fatalf("second makable item = %+v", list.Items[1])
	}
}

func TestParseMakingItemList(t *testing.T) {
	listData := make([]byte, 10)
	binary.LittleEndian.PutUint16(listData[0:2], PacketZCMakingItemList)
	binary.LittleEndian.PutUint16(listData[2:4], uint16(len(listData)))
	binary.LittleEndian.PutUint16(listData[4:6], 1)
	binary.LittleEndian.PutUint16(listData[6:8], 713)
	binary.LittleEndian.PutUint16(listData[8:10], 714)

	list, ok, err := ParseMakingItemList(Packet{ID: PacketZCMakingItemList, Data: listData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || list.Type != 1 || len(list.Items) != 2 || list.Items[0].ItemID != 713 || list.Items[1].ItemID != 714 {
		t.Fatalf("making item list ok=%v value=%+v", ok, list)
	}
}

func TestParseMakingItemAck(t *testing.T) {
	ackData := make([]byte, 6)
	binary.LittleEndian.PutUint16(ackData[0:2], PacketZCAckReqMakingItem)
	binary.LittleEndian.PutUint16(ackData[2:4], 2)
	binary.LittleEndian.PutUint16(ackData[4:6], 713)

	ack, ok, err := ParseMakingItemAck(Packet{ID: PacketZCAckReqMakingItem, Data: ackData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || ack.ItemID != 713 || ack.Result != 2 || !ack.Success() || !ack.Alchemist() {
		t.Fatalf("making item ack ok=%v value=%+v", ok, ack)
	}
}

func TestParseRepairItemListAndAck(t *testing.T) {
	listData := make([]byte, 4+13*2)
	binary.LittleEndian.PutUint16(listData[0:2], PacketZCRepairItemList)
	binary.LittleEndian.PutUint16(listData[2:4], uint16(len(listData)))
	writeTestEquipmentChoice(listData[4:17], 7, 1201, 5, [4]uint16{4001, 4002, 0, 0})
	writeTestEquipmentChoice(listData[17:30], 9, 2301, 2, [4]uint16{0, 0, 0, 0})

	list, ok, err := ParseRepairItemList(Packet{ID: PacketZCRepairItemList, Data: listData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(list.Items) != 2 {
		t.Fatalf("repair item list ok=%v value=%+v", ok, list)
	}
	if list.Items[0] != (RepairItem{Index: 7, ItemID: 1201, Refine: 5, Cards: [4]uint16{4001, 4002, 0, 0}}) {
		t.Fatalf("first repair item = %+v", list.Items[0])
	}
	if list.Items[1] != (RepairItem{Index: 9, ItemID: 2301, Refine: 2}) {
		t.Fatalf("second repair item = %+v", list.Items[1])
	}

	ackData := make([]byte, 5)
	binary.LittleEndian.PutUint16(ackData[0:2], PacketZCAckItemRepair)
	binary.LittleEndian.PutUint16(ackData[2:4], 7)
	ackData[4] = 0
	ack, ok, err := ParseRepairItemAck(Packet{ID: PacketZCAckItemRepair, Data: ackData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || ack.Index != 7 || !ack.Success() {
		t.Fatalf("repair ack ok=%v value=%+v", ok, ack)
	}
}

func TestParseWeaponRefineListAndAck(t *testing.T) {
	listData := make([]byte, 4+13)
	binary.LittleEndian.PutUint16(listData[0:2], PacketZCWeaponRefineList)
	binary.LittleEndian.PutUint16(listData[2:4], uint16(len(listData)))
	writeTestEquipmentChoice(listData[4:17], 11, 1101, 4, [4]uint16{4001, 0, 0, 0})

	list, ok, err := ParseWeaponRefineList(Packet{ID: PacketZCWeaponRefineList, Data: listData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(list.Items) != 1 {
		t.Fatalf("weapon refine list ok=%v value=%+v", ok, list)
	}
	if list.Items[0] != (WeaponRefineItem{Index: 11, ItemID: 1101, Refine: 4, Cards: [4]uint16{4001, 0, 0, 0}}) {
		t.Fatalf("weapon refine item = %+v", list.Items[0])
	}

	ackData := make([]byte, 8)
	binary.LittleEndian.PutUint16(ackData[0:2], PacketZCAckWeaponRefine)
	binary.LittleEndian.PutUint32(ackData[2:6], 1)
	binary.LittleEndian.PutUint16(ackData[6:8], 1101)
	ack, ok, err := ParseWeaponRefineAck(Packet{ID: PacketZCAckWeaponRefine, Data: ackData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || ack.ItemID != 1101 || ack.Result != 1 || ack.Success() {
		t.Fatalf("weapon refine ack ok=%v value=%+v", ok, ack)
	}
}

func TestBuildItemCompositionPackets(t *testing.T) {
	list := BuildItemCompositionListPacket(7)
	if got := binary.LittleEndian.Uint16(list[0:2]); got != 0x017A {
		t.Fatalf("list opcode = 0x%04X", got)
	}
	if got := binary.LittleEndian.Uint16(list[2:4]); got != 7 {
		t.Fatalf("list card index = %d", got)
	}
	compose := BuildItemCompositionPacket(7, 11)
	if got := binary.LittleEndian.Uint16(compose[0:2]); got != 0x017C {
		t.Fatalf("compose opcode = 0x%04X", got)
	}
	if got := binary.LittleEndian.Uint16(compose[2:4]); got != 7 {
		t.Fatalf("compose card index = %d", got)
	}
	if got := binary.LittleEndian.Uint16(compose[4:6]); got != 11 {
		t.Fatalf("compose equip index = %d", got)
	}
}

func TestBuildMakingArrowPacket(t *testing.T) {
	packet := BuildMakingArrowPacket(909)
	if len(packet) != 4 || ID(packet) != PacketCZReqMakingArrow {
		t.Fatalf("unexpected making arrow packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 909 {
		t.Fatalf("making arrow item id = %d, want 909", got)
	}
}

func TestBuildMakingItemPacket(t *testing.T) {
	packet := BuildMakingItemPacket(501, [3]uint16{1000, 990, 5})
	if len(packet) != 10 || ID(packet) != PacketCZReqMakingItem {
		t.Fatalf("unexpected making item packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 501 {
		t.Fatalf("making item id = %d, want 501", got)
	}
	if got := [3]uint16{
		binary.LittleEndian.Uint16(packet[4:6]),
		binary.LittleEndian.Uint16(packet[6:8]),
		binary.LittleEndian.Uint16(packet[8:10]),
	}; got != [3]uint16{1000, 990, 5} {
		t.Fatalf("making item material = %v", got)
	}
}

func TestBuildRepairItemPacket(t *testing.T) {
	packet := BuildRepairItemPacket(RepairItem{Index: 7, ItemID: 1201, Refine: 5, Cards: [4]uint16{4001, 4002, 0, 0}})
	if len(packet) != 15 || ID(packet) != PacketCZReqItemRepair {
		t.Fatalf("unexpected repair packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 7 {
		t.Fatalf("repair index = %d, want 7", got)
	}
	if got := binary.LittleEndian.Uint16(packet[4:6]); got != 1201 {
		t.Fatalf("repair item = %d, want 1201", got)
	}
	if packet[6] != 5 {
		t.Fatalf("repair refine = %d, want 5", packet[6])
	}
	if got := [4]uint16{
		binary.LittleEndian.Uint16(packet[7:9]),
		binary.LittleEndian.Uint16(packet[9:11]),
		binary.LittleEndian.Uint16(packet[11:13]),
		binary.LittleEndian.Uint16(packet[13:15]),
	}; got != [4]uint16{4001, 4002, 0, 0} {
		t.Fatalf("repair cards = %v", got)
	}
}

func TestBuildWeaponRefinePacket(t *testing.T) {
	packet := BuildWeaponRefinePacket(11)
	if len(packet) != 6 || ID(packet) != PacketCZReqWeaponRefine {
		t.Fatalf("unexpected weapon refine packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint32(packet[2:6]); got != 11 {
		t.Fatalf("weapon refine index = %d, want 11", got)
	}

	cancel := BuildWeaponRefinePacket(0xFFFFFFFF)
	if got := binary.LittleEndian.Uint32(cancel[2:6]); got != 0xFFFFFFFF {
		t.Fatalf("weapon refine cancel index = 0x%08X, want 0xFFFFFFFF", got)
	}
}

func TestBuildItemIdentifyPacket(t *testing.T) {
	packet := BuildItemIdentifyPacket(9)
	if len(packet) != 4 || ID(packet) != PacketCZReqItemIdentify {
		t.Fatalf("unexpected identify packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 9 {
		t.Fatalf("identify index = %d, want 9", got)
	}
}

func writeTestEquipmentChoice(dst []byte, index, itemID uint16, refine uint8, cards [4]uint16) {
	binary.LittleEndian.PutUint16(dst[0:2], index)
	binary.LittleEndian.PutUint16(dst[2:4], itemID)
	dst[4] = refine
	binary.LittleEndian.PutUint16(dst[5:7], cards[0])
	binary.LittleEndian.PutUint16(dst[7:9], cards[1])
	binary.LittleEndian.PutUint16(dst[9:11], cards[2])
	binary.LittleEndian.PutUint16(dst[11:13], cards[3])
}

func TestParseEquippedArrow(t *testing.T) {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:2], 0x013C)
	binary.LittleEndian.PutUint16(data[2:4], 9)

	arrow, ok, err := ParseEquippedArrow(Packet{ID: 0x013C, Data: data})
	if err != nil || !ok {
		t.Fatalf("ParseEquippedArrow ok=%v err=%v", ok, err)
	}
	if arrow.Index != 9 {
		t.Fatalf("arrow index = %d, want 9", arrow.Index)
	}
}

func TestParseInventoryItemListNormalLegacy(t *testing.T) {
	data := make([]byte, 4+10)
	binary.LittleEndian.PutUint16(data[0:2], 0x00A3)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], 7)
	binary.LittleEndian.PutUint16(data[6:8], 938)
	data[8] = 3
	data[9] = 1
	binary.LittleEndian.PutUint16(data[10:12], 4)
	binary.LittleEndian.PutUint16(data[12:14], 0x0001)

	items, ok, err := ParseInventoryItemList(Packet{ID: 0x00A3, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 {
		t.Fatalf("items ok=%v len=%d", ok, len(items))
	}
	if got := items[0]; got.Index != 7 || got.ItemID != 938 || got.Type != 3 || got.Location != 0x0001 || !got.Identified || got.Amount != 4 || got.Equip {
		t.Fatalf("unexpected item: %+v", got)
	}
}

func TestParseInventoryItemListNormal2008(t *testing.T) {
	data := make([]byte, 4+22)
	binary.LittleEndian.PutUint16(data[0:2], 0x02E8)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], 7)
	binary.LittleEndian.PutUint16(data[6:8], 938)
	data[8] = 3
	data[9] = 1
	binary.LittleEndian.PutUint16(data[10:12], 4)
	binary.LittleEndian.PutUint16(data[12:14], 0x0001)
	binary.LittleEndian.PutUint16(data[14:16], 4001)
	binary.LittleEndian.PutUint32(data[22:26], 123456)

	items, ok, err := ParseInventoryItemList(Packet{ID: 0x02E8, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 {
		t.Fatalf("items ok=%v len=%d", ok, len(items))
	}
	if got := items[0]; got.Index != 7 || got.ItemID != 938 || got.Type != 3 || got.Location != 0x0001 || !got.Identified || got.Amount != 4 || got.Equip {
		t.Fatalf("unexpected item: %+v", got)
	}
}

func TestParseInventoryItemListNormal4(t *testing.T) {
	data := make([]byte, 4+24)
	binary.LittleEndian.PutUint16(data[0:2], 0x0991)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], 7)
	binary.LittleEndian.PutUint16(data[6:8], 938)
	data[8] = 3
	binary.LittleEndian.PutUint16(data[9:11], 4)
	binary.LittleEndian.PutUint32(data[11:15], 0x00000001)
	binary.LittleEndian.PutUint32(data[23:27], 123456)
	data[27] = 1

	items, ok, err := ParseInventoryItemList(Packet{ID: 0x0991, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 {
		t.Fatalf("items ok=%v len=%d", ok, len(items))
	}
	if got := items[0]; got.Index != 7 || got.ItemID != 938 || got.Type != 3 || got.Location != 0x0001 || !got.Identified || got.Amount != 4 || got.Equip {
		t.Fatalf("unexpected item: %+v", got)
	}
}

func TestParseInventoryItemListEquipmentLegacy(t *testing.T) {
	data := make([]byte, 4+20)
	binary.LittleEndian.PutUint16(data[0:2], 0x00A4)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], 11)
	binary.LittleEndian.PutUint16(data[6:8], 1201)
	data[8] = 4
	data[9] = 1
	binary.LittleEndian.PutUint16(data[10:12], 0x0002)
	binary.LittleEndian.PutUint16(data[12:14], 0x0002)
	data[14] = 1
	data[15] = 5

	items, ok, err := ParseInventoryItemList(Packet{ID: 0x00A4, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 {
		t.Fatalf("items ok=%v len=%d", ok, len(items))
	}
	if got := items[0]; got.Index != 11 || got.ItemID != 1201 || got.Type != 4 || got.Location != 0x0002 || !got.Identified || got.Amount != 1 || !got.Equip || !got.Equipped || !got.Damaged || got.Refine != 5 {
		t.Fatalf("unexpected item: %+v", got)
	}
}

func TestParseInventoryItemListEquipment4(t *testing.T) {
	data := make([]byte, 4+31)
	binary.LittleEndian.PutUint16(data[0:2], 0x0992)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], 11)
	binary.LittleEndian.PutUint16(data[6:8], 1201)
	data[8] = 4
	binary.LittleEndian.PutUint32(data[9:13], 0x00000002)
	binary.LittleEndian.PutUint32(data[13:17], 0x00000002)
	data[17] = 5
	data[34] = 3

	items, ok, err := ParseInventoryItemList(Packet{ID: 0x0992, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 {
		t.Fatalf("items ok=%v len=%d", ok, len(items))
	}
	if got := items[0]; got.Index != 11 || got.ItemID != 1201 || got.Type != 4 || got.Location != 0x0002 || !got.Identified || got.Amount != 1 || !got.Equip || !got.Equipped || !got.Damaged || got.Refine != 5 {
		t.Fatalf("unexpected item: %+v", got)
	}
}

func TestParseInventoryItemListEquipment5(t *testing.T) {
	data := make([]byte, 4+57)
	binary.LittleEndian.PutUint16(data[0:2], 0x0A0D)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], 12)
	binary.LittleEndian.PutUint16(data[6:8], 1101)
	data[8] = 5
	binary.LittleEndian.PutUint32(data[9:13], 0x00000002)
	data[17] = 7
	data[60] = 1

	items, ok, err := ParseInventoryItemList(Packet{ID: 0x0A0D, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 {
		t.Fatalf("items ok=%v len=%d", ok, len(items))
	}
	if got := items[0]; got.Index != 12 || got.ItemID != 1101 || got.Type != 5 || got.Location != 0x0002 || !got.Identified || got.Amount != 1 || !got.Equip || got.Equipped || got.Damaged || got.Refine != 7 {
		t.Fatalf("unexpected item: %+v", got)
	}
}

func TestEquipmentListsKeepAllowedAndOccupiedSlots(t *testing.T) {
	for _, tc := range []struct {
		id   uint16
		size int
		wide bool
	}{
		{0x00A4, 20, false}, {0x0295, 24, false}, {0x02D0, 26, false},
		{0x0992, 31, true}, {0x0A0D, 57, true},
	} {
		for _, worn := range []uint16{0, 0x0008, 0x0080} {
			data := make([]byte, 4+tc.size)
			binary.LittleEndian.PutUint16(data[0:2], tc.id)
			binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
			binary.LittleEndian.PutUint16(data[4:6], 11)
			binary.LittleEndian.PutUint16(data[6:8], 2601)
			data[8] = 4 // Armor.
			if tc.wide {
				binary.LittleEndian.PutUint32(data[9:13], 0x0088)
				binary.LittleEndian.PutUint32(data[13:17], uint32(worn))
				data[len(data)-1] = 1
			} else {
				data[9] = 1
				binary.LittleEndian.PutUint16(data[10:12], 0x0088)
				binary.LittleEndian.PutUint16(data[12:14], worn)
			}
			items, ok, err := ParseInventoryItemList(Packet{ID: tc.id, Data: data})
			if err != nil || !ok || len(items) != 1 {
				t.Fatalf("packet 0x%04X: items=%+v ok=%v err=%v", tc.id, items, ok, err)
			}
			if item := items[0]; item.Location != 0x0088 || item.WearLocation != worn || item.Equipped != (worn != 0) {
				t.Errorf("packet 0x%04X wear 0x%04X: %+v", tc.id, worn, item)
			}
		}
	}
}

func TestParseInventoryItemListEquipment2008(t *testing.T) {
	data := make([]byte, 4+26)
	binary.LittleEndian.PutUint16(data[0:2], 0x02D0)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], 11)
	binary.LittleEndian.PutUint16(data[6:8], 1201)
	data[8] = 4
	data[9] = 1
	binary.LittleEndian.PutUint16(data[10:12], 0x0002)
	binary.LittleEndian.PutUint16(data[12:14], 0x0002)
	data[14] = 1
	data[15] = 5
	binary.LittleEndian.PutUint16(data[16:18], 4001)
	binary.LittleEndian.PutUint16(data[18:20], 4002)
	binary.LittleEndian.PutUint32(data[24:28], 123456)
	binary.LittleEndian.PutUint16(data[28:30], 1)

	items, ok, err := ParseInventoryItemList(Packet{ID: 0x02D0, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 {
		t.Fatalf("items ok=%v len=%d", ok, len(items))
	}
	if got := items[0]; got.Index != 11 || got.ItemID != 1201 || got.Type != 4 || got.Location != 0x0002 || !got.Identified || got.Amount != 1 || !got.Equip || !got.Equipped || !got.Damaged || got.Refine != 5 || got.Cards[0] != 4001 || got.Cards[1] != 4002 {
		t.Fatalf("unexpected item: %+v", got)
	}
}

func TestParseShopDealAndSellList(t *testing.T) {
	dealData := make([]byte, 6)
	binary.LittleEndian.PutUint16(dealData[0:2], 0x00C4)
	binary.LittleEndian.PutUint32(dealData[2:6], 0x11223344)
	deal, ok, err := ParseShopDealSelection(Packet{ID: 0x00C4, Data: dealData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || deal.NPCID != 0x11223344 {
		t.Fatalf("unexpected deal: ok=%v value=%+v", ok, deal)
	}

	sellData := make([]byte, 4+10)
	binary.LittleEndian.PutUint16(sellData[0:2], 0x00C7)
	binary.LittleEndian.PutUint16(sellData[2:4], uint16(len(sellData)))
	binary.LittleEndian.PutUint16(sellData[4:6], 12)
	binary.LittleEndian.PutUint32(sellData[6:10], 10)
	binary.LittleEndian.PutUint32(sellData[10:14], 12)
	items, ok, err := ParseShopSellList(Packet{ID: 0x00C7, Data: sellData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 {
		t.Fatalf("sell items ok=%v len=%d", ok, len(items))
	}
	if got := items[0]; got.Index != 12 || got.Price != 10 || got.OverchargePrice != 12 {
		t.Fatalf("unexpected sell item: %+v", got)
	}
}

func TestBuildShopPackets(t *testing.T) {
	deal := BuildShopDealSelectionPacket(0x11223344, 1)
	if len(deal) != 7 || ID(deal) != 0x00C5 || binary.LittleEndian.Uint32(deal[2:6]) != 0x11223344 || deal[6] != 1 {
		t.Fatalf("unexpected deal packet: % X", deal)
	}

	buy := BuildBuyItemListPacket([]BuyRequestItem{{ItemID: 501, Amount: 2}, {ItemID: 502, Amount: 3}})
	if len(buy) != 12 || ID(buy) != 0x00C8 || binary.LittleEndian.Uint16(buy[2:4]) != 12 {
		t.Fatalf("unexpected buy packet header: % X", buy)
	}
	if binary.LittleEndian.Uint16(buy[4:6]) != 2 || binary.LittleEndian.Uint16(buy[6:8]) != 501 || binary.LittleEndian.Uint16(buy[8:10]) != 3 || binary.LittleEndian.Uint16(buy[10:12]) != 502 {
		t.Fatalf("unexpected buy packet items: % X", buy)
	}

	sell := BuildSellItemListPacket([]SellRequestItem{{Index: 2, Amount: 3}, {Index: 4, Amount: 5}})
	if len(sell) != 12 || ID(sell) != 0x00C9 || binary.LittleEndian.Uint16(sell[2:4]) != 12 {
		t.Fatalf("unexpected sell packet header: % X", sell)
	}
	if binary.LittleEndian.Uint16(sell[4:6]) != 2 || binary.LittleEndian.Uint16(sell[6:8]) != 3 || binary.LittleEndian.Uint16(sell[8:10]) != 4 || binary.LittleEndian.Uint16(sell[10:12]) != 5 {
		t.Fatalf("unexpected sell packet items: % X", sell)
	}
}

func TestParseStoragePackets2008(t *testing.T) {
	normal := make([]byte, 4+22)
	binary.LittleEndian.PutUint16(normal[0:2], 0x02EA)
	binary.LittleEndian.PutUint16(normal[2:4], uint16(len(normal)))
	binary.LittleEndian.PutUint16(normal[4:6], 3)
	binary.LittleEndian.PutUint16(normal[6:8], 512)
	normal[8] = 0
	normal[9] = 1
	binary.LittleEndian.PutUint16(normal[10:12], 7)
	items, ok, err := ParseStorageItemList(Packet{ID: 0x02EA, Data: normal})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 || items[0].Index != 3 || items[0].ItemID != 512 || items[0].Amount != 7 || !items[0].Identified {
		t.Fatalf("normal storage items ok=%v items=%+v", ok, items)
	}

	equip := make([]byte, 4+26)
	binary.LittleEndian.PutUint16(equip[0:2], 0x02D1)
	binary.LittleEndian.PutUint16(equip[2:4], uint16(len(equip)))
	binary.LittleEndian.PutUint16(equip[4:6], 4)
	binary.LittleEndian.PutUint16(equip[6:8], 1201)
	equip[8] = 5
	equip[9] = 1
	binary.LittleEndian.PutUint16(equip[10:12], 0x0002)
	equip[15] = 4
	items, ok, err = ParseStorageItemList(Packet{ID: 0x02D1, Data: equip})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 || items[0].Index != 4 || items[0].ItemID != 1201 || !items[0].Equip || items[0].Refine != 4 {
		t.Fatalf("equip storage items ok=%v items=%+v", ok, items)
	}

	amountData := []byte{0xF2, 0x00, 0x02, 0x00, 0x58, 0x01}
	amount, ok, err := ParseStorageAmount(Packet{ID: 0x00F2, Data: amountData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || amount.Amount != 2 || amount.MaxAmount != 344 {
		t.Fatalf("storage amount ok=%v amount=%+v", ok, amount)
	}
}

func TestParseStorageItemDeltaPackets(t *testing.T) {
	add := make([]byte, 22)
	binary.LittleEndian.PutUint16(add[0:2], 0x01C4)
	binary.LittleEndian.PutUint16(add[2:4], 5)
	binary.LittleEndian.PutUint32(add[4:8], 12)
	binary.LittleEndian.PutUint16(add[8:10], 938)
	add[10] = 3
	add[11] = 1
	item, ok, err := ParseStorageItemAdded(Packet{ID: 0x01C4, Data: add})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || item.Index != 5 || item.ItemID != 938 || item.Type != 3 || item.Amount != 12 || !item.Identified {
		t.Fatalf("storage add ok=%v item=%+v", ok, item)
	}

	remove := make([]byte, 8)
	binary.LittleEndian.PutUint16(remove[0:2], 0x00F6)
	binary.LittleEndian.PutUint16(remove[2:4], 5)
	binary.LittleEndian.PutUint32(remove[4:8], 7)
	removed, ok, err := ParseStorageItemRemoved(Packet{ID: 0x00F6, Data: remove})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || removed.Index != 5 || removed.Amount != 7 {
		t.Fatalf("storage remove ok=%v value=%+v", ok, removed)
	}
	if !ParseStorageClosed(Packet{ID: 0x00F8, Data: []byte{0xF8, 0x00}}) {
		t.Fatal("storage close packet was not recognized")
	}
}

func TestBuildStoragePacketsFor2008ClientDate(t *testing.T) {
	toStorage := BuildMoveToStoragePacketForClientDate(7, 42, 20080910)
	if len(toStorage) != 14 || ID(toStorage) != 0x0094 || binary.LittleEndian.Uint16(toStorage[7:9]) != 7 || binary.LittleEndian.Uint32(toStorage[10:14]) != 42 {
		t.Fatalf("unexpected move-to-storage packet: % X", toStorage)
	}

	fromStorage := BuildMoveFromStoragePacketForClientDate(3, 9, 20080910)
	if len(fromStorage) != 22 || ID(fromStorage) != 0x00F7 || binary.LittleEndian.Uint16(fromStorage[14:16]) != 3 || binary.LittleEndian.Uint32(fromStorage[18:22]) != 9 {
		t.Fatalf("unexpected move-from-storage packet: % X", fromStorage)
	}

	closeStorage := BuildCloseStoragePacketForClientDate(20080910)
	if len(closeStorage) != 2 || ID(closeStorage) != 0x0193 {
		t.Fatalf("unexpected close-storage packet: % X", closeStorage)
	}
}

func TestParseCartPackets2008(t *testing.T) {
	normal := make([]byte, 4+22)
	binary.LittleEndian.PutUint16(normal[0:2], 0x02E9)
	binary.LittleEndian.PutUint16(normal[2:4], uint16(len(normal)))
	binary.LittleEndian.PutUint16(normal[4:6], 3)
	binary.LittleEndian.PutUint16(normal[6:8], 512)
	normal[8] = 0
	normal[9] = 1
	binary.LittleEndian.PutUint16(normal[10:12], 7)
	items, ok, err := ParseCartItemList(Packet{ID: 0x02E9, Data: normal})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(items) != 1 || items[0].Index != 3 || items[0].ItemID != 512 || items[0].Amount != 7 || !items[0].Identified {
		t.Fatalf("normal cart items ok=%v items=%+v", ok, items)
	}

	equip := make([]byte, 4+8*26)
	binary.LittleEndian.PutUint16(equip[0:2], 0x02D2)
	binary.LittleEndian.PutUint16(equip[2:4], uint16(len(equip)))
	for i := 0; i < 8; i++ {
		offset := 4 + i*26
		binary.LittleEndian.PutUint16(equip[offset:offset+2], uint16(10+i))
		binary.LittleEndian.PutUint16(equip[offset+2:offset+4], uint16(2301+i))
		equip[offset+4] = 5
		equip[offset+5] = 1
		binary.LittleEndian.PutUint16(equip[offset+6:offset+8], 0x0010)
		equip[offset+11] = uint8(i)
	}
	equipItems, ok, err := ParseCartItemList(Packet{ID: 0x02D2, Data: equip})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(equipItems) != 8 || equipItems[0].Index != 10 || equipItems[0].ItemID != 2301 || !equipItems[0].Equip || !equipItems[0].Identified || equipItems[7].Refine != 7 {
		t.Fatalf("equip cart items ok=%v items=%+v", ok, equipItems)
	}

	amountData := make([]byte, 14)
	binary.LittleEndian.PutUint16(amountData[0:2], 0x0121)
	binary.LittleEndian.PutUint16(amountData[2:4], 2)
	binary.LittleEndian.PutUint16(amountData[4:6], 100)
	binary.LittleEndian.PutUint32(amountData[6:10], 450)
	binary.LittleEndian.PutUint32(amountData[10:14], 80000)
	amount, ok, err := ParseCartAmount(Packet{ID: 0x0121, Data: amountData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || amount.Amount != 2 || amount.MaxAmount != 100 || amount.Weight != 450 || amount.MaxWeight != 80000 {
		t.Fatalf("cart amount ok=%v amount=%+v", ok, amount)
	}
}

func TestParseCartItemDeltaPackets(t *testing.T) {
	add := make([]byte, 22)
	binary.LittleEndian.PutUint16(add[0:2], 0x01C5)
	binary.LittleEndian.PutUint16(add[2:4], 5)
	binary.LittleEndian.PutUint32(add[4:8], 12)
	binary.LittleEndian.PutUint16(add[8:10], 938)
	add[10] = 3
	add[11] = 1
	item, ok, err := ParseCartItemAdded(Packet{ID: 0x01C5, Data: add})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || item.Index != 5 || item.ItemID != 938 || item.Type != 3 || item.Amount != 12 || !item.Identified {
		t.Fatalf("cart add ok=%v item=%+v", ok, item)
	}

	remove := make([]byte, 8)
	binary.LittleEndian.PutUint16(remove[0:2], 0x0125)
	binary.LittleEndian.PutUint16(remove[2:4], 5)
	binary.LittleEndian.PutUint32(remove[4:8], 7)
	removed, ok, err := ParseCartItemRemoved(Packet{ID: 0x0125, Data: remove})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || removed.Index != 5 || removed.Amount != 7 {
		t.Fatalf("cart remove ok=%v value=%+v", ok, removed)
	}
}

func TestBuildCartMovePackets(t *testing.T) {
	toCart := BuildMoveToCartPacket(7, 42)
	if len(toCart) != 8 || ID(toCart) != 0x0126 || binary.LittleEndian.Uint16(toCart[2:4]) != 7 || binary.LittleEndian.Uint32(toCart[4:8]) != 42 {
		t.Fatalf("unexpected move-to-cart packet: % X", toCart)
	}

	fromCart := BuildMoveFromCartPacket(3, 9)
	if len(fromCart) != 8 || ID(fromCart) != 0x0127 || binary.LittleEndian.Uint16(fromCart[2:4]) != 3 || binary.LittleEndian.Uint32(fromCart[4:8]) != 9 {
		t.Fatalf("unexpected move-from-cart packet: % X", fromCart)
	}

	storageToCart := BuildMoveStorageToCartPacket(5, 12)
	if len(storageToCart) != 8 || ID(storageToCart) != 0x0128 || binary.LittleEndian.Uint16(storageToCart[2:4]) != 5 || binary.LittleEndian.Uint32(storageToCart[4:8]) != 12 {
		t.Fatalf("unexpected move-storage-to-cart packet: % X", storageToCart)
	}

	cartToStorage := BuildMoveCartToStoragePacket(6, 18)
	if len(cartToStorage) != 8 || ID(cartToStorage) != 0x0129 || binary.LittleEndian.Uint16(cartToStorage[2:4]) != 6 || binary.LittleEndian.Uint32(cartToStorage[4:8]) != 18 {
		t.Fatalf("unexpected move-cart-to-storage packet: % X", cartToStorage)
	}
}

func TestParseCartAddAck(t *testing.T) {
	data := []byte{0x2C, 0x01, 0x01}
	ack, ok, err := ParseCartAddAck(Packet{ID: 0x012C, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || ack.Result != 1 {
		t.Fatalf("cart ack ok=%v value=%+v", ok, ack)
	}
}

func TestParseCartClosed(t *testing.T) {
	if !ParseCartClosed(Packet{ID: 0x012B, Data: []byte{0x2B, 0x01}}) {
		t.Fatal("cart close not parsed")
	}
	if ParseCartClosed(Packet{ID: 0x00F8, Data: []byte{0xF8, 0x00}}) {
		t.Fatal("storage close parsed as cart close")
	}
}
