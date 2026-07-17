package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	tradeWindowW     = 560
	tradeWindowH     = 360
	tradePanelW      = 250
	tradePanelH      = 216
	tradeRowH        = 28
	tradePanelPad    = 10
	tradePanelGap    = 12
	tradeZenyFieldW  = 96
	tradeZenyFieldH  = 24
	tradeFooterH     = 44
	tradeVisibleRows = 7
)

type TradeWindow struct {
	Window
	ctx Context

	partnerName string
	sendItems   []tradeWindowItem
	recvItems   []tradeWindowItem
	pending     map[uint16]session.InventoryItem

	zenyInput string
	sendZeny  uint32
	recvZeny  uint32
	selfOK    bool
	otherOK   bool
}

type tradeWindowItem struct {
	item session.InventoryItem
	name string
	icon image.Image
}

func (w *TradeWindow) Open(ctx Context, partnerName string) {
	w.EnsureWindow(tradeWindowW, tradeWindowH)
	w.ctx = ctx
	w.partnerName = strings.TrimSpace(partnerName)
	if w.partnerName == "" {
		w.partnerName = "Trade"
	}
	w.sendItems = nil
	w.recvItems = nil
	w.pending = make(map[uint16]session.InventoryItem)
	w.zenyInput = ""
	w.sendZeny = 0
	w.recvZeny = 0
	w.selfOK = false
	w.otherOK = false
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *TradeWindow) Update(ctx Context, itemInfo *ItemInfoWindow) bool {
	w.EnsureWindow(tradeWindowW, tradeWindowH)
	if !w.IsOpen() || ctx.Input == nil {
		return false
	}
	w.ctx = ctx
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *TradeWindow) Close(ctx Context) {
	w.EnsureWindow(tradeWindowW, tradeWindowH)
	w.Window.Close()
	w.Publish(ctx)
}

func (w *TradeWindow) AcceptInventoryDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	w.EnsureWindow(tradeWindowW, tradeWindowH)
	if !w.IsOpen() || w.selfOK || item.Index == 0 || !pointInRect(mx, my, w.sendPanelX(), w.sendPanelY(), tradePanelW, tradePanelH) {
		return false
	}
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if w.pending == nil {
		w.pending = make(map[uint16]session.InventoryItem)
	}
	w.pending[item.Index] = item
	if ctx.Network == nil {
		glog.Warnf("trade add item failed index=%d item=%d: not connected", item.Index, item.ItemID)
		return true
	}
	if err := ctx.Network.SendTradeAddItem(item.Index, amount); err != nil {
		glog.Warnf("trade add item failed index=%d item=%d amount=%d: %v", item.Index, item.ItemID, amount, err)
		delete(w.pending, item.Index)
	}
	return true
}

func (w *TradeWindow) AddOwnItemAck(ctx Context, ack network.TradeAddItemAck) {
	w.EnsureWindow(tradeWindowW, tradeWindowH)
	if ack.Result != 0 {
		delete(w.pending, ack.Index)
		glog.Warnf("trade add item rejected index=%d result=%d", ack.Index, ack.Result)
		return
	}
	if ack.Index == 0 {
		w.sendZeny = parseTradeZeny(w.zenyInput)
		w.refresh(ctx)
		return
	}
	item, ok := w.pending[ack.Index]
	delete(w.pending, ack.Index)
	if !ok {
		return
	}
	w.sendItems = append(w.sendItems, w.itemRow(ctx, item))
	w.refresh(ctx)
}

func (w *TradeWindow) AddReceivedItem(ctx Context, item network.TradeItem) {
	w.EnsureWindow(tradeWindowW, tradeWindowH)
	if item.ItemID == 0 {
		w.recvZeny = item.Amount
		w.refresh(ctx)
		return
	}
	w.recvItems = append(w.recvItems, w.itemRow(ctx, session.InventoryItem{
		ItemID:     item.ItemID,
		Amount:     minInt(int(item.Amount), int(^uint16(0))),
		Identified: item.Identified,
		Damaged:    item.Damaged,
		Refine:     item.Refine,
	}))
	w.refresh(ctx)
}

func (w *TradeWindow) SetConcluded(ctx Context, other bool) {
	w.EnsureWindow(tradeWindowW, tradeWindowH)
	if other {
		w.otherOK = true
	} else {
		w.selfOK = true
	}
	w.refresh(ctx)
}

func (w *TradeWindow) Undo(ctx Context) {
	w.EnsureWindow(tradeWindowW, tradeWindowH)
	w.selfOK = false
	w.sendItems = nil
	w.sendZeny = 0
	w.pending = make(map[uint16]session.InventoryItem)
	w.refresh(ctx)
}

func (w *TradeWindow) refresh(ctx Context) {
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *TradeWindow) itemRow(ctx Context, item session.InventoryItem) tradeWindowItem {
	return tradeWindowItem{
		item: item,
		name: inventoryItemDisplayName(ctx.Resources, item),
		icon: tradeItemIconImage(ctx.Resources, item),
	}
}

