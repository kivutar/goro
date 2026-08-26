package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/ui/rotheme"
	worldstate "github.com/kivutar/goro/world"
	xdraw "golang.org/x/image/draw"
)

const (
	minimapWidth           = 188
	minimapHeight          = 206
	minimapMargin          = windowScreenMargin
	minimapPad             = 10
	minimapInfoBandH       = 22
	minimapCompassDuration = 15 * time.Second
)

var (
	minimapTextColor   = TextColor
	minimapMutedColor  = MutedTextColor
	minimapPlayerColor = color.RGBA{R: 255, G: 232, B: 96, A: 255}
	minimapGuildColor  = color.RGBA{R: 255, G: 140, B: 26, A: 255}
)

type Minimap struct {
	mapName          string
	img              image.Image
	mapImageTried    bool
	scaled           image.Image
	scaledKey        string
	arrow            image.Image
	arrowLoadTried   bool
	arrowVariants    [8]image.Image
	window           Window
	widget           *minimapWidget
	hidden           bool
	markerMap        string
	markerX          int
	markerY          int
	markerDir        int
	hasPosition      bool
	visualKey        string
	compass          map[uint8]minimapCompassMarker
	compassRevision  uint64
	compassDrawnRev  uint64
	guild            map[uint32]minimapGuildMarker
	guildRevision    uint64
	guildDrawnRev    uint64
	guildSnapshotRev uint64
	guildSnapshotBuf []minimapGuildMarker
	pendingMarker    bool
	pendingMarkerOld minimapPlayerMarkerState
}

type minimapRect struct {
	x int
	y int
	w int
	h int
}

type minimapCompassMarker struct {
	id      uint8
	typ     int
	x       int
	y       int
	color   color.RGBA
	expires time.Time
}

type minimapGuildMarker struct {
	accountID uint32
	x         int
	y         int
}

type minimapPlayerMarkerState struct {
	valid bool
	mapID string
	x     int
	y     int
	dir   int
	arrow image.Image
}

func (m *Minimap) Update(ctx Context) bool {
	now := time.Now()
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
	m.ensureArrow(ctx.Resources)
	mapChanged := previousMap != m.mapName
	compassChanged := false
	if mapChanged && previousMap != "" {
		compassChanged = m.clearCompassMarkers()
		m.clearGuildMarkers()
	}
	if m.pruneCompassMarkers(now) {
		compassChanged = true
	}
	if m.widget == nil {
		m.widget = newMinimapWidget()
	}
	oldMarker := m.playerMarkerState()
	m.widget.ctx = ctx
	m.widget.image = m.scaledImage(minimapContentMapSize(w, h))
	m.widget.arrow = m.playerArrow(ctx.World.Player.Dir)
	m.widget.now = now
	m.widget.compassMarkers = m.compassSnapshot()
	m.widget.guildMarkers = m.guildSnapshot()
	markerChanged := m.playerMarkerChanged(ctx.World.Player.X, ctx.World.Player.Y, ctx.World.Player.Dir)
	visualKey := m.currentVisualKey(ctx, now)
	visualChanged := visualKey != m.visualKey
	needsPublish := false
	fullRedraw := mapChanged || visualChanged || compassChanged || m.compassDrawnRev != m.compassRevision || m.guildDrawnRev != m.guildRevision || len(m.widget.compassMarkers) > 0
	markerOnly := markerChanged && !fullRedraw
	drawPendingMarker := m.pendingMarker && !markerChanged
	needsRedraw := fullRedraw || drawPendingMarker
	if !m.window.IsOpen() {
		m.window.OpenAt(x, y, m.widgetTree())
		needsPublish = true
	} else {
		if m.window.SetAutoPosition(x, y) {
			needsPublish = true
		}
	}
	if m.window.published == nil {
		needsPublish = true
	}
	if needsPublish {
		m.window.Publish(ctx)
		m.clearPendingMarker()
		m.markRedraw(ctx)
		m.compassDrawnRev = m.compassRevision
		m.guildDrawnRev = m.guildRevision
	} else if needsRedraw {
		if fullRedraw {
			m.clearPendingMarker()
			m.markRedraw(ctx)
		} else if drawPendingMarker {
			m.markPlayerMarkerRedraw(ctx, m.pendingMarkerOld)
			m.clearPendingMarker()
		}
		m.compassDrawnRev = m.compassRevision
		m.guildDrawnRev = m.guildRevision
	}
	if markerOnly {
		m.queuePendingMarker(oldMarker)
	}
	m.visualKey = visualKey
	wasDragging := m.window.dragging
	m.window.Update(ctx)
	return wasDragging || m.window.dragging
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
	m.window = NewWindow(width, height)
	m.window.SetBackground(widget.Color{})
	m.window.CloseOnEsc = false
}

