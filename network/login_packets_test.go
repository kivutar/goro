package network

import (
	"encoding/binary"
	"testing"
)

func TestBuildAccountLoginPacket(t *testing.T) {
	packet := BuildAccountLoginPacket(AccountLogin{
		Version:    55,
		Username:   "alice",
		Password:   "secret",
		ClientType: 1,
	})

	if len(packet) != 55 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x64 || packet[1] != 0x00 {
		t.Fatalf("opcode = %02x %02x", packet[0], packet[1])
	}
	if packet[6] != 'a' || packet[30] != 's' {
		t.Fatalf("username/password offsets are wrong")
	}
	if packet[54] != 1 {
		t.Fatalf("client type = %d, want 1", packet[54])
	}
}

func TestBuildCharServerEnterPacket(t *testing.T) {
	packet := BuildCharServerEnterPacket(CharServerEnter{
		AccountID: 10,
		AuthCode:  20,
		UserLevel: 30,
		Sex:       1,
	})
	if len(packet) != 17 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x65 || packet[1] != 0x00 || packet[16] != 1 {
		t.Fatalf("unexpected packet bytes: % x", packet)
	}
}

func TestBuildLoginServerKeepalivePacket(t *testing.T) {
	packet := BuildLoginServerKeepalivePacket("Kivutar")
	if len(packet) != 26 {
		t.Fatalf("packet length = %d, want 26", len(packet))
	}
	if binary.LittleEndian.Uint16(packet[:2]) != PacketCAConnectInfoChanged {
		t.Fatalf("opcode = 0x%04X", binary.LittleEndian.Uint16(packet[:2]))
	}
	if got := string(packet[2:9]); got != "Kivutar" {
		t.Fatalf("username = %q, want Kivutar", got)
	}
}

func TestBuildSelectCharacterPacket(t *testing.T) {
	packet := BuildSelectCharacterPacket(2)
	if len(packet) != 3 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x66 || packet[1] != 0x00 || packet[2] != 2 {
		t.Fatalf("unexpected packet bytes: % x", packet)
	}
}

func TestBuildMakeCharacterPacket(t *testing.T) {
	packet := BuildMakeCharacterPacket(MakeCharacter{
		Name:      "Alice",
		Str:       5,
		Agi:       6,
		Vit:       7,
		Int:       5,
		Dex:       3,
		Luk:       4,
		Slot:      2,
		HairColor: 8,
		HairStyle: 9,
	})
	if len(packet) != 37 {
		t.Fatalf("len = %d", len(packet))
	}
	if binary.LittleEndian.Uint16(packet[0:2]) != PacketCHMakeChar {
		t.Fatalf("opcode = % x", packet[:2])
	}
	if string(packet[2:7]) != "Alice" {
		t.Fatalf("name bytes = %q", packet[2:26])
	}
	if packet[26] != 5 || packet[27] != 6 || packet[28] != 7 || packet[29] != 5 || packet[30] != 3 || packet[31] != 4 || packet[32] != 2 {
		t.Fatalf("stat/slot bytes = % x", packet[26:33])
	}
	if binary.LittleEndian.Uint16(packet[33:35]) != 8 || binary.LittleEndian.Uint16(packet[35:37]) != 9 {
		t.Fatalf("hair bytes = % x", packet[33:37])
	}
}

func TestBuildDeleteCharacterPacket(t *testing.T) {
	packet := BuildDeleteCharacterPacket(20080910, 0x11223344, "mail@example.com")
	if len(packet) != 56 {
		t.Fatalf("len = %d", len(packet))
	}
	if binary.LittleEndian.Uint16(packet[0:2]) != PacketCHDeleteChar {
		t.Fatalf("opcode = % x", packet[:2])
	}
	if binary.LittleEndian.Uint32(packet[2:6]) != 0x11223344 {
		t.Fatalf("char id = 0x%08x", binary.LittleEndian.Uint32(packet[2:6]))
	}
	if got := string(packet[6 : 6+16]); got != "mail@example.com" {
		t.Fatalf("key = %q", got)
	}
}

func TestBuildOldDeleteCharacterPacket(t *testing.T) {
	packet := BuildDeleteCharacterPacket(20030401, 0x11223344, "mail@example.com")
	if len(packet) != 46 {
		t.Fatalf("len = %d", len(packet))
	}
	if binary.LittleEndian.Uint16(packet[0:2]) != PacketCHDeleteCharOld {
		t.Fatalf("opcode = % x", packet[:2])
	}
	if binary.LittleEndian.Uint32(packet[2:6]) != 0x11223344 {
		t.Fatalf("char id = 0x%08x", binary.LittleEndian.Uint32(packet[2:6]))
	}
	if got := string(packet[6 : 6+16]); got != "mail@example.com" {
		t.Fatalf("key = %q", got)
	}
}

