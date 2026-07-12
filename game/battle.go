package game

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type attackIntent struct {
	targetID uint32
	expires  time.Time
	readyAt  time.Time
}

type damageFloater struct {
	actorID uint32
	x       int
	y       int
	text    string
	color   color.RGBA
	kind    damageFloaterKind
	starts  time.Time
	expires time.Time
}

type damageFloaterKind int

const (
	damageFloaterNormal damageFloaterKind = iota
	damageFloaterCritical
	damageFloaterIncoming
	damageFloaterRecoveryHP
	damageFloaterRecoverySP
	damageFloaterMiss
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
	if duration <= 0 {
		return 0
	}
	motion := firstActionSoundMotion(action)
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

func (m *WorldMode) nonPCResolvedAction(ctx client.Context, actor worldstate.Actor, actionFamily int) (res.ACTAction, bool) {
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return res.ACTAction{}, false
	}
	_, action, ok := resolveSpriteAction(view.act, actionFamily, actor.Dir)
	return action, ok
}

func (m *WorldMode) actorResolvedAction(ctx client.Context, actor worldstate.Actor, actionFamily int) (res.ACTAction, bool) {
	if res.HasPlayerJobToken(int(actor.Job)) {
		view := m.humanoidSpriteViewForActor(ctx, actor)
		if view == nil || view.body == nil {
			return res.ACTAction{}, false
		}
		_, action, ok := resolveSpriteAction(view.body.act, actionFamily, actor.Dir)
		return action, ok
	}
	return m.nonPCResolvedAction(ctx, actor, actionFamily)
}

func (m *WorldMode) actorActionFrameDelay(ctx client.Context, actor worldstate.Actor, actionFamily int, duration time.Duration) time.Duration {
	if duration <= 0 {
		return 0
	}
	action, ok := m.actorResolvedAction(ctx, actor, actionFamily)
	if !ok || len(action.Animations) == 0 {
		return 0
	}
	return duration / time.Duration(len(action.Animations))
}

func (m *WorldMode) nonPCActionACT(ctx client.Context, actor worldstate.Actor) *res.ACT {
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return nil
	}
	return view.act
}

func (m *WorldMode) actorActionACT(ctx client.Context, actor worldstate.Actor) *res.ACT {
	if res.HasPlayerJobToken(int(actor.Job)) {
		view := m.humanoidSpriteViewForActor(ctx, actor)
		if view == nil || view.body == nil {
			return nil
		}
		return view.body.act
	}
	return m.nonPCActionACT(ctx, actor)
}

func (m *WorldMode) actorActionDuration(ctx client.Context, actor worldstate.Actor, actionFamily int, fallback time.Duration) time.Duration {
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
	updatedAt time.Time
}

type actorCastBar struct {
	started  time.Time
	duration time.Duration
	color    color.RGBA
}

const attackRetryInterval = 1200 * time.Millisecond