func (m *Minimap) widgetTree() widget.Widget {
	return Win(
		Title("Mini Map"),
		CloseButton(false),
		Size(minimapWidth, minimapHeight),
		Content(m.widget),
	)
}

func (m *Minimap) ensureImage(manager *res.Manager, mapName string) {
	normalized := normalizeMinimapMapName(mapName)
	if normalized == "" {
		return
	}
	if m.mapName != normalized {
		m.mapName = normalized
		m.img = nil
		m.mapImageTried = false
		m.scaled = nil
		m.scaledKey = ""
	}
	if manager == nil || m.img != nil || m.mapImageTried {
		return
	}
	m.mapImageTried = true
	img, _, err := res.LoadImage(manager, minimapImageCandidates(normalized))
	if err != nil {
		return
	}
	m.img = img
}

func (m *Minimap) ensureArrow(manager *res.Manager) {
	if manager == nil || m.arrow != nil || m.arrowLoadTried {
		return
	}
	m.arrowLoadTried = true
	img, _, err := res.LoadImage(manager, minimapArrowCandidates())
	if err != nil {
		return
	}
	m.arrow = img
	m.arrowVariants = [8]image.Image{}
}

func (m *Minimap) playerMarkerChanged(x, y, dir int) bool {
	dir = normalizeMinimapDirection(dir)
	if m.hasPosition && m.markerMap == m.mapName && m.markerX == x && m.markerY == y && m.markerDir == dir {
		return false
	}
	m.markerMap = m.mapName
	m.markerX = x
	m.markerY = y
	m.markerDir = dir
	m.hasPosition = true
	return true
}

func (m *Minimap) playerMarkerState() minimapPlayerMarkerState {
	return minimapPlayerMarkerState{
		valid: m.hasPosition,
		mapID: m.markerMap,
		x:     m.markerX,
		y:     m.markerY,
		dir:   m.markerDir,
		arrow: m.playerArrow(m.markerDir),
	}
}

func (m *Minimap) queuePendingMarker(old minimapPlayerMarkerState) {
	if m.pendingMarker {
		return
	}
	m.pendingMarker = true
	m.pendingMarkerOld = old
}

func (m *Minimap) clearPendingMarker() {
	m.pendingMarker = false
	m.pendingMarkerOld = minimapPlayerMarkerState{}
}

func (m *Minimap) playerArrow(dir int) image.Image {
	if m.arrow == nil {
		return nil
	}
	dir = normalizeMinimapDirection(dir)
	if m.arrowVariants[dir] == nil {
		m.arrowVariants[dir] = rotateMinimapArrow(m.arrow, minimapPlayerArrowAngle(dir))
	}
	return m.arrowVariants[dir]
}

func (m *Minimap) ApplyCompass(id uint8, typ, x, y int, rgb uint32, now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	if typ == 2 {
		if len(m.compass) == 0 {
			return
		}
		if _, ok := m.compass[id]; !ok {
			return
		}
		delete(m.compass, id)
		m.compassRevision++
		m.markCompassDirty()
		return
	}
	if m.compass == nil {
		m.compass = make(map[uint8]minimapCompassMarker)
	}
	marker := minimapCompassMarker{
		id:    id,
		typ:   typ,
		x:     x,
		y:     y,
		color: minimapCompassColor(rgb),
	}
	if typ == 0 {
		marker.expires = now.Add(minimapCompassDuration)
	}
	m.compass[id] = marker
	m.compassRevision++
	m.markCompassDirty()
}

