package network

import (
	"encoding/binary"
	"testing"
)

func TestBuildWoEGuildRelationPackets(t *testing.T) {
	request := BuildGuildAllianceRequestPacket(11, 22, 33)
	if len(request) != 14 || ID(request) != PacketCZReqGuildAlliance || binary.LittleEndian.Uint32(request[2:6]) != 11 || binary.LittleEndian.Uint32(request[6:10]) != 22 || binary.LittleEndian.Uint32(request[10:14]) != 33 {
		t.Fatalf("alliance request = %x", request)
	}
	reply := BuildGuildAllianceReplyPacket(44, true)
	if len(reply) != 10 || ID(reply) != PacketCZGuildAllianceReply || binary.LittleEndian.Uint32(reply[2:6]) != 44 || binary.LittleEndian.Uint32(reply[6:10]) != 1 {
		t.Fatalf("alliance reply = %x", reply)
	}
	hostile := BuildGuildHostilityRequestPacket(55)
	if len(hostile) != 6 || ID(hostile) != PacketCZReqGuildHostility || binary.LittleEndian.Uint32(hostile[2:6]) != 55 {
		t.Fatalf("hostility request = %x", hostile)
	}
	remove := BuildDeleteGuildRelationPacket(66, 1)
	if len(remove) != 10 || ID(remove) != PacketCZDeleteGuildRelation || binary.LittleEndian.Uint32(remove[2:6]) != 66 || binary.LittleEndian.Uint32(remove[6:10]) != 1 {
		t.Fatalf("relation delete = %x", remove)
	}
}

func TestParseWoEGuildRelations(t *testing.T) {
	list := make([]byte, 4+2*32)
	binary.LittleEndian.PutUint16(list[0:2], PacketZCGuildRelations)
	binary.LittleEndian.PutUint16(list[2:4], uint16(len(list)))
	binary.LittleEndian.PutUint32(list[4:8], 0)
	binary.LittleEndian.PutUint32(list[8:12], 10)
	copy(list[12:36], "Allies")
	binary.LittleEndian.PutUint32(list[36:40], 1)
	binary.LittleEndian.PutUint32(list[40:44], 20)
	copy(list[44:68], "Enemies")
	relations, ok, err := ParseGuildRelations(Packet{ID: PacketZCGuildRelations, Data: list})
	if !ok || err != nil || len(relations) != 2 || relations[0] != (GuildRelation{Relation: 0, GuildID: 10, Name: "Allies"}) || relations[1] != (GuildRelation{Relation: 1, GuildID: 20, Name: "Enemies"}) {
		t.Fatalf("relations ok=%t err=%v value=%+v", ok, err, relations)
	}

	added := make([]byte, 34)
	binary.LittleEndian.PutUint16(added[0:2], PacketZCGuildRelationAdded)
	binary.LittleEndian.PutUint32(added[2:6], 1)
	binary.LittleEndian.PutUint32(added[6:10], 30)
	copy(added[10:34], "Rivals")
	relation, ok, err := ParseGuildRelationAdded(Packet{ID: PacketZCGuildRelationAdded, Data: added})
	if !ok || err != nil || relation != (GuildRelation{Relation: 1, GuildID: 30, Name: "Rivals"}) {
		t.Fatalf("added relation ok=%t err=%v value=%+v", ok, err, relation)
	}

	deleted := make([]byte, 10)
	binary.LittleEndian.PutUint16(deleted[0:2], PacketZCGuildRelationDeleted)
	binary.LittleEndian.PutUint32(deleted[2:6], 30)
	binary.LittleEndian.PutUint32(deleted[6:10], 1)
	removed, ok, err := ParseGuildRelationDeleted(Packet{ID: PacketZCGuildRelationDeleted, Data: deleted})
	if !ok || err != nil || removed != (GuildRelationDeleted{GuildID: 30, Relation: 1}) {
		t.Fatalf("deleted relation ok=%t err=%v value=%+v", ok, err, removed)
	}
}

func TestParseWoEGuildRequestsAndResults(t *testing.T) {
	request := make([]byte, 30)
	binary.LittleEndian.PutUint16(request[0:2], PacketZCGuildAllianceRequest)
	binary.LittleEndian.PutUint32(request[2:6], 123)
	copy(request[6:30], "Mandala")
	parsed, ok, err := ParseGuildAllianceRequest(Packet{ID: PacketZCGuildAllianceRequest, Data: request})
	if !ok || err != nil || parsed.AccountID != 123 || parsed.GuildName != "Mandala" {
		t.Fatalf("alliance request ok=%t err=%v value=%+v", ok, err, parsed)
	}

	alliance, ok, err := ParseGuildAllianceResult(Packet{ID: PacketZCGuildAllianceResult, Data: []byte{0x73, 0x01, 2}})
	if !ok || err != nil || alliance.Result != 2 {
		t.Fatalf("alliance result ok=%t err=%v value=%+v", ok, err, alliance)
	}
	hostility, ok, err := ParseGuildHostilityResult(Packet{ID: PacketZCGuildHostilityResult, Data: []byte{0x81, 0x01, 1}})
	if !ok || err != nil || hostility.Result != 1 {
		t.Fatalf("hostility result ok=%t err=%v value=%+v", ok, err, hostility)
	}
}

