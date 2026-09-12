package ui

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	characterWindowX                  = windowScreenMargin
	characterWindowY                  = windowScreenMargin
	characterWindowWidth              = 324
	characterWindowHeight             = 134
	characterEXPPanelPaddingX float32 = 6
	characterEXPPanelPaddingY float32 = 4
	characterEXPPanelGap      float32 = 2
	characterEXPPanelRadius   float32 = 5
	characterEXPLabelWidth    float32 = 76
	characterEXPLabelBarGap   float32 = 6
	characterEXPBarHeight     float32 = 6
	characterTextLineHeight   float32 = 1.2
)

var (
	characterWindowBarBack     = color.RGBA{R: 224, G: 232, B: 242, A: 255}
	characterWindowHPColor     = PlayerHPBarColor
	characterWindowSPColor     = PlayerSPBarColor
	characterWindowEXPColor    = WindowBorderColor
	characterWindowJobEXPColor = WindowBorderColor
)

type CharacterWindow struct {
	Window
	snapshot string
}

func (w *CharacterWindow) Update(ctx Context) bool {
	w.EnsureWindow(characterWindowWidth, characterWindowHeight)
	w.CloseOnEsc = false
	_, basicMenuHeight := basicMenuSize()
	w.dragBottom = basicMenuFollowGap + basicMenuHeight
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
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *CharacterWindow) widgetTree(ctx Context) widget.Widget {
	character, vitals, progress, inventory := characterWindowData(ctx.Session)
	name := strings.TrimSpace(character.Name)
	if name == "" {
		name = "Player"
	}
	jobName := strings.TrimSpace(db.JobDisplayName(int(character.Job)))
	title := trimRunes(name, 20)
	if jobName != "" {
		title = trimRunes(fmt.Sprintf("%s (%s)", name, jobName), 32)
	}

	weightColor := rotheme.Default.Colors.Text
	if inventory.MaxWeight > 0 && inventory.Weight*100 >= inventory.MaxWeight*50 {
		weightColor = Color(ErrorTextColor)
	}

	return Win(
		Title(title),
		CloseButton(false),
		Size(float32(characterWindowWidth), float32(characterWindowHeight)),
		Content(
			primitives.Box(
				primitives.HBox(
					characterRatioRow("HP", vitals.HP, vitals.MaxHP, Color(characterWindowHPColor), 146),
					characterRatioRow("SP", vitals.SP, vitals.MaxSP, Color(characterWindowSPColor), 146),
				).Gap(8),
				characterEXPPanel(progress, characterWindowWidth-24),
				primitives.HBox(
					characterAlignedTextCell(fmt.Sprintf("Weight : %d / %d", displayWeight(inventory.Weight), displayWeight(inventory.MaxWeight)), 146, weightColor, primitives.TextAlignStart),
					characterAlignedTextCell(fmt.Sprintf("Zeny : %s", formatHUDNumber(inventory.Zeny)), 146, rotheme.Default.Colors.Text, primitives.TextAlignEnd),
				),
			).
				PaddingXY(12, 9).
				Gap(8),
		),
	)
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

func characterRatioRow(label string, current, maxValue int, fill widget.Color, width float32) widget.Widget {
	return primitives.Box(
		rotheme.Text(fmt.Sprintf("%s %d / %d", label, current, maxValue)).
			Color(rotheme.Default.Colors.MutedText),
		newCharacterBarWidget(ratioInt(current, maxValue), fill, width, 7),
	).Width(width).Gap(2)
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

func newCharacterBarWidget(ratio float64, fill widget.Color, width, height float32) *characterBarWidget {
	return newCharacterBarWidgetWithBackground(ratio, fill, Color(characterWindowBarBack), width, height)
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
