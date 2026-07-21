package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	homunculusInfoWindowW     = 300
	homunculusInfoWindowH     = 320
	homunculusInfoContentPad  = 10
	homunculusInfoContentW    = homunculusInfoWindowW - homunculusInfoContentPad*2
	homunculusInfoColumnGap   = 10
	homunculusInfoRowH        = 17
	homunculusInfoNameH       = 24
	homunculusInfoStatLabelW  = 40
	homunculusInfoStatValueW  = 40
	homunculusInfoLeftW       = homunculusInfoStatLabelW + rotheme.TableGap + homunculusInfoStatValueW
	homunculusInfoRightW      = homunculusInfoContentW - homunculusInfoLeftW - homunculusInfoColumnGap
	homunculusInfoInfoLabelW  = 50
	homunculusInfoInfoGap     = 4
	homunculusInfoInfoValueW  = homunculusInfoRightW - homunculusInfoInfoLabelW - homunculusInfoInfoGap
	homunculusInfoNameGap     = 4
	homunculusInfoNameButtonW = 44
	homunculusInfoNameInputW  = homunculusInfoRightW - homunculusInfoInfoLabelW - homunculusInfoNameButtonW - homunculusInfoNameGap*2
	homunculusInfoBarPadX     = homunculusInfoInfoLabelW + homunculusInfoInfoGap
	homunculusInfoBarW        = homunculusInfoInfoValueW
	homunculusInfoBarH        = 7
	homunculusInfoBarGap      = 2
	homunculusInfoRowGap      = 4
)

var (
	homunculusInfoBarBack   = color.RGBA{R: 66, G: 66, B: 66, A: 255}
	homunculusInfoHPColor   = PlayerHPBarColor
	homunculusInfoSPColor   = PlayerSPBarColor
	homunculusInfoExpColor  = WindowBorderColor
	homunculusInfoFoodColor = WindowBorderColor
)

type HomunculusInfoActionKind uint8

const (
	HomunculusInfoActionNone HomunculusInfoActionKind = iota
	HomunculusInfoActionSkill
	HomunculusInfoActionFeed
	HomunculusInfoActionDelete
	HomunculusInfoActionRename
)

type HomunculusInfoAction struct {
	Kind HomunculusInfoActionKind
	Name string
}

type HomunculusInfoWindow struct {
	Window
	companion session.Companion
	name      string
	nameField *textfield.Widget
	snapshot  string
	action    HomunculusInfoAction
}

func (w *HomunculusInfoWindow) OpenInfo(ctx Context, companion session.Companion) {
	w.EnsureWindow(homunculusInfoWindowW, homunculusInfoWindowH)
	w.companion = companion
	w.name = companion.Name
	w.nameField = nil
	w.snapshot = homunculusInfoSnapshot(companion)
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *HomunculusInfoWindow) Update(ctx Context) bool {
	w.EnsureWindow(homunculusInfoWindowW, homunculusInfoWindowH)
	if !w.IsOpen() {
		return false
	}
	if ctx.Session == nil || !ctx.Session.Homunculus.Active {
		w.Close()
		w.Publish(ctx)
		return false
	}
	nextSnapshot := homunculusInfoSnapshot(ctx.Session.Homunculus)
	if nextSnapshot != w.snapshot {
		w.companion = ctx.Session.Homunculus
		w.snapshot = nextSnapshot
		if w.nameField == nil || !w.nameField.IsFocused() {
			w.name = ctx.Session.Homunculus.Name
		}
		w.SetContent(w.widgetTree(ctx))
	}
	if w.submitFromFocusedEnter(ctx) {
		w.Publish(ctx)
		return true
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *HomunculusInfoWindow) Rebind(ctx Context) {
	if !w.IsOpen() || ctx.Session == nil {
		return
	}
	w.companion = ctx.Session.Homunculus
	w.snapshot = homunculusInfoSnapshot(w.companion)
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *HomunculusInfoWindow) PopAction() HomunculusInfoAction {
	action := w.action
	w.action = HomunculusInfoAction{}
	return action
}

func (w *HomunculusInfoWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Homunculus Info"),
		CloseButton(true),
		OnClose(func() {
			w.Close()
			w.Publish(ctx)
		}),
		Size(homunculusInfoWindowW, homunculusInfoWindowH),
		Content(w.contentTree(ctx)),
		Footer(
			rotheme.Button("Skill", func() {
				w.action = HomunculusInfoAction{Kind: HomunculusInfoActionSkill}
			}).Width(72),
			rotheme.Button("Feed", func() {
				w.action = HomunculusInfoAction{Kind: HomunculusInfoActionFeed}
			}).Width(72),
			rotheme.Button("Delete", func() {
				w.action = HomunculusInfoAction{Kind: HomunculusInfoActionDelete}
			}).Width(72),
		),
	)
}

func (w *HomunculusInfoWindow) contentTree(ctx Context) widget.Widget {
	return primitives.HBox(
		w.statsTable(),
		w.detailsColumn(ctx),
	).
		Gap(homunculusInfoColumnGap).
		CrossAlign(primitives.CrossAxisStart).
		Padding(homunculusInfoContentPad)
}

