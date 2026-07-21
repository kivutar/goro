package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestHomunculusAssistQueuesNearestMonsterAttack(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		AccountID:            10,
		HomunculusAggressive: true,
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, X: 10, Y: 10, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
				400: {ID: 400, X: 25, Y: 25, HasObjectType: true, ObjectType: actorObjectTypeMob},
				500: {ID: 500, X: 12, Y: 11, HasObjectType: true, ObjectType: actorObjectTypeMob},
			},
		},
	}

	mode.maybeQueueAggressiveCompanionTarget(ctx, companionAIHomunculus, 300, nil)

	if got := mode.companionAI.msg[300]; got != "3,500" {
		t.Fatalf("queued message = %q, want nearest monster attack", got)
	}
}

func TestHomunculusStandByDoesNotQueueAggressiveAttack(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, X: 10, Y: 10, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
				500: {ID: 500, X: 12, Y: 11, HasObjectType: true, ObjectType: actorObjectTypeMob},
			},
		},
	}

	mode.maybeQueueAggressiveCompanionTarget(ctx, companionAIHomunculus, 300, nil)

	if got := mode.companionAI.msg[300]; got != "" {
		t.Fatalf("passive mode queued message = %q", got)
	}
}

func TestHomunculusParamChangeUpdatesSessionAndLife(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:          300,
			Active:      true,
			Level:       12,
			Hunger:      37,
			HP:          10,
			MaxHP:       20,
			SP:          5,
			MaxSP:       10,
			AttackRange: 2,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}

	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusHP, Value: 18})
	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusMaxSP, Value: 44})
	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseExp, Value: 5_000_000_000})
	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusNextBaseExp, Value: 6_000_000_000})
	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusSkillPoint, Value: 7})
	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusSpeed, Value: 150})
	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseLevel, Value: 13})

	if sess.Homunculus.HP != 18 || sess.Homunculus.MaxSP != 44 || sess.Homunculus.Level != 13 || sess.Homunculus.Exp != 5_000_000_000 || sess.Homunculus.MaxExp != 6_000_000_000 || sess.Homunculus.Skills.Points != 7 {
		t.Fatalf("homunculus session = %+v", sess.Homunculus)
	}
	if life := mode.actorLife[300]; life.hp != 18 || life.maxHP != 20 || life.sp != 5 || life.maxSP != 44 || !life.hasSP {
		t.Fatalf("actor life = %+v", life)
	}
	if actor := ctx.World.Actors[300]; actor.Speed != 150 || actor.AttackRange != 2 {
		t.Fatalf("actor = %+v", actor)
	}
}

func TestHomunculusPropertyStoresDisplayLifeAndHunger(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, Name: "Vanilmirth2", HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}

	mode.applyHomunculusProperty(ctx, network.HomunculusProperty{
		Name:        "Pipou",
		Level:       45,
		Hunger:      62,
		HP:          123,
		MaxHP:       456,
		SP:          78,
		MaxSP:       90,
		AttackRange: 4,
	})

	life, ok := mode.actorLifeForDisplay(ctx, ctx.World.Actors[300])
	if !ok {
		t.Fatal("homunculus display life missing")
	}
	if life.hp != 123 || life.maxHP != 456 || life.sp != 78 || life.maxSP != 90 || !life.hasSP || life.hunger != 62 || life.maxHunger != 100 || !life.hasHunger || !life.friendly {
		t.Fatalf("homunculus display life = %+v", life)
	}
	if actor := ctx.World.Actors[300]; actor.Name != "Pipou" || actor.AttackRange != 4 {
		t.Fatalf("homunculus actor = %+v", actor)
	}
}

func TestHomunculusActorEntryUsesKnownSessionName(t *testing.T) {
	sess := &session.Session{
		Homunculus: session.Companion{
			Name: "Pipou",
		},
	}
	ctx := client.Context{
		Session: sess,
		World:   worldstate.New(),
	}

	upsertNetworkActor(ctx, network.ActorEntry{
		ID:            300,
		Job:           6002,
		X:             10,
		Y:             20,
		HasObjectType: true,
		ObjectType:    actorObjectTypeHomunculus,
	})

	if actor := ctx.World.Actors[300]; actor.Name != "Pipou" {
		t.Fatalf("homunculus actor name = %q, want session name", actor.Name)
	}
}

