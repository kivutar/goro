package ui

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/ui/rotheme"
	worldstate "github.com/kivutar/goro/world"
	xdraw "golang.org/x/image/draw"
)

const (
	minimapWidth   = 188
	minimapHeight  = 206
	minimapMargin  = 16
	minimapPad     = 10
	minimapFooterH = 22
)

var (
	minimapTextColor   = TextColor
	minimapMutedColor  = MutedTextColor
	minimapPlayerColor = color.RGBA{R: 255, G: 232, B: 96, A: 255}
)

type Minimap struct {
	mapName     string
	img         image.Image
	scaled      image.Image
	scaledKey   string
	window      WindowState
	widget      *minimapWidget
	hidden      bool
	markerMap   string
	markerX     int
	markerY     int
	hasPosition bool
}

type minimapRect struct {
	x int
	y int
	w int
	h int
}

func (m *Minimap) Update(ctx Context) bool {
	width, height := ctx.ScreenSize()
	x, y, w, h := minimapBounds(width, height)
	m.ensureWindow(w, h)
	if ctx.World == nil || m.hidden {
		m.window.Close()
		m.window.Unpublish(ctx)
		m.hasPosition = false
		return false
	}
	previousMap := m.mapName
	m.ensureImage(ctx.Resources, ctx.World.MapName)
	if m.widget == nil {
		m.widget = newMinimapWidget()
	}
	m.widget.ctx = ctx
	m.widget.image = m.scaledImage(minimapContentMapSize(w, h))
	markerChanged := m.playerMarkerChanged(ctx.World.Player.X, ctx.World.Player.Y)
	needsPublish := false
	if !m.window.IsOpen() {
		m.window.OpenAt(x, y, m.widgetTree())
		needsPublish = true
	} else {
		if markerChanged || previousMap != m.mapName {
			m.window.SetContent(m.widgetTree())
			needsPublish = true
		}
		if m.window.SetAutoPosition(x, y) {
			needsPublish = true
		}
	}
	if m.window.published == nil {
		needsPublish = true
	}
	if needsPublish {
		m.widget.SetNeedsRedraw(true)
		m.window.Publish(ctx)
	}
	return false
}

func (m *Minimap) Toggle(ctx Context) {
	m.hidden = !m.hidden
	if m.hidden {
		m.window.Close()
		m.window.Unpublish(ctx)
		return
	}
	m.Update(ctx)
}

func (m *Minimap) ensureWindow(width, height int) {
	if m.window.width != 0 {
		return
	}
	m.window = NewWindowState(width, height)
	m.window.SetCloseOnEscape(false)
}

func (m *Minimap) widgetTree() widget.Widget {
	return Window(
		Title("Mini Map"),
		CloseButton(false),
		Size(minimapWidth, minimapHeight),
		Content(m.widget),
	)
}

func (m *Minimap) ensureImage(manager *res.Manager, mapName string) {
	normalized := normalizeMinimapMapName(mapName)
	if normalized == "" || manager == nil {
		return
	}
	if m.mapName == normalized {
		return
	}
	m.mapName = normalized
	m.img = nil
	m.scaled = nil
	m.scaledKey = ""
	img, _, err := res.LoadImage(manager, minimapImageCandidates(normalized))
	if err != nil {
		return
	}
	m.img = img
}

func (m *Minimap) playerMarkerChanged(x, y int) bool {
	if m.hasPosition && m.markerMap == m.mapName && m.markerX == x && m.markerY == y {
		return false
	}
	m.markerMap = m.mapName
	m.markerX = x
	m.markerY = y
	m.hasPosition = true
	return true
}

func minimapBounds(width, _ int) (int, int, int, int) {
	x := maxInt(minimapMargin, width-minimapWidth-minimapMargin)
	return x, minimapMargin, minimapWidth, minimapHeight
}

func MinimapBounds(width, height int) (int, int, int, int) {
	return minimapBounds(width, height)
}

