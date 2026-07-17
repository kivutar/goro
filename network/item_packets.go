package network

import (
	"encoding/binary"
	"fmt"
	"github.com/kivutar/goro/glog"
)

const PacketCZItemPickup uint16 = 0x009F

const (
	PacketCZACKSelectDealType      uint16 = 0x00C5
	PacketCZItemThrow              uint16 = 0x00A2
	PacketCZReqItemIdentify        uint16 = 0x0178
	PacketCZReqItemCompositionList uint16 = 0x017A
	PacketCZReqItemComposition     uint16 = 0x017C
	PacketCZReqMakingArrow         uint16 = 0x01AE
	PacketCZUseItem2               uint16 = 0x0439
	PacketCZUseItemLegacy          uint16 = 0x00A7
	PacketCZReqWearEquip           uint16 = 0x00A9
	PacketCZReqTakeoffEquip        uint16 = 0x00AB
	PacketCZPCPurchaseItemList     uint16 = 0x00C8
	PacketCZPCSellItemList         uint16 = 0x00C9
	PacketCZMoveToStorage          uint16 = 0x0094
	PacketCZMoveFromStorage        uint16 = 0x00F7
	PacketCZCloseStorage           uint16 = 0x0193
	PacketCZMoveToCart             uint16 = 0x0126
	PacketCZMoveFromCart           uint16 = 0x0127
	PacketCZMoveStorageToCart      uint16 = 0x0128
	PacketCZMoveCartToStorage      uint16 = 0x0129
)

type itemPickupPacketLayout struct {
	date   int
	opcode uint16
	length int
	offset int
}

var itemPickupPacketLayouts = []itemPickupPacketLayout{
	// Keep this table aligned with rAthena's clif_packetdb.hpp and reference client's
	// PacketVersions.js. For 20080910 the last active main-client remap is the
	// 20070212 shuffled 0x00F5 packet.
	{date: 20101124, opcode: 0x0362, length: 6, offset: 2},
	{date: 20070212, opcode: 0x00F5, length: 8, offset: 4},
	{date: 20070108, opcode: 0x00F5, length: 11, offset: 7},
	{date: 20050719, opcode: 0x00F5, length: 13, offset: 9},
	{date: 20050718, opcode: 0x00F5, length: 7, offset: 3},
	{date: 20050628, opcode: 0x00F5, length: 13, offset: 9},
	{date: 20050509, opcode: 0x00F5, length: 8, offset: 4},
	{date: 20050110, opcode: 0x00F5, length: 9, offset: 5},
	{date: 20041129, opcode: 0x00A2, length: 7, offset: 3},
	{date: 20041025, opcode: 0x0113, length: 9, offset: 5},
	{date: 20041005, opcode: 0x0113, length: 10, offset: 6},
	{date: 20040920, opcode: 0x0113, length: 14, offset: 10},
	{date: 20040906, opcode: 0x0113, length: 11, offset: 7},
	{date: 20040809, opcode: 0x0094, length: 13, offset: 9},
	{date: 20040726, opcode: 0x0094, length: 10, offset: 6},
	{date: 20040713, opcode: 0x009F, length: 10, offset: 6},
}

type useItemPacketLayout struct {
	date         int
	opcode       uint16
	length       int
	indexOffset  int
	targetOffset int
}

var useItemPacketLayouts = []useItemPacketLayout{
	// 2008-09-10 introduced the compact CZ_USE_ITEM2 packet. reference client maps
	// USE_ITEM to 0x0439 for this profile, and rAthena accepts it as the
	// canonical 8-byte item-use packet for the server version we run.
	{date: 20080910, opcode: PacketCZUseItem2, length: 8, indexOffset: 2, targetOffset: 4},
	{date: 20070212, opcode: 0x009F, length: 14, indexOffset: 4, targetOffset: 10},
	{date: 20050719, opcode: 0x009F, length: 19, indexOffset: 9, targetOffset: 15},
	{date: 20050718, opcode: 0x009F, length: 12, indexOffset: 3, targetOffset: 8},
	{date: 20050509, opcode: 0x009F, length: 14, indexOffset: 4, targetOffset: 10},
	{date: 20050110, opcode: 0x009F, length: 17, indexOffset: 5, targetOffset: 13},
	{date: 20041129, opcode: 0x0190, length: 15, indexOffset: 3, targetOffset: 11},
}

type storageMovePacketLayout struct {
	date         int
	opcode       uint16
	length       int
	indexOffset  int
	amountOffset int
}

type itemDropPacketLayout struct {
	date         int
	opcode       uint16
	length       int
	indexOffset  int
	amountOffset int
}

var itemDropPacketLayouts = []itemDropPacketLayout{
	{date: 20101124, opcode: 0x0363, length: 6, indexOffset: 2, amountOffset: 4},
	// Our rAthena is built as 2008-09-10 pre-renewal. That follows the main
	// 2007-02-12 shuffle, not the 2008-08-27 renewal 17-byte layout.
	{date: 20070212, opcode: 0x0116, length: 10, indexOffset: 5, amountOffset: 8},
}

var moveToStoragePacketLayouts = []storageMovePacketLayout{
	// 2008-09-10 keeps the 2007-02-12 shuffled Kafra packet offsets in
	// rAthena's packet DB. The unshuffled 0x00F3 packet is not accepted by
	// that server profile.
	{date: 20070212, opcode: PacketCZMoveToStorage, length: 14, indexOffset: 7, amountOffset: 10},
	{date: 20050509, opcode: PacketCZMoveToStorage, length: 14, indexOffset: 7, amountOffset: 10},
}

var moveFromStoragePacketLayouts = []storageMovePacketLayout{
	{date: 20070212, opcode: PacketCZMoveFromStorage, length: 22, indexOffset: 14, amountOffset: 18},
	{date: 20050509, opcode: PacketCZMoveFromStorage, length: 22, indexOffset: 14, amountOffset: 18},
}

type FloorItemEntry struct {
	ID         uint32
	ItemID     uint16
	Identified bool
	X          int
	Y          int
	SubX       uint8
	SubY       uint8
	Amount     uint16
	Falling    bool
}

type MakingArrowList struct {
	ItemIDs []uint16
}

type FloorItemDisappear struct {
	ID uint32
}

type ItemPickupAck struct {
	Index      uint16
	Amount     uint16
	ItemID     uint16
	Location   uint16
	Identified bool
	Type       uint8
	Damaged    bool
	Refine     uint8
	Result     uint8
}

type UseItemAck struct {
	Index  uint16
	Amount uint16
	ItemID uint16
	AID    uint32
	Result uint8
}

type InventoryItem struct {
	Index      uint16
	ItemID     uint16
	Type       uint8
	Location   uint16
	Identified bool
	Amount     uint16
	Equip      bool
	Equipped   bool
	Damaged    bool
	Refine     uint8
	Cards      [4]uint16
}

