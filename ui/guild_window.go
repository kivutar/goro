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
	"github.com/gogpu/ui/core/scrollview"
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
	guildTableViewportW  = guildWindowWidth - guildTablePadding*2
	guildTableWidth      = guildWindowWidth - guildTablePadding*2 - ROScrollbarGutter
	guildSkillRowHeight  = 32
	guildSkillIconSize   = 24
)

type GuildWindow struct {
	Window
	tab            guildWindowTab
	snapshot       string
	action         GuildWindowAction
	EmblemImage    func(Context) image.Image
	emblemOptions  []GuildEmblemOption
	positionDraft  map[uint32]guildPositionDraft
	positionSource string
	skillIcons     map[uint16]image.Image
	skillMiss      map[uint16]struct{}
	skillPending   map[uint16]int
	skillOrder     []uint16
	skillScrollY   state.Signal[float32]
	tooltip        tooltipState
}

type GuildWindowAction struct {
	RequestMenu          bool
	MenuTab              uint32
	SelectedEmblemPath   string
	ChangeMemberPosition bool
	MemberAccountID      uint32
	MemberCharID         uint32
	MemberPositionID     uint32
	LevelUpSkillIDs      []uint16
	UpdatePositions      bool
	Positions            []GuildPositionUpdate
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
		a.ChangeMemberPosition ||
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
	rows := make([]widget.Widget, 0, len(members)+1)
	rows = append(rows, guildMemberHeaderRow())
	totalExp := guildMembersTotalExp(members)
	if len(members) == 0 {
		rows = append(rows,
			primitives.Box(rotheme.Text("No guild members loaded.")).
				Height(24).
				Width(guildTableWidth).
				PaddingXY(4, 4).
				Background(rotheme.Default.Colors.WindowBody),
		)
	}
	for i, member := range members {
		rows = append(rows, w.guildMemberRow(member, guild.Positions, guild.IsMaster, totalExp, i%2 == 0))
	}
	return primitives.Box(
		scrollview.New(
			primitives.Box(rows...),
			scrollview.DirectionOpt(scrollview.Vertical),
			scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollStep(20),
		),
	).
		PaddingXY(guildTablePadding, guildTablePadding).
		Background(rotheme.Default.Colors.WindowBody)
}

func guildMemberHeaderRow() widget.Widget {
	return primitives.HBox(
		guildMemberCell("Name", 78, true, false),
		guildMemberCell("Position", 62, true, false),
		guildMemberCell("Job", 52, true, false),
		guildMemberCell("Lv", 26, true, false),
		guildMemberCell("Note", 62, true, false),
		guildMemberCell("Dev.", 38, true, false),
		guildMemberCell("Tax", 42, true, false),
		guildTableFiller(false),
	).
		Width(guildTableWidth).
		Height(16)
}

func (w *GuildWindow) guildMemberRow(member session.GuildMember, positions []session.GuildPosition, isMaster bool, totalExp uint32, dark bool) widget.Widget {
	return primitives.HBox(
		guildMemberCell(guildText(member.CharName), 78, false, dark),
		w.guildMemberPositionCell(member, positions, isMaster, dark),
		guildMemberCell(db.JobDisplayName(int(member.Job)), 52, false, dark),
		guildMemberCell(fmt.Sprintf("%d", member.Level), 26, false, dark),
		guildMemberCell(member.Memo, 62, false, dark),
		guildMemberCell(guildMemberDevotion(member.MemberExp, totalExp), 38, false, dark),
		guildMemberCell(fmt.Sprintf("%d", member.MemberExp), 42, false, dark),
		guildTableFiller(dark),
	).
		Width(guildTableWidth).
		Height(20)
}

func (w *GuildWindow) guildMemberPositionCell(member session.GuildMember, positions []session.GuildPosition, isMaster, dark bool) widget.Widget {
	if !isMaster || member.PositionID == 0 {
		return guildMemberCell(guildPositionName(positions, member.PositionID), 62, false, dark)
	}
	sortedPositions := guildSortedPositions(positions)
	items := make([]dropdown.ItemDef, 0, len(sortedPositions))
	selected := -1
	for i, position := range sortedPositions {
		if position.PositionID == member.PositionID {
			selected = i
		}
		items = append(items, dropdown.ItemDef{
			Value: strconv.FormatUint(uint64(position.PositionID), 10),
			Label: trimRunes(guildPositionTitle(position), 12),
		})
	}
	if len(items) == 0 {
		return guildMemberCell(guildPositionName(positions, member.PositionID), 62, false, dark)
	}
	return primitives.Box(
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
				w.action = GuildWindowAction{
					ChangeMemberPosition: true,
					MemberAccountID:      member.AccountID,
					MemberCharID:         member.CharID,
					MemberPositionID:     uint32(positionID),
				}
			}),
		),
	).
		Width(62).
		Height(20).
		Background(guildRowBackground(dark))
}

