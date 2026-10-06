package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/progressbar"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	characterWindowX                     = windowScreenMargin
	characterWindowY                     = windowScreenMargin
	characterWindowWidth                 = 324
	characterWindowHeight                = 140
	characterWindowCompactHeight         = 80
	characterEXPPanelPaddingX    float32 = 6
	characterEXPPanelPaddingY    float32 = 4
	characterEXPPanelGap         float32 = 2
	characterEXPPanelRadius      float32 = 5
	characterEXPLabelWidth       float32 = 76
	characterEXPLabelBarGap      float32 = 6
	characterEXPBarHeight        float32 = 6
	characterTextLineHeight      float32 = 1.2
	characterVitalsLabelWidth    float32 = 22
	characterVitalsPercentWidth  float32 = 34
	characterVitalsGap           float32 = 6
)

var (
	characterWindowEXPColor    = WindowBorderColor
	characterWindowJobEXPColor = WindowBorderColor
)

type CharacterWindow struct {
	Window
	snapshot string
	compact  bool
	title    state.Signal[string]
	body     *characterInfoBody
}

func (w *CharacterWindow) Update(ctx Context) bool {
	w.EnsureWindow(characterWindowWidth, w.windowHeight())
	w.CloseOnEsc = false
	if ctx.Session == nil {
		w.Close()
		w.Publish(ctx)
		return false
	}
	if !w.IsOpen() {
		w.snapshot = characterWindowSnapshot(ctx.Session)
		w.OpenAt(characterWindowX, characterWindowY, w.widgetTree(ctx))
	}
	nextSnapshot := characterWindowSnapshot(ctx.Session)
	if nextSnapshot != w.snapshot {
		w.snapshot = nextSnapshot
		if title := characterWindowTitle(ctx.Session, w.compact); title != w.title.Get() {
			w.title.Set(title)
		}
		w.body.replace(windowWidgetContext(ctx), w.bodyTree(ctx))
		invalidateWindowLayout(ctx)
		if overlay := w.positionedOverlay(); overlay != nil {
			invalidateWindowRect(ctx, overlay.markFrameDirty())
		}
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *CharacterWindow) Rebind(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.snapshot = characterWindowSnapshot(ctx.Session)
	w.RebindContent(ctx, w.widgetTree(ctx))
}

func (w *CharacterWindow) windowHeight() int {
	if w.compact {
		return characterWindowCompactHeight
	}
	return characterWindowHeight
}

func (w *CharacterWindow) ToggleCompact() {
	if !w.IsOpen() {
		return
	}
	w.compact = !w.compact
	w.SetSize(characterWindowWidth, w.windowHeight())
	if !w.compact {
		_, screenH := w.ctx.ScreenSize()
		maxY := maxInt(windowScreenMargin, screenH-w.height-w.dragBottom-windowScreenMargin)
		w.setPosition(w.ctx, w.x, min(w.y, maxY))
	}
	w.SetContent(w.widgetTree(w.ctx))
	w.Publish(w.ctx)
}

func (w *CharacterWindow) widgetTree(ctx Context) widget.Widget {
	kind := rotheme.IconButtonMinus
	if w.compact {
		kind = rotheme.IconButtonPlus
	}
	w.title = state.NewSignal(characterWindowTitle(ctx.Session, w.compact))
	w.body = newCharacterInfoBody(w.bodyTree(ctx))
	return Win(
		TitleSignal(w.title),
		TitleButton(kind, w.ToggleCompact),
		CloseButton(false),
		Size(float32(characterWindowWidth), float32(w.windowHeight())),
		Content(w.body),
	)
}

func characterWindowTitle(s *session.Session, compact bool) string {
	character := selectedCharacter(s)
	name := strings.TrimSpace(character.Name)
	if name == "" {
		name = "Player"
	}
	jobName := strings.TrimSpace(db.JobDisplayName(int(character.Job)))
	title := trimRunes(name, 20)
	if !compact && jobName != "" {
		title = trimRunes(fmt.Sprintf("%s (%s)", name, jobName), 32)
	}
	return title
}

func (w *CharacterWindow) bodyTree(ctx Context) *primitives.BoxWidget {
	character, vitals, progress, inventory := characterWindowData(ctx.Session)
	if w.compact {
		// Classic BasicInfo's small view keeps numeric vitals and a level/EXP
		// summary, without the bars, weight, or zeny.
		jobName := strings.TrimSpace(db.JobDisplayName(int(character.Job)))
		baseEXP := math.Floor(ratioInt64(progress.BaseExp, progress.NextBaseExp)*1000) / 10
		return primitives.Box(
			rotheme.Text(fmt.Sprintf("Lv. %d / %s / Lv. %d / Exp. %.1f%%", progress.BaseLevel, jobName, progress.JobLevel, baseEXP)).
				MaxLines(1).Ellipsis(),
			primitives.HBox(
				characterTextCell(fmt.Sprintf("HP %d / %d", vitals.HP, vitals.MaxHP), 146, rotheme.Default.Colors.MutedText),
				characterTextCell(fmt.Sprintf("SP %d / %d", vitals.SP, vitals.MaxSP), 146, rotheme.Default.Colors.MutedText),
			).Gap(8),
		).PaddingXY(12, 9).Gap(4).CrossAlign(primitives.CrossAxisStretch)
	}
	weightColor := rotheme.Default.Colors.Text
	if inventory.MaxWeight > 0 && inventory.Weight*100 >= inventory.MaxWeight*50 {
		weightColor = Color(ErrorTextColor)
	}
	return primitives.Box(
		primitives.Box(
			characterRatioRow("HP", vitals.HP, vitals.MaxHP, characterWindowWidth-24),
			characterRatioRow("SP", vitals.SP, vitals.MaxSP, characterWindowWidth-24),
		).Gap(4),
		characterEXPPanel(progress, characterWindowWidth-24),
		primitives.HBox(
			characterAlignedTextCell(fmt.Sprintf("Weight : %d / %d", displayWeight(inventory.Weight), displayWeight(inventory.MaxWeight)), 146, weightColor, primitives.TextAlignStart),
			characterAlignedTextCell(fmt.Sprintf("Zeny : %s", formatHUDNumber(inventory.Zeny)), 146, rotheme.Default.Colors.Text, primitives.TextAlignEnd),
		),
	).PaddingXY(12, 9).Gap(8)
}

// characterInfoBody lets the read-only stats refresh without unmounting the
// header and cancelling an in-progress click on its mode button.
type characterInfoBody struct {
	widget.WidgetBase
	child *primitives.BoxWidget
}

func newCharacterInfoBody(child *primitives.BoxWidget) *characterInfoBody {
	b := &characterInfoBody{child: child}
	b.SetVisible(true)
	child.SetParent(b)
	return b
}

func (b *characterInfoBody) replace(ctx widget.Context, child *primitives.BoxWidget) {
	widget.UnmountTree(b.child)
	b.child = child
	child.SetParent(b)
	if b.IsMounted() && ctx != nil {
		widget.MountTree(child, ctx)
	}
	b.MarkNeedsLayout()
	b.SetNeedsRedraw(true)
}

func (b *characterInfoBody) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := widget.LayoutChild(b.child, ctx, constraints)
	b.child.SetBounds(geometry.FromPointSize(geometry.Point{}, size))
	b.SetBounds(geometry.FromPointSize(b.Position(), size))
	return size
}

