package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

const (
	storageWindowWidth  = 312
	storageWindowHeight = 356
	storageWindowTitleH = 28
	storageWindowPad    = 10
	storageRowH         = 32
)

type StorageWindow struct {
	open          bool
	x             int
	y             int
	positioned    bool
	dragging      bool
	dragDX        int
	dragDY        int
	scroll        int
	status        string
	statusGood    bool
	statusAt      time.Time
	lastClickItem uint16
	lastClickAt   time.Time
}

func (w *StorageWindow) SetOpen(open bool) {
	w.open = open
	if !open {
		w.dragging = false
	}
}

func (w *StorageWindow) OpenWindow(ctx Context) {
	w.open = true
	w.EnsurePosition(ctx)
	w.ClampScroll(ctx.Session)
}

func (w *StorageWindow) Update(ctx Context, itemInfo *ItemInfoWindow) bool {
	if !w.open || ctx.Input == nil {
		return false
	}
	if ctx.Session == nil || !ctx.Session.Storage.Open {
		w.open = false
		w.dragging = false
		return false
	}
	w.EnsurePosition(ctx)
	width, height := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(render.MouseButtonLeft) {
			w.x = clampInventoryWindowInt(ctx.Input.MouseX-w.dragDX, 8, maxInt(8, width-storageWindowWidth-8))
			w.y = clampInventoryWindowInt(ctx.Input.MouseY-w.dragDY, 8, maxInt(8, height-storageWindowHeight-8))
			return true
		}
		w.dragging = false
		return true
	}
	inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, storageWindowWidth, storageWindowHeight)
	if inside && ctx.Input.WheelY != 0 {
		w.scrollBy(ctx.Input.WheelY, ctx.Session)
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.close(ctx)
		return true
	}
	if ctx.Input.MouseJustPressed(render.MouseButtonRight) {
		mx, my := ctx.Input.MouseX, ctx.Input.MouseY
		if !inside {
			return false
		}
		if item, ok := w.itemAt(ctx.Session, mx, my); ok {
			if itemInfo != nil {
				itemInfo.openItem(ctx, item, mx, my)
			}
			return true
		}
		return true
	}
	if !ctx.Input.MouseJustPressed(render.MouseButtonLeft) {
		return inside
	}
	mx, my := ctx.Input.MouseX, ctx.Input.MouseY
	if !inside {
		return false
	}
	cx, cy, cw, ch := w.closeBounds()
	if pointInRect(mx, my, cx, cy, cw, ch) {
		w.close(ctx)
		return true
	}
	if pointInRect(mx, my, w.x, w.y, storageWindowWidth, storageWindowTitleH) {
		w.dragging = true
		w.dragDX = mx - w.x
		w.dragDY = my - w.y
		return true
	}
	if item, ok := w.itemAt(ctx.Session, mx, my); ok {
		now := time.Now()
		if w.lastClickItem == item.Index && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
			w.withdraw(ctx, item)
			w.lastClickItem = 0
			return true
		}
		w.lastClickItem = item.Index
		w.lastClickAt = now
		w.setStatus(inventoryItemDisplayName(ctx.Resources, item), true)
		return true
	}
	return true
}

func (w *StorageWindow) Draw(screen *render.Image, ctx Context, assets AssetRenderer) {
	if !w.open || screen == nil {
		return
	}
	w.EnsurePosition(ctx)
	w.ClampScroll(ctx.Session)
	x, y := w.x, w.y
	DrawTitledWindowFrame(screen, x, y, storageWindowWidth, storageWindowHeight, storageWindowTitleH)
	DrawWindowTitle(screen, x, y, storageWindowTitleH, storageWindowPad, "Storage", inventoryTitleColor)
	cx, cy, cw, ch := w.closeBounds()
	DrawCloseButton(screen, cx, cy, cw, ch, inventoryButtonColor, inventoryTextColor)

	items := sortedStorageItems(ctx.Session)
	if len(items) == 0 {
		render.DebugPrintAtColor(screen, "No items", x+storageWindowPad, y+storageWindowTitleH+18, inventoryMutedColor)
	} else {
		mx, my := -1, -1
		if ctx.Input != nil {
			mx, my = ctx.Input.MouseX, ctx.Input.MouseY
		}
		for row, item := range visibleStorageItems(items, w.scroll) {
			rx, ry, rw, rh := w.rowBounds(row)
			fill := PanelAltColor
			if pointInRect(mx, my, rx, ry, rw, rh) {
				fill = inventoryHoverColor
			}
			DrawSurface(screen, rx, ry, rw, rh, fill, WindowBorderColor)
			if assets != nil {
				assets.DrawInventoryItemIcon(screen, ctx.Resources, item, rx+3, ry+3)
			}
			name := inventoryItemDisplayName(ctx.Resources, item)
			if item.Refine > 0 {
				name = fmt.Sprintf("+%d %s", item.Refine, name)
			}
			render.DebugPrintAtColor(screen, trimRunes(name, 28), rx+inventoryIconSize+10, ry+5, inventoryTextColor)
			render.DebugPrintAtColor(screen, fmt.Sprintf("x%d", item.Amount), rx+rw-42, ry+5, inventoryMutedColor)
		}
		w.drawScrollBar(screen, len(items))
	}
	if ctx.Session != nil {
		storage := ctx.Session.Storage
		render.DebugPrintAtColor(screen, fmt.Sprintf("Num:%d/%d", storage.Amount, storage.MaxAmount), x+storageWindowPad, y+storageWindowHeight-22, inventoryMutedColor)
	}
	if w.status != "" && time.Since(w.statusAt) < 2200*time.Millisecond {
		statusColor := inventoryMutedColor
		if !w.statusGood {
			statusColor = ErrorTextColor
		}
		render.DebugPrintAtColor(screen, trimRunes(w.status, 34), x+storageWindowPad+92, y+storageWindowHeight-22, statusColor)
	}
}

