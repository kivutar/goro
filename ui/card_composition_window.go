package ui

import (
	"fmt"
	"image"
	"sort"

	"github.com/gogpu/ui/core/datatable"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	cardCompositionWindowWidth   = 328
	cardCompositionWindowFooterH = 38
	cardCompositionTableHeaderH  = 24
	cardCompositionRowH          = 32
	cardCompositionRows          = 6
	cardCompositionWindowHeight  = ROWindowTitleHeight + cardCompositionTableHeaderH + cardCompositionRows*cardCompositionRowH + cardCompositionWindowFooterH
)

type CardCompositionWindow struct {
	Window
	scrollY     state.Signal[float32]
	selectedRow int
	cardIndex   uint16
	indexes     []uint16
	snapshot    string
	icons       map[identifyItemIconKey]image.Image
	iconMiss    map[identifyItemIconKey]struct{}
}

func (w *CardCompositionWindow) OpenList(ctx Context, cardIndex uint16, list network.ItemCompositionList) {
	w.EnsureWindow(cardCompositionWindowWidth, cardCompositionWindowHeight)
	w.cardIndex = cardIndex
	w.indexes = append(w.indexes[:0], list.Indexes...)
	w.selectedRow = -1
	w.ensureScrollSignal().Set(0)
	w.ClampScroll(ctx.Session)
	if len(w.items(ctx.Session)) == 0 {
		w.Close()
		w.Publish(ctx)
		return
	}
	w.snapshot = w.snapshotString(ctx.Session)
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *CardCompositionWindow) ApplyAck(ctx Context, ack network.ItemCompositionAck) {
	w.EnsureWindow(cardCompositionWindowWidth, cardCompositionWindowHeight)
	if !ack.Success {
		glog.Warnf("card composition failed card_index=%d equip_index=%d", ack.CardIndex, ack.EquipIndex)
		w.Close()
		w.Publish(ctx)
		return
	}
	w.Close()
	w.Publish(ctx)
}

