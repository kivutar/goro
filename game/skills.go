package game

import (
	"slices"
	"strconv"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) startSkillActionAnimation(ctx client.Context, id uint32, actor worldstate.Actor, spec effects.SkillActionSpec, started time.Time, duration time.Duration) {
	anim := spec.ActorAnimationForActorWithResources(ctx.Resources, actor, started, duration)
	if delay, ok := m.actorSystem.ComboDelays[id]; ok && anim.Combo {
		anim.Duration = delay
	}
	m.setActorAction(ctx, id, anim)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, id) {
		return
	}
	m.setActorAction(ctx, ctx.Session.AccountID, anim)
	m.setActorAction(ctx, ctx.Session.CharID, anim)
}

func (m *WorldMode) applySkillNameBubble(ctx client.Context, sourceID uint32, skillID uint16, now time.Time) {
	if skillID == 0 || skills.SkillNameBubbleExcluded(skillID) {
		return
	}
	source, ok, local := scene.ActorForCombatID(ctx, sourceID)
	if !ok || !skills.ActorSupportsSkillNameBubble(source, local) {
		return
	}
	m.actors().ApplySpeechBubble(ctx, network.ChatMessage{GID: sourceID, Text: skills.SkillNameBubbleText(ctx, skillID)}, now)
}

