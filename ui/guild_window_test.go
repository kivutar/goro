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
		geometry.Pt(guildTablePadding+10, guildTablePadding+30),
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
