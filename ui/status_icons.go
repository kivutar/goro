package ui

import (
	"image"
	"image/color"
	"sort"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	xdraw "golang.org/x/image/draw"
)

const (
	statusIconSize    = 32
	statusIconSpacing = 36
	statusIconGap     = 8
)

type statusIconInfo struct {
	icon  string
	label string
}

var statusIconInfos = map[uint16]statusIconInfo{
	0:  {icon: "\xc7\xc1\xb7\xce\xba\xb8\xc5\xa9.tga", label: "Provoke"},
	1:  {icon: "\xc0\xce\xb5\xe0\xbe\xee.tga", label: "Endure"},
	2:  {icon: "\xc5\xf5\xc7\xda\xb5\xe5\xc4\xfb\xc5\xab.tga", label: "Two Hand Quicken"},
	3:  {icon: "\xc1\xfd\xc1\xdf\xb7\xc2\xc7\xe2\xbb\xf3.tga", label: "Attention Concentration"},
	9:  {icon: "\xbe\xc8\xc1\xa9\xb7\xe7\xbd\xba.tga", label: "Angelus"},
	10: {icon: "\xba\xed\xb7\xb9\xbd\xcc.tga", label: "Blessing"},
	11: {icon: "\xbd\xc3\xb1\xd7\xb3\xd1\xc5\xa9\xb7\xe7\xbd\xc3\xbd\xba.tga", label: "Signum Crucis"},
	12: {icon: "\xb9\xce\xc3\xb8\xbc\xba\xc1\xf5\xb0\xa1.tga", label: "Increase Agility"},
	13: {icon: "\xb9\xce\xc3\xb8\xbc\xba\xb0\xa8\xbc\xd2.tga", label: "Decrease Agility"},
	15: {icon: "\xc0\xd3\xc6\xf7\xbd\xc3\xc6\xbc\xbf\xc0\xb8\xb6\xb4\xa9\xbd\xba.tga", label: "Impositio Manus"},
	16: {icon: "\xbc\xf6\xc1\xdd\xc0\xba\xc7\xcf\xb7\xe7\xc0\xc7\xbf\xec\xbf\xef.tga", label: "Suffragium"},
	19: {icon: "\xb1\xe2\xb8\xae\xbf\xa1\xbf\xa4\xb7\xb9\xc0\xcc\xbc\xd5.tga", label: "Kyrie Eleison"},
	20: {icon: "\xb8\xb6\xb4\xcf\xc7\xc7\xc4\xb1.tga", label: "Magnificat"},
	21: {icon: "\xb1\xdb\xb7\xce\xb8\xae\xbe\xc6.tga", label: "Gloria"},
	23: {icon: "\xbe\xc6\xb5\xe5\xb7\xb9\xb3\xaf\xb8\xb0\xb7\xaf\xbd\xac.tga", label: "Adrenaline Rush"},
	25: {icon: "\xbf\xc0\xb9\xf6\xc6\xae\xb7\xaf\xbd\xba\xc6\xae.tga", label: "Over Thrust"},
	26: {icon: "\xb8\xc6\xbd\xc3\xb8\xb6\xc0\xcc\xc1\xee\xc6\xc4\xbf\xf6.tga", label: "Maximize Power"},
	37: {icon: "\xb0\xf8\xbc\xd3\xb9\xb0\xbe\xe0.tga", label: "Attack Speed Potion"},
	38: {icon: "\xb0\xf8\xbc\xd3\xb9\xb0\xbe\xe0.tga", label: "Attack Speed Potion"},
	39: {icon: "\xb0\xf8\xbc\xd3\xb9\xb0\xbe\xe0.tga", label: "Attack Speed Potion"},
	41: {icon: "\xb9\xce\xc3\xb8\xbc\xba\xc1\xf5\xb0\xa1.tga", label: "Movement Speed Potion"},
}

type StatusIcons struct {
	widget  *statusIconsWidget
	root    widget.Widget
	icons   map[uint16]image.Image
	miss    map[uint16]struct{}
	visible bool
	x       int
	y       int
	width   int
	height  int
}

func (s *StatusIcons) Update(ctx Context, now time.Time) bool {
	if ctx.Session == nil {
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
	s.widget.SetNeedsRedraw(true)
	root := s.overlayRoot(x, y, w, h)
	if root != s.root {
		s.Unpublish(ctx)
		s.root = root
		ctx.UIManager.AddOverlay(root)
		s.visible = true
	} else if redraw, ok := root.(interface{ SetNeedsRedraw(bool) }); ok {
		redraw.SetNeedsRedraw(true)
	}
	return false
}

func (s *StatusIcons) Unpublish(ctx Context) {
	if ctx.UIManager == nil || !s.visible || s.root == nil {
		return
	}
	ctx.UIManager.RemoveOverlay(s.root)
	s.root = nil
	s.visible = false
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
	info, ok := statusIconInfos[id]
	if !ok || manager == nil {
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
	img, _, err := res.LoadImage(manager, res.EffectTextureCandidates(info.icon))
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
		if _, ok := statusIconInfos[id]; ok {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func statusIconOverlayBounds(width, height, count int) (int, int, int, int) {
	minimapX, minimapY, minimapW, minimapH := MinimapBounds(width, height)
	startY := minimapY + minimapH + statusIconGap
	maxRows := maxInt(1, (height-startY-16)/statusIconSpacing)
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
		if w.ctx.Input != nil && PointInRect(w.ctx.Input.MouseX, w.ctx.Input.MouseY, x, y, statusIconSize, statusIconSize) {
			hovered = int(id)
		}
	}
	if hovered >= 0 && w.ctx.Input != nil {
		w.drawTooltip(canvas, hovered, w.ctx.Input.MouseX, w.ctx.Input.MouseY)
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

func (w *statusIconsWidget) drawTooltip(canvas widget.Canvas, statusID int, mouseX, mouseY int) {
	info, ok := statusIconInfos[uint16(statusID)]
	if !ok || info.label == "" {
		return
	}
	text := info.label
	width, height := w.ctx.ScreenSize()
	tipW := len(text)*7 + 12
	tipH := 20
	x := clampWindowInt(mouseX+12, 4, maxInt(4, width-tipW-4))
	y := clampWindowInt(mouseY+12, 4, maxInt(4, height-tipH-4))
	rect := geometry.NewRect(float32(x), float32(y), float32(tipW), float32(tipH))
	canvas.DrawRect(rect, Color(color.RGBA{R: 32, G: 36, B: 44, A: 230}))
	canvas.StrokeRect(rect, Color(WindowBorderColor), 1)
	canvas.DrawText(text, geometry.NewRect(float32(x+6), float32(y+3), float32(tipW-12), 14), 11, Color(color.RGBA{R: 246, G: 246, B: 246, A: 255}), false, widget.TextAlignLeft)
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