func guildMemberCell(text string, width float32, header, dark bool) widget.Widget {
	text = trimRunes(strings.TrimSpace(text), int(width/7))
	color := rotheme.Default.Colors.Text
	height := float32(20)
	bg := guildTableRowBackground(dark)
	if header {
		color = rotheme.Default.Colors.MutedText
		height = 16
		bg = rotheme.Default.Colors.WindowBody
	}
	return primitives.HBox(
		rotheme.Text(text).
			Color(color).
			Align(widget.TextAlignLeft),
	).
		Width(width).
		Height(height).
		PaddingLeft(4).
		CrossAlign(primitives.CrossAxisCenter).
		Background(bg)
}

func guildTableFiller(dark bool) widget.Widget {
	return primitives.Expanded(
		primitives.Box().
			Background(guildTableRowBackground(dark)),
	)
}

func guildTableRowBackground(dark bool) widget.Color {
	if dark {
		return rotheme.Default.Colors.PanelBody
	}
	return rotheme.Default.Colors.WindowBody
}

func guildRowBackground(dark bool) widget.Color {
	return guildTableRowBackground(dark)
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

func (w *GuildWindow) positionsTab(ctx Context) widget.Widget {
	guild := guildSessionInfo(ctx.Session)
	w.ensurePositionDraft(ctx, guild.Positions)
	positions := guildSortedPositions(guild.Positions)
	rows := make([]widget.Widget, 0, len(positions)+1)
	rows = append(rows, guildPositionHeaderRow())
	if len(positions) == 0 {
		rows = append(rows,
			primitives.Box(rotheme.Text("No guild positions loaded.")).
				Height(24).
				Width(guildTableWidth).
				PaddingXY(4, 4).
				Background(rotheme.Default.Colors.WindowBody),
		)
	}
	for i, position := range positions {
		rows = append(rows, w.guildPositionRow(ctx, position, guild.IsMaster, i%2 == 0))
	}
	return primitives.Box(
		scrollview.New(
			primitives.Box(rows...),
			scrollview.DirectionOpt(scrollview.Vertical),
			scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollStep(20),
		),
	).
		PaddingXY(guildTablePadding, guildTablePadding).
		Background(rotheme.Default.Colors.WindowBody)
}

func guildPositionHeaderRow() widget.Widget {
	return primitives.HBox(
		guildMemberCell("Rank", 44, true, false),
		guildMemberCell("Position Title", 164, true, false),
		guildMemberCell("Invitation", 58, true, false),
		guildMemberCell("Punish", 48, true, false),
		guildMemberCell("Tax", 46, true, false),
		guildTableFiller(false),
	).
		Width(guildTableWidth).
		Height(16)
}

func (w *GuildWindow) guildPositionRow(ctx Context, position session.GuildPosition, isMaster, dark bool) widget.Widget {
	return primitives.HBox(
		guildMemberCell(fmt.Sprintf("%d", position.PositionID), 44, false, dark),
		w.guildPositionTitleCell(position, isMaster, dark),
		w.guildPositionRightCell(ctx, position.PositionID, 0x01, isMaster, dark),
		w.guildPositionRightCell(ctx, position.PositionID, 0x10, isMaster, dark),
		w.guildPositionTaxCell(position, isMaster, dark),
		guildTableFiller(dark),
	).
		Width(guildTableWidth).
		Height(20)
}

func (w *GuildWindow) guildPositionTitleCell(position session.GuildPosition, isMaster, dark bool) widget.Widget {
	if !isMaster {
		return guildMemberCell(guildPositionTitle(position), 164, false, dark)
	}
	return primitives.Box(
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
	).
		Width(164).
		Height(20).
		Background(guildRowBackground(dark))
}

func (w *GuildWindow) guildPositionRightCell(ctx Context, positionID, flag uint32, isMaster, dark bool) widget.Widget {
	enabled := w.positionDraft[positionID].right&flag != 0
	if !isMaster {
		return guildMemberCell(guildRightLabel(enabled), guildPositionRightWidth(flag), false, dark)
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
		Width(guildPositionRightWidth(flag)).
		Height(20).
		CrossAlign(primitives.CrossAxisCenter).
		Background(guildRowBackground(dark))
}

func (w *GuildWindow) guildPositionTaxCell(position session.GuildPosition, isMaster, dark bool) widget.Widget {
	if !isMaster {
		return guildMemberCell(fmt.Sprintf("%d %%", position.PayRate), 46, false, dark)
	}
	return primitives.Box(
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
	).
		Width(46).
		Height(20).
		Background(guildRowBackground(dark))
}

func guildPositionRightWidth(flag uint32) float32 {
	if flag == 0x10 {
		return 48
	}
	return 58
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
			rotheme.TableViewRowHeight(guildSkillRowHeight),
			rotheme.TableViewEmptyText("No guild skills loaded."),
			rotheme.TableViewScrollYSignal(w.ensureGuildSkillScrollSignal()),
			rotheme.TableViewBuildCell(func(cell rotheme.TableViewCellContext) widget.Widget {
				if cell.Row < 0 || cell.Row >= len(guild.Skills) {
					return primitives.Box()
				}
				return w.guildSkillCell(ctx, w.guildSkillWithPending(guild.Skills[cell.Row]), guild, cell)
			}),
		),
	).
		PaddingXY(guildTablePadding, guildTablePadding).
		Background(rotheme.Default.Colors.WindowBody).
		CrossAlign(primitives.CrossAxisStretch)
}

