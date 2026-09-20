package ui

import (
	"strings"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
)

func TestUseInventoryItemRejectsDeadPlayer(t *testing.T) {
	err := UseInventoryItem(client.Context{
		Session: &session.Session{Dead: true},
	}, session.InventoryItem{
		Index:  7,
		ItemID: 501,
		Type:   db.ItemTypeHealing,
	})

	if err == nil || !strings.Contains(err.Error(), "dead") {
		t.Fatalf("error = %v, want dead-player rejection", err)
	}
}

func TestInventoryActivatesBranches(t *testing.T) {
	for _, itemID := range []uint16{604, 12103} {
		connection, server := newIdentifyTestConnection(t)
		ctx := client.Context{
			Network: connection,
			Session: &session.Session{AccountID: 2000000},
		}
		var bag InventoryBagWindow
		bag.activateItem(ctx, session.InventoryItem{
			Index: 12, ItemID: itemID, Type: db.ItemTypeUsable, Amount: 4, Identified: true,
		})
		// 2008 Sakray CZ_USE_ITEM2: opcode, inventory index, account ID.
		readIdentifyTestPacket(t, server, []byte{0x39, 0x04, 12, 0, 0x80, 0x84, 0x1E, 0})
	}
}
