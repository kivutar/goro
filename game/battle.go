package game

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/world"
)

type attackIntent struct {
	targetID    uint32
	expires     time.Time
	readyAt     time.Time
	lastChaseAt time.Time
}

type damageFloater struct {
	actorID  uint32
	x        int
	y        int
	text     string
	color    color.RGBA
	kind     damageFloaterKind
	starts   time.Time
	expires  time.Time
	duration time.Duration
}

type damageFloaterKind int

const (
	damageFloaterNormal damageFloaterKind = iota
	damageFloaterCritical
	damageFloaterIncoming
	damageFloaterRecoveryHP
	damageFloaterRecoverySP
	damageFloaterMiss
	damageFloaterCombo
)

func actionAnimationDuration(action res.ACTAction, fallback time.Duration) time.Duration {
	if len(action.Animations) == 0 {
		return fallback
	}
	delayMS := float64(action.DelayMS)
	if delayMS <= 0 {
		delayMS = 150
	}
	duration := time.Duration(delayMS * float64(time.Millisecond) * float64(len(action.Animations)))
	if duration <= 0 {
		return fallback
	}
	if duration > maxCombatAnimationDuration {
		return maxCombatAnimationDuration
	}
	return duration
}

func combatHitDelayFromAction(action res.ACTAction, duration time.Duration) time.Duration {
	return combatMotionDelayFromAction(action, duration, firstActionSoundMotion(action))
}

func combatMotionDelayFromAction(action res.ACTAction, duration time.Duration, motion int) time.Duration {
	if duration <= 0 {
		return 0
	}
	if motion >= 0 && len(action.Animations) > 0 {
		return duration * time.Duration(motion) / time.Duration(len(action.Animations))
	}
	return duration / 2
}

func firstActionSoundMotion(action res.ACTAction) int {
	for index, animation := range action.Animations {
		if animation.Sound >= 0 {
			return index
		}
	}
	return -1
}

func (m *WorldMode) humanoidSpriteViewForActor(ctx client.Context, actor world.Actor) *humanoidSpriteView {
	if isLocalActor(ctx, actor.ID) {
		return m.playerView
	}
	if actorIsMercenary(actor) {
		return m.mercenaryHumanoidSpriteView(ctx, actor)
	}
	actor = actorWithVisualJob(actor)
	weapon, shield := res.NormalizePlayerWeaponShield(int(actor.Weapon), int(actor.Shield))
	key := actorSpriteKey{
		job:         int(actor.Job),
		head:        int(actor.Head),
		sex:         actor.Sex,
		admin:       actor.IsAdmin,
		bodyPalette: int(actor.BodyPal),
		headPalette: int(actor.HeadPal),
		weapon:      weapon,
		shield:      shield,
		headTop:     int(actor.HeadTop),
		headMid:     int(actor.HeadMid),
		headLow:     int(actor.HeadLow),
	}
	return m.actorViews[key]
}

func (m *WorldMode) nonPCResolvedAction(ctx client.Context, actor world.Actor, actionFamily int) (res.ACTAction, bool) {
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return res.ACTAction{}, false
	}
	_, action, ok := resolveSpriteAction(view.act, actionFamily, actor.Dir)
	return action, ok
}

func (m *WorldMode) actorResolvedAction(ctx client.Context, actor world.Actor, actionFamily int) (res.ACTAction, bool) {
	if res.HasPlayerJobToken(actorVisualJob(actor)) || actorIsMercenary(actor) {
		view := m.humanoidSpriteViewForActor(ctx, actor)
		if view == nil || view.body == nil {
			return res.ACTAction{}, false
		}
		_, action, ok := resolveSpriteAction(view.body.act, actionFamily, actor.Dir)
		return action, ok
	}
	return m.nonPCResolvedAction(ctx, actor, actionFamily)
}

func (m *WorldMode) actorActionFrameDelay(ctx client.Context, actor world.Actor, actionFamily int, duration time.Duration) time.Duration {
	if duration <= 0 {
		return 0
	}
	action, ok := m.actorResolvedAction(ctx, actor, actionFamily)
	if !ok || len(action.Animations) == 0 {
		return 0
	}
	return duration / time.Duration(len(action.Animations))
}

func (m *WorldMode) nonPCActionACT(ctx client.Context, actor world.Actor) *res.ACT {
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return nil
	}
	return view.act
}

func (m *WorldMode) actorActionACT(ctx client.Context, actor world.Actor) *res.ACT {
	if res.HasPlayerJobToken(actorVisualJob(actor)) || actorIsMercenary(actor) {
		view := m.humanoidSpriteViewForActor(ctx, actor)
		if view == nil || view.body == nil {
			return nil
		}
		return view.body.act
	}
	return m.nonPCActionACT(ctx, actor)
}

func (m *WorldMode) actorActionDuration(ctx client.Context, actor world.Actor, actionFamily int, fallback time.Duration) time.Duration {
	if !res.HasPlayerJobToken(int(actor.Job)) && !actorIsMercenary(actor) && m.nonPCActorHasGR2Model(ctx, actor) {
		if action, ok := gr2ActionForActionFamily(actionFamily); ok {
			if view := m.nonPCGR2ModelView(ctx, actor); view != nil {
				if duration := view.actionDuration(action); duration > 0 {
					return duration
				}
			}
		}
	}
	if action, ok := m.actorResolvedAction(ctx, actor, actionFamily); ok {
		return actionAnimationDuration(action, fallback)
	}
	return fallback
}

type actorAnimation struct {
	actionFamily   int
	started        time.Time
	duration       time.Duration
	startDelay     time.Duration
	loop           bool
	play           bool
	hasPlay        bool
	length         int
	hasLength      bool
	frameOffset    int
	hasFrameOffset bool
	holdFinal      bool
	fixedMotion    int
	hasFixedMotion bool
	speed          time.Duration
	hasSpeed       bool
	next           *actorAnimation
}

type actorLife struct {
	hp        int
	maxHP     int
	sp        int
	maxSP     int
	hasSP     bool
	player    bool
	friendly  bool
	hunger    int
	maxHunger int
	hasHunger bool
	updatedAt time.Time
}

type actorCastBar struct {
	started  time.Time
	duration time.Duration
	color    color.RGBA
}

const attackRetryInterval = 1200 * time.Millisecond

const (
	defaultAttackAnimationDuration  = 600 * time.Millisecond
	defaultHitAnimationDuration     = 250 * time.Millisecond
	defaultDeathAnimationDuration   = 900 * time.Millisecond
	maxCombatAnimationDuration      = 5 * time.Second
	nonPCDeathFadeDuration          = 5 * time.Second
	multiHitDelay                   = 200 * time.Millisecond
	referenceBowArrowFlightDuration = 8 * 24 * time.Millisecond
)

func combatDuration(speed int32, fallback time.Duration) time.Duration {
	if speed <= 0 {
		return fallback
	}
	duration := time.Duration(speed) * time.Millisecond
	if duration > maxCombatAnimationDuration {
		return maxCombatAnimationDuration
	}
	return duration
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (m *WorldMode) requestAttack(ctx client.Context, actor world.Actor, source string) {
	if playerIsDead(ctx) {
		return
	}
	if ctx.Network == nil {
		m.setWalkCooldown(walkErrorCooldown)
		return
	}
	m.focusAttackTarget(actor.ID, time.Now())
	if normalAttackLockActive(ctx) {
		m.lockAttack(actor.ID)
	} else {
		m.clearLockedAttack()
	}
	now := time.Now()
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := currentPlayerCell(ctx, now)
	targetX, targetY := actorCurrentCell(actor, now)
	if normalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange) {
		m.sendAttackAction(ctx, actor, source)
		return
	}
	chaseX, chaseY, ok := normalAttackApproachCellFromTarget(ctx, playerX, playerY, targetX, targetY, attackRange)
	if !ok {
		glog.Warnf("%s attack chase blocked target=%d player=%d,%d target=%d,%d range=%d", source, actor.ID, playerX, playerY, targetX, targetY, attackRange)
		m.setWalkCooldown(walkRequestCooldown)
		return
	}
	m.pendingAttack = attackIntent{
		targetID:    actor.ID,
		expires:     now.Add(8 * time.Second),
		lastChaseAt: now,
	}
	glog.Debugf("%s attack chase target=%d player=%d,%d target=%d,%d range=%d chase=%d,%d", source, actor.ID, playerX, playerY, targetX, targetY, attackRange, chaseX, chaseY)
	m.requestWalk(ctx, chaseX, chaseY, source+" attack chase")
}

func (m *WorldMode) lockAttack(targetID uint32) {
	if targetID == 0 || m.lockedAttackID == targetID {
		return
	}
	m.lockedAttackID = targetID
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
}

func (m *WorldMode) clearLockedAttack() {
	m.lockedAttackID = 0
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
}

func (m *WorldMode) cancelAttackIntent() {
	m.pendingAttack = attackIntent{}
	m.clearLockedAttack()
	m.clearAttackFocus()
}

func (m *WorldMode) focusAttackTarget(targetID uint32, now time.Time) {
	if targetID == 0 {
		m.clearAttackFocus()
		return
	}
	if m.attackFocusID == targetID {
		return
	}
	m.attackFocusID = targetID
	m.attackFocusStart = now
}

func (m *WorldMode) clearAttackFocus() {
	m.attackFocusID = 0
	m.attackFocusStart = time.Time{}
}

func (m *WorldMode) continuePendingAttack(ctx client.Context, source string) {
	m.updatePendingAttack(ctx, source, true)
}

