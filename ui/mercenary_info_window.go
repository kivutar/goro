package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	mercenaryInfoWindowW = homunculusInfoWindowW
	mercenaryInfoWindowH = 300
)

var (
	mercenaryInfoTimeColor  = WindowBorderColor
	mercenaryInfoTimeLow    = color.RGBA{R: 0xFF, G: 0x1E, B: 0x00, A: 0xFF}
	mercenaryInfoKillsColor = WindowBorderColor
)

type MercenaryInfoActionKind uint8

const (
	MercenaryInfoActionNone MercenaryInfoActionKind = iota
	MercenaryInfoActionSkill
	MercenaryInfoActionDelete
)

type MercenaryInfoAction struct {
	Kind MercenaryInfoActionKind
}

type MercenaryInfoWindow struct {
	Window
	companion session.Companion
	snapshot  string
	action    MercenaryInfoAction
}

func (w *MercenaryInfoWindow) OpenInfo(ctx Context, companion session.Companion) {
	w.EnsureWindow(mercenaryInfoWindowW, mercenaryInfoWindowH)
	w.companion = companion
	w.snapshot = mercenaryInfoSnapshot(companion)
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *MercenaryInfoWindow) Update(ctx Context) bool {
	w.EnsureWindow(mercenaryInfoWindowW, mercenaryInfoWindowH)
	if !w.IsOpen() {
		return false
	}
	if ctx.Session == nil || !ctx.Session.Mercenary.Active {
		w.Close()
		w.Publish(ctx)
		return false
	}
	nextSnapshot := mercenaryInfoSnapshot(ctx.Session.Mercenary)
	if nextSnapshot != w.snapshot {
		w.companion = ctx.Session.Mercenary
		w.snapshot = nextSnapshot
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *MercenaryInfoWindow) Rebind(ctx Context) {
	if !w.IsOpen() || ctx.Session == nil {
		return
	}
	w.companion = ctx.Session.Mercenary
	w.snapshot = mercenaryInfoSnapshot(w.companion)
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *MercenaryInfoWindow) PopAction() MercenaryInfoAction {
	action := w.action
	w.action = MercenaryInfoAction{}
	return action
}

func (w *MercenaryInfoWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Mercenary Info"),
		CloseButton(true),
		OnClose(func() {
			w.Close()
			w.Publish(ctx)
		}),
		Size(mercenaryInfoWindowW, mercenaryInfoWindowH),
		Content(w.contentTree(ctx)),
		Footer(
			rotheme.Button("Skill", func() {
				w.action = MercenaryInfoAction{Kind: MercenaryInfoActionSkill}
			}).Width(72),
			rotheme.Button("Dismiss", func() {
				w.action = MercenaryInfoAction{Kind: MercenaryInfoActionDelete}
			}).Width(72),
		),
	)
}

func (w *MercenaryInfoWindow) contentTree(ctx Context) widget.Widget {
	return primitives.HBox(
		w.statsTable(),
		w.detailsColumn(ctx),
	).
		Gap(homunculusInfoColumnGap).
		CrossAlign(primitives.CrossAxisStart).
		Padding(homunculusInfoContentPad)
}

func (w *MercenaryInfoWindow) statsTable() widget.Widget {
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

func (w *MercenaryInfoWindow) detailsColumn(ctx Context) widget.Widget {
	timeFill := Color(mercenaryInfoTimeColor)
	if mercenaryRemainingRatio(w.companion.ExpireTick) < 0.25 {
		timeFill = Color(mercenaryInfoTimeLow)
	}
	return primitives.Box(
		w.infoRow("Name", displayMercenaryName(w.companion)),
		w.infoRow("Level", fmt.Sprintf("%d", w.companion.Level)),
		w.barRow("HP", w.companion.HP, w.companion.MaxHP, Color(homunculusInfoHPColor), true),
		w.barRow("SP", w.companion.SP, w.companion.MaxSP, Color(homunculusInfoSPColor), true),
		w.barRow("Time", mercenaryRemainingSeconds(w.companion.ExpireTick), mercenaryLifetimeSeconds, timeFill, false),
		w.infoRow("Left", formatMercenaryExpireDate(w.companion.ExpireTick)),
		w.barRow("Kills", int(w.companion.Kills%50), 50, Color(mercenaryInfoKillsColor), true),
		w.infoRow("Faith", fmt.Sprintf("%d", w.companion.Faith)),
	).
		Width(homunculusInfoRightW).
		Gap(homunculusInfoRowGap)
}

func (w *MercenaryInfoWindow) infoRow(label, value string) widget.Widget {
	h := HomunculusInfoWindow{}
	return h.infoRow(label, value)
}

func (w *MercenaryInfoWindow) barRow(label string, current, maxValue int, fill widget.Color, showValues bool) widget.Widget {
	h := HomunculusInfoWindow{}
	return h.barRow(label, current, maxValue, fill, showValues)
}

func mercenaryInfoSnapshot(companion session.Companion) string {
	return fmt.Sprintf(
		"id=%d;active=%t;name=%s;level=%d;atk=%d;matk=%d;hit=%d;cri=%d;def=%d;mdef=%d;flee=%d;aspd=%d;hp=%d/%d;sp=%d/%d;expire=%d;timeBucket=%d;kills=%d;faith=%d;skill=%d",
		companion.ID,
		companion.Active,
		companion.Name,
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
		companion.ExpireTick,
		time.Now().Unix()/30,
		companion.Kills,
		companion.Faith,
		companion.Skills.Points,
	)
}

func displayMercenaryName(companion session.Companion) string {
	name := strings.TrimSpace(companion.Name)
	if name == "" {
		return "Mercenary"
	}
	return trimRunes(name, 20)
}

const mercenaryLifetimeSeconds = 30 * 60

func formatMercenaryExpireDate(timestamp uint32) string {
	remaining := mercenaryRemainingSeconds(timestamp)
	if remaining <= 0 {
		return "0h 0m"
	}
	hours := remaining / 3600
	minutes := (remaining % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func mercenaryRemainingSeconds(timestamp uint32) int {
	if timestamp == 0 {
		return 0
	}
	remaining := int64(timestamp) - time.Now().Unix()
	if remaining <= 0 {
		return 0
	}
	return int(remaining)
}

func mercenaryRemainingRatio(timestamp uint32) float64 {
	return ratioInt(mercenaryRemainingSeconds(timestamp), mercenaryLifetimeSeconds)
}
