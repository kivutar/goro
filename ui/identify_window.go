package ui

import (
	"fmt"
	"time"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

const (
	identifyWindowWidth  = 312
	identifyWindowHeight = 256
	identifyWindowTitleH = 28
	identifyWindowPad    = 10
	identifyRowH         = 32
	identifyCancelIndex  = uint16(0xFFFF)
)

type IdentifyWindow struct {
	open       bool
	x          int
	y          int
	positioned bool
	dragging   bool
	dragDX     int
	dragDY     int
	scroll     int
	indexes    []uint16
	status     string
	statusGood bool
	statusAt   time.Time
}

func (w *IdentifyWindow) OpenList(ctx Context, list network.ItemIdentifyList) {
	w.open = len(list.Indexes) > 0
	w.indexes = append(w.indexes[:0], list.Indexes...)
	w.scroll = 0
	w.status = ""
	w.statusGood = false
	w.statusAt = time.Time{}
	if w.open {
		w.EnsurePosition(ctx)
		w.ClampScroll(ctx.Session)
	}
}

func (w *IdentifyWindow) ApplyAck(ctx Context, ack network.ItemIdentifyAck) {
	if ack.Success {
		w.setStatus("Item identified", true)
		w.removeIndex(ack.Index)
		w.ClampScroll(ctx.Session)
		if len(w.items(ctx.Session)) == 0 {
			w.open = false
		}
		return
	}
	w.setStatus("Identify failed", false)
}

func (w *IdentifyWindow) IsOpen() bool {
	return w.open
}

func (w *IdentifyWindow) Update(ctx Context) bool {
	if !w.open || ctx.Input == nil {
		return w.open
	}
	w.EnsurePosition(ctx)
	w.ClampScroll(ctx.Session)
	width, height := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(render.MouseButtonLeft) {
			w.x = clampInventoryWindowInt(ctx.Input.MouseX-w.dragDX, 8, maxInt(8, width-identifyWindowWidth-8))
			w.y = clampInventoryWindowInt(ctx.Input.MouseY-w.dragDY, 8, maxInt(8, height-identifyWindowHeight-8))
			return true
		}
		w.dragging = false
		return true
	}
	inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, identifyWindowWidth, identifyWindowHeight)
	if inside && ctx.Input.WheelY != 0 {
		w.scrollBy(ctx.Input.WheelY, ctx.Session)
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) || ctx.Input.MouseJustPressed(render.MouseButtonRight) {
		w.cancel(ctx)
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
		w.cancel(ctx)
		return true
	}
	if pointInRect(mx, my, w.x, w.y, identifyWindowWidth, identifyWindowTitleH) {
		w.dragging = true
		w.dragDX = mx - w.x
		w.dragDY = my - w.y
		return true
	}
	for row, item := range w.visibleItems(ctx.Session) {
		rx, ry, rw, rh := w.rowBounds(row)
		if pointInRect(mx, my, rx, ry, rw, rh) {
			w.identify(ctx, item)
			return true
		}
	}
	return true
}

func (w *IdentifyWindow) Draw(screen *render.Image, ctx Context, assets AssetRenderer) {
	if !w.open || screen == nil {
		return
	}
	w.EnsurePosition(ctx)
	w.ClampScroll(ctx.Session)
	x, y := w.x, w.y
	DrawTitledWindowFrame(screen, x, y, identifyWindowWidth, identifyWindowHeight, identifyWindowTitleH)
	DrawWindowTitle(screen, x, y, identifyWindowTitleH, identifyWindowPad, "Item Appraisal", inventoryTitleColor)
	cx, cy, cw, ch := w.closeBounds()
	DrawCloseButton(screen, cx, cy, cw, ch, inventoryButtonColor, inventoryTextColor)

	items := w.items(ctx.Session)
	if len(items) == 0 {
		render.DebugPrintAtColor(screen, "No unidentified equipment", x+identifyWindowPad, y+identifyWindowTitleH+18, inventoryMutedColor)
	} else {
		mx, my := -1, -1
		if ctx.Input != nil {
			mx, my = ctx.Input.MouseX, ctx.Input.MouseY
		}
		for row, item := range w.visibleItems(ctx.Session) {
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
		}
		w.drawScrollBar(screen, len(items))
	}
	if w.status != "" && time.Since(w.statusAt) < 2200*time.Millisecond {
		statusColor := inventoryMutedColor
		if !w.statusGood {
			statusColor = ErrorTextColor
		}
		render.DebugPrintAtColor(screen, trimRunes(w.status, 34), x+identifyWindowPad, y+identifyWindowHeight-20, statusColor)
	}
}

