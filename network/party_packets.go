package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
	"strings"
)

const (
	PacketCZMakeGroup           uint16 = 0x00F9
	PacketCZReqJoinGroup        uint16 = 0x00FC
	PacketCZJoinGroup           uint16 = 0x00FF
	PacketCZReqLeaveGroup       uint16 = 0x0100
	PacketCZChangeGroupExp      uint16 = 0x0102
	PacketCZReqExpelGroupMember uint16 = 0x0103
	PacketCZPartyMessage        uint16 = 0x0108
	PacketCZMakeGroup2          uint16 = 0x01E8
	PacketCZPartyJoinReq        uint16 = 0x02C4
	PacketCZPartyJoinReqAck     uint16 = 0x02C7
	PacketCZPartyConfig         uint16 = 0x02C8

	PacketZCAckMakeGroup        uint16 = 0x00FA
	PacketZCGroupList           uint16 = 0x00FB
	PacketZCAckReqJoinGroup     uint16 = 0x00FD
	PacketZCReqJoinGroup        uint16 = 0x00FE
	PacketZCGroupInfoChange     uint16 = 0x0101
	PacketZCAddMemberToGroup    uint16 = 0x0104
	PacketZCAddMemberToGroup2   uint16 = 0x01E9
	PacketZCDeleteMemberFromGrp uint16 = 0x0105
	PacketZCNotifyHPToGroup     uint16 = 0x0106
	PacketZCNotifyPositionToGrp uint16 = 0x0107
	PacketZCNotifyChatParty     uint16 = 0x0109
	PacketZCPartyJoinReqAck     uint16 = 0x02C5
	PacketZCPartyJoinReq        uint16 = 0x02C6
	PacketZCPartyConfig         uint16 = 0x02C9
	PacketZCNotifyHPToGroupR2   uint16 = 0x080E
)

const partyNameLength = 24

type PartyMember struct {
	AccountID uint32
	GroupName string
	Name      string
	MapName   string
	Role      uint32
	State     uint8
	X         int
	Y         int
}

type PartyList struct {
	Name    string
	Members []PartyMember
}

type PartyCreateResult struct {
	Result uint8
}

type PartyInviteRequest struct {
	RequestID uint32
	Name      string
}

type PartyInviteAnswer struct {
	Name   string
	Answer uint8
}

type PartyOption struct {
	ExpOption uint32
}

type PartyInviteConfig struct {
	RefuseInvites bool
}

type PartyMemberLeave struct {
	AccountID uint32
	Name      string
	Result    uint8
}

type PartyMemberHP struct {
	AccountID uint32
	HP        int
	MaxHP     int
}

type PartyMemberPosition struct {
	AccountID uint32
	X         int
	Y         int
}

type PartyChat struct {
	AccountID uint32
	Message   string
}

func ParsePartyCreateResult(packet Packet) (PartyCreateResult, bool, error) {
	if packet.ID != PacketZCAckMakeGroup {
		return PartyCreateResult{}, false, nil
	}
	if len(packet.Data) < 3 {
		return PartyCreateResult{}, true, fmt.Errorf("ZC_ACK_MAKE_GROUP too short: %d", len(packet.Data))
	}
	return PartyCreateResult{Result: packet.Data[2]}, true, nil
}

func ParsePartyList(packet Packet) (PartyList, bool, error) {
	if packet.ID != PacketZCGroupList {
		return PartyList{}, false, nil
	}
	if len(packet.Data) < 28 {
		return PartyList{}, true, fmt.Errorf("ZC_GROUP_LIST too short: %d", len(packet.Data))
	}
	body := packet.Data[28:]
	if len(body)%46 != 0 {
		return PartyList{}, true, fmt.Errorf("ZC_GROUP_LIST bad member body len: %d", len(body))
	}
	list := PartyList{
		Name:    fixedPacketString(packet.Data[4:28]),
		Members: make([]PartyMember, 0, len(body)/46),
	}
	for len(body) > 0 {
		list.Members = append(list.Members, PartyMember{
			AccountID: binary.LittleEndian.Uint32(body[0:4]),
			Name:      fixedPacketString(body[4:28]),
			MapName:   fixedPacketString(body[28:44]),
			Role:      uint32(body[44]),
			State:     body[45],
		})
		body = body[46:]
	}
	return list, true, nil
}

