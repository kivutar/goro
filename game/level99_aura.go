package game

import (
	"time"

	"github.com/kivutar/goro/client"
)

const (
	level99AuraLevel    = 99
	level99AuraLifetime = 24 * time.Hour
)

var (
	level99AuraNormalEffects = []int{effectLevel99, effectLevel99Ground, effectLevel99Bubble}
	level99AuraSimpleEffects = []int{effectLevel99Bubble}
)

func (m *WorldMode) syncLevel99AuraEffects(ctx client.Context, now time.Time) {
	if m == nil || ctx.World == nil {
		return
	}
	if ctx.Session != nil {
		m.syncActorLevel99AuraEffects(ctx, localSkillTarget(ctx), ctx.Session.Progress.BaseLevel, true, now)
	}
	for _, actor := range ctx.World.Actors {
		m.syncActorLevel99AuraEffects(ctx, actor.ID, actor.Level, actor.HasLevel, now)
	}
}

func (m *WorldMode) syncActorEntryLevel99Aura(ctx client.Context, entryID uint32) {
	if m == nil || ctx.World == nil || entryID == 0 {
		return
	}
	actor, ok := ctx.World.Actors[entryID]
	if !ok {
		m.removeLevel99AuraEffects(entryID)
		return
	}
	m.syncActorLevel99AuraEffects(ctx, actor.ID, actor.Level, actor.HasLevel, time.Now())
}

func (m *WorldMode) syncActorLevel99AuraEffects(ctx client.Context, actorID uint32, level int, hasLevel bool, now time.Time) {
	if actorID == 0 {
		return
	}
	if !hasLevel {
		return
	}
	if actor, ok, _ := actorForCombatID(ctx, actorID); ok && actorHasStealth(actor) {
		m.removeLevel99AuraEffects(actorID)
		return
	}
	var wanted []int
	if level >= level99AuraLevel {
		wanted = level99AuraEffectIDs(ctx)
		for _, effectID := range wanted {
			if !m.hasActiveWorldEffect(effectID, actorID, now) {
				m.addLevel99AuraEffect(ctx, effectID, actorID, now)
			}
		}
	}
	for _, effectID := range level99AuraNormalEffects {
		if !level99AuraWantsEffect(wanted, effectID) {
			m.removeWorldEffect(effectID, actorID)
		}
	}
}

func (m *WorldMode) addLevel99AuraEffect(ctx client.Context, effectID int, actorID uint32, now time.Time) bool {
	x, y, ok := effectAnchor(ctx, actorID)
	if !ok {
		return false
	}
	return m.addWorldEffectAtCellLifetime(ctx, effectID, actorID, x, y, now, level99AuraLifetime, true)
}

func (m *WorldMode) removeLevel99AuraEffects(actorID uint32) {
	for _, effectID := range level99AuraNormalEffects {
		m.removeWorldEffect(effectID, actorID)
	}
}

func level99AuraEffectIDs(ctx client.Context) []int {
	if lessEffectsEnabled(ctx) {
		return level99AuraSimpleEffects
	}
	return level99AuraNormalEffects
}

func level99AuraWantsEffect(wanted []int, effectID int) bool {
	for _, wantedID := range wanted {
		if wantedID == effectID {
			return true
		}
	}
	return false
}
