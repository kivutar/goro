package client

import "fmt"

const TokenOfSiegfriedItemID uint16 = 7621

// AutoReviveAvailable reports whether the local player can offer the classic
// Token of Siegfried resurrection action.
func AutoReviveAvailable(ctx Context) bool {
	if ctx.Session == nil || !ctx.Session.Dead || ctx.World == nil {
		return false
	}
	if ctx.World.MapProperty.IsAutoReviveRestricted() {
		return false
	}
	for _, item := range ctx.Session.Inventory.Items {
		if item.ItemID == TokenOfSiegfriedItemID && item.Amount > 0 {
			return true
		}
	}
	return false
}

// RequestAutoRevive sends the normal client request. The server remains
// authoritative over consuming the token and resurrecting the character.
func RequestAutoRevive(ctx Context) error {
	if !AutoReviveAvailable(ctx) {
		return fmt.Errorf("auto-revive is not available")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	return ctx.Network.SendAutoRevive()
}