func (m *WorldMode) applySkillNoDamageNotify(ctx client.Context, notify network.SkillNoDamageNotify) {
	if notify.Result == 0 {
		return
	}
	if notify.SkillID == db.SkillHWSouldrain || notify.SkillID == db.SkillPFSoulchange || (notify.SkillID >= db.SkillSGSunWarm && notify.SkillID <= db.SkillSGStarWarm) {
		if _, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); !ok {
			return
		}
		if _, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); !ok {
			return
		}
	}
	now := time.Now()
	if notify.SkillID == db.SkillPRSanctuary || notify.SkillID == db.SkillMGSrecovery {
		if actor, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); ok && notify.Amount > 0 {
			kind := battle.FloaterRecoveryHP
			if notify.SkillID == db.SkillMGSrecovery {
				kind = battle.FloaterRecoverySP
			}
			m.battle().AddDamageFloater(notify.TargetID, actor.X, actor.Y, strconv.FormatUint(uint64(notify.Amount), 10), battle.DamageFloaterWhite, kind, now)
		}
		return
	}
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	m.startSkillNoDamageSourceAnimation(ctx, notify, now)
	if m.applyHunterTrapSkill(ctx, notify, now) {
		return
	}
	if notify.SkillID == db.SkillPaSacrifice {
		// Activation is silent; each sacrificed attack launches BASH3D.
		return
	}
	m.actors().ApplyFalconSkillNoDamageNotify(ctx, notify, now)
	m.applyClownGypsySkill(ctx, notify, now)
	m.effects().ApplyAlchemistSkill(ctx, notify, now)
	m.applyHighWizardSkill(ctx, notify, now)
	m.effects().ApplyProfessorSkill(ctx, notify, now)
	if notify.SkillID == db.SkillNJTatamigaeshi {
		m.scheduleSound(now, "effect\\다다미뒤집기.wav")
	}
	if notify.SkillID == db.SkillGSGlittering {
		m.scheduleSound(now, "effect\\플립.wav")
	} else if notify.SkillID == db.SkillGSCracker {
		m.scheduleSound(now, "effect\\크래커.wav")
	}
	m.applyTaekwonSkill(ctx, notify, now)
	m.applyStarGladiatorSkill(ctx, notify, now)
	m.applyWinkCharm(ctx, notify)
	for _, sound := range db.SkillEffects[notify.SkillID].SFX {
		m.scheduleSoundAtActor(now, notify.TargetID, sound)
	}
	effectIDs := effects.SkillEffectIDs(notify.SkillID)
	if notify.SkillID == db.SkillALHeal {
		effectIDs = []int{effects.HealEffectID(notify.Amount)}
	}
	for _, effectID := range effectIDs {
		actorID := notify.TargetID
		// The status packet may already have installed the persistent aura.
		if effectID == effects.SteelBody || effectID == effects.AsuraBody || effectID == effects.BabyBody || effectID == effects.Assumptio {
			if scene.IsLocalActor(ctx, actorID) {
				actorID = scene.LocalSkillTarget(ctx)
			}
			if m.effects().HasActive(effectID, actorID, now) {
				continue
			}
		}
		if m.effects().AddBetween(ctx, effectID, actorID, notify.SourceID, now) {
			glog.Debugf("skill effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range effects.SkillEffectOnCasterIDs(notify.SkillID) {
		if m.effects().AddAt(ctx, effectID, notify.SourceID, now) {
			glog.Debugf("skill caster effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range effects.SkillSuccessEffectIDs(notify.SkillID) {
		if notify.SkillID == db.SkillALLResurrection && effectID == effects.Revive {
			if actor, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); !ok || !res.HasPlayerJobToken(int(actor.Job)) {
				continue
			}
		}
		if m.effects().AddBetween(ctx, effectID, notify.TargetID, notify.SourceID, now) {
			glog.Debugf("skill success effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range effects.SkillSuccessEffectSelfIDs(notify.SkillID) {
		if effectID == effects.RedBody || effectID == effects.LKConcentration || effectID == effects.Gumgang {
			actorID := notify.SourceID
			if scene.IsLocalActor(ctx, actorID) {
				actorID = scene.LocalSkillTarget(ctx)
			}
			if effectID == effects.LKConcentration || effectID == effects.Gumgang {
				m.effects().Remove(effectID, actorID)
			}
			m.actors().SetActorStateEffect(ctx, actorID, effectID, true)
			continue
		}
		if effectID == effects.SpearQuicken || effectID == effects.AuraBlade || effectID == effects.Overthrust {
			m.effects().Remove(effectID, notify.SourceID)
		}
		if m.effects().AddAt(ctx, effectID, notify.SourceID, now) {
			glog.Debugf("skill success self effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	m.actors().ApplySoulLinkerMonsterSkill(ctx, notify)
}

func (m *WorldMode) applySkillCastNotify(ctx client.Context, notify network.SkillCastNotify) {
	now := time.Now()
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	if notify.SkillID == db.SkillChSoulcollect {
		m.effects().AddAt(ctx, effects.Portal5, notify.SourceID, now)
	}
	if notify.SkillID == db.SkillCRAciddemonstration && notify.SourceID != notify.TargetID {
		_, sourceOK, _ := scene.ActorForCombatID(ctx, notify.SourceID)
		if _, targetOK, _ := scene.ActorForCombatID(ctx, notify.TargetID); sourceOK && targetOK {
			m.effects().AddBetween(ctx, effects.ThrowItem4, notify.TargetID, notify.SourceID, now)
		}
	}
	if notify.DelayTime == 0 {
		if notify.SkillID == db.SkillTKHighjump {
			m.startSkillCastSourceAnimation(ctx, notify, 0, now)
		}
		return
	}
	duration := time.Duration(notify.DelayTime) * time.Millisecond
	m.startSkillCastSourceAnimation(ctx, notify, duration, now)
	if !effects.SkillHidesCastBar(notify.SkillID) {
		m.battle().StartActorCastBar(ctx, notify.SourceID, duration, now)
	}
	m.effects().AddSkillCastEffects(ctx, notify.SkillID, notify.Property, notify.SourceID, notify.TargetID, int(notify.X), int(notify.Y), duration, now, "server")
}

func (m *WorldMode) startSkillNoDamageSourceAnimation(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	source, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID)
	if !ok {
		return
	}
	m.faceSkillSource(ctx, notify.SourceID, notify.TargetID, 0, 0)
	if notify.SkillID == db.SkillKNAutocounter {
		// Native SetAction's second argument is speed; the frozen motion is zero.
		m.setCombatActorAction(ctx, notify.SourceID, scene.ActorAnimation{
			ActionFamily: sprites.AttackActionFamily(ctx.Resources, source),
			Started:      now, Duration: 1500 * time.Millisecond,
			HasPlay: true, HasFixedMotion: true, FixedMotion: 0, AutoCounter: true,
		})
		return
	}
	if notify.SkillID == db.SkillHWSouldrain {
		m.setCombatActorAction(ctx, notify.SourceID, scene.ActorAnimation{
			ActionFamily: sprites.ActionPCAttack3, Started: now, Duration: time.Second,
			Play: true, HasPlay: true, Length: 4, HasLength: true,
			Speed: 10 * db.EffectFrameDuration, HasSpeed: true,
			KeepMovement: true,
		})
		return
	}
	if notify.SkillID == db.SkillLKAurablade {
		m.setCombatActorAction(ctx, notify.SourceID, scene.ActorAnimation{
			ActionFamily: sprites.ActionPCReadyFight, Started: now, Duration: time.Second,
			Play: true, HasPlay: true, Length: 6, HasLength: true,
			Speed: 10 * db.EffectFrameDuration, HasSpeed: true,
		})
		return
	}
	m.startSkillActionAnimation(ctx, notify.SourceID, source, effects.SkillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
}

func (m *WorldMode) startSkillCastSourceAnimation(ctx client.Context, notify network.SkillCastNotify, duration time.Duration, now time.Time) {
	m.faceSkillSource(ctx, notify.SourceID, notify.TargetID, int(notify.X), int(notify.Y))
	m.startSkillSourceCastAnimation(ctx, notify.SourceID, notify.SkillID, duration, now)
	local := scene.IsLocalActor(ctx, notify.SourceID)
	for id, anim := range m.actorSystem.ActorAnims {
		if id == notify.SourceID || (local && scene.IsLocalActor(ctx, id)) {
			for step := &anim; step != nil; step = step.Next {
				step.Casting = true
			}
			m.actorSystem.ActorAnims[id] = anim
		}
	}
}

func (m *WorldMode) applySkillCastCancel(ctx client.Context, actorID uint32) {
	if actorID == 0 {
		return
	}
	m.battle().ClearActorCastBar(ctx, actorID)
	m.effects().RemoveSkillCastEffects(ctx, actorID)
	local := scene.IsLocalActor(ctx, actorID)
	for id, anim := range m.actorSystem.ActorAnims {
		if anim.Casting && (id == actorID || (local && scene.IsLocalActor(ctx, id))) {
			m.clearActorAction(ctx, id)
		}
	}
}

func (m *WorldMode) startSkillSourceCastAnimation(ctx client.Context, sourceID uint32, skillID uint16, duration time.Duration, now time.Time) {
	source, ok, _ := scene.ActorForCombatID(ctx, sourceID)
	if !ok {
		return
	}
	if skillID == db.SkillTKHighjump && res.HasPlayerJobToken(int(source.Job)) {
		m.startSkillActionAnimation(ctx, sourceID, source, effects.TaekwonJumpSequence(true), now, duration)
		if scene.IsLocalActor(ctx, sourceID) {
			m.effects().Remove(effects.LandBody, ctx.World.Player.ID)
			m.effects().Remove(effects.JumpBody, ctx.World.Player.ID)
			sourceID = scene.LocalSkillTarget(ctx)
		}
		m.effects().Remove(effects.LandBody, sourceID)
		m.effects().Remove(effects.JumpBody, sourceID)
		m.effects().AddAt(ctx, effects.JumpBody, sourceID, now)
		m.effects().AddAt(ctx, effects.Peong, sourceID, now)
		return
	}
	if spec, ok := effects.GunslingerCastAction(skillID, now); ok && res.HasPlayerJobToken(int(source.Job)) {
		m.startSkillActionAnimation(ctx, sourceID, source, spec, now, duration)
		return
	}
	if effects.NinjaCastingSkill(skillID) && res.HasPlayerJobToken(int(source.Job)) {
		spec := effects.SkillActionSpec{Defined: true, Action: effects.SkillActorActionSkill, HasFrame: true, Frame: 2 + int(now.UnixNano()&1), Duration: 500 * time.Millisecond}
		m.startSkillActionAnimation(ctx, sourceID, source, spec, now, duration)
		return
	}
	if skillID == db.SkillDCWinkcharm {
		spec := effects.DanceSkillActionSpec()
		spec.Next = &effects.IdleSkillActionSpec
		m.startSkillActionAnimation(ctx, sourceID, source, spec, now, duration)
		return
	}
	if (skillID == db.SkillCRSlimpitcher || skillID == db.SkillCRAciddemonstration) && res.HasPlayerJobToken(int(source.Job)) {
		m.startSkillActionAnimation(ctx, sourceID, source, effects.SkillActionSpec{
			Defined: true, Action: effects.SkillActorActionAttack1, Play: true, Length: 5,
			Speed: 10 * db.EffectFrameDuration, Duration: time.Second, Next: &effects.IdleSkillActionSpec,
		}, now, time.Second)
		return
	}
	actionFamily := skillCastActionFamilyForActor(source, skillID)
	if skillID == db.SkillBSHammerfall {
		actionFamily = sprites.AttackActionFamily(ctx.Resources, source)
	}
	if (skillID == db.SkillSNSharpshooting || skillID == db.SkillCGArrowvulcan || skillID == db.SkillChSoulcollect || skillID == db.SkillMOCallspirits || skillID == db.SkillMOKitranslation) && res.HasPlayerJobToken(int(source.Job)) {
		m.setCombatActorAction(ctx, sourceID, scene.ActorAnimation{
			ActionFamily: actionFamily, Started: now, Duration: duration,
			HasPlay: true, HasFixedMotion: true, FixedMotion: 0,
		})
		return
	}
	if actionFamily == scene.SpriteActionPickup {
		duration = m.actors().ActorActionDuration(ctx, source, actionFamily, 300*time.Millisecond)
	}
	m.startCombatAnimation(ctx, sourceID, actionFamily, now, duration)
}

func (m *WorldMode) faceSkillSource(ctx client.Context, sourceID, targetID uint32, cellX, cellY int) {
	source, sourceOK, sourceLocal := scene.ActorForCombatID(ctx, sourceID)
	if !sourceOK {
		return
	}
	if target, targetOK, _ := scene.ActorForCombatID(ctx, targetID); targetOK {
		m.faceCombatSource(ctx, source, sourceLocal, target)
		return
	}
	if x, y, ok := m.skillUnitCell(targetID); ok {
		cellX, cellY = x, y
	}
	if cellX == 0 && cellY == 0 {
		return
	}
	dir := actors.DirectionFromDelta(source.X, source.Y, cellX, cellY, source.Dir)
	if sourceLocal {
		ctx.World.Player.Dir = dir
		ctx.World.Dir = dir
		return
	}
	source.Dir = dir
	actors.UpsertActor(ctx, source)
}

func (m *WorldMode) applyGroundSkillNotify(ctx client.Context, notify network.GroundSkillNotify) {
	if m.applyHunterGroundSkill(ctx, notify, time.Now()) || m.applyPerformanceGroundSkill(ctx, notify, time.Now()) || m.applyCreatorGroundSkill(ctx, notify, time.Now()) {
		return
	}
	if notify.SkillID == db.SkillACShower {
		now := time.Now()
		m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
		if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
			m.startSkillActionAnimation(ctx, notify.SourceID, actor, effects.SkillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
			family := skillActionFamilyForActorWithResources(ctx.Resources, actor, notify.SkillID)
			if action, ok := m.actors().ActorResolvedAction(ctx, actor, family); ok {
				act := m.actors().ActorActionACT(ctx, actor)
				motion := actionAttackMarkerMotion(act, action)
				delay := combatMotionDelayFromAction(action, scene.DefaultAttackAnimationDuration, motion)
				m.scheduleSkillActionSoundAtActor(now.Add(delay), notify.SourceID, notify.SkillID, actionSFXCandidatesForActor(actor, act, action, motion)...)
			}
		}
		m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
		m.effects().AddArrowShower(ctx, notify.SourceID, float64(notify.X), float64(notify.Y), true, now)
		return
	}
	switch notify.SkillID {
	case db.SkillALPneuma, db.SkillALWarp, db.SkillMGSafetywall, db.SkillMGFirewall, db.SkillPRSanctuary, db.SkillHPBasilica,
		db.SkillWZFirepillar, db.SkillWZIcewall, db.SkillWZQuagmire,
		db.SkillAMDemonstration, db.SkillAMCannibalize, db.SkillAMSpheremine, db.SkillHWGravitation,
		db.SkillPFFogwall, db.SkillPFSpiderweb, db.SkillGSGrounddrift:
		now := time.Now()
		m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
		if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
			m.startSkillActionAnimation(ctx, notify.SourceID, actor, effects.SkillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
		}
		if notify.SkillID == db.SkillHWGravitation {
			m.scheduleSound(now, "effect\\wizard_earthspike.wav")
		}
		if notify.SkillID == db.SkillHPBasilica {
			m.scheduleSoundAtActor(now, notify.SourceID, "effect\\바실리카.wav")
		}
		// The subsequent skill-unit entry owns the ground visual and its lifetime.
		return
	}
	// rAthena sends Battle Chant's cast notification at 0,0 and follows it with
	// one UNT_GOSPEL entry per field cell. Play EF_BOTTOM_GOSPEL once on the
	// caster here; the unit entries own the persistent ground visuals.
	if notify.SkillID == db.SkillPaGospel && notify.X == 0 && notify.Y == 0 {
		for _, effectID := range effects.SkillEffectIDs(notify.SkillID) {
			if m.effects().AddIfMissing(ctx, effectID, notify.SourceID) {
				glog.Debugf("ground skill caster effect skill=%d src=%d level=%d effect=%d", notify.SkillID, notify.SourceID, notify.Level, effectID)
			}
		}
		return
	}
	effectIDs := effects.SkillGroundEffectIDs(notify.SkillID)
	if len(effectIDs) == 0 {
		effectIDs = effects.SkillEffectIDs(notify.SkillID)
	}
	if len(effectIDs) == 0 {
		return
	}
	now := time.Now()
	switch notify.SkillID {
	case db.SkillPRMagnus, db.SkillPRBenedictio, db.SkillMGThunderstorm,
		db.SkillBSHammerfall, db.SkillHWGanbantein,
		db.SkillWZStormgust, db.SkillWZVermilion, db.SkillWZMeteor, db.SkillWZHeavendrive,
		db.SkillSAVolcano, db.SkillSADeluge, db.SkillSAViolentgale, db.SkillSALandprotector:
		if notify.SkillID != db.SkillWZMeteor {
			m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
		}
		// Falling meteors must not interrupt a walk begun after casting.
		if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok &&
			(notify.SkillID != db.SkillWZMeteor || !scene.ActorIsMovingAt(actor, now)) {
			spec := effects.SkillAction(notify.SkillID)
			if notify.SkillID == db.SkillWZMeteor {
				// Each meteor holds the release pose instead of turning toward its impact.
				spec = effects.SkillActionSpec{Defined: true, Action: effects.SkillActorActionSkill, HasFrame: true}
				actor.Dir = 0
				if action, ok := m.actors().ActorResolvedAction(ctx, actor, sprites.ActionPCSkill); ok {
					spec.Frame = max(0, len(action.Animations)-1)
				}
			}
			m.startSkillActionAnimation(ctx, notify.SourceID, actor, spec, now, scene.DefaultAttackAnimationDuration)
		}
		if notify.SkillID == db.SkillHWGanbantein {
			m.scheduleSound(now, "effect\\EF_FireWall.wav")
		}
		for _, effectID := range effectIDs {
			m.effects().AddCellLifetime(ctx, effectID, 0, int(notify.X), int(notify.Y), now, 0, false)
		}
		return
	}
	m.actors().ApplyFalconGroundSkillNotify(ctx, notify, now)
	for _, effectID := range effectIDs {
		if m.effects().AddCellIfMissing(ctx, effectID, int(notify.X), int(notify.Y), now) {
			glog.Debugf("ground skill effect skill=%d src=%d level=%d cell=%d,%d effect=%d", notify.SkillID, notify.SourceID, notify.Level, notify.X, notify.Y, effectID)
		}
	}
}

func (m *WorldMode) applySkillUnitEntry(ctx client.Context, entry network.SkillUnitEntry) {
	if entry.HasMessage {
		m.skills().ApplyGraffitiEntry(ctx, entry)
		return
	}
	m.skills().RemoveGraffiti(entry.ID)
	if m.skills().ApplySkillUnitModelEntry(entry) {
		if entry.UnitID == 148 { // Shock Wave Trap owns its looping STR.
			m.effects().AddCellLifetime(ctx, effects.Shockwave, entry.ID, int(entry.X), int(entry.Y), time.Now(), effects.SkillUnitEffectFallbackDuration, true)
		}
		return
	}
	m.effects().AddSkillUnitEffects(ctx, entry)
}

func (m *WorldMode) applySkillUnitLookChange(ctx client.Context, look network.ActorLookChange) bool {
	if look.Type != 0 || look.ID == 0 {
		return false
	}
	if m.skills().ApplyUsedTrapLookChange(ctx, look) {
		return true
	}
	if m.skills().ApplyTrapModelLookChange(look) {
		return true
	}
	effectIDs := effects.SkillUnitEffectIDs(uint16(look.Value))
	if len(effectIDs) == 0 {
		return false
	}
	x, y, ok := m.skillUnitCell(look.ID)
	if !ok {
		return false
	}
	m.effects().RemoveUnit(look.ID)
	now := time.Now()
	for _, effectID := range effectIDs {
		if m.effects().AddCellLifetime(ctx, effectID, look.ID, x, y, now, effects.SkillUnitEffectFallbackDuration, true) {
			glog.Debugf("skill unit effect changed id=%d unit=%d cell=%d,%d effect=%d", look.ID, look.Value, x, y, effectID)
		}
	}
	return true
}

func (m *WorldMode) applySkillUnitDisappear(disappear network.SkillUnitDisappear) {
	if disappear.ID == 0 {
		return
	}
	m.skills().RemoveGraffiti(disappear.ID)
	removedEffects := m.effects().RemoveUnit(disappear.ID)
	removedModel := m.skills().RemoveSkillUnitModelOnly(disappear.ID)
	if removedEffects || removedModel {
		glog.Debugf("skill unit effect removed id=%d", disappear.ID)
	}
}

func (m *WorldMode) skillUnitCell(id uint32) (int, int, bool) {
	if x, y, ok := m.skills().SkillUnitModelCell(id); ok {
		return x, y, true
	}
	for _, effect := range m.effects().Active {
		if effect.ActorID == id {
			return effect.X, effect.Y, true
		}
	}
	return 0, 0, false
}

func (m *WorldMode) applySkillFailAck(ctx client.Context, ack network.SkillFailAck) {
	if ack.Result != 0 {
		return
	}
	m.battle().ClearSenseRequest(ack.SkillID)
	message := skills.SkillFailMessage(ack)
	glog.Debugf("skill fail ack skill=%d num=%d item=%d result=%d cause=%d msg=%q", ack.SkillID, ack.Number, ack.ItemID, ack.Result, ack.Cause, message)
	m.ui.console.AddErrorMessage("%s", message)
}

func (m *WorldMode) applyClownGypsySkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	source, target := notify.SourceID, notify.TargetID
	if scene.IsLocalActor(ctx, source) {
		source = scene.LocalSkillTarget(ctx)
	}
	if scene.IsLocalActor(ctx, target) {
		target = scene.LocalSkillTarget(ctx)
	}
	switch notify.SkillID {
	case db.SkillCGMoonlit:
		if _, ok, _ := scene.ActorForCombatID(ctx, source); ok {
			m.actors().SetActorStateEffect(ctx, source, effects.Moonlit, true)
		}
	case db.SkillCGMarionette:
		if _, ok, _ := scene.ActorForCombatID(ctx, source); !ok {
			return
		}
		if _, ok, _ := scene.ActorForCombatID(ctx, target); !ok {
			return
		}
		m.effects().RemoveMarionetteLinks(source)
		m.effects().AddPersistentBetween(ctx, effects.MarionetteLink, source, target, now)
		m.actors().SetActorStateEffect(ctx, source, effects.PinkBody, true)
		m.actors().SetActorStateEffect(ctx, target, effects.PinkBody, true)
	}
}

// The initial shot uses ACT 88. Each additional hit restarts the normal
// weapon attack, as AM_WILL_BE_ATTACK does in the original client.
func (m *WorldMode) startArrowVulcanHits(ctx client.Context, action network.ActorActionNotify, hitAt time.Time) {
	if action.SkillID != db.SkillCGArrowvulcan || skillSplashAction(action) || scene.ActionVisualHitCount(action) < 2 {
		return
	}
	anim, ok := m.actorSystem.ActorAnims[action.SourceID]
	source, sourceOK, _ := scene.ActorForCombatID(ctx, action.SourceID)
	if !ok || !sourceOK {
		return
	}
	anim.Speed = m.actors().ActorActionFrameDelay(ctx, source, anim.ActionFamily, anim.Duration)
	anim.HasSpeed = anim.Speed > 0
	count := min(9, scene.ActionVisualHitCount(action))
	next := readyFightAnimation(hitAt.Add(time.Duration(count) * scene.MultiHitDelay))
	for hit := count - 1; hit >= 1; hit-- {
		at := hitAt.Add(time.Duration(hit) * scene.MultiHitDelay)
		rest := scene.ActorAnimation{
			ActionFamily: sprites.AttackActionFamily(ctx.Resources, source),
			Started:      at.Add(24 * time.Millisecond), Duration: scene.MultiHitDelay - 24*time.Millisecond,
			Play: true, HasPlay: true, Speed: 24 * time.Millisecond, HasSpeed: true,
			FrameOffset: 1, HasFrameOffset: true, Next: next,
		}
		next = &scene.ActorAnimation{
			ActionFamily: rest.ActionFamily, Started: at, Duration: 24 * time.Millisecond,
			HasPlay: true, HasFixedMotion: true, FixedMotion: 4, Next: &rest,
		}
	}
	anim.Duration = next.Started.Sub(anim.Started)
	anim.Next = next
	m.setCombatActorAction(ctx, action.SourceID, anim)
}

func (m *WorldMode) applyCreatorGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	if notify.SkillID != db.SkillCRSlimpitcher && notify.SkillID != db.SkillCRCultivation {
		return false
	}
	m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
	if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, effects.SkillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
	}
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	if notify.SkillID == db.SkillCRCultivation {
		m.effects().AddAt(ctx, effects.BeginSpell6, notify.SourceID, now)
		return true
	}
	effectID := effects.Slim
	if notify.Level >= 10 {
		effectID = effects.Slim3
	} else if notify.Level >= 6 {
		effectID = effects.Slim2
	}
	m.effects().AddCellLifetime(ctx, effectID, 0, int(notify.X), int(notify.Y), now, 0, false)
	// The native sound is centered at a fixed distance of 150, with a 40-unit minimum.
	m.scheduleSoundVolume(now, 40.0/150, "assulter_attack.wav")
	return true
}

func (m *WorldMode) applyWinkCharm(ctx client.Context, notify network.SkillNoDamageNotify) {
	if notify.SkillID != db.SkillDCWinkcharm {
		return
	}
	if _, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); !ok {
		return
	}
	if _, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); !ok {
		return
	}
	m.applyEmotionNotify(ctx, network.EmotionNotify{GID: notify.SourceID, Type: 30}) // ET_CHUP
}

func (m *WorldMode) applyHighWizardSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	switch notify.SkillID {
	case db.SkillHWSouldrain:
		m.effects().AddBetween(ctx, effects.EnergyDrain2, notify.SourceID, notify.TargetID, now)
	case db.SkillHWMagicpower:
		id := notify.TargetID
		if scene.IsLocalActor(ctx, id) {
			id = scene.LocalSkillTarget(ctx)
		}
		if _, ok, _ := scene.ActorForCombatID(ctx, id); ok {
			m.effects().Remove(effects.LightBlade, id)
			m.actors().SetActorStateEffect(ctx, id, effects.LightBlade, true)
			m.scheduleSound(now, "effect\\마법력 증폭.wav")
		}
	}
}

func (m *WorldMode) applyHunterTrapSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) bool {
	if notify.SkillID != db.SkillHTRemovetrap && notify.SkillID != db.SkillHTSpringtrap {
		return false
	}
	x, y, ok := m.skillUnitCell(notify.TargetID)
	if ok {
		for _, effectID := range effects.SkillEffectIDs(notify.SkillID) {
			// The following disappearance packet must not cut off the burst.
			m.effects().AddCellLifetime(ctx, effectID, 0, x, y, now, 0, false)
		}
		if notify.SkillID == db.SkillHTSpringtrap {
			m.actors().StartFalconAttackAt(ctx, notify.SourceID, x, y, now)
		}
	}
	return true
}

