package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketCZAddFriends       uint16 = 0x0202
	PacketCZDeleteFriends    uint16 = 0x0203
	PacketCZAckReqAddFriends uint16 = 0x0208

	PacketZCFriendsList    uint16 = 0x0201
	PacketZCFriendsState   uint16 = 0x0206
	PacketZCReqAddFriends  uint16 = 0x0207
	PacketZCAddFriendsList uint16 = 0x0209
	PacketZCDeleteFriends  uint16 = 0x020A
)

type Friend struct {
	AccountID uint32
	CharID    uint32
	Name      string
	State     uint8
}

type FriendState struct {
	AccountID uint32
	CharID    uint32
	State     uint8
	Name      string
}

type FriendRequest struct {
	AccountID uint32
	CharID    uint32
	Name      string
}

type FriendAddResult struct {
	Result    uint16
	AccountID uint32
	CharID    uint32
	Name      string
}

type FriendDelete struct {
	AccountID uint32
	CharID    uint32
}

func ParseFriendsList(packet Packet) ([]Friend, bool, error) {
	if packet.ID != PacketZCFriendsList {
		return nil, false, nil
	}
	if len(packet.Data) < 4 {
		return nil, true, fmt.Errorf("ZC_FRIENDS_LIST too short: %d", len(packet.Data))
	}
	body := packet.Data[4:]
	if len(body) == 0 {
		return nil, true, nil
	}
	if len(body)%32 == 0 && friendListNamesLookValid(body) {
		return parseNamedFriendsList(body), true, nil
	}
	if len(body)%8 == 0 {
		return parseCompactFriendsList(body), true, nil
	}
	return nil, true, fmt.Errorf("ZC_FRIENDS_LIST bad body len: %d", len(body))
}

func parseNamedFriendsList(body []byte) []Friend {
	friends := make([]Friend, 0, len(body)/32)
	for len(body) > 0 {
		friends = append(friends, Friend{
			AccountID: binary.LittleEndian.Uint32(body[0:4]),
			CharID:    binary.LittleEndian.Uint32(body[4:8]),
			Name:      fixedPacketString(body[8:32]),
			State:     1,
		})
		body = body[32:]
	}
	return friends
}

func parseCompactFriendsList(body []byte) []Friend {
	friends := make([]Friend, 0, len(body)/8)
	for len(body) > 0 {
		friends = append(friends, Friend{
			AccountID: binary.LittleEndian.Uint32(body[0:4]),
			CharID:    binary.LittleEndian.Uint32(body[4:8]),
			State:     1,
		})
		body = body[8:]
	}
	return friends
}

func friendListNamesLookValid(body []byte) bool {
	for len(body) > 0 {
		if !validFriendNameBytes(body[8:32]) {
			return false
		}
		body = body[32:]
	}
	return true
}

func validFriendNameBytes(raw []byte) bool {
	seen := false
	for _, b := range raw {
		if b == 0 {
			break
		}
		if b < 0x20 || b == 0x7f {
			return false
		}
		seen = true
	}
	return seen
}

func ParseFriendState(packet Packet) (FriendState, bool, error) {
	if packet.ID != PacketZCFriendsState {
		return FriendState{}, false, nil
	}
	if len(packet.Data) < 11 {
		return FriendState{}, true, fmt.Errorf("ZC_FRIENDS_STATE too short: %d", len(packet.Data))
	}
	return FriendState{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		CharID:    binary.LittleEndian.Uint32(packet.Data[6:10]),
		State:     packet.Data[10],
		Name:      friendStateName(packet.Data),
	}, true, nil
}

func friendStateName(data []byte) string {
	if len(data) < 35 {
		return ""
	}
	return fixedPacketString(data[11:35])
}

