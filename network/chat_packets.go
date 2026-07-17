package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
	"strings"
)

const (
	PacketCZRequestChatLegacy uint16 = 0x008C
	PacketCZRequestChat       uint16 = 0x00F3
	PacketCZWhisper           uint16 = 0x0096
	PacketCZWhisperIgnore     uint16 = 0x00CF
	PacketCZWhisperIgnoreAll  uint16 = 0x00D0
	PacketCZCreateChatRoom    uint16 = 0x00D5
	PacketCZEnterChatRoom     uint16 = 0x00D9
	PacketCZExitChatRoom      uint16 = 0x00E3
	PacketZCNotifyChat        uint16 = 0x008D
	PacketZCNotifyPlayerChat  uint16 = 0x008E
	PacketZCWhisper           uint16 = 0x0097
	PacketZCAckWhisper        uint16 = 0x0098
	PacketZCWhisperIgnoreAck  uint16 = 0x00D1
	PacketZCWhisperAllAck     uint16 = 0x00D2
	PacketZCAckCreateChatRoom uint16 = 0x00D6
	PacketZCChatRoomBoard     uint16 = 0x00D7
	PacketZCDestroyChatRoom   uint16 = 0x00D8
	PacketZCChatRoomChanged   uint16 = 0x00DF
	PacketZCChatRoomEnter     uint16 = 0x00DB
	PacketZCChatRoomJoin      uint16 = 0x00DC
	PacketZCChatRoomLeave     uint16 = 0x00DD
	PacketZCChatRoomRole      uint16 = 0x00E1
	PacketZCBroadcast         uint16 = 0x009A
	PacketZCMsg               uint16 = 0x0291
	PacketZCMsgValue          uint16 = 0x07E2
	PacketZCMsgSkill          uint16 = 0x07E6
	PacketZCMsgColor          uint16 = 0x09CD
)

type ChatMessage struct {
	GID       uint32
	Text      string
	MessageID int
	Value     int32
	SkillID   uint16
	Color     uint32
}

type WhisperMessage struct {
	Sender  string
	Message string
}

type WhisperAck struct {
	Result uint8
}

type WhisperIgnoreAck struct {
	TargetAll bool
	Allow     bool
	Result    uint8
}

type ChatRoomCreate struct {
	Title    string
	Password string
	Limit    uint16
	Public   bool
}

type ChatRoomCreateAck struct {
	Result uint8
}

type ChatRoomBoard struct {
	OwnerID uint32
	RoomID  uint32
	Limit   uint16
	Count   uint16
	Public  bool
	Title   string
}

type ChatRoomDestroy struct {
	RoomID uint32
}

type ChatRoomMember struct {
	Owner bool
	Name  string
}

type ChatRoomEnter struct {
	RoomID  uint32
	Members []ChatRoomMember
}

type ChatRoomMemberJoin struct {
	Count uint16
	Name  string
}

type ChatRoomMemberLeave struct {
	Count  uint16
	Name   string
	Kicked bool
}

type ChatRoomChange struct {
	OwnerID uint32
	RoomID  uint32
	Limit   uint16
	Count   uint16
	Public  bool
	Title   string
}

type ChatRoomRoleChange struct {
	Owner bool
	Name  string
}

func BuildGlobalChatPacket(name, message string) []byte {
	return BuildGlobalChatPacketForClientDate(name, message, 20080910)
}

func BuildGlobalChatPacketForClientDate(name, message string, clientDate int) []byte {
	payload := strings.TrimSpace(name) + " : " + strings.TrimSpace(message)
	if strings.TrimSpace(name) == "" || strings.TrimSpace(message) == "" {
		return nil
	}
	size := 4 + len([]byte(payload)) + 1
	if size > 0xffff {
		return nil
	}
	opcode := PacketCZRequestChat
	if clientDate < 20040726 {
		opcode = PacketCZRequestChatLegacy
	}
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], opcode)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	copy(packet[4:], []byte(payload))
	return packet
}

func BuildWhisperPacket(receiver, message string) []byte {
	receiver = strings.TrimSpace(receiver)
	message = strings.TrimSpace(message)
	if receiver == "" || message == "" {
		return nil
	}
	size := 2 + 2 + 24 + len([]byte(message)) + 1
	if size > 0xffff {
		return nil
	}
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZWhisper)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	writeFixedCString(packet[4:28], receiver)
	copy(packet[28:], []byte(message))
	return packet
}

