package game

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type skillController struct {
	mode *WorldMode
}

type skillCasterKind uint8

const (
	skillCasterPlayer skillCasterKind = iota
	skillCasterHomunculus
	skillCasterMercenary
)

type skillCaster struct {
	kind  skillCasterKind
	id    uint32
	actor worldstate.Actor
}

func (m *WorldMode) skills() skillController {
	return skillController{mode: m}
}

func skillByID(s *session.Session, skillID uint16) (session.Skill, bool) {
	if s == nil || skillID == 0 {
		return session.Skill{}, false
	}
	for _, skill := range s.Skills.List {
		if skill.ID == skillID {
			return skill, true
		}
	}
	return session.Skill{}, false
}

func skillLabel(skill session.Skill) string {
	if skill.Name != "" {
		return skill.Name
	}
	return fmt.Sprintf("Skill %d", skill.ID)
}

func sessionSkillFromNetwork(skill network.SkillInfo) session.Skill {
	return session.Skill{
		ID:         skill.ID,
		Type:       skill.Type,
		Level:      skill.Level,
		SPCost:     skill.SPCost,
		Range:      skill.Range,
		Name:       skill.Name,
		Upgradable: skill.Upgradable,
	}
}

func sessionSkillFromNetworkWithResources(manager *res.Manager, skill network.SkillInfo) session.Skill {
	out := sessionSkillFromNetwork(skill)
	if manager != nil {
		if maxLevel, ok := manager.SkillMaxLevel(int(skill.ID)); ok {
			out.MaxLevel = maxLevel
		}
	}
	return out
}

func localSkillTarget(ctx client.Context) uint32 {
	if ctx.Session == nil {
		return 0
	}
	if ctx.Session.AccountID != 0 {
		return ctx.Session.AccountID
	}
	return ctx.Session.CharID
}

func isGroundTargetSkill(skill session.Skill) bool {
	return skill.Type&skillTargetPlace != 0 || skillForcesGroundTarget(skill.ID)
}

func isSelfTargetSkill(skill session.Skill) bool {
	return skillForcesSelfTarget(skill.ID) || (skill.Type&skillTargetSelf != 0 && !isGroundTargetSkill(skill))
}

func skillCasterKindForID(skillID uint16) skillCasterKind {
	switch {
	case skillID > db.SkillHomunBegin && skillID < db.SkillHomunLast:
		return skillCasterHomunculus
	case skillID > db.SkillMercenaryBegin && skillID < db.SkillMercenaryLast:
		return skillCasterMercenary
	default:
		return skillCasterPlayer
	}
}

func skillCasterKindName(kind skillCasterKind) string {
	switch kind {
	case skillCasterHomunculus:
		return "homunculus"
	case skillCasterMercenary:
		return "mercenary"
	default:
		return "player"
	}
}

func skillCasterForSkill(ctx client.Context, skill session.Skill) (skillCaster, bool) {
	if ctx.World == nil {
		return skillCaster{}, false
	}
	kind := skillCasterKindForID(skill.ID)
	switch kind {
	case skillCasterHomunculus:
		if ctx.Session == nil || !ctx.Session.Homunculus.Active {
			return skillCaster{}, false
		}
		id := ctx.Session.Homunculus.ID
		if id == 0 {
			id = findCompanionActorID(ctx, actorObjectTypeHomunculus)
		}
		actor, ok := companionActorByID(ctx, id)
		if !ok {
			return skillCaster{}, false
		}
		return skillCaster{kind: kind, id: id, actor: actor}, true
	case skillCasterMercenary:
		if ctx.Session == nil || !ctx.Session.Mercenary.Active {
			return skillCaster{}, false
		}
		id := ctx.Session.Mercenary.ID
		if id == 0 {
			id = findCompanionActorID(ctx, actorObjectTypeMercenary)
		}
		actor, ok := companionActorByID(ctx, id)
		if !ok {
			return skillCaster{}, false
		}
		return skillCaster{kind: kind, id: id, actor: actor}, true
	default:
		actor := ctx.World.Player
		if actor.ID == 0 {
			actor.ID = localSkillTarget(ctx)
		}
		return skillCaster{kind: kind, id: actor.ID, actor: actor}, true
	}
}

