package ui

import (
	"fmt"
	"image"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	guildWindowWidth     = 400
	guildWindowHeight    = 325
	guildWindowTabHeight = 23
	guildWindowTabWidth  = 64
	guildEmblemSize      = 24
	guildTablePadding    = 7
	guildTableViewportW  = guildWindowWidth
	guildTableRowH       = 32
	guildTableHeaderH    = 24
	guildTableControlH   = 22
)

type GuildWindow struct {
	Window
	tab             guildWindowTab
	snapshot        string
	action          GuildWindowAction
	EmblemImage     func(Context) image.Image
	emblemOptions   []GuildEmblemOption
	memberScrollY   state.Signal[float32]
	memberDraft     map[guildMemberKey]uint32
	memberSource    string
	positionDraft   map[uint32]guildPositionDraft
	positionSource  string
	positionScrollY state.Signal[float32]
	skillIcons      map[uint16]image.Image
	skillMiss       map[uint16]struct{}
	skillPending    map[uint16]int
	skillOrder      []uint16
	skillScrollY    state.Signal[float32]
	historyScrollY  state.Signal[float32]
	tooltip         tooltipState
}

type GuildWindowAction struct {
	RequestMenu        bool
	MenuTab            uint32
	SelectedEmblemPath string
	MemberPositions    []GuildMemberPositionUpdate
	LevelUpSkillIDs    []uint16
	UpdatePositions    bool
	Positions          []GuildPositionUpdate
}

type GuildMemberPositionUpdate struct {
	AccountID  uint32
	CharID     uint32
	PositionID uint32
}

type GuildPositionUpdate struct {
	PositionID uint32
	Right      uint32
	Ranking    uint32
	PayRate    uint32
	PosName    string
}

type GuildEmblemOption struct {
	Label string
	Path  string
}

type guildPositionDraft struct {
	posName string
	right   uint32
	payRate string
}

type guildMemberKey struct {
	accountID uint32
	charID    uint32
}

type guildWindowTab int

const (
	guildWindowTabInfo guildWindowTab = iota
	guildWindowTabMembers
	guildWindowTabPositions
	guildWindowTabSkills
	guildWindowTabHistory
	guildWindowTabNotice
)

type guildWindowTabDef struct {
	tab   guildWindowTab
	label string
}

var guildWindowTabs = []guildWindowTabDef{
	{tab: guildWindowTabInfo, label: "Info"},
	{tab: guildWindowTabMembers, label: "Members"},
	{tab: guildWindowTabPositions, label: "Position"},
	{tab: guildWindowTabSkills, label: "Skill"},
	{tab: guildWindowTabHistory, label: "History"},
	{tab: guildWindowTabNotice, label: "Notice"},
}

func (w *GuildWindow) Toggle(ctx Context) {
	if w.IsOpen() {
		w.Close()
		return
	}
	w.OpenWindow(ctx)
}

func (w *GuildWindow) OpenWindow(ctx Context) {
	w.EnsureWindow(guildWindowWidth, guildWindowHeight)
	w.ctx = ctx
	if w.tab == 0 {
		w.tab = guildWindowTabInfo
	}
	w.snapshot = guildWindowSnapshot(ctx.Session)
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *GuildWindow) Update(ctx Context) bool {
	w.EnsureWindow(guildWindowWidth, guildWindowHeight)
	w.ctx = ctx
	if !w.IsOpen() {
		w.hideTooltip()
		return false
	}
	w.updateSkillTooltipHover(ctx)
	nextSnapshot := guildWindowSnapshot(ctx.Session)
	if nextSnapshot != w.snapshot {
		w.snapshot = nextSnapshot
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	hasAction := w.action.hasAction()
	w.Publish(ctx)
	return consumed || hasAction
}

func (w *GuildWindow) PopAction() GuildWindowAction {
	action := w.action
	w.action = GuildWindowAction{}
	return action
}

func (a GuildWindowAction) hasAction() bool {
	return a.RequestMenu ||
		a.SelectedEmblemPath != "" ||
		len(a.MemberPositions) > 0 ||
		len(a.LevelUpSkillIDs) > 0 ||
		a.UpdatePositions
}

func (w *GuildWindow) SetEmblemOptions(ctx Context, options []GuildEmblemOption) {
	w.emblemOptions = append(w.emblemOptions[:0], options...)
	if w.IsOpen() {
		w.refresh(ctx)
	}
}

func (w *GuildWindow) Rebind(ctx Context) {
	w.EnsureWindow(guildWindowWidth, guildWindowHeight)
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.snapshot = guildWindowSnapshot(ctx.Session)
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *GuildWindow) widgetTree(ctx Context) widget.Widget {
	options := []WindowOption{
		Title("Guild"),
		CloseButton(true),
		OnClose(w.Close),
		Size(guildWindowWidth, guildWindowHeight),
		Content(
			primitives.Box(
				w.tabStrip(),
				primitives.Expanded(w.tabContent(ctx)),
			).
				CrossAlign(primitives.CrossAxisStretch),
		),
	}
	if footer := w.tabFooter(ctx); footer != nil {
		options = append(options, Footer(footer...))
	}
	return Win(options...)
}

func (w *GuildWindow) tabStrip() widget.Widget {
	tabs := make([]widget.Widget, 0, len(guildWindowTabs)+1)
	for _, def := range guildWindowTabs {
		def := def
		tabs = append(tabs,
			newTabWidget(tabWidgetConfig{
				label:      def.label,
				active:     w.tab == def.tab,
				width:      guildWindowTabWidth,
				height:     guildWindowTabHeight,
				blendEdge:  tabBlendBottom,
				blendInset: 1,
				onClick: func() {
					w.tab = def.tab
					w.hideTooltip()
					w.action = GuildWindowAction{RequestMenu: true, MenuTab: uint32(def.tab)}
					w.refresh(w.ctx)
				},
			}),
		)
	}
	tabs = append(tabs, primitives.Expanded(primitives.Box()))
	return primitives.HBox(tabs...).
		Gap(-1).
		CrossAlign(primitives.CrossAxisStretch).
		Background(rotheme.Default.Colors.FooterLine)
}

func (w *GuildWindow) tabContent(ctx Context) widget.Widget {
	switch w.tab {
	case guildWindowTabInfo:
		return w.infoTab(ctx)
	case guildWindowTabMembers:
		return w.membersTab(ctx)
	case guildWindowTabPositions:
		return w.positionsTab(ctx)
	case guildWindowTabSkills:
		return w.skillsTab(ctx)
	case guildWindowTabHistory:
		return w.historyTab(ctx)
	case guildWindowTabNotice:
		return w.noticeTab(ctx)
	default:
		return guildWindowPlaceholder("")
	}
}

func (w *GuildWindow) tabFooter(ctx Context) []widget.Widget {
	guild := guildSessionInfo(ctx.Session)
	switch w.tab {
	case guildWindowTabMembers:
		if !guild.IsMaster {
			return nil
		}
		return []widget.Widget{
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Reset", func() {
				w.resetMemberDraft(ctx)
				w.refresh(ctx)
			}),
			rotheme.Button("Confirm", func() {
				w.confirmMemberPositionChanges(ctx)
			}),
		}
	case guildWindowTabPositions:
		if !guild.IsMaster {
			return nil
		}
		return []widget.Widget{
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Reset", func() {
				w.resetPositionDraft(ctx)
				w.refresh(ctx)
			}),
			rotheme.Button("Confirm", func() {
				changes := w.positionChanges(ctx)
				if len(changes) == 0 {
					return
				}
				w.action = GuildWindowAction{
					UpdatePositions: true,
					Positions:       changes,
				}
			}),
		}
	case guildWindowTabSkills:
		return []widget.Widget{
			footerLabel(fmt.Sprintf("Skill Points: %d", maxInt(0, guild.SkillPoints-w.skillPendingCount()))),
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Reset", func() {
				w.clearGuildSkillPending()
				w.refresh(ctx)
			}),
			rotheme.Button("Confirm", func() {
				w.confirmGuildSkillPending(ctx)
			}),
		}
	default:
		return nil
	}
}

