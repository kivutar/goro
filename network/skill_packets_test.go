package network

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"github.com/kivutar/goro/db"
)

func TestParseSkillInfoList(t *testing.T) {
	data := make([]byte, 4+skillInfoEntryLen)
	binary.LittleEndian.PutUint16(data[0:2], 0x010F)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	writeSkillInfoEntry(data[4:], 1, 1001, 3, 14, 9, "First Aid", true)

	list, ok, err := ParseSkillInfoList(Packet{ID: 0x010F, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill list not parsed")
	}
	if len(list.Skills) != 1 {
		t.Fatalf("skill count = %d", len(list.Skills))
	}
	skill := list.Skills[0]
	if skill.ID != 1001 || skill.Type != 1 || skill.Level != 3 || skill.SPCost != 14 || skill.Range != 9 || skill.Name != "First Aid" || !skill.Upgradable {
		t.Fatalf("skill = %+v", skill)
	}
}

func TestParseSkillInfoUpdate(t *testing.T) {
	data := make([]byte, 11)
	binary.LittleEndian.PutUint16(data[0:2], 0x010E)
	binary.LittleEndian.PutUint16(data[2:4], 1001)
	binary.LittleEndian.PutUint16(data[4:6], 4)
	binary.LittleEndian.PutUint16(data[6:8], 15)
	binary.LittleEndian.PutUint16(data[8:10], 10)
	data[10] = 1

	update, ok, err := ParseSkillInfoUpdate(Packet{ID: 0x010E, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill update not parsed")
	}
	if update.Skill.ID != 1001 || update.Skill.Level != 4 || update.Skill.SPCost != 15 || update.Skill.Range != 10 || !update.Skill.Upgradable {
		t.Fatalf("update = %+v", update)
	}
}

func TestParseAddSkill(t *testing.T) {
	data := make([]byte, 39)
	binary.LittleEndian.PutUint16(data[0:2], 0x0111)
	writeSkillInfoEntry(data[2:], 0, 2, 1, 8, 1, "Heal", false)

	update, ok, err := ParseSkillInfoUpdate(Packet{ID: 0x0111, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("add skill not parsed")
	}
	if update.Skill.ID != 2 || update.Skill.Name != "Heal" || update.Skill.Upgradable {
		t.Fatalf("update = %+v", update)
	}
}

func TestParseAutoRunSkill(t *testing.T) {
	data := make([]byte, 39)
	binary.LittleEndian.PutUint16(data[0:2], 0x0147)
	writeSkillInfoEntry(data[2:], 1, 26, 1, 10, 9, "Teleportation", false)

	auto, ok, err := ParseAutoRunSkill(Packet{ID: 0x0147, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("auto-run skill not parsed")
	}
	if auto.Skill.ID != 26 || auto.Skill.Type != 1 || auto.Skill.Level != 1 || auto.Skill.SPCost != 10 || auto.Skill.Range != 9 || auto.Skill.Name != "Teleportation" {
		t.Fatalf("auto-run skill = %+v", auto.Skill)
	}
}

func TestParseMonsterInfo(t *testing.T) {
	data := make([]byte, 29)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCMonsterInfo)
	binary.LittleEndian.PutUint16(data[2:4], 1002)
	binary.LittleEndian.PutUint16(data[4:6], 17)
	binary.LittleEndian.PutUint16(data[6:8], 1)
	binary.LittleEndian.PutUint32(data[8:12], 2345)
	binary.LittleEndian.PutUint16(data[12:14], 0xFFFC)
	binary.LittleEndian.PutUint16(data[14:16], 3)
	binary.LittleEndian.PutUint16(data[16:18], uint16(int16(12)))
	binary.LittleEndian.PutUint16(data[18:20], 2)
	copy(data[20:29], []byte{100, 25, 150, 75, 90, 125, 50, 0, 200})

	info, ok, err := ParseMonsterInfo(Packet{ID: PacketZCMonsterInfo, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("monster info not parsed")
	}
	if info.Class != 1002 || info.Level != 17 || info.Size != 1 || info.HP != 2345 || info.Defense != -4 || info.Race != 3 || info.MagicDefense != 12 || info.Property != 2 {
		t.Fatalf("monster info = %+v", info)
	}
	wantElements := (MonsterElementRates{Water: 100, Earth: 25, Fire: 150, Wind: 75, Poison: 90, Holy: 125, Shadow: 50, Ghost: 0, Undead: 200})
	if info.Elements != wantElements {
		t.Fatalf("element rates = %+v, want %+v", info.Elements, wantElements)
	}
}

func TestParseMonsterInfoRejectsMalformedPacket(t *testing.T) {
	if _, ok, err := ParseMonsterInfo(Packet{ID: PacketZCMonsterInfo, Data: make([]byte, 28)}); !ok || err == nil {
		t.Fatalf("short monster info: ok=%t err=%v, want recognized error", ok, err)
	}
	if _, ok, err := ParseMonsterInfo(Packet{ID: 0x0080, Data: make([]byte, 29)}); ok || err != nil {
		t.Fatalf("unrelated packet: ok=%t err=%v", ok, err)
	}
}

func TestParseAutoSpellList(t *testing.T) {
	data := make([]byte, 30)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCAutoSpellList)
	for i, skillID := range []uint32{11, 0, 14, 19, 0, 20, 0} {
		binary.LittleEndian.PutUint32(data[2+i*4:6+i*4], skillID)
	}

	list, ok, err := ParseAutoSpellList(Packet{ID: PacketZCAutoSpellList, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("auto spell list not parsed")
	}
	want := []uint16{11, 14, 19, 20}
	if !slices.Equal(list.SkillIDs, want) {
		t.Fatalf("auto spell skills = %v, want %v", list.SkillIDs, want)
	}
}

func TestParseAutoSpellListRejectsMalformedPacket(t *testing.T) {
	if _, ok, err := ParseAutoSpellList(Packet{ID: PacketZCAutoSpellList, Data: make([]byte, 29)}); !ok || err == nil {
		t.Fatalf("short auto spell list ok=%t err=%v", ok, err)
	}

	data := make([]byte, 30)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCAutoSpellList)
	binary.LittleEndian.PutUint32(data[2:6], 0xFFFFFFFF)
	if _, ok, err := ParseAutoSpellList(Packet{ID: PacketZCAutoSpellList, Data: data}); !ok || err == nil {
		t.Fatalf("invalid auto spell skill ok=%t err=%v", ok, err)
	}
}

func TestBuildSelectAutoSpellPacket(t *testing.T) {
	packet := BuildSelectAutoSpellPacket(19)
	if len(packet) != 6 {
		t.Fatalf("packet len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[0:2]); got != PacketCZSelectAutoSpell {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if got := binary.LittleEndian.Uint32(packet[2:6]); got != 19 {
		t.Fatalf("skill id = %d", got)
	}

	cancel := BuildSelectAutoSpellPacket(0)
	if got := binary.LittleEndian.Uint32(cancel[2:6]); got != 0 {
		t.Fatalf("cancel skill id = %d", got)
	}
}

func TestBuildSelectWarpPointPacket(t *testing.T) {
	packet := BuildSelectWarpPointPacket(26, "Random")
	if len(packet) != 20 {
		t.Fatalf("packet len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[0:2]); got != 0x011B {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 26 {
		t.Fatalf("skill id = %d", got)
	}
	if got := string(packet[4:10]); got != "Random" {
		t.Fatalf("map name = %q", got)
	}
	for i, b := range packet[10:20] {
		if b != 0 {
			t.Fatalf("padding byte %d = %d", i, b)
		}
	}
}

func TestBuildRememberWarpPointPacket(t *testing.T) {
	packet := BuildRememberWarpPointPacket()
	if len(packet) != 2 {
		t.Fatalf("packet len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[0:2]); got != 0x011D {
		t.Fatalf("opcode = 0x%04X", got)
	}
}

func TestParseRememberWarpPointAck(t *testing.T) {
	data := []byte{0x1E, 0x01, 0x02}
	ack, ok, err := ParseRememberWarpPointAck(Packet{ID: 0x011E, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("remember warp point ack not parsed")
	}
	if ack.Result != 2 {
		t.Fatalf("result = %d", ack.Result)
	}
}

func TestParseWarpPointList(t *testing.T) {
	for _, tc := range []struct {
		name      string
		packetID  uint16
		extension string
	}{
		{"classic", 0x011C, ""},
		{"classic with extensions", 0x011C, ".gat"},
		{"variable", 0x0ABE, ""},
		{"variable with extensions", 0x0ABE, ".gat"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 68)
			skillOffset := 2
			if tc.packetID == 0x0ABE {
				data = make([]byte, 38)
				binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
				skillOffset = 4
			}
			binary.LittleEndian.PutUint16(data[0:2], tc.packetID)
			binary.LittleEndian.PutUint16(data[skillOffset:skillOffset+2], 26)
			namesOffset := skillOffset + 2
			copy(data[namesOffset:namesOffset+16], "Random"+tc.extension)
			copy(data[namesOffset+16:namesOffset+32], "prontera"+tc.extension)

			list, ok, err := ParseWarpPointList(Packet{ID: tc.packetID, Data: data})
			if err != nil || !ok {
				t.Fatalf("parse warp list: ok=%t err=%v", ok, err)
			}
			if list.SkillID != 26 || len(list.MapNames) != 2 || list.MapNames[0] != "Random" || list.MapNames[1] != "prontera" {
				t.Fatalf("warp point list = %+v", list)
			}
		})
	}
}

func TestParseSkillNoDamageNotify(t *testing.T) {
	tests := []struct {
		name string
		id   uint16
		data []byte
	}{
		{
			name: "011A",
			id:   0x011A,
			data: func() []byte {
				data := make([]byte, 15)
				binary.LittleEndian.PutUint16(data[0:2], 0x011A)
				binary.LittleEndian.PutUint16(data[2:4], 6)
				binary.LittleEndian.PutUint16(data[4:6], 2)
				binary.LittleEndian.PutUint32(data[6:10], 0x11223344)
				binary.LittleEndian.PutUint32(data[10:14], 0x55667788)
				data[14] = 1
				return data
			}(),
		},
		{
			name: "09CB",
			id:   0x09CB,
			data: func() []byte {
				data := make([]byte, 17)
				binary.LittleEndian.PutUint16(data[0:2], 0x09CB)
				binary.LittleEndian.PutUint16(data[2:4], 6)
				binary.LittleEndian.PutUint32(data[4:8], 2)
				binary.LittleEndian.PutUint32(data[8:12], 0x11223344)
				binary.LittleEndian.PutUint32(data[12:16], 0x55667788)
				data[16] = 1
				return data
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notify, ok, err := ParseSkillNoDamageNotify(Packet{ID: tt.id, Data: tt.data})
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("skill notification not parsed")
			}
			if notify.SkillID != 6 || notify.Amount != 2 || notify.TargetID != 0x11223344 || notify.SourceID != 0x55667788 || notify.Result != 1 {
				t.Fatalf("notify = %+v", notify)
			}
		})
	}
}

func TestParseSkillCastNotify(t *testing.T) {
	data := make([]byte, 24)
	binary.LittleEndian.PutUint16(data[0:2], 0x013E)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)
	binary.LittleEndian.PutUint32(data[6:10], 0x55667788)
	binary.LittleEndian.PutUint16(data[10:12], 123)
	binary.LittleEndian.PutUint16(data[12:14], 456)
	binary.LittleEndian.PutUint16(data[14:16], 20)
	binary.LittleEndian.PutUint32(data[16:20], 4)
	binary.LittleEndian.PutUint32(data[20:24], 2500)

	notify, ok, err := ParseSkillCastNotify(Packet{ID: 0x013E, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill cast notification not parsed")
	}
	if notify.SourceID != 0x11223344 || notify.TargetID != 0x55667788 || notify.X != 123 || notify.Y != 456 || notify.SkillID != 20 || notify.Property != 4 || notify.DelayTime != 2500 {
		t.Fatalf("notify = %+v", notify)
	}
}

func TestParseSkillCastCancel(t *testing.T) {
	packet := Packet{ID: PacketZCSkillCastCancel, Data: []byte{0xB9, 0x01, 0x44, 0x33, 0x22, 0x11}}
	if id, ok, err := ParseSkillCastCancel(packet); err != nil || !ok || id != 0x11223344 {
		t.Fatalf("cast cancellation: id=%d ok=%v err=%v", id, ok, err)
	}
	if _, ok, err := ParseSkillCastCancel(Packet{ID: 0x013E}); ok || err != nil {
		t.Fatalf("unrelated packet: ok=%v err=%v", ok, err)
	}
	for size := 0; size < 6; size++ {
		if _, ok, err := ParseSkillCastCancel(Packet{ID: PacketZCSkillCastCancel, Data: packet.Data[:size]}); ok || err == nil {
			t.Errorf("truncated size %d: ok=%v err=%v", size, ok, err)
		}
	}
}

func TestParseGroundSkillNotify(t *testing.T) {
	data := make([]byte, 18)
	binary.LittleEndian.PutUint16(data[0:2], 0x0117)
	binary.LittleEndian.PutUint16(data[2:4], 21)
	binary.LittleEndian.PutUint32(data[4:8], 0x11223344)
	binary.LittleEndian.PutUint16(data[8:10], 4)
	binary.LittleEndian.PutUint16(data[10:12], 123)
	binary.LittleEndian.PutUint16(data[12:14], 456)

	notify, ok, err := ParseGroundSkillNotify(Packet{ID: 0x0117, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ground skill notification not parsed")
	}
	if notify.SkillID != 21 || notify.SourceID != 0x11223344 || notify.Level != 4 || notify.X != 123 || notify.Y != 456 {
		t.Fatalf("notify = %+v", notify)
	}
}

func TestParseSkillUnitEntry(t *testing.T) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint16(data[0:2], 0x011F)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)
	binary.LittleEndian.PutUint32(data[6:10], 0x55667788)
	binary.LittleEndian.PutUint16(data[10:12], 123)
	binary.LittleEndian.PutUint16(data[12:14], 456)
	data[14] = 126
	data[15] = 1

	entry, ok, err := ParseSkillUnitEntry(Packet{ID: 0x011F, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill unit entry not parsed")
	}
	if entry.ID != 0x11223344 || entry.CreatorID != 0x55667788 || entry.X != 123 || entry.Y != 456 || entry.UnitID != 126 || !entry.Visible {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestParseSkillUnitDisappear(t *testing.T) {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[0:2], 0x0120)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)

	disappear, ok, err := ParseSkillUnitDisappear(Packet{ID: 0x0120, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill unit disappear not parsed")
	}
	if disappear.ID != 0x11223344 {
		t.Fatalf("disappear = %+v", disappear)
	}
}

func TestParseGraffitiEntry(t *testing.T) {
	data := make([]byte, 97)
	binary.LittleEndian.PutUint32(data[2:6], 1234)
	binary.LittleEndian.PutUint32(data[6:10], 2000000)
	binary.LittleEndian.PutUint16(data[10:12], 120)
	binary.LittleEndian.PutUint16(data[12:14], 80)
	data[14], data[15], data[16] = 176, 1, 1
	copy(data[17:], "Hello Midgard!\x00unused")
	entry, ok, err := ParseSkillUnitEntry(Packet{ID: 0x01C9, Data: data})
	if err != nil || !ok || entry.ID != 1234 || entry.CreatorID != 2000000 ||
		entry.X != 120 || entry.Y != 80 || entry.UnitID != 176 || !entry.Visible || !entry.HasMessage || entry.Message != "Hello Midgard!" {
		t.Fatalf("graffiti entry = %+v, ok=%t err=%v", entry, ok, err)
	}
	for size := 0; size < 97; size++ {
		if _, _, err := ParseSkillUnitEntry(Packet{ID: 0x01C9, Data: data[:size]}); err == nil {
			t.Fatalf("accepted truncated graffiti packet of length %d", size)
		}
	}
	data[16] = 0
	entry, ok, err = ParseSkillUnitEntry(Packet{ID: 0x01C9, Data: data})
	if err != nil || !ok || entry.HasMessage || entry.ID != 1234 {
		t.Fatalf("extended entry without message = %+v, %v", entry, err)
	}
	data[16] = 1
	copy(data[17:], "  Hi  \x00")
	entry, _, _ = ParseSkillUnitEntry(Packet{ID: 0x01C9, Data: data})
	if entry.Message != "  Hi  " {
		t.Fatal("graffiti layout must preserve spaces")
	}
}

func TestParseSkillUnitUpdate(t *testing.T) {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[0:2], 0x01AC)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)

	update, ok, err := ParseSkillUnitUpdate(Packet{ID: 0x01AC, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill unit update not parsed")
	}
	if update.ID != 0x11223344 {
		t.Fatalf("update = %+v", update)
	}
}

func TestParseSkillFailAck(t *testing.T) {
	data := make([]byte, 10)
	binary.LittleEndian.PutUint16(data[0:2], 0x0110)
	binary.LittleEndian.PutUint16(data[2:4], 6)
	binary.LittleEndian.PutUint16(data[4:6], 2)
	binary.LittleEndian.PutUint16(data[6:8], 501)
	data[8] = 0
	data[9] = 10

	ack, ok, err := ParseSkillFailAck(Packet{ID: 0x0110, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill fail ack not parsed")
	}
	if ack.SkillID != 6 || ack.Number != 2 || ack.ItemID != 501 || ack.Result != 0 || ack.Cause != 10 {
		t.Fatalf("ack = %+v", ack)
	}
}

func TestBuildSkillLevelUpPacket(t *testing.T) {
	packet := BuildSkillLevelUpPacket(1001)
	if got := ID(packet); got != 0x0112 {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if len(packet) != 4 {
		t.Fatalf("len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 1001 {
		t.Fatalf("skill id = %d", got)
	}
}

func TestBuildUseSkillToIDPacketForClientDate20080910(t *testing.T) {
	packet := BuildUseSkillToIDPacketForClientDate(5, 3, 0x11223344, 20080910)
	if got := ID(packet); got != 0x0438 {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if len(packet) != 10 {
		t.Fatalf("len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 3 {
		t.Fatalf("level = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[4:6]); got != 5 {
		t.Fatalf("skill = %d", got)
	}
	if got := binary.LittleEndian.Uint32(packet[6:10]); got != 0x11223344 {
		t.Fatalf("target = 0x%08X", got)
	}
}

func TestBuildWeddingSkillToIDPacketForClientDate20080910(t *testing.T) {
	packet := BuildUseSkillToIDPacketForClientDate(db.SkillWECallpartner, 1, 0x11223344, 20080910)
	if got := ID(packet); got != 0x0438 {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if len(packet) != 10 {
		t.Fatalf("len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 1 {
		t.Fatalf("level = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[4:6]); got != db.SkillWECallpartner {
		t.Fatalf("skill = %d", got)
	}
	if got := binary.LittleEndian.Uint32(packet[6:10]); got != 0x11223344 {
		t.Fatalf("target = 0x%08X", got)
	}
}

func TestBuildChangeCartPacket(t *testing.T) {
	packet := BuildChangeCartPacket(4)
	if len(packet) != 4 || ID(packet) != 0x01AF || binary.LittleEndian.Uint16(packet[2:4]) != 4 {
		t.Fatalf("unexpected change cart packet: % X", packet)
	}
}

func TestBuildUseSkillToGroundPacketForClientDate20080910(t *testing.T) {
	packet := BuildUseSkillToGroundPacketForClientDate(21, 4, 123, 456, 20080910)
	if got := ID(packet); got != 0x0113 {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if len(packet) != 22 {
		t.Fatalf("len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[5:7]); got != 4 {
		t.Fatalf("level = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[9:11]); got != 21 {
		t.Fatalf("skill = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[12:14]); got != 123 {
		t.Fatalf("x = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[20:22]); got != 456 {
		t.Fatalf("y = %d", got)
	}
}

func TestBuildUseSkillToGroundWithTextPacketForClientDate20080910(t *testing.T) {
	packet := BuildUseSkillToGroundWithTextPacketForClientDate(125, 4, 123, 456, "hello", 20080910)
	if got := ID(packet); got != 0x007E {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if len(packet) != 102 {
		t.Fatalf("len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[5:7]); got != 4 {
		t.Fatalf("level = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[9:11]); got != 125 {
		t.Fatalf("skill = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[12:14]); got != 123 {
		t.Fatalf("x = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[20:22]); got != 456 {
		t.Fatalf("y = %d", got)
	}
	if got := string(packet[22:27]); got != "hello" {
		t.Fatalf("text = %q", got)
	}
	if packet[27] != 0 {
		t.Fatalf("text was not nul-terminated")
	}
}

func TestBuildUseSkillToGroundWithTextPacketTruncatesToMessageSize(t *testing.T) {
	packet := BuildUseSkillToGroundWithTextPacketForClientDate(125, 4, 123, 456, strings.Repeat("a", 90), 20080910)
	message := packet[22 : 22+skillGroundMessageLen]
	if got := len(packetCString(message)); got != 79 {
		t.Fatalf("message len = %d, want 79", got)
	}
	if message[79] != 0 {
		t.Fatal("message was not nul-terminated at byte 79")
	}
}

func TestPacketLengths2008IncludesUseSkillToGroundWithText(t *testing.T) {
	if got := PacketLengths2008()[0x007E]; got != 102 {
		t.Fatalf("0x007E len = %d, want 102", got)
	}
}

func TestBuildUseSkillToGroundPacketLegacy(t *testing.T) {
	packet := BuildUseSkillToGroundPacketForClientDate(21, 4, 123, 456, 20070101)
	if got := ID(packet); got != 0x0116 {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if len(packet) != 10 {
		t.Fatalf("len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 4 {
		t.Fatalf("level = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[4:6]); got != 21 {
		t.Fatalf("skill = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[6:8]); got != 123 {
		t.Fatalf("x = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[8:10]); got != 456 {
		t.Fatalf("y = %d", got)
	}
}

func TestBuildUseSkillToGroundPacketForClientDate20120307(t *testing.T) {
	packet := BuildUseSkillToGroundPacketForClientDate(21, 4, 123, 456, 20120307)
	if got := ID(packet); got != 0x0438 {
		t.Fatalf("opcode = 0x%04X", got)
	}
	if len(packet) != 10 {
		t.Fatalf("len = %d", len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != 4 {
		t.Fatalf("level = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[4:6]); got != 21 {
		t.Fatalf("skill = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[6:8]); got != 123 {
		t.Fatalf("x = %d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[8:10]); got != 456 {
		t.Fatalf("y = %d", got)
	}
}

func writeSkillInfoEntry(data []byte, typ uint32, id uint16, level, sp, attackRange int, name string, upgradable bool) {
	binary.LittleEndian.PutUint16(data[0:2], id)
	binary.LittleEndian.PutUint32(data[2:6], typ)
	binary.LittleEndian.PutUint16(data[6:8], uint16(level))
	binary.LittleEndian.PutUint16(data[8:10], uint16(sp))
	binary.LittleEndian.PutUint16(data[10:12], uint16(attackRange))
	copy(data[12:36], []byte(name))
	if upgradable {
		data[36] = 1
	}
}
