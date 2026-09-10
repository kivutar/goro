package ui

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	xdraw "golang.org/x/image/draw"
)

const (
	statusIconSize           = 32
	statusIconSpacing        = 36
	statusIconGap            = 8
	statusIconRedrawInterval = 250 * time.Millisecond
	// Status-icon descriptions were introduced after our 2008 target client.
	// Keep the implementation available for now, but do not display it.
	statusIconTooltipsEnabled = false
)

type StatusIcons struct {
	widget     *statusIconsWidget
	root       widget.Widget
	icons      map[uint16]image.Image
	miss       map[uint16]struct{}
	visible    bool
	x          int
	y          int
	width      int
	height     int
	lastIDs    []uint16
	lastRedraw time.Time
}

func (s *StatusIcons) Update(ctx Context, now time.Time) bool {
	if ctx.UIManager == nil || ctx.Session == nil {
		s.Unpublish(ctx)
		return false
	}
	ids := VisibleStatusIconIDs(ctx.Session.Statuses.Active)
	if len(ids) == 0 {
		s.Unpublish(ctx)
		return false
	}
	if s.widget == nil {
		s.widget = newStatusIconsWidget()
	}
	width, height := ctx.ScreenSize()
	x, y, w, h := statusIconOverlayBounds(width, height, len(ids))
	s.widget.ctx = ctx
	s.widget.now = now
	s.widget.ids = ids
	s.widget.icons = s.statusIconImages(ctx.Resources, ids)
	needsRedraw := !sameStatusIconIDs(ids, s.lastIDs) || s.lastRedraw.IsZero() || now.Sub(s.lastRedraw) >= statusIconRedrawInterval
	root := s.overlayRoot(x, y, w, h)
	if root != s.root {
		s.Unpublish(ctx)
		s.root = root
		ctx.UIManager.AddOverlay(root)
		s.visible = true
		needsRedraw = true
	} else if redraw, ok := root.(interface{ SetNeedsRedraw(bool) }); ok {
		if needsRedraw {
			redraw.SetNeedsRedraw(true)
		}
	}
	if needsRedraw {
		s.widget.SetNeedsRedraw(true)
		s.lastRedraw = now
	}
	s.lastIDs = append(s.lastIDs[:0], ids...)
	return false
}

func (s *StatusIcons) Unpublish(ctx Context) {
	if ctx.UIManager == nil || !s.visible || s.root == nil {
		return
	}
	ctx.UIManager.RemoveOverlay(s.root)
	s.root = nil
	s.visible = false
	s.lastIDs = s.lastIDs[:0]
	s.lastRedraw = time.Time{}
}

func (s *StatusIcons) overlayRoot(x, y, w, h int) widget.Widget {
	if s.root != nil && s.x == x && s.y == y && s.width == w && s.height == h {
		return s.root
	}
	s.x = x
	s.y = y
	s.width = w
	s.height = h
	return positionedWidget(s.widget, x, y, w, h)
}

func (s *StatusIcons) statusIconImages(manager *res.Manager, ids []uint16) map[uint16]image.Image {
	images := make(map[uint16]image.Image, len(ids))
	for _, id := range ids {
		if img := s.statusIconImage(manager, id); img != nil {
			images[id] = img
		}
	}
	return images
}

func (s *StatusIcons) statusIconImage(manager *res.Manager, id uint16) image.Image {
	info, ok := db.StatusIconInfoByID(id)
	if !ok || info.Icon == "" || manager == nil {
		return nil
	}
	if s.icons == nil {
		s.icons = make(map[uint16]image.Image)
	}
	if s.miss == nil {
		s.miss = make(map[uint16]struct{})
	}
	if icon, ok := s.icons[id]; ok {
		return icon
	}
	if _, ok := s.miss[id]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.EffectTextureCandidates(info.Icon))
	if err != nil {
		s.miss[id] = struct{}{}
		return nil
	}
	icon := image.NewRGBA(image.Rect(0, 0, statusIconSize, statusIconSize))
	bounds := img.Bounds()
	if bounds.Dx() > 0 && bounds.Dy() > 0 {
		scale := minFloat64(float64(statusIconSize)/float64(bounds.Dx()), float64(statusIconSize)/float64(bounds.Dy()))
		drawW := int(float64(bounds.Dx())*scale + 0.5)
		drawH := int(float64(bounds.Dy())*scale + 0.5)
		drawX := (statusIconSize - drawW) / 2
		drawY := (statusIconSize - drawH) / 2
		xdraw.NearestNeighbor.Scale(icon, image.Rect(drawX, drawY, drawX+drawW, drawY+drawH), img, bounds, xdraw.Over, nil)
	}
	s.icons[id] = icon
	return icon
}

