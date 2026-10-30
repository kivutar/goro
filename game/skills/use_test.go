package skills

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestSkillTargetModes(t *testing.T) {
	if !IsGroundTargetSkill(session.Skill{ID: 18, Type: 0x02}) {
		t.Fatal("ground skill type bit should request floor target")
	}
	if !IsSelfTargetSkill(session.Skill{ID: 26, Type: 0x04}) {
		t.Fatal("self skill type bit should target the player")
	}
	if IsSelfTargetSkill(session.Skill{ID: 21, Type: 0x06}) {
		t.Fatal("ground bit should win over self bit")
	}
}

func TestDeadPlayerCannotStartSkillUse(t *testing.T) {
	mode := &System{}
	err := mode.Use(client.Context{
		Session: &session.Session{Dead: true},
	}, session.Skill{ID: 28, Type: SkillTargetSelf, Level: 1}, "test")

	if err == nil {
		t.Fatal("dead player started a skill")
	}
}

func TestTargetSkillPendingLevelUsesDBMaxBeforeResourceMax(t *testing.T) {
	mode := &System{}
	controller := mode
	skill := session.Skill{ID: db.SkillHTBlitzbeat, Level: 10, MaxLevel: 10, Type: SkillTargetEnemy, Range: 9}

	if err := controller.Use(client.Context{}, skill, "test"); err != nil {
		t.Fatalf("blitz beat use failed: %v", err)
	}
	if mode.Pending.Skill.Level != 5 || mode.Pending.MaxLevel != 5 {
		t.Fatalf("pending blitz beat = %+v, want level/max capped to db max 5", mode.Pending)
	}

	inputState := input.NewState()
	inputState.AddWheel(0, 20)
	if !mode.AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("pending skill wheel was not consumed")
	}
	if mode.Pending.Skill.Level != 5 {
		t.Fatalf("pending blitz beat level = %d, want capped to 5", mode.Pending.Skill.Level)
	}
}

func TestAutoRunTargetSkillStartsTargetSelection(t *testing.T) {
	mode := &System{}
	mode.ApplyAutoRun(client.Context{}, network.AutoRunSkill{Skill: network.SkillInfo{
		ID:    db.SkillALLResurrection,
		Type:  SkillTargetFriend,
		Level: 1,
		Range: 9,
		Name:  "Resurrection",
	}})

	if mode.Pending.Skill.ID != db.SkillALLResurrection {
		t.Fatalf("pending skill = %+v, want Resurrection", mode.Pending.Skill)
	}
	if mode.Pending.Skill.Level != 1 || mode.Pending.Skill.Type != SkillTargetFriend {
		t.Fatalf("pending Resurrection = %+v", mode.Pending.Skill)
	}
}

func TestSkillCasterRangeMatchesReference(t *testing.T) {
	skill := session.Skill{ID: db.SkillALHeal, Range: 9}
	for _, tc := range []struct {
		name    string
		kind    skillCasterKind
		dx, dy  int
		inRange bool
	}{
		{"player cardinal allowance", skillCasterPlayer, 10, 0, true},
		{"player diagonal inside circle", skillCasterPlayer, 7, 7, true},
		{"player square corner outside circle", skillCasterPlayer, 9, 9, false},
		{"player beyond allowance", skillCasterPlayer, 11, 0, false},
		{"mercenary square corner", skillCasterMercenary, 9, 9, true},
		{"mercenary no allowance", skillCasterMercenary, 10, 0, false},
		{"homunculus square corner", skillCasterHomunculus, 9, 9, true},
		{"homunculus no allowance", skillCasterHomunculus, 10, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := skillCaster{Kind: tc.kind}
			rangeCells := skillTargetRangeForCaster(caster, skill)
			if got := caster.targetWithinRange(20, 20, 20+tc.dx, 20+tc.dy, rangeCells); got != tc.inRange {
				t.Fatalf("in range = %t, want %t (range %d)", got, tc.inRange, rangeCells)
			}
		})
	}
}

func TestTargetSkillRangeUsesMovingActorCurrentCell(t *testing.T) {
	now := time.Now()
	target := worldstate.Actor{
		ID:           300,
		X:            30,
		Y:            10,
		FromX:        20,
		FromY:        10,
		ToX:          30,
		ToY:          10,
		Moving:       true,
		MoveStarted:  now,
		MoveDuration: 10 * time.Second,
		MovePath: []worldstate.WalkStep{
			{X: 20, Y: 10},
			{X: 30, Y: 10},
		},
	}

	targetX, targetY := scene.ActorCurrentCell(target, now)
	if !targetSkillWithinRangeCells(11, 10, targetX, targetY, 9) {
		t.Fatal("moving target should be in range at its current rendered cell")
	}
	if targetSkillWithinRangeCells(11, 10, target.X, target.Y, 9) {
		t.Fatal("test target final destination should be out of range")
	}
}

func TestTeleportModalRules(t *testing.T) {
	if !gameui.TeleportWarpListBypassesModal(network.WarpPointList{SkillID: 26, MapNames: []string{"Random"}}) {
		t.Fatal("Teleport level 1 should bypass the modal")
	}
	if gameui.TeleportWarpListBypassesModal(network.WarpPointList{SkillID: 26, MapNames: []string{"Random", "prontera"}}) {
		t.Fatal("Teleport level 2 with a save point should show the modal")
	}
}
