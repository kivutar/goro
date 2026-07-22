package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
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

func TestMercenaryPropertyStoresDisplayLife(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				400: {ID: 400, HasObjectType: true, ObjectType: actorObjectTypeMercenary},
			},
		},
	}

	mode.applyMercenaryProperty(ctx, network.MercenaryProperty{
		ID:          400,
		Name:        "David",
		Level:       1,
		HP:          123,
		MaxHP:       456,
		SP:          78,
		MaxSP:       90,
		AttackRange: 2,
	})

	life, ok := mode.actorLifeForDisplay(ctx, ctx.World.Actors[400])
	if !ok {
		t.Fatal("mercenary display life missing")
	}
	if life.hp != 123 || life.maxHP != 456 || life.sp != 78 || life.maxSP != 90 || !life.hasSP || !life.friendly {
		t.Fatalf("mercenary display life = %+v", life)
	}
	if actor := ctx.World.Actors[400]; actor.Name != "David" || actor.AttackRange != 2 {
		t.Fatalf("mercenary actor = %+v", actor)
	}
}

func TestMercenaryLifeForDisplayUsesCachedParamChanges(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Mercenary: session.Companion{
			ID:     400,
			Active: true,
			HP:     100,
			MaxHP:  200,
			SP:     30,
			MaxSP:  60,
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

	life, ok := mode.actorLifeForDisplay(ctx, ctx.World.Actors[400])
	if !ok {
		t.Fatal("mercenary display life missing")
	}
	if life.hp != 77 || life.maxHP != 200 || life.sp != 12 || life.maxSP != 60 || !life.hasSP || !life.friendly {
		t.Fatalf("mercenary display life = %+v", life)
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

func TestMercenaryActionFamiliesUseGuildWeaponActions(t *testing.T) {
	archer := worldstate.Actor{Job: 6017, HasObjectType: true, ObjectType: actorObjectTypeMercenary}
	lancer := worldstate.Actor{Job: 6027, HasObjectType: true, ObjectType: actorObjectTypeMercenary}
	sword := worldstate.Actor{Job: 6037, HasObjectType: true, ObjectType: actorObjectTypeMercenary}
	mob := worldstate.Actor{Job: 1002, HasObjectType: true, ObjectType: actorObjectTypeMob}

	if got := attackActionFamilyForActor(archer); got != spriteActionPCAttack2 {
		t.Fatalf("archer mercenary attack action = %d, want %d", got, spriteActionPCAttack2)
	}
	if got := attackActionFamilyForActor(lancer); got != spriteActionPCAttack3 {
		t.Fatalf("lancer mercenary attack action = %d, want %d", got, spriteActionPCAttack3)
	}
	if got := attackActionFamilyForActor(sword); got != spriteActionPCAttack2 {
		t.Fatalf("sword mercenary attack action = %d, want %d", got, spriteActionPCAttack2)
	}
	if got := hurtActionFamilyForActor(archer); got != spriteActionPCHurt {
		t.Fatalf("archer mercenary hurt action = %d, want %d", got, spriteActionPCHurt)
	}
	if got := deathActionFamilyForActor(archer); got != spriteActionNonPCDeath {
		t.Fatalf("archer mercenary death action = %d, want %d", got, spriteActionNonPCDeath)
	}
	if got := deathActionFamilyForActor(sword); got != spriteActionPCDeath {
		t.Fatalf("sword mercenary death action = %d, want %d", got, spriteActionPCDeath)
	}
	if got := skillCastActionFamilyForActor(sword, 0); got != spriteActionPCSkill {
		t.Fatalf("sword mercenary cast action = %d, want %d", got, spriteActionPCSkill)
	}
	if got := readyFightSkillActionSpec.actionFamilyForActor(sword); got != spriteActionPCSkill {
		t.Fatalf("sword mercenary readyfight skill action = %d, want %d", got, spriteActionPCSkill)
	}
	if got := attackActionFamilyForActor(mob); got != spriteActionNonPCAttack {
		t.Fatalf("mob attack action = %d, want %d", got, spriteActionNonPCAttack)
	}
}

func TestMercenarySpriteKeyDerivesSexFromMercenaryResource(t *testing.T) {
	archer := worldstate.Actor{Job: 6017, Sex: 1, HasObjectType: true, ObjectType: actorObjectTypeMercenary}
	sword := worldstate.Actor{Job: 6037, Sex: 0, HasObjectType: true, ObjectType: actorObjectTypeMercenary}

	if got := mercenarySpriteKeyForActor(archer).sex; got != 0 {
		t.Fatalf("archer mercenary sex = %d, want female resource sex 0", got)
	}
	if got := mercenarySpriteKeyForActor(sword).sex; got != 1 {
		t.Fatalf("sword mercenary sex = %d, want male resource sex 1", got)
	}
}

func TestMercenarySpriteKeyUsesGuildDefaultWeapon(t *testing.T) {
	cases := []struct {
		name string
		job  int16
		want int
	}{
		{name: "archer", job: 6017, want: db.WeaponBow},
		{name: "lancer", job: 6027, want: db.WeaponSpear},
		{name: "sword", job: 6037, want: db.WeaponSword},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actor := worldstate.Actor{Job: tc.job, HasObjectType: true, ObjectType: actorObjectTypeMercenary}
			if got := mercenarySpriteKeyForActor(actor).weapon; got != tc.want {
				t.Fatalf("default weapon = %d, want %d", got, tc.want)
			}
		})
	}

	actor := worldstate.Actor{Job: 6037, Weapon: 7, HasObjectType: true, ObjectType: actorObjectTypeMercenary}
	if got := mercenarySpriteKeyForActor(actor).weapon; got != 7 {
		t.Fatalf("explicit weapon = %d, want 7", got)
	}
}

func TestMercenaryWeaponBaseJobUsesGuildHumanoidResources(t *testing.T) {
	cases := []struct {
		name string
		job  int
		want int
	}{
		{name: "archer", job: 6017, want: db.JobArcher},
		{name: "lancer", job: 6027, want: db.JobSwordman},
		{name: "sword", job: 6037, want: db.JobSwordman},
		{name: "unknown", job: 9999, want: db.JobNovice},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mercenaryWeaponBaseJob(tc.job); got != tc.want {
				t.Fatalf("weapon base job = %d, want %d", got, tc.want)
			}
		})
	}
}