// ApplyGuildMemberPosition updates one guild member marker. The server uses
// negative coordinates to remove a member who left the map or went offline.
func (m *Minimap) ApplyGuildMemberPosition(accountID uint32, x, y int) {
	if accountID == 0 {
		return
	}
	if x < 0 || y < 0 {
		if _, ok := m.guild[accountID]; !ok {
			return
		}
		delete(m.guild, accountID)
		m.guildRevision++
		m.markGuildDirty()
		return
	}
	if m.guild == nil {
		m.guild = make(map[uint32]minimapGuildMarker)
	}
	marker := minimapGuildMarker{accountID: accountID, x: x, y: y}
	if current, ok := m.guild[accountID]; ok && current == marker {
		return
	}
	m.guild[accountID] = marker
	m.guildRevision++
	m.markGuildDirty()
}

func (m *Minimap) markGuildDirty() {
	if m.widget != nil {
		m.widget.SetNeedsRedraw(true)
	}
}

func (m *Minimap) markCompassDirty() {
	if m.widget != nil {
		m.widget.SetNeedsRedraw(true)
	}
}

func (m *Minimap) markRedraw(ctx Context) {
	if m.widget != nil {
		m.widget.SetNeedsRedraw(true)
	}
	if m.window.published != nil {
		if redraw, ok := m.window.published.(interface{ SetNeedsRedraw(bool) }); ok {
			redraw.SetNeedsRedraw(true)
		}
	}
	if ctx.UIApp != nil {
		ctx.UIApp.Invalidate()
	}
}

func (m *Minimap) markPlayerMarkerRedraw(ctx Context, old minimapPlayerMarkerState) {
	if m.widget == nil || ctx.World == nil {
		m.markRedraw(ctx)
		return
	}
	current := minimapPlayerMarkerState{
		valid: true,
		mapID: m.mapName,
		x:     ctx.World.Player.X,
		y:     ctx.World.Player.Y,
		dir:   normalizeMinimapDirection(ctx.World.Player.Dir),
		arrow: m.widget.arrow,
	}
	if !m.widget.markPlayerDirty(old, current) {
		m.markRedraw(ctx)
		return
	}
	if ctx.UIApp != nil {
		if app, ok := ctx.UIApp.(rectInvalidatingUIApp); ok {
			app.InvalidateRect(m.widget.ScreenBounds())
		} else {
			ctx.UIApp.Invalidate()
		}
	}
}

func (m *Minimap) clearCompassMarkers() bool {
	if len(m.compass) == 0 {
		return false
	}
	for id := range m.compass {
		delete(m.compass, id)
	}
	m.compassRevision++
	m.markCompassDirty()
	return true
}

func (m *Minimap) clearGuildMarkers() bool {
	if len(m.guild) == 0 {
		return false
	}
	clear(m.guild)
	m.guildRevision++
	m.markGuildDirty()
	return true
}

func (m *Minimap) pruneCompassMarkers(now time.Time) bool {
	if len(m.compass) == 0 {
		return false
	}
	changed := false
	for id, marker := range m.compass {
		if marker.expires.IsZero() || now.Before(marker.expires) {
			continue
		}
		delete(m.compass, id)
		changed = true
	}
	if changed {
		m.compassRevision++
		m.markCompassDirty()
	}
	return changed
}

func (m *Minimap) compassSnapshot() []minimapCompassMarker {
	if len(m.compass) == 0 {
		return nil
	}
	markers := make([]minimapCompassMarker, 0, len(m.compass))
	for _, marker := range m.compass {
		markers = append(markers, marker)
	}
	sort.Slice(markers, func(i, j int) bool {
		return markers[i].id < markers[j].id
	})
	return markers
}