func (b *characterInfoBody) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !b.IsVisible() {
		return
	}
	canvas.PushTransform(b.Bounds().Min)
	widget.StampScreenOrigin(b.child, canvas)
	widget.DrawChild(b.child, ctx, canvas)
	canvas.PopTransform()
}

func (b *characterInfoBody) Event(widget.Context, event.Event) bool { return false }

func (b *characterInfoBody) Children() []widget.Widget {
	return []widget.Widget{b.child}
}

func characterTextCell(text string, width float32, color widget.Color) widget.Widget {
	return characterAlignedTextCell(text, width, color, primitives.TextAlignStart)
}

func characterAlignedTextCell(text string, width float32, color widget.Color, align primitives.TextAlign) widget.Widget {
	return primitives.Box(
		rotheme.Text(text).
			Color(color).
			Align(align),
	).Width(width).CrossAlign(primitives.CrossAxisStretch)
}

func characterWindowData(s *session.Session) (session.Character, session.Vitals, session.Progress, session.Inventory) {
	character := selectedCharacter(s)
	if s == nil {
		return character, session.Vitals{}, session.Progress{}, session.Inventory{}
	}
	vitals := s.Vitals
	if vitals.HP == 0 && vitals.MaxHP == 0 && vitals.SP == 0 && vitals.MaxSP == 0 {
		vitals = sessionVitalsFromCharacter(character)
	}
	progress := s.Progress
	if progress.BaseLevel == 0 {
		progress = sessionProgressFromCharacter(character)
		progress.BaseExp = s.Progress.BaseExp
		progress.NextBaseExp = s.Progress.NextBaseExp
		progress.JobExp = s.Progress.JobExp
		progress.NextJobExp = s.Progress.NextJobExp
	}
	if progress.JobLevel == 0 && character.JobLevel > 0 {
		progress.JobLevel = int(character.JobLevel)
	}
	return character, vitals, progress, s.Inventory
}

