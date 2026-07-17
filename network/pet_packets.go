package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketZCStartCapture      uint16 = 0x019E
	PacketCZTryCaptureMonster uint16 = 0x019F
	PacketZCTryCaptureMonster uint16 = 0x01A0
	PacketCZCommandPet        uint16 = 0x01A1
	PacketZCPropertyPet       uint16 = 0x01A2
	PacketZCFeedPet           uint16 = 0x01A3
	PacketZCChangeStatePet    uint16 = 0x01A4
	PacketCZRenamePet         uint16 = 0x01A5
	PacketZCPetEggList        uint16 = 0x01A6
	PacketCZSelectPetEgg      uint16 = 0x01A7
	PacketCZPetAct            uint16 = 0x01A9
	PacketZCPetAct            uint16 = 0x01AA
)

const (
	PetCommandInfo uint8 = iota
	PetCommandFeed
	PetCommandPerformance
	PetCommandBackToEgg
	PetCommandUnequipAccessory
)

type PetCaptureResult struct {
	Success bool
}

type PetCaptureStart struct{}

type PetEggList struct {
	Indexes []uint16
}

type PetProperty struct {
	Name         string
	Modified     bool
	Level        uint16
	Fullness     uint16
	Relationship uint16
	AccessoryID  uint16
	Job          uint16
}

type PetFeedResult struct {
	Success bool
	ItemID  uint16
}

type PetStateChange struct {
	Type uint8
	ID   uint32
	Data uint32
}

type PetAction struct {
	ID   uint32
	Data uint32
}

func ParsePetCaptureStart(packet Packet) (PetCaptureStart, bool, error) {
	if packet.ID != PacketZCStartCapture {
		return PetCaptureStart{}, false, nil
	}
	if len(packet.Data) < 2 {
		return PetCaptureStart{}, false, fmt.Errorf("ZC_START_CAPTURE too short: %d", len(packet.Data))
	}
	return PetCaptureStart{}, true, nil
}

func ParsePetCaptureResult(packet Packet) (PetCaptureResult, bool, error) {
	if packet.ID != PacketZCTryCaptureMonster {
		return PetCaptureResult{}, false, nil
	}
	if len(packet.Data) < 3 {
		return PetCaptureResult{}, false, fmt.Errorf("ZC_TRYCAPTURE_MONSTER too short: %d", len(packet.Data))
	}
	return PetCaptureResult{Success: packet.Data[2] != 0}, true, nil
}

func ParsePetProperty(packet Packet) (PetProperty, bool, error) {
	if packet.ID != PacketZCPropertyPet {
		return PetProperty{}, false, nil
	}
	if len(packet.Data) < 35 {
		return PetProperty{}, false, fmt.Errorf("ZC_PROPERTY_PET too short: %d", len(packet.Data))
	}
	property := PetProperty{
		Name:         decodeROFixedString(packet.Data[2:26]),
		Modified:     packet.Data[26] != 0,
		Level:        binary.LittleEndian.Uint16(packet.Data[27:29]),
		Fullness:     binary.LittleEndian.Uint16(packet.Data[29:31]),
		Relationship: binary.LittleEndian.Uint16(packet.Data[31:33]),
		AccessoryID:  binary.LittleEndian.Uint16(packet.Data[33:35]),
	}
	if len(packet.Data) >= 37 {
		property.Job = binary.LittleEndian.Uint16(packet.Data[35:37])
	}
	return property, true, nil
}

func ParsePetFeedResult(packet Packet) (PetFeedResult, bool, error) {
	if packet.ID != PacketZCFeedPet {
		return PetFeedResult{}, false, nil
	}
	if len(packet.Data) < 5 {
		return PetFeedResult{}, false, fmt.Errorf("ZC_FEED_PET too short: %d", len(packet.Data))
	}
	return PetFeedResult{
		Success: packet.Data[2] != 0,
		ItemID:  binary.LittleEndian.Uint16(packet.Data[3:5]),
	}, true, nil
}

func ParsePetStateChange(packet Packet) (PetStateChange, bool, error) {
	if packet.ID != PacketZCChangeStatePet {
		return PetStateChange{}, false, nil
	}
	if len(packet.Data) < 11 {
		return PetStateChange{}, false, fmt.Errorf("ZC_CHANGESTATE_PET too short: %d", len(packet.Data))
	}
	return PetStateChange{
		Type: packet.Data[2],
		ID:   binary.LittleEndian.Uint32(packet.Data[3:7]),
		Data: binary.LittleEndian.Uint32(packet.Data[7:11]),
	}, true, nil
}

