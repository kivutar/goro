package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestGuildSkillTooltipUsesSkillDetails(t *testing.T) {
	window := &GuildWindow{}
	ctx := Context{}
	skill := session.Skill{ID: db.SkillGdApproval, Name: "Approval", Level: 1, Upgradable: true}

	window.showSkillTooltip(ctx, skill, 100, 120)

	if !window.tooltip.Open() {
		t.Fatal("tooltip should be open")
	}
	if text := window.tooltip.Text(); !strings.Contains(text, "Approval") || !strings.Contains(text, "Lv 1") {
		t.Fatalf("tooltip text = %q", text)
	}
}

func TestGuildSkillTooltipHidesWhenCursorLeavesTable(t *testing.T) {
	window := &GuildWindow{tab: guildWindowTabSkills}
	window.EnsureWindow(guildWindowWidth, guildWindowHeight)
	window.showSkillTooltip(Context{}, session.Skill{ID: db.SkillGdApproval, Name: "Approval", Level: 1}, 100, 120)

	window.updateSkillTooltipHover(Context{Input: &input.State{MouseX: window.x - 10, MouseY: window.y - 10}})

	if window.tooltip.Open() {
		t.Fatal("tooltip should close when cursor leaves the guild skills table")
	}
}

func TestGuildSkillTooltipAreaUsesInputCursorPosition(t *testing.T) {
	window := &GuildWindow{}
	ctx := Context{
		Input: &input.State{MouseX: 100, MouseY: 120},
		Session: &session.Session{
			Guild: session.Guild{
				Skills: []session.Skill{{ID: db.SkillGdApproval, Name: "Approval", Level: 1}},
			},
		},
	}
	tree := window.skillsTab(ctx)
	uiCtx := widget.NewContext()
	tree.Layout(uiCtx, geometry.Tight(geometry.Sz(guildTableViewportW, 100)))
	tree.(interface{ SetBounds(geometry.Rect) }).SetBounds(geometry.NewRect(0, 0, guildTableViewportW, 100))

	tree.Event(uiCtx, event.NewMouseEvent(
		event.MouseMove,
		event.ButtonNone,
		0,
		geometry.Pt(10, guildTableHeaderH+6),
		geometry.Pt(300, 120),
		0,
	))

	const tooltipW = 292
	if got, want := window.tooltip.centerX, 100+16+tooltipW/2; got != want {
		t.Fatalf("tooltip centerX = %d, want %d", got, want)
	}
}

func TestGuildSkillLevelUpStaysPendingUntilConfirm(t *testing.T) {
	window := &GuildWindow{}

	window.stageGuildSkill(db.SkillGdApproval)
	window.stageGuildSkill(db.SkillGdApproval)
	if action := window.PopAction(); action.hasAction() {
		t.Fatalf("staging should not publish action: %+v", action)
	}

	window.confirmGuildSkillPending(Context{})
	action := window.PopAction()
	if got := action.LevelUpSkillIDs; len(got) != 2 || got[0] != db.SkillGdApproval || got[1] != db.SkillGdApproval {
		t.Fatalf("level up ids = %v", got)
	}
}

func TestGuildRelationRowUsesRightClickForDeletion(t *testing.T) {
	relation := session.GuildRelation{Relation: session.GuildRelationAlliance, GuildID: 10, Name: "Allies"}
	window := &GuildWindow{}
	row := &guildRelationRowWidget{
		relation:  relation,
		canDelete: true,
		onDelete: func() {
			window.action.DeleteRelation = &GuildRelationDelete{Relation: relation}
		},
	}
	row.Event(widget.NewContext(), event.NewMouseEvent(event.MousePress, event.ButtonRight, 0, geometry.Point{}, geometry.Point{}, 0))
	action := window.PopAction()
	if action.DeleteRelation == nil || action.DeleteRelation.Relation != relation {
		t.Fatalf("delete relation action = %+v", action)
	}
}

func TestGuildWindowSnapshotIncludesRelations(t *testing.T) {
	s := &session.Session{Guild: session.Guild{Relations: []session.GuildRelation{{Relation: 0, GuildID: 10, Name: "Allies"}}}}
	before := guildWindowSnapshot(s)
	s.Guild.Relations[0].Name = "New Allies"
	if after := guildWindowSnapshot(s); after == before {
		t.Fatal("guild relation change did not invalidate guild window snapshot")
	}
}

