package ui

import (
	"fmt"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	statsWindowWidth  = 286
	statsWindowHeight = 302
	statsWindowPad    = 12
	statsRowH         = 22
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
	x := minInt(characterWindowX+characterWindowWidth+12, maxInt(8, width-statsWindowWidth-8))
	y := minInt(characterWindowY, maxInt(8, height-statsWindowHeight-8))
	if x < 8 {
		x = 8
	}
	if y < 8 {
		y = 8
	}
	return x, y
}

func (w *StatsWindow) widgetTree(ctx Context) widget.Widget {
	stats := sessionStats(ctx.Session)
	return Win(
		Title("Status"),
		CloseButton(true),
		OnClose(func() { w.close(ctx) }),
		Size(statsWindowWidth, statsWindowHeight),
		Content(
			primitives.Box(
				rotheme.Text(fmt.Sprintf("Status Point : %d", stats.Points)),
				primitives.HBox(
					statsTextCell("Stat", 48, rotheme.Default.Colors.MutedText),
					statsTextCell("Value", 58, rotheme.Default.Colors.MutedText),
					statsTextCell("Need", 42, rotheme.Default.Colors.MutedText),
				).
					Height(16).
					CrossAlign(primitives.CrossAxisCenter),
				w.statRowsWidget(ctx),
				statsDerivedWidget(stats),
			).
				Padding(statsWindowPad).
				Gap(4),
		),
	)
}

func (w *StatsWindow) statRowsWidget(ctx Context) widget.Widget {
	rows := statsRows(ctx.Session)
	children := make([]widget.Widget, 0, len(rows))
	for i, row := range rows {
		row := row
		bg := rotheme.Default.Colors.Button
		if i%2 == 1 {
			bg = rotheme.Default.Colors.PanelBody
		}
		children = append(children,
			primitives.HBox(
				statsTextCell(row.label, 48, rotheme.Default.Colors.Text),
				statsTextCell(formatStatValue(row.value, row.bonus), 58, rotheme.Default.Colors.Text),
				statsTextCell(fmt.Sprintf("%d", statCost(row)), 42, rotheme.Default.Colors.MutedText),
				primitives.Expanded(primitives.Box()),
				rotheme.IconButtonDisabled(rotheme.IconButtonPlus, !canIncreaseStat(ctx.Session, row), func() {
					w.requestStatIncrease(ctx, row)
				}),
			).
				Height(statsRowH-2).
				PaddingXY(4, 0).
				CrossAlign(primitives.CrossAxisCenter).
				Background(bg),
		)
	}
	return primitives.Box(children...).Gap(0)
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

func statsTextCell(text string, width float32, color widget.Color) widget.Widget {
	return primitives.Box(
		rotheme.Text(text).Color(color),
	).Width(width)
}

func statsDerivedWidget(stats session.Stats) widget.Widget {
	rows := []rotheme.TableRow{
		{
			{Text: "ATK", Width: 46, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d + %d", stats.Attack, stats.AttackBonus), Width: 62, Align: widget.TextAlignLeft},
			{Text: "DEF", Width: 46, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d + %d", stats.Defense, stats.DefenseBonus), Width: 62, Align: widget.TextAlignLeft},
		},
		{
			{Text: "MATK", Width: 46, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d - %d", stats.MatkMin, stats.MatkMax), Width: 62, Align: widget.TextAlignLeft},
			{Text: "MDEF", Width: 46, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d + %d", stats.MDefense, stats.MDefenseBonus), Width: 62, Align: widget.TextAlignLeft},
		},
		{
			{Text: "HIT", Width: 46, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d", stats.Hit), Width: 62, Align: widget.TextAlignLeft},
			{Text: "FLEE", Width: 46, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d + %d", stats.Flee, stats.FleeBonus), Width: 62, Align: widget.TextAlignLeft},
		},
		{
			{Text: "CRIT", Width: 46, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d", stats.Critical), Width: 62, Align: widget.TextAlignLeft},
			{Text: "ASPD", Width: 46, Align: widget.TextAlignLeft, Head: true},
			{Text: fmt.Sprintf("%d", displayASPD(stats.ASPD+stats.ASPDBonus)), Width: 62, Align: widget.TextAlignLeft},
		},
	}
	return primitives.Box(
		rotheme.Table(
			rows,
			rotheme.TableRowHeightOpt(18),
			rotheme.TableColors(rotheme.Default.Colors.ButtonHover, rotheme.Default.Colors.WindowFooter),
		),
	).
		PaddingTop(8)
}

func statsWindowSnapshot(s *session.Session) string {
	stats := sessionStats(s)
	rows := statsRows(s)
	return fmt.Sprintf(
		"%d|%d/%d/%d/%d/%d/%d|%d/%d/%d/%d/%d/%d|%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d",
		stats.Points,
		rows[0].value, rows[1].value, rows[2].value, rows[3].value, rows[4].value, rows[5].value,
		rows[0].bonus, rows[1].bonus, rows[2].bonus, rows[3].bonus, rows[4].bonus, rows[5].bonus,
		stats.Attack, stats.AttackBonus, stats.MatkMin, stats.MatkMax,
		stats.Hit, stats.Critical, stats.Defense, stats.DefenseBonus,
		stats.MDefense, stats.MDefenseBonus, stats.Flee, stats.FleeBonus, stats.ASPD+stats.ASPDBonus,
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