func TestParseWoEGuildMemberUpdates(t *testing.T) {
	state := make([]byte, 14)
	binary.LittleEndian.PutUint16(state[0:2], PacketZCGuildMemberState)
	binary.LittleEndian.PutUint32(state[2:6], 10)
	binary.LittleEndian.PutUint32(state[6:10], 20)
	binary.LittleEndian.PutUint32(state[10:14], 1)
	parsedState, ok, err := ParseGuildMemberState(Packet{ID: PacketZCGuildMemberState, Data: state})
	if !ok || err != nil || parsedState != (GuildMemberState{AccountID: 10, CharID: 20, State: 1}) {
		t.Fatalf("member state ok=%t err=%v value=%+v", ok, err, parsedState)
	}
	state2 := make([]byte, 20)
	binary.LittleEndian.PutUint16(state2[0:2], PacketZCGuildMemberState2)
	binary.LittleEndian.PutUint32(state2[2:6], 30)
	binary.LittleEndian.PutUint32(state2[6:10], 40)
	binary.LittleEndian.PutUint32(state2[10:14], 1)
	binary.LittleEndian.PutUint16(state2[14:16], 1)
	binary.LittleEndian.PutUint16(state2[16:18], 7)
	binary.LittleEndian.PutUint16(state2[18:20], 8)
	parsedState, ok, err = ParseGuildMemberState(Packet{ID: PacketZCGuildMemberState2, Data: state2})
	if !ok || err != nil || parsedState != (GuildMemberState{AccountID: 30, CharID: 40, State: 1, HasAppearance: true, Sex: 1, HeadType: 7, HeadPalette: 8}) {
		t.Fatalf("extended member state ok=%t err=%v value=%+v", ok, err, parsedState)
	}

	location := make([]byte, 10)
	binary.LittleEndian.PutUint16(location[0:2], PacketZCGuildMemberLocation)
	binary.LittleEndian.PutUint32(location[2:6], 10)
	binary.LittleEndian.PutUint16(location[6:8], ^uint16(0))
	binary.LittleEndian.PutUint16(location[8:10], 45)
	parsedLocation, ok, err := ParseGuildMemberLocation(Packet{ID: PacketZCGuildMemberLocation, Data: location})
	if !ok || err != nil || parsedLocation.AccountID != 10 || parsedLocation.X != -1 || parsedLocation.Y != 45 {
		t.Fatalf("member location ok=%t err=%v value=%+v", ok, err, parsedLocation)
	}
}

func TestParseGuildMemberAddedUsesMemberInfoLayout(t *testing.T) {
	packet := make([]byte, 106)
	binary.LittleEndian.PutUint16(packet[0:2], PacketZCGuildMemberAdded)
	binary.LittleEndian.PutUint32(packet[2:6], 10)
	binary.LittleEndian.PutUint32(packet[6:10], 20)
	binary.LittleEndian.PutUint16(packet[10:12], 7)
	binary.LittleEndian.PutUint16(packet[16:18], 1)
	binary.LittleEndian.PutUint16(packet[18:20], 99)
	binary.LittleEndian.PutUint32(packet[24:28], 1)
	copy(packet[82:106], "New Member")

	member, ok, err := ParseGuildMemberInfo(Packet{ID: PacketZCGuildMemberAdded, Data: packet})
	if !ok || err != nil || member.AccountID != 10 || member.CharID != 20 || member.HeadType != 7 || member.Job != 1 || member.Level != 99 || member.CurrentState != 1 || member.CharName != "New Member" {
		t.Fatalf("member added ok=%t err=%v member=%+v", ok, err, member)
	}
}

func TestPacketLengths2008FramesGuildRelationResponses(t *testing.T) {
	data := make([]byte, 3+10+34+20)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCGuildHostilityResult)
	binary.LittleEndian.PutUint16(data[3:5], PacketZCGuildRelationDeleted)
	binary.LittleEndian.PutUint16(data[13:15], PacketZCGuildRelationAdded)
	binary.LittleEndian.PutUint16(data[47:49], PacketZCGuildMemberState2)
	packets, err := NewFramer(PacketLengths2008()).Push(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(packets) != 4 || packets[0].ID != PacketZCGuildHostilityResult || packets[1].ID != PacketZCGuildRelationDeleted || packets[2].ID != PacketZCGuildRelationAdded || packets[3].ID != PacketZCGuildMemberState2 {
		t.Fatalf("packets = %+v", packets)
	}
}