func (w *GuildWindow) membersTab(ctx Context) widget.Widget {
	guild := guildSessionInfo(ctx.Session)
	members := guild.Members
	w.ensureMemberDraft(ctx, members)
	totalExp := guildMembersTotalExp(members)
	return primitives.Box(
		rotheme.TableView(
			rotheme.TableViewColumns(guildMemberTableColumns),
			rotheme.TableViewRowCount(len(members)),
			rotheme.TableViewRowHeight(guildTableRowH),
			rotheme.TableViewHeaderHeight(guildTableHeaderH),
			rotheme.TableViewEmptyText("No guild members loaded."),
			rotheme.TableViewScrollYSignal(w.ensureGuildMemberScrollSignal()),
			rotheme.TableViewInvalidateHover(false),
			rotheme.TableViewBuildCell(func(cell rotheme.TableViewCellContext) widget.Widget {
				if cell.Row < 0 || cell.Row >= len(members) {
					return primitives.Box()
				}
				return w.guildMemberTableCell(ctx, members[cell.Row], guild.Positions, guild.IsMaster, totalExp, cell)
			}),
		),
	).
		Background(rotheme.Default.Colors.WindowBody).
		CrossAlign(primitives.CrossAxisStretch)
}

var guildMemberTableColumns = []rotheme.TableViewColumn{
	{Key: "name", Title: "Name", Width: 78},
	{Key: "position", Title: "Position", Width: 62},
	{Key: "job", Title: "Job", Width: 52},
	{Key: "level", Title: "Lv", Width: 26},
	{Key: "note", Title: "Note", Width: 62},
	{Key: "devotion", Title: "Dev.", Width: 38},
	{Key: "tax", Title: "Tax", Width: 42},
	{Key: "fill", Flex: 1},
}

func (w *GuildWindow) guildMemberTableCell(ctx Context, member session.GuildMember, positions []session.GuildPosition, isMaster bool, totalExp uint32, cell rotheme.TableViewCellContext) widget.Widget {
	switch cell.Column.Key {
	case "name":
		return guildTableViewTextCell(guildText(member.CharName), cell.Width, guildTableRowH)
	case "position":
		return w.guildMemberPositionCell(ctx, member, positions, isMaster, cell.Width)
	case "job":
		return guildTableViewTextCell(db.JobDisplayName(int(member.Job)), cell.Width, guildTableRowH)
	case "level":
		return guildTableViewTextCell(fmt.Sprintf("%d", member.Level), cell.Width, guildTableRowH)
	case "note":
		return guildTableViewTextCell(member.Memo, cell.Width, guildTableRowH)
	case "devotion":
		return guildTableViewTextCell(guildMemberDevotion(member.MemberExp, totalExp), cell.Width, guildTableRowH)
	case "tax":
		return guildTableViewTextCell(fmt.Sprintf("%d", member.MemberExp), cell.Width, guildTableRowH)
	default:
		return primitives.Box()
	}
}