type InventoryItemDelete struct {
	Index  uint16
	Amount uint16
	Reason uint16
}

type ItemIdentifyList struct {
	Indexes []uint16
}

type ItemIdentifyAck struct {
	Index   uint16
	Success bool
}

type ItemCompositionList struct {
	Indexes []uint16
}

type ItemCompositionAck struct {
	EquipIndex uint16
	CardIndex  uint16
	Success    bool
}

type InventoryEquipAck struct {
	Index    uint16
	Location uint16
	Success  bool
	Unequip  bool
}

type EquippedArrow struct {
	Index uint16
}

type StorageAmount struct {
	Amount    uint16
	MaxAmount uint16
}

type CartAmount struct {
	Amount    uint16
	MaxAmount uint16
	Weight    uint32
	MaxWeight uint32
}

type StorageItemRemoved struct {
	Index  uint16
	Amount uint32
}

type CartItemRemoved struct {
	Index  uint16
	Amount uint32
}

type CartAddAck struct {
	Result uint8
}

type ShopDealSelection struct {
	NPCID uint32
}

type ShopBuyItem struct {
	ItemID        uint16
	Type          uint8
	Price         uint32
	DiscountPrice uint32
}

type ShopSellItem struct {
	Index           uint16
	Price           uint32
	OverchargePrice uint32
}

type ShopResult struct {
	Sell   bool
	Result uint8
}

type SellRequestItem struct {
	Index  uint16
	Amount uint16
}

type BuyRequestItem struct {
	ItemID uint16
	Amount uint16
}

func ParseFloorItemEntry(packet Packet) (FloorItemEntry, bool, error) {
	switch packet.ID {
	case 0x009D:
		if len(packet.Data) < 17 {
			return FloorItemEntry{}, false, fmt.Errorf("ZC_ITEM_ENTRY too short: %d", len(packet.Data))
		}
		return FloorItemEntry{
			ID:         binary.LittleEndian.Uint32(packet.Data[2:6]),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[6:8]),
			Identified: packet.Data[8] != 0,
			X:          int(binary.LittleEndian.Uint16(packet.Data[9:11])),
			Y:          int(binary.LittleEndian.Uint16(packet.Data[11:13])),
			Amount:     binary.LittleEndian.Uint16(packet.Data[13:15]),
			SubX:       packet.Data[15],
			SubY:       packet.Data[16],
		}, true, nil
	case 0x009E:
		if len(packet.Data) < 17 {
			return FloorItemEntry{}, false, fmt.Errorf("ZC_ITEM_FALL_ENTRY too short: %d", len(packet.Data))
		}
		return FloorItemEntry{
			ID:         binary.LittleEndian.Uint32(packet.Data[2:6]),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[6:8]),
			Identified: packet.Data[8] != 0,
			X:          int(binary.LittleEndian.Uint16(packet.Data[9:11])),
			Y:          int(binary.LittleEndian.Uint16(packet.Data[11:13])),
			SubX:       packet.Data[13],
			SubY:       packet.Data[14],
			Amount:     binary.LittleEndian.Uint16(packet.Data[15:17]),
			Falling:    true,
		}, true, nil
	default:
		return FloorItemEntry{}, false, nil
	}
}

