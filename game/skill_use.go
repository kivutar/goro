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

func knownSkillMaxLevel(skill session.Skill) int {
	if maxLevel, ok := db.SkillMaxLevel(skill.ID); ok {
		return maxLevel
	}
	if skill.MaxLevel > 0 {
		return skill.MaxLevel
	}
	return 0
}

func normalizeSessionSkillLevelCap(skill session.Skill) session.Skill {
	maxLevel := knownSkillMaxLevel(skill)
	if maxLevel <= 0 {
		return skill
	}
	skill.MaxLevel = maxLevel
	if skill.Level > maxLevel {
		skill.Level = maxLevel
	}
	return skill
}

func skillUseMaxLevel(skill session.Skill) int {
	if maxLevel := knownSkillMaxLevel(skill); maxLevel > 0 {
		return maxLevel
	}
	return maxInt(1, skill.Level)
}

func skillUseLevel(skill session.Skill) uint16 {
	skill = normalizeSessionSkillLevelCap(skill)
	return uint16(maxInt(1, skill.Level))
}

func sessionSkillFromNetwork(skill network.SkillInfo) session.Skill {
	return normalizeSessionSkillLevelCap(session.Skill{
		ID:         skill.ID,
		Type:       skill.Type,
		Level:      skill.Level,
		SPCost:     skill.SPCost,
		Range:      skill.Range,
		Name:       skill.Name,
		Upgradable: skill.Upgradable,
	})
}

func sessionSkillFromNetworkWithResources(manager *res.Manager, skill network.SkillInfo) session.Skill {
	out := sessionSkillFromNetwork(skill)
	if manager != nil {
		if maxLevel, ok := manager.SkillMaxLevel(int(skill.ID)); ok {
			out.MaxLevel = maxLevel
		}
	}
	return normalizeSessionSkillLevelCap(out)
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
		return 1
	}
	return targetSkillRange(skill)
}

func (caster skillCaster) targetWithinRange(sourceX, sourceY, targetX, targetY, skillRange int) bool {
	if caster.kind == skillCasterPlayer {
		return targetSkillWithinRangeCells(sourceX, sourceY, targetX, targetY, skillRange)
	}
	// rAthena uses square range for non-player casters.
	return attackTargetWithinRange(sourceX, sourceY, targetX, targetY, skillRange)
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
	if playerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	skill = normalizeSessionSkillLevelCap(skill)
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
			level := skillUseLevel(skill)
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
		c.mode.pendingSkill = pendingSkillTarget{skill: skill, maxLevel: skillUseMaxLevel(skill), started: time.Now()}
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
	if playerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if target == 0 {
		return fmt.Errorf("missing skill target")
	}
	level := skillUseLevel(skill)
	glog.Debugf("%s skill use skill=%d level=%d target=%d", source, skill.ID, level, target)
	if err := ctx.Network.SendUseSkillToID(skill.ID, level, target); err != nil {
		return err
	}
	c.mode.rememberSenseTarget(skill.ID, target, time.Now())
	if gameui.IsLevelOneTeleportSkill(skill) {
		// Level-one Teleport is the direct, random destination variant. Queue
		// the selection with the cast instead of waiting for the server's warp
		// list; servers that still send the list are handled by the normal
		// warp-list path as a harmless compatibility fallback.
		if err := ctx.Network.SendSelectWarpPoint(skill.ID, gameui.TeleportRandomMap); err != nil {
			return err
		}
	}
	return nil
}

func (c skillController) SendToGround(ctx client.Context, skill session.Skill, x, y int, source string) error {
	if playerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if !walkTargetInBounds(ctx, x, y) {
		return fmt.Errorf("invalid ground target %d,%d", x, y)
	}
	level := skillUseLevel(skill)
	glog.Debugf("%s ground skill use skill=%d level=%d target=%d,%d", source, skill.ID, level, x, y)
	if err := ctx.Network.SendUseSkillToGround(skill.ID, level, x, y); err != nil {
		return err
	}
	return nil
}