func (w *GuildWindow) guildMemberPositionCell(ctx Context, member session.GuildMember, positions []session.GuildPosition, isMaster bool, width float32) widget.Widget {
	if !isMaster || member.PositionID == 0 {
		return guildTableViewTextCell(guildPositionName(positions, member.PositionID), width, guildTableRowH)
	}
	positionID := member.PositionID
	if w.memberDraft != nil {
		if draftPositionID, ok := w.memberDraft[guildMemberKey{accountID: member.AccountID, charID: member.CharID}]; ok {
			positionID = draftPositionID
		}
	}
	sortedPositions := guildSortedPositions(positions)
	items := make([]dropdown.ItemDef, 0, len(sortedPositions))
	selected := -1
	for i, position := range sortedPositions {
		if position.PositionID == positionID {
			selected = i
		}
		items = append(items, dropdown.ItemDef{
			Value: strconv.FormatUint(uint64(position.PositionID), 10),
			Label: trimRunes(guildPositionTitle(position), 12),
		})
	}
	if len(items) == 0 {
		return guildTableViewTextCell(guildPositionName(positions, member.PositionID), width, guildTableRowH)
	}
	return guildTableViewControlCell(
		dropdown.New(
			dropdown.ItemDefs(items),
			dropdown.Selected(selected),
			dropdown.MaxVisibleItems(5),
			dropdown.PainterOpt(rotheme.DropdownPainter{}),
			dropdown.OnChange(func(_ int, value string) {
				positionID, err := strconv.ParseUint(value, 10, 32)
				if err != nil {
					return
				}
				w.stageMemberPosition(ctx, member, uint32(positionID))
				w.refresh(ctx)
			}),
		),
		width,
	)
}

func guildMembersTotalExp(members []session.GuildMember) uint32 {
	var total uint32
	for _, member := range members {
		total += member.MemberExp
	}
	return total
}

func guildMemberDevotion(memberExp, totalExp uint32) string {
	if memberExp == 0 || totalExp == 0 {
		return "0 %"
	}
	return fmt.Sprintf("%d %%", int(math.Round(float64(memberExp)/float64(totalExp)*100)))
}

func (w *GuildWindow) ensureGuildMemberScrollSignal() state.Signal[float32] {
	if w.memberScrollY == nil {
		w.memberScrollY = state.NewSignal[float32](0)
	}
	return w.memberScrollY
}

func (w *GuildWindow) ensureMemberDraft(ctx Context, members []session.GuildMember) {
	source := guildMemberDraftSource(members)
	if w.memberDraft != nil && w.memberSource == source {
		return
	}
	w.resetMemberDraft(ctx)
}

func (w *GuildWindow) resetMemberDraft(ctx Context) {
	guild := guildSessionInfo(ctx.Session)
	w.memberSource = guildMemberDraftSource(guild.Members)
	w.memberDraft = make(map[guildMemberKey]uint32, len(guild.Members))
	for _, member := range guild.Members {
		w.memberDraft[guildMemberKey{accountID: member.AccountID, charID: member.CharID}] = member.PositionID
	}
}

func (w *GuildWindow) stageMemberPosition(ctx Context, member session.GuildMember, positionID uint32) {
	guild := guildSessionInfo(ctx.Session)
	w.ensureMemberDraft(ctx, guild.Members)
	key := guildMemberKey{accountID: member.AccountID, charID: member.CharID}
	if positionID == 0 {
		for _, other := range guild.Members {
			otherKey := guildMemberKey{accountID: other.AccountID, charID: other.CharID}
			if otherKey != key && other.PositionID != 0 && w.memberDraft[otherKey] == 0 {
				w.memberDraft[otherKey] = other.PositionID
			}
		}
	}
	w.memberDraft[key] = positionID
}

func (w *GuildWindow) memberPositionChanges(ctx Context) []GuildMemberPositionUpdate {
	guild := guildSessionInfo(ctx.Session)
	changes := make([]GuildMemberPositionUpdate, 0)
	for _, member := range guild.Members {
		key := guildMemberKey{accountID: member.AccountID, charID: member.CharID}
		positionID, ok := w.memberDraft[key]
		if !ok || positionID == member.PositionID {
			continue
		}
		update := GuildMemberPositionUpdate{
			AccountID:  member.AccountID,
			CharID:     member.CharID,
			PositionID: positionID,
		}
		if positionID == 0 {
			// Position 0 transfers guild master; rAthena stops processing the packet after it.
			return []GuildMemberPositionUpdate{update}
		}
		changes = append(changes, update)
	}
	return changes
}

func (w *GuildWindow) confirmMemberPositionChanges(ctx Context) {
	changes := w.memberPositionChanges(ctx)
	if len(changes) == 0 {
		return
	}
	w.action = GuildWindowAction{MemberPositions: changes}
}

func guildMemberDraftSource(members []session.GuildMember) string {
	var out strings.Builder
	for _, member := range members {
		fmt.Fprintf(&out, "|%d:%d:%d", member.AccountID, member.CharID, member.PositionID)
	}
	return out.String()
}

func (w *GuildWindow) positionsTab(ctx Context) widget.Widget {
	guild := guildSessionInfo(ctx.Session)
	w.ensurePositionDraft(ctx, guild.Positions)
	positions := guildSortedPositions(guild.Positions)
	return primitives.Box(
		rotheme.TableView(
			rotheme.TableViewColumns(guildPositionTableColumns),
			rotheme.TableViewRowCount(len(positions)),
			rotheme.TableViewRowHeight(guildTableRowH),
			rotheme.TableViewHeaderHeight(guildTableHeaderH),
			rotheme.TableViewEmptyText("No guild positions loaded."),
			rotheme.TableViewScrollYSignal(w.ensureGuildPositionScrollSignal()),
			rotheme.TableViewInvalidateHover(false),
			rotheme.TableViewBuildCell(func(cell rotheme.TableViewCellContext) widget.Widget {
				if cell.Row < 0 || cell.Row >= len(positions) {
					return primitives.Box()
				}
				return w.guildPositionCell(ctx, positions[cell.Row], guild.IsMaster, cell)
			}),
		),
	).
		Background(rotheme.Default.Colors.WindowBody).
		CrossAlign(primitives.CrossAxisStretch)
}

