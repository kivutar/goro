package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const (
	PacketCZReqCloseStore            uint16 = 0x012E
	PacketCZReqOpenStore             uint16 = 0x012F
	PacketCZReqBuyFromMC             uint16 = 0x0130
	PacketCZPCPurchaseItemListFromMC uint16 = 0x0134
	PacketCZReqOpenStore2            uint16 = 0x01B2
	PacketZCOpenStore                uint16 = 0x012D
	PacketZCStoreEntry               uint16 = 0x0131
	PacketZCDisappearEntry           uint16 = 0x0132
	PacketZCPCPurchaseItemListFromMC uint16 = 0x0133
	PacketZCPCPurchaseResultFromMC   uint16 = 0x0135
	PacketZCPCPurchaseMyItemList     uint16 = 0x0136
	PacketZCDeleteItemFromMCStore    uint16 = 0x0137
	vendingStoreNamePacketSize              = 80
	vendingItemPacketSize                   = 22
)

type VendingOpenRequest struct {
	MaxItems uint16
}

type VendingBoard struct {
	OwnerAID uint32
	Name     string
}

type VendingBoardDisappear struct {
	OwnerAID uint32
}

type VendingItem struct {
	Price      uint32
	Amount     uint16
	Index      uint16
	Type       uint8
	ItemID     uint16
	Identified bool
	Damaged    bool
	Refine     uint8
	Cards      [4]uint16
}

type VendingItemList struct {
	OwnerAID uint32
	Items    []VendingItem
	Own      bool
}

type VendingPurchaseResult struct {
	Index  uint16
	Amount uint16
	Result uint8
}

type VendingSoldItem struct {
	Index  uint16
	Amount uint16
}

type VendingOpenItem struct {
	Index  uint16
	Amount uint16
	Price  uint32
}

type VendingPurchaseItem struct {
	Index  uint16
	Amount uint16
}

func ParseVendingOpenRequest(packet Packet) (VendingOpenRequest, bool, error) {
	if packet.ID != PacketZCOpenStore {
		return VendingOpenRequest{}, false, nil
	}
	if len(packet.Data) < 4 {
		return VendingOpenRequest{}, false, fmt.Errorf("ZC_OPENSTORE too short: %d", len(packet.Data))
	}
	return VendingOpenRequest{MaxItems: binary.LittleEndian.Uint16(packet.Data[2:4])}, true, nil
}

func ParseVendingBoard(packet Packet) (VendingBoard, bool, error) {
	if packet.ID != PacketZCStoreEntry {
		return VendingBoard{}, false, nil
	}
	if len(packet.Data) < 86 {
		return VendingBoard{}, false, fmt.Errorf("ZC_STORE_ENTRY too short: %d", len(packet.Data))
	}
	return VendingBoard{
		OwnerAID: binary.LittleEndian.Uint32(packet.Data[2:6]),
		Name:     fixedPacketString(packet.Data[6:86]),
	}, true, nil
}

