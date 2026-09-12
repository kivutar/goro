package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

func TestEscapeCancelsOnlyTopServerInteraction(t *testing.T) {
	netClient, server := newBotTestConnection(t, 20080910)
	manager := gameui.NewManager()
	ctx := client.Context{Network: netClient, Input: input.NewState(), UIManager: manager, ScreenW: 1024, ScreenH: 768}
	var trade gameui.TradeWindow
	var vending gameui.VendingWindow
	trade.Open(ctx, "Alice")
	vending.OpenSetup(ctx, network.VendingOpenRequest{MaxItems: 3})
	ctx.Input.SetKey(input.KeyEscape, true)
	if trade.Update(ctx, nil) || !trade.IsOpen() {
		t.Fatal("Escape canceled trade underneath the vending setup")
	}
	if !vending.Update(ctx, nil) || vending.KeyboardShortcutsBlocked() || !trade.IsOpen() {
		t.Fatal("Escape did not close both vending setup windows")
	}
	readBotTestPackets(t, server, network.BuildCancelVendingStoreOpenPacket())
	ctx.Input.ResetKeyboard()
	ctx.Input.SetKey(input.KeyEscape, true)
	if !trade.Update(ctx, nil) || trade.IsOpen() {
		t.Fatal("Escape did not close trade after vending")
	}
	readBotTestPackets(t, server, network.BuildTradeCancelPacket())
	ctx.Input.ResetKeyboard()
	vending.ApplyOwnList(ctx, network.VendingItemList{OwnerAID: 150004})
	ctx.Input.SetKey(input.KeyEscape, true)
	if !vending.Update(ctx, nil) || vending.KeyboardShortcutsBlocked() || manager.TopEscapeOverlay() != nil {
		t.Fatal("Escape left the player's vending store open")
	}
	readBotTestPackets(t, server, network.BuildCloseVendingStorePacket())
}
