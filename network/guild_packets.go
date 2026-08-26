package network

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/kivutar/goro/glog"
)

const (
	PacketCZReqMakeGuild         uint16 = 0x0165
	PacketZCResultMakeGuild      uint16 = 0x0167
	PacketCZReqJoinGuild         uint16 = 0x0168
	PacketZCAckReqJoinGuild      uint16 = 0x0169
	PacketZCReqJoinGuild         uint16 = 0x016A
	PacketCZJoinGuild            uint16 = 0x016B
	PacketCZGuildNotice          uint16 = 0x016E
	PacketZCGuildInfo            uint16 = 0x0150
	PacketZCGuildInfo2           uint16 = 0x01B6
	PacketZCGuildMembers         uint16 = 0x0154
	PacketCZReqChangeMember      uint16 = 0x0155
	PacketZCAckChangeMember      uint16 = 0x0156
	PacketCZReqOpenMember        uint16 = 0x0157
	PacketZCAckOpenMember        uint16 = 0x0158
	PacketZCGuildPositions       uint16 = 0x0160
	PacketCZRegGuildPosInfo      uint16 = 0x0161
	PacketZCGuildSkillInfo       uint16 = 0x0162
	PacketZCGuildBanList         uint16 = 0x0163
	PacketZCGuildPosNames        uint16 = 0x0166
	PacketZCGuildNotice          uint16 = 0x016F
	PacketZCAckGuildPosInfo      uint16 = 0x0174
	PacketCZReqGuildMember       uint16 = 0x0175
	PacketZCGuildMemberInfo      uint16 = 0x0176
	PacketCZGuildMessage         uint16 = 0x017E
	PacketZCUpdateGuildID        uint16 = 0x016C
	PacketCZReqGuildMenu         uint16 = 0x014F
	PacketCZReqGuildEmblem       uint16 = 0x0151
	PacketZCGuildEmblem          uint16 = 0x0152
	PacketCZRegGuildEmblem       uint16 = 0x0153
	PacketZCChangeGuild          uint16 = 0x01B4
	PacketZCGuildRelations       uint16 = 0x014C
	PacketZCGuildMemberState     uint16 = 0x016D
	PacketZCGuildMemberState2    uint16 = 0x01F2
	PacketCZReqGuildAlliance     uint16 = 0x0170
	PacketZCGuildAllianceRequest uint16 = 0x0171
	PacketCZGuildAllianceReply   uint16 = 0x0172
	PacketZCGuildAllianceResult  uint16 = 0x0173
	PacketCZReqGuildHostility    uint16 = 0x0180
	PacketZCGuildHostilityResult uint16 = 0x0181
	PacketCZDeleteGuildRelation  uint16 = 0x0183
	PacketZCGuildRelationDeleted uint16 = 0x0184
	PacketZCGuildRelationAdded   uint16 = 0x0185
	PacketZCGuildMemberLocation  uint16 = 0x01EB
)

const (
	guildNameLength          = 24
	guildNoticeHeaderLength  = 6
	guildNoticeSubjectLength = 60
	guildNoticeBodyLength    = 120
)

type GuildCreationResult struct {
	Result uint8
}

type GuildInviteAck struct {
	Result uint8
}

type GuildInviteRequest struct {
	GuildID   uint32
	GuildName string
}

type GuildBelonging struct {
	GuildID       uint32
	EmblemVersion uint32
	Mode          uint32
	IsMaster      bool
	GuildName     string
}

type GuildInfo struct {
	GuildID          uint32
	Level            uint32
	UserNum          uint32
	MaxUserNum       uint32
	UserAverageLevel uint32
	Exp              uint32
	MaxExp           uint32
	Point            uint32
	Honor            uint32
	Virtue           uint32
	EmblemVersion    uint32
	GuildName        string
	MasterName       string
	ManageLand       string
	Zeny             uint32
}

type GuildMember struct {
	AccountID    uint32
	CharID       uint32
	HeadType     uint16
	HeadPalette  uint16
	Sex          uint16
	Job          uint16
	Level        uint16
	MemberExp    uint32
	CurrentState uint32
	PositionID   uint32
	Memo         string
	CharName     string
}

type GuildMemberPosition struct {
	AccountID  uint32
	CharID     uint32
	PositionID uint32
}

type GuildPosition struct {
	PositionID uint32
	Right      uint32
	Ranking    uint32
	PayRate    uint32
	PosName    string
}

type GuildSkillInfo struct {
	SkillPoints int
	Skills      []SkillInfo
}

type GuildExpelHistory struct {
	CharName string
	Account  string
	Reason   string
}

type GuildNotice struct {
	Subject string
	Notice  string
}