func (m *WorldMode) updatePendingAttack(ctx client.Context, source string, logOutOfRange bool) {
	if m.pendingAttack.targetID == 0 {
		return
	}
	if ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(m.pendingAttack.expires) {
		glog.Debugf("%s pending attack expired target=%d", source, m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	actor, ok := ctx.World.Actors[m.pendingAttack.targetID]
	if !ok {
		glog.Debugf("%s pending attack target vanished id=%d", source, m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := currentPlayerCell(ctx, now)
	targetX, targetY := actorCurrentCell(actor, now)
	if !normalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange) {
		if logOutOfRange {
			glog.Debugf("%s pending attack still out of range target=%d player=%d,%d target=%d,%d range=%d", source, actor.ID, playerX, playerY, targetX, targetY, attackRange)
		}
		if movingActorDestinationWithinRange(ctx.World.Player, targetX, targetY, attackRange, normalAttackTargetWithinRange) {
			return
		}
		if attackRetryDue(m.pendingAttack.lastChaseAt, now) {
			m.requestAttack(ctx, actor, "pending")
		}
		return
	}
	readyAt := pendingAttackReadyAt(ctx.World.Player, now)
	if m.pendingAttack.readyAt.IsZero() {
		m.pendingAttack.readyAt = readyAt
	}
	if logOutOfRange {
		glog.Debugf("%s pending attack scheduled target=%d delay_ms=%d", source, actor.ID, maxInt(0, int(m.pendingAttack.readyAt.Sub(now).Milliseconds())))
	}
}

func (m *WorldMode) processPendingAttack(ctx client.Context) {
	if m.pendingAttack.targetID == 0 || m.pendingAttack.readyAt.IsZero() {
		return
	}
	now := time.Now()
	if now.After(m.pendingAttack.expires) {
		glog.Debugf("pending attack expired target=%d", m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	if now.Before(m.pendingAttack.readyAt) {
		return
	}
	actor, ok := ctx.World.Actors[m.pendingAttack.targetID]
	if !ok {
		glog.Debugf("pending attack target vanished id=%d", m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := currentPlayerCell(ctx, now)
	targetX, targetY := actorCurrentCell(actor, now)
	if !normalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange) {
		glog.Debugf("pending attack became out of range target=%d player=%d,%d target=%d,%d range=%d", actor.ID, playerX, playerY, targetX, targetY, attackRange)
		m.pendingAttack.readyAt = time.Time{}
		m.requestAttack(ctx, actor, "pending")
		return
	}
	m.pendingAttack = attackIntent{}
	m.sendAttackAction(ctx, actor, "pending")
}

func (m *WorldMode) processLockedAttack(ctx client.Context) {
	if m.lockedAttackID == 0 || ctx.Network == nil {
		return
	}
	if !normalAttackLockActive(ctx) {
		m.clearLockedAttack()
		return
	}
	if m.pendingAttack.targetID == m.lockedAttackID {
		return
	}
	now := time.Now()
	if actorIsMovingAt(ctx.World.Player, now) {
		return
	}
	actor, ok := ctx.World.Actors[m.lockedAttackID]
	if !ok {
		glog.Debugf("locked attack target vanished id=%d", m.lockedAttackID)
		m.clearLockedAttack()
		return
	}
	if !actorCanBeAttackClicked(ctx, actor) {
		glog.Debugf("locked attack target no longer attackable id=%d object_type=%d", actor.ID, actor.ObjectType)
		m.clearLockedAttack()
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := currentPlayerCell(ctx, now)
	targetX, targetY := actorCurrentCell(actor, now)
	if normalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange) {
		if !attackRetryDue(m.lastAttackAt, now) {
			return
		}
		glog.Debugf("locked attack retry target=%d player=%d,%d target=%d,%d range=%d", actor.ID, playerX, playerY, targetX, targetY, attackRange)
		m.sendAttackAction(ctx, actor, "locked")
		return
	}
	if !attackRetryDue(m.lastChaseAt, now) {
		return
	}
	m.lastChaseAt = now
	glog.Debugf("locked attack chase retry target=%d player=%d,%d target=%d,%d range=%d", actor.ID, playerX, playerY, targetX, targetY, attackRange)
	m.requestAttack(ctx, actor, "locked")
}

func attackRetryDue(last time.Time, now time.Time) bool {
	return last.IsZero() || now.Sub(last) >= attackRetryInterval
}

func normalAttackLockActive(ctx client.Context) bool {
	return (ctx.Input != nil && ctx.Input.Pressed(input.KeyCtrl)) || (ctx.Session != nil && ctx.Session.NoCtrl)
}

func pendingAttackReadyAt(player world.Actor, now time.Time) time.Time {
	readyAt := now.Add(60 * time.Millisecond)
	if actorIsMovingAt(player, now) && player.MoveDuration > 0 {
		walkReadyAt := player.MoveStarted.Add(player.MoveDuration).Add(60 * time.Millisecond)
		if walkReadyAt.After(readyAt) {
			readyAt = walkReadyAt
		}
	}
	return readyAt
}

func (m *WorldMode) sendAttackAction(ctx client.Context, actor world.Actor, source string) {
	if err := ctx.Network.SendActionRequest(actor.ID, network.ActionAttack); err == nil {
		m.lastAttackAt = time.Now()
		m.setWalkCooldown(walkRequestCooldown)
	} else {
		glog.Warnf("%s attack request failed target=%d action=%d: %v", source, actor.ID, network.ActionAttack, err)
		m.setWalkCooldown(walkErrorCooldown)
	}
}

func (m *WorldMode) applyActorActionNotify(ctx client.Context, action network.ActorActionNotify) {
	glog.Debugf("actor action src=%d dst=%d skill=%d level=%d damage=%d left_damage=%d hits=%d action=%d src_speed=%d dst_speed=%d tick=%d", action.SourceID, action.TargetID, action.SkillID, action.SkillLevel, action.Damage, action.LeftDamage, action.HitCount, action.Action, action.SourceSpeed, action.TargetSpeed, action.ServerTick)
	now := time.Now()
	m.applyActorActionAIState(ctx, action, now)
	if action.Action == network.ActorActionPickupItem {
		m.applyActorPickupActionNotify(ctx, action, now)
		return
	}
	if action.Action == network.ActionSitDown || action.Action == network.ActionStandUp {
		m.applyActorSitStandActionNotify(ctx, action)
		return
	}
	source, sourceOK, sourceLocal := actorForCombatID(ctx, action.SourceID)
	target, targetOK, targetLocal := actorForCombatID(ctx, action.TargetID)
	if action.SkillID > 0 {
		m.applySkillNameBubble(ctx, action.SourceID, action.SkillID, now)
		m.applyFalconActorActionNotify(ctx, action, now)
	}
	if sourceOK && targetOK {
		m.faceCombatSource(ctx, source, sourceLocal, target)
		source.Dir = directionFromDelta(source.X, source.Y, target.X, target.Y, source.Dir)
	}
	attackDuration := combatDuration(action.SourceSpeed, defaultAttackAnimationDuration)
	attackFamily := spriteActionNonPCAttack
	playerBowAttack := sourceOK && res.HasPlayerJobToken(int(source.Job)) && actorUsesBow(ctx.Resources, source)
	normalPlayerBowAttack := action.SkillID == 0 && playerBowAttack
	doubleStrafeBowAttack := action.SkillID == db.SkillACDouble && playerBowAttack
	referenceBowFlightAttack := normalPlayerBowAttack || doubleStrafeBowAttack
	if sourceOK {
		attackFamily = skillActionFamilyForActorWithResources(ctx.Resources, source, action.SkillID)
		if action.SkillID > 0 {
			m.startSkillActionAnimation(ctx, action.SourceID, source, skillAction(action.SkillID), now, attackDuration)
		} else if sourceLocal && res.HasPlayerJobToken(int(source.Job)) {
			m.startCombatAnimationWithTimingAndNext(ctx, action.SourceID, source, attackFamily, now, attackDuration, readyFightAnimation(now.Add(attackDuration)))
		} else {
			m.startCombatAnimationWithTiming(ctx, action.SourceID, source, attackFamily, now, attackDuration)
		}
	}
	hitDelay := combatDuration(action.SourceSpeed, 0)
	if sourceOK {
		if actionDef, ok := m.actorResolvedAction(ctx, source, attackFamily); ok {
			actionACT := m.actorActionACT(ctx, source)
			soundMotion := firstActionSoundMotion(actionDef)
			if referenceBowFlightAttack {
				soundMotion = actionAttackMarkerMotion(actionACT, actionDef)
			}
			hitDelay = combatMotionDelayFromAction(actionDef, attackDuration, soundMotion)
			if sounds := actionSFXCandidatesForActor(source, actionACT, actionDef, soundMotion); len(sounds) > 0 {
				m.scheduleSoundAtActor(now.Add(hitDelay), action.SourceID, sounds...)
			}
		}
	}
	releaseAt := now.Add(hitDelay)
	hitAt := releaseAt
	projectileAt := now
	projectileDuration := time.Duration(0)
	if referenceBowFlightAttack {
		// The reference client reveals the arrow at the ACT "atk" marker, then
		// gives it eight 24 ms ticks to reach the target before applying the hit.
		projectileAt = releaseAt
		projectileDuration = referenceBowArrowFlightDuration
		hitAt = releaseAt.Add(referenceBowArrowFlightDuration)
	}
	dealsDamage := actionDealsDamage(action)
	if targetOK {
		if hitAt.Before(now) {
			hitAt = now
		}
		if action.SkillID > 0 {
			m.addSkillBeginEffect(ctx, action, now)
			if doubleStrafeBowAttack {
				// Double Strafe releases one visible arrow; its two impacts are
				// delivered separately after that arrow reaches the target.
				m.addSkillBeforeHitEffectOnce(ctx, action, projectileAt, projectileDuration)
			} else {
				m.addSkillBeforeHitEffect(ctx, action, now)
			}
		} else {
			m.addNormalAttackBeforeHitEffect(ctx, action, source, sourceOK, projectileAt, projectileDuration)
		}
		if dealsDamage {
			m.clearActorCastBar(ctx, action.TargetID)
			m.addNormalAttackHitEffect(ctx, action, target, targetOK, hitAt)
			if actionHasHitReaction(action) && skillTargetUsesHitReaction(action, sourceLocal, targetLocal) {
				hurtDuration := combatDuration(action.TargetSpeed, defaultHitAnimationDuration)
				m.startCombatAnimationWithNext(ctx, action.TargetID, hurtActionFamilyForActor(target), hitAt, hurtDuration, postHurtAnimation(target, hitAt.Add(hurtDuration)))
			}
			hitSoundCount := 1
			if doubleStrafeBowAttack {
				hitSoundCount = actionVisualHitCount(action)
			}
			hitSounds := combatHitSFXCandidates(action, source, sourceOK, target, targetOK)
			for i := 0; i < hitSoundCount; i++ {
				m.scheduleSoundAtActor(hitAt.Add(multiHitDelay*time.Duration(i)), action.TargetID, hitSounds...)
			}
		}
		if action.SkillID > 0 {
			m.addSkillEffect(ctx, action, hitAt)
			if dealsDamage {
				m.addSkillHitEffect(ctx, action, hitAt)
			}
		}
	}
	x, y := ctx.World.Player.X, ctx.World.Player.Y
	if targetOK {
		x, y = target.X, target.Y
	} else if isLocalActor(ctx, action.TargetID) {
		x, y = ctx.World.Player.X, ctx.World.Player.Y
	}
	m.addActionDamageFloaters(ctx, action, target, targetOK, targetLocal, sourceLocal, x, y, hitAt)
	if sourceLocal {
		m.maybeSendPetHuntingTalk(ctx, now)
	}
}

func (m *WorldMode) applyActorActionAIState(ctx client.Context, action network.ActorActionNotify, now time.Time) {
	if action.SourceID != 0 {
		motion := aiMotionAttack
		expires := now.Add(combatDuration(action.SourceSpeed, defaultAttackAnimationDuration))
		if action.Action == network.ActorActionPickupItem || action.Action == network.ActionSitDown || action.Action == network.ActionStandUp {
			motion = aiMotionStand
			expires = time.Time{}
		}
		m.setActorAIMotion(ctx, action.SourceID, motion, action.TargetID, expires)
	}
	if action.TargetID != 0 && actionHasHitReaction(action) {
		m.setActorAIMotion(ctx, action.TargetID, aiMotionStand, action.SourceID, time.Time{})
	}
}

func (m *WorldMode) addSkillBeginEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	m.addSkillBeginEffectsAt(ctx, action.SkillID, action.SourceID, action.TargetID, starts)
}

func (m *WorldMode) addSkillBeginEffectsAt(ctx client.Context, skillID uint16, sourceID, targetID uint32, starts time.Time) {
	if skillID == 0 {
		return
	}
	for _, effectID := range skillBeginEffectIDs(skillID) {
		actorID := sourceID
		if effectDetachesLocalActor(effectID) && isLocalActor(ctx, actorID) {
			actorID = 0
		}
		if m.addWorldEffectAt(ctx, effectID, actorID, starts) {
			glog.Debugf("skill begin effect skill=%d src=%d target=%d effect=%d", skillID, sourceID, targetID, effectID)
		}
	}
}

func (m *WorldMode) addNormalAttackBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, source world.Actor, sourceOK bool, starts time.Time, duration time.Duration) {
	if action.SkillID != 0 || !sourceOK {
		return
	}
	effectID, ok := normalAttackBeforeHitEffectID(ctx.Resources, source)
	if !ok {
		return
	}
	if m.addWorldEffectBetweenAtDuration(ctx, effectID, action.TargetID, action.SourceID, starts, duration) {
		glog.Debugf("normal attack before-hit effect src=%d target=%d effect=%d", action.SourceID, action.TargetID, effectID)
	}
}

func (m *WorldMode) addNormalAttackHitEffect(ctx client.Context, action network.ActorActionNotify, target world.Actor, targetOK bool, starts time.Time) {
	if action.SkillID != 0 || action.Damage <= 0 || !targetOK || !actorUsesReferenceNormalHitEffect(target) {
		return
	}
	count := actionVisualHitCount(action)
	for i := 0; i < count; i++ {
		effectStarts := starts.Add(multiHitDelay * time.Duration(i))
		if m.addWorldEffectAt(ctx, effectHit1, action.TargetID, effectStarts) {
			glog.Debugf("normal hit effect src=%d target=%d effect=%d hit=%d/%d", action.SourceID, action.TargetID, effectHit1, i+1, count)
		}
		if actionUsesReferenceCriticalHitEffect(action) {
			if m.addWorldEffectAt(ctx, effectBashHit, action.TargetID, effectStarts) {
				glog.Debugf("critical hit effect src=%d target=%d effect=%d hit=%d/%d", action.SourceID, action.TargetID, effectBashHit, i+1, count)
			}
		}
	}
}

func actionUsesReferenceCriticalHitEffect(action network.ActorActionNotify) bool {
	return action.Action == 10 || action.Action == 13
}

func actorUsesReferenceNormalHitEffect(actor world.Actor) bool {
	if !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case actorObjectTypeMob, actorObjectTypeNPCABR, actorObjectTypeNPCBionic:
		return true
	default:
		return false
	}
}

func actorUsesBow(manager *res.Manager, actor world.Actor) bool {
	if actor.Weapon <= 0 {
		return false
	}
	return res.PlayerWeaponIsBow(manager, int(actor.Weapon))
}

func normalAttackBeforeHitEffectID(manager *res.Manager, actor world.Actor) (int, bool) {
	if res.HasPlayerJobToken(int(actor.Job)) && actorUsesBow(manager, actor) {
		return effectArrowShot, true
	}
	return normalAttackProjectileEffectIDForName(db.MonsterAttackProjectile[int(actor.Job)])
}

func normalAttackProjectileEffectIDForName(name string) (int, bool) {
	switch name {
	case "ef_arrow_projectile":
		return effectArrowShot, true
	default:
		return 0, false
	}
}

func (m *WorldMode) addSkillEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 {
		return
	}
	for _, effectID := range skillEffectIDs(action.SkillID) {
		if m.addWorldEffectBetweenAt(ctx, effectID, action.TargetID, action.SourceID, starts) {
			glog.Debugf("skill effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
	for _, effectID := range skillEffectOnCasterIDs(action.SkillID) {
		if m.addWorldEffectAt(ctx, effectID, action.SourceID, starts) {
			glog.Debugf("skill caster effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
}

func (m *WorldMode) addSkillBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 {
		return
	}
	effectIDs := skillBeforeHitEffectIDs(action.SkillID)
	selfEffectIDs := skillBeforeHitEffectSelfIDs(action.SkillID)
	if len(effectIDs) == 0 && len(selfEffectIDs) == 0 {
		return
	}
	count := actionVisualHitCount(action)
	for i := 0; i < count; i++ {
		effectStarts := starts.Add(multiHitDelay * time.Duration(i))
		for _, effectID := range effectIDs {
			if m.addWorldEffectBetweenAt(ctx, effectID, action.TargetID, action.SourceID, effectStarts) {
				glog.Debugf("skill before-hit effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
		for _, effectID := range selfEffectIDs {
			if m.addWorldEffectAt(ctx, effectID, action.SourceID, effectStarts) {
				glog.Debugf("skill before-hit self effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
	}
}

func (m *WorldMode) addSkillBeforeHitEffectOnce(ctx client.Context, action network.ActorActionNotify, starts time.Time, duration time.Duration) {
	if action.SkillID == 0 {
		return
	}
	for _, effectID := range skillBeforeHitEffectIDs(action.SkillID) {
		if m.addWorldEffectBetweenAtDuration(ctx, effectID, action.TargetID, action.SourceID, starts, duration) {
			glog.Debugf("skill before-hit effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
	for _, effectID := range skillBeforeHitEffectSelfIDs(action.SkillID) {
		if m.addWorldEffectBetweenAtDuration(ctx, effectID, action.SourceID, 0, starts, duration) {
			glog.Debugf("skill before-hit self effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
}

func (m *WorldMode) addSkillHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 || action.Damage == 0 {
		return
	}
	effectIDs := skillHitEffectIDs(action.SkillID)
	casterEffectIDs := skillHitEffectOnCasterIDs(action.SkillID)
	if len(effectIDs) == 0 && len(casterEffectIDs) == 0 {
		return
	}
	count := actionVisualHitCount(action)
	for i := 0; i < count; i++ {
		effectStarts := starts.Add(multiHitDelay * time.Duration(i))
		for _, effectID := range effectIDs {
			if m.addWorldEffectAt(ctx, effectID, action.TargetID, effectStarts) {
				glog.Debugf("skill hit effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
		for _, effectID := range casterEffectIDs {
			if m.addWorldEffectAt(ctx, effectID, action.SourceID, effectStarts) {
				glog.Debugf("skill hit caster effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
	}
}

func actionVisualHitCount(action network.ActorActionNotify) int {
	if action.HitCount == 0 {
		return 1
	}
	return maxInt(1, int(action.HitCount))
}

func (m *WorldMode) addActionDamageFloaters(ctx client.Context, action network.ActorActionNotify, target world.Actor, targetOK, targetLocal, sourceLocal bool, x, y int, hitAt time.Time) {
	text, kind, floaterColor := actionDamageFloater(action, targetLocal, sourceLocal)
	if text == "" {
		return
	}
	if ctx.World != nil && ctx.World.MapProperty.IsSiege() && damageFloaterHiddenInSiege(kind) {
		return
	}
	count := actionVisualHitCount(action)
	total := action.Damage + action.LeftDamage
	if count <= 1 || total <= 0 {
		m.addDamageFloater(action.TargetID, x, y, text, floaterColor, kind, hitAt)
		return
	}
	values := actionDamageFloaterValues(action, count)
	combo := actionShowsComboDamageFloaters(action, target, targetOK, targetLocal)
	for i := 0; i < count; i++ {
		starts := hitAt.Add(multiHitDelay * time.Duration(i))
		m.addDamageFloater(action.TargetID, x, y, strconv.Itoa(values[i]), floaterColor, kind, starts)
		if combo {
			m.addComboDamageFloater(action.TargetID, x, y, actionComboDamageFloaterValue(action, count, i), starts, i+1 == count)
		}
	}
}

func damageFloaterHiddenInSiege(kind damageFloaterKind) bool {
	switch kind {
	case damageFloaterNormal, damageFloaterCritical, damageFloaterIncoming, damageFloaterCombo:
		return true
	default:
		return false
	}
}

func (m *WorldMode) addDamageFloater(actorID uint32, x, y int, text string, floaterColor color.RGBA, kind damageFloaterKind, starts time.Time) {
	duration := damageFloaterDuration(kind)
	m.addDamageFloaterWithTiming(actorID, x, y, text, floaterColor, kind, starts, duration, duration)
}

func (m *WorldMode) addComboDamageFloater(actorID uint32, x, y int, value int, starts time.Time, final bool) {
	if value <= 0 {
		return
	}
	visible := damageFloaterComboTransientDuration()
	if final {
		visible = damageFloaterDuration(damageFloaterCombo)
	}
	m.addDamageFloaterWithTiming(actorID, x, y, strconv.Itoa(value), damageFloaterYellow, damageFloaterCombo, starts, visible, damageFloaterDuration(damageFloaterCombo))
}

func (m *WorldMode) addDamageFloaterWithTiming(actorID uint32, x, y int, text string, floaterColor color.RGBA, kind damageFloaterKind, starts time.Time, visible, duration time.Duration) {
	if visible <= 0 {
		visible = damageFloaterDuration(kind)
	}
	if duration <= 0 {
		duration = visible
	}
	m.damageFloaters = append(m.damageFloaters, damageFloater{
		actorID:  actorID,
		x:        x,
		y:        y,
		text:     text,
		color:    floaterColor,
		kind:     kind,
		starts:   starts,
		expires:  starts.Add(visible),
		duration: duration,
	})
}

func actionDamageFloaterValues(action network.ActorActionNotify, count int) []int {
	values := make([]int, count)
	if count <= 0 {
		return values
	}
	total := int(action.Damage + action.LeftDamage)
	if action.SkillID != 0 && action.Damage > 0 {
		value := int(math.Floor(float64(action.Damage) / float64(count)))
		for i := range values {
			values[i] = value
		}
		return values
	}
	base := total / count
	rem := total % count
	for i := range values {
		values[i] = base
		if i < rem {
			values[i]++
		}
	}
	return values
}

func actionComboDamageFloaterValue(action network.ActorActionNotify, count, index int) int {
	if action.SkillID != 0 && action.Damage > 0 && count > 0 {
		return int(math.Floor((float64(action.Damage) / float64(count)) * float64(index+1)))
	}
	values := actionDamageFloaterValues(action, count)
	total := 0
	for i := 0; i <= index && i < len(values); i++ {
		total += values[i]
	}
	return total
}

func actionShowsComboDamageFloaters(action network.ActorActionNotify, target world.Actor, targetOK, targetLocal bool) bool {
	if !targetOK || targetLocal || action.Damage <= 0 || actionVisualHitCount(action) <= 1 {
		return false
	}
	if action.SkillID != 0 {
		return !actorIsPlayerObject(target)
	}
	switch action.Action {
	case 8, 9, 13:
		return actorHasMobObjectType(target)
	default:
		return false
	}
}

func actorIsPlayerObject(actor world.Actor) bool {
	if actor.HasObjectType {
		return actor.ObjectType == actorObjectTypePC
	}
	return res.HasPlayerJobToken(int(actor.Job))
}

func (m *WorldMode) applyActorPickupActionNotify(ctx client.Context, action network.ActorActionNotify, now time.Time) {
	source, sourceOK, sourceLocal := actorForCombatID(ctx, action.SourceID)
	if !sourceOK {
		return
	}
	if ctx.World != nil {
		if item, ok := ctx.World.Items[action.TargetID]; ok {
			dir := directionFromDelta(source.X, source.Y, item.X, item.Y, source.Dir)
			if sourceLocal {
				ctx.World.Player.Dir = dir
				ctx.World.Dir = dir
				if ctx.Session != nil {
					ctx.Session.PlayerDir = dir
				}
			} else {
				source.Dir = dir
				upsertActor(ctx, source)
			}
		}
	}
	m.startCombatAnimation(ctx, action.SourceID, spriteActionPickup, now, pickupAnimationDuration)
}

func (m *WorldMode) applyActorSitStandActionNotify(ctx client.Context, action network.ActorActionNotify) {
	id := action.SourceID
	if id == 0 {
		id = action.TargetID
	}
	if id == 0 || ctx.World == nil {
		return
	}
	sitting := action.Action == network.ActionSitDown
	if isLocalActor(ctx, id) {
		ctx.World.Player.Sitting = sitting
		if sitting {
			ctx.World.Player.Moving = false
			m.cancelAttackIntent()
		}
		m.clearLocalActorAction(ctx)
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return
	}
	actor.Sitting = sitting
	if sitting {
		actor.Moving = false
	}
	upsertActor(ctx, actor)
	if !sitting {
		actor = ctx.World.Actors[id]
		actor.Sitting = false
		ctx.World.Actors[id] = actor
	}
	m.clearActorAction(ctx, id)
}

func (m *WorldMode) applyActorHPUpdate(update network.ActorHPUpdate) {
	if update.ID == 0 || update.MaxHP <= 0 {
		return
	}
	hp := update.HP
	if hp < 0 {
		hp = 0
	}
	if hp > update.MaxHP {
		hp = update.MaxHP
	}
	if m.actorLife == nil {
		m.actorLife = make(map[uint32]actorLife)
	}
	m.actorLife[update.ID] = actorLife{
		hp:        hp,
		maxHP:     update.MaxHP,
		updatedAt: time.Now(),
	}
	glog.Debugf("actor hp id=%d hp=%d max_hp=%d tiny=%t", update.ID, hp, update.MaxHP, update.Tiny)
}

func actorForCombatID(ctx client.Context, id uint32) (world.Actor, bool, bool) {
	if ctx.World == nil || id == 0 {
		return world.Actor{}, false, false
	}
	if isLocalActor(ctx, id) {
		actor := ctx.World.Player
		character := ctx.Session.SelectedCharacter()
		actor.ID = id
		actor.Job = character.Job
		actor.Head = character.Hair
		actor.Weapon = character.Weapon
		actor.Shield = character.Shield
		actor.HeadTop = character.HeadTop
		actor.HeadMid = character.HeadMid
		actor.HeadLow = character.HeadLow
		actor.Sex = ctx.Session.Sex
		actor.Level = ctx.Session.Progress.BaseLevel
		actor.HasLevel = actor.Level > 0
		actor.Appearance = true
		return actor, true, true
	}
	actor, ok := ctx.World.Actors[id]
	return actor, ok, false
}

func (m *WorldMode) faceCombatSource(ctx client.Context, source world.Actor, sourceLocal bool, target world.Actor) {
	dir := directionFromDelta(source.X, source.Y, target.X, target.Y, source.Dir)
	if sourceLocal {
		ctx.World.Player.Dir = dir
		ctx.World.Dir = dir
		return
	}
	source.Dir = dir
	upsertActor(ctx, source)
}

func (m *WorldMode) startActorAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimationWithOptions(ctx, id, actionFamily, started, duration, false)
}

func (m *WorldMode) startHeldActorAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimationWithOptions(ctx, id, actionFamily, started, duration, true)
}

func (m *WorldMode) startActorAnimationWithOptions(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, holdFinal bool) {
	m.setActorAction(ctx, id, actorAnimation{
		actionFamily: actionFamily,
		started:      started,
		duration:     duration,
		play:         true,
		hasPlay:      true,
		holdFinal:    holdFinal,
	})
}

func (m *WorldMode) startActorAnimationWithNext(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, next *actorAnimation) {
	m.setActorAction(ctx, id, actorAnimation{
		actionFamily: actionFamily,
		started:      started,
		duration:     duration,
		play:         true,
		hasPlay:      true,
		next:         cloneActorAnimation(next),
	})
}

func (m *WorldMode) setActorAction(ctx client.Context, id uint32, anim actorAnimation) {
	if id == 0 || anim.actionFamily < 0 {
		return
	}
	if anim.started.IsZero() {
		anim.started = time.Now()
	}
	if actorActionResetsWalk(anim) {
		resumeWalk := m.actorActionShouldResumeWalk(ctx, id, anim)
		if anim.started.After(time.Now()) {
			m.scheduleActorStop(id, anim.started, resumeWalk, anim.started.Add(anim.duration))
		} else {
			if resumeWalk {
				m.pauseActorMovementForResume(ctx, id, anim.started, anim.started.Add(anim.duration))
			} else {
				m.stopActorMovementAt(ctx, id, anim.started)
			}
		}
	}
	if m.actorAnims == nil {
		m.actorAnims = make(map[uint32]actorAnimation)
	}
	anim.next = cloneActorAnimation(anim.next)
	m.actorAnims[id] = anim
}

func actorActionResetsWalk(anim actorAnimation) bool {
	switch anim.actionFamily {
	case spriteActionWalk:
		return false
	default:
		return true
	}
}

func (m *WorldMode) actorActionShouldResumeWalk(ctx client.Context, id uint32, anim actorAnimation) bool {
	if anim.actionFamily != spriteActionPCHurt && anim.actionFamily != spriteActionNonPCHurt {
		return false
	}
	return isLocalActor(ctx, id) && m.hasCombatFocus()
}

func (m *WorldMode) hasCombatFocus() bool {
	return m.lockedAttackID != 0 || m.pendingAttack.targetID != 0
}

func cloneActorAnimation(anim *actorAnimation) *actorAnimation {
	if anim == nil {
		return nil
	}
	cloned := *anim
	cloned.next = cloneActorAnimation(anim.next)
	return &cloned
}

func readyFightAnimation(started time.Time) *actorAnimation {
	return &actorAnimation{
		actionFamily: spriteActionPCReadyFight,
		started:      started,
		loop:         true,
		play:         true,
		hasPlay:      true,
	}
}

func postHurtAnimation(actor world.Actor, started time.Time) *actorAnimation {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return readyFightAnimation(started)
	}
	return nil
}

func (m *WorldMode) clearLocalActorAction(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	m.clearActorAction(ctx, ctx.Session.AccountID)
	m.clearActorAction(ctx, ctx.Session.CharID)
}

func (m *WorldMode) clearActorAction(ctx client.Context, id uint32) {
	if id == 0 || m.actorAnims == nil {
		return
	}
	if anim, ok := m.actorAnims[id]; ok && m.actorActionAnimationIsDeath(ctx, id, anim) {
		return
	}
	delete(m.actorAnims, id)
}

func (m *WorldMode) actorActionAnimationIsDeath(ctx client.Context, id uint32, anim actorAnimation) bool {
	if anim.actionFamily == spriteActionPCDeath {
		return true
	}
	if anim.actionFamily != spriteActionNonPCDeath {
		return false
	}
	actor, ok, local := actorForCombatID(ctx, id)
	if !ok {
		return true
	}
	return !local && !res.HasPlayerJobToken(int(actor.Job))
}

func (m *WorldMode) startCombatAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimation(ctx, id, actionFamily, started, duration)
	if ctx.Session == nil || !isLocalActor(ctx, id) {
		return
	}
	m.startActorAnimation(ctx, ctx.Session.AccountID, actionFamily, started, duration)
	m.startActorAnimation(ctx, ctx.Session.CharID, actionFamily, started, duration)
}

func (m *WorldMode) startCombatAnimationWithTiming(ctx client.Context, id uint32, actor world.Actor, actionFamily int, started time.Time, duration time.Duration) {
	anim := timedCombatAnimation(m.actorActionFrameDelay(ctx, actor, actionFamily, duration), actionFamily, started, duration, nil)
	m.setCombatActorAction(ctx, id, anim)
}

func (m *WorldMode) startCombatAnimationWithNext(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, next *actorAnimation) {
	m.startActorAnimationWithNext(ctx, id, actionFamily, started, duration, next)
	if ctx.Session == nil || !isLocalActor(ctx, id) {
		return
	}
	m.startActorAnimationWithNext(ctx, ctx.Session.AccountID, actionFamily, started, duration, next)
	m.startActorAnimationWithNext(ctx, ctx.Session.CharID, actionFamily, started, duration, next)
}

func (m *WorldMode) startCombatAnimationWithTimingAndNext(ctx client.Context, id uint32, actor world.Actor, actionFamily int, started time.Time, duration time.Duration, next *actorAnimation) {
	anim := timedCombatAnimation(m.actorActionFrameDelay(ctx, actor, actionFamily, duration), actionFamily, started, duration, next)
	m.setCombatActorAction(ctx, id, anim)
}

func (m *WorldMode) setCombatActorAction(ctx client.Context, id uint32, anim actorAnimation) {
	m.setActorAction(ctx, id, anim)
	if ctx.Session == nil || !isLocalActor(ctx, id) {
		return
	}
	m.setActorAction(ctx, ctx.Session.AccountID, anim)
	m.setActorAction(ctx, ctx.Session.CharID, anim)
}

func timedCombatAnimation(frameDelay time.Duration, actionFamily int, started time.Time, duration time.Duration, next *actorAnimation) actorAnimation {
	anim := actorAnimation{
		actionFamily: actionFamily,
		started:      started,
		duration:     duration,
		play:         true,
		hasPlay:      true,
		next:         cloneActorAnimation(next),
	}
	if frameDelay > 0 {
		anim.speed = frameDelay
		anim.hasSpeed = true
	}
	return anim
}

func (m *WorldMode) startHeldCombatAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startHeldActorAnimation(ctx, id, actionFamily, started, duration)
	if ctx.Session == nil || !isLocalActor(ctx, id) {
		return
	}
	m.startHeldActorAnimation(ctx, ctx.Session.AccountID, actionFamily, started, duration)
	m.startHeldActorAnimation(ctx, ctx.Session.CharID, actionFamily, started, duration)
}

func (m *WorldMode) clearLocalDeathStateIfAlive(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	if ctx.Session.Vitals.HP <= 0 && ctx.Session.Selected.HP <= 0 {
		return
	}
	// rAthena restores HP before sending the map change for a save-point
	// respawn. Keep the held death pose during that gap; handleMapChange clears
	// it only after the fade-out has reached black.
	if m.ui.escapeMenu.PendingAction() == ui.EscapeMenuActionSavePoint {
		return
	}
	m.clearLocalDeathState(ctx)
}

func (m *WorldMode) clearLocalDeathState(ctx client.Context) {
	m.ui.escapeMenu.ResetDeath(ctx)
	if ctx.Session != nil {
		ctx.Session.Dead = false
	}
	if ctx.Session == nil || m.actorAnims == nil {
		return
	}
	m.clearActorDeathAnimation(ctx.Session.AccountID)
	m.clearActorDeathAnimation(ctx.Session.CharID)
}

func (m *WorldMode) clearActorDeathAnimation(id uint32) {
	if id == 0 || m.actorAnims == nil {
		return
	}
	anim, ok := m.actorAnims[id]
	if !ok {
		return
	}
	if anim.actionFamily != spriteActionPCDeath && anim.actionFamily != spriteActionNonPCDeath {
		return
	}
	delete(m.actorAnims, id)
}

func (m *WorldMode) actorAnimation(id uint32, now time.Time) (actorAnimation, bool) {
	if m.actorAnims == nil || id == 0 {
		return actorAnimation{}, false
	}
	anim, ok := m.actorAnims[id]
	if !ok {
		return actorAnimation{}, false
	}
	if anim.duration <= 0 {
		anim.duration = defaultAttackAnimationDuration
	}
	if now.Before(anim.started) {
		return actorAnimation{}, false
	}
	if anim.loop {
		return anim, true
	}
	if !now.Before(anim.started.Add(anim.duration)) {
		if anim.holdFinal {
			return anim, true
		}
		if anim.next != nil {
			next := *anim.next
			if next.started.IsZero() {
				next.started = anim.started.Add(anim.duration).Add(next.startDelay)
				next.startDelay = 0
			}
			m.actorAnims[id] = next
			return m.actorAnimation(id, now)
		}
		delete(m.actorAnims, id)
		return actorAnimation{}, false
	}
	return anim, true
}

func attackActionFamilyForActor(actor world.Actor) int {
	return attackActionFamilyForActorWithResources(nil, actor)
}

func attackActionFamilyForActorWithResources(manager *res.Manager, actor world.Actor) int {
	if actorIsMercenary(actor) {
		return mercenaryAttackActionFamily(int(actor.Job), actor.Sex, int(actor.Weapon))
	}
	job := actorVisualJob(actor)
	if res.HasPlayerJobToken(job) {
		weapon, _ := res.NormalizePlayerWeaponShield(int(actor.Weapon), int(actor.Shield))
		viewID := res.PlayerWeaponViewID(manager, weapon)
		if db.PlayerWeaponType(viewID) > 0 {
			weapon = viewID
		}
		switch db.PlayerWeaponAction(job, actor.Sex, weapon) {
		case db.PlayerWeaponActionAttack2:
			return spriteActionPCAttack2
		case db.PlayerWeaponActionAttack3:
			return spriteActionPCAttack3
		default:
			return spriteActionPCAttack1
		}
	}
	return spriteActionNonPCAttack
}

func skillActionFamilyForActorWithResources(manager *res.Manager, actor world.Actor, skillID uint16) int {
	if skillID == 0 {
		return attackActionFamilyForActorWithResources(manager, actor)
	}
	return skillAction(skillID).actionFamilyForActorWithResources(manager, actor)
}

func skillCastActionFamilyForActor(actor world.Actor, skillID uint16) int {
	if actorIsMercenary(actor) {
		return spriteActionPCSkill
	}
	if !res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionNonPCAttack
	}
	return spriteActionPCReadyFight
}

func skillTargetUsesHitReaction(action network.ActorActionNotify, sourceLocal, targetLocal bool) bool {
	if action.SkillID > 0 && sourceLocal && targetLocal && action.Action == network.ActorActionSkill {
		return false
	}
	return true
}

func hurtActionFamilyForActor(actor world.Actor) int {
	if actorIsMercenary(actor) {
		return spriteActionPCHurt
	}
	if res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionPCHurt
	}
	return spriteActionNonPCHurt
}

func deathActionFamilyForActor(actor world.Actor) int {
	if actorIsMercenary(actor) {
		if mercenaryUsesArcherActions(actor.Job) {
			return spriteActionNonPCDeath
		}
		return spriteActionPCDeath
	}
	if res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionPCDeath
	}
	return spriteActionNonPCDeath
}

func actionHasHitReaction(action network.ActorActionNotify) bool {
	if action.Action == 4 || action.Action == 9 || action.Action == 11 {
		return false
	}
	return actionDealsDamage(action)
}

func actionDealsDamage(action network.ActorActionNotify) bool {
	return action.Damage > 0 || action.LeftDamage > 0
}

func (m *WorldMode) applyAttackFailureForDistance(ctx client.Context, failure network.AttackFailureForDistance) {
	attackRange := maxInt(1, failure.AttackRange)
	if ctx.Session != nil {
		ctx.Session.AttackRange = attackRange
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	glog.Debugf("attack distance failure target=%d server_player=%d,%d server_target=%d,%d range=%d client_player=%d,%d", failure.TargetID, failure.SourceX, failure.SourceY, failure.TargetX, failure.TargetY, attackRange, playerX, playerY)
	ctx.World.SetPlayerPosition(failure.SourceX, failure.SourceY, ctx.World.Player.Dir)
	if actor, ok := ctx.World.Actors[failure.TargetID]; ok {
		actor.X = failure.TargetX
		actor.Y = failure.TargetY
		actor.Moving = false
		actor.FromX = failure.TargetX
		actor.FromY = failure.TargetY
		actor.ToX = failure.TargetX
		actor.ToY = failure.TargetY
		actor.MovePath = nil
		upsertActor(ctx, actor)
	}
	if m.lockedAttackID != failure.TargetID && m.pendingAttack.targetID != failure.TargetID {
		return
	}
	m.pendingAttack = attackIntent{}
	m.lastAttackAt = time.Now()
	if !normalAttackTargetWithinRange(failure.SourceX, failure.SourceY, failure.TargetX, failure.TargetY, attackRange) {
		if actor, ok := ctx.World.Actors[failure.TargetID]; ok {
			m.requestAttack(ctx, actor, "attack failure")
		}
	}
}

func (m *WorldMode) applyRecovery(ctx client.Context, recovery network.Recovery) {
	if ctx.Session == nil || recovery.Amount <= 0 {
		return
	}
	visual, ok := statusVisualEffects[recovery.StatusID]
	if !ok || visual.recover == nil {
		return
	}
	if visual.recover(ctx.Session, recovery.Amount) {
		m.addLocalRecoveryFloater(ctx, recovery.Amount, visual.recoveryColor, visual.recoveryKind)
		if visual.clearsDeath {
			m.clearLocalDeathStateIfAlive(ctx)
		}
		m.scheduleSound(time.Now(), visual.sfxCandidates()...)
	}
	glog.Debugf("recovery status=%d amount=%d hp=%d/%d sp=%d/%d", recovery.StatusID, recovery.Amount, ctx.Session.Vitals.HP, ctx.Session.Vitals.MaxHP, ctx.Session.Vitals.SP, ctx.Session.Vitals.MaxSP)
}

func (m *WorldMode) addLocalRecoveryFloater(ctx client.Context, amount int, floaterColor color.RGBA, kind damageFloaterKind) {
	if ctx.World == nil || amount <= 0 {
		return
	}
	now := time.Now()
	actorID := uint32(0)
	if ctx.Session != nil {
		actorID = ctx.Session.AccountID
		if actorID == 0 {
			actorID = ctx.Session.CharID
		}
	}
	m.damageFloaters = append(m.damageFloaters, damageFloater{
		actorID:  actorID,
		x:        ctx.World.Player.X,
		y:        ctx.World.Player.Y,
		text:     fmt.Sprintf("%d", amount),
		color:    floaterColor,
		kind:     kind,
		starts:   now,
		expires:  now.Add(damageFloaterDuration(kind)),
		duration: damageFloaterDuration(kind),
	})
}

func actionDamageFloater(action network.ActorActionNotify, targetLocal, sourceLocal bool) (string, damageFloaterKind, color.RGBA) {
	total := action.Damage + action.LeftDamage
	if total > 0 {
		if action.Action == 10 || action.Action == 13 {
			return strconv.Itoa(int(total)), damageFloaterCritical, damageFloaterYellow
		}
		if targetLocal && !sourceLocal {
			return strconv.Itoa(int(total)), damageFloaterIncoming, damageFloaterRed
		}
		return strconv.Itoa(int(total)), damageFloaterNormal, damageFloaterWhite
	}
	if action.Action == 11 {
		return "miss", damageFloaterMiss, damageFloaterWhite
	}
	if action.Action == 0 || action.Action == 7 || (action.SkillID > 0 && total == 0) {
		return "miss", damageFloaterMiss, damageFloaterWhite
	}
	return "", damageFloaterNormal, color.RGBA{}
}

func damageFloaterDuration(kind damageFloaterKind) time.Duration {
	switch kind {
	case damageFloaterMiss:
		return 800 * time.Millisecond
	case damageFloaterCombo:
		return 3000 * time.Millisecond
	default:
		return 1500 * time.Millisecond
	}
}

func damageFloaterComboTransientDuration() time.Duration {
	const robrowserComboTransient = 450 * time.Millisecond
	if multiHitDelay < robrowserComboTransient {
		return multiHitDelay
	}
	return robrowserComboTransient
}

func currentNormalAttackRange(ctx client.Context) int {
	attackRange := 1
	if ctx.Session != nil {
		attackRange = maxInt(attackRange, ctx.Session.AttackRange)
		attackRange = maxInt(attackRange, normalAttackRangeFromEquippedItems(ctx.Session, ctx.Resources))
	}
	return maxInt(1, attackRange)
}

func attackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange int) bool {
	return maxInt(absInt(playerX-targetX), absInt(playerY-targetY)) <= maxInt(1, attackRange)
}

func normalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange int) bool {
	attackRange = maxInt(1, attackRange)
	if attackRange <= 1 {
		return attackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange)
	}
	dx := absInt(playerX - targetX)
	dy := absInt(playerY - targetY)
	return dx*dx+dy*dy <= attackRange*attackRange
}

func attackApproachCell(ctx client.Context, actor world.Actor, attackRange int) (int, int, bool) {
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	return attackApproachCellFrom(ctx, playerX, playerY, actor, attackRange)
}

func attackApproachCellFrom(ctx client.Context, sourceX, sourceY int, actor world.Actor, attackRange int) (int, int, bool) {
	targetX, targetY := actorCurrentCell(actor, time.Now())
	return normalAttackApproachCellFromTarget(ctx, sourceX, sourceY, targetX, targetY, attackRange)
}

func normalAttackApproachCellFromTarget(ctx client.Context, sourceX, sourceY, targetX, targetY int, attackRange int) (int, int, bool) {
	attackRange = maxInt(1, attackRange)
	if attackRange > 1 {
		if x, y, ok := rangedAttackApproachCellFromTargetMatching(ctx, sourceX, sourceY, targetX, targetY, attackRange, normalAttackTargetWithinRange); ok {
			return x, y, true
		}
	}
	return attackApproachCellFromTarget(ctx, sourceX, sourceY, targetX, targetY, 1)
}

func attackApproachCellFromTarget(ctx client.Context, sourceX, sourceY, targetX, targetY int, attackRange int) (int, int, bool) {
	attackRange = maxInt(1, attackRange)
	if attackRange > 1 {
		if x, y, ok := rangedAttackApproachCellFromTarget(ctx, sourceX, sourceY, targetX, targetY, attackRange); ok {
			return x, y, true
		}
	}
	bestX, bestY := 0, 0
	bestDistance := math.Inf(1)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			x := targetX + dx
			y := targetY + dy
			if !walkTargetInBounds(ctx, x, y) {
				continue
			}
			if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			if !walkTargetReachable(ctx, sourceX, sourceY, x, y) {
				continue
			}
			distance := math.Hypot(float64(x-sourceX), float64(y-sourceY))
			if distance < bestDistance {
				bestDistance = distance
				bestX = x
				bestY = y
			}
		}
	}
	return bestX, bestY, bestDistance < math.Inf(1)
}

type cellRangePredicate func(sourceX, sourceY, targetX, targetY, attackRange int) bool

func rangedAttackApproachCellFromTarget(ctx client.Context, sourceX, sourceY, targetX, targetY int, attackRange int) (int, int, bool) {
	return rangedAttackApproachCellFromTargetMatching(ctx, sourceX, sourceY, targetX, targetY, attackRange, attackTargetWithinRange)
}

func rangedAttackApproachCellFromTargetMatching(ctx client.Context, sourceX, sourceY, targetX, targetY int, attackRange int, inRange cellRangePredicate) (int, int, bool) {
	if ctx.World == nil {
		return 0, 0, false
	}
	attackRange = maxInt(1, attackRange)
	stepX := approachSign(targetX - sourceX)
	stepY := approachSign(targetY - sourceY)
	preferredX := targetX - stepX*attackRange
	preferredY := targetY - stepY*attackRange
	type candidate struct {
		x                 int
		y                 int
		sourceDistance    int
		preferredDistance int
	}
	candidates := make([]candidate, 0, (attackRange*2+1)*(attackRange*2+1))
	for dy := -attackRange; dy <= attackRange; dy++ {
		for dx := -attackRange; dx <= attackRange; dx++ {
			ringDistance := maxInt(absInt(dx), absInt(dy))
			if ringDistance == 0 || ringDistance > attackRange {
				continue
			}
			x := targetX + dx
			y := targetY + dy
			if inRange != nil && !inRange(x, y, targetX, targetY, attackRange) {
				continue
			}
			if !walkTargetInBounds(ctx, x, y) {
				continue
			}
			if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			candidates = append(candidates, candidate{
				x:                 x,
				y:                 y,
				sourceDistance:    maxInt(absInt(x-sourceX), absInt(y-sourceY)),
				preferredDistance: maxInt(absInt(x-preferredX), absInt(y-preferredY)),
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.sourceDistance != right.sourceDistance {
			return left.sourceDistance < right.sourceDistance
		}
		if left.preferredDistance != right.preferredDistance {
			return left.preferredDistance < right.preferredDistance
		}
		if left.y != right.y {
			return left.y < right.y
		}
		return left.x < right.x
	})
	for _, candidate := range candidates {
		if !walkTargetReachable(ctx, sourceX, sourceY, candidate.x, candidate.y) {
			continue
		}
		return candidate.x, candidate.y, true
	}
	return 0, 0, false
}

func approachSign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}

func movingActorDestinationWithinRange(actor world.Actor, targetX, targetY, attackRange int, inRange cellRangePredicate) bool {
	if !actor.Moving || inRange == nil {
		return false
	}
	return inRange(actor.X, actor.Y, targetX, targetY, attackRange)
}

func walkTargetReachable(ctx client.Context, sourceX, sourceY, targetX, targetY int) bool {
	if ctx.World == nil || ctx.World.GAT == nil {
		return true
	}
	if !ctx.World.GAT.InBounds(sourceX, sourceY) || !ctx.World.GAT.InBounds(targetX, targetY) {
		return false
	}
	if !ctx.World.GAT.Walkable(targetX, targetY) {
		return false
	}
	if sourceX == targetX && sourceY == targetY {
		return true
	}
	_, ok := findWalkPath(ctx.World.GAT, sourceX, sourceY, targetX, targetY)
	return ok
}

func (m *WorldMode) drawDamageFloaters(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
	if len(m.damageFloaters) == 0 {
		return
	}
	active := m.damageFloaters[:0]
	for _, floater := range m.damageFloaters {
		if now.After(floater.expires) {
			continue
		}
		active = append(active, floater)
		if now.Before(floater.starts) {
			continue
		}
		if ctx.World != nil && ctx.World.MapProperty.IsSiege() && damageFloaterHiddenInSiege(floater.kind) {
			continue
		}
		x, y := float64(floater.x), float64(floater.y)
		if actor, ok := ctx.World.Actors[floater.actorID]; ok {
			x, y = actorRenderPosition(actor, now)
		} else if isLocalActor(ctx, floater.actorID) {
			x, y = actorRenderPosition(ctx.World.Player, now)
		}
		progress := damageFloaterProgress(floater, now)
		dx, dy, zLift, scale, alpha := damageFloaterPlacement(floater.kind, progress)
		floaterColor := damageFloaterColor(floater.kind, floater.color)
		terrainZ := terrainHeightAt(ctx.World, x, y)
		worldX := cellCenter(x) + dx
		worldY := cellCenter(y) + dy
		screenScale := actorBillboardScreenScale(projection, worldX, worldY, terrainZ)
		if floater.kind == damageFloaterMiss {
			if billboard, ok := m.damageMessageBillboard(ctx, 0, 0); ok {
				drawSpriteBillboardTintAlphaOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift, screenScale*scale, alpha, 1, floaterColor)
				continue
			}
		}
		if floater.kind == damageFloaterCritical {
			if billboard, ok := m.damageMessageBillboard(ctx, 2, 0); ok {
				drawSpriteBillboardTintAlphaOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift+0.05, screenScale*scale*0.6, alpha, 1, color.RGBA{R: 168, G: 168, B: 168, A: 255})
			}
		}
		if billboard, ok := m.damageNumberBillboard(ctx, floater.text); ok {
			drawSpriteBillboardTintAlphaOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift, screenScale*scale, alpha, 1, floaterColor)
			continue
		}
		point := projection.Project(worldX, worldY, terrainZ+zLift)
		render.DrawBitmapTextAtColor(screen, floater.text, int(point.x)-8, int(point.y)-40, withAlpha(floaterColor, alpha))
	}
	m.damageFloaters = active
}

func (m *WorldMode) startActorDeath(ctx client.Context, id uint32) {
	actor, ok, local := actorForCombatID(ctx, id)
	if !ok {
		if !local {
			m.removeLevel99AuraEffects(id)
			ctx.World.RemoveActor(id)
		}
		return
	}
	now := time.Now()
	deathX, deathY := actorRenderPosition(actor, now)
	actor.X = int(math.Round(deathX))
	actor.Y = int(math.Round(deathY))
	actor.Moving = false
	actor.FromX = actor.X
	actor.FromY = actor.Y
	actor.ToX = actor.X
	actor.ToY = actor.Y
	actor.MovePath = nil
	actor.HasMoveStart = false
	actor.MoveStartX = 0
	actor.MoveStartY = 0
	actor.WalkDistance = 0
	if local {
		ctx.World.Player.Moving = false
		ctx.World.Player.X = actor.X
		ctx.World.Player.Y = actor.Y
		ctx.World.Player.FromX = actor.X
		ctx.World.Player.FromY = actor.Y
		ctx.World.Player.ToX = actor.X
		ctx.World.Player.ToY = actor.Y
		ctx.World.Player.MovePath = nil
		ctx.World.Player.HasMoveStart = false
		ctx.World.Player.MoveStartX = 0
		ctx.World.Player.MoveStartY = 0
		ctx.World.Player.WalkDistance = 0
		if ctx.Session != nil {
			ctx.Session.Dead = true
		}
		m.cancelAttackIntent()
		m.pendingPickup = pickupIntent{}
		m.pendingSkill = pendingSkillTarget{}
		m.ui.petContext.Close()
		m.ui.homunculusContext.Close()
		m.ui.mercenaryContext.Close()
		m.ui.playerContext.Close()
		m.ui.escapeMenu.OpenDeath(ctx)
	} else {
		upsertActor(ctx, actor)
	}
	actionFamily := deathActionFamilyForActor(actor)
	deathDuration := m.actorActionDuration(ctx, actor, actionFamily, defaultDeathAnimationDuration)
	persistent := !local && actorRepresentsPlayer(actor)
	visibleDuration := deathDuration
	if !local && !persistent {
		visibleDuration = maxDuration(deathDuration, nonPCDeathFadeDuration)
	}
	if local || persistent {
		m.startHeldCombatAnimation(ctx, id, actionFamily, now, deathDuration)
	} else {
		m.startCombatAnimation(ctx, id, actionFamily, now, visibleDuration)
	}
	m.setActorAIMotion(ctx, id, aiMotionDead, 0, time.Time{})
	if !local {
		if m.actorDeaths == nil {
			m.actorDeaths = make(map[uint32]time.Time)
		}
		if persistent {
			m.actorDeaths[id] = time.Time{}
		} else {
			m.actorDeaths[id] = now.Add(visibleDuration)
		}
		if m.scriptHighlight.id == id {
			m.clearScriptHighlight()
		}
		if m.actorLife != nil {
			if life, ok := m.actorLife[id]; ok {
				life.hp = 0
				life.updatedAt = now
				m.actorLife[id] = life
			}
		}
	}
	if !local && actor.HasObjectType && actor.ObjectType != actorObjectTypePC {
		m.removeLevel99AuraEffects(id)
	}
	glog.Debugf("actor death id=%d job=%d local=%t persistent=%t action=%d death_ms=%d remove_ms=%d", id, actor.Job, local, persistent, actionFamily, deathDuration.Milliseconds(), visibleDuration.Milliseconds())
}

func (m *WorldMode) applyActorResurrection(ctx client.Context, resurrection network.ActorResurrection) {
	if resurrection.ID == 0 {
		return
	}
	if isLocalActor(ctx, resurrection.ID) {
		m.clearLocalDeathState(ctx)
	} else {
		m.clearActorDeath(resurrection.ID)
	}
	m.setActorAIMotion(ctx, resurrection.ID, aiMotionStand, 0, time.Time{})
	glog.Debugf("actor resurrected id=%d type=%d", resurrection.ID, resurrection.Type)
}

func (m *WorldMode) clearActorDeath(id uint32) {
	delete(m.actorDeaths, id)
	delete(m.actorAnims, id)
	delete(m.actorSoundFrames, id)
}

func (m *WorldMode) actorDeathAlpha(id uint32, now time.Time) float64 {
	removeAt, ok := m.actorDeaths[id]
	if !ok {
		return 1
	}
	if removeAt.IsZero() {
		return 1
	}
	started := now
	if anim, ok := m.actorAnims[id]; ok && !anim.started.IsZero() {
		started = anim.started
	}
	total := removeAt.Sub(started)

	if total <= 0 {
		return 0
	}
	elapsed := now.Sub(started)
	if elapsed <= 0 {
		return 1
	}
	alpha := 1 - float64(elapsed)/float64(total)
	if alpha < 0 {
		return 0
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}

func (m *WorldMode) actorVisualAlpha(id uint32, now time.Time) float64 {
	return math.Min(m.actorDeathAlpha(id, now), m.actorVanishAlpha(id, now))
}

func (m *WorldMode) actorVanishAlpha(id uint32, now time.Time) float64 {
	fade, ok := m.actorVanishes[id]
	if !ok {
		return 1
	}
	total := fade.removeAt.Sub(fade.started)
	if total <= 0 {
		return 0
	}
	elapsed := now.Sub(fade.started)
	if elapsed <= 0 {
		return 1
	}
	alpha := 1 - float64(elapsed)/float64(total)
	if alpha < 0 {
		return 0
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}

func (m *WorldMode) drawAttackFocusMarker(screen *render.Frame, ctx client.Context, now time.Time, entries []sceneActorDrawEntry) {
	m.drawActorFocusMarker(screen, ctx, now, entries, m.attackFocusID, &m.attackFocusStart, false)
}

func (m *WorldMode) drawScriptHighlightMarker(screen *render.Frame, ctx client.Context, now time.Time, entries []sceneActorDrawEntry) {
	if actorIDsMatch(ctx, m.scriptHighlight.id, m.attackFocusID) {
		return
	}
	m.drawActorFocusMarker(screen, ctx, now, entries, m.scriptHighlight.id, &m.scriptHighlight.started, true)
}

func (m *WorldMode) drawActorFocusMarker(screen *render.Frame, ctx client.Context, now time.Time, entries []sceneActorDrawEntry, targetID uint32, started *time.Time, matchLocalAlias bool) {
	if targetID == 0 || started == nil || screen == nil {
		return
	}
	if started.IsZero() {
		*started = now
	}
	action := cursorActionLock
	state := m.loadedCursorState(ctx)
	frame, ok := state.frameAt(action, cursorInfo(action), *started, now)
	if !ok {
		return
	}
	for _, entry := range entries {
		if entry.actor.ID != targetID && !(matchLocalAlias && entry.isPlayer && isLocalActor(ctx, targetID)) {
			continue
		}
		x, y := actorPickBoundsCenter(entry.screenX, entry.screenY, entry.scale)
		var opts render.DrawImageOptions
		opts.GeoM.Translate(math.Round(x-frame.anchorX), math.Round(y-frame.anchorY))
		opts.Filter = spriteDrawFilter()
		screen.DrawImage(frame.image, &opts)
		return
	}
}

func actorIDsMatch(ctx client.Context, a, b uint32) bool {
	if a == 0 || b == 0 {
		return false
	}
	return a == b || (isLocalActor(ctx, a) && isLocalActor(ctx, b))
}

func (m *WorldMode) clearScriptHighlight() {
	if m != nil {
		m.scriptHighlight = actorHighlight{}
	}
}

func actorCastBarProgress(bar actorCastBar, now time.Time) (float64, bool) {
	if bar.duration <= 0 || bar.started.IsZero() {
		return 0, false
	}
	progress := float64(now.Sub(bar.started)) / float64(bar.duration)
	if progress < 0 {
		progress = 0
	}
	if progress >= 1 {
		return 1, false
	}
	return progress, true
}

func actorLifeBarHeight(life actorLife) float64 {
	height := actorOverlayBarHeight
	if life.hasSP {
		height += actorOverlayBarRowSpacing
	}
	if life.hasHunger {
		height += actorOverlayBarRowSpacing
	}
	return height
}

func actorNameBelowLifeBarY(baseY, scale float64, life actorLife) float64 {
	return actorLifeBarY(baseY, scale) + actorLifeBarHeight(life) + 3
}

const (
	actorOverlayBarWidth      = 60.0
	actorOverlayBarRowHeight  = 3.0
	actorOverlayBarRowSpacing = 4.0
	actorOverlayBarHeight     = actorOverlayBarRowHeight + 2
)

func actorOverlayBarX(centerX float64) float64 {
	return math.Round(centerX - actorOverlayBarWidth/2)
}

func actorOverlayBarY(y float64) float64 {
	return math.Round(y)
}

func actorOverlayBarFillWidth(ratio float64) float64 {
	if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return math.Round((actorOverlayBarWidth - 2) * ratio)
}

func (m *WorldMode) drawActorLifeBar(screen *render.Frame, ctx client.Context, entry sceneActorDrawEntry) {
	life, ok := m.actorLifeForDisplay(ctx, entry.actor)
	if !ok {
		return
	}
	ratio := actorLifeRatio(life.hp, life.maxHP)
	height := actorLifeBarHeight(life)
	x := actorOverlayBarX(entry.screenX)
	y := actorOverlayBarY(actorLifeBarY(entry.screenY, entry.scale))
	fillWidth := actorOverlayBarFillWidth(ratio)
	fill := color.RGBA{R: 255, G: 0, B: 231, A: 255}
	if life.player || life.friendly {
		fill = ui.PlayerHPBarColor
		if ratio < 0.25 {
			fill = color.RGBA{R: 255, G: 0, B: 0, A: 255}
		}
	} else if ratio < 0.25 {
		fill = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	}
	render.DrawRect(screen, x, y, actorOverlayBarWidth, height, color.RGBA{R: 16, G: 24, B: 156, A: 255})
	render.DrawRect(screen, x+1, y+1, actorOverlayBarWidth-2, height-2, color.RGBA{R: 66, G: 66, B: 66, A: 255})
	if fillWidth > 0 {
		render.DrawRect(screen, x+1, y+1, fillWidth, actorOverlayBarRowHeight, fill)
	}
	rowY := y + 1
	if life.hasSP {
		rowY += actorOverlayBarRowSpacing
		spRatio := actorLifeRatio(life.sp, life.maxSP)
		render.DrawRect(screen, x, rowY-1, actorOverlayBarWidth, 1, color.RGBA{R: 16, G: 24, B: 156, A: 255})
		if spWidth := actorOverlayBarFillWidth(spRatio); spWidth > 0 {
			render.DrawRect(screen, x+1, rowY, spWidth, actorOverlayBarRowHeight, ui.PlayerSPBarColor)
		}
	}
	if life.hasHunger {
		rowY += actorOverlayBarRowSpacing
		hungerRatio := actorLifeRatio(life.hunger, life.maxHunger)
		hungerFill := ui.HungerBarFillColor(life.hunger, life.maxHunger)
		render.DrawRect(screen, x+1, rowY-1, actorOverlayBarWidth-2, 1, color.RGBA{R: 66, G: 66, B: 66, A: 255})
		if hungerWidth := actorOverlayBarFillWidth(hungerRatio); hungerWidth > 0 {
			render.DrawRect(screen, x+1, rowY, hungerWidth, actorOverlayBarRowHeight, hungerFill)
		}
	}
}

func actorLifeRatio(value, maxValue int) float64 {
	if maxValue <= 0 {
		return 0
	}
	ratio := float64(value) / float64(maxValue)
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func (m *WorldMode) drawActorCastBar(screen *render.Frame, entry sceneActorDrawEntry, now time.Time) {
	if entry.actor.ID == 0 || m.actorCastBars == nil {
		return
	}
	bar, ok := m.actorCastBars[entry.actor.ID]
	if !ok {
		return
	}
	ratio, active := actorCastBarProgress(bar, now)
	if !active {
		delete(m.actorCastBars, entry.actor.ID)
		return
	}
	x := actorOverlayBarX(entry.screenX)
	y := actorOverlayBarY(actorCastBarY(entry.screenY, entry.scale))
	fillWidth := actorOverlayBarFillWidth(ratio)
	render.DrawRect(screen, x, y, actorOverlayBarWidth, actorOverlayBarHeight, color.RGBA{R: 16, G: 24, B: 156, A: 255})
	render.DrawRect(screen, x+1, y+1, actorOverlayBarWidth-2, actorOverlayBarHeight-2, color.RGBA{R: 66, G: 66, B: 66, A: 255})
	if fillWidth > 0 {
		fill := bar.color
		if fill.A == 0 {
			fill = color.RGBA{R: 0, G: 255, B: 0, A: 255}
		}
		render.DrawRect(screen, x+1, y+1, fillWidth, actorOverlayBarRowHeight, fill)
	}
}

func (m *WorldMode) actorLifeForDisplay(ctx client.Context, actor world.Actor) (actorLife, bool) {
	if actor.ID == 0 {
		return actorLife{}, false
	}
	if specialNPCVisualForActor(ctx, actor) != specialNPCVisualNone {
		return actorLife{}, false
	}
	if isLocalActor(ctx, actor.ID) {
		return localPlayerLifeForDisplay(ctx)
	}
	if life, ok := partyMemberLifeForDisplay(ctx, actor); ok {
		return life, true
	}
	if life, ok := m.homunculusLifeForDisplay(ctx, actor); ok {
		return life, true
	}
	if life, ok := m.mercenaryLifeForDisplay(ctx, actor); ok {
		return life, true
	}
	// Monster HP bars are a 2012+ client feature. The 2008 client exposes
	// monster HP through WZ_ESTIMATION/Sense instead, so keep the combat HP
	// cache hidden from the normal actor overlay.
	return actorLife{}, false
}

func (m *WorldMode) homunculusLifeForDisplay(ctx client.Context, actor world.Actor) (actorLife, bool) {
	if ctx.Session == nil || actor.ID == 0 {
		return actorLife{}, false
	}
	hom := ctx.Session.Homunculus
	if !hom.Active {
		return actorLife{}, false
	}
	if hom.ID != 0 {
		if actor.ID != hom.ID {
			return actorLife{}, false
		}
	} else if !actor.HasObjectType || actor.ObjectType != actorObjectTypeHomunculus {
		return actorLife{}, false
	}

	life := actorLife{
		hp:       hom.HP,
		maxHP:    hom.MaxHP,
		sp:       hom.SP,
		maxSP:    hom.MaxSP,
		hasSP:    hom.MaxSP > 0,
		friendly: true,
	}
	if m.actorLife != nil {
		if cached, ok := m.actorLife[actor.ID]; ok {
			if cached.maxHP > 0 {
				life.hp = cached.hp
				life.maxHP = cached.maxHP
			}
			if cached.hasSP && cached.maxSP > 0 {
				life.sp = cached.sp
				life.maxSP = cached.maxSP
				life.hasSP = true
			}
			if cached.hasHunger && cached.maxHunger > 0 {
				life.hunger = cached.hunger
				life.maxHunger = cached.maxHunger
				life.hasHunger = true
			}
		}
	}
	if !life.hasHunger && hom.Hunger > 0 {
		life.hunger = hom.Hunger
		life.maxHunger = homunculusHungerMax
		life.hasHunger = true
	}
	if life.maxHP <= 0 || life.hp < 0 {
		return actorLife{}, false
	}
	life.hp = clampGameInt(life.hp, 0, life.maxHP)
	if life.hasSP {
		life.sp = clampGameInt(life.sp, 0, life.maxSP)
	}
	if life.hasHunger {
		life.hunger = clampGameInt(life.hunger, 0, life.maxHunger)
	}
	return life, true
}

func (m *WorldMode) mercenaryLifeForDisplay(ctx client.Context, actor world.Actor) (actorLife, bool) {
	if ctx.Session == nil || actor.ID == 0 {
		return actorLife{}, false
	}
	merc := ctx.Session.Mercenary
	if !merc.Active {
		return actorLife{}, false
	}
	if merc.ID != 0 {
		if actor.ID != merc.ID {
			return actorLife{}, false
		}
	} else if !actor.HasObjectType || actor.ObjectType != actorObjectTypeMercenary {
		return actorLife{}, false
	}

	life := actorLife{
		hp:       merc.HP,
		maxHP:    merc.MaxHP,
		sp:       merc.SP,
		maxSP:    merc.MaxSP,
		hasSP:    merc.MaxSP > 0,
		friendly: true,
	}
	if m.actorLife != nil {
		if cached, ok := m.actorLife[actor.ID]; ok {
			if cached.maxHP > 0 {
				life.hp = cached.hp
				life.maxHP = cached.maxHP
			}
			if cached.hasSP && cached.maxSP > 0 {
				life.sp = cached.sp
				life.maxSP = cached.maxSP
				life.hasSP = true
			}
		}
	}
	if life.maxHP <= 0 || life.hp < 0 {
		return actorLife{}, false
	}
	life.hp = clampGameInt(life.hp, 0, life.maxHP)
	if life.hasSP {
		life.sp = clampGameInt(life.sp, 0, life.maxSP)
	}
	return life, true
}

func (m *WorldMode) monsterLifeForSense(actorID uint32) (actorLife, bool) {
	if m.actorLife == nil {
		return actorLife{}, false
	}
	life, ok := m.actorLife[actorID]
	if !ok || life.maxHP <= 0 || life.hp < 0 {
		return actorLife{}, false
	}
	return life, true
}

func localPlayerLifeForDisplay(ctx client.Context) (actorLife, bool) {
	if ctx.Session == nil {
		return actorLife{}, false
	}
	hp := ctx.Session.Vitals.HP
	maxHP := ctx.Session.Vitals.MaxHP
	sp := ctx.Session.Vitals.SP
	maxSP := ctx.Session.Vitals.MaxSP
	if maxHP <= 0 {
		character := ctx.Session.SelectedCharacter()
		hp = int(character.HP)
		maxHP = int(character.MaxHP)
	}
	if maxSP <= 0 {
		character := ctx.Session.SelectedCharacter()
		sp = int(character.SP)
		maxSP = int(character.MaxSP)
	}
	if maxHP <= 0 {
		return actorLife{}, false
	}
	return actorLife{
		hp:     clampGameInt(hp, 0, maxHP),
		maxHP:  maxHP,
		sp:     clampGameInt(sp, 0, maxSP),
		maxSP:  maxSP,
		hasSP:  maxSP > 0,
		player: true,
	}, true
}
