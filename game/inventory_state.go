package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func applyInventoryItemList(ctx client.Context, items []network.InventoryItem) {
	if ctx.Session == nil {
		return
	}
	for _, item := range items {
		addOrReplaceSessionInventoryItem(ctx.Session, sessionItemFromNetwork(item))
	}
	rebuildLocalEquipmentAppearance(ctx)
}

func applyStorageItemList(ctx client.Context, items []network.InventoryItem) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Storage.Open = !ctx.Session.Storage.PasswordPending
	for _, item := range items {
		addOrReplaceSessionStorageItem(ctx.Session, sessionItemFromNetwork(item))
	}
}

func applyCartItemList(ctx client.Context, items []network.InventoryItem) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Cart.Open = true
	for _, item := range items {
		addOrReplaceSessionCartItem(ctx.Session, sessionItemFromNetwork(item))
	}
}

func applyStorageAmount(ctx client.Context, amount network.StorageAmount) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Storage.Open = !ctx.Session.Storage.PasswordPending
	ctx.Session.Storage.Amount = int(amount.Amount)
	ctx.Session.Storage.MaxAmount = int(amount.MaxAmount)
}

func applyCartAmount(ctx client.Context, amount network.CartAmount) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Cart.Open = true
	ctx.Session.Cart.Amount = int(amount.Amount)
	ctx.Session.Cart.MaxAmount = int(amount.MaxAmount)
	ctx.Session.Cart.Weight = int(amount.Weight)
	ctx.Session.Cart.MaxWeight = int(amount.MaxWeight)
	refreshLocalPlayerMoveSpeed(ctx)
}

func applyStorageItemAdded(ctx client.Context, item network.InventoryItem) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Storage.Open = !ctx.Session.Storage.PasswordPending
	addOrReplaceSessionStorageItem(ctx.Session, sessionItemFromNetwork(item))
}

func applyCartItemAdded(ctx client.Context, item network.InventoryItem) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Cart.Open = true
	addOrReplaceSessionCartItem(ctx.Session, sessionItemFromNetwork(item))
}

func applyStorageItemRemoved(ctx client.Context, item network.StorageItemRemoved) {
	removeSessionStorageItem(ctx.Session, item.Index, int(item.Amount))
}

func applyCartItemRemoved(ctx client.Context, item network.CartItemRemoved) {
	removeSessionCartItem(ctx.Session, item.Index, int(item.Amount))
}

func applyStorageClosed(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Storage.Open = false
	ctx.Session.Storage.PasswordPending = false
	ctx.Session.Storage.Items = nil
	ctx.Session.Storage.Amount = 0
	ctx.Session.Storage.MaxAmount = 0
}

func applyCartClosed(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Cart.Open = false
}

func applyInventoryItemDelete(ctx client.Context, item network.InventoryItemDelete) {
	removeSessionInventoryItem(ctx.Session, item.Index, int(item.Amount))
	rebuildLocalEquipmentAppearance(ctx)
}

func applyItemIdentifyAck(ctx client.Context, ack network.ItemIdentifyAck) {
	if ctx.Session == nil || !ack.Success {
		return
	}
	for i := range ctx.Session.Inventory.Items {
		if ctx.Session.Inventory.Items[i].Index == ack.Index {
			ctx.Session.Inventory.Items[i].Identified = true
			return
		}
	}
}

func applyItemCompositionAck(ctx client.Context, ack network.ItemCompositionAck) {
	if ctx.Session == nil || !ack.Success {
		return
	}
	cardItem, ok := findSessionInventoryItem(ctx.Session, ack.CardIndex)
	if !ok {
		return
	}
	removeSessionInventoryItem(ctx.Session, ack.CardIndex, 1)
	for i := range ctx.Session.Inventory.Items {
		if ctx.Session.Inventory.Items[i].Index != ack.EquipIndex {
			continue
		}
		for slot := range ctx.Session.Inventory.Items[i].Cards {
			if ctx.Session.Inventory.Items[i].Cards[slot] == 0 {
				ctx.Session.Inventory.Items[i].Cards[slot] = cardItem.ItemID
				break
			}
		}
		normalizeSessionInventoryItem(&ctx.Session.Inventory.Items[i])
		return
	}
}

func useItemAckIsLocal(ctx client.Context, ack network.UseItemAck) bool {
	// AID identifies an account, never a character. Legacy 0x00A8
	// acknowledgements omit AID and update the local inventory.
	return ctx.Session != nil && (ack.AID == 0 || ack.AID == ctx.Session.AccountID)
}

func applyUseItemAck(ctx client.Context, ack network.UseItemAck) {
	if ack.Result == 0 || !useItemAckIsLocal(ctx, ack) {
		return
	}
	setSessionInventoryItemAmount(ctx.Session, ack.Index, int(ack.Amount))
}