func (m *WorldMode) applyHunterGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	switch notify.SkillID {
	case db.SkillHTSkidtrap, db.SkillHTLandmine, db.SkillHTAnklesnare, db.SkillHTShockwave,
		db.SkillHTSandman, db.SkillHTFlasher, db.SkillHTFreezingtrap, db.SkillHTBlastmine,
		db.SkillHTClaymoretrap, db.SkillHTTalkiebox, db.SkillHTDetecting:
	default:
		return false
	}
	m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
	if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, effects.SkillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
	}
	m.actors().ApplyFalconGroundSkillNotify(ctx, notify, now)
	for _, effectID := range effects.SkillGroundEffectIDs(notify.SkillID) {
		m.effects().AddCellLifetime(ctx, effectID, 0, int(notify.X), int(notify.Y), now, 0, false)
	}
	return true
}

func (m *WorldMode) applyPerformanceGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	if !effects.IsBardPerformance(notify.SkillID) && !effects.IsDancerPerformance(notify.SkillID) && notify.SkillID != db.SkillCGHermode && notify.SkillID != db.SkillCGMoonlit {
		return false
	}
	if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, effects.SkillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
	}
	for _, sound := range db.SkillEffects[notify.SkillID].SFX {
		m.scheduleSoundAtActor(now, notify.SourceID, sound)
	}
	if notify.SkillID == db.SkillCGMoonlit {
		m.applyClownGypsySkill(ctx, network.SkillNoDamageNotify{SkillID: notify.SkillID, SourceID: notify.SourceID}, now)
	}
	// Skill-unit packets own each symbol and its lifetime.
	return true
}

