package game

import (
	"context"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestWarpPortalActorEntryAddsWarpZoneEffect(t *testing.T) {
	for _, job := range []int16{actorJobWarpPortal, actorJobWarpPortalActive} {
		t.Run(fmt.Sprintf("job_%d", job), func(t *testing.T) {
			world := worldstate.New()
			sessionState := &session.Session{AccountID: 2000000}
			mode := &WorldMode{}
			ctx := client.Context{Session: sessionState, World: world}
			entry := network.ActorEntry{ID: 900, Job: job, X: 30, Y: 40}

			upsertNetworkActor(ctx, entry)
			mode.applyWarpPortalEntry(ctx, entry)
			mode.applyWarpPortalEntry(ctx, entry)

			if len(mode.worldEffects) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
			}
			if effect := mode.worldEffects[0]; effect.actorID != 900 || effect.effectID != effectWarpZone2 || effect.x != 30 || effect.y != 40 || effect.duration != warpPortalActorEffectLifetime {
				t.Fatalf("effect = %+v", effect)
			}
		})
	}
}

func TestWaitingWarpPortalActorEntryAddsReadyPortalEffect(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}
	entry := network.ActorEntry{ID: 900, Job: actorJobWarpPortalWaiting, X: 30, Y: 40}

	upsertNetworkActor(ctx, entry)
	mode.applyWarpPortalEntry(ctx, entry)

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 900 || effect.effectID != effectReadyPortal || effect.x != 30 || effect.y != 40 || effect.duration != warpPortalActorEffectLifetime {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestWarpPortalActorEntryReplacesWaitingEffect(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}
	entry := network.ActorEntry{ID: 900, Job: actorJobWarpPortalWaiting, X: 30, Y: 40}

	upsertNetworkActor(ctx, entry)
	mode.applyWarpPortalEntry(ctx, entry)
	entry.Job = actorJobWarpPortalActive
	upsertNetworkActor(ctx, entry)
	mode.applyWarpPortalEntry(ctx, entry)

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 900 || effect.effectID != effectWarpZone2 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestRemoveActorNowRemovesActorWorldEffects(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}
	entry := network.ActorEntry{ID: 900, Job: actorJobWarpPortal, X: 30, Y: 40}

	upsertNetworkActor(ctx, entry)
	mode.applyWarpPortalEntry(ctx, entry)
	mode.removeActorNow(ctx, entry.ID)

	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after remove = %+v, want none", mode.worldEffects)
	}
}

func TestWarpActorUsesCenteredWorldAnchor(t *testing.T) {
	actor := worldstate.Actor{Job: actorJobWarpPortal}
	x, y := actorWorldAnchor(actor, 30, 40)
	if x != 30.5 || y != 40.5 {
		t.Fatalf("warp anchor = %.1f, %.1f; want 30.5, 40.5", x, y)
	}
}

func TestNormalActorUsesCenteredWorldAnchor(t *testing.T) {
	actor := worldstate.Actor{Job: 1002}
	x, y := actorWorldAnchor(actor, 30, 40)
	if x != 30.5 || y != 40.5 {
		t.Fatalf("normal actor anchor = %.1f, %.1f; want 30.5, 40.5", x, y)
	}
}

func TestApplyActorActionNotifyUpdatesLocalSitState(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Moving: true}
	mode := &WorldMode{
		actorAnims: map[uint32]actorAnimation{
			2000000: {actionFamily: spriteActionPCReadyFight, loop: true},
			150000:  {actionFamily: spriteActionPCReadyFight, loop: true},
		},
		pendingAttack:  attackIntent{targetID: 300},
		lockedAttackID: 300,
		attackFocusID:  300,
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000,
		Action:   network.ActionSitDown,
	})
	if !world.Player.Sitting {
		t.Fatal("local player did not sit")
	}
	if world.Player.Moving {
		t.Fatal("local player kept moving while sitting")
	}
	if _, ok := mode.actorAnims[2000000]; ok {
		t.Fatal("local account animation survived sitting")
	}
	if _, ok := mode.actorAnims[150000]; ok {
		t.Fatal("local character animation survived sitting")
	}
	if mode.pendingAttack.targetID != 0 || mode.lockedAttackID != 0 || mode.attackFocusID != 0 {
		t.Fatalf("local attack intent survived sitting: pending=%d locked=%d focus=%d", mode.pendingAttack.targetID, mode.lockedAttackID, mode.attackFocusID)
	}

	mode.actorAnims[2000000] = actorAnimation{actionFamily: spriteActionPCReadyFight, loop: true}
	mode.actorAnims[150000] = actorAnimation{actionFamily: spriteActionPCReadyFight, loop: true}
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000,
		Action:   network.ActionStandUp,
	})
	if world.Player.Sitting {
		t.Fatal("local player did not stand")
	}
	if _, ok := mode.actorAnims[2000000]; ok {
		t.Fatal("local account animation survived standing")
	}
	if _, ok := mode.actorAnims[150000]; ok {
		t.Fatal("local character animation survived standing")
	}
}

func TestApplyActorActionNotifyUpdatesRemoteSitState(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, X: 10, Y: 20, Moving: true})
	mode := &WorldMode{actorAnims: map[uint32]actorAnimation{
		300: {actionFamily: spriteActionPCReadyFight, loop: true},
	}}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 300,
		Action:   network.ActionSitDown,
	})
	if actor := world.Actors[300]; !actor.Sitting || actor.Moving {
		t.Fatalf("remote actor sit state = sitting %t moving %t", actor.Sitting, actor.Moving)
	}
	if _, ok := mode.actorAnims[300]; ok {
		t.Fatal("remote actor animation survived sitting")
	}

	mode.actorAnims[300] = actorAnimation{actionFamily: spriteActionPCReadyFight, loop: true}
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 300,
		Action:   network.ActionStandUp,
	})
	if actor := world.Actors[300]; actor.Sitting {
		t.Fatalf("remote actor stayed sitting: %+v", actor)
	}
	if _, ok := mode.actorAnims[300]; ok {
		t.Fatal("remote actor animation survived standing")
	}
}

func TestPickupWaitsForServerAnimationAndDisappear(t *testing.T) {
	networkClient, serverConn := newBotTestConnection(t, 20080910)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	item := worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20, Amount: 1}
	world.UpsertItem(item)
	mode := &WorldMode{
		actorAnims: make(map[uint32]actorAnimation),
	}
	mode.actorAnims[150000] = actorAnimation{actionFamily: spriteActionPCReadyFight, loop: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
		},
		Network: networkClient,
		World:   world,
	}

	if !mode.requestPickup(ctx, item, "test") {
		t.Fatal("pickup request was rejected")
	}
	readBotTestPackets(t, serverConn, network.BuildItemPickupPacketForClientDate(item.ID, 20080910))

	if _, ok := world.Items[item.ID]; !ok {
		t.Fatal("pickup request removed item before server disappearance")
	}
	if anim := mode.actorAnims[150000]; anim.actionFamily != spriteActionPCReadyFight || len(mode.actorAnims) != 1 {
		t.Fatalf("pickup request changed the player's animation before server confirmation: %+v", mode.actorAnims)
	}
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: world.Player.ID,
		TargetID: item.ID,
		Action:   network.ActorActionPickupItem,
	})
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("local pickup animation missing")
	}
	if anim.actionFamily != spriteActionPickup {
		t.Fatalf("pickup action = %d, want %d", anim.actionFamily, spriteActionPickup)
	}
	if anim.next != nil {
		t.Fatalf("pickup ack animation next = %+v, want nil so it returns to idle", anim.next)
	}
	if expired, ok := mode.actorAnimation(150000, anim.started.Add(anim.duration)); ok {
		t.Fatalf("pickup request expired animation = %+v, want idle fallback", expired)
	}
	if world.Dir != directionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
	}

	mode.applyItemPickupAck(ctx, network.ItemPickupAck{ItemID: 909, Amount: 1})
	if _, ok := world.Items[item.ID]; !ok {
		t.Fatal("pickup acknowledgment removed item before server disappearance")
	}

	mode.applyFloorItemDisappear(ctx, network.FloorItemDisappear{ID: item.ID})
	if _, ok := world.Items[item.ID]; ok {
		t.Fatal("server disappearance did not remove picked item")
	}
}

