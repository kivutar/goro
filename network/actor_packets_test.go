package network

import (
	"encoding/binary"
	"testing"
)

func TestParseActorStandEntry2(t *testing.T) {
	data := make([]byte, 54)
	binary.LittleEndian.PutUint16(data[0:2], 0x01D8)
	binary.LittleEndian.PutUint32(data[2:6], 2000001)
	binary.LittleEndian.PutUint16(data[6:8], 400)
	binary.LittleEndian.PutUint16(data[14:16], 1002)
	binary.LittleEndian.PutUint16(data[16:18], 7)
	data[45] = 1
	data[46], data[47], data[48] = packPosition(102, 134, 3)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x01D8, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not parsed")
	}
	if !entry.Appearance {
		t.Fatal("stand entry should include appearance")
	}
	if entry.ID != 2000001 || entry.Speed != 400 || entry.Job != 1002 || entry.Head != 7 || entry.Sex != 1 || entry.X != 102 || entry.Y != 134 || entry.Dir != 3 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}

func TestParseActorStandEntryLegacy(t *testing.T) {
	data := make([]byte, 55)
	binary.LittleEndian.PutUint16(data[0:2], 0x0078)
	data[2] = 5
	binary.LittleEndian.PutUint32(data[3:7], 2000003)
	binary.LittleEndian.PutUint16(data[7:9], 420)
	binary.LittleEndian.PutUint16(data[9:11], 2)
	binary.LittleEndian.PutUint16(data[11:13], 0x0010)
	binary.LittleEndian.PutUint16(data[13:15], 0x0040)
	binary.LittleEndian.PutUint16(data[15:17], 1011)
	binary.LittleEndian.PutUint16(data[17:19], 2)
	binary.LittleEndian.PutUint16(data[19:21], 3)
	binary.LittleEndian.PutUint16(data[21:23], 4)
	binary.LittleEndian.PutUint16(data[23:25], 5)
	binary.LittleEndian.PutUint16(data[25:27], 6)
	binary.LittleEndian.PutUint16(data[27:29], 7)
	binary.LittleEndian.PutUint16(data[29:31], 8)
	binary.LittleEndian.PutUint16(data[31:33], 9)
	binary.LittleEndian.PutUint16(data[33:35], 2)
	binary.LittleEndian.PutUint32(data[35:39], 0x01020304)
	binary.LittleEndian.PutUint16(data[39:41], 11)
	data[46] = 0
	data[47], data[48], data[49] = packPosition(44, 55, 6)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x0078, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not parsed")
	}
	if entry.ID != 2000003 || entry.Speed != 420 || entry.Job != 1011 || entry.ObjectType != 5 || entry.X != 44 || entry.Y != 55 || entry.Dir != 6 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if !entry.HasState || entry.BodyState != 2 || entry.HealthState != 0x0010 || entry.EffectState != 0x0040 {
		t.Fatalf("unexpected state: %+v", entry)
	}
	if entry.Weapon != 3 || entry.HeadLow != 4 || entry.Shield != 5 || entry.HeadTop != 6 || entry.HeadMid != 7 || entry.HeadPal != 8 || entry.BodyPal != 9 || entry.HeadDir != 2 {
		t.Fatalf("unexpected appearance: %+v", entry)
	}
	if entry.GuildID != 0x01020304 || entry.EmblemVersion != 11 {
		t.Fatalf("unexpected guild identity: %+v", entry)
	}
}

func TestParseGuildFlagSpawnEntry2008(t *testing.T) {
	data := make([]byte, 42)
	binary.LittleEndian.PutUint16(data[0:2], 0x007C)
	data[2] = 6
	binary.LittleEndian.PutUint32(data[3:7], 110005864)
	binary.LittleEndian.PutUint16(data[21:23], uint16(legacyGuildFlagJob))
	binary.LittleEndian.PutUint16(data[23:25], 4)
	binary.LittleEndian.PutUint16(data[25:27], 0x0102)
	binary.LittleEndian.PutUint16(data[27:29], 0x0304)
	data[37], data[38], data[39] = packPosition(155, 190, 4)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x007C, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not parsed")
	}
	if entry.Job != legacyGuildFlagJob || entry.GuildID != 0x01020304 || entry.EmblemVersion != 4 {
		t.Fatalf("unexpected guild flag identity: %+v", entry)
	}
	if entry.X != 155 || entry.Y != 190 || entry.Dir != 4 {
		t.Fatalf("unexpected guild flag position: %+v", entry)
	}
}

