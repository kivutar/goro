package ui

import (
	"fmt"
	"image"
	"log"
	"sort"

	"github.com/gogpu/ui/core/datatable"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	identifyWindowWidth   = 312
	identifyWindowFooterH = 38
	identifyTableHeaderH  = 36
	identifyRowH          = 32
	identifyRows          = 6
	identifyWindowHeight  = ROWindowTitleHeight + identifyTableHeaderH + identifyRows*identifyRowH + identifyWindowFooterH
	identifyCancelIndex   = uint16(0xFFFF)
)

type IdentifyWindow struct {
	window      WindowState
	scrollY     state.Signal[float32]
	selectedRow int
	indexes     []uint16
	snapshot    string
	icons       map[identifyItemIconKey]image.Image
	iconMiss    map[identifyItemIconKey]struct{}
}

type identifyItemIconKey struct {
	itemID     uint16
	identified bool
}

type identifyTableRow struct {
	name string
}

func (w *IdentifyWindow) OpenList(ctx Context, list network.ItemIdentifyList) {
	w.ensureWindow()
	w.indexes = append(w.indexes[:0], list.Indexes...)
	w.selectedRow = -1
	w.ensureScrollSignal().Set(0)
	w.ClampScroll(ctx.Session)
	if len(w.items(ctx.Session)) == 0 {
		w.window.Close()
		w.Publish(ctx)
		return
	}
	w.snapshot = w.identifySnapshot(ctx.Session)
	w.window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *IdentifyWindow) ApplyAck(ctx Context, ack network.ItemIdentifyAck) {
	w.ensureWindow()
	if ack.Success {
		w.removeIndex(ack.Index)
		w.ClampScroll(ctx.Session)
		w.selectedRow = -1
		if len(w.items(ctx.Session)) == 0 {
			w.window.Close()
		} else {
			w.snapshot = w.identifySnapshot(ctx.Session)
			w.window.SetContent(w.widgetTree(ctx))
		}
		w.Publish(ctx)
		return
	}
	log.Printf("identify failed index=%d", ack.Index)
}

func (w *IdentifyWindow) IsOpen() bool {
	w.ensureWindow()
	return w.window.IsOpen()
}

func (w *IdentifyWindow) Update(ctx Context) bool {
	w.ensureWindow()
	if !w.window.IsOpen() {
		return false
	}
	if ctx.Input == nil {
		w.Publish(ctx)
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) || ctx.Input.MouseJustPressed(render.MouseButtonRight) {
		w.cancel(ctx)
		w.Publish(ctx)
		return true
	}
	w.ClampScroll(ctx.Session)
	snapshot := w.identifySnapshot(ctx.Session)
	if snapshot != w.snapshot {
		w.snapshot = snapshot
		w.window.SetContent(w.widgetTree(ctx))
	}
	consumed := w.window.Update(ctx)
	if !w.window.IsOpen() {
		w.Publish(ctx)
		return consumed
	}
	w.Publish(ctx)
	return consumed
}

func (w *IdentifyWindow) Draw(screen *render.Image, ctx Context, assets AssetRenderer) {
	w.Publish(ctx)
}

func (w *IdentifyWindow) Publish(ctx Context) {
	w.ensureWindow()
	if !w.window.IsOpen() {
		w.window.Unpublish(ctx)
		return
	}
	w.window.Publish(ctx)
}

func (w *IdentifyWindow) ensureWindow() {
	if w.window.width == 0 {
		w.window = NewWindowState(identifyWindowWidth, identifyWindowHeight)
		w.window.SetCloseOnEscape(false)
		w.selectedRow = -1
	}
}

func (w *IdentifyWindow) widgetTree(ctx Context) widget.Widget {
	return Window(
		Title("Item Appraisal"),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
			w.Publish(ctx)
		}),
		Size(identifyWindowWidth, identifyWindowHeight),
		FooterHeight(identifyWindowFooterH),
		FooterPadding(10),
		Content(
			primitives.Box(w.identifyTableWidget(ctx)).
				Height(identifyTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Cancel", func() {
					w.cancel(ctx)
					w.Publish(ctx)
				}).Width(68),
			).
				CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *IdentifyWindow) identifyTableWidget(ctx Context) *datatable.Widget {
	items := w.items(ctx.Session)
	rows := w.identifyRows(ctx, items)
	return datatable.New(
		datatable.Columns([]datatable.Column{
			{Key: "item", Title: "Item", Width: identifyWindowWidth},
		}),
		datatable.RowCount(len(rows)),
		datatable.RowHeight(identifyRowH),
		datatable.ScrollYSignal(w.ensureScrollSignal()),
		datatable.SelectionModeOpt(datatable.SelectionSingle),
		datatable.SelectedRow(w.selectedRow),
		datatable.PainterOpt(identifyTablePainter{icons: w.identifyItemIcons(ctx, items)}),
		datatable.CellValue(func(row int, col string) string {
			if row < 0 || row >= len(rows) {
				return ""
			}
			return rows[row].name
		}),
		datatable.OnRowSelect(func(row int) {
			if row >= 0 && row < len(rows) {
				w.selectedRow = row
				w.identify(ctx, items[row])
			}
		}),
	)
}

