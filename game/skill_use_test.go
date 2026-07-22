package game

import (
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestSkillTargetModes(t *testing.T) {
	if !isGroundTargetSkill(session.Skill{ID: 18, Type: 0x02}) {
		t.Fatal("ground skill type bit should request floor target")
	}
	if !isSelfTargetSkill(session.Skill{ID: 26, Type: 0x04}) {
		t.Fatal("self skill type bit should target the player")
	}
	if isSelfTargetSkill(session.Skill{ID: 21, Type: 0x06}) {
		t.Fatal("ground bit should win over self bit")
	}
}

func TestChangeCartSkillOpensSelector(t *testing.T) {
	mode := &WorldMode{}
	controller := skillController{mode: mode}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}}

	if err := controller.Use(ctx, session.Skill{ID: skillChangeCart, Level: 1, Type: skillTargetSelf}, "test"); err != nil {
		t.Fatalf("change cart use failed: %v", err)
	}
	if !mode.ui.changeCartWindow.IsOpen() {
		t.Fatal("change cart window was not opened")
	}
	if mode.pendingSkill.skill.ID != 0 {
		t.Fatalf("pending skill = %+v, want none", mode.pendingSkill.skill)
	}
}

func TestTextGroundSkillClickOpensPrompt(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(32, 32)
	inputState := input.NewState()
	projection := newSceneProjectionForTarget(1280, 720, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(12), cellCenter(20), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{skill: session.Skill{ID: db.SkillHTTalkiebox, Name: "Talkie Box", Level: 1, Type: skillTargetPlace, Range: 9}},
	}
	ctx := client.Context{
		Input:     inputState,
		Session:   &session.Session{AccountID: 2000000, CharID: 150000},
		World:     world,
		ScreenW:   1280,
		ScreenH:   720,
		UIManager: &worldModeTestUIManager{},
	}

	mode.skills().HandleClick(ctx, projection, time.Now())

	if mode.pendingSkill.skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want cleared while prompt is open", mode.pendingSkill.skill.ID)
	}
	if mode.pendingSkillText.skill.ID != db.SkillHTTalkiebox || mode.pendingSkillText.x != 12 || mode.pendingSkillText.y != 20 {
		t.Fatalf("pending text skill = %+v", mode.pendingSkillText)
	}
	if !mode.ui.skillTextPrompt.IsOpen() {
		t.Fatal("skill text prompt was not opened")
	}
}

func TestMercenaryTargetSkillChasesFromMercenaryPosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 24, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		ObjectType:    actorObjectTypeMercenary,
		HasObjectType: true,
	})
	target := worldstate.Actor{
		ID:            300,
		X:             25,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(target)
	skill := session.Skill{ID: db.SkillMsBash, Level: 1, Type: skillTargetEnemy, Range: 1}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{
			Mercenary: session.Companion{
				ID:     400,
				Active: true,
				Skills: session.Skills{List: []session.Skill{skill}},
			},
		},
		World: world,
	}

	if !mode.skills().chaseTargetIfNeeded(ctx, skill, target, "test") {
		t.Fatal("mercenary skill did not chase even though only the player was in range")
	}
	if mode.pendingSkill.targetID != target.ID || mode.pendingSkill.skill.ID != db.SkillMsBash {
		t.Fatalf("pending mercenary skill = %+v", mode.pendingSkill)
	}
}

func TestMercenaryTargetSkillUsesRawServerRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 15, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		ObjectType:    actorObjectTypeMercenary,
		HasObjectType: true,
	})
	target := worldstate.Actor{
		ID:            300,
		X:             20,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(target)
	skill := session.Skill{ID: db.SkillMaDouble, Level: 2, Type: skillTargetEnemy, Range: 9}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{
			Mercenary: session.Companion{
				ID:     400,
				Active: true,
				Skills: session.Skills{List: []session.Skill{skill}},
			},
		},
		World: world,
	}

	if !mode.skills().chaseTargetIfNeeded(ctx, skill, target, "test") {
		t.Fatal("mercenary skill at distance 10 should chase for raw range 9")
	}
	if mode.pendingSkill.targetID != target.ID || mode.pendingSkill.skill.ID != db.SkillMaDouble {
		t.Fatalf("pending mercenary skill = %+v", mode.pendingSkill)
	}
}

func TestMercenaryPendingTargetSkillSchedulesFromMercenaryPosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             24,
		Y:             20,
		ObjectType:    actorObjectTypeMercenary,
		HasObjectType: true,
	})
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             25,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	skill := session.Skill{ID: db.SkillMsBash, Level: 1, Type: skillTargetEnemy, Range: 1}
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    skill,
			targetID: 300,
			expires:  time.Now().Add(time.Second),
		},
	}
	ctx := client.Context{
		Session: &session.Session{
			Mercenary: session.Companion{
				ID:     400,
				Active: true,
				Skills: session.Skills{List: []session.Skill{skill}},
			},
		},
		World: world,
	}

	mode.skills().UpdatePendingTarget(ctx, "test", false)

	if mode.pendingSkill.targetID != 300 {
		t.Fatal("pending mercenary skill was cleared")
	}
	if mode.pendingSkill.readyAt.IsZero() {
		t.Fatal("pending mercenary skill was not scheduled from the mercenary position")
	}
}
