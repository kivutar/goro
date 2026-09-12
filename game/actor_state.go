package game

import (
	"fmt"
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) applyActorStateChange(ctx client.Context, change network.ActorStateChange) {
	oldVisualJob := localPlayerVisualJob(ctx)
	oldState, local, ok := applyActorStateSnapshot(ctx, change)
	if !ok {
		return
	}
	m.applyActorEffectStateEffects(ctx, change.ID, oldState, change.EffectState)
	if (oldState^change.EffectState)&db.EffectStateHide != 0 {
		m.addWorldEffect(ctx, effectSummonSlave, change.ID)
		actor, _, _ := actorForCombatID(ctx, change.ID)
		if change.EffectState&db.EffectStateHide != 0 && actor.Job != db.JobNinja {
			m.addWorldEffect(ctx, effectBashBegin, change.ID)
		}
	}
	if change.EffectState&actorStealthMask != 0 {
		m.removeLevel99AuraEffects(change.ID)
		delete(m.speechBubbles, change.ID)
		if local {
			m.removeLevel99AuraEffects(localSkillTarget(ctx))
			delete(m.speechBubbles, localSkillTarget(ctx))
		}
	}
	if local {
		if !localStealthAllowsSkill(ctx, 0) {
			m.cancelAttackIntent()
		}
		if ctx.PlayerHasEffectState(db.EffectStateHide) {
			m.pendingPickup = pickupIntent{}
		}
		if !localStealthAllowsSkill(ctx, m.pendingSkill.skill.ID) {
			m.pendingSkill = pendingSkillTarget{}
		}
		if newVisualJob := localPlayerVisualJob(ctx); newVisualJob != oldVisualJob {
			m.reloadPlayerSpriteView(ctx, fmt.Sprintf("state effect=0x%08X", change.EffectState))
		}
		glog.Debugf("actor state local id=%d body=%d health=0x%04X effect=0x%08X", change.ID, change.BodyState, change.HealthState, change.EffectState)
		return
	}
	if change.EffectState&actorStealthMask != 0 {
		if m.pendingAttack.targetID == change.ID || m.lockedAttackID == change.ID {
			m.cancelAttackIntent()
		}
		if m.pendingSkill.targetID == change.ID {
			m.pendingSkill = pendingSkillTarget{}
		}
		if m.attackFocusID == change.ID {
			m.clearAttackFocus()
		}
	}
	glog.Debugf("actor state id=%d body=%d health=0x%04X effect=0x%08X", change.ID, change.BodyState, change.HealthState, change.EffectState)
}

// applyActorStateSnapshot updates state shared by the login and world modes.
// Rendering transitions remain the responsibility of WorldMode.
func applyActorStateSnapshot(ctx client.Context, change network.ActorStateChange) (oldState uint32, local, ok bool) {
	if change.ID == 0 || ctx.World == nil {
		return 0, false, false
	}
	if isLocalActor(ctx, change.ID) {
		oldState = ctx.World.Player.EffectState
		setActorRenderState(&ctx.World.Player, change.BodyState, change.HealthState, change.EffectState)
		setSelectedCharacterOptionBit(ctx, db.EffectStateWedding, change.EffectState&db.EffectStateWedding != 0)
		return oldState, true, true
	}
	actor, ok := ctx.World.Actors[change.ID]
	if !ok {
		return 0, false, false
	}
	oldState = actor.EffectState
	setActorRenderState(&actor, change.BodyState, change.HealthState, change.EffectState)
	upsertActor(ctx, actor)
	return oldState, false, true
}

func (m *WorldMode) applyActorBladeStop(ctx client.Context, blade network.ActorBladeStop) {
	if blade.SourceID == 0 || blade.TargetID == 0 || ctx.World == nil {
		return
	}
	m.applyActorBladeStopSide(ctx, blade.SourceID, blade.TargetID, blade.Active)
	m.applyActorBladeStopSide(ctx, blade.TargetID, blade.SourceID, blade.Active)
	glog.Debugf("actor blade stop src=%d target=%d active=%t", blade.SourceID, blade.TargetID, blade.Active)
}

func (m *WorldMode) applyActorBladeStopSide(ctx client.Context, actorID, lookID uint32, active bool) {
	actor, ok, local := actorForCombatID(ctx, actorID)
	if !ok {
		return
	}
	target, targetOK, _ := actorForCombatID(ctx, lookID)
	if targetOK {
		actor.Dir = directionFromDelta(actor.X, actor.Y, target.X, target.Y, actor.Dir)
		if local {
			ctx.World.Player.Dir = actor.Dir
			ctx.World.Dir = actor.Dir
		} else {
			upsertActor(ctx, actor)
		}
	}
	action := spriteActionIdle
	if active {
		action = readyFightActionFamily(actor)
	}
	anim := actorAnimation{
		actionFamily: action,
		started:      time.Now(),
		play:         true,
		hasPlay:      true,
		loop:         active,
	}
	if local && ctx.Session != nil {
		m.setActorAction(ctx, ctx.Session.AccountID, anim)
		m.setActorAction(ctx, ctx.Session.CharID, anim)
		return
	}
	m.setActorAction(ctx, actorID, anim)
}

