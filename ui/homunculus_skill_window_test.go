package ui

import (
	"testing"
	"time"

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
