package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/bot"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

// Bot callbacks remain in the world update phases so modal and death handling
// keep their existing priority. The mode manager owns the Lua state.
func (m *WorldMode) updateBot(ctx client.Context, now time.Time) {
	b := m.bot
	if b == nil || b.Disabled || b.Path != ctx.ScriptPath() || now.Before(b.NextTick) {
		return
	}
	b.Context = ctx
	b.NextTick = now.Add(bot.TickInterval)
	if err := b.Tick(); err != nil {
		b.Fail("tick", err)
	}
}

func (m *WorldMode) updateBotInput(ctx client.Context, keyboardAvailable bool) {
	b := m.bot
	if b == nil || b.Path != ctx.ScriptPath() {
		return
	}
	b.Context = ctx
	if err := b.InputFrame(keyboardAvailable); err != nil {
		b.Fail("input", err)
	}
}

func (m *WorldMode) scriptAttack(ctx client.Context, id uint32) bool {
	if scene.PlayerIsDead(ctx) || ctx.World == nil {
		return false
	}
	actor, ok := ctx.World.Actors[id]
	if !ok || !actors.ActorCanBeAttackClicked(ctx, actor) {
		return false
	}
	m.requestAttack(ctx, actor, "script")
	return true
}

func (m *WorldMode) scriptSkill(ctx client.Context, id uint32, skill session.Skill, requestedLevel int) bool {
	if ctx.World == nil {
		return false
	}
	skill = scene.NormalizeSessionSkillLevelCap(skill)
	if requestedLevel != -1 {
		if requestedLevel < 1 || requestedLevel > skill.Level {
			glog.Debugf("script skill invalid level skill=%d requested=%d learned=%d", skill.ID, requestedLevel, skill.Level)
			return false
		}
		if selectable, known := db.SkillLevelSelectable(skill.ID); known && !selectable && requestedLevel != skill.Level {
			glog.Debugf("script skill level not selectable skill=%d requested=%d learned=%d", skill.ID, requestedLevel, skill.Level)
			return false
		}
		skill.Level = requestedLevel
	}
	if skills.IsSelfTargetSkill(skill) {
		if id != scene.LocalSkillTarget(ctx) {
			return false
		}
		if err := m.skills().Use(ctx, skill, "script"); err != nil {
			glog.Debugf("script self skill failed skill=%d: %v", skill.ID, err)
			return false
		}
		return true
	}
	actor, ok, _ := scene.ActorForCombatID(ctx, id)
	_, dead := m.actorSystem.ActorDeaths[id]
	if !ok || dead || !actors.ActorCanBeSkillTargeted(ctx, skill, actor) {
		glog.Debugf("script skill invalid target skill=%d target=%d", skill.ID, id)
		return false
	}
	if err := m.skills().UseTarget(ctx, skill, actor, "script"); err != nil {
		glog.Debugf("script skill failed skill=%d target=%d: %v", skill.ID, id, err)
		return false
	}
	return true
}

func (m *WorldMode) scriptLoot(ctx client.Context, id uint32) bool {
	if ctx.World == nil {
		return false
	}
	item, ok := ctx.World.Items[id]
	if !ok {
		return false
	}
	m.clearAttackFocus()
	return m.requestPickup(ctx, item, "script")
}

func scriptUseItem(ctx client.Context, index int) bool {
	if ctx.Session == nil || index <= 0 || index > int(^uint16(0)) {
		return false
	}
	item, ok := session.FindInventoryItem(ctx.Session, uint16(index))
	if !ok {
		return false
	}
	if err := gameui.UseInventoryItem(ctx, item); err != nil {
		glog.Debugf("script item use failed index=%d item=%d: %v", item.Index, item.ItemID, err)
		return false
	}
	return true
}

func (m *WorldMode) botKeyPress(ctx client.Context, code input.KeyCode) {
	path := ctx.ScriptPath()
	if path == "" || m.bot == nil || m.bot.Path != path || m.bot.Disabled {
		return
	}
	m.bot.Context = ctx
	if err := m.bot.KeyPress(code); err != nil {
		m.bot.Fail("keypress", err)
	}
}