func ParseFloorItemDisappear(packet Packet) (FloorItemDisappear, bool, error) {
	if packet.ID != 0x00A1 {
		return FloorItemDisappear{}, false, nil
	}
	if len(packet.Data) < 6 {
		return FloorItemDisappear{}, false, fmt.Errorf("ZC_ITEM_DISAPPEAR too short: %d", len(packet.Data))
	}
	return FloorItemDisappear{ID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
}

func ParseItemPickupAck(packet Packet) (ItemPickupAck, bool, error) {
	switch packet.ID {
	case 0x00A0, 0x029A, 0x02D4:
		return parseItemPickupAckLocation16(packet)
	case 0x0990, 0x0A0C, 0x0A37:
		return parseItemPickupAckLocation32(packet)
	default:
		return ItemPickupAck{}, false, nil
	}
}

func ParseUseItemAck(packet Packet) (UseItemAck, bool, error) {
	switch packet.ID {
	case 0x00A8:
		if len(packet.Data) < 7 {
			return UseItemAck{}, false, fmt.Errorf("ZC_USE_ITEM_ACK too short: %d", len(packet.Data))
		}
		return UseItemAck{
			Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount: binary.LittleEndian.Uint16(packet.Data[4:6]),
			Result: packet.Data[6],
		}, true, nil
	case 0x01C8:
		switch {
		case len(packet.Data) >= 15:
			return UseItemAck{
				Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
				ItemID: uint16(binary.LittleEndian.Uint32(packet.Data[4:8])),
				AID:    binary.LittleEndian.Uint32(packet.Data[8:12]),
				Amount: binary.LittleEndian.Uint16(packet.Data[12:14]),
				Result: packet.Data[14],
			}, true, nil
		case len(packet.Data) >= 13:
			return UseItemAck{
				Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
				ItemID: binary.LittleEndian.Uint16(packet.Data[4:6]),
				AID:    binary.LittleEndian.Uint32(packet.Data[6:10]),
				Amount: binary.LittleEndian.Uint16(packet.Data[10:12]),
				Result: packet.Data[12],
			}, true, nil
		default:
			return UseItemAck{}, false, fmt.Errorf("ZC_USE_ITEM_ACK2 too short: %d", len(packet.Data))
		}
	default:
		return UseItemAck{}, false, nil
	}
}

func parseItemPickupAckLocation16(packet Packet) (ItemPickupAck, bool, error) {
	if len(packet.Data) < 23 {
		return ItemPickupAck{}, false, fmt.Errorf("ZC_ITEM_PICKUP_ACK 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	return ItemPickupAck{
		Index:      binary.LittleEndian.Uint16(packet.Data[2:4]),
		Amount:     binary.LittleEndian.Uint16(packet.Data[4:6]),
		ItemID:     binary.LittleEndian.Uint16(packet.Data[6:8]),
		Identified: packet.Data[8] != 0,
		Damaged:    packet.Data[9] != 0,
		Refine:     packet.Data[10],
		Location:   binary.LittleEndian.Uint16(packet.Data[19:21]),
		Type:       packet.Data[21],
		Result:     packet.Data[22],
	}, true, nil
}

func parseItemPickupAckLocation32(packet Packet) (ItemPickupAck, bool, error) {
	if len(packet.Data) < 25 {
		return ItemPickupAck{}, false, fmt.Errorf("ZC_ITEM_PICKUP_ACK 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	return ItemPickupAck{
		Index:      binary.LittleEndian.Uint16(packet.Data[2:4]),
		Amount:     binary.LittleEndian.Uint16(packet.Data[4:6]),
		ItemID:     binary.LittleEndian.Uint16(packet.Data[6:8]),
		Identified: packet.Data[8] != 0,
		Damaged:    packet.Data[9] != 0,
		Refine:     packet.Data[10],
		Location:   uint16(binary.LittleEndian.Uint32(packet.Data[19:23])),
		Type:       packet.Data[23],
		Result:     packet.Data[24],
	}, true, nil
}

func ParseInventoryItemList(packet Packet) ([]InventoryItem, bool, error) {
	switch packet.ID {
	case 0x00A3:
		return parseNormalInventoryItems(packet, 10)
	case 0x01EE:
		return parseNormalInventoryItems(packet, 18)
	case 0x02E8:
		return parseNormalInventoryItems(packet, 22)
	case 0x0991:
		return parseNormalInventoryItems4(packet)
	case 0x00A4:
		return parseEquipInventoryItems(packet, 20)
	case 0x0295:
		return parseEquipInventoryItems(packet, 24)
	case 0x02D0:
		return parseEquipInventoryItems(packet, 26)
	case 0x0992:
		return parseEquipInventoryItems4(packet, 31)
	case 0x0A0D:
		return parseEquipInventoryItems4(packet, 57)
	default:
		return nil, false, nil
	}
}

func ParseStorageItemList(packet Packet) ([]InventoryItem, bool, error) {
	switch packet.ID {
	case 0x00A5:
		return parseNormalInventoryItems(packet, 10)
	case 0x02EA:
		return parseNormalInventoryItems(packet, 22)
	case 0x00A6:
		return parseEquipInventoryItems(packet, 20)
	case 0x02D1:
		return parseEquipInventoryItems(packet, 26)
	default:
		return nil, false, nil
	}
}

func ParseCartItemList(packet Packet) ([]InventoryItem, bool, error) {
	switch packet.ID {
	case 0x0123:
		return parseNormalInventoryItems(packet, 10)
	case 0x02E9:
		return parseNormalInventoryItems(packet, 22)
	case 0x0993:
		return parseNormalInventoryItems4(packet)
	case 0x0122:
		return parseEquipInventoryItems(packet, 20)
	case 0x02D2:
		return parseEquipInventoryItems(packet, 26)
	case 0x0994:
		return parseEquipInventoryItems4(packet, 31)
	default:
		return nil, false, nil
	}
}

func parseNormalInventoryItems(packet Packet, entrySize int) ([]InventoryItem, bool, error) {
	if len(packet.Data) < 4 {
		return nil, false, fmt.Errorf("ZC_NORMAL_ITEMLIST 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, false, fmt.Errorf("ZC_NORMAL_ITEMLIST 0x%04X invalid length: %d", packet.ID, len(packet.Data))
	}
	items := make([]InventoryItem, 0, (len(packet.Data)-4)/entrySize)
	for offset := 4; offset+entrySize <= len(packet.Data); offset += entrySize {
		var cards [4]uint16
		if entrySize >= 18 {
			cards = readItemCards(packet.Data, offset+10)
		}
		items = append(items, InventoryItem{
			Index:      binary.LittleEndian.Uint16(packet.Data[offset : offset+2]),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[offset+2 : offset+4]),
			Type:       packet.Data[offset+4],
			Identified: packet.Data[offset+5] != 0,
			Amount:     binary.LittleEndian.Uint16(packet.Data[offset+6 : offset+8]),
			Location:   binary.LittleEndian.Uint16(packet.Data[offset+8 : offset+10]),
			Cards:      cards,
		})
	}
	return items, true, nil
}

func parseNormalInventoryItems4(packet Packet) ([]InventoryItem, bool, error) {
	const entrySize = 24
	if len(packet.Data) < 4 {
		return nil, false, fmt.Errorf("ZC_NORMAL_ITEMLIST4 too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, false, fmt.Errorf("ZC_NORMAL_ITEMLIST4 invalid length: %d", len(packet.Data))
	}
	items := make([]InventoryItem, 0, (len(packet.Data)-4)/entrySize)
	for offset := 4; offset+entrySize <= len(packet.Data); offset += entrySize {
		flag := packet.Data[offset+23]
		items = append(items, InventoryItem{
			Index:      binary.LittleEndian.Uint16(packet.Data[offset : offset+2]),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[offset+2 : offset+4]),
			Type:       packet.Data[offset+4],
			Identified: flag&1 != 0,
			Amount:     binary.LittleEndian.Uint16(packet.Data[offset+5 : offset+7]),
			Location:   uint16(binary.LittleEndian.Uint32(packet.Data[offset+7 : offset+11])),
			Cards:      readItemCards(packet.Data, offset+11),
		})
	}
	return items, true, nil
}

func parseEquipInventoryItems(packet Packet, entrySize int) ([]InventoryItem, bool, error) {
	if len(packet.Data) < 4 {
		return nil, false, fmt.Errorf("ZC_EQUIPMENT_ITEMLIST 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, false, fmt.Errorf("ZC_EQUIPMENT_ITEMLIST 0x%04X invalid length: %d", packet.ID, len(packet.Data))
	}
	items := make([]InventoryItem, 0, (len(packet.Data)-4)/entrySize)
	for offset := 4; offset+entrySize <= len(packet.Data); offset += entrySize {
		wearState := binary.LittleEndian.Uint16(packet.Data[offset+8 : offset+10])
		items = append(items, InventoryItem{
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
	return items, true, nil
}

func parseEquipInventoryItems4(packet Packet, entrySize int) ([]InventoryItem, bool, error) {
	if len(packet.Data) < 4 {
		return nil, false, fmt.Errorf("ZC_EQUIPMENT_ITEMLIST4 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	if (len(packet.Data)-4)%entrySize != 0 {
		return nil, false, fmt.Errorf("ZC_EQUIPMENT_ITEMLIST4 0x%04X invalid length: %d", packet.ID, len(packet.Data))
	}
	items := make([]InventoryItem, 0, (len(packet.Data)-4)/entrySize)
	for offset := 4; offset+entrySize <= len(packet.Data); offset += entrySize {
		flag := packet.Data[offset+entrySize-1]
		items = append(items, InventoryItem{
			Index:      binary.LittleEndian.Uint16(packet.Data[offset : offset+2]),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[offset+2 : offset+4]),
			Type:       packet.Data[offset+4],
			Identified: flag&1 != 0,
			Location:   uint16(binary.LittleEndian.Uint32(packet.Data[offset+5 : offset+9])),
			Amount:     1,
			Equip:      true,
			Equipped:   binary.LittleEndian.Uint32(packet.Data[offset+9:offset+13]) != 0,
			Damaged:    flag&2 != 0,
			Refine:     packet.Data[offset+13],
			Cards:      readItemCards(packet.Data, offset+14),
		})
	}
	return items, true, nil
}

func readItemCards(data []byte, offset int) [4]uint16 {
	var cards [4]uint16
	for i := range cards {
		start := offset + i*2
		if start+2 > len(data) {
			break
		}
		cards[i] = binary.LittleEndian.Uint16(data[start : start+2])
	}
	return cards
}

func ParseInventoryItemDelete(packet Packet) (InventoryItemDelete, bool, error) {
	switch packet.ID {
	case 0x00AF:
		if len(packet.Data) < 6 {
			return InventoryItemDelete{}, false, fmt.Errorf("ZC_ITEM_THROW_ACK too short: %d", len(packet.Data))
		}
		return InventoryItemDelete{
			Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount: binary.LittleEndian.Uint16(packet.Data[4:6]),
		}, true, nil
	case 0x07FA:
		if len(packet.Data) < 8 {
			return InventoryItemDelete{}, false, fmt.Errorf("ZC_DELETE_ITEM_FROM_BODY too short: %d", len(packet.Data))
		}
		return InventoryItemDelete{
			Reason: binary.LittleEndian.Uint16(packet.Data[2:4]),
			Index:  binary.LittleEndian.Uint16(packet.Data[4:6]),
			Amount: binary.LittleEndian.Uint16(packet.Data[6:8]),
		}, true, nil
	default:
		return InventoryItemDelete{}, false, nil
	}
}

func ParseItemIdentifyList(packet Packet) (ItemIdentifyList, bool, error) {
	if packet.ID != 0x0177 {
		return ItemIdentifyList{}, false, nil
	}
	if len(packet.Data) < 4 {
		return ItemIdentifyList{}, false, fmt.Errorf("ZC_ITEMIDENTIFY_LIST too short: %d", len(packet.Data))
	}
	size := int(binary.LittleEndian.Uint16(packet.Data[2:4]))
	if size > len(packet.Data) {
		return ItemIdentifyList{}, false, fmt.Errorf("ZC_ITEMIDENTIFY_LIST invalid length: header=%d data=%d", size, len(packet.Data))
	}
	if size < 4 {
		return ItemIdentifyList{}, false, fmt.Errorf("ZC_ITEMIDENTIFY_LIST invalid length: %d", size)
	}
	if (size-4)%2 != 0 {
		return ItemIdentifyList{}, false, fmt.Errorf("ZC_ITEMIDENTIFY_LIST odd item payload: %d", size)
	}
	indexes := make([]uint16, 0, (size-4)/2)
	for offset := 4; offset+2 <= size; offset += 2 {
		indexes = append(indexes, binary.LittleEndian.Uint16(packet.Data[offset:offset+2]))
	}
	return ItemIdentifyList{Indexes: indexes}, true, nil
}

func ParseItemIdentifyAck(packet Packet) (ItemIdentifyAck, bool, error) {
	if packet.ID != 0x0179 {
		return ItemIdentifyAck{}, false, nil
	}
	if len(packet.Data) < 5 {
		return ItemIdentifyAck{}, false, fmt.Errorf("ZC_ACK_ITEMIDENTIFY too short: %d", len(packet.Data))
	}
	return ItemIdentifyAck{
		Index:   binary.LittleEndian.Uint16(packet.Data[2:4]),
		Success: packet.Data[4] == 0,
	}, true, nil
}

func ParseMakingArrowList(packet Packet) (MakingArrowList, bool, error) {
	if packet.ID != 0x01AD {
		return MakingArrowList{}, false, nil
	}
	if len(packet.Data) < 4 {
		return MakingArrowList{}, false, fmt.Errorf("ZC_MAKINGARROW_LIST too short: %d", len(packet.Data))
	}
	size := int(binary.LittleEndian.Uint16(packet.Data[2:4]))
	if size > len(packet.Data) {
		return MakingArrowList{}, false, fmt.Errorf("ZC_MAKINGARROW_LIST invalid length: header=%d data=%d", size, len(packet.Data))
	}
	if size < 4 {
		return MakingArrowList{}, false, fmt.Errorf("ZC_MAKINGARROW_LIST invalid length: %d", size)
	}
	if (size-4)%2 != 0 {
		return MakingArrowList{}, false, fmt.Errorf("ZC_MAKINGARROW_LIST odd item payload: %d", size)
	}
	itemIDs := make([]uint16, 0, (size-4)/2)
	for offset := 4; offset+2 <= size; offset += 2 {
		itemIDs = append(itemIDs, binary.LittleEndian.Uint16(packet.Data[offset:offset+2]))
	}
	return MakingArrowList{ItemIDs: itemIDs}, true, nil
}

func ParseItemCompositionList(packet Packet) (ItemCompositionList, bool, error) {
	if packet.ID != 0x017B {
		return ItemCompositionList{}, false, nil
	}
	if len(packet.Data) < 4 {
		return ItemCompositionList{}, false, fmt.Errorf("ZC_ITEMCOMPOSITION_LIST too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%2 != 0 {
		return ItemCompositionList{}, false, fmt.Errorf("ZC_ITEMCOMPOSITION_LIST invalid length: %d", len(packet.Data))
	}
	indexes := make([]uint16, 0, (len(packet.Data)-4)/2)
	for offset := 4; offset+2 <= len(packet.Data); offset += 2 {
		indexes = append(indexes, binary.LittleEndian.Uint16(packet.Data[offset:offset+2]))
	}
	return ItemCompositionList{Indexes: indexes}, true, nil
}

func ParseItemCompositionAck(packet Packet) (ItemCompositionAck, bool, error) {
	if packet.ID != 0x017D {
		return ItemCompositionAck{}, false, nil
	}
	if len(packet.Data) < 7 {
		return ItemCompositionAck{}, false, fmt.Errorf("ZC_ACK_ITEMCOMPOSITION too short: %d", len(packet.Data))
	}
	return ItemCompositionAck{
		EquipIndex: binary.LittleEndian.Uint16(packet.Data[2:4]),
		CardIndex:  binary.LittleEndian.Uint16(packet.Data[4:6]),
		Success:    packet.Data[6] == 0,
	}, true, nil
}

func ParseStorageAmount(packet Packet) (StorageAmount, bool, error) {
	if packet.ID != 0x00F2 {
		return StorageAmount{}, false, nil
	}
	if len(packet.Data) < 6 {
		return StorageAmount{}, false, fmt.Errorf("ZC_NOTIFY_STOREITEM_COUNTINFO too short: %d", len(packet.Data))
	}
	return StorageAmount{
		Amount:    binary.LittleEndian.Uint16(packet.Data[2:4]),
		MaxAmount: binary.LittleEndian.Uint16(packet.Data[4:6]),
	}, true, nil
}

func ParseCartAmount(packet Packet) (CartAmount, bool, error) {
	if packet.ID != 0x0121 {
		return CartAmount{}, false, nil
	}
	if len(packet.Data) < 14 {
		return CartAmount{}, false, fmt.Errorf("ZC_NOTIFY_CARTITEM_COUNTINFO too short: %d", len(packet.Data))
	}
	return CartAmount{
		Amount:    binary.LittleEndian.Uint16(packet.Data[2:4]),
		MaxAmount: binary.LittleEndian.Uint16(packet.Data[4:6]),
		Weight:    binary.LittleEndian.Uint32(packet.Data[6:10]),
		MaxWeight: binary.LittleEndian.Uint32(packet.Data[10:14]),
	}, true, nil
}

func ParseStorageItemAdded(packet Packet) (InventoryItem, bool, error) {
	switch packet.ID {
	case 0x00F4:
		if len(packet.Data) < 21 {
			return InventoryItem{}, false, fmt.Errorf("ZC_ADD_ITEM_TO_STORE too short: %d", len(packet.Data))
		}
		return InventoryItem{
			Index:      binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount:     uint16(minIntNetwork(int(binary.LittleEndian.Uint32(packet.Data[4:8])), int(^uint16(0)))),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[8:10]),
			Identified: packet.Data[10] != 0,
			Damaged:    packet.Data[11] != 0,
			Refine:     packet.Data[12],
		}, true, nil
	case 0x01C4:
		if len(packet.Data) < 22 {
			return InventoryItem{}, false, fmt.Errorf("ZC_ADD_ITEM_TO_STORE2 too short: %d", len(packet.Data))
		}
		return InventoryItem{
			Index:      binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount:     uint16(minIntNetwork(int(binary.LittleEndian.Uint32(packet.Data[4:8])), int(^uint16(0)))),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[8:10]),
			Type:       packet.Data[10],
			Identified: packet.Data[11] != 0,
			Damaged:    packet.Data[12] != 0,
			Refine:     packet.Data[13],
		}, true, nil
	default:
		return InventoryItem{}, false, nil
	}
}

func ParseCartItemAdded(packet Packet) (InventoryItem, bool, error) {
	switch packet.ID {
	case 0x0124:
		if len(packet.Data) < 21 {
			return InventoryItem{}, false, fmt.Errorf("ZC_ADD_ITEM_TO_CART too short: %d", len(packet.Data))
		}
		return InventoryItem{
			Index:      binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount:     uint16(minIntNetwork(int(binary.LittleEndian.Uint32(packet.Data[4:8])), int(^uint16(0)))),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[8:10]),
			Identified: packet.Data[10] != 0,
			Damaged:    packet.Data[11] != 0,
			Refine:     packet.Data[12],
		}, true, nil
	case 0x01C5:
		if len(packet.Data) < 22 {
			return InventoryItem{}, false, fmt.Errorf("ZC_ADD_ITEM_TO_CART2 too short: %d", len(packet.Data))
		}
		return InventoryItem{
			Index:      binary.LittleEndian.Uint16(packet.Data[2:4]),
			Amount:     uint16(minIntNetwork(int(binary.LittleEndian.Uint32(packet.Data[4:8])), int(^uint16(0)))),
			ItemID:     binary.LittleEndian.Uint16(packet.Data[8:10]),
			Type:       packet.Data[10],
			Identified: packet.Data[11] != 0,
			Damaged:    packet.Data[12] != 0,
			Refine:     packet.Data[13],
		}, true, nil
	default:
		return InventoryItem{}, false, nil
	}
}

func ParseStorageItemRemoved(packet Packet) (StorageItemRemoved, bool, error) {
	if packet.ID != 0x00F6 {
		return StorageItemRemoved{}, false, nil
	}
	if len(packet.Data) < 8 {
		return StorageItemRemoved{}, false, fmt.Errorf("ZC_DELETE_ITEM_FROM_STORE too short: %d", len(packet.Data))
	}
	return StorageItemRemoved{
		Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
		Amount: binary.LittleEndian.Uint32(packet.Data[4:8]),
	}, true, nil
}

func ParseCartItemRemoved(packet Packet) (CartItemRemoved, bool, error) {
	if packet.ID != 0x0125 {
		return CartItemRemoved{}, false, nil
	}
	if len(packet.Data) < 8 {
		return CartItemRemoved{}, false, fmt.Errorf("ZC_DELETE_ITEM_FROM_CART too short: %d", len(packet.Data))
	}
	return CartItemRemoved{
		Index:  binary.LittleEndian.Uint16(packet.Data[2:4]),
		Amount: binary.LittleEndian.Uint32(packet.Data[4:8]),
	}, true, nil
}

func ParseCartAddAck(packet Packet) (CartAddAck, bool, error) {
	if packet.ID != 0x012C {
		return CartAddAck{}, false, nil
	}
	if len(packet.Data) < 3 {
		return CartAddAck{}, false, fmt.Errorf("ZC_ACK_ADDITEM_TO_CART too short: %d", len(packet.Data))
	}
	return CartAddAck{Result: packet.Data[2]}, true, nil
}

func ParseStorageClosed(packet Packet) bool {
	return packet.ID == 0x00F8
}

func ParseCartClosed(packet Packet) bool {
	return packet.ID == 0x012B
}

func minIntNetwork(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ParseInventoryEquipAck(packet Packet) (InventoryEquipAck, bool, error) {
	switch packet.ID {
	case 0x00AA:
		if len(packet.Data) < 7 {
			return InventoryEquipAck{}, false, fmt.Errorf("ZC_REQ_WEAR_EQUIP_ACK too short: %d", len(packet.Data))
		}
		return InventoryEquipAck{
			Index:    binary.LittleEndian.Uint16(packet.Data[2:4]),
			Location: binary.LittleEndian.Uint16(packet.Data[4:6]),
			Success:  packet.Data[6] != 0,
		}, true, nil
	case 0x00AC:
		if len(packet.Data) < 7 {
			return InventoryEquipAck{}, false, fmt.Errorf("ZC_REQ_TAKEOFF_EQUIP_ACK too short: %d", len(packet.Data))
		}
		return InventoryEquipAck{
			Index:    binary.LittleEndian.Uint16(packet.Data[2:4]),
			Location: binary.LittleEndian.Uint16(packet.Data[4:6]),
			Success:  packet.Data[6] != 0,
			Unequip:  true,
		}, true, nil
	default:
		return InventoryEquipAck{}, false, nil
	}
}

func ParseEquippedArrow(packet Packet) (EquippedArrow, bool, error) {
	if packet.ID != 0x013C {
		return EquippedArrow{}, false, nil
	}
	if len(packet.Data) < 4 {
		return EquippedArrow{}, false, fmt.Errorf("ZC_EQUIP_ARROW too short: %d", len(packet.Data))
	}
	return EquippedArrow{
		Index: binary.LittleEndian.Uint16(packet.Data[2:4]),
	}, true, nil
}

func ParseShopDealSelection(packet Packet) (ShopDealSelection, bool, error) {
	if packet.ID != 0x00C4 {
		return ShopDealSelection{}, false, nil
	}
	if len(packet.Data) < 6 {
		return ShopDealSelection{}, false, fmt.Errorf("ZC_SELECT_DEALTYPE too short: %d", len(packet.Data))
	}
	return ShopDealSelection{NPCID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
}

func ParseShopBuyList(packet Packet) ([]ShopBuyItem, bool, error) {
	if packet.ID != 0x00C6 {
		return nil, false, nil
	}
	if len(packet.Data) < 4 {
		return nil, false, fmt.Errorf("ZC_PC_PURCHASE_ITEMLIST too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%11 != 0 {
		return nil, false, fmt.Errorf("ZC_PC_PURCHASE_ITEMLIST invalid length: %d", len(packet.Data))
	}
	items := make([]ShopBuyItem, 0, (len(packet.Data)-4)/11)
	for offset := 4; offset+11 <= len(packet.Data); offset += 11 {
		items = append(items, ShopBuyItem{
			Price:         binary.LittleEndian.Uint32(packet.Data[offset : offset+4]),
			DiscountPrice: binary.LittleEndian.Uint32(packet.Data[offset+4 : offset+8]),
			Type:          packet.Data[offset+8],
			ItemID:        binary.LittleEndian.Uint16(packet.Data[offset+9 : offset+11]),
		})
	}
	return items, true, nil
}

func ParseShopSellList(packet Packet) ([]ShopSellItem, bool, error) {
	if packet.ID != 0x00C7 {
		return nil, false, nil
	}
	if len(packet.Data) < 4 {
		return nil, false, fmt.Errorf("ZC_PC_SELL_ITEMLIST too short: %d", len(packet.Data))
	}
	if (len(packet.Data)-4)%10 != 0 {
		return nil, false, fmt.Errorf("ZC_PC_SELL_ITEMLIST invalid length: %d", len(packet.Data))
	}
	items := make([]ShopSellItem, 0, (len(packet.Data)-4)/10)
	for offset := 4; offset+10 <= len(packet.Data); offset += 10 {
		items = append(items, ShopSellItem{
			Index:           binary.LittleEndian.Uint16(packet.Data[offset : offset+2]),
			Price:           binary.LittleEndian.Uint32(packet.Data[offset+2 : offset+6]),
			OverchargePrice: binary.LittleEndian.Uint32(packet.Data[offset+6 : offset+10]),
		})
	}
	return items, true, nil
}

func ParseShopResult(packet Packet) (ShopResult, bool, error) {
	switch packet.ID {
	case 0x00CA:
		if len(packet.Data) < 3 {
			return ShopResult{}, false, fmt.Errorf("ZC_PC_PURCHASE_RESULT too short: %d", len(packet.Data))
		}
		return ShopResult{Result: packet.Data[2]}, true, nil
	case 0x00CB:
		if len(packet.Data) < 3 {
			return ShopResult{}, false, fmt.Errorf("ZC_PC_SELL_RESULT too short: %d", len(packet.Data))
		}
		return ShopResult{Sell: true, Result: packet.Data[2]}, true, nil
	default:
		return ShopResult{}, false, nil
	}
}

func BuildItemPickupPacket(gid uint32) []byte {
	var w Writer
	w.Uint16(PacketCZItemPickup)
	w.Uint32(gid)
	return w.Bytes()
}

func BuildItemPickupPacketForClientDate(gid uint32, clientDate int) []byte {
	for _, layout := range itemPickupPacketLayouts {
		if clientDate >= layout.date {
			packet := make([]byte, layout.length)
			binary.LittleEndian.PutUint16(packet[0:2], layout.opcode)
			binary.LittleEndian.PutUint32(packet[layout.offset:layout.offset+4], gid)
			return packet
		}
	}
	return BuildItemPickupPacket(gid)
}

func BuildUseInventoryItemPacketForClientDate(index uint16, targetAID uint32, clientDate int) []byte {
	for _, layout := range useItemPacketLayouts {
		if clientDate >= layout.date {
			packet := make([]byte, layout.length)
			binary.LittleEndian.PutUint16(packet[0:2], layout.opcode)
			if layout.indexOffset > 2 {
				fillLegacyPacketPadding(packet[2:layout.indexOffset])
			}
			binary.LittleEndian.PutUint16(packet[layout.indexOffset:layout.indexOffset+2], index)
			if layout.targetOffset > layout.indexOffset+2 {
				fillLegacyPacketPadding(packet[layout.indexOffset+2 : layout.targetOffset])
			}
			binary.LittleEndian.PutUint32(packet[layout.targetOffset:layout.targetOffset+4], targetAID)
			return packet
		}
	}
	packet := make([]byte, 8)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZUseItemLegacy)
	binary.LittleEndian.PutUint16(packet[2:4], index)
	binary.LittleEndian.PutUint32(packet[4:8], targetAID)
	return packet
}

func BuildDropInventoryItemPacket(index, amount uint16) []byte {
	if amount == 0 {
		amount = 1
	}
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZItemThrow)
	binary.LittleEndian.PutUint16(packet[2:4], index)
	binary.LittleEndian.PutUint16(packet[4:6], amount)
	return packet
}

func BuildDropInventoryItemPacketForClientDate(index, amount uint16, clientDate int) []byte {
	if amount == 0 {
		amount = 1
	}
	for _, layout := range itemDropPacketLayouts {
		if clientDate < layout.date {
			continue
		}
		packet := make([]byte, layout.length)
		binary.LittleEndian.PutUint16(packet[0:2], layout.opcode)
		if layout.indexOffset > 2 {
			fillLegacyPacketPadding(packet[2:layout.indexOffset])
		}
		binary.LittleEndian.PutUint16(packet[layout.indexOffset:layout.indexOffset+2], index)
		if layout.amountOffset > layout.indexOffset+2 {
			fillLegacyPacketPadding(packet[layout.indexOffset+2 : layout.amountOffset])
		}
		binary.LittleEndian.PutUint16(packet[layout.amountOffset:layout.amountOffset+2], amount)
		return packet
	}
	return BuildDropInventoryItemPacket(index, amount)
}

func BuildItemIdentifyPacket(index uint16) []byte {
	packet := make([]byte, 4)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqItemIdentify)
	binary.LittleEndian.PutUint16(packet[2:4], index)
	return packet
}

func BuildItemCompositionListPacket(cardIndex uint16) []byte {
	packet := make([]byte, 4)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqItemCompositionList)
	binary.LittleEndian.PutUint16(packet[2:4], cardIndex)
	return packet
}

func BuildItemCompositionPacket(cardIndex, equipIndex uint16) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqItemComposition)
	binary.LittleEndian.PutUint16(packet[2:4], cardIndex)
	binary.LittleEndian.PutUint16(packet[4:6], equipIndex)
	return packet
}

func BuildMakingArrowPacket(itemID uint16) []byte {
	packet := make([]byte, 4)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqMakingArrow)
	binary.LittleEndian.PutUint16(packet[2:4], itemID)
	return packet
}

func BuildWearEquipPacket(index, location uint16) []byte {
	packet := make([]byte, 6)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqWearEquip)
	binary.LittleEndian.PutUint16(packet[2:4], index)
	binary.LittleEndian.PutUint16(packet[4:6], location)
	return packet
}

