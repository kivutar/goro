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
	deleteCommand := BuildHomunculusCommandPacketWithType(PacketZCFeedMercenary, HomunculusCommandDelete)
	if binary.LittleEndian.Uint16(deleteCommand[2:4]) != PacketZCFeedMercenary || deleteCommand[4] != HomunculusCommandDelete {
		t.Fatalf("homunculus delete command payload = %x", deleteCommand)
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

func TestParseHomunculusPropertyVariantsMatchRobrowserLayouts(t *testing.T) {
	tests := []struct {
		name   string
		id     uint16
		hp     int
		maxHP  int
		sp     int
		maxSP  int
		exp    uint64
		maxExp uint64
	}{
		{"property2", PacketZCPropertyHomunculus2, 70000, 80000, 300, 400, 123456, 999999},
		{"property3", PacketZCPropertyHomunculus3, 70001, 80001, 301, 401, 123457, 999998},
		{"property4", PacketZCPropertyHomunculus4, 70002, 80002, 50000, 60000, 123458, 999997},
		{"property5", PacketZCPropertyHomunculus5, 70003, 80003, 50001, 60001, 5_000_000_000, 6_000_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			property, ok, err := ParseHomunculusProperty(Packet{
				ID:   tt.id,
				Data: buildHomunculusPropertyTestPacket(t, tt.id, tt.hp, tt.maxHP, tt.sp, tt.maxSP, tt.exp, tt.maxExp),
			})
			if err != nil || !ok {
				t.Fatalf("ParseHomunculusProperty ok=%t err=%v", ok, err)
			}
			if property.Name != "Lif" || property.Level != 45 || property.Hunger != 62 || property.Intimacy != 911 {
				t.Fatalf("identity = %+v", property)
			}
			if property.HP != tt.hp || property.MaxHP != tt.maxHP || property.SP != tt.sp || property.MaxSP != tt.maxSP || property.Exp != tt.exp || property.MaxExp != tt.maxExp || property.SkillPoints != 4 || property.AttackRange != 7 {
				t.Fatalf("life/range = %+v, want hp=%d/%d sp=%d/%d exp=%d/%d", property, tt.hp, tt.maxHP, tt.sp, tt.maxSP, tt.exp, tt.maxExp)
			}
		})
	}
}

func TestParseHomunculusParamChangePackets(t *testing.T) {
	paramData := make([]byte, 8)
	binary.LittleEndian.PutUint16(paramData[0:2], PacketZCHomunculusParamChange)
	binary.LittleEndian.PutUint16(paramData[2:4], StatusHP)
	binary.LittleEndian.PutUint32(paramData[4:8], 4321)
	change, ok, err := ParseHomunculusParamChange(Packet{ID: PacketZCHomunculusParamChange, Data: paramData})
	if err != nil || !ok {
		t.Fatalf("ParseHomunculusParamChange ok=%t err=%v", ok, err)
	}
	if change.Param != StatusHP || change.Value != 4321 {
		t.Fatalf("homunculus param = %+v", change)
	}

	paramData2 := make([]byte, 12)
	binary.LittleEndian.PutUint16(paramData2[0:2], PacketZCHomunculusParamChange2)
	binary.LittleEndian.PutUint16(paramData2[2:4], StatusBaseExp)
	binary.LittleEndian.PutUint64(paramData2[4:12], 5_000_000_000)
	change, ok, err = ParseHomunculusParamChange(Packet{ID: PacketZCHomunculusParamChange2, Data: paramData2})
	if err != nil || !ok {
		t.Fatalf("ParseHomunculusParamChange2 ok=%t err=%v", ok, err)
	}
	if change.Param != StatusBaseExp || change.Value != 5_000_000_000 {
		t.Fatalf("homunculus param2 = %+v", change)
	}
}

func TestPacketLengths2008IncludesHomunculusRobrowserVariants(t *testing.T) {
	lengths := PacketLengths2008()
	for _, tt := range []struct {
		id   uint16
		want int
	}{
		{PacketZCHomunculusParamChange, 8},
		{PacketZCPropertyHomunculus2, 75},
		{PacketZCPropertyHomunculus3, 73},
		{PacketZCPropertyHomunculus4, 77},
		{PacketZCPropertyHomunculus5, 85},
		{PacketZCHomunculusParamChange2, 12},
	} {
		if got := lengths[tt.id]; got != tt.want {
			t.Fatalf("packet 0x%04X length = %d, want %d", tt.id, got, tt.want)
		}
	}
}

func buildHomunculusPropertyTestPacket(t *testing.T, id uint16, hp, maxHP, sp, maxSP int, exp, maxExp uint64) []byte {
	t.Helper()
	layout, ok := homunculusPropertyLayoutForPacket(id)
	if !ok {
		t.Fatalf("unknown homunculus property id 0x%04X", id)
	}
	data := make([]byte, layout.size)
	binary.LittleEndian.PutUint16(data[0:2], id)
	offset := 2
	copy(data[offset:offset+24], []byte("Lif"))
	offset += 24
	data[offset] = 3
	offset++
	binary.LittleEndian.PutUint16(data[offset:offset+2], 45)
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:offset+2], 62)
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:offset+2], 911)
	offset += 2
	if layout.hasItemID {
		binary.LittleEndian.PutUint16(data[offset:offset+2], 607)
		offset += 2
	}
	for stat := 100; stat < 108; stat++ {
		binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(stat))
		offset += 2
	}
	offset = putHomunculusSizedInt(data, offset, hp, layout.hp32)
	offset = putHomunculusSizedInt(data, offset, maxHP, layout.hp32)
	offset = putHomunculusSizedInt(data, offset, sp, layout.sp32)
	offset = putHomunculusSizedInt(data, offset, maxSP, layout.sp32)
	if layout.exp64 {
		binary.LittleEndian.PutUint64(data[offset:offset+8], exp)
		offset += 8
		binary.LittleEndian.PutUint64(data[offset:offset+8], maxExp)
		offset += 8
	} else {
		binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(exp))
		offset += 4
		binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(maxExp))
		offset += 4
	}
	binary.LittleEndian.PutUint16(data[offset:offset+2], 4)
	offset += 2
	binary.LittleEndian.PutUint16(data[offset:offset+2], 7)
	return data
}

func putHomunculusSizedInt(data []byte, offset, value int, long bool) int {
	if long {
		binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(value))
		return offset + 4
	}
	binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(value))
	return offset + 2
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