func BuildWhisperIgnorePacket(name string, allow bool) []byte {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	packet := make([]byte, 27)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZWhisperIgnore)
	writeFixedName(packet[2:26], name)
	packet[26] = whisperAllowByte(allow)
	return packet
}

func BuildWhisperIgnoreAllPacket(allow bool) []byte {
	return []byte{byte(PacketCZWhisperIgnoreAll), byte(PacketCZWhisperIgnoreAll >> 8), whisperAllowByte(allow)}
}

func BuildCreateChatRoomPacket(room ChatRoomCreate) []byte {
	title := strings.TrimSpace(room.Title)
	if title == "" || room.Limit == 0 {
		return nil
	}
	titleBytes := []byte(title)
	size := 2 + 2 + 2 + 1 + 8 + len(titleBytes)
	if size > 0xffff {
		return nil
	}
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZCreateChatRoom)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	binary.LittleEndian.PutUint16(packet[4:6], room.Limit)
	if room.Public {
		packet[6] = 1
	}
	writeFixedCString(packet[7:15], strings.TrimSpace(room.Password))
	copy(packet[15:], titleBytes)
	return packet
}

func BuildExitChatRoomPacket() []byte {
	return []byte{byte(PacketCZExitChatRoom), byte(PacketCZExitChatRoom >> 8)}
}

func BuildEnterChatRoomPacket(roomID uint32, password string) []byte {
	if roomID == 0 {
		return nil
	}
	packet := make([]byte, 14)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZEnterChatRoom)
	binary.LittleEndian.PutUint32(packet[2:6], roomID)
	writeFixedCString(packet[6:14], strings.TrimSpace(password))
	return packet
}

func whisperAllowByte(allow bool) byte {
	if allow {
		return 1
	}
	return 0
}

func ParseChatMessage(packet Packet) (ChatMessage, bool, error) {
	switch packet.ID {
	case PacketZCNotifyChat:
		if len(packet.Data) < 8 {
			return ChatMessage{}, false, fmt.Errorf("ZC_NOTIFY_CHAT too short: %d", len(packet.Data))
		}
		return ChatMessage{
			GID:  binary.LittleEndian.Uint32(packet.Data[4:8]),
			Text: packetCString(packet.Data[8:]),
		}, true, nil
	case PacketZCNotifyPlayerChat:
		if len(packet.Data) < 4 {
			return ChatMessage{}, false, fmt.Errorf("ZC_NOTIFY_PLAYERCHAT too short: %d", len(packet.Data))
		}
		return ChatMessage{
			Text: packetCString(packet.Data[4:]),
		}, true, nil
	case PacketZCBroadcast:
		if len(packet.Data) < 4 {
			return ChatMessage{}, false, fmt.Errorf("ZC_BROADCAST too short: %d", len(packet.Data))
		}
		return ChatMessage{
			Text: packetCString(packet.Data[4:]),
		}, true, nil
	case PacketZCMsg:
		if len(packet.Data) < 4 {
			return ChatMessage{}, false, fmt.Errorf("ZC_MSG too short: %d", len(packet.Data))
		}
		return ChatMessage{MessageID: int(binary.LittleEndian.Uint16(packet.Data[2:4]))}, true, nil
	case PacketZCMsgValue:
		if len(packet.Data) < 8 {
			return ChatMessage{}, false, fmt.Errorf("ZC_MSG_VALUE too short: %d", len(packet.Data))
		}
		messageID := binary.LittleEndian.Uint16(packet.Data[2:4])
		value := int32(binary.LittleEndian.Uint32(packet.Data[4:8]))
		return ChatMessage{MessageID: int(messageID), Value: value}, true, nil
	case PacketZCMsgSkill:
		if len(packet.Data) < 8 {
			return ChatMessage{}, false, fmt.Errorf("ZC_MSG_SKILL too short: %d", len(packet.Data))
		}
		skillID := binary.LittleEndian.Uint16(packet.Data[2:4])
		messageID := int32(binary.LittleEndian.Uint32(packet.Data[4:8]))
		return ChatMessage{MessageID: int(messageID), SkillID: skillID}, true, nil
	case PacketZCMsgColor:
		if len(packet.Data) < 8 {
			return ChatMessage{}, false, fmt.Errorf("ZC_MSG_COLOR too short: %d", len(packet.Data))
		}
		messageID := binary.LittleEndian.Uint16(packet.Data[2:4])
		color := binary.LittleEndian.Uint32(packet.Data[4:8])
		return ChatMessage{MessageID: int(messageID), Color: color}, true, nil
	default:
		return ChatMessage{}, false, nil
	}
}