func (m *WorldMode) stopPerformance(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusBdplaying || change.Active {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	ids := []uint32{id}
	if scene.IsLocalActor(ctx, id) {
		ids = []uint32{ctx.Session.AccountID, ctx.Session.CharID}
	}
	for _, id := range ids {
		if anim, ok := m.actorSystem.ActorAnims[id]; ok && anim.Loop && anim.ActionFamily == sprites.ActionPCSkill && anim.HasFrameOffset && anim.FrameOffset == 1 && anim.Length == 3 {
			m.clearActorAction(ctx, id)
		}
	}
}

func (m *WorldMode) applyCloseConfineStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusRgCconfineM && change.StatusID != db.StatusRgCconfineS {
		return
	}
	id := change.ActorID
	if id == 0 || scene.IsLocalActor(ctx, id) {
		id = scene.LocalSkillTarget(ctx)
	}
	m.actors().SetActorStateEffect(ctx, id, effects.NPCStop2, change.Active)
	if change.Active && change.HasDuration {
		now := time.Now()
		for i := range m.effects().Active {
			effect := &m.effects().Active[i]
			if effect.EffectID == effects.NPCStop2 && effect.ActorID == id && !effect.Expired(now) {
				effect.Expires = now.Add(change.Duration)
			}
		}
	}
}

