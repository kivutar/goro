package game

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestApplyLocalActorLookChangeUpdatesSelectedCharacter(t *testing.T) {
	sessionState := &session.Session{
		AccountID: 100,
		CharID:    200,
		Selected:  session.Character{ID: 200, Job: 0, Hair: 1},
		Characters: []session.Character{
			{ID: 200, Job: 0, Hair: 1},
		},
	}
	ctx := client.Context{
		Session: sessionState,
		World:   worldstate.New(),
	}
	look := network.ActorLookChange{
		ID:    200,
		Type:  2,
		Value: uint32(2101)<<16 | 1201,
	}

	if !applyActorLookChange(ctx, look) {
		t.Fatal("local look change should request player sprite reload")
	}
	if sessionState.Selected.Weapon != 1201 || sessionState.Selected.Shield != 2101 {
		t.Fatalf("selected appearance = weapon %d shield %d", sessionState.Selected.Weapon, sessionState.Selected.Shield)
	}
	if sessionState.Characters[0].Weapon != 1201 || sessionState.Characters[0].Shield != 2101 {
		t.Fatalf("character appearance = weapon %d shield %d", sessionState.Characters[0].Weapon, sessionState.Characters[0].Shield)
	}
	if ctx.World.Player.Weapon != 1201 || ctx.World.Player.Shield != 2101 {
		t.Fatalf("world player appearance = weapon %d shield %d", ctx.World.Player.Weapon, ctx.World.Player.Shield)
	}
}

func TestApplyStatusEffectChangeTracksLocalStatus(t *testing.T) {
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID:    10,
		ActorID:     2000000,
		Active:      true,
		HasDuration: true,
		Duration:    30 * time.Second,
	})
	effect, ok := sessionState.Statuses.Active[10]
	if !ok {
		t.Fatal("status was not tracked")
	}
	if !effect.HasDuration || effect.ExpiresAt.IsZero() || effect.Source != 2000000 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: 10,
		ActorID:  2000000,
		Active:   false,
	})
	if _, ok := sessionState.Statuses.Active[10]; ok {
		t.Fatal("inactive status was not removed")
	}
}

func TestHidingUsesActorOptionsInsteadOfStatusIcons(t *testing.T) {
	ctx := client.Context{
		World:   worldstate.New(),
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
	}
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusHiding, ActorID: 2000000, Active: true, HasDuration: true, Duration: time.Second})
	if ctx.PlayerHasEffectState(db.EffectStateHide) || len(mode.worldEffects) != 0 {
		t.Fatal("status icon changed actor visibility or duplicated its transition")
	}
	change := network.ActorStateChange{ID: 2000000, EffectState: db.EffectStateHide}
	mode.applyActorStateChange(ctx, change)
	mode.applyActorStateChange(ctx, change)
	removeExpiredStatusEffects(ctx.Session, time.Now().Add(time.Hour))
	if !ctx.PlayerHasEffectState(db.EffectStateHide) {
		t.Fatal("status icon expiry revealed the player")
	}
	if len(mode.worldEffects) != 2 || mode.worldEffects[0].effectID != effectSummonSlave || mode.worldEffects[1].effectID != effectBashBegin {
		t.Fatalf("hide effects = %+v, want one transition", mode.worldEffects)
	}
	change.EffectState = 0
	mode.applyActorStateChange(ctx, change)
	if ctx.PlayerHasEffectState(db.EffectStateHide) {
		t.Fatal("actor remained hidden after option cleared")
	}
}

func TestApplyStatusEffectChangeIgnoresRemoteActor(t *testing.T) {
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: 12,
		ActorID:  110000000,
		Active:   true,
	})
	if len(sessionState.Statuses.Active) != 0 {
		t.Fatalf("remote status changed local list: %+v", sessionState.Statuses.Active)
	}
}

func TestApplyFalconStatusUpdatesLocalActorAndSelectedOption(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:          150000,
		Job:         db.JobHunter,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: db.JobHunter},
		Characters: []session.Character{
			{ID: 150000, Job: db.JobHunter},
		},
	}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusFalcon,
		ActorID:  2000000,
		Active:   true,
	})
	if world.Player.EffectState&db.EffectStateFalcon == 0 {
		t.Fatalf("world player effect state = 0x%08X, want falcon bit", world.Player.EffectState)
	}
	if world.Player.EffectState&db.EffectStateRuwach == 0 {
		t.Fatalf("world player effect state = 0x%08X, want existing state preserved", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateFalcon == 0 {
		t.Fatalf("selected option = 0x%08X, want falcon bit", sessionState.Selected.Option)
	}
	if sessionState.Characters[0].Option&db.EffectStateFalcon == 0 {
		t.Fatalf("character option = 0x%08X, want falcon bit", sessionState.Characters[0].Option)
	}
	if _, ok := sessionState.Statuses.Active[db.StatusFalcon]; !ok {
		t.Fatal("falcon status should still be tracked for the status icon")
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusFalcon,
		ActorID:  2000000,
		Active:   false,
	})
	if world.Player.EffectState&db.EffectStateFalcon != 0 {
		t.Fatalf("world player effect state = 0x%08X, want falcon bit cleared", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateFalcon != 0 {
		t.Fatalf("selected option = 0x%08X, want falcon bit cleared", sessionState.Selected.Option)
	}
	if _, ok := sessionState.Statuses.Active[db.StatusFalcon]; ok {
		t.Fatal("inactive falcon status should be removed from status tracking")
	}
}

func TestApplyFalconStatusUpdatesRemoteActor(t *testing.T) {
	world := worldstate.New()
	world.Actors[110000001] = worldstate.Actor{ID: 110000001, Job: db.JobHunter, HasState: true}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusFalcon,
		ActorID:  110000001,
		Active:   true,
	})
	actor := world.Actors[110000001]
	if actor.EffectState&db.EffectStateFalcon == 0 {
		t.Fatalf("remote actor effect state = 0x%08X, want falcon bit", actor.EffectState)
	}
	if len(ctx.Session.Statuses.Active) != 0 {
		t.Fatalf("remote falcon status changed local list: %+v", ctx.Session.Statuses.Active)
	}
}

func TestApplyTrickDeadStatusHoldsDeathPose(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: 0, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusTrickdead,
		ActorID:  2000000,
		Active:   true,
	})
	anim, ok := mode.actorAnimation(150000, time.Now())
	if !ok || anim.actionFamily != spriteActionPCDeath || !anim.holdFinal {
		t.Fatalf("trick dead animation = %+v ok=%t, want held death pose", anim, ok)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusTrickdead,
		ActorID:  2000000,
		Active:   false,
	})
	anim, ok = mode.actorAnimation(150000, time.Now())
	if !ok || anim.actionFamily != spriteActionIdle || anim.holdFinal {
		t.Fatalf("trick dead inactive animation = %+v ok=%t, want idle", anim, ok)
	}
}

