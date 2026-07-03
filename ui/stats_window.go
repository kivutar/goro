package ui

import (
	"fmt"
	"log"
	"time"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

const (
	statsWindowWidth  = 286
	statsWindowHeight = 302
	statsWindowTitleH = 28
	statsWindowPad    = 12
	statsRowH         = 22
	statsButtonSize   = IconButtonSize
)

var (
	statsWindowTitleColor  = TitleTextColor
	statsWindowTextColor   = TextColor
	statsWindowMutedColor  = MutedTextColor
	statsWindowGoodColor   = GoodTextColor
	statsWindowButtonColor = ButtonColor
	statsWindowHoverColor  = ButtonHoverColor
	statsWindowDownColor   = ButtonDownColor
	statsWindowDisabled    = DisabledColor
)

type StatsWindow struct {
	open       bool
	x          int
	y          int
	positioned bool
	dragging   bool
	dragDX     int
	dragDY     int
	status     string
	statusGood bool
	statusAt   time.Time
}

type statRow struct {
	label    string
	statusID uint16
	value    int
	bonus    int
	cost     int
}

func (w *StatsWindow) Toggle(ctx Context) {
	if w.open {
		w.open = false
		w.dragging = false
		return
	}
	w.open = true
	w.EnsurePosition(ctx)
}

func (w *StatsWindow) Update(ctx Context) bool {
	if !w.open || ctx.Input == nil {
		return false
	}
	w.EnsurePosition(ctx)
	width, height := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(render.MouseButtonLeft) {
			w.x = clampStatsWindowInt(ctx.Input.MouseX-w.dragDX, 8, maxInt(8, width-statsWindowWidth-8))
			w.y = clampStatsWindowInt(ctx.Input.MouseY-w.dragDY, 8, maxInt(8, height-statsWindowHeight-8))
			return true
		}
		w.dragging = false
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.open = false
		return true
	}
	if !ctx.Input.MouseJustPressed(render.MouseButtonLeft) {
		return pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, statsWindowWidth, statsWindowHeight)
	}
	mx, my := ctx.Input.MouseX, ctx.Input.MouseY
	if !pointInRect(mx, my, w.x, w.y, statsWindowWidth, statsWindowHeight) {
		return false
	}
	cx, cy, cw, ch := w.closeBounds()
	if pointInRect(mx, my, cx, cy, cw, ch) {
		w.open = false
		return true
	}
	if pointInRect(mx, my, w.x, w.y, statsWindowWidth, statsWindowTitleH) {
		w.dragging = true
		w.dragDX = mx - w.x
		w.dragDY = my - w.y
		return true
	}
	for _, row := range statsRows(ctx.Session) {
		bx, by, bw, bh := w.statButtonBounds(row.statusID)
		if !pointInRect(mx, my, bx, by, bw, bh) {
			continue
		}
		if !canIncreaseStat(ctx.Session, row) {
			w.setStatus("Not enough status points", false)
			return true
		}
		if ctx.Network == nil {
			w.setStatus("Not connected", false)
			return true
		}
		if err := ctx.Network.SendStatusIncrease(row.statusID); err != nil {
			w.setStatus(err.Error(), false)
			return true
		}
		w.setStatus(fmt.Sprintf("%s increase requested", row.label), true)
		return true
	}
	return true
}

func (w *StatsWindow) Draw(screen *render.Image, ctx Context) {
	if !w.open || screen == nil {
		return
	}
	w.EnsurePosition(ctx)
	x, y := w.x, w.y
	DrawTitledWindowFrame(screen, x, y, statsWindowWidth, statsWindowHeight, statsWindowTitleH)
	DrawWindowTitle(screen, x, y, statsWindowTitleH, statsWindowPad, "Status", statsWindowTitleColor)
	cx, cy, cw, ch := w.closeBounds()
	DrawCloseButton(screen, cx, cy, cw, ch, statsWindowButtonColor, statsWindowTextColor)

	stats := sessionStats(ctx.Session)
	render.DebugPrintAtColor(screen, fmt.Sprintf("Status Point : %d", stats.Points), x+statsWindowPad, y+statsWindowTitleH+10, statsWindowTextColor)
	render.DebugPrintAtColor(screen, "Stat", x+statsWindowPad, y+statsWindowTitleH+32, statsWindowMutedColor)
	render.DebugPrintAtColor(screen, "Value", x+72, y+statsWindowTitleH+32, statsWindowMutedColor)
	render.DebugPrintAtColor(screen, "Need", x+132, y+statsWindowTitleH+32, statsWindowMutedColor)

	mx, my := -1, -1
	mouseDown := false
	if ctx.Input != nil {
		mx, my = ctx.Input.MouseX, ctx.Input.MouseY
		mouseDown = ctx.Input.MousePressed(render.MouseButtonLeft)
	}
	for i, row := range statsRows(ctx.Session) {
		ry := w.statRowY(i)
		rowColor := PanelAltColor
		if i%2 == 1 {
			rowColor = PanelBodyColor
		}
		DrawRowSurface(screen, x+statsWindowPad, ry, statsWindowWidth-2*statsWindowPad, statsRowH-2, rowColor)
		render.DebugPrintAtColor(screen, row.label, x+statsWindowPad, ry+4, statsWindowTextColor)
		render.DebugPrintAtColor(screen, formatStatValue(row.value, row.bonus), x+72, ry+4, statsWindowTextColor)
		render.DebugPrintAtColor(screen, fmt.Sprintf("%d", statCost(row)), x+132, ry+4, statsWindowMutedColor)
		bx, by, bw, bh := w.statButtonBounds(row.statusID)
		fill := statsWindowButtonColor
		textColor := statsWindowGoodColor
		if !canIncreaseStat(ctx.Session, row) {
			fill = statsWindowDisabled
			textColor = statsWindowMutedColor
		} else if pointInRect(mx, my, bx, by, bw, bh) {
			if mouseDown {
				fill = statsWindowDownColor
			} else {
				fill = statsWindowHoverColor
			}
		}
		DrawPlusButton(screen, bx, by, fill, textColor)
	}

	leftX := x + statsWindowPad
	rightX := x + 150
	derivedY := y + 220
	drawStatsDerived(screen, leftX, derivedY, "ATK", fmt.Sprintf("%d + %d", stats.Attack, stats.AttackBonus))
	drawStatsDerived(screen, leftX, derivedY+18, "MATK", fmt.Sprintf("%d - %d", stats.MatkMin, stats.MatkMax))
	drawStatsDerived(screen, leftX, derivedY+36, "HIT", fmt.Sprintf("%d", stats.Hit))
	drawStatsDerived(screen, leftX, derivedY+54, "CRIT", fmt.Sprintf("%d", stats.Critical))
	drawStatsDerived(screen, rightX, derivedY, "DEF", fmt.Sprintf("%d + %d", stats.Defense, stats.DefenseBonus))
	drawStatsDerived(screen, rightX, derivedY+18, "MDEF", fmt.Sprintf("%d + %d", stats.MDefense, stats.MDefenseBonus))
	drawStatsDerived(screen, rightX, derivedY+36, "FLEE", fmt.Sprintf("%d + %d", stats.Flee, stats.FleeBonus))
	drawStatsDerived(screen, rightX, derivedY+54, "ASPD", fmt.Sprintf("%d", displayASPD(stats.ASPD+stats.ASPDBonus)))
}

