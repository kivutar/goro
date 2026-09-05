package network

import (
	"encoding/binary"
	"testing"
)

func TestParseTaekwonMission(t *testing.T) {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCStarSkill)
	copy(data[2:26], "Spore")
	binary.LittleEndian.PutUint32(data[26:30], 1014)
	data[30] = 37
	data[31] = 1

	mission, ok, err := ParseTaekwonMission(Packet{ID: PacketZCStarSkill, Data: data})
	if err != nil || !ok {
		t.Fatalf("parse mission ok=%t err=%v", ok, err)
	}
	if mission.MonsterName != "Spore" || mission.MonsterID != 1014 || mission.Progress != 37 || mission.Result != 1 {
		t.Fatalf("mission = %+v", mission)
	}
}

func TestParseStarPlace(t *testing.T) {
	place, ok, err := ParseStarPlace(Packet{ID: PacketZCStarPlace, Data: []byte{0x53, 0x02, 2}})
	if err != nil || !ok || place.Place != 2 {
		t.Fatalf("place = %+v ok=%t err=%v", place, ok, err)
	}
	if _, ok, err := ParseStarPlace(Packet{ID: PacketZCStarPlace, Data: []byte{0x53, 0x02, 3}}); !ok || err == nil {
		t.Fatalf("invalid place ok=%t err=%v", ok, err)
	}
}

func TestBuildAgreeStarPlacePacket(t *testing.T) {
	packet := BuildAgreeStarPlacePacket(1)
	if len(packet) != 3 {
		t.Fatalf("packet len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[0:2]); got != PacketCZAgreeStarPlace {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if packet[2] != 1 {
		t.Fatalf("place = %d", packet[2])
	}
}

func TestTaekwonPacketLengthsAreFramedFor2008(t *testing.T) {
	lengths := PacketLengths2008()
	want := map[uint16]int{
		PacketZCStarSkill: 32, PacketZCStarPlace: 3,
	}
	for packetID, wantLen := range want {
		if got := lengths[packetID]; got != wantLen {
			t.Fatalf("packet 0x%04X length = %d, want %d", packetID, got, wantLen)
		}
	}
}

func TestTaekwonPacketParsersRejectShortPackets(t *testing.T) {
	if _, ok, err := ParseTaekwonMission(Packet{ID: PacketZCStarSkill, Data: make([]byte, 31)}); !ok || err == nil {
		t.Fatalf("short mission ok=%t err=%v", ok, err)
	}
	if _, ok, err := ParseStarPlace(Packet{ID: PacketZCStarPlace, Data: make([]byte, 2)}); !ok || err == nil {
		t.Fatalf("short star place ok=%t err=%v", ok, err)
	}
}
