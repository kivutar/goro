package network

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	StatusSpeed       uint16 = 0
	StatusBaseExp     uint16 = 1
	StatusJobExp      uint16 = 2
	StatusHP          uint16 = 5
	StatusMaxHP       uint16 = 6
	StatusSP          uint16 = 7
	StatusMaxSP       uint16 = 8
	StatusPoint       uint16 = 9
	StatusBaseLevel   uint16 = 11
	StatusSkillPoint  uint16 = 12
	StatusStr         uint16 = 13
	StatusAgi         uint16 = 14
	StatusVit         uint16 = 15
	StatusInt         uint16 = 16
	StatusDex         uint16 = 17
	StatusLuk         uint16 = 18
	StatusZeny        uint16 = 20
	StatusNextBaseExp uint16 = 22
	StatusNextJobExp  uint16 = 23
	StatusWeight      uint16 = 24
	StatusMaxWeight   uint16 = 25
	StatusUStr        uint16 = 32
	StatusUAgi        uint16 = 33
	StatusUVit        uint16 = 34
	StatusUInt        uint16 = 35
	StatusUDex        uint16 = 36
	StatusULuk        uint16 = 37
	StatusJobLevel    uint16 = 55
	StatusMercFlee    uint16 = 165
	StatusMercKills   uint16 = 189
	StatusMercFaith   uint16 = 190
)

type ParameterChange struct {
	VarID uint16
	Value int64
}

type StatusSnapshot struct {
	Points int
	Str    int
	Agi    int
	Vit    int
	Int    int
	Dex    int
	Luk    int

	StrBonus int
	AgiBonus int
	VitBonus int
	IntBonus int
	DexBonus int
	LukBonus int

	StrCost int
	AgiCost int
	VitCost int
	IntCost int
	DexCost int
	LukCost int

	Attack        int
	AttackBonus   int
	MatkMax       int
	MatkMin       int
	Defense       int
	DefenseBonus  int
	MDefense      int
	MDefenseBonus int
	Hit           int
	Flee          int
	FleeBonus     int
	Critical      int
	ASPD          int
	ASPDBonus     int
}

type StatusChangeAck struct {
	StatusID uint16
	Success  bool
	Value    int
}

type Recovery struct {
	StatusID uint16
	Amount   int
}

type StatusEffectChange struct {
	StatusID    uint16
	ActorID     uint32
	Active      bool
	HasDuration bool
	Duration    time.Duration
	HasValues   bool
	Values      [3]int32
}

func ParseParameterChange(packet Packet) (ParameterChange, bool, error) {
	if packet.ID == 0x00BE {
		if len(packet.Data) < 5 {
			return ParameterChange{}, false, fmt.Errorf("ZC_STATUS_CHANGE too short: %d", len(packet.Data))
		}
		return ParameterChange{
			VarID: binary.LittleEndian.Uint16(packet.Data[2:4]),
			Value: int64(packet.Data[4]),
		}, true, nil
	}
	if packet.ID != 0x00B0 && packet.ID != 0x00B1 {
		return ParameterChange{}, false, nil
	}
	if len(packet.Data) < 8 {
		return ParameterChange{}, false, fmt.Errorf("ZC_PAR_CHANGE too short: %d", len(packet.Data))
	}
	return ParameterChange{
		VarID: binary.LittleEndian.Uint16(packet.Data[2:4]),
		Value: int64(binary.LittleEndian.Uint32(packet.Data[4:8])),
	}, true, nil
}

func ParseStatusSnapshot(packet Packet) (StatusSnapshot, bool, error) {
	if packet.ID != 0x00BD {
		return StatusSnapshot{}, false, nil
	}
	if len(packet.Data) < 44 {
		return StatusSnapshot{}, false, fmt.Errorf("ZC_STATUS too short: %d", len(packet.Data))
	}
	return StatusSnapshot{
		Points:        int(binary.LittleEndian.Uint16(packet.Data[2:4])),
		Str:           int(packet.Data[4]),
		StrCost:       int(packet.Data[5]),
		Agi:           int(packet.Data[6]),
		AgiCost:       int(packet.Data[7]),
		Vit:           int(packet.Data[8]),
		VitCost:       int(packet.Data[9]),
		Int:           int(packet.Data[10]),
		IntCost:       int(packet.Data[11]),
		Dex:           int(packet.Data[12]),
		DexCost:       int(packet.Data[13]),
		Luk:           int(packet.Data[14]),
		LukCost:       int(packet.Data[15]),
		Attack:        int(binary.LittleEndian.Uint16(packet.Data[16:18])),
		AttackBonus:   int(binary.LittleEndian.Uint16(packet.Data[18:20])),
		MatkMax:       int(binary.LittleEndian.Uint16(packet.Data[20:22])),
		MatkMin:       int(binary.LittleEndian.Uint16(packet.Data[22:24])),
		Defense:       int(binary.LittleEndian.Uint16(packet.Data[24:26])),
		DefenseBonus:  int(binary.LittleEndian.Uint16(packet.Data[26:28])),
		MDefense:      int(binary.LittleEndian.Uint16(packet.Data[28:30])),
		MDefenseBonus: int(binary.LittleEndian.Uint16(packet.Data[30:32])),
		Hit:           int(binary.LittleEndian.Uint16(packet.Data[32:34])),
		Flee:          int(binary.LittleEndian.Uint16(packet.Data[34:36])),
		FleeBonus:     int(binary.LittleEndian.Uint16(packet.Data[36:38])),
		Critical:      int(binary.LittleEndian.Uint16(packet.Data[38:40])),
		ASPD:          int(binary.LittleEndian.Uint16(packet.Data[40:42])),
		ASPDBonus:     int(binary.LittleEndian.Uint16(packet.Data[42:44])),
	}, true, nil
}

