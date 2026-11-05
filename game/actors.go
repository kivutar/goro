package game

import (
	"fmt"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *WorldMode) upsertNetworkActor(ctx client.Context, entry network.ActorEntry) {
	if ctx.World == nil {
		return
	}
	m.actors().ClearActorVanish(entry.ID)
	previous := ctx.World.Actors[entry.ID]
	oldState := uint32(0)
	if existing, ok := ctx.World.Actors[entry.ID]; ok && existing.HasState {
		oldState = existing.EffectState
	}
	if entry.Moving {
		m.clearActorAction(ctx, entry.ID)
	}
	actors.UpsertNetworkActor(ctx, entry)
	m.actors().SyncActorSpiritEffects(ctx, entry.ID)
	m.effects().SyncActorEntryLevel99Aura(ctx, entry.ID)
	m.social().RequestActorGuildEmblem(ctx, entry.GuildID, entry.EmblemVersion)
	if !entry.HasState {
		return
	}
	actor, ok := ctx.World.Actors[entry.ID]
	if !ok || !actor.HasState {
		return
	}
	m.actors().ApplyActorEffectStateEffects(ctx, actor.ID, oldState, actor.EffectState)
	m.actors().SyncActorAilments(ctx, actor.ID, previous, previous.HasState)
}

func (m *WorldMode) applyActorVanish(ctx client.Context, vanish network.ActorVanish) {
	glog.Debugf("actor vanish id=%d reason=%d", vanish.ID, vanish.Reason)
	pendingHomunculusDelete := m.companionSystem.HomunculusDeleteID != 0 && vanish.ID == m.companionSystem.HomunculusDeleteID
	pendingMercenaryDelete := m.companionSystem.MercenaryDeleteID != 0 && vanish.ID == m.companionSystem.MercenaryDeleteID
	if m.attackFocusID == vanish.ID {
		m.clearAttackFocus()
	}
	if vanish.Reason == actors.ActorVanishDeath {
		m.actors().ClearActorVanish(vanish.ID)
		m.startActorDeath(ctx, vanish.ID)
		return
	}
	if vanish.Reason == actors.ActorVanishOutOfSight && !pendingHomunculusDelete && !pendingMercenaryDelete {
		if m.startActorOutOfSightVanish(ctx, vanish.ID) {
			return
		}
	}
	m.actors().AddActorVanishTeleportEffect(ctx, vanish)
	m.actors().RemoveActorEffectStateEffects(vanish.ID)
	m.effects().RemoveLevel99AuraEffects(vanish.ID)
	m.removeActorNow(ctx, vanish.ID)
	if pendingHomunculusDelete {
		m.companions().ClearDeletedHomunculus(ctx)
	}
	if pendingMercenaryDelete {
		m.companions().ClearDeletedMercenary(ctx)
	}
}

func (m *WorldMode) startActorOutOfSightVanish(ctx client.Context, id uint32) bool {
	if ctx.World == nil {
		return false
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return false
	}
	now := time.Now()
	actor = actors.FreezeActorAtRenderedPosition(actor, now)
	ctx.World.Actors[id] = actor
	if m.actorSystem.ActorVanishes == nil {
		m.actorSystem.ActorVanishes = make(map[uint32]actors.VanishFade)
	}
	m.actorSystem.ActorVanishes[id] = actors.VanishFade{
		Started:  now,
		RemoveAt: now.Add(actors.ActorVanishOutOfSightFadeDuration),
	}
	m.actors().RemoveActorEffectStateEffects(id)
	m.effects().RemoveLevel99AuraEffects(id)
	delete(m.battleSystem.ActorLife, id)
	delete(m.battleSystem.ActorCastBars, id)
	delete(m.actorSystem.SpeechBubbles, id)
	glog.Debugf("actor out-of-sight fade id=%d duration_ms=%d", id, actors.ActorVanishOutOfSightFadeDuration.Milliseconds())
	return true
}

func (m *WorldMode) removeActorNow(ctx client.Context, id uint32) {
	if ctx.World != nil {
		ctx.World.RemoveActor(id)
	}
	m.clearRemovedActorState(id)
}

func (m *WorldMode) clearRemovedActorState(id uint32) {
	delete(m.actorSystem.Ailments, id)
	delete(m.effects().ActorBlurs, id)
	m.effects().RemoveActor(id)
	if m.scriptHighlight.id == id {
		m.clearScriptHighlight()
	}
	delete(m.actorSystem.ActorAnims, id)
	delete(m.actorSystem.ComboDelays, id)
	delete(m.actorSystem.ActorDeaths, id)
	delete(m.actorSystem.ActorVanishes, id)
	delete(m.actorSoundFrames, id)
	delete(m.battleSystem.ActorLife, id)
	delete(m.battleSystem.ActorCastBars, id)
	delete(m.actorSystem.SpeechBubbles, id)
	delete(m.actorSystem.PetAccessoryIDs, id)
}