var guildPositionTableColumns = []rotheme.TableViewColumn{
	{Key: "rank", Title: "Rank", Width: 44},
	{Key: "title", Title: "Position Title", Width: 164},
	{Key: "invite", Title: "Invitation", Width: 58},
	{Key: "punish", Title: "Punish", Width: 48},
	{Key: "tax", Title: "Tax", Width: 46},
	{Key: "fill", Flex: 1},
}

func (w *GuildWindow) guildPositionCell(ctx Context, position session.GuildPosition, isMaster bool, cell rotheme.TableViewCellContext) widget.Widget {
	switch cell.Column.Key {
	case "rank":
		return guildTableViewTextCell(fmt.Sprintf("%d", position.PositionID), cell.Width, guildTableRowH)
	case "title":
		return w.guildPositionTitleCell(position, isMaster, cell.Width)
	case "invite":
		return w.guildPositionRightCell(ctx, position.PositionID, 0x01, isMaster, cell.Width)
	case "punish":
		return w.guildPositionRightCell(ctx, position.PositionID, 0x10, isMaster, cell.Width)
	case "tax":
		return w.guildPositionTaxCell(position, isMaster, cell.Width)
	default:
		return primitives.Box()
	}
}

func (w *GuildWindow) guildPositionTitleCell(position session.GuildPosition, isMaster bool, width float32) widget.Widget {
	if !isMaster {
		return guildTableViewTextCell(guildPositionTitle(position), width, guildTableRowH)
	}
	return guildTableViewControlCell(
		rotheme.TextField(
			w.positionDraft[position.PositionID].posName,
			textfield.TypeText,
			func(value string) {
				draft := w.positionDraft[position.PositionID]
				draft.posName = value
				w.positionDraft[position.PositionID] = draft
			},
			nil,
			textfield.MaxLength(24),
		),
		width,
	)
}

func (w *GuildWindow) guildPositionRightCell(ctx Context, positionID, flag uint32, isMaster bool, width float32) widget.Widget {
	enabled := w.positionDraft[positionID].right&flag != 0
	if !isMaster {
		return guildTableViewTextCell(guildRightLabel(enabled), width, guildTableRowH)
	}
	return primitives.HBox(
		rotheme.Checkbox(
			checkbox.Checked(enabled),
			checkbox.OnToggle(func(checked bool) {
				draft := w.positionDraft[positionID]
				if checked {
					draft.right |= flag
				} else {
					draft.right &^= flag
				}
				w.positionDraft[positionID] = draft
				w.refresh(ctx)
			}),
		),
	).
		Width(width).
		Height(guildTableRowH).
		CrossAlign(primitives.CrossAxisCenter)
}

func (w *GuildWindow) guildPositionTaxCell(position session.GuildPosition, isMaster bool, width float32) widget.Widget {
	if !isMaster {
		return guildTableViewTextCell(fmt.Sprintf("%d %%", position.PayRate), width, guildTableRowH)
	}
	return guildTableViewControlCell(
		rotheme.TextField(
			w.positionDraft[position.PositionID].payRate,
			textfield.TypeNumber,
			func(value string) {
				draft := w.positionDraft[position.PositionID]
				draft.payRate = value
				w.positionDraft[position.PositionID] = draft
			},
			nil,
			textfield.MaxLength(3),
		),
		width,
	)
}

func guildTableViewTextCell(text string, width, height float32) widget.Widget {
	return primitives.HBox(
		rotheme.Text(trimRunes(strings.TrimSpace(text), int(width/7))).
			Align(widget.TextAlignLeft),
	).
		Width(width).
		Height(height).
		PaddingLeft(6).
		CrossAlign(primitives.CrossAxisCenter)
}

func guildTableViewControlCell(child widget.Widget, width float32) widget.Widget {
	return primitives.Box(child).
		Width(width).
		Height(guildTableControlH).
		CrossAlign(primitives.CrossAxisStretch)
}

func (w *GuildWindow) ensurePositionDraft(ctx Context, positions []session.GuildPosition) {
	source := guildPositionDraftSource(positions)
	if w.positionDraft != nil && w.positionSource == source {
		return
	}
	w.resetPositionDraft(ctx)
}

func (w *GuildWindow) resetPositionDraft(ctx Context) {
	guild := guildSessionInfo(ctx.Session)
	w.positionSource = guildPositionDraftSource(guild.Positions)
	w.positionDraft = make(map[uint32]guildPositionDraft, len(guild.Positions))
	for _, position := range guild.Positions {
		w.positionDraft[position.PositionID] = guildPositionDraft{
			posName: guildPositionTitle(position),
			right:   position.Right,
			payRate: strconv.FormatUint(uint64(position.PayRate), 10),
		}
	}
}

func (w *GuildWindow) positionChanges(ctx Context) []GuildPositionUpdate {
	guild := guildSessionInfo(ctx.Session)
	changes := make([]GuildPositionUpdate, 0)
	for _, position := range guildSortedPositions(guild.Positions) {
		draft, ok := w.positionDraft[position.PositionID]
		if !ok {
			continue
		}
		payRate, err := strconv.ParseUint(strings.TrimSpace(draft.payRate), 10, 32)
		if err != nil {
			payRate = uint64(position.PayRate)
		}
		posName := trimRunes(strings.TrimSpace(draft.posName), 24)
		if posName == "" {
			posName = guildPositionTitle(position)
		}
		update := GuildPositionUpdate{
			PositionID: position.PositionID,
			Right:      draft.right,
			Ranking:    position.Ranking,
			PayRate:    uint32(payRate),
			PosName:    posName,
		}
		if update.Right != position.Right || update.PayRate != position.PayRate || update.PosName != guildPositionTitle(position) {
			changes = append(changes, update)
		}
	}
	return changes
}

func guildPositionDraftSource(positions []session.GuildPosition) string {
	var out strings.Builder
	for _, position := range guildSortedPositions(positions) {
		fmt.Fprintf(&out, "|%d:%d:%d:%d:%s",
			position.PositionID,
			position.Right,
			position.Ranking,
			position.PayRate,
			guildPositionTitle(position),
		)
	}
	return out.String()
}