func (w *HomunculusInfoWindow) statsTable() widget.Widget {
	rows := []rotheme.TableRow{
		homunculusInfoStatRow("ATK", fmt.Sprintf("%d", w.companion.Attack)),
		homunculusInfoStatRow("MATK", fmt.Sprintf("%d", w.companion.MagicAttack)),
		homunculusInfoStatRow("HIT", fmt.Sprintf("%d", w.companion.Hit)),
		homunculusInfoStatRow("CRI", fmt.Sprintf("%d", w.companion.Critical)),
		homunculusInfoStatRow("DEF", fmt.Sprintf("%d", w.companion.Defense)),
		homunculusInfoStatRow("MDEF", fmt.Sprintf("%d", w.companion.MagicDefense)),
		homunculusInfoStatRow("FLEE", fmt.Sprintf("%d", w.companion.Flee)),
		homunculusInfoStatRow("ASPD", fmt.Sprintf("%d", HomunculusASPDDisplay(w.companion.ASPD))),
	}
	return homunculusInfoTable(rows)
}

func (w *HomunculusInfoWindow) detailsColumn(ctx Context) widget.Widget {
	return primitives.Box(
		w.nameRow(ctx),
		w.infoRow("Level", fmt.Sprintf("%d", w.companion.Level)),
		w.barRow("HP", w.companion.HP, w.companion.MaxHP, Color(homunculusInfoHPColor), true),
		w.barRow("SP", w.companion.SP, w.companion.MaxSP, Color(homunculusInfoSPColor), true),
		w.expBarRow("EXP", w.companion.Exp, w.companion.MaxExp, Color(homunculusInfoExpColor)),
		w.barRow("Hunger", w.companion.Hunger, 100, Color(homunculusInfoFoodColor), true),
		w.infoRow("Intimacy", HomunculusIntimacyText(w.companion.Intimacy)),
		w.infoRow("Skill Pt", fmt.Sprintf("%d", w.companion.Skills.Points)),
	).
		Width(homunculusInfoRightW).
		Gap(homunculusInfoRowGap)
}

func (w *HomunculusInfoWindow) nameRow(ctx Context) widget.Widget {
	if !homunculusCanRename(w.companion) {
		return w.infoRow("Name", displayHomunculusName(w.companion))
	}
	return primitives.HBox(
		w.rowLabel("Name", homunculusInfoInfoLabelW),
		primitives.Box(w.nameInput(ctx)).
			Width(homunculusInfoNameInputW).
			Height(homunculusInfoNameH).
			CrossAlign(primitives.CrossAxisStretch),
		rotheme.Button("Edit", func() {
			w.rename()
		}).Width(homunculusInfoNameButtonW),
	).
		Gap(homunculusInfoNameGap).
		CrossAlign(primitives.CrossAxisCenter).
		Height(homunculusInfoNameH)
}

func (w *HomunculusInfoWindow) nameInput(ctx Context) *textfield.Widget {
	if w.nameField != nil {
		return w.nameField
	}
	w.nameField = rotheme.TextField(
		w.name,
		textfield.TypeText,
		func(value string) {
			w.name = value
		},
		func(string) {
			w.rename()
		},
		textfield.MaxLength(24),
	)
	return w.nameField
}

func (w *HomunculusInfoWindow) infoRow(label, value string) widget.Widget {
	return primitives.HBox(
		w.rowLabel(label, homunculusInfoInfoLabelW),
		primitives.Box(rotheme.Text(value)).
			Width(homunculusInfoInfoValueW),
	).
		Gap(homunculusInfoInfoGap).
		CrossAlign(primitives.CrossAxisCenter).
		Height(homunculusInfoRowH)
}

func (w *HomunculusInfoWindow) barRow(label string, current, maxValue int, fill widget.Color, showValues bool) widget.Widget {
	text := formatHomunculusBarText(current, maxValue, showValues)
	return primitives.Box(
		primitives.HBox(
			w.rowLabel(label, homunculusInfoInfoLabelW),
			primitives.Box(rotheme.Text(text).Align(widget.TextAlignRight)).
				Width(homunculusInfoInfoValueW),
		).
			Gap(homunculusInfoInfoGap).
			CrossAlign(primitives.CrossAxisCenter).
			Height(homunculusInfoRowH),
		primitives.Box(
			newHomunculusBarWidget(ratioInt(current, maxValue), fill, homunculusInfoBarW, homunculusInfoBarH),
		).
			PaddingLeft(homunculusInfoBarPadX),
	).
		Gap(homunculusInfoBarGap)
}

func (w *HomunculusInfoWindow) expBarRow(label string, current, maxValue uint64, fill widget.Color) widget.Widget {
	return primitives.Box(
		primitives.HBox(
			w.rowLabel(label, homunculusInfoInfoLabelW),
			primitives.Box(rotheme.Text(formatHomunculusEXPBarText(current, maxValue)).Align(widget.TextAlignRight)).
				Width(homunculusInfoInfoValueW),
		).
			Gap(homunculusInfoInfoGap).
			CrossAlign(primitives.CrossAxisCenter).
			Height(homunculusInfoRowH),
		primitives.Box(
			newHomunculusBarWidget(ratioUint64(current, maxValue), fill, homunculusInfoBarW, homunculusInfoBarH),
		).
			PaddingLeft(homunculusInfoBarPadX),
	).
		Gap(homunculusInfoBarGap)
}

