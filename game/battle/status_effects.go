package battle

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

func ApplyHallucination(screen *render.Frame, s *session.Session, now time.Time) {
	if s == nil || s.NoEffects {
		return
	}
	effect, ok := s.Statuses.Active[db.StatusIllusion]
	if !ok || effect.HasDuration && !effect.ExpiresAt.IsZero() && !now.Before(effect.ExpiresAt) {
		return
	}
	screen.SetHallucination(now.Sub(effect.StartedAt))
}

func ApplyLocalStatusEffectChange(s *session.Session, change network.StatusEffectChange, now time.Time) bool {
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

func (m *System) ApplyTaekwonNightStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusSoullink && change.StatusID != db.StatusSke {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	if id == 0 || !scene.IsLocalActor(ctx, id) || m.bindings.Map.Night == change.Active {
		return
	}
	m.bindings.Map.Night = change.Active
	// Ground and static-model colors include scene lighting, so rebuild their
	// retained vertices when Soul Link or Eske changes the local night mode.
	m.bindings.Map.InvalidateLighting()
}

func (m *System) ApplyTaekwonStatus(ctx client.Context, change network.StatusEffectChange) {
	if ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	if id == 0 {
		return
	}
	if scene.IsLocalActor(ctx, id) {
		id = scene.LocalSkillTarget(ctx)
	}
	now := time.Now()
	switch change.StatusID {
	case db.StatusRun:
		if m.bindings.Effects.HasActive(effects.Run, id, now) {
			m.bindings.Effects.AddAt(ctx, effects.StopRun, id, now)
		}
		m.bindings.Effects.Remove(effects.Run, id)
		return
	case db.StatusTing:
		m.bindings.ScheduleSoundAtActor(now, id, "effect\\T_벽튕김.wav")
		m.bindings.Effects.AddAt(ctx, effects.QuakeBody, id, now)
		m.bindings.Effects.Remove(effects.Run, id)
		return
	}
}

func ApplyActorOpt3StateStatus(ctx client.Context, change network.StatusEffectChange) uint32 {
	bit, ok := actorOpt3StateBitForStatus(change.StatusID)
	if !ok || ctx.World == nil {
		return 0
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	if id == 0 {
		return 0
	}
	actor, ok, local := scene.ActorForCombatID(ctx, id)
	if !ok {
		return 0
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
	} else {
		actors.UpsertActor(ctx, actor)
	}
	return id
}

func actorOpt3StateBitForStatus(statusID uint16) (uint32, bool) {
	bit, ok := db.StatusOpt3State[statusID]
	if !ok {
		return 0, false
	}
	return bit, true
}

func (m *System) ApplyTrickDeadStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusTrickdead || ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	actor, ok, _ := scene.ActorForCombatID(ctx, id)
	if !ok {
		return
	}
	now := time.Now()
	if change.Active {
		actionFamily := actors.DeathActionFamilyForActor(actor)
		m.setTrickDeadStatusAction(ctx, id, actionFamily, now, m.bindings.Actors.ActorActionDuration(ctx, actor, actionFamily, scene.DefaultDeathAnimationDuration), true)
		glog.Debugf("trick dead active actor=%d", id)
		return
	}
	m.setTrickDeadStatusAction(ctx, id, scene.SpriteActionIdle, now, scene.DefaultAttackAnimationDuration, false)
	glog.Debugf("trick dead inactive actor=%d", id)
}

func (m *System) setTrickDeadStatusAction(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, holdFinal bool) {
	m.bindings.SetActorAction(ctx, id, scene.ActorAnimation{
		ActionFamily: actionFamily,
		Started:      started,
		Duration:     duration,
		Play:         true,
		HasPlay:      true,
		HoldFinal:    holdFinal,
	})
}

func (m *System) RemoveExpiredStatusEffects(s *session.Session, now time.Time) {
	if s == nil {
		return
	}
	for id, effect := range s.Statuses.Active {
		if effect.HasDuration && !effect.ExpiresAt.IsZero() && now.After(effect.ExpiresAt) {
			delete(s.Statuses.Active, id)
			if id == db.StatusCashBossAlarm {
				m.bindings.Minimap.ClearBossMarker()
			}
		}
	}
}
