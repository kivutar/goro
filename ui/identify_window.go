package ui

import (
	"fmt"
	"image"
	"slices"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	identifyWindowWidth  = 312
	identifyTableHeaderH = 36
	identifyRowH         = 32
	identifyRows         = 6
	identifyWindowHeight = ROWindowTitleHeight + identifyTableHeaderH + identifyRows*identifyRowH + ROWindowFooterHeight
	identifyCancelIndex  = uint16(0xFFFF)
)

type IdentifyWindow struct {
	Window
	scrollY  state.Signal[float32]
	selected session.InventoryItem
	indexes  []uint16
	snapshot []session.InventoryItem
	itemList itemDialogList
	icons    map[identifyItemIconKey]image.Image
	iconMiss map[identifyItemIconKey]struct{}
}

type identifyItemIconKey struct {
	itemID     uint16
	identified bool
}

func (w *IdentifyWindow) OpenList(ctx Context, list network.ItemIdentifyList) {
	w.EnsureWindow(identifyWindowWidth, identifyWindowHeight)
	w.indexes = append(w.indexes[:0], list.Indexes...)
	w.selected = session.InventoryItem{}
	w.ensureScrollSignal().Set(0)
	w.ClampScroll(ctx.Session)
	if len(w.items(ctx.Session)) == 0 {
		w.Close()
		w.Publish(ctx)
		return
	}
	w.snapshot = w.items(ctx.Session)
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *IdentifyWindow) Update(ctx Context) bool {
	w.EnsureWindow(identifyWindowWidth, identifyWindowHeight)
	if !w.IsOpen() {
		return false
	}
	if ctx.Input == nil {
		w.Publish(ctx)
		return true
	}
	w.ClampScroll(ctx.Session)
	snapshot := w.items(ctx.Session)
	if !slices.Equal(snapshot, w.snapshot) {
		w.snapshot = snapshot
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *IdentifyWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Item Appraisal"),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
			w.Publish(ctx)
		}),
		Size(identifyWindowWidth, identifyWindowHeight),
		Content(
			primitives.Box(w.identifyTableWidget(ctx)).
				Height(identifyTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Cancel", func() {
				w.cancel(ctx)
				w.Publish(ctx)
			}),
			rotheme.Button("OK", func() {
				w.identifySelected(ctx)
				w.Publish(ctx)
			}),
		),
	)
}

func (w *IdentifyWindow) identifyTableWidget(ctx Context) *rotheme.TableViewWidget {
	items := w.items(ctx.Session)
	return itemTableView(
		w.identifyTableRows(ctx, items),
		"Item",
		identifyRowH,
		identifyTableHeaderH,
		"No unidentified equipment",
		w.ensureScrollSignal(),
		w.selectedRow(items),
		func(row int) {
			w.selected = items[row]
		},
	)
}

func (w *IdentifyWindow) ClampScroll(s *session.Session) {
	items := w.items(s)
	if w.selectedRow(items) < 0 {
		w.selected = session.InventoryItem{}
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

func (w *IdentifyWindow) selectedRow(items []session.InventoryItem) int {
	if w.selected.Index == 0 {
		return -1
	}
	for row, item := range items {
		// The displayed item may have been identified, removed or replaced.
		if item == w.selected {
			return row
		}
	}
	return -1
}

func (w *IdentifyWindow) identifyTableRows(ctx Context, items []session.InventoryItem) []itemTableRow {
	rows := make([]itemTableRow, len(items))
	for i, item := range items {
		name := inventoryItemDisplayName(ctx.Resources, item)
		if item.Refine > 0 {
			name = fmt.Sprintf("+%d %s", item.Refine, name)
		}
		rows[i] = itemTableRow{
			name: name,
			icon: w.itemIconImage(ctx.Resources, item),
		}
	}
	return rows
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
	return w.itemList.get(s, w.indexes, true)
}

func (w *IdentifyWindow) identifySelected(ctx Context) {
	if !w.IsOpen() {
		return
	}
	items := w.items(ctx.Session)
	if w.selectedRow(items) < 0 {
		w.selected = session.InventoryItem{}
		return
	}
	item := w.selected
	if ctx.Network == nil {
		glog.Warnf("identify failed: not connected")
		return
	}
	if err := ctx.Network.SendItemIdentify(item.Index); err != nil {
		glog.Warnf("identify failed: %v", err)
		return
	}
	w.Close()
	glog.Debugf("identify requested index=%d item=%d", item.Index, item.ItemID)
}

func (w *IdentifyWindow) cancel(ctx Context) {
	if !w.IsOpen() {
		return
	}
	if ctx.Network != nil {
		if err := ctx.Network.SendItemIdentify(identifyCancelIndex); err != nil {
			glog.Warnf("identify cancel failed: %v", err)
			return
		}
	}
	w.Close()
}

func (w *IdentifyWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
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