func ParseWhisperMessage(packet Packet) (WhisperMessage, bool, error) {
	if packet.ID != PacketZCWhisper {
		return WhisperMessage{}, false, nil
	}
	if len(packet.Data) < 28 {
		return WhisperMessage{}, false, fmt.Errorf("ZC_WHISPER too short: %d", len(packet.Data))
	}
	return WhisperMessage{
		Sender:  packetCString(packet.Data[4:28]),
		Message: packetCString(packet.Data[28:]),
	}, true, nil
}

func ParseWhisperAck(packet Packet) (WhisperAck, bool, error) {
	if packet.ID != PacketZCAckWhisper {
		return WhisperAck{}, false, nil
	}
	if len(packet.Data) < 3 {
		return WhisperAck{}, false, fmt.Errorf("ZC_ACK_WHISPER too short: %d", len(packet.Data))
	}
	return WhisperAck{Result: packet.Data[2]}, true, nil
}

func ParseWhisperIgnoreAck(packet Packet) (WhisperIgnoreAck, bool, error) {
	switch packet.ID {
	case PacketZCWhisperIgnoreAck:
		if len(packet.Data) < 4 {
			return WhisperIgnoreAck{}, false, fmt.Errorf("ZC_SETTING_WHISPER_PC too short: %d", len(packet.Data))
		}
		return WhisperIgnoreAck{
			Allow:  packet.Data[2] != 0,
			Result: packet.Data[3],
		}, true, nil
	case PacketZCWhisperAllAck:
		if len(packet.Data) < 4 {
			return WhisperIgnoreAck{}, false, fmt.Errorf("ZC_SETTING_WHISPER_STATE too short: %d", len(packet.Data))
		}
		return WhisperIgnoreAck{
			TargetAll: true,
			Allow:     packet.Data[2] != 0,
			Result:    packet.Data[3],
		}, true, nil
	default:
		return WhisperIgnoreAck{}, false, nil
	}
}

func ParseChatRoomCreateAck(packet Packet) (ChatRoomCreateAck, bool, error) {
	if packet.ID != PacketZCAckCreateChatRoom {
		return ChatRoomCreateAck{}, false, nil
	}
	if len(packet.Data) < 3 {
		return ChatRoomCreateAck{}, false, fmt.Errorf("ZC_ACK_CREATE_CHATROOM too short: %d", len(packet.Data))
	}
	return ChatRoomCreateAck{Result: packet.Data[2]}, true, nil
}

func ParseChatRoomBoard(packet Packet) (ChatRoomBoard, bool, error) {
	if packet.ID != PacketZCChatRoomBoard {
		return ChatRoomBoard{}, false, nil
	}
	if len(packet.Data) < 17 {
		return ChatRoomBoard{}, false, fmt.Errorf("ZC_ROOM_NEWENTRY too short: %d", len(packet.Data))
	}
	return ChatRoomBoard{
		OwnerID: binary.LittleEndian.Uint32(packet.Data[4:8]),
		RoomID:  binary.LittleEndian.Uint32(packet.Data[8:12]),
		Limit:   binary.LittleEndian.Uint16(packet.Data[12:14]),
		Count:   binary.LittleEndian.Uint16(packet.Data[14:16]),
		Public:  packet.Data[16] != 0,
		Title:   packetCString(packet.Data[17:]),
	}, true, nil
}

