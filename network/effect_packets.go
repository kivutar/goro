package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketCZLessEffect    uint16 = 0x021D
	PacketZCLessEffect    uint16 = 0x021E
	PacketZCNotifyEffect  uint16 = 0x019B
	PacketZCNotifyEffect2 uint16 = 0x01F3
	PacketZCMVP           uint16 = 0x010C

	SpecialEffectBaseLevelUp = 0
	SpecialEffectJobLevelUp  = 1
)

type SpecialEffectNotify struct {
	AID      uint32
	EffectID uint32
}

type MVPNotify struct {
	AID uint32
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