func TestParseActorMoveEntry2(t *testing.T) {
	data := make([]byte, 60)
	binary.LittleEndian.PutUint16(data[0:2], 0x01DA)
	binary.LittleEndian.PutUint32(data[2:6], 2000002)
	binary.LittleEndian.PutUint16(data[6:8], 480)
	binary.LittleEndian.PutUint16(data[14:16], 1002)
	binary.LittleEndian.PutUint32(data[24:28], 654321)
	data[49] = 0
	data[50], data[51], data[52], data[53], data[54], data[55] = packMovePosition(10, 20, 30, 40)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x01DA, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !entry.Moving {
		t.Fatalf("move not parsed: ok=%v entry=%+v", ok, entry)
	}
	if !entry.Appearance {
		t.Fatal("move entry should include appearance")
	}
	if entry.FromX != 10 || entry.FromY != 20 || entry.ToX != 30 || entry.ToY != 40 || entry.X != 30 || entry.Y != 40 {
		t.Fatalf("unexpected move entry: %+v", entry)
	}
	if entry.Speed != 480 {
		t.Fatalf("speed = %d, want 480", entry.Speed)
	}
	if !entry.HasMoveStartTick || entry.MoveStartTick != 654321 {
		t.Fatalf("move start tick = %d has=%t, want 654321 true", entry.MoveStartTick, entry.HasMoveStartTick)
	}
}

func TestParseActorStandEntry2008(t *testing.T) {
	data := make([]byte, 60)
	binary.LittleEndian.PutUint16(data[0:2], 0x02EE)
	binary.LittleEndian.PutUint32(data[2:6], 2000005)
	binary.LittleEndian.PutUint16(data[6:8], 430)
	binary.LittleEndian.PutUint16(data[8:10], 1)
	binary.LittleEndian.PutUint16(data[10:12], 2)
	binary.LittleEndian.PutUint32(data[12:16], 0x00000408)
	binary.LittleEndian.PutUint16(data[16:18], 5)
	binary.LittleEndian.PutUint16(data[18:20], 3)
	binary.LittleEndian.PutUint32(data[20:24], uint32(1201)|uint32(2101)<<16)
	binary.LittleEndian.PutUint16(data[24:26], 11)
	binary.LittleEndian.PutUint16(data[26:28], 22)
	binary.LittleEndian.PutUint16(data[28:30], 33)
	binary.LittleEndian.PutUint16(data[30:32], 8)
	binary.LittleEndian.PutUint16(data[32:34], 6)
	binary.LittleEndian.PutUint16(data[34:36], 4)
	binary.LittleEndian.PutUint32(data[36:40], 0x01020304)
	binary.LittleEndian.PutUint16(data[40:42], 7)
	binary.LittleEndian.PutUint16(data[56:58], 99)
	data[49] = 1
	data[50], data[51], data[52] = packPosition(120, 140, 5)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x02EE, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !entry.Appearance {
		t.Fatalf("stand entry not parsed: ok=%v entry=%+v", ok, entry)
	}
	if entry.ID != 2000005 || entry.Speed != 430 || entry.Job != 5 || entry.Head != 3 || entry.Weapon != 1201 || entry.Shield != 2101 || entry.HeadLow != 11 || entry.HeadTop != 22 || entry.HeadMid != 33 || entry.HeadPal != 8 || entry.BodyPal != 6 || entry.HeadDir != 4 || entry.Sex != 1 || entry.X != 120 || entry.Y != 140 || entry.Dir != 5 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if !entry.HasState || entry.BodyState != 1 || entry.HealthState != 2 || entry.EffectState != 0x00000408 {
		t.Fatalf("unexpected state: %+v", entry)
	}
	if !entry.HasLevel || entry.Level != 99 {
		t.Fatalf("level = %d has=%t, want 99 true", entry.Level, entry.HasLevel)
	}
	if entry.GuildID != 0x01020304 || entry.EmblemVersion != 7 {
		t.Fatalf("unexpected guild emblem fields: %+v", entry)
	}
}