func skillCasterCell(caster skillCaster, now time.Time) (int, int) {
	if caster.kind == skillCasterPlayer {
		x, y := actorRenderPosition(caster.actor, now)
		return int(math.Round(x)), int(math.Round(y))
	}
	return companionActorRenderCell(caster.actor, now)
}

func skillTargetRangeForCaster(caster skillCaster, skill session.Skill) int {
	if caster.kind != skillCasterPlayer {
		if skill.Range > 0 {
			return skill.Range
		}
		level := maxInt(1, skill.Level)
		if attackRange, ok := db.SkillAttackRange(skill.ID, level); ok {
			return attackRange
		}
		if caster.actor.AttackRange > 0 {
			return caster.actor.AttackRange
		}
	}
	return targetSkillRange(skill)
}

const (
	skillTargetEnemy  = 1
	skillTargetPlace  = 2
	skillTargetSelf   = 4
	skillTargetFriend = 16
	skillTargetTrap   = 32
	skillTargetPet    = 64
	skillTargetHomun  = 128
)

const skillChangeCart = 154
const skillGroundTextMaxBytes = 79

func (c skillController) Use(ctx client.Context, skill session.Skill, source string) error {
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if skill.Type == 0 || skillForcesPassive(skill.ID) {
		return fmt.Errorf("passive skill")
	}
	if skill.ID == skillChangeCart {
		if c.mode == nil {
			return fmt.Errorf("missing world mode")
		}
		target := localSkillTarget(ctx)
		if target == 0 {
			return fmt.Errorf("missing skill target")
		}
		if ctx.Network != nil {
			level := uint16(maxInt(1, skill.Level))
			if err := ctx.Network.SendUseSkillToID(skill.ID, level, target); err != nil {
				return err
			}
		}
		c.mode.ui.changeCartWindow.Open(ctx)
		glog.Debugf("%s skill opens change cart selector skill=%d", source, skill.ID)
		return nil
	}
	if isSelfTargetSkill(skill) {
		target := localSkillTarget(ctx)
		if target == 0 {
			return fmt.Errorf("missing skill target")
		}
		return c.SendToID(ctx, skill, target, source)
	}
	if skill.Range > 0 || isGroundTargetSkill(skill) {
		c.mode.pendingSkill = pendingSkillTarget{skill: skill, maxLevel: maxInt(1, skill.Level), started: time.Now()}
		glog.Debugf("%s skill target pending skill=%d level=%d range=%d", source, skill.ID, skill.Level, skill.Range)
		return nil
	}
	target := localSkillTarget(ctx)
	if target == 0 {
		return fmt.Errorf("missing skill target")
	}
	return c.SendToID(ctx, skill, target, source)
}

func (c skillController) SendToID(ctx client.Context, skill session.Skill, target uint32, source string) error {
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if target == 0 {
		return fmt.Errorf("missing skill target")
	}
	level := uint16(maxInt(1, skill.Level))
	glog.Debugf("%s skill use skill=%d level=%d target=%d", source, skill.ID, level, target)
	if err := ctx.Network.SendUseSkillToID(skill.ID, level, target); err != nil {
		return err
	}
	if gameui.IsLevelOneTeleportSkill(skill) {
		c.mode.addWorldEffect(ctx, effectTeleportation, localSkillTarget(ctx))
	}
	return nil
}

func (c skillController) SendToGround(ctx client.Context, skill session.Skill, x, y int, source string) error {
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if !walkTargetInBounds(ctx, x, y) {
		return fmt.Errorf("invalid ground target %d,%d", x, y)
	}
	level := uint16(maxInt(1, skill.Level))
	glog.Debugf("%s ground skill use skill=%d level=%d target=%d,%d", source, skill.ID, level, x, y)
	if err := ctx.Network.SendUseSkillToGround(skill.ID, level, x, y); err != nil {
		return err
	}
	return nil
}

func (c skillController) SendToGroundWithText(ctx client.Context, skill session.Skill, x, y int, text string, source string) error {
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if !walkTargetInBounds(ctx, x, y) {
		return fmt.Errorf("invalid ground target %d,%d", x, y)
	}
	level := uint16(maxInt(1, skill.Level))
	glog.Debugf("%s ground skill with text use skill=%d level=%d target=%d,%d text_len=%d", source, skill.ID, level, x, y, len([]byte(text)))
	if err := ctx.Network.SendUseSkillToGroundWithText(skill.ID, level, x, y, text); err != nil {
		return err
	}
	return nil
}