func TestRejectedPickupDoesNotAnimate(t *testing.T) {
	networkClient, serverConn := newBotTestConnection(t, 20080910)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	item := worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20, Amount: 1}
	world.UpsertItem(item)
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: world.Player.ID},
		Network: networkClient,
		World:   world,
	}
	if !mode.requestPickup(ctx, item, "test") {
		t.Fatal("pickup request was not sent")
	}
	readBotTestPackets(t, serverConn, network.BuildItemPickupPacketForClientDate(item.ID, 20080910))
	mode.applyItemPickupAck(ctx, network.ItemPickupAck{ItemID: item.ItemID, Result: 1})
	if len(mode.actorAnims) != 0 || len(mode.worldEffects) != 0 || len(mode.scheduledSounds) != 0 {
		t.Fatalf("rejected pickup played an animation, effect or sound: animations=%+v effects=%+v sounds=%+v", mode.actorAnims, mode.worldEffects, mode.scheduledSounds)
	}
	if len(ctx.Session.Inventory.Items) != 0 || len(world.Items) != 1 {
		t.Fatal("rejected pickup changed the inventory or floor items")
	}
}

func TestApplyItemPickupAckReceiveItemReplacesStackAndReportsGain(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 512, Amount: 3, Identified: true}},
		},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState}

	item, gained, ok := mode.applyItemPickupAck(ctx, network.ItemPickupAck{
		Index:      7,
		ItemID:     512,
		Amount:     5,
		Type:       0,
		Identified: true,
		Result:     itemPickupResultReceive,
	})

	if !ok {
		t.Fatal("receive item ack was not treated as an inventory add")
	}
	if gained != 2 {
		t.Fatalf("gained = %d, want 2", gained)
	}
	if item.Amount != 5 {
		t.Fatalf("display item amount = %d, want server total 5", item.Amount)
	}
	if got := sessionState.Inventory.Items[0].Amount; got != 5 {
		t.Fatalf("inventory amount = %d, want replaced server total 5", got)
	}
}

func TestApplyItemPickupAckFailureDoesNotAddItemOrClearPendingPickup(t *testing.T) {
	sessionState := &session.Session{}
	mode := &WorldMode{
		pendingPickup: pickupIntent{itemID: 9001},
	}
	ctx := client.Context{Session: sessionState}

	if _, _, ok := mode.applyItemPickupAck(ctx, network.ItemPickupAck{Index: 7, ItemID: 512, Amount: 1, Result: 1}); ok {
		t.Fatal("failed pickup ack was treated as success")
	}
	if len(sessionState.Inventory.Items) != 0 {
		t.Fatalf("inventory items = %+v, want none", sessionState.Inventory.Items)
	}
	if mode.pendingPickup.itemID != 9001 {
		t.Fatalf("pending pickup = %d, want 9001", mode.pendingPickup.itemID)
	}
}

func TestApplyItemPickupAckSuccessDoesNotClearPendingPickup(t *testing.T) {
	mode := &WorldMode{
		pendingPickup: pickupIntent{itemID: 9002},
	}
	ctx := client.Context{Session: &session.Session{}}

	if _, _, ok := mode.applyItemPickupAck(ctx, network.ItemPickupAck{
		Index:  7,
		ItemID: 512,
		Amount: 1,
		Result: itemPickupResultSuccess,
	}); !ok {
		t.Fatal("successful pickup acknowledgment was rejected")
	}
	if mode.pendingPickup.itemID != 9002 {
		t.Fatalf("pending pickup = %d, want 9002", mode.pendingPickup.itemID)
	}
}

func TestApplyActorPickupActionNotifyStartsPickupInsteadOfAttack(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertItem(worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20, Amount: 1})
	mode := &WorldMode{
		actorAnims: make(map[uint32]actorAnimation),
	}
	mode.actorAnims[150000] = actorAnimation{actionFamily: spriteActionPCReadyFight, loop: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
		},
		World: world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000,
		TargetID: 9001,
		Action:   network.ActorActionPickupItem,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("local pickup animation missing")
	}
	if anim.actionFamily != spriteActionPickup {
		t.Fatalf("pickup action = %d, want %d", anim.actionFamily, spriteActionPickup)
	}
	if anim.next != nil {
		t.Fatalf("pickup animation next = %+v, want nil so it returns to idle", anim.next)
	}
	if expired, ok := mode.actorAnimation(150000, anim.started.Add(anim.duration)); ok {
		t.Fatalf("pickup expired animation = %+v, want idle fallback", expired)
	}
	if len(mode.damageFloaters) != 0 {
		t.Fatalf("pickup notify should not create damage floaters: %+v", mode.damageFloaters)
	}
	if world.Dir != directionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
	}
}

func TestApplyActorHPUpdateStoresExactLife(t *testing.T) {
	mode := &WorldMode{}
	mode.applyActorHPUpdate(network.ActorHPUpdate{ID: 300, HP: 12, MaxHP: 48})

	life, ok := mode.actorLife[300]
	if !ok {
		t.Fatal("life missing")
	}
	if life.hp != 12 || life.maxHP != 48 {
		t.Fatalf("life = %+v, want exact 12/48", life)
	}
}

func TestCombatDamageDoesNotInventMonsterLife(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		Job:           1008,
		X:             11,
		Y:             20,
		HasObjectType: true,
		ObjectType:    actorObjectTypeMob,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	if _, ok := mode.actorLife[300]; ok {
		t.Fatal("life should not be created from combat damage")
	}
}

func TestCombatDamageDoesNotMutateExactMonsterLife(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		Job:           1002,
		X:             11,
		Y:             20,
		HasObjectType: true,
		ObjectType:    actorObjectTypeMob,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorHPUpdate(network.ActorHPUpdate{ID: 300, HP: 50, MaxHP: 100})
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      12,
		Action:      0,
	})

	life, ok := mode.actorLife[300]
	if !ok {
		t.Fatal("life missing")
	}
	if life.hp != 50 || life.maxHP != 100 {
		t.Fatalf("exact life = %+v, want unchanged 50/100", life)
	}
}

func TestActorLifeForDisplayUsesLocalPlayerHPAndSP(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Vitals: session.Vitals{
				HP:    75,
				MaxHP: 100,
				SP:    8,
				MaxSP: 20,
			},
		},
	}

	life, ok := mode.actorLifeForDisplay(ctx, worldstate.Actor{ID: 150000})
	if !ok {
		t.Fatal("local player life missing")
	}
	if life.hp != 75 || life.maxHP != 100 || life.sp != 8 || life.maxSP != 20 || !life.hasSP || !life.player {
		t.Fatalf("local player life = %+v", life)
	}
}

func TestActorLifeForDisplayHidesMonsterHPBarsFor2008Client(t *testing.T) {
	mode := &WorldMode{
		actorLife: map[uint32]actorLife{
			300: {hp: 12, maxHP: 48},
		},
	}
	ctx := client.Context{
		Session: &session.Session{CharID: 150000},
	}

	actor := worldstate.Actor{ID: 300, ObjectType: actorObjectTypeMob, HasObjectType: true}
	if _, ok := mode.actorLifeForDisplay(ctx, actor); ok {
		t.Fatal("monster HP bar should be hidden for the 2008 client profile")
	}
	if life, ok := mode.monsterLifeForSense(300); !ok || life.hp != 12 || life.maxHP != 48 {
		t.Fatalf("sense life cache = %+v ok=%v, want 12/48", life, ok)
	}
}

func TestActorOverlayLifeBarIsBelowNameLabel(t *testing.T) {
	nameY := actorNameLabelY(100, 1.2)
	barY := actorLifeBarY(100, 1.2)
	if barY <= nameY+10 {
		t.Fatalf("bar y = %.1f, name y = %.1f; want bar below name", barY, nameY)
	}
}

func TestActorOverlayCastBarIsBelowSpeechBubble(t *testing.T) {
	bubbleBottomY := actorSpeechBubbleBottomY(100, 1.2)
	barY := actorCastBarY(100, 1.2)
	if barY <= bubbleBottomY {
		t.Fatalf("cast bar y = %.1f, bubble bottom y = %.1f; want cast bar below bubble", barY, bubbleBottomY)
	}
}

func TestActorOverlayBarFillWidthRoundsAndClamps(t *testing.T) {
	if got, want := actorOverlayBarFillWidth(0.333), math.Round((actorOverlayBarWidth-2)*0.333); got != want {
		t.Fatalf("fill width = %.1f, want %.1f", got, want)
	}
	if got := actorOverlayBarFillWidth(1.5); got != actorOverlayBarWidth-2 {
		t.Fatalf("clamped fill width = %.1f, want %.1f", got, actorOverlayBarWidth-2)
	}
	if got := actorOverlayBarFillWidth(math.NaN()); got != 0 {
		t.Fatalf("nan fill width = %.1f, want 0", got)
	}
}