func TestTrickDeadSkillDoesNotStartDefaultSkillAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: 0, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	if action := skillAction(db.SkillNVTrickdead); !action.defined || action.action != skillActorActionNone {
		t.Fatalf("trick dead skill action = %+v, want no source action", action)
	}
	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SkillID:  db.SkillNVTrickdead,
		SourceID: 2000000,
		TargetID: 2000000,
		Result:   1,
	})
	if len(mode.actorAnims) != 0 {
		t.Fatalf("trick dead skill animation = %+v, want none before status", mode.actorAnims)
	}
}

func TestBackSlideSkillDoesNotStartDefaultSkillAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	if action := skillAction(db.SkillTFBacksliding); !action.defined || action.action != skillActorActionNone {
		t.Fatalf("back slide skill action = %+v, want no source action", action)
	}
	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SkillID:  db.SkillTFBacksliding,
		SourceID: 2000000,
		TargetID: 2000000,
		Result:   1,
	})
	if len(mode.actorAnims) != 0 {
		t.Fatalf("back slide skill animation = %+v, want none before jump packet", mode.actorAnims)
	}
}

func TestApplyActorJumpPositionMovesLocalPlayer(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4, Moving: true}
	sessionState := &session.Session{AccountID: 2000000, PlayerX: 10, PlayerY: 20}
	ctx := client.Context{Session: sessionState, World: world}

	applyActorJumpPosition(ctx, network.ActorJumpPosition{ID: 2000000, X: 7, Y: 20})

	if world.Player.X != 7 || world.Player.Y != 20 || world.Player.Moving {
		t.Fatalf("player after jump = %+v, want stopped at 7,20", world.Player)
	}
	if sessionState.PlayerX != 7 || sessionState.PlayerY != 20 {
		t.Fatalf("session position = %d,%d, want 7,20", sessionState.PlayerX, sessionState.PlayerY)
	}
}

func TestApplyMapAcceptEnterMarksAdminPlayer(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000, AdminList: []uint32{2000000}}
	ctx := client.Context{Session: sessionState, World: world}

	applyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 10, Y: 20, Dir: 4})

	if !world.Player.IsAdmin {
		t.Fatalf("world player admin state = false")
	}
}

func TestApplyMapAcceptEnterResetsWorldForChangedSelectedCharacter(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           150000,
		Job:          db.JobAlchemist,
		HasCartState: true,
		HasCart:      true,
		CartNum:      4,
		EffectState:  db.EffectStateCart4,
		Opt3State:    db.Opt3Quicken,
		HasState:     true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "stale actor"}
	world.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150001,
		Sex:       1,
		Selected: session.Character{
			ID:     150001,
			Name:   "Wizard",
			Job:    db.JobWizard,
			Hair:   7,
			Weapon: 1601,
		},
		Progress: session.Progress{BaseLevel: 42},
	}
	ctx := client.Context{Session: sessionState, World: world}

	applyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 20, Y: 30, Dir: 3})

	if world.Player.ID != 150001 || world.Player.Name != "Wizard" || world.Player.Job != db.JobWizard || world.Player.Weapon != 1601 {
		t.Fatalf("world player identity/appearance = %+v", world.Player)
	}
	if world.Player.HasCart || world.Player.CartNum != 0 || world.Player.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("old cart state leaked into selected character: %+v", world.Player)
	}
	if world.Player.Opt3State != 0 {
		t.Fatalf("old opt3 state leaked into selected character: 0x%08X", world.Player.Opt3State)
	}
	if world.Player.X != 20 || world.Player.Y != 30 || world.Player.Dir != 3 || world.Dir != 3 {
		t.Fatalf("world player position = %+v world dir=%d", world.Player, world.Dir)
	}
	if len(world.Actors) != 0 || len(world.Items) != 0 {
		t.Fatalf("stale world state survived character switch actors=%+v items=%+v", world.Actors, world.Items)
	}
}

func TestApplyMapAcceptEnterKeepsWorldStateForSameCharacter(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           150000,
		Job:          db.JobAlchemist,
		HasCartState: true,
		HasCart:      true,
		CartNum:      4,
		EffectState:  db.EffectStateCart4,
		Opt3State:    db.Opt3Quicken,
		HasState:     true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "same map actor"}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Playing:   true,
		Selected:  session.Character{ID: 150000, Job: db.JobAlchemist},
	}
	ctx := client.Context{Session: sessionState, World: world}

	applyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 20, Y: 30, Dir: 3})

	if !world.Player.HasCartState || !world.Player.HasCart || world.Player.CartNum != 4 || world.Player.EffectState&db.EffectStateCart4 == 0 {
		t.Fatalf("same-character cart state was not preserved: %+v", world.Player)
	}
	if world.Player.Opt3State&db.Opt3Quicken == 0 {
		t.Fatalf("same-character opt3 state was not preserved: 0x%08X", world.Player.Opt3State)
	}
	if len(world.Actors) != 1 {
		t.Fatalf("same-character actors were cleared: %+v", world.Actors)
	}
}

func TestApplyMapAcceptEnterResetsWorldForCharacterSelectReentry(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           150000,
		Job:          db.JobAlchemist,
		HasCartState: true,
		HasCart:      true,
		CartNum:      4,
		EffectState:  db.EffectStateCart4,
		Opt3State:    db.Opt3Quicken,
		HasState:     true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "stale actor"}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Playing:   false,
		Selected:  session.Character{ID: 150000, Job: db.JobAlchemist},
	}
	ctx := client.Context{Session: sessionState, World: world}

	applyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 20, Y: 30, Dir: 3})

	if world.Player.HasCart || world.Player.CartNum != 0 || world.Player.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("cart state leaked through character-select re-entry: %+v", world.Player)
	}
	if world.Player.Opt3State != 0 {
		t.Fatalf("opt3 state leaked through character-select re-entry: 0x%08X", world.Player.Opt3State)
	}
	if len(world.Actors) != 0 {
		t.Fatalf("stale actors survived character-select re-entry: %+v", world.Actors)
	}
}