type GuildEmblemImage struct {
	GuildID       uint32
	EmblemVersion uint32
	Data          []byte
}

type GuildEmblemChange struct {
	ActorID       uint32
	GuildID       uint32
	EmblemVersion uint32
}

type GuildRelation struct {
	Relation uint32
	GuildID  uint32
	Name     string
}

type GuildAllianceRequest struct {
	AccountID uint32
	GuildName string
}

type GuildAllianceResult struct {
	Result uint8
}

type GuildHostilityResult struct {
	Result uint8
}

type GuildRelationDeleted struct {
	GuildID  uint32
	Relation uint32
}

type GuildMemberState struct {
	AccountID     uint32
	CharID        uint32
	State         uint32
	HasAppearance bool
	Sex           uint16
	HeadType      uint16
	HeadPalette   uint16
}

type GuildMemberLocation struct {
	AccountID uint32
	X         int16
	Y         int16
}

func ParseGuildRelations(packet Packet) ([]GuildRelation, bool, error) {
	if packet.ID != PacketZCGuildRelations {
		return nil, false, nil
	}
	const entrySize = 32
	if len(packet.Data) < 4 {
		return nil, true, fmt.Errorf("ZC_GUILD_ALLIANCE_LIST too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, true, fmt.Errorf("ZC_GUILD_ALLIANCE_LIST invalid length: %d", len(packet.Data))
	}
	body := packet.Data[4:]
	relations := make([]GuildRelation, 0, len(body)/entrySize)
	for offset := 0; offset < len(body); offset += entrySize {
		entry := body[offset : offset+entrySize]
		relations = append(relations, GuildRelation{
			Relation: binary.LittleEndian.Uint32(entry[0:4]),
			GuildID:  binary.LittleEndian.Uint32(entry[4:8]),
			Name:     decodeROFixedString(entry[8:32]),
		})
	}
	return relations, true, nil
}

func ParseGuildAllianceRequest(packet Packet) (GuildAllianceRequest, bool, error) {
	if packet.ID != PacketZCGuildAllianceRequest {
		return GuildAllianceRequest{}, false, nil
	}
	if len(packet.Data) < 30 {
		return GuildAllianceRequest{}, true, fmt.Errorf("ZC_REQ_ALLY_GUILD too short: %d", len(packet.Data))
	}
	return GuildAllianceRequest{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		GuildName: decodeROFixedString(packet.Data[6:30]),
	}, true, nil
}

func ParseGuildAllianceResult(packet Packet) (GuildAllianceResult, bool, error) {
	if packet.ID != PacketZCGuildAllianceResult {
		return GuildAllianceResult{}, false, nil
	}
	if len(packet.Data) < 3 {
		return GuildAllianceResult{}, true, fmt.Errorf("ZC_ACK_REQ_ALLY_GUILD too short: %d", len(packet.Data))
	}
	return GuildAllianceResult{Result: packet.Data[2]}, true, nil
}

func ParseGuildHostilityResult(packet Packet) (GuildHostilityResult, bool, error) {
	if packet.ID != PacketZCGuildHostilityResult {
		return GuildHostilityResult{}, false, nil
	}
	if len(packet.Data) < 3 {
		return GuildHostilityResult{}, true, fmt.Errorf("ZC_ACK_REQ_HOSTILE_GUILD too short: %d", len(packet.Data))
	}
	return GuildHostilityResult{Result: packet.Data[2]}, true, nil
}

func ParseGuildRelationDeleted(packet Packet) (GuildRelationDeleted, bool, error) {
	if packet.ID != PacketZCGuildRelationDeleted {
		return GuildRelationDeleted{}, false, nil
	}
	if len(packet.Data) < 10 {
		return GuildRelationDeleted{}, true, fmt.Errorf("ZC_DELETE_RELATED_GUILD too short: %d", len(packet.Data))
	}
	return GuildRelationDeleted{
		GuildID:  binary.LittleEndian.Uint32(packet.Data[2:6]),
		Relation: binary.LittleEndian.Uint32(packet.Data[6:10]),
	}, true, nil
}

func ParseGuildRelationAdded(packet Packet) (GuildRelation, bool, error) {
	if packet.ID != PacketZCGuildRelationAdded {
		return GuildRelation{}, false, nil
	}
	if len(packet.Data) < 34 {
		return GuildRelation{}, true, fmt.Errorf("ZC_ADD_RELATED_GUILD too short: %d", len(packet.Data))
	}
	return GuildRelation{
		Relation: binary.LittleEndian.Uint32(packet.Data[2:6]),
		GuildID:  binary.LittleEndian.Uint32(packet.Data[6:10]),
		Name:     decodeROFixedString(packet.Data[10:34]),
	}, true, nil
}

func ParseGuildMemberState(packet Packet) (GuildMemberState, bool, error) {
	if packet.ID != PacketZCGuildMemberState && packet.ID != PacketZCGuildMemberState2 {
		return GuildMemberState{}, false, nil
	}
	minimumLength := 14
	if packet.ID == PacketZCGuildMemberState2 {
		minimumLength = 20
	}
	if len(packet.Data) < minimumLength {
		return GuildMemberState{}, true, fmt.Errorf("ZC_GUILD_MEMBER_STATE too short: %d", len(packet.Data))
	}
	state := GuildMemberState{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		CharID:    binary.LittleEndian.Uint32(packet.Data[6:10]),
		State:     binary.LittleEndian.Uint32(packet.Data[10:14]),
	}
	if packet.ID == PacketZCGuildMemberState2 {
		state.HasAppearance = true
		state.Sex = binary.LittleEndian.Uint16(packet.Data[14:16])
		state.HeadType = binary.LittleEndian.Uint16(packet.Data[16:18])
		state.HeadPalette = binary.LittleEndian.Uint16(packet.Data[18:20])
	}
	return state, true, nil
}

func ParseGuildMemberLocation(packet Packet) (GuildMemberLocation, bool, error) {
	if packet.ID != PacketZCGuildMemberLocation {
		return GuildMemberLocation{}, false, nil
	}
	if len(packet.Data) < 10 {
		return GuildMemberLocation{}, true, fmt.Errorf("ZC_NOTIFY_POSITION_TO_GUILDM too short: %d", len(packet.Data))
	}
	return GuildMemberLocation{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		X:         int16(binary.LittleEndian.Uint16(packet.Data[6:8])),
		Y:         int16(binary.LittleEndian.Uint16(packet.Data[8:10])),
	}, true, nil
}

func ParseGuildNotice(packet Packet) (GuildNotice, bool, error) {
	if packet.ID != PacketZCGuildNotice {
		return GuildNotice{}, false, nil
	}
	if len(packet.Data) < 182 {
		return GuildNotice{}, true, fmt.Errorf("ZC_GUILD_NOTICE too short: %d", len(packet.Data))
	}
	return GuildNotice{
		Subject: decodeROFixedString(packet.Data[2:62]),
		Notice:  decodeROFixedString(packet.Data[62:182]),
	}, true, nil
}

func ParseGuildExpelHistory(packet Packet) ([]GuildExpelHistory, bool, error) {
	if packet.ID != PacketZCGuildBanList {
		return nil, false, nil
	}
	const entrySize = 88
	if len(packet.Data) < 4 {
		return nil, true, fmt.Errorf("ZC_BAN_LIST too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, true, fmt.Errorf("ZC_BAN_LIST invalid length: %d", len(packet.Data))
	}
	body := packet.Data[4:]
	history := make([]GuildExpelHistory, 0, len(body)/entrySize)
	for offset := 0; offset < len(body); offset += entrySize {
		entry := body[offset : offset+entrySize]
		history = append(history, GuildExpelHistory{
			CharName: decodeROFixedString(entry[0:24]),
			Account:  decodeROFixedString(entry[24:48]),
			Reason:   decodeROFixedString(entry[48:88]),
		})
	}
	return history, true, nil
}

func ParseGuildSkillInfo(packet Packet) (GuildSkillInfo, bool, error) {
	if packet.ID != PacketZCGuildSkillInfo {
		return GuildSkillInfo{}, false, nil
	}
	if len(packet.Data) < 6 {
		return GuildSkillInfo{}, true, fmt.Errorf("ZC_GUILD_SKILLINFO too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-6)%skillInfoEntryLen != 0 {
		return GuildSkillInfo{}, true, fmt.Errorf("ZC_GUILD_SKILLINFO invalid length: %d", len(packet.Data))
	}
	body := packet.Data[6:]
	skills := make([]SkillInfo, 0, len(body)/skillInfoEntryLen)
	for offset := 0; offset < len(body); offset += skillInfoEntryLen {
		skills = append(skills, parseSkillInfoEntry(body[offset:offset+skillInfoEntryLen], 0))
	}
	return GuildSkillInfo{
		SkillPoints: int(binary.LittleEndian.Uint16(packet.Data[4:6])),
		Skills:      skills,
	}, true, nil
}

func ParseGuildPositions(packet Packet) ([]GuildPosition, bool, error) {
	switch packet.ID {
	case PacketZCGuildPositions:
	case PacketZCAckGuildPosInfo:
		return parseGuildPositionInfoAck(packet)
	default:
		return nil, false, nil
	}
	const entrySize = 16
	if len(packet.Data) < 4 {
		return nil, true, fmt.Errorf("ZC_POSITION_INFO too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, true, fmt.Errorf("ZC_POSITION_INFO invalid length: %d", len(packet.Data))
	}
	body := packet.Data[4:]
	positions := make([]GuildPosition, 0, len(body)/entrySize)
	for offset := 0; offset < len(body); offset += entrySize {
		entry := body[offset : offset+entrySize]
		positions = append(positions, GuildPosition{
			PositionID: binary.LittleEndian.Uint32(entry[0:4]),
			Right:      binary.LittleEndian.Uint32(entry[4:8]),
			Ranking:    binary.LittleEndian.Uint32(entry[8:12]),
			PayRate:    binary.LittleEndian.Uint32(entry[12:16]),
		})
	}
	return positions, true, nil
}

func parseGuildPositionInfoAck(packet Packet) ([]GuildPosition, bool, error) {
	const entrySize = 40
	if len(packet.Data) < 4 {
		return nil, true, fmt.Errorf("ZC_ACK_CHANGE_GUILD_POSITIONINFO too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, true, fmt.Errorf("ZC_ACK_CHANGE_GUILD_POSITIONINFO invalid length: %d", len(packet.Data))
	}
	body := packet.Data[4:]
	positions := make([]GuildPosition, 0, len(body)/entrySize)
	for offset := 0; offset < len(body); offset += entrySize {
		entry := body[offset : offset+entrySize]
		positions = append(positions, GuildPosition{
			PositionID: binary.LittleEndian.Uint32(entry[0:4]),
			Right:      binary.LittleEndian.Uint32(entry[4:8]),
			Ranking:    binary.LittleEndian.Uint32(entry[8:12]),
			PayRate:    binary.LittleEndian.Uint32(entry[12:16]),
			PosName:    decodeROFixedString(entry[16:40]),
		})
	}
	return positions, true, nil
}

func ParseGuildPositionNames(packet Packet) ([]GuildPosition, bool, error) {
	if packet.ID != PacketZCGuildPosNames {
		return nil, false, nil
	}
	const entrySize = 28
	if len(packet.Data) < 4 {
		return nil, true, fmt.Errorf("ZC_POSITION_ID_NAME_INFO too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, true, fmt.Errorf("ZC_POSITION_ID_NAME_INFO invalid length: %d", len(packet.Data))
	}
	body := packet.Data[4:]
	positions := make([]GuildPosition, 0, len(body)/entrySize)
	for offset := 0; offset < len(body); offset += entrySize {
		entry := body[offset : offset+entrySize]
		positions = append(positions, GuildPosition{
			PositionID: binary.LittleEndian.Uint32(entry[0:4]),
			PosName:    decodeROFixedString(entry[4:28]),
		})
	}
	return positions, true, nil
}

func ParseGuildMembers(packet Packet) ([]GuildMember, bool, error) {
	if packet.ID != PacketZCGuildMembers {
		return nil, false, nil
	}
	const entrySize = 104
	if len(packet.Data) < 4 {
		return nil, true, fmt.Errorf("ZC_MEMBERMGR_INFO too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, true, fmt.Errorf("ZC_MEMBERMGR_INFO invalid length: %d", len(packet.Data))
	}
	body := packet.Data[4:]
	members := make([]GuildMember, 0, len(body)/entrySize)
	for offset := 0; offset < len(body); offset += entrySize {
		members = append(members, parseGuildMemberEntry(body[offset:offset+entrySize]))
	}
	return members, true, nil
}

func ParseGuildMemberInfo(packet Packet) (GuildMember, bool, error) {
	if packet.ID != PacketZCGuildMemberInfo {
		return GuildMember{}, false, nil
	}
	if len(packet.Data) < 106 {
		return GuildMember{}, true, fmt.Errorf("ZC_ACK_GUILD_MEMBER_INFO too short: %d", len(packet.Data))
	}
	member := parseGuildMemberEntry(packet.Data[2:106])
	return member, true, nil
}

func ParseGuildMemberPositions(packet Packet) ([]GuildMemberPosition, bool, error) {
	if packet.ID != PacketZCAckChangeMember {
		return nil, false, nil
	}
	const entrySize = 12
	if len(packet.Data) < 4 {
		return nil, true, fmt.Errorf("ZC_ACK_REQ_CHANGE_MEMBERS too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, true, fmt.Errorf("ZC_ACK_REQ_CHANGE_MEMBERS invalid length: %d", len(packet.Data))
	}
	body := packet.Data[4:]
	positions := make([]GuildMemberPosition, 0, len(body)/entrySize)
	for offset := 0; offset < len(body); offset += entrySize {
		entry := body[offset : offset+entrySize]
		positions = append(positions, GuildMemberPosition{
			AccountID:  binary.LittleEndian.Uint32(entry[0:4]),
			CharID:     binary.LittleEndian.Uint32(entry[4:8]),
			PositionID: binary.LittleEndian.Uint32(entry[8:12]),
		})
	}
	return positions, true, nil
}

func parseGuildMemberEntry(entry []byte) GuildMember {
	return GuildMember{
		AccountID:    binary.LittleEndian.Uint32(entry[0:4]),
		CharID:       binary.LittleEndian.Uint32(entry[4:8]),
		HeadType:     binary.LittleEndian.Uint16(entry[8:10]),
		HeadPalette:  binary.LittleEndian.Uint16(entry[10:12]),
		Sex:          binary.LittleEndian.Uint16(entry[12:14]),
		Job:          binary.LittleEndian.Uint16(entry[14:16]),
		Level:        binary.LittleEndian.Uint16(entry[16:18]),
		MemberExp:    binary.LittleEndian.Uint32(entry[18:22]),
		CurrentState: binary.LittleEndian.Uint32(entry[22:26]),
		PositionID:   binary.LittleEndian.Uint32(entry[26:30]),
		Memo:         decodeROFixedString(entry[30:80]),
		CharName:     decodeROFixedString(entry[80:104]),
	}
}

func ParseGuildInfo(packet Packet) (GuildInfo, bool, error) {
	switch packet.ID {
	case PacketZCGuildInfo, PacketZCGuildInfo2:
	default:
		return GuildInfo{}, false, nil
	}
	minLen := 110
	if packet.ID == PacketZCGuildInfo2 {
		minLen = 114
	}
	if len(packet.Data) < minLen {
		return GuildInfo{}, true, fmt.Errorf("ZC_GUILD_INFO 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	emblemVersion := binary.LittleEndian.Uint32(packet.Data[42:46])
	var zeny uint32
	if packet.ID == PacketZCGuildInfo2 {
		zeny = binary.LittleEndian.Uint32(packet.Data[110:114])
	}
	return GuildInfo{
		GuildID:          binary.LittleEndian.Uint32(packet.Data[2:6]),
		Level:            binary.LittleEndian.Uint32(packet.Data[6:10]),
		UserNum:          binary.LittleEndian.Uint32(packet.Data[10:14]),
		MaxUserNum:       binary.LittleEndian.Uint32(packet.Data[14:18]),
		UserAverageLevel: binary.LittleEndian.Uint32(packet.Data[18:22]),
		Exp:              binary.LittleEndian.Uint32(packet.Data[22:26]),
		MaxExp:           binary.LittleEndian.Uint32(packet.Data[26:30]),
		Point:            binary.LittleEndian.Uint32(packet.Data[30:34]),
		Honor:            binary.LittleEndian.Uint32(packet.Data[34:38]),
		Virtue:           binary.LittleEndian.Uint32(packet.Data[38:42]),
		EmblemVersion:    emblemVersion,
		GuildName:        decodeROFixedString(packet.Data[46:70]),
		MasterName:       decodeROFixedString(packet.Data[70:94]),
		ManageLand:       decodeROFixedString(packet.Data[94:110]),
		Zeny:             zeny,
	}, true, nil
}

func ParseGuildBelonging(packet Packet) (GuildBelonging, bool, error) {
	if packet.ID != PacketZCUpdateGuildID {
		return GuildBelonging{}, false, nil
	}
	if len(packet.Data) < 43 {
		return GuildBelonging{}, true, fmt.Errorf("ZC_UPDATE_GDID too short: %d", len(packet.Data))
	}
	return GuildBelonging{
		GuildID:       binary.LittleEndian.Uint32(packet.Data[2:6]),
		EmblemVersion: binary.LittleEndian.Uint32(packet.Data[6:10]),
		Mode:          binary.LittleEndian.Uint32(packet.Data[10:14]),
		IsMaster:      packet.Data[14] != 0,
		GuildName:     decodeROFixedString(packet.Data[19:43]),
	}, true, nil
}

func ParseGuildCreationResult(packet Packet) (GuildCreationResult, bool, error) {
	if packet.ID != PacketZCResultMakeGuild {
		return GuildCreationResult{}, false, nil
	}
	if len(packet.Data) < 3 {
		return GuildCreationResult{}, true, fmt.Errorf("ZC_RESULT_MAKE_GUILD too short: %d", len(packet.Data))
	}
	return GuildCreationResult{Result: packet.Data[2]}, true, nil
}

func ParseGuildInviteAck(packet Packet) (GuildInviteAck, bool, error) {
	if packet.ID != PacketZCAckReqJoinGuild {
		return GuildInviteAck{}, false, nil
	}
	if len(packet.Data) < 3 {
		return GuildInviteAck{}, true, fmt.Errorf("ZC_ACK_REQ_JOIN_GUILD too short: %d", len(packet.Data))
	}
	return GuildInviteAck{Result: packet.Data[2]}, true, nil
}

func ParseGuildInviteRequest(packet Packet) (GuildInviteRequest, bool, error) {
	if packet.ID != PacketZCReqJoinGuild {
		return GuildInviteRequest{}, false, nil
	}
	if len(packet.Data) < 30 {
		return GuildInviteRequest{}, true, fmt.Errorf("ZC_REQ_JOIN_GUILD too short: %d", len(packet.Data))
	}
	return GuildInviteRequest{
		GuildID:   binary.LittleEndian.Uint32(packet.Data[2:6]),
		GuildName: decodeROFixedString(packet.Data[6:30]),
	}, true, nil
}

func ParseGuildEmblemImage(packet Packet) (GuildEmblemImage, bool, error) {
	if packet.ID != PacketZCGuildEmblem {
		return GuildEmblemImage{}, false, nil
	}
	if len(packet.Data) < 12 {
		return GuildEmblemImage{}, true, fmt.Errorf("ZC_GUILD_EMBLEM_IMG too short: %d", len(packet.Data))
	}
	return GuildEmblemImage{
		GuildID:       binary.LittleEndian.Uint32(packet.Data[4:8]),
		EmblemVersion: binary.LittleEndian.Uint32(packet.Data[8:12]),
		Data:          append([]byte(nil), packet.Data[12:]...),
	}, true, nil
}

func ParseGuildEmblemChange(packet Packet) (GuildEmblemChange, bool, error) {
	if packet.ID != PacketZCChangeGuild {
		return GuildEmblemChange{}, false, nil
	}
	if len(packet.Data) < 12 {
		return GuildEmblemChange{}, true, fmt.Errorf("ZC_CHANGE_GUILD too short: %d", len(packet.Data))
	}
	return GuildEmblemChange{
		ActorID:       binary.LittleEndian.Uint32(packet.Data[2:6]),
		GuildID:       binary.LittleEndian.Uint32(packet.Data[6:10]),
		EmblemVersion: uint32(binary.LittleEndian.Uint16(packet.Data[10:12])),
	}, true, nil
}

func BuildCreateGuildPacket(charID uint32, name string) []byte {
	packet := make([]byte, 30)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqMakeGuild)
	binary.LittleEndian.PutUint32(packet[2:6], charID)
	copy(packet[6:30], encodeROFixedString(name, guildNameLength))
	return packet
}

func BuildRequestGuildInvitePacket(targetAID, inviterAID, inviterCharID uint32) []byte {
	packet := make([]byte, 14)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqJoinGuild)
	binary.LittleEndian.PutUint32(packet[2:6], targetAID)
	binary.LittleEndian.PutUint32(packet[6:10], inviterAID)
	binary.LittleEndian.PutUint32(packet[10:14], inviterCharID)
	return packet
}

func BuildGuildInviteReplyPacket(guildID uint32, accept bool) []byte {
	packet := make([]byte, 10)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZJoinGuild)
	binary.LittleEndian.PutUint32(packet[2:6], guildID)
	if accept {
		binary.LittleEndian.PutUint32(packet[6:10], 1)
	}
	return packet
}

func BuildGuildAllianceRequestPacket(targetAID, accountID, charID uint32) []byte {
	packet := make([]byte, 14)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqGuildAlliance)
	binary.LittleEndian.PutUint32(packet[2:6], targetAID)
	binary.LittleEndian.PutUint32(packet[6:10], accountID)
	binary.LittleEndian.PutUint32(packet[10:14], charID)
	return packet
}

func BuildGuildAllianceReplyPacket(accountID uint32, accept bool) []byte {
	packet := make([]byte, 10)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZGuildAllianceReply)
	binary.LittleEndian.PutUint32(packet[2:6], accountID)
	if accept {
		binary.LittleEndian.PutUint32(packet[6:10], 1)
	}
	return packet
}

func BuildGuildHostilityRequestPacket(targetAID uint32) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqGuildHostility)
	binary.LittleEndian.PutUint32(packet[2:6], targetAID)
	return packet
}

func BuildDeleteGuildRelationPacket(guildID, relation uint32) []byte {
	packet := make([]byte, 10)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZDeleteGuildRelation)
	binary.LittleEndian.PutUint32(packet[2:6], guildID)
	binary.LittleEndian.PutUint32(packet[6:10], relation)
	return packet
}

func BuildChangeGuildMemberPositionPacket(members []GuildMemberPosition) []byte {
	packet := make([]byte, 4+len(members)*12)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqChangeMember)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(len(packet)))
	for i, member := range members {
		offset := 4 + i*12
		binary.LittleEndian.PutUint32(packet[offset:offset+4], member.AccountID)
		binary.LittleEndian.PutUint32(packet[offset+4:offset+8], member.CharID)
		binary.LittleEndian.PutUint32(packet[offset+8:offset+12], member.PositionID)
	}
	return packet
}