func (m *WorldMode) applyStarGladiatorSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	id := notify.SourceID
	if scene.IsLocalActor(ctx, id) {
		id = scene.LocalSkillTarget(ctx)
	}
	switch notify.SkillID {
	case db.SkillSGFusion:
		if _, ok, _ := scene.ActorForCombatID(ctx, id); !ok {
			return
		}
		m.scheduleSound(now, "effect\\T_변신.wav")
	case db.SkillSGHate:
		if _, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); ok {
			m.scheduleSound(now, "effect\\T_등록.wav")
			m.effects().Remove(effects.Hated, notify.TargetID)
		}
	case db.SkillSGSunComfort, db.SkillSGMoonComfort, db.SkillSGStarComfort:
		m.effects().Remove(effects.FlowerCast, notify.SourceID)
		m.effects().Remove(effects.FlowerCast, id)
	case db.SkillSGSunWarm, db.SkillSGMoonWarm, db.SkillSGStarWarm:
		actor, ok, local := scene.ActorForCombatID(ctx, id)
		if !ok {
			return
		}
		actor.Opt3State |= db.Opt3Warm
		actor.HasState = true
		if local {
			ctx.World.Player.Opt3State, ctx.World.Player.HasState = actor.Opt3State, true
		} else {
			ctx.World.Actors[id] = actor
		}
		m.effects().Remove(effects.DoubleGumgang, id)
		m.effects().Remove(effects.RedLightBody, id)
		m.actors().SyncActorSpiritEffects(ctx, id)
	}
}