func (w *GuildWindow) ensureGuildPositionScrollSignal() state.Signal[float32] {
	if w.positionScrollY == nil {
		w.positionScrollY = state.NewSignal[float32](0)
	}
	return w.positionScrollY
}

func guildSortedPositions(positions []session.GuildPosition) []session.GuildPosition {
	out := append([]session.GuildPosition(nil), positions...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].PositionID < out[j].PositionID
	})
	return out
}

func guildPositionName(positions []session.GuildPosition, id uint32) string {
	for _, position := range positions {
		if position.PositionID == id {
			return guildPositionTitle(position)
		}
	}
	return fmt.Sprintf("Position %d", id)
}

func guildPositionTitle(position session.GuildPosition) string {
	if title := strings.TrimSpace(position.PosName); title != "" {
		return title
	}
	return fmt.Sprintf("Position %d", position.PositionID)
}

func guildRightLabel(enabled bool) string {
	if enabled {
		return "Yes"
	}
	return "-"
}

func (w *GuildWindow) skillsTab(ctx Context) widget.Widget {
	guild := guildSessionInfo(ctx.Session)
	return primitives.Box(
		rotheme.TableView(
			rotheme.TableViewColumns(guildSkillTableColumns),
			rotheme.TableViewRowCount(len(guild.Skills)),
			rotheme.TableViewRowHeight(guildTableRowH),
			rotheme.TableViewHeaderHeight(guildTableHeaderH),
			rotheme.TableViewEmptyText("No guild skills loaded."),
			rotheme.TableViewScrollYSignal(w.ensureGuildSkillScrollSignal()),
			rotheme.TableViewInvalidateHover(false),
			rotheme.TableViewDispatchHoverToCells(false),
			rotheme.TableViewBuildSimpleCell(func(cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
				if cell.Row < 0 || cell.Row >= len(guild.Skills) {
					return rotheme.TableViewSimpleCell{Hidden: true}
				}
				return w.guildSkillCell(ctx, w.guildSkillWithPending(guild.Skills[cell.Row]), guild, cell)
			}),
			rotheme.TableViewOnRowEventWithContext(func(widgetCtx widget.Context, row int, e event.Event) bool {
				return w.handleGuildSkillTableRowEvent(widgetCtx, ctx, guild, guild.Skills, row, e)
			}),
		),
	).
		Background(rotheme.Default.Colors.WindowBody).
		CrossAlign(primitives.CrossAxisStretch)
}

var guildSkillTableColumns = []rotheme.TableViewColumn{
	{Key: "icon", Width: 34},
	{Key: "type", Width: 16},
	{Key: "name", Title: "Name", Width: 234},
	{Key: "level", Title: "Lv", Width: 40},
	{Key: "levelup", Width: 22},
	{Key: "fill", Flex: 1},
}

func (w *GuildWindow) guildSkillCell(ctx Context, skill session.Skill, guild session.Guild, cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
	nameColor := rotheme.Default.Colors.Text
	if skill.Level <= 0 {
		nameColor = rotheme.Default.Colors.MutedText
	}
	switch cell.Column.Key {
	case "icon":
		return rotheme.TableViewSimpleCell{Icon: w.guildSkillIconImage(ctx, skill)}
	case "type":
		return rotheme.TableViewSimpleCell{
			Text:  skillTypeLabel(skill),
			Color: skillTypeColor(skill),
		}
	case "name":
		return rotheme.TableViewSimpleCell{
			Text:  trimRunes(skillDisplayName(ctx.Resources, skill), 28),
			Color: nameColor,
		}
	case "level":
		return rotheme.TableViewSimpleCell{
			Text:  guildSkillLevelText(skill),
			Color: nameColor,
		}
	case "levelup":
		return rotheme.TableViewIconButtonCell(rotheme.IconButtonPlus, !w.canStageGuildSkill(skill, guild))
	default:
		return rotheme.TableViewSimpleCell{Hidden: true}
	}
}

func (w *GuildWindow) canStageGuildSkill(skill session.Skill, guild session.Guild) bool {
	return guild.IsMaster && guild.SkillPoints > w.skillPendingCount() && skill.Upgradable && guildSkillCanLevelUp(skill)
}

func (w *GuildWindow) guildSkillIconImage(ctx Context, skill session.Skill) image.Image {
	if ctx.Resources == nil || skill.ID == 0 {
		return nil
	}
	if w.skillIcons == nil {
		w.skillIcons = make(map[uint16]image.Image)
	}
	if img := w.skillIcons[skill.ID]; img != nil {
		return img
	}
	if w.skillMiss != nil {
		if _, ok := w.skillMiss[skill.ID]; ok {
			return nil
		}
	}
	resourceName, ok := ctx.Resources.SkillResourceName(int(skill.ID))
	if !ok {
		resourceName = strings.ToUpper(strings.ReplaceAll(skillLabel(skill), " ", "_"))
	}
	img, _, err := res.LoadImage(ctx.Resources, res.SkillIconTextureCandidates(resourceName, int(skill.ID)))
	if err != nil {
		if w.skillMiss == nil {
			w.skillMiss = make(map[uint16]struct{})
		}
		w.skillMiss[skill.ID] = struct{}{}
		return nil
	}
	w.skillIcons[skill.ID] = img
	return img
}

func guildSkillLevelText(skill session.Skill) string {
	return fmt.Sprintf("%d", skill.Level)
}

func guildSkillCanLevelUp(skill session.Skill) bool {
	return skill.ID != 0 && (skill.MaxLevel <= 0 || skill.Level < skill.MaxLevel)
}