var guildSkillTableColumns = []rotheme.TableViewColumn{
	{Key: "kind", Width: 48},
	{Key: "name", Title: "Name", Width: 234},
	{Key: "level", Title: "Lv", Width: 40},
	{Key: "fill", Flex: 1},
	{Key: "levelup", Width: 22},
}

func (w *GuildWindow) guildSkillCell(ctx Context, skill session.Skill, guild session.Guild, cell rotheme.TableViewCellContext) widget.Widget {
	switch cell.Column.Key {
	case "kind":
		return w.guildSkillTooltipArea(ctx, skill,
			primitives.HBox(
				guildSkillIconCell(w.guildSkillIcon(ctx, skill)),
				primitives.HBox(
					rotheme.Text(guildSkillTypeLabel(skill)).
						Color(guildSkillTypeColor(skill)).
						Align(widget.TextAlignLeft),
				).
					Width(16).
					Height(guildSkillRowHeight).
					CrossAlign(primitives.CrossAxisCenter),
			).
				Height(guildSkillRowHeight).
				CrossAlign(primitives.CrossAxisCenter),
		)
	case "name":
		return w.guildSkillTooltipArea(ctx, skill, guildSkillTextCell(trimRunes(skillDisplayName(ctx.Resources, skill), 28), cell.Width))
	case "level":
		return w.guildSkillTooltipArea(ctx, skill, guildSkillTextCell(guildSkillLevelText(skill), cell.Width))
	case "levelup":
		return w.guildSkillLevelUpCell(skill, guild)
	default:
		return primitives.Box()
	}
}

func (w *GuildWindow) guildSkillTooltipArea(ctx Context, skill session.Skill, content widget.Widget) widget.Widget {
	return &guildSkillTooltipWidget{
		child: content,
		onHover: func(mx, my int) {
			if ctx.Input != nil {
				mx, my = ctx.Input.MouseX, ctx.Input.MouseY
			}
			w.showSkillTooltip(ctx, skill, mx, my)
		},
		onLeave: func() {
			w.hideTooltip()
		},
	}
}

func guildSkillTextCell(text string, width float32) widget.Widget {
	return primitives.HBox(
		rotheme.Text(text).
			Align(widget.TextAlignLeft),
	).
		Width(width).
		Height(guildSkillRowHeight).
		PaddingLeft(4).
		CrossAlign(primitives.CrossAxisCenter)
}

func guildSkillIconCell(icon widget.Widget) widget.Widget {
	return primitives.Box(icon).
		Width(32).
		Height(guildSkillRowHeight).
		PaddingTop((guildSkillRowHeight - guildSkillIconSize) / 2).
		CrossAlign(primitives.CrossAxisCenter)
}

