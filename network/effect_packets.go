package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketCZLessEffect        uint16 = 0x021D
	PacketZCLessEffect        uint16 = 0x021E
	PacketZCNotifyEffect      uint16 = 0x019B
	PacketZCNotifyEffect2     uint16 = 0x01F3
	PacketZCMVP               uint16 = 0x010C
	PacketZCSpirits           uint16 = 0x01D0
	PacketZCSpirits2          uint16 = 0x01E1
	PacketZCNPCShowEFSTUpdate uint16 = 0x028A

	SpecialEffectBaseLevelUp            = 0
	SpecialEffectJobLevelUp             = 1
	SpecialEffectRefineFailure          = 2
	SpecialEffectRefineSuccess          = 3
	SpecialEffectGameOver               = 4
	SpecialEffectPharmacySuccess        = 5
	SpecialEffectPharmacyFailure        = 6
	SpecialEffectSuperNoviceBaseLevelUp = 7
	SpecialEffectSuperNoviceJobLevelUp  = 8
	SpecialEffectTaekwonBaseLevelUp     = 9
)

type SpecialEffectNotify struct {
	AID      uint32
	EffectID uint32
}

type MVPNotify struct {
	AID uint32
}

type SpiritSphereCount struct {
	ActorID uint32
	Count   uint16
}

type NPCShowEffectUpdate struct {
	ActorID   uint32
	Level     int
	Opt3State uint32
}

func ParseNPCShowEffectUpdate(packet Packet) (NPCShowEffectUpdate, bool, error) {
	if packet.ID != PacketZCNPCShowEFSTUpdate {
		return NPCShowEffectUpdate{}, false, nil
	}
	if len(packet.Data) < 18 {
		return NPCShowEffectUpdate{}, false, fmt.Errorf("ZC_NPC_SHOWEFST_UPDATE too short: %d", len(packet.Data))
	}
	return NPCShowEffectUpdate{
		ActorID:   binary.LittleEndian.Uint32(packet.Data[2:6]),
		Level:     int(int32(binary.LittleEndian.Uint32(packet.Data[10:14]))),
		Opt3State: binary.LittleEndian.Uint32(packet.Data[14:18]),
	}, true, nil
}

func ParseSpiritSphereCount(packet Packet) (SpiritSphereCount, bool, error) {
	if packet.ID != PacketZCSpirits && packet.ID != PacketZCSpirits2 {
		return SpiritSphereCount{}, false, nil
	}
	if len(packet.Data) < 8 {
		return SpiritSphereCount{}, false, fmt.Errorf("ZC_SPIRITS 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	return SpiritSphereCount{
		ActorID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		Count:   binary.LittleEndian.Uint16(packet.Data[6:8]),
	}, true, nil
}

func BuildLessEffectPacket(enabled bool) []byte {
	state := uint32(0)
	if enabled {
		state = 1
	}
	var w Writer
	w.Uint16(PacketCZLessEffect)
	w.Uint32(state)
	return w.Bytes()
}

func ParseLessEffect(packet Packet) (bool, bool, error) {
	if packet.ID != PacketZCLessEffect {
		return false, false, nil
	}
	if len(packet.Data) < 6 {
		return false, false, fmt.Errorf("ZC_LESSEFFECT too short: %d", len(packet.Data))
	}
	return binary.LittleEndian.Uint32(packet.Data[2:6]) != 0, true, nil
}

func ParseSpecialEffectNotify(packet Packet) (SpecialEffectNotify, bool, error) {
	switch packet.ID {
	case PacketZCNotifyEffect, PacketZCNotifyEffect2:
	default:
		return SpecialEffectNotify{}, false, nil
	}
	if len(packet.Data) < 10 {
		return SpecialEffectNotify{}, false, fmt.Errorf("ZC_NOTIFY_EFFECT 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	return SpecialEffectNotify{
		AID:      binary.LittleEndian.Uint32(packet.Data[2:6]),
		EffectID: binary.LittleEndian.Uint32(packet.Data[6:10]),
	}, true, nil
}

func ParseMVPNotify(packet Packet) (MVPNotify, bool, error) {
	if packet.ID != PacketZCMVP {
		return MVPNotify{}, false, nil
	}
	if len(packet.Data) < 6 {
		return MVPNotify{}, false, fmt.Errorf("ZC_MVP too short: %d", len(packet.Data))
	}
	return MVPNotify{
		AID: binary.LittleEndian.Uint32(packet.Data[2:6]),
	}, true, nil
}

func (c *Client) SendLessEffect(enabled bool) error {
	packet := BuildLessEffectPacket(enabled)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_LESSEFFECT opcode=0x%04X enabled=%t client_date=%d", ID(packet), enabled, c.clientDate)
	} else {
		glog.Warnf("send CZ_LESSEFFECT failed opcode=0x%04X len=%d enabled=%t client_date=%d: %v", ID(packet), len(packet), enabled, c.clientDate, err)
	}
	return err
}
