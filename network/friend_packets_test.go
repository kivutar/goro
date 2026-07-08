package network

import (
	"encoding/binary"
	"testing"
)

func TestParseFriendsList(t *testing.T) {
	data := make([]byte, 4+32*2)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCFriendsList)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint32(data[4:8], 10)
	binary.LittleEndian.PutUint32(data[8:12], 20)
	copy(data[12:36], "Alice")
	binary.LittleEndian.PutUint32(data[36:40], 11)
	binary.LittleEndian.PutUint32(data[40:44], 21)
	copy(data[44:68], "Bob")

	friends, ok, err := ParseFriendsList(Packet{ID: PacketZCFriendsList, Data: data})
	if err != nil || !ok {
		t.Fatalf("ParseFriendsList ok=%t err=%v", ok, err)
	}
	if len(friends) != 2 {
		t.Fatalf("friends len = %d, want 2", len(friends))
	}
	if friends[0].AccountID != 10 || friends[0].CharID != 20 || friends[0].Name != "Alice" || friends[0].State != 1 {
		t.Fatalf("friend 0 = %+v", friends[0])
	}
	if friends[1].AccountID != 11 || friends[1].CharID != 21 || friends[1].Name != "Bob" || friends[1].State != 1 {
		t.Fatalf("friend 1 = %+v", friends[1])
	}
}

func TestParseCompactFriendsList(t *testing.T) {
	data := make([]byte, 4+8*2)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCFriendsList)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint32(data[4:8], 10)
	binary.LittleEndian.PutUint32(data[8:12], 20)
	binary.LittleEndian.PutUint32(data[12:16], 11)
	binary.LittleEndian.PutUint32(data[16:20], 21)

	friends, ok, err := ParseFriendsList(Packet{ID: PacketZCFriendsList, Data: data})
	if err != nil || !ok {
		t.Fatalf("ParseFriendsList ok=%t err=%v", ok, err)
	}
	if len(friends) != 2 {
		t.Fatalf("friends len = %d, want 2", len(friends))
	}
	if friends[0].AccountID != 10 || friends[0].CharID != 20 || friends[0].Name != "" || friends[0].State != 1 {
		t.Fatalf("friend 0 = %+v", friends[0])
	}
	if friends[1].AccountID != 11 || friends[1].CharID != 21 || friends[1].Name != "" || friends[1].State != 1 {
		t.Fatalf("friend 1 = %+v", friends[1])
	}
}

func TestParseFriendPackets(t *testing.T) {
	state := []byte{0x06, 0x02, 1, 0, 0, 0, 2, 0, 0, 0, 0}
	parsedState, ok, err := ParseFriendState(Packet{ID: PacketZCFriendsState, Data: state})
	if err != nil || !ok {
		t.Fatalf("ParseFriendState ok=%t err=%v", ok, err)
	}
	if parsedState.AccountID != 1 || parsedState.CharID != 2 || parsedState.State != 0 {
		t.Fatalf("state = %+v", parsedState)
	}

	stateWithName := make([]byte, 35)
	binary.LittleEndian.PutUint16(stateWithName[0:2], PacketZCFriendsState)
	binary.LittleEndian.PutUint32(stateWithName[2:6], 1)
	binary.LittleEndian.PutUint32(stateWithName[6:10], 2)
	copy(stateWithName[11:35], "Alice")
	parsedStateWithName, ok, err := ParseFriendState(Packet{ID: PacketZCFriendsState, Data: stateWithName})
	if err != nil || !ok {
		t.Fatalf("ParseFriendState with name ok=%t err=%v", ok, err)
	}
	if parsedStateWithName.Name != "Alice" {
		t.Fatalf("state with name = %+v", parsedStateWithName)
	}

	request := make([]byte, 34)
	binary.LittleEndian.PutUint16(request[0:2], PacketZCReqAddFriends)
	binary.LittleEndian.PutUint32(request[2:6], 3)
	binary.LittleEndian.PutUint32(request[6:10], 4)
	copy(request[10:34], "Charlie")
	parsedRequest, ok, err := ParseFriendRequest(Packet{ID: PacketZCReqAddFriends, Data: request})
	if err != nil || !ok {
		t.Fatalf("ParseFriendRequest ok=%t err=%v", ok, err)
	}
	if parsedRequest.AccountID != 3 || parsedRequest.CharID != 4 || parsedRequest.Name != "Charlie" {
		t.Fatalf("request = %+v", parsedRequest)
	}

	added := make([]byte, 36)
	binary.LittleEndian.PutUint16(added[0:2], PacketZCAddFriendsList)
	binary.LittleEndian.PutUint16(added[2:4], 0)
	binary.LittleEndian.PutUint32(added[4:8], 5)
	binary.LittleEndian.PutUint32(added[8:12], 6)
	copy(added[12:36], "Daphne")
	parsedAdded, ok, err := ParseFriendAddResult(Packet{ID: PacketZCAddFriendsList, Data: added})
	if err != nil || !ok {
		t.Fatalf("ParseFriendAddResult ok=%t err=%v", ok, err)
	}
	if parsedAdded.Result != 0 || parsedAdded.AccountID != 5 || parsedAdded.CharID != 6 || parsedAdded.Name != "Daphne" {
		t.Fatalf("added = %+v", parsedAdded)
	}

	deleted := []byte{0x0a, 0x02, 7, 0, 0, 0, 8, 0, 0, 0}
	parsedDeleted, ok, err := ParseFriendDelete(Packet{ID: PacketZCDeleteFriends, Data: deleted})
	if err != nil || !ok {
		t.Fatalf("ParseFriendDelete ok=%t err=%v", ok, err)
	}
	if parsedDeleted.AccountID != 7 || parsedDeleted.CharID != 8 {
		t.Fatalf("deleted = %+v", parsedDeleted)
	}
}

func TestBuildFriendPackets(t *testing.T) {
	add := BuildAddFriendPacket("Alice")
	if len(add) != 26 || ID(add) != PacketCZAddFriends || string(add[2:7]) != "Alice" {
		t.Fatalf("add packet len=%d id=0x%04X data=%q", len(add), ID(add), add[2:7])
	}

	del := BuildDeleteFriendPacket(0x11223344, 0x55667788)
	if len(del) != 10 || ID(del) != PacketCZDeleteFriends {
		t.Fatalf("delete packet len=%d id=0x%04X", len(del), ID(del))
	}
	if binary.LittleEndian.Uint32(del[2:6]) != 0x11223344 || binary.LittleEndian.Uint32(del[6:10]) != 0x55667788 {
		t.Fatalf("delete packet ids=%x/%x", del[2:6], del[6:10])
	}

	ack := BuildAckFriendRequestPacket(0x01020304, 0x05060708, true)
	if len(ack) != 14 || ID(ack) != PacketCZAckReqAddFriends {
		t.Fatalf("ack packet len=%d id=0x%04X", len(ack), ID(ack))
	}
	if binary.LittleEndian.Uint32(ack[2:6]) != 0x01020304 || binary.LittleEndian.Uint32(ack[6:10]) != 0x05060708 || binary.LittleEndian.Uint32(ack[10:14]) != 1 {
		t.Fatalf("ack packet = %x", ack)
	}
}