func TestApplyPushCartStatusTracksLocalAndRemoteActors(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: 5}
	world.UpsertActor(worldstate.Actor{ID: 110000001, X: 10, Y: 20, Job: 5, Appearance: true})
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID:  db.StatusOnPushCart,
		ActorID:   2000000,
		Active:    true,
		HasValues: true,
		Values:    [3]int32{4, 0, 0},
	})
	if !world.Player.HasCartState || !world.Player.HasCart || world.Player.CartNum != 4 {
		t.Fatalf("local cart state = %+v", world.Player)
	}
	if len(ctx.Session.Statuses.Active) != 0 {
		t.Fatalf("pushcart should not create a buff icon: %+v", ctx.Session.Statuses.Active)
	}
	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          2000000,
		BodyState:   0,
		HealthState: 0,
		EffectState: 0,
	})
	if !world.Player.HasCartState || world.Player.HasCart || world.Player.CartNum != 0 || world.Player.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("local cart state after actor state refresh = %+v", world.Player)
	}
	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          2000000,
		BodyState:   0,
		HealthState: 0,
		EffectState: db.EffectStateCart2,
	})
	if !world.Player.HasCartState || !world.Player.HasCart || world.Player.CartNum != 2 {
		t.Fatalf("local cart state after change-cart actor state = %+v", world.Player)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID:  db.StatusOnPushCart,
		ActorID:   110000001,
		Active:    true,
		HasValues: true,
		Values:    [3]int32{2, 0, 0},
	})
	remote := world.Actors[110000001]
	if !remote.HasCartState || !remote.HasCart || remote.CartNum != 2 {
		t.Fatalf("remote cart state = %+v", remote)
	}
	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          110000001,
		BodyState:   0,
		HealthState: 0,
		EffectState: 0,
	})
	remote = world.Actors[110000001]
	if !remote.HasCartState || remote.HasCart || remote.CartNum != 0 || remote.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("remote cart state after actor state refresh = %+v", remote)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusOnPushCart,
		ActorID:  110000001,
		Active:   false,
	})
	remote = world.Actors[110000001]
	if !remote.HasCartState || remote.HasCart || remote.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("inactive remote cart state = %+v", remote)
	}
}

func TestApplyActorStateChangeSyncsSelectedWeddingOption(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobWizard, HasState: true}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: db.JobWizard},
		Characters: []session.Character{
			{ID: 150000, Job: db.JobWizard},
		},
	}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          2000000,
		BodyState:   0,
		HealthState: 0,
		EffectState: db.EffectStateWedding,
	})
	if world.Player.EffectState&db.EffectStateWedding == 0 {
		t.Fatalf("world player effect state = 0x%08X, want wedding bit", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateWedding == 0 {
		t.Fatalf("selected option = 0x%08X, want wedding bit", sessionState.Selected.Option)
	}
	if sessionState.Characters[0].Option&db.EffectStateWedding == 0 {
		t.Fatalf("character option = 0x%08X, want wedding bit", sessionState.Characters[0].Option)
	}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          2000000,
		BodyState:   0,
		HealthState: 0,
		EffectState: 0,
	})
	if world.Player.EffectState&db.EffectStateWedding != 0 {
		t.Fatalf("world player effect state = 0x%08X, want wedding bit cleared", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateWedding != 0 {
		t.Fatalf("selected option = 0x%08X, want wedding bit cleared", sessionState.Selected.Option)
	}
	if sessionState.Characters[0].Option&db.EffectStateWedding != 0 {
		t.Fatalf("character option = 0x%08X, want wedding bit cleared", sessionState.Characters[0].Option)
	}
}

func TestActorCartStateFromEffectUsesReferenceCartNumbers(t *testing.T) {
	actor := worldstate.Actor{Job: 5, EffectState: db.EffectStateCart3}
	hasCart, cartNum := actorCartState(actor)
	if !hasCart || cartNum != 3 {
		t.Fatalf("cart from effect = %t, %d", hasCart, cartNum)
	}
	actor = worldstate.Actor{Job: 23, EffectState: db.EffectStateCart5}
	hasCart, cartNum = actorCartState(actor)
	if !hasCart || cartNum != 0 {
		t.Fatalf("super novice cart from effect = %t, %d", hasCart, cartNum)
	}
}

func TestActorHasFalconUsesReferenceEffectBitAndJobs(t *testing.T) {
	if !actorHasFalcon(worldstate.Actor{Job: db.JobHunter, EffectState: db.EffectStateFalcon}) {
		t.Fatal("hunter falcon bit should create a falcon")
	}
	if !actorHasFalcon(worldstate.Actor{Job: db.JobHunterH, EffectState: db.EffectStateFalcon}) {
		t.Fatal("sniper falcon bit should create a falcon")
	}
	if actorHasFalcon(worldstate.Actor{Job: db.JobKnight, EffectState: db.EffectStateFalcon}) {
		t.Fatal("non-hunter falcon bit should not create a falcon sprite")
	}
	if actorHasFalcon(worldstate.Actor{Job: db.JobHunter}) {
		t.Fatal("hunter without falcon bit should not create a falcon")
	}
}

func TestFalconStateFollowsAsIndependentEntity(t *testing.T) {
	mode := &WorldMode{}
	start := time.Unix(100, 0)
	actor := worldstate.Actor{
		ID:          150011,
		Job:         db.JobHunter,
		EffectState: db.EffectStateFalcon,
		X:           10,
		Y:           20,
		Dir:         4,
		Speed:       150,
	}

	falcon := mode.updateFalconState(actor, start)
	if falcon == nil {
		t.Fatal("falcon state was not created")
	}
	if falcon.x != 10 || falcon.y != 20 || falcon.moving {
		t.Fatalf("initial falcon = x %.2f y %.2f moving=%t, want owner cell and idle", falcon.x, falcon.y, falcon.moving)
	}

	actor.X = 13
	falcon = mode.updateFalconState(actor, start.Add(falconRetargetInterval+time.Millisecond))
	if falcon == nil || !falcon.moving {
		t.Fatalf("falcon = %+v, want independent follow movement", falcon)
	}
	if falcon.moveSpeedMS != 100 {
		t.Fatalf("falcon speed = %d, want owner speed - 50", falcon.moveSpeedMS)
	}
	if len(falcon.path) == 0 {
		t.Fatal("falcon follow path is empty")
	}
	end := falcon.path[len(falcon.path)-1]
	if end.X != 12 || end.Y != 20 {
		t.Fatalf("falcon follow endpoint = %d,%d, want to stop one cell from owner", end.X, end.Y)
	}

	falcon.advance(start.Add(falconRetargetInterval + 50*time.Millisecond))
	if falcon.x <= 10 || falcon.x >= 13 {
		t.Fatalf("falcon x = %.2f, want independent position between old position and owner", falcon.x)
	}
}

func TestFalconFollowUsesRobrowserRangeFloor(t *testing.T) {
	if got := falconFollowDistance(11.9, 20, 10, 20); got != 1 {
		t.Fatalf("falcon follow distance = %d, want floored euclidean distance", got)
	}
	path := falconFollowPath(10, 20, 12, 21, falconStopRange)
	end := path[len(path)-1]
	if end.X != 11 || end.Y != 21 {
		t.Fatalf("diagonal-ish falcon endpoint = %d,%d, want floor(distance) <= 1 stop", end.X, end.Y)
	}
}

func TestFalconSpriteStateUsesIdleGlideAction(t *testing.T) {
	falcon := &falconRenderState{direction: 3, moving: true}
	state := falcon.spriteState(45)
	if state.actionFamily != spriteActionIdle || !state.loopIdle {
		t.Fatalf("falcon sprite state = %+v, want looping idle glide action", state)
	}
	if state.actionFamily == spriteActionWalk {
		t.Fatal("falcon follow should not use character walk action")
	}

	falcon.attacking = true
	state = falcon.spriteState(45)
	if state.actionFamily != spriteActionWalk {
		t.Fatalf("falcon attack sprite state = %+v, want walk action like robr", state)
	}
}

func TestFalconNoDamageSkillAttackUsesRobrowserOvershoot(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150006, X: 10, Y: 20, Dir: 4}
	world.Actors[300] = worldstate.Actor{ID: 300, Job: 1002, X: 12, Y: 20}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150006,
			Selected: session.Character{
				ID:     150006,
				Job:    db.JobHunter,
				Option: db.EffectStateFalcon,
			},
		},
		World: world,
	}
	mode := &WorldMode{}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SkillID:  db.SkillHTBlitzbeat,
		SourceID: 2000000,
		TargetID: 300,
		Result:   1,
	})
	falcon := mode.falcons[150006]
	if falcon == nil || !falcon.attacking {
		t.Fatalf("falcon = %+v, want attacking state", falcon)
	}
	if falcon.moveSpeedMS != falconAttackMoveSpeedMS {
		t.Fatalf("falcon attack speed = %d, want %d", falcon.moveSpeedMS, falconAttackMoveSpeedMS)
	}
	end := falcon.path[len(falcon.path)-1]
	if end.X != 17 || end.Y != 20 {
		t.Fatalf("falcon attack endpoint = %d,%d, want robr overshoot past target", end.X, end.Y)
	}
}

