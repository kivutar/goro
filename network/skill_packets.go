package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const skillInfoEntryLen = 37
const skillGroundMessageLen = 80

const (
	PacketZCMonsterInfo     uint16 = 0x018C
	PacketZCSkillCastCancel uint16 = 0x01B9
	PacketZCAutoSpellList   uint16 = 0x01CD
	PacketCZSelectAutoSpell uint16 = 0x01CE

	autoSpellChoiceCount = 7
)

type SkillInfo struct {
	ID         uint16
	Type       uint32
	Level      int
	SPCost     int
	Range      int
	Name       string
	Upgradable bool
}

type SkillInfoList struct {
	Skills []SkillInfo
}

type SkillInfoUpdate struct {
	Skill SkillInfo
}

type AutoRunSkill struct {
	Skill SkillInfo
}

type AutoSpellList struct {
	SkillIDs []uint16
}

type MonsterElementRates struct {
	Water  uint8
	Earth  uint8
	Fire   uint8
	Wind   uint8
	Poison uint8
	Holy   uint8
	Shadow uint8
	Ghost  uint8
	Undead uint8
}

type MonsterInfo struct {
	Class        uint16
	Level        uint16
	Size         uint16
	HP           uint32
	Defense      int16
	Race         uint16
	MagicDefense int16
	Property     uint16
	Elements     MonsterElementRates
}

type SkillNoDamageNotify struct {
	SkillID  uint16
	Amount   uint32
	TargetID uint32
	SourceID uint32
	Result   byte
}

type SkillCastNotify struct {
	SourceID  uint32
	TargetID  uint32
	X         uint16
	Y         uint16
	SkillID   uint16
	Property  uint32
	DelayTime uint32
}

type GroundSkillNotify struct {
	SkillID  uint16
	SourceID uint32
	Level    uint16
	X        uint16
	Y        uint16
}

type SkillUnitEntry struct {
	ID         uint32
	CreatorID  uint32
	X          uint16
	Y          uint16
	UnitID     uint16
	Visible    bool
	HasMessage bool
	Message    string
}

type SkillUnitDisappear struct {
	ID uint32
}

type SkillUnitUpdate struct {
	ID uint32
}

type SkillFailAck struct {
	SkillID uint16
	Number  uint32
	ItemID  uint32
	Result  byte
	Cause   byte
}

type WarpPointList struct {
	SkillID  uint16
	MapNames []string
}

type RememberWarpPointAck struct {
	Result byte
}

func ParseSkillInfoList(packet Packet) (SkillInfoList, bool, error) {
	if packet.ID != 0x010F {
		return SkillInfoList{}, false, nil
	}
	if len(packet.Data) < 4 {
		return SkillInfoList{}, false, fmt.Errorf("ZC_SKILLINFO_LIST too short: %d", len(packet.Data))
	}
	packetLen := int(binary.LittleEndian.Uint16(packet.Data[2:4]))
	if packetLen <= 0 || packetLen > len(packet.Data) {
		packetLen = len(packet.Data)
	}
	body := packet.Data[4:packetLen]
	if len(body)%skillInfoEntryLen != 0 {
		return SkillInfoList{}, false, fmt.Errorf("ZC_SKILLINFO_LIST bad body len: %d", len(body))
	}
	skills := make([]SkillInfo, 0, len(body)/skillInfoEntryLen)
	for offset := 0; offset < len(body); offset += skillInfoEntryLen {
		skills = append(skills, parseSkillInfoEntry(body[offset:offset+skillInfoEntryLen], 0))
	}
	return SkillInfoList{Skills: skills}, true, nil
}

func ParseSkillInfoUpdate(packet Packet) (SkillInfoUpdate, bool, error) {
	switch packet.ID {
	case 0x010E:
		if len(packet.Data) < 11 {
			return SkillInfoUpdate{}, false, fmt.Errorf("ZC_SKILLINFO_UPDATE too short: %d", len(packet.Data))
		}
		return SkillInfoUpdate{Skill: SkillInfo{
			ID:         binary.LittleEndian.Uint16(packet.Data[2:4]),
			Level:      int(binary.LittleEndian.Uint16(packet.Data[4:6])),
			SPCost:     int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			Range:      int(binary.LittleEndian.Uint16(packet.Data[8:10])),
			Upgradable: packet.Data[10] != 0,
		}}, true, nil
	case 0x0111:
		if len(packet.Data) < 39 {
			return SkillInfoUpdate{}, false, fmt.Errorf("ZC_ADD_SKILL too short: %d", len(packet.Data))
		}
		return SkillInfoUpdate{Skill: parseSkillInfoEntry(packet.Data[2:39], 0)}, true, nil
	default:
		return SkillInfoUpdate{}, false, nil
	}
}