func (c skillController) CancelFromInput(ctx client.Context) bool {
	if c.mode.pendingSkill.skill.ID == 0 || ctx.Input == nil {
		return false
	}
	if !ctx.Input.JustPressed(input.KeyEscape) && !ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		return false
	}
	c.Cancel("input")
	return true
}

func (c skillController) Cancel(source string) {
	if c.mode.pendingSkill.skill.ID == 0 {
		return
	}
	glog.Debugf("skill target canceled skill=%d source=%s", c.mode.pendingSkill.skill.ID, source)
	c.mode.pendingSkill = pendingSkillTarget{}
}

func (c skillController) AdjustPendingLevelFromWheel(ctx client.Context) bool {
	if ctx.Input == nil || ctx.Input.WheelY == 0 || c.mode.pendingSkill.skill.ID == 0 {
		return false
	}
	pending := c.mode.pendingSkill
	maxLevel := pending.maxLevel
	if maxLevel <= 0 {
		maxLevel = maxInt(1, pending.skill.Level)
	}
	step := int(math.Ceil(math.Abs(ctx.Input.WheelY)))
	if step < 1 {
		step = 1
	}
	level := pending.skill.Level
	if ctx.Input.WheelY > 0 {
		level += step
	} else {
		level -= step
	}
	level = clampInt(level, 1, maxLevel)
	ctx.Input.WheelY = 0
	if level == pending.skill.Level {
		return true
	}
	pending.skill.Level = level
	c.mode.pendingSkill = pending
	glog.Debugf("skill target level changed skill=%d level=%d max=%d", pending.skill.ID, level, maxLevel)
	return true
}

func (c skillController) HandleClick(ctx client.Context, projection sceneProjection, now time.Time) {
	skill := c.mode.pendingSkill.skill
	if skill.ID == 0 {
		return
	}
	if isGroundTargetSkill(skill) {
		targetX, targetY, ok := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY)
		if !ok {
			glog.Debugf("skill ground target miss skill=%d mouse=%d,%d", skill.ID, ctx.Input.MouseX, ctx.Input.MouseY)
			return
		}
		if skillNeedsGroundText(skill.ID) {
			c.mode.openSkillTextPrompt(ctx, skill, targetX, targetY, "target")
			c.mode.pendingSkill = pendingSkillTarget{}
			glog.Debugf("skill ground text prompt opened skill=%d target=%d,%d", skill.ID, targetX, targetY)
			return
		}
		if err := c.SendToGround(ctx, skill, targetX, targetY, "target"); err != nil {
			glog.Warnf("skill ground target failed skill=%d target=%d,%d: %v", skill.ID, targetX, targetY, err)
			return
		}
		c.mode.pendingSkill = pendingSkillTarget{}
		glog.Debugf("skill ground target sent skill=%d target=%d,%d", skill.ID, targetX, targetY)
		return
	}
	actor, ok := clickedSkillTarget(ctx, projection, skill, ctx.Input.MouseX, ctx.Input.MouseY, now, c.mode.actorDeaths)
	if !ok {
		if x, y, groundOK := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); groundOK {
			glog.Debugf("skill target canceled by ground click skill=%d mouse=%d,%d target=%d,%d", skill.ID, ctx.Input.MouseX, ctx.Input.MouseY, x, y)
			c.Cancel("ground-click")
			return
		}
		glog.Debugf("skill target miss skill=%d mouse=%d,%d", skill.ID, ctx.Input.MouseX, ctx.Input.MouseY)
		return
	}
	if c.chaseTargetIfNeeded(ctx, skill, actor, "target") {
		return
	}
	if err := c.SendToID(ctx, skill, actor.ID, "target"); err != nil {
		glog.Warnf("skill target failed skill=%d target=%d: %v", skill.ID, actor.ID, err)
		return
	}
	c.mode.pendingSkill = pendingSkillTarget{}
	glog.Debugf("skill target sent skill=%d target=%d name=%q job=%d object_type=%d", skill.ID, actor.ID, actor.Name, actor.Job, actor.ObjectType)
}