func (m *WorldMode) scriptWalk(ctx client.Context, x, y int) bool {
	now := time.Now()
	if ctx.World == nil || ctx.Network == nil || !m.walkReady(now) || !scene.WalkTargetInBounds(ctx, x, y) {
		return false
	}
	if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
		return false
	}
	if playerAtWalkTarget(ctx.World.Player, x, y, now) {
		return false
	}
	return m.requestMove(ctx, x, y, "script")
}

func (m *WorldMode) scriptStop(ctx client.Context) bool {
	m.cancelPlayerAction(ctx)
	return m.requestWalkStop(ctx, "script")
}

// Use the same eligibility rules as pointer targeting and use_pending_skill,
// optionally ignoring Shift when the script uses it for a different action.
func (mode *WorldMode) scriptSkillTargets(ctx client.Context, ignoreShift bool, now time.Time) []worldstate.Actor {
	var result []worldstate.Actor
	if mode == nil || ctx.World == nil {
		return result
	}
	pending := mode.skillSystem.Pending
	if pending.Skill.ID == 0 || pending.TargetID != 0 || skills.IsGroundTargetSkill(pending.Skill) || skills.IsSelfTargetSkill(pending.Skill) {
		return result
	}
	override := skills.SkillTargetOverrideActive(ctx)
	if ignoreShift {
		override = ctx.Session != nil && ctx.Session.NoShift
	}
	x, y := scene.CurrentPlayerCell(ctx, now)
	appendTarget := func(actor worldstate.Actor) {
		if _, dead := mode.actorSystem.ActorDeaths[actor.ID]; dead || !actors.ActorCanBeSkillTargetedWithOverride(ctx, pending.Skill, actor, override) {
			return
		}
		result = append(result, actor)
	}
	if actor, ok, _ := scene.ActorForCombatID(ctx, scene.LocalSkillTarget(ctx)); ok && !scene.PlayerIsDead(ctx) {
		actor.X, actor.Y = x, y
		appendTarget(actor)
	}
	for _, actor := range ctx.World.Actors {
		if !scene.IsLocalActor(ctx, actor.ID) {
			appendTarget(actor)
		}
	}
	return result
}