func ParseAutoRunSkill(packet Packet) (AutoRunSkill, bool, error) {
	if packet.ID != 0x0147 {
		return AutoRunSkill{}, false, nil
	}
	if len(packet.Data) < 39 {
		return AutoRunSkill{}, false, fmt.Errorf("ZC_AUTORUN_SKILL too short: %d", len(packet.Data))
	}
	return AutoRunSkill{Skill: parseSkillInfoEntry(packet.Data[2:39], 0)}, true, nil
}

func ParseMonsterInfo(packet Packet) (MonsterInfo, bool, error) {
	if packet.ID != PacketZCMonsterInfo {
		return MonsterInfo{}, false, nil
	}
	if len(packet.Data) < 29 {
		return MonsterInfo{}, true, fmt.Errorf("ZC_MONSTER_INFO too short: %d", len(packet.Data))
	}
	return MonsterInfo{
		Class:        binary.LittleEndian.Uint16(packet.Data[2:4]),
		Level:        binary.LittleEndian.Uint16(packet.Data[4:6]),
		Size:         binary.LittleEndian.Uint16(packet.Data[6:8]),
		HP:           binary.LittleEndian.Uint32(packet.Data[8:12]),
		Defense:      int16(binary.LittleEndian.Uint16(packet.Data[12:14])),
		Race:         binary.LittleEndian.Uint16(packet.Data[14:16]),
		MagicDefense: int16(binary.LittleEndian.Uint16(packet.Data[16:18])),
		Property:     binary.LittleEndian.Uint16(packet.Data[18:20]),
		Elements: MonsterElementRates{
			Water:  packet.Data[20],
			Earth:  packet.Data[21],
			Fire:   packet.Data[22],
			Wind:   packet.Data[23],
			Poison: packet.Data[24],
			Holy:   packet.Data[25],
			Shadow: packet.Data[26],
			Ghost:  packet.Data[27],
			Undead: packet.Data[28],
		},
	}, true, nil
}

func ParseAutoSpellList(packet Packet) (AutoSpellList, bool, error) {
	if packet.ID != PacketZCAutoSpellList {
		return AutoSpellList{}, false, nil
	}
	const packetLen = 2 + autoSpellChoiceCount*4
	if len(packet.Data) < packetLen {
		return AutoSpellList{}, true, fmt.Errorf("ZC_AUTOSPELLLIST too short: %d", len(packet.Data))
	}
	skillIDs := make([]uint16, 0, autoSpellChoiceCount)
	for i := 0; i < autoSpellChoiceCount; i++ {
		offset := 2 + i*4
		skillID := binary.LittleEndian.Uint32(packet.Data[offset : offset+4])
		if skillID == 0 {
			continue
		}
		if skillID > uint32(^uint16(0)) {
			return AutoSpellList{}, true, fmt.Errorf("ZC_AUTOSPELLLIST invalid skill id: %d", skillID)
		}
		skillIDs = append(skillIDs, uint16(skillID))
	}
	return AutoSpellList{SkillIDs: skillIDs}, true, nil
}

func ParseSkillNoDamageNotify(packet Packet) (SkillNoDamageNotify, bool, error) {
	switch packet.ID {
	case 0x011A:
		if len(packet.Data) < 15 {
			return SkillNoDamageNotify{}, false, fmt.Errorf("ZC_USE_SKILL too short: %d", len(packet.Data))
		}
		return SkillNoDamageNotify{
			SkillID:  binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount:   uint32(binary.LittleEndian.Uint16(packet.Data[4:6])),
			TargetID: binary.LittleEndian.Uint32(packet.Data[6:10]),
			SourceID: binary.LittleEndian.Uint32(packet.Data[10:14]),
			Result:   packet.Data[14],
		}, true, nil
	case 0x09CB:
		if len(packet.Data) < 17 {
			return SkillNoDamageNotify{}, false, fmt.Errorf("ZC_USE_SKILL2 too short: %d", len(packet.Data))
		}
		return SkillNoDamageNotify{
			SkillID:  binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount:   binary.LittleEndian.Uint32(packet.Data[4:8]),
			TargetID: binary.LittleEndian.Uint32(packet.Data[8:12]),
			SourceID: binary.LittleEndian.Uint32(packet.Data[12:16]),
			Result:   packet.Data[16],
		}, true, nil
	default:
		return SkillNoDamageNotify{}, false, nil
	}
}