func (w *GuildWindow) handleGuildSkillTableRowEvent(widgetCtx widget.Context, ctx Context, guild session.Guild, skills []session.Skill, row int, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok || row < 0 || row >= len(skills) {
		return false
	}
	skill := w.guildSkillWithPending(skills[row])
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		if widgetCtx != nil && w.canStageGuildSkill(skill, guild) && guildSkillLevelUpButtonBounds(row).Contains(mouse.Position) {
			widgetCtx.SetCursor(widget.CursorPointer)
		}
		mx, my := int(mouse.GlobalPosition.X), int(mouse.GlobalPosition.Y)
		if ctx.Input != nil {
			mx, my = ctx.Input.MouseX, ctx.Input.MouseY
		}
		w.showSkillTooltip(ctx, skill, mx, my)
		return false
	case event.MousePress:
		if mouse.Button != event.ButtonLeft || !guildSkillLevelUpButtonBounds(row).Contains(mouse.Position) {
			return false
		}
		if !w.canStageGuildSkill(skill, guild) {
			return true
		}
		w.stageGuildSkill(skill.ID)
		w.refresh(ctx)
		return true
	}
	return false
}

func guildSkillLevelUpButtonBounds(row int) geometry.Rect {
	x := float32(0)
	for _, col := range guildSkillTableColumns {
		if col.Key == "levelup" {
			return geometry.NewRect(
				x+(col.Width-rotheme.IconButtonSize)/2,
				float32(row)*guildTableRowH+(guildTableRowH-rotheme.IconButtonSize)/2,
				rotheme.IconButtonSize,
				rotheme.IconButtonSize,
			)
		}
		x += col.Width
	}
	return geometry.Rect{}
}

func (w *GuildWindow) clearGuildSkillPending() {
	w.skillPending = nil
	w.skillOrder = nil
}

func (w *GuildWindow) skillPendingCount() int {
	total := 0
	for _, count := range w.skillPending {
		total += count
	}
	return total
}

func (w *GuildWindow) guildSkillWithPending(skill session.Skill) session.Skill {
	if w.skillPending != nil {
		skill.Level += w.skillPending[skill.ID]
	}
	return skill
}

func (w *GuildWindow) stageGuildSkill(skillID uint16) {
	if skillID == 0 {
		return
	}
	if w.skillPending == nil {
		w.skillPending = make(map[uint16]int)
	}
	if w.skillPending[skillID] == 0 {
		w.skillOrder = append(w.skillOrder, skillID)
	}
	w.skillPending[skillID]++
}

func (w *GuildWindow) confirmGuildSkillPending(ctx Context) {
	if len(w.skillOrder) == 0 {
		return
	}
	skillIDs := make([]uint16, 0, w.skillPendingCount())
	for _, skillID := range w.skillOrder {
		for i := 0; i < w.skillPending[skillID]; i++ {
			skillIDs = append(skillIDs, skillID)
		}
	}
	w.action = GuildWindowAction{LevelUpSkillIDs: skillIDs}
	w.clearGuildSkillPending()
	w.refresh(ctx)
}

func (w *GuildWindow) showSkillTooltip(ctx Context, skill session.Skill, mx, my int) {
	if skill.ID == 0 {
		w.hideTooltip()
		return
	}
	const tooltipW = 292
	w.tooltip.ShowBox(ctx, skillTooltipText(ctx, skill), mx+16+tooltipW/2, my+18, my-6, tooltipW, 24)
}

func (w *GuildWindow) hideTooltip() {
	w.tooltip.Hide()
}

func (w *GuildWindow) DrawTooltip(screen *render.Frame) {
	w.tooltip.Draw(screen)
}

func (w *GuildWindow) updateSkillTooltipHover(ctx Context) {
	if !w.tooltip.Open() {
		return
	}
	if w.tab != guildWindowTabSkills || ctx.Input == nil {
		w.hideTooltip()
		return
	}
	x, y, width, height := w.guildSkillTableBounds()
	if !pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, x, y, width, height) {
		w.hideTooltip()
	}
}

func (w *GuildWindow) guildSkillTableBounds() (int, int, int, int) {
	x := w.x
	y := w.y + ROWindowTitleHeight + guildWindowTabHeight
	height := w.height - ROWindowTitleHeight - guildWindowTabHeight - ROWindowFooterHeight
	if height < 0 {
		height = 0
	}
	return x, y, guildTableViewportW, height
}

func (w *GuildWindow) ensureGuildSkillScrollSignal() state.Signal[float32] {
	if w.skillScrollY == nil {
		w.skillScrollY = state.NewSignal[float32](0)
	}
	return w.skillScrollY
}

func (w *GuildWindow) historyTab(ctx Context) widget.Widget {
	history := guildSessionInfo(ctx.Session).ExpelHistory
	return primitives.Box(
		rotheme.TableView(
			rotheme.TableViewColumns(guildHistoryTableColumns),
			rotheme.TableViewRowCount(len(history)),
			rotheme.TableViewRowHeight(guildTableRowH),
			rotheme.TableViewHeaderHeight(guildTableHeaderH),
			rotheme.TableViewEmptyText("No expel history loaded."),
			rotheme.TableViewScrollYSignal(w.ensureGuildHistoryScrollSignal()),
			rotheme.TableViewInvalidateHover(false),
			rotheme.TableViewBuildSimpleCell(func(cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
				if cell.Row < 0 || cell.Row >= len(history) {
					return rotheme.TableViewSimpleCell{Hidden: true}
				}
				return guildHistoryCell(history[cell.Row], cell)
			}),
		),
	).
		Background(rotheme.Default.Colors.WindowBody).
		CrossAlign(primitives.CrossAxisStretch)
}

var guildHistoryTableColumns = []rotheme.TableViewColumn{
	{Key: "name", Title: "Name", Width: 116},
	{Key: "reason", Title: "The Reason of Expulsion", Flex: 1},
}

