package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
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
