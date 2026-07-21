package network

import (
	"encoding/binary"
	"testing"
)

func TestBuildCompanionPackets(t *testing.T) {
	command := BuildHomunculusCommandPacketWithType(PacketZCFeedMercenary, HomunculusCommandFeed)
	if ID(command) != PacketCZCommandMercenary || len(command) != 5 {
		t.Fatalf("homunculus command packet id=0x%04x len=%d", ID(command), len(command))
	}
	if binary.LittleEndian.Uint16(command[2:4]) != PacketZCFeedMercenary || command[4] != HomunculusCommandFeed {
		t.Fatalf("homunculus command payload = %x", command)
	}

	move, ok := BuildCompanionMovePacket(0x11223344, 150, 200)
	if !ok {
		t.Fatal("BuildCompanionMovePacket returned !ok")
	}
	if ID(move) != PacketCZRequestMoveNPC || len(move) != 9 || binary.LittleEndian.Uint32(move[2:6]) != 0x11223344 {
		t.Fatalf("companion move packet = %x", move)
	}
	dest, _ := EncodeMoveDestination(150, 200)
	if string(move[6:9]) != string(dest[:]) {
		t.Fatalf("companion move dest = %x, want %x", move[6:9], dest)
	}

	attack := BuildCompanionAttackPacket(0x11223344, 0x55667788, 0)
	if ID(attack) != PacketCZRequestActNPC || len(attack) != 11 {
		t.Fatalf("companion attack packet id=0x%04x len=%d", ID(attack), len(attack))
	}
	if binary.LittleEndian.Uint32(attack[2:6]) != 0x11223344 || binary.LittleEndian.Uint32(attack[6:10]) != 0x55667788 || attack[10] != 0 {
		t.Fatalf("companion attack payload = %x", attack)
	}

	follow := BuildCompanionMoveToOwnerPacket(0x11223344)
	if ID(follow) != PacketCZRequestMoveToOwner || len(follow) != 6 || binary.LittleEndian.Uint32(follow[2:6]) != 0x11223344 {
		t.Fatalf("companion follow packet = %x", follow)
	}

	merc := BuildMercenaryCommandPacket(MercenaryCommandDelete)
	if ID(merc) != PacketCZMercenaryCommand || len(merc) != 3 || merc[2] != MercenaryCommandDelete {
		t.Fatalf("mercenary command packet = %x", merc)
	}
}

func TestParseHomunculusPackets(t *testing.T) {
	data := make([]byte, 71)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCPropertyHomunculus)
	copy(data[2:26], []byte("Lif"))
	data[26] = 3
	binary.LittleEndian.PutUint16(data[27:29], 45)
	binary.LittleEndian.PutUint16(data[29:31], 62)
	binary.LittleEndian.PutUint16(data[31:33], 911)
	binary.LittleEndian.PutUint16(data[51:53], 1234)
	binary.LittleEndian.PutUint16(data[53:55], 2345)
	binary.LittleEndian.PutUint16(data[55:57], 98)
	binary.LittleEndian.PutUint16(data[57:59], 210)
	binary.LittleEndian.PutUint32(data[59:63], 3456)
	binary.LittleEndian.PutUint32(data[63:67], 4567)
	binary.LittleEndian.PutUint16(data[67:69], 4)
	binary.LittleEndian.PutUint16(data[69:71], 7)

	property, ok, err := ParseHomunculusProperty(Packet{ID: PacketZCPropertyHomunculus, Data: data})
	if err != nil || !ok {
		t.Fatalf("ParseHomunculusProperty ok=%t err=%v", ok, err)
	}
	if property.Name != "Lif" || property.Flags != 3 || property.Level != 45 || property.Hunger != 62 || property.Intimacy != 911 {
		t.Fatalf("homunculus property identity = %+v", property)
	}
	if property.HP != 1234 || property.MaxHP != 2345 || property.SP != 98 || property.MaxSP != 210 || property.Exp != 3456 || property.MaxExp != 4567 || property.SkillPoints != 4 || property.AttackRange != 7 {
		t.Fatalf("homunculus property life/range = %+v", property)
	}

	stateData := make([]byte, 12)
	binary.LittleEndian.PutUint16(stateData[0:2], PacketZCChangeStateMercenary)
	stateData[2] = 0
	stateData[3] = 2
	binary.LittleEndian.PutUint32(stateData[4:8], 0x11223344)
	binary.LittleEndian.PutUint32(stateData[8:12], 76)
	state, ok, err := ParseHomunculusStateChange(Packet{ID: PacketZCChangeStateMercenary, Data: stateData})
	if err != nil || !ok {
		t.Fatalf("ParseHomunculusStateChange ok=%t err=%v", ok, err)
	}
	if state.GID != 0x11223344 || state.State != 2 || state.Data != 76 {
		t.Fatalf("homunculus state = %+v", state)
	}
}

func TestParseMercenaryPackets(t *testing.T) {
	data := make([]byte, 80)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCMercenaryInit)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)
	binary.LittleEndian.PutUint16(data[6:8], 101)
	copy(data[22:46], []byte("Swordsman"))
	binary.LittleEndian.PutUint16(data[46:48], 5)
	binary.LittleEndian.PutUint32(data[48:52], 1234)
	binary.LittleEndian.PutUint32(data[52:56], 2345)
	binary.LittleEndian.PutUint32(data[56:60], 98)
	binary.LittleEndian.PutUint32(data[60:64], 210)
	binary.LittleEndian.PutUint32(data[64:68], 777)
	binary.LittleEndian.PutUint16(data[68:70], 42)
	binary.LittleEndian.PutUint32(data[70:74], 11)
	binary.LittleEndian.PutUint32(data[74:78], 12)
	binary.LittleEndian.PutUint16(data[78:80], 3)

	property, ok, err := ParseMercenaryProperty(Packet{ID: PacketZCMercenaryInit, Data: data})
	if err != nil || !ok {
		t.Fatalf("ParseMercenaryProperty ok=%t err=%v", ok, err)
	}
	if property.ID != 0x11223344 || property.Name != "Swordsman" || property.Level != 5 || property.Attack != 101 {
		t.Fatalf("mercenary property identity = %+v", property)
	}
	if property.HP != 1234 || property.MaxHP != 2345 || property.SP != 98 || property.MaxSP != 210 || property.ExpireTick != 777 || property.Faith != 42 || property.Calls != 11 || property.Kills != 12 || property.AttackRange != 3 {
		t.Fatalf("mercenary property life/range = %+v", property)
	}

	paramData := make([]byte, 8)
	binary.LittleEndian.PutUint16(paramData[0:2], PacketZCMercenaryParamChange)
	binary.LittleEndian.PutUint16(paramData[2:4], StatusHP)
	binary.LittleEndian.PutUint32(paramData[4:8], 4321)
	change, ok, err := ParseMercenaryParamChange(Packet{ID: PacketZCMercenaryParamChange, Data: paramData})
	if err != nil || !ok {
		t.Fatalf("ParseMercenaryParamChange ok=%t err=%v", ok, err)
	}
	if change.Param != StatusHP || change.Value != 4321 {
		t.Fatalf("mercenary param = %+v", change)
	}
}
