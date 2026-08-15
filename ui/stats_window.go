package ui

import (
	"fmt"
	"strings"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	statsWindowHeight = 161
	statsWindowPad    = 10
	statsRowH         = 18
	statsRowGap       = rotheme.TableGap
	statsSectionGap   = 10

	statsPrimaryLabelWidth  float32 = 44
	statsPrimaryValueWidth  float32 = 56
	statsPrimaryCostWidth   float32 = 28
	statsPrimaryButtonGap   float32 = 7
	statsPrimaryColumnWidth         = statsPrimaryLabelWidth + statsPrimaryValueWidth + statsPrimaryCostWidth + rotheme.IconButtonSize + statsPrimaryButtonGap

	statsDerivedLabelWidth  float32 = 46
	statsDerivedValueWidth  float32 = 62
	statsDerivedPairWidth           = statsDerivedLabelWidth + rotheme.TableGap + statsDerivedValueWidth
	statsDerivedColumnWidth         = 2*(statsDerivedLabelWidth+statsDerivedValueWidth) + 3*rotheme.TableGap
	statsWindowWidth                = int(2*statsWindowPad + statsPrimaryColumnWidth + statsSectionGap + statsDerivedColumnWidth)
)

type StatsWindow struct {
	Window
	snapshot string
}

type statRow struct {
	label    string
	statusID uint16
	value    int
	bonus    int
	cost     int
}