func BuildTakeoffEquipPacket(index uint16) []byte {
	packet := make([]byte, 4)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZReqTakeoffEquip)
	binary.LittleEndian.PutUint16(packet[2:4], index)
	return packet
}

func BuildMoveToStoragePacketForClientDate(index uint16, amount uint32, clientDate int) []byte {
	return buildStorageMovePacket(index, amount, clientDate, moveToStoragePacketLayouts, 0x00F3)
}

func BuildMoveFromStoragePacketForClientDate(index uint16, amount uint32, clientDate int) []byte {
	return buildStorageMovePacket(index, amount, clientDate, moveFromStoragePacketLayouts, 0x00F5)
}

func buildStorageMovePacket(index uint16, amount uint32, clientDate int, layouts []storageMovePacketLayout, fallbackOpcode uint16) []byte {
	if amount == 0 {
		amount = 1
	}
	for _, layout := range layouts {
		if clientDate < layout.date {
			continue
		}
		packet := make([]byte, layout.length)
		binary.LittleEndian.PutUint16(packet[0:2], layout.opcode)
		if layout.indexOffset > 2 {
			fillLegacyPacketPadding(packet[2:layout.indexOffset])
		}
		binary.LittleEndian.PutUint16(packet[layout.indexOffset:layout.indexOffset+2], index)
		if layout.amountOffset > layout.indexOffset+2 {
			fillLegacyPacketPadding(packet[layout.indexOffset+2 : layout.amountOffset])
		}
		binary.LittleEndian.PutUint32(packet[layout.amountOffset:layout.amountOffset+4], amount)
		return packet
	}
	packet := make([]byte, 8)
	binary.LittleEndian.PutUint16(packet[0:2], fallbackOpcode)
	binary.LittleEndian.PutUint16(packet[2:4], index)
	binary.LittleEndian.PutUint32(packet[4:8], amount)
	return packet
}