func TestParseActorNewEntry2008Level(t *testing.T) {
	data := make([]byte, 59)
	binary.LittleEndian.PutUint16(data[0:2], 0x02ED)
	binary.LittleEndian.PutUint32(data[2:6], 2000007)
	binary.LittleEndian.PutUint16(data[6:8], 430)
	binary.LittleEndian.PutUint16(data[16:18], 5)
	binary.LittleEndian.PutUint16(data[18:20], 3)
	binary.LittleEndian.PutUint16(data[55:57], 99)
	data[49] = 1
	data[50], data[51], data[52] = packPosition(120, 140, 5)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x02ED, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !entry.Appearance {
		t.Fatalf("new entry not parsed: ok=%v entry=%+v", ok, entry)
	}
	if entry.ID != 2000007 || entry.X != 120 || entry.Y != 140 || !entry.HasLevel || entry.Level != 99 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}

func TestParseActorMoveEntry2008Palettes(t *testing.T) {
	data := make([]byte, 67)
	binary.LittleEndian.PutUint16(data[0:2], 0x02EC)
	data[2] = 0
	binary.LittleEndian.PutUint32(data[3:7], 2000006)
	binary.LittleEndian.PutUint16(data[7:9], 450)
	binary.LittleEndian.PutUint16(data[17:19], 5)
	binary.LittleEndian.PutUint16(data[19:21], 4)
	binary.LittleEndian.PutUint32(data[21:25], uint32(1201)|uint32(2101)<<16)
	binary.LittleEndian.PutUint16(data[25:27], 12)
	binary.LittleEndian.PutUint32(data[27:31], 234567)
	binary.LittleEndian.PutUint16(data[31:33], 23)
	binary.LittleEndian.PutUint16(data[33:35], 34)
	binary.LittleEndian.PutUint16(data[35:37], 9)
	binary.LittleEndian.PutUint16(data[37:39], 7)
	binary.LittleEndian.PutUint16(data[39:41], 3)
	binary.LittleEndian.PutUint32(data[41:45], 0x01020304)
	binary.LittleEndian.PutUint16(data[45:47], 8)
	binary.LittleEndian.PutUint16(data[63:65], 99)
	data[54] = 1
	data[55], data[56], data[57], data[58], data[59], data[60] = packMovePosition(10, 20, 30, 40)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x02EC, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !entry.Moving || !entry.Appearance {
		t.Fatalf("move entry not parsed: ok=%v entry=%+v", ok, entry)
	}
	if entry.ID != 2000006 || entry.HeadPal != 9 || entry.BodyPal != 7 || entry.HeadDir != 3 || entry.FromX != 10 || entry.FromY != 20 || entry.ToX != 30 || entry.ToY != 40 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.GuildID != 0x01020304 || entry.EmblemVersion != 8 {
		t.Fatalf("unexpected guild emblem fields: %+v", entry)
	}
	if !entry.HasLevel || entry.Level != 99 {
		t.Fatalf("level = %d has=%t, want 99 true", entry.Level, entry.HasLevel)
	}
	if !entry.HasMoveStartTick || entry.MoveStartTick != 234567 {
		t.Fatalf("move start tick = %d has=%t, want 234567 true", entry.MoveStartTick, entry.HasMoveStartTick)
	}
}

func TestParseActorMoveUpdate(t *testing.T) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint16(data[0:2], 0x0086)
	binary.LittleEndian.PutUint32(data[2:6], 2000004)
	data[6], data[7], data[8], data[9], data[10], data[11] = packMovePosition(11, 21, 31, 41)
	binary.LittleEndian.PutUint32(data[12:16], 123456)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x0086, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !entry.Moving {
		t.Fatalf("move update not parsed: ok=%v entry=%+v", ok, entry)
	}
	if entry.Appearance {
		t.Fatal("move-only update should not include appearance")
	}
	if entry.ID != 2000004 || entry.FromX != 11 || entry.FromY != 21 || entry.ToX != 31 || entry.ToY != 41 || entry.X != 31 || entry.Y != 41 {
		t.Fatalf("unexpected move update: %+v", entry)
	}
	if !entry.HasMoveStartTick || entry.MoveStartTick != 123456 {
		t.Fatalf("move start tick = %d has=%t, want 123456 true", entry.MoveStartTick, entry.HasMoveStartTick)
	}
}