func ParseFriendRequest(packet Packet) (FriendRequest, bool, error) {
	if packet.ID != PacketZCReqAddFriends {
		return FriendRequest{}, false, nil
	}
	if len(packet.Data) < 34 {
		return FriendRequest{}, true, fmt.Errorf("ZC_REQ_ADD_FRIENDS too short: %d", len(packet.Data))
	}
	return FriendRequest{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		CharID:    binary.LittleEndian.Uint32(packet.Data[6:10]),
		Name:      fixedPacketString(packet.Data[10:34]),
	}, true, nil
}

func ParseFriendAddResult(packet Packet) (FriendAddResult, bool, error) {
	if packet.ID != PacketZCAddFriendsList {
		return FriendAddResult{}, false, nil
	}
	if len(packet.Data) < 36 {
		return FriendAddResult{}, true, fmt.Errorf("ZC_ADD_FRIENDS_LIST too short: %d", len(packet.Data))
	}
	return FriendAddResult{
		Result:    binary.LittleEndian.Uint16(packet.Data[2:4]),
		AccountID: binary.LittleEndian.Uint32(packet.Data[4:8]),
		CharID:    binary.LittleEndian.Uint32(packet.Data[8:12]),
		Name:      fixedPacketString(packet.Data[12:36]),
	}, true, nil
}

func ParseFriendDelete(packet Packet) (FriendDelete, bool, error) {
	if packet.ID != PacketZCDeleteFriends {
		return FriendDelete{}, false, nil
	}
	if len(packet.Data) < 10 {
		return FriendDelete{}, true, fmt.Errorf("ZC_DELETE_FRIENDS too short: %d", len(packet.Data))
	}
	return FriendDelete{
		AccountID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		CharID:    binary.LittleEndian.Uint32(packet.Data[6:10]),
	}, true, nil
}

func BuildAddFriendPacket(name string) []byte {
	var w Writer
	w.Uint16(PacketCZAddFriends)
	w.CString(name, 24)
	return w.Bytes()
}

func BuildDeleteFriendPacket(accountID, charID uint32) []byte {
	var w Writer
	w.Uint16(PacketCZDeleteFriends)
	w.Uint32(accountID)
	w.Uint32(charID)
	return w.Bytes()
}

func BuildAckFriendRequestPacket(accountID, charID uint32, accepted bool) []byte {
	var w Writer
	w.Uint16(PacketCZAckReqAddFriends)
	w.Uint32(accountID)
	w.Uint32(charID)
	if accepted {
		w.Uint32(1)
	} else {
		w.Uint32(0)
	}
	return w.Bytes()
}

func (c *Client) SendAddFriend(name string) error {
	packet := BuildAddFriendPacket(name)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ADD_FRIENDS opcode=0x%04X name=%q client_date=%d", ID(packet), name, c.clientDate)
	} else {
		glog.Warnf("send CZ_ADD_FRIENDS failed opcode=0x%04X len=%d name=%q client_date=%d: %v", ID(packet), len(packet), name, c.clientDate, err)
	}
	return err
}

func (c *Client) SendDeleteFriend(accountID, charID uint32) error {
	packet := BuildDeleteFriendPacket(accountID, charID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_DELETE_FRIENDS opcode=0x%04X aid=%d gid=%d client_date=%d", ID(packet), accountID, charID, c.clientDate)
	} else {
		glog.Warnf("send CZ_DELETE_FRIENDS failed opcode=0x%04X len=%d aid=%d gid=%d client_date=%d: %v", ID(packet), len(packet), accountID, charID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendFriendRequestAck(accountID, charID uint32, accepted bool) error {
	packet := BuildAckFriendRequestPacket(accountID, charID, accepted)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ACK_REQ_ADD_FRIENDS opcode=0x%04X aid=%d gid=%d accepted=%t client_date=%d", ID(packet), accountID, charID, accepted, c.clientDate)
	} else {
		glog.Warnf("send CZ_ACK_REQ_ADD_FRIENDS failed opcode=0x%04X len=%d aid=%d gid=%d accepted=%t client_date=%d: %v", ID(packet), len(packet), accountID, charID, accepted, c.clientDate, err)
	}
	return err
}