func BuildCloseStoragePacketForClientDate(clientDate int) []byte {
	opcode := PacketCZCloseStorage
	if clientDate < 20050523 {
		opcode = 0x00F7
	}
	packet := make([]byte, 2)
	binary.LittleEndian.PutUint16(packet[0:2], opcode)
	return packet
}

func BuildMoveToCartPacket(index uint16, amount uint32) []byte {
	return buildCartMovePacket(PacketCZMoveToCart, index, amount)
}

func BuildMoveFromCartPacket(index uint16, amount uint32) []byte {
	return buildCartMovePacket(PacketCZMoveFromCart, index, amount)
}

func BuildMoveStorageToCartPacket(index uint16, amount uint32) []byte {
	return buildCartMovePacket(PacketCZMoveStorageToCart, index, amount)
}

func BuildMoveCartToStoragePacket(index uint16, amount uint32) []byte {
	return buildCartMovePacket(PacketCZMoveCartToStorage, index, amount)
}

func buildCartMovePacket(opcode uint16, index uint16, amount uint32) []byte {
	if amount == 0 {
		amount = 1
	}
	packet := make([]byte, 8)
	binary.LittleEndian.PutUint16(packet[0:2], opcode)
	binary.LittleEndian.PutUint16(packet[2:4], index)
	binary.LittleEndian.PutUint32(packet[4:8], amount)
	return packet
}

