package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestApplyMercenaryParamChangeUpdatesRobrowserStatusFields(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Mercenary: session.Companion{
			ID:          400,
			Active:      true,
			HP:          100,
			MaxHP:       200,
			SP:          30,
			MaxSP:       60,
			AttackRange: 2,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				400: {ID: 400, HasObjectType: true, ObjectType: actorObjectTypeMercenary},
			},
		},
	}

	mode.applyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusHP, Value: 77})
	mode.applyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusSP, Value: 12})
	mode.applyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusMercFlee, Value: 141})
	mode.applyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusMercKills, Value: 9})
	mode.applyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusMercFaith, Value: 321})

	if got := sess.Mercenary.HP; got != 77 {
		t.Fatalf("mercenary hp = %d, want 77", got)
	}
	if got := sess.Mercenary.SP; got != 12 {
		t.Fatalf("mercenary sp = %d, want 12", got)
	}
	if got := sess.Mercenary.Flee; got != 141 {
		t.Fatalf("mercenary flee = %d, want 141", got)
	}
	if got := sess.Mercenary.Kills; got != 9 {
		t.Fatalf("mercenary kills = %d, want 9", got)
	}
	if got := sess.Mercenary.Faith; got != 321 {
		t.Fatalf("mercenary faith = %d, want 321", got)
	}
	if life, ok := mode.actorLife[400]; !ok || life.hp != 77 || life.sp != 12 || !life.hasSP {
		t.Fatalf("actor life = %+v ok=%t, want updated mercenary life", life, ok)
	}
	if actor := ctx.World.Actors[400]; actor.AttackRange != 2 {
		t.Fatalf("actor attack range = %d, want 2", actor.AttackRange)
	}
}

func TestPendingMercenaryDeleteVanishClearsSession(t *testing.T) {
	mode := NewWorldMode()
	mode.mercDeleteID = 400
	mode.companionAI.msg = map[uint32]string{400: "3,500"}
	mode.companionAI.resMsg = map[uint32]string{400: "1,400"}
	sess := &session.Session{
		Mercenary: session.Companion{
			ID:     400,
			Active: true,
			Name:   "Sword Mercenary",
			HP:     1,
			MaxHP:  10,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				400: {ID: 400, HasObjectType: true, ObjectType: actorObjectTypeMercenary},
			},
		},
	}
	mode.ui.mercenaryInfo.OpenInfo(ctx, sess.Mercenary)
	mode.ui.mercenarySkill.Open(ctx, nil)

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 400, Reason: actorVanishOutOfSight})

	if sess.Mercenary.Active || sess.Mercenary.ID != 0 || sess.Mercenary.Name != "" {
		t.Fatalf("mercenary session after delete = %+v", sess.Mercenary)
	}
	if mode.mercDeleteID != 0 {
		t.Fatalf("pending delete id = %d, want 0", mode.mercDeleteID)
	}
	if _, ok := ctx.World.Actors[400]; ok {
		t.Fatal("deleted mercenary actor remained in world")
	}
	if mode.ui.mercenaryInfo.IsOpen() {
		t.Fatal("mercenary info window stayed open after delete")
	}
	if mode.ui.mercenarySkill.IsOpen() {
		t.Fatal("mercenary skill window stayed open after delete")
	}
	if _, ok := mode.companionAI.msg[400]; ok {
		t.Fatal("mercenary AI message remained after delete")
	}
	if _, ok := mode.companionAI.resMsg[400]; ok {
		t.Fatal("mercenary AI response message remained after delete")
	}
}
