package ui

import (
	"cmp"
	"slices"

	"github.com/kivutar/goro/session"
)

// itemDialogList caches the sorted inventory entries offered by the server.
// Returned slices are immutable: table callbacks may still refer to the previous
// list when an inventory update arrives.
type itemDialogList struct {
	inventory        []session.InventoryItem
	indexes          []uint16
	unidentifiedOnly bool
	items            []session.InventoryItem
}

func (l *itemDialogList) get(s *session.Session, indexes []uint16, unidentifiedOnly bool) []session.InventoryItem {
	var inventory []session.InventoryItem
	if s != nil {
		inventory = s.Inventory.Items
	}
	if slices.Equal(l.inventory, inventory) && slices.Equal(l.indexes, indexes) && l.unidentifiedOnly == unidentifiedOnly {
		return l.items
	}
	l.inventory = append(l.inventory[:0], inventory...)
	l.indexes = append(l.indexes[:0], indexes...)
	l.unidentifiedOnly = unidentifiedOnly

	items := make([]session.InventoryItem, 0, len(indexes))
	for _, index := range indexes {
		if item, ok := findInventoryItemByIndex(s, index); ok {
			if unidentifiedOnly && (item.Identified || !inventoryItemCanEquip(item)) {
				continue
			}
			items = append(items, item)
		}
	}
	slices.SortFunc(items, func(a, b session.InventoryItem) int { return cmp.Compare(a.Index, b.Index) })
	l.items = items
	return items
}
