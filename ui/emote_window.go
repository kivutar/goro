package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	emoteWindowWidth      = 250
	emoteWindowHeight     = 330
	emoteCols             = 5
	emoteRows             = 6
	emoteIconsPerPage     = emoteCols * emoteRows
	emoteCellSize         = 40
	emoteCellGap          = 4
	emoteContentPadX      = 10
	emoteContentPadY      = 6
	emoteNavHeight        = 24
	emoteDefaultX         = 600
	emoteDefaultY         = 200
	emoteDoubleClickDelay = 360
)

type EmoteWindow struct {
	Window
	// OnSelect can handle a selection in another editor instead of the console.
	OnSelect   func(command string) bool
	page       int
	contentKey string
	sprite     emoteSpriteSet
}

func (w *EmoteWindow) Toggle(ctx Context, console *ChatConsole) {
	w.EnsureWindow(emoteWindowWidth, emoteWindowHeight)
	if w.IsOpen() {
		w.Close()
		return
	}
	w.OpenWindow(ctx, console)
}

func (w *EmoteWindow) OpenWindow(ctx Context, console *ChatConsole) {
	w.EnsureWindow(emoteWindowWidth, emoteWindowHeight)
	w.ctx = ctx
	w.page = clampEmotePage(w.page)
	w.contentKey = w.renderKey(ctx)
	x, y := emoteDefaultPosition(ctx)
	w.OpenAt(x, y, w.widgetTree(ctx, console))
	w.Publish(ctx)
}