func ParseVendingBoardDisappear(packet Packet) (VendingBoardDisappear, bool, error) {
	if packet.ID != PacketZCDisappearEntry {
		return VendingBoardDisappear{}, false, nil
	}
	if len(packet.Data) < 6 {
		return VendingBoardDisappear{}, false, fmt.Errorf("ZC_DISAPPEAR_ENTRY too short: %d", len(packet.Data))
	}
	return VendingBoardDisappear{OwnerAID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
}

func ParseVendingItemList(packet Packet) (VendingItemList, bool, error) {
	switch packet.ID {
	case PacketZCPCPurchaseItemListFromMC:
		if len(packet.Data) < 8 {
			return VendingItemList{}, false, fmt.Errorf("ZC_PC_PURCHASE_ITEMLIST_FROMMC too short: %d", len(packet.Data))
		}
		if (len(packet.Data)-8)%vendingItemPacketSize != 0 {
			return VendingItemList{}, false, fmt.Errorf("ZC_PC_PURCHASE_ITEMLIST_FROMMC invalid length: %d", len(packet.Data))
		}
		return VendingItemList{
			OwnerAID: binary.LittleEndian.Uint32(packet.Data[4:8]),
			Items:    parseVendingItems(packet.Data[8:]),
		}, true, nil
	case PacketZCPCPurchaseMyItemList:
		if len(packet.Data) < 8 {
			return VendingItemList{}, false, fmt.Errorf("ZC_PC_PURCHASE_MYITEMLIST too short: %d", len(packet.Data))
		}
		if (len(packet.Data)-8)%vendingItemPacketSize != 0 {
			return VendingItemList{}, false, fmt.Errorf("ZC_PC_PURCHASE_MYITEMLIST invalid length: %d", len(packet.Data))
		}
		return VendingItemList{
			OwnerAID: binary.LittleEndian.Uint32(packet.Data[4:8]),
			Items:    parseOwnVendingItems(packet.Data[8:]),
			Own:      true,
		}, true, nil
	default:
		return VendingItemList{}, false, nil
	}
}

func parseVendingItems(data []byte) []VendingItem {
	items := make([]VendingItem, 0, len(data)/vendingItemPacketSize)
	for offset := 0; offset+vendingItemPacketSize <= len(data); offset += vendingItemPacketSize {
		items = append(items, VendingItem{
			Price:      binary.LittleEndian.Uint32(data[offset : offset+4]),
			Amount:     binary.LittleEndian.Uint16(data[offset+4 : offset+6]),
			Index:      binary.LittleEndian.Uint16(data[offset+6 : offset+8]),
			Type:       data[offset+8],
			ItemID:     binary.LittleEndian.Uint16(data[offset+9 : offset+11]),
			Identified: data[offset+11] != 0,
			Damaged:    data[offset+12] != 0,
			Refine:     data[offset+13],
			Cards: [4]uint16{
				binary.LittleEndian.Uint16(data[offset+14 : offset+16]),
				binary.LittleEndian.Uint16(data[offset+16 : offset+18]),
				binary.LittleEndian.Uint16(data[offset+18 : offset+20]),
				binary.LittleEndian.Uint16(data[offset+20 : offset+22]),
			},
		})
	}
	return items
}

func parseOwnVendingItems(data []byte) []VendingItem {
	items := make([]VendingItem, 0, len(data)/vendingItemPacketSize)
	for offset := 0; offset+vendingItemPacketSize <= len(data); offset += vendingItemPacketSize {
		items = append(items, VendingItem{
			Price:      binary.LittleEndian.Uint32(data[offset : offset+4]),
			Index:      binary.LittleEndian.Uint16(data[offset+4 : offset+6]),
			Amount:     binary.LittleEndian.Uint16(data[offset+6 : offset+8]),
			Type:       data[offset+8],
			ItemID:     binary.LittleEndian.Uint16(data[offset+9 : offset+11]),
			Identified: data[offset+11] != 0,
			Damaged:    data[offset+12] != 0,
			Refine:     data[offset+13],
			Cards: [4]uint16{
				binary.LittleEndian.Uint16(data[offset+14 : offset+16]),
				binary.LittleEndian.Uint16(data[offset+16 : offset+18]),
				binary.LittleEndian.Uint16(data[offset+18 : offset+20]),
				binary.LittleEndian.Uint16(data[offset+20 : offset+22]),
			},
		})
	}
	return items
}

func ParseVendingPurchaseResult(packet Packet) (VendingPurchaseResult, bool, error) {
	if packet.ID != PacketZCPCPurchaseResultFromMC {
		return VendingPurchaseResult{}, false, nil
	}
	if len(packet.Data) < 7 {
		return VendingPurchaseResult{}, false, fmt.Errorf("ZC_PC_PURCHASE_RESULT_FROMMC too short: %d", len(packet.Data))
	}
	return VendingPurchaseResult{
		Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
		Amount: binary.LittleEndian.Uint16(packet.Data[4:6]),
		Result: packet.Data[6],
	}, true, nil
}

func ParseVendingSoldItem(packet Packet) (VendingSoldItem, bool, error) {
	if packet.ID != PacketZCDeleteItemFromMCStore {
		return VendingSoldItem{}, false, nil
	}
	if len(packet.Data) < 6 {
		return VendingSoldItem{}, false, fmt.Errorf("ZC_DELETEITEM_FROM_MCSTORE too short: %d", len(packet.Data))
	}
	return VendingSoldItem{
		Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
		Amount: binary.LittleEndian.Uint16(packet.Data[4:6]),
	}, true, nil
}

func BuildCloseVendingStorePacket() []byte {
	var w Writer
	w.Uint16(PacketCZReqCloseStore)
	return w.Bytes()
}

func BuildOpenVendingStorePacket(name string, items []VendingOpenItem) []byte {
	size := 2 + 2 + vendingStoreNamePacketSize + 1 + len(items)*8
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqOpenStore2)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	copyCString(packet[4:4+vendingStoreNamePacketSize], name)
	packet[4+vendingStoreNamePacketSize] = 1
	offset := 4 + vendingStoreNamePacketSize + 1
	for _, item := range items {
		binary.LittleEndian.PutUint16(packet[offset:offset+2], item.Index)
		binary.LittleEndian.PutUint16(packet[offset+2:offset+4], item.Amount)
		binary.LittleEndian.PutUint32(packet[offset+4:offset+8], item.Price)
		offset += 8
	}
	return packet
}