func TestLocalPlayerNameIsBelowHPAndSPBars(t *testing.T) {
	life := actorLife{hasSP: true}
	barY := actorLifeBarY(100, 1.2)
	nameY := actorNameBelowLifeBarY(100, 1.2, life)
	if nameY <= barY+actorLifeBarHeight(life) {
		t.Fatalf("name y = %.1f, bar y = %.1f; want name below hp/sp bars", nameY, barY)
	}
}

func TestActorLifeBarHeightAddsHomunculusHungerRow(t *testing.T) {
	if got := actorLifeBarHeight(actorLife{hasSP: true, hasHunger: true}); got != 13 {
		t.Fatalf("life bar height = %.1f, want hp/sp/hunger height", got)
	}
}

func TestHomunculusNameYUsesThreeBarHeight(t *testing.T) {
	life := actorLife{hasSP: true, hasHunger: true}
	barY := actorLifeBarY(100, 1.2)
	nameY := actorNameBelowLifeBarY(100, 1.2, life)
	if got := nameY - (barY + actorLifeBarHeight(life)); got != 3 {
		t.Fatalf("name gap = %.1f, want 3px below hp/sp/hunger bars", got)
	}
}

func TestCombatHitDelayUsesActionSoundMotion(t *testing.T) {
	action := res.ACTAction{Animations: []res.ACTAnimation{
		{Sound: -1},
		{Sound: -1},
		{Sound: 0},
		{Sound: -1},
	}}
	if got := combatHitDelayFromAction(action, 800*time.Millisecond); got != 400*time.Millisecond {
		t.Fatalf("hit delay = %s, want 400ms", got)
	}
}

func TestCombatHitDelayFallsBackToMidpoint(t *testing.T) {
	action := res.ACTAction{Animations: []res.ACTAnimation{
		{Sound: -1},
		{Sound: -1},
		{Sound: -1},
		{Sound: -1},
	}}
	if got := combatHitDelayFromAction(action, 800*time.Millisecond); got != 400*time.Millisecond {
		t.Fatalf("hit delay = %s, want midpoint", got)
	}
}

func TestAttackActionFamilyUsesRobrowserWizardRodAction(t *testing.T) {
	femaleWizard := worldstate.Actor{Job: db.JobWizard, Sex: 0, Weapon: 1601}
	if got := attackActionFamilyForActor(femaleWizard); got != spriteActionPCAttack3 {
		t.Fatalf("female Wizard rod attack action = %d, want ATTACK3", got)
	}
	maleWizard := worldstate.Actor{Job: db.JobWizard, Sex: 1, Weapon: 1601}
	if got := attackActionFamilyForActor(maleWizard); got != spriteActionPCAttack2 {
		t.Fatalf("male Wizard rod attack action = %d, want ATTACK2", got)
	}
	unarmedWizard := worldstate.Actor{Job: db.JobWizard, Sex: 0}
	if got := attackActionFamilyForActor(unarmedWizard); got != spriteActionPCAttack1 {
		t.Fatalf("female Wizard unarmed attack action = %d, want ATTACK1", got)
	}
	leftHandRod := worldstate.Actor{Job: db.JobWizard, Sex: 0, Shield: 1601}
	if got := attackActionFamilyForActor(leftHandRod); got != spriteActionPCAttack3 {
		t.Fatalf("female Wizard left-hand rod attack action = %d, want ATTACK3", got)
	}
}

func TestAttackActionFamilyUsesExpandedWeaponViewIDs(t *testing.T) {
	knightSword := worldstate.Actor{Job: db.JobKnight, Weapon: 48}
	if got := attackActionFamilyForActor(knightSword); got != spriteActionPCAttack2 {
		t.Fatalf("Knight two-hand sword view attack action = %d, want ATTACK2", got)
	}
	archerBow := worldstate.Actor{Job: db.JobArcher, Weapon: 73}
	if got := attackActionFamilyForActor(archerBow); got != spriteActionPCAttack2 {
		t.Fatalf("Archer CrossBow view attack action = %d, want ATTACK2", got)
	}
	hunterBow := worldstate.Actor{Job: db.JobHunter, Weapon: 74}
	if got := attackActionFamilyForActor(hunterBow); got != spriteActionPCAttack3 {
		t.Fatalf("Hunter Arbalest view attack action = %d, want ATTACK3", got)
	}
}

func TestActorActionFrameDelayUsesPlayerWeaponActionFrames(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 1, Weapon: 3, Dir: 0}
	actionFamily := attackActionFamilyForActor(world.Player)
	mode := &WorldMode{playerView: humanoidTimingView(actionFamily, 4)}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	if got := mode.actorActionFrameDelay(ctx, world.Player, actionFamily, 800*time.Millisecond); got != 200*time.Millisecond {
		t.Fatalf("frame delay = %s, want attack duration divided by weapon action frames", got)
	}
}

func TestActorActionNotifySetsPlayerWeaponAttackFrameDelay(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 1, Weapon: 3, Dir: 0}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	actionFamily := attackActionFamilyForActor(world.Player)
	mode := &WorldMode{playerView: humanoidTimingView(actionFamily, 4)}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Weapon: 3}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 800,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("attack animation missing")
	}
	if !anim.hasSpeed || anim.speed != 200*time.Millisecond {
		t.Fatalf("attack animation = %+v, want frame delay from packet speed/action frame count", anim)
	}
}

func humanoidTimingView(actionFamily int, frames int) *humanoidSpriteView {
	actions := make([]res.ACTAction, actionFamily*8+8)
	for dir := 0; dir < 8; dir++ {
		actions[actionFamily*8+dir] = res.ACTAction{DelayMS: 150, Animations: make([]res.ACTAnimation, frames)}
	}
	return &humanoidSpriteView{body: &spriteView{act: &res.ACT{Actions: actions}}}
}

func humanoidSoundView(actionFamily int, frames int, soundMotion int, sounds []string) *humanoidSpriteView {
	actions := make([]res.ACTAction, actionFamily*8+8)
	for dir := 0; dir < 8; dir++ {
		animations := make([]res.ACTAnimation, frames)
		for i := range animations {
			animations[i].Sound = -1
		}
		if soundMotion >= 0 && soundMotion < len(animations) {
			animations[soundMotion].Sound = 0
		}
		actions[actionFamily*8+dir] = res.ACTAction{DelayMS: 150, Animations: animations}
	}
	return &humanoidSpriteView{body: &spriteView{act: &res.ACT{Actions: actions, Sounds: sounds}}}
}

func TestActionSoundNameResolvesACTSound(t *testing.T) {
	act := &res.ACT{Sounds: []string{"attack.wav"}}
	action := res.ACTAction{Animations: []res.ACTAnimation{{Sound: -1}, {Sound: 0}}}
	if got := actionSoundName(act, action, 1); got != "attack.wav" {
		t.Fatalf("sound = %q, want attack.wav", got)
	}
}

func TestActionSoundNameIgnoresAttackMarker(t *testing.T) {
	act := &res.ACT{Sounds: []string{"atk"}}
	action := res.ACTAction{Animations: []res.ACTAnimation{{Sound: 0}}}
	if got := actionSoundName(act, action, 0); got != "" {
		t.Fatalf("sound = %q, want empty marker", got)
	}
}

func TestActionAttackMarkerMotionPrefersATKOverEarlierSound(t *testing.T) {
	act := &res.ACT{Sounds: []string{"clothes.wav", "atk"}}
	action := res.ACTAction{Animations: []res.ACTAnimation{
		{Sound: -1},
		{Sound: 0},
		{Sound: -1},
		{Sound: 1},
	}}
	if got := actionAttackMarkerMotion(act, action); got != 3 {
		t.Fatalf("attack marker motion = %d, want ATK motion 3", got)
	}
}

func TestSkillHitSoundUsesReferenceEnemyNormalSounds(t *testing.T) {
	source := worldstate.Actor{Job: db.JobWizard, Weapon: 1601}
	target := worldstate.Actor{Job: 1002, ObjectType: actorObjectTypeMob, HasObjectType: true}

	got := combatHitSFXCandidates(network.ActorActionNotify{SkillID: db.SkillWZEarthspike}, source, true, target, true)
	requireSameStringSet(t, got, db.EnemyHitNormalSounds())
	if got[0] == "_hit_rod.wav" || got[0] == "_hit_arrow.wav" {
		t.Fatalf("skill hit sound = %q, want enemy hit normal sound", got[0])
	}
}