// The status is local, but can precede the skill list on login. Reconcile both
// packet orders, restore it on map entry, and ignore repeated status packets.
func (m *WorldMode) syncStarDevil(ctx client.Context) {
	if ctx.Session == nil || ctx.World == nil {
		return
	}
	id, wanted := scene.LocalSkillTarget(ctx), 0
	if status, ok := ctx.Session.Statuses.Active[db.StatusDevil1]; ok && (!status.HasDuration || time.Now().Before(status.ExpiresAt)) {
		if level := scene.SessionSkillLevel(ctx.Session, db.SkillSGDevil); level >= 1 && level <= 10 {
			wanted = effects.Devil1 + level - 1
		}
	}
	now := time.Now()
	for effectID := effects.Devil1; effectID <= effects.Devil10; effectID++ {
		if effectID != wanted {
			m.effects().Remove(effectID, id)
		}
	}
	if wanted == 0 {
		m.effects().Remove(effects.BlackDevil, id)
	} else if !m.effects().HasActive(wanted, id, now) {
		m.actors().SetActorStateEffect(ctx, id, wanted, true)
		m.effects().AddAt(ctx, effects.BlackDevil, id, now)
		m.scheduleSoundVolume(now, .4, "effect\\_blind.wav")
	}
}

func (m *WorldMode) applyTaekwonSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	id := notify.SourceID
	if _, ok, _ := scene.ActorForCombatID(ctx, id); !ok {
		return
	}
	if scene.IsLocalActor(ctx, id) {
		id = scene.LocalSkillTarget(ctx)
	}
	switch notify.SkillID {
	case db.SkillTKRun:
		if m.effects().HasActive(effects.StopRun, id, now) {
			m.effects().AddAt(ctx, effects.StopEffect, id, now)
		} else if !m.effects().HasActive(effects.Run, id, now) {
			m.effects().AddAt(ctx, effects.Run, id, now)
		}
	case db.SkillTKHighjump:
		m.effects().Remove(effects.JumpBody, notify.SourceID)
		m.effects().Remove(effects.JumpBody, id)
		m.scheduledSounds = slices.DeleteFunc(m.scheduledSounds, func(sound scheduledSound) bool {
			return (sound.actorID == notify.SourceID || sound.actorID == id) && sound.at.After(now) && slices.Contains(sound.paths, "effect\\T_회피2.wav")
		})
	case db.SkillTKDodge:
		m.scheduleSoundVolume(now, 4.0/15, "effect\\T_낙법.wav")
	case db.SkillTKMission:
		m.scheduleSoundVolume(now, 4.0/10, "effect\\T_피링.wav")
	case db.SkillTKSevenwind:
		if notify.Amount < 1 || notify.Amount > 7 {
			return
		}
		m.scheduleSound(now, "effect\\T_바람방출.wav")
		for _, effectID := range []int{effects.StormKick3, effects.BeginAsura1 + int(notify.Amount) - 1} {
			m.effects().Remove(effectID, id)
			m.effects().AddAt(ctx, effectID, id, now)
		}
	}
}

func (m *WorldMode) skills() *skills.System {
	m.skillSystem.Bind(skills.Bindings{
		ChangeCart:          &m.ui.changeCartWindow,
		TextPrompt:          &m.ui.skillTextPrompt,
		Console:             &m.ui.console,
		TeleportModal:       &m.ui.teleportModal,
		StarPlaceConfirm:    &m.ui.starPlaceConfirm,
		Error:               m.ui.console.AddErrorMessage,
		RequestWalk:         m.requestWalk,
		WalkBlocked:         func() { m.setWalkCooldown(walkRequestCooldown) },
		RememberSenseTarget: m.battleSystem.RememberSenseTarget,
		ClickActor: func(ctx client.Context, projection scene.Projection, skill session.Skill, x, y int, now time.Time) (worldstate.Actor, bool) {
			return clickedSkillTarget(ctx, projection, skill, x, y, now, m.actorSystem.ActorDeaths)
		},
		NumberBillboard: m.battleSystem.DamageNumberBillboard,
		Effects:         m.effects(),
		Map:             &m.mapScene,
	})
	return &m.skillSystem
}