func (m *Minimap) guildSnapshot() []minimapGuildMarker {
	if m.guildSnapshotRev == m.guildRevision {
		return m.guildSnapshotBuf
	}
	if len(m.guild) == 0 {
		m.guildSnapshotBuf = m.guildSnapshotBuf[:0]
		m.guildSnapshotRev = m.guildRevision
		return m.guildSnapshotBuf
	}
	markers := m.guildSnapshotBuf[:0]
	if cap(markers) < len(m.guild) {
		markers = make([]minimapGuildMarker, 0, len(m.guild))
	}
	for _, marker := range m.guild {
		markers = append(markers, marker)
	}
	sort.Slice(markers, func(i, j int) bool {
		return markers[i].accountID < markers[j].accountID
	})
	m.guildSnapshotBuf = markers
	m.guildSnapshotRev = m.guildRevision
	return m.guildSnapshotBuf
}

func (m *Minimap) currentVisualKey(ctx Context, now time.Time) string {
	if ctx.World == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "map=%s|img=%s|compass=%d|guild=%d", m.mapName, minimapImageStateKey(m.widget.image), m.compassRevision, m.guildRevision)
	if len(m.widget.compassMarkers) > 0 {
		fmt.Fprintf(&b, "|blink=%d", minimapCompassBlinkPhase(now, ctx.Started))
		for _, marker := range m.widget.compassMarkers {
			fmt.Fprintf(&b, "|c=%d,%d,%d,%d,%02x%02x%02x,%t", marker.id, marker.typ, marker.x, marker.y, marker.color.R, marker.color.G, marker.color.B, marker.expires.IsZero())
		}
	}
	if ctx.Session != nil && ctx.Session.Party.Active() {
		currentMap := normalizeMinimapMapName(ctx.World.MapName)
		for _, member := range ctx.Session.Party.Members {
			if member.AccountID == 0 || member.AccountID == ctx.Session.AccountID || !member.Online() || member.X < 0 || member.Y < 0 {
				continue
			}
			if memberMap := normalizeMinimapMapName(member.MapName); memberMap != "" && memberMap != currentMap {
				continue
			}
			fmt.Fprintf(&b, "|p=%d,%d,%d,%s", member.AccountID, member.X, member.Y, normalizeMinimapMapName(member.MapName))
		}
	}
	return b.String()
}

func minimapBounds(width, _ int) (int, int, int, int) {
	x := maxInt(minimapMargin, width-minimapWidth-minimapMargin)
	return x, minimapMargin, minimapWidth, minimapHeight
}

func MinimapBounds(width, height int) (int, int, int, int) {
	return minimapBounds(width, height)
}

func minimapMapRect(x, y, w, h int) minimapRect {
	available := h - ROWindowTitleHeight - minimapInfoBandH - minimapPad
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
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), m.img, bounds, xdraw.Over, nil)
	m.scaled = dst
	m.scaledKey = key
	return m.scaled
}

type minimapWidget struct {
	widget.WidgetBase
	ctx            Context
	image          image.Image
	arrow          image.Image
	now            time.Time
	compassMarkers []minimapCompassMarker
	guildMarkers   []minimapGuildMarker
	dirtyRect      geometry.Rect
}