func TestEarthSpikeSchedulesEnemyHitAndSpikeSounds(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: db.JobWizard, Weapon: 1601, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SkillID:     db.SkillWZEarthspike,
		SourceSpeed: 800,
		TargetSpeed: 480,
		Damage:      84,
		Action:      network.ActorActionSkill,
	})

	if len(mode.scheduledSounds) != 2 {
		t.Fatalf("scheduled sounds = %+v, want enemy hit and earth spike sounds", mode.scheduledSounds)
	}
	if got := mode.scheduledSounds[0].paths; !sameStringSet(got, db.EnemyHitNormalSounds()) {
		t.Fatalf("enemy hit sound candidates = %v, want rotated %v", got, db.EnemyHitNormalSounds())
	}
	if sound := mode.scheduledSounds[0]; !sound.positioned || sound.actorID != 300 {
		t.Fatalf("enemy hit sound source = %+v, want target actor", sound)
	}
	if got := mode.scheduledSounds[1].paths; len(got) != 1 || got[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("earth spike sound = %+v", mode.scheduledSounds[1])
	}
}

func requireSameStringSet(t *testing.T, got, want []string) {
	t.Helper()
	if !sameStringSet(got, want) {
		t.Fatalf("strings = %v, want same set as %v", got, want)
	}
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(want))
	for _, value := range want {
		seen[value]++
	}
	for _, value := range got {
		if seen[value] == 0 {
			return false
		}
		seen[value]--
	}
	return true
}

func TestMercenaryAttackSchedulesWeaponSwingAndHitSounds(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 8, Y: 20, Dir: 4}
	mercenary := worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		Job:           6037,
		ObjectType:    actorObjectTypeMercenary,
		HasObjectType: true,
	}
	world.UpsertActor(mercenary)
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	actionFamily := attackActionFamilyForActor(mercenary)
	mode := &WorldMode{
		mercenaryViews: map[actorSpriteKey]*humanoidSpriteView{
			mercenarySpriteKeyForActor(mercenary): humanoidSoundView(actionFamily, 2, 1, []string{"atk"}),
		},
	}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    300,
		SourceSpeed: 800,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	if len(mode.scheduledSounds) != 2 {
		t.Fatalf("scheduled sounds = %+v, want swing and hit sounds", mode.scheduledSounds)
	}
	if got := mode.scheduledSounds[0].paths; len(got) != 1 || got[0] != "attack_sword.wav" {
		t.Fatalf("swing sound = %+v", mode.scheduledSounds[0])
	}
	if sound := mode.scheduledSounds[0]; !sound.positioned || sound.actorID != 400 {
		t.Fatalf("swing sound source = %+v, want source actor", sound)
	}
	if got := mode.scheduledSounds[1].paths; len(got) != 1 || got[0] != "_hit_sword.wav" {
		t.Fatalf("hit sound = %+v", mode.scheduledSounds[1])
	}
	if sound := mode.scheduledSounds[1]; !sound.positioned || sound.actorID != 300 {
		t.Fatalf("hit sound source = %+v, want target actor", sound)
	}
}

func TestPlayerBowNormalAttackUsesReferenceReleaseAndImpactTimeline(t *testing.T) {
	for _, tc := range []struct {
		name   string
		job    int16
		damage int32
	}{
		{name: "archer_hit", job: db.JobArcher, damage: 42},
		{name: "archer_miss", job: db.JobArcher},
		{name: "hunter_hit", job: db.JobHunter, damage: 42},
		{name: "hunter_miss", job: db.JobHunter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const (
				playerID = uint32(200)
				targetID = uint32(300)
			)
			player := worldstate.Actor{ID: playerID, X: 10, Y: 20, Job: tc.job, Weapon: db.WeaponBow, Dir: 4}
			world := worldstate.New()
			world.Player = player
			world.UpsertActor(worldstate.Actor{
				ID:            targetID,
				X:             15,
				Y:             20,
				Job:           1002,
				ObjectType:    actorObjectTypeMob,
				HasObjectType: true,
			})
			actionFamily := attackActionFamilyForActor(player)
			mode := &WorldMode{playerView: humanoidSoundView(actionFamily, 4, 2, []string{"atk"})}
			ctx := client.Context{
				Session: &session.Session{
					AccountID: playerID,
					CharID:    playerID,
					Selected:  session.Character{ID: playerID, Job: tc.job, Weapon: db.WeaponBow},
				},
				World: world,
			}

			mode.applyActorActionNotify(ctx, network.ActorActionNotify{
				SourceID:    playerID,
				TargetID:    targetID,
				SourceSpeed: 800,
				TargetSpeed: 480,
				Damage:      tc.damage,
				Action:      0,
			})

			sourceAnim, ok := mode.actorAnims[playerID]
			if !ok {
				t.Fatal("bow attack animation missing")
			}
			if len(mode.worldEffects) == 0 {
				t.Fatal("arrow projectile missing")
			}
			projectile := mode.worldEffects[0]
			if projectile.effectID != effectArrowShot || projectile.actorID != targetID || projectile.targetID != playerID {
				t.Fatalf("arrow projectile = %+v", projectile)
			}
			if delay := projectile.starts.Sub(sourceAnim.started); delay != 400*time.Millisecond {
				t.Fatalf("arrow release delay = %s, want ACT marker at 400ms", delay)
			}
			if projectile.duration != referenceBowArrowFlightDuration || projectile.expires.Sub(projectile.starts) != referenceBowArrowFlightDuration {
				t.Fatalf("arrow flight = duration %s lifetime %s, want %s", projectile.duration, projectile.expires.Sub(projectile.starts), referenceBowArrowFlightDuration)
			}

			if len(mode.scheduledSounds) == 0 {
				t.Fatal("bow release sound missing")
			}
			releaseSound := mode.scheduledSounds[0]
			if !releaseSound.at.Equal(projectile.starts) || releaseSound.actorID != playerID || !sameStringSet(releaseSound.paths, db.WeaponAttackSounds(db.WeaponBow)) {
				t.Fatalf("bow release sound = %+v projectile=%+v", releaseSound, projectile)
			}

			if tc.damage > 0 {
				if len(mode.worldEffects) != 2 {
					t.Fatalf("world effects = %+v, want arrow and impact", mode.worldEffects)
				}
				impact := mode.worldEffects[1]
				if impact.effectID != effectHit1 || impact.actorID != targetID || !impact.starts.Equal(projectile.expires) {
					t.Fatalf("impact = %+v projectile=%+v", impact, projectile)
				}
				targetAnim, ok := mode.actorAnims[targetID]
				if !ok || !targetAnim.started.Equal(impact.starts) {
					t.Fatalf("target hurt animation = %+v ok=%t impact=%+v", targetAnim, ok, impact)
				}
				if len(mode.scheduledSounds) != 2 {
					t.Fatalf("scheduled sounds = %+v, want release and impact", mode.scheduledSounds)
				}
				impactSound := mode.scheduledSounds[1]
				if !impactSound.at.Equal(impact.starts) || impactSound.actorID != targetID || len(impactSound.paths) != 1 || impactSound.paths[0] != "_hit_arrow.wav" {
					t.Fatalf("arrow impact sound = %+v impact=%+v", impactSound, impact)
				}
				if len(mode.damageFloaters) != 1 || !mode.damageFloaters[0].starts.Equal(impact.starts) {
					t.Fatalf("damage floater = %+v impact=%+v", mode.damageFloaters, impact)
				}
				return
			}

			if len(mode.worldEffects) != 1 {
				t.Fatalf("miss effects = %+v, want arrow only", mode.worldEffects)
			}
			if len(mode.scheduledSounds) != 1 {
				t.Fatalf("miss sounds = %+v, want bow release only", mode.scheduledSounds)
			}
			if _, ok := mode.actorAnims[targetID]; ok {
				t.Fatalf("miss started target hurt animation: %+v", mode.actorAnims[targetID])
			}
			if len(mode.damageFloaters) != 1 || mode.damageFloaters[0].kind != damageFloaterMiss || !mode.damageFloaters[0].starts.Equal(projectile.expires) {
				t.Fatalf("miss floater = %+v projectile=%+v", mode.damageFloaters, projectile)
			}
		})
	}
}