func guildHistoryCell(entry session.GuildExpelHistory, cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
	switch cell.Column.Key {
	case "name":
		return rotheme.TableViewSimpleCell{
			Text: trimRunes(guildText(entry.CharName), int(cell.Width/7)),
		}
	case "reason":
		return rotheme.TableViewSimpleCell{
			Text: trimRunes(guildText(entry.Reason), int(cell.Width/7)),
		}
	default:
		return rotheme.TableViewSimpleCell{Hidden: true}
	}
}

func (w *GuildWindow) ensureGuildHistoryScrollSignal() state.Signal[float32] {
	if w.historyScrollY == nil {
		w.historyScrollY = state.NewSignal[float32](0)
	}
	return w.historyScrollY
}

func (w *GuildWindow) noticeTab(ctx Context) widget.Widget {
	guild := guildSessionInfo(ctx.Session)
	return primitives.Box(
		rotheme.Text("Title"),
		guildNoticeBox(guildText(guild.NoticeSubject), 28, 1),
		rotheme.Text("Contents"),
		guildNoticeBox(guildText(guild.Notice), 140, 8),
	).
		PaddingXY(9, 10).
		Gap(5).
		CrossAlign(primitives.CrossAxisStretch).
		Background(rotheme.Default.Colors.WindowBody)
}

func guildNoticeBox(text string, height float32, maxLines int) widget.Widget {
	return primitives.Box(
		rotheme.Text(text).
			Align(widget.TextAlignLeft).
			MaxLines(maxLines).
			LineHeight(16/rotheme.Default.Typography.TextSize),
	).
		Height(height).
		PaddingXY(5, 4).
		CrossAlign(primitives.CrossAxisStretch).
		Background(rotheme.Default.Colors.WindowFooter).
		BorderStyle(1, rotheme.Default.Colors.FooterLine)
}

func (w *GuildWindow) infoTab(ctx Context) widget.Widget {
	guild := guildSessionInfo(ctx.Session)
	leftRows := []widget.Widget{
		guildInfoRow("Guild Name", guild.Name),
		guildInfoRow("Guild lvl", guildNumber(guild.Level)),
		guildInfoRow("Guild Master", guildText(guild.MasterName)),
		guildInfoRow("Guildsmen", guildMembers(guild.UserNum, guild.MaxUserNum)),
		guildInfoRow("Avg.lvl of Guildsmen", guildNumber(guild.UserAverageLevel)),
		guildInfoRow("Territory", guildText(guild.ManageLand)),
		guildInfoSection("Tendency"),
		guildTendencyBox(guild.Honor, guild.Virtue),
	}
	rightRows := []widget.Widget{
		guildInfoRow("EXP", guildExp(guild.Exp, guild.MaxExp)),
		guildInfoSection("Emblem"),
		w.guildEmblemEditor(ctx, guild.EmblemVersion),
		guildInfoRow("Tax Point", guildNumberAllowZero(guild.Point)),
		guildInfoSection("Alliance"),
		guildListBox(""),
		guildInfoSection("Antagonist"),
		guildListBox(""),
	}
	return primitives.HBox(
		primitives.Box(leftRows...).
			Gap(3).
			Width(188),
		primitives.Box(rightRows...).
			Gap(3).
			Width(174),
	).
		PaddingXY(9, 10).
		Gap(10).
		CrossAlign(primitives.CrossAxisStart)
}

func (w *GuildWindow) guildEmblemEditor(ctx Context, version uint32) widget.Widget {
	return primitives.HBox(
		w.guildEmblemBox(ctx, version),
		primitives.Box(w.guildEmblemDropdown()).
			Width(132).
			Height(22).
			CrossAlign(primitives.CrossAxisStretch),
	).
		Gap(6).
		CrossAlign(primitives.CrossAxisCenter)
}

func (w *GuildWindow) guildEmblemDropdown() widget.Widget {
	if len(w.emblemOptions) == 0 {
		return dropdown.New(
			dropdown.ItemDefs([]dropdown.ItemDef{{Value: "", Label: "No emblems found", Disabled: true}}),
			dropdown.Selected(0),
			dropdown.Disabled(true),
			dropdown.MaxVisibleItems(5),
			dropdown.PainterOpt(rotheme.DropdownPainter{}),
		)
	}
	items := make([]dropdown.ItemDef, 0, len(w.emblemOptions))
	for _, option := range w.emblemOptions {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = "Emblem"
		}
		items = append(items, dropdown.ItemDef{
			Value: option.Path,
			Label: trimRunes(label, 20),
		})
	}
	return dropdown.New(
		dropdown.ItemDefs(items),
		dropdown.Placeholder("Edit"),
		dropdown.MaxVisibleItems(5),
		dropdown.PainterOpt(rotheme.DropdownPainter{}),
		dropdown.OnChange(func(_ int, value string) {
			w.action.SelectedEmblemPath = value
		}),
	)
}

func guildInfoRow(label, value string) widget.Widget {
	return primitives.Box(
		rotheme.Text(label + " : " + value),
	).
		Height(16)
}

func guildInfoSection(label string) widget.Widget {
	return primitives.Box(rotheme.Text(label + ":")).
		Height(16)
}

func guildTendencyBox(honor, virtue uint32) widget.Widget {
	value := ""
	if honor != 0 || virtue != 0 {
		value = fmt.Sprintf("H %d  V %d", honor, virtue)
	}
	return primitives.Box(
		primitives.Box(rotheme.Text("R")).Width(16),
		primitives.HBox(
			primitives.Box(
				primitives.Expanded(primitives.Box()),
				rotheme.Text("V"),
				primitives.Expanded(primitives.Box()),
			).
				PaddingLeft(3).
				Width(16).
				Height(90).
				CrossAlign(primitives.CrossAxisCenter),
			guildFramedBox(90, 90, value),
			primitives.Box(
				primitives.Expanded(primitives.Box()),
				rotheme.Text("F"),
				primitives.Expanded(primitives.Box()),
			).
				Width(16).
				Height(90).
				CrossAlign(primitives.CrossAxisCenter),
		).
			Gap(4).
			CrossAlign(primitives.CrossAxisCenter),
		primitives.Box(rotheme.Text("W")).
			PaddingTop(3).
			Width(16).
			Height(19),
	).
		CrossAlign(primitives.CrossAxisCenter).
		Gap(2)
}