func TestFalconActorActionSkillAttackForFalconAssault(t *testing.T) {
	world := worldstate.New()
	world.Actors[200] = worldstate.Actor{
		ID:          200,
		Job:         db.JobHunterH,
		X:           10,
		Y:           20,
		Dir:         4,
		EffectState: db.EffectStateFalcon,
		HasState:    true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Job: 1002, X: 8, Y: 19}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150006},
		World:   world,
	}
	mode := &WorldMode{}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:  db.SkillSNFalconassault,
		SourceID: 200,
		TargetID: 300,
	})
	falcon := mode.falcons[200]
	if falcon == nil || !falcon.attacking {
		t.Fatalf("falcon = %+v, want Falcon Assault attack state", falcon)
	}
	end := falcon.path[len(falcon.path)-1]
	if end.X != 3 || end.Y != 14 {
		t.Fatalf("falcon assault endpoint = %d,%d, want robr diagonal overshoot", end.X, end.Y)
	}
}

func TestFalconDetectingSkillCastAttacksGroundWithoutDelay(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150006, X: 10, Y: 20, Dir: 4}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150006,
			Selected: session.Character{
				ID:     150006,
				Job:    db.JobHunter,
				Option: db.EffectStateFalcon,
			},
		},
		World: world,
	}
	mode := &WorldMode{}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{
		SkillID:  db.SkillHTDetecting,
		SourceID: 2000000,
		X:        12,
		Y:        22,
	})
	falcon := mode.falcons[150006]
	if falcon == nil || !falcon.attacking {
		t.Fatalf("falcon = %+v, want Detecting attack even with zero cast delay", falcon)
	}
	end := falcon.path[len(falcon.path)-1]
	if end.X != 17 || end.Y != 27 {
		t.Fatalf("detecting endpoint = %d,%d, want robr ground overshoot", end.X, end.Y)
	}
}

func TestFalconAttackReturnsToFollowTarget(t *testing.T) {
	start := time.Unix(100, 0)
	state := &falconRenderState{
		x:           10,
		y:           20,
		direction:   4,
		moveSpeedMS: 100,
		hasTarget:   true,
		targetX:     10,
		targetY:     20,
	}
	state.startAttack(12, 20, start)
	mode := &WorldMode{falcons: map[uint32]*falconRenderState{150006: state}}
	actor := worldstate.Actor{
		ID:          150006,
		Job:         db.JobHunter,
		EffectState: db.EffectStateFalcon,
		X:           10,
		Y:           20,
		Speed:       150,
	}

	mode.updateFalconState(actor, start.Add(falconAttackReturnDelay+time.Millisecond))
	if state.attacking {
		t.Fatal("falcon should leave attacking state after robr return delay")
	}
	if state.moveSpeedMS != falconAttackMoveSpeedMS {
		t.Fatalf("return speed = %d, want attack speed retained like robr", state.moveSpeedMS)
	}
	end := state.path[len(state.path)-1]
	if end.X != 10 || end.Y != 20 {
		t.Fatalf("return endpoint = %d,%d, want last follow target", end.X, end.Y)
	}
}

func TestDrawSceneModelsAndActorsRunsFalconPass(t *testing.T) {
	world := worldstate.New()
	world.MapName = "hugel"
	world.Player = worldstate.Actor{ID: 150006, X: 209, Y: 220, Dir: 4}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150006,
			Selected: session.Character{
				ID:     150006,
				Job:    db.JobHunter,
				Option: db.EffectStateFalcon,
			},
		},
		Resources: &res.Manager{},
		World:     world,
	}
	mode := &WorldMode{
		falconViews: map[int]*spriteView{
			db.JobHunter: {},
		},
	}
	screen := render.NewFrame(800, 600)
	projection := newSceneProjectionForTarget(800, 600, cellCenter(209), cellCenter(220), 0)

	actors := mode.drawSceneModelsAndActors(screen, ctx, projection, sceneFog{}, time.Unix(100, 0))
	if len(actors) == 0 {
		t.Fatal("no actor entries collected")
	}
	if _, ok := mode.falcons[150006]; !ok {
		t.Fatal("mixed scene draw path did not run falcon pass")
	}
}

func TestCollectSceneActorEntriesUsesSelectedCharacterCartOption(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.Player = worldstate.Actor{ID: 150004, X: 10, Y: 20, Dir: 4}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150004,
			AdminList: []uint32{2000000},
			Selected: session.Character{
				ID:     150004,
				Job:    5,
				Option: db.EffectStateCart1,
			},
		},
		World: world,
	}
	mode := &WorldMode{}
	screen := render.NewFrame(800, 600)
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)

	entries := mode.collectSceneActorEntries(screen, ctx, projection)
	if len(entries) == 0 {
		t.Fatal("no scene actor entries collected")
	}
	if hasCart, cartNum := actorCartState(entries[0].actor); !hasCart || cartNum != 1 {
		t.Fatalf("local cart from selected character option = has %t num %d actor %+v", hasCart, cartNum, entries[0].actor)
	}
	if !entries[0].actor.IsAdmin || !world.Player.IsAdmin {
		t.Fatalf("local admin state not applied: entry=%t world=%t", entries[0].actor.IsAdmin, world.Player.IsAdmin)
	}
}

