package network

import (
	"encoding/binary"
	"testing"
)

func TestParseFamePointUpdate(t *testing.T) {
	tests := []struct {
		packetID uint16
		kind     FameRankingKind
	}{
		{PacketZCBlacksmithPoint, FameRankingBlacksmith},
		{PacketZCAlchemistPoint, FameRankingAlchemist},
		{PacketZCTaekwonPoint, FameRankingTaekwon},
	}
	for _, test := range tests {
		data := make([]byte, 10)
		binary.LittleEndian.PutUint16(data[0:2], test.packetID)
		binary.LittleEndian.PutUint32(data[2:6], 25)
		binary.LittleEndian.PutUint32(data[6:10], 120)

		update, ok, err := ParseFamePointUpdate(Packet{ID: test.packetID, Data: data})
		if err != nil || !ok || update.Kind != test.kind || update.GainedPoints != 25 || update.TotalPoints != 120 {
			t.Fatalf("packet 0x%04X update=%+v ok=%t err=%v", test.packetID, update, ok, err)
		}
	}
}

func TestParseFameRanking(t *testing.T) {
	tests := []struct {
		packetID uint16
		kind     FameRankingKind
	}{
		{PacketZCBlacksmithRank, FameRankingBlacksmith},
		{PacketZCAlchemistRank, FameRankingAlchemist},
		{PacketZCTaekwonRank, FameRankingTaekwon},
	}
	for _, test := range tests {
		data := make([]byte, 282)
		binary.LittleEndian.PutUint16(data[0:2], test.packetID)
		for i := 0; i < fameRankingEntryCount; i++ {
			name := []byte{'R', 'a', 'n', 'k', byte('0' + i)}
			copy(data[2+i*24:2+(i+1)*24], name)
			pointOffset := 2 + fameRankingEntryCount*24 + i*4
			binary.LittleEndian.PutUint32(data[pointOffset:pointOffset+4], uint32(1000-i*10))
		}

		ranking, ok, err := ParseFameRanking(Packet{ID: test.packetID, Data: data})
		if err != nil || !ok {
			t.Fatalf("packet 0x%04X parse ranking ok=%t err=%v", test.packetID, ok, err)
		}
		if ranking.Kind != test.kind || len(ranking.Entries) != 10 || ranking.Entries[0].Name != "Rank0" || ranking.Entries[0].Points != 1000 || ranking.Entries[9].Name != "Rank9" || ranking.Entries[9].Points != 910 {
			t.Fatalf("packet 0x%04X ranking=%+v", test.packetID, ranking)
		}
	}
}

func TestBuildFameRankingRequestPacket(t *testing.T) {
	tests := []struct {
		kind     FameRankingKind
		packetID uint16
	}{
		{FameRankingBlacksmith, PacketCZBlacksmithRank},
		{FameRankingAlchemist, PacketCZAlchemistRank},
		{FameRankingTaekwon, PacketCZTaekwonRank},
	}
	for _, test := range tests {
		packet, err := BuildFameRankingRequestPacket(test.kind)
		if err != nil {
			t.Fatalf("kind %d: %v", test.kind, err)
		}
		if len(packet) != 2 || binary.LittleEndian.Uint16(packet) != test.packetID {
			t.Fatalf("kind %d packet=% X", test.kind, packet)
		}
	}
	if _, err := BuildFameRankingRequestPacket(FameRankingKind(255)); err == nil {
		t.Fatal("invalid ranking kind was accepted")
	}
	if _, err := BuildFameRankingRequestPacket(FameRankingUnknown); err == nil {
		t.Fatal("zero-value ranking kind was accepted")
	}
}

func TestFameRankingPacketLengthsAreFramedFor2008(t *testing.T) {
	lengths := PacketLengths2008()
	want := map[uint16]int{
		PacketZCBlacksmithRank: 282, PacketZCAlchemistRank: 282,
		PacketZCBlacksmithPoint: 10, PacketZCAlchemistPoint: 10,
		PacketZCTaekwonPoint: 10, PacketZCTaekwonRank: 282,
	}
	for packetID, wantLen := range want {
		if got := lengths[packetID]; got != wantLen {
			t.Fatalf("packet 0x%04X length=%d want=%d", packetID, got, wantLen)
		}
	}
}

func TestFameRankingPacketParsersRejectShortPackets(t *testing.T) {
	for _, packetID := range []uint16{PacketZCBlacksmithPoint, PacketZCAlchemistPoint, PacketZCTaekwonPoint} {
		if _, ok, err := ParseFamePointUpdate(Packet{ID: packetID, Data: make([]byte, 9)}); !ok || err == nil {
			t.Fatalf("short point packet 0x%04X ok=%t err=%v", packetID, ok, err)
		}
	}
	for _, packetID := range []uint16{PacketZCBlacksmithRank, PacketZCAlchemistRank, PacketZCTaekwonRank} {
		if _, ok, err := ParseFameRanking(Packet{ID: packetID, Data: make([]byte, 281)}); !ok || err == nil {
			t.Fatalf("short rank packet 0x%04X ok=%t err=%v", packetID, ok, err)
		}
	}
}