func (m *WorldMode) cleanupDeadActors(ctx client.Context, now time.Time) {
	if len(m.actorSystem.ActorDeaths) == 0 || ctx.World == nil {
		return
	}
	for id, removeAt := range m.actorSystem.ActorDeaths {
		// Dead player characters remain in the world so resurrection skills can
		// target them. Their zero deadline is cleared by ZC_RESURRECTION, an
		// actor refresh, or a later non-death vanish packet.
		if removeAt.IsZero() {
			continue
		}
		if now.Before(removeAt) {
			continue
		}
		ctx.World.RemoveActor(id)
		m.actors().RemoveActorEffectStateEffects(id)
		m.effects().RemoveLevel99AuraEffects(id)
		delete(m.actorSystem.ActorDeaths, id)
		delete(m.actorSystem.ActorAnims, id)
		delete(m.actorSystem.ComboDelays, id)
		delete(m.actorSoundFrames, id)
		delete(m.battleSystem.ActorLife, id)
		delete(m.actorSystem.SpeechBubbles, id)
		delete(m.actorSystem.PetAccessoryIDs, id)
		if m.pendingAttack.targetID == id {
			m.cancelAttackIntent()
		}
		if m.attackFocusID == id {
			m.clearAttackFocus()
		}
		if m.scriptHighlight.id == id {
			m.clearScriptHighlight()
		}
		glog.Debugf("actor death removed id=%d", id)
	}
}

func (m *WorldMode) cleanupVanishedActors(ctx client.Context, now time.Time) {
	if len(m.actorSystem.ActorVanishes) == 0 {
		return
	}
	for id, fade := range m.actorSystem.ActorVanishes {
		if now.Before(fade.RemoveAt) {
			continue
		}
		m.removeActorNow(ctx, id)
		glog.Debugf("actor out-of-sight removed id=%d", id)
	}
}

func (m *WorldMode) drawSceneActorOverlays(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, entries []scene.ActorDrawEntry) {
	combat := m.battle()
	visible := make([]scene.ActorDrawEntry, 0, len(entries))
	for _, entry := range entries {
		if scene.ActorHasStealth(entry.Actor) {
			if entry.IsPlayer {
				combat.DrawActorCastBar(screen, entry, now)
				combat.DrawActorLifeBar(screen, ctx, entry)
			}
			continue
		}
		visible = append(visible, entry)
	}
	entries = visible
	m.drawSiegeGuildEmblems(screen, ctx, projection, now, entries)
	for _, entry := range entries {
		combat.DrawActorCastBar(screen, entry, now)
		combat.DrawActorLifeBar(screen, ctx, entry)
	}
	m.drawAttackFocusMarker(screen, ctx, now, entries)
	m.drawScriptHighlightMarker(screen, ctx, now, entries)
	m.actors().DrawVendingBoardLabels(screen, ctx, entries)
	m.actors().DrawChatRoomBoardLabels(screen, ctx, entries)
	m.actors().DrawSpeechBubbles(screen, ctx, projection, entries, now)
	m.drawHoveredLocalPlayerNameLabel(screen, ctx, entries)
	m.drawHoveredActorNameLabel(screen, ctx, projection, now)
}

func (m *WorldMode) drawHoveredLocalPlayerNameLabel(screen *render.Frame, ctx client.Context, entries []scene.ActorDrawEntry) {
	if ctx.Input == nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsPlayer {
			continue
		}
		if !actors.PointInActorPickBounds(float64(ctx.Input.MouseX), float64(ctx.Input.MouseY), entry.ScreenX, entry.ScreenY, entry.Scale) {
			return
		}
		labelY := actors.ActorNameLabelY(entry.ScreenY, entry.Scale)
		if life, ok := m.battle().ActorLifeForDisplay(ctx, entry.Actor); ok {
			labelY = battle.ActorNameBelowLifeBarY(entry.ScreenY, entry.Scale, life)
		}
		actors.DrawActorNameLabelsAtY(screen, actors.ActorDisplayLabels(ctx, entry.Actor, true), m.social().ActorGuildEmblem(ctx, entry.Actor, true), entry.ScreenX, labelY, actors.ActorNameLabelColor(entry.Actor, true))
		return
	}
}