func (w *StorageWindow) CursorAction(ctx Context) (int, bool) {
	if !w.open || ctx.Input == nil {
		return 0, false
	}
	if pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, storageWindowWidth, storageWindowHeight) {
		return CursorActionClick, true
	}
	return 0, false
}

func (w *StorageWindow) AcceptInventoryDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	if !w.open || !pointInRect(mx, my, w.x, w.y, storageWindowWidth, storageWindowHeight) {
		return false
	}
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		w.setStatus("Not connected", false)
		return true
	}
	if err := ctx.Network.SendMoveToStorage(item.Index, amount); err != nil {
		w.setStatus(err.Error(), false)
		return true
	}
	w.setStatus("Store requested", true)
	return true
}

func (w *StorageWindow) close(ctx Context) {
	if ctx.Network != nil {
		if err := ctx.Network.SendCloseStorage(); err != nil {
			w.setStatus(err.Error(), false)
			return
		}
	}
	w.open = false
	if ctx.Session != nil {
		ctx.Session.Storage.Open = false
	}
}

func (w *StorageWindow) withdraw(ctx Context, item session.InventoryItem) {
	amount := uint32(item.Amount)
	if amount == 0 {
		amount = 1
	}
	if ctx.Network == nil {
		w.setStatus("Not connected", false)
		return
	}
	if err := ctx.Network.SendMoveFromStorage(item.Index, amount); err != nil {
		w.setStatus(err.Error(), false)
		return
	}
	w.setStatus("Withdraw requested", true)
}

func (w *StorageWindow) setStatus(text string, good bool) {
	w.status = text
	w.statusGood = good
	w.statusAt = time.Now()
}

func (w *StorageWindow) EnsurePosition(ctx Context) {
	if w.positioned {
		return
	}
	width, _ := ctx.ScreenSize()
	w.x = maxInt(8, width-storageWindowWidth-24)
	w.y = 118
	w.positioned = true
}

func (w *StorageWindow) closeBounds() (int, int, int, int) {
	return w.x + storageWindowWidth - 24, w.y + 7, IconButtonSize, IconButtonSize
}

func (w *StorageWindow) rowBounds(row int) (int, int, int, int) {
	x := w.x + storageWindowPad
	y := w.y + storageWindowTitleH + 10 + row*storageRowH
	return x, y, storageWindowWidth - storageWindowPad*2 - 8, storageRowH - 4
}

func (w *StorageWindow) itemAt(s *session.Session, mx, my int) (session.InventoryItem, bool) {
	items := visibleStorageItems(sortedStorageItems(s), w.scroll)
	for row, item := range items {
		x, y, width, height := w.rowBounds(row)
		if pointInRect(mx, my, x, y, width, height) {
			return item, true
		}
	}
	return session.InventoryItem{}, false
}

func (w *StorageWindow) scrollBy(wheelY float64, s *session.Session) {
	if wheelY > 0 {
		w.scroll--
	} else if wheelY < 0 {
		w.scroll++
	}
	w.ClampScroll(s)
}

func (w *StorageWindow) ClampScroll(s *session.Session) {
	maxScroll := maxInt(0, len(sortedStorageItems(s))-visibleStorageRows())
	if w.scroll < 0 {
		w.scroll = 0
	}
	if w.scroll > maxScroll {
		w.scroll = maxScroll
	}
}

func (w *StorageWindow) drawScrollBar(screen *render.Image, total int) {
	visible := visibleStorageRows()
	if total <= visible {
		return
	}
	trackX := w.x + storageWindowWidth - 14
	trackY := w.y + storageWindowTitleH + 10
	trackH := visible*storageRowH - 4
	render.DrawRect(screen, float64(trackX), float64(trackY), 4, float64(trackH), PanelAltColor)
	maxScroll := maxInt(1, total-visible)
	thumbH := maxInt(18, trackH*visible/total)
	thumbTravel := trackH - thumbH
	thumbY := trackY + thumbTravel*w.scroll/maxScroll
	render.DrawRect(screen, float64(trackX), float64(thumbY), 4, float64(thumbH), inventoryMutedColor)
}

func visibleStorageRows() int {
	return (storageWindowHeight - storageWindowTitleH - 44) / storageRowH
}

func visibleStorageItems(items []session.InventoryItem, scroll int) []session.InventoryItem {
	if scroll < 0 {
		scroll = 0
	}
	if scroll >= len(items) {
		return nil
	}
	end := minInt(len(items), scroll+visibleStorageRows())
	return items[scroll:end]
}

func sortedStorageItems(s *session.Session) []session.InventoryItem {
	if s == nil || len(s.Storage.Items) == 0 {
		return nil
	}
	items := append([]session.InventoryItem(nil), s.Storage.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}
