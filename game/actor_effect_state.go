package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
)

type actorEffectStateEffect struct {
	bit      uint32
	effectID int
}

var actorEffectStateEffects = []actorEffectStateEffect{
	{bit: db.EffectStateSight, effectID: effectSight},
	{bit: db.EffectStateRuwach, effectID: effectRuwach},
}

func (m *WorldMode) applyActorEffectStateEffects(ctx client.Context, actorID uint32, oldState, newState uint32) {
	if actorID == 0 {
		return
	}
	for _, stateEffect := range actorEffectStateEffects {
		if newState&stateEffect.bit != 0 {
			m.addWorldEffectIfMissing(ctx, stateEffect.effectID, actorID)
		} else if oldState&stateEffect.bit != 0 {
			m.removeWorldEffect(stateEffect.effectID, actorID)
		}
	}
}

func (m *WorldMode) syncCurrentActorEffectStateEffects(ctx client.Context) {
	if ctx.World == nil {
		return
	}
	if ctx.World.Player.HasState {
		m.applyActorEffectStateEffects(ctx, ctx.World.Player.ID, 0, ctx.World.Player.EffectState)
	}
	if ctx.Session != nil {
		for _, id := range []uint16{db.StatusRgCconfineM, db.StatusRgCconfineS} {
			if status, ok := ctx.Session.Statuses.Active[id]; ok {
				remaining := time.Until(status.ExpiresAt)
				m.applyCloseConfineStatus(ctx, network.StatusEffectChange{
					StatusID: id, Active: !status.HasDuration || remaining > 0,
					HasDuration: status.HasDuration, Duration: remaining,
				})
			}
		}
	}
	m.syncActorSpiritEffects(ctx, ctx.World.Player.ID)
	for _, actor := range ctx.World.Actors {
		if actor.HasState {
			m.applyActorEffectStateEffects(ctx, actor.ID, 0, actor.EffectState)
		}
		m.syncActorSpiritEffects(ctx, actor.ID)
	}
}

func (m *WorldMode) removeActorEffectStateEffects(actorID uint32) {
	for _, stateEffect := range actorEffectStateEffects {
		m.removeWorldEffect(stateEffect.effectID, actorID)
	}
	m.removeWorldEffect(effectNPCStop2, actorID)
	m.removeWorldEffect(effectGumgang, actorID)
	m.removeWorldEffect(effectSteelBody, actorID)
	m.removeWorldEffect(effectAsuraBody, actorID)
	m.removeWorldEffect(effectAsuraBodyMonster, actorID)
	m.removeWorldEffect(effectBabyBody, actorID)
	m.removeWorldEffect(effectBabyBodyBack, actorID)
	m.removeWorldEffect(effectChookgi, actorID)
	m.removeWorldEffect(effectChookgi2, actorID)
	m.removeWorldEffect(effectChookgi3, actorID)
}

// Sphere counts are authoritative server state, not a side effect of casting
// Call Spirits: spheres can also be spent, absorbed, or expire.
func applySpiritSphereCount(ctx client.Context, update network.SpiritSphereCount) bool {
	actor, ok, local := actorForCombatID(ctx, update.ActorID)
	if !ok {
		return false
	}
	actor.SpiritSpheres = update.Count
	actor.HasSpiritSpheres = true
	if local {
		ctx.World.Player.SpiritSpheres = update.Count
		ctx.World.Player.HasSpiritSpheres = true
	} else {
		ctx.World.Actors[actor.ID] = actor
	}
	return true
}

// Like GameModeShowEFST, this updates the displayed effects without replacing
// the actor's movement, appearance, or body/health states.
func applyNPCShowEffectUpdate(ctx client.Context, update network.NPCShowEffectUpdate) bool {
	if ctx.World == nil {
		return false
	}
	actor, ok := ctx.World.Actors[update.ActorID]
	if !ok {
		return false
	}
	actor.Opt3State = update.Opt3State
	actor.HasState = true
	actor.Level, actor.HasLevel = update.Level, true
	ctx.World.Actors[actor.ID] = actor
	return true
}

