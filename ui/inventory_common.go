package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

const (
	inventoryWindowPad = 10
	inventoryIconSize  = 24
)

var (
	inventoryTextColor  = TextColor
	inventoryMutedColor = MutedTextColor
)

func sortedInventoryItems(s *session.Session) []session.InventoryItem {
	if s == nil {
		return nil
	}
	items := make([]session.InventoryItem, 0, len(s.Inventory.Items))
	items = append(items, s.Inventory.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}

func inventoryItemDisplayName(manager *res.Manager, item session.InventoryItem) string {
	if manager != nil {
		if name, ok := manager.ItemDisplayName(int(item.ItemID), item.Identified); ok && strings.TrimSpace(name) != "" {
			return inventoryItemDisplayNameWithSlots(manager, item, name)
		}
	}
	return fmt.Sprintf("item %d", item.ItemID)
}

func drawInventoryItemTooltip(screen *render.Image, ctx Context, item session.InventoryItem) {
	if screen == nil || ctx.Input == nil {
		return
	}
	text := inventoryItemDisplayName(ctx.Resources, item)
	if strings.TrimSpace(text) == "" {
		return
	}
	render.DrawUITooltip(screen, text, float64(ctx.Input.MouseX), float64(ctx.Input.MouseY+18), float64(ctx.Input.MouseY-6))
}

func inventoryItemDisplayNameWithSlots(manager *res.Manager, item session.InventoryItem, name string) string {
	if !item.Identified || manager == nil || itemDisplayNameHasSlotSuffix(name) {
		return name
	}
	slotCount, ok := manager.ItemSlotCount(int(item.ItemID))
	if !ok || slotCount <= 0 {
		return name
	}
	return fmt.Sprintf("%s [%d]", name, slotCount)
}

func itemDisplayNameHasSlotSuffix(name string) bool {
	name = strings.TrimSpace(name)
	if !strings.HasSuffix(name, "]") {
		return false
	}
	open := strings.LastIndex(name, "[")
	if open < 0 || open+1 >= len(name)-1 {
		return false
	}
	_, err := strconv.Atoi(strings.TrimSpace(name[open+1 : len(name)-1]))
	return err == nil
}

func clampInventoryWindowInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