func ParseSkillCastNotify(packet Packet) (SkillCastNotify, bool, error) {
	if packet.ID != 0x013E {
		return SkillCastNotify{}, false, nil
	}
	if len(packet.Data) < 24 {
		return SkillCastNotify{}, false, fmt.Errorf("ZC_USESKILL_ACK too short: %d", len(packet.Data))
	}
	return SkillCastNotify{
		SourceID:  binary.LittleEndian.Uint32(packet.Data[2:6]),
		TargetID:  binary.LittleEndian.Uint32(packet.Data[6:10]),
		X:         binary.LittleEndian.Uint16(packet.Data[10:12]),
		Y:         binary.LittleEndian.Uint16(packet.Data[12:14]),
		SkillID:   binary.LittleEndian.Uint16(packet.Data[14:16]),
		Property:  binary.LittleEndian.Uint32(packet.Data[16:20]),
		DelayTime: binary.LittleEndian.Uint32(packet.Data[20:24]),
	}, true, nil
}

func ParseSkillCastCancel(packet Packet) (uint32, bool, error) {
	if packet.ID != PacketZCSkillCastCancel {
		return 0, false, nil
	}
	if len(packet.Data) < 6 {
		return 0, false, fmt.Errorf("ZC_DISPEL too short: %d", len(packet.Data))
	}
	return binary.LittleEndian.Uint32(packet.Data[2:6]), true, nil
}

func ParseGroundSkillNotify(packet Packet) (GroundSkillNotify, bool, error) {
	if packet.ID != 0x0117 {
		return GroundSkillNotify{}, false, nil
	}
	if len(packet.Data) < 18 {
		return GroundSkillNotify{}, false, fmt.Errorf("ZC_SKILL_ENTRY too short: %d", len(packet.Data))
	}
	return GroundSkillNotify{
		SkillID:  binary.LittleEndian.Uint16(packet.Data[2:4]),
		SourceID: binary.LittleEndian.Uint32(packet.Data[4:8]),
		Level:    binary.LittleEndian.Uint16(packet.Data[8:10]),
		X:        binary.LittleEndian.Uint16(packet.Data[10:12]),
		Y:        binary.LittleEndian.Uint16(packet.Data[12:14]),
	}, true, nil
}

func ParseSkillUnitEntry(packet Packet) (SkillUnitEntry, bool, error) {
	if packet.ID != 0x011F && packet.ID != 0x01C9 {
		return SkillUnitEntry{}, false, nil
	}
	size := 16
	if packet.ID == 0x01C9 {
		size = 97
	}
	if len(packet.Data) < size {
		return SkillUnitEntry{}, false, fmt.Errorf("ZC_SKILL_ENTRY %#04x too short: %d", packet.ID, len(packet.Data))
	}
	entry := SkillUnitEntry{
		ID:        binary.LittleEndian.Uint32(packet.Data[2:6]),
		CreatorID: binary.LittleEndian.Uint32(packet.Data[6:10]),
		X:         binary.LittleEndian.Uint16(packet.Data[10:12]),
		Y:         binary.LittleEndian.Uint16(packet.Data[12:14]),
		UnitID:    uint16(packet.Data[14]),
		Visible:   packet.Data[15] != 0,
	}
	if packet.ID == 0x01C9 {
		entry.HasMessage = packet.Data[16] != 0
		message, _, _ := bytes.Cut(packet.Data[17:97], []byte{0})
		entry.Message = string(message)
	}
	return entry, true, nil
}