func sessionItemFromNetwork(item network.InventoryItem) session.InventoryItem {
	amount := int(item.Amount)
	if amount <= 0 {
		amount = 1
	}
	sessionItem := session.InventoryItem{
		Index:      item.Index,
		ItemID:     item.ItemID,
		Type:       item.Type,
		Location:   item.Location,
		Identified: item.Identified,
		Amount:     amount,
		Equip:      item.Equip || inventoryItemTypeIsEquipment(item.Type),
		Equipped:   item.Equipped,
		Damaged:    item.Damaged,
		Refine:     item.Refine,
		Cards:      item.Cards,
	}
	normalizeSessionInventoryItem(&sessionItem)
	return sessionItem
}

func addOrReplaceSessionInventoryItem(s *session.Session, item session.InventoryItem) {
	if s == nil || item.Index == 0 {
		return
	}
	normalizeSessionInventoryItem(&item)
	if item.Amount <= 0 {
		item.Amount = 1
	}
	for i := range s.Inventory.Items {
		if s.Inventory.Items[i].Index != item.Index {
			continue
		}
		s.Inventory.Items[i] = item
		return
	}
	s.Inventory.Items = append(s.Inventory.Items, item)
}

func addOrReplaceSessionStorageItem(s *session.Session, item session.InventoryItem) {
	if s == nil || item.Index == 0 {
		return
	}
	normalizeSessionInventoryItem(&item)
	if item.Amount <= 0 {
		item.Amount = 1
	}
	for i := range s.Storage.Items {
		if s.Storage.Items[i].Index != item.Index {
			continue
		}
		s.Storage.Items[i] = item
		return
	}
	s.Storage.Items = append(s.Storage.Items, item)
}

func addOrReplaceSessionCartItem(s *session.Session, item session.InventoryItem) {
	if s == nil || item.Index == 0 {
		return
	}
	normalizeSessionInventoryItem(&item)
	if item.Amount <= 0 {
		item.Amount = 1
	}
	for i := range s.Cart.Items {
		if s.Cart.Items[i].Index != item.Index {
			continue
		}
		s.Cart.Items[i] = item
		return
	}
	s.Cart.Items = append(s.Cart.Items, item)
}

func applyInventoryEquipAck(ctx client.Context, ack network.InventoryEquipAck) {
	if ctx.Session == nil || !ack.Success || ack.Index == 0 {
		return
	}
	location := ack.Location
	if !ack.Unequip && location == 0 {
		location = inventoryEquipAckDefaultLocation(ctx.Session, ack.Index)
	}
	for i := range ctx.Session.Inventory.Items {
		item := &ctx.Session.Inventory.Items[i]
		if item.Index != ack.Index {
			if !ack.Unequip && location != 0 && item.Equipped && item.Location&location != 0 {
				item.Equipped = false
			}
			continue
		}
		item.Equipped = !ack.Unequip
		if !ack.Unequip && location != 0 {
			item.Location = location
		}
		if !ack.Unequip {
			item.Equip = true
			normalizeSessionInventoryItem(item)
		}
	}
	rebuildLocalEquipmentAppearance(ctx)
}

func applyEquippedArrow(ctx client.Context, arrow network.EquippedArrow) {
	applyInventoryEquipAck(ctx, network.InventoryEquipAck{
		Index:    arrow.Index,
		Location: db.EquipAmmo,
		Success:  true,
	})
}

func normalizeSessionInventoryItem(item *session.InventoryItem) {
	if item == nil {
		return
	}
	if item.Type == db.ItemTypeCard {
		item.Equip = false
		item.Equipped = false
		return
	}
	if inventoryItemTypeIsEquipment(item.Type) {
		item.Equip = true
	}
	if item.Location == 0 {
		item.Location = inventoryItemDefaultEquipLocation(item.Type)
	}
}

func inventoryEquipAckDefaultLocation(s *session.Session, index uint16) uint16 {
	if s == nil || index == 0 {
		return 0
	}
	for _, item := range s.Inventory.Items {
		if item.Index == index {
			return inventoryItemDefaultEquipLocation(item.Type)
		}
	}
	return 0
}

func rebuildLocalEquipmentAppearance(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	sawEquipment := false
	hasWeapon := false
	hasShield := false
	weapon := int(ctx.Session.Selected.Weapon)
	shield := int(ctx.Session.Selected.Shield)
	for _, item := range ctx.Session.Inventory.Items {
		if !item.Equip {
			continue
		}
		sawEquipment = true
		if !item.Equipped || item.Location == 0 {
			continue
		}
		occupiesRightHand := item.Location&db.EquipWeapon != 0
		occupiesLeftHand := item.Location&db.EquipShield != 0
		if occupiesRightHand {
			hasWeapon = true
			weapon = int(item.ItemID)
		}
		if occupiesLeftHand && !occupiesRightHand {
			hasShield = true
			shield = int(item.ItemID)
		}
	}
	if !sawEquipment {
		return
	}
	ctx.Session.AttackRange = normalAttackRangeFromEquippedItems(ctx.Session, ctx.Resources)
	if !hasWeapon {
		weapon = 0
	}
	if !hasShield {
		shield = 0
	}
	updateLocalWeaponAppearance(ctx, weapon, shield)
}