func BuildCancelVendingStoreOpenPacket() []byte {
	size := 2 + 2 + vendingStoreNamePacketSize + 1
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqOpenStore2)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	return packet
}

func BuildVendingListRequestPacket(ownerAID uint32) []byte {
	var w Writer
	w.Uint16(PacketCZReqBuyFromMC)
	w.Uint32(ownerAID)
	return w.Bytes()
}

func BuildVendingPurchasePacket(ownerAID uint32, items []VendingPurchaseItem) []byte {
	size := 2 + 2 + 4 + len(items)*4
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZPCPurchaseItemListFromMC)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	binary.LittleEndian.PutUint32(packet[4:8], ownerAID)
	offset := 8
	for _, item := range items {
		binary.LittleEndian.PutUint16(packet[offset:offset+2], item.Amount)
		binary.LittleEndian.PutUint16(packet[offset+2:offset+4], item.Index)
		offset += 4
	}
	return packet
}

func copyCString(dst []byte, value string) {
	if len(dst) == 0 {
		return
	}
	n := copy(dst, []byte(value))
	if n >= len(dst) {
		dst[len(dst)-1] = 0
	}
}

func (c *Client) SendCloseVendingStore() error {
	packet := BuildCloseVendingStorePacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_CLOSESTORE opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_CLOSESTORE failed opcode=0x%04X client_date=%d: %v", ID(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendOpenVendingStore(name string, items []VendingOpenItem) error {
	packet := BuildOpenVendingStorePacket(name, items)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_OPENSTORE2 opcode=0x%04X items=%d client_date=%d", ID(packet), len(items), c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_OPENSTORE2 failed opcode=0x%04X len=%d items=%d client_date=%d: %v", ID(packet), len(packet), len(items), c.clientDate, err)
	}
	return err
}

func (c *Client) SendCancelVendingStoreOpen() error {
	packet := BuildCancelVendingStoreOpenPacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_OPENSTORE2 cancel opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_OPENSTORE2 cancel failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendVendingListRequest(ownerAID uint32) error {
	packet := BuildVendingListRequestPacket(ownerAID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_BUY_FROMMC opcode=0x%04X owner=%d client_date=%d", ID(packet), ownerAID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_BUY_FROMMC failed opcode=0x%04X owner=%d client_date=%d: %v", ID(packet), ownerAID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendVendingPurchase(ownerAID uint32, items []VendingPurchaseItem) error {
	packet := BuildVendingPurchasePacket(ownerAID, items)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_PC_PURCHASE_ITEMLIST_FROMMC opcode=0x%04X owner=%d count=%d client_date=%d", ID(packet), ownerAID, len(items), c.clientDate)
	} else {
		glog.Warnf("send CZ_PC_PURCHASE_ITEMLIST_FROMMC failed opcode=0x%04X owner=%d count=%d client_date=%d: %v", ID(packet), ownerAID, len(items), c.clientDate, err)
	}
	return err
}
