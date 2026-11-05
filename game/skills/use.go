package skills

import (
	"fmt"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type skillCasterKind uint8

const (
	skillCasterPlayer skillCasterKind = iota
	skillCasterHomunculus
	skillCasterMercenary
)

type skillCaster struct {
	Kind  skillCasterKind
	ID    uint32
	actor worldstate.Actor
}

func SkillByID(s *session.Session, skillID uint16) (session.Skill, bool) {
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

func SkillLabel(skill session.Skill) string {
	if skill.Name != "" {
		return skill.Name
	}
	return fmt.Sprintf("Skill %d", skill.ID)
}

func SkillUseMaxLevel(skill session.Skill) int {
	if maxLevel := scene.KnownSkillMaxLevel(skill); maxLevel > 0 {
		return maxLevel
	}
	return scene.MaxInt(1, skill.Level)
}

func skillUseLevel(skill session.Skill) uint16 {
	skill = scene.NormalizeSessionSkillLevelCap(skill)
	return uint16(scene.MaxInt(1, skill.Level))
}

func IsGroundTargetSkill(skill session.Skill) bool {
	return skill.Type&SkillTargetPlace != 0 || effects.SkillForcesGroundTarget(skill.ID)
}

func IsSelfTargetSkill(skill session.Skill) bool {
	return effects.SkillForcesSelfTarget(skill.ID) || (skill.Type&SkillTargetSelf != 0 && !IsGroundTargetSkill(skill))
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

func SkillCasterKindName(kind skillCasterKind) string {
	switch kind {
	case skillCasterHomunculus:
		return "homunculus"
	case skillCasterMercenary:
		return "mercenary"
	default:
		return "player"
	}
}

func SkillCasterForSkill(ctx client.Context, skill session.Skill) (skillCaster, bool) {
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
			id = companion.FindActorID(ctx, scene.ActorObjectTypeHomunculus)
		}
		actor, ok := companion.ActorByID(ctx, id)
		if !ok {
			return skillCaster{}, false
		}
		return skillCaster{Kind: kind, ID: id, actor: actor}, true
	case skillCasterMercenary:
		if ctx.Session == nil || !ctx.Session.Mercenary.Active {
			return skillCaster{}, false
		}
		id := ctx.Session.Mercenary.ID
		if id == 0 {
			id = companion.FindActorID(ctx, scene.ActorObjectTypeMercenary)
		}
		actor, ok := companion.ActorByID(ctx, id)
		if !ok {
			return skillCaster{}, false
		}
		return skillCaster{Kind: kind, ID: id, actor: actor}, true
	default:
		actor := ctx.World.Player
		if actor.ID == 0 {
			actor.ID = scene.LocalSkillTarget(ctx)
		}
		return skillCaster{Kind: kind, ID: actor.ID, actor: actor}, true
	}
}

func SkillCasterCell(caster skillCaster, now time.Time) (int, int) {
	if caster.Kind == skillCasterPlayer {
		x, y := scene.ActorRenderPosition(caster.actor, now)
		return int(math.Round(x)), int(math.Round(y))
	}
	return companion.ActorRenderCell(caster.actor, now)
}

func skillTargetRangeForCaster(caster skillCaster, skill session.Skill) int {
	if caster.Kind != skillCasterPlayer {
		if skill.Range > 0 {
			return skill.Range
		}
		level := scene.MaxInt(1, skill.Level)
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
	if caster.Kind == skillCasterPlayer {
		return targetSkillWithinRangeCells(sourceX, sourceY, targetX, targetY, skillRange)
	}
	// rAthena uses square range for non-player casters.
	return maps.AttackTargetWithinRange(sourceX, sourceY, targetX, targetY, skillRange)
}

const (
	SkillTargetEnemy  = 1
	SkillTargetPlace  = 2
	SkillTargetSelf   = 4
	SkillTargetFriend = 16
	SkillTargetTrap   = 32
	SkillTargetPet    = 64
	SkillTargetHomun  = 128
)

const SkillChangeCart = 154

const skillGroundTextMaxBytes = 79

func (c *System) Use(ctx client.Context, skill session.Skill, source string) error {
	if err := CheckStealthSkill(ctx, skill.ID, 0); err != nil {
		return err
	}
	if scene.PlayerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	skill = scene.NormalizeSessionSkillLevelCap(skill)
	if skill.Type == 0 || effects.SkillForcesPassive(skill.ID) {
		return fmt.Errorf("passive skill")
	}
	if skill.ID == SkillChangeCart {
		if c.bindings.ChangeCart == nil {
			return fmt.Errorf("missing world mode")
		}
		target := scene.LocalSkillTarget(ctx)
		if target == 0 {
			return fmt.Errorf("missing skill target")
		}
		if ctx.Network != nil {
			level := skillUseLevel(skill)
			if err := ctx.Network.SendUseSkillToID(skill.ID, level, target); err != nil {
				return err
			}
		}
		c.bindings.ChangeCart.Open(ctx)
		glog.Debugf("%s skill opens change cart selector skill=%d", source, skill.ID)
		return nil
	}
	if IsSelfTargetSkill(skill) {
		target := scene.LocalSkillTarget(ctx)
		if target == 0 {
			return fmt.Errorf("missing skill target")
		}
		return c.SendToID(ctx, skill, target, source)
	}
	if skill.Range > 0 || IsGroundTargetSkill(skill) {
		c.Pending = PendingTarget{Skill: skill, MaxLevel: SkillUseMaxLevel(skill), Started: time.Now()}
		glog.Debugf("%s skill target pending skill=%d level=%d range=%d", source, skill.ID, skill.Level, skill.Range)
		return nil
	}
	target := scene.LocalSkillTarget(ctx)
	if target == 0 {
		return fmt.Errorf("missing skill target")
	}
	return c.SendToID(ctx, skill, target, source)
}

// Companion orders do not replace the player's own movement or attack.
func (c *System) beginPlayerAction(ctx client.Context, skill session.Skill) {
	if IsSelfTargetSkill(skill) {
		switch skill.ID {
		case db.SkillMOChaincombo, db.SkillMOCombofinish, db.SkillChTigerfist, db.SkillChChaincrush,
			db.SkillMOExtremityfist, db.SkillTKTurnkick, db.SkillTKStormkick, db.SkillTKDownkick,
			db.SkillTKCounter, db.SkillTKJumpkick, db.SkillHTPower:
			// These skills use the server's current attack target. Canceling
			// the attack first would erase that target before the skill arrives.
			return
		}
	}
	if skillCasterKindForID(skill.ID) == skillCasterPlayer && c.bindings.BeginPlayerAction != nil {
		c.bindings.BeginPlayerAction(ctx)
	}
}

func (c *System) PlayerChasing() bool {
	return c.Pending.hasTarget() && skillCasterKindForID(c.Pending.Skill.ID) == skillCasterPlayer
}

func (c *System) CancelPlayerChase() {
	if c.PlayerChasing() {
		c.Pending = PendingTarget{}
	}
}

func (c *System) SendToID(ctx client.Context, skill session.Skill, target uint32, source string) error {
	if err := CheckStealthSkill(ctx, skill.ID, target); err != nil {
		return err
	}
	if scene.PlayerIsDead(ctx) {
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
	c.beginPlayerAction(ctx, skill)
	level := skillUseLevel(skill)
	glog.Debugf("%s skill use skill=%d level=%d target=%d", source, skill.ID, level, target)
	if err := ctx.Network.SendUseSkillToID(skill.ID, level, target); err != nil {
		return err
	}
	if c.bindings.RememberSenseTarget != nil {
		c.bindings.RememberSenseTarget(skill.ID, target, time.Now())
	}
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

func (c *System) SendToGround(ctx client.Context, skill session.Skill, x, y int, source string) error {
	if err := CheckStealthSkill(ctx, skill.ID, 0); err != nil {
		return err
	}
	if scene.PlayerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if !scene.WalkTargetInBounds(ctx, x, y) {
		return fmt.Errorf("invalid ground target %d,%d", x, y)
	}
	c.beginPlayerAction(ctx, skill)
	level := skillUseLevel(skill)
	glog.Debugf("%s ground skill use skill=%d level=%d target=%d,%d", source, skill.ID, level, x, y)
	if err := ctx.Network.SendUseSkillToGround(skill.ID, level, x, y); err != nil {
		return err
	}
	return nil
}

func (c *System) SendToGroundWithText(ctx client.Context, skill session.Skill, x, y int, text string, source string) error {
	if err := CheckStealthSkill(ctx, skill.ID, 0); err != nil {
		return err
	}
	if scene.PlayerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if !scene.WalkTargetInBounds(ctx, x, y) {
		return fmt.Errorf("invalid ground target %d,%d", x, y)
	}
	c.beginPlayerAction(ctx, skill)
	level := skillUseLevel(skill)
	glog.Debugf("%s ground skill with text use skill=%d level=%d target=%d,%d text_len=%d", source, skill.ID, level, x, y, len([]byte(text)))
	if err := ctx.Network.SendUseSkillToGroundWithText(skill.ID, level, x, y, text); err != nil {
		return err
	}
	return nil
}

func (c *System) CancelFromInput(ctx client.Context) bool {
	if c.Pending.Skill.ID == 0 || ctx.Input == nil {
		return false
	}
	if !ctx.Input.JustPressed(input.KeyEscape) && !ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		return false
	}
	c.Cancel("input")
	return true
}

func (c *System) Cancel(source string) {
	if c.Pending.Skill.ID == 0 {
		return
	}
	glog.Debugf("skill target canceled skill=%d source=%s", c.Pending.Skill.ID, source)
	c.Pending = PendingTarget{}
}

func (c *System) AdjustPendingLevelFromWheel(ctx client.Context) bool {
	if ctx.Input == nil || ctx.Input.WheelY == 0 || c.Pending.Skill.ID == 0 {
		return false
	}
	pending := c.Pending
	if selectable, known := db.SkillLevelSelectable(pending.Skill.ID); known && !selectable {
		return false
	}
	maxLevel := pending.MaxLevel
	if knownMax := scene.KnownSkillMaxLevel(pending.Skill); knownMax > 0 && (maxLevel <= 0 || knownMax < maxLevel) {
		maxLevel = knownMax
	}
	if maxLevel <= 0 {
		maxLevel = SkillUseMaxLevel(pending.Skill)
	}
	step := int(math.Ceil(math.Abs(ctx.Input.WheelY)))
	if step < 1 {
		step = 1
	}
	level := pending.Skill.Level
	if ctx.Input.WheelY > 0 {
		level += step
	} else {
		level -= step
	}
	level = scene.ClampInt(level, 1, maxLevel)
	ctx.Input.WheelY = 0
	if level == pending.Skill.Level {
		return true
	}
	pending.Skill.Level = level
	c.Pending = pending
	glog.Debugf("skill target level changed skill=%d level=%d max=%d", pending.Skill.ID, level, maxLevel)
	return true
}

func (c *System) HandleClick(ctx client.Context, projection scene.Projection, now time.Time) {
	skill := c.Pending.Skill
	if skill.ID == 0 {
		return
	}
	if IsGroundTargetSkill(skill) {
		targetX, targetY, ok := maps.ClickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY)
		if !ok {
			glog.Debugf("skill ground target miss skill=%d mouse=%d,%d", skill.ID, ctx.Input.MouseX, ctx.Input.MouseY)
			return
		}
		if skillNeedsGroundText(skill.ID) {
			c.OpenSkillTextPrompt(ctx, skill, targetX, targetY, "target")
			c.Pending = PendingTarget{}
			glog.Debugf("skill ground text prompt opened skill=%d target=%d,%d", skill.ID, targetX, targetY)
			return
		}
		if err := c.UseGround(ctx, skill, targetX, targetY, "", "target"); err != nil {
			glog.Warnf("skill ground target failed skill=%d target=%d,%d: %v", skill.ID, targetX, targetY, err)
			return
		}
		return
	}
	actor, ok := c.ClickedSkillTarget(ctx, projection, skill, ctx.Input.MouseX, ctx.Input.MouseY, now)
	if !ok {
		if x, y, groundOK := maps.ClickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); groundOK {
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

func (c *System) UseTarget(ctx client.Context, skill session.Skill, actor worldstate.Actor, source string) error {
	if err := CheckStealthSkill(ctx, skill.ID, actor.ID); err != nil {
		return err
	}
	if scene.PlayerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	skill = scene.NormalizeSessionSkillLevelCap(skill)
	if skill.Type == 0 || effects.SkillForcesPassive(skill.ID) {
		return fmt.Errorf("passive skill")
	}
	if IsGroundTargetSkill(skill) || IsSelfTargetSkill(skill) {
		return fmt.Errorf("not an actor target skill")
	}
	c.beginPlayerAction(ctx, skill)
	if c.chaseTargetIfNeeded(ctx, skill, actor, source) {
		return nil
	}
	if err := c.SendToID(ctx, skill, actor.ID, source); err != nil {
		return err
	}
	c.Pending = PendingTarget{}
	glog.Debugf("%s skill target sent skill=%d target=%d name=%q job=%d object_type=%d", source, skill.ID, actor.ID, actor.Name, actor.Job, actor.ObjectType)
	return nil
}

func skillNeedsGroundText(skillID uint16) bool {
	return skillID == db.SkillHTTalkiebox || skillID == db.SkillRGGraffiti
}

func (c *System) UseGround(ctx client.Context, skill session.Skill, x, y int, text, source string) error {
	if err := CheckStealthSkill(ctx, skill.ID, 0); err != nil {
		return err
	}
	if scene.PlayerIsDead(ctx) {
		return fmt.Errorf("player is dead")
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	skill = scene.NormalizeSessionSkillLevelCap(skill)
	if skill.Type == 0 || effects.SkillForcesPassive(skill.ID) || !IsGroundTargetSkill(skill) {
		return fmt.Errorf("not a ground target skill")
	}
	if !scene.WalkTargetInBounds(ctx, x, y) {
		return fmt.Errorf("invalid ground target %d,%d", x, y)
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	c.beginPlayerAction(ctx, skill)
	pending := PendingTarget{Skill: skill, Ground: true, X: x, Y: y, Text: text}
	if c.chaseSkillTargetIfNeeded(ctx, pending, x, y, source) {
		return nil
	}
	if err := c.sendTarget(ctx, pending, source); err != nil {
		return err
	}
	c.Pending = PendingTarget{}
	return nil
}

func (pending PendingTarget) hasTarget() bool {
	return pending.Ground || pending.TargetID != 0
}

func (c *System) targetCell(ctx client.Context, pending PendingTarget, now time.Time) (int, int, bool) {
	if !LocalStealthAllowsSkill(ctx, pending.Skill.ID) {
		return 0, 0, false
	}
	if pending.Ground {
		return pending.X, pending.Y, scene.WalkTargetInBounds(ctx, pending.X, pending.Y)
	}
	if pending.Skill.Type&SkillTargetTrap != 0 {
		unit, ok := c.Units[pending.TargetID]
		return unit.X, unit.Y, ok
	}
	actor, ok, _ := scene.ActorForCombatID(ctx, pending.TargetID)
	if !ok || scene.ActorHasStealth(actor) {
		return 0, 0, false
	}
	x, y := scene.ActorCurrentCell(actor, now)
	return x, y, true
}

func (c *System) sendTarget(ctx client.Context, pending PendingTarget, source string) error {
	if pending.Ground {
		if skillNeedsGroundText(pending.Skill.ID) {
			return c.SendToGroundWithText(ctx, pending.Skill, pending.X, pending.Y, pending.Text, source)
		}
		return c.SendToGround(ctx, pending.Skill, pending.X, pending.Y, source)
	}
	return c.SendToID(ctx, pending.Skill, pending.TargetID, source)
}

func (c *System) chaseTargetIfNeeded(ctx client.Context, skill session.Skill, actor worldstate.Actor, source string) bool {
	x, y := scene.ActorCurrentCell(actor, time.Now())
	return c.chaseSkillTargetIfNeeded(ctx, PendingTarget{Skill: skill, TargetID: actor.ID}, x, y, source)
}

func (c *System) chaseSkillTargetIfNeeded(ctx client.Context, pending PendingTarget, targetX, targetY int, source string) bool {
	skill := pending.Skill
	if ctx.World == nil || skill.Range <= 0 {
		return false
	}
	caster, ok := SkillCasterForSkill(ctx, skill)
	if !ok {
		glog.Warnf("%s skill caster missing skill=%d kind=%s target=%d", source, skill.ID, SkillCasterKindName(skillCasterKindForID(skill.ID)), pending.TargetID)
		c.Cancel("missing caster")
		return true
	}
	now := time.Now()
	casterX, casterY := SkillCasterCell(caster, now)
	skillRange := skillTargetRangeForCaster(caster, skill)
	inRange := caster.targetWithinRange(casterX, casterY, targetX, targetY, skillRange)
	if inRange && (caster.Kind != skillCasterPlayer || c.bindings.PlayerCanUseSkill == nil || c.bindings.PlayerCanUseSkill(ctx, now)) {
		return false
	}
	approachCell := maps.AttackApproachCellFromTarget
	if caster.Kind == skillCasterPlayer {
		approachCell = maps.NormalAttackApproachCellFromTarget
	}
	chaseX, chaseY, ok := casterX, casterY, true
	if !inRange {
		chaseX, chaseY, ok = approachCell(ctx, casterX, casterY, targetX, targetY, skillRange)
	}
	if !ok {
		glog.Warnf("%s skill chase blocked skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", source, skill.ID, SkillCasterKindName(caster.Kind), caster.ID, casterX, casterY, pending.TargetID, targetX, targetY, skillRange)
		c.Cancel("unreachable target")
		if c.bindings.WalkBlocked != nil {
			c.bindings.WalkBlocked()
		}
		return true
	}
	maxLevel := c.Pending.MaxLevel
	if maxLevel <= 0 {
		maxLevel = SkillUseMaxLevel(skill)
	}
	pending.MaxLevel = maxLevel
	if pending.Expires.IsZero() {
		pending.Expires = now.Add(8 * time.Second)
	}
	pending.Source = source
	pending.Started = c.Pending.Started
	if pending.Started.IsZero() {
		pending.Started = now
	}
	pending.ReadyAt = time.Time{}
	pending.LastChaseAt = now
	c.Pending = pending
	glog.Debugf("%s skill chase target skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d chase=%d,%d", source, skill.ID, SkillCasterKindName(caster.Kind), caster.ID, casterX, casterY, pending.TargetID, targetX, targetY, skillRange, chaseX, chaseY)
	if !inRange {
		c.requestSkillCasterMove(ctx, caster, chaseX, chaseY, source+" skill chase")
	}
	return true
}

func (c *System) requestSkillCasterMove(ctx client.Context, caster skillCaster, targetX, targetY int, source string) {
	if caster.Kind == skillCasterPlayer {
		if ctx.Network == nil {
			glog.Warnf("%s player move failed target=%d,%d: not connected", source, targetX, targetY)
			return
		}
		if c.bindings.RequestWalk != nil {
			c.bindings.RequestWalk(ctx, targetX, targetY, source)
		}
		return
	}
	if ctx.Network == nil {
		glog.Warnf("%s companion move failed caster=%s caster_id=%d target=%d,%d: not connected", source, SkillCasterKindName(caster.Kind), caster.ID, targetX, targetY)
		return
	}
	if !scene.WalkTargetInBounds(ctx, targetX, targetY) {
		glog.Warnf("%s companion move blocked caster=%s caster_id=%d target=%d,%d", source, SkillCasterKindName(caster.Kind), caster.ID, targetX, targetY)
		return
	}
	if err := ctx.Network.SendCompanionMove(caster.ID, targetX, targetY); err != nil {
		glog.Warnf("%s companion move failed caster=%s caster_id=%d target=%d,%d: %v", source, SkillCasterKindName(caster.Kind), caster.ID, targetX, targetY, err)
	}
}

func (c *System) ContinuePendingTarget(ctx client.Context, source string) {
	c.UpdatePendingTarget(ctx, source, true)
}

func (c *System) UpdatePendingTarget(ctx client.Context, source string, logOutOfRange bool) {
	pending := c.Pending
	if pending.Skill.ID == 0 || !pending.hasTarget() || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(pending.Expires) {
		glog.Debugf("%s pending skill expired skill=%d target=%d", source, pending.Skill.ID, pending.TargetID)
		c.Pending = PendingTarget{}
		return
	}
	targetX, targetY, ok := c.targetCell(ctx, pending, now)
	if !ok {
		glog.Debugf("%s pending skill target vanished skill=%d target=%d", source, pending.Skill.ID, pending.TargetID)
		c.Pending = PendingTarget{}
		return
	}
	caster, ok := SkillCasterForSkill(ctx, pending.Skill)
	if !ok {
		glog.Debugf("%s pending skill caster vanished skill=%d kind=%s target=%d", source, pending.Skill.ID, SkillCasterKindName(skillCasterKindForID(pending.Skill.ID)), pending.TargetID)
		c.Pending = PendingTarget{}
		return
	}
	casterX, casterY := SkillCasterCell(caster, now)
	skillRange := skillTargetRangeForCaster(caster, pending.Skill)
	if !caster.targetWithinRange(casterX, casterY, targetX, targetY, skillRange) {
		if logOutOfRange {
			glog.Debugf("%s pending skill still out of range skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", source, pending.Skill.ID, SkillCasterKindName(caster.Kind), caster.ID, casterX, casterY, pending.TargetID, targetX, targetY, skillRange)
		}
		if scene.MovingActorDestinationWithinRange(caster.actor, targetX, targetY, skillRange, caster.targetWithinRange) {
			return
		}
		if pending.LastChaseAt.IsZero() || now.Sub(pending.LastChaseAt) >= scene.ChaseRetryInterval {
			c.Pending = pending
			c.chaseSkillTargetIfNeeded(ctx, pending, targetX, targetY, "pending")
		}
		return
	}
	readyAt := now
	if pending.ReadyAt.IsZero() {
		pending.ReadyAt = readyAt
	}
	c.Pending = pending
	if logOutOfRange {
		glog.Debugf("%s pending skill scheduled skill=%d caster=%s caster_id=%d target=%d delay_ms=%d", source, pending.Skill.ID, SkillCasterKindName(caster.Kind), caster.ID, pending.TargetID, scene.MaxInt(0, int(pending.ReadyAt.Sub(now).Milliseconds())))
	}
}

func (c *System) ProcessPendingTarget(ctx client.Context) {
	pending := c.Pending
	if pending.Skill.ID == 0 || !pending.hasTarget() || pending.ReadyAt.IsZero() || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(pending.Expires) {
		glog.Debugf("pending skill expired skill=%d target=%d", pending.Skill.ID, pending.TargetID)
		c.Pending = PendingTarget{}
		return
	}
	if now.Before(pending.ReadyAt) {
		return
	}
	targetX, targetY, ok := c.targetCell(ctx, pending, now)
	if !ok {
		glog.Debugf("pending skill target vanished skill=%d target=%d", pending.Skill.ID, pending.TargetID)
		c.Pending = PendingTarget{}
		return
	}
	caster, ok := SkillCasterForSkill(ctx, pending.Skill)
	if !ok {
		glog.Debugf("pending skill caster vanished skill=%d kind=%s target=%d", pending.Skill.ID, SkillCasterKindName(skillCasterKindForID(pending.Skill.ID)), pending.TargetID)
		c.Pending = PendingTarget{}
		return
	}
	if caster.Kind == skillCasterPlayer && c.bindings.PlayerCanUseSkill != nil && !c.bindings.PlayerCanUseSkill(ctx, now) {
		return
	}
	casterX, casterY := SkillCasterCell(caster, now)
	skillRange := skillTargetRangeForCaster(caster, pending.Skill)
	if !caster.targetWithinRange(casterX, casterY, targetX, targetY, skillRange) {
		glog.Debugf("pending skill became out of range skill=%d caster=%s caster_id=%d caster=%d,%d target=%d target_cell=%d,%d range=%d", pending.Skill.ID, SkillCasterKindName(caster.Kind), caster.ID, casterX, casterY, pending.TargetID, targetX, targetY, skillRange)
		pending.ReadyAt = time.Time{}
		c.Pending = pending
		c.chaseSkillTargetIfNeeded(ctx, pending, targetX, targetY, "pending")
		return
	}
	c.Pending = PendingTarget{}
	if err := c.sendTarget(ctx, pending, "pending"); err != nil {
		glog.Warnf("pending skill failed skill=%d target=%d cell=%d,%d: %v", pending.Skill.ID, pending.TargetID, targetX, targetY, err)
		return
	}
	glog.Debugf("pending skill sent skill=%d target=%d cell=%d,%d", pending.Skill.ID, pending.TargetID, targetX, targetY)
}

func (c *System) ApplyAutoRun(ctx client.Context, auto network.AutoRunSkill) {
	skill := scene.SessionSkillFromNetwork(auto.Skill)
	glog.Debugf("auto-run skill received skill=%d level=%d type=%d range=%d name=%q", skill.ID, skill.Level, skill.Type, skill.Range, skill.Name)
	if err := c.Use(ctx, skill, "auto"); err != nil {
		glog.Warnf("auto-run skill use failed skill=%d: %v", skill.ID, err)
		return
	}
}

func SkillTargetOverrideActive(ctx client.Context) bool {
	return (ctx.Input != nil && ctx.Input.Pressed(input.KeyShift)) || (ctx.Session != nil && ctx.Session.NoShift)
}

func targetSkillRange(skill session.Skill) int {
	return scene.MaxInt(1, skill.Range)
}

func targetSkillWithinRangeCells(sourceX, sourceY, targetX, targetY, skillRange int) bool {
	return maps.NormalAttackTargetWithinRange(sourceX, sourceY, targetX, targetY, skillRange)
}