func (w *CardCompositionWindow) Update(ctx Context) bool {
	w.EnsureWindow(cardCompositionWindowWidth, cardCompositionWindowHeight)
	if !w.IsOpen() {
		return false
	}
	w.ClampScroll(ctx.Session)
	snapshot := w.snapshotString(ctx.Session)
	if snapshot != w.snapshot {
		w.snapshot = snapshot
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *CardCompositionWindow) widgetTree(ctx Context) widget.Widget {
	cardName := inventoryItemDisplayName(ctx.Resources, session.InventoryItem{ItemID: w.cardItemID(ctx.Session), Type: db.ItemTypeCard, Identified: true})
	title := "Insert Card"
	if w.cardIndex != 0 && cardName != "" {
		title = fmt.Sprintf("Insert Card (%s)", cardName)
	}
	return Win(
		Title(title),
		CloseButton(true),
		OnClose(func() {
			w.Close()
			w.Publish(ctx)
		}),
		Size(cardCompositionWindowWidth, cardCompositionWindowHeight),
		FooterHeight(cardCompositionWindowFooterH),
		FooterPadding(10),
		Content(
			primitives.Box(w.tableWidget(ctx)).
				Height(cardCompositionTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Cancel", func() {
					w.Close()
					w.Publish(ctx)
				}).Width(68),
				rotheme.Button("OK", func() {
					w.composeSelected(ctx)
				}).Width(56),
			).
				Gap(8).
				CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *CardCompositionWindow) tableWidget(ctx Context) *datatable.Widget {
	items := w.items(ctx.Session)
	return datatable.New(
		datatable.Columns([]datatable.Column{
			{Key: "item", Title: "Item", Width: cardCompositionWindowWidth},
		}),
		datatable.RowCount(len(items)),
		datatable.RowHeight(cardCompositionRowH),
		datatable.ScrollYSignal(w.ensureScrollSignal()),
		datatable.SelectionModeOpt(datatable.SelectionSingle),
		datatable.SelectedRow(w.selectedRow),
		datatable.PainterOpt(cardCompositionTablePainter{icons: w.itemIcons(ctx, items)}),
		datatable.CellValue(func(row int, col string) string {
			if row < 0 || row >= len(items) {
				return ""
			}
			return inventoryItemDisplayName(ctx.Resources, items[row])
		}),
		datatable.OnRowSelect(func(row int) {
			w.selectedRow = row
		}),
	)
}

func (w *CardCompositionWindow) composeSelected(ctx Context) {
	items := w.items(ctx.Session)
	if w.selectedRow < 0 || w.selectedRow >= len(items) {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("card composition failed: not connected")
		return
	}
	equipIndex := items[w.selectedRow].Index
	if err := ctx.Network.SendItemComposition(w.cardIndex, equipIndex); err != nil {
		glog.Warnf("card composition failed: %v", err)
	}
}

func (w *CardCompositionWindow) ClampScroll(s *session.Session) {
	items := w.items(s)
	if w.selectedRow >= len(items) {
		w.selectedRow = -1
	}
	scroll := w.ensureScrollSignal()
	maxScroll := float32(maxInt(0, len(items)-cardCompositionRows) * cardCompositionRowH)
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *CardCompositionWindow) items(s *session.Session) []session.InventoryItem {
	if s == nil {
		return nil
	}
	items := make([]session.InventoryItem, 0, len(w.indexes))
	for _, index := range w.indexes {
		if item, ok := findInventoryItemByIndex(s, index); ok {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}

func (w *CardCompositionWindow) cardItemID(s *session.Session) uint16 {
	if item, ok := findInventoryItemByIndex(s, w.cardIndex); ok {
		return item.ItemID
	}
	return 0
}

func (w *CardCompositionWindow) itemIcons(ctx Context, items []session.InventoryItem) []image.Image {
	icons := make([]image.Image, len(items))
	for i, item := range items {
		icons[i] = w.itemIconImage(ctx.Resources, item)
	}
	return icons
}

func (w *CardCompositionWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
	if manager == nil || item.ItemID == 0 {
		return nil
	}
	key := identifyItemIconKey{itemID: item.ItemID, identified: item.Identified}
	if w.icons != nil {
		if img := w.icons[key]; img != nil {
			return img
		}
	}
	if _, ok := w.iconMiss[key]; ok {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(item.ItemID), item.Identified)
	if !ok {
		w.markIconMiss(key)
		return nil
	}
	img, _, err := res.LoadImage(manager, res.ItemIconTextureCandidates(resourceName))
	if err != nil {
		w.markIconMiss(key)
		return nil
	}
	if w.icons == nil {
		w.icons = make(map[identifyItemIconKey]image.Image)
	}
	w.icons[key] = img
	return img
}

func (w *CardCompositionWindow) markIconMiss(key identifyItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[identifyItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func (w *CardCompositionWindow) snapshotString(s *session.Session) string {
	return fmt.Sprintf("%d:%v:%v", w.cardIndex, w.indexes, w.items(s))
}

func (w *CardCompositionWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func cardCompositionTableHeight() float32 {
	return cardCompositionTableHeaderH + cardCompositionRows*cardCompositionRowH
}

type cardCompositionTablePainter struct {
	datatable.DefaultPainter
	icons []image.Image
}

func (p cardCompositionTablePainter) PaintHeader(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderPaintState) {
	if bounds.IsEmpty() {
		return
	}
	canvas.DrawRect(bounds, rotheme.Default.Colors.PanelBody)
}

func (p cardCompositionTablePainter) PaintHeaderCell(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderCellPaintState) {
}

func (p cardCompositionTablePainter) PaintRow(canvas widget.Canvas, s datatable.RowPaintState) {
	fill := widget.RGBA8(246, 249, 253, 255)
	if s.RowIndex%2 == 1 {
		fill = rotheme.Default.Colors.PanelBody
	}
	if s.Hovered {
		fill = rotheme.Default.Colors.ButtonHover
	}
	if s.Selected {
		fill = rotheme.Default.Colors.ButtonDown
	}
	canvas.DrawRect(s.Bounds, fill)
}

func (p cardCompositionTablePainter) PaintCell(canvas widget.Canvas, s datatable.CellPaintState) {
	textBounds := geometry.NewRect(s.Bounds.Min.X+34, s.Bounds.Min.Y+4, s.Bounds.Width()-38, s.Bounds.Height()-8)
	if s.RowIndex >= 0 && s.RowIndex < len(p.icons) && p.icons[s.RowIndex] != nil {
		canvas.DrawImage(p.icons[s.RowIndex], geometry.Pt(s.Bounds.Min.X+5, s.Bounds.Min.Y+4))
	}
	rotheme.DrawText(canvas, s.Value, textBounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignLeft)
}
