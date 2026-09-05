package network

import (
	"encoding/binary"
	"fmt"
)

const (
	PacketZCStarSkill      uint16 = 0x020E
	PacketZCStarPlace      uint16 = 0x0253
	PacketCZAgreeStarPlace uint16 = 0x0254
)

type TaekwonMission struct {
	MonsterName string
	MonsterID   uint32
	Progress    uint8
	Result      uint8
}

type StarPlace struct {
	Place uint8
}

func ParseTaekwonMission(packet Packet) (TaekwonMission, bool, error) {
	if packet.ID != PacketZCStarSkill {
		return TaekwonMission{}, false, nil
	}
	if len(packet.Data) < 32 {
		return TaekwonMission{}, true, fmt.Errorf("ZC_STARSKILL too short: %d", len(packet.Data))
	}
	return TaekwonMission{
		MonsterName: decodeROFixedString(packet.Data[2:26]),
		MonsterID:   binary.LittleEndian.Uint32(packet.Data[26:30]),
		Progress:    packet.Data[30],
		Result:      packet.Data[31],
	}, true, nil
}

func ParseStarPlace(packet Packet) (StarPlace, bool, error) {
	if packet.ID != PacketZCStarPlace {
		return StarPlace{}, false, nil
	}
	if len(packet.Data) < 3 {
		return StarPlace{}, true, fmt.Errorf("ZC_STARPLACE too short: %d", len(packet.Data))
	}
	place := packet.Data[2]
	if place > 2 {
		return StarPlace{}, true, fmt.Errorf("ZC_STARPLACE invalid place: %d", place)
	}
	return StarPlace{Place: place}, true, nil
}

func BuildAgreeStarPlacePacket(place uint8) []byte {
	packet := make([]byte, 3)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZAgreeStarPlace)
	packet[2] = place
	return packet
}

func (c *Client) SendAgreeStarPlace(place uint8) error {
	if place > 2 {
		return fmt.Errorf("invalid Star Gladiator place: %d", place)
	}
	return c.Send(BuildAgreeStarPlacePacket(place))
}