func skillNeedsGroundText(skillID uint16) bool {
	return skillID == db.SkillHTTalkiebox || skillID == db.SkillRGGraffiti
}

func (c skillController) chaseTargetIfNeeded(ctx client.Context, skill session.Skill, actor worldstate.Actor, source string) bool {
	if ctx.World == nil || skill.Range <= 0 {
		return false
	}
	caster, ok := skillCasterForSkill(ctx, skill)
	if !ok {
		glog.Warnf("%s skill caster missing skill=%d kind=%s target=%d", source, skill.ID, skillCasterKindName(skillCasterKindForID(skill.ID)), actor.ID)
		return true
	}
	now := time.Now()
	casterX, casterY := skillCasterCell(caster, now)
	skillRange := skillTargetRangeForCaster(caster, skill)
	if targetSkillWithinRangeFrom(casterX, casterY, skillRange, actor) {
		return false
	}
	targetX, targetY, ok := attackApproachCellFrom(ctx, casterX, casterY, actor, skillRange)
	if !ok {
		glog.Warnf("%s skill chase blocked skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", source, skill.ID, skillCasterKindName(caster.kind), caster.id, casterX, casterY, actor.ID, actor.X, actor.Y, skillRange)
		c.mode.setWalkCooldown(walkRequestCooldown)
		return true
	}
	c.mode.pendingSkill = pendingSkillTarget{
		skill:    skill,
		maxLevel: c.mode.pendingSkill.maxLevel,
		targetID: actor.ID,
		expires:  time.Now().Add(8 * time.Second),
		source:   source,
		started:  c.mode.pendingSkill.started,
	}
	if c.mode.pendingSkill.started.IsZero() {
		c.mode.pendingSkill.started = time.Now()
	}
	glog.Debugf("%s skill chase target skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d chase=%d,%d", source, skill.ID, skillCasterKindName(caster.kind), caster.id, casterX, casterY, actor.ID, actor.X, actor.Y, skillRange, targetX, targetY)
	c.requestSkillCasterMove(ctx, caster, targetX, targetY, source+" skill chase")
	return true
}

func (c skillController) requestSkillCasterMove(ctx client.Context, caster skillCaster, targetX, targetY int, source string) {
	if caster.kind == skillCasterPlayer {
		c.mode.requestWalk(ctx, targetX, targetY, source)
		return
	}
	if ctx.Network == nil {
		glog.Warnf("%s companion move failed caster=%s caster_id=%d target=%d,%d: not connected", source, skillCasterKindName(caster.kind), caster.id, targetX, targetY)
		return
	}
	if !walkTargetInBounds(ctx, targetX, targetY) {
		glog.Warnf("%s companion move blocked caster=%s caster_id=%d target=%d,%d", source, skillCasterKindName(caster.kind), caster.id, targetX, targetY)
		return
	}
	if err := ctx.Network.SendCompanionMove(caster.id, targetX, targetY); err != nil {
		glog.Warnf("%s companion move failed caster=%s caster_id=%d target=%d,%d: %v", source, skillCasterKindName(caster.kind), caster.id, targetX, targetY, err)
	}
}

func (c skillController) ContinuePendingTarget(ctx client.Context, source string) {
	c.UpdatePendingTarget(ctx, source, true)
}

