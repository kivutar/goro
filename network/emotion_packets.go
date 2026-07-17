package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketCZReqEmotion uint16 = 0x00BF
	PacketZCEmotion    uint16 = 0x00C0
)

type EmotionNotify struct {
	GID  uint32
	Type uint8
}

func BuildEmotionPacket(emotionType uint8) []byte {
	packet := make([]byte, 3)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqEmotion)
	packet[2] = emotionType
	return packet
}

func ParseEmotionNotify(packet Packet) (EmotionNotify, bool, error) {
	if packet.ID != PacketZCEmotion {
		return EmotionNotify{}, false, nil
	}
	if len(packet.Data) < 7 {
		return EmotionNotify{}, false, fmt.Errorf("ZC_EMOTION too short: %d", len(packet.Data))
	}
	return EmotionNotify{
		GID:  binary.LittleEndian.Uint32(packet.Data[2:6]),
		Type: packet.Data[6],
	}, true, nil
}

func (c *Client) SendEmotion(emotionType uint8) error {
	packet := BuildEmotionPacket(emotionType)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_EMOTION opcode=0x%04X type=%d client_date=%d", ID(packet), emotionType, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_EMOTION failed opcode=0x%04X type=%d client_date=%d: %v", ID(packet), emotionType, c.clientDate, err)
	}
	return err
}
