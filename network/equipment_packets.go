package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketCZReqCartOff         uint16 = 0x012A
	PacketCZEquipWinMicroscope uint16 = 0x02D6
	PacketZCEquipWinMicroscope uint16 = 0x02D7
	PacketCZConfig             uint16 = 0x02D8
	PacketZCConfigNotify       uint16 = 0x02DA

	ConfigOpenEquipmentWindow uint32 = 0
)

type ViewedEquipment struct {
	Name        string
	Job         uint16
	Head        uint16
	HeadTop     uint16
	HeadMid     uint16
	HeadLow     uint16
	HeadPalette uint16
	BodyPalette uint16
	Sex         byte
	Items       []InventoryItem
}

func ParseViewedEquipment(packet Packet) (ViewedEquipment, bool, error) {
	if packet.ID != PacketZCEquipWinMicroscope {
		return ViewedEquipment{}, false, nil
	}
	const (
		headerSize = 43
		entrySize  = 26
	)
	if len(packet.Data) < headerSize {
		return ViewedEquipment{}, false, fmt.Errorf("ZC_EQUIPWIN_MICROSCOPE too short: %d", len(packet.Data))
	}
	if int(binary.LittleEndian.Uint16(packet.Data[2:4])) != len(packet.Data) {
		return ViewedEquipment{}, false, fmt.Errorf("ZC_EQUIPWIN_MICROSCOPE length mismatch: packet=%d data=%d", binary.LittleEndian.Uint16(packet.Data[2:4]), len(packet.Data))
	}
	if (len(packet.Data)-headerSize)%entrySize != 0 {
		return ViewedEquipment{}, false, fmt.Errorf("ZC_EQUIPWIN_MICROSCOPE invalid item length: %d", len(packet.Data))
	}
	view := ViewedEquipment{
		Name:        trimPacketString(packet.Data[4:28]),
		Job:         binary.LittleEndian.Uint16(packet.Data[28:30]),
		Head:        binary.LittleEndian.Uint16(packet.Data[30:32]),
		HeadLow:     binary.LittleEndian.Uint16(packet.Data[32:34]),
		HeadMid:     binary.LittleEndian.Uint16(packet.Data[34:36]),
		HeadTop:     binary.LittleEndian.Uint16(packet.Data[36:38]),
		HeadPalette: binary.LittleEndian.Uint16(packet.Data[38:40]),
		BodyPalette: binary.LittleEndian.Uint16(packet.Data[40:42]),
		Sex:         packet.Data[42],
		Items:       make([]InventoryItem, 0, (len(packet.Data)-headerSize)/entrySize),
	}
	for offset := headerSize; offset+entrySize <= len(packet.Data); offset += entrySize {
		wearState := binary.LittleEndian.Uint16(packet.Data[offset+8 : offset+10])
		view.Items = append(view.Items, InventoryItem{
			Index:      binary.LittleEndian.Uint16(packet.Data[offset : offset+2]),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[offset+2 : offset+4]),
			Type:       packet.Data[offset+4],
			Identified: packet.Data[offset+5] != 0,
			Location:   binary.LittleEndian.Uint16(packet.Data[offset+6 : offset+8]),
			Amount:     1,
			Equip:      true,
			Equipped:   wearState != 0,
			Damaged:    packet.Data[offset+10] != 0,
			Refine:     packet.Data[offset+11],
			Cards:      readItemCards(packet.Data, offset+12),
		})
	}
	return view, true, nil
}

func ParseShowEquipConfig(packet Packet) (bool, bool, error) {
	if packet.ID != PacketZCConfigNotify {
		return false, false, nil
	}
	if len(packet.Data) < 3 {
		return false, false, fmt.Errorf("ZC_CONFIG_NOTIFY too short: %d", len(packet.Data))
	}
	return packet.Data[2] != 0, true, nil
}

func BuildViewPlayerEquipmentPacket(targetID uint32) []byte {
	var w Writer
	w.Uint16(PacketCZEquipWinMicroscope)
	w.Uint32(targetID)
	return w.Bytes()
}

func BuildShowEquipConfigPacket(enabled bool) []byte {
	value := uint32(0)
	if enabled {
		value = 1
	}
	var w Writer
	w.Uint16(PacketCZConfig)
	w.Uint32(ConfigOpenEquipmentWindow)
	w.Uint32(value)
	return w.Bytes()
}

func BuildRemoveOptionPacket() []byte {
	var w Writer
	w.Uint16(PacketCZReqCartOff)
	return w.Bytes()
}

func (c *Client) SendViewPlayerEquipment(targetID uint32) error {
	packet := BuildViewPlayerEquipmentPacket(targetID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_EQUIPWIN_MICROSCOPE opcode=0x%04X target=%d client_date=%d", ID(packet), targetID, c.clientDate)
	} else {
		glog.Warnf("send CZ_EQUIPWIN_MICROSCOPE failed opcode=0x%04X target=%d client_date=%d: %v", ID(packet), targetID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendRemoveOption() error {
	packet := BuildRemoveOptionPacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_CARTOFF opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_CARTOFF failed opcode=0x%04X client_date=%d: %v", ID(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendShowEquipConfig(enabled bool) error {
	packet := BuildShowEquipConfigPacket(enabled)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CONFIG opcode=0x%04X config=%d value=%t client_date=%d", ID(packet), ConfigOpenEquipmentWindow, enabled, c.clientDate)
	} else {
		glog.Warnf("send CZ_CONFIG failed opcode=0x%04X config=%d value=%t client_date=%d: %v", ID(packet), ConfigOpenEquipmentWindow, enabled, c.clientDate, err)
	}
	return err
}