func (c skillController) SendToGroundWithText(ctx client.Context, skill session.Skill, x, y int, text string, source string) error {
	if playerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if !walkTargetInBounds(ctx, x, y) {
		return fmt.Errorf("invalid ground target %d,%d", x, y)
	}
	level := skillUseLevel(skill)
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
	if selectable, known := db.SkillLevelSelectable(pending.skill.ID); known && !selectable {
		return false
	}
	maxLevel := pending.maxLevel
	if knownMax := knownSkillMaxLevel(pending.skill); knownMax > 0 && (maxLevel <= 0 || knownMax < maxLevel) {
		maxLevel = knownMax
	}
	if maxLevel <= 0 {
		maxLevel = skillUseMaxLevel(pending.skill)
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
		if err := c.UseGround(ctx, skill, targetX, targetY, "", "target"); err != nil {
			glog.Warnf("skill ground target failed skill=%d target=%d,%d: %v", skill.ID, targetX, targetY, err)
			return
		}
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
	if err := c.UseTarget(ctx, skill, actor, "target"); err != nil {
		glog.Warnf("skill target failed skill=%d target=%d: %v", skill.ID, actor.ID, err)
	}
}

func (c skillController) UseTarget(ctx client.Context, skill session.Skill, actor worldstate.Actor, source string) error {
	if playerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	skill = normalizeSessionSkillLevelCap(skill)
	if skill.Type == 0 || skillForcesPassive(skill.ID) {
		return fmt.Errorf("passive skill")
	}
	if isGroundTargetSkill(skill) || isSelfTargetSkill(skill) {
		return fmt.Errorf("not an actor target skill")
	}
	if c.chaseTargetIfNeeded(ctx, skill, actor, source) {
		return nil
	}
	if err := c.SendToID(ctx, skill, actor.ID, source); err != nil {
		return err
	}
	c.mode.pendingSkill = pendingSkillTarget{}
	glog.Debugf("%s skill target sent skill=%d target=%d name=%q job=%d object_type=%d", source, skill.ID, actor.ID, actor.Name, actor.Job, actor.ObjectType)
	return nil
}

func skillNeedsGroundText(skillID uint16) bool {
	return skillID == db.SkillHTTalkiebox || skillID == db.SkillRGGraffiti
}

func (c skillController) UseGround(ctx client.Context, skill session.Skill, x, y int, text, source string) error {
	if playerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	skill = normalizeSessionSkillLevelCap(skill)
	if skill.Type == 0 || skillForcesPassive(skill.ID) || !isGroundTargetSkill(skill) {
		return fmt.Errorf("not a ground target skill")
	}
	if !walkTargetInBounds(ctx, x, y) {
		return fmt.Errorf("invalid ground target %d,%d", x, y)
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	pending := pendingSkillTarget{skill: skill, ground: true, x: x, y: y, text: text}
	if c.chaseSkillTargetIfNeeded(ctx, pending, x, y, source) {
		return nil
	}
	if err := c.sendTarget(ctx, pending, source); err != nil {
		return err
	}
	c.mode.pendingSkill = pendingSkillTarget{}
	return nil
}

func (pending pendingSkillTarget) hasTarget() bool {
	return pending.ground || pending.targetID != 0
}

func (pending pendingSkillTarget) targetCell(ctx client.Context, now time.Time) (int, int, bool) {
	if pending.ground {
		return pending.x, pending.y, walkTargetInBounds(ctx, pending.x, pending.y)
	}
	actor, ok := ctx.World.Actors[pending.targetID]
	if !ok {
		return 0, 0, false
	}
	x, y := actorCurrentCell(actor, now)
	return x, y, true
}

func (c skillController) sendTarget(ctx client.Context, pending pendingSkillTarget, source string) error {
	if pending.ground {
		if skillNeedsGroundText(pending.skill.ID) {
			return c.SendToGroundWithText(ctx, pending.skill, pending.x, pending.y, pending.text, source)
		}
		return c.SendToGround(ctx, pending.skill, pending.x, pending.y, source)
	}
	return c.SendToID(ctx, pending.skill, pending.targetID, source)
}

func (c skillController) chaseTargetIfNeeded(ctx client.Context, skill session.Skill, actor worldstate.Actor, source string) bool {
	x, y := actorCurrentCell(actor, time.Now())
	return c.chaseSkillTargetIfNeeded(ctx, pendingSkillTarget{skill: skill, targetID: actor.ID}, x, y, source)
}

func (c skillController) chaseSkillTargetIfNeeded(ctx client.Context, pending pendingSkillTarget, targetX, targetY int, source string) bool {
	skill := pending.skill
	if ctx.World == nil || skill.Range <= 0 {
		return false
	}
	caster, ok := skillCasterForSkill(ctx, skill)
	if !ok {
		glog.Warnf("%s skill caster missing skill=%d kind=%s target=%d", source, skill.ID, skillCasterKindName(skillCasterKindForID(skill.ID)), pending.targetID)
		c.Cancel("missing caster")
		return true
	}
	now := time.Now()
	casterX, casterY := skillCasterCell(caster, now)
	skillRange := skillTargetRangeForCaster(caster, skill)
	if caster.targetWithinRange(casterX, casterY, targetX, targetY, skillRange) {
		return false
	}
	approachCell := attackApproachCellFromTarget
	if caster.kind == skillCasterPlayer {
		approachCell = normalAttackApproachCellFromTarget
	}
	chaseX, chaseY, ok := approachCell(ctx, casterX, casterY, targetX, targetY, skillRange)
	if !ok {
		glog.Warnf("%s skill chase blocked skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", source, skill.ID, skillCasterKindName(caster.kind), caster.id, casterX, casterY, pending.targetID, targetX, targetY, skillRange)
		c.Cancel("unreachable target")
		c.mode.setWalkCooldown(walkRequestCooldown)
		return true
	}
	maxLevel := c.mode.pendingSkill.maxLevel
	if maxLevel <= 0 {
		maxLevel = skillUseMaxLevel(skill)
	}
	pending.maxLevel = maxLevel
	if pending.expires.IsZero() {
		pending.expires = now.Add(8 * time.Second)
	}
	pending.source = source
	pending.started = c.mode.pendingSkill.started
	if pending.started.IsZero() {
		pending.started = now
	}
	pending.readyAt = time.Time{}
	pending.lastChaseAt = now
	c.mode.pendingSkill = pending
	glog.Debugf("%s skill chase target skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d chase=%d,%d", source, skill.ID, skillCasterKindName(caster.kind), caster.id, casterX, casterY, pending.targetID, targetX, targetY, skillRange, chaseX, chaseY)
	c.requestSkillCasterMove(ctx, caster, chaseX, chaseY, source+" skill chase")
	return true
}

func (c skillController) requestSkillCasterMove(ctx client.Context, caster skillCaster, targetX, targetY int, source string) {
	if caster.kind == skillCasterPlayer {
		if ctx.Network == nil {
			glog.Warnf("%s player move failed target=%d,%d: not connected", source, targetX, targetY)
			return
		}
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
	if pending.skill.ID == 0 || !pending.hasTarget() || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(pending.expires) {
		glog.Debugf("%s pending skill expired skill=%d target=%d", source, pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	targetX, targetY, ok := pending.targetCell(ctx, now)
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
	if !caster.targetWithinRange(casterX, casterY, targetX, targetY, skillRange) {
		if logOutOfRange {
			glog.Debugf("%s pending skill still out of range skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", source, pending.skill.ID, skillCasterKindName(caster.kind), caster.id, casterX, casterY, pending.targetID, targetX, targetY, skillRange)
		}
		if movingActorDestinationWithinRange(caster.actor, targetX, targetY, skillRange, caster.targetWithinRange) {
			return
		}
		if attackRetryDue(pending.lastChaseAt, now) {
			c.mode.pendingSkill = pending
			c.chaseSkillTargetIfNeeded(ctx, pending, targetX, targetY, "pending")
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
	if pending.skill.ID == 0 || !pending.hasTarget() || pending.readyAt.IsZero() || ctx.World == nil {
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
	targetX, targetY, ok := pending.targetCell(ctx, now)
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
	if !caster.targetWithinRange(casterX, casterY, targetX, targetY, skillRange) {
		glog.Debugf("pending skill became out of range skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", pending.skill.ID, skillCasterKindName(caster.kind), caster.id, casterX, casterY, pending.targetID, targetX, targetY, skillRange)
		pending.readyAt = time.Time{}
		c.mode.pendingSkill = pending
		c.chaseSkillTargetIfNeeded(ctx, pending, targetX, targetY, "pending")
		return
	}
	c.mode.pendingSkill = pendingSkillTarget{}
	if err := c.sendTarget(ctx, pending, "pending"); err != nil {
		glog.Warnf("pending skill failed skill=%d target=%d cell=%d,%d: %v", pending.skill.ID, pending.targetID, targetX, targetY, err)
		return
	}
	glog.Debugf("pending skill sent skill=%d target=%d cell=%d,%d", pending.skill.ID, pending.targetID, targetX, targetY)
}

func (c skillController) ApplyAutoRun(ctx client.Context, auto network.AutoRunSkill) {
	skill := sessionSkillFromNetwork(auto.Skill)
	glog.Debugf("auto-run skill received skill=%d level=%d type=%d range=%d name=%q", skill.ID, skill.Level, skill.Type, skill.Range, skill.Name)
	if err := c.Use(ctx, skill, "auto"); err != nil {
		glog.Warnf("auto-run skill use failed skill=%d: %v", skill.ID, err)
		return
	}
}

func skillTargetOverrideActive(ctx client.Context) bool {
	return (ctx.Input != nil && ctx.Input.Pressed(input.KeyShift)) || (ctx.Session != nil && ctx.Session.NoShift)
}

func targetSkillRange(skill session.Skill) int {
	// robr approaches player skill targets within a circle of server range + 1.
	// This fits rAthena's truncated client distance without using square corners.
	return maxInt(1, skill.Range+1)
}

func targetSkillWithinRangeFrom(sourceX, sourceY, skillRange int, actor worldstate.Actor) bool {
	targetX, targetY := actorCurrentCell(actor, time.Now())
	return targetSkillWithinRangeCells(sourceX, sourceY, targetX, targetY, skillRange)
}

func targetSkillWithinRangeCells(sourceX, sourceY, targetX, targetY, skillRange int) bool {
	return normalAttackTargetWithinRange(sourceX, sourceY, targetX, targetY, skillRange)
}