func (m *WorldMode) applySoulLinkerMonsterSkill(ctx client.Context, notify network.SkillNoDamageNotify) {
	var bit uint32
	switch notify.SkillID {
	case db.SkillSLSwoo:
		bit = db.Opt3Overthrust
	case db.SkillSLSke:
		bit = db.Opt3Energycoat
	case db.SkillSLSka:
		bit = db.Opt3Steelbody
	default:
		return
	}
	actor, ok, local := actorForCombatID(ctx, notify.TargetID)
	if !ok || local || !isMonsterLikeHoverActor(actor) {
		return
	}
	// Monsters need not receive status icons. The successful skill starts the
	// effect; actor entries and ZC_NPC_SHOWEFST_UPDATE carry its later state.
	actor.Opt3State |= bit
	actor.HasState = true
	ctx.World.Actors[actor.ID] = actor
	m.syncActorSpiritEffects(ctx, actor.ID)
}

func (m *WorldMode) syncActorSpiritEffects(ctx client.Context, actorID uint32) {
	if isLocalActor(ctx, actorID) {
		actorID = localSkillTarget(ctx)
	}
	actor, ok, _ := actorForCombatID(ctx, actorID)
	if !ok {
		return
	}
	visible := !actorHasStealth(actor)
	m.setActorStateEffect(ctx, actorID, effectAsuraBody, visible && actor.Opt3State&db.Opt3Soullink != 0)
	if actor.HasState && isMonsterLikeHoverActor(actor) {
		shrinking := actor.Opt3State&db.Opt3Overthrust != 0
		if shrinking && visible {
			m.removeWorldEffect(effectBabyBodyBack, actorID)
			m.setActorStateEffect(ctx, actorID, effectBabyBody, true)
		} else {
			for _, effect := range m.worldEffects {
				if effect.effectID == effectBabyBody && effect.actorID == actorID && effect.persistent {
					m.removeWorldEffect(effectBabyBody, actorID)
					if visible && !shrinking {
						m.addWorldEffect(ctx, effectBabyBodyBack, actorID)
					}
					break
				}
			}
		}
	}
	if actor.HasState && isMonsterLikeHoverActor(actor) && actor.Opt3State&db.Opt3Energycoat == 0 {
		m.removeWorldEffect(effectAsuraBodyMonster, actorID)
	}
	m.setActorStateEffect(ctx, actorID, effectGumgang, visible && actor.Opt3State&db.Opt3Explosionspirits != 0)
	m.setActorStateEffect(ctx, actorID, effectSteelBody, visible && actor.Opt3State&db.Opt3Steelbody != 0)
	sphereEffect := effectChookgi
	switch actor.Job {
	case db.JobMonkB:
		sphereEffect = effectChookgi2
	case db.JobGunslinger, db.JobGunslingerB:
		sphereEffect = effectChookgi3
	}
	for _, id := range [...]int{effectChookgi, effectChookgi2, effectChookgi3} {
		m.setActorStateEffect(ctx, actorID, id, visible && actor.SpiritSpheres > 0 && id == sphereEffect)
	}
}

// A state effect loops until the server clears it. Ground skill units still use
// persistent=true with a deadline; a zero deadline means explicitly removed.
func (m *WorldMode) setActorStateEffect(ctx client.Context, actorID uint32, effectID int, active bool) {
	if !active {
		m.removeWorldEffect(effectID, actorID)
		return
	}
	m.addWorldEffectIfMissing(ctx, effectID, actorID)
	now := time.Now()
	for i := range m.worldEffects {
		effect := &m.worldEffects[i]
		if effect.effectID == effectID && effect.actorID == actorID && !effect.expired(now) {
			effect.persistent = true
			effect.expires = time.Time{}
		}
	}
}
