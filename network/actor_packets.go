package network

import (
	"encoding/binary"
	"fmt"
)

const legacyGuildFlagJob int16 = 722

type ActorEntry struct {
	ID               uint32
	GuildID          uint32
	EmblemVersion    uint32
	Job              int16
	Head             int16
	Weapon           int16
	Shield           int16
	HeadTop          int16
	HeadMid          int16
	HeadLow          int16
	HeadPal          int16
	BodyPal          int16
	Sex              uint8
	HeadDir          uint8
	Appearance       bool
	X                int
	Y                int
	Dir              int
	Moving           bool
	FromX            int
	FromY            int
	ToX              int
	ToY              int
	MoveStartTick    uint32
	HasMoveStartTick bool
	ObjectType       uint8
	HasObjectType    bool
	Speed            int
	BodyState        uint16
	HealthState      uint16
	EffectState      uint32
	HasState         bool
	Level            int
	HasLevel         bool
}

type ActorVanish struct {
	ID     uint32
	Reason uint8
}

type ActorResurrection struct {
	ID   uint32
	Type uint16
}

type SelfMoveAck struct {
	ServerTick uint32
	FromX      int
	FromY      int
	ToX        int
	ToY        int
}

type ActorSetPosition struct {
	ID uint32
	X  int
	Y  int
}

type ActorJumpPosition struct {
	ID uint32
	X  int
	Y  int
}

type ActorLookChange struct {
	ID    uint32
	Type  uint8
	Value uint32
}

type ActorDirectionChange struct {
	ID      uint32
	HeadDir uint8
	Dir     uint8
}

type ActorStateChange struct {
	ID          uint32
	BodyState   uint16
	HealthState uint16
	EffectState uint32
}

type ActorBladeStop struct {
	SourceID uint32
	TargetID uint32
	Active   bool
}

type ActorActionNotify struct {
	SourceID    uint32
	TargetID    uint32
	ServerTick  uint32
	SourceSpeed int32
	TargetSpeed int32
	Damage      int32
	HitCount    uint16
	Action      uint8
	LeftDamage  int32
	SkillID     uint16
	SkillLevel  uint16
}

const (
	ActorActionPickupItem uint8 = 1
	ActorActionSkill      uint8 = 6
)

type ActorHPUpdate struct {
	ID    uint32
	HP    int
	MaxHP int
	Tiny  bool
}

type AttackFailureForDistance struct {
	TargetID    uint32
	TargetX     int
	TargetY     int
	SourceX     int
	SourceY     int
	AttackRange int
}

func ParseActorHPUpdate(packet Packet) (ActorHPUpdate, bool, error) {
	switch packet.ID {
	case 0x0977:
		if len(packet.Data) < 14 {
			return ActorHPUpdate{}, false, fmt.Errorf("ZC_NOTIFY_MONSTER_HP too short: %d", len(packet.Data))
		}
		return ActorHPUpdate{
			ID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
			HP:    int(binary.LittleEndian.Uint32(packet.Data[6:10])),
			MaxHP: int(binary.LittleEndian.Uint32(packet.Data[10:14])),
		}, true, nil
	case 0x0A36:
		if len(packet.Data) < 7 {
			return ActorHPUpdate{}, false, fmt.Errorf("ZC_HP_INFO_TINY too short: %d", len(packet.Data))
		}
		return ActorHPUpdate{
			ID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
			HP:    int(packet.Data[6]) * 5,
			MaxHP: 100,
			Tiny:  true,
		}, true, nil
	default:
		return ActorHPUpdate{}, false, nil
	}
}

