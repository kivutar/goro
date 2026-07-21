package game

import (
	"testing"

	"github.com/kivutar/goro/client"
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