func characterWindowSnapshot(s *session.Session) string {
	character, vitals, progress, inventory := characterWindowData(s)
	return fmt.Sprintf(
		"name=%s;job=%d;%s;hp=%d/%d;sp=%d/%d;bl=%d;jl=%d;bexp=%d/%d;jexp=%d/%d;zeny=%d;weight=%d/%d",
		character.Name,
		character.Job,
		db.JobDisplayName(int(character.Job)),
		vitals.HP,
		vitals.MaxHP,
		vitals.SP,
		vitals.MaxSP,
		progress.BaseLevel,
		progress.JobLevel,
		progress.BaseExp,
		progress.NextBaseExp,
		progress.JobExp,
		progress.NextJobExp,
		inventory.Zeny,
		inventory.Weight,
		inventory.MaxWeight,
	)
}

func characterRatioRow(label string, current, maxValue int, width float32) widget.Widget {
	ratio := ratioInt(current, maxValue)
	values := fmt.Sprintf("%d / %d", current, maxValue)
	return primitives.HBox(
		characterVitalsText(label, characterVitalsLabelWidth, primitives.TextAlignStart),
		primitives.Expanded(rotheme.ProgressBar(
			progressbar.Value(ratio),
			progressbar.ShowLabel(true),
			progressbar.FormatLabelFn(func(float64) string { return values }),
		)),
		characterVitalsText(fmt.Sprintf("%.0f%%", ratio*100), characterVitalsPercentWidth, primitives.TextAlignEnd),
	).Width(width).Gap(characterVitalsGap).CrossAlign(primitives.CrossAxisCenter)
}

func characterVitalsText(text string, width float32, align primitives.TextAlign) widget.Widget {
	return primitives.Box(
		rotheme.Text(text).Align(align).
			LineHeight(rotheme.ProgressBarHeight / rotheme.Default.Typography.TextSize),
	).Width(width).CrossAlign(primitives.CrossAxisStretch)
}

func characterLevelProgressRow(label string, level int, current, next int64, fill widget.Color, width float32) widget.Widget {
	barWidth := max(float32(0), width-characterEXPLabelWidth-characterEXPLabelBarGap)
	textHeight := rotheme.Default.Typography.TextSize * characterTextLineHeight
	barTop := max(float32(0), (textHeight-characterEXPBarHeight)/2)
	return primitives.HBox(
		characterTextCell(fmt.Sprintf("%s Lv. %d", label, level), characterEXPLabelWidth, rotheme.Default.Colors.Text),
		primitives.Box(
			newCharacterBarWidgetWithBackground(ratioInt64(current, next), fill, widget.ColorWhite, barWidth, characterEXPBarHeight),
		).PaddingTop(barTop),
	).
		Width(width).
		Gap(characterEXPLabelBarGap)
}