func TestCollectSceneActorEntriesMergesSelectedCharacterFalconOption(t *testing.T) {
	world := worldstate.New()
	world.MapName = "pay_arche"
	world.Player = worldstate.Actor{
		ID:          150011,
		X:           10,
		Y:           20,
		Dir:         4,
		HasState:    true,
		EffectState: db.EffectStateRuwach,
	}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150011,
			Selected: session.Character{
				ID:     150011,
				Job:    db.JobHunter,
				Option: db.EffectStateFalcon,
			},
		},
		World: world,
	}
	mode := &WorldMode{}
	screen := render.NewFrame(800, 600)
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)

	entries := mode.collectSceneActorEntries(screen, ctx, projection)
	if len(entries) == 0 {
		t.Fatal("no scene actor entries collected")
	}
	if entries[0].actor.EffectState&db.EffectStateFalcon == 0 {
		t.Fatalf("entry effect state = 0x%08X, want selected falcon option merged", entries[0].actor.EffectState)
	}
	if entries[0].actor.EffectState&db.EffectStateRuwach == 0 {
		t.Fatalf("entry effect state = 0x%08X, want live effect state preserved", entries[0].actor.EffectState)
	}
}

func TestCollectSceneActorEntriesMergesSelectedCharacterWeddingOption(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.Player = worldstate.Actor{
		ID:          150012,
		X:           10,
		Y:           20,
		Dir:         4,
		HasState:    true,
		EffectState: db.EffectStateRuwach,
	}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150012,
			Selected: session.Character{
				ID:     150012,
				Job:    db.JobWizard,
				Option: db.EffectStateWedding,
			},
		},
		World: world,
	}
	mode := &WorldMode{}
	screen := render.NewFrame(800, 600)
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)

	entries := mode.collectSceneActorEntries(screen, ctx, projection)
	if len(entries) == 0 {
		t.Fatal("no scene actor entries collected")
	}
	if entries[0].actor.EffectState&db.EffectStateWedding == 0 {
		t.Fatalf("entry effect state = 0x%08X, want selected wedding option merged", entries[0].actor.EffectState)
	}
	if entries[0].actor.EffectState&db.EffectStateRuwach == 0 {
		t.Fatalf("entry effect state = 0x%08X, want live effect state preserved", entries[0].actor.EffectState)
	}
	if entries[0].actor.Job != db.JobMarried {
		t.Fatalf("entry visual job = %d, want married job %d", entries[0].actor.Job, db.JobMarried)
	}
}

func TestCartOffsetBillboardAppliesReferencePixelOffset(t *testing.T) {
	base := &spriteBillboard{anchorX: 100, anchorY: 200}
	dx, dy := cartSpriteOffset(2)
	got := cartOffsetBillboard(base, dx, dy)
	if got == base {
		t.Fatal("cart offset should copy billboard")
	}
	if got.anchorX != 60 || got.anchorY != 200 {
		t.Fatalf("offset billboard anchor = %.0f, %.0f", got.anchorX, got.anchorY)
	}
	if base.anchorX != 100 || base.anchorY != 200 {
		t.Fatalf("base billboard mutated = %.0f, %.0f", base.anchorX, base.anchorY)
	}
}

func TestCartDirectionOffsetBillboardPlacesShadowAtCart(t *testing.T) {
	base := &spriteBillboard{anchorX: 100, anchorY: 200}
	got := cartDirectionOffsetBillboard(base, 5)
	if got == base {
		t.Fatal("cart shadow offset should copy billboard")
	}
	if got.anchorX != 130 || got.anchorY != 190 {
		t.Fatalf("cart shadow anchor = %.0f, %.0f, want 130, 190", got.anchorX, got.anchorY)
	}
	if base.anchorX != 100 || base.anchorY != 200 {
		t.Fatalf("base shadow billboard mutated = %.0f, %.0f", base.anchorX, base.anchorY)
	}
}

func TestDrawActorCartShadowUsesLoadedShadowSprite(t *testing.T) {
	shadowView := &spriteView{
		spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  16,
				Height: 8,
				Data:   solidRGBAFrame(16, 8),
			}},
		},
		act: &res.ACT{Actions: []res.ACTAction{{
			Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{
				Index:   0,
				SPRType: res.SPRFrameRGBA,
				ScaleX:  1,
				ScaleY:  1,
				Color:   [4]float32{1, 1, 1, 1},
			}}}},
		}}},
		images:     make(map[spriteFrameKey]*render.Image),
		billboards: make(map[singleSpriteBillboardKey]*spriteBillboard),
	}
	mode := &WorldMode{shadowView: shadowView}
	entry := sceneActorDrawEntry{
		actor: worldstate.Actor{
			ID:           300,
			Job:          db.JobMerchant,
			Dir:          2,
			HasCartState: true,
			HasCart:      true,
			CartNum:      1,
		},
		worldX:      cellCenter(10),
		worldY:      cellCenter(20),
		scale:       1,
		shadow:      1,
		castShadow:  true,
		shadowScale: 1,
	}
	projection := newSceneProjectionForTarget(800, 600, entry.worldX, entry.worldY, 0)
	if !mode.drawActorCartShadow3D(render.NewFrame(800, 600), client.Context{}, projection, entry, 0, time.Now()) {
		t.Fatal("cart shadow was not drawn")
	}
	entry.actor.HasCart = false
	if mode.drawActorCartShadow3D(render.NewFrame(800, 600), client.Context{}, projection, entry, 0, time.Now()) {
		t.Fatal("cart shadow drawn without a cart")
	}
}

func TestApplyActorStateChangeTracksRemoteActorRenderState(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 110000001, X: 10, Y: 20, Job: 1002, Speed: 400, Appearance: true})
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          110000001,
		BodyState:   db.BodyStateFreeze,
		HealthState: db.HealthStateBlind,
		EffectState: 0x00402000,
	})

	actor := world.Actors[110000001]
	if !actor.HasState || actor.BodyState != db.BodyStateFreeze || actor.HealthState != db.HealthStateBlind || actor.EffectState != 0x00402000 {
		t.Fatalf("actor state = %+v", actor)
	}
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectRuwach || mode.worldEffects[0].actorID != 110000001 {
		t.Fatalf("actor state effects = %+v, want Ruwach", mode.worldEffects)
	}
	state := mode.nonPCSpriteState(actor, time.Now())
	if state.actionFamily != spriteActionIdle || !state.hasPlay || state.play || !state.hasFixedMotion || state.fixedMotion != 0 {
		t.Fatalf("frozen non-pc sprite state = %+v", state)
	}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          110000001,
		BodyState:   db.BodyStateFreeze,
		HealthState: db.HealthStateBlind,
		EffectState: 0,
	})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("actor state effects after clear = %+v, want none", mode.worldEffects)
	}
}

func TestActorEntryWithEffectStateStartsStateEffect(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:          110000001,
		X:           10,
		Y:           20,
		Job:         1002,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	})
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectRuwach || mode.worldEffects[0].actorID != 110000001 {
		t.Fatalf("actor entry effects = %+v, want Ruwach", mode.worldEffects)
	}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:          110000001,
		X:           10,
		Y:           20,
		Job:         1002,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	})
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectRuwach {
		t.Fatalf("refreshed actor entry effects = %+v, want one Ruwach", mode.worldEffects)
	}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:          110000001,
		X:           10,
		Y:           20,
		Job:         1002,
		EffectState: 0,
		HasState:    true,
	})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("actor entry effects after clear = %+v, want none", mode.worldEffects)
	}
}