func minimapMapRect(x, y, w, h int) minimapRect {
	available := h - ROWindowTitleHeight - minimapFooterH - minimapPad
	size := minInt(w-2*minimapPad, available)
	if size < 32 {
		size = 32
	}
	return minimapRect{
		x: x + (w-size)/2,
		y: y + ROWindowTitleHeight + 4,
		w: size,
		h: size,
	}
}

func minimapContentMapSize(w, h int) int {
	return minimapMapRect(0, 0, w, h).w
}

func (m *Minimap) scaledImage(size int) image.Image {
	if m.img == nil || size <= 0 {
		return nil
	}
	key := fmt.Sprintf("%s:%d", m.mapName, size)
	if m.scaled != nil && m.scaledKey == key {
		return m.scaled
	}
	bounds := m.img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	srcAspect := float64(bounds.Dx()) / float64(bounds.Dy())
	drawW, drawH := size, size
	if srcAspect > 1 {
		drawH = int(float64(drawW)/srcAspect + 0.5)
	} else if srcAspect < 1 {
		drawW = int(float64(drawH)*srcAspect + 0.5)
	}
	drawX := (size - drawW) / 2
	drawY := (size - drawH) / 2
	xdraw.ApproxBiLinear.Scale(dst, image.Rect(drawX, drawY, drawX+drawW, drawY+drawH), m.img, bounds, xdraw.Over, nil)
	m.scaled = dst
	m.scaledKey = key
	return m.scaled
}

type minimapWidget struct {
	widget.WidgetBase
	ctx   Context
	image image.Image
}

func newMinimapWidget() *minimapWidget {
	w := &minimapWidget{}
	w.SetVisible(true)
	w.SetEnabled(false)
	return w
}

func (w *minimapWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Size{Width: minimapWidth, Height: minimapHeight - ROWindowTitleHeight})
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *minimapWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	if bounds.IsEmpty() {
		return
	}
	rect := minimapContentMapRect(bounds)
	drawMinimapFallback(canvas, rect)
	if w.image != nil {
		canvas.DrawImage(w.image, geometry.Pt(float32(rect.x), float32(rect.y)))
	}
	strokeMinimapRect(canvas, rect)
	if w.ctx.World != nil {
		mapW, mapH := minimapWorldSize(w.ctx.World)
		if mapW > 0 && mapH > 0 {
			drawMinimapMarker(canvas, rect, mapW, mapH, w.ctx.World.Player.X, w.ctx.World.Player.Y, minimapPlayerColor, 4)
		}
		label := minimapDisplayName(w.ctx.World.MapName)
		footerY := bounds.Min.Y + bounds.Height() - 19
		canvas.DrawText(trimRunes(label, 13), geometry.NewRect(bounds.Min.X+minimapPad, footerY, bounds.Width()/2, 16), float32(rotheme.Default.Typography.TextSize), Color(minimapTextColor), false, widget.TextAlignLeft)
		coords := fmt.Sprintf("X:%d Y:%d", w.ctx.World.Player.X, w.ctx.World.Player.Y)
		canvas.DrawText(coords, geometry.NewRect(bounds.Min.X+bounds.Width()/2, footerY, bounds.Width()/2-minimapPad, 16), float32(rotheme.Default.Typography.TextSize), Color(minimapMutedColor), false, widget.TextAlignRight)
	}
}

func (w *minimapWidget) Event(_ widget.Context, _ event.Event) bool {
	return false
}

func minimapContentMapRect(bounds geometry.Rect) minimapRect {
	available := int(bounds.Height()) - minimapFooterH - minimapPad
	size := minInt(int(bounds.Width())-2*minimapPad, available)
	if size < 32 {
		size = 32
	}
	return minimapRect{
		x: int(bounds.Min.X) + (int(bounds.Width())-size)/2,
		y: int(bounds.Min.Y) + 4,
		w: size,
		h: size,
	}
}

