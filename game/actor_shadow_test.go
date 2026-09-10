package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	worldstate "github.com/kivutar/goro/world"
)

func TestActorShadowSuppressionUsesActorActions(t *testing.T) {
	player := worldstate.Actor{ID: 200, Job: db.JobAcolyte}
	bat := worldstate.Actor{ID: 300, Job: 1005}
	homunculus := worldstate.Actor{ID: 400, Job: 6001}
	archer := worldstate.Actor{ID: 500, Job: 6017, ObjectType: actorObjectTypeMercenary, HasObjectType: true}
	lancer := worldstate.Actor{ID: 600, Job: 6027, ObjectType: actorObjectTypeMercenary, HasObjectType: true}
	fencer := worldstate.Actor{ID: 700, Job: 6037, ObjectType: actorObjectTypeMercenary, HasObjectType: true}
	for _, tc := range []struct {
		name       string
		actor      worldstate.Actor
		action     int
		suppressed bool
	}{
		{"player idle", player, spriteActionIdle, false},
		{"player attack 1", player, spriteActionPCAttack1, false},
		{"player attack 2", player, spriteActionPCAttack2, false},
		{"player attack 3", player, spriteActionPCAttack3, false},
		{"player combat ready", player, spriteActionPCReadyFight, false},
		{"player sitting", player, spriteActionSit, true},
		{"player dead", player, spriteActionPCDeath, true},
		{"bat idle", bat, spriteActionIdle, false},
		{"bat attacking", bat, spriteActionNonPCAttack, false},
		{"bat hurt", bat, spriteActionNonPCHurt, false},
		{"bat dead", bat, spriteActionNonPCDeath, true},
		{"monster special", bat, spriteActionNonPCSpecial, false},
		{"homunculus attacking", homunculus, spriteActionNonPCAttack, false},
		{"homunculus dead", homunculus, spriteActionNonPCDeath, true},
		{"archer attacking", archer, spriteActionPCAttack2, false},
		{"archer sitting", archer, spriteActionSit, true},
		{"archer dead", archer, spriteActionNonPCDeath, true},
		// Archer mercenaries use action 8 for freeze, not player death.
		{"archer frozen", archer, 8, false},
		{"lancer attacking", lancer, spriteActionPCAttack3, false},
		{"lancer dead", lancer, spriteActionPCDeath, true},
		{"fencer attacking", fencer, spriteActionPCAttack2, false},
		{"fencer dead", fencer, spriteActionPCDeath, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			mode := &WorldMode{actorAnims: map[uint32]actorAnimation{
				tc.actor.ID: {actionFamily: tc.action, started: now, loop: true},
			}}
			if got := mode.actorShadowSuppressed(tc.actor, now); got != tc.suppressed {
				t.Fatalf("shadow suppressed = %t, want %t", got, tc.suppressed)
			}
		})
	}
}

func TestActorShadowRemainsAfterMaceSwing(t *testing.T) {
	now := time.Now()
	player := worldstate.Actor{ID: 200, Job: db.JobAcolyte, Weapon: db.WeaponMace}
	mode := &WorldMode{actorAnims: map[uint32]actorAnimation{
		player.ID: {
			actionFamily: attackActionFamilyForActor(player),
			started:      now,
			duration:     time.Second,
			next:         readyFightAnimation(now.Add(time.Second)),
		},
	}}
	for _, elapsed := range []time.Duration{0, time.Second, 2 * time.Second} {
		if mode.actorShadowSuppressed(player, now.Add(elapsed)) {
			t.Fatalf("shadow disappeared at %s into the attack/ready transition", elapsed)
		}
	}
}

func TestSittingActorSuppressesShadowWithoutAnimation(t *testing.T) {
	mode := &WorldMode{}
	player := worldstate.Actor{ID: 200, Job: db.JobAcolyte, Sitting: true}
	if !mode.actorShadowSuppressed(player, time.Now()) {
		t.Fatal("sitting player retained their shadow")
	}
}