func TestGuildMemberPositionChangeStaysPendingUntilConfirm(t *testing.T) {
	s := &session.Session{Guild: session.Guild{
		IsMaster: true,
		Members: []session.GuildMember{
			{AccountID: 1, CharID: 10, PositionID: 0, CharName: "Kivutar"},
			{AccountID: 2, CharID: 20, PositionID: 1, CharName: "Arcer"},
		},
		Positions: []session.GuildPosition{
			{PositionID: 0, PosName: "Guild Master"},
			{PositionID: 1, PosName: "Member"},
			{PositionID: 2, PosName: "Officer"},
		},
	}}
	ctx := Context{Session: s}
	window := &GuildWindow{}

	window.stageMemberPosition(ctx, s.Guild.Members[1], 2)
	if action := window.PopAction(); action.hasAction() {
		t.Fatalf("staging should not publish action: %+v", action)
	}

	window.confirmMemberPositionChanges(ctx)
	action := window.PopAction()
	if got := action.MemberPositions; len(got) != 1 ||
		got[0].AccountID != 2 ||
		got[0].CharID != 20 ||
		got[0].PositionID != 2 {
		t.Fatalf("member positions = %+v", got)
	}
}

func TestGuildNoticeChangeStaysPendingUntilConfirm(t *testing.T) {
	s := &session.Session{GuildID: 99, Guild: session.Guild{
		IsMaster:      true,
		NoticeSubject: "Old title",
		Notice:        "Old contents",
	}}
	ctx := Context{Session: s}
	window := &GuildWindow{}
	window.ensureNoticeDraft(ctx)

	window.noticeDraft.subject = "New title"
	window.noticeDraft.notice = "New contents"
	if action := window.PopAction(); action.hasAction() {
		t.Fatalf("staging should not publish action: %+v", action)
	}

	window.confirmGuildNoticeDraft(ctx)
	action := window.PopAction()
	if !action.UpdateNotice || action.NoticeSubject != "New title" || action.Notice != "New contents" {
		t.Fatalf("notice action = %+v", action)
	}
}

func TestGuildNoticeResetRestoresSessionNotice(t *testing.T) {
	s := &session.Session{Guild: session.Guild{
		IsMaster:      true,
		NoticeSubject: "Original title",
		Notice:        "Original contents",
	}}
	ctx := Context{Session: s}
	window := &GuildWindow{}
	window.ensureNoticeDraft(ctx)
	window.noticeDraft.subject = "Changed title"
	window.noticeDraft.notice = "Changed contents"

	window.resetNoticeDraft(ctx)

	if window.noticeDraft.subject != "Original title" || window.noticeDraft.notice != "Original contents" {
		t.Fatalf("notice draft = %+v", window.noticeDraft)
	}
}

func TestGuildNoticeTabDoesNotRequestGuildMenu(t *testing.T) {
	if _, ok := guildWindowMenuRequestTab(guildWindowTabNotice); ok {
		t.Fatal("notice tab should use cached ZC_GUILD_NOTICE data, not CZ_REQ_GUILD_MENU type 5")
	}
}

func TestGuildMenuRequestTabsMatchServerTypes(t *testing.T) {
	for _, tc := range []struct {
		tab  guildWindowTab
		want uint32
	}{
		{guildWindowTabInfo, 0},
		{guildWindowTabMembers, 1},
		{guildWindowTabPositions, 2},
		{guildWindowTabSkills, 3},
		{guildWindowTabHistory, 4},
	} {
		got, ok := guildWindowMenuRequestTab(tc.tab)
		if !ok || got != tc.want {
			t.Fatalf("menu request tab for %v = %d, %t; want %d, true", tc.tab, got, ok, tc.want)
		}
	}
}

func TestGuildEmblemControlHidesEditorForMembers(t *testing.T) {
	window := &GuildWindow{}

	memberControl := window.guildEmblemControl(Context{}, session.Guild{EmblemVersion: 7})
	memberSize := memberControl.Layout(widget.NewContext(), geometry.Constraints{
		MaxWidth:  300,
		MaxHeight: 40,
	})
	if memberSize.Width != guildEmblemSize {
		t.Fatalf("member emblem control width = %.1f, want preview width %.1f", memberSize.Width, float32(guildEmblemSize))
	}

	masterControl := window.guildEmblemControl(Context{}, session.Guild{IsMaster: true, EmblemVersion: 7})
	masterSize := masterControl.Layout(widget.NewContext(), geometry.Constraints{
		MaxWidth:  300,
		MaxHeight: 40,
	})
	if masterSize.Width <= guildEmblemSize {
		t.Fatalf("master emblem control width = %.1f, want editor wider than preview %.1f", masterSize.Width, float32(guildEmblemSize))
	}
}

