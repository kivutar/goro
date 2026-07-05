package ui

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/ui/rotheme"
)

const CharacterCreateStatCount = 6

const (
	CharacterCreateStatStr = iota
	CharacterCreateStatAgi
	CharacterCreateStatVit
	CharacterCreateStatInt
	CharacterCreateStatDex
	CharacterCreateStatLuk
)

const (
	characterCreatePanelW = 142
	characterCreatePanelH = 166
	characterCreateGraphW = 166
	characterCreateListW  = 136
)

type CharacterCreateWindowOptions struct {
	X, Y, W, H int
	FooterH    int
	FooterPadX int
	FooterGap  int
	Name       string
	Stats      [CharacterCreateStatCount]uint8
	Preview    image.Image
}

type CharacterCreateWindowCallbacks struct {
	OnNameChange func(string)
	OnSubmit     func()
	OnCancel     func()
	OnHairPrev   func()
	OnHairNext   func()
	OnHairColor  func()
	OnStat       func(int)
}

type CharacterCreateWindow struct {
	opts      CharacterCreateWindowOptions
	callbacks CharacterCreateWindowCallbacks
	window    WindowState
	name      *textfield.Widget
}

func NewCharacterCreateWindow(opts CharacterCreateWindowOptions, callbacks CharacterCreateWindowCallbacks) *CharacterCreateWindow {
	w := &CharacterCreateWindow{
		opts:      opts,
		callbacks: callbacks,
		window:    NewWindowState(opts.W, opts.H),
	}
	w.window.OpenAt(opts.X, opts.Y, w.widgetTree())
	return w
}

func (w *CharacterCreateWindow) SetOptions(opts CharacterCreateWindowOptions) {
	if w == nil {
		return
	}
	sameTree := characterCreateWindowTreeEqual(w.opts, opts)
	w.opts = opts
	w.window.SetAutoPosition(opts.X, opts.Y)
	w.window.SetSize(opts.W, opts.H)
	if sameTree {
		return
	}
	focused := w.name != nil && w.name.IsFocused()
	w.window.SetContent(w.widgetTree())
	if w.name != nil {
		w.name.SetFocused(focused)
	}
}

func (w *CharacterCreateWindow) Widget() widget.Widget {
	if w == nil {
		return nil
	}
	return w.window.Widget()
}

func (w *CharacterCreateWindow) Update(ctx client.Context) bool {
	if w == nil {
		return false
	}
	return w.window.Update(ctx)
}

func characterCreateWindowTreeEqual(a, b CharacterCreateWindowOptions) bool {
	return a.W == b.W &&
		a.H == b.H &&
		a.FooterH == b.FooterH &&
		a.FooterPadX == b.FooterPadX &&
		a.FooterGap == b.FooterGap &&
		a.Preview == b.Preview &&
		a.Stats == b.Stats
}

func (w *CharacterCreateWindow) widgetTree() widget.Widget {
	nameValue := w.opts.Name
	if w.name != nil {
		nameValue = w.name.Text()
	}
	name := rotheme.TextField(
		nameValue,
		textfield.TypeText,
		func(value string) {
			if w.callbacks.OnNameChange != nil {
				w.callbacks.OnNameChange(value)
			}
		},
		func(string) {
			if w.callbacks.OnSubmit != nil {
				w.callbacks.OnSubmit()
			}
		},
		textfield.MaxLength(23),
	)
	if w.name == nil {
		name.SetFocused(true)
	} else if w.name.IsFocused() {
		name.SetFocused(true)
	}
	w.name = name

	buttonW := func(label string) float32 {
		return float32(ButtonLabelWidth(label))
	}
	return Window(
		Title("Make Character"),
		CloseButton(false),
		Size(float32(w.opts.W), float32(w.opts.H)),
		FooterHeight(float32(w.opts.FooterH)),
		FooterPadding(float32(w.opts.FooterPadX)),
		Content(
			primitives.Box(
				primitives.HBox(
					primitives.Box(
						newCharacterCreatePreview(w.opts.Preview, characterCreatePreviewCallbacks{
							prev:  w.callbacks.OnHairPrev,
							next:  w.callbacks.OnHairNext,
							color: w.callbacks.OnHairColor,
						}).
							Width(characterCreatePanelW).
							Height(characterCreatePanelH),
						primitives.Box(
							rotheme.Text("Name").
								LineHeight(22/rotheme.Default.Typography.TextSize),
							primitives.Box(name).
								Width(characterCreatePanelW).
								Height(22),
						).
							Gap(5).
							PaddingTop(8),
					).
						Width(characterCreatePanelW),
					newCharacterCreateStatGraph(w.opts.Stats, func(stat int) {
						if w.callbacks.OnStat != nil {
							w.callbacks.OnStat(stat)
						}
					}).
						Width(characterCreateGraphW).
						Height(characterCreatePanelH),
					primitives.Box(
						characterCreateStatListRow(0, w.opts.Stats[0], true),
						characterCreateStatListRow(1, w.opts.Stats[1], false),
						characterCreateStatListRow(2, w.opts.Stats[2], true),
						characterCreateStatListRow(3, w.opts.Stats[3], false),
						characterCreateStatListRow(4, w.opts.Stats[4], true),
						characterCreateStatListRow(5, w.opts.Stats[5], false),
					).
						Width(characterCreateListW).
						Height(characterCreatePanelH).
						Background(rotheme.Default.Colors.PanelBody).
						BorderStyle(1, rotheme.Default.Colors.WindowBorder).
						PaddingTop(13),
				).
					Gap(32).
					CrossAlign(primitives.CrossAxisStart),
			).
				PaddingTop(30).
				PaddingLeft(32).
				PaddingRight(32),
		),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Make", func() {
					if w.callbacks.OnSubmit != nil {
						w.callbacks.OnSubmit()
					}
				}).
					Width(buttonW("Make")),
				rotheme.Button("Cancel", func() {
					if w.callbacks.OnCancel != nil {
						w.callbacks.OnCancel()
					}
				}).
					Width(buttonW("Cancel")),
			).
				CrossAlign(primitives.CrossAxisCenter).
				Gap(float32(w.opts.FooterGap)),
		),
	)
}