func (c skillController) UpdatePendingTarget(ctx client.Context, source string, logOutOfRange bool) {
	pending := c.mode.pendingSkill
	if pending.skill.ID == 0 || pending.targetID == 0 || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(pending.expires) {
		glog.Debugf("%s pending skill expired skill=%d target=%d", source, pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	actor, ok := ctx.World.Actors[pending.targetID]
	if !ok {
		glog.Debugf("%s pending skill target vanished skill=%d target=%d", source, pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	caster, ok := skillCasterForSkill(ctx, pending.skill)
	if !ok {
		glog.Debugf("%s pending skill caster vanished skill=%d kind=%s target=%d", source, pending.skill.ID, skillCasterKindName(skillCasterKindForID(pending.skill.ID)), pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	casterX, casterY := skillCasterCell(caster, now)
	skillRange := skillTargetRangeForCaster(caster, pending.skill)
	if !targetSkillWithinRangeFrom(casterX, casterY, skillRange, actor) {
		if logOutOfRange {
			glog.Debugf("%s pending skill still out of range skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", source, pending.skill.ID, skillCasterKindName(caster.kind), caster.id, casterX, casterY, actor.ID, actor.X, actor.Y, skillRange)
		}
		return
	}
	readyAt := pendingAttackReadyAt(caster.actor, now)
	if pending.readyAt.IsZero() {
		pending.readyAt = readyAt
	}
	c.mode.pendingSkill = pending
	if logOutOfRange {
		glog.Debugf("%s pending skill scheduled skill=%d caster=%s caster_id=%d target=%d delay_ms=%d", source, pending.skill.ID, skillCasterKindName(caster.kind), caster.id, pending.targetID, maxInt(0, int(pending.readyAt.Sub(now).Milliseconds())))
	}
}

func (c skillController) ProcessPendingTarget(ctx client.Context) {
	pending := c.mode.pendingSkill
	if pending.skill.ID == 0 || pending.targetID == 0 || pending.readyAt.IsZero() || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(pending.expires) {
		glog.Debugf("pending skill expired skill=%d target=%d", pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	if now.Before(pending.readyAt) {
		return
	}
	actor, ok := ctx.World.Actors[pending.targetID]
	if !ok {
		glog.Debugf("pending skill target vanished skill=%d target=%d", pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	caster, ok := skillCasterForSkill(ctx, pending.skill)
	if !ok {
		glog.Debugf("pending skill caster vanished skill=%d kind=%s target=%d", pending.skill.ID, skillCasterKindName(skillCasterKindForID(pending.skill.ID)), pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	casterX, casterY := skillCasterCell(caster, now)
	skillRange := skillTargetRangeForCaster(caster, pending.skill)
	if !targetSkillWithinRangeFrom(casterX, casterY, skillRange, actor) {
		glog.Debugf("pending skill became out of range skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", pending.skill.ID, skillCasterKindName(caster.kind), caster.id, casterX, casterY, actor.ID, actor.X, actor.Y, skillRange)
		pending.readyAt = time.Time{}
		c.mode.pendingSkill = pending
		c.chaseTargetIfNeeded(ctx, pending.skill, actor, "pending")
		return
	}
	c.mode.pendingSkill = pendingSkillTarget{}
	if err := c.SendToID(ctx, pending.skill, actor.ID, "pending"); err != nil {
		glog.Warnf("pending skill failed skill=%d target=%d: %v", pending.skill.ID, actor.ID, err)
		return
	}
	glog.Debugf("pending skill sent skill=%d target=%d name=%q job=%d object_type=%d", pending.skill.ID, actor.ID, actor.Name, actor.Job, actor.ObjectType)
}

func (c skillController) ApplyAutoRun(ctx client.Context, auto network.AutoRunSkill) {
	skill := sessionSkillFromNetwork(auto.Skill)
	target := localSkillTarget(ctx)
	glog.Debugf("auto-run skill received skill=%d level=%d range=%d name=%q target=%d", skill.ID, skill.Level, skill.Range, skill.Name, target)
	if target == 0 {
		return
	}
	if err := c.SendToID(ctx, skill, target, "auto"); err != nil {
		glog.Warnf("auto-run skill use failed skill=%d target=%d: %v", skill.ID, target, err)
		return
	}
}

func skillTargetOverrideActive(ctx client.Context) bool {
	return (ctx.Input != nil && ctx.Input.Pressed(input.KeyShift)) || (ctx.Session != nil && ctx.Session.NoShift)
}

func targetSkillRange(skill session.Skill) int {
	return maxInt(1, skill.Range)
}

func targetSkillWithinRange(ctx client.Context, skill session.Skill, actor worldstate.Actor) bool {
	if ctx.World == nil {
		return false
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	return targetSkillWithinRangeFrom(playerX, playerY, targetSkillRange(skill), actor)
}

func targetSkillWithinRangeFrom(sourceX, sourceY, skillRange int, actor worldstate.Actor) bool {
	return attackTargetWithinRange(sourceX, sourceY, actor.X, actor.Y, skillRange)
}