func TestBuildLoadEndAckPacket(t *testing.T) {
	packet := BuildLoadEndAckPacket()
	if len(packet) != 2 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x7D || packet[1] != 0x00 {
		t.Fatalf("unexpected packet bytes: % x", packet)
	}
}

func TestBuildRestartPacket(t *testing.T) {
	packet := BuildRestartPacket(RestartTypeRespawn)
	want := []byte{0xB2, 0x00, RestartTypeRespawn}
	if len(packet) != len(want) {
		t.Fatalf("len = %d", len(packet))
	}
	for i := range want {
		if packet[i] != want[i] {
			t.Fatalf("packet = % x, want % x", packet, want)
		}
	}

	packet = BuildRestartPacket(RestartTypeCharSelect)
	if len(packet) != 3 || packet[2] != RestartTypeCharSelect {
		t.Fatalf("unexpected char-select packet: % x", packet)
	}
}

func TestBuildQuitGamePacket(t *testing.T) {
	packet := BuildQuitGamePacket()
	want := []byte{0x8A, 0x01, 0x00, 0x00}
	if len(packet) != len(want) {
		t.Fatalf("len = %d", len(packet))
	}
	for i := range want {
		if packet[i] != want[i] {
			t.Fatalf("packet = % x, want % x", packet, want)
		}
	}
}

func TestBuildPingPacket(t *testing.T) {
	packet := BuildPingPacket(0x11223344)
	if len(packet) != 6 {
		t.Fatalf("len = %d", len(packet))
	}
	if binary.LittleEndian.Uint16(packet[0:2]) != PacketPing {
		t.Fatalf("opcode = % x", packet[:2])
	}
	if binary.LittleEndian.Uint32(packet[2:6]) != 0x11223344 {
		t.Fatalf("account id = 0x%08x", binary.LittleEndian.Uint32(packet[2:6]))
	}
}

func TestBuildMapServerEnterPacket(t *testing.T) {
	packet := BuildMapServerEnterPacketForClientDate(MapServerEnter{
		AccountID:  0x11223344,
		CharID:     0x55667788,
		AuthCode:   0x99aabbcc,
		ClientTick: 0xddeeff00,
		Sex:        1,
	}, 20080910)
	if len(packet) != 19 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x36 || packet[1] != 0x04 {
		t.Fatalf("unexpected opcode: % x", packet[:2])
	}
	if packet[18] != 1 {
		t.Fatalf("unexpected sex offset: % x", packet)
	}
}

func TestBuildMapServerEnterPacketFor2021ClientDate(t *testing.T) {
	packet := BuildMapServerEnterPacketForClientDate(MapServerEnter{
		AccountID:  0x11223344,
		CharID:     0x55667788,
		AuthCode:   0x99aabbcc,
		ClientTick: 0xddeeff00,
		Sex:        1,
	}, 20211103)
	if len(packet) != 23 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x36 || packet[1] != 0x04 {
		t.Fatalf("unexpected opcode: % x", packet[:2])
	}
	if packet[22] != 1 {
		t.Fatalf("unexpected sex offset: % x", packet)
	}
	for i := 18; i < 22; i++ {
		if packet[i] != 0 {
			t.Fatalf("expected zero padding at %d: % x", i, packet)
		}
	}
}

func TestBuildTickSendPacket(t *testing.T) {
	packet := BuildTickSendPacketForClientDate(0x11223344, 20080910)
	if len(packet) != 8 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x89 || packet[1] != 0x00 {
		t.Fatalf("unexpected opcode: % x", packet[:2])
	}
	if packet[4] != 0x44 || packet[5] != 0x33 || packet[6] != 0x22 || packet[7] != 0x11 {
		t.Fatalf("unexpected tick bytes: % x", packet[4:8])
	}
}

func TestBuildTickSendPacketFor2021ClientDate(t *testing.T) {
	packet := BuildTickSendPacketForClientDate(0x11223344, 20211103)
	if len(packet) != 6 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x60 || packet[1] != 0x03 {
		t.Fatalf("unexpected opcode: % x", packet[:2])
	}
	if packet[2] != 0x44 || packet[3] != 0x33 || packet[4] != 0x22 || packet[5] != 0x11 {
		t.Fatalf("unexpected tick bytes: % x", packet[2:6])
	}
}