func BuildRegisterGuildPositionsPacket(positions []GuildPosition) []byte {
	packet := make([]byte, 4+len(positions)*40)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZRegGuildPosInfo)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(len(packet)))
	for i, position := range positions {
		offset := 4 + i*40
		binary.LittleEndian.PutUint32(packet[offset:offset+4], position.PositionID)
		binary.LittleEndian.PutUint32(packet[offset+4:offset+8], position.Right)
		binary.LittleEndian.PutUint32(packet[offset+8:offset+12], position.Ranking)
		binary.LittleEndian.PutUint32(packet[offset+12:offset+16], position.PayRate)
		copy(packet[offset+16:offset+40], encodeROFixedString(position.PosName, guildNameLength))
	}
	return packet
}

func BuildGuildEmblemRequestPacket(guildID uint32) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqGuildEmblem)
	binary.LittleEndian.PutUint32(packet[2:6], guildID)
	return packet
}

func BuildGuildMenuRequestPacket(tab uint32) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqGuildMenu)
	binary.LittleEndian.PutUint32(packet[2:6], tab)
	return packet
}

func BuildGuildNoticePacket(guildID uint32, subject, notice string) []byte {
	packet := make([]byte, guildNoticeHeaderLength+guildNoticeSubjectLength+guildNoticeBodyLength)
	subjectOffset := guildNoticeHeaderLength
	noticeOffset := subjectOffset + guildNoticeSubjectLength
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZGuildNotice)
	binary.LittleEndian.PutUint32(packet[2:6], guildID)
	copy(packet[subjectOffset:noticeOffset], encodeROFixedString(subject, guildNoticeSubjectLength))
	copy(packet[noticeOffset:], encodeROFixedString(notice, guildNoticeBodyLength))
	return packet
}

