package network

import (
	"encoding/binary"
	"fmt"
)

const (
	PacketCZBlacksmithRank  uint16 = 0x0217
	PacketCZAlchemistRank   uint16 = 0x0218
	PacketZCBlacksmithRank  uint16 = 0x0219
	PacketZCAlchemistRank   uint16 = 0x021A
	PacketZCBlacksmithPoint uint16 = 0x021B
	PacketZCAlchemistPoint  uint16 = 0x021C
	PacketZCTaekwonPoint    uint16 = 0x0224
	PacketCZTaekwonRank     uint16 = 0x0225
	PacketZCTaekwonRank     uint16 = 0x0226
)

const fameRankingEntryCount = 10

type FameRankingKind uint8

const (
	FameRankingUnknown FameRankingKind = iota
	FameRankingBlacksmith
	FameRankingAlchemist
	FameRankingTaekwon
)

type FamePointUpdate struct {
	Kind         FameRankingKind
	GainedPoints uint32
	TotalPoints  uint32
}

type FameRankingEntry struct {
	Name   string
	Points uint32
}

type FameRanking struct {
	Kind    FameRankingKind
	Entries []FameRankingEntry
}

func ParseFamePointUpdate(packet Packet) (FamePointUpdate, bool, error) {
	kind, ok := fameRankingKindForPointPacket(packet.ID)
	if !ok {
		return FamePointUpdate{}, false, nil
	}
	if len(packet.Data) < 10 {
		return FamePointUpdate{}, true, fmt.Errorf("fame point packet 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	return FamePointUpdate{
		Kind:         kind,
		GainedPoints: binary.LittleEndian.Uint32(packet.Data[2:6]),
		TotalPoints:  binary.LittleEndian.Uint32(packet.Data[6:10]),
	}, true, nil
}

func ParseFameRanking(packet Packet) (FameRanking, bool, error) {
	kind, ok := fameRankingKindForResponsePacket(packet.ID)
	if !ok {
		return FameRanking{}, false, nil
	}
	if len(packet.Data) < 282 {
		return FameRanking{}, true, fmt.Errorf("fame ranking packet 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	entries := make([]FameRankingEntry, fameRankingEntryCount)
	for i := range entries {
		nameOffset := 2 + i*24
		pointOffset := 2 + fameRankingEntryCount*24 + i*4
		entries[i] = FameRankingEntry{
			Name:   decodeROFixedString(packet.Data[nameOffset : nameOffset+24]),
			Points: binary.LittleEndian.Uint32(packet.Data[pointOffset : pointOffset+4]),
		}
	}
	return FameRanking{Kind: kind, Entries: entries}, true, nil
}

func BuildFameRankingRequestPacket(kind FameRankingKind) ([]byte, error) {
	packetID, ok := fameRankingRequestPacketID(kind)
	if !ok {
		return nil, fmt.Errorf("invalid fame ranking kind: %d", kind)
	}
	packet := make([]byte, 2)
	binary.LittleEndian.PutUint16(packet, packetID)
	return packet, nil
}

func (c *Client) SendFameRankingRequest(kind FameRankingKind) error {
	packet, err := BuildFameRankingRequestPacket(kind)
	if err != nil {
		return err
	}
	return c.Send(packet)
}

func fameRankingKindForPointPacket(packetID uint16) (FameRankingKind, bool) {
	switch packetID {
	case PacketZCBlacksmithPoint:
		return FameRankingBlacksmith, true
	case PacketZCAlchemistPoint:
		return FameRankingAlchemist, true
	case PacketZCTaekwonPoint:
		return FameRankingTaekwon, true
	default:
		return 0, false
	}
}

func fameRankingKindForResponsePacket(packetID uint16) (FameRankingKind, bool) {
	switch packetID {
	case PacketZCBlacksmithRank:
		return FameRankingBlacksmith, true
	case PacketZCAlchemistRank:
		return FameRankingAlchemist, true
	case PacketZCTaekwonRank:
		return FameRankingTaekwon, true
	default:
		return 0, false
	}
}

func fameRankingRequestPacketID(kind FameRankingKind) (uint16, bool) {
	switch kind {
	case FameRankingBlacksmith:
		return PacketCZBlacksmithRank, true
	case FameRankingAlchemist:
		return PacketCZAlchemistRank, true
	case FameRankingTaekwon:
		return PacketCZTaekwonRank, true
	default:
		return 0, false
	}
}
