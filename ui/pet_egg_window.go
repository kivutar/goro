package ui

import (
	"fmt"
	"image"
	"sort"

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
	petEggWindowWidth  = 312
	petEggTableHeaderH = 24
	petEggRowH         = 32
	petEggRows         = 6
	petEggWindowHeight = ROWindowTitleHeight + petEggTableHeaderH + petEggRows*petEggRowH + ROWindowFooterHeight
)

type PetEggWindow struct {
	Window
	scrollY     state.Signal[float32]
	selectedRow int
	indexes     []uint16
	snapshot    string
	icons       map[identifyItemIconKey]image.Image
	iconMiss    map[identifyItemIconKey]struct{}
}

func (w *PetEggWindow) OpenList(ctx Context, list network.PetEggList) {
	w.EnsureWindow(petEggWindowWidth, petEggWindowHeight)
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

func (w *PetEggWindow) Update(ctx Context) bool {
	w.EnsureWindow(petEggWindowWidth, petEggWindowHeight)
	if !w.IsOpen() {
		return false
	}
	w.ClampScroll(ctx.Session)
	if len(w.items(ctx.Session)) == 0 {
		w.Close()
		w.Publish(ctx)
		return true
	}
	snapshot := w.snapshotString(ctx.Session)
	if snapshot != w.snapshot {
		w.snapshot = snapshot
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *PetEggWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Select Pet Egg"),
		CloseButton(true),
		OnClose(func() {
			w.Close()
			w.Publish(ctx)
		}),
		Size(petEggWindowWidth, petEggWindowHeight),
		Content(
			primitives.Box(w.tableWidget(ctx)).
				Height(petEggTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Cancel", func() {
				w.Close()
				w.Publish(ctx)
			}),
			rotheme.Button("OK", func() {
				w.hatchSelected(ctx)
				w.Publish(ctx)
			}),
		),
	)
}

func (w *PetEggWindow) tableWidget(ctx Context) *rotheme.TableViewWidget {
	items := w.items(ctx.Session)
	return itemTableView(
		w.itemTableRows(ctx, items),
		"Pet Egg",
		petEggRowH,
		petEggTableHeaderH,
		"No items",
		w.ensureScrollSignal(),
		w.selectedRow,
		func(row int) {
			w.selectedRow = row
		},
	)
}

func (w *PetEggWindow) hatchSelected(ctx Context) {
	items := w.items(ctx.Session)
	if w.selectedRow < 0 || w.selectedRow >= len(items) {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("pet egg hatch failed: not connected")
		return
	}
	item := items[w.selectedRow]
	if err := ctx.Network.SendSelectPetEgg(item.Index); err != nil {
		glog.Warnf("pet egg hatch failed: %v", err)
		return
	}
	glog.Debugf("pet egg hatch selected index=%d item=%d", item.Index, item.ItemID)
	w.Close()
}

func (w *PetEggWindow) ClampScroll(s *session.Session) {
	items := w.items(s)
	if w.selectedRow >= len(items) {
		w.selectedRow = -1
	}
	scroll := w.ensureScrollSignal()
	maxScroll := float32(maxInt(0, len(items)-petEggRows) * petEggRowH)
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *PetEggWindow) items(s *session.Session) []session.InventoryItem {
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

func (w *PetEggWindow) itemTableRows(ctx Context, items []session.InventoryItem) []itemTableRow {
	rows := make([]itemTableRow, len(items))
	for i, item := range items {
		rows[i] = itemTableRow{
			name: inventoryItemDisplayName(ctx.Resources, item),
			icon: w.itemIconImage(ctx.Resources, item),
		}
	}
	return rows
}

func (w *PetEggWindow) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
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

func (w *PetEggWindow) markIconMiss(key identifyItemIconKey) {
	if w.iconMiss == nil {
		w.iconMiss = make(map[identifyItemIconKey]struct{})
	}
	w.iconMiss[key] = struct{}{}
}

func (w *PetEggWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func (w *PetEggWindow) snapshotString(s *session.Session) string {
	return fmt.Sprintf("%v:%v", w.indexes, w.items(s))
}

func petEggTableHeight() float32 {
	return petEggTableHeaderH + petEggRows*petEggRowH
}