func (w *IdentifyWindow) ClampScroll(s *session.Session) {
	items := w.items(s)
	if w.selectedRow >= len(items) {
		w.selectedRow = -1
	}
	scroll := w.ensureScrollSignal()
	maxScroll := float32(maxInt(0, len(items)-identifyRows) * identifyRowH)
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *IdentifyWindow) identifyRows(ctx Context, items []session.InventoryItem) []identifyTableRow {
	rows := make([]identifyTableRow, len(items))
	for i, item := range items {
		name := inventoryItemDisplayName(ctx.Resources, item)
		if item.Refine > 0 {
			name = fmt.Sprintf("+%d %s", item.Refine, name)
		}
		rows[i] = identifyTableRow{name: name}
	}
	return rows
}

func (w *IdentifyWindow) identifyItemIcons(ctx Context, items []session.InventoryItem) []image.Image {
	icons := make([]image.Image, len(items))
	for i, item := range items {
		icons[i] = w.itemIconImage(ctx.Resources, item)
	}
	return icons
}

func (w *IdentifyWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
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

func (w *IdentifyWindow) markIconMiss(key identifyItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[identifyItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func (w *IdentifyWindow) items(s *session.Session) []session.InventoryItem {
	if s == nil {
		return nil
	}
	items := make([]session.InventoryItem, 0, len(w.indexes))
	for _, index := range w.indexes {
		if item, ok := findInventoryItemByIndex(s, index); ok && !item.Identified && inventoryItemIsEquipment(item) {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}

func (w *IdentifyWindow) identify(ctx Context, item session.InventoryItem) {
	if ctx.Network == nil {
		log.Printf("identify failed: not connected")
		return
	}
	if err := ctx.Network.SendItemIdentify(item.Index); err != nil {
		log.Printf("identify failed: %v", err)
		return
	}
	log.Printf("identify requested index=%d item=%d", item.Index, item.ItemID)
}

func (w *IdentifyWindow) cancel(ctx Context) {
	if ctx.Network != nil {
		if err := ctx.Network.SendItemIdentify(identifyCancelIndex); err != nil {
			log.Printf("identify cancel failed: %v", err)
			return
		}
	}
	w.window.Close()
}

func (w *IdentifyWindow) removeIndex(index uint16) {
	for i, candidate := range w.indexes {
		if candidate == index {
			w.indexes = append(w.indexes[:i], w.indexes[i+1:]...)
			return
		}
	}
}

func (w *IdentifyWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func (w *IdentifyWindow) identifySnapshot(s *session.Session) string {
	return fmt.Sprintf("%v:%v", w.indexes, w.items(s))
}

func identifyTableHeight() float32 {
	return identifyTableHeaderH + identifyRows*identifyRowH
}

func findInventoryItemByIndex(s *session.Session, index uint16) (session.InventoryItem, bool) {
	if s == nil {
		return session.InventoryItem{}, false
	}
	for _, item := range s.Inventory.Items {
		if item.Index == index {
			return item, true
		}
	}
	return session.InventoryItem{}, false
}

type identifyTablePainter struct {
	datatable.DefaultPainter
	icons []image.Image
}

func (p identifyTablePainter) PaintHeader(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderPaintState) {
	if bounds.IsEmpty() {
		return
	}
	canvas.DrawRect(bounds, rotheme.Default.Colors.PanelBody)
}

func (p identifyTablePainter) PaintHeaderCell(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderCellPaintState) {
}

func (p identifyTablePainter) PaintRow(canvas widget.Canvas, s datatable.RowPaintState) {
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

func (p identifyTablePainter) PaintCell(canvas widget.Canvas, s datatable.CellPaintState) {
	textBounds := geometry.NewRect(s.Bounds.Min.X+4, s.Bounds.Min.Y+4, s.Bounds.Width()-8, s.Bounds.Height()-8)
	if s.RowIndex >= 0 && s.RowIndex < len(p.icons) && p.icons[s.RowIndex] != nil {
		icon := p.icons[s.RowIndex]
		iconBounds := icon.Bounds()
		iconW := float32(iconBounds.Dx())
		iconH := float32(iconBounds.Dy())
		canvas.DrawImage(icon, geometry.Pt(s.Bounds.Min.X+6, s.Bounds.Min.Y+(s.Bounds.Height()-iconH)/2))
		textBounds = geometry.NewRect(s.Bounds.Min.X+iconW+12, s.Bounds.Min.Y+4, s.Bounds.Width()-iconW-16, s.Bounds.Height()-8)
	}
	rotheme.DrawText(canvas, s.Value, textBounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, s.Align)
}

func (p identifyTablePainter) PaintEmptyState(canvas widget.Canvas, bounds geometry.Rect) {
	rotheme.DrawText(canvas, "No unidentified equipment", bounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignCenter)
}