func (w *GuildWindow) guildSkillLevelUpCell(skill session.Skill, guild session.Guild) widget.Widget {
	canLevelUp := w.canStageGuildSkill(skill, guild)
	var child widget.Widget = primitives.Box()
	if canLevelUp {
		child = rotheme.IconButton(rotheme.IconButtonPlus, func() {
			w.stageGuildSkill(skill.ID)
			w.refresh(w.ctx)
		})
	}
	return primitives.Box(child).
		Width(22).
		Height(guildSkillRowHeight).
		PaddingTop((guildSkillRowHeight - rotheme.IconButtonSize) / 2)
}

func (w *GuildWindow) canStageGuildSkill(skill session.Skill, guild session.Guild) bool {
	return guild.IsMaster && guild.SkillPoints > w.skillPendingCount() && skill.Upgradable && guildSkillCanLevelUp(skill)
}

func (w *GuildWindow) guildSkillIcon(ctx Context, skill session.Skill) widget.Widget {
	if img := w.guildSkillIconImage(ctx, skill); img != nil {
		return newStaticImageWidget(img, guildSkillIconSize, guildSkillIconSize)
	}
	return primitives.Box()
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
	x := w.x + guildTablePadding
	y := w.y + ROWindowTitleHeight + guildWindowTabHeight + guildTablePadding
	height := w.height - ROWindowTitleHeight - guildWindowTabHeight - ROWindowFooterHeight - guildTablePadding*2
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

type guildSkillTooltipWidget struct {
	widget.WidgetBase
	child   widget.Widget
	onHover func(mx, my int)
	onLeave func()
}

func (w *guildSkillTooltipWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := w.child.Layout(ctx, constraints)
	w.child.(interface{ SetBounds(geometry.Rect) }).SetBounds(geometry.FromPointSize(geometry.Pt(0, 0), size))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *guildSkillTooltipWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	canvas.PushTransform(bounds.Min)
	widget.DrawChild(w.child, ctx, canvas)
	canvas.PopTransform()
}

func (w *guildSkillTooltipWidget) Event(ctx widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return w.child.Event(ctx, e)
	}
	local := *mouse
	local.Position = mouse.Position.Sub(w.Bounds().Min)
	consumed := w.child.Event(ctx, &local)
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		if w.onHover != nil {
			w.onHover(int(mouse.GlobalPosition.X), int(mouse.GlobalPosition.Y))
		}
	case event.MouseLeave:
		if w.onLeave != nil {
			w.onLeave()
		}
	}
	return consumed
}

func (w *guildSkillTooltipWidget) Children() []widget.Widget {
	return []widget.Widget{w.child}
}

func guildSkillTypeLabel(skill session.Skill) string {
	if skill.Type == 0 {
		return "P"
	}
	return "A"
}

func guildSkillTypeColor(skill session.Skill) widget.Color {
	if skill.Type == 0 {
		return widget.RGBA8(34, 142, 158, 255)
	}
	return widget.RGBA8(44, 92, 184, 255)
}

func (w *GuildWindow) historyTab(ctx Context) widget.Widget {
	history := guildSessionInfo(ctx.Session).ExpelHistory
	rows := make([]widget.Widget, 0, len(history)+1)
	rows = append(rows, guildHistoryHeaderRow())
	if len(history) == 0 {
		rows = append(rows,
			primitives.Box(rotheme.Text("No expel history loaded.").Color(rotheme.Default.Colors.MutedText)).
				Height(24).
				Width(guildTableWidth).
				PaddingXY(4, 4).
				Background(rotheme.Default.Colors.WindowBody),
		)
	}
	for i, entry := range history {
		rows = append(rows, guildHistoryRow(entry, i%2 == 0))
	}
	return primitives.Box(
		scrollview.New(
			primitives.Box(rows...),
			scrollview.DirectionOpt(scrollview.Vertical),
			scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollStep(20),
		),
	).
		PaddingXY(guildTablePadding, guildTablePadding).
		Background(rotheme.Default.Colors.WindowBody)
}

func guildHistoryHeaderRow() widget.Widget {
	return primitives.HBox(
		guildMemberCell("Name", 116, true, false),
		guildMemberCell("The Reason of Expulsion", 244, true, false),
		guildTableFiller(false),
	).
		Width(guildTableWidth).
		Height(16)
}

func guildHistoryRow(entry session.GuildExpelHistory, dark bool) widget.Widget {
	return primitives.HBox(
		guildMemberCell(guildText(entry.CharName), 116, false, dark),
		guildMemberCell(guildText(entry.Reason), 244, false, dark),
		guildTableFiller(dark),
	).
		Width(guildTableWidth).
		Height(20)
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