const (
	defaultAttackAnimationDuration = 600 * time.Millisecond
	defaultHitAnimationDuration    = 250 * time.Millisecond
	defaultDeathAnimationDuration  = 900 * time.Millisecond
	maxCombatAnimationDuration     = 5 * time.Second
	nonPCDeathFadeDuration         = 5 * time.Second
	multiHitDelay                  = 200 * time.Millisecond
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

func (m *WorldMode) requestAttack(ctx client.Context, actor worldstate.Actor, source string) {
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
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	if attackTargetWithinRange(playerX, playerY, actor.X, actor.Y, attackRange) {
		m.sendAttackAction(ctx, actor, source)
		return
	}
	targetX, targetY, ok := attackApproachCell(ctx, actor, attackRange)
	if !ok {
		log.Printf("%s attack chase blocked target=%d player=%d,%d target=%d,%d range=%d", source, actor.ID, playerX, playerY, actor.X, actor.Y, attackRange)
		m.setWalkCooldown(walkRequestCooldown)
		return
	}
	m.pendingAttack = attackIntent{
		targetID: actor.ID,
		expires:  time.Now().Add(8 * time.Second),
	}
	log.Printf("%s attack chase target=%d player=%d,%d target=%d,%d range=%d chase=%d,%d", source, actor.ID, playerX, playerY, actor.X, actor.Y, attackRange, targetX, targetY)
	m.requestWalk(ctx, targetX, targetY, source+" attack chase")
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
		log.Printf("%s pending attack expired target=%d", source, m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	actor, ok := ctx.World.Actors[m.pendingAttack.targetID]
	if !ok {
		log.Printf("%s pending attack target vanished id=%d", source, m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := currentPlayerCell(ctx, now)
	if !attackTargetWithinRange(playerX, playerY, actor.X, actor.Y, attackRange) {
		if logOutOfRange {
			log.Printf("%s pending attack still out of range target=%d player=%d,%d target=%d,%d range=%d", source, actor.ID, playerX, playerY, actor.X, actor.Y, attackRange)
		}
		return
	}
	readyAt := pendingAttackReadyAt(ctx.World.Player, now)
	if m.pendingAttack.readyAt.IsZero() {
		m.pendingAttack.readyAt = readyAt
	}
	if logOutOfRange {
		log.Printf("%s pending attack scheduled target=%d delay_ms=%d", source, actor.ID, maxInt(0, int(m.pendingAttack.readyAt.Sub(now).Milliseconds())))
	}
}

func (m *WorldMode) processPendingAttack(ctx client.Context) {
	if m.pendingAttack.targetID == 0 || m.pendingAttack.readyAt.IsZero() {
		return
	}
	now := time.Now()
	if now.After(m.pendingAttack.expires) {
		log.Printf("pending attack expired target=%d", m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	if now.Before(m.pendingAttack.readyAt) {
		return
	}
	actor, ok := ctx.World.Actors[m.pendingAttack.targetID]
	if !ok {
		log.Printf("pending attack target vanished id=%d", m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := currentPlayerCell(ctx, now)
	if !attackTargetWithinRange(playerX, playerY, actor.X, actor.Y, attackRange) {
		log.Printf("pending attack became out of range target=%d player=%d,%d target=%d,%d range=%d", actor.ID, playerX, playerY, actor.X, actor.Y, attackRange)
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
	if ctx.World.Player.IsMovingAt(now) {
		return
	}
	actor, ok := ctx.World.Actors[m.lockedAttackID]
	if !ok {
		log.Printf("locked attack target vanished id=%d", m.lockedAttackID)
		m.clearLockedAttack()
		return
	}
	if !actorCanBeAttackClicked(ctx, actor) {
		log.Printf("locked attack target no longer attackable id=%d object_type=%d", actor.ID, actor.ObjectType)
		m.clearLockedAttack()
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := currentPlayerCell(ctx, now)
	if attackTargetWithinRange(playerX, playerY, actor.X, actor.Y, attackRange) {
		if !attackRetryDue(m.lastAttackAt, now) {
			return
		}
		log.Printf("locked attack retry target=%d player=%d,%d target=%d,%d range=%d", actor.ID, playerX, playerY, actor.X, actor.Y, attackRange)
		m.sendAttackAction(ctx, actor, "locked")
		return
	}
	if !attackRetryDue(m.lastChaseAt, now) {
		return
	}
	m.lastChaseAt = now
	log.Printf("locked attack chase retry target=%d player=%d,%d target=%d,%d range=%d", actor.ID, playerX, playerY, actor.X, actor.Y, attackRange)
	m.requestAttack(ctx, actor, "locked")
}

func attackRetryDue(last time.Time, now time.Time) bool {
	return last.IsZero() || now.Sub(last) >= attackRetryInterval
}

func normalAttackLockActive(ctx client.Context) bool {
	return (ctx.Input != nil && ctx.Input.Pressed(render.KeyCtrl)) || (ctx.Session != nil && ctx.Session.NoCtrl)
}

func pendingAttackReadyAt(player worldstate.Actor, now time.Time) time.Time {
	readyAt := now.Add(60 * time.Millisecond)
	if player.IsMovingAt(now) && player.MoveDuration > 0 {
		walkReadyAt := player.MoveStarted.Add(player.MoveDuration).Add(60 * time.Millisecond)
		if walkReadyAt.After(readyAt) {
			readyAt = walkReadyAt
		}
	}
	return readyAt
}

func (m *WorldMode) sendAttackAction(ctx client.Context, actor worldstate.Actor, source string) {
	if err := ctx.Network.SendActionRequest(actor.ID, network.ActionAttack); err == nil {
		m.lastAttackAt = time.Now()
		m.setWalkCooldown(walkRequestCooldown)
	} else {
		log.Printf("%s attack request failed target=%d action=%d: %v", source, actor.ID, network.ActionAttack, err)
		m.setWalkCooldown(walkErrorCooldown)
	}
}

func (m *WorldMode) applyActorActionNotify(ctx client.Context, action network.ActorActionNotify) {
	log.Printf("actor action src=%d dst=%d skill=%d level=%d damage=%d left_damage=%d hits=%d action=%d src_speed=%d dst_speed=%d tick=%d", action.SourceID, action.TargetID, action.SkillID, action.SkillLevel, action.Damage, action.LeftDamage, action.HitCount, action.Action, action.SourceSpeed, action.TargetSpeed, action.ServerTick)
	now := time.Now()
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
	if sourceOK && targetOK {
		m.faceCombatSource(ctx, source, sourceLocal, target)
		source.Dir = directionFromDelta(source.X, source.Y, target.X, target.Y, source.Dir)
	}
	attackDuration := combatDuration(action.SourceSpeed, defaultAttackAnimationDuration)
	attackFamily := spriteActionNonPCAttack
	if sourceOK {
		attackFamily = skillActionFamilyForActor(source, action.SkillID)
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
			hitDelay = combatHitDelayFromAction(actionDef, attackDuration)
			if sound := actionSoundName(m.actorActionACT(ctx, source), actionDef, firstActionSoundMotion(actionDef)); sound != "" {
				m.scheduleSound(now.Add(hitDelay), sound)
			}
		}
	}
	hitAt := now.Add(hitDelay)
	if targetOK && actionHasHitReaction(action) {
		if hitAt.Before(now) {
			hitAt = now
		}
		m.clearActorCastBar(ctx, action.TargetID)
		m.addSkillBeginEffect(ctx, action, now)
		m.addNormalAttackBeforeHitEffect(ctx, action, source, sourceOK, now)
		m.addSkillBeforeHitEffect(ctx, action, now)
		if skillTargetUsesHitReaction(action, sourceLocal, targetLocal) {
			hurtDuration := combatDuration(action.TargetSpeed, defaultHitAnimationDuration)
			m.startCombatAnimationWithNext(ctx, action.TargetID, hurtActionFamilyForActor(target), hitAt, hurtDuration, postHurtAnimation(target, hitAt.Add(hurtDuration)))
		}
		m.scheduleSound(hitAt, combatHitSFXCandidates(source, sourceOK, target, targetOK)...)
		m.addSkillEffect(ctx, action, hitAt)
		m.addSkillHitEffect(ctx, action, hitAt)
	}
	x, y := ctx.World.Player.X, ctx.World.Player.Y
	if targetOK {
		x, y = target.X, target.Y
	} else if isLocalActor(ctx, action.TargetID) {
		x, y = ctx.World.Player.X, ctx.World.Player.Y
	}
	m.addActionDamageFloaters(action, targetLocal, sourceLocal, x, y, hitAt)
}

func (m *WorldMode) addSkillBeginEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 {
		return
	}
	for _, effectID := range skillBeginEffectIDs(action.SkillID) {
		actorID := action.SourceID
		if effectDetachesLocalActor(effectID) && isLocalActor(ctx, actorID) {
			actorID = 0
		}
		if m.addWorldEffectAt(ctx, effectID, actorID, starts) {
			log.Printf("skill begin effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
}

func (m *WorldMode) addNormalAttackBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, source worldstate.Actor, sourceOK bool, starts time.Time) {
	if action.SkillID != 0 || !sourceOK || !actorUsesBow(ctx.Resources, source) {
		return
	}
	if m.addWorldEffectBetweenAt(ctx, effectArrowShot, action.TargetID, action.SourceID, starts) {
		log.Printf("normal attack before-hit effect src=%d target=%d effect=%d", action.SourceID, action.TargetID, effectArrowShot)
	}
}

func actorUsesBow(manager *res.Manager, actor worldstate.Actor) bool {
	if actor.Weapon <= 0 {
		return false
	}
	return res.PlayerWeaponViewID(manager, int(actor.Weapon)) == 11
}

func (m *WorldMode) addSkillEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 {
		return
	}
	for _, effectID := range skillEffectIDs(action.SkillID) {
		if m.addWorldEffectBetweenAt(ctx, effectID, action.TargetID, action.SourceID, starts) {
			log.Printf("skill effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
	for _, effectID := range skillEffectOnCasterIDs(action.SkillID) {
		if m.addWorldEffectAt(ctx, effectID, action.SourceID, starts) {
			log.Printf("skill caster effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
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
				log.Printf("skill before-hit effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
		for _, effectID := range selfEffectIDs {
			if m.addWorldEffectAt(ctx, effectID, action.SourceID, effectStarts) {
				log.Printf("skill before-hit self effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
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
				log.Printf("skill hit effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
		for _, effectID := range casterEffectIDs {
			if m.addWorldEffectAt(ctx, effectID, action.SourceID, effectStarts) {
				log.Printf("skill hit caster effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
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

func (m *WorldMode) addActionDamageFloaters(action network.ActorActionNotify, targetLocal, sourceLocal bool, x, y int, hitAt time.Time) {
	text, kind, floaterColor := actionDamageFloater(action, targetLocal, sourceLocal)
	if text == "" {
		return
	}
	count := actionVisualHitCount(action)
	if count <= 1 || action.Damage+action.LeftDamage <= 0 || kind == damageFloaterCritical {
		m.damageFloaters = append(m.damageFloaters, damageFloater{
			actorID: action.TargetID,
			x:       x,
			y:       y,
			text:    text,
			color:   floaterColor,
			kind:    kind,
			starts:  hitAt,
			expires: hitAt.Add(damageFloaterDuration(kind)),
		})
		return
	}
	total := int(action.Damage + action.LeftDamage)
	base := total / count
	rem := total % count
	for i := 0; i < count; i++ {
		value := base
		if i < rem {
			value++
		}
		starts := hitAt.Add(multiHitDelay * time.Duration(i))
		m.damageFloaters = append(m.damageFloaters, damageFloater{
			actorID: action.TargetID,
			x:       x,
			y:       y,
			text:    strconv.Itoa(value),
			color:   floaterColor,
			kind:    kind,
			starts:  starts,
			expires: starts.Add(damageFloaterDuration(kind)),
		})
	}
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
				ctx.World.UpsertActor(source)
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
		}
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
	ctx.World.UpsertActor(actor)
	if !sitting {
		actor = ctx.World.Actors[id]
		actor.Sitting = false
		ctx.World.Actors[id] = actor
	}
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
	log.Printf("actor hp id=%d hp=%d max_hp=%d tiny=%t", update.ID, hp, update.MaxHP, update.Tiny)
}

func actorForCombatID(ctx client.Context, id uint32) (worldstate.Actor, bool, bool) {
	if ctx.World == nil || id == 0 {
		return worldstate.Actor{}, false, false
	}
	if isLocalActor(ctx, id) {
		actor := ctx.World.Player
		character := selectedCharacter(ctx.Session)
		actor.ID = id
		actor.Job = character.Job
		actor.Head = character.Hair
		actor.Weapon = character.Weapon
		actor.Shield = character.Shield
		actor.HeadTop = character.HeadTop
		actor.HeadMid = character.HeadMid
		actor.HeadLow = character.HeadLow
		actor.Sex = ctx.Session.Sex
		actor.Appearance = true
		return actor, true, true
	}
	actor, ok := ctx.World.Actors[id]
	return actor, ok, false
}

func (m *WorldMode) faceCombatSource(ctx client.Context, source worldstate.Actor, sourceLocal bool, target worldstate.Actor) {
	dir := directionFromDelta(source.X, source.Y, target.X, target.Y, source.Dir)
	if sourceLocal {
		ctx.World.Player.Dir = dir
		ctx.World.Dir = dir
		return
	}
	source.Dir = dir
	ctx.World.UpsertActor(source)
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

func postHurtAnimation(actor worldstate.Actor, started time.Time) *actorAnimation {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return readyFightAnimation(started)
	}
	return nil
}

func (m *WorldMode) clearLocalActorAction(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	m.clearActorAction(ctx.Session.AccountID)
	m.clearActorAction(ctx.Session.CharID)
}

func (m *WorldMode) clearActorAction(id uint32) {
	if id == 0 || m.actorAnims == nil {
		return
	}
	if anim, ok := m.actorAnims[id]; ok && (anim.actionFamily == spriteActionPCDeath || anim.actionFamily == spriteActionNonPCDeath) {
		return
	}
	delete(m.actorAnims, id)
}

func (m *WorldMode) startCombatAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimation(ctx, id, actionFamily, started, duration)
	if ctx.Session == nil || !isLocalActor(ctx, id) {
		return
	}
	m.startActorAnimation(ctx, ctx.Session.AccountID, actionFamily, started, duration)
	m.startActorAnimation(ctx, ctx.Session.CharID, actionFamily, started, duration)
}

func (m *WorldMode) startCombatAnimationWithTiming(ctx client.Context, id uint32, actor worldstate.Actor, actionFamily int, started time.Time, duration time.Duration) {
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

func (m *WorldMode) startCombatAnimationWithTimingAndNext(ctx client.Context, id uint32, actor worldstate.Actor, actionFamily int, started time.Time, duration time.Duration, next *actorAnimation) {
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
	m.clearLocalDeathState(ctx)
}

func (m *WorldMode) clearLocalDeathState(ctx client.Context) {
	m.deathModal.Reset()
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

func attackActionFamilyForActor(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		if isSecondPCAttack(int(actor.Job), actor.Sex, int(actor.Weapon)) {
			return spriteActionPCAttack3
		}
		return spriteActionPCAttack2
	}
	return spriteActionNonPCAttack
}

func skillActionFamilyForActor(actor worldstate.Actor, skillID uint16) int {
	if skillID == 0 {
		return attackActionFamilyForActor(actor)
	}
	return skillAction(skillID).actionFamilyForActor(actor)
}

func skillCastActionFamilyForActor(actor worldstate.Actor, skillID uint16) int {
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

func hurtActionFamilyForActor(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionPCHurt
	}
	return spriteActionNonPCHurt
}

func deathActionFamilyForActor(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionPCDeath
	}
	return spriteActionNonPCDeath
}

func isSecondPCAttack(job int, sex byte, weaponValue int) bool {
	weaponType := db.PlayerWeaponType(weaponValue)
	switch job {
	case 0, 23, 4001, 4045:
		if sex != 0 {
			return weaponType == 2 || weaponType == 3 || (weaponType >= 6 && weaponType <= 10) || weaponType == 23
		}
		return weaponType == 1
	case 1, 7, 13, 14, 21:
		return weaponType >= 4 && weaponType <= 5
	case 2, 5:
		return weaponType == 1
	case 3:
		return weaponType != 11
	case 6, 11, 17, 19, 20:
		return weaponType == 11
	case 8:
		return weaponType == 15
	case 10, 18:
		return weaponType == 2 || (weaponType > 5 && weaponType <= 8)
	case 12:
		return weaponType == 16 || (weaponType > 24 && weaponType <= 30)
	case 15:
		return weaponType == 0 || weaponType == 12
	case 16:
		return weaponType == 5 || weaponType == 10 || weaponType == 15 || weaponType == 23
	case 24:
		return weaponType >= 18 && weaponType <= 21
	case 25:
		return weaponType == 22
	default:
		return false
	}
}

func actionHasHitReaction(action network.ActorActionNotify) bool {
	if action.Action == 4 || action.Action == 9 || action.Action == 11 {
		return false
	}
	return action.Damage > 0 || action.LeftDamage > 0
}

func (m *WorldMode) applyAttackFailureForDistance(ctx client.Context, failure network.AttackFailureForDistance) {
	attackRange := maxInt(1, failure.AttackRange)
	if ctx.Session != nil {
		ctx.Session.AttackRange = attackRange
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	log.Printf("attack distance failure target=%d server_player=%d,%d server_target=%d,%d range=%d client_player=%d,%d", failure.TargetID, failure.SourceX, failure.SourceY, failure.TargetX, failure.TargetY, attackRange, playerX, playerY)
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
		ctx.World.UpsertActor(actor)
	}
	if m.lockedAttackID != failure.TargetID && m.pendingAttack.targetID != failure.TargetID {
		return
	}
	m.pendingAttack = attackIntent{}
	m.lastAttackAt = time.Now()
	if !attackTargetWithinRange(failure.SourceX, failure.SourceY, failure.TargetX, failure.TargetY, attackRange) {
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
	log.Printf("recovery status=%d amount=%d hp=%d/%d sp=%d/%d", recovery.StatusID, recovery.Amount, ctx.Session.Vitals.HP, ctx.Session.Vitals.MaxHP, ctx.Session.Vitals.SP, ctx.Session.Vitals.MaxSP)
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
		actorID: actorID,
		x:       ctx.World.Player.X,
		y:       ctx.World.Player.Y,
		text:    fmt.Sprintf("%d", amount),
		color:   floaterColor,
		kind:    kind,
		starts:  now,
		expires: now.Add(damageFloaterDuration(kind)),
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
	if action.Action == 0 || action.Action == 7 {
		return "miss", damageFloaterMiss, damageFloaterWhite
	}
	return "", damageFloaterNormal, color.RGBA{}
}

func damageFloaterDuration(kind damageFloaterKind) time.Duration {
	switch kind {
	case damageFloaterMiss:
		return 800 * time.Millisecond
	default:
		return 1500 * time.Millisecond
	}
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

func attackApproachCell(ctx client.Context, actor worldstate.Actor, attackRange int) (int, int, bool) {
	attackRange = maxInt(1, attackRange)
	if attackRange > 1 {
		if x, y, ok := rangedAttackApproachCell(ctx, actor, attackRange); ok {
			return x, y, true
		}
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	bestX, bestY := 0, 0
	bestDistance := math.Inf(1)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			x := actor.X + dx
			y := actor.Y + dy
			if !walkTargetInBounds(ctx, x, y) {
				continue
			}
			if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			distance := math.Hypot(float64(x-playerX), float64(y-playerY))
			if distance < bestDistance {
				bestDistance = distance
				bestX = x
				bestY = y
			}
		}
	}
	return bestX, bestY, bestDistance < math.Inf(1)
}

func rangedAttackApproachCell(ctx client.Context, actor worldstate.Actor, attackRange int) (int, int, bool) {
	if ctx.World == nil {
		return 0, 0, false
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	stepX := approachSign(actor.X - playerX)
	stepY := approachSign(actor.Y - playerY)
	preferredX := actor.X - stepX*attackRange
	preferredY := actor.Y - stepY*attackRange
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
			x := actor.X + dx
			y := actor.Y + dy
			if !walkTargetInBounds(ctx, x, y) {
				continue
			}
			if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			candidates = append(candidates, candidate{
				x:                 x,
				y:                 y,
				sourceDistance:    maxInt(absInt(x-playerX), absInt(y-playerY)),
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

func (m *WorldMode) drawDamageFloaters(screen *render.Image, ctx client.Context, projection sceneProjection, now time.Time) {
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
		x, y := float64(floater.x), float64(floater.y)
		if actor, ok := ctx.World.Actors[floater.actorID]; ok {
			x, y = actor.RenderPosition(now)
		} else if isLocalActor(ctx, floater.actorID) {
			x, y = ctx.World.Player.RenderPosition(now)
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
		render.DebugPrintAtColor(screen, floater.text, int(point.x)-8, int(point.y)-40, withAlpha(floaterColor, alpha))
	}
	m.damageFloaters = active
}

func (m *WorldMode) startActorDeath(ctx client.Context, id uint32) {
	actor, ok, local := actorForCombatID(ctx, id)
	if !ok {
		if !local {
			ctx.World.RemoveActor(id)
		}
		return
	}
	now := time.Now()
	deathX, deathY := actor.RenderPosition(now)
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
		m.deathModal.OpenDeath()
	} else {
		ctx.World.UpsertActor(actor)
	}
	actionFamily := deathActionFamilyForActor(actor)
	deathDuration := m.actorActionDuration(ctx, actor, actionFamily, defaultDeathAnimationDuration)
	visibleDuration := deathDuration
	if !local {
		visibleDuration = maxDuration(deathDuration, nonPCDeathFadeDuration)
	}
	if local {
		m.startHeldCombatAnimation(ctx, id, actionFamily, now, deathDuration)
	} else {
		m.startCombatAnimation(ctx, id, actionFamily, now, visibleDuration)
	}
	if !local {
		if m.actorDeaths == nil {
			m.actorDeaths = make(map[uint32]time.Time)
		}
		m.actorDeaths[id] = now.Add(visibleDuration)
		if m.actorLife != nil {
			if life, ok := m.actorLife[id]; ok {
				life.hp = 0
				life.updatedAt = now
				m.actorLife[id] = life
			}
		}
	}
	log.Printf("actor death id=%d job=%d local=%t action=%d death_ms=%d remove_ms=%d", id, actor.Job, local, actionFamily, deathDuration.Milliseconds(), visibleDuration.Milliseconds())
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

func (m *WorldMode) drawAttackFocusMarker(screen *render.Image, ctx client.Context, now time.Time, entries []sceneActorDrawEntry) {
	if m.attackFocusID == 0 || screen == nil {
		return
	}
	view := m.cursorSpriteView(ctx)
	if view == nil || view.act == nil {
		return
	}
	info := cursorInfo(cursorActionLock)
	if m.attackFocusStart.IsZero() {
		m.attackFocusStart = now
	}
	action := cursorActionLock
	if action < 0 || action >= len(view.act.Actions) || len(view.act.Actions[action].Animations) == 0 {
		return
	}
	actionDef := view.act.Actions[action]
	delay := float64(actionDef.DelayMS) * info.delayMult
	motion := spriteMotionIndexWithDelay(actionDef, m.attackFocusStart, now, true, delay)
	frame, ok := cursorFrameBillboard(view, action, motion, info.drawX, info.drawY)
	if !ok {
		return
	}
	for _, entry := range entries {
		if entry.actor.ID != m.attackFocusID {
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
	if life.hasSP {
		return 9
	}
	return 5
}

func actorNameBelowLifeBarY(baseY, scale float64, life actorLife) float64 {
	return actorLifeBarY(baseY, scale) + actorLifeBarHeight(life) + 3
}

func (m *WorldMode) drawActorLifeBar(screen *render.Image, ctx client.Context, entry sceneActorDrawEntry) {
	life, ok := m.actorLifeForDisplay(ctx, entry.actor)
	if !ok {
		return
	}
	ratio := float64(life.hp) / float64(life.maxHP)
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	const width = 60.0
	height := actorLifeBarHeight(life)
	x := math.Round(entry.screenX - width/2)
	y := math.Round(actorLifeBarY(entry.screenY, entry.scale))
	fillWidth := math.Round((width - 2) * ratio)
	fill := color.RGBA{R: 255, G: 0, B: 231, A: 255}
	if life.player {
		fill = gameui.PlayerHPBarColor
		if ratio < 0.25 {
			fill = color.RGBA{R: 255, G: 0, B: 0, A: 255}
		}
	} else if ratio < 0.25 {
		fill = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	}
	render.DrawRect(screen, x, y, width, height, color.RGBA{R: 16, G: 24, B: 156, A: 255})
	render.DrawRect(screen, x+1, y+1, width-2, height-2, color.RGBA{R: 66, G: 66, B: 66, A: 255})
	if fillWidth > 0 {
		render.DrawRect(screen, x+1, y+1, fillWidth, 3, fill)
	}
	if life.hasSP {
		spRatio := float64(life.sp) / float64(life.maxSP)
		if spRatio < 0 {
			spRatio = 0
		} else if spRatio > 1 {
			spRatio = 1
		}
		render.DrawRect(screen, x, y+4, width, 1, color.RGBA{R: 16, G: 24, B: 156, A: 255})
		if spWidth := math.Round((width - 2) * spRatio); spWidth > 0 {
			render.DrawRect(screen, x+1, y+5, spWidth, 3, gameui.PlayerSPBarColor)
		}
	}
}

func (m *WorldMode) drawActorCastBar(screen *render.Image, entry sceneActorDrawEntry, now time.Time) {
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
	const width = 60.0
	const height = 6.0
	x := math.Round(entry.screenX - width/2)
	y := math.Round(actorCastBarY(entry.screenY, entry.scale))
	fillWidth := math.Round((width - 2) * ratio)
	render.DrawRect(screen, x, y, width, height, color.RGBA{R: 16, G: 24, B: 156, A: 255})
	render.DrawRect(screen, x+1, y+1, width-2, height-2, color.RGBA{R: 66, G: 66, B: 66, A: 255})
	if fillWidth > 0 {
		fill := bar.color
		if fill.A == 0 {
			fill = color.RGBA{R: 0, G: 255, B: 0, A: 255}
		}
		render.DrawRect(screen, x+1, y+1, fillWidth, height-2, fill)
	}
}

func (m *WorldMode) actorLifeForDisplay(ctx client.Context, actor worldstate.Actor) (actorLife, bool) {
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
	// Monster HP bars are a 2012+ client feature. The 2008 client exposes
	// monster HP through WZ_ESTIMATION/Sense instead, so keep the combat HP
	// cache hidden from the normal actor overlay.
	return actorLife{}, false
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
		character := selectedCharacter(ctx.Session)
		hp = int(character.HP)
		maxHP = int(character.MaxHP)
	}
	if maxSP <= 0 {
		character := selectedCharacter(ctx.Session)
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