func (w *StatsWindow) Toggle(ctx Context) {
	w.EnsureWindow(statsWindowWidth, statsWindowHeight)
	if w.IsOpen() {
		w.Close()
		w.Publish(ctx)
		return
	}
	x, y := statsWindowPosition(ctx)
	w.snapshot = statsWindowSnapshot(ctx.Session)
	w.OpenAt(x, y, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *StatsWindow) Update(ctx Context) bool {
	w.EnsureWindow(statsWindowWidth, statsWindowHeight)
	if !w.IsOpen() {
		return false
	}
	nextSnapshot := statsWindowSnapshot(ctx.Session)
	if nextSnapshot != w.snapshot {
		w.snapshot = nextSnapshot
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *StatsWindow) Rebind(ctx Context) {
	w.EnsureWindow(statsWindowWidth, statsWindowHeight)
	if !w.IsOpen() {
		return
	}
	w.refresh(ctx)
}

func (w *StatsWindow) refresh(ctx Context) {
	w.EnsureWindow(statsWindowWidth, statsWindowHeight)
	if !w.IsOpen() {
		return
	}
	w.snapshot = statsWindowSnapshot(ctx.Session)
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *StatsWindow) close(ctx Context) {
	w.EnsureWindow(statsWindowWidth, statsWindowHeight)
	w.Close()
	w.Publish(ctx)
}

func statsWindowPosition(ctx Context) (int, int) {
	width, height := ctx.ScreenSize()
	x := minInt(characterWindowX+characterWindowWidth+12, maxInt(windowScreenMargin, width-statsWindowWidth-windowScreenMargin))
	y := minInt(characterWindowY, maxInt(windowScreenMargin, height-statsWindowHeight-windowScreenMargin))
	if x < windowScreenMargin {
		x = windowScreenMargin
	}
	if y < windowScreenMargin {
		y = windowScreenMargin
	}
	return x, y
}

func (w *StatsWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Status"),
		CloseButton(true),
		OnClose(func() { w.close(ctx) }),
		Size(float32(statsWindowWidth), statsWindowHeight),
		Content(
			primitives.Box(w.statsBodyWidget(ctx)).Padding(statsWindowPad),
		),
	)
}

func (w *StatsWindow) statsBodyWidget(ctx Context) widget.Widget {
	stats := sessionStats(ctx.Session)
	primary := primitives.Box(
		w.statRowsWidget(ctx),
	).
		Width(statsPrimaryColumnWidth)

	return primitives.HBox(
		primary,
		statsRightColumnWidget(ctx.Session, stats),
	).
		Gap(statsSectionGap).
		CrossAlign(primitives.CrossAxisStart)
}

func (w *StatsWindow) statRowsWidget(ctx Context) widget.Widget {
	rows := statsRows(ctx.Session)
	children := make([]widget.Widget, 0, len(rows))
	for _, row := range rows {
		row := row
		children = append(children,
			primitives.HBox(
				statsLabelCell(row.label, statsPrimaryLabelWidth).
					Height(statsRowH).
					Background(rotheme.Default.Colors.ButtonHover),
				statsTextCell(formatStatValue(row.value, row.bonus), statsPrimaryValueWidth, rotheme.Default.Colors.Text),
				statsTextCell(fmt.Sprintf("%d", statCost(row)), statsPrimaryCostWidth, rotheme.Default.Colors.MutedText),
				primitives.Expanded(primitives.Box()),
				rotheme.IconButtonDisabled(rotheme.IconButtonPlus, !canIncreaseStat(ctx.Session, row), func() {
					w.requestStatIncrease(ctx, row)
				}),
			).
				Height(statsRowH).
				CrossAlign(primitives.CrossAxisCenter).
				Background(rotheme.Default.Colors.WindowFooter),
		)
	}
	return primitives.Box(children...).Gap(statsRowGap)
}

func (w *StatsWindow) requestStatIncrease(ctx Context, row statRow) {
	if !canIncreaseStat(ctx.Session, row) {
		return
	}
	if ctx.Network == nil {
		return
	}
	if err := ctx.Network.SendStatusIncrease(row.statusID); err != nil {
		glog.Warnf("status increase request status=%d failed: %v", row.statusID, err)
		return
	}
}

func statsTextCell(text string, width float32, color widget.Color) *primitives.BoxWidget {
	return statsWidgetCell(rotheme.Text(text).Color(color), width)
}

func statsLabelCell(text string, width float32) *primitives.BoxWidget {
	return statsWidgetCell(rotheme.Label(text), width)
}

func statsWidgetCell(content widget.Widget, width float32) *primitives.BoxWidget {
	return primitives.HBox(
		content,
	).
		Width(width).
		PaddingXY(rotheme.TableCellPadX, 0).
		CrossAlign(primitives.CrossAxisCenter)
}

func statsDerivedWidget(stats session.Stats) widget.Widget {
	rows := []rotheme.TableRow{
		{
			{Text: "ATK", Width: statsDerivedLabelWidth, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d + %d", stats.Attack, stats.AttackBonus), Width: statsDerivedValueWidth, Align: widget.TextAlignRight},
			{Text: "DEF", Width: statsDerivedLabelWidth, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d + %d", stats.Defense, stats.DefenseBonus), Width: statsDerivedValueWidth, Align: widget.TextAlignRight},
		},
		{
			{Text: "MATK", Width: statsDerivedLabelWidth, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d - %d", stats.MatkMin, stats.MatkMax), Width: statsDerivedValueWidth, Align: widget.TextAlignRight},
			{Text: "MDEF", Width: statsDerivedLabelWidth, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d + %d", stats.MDefense, stats.MDefenseBonus), Width: statsDerivedValueWidth, Align: widget.TextAlignRight},
		},
		{
			{Text: "HIT", Width: statsDerivedLabelWidth, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d", stats.Hit), Width: statsDerivedValueWidth, Align: widget.TextAlignRight},
			{Text: "FLEE", Width: statsDerivedLabelWidth, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d + %d", stats.Flee, stats.FleeBonus), Width: statsDerivedValueWidth, Align: widget.TextAlignRight},
		},
		{
			{Text: "CRIT", Width: statsDerivedLabelWidth, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d", stats.Critical), Width: statsDerivedValueWidth, Align: widget.TextAlignRight},
			{Text: "ASPD", Width: statsDerivedLabelWidth, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d", displayASPD(stats.ASPD+stats.ASPDBonus)), Width: statsDerivedValueWidth, Align: widget.TextAlignRight},
		},
	}
	return primitives.Box(
		rotheme.Table(
			rows,
			rotheme.TableRowHeightOpt(18),
			rotheme.TableColors(rotheme.Default.Colors.ButtonHover, rotheme.Default.Colors.WindowFooter),
		),
	)
}

func statsRightColumnWidget(s *session.Session, stats session.Stats) widget.Widget {
	return primitives.Box(
		statsDerivedWidget(stats),
		statsDetailsWidget(s, stats.Points),
	).Gap(rotheme.TableGap)
}

func statsDetailsWidget(s *session.Session, points int) widget.Widget {
	return rotheme.Table(
		[]rotheme.TableRow{
			{
				{Text: "Status Point", Width: statsDerivedPairWidth, Align: widget.TextAlignLeft, Head: true},
				{Text: fmt.Sprintf("%d", points), Width: statsDerivedPairWidth, Align: widget.TextAlignRight},
			},
			{
				{Text: "Guild", Width: statsDerivedPairWidth, Align: widget.TextAlignLeft, Head: true},
				{Text: statsGuildName(s), Width: statsDerivedPairWidth, Align: widget.TextAlignRight},
			},
		},
		rotheme.TableRowHeightOpt(statsRowH),
		rotheme.TableColors(rotheme.Default.Colors.ButtonHover, rotheme.Default.Colors.WindowFooter),
	)
}

func statsGuildName(s *session.Session) string {
	if s == nil {
		return ""
	}
	if name := strings.TrimSpace(s.Guild.Name); name != "" {
		return name
	}
	return strings.TrimSpace(s.GuildName)
}

func statsWindowSnapshot(s *session.Session) string {
	stats := sessionStats(s)
	rows := statsRows(s)
	return fmt.Sprintf(
		"%d|%d/%d/%d/%d/%d/%d|%d/%d/%d/%d/%d/%d|%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d|%s",
		stats.Points,
		rows[0].value, rows[1].value, rows[2].value, rows[3].value, rows[4].value, rows[5].value,
		rows[0].bonus, rows[1].bonus, rows[2].bonus, rows[3].bonus, rows[4].bonus, rows[5].bonus,
		stats.Attack, stats.AttackBonus, stats.MatkMin, stats.MatkMax,
		stats.Hit, stats.Critical, stats.Defense, stats.DefenseBonus,
		stats.MDefense, stats.MDefenseBonus, stats.Flee, stats.FleeBonus, stats.ASPD+stats.ASPDBonus,
		statsGuildName(s),
	)
}

func statsRows(s *session.Session) []statRow {
	stats := sessionStats(s)
	return []statRow{
		{label: "STR", statusID: network.StatusStr, value: stats.Str, bonus: stats.StrBonus, cost: stats.StrCost},
		{label: "AGI", statusID: network.StatusAgi, value: stats.Agi, bonus: stats.AgiBonus, cost: stats.AgiCost},
		{label: "VIT", statusID: network.StatusVit, value: stats.Vit, bonus: stats.VitBonus, cost: stats.VitCost},
		{label: "INT", statusID: network.StatusInt, value: stats.Int, bonus: stats.IntBonus, cost: stats.IntCost},
		{label: "DEX", statusID: network.StatusDex, value: stats.Dex, bonus: stats.DexBonus, cost: stats.DexCost},
		{label: "LUK", statusID: network.StatusLuk, value: stats.Luk, bonus: stats.LukBonus, cost: stats.LukCost},
	}
}

func sessionStats(s *session.Session) session.Stats {
	if s == nil {
		return session.Stats{}
	}
	stats := s.Stats
	character := selectedCharacter(s)
	if stats.Str == 0 {
		stats.Str = int(character.Str)
	}
	if stats.Agi == 0 {
		stats.Agi = int(character.Agi)
	}
	if stats.Vit == 0 {
		stats.Vit = int(character.Vit)
	}
	if stats.Int == 0 {
		stats.Int = int(character.Int)
	}
	if stats.Dex == 0 {
		stats.Dex = int(character.Dex)
	}
	if stats.Luk == 0 {
		stats.Luk = int(character.Luk)
	}
	return stats
}

func canIncreaseStat(s *session.Session, row statRow) bool {
	if s == nil || row.value <= 0 || row.value >= 99 {
		return false
	}
	return sessionStats(s).Points >= statCost(row)
}

func statCost(row statRow) int {
	if row.cost > 0 {
		return row.cost
	}
	return statPointCost(row.value)
}

func statPointCost(current int) int {
	if current < 1 {
		current = 1
	}
	return 1 + (current+9)/10
}

func formatStatValue(value, bonus int) string {
	if bonus == 0 {
		return fmt.Sprintf("%d", value)
	}
	if bonus > 0 {
		return fmt.Sprintf("%d + %d", value, bonus)
	}
	return fmt.Sprintf("%d - %d", value, -bonus)
}

func displayASPD(raw int) int {
	if raw <= 0 {
		return 0
	}
	return raw / 4
}

func (w *StatsWindow) ApplyStatusChangeAck(ctx Context, ack network.StatusChangeAck) {
	if ctx.Session == nil {
		return
	}
	if !ack.Success {
		glog.Debugf("status increase ack status=%d success=false value=%d", ack.StatusID, ack.Value)
		return
	}
	setSessionStat(ctx.Session, ack.StatusID, ack.Value)
	if ctx.Session.Stats.Points > 0 {
		ctx.Session.Stats.Points--
	}
	glog.Debugf("status increase ack status=%d success=true value=%d", ack.StatusID, ack.Value)
	w.refresh(ctx)
}

func setSessionStat(s *session.Session, statusID uint16, value int) {
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}
	switch statusID {
	case network.StatusStr:
		s.Stats.Str = value
		s.Selected.Str = uint8(value)
	case network.StatusAgi:
		s.Stats.Agi = value
		s.Selected.Agi = uint8(value)
	case network.StatusVit:
		s.Stats.Vit = value
		s.Selected.Vit = uint8(value)
	case network.StatusInt:
		s.Stats.Int = value
		s.Selected.Int = uint8(value)
	case network.StatusDex:
		s.Stats.Dex = value
		s.Selected.Dex = uint8(value)
	case network.StatusLuk:
		s.Stats.Luk = value
		s.Selected.Luk = uint8(value)
	}
}