func drawMinimapFallback(canvas widget.Canvas, rect minimapRect) {
	canvas.DrawRect(geometry.NewRect(float32(rect.x), float32(rect.y), float32(rect.w), float32(rect.h)), Color(color.RGBA{R: 212, G: 228, B: 202, A: 255}))
	for i := 1; i < 8; i++ {
		x := rect.x + rect.w*i/8
		y := rect.y + rect.h*i/8
		canvas.DrawRect(geometry.NewRect(float32(x), float32(rect.y), 1, float32(rect.h)), Color(color.RGBA{R: 132, G: 164, B: 118, A: 105}))
		canvas.DrawRect(geometry.NewRect(float32(rect.x), float32(y), float32(rect.w), 1), Color(color.RGBA{R: 132, G: 164, B: 118, A: 105}))
	}
}

func strokeMinimapRect(canvas widget.Canvas, rect minimapRect) {
	canvas.StrokeRect(geometry.NewRect(float32(rect.x), float32(rect.y), float32(rect.w), float32(rect.h)), Color(WindowBorderColor), 1)
}

func drawMinimapMarker(canvas widget.Canvas, rect minimapRect, mapW, mapH, cellX, cellY int, fill color.RGBA, radius int) {
	x, y, ok := minimapCellToScreen(rect, mapW, mapH, cellX, cellY)
	if !ok {
		return
	}
	canvas.DrawRect(geometry.NewRect(float32(x-radius-1), float32(y-radius-1), float32(radius*2+3), float32(radius*2+3)), Color(color.RGBA{A: 190}))
	canvas.DrawRect(geometry.NewRect(float32(x-radius), float32(y-radius), float32(radius*2+1), float32(radius*2+1)), Color(fill))
}

func minimapCellToScreen(rect minimapRect, mapW, mapH, cellX, cellY int) (int, int, bool) {
	if mapW <= 0 || mapH <= 0 || rect.w <= 0 || rect.h <= 0 {
		return 0, 0, false
	}
	nx := clampUnit((float64(cellX) + 0.5) / float64(mapW))
	ny := clampUnit((float64(cellY) + 0.5) / float64(mapH))
	x := rect.x + int(nx*float64(rect.w-1)+0.5)
	y := rect.y + int((1-ny)*float64(rect.h-1)+0.5)
	return x, y, true
}

func minimapWorldSize(world *worldstate.World) (int, int) {
	if world == nil {
		return 0, 0
	}
	if world.GAT != nil && world.GAT.Width > 0 && world.GAT.Height > 0 {
		return world.GAT.Width, world.GAT.Height
	}
	if world.GND != nil && world.GND.Width > 0 && world.GND.Height > 0 {
		return world.GND.Width, world.GND.Height
	}
	return 0, 0
}

func minimapImageCandidates(mapName string) []string {
	base := normalizeMinimapMapName(mapName)
	if base == "" {
		return nil
	}
	file := base + ".bmp"
	koreanInterface := "\xc0\xaf\xc0\xfa\xc0\xce\xc5\xcd\xc6\xe4\xc0\xcc\xbd\xba"
	return []string{
		"data\\texture\\" + koreanInterface + "\\map\\" + file,
		"data\\texture\\" + koreanInterface + "\\minimap\\" + file,
		"texture\\" + koreanInterface + "\\map\\" + file,
		"texture\\" + koreanInterface + "\\minimap\\" + file,
		"data\\texture\\interface\\map\\" + file,
		"data\\texture\\interface\\minimap\\" + file,
		"data\\texture\\map\\" + file,
		"data\\texture\\minimap\\" + file,
		"texture\\interface\\map\\" + file,
		"texture\\interface\\minimap\\" + file,
		"texture\\map\\" + file,
		"texture\\minimap\\" + file,
		file,
	}
}

func normalizeMinimapMapName(mapName string) string {
	name := strings.TrimSpace(mapName)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".rsw"), ".gat")
	name = strings.TrimSuffix(name, ".gnd")
	name = strings.TrimSuffix(name, ".bmp")
	return strings.ToLower(name)
}

func minimapDisplayName(mapName string) string {
	name := normalizeMinimapMapName(mapName)
	if name == "" {
		return "unknown"
	}
	return name
}