func characterEXPPanel(progress session.Progress, width float32) widget.Widget {
	rowWidth := max(float32(0), width-2*characterEXPPanelPaddingX)
	return primitives.Box(
		characterLevelProgressRow("Base", progress.BaseLevel, progress.BaseExp, progress.NextBaseExp, Color(characterWindowEXPColor), rowWidth),
		characterLevelProgressRow("Job", progress.JobLevel, progress.JobExp, progress.NextJobExp, Color(characterWindowJobEXPColor), rowWidth),
	).
		Width(width).
		PaddingXY(characterEXPPanelPaddingX, characterEXPPanelPaddingY).
		Gap(characterEXPPanelGap).
		Background(rotheme.Default.Colors.WindowFooter).
		Rounded(characterEXPPanelRadius)
}

type characterBarWidget struct {
	widget.WidgetBase
	ratio      float64
	fill       widget.Color
	background widget.Color
	width      float32
	height     float32
}

func newCharacterBarWidgetWithBackground(ratio float64, fill, background widget.Color, width, height float32) *characterBarWidget {
	w := &characterBarWidget{ratio: ratio, fill: fill, background: background, width: width, height: height}
	w.SetVisible(true)
	w.SetEnabled(false)
	return w
}

func (w *characterBarWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(w.width, w.height))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *characterBarWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	canvas.DrawRect(bounds, w.background)
	if w.ratio > 0 {
		fillW := float32(math.Round(float64(bounds.Width()) * w.ratio))
		if fillW < 1 {
			fillW = 1
		}
		if fillW > bounds.Width() {
			fillW = bounds.Width()
		}
		canvas.DrawRect(geometry.NewRect(bounds.Min.X, bounds.Min.Y, fillW, bounds.Height()), w.fill)
	}
	canvas.StrokeRect(bounds, rotheme.Default.Colors.WindowBorder, 1)
}

func (w *characterBarWidget) Event(ctx widget.Context, e event.Event) bool {
	return false
}

func displayWeight(raw int) int {
	return raw / 10
}

func DisplayWeight(raw int) int {
	return displayWeight(raw)
}

func ratioInt(current, maxValue int) float64 {
	if maxValue <= 0 {
		return 0
	}
	return clampUnit(float64(current) / float64(maxValue))
}

func ratioInt64(current, maxValue int64) float64 {
	if maxValue <= 0 {
		return 0
	}
	return clampUnit(float64(current) / float64(maxValue))
}

func sessionVitalsFromCharacter(character session.Character) session.Vitals {
	return session.Vitals{
		HP:    int(character.HP),
		MaxHP: int(character.MaxHP),
		SP:    int(character.SP),
		MaxSP: int(character.MaxSP),
	}
}

func sessionProgressFromCharacter(character session.Character) session.Progress {
	return session.Progress{
		BaseLevel: int(character.Level),
		JobLevel:  int(character.JobLevel),
	}
}

func formatEXPPercent(current, next int64) string {
	if next <= 0 {
		return "--"
	}
	percent := 100 * float64(current) / float64(next)
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return fmt.Sprintf("%.1f%%", math.Floor(percent*10)/10)
}

func FormatEXPPercent(current, next int64) string {
	return formatEXPPercent(current, next)
}

func formatHUDNumber(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	text := strconv.FormatInt(value, 10)
	if len(text) <= 3 {
		return sign + text
	}
	var b strings.Builder
	prefix := len(text) % 3
	if prefix == 0 {
		prefix = 3
	}
	b.WriteString(text[:prefix])
	for i := prefix; i < len(text); i += 3 {
		b.WriteByte(',')
		b.WriteString(text[i : i+3])
	}
	return sign + b.String()
}

func FormatHUDNumber(value int64) string {
	return formatHUDNumber(value)
}
