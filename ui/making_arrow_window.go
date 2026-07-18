package ui

import (
	"image"
	"time"

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
	makingArrowWindowWidth  = 312
	makingArrowTableHeaderH = 24
	makingArrowRowH         = 32
	makingArrowRows         = 6
	makingArrowWindowHeight = ROWindowTitleHeight + makingArrowTableHeaderH + makingArrowRows*makingArrowRowH + ROWindowFooterHeight
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
		Content(
			primitives.Box(w.tableWidget(ctx)).
				Height(makingArrowTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Cancel", func() {
				w.cancel(ctx)
				w.Publish(ctx)
			}),
			rotheme.Button("OK", func() {
				w.confirm(ctx)
				w.Publish(ctx)
			}),
		),
	)
}

func (w *MakingArrowWindow) tableWidget(ctx Context) *rotheme.TableViewWidget {
	ids := append([]uint16(nil), w.itemIDs...)
	return itemTableView(
		w.itemTableRows(ctx, ids),
		"Item",
		makingArrowRowH,
		makingArrowTableHeaderH,
		"No arrow materials",
		w.ensureScrollSignal(),
		w.selectedRow,
		func(row int) {
			w.selectedRow = row
		},
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
	if !pointInRect(mouseX, mouseY, tableX, rowY, scrollbarSafeIntWidth(makingArrowWindowWidth), makingArrowRows*makingArrowRowH) {
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

func (w *MakingArrowWindow) itemTableRows(ctx Context, itemIDs []uint16) []itemTableRow {
	rows := make([]itemTableRow, len(itemIDs))
	for i, itemID := range itemIDs {
		rows[i] = itemTableRow{
			name: inventoryItemDisplayName(ctx.Resources, session.InventoryItem{ItemID: itemID, Identified: true}),
			icon: w.itemIconImage(ctx.Resources, itemID),
		}
	}
	return rows
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
