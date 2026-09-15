package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
)

func TestTradeAcceptancePreservesPartnerName(t *testing.T) {
	for _, tc := range []struct {
		name     string
		incoming bool
		partner  string
		want     string
	}{
		{name: "incoming", incoming: true, partner: " Zambla ", want: "Zambla"},
		{name: "outgoing", partner: " Kivy ", want: "Kivy"},
		{name: "unnamed incoming", incoming: true, want: "Player"},
		{name: "unnamed outgoing", want: "Player"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			netClient, server := newBotTestConnection(t, 20080910)
			ctx := client.Context{Network: netClient, ScreenW: 1024, ScreenH: 768}
			mode := &WorldMode{}
			if tc.incoming {
				mode.openTradeRequest(ctx, network.TradeRequest{Name: tc.partner})
				if !mode.ui.tradeRequest.IsOpen() || mode.ui.tradeWindow.IsOpen() {
					t.Fatal("incoming request should only open the confirmation")
				}
				mode.ui.tradeRequest.Confirm(ctx)
				readBotTestPackets(t, server, network.BuildTradeAckPacket(true))
				if mode.ui.tradeRequest.IsOpen() {
					t.Fatal("accepted request left its confirmation open")
				}
			} else {
				mode.sendTradeRequest(ctx, 200, tc.partner)
				readBotTestPackets(t, server, network.BuildTradeRequestPacket(200))
			}
			if mode.ui.tradeWindow.IsOpen() {
				t.Fatal("trade opened before server acceptance")
			}

			mode.handleTradeResponse(ctx, network.TradeResponse{Result: 3})
			if !mode.ui.tradeWindow.IsOpen() {
				t.Fatal("server acceptance did not open trade")
			}
			if text := whisperWindowText(mode.ui.tradeWindow.Widget()); !strings.Contains(text, "Trade with "+tc.want) {
				t.Fatalf("trade window text = %q, want partner %q", text, tc.want)
			}
			if mode.pendingTradeName != "" {
				t.Fatalf("accepted trade retained pending name %q", mode.pendingTradeName)
			}
		})
	}
}

func TestTradeRejectionDoesNotOpenTrade(t *testing.T) {
	netClient, server := newBotTestConnection(t, 20080910)
	ctx := client.Context{Network: netClient}
	mode := &WorldMode{}
	mode.openTradeRequest(ctx, network.TradeRequest{Name: "Zambla"})
	mode.ui.tradeRequest.Cancel(ctx)
	readBotTestPackets(t, server, network.BuildTradeAckPacket(false))
	if mode.ui.tradeRequest.IsOpen() || mode.ui.tradeWindow.IsOpen() || mode.pendingTradeName != "" {
		t.Fatal("rejected request left an open window or a pending trade")
	}
}

func TestTradeFailureClearsPendingName(t *testing.T) {
	for _, result := range []uint8{0, 1, 2, 4, 5, 255} {
		t.Run(fmt.Sprintf("result_%d", result), func(t *testing.T) {
			netClient, server := newBotTestConnection(t, 20080910)
			ctx := client.Context{Network: netClient}
			mode := &WorldMode{}
			mode.openTradeRequest(ctx, network.TradeRequest{Name: "Zambla"})
			mode.ui.tradeRequest.Confirm(ctx)
			readBotTestPackets(t, server, network.BuildTradeAckPacket(true))
			mode.handleTradeResponse(ctx, network.TradeResponse{Result: result})
			if mode.ui.tradeWindow.IsOpen() || mode.pendingTradeName != "" {
				t.Fatal("failed trade left an open window or a pending name")
			}
		})
	}
}

func TestTradeSendFailureDoesNotOpenTrade(t *testing.T) {
	for _, netClient := range []*network.Client{nil, network.NewClient(20080910, false)} {
		ctx := client.Context{Network: netClient}
		mode := &WorldMode{}
		mode.openTradeRequest(ctx, network.TradeRequest{Name: "Zambla"})
		mode.ui.tradeRequest.Confirm(ctx)
		if mode.ui.tradeWindow.IsOpen() || mode.pendingTradeName != "" {
			t.Fatal("failed acknowledgement opened a trade or retained its name")
		}
		mode.sendTradeRequest(ctx, 200, "Kivy")
		if mode.ui.tradeWindow.IsOpen() || mode.pendingTradeName != "" {
			t.Fatal("failed request opened a trade or retained its name")
		}
	}
}