func ParseStatusChangeAck(packet Packet) (StatusChangeAck, bool, error) {
	if packet.ID != 0x00BC {
		return StatusChangeAck{}, false, nil
	}
	if len(packet.Data) < 6 {
		return StatusChangeAck{}, false, fmt.Errorf("ZC_STATUS_CHANGE_ACK too short: %d", len(packet.Data))
	}
	return StatusChangeAck{
		StatusID: binary.LittleEndian.Uint16(packet.Data[2:4]),
		Success:  packet.Data[4] != 0,
		Value:    int(packet.Data[5]),
	}, true, nil
}

func ParseRecovery(packet Packet) (Recovery, bool, error) {
	switch packet.ID {
	case 0x013D:
		if len(packet.Data) < 6 {
			return Recovery{}, false, fmt.Errorf("ZC_RECOVERY too short: %d", len(packet.Data))
		}
		return Recovery{
			StatusID: binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount:   int(binary.LittleEndian.Uint16(packet.Data[4:6])),
		}, true, nil
	case 0x0A27:
		if len(packet.Data) < 8 {
			return Recovery{}, false, fmt.Errorf("ZC_RECOVERY2 too short: %d", len(packet.Data))
		}
		return Recovery{
			StatusID: binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount:   int(binary.LittleEndian.Uint32(packet.Data[4:8])),
		}, true, nil
	default:
		return Recovery{}, false, nil
	}
}

func ParseStatusEffectChange(packet Packet) (StatusEffectChange, bool, error) {
	switch packet.ID {
	case 0x0196:
		if len(packet.Data) < 9 {
			return StatusEffectChange{}, false, fmt.Errorf("ZC_MSG_STATE_CHANGE too short: %d", len(packet.Data))
		}
		return StatusEffectChange{
			StatusID: binary.LittleEndian.Uint16(packet.Data[2:4]),
			ActorID:  binary.LittleEndian.Uint32(packet.Data[4:8]),
			Active:   packet.Data[8] != 0,
		}, true, nil
	case 0x043F:
		if len(packet.Data) < 25 {
			return StatusEffectChange{}, false, fmt.Errorf("ZC_MSG_STATE_CHANGE2 too short: %d", len(packet.Data))
		}
		duration := binary.LittleEndian.Uint32(packet.Data[9:13])
		change := statusEffectChangeWithDuration(
			binary.LittleEndian.Uint16(packet.Data[2:4]),
			binary.LittleEndian.Uint32(packet.Data[4:8]),
			packet.Data[8] != 0,
			duration,
		)
		change.HasValues = true
		change.Values = [3]int32{
			int32(binary.LittleEndian.Uint32(packet.Data[13:17])),
			int32(binary.LittleEndian.Uint32(packet.Data[17:21])),
			int32(binary.LittleEndian.Uint32(packet.Data[21:25])),
		}
		return change, true, nil
	case 0x0983:
		if len(packet.Data) < 29 {
			return StatusEffectChange{}, false, fmt.Errorf("ZC_MSG_STATE_CHANGE3 too short: %d", len(packet.Data))
		}
		remaining := binary.LittleEndian.Uint32(packet.Data[13:17])
		change := statusEffectChangeWithDuration(
			binary.LittleEndian.Uint16(packet.Data[2:4]),
			binary.LittleEndian.Uint32(packet.Data[4:8]),
			packet.Data[8] != 0,
			remaining,
		)
		change.HasValues = true
		change.Values = [3]int32{
			int32(binary.LittleEndian.Uint32(packet.Data[17:21])),
			int32(binary.LittleEndian.Uint32(packet.Data[21:25])),
			int32(binary.LittleEndian.Uint32(packet.Data[25:29])),
		}
		return change, true, nil
	default:
		return StatusEffectChange{}, false, nil
	}
}

func statusEffectChangeWithDuration(statusID uint16, actorID uint32, active bool, durationMS uint32) StatusEffectChange {
	change := StatusEffectChange{
		StatusID: statusID,
		ActorID:  actorID,
		Active:   active,
	}
	if active && durationMS > 0 {
		change.HasDuration = true
		change.Duration = time.Duration(durationMS) * time.Millisecond
	}
	return change
}

func BuildStatusIncreasePacket(statusID uint16) []byte {
	packet := make([]byte, 5)
	binary.LittleEndian.PutUint16(packet[0:2], 0x00BB)
	binary.LittleEndian.PutUint16(packet[2:4], statusID)
	packet[4] = 1
	return packet
}