func TestSyncCurrentActorEffectStateEffectsStartsExistingActorEffects(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{
		ID:          110000001,
		X:           10,
		Y:           20,
		Job:         1002,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	})
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.syncCurrentActorEffectStateEffects(ctx)
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectRuwach || mode.worldEffects[0].actorID != 110000001 {
		t.Fatalf("synced actor effects = %+v, want Ruwach", mode.worldEffects)
	}
}

func TestActorVanishRemovesActorEffectStateEffects(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 110000001, X: 10, Y: 20, Job: 1002, Appearance: true})
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          110000001,
		EffectState: db.EffectStateRuwach,
	})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects before vanish = %+v, want Ruwach", mode.worldEffects)
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 110000001})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after vanish = %+v, want none", mode.worldEffects)
	}
}

func TestActorStateTintMatchesReferenceBodyAndHealthTints(t *testing.T) {
	tint := actorStateTint(worldstate.Actor{
		BodyState:   db.BodyStateFreeze,
		HealthState: db.HealthStateBlind,
		HasState:    true,
	})
	if tint.R != 0 || tint.G != 20 || tint.B != 40 || tint.A != 255 {
		t.Fatalf("tint = %+v, want frozen blue darkened by blind", tint)
	}
}

func TestEnergyCoatStatusSetsActorOpt3StateAndTint(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusEnergycoat,
		ActorID:  2000000,
		Active:   true,
	})

	if world.Player.Opt3State&db.Opt3Energycoat == 0 {
		t.Fatalf("opt3 state = 0x%08X, want energy coat bit", world.Player.Opt3State)
	}
	tint := actorStateTint(world.Player)
	if tint.R != 127 || tint.G != 127 || tint.B != 216 || tint.A != 255 {
		t.Fatalf("energy coat tint = %+v, want robr OPT3 tint", tint)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusEnergycoat,
		ActorID:  2000000,
		Active:   false,
	})
	if world.Player.Opt3State&db.Opt3Energycoat != 0 {
		t.Fatalf("opt3 state = 0x%08X, want energy coat cleared", world.Player.Opt3State)
	}
}

func TestTwoHandQuickenStatusUsesOpt3WithoutSightEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusTwohandquicken,
		ActorID:  2000000,
		Active:   true,
	})

	if world.Player.Opt3State&db.Opt3Quicken == 0 {
		t.Fatalf("opt3 state = 0x%08X, want quicken bit", world.Player.Opt3State)
	}
	if world.Player.EffectState&db.EffectStateSight != 0 {
		t.Fatalf("effect state = 0x%08X, want no Sight bit from quicken", world.Player.EffectState)
	}
	if tint := actorStateTint(world.Player); tint.B != 0 {
		t.Fatalf("quicken tint = %+v, want robr OPT3 quicken blue channel removed", tint)
	}
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects = %+v, want no EF_SIGHT from quicken status", mode.worldEffects)
	}
}

func TestActorEntryPreservesExistingOpt3State(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, X: 10, Y: 20, Job: 1, Opt3State: db.Opt3Quicken, HasState: true})
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:          300,
		X:           11,
		Y:           20,
		Job:         1,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	})

	actor := world.Actors[300]
	if actor.Opt3State&db.Opt3Quicken == 0 {
		t.Fatalf("actor opt3 state = 0x%08X, want quicken preserved", actor.Opt3State)
	}
	if actor.EffectState != db.EffectStateRuwach {
		t.Fatalf("actor effect state = 0x%08X, want packet effect state", actor.EffectState)
	}
}

func TestBerserkStatusSetsActorOpt3StateFromImportedTable(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusBerserk,
		ActorID:  2000000,
		Active:   true,
	})

	if world.Player.Opt3State&db.Opt3Berserk == 0 {
		t.Fatalf("opt3 state = 0x%08X, want berserk bit", world.Player.Opt3State)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusBerserk,
		ActorID:  2000000,
		Active:   false,
	})
	if world.Player.Opt3State&db.Opt3Berserk != 0 {
		t.Fatalf("opt3 state = 0x%08X, want berserk cleared", world.Player.Opt3State)
	}
}

func TestCollectSceneActorEntriesPreservesLocalOpt3State(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:        2000000,
		X:         10,
		Y:         20,
		Opt3State: db.Opt3Energycoat,
		HasState:  true,
	}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: 2, Option: db.EffectStateCart1},
	}
	screen := render.NewFrame(800, 600)
	ctx := client.Context{Session: sessionState, World: world}
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	mode := &WorldMode{}

	entries := mode.collectSceneActorEntries(screen, ctx, projection)
	if len(entries) == 0 {
		t.Fatal("no actor entries collected")
	}
	if entries[0].actor.Opt3State&db.Opt3Energycoat == 0 {
		t.Fatalf("entry opt3 state = 0x%08X, want energy coat preserved", entries[0].actor.Opt3State)
	}
	if entries[0].actor.EffectState&db.EffectStateCart1 == 0 {
		t.Fatalf("entry effect state = 0x%08X, want cart option merged", entries[0].actor.EffectState)
	}
}

func TestVisibleStatusIconIDsAreKnownAndSorted(t *testing.T) {
	active := map[uint16]session.StatusEffect{
		99:           {ID: 99},
		12:           {ID: 12},
		10:           {ID: 10},
		db.StatusSit: {ID: db.StatusSit},
	}
	ids := gameui.VisibleStatusIconIDs(active)
	if !reflect.DeepEqual(ids, []uint16{10, 12}) {
		t.Fatalf("ids = %+v", ids)
	}
}

func TestApplyRemoteActorLookChangeUpdatesWorldActor(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 0, Head: 1, Appearance: true})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	if applyActorLookChange(ctx, network.ActorLookChange{ID: 300, Type: 4, Value: 7}) {
		t.Fatal("remote look change should not request local player sprite reload")
	}
	actor := world.Actors[300]
	if actor.HeadTop != 7 {
		t.Fatalf("remote head top = %d, want 7", actor.HeadTop)
	}
}

func TestDirectionFromDeltaUsesRathenaDirectionOrder(t *testing.T) {
	cases := []struct {
		name string
		toX  int
		toY  int
		want int
	}{
		{name: "north", toX: 10, toY: 11, want: 0},
		{name: "northwest", toX: 9, toY: 11, want: 1},
		{name: "west", toX: 9, toY: 10, want: 2},
		{name: "southwest", toX: 9, toY: 9, want: 3},
		{name: "south", toX: 10, toY: 9, want: 4},
		{name: "southeast", toX: 11, toY: 9, want: 5},
		{name: "east", toX: 11, toY: 10, want: 6},
		{name: "northeast", toX: 11, toY: 11, want: 7},
		{name: "long mostly north", toX: 11, toY: 20, want: 0},
		{name: "long mostly west", toX: 0, toY: 11, want: 2},
		{name: "long mostly south", toX: 11, toY: 0, want: 4},
		{name: "long mostly east", toX: 20, toY: 11, want: 6},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := directionFromDelta(10, 10, tt.toX, tt.toY, 6); got != tt.want {
				t.Fatalf("directionFromDelta = %d, want %d", got, tt.want)
			}
		})
	}
	if got := directionFromDelta(10, 10, 10, 10, -1); got != 7 {
		t.Fatalf("stationary fallback = %d, want 7", got)
	}
}