func (m *WorldMode) drawHoveredActorNameLabel(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) {
	if ctx.Input == nil || ctx.World == nil {
		return
	}
	if _, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
		return
	}
	actor, ok := hoveredCursorActor(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorSystem.ActorDeaths)
	if !ok || npc.IsWarpActor(actor) {
		return
	}
	labels := m.actors().HoveredActorDisplayLabels(ctx, actor, now)
	if len(labels) == 0 {
		return
	}
	actorX, actorY := scene.ActorRenderPosition(actor, now)
	terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	isPlayer := scene.IsLocalActor(ctx, actor.ID)
	labelY := actors.ActorNameLabelY(float64(point.Y), scale)
	if life, ok := m.battle().ActorLifeForDisplay(ctx, actor); ok {
		labelY = battle.ActorNameBelowLifeBarY(float64(point.Y), scale, life)
	}
	actors.DrawActorNameLabelsAtY(screen, labels, m.social().ActorGuildEmblem(ctx, actor, isPlayer), float64(point.X), labelY, actors.ActorNameLabelColor(actor, isPlayer))
}

func (m *WorldMode) syncCurrentActorEffectStateEffects(ctx client.Context) {
	if ctx.World == nil {
		return
	}
	if ctx.World.Player.HasState {
		m.actors().ApplyActorEffectStateEffects(ctx, ctx.World.Player.ID, 0, ctx.World.Player.EffectState)
		m.actors().SyncActorAilments(ctx, ctx.World.Player.ID, ctx.World.Player, false)
	}
	if ctx.Session != nil {
		if status, ok := ctx.Session.Statuses.Active[db.StatusLkconcentration]; ok && (!status.HasDuration || time.Now().Before(status.ExpiresAt)) {
			m.actors().SetActorStateEffect(ctx, scene.LocalSkillTarget(ctx), effects.LKConcentration, true)
		}
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
	m.syncStarDevil(ctx)
	m.actors().SyncActorSpiritEffects(ctx, ctx.World.Player.ID)
	for _, actor := range ctx.World.Actors {
		if actor.HasState {
			m.actors().ApplyActorEffectStateEffects(ctx, actor.ID, 0, actor.EffectState)
			m.actors().SyncActorAilments(ctx, actor.ID, actor, false)
		}
		m.actors().SyncActorSpiritEffects(ctx, actor.ID)
	}
}

func (m *WorldMode) applyActorStateChange(ctx client.Context, change network.ActorStateChange) {
	oldVisualJob := actors.LocalPlayerVisualJob(ctx)
	previous, _, _ := scene.ActorForCombatID(ctx, change.ID)
	oldState, local, ok := actors.ApplyActorStateSnapshot(ctx, change)
	if !ok {
		return
	}
	m.actors().ApplyActorEffectStateEffects(ctx, change.ID, oldState, change.EffectState)
	m.actors().SyncActorAilments(ctx, change.ID, previous, true)
	m.actors().SyncActorSpiritEffects(ctx, change.ID)
	if (oldState^change.EffectState)&db.EffectStateHide != 0 {
		m.effects().Add(ctx, effects.SummonSlave, change.ID)
		actor, _, _ := scene.ActorForCombatID(ctx, change.ID)
		if change.EffectState&db.EffectStateHide != 0 && actor.Job != db.JobNinja {
			m.effects().Add(ctx, effects.BashBegin, change.ID)
		}
	}
	if change.EffectState&scene.ActorStealthMask != 0 {
		m.effects().RemoveLevel99AuraEffects(change.ID)
		delete(m.actorSystem.SpeechBubbles, change.ID)
		if local {
			m.effects().RemoveLevel99AuraEffects(scene.LocalSkillTarget(ctx))
			delete(m.actorSystem.SpeechBubbles, scene.LocalSkillTarget(ctx))
		}
	}
	if local {
		if !skills.LocalStealthAllowsSkill(ctx, 0) {
			m.cancelAttackIntent()
		}
		if ctx.PlayerHasEffectState(db.EffectStateHide) {
			m.pendingPickup = pickupIntent{}
		}
		if !skills.LocalStealthAllowsSkill(ctx, m.skillSystem.Pending.Skill.ID) {
			m.skillSystem.Pending = skills.PendingTarget{}
		}
		if newVisualJob := actors.LocalPlayerVisualJob(ctx); newVisualJob != oldVisualJob {
			m.actors().ReloadPlayerSpriteView(ctx, fmt.Sprintf("state effect=0x%08X", change.EffectState))
		}
		glog.Debugf("actor state local id=%d body=%d health=0x%04X effect=0x%08X", change.ID, change.BodyState, change.HealthState, change.EffectState)
		return
	}
	if change.EffectState&scene.ActorStealthMask != 0 {
		if m.pendingAttack.targetID == change.ID {
			m.cancelAttackIntent()
		}
		if m.skillSystem.Pending.TargetID == change.ID {
			m.skillSystem.Pending = skills.PendingTarget{}
		}
		if m.attackFocusID == change.ID {
			m.clearAttackFocus()
		}
	}
	glog.Debugf("actor state id=%d body=%d health=0x%04X effect=0x%08X", change.ID, change.BodyState, change.HealthState, change.EffectState)
}

func (m *WorldMode) applyActorBladeStop(ctx client.Context, blade network.ActorBladeStop) {
	if blade.SourceID == 0 || blade.TargetID == 0 || ctx.World == nil {
		return
	}
	m.applyActorBladeStopSide(ctx, blade.SourceID, blade.TargetID, blade.Active, true)
	m.applyActorBladeStopSide(ctx, blade.TargetID, blade.SourceID, blade.Active, false)
	glog.Debugf("actor blade stop src=%d target=%d active=%t", blade.SourceID, blade.TargetID, blade.Active)
}

func (m *WorldMode) applyActorBladeStopSide(ctx client.Context, actorID, lookID uint32, active, caster bool) {
	actor, ok, local := scene.ActorForCombatID(ctx, actorID)
	if !ok {
		return
	}
	now := time.Now()
	anim := scene.ActorAnimation{ActionFamily: scene.SpriteActionIdle, Started: now, Play: true, HasPlay: true}
	if active {
		// Freeze the victim's current frame, before stopping movement or turning.
		state := m.actors().NonPCSpriteState(actor, now)
		if actor.Sitting {
			state.ActionFamily = scene.SpriteActionSit
		}
		started := now
		if res.HasPlayerJobToken(scene.ActorVisualJob(actor)) || companion.IsMercenary(actor) {
			state.LoopIdle = false
			state.HeadTurn, state.HeadDir = true, actor.HeadDir
			if view := m.actors().HumanoidSpriteViewForActor(ctx, actor); view != nil {
				started = view.Started
			}
		} else if view := m.actors().NonPCSpriteView(ctx, actor); view != nil {
			started = view.Started
		}
		anim.ActionFamily = state.ActionFamily
		anim.FixedMotion = state.FixedMotion
		renderActor := actor
		renderActor.Dir = sprites.DirectionForCamera(actor.Dir, scene.CameraYawForMap(ctx))
		if action, ok := m.actors().ActorResolvedAction(ctx, renderActor, state.ActionFamily); ok {
			anim.FixedMotion = sprites.BodyMotionForState(action, state, started, now)
		}
		if caster && res.HasPlayerJobToken(scene.ActorVisualJob(actor)) {
			anim.ActionFamily = sprites.ActionPCSkill
			anim.FixedMotion = 4
		}
		anim.Play = false
		anim.HasFixedMotion = true
		anim.HoldFinal = true
		actor.Opt3State |= db.Opt3Bladestop
	} else {
		actor.Opt3State &^= db.Opt3Bladestop
	}
	target, targetOK, _ := scene.ActorForCombatID(ctx, lookID)
	if targetOK {
		actor.Dir = actors.DirectionFromDelta(actor.X, actor.Y, target.X, target.Y, actor.Dir)
	}
	if local {
		ctx.World.Player.Opt3State = actor.Opt3State
		ctx.World.Player.Dir = actor.Dir
		ctx.World.Dir = actor.Dir
	} else {
		ctx.World.Actors[actor.ID] = actor
	}
	if active {
		m.stopActorMovementAt(ctx, actorID, now)
	}
	m.setCombatActorAction(ctx, actorID, anim)
}

func (m *WorldMode) applyPushCartStatus(ctx client.Context, change network.StatusEffectChange) bool {
	if change.StatusID != db.StatusOnPushCart || ctx.World == nil {
		return false
	}
	cartNum := 1
	if change.HasValues {
		cartNum = int(change.Values[0])
	}
	if change.ActorID == 0 || scene.IsLocalActor(ctx, change.ActorID) {
		actors.SetActorPushCartStatus(&ctx.World.Player, change.Active, cartNum)
		refreshLocalPlayerMoveSpeed(ctx)
		glog.Debugf("actor cart status local actor=%d active=%t cart=%d", change.ActorID, change.Active, ctx.World.Player.CartNum)
		return true
	}
	actor, ok := ctx.World.Actors[change.ActorID]
	if !ok {
		return true
	}
	actors.SetActorPushCartStatus(&actor, change.Active, cartNum)
	actors.UpsertActor(ctx, actor)
	glog.Debugf("actor cart status actor=%d active=%t cart=%d", change.ActorID, change.Active, actor.CartNum)
	return true
}

func (m *WorldMode) actors() *actors.System {
	m.actorSystem.Bind(actors.Bindings{
		Map:                  &m.mapScene,
		Effects:              m.effects(),
		Skills:               &m.skillSystem,
		Social:               m.social(),
		SpriteTexture:        m.spriteTexture,
		ScheduleSoundAtActor: m.scheduleSoundAtActor,
	})
	return &m.actorSystem
}
