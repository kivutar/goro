package world

import "testing"

func TestMapPropertyPlayerCombatModes(t *testing.T) {
	for _, property := range []MapProperty{MapPropertyFreePvPZone, MapPropertyEventPvPZone, MapPropertyPvPServerZone} {
		if !property.PlayerCombatEnabled() {
			t.Fatalf("property %d did not enable player combat", property)
		}
	}
	for _, property := range []MapProperty{MapPropertyNothing, MapPropertyAgitZone, MapPropertyPKServerZone, MapPropertyDenySkillZone} {
		if property.PlayerCombatEnabled() {
			t.Fatalf("property %d unexpectedly enabled player combat", property)
		}
	}
}

func TestUpsertActorMovePreservesAppearance(t *testing.T) {
	w := New()
	w.UpsertActor(Actor{
		ID:         2000001,
		Name:       "remote",
		PartyName:  "Adventurers",
		X:          10,
		Y:          20,
		Job:        3,
		Head:       7,
		Weapon:     1201,
		Shield:     2101,
		HeadTop:    22,
		HeadMid:    33,
		HeadLow:    11,
		HeadPal:    8,
		BodyPal:    6,
		Sex:        1,
		IsAdmin:    true,
		Appearance: true,
	})

	w.UpsertActor(Actor{
		ID:     2000001,
		X:      12,
		Y:      24,
		Moving: true,
		FromX:  10,
		FromY:  20,
		ToX:    12,
		ToY:    24,
	})

	actor := w.Actors[2000001]
	if actor.Job != 3 || actor.Head != 7 || actor.Sex != 1 || actor.Weapon != 1201 || actor.Shield != 2101 || actor.HeadTop != 22 || actor.HeadMid != 33 || actor.HeadLow != 11 || actor.HeadPal != 8 || actor.BodyPal != 6 || !actor.Appearance {
		t.Fatalf("appearance not preserved: %+v", actor)
	}
	if !actor.IsAdmin {
		t.Fatalf("admin skin state not preserved: %+v", actor)
	}
	if !actor.Moving || actor.FromX != 10 || actor.FromY != 20 || actor.ToX != 12 || actor.ToY != 24 {
		t.Fatalf("movement state not stored: %+v", actor)
	}
	if actor.Name != "remote" {
		t.Fatalf("name = %q, want remote", actor.Name)
	}
	if actor.PartyName != "Adventurers" {
		t.Fatalf("party name = %q, want Adventurers", actor.PartyName)
	}
}

func TestUpsertActorMovePreservesObjectType(t *testing.T) {
	w := New()
	w.UpsertActor(Actor{
		ID:            2000001,
		X:             10,
		Y:             20,
		ObjectType:    5,
		HasObjectType: true,
	})

	w.UpsertActor(Actor{
		ID:     2000001,
		X:      12,
		Y:      24,
		Moving: true,
		FromX:  10,
		FromY:  20,
		ToX:    12,
		ToY:    24,
	})

	actor := w.Actors[2000001]
	if actor.ObjectType != 5 || !actor.HasObjectType {
		t.Fatalf("object type not preserved: %+v", actor)
	}
}

func TestUpsertActorMovePreservesCartState(t *testing.T) {
	w := New()
	w.UpsertActor(Actor{
		ID:           2000001,
		X:            10,
		Y:            20,
		Job:          5,
		HasCart:      true,
		CartNum:      3,
		HasCartState: true,
	})

	w.UpsertActor(Actor{
		ID:     2000001,
		X:      12,
		Y:      24,
		Moving: true,
		FromX:  10,
		FromY:  20,
		ToX:    12,
		ToY:    24,
	})

	actor := w.Actors[2000001]
	if !actor.HasCartState || !actor.HasCart || actor.CartNum != 3 {
		t.Fatalf("cart state not preserved: %+v", actor)
	}
}

func TestUpsertActorPreservesVendingState(t *testing.T) {
	w := New()
	w.UpsertActor(Actor{
		ID:          2000001,
		Vending:     true,
		VendingName: "Cheap pots",
	})

	w.UpsertActor(Actor{
		ID:         2000001,
		X:          10,
		Y:          20,
		Job:        5,
		Appearance: true,
	})

	actor := w.Actors[2000001]
	if !actor.Vending || actor.VendingName != "Cheap pots" {
		t.Fatalf("vending state not preserved: %+v", actor)
	}
}

func TestUpsertActorPreservesChatRoomState(t *testing.T) {
	w := New()
	w.UpsertActor(Actor{
		ID:             2000001,
		ChatRoom:       true,
		ChatRoomID:     77,
		ChatRoomTitle:  "Chat",
		ChatRoomCount:  1,
		ChatRoomLimit:  20,
		ChatRoomPublic: true,
	})

	w.UpsertActor(Actor{
		ID:         2000001,
		X:          10,
		Y:          20,
		Job:        5,
		Appearance: true,
	})

	actor := w.Actors[2000001]
	if !actor.ChatRoom || actor.ChatRoomID != 77 || actor.ChatRoomTitle != "Chat" || actor.ChatRoomCount != 1 || actor.ChatRoomLimit != 20 || !actor.ChatRoomPublic {
		t.Fatalf("chat room state not preserved: %+v", actor)
	}
}

func TestUpsertActorPreservesLevelWhenUpdateOmitsIt(t *testing.T) {
	w := New()
	w.UpsertActor(Actor{
		ID:       2000001,
		X:        10,
		Y:        20,
		Level:    99,
		HasLevel: true,
	})

	w.UpsertActor(Actor{ID: 2000001, X: 11, Y: 20})

	actor := w.Actors[2000001]
	if !actor.HasLevel || actor.Level != 99 {
		t.Fatalf("level = %d has=%t, want 99 true", actor.Level, actor.HasLevel)
	}
}

func TestUpsertActorPreservesSittingUntilMove(t *testing.T) {
	w := New()
	w.UpsertActor(Actor{ID: 2000001, X: 10, Y: 20, Sitting: true})

	w.UpsertActor(Actor{ID: 2000001, X: 10, Y: 20})
	if actor := w.Actors[2000001]; !actor.Sitting {
		t.Fatalf("sitting state was not preserved: %+v", actor)
	}

	w.UpsertActor(Actor{ID: 2000001, X: 11, Y: 20, Moving: true, FromX: 10, FromY: 20, ToX: 11, ToY: 20})
	if actor := w.Actors[2000001]; actor.Sitting {
		t.Fatalf("moving actor stayed sitting: %+v", actor)
	}
}

func TestRemoveActor(t *testing.T) {
	w := New()
	w.UpsertActor(Actor{ID: 2000002, X: 1, Y: 2})
	w.RemoveActor(2000002)
	if _, ok := w.Actors[2000002]; ok {
		t.Fatal("actor was not removed")
	}
}
