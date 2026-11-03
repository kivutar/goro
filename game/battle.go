package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
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

func combatDuration(speed int32, fallback time.Duration) time.Duration {
	if speed <= 0 {
		return fallback
	}
	duration := time.Duration(speed) * time.Millisecond
	if duration > scene.MaxCombatAnimationDuration {
		return scene.MaxCombatAnimationDuration
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
	if !skills.LocalStealthAllowsSkill(ctx, 0) || scene.ActorHasStealth(actor) {
		m.cancelAttackIntent()
		return
	}
	if scene.PlayerIsDead(ctx) {
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
	playerX, playerY := scene.CurrentPlayerCell(ctx, now)
	targetX, targetY := scene.ActorCurrentCell(actor, now)
	if maps.NormalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange) {
		m.sendAttackAction(ctx, actor, source)
		return
	}
	chaseX, chaseY, ok := maps.NormalAttackApproachCellFromTarget(ctx, playerX, playerY, targetX, targetY, attackRange)
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
	playerX, playerY := scene.CurrentPlayerCell(ctx, now)
	targetX, targetY := scene.ActorCurrentCell(actor, now)
	if !maps.NormalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange) {
		if logOutOfRange {
			glog.Debugf("%s pending attack still out of range target=%d player=%d,%d target=%d,%d range=%d", source, actor.ID, playerX, playerY, targetX, targetY, attackRange)
		}
		if scene.MovingActorDestinationWithinRange(ctx.World.Player, targetX, targetY, attackRange, maps.NormalAttackTargetWithinRange) {
			return
		}
		if scene.AttackRetryDue(m.pendingAttack.lastChaseAt, now) {
			m.requestAttack(ctx, actor, "pending")
		}
		return
	}
	readyAt := scene.PendingAttackReadyAt(ctx.World.Player, now)
	if m.pendingAttack.readyAt.IsZero() {
		m.pendingAttack.readyAt = readyAt
	}
	if logOutOfRange {
		glog.Debugf("%s pending attack scheduled target=%d delay_ms=%d", source, actor.ID, scene.MaxInt(0, int(m.pendingAttack.readyAt.Sub(now).Milliseconds())))
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
	playerX, playerY := scene.CurrentPlayerCell(ctx, now)
	targetX, targetY := scene.ActorCurrentCell(actor, now)
	if !maps.NormalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange) {
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
	if scene.ActorIsMovingAt(ctx.World.Player, now) {
		return
	}
	actor, ok := ctx.World.Actors[m.lockedAttackID]
	if !ok {
		glog.Debugf("locked attack target vanished id=%d", m.lockedAttackID)
		m.clearLockedAttack()
		return
	}
	if !actors.ActorCanBeAttackClicked(ctx, actor) {
		glog.Debugf("locked attack target no longer attackable id=%d object_type=%d", actor.ID, actor.ObjectType)
		m.clearLockedAttack()
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	playerX, playerY := scene.CurrentPlayerCell(ctx, now)
	targetX, targetY := scene.ActorCurrentCell(actor, now)
	if maps.NormalAttackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange) {
		if !scene.AttackRetryDue(m.lastAttackAt, now) {
			return
		}
		glog.Debugf("locked attack retry target=%d player=%d,%d target=%d,%d range=%d", actor.ID, playerX, playerY, targetX, targetY, attackRange)
		m.sendAttackAction(ctx, actor, "locked")
		return
	}
	if !scene.AttackRetryDue(m.lastChaseAt, now) {
		return
	}
	m.lastChaseAt = now
	glog.Debugf("locked attack chase retry target=%d player=%d,%d target=%d,%d range=%d", actor.ID, playerX, playerY, targetX, targetY, attackRange)
	m.requestAttack(ctx, actor, "locked")
}

func normalAttackLockActive(ctx client.Context) bool {
	return (ctx.Input != nil && ctx.Input.Pressed(input.KeyCtrl)) || (ctx.Session != nil && ctx.Session.NoCtrl)
}

func (m *WorldMode) sendAttackAction(ctx client.Context, actor world.Actor, source string) {
	if !skills.LocalStealthAllowsSkill(ctx, 0) || scene.ActorHasStealth(actor) {
		m.cancelAttackIntent()
		return
	}
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
	source, sourceOK, sourceLocal := scene.ActorForCombatID(ctx, action.SourceID)
	target, targetOK, targetLocal := scene.ActorForCombatID(ctx, action.TargetID)
	splash := skillSplashAction(action)
	if action.SkillID > 0 && !splash {
		m.applySkillNameBubble(ctx, action.SourceID, action.SkillID, now)
		m.actors().ApplyFalconActorActionNotify(ctx, action, now)
	}
	if sourceOK && targetOK && !splash {
		m.faceCombatSource(ctx, source, sourceLocal, target)
		source.Dir = actors.DirectionFromDelta(source.X, source.Y, target.X, target.Y, source.Dir)
	}
	attackDuration := combatDuration(action.SourceSpeed, scene.DefaultAttackAnimationDuration)
	attackFamily := sprites.ActionNonPCAttack
	if sourceOK {
		attackFamily = skillActionFamilyForActorWithResources(ctx.Resources, source, action.SkillID)
	}
	playerBowAttack := sourceOK && res.HasPlayerJobToken(int(source.Job)) && actorUsesBow(ctx.Resources, source)
	normalPlayerBowAttack := action.SkillID == 0 && playerBowAttack
	doubleStrafeBowAttack := action.SkillID == db.SkillACDouble && playerBowAttack
	showerBowAttack := action.SkillID == db.SkillACShower && playerBowAttack
	phantasmicBowAttack := (action.SkillID == db.SkillHTPhantasmic || action.SkillID == db.SkillHTPower) && playerBowAttack
	sharpShootingBowAttack := action.SkillID == db.SkillSNSharpshooting && playerBowAttack
	instrumentArrowAttack := sourceOK && res.HasPlayerJobToken(int(source.Job)) && (action.SkillID == db.SkillBaMusicalstrike || action.SkillID == db.SkillDCThrowarrow || action.SkillID == db.SkillCGArrowvulcan)
	referenceArrowFlightAttack := instrumentArrowAttack || normalPlayerBowAttack || doubleStrafeBowAttack || showerBowAttack || phantasmicBowAttack || sharpShootingBowAttack
	if sourceOK && !splash {
		if anim, ok := m.actors().ActorAnimation(action.SourceID, now); ok && anim.AutoCounter && action.SkillID == 0 {
			m.effects().AddAt(ctx, effects.AutoCounter, action.SourceID, now)
		}
		if action.SkillID > 0 {
			spec := effects.SkillAction(action.SkillID)
			if action.SkillID == db.SkillTKCounter && source.Job == db.JobLinker {
				spec = effects.TaekwonKickSequence(effects.SkillActorActionAttack3, [5]int{4, 5, 6, 7, 2}, 4)
			}
			m.startSkillActionAnimation(ctx, action.SourceID, source, spec, now, attackDuration)
		} else if sourceLocal && res.HasPlayerJobToken(int(source.Job)) {
			m.startCombatAnimationWithTimingAndNext(ctx, action.SourceID, source, attackFamily, now, attackDuration, readyFightAnimation(now.Add(attackDuration)))
		} else {
			m.startCombatAnimationWithTiming(ctx, action.SourceID, source, attackFamily, now, attackDuration)
		}
	}
	hitDelay := combatDuration(action.SourceSpeed, 0)
	if sourceOK {
		if actionDef, ok := m.actors().ActorResolvedAction(ctx, source, attackFamily); ok {
			actionACT := m.actors().ActorActionACT(ctx, source)
			soundMotion := firstActionSoundMotion(actionDef)
			if referenceArrowFlightAttack {
				soundMotion = actionAttackMarkerMotion(actionACT, actionDef)
			}
			hitDelay = combatMotionDelayFromAction(actionDef, attackDuration, soundMotion)
			if sounds := actionSFXCandidatesForActor(source, actionACT, actionDef, soundMotion); len(sounds) > 0 && !splash {
				if action.SkillID == db.SkillACShower {
					m.scheduleSkillActionSoundAtActor(now.Add(hitDelay), action.SourceID, action.SkillID, sounds...)
				} else {
					m.scheduleSoundAtActor(now.Add(hitDelay), action.SourceID, sounds...)
				}
			}
		}
	}
	releaseAt := now.Add(hitDelay)
	hitAt := releaseAt
	if !splash && (action.SkillID == db.SkillTKDownkick || action.SkillID == db.SkillTKTurnkick) {
		hitAt = hitAt.Add(300 * time.Millisecond)
	}
	if action.SkillID == db.SkillPRTurnundead && !splash {
		hitAt = hitAt.Add(500 * time.Millisecond)
	}
	projectileAt := now
	projectileDuration := time.Duration(0)
	if referenceArrowFlightAttack {
		// The reference client reveals the arrow at the ACT "atk" marker, then
		// gives it eight 24 ms ticks to reach the target before applying the hit.
		projectileAt = releaseAt
		projectileDuration = scene.ReferenceBowArrowFlightDuration
		hitAt = releaseAt.Add(scene.ReferenceBowArrowFlightDuration)
	}
	if sourceOK {
		m.startArrowVulcanHits(ctx, action, hitAt)
	}
	dealsDamage := actionDealsDamage(action)
	if targetOK {
		if hitAt.Before(now) {
			hitAt = now
		}
		if action.SkillID > 0 {
			m.addSkillBeginEffect(ctx, action, now)
			if doubleStrafeBowAttack || showerBowAttack || phantasmicBowAttack || sharpShootingBowAttack || instrumentArrowAttack {
				// These ranged skills release one arrow at the ACT marker.
				m.addSkillBeforeHitEffectOnce(ctx, action, projectileAt, projectileDuration)
			} else {
				m.addSkillBeforeHitEffect(ctx, action, now)
			}
		} else {
			m.addNormalAttackBeforeHitEffect(ctx, action, source, sourceOK, projectileAt, projectileDuration)
		}
		if dealsDamage {
			m.addNormalAttackHitEffect(ctx, action, target, targetOK, hitAt)
			if actionHasHitReaction(action) && skillTargetUsesHitReaction(action, sourceLocal, targetLocal) {
				hurtDuration := combatDuration(action.TargetSpeed, scene.DefaultHitAnimationDuration)
				m.startCombatAnimationWithNext(ctx, action.TargetID, hurtActionFamilyForActor(target), hitAt, hurtDuration, postHurtAnimation(target, hitAt.Add(hurtDuration)))
			}
			hitSoundCount := 1
			if doubleStrafeBowAttack {
				hitSoundCount = scene.ActionVisualHitCount(action)
			}
			hitSounds := combatHitSFXCandidates(action, source, sourceOK, target, targetOK)
			for i := 0; i < hitSoundCount; i++ {
				m.scheduleSoundAtActor(hitAt.Add(scene.MultiHitDelay*time.Duration(i)), action.TargetID, hitSounds...)
			}
		}
		if action.SkillID > 0 {
			m.addSkillEffect(ctx, action, now, hitAt)
			if dealsDamage {
				m.addSkillHitEffect(ctx, action, hitAt)
			}
		}
	}
	x, y := ctx.World.Player.X, ctx.World.Player.Y
	if targetOK {
		x, y = target.X, target.Y
	} else if scene.IsLocalActor(ctx, action.TargetID) {
		x, y = ctx.World.Player.X, ctx.World.Player.Y
	}
	m.battle().AddActionDamageFloaters(ctx, action, target, targetOK, targetLocal, sourceLocal, x, y, hitAt)
	if sourceLocal {
		m.pets().MaybeSendHuntingTalk(ctx, now)
	}
}

func (m *WorldMode) applyActorActionAIState(ctx client.Context, action network.ActorActionNotify, now time.Time) {
	if action.SourceID != 0 {
		motion := companion.MotionAttack
		expires := now.Add(combatDuration(action.SourceSpeed, scene.DefaultAttackAnimationDuration))
		if action.Action == network.ActorActionPickupItem || action.Action == network.ActionSitDown || action.Action == network.ActionStandUp {
			motion = companion.MotionStand
			expires = time.Time{}
		}
		companion.SetActorMotion(ctx, action.SourceID, motion, action.TargetID, expires)
	}
	if action.TargetID != 0 && actionHasHitReaction(action) {
		companion.SetActorMotion(ctx, action.TargetID, companion.MotionStand, action.SourceID, time.Time{})
	}
}

func skillSplashAction(action network.ActorActionNotify) bool {
	if action.Action != 5 { // ACTION_SPLASH: secondary victims do not replay the skill.
		return false
	}
	switch action.SkillID {
	case db.SkillHTBlitzbeat, db.SkillHTPhantasmic, db.SkillHTPower, db.SkillSNFalconassault, db.SkillSNSharpshooting, db.SkillBaMusicalstrike, db.SkillDCThrowarrow, db.SkillCGArrowvulcan:
		return true
	case db.SkillACDouble, db.SkillACChargearrow, db.SkillMCMammonite, db.SkillMCCartrevolution, db.SkillWSCarttermination, db.SkillAMAcidterror, db.SkillCRAciddemonstration:
		return true
	case db.SkillTFPoison, db.SkillTFSprinklesand, db.SkillTFThrowstone:
		return true
	case db.SkillWZFirepillar, db.SkillWZSightrasher, db.SkillWZMeteor, db.SkillWZJupitel, db.SkillWZVermilion,
		db.SkillWZWaterball, db.SkillWZFrostnova, db.SkillWZStormgust, db.SkillWZEarthspike, db.SkillWZHeavendrive:
		return true
	case db.SkillMGSight, db.SkillMGSoulstrike, db.SkillMGColdbolt, db.SkillMGFrostdiver, db.SkillMGStonecurse,
		db.SkillMGFireball, db.SkillMGFirewall, db.SkillMGFirebolt, db.SkillMGLightningbolt, db.SkillMGThunderstorm:
		return true
	case db.SkillALLResurrection, db.SkillPRAspersio, db.SkillPRBenedictio, db.SkillPRSanctuary, db.SkillPRTurnundead, db.SkillPRMagnus:
		return true
	case db.SkillMOTripleattack, db.SkillMOInvestigate, db.SkillMOFingeroffensive, db.SkillMOExtremityfist, db.SkillMOChaincombo, db.SkillMOBalkyoung, db.SkillChPalmstrike, db.SkillChTigerfist, db.SkillChChaincrush:
		return true
	case db.SkillLKSpiralpierce, db.SkillLKHeadcrush, db.SkillLKJointbeat:
		return true
	case db.SkillHWMagiccrasher, db.SkillHWNapalmvulcan, db.SkillHWGravitation, db.SkillPFSoulburn:
		return true
	case db.SkillASCBreaker, db.SkillASCMeteorassault:
		return true
	case db.SkillTKDownkick, db.SkillTKTurnkick, db.SkillTKCounter, db.SkillTKJumpkick:
		return true
	case db.SkillCRShieldcharge, db.SkillCRShieldboomerang, db.SkillCRHolycross, db.SkillCRGrandcross, db.SkillPaPressure, db.SkillPaSacrifice, db.SkillPaShieldchain,
		db.SkillKNPierce, db.SkillKNBrandishspear, db.SkillKNSpearstab, db.SkillKNSpearboomerang, db.SkillKNBowlingbash, db.SkillKNChargeatk:
		return true
	default:
		return false
	}
}

func (m *WorldMode) addSkillBeginEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if skillSplashAction(action) || action.SkillID == 0 {
		return
	}
	for _, effectID := range effects.SkillBeginEffectIDs(action.SkillID) {
		actorID := action.SourceID
		if effects.EffectDetachesLocalActor(effectID) && scene.IsLocalActor(ctx, actorID) {
			actorID = 0
		}
		if m.effects().AddAt(ctx, effectID, actorID, starts) {
			glog.Debugf("skill begin effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
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
	if m.effects().AddBetweenDuration(ctx, effectID, action.TargetID, action.SourceID, starts, duration) {
		glog.Debugf("normal attack before-hit effect src=%d target=%d effect=%d", action.SourceID, action.TargetID, effectID)
	}
}

func (m *WorldMode) addNormalAttackHitEffect(ctx client.Context, action network.ActorActionNotify, target world.Actor, targetOK bool, starts time.Time) {
	if action.SkillID != 0 || action.Damage <= 0 || !targetOK || !actorUsesReferenceNormalHitEffect(target) {
		return
	}
	count := scene.ActionVisualHitCount(action)
	for i := 0; i < count; i++ {
		effectStarts := starts.Add(scene.MultiHitDelay * time.Duration(i))
		if m.effects().AddAt(ctx, effects.Hit1, action.TargetID, effectStarts) {
			glog.Debugf("normal hit effect src=%d target=%d effect=%d hit=%d/%d", action.SourceID, action.TargetID, effects.Hit1, i+1, count)
		}
		if actionUsesReferenceCriticalHitEffect(action) {
			if m.effects().AddAt(ctx, effects.BashHit, action.TargetID, effectStarts) {
				glog.Debugf("critical hit effect src=%d target=%d effect=%d hit=%d/%d", action.SourceID, action.TargetID, effects.BashHit, i+1, count)
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
	case scene.ActorObjectTypeMob, scene.ActorObjectTypeNPCABR, scene.ActorObjectTypeNPCBionic:
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
		return effects.ArrowShot, true
	}
	return normalAttackProjectileEffectIDForName(db.MonsterAttackProjectile[int(actor.Job)])
}

func normalAttackProjectileEffectIDForName(name string) (int, bool) {
	switch name {
	case "ef_arrow_projectile":
		return effects.ArrowShot, true
	default:
		return 0, false
	}
}

func (m *WorldMode) addSkillEffect(ctx client.Context, action network.ActorActionNotify, actionAt, hitAt time.Time) {
	if action.SkillID == 0 || skillSplashAction(action) {
		return
	}
	if action.SkillID == db.SkillASCBreaker {
		if _, sourceOK, _ := scene.ActorForCombatID(ctx, action.SourceID); sourceOK {
			if _, targetOK, _ := scene.ActorForCombatID(ctx, action.TargetID); targetOK {
				m.effects().AddBetween(ctx, effects.SoulBreaker, action.SourceID, action.TargetID, actionAt)
			}
		}
		return
	}
	if action.SkillID == db.SkillACShower {
		if target, ok, _ := scene.ActorForCombatID(ctx, action.TargetID); ok {
			x, y := scene.ActorRenderPosition(target, actionAt)
			m.effects().AddArrowShower(ctx, action.SourceID, x, y, false, actionAt)
		}
		return
	}
	if action.SkillID == db.SkillASSplasher {
		// ZC_NOTIFY_SKILL's level -1 marks the server's delayed explosion.
		if action.SkillLevel == 0xffff {
			m.effects().AddBetween(ctx, effects.VenomSplasher, action.TargetID, action.SourceID, actionAt)
		}
		return
	}
	if action.SkillID == db.SkillASGrimtooth {
		if action.Action != network.ActorActionSkill {
			return
		}
		if _, ok, _ := scene.ActorForCombatID(ctx, action.SourceID); !ok {
			return
		}
		// The preamble and primary hit share a caster and server tick. Damage
		// cannot identify the preamble: WoE and Hallucination can rewrite it.
		for _, effect := range m.effects().Active {
			if effect.EffectID == effects.Grimtooth && effect.ActorID == action.SourceID && effect.ServerTick == action.ServerTick {
				return
			}
		}
	}
	spec := effects.SkillEffectSpecFor(action.SkillID)
	starts := hitAt
	if spec.EffectTiming == db.SkillEffectAtActionStart {
		starts = actionAt
	}
	for _, sound := range db.SkillEffects[action.SkillID].SFX {
		m.scheduleSoundAtActor(actionAt, action.SourceID, sound)
	}
	for _, effectID := range spec.EffectIDs {
		if m.effects().AddBetween(ctx, effectID, action.TargetID, action.SourceID, starts) {
			m.effects().Active[len(m.effects().Active)-1].ServerTick = action.ServerTick
			if effectID == effects.Tanji {
				// LaunchSkillSpecialEffect receives the hit count (spheres used).
				count := scene.MinInt(5, scene.ActionVisualHitCount(action))
				m.effects().Active[len(m.effects().Active)-1].Size = math.Sqrt2 * (2 + 0.4*float64(count)) / 5
			}
			glog.Debugf("skill effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
	for _, effectID := range spec.EffectIDsOnCaster {
		if m.effects().AddAt(ctx, effectID, action.SourceID, starts) {
			glog.Debugf("skill caster effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
}

func (m *WorldMode) addSkillBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 || skillSplashAction(action) {
		return
	}
	if m.effects().AddMageProjectileEffect(ctx, action, starts) || m.effects().AddNinjaBeforeHitEffect(ctx, action, starts) || m.effects().AddGunslingerBeforeHitEffect(ctx, action, starts) {
		return
	}
	effectIDs := effects.SkillBeforeHitEffectIDs(action.SkillID)
	selfEffectIDs := effects.SkillBeforeHitEffectSelfIDs(action.SkillID)
	if len(effectIDs) == 0 && len(selfEffectIDs) == 0 {
		return
	}
	count := scene.ActionVisualHitCount(action)
	if action.SkillID == db.SkillKNSpearboomerang {
		count = 1
	}
	for i := 0; i < count; i++ {
		effectStarts := starts.Add(scene.MultiHitDelay * time.Duration(i))
		for _, effectID := range effectIDs {
			if m.effects().AddBetween(ctx, effectID, action.TargetID, action.SourceID, effectStarts) {
				glog.Debugf("skill before-hit effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
		for _, effectID := range selfEffectIDs {
			if m.effects().AddAt(ctx, effectID, action.SourceID, effectStarts) {
				glog.Debugf("skill before-hit self effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
	}
}

func (m *WorldMode) addSkillBeforeHitEffectOnce(ctx client.Context, action network.ActorActionNotify, starts time.Time, duration time.Duration) {
	if action.SkillID == 0 || skillSplashAction(action) {
		return
	}
	for _, effectID := range effects.SkillBeforeHitEffectIDs(action.SkillID) {
		if m.effects().AddBetweenDuration(ctx, effectID, action.TargetID, action.SourceID, starts, duration) {
			glog.Debugf("skill before-hit effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
	for _, effectID := range effects.SkillBeforeHitEffectSelfIDs(action.SkillID) {
		if m.effects().AddBetweenDuration(ctx, effectID, action.SourceID, 0, starts, duration) {
			glog.Debugf("skill before-hit self effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
}

func (m *WorldMode) addSkillHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 || action.Damage == 0 || skillSplashAction(action) {
		return
	}
	effectIDs := effects.SkillHitEffectIDs(action.SkillID)
	casterEffectIDs := effects.SkillHitEffectOnCasterIDs(action.SkillID)
	if len(effectIDs) == 0 && len(casterEffectIDs) == 0 {
		return
	}
	count := scene.ActionVisualHitCount(action)
	switch action.SkillID {
	case db.SkillKNPierce, db.SkillKNSpearstab, db.SkillKNSpearboomerang, db.SkillKNChargeatk, db.SkillLKSpiralpierce:
		// The original clears its secondary impact effect after the first hit.
		count = 1
	}
	for i := 0; i < count; i++ {
		effectStarts := starts.Add(scene.MultiHitDelay * time.Duration(i))
		for secondary, effectID := range effectIDs {
			if i > 0 && secondary > 0 && action.SkillID >= db.SkillWZFirepillar && action.SkillID <= db.SkillWZQuagmire {
				continue
			}
			if (effectID == effects.BowlingBash || effectID == effects.BlitzBeat) && i > 0 {
				continue
			}
			if i > 0 && effectID == effects.BashHit && (action.SkillID == db.SkillTFPoison || action.SkillID == db.SkillACDouble || action.SkillID == db.SkillSNSharpshooting || action.SkillID == db.SkillCGArrowvulcan || action.SkillID == db.SkillHTPhantasmic || action.SkillID == db.SkillHTPower || action.SkillID == db.SkillMOFingeroffensive || action.SkillID == db.SkillMOCombofinish || action.SkillID == db.SkillChTigerfist || action.SkillID == db.SkillChChaincrush) {
				continue
			}
			if i > 0 && (action.SkillID == db.SkillMGNapalmbeat || action.SkillID == db.SkillMGSoulstrike || action.SkillID == db.SkillMGFireball || action.SkillID == db.SkillMGFrostdiver) && (effectID == effects.BashHit || effectID == effects.FrostDiverHit) {
				continue
			}
			if m.effects().AddAt(ctx, effectID, action.TargetID, effectStarts) {
				glog.Debugf("skill hit effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
		for _, effectID := range casterEffectIDs {
			if m.effects().AddAt(ctx, effectID, action.SourceID, effectStarts) {
				glog.Debugf("skill hit caster effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
	}
}

func (m *WorldMode) applyActorPickupActionNotify(ctx client.Context, action network.ActorActionNotify, now time.Time) {
	source, sourceOK, sourceLocal := scene.ActorForCombatID(ctx, action.SourceID)
	if !sourceOK {
		return
	}
	if ctx.World != nil {
		if item, ok := ctx.World.Items[action.TargetID]; ok {
			dir := actors.DirectionFromDelta(source.X, source.Y, item.X, item.Y, source.Dir)
			if sourceLocal {
				ctx.World.Player.Dir = dir
				ctx.World.Dir = dir
				if ctx.Session != nil {
					ctx.Session.PlayerDir = dir
				}
			} else {
				source.Dir = dir
				actors.UpsertActor(ctx, source)
			}
		}
	}
	m.startCombatAnimation(ctx, action.SourceID, scene.SpriteActionPickup, now, pickupAnimationDuration)
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
	if scene.IsLocalActor(ctx, id) {
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
	actors.UpsertActor(ctx, actor)
	if !sitting {
		actor = ctx.World.Actors[id]
		actor.Sitting = false
		ctx.World.Actors[id] = actor
	}
	m.clearActorAction(ctx, id)
}

func (m *WorldMode) faceCombatSource(ctx client.Context, source world.Actor, sourceLocal bool, target world.Actor) {
	dir := actors.DirectionFromDelta(source.X, source.Y, target.X, target.Y, source.Dir)
	if sourceLocal {
		ctx.World.Player.Dir = dir
		ctx.World.Dir = dir
		return
	}
	source.Dir = dir
	actors.UpsertActor(ctx, source)
}

func (m *WorldMode) startActorAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimationWithOptions(ctx, id, actionFamily, started, duration, false)
}

func (m *WorldMode) startHeldActorAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimationWithOptions(ctx, id, actionFamily, started, duration, true)
}

func (m *WorldMode) startActorAnimationWithOptions(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, holdFinal bool) {
	m.setActorAction(ctx, id, scene.ActorAnimation{
		ActionFamily: actionFamily,
		Started:      started,
		Duration:     duration,
		Play:         true,
		HasPlay:      true,
		HoldFinal:    holdFinal,
	})
}

func (m *WorldMode) startActorAnimationWithNext(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, next *scene.ActorAnimation) {
	m.setActorAction(ctx, id, scene.ActorAnimation{
		ActionFamily: actionFamily,
		Started:      started,
		Duration:     duration,
		Play:         true,
		HasPlay:      true,
		Next:         cloneActorAnimation(next),
	})
}

func (m *WorldMode) setActorAction(ctx client.Context, id uint32, anim scene.ActorAnimation) {
	if id == 0 || anim.ActionFamily < 0 {
		return
	}
	if anim.Started.IsZero() {
		anim.Started = time.Now()
	}
	if actorActionResetsWalk(anim) {
		resumeWalk := m.actorActionShouldResumeWalk(ctx, id, anim)
		if anim.Started.After(time.Now()) {
			m.scheduleActorStop(id, anim.Started, resumeWalk, anim.Started.Add(anim.Duration))
		} else {
			if resumeWalk {
				m.pauseActorMovementForResume(ctx, id, anim.Started, anim.Started.Add(anim.Duration))
			} else {
				m.stopActorMovementAt(ctx, id, anim.Started)
			}
		}
	}
	if m.actorSystem.ActorAnims == nil {
		m.actorSystem.ActorAnims = make(map[uint32]scene.ActorAnimation)
	}
	anim.Next = cloneActorAnimation(anim.Next)
	m.actorSystem.ActorAnims[id] = anim
}

func actorActionResetsWalk(anim scene.ActorAnimation) bool {
	return !anim.KeepMovement && anim.ActionFamily != scene.SpriteActionWalk
}

func (m *WorldMode) actorActionShouldResumeWalk(ctx client.Context, id uint32, anim scene.ActorAnimation) bool {
	if anim.ActionFamily != sprites.ActionPCHurt && anim.ActionFamily != sprites.ActionNonPCHurt {
		return false
	}
	return scene.IsLocalActor(ctx, id) && m.hasCombatFocus()
}

func (m *WorldMode) hasCombatFocus() bool {
	return m.lockedAttackID != 0 || m.pendingAttack.targetID != 0
}

func cloneActorAnimation(anim *scene.ActorAnimation) *scene.ActorAnimation {
	if anim == nil {
		return nil
	}
	cloned := *anim
	cloned.Next = cloneActorAnimation(anim.Next)
	return &cloned
}

func readyFightAnimation(started time.Time) *scene.ActorAnimation {
	return &scene.ActorAnimation{
		ActionFamily: sprites.ActionPCReadyFight,
		Started:      started,
		Loop:         true,
		Play:         true,
		HasPlay:      true,
	}
}

func postHurtAnimation(actor world.Actor, started time.Time) *scene.ActorAnimation {
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
	if id == 0 || m.actorSystem.ActorAnims == nil {
		return
	}
	if anim, ok := m.actorSystem.ActorAnims[id]; ok && m.actorActionAnimationIsDeath(ctx, id, anim) {
		return
	}
	delete(m.actorSystem.ActorAnims, id)
}

func (m *WorldMode) actorActionAnimationIsDeath(ctx client.Context, id uint32, anim scene.ActorAnimation) bool {
	if anim.ActionFamily == sprites.ActionPCDeath {
		return true
	}
	if anim.ActionFamily != sprites.ActionNonPCDeath {
		return false
	}
	actor, ok, local := scene.ActorForCombatID(ctx, id)
	if !ok {
		return true
	}
	return !local && !res.HasPlayerJobToken(int(actor.Job))
}

func (m *WorldMode) startCombatAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimation(ctx, id, actionFamily, started, duration)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, id) {
		return
	}
	m.startActorAnimation(ctx, ctx.Session.AccountID, actionFamily, started, duration)
	m.startActorAnimation(ctx, ctx.Session.CharID, actionFamily, started, duration)
}

func (m *WorldMode) startCombatAnimationWithTiming(ctx client.Context, id uint32, actor world.Actor, actionFamily int, started time.Time, duration time.Duration) {
	anim := timedCombatAnimation(m.actors().ActorActionFrameDelay(ctx, actor, actionFamily, duration), actionFamily, started, duration, nil)
	m.setCombatActorAction(ctx, id, anim)
}

func (m *WorldMode) startCombatAnimationWithNext(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, next *scene.ActorAnimation) {
	m.startActorAnimationWithNext(ctx, id, actionFamily, started, duration, next)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, id) {
		return
	}
	m.startActorAnimationWithNext(ctx, ctx.Session.AccountID, actionFamily, started, duration, next)
	m.startActorAnimationWithNext(ctx, ctx.Session.CharID, actionFamily, started, duration, next)
}

func (m *WorldMode) startCombatAnimationWithTimingAndNext(ctx client.Context, id uint32, actor world.Actor, actionFamily int, started time.Time, duration time.Duration, next *scene.ActorAnimation) {
	anim := timedCombatAnimation(m.actors().ActorActionFrameDelay(ctx, actor, actionFamily, duration), actionFamily, started, duration, next)
	m.setCombatActorAction(ctx, id, anim)
}

func (m *WorldMode) setCombatActorAction(ctx client.Context, id uint32, anim scene.ActorAnimation) {
	m.setActorAction(ctx, id, anim)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, id) {
		return
	}
	m.setActorAction(ctx, ctx.Session.AccountID, anim)
	m.setActorAction(ctx, ctx.Session.CharID, anim)
}

func timedCombatAnimation(frameDelay time.Duration, actionFamily int, started time.Time, duration time.Duration, next *scene.ActorAnimation) scene.ActorAnimation {
	anim := scene.ActorAnimation{
		ActionFamily: actionFamily,
		Started:      started,
		Duration:     duration,
		Play:         true,
		HasPlay:      true,
		Next:         cloneActorAnimation(next),
	}
	if frameDelay > 0 {
		anim.Speed = frameDelay
		anim.HasSpeed = true
	}
	return anim
}

func (m *WorldMode) startHeldCombatAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startHeldActorAnimation(ctx, id, actionFamily, started, duration)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, id) {
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
	if ctx.Session == nil || m.actorSystem.ActorAnims == nil {
		return
	}
	m.clearActorDeathAnimation(ctx.Session.AccountID)
	m.clearActorDeathAnimation(ctx.Session.CharID)
}

func (m *WorldMode) clearActorDeathAnimation(id uint32) {
	if id == 0 || m.actorSystem.ActorAnims == nil {
		return
	}
	anim, ok := m.actorSystem.ActorAnims[id]
	if !ok {
		return
	}
	if anim.ActionFamily != sprites.ActionPCDeath && anim.ActionFamily != sprites.ActionNonPCDeath {
		return
	}
	delete(m.actorSystem.ActorAnims, id)
}

func skillActionFamilyForActorWithResources(manager *res.Manager, actor world.Actor, skillID uint16) int {
	if skillID == 0 {
		return sprites.AttackActionFamily(manager, actor)
	}
	return effects.SkillAction(skillID).ActionFamilyForActorWithResources(manager, actor)
}

func skillCastActionFamilyForActor(actor world.Actor, skillID uint16) int {
	if companion.IsMercenary(actor) {
		return sprites.ActionPCSkill
	}
	if !res.HasPlayerJobToken(int(actor.Job)) {
		return sprites.ActionNonPCAttack
	}
	if skillID == db.SkillSNSharpshooting || skillID == db.SkillCGArrowvulcan {
		return sprites.ActionPCAttack3
	}
	if skillID == db.SkillTFPickstone {
		return scene.SpriteActionPickup
	}
	if skillID == db.SkillALBlessing || skillID == db.SkillMOSteelbody || skillID == db.SkillMOExplosionspirits || skillID == db.SkillKNChargeatk || skillID == db.SkillPRRedemptio || skillID == db.SkillBaPangvoice {
		return scene.SpriteActionIdle
	}
	switch skillID {
	case db.SkillAMPotionpitcher, db.SkillAMAcidterror, db.SkillAMCpWeapon, db.SkillAMCpShield, db.SkillAMCpArmor, db.SkillAMCpHelm, db.SkillAMCannibalize, db.SkillAMCallhomun, db.SkillAMRest, db.SkillAMResurrecthomun:
		return scene.SpriteActionIdle
	case db.SkillAMBerserkpitcher, db.SkillCRSlimpitcher, db.SkillCRAciddemonstration:
		return sprites.ActionPCAttack1
	case db.SkillCRFullprotection, db.SkillSTChasewalk, db.SkillSTPreserve:
		return scene.SpriteActionIdle
	case db.SkillTKSevenwind, db.SkillTKMission:
		return scene.SpriteActionIdle
	case db.SkillAMTwilight1, db.SkillAMTwilight2, db.SkillAMTwilight3, db.SkillChSoulcollect, db.SkillMOCallspirits, db.SkillMOKitranslation:
		return sprites.ActionPCSkill
	}
	if skillID == db.SkillALIncagi {
		return sprites.ActionPCSkill
	}
	return sprites.ActionPCReadyFight
}

func skillTargetUsesHitReaction(action network.ActorActionNotify, sourceLocal, targetLocal bool) bool {
	if action.SkillID > 0 && sourceLocal && targetLocal && action.Action == network.ActorActionSkill {
		return false
	}
	return true
}

func hurtActionFamilyForActor(actor world.Actor) int {
	if companion.IsMercenary(actor) {
		return sprites.ActionPCHurt
	}
	if res.HasPlayerJobToken(int(actor.Job)) {
		return sprites.ActionPCHurt
	}
	return sprites.ActionNonPCHurt
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

func applyAttackRange(ctx client.Context, attackRange int) {
	attackRange = scene.MaxInt(1, attackRange)
	if ctx.Session != nil {
		ctx.Session.AttackRange = attackRange
	}
	if ctx.World != nil {
		ctx.World.Player.AttackRange = attackRange
	}
}

func (m *WorldMode) applyAttackFailureForDistance(ctx client.Context, failure network.AttackFailureForDistance) {
	attackRange := scene.MaxInt(1, failure.AttackRange)
	applyAttackRange(ctx, attackRange)
	playerX, playerY := scene.CurrentPlayerCell(ctx, time.Now())
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
		actors.UpsertActor(ctx, actor)
	}
	if m.lockedAttackID != failure.TargetID && m.pendingAttack.targetID != failure.TargetID {
		return
	}
	m.pendingAttack = attackIntent{}
	m.lastAttackAt = time.Now()
	if !maps.NormalAttackTargetWithinRange(failure.SourceX, failure.SourceY, failure.TargetX, failure.TargetY, attackRange) {
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
		m.battle().AddLocalRecoveryFloater(ctx, recovery.Amount, visual.recoveryColor, visual.recoveryKind)
		if visual.clearsDeath {
			m.clearLocalDeathStateIfAlive(ctx)
		}
		m.scheduleSound(time.Now(), visual.sfxCandidates()...)
	}
	glog.Debugf("recovery status=%d amount=%d hp=%d/%d sp=%d/%d", recovery.StatusID, recovery.Amount, ctx.Session.Vitals.HP, ctx.Session.Vitals.MaxHP, ctx.Session.Vitals.SP, ctx.Session.Vitals.MaxSP)
}

func currentNormalAttackRange(ctx client.Context) int {
	if ctx.Session != nil && ctx.Session.AttackRange > 0 {
		return ctx.Session.AttackRange
	}
	return normalAttackRangeFromEquippedItems(ctx.Session, ctx.Resources)
}

func (m *WorldMode) startActorDeath(ctx client.Context, id uint32) {
	actor, ok, local := scene.ActorForCombatID(ctx, id)
	if !ok {
		if !local {
			m.effects().RemoveLevel99AuraEffects(id)
			ctx.World.RemoveActor(id)
		}
		return
	}
	now := time.Now()
	deathX, deathY := scene.ActorRenderPosition(actor, now)
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
		m.skillSystem.Pending = skills.PendingTarget{}
		m.ui.pet.Context.Close()
		m.ui.companions.HomunculusContext.Close()
		m.ui.companions.MercenaryContext.Close()
		m.ui.social.PlayerContext.Close()
		m.ui.escapeMenu.OpenDeath(ctx)
	} else {
		actors.UpsertActor(ctx, actor)
	}
	actionFamily := actors.DeathActionFamilyForActor(actor)
	deathDuration := m.actors().ActorActionDuration(ctx, actor, actionFamily, scene.DefaultDeathAnimationDuration)
	persistent := !local && scene.ActorRepresentsPlayer(actor)
	visibleDuration := deathDuration
	if !local && !persistent {
		visibleDuration = maxDuration(deathDuration, scene.NonPCDeathFadeDuration)
	}
	if local || persistent {
		m.startHeldCombatAnimation(ctx, id, actionFamily, now, deathDuration)
	} else {
		m.startCombatAnimation(ctx, id, actionFamily, now, visibleDuration)
	}
	companion.SetActorMotion(ctx, id, companion.MotionDead, 0, time.Time{})
	if !local {
		if m.actorSystem.ActorDeaths == nil {
			m.actorSystem.ActorDeaths = make(map[uint32]time.Time)
		}
		if persistent {
			m.actorSystem.ActorDeaths[id] = time.Time{}
		} else {
			m.actorSystem.ActorDeaths[id] = now.Add(visibleDuration)
		}
		if m.scriptHighlight.id == id {
			m.clearScriptHighlight()
		}
		if m.battleSystem.ActorLife != nil {
			if life, ok := m.battleSystem.ActorLife[id]; ok {
				life.HP = 0
				m.battleSystem.ActorLife[id] = life
			}
		}
	}
	if !local && actor.HasObjectType && actor.ObjectType != scene.ActorObjectTypePC {
		m.effects().RemoveLevel99AuraEffects(id)
	}
	glog.Debugf("actor death id=%d job=%d local=%t persistent=%t action=%d death_ms=%d remove_ms=%d", id, actor.Job, local, persistent, actionFamily, deathDuration.Milliseconds(), visibleDuration.Milliseconds())
}

func (m *WorldMode) applyActorResurrection(ctx client.Context, resurrection network.ActorResurrection) {
	if resurrection.ID == 0 {
		return
	}
	if scene.IsLocalActor(ctx, resurrection.ID) {
		m.clearLocalDeathState(ctx)
	} else {
		m.clearActorDeath(resurrection.ID)
	}
	companion.SetActorMotion(ctx, resurrection.ID, companion.MotionStand, 0, time.Time{})
	glog.Debugf("actor resurrected id=%d type=%d", resurrection.ID, resurrection.Type)
}

func (m *WorldMode) clearActorDeath(id uint32) {
	delete(m.actorSystem.ActorDeaths, id)
	delete(m.actorSystem.ActorAnims, id)
	delete(m.actorSoundFrames, id)
}

func (m *WorldMode) drawAttackFocusMarker(screen *render.Frame, ctx client.Context, now time.Time, entries []scene.ActorDrawEntry) {
	m.drawActorFocusMarker(screen, ctx, now, entries, m.attackFocusID, &m.attackFocusStart, false)
}

func (m *WorldMode) drawScriptHighlightMarker(screen *render.Frame, ctx client.Context, now time.Time, entries []scene.ActorDrawEntry) {
	if actorIDsMatch(ctx, m.scriptHighlight.id, m.attackFocusID) {
		return
	}
	m.drawActorFocusMarker(screen, ctx, now, entries, m.scriptHighlight.id, &m.scriptHighlight.started, true)
}

func (m *WorldMode) drawActorFocusMarker(screen *render.Frame, ctx client.Context, now time.Time, entries []scene.ActorDrawEntry, targetID uint32, started *time.Time, matchLocalAlias bool) {
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
		if entry.Actor.ID != targetID && !(matchLocalAlias && entry.IsPlayer && scene.IsLocalActor(ctx, targetID)) {
			continue
		}
		x, y := actors.ActorPickBoundsCenter(entry.ScreenX, entry.ScreenY, entry.Scale)
		var opts render.DrawImageOptions
		opts.GeoM.Translate(math.Round(x-frame.AnchorX), math.Round(y-frame.AnchorY))
		opts.Filter = scene.SpriteDrawFilter()
		screen.DrawImage(frame.Image, &opts)
		return
	}
}

func actorIDsMatch(ctx client.Context, a, b uint32) bool {
	if a == 0 || b == 0 {
		return false
	}
	return a == b || (scene.IsLocalActor(ctx, a) && scene.IsLocalActor(ctx, b))
}

func (m *WorldMode) clearScriptHighlight() {
	if m != nil {
		m.scriptHighlight = actorHighlight{}
	}
}

func (m *WorldMode) battle() *battle.System {
	m.battleSystem.Bind(battle.Bindings{
		Windows:              &m.ui.battle,
		Console:              &m.ui.console,
		Minimap:              &m.ui.minimap,
		Map:                  &m.mapScene,
		Actors:               m.actors(),
		Effects:              m.effects(),
		Progress:             &m.npcSystem.Progress,
		SetActorAction:       m.setCombatActorAction,
		ScheduleSoundAtActor: m.scheduleSoundAtActor,
		MonsterPreview:       m.monsterInfoPreviewImage,
	})
	return &m.battleSystem
}

func (m *WorldMode) applyStatusEffectChange(ctx client.Context, change network.StatusEffectChange) {
	if ctx.Session == nil || change.StatusID == 0xFFFF {
		return
	}
	if m.applyPushCartStatus(ctx, change) {
		return
	}
	m.actors().ApplyFalconStatus(ctx, change)
	m.stopPerformance(ctx, change)
	if actorID := battle.ApplyActorOpt3StateStatus(ctx, change); actorID != 0 {
		m.actors().SyncActorSpiritEffects(ctx, actorID)
		if change.StatusID == db.StatusLkconcentration {
			if scene.IsLocalActor(ctx, actorID) {
				actorID = scene.LocalSkillTarget(ctx)
			}
			m.actors().SetActorStateEffect(ctx, actorID, effects.LKConcentration, change.Active)
		}
	}
	m.battle().ApplyTrickDeadStatus(ctx, change)
	m.battle().ApplyTaekwonStatus(ctx, change)
	m.applyCloseConfineStatus(ctx, change)
	if change.StatusID == db.StatusWzSightblaster {
		id := change.ActorID
		if id == 0 || scene.IsLocalActor(ctx, id) {
			id = scene.LocalSkillTarget(ctx)
		}
		m.actors().SetActorStateEffect(ctx, id, effects.SightBlaster, change.Active)
	}
	if change.StatusID == db.StatusSmaReady && change.Active {
		id := change.ActorID
		if id == 0 {
			id = scene.LocalSkillTarget(ctx)
		}
		m.effects().Add(ctx, effects.SmaReady, id)
	}
	m.battle().ApplyTaekwonNightStatus(ctx, change)
	if !battle.ApplyLocalStatusEffectChange(ctx.Session, change, time.Now()) {
		return
	}
	if change.StatusID == db.StatusDevil1 {
		m.syncStarDevil(ctx)
	}
	if !change.Active {
		if change.StatusID == db.StatusCashBossAlarm {
			m.ui.minimap.ClearBossMarker()
		}
		glog.Debugf("status effect inactive id=%d actor=%d", change.StatusID, change.ActorID)
		return
	}
	glog.Debugf("status effect active id=%d actor=%d duration_ms=%d", change.StatusID, change.ActorID, change.Duration.Milliseconds())
}