func homunculusInfoTable(rows []rotheme.TableRow) widget.Widget {
	return rotheme.Table(
		rows,
		rotheme.TableRowHeightOpt(homunculusInfoRowH),
		rotheme.TableColors(rotheme.Default.Colors.ButtonHover, rotheme.Default.Colors.WindowFooter),
	)
}

func homunculusInfoStatRow(label, value string) rotheme.TableRow {
	return rotheme.TableRow{
		{Text: label, Width: homunculusInfoStatLabelW, Align: widget.TextAlignLeft, Head: true},
		{Text: value, Width: homunculusInfoStatValueW, Align: widget.TextAlignRight},
	}
}

func (w *HomunculusInfoWindow) rowLabel(label string, width float32) widget.Widget {
	return primitives.Box(
		rotheme.Text(label).
			Color(rotheme.Default.Colors.MutedText),
	).
		Width(width)
}

func (w *HomunculusInfoWindow) rename() {
	name := strings.TrimSpace(w.name)
	if name == "" || name == strings.TrimSpace(w.companion.Name) {
		return
	}
	w.action = HomunculusInfoAction{Kind: HomunculusInfoActionRename, Name: name}
}

func (w *HomunculusInfoWindow) submitFromFocusedEnter(ctx Context) bool {
	if ctx.Input == nil || w.nameField == nil || !w.nameField.IsFocused() {
		return false
	}
	if !ctx.Input.JustPressed(input.KeyEnter) {
		return false
	}
	w.rename()
	return true
}

type homunculusBarWidget struct {
	widget.WidgetBase
	ratio  float64
	fill   widget.Color
	width  float32
	height float32
}

func newHomunculusBarWidget(ratio float64, fill widget.Color, width, height float32) *homunculusBarWidget {
	w := &homunculusBarWidget{ratio: ratio, fill: fill, width: width, height: height}
	w.SetVisible(true)
	w.SetEnabled(false)
	return w
}

func (w *homunculusBarWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(w.width, w.height))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *homunculusBarWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	canvas.DrawRect(bounds, Color(homunculusInfoBarBack))
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

func (w *homunculusBarWidget) Event(ctx widget.Context, e event.Event) bool {
	return false
}

func homunculusInfoSnapshot(companion session.Companion) string {
	return fmt.Sprintf(
		"id=%d;active=%t;name=%s;flags=%d;level=%d;atk=%d;matk=%d;hit=%d;cri=%d;def=%d;mdef=%d;flee=%d;aspd=%d;hp=%d/%d;sp=%d/%d;exp=%d/%d;hunger=%d;intimacy=%d;skill=%d",
		companion.ID,
		companion.Active,
		companion.Name,
		companion.Flags,
		companion.Level,
		companion.Attack,
		companion.MagicAttack,
		companion.Hit,
		companion.Critical,
		companion.Defense,
		companion.MagicDefense,
		companion.Flee,
		companion.ASPD,
		companion.HP,
		companion.MaxHP,
		companion.SP,
		companion.MaxSP,
		companion.Exp,
		companion.MaxExp,
		companion.Hunger,
		companion.Intimacy,
		companion.Skills.Points,
	)
}

func displayHomunculusName(companion session.Companion) string {
	name := strings.TrimSpace(companion.Name)
	if name == "" {
		return "Homunculus"
	}
	return trimRunes(name, 20)
}

func homunculusCanRename(companion session.Companion) bool {
	return companion.Flags < 5
}

func formatHomunculusBarText(current, maxValue int, showValues bool) string {
	if maxValue <= 0 {
		if showValues {
			return fmt.Sprintf("%d / 0", current)
		}
		return "0%"
	}
	if showValues {
		return fmt.Sprintf("%d / %d", current, maxValue)
	}
	return formatEXPPercent(int64(current), int64(maxValue))
}

func formatHomunculusEXPBarText(current, maxValue uint64) string {
	if maxValue == 0 {
		return "--"
	}
	percent := 100 * float64(current) / float64(maxValue)
	if percent > 100 {
		percent = 100
	}
	return fmt.Sprintf("%.1f%%", math.Floor(percent*10)/10)
}

func ratioUint64(current, maxValue uint64) float64 {
	if maxValue == 0 {
		return 0
	}
	return clampUnit(float64(current) / float64(maxValue))
}

func HomunculusIntimacyText(value int) string {
	switch {
	case value < 100:
		return "Awkward"
	case value < 250:
		return "Shy"
	case value < 600:
		return "Neutral"
	case value < 900:
		return "Cordial"
	default:
		return "Loyal"
	}
}

func HomunculusASPDDisplay(value int) int {
	if value <= 0 {
		return 0
	}
	return maxInt(0, 200-value/10)
}