func VisibleStatusIconIDs(active map[uint16]session.StatusEffect) []uint16 {
	ids := make([]uint16, 0, len(active))
	for id := range active {
		// EFST_SIT and its 앉기.tga icon postdate the 2008 client. rAthena may
		// still send the status, but the original client displayed no icon.
		if id == db.StatusSit {
			continue
		}
		info, ok := db.StatusIconInfoByID(id)
		if ok && info.Icon != "" {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sameStatusIconIDs(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func statusIconOverlayBounds(width, height, count int) (int, int, int, int) {
	minimapX, minimapY, minimapW, minimapH := MinimapBounds(width, height)
	startY := minimapY + minimapH + statusIconGap
	maxRows := maxInt(1, (height-startY-windowScreenMargin)/statusIconSpacing)
	cols := maxInt(1, (count+maxRows-1)/maxRows)
	return minimapX + minimapW - statusIconSize - (cols-1)*(statusIconSize+statusIconGap), startY, cols*statusIconSize + (cols-1)*statusIconGap, minInt(count, maxRows) * statusIconSpacing
}

type statusIconsWidget struct {
	widget.WidgetBase
	ctx   Context
	now   time.Time
	ids   []uint16
	icons map[uint16]image.Image
}

func newStatusIconsWidget() *statusIconsWidget {
	w := &statusIconsWidget{}
	w.SetVisible(true)
	w.SetEnabled(false)
	return w
}

func (w *statusIconsWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.BiggestFinite(float32(statusIconSize), float32(statusIconSize))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *statusIconsWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	if w.ctx.Session == nil || len(w.ids) == 0 {
		return
	}
	bounds := w.Bounds()
	height := w.ctx.ScreenH
	startY := int(bounds.Min.Y)
	maxRows := maxInt(1, (height-startY-16)/statusIconSpacing)
	hovered := -1
	for i, id := range w.ids {
		col := i / maxRows
		row := i % maxRows
		x := int(bounds.Max.X) - statusIconSize - col*(statusIconSize+statusIconGap)
		y := startY + row*statusIconSpacing
		effect := w.ctx.Session.Statuses.Active[id]
		w.drawStatusIcon(canvas, id, effect, x, y)
		if statusIconTooltipsEnabled && w.ctx.Input != nil && PointInRect(w.ctx.Input.MouseX, w.ctx.Input.MouseY, x, y, statusIconSize, statusIconSize) {
			hovered = int(id)
		}
	}
	if statusIconTooltipsEnabled && hovered >= 0 && w.ctx.Input != nil && !TooltipsSuppressed(w.ctx) {
		w.drawTooltip(canvas, uint16(hovered), w.ctx.Session.Statuses.Active[uint16(hovered)], w.ctx.Input.MouseX, w.ctx.Input.MouseY)
	}
}

func (w *statusIconsWidget) Event(_ widget.Context, _ event.Event) bool {
	return false
}

func (w *statusIconsWidget) drawStatusIcon(canvas widget.Canvas, id uint16, effect session.StatusEffect, x, y int) {
	canvas.DrawRect(geometry.NewRect(float32(x-1), float32(y-1), statusIconSize+2, statusIconSize+2), Color(color.RGBA{R: 60, G: 74, B: 96, A: 170}))
	canvas.DrawRect(geometry.NewRect(float32(x), float32(y), statusIconSize, statusIconSize), Color(color.RGBA{R: 236, G: 242, B: 250, A: 215}))
	if icon := w.icons[id]; icon != nil {
		canvas.DrawImage(icon, geometry.Pt(float32(x), float32(y)))
	} else {
		canvas.DrawText("?", geometry.NewRect(float32(x), float32(y+7), statusIconSize, 16), 11, Color(MutedTextColor), false, widget.TextAlignCenter)
	}
	if effect.HasDuration && !effect.ExpiresAt.IsZero() && effect.ExpiresAt.After(effect.StartedAt) {
		total := effect.ExpiresAt.Sub(effect.StartedAt)
		remaining := effect.ExpiresAt.Sub(w.now)
		if remaining < 0 {
			remaining = 0
		}
		frac := float64(remaining) / float64(total)
		fillW := int(float64(statusIconSize) * clampUnit(frac))
		canvas.DrawRect(geometry.NewRect(float32(x), float32(y+statusIconSize-4), statusIconSize, 4), Color(color.RGBA{R: 18, G: 24, B: 34, A: 180}))
		if fillW > 0 {
			canvas.DrawRect(geometry.NewRect(float32(x), float32(y+statusIconSize-4), float32(fillW), 4), Color(color.RGBA{R: 244, G: 228, B: 130, A: 230}))
		}
	}
}

func (w *statusIconsWidget) drawTooltip(canvas widget.Canvas, statusID uint16, effect session.StatusEffect, mouseX, mouseY int) {
	info, ok := db.StatusIconInfoByID(statusID)
	if !ok || len(info.Lines) == 0 {
		return
	}
	lines := statusIconTooltipLines(info, effect, w.now)
	if len(lines) == 0 {
		return
	}
	width, height := w.ctx.ScreenSize()
	tipW := statusIconTooltipWidth(lines)
	tipH := len(lines)*16 + 6
	x := clampWindowInt(mouseX+12, 4, maxInt(4, width-tipW-4))
	y := clampWindowInt(mouseY+12, 4, maxInt(4, height-tipH-4))
	rect := geometry.NewRect(float32(x), float32(y), float32(tipW), float32(tipH))
	canvas.DrawRect(rect, Color(color.RGBA{R: 0, G: 0, B: 0, A: 128}))
	canvas.StrokeRect(rect, Color(color.RGBA{R: 198, G: 198, B: 198, A: 255}), 1)
	for i, line := range lines {
		canvas.DrawText(line.text, geometry.NewRect(float32(x+5), float32(y+3+i*16), float32(tipW-10), 14), 11, Color(line.color), false, widget.TextAlignLeft)
	}
}

type statusIconTooltipLine struct {
	text  string
	color color.RGBA
}

func statusIconTooltipLines(info db.StatusIconInfo, effect session.StatusEffect, now time.Time) []statusIconTooltipLine {
	out := make([]statusIconTooltipLine, 0, len(info.Lines))
	for _, line := range info.Lines {
		text := line.Text
		if text == "%s" {
			text = statusIconRemainingTime(effect, now)
			if text == "" {
				continue
			}
		}
		c := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		if line.HasColor {
			c = line.Color
		}
		out = append(out, statusIconTooltipLine{text: text, color: c})
	}
	return out
}

func statusIconRemainingTime(effect session.StatusEffect, now time.Time) string {
	if !effect.HasDuration || effect.ExpiresAt.IsZero() || !effect.ExpiresAt.After(now) {
		return ""
	}
	remaining := int(effect.ExpiresAt.Sub(now).Seconds())
	minutes := remaining / 60
	seconds := remaining % 60
	if minutes > 0 {
		return fmt.Sprintf("%d minute %d second", minutes, seconds)
	}
	return fmt.Sprintf("%d second", seconds)
}

func statusIconTooltipWidth(lines []statusIconTooltipLine) int {
	width := 0
	for _, line := range lines {
		width = maxInt(width, len([]rune(line.text))*7+10)
	}
	return clampWindowInt(width, 42, 300)
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