func TestHomunculusActorEntryDoesNotUseSessionNameForDifferentID(t *testing.T) {
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:   300,
			Name: "Pipou",
		},
	}
	ctx := client.Context{
		Session: sess,
		World:   worldstate.New(),
	}

	upsertNetworkActor(ctx, network.ActorEntry{
		ID:            301,
		Job:           6002,
		X:             10,
		Y:             20,
		HasObjectType: true,
		ObjectType:    actorObjectTypeHomunculus,
	})

	if actor := ctx.World.Actors[301]; actor.Name != "" {
		t.Fatalf("other homunculus actor name = %q, want empty before server ack", actor.Name)
	}
}

func TestHomunculusStateChangeAppliesKnownSessionName(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			Name: "Pipou",
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, Name: "Vanilmirth2", HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}

	mode.applyHomunculusStateChange(ctx, network.HomunculusStateChange{GID: 300, State: 0})

	if actor := ctx.World.Actors[300]; actor.Name != "Pipou" {
		t.Fatalf("homunculus actor name = %q, want known session name", actor.Name)
	}
}

func TestHomunculusStateChangeUpdatesDisplayHunger(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			HP:     50,
			MaxHP:  100,
			SP:     10,
			MaxSP:  20,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}

	mode.applyHomunculusStateChange(ctx, network.HomunculusStateChange{GID: 300, State: 2, Data: 24})

	life, ok := mode.actorLifeForDisplay(ctx, ctx.World.Actors[300])
	if !ok {
		t.Fatal("homunculus display life missing")
	}
	if sess.Homunculus.Hunger != 24 || life.hunger != 24 || life.maxHunger != 100 || !life.hasHunger {
		t.Fatalf("homunculus hunger session=%+v life=%+v", sess.Homunculus, life)
	}
}

func TestHomunculusParamLevelUpAddsHoUpEffect(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			Level:  9,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, X: 12, Y: 34, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}

	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseLevel, Value: 10})
	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseLevel, Value: 10})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.effectID != effectHoUp || effect.actorID != 300 {
		t.Fatalf("world effect = %+v", effect)
	}
}

func TestHomunculusNotifyEffect2AddsHoUpEffectWithoutSound(t *testing.T) {
	mode := NewWorldMode()
	ctx := client.Context{
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, X: 12, Y: 34, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 300, EffectID: effectHoUp})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.effectID != effectHoUp || effect.actorID != 300 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("scheduled sounds = %+v, want none for robr EF_HO_UP", mode.scheduledSounds)
	}
}

func TestHomunculusParamInitialLevelDoesNotAddHoUpEffect(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, X: 12, Y: 34, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}

	mode.applyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseLevel, Value: 10})

	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects = %+v, want none for initial level load", mode.worldEffects)
	}
}

func TestPendingHomunculusDeleteVanishClearsSession(t *testing.T) {
	mode := NewWorldMode()
	mode.homDeleteID = 300
	mode.companionAI.msg = map[uint32]string{300: "3,500"}
	mode.companionAI.resMsg = map[uint32]string{300: "1,300"}
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			Name:   "Pipou",
			HP:     1,
			MaxHP:  10,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}
	mode.ui.homunculusInfo.OpenInfo(ctx, sess.Homunculus)
	mode.ui.homunculusSkill.Open(ctx, nil)

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actorVanishOutOfSight})

	if sess.Homunculus.Active || sess.Homunculus.ID != 0 || sess.Homunculus.Name != "" {
		t.Fatalf("homunculus session after delete = %+v", sess.Homunculus)
	}
	if mode.homDeleteID != 0 {
		t.Fatalf("pending delete id = %d, want 0", mode.homDeleteID)
	}
	if _, ok := ctx.World.Actors[300]; ok {
		t.Fatal("deleted homunculus actor remained in world")
	}
	if mode.ui.homunculusInfo.IsOpen() {
		t.Fatal("homunculus info window stayed open after delete")
	}
	if mode.ui.homunculusSkill.IsOpen() {
		t.Fatal("homunculus skill window stayed open after delete")
	}
	if _, ok := mode.companionAI.msg[300]; ok {
		t.Fatal("homunculus AI message remained after delete")
	}
	if _, ok := mode.companionAI.resMsg[300]; ok {
		t.Fatal("homunculus AI response message remained after delete")
	}
}

func TestHomunculusVanishWithoutDeleteKeepsSession(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			Name:   "Pipou",
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
			},
		},
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actorVanishOutOfSight})

	if !sess.Homunculus.Active || sess.Homunculus.ID != 300 {
		t.Fatalf("homunculus session should survive non-delete vanish: %+v", sess.Homunculus)
	}
}