func TestDoubleStrafeUsesReferenceBowReleaseAndMultiHitTimeline(t *testing.T) {
	for _, tc := range []struct {
		name   string
		damage int32
	}{
		{name: "hit", damage: 100},
		{name: "miss"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const (
				playerID = uint32(200)
				targetID = uint32(300)
			)
			player := worldstate.Actor{ID: playerID, X: 10, Y: 20, Job: db.JobHunter, Weapon: db.WeaponBow, Dir: 4}
			world := worldstate.New()
			world.Player = player
			world.UpsertActor(worldstate.Actor{
				ID:            targetID,
				X:             15,
				Y:             20,
				Job:           1002,
				ObjectType:    actorObjectTypeMob,
				HasObjectType: true,
			})
			actionFamily := skillAction(db.SkillACDouble).actionFamilyForActor(player)
			mode := &WorldMode{playerView: humanoidSoundView(actionFamily, 4, 2, []string{"atk"})}
			ctx := client.Context{
				Session: &session.Session{
					AccountID: playerID,
					CharID:    playerID,
					Selected:  session.Character{ID: playerID, Job: db.JobHunter, Weapon: db.WeaponBow},
				},
				World: world,
			}

			mode.applyActorActionNotify(ctx, network.ActorActionNotify{
				SourceID:    playerID,
				TargetID:    targetID,
				SourceSpeed: 800,
				TargetSpeed: 480,
				Damage:      tc.damage,
				HitCount:    2,
				Action:      8,
				SkillID:     db.SkillACDouble,
				SkillLevel:  10,
			})

			sourceAnim, ok := mode.actorAnims[playerID]
			if !ok {
				t.Fatal("Double Strafe animation missing")
			}
			if len(mode.worldEffects) < 2 || mode.worldEffects[0].effectID != effectBashBegin {
				t.Fatalf("Double Strafe effects = %+v, want begin effect first", mode.worldEffects)
			}
			projectile := mode.worldEffects[1]
			if projectile.effectID != effectArrowShot || projectile.actorID != targetID || projectile.targetID != playerID {
				t.Fatalf("Double Strafe projectile = %+v", projectile)
			}
			if delay := projectile.starts.Sub(sourceAnim.started); delay != 400*time.Millisecond {
				t.Fatalf("Double Strafe release delay = %s, want ACT marker at 400ms", delay)
			}
			if projectile.duration != referenceBowArrowFlightDuration || projectile.expires.Sub(projectile.starts) != referenceBowArrowFlightDuration {
				t.Fatalf("Double Strafe flight = duration %s lifetime %s, want %s", projectile.duration, projectile.expires.Sub(projectile.starts), referenceBowArrowFlightDuration)
			}

			var releaseSounds, impactSounds []scheduledSound
			for _, sound := range mode.scheduledSounds {
				switch {
				case sameStringSet(sound.paths, db.WeaponAttackSounds(db.WeaponBow)):
					releaseSounds = append(releaseSounds, sound)
				case sameStringSet(sound.paths, db.EnemyHitNormalSounds()):
					impactSounds = append(impactSounds, sound)
				}
			}
			if len(releaseSounds) != 1 || !releaseSounds[0].at.Equal(projectile.starts) || releaseSounds[0].actorID != playerID {
				t.Fatalf("Double Strafe release sounds = %+v projectile=%+v", releaseSounds, projectile)
			}

			if tc.damage > 0 {
				if len(mode.worldEffects) != 4 {
					t.Fatalf("Double Strafe effects = %+v, want begin, one arrow, and two impacts", mode.worldEffects)
				}
				for i, impact := range mode.worldEffects[2:] {
					wantStarts := projectile.expires.Add(multiHitDelay * time.Duration(i))
					if impact.effectID != effectBashHit || impact.actorID != targetID || !impact.starts.Equal(wantStarts) {
						t.Fatalf("Double Strafe impact %d = %+v, want start %s", i, impact, wantStarts)
					}
				}
				targetAnim, ok := mode.actorAnims[targetID]
				if !ok || !targetAnim.started.Equal(projectile.expires) {
					t.Fatalf("Double Strafe target animation = %+v ok=%t, want first impact at %s", targetAnim, ok, projectile.expires)
				}
				if len(impactSounds) != 2 {
					t.Fatalf("Double Strafe impact sounds = %+v, want two", impactSounds)
				}
				for i, sound := range impactSounds {
					wantStarts := projectile.expires.Add(multiHitDelay * time.Duration(i))
					if !sound.at.Equal(wantStarts) || sound.actorID != targetID {
						t.Fatalf("Double Strafe impact sound %d = %+v, want target=%d at=%s", i, sound, targetID, wantStarts)
					}
				}
				normalFloaters := make([]damageFloater, 0, 2)
				for _, floater := range mode.damageFloaters {
					if floater.kind == damageFloaterNormal {
						normalFloaters = append(normalFloaters, floater)
					}
				}
				if len(normalFloaters) != 2 {
					t.Fatalf("Double Strafe damage floaters = %+v, want two normal hits", mode.damageFloaters)
				}
				for i, floater := range normalFloaters {
					wantStarts := projectile.expires.Add(multiHitDelay * time.Duration(i))
					if !floater.starts.Equal(wantStarts) {
						t.Fatalf("Double Strafe damage floater %d = %+v, want start %s", i, floater, wantStarts)
					}
				}
				return
			}

			if len(mode.worldEffects) != 2 {
				t.Fatalf("Double Strafe miss effects = %+v, want begin and arrow only", mode.worldEffects)
			}
			if len(impactSounds) != 0 {
				t.Fatalf("Double Strafe miss impact sounds = %+v, want none", impactSounds)
			}
			if _, ok := mode.actorAnims[targetID]; ok {
				t.Fatalf("Double Strafe miss started target hurt animation: %+v", mode.actorAnims[targetID])
			}
			if len(mode.damageFloaters) != 1 || mode.damageFloaters[0].kind != damageFloaterMiss || !mode.damageFloaters[0].starts.Equal(projectile.expires) {
				t.Fatalf("Double Strafe miss floater = %+v projectile=%+v", mode.damageFloaters, projectile)
			}
		})
	}
}

func TestApplyActorActionNotifyUsesMobACTHitPhase(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 11, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             10,
		Y:             20,
		Dir:           4,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{
		nonPCViews: map[int]*spriteView{
			1002: {
				act: &res.ACT{
					Actions: []res.ACTAction{
						{},
						{},
						{Animations: []res.ACTAnimation{
							{Sound: -1},
							{Sound: -1},
							{Sound: 0},
							{Sound: -1},
						}},
					},
					Sounds: []string{"poring_attack.wav"},
				},
			},
		},
	}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Sex:       0,
			Selected:  session.Character{ID: 150000, Job: 0, Hair: 1, Weapon: 1201},
		},
		World: world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    300,
		TargetID:    2000000,
		SourceSpeed: 800,
		TargetSpeed: 480,
		Damage:      1,
		Action:      0,
	})

	sourceAnim, ok := mode.actorAnims[300]
	if !ok {
		t.Fatal("source animation missing")
	}
	targetAnim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("local target animation missing")
	}
	if got := targetAnim.started.Sub(sourceAnim.started); got != 400*time.Millisecond {
		t.Fatalf("hit delay = %s, want ACT sound phase", got)
	}
	if len(mode.scheduledSounds) != 2 {
		t.Fatalf("scheduled sounds = %+v, want attack and hit sounds", mode.scheduledSounds)
	}
	if !mode.scheduledSounds[0].at.Equal(targetAnim.started) || mode.scheduledSounds[0].paths[0] != "poring_attack.wav" {
		t.Fatalf("attack sound = %+v targetStarted=%s", mode.scheduledSounds[0], targetAnim.started)
	}
}