func setActorRenderState(actor *worldstate.Actor, bodyState, healthState uint16, effectState uint32) {
	actor.BodyState = bodyState
	actor.HealthState = healthState
	actor.EffectState = effectState
	actor.HasState = true
	applyActorCartStateFromEffect(actor)
}

func applyActorBodyState(actor worldstate.Actor, state *spriteState) {
	switch actor.BodyState {
	case db.BodyStateFreeze:
		state.actionFamily = freezeActionFamily(actor)
		state.moving = false
		state.loop = false
		state.loopIdle = false
		state.play = false
		state.hasPlay = true
		state.fixedMotion = 0
		state.hasFixedMotion = true
	case db.BodyStateStone:
		state.moving = false
		state.loop = false
		state.loopIdle = false
		state.play = false
		state.hasPlay = true
	}
}

func actorStateTint(actor worldstate.Actor) color.RGBA {
	r, g, b, a := 1.0, 1.0, 1.0, 1.0
	switch actor.BodyState {
	case db.BodyStateStone:
		r, g, b = 0.1, 0.1, 0.1
	case db.BodyStateStonewait:
		r, g, b = 0.3, 0.3, 0.3
	case db.BodyStateFreeze:
		r, g, b = 0.0, 0.4, 0.8
	}
	if actor.HealthState&db.HealthStateCurse != 0 {
		r *= 0.5
		g *= 0.15
		b *= 0.1
	}
	if actor.HealthState&db.HealthStatePoison != 0 {
		r *= 0.9
		g *= 0.4
		b *= 0.8
	}
	if actor.HealthState&db.HealthStateBlind != 0 {
		r *= 0.2
		g *= 0.2
		b *= 0.2
	}
	vr, vg, vb, va := actorOpt3StateTint(actor.Opt3State)
	r *= vr
	g *= vg
	b *= vb
	a *= va
	return color.RGBA{R: byte(clampUnit(r) * 255), G: byte(clampUnit(g) * 255), B: byte(clampUnit(b) * 255), A: byte(clampUnit(a) * 255)}
}

func actorOpt3StateTint(opt3State uint32) (float64, float64, float64, float64) {
	r, g, b, a := 1.0, 1.0, 1.0, 1.0
	if opt3State&db.Opt3Quicken != 0 {
		b = 0
	}
	if opt3State&db.Opt3Energycoat != 0 || opt3State&db.Opt3Bunsin != 0 {
		r, g, b = 0.5, 0.5, 0.85
	}
	if opt3State&db.Opt3Overthrust != 0 {
		g, b = 0.75, 0.75
	}
	if opt3State&db.Opt3Warm != 0 {
		g, b = 0.4, 0.4
	}
	if opt3State&db.Opt3Soullink != 0 {
		r, g, b, a = 0.35, 0.35, 0.9, 0.9
	}
	return r, g, b, a
}

func (m *WorldMode) actorRenderTint(actor worldstate.Actor, now time.Time) color.RGBA {
	return m.actorBodyColorTint(actor.ID, actorStateTint(actor), now)
}

func (m *WorldMode) playerRenderTint(ctx client.Context, actor worldstate.Actor, now time.Time) color.RGBA {
	tint := m.actorRenderTint(actor, now)
	if ctx.Session == nil {
		return tint
	}
	if ctx.Session.AccountID != actor.ID {
		tint = m.actorBodyColorTint(ctx.Session.AccountID, tint, now)
	}
	if ctx.Session.CharID != actor.ID && ctx.Session.CharID != ctx.Session.AccountID {
		tint = m.actorBodyColorTint(ctx.Session.CharID, tint, now)
	}
	return tint
}

func (m *WorldMode) actorBodyColorTint(actorID uint32, tint color.RGBA, now time.Time) color.RGBA {
	if actorID == 0 {
		return tint
	}
	for _, effect := range m.worldEffects {
		if effect.actorID != actorID || now.Before(effect.starts) || now.After(effect.expires) {
			continue
		}
		switch effect.effectID {
		case effectBodyColor:
			tint = effectBodyColorTint(tint, now.Sub(effect.starts))
		case effectPortal5:
			tint = portal5BodyColorTint(tint, now.Sub(effect.starts))
		case effectMagicCrasher2:
			tint = magicCrasher2BodyColorTint(tint, actorID, effect.starts, now.Sub(effect.starts))
		}
	}
	return tint
}

func effectBodyColorTint(tint color.RGBA, elapsed time.Duration) color.RGBA {
	alpha := clampFloat(float64(elapsed)/float64(100*time.Millisecond), 0, 1)
	return color.RGBA{
		R: mixTintChannel(tint.R, 255, alpha),
		G: mixTintChannel(tint.G, 0, alpha),
		B: mixTintChannel(tint.B, 0, alpha),
		A: tint.A,
	}
}