func (w *TradeWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Trade with "+w.partnerName),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
		}),
		Size(tradeWindowW, tradeWindowH),
		Content(
			primitives.Box(
				primitives.HBox(
					w.tradePanel("My Items", w.sendItems, w.sendZeny, true),
					w.tradePanel(w.partnerName, w.recvItems, w.recvZeny, false),
				).
					Gap(tradePanelGap),
			).
				Padding(tradePanelPad),
		),
		FooterHeight(tradeFooterH),
		Footer(
			primitives.HBox(
				rotheme.ButtonDisabledFn("OK", func() bool { return w.selfOK }, func() {
					w.conclude(ctx)
				}).
					Width(86),
				rotheme.ButtonDisabledFn("Trade", func() bool { return !w.selfOK || !w.otherOK }, func() {
					w.commit(ctx)
				}).
					Width(86),
				rotheme.Button("Cancel", func() {
					w.cancel(ctx)
				}).
					Width(86),
			).
				Gap(8),
		),
	)
}

func (w *TradeWindow) tradePanel(title string, items []tradeWindowItem, zeny uint32, editable bool) widget.Widget {
	rows := make([]widget.Widget, 0, tradeVisibleRows+3)
	rows = append(rows,
		primitives.Box(rotheme.Text(title)).
			Height(20),
	)
	for i := 0; i < tradeVisibleRows; i++ {
		var row tradeWindowItem
		if i < len(items) {
			row = items[i]
		}
		rows = append(rows, tradeItemRowWidget(row))
	}
	rows = append(rows,
		primitives.HBox(
			primitives.Box(rotheme.Text("Zeny")).
				Width(42),
			primitives.Expanded(primitives.Box()),
			w.zenyWidget(zeny, editable),
		).
			CrossAlign(primitives.CrossAxisCenter).
			Height(30),
	)
	status := ""
	if editable && w.selfOK {
		status = "OK"
	}
	if !editable && w.otherOK {
		status = "OK"
	}
	if status != "" {
		rows = append(rows, primitives.Box(rotheme.Text(status).Color(rotheme.Default.Colors.MutedText)).Height(18))
	}
	return primitives.Box(rows...).
		Width(tradePanelW).
		Height(tradePanelH + 40).
		Gap(2)
}

func tradeItemRowWidget(item tradeWindowItem) widget.Widget {
	name := ""
	qty := ""
	if item.item.ItemID != 0 {
		name = item.name
		if item.item.Amount > 1 {
			qty = fmt.Sprintf("%d", item.item.Amount)
		}
	}
	return primitives.HBox(
		newTradeIconWidget(item.icon),
		primitives.Box(rotheme.Text(name)).
			Width(166),
		primitives.Box(rotheme.Text(qty).Align(widget.TextAlignRight)).
			Width(42).
			CrossAlign(primitives.CrossAxisCenter),
	).
		CrossAlign(primitives.CrossAxisCenter).
		Height(tradeRowH).
		Background(rotheme.Default.Colors.PanelBody)
}

func (w *TradeWindow) zenyWidget(zeny uint32, editable bool) widget.Widget {
	if !editable {
		return primitives.Box(rotheme.Text(formatTradeZeny(zeny)).Align(widget.TextAlignRight)).
			Width(tradeZenyFieldW).
			CrossAlign(primitives.CrossAxisCenter)
	}
	return primitives.Box(rotheme.TextField(w.zenyInput, textfield.TypeNumber, func(value string) {
		w.zenyInput = value
	}, nil)).
		Width(tradeZenyFieldW).
		Height(tradeZenyFieldH)
}

func newTradeIconWidget(img image.Image) widget.Widget {
	return newStaticImageWidget(img, inventoryIconSize+8, tradeRowH)
}

func (w *TradeWindow) sendPanelX() int {
	return w.x + tradePanelPad
}

func (w *TradeWindow) sendPanelY() int {
	return w.y + ROWindowTitleHeight + tradePanelPad
}

func (w *TradeWindow) conclude(ctx Context) {
	if w.selfOK {
		return
	}
	zeny := parseTradeZeny(w.zenyInput)
	if zeny > 0 && w.sendZeny == 0 {
		if ctx.Network == nil {
			glog.Warnf("trade add zeny failed amount=%d: not connected", zeny)
			return
		}
		if err := ctx.Network.SendTradeAddItem(0, zeny); err != nil {
			glog.Warnf("trade add zeny failed amount=%d: %v", zeny, err)
			return
		}
	}
	if ctx.Network == nil {
		glog.Warnf("trade conclude failed: not connected")
		return
	}
	if err := ctx.Network.SendTradeConclude(); err != nil {
		glog.Warnf("trade conclude failed: %v", err)
	}
}

func (w *TradeWindow) cancel(ctx Context) {
	if ctx.Network != nil && w.IsOpen() {
		if err := ctx.Network.SendTradeCancel(); err != nil {
			glog.Warnf("trade cancel failed: %v", err)
		}
	}
	w.Close(ctx)
}

func (w *TradeWindow) commit(ctx Context) {
	if ctx.Network == nil {
		glog.Warnf("trade commit failed: not connected")
		return
	}
	if err := ctx.Network.SendTradeCommit(); err != nil {
		glog.Warnf("trade commit failed: %v", err)
	}
}

func parseTradeZeny(value string) uint32 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(parsed)
}

func formatTradeZeny(value uint32) string {
	if value == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", value)
}

func tradeItemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
	if manager == nil || item.ItemID == 0 {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(item.ItemID), item.Identified)
	if !ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.ItemIconTextureCandidates(resourceName))
	if err != nil {
		return nil
	}
	return img
}