func (w *StatsWindow) CursorAction(ctx Context) (int, bool) {
	if !w.open || ctx.Input == nil {
		return 0, false
	}
	mx, my := ctx.Input.MouseX, ctx.Input.MouseY
	cx, cy, cw, ch := w.closeBounds()
	if pointInRect(mx, my, cx, cy, cw, ch) {
		return CursorActionClick, true
	}
	if pointInRect(mx, my, w.x, w.y, statsWindowWidth, statsWindowTitleH) {
		return CursorActionClick, true
	}
	for _, row := range statsRows(ctx.Session) {
		if !canIncreaseStat(ctx.Session, row) {
			continue
		}
		bx, by, bw, bh := w.statButtonBounds(row.statusID)
		if pointInRect(mx, my, bx, by, bw, bh) {
			return CursorActionClick, true
		}
	}
	if pointInRect(mx, my, w.x, w.y, statsWindowWidth, statsWindowHeight) {
		return CursorActionDefault, true
	}
	return 0, false
}

func (w *StatsWindow) EnsurePosition(ctx Context) {
	if w.positioned {
		return
	}
	width, height := ctx.ScreenSize()
	w.x = minInt(characterWindowX+characterWindowWidth+12, maxInt(8, width-statsWindowWidth-8))
	w.y = minInt(characterWindowY, maxInt(8, height-statsWindowHeight-8))
	if w.x < 8 {
		w.x = 8
	}
	if w.y < 8 {
		w.y = 8
	}
	w.positioned = true
}

func (w *StatsWindow) closeBounds() (int, int, int, int) {
	return w.x + statsWindowWidth - 25, w.y + 6, IconButtonSize, IconButtonSize
}

func (w *StatsWindow) statRowY(index int) int {
	return w.y + statsWindowTitleH + 50 + index*statsRowH
}

func (w *StatsWindow) statButtonBounds(statusID uint16) (int, int, int, int) {
	rows := statsRows(nil)
	for i, row := range rows {
		if row.statusID == statusID {
			return w.x + statsWindowWidth - statsWindowPad - statsButtonSize, w.statRowY(i) + 3, statsButtonSize, statsButtonSize
		}
	}
	return 0, 0, 0, 0
}

func (w *StatsWindow) setStatus(status string, good bool) {
	w.status = status
	w.statusGood = good
	w.statusAt = time.Now()
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

func drawStatsDerived(screen *render.Image, x, y int, label, value string) {
	render.DebugPrintAtColor(screen, label, x, y, statsWindowMutedColor)
	render.DebugPrintAtColor(screen, value, x+46, y, statsWindowTextColor)
}

func displayASPD(raw int) int {
	if raw <= 0 {
		return 0
	}
	return raw / 4
}

func clampStatsWindowInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (w *StatsWindow) ApplyStatusChangeAck(ctx Context, ack network.StatusChangeAck) {
	if ctx.Session == nil {
		return
	}
	label := statLabel(ack.StatusID)
	if !ack.Success {
		w.setStatus(fmt.Sprintf("%s increase failed", label), false)
		log.Printf("status increase ack status=%d success=false value=%d", ack.StatusID, ack.Value)
		return
	}
	setSessionStat(ctx.Session, ack.StatusID, ack.Value)
	if ctx.Session.Stats.Points > 0 {
		ctx.Session.Stats.Points--
	}
	w.setStatus(fmt.Sprintf("%s increased to %d", label, ack.Value), true)
	log.Printf("status increase ack status=%d success=true value=%d", ack.StatusID, ack.Value)
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

func statLabel(statusID uint16) string {
	switch statusID {
	case network.StatusStr:
		return "STR"
	case network.StatusAgi:
		return "AGI"
	case network.StatusVit:
		return "VIT"
	case network.StatusInt:
		return "INT"
	case network.StatusDex:
		return "DEX"
	case network.StatusLuk:
		return "LUK"
	default:
		return fmt.Sprintf("Stat %d", statusID)
	}
}
