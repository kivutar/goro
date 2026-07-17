package game

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

func (m *WorldMode) sendTradeRequest(ctx client.Context, actorID uint32, name string) {
	if actorID == 0 {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("trade request failed target=%d name=%q: not connected", actorID, name)
		return
	}
	if err := ctx.Network.SendTradeRequest(actorID); err != nil {
		glog.Warnf("trade request failed target=%d name=%q: %v", actorID, name, err)
		return
	}
	m.pendingTradeName = strings.TrimSpace(name)
}

func (m *WorldMode) openTradeRequest(ctx client.Context, request network.TradeRequest) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "Player"
	}
	message := fmt.Sprintf("%s wants to trade with you.", name)
	if request.TargetID != 0 || request.Level != 0 {
		message = fmt.Sprintf("%s\nLv.%d", message, request.Level)
	}
	m.ui.tradeRequest.Open(ctx, "Trade Request", message, func() {
		if ctx.Network == nil {
			glog.Warnf("trade request accept failed: not connected")
			return
		}
		if err := ctx.Network.SendTradeAck(true); err != nil {
			glog.Warnf("trade request accept failed name=%q: %v", request.Name, err)
			return
		}
		m.ui.tradeWindow.Open(ctx, name)
	}, func() {
		if ctx.Network == nil {
			glog.Warnf("trade request reject failed: not connected")
			return
		}
		if err := ctx.Network.SendTradeAck(false); err != nil {
			glog.Warnf("trade request reject failed name=%q: %v", request.Name, err)
		}
	})
}

func (m *WorldMode) handleTradeResponse(ctx client.Context, response network.TradeResponse) {
	name := m.pendingTradeName
	m.pendingTradeName = ""
	switch response.Result {
	case 0:
		m.ui.tradeWindow.Close(ctx)
		m.ui.console.AddErrorMessage("That character is too far away.")
	case 1:
		m.ui.tradeWindow.Close(ctx)
		m.ui.console.AddErrorMessage("Character does not exist.")
	case 2:
		m.ui.tradeWindow.Close(ctx)
		m.ui.console.AddErrorMessage("Trade failed.")
	case 3:
		if name == "" {
			name = "Player"
		}
		m.ui.tradeWindow.Open(ctx, name)
	case 4:
		m.ui.tradeWindow.Close(ctx)
		m.ui.console.AddErrorMessage("Trade canceled.")
	case 5:
		m.ui.tradeWindow.Close(ctx)
		m.ui.console.AddErrorMessage("That character is busy.")
	default:
		m.ui.tradeWindow.Close(ctx)
		m.ui.console.AddErrorMessage("Trade failed.")
	}
}

func (m *WorldMode) handleTradeExec(ctx client.Context, exec network.TradeExec) {
	m.ui.tradeWindow.Close(ctx)
	if exec.Result == 0 {
		m.ui.console.AddBlueMessage("Trade completed.")
		return
	}
	m.ui.console.AddErrorMessage("Trade failed.")
}