func ParseSkillUnitDisappear(packet Packet) (SkillUnitDisappear, bool, error) {
	if packet.ID != 0x0120 {
		return SkillUnitDisappear{}, false, nil
	}
	if len(packet.Data) < 6 {
		return SkillUnitDisappear{}, false, fmt.Errorf("ZC_SKILL_DISAPPEAR too short: %d", len(packet.Data))
	}
	return SkillUnitDisappear{ID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
}

func ParseSkillUnitUpdate(packet Packet) (SkillUnitUpdate, bool, error) {
	if packet.ID != 0x01AC {
		return SkillUnitUpdate{}, false, nil
	}
	if len(packet.Data) < 6 {
		return SkillUnitUpdate{}, false, fmt.Errorf("ZC_SKILL_UPDATE too short: %d", len(packet.Data))
	}
	return SkillUnitUpdate{ID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
}

func ParseSkillFailAck(packet Packet) (SkillFailAck, bool, error) {
	if packet.ID != 0x0110 {
		return SkillFailAck{}, false, nil
	}
	if len(packet.Data) < 10 {
		return SkillFailAck{}, false, fmt.Errorf("ZC_ACK_TOUSESKILL too short: %d", len(packet.Data))
	}
	return SkillFailAck{
		SkillID: binary.LittleEndian.Uint16(packet.Data[2:4]),
		Number:  uint32(binary.LittleEndian.Uint16(packet.Data[4:6])),
		ItemID:  uint32(binary.LittleEndian.Uint16(packet.Data[6:8])),
		Result:  packet.Data[8],
		Cause:   packet.Data[9],
	}, true, nil
}

func ParseWarpPointList(packet Packet) (WarpPointList, bool, error) {
	var skillOffset int
	var namesOffset int
	var packetLen int
	switch packet.ID {
	case 0x011C:
		if len(packet.Data) < 68 {
			return WarpPointList{}, false, fmt.Errorf("ZC_WARPLIST too short: %d", len(packet.Data))
		}
		skillOffset = 2
		namesOffset = 4
		packetLen = 68
	case 0x0ABE:
		if len(packet.Data) < 6 {
			return WarpPointList{}, false, fmt.Errorf("ZC_WARPLIST2 too short: %d", len(packet.Data))
		}
		packetLen = int(binary.LittleEndian.Uint16(packet.Data[2:4]))
		if packetLen <= 0 || packetLen > len(packet.Data) {
			packetLen = len(packet.Data)
		}
		skillOffset = 4
		namesOffset = 6
	default:
		return WarpPointList{}, false, nil
	}
	if packetLen < namesOffset || (packetLen-namesOffset)%16 != 0 {
		return WarpPointList{}, false, fmt.Errorf("ZC_WARPLIST bad body len: %d", packetLen-namesOffset)
	}
	names := make([]string, 0, (packetLen-namesOffset)/16)
	for offset := namesOffset; offset+16 <= packetLen; offset += 16 {
		name := normalizeMapName(fixedString(packet.Data[offset : offset+16]))
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return WarpPointList{
		SkillID:  binary.LittleEndian.Uint16(packet.Data[skillOffset : skillOffset+2]),
		MapNames: names,
	}, true, nil
}

func ParseRememberWarpPointAck(packet Packet) (RememberWarpPointAck, bool, error) {
	if packet.ID != 0x011E {
		return RememberWarpPointAck{}, false, nil
	}
	if len(packet.Data) < 3 {
		return RememberWarpPointAck{}, false, fmt.Errorf("ZC_ACK_REMEMBER_WARPPOINT too short: %d", len(packet.Data))
	}
	return RememberWarpPointAck{Result: packet.Data[2]}, true, nil
}

func BuildSkillLevelUpPacket(skillID uint16) []byte {
	packet := make([]byte, 4)
	binary.LittleEndian.PutUint16(packet[0:2], 0x0112)
	binary.LittleEndian.PutUint16(packet[2:4], skillID)
	return packet
}

func BuildRememberWarpPointPacket() []byte {
	packet := make([]byte, 2)
	binary.LittleEndian.PutUint16(packet[0:2], 0x011D)
	return packet
}

func BuildSelectAutoSpellPacket(skillID uint16) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZSelectAutoSpell)
	binary.LittleEndian.PutUint32(packet[2:6], uint32(skillID))
	return packet
}

func BuildUseSkillToIDPacketForClientDate(skillID, level uint16, targetID uint32, clientDate int) []byte {
	opcode := uint16(0x0113)
	if clientDate >= 20080910 {
		opcode = 0x0438
	}
	packet := make([]byte, 10)
	binary.LittleEndian.PutUint16(packet[0:2], opcode)
	binary.LittleEndian.PutUint16(packet[2:4], level)
	binary.LittleEndian.PutUint16(packet[4:6], skillID)
	binary.LittleEndian.PutUint32(packet[6:10], targetID)
	return packet
}

func BuildUseSkillToGroundPacketForClientDate(skillID, level uint16, x, y int, clientDate int) []byte {
	if clientDate >= 20120307 {
		packet := make([]byte, 10)
		binary.LittleEndian.PutUint16(packet[0:2], 0x0438)
		binary.LittleEndian.PutUint16(packet[2:4], level)
		binary.LittleEndian.PutUint16(packet[4:6], skillID)
		binary.LittleEndian.PutUint16(packet[6:8], uint16(x))
		binary.LittleEndian.PutUint16(packet[8:10], uint16(y))
		return packet
	}
	if clientDate >= 20080910 {
		packet := make([]byte, 22)
		binary.LittleEndian.PutUint16(packet[0:2], 0x0113)
		binary.LittleEndian.PutUint16(packet[5:7], level)
		binary.LittleEndian.PutUint16(packet[9:11], skillID)
		binary.LittleEndian.PutUint16(packet[12:14], uint16(x))
		binary.LittleEndian.PutUint16(packet[20:22], uint16(y))
		return packet
	}
	packet := make([]byte, 10)
	binary.LittleEndian.PutUint16(packet[0:2], 0x0116)
	binary.LittleEndian.PutUint16(packet[2:4], level)
	binary.LittleEndian.PutUint16(packet[4:6], skillID)
	binary.LittleEndian.PutUint16(packet[6:8], uint16(x))
	binary.LittleEndian.PutUint16(packet[8:10], uint16(y))
	return packet
}

func BuildUseSkillToGroundWithTextPacketForClientDate(skillID, level uint16, x, y int, text string, clientDate int) []byte {
	if clientDate >= 20120307 {
		packet := make([]byte, 10+skillGroundMessageLen)
		binary.LittleEndian.PutUint16(packet[0:2], 0x0367)
		binary.LittleEndian.PutUint16(packet[2:4], level)
		binary.LittleEndian.PutUint16(packet[4:6], skillID)
		binary.LittleEndian.PutUint16(packet[6:8], uint16(x))
		binary.LittleEndian.PutUint16(packet[8:10], uint16(y))
		writeSkillGroundMessage(packet[10:10+skillGroundMessageLen], text)
		return packet
	}
	if clientDate >= 20080910 {
		packet := make([]byte, 102)
		binary.LittleEndian.PutUint16(packet[0:2], 0x007E)
		binary.LittleEndian.PutUint16(packet[5:7], level)
		binary.LittleEndian.PutUint16(packet[9:11], skillID)
		binary.LittleEndian.PutUint16(packet[12:14], uint16(x))
		binary.LittleEndian.PutUint16(packet[20:22], uint16(y))
		writeSkillGroundMessage(packet[22:22+skillGroundMessageLen], text)
		return packet
	}
	packet := make([]byte, 10+skillGroundMessageLen)
	binary.LittleEndian.PutUint16(packet[0:2], 0x0190)
	binary.LittleEndian.PutUint16(packet[2:4], level)
	binary.LittleEndian.PutUint16(packet[4:6], skillID)
	binary.LittleEndian.PutUint16(packet[6:8], uint16(x))
	binary.LittleEndian.PutUint16(packet[8:10], uint16(y))
	writeSkillGroundMessage(packet[10:10+skillGroundMessageLen], text)
	return packet
}

func writeSkillGroundMessage(dst []byte, text string) {
	if len(dst) == 0 {
		return
	}
	src := []byte(text)
	if len(src) >= len(dst) {
		src = src[:len(dst)-1]
	}
	copy(dst, src)
	dst[len(src)] = 0
}

func BuildChangeCartPacket(cartNum uint16) []byte {
	packet := make([]byte, 4)
	binary.LittleEndian.PutUint16(packet[0:2], 0x01AF)
	binary.LittleEndian.PutUint16(packet[2:4], cartNum)
	return packet
}

func BuildSelectWarpPointPacket(skillID uint16, mapName string) []byte {
	packet := make([]byte, 20)
	binary.LittleEndian.PutUint16(packet[0:2], 0x011B)
	binary.LittleEndian.PutUint16(packet[2:4], skillID)
	copy(packet[4:20], []byte(mapName))
	return packet
}

func parseSkillInfoEntry(data []byte, offset int) SkillInfo {
	return SkillInfo{
		ID:         binary.LittleEndian.Uint16(data[offset : offset+2]),
		Type:       binary.LittleEndian.Uint32(data[offset+2 : offset+6]),
		Level:      int(binary.LittleEndian.Uint16(data[offset+6 : offset+8])),
		SPCost:     int(binary.LittleEndian.Uint16(data[offset+8 : offset+10])),
		Range:      int(binary.LittleEndian.Uint16(data[offset+10 : offset+12])),
		Name:       decodeROFixedString(data[offset+12 : offset+36]),
		Upgradable: data[offset+36] != 0,
	}
}

func decodeROFixedString(data []byte) string {
	if end := bytes.IndexByte(data, 0); end >= 0 {
		data = data[:end]
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return ""
	}
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), data)
	if err != nil {
		return string(data)
	}
	return strings.TrimSpace(string(decoded))
}
