package actors

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
)

type actorEffectStateEffect struct {
	bit      uint32
	effectID int
}

var actorEffectStateEffects = []actorEffectStateEffect{
	{bit: db.EffectStateSight, effectID: effects.Sight},
	{bit: db.EffectStateRuwach, effectID: effects.Ruwach},
}

func (m *System) ApplyActorEffectStateEffects(ctx client.Context, actorID uint32, oldState, newState uint32) {
	if actorID == 0 {
		return
	}
	for _, stateEffect := range actorEffectStateEffects {
		if stateEffect.effectID == effects.Sight {
			m.SetActorStateEffect(ctx, actorID, effects.Sight, newState&stateEffect.bit != 0)
			continue
		}
		if newState&stateEffect.bit != 0 {
			m.bindings.Effects.AddIfMissing(ctx, stateEffect.effectID, actorID)
		} else if oldState&stateEffect.bit != 0 {
			m.bindings.Effects.Remove(stateEffect.effectID, actorID)
		}
	}
}

func (m *System) RemoveActorEffectStateEffects(actorID uint32) {
	for _, stateEffect := range actorEffectStateEffects {
		m.bindings.Effects.Remove(stateEffect.effectID, actorID)
	}
	m.bindings.Effects.Remove(effects.NPCStop2, actorID)
	m.bindings.Effects.Remove(effects.Gumgang, actorID)
	m.bindings.Effects.Remove(effects.DoubleGumgang, actorID)
	m.bindings.Effects.Remove(effects.RedLightBody, actorID)
	m.bindings.Effects.Remove(effects.SteelBody, actorID)
	m.bindings.Effects.Remove(effects.Assumptio, actorID)
	m.bindings.Effects.Remove(effects.RedBody, actorID)
	m.bindings.Effects.Remove(effects.Moonlit, actorID)
	m.bindings.Effects.Remove(effects.PinkBody, actorID)
	m.bindings.Effects.RemoveMarionetteLinks(actorID)
	m.bindings.Effects.Remove(effects.LKConcentration, actorID)
	m.bindings.Effects.Remove(effects.LightBlade, actorID)
	m.bindings.Effects.Remove(effects.AsuraBody, actorID)
	m.bindings.Effects.Remove(effects.AsuraBodyMonster, actorID)
	m.bindings.Effects.Remove(effects.BabyBody, actorID)
	m.bindings.Effects.Remove(effects.BabyBodyBack, actorID)
	m.bindings.Effects.Remove(effects.Chookgi, actorID)
	m.bindings.Effects.Remove(effects.Chookgi2, actorID)
	m.bindings.Effects.Remove(effects.Chookgi3, actorID)
}

