package ui

import (
	"image"
	"time"

	"github.com/gogpu/ui/core/datatable"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	makingArrowWindowWidth   = 312
	makingArrowWindowFooterH = 38
	makingArrowTableHeaderH  = 24
	makingArrowRowH          = 32
	makingArrowRows          = 6
	makingArrowWindowHeight  = ROWindowTitleHeight + makingArrowTableHeaderH + makingArrowRows*makingArrowRowH + makingArrowWindowFooterH
)

type MakingArrowWindow struct {
	Window
	scrollY      state.Signal[float32]
	selectedRow  int
	itemIDs      []uint16
	lastClickAt  time.Time
	lastClickRow int
	icons        map[identifyItemIconKey]image.Image
	iconMiss     map[identifyItemIconKey]struct{}
}

func (w *MakingArrowWindow) OpenList(ctx Context, list network.MakingArrowList) {
	w.EnsureWindow(makingArrowWindowWidth, makingArrowWindowHeight)
	w.itemIDs = append(w.itemIDs[:0], list.ItemIDs...)
	w.selectedRow = 0
	w.lastClickRow = -1
	w.lastClickAt = time.Time{}
	w.ensureScrollSignal().Set(0)
	w.clampScroll()
	if len(w.itemIDs) == 0 {
		w.Close()
		w.Publish(ctx)
		return
	}
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *MakingArrowWindow) Update(ctx Context) bool {
	w.EnsureWindow(makingArrowWindowWidth, makingArrowWindowHeight)
	if !w.IsOpen() {
		return false
	}
	w.clampScroll()
	consumed := w.Window.Update(ctx)
	if w.IsOpen() {
		w.updateDoubleClick(ctx)
	}
	w.Publish(ctx)
	return consumed
}

func (w *MakingArrowWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Arrow Crafting"),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
			w.Publish(ctx)
		}),
		Size(makingArrowWindowWidth, makingArrowWindowHeight),
		FooterHeight(makingArrowWindowFooterH),
		FooterPadding(10),
		Content(
			primitives.Box(w.tableWidget(ctx)).
				Height(makingArrowTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Cancel", func() {
					w.cancel(ctx)
					w.Publish(ctx)
				}).Width(68),
				rotheme.Button("OK", func() {
					w.confirm(ctx)
					w.Publish(ctx)
				}).Width(56),
			).
				Gap(8).
				CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *MakingArrowWindow) tableWidget(ctx Context) *datatable.Widget {
	ids := append([]uint16(nil), w.itemIDs...)
	return datatable.New(
		datatable.Columns([]datatable.Column{
			{Key: "item", Title: "Item", Width: makingArrowWindowWidth},
		}),
		datatable.RowCount(len(ids)),
		datatable.RowHeight(makingArrowRowH),
		datatable.ScrollYSignal(w.ensureScrollSignal()),
		datatable.SelectionModeOpt(datatable.SelectionSingle),
		datatable.SelectedRow(w.selectedRow),
		datatable.PainterOpt(makingArrowTablePainter{icons: w.itemIcons(ctx, ids)}),
		datatable.CellValue(func(row int, col string) string {
			if row < 0 || row >= len(ids) {
				return ""
			}
			return inventoryItemDisplayName(ctx.Resources, session.InventoryItem{ItemID: ids[row], Identified: true})
		}),
		datatable.OnRowSelect(func(row int) {
			if row >= 0 && row < len(ids) {
				w.selectedRow = row
			}
		}),
	)
}

func (w *MakingArrowWindow) confirm(ctx Context) {
	if w.selectedRow < 0 || w.selectedRow >= len(w.itemIDs) {
		return
	}
	w.makeArrow(ctx, w.itemIDs[w.selectedRow])
}

func (w *MakingArrowWindow) updateDoubleClick(ctx Context) {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return
	}
	row, ok := w.rowAtMouse(ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok {
		return
	}
	now := time.Now()
	if w.lastClickRow == row && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.selectedRow = row
		w.lastClickRow = -1
		w.lastClickAt = time.Time{}
		w.confirm(ctx)
		return
	}
	w.lastClickRow = row
	w.lastClickAt = now
}

func (w *MakingArrowWindow) rowAtMouse(mouseX, mouseY int) (int, bool) {
	tableX := w.x
	tableY := w.y + ROWindowTitleHeight
	rowY := tableY + makingArrowTableHeaderH
	if !pointInRect(mouseX, mouseY, tableX, rowY, makingArrowWindowWidth, makingArrowRows*makingArrowRowH) {
		return 0, false
	}
	row := int((float32(mouseY-rowY) + w.ensureScrollSignal().Get()) / makingArrowRowH)
	if row < 0 || row >= len(w.itemIDs) {
		return 0, false
	}
	return row, true
}

func (w *MakingArrowWindow) makeArrow(ctx Context, itemID uint16) {
	if ctx.Network == nil {
		glog.Warnf("making arrow failed: not connected")
		return
	}
	if err := ctx.Network.SendMakingArrow(itemID); err != nil {
		glog.Warnf("making arrow failed: %v", err)
		return
	}
	glog.Debugf("making arrow requested item=%d", itemID)
	w.Close()
}

func (w *MakingArrowWindow) cancel(ctx Context) {
	if ctx.Network != nil {
		if err := ctx.Network.SendMakingArrow(0xFFFF); err != nil {
			glog.Warnf("making arrow cancel failed: %v", err)
		}
	}
	w.Close()
}

func (w *MakingArrowWindow) itemIcons(ctx Context, itemIDs []uint16) []image.Image {
	icons := make([]image.Image, len(itemIDs))
	for i, itemID := range itemIDs {
		icons[i] = w.itemIconImage(ctx.Resources, itemID)
	}
	return icons
}

func (w *MakingArrowWindow) itemIconImage(manager *res.Manager, itemID uint16) image.Image {
	if manager == nil || itemID == 0 {
		return nil
	}
	key := identifyItemIconKey{itemID: itemID, identified: true}
	if w.icons != nil {
		if img := w.icons[key]; img != nil {
			return img
		}
	}
	if _, ok := w.iconMiss[key]; ok {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(itemID), true)
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

func (w *MakingArrowWindow) markIconMiss(key identifyItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[identifyItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func (w *MakingArrowWindow) clampScroll() {
	if w.selectedRow >= len(w.itemIDs) {
		w.selectedRow = -1
	}
	scroll := w.ensureScrollSignal()
	maxScroll := float32(maxInt(0, len(w.itemIDs)-makingArrowRows) * makingArrowRowH)
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *MakingArrowWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func makingArrowTableHeight() float32 {
	return makingArrowTableHeaderH + makingArrowRows*makingArrowRowH
}

type makingArrowTablePainter struct {
	datatable.DefaultPainter
	icons []image.Image
}

func (p makingArrowTablePainter) PaintHeader(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderPaintState) {
	if bounds.IsEmpty() {
		return
	}
	canvas.DrawRect(bounds, rotheme.Default.Colors.PanelBody)
}

func (p makingArrowTablePainter) PaintHeaderCell(canvas widget.Canvas, bounds geometry.Rect, s datatable.HeaderCellPaintState) {
}

func (p makingArrowTablePainter) PaintRow(canvas widget.Canvas, s datatable.RowPaintState) {
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

func (p makingArrowTablePainter) PaintCell(canvas widget.Canvas, s datatable.CellPaintState) {
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

func (p makingArrowTablePainter) PaintEmptyState(canvas widget.Canvas, bounds geometry.Rect) {
	rotheme.DrawText(canvas, "No arrow materials", bounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignCenter)
}