func newMinimapWidget() *minimapWidget {
	w := &minimapWidget{}
	w.SetVisible(true)
	w.SetEnabled(false)
	w.SetRepaintBoundary(true)
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
	if w.image != nil {
		canvas.DrawImage(w.image, geometry.Pt(float32(rect.x), float32(rect.y)))
	}
	if w.ctx.World != nil {
		mapW, mapH := minimapWorldSize(w.ctx.World)
		if mapW > 0 && mapH > 0 {
			drawMinimapGuildMarkers(canvas, rect, mapW, mapH, w.ctx, w.guildMarkers)
			drawMinimapPartyMarkers(canvas, rect, mapW, mapH, w.ctx)
			drawMinimapPlayerMarker(canvas, rect, mapW, mapH, w.ctx.World.Player.X, w.ctx.World.Player.Y, w.arrow)
			drawMinimapCompassMarkers(canvas, rect, mapW, mapH, w.compassMarkers, w.now, w.ctx.Started)
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

func (w *minimapWidget) ScreenBounds() geometry.Rect {
	if w.NeedsRedraw() && !w.dirtyRect.IsEmpty() {
		bounds := w.WidgetBase.Bounds()
		screen := w.WidgetBase.ScreenBounds()
		return w.dirtyRect.TranslateXY(screen.Min.X-bounds.Min.X, screen.Min.Y-bounds.Min.Y)
	}
	return w.WidgetBase.ScreenBounds()
}

func (w *minimapWidget) ClearRedraw() {
	w.WidgetBase.ClearRedraw()
	w.dirtyRect = geometry.Rect{}
}

func (w *minimapWidget) markPlayerDirty(old, current minimapPlayerMarkerState) bool {
	if w.ctx.World == nil || !current.valid {
		return false
	}
	bounds := w.Bounds()
	if bounds.IsEmpty() {
		return false
	}
	mapW, mapH := minimapWorldSize(w.ctx.World)
	if mapW <= 0 || mapH <= 0 {
		return false
	}
	mapRect := minimapContentMapRect(bounds)
	dirty := minimapPlayerMarkerDirtyRect(mapRect, mapW, mapH, current)
	if old.valid && old.mapID == current.mapID {
		dirty = dirty.Union(minimapPlayerMarkerDirtyRect(mapRect, mapW, mapH, old))
	}
	dirty = dirty.Union(minimapCoordsDirtyRect(bounds))
	dirty = dirty.Intersection(bounds)
	if dirty.IsEmpty() {
		return false
	}
	w.dirtyRect = dirty
	w.SetNeedsRedraw(true)
	return true
}

func minimapContentMapRect(bounds geometry.Rect) minimapRect {
	available := int(bounds.Height()) - minimapInfoBandH - minimapPad
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

func drawMinimapPlayerMarker(canvas widget.Canvas, rect minimapRect, mapW, mapH, cellX, cellY int, arrow image.Image) {
	x, y, ok := minimapCellToScreen(rect, mapW, mapH, cellX, cellY)
	if !ok {
		return
	}
	if arrow == nil {
		drawMinimapMarkerAt(canvas, x, y, minimapPlayerColor, 4)
		return
	}
	bounds := arrow.Bounds()
	canvas.DrawImage(arrow, geometry.Pt(float32(x-bounds.Dx()/2), float32(y-bounds.Dy()/2)))
}

func minimapPlayerMarkerDirtyRect(rect minimapRect, mapW, mapH int, marker minimapPlayerMarkerState) geometry.Rect {
	x, y, ok := minimapCellToScreen(rect, mapW, mapH, marker.x, marker.y)
	if !ok {
		return geometry.Rect{}
	}
	width, height := 11, 11
	if marker.arrow != nil {
		bounds := marker.arrow.Bounds()
		width = maxInt(width, bounds.Dx())
		height = maxInt(height, bounds.Dy())
	}
	return geometry.NewRect(float32(x-width/2), float32(y-height/2), float32(width), float32(height)).Expand(4)
}

func minimapCoordsDirtyRect(bounds geometry.Rect) geometry.Rect {
	footerY := bounds.Min.Y + bounds.Height() - 19
	return geometry.NewRect(bounds.Min.X+bounds.Width()/2-2, footerY-2, bounds.Width()/2-minimapPad+4, 20)
}

func drawMinimapMarkerAt(canvas widget.Canvas, x, y int, fill color.RGBA, radius int) {
	canvas.DrawRect(geometry.NewRect(float32(x-radius-1), float32(y-radius-1), float32(radius*2+3), float32(radius*2+3)), Color(color.RGBA{A: 190}))
	canvas.DrawRect(geometry.NewRect(float32(x-radius), float32(y-radius), float32(radius*2+1), float32(radius*2+1)), Color(fill))
}

func drawMinimapCompassMarkers(canvas widget.Canvas, rect minimapRect, mapW, mapH int, markers []minimapCompassMarker, now, started time.Time) {
	if len(markers) == 0 || !minimapCompassBlinkVisible(now, started) {
		return
	}
	for _, marker := range markers {
		x, y, ok := minimapCellToScreen(rect, mapW, mapH, marker.x, marker.y)
		if !ok {
			continue
		}
		drawMinimapCross(canvas, x, y, marker.color)
	}
}

func drawMinimapPartyMarkers(canvas widget.Canvas, rect minimapRect, mapW, mapH int, ctx Context) {
	if ctx.World == nil || ctx.Session == nil || !ctx.Session.Party.Active() {
		return
	}
	currentMap := normalizeMinimapMapName(ctx.World.MapName)
	for _, member := range ctx.Session.Party.Members {
		if member.AccountID == 0 || member.AccountID == ctx.Session.AccountID || !member.Online() || member.X < 0 || member.Y < 0 {
			continue
		}
		if memberMap := normalizeMinimapMapName(member.MapName); memberMap != "" && memberMap != currentMap {
			continue
		}
		x, y, ok := minimapCellToScreen(rect, mapW, mapH, member.X, member.Y)
		if !ok {
			continue
		}
		drawMinimapSquareCentered(canvas, x, y, 6, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		drawMinimapSquareCentered(canvas, x, y, 4, minimapMemberColor(member.AccountID))
	}
}

func drawMinimapGuildMarkers(canvas widget.Canvas, rect minimapRect, mapW, mapH int, ctx Context, markers []minimapGuildMarker) {
	if len(markers) == 0 {
		return
	}
	localAccountID := uint32(0)
	if ctx.Session != nil {
		localAccountID = ctx.Session.AccountID
	}
	for _, marker := range markers {
		if marker.accountID == localAccountID {
			continue
		}
		x, y, ok := minimapCellToScreen(rect, mapW, mapH, marker.x, marker.y)
		if !ok {
			continue
		}
		drawMinimapSquareCentered(canvas, x, y, 3, minimapGuildColor)
	}
}

func drawMinimapCross(canvas widget.Canvas, x, y int, fill color.RGBA) {
	canvas.DrawRect(geometry.NewRect(float32(x-1), float32(y-4), 2, 8), Color(fill))
	canvas.DrawRect(geometry.NewRect(float32(x-4), float32(y-1), 8, 2), Color(fill))
}

func drawMinimapSquareCentered(canvas widget.Canvas, x, y, size int, fill color.RGBA) {
	if size <= 0 {
		return
	}
	half := size / 2
	canvas.DrawRect(geometry.NewRect(float32(x-half), float32(y-half), float32(size), float32(size)), Color(fill))
}

func minimapCellToScreen(rect minimapRect, mapW, mapH, cellX, cellY int) (int, int, bool) {
	if mapW <= 0 || mapH <= 0 || rect.w <= 0 || rect.h <= 0 {
		return 0, 0, false
	}
	projected := minimapProjectedMapRect(rect, mapW, mapH)
	x := projected.x + int(float64(cellX)*float64(projected.w)/float64(mapW))
	y := projected.y + projected.h - int(float64(cellY)*float64(projected.h)/float64(mapH))
	return x, y, true
}

func minimapProjectedMapRect(rect minimapRect, mapW, mapH int) minimapRect {
	if mapW <= 0 || mapH <= 0 || rect.w <= 0 || rect.h <= 0 {
		return minimapRect{}
	}
	maxSide := maxInt(mapW, mapH)
	drawW := int(float64(rect.w)*float64(mapW)/float64(maxSide) + 0.5)
	drawH := int(float64(rect.h)*float64(mapH)/float64(maxSide) + 0.5)
	drawW = maxInt(1, minInt(rect.w, drawW))
	drawH = maxInt(1, minInt(rect.h, drawH))
	return minimapRect{
		x: rect.x + (rect.w-drawW)/2,
		y: rect.y + (rect.h-drawH)/2,
		w: drawW,
		h: drawH,
	}
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
	koreanInterface := "유저인터페이스"
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

func minimapArrowCandidates() []string {
	koreanInterface := "유저인터페이스"
	return []string{
		"data\\texture\\" + koreanInterface + "\\map\\map_arrow.bmp",
		"texture\\" + koreanInterface + "\\map\\map_arrow.bmp",
		"data\\texture\\interface\\map\\map_arrow.bmp",
		"texture\\interface\\map\\map_arrow.bmp",
		"data\\texture\\map\\map_arrow.bmp",
		"texture\\map\\map_arrow.bmp",
		"map_arrow.bmp",
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

func normalizeMinimapDirection(direction int) int {
	direction %= 8
	if direction < 0 {
		direction += 8
	}
	return direction
}

func minimapPlayerArrowAngle(direction int) float64 {
	robrDirection := minimapROBrowserDirection(direction)
	return float64((robrDirection+4)&7) * math.Pi / 4
}

func minimapROBrowserDirection(direction int) int {
	return [...]int{4, 3, 2, 1, 0, 7, 6, 5}[normalizeMinimapDirection(direction)]
}

func rotateMinimapArrow(src image.Image, angle float64) image.Image {
	if src == nil {
		return nil
	}
	bounds := src.Bounds()
	sw, sh := bounds.Dx(), bounds.Dy()
	if sw <= 0 || sh <= 0 {
		return nil
	}
	sin, cos := math.Sin(angle), math.Cos(angle)
	if math.Abs(sin) < 1e-12 {
		sin = 0
	}
	if math.Abs(cos) < 1e-12 {
		cos = 0
	}
	dw := int(math.Ceil(math.Abs(float64(sw)*cos) + math.Abs(float64(sh)*sin)))
	dh := int(math.Ceil(math.Abs(float64(sw)*sin) + math.Abs(float64(sh)*cos)))
	dw = maxInt(1, dw)
	dh = maxInt(1, dh)
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	scx := float64(sw-1) * 0.5
	scy := float64(sh-1) * 0.5
	dcx := float64(dw-1) * 0.5
	dcy := float64(dh-1) * 0.5
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			dx := float64(x) - dcx
			dy := float64(y) - dcy
			sx := cos*dx + sin*dy + scx
			sy := -sin*dx + cos*dy + scy
			ix := int(math.Round(sx))
			iy := int(math.Round(sy))
			if ix < 0 || iy < 0 || ix >= sw || iy >= sh {
				continue
			}
			c := color.NRGBAModel.Convert(src.At(bounds.Min.X+ix, bounds.Min.Y+iy)).(color.NRGBA)
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

func minimapCompassColor(rgb uint32) color.RGBA {
	return color.RGBA{
		R: uint8((rgb >> 16) & 0xff),
		G: uint8((rgb >> 8) & 0xff),
		B: uint8(rgb & 0xff),
		A: 255,
	}
}

func minimapCompassBlinkVisible(now, started time.Time) bool {
	return minimapCompassBlinkPhase(now, started) == 1
}

func minimapCompassBlinkPhase(now, started time.Time) int {
	if now.IsZero() {
		now = time.Now()
	}
	var elapsed time.Duration
	if !started.IsZero() {
		elapsed = now.Sub(started)
	} else {
		elapsed = time.Duration(now.UnixNano())
	}
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed.Milliseconds()%1000 > 500 {
		return 1
	}
	return 0
}

func minimapImageStateKey(img image.Image) string {
	if img == nil {
		return "nil"
	}
	bounds := img.Bounds()
	return fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())
}

func minimapMemberColor(accountID uint32) color.RGBA {
	x := accountID
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	return color.RGBA{
		R: uint8(48 + (x & 0xbf)),
		G: uint8(48 + ((x >> 8) & 0xbf)),
		B: uint8(48 + ((x >> 16) & 0xbf)),
		A: 255,
	}
}

func minimapDisplayName(mapName string) string {
	name := normalizeMinimapMapName(mapName)
	if name == "" {
		return "unknown"
	}
	return name
}