func (m *WorldMode) scriptUsePendingSkill(ctx client.Context, id uint32) bool {
	if m == nil || id == 0 {
		return false
	}
	pending := m.skillSystem.Pending
	if pending.Skill.ID == 0 || pending.TargetID != 0 || skills.IsGroundTargetSkill(pending.Skill) || skills.IsSelfTargetSkill(pending.Skill) {
		return false
	}
	actor, ok, _ := scene.ActorForCombatID(ctx, id)
	_, dead := m.actorSystem.ActorDeaths[id]
	if !ok || dead || !actors.ActorCanBeSkillTargeted(ctx, pending.Skill, actor) {
		return false
	}
	if err := m.skills().UseTarget(ctx, pending.Skill, actor, "script"); err != nil {
		glog.Debugf("script pending skill failed skill=%d target=%d: %v", pending.Skill.ID, id, err)
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
	actor, ok, _ := scene.ActorForCombatID(ctx, id)
	_, dead := m.actorSystem.ActorDeaths[id]
	if !ok || dead || npc.IsWarpActor(actor) || actor.Job == scene.ActorJobHiddenWarpNPC {
		return false
	}
	if m.scriptHighlight.id != id {
		m.scriptHighlight = actorHighlight{id: id, started: time.Now()}
	}
	return true
}

func (mode *WorldMode) scriptPendingSkill(ctx client.Context) (bot.PendingSkill, bool) {
	if mode == nil {
		return bot.PendingSkill{}, false
	}
	pending := mode.skillSystem.Pending
	if pending.Skill.ID == 0 || pending.TargetID != 0 {
		return bot.PendingSkill{}, false
	}
	target := "actor"
	if skills.IsGroundTargetSkill(pending.Skill) {
		target = "ground"
	} else if skills.IsSelfTargetSkill(pending.Skill) {
		target = "self"
	}
	maxLevel := pending.MaxLevel
	if maxLevel <= 0 {
		maxLevel = skills.SkillUseMaxLevel(pending.Skill)
	}
	result := bot.PendingSkill{Skill: pending.Skill, MaxLevel: maxLevel, Target: target}
	if caster, ok := skills.SkillCasterForSkill(ctx, pending.Skill); ok {
		result.CasterID = caster.ID
		result.CasterKind = skills.SkillCasterKindName(caster.Kind)
		result.CasterX, result.CasterY = skills.SkillCasterCell(caster, time.Now())
	}
	return result, true
}

type scriptUI interface {
	controlScriptUI(client.Context, string) bool
}

func (m *Manager) syncScript(ctx client.Context) {
	path := ctx.ScriptPath()
	if m.script != nil && m.script.Path != path {
		m.closeScript()
	}
	if path == "" || path == "none" {
		return
	}
	if m.script == nil {
		script, err := newLuaScript(ctx, m.mode, path)
		if err != nil {
			glog.Warnf("lua script load failed path=%q: %v", path, err)
			script = &bot.Script{Path: path, Disabled: true}
		}
		m.script = script
		if world, ok := m.mode.(*WorldMode); ok {
			world.bot = script
		}
	}
	m.script.Context = ctx
	m.script.BlockedButtons = m.scriptBlocked
}

func (m *Manager) closeScript() {
	if m.script != nil {
		m.script.Close()
		m.script = nil
	}
	if world, ok := m.mode.(*WorldMode); ok {
		world.bot = nil
	}
}

func newLuaScript(ctx client.Context, mode Mode, path string) (*bot.Script, error) {
	world, _ := mode.(*WorldMode)
	bindings := bot.Bindings{}
	if ui, ok := mode.(scriptUI); ok {
		bindings.ControlUI = ui.controlScriptUI
	}
	if world != nil {
		bindings.InputState = func(ctx client.Context) (bool, bool) {
			if world.uiInputSuspended() {
				return true, false
			}
			return false, !world.ui.KeyboardShortcutsBlocked(ctx)
		}
		bindings.ClearHighlight = world.clearScriptHighlight
		bindings.World = &bot.World{
			Walk: world.scriptWalk, Stop: world.scriptStop, Attack: world.scriptAttack,
			Skill: world.scriptSkill, Loot: world.scriptLoot, UseItem: scriptUseItem,
			UsePendingSkill: world.scriptUsePendingSkill, HighlightActor: world.scriptHighlightActor,
			SkillTargets: world.scriptSkillTargets, PendingSkill: world.scriptPendingSkill,
			CanAttack: actors.ActorCanBeAttackClicked, CompanionLife: world.companionLife,
			ActorDead:    func(id uint32) bool { _, dead := world.actorSystem.ActorDeaths[id]; return dead },
			WalkSequence: func() uint64 { return world.walkSequence },
			UseShortcut: func(ctx client.Context, slot int) (uint16, bool) {
				return world.ui.shortcutBar.ActivateSlot(ctx, world, slot)
			},
			RotateCamera: func(ctx client.Context, yaw, pitch float64) {
				if !scene.CameraRotationLockedForMap(ctx) {
					world.camera.Rotate(yaw)
					world.camera.Tilt(pitch)
				}
			},
			ZoomCamera: func(ctx client.Context, delta float64) {
				if !cameraZoomLockedForMap(ctx) {
					world.camera.ZoomByDelta(delta)
				}
			},
			CameraYaw: func(ctx client.Context) float64 {
				yaw := scene.CameraYawForMap(ctx)
				if !scene.CameraRotationLockedForMap(ctx) {
					yaw += world.camera.yawOffset
				}
				return yaw
			},
			CancelSkill: func() { world.skills().Cancel("script") },
		}
	}
	return bot.New(ctx, path, bindings)
}

func (m *WorldMode) controlScriptUI(ctx client.Context, action string) bool {
	if m.uiInputSuspended() || !m.ui.npcInputAvailable() || !m.ui.npc.Dialog.IsOpen() {
		return false
	}
	if action == "" {
		return true
	}
	return m.ui.npc.Dialog.Control(ctx, action)
}
