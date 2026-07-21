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

	mode.maybeQueueAggressiveCompanionTarget(ctx, companionAIHomunculus, 300)

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

	mode.maybeQueueAggressiveCompanionTarget(ctx, companionAIHomunculus, 300)

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

	if sess.Homunculus.HP != 18 || sess.Homunculus.MaxSP != 44 || sess.Homunculus.Exp != 5_000_000_000 || sess.Homunculus.MaxExp != 6_000_000_000 || sess.Homunculus.Skills.Points != 7 {
		t.Fatalf("homunculus session = %+v", sess.Homunculus)
	}
	if life := mode.actorLife[300]; life.hp != 18 || life.maxHP != 20 || life.sp != 5 || life.maxSP != 44 || !life.hasSP {
		t.Fatalf("actor life = %+v", life)
	}
	if actor := ctx.World.Actors[300]; actor.Speed != 150 || actor.AttackRange != 2 {
		t.Fatalf("actor = %+v", actor)
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