func ParseActorEntry(packet Packet) (ActorEntry, bool, error) {
	switch packet.ID {
	case 0x0078:
		if len(packet.Data) < 55 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_STANDENTRY too short: %d", len(packet.Data))
		}
		x, y, dir := unpackPos(packet.Data[47:50])
		job := int16(binary.LittleEndian.Uint16(packet.Data[15:17]))
		shield := int16(binary.LittleEndian.Uint16(packet.Data[23:25]))
		headTop := int16(binary.LittleEndian.Uint16(packet.Data[25:27]))
		headMid := int16(binary.LittleEndian.Uint16(packet.Data[27:29]))
		guildID := binary.LittleEndian.Uint32(packet.Data[35:39])
		emblemVersion := uint32(binary.LittleEndian.Uint16(packet.Data[39:41]))
		guildID, emblemVersion = legacyGuildFlagIdentity(job, shield, headTop, headMid, guildID, emblemVersion)
		return ActorEntry{
			ObjectType:    packet.Data[2],
			HasObjectType: true,
			ID:            binary.LittleEndian.Uint32(packet.Data[3:7]),
			Speed:         int(binary.LittleEndian.Uint16(packet.Data[7:9])),
			BodyState:     binary.LittleEndian.Uint16(packet.Data[9:11]),
			HealthState:   binary.LittleEndian.Uint16(packet.Data[11:13]),
			EffectState:   uint32(binary.LittleEndian.Uint16(packet.Data[13:15])),
			HasState:      true,
			Job:           job,
			Head:          int16(binary.LittleEndian.Uint16(packet.Data[17:19])),
			Weapon:        int16(binary.LittleEndian.Uint16(packet.Data[19:21])),
			HeadLow:       int16(binary.LittleEndian.Uint16(packet.Data[21:23])),
			Shield:        shield,
			HeadTop:       headTop,
			HeadMid:       headMid,
			HeadPal:       int16(binary.LittleEndian.Uint16(packet.Data[29:31])),
			BodyPal:       int16(binary.LittleEndian.Uint16(packet.Data[31:33])),
			HeadDir:       uint8(binary.LittleEndian.Uint16(packet.Data[33:35])),
			GuildID:       guildID,
			EmblemVersion: emblemVersion,
			Sex:           packet.Data[46],
			Appearance:    true,
			X:             x,
			Y:             y,
			Dir:           dir,
		}, true, nil
	case 0x0079, 0x007A:
		if len(packet.Data) < 53 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_ENTRY too short: %d", len(packet.Data))
		}
		x, y, dir := unpackPos(packet.Data[46:49])
		return ActorEntry{
			ID:          binary.LittleEndian.Uint32(packet.Data[2:6]),
			Speed:       int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			BodyState:   binary.LittleEndian.Uint16(packet.Data[8:10]),
			HealthState: binary.LittleEndian.Uint16(packet.Data[10:12]),
			EffectState: uint32(binary.LittleEndian.Uint16(packet.Data[12:14])),
			HasState:    true,
			Job:         int16(binary.LittleEndian.Uint16(packet.Data[14:16])),
			Head:        int16(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Weapon:      int16(binary.LittleEndian.Uint16(packet.Data[18:20])),
			HeadLow:     int16(binary.LittleEndian.Uint16(packet.Data[26:28])),
			Shield:      int16(binary.LittleEndian.Uint16(packet.Data[28:30])),
			HeadTop:     int16(binary.LittleEndian.Uint16(packet.Data[30:32])),
			HeadMid:     int16(binary.LittleEndian.Uint16(packet.Data[32:34])),
			Sex:         packet.Data[45],
			Appearance:  true,
			X:           x,
			Y:           y,
			Dir:         dir,
		}, true, nil
	case 0x007B:
		if len(packet.Data) < 60 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_MOVEENTRY too short: %d", len(packet.Data))
		}
		fromX, fromY, toX, toY := unpackMovePos(packet.Data[50:56])
		return ActorEntry{
			ID:               binary.LittleEndian.Uint32(packet.Data[2:6]),
			Speed:            int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			BodyState:        binary.LittleEndian.Uint16(packet.Data[8:10]),
			HealthState:      binary.LittleEndian.Uint16(packet.Data[10:12]),
			EffectState:      uint32(binary.LittleEndian.Uint16(packet.Data[12:14])),
			HasState:         true,
			Job:              int16(binary.LittleEndian.Uint16(packet.Data[14:16])),
			Head:             int16(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Weapon:           int16(binary.LittleEndian.Uint16(packet.Data[18:20])),
			HeadLow:          int16(binary.LittleEndian.Uint16(packet.Data[26:28])),
			Shield:           int16(binary.LittleEndian.Uint16(packet.Data[28:30])),
			HeadTop:          int16(binary.LittleEndian.Uint16(packet.Data[30:32])),
			HeadMid:          int16(binary.LittleEndian.Uint16(packet.Data[32:34])),
			Sex:              packet.Data[49],
			Appearance:       true,
			X:                toX,
			Y:                toY,
			FromX:            fromX,
			FromY:            fromY,
			ToX:              toX,
			ToY:              toY,
			MoveStartTick:    binary.LittleEndian.Uint32(packet.Data[22:26]),
			HasMoveStartTick: true,
			Moving:           true,
		}, true, nil
	case 0x007C:
		if len(packet.Data) < 42 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_STANDENTRY_NPC too short: %d", len(packet.Data))
		}
		x, y, dir := unpackPos(packet.Data[37:40])
		job := int16(binary.LittleEndian.Uint16(packet.Data[21:23]))
		shield := int16(binary.LittleEndian.Uint16(packet.Data[23:25]))
		headTop := int16(binary.LittleEndian.Uint16(packet.Data[25:27]))
		headMid := int16(binary.LittleEndian.Uint16(packet.Data[27:29]))
		guildID, emblemVersion := legacyGuildFlagIdentity(job, shield, headTop, headMid, 0, 0)
		return ActorEntry{
			ObjectType:    packet.Data[2],
			HasObjectType: true,
			ID:            binary.LittleEndian.Uint32(packet.Data[3:7]),
			Speed:         int(binary.LittleEndian.Uint16(packet.Data[7:9])),
			BodyState:     binary.LittleEndian.Uint16(packet.Data[9:11]),
			HealthState:   binary.LittleEndian.Uint16(packet.Data[11:13]),
			EffectState:   uint32(binary.LittleEndian.Uint16(packet.Data[13:15])),
			HasState:      true,
			Job:           job,
			Head:          int16(binary.LittleEndian.Uint16(packet.Data[15:17])),
			Weapon:        int16(binary.LittleEndian.Uint16(packet.Data[17:19])),
			HeadLow:       int16(binary.LittleEndian.Uint16(packet.Data[19:21])),
			Shield:        shield,
			HeadTop:       headTop,
			HeadMid:       headMid,
			HeadPal:       int16(binary.LittleEndian.Uint16(packet.Data[29:31])),
			BodyPal:       int16(binary.LittleEndian.Uint16(packet.Data[31:33])),
			HeadDir:       uint8(binary.LittleEndian.Uint16(packet.Data[33:35])),
			GuildID:       guildID,
			EmblemVersion: emblemVersion,
			Sex:           packet.Data[36],
			Appearance:    true,
			X:             x,
			Y:             y,
			Dir:           dir,
		}, true, nil
	case 0x0086:
		if len(packet.Data) < 16 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_MOVE too short: %d", len(packet.Data))
		}
		fromX, fromY, toX, toY := unpackMovePos(packet.Data[6:12])
		return ActorEntry{
			ID:               binary.LittleEndian.Uint32(packet.Data[2:6]),
			X:                toX,
			Y:                toY,
			FromX:            fromX,
			FromY:            fromY,
			ToX:              toX,
			ToY:              toY,
			MoveStartTick:    binary.LittleEndian.Uint32(packet.Data[12:16]),
			HasMoveStartTick: true,
			Moving:           true,
		}, true, nil
	case 0x01D8:
		if len(packet.Data) < 54 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_STANDENTRY2 too short: %d", len(packet.Data))
		}
		x, y, dir := unpackPos(packet.Data[46:49])
		return ActorEntry{
			ID:          binary.LittleEndian.Uint32(packet.Data[2:6]),
			Speed:       int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			BodyState:   binary.LittleEndian.Uint16(packet.Data[8:10]),
			HealthState: binary.LittleEndian.Uint16(packet.Data[10:12]),
			EffectState: uint32(binary.LittleEndian.Uint16(packet.Data[12:14])),
			HasState:    true,
			Job:         int16(binary.LittleEndian.Uint16(packet.Data[14:16])),
			Head:        int16(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Weapon:      int16(binary.LittleEndian.Uint16(packet.Data[18:20])),
			HeadLow:     int16(binary.LittleEndian.Uint16(packet.Data[26:28])),
			Shield:      int16(binary.LittleEndian.Uint16(packet.Data[28:30])),
			HeadTop:     int16(binary.LittleEndian.Uint16(packet.Data[30:32])),
			HeadMid:     int16(binary.LittleEndian.Uint16(packet.Data[32:34])),
			Sex:         packet.Data[45],
			Appearance:  true,
			X:           x,
			Y:           y,
			Dir:         dir,
		}, true, nil
	case 0x01D9:
		if len(packet.Data) < 53 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_NEWENTRY2 too short: %d", len(packet.Data))
		}
		x, y, dir := unpackPos(packet.Data[46:49])
		return ActorEntry{
			ID:          binary.LittleEndian.Uint32(packet.Data[2:6]),
			Speed:       int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			BodyState:   binary.LittleEndian.Uint16(packet.Data[8:10]),
			HealthState: binary.LittleEndian.Uint16(packet.Data[10:12]),
			EffectState: uint32(binary.LittleEndian.Uint16(packet.Data[12:14])),
			HasState:    true,
			Job:         int16(binary.LittleEndian.Uint16(packet.Data[14:16])),
			Head:        int16(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Weapon:      int16(binary.LittleEndian.Uint16(packet.Data[18:20])),
			HeadLow:     int16(binary.LittleEndian.Uint16(packet.Data[26:28])),
			Shield:      int16(binary.LittleEndian.Uint16(packet.Data[28:30])),
			HeadTop:     int16(binary.LittleEndian.Uint16(packet.Data[30:32])),
			HeadMid:     int16(binary.LittleEndian.Uint16(packet.Data[32:34])),
			Sex:         packet.Data[45],
			Appearance:  true,
			X:           x,
			Y:           y,
			Dir:         dir,
		}, true, nil
	case 0x01DA:
		if len(packet.Data) < 60 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_MOVEENTRY2 too short: %d", len(packet.Data))
		}
		fromX, fromY, toX, toY := unpackMovePos(packet.Data[50:56])
		return ActorEntry{
			ID:               binary.LittleEndian.Uint32(packet.Data[2:6]),
			Speed:            int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			Job:              int16(binary.LittleEndian.Uint16(packet.Data[14:16])),
			Head:             int16(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Weapon:           int16(binary.LittleEndian.Uint16(packet.Data[18:20])),
			HeadLow:          int16(binary.LittleEndian.Uint16(packet.Data[26:28])),
			Shield:           int16(binary.LittleEndian.Uint16(packet.Data[28:30])),
			HeadTop:          int16(binary.LittleEndian.Uint16(packet.Data[30:32])),
			HeadMid:          int16(binary.LittleEndian.Uint16(packet.Data[32:34])),
			Sex:              packet.Data[49],
			Appearance:       true,
			X:                toX,
			Y:                toY,
			FromX:            fromX,
			FromY:            fromY,
			ToX:              toX,
			ToY:              toY,
			MoveStartTick:    binary.LittleEndian.Uint32(packet.Data[24:28]),
			HasMoveStartTick: true,
			Moving:           true,
		}, true, nil
	case 0x02ED:
		if len(packet.Data) < 59 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_NEWENTRY3 too short: %d", len(packet.Data))
		}
		x, y, dir := unpackPos(packet.Data[50:53])
		weaponValue := binary.LittleEndian.Uint32(packet.Data[20:24])
		return ActorEntry{
			ID:            binary.LittleEndian.Uint32(packet.Data[2:6]),
			Speed:         int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			BodyState:     binary.LittleEndian.Uint16(packet.Data[8:10]),
			HealthState:   binary.LittleEndian.Uint16(packet.Data[10:12]),
			EffectState:   binary.LittleEndian.Uint32(packet.Data[12:16]),
			HasState:      true,
			Job:           int16(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Head:          int16(binary.LittleEndian.Uint16(packet.Data[18:20])),
			Weapon:        int16(weaponValue & 0xFFFF),
			Shield:        int16((weaponValue >> 16) & 0xFFFF),
			HeadLow:       int16(binary.LittleEndian.Uint16(packet.Data[24:26])),
			HeadTop:       int16(binary.LittleEndian.Uint16(packet.Data[26:28])),
			HeadMid:       int16(binary.LittleEndian.Uint16(packet.Data[28:30])),
			HeadPal:       int16(binary.LittleEndian.Uint16(packet.Data[30:32])),
			BodyPal:       int16(binary.LittleEndian.Uint16(packet.Data[32:34])),
			HeadDir:       uint8(binary.LittleEndian.Uint16(packet.Data[34:36])),
			GuildID:       binary.LittleEndian.Uint32(packet.Data[36:40]),
			EmblemVersion: uint32(binary.LittleEndian.Uint16(packet.Data[40:42])),
			Level:         int(binary.LittleEndian.Uint16(packet.Data[55:57])),
			HasLevel:      true,
			Sex:           packet.Data[49],
			Appearance:    true,
			X:             x,
			Y:             y,
			Dir:           dir,
		}, true, nil
	case 0x02EE:
		if len(packet.Data) < 60 {
			return ActorEntry{}, false, fmt.Errorf("ZC_NOTIFY_STANDENTRY3 too short: %d", len(packet.Data))
		}
		x, y, dir := unpackPos(packet.Data[50:53])
		weaponValue := binary.LittleEndian.Uint32(packet.Data[20:24])
		return ActorEntry{
			ID:            binary.LittleEndian.Uint32(packet.Data[2:6]),
			Speed:         int(binary.LittleEndian.Uint16(packet.Data[6:8])),
			BodyState:     binary.LittleEndian.Uint16(packet.Data[8:10]),
			HealthState:   binary.LittleEndian.Uint16(packet.Data[10:12]),
			EffectState:   binary.LittleEndian.Uint32(packet.Data[12:16]),
			HasState:      true,
			Job:           int16(binary.LittleEndian.Uint16(packet.Data[16:18])),
			Head:          int16(binary.LittleEndian.Uint16(packet.Data[18:20])),
			Weapon:        int16(weaponValue & 0xFFFF),
			Shield:        int16((weaponValue >> 16) & 0xFFFF),
			HeadLow:       int16(binary.LittleEndian.Uint16(packet.Data[24:26])),
			HeadTop:       int16(binary.LittleEndian.Uint16(packet.Data[26:28])),
			HeadMid:       int16(binary.LittleEndian.Uint16(packet.Data[28:30])),
			HeadPal:       int16(binary.LittleEndian.Uint16(packet.Data[30:32])),
			BodyPal:       int16(binary.LittleEndian.Uint16(packet.Data[32:34])),
			HeadDir:       uint8(binary.LittleEndian.Uint16(packet.Data[34:36])),
			GuildID:       binary.LittleEndian.Uint32(packet.Data[36:40]),
			EmblemVersion: uint32(binary.LittleEndian.Uint16(packet.Data[40:42])),
			Level:         int(binary.LittleEndian.Uint16(packet.Data[56:58])),
			HasLevel:      true,
			Sex:           packet.Data[49],
			Appearance:    true,
			X:             x,
			Y:             y,
			Dir:           dir,
		}, true, nil
	case 0x022C, 0x02EC:
		entry, err := parseActorMoveEntryModern(packet)
		if err != nil {
			return ActorEntry{}, false, err
		}
		return entry, true, nil
	case 0x07F7:
		entry, err := parseActorMoveEntryVariable(packet, false)
		if err != nil {
			return ActorEntry{}, false, err
		}
		return entry, true, nil
	case 0x0856:
		entry, err := parseActorMoveEntryVariable(packet, true)
		if err != nil {
			return ActorEntry{}, false, err
		}
		return entry, true, nil
	default:
		return ActorEntry{}, false, nil
	}
}