func portal5BodyColorTint(tint color.RGBA, elapsed time.Duration) color.RGBA {
	if elapsed < 0 || elapsed >= 800*time.Millisecond {
		return tint
	}
	blue := 0.0
	alpha := 0.1
	if elapsed >= 200*time.Millisecond {
		progress := clampFloat(float64(elapsed-200*time.Millisecond)/float64(600*time.Millisecond), 0, 1)
		blue = progress
		alpha = 0.1 + 0.9*progress
	}
	return multiplyTint(tint, 1, 1, blue, alpha)
}

func magicCrasher2BodyColorTint(tint color.RGBA, actorID uint32, starts time.Time, elapsed time.Duration) color.RGBA {
	if elapsed < 0 || elapsed >= time.Second {
		return tint
	}
	bucket := uint64(elapsed / (50 * time.Millisecond))
	seed := uint64(actorID)*0x9e3779b185ebca87 ^ uint64(starts.UnixNano()) ^ bucket*0xc2b2ae3d27d4eb4f
	return multiplyTint(tint, randomBodyColorChannel(seed), randomBodyColorChannel(seed+1), randomBodyColorChannel(seed+2), 1)
}

func randomBodyColorChannel(seed uint64) float64 {
	seed ^= seed >> 33
	seed *= 0xff51afd7ed558ccd
	seed ^= seed >> 33
	seed *= 0xc4ceb9fe1a85ec53
	seed ^= seed >> 33
	return float64(byte(seed>>56)) / 255
}

func multiplyTint(tint color.RGBA, red, green, blue, alpha float64) color.RGBA {
	return color.RGBA{
		R: uint8(clampFloat(float64(tint.R)*red, 0, 255)),
		G: uint8(clampFloat(float64(tint.G)*green, 0, 255)),
		B: uint8(clampFloat(float64(tint.B)*blue, 0, 255)),
		A: uint8(clampFloat(float64(tint.A)*alpha, 0, 255)),
	}
}

func mixTintChannel(from, to uint8, alpha float64) uint8 {
	value := float64(from)*(1-alpha) + float64(to)*alpha
	return uint8(clampFloat(value, 0, 255))
}

func (m *WorldMode) actorRenderScale(actorID uint32, baseScale float64, now time.Time) float64 {
	return baseScale * m.actorBodySizeMultiplier(actorID, now)
}

func (m *WorldMode) playerRenderScale(ctx client.Context, actor worldstate.Actor, baseScale float64, now time.Time) float64 {
	if scale := m.playerBodyRenderScale(actor.ID, actor.Job, baseScale, now); scale != baseScale {
		return scale
	}
	if ctx.Session == nil {
		return baseScale
	}
	if ctx.Session.AccountID != actor.ID {
		if scale := m.playerBodyRenderScale(ctx.Session.AccountID, actor.Job, baseScale, now); scale != baseScale {
			return scale
		}
	}
	if ctx.Session.CharID != actor.ID && ctx.Session.CharID != ctx.Session.AccountID {
		if scale := m.playerBodyRenderScale(ctx.Session.CharID, actor.Job, baseScale, now); scale != baseScale {
			return scale
		}
	}
	return baseScale
}

func (m *WorldMode) playerBodyRenderScale(actorID uint32, job int16, baseScale float64, now time.Time) float64 {
	bodyScale := playerBodyScaleForJob(job)
	multiplier := m.actorBodySizeMultiplierFrom(actorID, now, bodyScale)
	if multiplier == bodyScale {
		return baseScale
	}
	return baseScale / bodyScale * multiplier
}

func (m *WorldMode) actorBodySizeMultiplier(actorID uint32, now time.Time) float64 {
	return m.actorBodySizeMultiplierFrom(actorID, now, 1)
}

func (m *WorldMode) actorBodySizeMultiplierFrom(actorID uint32, now time.Time, baseMultiplier float64) float64 {
	if actorID == 0 {
		return baseMultiplier
	}
	multiplier := baseMultiplier
	var latest time.Time
	for _, effect := range m.worldEffects {
		if effect.actorID != actorID || now.Before(effect.starts) || now.After(effect.expires) {
			continue
		}
		targetSize, transition, ok := actorBodySizeTarget(effect.effectID)
		if !ok || effect.starts.Before(latest) {
			continue
		}
		targetMultiplier := targetSize / 5
		if transition {
			duration := effect.duration
			if duration <= 0 {
				duration = 300 * time.Millisecond
			}
			progress := clampFloat(float64(now.Sub(effect.starts))/float64(duration), 0, 1)
			targetMultiplier = baseMultiplier + (targetMultiplier-baseMultiplier)*progress
		}
		multiplier = targetMultiplier
		latest = effect.starts
	}
	return multiplier
}

func actorBodySizeTarget(effectID int) (targetSize float64, transition bool, ok bool) {
	switch effectID {
	case effectBabyBody:
		return 2.5, true, true
	case effectBabyBody2:
		return 2.5, false, true
	case effectGiantBody:
		return 7.5, true, true
	case effectGiantBody2:
		return 7.5, false, true
	default:
		return 0, false, false
	}
}

func freezeActionFamily(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionPCFreeze2
	}
	return spriteActionIdle
}

func readyFightActionFamily(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionPCReadyFight
	}
	return spriteActionIdle
}