func TestGuildMemberPositionTransferSendsOnlyMasterChange(t *testing.T) {
	s := &session.Session{Guild: session.Guild{
		IsMaster: true,
		Members: []session.GuildMember{
			{AccountID: 1, CharID: 10, PositionID: 0, CharName: "Kivutar"},
			{AccountID: 2, CharID: 20, PositionID: 1, CharName: "Arcer"},
			{AccountID: 3, CharID: 30, PositionID: 2, CharName: "Zed"},
		},
	}}
	ctx := Context{Session: s}
	window := &GuildWindow{}
	window.ensureMemberDraft(ctx, s.Guild.Members)
	window.memberDraft[guildMemberKey{accountID: 2, charID: 20}] = 0
	window.memberDraft[guildMemberKey{accountID: 3, charID: 30}] = 1

	changes := window.memberPositionChanges(ctx)
	if len(changes) != 1 {
		t.Fatalf("member position changes = %+v, want only the master transfer", changes)
	}
	if changes[0].AccountID != 2 || changes[0].CharID != 20 || changes[0].PositionID != 0 {
		t.Fatalf("member position changes = %+v", changes)
	}
}

func TestGuildTableControlCellsKeepControlHeight(t *testing.T) {
	s := &session.Session{Guild: session.Guild{
		IsMaster: true,
		Members: []session.GuildMember{
			{AccountID: 2, CharID: 20, PositionID: 1, CharName: "Arcer"},
		},
		Positions: []session.GuildPosition{
			{PositionID: 1, PosName: "Member"},
			{PositionID: 2, PosName: "Officer"},
		},
	}}
	ctx := Context{Session: s}
	window := &GuildWindow{}
	window.ensurePositionDraft(ctx, s.Guild.Positions)

	cells := []struct {
		name string
		cell widget.Widget
	}{
		{
			name: "dropdown",
			cell: window.guildMemberPositionCell(ctx, s.Guild.Members[0], s.Guild.Positions, true, 62),
		},
		{
			name: "title textfield",
			cell: window.guildPositionTitleCell(s.Guild.Positions[0], true, 120),
		},
		{
			name: "tax textfield",
			cell: window.guildPositionTaxCell(s.Guild.Positions[0], true, 46),
		},
	}

	for _, tc := range cells {
		size := tc.cell.Layout(widget.NewContext(), geometry.Constraints{
			MaxWidth:  160,
			MaxHeight: guildTableRowH,
		})
		if size.Height != guildTableControlH {
			t.Fatalf("%s height = %.1f, want %.1f", tc.name, size.Height, float32(guildTableControlH))
		}
	}
}

func TestGuildPositionCheckboxCellCentersCheckboxHorizontally(t *testing.T) {
	s := &session.Session{Guild: session.Guild{
		IsMaster: true,
		Positions: []session.GuildPosition{
			{PositionID: 1, Right: 0x01, PosName: "Member"},
		},
	}}
	ctx := Context{Session: s}
	window := &GuildWindow{}
	window.ensurePositionDraft(ctx, s.Guild.Positions)

	const cellW = float32(58)
	cell := window.guildPositionRightCell(ctx, 1, 0x01, true, cellW)
	size := cell.Layout(widget.NewContext(), geometry.Constraints{
		MaxWidth:  160,
		MaxHeight: guildTableRowH,
	})
	if size.Width != cellW {
		t.Fatalf("cell width = %.1f, want %.1f", size.Width, cellW)
	}
	children := cell.(interface{ Children() []widget.Widget }).Children()
	if len(children) != 3 {
		t.Fatalf("checkbox cell children = %d, want 3", len(children))
	}
	checkboxBounds := children[1].(interface{ Bounds() geometry.Rect }).Bounds()
	got := checkboxBounds.Min.X + checkboxBounds.Width()/2
	want := cellW / 2
	if got < want-0.01 || got > want+0.01 {
		t.Fatalf("checkbox center x = %.1f, want %.1f", got, want)
	}
	if size.Height >= guildTableRowH {
		t.Fatalf("checkbox cell height = %.1f, want less than row height %.1f", size.Height, float32(guildTableRowH))
	}
}

