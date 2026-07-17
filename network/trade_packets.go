package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
	"strings"
)

const (
	PacketCZReqExchangeItem      uint16 = 0x00E4
	PacketCZAckExchangeItem      uint16 = 0x00E6
	PacketCZAddExchangeItem      uint16 = 0x00E8
	PacketCZConcludeExchangeItem uint16 = 0x00EB
	PacketCZCancelExchangeItem   uint16 = 0x00ED
	PacketCZExecExchangeItem     uint16 = 0x00EF

	PacketZCReqExchangeItem      uint16 = 0x00E5
	PacketZCAckExchangeItem      uint16 = 0x00E7
	PacketZCAddExchangeItem      uint16 = 0x00E9
	PacketZCAckAddExchangeItem   uint16 = 0x00EA
	PacketZCConcludeExchangeItem uint16 = 0x00EC
	PacketZCCancelExchangeItem   uint16 = 0x00EE
	PacketZCExecExchangeItem     uint16 = 0x00F0
	PacketZCExchangeItemUndo     uint16 = 0x00F1
	PacketZCReqExchangeItem2     uint16 = 0x01F4
	PacketZCAckExchangeItem2     uint16 = 0x01F5
)

const (
	TradeAckAccept uint8 = 3
	TradeAckCancel uint8 = 4
)

type TradeRequest struct {
	Name     string
	TargetID uint32
	Level    uint16
}

type TradeResponse struct {
	Result   uint8
	TargetID uint32
	Level    uint16
}

type TradeItem struct {
	Amount     uint32
	ItemID     uint16
	Identified bool
	Damaged    bool
	Refine     uint8
	Cards      [4]uint16
}

type TradeAddItemAck struct {
	Index  uint16
	Result uint8
}

type TradeConclude struct {
	Other bool
}

type TradeExec struct {
	Result uint8
}

func ParseTradeRequest(packet Packet) (TradeRequest, bool, error) {
	switch packet.ID {
	case PacketZCReqExchangeItem:
		if len(packet.Data) < 26 {
			return TradeRequest{}, false, fmt.Errorf("ZC_REQ_EXCHANGE_ITEM too short: %d", len(packet.Data))
		}
		return TradeRequest{Name: trimPacketString(packet.Data[2:26])}, true, nil
	case PacketZCReqExchangeItem2:
		if len(packet.Data) < 32 {
			return TradeRequest{}, false, fmt.Errorf("ZC_REQ_EXCHANGE_ITEM2 too short: %d", len(packet.Data))
		}
		return TradeRequest{
			Name:     trimPacketString(packet.Data[2:26]),
			TargetID: binary.LittleEndian.Uint32(packet.Data[26:30]),
			Level:    binary.LittleEndian.Uint16(packet.Data[30:32]),
		}, true, nil
	default:
		return TradeRequest{}, false, nil
	}
}

func ParseTradeResponse(packet Packet) (TradeResponse, bool, error) {
	switch packet.ID {
	case PacketZCAckExchangeItem:
		if len(packet.Data) < 3 {
			return TradeResponse{}, false, fmt.Errorf("ZC_ACK_EXCHANGE_ITEM too short: %d", len(packet.Data))
		}
		return TradeResponse{Result: packet.Data[2]}, true, nil
	case PacketZCAckExchangeItem2:
		if len(packet.Data) < 9 {
			return TradeResponse{}, false, fmt.Errorf("ZC_ACK_EXCHANGE_ITEM2 too short: %d", len(packet.Data))
		}
		return TradeResponse{
			Result:   packet.Data[2],
			TargetID: binary.LittleEndian.Uint32(packet.Data[3:7]),
			Level:    binary.LittleEndian.Uint16(packet.Data[7:9]),
		}, true, nil
	default:
		return TradeResponse{}, false, nil
	}
}

func ParseTradeItem(packet Packet) (TradeItem, bool, error) {
	if packet.ID != PacketZCAddExchangeItem {
		return TradeItem{}, false, nil
	}
	if len(packet.Data) < 19 {
		return TradeItem{}, false, fmt.Errorf("ZC_ADD_EXCHANGE_ITEM too short: %d", len(packet.Data))
	}
	item := TradeItem{
		Amount:     binary.LittleEndian.Uint32(packet.Data[2:6]),
		ItemID:     binary.LittleEndian.Uint16(packet.Data[6:8]),
		Identified: packet.Data[8] != 0,
		Damaged:    packet.Data[9] != 0,
		Refine:     packet.Data[10],
	}
	for i := range item.Cards {
		item.Cards[i] = binary.LittleEndian.Uint16(packet.Data[11+i*2 : 13+i*2])
	}
	return item, true, nil
}