func ParseChatRoomDestroy(packet Packet) (ChatRoomDestroy, bool, error) {
	if packet.ID != PacketZCDestroyChatRoom {
		return ChatRoomDestroy{}, false, nil
	}
	if len(packet.Data) < 6 {
		return ChatRoomDestroy{}, false, fmt.Errorf("ZC_DESTROY_ROOM too short: %d", len(packet.Data))
	}
	return ChatRoomDestroy{RoomID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
}

func ParseChatRoomEnter(packet Packet) (ChatRoomEnter, bool, error) {
	if packet.ID != PacketZCChatRoomEnter {
		return ChatRoomEnter{}, false, nil
	}
	if len(packet.Data) < 8 {
		return ChatRoomEnter{}, false, fmt.Errorf("ZC_ENTER_ROOM too short: %d", len(packet.Data))
	}
	remaining := len(packet.Data) - 8
	if remaining%28 != 0 {
		return ChatRoomEnter{}, false, fmt.Errorf("ZC_ENTER_ROOM malformed members len=%d", remaining)
	}
	enter := ChatRoomEnter{
		RoomID:  binary.LittleEndian.Uint32(packet.Data[4:8]),
		Members: make([]ChatRoomMember, 0, remaining/28),
	}
	for offset := 8; offset < len(packet.Data); offset += 28 {
		role := binary.LittleEndian.Uint32(packet.Data[offset : offset+4])
		enter.Members = append(enter.Members, ChatRoomMember{
			Owner: role == 0,
			Name:  packetCString(packet.Data[offset+4 : offset+28]),
		})
	}
	return enter, true, nil
}

func ParseChatRoomMemberJoin(packet Packet) (ChatRoomMemberJoin, bool, error) {
	if packet.ID != PacketZCChatRoomJoin {
		return ChatRoomMemberJoin{}, false, nil
	}
	if len(packet.Data) < 28 {
		return ChatRoomMemberJoin{}, false, fmt.Errorf("ZC_MEMBER_NEWENTRY too short: %d", len(packet.Data))
	}
	return ChatRoomMemberJoin{
		Count: binary.LittleEndian.Uint16(packet.Data[2:4]),
		Name:  packetCString(packet.Data[4:28]),
	}, true, nil
}

func ParseChatRoomMemberLeave(packet Packet) (ChatRoomMemberLeave, bool, error) {
	if packet.ID != PacketZCChatRoomLeave {
		return ChatRoomMemberLeave{}, false, nil
	}
	if len(packet.Data) < 29 {
		return ChatRoomMemberLeave{}, false, fmt.Errorf("ZC_MEMBER_EXIT too short: %d", len(packet.Data))
	}
	return ChatRoomMemberLeave{
		Count:  binary.LittleEndian.Uint16(packet.Data[2:4]),
		Name:   packetCString(packet.Data[4:28]),
		Kicked: packet.Data[28] != 0,
	}, true, nil
}

func ParseChatRoomChange(packet Packet) (ChatRoomChange, bool, error) {
	if packet.ID != PacketZCChatRoomChanged {
		return ChatRoomChange{}, false, nil
	}
	if len(packet.Data) < 17 {
		return ChatRoomChange{}, false, fmt.Errorf("ZC_CHANGE_CHATROOM too short: %d", len(packet.Data))
	}
	return ChatRoomChange{
		OwnerID: binary.LittleEndian.Uint32(packet.Data[4:8]),
		RoomID:  binary.LittleEndian.Uint32(packet.Data[8:12]),
		Limit:   binary.LittleEndian.Uint16(packet.Data[12:14]),
		Count:   binary.LittleEndian.Uint16(packet.Data[14:16]),
		Public:  packet.Data[16] != 0,
		Title:   packetCString(packet.Data[17:]),
	}, true, nil
}

func ParseChatRoomRoleChange(packet Packet) (ChatRoomRoleChange, bool, error) {
	if packet.ID != PacketZCChatRoomRole {
		return ChatRoomRoleChange{}, false, nil
	}
	if len(packet.Data) < 30 {
		return ChatRoomRoleChange{}, false, fmt.Errorf("ZC_ROLE_CHANGE too short: %d", len(packet.Data))
	}
	return ChatRoomRoleChange{
		Owner: binary.LittleEndian.Uint32(packet.Data[2:6]) == 0,
		Name:  packetCString(packet.Data[6:30]),
	}, true, nil
}

func packetCString(data []byte) string {
	if i := bytes.IndexByte(data, 0); i >= 0 {
		data = data[:i]
	}
	return strings.TrimSpace(string(data))
}

func writeFixedCString(dst []byte, value string) {
	src := []byte(value)
	copy(dst, src)
	if len(src) < len(dst) {
		dst[len(src)] = 0
	}
}

func writeFixedName(dst []byte, value string) {
	if len(dst) == 0 {
		return
	}
	src := []byte(value)
	if len(src) >= len(dst) {
		src = src[:len(dst)-1]
	}
	copy(dst, src)
	dst[len(src)] = 0
}

func (c *Client) SendGlobalChat(name, message string) error {
	packet := BuildGlobalChatPacketForClientDate(name, message, c.clientDate)
	if len(packet) == 0 {
		return fmt.Errorf("empty chat message")
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQUEST_CHAT opcode=0x%04X len=%d name=%q client_date=%d", ID(packet), len(packet), name, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQUEST_CHAT failed opcode=0x%04X len=%d name=%q client_date=%d: %v", ID(packet), len(packet), name, c.clientDate, err)
	}
	return err
}