func (w *EmoteWindow) Update(ctx Context, console *ChatConsole) bool {
	w.EnsureWindow(emoteWindowWidth, emoteWindowHeight)
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	nextKey := w.renderKey(ctx)
	if nextKey != w.contentKey {
		w.contentKey = nextKey
		w.SetContent(w.widgetTree(ctx, console))
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *EmoteWindow) Rebind(ctx Context, console *ChatConsole) {
	w.EnsureWindow(emoteWindowWidth, emoteWindowHeight)
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.contentKey = w.renderKey(ctx)
	w.SetContent(w.widgetTree(ctx, console))
	w.Publish(ctx)
}

func (w *EmoteWindow) widgetTree(ctx Context, console *ChatConsole) widget.Widget {
	return Win(
		Title("Emotion icon List"),
		CloseButton(true),
		OnClose(w.Close),
		Size(emoteWindowWidth, emoteWindowHeight),
		Content(
			primitives.Box(
				w.gridWidget(ctx, console),
				w.navWidget(ctx, console),
			).
				PaddingXY(emoteContentPadX, emoteContentPadY).
				Gap(4).
				CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *EmoteWindow) gridWidget(ctx Context, console *ChatConsole) widget.Widget {
	emotes := db.EmotionList()
	page := clampEmotePage(w.page)
	start := page * emoteIconsPerPage
	end := minInt(len(emotes), start+emoteIconsPerPage)
	rows := make([]widget.Widget, 0, emoteRows)
	for row := 0; row < emoteRows; row++ {
		cells := make([]widget.Widget, 0, emoteCols)
		for col := 0; col < emoteCols; col++ {
			index := start + row*emoteCols + col
			if index < end {
				emote := emotes[index]
				entry := emote
				cells = append(cells, newEmoteCellWidget(
					entry,
					w.sprite.icon(ctx.Resources, entry.Frame),
					func() {
						w.selectEmotion(ctx, console, entry)
					},
					func() {
						w.playEmotion(ctx, console, entry)
					},
				))
				continue
			}
			cells = append(cells, primitives.Box().Width(emoteCellSize).Height(emoteCellSize))
		}
		rows = append(rows,
			primitives.HBox(cells...).
				Gap(emoteCellGap).
				CrossAlign(primitives.CrossAxisStretch),
		)
	}
	return primitives.Box(rows...).
		Width(float32(emoteCols*emoteCellSize + (emoteCols-1)*emoteCellGap)).
		Height(float32(emoteRows*emoteCellSize + (emoteRows-1)*emoteCellGap)).
		Gap(emoteCellGap).
		CrossAlign(primitives.CrossAxisStretch)
}

func (w *EmoteWindow) navWidget(ctx Context, console *ChatConsole) widget.Widget {
	total := emotePageCount()
	page := clampEmotePage(w.page)
	w.page = page
	return primitives.HBox(
		rotheme.ButtonDisabled("Prev", page <= 0, func() {
			w.movePage(ctx, console, -1)
		}).Width(48).Height(22),
		primitives.Box(
			rotheme.Text(fmt.Sprintf("%d / %d", page+1, total)).
				MaxLines(1),
		).
			Width(56).
			Height(22).
			PaddingTop(4).
			CrossAlign(primitives.CrossAxisCenter),
		rotheme.ButtonDisabled("Next", page >= total-1, func() {
			w.movePage(ctx, console, 1)
		}).Width(48).Height(22),
	).
		Height(emoteNavHeight).
		Gap(8).
		CrossAlign(primitives.CrossAxisCenter)
}

func (w *EmoteWindow) movePage(ctx Context, console *ChatConsole, delta int) {
	w.page = clampEmotePage(w.page + delta)
	w.contentKey = w.renderKey(ctx)
	w.SetContent(w.widgetTree(ctx, console))
	w.Publish(ctx)
}

func (w *EmoteWindow) selectEmotion(ctx Context, console *ChatConsole, emote db.Emotion) {
	command := strings.TrimSpace(emote.Command)
	if command == "" {
		return
	}
	if w.OnSelect != nil && w.OnSelect("/"+command) {
		return
	}
	if console == nil {
		return
	}
	console.setInput("/" + command)
	console.setActive(true)
	console.Publish(ctx)
}

func (w *EmoteWindow) playEmotion(ctx Context, console *ChatConsole, emote db.Emotion) {
	command := strings.TrimSpace(emote.Command)
	if command == "" {
		return
	}
	if w.OnSelect != nil && w.OnSelect("/"+command) {
		return
	}
	if console != nil {
		console.setInput("/" + command)
		console.setActive(true)
		console.submit(ctx)
		console.Publish(ctx)
		return
	}
	if ctx.Network != nil {
		_ = ctx.Network.SendEmotion(emote.ID)
	}
}

func (w *EmoteWindow) renderKey(ctx Context) string {
	resourceKey := ""
	if ctx.Resources != nil {
		resourceKey = ctx.Resources.Root
	}
	return fmt.Sprintf("%s:%d", resourceKey, clampEmotePage(w.page))
}

func emoteDefaultPosition(ctx Context) (int, int) {
	screenW, screenH := ctx.ScreenSize()
	maxX := maxInt(windowScreenMargin, screenW-emoteWindowWidth-windowScreenMargin)
	maxY := maxInt(windowScreenMargin, screenH-emoteWindowHeight-windowScreenMargin)
	x := clampWindowInt(emoteDefaultX, windowScreenMargin, maxX)
	y := clampWindowInt(emoteDefaultY, windowScreenMargin, maxY)
	return x, y
}

func emotePageCount() int {
	count := len(db.EmotionList())
	pages := (count + emoteIconsPerPage - 1) / emoteIconsPerPage
	if pages < 1 {
		return 1
	}
	return pages
}

func clampEmotePage(page int) int {
	maxPage := emotePageCount() - 1
	if page < 0 {
		return 0
	}
	if page > maxPage {
		return maxPage
	}
	return page
}

type emoteCellWidget struct {
	widget.WidgetBase
	emote     db.Emotion
	icon      image.Image
	onSelect  func()
	onPlay    func()
	hovered   bool
	lastClick int64
	lastPlay  int64
}

func newEmoteCellWidget(emote db.Emotion, icon image.Image, onSelect, onPlay func()) *emoteCellWidget {
	w := &emoteCellWidget{
		emote:    emote,
		icon:     icon,
		onSelect: onSelect,
		onPlay:   onPlay,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *emoteCellWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(emoteCellSize, emoteCellSize))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *emoteCellWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	if w.hovered {
		canvas.DrawRoundRect(bounds, rotheme.Default.Colors.PanelBody, 5)
		canvas.StrokeRoundRect(bounds, rotheme.Default.Colors.FooterLine, 5, 1)
	}
	if w.icon != nil {
		imgBounds := w.icon.Bounds()
		x := bounds.Min.X + (bounds.Width()-float32(imgBounds.Dx()))/2
		y := bounds.Min.Y + (bounds.Height()-float32(imgBounds.Dy()))/2
		canvas.DrawImage(w.icon, geometry.Pt(x, y))
		return
	}
	rotheme.DrawText(
		canvas,
		"/"+w.emote.Command,
		bounds,
		rotheme.Default.Typography.TextSize,
		rotheme.Default.Colors.MutedText,
		false,
		widget.TextAlignCenter,
	)
}

func (w *emoteCellWidget) Event(ctx widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove:
		w.setHovered(ctx, true)
		ctx.SetCursor(widget.CursorPointer)
		return true
	case event.MouseLeave:
		w.setHovered(ctx, false)
		ctx.SetCursor(widget.CursorDefault)
		return true
	case event.MouseDoubleClick:
		if mouse.Button == event.ButtonLeft {
			w.play(mouse.Time().UnixMilli())
		}
		return true
	case event.MousePress:
		if mouse.Button != event.ButtonLeft {
			return true
		}
		now := mouse.Time().UnixMilli()
		if w.lastClick > 0 && now-w.lastClick <= int64(emoteDoubleClickDelay) {
			w.lastClick = 0
			w.play(now)
			return true
		}
		w.lastClick = now
		if w.onSelect != nil {
			w.onSelect()
		}
		return true
	}
	return true
}

func (w *emoteCellWidget) play(now int64) {
	if w.onPlay == nil {
		return
	}
	if w.lastPlay > 0 && now-w.lastPlay <= int64(emoteDoubleClickDelay) {
		return
	}
	w.lastPlay = now
	w.onPlay()
}

func (w *emoteCellWidget) setHovered(ctx widget.Context, hovered bool) {
	if w.hovered == hovered {
		return
	}
	w.hovered = hovered
	w.MarkRedrawLocal()
	ctx.InvalidateRect(w.Bounds())
}

type emoteSpriteSet struct {
	root      string
	act       *res.ACT
	spr       *res.SPR
	frames    map[emoteFrameKey]*render.Image
	icons     map[int]image.Image
	miss      bool
	loadedLog bool
}

type emoteFrameKey struct {
	index   int32
	sprType int32
}

func (s *emoteSpriteSet) icon(manager *res.Manager, frame int) image.Image {
	if manager == nil || frame < 0 {
		return nil
	}
	if s.root != manager.Root {
		*s = emoteSpriteSet{root: manager.Root}
	}
	if s.icons != nil {
		if icon, ok := s.icons[frame]; ok {
			return icon
		}
	}
	if !s.ensure(manager) {
		return nil
	}
	icon := s.composeIcon(frame)
	if icon == nil {
		return nil
	}
	if s.icons == nil {
		s.icons = make(map[int]image.Image)
	}
	s.icons[frame] = icon
	return icon
}

func (s *emoteSpriteSet) ensure(manager *res.Manager) bool {
	if s.act != nil && s.spr != nil {
		return true
	}
	if s.miss {
		return false
	}
	actSource, actData, actOK := manager.ReadFirst(emoteSpriteResourceCandidates("act"))
	if !actOK {
		s.miss = true
		glog.Warnf("emotion window act unavailable")
		return false
	}
	sprSource, sprData, sprOK := manager.ReadFirst(emoteSpriteResourceCandidates("spr"))
	if !sprOK {
		s.miss = true
		glog.Warnf("emotion window spr unavailable")
		return false
	}
	act, err := res.ParseACT(actData)
	if err != nil {
		s.miss = true
		glog.Warnf("emotion window act parse %s: %v", actSource, err)
		return false
	}
	spr, err := res.ParseSPR(sprData)
	if err != nil {
		s.miss = true
		glog.Warnf("emotion window spr parse %s: %v", sprSource, err)
		return false
	}
	s.act = act
	s.spr = spr
	s.frames = make(map[emoteFrameKey]*render.Image)
	if !s.loadedLog {
		glog.Debugf("emotion window resources act=%s spr=%s actions=%d frames=%d", actSource, sprSource, len(act.Actions), len(spr.Frames))
		s.loadedLog = true
	}
	return true
}

func emoteSpriteResourceCandidates(ext string) []string {
	path := "data\\sprite\\이팩트\\emotion." + ext
	return []string{path, strings.ReplaceAll(path, "\\", "/")}
}

func (s *emoteSpriteSet) composeIcon(frame int) image.Image {
	if s.act == nil || s.spr == nil || frame < 0 || frame >= len(s.act.Actions) {
		return nil
	}
	action := s.act.Actions[frame]
	if len(action.Animations) == 0 {
		return nil
	}
	motion := len(action.Animations) / 5
	if motion < 0 || motion >= len(action.Animations) {
		motion = 0
	}
	anim := action.Animations[motion]
	if len(anim.Layers) == 0 {
		return nil
	}
	first := anim.Layers[0]
	target := render.NewImage(emoteCellSize, emoteCellSize)
	centerX, centerY := emoteIconAnchor(first)
	rendered := false
	for _, layer := range anim.Layers {
		if layer.Index < 0 {
			continue
		}
		img := s.frameImage(layer.Index, layer.SPRType)
		if img == nil {
			continue
		}
		drawUISpriteLayer(target, img, layer, centerX, centerY)
		rendered = true
	}
	if !rendered {
		return nil
	}
	return target.RGBA()
}

func emoteIconAnchor(first res.ACTLayer) (float64, float64) {
	return float64(emoteCellSize/2) - float64(first.X),
		float64(emoteCellSize) - float64(first.Y) - uiSpriteRendererMidCellY
}

func (s *emoteSpriteSet) frameImage(index int32, sprType int32) *render.Image {
	key := emoteFrameKey{index: index, sprType: sprType}
	if img, ok := s.frames[key]; ok {
		return img
	}
	frame, ok := s.spr.FrameImage(int(index), int(sprType))
	if !ok {
		return nil
	}
	img := render.NewImageFromImage(frame)
	s.frames[key] = img
	return img
}