func characterCreateStatListRow(stat int, value uint8, shaded bool) widget.Widget {
	bg := rotheme.Default.Colors.PanelBody
	if shaded {
		bg = rotheme.Default.Colors.WindowFooter
	}
	return primitives.HBox(
		primitives.Box(rotheme.Text(CharacterCreateStatLabels()[stat])).
			Width(72),
		primitives.Box(rotheme.Text(fmt.Sprintf("%d", value))).
			Width(32),
	).
		Height(22).
		PaddingXY(12, 0).
		CrossAlign(primitives.CrossAxisCenter).
		Background(bg)
}

func CharacterCreateGraphDrawOrder() [CharacterCreateStatCount]int {
	return [CharacterCreateStatCount]int{
		CharacterCreateStatStr,
		CharacterCreateStatVit,
		CharacterCreateStatLuk,
		CharacterCreateStatInt,
		CharacterCreateStatDex,
		CharacterCreateStatAgi,
	}
}

func CharacterCreateGraphPoints(cx, cy int, radius float64) [CharacterCreateStatCount][2]float64 {
	dirs := [CharacterCreateStatCount][2]float64{
		{0, -1},
		{-0.866, -0.5},
		{0.866, -0.5},
		{0, 1},
		{-0.866, 0.5},
		{0.866, 0.5},
	}
	points := [CharacterCreateStatCount][2]float64{}
	for i := range dirs {
		points[i][0] = float64(cx) + dirs[i][0]*radius
		points[i][1] = float64(cy) + dirs[i][1]*radius
	}
	return points
}

func CharacterCreateStatLabels() [CharacterCreateStatCount]string {
	return [CharacterCreateStatCount]string{"STR", "AGI", "VIT", "INT", "DEX", "LUK"}
}

type characterCreateStatGraph struct {
	widget.WidgetBase
	stats   [CharacterCreateStatCount]uint8
	onClick func(int)
	hovered int
	width   float32
	height  float32
}

type characterCreatePreviewCallbacks struct {
	prev  func()
	next  func()
	color func()
}

type characterCreatePreview struct {
	widget.WidgetBase
	image     image.Image
	callbacks characterCreatePreviewCallbacks
	hovered   int
	width     float32
	height    float32
}