func TestApplyActorVanishDeathKeepsMobForDeathAnimation(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{
		nonPCViews: map[int]*spriteView{
			1002: {
				act: &res.ACT{
					Actions: []res.ACTAction{
						{},
						{},
						{},
						{},
						{Animations: []res.ACTAnimation{{Sound: 0}, {Sound: -1}}, DelayMS: 100},
					},
					Sounds: []string{"poring_die.wav"},
				},
			},
		},
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000}},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: 1})

	if _, ok := world.Actors[300]; !ok {
		t.Fatal("dead actor was removed immediately")
	}
	anim, ok := mode.actorAnims[300]
	if !ok {
		t.Fatal("death animation missing")
	}
	if anim.actionFamily != spriteActionNonPCDeath {
		t.Fatalf("death action = %d, want %d", anim.actionFamily, spriteActionNonPCDeath)
	}
	if anim.holdFinal {
		t.Fatal("non-player death animation should not hold forever")
	}
	if removeAt, ok := mode.actorDeaths[300]; !ok || !removeAt.After(anim.started) {
		t.Fatalf("death removal time = %s ok=%t", removeAt, ok)
	}
	if got := mode.actorDeaths[300].Sub(anim.started); got != nonPCDeathFadeDuration {
		t.Fatalf("death visible duration = %s, want %s", got, nonPCDeathFadeDuration)
	}
	mode.processNonPCMotionSound(ctx, world.Actors[300], anim.started)
	if len(mode.scheduledSounds) != 1 || mode.scheduledSounds[0].paths[0] != "poring_die.wav" {
		t.Fatalf("death sounds = %+v", mode.scheduledSounds)
	}

	mode.cleanupDeadActors(ctx, mode.actorDeaths[300].Add(time.Millisecond))
	if _, ok := world.Actors[300]; ok {
		t.Fatal("dead actor was not removed after death hold")
	}
}

func TestApplyActorVanishDeathFreezesMovingMobAtRenderedPosition(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             20,
		Y:             20,
		FromX:         10,
		FromY:         20,
		ToX:           20,
		ToY:           20,
		Moving:        true,
		MoveStarted:   now.Add(-50 * time.Second),
		MoveDuration:  100 * time.Second,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000}},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: 1})

	actor, ok := world.Actors[300]
	if !ok {
		t.Fatal("dead actor was removed immediately")
	}
	if actor.Moving {
		t.Fatal("dead actor should stop moving")
	}
	if actor.X != 15 || actor.Y != 20 {
		t.Fatalf("dead actor position = %d,%d, want rendered position 15,20 instead of destination 20,20", actor.X, actor.Y)
	}
	if actor.FromX != actor.X || actor.ToX != actor.X || actor.FromY != actor.Y || actor.ToY != actor.Y {
		t.Fatalf("dead actor movement endpoints = from %d,%d to %d,%d, want frozen at %d,%d", actor.FromX, actor.FromY, actor.ToX, actor.ToY, actor.X, actor.Y)
	}
}

func TestRemotePlayerDeathRemainsUntilResurrection(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1,
		Appearance:    true,
		ObjectType:    actorObjectTypePC,
		HasObjectType: true,
	}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actorVanishDeath})

	anim, ok := mode.actorAnims[300]
	if !ok || anim.actionFamily != spriteActionPCDeath || !anim.holdFinal {
		t.Fatalf("remote player death animation = %+v ok=%t", anim, ok)
	}
	if removeAt, ok := mode.actorDeaths[300]; !ok || !removeAt.IsZero() {
		t.Fatalf("remote player death deadline = %s ok=%t, want persistent zero deadline", removeAt, ok)
	}
	if alpha := mode.actorDeathAlpha(300, anim.started.Add(time.Hour)); alpha != 1 {
		t.Fatalf("remote player death alpha = %f, want 1", alpha)
	}
	mode.cleanupDeadActors(ctx, anim.started.Add(time.Hour))
	if _, ok := world.Actors[300]; !ok {
		t.Fatal("remote player body was removed by death cleanup")
	}

	mode.applyActorResurrection(ctx, network.ActorResurrection{ID: 300})
	if _, ok := mode.actorDeaths[300]; ok {
		t.Fatal("remote player remained marked dead after resurrection")
	}
	if _, ok := mode.actorAnims[300]; ok {
		t.Fatal("remote player death animation remained after resurrection")
	}
	if actor := world.Actors[300]; !actor.HasAIMotion || actor.AIMotion != aiMotionStand {
		t.Fatalf("resurrected player motion = %+v, want standing", actor)
	}
}

func TestApplyActorVanishLogoutAndTeleportAddTeleportEffect(t *testing.T) {
	for _, reason := range []uint8{actorVanishLogout, actorVanishTeleport} {
		t.Run(fmt.Sprintf("reason_%d", reason), func(t *testing.T) {
			now := time.Now()
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			world.Actors[300] = worldstate.Actor{
				ID:           300,
				X:            20,
				Y:            20,
				FromX:        10,
				FromY:        20,
				ToX:          20,
				ToY:          20,
				Moving:       true,
				MoveStarted:  now.Add(-50 * time.Second),
				MoveDuration: 100 * time.Second,
			}
			mode := &WorldMode{}
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   world,
			}

			mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: reason})

			if _, ok := world.Actors[300]; ok {
				t.Fatal("vanished actor was not removed")
			}
			if len(mode.worldEffects) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
			}
			if effect := mode.worldEffects[0]; effect.actorID != 0 || effect.effectID != effectTeleportation || effect.x != 15 || effect.y != 20 {
				t.Fatalf("effect = %+v, want pinned teleportation at rendered position 15,20", effect)
			}
		})
	}
}

func TestApplyActorVanishOutOfSightDoesNotAddTeleportEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actorVanishOutOfSight})

	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects = %+v, want none", mode.worldEffects)
	}
}

func TestApplyActorVanishOutOfSightFadesBeforeRemovingNPC(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             20,
		Y:             20,
		Job:           100,
		ObjectType:    actorObjectTypeNPC,
		HasObjectType: true,
	}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actorVanishOutOfSight})

	if _, ok := world.Actors[300]; !ok {
		t.Fatal("out-of-sight NPC was removed immediately")
	}
	fade, ok := mode.actorVanishes[300]
	if !ok {
		t.Fatal("out-of-sight fade missing")
	}
	if got := fade.removeAt.Sub(fade.started); got != actorVanishOutOfSightFadeDuration {
		t.Fatalf("fade duration = %s, want %s", got, actorVanishOutOfSightFadeDuration)
	}
	if got := mode.actorVanishAlpha(300, fade.started.Add(actorVanishOutOfSightFadeDuration/2)); math.Abs(got-0.5) > 0.001 {
		t.Fatalf("fade alpha halfway = %.3f, want 0.5", got)
	}

	mode.cleanupVanishedActors(ctx, fade.started.Add(actorVanishOutOfSightFadeDuration/2))
	if _, ok := world.Actors[300]; !ok {
		t.Fatal("out-of-sight NPC was removed before fade completed")
	}
	mode.cleanupVanishedActors(ctx, fade.removeAt.Add(time.Millisecond))
	if _, ok := world.Actors[300]; ok {
		t.Fatal("out-of-sight NPC remained after fade completed")
	}
	if _, ok := mode.actorVanishes[300]; ok {
		t.Fatal("out-of-sight fade state remained after cleanup")
	}
}

func TestMobLookChangeToPlayerJobDoesNotChangeDeathSpriteFamily(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1161,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
		Appearance:    true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	applyActorLookChange(ctx, network.ActorLookChange{ID: 300, Type: 0, Value: 0})
	if got := world.Actors[300].Job; got != 1161 {
		t.Fatalf("mob job after look change = %d, want 1161", got)
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: 1})
	anim, ok := mode.actorAnims[300]
	if !ok {
		t.Fatal("death animation missing")
	}
	if anim.actionFamily != spriteActionNonPCDeath {
		t.Fatalf("death action = %d, want %d", anim.actionFamily, spriteActionNonPCDeath)
	}
}

func TestPendingSkillTargetCancelWithEscape(t *testing.T) {
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{skill: session.Skill{ID: 6, Level: 2, Range: 9}},
	}
	inputState := input.NewState()
	inputState.SetKey(input.KeyEscape, true)

	if !mode.skills().CancelFromInput(client.Context{Input: inputState}) {
		t.Fatal("pending skill target was not canceled")
	}
	if mode.pendingSkill.skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want 0", mode.pendingSkill.skill.ID)
	}
}

func TestBasicMenuOptionTogglesEscapeMenu(t *testing.T) {
	mode := &WorldMode{}

	mode.basicMenuCallbacks(client.Context{}).OnOption()

	if !mode.ui.escapeMenu.IsOpen() {
		t.Fatal("escape menu did not open")
	}
	if mode.ui.escapeMenu.Action() != gameui.EscapeMenuActionNone {
		t.Fatalf("escape menu action = %d, want none", mode.ui.escapeMenu.Action())
	}
	if mode.ui.escapeMenu.Pending() {
		t.Fatal("escape menu kept stale pending state")
	}

	mode.basicMenuCallbacks(client.Context{}).OnOption()

	if mode.ui.escapeMenu.IsOpen() {
		t.Fatal("escape menu stayed open after second option click")
	}
}

