package game

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type skillController struct {
	mode *WorldMode
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

func skillDisplayName(manager *res.Manager, skill session.Skill) string {
	if manager != nil {
		if name, ok := manager.SkillDisplayName(int(skill.ID)); ok {
			return name
		}
	}
	return skillLabel(skill)
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

const (
	skillTargetEnemy  = 1
	skillTargetPlace  = 2
	skillTargetSelf   = 4
	skillTargetFriend = 16
	skillTargetTrap   = 32
	skillTargetPet    = 64
	skillTargetHomun  = 128
)

func (c skillController) Use(ctx client.Context, skill session.Skill, source string) error {
	if skill.ID == 0 || skill.Level <= 0 {
		return fmt.Errorf("skill is not learned")
	}
	if skill.Type == 0 || skillForcesPassive(skill.ID) {
		return fmt.Errorf("passive skill")
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
		c.mode.status = fmt.Sprintf("select target: %s", skillDisplayName(ctx.Resources, skill))
		log.Printf("%s skill target pending skill=%d level=%d range=%d", source, skill.ID, skill.Level, skill.Range)
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
	log.Printf("%s skill use skill=%d level=%d target=%d", source, skill.ID, level, target)
	if err := ctx.Network.SendUseSkillToID(skill.ID, level, target); err != nil {
		return err
	}
	if property, duration := skillCastFallback(skill.ID, level); duration > 0 {
		c.mode.addLocalSkillCastFallback(ctx, skill.ID, property, localSkillTarget(ctx), target, 0, 0, duration, time.Now(), source)
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
	log.Printf("%s ground skill use skill=%d level=%d target=%d,%d", source, skill.ID, level, x, y)
	if err := ctx.Network.SendUseSkillToGround(skill.ID, level, x, y); err != nil {
		return err
	}
	property, castDuration := skillCastFallback(skill.ID, level)
	if castDuration > 0 {
		c.mode.addLocalSkillCastFallback(ctx, skill.ID, property, localSkillTarget(ctx), 0, x, y, castDuration, time.Now(), source+"-ground")
	}
	if castDuration <= 0 {
		now := time.Now()
		for _, effectID := range skillGroundEffectIDs(skill.ID) {
			c.mode.addWorldEffectAtCellIfMissing(ctx, effectID, x, y, now)
		}
	}
	return nil
}

func (c skillController) CancelFromInput(ctx client.Context) bool {
	if c.mode.pendingSkill.skill.ID == 0 || ctx.Input == nil {
		return false
	}
	if !ctx.Input.JustPressed(render.KeyEscape) && !ctx.Input.MouseJustPressed(render.MouseButtonRight) {
		return false
	}
	c.Cancel("input")
	return true
}

func (c skillController) Cancel(source string) {
	if c.mode.pendingSkill.skill.ID == 0 {
		return
	}
	log.Printf("skill target canceled skill=%d source=%s", c.mode.pendingSkill.skill.ID, source)
	c.mode.pendingSkill = pendingSkillTarget{}
	c.mode.status = "skill canceled"
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
	c.mode.status = fmt.Sprintf("select target: %s Lv%d", skillDisplayName(ctx.Resources, pending.skill), level)
	log.Printf("skill target level changed skill=%d level=%d max=%d", pending.skill.ID, level, maxLevel)
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
			c.mode.status = fmt.Sprintf("select target: %s", skillDisplayName(ctx.Resources, skill))
			log.Printf("skill ground target miss skill=%d mouse=%d,%d", skill.ID, ctx.Input.MouseX, ctx.Input.MouseY)
			return
		}
		if err := c.SendToGround(ctx, skill, targetX, targetY, "target"); err != nil {
			c.mode.status = "skill failed: " + err.Error()
			log.Printf("skill ground target failed skill=%d target=%d,%d: %v", skill.ID, targetX, targetY, err)
			return
		}
		c.mode.pendingSkill = pendingSkillTarget{}
		c.mode.status = fmt.Sprintf("%s: %d,%d", skillDisplayName(ctx.Resources, skill), targetX, targetY)
		log.Printf("skill ground target sent skill=%d target=%d,%d", skill.ID, targetX, targetY)
		return
	}
	actor, ok := clickedSkillTarget(ctx, projection, skill, ctx.Input.MouseX, ctx.Input.MouseY, now, c.mode.actorDeaths)
	if !ok {
		if x, y, groundOK := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); groundOK {
			log.Printf("skill target canceled by ground click skill=%d mouse=%d,%d target=%d,%d", skill.ID, ctx.Input.MouseX, ctx.Input.MouseY, x, y)
			c.Cancel("ground-click")
			return
		}
		c.mode.status = fmt.Sprintf("select target: %s", skillDisplayName(ctx.Resources, skill))
		log.Printf("skill target miss skill=%d mouse=%d,%d", skill.ID, ctx.Input.MouseX, ctx.Input.MouseY)
		return
	}
	if c.chaseTargetIfNeeded(ctx, skill, actor, "target") {
		return
	}
	if err := c.SendToID(ctx, skill, actor.ID, "target"); err != nil {
		c.mode.status = "skill failed: " + err.Error()
		log.Printf("skill target failed skill=%d target=%d: %v", skill.ID, actor.ID, err)
		return
	}
	c.mode.pendingSkill = pendingSkillTarget{}
	c.mode.status = fmt.Sprintf("%s: %d", skillDisplayName(ctx.Resources, skill), actor.ID)
	log.Printf("skill target sent skill=%d target=%d name=%q job=%d object_type=%d", skill.ID, actor.ID, actor.Name, actor.Job, actor.ObjectType)
}

func (c skillController) chaseTargetIfNeeded(ctx client.Context, skill session.Skill, actor worldstate.Actor, source string) bool {
	if ctx.World == nil || skill.Range <= 0 || targetSkillWithinRange(ctx, skill, actor) {
		return false
	}
	targetX, targetY, ok := attackApproachCell(ctx, actor, targetSkillRange(skill))
	if !ok {
		c.mode.status = fmt.Sprintf("%s chase blocked: %d", skillDisplayName(ctx.Resources, skill), actor.ID)
		log.Printf("%s skill chase blocked skill=%d target=%d player=%d,%d target=%d,%d range=%d", source, skill.ID, actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, targetSkillRange(skill))
		c.mode.walkCooldown = 12
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
	log.Printf("%s skill chase target skill=%d target=%d player=%d,%d target=%d,%d range=%d chase=%d,%d", source, skill.ID, actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, targetSkillRange(skill), targetX, targetY)
	c.mode.requestWalk(ctx, targetX, targetY, source+" skill chase")
	return true
}

func (c skillController) ContinuePendingTarget(ctx client.Context, source string) {
	pending := c.mode.pendingSkill
	if pending.skill.ID == 0 || pending.targetID == 0 || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(pending.expires) {
		log.Printf("%s pending skill expired skill=%d target=%d", source, pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	actor, ok := ctx.World.Actors[pending.targetID]
	if !ok {
		log.Printf("%s pending skill target vanished skill=%d target=%d", source, pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	if !targetSkillWithinRange(ctx, pending.skill, actor) {
		log.Printf("%s pending skill still out of range skill=%d target=%d player=%d,%d target=%d,%d range=%d", source, pending.skill.ID, actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, targetSkillRange(pending.skill))
		return
	}
	readyAt := pendingAttackReadyAt(ctx.World.Player, now)
	if pending.readyAt.IsZero() || readyAt.After(pending.readyAt) {
		pending.readyAt = readyAt
	}
	c.mode.pendingSkill = pending
	log.Printf("%s pending skill scheduled skill=%d target=%d delay_ms=%d", source, pending.skill.ID, pending.targetID, maxInt(0, int(pending.readyAt.Sub(now).Milliseconds())))
}

func (c skillController) ProcessPendingTarget(ctx client.Context) {
	pending := c.mode.pendingSkill
	if pending.skill.ID == 0 || pending.targetID == 0 || pending.readyAt.IsZero() || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(pending.expires) {
		log.Printf("pending skill expired skill=%d target=%d", pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	if now.Before(pending.readyAt) {
		return
	}
	actor, ok := ctx.World.Actors[pending.targetID]
	if !ok {
		log.Printf("pending skill target vanished skill=%d target=%d", pending.skill.ID, pending.targetID)
		c.mode.pendingSkill = pendingSkillTarget{}
		return
	}
	if !targetSkillWithinRange(ctx, pending.skill, actor) {
		log.Printf("pending skill became out of range skill=%d target=%d player=%d,%d target=%d,%d range=%d", pending.skill.ID, actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, targetSkillRange(pending.skill))
		pending.readyAt = time.Time{}
		c.mode.pendingSkill = pending
		c.chaseTargetIfNeeded(ctx, pending.skill, actor, "pending")
		return
	}
	c.mode.pendingSkill = pendingSkillTarget{}
	if err := c.SendToID(ctx, pending.skill, actor.ID, "pending"); err != nil {
		c.mode.status = "skill failed: " + err.Error()
		log.Printf("pending skill failed skill=%d target=%d: %v", pending.skill.ID, actor.ID, err)
		return
	}
	c.mode.status = fmt.Sprintf("%s: %d", skillDisplayName(ctx.Resources, pending.skill), actor.ID)
	log.Printf("pending skill sent skill=%d target=%d name=%q job=%d object_type=%d", pending.skill.ID, actor.ID, actor.Name, actor.Job, actor.ObjectType)
}

func (c skillController) ApplyAutoRun(ctx client.Context, auto network.AutoRunSkill) {
	skill := sessionSkillFromNetwork(auto.Skill)
	target := localSkillTarget(ctx)
	log.Printf("auto-run skill received skill=%d level=%d range=%d name=%q target=%d", skill.ID, skill.Level, skill.Range, skill.Name, target)
	if target == 0 {
		c.mode.status = "auto skill failed: missing player id"
		return
	}
	if err := c.SendToID(ctx, skill, target, "auto"); err != nil {
		c.mode.status = "auto skill failed: " + err.Error()
		log.Printf("auto-run skill use failed skill=%d target=%d: %v", skill.ID, target, err)
		return
	}
	c.mode.status = skillDisplayName(ctx.Resources, skill)
}

func skillTargetOverrideActive(ctx client.Context) bool {
	return (ctx.Input != nil && ctx.Input.Pressed(render.KeyShift)) || (ctx.Session != nil && ctx.Session.NoShift)
}

func targetSkillRange(skill session.Skill) int {
	return maxInt(1, skill.Range)
}

func targetSkillWithinRange(ctx client.Context, skill session.Skill, actor worldstate.Actor) bool {
	return ctx.World != nil && attackTargetWithinRange(ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, targetSkillRange(skill))
}