// Sphere counts are authoritative server state, not a side effect of casting
// Call Spirits: spheres can also be spent, absorbed, or expire.
func ApplySpiritSphereCount(ctx client.Context, update network.SpiritSphereCount) bool {
	actor, ok, local := scene.ActorForCombatID(ctx, update.ActorID)
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

func (m *System) ApplySoulLinkerMonsterSkill(ctx client.Context, notify network.SkillNoDamageNotify) {
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
	actor, ok, local := scene.ActorForCombatID(ctx, notify.TargetID)
	if !ok || local || !scene.IsMonsterLikeHoverActor(actor) {
		return
	}
	// Monsters need not receive status icons. The successful skill starts the
	// effect; actor entries and ZC_NPC_SHOWEFST_UPDATE carry its later state.
	actor.Opt3State |= bit
	actor.HasState = true
	ctx.World.Actors[actor.ID] = actor
	m.SyncActorSpiritEffects(ctx, actor.ID)
}

func (m *System) SyncActorSpiritEffects(ctx client.Context, actorID uint32) {
	if scene.IsLocalActor(ctx, actorID) {
		actorID = scene.LocalSkillTarget(ctx)
	}
	actor, ok, _ := scene.ActorForCombatID(ctx, actorID)
	if !ok {
		return
	}
	visible := !scene.ActorHasStealth(actor)
	m.SetActorStateEffect(ctx, actorID, effects.DoubleGumgang, visible && actor.Opt3State&db.Opt3Warm != 0)
	m.SetActorStateEffect(ctx, actorID, effects.RedLightBody, visible && actor.Opt3State&db.Opt3Warm != 0)
	m.SetActorStateEffect(ctx, actorID, effects.AsuraBody, visible && actor.Opt3State&db.Opt3Soullink != 0)
	if actor.HasState && scene.IsMonsterLikeHoverActor(actor) {
		shrinking := actor.Opt3State&db.Opt3Overthrust != 0
		if shrinking && visible {
			m.bindings.Effects.Remove(effects.BabyBodyBack, actorID)
			m.SetActorStateEffect(ctx, actorID, effects.BabyBody, true)
		} else {
			for _, effect := range m.bindings.Effects.Active {
				if effect.EffectID == effects.BabyBody && effect.ActorID == actorID && effect.Persistent {
					m.bindings.Effects.Remove(effects.BabyBody, actorID)
					if visible && !shrinking {
						m.bindings.Effects.Add(ctx, effects.BabyBodyBack, actorID)
					}
					break
				}
			}
		}
	}
	if actor.HasState && scene.IsMonsterLikeHoverActor(actor) && actor.Opt3State&db.Opt3Energycoat == 0 {
		m.bindings.Effects.Remove(effects.AsuraBodyMonster, actorID)
	}
	m.SetActorStateEffect(ctx, actorID, effects.Gumgang, visible && actor.Opt3State&db.Opt3Explosionspirits != 0)
	m.SetActorStateEffect(ctx, actorID, effects.SteelBody, visible && actor.Opt3State&db.Opt3Steelbody != 0)
	m.SetActorStateEffect(ctx, actorID, effects.Assumptio, visible && actor.Opt3State&db.Opt3Assumptio != 0)
	m.SetActorStateEffect(ctx, actorID, effects.RedBody, visible && actor.Opt3State&db.Opt3Berserk != 0)
	m.SetActorStateEffect(ctx, actorID, effects.LightBlade, visible && actor.Opt3State&db.Opt3Lightblade != 0)
	m.SetActorStateEffect(ctx, actorID, effects.Moonlit, visible && actor.Opt3State&db.Opt3Moonlit != 0)
	m.SetActorStateEffect(ctx, actorID, effects.PinkBody, visible && actor.Opt3State&db.Opt3Marionette != 0)
	if !visible || actor.HasState && actor.Opt3State&db.Opt3Marionette == 0 {
		m.bindings.Effects.RemoveMarionetteLinks(actorID)
	}
	sphereEffect := effects.Chookgi
	switch actor.Job {
	case db.JobMonkB:
		sphereEffect = effects.Chookgi2
	case db.JobGunslinger, db.JobGunslingerB:
		sphereEffect = effects.Chookgi3
	}
	if visible && actor.SpiritSpheres > 0 {
		for i := range m.bindings.Effects.Active {
			effect := &m.bindings.Effects.Active[i]
			if effect.EffectID == sphereEffect && effect.ActorID == actorID && effect.HitCount != min(int(actor.SpiritSpheres), 10) {
				effect.HitCount = min(int(actor.SpiritSpheres), 10)
				effect.Starts = time.Now()
				effect.SpiritSpheres = effects.NewSpiritSpheres(*effect)
			}
		}
	}
	for _, id := range [...]int{effects.Chookgi, effects.Chookgi2, effects.Chookgi3} {
		m.SetActorStateEffect(ctx, actorID, id, visible && actor.SpiritSpheres > 0 && id == sphereEffect)
	}
}

// A state effect lives until the server clears it. Ground skill units still use
// persistent=true with a deadline; a zero deadline means explicitly removed.
func (m *System) SetActorStateEffect(ctx client.Context, actorID uint32, effectID int, active bool) {
	if !active {
		m.bindings.Effects.Remove(effectID, actorID)
		return
	}
	now := time.Now()
	if !m.bindings.Effects.HasActive(effectID, actorID, now) {
		m.bindings.Effects.AddPersistentBetween(ctx, effectID, actorID, 0, now)
	}
	for i := range m.bindings.Effects.Active {
		effect := &m.bindings.Effects.Active[i]
		if effect.EffectID == effectID && effect.ActorID == actorID && !effect.Expired(now) {
			effect.Persistent = true
			effect.Expires = time.Time{}
		}
	}
}