func TestGuildSkillTablePlusStagesSkill(t *testing.T) {
	skill := session.Skill{ID: db.SkillGdApproval, Name: "Approval", Level: 1, MaxLevel: 5, Upgradable: true}
	guild := session.Guild{
		IsMaster:    true,
		SkillPoints: 1,
		Skills:      []session.Skill{skill},
	}
	window := &GuildWindow{}
	bounds := guildSkillLevelUpButtonBounds(0)

	consumed := window.handleGuildSkillTableRowEvent(
		nil,
		Context{Session: &session.Session{Guild: guild}},
		guild,
		guild.Skills,
		0,
		event.NewMouseEvent(
			event.MousePress,
			event.ButtonLeft,
			event.ButtonStateLeft,
			geometry.Pt(bounds.Min.X+1, bounds.Min.Y+1),
			geometry.Pt(100, 120),
			0,
		),
	)

	if !consumed {
		t.Fatal("plus press was not consumed")
	}
	if got := window.skillPending[skill.ID]; got != 1 {
		t.Fatalf("pending skill levels = %d, want 1", got)
	}
	if action := window.PopAction(); action.hasAction() {
		t.Fatalf("staging should not publish action: %+v", action)
	}
}

func TestGuildSkillTableHidesUnavailableLevelUpButton(t *testing.T) {
	window := &GuildWindow{}
	skill := session.Skill{ID: db.SkillGdApproval, Level: 1, MaxLevel: 5, Upgradable: true}
	cell := window.guildSkillCell(
		Context{},
		skill,
		session.Guild{IsMaster: true},
		rotheme.TableViewCellContext{Column: rotheme.TableViewColumn{Key: "levelup"}},
	)
	if !cell.Hidden || cell.HasIconButton {
		t.Fatalf("unavailable guild skill level-up cell = %+v, want hidden", cell)
	}
}

func TestGuildSkillWindowDoubleClickUsesSharedSkillController(t *testing.T) {
	ctx := Context{ScreenW: 800, ScreenH: 600}
	actions := &skillWindowTestRenderer{}
	window := &GuildWindow{actions: actions}
	skill := session.Skill{ID: db.SkillGdBattleorder, Type: 4, Level: 1, Range: 1}

	window.pressGuildSkill(ctx, skill)
	if actions.used.ID != 0 {
		t.Fatalf("guild skill used after first click = %+v, want none", actions.used)
	}
	if !window.dragActive || window.dragSkill.ID != skill.ID {
		t.Fatalf("guild skill drag = active %t skill %+v", window.dragActive, window.dragSkill)
	}

	window.pressGuildSkill(ctx, skill)
	if actions.used.ID != skill.ID || actions.used.Level != skill.Level {
		t.Fatalf("used guild skill = %+v, want %+v", actions.used, skill)
	}
}

func TestGuildSkillDragReleaseOverShortcutStoresSkill(t *testing.T) {
	inputState := input.NewState()
	bar := &ShortcutBar{}
	x, y := bar.slotBounds(Context{ScreenW: 800, ScreenH: 600}, 0)
	inputState.SetMousePosition(x+shortcutSlot/2, y+shortcutSlot/2)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)

	window := &GuildWindow{
		dragSkill:  session.Skill{ID: db.SkillGdRegeneration, Level: 3, Type: 4},
		dragActive: true,
		dragFrom:   time.Now().Add(-time.Second),
	}
	if !window.UpdateDrag(Context{Input: inputState, ScreenW: 800, ScreenH: 600}, bar) {
		t.Fatal("guild skill drag release was not consumed")
	}
	if got := bar.slots[0]; got.kind != shortcutSkill || got.skillID != db.SkillGdRegeneration || got.skillLevel != 3 {
		t.Fatalf("shortcut slot = %+v, want guild Regeneration level 3", got)
	}
}

func TestGuildSkillWindowRejectsPassiveDrag(t *testing.T) {
	window := &GuildWindow{}
	window.pressGuildSkill(Context{}, session.Skill{ID: db.SkillGdApproval, Type: 0, Level: 1})
	if window.dragActive {
		t.Fatal("passive guild skill started a drag")
	}
}

func TestGuildSkillDragUsesLearnedRatherThanPendingLevel(t *testing.T) {
	skill := session.Skill{ID: db.SkillGdRegeneration, Type: 4, Level: 1}
	window := &GuildWindow{
		skillPending: map[uint16]int{skill.ID: 2},
	}
	guild := session.Guild{Skills: []session.Skill{skill}}

	window.handleGuildSkillTableRowEvent(
		nil,
		Context{},
		guild,
		guild.Skills,
		0,
		event.NewMouseEvent(
			event.MousePress,
			event.ButtonLeft,
			event.ButtonStateLeft,
			geometry.Pt(1, 1),
			geometry.Pt(100, 120),
			0,
		),
	)

	if !window.dragActive || window.dragSkill.Level != 1 {
		t.Fatalf("dragged guild skill = %+v, want learned level 1", window.dragSkill)
	}
}