func BuildRegisterGuildEmblemPacket(bmp []byte) ([]byte, error) {
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(bmp); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	packetLen := 4 + compressed.Len()
	if packetLen > 0xFFFF {
		return nil, fmt.Errorf("guild emblem packet too large: %d", packetLen)
	}
	packet := make([]byte, packetLen)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZRegGuildEmblem)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(len(packet)))
	copy(packet[4:], compressed.Bytes())
	return packet, nil
}

func BuildGuildMessagePacket(message string) []byte {
	message = strings.TrimSpace(message)
	size := 4 + len([]byte(message)) + 1
	if message == "" || size > 0xffff {
		return nil
	}
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZGuildMessage)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	copy(packet[4:], []byte(message))
	return packet
}

func (c *Client) SendCreateGuild(charID uint32, name string) error {
	packet := BuildCreateGuildPacket(charID, name)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_MAKE_GUILD opcode=0x%04X char_id=%d name=%q client_date=%d", ID(packet), charID, name, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_MAKE_GUILD failed opcode=0x%04X len=%d char_id=%d name=%q client_date=%d: %v", ID(packet), len(packet), charID, name, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildInvite(targetAID, inviterAID, inviterCharID uint32) error {
	packet := BuildRequestGuildInvitePacket(targetAID, inviterAID, inviterCharID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_JOIN_GUILD opcode=0x%04X target=%d inviter_aid=%d inviter_char=%d client_date=%d", ID(packet), targetAID, inviterAID, inviterCharID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_JOIN_GUILD failed opcode=0x%04X len=%d target=%d inviter_aid=%d inviter_char=%d client_date=%d: %v", ID(packet), len(packet), targetAID, inviterAID, inviterCharID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildInviteReply(guildID uint32, accept bool) error {
	packet := BuildGuildInviteReplyPacket(guildID, accept)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_JOIN_GUILD opcode=0x%04X guild_id=%d accept=%t client_date=%d", ID(packet), guildID, accept, c.clientDate)
	} else {
		glog.Warnf("send CZ_JOIN_GUILD failed opcode=0x%04X len=%d guild_id=%d accept=%t client_date=%d: %v", ID(packet), len(packet), guildID, accept, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildAllianceRequest(targetAID, accountID, charID uint32) error {
	packet := BuildGuildAllianceRequestPacket(targetAID, accountID, charID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_ALLY_GUILD opcode=0x%04X target=%d account=%d char=%d client_date=%d", ID(packet), targetAID, accountID, charID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_ALLY_GUILD failed opcode=0x%04X target=%d client_date=%d: %v", ID(packet), targetAID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildAllianceReply(accountID uint32, accept bool) error {
	packet := BuildGuildAllianceReplyPacket(accountID, accept)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ALLY_GUILD opcode=0x%04X account=%d accept=%t client_date=%d", ID(packet), accountID, accept, c.clientDate)
	} else {
		glog.Warnf("send CZ_ALLY_GUILD failed opcode=0x%04X account=%d accept=%t client_date=%d: %v", ID(packet), accountID, accept, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildHostilityRequest(targetAID uint32) error {
	packet := BuildGuildHostilityRequestPacket(targetAID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_HOSTILE_GUILD opcode=0x%04X target=%d client_date=%d", ID(packet), targetAID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_HOSTILE_GUILD failed opcode=0x%04X target=%d client_date=%d: %v", ID(packet), targetAID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendDeleteGuildRelation(guildID, relation uint32) error {
	packet := BuildDeleteGuildRelationPacket(guildID, relation)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_DELETE_RELATED_GUILD opcode=0x%04X guild=%d relation=%d client_date=%d", ID(packet), guildID, relation, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_DELETE_RELATED_GUILD failed opcode=0x%04X guild=%d relation=%d client_date=%d: %v", ID(packet), guildID, relation, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildMemberPositions(members []GuildMemberPosition) error {
	packet := BuildChangeGuildMemberPositionPacket(members)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_CHANGE_MEMBERPOS opcode=0x%04X members=%d client_date=%d", ID(packet), len(members), c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_CHANGE_MEMBERPOS failed opcode=0x%04X len=%d members=%d client_date=%d: %v", ID(packet), len(packet), len(members), c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildPositions(positions []GuildPosition) error {
	packet := BuildRegisterGuildPositionsPacket(positions)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REG_CHANGE_GUILD_POSITIONINFO opcode=0x%04X positions=%d client_date=%d", ID(packet), len(positions), c.clientDate)
	} else {
		glog.Warnf("send CZ_REG_CHANGE_GUILD_POSITIONINFO failed opcode=0x%04X len=%d positions=%d client_date=%d: %v", ID(packet), len(packet), len(positions), c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildEmblemRequest(guildID uint32) error {
	packet := BuildGuildEmblemRequestPacket(guildID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_GUILD_EMBLEM_IMG opcode=0x%04X guild_id=%d client_date=%d", ID(packet), guildID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_GUILD_EMBLEM_IMG failed opcode=0x%04X len=%d guild_id=%d client_date=%d: %v", ID(packet), len(packet), guildID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildMenuRequest(tab uint32) error {
	packet := BuildGuildMenuRequestPacket(tab)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_GUILD_MENU opcode=0x%04X tab=%d client_date=%d", ID(packet), tab, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_GUILD_MENU failed opcode=0x%04X len=%d tab=%d client_date=%d: %v", ID(packet), len(packet), tab, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildNotice(guildID uint32, subject, notice string) error {
	packet := BuildGuildNoticePacket(guildID, subject, notice)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_GUILD_NOTICE opcode=0x%04X guild_id=%d subject=%q notice_len=%d client_date=%d", ID(packet), guildID, subject, len([]rune(notice)), c.clientDate)
	} else {
		glog.Warnf("send CZ_GUILD_NOTICE failed opcode=0x%04X len=%d guild_id=%d subject=%q notice_len=%d client_date=%d: %v", ID(packet), len(packet), guildID, subject, len([]rune(notice)), c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildMessage(message string) error {
	packet := BuildGuildMessagePacket(message)
	if len(packet) == 0 {
		return fmt.Errorf("empty guild message")
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQUEST_CHAT_GUILD opcode=0x%04X message=%q client_date=%d", ID(packet), message, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQUEST_CHAT_GUILD failed opcode=0x%04X len=%d message=%q client_date=%d: %v", ID(packet), len(packet), message, c.clientDate, err)
	}
	return err
}

func (c *Client) SendGuildEmblem(bmp []byte) error {
	packet, err := BuildRegisterGuildEmblemPacket(bmp)
	if err != nil {
		return err
	}
	err = c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REGISTER_GUILD_EMBLEM_IMG opcode=0x%04X bytes=%d client_date=%d", ID(packet), len(bmp), c.clientDate)
	} else {
		glog.Warnf("send CZ_REGISTER_GUILD_EMBLEM_IMG failed opcode=0x%04X len=%d bytes=%d client_date=%d: %v", ID(packet), len(packet), len(bmp), c.clientDate, err)
	}
	return err
}
