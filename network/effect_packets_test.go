package network

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestParseNPCShowEffectUpdate(t *testing.T) {
	data := make([]byte, 18)
	binary.LittleEndian.PutUint16(data, PacketZCNPCShowEFSTUpdate)
	binary.LittleEndian.PutUint32(data[2:], 0x00009500)
	binary.LittleEndian.PutUint32(data[6:], 0x11223344)
	binary.LittleEndian.PutUint32(data[10:], 75)
	binary.LittleEndian.PutUint32(data[14:], 0x80000006)
	packet := Packet{ID: PacketZCNPCShowEFSTUpdate, Data: data}
	want := NPCShowEffectUpdate{ActorID: 0x00009500, Level: 75, Opt3State: 0x80000006}
	if got, ok, err := ParseNPCShowEffectUpdate(packet); err != nil || !ok || got != want {
		t.Fatalf("update=%+v ok=%t err=%v", got, ok, err)
	}
	for n := 0; n < len(data); n++ {
		if _, ok, err := ParseNPCShowEffectUpdate(Packet{ID: packet.ID, Data: data[:n]}); err == nil || ok {
			t.Fatalf("accepted truncated update of length %d", n)
		}
	}
	if _, ok, err := ParseNPCShowEffectUpdate(Packet{ID: PacketZCSpirits}); ok || err != nil {
		t.Fatal("consumed unrelated packet")
	}
	// A valid opcode in the actor ID must not cause stream resynchronization.
	stream := append(append([]byte(nil), data...), 0x7f, 0, 1, 2, 3, 4)
	for _, chunk := range []int{1, 7, len(stream)} {
		framer := NewFramer(PacketLengths2008())
		var got []Packet
		for offset := 0; offset < len(stream); offset += chunk {
			packets, err := framer.Push(stream[offset:min(offset+chunk, len(stream))])
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, packets...)
		}
		if !reflect.DeepEqual(got, []Packet{packet, {ID: 0x007f, Data: stream[18:]}}) {
			t.Fatalf("chunk %d: corrupted stream: %+v", chunk, got)
		}
	}
}

func TestParseSpiritSphereCount(t *testing.T) {
	for _, id := range []uint16{PacketZCSpirits, PacketZCSpirits2} {
		data := make([]byte, 8)
		binary.LittleEndian.PutUint16(data, id)
		binary.LittleEndian.PutUint32(data[2:], 0x12345678)
		binary.LittleEndian.PutUint16(data[6:], 3)
		update, ok, err := ParseSpiritSphereCount(Packet{ID: id, Data: data})
		if err != nil || !ok || update.ActorID != 0x12345678 || update.Count != 3 {
			t.Fatalf("packet 0x%04X: %+v, ok=%t, err=%v", id, update, ok, err)
		}
		if PacketLengths2008()[id] != len(data) {
			t.Fatalf("packet 0x%04X missing from receive framer", id)
		}
		binary.LittleEndian.PutUint16(data[6:], 0)
		update, ok, err = ParseSpiritSphereCount(Packet{ID: id, Data: data})
		if err != nil || !ok || update.Count != 0 {
			t.Fatal("cleared sphere count was not parsed")
		}
		for n := 0; n < len(data); n++ {
			if _, _, err := ParseSpiritSphereCount(Packet{ID: id, Data: data[:n]}); err == nil {
				t.Fatalf("packet 0x%04X accepted %d bytes", id, n)
			}
		}
	}
	if _, ok, err := ParseSpiritSphereCount(Packet{ID: PacketZCMVP}); ok || err != nil {
		t.Fatal("unrelated packet accepted")
	}
}

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