func TestParseActorMoveEntryModern(t *testing.T) {
	data := make([]byte, 65)
	binary.LittleEndian.PutUint16(data[0:2], 0x022C)
	data[2] = 5
	binary.LittleEndian.PutUint32(data[3:7], 1100001)
	binary.LittleEndian.PutUint16(data[7:9], 360)
	binary.LittleEndian.PutUint16(data[17:19], 1002)
	binary.LittleEndian.PutUint16(data[19:21], 3)
	binary.LittleEndian.PutUint32(data[21:25], uint32(2101)<<16|1201)
	binary.LittleEndian.PutUint16(data[25:27], 11)
	binary.LittleEndian.PutUint32(data[27:31], 345678)
	binary.LittleEndian.PutUint16(data[31:33], 22)
	binary.LittleEndian.PutUint16(data[33:35], 33)
	binary.LittleEndian.PutUint16(data[63:65], 99)
	data[54] = 1
	data[55], data[56], data[57], data[58], data[59], data[60] = packMovePosition(10, 20, 11, 21)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x022C, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !entry.Moving || !entry.Appearance {
		t.Fatalf("modern move entry not parsed: ok=%v entry=%+v", ok, entry)
	}
	if entry.ID != 1100001 || entry.Speed != 360 || entry.ObjectType != 5 || entry.Job != 1002 || entry.Head != 3 || entry.Weapon != 1201 || entry.Shield != 2101 || entry.HeadLow != 11 || entry.HeadTop != 22 || entry.HeadMid != 33 || entry.Sex != 1 {
		t.Fatalf("unexpected appearance: %+v", entry)
	}
	if entry.FromX != 10 || entry.FromY != 20 || entry.ToX != 11 || entry.ToY != 21 || entry.X != 11 || entry.Y != 21 {
		t.Fatalf("unexpected movement: %+v", entry)
	}
	if !entry.HasMoveStartTick || entry.MoveStartTick != 345678 {
		t.Fatalf("move start tick = %d has=%t, want 345678 true", entry.MoveStartTick, entry.HasMoveStartTick)
	}
	if !entry.HasLevel || entry.Level != 99 {
		t.Fatalf("level = %d has=%t, want 99 true", entry.Level, entry.HasLevel)
	}
}

func TestParseActorMoveEntryVariableRobe(t *testing.T) {
	data := make([]byte, 71)
	binary.LittleEndian.PutUint16(data[0:2], 0x0856)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	data[4] = 5
	binary.LittleEndian.PutUint32(data[5:9], 1100002)
	binary.LittleEndian.PutUint16(data[9:11], 620)
	binary.LittleEndian.PutUint16(data[19:21], 1063)
	binary.LittleEndian.PutUint16(data[21:23], 4)
	binary.LittleEndian.PutUint32(data[23:27], uint32(2102)<<16|1202)
	binary.LittleEndian.PutUint16(data[27:29], 12)
	binary.LittleEndian.PutUint32(data[29:33], 456789)
	binary.LittleEndian.PutUint16(data[33:35], 23)
	binary.LittleEndian.PutUint16(data[35:37], 34)
	data[58] = 0
	data[59], data[60], data[61], data[62], data[63], data[64] = packMovePosition(30, 40, 31, 41)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x0856, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !entry.Moving || !entry.Appearance {
		t.Fatalf("robe move entry not parsed: ok=%v entry=%+v", ok, entry)
	}
	if entry.ID != 1100002 || entry.Speed != 620 || entry.ObjectType != 5 || entry.Job != 1063 || entry.Head != 4 || entry.Weapon != 1202 || entry.Shield != 2102 || entry.HeadLow != 12 || entry.HeadTop != 23 || entry.HeadMid != 34 || entry.Sex != 0 {
		t.Fatalf("unexpected robe appearance: %+v", entry)
	}
	if entry.FromX != 30 || entry.FromY != 40 || entry.ToX != 31 || entry.ToY != 41 || entry.X != 31 || entry.Y != 41 {
		t.Fatalf("unexpected robe movement: %+v", entry)
	}
	if !entry.HasMoveStartTick || entry.MoveStartTick != 456789 {
		t.Fatalf("move start tick = %d has=%t, want 456789 true", entry.MoveStartTick, entry.HasMoveStartTick)
	}
}

