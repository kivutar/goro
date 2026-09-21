package network

import (
	"encoding/binary"
	"testing"
)

func TestBuildEquipmentPackets(t *testing.T) {
	removeOption := BuildRemoveOptionPacket()
	if len(removeOption) != 2 || ID(removeOption) != PacketCZReqCartOff {
		t.Fatalf("remove option packet = % X", removeOption)
	}

	view := BuildViewPlayerEquipmentPacket(0x11223344)
	if len(view) != 6 || ID(view) != PacketCZEquipWinMicroscope || binary.LittleEndian.Uint32(view[2:6]) != 0x11223344 {
		t.Fatalf("view packet = % X", view)
	}

	show := BuildShowEquipConfigPacket(true)
	if len(show) != 10 || ID(show) != PacketCZConfig {
		t.Fatalf("show config packet = % X", show)
	}
	if got := binary.LittleEndian.Uint32(show[2:6]); got != ConfigOpenEquipmentWindow {
		t.Fatalf("config = %d, want %d", got, ConfigOpenEquipmentWindow)
	}
	if got := binary.LittleEndian.Uint32(show[6:10]); got != 1 {
		t.Fatalf("value = %d, want 1", got)
	}

	hide := BuildShowEquipConfigPacket(false)
	if got := binary.LittleEndian.Uint32(hide[6:10]); got != 0 {
		t.Fatalf("value = %d, want 0", got)
	}
}

func TestRemoveOptionIsOutgoingOnly(t *testing.T) {
	if _, ok := PacketLengths2008()[PacketCZReqCartOff]; ok {
		t.Fatal("0x012A is client-to-server and must not be in the receive framer")
	}
}

func TestParseShowEquipConfig(t *testing.T) {
	enabled, ok, err := ParseShowEquipConfig(Packet{ID: PacketZCConfigNotify, Data: []byte{0xDA, 0x02, 1}})
	if err != nil || !ok || !enabled {
		t.Fatalf("ParseShowEquipConfig enabled=%t ok=%t err=%v", enabled, ok, err)
	}
}

func TestParseViewedEquipment2008Layout(t *testing.T) {
	const packetLen = 43 + 26
	data := make([]byte, packetLen)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCEquipWinMicroscope)
	binary.LittleEndian.PutUint16(data[2:4], packetLen)
	copy(data[4:28], []byte("Alice"))
	binary.LittleEndian.PutUint16(data[28:30], 4)
	binary.LittleEndian.PutUint16(data[30:32], 2)
	binary.LittleEndian.PutUint16(data[32:34], 30)
	binary.LittleEndian.PutUint16(data[34:36], 20)
	binary.LittleEndian.PutUint16(data[36:38], 10)
	binary.LittleEndian.PutUint16(data[38:40], 3)
	binary.LittleEndian.PutUint16(data[40:42], 4)
	data[42] = 1

	offset := 43
	binary.LittleEndian.PutUint16(data[offset:offset+2], 7)
	binary.LittleEndian.PutUint16(data[offset+2:offset+4], 1201)
	data[offset+4] = 5
	data[offset+5] = 1
	// A dual-wieldable weapon can use either hand but occupies only one.
	binary.LittleEndian.PutUint16(data[offset+6:offset+8], 0x0022)
	binary.LittleEndian.PutUint16(data[offset+8:offset+10], 2)
	data[offset+10] = 0
	data[offset+11] = 3

	view, ok, err := ParseViewedEquipment(Packet{ID: PacketZCEquipWinMicroscope, Data: data})
	if err != nil || !ok {
		t.Fatalf("ParseViewedEquipment ok=%t err=%v", ok, err)
	}
	if view.Name != "Alice" || view.Job != 4 || view.Head != 2 || view.HeadTop != 10 || view.HeadMid != 20 || view.HeadLow != 30 || view.Sex != 1 {
		t.Fatalf("unexpected header: %+v", view)
	}
	if len(view.Items) != 1 {
		t.Fatalf("items len = %d", len(view.Items))
	}
	item := view.Items[0]
	if item.Index != 7 || item.ItemID != 1201 || item.Type != 5 || item.Location != 0x0022 || item.WearLocation != 2 || !item.Identified || !item.Equipped || item.Refine != 3 {
		t.Fatalf("unexpected item: %+v", item)
	}
}