func (c *Client) SendWhisper(receiver, message string) error {
	packet := BuildWhisperPacket(receiver, message)
	if len(packet) == 0 {
		return fmt.Errorf("empty whisper")
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_WHISPER opcode=0x%04X len=%d receiver=%q client_date=%d", ID(packet), len(packet), receiver, c.clientDate)
	} else {
		glog.Warnf("send CZ_WHISPER failed opcode=0x%04X len=%d receiver=%q client_date=%d: %v", ID(packet), len(packet), receiver, c.clientDate, err)
	}
	return err
}

func (c *Client) SendWhisperIgnore(name string, allow bool) error {
	packet := BuildWhisperIgnorePacket(name, allow)
	if len(packet) == 0 {
		return fmt.Errorf("empty whisper ignore target")
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_SETTING_WHISPER_PC opcode=0x%04X name=%q allow=%t client_date=%d", ID(packet), name, allow, c.clientDate)
	} else {
		glog.Warnf("send CZ_SETTING_WHISPER_PC failed opcode=0x%04X name=%q allow=%t client_date=%d: %v", ID(packet), name, allow, c.clientDate, err)
	}
	return err
}

func (c *Client) SendWhisperIgnoreAll(allow bool) error {
	packet := BuildWhisperIgnoreAllPacket(allow)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_SETTING_WHISPER_STATE opcode=0x%04X allow=%t client_date=%d", ID(packet), allow, c.clientDate)
	} else {
		glog.Warnf("send CZ_SETTING_WHISPER_STATE failed opcode=0x%04X allow=%t client_date=%d: %v", ID(packet), allow, c.clientDate, err)
	}
	return err
}

func (c *Client) SendCreateChatRoom(room ChatRoomCreate) error {
	packet := BuildCreateChatRoomPacket(room)
	if len(packet) == 0 {
		return fmt.Errorf("empty chat room")
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CREATE_CHATROOM opcode=0x%04X len=%d title=%q limit=%d public=%t client_date=%d", ID(packet), len(packet), room.Title, room.Limit, room.Public, c.clientDate)
	} else {
		glog.Warnf("send CZ_CREATE_CHATROOM failed opcode=0x%04X len=%d title=%q limit=%d public=%t client_date=%d: %v", ID(packet), len(packet), room.Title, room.Limit, room.Public, c.clientDate, err)
	}
	return err
}

func (c *Client) SendExitChatRoom() error {
	packet := BuildExitChatRoomPacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_EXIT_ROOM opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_EXIT_ROOM failed opcode=0x%04X client_date=%d: %v", ID(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendEnterChatRoom(roomID uint32, password string) error {
	packet := BuildEnterChatRoomPacket(roomID, password)
	if len(packet) == 0 {
		return fmt.Errorf("empty chat room id")
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_ENTER_ROOM opcode=0x%04X room=%d client_date=%d", ID(packet), roomID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_ENTER_ROOM failed opcode=0x%04X room=%d client_date=%d: %v", ID(packet), roomID, c.clientDate, err)
	}
	return err
}