func ParseTradeAddItemAck(packet Packet) (TradeAddItemAck, bool, error) {
	if packet.ID != PacketZCAckAddExchangeItem {
		return TradeAddItemAck{}, false, nil
	}
	if len(packet.Data) < 5 {
		return TradeAddItemAck{}, false, fmt.Errorf("ZC_ACK_ADD_EXCHANGE_ITEM too short: %d", len(packet.Data))
	}
	return TradeAddItemAck{
		Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
		Result: packet.Data[4],
	}, true, nil
}

func ParseTradeConclude(packet Packet) (TradeConclude, bool, error) {
	if packet.ID != PacketZCConcludeExchangeItem {
		return TradeConclude{}, false, nil
	}
	if len(packet.Data) < 3 {
		return TradeConclude{}, false, fmt.Errorf("ZC_CONCLUDE_EXCHANGE_ITEM too short: %d", len(packet.Data))
	}
	return TradeConclude{Other: packet.Data[2] != 0}, true, nil
}

func ParseTradeExec(packet Packet) (TradeExec, bool, error) {
	if packet.ID != PacketZCExecExchangeItem {
		return TradeExec{}, false, nil
	}
	if len(packet.Data) < 3 {
		return TradeExec{}, false, fmt.Errorf("ZC_EXEC_EXCHANGE_ITEM too short: %d", len(packet.Data))
	}
	return TradeExec{Result: packet.Data[2]}, true, nil
}

func ParseTradeCanceled(packet Packet) bool {
	return packet.ID == PacketZCCancelExchangeItem
}

func ParseTradeUndo(packet Packet) bool {
	return packet.ID == PacketZCExchangeItemUndo
}

func BuildTradeRequestPacket(targetID uint32) []byte {
	var w Writer
	w.Uint16(PacketCZReqExchangeItem)
	w.Uint32(targetID)
	return w.Bytes()
}

func BuildTradeAckPacket(accept bool) []byte {
	result := TradeAckCancel
	if accept {
		result = TradeAckAccept
	}
	var w Writer
	w.Uint16(PacketCZAckExchangeItem)
	w.Uint8(result)
	return w.Bytes()
}

func BuildTradeAddItemPacket(index uint16, amount uint32) []byte {
	var w Writer
	w.Uint16(PacketCZAddExchangeItem)
	w.Uint16(index)
	w.Uint32(amount)
	return w.Bytes()
}

func BuildTradeConcludePacket() []byte {
	var w Writer
	w.Uint16(PacketCZConcludeExchangeItem)
	return w.Bytes()
}

func BuildTradeCancelPacket() []byte {
	var w Writer
	w.Uint16(PacketCZCancelExchangeItem)
	return w.Bytes()
}

func BuildTradeCommitPacket() []byte {
	var w Writer
	w.Uint16(PacketCZExecExchangeItem)
	return w.Bytes()
}

func (c *Client) SendTradeRequest(targetID uint32) error {
	packet := BuildTradeRequestPacket(targetID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_EXCHANGE_ITEM opcode=0x%04X target=%d client_date=%d", ID(packet), targetID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_EXCHANGE_ITEM failed opcode=0x%04X target=%d client_date=%d: %v", ID(packet), targetID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendTradeAck(accept bool) error {
	packet := BuildTradeAckPacket(accept)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ACK_EXCHANGE_ITEM opcode=0x%04X accept=%t client_date=%d", ID(packet), accept, c.clientDate)
	} else {
		glog.Warnf("send CZ_ACK_EXCHANGE_ITEM failed opcode=0x%04X accept=%t client_date=%d: %v", ID(packet), accept, c.clientDate, err)
	}
	return err
}

func (c *Client) SendTradeAddItem(index uint16, amount uint32) error {
	packet := BuildTradeAddItemPacket(index, amount)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ADD_EXCHANGE_ITEM opcode=0x%04X index=%d amount=%d client_date=%d", ID(packet), index, amount, c.clientDate)
	} else {
		glog.Warnf("send CZ_ADD_EXCHANGE_ITEM failed opcode=0x%04X index=%d amount=%d client_date=%d: %v", ID(packet), index, amount, c.clientDate, err)
	}
	return err
}

func (c *Client) SendTradeConclude() error {
	packet := BuildTradeConcludePacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CONCLUDE_EXCHANGE_ITEM opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_CONCLUDE_EXCHANGE_ITEM failed opcode=0x%04X client_date=%d: %v", ID(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendTradeCancel() error {
	packet := BuildTradeCancelPacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CANCEL_EXCHANGE_ITEM opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_CANCEL_EXCHANGE_ITEM failed opcode=0x%04X client_date=%d: %v", ID(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendTradeCommit() error {
	packet := BuildTradeCommitPacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_EXEC_EXCHANGE_ITEM opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_EXEC_EXCHANGE_ITEM failed opcode=0x%04X client_date=%d: %v", ID(packet), c.clientDate, err)
	}
	return err
}

func trimPacketString(data []byte) string {
	if i := strings.IndexByte(string(data), 0); i >= 0 {
		data = data[:i]
	}
	return strings.TrimSpace(string(data))
}
