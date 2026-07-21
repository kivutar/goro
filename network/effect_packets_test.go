package network

import (
	"encoding/binary"
	"testing"
)

func TestParseSpecialEffectNotify(t *testing.T) {
	data := make([]byte, 10)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCNotifyEffect)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)
	binary.LittleEndian.PutUint32(data[6:10], SpecialEffectBaseLevelUp)

	notify, ok, err := ParseSpecialEffectNotify(Packet{ID: PacketZCNotifyEffect, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("special effect notify not parsed")
	}
	if notify.AID != 0x11223344 || notify.EffectID != SpecialEffectBaseLevelUp {
		t.Fatalf("notify = %+v", notify)
	}
}

func TestParseSpecialEffectNotify2(t *testing.T) {
	data := make([]byte, 10)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCNotifyEffect2)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)
	binary.LittleEndian.PutUint32(data[6:10], 568)

	notify, ok, err := ParseSpecialEffectNotify(Packet{ID: PacketZCNotifyEffect2, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("special effect2 notify not parsed")
	}
	if notify.AID != 0x11223344 || notify.EffectID != 568 {
		t.Fatalf("notify = %+v", notify)
	}
}

func TestParseMVPNotify(t *testing.T) {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCMVP)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)

	notify, ok, err := ParseMVPNotify(Packet{ID: PacketZCMVP, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("mvp notify not parsed")
	}
	if notify.AID != 0x11223344 {
		t.Fatalf("notify = %+v", notify)
	}
}

func TestBuildLessEffectPacket(t *testing.T) {
	packet := BuildLessEffectPacket(true)
	if len(packet) != 6 || ID(packet) != PacketCZLessEffect {
		t.Fatalf("less effect packet = % X", packet)
	}
	if got := binary.LittleEndian.Uint32(packet[2:6]); got != 1 {
		t.Fatalf("less effect state = %d, want 1", got)
	}

	packet = BuildLessEffectPacket(false)
	if got := binary.LittleEndian.Uint32(packet[2:6]); got != 0 {
		t.Fatalf("less effect state = %d, want 0", got)
	}
}

func TestParseLessEffect(t *testing.T) {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCLessEffect)
	binary.LittleEndian.PutUint32(data[2:6], 1)

	enabled, ok, err := ParseLessEffect(Packet{ID: PacketZCLessEffect, Data: data})
	if err != nil || !ok || !enabled {
		t.Fatalf("ParseLessEffect enabled=%t ok=%t err=%v", enabled, ok, err)
	}
}

func TestLessEffectPacketDirections(t *testing.T) {
	if _, ok := PacketLengths2008()[PacketCZLessEffect]; ok {
		t.Fatal("0x021D is client-to-server and must not be in the receive framer")
	}
	if got := PacketLengths2008()[PacketZCLessEffect]; got != 6 {
		t.Fatalf("0x021E receive length = %d, want 6", got)
	}
	if got := PacketLengths2008()[PacketZCMVP]; got != 6 {
		t.Fatalf("0x010C receive length = %d, want 6", got)
	}
}

func TestParseSpecialEffectNotifyIgnoresOtherPackets(t *testing.T) {
	_, ok, err := ParseSpecialEffectNotify(Packet{ID: 0x019A, Data: make([]byte, 10)})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected special effect notify")
	}
}

func TestParseMVPNotifyIgnoresOtherPackets(t *testing.T) {
	_, ok, err := ParseMVPNotify(Packet{ID: 0x010B, Data: make([]byte, 6)})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected mvp notify")
	}
}
