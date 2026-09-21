package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	lua "github.com/yuin/gopher-lua"
)

func (m *WorldMode) scriptUsePendingSkill(ctx client.Context, id uint32) bool {
	if m == nil || id == 0 {
		return false
	}
	pending := m.pendingSkill
	if pending.skill.ID == 0 || pending.targetID != 0 || isGroundTargetSkill(pending.skill) || isSelfTargetSkill(pending.skill) {
		return false
	}
	actor, ok, _ := actorForCombatID(ctx, id)
	_, dead := m.actorDeaths[id]
	if !ok || dead || !actorCanBeSkillTargeted(ctx, pending.skill, actor) {
		return false
	}
	if err := m.skills().UseTarget(ctx, pending.skill, actor, "script"); err != nil {
		glog.Debugf("script pending skill failed skill=%d target=%d: %v", pending.skill.ID, id, err)
		return false
	}
	return true
}

func (m *WorldMode) scriptHighlightActor(ctx client.Context, id uint32) bool {
	if m == nil {
		return false
	}
	if id == 0 {
		m.clearScriptHighlight()
		return true
	}
	actor, ok, _ := actorForCombatID(ctx, id)
	_, dead := m.actorDeaths[id]
	if !ok || dead || isWarpActor(actor) || actor.Job == actorJobHiddenWarpNPC {
		return false
	}
	if m.scriptHighlight.id != id {
		m.scriptHighlight = actorHighlight{id: id, started: time.Now()}
	}
	return true
}

func luaPendingSkill(L *lua.LState, ctx client.Context, mode *WorldMode) lua.LValue {
	if mode == nil {
		return lua.LNil
	}
	pending := mode.pendingSkill
	if pending.skill.ID == 0 || pending.targetID != 0 {
		return lua.LNil
	}
	target := "actor"
	if isGroundTargetSkill(pending.skill) {
		target = "ground"
	} else if isSelfTargetSkill(pending.skill) {
		target = "self"
	}
	maxLevel := pending.maxLevel
	if maxLevel <= 0 {
		maxLevel = skillUseMaxLevel(pending.skill)
	}
	result := L.NewTable()
	result.RawSetString("id", lua.LNumber(pending.skill.ID))
	result.RawSetString("name", lua.LString(pending.skill.Name))
	result.RawSetString("level", lua.LNumber(pending.skill.Level))
	result.RawSetString("max_level", lua.LNumber(maxLevel))
	result.RawSetString("type", lua.LNumber(pending.skill.Type))
	result.RawSetString("range", lua.LNumber(pending.skill.Range))
	result.RawSetString("target", lua.LString(target))
	if caster, ok := skillCasterForSkill(ctx, pending.skill); ok {
		x, y := skillCasterCell(caster, time.Now())
		result.RawSetString("caster_id", lua.LNumber(caster.id))
		result.RawSetString("caster_kind", lua.LString(skillCasterKindName(caster.kind)))
		result.RawSetString("caster_x", lua.LNumber(x))
		result.RawSetString("caster_y", lua.LNumber(y))
	}
	return result
}

func luaOptionalActorID(L *lua.LState, index int) uint32 {
	if L.Get(index) == lua.LNil {
		return 0
	}
	value := float64(L.CheckNumber(index))
	if value < 0 || value > float64(^uint32(0)) || value != math.Trunc(value) {
		L.ArgError(index, "actor id must be a non-negative integer")
		return 0
	}
	return uint32(value)
}