func TestResolveTurnOnlyDirectionUsesHeadBeforeBody(t *testing.T) {
	head, body, ok := resolveTurnOnlyDirection(4, 0, 3)
	if !ok {
		t.Fatal("direction not resolved")
	}
	if head != 1 || body != 4 {
		t.Fatalf("one-octant left turn = head %d body %d, want head 1 body 4", head, body)
	}

	head, body, ok = resolveTurnOnlyDirection(4, 1, 3)
	if !ok {
		t.Fatal("direction not resolved")
	}
	if head != 0 || body != 3 {
		t.Fatalf("repeated left turn = head %d body %d, want head 0 body 3", head, body)
	}
}

func TestResolveTurnOnlyDirectionRotatesBodyForWideTurn(t *testing.T) {
	head, body, ok := resolveTurnOnlyDirection(4, 0, 1)
	if !ok {
		t.Fatal("direction not resolved")
	}
	if head != 1 || body != 2 {
		t.Fatalf("wide left turn = head %d body %d, want head 1 body 2", head, body)
	}

	head, body, ok = resolveTurnOnlyDirection(4, 0, 7)
	if !ok {
		t.Fatal("direction not resolved")
	}
	if head != 2 || body != 6 {
		t.Fatalf("wide right turn = head %d body %d, want head 2 body 6", head, body)
	}
}

func TestActorBillboardScreenScaleUsesProjectedReferenceHeight(t *testing.T) {
	if actorBillboardWorldHeightUnit != 5 {
		t.Fatalf("actor billboard world height = %.1f, want 5.0", actorBillboardWorldHeightUnit)
	}

	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 5)

	scale := actorBillboardScreenScale(projection, 10.5, 20.5, 5)
	if math.Abs(scale-1.04) > 0.01 {
		t.Fatalf("camera billboard scale = %.3f, want about 1.04 at reference client default zoom", scale)
	}
}

func TestActorBillboardScreenScaleDoesNotChangeWithCameraPitch(t *testing.T) {
	const (
		width  = 800
		height = 600
		x      = 10.5
		y      = 20.5
		z      = 5
	)
	want := actorBillboardScreenScale(newSceneProjectionForTargetYawPitchZoom(width, height, x, y, z, 0, sceneCameraPitch(), 125), x, y, z)
	for _, pitch := range []float64{defaultCameraMinPitch, 215, 235, defaultCameraMaxPitch} {
		projection := newSceneProjectionForTargetYawPitchZoom(width, height, x, y, z, 0, pitch, 125)
		if got := actorBillboardScreenScale(projection, x, y, z); math.Abs(got-want) > 0.000001 {
			t.Fatalf("actor scale at pitch %.1f = %.6f, want %.6f", pitch, got, want)
		}
	}
}

func TestActorBillboardScreenScaleFollowsZoomAndPerspectiveDepth(t *testing.T) {
	const (
		width  = 800
		height = 600
		x      = 10.5
		y      = 20.5
		z      = 5
	)
	zoomedIn := actorBillboardScreenScale(newSceneProjectionForTargetYawPitchZoom(width, height, x, y, z, 0, 235, defaultCameraMinZoom), x, y, z)
	zoomedOut := actorBillboardScreenScale(newSceneProjectionForTargetYawPitchZoom(width, height, x, y, z, 0, 235, defaultCameraMaxZoom), x, y, z)
	if zoomedIn <= zoomedOut {
		t.Fatalf("actor scale did not follow zoom: zoomed in %.6f, zoomed out %.6f", zoomedIn, zoomedOut)
	}

	nearer := actorBillboardScreenScale(newSceneProjectionForTargetYawPitchZoom(width, height, x, y, z, 0, 235, 125), x, y-5, z)
	farther := actorBillboardScreenScale(newSceneProjectionForTargetYawPitchZoom(width, height, x, y, z, 0, 235, 125), x, y+5, z)
	if nearer <= farther {
		t.Fatalf("actor scale did not follow perspective depth: nearer %.6f, farther %.6f", nearer, farther)
	}
}

func TestActorAnchorOutsideViewportKeepsBodyVisibleBelowScreen(t *testing.T) {
	if actorAnchorOutsideViewport(400, 600+150, 800, 600, 1) {
		t.Fatal("actor should remain visible while its body can still overlap the bottom edge")
	}
	if !actorAnchorOutsideViewport(400, 600+260, 800, 600, 1) {
		t.Fatal("actor should be culled after the whole billboard is beyond the bottom edge")
	}
}

func TestActorViewportCullMarginsScaleWithZoom(t *testing.T) {
	_, _, _, bottom1 := actorViewportCullMargins(1)
	_, _, _, bottom2 := actorViewportCullMargins(2)
	if bottom2 <= bottom1 {
		t.Fatalf("bottom margin did not scale: %.1f <= %.1f", bottom2, bottom1)
	}
	left, right, top, bottom := actorViewportCullMargins(0)
	if left <= 0 || right <= 0 || top <= 0 || bottom <= 0 {
		t.Fatalf("invalid fallback margins: %.1f %.1f %.1f %.1f", left, right, top, bottom)
	}
}

func TestClickedAttackTargetPicksMobOnly(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    6,
		HasObjectType: true,
	})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	npcPoint := projection.Project(cellCenter(11), cellCenter(20), 0)

	if actor, ok := clickedAttackTarget(ctx, projection, int(npcPoint.x), int(npcPoint.y), now, nil); ok {
		t.Fatalf("npc should not be attack-clickable: %+v", actor)
	}

	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             12,
		Y:             20,
		ObjectType:    5,
		HasObjectType: true,
	})
	mobPoint := projection.Project(cellCenter(12), cellCenter(20), 0)

	actor, ok := clickedAttackTarget(ctx, projection, int(mobPoint.x), int(mobPoint.y), now, nil)
	if !ok {
		t.Fatal("expected mob hit")
	}
	if actor.ID != 400 {
		t.Fatalf("target id = %d, want 400", actor.ID)
	}
}

func TestDefaultHoveredUIRootDoesNotHideWorldNPCCursor(t *testing.T) {
	ctx, projection := cursorHoverTestContext(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypeNPC,
		HasObjectType: true,
	})
	mode := &WorldMode{}

	if got := mode.cursorDesiredAction(ctx, projection, time.Now()); got != cursorActionTalk {
		t.Fatalf("cursor action = %d, want talk", got)
	}
}

func TestDefaultHoveredUIRootDoesNotHideWorldWarpCursor(t *testing.T) {
	ctx, projection := cursorHoverTestContext(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           actorJobWarpPortal,
		ObjectType:    actorObjectTypeNPC,
		HasObjectType: true,
	})
	mode := &WorldMode{}

	if got := mode.cursorDesiredAction(ctx, projection, time.Now()); got != cursorActionWarp {
		t.Fatalf("cursor action = %d, want warp", got)
	}
}