func BuildShopDealSelectionPacket(npcID uint32, dealType uint8) []byte {
	var w Writer
	w.Uint16(PacketCZACKSelectDealType)
	w.Uint32(npcID)
	w.Uint8(dealType)
	return w.Bytes()
}

func BuildSellItemListPacket(items []SellRequestItem) []byte {
	size := 4 + len(items)*4
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZPCSellItemList)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	offset := 4
	for _, item := range items {
		binary.LittleEndian.PutUint16(packet[offset:offset+2], item.Index)
		binary.LittleEndian.PutUint16(packet[offset+2:offset+4], item.Amount)
		offset += 4
	}
	return packet
}

func BuildBuyItemListPacket(items []BuyRequestItem) []byte {
	size := 4 + len(items)*4
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZPCPurchaseItemList)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(size))
	offset := 4
	for _, item := range items {
		binary.LittleEndian.PutUint16(packet[offset:offset+2], item.Amount)
		binary.LittleEndian.PutUint16(packet[offset+2:offset+4], item.ItemID)
		offset += 4
	}
	return packet
}

func (c *Client) SendItemPickup(gid uint32) error {
	packet := BuildItemPickupPacketForClientDate(gid, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ITEM_PICKUP opcode=0x%04X target=%d client_date=%d", ID(packet), gid, c.clientDate)
	} else {
		glog.Warnf("send CZ_ITEM_PICKUP failed opcode=0x%04X len=%d target=%d client_date=%d: %v", ID(packet), len(packet), gid, c.clientDate, err)
	}
	return err
}

