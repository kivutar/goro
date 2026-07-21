package network

import (
	"encoding/binary"
	"fmt"
)

const (
	PacketCZCommandMercenary      uint16 = 0x022D
	PacketZCPropertyHomunculus    uint16 = 0x022E
	PacketZCFeedMercenary         uint16 = 0x022F
	PacketZCChangeStateMercenary  uint16 = 0x0230
	PacketCZRenameMercenary       uint16 = 0x0231
	PacketCZRequestMoveNPC        uint16 = 0x0232
	PacketCZRequestActNPC         uint16 = 0x0233
	PacketCZRequestMoveToOwner    uint16 = 0x0234
	PacketZCHomunculusSkillList   uint16 = 0x0235
	PacketZCHomunculusSkillUpdate uint16 = 0x0239
	PacketZCPropertyMercenaryOld  uint16 = 0x027D
	PacketZCMercenaryInit         uint16 = 0x029B
	PacketZCMercenaryProperty     uint16 = 0x029C
	PacketZCMercenarySkillList    uint16 = 0x029D
	PacketZCMercenarySkillUpdate  uint16 = 0x029E
	PacketCZMercenaryCommand      uint16 = 0x029F
	PacketZCMercenaryParamChange  uint16 = 0x02A2
)

const (
	HomunculusCommandInfo   uint8 = 0
	HomunculusCommandFeed   uint8 = 1
	HomunculusCommandDelete uint8 = 2

	MercenaryCommandInfo   uint8 = 1
	MercenaryCommandDelete uint8 = 2
)

type HomunculusProperty struct {
	Name         string
	Flags        uint8
	Level        int
	Hunger       int
	Intimacy     int
	ItemID       uint32
	Attack       int
	MagicAttack  int
	Hit          int
	Critical     int
	Defense      int
	MagicDefense int
	Flee         int
	ASPD         int
	HP           int
	MaxHP        int
	SP           int
	MaxSP        int
	Exp          uint32
	MaxExp       uint32
	SkillPoints  int
	AttackRange  int
}

type HomunculusFeedResult struct {
	Result bool
	ItemID uint16
}

type HomunculusStateChange struct {
	Type  uint8
	State uint8
	GID   uint32
	Data  uint32
}

type HomunculusSkillInfoList struct {
	Skills []SkillInfo
}

type HomunculusSkillInfoUpdate struct {
	Skill SkillInfo
}

type MercenaryProperty struct {
	ID           uint32
	Name         string
	Level        int
	Faith        int
	SummonCount  int
	Calls        uint32
	Kills        uint32
	ExpireTick   uint32
	Attack       int
	MagicAttack  int
	Hit          int
	Critical     int
	Defense      int
	MagicDefense int
	Flee         int
	ASPD         int
	HP           int
	MaxHP        int
	SP           int
	MaxSP        int
	Exp          uint32
	AttackRange  int
}

type MercenaryParamChange struct {
	Param uint16
	Value int32
}

type MercenarySkillInfoList struct {
	Skills []SkillInfo
}

type MercenarySkillInfoUpdate struct {
	Skill SkillInfo
}

func ParseHomunculusProperty(packet Packet) (HomunculusProperty, bool, error) {
	if packet.ID != PacketZCPropertyHomunculus {
		return HomunculusProperty{}, false, nil
	}
	if len(packet.Data) < 71 {
		return HomunculusProperty{}, false, fmt.Errorf("ZC_PROPERTY_HOMUN too short: %d", len(packet.Data))
	}
	return HomunculusProperty{
		Name:         decodeROFixedString(packet.Data[2:26]),
		Flags:        packet.Data[26],
		Level:        int(binary.LittleEndian.Uint16(packet.Data[27:29])),
		Hunger:       int(binary.LittleEndian.Uint16(packet.Data[29:31])),
		Intimacy:     int(binary.LittleEndian.Uint16(packet.Data[31:33])),
		ItemID:       uint32(binary.LittleEndian.Uint16(packet.Data[33:35])),
		Attack:       int(binary.LittleEndian.Uint16(packet.Data[35:37])),
		MagicAttack:  int(binary.LittleEndian.Uint16(packet.Data[37:39])),
		Hit:          int(binary.LittleEndian.Uint16(packet.Data[39:41])),
		Critical:     int(binary.LittleEndian.Uint16(packet.Data[41:43])),
		Defense:      int(binary.LittleEndian.Uint16(packet.Data[43:45])),
		MagicDefense: int(binary.LittleEndian.Uint16(packet.Data[45:47])),
		Flee:         int(binary.LittleEndian.Uint16(packet.Data[47:49])),
		ASPD:         int(binary.LittleEndian.Uint16(packet.Data[49:51])),
		HP:           int(binary.LittleEndian.Uint16(packet.Data[51:53])),
		MaxHP:        int(binary.LittleEndian.Uint16(packet.Data[53:55])),
		SP:           int(binary.LittleEndian.Uint16(packet.Data[55:57])),
		MaxSP:        int(binary.LittleEndian.Uint16(packet.Data[57:59])),
		Exp:          binary.LittleEndian.Uint32(packet.Data[59:63]),
		MaxExp:       binary.LittleEndian.Uint32(packet.Data[63:67]),
		SkillPoints:  int(binary.LittleEndian.Uint16(packet.Data[67:69])),
		AttackRange:  int(binary.LittleEndian.Uint16(packet.Data[69:71])),
	}, true, nil
}