func ParsePartyInviteAnswer(packet Packet) (PartyInviteAnswer, bool, error) {
	switch packet.ID {
	case PacketZCPartyJoinReqAck:
		if len(packet.Data) < 30 {
			return PartyInviteAnswer{}, true, fmt.Errorf("ZC_PARTY_JOIN_REQ_ACK too short: %d", len(packet.Data))
		}
		return PartyInviteAnswer{
			Name:   fixedPacketString(packet.Data[2:26]),
			Answer: uint8(binary.LittleEndian.Uint32(packet.Data[26:30])),
		}, true, nil
	default:
		return PartyInviteAnswer{}, false, nil
	}
}

func ParsePartyInviteRequest(packet Packet) (PartyInviteRequest, bool, error) {
	if packet.ID != PacketZCPartyJoinReq {
		return PartyInviteRequest{}, false, nil
	}
	if len(packet.Data) < 30 {
		return PartyInviteRequest{}, true, fmt.Errorf("ZC_REQ_JOIN_GROUP too short: %d", len(packet.Data))
	}
	return PartyInviteRequest{
		RequestID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		Name:      fixedPacketString(packet.Data[6:30]),
	}, true, nil
}

func ParsePartyOption(packet Packet) (PartyOption, bool, error) {
	if packet.ID != PacketZCGroupInfoChange {
		return PartyOption{}, false, nil
	}
	if len(packet.Data) < 6 {
		return PartyOption{}, true, fmt.Errorf("ZC_GROUPINFO_CHANGE too short: %d", len(packet.Data))
	}
	return PartyOption{ExpOption: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
}

func ParsePartyInviteConfig(packet Packet) (PartyInviteConfig, bool, error) {
	if packet.ID != PacketZCPartyConfig {
		return PartyInviteConfig{}, false, nil
	}
	if len(packet.Data) < 3 {
		return PartyInviteConfig{}, true, fmt.Errorf("ZC_PARTY_CONFIG too short: %d", len(packet.Data))
	}
	return PartyInviteConfig{RefuseInvites: packet.Data[2] != 0}, true, nil
}

func ParsePartyMemberJoin(packet Packet) (PartyMember, bool, error) {
	switch packet.ID {
	case PacketZCAddMemberToGroup, PacketZCAddMemberToGroup2:
	default:
		return PartyMember{}, false, nil
	}
	if len(packet.Data) < 79 {
		return PartyMember{}, true, fmt.Errorf("ZC_ADD_MEMBER_TO_GROUP 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	return PartyMember{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		Role:      binary.LittleEndian.Uint32(packet.Data[6:10]),
		X:         int(int16(binary.LittleEndian.Uint16(packet.Data[10:12]))),
		Y:         int(int16(binary.LittleEndian.Uint16(packet.Data[12:14]))),
		State:     packet.Data[14],
		GroupName: fixedPacketString(packet.Data[15:39]),
		Name:      fixedPacketString(packet.Data[39:63]),
		MapName:   fixedPacketString(packet.Data[63:79]),
	}, true, nil
}

func ParsePartyMemberLeave(packet Packet) (PartyMemberLeave, bool, error) {
	if packet.ID != PacketZCDeleteMemberFromGrp {
		return PartyMemberLeave{}, false, nil
	}
	if len(packet.Data) < 31 {
		return PartyMemberLeave{}, true, fmt.Errorf("ZC_DELETE_MEMBER_FROM_GROUP too short: %d", len(packet.Data))
	}
	return PartyMemberLeave{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		Name:      fixedPacketString(packet.Data[6:30]),
		Result:    packet.Data[30],
	}, true, nil
}

func ParsePartyMemberHP(packet Packet) (PartyMemberHP, bool, error) {
	switch packet.ID {
	case PacketZCNotifyHPToGroup:
		if len(packet.Data) < 10 {
			return PartyMemberHP{}, true, fmt.Errorf("ZC_NOTIFY_HP_TO_GROUPM too short: %d", len(packet.Data))
		}
		return PartyMemberHP{
			AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
			HP:        int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			MaxHP:     int(binary.LittleEndian.Uint16(packet.Data[8:10])),
		}, true, nil
	case PacketZCNotifyHPToGroupR2:
		if len(packet.Data) < 14 {
			return PartyMemberHP{}, true, fmt.Errorf("ZC_NOTIFY_HP_TO_GROUPM_R2 too short: %d", len(packet.Data))
		}
		return PartyMemberHP{
			AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
			HP:        int(binary.LittleEndian.Uint32(packet.Data[6:10])),
			MaxHP:     int(binary.LittleEndian.Uint32(packet.Data[10:14])),
		}, true, nil
	default:
		return PartyMemberHP{}, false, nil
	}
}

func ParsePartyMemberPosition(packet Packet) (PartyMemberPosition, bool, error) {
	if packet.ID != PacketZCNotifyPositionToGrp {
		return PartyMemberPosition{}, false, nil
	}
	if len(packet.Data) < 10 {
		return PartyMemberPosition{}, true, fmt.Errorf("ZC_NOTIFY_POSITION_TO_GROUPM too short: %d", len(packet.Data))
	}
	return PartyMemberPosition{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		X:         int(int16(binary.LittleEndian.Uint16(packet.Data[6:8]))),
		Y:         int(int16(binary.LittleEndian.Uint16(packet.Data[8:10]))),
	}, true, nil
}

func ParsePartyChat(packet Packet) (PartyChat, bool, error) {
	if packet.ID != PacketZCNotifyChatParty {
		return PartyChat{}, false, nil
	}
	if len(packet.Data) < 8 {
		return PartyChat{}, true, fmt.Errorf("ZC_NOTIFY_CHAT_PARTY too short: %d", len(packet.Data))
	}
	return PartyChat{
		AccountID: binary.LittleEndian.Uint32(packet.Data[4:8]),
		Message:   packetCString(packet.Data[8:]),
	}, true, nil
}

func BuildMakePartyPacket(name string) []byte {
	var w Writer
	w.Uint16(PacketCZMakeGroup)
	w.CString(name, partyNameLength)
	return w.Bytes()
}

func BuildMakeParty2Packet(name string, itemPickupRule, itemDivisionRule uint8) []byte {
	var w Writer
	w.Uint16(PacketCZMakeGroup2)
	w.CString(name, partyNameLength)
	w.Uint8(itemPickupRule)
	w.Uint8(itemDivisionRule)
	return w.Bytes()
}

func BuildPartyInvitePacket(accountID uint32, name string) []byte {
	var w Writer
	name = strings.TrimSpace(name)
	if name != "" {
		w.Uint16(PacketCZPartyJoinReq)
		w.CString(name, partyNameLength)
		return w.Bytes()
	}
	w.Uint16(PacketCZReqJoinGroup)
	w.Uint32(accountID)
	return w.Bytes()
}

func BuildPartyInviteAckPacket(requestID uint32, accepted bool) []byte {
	var w Writer
	w.Uint16(PacketCZPartyJoinReqAck)
	w.Uint32(requestID)
	if accepted {
		w.Uint8(1)
	} else {
		w.Uint8(0)
	}
	return w.Bytes()
}

func BuildLeavePartyPacket() []byte {
	var w Writer
	w.Uint16(PacketCZReqLeaveGroup)
	return w.Bytes()
}

func BuildPartyOptionPacket(expOption uint32) []byte {
	var w Writer
	w.Uint16(PacketCZChangeGroupExp)
	w.Uint32(expOption)
	return w.Bytes()
}

func BuildPartyInviteConfigPacket(refuseInvites bool) []byte {
	var w Writer
	w.Uint16(PacketCZPartyConfig)
	if refuseInvites {
		w.Uint8(1)
	} else {
		w.Uint8(0)
	}
	return w.Bytes()
}

func BuildExpelPartyMemberPacket(accountID uint32, name string) []byte {
	var w Writer
	w.Uint16(PacketCZReqExpelGroupMember)
	w.Uint32(accountID)
	w.CString(name, partyNameLength)
	return w.Bytes()
}

func BuildPartyMessagePacket(message string) []byte {
	message = strings.TrimSpace(message)
	var w Writer
	length := uint16(2 + 2 + len([]byte(message)) + 1)
	w.Uint16(PacketCZPartyMessage)
	w.Uint16(length)
	w.CString(message, len([]byte(message))+1)
	return w.Bytes()
}

func (c *Client) SendMakeParty(name string) error {
	packet := BuildMakePartyPacket(name)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MAKE_GROUP opcode=0x%04X name=%q client_date=%d", ID(packet), name, c.clientDate)
	} else {
		glog.Warnf("send CZ_MAKE_GROUP failed opcode=0x%04X len=%d name=%q client_date=%d: %v", ID(packet), len(packet), name, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMakeParty2(name string, itemPickupRule, itemDivisionRule uint8) error {
	packet := BuildMakeParty2Packet(name, itemPickupRule, itemDivisionRule)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MAKE_GROUP2 opcode=0x%04X name=%q pickup=%d division=%d client_date=%d", ID(packet), name, itemPickupRule, itemDivisionRule, c.clientDate)
	} else {
		glog.Warnf("send CZ_MAKE_GROUP2 failed opcode=0x%04X len=%d name=%q pickup=%d division=%d client_date=%d: %v", ID(packet), len(packet), name, itemPickupRule, itemDivisionRule, c.clientDate, err)
	}
	return err
}

func (c *Client) SendPartyInvite(accountID uint32, name string) error {
	packet := BuildPartyInvitePacket(accountID, name)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_PARTY_JOIN_REQ opcode=0x%04X aid=%d name=%q client_date=%d", ID(packet), accountID, name, c.clientDate)
	} else {
		glog.Warnf("send CZ_PARTY_JOIN_REQ failed opcode=0x%04X len=%d aid=%d name=%q client_date=%d: %v", ID(packet), len(packet), accountID, name, c.clientDate, err)
	}
	return err
}

func (c *Client) SendPartyInviteAck(requestID uint32, accepted bool) error {
	packet := BuildPartyInviteAckPacket(requestID, accepted)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_PARTY_JOIN_REQ_ACK opcode=0x%04X request=%d accepted=%t client_date=%d", ID(packet), requestID, accepted, c.clientDate)
	} else {
		glog.Warnf("send CZ_PARTY_JOIN_REQ_ACK failed opcode=0x%04X len=%d request=%d accepted=%t client_date=%d: %v", ID(packet), len(packet), requestID, accepted, c.clientDate, err)
	}
	return err
}

func (c *Client) SendLeaveParty() error {
	packet := BuildLeavePartyPacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_LEAVE_GROUP opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_LEAVE_GROUP failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendPartyOption(expOption uint32) error {
	packet := BuildPartyOptionPacket(expOption)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CHANGE_GROUPEXPOPTION opcode=0x%04X exp=%d client_date=%d", ID(packet), expOption, c.clientDate)
	} else {
		glog.Warnf("send CZ_CHANGE_GROUPEXPOPTION failed opcode=0x%04X len=%d exp=%d client_date=%d: %v", ID(packet), len(packet), expOption, c.clientDate, err)
	}
	return err
}

func (c *Client) SendPartyInviteConfig(refuseInvites bool) error {
	packet := BuildPartyInviteConfigPacket(refuseInvites)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_PARTY_CONFIG opcode=0x%04X refuse_invites=%t client_date=%d", ID(packet), refuseInvites, c.clientDate)
	} else {
		glog.Warnf("send CZ_PARTY_CONFIG failed opcode=0x%04X len=%d refuse_invites=%t client_date=%d: %v", ID(packet), len(packet), refuseInvites, c.clientDate, err)
	}
	return err
}

func (c *Client) SendExpelPartyMember(accountID uint32, name string) error {
	packet := BuildExpelPartyMemberPacket(accountID, name)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_EXPEL_GROUP_MEMBER opcode=0x%04X aid=%d name=%q client_date=%d", ID(packet), accountID, name, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_EXPEL_GROUP_MEMBER failed opcode=0x%04X len=%d aid=%d name=%q client_date=%d: %v", ID(packet), len(packet), accountID, name, c.clientDate, err)
	}
	return err
}

func (c *Client) SendPartyMessage(message string) error {
	packet := BuildPartyMessagePacket(message)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQUEST_CHAT_PARTY opcode=0x%04X message=%q client_date=%d", ID(packet), message, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQUEST_CHAT_PARTY failed opcode=0x%04X len=%d message=%q client_date=%d: %v", ID(packet), len(packet), message, c.clientDate, err)
	}
	return err
}