func newCharacterCreatePreview(img image.Image, callbacks characterCreatePreviewCallbacks) *characterCreatePreview {
	w := &characterCreatePreview{
		image:     img,
		callbacks: callbacks,
		hovered:   -1,
		width:     characterCreatePanelW,
		height:    characterCreatePanelH,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *characterCreatePreview) Width(width float32) *characterCreatePreview {
	w.width = width
	return w
}

func (w *characterCreatePreview) Height(height float32) *characterCreatePreview {
	w.height = height
	return w
}

func (w *characterCreatePreview) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(w.width, w.height))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *characterCreatePreview) Draw(_ widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	canvas.DrawRect(bounds, rotheme.Default.Colors.PanelBody)
	canvas.StrokeRect(bounds, rotheme.Default.Colors.WindowBorder, 1)
	if w.image != nil {
		imgBounds := w.image.Bounds()
		x := bounds.Min.X + (bounds.Width()-float32(imgBounds.Dx()))/2
		y := bounds.Min.Y + (bounds.Height()-float32(imgBounds.Dy()))/2
		canvas.DrawImage(w.image, geometry.Pt(x, y))
	}
	for i, kind := range [...]rotheme.IconButtonKind{rotheme.IconButtonLeft, rotheme.IconButtonUp, rotheme.IconButtonRight} {
		rect := characterCreatePreviewButtonRect(bounds, i)
		rotheme.DrawIconButton(canvas, rect, kind, w.hovered == i, false)
	}
}

func (w *characterCreatePreview) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	hit := characterCreatePreviewButtonAt(w.Bounds(), mouse.Position)
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		w.setHovered(ctx, hit)
		return hit >= 0
	case event.MouseLeave:
		w.setHovered(ctx, -1)
		return false
	case event.MousePress:
		if mouse.Button != event.ButtonLeft || hit < 0 {
			return false
		}
		switch hit {
		case 0:
			if w.callbacks.prev != nil {
				w.callbacks.prev()
			}
		case 1:
			if w.callbacks.color != nil {
				w.callbacks.color()
			}
		case 2:
			if w.callbacks.next != nil {
				w.callbacks.next()
			}
		}
		return true
	}
	return hit >= 0
}

func (w *characterCreatePreview) setHovered(ctx widget.Context, button int) {
	if button >= 0 {
		ctx.SetCursor(widget.CursorPointer)
	} else {
		ctx.SetCursor(widget.CursorDefault)
	}
	if w.hovered == button {
		return
	}
	w.hovered = button
	w.SetNeedsRedraw(true)
}

func characterCreatePreviewButtonAt(bounds geometry.Rect, point geometry.Point) int {
	for i := 0; i < 3; i++ {
		if characterCreatePreviewButtonRect(bounds, i).Contains(point) {
			return i
		}
	}
	return -1
}

func characterCreatePreviewButtonRect(bounds geometry.Rect, index int) geometry.Rect {
	size := rotheme.IconButtonSize
	xs := [...]float32{bounds.Min.X + 16, bounds.Min.X + bounds.Width()/2 - size/2, bounds.Max.X - 16 - size}
	ys := [...]float32{bounds.Min.Y + 70, bounds.Min.Y + 16, bounds.Min.Y + 70}
	if index < 0 || index >= len(xs) {
		index = 0
	}
	return geometry.NewRect(xs[index], ys[index], size, size)
}

func newCharacterCreateStatGraph(stats [CharacterCreateStatCount]uint8, onClick func(int)) *characterCreateStatGraph {
	w := &characterCreateStatGraph{
		stats:   stats,
		onClick: onClick,
		hovered: -1,
		width:   characterCreateGraphW,
		height:  characterCreatePanelH,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *characterCreateStatGraph) Width(width float32) *characterCreateStatGraph {
	w.width = width
	return w
}

func (w *characterCreateStatGraph) Height(height float32) *characterCreateStatGraph {
	w.height = height
	return w
}

func (w *characterCreateStatGraph) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(w.width, w.height))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *characterCreateStatGraph) Draw(_ widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	canvas.DrawRect(bounds, rotheme.Default.Colors.PanelBody)
	canvas.StrokeRect(bounds, rotheme.Default.Colors.WindowBorder, 1)

	cx := bounds.Min.X + bounds.Width()/2
	cy := bounds.Min.Y + bounds.Height()/2
	outer := float32(58)
	inner := float32(29)
	points := characterCreateWidgetGraphPoints(cx, cy, outer)
	mid := characterCreateWidgetGraphPoints(cx, cy, inner)
	order := CharacterCreateGraphDrawOrder()
	for i := 0; i < CharacterCreateStatCount; i++ {
		current := order[i]
		next := order[(i+1)%CharacterCreateStatCount]
		canvas.DrawLine(points[current], points[next], rotheme.Default.Colors.FooterLine, 1)
		canvas.DrawLine(mid[current], mid[next], rotheme.Default.Colors.FooterLine, 1)
		canvas.DrawLine(geometry.Pt(cx, cy), points[current], widget.RGBA8(185, 204, 224, 150), 1)
	}
	if filler, ok := canvas.(widget.SVGFiller); ok {
		filler.FillSVGPath(w.statPolygonSVG(bounds.Width()/2, bounds.Height()/2, outer, order), maxFloat32(bounds.Width(), bounds.Height()), bounds, widget.RGBA8(36, 92, 154, 220))
	}

	for stat := 0; stat < CharacterCreateStatCount; stat++ {
		rect := characterCreateStatButtonRect(bounds, stat)
		bg := rotheme.Default.Colors.Button
		if stat == w.hovered {
			bg = rotheme.Default.Colors.ButtonHover
		}
		canvas.DrawRoundRect(rect, bg, rotheme.ButtonRadius)
		canvas.StrokeRoundRect(rect, rotheme.Default.Colors.ButtonBorder, rotheme.ButtonRadius, 1)
		rotheme.DrawText(canvas, CharacterCreateStatLabels()[stat], geometry.NewRect(rect.Min.X, rect.Min.Y+3, rect.Width(), 12), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignCenter)
		rotheme.DrawText(canvas, fmt.Sprintf("%d", w.stats[stat]), geometry.NewRect(rect.Min.X, rect.Min.Y+18, rect.Width(), 12), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignCenter)
	}
}