func TestEscapeKeyOpensEscapeMenuGlobally(t *testing.T) {
	mode := &WorldMode{}
	inputState := input.NewState()
	inputState.SetKey(input.KeyEscape, true)
	manager := &worldModeTestUIManager{}

	if !mode.openEscapeMenuFromInput(client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}) {
		t.Fatal("escape key did not open escape menu")
	}
	if !mode.ui.escapeMenu.IsOpen() {
		t.Fatal("escape menu is not open")
	}
	if len(manager.overlays) != 1 {
		t.Fatalf("overlays = %d, want 1", len(manager.overlays))
	}
}

func TestDeathEscapeBypassesInactiveChatConsole(t *testing.T) {
	mode := &WorldMode{}
	inputState := input.NewState()
	manager := &worldModeTestUIManager{}
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}
	mode.ui.console.UpdatePresentation(ctx)
	mode.ui.escapeMenu.OpenDeath(ctx)

	inputState.SetKey(input.KeyEscape, true)
	if !mode.updateDeathUIInput(ctx) {
		t.Fatal("death UI did not consume Escape")
	}
	if mode.ui.escapeMenu.IsOpen() {
		t.Fatal("inactive chat console consumed Escape before the death menu")
	}
	if !mode.ui.escapeMenu.DeathMode() {
		t.Fatal("hiding the death menu ended death mode")
	}
}

func TestActiveChatGetsFirstDeathEscape(t *testing.T) {
	mode := &WorldMode{}
	inputState := input.NewState()
	manager := &worldModeTestUIManager{}
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}
	mode.ui.console.UpdatePresentation(ctx)
	mode.ui.escapeMenu.OpenDeath(ctx)

	inputState.SetKey(input.KeyEnter, true)
	if !mode.updateDeathUIInput(ctx) || !mode.ui.console.Active() {
		t.Fatal("Enter did not activate chat while dead")
	}
	inputState.EndFrame()
	inputState.SetKey(input.KeyEnter, false)
	inputState.EndFrame()
	inputState.SetKey(input.KeyEscape, true)

	if !mode.updateDeathUIInput(ctx) {
		t.Fatal("active chat did not consume Escape")
	}
	if mode.ui.console.Active() {
		t.Fatal("Escape did not leave chat input")
	}
	if !mode.ui.escapeMenu.IsOpen() {
		t.Fatal("chat Escape also hid the death menu")
	}
}

func TestPendingSkillTargetCancelWithRightClick(t *testing.T) {
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{skill: session.Skill{ID: 6, Level: 2, Range: 9}},
	}
	inputState := input.NewState()
	inputState.SetMouseButton(input.MouseButtonRight, true)

	if !mode.skills().CancelFromInput(client.Context{Input: inputState}) {
		t.Fatal("pending skill target was not canceled")
	}
	if mode.pendingSkill.skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want 0", mode.pendingSkill.skill.ID)
	}
}

func TestPendingSkillTargetClickIgnoresWalkCooldown(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	lunatic := worldstate.Actor{
		ID:            300,
		X:             30,
		Y:             20,
		Job:           1063,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(lunatic)

	inputState := input.NewState()
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    session.Skill{ID: db.SkillMGSoulstrike, Level: 1, Type: skillTargetEnemy, Range: 9},
			maxLevel: 10,
		},
		walkCooldownUntil: time.Now().Add(time.Hour),
	}
	ctx := client.Context{
		Input:   inputState,
		Network: network.NewClient(20080910, false),
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	projection := mode.sceneProjection(ctx, ctx.ScreenW, ctx.ScreenH, time.Now())
	point := projection.Project(cellCenter(float64(lunatic.X)), cellCenter(float64(lunatic.Y)), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.pendingSkill.targetID != lunatic.ID {
		t.Fatalf("pending skill target = %d, want Lunatic %d despite walk cooldown", mode.pendingSkill.targetID, lunatic.ID)
	}
}

func TestPendingSkillWheelAdjustsLevelAndConsumesWheel(t *testing.T) {
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    session.Skill{ID: 19, Level: 10, Range: 9},
			maxLevel: 10,
		},
	}
	inputState := input.NewState()
	inputState.AddWheel(0, -2)

	if !mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("pending skill level was not adjusted")
	}
	if mode.pendingSkill.skill.Level != 8 {
		t.Fatalf("pending skill level = %d, want 8", mode.pendingSkill.skill.Level)
	}
	if inputState.WheelY != 0 {
		t.Fatalf("wheel was not consumed: %f", inputState.WheelY)
	}

	inputState.AddWheel(0, 20)
	if !mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("pending skill level cap was not handled")
	}
	if mode.pendingSkill.skill.Level != 10 {
		t.Fatalf("pending skill level = %d, want capped to 10", mode.pendingSkill.skill.Level)
	}
	if inputState.WheelY != 0 {
		t.Fatalf("wheel was not consumed at cap: %f", inputState.WheelY)
	}
}

func TestPendingSkillWheelDoesNotGoBelowLevelOne(t *testing.T) {
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    session.Skill{ID: 19, Level: 2, Range: 9},
			maxLevel: 10,
		},
	}
	inputState := input.NewState()
	inputState.AddWheel(0, -10)

	if !mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("pending skill level was not adjusted")
	}
	if mode.pendingSkill.skill.Level != 1 {
		t.Fatalf("pending skill level = %d, want capped to 1", mode.pendingSkill.skill.Level)
	}
}

func TestPendingSkillWheelIgnoresKnownFixedLevelSkill(t *testing.T) {
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    session.Skill{ID: db.SkillCRShieldcharge, Level: 5, Range: 3},
			maxLevel: 5,
		},
	}
	inputState := input.NewState()
	inputState.AddWheel(0, -1)

	if mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("fixed-level Shield Charge consumed the skill-level wheel")
	}
	if mode.pendingSkill.skill.Level != 5 || inputState.WheelY != -1 {
		t.Fatalf("fixed-level wheel changed state: skill=%+v wheel=%f", mode.pendingSkill.skill, inputState.WheelY)
	}
}

func TestPendingSkillWheelIgnoredWithoutPendingSkill(t *testing.T) {
	mode := &WorldMode{}
	inputState := input.NewState()
	inputState.AddWheel(0, -1)

	if mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("wheel was consumed without a pending skill")
	}
	if inputState.WheelY != -1 {
		t.Fatalf("wheel = %f, want unchanged", inputState.WheelY)
	}
}

func TestPendingTargetSkillCancelsWhenClickingGround(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(32, 32)
	inputState := input.NewState()
	projection := newSceneProjectionForTarget(1280, 720, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(12), cellCenter(20), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{skill: session.Skill{ID: 6, Level: 2, Type: skillTargetEnemy, Range: 9}},
	}
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.skills().HandleClick(ctx, projection, time.Now())

	if mode.pendingSkill.skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want canceled", mode.pendingSkill.skill.ID)
	}
}

func TestPendingGroundSkillDoesNotCancelWhenClickingGround(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(32, 32)
	inputState := input.NewState()
	projection := newSceneProjectionForTarget(1280, 720, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(12), cellCenter(20), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{skill: session.Skill{ID: 18, Level: 1, Type: skillTargetPlace, Range: 9}},
	}
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.skills().HandleClick(ctx, projection, time.Now())

	if mode.pendingSkill.skill.ID != 18 {
		t.Fatalf("pending ground skill id = %d, want still pending after send failure", mode.pendingSkill.skill.ID)
	}
}

func TestLocalDeathAnimationHoldsUntilPlayerAlive(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Selected:  session.Character{ID: 150000, Job: 0, HP: 0},
			Vitals:    session.Vitals{HP: 0},
		},
		World: world,
	}

	mode.startActorDeath(ctx, 150000)

	if !mode.ui.escapeMenu.IsOpen() || !mode.ui.escapeMenu.DeathMode() {
		t.Fatal("death escape menu should open for local death")
	}
	if !ctx.Session.Dead {
		t.Fatal("session should be marked dead")
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("character death animation missing")
	}
	if anim.actionFamily != spriteActionPCDeath {
		t.Fatalf("death action = %d, want %d", anim.actionFamily, spriteActionPCDeath)
	}
	if !anim.holdFinal {
		t.Fatal("local death animation should hold final frame")
	}
	accountAnim, ok := mode.actorAnims[2000000]
	if !ok || !accountAnim.holdFinal || accountAnim.actionFamily != spriteActionPCDeath {
		t.Fatalf("account death animation = %+v ok=%t", accountAnim, ok)
	}
	if held, ok := mode.actorAnimation(150000, anim.started.Add(anim.duration+time.Second)); !ok || held.actionFamily != spriteActionPCDeath {
		t.Fatalf("expired local death animation = %+v ok=%t", held, ok)
	}

	ctx.Session.Vitals.HP = 1
	mode.clearLocalDeathStateIfAlive(ctx)

	if mode.ui.escapeMenu.IsOpen() || mode.ui.escapeMenu.DeathMode() {
		t.Fatal("death escape menu should clear when player is alive")
	}
	if ctx.Session.Dead {
		t.Fatal("session should be marked alive")
	}
	if _, ok := mode.actorAnims[150000]; ok {
		t.Fatal("character death animation should clear when player is alive")
	}
	if _, ok := mode.actorAnims[2000000]; ok {
		t.Fatal("account death animation should clear when player is alive")
	}
}