func (c *Client) SendUseInventoryItem(index uint16, targetAID uint32) error {
	packet := BuildUseInventoryItemPacketForClientDate(index, targetAID, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_USE_ITEM opcode=0x%04X index=%d target=%d client_date=%d", ID(packet), index, targetAID, c.clientDate)
	} else {
		glog.Warnf("send CZ_USE_ITEM failed opcode=0x%04X len=%d index=%d target=%d client_date=%d: %v", ID(packet), len(packet), index, targetAID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendDropInventoryItem(index, amount uint16) error {
	packet := BuildDropInventoryItemPacketForClientDate(index, amount, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ITEM_THROW opcode=0x%04X index=%d amount=%d client_date=%d", ID(packet), index, amount, c.clientDate)
	} else {
		glog.Warnf("send CZ_ITEM_THROW failed opcode=0x%04X len=%d index=%d amount=%d client_date=%d: %v", ID(packet), len(packet), index, amount, c.clientDate, err)
	}
	return err
}

func (c *Client) SendItemIdentify(index uint16) error {
	packet := BuildItemIdentifyPacket(index)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_ITEMIDENTIFY opcode=0x%04X index=%d client_date=%d", ID(packet), index, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_ITEMIDENTIFY failed opcode=0x%04X len=%d index=%d client_date=%d: %v", ID(packet), len(packet), index, c.clientDate, err)
	}
	return err
}

func (c *Client) SendItemCompositionList(cardIndex uint16) error {
	packet := BuildItemCompositionListPacket(cardIndex)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_ITEMCOMPOSITION_LIST opcode=0x%04X card_index=%d client_date=%d", ID(packet), cardIndex, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_ITEMCOMPOSITION_LIST failed opcode=0x%04X len=%d card_index=%d client_date=%d: %v", ID(packet), len(packet), cardIndex, c.clientDate, err)
	}
	return err
}