func (w *characterCreateStatGraph) statPolygonSVG(cx, cy, outer float32, order [CharacterCreateStatCount]int) string {
	points := characterCreateWidgetGraphPoints(cx, cy, outer)
	var b strings.Builder
	for i, stat := range order {
		scale := float32(0.22) + float32(w.stats[stat])/9*0.78
		x := cx + (points[stat].X-cx)*scale
		y := cy + (points[stat].Y-cy)*scale
		if i == 0 {
			fmt.Fprintf(&b, "M%.2f %.2f", x, y)
		} else {
			fmt.Fprintf(&b, "L%.2f %.2f", x, y)
		}
	}
	b.WriteString("Z")
	return b.String()
}

func (w *characterCreateStatGraph) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	hit := characterCreateStatButtonAt(w.Bounds(), mouse.Position)
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		w.setHovered(ctx, hit)
		return hit >= 0
	case event.MouseLeave:
		w.setHovered(ctx, -1)
		return false
	case event.MousePress:
		if mouse.Button != event.ButtonLeft || hit < 0 {
			return false
		}
		if w.onClick != nil {
			w.onClick(hit)
		}
		return true
	}
	return hit >= 0
}

func (w *characterCreateStatGraph) setHovered(ctx widget.Context, stat int) {
	if stat >= 0 {
		ctx.SetCursor(widget.CursorPointer)
	} else {
		ctx.SetCursor(widget.CursorDefault)
	}
	if w.hovered == stat {
		return
	}
	w.hovered = stat
	w.SetNeedsRedraw(true)
}

func characterCreateWidgetGraphPoints(cx, cy, radius float32) [CharacterCreateStatCount]geometry.Point {
	model := CharacterCreateGraphPoints(0, 0, float64(radius))
	points := [CharacterCreateStatCount]geometry.Point{}
	for i := range model {
		points[i] = geometry.Pt(cx+float32(model[i][0]), cy+float32(model[i][1]))
	}
	return points
}

func characterCreateStatButtonAt(bounds geometry.Rect, point geometry.Point) int {
	for stat := 0; stat < CharacterCreateStatCount; stat++ {
		if characterCreateStatButtonRect(bounds, stat).Contains(point) {
			return stat
		}
	}
	return -1
}

func characterCreateStatButtonRect(bounds geometry.Rect, stat int) geometry.Rect {
	cx := bounds.Min.X + bounds.Width()/2
	top := bounds.Min.Y + 5
	midTop := bounds.Min.Y + 47
	midBottom := bounds.Min.Y + 92
	bottom := bounds.Max.Y - 41
	left := bounds.Min.X + 8
	right := bounds.Max.X - 46
	rects := [CharacterCreateStatCount]geometry.Rect{
		geometry.NewRect(cx-19, top, 38, 36),
		geometry.NewRect(left, midTop, 38, 36),
		geometry.NewRect(right, midTop, 38, 36),
		geometry.NewRect(cx-19, bottom, 38, 36),
		geometry.NewRect(left, midBottom, 38, 36),
		geometry.NewRect(right, midBottom, 38, 36),
	}
	if stat < 0 || stat >= len(rects) {
		stat = 0
	}
	return rects[stat]
}

func maxFloat32(a, b float32) float32 {
	return float32(math.Max(float64(a), float64(b)))
}
