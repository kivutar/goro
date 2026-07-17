package game

import (
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
	if change.ID == 0 || ctx.World == nil {
		return
	}
	if isLocalActor(ctx, change.ID) {
		oldState := ctx.World.Player.EffectState
		setActorRenderState(&ctx.World.Player, change.BodyState, change.HealthState, change.EffectState)
		m.applyActorEffectStateEffects(ctx, change.ID, oldState, change.EffectState)
		glog.Debugf("actor state local id=%d body=%d health=0x%04X effect=0x%08X", change.ID, change.BodyState, change.HealthState, change.EffectState)
		return
	}
	actor, ok := ctx.World.Actors[change.ID]
	if !ok {
		return
	}
	oldState := actor.EffectState
	setActorRenderState(&actor, change.BodyState, change.HealthState, change.EffectState)
	m.applyActorEffectStateEffects(ctx, change.ID, oldState, change.EffectState)
	upsertActor(ctx, actor)
	glog.Debugf("actor state id=%d body=%d health=0x%04X effect=0x%08X", change.ID, change.BodyState, change.HealthState, change.EffectState)
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
	r, g, b := 1.0, 1.0, 1.0
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
	if actor.EffectState&db.Opt3Energycoat != 0 {
		r *= 0.5
		g *= 0.5
		b *= 0.85
	}
	return color.RGBA{R: byte(clampUnit(r) * 255), G: byte(clampUnit(g) * 255), B: byte(clampUnit(b) * 255), A: 255}
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