func (w *IdentifyWindow) EnsurePosition(ctx Context) {
	if w.positioned {
		return
	}
	width, height := ctx.ScreenSize()
	w.x = maxInt(8, (width-identifyWindowWidth)/2)
	w.y = maxInt(8, (height-identifyWindowHeight)/2)
	w.positioned = true
}

func (w *IdentifyWindow) ClampScroll(s *session.Session) {
	w.scroll = clampInventoryWindowInt(w.scroll, 0, w.maxScroll(s))
}

func (w *IdentifyWindow) closeBounds() (int, int, int, int) {
	return w.x + identifyWindowWidth - 24, w.y + 7, IconButtonSize, IconButtonSize
}

func (w *IdentifyWindow) rowBounds(row int) (int, int, int, int) {
	x := w.x + identifyWindowPad
	y := w.y + identifyWindowTitleH + 10 + row*identifyRowH
	return x, y, identifyWindowWidth - identifyWindowPad*2 - 8, identifyRowH - 4
}

func (w *IdentifyWindow) visibleRows() int {
	return (identifyWindowHeight - identifyWindowTitleH - 44) / identifyRowH
}

func (w *IdentifyWindow) visibleItems(s *session.Session) []session.InventoryItem {
	items := w.items(s)
	rows := w.visibleRows()
	start := minInt(w.scroll, maxInt(0, len(items)-rows))
	end := minInt(len(items), start+rows)
	if start >= end {
		return nil
	}
	return items[start:end]
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
	return items
}

func (w *IdentifyWindow) maxScroll(s *session.Session) int {
	return maxInt(0, len(w.items(s))-w.visibleRows())
}

func (w *IdentifyWindow) scrollBy(wheelY float64, s *session.Session) {
	if wheelY > 0 {
		w.scroll--
	} else if wheelY < 0 {
		w.scroll++
	}
	w.ClampScroll(s)
}

func (w *IdentifyWindow) drawScrollBar(screen *render.Image, total int) {
	rows := w.visibleRows()
	if total <= rows {
		return
	}
	trackX := w.x + identifyWindowWidth - 14
	trackY := w.y + identifyWindowTitleH + 10
	trackH := rows*identifyRowH - 4
	render.DrawRect(screen, float64(trackX), float64(trackY), 4, float64(trackH), PanelAltColor)
	maxScroll := maxInt(1, total-rows)
	thumbH := maxInt(18, trackH*rows/total)
	thumbTravel := trackH - thumbH
	thumbY := trackY + thumbTravel*w.scroll/maxScroll
	render.DrawRect(screen, float64(trackX), float64(thumbY), 4, float64(thumbH), inventoryMutedColor)
}

func (w *IdentifyWindow) identify(ctx Context, item session.InventoryItem) {
	if ctx.Network == nil {
		w.setStatus("Not connected", false)
		return
	}
	if err := ctx.Network.SendItemIdentify(item.Index); err != nil {
		w.setStatus(err.Error(), false)
		return
	}
	w.setStatus("Identify requested", true)
}

func (w *IdentifyWindow) cancel(ctx Context) {
	if ctx.Network != nil {
		if err := ctx.Network.SendItemIdentify(identifyCancelIndex); err != nil {
			w.setStatus(fmt.Sprintf("Cancel failed: %v", err), false)
			return
		}
	}
	w.open = false
}

func (w *IdentifyWindow) setStatus(text string, good bool) {
	w.status = text
	w.statusGood = good
	w.statusAt = time.Now()
}

func (w *IdentifyWindow) removeIndex(index uint16) {
	for i, candidate := range w.indexes {
		if candidate == index {
			w.indexes = append(w.indexes[:i], w.indexes[i+1:]...)
			return
		}
	}
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