func TestParseActorMoveEntryVariableNoRobe(t *testing.T) {
	data := make([]byte, 69)
	binary.LittleEndian.PutUint16(data[0:2], 0x07F7)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	data[4] = 5
	binary.LittleEndian.PutUint32(data[5:9], 1100003)
	binary.LittleEndian.PutUint16(data[9:11], 390)
	binary.LittleEndian.PutUint16(data[19:21], 1002)
	binary.LittleEndian.PutUint16(data[21:23], 5)
	binary.LittleEndian.PutUint32(data[23:27], uint32(2103)<<16|1203)
	binary.LittleEndian.PutUint16(data[27:29], 13)
	binary.LittleEndian.PutUint32(data[29:33], 567890)
	binary.LittleEndian.PutUint16(data[33:35], 24)
	binary.LittleEndian.PutUint16(data[35:37], 35)
	data[56] = 1
	data[57], data[58], data[59], data[60], data[61], data[62] = packMovePosition(50, 60, 51, 61)

	entry, ok, err := ParseActorEntry(Packet{ID: 0x07F7, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !entry.Moving || !entry.Appearance {
		t.Fatalf("variable move entry not parsed: ok=%v entry=%+v", ok, entry)
	}
	if entry.ID != 1100003 || entry.Speed != 390 || entry.ObjectType != 5 || entry.Job != 1002 || entry.Head != 5 || entry.Weapon != 1203 || entry.Shield != 2103 || entry.HeadLow != 13 || entry.HeadTop != 24 || entry.HeadMid != 35 || entry.Sex != 1 {
		t.Fatalf("unexpected variable appearance: %+v", entry)
	}
	if entry.FromX != 50 || entry.FromY != 60 || entry.ToX != 51 || entry.ToY != 61 || entry.X != 51 || entry.Y != 61 {
		t.Fatalf("unexpected variable movement: %+v", entry)
	}
	if !entry.HasMoveStartTick || entry.MoveStartTick != 567890 {
		t.Fatalf("move start tick = %d has=%t, want 567890 true", entry.MoveStartTick, entry.HasMoveStartTick)
	}
}

func TestParseActorVanish(t *testing.T) {
	data := make([]byte, 7)
	binary.LittleEndian.PutUint16(data[0:2], 0x0080)
	binary.LittleEndian.PutUint32(data[2:6], 2000005)
	data[6] = 1

	vanish, ok, err := ParseActorVanish(Packet{ID: 0x0080, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("vanish not parsed")
	}
	if vanish.ID != 2000005 || vanish.Reason != 1 {
		t.Fatalf("unexpected vanish: %+v", vanish)
	}
}

func TestParseActorResurrection(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:2], 0x0148)
	binary.LittleEndian.PutUint32(data[2:6], 2000005)
	binary.LittleEndian.PutUint16(data[6:8], 1)

	resurrection, ok, err := ParseActorResurrection(Packet{ID: 0x0148, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("resurrection not parsed")
	}
	if resurrection.ID != 2000005 || resurrection.Type != 1 {
		t.Fatalf("unexpected resurrection: %+v", resurrection)
	}
}

func TestParseActorLookChangeModern(t *testing.T) {
	data := make([]byte, 11)
	binary.LittleEndian.PutUint16(data[0:2], 0x01D7)
	binary.LittleEndian.PutUint32(data[2:6], 2000006)
	data[6] = 2
	binary.LittleEndian.PutUint32(data[7:11], uint32(2101)<<16|1201)

	look, ok, err := ParseActorLookChange(Packet{ID: 0x01D7, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("look change not parsed")
	}
	if look.ID != 2000006 || look.Type != 2 || look.Value != uint32(2101)<<16|1201 {
		t.Fatalf("unexpected look change: %+v", look)
	}
}

func TestParseActorLookChangeIgnoresStateChange3(t *testing.T) {
	data := make([]byte, 15)
	binary.LittleEndian.PutUint16(data[0:2], 0x0229)
	binary.LittleEndian.PutUint32(data[2:6], 2000006)
	binary.LittleEndian.PutUint16(data[6:8], 0)
	binary.LittleEndian.PutUint16(data[8:10], 0)
	binary.LittleEndian.PutUint32(data[10:14], 0)

	look, ok, err := ParseActorLookChange(Packet{ID: 0x0229, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("state change parsed as look change: %+v", look)
	}
}

func TestParseActorStateChange3(t *testing.T) {
	data := make([]byte, 15)
	binary.LittleEndian.PutUint16(data[0:2], 0x0229)
	binary.LittleEndian.PutUint32(data[2:6], 2000006)
	binary.LittleEndian.PutUint16(data[6:8], 2)
	binary.LittleEndian.PutUint16(data[8:10], 0x0010)
	binary.LittleEndian.PutUint32(data[10:14], 0x00402000)

	state, ok, err := ParseActorStateChange(Packet{ID: 0x0229, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("state change not parsed")
	}
	if state.ID != 2000006 || state.BodyState != 2 || state.HealthState != 0x0010 || state.EffectState != 0x00402000 {
		t.Fatalf("unexpected state change: %+v", state)
	}
}

func TestParseActorBladeStop(t *testing.T) {
	data := make([]byte, 14)
	binary.LittleEndian.PutUint16(data[0:2], 0x01D1)
	binary.LittleEndian.PutUint32(data[2:6], 10)
	binary.LittleEndian.PutUint32(data[6:10], 20)
	binary.LittleEndian.PutUint32(data[10:14], 1)

	blade, ok, err := ParseActorBladeStop(Packet{ID: 0x01D1, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || blade.SourceID != 10 || blade.TargetID != 20 || !blade.Active {
		t.Fatalf("unexpected blade stop: ok=%v blade=%+v", ok, blade)
	}
}

func TestParseActorLookChangeLegacy(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:2], 0x00C3)
	binary.LittleEndian.PutUint32(data[2:6], 2000006)
	data[6] = 4
	data[7] = 7

	look, ok, err := ParseActorLookChange(Packet{ID: 0x00C3, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || look.ID != 2000006 || look.Type != 4 || look.Value != 7 {
		t.Fatalf("unexpected legacy look change: ok=%v look=%+v", ok, look)
	}
}

func TestParseSelfMoveAck(t *testing.T) {
	data := make([]byte, 12)
	binary.LittleEndian.PutUint16(data[0:2], 0x0087)
	binary.LittleEndian.PutUint32(data[2:6], 123456)
	data[6], data[7], data[8], data[9], data[10], data[11] = packMovePosition(120, 121, 122, 123)

	ack, ok, err := ParseSelfMoveAck(Packet{ID: 0x0087, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("self move ack not parsed")
	}
	if ack.ServerTick != 123456 || ack.FromX != 120 || ack.FromY != 121 || ack.ToX != 122 || ack.ToY != 123 {
		t.Fatalf("unexpected self move ack: %+v", ack)
	}
}

func TestParseActorSetPosition(t *testing.T) {
	data := make([]byte, 10)
	binary.LittleEndian.PutUint16(data[0:2], 0x0088)
	binary.LittleEndian.PutUint32(data[2:6], 2000006)
	x := int16(-12)
	binary.LittleEndian.PutUint16(data[8:10], uint16(int16(34)))
	binary.LittleEndian.PutUint16(data[6:8], uint16(x))

	position, ok, err := ParseActorSetPosition(Packet{ID: 0x0088, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("actor set position not parsed")
	}
	if position.ID != 2000006 || position.X != -12 || position.Y != 34 {
		t.Fatalf("unexpected actor position: %+v", position)
	}
}

func TestParseActorJumpPosition(t *testing.T) {
	data := make([]byte, 10)
	binary.LittleEndian.PutUint16(data[0:2], 0x01FF)
	binary.LittleEndian.PutUint32(data[2:6], 2000006)
	x := int16(-12)
	binary.LittleEndian.PutUint16(data[8:10], uint16(int16(34)))
	binary.LittleEndian.PutUint16(data[6:8], uint16(x))

	position, ok, err := ParseActorJumpPosition(Packet{ID: 0x01FF, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("actor jump position not parsed")
	}
	if position.ID != 2000006 || position.X != -12 || position.Y != 34 {
		t.Fatalf("unexpected actor jump position: %+v", position)
	}
}

func TestParseAttackFailureForDistance(t *testing.T) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint16(data[0:2], 0x0139)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)
	binary.LittleEndian.PutUint16(data[6:8], uint16(int16(164)))
	binary.LittleEndian.PutUint16(data[8:10], uint16(int16(281)))
	binary.LittleEndian.PutUint16(data[10:12], uint16(int16(165)))
	binary.LittleEndian.PutUint16(data[12:14], uint16(int16(282)))
	binary.LittleEndian.PutUint16(data[14:16], 1)

	failure, ok, err := ParseAttackFailureForDistance(Packet{ID: 0x0139, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("attack failure not parsed")
	}
	if failure.TargetID != 0x11223344 || failure.TargetX != 164 || failure.TargetY != 281 || failure.SourceX != 165 || failure.SourceY != 282 || failure.AttackRange != 1 {
		t.Fatalf("unexpected attack failure: %+v", failure)
	}
}

func TestParseActorActionNotifyLegacy(t *testing.T) {
	data := make([]byte, 29)
	binary.LittleEndian.PutUint16(data[0:2], 0x008A)
	binary.LittleEndian.PutUint32(data[2:6], 2000000)
	binary.LittleEndian.PutUint32(data[6:10], 110014894)
	binary.LittleEndian.PutUint32(data[10:14], 123456)
	binary.LittleEndian.PutUint32(data[14:18], 432)
	binary.LittleEndian.PutUint32(data[18:22], 288)
	binary.LittleEndian.PutUint16(data[22:24], 42)
	binary.LittleEndian.PutUint16(data[24:26], 1)
	data[26] = 0
	binary.LittleEndian.PutUint16(data[27:29], 0)

	action, ok, err := ParseActorActionNotify(Packet{ID: 0x008A, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("action notify not parsed")
	}
	if action.SourceID != 2000000 || action.TargetID != 110014894 || action.Damage != 42 || action.HitCount != 1 || action.Action != 0 {
		t.Fatalf("unexpected action: %+v", action)
	}
}

func TestParseActorActionNotify2(t *testing.T) {
	data := make([]byte, 33)
	binary.LittleEndian.PutUint16(data[0:2], 0x02E1)
	binary.LittleEndian.PutUint32(data[2:6], 2000000)
	binary.LittleEndian.PutUint32(data[6:10], 110014894)
	binary.LittleEndian.PutUint32(data[10:14], 123456)
	binary.LittleEndian.PutUint32(data[14:18], 432)
	binary.LittleEndian.PutUint32(data[18:22], 288)
	binary.LittleEndian.PutUint32(data[22:26], 1234)
	binary.LittleEndian.PutUint16(data[26:28], 2)
	data[28] = 8
	binary.LittleEndian.PutUint32(data[29:33], 7)

	action, ok, err := ParseActorActionNotify(Packet{ID: 0x02E1, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("action notify2 not parsed")
	}
	if action.SourceID != 2000000 || action.TargetID != 110014894 || action.Damage != 1234 || action.HitCount != 2 || action.Action != 8 || action.LeftDamage != 7 {
		t.Fatalf("unexpected action: %+v", action)
	}
}

func TestParseActorSkillNotifyLegacy(t *testing.T) {
	data := make([]byte, 31)
	binary.LittleEndian.PutUint16(data[0:2], 0x0114)
	binary.LittleEndian.PutUint16(data[2:4], 5)
	binary.LittleEndian.PutUint32(data[4:8], 2000000)
	binary.LittleEndian.PutUint32(data[8:12], 110014894)
	binary.LittleEndian.PutUint32(data[12:16], 123456)
	binary.LittleEndian.PutUint32(data[16:20], 580)
	binary.LittleEndian.PutUint32(data[20:24], 480)
	binary.LittleEndian.PutUint16(data[24:26], 84)
	binary.LittleEndian.PutUint16(data[26:28], 3)
	binary.LittleEndian.PutUint16(data[28:30], 1)
	data[30] = 6

	action, ok, err := ParseActorActionNotify(Packet{ID: 0x0114, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill notify not parsed")
	}
	if action.SkillID != 5 || action.SkillLevel != 3 || action.SourceID != 2000000 || action.TargetID != 110014894 || action.Damage != 84 || action.HitCount != 1 || action.Action != 6 {
		t.Fatalf("unexpected skill action: %+v", action)
	}
}

func TestParseActorSkillNotify2(t *testing.T) {
	data := make([]byte, 33)
	binary.LittleEndian.PutUint16(data[0:2], 0x01DE)
	binary.LittleEndian.PutUint16(data[2:4], 5)
	binary.LittleEndian.PutUint32(data[4:8], 2000000)
	binary.LittleEndian.PutUint32(data[8:12], 110014894)
	binary.LittleEndian.PutUint32(data[12:16], 123456)
	binary.LittleEndian.PutUint32(data[16:20], 580)
	binary.LittleEndian.PutUint32(data[20:24], 480)
	binary.LittleEndian.PutUint32(data[24:28], 84000)
	binary.LittleEndian.PutUint16(data[28:30], 3)
	binary.LittleEndian.PutUint16(data[30:32], 2)
	data[32] = 8

	action, ok, err := ParseActorActionNotify(Packet{ID: 0x01DE, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("skill notify2 not parsed")
	}
	if action.SkillID != 5 || action.SkillLevel != 3 || action.SourceID != 2000000 || action.TargetID != 110014894 || action.Damage != 84000 || action.HitCount != 2 || action.Action != 8 {
		t.Fatalf("unexpected skill action2: %+v", action)
	}
}

func TestParseActorDirectionChange(t *testing.T) {
	data := make([]byte, 9)
	binary.LittleEndian.PutUint16(data[0:2], 0x009C)
	binary.LittleEndian.PutUint32(data[2:6], 2000000)
	binary.LittleEndian.PutUint16(data[6:8], 2)
	data[8] = 6

	direction, ok, err := ParseActorDirectionChange(Packet{ID: 0x009C, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("direction change not parsed")
	}
	if direction.ID != 2000000 || direction.HeadDir != 2 || direction.Dir != 6 {
		t.Fatalf("unexpected direction change: %+v", direction)
	}
}

func TestParseActorHPUpdate(t *testing.T) {
	data := make([]byte, 14)
	binary.LittleEndian.PutUint16(data[0:2], 0x0977)
	binary.LittleEndian.PutUint32(data[2:6], 110014894)
	binary.LittleEndian.PutUint32(data[6:10], 42)
	binary.LittleEndian.PutUint32(data[10:14], 100)

	update, ok, err := ParseActorHPUpdate(Packet{ID: 0x0977, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("monster hp not parsed")
	}
	if update.ID != 110014894 || update.HP != 42 || update.MaxHP != 100 || update.Tiny {
		t.Fatalf("unexpected hp update: %+v", update)
	}
}

func TestParseActorHPTinyUpdate(t *testing.T) {
	data := make([]byte, 7)
	binary.LittleEndian.PutUint16(data[0:2], 0x0A36)
	binary.LittleEndian.PutUint32(data[2:6], 110014894)
	data[6] = 13

	update, ok, err := ParseActorHPUpdate(Packet{ID: 0x0A36, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("tiny hp not parsed")
	}
	if update.ID != 110014894 || update.HP != 65 || update.MaxHP != 100 || !update.Tiny {
		t.Fatalf("unexpected tiny hp update: %+v", update)
	}
}

func packMovePosition(fromX, fromY, toX, toY int) (byte, byte, byte, byte, byte, byte) {
	return byte(fromX >> 2),
		byte(((fromX & 0x03) << 6) | ((fromY >> 4) & 0x3f)),
		byte(((fromY & 0x0f) << 4) | ((toX >> 6) & 0x0f)),
		byte(((toX & 0x3f) << 2) | ((toY >> 8) & 0x03)),
		byte(toY),
		0
}