func (c *Client) SendItemComposition(cardIndex, equipIndex uint16) error {
	packet := BuildItemCompositionPacket(cardIndex, equipIndex)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_ITEMCOMPOSITION opcode=0x%04X card_index=%d equip_index=%d client_date=%d", ID(packet), cardIndex, equipIndex, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_ITEMCOMPOSITION failed opcode=0x%04X len=%d card_index=%d equip_index=%d client_date=%d: %v", ID(packet), len(packet), cardIndex, equipIndex, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMakingArrow(itemID uint16) error {
	packet := BuildMakingArrowPacket(itemID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_MAKINGARROW opcode=0x%04X item=%d client_date=%d", ID(packet), itemID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_MAKINGARROW failed opcode=0x%04X len=%d item=%d client_date=%d: %v", ID(packet), len(packet), itemID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendWearEquip(index, location uint16) error {
	packet := BuildWearEquipPacket(index, location)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_WEAR_EQUIP opcode=0x%04X index=%d location=0x%04X client_date=%d", ID(packet), index, location, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_WEAR_EQUIP failed opcode=0x%04X index=%d location=0x%04X client_date=%d: %v", ID(packet), index, location, c.clientDate, err)
	}
	return err
}

func (c *Client) SendTakeoffEquip(index uint16) error {
	packet := BuildTakeoffEquipPacket(index)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_TAKEOFF_EQUIP opcode=0x%04X index=%d client_date=%d", ID(packet), index, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_TAKEOFF_EQUIP failed opcode=0x%04X index=%d client_date=%d: %v", ID(packet), index, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMoveToStorage(index uint16, amount uint32) error {
	packet := BuildMoveToStoragePacketForClientDate(index, amount, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MOVE_ITEM_TO_STORAGE opcode=0x%04X index=%d amount=%d client_date=%d", ID(packet), index, amount, c.clientDate)
	} else {
		glog.Warnf("send CZ_MOVE_ITEM_TO_STORAGE failed opcode=0x%04X len=%d index=%d amount=%d client_date=%d: %v", ID(packet), len(packet), index, amount, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMoveFromStorage(index uint16, amount uint32) error {
	packet := BuildMoveFromStoragePacketForClientDate(index, amount, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MOVE_ITEM_FROM_STORAGE opcode=0x%04X index=%d amount=%d client_date=%d", ID(packet), index, amount, c.clientDate)
	} else {
		glog.Warnf("send CZ_MOVE_ITEM_FROM_STORAGE failed opcode=0x%04X len=%d index=%d amount=%d client_date=%d: %v", ID(packet), len(packet), index, amount, c.clientDate, err)
	}
	return err
}

func (c *Client) SendCloseStorage() error {
	packet := BuildCloseStoragePacketForClientDate(c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CLOSE_STORAGE opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_CLOSE_STORAGE failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendMoveToCart(index uint16, amount uint32) error {
	packet := BuildMoveToCartPacket(index, amount)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MOVE_ITEM_TO_CART opcode=0x%04X index=%d amount=%d client_date=%d", ID(packet), index, amount, c.clientDate)
	} else {
		glog.Warnf("send CZ_MOVE_ITEM_TO_CART failed opcode=0x%04X len=%d index=%d amount=%d client_date=%d: %v", ID(packet), len(packet), index, amount, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMoveFromCart(index uint16, amount uint32) error {
	packet := BuildMoveFromCartPacket(index, amount)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MOVE_ITEM_FROM_CART opcode=0x%04X index=%d amount=%d client_date=%d", ID(packet), index, amount, c.clientDate)
	} else {
		glog.Warnf("send CZ_MOVE_ITEM_FROM_CART failed opcode=0x%04X len=%d index=%d amount=%d client_date=%d: %v", ID(packet), len(packet), index, amount, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMoveStorageToCart(index uint16, amount uint32) error {
	packet := BuildMoveStorageToCartPacket(index, amount)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MOVE_ITEM_FROM_STORAGE_TO_CART opcode=0x%04X index=%d amount=%d client_date=%d", ID(packet), index, amount, c.clientDate)
	} else {
		glog.Warnf("send CZ_MOVE_ITEM_FROM_STORAGE_TO_CART failed opcode=0x%04X len=%d index=%d amount=%d client_date=%d: %v", ID(packet), len(packet), index, amount, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMoveCartToStorage(index uint16, amount uint32) error {
	packet := BuildMoveCartToStoragePacket(index, amount)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MOVE_ITEM_FROM_CART_TO_STORAGE opcode=0x%04X index=%d amount=%d client_date=%d", ID(packet), index, amount, c.clientDate)
	} else {
		glog.Warnf("send CZ_MOVE_ITEM_FROM_CART_TO_STORAGE failed opcode=0x%04X len=%d index=%d amount=%d client_date=%d: %v", ID(packet), len(packet), index, amount, c.clientDate, err)
	}
	return err
}

func (c *Client) SendShopDealSelection(npcID uint32, dealType uint8) error {
	packet := BuildShopDealSelectionPacket(npcID, dealType)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ACK_SELECT_DEALTYPE opcode=0x%04X npc=%d type=%d client_date=%d", ID(packet), npcID, dealType, c.clientDate)
	} else {
		glog.Warnf("send CZ_ACK_SELECT_DEALTYPE failed opcode=0x%04X npc=%d type=%d client_date=%d: %v", ID(packet), npcID, dealType, c.clientDate, err)
	}
	return err
}

func (c *Client) SendShopSellItems(items []SellRequestItem) error {
	packet := BuildSellItemListPacket(items)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_PC_SELL_ITEMLIST opcode=0x%04X count=%d client_date=%d", ID(packet), len(items), c.clientDate)
	} else {
		glog.Warnf("send CZ_PC_SELL_ITEMLIST failed opcode=0x%04X count=%d client_date=%d: %v", ID(packet), len(items), c.clientDate, err)
	}
	return err
}

func (c *Client) SendShopBuyItems(items []BuyRequestItem) error {
	packet := BuildBuyItemListPacket(items)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_PC_PURCHASE_ITEMLIST opcode=0x%04X count=%d client_date=%d", ID(packet), len(items), c.clientDate)
	} else {
		glog.Warnf("send CZ_PC_PURCHASE_ITEMLIST failed opcode=0x%04X count=%d client_date=%d: %v", ID(packet), len(items), c.clientDate, err)
	}
	return err
}