func TestCursorActionDefaultOverPC(t *testing.T) {
	ctx, projection := cursorHoverTestContext(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypePC,
		HasObjectType: true,
	})
	mode := &WorldMode{}

	if got := mode.cursorDesiredAction(ctx, projection, time.Now()); got != cursorActionDefault {
		t.Fatalf("cursor action = %d, want default", got)
	}
}

func TestCursorActionClickOverVendingBoard(t *testing.T) {
	now := time.Now()
	actor := worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypePC,
		HasObjectType: true,
		Vending:       true,
		VendingName:   "Fresh Fish",
	}
	ctx, projection := cursorHoverTestContext(actor)
	bounds, ok := vendingBoardActorBounds(ctx, projection, actor, now)
	if !ok {
		t.Fatal("expected vending board bounds")
	}
	ctx.Input.SetMousePosition(int(bounds.x+bounds.w/2), int(bounds.y+bounds.h/2))
	mode := &WorldMode{}

	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionClick {
		t.Fatalf("cursor action = %d, want click", got)
	}
}

func TestCursorMagnetOffsetFollowsTargetSnapSetting(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(11), cellCenter(20), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.x)+7, int(point.y)+3)
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	mode := &WorldMode{}

	if dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionAttack, now); dx != 0 || dy != 0 {
		t.Fatalf("disabled target snap offset = %.1f,%.1f, want zero", dx, dy)
	}
	ctx.Session.SnapTargets = true
	scale := actorBillboardScreenScale(projection, cellCenter(11), cellCenter(20), 0)
	targetX, targetY := actorPickBoundsCenter(float64(point.x), float64(point.y), scale)
	inputState.SetMousePosition(int(math.Round(targetX))+7, int(math.Round(targetY))+3)
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionAttack, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("target snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionAttack {
		t.Fatalf("target snap cursor action = %d, want attack inside snap distance", got)
	}
	inputState.SetMousePosition(int(math.Round(targetX)), int(math.Round(targetY+actorCursorSnapRadius(scale)*1.05)))
	if dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionAttack, now); dx != 0 || dy != 0 {
		t.Fatalf("outer target snap offset = %.1f,%.1f, want zero inside pick bounds but outside snap distance", dx, dy)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionDefault {
		t.Fatalf("outer target snap cursor action = %d, want default inside pick bounds but outside snap distance", got)
	}
}

func TestCursorMagnetOffsetSnapsTargetSkillToSelf(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:            200,
		X:             10,
		Y:             20,
		ObjectType:    actorObjectTypePC,
		HasObjectType: true,
	}
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(10), cellCenter(20), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.x)+7, int(point.y)+3)
	ctx := client.Context{
		Input: inputState,
		Session: &session.Session{
			AccountID:   100,
			CharID:      200,
			SnapTargets: true,
		},
		World: world,
	}
	mode := &WorldMode{pendingSkill: pendingSkillTarget{skill: session.Skill{
		ID:    db.SkillAMPotionpitcher,
		Type:  skillTargetFriend,
		Range: 9,
	}}}

	scale := actorBillboardScreenScale(projection, cellCenter(10), cellCenter(20), 0)
	targetX, targetY := actorPickBoundsCenter(float64(point.x), float64(point.y), scale)
	inputState.SetMousePosition(int(math.Round(targetX))+7, int(math.Round(targetY))+3)
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionTarget2, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("self target skill snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
	}
}

func TestCursorMagnetOffsetSnapsTargetSkillToHomunculus(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypeHomunculus,
		HasObjectType: true,
	})
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(11), cellCenter(20), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.x)+7, int(point.y)+3)
	ctx := client.Context{
		Input: inputState,
		Session: &session.Session{
			AccountID:   100,
			CharID:      200,
			SnapTargets: true,
		},
		World: world,
	}
	mode := &WorldMode{pendingSkill: pendingSkillTarget{skill: session.Skill{
		ID:    db.SkillAMPotionpitcher,
		Type:  skillTargetFriend,
		Range: 9,
	}}}

	scale := actorBillboardScreenScale(projection, cellCenter(11), cellCenter(20), 0)
	targetX, targetY := actorPickBoundsCenter(float64(point.x), float64(point.y), scale)
	inputState.SetMousePosition(int(math.Round(targetX))+7, int(math.Round(targetY))+3)
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionTarget2, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("homunculus target skill snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionTarget2 {
		t.Fatalf("homunculus target skill cursor action = %d, want target2 inside snap distance", got)
	}
	inputState.SetMousePosition(int(math.Round(targetX)), int(math.Round(targetY+actorCursorSnapRadius(scale)*1.05)))
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionTarget {
		t.Fatalf("outer homunculus target skill cursor action = %d, want target inside pick bounds but outside snap distance", got)
	}
}

func TestCursorMagnetOffsetFollowsItemSnapSetting(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	item := worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20}
	world.UpsertItem(item)
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	x, y := floorItemWorldPosition(item)
	point := projection.Project(cellCenter(x), cellCenter(y), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.x)+5, int(point.y)+2)
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	mode := &WorldMode{}

	if dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionPick, now); dx != 0 || dy != 0 {
		t.Fatalf("disabled item snap offset = %.1f,%.1f, want zero", dx, dy)
	}
	ctx.Session.SnapItems = true
	scale := actorBillboardScreenScale(projection, cellCenter(x), cellCenter(y), 0) * 0.42
	targetX, targetY := groundItemPickBoundsCenter(float64(point.x), float64(point.y), scale)
	inputState.SetMousePosition(int(math.Round(targetX)), int(math.Round(targetY)))
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionPick, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("item snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionPick {
		t.Fatalf("item snap cursor action = %d, want pick inside snap distance", got)
	}
	inputState.SetMousePosition(int(math.Round(targetX+groundItemCursorSnapRadius(scale)*1.1)), int(math.Round(targetY)))
	if dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionPick, now); dx != 0 || dy != 0 {
		t.Fatalf("outer item snap offset = %.1f,%.1f, want zero inside pick bounds but outside snap distance", dx, dy)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionDefault {
		t.Fatalf("outer item snap cursor action = %d, want default inside pick bounds but outside snap distance", got)
	}
}

func TestSpriteBillboardScreenCenterUsesImageAndAnchor(t *testing.T) {
	billboard := &spriteBillboard{
		image:   render.NewImage(40, 20),
		anchorX: 10,
		anchorY: 30,
	}
	x, y, ok := spriteBillboardScreenCenter(billboard, screenPoint{x: 100, y: 200}, 2)
	if !ok {
		t.Fatal("expected billboard center")
	}
	if x != 120 || y != 160 {
		t.Fatalf("center = %.1f,%.1f, want 120,160", x, y)
	}
}