func TestSavePointRespawnHoldsDeathAnimationUntilMapChange(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := ln.Accept()
		accepted <- conn
	}()

	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	addr := ln.Addr().(*net.TCPAddr)
	if err := netClient.Connect(context.Background(), addr.IP.String(), addr.Port); err != nil {
		t.Fatal(err)
	}
	serverConn := <-accepted
	if serverConn == nil {
		t.Fatal("server did not accept test client")
	}
	defer serverConn.Close()

	world := worldstate.New()
	world.MapName = "prontera"
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	mode := &WorldMode{}
	ctx := client.Context{
		Network: netClient,
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Selected:  session.Character{ID: 150000, Job: 0, HP: 0},
			Vitals:    session.Vitals{HP: 0},
		},
		World: world,
	}

	mode.startActorDeath(ctx, 150000)
	mode.ui.escapeMenu.ReturnToSavePoint(ctx)
	if got := mode.ui.escapeMenu.PendingAction(); got != gameui.EscapeMenuActionSavePoint {
		t.Fatalf("pending death action = %d, want save point", got)
	}

	ctx.Session.Vitals.HP = 1
	ctx.Session.Selected.HP = 1
	mode.clearLocalDeathStateIfAlive(ctx)
	if _, ok := mode.actorAnims[150000]; !ok {
		t.Fatal("positive HP cleared death animation before respawn map change")
	}
	if !mode.ui.escapeMenu.IsOpen() || !mode.ui.escapeMenu.DeathMode() {
		t.Fatal("positive HP closed death menu before respawn map change")
	}

	next := mode.handleMapChange(ctx, network.MapChange{MapName: "geffen", X: 120, Y: 80})
	if next == nil {
		t.Fatal("respawn map change did not create the destination world mode")
	}
	if _, ok := mode.actorAnims[150000]; ok {
		t.Fatal("death animation remained after respawn map change")
	}
	if mode.ui.escapeMenu.IsOpen() || mode.ui.escapeMenu.DeathMode() {
		t.Fatal("death menu remained open after respawn map change")
	}
	if ctx.Session.Dead {
		t.Fatal("session remained dead after respawn map change")
	}
}

func TestActorDeathAlphaFadesOverVisibleDuration(t *testing.T) {
	started := time.Unix(10, 0)
	mode := &WorldMode{
		actorAnims: map[uint32]actorAnimation{
			300: {actionFamily: spriteActionNonPCDeath, started: started, duration: nonPCDeathFadeDuration},
		},
		actorDeaths: map[uint32]time.Time{
			300: started.Add(nonPCDeathFadeDuration),
		},
	}
	if got := mode.actorDeathAlpha(300, started); got != 1 {
		t.Fatalf("alpha at start = %.2f, want 1", got)
	}
	if got := mode.actorDeathAlpha(300, started.Add(nonPCDeathFadeDuration/2)); math.Abs(got-0.5) > 0.001 {
		t.Fatalf("alpha halfway = %.2f, want 0.5", got)
	}
	if got := mode.actorDeathAlpha(300, started.Add(nonPCDeathFadeDuration)); got != 0 {
		t.Fatalf("alpha at end = %.2f, want 0", got)
	}
}

func TestProcessNonPCMotionSoundSchedulesIdleACTSound(t *testing.T) {
	now := time.Unix(10, 0)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	actor := worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(actor)
	mode := &WorldMode{
		nonPCViews: map[int]*spriteView{
			1002: {
				started: now,
				act: &res.ACT{
					Actions: []res.ACTAction{
						{Animations: []res.ACTAnimation{{Sound: 0}, {Sound: -1}}, DelayMS: 100},
					},
					Sounds: []string{"poring_idle.wav"},
				},
			},
		},
	}
	ctx := client.Context{World: world}

	mode.processNonPCMotionSound(ctx, actor, now)
	mode.processNonPCMotionSound(ctx, actor, now)

	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v, want one idle sound", mode.scheduledSounds)
	}
	if mode.scheduledSounds[0].paths[0] != "poring_idle.wav" {
		t.Fatalf("idle sound = %+v", mode.scheduledSounds[0])
	}
	if sound := mode.scheduledSounds[0]; !sound.positioned || sound.actorID != 300 {
		t.Fatalf("idle sound source = %+v, want actor position", sound)
	}
}

func TestSpatialSoundGainUsesDHXJAndClassicRODistanceCurve(t *testing.T) {
	for _, tc := range []struct {
		name string
		dist float64
		want float64
	}{
		{name: "same cell", dist: 0, want: 1},
		{name: "inside min", dist: 3, want: 1},
		{name: "at min", dist: 4, want: 1},
		{name: "inverse distance", dist: 8, want: 0.5},
		{name: "at max", dist: 25, want: 4.0 / 25.0},
		{name: "beyond max clamps", dist: 50, want: 4.0 / 25.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := referenceSpatialSoundGain(0, 0, tc.dist, 0)
			if math.Abs(got-tc.want) > 0.0001 {
				t.Fatalf("gain = %.4f, want %.4f", got, tc.want)
			}
		})
	}
}

func TestProcessMapSoundsSchedulesNearbyRSWSound(t *testing.T) {
	now := time.Unix(20, 0)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 110, Y: 220}
	world.GND = &res.GND{Width: 100, Height: 200}
	world.RSW = &res.RSW{Sounds: []res.RSWSound{
		{
			File:     "water.wav",
			Position: res.RSWVector3{X: 10, Z: 20},
			Volume:   0.7,
			Range:    5,
			Cycle:    2,
		},
		{
			File:     "far.wav",
			Position: res.RSWVector3{X: 40, Z: 20},
			Volume:   1,
			Range:    5,
			Cycle:    2,
		},
	}}
	mode := &WorldMode{}
	ctx := client.Context{World: world}

	mode.processMapSounds(ctx, now)
	mode.processMapSounds(ctx, now.Add(time.Second))

	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v, want one nearby map sound", mode.scheduledSounds)
	}
	sound := mode.scheduledSounds[0]
	if sound.paths[0] != "water.wav" || math.Abs(sound.volume-0.7) > 0.0001 {
		t.Fatalf("scheduled map sound = %+v", sound)
	}
	if !sound.positioned || sound.actorID != 0 || sound.x != 110 || sound.y != 220 {
		t.Fatalf("scheduled map sound source = %+v, want fixed world position", sound)
	}
	if got := mode.mapSoundNext[0]; !got.Equal(now.Add(2 * time.Second)) {
		t.Fatalf("next map sound time = %s, want %s", got, now.Add(2*time.Second))
	}
	if _, ok := mode.mapSoundNext[1]; ok {
		t.Fatalf("far sound should not have a timer: %+v", mode.mapSoundNext)
	}
}

func TestProcessMapSoundsUsesMinimumReplayDelayForZeroCycle(t *testing.T) {
	now := time.Unix(20, 0)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GND = &res.GND{Width: 10, Height: 20}
	world.RSW = &res.RSW{Sounds: []res.RSWSound{
		{
			File:   "loop.wav",
			Volume: 1,
			Range:  5,
		},
	}}
	mode := &WorldMode{}
	ctx := client.Context{World: world}

	mode.processMapSounds(ctx, now)
	mode.processMapSounds(ctx, now.Add(50*time.Millisecond))
	mode.processMapSounds(ctx, now.Add(100*time.Millisecond))

	if len(mode.scheduledSounds) != 2 {
		t.Fatalf("scheduled sounds = %+v, want initial sound and replay after 100ms", mode.scheduledSounds)
	}
}
