package ui

import (
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestHomunculusSkillWindowUsesHomunculusSkillList(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{{ID: db.SkillSMBash, Level: 10}},
		},
		Homunculus: session.Companion{
			Active: true,
			Skills: session.Skills{
				List: []session.Skill{{ID: db.SkillHvanCaprice, Level: 3}},
			},
		},
	}

	window := &HomunculusSkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	if len(skills) != 1 || skills[0].ID != db.SkillHvanCaprice {
		t.Fatalf("visible homunculus skills = %+v", skills)
	}
}

func TestHomunculusSkillWindowRejectsPassiveDrag(t *testing.T) {
	window := &HomunculusSkillWindow{}
	window.pressSkill(
		Context{},
		nil,
		session.Skill{ID: db.SkillHvanInstruct, Type: 0, Level: 5},
		20,
		30,
	)
	if window.dragActive {
		t.Fatal("passive homunculus skill started a drag")
	}
}

func TestHomunculusSkillWindowCanStageSkillHonorsMaxLevel(t *testing.T) {
	s := &session.Session{
		Homunculus: session.Companion{
			Active: true,
			Skills: session.Skills{Points: 3},
		},
	}
	window := &HomunculusSkillWindow{}
	skill := session.Skill{ID: db.SkillHvanCaprice, Level: 4, MaxLevel: 5, Upgradable: true}
	if !window.canStageSkill(s, skill) {
		t.Fatal("expected level 4/5 homunculus skill to allow one staged level")
	}
	window.stageSkill(skill.ID)
	if window.canStageSkill(s, skill) {
		t.Fatal("homunculus skill should not stage past max level")
	}
	if got := window.skillWithPending(skill).Level; got != 5 {
		t.Fatalf("displayed pending level = %d, want 5", got)
	}
}

func TestHomunculusSkillWindowCanStageSkillUsesHomunculusPoints(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{Points: 10},
		Homunculus: session.Companion{
			Active: true,
			Skills: session.Skills{Points: 2},
		},
	}
	window := &HomunculusSkillWindow{}
	skill := session.Skill{ID: db.SkillHvanCaprice, Level: 1, MaxLevel: 5, Upgradable: true}
	for i := 0; i < s.Homunculus.Skills.Points; i++ {
		if !window.canStageSkill(s, skill) {
			t.Fatalf("expected homunculus point %d to be stageable", i+1)
		}
		window.stageSkill(skill.ID)
	}
	if window.canStageSkill(s, skill) {
		t.Fatal("homunculus skill should not use player skill points")
	}
}

func TestHomunculusSkillWindowResetClearsPending(t *testing.T) {
	window := &HomunculusSkillWindow{}
	window.stageSkill(db.SkillHvanCaprice)
	window.stageSkill(db.SkillHvanCaprice)

	window.clearPending()

	if got := window.pendingCount(); got != 0 {
		t.Fatalf("pending count = %d, want 0", got)
	}
	if got := len(window.pendingOrder); got != 0 {
		t.Fatalf("pending order length = %d, want 0", got)
	}
}

func TestHomunculusSkillWindowChangingHomunculusClearsPending(t *testing.T) {
	s := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			Skills: session.Skills{Points: 1},
		},
	}
	window := &HomunculusSkillWindow{}
	window.syncHomunculusIdentity(Context{Session: s})
	window.stageSkill(db.SkillHvanCaprice)

	s.Homunculus.ID = 301
	window.syncHomunculusIdentity(Context{Session: s})

	if got := window.pendingCount(); got != 0 {
		t.Fatalf("pending count after homunculus change = %d, want 0", got)
	}
}

func TestHomunculusSkillWindowTablePlusStagesSkill(t *testing.T) {
	skill := session.Skill{ID: db.SkillHvanCaprice, Level: 1, MaxLevel: 5, Upgradable: true}
	s := &session.Session{
		Homunculus: session.Companion{
			Active: true,
			Skills: session.Skills{Points: 1},
		},
	}
	window := &HomunculusSkillWindow{}
	bounds := skillTableLevelUpButtonBounds(0)

	consumed := window.handleSkillTableRowEvent(
		nil,
		Context{Session: s},
		nil,
		[]session.Skill{skill},
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
	if got := window.pendingFor(skill.ID); got != 1 {
		t.Fatalf("pending skill levels = %d, want 1", got)
	}
	if !window.dirty {
		t.Fatal("plus press should mark the window dirty")
	}
}

func TestHomunculusSkillDragReleaseOverShortcutStoresSkill(t *testing.T) {
	inputState := input.NewState()
	bar := &ShortcutBar{}
	x, y := bar.slotBounds(Context{ScreenW: 800, ScreenH: 600}, 0)
	inputState.SetMousePosition(x+shortcutSlot/2, y+shortcutSlot/2)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)

	window := &HomunculusSkillWindow{
		dragSkill:  session.Skill{ID: db.SkillHvanCaprice, Level: 3, Type: 1},
		dragActive: true,
		dragFrom:   time.Now().Add(-time.Second),
	}
	if !window.UpdateDrag(Context{Input: inputState, ScreenW: 800, ScreenH: 600}, bar) {
		t.Fatal("homunculus skill drag release was not consumed")
	}
	if got := bar.slots[0]; got.kind != shortcutSkill || got.skillID != db.SkillHvanCaprice || got.skillLevel != 3 {
		t.Fatalf("shortcut slot = %+v, want homunculus skill level 3", got)
	}
}
