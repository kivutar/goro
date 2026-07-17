package network

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketCZReqMakeGuild    uint16 = 0x0165
	PacketZCResultMakeGuild uint16 = 0x0167
	PacketCZReqJoinGuild    uint16 = 0x0168
	PacketZCAckReqJoinGuild uint16 = 0x0169
	PacketZCReqJoinGuild    uint16 = 0x016A
	PacketCZJoinGuild       uint16 = 0x016B
	PacketZCGuildInfo       uint16 = 0x0150
	PacketZCGuildInfo2      uint16 = 0x01B6
	PacketZCUpdateGuildID   uint16 = 0x016C
	PacketCZReqGuildEmblem  uint16 = 0x0151
	PacketZCGuildEmblem     uint16 = 0x0152
	PacketCZRegGuildEmblem  uint16 = 0x0153
	PacketZCChangeGuild     uint16 = 0x01B4
)

const guildNameLength = 24

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
	GuildID       uint32
	EmblemVersion uint32
	GuildName     string
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

func ParseGuildInfo(packet Packet) (GuildInfo, bool, error) {
	switch packet.ID {
	case PacketZCGuildInfo, PacketZCGuildInfo2:
	default:
		return GuildInfo{}, false, nil
	}
	if len(packet.Data) < 66 {
		return GuildInfo{}, true, fmt.Errorf("ZC_GUILD_INFO 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	nameOffset := 42
	var emblemVersion uint32
	if len(packet.Data) >= 70 {
		emblemVersion = binary.LittleEndian.Uint32(packet.Data[42:46])
		nameOffset = 46
	}
	return GuildInfo{
		GuildID:       binary.LittleEndian.Uint32(packet.Data[2:6]),
		EmblemVersion: emblemVersion,
		GuildName:     decodeROFixedString(packet.Data[nameOffset : nameOffset+guildNameLength]),
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

func BuildGuildEmblemRequestPacket(guildID uint32) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqGuildEmblem)
	binary.LittleEndian.PutUint32(packet[2:6], guildID)
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