func (w *GuildWindow) guildEmblemBox(ctx Context, version uint32) widget.Widget {
	var content widget.Widget
	if w.EmblemImage != nil {
		if img := w.EmblemImage(ctx); img != nil {
			content = newStaticImageWidget(img, guildEmblemSize, guildEmblemSize)
		}
	}
	if content != nil {
		return guildFramedWidget(guildEmblemSize, guildEmblemSize, 0, content)
	}
	text := "-"
	if version != 0 {
		text = fmt.Sprintf("v%d", version)
	}
	return guildFramedBox(guildEmblemSize, guildEmblemSize, text)
}

func guildListBox(text string) widget.Widget {
	return guildFramedBox(168, 42, text)
}

func guildFramedBox(width, height float32, text string) widget.Widget {
	content := widget.Widget(rotheme.Text(text))
	if strings.TrimSpace(text) == "" {
		content = primitives.Box()
	}
	return guildFramedWidget(width, height, 4, content)
}

func guildFramedWidget(width, height, padding float32, content widget.Widget) widget.Widget {
	return primitives.Box(content).
		Width(width).
		Height(height).
		Padding(padding).
		Background(rotheme.Default.Colors.WindowFooter).
		BorderStyle(1, rotheme.Default.Colors.FooterLine)
}

func guildWindowPlaceholder(text string) widget.Widget {
	if strings.TrimSpace(text) == "" {
		text = "Not loaded yet."
	}
	return primitives.Box(
		rotheme.Text(text),
	).
		Padding(10).
		Background(rotheme.Default.Colors.WindowBody)
}

func (w *GuildWindow) refresh(ctx Context) {
	w.snapshot = guildWindowSnapshot(ctx.Session)
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *GuildWindow) Refresh(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.refresh(ctx)
}

func guildWindowSnapshot(s *session.Session) string {
	if s == nil {
		return ""
	}
	memberSnapshot := strings.Builder{}
	for _, member := range s.Guild.Members {
		fmt.Fprintf(&memberSnapshot, "|%d:%d:%d:%d:%d:%d:%d:%s:%s",
			member.AccountID,
			member.CharID,
			member.Job,
			member.Level,
			member.MemberExp,
			member.CurrentState,
			member.PositionID,
			member.CharName,
			member.Memo,
		)
	}
	positionSnapshot := strings.Builder{}
	for _, position := range s.Guild.Positions {
		fmt.Fprintf(&positionSnapshot, "|%d:%d:%d:%d:%s",
			position.PositionID,
			position.Right,
			position.Ranking,
			position.PayRate,
			position.PosName,
		)
	}
	skillSnapshot := strings.Builder{}
	for _, skill := range s.Guild.Skills {
		fmt.Fprintf(&skillSnapshot, "|%d:%d:%d:%d:%d:%s",
			skill.ID,
			skill.Type,
			skill.Level,
			skill.SPCost,
			skill.Range,
			skill.Name,
		)
	}
	historySnapshot := strings.Builder{}
	for _, entry := range s.Guild.ExpelHistory {
		fmt.Fprintf(&historySnapshot, "|%s:%s:%s",
			entry.CharName,
			entry.Account,
			entry.Reason,
		)
	}
	return fmt.Sprintf("%d|%d|%s|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%s|%s|%s|%d|%d|%s|%s%s%s%s%s",
		s.GuildID,
		s.EmblemVersion,
		s.GuildName,
		boolSnapshot(s.Guild.IsMaster),
		s.Guild.Level,
		s.Guild.UserNum,
		s.Guild.MaxUserNum,
		s.Guild.UserAverageLevel,
		s.Guild.Exp,
		s.Guild.MaxExp,
		s.Guild.Point,
		s.Guild.Honor,
		s.Guild.Virtue,
		s.Guild.MasterName,
		s.Guild.ManageLand,
		s.Guild.Name,
		s.Guild.Zeny,
		s.Guild.SkillPoints,
		s.Guild.NoticeSubject,
		s.Guild.Notice,
		memberSnapshot.String(),
		positionSnapshot.String(),
		skillSnapshot.String(),
		historySnapshot.String(),
	)
}

func boolSnapshot(value bool) int {
	if value {
		return 1
	}
	return 0
}

func guildSessionInfo(s *session.Session) session.Guild {
	if s == nil {
		return session.Guild{Name: "-"}
	}
	guild := s.Guild
	if guild.ID == 0 {
		guild.ID = s.GuildID
	}
	if guild.EmblemVersion == 0 {
		guild.EmblemVersion = s.EmblemVersion
	}
	if strings.TrimSpace(guild.Name) == "" {
		guild.Name = strings.TrimSpace(s.GuildName)
	}
	if strings.TrimSpace(guild.Name) == "" {
		guild.Name = "-"
	}
	return guild
}

func guildText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "-"
	}
	return text
}

func guildNumber(value uint32) string {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", value)
}

func guildNumberAllowZero(value uint32) string {
	return fmt.Sprintf("%d", value)
}

func guildMembers(current, max uint32) string {
	if current == 0 && max == 0 {
		return "- / -"
	}
	return fmt.Sprintf("%d / %d", current, max)
}

func guildExp(current, max uint32) string {
	if current == 0 && max == 0 {
		return "-"
	}
	if max == 0 {
		return fmt.Sprintf("%d", current)
	}
	return fmt.Sprintf("%d / %d", current, max)
}