func TestBuildNameRequestPacketForClientDate20080910(t *testing.T) {
	packet, ok := BuildNameRequestPacketForClientDate(0x11223344, 20080910)
	if !ok {
		t.Fatal("expected packet")
	}
	if len(packet) != 11 {
		t.Fatalf("len = %d, want 11", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[:2]); got != 0x008C {
		t.Fatalf("opcode = 0x%04X, want 0x008C", got)
	}
	if got := binary.LittleEndian.Uint32(packet[7:11]); got != 0x11223344 {
		t.Fatalf("gid = 0x%08X, want 0x11223344", got)
	}
}

func TestBuildNameRequestPacketForClientDate20101124(t *testing.T) {
	packet, ok := BuildNameRequestPacketForClientDate(0x11223344, 20101124)
	if !ok {
		t.Fatal("expected packet")
	}
	if len(packet) != 6 {
		t.Fatalf("len = %d, want 6", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[:2]); got != PacketCZReqNameRE {
		t.Fatalf("opcode = 0x%04X, want 0x%04X", got, PacketCZReqNameRE)
	}
	if got := binary.LittleEndian.Uint32(packet[2:6]); got != 0x11223344 {
		t.Fatalf("gid = 0x%08X, want 0x11223344", got)
	}
}

func TestBuildWalkToXYPacketFor2008ClientDate(t *testing.T) {
	packet, ok := BuildWalkToXYPacketForClientDate(150, 200, 20080910)
	if !ok {
		t.Fatal("expected packet")
	}
	if len(packet) != 8 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0xA7 || packet[1] != 0x00 {
		t.Fatalf("unexpected opcode: % x", packet[:2])
	}
	want0, want1, want2 := packPosition(150, 200, 0)
	if packet[5] != want0 || packet[6] != want1 || packet[7] != want2 {
		t.Fatalf("unexpected dest: got % x want %02x %02x %02x", packet[5:8], want0, want1, want2)
	}
}

func TestBuildWalkToXYPacketFor2021ClientDate(t *testing.T) {
	packet, ok := BuildWalkToXYPacketForClientDate(150, 200, 20211103)
	if !ok {
		t.Fatal("expected packet")
	}
	if len(packet) != 5 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x5F || packet[1] != 0x03 {
		t.Fatalf("unexpected opcode: % x", packet[:2])
	}
	want0, want1, want2 := packPosition(150, 200, 0)
	if packet[2] != want0 || packet[3] != want1 || packet[4] != want2 {
		t.Fatalf("unexpected dest: got % x want %02x %02x %02x", packet[2:5], want0, want1, want2)
	}
}

func TestBuildActionRequestPacketFor2008ClientDate(t *testing.T) {
	packet := BuildActionRequestPacketForClientDate(0x11223344, ActionSitDown, 20080910)
	if len(packet) != 19 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x90 || packet[1] != 0x01 {
		t.Fatalf("unexpected opcode: % x", packet[:2])
	}
	if got := packet[5:9]; got[0] != 0x44 || got[1] != 0x33 || got[2] != 0x22 || got[3] != 0x11 {
		t.Fatalf("unexpected target: % x", got)
	}
	if packet[18] != ActionSitDown {
		t.Fatalf("action = %d", packet[18])
	}
}

func TestBuildChangeDirectionPacketFor2008ClientDate(t *testing.T) {
	packet := BuildChangeDirectionPacketForClientDate(2, 6, 20080910)
	if len(packet) != 11 {
		t.Fatalf("len = %d", len(packet))
	}
	if packet[0] != 0x85 || packet[1] != 0x00 {
		t.Fatalf("unexpected opcode: % x", packet[:2])
	}
	if got := binary.LittleEndian.Uint16(packet[7:9]); got != 2 {
		t.Fatalf("head dir = %d", got)
	}
	if packet[10] != 6 {
		t.Fatalf("dir = %d", packet[10])
	}
}

func TestBuildActionRequestPacketFor2012ClientDate(t *testing.T) {
	packet := BuildActionRequestPacketForClientDate(0x11223344, ActionAttack, 20120410)
	want := []byte{0x69, 0x03, 0x44, 0x33, 0x22, 0x11, 0x07}
	if len(packet) != len(want) {
		t.Fatalf("len = %d", len(packet))
	}
	for i := range want {
		if packet[i] != want[i] {
			t.Fatalf("packet = % x, want % x", packet, want)
		}
	}
}

func TestBuildActionRequestPacketFor2021ClientDate(t *testing.T) {
	packet := BuildActionRequestPacketForClientDate(0x11223344, ActionStandUp, 20211103)
	want := []byte{0x37, 0x04, 0x44, 0x33, 0x22, 0x11, 0x03}
	if len(packet) != len(want) {
		t.Fatalf("len = %d", len(packet))
	}
	for i := range want {
		if packet[i] != want[i] {
			t.Fatalf("packet = % x, want % x", packet, want)
		}
	}
}