func ParseHomunculusFeedResult(packet Packet) (HomunculusFeedResult, bool, error) {
	if packet.ID != PacketZCFeedMercenary {
		return HomunculusFeedResult{}, false, nil
	}
	if len(packet.Data) < 5 {
		return HomunculusFeedResult{}, false, fmt.Errorf("ZC_FEED_MER too short: %d", len(packet.Data))
	}
	return HomunculusFeedResult{
		Result: packet.Data[2] != 0,
		ItemID: binary.LittleEndian.Uint16(packet.Data[3:5]),
	}, true, nil
}

func ParseHomunculusStateChange(packet Packet) (HomunculusStateChange, bool, error) {
	if packet.ID != PacketZCChangeStateMercenary {
		return HomunculusStateChange{}, false, nil
	}
	if len(packet.Data) < 12 {
		return HomunculusStateChange{}, false, fmt.Errorf("ZC_CHANGESTATE_MER too short: %d", len(packet.Data))
	}
	return HomunculusStateChange{
		Type:  packet.Data[2],
		State: packet.Data[3],
		GID:   binary.LittleEndian.Uint32(packet.Data[4:8]),
		Data:  binary.LittleEndian.Uint32(packet.Data[8:12]),
	}, true, nil
}

func ParseHomunculusSkillInfoList(packet Packet) (HomunculusSkillInfoList, bool, error) {
	if packet.ID != PacketZCHomunculusSkillList {
		return HomunculusSkillInfoList{}, false, nil
	}
	skills, err := parseCompanionSkillList(packet, 0)
	if err != nil {
		return HomunculusSkillInfoList{}, false, err
	}
	return HomunculusSkillInfoList{Skills: skills}, true, nil
}

func ParseHomunculusSkillInfoUpdate(packet Packet) (HomunculusSkillInfoUpdate, bool, error) {
	if packet.ID != PacketZCHomunculusSkillUpdate {
		return HomunculusSkillInfoUpdate{}, false, nil
	}
	skill, err := parseCompanionSkillUpdate(packet)
	if err != nil {
		return HomunculusSkillInfoUpdate{}, false, err
	}
	return HomunculusSkillInfoUpdate{Skill: skill}, true, nil
}

