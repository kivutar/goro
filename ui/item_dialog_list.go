package ui

import (
	"cmp"
	"slices"

	"github.com/kivutar/goro/session"
)

// itemDialogList caches the sorted inventory entries offered by the server.
// All cached slices are immutable: windows are copied across map changes, and
// table callbacks may still refer to the previous window and its list.
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
	l.inventory = slices.Clone(inventory)
	l.indexes = slices.Clone(indexes)
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