func normalAttackRangeFromEquippedItems(s *session.Session, manager *res.Manager) int {
	if s == nil {
		return 1
	}
	for _, item := range s.Inventory.Items {
		if !item.Equip || !item.Equipped || item.Location&db.EquipWeapon == 0 {
			continue
		}
		if res.PlayerWeaponIsBow(manager, int(item.ItemID)) {
			return 5 + learnedSkillLevel(s, 44)
		}
		return 1
	}
	return 1
}

func learnedSkillLevel(s *session.Session, id uint16) int {
	if s == nil {
		return 0
	}
	for _, skill := range s.Skills.List {
		if skill.ID == id {
			return maxInt(0, skill.Level)
		}
	}
	return 0
}

func updateLocalWeaponAppearance(ctx client.Context, weapon, shield int) {
	if ctx.Session == nil {
		return
	}
	weapon, shield = res.NormalizePlayerWeaponShield(weapon, shield)
	ctx.Session.Selected.Weapon = int16(weapon)
	ctx.Session.Selected.Shield = int16(shield)
	for i := range ctx.Session.Characters {
		if ctx.Session.Characters[i].ID == ctx.Session.CharID || ctx.Session.Characters[i].ID == ctx.Session.Selected.ID {
			ctx.Session.Characters[i].Weapon = int16(weapon)
			ctx.Session.Characters[i].Shield = int16(shield)
		}
	}
	if ctx.World != nil {
		ctx.World.Player.Weapon = int16(weapon)
		ctx.World.Player.Shield = int16(shield)
	}
}

func addPickedSessionInventoryItem(s *session.Session, item session.InventoryItem) {
	if s == nil || item.Index == 0 {
		return
	}
	normalizeSessionInventoryItem(&item)
	if item.Amount <= 0 {
		item.Amount = 1
	}
	for i := range s.Inventory.Items {
		if s.Inventory.Items[i].Index != item.Index {
			continue
		}
		if s.Inventory.Items[i].ItemID == item.ItemID && !s.Inventory.Items[i].Equip {
			s.Inventory.Items[i].Amount += item.Amount
			if item.Location != 0 {
				s.Inventory.Items[i].Location = item.Location
			}
			return
		}
		s.Inventory.Items[i] = item
		return
	}
	s.Inventory.Items = append(s.Inventory.Items, item)
}

func removeSessionInventoryItem(s *session.Session, index uint16, amount int) {
	if s == nil || index == 0 {
		return
	}
	if amount <= 0 {
		amount = 1
	}
	for i := range s.Inventory.Items {
		if s.Inventory.Items[i].Index != index {
			continue
		}
		s.Inventory.Items[i].Amount -= amount
		if s.Inventory.Items[i].Amount > 0 {
			return
		}
		s.Inventory.Items = append(s.Inventory.Items[:i], s.Inventory.Items[i+1:]...)
		return
	}
}

func findSessionInventoryItem(s *session.Session, index uint16) (session.InventoryItem, bool) {
	if s == nil || index == 0 {
		return session.InventoryItem{}, false
	}
	for _, item := range s.Inventory.Items {
		if item.Index == index {
			return item, true
		}
	}
	return session.InventoryItem{}, false
}

func removeSessionStorageItem(s *session.Session, index uint16, amount int) {
	if s == nil || index == 0 {
		return
	}
	if amount <= 0 {
		amount = 1
	}
	for i := range s.Storage.Items {
		if s.Storage.Items[i].Index != index {
			continue
		}
		s.Storage.Items[i].Amount -= amount
		if s.Storage.Items[i].Amount > 0 {
			return
		}
		s.Storage.Items = append(s.Storage.Items[:i], s.Storage.Items[i+1:]...)
		return
	}
}

func removeSessionCartItem(s *session.Session, index uint16, amount int) {
	if s == nil || index == 0 {
		return
	}
	if amount <= 0 {
		amount = 1
	}
	for i := range s.Cart.Items {
		if s.Cart.Items[i].Index != index {
			continue
		}
		s.Cart.Items[i].Amount -= amount
		if s.Cart.Items[i].Amount > 0 {
			return
		}
		s.Cart.Items = append(s.Cart.Items[:i], s.Cart.Items[i+1:]...)
		return
	}
}

func setSessionInventoryItemAmount(s *session.Session, index uint16, amount int) {
	if s == nil || index == 0 {
		return
	}
	for i := range s.Inventory.Items {
		if s.Inventory.Items[i].Index != index {
			continue
		}
		if amount > 0 {
			s.Inventory.Items[i].Amount = amount
			return
		}
		s.Inventory.Items = append(s.Inventory.Items[:i], s.Inventory.Items[i+1:]...)
		return
	}
}