func legacyGuildFlagIdentity(job, shield, headTop, headMid int16, guildID, emblemVersion uint32) (uint32, uint32) {
	if job != legacyGuildFlagJob {
		return guildID, emblemVersion
	}
	if guildID == 0 {
		guildID = uint32(uint16(headTop))<<16 | uint32(uint16(headMid))
	}
	if emblemVersion == 0 {
		emblemVersion = uint32(uint16(shield))
	}
	return guildID, emblemVersion
}

func parseActorMoveEntryModern(packet Packet) (ActorEntry, error) {
	minLength := 65
	if packet.ID == 0x02EC {
		minLength = 67
	}
	if len(packet.Data) < minLength {
		return ActorEntry{}, fmt.Errorf("ZC_NOTIFY_MOVEENTRY modern 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	fromX, fromY, toX, toY := unpackMovePos(packet.Data[55:61])
	weaponValue := binary.LittleEndian.Uint32(packet.Data[21:25])
	return ActorEntry{
		ObjectType:       packet.Data[2],
		HasObjectType:    true,
		ID:               binary.LittleEndian.Uint32(packet.Data[3:7]),
		Speed:            int(binary.LittleEndian.Uint16(packet.Data[7:9])),
		BodyState:        binary.LittleEndian.Uint16(packet.Data[9:11]),
		HealthState:      binary.LittleEndian.Uint16(packet.Data[11:13]),
		EffectState:      binary.LittleEndian.Uint32(packet.Data[13:17]),
		HasState:         true,
		Job:              int16(binary.LittleEndian.Uint16(packet.Data[17:19])),
		Head:             int16(binary.LittleEndian.Uint16(packet.Data[19:21])),
		Weapon:           int16(weaponValue & 0xFFFF),
		Shield:           int16((weaponValue >> 16) & 0xFFFF),
		HeadLow:          int16(binary.LittleEndian.Uint16(packet.Data[25:27])),
		HeadTop:          int16(binary.LittleEndian.Uint16(packet.Data[31:33])),
		HeadMid:          int16(binary.LittleEndian.Uint16(packet.Data[33:35])),
		HeadPal:          int16(binary.LittleEndian.Uint16(packet.Data[35:37])),
		BodyPal:          int16(binary.LittleEndian.Uint16(packet.Data[37:39])),
		HeadDir:          uint8(binary.LittleEndian.Uint16(packet.Data[39:41])),
		GuildID:          binary.LittleEndian.Uint32(packet.Data[41:45]),
		EmblemVersion:    uint32(binary.LittleEndian.Uint16(packet.Data[45:47])),
		Level:            int(binary.LittleEndian.Uint16(packet.Data[63:65])),
		HasLevel:         true,
		Sex:              packet.Data[54],
		Appearance:       true,
		X:                toX,
		Y:                toY,
		FromX:            fromX,
		FromY:            fromY,
		ToX:              toX,
		ToY:              toY,
		MoveStartTick:    binary.LittleEndian.Uint32(packet.Data[27:31]),
		HasMoveStartTick: true,
		Moving:           true,
	}, nil
}

func parseActorMoveEntryVariable(packet Packet, hasRobe bool) (ActorEntry, error) {
	minLength := 69
	if hasRobe {
		minLength = 71
	}
	if len(packet.Data) < minLength {
		return ActorEntry{}, fmt.Errorf("ZC_NOTIFY_MOVEENTRY variable 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	moveOffset := 57
	if hasRobe {
		moveOffset = 59
	}
	sexOffset := moveOffset - 1
	guildOffset := 43
	if hasRobe {
		guildOffset = 45
	}
	fromX, fromY, toX, toY := unpackMovePos(packet.Data[moveOffset : moveOffset+6])
	weaponValue := binary.LittleEndian.Uint32(packet.Data[23:27])
	return ActorEntry{
		ObjectType:       packet.Data[4],
		HasObjectType:    true,
		ID:               binary.LittleEndian.Uint32(packet.Data[5:9]),
		Speed:            int(binary.LittleEndian.Uint16(packet.Data[9:11])),
		BodyState:        binary.LittleEndian.Uint16(packet.Data[11:13]),
		HealthState:      binary.LittleEndian.Uint16(packet.Data[13:15]),
		EffectState:      uint32(binary.LittleEndian.Uint32(packet.Data[15:19])),
		HasState:         true,
		Job:              int16(binary.LittleEndian.Uint16(packet.Data[19:21])),
		Head:             int16(binary.LittleEndian.Uint16(packet.Data[21:23])),
		Weapon:           int16(weaponValue & 0xFFFF),
		Shield:           int16((weaponValue >> 16) & 0xFFFF),
		HeadLow:          int16(binary.LittleEndian.Uint16(packet.Data[27:29])),
		HeadTop:          int16(binary.LittleEndian.Uint16(packet.Data[33:35])),
		HeadMid:          int16(binary.LittleEndian.Uint16(packet.Data[35:37])),
		HeadPal:          int16(binary.LittleEndian.Uint16(packet.Data[37:39])),
		BodyPal:          int16(binary.LittleEndian.Uint16(packet.Data[39:41])),
		HeadDir:          uint8(binary.LittleEndian.Uint16(packet.Data[41:43])),
		GuildID:          binary.LittleEndian.Uint32(packet.Data[guildOffset : guildOffset+4]),
		EmblemVersion:    uint32(binary.LittleEndian.Uint16(packet.Data[guildOffset+4 : guildOffset+6])),
		Sex:              packet.Data[sexOffset],
		Appearance:       true,
		X:                toX,
		Y:                toY,
		FromX:            fromX,
		FromY:            fromY,
		ToX:              toX,
		ToY:              toY,
		MoveStartTick:    binary.LittleEndian.Uint32(packet.Data[29:33]),
		HasMoveStartTick: true,
		Moving:           true,
	}, nil
}

func ParseActorActionNotify(packet Packet) (ActorActionNotify, bool, error) {
	switch packet.ID {
	case 0x008A:
		if len(packet.Data) < 29 {
			return ActorActionNotify{}, false, fmt.Errorf("ZC_NOTIFY_ACT too short: %d", len(packet.Data))
		}
		return ActorActionNotify{
			SourceID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
			TargetID:    binary.LittleEndian.Uint32(packet.Data[6:10]),
			ServerTick:  binary.LittleEndian.Uint32(packet.Data[10:14]),
			SourceSpeed: int32(binary.LittleEndian.Uint32(packet.Data[14:18])),
			TargetSpeed: int32(binary.LittleEndian.Uint32(packet.Data[18:22])),
			Damage:      int32(int16(binary.LittleEndian.Uint16(packet.Data[22:24]))),
			HitCount:    binary.LittleEndian.Uint16(packet.Data[24:26]),
			Action:      packet.Data[26],
			LeftDamage:  int32(int16(binary.LittleEndian.Uint16(packet.Data[27:29]))),
		}, true, nil
	case 0x02E1:
		if len(packet.Data) < 33 {
			return ActorActionNotify{}, false, fmt.Errorf("ZC_NOTIFY_ACT2 too short: %d", len(packet.Data))
		}
		return ActorActionNotify{
			SourceID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
			TargetID:    binary.LittleEndian.Uint32(packet.Data[6:10]),
			ServerTick:  binary.LittleEndian.Uint32(packet.Data[10:14]),
			SourceSpeed: int32(binary.LittleEndian.Uint32(packet.Data[14:18])),
			TargetSpeed: int32(binary.LittleEndian.Uint32(packet.Data[18:22])),
			Damage:      int32(binary.LittleEndian.Uint32(packet.Data[22:26])),
			HitCount:    binary.LittleEndian.Uint16(packet.Data[26:28]),
			Action:      packet.Data[28],
			LeftDamage:  int32(binary.LittleEndian.Uint32(packet.Data[29:33])),
		}, true, nil
	case 0x08C8:
		if len(packet.Data) < 34 {
			return ActorActionNotify{}, false, fmt.Errorf("ZC_NOTIFY_ACT3 too short: %d", len(packet.Data))
		}
		return ActorActionNotify{
			SourceID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
			TargetID:    binary.LittleEndian.Uint32(packet.Data[6:10]),
			ServerTick:  binary.LittleEndian.Uint32(packet.Data[10:14]),
			SourceSpeed: int32(binary.LittleEndian.Uint32(packet.Data[14:18])),
			TargetSpeed: int32(binary.LittleEndian.Uint32(packet.Data[18:22])),
			Damage:      int32(binary.LittleEndian.Uint32(packet.Data[22:26])),
			HitCount:    binary.LittleEndian.Uint16(packet.Data[27:29]),
			Action:      packet.Data[29],
			LeftDamage:  int32(binary.LittleEndian.Uint32(packet.Data[30:34])),
		}, true, nil
	case 0x0114:
		if len(packet.Data) < 31 {
			return ActorActionNotify{}, false, fmt.Errorf("ZC_NOTIFY_SKILL too short: %d", len(packet.Data))
		}
		return ActorActionNotify{
			SkillID:     binary.LittleEndian.Uint16(packet.Data[2:4]),
			SourceID:    binary.LittleEndian.Uint32(packet.Data[4:8]),
			TargetID:    binary.LittleEndian.Uint32(packet.Data[8:12]),
			ServerTick:  binary.LittleEndian.Uint32(packet.Data[12:16]),
			SourceSpeed: int32(binary.LittleEndian.Uint32(packet.Data[16:20])),
			TargetSpeed: int32(binary.LittleEndian.Uint32(packet.Data[20:24])),
			Damage:      int32(int16(binary.LittleEndian.Uint16(packet.Data[24:26]))),
			SkillLevel:  binary.LittleEndian.Uint16(packet.Data[26:28]),
			HitCount:    binary.LittleEndian.Uint16(packet.Data[28:30]),
			Action:      packet.Data[30],
		}, true, nil
	case 0x01DE:
		if len(packet.Data) < 33 {
			return ActorActionNotify{}, false, fmt.Errorf("ZC_NOTIFY_SKILL2 too short: %d", len(packet.Data))
		}
		return ActorActionNotify{
			SkillID:     binary.LittleEndian.Uint16(packet.Data[2:4]),
			SourceID:    binary.LittleEndian.Uint32(packet.Data[4:8]),
			TargetID:    binary.LittleEndian.Uint32(packet.Data[8:12]),
			ServerTick:  binary.LittleEndian.Uint32(packet.Data[12:16]),
			SourceSpeed: int32(binary.LittleEndian.Uint32(packet.Data[16:20])),
			TargetSpeed: int32(binary.LittleEndian.Uint32(packet.Data[20:24])),
			Damage:      int32(binary.LittleEndian.Uint32(packet.Data[24:28])),
			SkillLevel:  binary.LittleEndian.Uint16(packet.Data[28:30]),
			HitCount:    binary.LittleEndian.Uint16(packet.Data[30:32]),
			Action:      packet.Data[32],
		}, true, nil
	default:
		return ActorActionNotify{}, false, nil
	}
}

func ParseActorLookChange(packet Packet) (ActorLookChange, bool, error) {
	switch packet.ID {
	case 0x00C3:
		if len(packet.Data) < 8 {
			return ActorLookChange{}, false, fmt.Errorf("ZC_CHANGE_LOOK too short: %d", len(packet.Data))
		}
		return ActorLookChange{
			ID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
			Type:  packet.Data[6],
			Value: uint32(packet.Data[7]),
		}, true, nil
	case 0x01D7:
		if len(packet.Data) < 11 {
			return ActorLookChange{}, false, fmt.Errorf("ZC_CHANGE_LOOK2 too short: %d", len(packet.Data))
		}
		return ActorLookChange{
			ID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
			Type:  packet.Data[6],
			Value: binary.LittleEndian.Uint32(packet.Data[7:11]),
		}, true, nil
	default:
		return ActorLookChange{}, false, nil
	}
}

func ParseActorDirectionChange(packet Packet) (ActorDirectionChange, bool, error) {
	if packet.ID != 0x009C {
		return ActorDirectionChange{}, false, nil
	}
	if len(packet.Data) < 9 {
		return ActorDirectionChange{}, false, fmt.Errorf("ZC_CHANGE_DIRECTION too short: %d", len(packet.Data))
	}
	return ActorDirectionChange{
		ID:      binary.LittleEndian.Uint32(packet.Data[2:6]),
		HeadDir: uint8(binary.LittleEndian.Uint16(packet.Data[6:8])),
		Dir:     packet.Data[8] & 7,
	}, true, nil
}

func ParseActorStateChange(packet Packet) (ActorStateChange, bool, error) {
	switch packet.ID {
	case 0x0119:
		if len(packet.Data) < 13 {
			return ActorStateChange{}, false, fmt.Errorf("ZC_STATE_CHANGE too short: %d", len(packet.Data))
		}
		return ActorStateChange{
			ID:          binary.LittleEndian.Uint32(packet.Data[2:6]),
			BodyState:   binary.LittleEndian.Uint16(packet.Data[6:8]),
			HealthState: binary.LittleEndian.Uint16(packet.Data[8:10]),
			EffectState: uint32(binary.LittleEndian.Uint16(packet.Data[10:12])),
		}, true, nil
	case 0x0229:
		if len(packet.Data) < 15 {
			return ActorStateChange{}, false, fmt.Errorf("ZC_STATE_CHANGE3 too short: %d", len(packet.Data))
		}
		return ActorStateChange{
			ID:          binary.LittleEndian.Uint32(packet.Data[2:6]),
			BodyState:   binary.LittleEndian.Uint16(packet.Data[6:8]),
			HealthState: binary.LittleEndian.Uint16(packet.Data[8:10]),
			EffectState: binary.LittleEndian.Uint32(packet.Data[10:14]),
		}, true, nil
	default:
		return ActorStateChange{}, false, nil
	}
}

func ParseActorBladeStop(packet Packet) (ActorBladeStop, bool, error) {
	if packet.ID != 0x01D1 {
		return ActorBladeStop{}, false, nil
	}
	if len(packet.Data) < 14 {
		return ActorBladeStop{}, false, fmt.Errorf("ZC_BLADESTOP too short: %d", len(packet.Data))
	}
	return ActorBladeStop{
		SourceID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		TargetID: binary.LittleEndian.Uint32(packet.Data[6:10]),
		Active:   binary.LittleEndian.Uint32(packet.Data[10:14]) != 0,
	}, true, nil
}

func ParseActorVanish(packet Packet) (ActorVanish, bool, error) {
	if packet.ID != 0x0080 {
		return ActorVanish{}, false, nil
	}
	if len(packet.Data) < 7 {
		return ActorVanish{}, false, fmt.Errorf("ZC_NOTIFY_VANISH too short: %d", len(packet.Data))
	}
	return ActorVanish{
		ID:     binary.LittleEndian.Uint32(packet.Data[2:6]),
		Reason: packet.Data[6],
	}, true, nil
}

func ParseActorResurrection(packet Packet) (ActorResurrection, bool, error) {
	if packet.ID != 0x0148 {
		return ActorResurrection{}, false, nil
	}
	if len(packet.Data) < 8 {
		return ActorResurrection{}, false, fmt.Errorf("ZC_RESURRECTION too short: %d", len(packet.Data))
	}
	return ActorResurrection{
		ID:   binary.LittleEndian.Uint32(packet.Data[2:6]),
		Type: binary.LittleEndian.Uint16(packet.Data[6:8]),
	}, true, nil
}

func ParseSelfMoveAck(packet Packet) (SelfMoveAck, bool, error) {
	if packet.ID != 0x0087 {
		return SelfMoveAck{}, false, nil
	}
	if len(packet.Data) < 12 {
		return SelfMoveAck{}, false, fmt.Errorf("ZC_NOTIFY_PLAYERMOVE too short: %d", len(packet.Data))
	}
	fromX, fromY, toX, toY := unpackMovePos(packet.Data[6:12])
	return SelfMoveAck{
		ServerTick: binary.LittleEndian.Uint32(packet.Data[2:6]),
		FromX:      fromX,
		FromY:      fromY,
		ToX:        toX,
		ToY:        toY,
	}, true, nil
}

func ParseActorSetPosition(packet Packet) (ActorSetPosition, bool, error) {
	if packet.ID != 0x0088 {
		return ActorSetPosition{}, false, nil
	}
	if len(packet.Data) < 10 {
		return ActorSetPosition{}, false, fmt.Errorf("ZC_STOPMOVE too short: %d", len(packet.Data))
	}
	return ActorSetPosition{
		ID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		X:  int(int16(binary.LittleEndian.Uint16(packet.Data[6:8]))),
		Y:  int(int16(binary.LittleEndian.Uint16(packet.Data[8:10]))),
	}, true, nil
}

func ParseActorJumpPosition(packet Packet) (ActorJumpPosition, bool, error) {
	if packet.ID != 0x01FF {
		return ActorJumpPosition{}, false, nil
	}
	if len(packet.Data) < 10 {
		return ActorJumpPosition{}, false, fmt.Errorf("ZC_HIGHJUMP too short: %d", len(packet.Data))
	}
	return ActorJumpPosition{
		ID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		X:  int(int16(binary.LittleEndian.Uint16(packet.Data[6:8]))),
		Y:  int(int16(binary.LittleEndian.Uint16(packet.Data[8:10]))),
	}, true, nil
}

func ParseAttackFailureForDistance(packet Packet) (AttackFailureForDistance, bool, error) {
	if packet.ID != 0x0139 {
		return AttackFailureForDistance{}, false, nil
	}
	if len(packet.Data) < 16 {
		return AttackFailureForDistance{}, false, fmt.Errorf("ZC_ATTACK_FAILURE_FOR_DISTANCE too short: %d", len(packet.Data))
	}
	return AttackFailureForDistance{
		TargetID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
		TargetX:     int(int16(binary.LittleEndian.Uint16(packet.Data[6:8]))),
		TargetY:     int(int16(binary.LittleEndian.Uint16(packet.Data[8:10]))),
		SourceX:     int(int16(binary.LittleEndian.Uint16(packet.Data[10:12]))),
		SourceY:     int(int16(binary.LittleEndian.Uint16(packet.Data[12:14]))),
		AttackRange: int(int16(binary.LittleEndian.Uint16(packet.Data[14:16]))),
	}, true, nil
}

func unpackPos(data []byte) (x, y, dir int) {
	if len(data) < 3 {
		return 0, 0, 0
	}
	return int(data[0])<<2 | int(data[1]>>6),
		int(data[1]&0x3f)<<4 | int(data[2]>>4),
		int(data[2] & 0x0f)
}

func unpackMovePos(data []byte) (fromX, fromY, toX, toY int) {
	if len(data) < 6 {
		return 0, 0, 0, 0
	}
	a, b, c, d, e := data[0], data[1], data[2], data[3], data[4]
	return int(a)<<2 | int((b&0xc0)>>6),
		int(b&0x3f)<<4 | int((c&0xf0)>>4),
		int((d&0xfc)>>2) | int(c&0x0f)<<6,
		int(d&0x03)<<8 | int(e)
}