func ParseMercenaryProperty(packet Packet) (MercenaryProperty, bool, error) {
	switch packet.ID {
	case PacketZCMercenaryInit:
		if len(packet.Data) < 80 {
			return MercenaryProperty{}, false, fmt.Errorf("ZC_MER_INIT too short: %d", len(packet.Data))
		}
		return MercenaryProperty{
			ID:           binary.LittleEndian.Uint32(packet.Data[2:6]),
			Attack:       int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			MagicAttack:  int(binary.LittleEndian.Uint16(packet.Data[8:10])),
			Hit:          int(binary.LittleEndian.Uint16(packet.Data[10:12])),
			Critical:     int(binary.LittleEndian.Uint16(packet.Data[12:14])),
			Defense:      int(binary.LittleEndian.Uint16(packet.Data[14:16])),
			MagicDefense: int(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Flee:         int(binary.LittleEndian.Uint16(packet.Data[18:20])),
			ASPD:         int(binary.LittleEndian.Uint16(packet.Data[20:22])),
			Name:         decodeROFixedString(packet.Data[22:46]),
			Level:        int(binary.LittleEndian.Uint16(packet.Data[46:48])),
			HP:           int(binary.LittleEndian.Uint32(packet.Data[48:52])),
			MaxHP:        int(binary.LittleEndian.Uint32(packet.Data[52:56])),
			SP:           int(binary.LittleEndian.Uint32(packet.Data[56:60])),
			MaxSP:        int(binary.LittleEndian.Uint32(packet.Data[60:64])),
			ExpireTick:   binary.LittleEndian.Uint32(packet.Data[64:68]),
			Faith:        int(binary.LittleEndian.Uint16(packet.Data[68:70])),
			Calls:        binary.LittleEndian.Uint32(packet.Data[70:74]),
			Kills:        binary.LittleEndian.Uint32(packet.Data[74:78]),
			AttackRange:  int(binary.LittleEndian.Uint16(packet.Data[78:80])),
		}, true, nil
	case PacketZCMercenaryProperty:
		if len(packet.Data) < 66 {
			return MercenaryProperty{}, false, fmt.Errorf("ZC_MER_PROPERTY too short: %d", len(packet.Data))
		}
		return MercenaryProperty{
			Attack:       int(binary.LittleEndian.Uint16(packet.Data[2:4])),
			MagicAttack:  int(binary.LittleEndian.Uint16(packet.Data[4:6])),
			Hit:          int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			Critical:     int(binary.LittleEndian.Uint16(packet.Data[8:10])),
			Defense:      int(binary.LittleEndian.Uint16(packet.Data[10:12])),
			MagicDefense: int(binary.LittleEndian.Uint16(packet.Data[12:14])),
			Flee:         int(binary.LittleEndian.Uint16(packet.Data[14:16])),
			ASPD:         int(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Name:         decodeROFixedString(packet.Data[18:42]),
			Level:        int(binary.LittleEndian.Uint16(packet.Data[42:44])),
			HP:           int(binary.LittleEndian.Uint16(packet.Data[44:46])),
			MaxHP:        int(binary.LittleEndian.Uint16(packet.Data[46:48])),
			SP:           int(binary.LittleEndian.Uint16(packet.Data[48:50])),
			MaxSP:        int(binary.LittleEndian.Uint16(packet.Data[50:52])),
			ExpireTick:   binary.LittleEndian.Uint32(packet.Data[52:56]),
			Faith:        int(binary.LittleEndian.Uint16(packet.Data[56:58])),
			Calls:        binary.LittleEndian.Uint32(packet.Data[58:62]),
			Kills:        binary.LittleEndian.Uint32(packet.Data[62:66]),
		}, true, nil
	case PacketZCPropertyMercenaryOld:
		if len(packet.Data) < 62 {
			return MercenaryProperty{}, false, fmt.Errorf("ZC_PROPERTY_MERCE too short: %d", len(packet.Data))
		}
		return MercenaryProperty{
			Name:         decodeROFixedString(packet.Data[2:26]),
			Level:        int(binary.LittleEndian.Uint16(packet.Data[26:28])),
			Faith:        int(binary.LittleEndian.Uint16(packet.Data[28:30])),
			SummonCount:  int(binary.LittleEndian.Uint16(packet.Data[30:32])),
			Attack:       int(binary.LittleEndian.Uint16(packet.Data[32:34])),
			MagicAttack:  int(binary.LittleEndian.Uint16(packet.Data[34:36])),
			Hit:          int(binary.LittleEndian.Uint16(packet.Data[36:38])),
			Critical:     int(binary.LittleEndian.Uint16(packet.Data[38:40])),
			Defense:      int(binary.LittleEndian.Uint16(packet.Data[40:42])),
			MagicDefense: int(binary.LittleEndian.Uint16(packet.Data[42:44])),
			Flee:         int(binary.LittleEndian.Uint16(packet.Data[44:46])),
			ASPD:         int(binary.LittleEndian.Uint16(packet.Data[46:48])),
			HP:           int(binary.LittleEndian.Uint16(packet.Data[48:50])),
			MaxHP:        int(binary.LittleEndian.Uint16(packet.Data[50:52])),
			SP:           int(binary.LittleEndian.Uint16(packet.Data[52:54])),
			MaxSP:        int(binary.LittleEndian.Uint16(packet.Data[54:56])),
			AttackRange:  int(binary.LittleEndian.Uint16(packet.Data[56:58])),
			Exp:          binary.LittleEndian.Uint32(packet.Data[58:62]),
		}, true, nil
	default:
		return MercenaryProperty{}, false, nil
	}
}

func ParseMercenaryParamChange(packet Packet) (MercenaryParamChange, bool, error) {
	if packet.ID != PacketZCMercenaryParamChange {
		return MercenaryParamChange{}, false, nil
	}
	if len(packet.Data) < 8 {
		return MercenaryParamChange{}, false, fmt.Errorf("ZC_MER_PAR_CHANGE too short: %d", len(packet.Data))
	}
	return MercenaryParamChange{
		Param: binary.LittleEndian.Uint16(packet.Data[2:4]),
		Value: int32(binary.LittleEndian.Uint32(packet.Data[4:8])),
	}, true, nil
}

func ParseMercenarySkillInfoList(packet Packet) (MercenarySkillInfoList, bool, error) {
	if packet.ID != PacketZCMercenarySkillList {
		return MercenarySkillInfoList{}, false, nil
	}
	skills, err := parseCompanionSkillList(packet, 0)
	if err != nil {
		return MercenarySkillInfoList{}, false, err
	}
	return MercenarySkillInfoList{Skills: skills}, true, nil
}

func ParseMercenarySkillInfoUpdate(packet Packet) (MercenarySkillInfoUpdate, bool, error) {
	if packet.ID != PacketZCMercenarySkillUpdate {
		return MercenarySkillInfoUpdate{}, false, nil
	}
	skill, err := parseCompanionSkillUpdate(packet)
	if err != nil {
		return MercenarySkillInfoUpdate{}, false, err
	}
	return MercenarySkillInfoUpdate{Skill: skill}, true, nil
}

func parseCompanionSkillList(packet Packet, entryOffset int) ([]SkillInfo, error) {
	if len(packet.Data) < 4 {
		return nil, fmt.Errorf("companion skill list too short: %d", len(packet.Data))
	}
	packetLen := int(binary.LittleEndian.Uint16(packet.Data[2:4]))
	if packetLen <= 0 || packetLen > len(packet.Data) {
		packetLen = len(packet.Data)
	}
	body := packet.Data[4:packetLen]
	if len(body)%skillInfoEntryLen != 0 {
		return nil, fmt.Errorf("companion skill list bad body len: %d", len(body))
	}
	skills := make([]SkillInfo, 0, len(body)/skillInfoEntryLen)
	for offset := 0; offset < len(body); offset += skillInfoEntryLen {
		skills = append(skills, parseSkillInfoEntry(body[offset:offset+skillInfoEntryLen], entryOffset))
	}
	return skills, nil
}

func parseCompanionSkillUpdate(packet Packet) (SkillInfo, error) {
	if len(packet.Data) < 11 {
		return SkillInfo{}, fmt.Errorf("companion skill update too short: %d", len(packet.Data))
	}
	return SkillInfo{
		ID:         binary.LittleEndian.Uint16(packet.Data[2:4]),
		Level:      int(binary.LittleEndian.Uint16(packet.Data[4:6])),
		SPCost:     int(binary.LittleEndian.Uint16(packet.Data[6:8])),
		Range:      int(binary.LittleEndian.Uint16(packet.Data[8:10])),
		Upgradable: packet.Data[10] != 0,
	}, nil
}

func BuildHomunculusCommandPacket(command uint8) []byte {
	return BuildHomunculusCommandPacketWithType(0, command)
}

func BuildHomunculusCommandPacketWithType(commandType uint16, command uint8) []byte {
	var w Writer
	w.Uint16(PacketCZCommandMercenary)
	w.Uint16(commandType)
	w.Uint8(command)
	return w.Bytes()
}

func BuildHomunculusRenamePacket(name string) []byte {
	var w Writer
	w.Uint16(PacketCZRenameMercenary)
	w.CString(name, 24)
	return w.Bytes()
}

func BuildCompanionMovePacket(gid uint32, x, y int) ([]byte, bool) {
	dest, ok := EncodeMoveDestination(x, y)
	if !ok {
		return nil, false
	}
	var w Writer
	w.Uint16(PacketCZRequestMoveNPC)
	w.Uint32(gid)
	_, _ = w.Write(dest[:])
	return w.Bytes(), true
}

func BuildCompanionAttackPacket(gid, targetGID uint32, action uint8) []byte {
	var w Writer
	w.Uint16(PacketCZRequestActNPC)
	w.Uint32(gid)
	w.Uint32(targetGID)
	w.Uint8(action)
	return w.Bytes()
}

func BuildCompanionMoveToOwnerPacket(gid uint32) []byte {
	var w Writer
	w.Uint16(PacketCZRequestMoveToOwner)
	w.Uint32(gid)
	return w.Bytes()
}

func BuildMercenaryCommandPacket(command uint8) []byte {
	var w Writer
	w.Uint16(PacketCZMercenaryCommand)
	w.Uint8(command)
	return w.Bytes()
}
