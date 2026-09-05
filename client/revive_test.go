package client

import (
	"testing"

	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestAutoReviveAvailable(t *testing.T) {
	sessionState := session.New()
	sessionState.Dead = true
	sessionState.Inventory.Items = []session.InventoryItem{{ItemID: TokenOfSiegfriedItemID, Amount: 1}}
	world := worldstate.New()
	ctx := Context{Session: sessionState, World: world}
	if !AutoReviveAvailable(ctx) {
		t.Fatal("Token of Siegfried was not available on an ordinary map")
	}

	for _, property := range []worldstate.MapProperty{
		worldstate.MapPropertyEventPvPZone,
		worldstate.MapPropertyAgitZone,
		worldstate.MapPropertyPKServerZone,
	} {
		world.MapProperty = property
		if AutoReviveAvailable(ctx) {
			t.Fatalf("Token of Siegfried was available on restricted map property %d", property)
		}
	}

	world.MapProperty = worldstate.MapPropertyNothing
	sessionState.Dead = false
	if AutoReviveAvailable(ctx) {
		t.Fatal("Token of Siegfried was available while alive")
	}

	sessionState.Dead = true
	sessionState.Inventory.Items[0].Amount = 0
	if AutoReviveAvailable(ctx) {
		t.Fatal("depleted Token of Siegfried remained available")
	}
}
