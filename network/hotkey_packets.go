package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketZCShortcutKeyList   uint16 = 0x02B9
	PacketCZShortcutKeyChange uint16 = 0x02BA

	HotkeyTypeItem  uint8 = 0
	HotkeyTypeSkill uint8 = 1

	HotkeyListSlots2008 = 27
)

type HotkeySlot struct {
	Type  uint8
	ID    uint32
	Level uint16
}

type HotkeyList struct {
	Slots []HotkeySlot
}

func ParseHotkeyList(packet Packet) (HotkeyList, bool, error) {
	if packet.ID != PacketZCShortcutKeyList {
		return HotkeyList{}, false, nil
	}
	const (
		headerSize = 2
		entrySize  = 7
	)
	if len(packet.Data) < headerSize+HotkeyListSlots2008*entrySize {
		return HotkeyList{}, false, fmt.Errorf("ZC_SHORTCUT_KEY_LIST too short: %d", len(packet.Data))
	}
	slots := make([]HotkeySlot, HotkeyListSlots2008)
	for i := 0; i < HotkeyListSlots2008; i++ {
		offset := headerSize + i*entrySize
		slots[i] = HotkeySlot{
			Type:  packet.Data[offset],
			ID:    binary.LittleEndian.Uint32(packet.Data[offset+1 : offset+5]),
			Level: binary.LittleEndian.Uint16(packet.Data[offset+5 : offset+7]),
		}
	}
	return HotkeyList{Slots: slots}, true, nil
}

func BuildHotkeyPacket(index uint16, slot HotkeySlot) []byte {
	var w Writer
	w.Uint16(PacketCZShortcutKeyChange)
	w.Uint16(index)
	w.Uint8(slot.Type)
	w.Uint32(slot.ID)
	w.Uint16(slot.Level)
	return w.Bytes()
}

func (c *Client) SendHotkey(index uint16, slot HotkeySlot) error {
	packet := BuildHotkeyPacket(index, slot)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_SHORTCUT_KEY_CHANGE opcode=0x%04X index=%d type=%d id=%d level=%d client_date=%d", ID(packet), index, slot.Type, slot.ID, slot.Level, c.clientDate)
	} else {
		glog.Warnf("send CZ_SHORTCUT_KEY_CHANGE failed opcode=0x%04X len=%d index=%d type=%d id=%d level=%d client_date=%d: %v", ID(packet), len(packet), index, slot.Type, slot.ID, slot.Level, c.clientDate, err)
	}
	return err
}