func ParsePetAction(packet Packet) (PetAction, bool, error) {
	if packet.ID != PacketZCPetAct {
		return PetAction{}, false, nil
	}
	if len(packet.Data) < 10 {
		return PetAction{}, false, fmt.Errorf("ZC_PET_ACT too short: %d", len(packet.Data))
	}
	return PetAction{
		ID:   binary.LittleEndian.Uint32(packet.Data[2:6]),
		Data: binary.LittleEndian.Uint32(packet.Data[6:10]),
	}, true, nil
}

func ParsePetEggList(packet Packet) (PetEggList, bool, error) {
	if packet.ID != PacketZCPetEggList {
		return PetEggList{}, false, nil
	}
	if len(packet.Data) < 4 {
		return PetEggList{}, false, fmt.Errorf("ZC_PETEGG_LIST too short: %d", len(packet.Data))
	}
	packetLen := int(binary.LittleEndian.Uint16(packet.Data[2:4]))
	if packetLen <= 0 || packetLen > len(packet.Data) {
		packetLen = len(packet.Data)
	}
	body := packet.Data[4:packetLen]
	if len(body)%2 != 0 {
		return PetEggList{}, false, fmt.Errorf("ZC_PETEGG_LIST bad body len: %d", len(body))
	}
	indexes := make([]uint16, 0, len(body)/2)
	for offset := 0; offset < len(body); offset += 2 {
		indexes = append(indexes, binary.LittleEndian.Uint16(body[offset:offset+2]))
	}
	return PetEggList{Indexes: indexes}, true, nil
}

func BuildTryCaptureMonsterPacket(targetID uint32) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZTryCaptureMonster)
	binary.LittleEndian.PutUint32(packet[2:6], targetID)
	return packet
}

func BuildSelectPetEggPacket(index uint16) []byte {
	packet := make([]byte, 4)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZSelectPetEgg)
	binary.LittleEndian.PutUint16(packet[2:4], index)
	return packet
}

func BuildCommandPetPacket(command uint8) []byte {
	packet := make([]byte, 3)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZCommandPet)
	packet[2] = command
	return packet
}

func BuildPetActPacket(data uint32) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZPetAct)
	binary.LittleEndian.PutUint32(packet[2:6], data)
	return packet
}

func BuildRenamePetPacket(name string) []byte {
	packet := make([]byte, 26)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZRenamePet)
	copy(packet[2:26], encodeROFixedString(name, 24))
	return packet
}

func (c *Client) SendTryCaptureMonster(targetID uint32) error {
	packet := BuildTryCaptureMonsterPacket(targetID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_TRYCAPTURE_MONSTER opcode=0x%04X target=%d client_date=%d", ID(packet), targetID, c.clientDate)
	} else {
		glog.Warnf("send CZ_TRYCAPTURE_MONSTER failed opcode=0x%04X len=%d target=%d client_date=%d: %v", ID(packet), len(packet), targetID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendPetCommand(command uint8) error {
	packet := BuildCommandPetPacket(command)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_COMMAND_PET opcode=0x%04X command=%d client_date=%d", ID(packet), command, c.clientDate)
	} else {
		glog.Warnf("send CZ_COMMAND_PET failed opcode=0x%04X len=%d command=%d client_date=%d: %v", ID(packet), len(packet), command, c.clientDate, err)
	}
	return err
}

func (c *Client) SendPetAct(data uint32) error {
	packet := BuildPetActPacket(data)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_PET_ACT opcode=0x%04X data=%d client_date=%d", ID(packet), data, c.clientDate)
	} else {
		glog.Warnf("send CZ_PET_ACT failed opcode=0x%04X len=%d data=%d client_date=%d: %v", ID(packet), len(packet), data, c.clientDate, err)
	}
	return err
}

func (c *Client) SendRenamePet(name string) error {
	packet := BuildRenamePetPacket(name)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_RENAME_PET opcode=0x%04X name=%q client_date=%d", ID(packet), name, c.clientDate)
	} else {
		glog.Warnf("send CZ_RENAME_PET failed opcode=0x%04X len=%d name=%q client_date=%d: %v", ID(packet), len(packet), name, c.clientDate, err)
	}
	return err
}

func (c *Client) SendSelectPetEgg(index uint16) error {
	packet := BuildSelectPetEggPacket(index)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_SELECT_PETEGG opcode=0x%04X index=%d client_date=%d", ID(packet), index, c.clientDate)
	} else {
		glog.Warnf("send CZ_SELECT_PETEGG failed opcode=0x%04X len=%d index=%d client_date=%d: %v", ID(packet), len(packet), index, c.clientDate, err)
	}
	return err
}
