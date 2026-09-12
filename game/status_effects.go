package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func (m *WorldMode) applyStatusEffectChange(ctx client.Context, change network.StatusEffectChange) {
	if ctx.Session == nil || change.StatusID == 0xFFFF {
		return
	}
	if m.applyPushCartStatus(ctx, change) {
		return
	}
	m.applyFalconStatus(ctx, change)
	m.applyActorOpt3StateStatus(ctx, change)
	m.applyTrickDeadStatus(ctx, change)
	m.applyTaekwonStatus(ctx, change)
	m.applyTaekwonNightStatus(ctx, change)
	if !applyLocalStatusEffectChange(ctx.Session, change, time.Now()) {
		return
	}
	if !change.Active {
		glog.Debugf("status effect inactive id=%d actor=%d", change.StatusID, change.ActorID)
		return
	}
	glog.Debugf("status effect active id=%d actor=%d duration_ms=%d", change.StatusID, change.ActorID, change.Duration.Milliseconds())
}

func applyLocalStatusEffectChange(s *session.Session, change network.StatusEffectChange, now time.Time) bool {
	if s == nil || change.StatusID == 0xFFFF || change.StatusID == db.StatusOnPushCart {
		return false
	}
	localID := s.AccountID
	if localID == 0 {
		localID = s.CharID
	}
	if change.ActorID != 0 && localID != 0 && change.ActorID != localID && change.ActorID != s.CharID {
		return false
	}
	if s.Statuses.Active == nil {
		s.Statuses.Active = make(map[uint16]session.StatusEffect)
	}
	if !change.Active {
		delete(s.Statuses.Active, change.StatusID)
		return true
	}
	effect := session.StatusEffect{
		ID:          change.StatusID,
		Source:      change.ActorID,
		StartedAt:   now,
		HasDuration: change.HasDuration,
	}
	if change.HasDuration {
		effect.ExpiresAt = now.Add(change.Duration)
	}
	s.Statuses.Active[change.StatusID] = effect
	return true
}

func (m *WorldMode) applyTaekwonNightStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusSoullink && change.StatusID != db.StatusSke {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = localSkillTarget(ctx)
	}
	if id == 0 || !isLocalActor(ctx, id) || m.taekwonNight == change.Active {
		return
	}
	m.taekwonNight = change.Active
	// Ground and static-model colors include scene lighting, so rebuild their
	// retained vertices when Soul Link or Eske changes the local night mode.
	m.gndMeshCache = nil
	m.rsmMeshCache = make(map[int][]retainedWorldMesh)
}

func (m *WorldMode) applyTaekwonStatus(ctx client.Context, change network.StatusEffectChange) {
	if ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = localSkillTarget(ctx)
	}
	if id == 0 {
		return
	}

	switch change.StatusID {
	case db.StatusRun:
		effectID := 442 // EF_RUN; the reference effect currently only reserves the footprint hook.
		if !change.Active {
			effectID = effectStopEffect
		}
		if m.addWorldEffect(ctx, effectID, id) {
			glog.Debugf("taekwon run status actor=%d active=%t effect=%d", id, change.Active, effectID)
		}
		return
	case db.StatusTing:
		if m.addWorldEffect(ctx, effectQuakeBody, id) {
			glog.Debugf("taekwon ting status actor=%d effect=%d", id, effectQuakeBody)
		}
		return
	}

	actionFamily, frame, ok := taekwonStanceAction(change.StatusID)
	if !ok {
		return
	}
	m.setCombatActorAction(ctx, id, actorAnimation{
		actionFamily:   actionFamily,
		started:        time.Now(),
		play:           false,
		hasPlay:        true,
		holdFinal:      true,
		fixedMotion:    frame,
		hasFixedMotion: true,
	})
}

func taekwonStanceAction(statusID uint16) (actionFamily, frame int, ok bool) {
	switch statusID {
	case db.StatusStormkickOn, db.StatusStormkickReady:
		return spriteActionPCSkill, 0, true
	case db.StatusDownkickOn, db.StatusDownkickReady:
		return spriteActionPCSkill, 2, true
	case db.StatusTurnkickOn, db.StatusTurnkickReady:
		return spriteActionPCSkill, 3, true
	case db.StatusCounterOn, db.StatusCounterReady:
		return spriteActionPCSkill, 4, true
	case db.StatusDodgeOn, db.StatusDodgeReady:
		return spriteActionPickup, 1, true
	default:
		return 0, 0, false
	}
}

func (m *WorldMode) applyActorOpt3StateStatus(ctx client.Context, change network.StatusEffectChange) {
	bit, ok := actorOpt3StateBitForStatus(change.StatusID)
	if !ok || ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = localSkillTarget(ctx)
	}
	if id == 0 {
		return
	}
	actor, ok, local := actorForCombatID(ctx, id)
	if !ok {
		return
	}
	if change.Active {
		actor.Opt3State |= bit
	} else {
		actor.Opt3State &^= bit
	}
	actor.HasState = true
	if local {
		ctx.World.Player.Opt3State = actor.Opt3State
		ctx.World.Player.HasState = true
		return
	}
	upsertActor(ctx, actor)
}

func actorOpt3StateBitForStatus(statusID uint16) (uint32, bool) {
	bit, ok := db.StatusOpt3State[statusID]
	if !ok {
		return 0, false
	}
	return bit, true
}

func (m *WorldMode) applyTrickDeadStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusTrickdead || ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = localSkillTarget(ctx)
	}
	actor, ok, _ := actorForCombatID(ctx, id)
	if !ok {
		return
	}
	now := time.Now()
	if change.Active {
		actionFamily := deathActionFamilyForActor(actor)
		m.setTrickDeadStatusAction(ctx, id, actionFamily, now, m.actorActionDuration(ctx, actor, actionFamily, defaultDeathAnimationDuration), true)
		glog.Debugf("trick dead active actor=%d", id)
		return
	}
	m.setTrickDeadStatusAction(ctx, id, spriteActionIdle, now, defaultAttackAnimationDuration, false)
	glog.Debugf("trick dead inactive actor=%d", id)
}

func (m *WorldMode) setTrickDeadStatusAction(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, holdFinal bool) {
	m.setCombatActorAction(ctx, id, actorAnimation{
		actionFamily: actionFamily,
		started:      started,
		duration:     duration,
		play:         true,
		hasPlay:      true,
		holdFinal:    holdFinal,
	})
}

func removeExpiredStatusEffects(s *session.Session, now time.Time) {
	if s == nil {
		return
	}
	for id, effect := range s.Statuses.Active {
		if effect.HasDuration && !effect.ExpiresAt.IsZero() && now.After(effect.ExpiresAt) {
			delete(s.Statuses.Active, id)
		}
	}
}

func localActorHasStatus(ctx client.Context, statusID uint16) bool {
	if ctx.Session == nil || ctx.Session.Statuses.Active == nil {
		return false
	}
	_, ok := ctx.Session.Statuses.Active[statusID]
	return ok
}
