package game

import (
	"fmt"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestIdentifyAckDoesNotControlPicker(t *testing.T) {
	for _, ack := range []network.ItemIdentifyAck{
		{Index: 7, Success: true},
		{Index: 7, Success: false},
		{Index: 0xFFFF, Success: false},
	} {
		for _, newerPicker := range []bool{false, true} {
			t.Run(fmt.Sprintf("index_%d_success_%t_newer_picker_%t", ack.Index, ack.Success, newerPicker), func(t *testing.T) {
				manager := &worldModeTestUIManager{}
				ctx := client.Context{
					Session: &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
						{Index: 7, ItemID: 2301, Type: db.ItemTypeArmor},
						{Index: 9, ItemID: 1201, Type: db.ItemTypeWeapon},
					}}},
					UIManager: manager,
					ScreenW:   800,
					ScreenH:   600,
				}
				mode := &WorldMode{}
				mode.ui.identifyWindow.OpenList(ctx, network.ItemIdentifyList{Indexes: []uint16{7, 9}})
				mode.ui.identifyWindow.Close()
				if newerPicker {
					mode.ui.identifyWindow.OpenList(ctx, network.ItemIdentifyList{Indexes: []uint16{9}})
				}
				content := mode.ui.identifyWindow.Widget()
				if content != nil {
					content = content.Children()[0]
				}
				result := byte(1)
				if ack.Success {
					result = 0
				}
				packet := network.Packet{
					ID:   0x0179,
					Data: []byte{0x79, 0x01, byte(ack.Index), byte(ack.Index >> 8), result},
				}
				if next, stop := mode.handleNetworkPacket(ctx, packet, time.Now()); next != nil || stop {
					t.Fatalf("identify ack changed mode: next=%T stop=%t", next, stop)
				}
				after := mode.ui.identifyWindow.Widget()
				if after != nil {
					after = after.Children()[0]
				}
				if mode.ui.identifyWindow.IsOpen() != newerPicker || after != content {
					t.Fatal("acknowledgement changed the picker lifecycle or content")
				}
				wantOverlays := 0
				if newerPicker {
					wantOverlays = 1
				}
				if len(manager.overlays) != wantOverlays {
					t.Fatalf("acknowledgement changed overlays: %d", len(manager.overlays))
				}
				if ctx.Session.Inventory.Items[0].Identified != ack.Success || ctx.Session.Inventory.Items[1].Identified {
					t.Fatalf("inventory after acknowledgement = %+v", ctx.Session.Inventory.Items)
				}
			})
		}
	}
}
