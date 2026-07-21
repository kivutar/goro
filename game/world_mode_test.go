package game

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
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

type worldModeTestUIManager struct {
	overlays []widget.Widget
}

func (m *worldModeTestUIManager) AddOverlay(root widget.Widget) {
	m.overlays = append(m.overlays, root)
}

func (m *worldModeTestUIManager) RemoveOverlay(root widget.Widget) {
	for i, overlay := range m.overlays {
		if overlay == root {
			m.overlays = append(m.overlays[:i], m.overlays[i+1:]...)
			return
		}
	}
}

func (m *worldModeTestUIManager) Clear() {
	m.overlays = nil
}

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

func TestApplyHidingStatusTogglesLocalHiddenStateAndTransitionEffects(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusHiding,
		ActorID:  2000000,
		Active:   true,
	})
	if !localActorHidden(ctx) {
		t.Fatal("hiding status did not mark the local actor hidden")
	}
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectBashBegin {
		t.Fatalf("hide enter effects = %+v, want EF_BASH", mode.worldEffects)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusHiding,
		ActorID:  2000000,
		Active:   false,
	})
	if localActorHidden(ctx) {
		t.Fatal("inactive hiding status still marks the local actor hidden")
	}
	if len(mode.worldEffects) != 2 || mode.worldEffects[1].effectID != effectSummonSlave {
		t.Fatalf("hide exit effects = %+v, want EF_SUMMONSLAVE", mode.worldEffects)
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

func TestCollectSceneActorEntriesUsesSelectedCharacterCartOption(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.Player = worldstate.Actor{ID: 150004, X: 10, Y: 20, Dir: 4}
	ctx := client.Context{
		Session: &session.Session{
			CharID: 150004,
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
		99: {ID: 99},
		12: {ID: 12},
		10: {ID: 10},
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
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionAttack, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("target snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
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
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionPick, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("item snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
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

func cursorHoverTestContext(actor worldstate.Actor) (client.Context, sceneProjection) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(actor)
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(float64(actor.X)), cellCenter(float64(actor.Y)), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.x), int(point.y))
	return client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
		UIApp:   fakeCursorUIApp{hovered: primitives.Box()},
	}, projection
}

func TestClickedSkillTargetUsesRobrowserTargetFlags(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypePC,
		HasObjectType: true,
	})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	playerPoint := projection.Project(cellCenter(10), cellCenter(20), 0)
	pcPoint := projection.Project(cellCenter(11), cellCenter(20), 0)

	if actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 6, Type: skillTargetEnemy}, int(playerPoint.x), int(playerPoint.y), now, nil); ok {
		t.Fatalf("enemy skill should not self-target: %+v", actor)
	}

	actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 28, Type: skillTargetFriend}, int(playerPoint.x), int(playerPoint.y), now, nil)
	if !ok {
		t.Fatal("expected friend skill to target local player")
	}
	if actor.ID != 100 {
		t.Fatalf("target id = %d, want local account id 100", actor.ID)
	}

	actor, ok = clickedSkillTarget(ctx, projection, session.Skill{ID: 29, Type: skillTargetFriend}, int(pcPoint.x), int(pcPoint.y), now, nil)
	if !ok || actor.ID != 300 {
		t.Fatalf("friend skill target = %+v ok=%t, want pc 300", actor, ok)
	}
	if actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 13, Type: skillTargetEnemy}, int(pcPoint.x), int(pcPoint.y), now, nil); ok {
		t.Fatalf("enemy skill should not target pc outside pvp: %+v", actor)
	}

	world = worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             12,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	ctx.World = world
	projection = newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	mobPoint := projection.Project(cellCenter(12), cellCenter(20), 0)

	if actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 29, Type: skillTargetFriend}, int(mobPoint.x), int(mobPoint.y), now, nil); ok {
		t.Fatalf("friend skill should not target mob without noshift: %+v", actor)
	}

	actor, ok = clickedSkillTarget(ctx, projection, session.Skill{ID: 13, Type: skillTargetEnemy}, int(mobPoint.x), int(mobPoint.y), now, nil)
	if !ok || actor.ID != 400 {
		t.Fatalf("enemy skill target = %+v ok=%t, want mob 400", actor, ok)
	}
}

func TestClickedSkillTargetNoShiftAllowsSupportOnEnemies(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             12,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200, NoShift: true},
		World:   world,
	}
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	mobPoint := projection.Project(cellCenter(12), cellCenter(20), 0)

	actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 28, Type: skillTargetFriend}, int(mobPoint.x), int(mobPoint.y), now, nil)
	if !ok || actor.ID != 400 {
		t.Fatalf("noshift friend skill target = %+v ok=%t, want mob 400", actor, ok)
	}
}

func TestAttackTargetWithinRangeUsesMeleeAdjacency(t *testing.T) {
	if !attackTargetWithinRange(10, 20, 11, 21, 1) {
		t.Fatal("diagonal adjacent target should be in melee range")
	}
	if attackTargetWithinRange(10, 20, 12, 20, 1) {
		t.Fatal("two cells away should be out of melee range")
	}
	if !attackTargetWithinRange(10, 20, 15, 20, 5) {
		t.Fatal("five cells away should be in bow range")
	}
}

func TestAttackApproachCellChoosesClosestWalkableNeighbor(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 112, Y: 302}
	world.GAT = &res.GAT{
		Width:  200,
		Height: 400,
		Cells:  make([]res.GATCell, 200*400),
	}
	for i := range world.GAT.Cells {
		world.GAT.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
	ctx := client.Context{World: world}
	actor := worldstate.Actor{ID: 300, X: 116, Y: 303}

	x, y, ok := attackApproachCell(ctx, actor, 1)
	if !ok {
		t.Fatal("expected approach cell")
	}
	if x != 115 || y != 302 {
		t.Fatalf("approach = %d,%d, want 115,302", x, y)
	}

	world.GAT.Cells[302*world.GAT.Width+115] = res.GATCell{}
	x, y, ok = attackApproachCell(ctx, actor, 1)
	if !ok {
		t.Fatal("expected fallback approach cell")
	}
	if x == 115 && y == 302 {
		t.Fatalf("blocked approach cell was selected")
	}
}

func TestAttackApproachCellUsesRangedAttackRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 112, Y: 302}
	world.GAT = &res.GAT{
		Width:  200,
		Height: 400,
		Cells:  make([]res.GATCell, 200*400),
	}
	for i := range world.GAT.Cells {
		world.GAT.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
	ctx := client.Context{World: world}
	actor := worldstate.Actor{ID: 300, X: 120, Y: 302}

	x, y, ok := attackApproachCell(ctx, actor, 5)
	if !ok {
		t.Fatal("expected ranged approach cell")
	}
	targetDistance := maxInt(absInt(x-actor.X), absInt(y-actor.Y))
	if targetDistance > 5 {
		t.Fatalf("ranged approach = %d,%d, distance %d from target, want within 5", x, y, targetDistance)
	}
	if targetDistance <= 1 {
		t.Fatalf("ranged approach = %d,%d, want not adjacent contact", x, y)
	}
}

func TestCurrentNormalAttackRangeUsesEquippedBowAndVultureEye(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{Items: []session.InventoryItem{
			{Index: 1, ItemID: 1701, Location: db.EquipWeapon, Equip: true, Equipped: true},
		}},
		Skills: session.Skills{List: []session.Skill{
			{ID: 44, Level: 3},
		}},
	}
	ctx := client.Context{Session: sessionState}

	if got := currentNormalAttackRange(ctx); got != 8 {
		t.Fatalf("normal attack range = %d, want bow 5 + vulture 3", got)
	}
}

func TestApplyAttackFailureForDistanceStoresServerRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	sessionState := &session.Session{}
	ctx := client.Context{
		Session: sessionState,
		World:   world,
	}
	mode := &WorldMode{}

	mode.applyAttackFailureForDistance(ctx, network.AttackFailureForDistance{
		TargetID:    300,
		TargetX:     16,
		TargetY:     20,
		SourceX:     10,
		SourceY:     20,
		AttackRange: 7,
	})

	if sessionState.AttackRange != 7 {
		t.Fatalf("session attack range = %d, want server range 7", sessionState.AttackRange)
	}
}

func TestContinuePendingAttackSchedulesDelayedAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{
		pendingAttack: attackIntent{
			targetID: 300,
			expires:  time.Now().Add(time.Second),
		},
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	mode.continuePendingAttack(ctx, "test")

	if mode.pendingAttack.targetID != 300 {
		t.Fatalf("pending target cleared")
	}
	if mode.pendingAttack.readyAt.IsZero() {
		t.Fatal("pending attack was not scheduled")
	}
	if time.Until(mode.pendingAttack.readyAt) > 100*time.Millisecond {
		t.Fatalf("readyAt too far in future: %s", time.Until(mode.pendingAttack.readyAt))
	}
}

func TestUpdatePendingAttackDoesNotKeepDelayingScheduledAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	readyAt := time.Now().Add(10 * time.Millisecond)
	mode := &WorldMode{
		pendingAttack: attackIntent{
			targetID: 300,
			expires:  time.Now().Add(time.Second),
			readyAt:  readyAt,
		},
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	mode.updatePendingAttack(ctx, "test", false)

	if !mode.pendingAttack.readyAt.Equal(readyAt) {
		t.Fatalf("readyAt moved from %s to %s", readyAt, mode.pendingAttack.readyAt)
	}
}

func TestPendingAttackReadyAtWaitsForWalkEnd(t *testing.T) {
	now := time.Unix(100, 0)
	player := worldstate.Actor{
		Moving:       true,
		MoveStarted:  now.Add(-100 * time.Millisecond),
		MoveDuration: 600 * time.Millisecond,
	}

	got := pendingAttackReadyAt(player, now)
	want := now.Add(560 * time.Millisecond)
	if !got.Equal(want) {
		t.Fatalf("readyAt = %s, want %s", got.Sub(now), want.Sub(now))
	}
}

func TestUpdatePendingPickupDoesNotKeepDelayingScheduledAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	world.UpsertItem(worldstate.FloorItem{ID: 400, X: 10, Y: 20})
	readyAt := time.Now().Add(10 * time.Millisecond)
	mode := &WorldMode{
		pendingPickup: pickupIntent{
			itemID:  400,
			expires: time.Now().Add(time.Second),
			readyAt: readyAt,
		},
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	mode.updatePendingPickup(ctx, "test", false)

	if !mode.pendingPickup.readyAt.Equal(readyAt) {
		t.Fatalf("readyAt moved from %s to %s", readyAt, mode.pendingPickup.readyAt)
	}
}

func TestContinuePendingTargetSkillSchedulesAfterEnteringRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    session.Skill{ID: 50, Level: 1, Type: skillTargetEnemy, Range: 1},
			targetID: 300,
			expires:  time.Now().Add(time.Second),
		},
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	mode.skills().ContinuePendingTarget(ctx, "test")

	if mode.pendingSkill.targetID != 300 {
		t.Fatalf("pending target cleared")
	}
	if mode.pendingSkill.readyAt.IsZero() {
		t.Fatal("pending skill was not scheduled")
	}
	if time.Until(mode.pendingSkill.readyAt) > 100*time.Millisecond {
		t.Fatalf("readyAt too far in future: %s", time.Until(mode.pendingSkill.readyAt))
	}
}

func TestUpdatePendingTargetSkillDoesNotKeepDelayingScheduledAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	readyAt := time.Now().Add(10 * time.Millisecond)
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    session.Skill{ID: 50, Level: 1, Type: skillTargetEnemy, Range: 1},
			targetID: 300,
			expires:  time.Now().Add(time.Second),
			readyAt:  readyAt,
		},
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	mode.skills().UpdatePendingTarget(ctx, "test", false)

	if !mode.pendingSkill.readyAt.Equal(readyAt) {
		t.Fatalf("readyAt moved from %s to %s", readyAt, mode.pendingSkill.readyAt)
	}
}

func TestContinuePendingTargetSkillWaitsWhileOutOfRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             15,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    session.Skill{ID: 50, Level: 1, Type: skillTargetEnemy, Range: 1},
			targetID: 300,
			expires:  time.Now().Add(time.Second),
		},
	}
	ctx := client.Context{Session: &session.Session{AccountID: 100, CharID: 200}, World: world}

	mode.skills().ContinuePendingTarget(ctx, "test")

	if mode.pendingSkill.targetID != 300 {
		t.Fatalf("pending target cleared")
	}
	if !mode.pendingSkill.readyAt.IsZero() {
		t.Fatalf("pending skill readyAt = %s, want zero while out of range", mode.pendingSkill.readyAt)
	}
}

func TestAttackRetryDueUsesOpenMidgardInterval(t *testing.T) {
	now := time.Unix(100, 0)
	if !attackRetryDue(time.Time{}, now) {
		t.Fatal("zero last attack should be due")
	}
	if attackRetryDue(now.Add(-attackRetryInterval+time.Millisecond), now) {
		t.Fatal("attack should not retry before the interval")
	}
	if !attackRetryDue(now.Add(-attackRetryInterval), now) {
		t.Fatal("attack should retry at the interval")
	}
}

func TestNormalAttackLockActiveUsesNoCtrlOrHeldCtrl(t *testing.T) {
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, Session: &session.Session{}}

	if normalAttackLockActive(ctx) {
		t.Fatal("attack lock should be inactive without noctrl or held ctrl")
	}

	ctx.Session.NoCtrl = true
	if !normalAttackLockActive(ctx) {
		t.Fatal("attack lock should be active with noctrl")
	}

	ctx.Session.NoCtrl = false
	inputState.SetKey(input.KeyCtrl, true)
	if !normalAttackLockActive(ctx) {
		t.Fatal("attack lock should be active while ctrl is held")
	}
}

func TestLockAttackKeepsExistingRetryTimersForSameTarget(t *testing.T) {
	firstAttack := time.Unix(100, 0)
	firstChase := time.Unix(101, 0)
	mode := &WorldMode{
		lockedAttackID: 300,
		lastAttackAt:   firstAttack,
		lastChaseAt:    firstChase,
	}

	mode.lockAttack(300)
	if mode.lastAttackAt != firstAttack || mode.lastChaseAt != firstChase {
		t.Fatal("same target lock reset retry timers")
	}

	mode.lockAttack(400)
	if mode.lockedAttackID != 400 {
		t.Fatalf("locked target = %d, want 400", mode.lockedAttackID)
	}
	if !mode.lastAttackAt.IsZero() || !mode.lastChaseAt.IsZero() {
		t.Fatal("new target lock should reset retry timers")
	}
}

func TestApplyParameterChangeUpdatesVitals(t *testing.T) {
	sessionState := &session.Session{
		Selected: session.Character{HP: 70, MaxHP: 100, SP: 20, MaxSP: 30},
		Vitals:   session.Vitals{HP: 70, MaxHP: 100, SP: 20, MaxSP: 30},
	}
	ctx := client.Context{Session: sessionState}

	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusHP, Value: 42})
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusMaxSP, Value: 55})

	if sessionState.Vitals.HP != 42 || sessionState.Vitals.MaxHP != 100 || sessionState.Vitals.SP != 20 || sessionState.Vitals.MaxSP != 55 {
		t.Fatalf("vitals = %+v", sessionState.Vitals)
	}
	if sessionState.Selected.HP != 42 || sessionState.Selected.MaxSP != 55 {
		t.Fatalf("selected vitals = hp %d maxsp %d", sessionState.Selected.HP, sessionState.Selected.MaxSP)
	}
}

func TestApplyParameterChangeUpdatesPlayerSpeed(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState, World: world}

	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusSpeed, Value: 100})

	if world.Player.Speed != 100 {
		t.Fatalf("player speed = %d, want 100", world.Player.Speed)
	}
}

func TestLocalPlayerMoveSpeedAppliesPushcartMalus(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{
		Selected: session.Character{ID: 150004, Option: db.EffectStateCart1},
		Skills: session.Skills{List: []session.Skill{
			{ID: skillPushCart, Level: 5},
		}},
	}
	ctx := client.Context{Session: sessionState, World: world}

	refreshLocalPlayerMoveSpeed(ctx)

	if world.Player.Speed != 187 {
		t.Fatalf("player speed = %d, want 187", world.Player.Speed)
	}
}

func TestLocalPlayerMoveSpeedHasNoPushcartMalusAtLevelTen(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{
		Selected: session.Character{ID: 150004, Option: db.EffectStateCart1},
		Skills: session.Skills{List: []session.Skill{
			{ID: skillPushCart, Level: 10},
		}},
	}
	ctx := client.Context{Session: sessionState, World: world}

	refreshLocalPlayerMoveSpeed(ctx)

	if world.Player.Speed != defaultPlayerMoveSpeedMS {
		t.Fatalf("player speed = %d, want %d", world.Player.Speed, defaultPlayerMoveSpeedMS)
	}
}

func TestLocalPlayerMoveSpeedUsesServerSpeedBeforePushcartMalus(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{
		Selected: session.Character{ID: 150004, Option: db.EffectStateCart1},
		Movement: session.Movement{
			ServerSpeed:    100,
			HasServerSpeed: true,
		},
		Skills: session.Skills{List: []session.Skill{
			{ID: skillPushCart, Level: 5},
		}},
	}
	ctx := client.Context{Session: sessionState, World: world}

	refreshLocalPlayerMoveSpeed(ctx)

	if world.Player.Speed != 125 {
		t.Fatalf("player speed = %d, want 125", world.Player.Speed)
	}
}

func TestWorldModeParameterChangeRecoveryDoesNotDisplayFeedback(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Selected:  session.Character{HP: 70, MaxHP: 100, SP: 20, MaxSP: 30},
		Vitals:    session.Vitals{HP: 70, MaxHP: 100, SP: 20, MaxSP: 30},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusHP, Value: 85})
	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusSP, Value: 25})

	if sessionState.Vitals.HP != 85 || sessionState.Vitals.SP != 25 {
		t.Fatalf("vitals = hp %d sp %d, want 85/25", sessionState.Vitals.HP, sessionState.Vitals.SP)
	}
	if len(mode.damageFloaters) != 0 {
		t.Fatalf("parameter changes should not add recovery floaters: %+v", mode.damageFloaters)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
}

func TestApplyRecoveryUpdatesHPAndAddsFloater(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Selected:  session.Character{HP: 70, MaxHP: 100},
		Vitals:    session.Vitals{HP: 70, MaxHP: 100},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyRecovery(ctx, network.Recovery{StatusID: network.StatusHP, Amount: 12})

	if sessionState.Vitals.HP != 82 || sessionState.Selected.HP != 82 {
		t.Fatalf("hp = vitals %d selected %d, want 82", sessionState.Vitals.HP, sessionState.Selected.HP)
	}
	if len(mode.damageFloaters) != 1 {
		t.Fatalf("floaters = %d, want 1", len(mode.damageFloaters))
	}
	floater := mode.damageFloaters[0]
	if floater.actorID != 2000000 || floater.text != "12" || floater.color != recoveryHPColor {
		t.Fatalf("floater = %+v", floater)
	}
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %d, want 1", len(mode.scheduledSounds))
	}
	if got := mode.scheduledSounds[0].paths; len(got) != 1 || got[0] != recoveryHPSFX {
		t.Fatalf("scheduled sound paths = %v, want %q", got, recoveryHPSFX)
	}
}

func TestApplyRecoveryUpdatesSPAndAddsFloater(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Selected:  session.Character{SP: 20, MaxSP: 30},
		Vitals:    session.Vitals{SP: 20, MaxSP: 30},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyRecovery(ctx, network.Recovery{StatusID: network.StatusSP, Amount: 7})

	if sessionState.Vitals.SP != 27 || sessionState.Selected.SP != 27 {
		t.Fatalf("sp = vitals %d selected %d, want 27", sessionState.Vitals.SP, sessionState.Selected.SP)
	}
	if len(mode.damageFloaters) != 1 {
		t.Fatalf("floaters = %d, want 1", len(mode.damageFloaters))
	}
	floater := mode.damageFloaters[0]
	if floater.actorID != 2000000 || floater.text != "7" || floater.color != recoverySPColor || floater.kind != damageFloaterRecoverySP {
		t.Fatalf("floater = %+v", floater)
	}
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %d, want 1", len(mode.scheduledSounds))
	}
	if got := mode.scheduledSounds[0].paths; len(got) != 1 || got[0] != recoverySPSFX {
		t.Fatalf("scheduled sound paths = %v, want %q", got, recoverySPSFX)
	}
}

func TestApplyParameterChangeUpdatesProgress(t *testing.T) {
	sessionState := &session.Session{
		Selected: session.Character{Level: 4},
		Progress: session.Progress{
			BaseLevel: 4,
		},
	}
	ctx := client.Context{Session: sessionState}

	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseExp, Value: 123})
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusNextBaseExp, Value: 1000})
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobExp, Value: 45})
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusNextJobExp, Value: 500})
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseLevel, Value: 5})
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobLevel, Value: 3})

	if sessionState.Progress.BaseLevel != 5 || sessionState.Progress.JobLevel != 3 {
		t.Fatalf("levels = base %d job %d", sessionState.Progress.BaseLevel, sessionState.Progress.JobLevel)
	}
	if sessionState.Progress.BaseExp != 123 || sessionState.Progress.NextBaseExp != 1000 || sessionState.Progress.JobExp != 45 || sessionState.Progress.NextJobExp != 500 {
		t.Fatalf("progress = %+v", sessionState.Progress)
	}
	if sessionState.Selected.Level != 5 {
		t.Fatalf("selected level = %d, want 5", sessionState.Selected.Level)
	}
}

func TestApplyParameterChangeUpdatesInventory(t *testing.T) {
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusZeny, Value: 1234567})
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusWeight, Value: 240})
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusMaxWeight, Value: 2000})

	if sessionState.Inventory.Zeny != 1234567 || sessionState.Inventory.Weight != 240 || sessionState.Inventory.MaxWeight != 2000 {
		t.Fatalf("inventory = %+v", sessionState.Inventory)
	}
}

func TestSelectCharacterSeedsInventoryZeny(t *testing.T) {
	sessionState := &session.Session{}

	sessionState.SelectCharacter(session.Character{ID: 1234, Money: 95000})

	if sessionState.Inventory.Zeny != 95000 {
		t.Fatalf("zeny = %d, want 95000", sessionState.Inventory.Zeny)
	}
}

func TestFormatHUDNumberGroupsThousands(t *testing.T) {
	if got := gameui.FormatHUDNumber(123456789); got != "123,456,789" {
		t.Fatalf("formatted number = %q", got)
	}
}

func TestSessionProgressFromCharacterUsesBaseLevel(t *testing.T) {
	progress := session.ProgressFromCharacter(session.Character{Level: 12, JobLevel: 7})
	if progress.BaseLevel != 12 || progress.JobLevel != 7 {
		t.Fatalf("progress = %+v", progress)
	}
}

func TestFormatEXPPercent(t *testing.T) {
	if got := gameui.FormatEXPPercent(123, 1000); got != "12.3%" {
		t.Fatalf("formatted exp percent = %q", got)
	}
	if got := gameui.FormatEXPPercent(12, 0); got != "--" {
		t.Fatalf("formatted exp percent without next = %q", got)
	}
	if got := gameui.FormatEXPPercent(1001, 1000); got != "100.0%" {
		t.Fatalf("formatted capped exp percent = %q", got)
	}
}

func TestDisplayWeightUsesROVisibleUnits(t *testing.T) {
	if got := gameui.DisplayWeight(240); got != 24 {
		t.Fatalf("display weight = %d, want 24", got)
	}
}

func TestApplyActorActionNotifySchedulesAttackAndHitAnimations(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
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
		SourceID:    2000000,
		TargetID:    300,
		SkillID:     0,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	sourceAnim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("local character animation missing")
	}
	if sourceAnim.actionFamily != spriteActionPCAttack3 {
		t.Fatalf("source action = %d, want %d", sourceAnim.actionFamily, spriteActionPCAttack3)
	}
	earlyAnim, ok := mode.actorAnimation(150000, sourceAnim.started.Add(sourceAnim.duration-time.Millisecond))
	if !ok || earlyAnim.actionFamily != spriteActionPCAttack3 {
		t.Fatalf("early source animation = %+v ok=%t, want attack until it finishes", earlyAnim, ok)
	}
	readyAnim, ok := mode.actorAnimation(150000, sourceAnim.started.Add(sourceAnim.duration))
	if !ok || readyAnim.actionFamily != spriteActionPCReadyFight || !readyAnim.loop {
		t.Fatalf("post-attack animation = %+v ok=%t, want looping READYFIGHT after attack", readyAnim, ok)
	}
	targetAnim, ok := mode.actorAnims[300]
	if !ok {
		t.Fatal("target animation missing")
	}
	if targetAnim.actionFamily != spriteActionNonPCHurt {
		t.Fatalf("target action = %d, want %d", targetAnim.actionFamily, spriteActionNonPCHurt)
	}
	if targetAnim.started.Sub(sourceAnim.started) != 580*time.Millisecond {
		t.Fatalf("hit delay = %s, want 580ms", targetAnim.started.Sub(sourceAnim.started))
	}
	if world.Dir != directionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
	}
	if len(mode.damageFloaters) != 1 || !mode.damageFloaters[0].starts.Equal(targetAnim.started) {
		t.Fatalf("damage floater = %+v targetStarted=%s", mode.damageFloaters, targetAnim.started)
	}
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want regular hit effect", len(mode.worldEffects))
	}
	hitEffect := mode.worldEffects[0]
	if hitEffect.effectID != effectHit1 || hitEffect.actorID != 300 || !hitEffect.starts.Equal(targetAnim.started) {
		t.Fatalf("regular hit effect = %+v targetStarted=%s", hitEffect, targetAnim.started)
	}
	if _, ok := mode.actorLife[300]; ok {
		t.Fatal("target life should not be estimated from damage")
	}

	applySelfMoveAck(ctx, network.SelfMoveAck{FromX: 10, FromY: 20, ToX: 12, ToY: 20})
	mode.clearLocalActorAction(ctx)
	if anim, ok := mode.actorAnimation(150000, time.Now()); ok {
		t.Fatalf("post-move animation = %+v, want cleared", anim)
	}
}

func TestApplyActorActionNotifyAddsCriticalNormalHitEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
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
		SourceID:    2000000,
		TargetID:    300,
		SkillID:     0,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      10,
	})

	targetAnim, ok := mode.actorAnims[300]
	if !ok {
		t.Fatal("target animation missing")
	}
	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %+v, want regular and critical hit effects", mode.worldEffects)
	}
	if effect := mode.worldEffects[0]; effect.effectID != effectHit1 || effect.actorID != 300 || !effect.starts.Equal(targetAnim.started) {
		t.Fatalf("regular hit effect = %+v targetStarted=%s", effect, targetAnim.started)
	}
	if effect := mode.worldEffects[1]; effect.effectID != effectBashHit || effect.actorID != 300 || !effect.starts.Equal(targetAnim.started) {
		t.Fatalf("critical hit effect = %+v targetStarted=%s", effect, targetAnim.started)
	}
	if len(mode.damageFloaters) != 1 || mode.damageFloaters[0].kind != damageFloaterCritical {
		t.Fatalf("damage floaters = %+v, want critical", mode.damageFloaters)
	}
}

func TestApplyActorActionNotifyAddsBashHitEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Hair: 1}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     5,
		SkillLevel:  1,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      6,
	})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want begin and hit effects", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.effectID != effectBashBegin || effect.actorID != 2000000 {
		t.Fatalf("begin effect = %+v", effect)
	}
	effect := mode.worldEffects[1]
	if effect.effectID != effectBashHit || effect.actorID != 300 || effect.x != 11 || effect.y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
	if delay := effect.starts.Sub(mode.actorAnims[150000].started); delay != 580*time.Millisecond {
		t.Fatalf("effect delay = %s, want 580ms", delay)
	}
}

func TestApplyActorActionNotifyChainsPlayerHurtToReadyFight(t *testing.T) {
	world := worldstate.New()
	moveStarted := time.Now()
	world.UpsertActor(worldstate.Actor{
		ID:           300,
		X:            15,
		Y:            20,
		Job:          1,
		Moving:       true,
		FromX:        10,
		FromY:        20,
		ToX:          15,
		ToY:          20,
		MoveStarted:  moveStarted,
		MoveDuration: 5 * time.Second,
	})
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	hurtAnim, ok := mode.actorAnims[300]
	if !ok {
		t.Fatal("target hurt animation missing")
	}
	if hurtAnim.actionFamily != spriteActionPCHurt {
		t.Fatalf("target action = %d, want %d", hurtAnim.actionFamily, spriteActionPCHurt)
	}
	if actor := world.Actors[300]; !actor.Moving {
		t.Fatal("target should keep moving before scheduled hit time")
	}
	mode.processScheduledActorStops(ctx, hurtAnim.started)
	actor := world.Actors[300]
	if actor.Moving || actor.MovePath != nil || actor.FromX != actor.X || actor.ToX != actor.X {
		t.Fatalf("target movement after hit = %+v, want stopped at hit position", actor)
	}
	if actor.X == 15 {
		t.Fatalf("target stopped at destination, want interpolated hit position: %+v", actor)
	}
	readyAnim, ok := mode.actorAnimation(300, hurtAnim.started.Add(hurtAnim.duration))
	if !ok || readyAnim.actionFamily != spriteActionPCReadyFight || !readyAnim.loop {
		t.Fatalf("post-hurt animation = %+v ok=%t, want looping READYFIGHT", readyAnim, ok)
	}
}

func TestApplyActorActionNotifyStopsLocalPlayerMovementOnHit(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           2000000,
		X:            15,
		Y:            20,
		Job:          1,
		Moving:       true,
		FromX:        10,
		FromY:        20,
		ToX:          15,
		ToY:          20,
		MoveStarted:  time.Now(),
		MoveDuration: 5 * time.Second,
	}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    2000000,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	hurtAnim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("local hurt animation missing")
	}
	if world.Player.Moving == false {
		t.Fatal("local player should keep moving before scheduled hit time")
	}
	mode.processScheduledActorStops(ctx, hurtAnim.started)
	if world.Player.Moving || world.Player.MovePath != nil || world.Player.FromX != world.Player.X || world.Player.ToX != world.Player.X {
		t.Fatalf("local player movement after hit = %+v, want stopped", world.Player)
	}
	if world.Player.X == 15 {
		t.Fatalf("local player stopped at destination, want interpolated hit position: %+v", world.Player)
	}
	if ctx.Session.PlayerX != world.Player.X || ctx.Session.PlayerY != world.Player.Y {
		t.Fatalf("session player position = %d,%d world=%d,%d", ctx.Session.PlayerX, ctx.Session.PlayerY, world.Player.X, world.Player.Y)
	}
}

func TestApplyActorActionNotifyClearsLocalCastBarOnHit(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 15, Y: 20, Job: 2}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}
	mode.startActorCastBar(ctx, 2000000, 2*time.Second, time.Now())
	if len(mode.actorCastBars) != 2 {
		t.Fatalf("cast bars = %d, want account and char aliases", len(mode.actorCastBars))
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    2000000,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	if len(mode.actorCastBars) != 0 {
		t.Fatalf("cast bars after hit = %+v, want cleared", mode.actorCastBars)
	}
}

func TestApplyActorActionNotifyResumesFocusedLocalWalkAfterHurt(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           2000000,
		X:            15,
		Y:            20,
		Job:          1,
		Moving:       true,
		FromX:        10,
		FromY:        20,
		ToX:          15,
		ToY:          20,
		MoveStarted:  time.Now(),
		MoveDuration: 5 * time.Second,
		MovePath: []worldstate.WalkStep{
			{X: 10, Y: 20},
			{X: 15, Y: 20},
		},
	}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{lockedAttackID: 400}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    2000000,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	hurtAnim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("local hurt animation missing")
	}
	mode.processScheduledActorStops(ctx, hurtAnim.started)
	if world.Player.Moving {
		t.Fatalf("local player moving during hurt = %+v, want paused", world.Player)
	}
	if len(mode.scheduledResumes) == 0 {
		t.Fatal("scheduled walk resume missing")
	}
	mode.processScheduledWalkResumes(ctx, hurtAnim.started.Add(hurtAnim.duration))
	if !world.Player.Moving {
		t.Fatalf("local player after hurt = %+v, want resumed walk", world.Player)
	}
	if world.Player.ToX != 15 || world.Player.ToY != 20 {
		t.Fatalf("local player resumed target = %d,%d, want 15,20", world.Player.ToX, world.Player.ToY)
	}
	afterWalk := world.Player.MoveStarted.Add(world.Player.MoveDuration).Add(time.Millisecond)
	if anim, ok := mode.actorAnimation(150000, afterWalk); ok {
		t.Fatalf("local player animation after resumed walk = %+v, want idle/no stale READYFIGHT", anim)
	}
	if anim, ok := mode.actorAnimation(2000000, afterWalk); ok {
		t.Fatalf("local account animation after resumed walk = %+v, want idle/no stale READYFIGHT", anim)
	}
}

func TestMovingActorPacketClearsReadyFightAction(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, X: 10, Y: 20, Job: 1, Speed: 150})
	mode := &WorldMode{actorAnims: map[uint32]actorAnimation{
		300: {
			actionFamily: spriteActionPCReadyFight,
			loop:         true,
			play:         true,
			hasPlay:      true,
		},
	}}
	ctx := client.Context{World: world}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:         300,
		X:          15,
		Y:          20,
		Job:        1,
		Moving:     true,
		FromX:      10,
		FromY:      20,
		ToX:        15,
		ToY:        20,
		Appearance: true,
		Speed:      150,
	})

	if anim, ok := mode.actorAnims[300]; ok {
		t.Fatalf("moving actor animation = %+v, want movement to replace stale READYFIGHT", anim)
	}
	if actor := world.Actors[300]; !actor.Moving || actor.FromX != 10 || actor.ToX != 15 {
		t.Fatalf("moving actor = %+v, want movement preserved", actor)
	}
}

func TestSetActorActionStopsMovingActor(t *testing.T) {
	world := worldstate.New()
	started := time.Now().Add(-time.Second)
	world.UpsertActor(worldstate.Actor{
		ID:           300,
		X:            15,
		Y:            20,
		Job:          1,
		Moving:       true,
		FromX:        10,
		FromY:        20,
		ToX:          15,
		ToY:          20,
		MoveStarted:  started,
		MoveDuration: 5 * time.Second,
	})
	mode := &WorldMode{}
	ctx := client.Context{World: world}

	mode.startCombatAnimation(ctx, 300, spriteActionPCAttack2, time.Now(), defaultAttackAnimationDuration)

	actor := world.Actors[300]
	if actor.Moving || actor.MovePath != nil {
		t.Fatalf("actor movement after action = %+v, want stopped", actor)
	}
	if actor.X == 15 {
		t.Fatalf("actor stopped at destination, want interpolated action position: %+v", actor)
	}
}

func TestActorAnimationHonorsDelayedNextAction(t *testing.T) {
	started := time.Now()
	mode := &WorldMode{actorAnims: map[uint32]actorAnimation{
		300: {
			actionFamily: spriteActionPCSkill,
			started:      started,
			duration:     100 * time.Millisecond,
			next: &actorAnimation{
				actionFamily: spriteActionPCReadyFight,
				startDelay:   50 * time.Millisecond,
				loop:         true,
				play:         true,
				hasPlay:      true,
			},
		},
	}}

	if anim, ok := mode.actorAnimation(300, started.Add(125*time.Millisecond)); ok {
		t.Fatalf("actorAnimation before delayed next = %+v, want inactive gap", anim)
	}
	anim, ok := mode.actorAnimation(300, started.Add(150*time.Millisecond))
	if !ok || anim.actionFamily != spriteActionPCReadyFight || !anim.loop {
		t.Fatalf("actorAnimation delayed next = %+v ok=%t, want ready fight loop", anim, ok)
	}
	if got := anim.started.Sub(started); got != 150*time.Millisecond {
		t.Fatalf("delayed next start = %s, want 150ms", got)
	}
}

func TestBodyMotionHonorsRobrowserActionMetadata(t *testing.T) {
	action := res.ACTAction{
		DelayMS: 100,
		Animations: []res.ACTAnimation{
			{}, {}, {}, {}, {},
		},
	}
	started := time.Now()
	state := spriteState{
		started:        started,
		actionFamily:   spriteActionPCSkill,
		frameOffset:    1,
		hasFrameOffset: true,
		length:         3,
		hasLength:      true,
		speed:          50 * time.Millisecond,
		hasSpeed:       true,
	}

	if got := bodyMotionForState(action, state, started, started.Add(25*time.Millisecond)); got != 1 {
		t.Fatalf("initial offset motion = %d, want 1", got)
	}
	if got := bodyMotionForState(action, state, started, started.Add(125*time.Millisecond)); got != 3 {
		t.Fatalf("length-limited motion = %d, want 3", got)
	}
	state.play = false
	state.hasPlay = true
	if got := bodyMotionForState(action, state, started, started.Add(time.Second)); got != 1 {
		t.Fatalf("play=false motion = %d, want fixed frame offset 1", got)
	}
}

func TestBashBeginEffectSpecUsesCylinderComponents(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing")
	}
	if spec.duration != time.Second {
		t.Fatalf("duration = %s, want 1s", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_bash.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 3 {
		t.Fatalf("components = %d, want 3", len(spec.components))
	}
	first := spec.components[0]
	if first.kind != effectComponentCylinder || first.textureName != "alpha_down" || first.circleSides != 20 || first.totalCircleSides != 20 {
		t.Fatalf("first component = %+v", first)
	}
	second := spec.components[1]
	if second.kind != effectComponentCylinder || second.textureName != "alpha_center" || second.duplicate != 10 || second.circleSides != 1 || second.totalCircleSides != 30 {
		t.Fatalf("second component = %+v", second)
	}
	third := spec.components[2]
	if third.kind != effectComponentCylinder || third.textureName != "alpha_center" || third.duplicate != 8 || third.topSize != 4.0 {
		t.Fatalf("third component = %+v", third)
	}
}

func TestWorldEffectSpecCatalogCoverage(t *testing.T) {
	coverage := effectCoverageSnapshot()
	if coverage.Implemented != 641 {
		t.Fatalf("implemented effects = %d, want 641", coverage.Implemented)
	}
	if coverage.ReferenceActive != 607 || coverage.ReferenceAll != 1147 {
		t.Fatalf("reference client totals = active %d all %d", coverage.ReferenceActive, coverage.ReferenceAll)
	}
	if coverage.ActivePercent < 105.5 || coverage.ActivePercent > 105.8 {
		t.Fatalf("active coverage = %.3f, want about 105.6", coverage.ActivePercent)
	}
}

func TestRobrowserActiveEffectsZeroToFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectHit1:           "EF_HIT1",
		effectBashHit:        "EF_HIT2",
		effectHit3:           "EF_HIT3",
		effectHit4:           "EF_HIT4",
		effectHit5:           "EF_HIT5",
		effectHit6:           "EF_HIT6",
		effectEntry:          "EF_ENTRY",
		effectExit:           "EF_EXIT",
		effectWarp:           "EF_WARP",
		effectEnhance:        "EF_ENHANCE",
		effectMammonite:      "EF_COIN",
		effectEndure:         "EF_ENDURE",
		effectBeginSpell:     "EF_BEGINSPELL",
		effectGlassWall:      "EF_GLASSWALL",
		effectHealSP:         "EF_HEALSP",
		effectSoulStrike:     "EF_SOULSTRIKE",
		effectBashBegin:      "EF_BASH",
		effectMagnumBreak:    "EF_MAGNUMBREAK",
		effectSteal:          "EF_STEAL",
		effectPoisonAttack:   "EF_PATTACK",
		effectDetoxication:   "EF_DETOXICATION",
		effectSight:          "EF_SIGHT",
		effectStoneCurse:     "EF_STONECURSE",
		effectFireBall:       "EF_FIREBALL",
		effectFireWall:       "EF_FIREWALL",
		effectIceArrow:       "EF_ICEARROW",
		effectFrostDiver:     "EF_FROSTDIVER",
		effectFrostDiverHit:  "EF_FROSTDIVER2",
		effectLightningBolt:  "EF_LIGHTBOLT",
		effectThunderStorm:   "EF_THUNDERSTORM",
		effectFireArrow:      "EF_FIREARROW",
		effectNapalmBeat:     "EF_NAPALMBEAT",
		effectRuwach:         "EF_RUWACH",
		effectTeleportOld:    "EF_TELEPORTATION",
		effectReadyPortalOld: "EF_READYPORTAL",
		effectIncAgility:     "EF_INCAGILITY",
		effectDecAgility:     "EF_DECAGILITY",
		effectAqua:           "EF_AQUA",
		effectSignum:         "EF_SIGNUM",
		effectAngelus:        "EF_ANGELUS",
		effectBlessing:       "EF_BLESSING",
		effectIncAgiDex:      "EF_INCAGIDEX",
		effectSmoke:          "EF_SMOKE",
		effectFirefly:        "EF_FIREFLY",
		effectTorch:          "EF_TORCH",
		effectFireHit:        "EF_FIREHIT",
		effectFireSplashHit:  "EF_FIRESPLASHHIT",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiftyToOneHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectFireSplashHit:  "EF_FIRESPLASHHIT",
		effectColdHit:        "EF_COLDHIT",
		effectWindHit:        "EF_WINDHIT",
		effectPoisonHit:      "EF_POISONHIT",
		effectBeginSpell2:    "EF_BEGINSPELL2",
		effectBeginSpell3:    "EF_BEGINSPELL3",
		effectBeginSpell4:    "EF_BEGINSPELL4",
		effectBeginSpell5:    "EF_BEGINSPELL5",
		effectBeginSpell6:    "EF_BEGINSPELL6",
		effectBeginSpell7:    "EF_BEGINSPELL7",
		effectLockOnTarget:   "EF_LOCKON",
		effectWarpZone:       "EF_WARPZONE",
		effectSightTrasher:   "EF_SIGHTRASHER",
		effectArrowShotRO:    "EF_ARROWSHOT",
		effectInvenom:        "EF_INVENOM",
		effectCure:           "EF_CURE",
		effectProvoke:        "EF_PROVOKE",
		effectMvp:            "EF_MVP",
		effectSkidTrap:       "EF_SKIDTRAP",
		effectBrandishSpear:  "EF_BRANDISHSPEAR",
		effectIceWall:        "EF_ICEWALL",
		effectGloria:         "EF_GLORIA",
		effectMagnificat:     "EF_MAGNIFICAT",
		effectResurrection:   "EF_RESURRECTION",
		effectRecovery:       "EF_RECOVERY",
		effectEarthSpike:     "EF_EARTHSPIKE",
		effectSpearBoomerang: "EF_SPEARBMR",
		effectPierce:         "EF_PIERCE",
		effectTurnUndead:     "EF_TURNUNDEAD",
		effectSanctuary:      "EF_SANCTUARY",
		effectImpositio:      "EF_IMPOSITIO",
		effectLexAeterna:     "EF_LEXAETERNA",
		effectAspersio:       "EF_ASPERSIO",
		effectLexDivina:      "EF_LEXDIVINA",
		effectSuffragium:     "EF_SUFFRAGIUM",
		effectStormGust:      "EF_STORMGUST",
		effectLordVermilion:  "EF_LORD",
		effectBenedictio:     "EF_BENEDICTIO",
		effectMeteorStorm:    "EF_METEORSTORM",
		effectJupitelThunder: "EF_YUFITEL",
		effectJupitelHit:     "EF_YUFITELHIT",
		effectQuagmire:       "EF_QUAGMIRE",
		effectFirePillar:     "EF_FIREPILLAR",
		effectFirePillarBomb: "EF_FIREPILLARBOMB",
		effectHasteUp:        "EF_HASTEUP",
		effectFlasher:        "EF_FLASHER",
		effectRemoveTrap:     "EF_REMOVETRAP",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsFiftyToOneHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
	}{
		{"EF_ARROWSHOT", effectArrowShotRO, "arrowshot", "", true},
		{"EF_INVENOM", effectInvenom, "invenom", "effect\\thief_invenom.wav", true},
		{"EF_SKIDTRAP", effectSkidTrap, "skidtrap", "effect\\hunter_skidtrap.wav", false},
		{"EF_BRANDISHSPEAR", effectBrandishSpear, "brandish", "effect\\knight_brandish_spear.wav", false},
		{"EF_RECOVERY", effectRecovery, "recovery", "effect\\priest_recovery.wav", true},
		{"EF_SANCTUARY", effectSanctuary, "sanctuary", "effect\\priest_sanctuary.wav", true},
		{"EF_IMPOSITIO", effectImpositio, "impositio", "effect\\priest_impositio.wav", true},
		{"EF_ASPERSIO", effectAspersio, "aspersio", "effect\\priest_aspersio.wav", true},
		{"EF_LEXDIVINA", effectLexDivina, "lexdivina", "effect\\priest_lexdivina.wav", true},
		{"EF_LORD", effectLordVermilion, "lord", "effect\\wizard_fire_ivy.wav", true},
		{"EF_BENEDICTIO", effectBenedictio, "benedictio", "effect\\priest_benedictio.wav", true},
		{"EF_QUAGMIRE", effectQuagmire, "quagmire", "effect\\wizard_quagmire.wav", false},
		{"EF_FIREPILLAR", effectFirePillar, "firepillar", "effect\\wizard_fire_pillar_a.wav", false},
		{"EF_FIREPILLARBOMB", effectFirePillarBomb, "firepillarbomb", "effect\\wizard_fire_pillar_b.wav", false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_SPEARBMR", effectSpearBoomerang, "effect\\ef_fireball.wav"},
		{"EF_PIERCE", effectPierce, "effect\\ef_bash.wav"},
		{"EF_TURNUNDEAD", effectTurnUndead, "effect\\ef_bash.wav"},
		{"EF_HASTEUP", effectHasteUp, "effect\\black_adrenalinerush_b.wav"},
		{"EF_FLASHER", effectFlasher, "effect\\hunter_flasher.wav"},
		{"EF_REMOVETRAP", effectRemoveTrap, "effect\\hunter_removetrap.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserWarpZoneAndPoisonHitSpecs(t *testing.T) {
	poison, ok := worldEffectSpecForID(effectPoisonHit)
	if !ok || len(poison.components) != 1 {
		t.Fatalf("EF_POISONHIT spec = %+v ok=%t, want one SPR component", poison, ok)
	}
	if len(poison.sfx) != 1 || poison.sfx[0] != "effect\\ef_poisonattack.wav" {
		t.Fatalf("EF_POISONHIT sfx = %v", poison.sfx)
	}
	if component := poison.components[0]; component.kind != effectComponentSPR || component.spriteFile != "poisonhit" || component.attachedEntity {
		t.Fatalf("EF_POISONHIT component = %+v", component)
	}

	warp, ok := worldEffectSpecForID(effectWarpZone)
	if !ok || len(warp.components) != 3 || warp.duration != 2800*time.Millisecond {
		t.Fatalf("EF_WARPZONE spec = %+v ok=%t", warp, ok)
	}
	first, second, particle := warp.components[0], warp.components[1], warp.components[2]
	if first.kind != effectComponentCylinder || second.kind != effectComponentCylinder || first.textureName != "ring_blue" || second.textureName != "ring_blue" {
		t.Fatalf("EF_WARPZONE cylinders = %+v %+v", first, second)
	}
	if first.bottomSize != 2 || first.topSize != 3.3 || second.bottomSize != 1.9 || second.topSize != 3.2 || first.height != 1.1 || second.height != 1.1 {
		t.Fatalf("EF_WARPZONE cylinder sizes = %+v %+v", first, second)
	}
	if !first.attachedEntity || !second.attachedEntity || !first.blendAdditive || !second.blendAdditive || !first.fade || !second.fade || !first.rotate || !second.rotate || first.animation != 3 || second.animation != 3 {
		t.Fatalf("EF_WARPZONE cylinder flags = %+v %+v", first, second)
	}
	if particle.kind != effectComponent3D || particle.textureFile != "effect/pok1.tga" || particle.duration != time.Second || particle.duplicate != 3 || !particle.attachedEntity {
		t.Fatalf("EF_WARPZONE particle = %+v", particle)
	}
	if particle.posXStartRand != 3 || particle.posYStartRand != 3 || particle.posZEndRand != 2 || particle.posZEndMiddle != 2 || particle.sizeStart != effectTableSize(100) || particle.sizeRand != effectTableSize(17) {
		t.Fatalf("EF_WARPZONE particle motion/size = %+v", particle)
	}
}

func TestRobrowserSightTrasherEffectSpec(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectSightTrasher)
	if !ok || len(spec.components) != 16 || spec.duration != 800*time.Millisecond {
		t.Fatalf("EF_SIGHTRASHER spec = %+v ok=%t", spec, ok)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\wizard_sightrasher.wav" {
		t.Fatalf("EF_SIGHTRASHER sfx = %v", spec.sfx)
	}
	shadow, sight := spec.components[0], spec.components[1]
	if shadow.kind != effectComponent3D || !shadow.shadowTexture || shadow.spriteFile != "data\\sprite\\shadow" || shadow.duplicate != 4 || shadow.duplicateDelay != 100*time.Millisecond {
		t.Fatalf("EF_SIGHTRASHER shadow = %+v", shadow)
	}
	if shadow.posXEnd != 0 || shadow.posYEnd != -8 || shadow.posZ != 2 || shadow.posZEnd != 2 || shadow.sizeStart != effectTableSize(60) || shadow.sizeEnd != effectTableSize(160) || shadow.sizeDelta != -60 || shadow.blendMode != 8 {
		t.Fatalf("EF_SIGHTRASHER shadow motion/size = %+v", shadow)
	}
	if sight.kind != effectComponent3D || sight.shadowTexture || sight.spriteFile != "sight" || sight.alphaMax != 123.0/255.0 || sight.alphaMaxDelta != 3.0/255.0 || sight.sizeStart != effectTableSize(20) || sight.sizeEnd != effectTableSize(260) {
		t.Fatalf("EF_SIGHTRASHER sight = %+v", sight)
	}
	last := spec.components[len(spec.components)-1]
	if last.posXEnd != -5.66 || last.posYEnd != -5.66 {
		t.Fatalf("EF_SIGHTRASHER northwest component = %+v", last)
	}
}

func TestRobrowserQuadHornEffectSpecs(t *testing.T) {
	ice, ok := worldEffectSpecForID(effectIceWall)
	if !ok || len(ice.components) != 3 || ice.duration != 5*time.Minute {
		t.Fatalf("EF_ICEWALL spec = %+v ok=%t", ice, ok)
	}
	if len(ice.sfx) != 1 || ice.sfx[0] != "effect\\wizard_icewall.wav" {
		t.Fatalf("EF_ICEWALL sfx = %v", ice.sfx)
	}
	firstIce := ice.components[0]
	if firstIce.kind != effectComponentQuadHorn || firstIce.textureFile != "effect/ice.tga" || firstIce.duration != 5*time.Minute || firstIce.blendMode != 8 || firstIce.blendAdditive || firstIce.animation != 1 || firstIce.quadHornAnimSpeed != 50*time.Millisecond {
		t.Fatalf("EF_ICEWALL first component = %+v", firstIce)
	}
	if firstIce.quadHornHeightMin != 2.8 || firstIce.quadHornHeightMax != 3.3 || firstIce.quadHornOffsetXMin != 0.25 || firstIce.quadHornOffsetXMax != 0.75 || firstIce.quadHornOffsetZ != -0.1 || firstIce.quadHornBottomMin != 0.3 || firstIce.quadHornBottomMax != 0.5 || firstIce.quadHornRotateYMin != 1 || firstIce.quadHornRotateYMax != 360 {
		t.Fatalf("EF_ICEWALL robr ranges = %+v", firstIce)
	}
	if ice.components[2].quadHornHeightMin != 2.5 || ice.components[2].quadHornHeightMax != 2.9 {
		t.Fatalf("EF_ICEWALL third height range = %+v", ice.components[2])
	}

	earth, ok := worldEffectSpecForID(effectEarthSpike)
	if !ok || len(earth.components) != 5 || earth.duration != 5*time.Second || earth.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_EARTHSPIKE spec = %+v ok=%t", earth, ok)
	}
	if len(earth.sfx) != 1 || earth.sfx[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("EF_EARTHSPIKE sfx = %v", earth.sfx)
	}
	main := earth.components[0]
	if main.kind != effectComponentQuadHorn || main.textureFile != "effect/stone.bmp" || main.duration != 5*time.Second || main.blendMode != 1 || main.animation != 3 || main.quadHornAnimSpeed != 120*time.Millisecond || !main.quadHornAnimOut {
		t.Fatalf("EF_EARTHSPIKE main = %+v", main)
	}
	if main.quadHornHeightMin != 0.95 || main.quadHornHeightMax != 1.5 || main.quadHornRotateZMin != -8 || main.quadHornRotateZMax != 8 {
		t.Fatalf("EF_EARTHSPIKE main ranges = %+v", main)
	}
	last := earth.components[4]
	if last.quadHornOffsetXMin != 0.5 || last.quadHornOffsetXMax != 0.7 || last.quadHornOffsetYMin != 0 || last.quadHornOffsetYMax != -0.2 || last.quadHornAnimSpeed != 100*time.Millisecond {
		t.Fatalf("EF_EARTHSPIKE last ranges = %+v", last)
	}
}

func TestRobrowserMeteorAndJupitelSpecs(t *testing.T) {
	meteor, ok := worldEffectSpecForID(effectMeteorStorm)
	if !ok || len(meteor.components) != 1 {
		t.Fatalf("EF_METEORSTORM spec = %+v ok=%t", meteor, ok)
	}
	if meteor.cameraShakeDelay != 600*time.Millisecond || meteor.cameraShake != 650*time.Millisecond {
		t.Fatalf("EF_METEORSTORM camera shake = delay %s duration %s", meteor.cameraShakeDelay, meteor.cameraShake)
	}
	if len(meteor.sfx) != 1 || meteor.sfx[0] != "effect\\wizard_meteor.wav" {
		t.Fatalf("EF_METEORSTORM sfx = %v", meteor.sfx)
	}
	component := meteor.components[0]
	if component.kind != effectComponentSTR || component.strFile != "meteor%d" || component.strRandMin != 1 || component.strRandMax != 4 || !component.attachedEntity {
		t.Fatalf("EF_METEORSTORM component = %+v", component)
	}

	jupitel, ok := worldEffectSpecForID(effectJupitelThunder)
	if !ok || len(jupitel.components) != 2 || jupitel.duration != 200*time.Millisecond {
		t.Fatalf("EF_YUFITEL spec = %+v ok=%t", jupitel, ok)
	}
	if len(jupitel.sfx) != 1 || jupitel.sfx[0] != "effect\\hunter_shockwavetrap.wav" {
		t.Fatalf("EF_YUFITEL sfx = %v", jupitel.sfx)
	}
	center, ball := jupitel.components[0], jupitel.components[1]
	if center.kind != effectComponent3D || center.textureFile != "effect/thunder_center.bmp" || center.duration != 200*time.Millisecond || center.sizeStart != effectTableSize(35) || !center.toSrc || !center.blendAdditive || !center.overlay || center.alphaMax != 0.66 {
		t.Fatalf("EF_YUFITEL center = %+v", center)
	}
	if ball.kind != effectComponent3D || len(ball.textureFiles) != 6 || ball.textureFiles[0] != "effect/thunder_ball_a.bmp" || ball.textureFiles[5] != "effect/thunder_ball_f.bmp" || ball.frameDelay != 10*time.Millisecond || ball.sizeStart != effectTableSize(45) || !ball.toSrc || !ball.blendAdditive || !ball.overlay {
		t.Fatalf("EF_YUFITEL ball = %+v", ball)
	}

	hit, ok := worldEffectSpecForID(effectJupitelHit)
	if !ok || len(hit.components) != 2 || hit.duration != 300*time.Millisecond {
		t.Fatalf("EF_YUFITELHIT spec = %+v ok=%t", hit, ok)
	}
	pang, blast := hit.components[0], hit.components[1]
	if pang.kind != effectComponent3D || pang.textureFile != "effect/thunder_pang.bmp" || pang.duration != 100*time.Millisecond || pang.sizeStart != 0 || pang.sizeEnd != effectTableSize(25) || !pang.rotateToTarget || !pang.fadeOut || !pang.overlay || !pang.attachedEntity || !pang.blendAdditive {
		t.Fatalf("EF_YUFITELHIT pang = %+v", pang)
	}
	if blast.kind != effectComponent3D || len(blast.textureFiles) != 5 || blast.textureFiles[0] != "effect/thunder_plazma_blast_a.bmp" || blast.textureFiles[4] != "effect/thunder_ball_f.bmp" || blast.frameDelay != 10*time.Millisecond || blast.duration != 300*time.Millisecond || blast.sizeStart != effectTableSize(75) || !blast.overlay || !blast.attachedEntity || !blast.blendAdditive {
		t.Fatalf("EF_YUFITELHIT blast = %+v", blast)
	}
}

func TestRobrowserActiveEffectsOneHundredToOneFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectRemoveTrap:     "EF_REMOVETRAP",
		effectRepairWeapon:   "EF_REPAIRWEAPON",
		effectCrashEarth:     "EF_CRASHEARTH",
		effectWeaponPerfect:  "EF_PERFECTION",
		effectMaximizePower:  "EF_MAXPOWER",
		effectBlastMine:      "EF_BLASTMINE",
		effectBlastMineBomb:  "EF_BLASTMINEBOMB",
		effectClaymore:       "EF_CLAYMORE",
		effectFreezingTrap:   "EF_FREEZING",
		effectBubble:         "EF_BUBBLE",
		effectGasPush:        "EF_GASPUSH",
		effectSpringTrap:     "EF_SPRINGTRAP",
		effectKyrie:          "EF_KYRIE",
		effectMagnus:         "EF_MAGNUS",
		effectBlitzBeat:      "EF_BLITZBEAT",
		effectWaterBall:      "EF_WATERBALL",
		effectWaterBall2:     "EF_WATERBALL2",
		effectDetecting:      "EF_DETECTING",
		effectCloaking:       "EF_CLOAKING",
		effectSonicBlow:      "EF_SONICBLOW",
		effectSonicBlowHit:   "EF_SONICBLOWHIT",
		effectGrimtooth:      "EF_GRIMTOOTH",
		effectVenomDust:      "EF_VENOMDUST",
		effectPoisonReact:    "EF_POISONREACT",
		effectPoisonReact2:   "EF_POISONREACT2",
		effectOverthrust:     "EF_OVERTHRUST",
		effectVenomSplasher:  "EF_SPLASHER",
		effectTwoHandQuicken: "EF_TWOHANDQUICKEN",
		effectAutoCounter:    "EF_AUTOCOUNTER",
		effectGrimtoothAtk:   "EF_GRIMTOOTHATK",
		effectFreeze:         "EF_FREEZE",
		effectFreezed:        "EF_FREEZED",
		effectIceCrash:       "EF_ICECRASH",
		effectSlowPoison:     "EF_SLOWPOISON",
		effectFirePillarOn:   "EF_FIREPILLARON",
		effectSandman:        "EF_SANDMAN",
		effectRevive:         "EF_REVIVE",
		effectPneuma:         "EF_PNEUMA",
		effectHeavenDrive:    "EF_HEAVENDRIVE",
		effectSonicBlow2:     "EF_SONICBLOW2",
		effectBrandishSpear2: "EF_BRANDISH2",
		effectShockwave:      "EF_SHOCKWAVE",
		effectShockwaveHit:   "EF_SHOCKWAVEHIT",
		effectEarthHit:       "EF_EARTHHIT",
		effectPierceSelf:     "EF_PIERCESELF",
		effectBowlingSelf:    "EF_BOWLINGSELF",
		effectSpearStabSelf:  "EF_SPEARSTABSELF",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsOneHundredToOneFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_BLASTMINEBOMB", effectBlastMineBomb, "blastmine", "effect\\hunter_blastmine.wav", false, false},
		{"EF_CLAYMORE", effectClaymore, "claymore", "effect\\hunter_claymoretrap.wav", false, false},
		{"EF_FREEZING", effectFreezingTrap, "freezing", "effect\\hunter_freezingtrap.wav", false, false},
		{"EF_GASPUSH", effectGasPush, "gaspush", "", false, false},
		{"EF_SPRINGTRAP", effectSpringTrap, "spring", "effect\\hunter_springtrap.wav", false, false},
		{"EF_MAGNUS", effectMagnus, "magnus", "effect\\priest_magnus.wav", false, false},
		{"EF_VENOMDUST", effectVenomDust, "venomdust", "effect\\assasin_poisonreact.wav", false, false},
		{"EF_POISONREACT", effectPoisonReact, "poisonreact_1st", "effect\\assasin_poisonreact.wav", true, false},
		{"EF_POISONREACT2", effectPoisonReact2, "poisonreact", "effect\\assasin_poisonreact.wav", true, false},
		{"EF_SPLASHER", effectVenomSplasher, "venomsplasher", "effect\\assasin_venomsplasher.wav", true, false},
		{"EF_TWOHANDQUICKEN", effectTwoHandQuicken, "twohand", "effect\\knight_twohandquicken.wav", true, true},
		{"EF_AUTOCOUNTER", effectAutoCounter, "autocounter", "effect\\knight_autocounter.wav", true, false},
		{"EF_FREEZE", effectFreeze, "freeze", "", true, false},
		{"EF_FREEZED", effectFreezed, "freezed", "", true, false},
		{"EF_ICECRASH", effectIceCrash, "icecrash", "", true, false},
		{"EF_SLOWPOISON", effectSlowPoison, "slowp", "effect\\priest_slowpoison.wav", false, false},
		{"EF_SANDMAN", effectSandman, "sandman", "effect\\hunter_sandman.wav", false, false},
		{"EF_SONICBLOW2", effectSonicBlow2, "sonicblow", "", true, false},
		{"EF_BRANDISH2", effectBrandishSpear2, "brandish2", "effect\\knight_brandish_spear.wav", true, false},
		{"EF_SHOCKWAVEHIT", effectShockwaveHit, "shockwavehit", "", true, false},
		{"EF_EARTHHIT", effectEarthHit, "earthhit", "", true, false},
		{"EF_PIERCESELF", effectPierceSelf, "pierce", "", true, false},
		{"EF_BOWLINGSELF", effectBowlingSelf, "bowling", "_enemy_hit_normal1.wav", true, true},
		{"EF_SPEARSTABSELF", effectSpearStabSelf, "spearstab", "_enemy_hit_normal1.wav", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_BLASTMINE", effectBlastMine, "effect\\hun_anklesnare.wav"},
		{"EF_BLITZBEAT", effectBlitzBeat, "effect\\hunter_blitzbeat.wav"},
		{"EF_DETECTING", effectDetecting, "effect\\hunter_detecting.wav"},
		{"EF_CLOAKING", effectCloaking, "effect\\assasin_cloaking.wav"},
		{"EF_GRIMTOOTH", effectGrimtooth, "effect\\ef_frostdiver.wav"},
		{"EF_OVERTHRUST", effectOverthrust, "effect\\black_overthrust.wav"},
		{"EF_REVIVE", effectRevive, "effect\\priest_resurrection.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserActiveEffectsOneFiftyToTwoHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectSpearStabSelf:  "EF_SPEARSTABSELF",
		effectSpearBmrSelf:   "EF_SPEARBMRSELF",
		effectHolyLight:      "EF_HOLYHIT",
		effectConcentration:  "EF_CONCENTRATION",
		effectRefineOK:       "EF_REFINEOK",
		effectRefineFail:     "EF_REFINEFAIL",
		effectJobLevelUp:     "EF_JOBLVUP",
		effectRain:           "EF_RAIN",
		effectSnow:           "EF_SNOW",
		effectSakura:         "EF_SAKURA",
		effectBanjjakii:      "EF_BANJJAKII",
		effectMakeBlur:       "EF_MAKEBLUR",
		effectEnergyCoat:     "EF_ENERGYCOAT",
		effectCartRevolution: "EF_CARTREVOLUTION",
		effectVenomDust2:     "EF_VENOMDUST2",
		effectMentalBreak:    "EF_MENTALBREAK",
		effectMagicalAtkHit:  "EF_MAGICALATTHIT",
		effectSuiExplosion:   "EF_SUI_EXPLOSION",
		effectSuicide:        "EF_SUICIDE",
		effectComboAttack1:   "EF_COMBOATTACK1",
		effectComboAttack2:   "EF_COMBOATTACK2",
		effectComboAttack3:   "EF_COMBOATTACK3",
		effectComboAttack4:   "EF_COMBOATTACK4",
		effectComboAttack5:   "EF_COMBOATTACK5",
		effectGuidedAttack:   "EF_GUIDEDATTACK",
		effectPoisonAttack2:  "EF_POISONATTACK",
		effectSilenceAttack:  "EF_SILENCEATTACK",
		effectStunAttack:     "EF_STUNATTACK",
		effectPetrifyAttack:  "EF_PETRIFYATTACK",
		effectSleepAttack:    "EF_SLEEPATTACK",
		effectPong:           "EF_PONG",
		effectLevel99:        "EF_LEVEL99",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsOneFiftyToTwoHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_SPEARBMRSELF", effectSpearBmrSelf, "spearboomerang", "effect\\knight_spear_boomerang.wav", true, true},
		{"EF_HOLYHIT", effectHolyLight, "holyhit", "", true, false},
		{"EF_CONCENTRATION", effectConcentration, "concentration", "effect\\ac_concentration.wav", true, false},
		{"EF_REFINEOK", effectRefineOK, "bs_refinesuccess", "effect\\bs_refinesuccess.wav", true, false},
		{"EF_REFINEFAIL", effectRefineFail, "bs_refinefailed", "effect\\bs_refinefailed.wav", true, false},
		{"EF_ENERGYCOAT", effectEnergyCoat, "energycoat", "", true, false},
		{"EF_CARTREVOLUTION", effectCartRevolution, "cartrevolution", "effect\\ef_magnumbreak.wav", true, false},
		{"EF_MENTALBREAK", effectMentalBreak, "mentalbreak", "", true, false},
		{"EF_MAGICALATTHIT", effectMagicalAtkHit, "magical", "", true, false},
		{"EF_SUICIDE", effectSuicide, "suicide", "", true, false},
		{"EF_COMBOATTACK1", effectComboAttack1, "yunta_1", "", true, false},
		{"EF_COMBOATTACK2", effectComboAttack2, "yunta_2", "", true, false},
		{"EF_COMBOATTACK3", effectComboAttack3, "yunta_3", "", true, false},
		{"EF_COMBOATTACK4", effectComboAttack4, "yunta_4", "", true, false},
		{"EF_COMBOATTACK5", effectComboAttack5, "yunta_5", "", true, false},
		{"EF_GUIDEDATTACK", effectGuidedAttack, "homing", "", true, false},
		{"EF_POISONATTACK", effectPoisonAttack2, "poison", "", true, false},
		{"EF_SILENCEATTACK", effectSilenceAttack, "silence", "", true, false},
		{"EF_STUNATTACK", effectStunAttack, "stun", "", true, false},
		{"EF_PETRIFYATTACK", effectPetrifyAttack, "stonecurse", "", true, false},
		{"EF_SLEEPATTACK", effectSleepAttack, "sleep", "", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserSpecialEffectsOneFiftyToTwoHundredMatchTableRows(t *testing.T) {
	banjjakii, ok := worldEffectSpecForID(effectBanjjakii)
	if !ok || banjjakii.duration != time.Second || len(banjjakii.components) != 1 {
		t.Fatalf("EF_BANJJAKII spec = %+v ok=%t", banjjakii, ok)
	}
	if component := banjjakii.components[0]; component.kind != effectComponentSPR || component.spriteFile != "크리스마스" || component.duration != time.Second || !component.attachedEntity {
		t.Fatalf("EF_BANJJAKII component = %+v", component)
	}

	makeBlur, ok := worldEffectSpecForID(effectMakeBlur)
	if !ok || makeBlur.duration != 2*time.Second || len(makeBlur.components) != 1 {
		t.Fatalf("EF_MAKEBLUR spec = %+v ok=%t", makeBlur, ok)
	}
	if component := makeBlur.components[0]; component.kind != effectComponentFUNC || component.funcName != "MakeBlur" || component.funcAdapter != effectFuncUnknown || component.attachedEntity {
		t.Fatalf("EF_MAKEBLUR component = %+v", component)
	}

	venomDust, ok := worldEffectSpecForID(effectVenomDust2)
	if !ok || venomDust.duration != 100*time.Millisecond || len(venomDust.components) != 1 {
		t.Fatalf("EF_VENOMDUST2 spec = %+v ok=%t", venomDust, ok)
	}
	component := venomDust.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "particle3" || component.duration != 100*time.Millisecond || !component.repeat || !component.spriteRepeat || !component.attachedEntity {
		t.Fatalf("EF_VENOMDUST2 component = %+v", component)
	}
	if component.alphaMax != 1 || component.sizeStart != effectTableSize(80) || component.sizeEnd != effectTableSize(80) || component.posZ != 0 || component.posZEnd != 0.5 {
		t.Fatalf("EF_VENOMDUST2 scalar fields = %+v", component)
	}

	sui, ok := worldEffectSpecForID(effectSuiExplosion)
	if !ok || len(sui.components) != 2 || sui.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_SUI_EXPLOSION spec = %+v ok=%t", sui, ok)
	}
	if len(sui.sfx) != 1 || sui.sfx[0] != "effect\\ef_hit2.wav" {
		t.Fatalf("EF_SUI_EXPLOSION sfx = %v", sui.sfx)
	}
	if str, quake := sui.components[0], sui.components[1]; str.kind != effectComponentSTR || str.strFile != "sui_explosion" || !str.attachedEntity || quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || !quake.attachedEntity {
		t.Fatalf("EF_SUI_EXPLOSION components = %+v %+v", str, quake)
	}

	pong, ok := worldEffectSpecForID(effectPong)
	if !ok || len(pong.components) != 1 {
		t.Fatalf("EF_PONG spec = %+v ok=%t", pong, ok)
	}
	if component := pong.components[0]; component.kind != effectComponentSTR || component.strFile != "pong%d" || component.strRandMin != 1 || component.strRandMax != 3 || component.attachedEntity {
		t.Fatalf("EF_PONG component = %+v", component)
	}

	level99, ok := worldEffectSpecForID(effectLevel99)
	if !ok || level99.duration != 5*time.Minute || len(level99.components) != 1 {
		t.Fatalf("EF_LEVEL99 spec = %+v ok=%t", level99, ok)
	}
	if component := level99.components[0]; component.kind != effectComponentFUNC || component.funcName != "Level99Aura" || component.funcAdapter != effectFuncLevel99Aura || component.textureFile != "effect/ring_blue.tga" || !component.attachedEntity {
		t.Fatalf("EF_LEVEL99 component = %+v", component)
	}
}

func TestRobrowserActiveEffectsTwoHundredToTwoFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectLevel99:       "EF_LEVEL99",
		effectLevel99Ground: "EF_LEVEL99_2",
		effectLevel99Bubble: "EF_LEVEL99_3",
		effectGumgang:       "EF_GUMGANG",
		effectPotionRed:     "EF_POTION1",
		effectPotionOrange:  "EF_POTION2",
		effectPotionYellow:  "EF_POTION3",
		effectPotionWhite:   "EF_POTION4",
		effectPotionBlue:    "EF_POTION5",
		effectPotionGreen:   "EF_POTION6",
		effectFood:          "EF_POTION7",
		effectFoodBlue:      "EF_POTION8",
		effectDarkBreath:    "EF_DARKBREATH",
		effectDefender:      "EF_DEFFENDER",
		effectKeeping:       "EF_KEEPING",
		effectSummonSlave:   "EF_SUMMONSLAVE",
		effectBloodDrain:    "EF_BLOODDRAIN",
		effectEnergyDrain:   "EF_ENERGYDRAIN",
		effectItemFast:      "EF_POTION_CON",
		effectItemFast2:     "EF_POTION_",
		effectItemFast3:     "EF_POTION_BERSERK",
		effectCrusaderDef:   "EF_DEFENDER",
		effectGrandCross:    "EF_GRANDCROSS",
		effectIntimidate:    "EF_INTIMIDATE",
		effectChookgi:       "EF_CHOOKGI",
		effectCloud:         "EF_CLOUD",
		effectCloud2:        "EF_CLOUD2",
		effectLineLink:      "EF_LINELINK",
		effectCloud3:        "EF_CLOUD3",
		effectSpellBreaker:  "EF_SPELLBREAKER",
		effectDispell:       "EF_DISPELL",
		effectBottomVolcano: "EF_BOTTOM_VO",
		effectBottomDeluge:  "EF_BOTTOM_DE",
		effectBottomViolent: "EF_BOTTOM_VI",
		effectBottomLand:    "EF_BOTTOM_LA",
		effectMagicRod:      "EF_MAGICROD",
		effectHolyCross:     "EF_HOLYCROSS",
		effectShieldCharge:  "EF_SHIELDCHARGE",
		effectProvidence:    "EF_PROVIDENCE",
		effectShieldBoomer:  "EF_SHIELDBOOMERANG",
		effectSpearQuicken:  "EF_SPEARQUICKEN",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsTwoHundredToTwoFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_DEFFENDER", effectDefender, "deffender", "", true, false},
		{"EF_KEEPING", effectKeeping, "keeping", "", true, false},
		{"EF_POTION_CON", effectItemFast, "집중", "effect\\ac_concentration.wav", true, false},
		{"EF_POTION_", effectItemFast2, "각성", "effect\\ac_concentration.wav", true, false},
		{"EF_POTION_BERSERK", effectItemFast3, "버서크", "effect\\ac_concentration.wav", true, false},
		{"EF_SPELLBREAKER", effectSpellBreaker, "spell", "effect\\sage_spell breake.wav", true, false},
		{"EF_DISPELL", effectDispell, "디스펠", "", true, false},
		{"EF_MAGICROD", effectMagicRod, "매직로드", "effect\\sage_magic rod.wav", true, false},
		{"EF_HOLYCROSS", effectHolyCross, "holy_cross", "effect\\cru_holy cross.wav", true, false},
		{"EF_SHIELDCHARGE", effectShieldCharge, "shield_charge", "", true, false},
		{"EF_PROVIDENCE", effectProvidence, "providence", "", true, false},
		{"EF_SPEARQUICKEN", effectSpearQuicken, "twohand", "effect\\knight_twohandquicken.wav", true, true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_INTIMIDATE", effectIntimidate, "effect\\rog_intimidate.wav"},
		{"EF_SHIELDBOOMERANG", effectShieldBoomer, "effect\\cru_shield boomerang.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserPotionEffectsTwoHundredRowsAreAttached(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_POTION1", effectPotionRed, "빨간포션", ""},
		{"EF_POTION2", effectPotionOrange, "주홍포션", ""},
		{"EF_POTION3", effectPotionYellow, "노란포션", ""},
		{"EF_POTION4", effectPotionWhite, "하얀포션", ""},
		{"EF_POTION5", effectPotionBlue, "파란포션", "effect\\흡기.wav"},
		{"EF_POTION6", effectPotionGreen, "초록포션", ""},
		{"EF_POTION7", effectFood, "fruit", "_heal_effect.wav"},
		{"EF_POTION8", effectFoodBlue, "fruit_", "effect\\흡기.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserSpecialEffectsTwoHundredToTwoFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		funcName string
		adapter  effectFuncAdapter
		texture  string
	}{
		{"EF_LEVEL99_2", effectLevel99Ground, "GroundAura", effectFuncGroundAura, "effect/pikapika2.bmp"},
		{"EF_LEVEL99_3", effectLevel99Bubble, "Level99Bubble", effectFuncLevel99Bubble, "effect/whitelight.tga"},
		{"EF_CHOOKGI", effectChookgi, "SpiritSphere", effectFuncSpiritSphere, "effect/thunder_center.bmp"},
		{"EF_BOTTOM_LA", effectBottomLand, "LandProtectorGround", effectFuncLandProtectorGround, "effect/aaa copy.bmp"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one FUNC component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != tc.adapter || component.textureFile != tc.texture {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	gumgang, ok := worldEffectSpecForID(effectGumgang)
	if !ok || len(gumgang.components) != 5 || gumgang.duration != 3600*time.Millisecond {
		t.Fatalf("EF_GUMGANG spec = %+v ok=%t", gumgang, ok)
	}
	for i, component := range gumgang.components {
		wantFile := fmt.Sprintf("effect/super%d.bmp", i+1)
		wantDelay := time.Duration(i) * 400 * time.Millisecond
		if component.kind != effectComponent3D || component.textureFile != wantFile || component.duration != 2*time.Second || component.delay != wantDelay || !component.fadeOut || !component.attachedEntity {
			t.Fatalf("EF_GUMGANG component %d = %+v", i, component)
		}
	}

	dark, ok := worldEffectSpecForID(effectDarkBreath)
	if !ok || len(dark.components) != 1 {
		t.Fatalf("EF_DARKBREATH spec = %+v ok=%t", dark, ok)
	}
	if component := dark.components[0]; component.kind != effectComponentSPR || component.spriteFile != "darkbreath" || !component.spriteHead || !component.attachedEntity {
		t.Fatalf("EF_DARKBREATH component = %+v", component)
	}
	slave, ok := worldEffectSpecForID(effectSummonSlave)
	if !ok || len(slave.components) != 1 {
		t.Fatalf("EF_SUMMONSLAVE spec = %+v ok=%t", slave, ok)
	}
	if component := slave.components[0]; component.kind != effectComponentSPR || component.spriteFile != "smoke" || !component.attachedEntity {
		t.Fatalf("EF_SUMMONSLAVE component = %+v", component)
	}

	for _, tc := range []struct {
		name  string
		id    int
		r     uint8
		g     uint8
		b     uint8
		funcs int
	}{
		{"EF_BLOODDRAIN", effectBloodDrain, 255, 102, 102, 0},
		{"EF_ENERGYDRAIN", effectEnergyDrain, 102, 102, 255, 2},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1+tc.funcs {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.spriteFile != "data/sprite/이팩트/particle1" || component.duration != 600*time.Millisecond || component.duplicate != 5 || !component.toSrc || !component.rotateToTarget || component.arc != 3 || component.retreat != 3 {
			t.Fatalf("%s particle = %+v", tc.name, component)
		}
		if component.color.R != tc.r || component.color.G != tc.g || component.color.B != tc.b || component.sizeStart != effectTableSize(150) || component.sizeEnd != effectTableSize(180) || component.posZ != 5 {
			t.Fatalf("%s particle scalar fields = %+v", tc.name, component)
		}
	}

	defender, ok := worldEffectSpecForID(effectCrusaderDef)
	if !ok || len(defender.components) != 1 || defender.duration != 3*time.Second {
		t.Fatalf("EF_DEFENDER spec = %+v ok=%t", defender, ok)
	}
	if component := defender.components[0]; component.kind != effectComponentCylinder || component.textureName != "ring_black" || component.duration != 3*time.Second || component.alphaMax != 0.6 || component.blendMode != 8 || component.bottomSize != 1.5 || component.topSize != 1.5 || component.height != 10 || !component.rotate || !component.fade || !component.attachedEntity {
		t.Fatalf("EF_DEFENDER component = %+v", component)
	}

	grand, ok := worldEffectSpecForID(effectGrandCross)
	if !ok || len(grand.components) != 25 || grand.duration != 2*time.Second {
		t.Fatalf("EF_GRANDCROSS spec = %+v ok=%t", grand, ok)
	}
	if len(grand.sfx) != 1 || grand.sfx[0] != "effect\\cru_grand cross.wav" {
		t.Fatalf("EF_GRANDCROSS sfx = %v", grand.sfx)
	}
	first := grand.components[0]
	if first.kind != effectComponentCylinder || first.textureName != "ring_red" || first.totalCircleSides != 4 || first.circleSides != 4 || first.bottomSize != 0.7 || first.topSize != 0.7 || first.duplicate != 3 || first.duplicateDelay != 500*time.Millisecond || first.alphaMax != 0.1 || first.angleY != 45 {
		t.Fatalf("EF_GRANDCROSS center = %+v", first)
	}
	arc := grand.components[len(grand.components)-1]
	if arc.totalCircleSides != 20 || arc.circleSides != 5 || arc.bottomSize != 3 || arc.topSize != 3 || arc.posX != -3.5 || arc.posY != 3.5 || arc.angleY != -90 {
		t.Fatalf("EF_GRANDCROSS final arc = %+v", arc)
	}

	line, ok := worldEffectSpecForID(effectLineLink)
	if !ok || len(line.components) != 1 || line.duration != 100*time.Millisecond {
		t.Fatalf("EF_LINELINK spec = %+v ok=%t", line, ok)
	}
	if component := line.components[0]; component.kind != effectComponent3D || component.textureFile != "effect/alpha_center.tga" || component.alphaMax != 0.5 || !component.fromSrc || !component.rotateToTarget || !component.rotateWithCamera || component.sizeStartX != effectTableSize(5) || component.sizeStartY != effectTableSize(50) || component.posZ != 1 {
		t.Fatalf("EF_LINELINK component = %+v", component)
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
	}{
		{"EF_BOTTOM_VO", effectBottomVolcano, "ring_red"},
		{"EF_BOTTOM_DE", effectBottomDeluge, "ring_blue"},
		{"EF_BOTTOM_VI", effectBottomViolent, "ring_yellow"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != "PropertyGround" || component.funcAdapter != effectFuncPropertyGround || component.textureName != tc.texture || component.topSize != 3 || component.bottomSize != 1 || component.height != 2 || !component.repeat {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserActiveEffectsTwoFiftyToThreeHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectSpearQuicken:   "EF_SPEARQUICKEN",
		effectDevotion:       "EF_DEVOTION",
		effectReflectShield:  "EF_REFLECTSHIELD",
		effectAbsorbSpirits:  "EF_ABSORBSPIRITS",
		effectSteelBody:      "EF_STEELBODY",
		effectFlameLauncher:  "EF_FLAMELAUNCHER",
		effectFrostWeapon:    "EF_FROSTWEAPON",
		effectLightningLoad:  "EF_LIGHTNINGLOADER",
		effectSeismicWeapon:  "EF_SEISMICWEAPON",
		effectGumgang2:       "EF_GUMGANG2",
		effectTeiHit1:        "EF_TEIHIT1",
		effectGumgang3:       "EF_GUMGANG3",
		effectTanji:          "EF_TANJI",
		effectTeiHit1X:       "EF_TEIHIT1X",
		effectChimto:         "EF_CHIMTO",
		effectStealCoin:      "EF_STEALCOIN",
		effectStripWeapon:    "EF_STRIPWEAPON",
		effectStripShield:    "EF_STRIPSHIELD",
		effectStripArmor:     "EF_STRIPARMOR",
		effectStripHelm:      "EF_STRIPHELM",
		effectChainCombo:     "EF_CHAINCOMBO",
		effectRogueCoin:      "EF_RG_COIN",
		effectBackStab:       "EF_BACKSTAP",
		effectTeiHit3:        "EF_TEIHIT3",
		effectBottomLullaby:  "EF_BOTTOM_LULLABY",
		effectBottomRichKim:  "EF_BOTTOM_RICHMANKIM",
		effectBottomChaos:    "EF_BOTTOM_ETERNALCHAOS",
		effectBottomDrum:     "EF_BOTTOM_DRUMBATTLEFIELD",
		effectBottomNibelung: "EF_BOTTOM_RINGNIBELUNGEN",
		effectBottomRoki:     "EF_BOTTOM_ROKISWEIL",
		effectBottomAbyss:    "EF_BOTTOM_INTOABYSS",
		effectBottomSieg:     "EF_BOTTOM_SIEGFRIED",
		effectBottomWhistle:  "EF_BOTTOM_WHISTLE",
		effectBottomSinX:     "EF_BOTTOM_ASSASSINCROSS",
		effectBottomBragi:    "EF_BOTTOM_POEMBRAGI",
		effectBottomApple:    "EF_BOTTOM_APPLEIDUN",
		effectBottomHumming:  "EF_BOTTOM_HUMMING",
		effectBottomForget:   "EF_BOTTOM_DONTFORGETME",
		effectBottomFortune:  "EF_BOTTOM_FORTUNEKISS",
		effectBottomService:  "EF_BOTTOM_SERVICEFORYOU",
		effectTalkFrostJoke:  "EF_TALK_FROSTJOKE",
		effectTalkScream:     "EF_TALK_SCREAM",
		effectPokJuk:         "EF_POKJUK",
		effectThrowItem:      "EF_THROWITEM",
		effectChemicalProt:   "EF_CHEMICALPROTECTION",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}

	for id, name := range map[int]string{
		effectBottomDissonance: "EF_BOTTOM_DISSONANCE",
		effectBottomUglyDance:  "EF_BOTTOM_UGLYDANCE",
	} {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsThreeHundredToThreeFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectChemicalProt:  "EF_CHEMICALPROTECTION",
		effectDemonstration: "EF_DEMONSTRATION",
		effectChemical2:     "EF_CHEMICAL2",
		effectTeleportation: "EF_TELEPORTATION2",
		effectPharmacyOK:    "EF_PHARMACY_OK",
		effectPharmacyFail:  "EF_PHARMACY_FAIL",
		effectThrowItem3:    "EF_THROWITEM3",
		effectFirstAid:      "EF_FIRSTAID",
		effectLoud:          "EF_LOUD",
		effectHeal:          "EF_HEAL",
		effectHeal2:         "EF_HEAL2",
		effectExit2:         "EF_EXIT2",
		effectSafetyWall:    "EF_GLASSWALL2",
		effectReadyPortal:   "EF_READYPORTAL2",
		effectPortal:        "EF_PORTAL2",
		effectBottomMagnus:  "EF_BOTTOM_MAG",
		effectBottomSanc:    "EF_BOTTOM_SANC",
		effectHealOffensive: "EF_HEAL3",
		effectWarpZone2:     "EF_WARPZONE2",
		effectHeal4:         "EF_HEAL4",
		effectBeginAsura:    "EF_BEGINASURA",
		effectTripleAttack:  "EF_TRIPLEATTACK",
		effectHPTime:        "EF_HPTIME",
		effectSPTime:        "EF_SPTIME",
		effectMaple:         "EF_MAPLE",
		effectBlind:         "EF_BLIND",
		effectPoisonStatus:  "EF_POISON",
		effectGuard:         "EF_GUARD",
		effectJobLvUp50:     "EF_JOBLVUP50",
		effectMagnum2:       "EF_MAGNUM2",
		effectEntry2:        "EF_ENTRY2",
		effectColorPaper:    "EF_COLORPAPER",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsThreeFiftyToFourHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectSoulBreaker:       "EF_SOULBREAKER",
		effectLevel99Aura1:      "EF_LEVEL99_4",
		effectFoodChocolate:     "EF_VALLENTINE",
		effectPressure:          "EF_PRESSURE",
		effectBash3D:            "EF_BASH3D",
		effectAuraBlade:         "EF_AURABLADE",
		effectRedBody:           "EF_REDBODY",
		effectLKConcentration:   "EF_LKCONCENTRATION",
		effectBottomGospel:      "EF_BOTTOM_GOSPEL",
		effectBaseLevelUp:       "EF_ANGEL",
		effectDeath:             "EF_DEVIL",
		effectDragonSmoke:       "EF_DRAGONSMOKE",
		effectBottomBasilica:    "EF_BOTTOM_BASILICA",
		effectHitLine2:          "EF_HITLINE2",
		effectBash3D2:           "EF_BASH3D2",
		effectEnergyDrain2:      "EF_ENERGYDRAIN2",
		effectTransBlueBody:     "EF_TRANSBLUEBODY",
		effectMagicCrasher:      "EF_MAGICCRASHER",
		effectLightBlade:        "EF_LIGHTBLADE",
		effectEnergyDrain3:      "EF_ENERGYDRAIN3",
		effectLineLink2:         "EF_LINELINK2",
		effectTrueSight:         "EF_TRUESIGHT",
		effectFalconAssault:     "EF_FALCONASSAULT",
		effectTripleAttack2:     "EF_TRIPLEATTACK2",
		effectPortal4:           "EF_PORTAL4",
		effectMeltdown:          "EF_MELTDOWN",
		effectCartBoost:         "EF_CARTBOOST",
		effectRejectSword:       "EF_REJECTSWORD",
		effectTripleAttack3:     "EF_TRIPLEATTACK3",
		effectMoonlit:           "EF_SPHEREWIND2",
		effectLevel99AuraMid:    "EF_LEVEL99_5",
		effectLevel99AuraBottom: "EF_LEVEL99_6",
		effectBash3D3:           "EF_BASH3D3",
		effectBash3D4:           "EF_BASH3D4",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFourHundredToFourFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectPortal5:       "EF_PORTAL5",
		effectMagicCrasher2: "EF_MAGICCRASHER2",
		effectBottomSpider:  "EF_BOTTOM_SPIDER",
		effectSoulBurn:      "EF_SOULBURN",
		effectSoulChange:    "EF_SOULCHANGE",
		effectSoulBreaker2:  "EF_SOULBREAKER2",
		effectBabyBody:      "EF_BABYBODY",
		effectBabyBody2:     "EF_BABYBODY2",
		effectGiantBody:     "EF_GIANTBODY",
		effectGiantBody2:    "EF_GIANTBODY2",
		effectQuakeBody:     "EF_QUAKEBODY",
		effectAssumptio2:    "EF_ASSUMPTIO2",
		effectStopEffect:    "EF_STOPEFFECT",
		effectJumpBody:      "EF_JUMPBODY",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFourFiftyToFiveHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectDarkGrandCross: "EF_GRANDCROSS2",
		effectDarkSoulStrike: "EF_SOULSTRIKE2",
		effectDarkJupitelHit: "EF_YUFITEL2",
		effectNPCStop:        "EF_NPC_STOP",
		effectDarkCasting:    "EF_DARKCASTING",
		effectNPCPowerUp:     "EF_AGIUP",
		effectJumpKick:       "EF_JUMPKICK",
		effectBeginAsura1:    "EF_BEGINASURA1",
		effectBeginAsura2:    "EF_BEGINASURA2",
		effectBeginAsura3:    "EF_BEGINASURA3",
		effectBeginAsura4:    "EF_BEGINASURA4",
		effectBeginAsura5:    "EF_BEGINASURA5",
		effectBeginAsura6:    "EF_BEGINASURA6",
		effectBeginAsura7:    "EF_BEGINASURA7",
		effectMochi:          "EF_MOCHI",
		effectRamadan:        "EF_LAMADAN",
		effectEDP:            "EF_EDP",
		effectPreserve:       "EF_GUARD2",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiveHundredToFiveFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectCastSpin:      "EF_CASTSPIN",
		effectChookgi2:      "EF_CHOOKGI2",
		effectMapae:         "EF_MAPAE",
		effectItemPokJuk:    "EF_ITEMPOKJUK",
		effectValentine05:   "EF_05VAL",
		effectBeginAsura11:  "EF_BEGINASURA11",
		effectChemical2Dash: "EF_CHEMICAL2DASH",
		effectGroundSample:  "EF_GROUNDSAMPLE",
		effectCloud4:        "EF_CLOUD4",
		effectCloud5:        "EF_CLOUD5",
		effectBottomHermode: "EF_BOTTOM_HERMODE",
		effectItemFastDown:  "EF_ITEMFAST",
		effectTarotCard1:    "EF_TAROTCARD1",
		effectTarotCard2:    "EF_TAROTCARD2",
		effectTarotCard3:    "EF_TAROTCARD3",
		effectTarotCard4:    "EF_TAROTCARD4",
		effectTarotCard5:    "EF_TAROTCARD5",
		effectTarotCard6:    "EF_TAROTCARD6",
		effectTarotCard7:    "EF_TAROTCARD7",
		effectTarotCard8:    "EF_TAROTCARD8",
		effectTarotCard9:    "EF_TAROTCARD9",
		effectTarotCard10:   "EF_TAROTCARD10",
		effectTarotCard11:   "EF_TAROTCARD11",
		effectTarotCard12:   "EF_TAROTCARD12",
		effectTarotCard13:   "EF_TAROTCARD13",
		effectTarotCard14:   "EF_TAROTCARD14",
		effectAcidDemon:     "EF_ACIDDEMON",
		effectHated:         "EF_HATED",
		effectStin:          "EF_STIN",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiveFiftyToSixHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectStin2:       "EF_STIN2",
		effectStin3:       "EF_STIN3",
		effectScreenQuake: "EF_SCREEN_QUAKE",
		effectHfliMoon1:   "EF_HFLIMOON1",
		effectHfliMoon2:   "EF_HFLIMOON2",
		effectHfliMoon3:   "EF_HFLIMOON3",
		effectHoUp:        "EF_HO_UP",
		effectHamiDefence: "EF_HAMIDEFENCE",
		effectHamiCastle:  "EF_HAMICASTLE",
		effectHamiBlood:   "EF_HAMIBLOOD",
		effectItemThunder: "EF_ITEM_THUNDER",
		effectItemCloud:   "EF_ITEM_CLOUD",
		effectItemCurse:   "EF_ITEM_CURSE",
		effectItemZZZ:     "EF_ITEM_ZZZ",
		effectItemRain:    "EF_ITEM_RAIN",
		effectM01:         "EF_M01",
		effectM02:         "EF_M02",
		effectM03:         "EF_M03",
		effectM04:         "EF_M04",
		effectM05:         "EF_M05",
		effectM06:         "EF_M06",
		effectM07:         "EF_M07",
		effectKaizel:      "EF_KAIZEL",
		effectCloud6:      "EF_CLOUD6",
		effectStatFoodSTR: "EF_FOOD01",
		effectStatFoodINT: "EF_FOOD02",
		effectStatFoodVIT: "EF_FOOD03",
		effectStatFoodAGI: "EF_FOOD04",
		effectStatFoodDEX: "EF_FOOD05",
		effectStatFoodLUK: "EF_FOOD06",
		effectThrowItem6:  "EF_THROWITEM6",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSixHundredToSixFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectThrowItem6:    "EF_THROWITEM6",
		effectFireHit2:      "EF_FIREHIT2",
		effectNPCStop2:      "EF_NPC_STOP2",
		effectFVoice:        "EF_FVOICE",
		effectWink:          "EF_WINK",
		effectCookingOK:     "EF_COOKING_OK",
		effectCookingFail:   "EF_COOKING_FAIL",
		effectHapgyeok:      "EF_HAPGYEOK",
		effectThrowItem7:    "EF_THROWITEM7",
		effectThrowItem8:    "EF_THROWITEM8",
		effectThrowItem9:    "EF_THROWITEM9",
		effectThrowItem10:   "EF_THROWITEM10",
		effectKouenka:       "EF_KOUENKA",
		effectHyousensou:    "EF_HYOUSENSOU",
		effectStin4:         "EF_STIN4",
		effectThunderStorm2: "EF_THUNDERSTORM2",
		effectRGCoin3:       "EF_RG_COIN3",
		effectBash3D5:       "EF_BASH3D5",
		effectChookgi3:      "EF_CHOOKGI3",
		effectKirikage:      "EF_KIRIKAGE",
		effectTatami:        "EF_TATAMI",
		effectKasumikiri:    "EF_KASUMIKIRI",
		effectIssen:         "EF_ISSEN",
		effectKaen:          "EF_KAEN",
		effectBaku:          "EF_BAKU",
		effectHyousyouraku:  "EF_HYOUSYOURAKU",
		effectDesperado:     "EF_DESPERADO",
		effectLightningS:    "EF_LIGHTNING_S",
		effectBlindS:        "EF_BLIND_S",
		effectPoisonS:       "EF_POISON_S",
		effectFreezingS:     "EF_FREEZING_S",
		effectFlareS:        "EF_FLARE_S",
		effectRapidShower:   "EF_RAPIDSHOWER",
		effectMagicalBullet: "EF_MAGICALBULLET",
		effectSpreadAttack:  "EF_SPREADATTACK",
		effectTrackCasting:  "EF_TRACKCASTING",
		effectTracking:      "EF_TRACKING",
		effectTripleAction:  "EF_TRIPLEACTION",
		effectBullseye:      "EF_BULLSEYE",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSixFiftyToSevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectNPCEarthquake:  "EF_NPC_EARTHQUAKE",
		effectDragonFear:     "EF_DRAGONFEAR",
		effectWideBleeding:   "EF_BLEEDING",
		effectWideConfuse:    "EF_WIDECONFUSE",
		effectBottomRunner:   "EF_BOTTOM_RUNNER",
		effectBottomTransfer: "EF_BOTTOM_TRANSFER",
		effectBottomEvilLand: "EF_BOTTOM_EVILLAND",
		effectGuard3:         "EF_GUARD3",
		effectCriticalWound:  "EF_CRITICALWOUND",
		effectFirecracker2:   "EF_POK_LOVE",
		effectFirecracker3:   "EF_POK_WHITE",
		effectFirecracker4:   "EF_POK_VALEN",
		effectFirecracker5:   "EF_POK_BIRTH",
		effectFirecracker6:   "EF_POK_CHRISTMAS",
		effectCloud7:         "EF_CLOUD7",
		effectCloud8:         "EF_CLOUD8",
		effectFlowerLeaf:     "EF_FLOWERLEAF",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSevenHundredToSevenFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectItem315:          "EF_ITEM315",
		effectItem316:          "EF_ITEM316",
		effectItem317:          "EF_ITEM317",
		effectStormMin:         "EF_STORM_MIN",
		effectFirecracker7:     "EF_POK_JAP",
		effectBottomBlue:       "EF_BOTTOM_BLUE",
		effectBottomBlue2:      "EF_BOTTOM_BLUE2",
		effectChristmasCarol:   "EF_WEWISH",
		effectFirePillarOn2:    "EF_FIREPILLARON2",
		effectForestLight5:     "EF_FORESTLIGHT5",
		effectAdoramus:         "EF_ADO_STR",
		effectIgnitionBreak:    "EF_IGN_STR",
		effectFrostMisty:       "EF_FROSTMYSTY",
		effectCrimsonRock:      "EF_CRIMSON_STR",
		effectHellInferno:      "EF_HELL_STR",
		effectMarshOfAbyss:     "EF_SPR_MASH",
		effectDragonHowling:    "EF_DHOWL_STR",
		effectEarthWall:        "EF_EARTHWALL",
		effectChainLightning:   "EF_CHAINL_STR",
		effectAimedBolt:        "EF_AIMED_STR",
		effectArrowStorm:       "EF_ARROWSTORM_STR",
		effectLaulamus:         "EF_LAULAMUS_STR",
		effectLauagnus:         "EF_LAUAGNUS_STR",
		effectMillenniumShield: "EF_MILSHIELD_STR",
		effectConcentration2:   "EF_CONCENTRATION2",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSevenFiftyToEightHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectGlassWall3:     "EF_GLASSWALL3",
		effectBerserkPotion2: "EF_POTION_BERSERK2",
		effectRolling1:       "EF_ROLLING1",
		effectRolling2:       "EF_ROLLING2",
		effectRolling3:       "EF_ROLLING3",
		effectRolling4:       "EF_ROLLING4",
		effectRolling5:       "EF_ROLLING5",
		effectRolling6:       "EF_ROLLING6",
		effectRolling7:       "EF_ROLLING7",
		effectRolling8:       "EF_ROLLING8",
		effectRolling9:       "EF_ROLLING9",
		effectRolling10:      "EF_ROLLING10",
		effectCastSpin2:      "EF_CASTSPIN2",
		effectCrashAxe:       "EF_CRASHAXE",
		effectStasis:         "EF_STASIS",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsEightHundredToEightFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectBottomBasilica2:  "EF_BOTTOM_BASILICA2",
		effectRecognized:       "EF_RECOGNIZED",
		effectTetra:            "EF_TETRA",
		effectTetraCasting:     "EF_TETRACASTING",
		effectStretch:          "EF_STRETCH",
		effectEnervation:       "EF_ENERVATION",
		effectEnervation2:      "EF_ENERVATION2",
		effectEnervation3:      "EF_ENERVATION3",
		effectEnervation4:      "EF_ENERVATION4",
		effectEnervation5:      "EF_ENERVATION5",
		effectEnervation6:      "EF_ENERVATION6",
		effectBottomManhole:    "EF_BOTTOM_MANHOLE",
		effectManhole:          "EF_MANHOLE",
		effectForestLight6:     "EF_FORESTLIGHT6",
		effectBottomAni:        "EF_BOTTOM_ANI",
		effectBottomMaelstrom:  "EF_BOTTOM_MAELSTROM",
		effectBottomBloodyLust: "EF_BOTTOM_BLOODYLUST",
		effectHealN:            "EF_HEAL_N",
		effectChookgiN:         "EF_CHOOKGI_N",
		effectDance1:           "EF_DANCE1",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsEightFiftyToNineHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectBotReverb:    "EF_BOT_REVERB",
		effectRainParticle: "EF_RAIN_PARTICLE",
		effectChemicalV2:   "EF_CHEMICAL_V2",
		effectBotReverb2:   "EF_BOT_REVERB2",
		effectCirclePower2: "EF_CIRCLEPOWER2",
		effectSecra2:       "EF_SECRA2",
		effectSprPlant2:    "EF_SPR_PLANT2",
		effectSprPlant3:    "EF_SPR_PLANT3",
		effectSprPlant4:    "EF_SPR_PLANT4",
		effectSprPlant5:    "EF_SPR_PLANT5",
		effectSprPlant6:    "EF_SPR_PLANT6",
		effectSprPlant7:    "EF_SPR_PLANT7",
		effectSprPlant8:    "EF_SPR_PLANT8",
		effectHeartAsura:   "EF_HEARTASURA",
		effectGlassWall4:   "EF_GLASSWALL4",
		effectBash3D6:      "EF_BASH3D6",
		effectElectric4:    "EF_ELECTRIC4",
		effectTeiHit1T:     "EF_TEIHIT1T",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsNineHundredToNineFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectPressure2:    "EF_PRESSURE2",
		effectPrimeCharge2: "EF_PRIMECHARGE2",
		effectPrimeCharge3: "EF_PRIMECHARGE3",
		effectPrimeCharge4: "EF_PRIMECHARGE4",
		effectFireWall2:    "EF_FIREWALL2",
		effectSprPlant10:   "EF_SPR_PLANT10",
		effectShockwave2:   "EF_SHOCKWAVE2",
		effectColdThrow2:   "EF_COLDTHROW2",
		effectDemonicFire4: "EF_DEMONICFIRE4",
		effectPressure3:    "EF_PRESSURE3",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsNineFiftyToOneThousandHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectPoisonMist:     "EF_POISON_MIST",
		effectEraserCutter:   "EF_ERASER_CUTTER",
		effectLavaSlide:      "EF_LAVA_SLIDE",
		effectSonicClaw:      "EF_SONIC_CLAW",
		effectTinderBreaker:  "EF_TINDER_BREAKER",
		effectMidnightFrenzy: "EF_MIDNIGHT_FRENZY",
		effectVolcanicAsh:    "EF_VOLCANIC_ASH",
		effectRWC2011:        "EF_2011RWC",
		effectRWC2011Two:     "EF_2011RWC2",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsOneThousandToTenFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectRunMakeOK:        "EF_RUN_MAKE_OK",
		effectRunMakeFailure:   "EF_RUN_MAKE_FAILURE",
		effectMIResultMakeOK:   "EF_MIRESULT_MAKE_OK",
		effectMIResultMakeFail: "EF_MIRESULT_MAKE_FAIL",
		effectAllRayProtect:    "EF_ALL_RAY_OF_PROTECTION",
		effectVenomFog:         "EF_VENOMFOG",
		effectDustStorm:        "EF_DUSTSTORM",
		effectDanceBladeAtk:    "EF_DANCE_BLADE_ATK",
		effectInvincibleOff2:   "EF_INVINCIBLEOFF2",
		effectDeathSummon:      "EF_DEATHSUMMON",
		effectGCDarkCrow:       "EF_GC_DARKCROW",
		effectAllFullThrottle:  "EF_ALL_FULL_THROTTLE",
		effectSRFlashCombo:     "EF_SR_FLASHCOMBO",
		effectRKLuxAnima:       "EF_RK_LUXANIMA",
		effectSOElemShield:     "EF_SO_ELEMENTAL_SHIELD",
		effectABOffertorium:    "EF_AB_OFFERTORIUM",
		effectWLTelekinesis:    "EF_WL_TELEKINESIS_INTENSE",
		effectGNIllusionDoping: "EF_GN_ILLUSIONDOPING",
		effectNCMagmaEruption:  "EF_NC_MAGMA_ERUPTION",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsTenFiftyToElevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectNPCChill:        "EF_NPC_CHILL",
		effectOffertoriumRing: "EF_AB_OFFERTORIUM_RING",
		effectHammerOfGod:     "EF_HAMMER_OF_GOD",
		effectAchComplete:     "EF_ACH_COMPLETE",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsPostElevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectBodyColor:        "EffectBodyColor",
		effectBakuretsuHadou:   "EF_BAKURETSU_HADOU",
		dropEffectPink:         "DROPEFFECT_PINK",
		dropEffectYellow:       "DROPEFFECT_YELLOW",
		dropEffectPurple:       "DROPEFFECT_PURPLE",
		effectDigitalSpace:     "EF_DIGITAL_SPACE",
		dropEffectBlue:         "DROPEFFECT_BLUE",
		dropEffectGreen:        "DROPEFFECT_GREEN",
		dropEffectRed:          "DROPEFFECT_RED",
		effectNewSuccess:       "EF_NEW_SUCCESS",
		effectNewFailure:       "EF_NEW_FAILURE",
		effectNewIntro:         "EF_NEW_INTRO",
		effectEnchantYellow:    "EF_UI_ENCHANT_INTRO_YELLOW",
		effectEnchantSuccess:   "EF_UI_ENCHANT_SUCCESS",
		effectEnchantFail:      "EF_UI_ENCHANT_FAIL",
		effectEnchantBlue:      "EF_UI_ENCHANT_INTRO_BLUE",
		effectEnchantUpSuccess: "EF_UI_ENCHANT_UP_SUCCESS",
		effectEnchantUpFail:    "EF_UI_ENCHANT_UP_FAIL",
		effectEnchantGreen:     "EF_UI_ENCHANT_INTRO_GREEN",
		effectEnchantResetOK:   "EF_UI_ENCHANT_RESET_SUCCESS",
		effectEnchantResetFail: "EF_UI_ENCHANT_RESET_FAIL",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsTwoFiftyToThreeHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_DEVOTION", effectDevotion, "devotion", "", true, false},
		{"EF_FLAMELAUNCHER", effectFlameLauncher, "enc_fire", "_enemy_hit_wind1.wav", true, false},
		{"EF_FROSTWEAPON", effectFrostWeapon, "enc_ice", "_enemy_hit_wind1.wav", true, false},
		{"EF_LIGHTNINGLOADER", effectLightningLoad, "enc_wind", "effect\\_enemy_hit_wind1.wav", true, false},
		{"EF_SEISMICWEAPON", effectSeismicWeapon, "enc_earth", "_enemy_hit_wind1.wav", true, false},
		{"EF_STEALCOIN", effectStealCoin, "steal_coin", "", true, false},
		{"EF_STRIPWEAPON", effectStripWeapon, "strip_weapon", "effect\\t_벗김.wav", true, false},
		{"EF_STRIPSHIELD", effectStripShield, "strip_shield", "effect\\t_벗김.wav", true, false},
		{"EF_STRIPARMOR", effectStripArmor, "strip_armor", "effect\\t_벗김.wav", true, false},
		{"EF_STRIPHELM", effectStripHelm, "strip_helm", "effect\\t_벗김.wav", true, false},
		{"EF_CHAINCOMBO", effectChainCombo, "연환", "effect\\mon_연환.wav", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_STEELBODY", effectSteelBody, "effect\\mon_금강불괴.wav"},
		{"EF_CHIMTO", effectChimto, "effect\\mon_침투경.wav"},
		{"EF_BACKSTAP", effectBackStab, "effect\\rog_back stap.wav"},
		{"EF_BOTTOM_LULLABY", effectBottomLullaby, "effect\\자장가.wav"},
		{"EF_BOTTOM_RICHMANKIM", effectBottomRichKim, "effect\\김서방돈.wav"},
		{"EF_BOTTOM_ETERNALCHAOS", effectBottomChaos, "effect\\영원의 혼돈.wav"},
		{"EF_BOTTOM_DRUMBATTLEFIELD", effectBottomDrum, "effect\\전장의.wav"},
		{"EF_BOTTOM_RINGNIBELUNGEN", effectBottomNibelung, "effect\\니벨룽겐의 반지.wav"},
		{"EF_BOTTOM_ROKISWEIL", effectBottomRoki, "effect\\로키.wav"},
		{"EF_BOTTOM_INTOABYSS", effectBottomAbyss, "effect\\심연속으로.wav"},
		{"EF_BOTTOM_SIEGFRIED", effectBottomSieg, "effect\\불사신.wav"},
		{"EF_BOTTOM_WHISTLE", effectBottomWhistle, "effect\\달빛세레나데.wav"},
		{"EF_BOTTOM_ASSASSINCROSS", effectBottomSinX, "effect\\석양의 어쌔신.wav"},
		{"EF_BOTTOM_POEMBRAGI", effectBottomBragi, "effect\\브라기의 시.wav"},
		{"EF_BOTTOM_APPLEIDUN", effectBottomApple, "effect\\이둔의 사과.wav"},
		{"EF_BOTTOM_HUMMING", effectBottomHumming, "effect\\흥얼거림.wav"},
		{"EF_BOTTOM_DONTFORGETME", effectBottomForget, "effect\\나를잊지말아요.wav"},
		{"EF_BOTTOM_FORTUNEKISS", effectBottomFortune, "effect\\행운의.wav"},
		{"EF_BOTTOM_SERVICEFORYOU", effectBottomService, "effect\\당신을 위한 서비스.wav"},
		{"EF_CHEMICALPROTECTION", effectChemicalProt, "apocalips_attack.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserSpecialEffectsTwoFiftyToThreeHundredMatchTableRows(t *testing.T) {
	reflectShield, ok := worldEffectSpecForID(effectReflectShield)
	if !ok || len(reflectShield.components) != 1 || reflectShield.duration != 3*time.Second {
		t.Fatalf("EF_REFLECTSHIELD spec = %+v ok=%t", reflectShield, ok)
	}
	if component := reflectShield.components[0]; component.kind != effectComponentCylinder || component.textureName != "ring_yellow" || component.duration != 3*time.Second || component.alphaMax != 0.6 || component.animation != 1 || component.blendMode != 8 || component.bottomSize != 1.5 || component.topSize != 1.5 || component.height != 10 || !component.rotate || !component.fade || !component.attachedEntity {
		t.Fatalf("EF_REFLECTSHIELD component = %+v", component)
	}

	absorb, ok := worldEffectSpecForID(effectAbsorbSpirits)
	if !ok || len(absorb.components) != 6 || absorb.duration != 1890*time.Millisecond {
		t.Fatalf("EF_ABSORBSPIRITS spec = %+v ok=%t", absorb, ok)
	}
	if len(absorb.sfx) != 1 || absorb.sfx[0] != "effect\\흡기.wav" {
		t.Fatalf("EF_ABSORBSPIRITS sfx = %v", absorb.sfx)
	}
	first, second, third := absorb.components[0], absorb.components[1], absorb.components[2]
	for i, component := range []worldEffectComponent{first, second, third} {
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond || component.alphaMax != 0.3 || component.animation != 1 || component.blendMode != 2 || !component.blendAdditive || !component.fade || !component.rotate || !component.attachedEntity {
			t.Fatalf("EF_ABSORBSPIRITS cylinder %d = %+v", i, component)
		}
		if component.color.R != 77 || component.color.G != 77 || component.color.B != 255 {
			t.Fatalf("EF_ABSORBSPIRITS cylinder %d color = %+v", i, component.color)
		}
	}
	if first.bottomSize != 1.1 || first.topSize != 1.1 || first.height != 15 || second.bottomSize != 1 || second.topSize != 1 || second.height != 13 || third.bottomSize != 1.1 || third.topSize != 3 || third.height != 2 {
		t.Fatalf("EF_ABSORBSPIRITS cylinder sizes = %+v %+v %+v", first, second, third)
	}
	sparkA, sparkB, sparkC := absorb.components[3], absorb.components[4], absorb.components[5]
	if sparkA.kind != effectComponent3D || sparkA.textureFile != "effect/pok3.tga" || sparkA.duration != 1500*time.Millisecond || sparkA.duplicate != 4 || sparkA.duplicateDelay != 10*time.Millisecond || sparkA.posXRand != 1.2 || sparkA.posYRand != 1.2 || sparkA.posZEndRand != 1 || sparkA.posZEndMiddle != 8 || !sparkA.sparkling || sparkA.sparkNumber != 2 {
		t.Fatalf("EF_ABSORBSPIRITS first particle = %+v", sparkA)
	}
	if sparkB.duration != 1300*time.Millisecond || sparkB.delay != 400*time.Millisecond || sparkB.duplicate != 20 || sparkB.posXRand != 1.5 || sparkB.posYRand != 1.5 || sparkB.posZEndRand != 3 || sparkB.posZEndMiddle != 6 || !sparkB.sparkling || sparkB.sparkNumber != 2 {
		t.Fatalf("EF_ABSORBSPIRITS second particle = %+v", sparkB)
	}
	if sparkC.duration != 1100*time.Millisecond || sparkC.delay != 200*time.Millisecond || sparkC.duplicate != 10 || sparkC.duplicateDelay != 50*time.Millisecond || sparkC.posXRand != 1 || sparkC.posYRand != 1 || sparkC.posZEnd != 6 || sparkC.posZStartRand != 1 || sparkC.sparkling {
		t.Fatalf("EF_ABSORBSPIRITS third particle = %+v", sparkC)
	}

	for _, tc := range []struct {
		name     string
		id       int
		duration time.Duration
		alpha    float64
		bottom   float64
		top      float64
		wav      string
	}{
		{"EF_GUMGANG2", effectGumgang2, 1500 * time.Millisecond, 0.5, 2, 5, "effect\\mon_폭기.wav"},
		{"EF_GUMGANG3", effectGumgang3, 1000 * time.Millisecond, 0.3, 3, 6, ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != tc.duration+300*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentCylinder || component.textureName != "ring_yellow" || component.duration != tc.duration || component.alphaMax != tc.alpha || component.animation != 4 || component.blendMode != 8 || component.blendAdditive || component.duplicate != 4 || component.duplicateDelay != 100*time.Millisecond || component.bottomSize != tc.bottom || component.topSize != tc.top || component.height != 2 || !component.fade || !component.rotate || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		texture   string
		wav       string
		duplicate int
		delay     time.Duration
		blueRaid  bool
	}{
		{"EF_TEIHIT1", effectTeiHit1, "effect/alpha_center.tga", "effect\\mon_폭기.wav", 12, 250 * time.Millisecond, false},
		{"EF_TEIHIT1X", effectTeiHit1X, "effect/lens1.tga", "effect\\mon_아수라 패황권.wav", 24, 100 * time.Millisecond, false},
		{"EF_TEIHIT3", effectTeiHit3, "effect/lens1.tga", "", 20, 100 * time.Millisecond, true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 550*time.Millisecond+tc.delay {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != tc.texture || component.duration != 550*time.Millisecond || component.delay != tc.delay || component.duplicate != tc.duplicate || component.alphaMax != 0.8 || !component.fadeIn || !component.fadeOut || component.posXEndRand != 40 || component.posYEndRand != 40 || component.sizeStartX != effectTableSize(10) || component.sizeStartY != effectTableSize(150) || component.sizeEndX != effectTableSize(10) || component.sizeEndY != effectTableSize(150) || component.blendMode != 2 || !component.blendAdditive || !component.attachedEntity || !component.overlay || !component.rotateToTarget || !component.rotateWithCamera {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.blueRaid {
			if component.color.R != 26 || component.color.G != 26 || component.color.B != 255 {
				t.Fatalf("%s color = %+v", tc.name, component.color)
			}
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	tanji, ok := worldEffectSpecForID(effectTanji)
	if !ok || len(tanji.components) != 1 || tanji.duration != 150*time.Millisecond {
		t.Fatalf("EF_TANJI spec = %+v ok=%t", tanji, ok)
	}
	if len(tanji.sfx) != 1 || tanji.sfx[0] != "effect\\mon_탄지신통.wav" {
		t.Fatalf("EF_TANJI sfx = %v", tanji.sfx)
	}
	if component := tanji.components[0]; component.kind != effectComponent3D || component.textureFile != "effect/blue_ivy.bmp" || component.duration != 150*time.Millisecond || component.alphaMax != 1 || component.blendMode != 2 || !component.blendAdditive || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || component.angleStart != 90 || component.angleEnd != 90 || component.posZ != 1 || component.sizeStart != effectTableSize(50) || !component.attachedEntity {
		t.Fatalf("EF_TANJI component = %+v", component)
	}

	coin, ok := worldEffectSpecForID(effectRogueCoin)
	if !ok || len(coin.components) != 1 || coin.duration != 2950*time.Millisecond {
		t.Fatalf("EF_RG_COIN spec = %+v ok=%t", coin, ok)
	}
	if len(coin.sfx) != 1 || coin.sfx[0] != "effect\\rog_steal coin.wav" {
		t.Fatalf("EF_RG_COIN sfx = %v", coin.sfx)
	}
	if component := coin.components[0]; component.kind != effectComponent2D || component.textureFile != "effect/coin_a.bmp" || component.duration != 1500*time.Millisecond || component.duplicate != 30 || component.duplicateDelay != 50*time.Millisecond || component.alphaMax != 0.8 || !component.fadeOut || component.posXEndRand != 10 || component.posYEndRand != 10 || component.posZ != 2 || component.sizeStart != effectTableSize(20) || component.blendMode != 2 || !component.blendAdditive || !component.overlay || !component.rotateToTarget || !component.attachedEntity {
		t.Fatalf("EF_RG_COIN component = %+v", component)
	}

	for _, tc := range []struct {
		name     string
		id       int
		funcName string
	}{
		{"EF_TALK_FROSTJOKE", effectTalkFrostJoke, "FrostJokeTalk"},
		{"EF_TALK_SCREAM", effectTalkScream, "ScreamTalk"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	throwItem, ok := worldEffectSpecForID(effectThrowItem)
	if !ok || len(throwItem.components) != 1 || throwItem.duration != 300*time.Millisecond {
		t.Fatalf("EF_THROWITEM spec = %+v ok=%t", throwItem, ok)
	}
	if component := throwItem.components[0]; component.kind != effectComponent3D || component.textureFile != "유저인터페이스/item/염산병.bmp" || component.duration != 300*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || component.angleStart != 180 || component.angleEnd != 360 || component.posZ != 1 || component.sizeStart != effectTableSize(30) || !component.attachedEntity {
		t.Fatalf("EF_THROWITEM component = %+v", component)
	}
}

func TestRobrowserSimpleEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	demo, ok := worldEffectSpecForID(effectDemonstration)
	if !ok || len(demo.components) != 1 {
		t.Fatalf("EF_DEMONSTRATION spec = %+v ok=%t", demo, ok)
	}
	if component := demo.components[0]; component.kind != effectComponentSPR || component.spriteFile != "데몬스트레이션" || component.attachedEntity {
		t.Fatalf("EF_DEMONSTRATION component = %+v", component)
	}

	job, ok := worldEffectSpecForID(effectJobLvUp50)
	if !ok || len(job.components) != 1 {
		t.Fatalf("EF_JOBLVUP50 spec = %+v ok=%t", job, ok)
	}
	if component := job.components[0]; component.kind != effectComponentSTR || component.strFile != "joblvup" || !component.attachedEntity {
		t.Fatalf("EF_JOBLVUP50 component = %+v", component)
	}

	colorPaper, ok := worldEffectSpecForID(effectColorPaper)
	if !ok || len(colorPaper.components) != 0 || colorPaper.duration != 500*time.Millisecond {
		t.Fatalf("EF_COLORPAPER spec = %+v ok=%t", colorPaper, ok)
	}
	if len(colorPaper.sfx) != 1 || colorPaper.sfx[0] != "effect\\wedding.wav" {
		t.Fatalf("EF_COLORPAPER sfx = %#v", colorPaper.sfx)
	}

	for _, tc := range []struct {
		name   string
		id     int
		sfx    []string
		delays []time.Duration
	}{
		{"EF_TRIPLEATTACK", effectTripleAttack, []string{"effect\\ef_hit2.wav", "effect\\ef_hit4.wav", "effect\\ef_hit2.wav"}, []time.Duration{0, 200 * time.Millisecond, 400 * time.Millisecond}},
		{"EF_MAGNUM2", effectMagnum2, []string{"permeter_attack.wav", "effect\\ef_magnumbreak.wav"}, []time.Duration{0, 300 * time.Millisecond}},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if !reflect.DeepEqual(spec.sfx, tc.sfx) || !reflect.DeepEqual(spec.sfxDelays, tc.delays) {
			t.Fatalf("%s sound = %v delays %v", tc.name, spec.sfx, spec.sfxDelays)
		}
	}

	chemical, ok := worldEffectSpecForID(effectChemical2)
	if !ok || len(chemical.components) != 1 || chemical.duration != 500*time.Millisecond {
		t.Fatalf("EF_CHEMICAL2 spec = %+v ok=%t", chemical, ok)
	}
	if chemical.cameraShake != 200*time.Millisecond || chemical.cameraShakeDelay != 132*time.Millisecond {
		t.Fatalf("EF_CHEMICAL2 camera shake = delay %s duration %s", chemical.cameraShakeDelay, chemical.cameraShake)
	}
	if component := chemical.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
		t.Fatalf("EF_CHEMICAL2 component = %+v", component)
	}

	blind, ok := worldEffectSpecForID(effectBlind)
	if !ok || len(blind.components) != 1 || blind.duration != 500*time.Millisecond {
		t.Fatalf("EF_BLIND spec = %+v ok=%t", blind, ok)
	}
	if len(blind.sfx) != 1 || blind.sfx[0] != "_blind.wav" {
		t.Fatalf("EF_BLIND sfx = %#v", blind.sfx)
	}
	if component := blind.components[0]; component.kind != effectComponentFUNC || component.funcName != "Blind" || component.funcAdapter != effectFuncUnknown || component.attachedEntity {
		t.Fatalf("EF_BLIND component = %+v", component)
	}

	poison, ok := worldEffectSpecForID(effectPoisonStatus)
	if !ok || len(poison.components) != 1 || poison.duration != 500*time.Millisecond {
		t.Fatalf("EF_POISON spec = %+v ok=%t", poison, ok)
	}
	if component := poison.components[0]; component.kind != effectComponentFUNC || component.funcName != "Poison" || component.funcAdapter != effectFuncUnknown || component.attachedEntity {
		t.Fatalf("EF_POISON component = %+v", component)
	}
}

func TestRobrowserPortalAndGroundEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	exit, ok := worldEffectSpecForID(effectExit2)
	if !ok || len(exit.components) != 3 || exit.duration != 1500*time.Millisecond {
		t.Fatalf("EF_EXIT2 spec = %+v ok=%t", exit, ok)
	}
	if len(exit.sfx) != 1 || exit.sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("EF_EXIT2 sfx = %#v", exit.sfx)
	}
	for i, want := range []struct {
		bottom float64
		top    float64
		height float64
	}{
		{0.3, 0.3, 35},
		{0.4, 0.6, 23},
		{0.5, 0.7, 5},
	} {
		component := exit.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond || component.alphaMax != 0.3 || component.animation != 1 || component.blendMode != 2 || !component.blendAdditive || !component.fade || !component.rotate || !component.attachedEntity {
			t.Fatalf("EF_EXIT2 component %d = %+v", i, component)
		}
		if component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height {
			t.Fatalf("EF_EXIT2 component %d size = %+v", i, component)
		}
		if component.color != (color.RGBA{R: 128, G: 128, B: 255, A: 255}) {
			t.Fatalf("EF_EXIT2 component %d color = %+v", i, component.color)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
		color   color.RGBA
		alpha   float64
		height  float64
	}{
		{"EF_BOTTOM_MAG", effectBottomMagnus, "ring_red", color.RGBA{}, 0.2, 5},
		{"EF_BOTTOM_SANC", effectBottomSanc, "magic_green", color.RGBA{R: 128, G: 230, B: 128, A: 255}, 0.3, 2},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 2 || spec.duration != 50*time.Second {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		first, second := spec.components[0], spec.components[1]
		for i, component := range []worldEffectComponent{first, second} {
			if component.kind != effectComponentCylinder || component.textureName != tc.texture || component.totalCircleSides != 4 || component.circleSides != 4 || component.bottomSize != 0.7 || component.topSize != 0.7 || component.height != tc.height || component.angleY != 45 || component.blendMode != 2 || !component.blendAdditive || component.rotate || !component.attachedEntity {
				t.Fatalf("%s component %d = %+v", tc.name, i, component)
			}
			if component.color != tc.color {
				t.Fatalf("%s component %d color = %+v", tc.name, i, component.color)
			}
		}
		if first.duration != 50*time.Second || first.alphaMax != tc.alpha || first.fade || first.repeat || first.animation != 0 {
			t.Fatalf("%s first cylinder = %+v", tc.name, first)
		}
		if second.duration != 2*time.Second || second.alphaMax != 0.1 || !second.fade || !second.repeat || second.animation != 1 {
			t.Fatalf("%s second cylinder = %+v", tc.name, second)
		}
	}

	warp, ok := worldEffectSpecForID(effectWarpZone2)
	if !ok || len(warp.components) != 3 || warp.duration != 7*time.Second {
		t.Fatalf("EF_WARPZONE2 spec = %+v ok=%t", warp, ok)
	}
	for i, component := range warp.components[:2] {
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 4*time.Second || component.duplicate != 4 || component.duplicateDelay != time.Second || component.alphaMax != 0.4 || component.animation != 3 || component.bottomSize < 1.9 || component.topSize < 3.2 || component.height != 1.1 || !component.repeat || component.rotate || !component.fade || !component.attachedEntity {
			t.Fatalf("EF_WARPZONE2 cylinder %d = %+v", i, component)
		}
	}
	particle := warp.components[2]
	if particle.kind != effectComponent3D || particle.textureFile != "effect/pok1.tga" || particle.duration != time.Second || particle.duplicate != 5 || particle.duplicateDelay != 300*time.Millisecond || particle.sizeStart != effectTableSize(50) || particle.posXStartRand != 3 || particle.posYStartRand != 3 || particle.posZEndRand != 2 || particle.posZEndMiddle != 2 || !particle.repeat || !particle.blendAdditive || !particle.attachedEntity {
		t.Fatalf("EF_WARPZONE2 particle = %+v", particle)
	}

	entry, ok := worldEffectSpecForID(effectEntry2)
	if !ok || len(entry.components) != 4 || entry.duration != 1500*time.Millisecond {
		t.Fatalf("EF_ENTRY2 spec = %+v ok=%t", entry, ok)
	}
	if len(entry.sfx) != 1 || entry.sfx[0] != "effect\\ef_portal.wav" {
		t.Fatalf("EF_ENTRY2 sfx = %#v", entry.sfx)
	}
	if entry.components[0].bottomSize != 0.3 || entry.components[3].topSize != 1.3 {
		t.Fatalf("EF_ENTRY2 cylinder sizes = %+v", entry.components)
	}
	for i, component := range entry.components {
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond || component.alphaMax != 0.5 || component.animation != 5 || component.blendMode != 2 || !component.blendAdditive || !component.fade || !component.rotate || !component.attachedEntity {
			t.Fatalf("EF_ENTRY2 component %d = %+v", i, component)
		}
	}
}

func TestRobrowserHealAndRecoveryEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name            string
		id              int
		duration        time.Duration
		firstHeight     float64
		secondHeight    float64
		thirdHeight     float64
		firstDuplicate  int
		secondDuplicate int
		thirdDuplicate  int
		secondSize      float64
		secondSizeRand  float64
		thirdSize       float64
		sparkNumber     int
	}{
		{"EF_HEAL2", effectHeal2, 1890 * time.Millisecond, 15, 13, 2, 4, 20, 10, 9, 2, 9, 2},
		{"EF_HEAL4", effectHeal4, 2 * time.Second, 18, 15, 3, 7, 25, 15, 10, 5, 11, 3},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 6 || spec.duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "_heal_effect.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		heights := []float64{tc.firstHeight, tc.secondHeight, tc.thirdHeight}
		for i, height := range heights {
			component := spec.components[i]
			if component.kind != effectComponentCylinder || component.textureName != "ring_white" || component.duration != 1500*time.Millisecond || component.alphaMax != 0.3 || component.animation != 1 || component.blendMode != 2 || !component.blendAdditive || !component.fade || !component.rotate || !component.attachedEntity || component.height != height {
				t.Fatalf("%s cylinder %d = %+v", tc.name, i, component)
			}
			if component.color != (color.RGBA{R: 178, G: 255, B: 178, A: 255}) {
				t.Fatalf("%s cylinder %d color = %+v", tc.name, i, component.color)
			}
		}
		if spec.components[0].bottomSize != 1.1 || spec.components[0].topSize != 1.1 || spec.components[1].bottomSize != 1 || spec.components[2].topSize != 3 {
			t.Fatalf("%s cylinder sizes = %+v", tc.name, spec.components[:3])
		}
		first, second, third := spec.components[3], spec.components[4], spec.components[5]
		if first.kind != effectComponent3D || first.textureFile != "effect/pok3.tga" || first.duration != 1500*time.Millisecond || first.duplicate != tc.firstDuplicate || first.duplicateDelay != 10*time.Millisecond || first.posXRand != 1.2 || first.posYRand != 1.2 || first.posZEndRand != 1 || first.posZEndMiddle != 8 || first.sizeStart != effectTableSize(9) || !first.sparkling || first.sparkNumber != tc.sparkNumber {
			t.Fatalf("%s first particle = %+v", tc.name, first)
		}
		if second.duration != 1300*time.Millisecond || second.delay != 400*time.Millisecond || second.duplicate != tc.secondDuplicate || second.posXRand != 1.5 || second.posYRand != 1.5 || second.posZEndRand != 3 || second.posZEndMiddle != 6 || second.sizeStart != effectTableSize(tc.secondSize) || second.sizeRand != effectTableSize(tc.secondSizeRand) || !second.sparkling || second.sparkNumber != tc.sparkNumber {
			t.Fatalf("%s second particle = %+v", tc.name, second)
		}
		if third.duration != 1100*time.Millisecond || third.delay != 200*time.Millisecond || third.duplicate != tc.thirdDuplicate || third.duplicateDelay != 50*time.Millisecond || third.posXRand != 1 || third.posYRand != 1 || third.posZEnd != 6 || third.posZStartRand != 1 || third.sizeStart != effectTableSize(tc.thirdSize) || third.sparkling {
			t.Fatalf("%s third particle = %+v", tc.name, third)
		}
	}

	for _, tc := range []struct {
		name  string
		id    int
		color color.RGBA
		wav   string
	}{
		{"EF_HPTIME", effectHPTime, color.RGBA{R: 230, G: 255, B: 230, A: 255}, "_heal_effect.wav"},
		{"EF_SPTIME", effectSPTime, color.RGBA{R: 230, G: 230, B: 255, A: 255}, "effect\\흡기.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 1110*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != "effect/pok1.tga" || component.color != tc.color || component.duration != 500*time.Millisecond || component.delay != 500*time.Millisecond || component.duplicate != 12 || component.duplicateDelay != 10*time.Millisecond || component.alphaMax != 0.8 || component.sizeStart != effectTableSize(30) || component.sizeRand != effectTableSize(20) || component.posXRand != 0.6 || component.posYRand != 0.6 || component.posZStartRand != 1.5 || component.posZStartMiddle != 2 || component.posZEndRand != 1 || component.posZEndMiddle != 5 || !component.sparkling || component.sparkNumber != 3 || !component.blendAdditive || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserAsuraAndGuardEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	asura, ok := worldEffectSpecForID(effectBeginAsura)
	if !ok || len(asura.components) != 14 || asura.duration != 2100*time.Millisecond {
		t.Fatalf("EF_BEGINASURA spec = %+v ok=%t", asura, ok)
	}
	firstRing, secondRing := asura.components[0], asura.components[1]
	if firstRing.kind != effectComponentCylinder || firstRing.textureName != "ring_white" || firstRing.duration != 800*time.Millisecond || firstRing.animation != 2 || firstRing.bottomSize != 1 || firstRing.topSize != 4.5 || firstRing.height != -4 || !firstRing.fade || !firstRing.attachedEntity || firstRing.blendMode != 2 {
		t.Fatalf("EF_BEGINASURA first ring = %+v", firstRing)
	}
	if secondRing.topSize != 2.5 || secondRing.height != -4 {
		t.Fatalf("EF_BEGINASURA second ring = %+v", secondRing)
	}
	firstGlyph := asura.components[2]
	if firstGlyph.kind != effectComponent3D || firstGlyph.textureFile != "effect/asura1.tga" || firstGlyph.duration != 1200*time.Millisecond || firstGlyph.delay != 0 || firstGlyph.duplicate != 3 || firstGlyph.duplicateDelay != 150*time.Millisecond || firstGlyph.alphaMax != 1 || firstGlyph.alphaMaxDelta != -0.25 || !firstGlyph.fadeIn || firstGlyph.fadeOut || firstGlyph.sizeStart != effectTableSize(250) || firstGlyph.sizeEnd != effectTableSize(120) || !firstGlyph.sizeSmooth || firstGlyph.posX != -6 || firstGlyph.posZ != 4 || !firstGlyph.overlay || !firstGlyph.attachedEntity {
		t.Fatalf("EF_BEGINASURA first glyph = %+v", firstGlyph)
	}
	if firstGlyph.color != (color.RGBA{R: 26, G: 26, B: 26, A: 255}) {
		t.Fatalf("EF_BEGINASURA glyph color = %+v", firstGlyph.color)
	}
	lastGlyph := asura.components[13]
	if lastGlyph.textureFile != "effect/asura6.tga" || lastGlyph.duration != 400*time.Millisecond || lastGlyph.delay != 1700*time.Millisecond || lastGlyph.duplicate != 0 || !lastGlyph.fadeOut || lastGlyph.sizeStart != effectTableSize(120) || lastGlyph.sizeEnd != effectTableSize(200) || lastGlyph.posX != 6 || lastGlyph.posZ != 4 {
		t.Fatalf("EF_BEGINASURA last glyph = %+v", lastGlyph)
	}

	guard, ok := worldEffectSpecForID(effectGuard)
	if !ok || len(guard.components) != 3 || guard.duration != 600*time.Millisecond {
		t.Fatalf("EF_GUARD spec = %+v ok=%t", guard, ok)
	}
	if len(guard.sfx) != 1 || guard.sfx[0] != "effect\\kyrie_guard.wav" {
		t.Fatalf("EF_GUARD sfx = %#v", guard.sfx)
	}
	for i, want := range []struct {
		bottom float64
		top    float64
		height float64
		posZ   float64
	}{
		{1.5, 1, 0.7, 2.14},
		{1.5, 1.5, 1.14, 1},
		{1, 1.5, 0.7, 0.3},
	} {
		component := guard.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "guardk" || component.duration != 600*time.Millisecond || component.alphaMax != 0.6 || component.blendMode != 2 || !component.blendAdditive || !component.fade || component.totalCircleSides != 8 || component.circleSides != 5 || component.angleY != 112.5 || !component.attachedEntity {
			t.Fatalf("EF_GUARD component %d = %+v", i, component)
		}
		if component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height || component.posZ != want.posZ {
			t.Fatalf("EF_GUARD component %d shape = %+v", i, component)
		}
		if component.color != (color.RGBA{R: 232, G: 255, B: 230, A: 255}) {
			t.Fatalf("EF_GUARD component %d color = %+v", i, component.color)
		}
	}
}

func TestRobrowserSimpleEffectsThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_LKCONCENTRATION", effectLKConcentration, "twohand", "effect\\knight_twohandquicken.wav", true, true},
		{"EF_DEVIL", effectDeath, "devil", "", true, false},
		{"EF_MELTDOWN", effectMeltdown, "melt", "", true, false},
		{"EF_CARTBOOST", effectCartBoost, "cart", "effect\\ef_incagility.wav", true, false},
		{"EF_REJECTSWORD", effectRejectSword, "sword", "effect\\kyrie_guard.wav", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
		head bool
	}{
		{"EF_VALLENTINE", effectFoodChocolate, "vallentine", "effect\\vallentine.wav", false},
		{"EF_DRAGONSMOKE", effectDragonSmoke, "poisonhit", "", false},
		{"EF_LIGHTBLADE", effectLightBlade, "한복천사", "", true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || !component.attachedEntity || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_AURABLADE", effectAuraBlade, "effect\\오라 블레이드.wav"},
		{"EF_REDBODY", effectRedBody, "effect\\버서크.wav"},
		{"EF_BOTTOM_GOSPEL", effectBottomGospel, "effect\\가스펠.wav"},
		{"EF_HITLINE2", effectHitLine2, "effect\\맹호경파산.wav"},
		{"EF_LINELINK2", effectLineLink2, "effect\\소울 체인지.wav"},
		{"EF_TRUESIGHT", effectTrueSight, "effect\\hunter_detecting.wav"},
		{"EF_TRIPLEATTACK2", effectTripleAttack2, "effect\\샤프슈팅.wav"},
		{"EF_PORTAL4", effectPortal4, "effect\\윈드워크.wav"},
		{"EF_TRIPLEATTACK3", effectTripleAttack3, "effect\\애로우 발칸.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	base, ok := worldEffectSpecForID(effectBaseLevelUp)
	if !ok || len(base.components) != 1 || base.components[0].kind != effectComponentSTR || base.components[0].strFile != "angel" || !base.components[0].attachedEntity {
		t.Fatalf("EF_ANGEL spec = %+v ok=%t", base, ok)
	}
}

func TestRobrowserLevel99AliasesThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		funcName string
		adapter  effectFuncAdapter
		texture  string
	}{
		{"EF_LEVEL99_4", effectLevel99Aura1, "Level99Bubble", effectFuncLevel99Bubble, "effect/whitelight.tga"},
		{"EF_LEVEL99_5", effectLevel99AuraMid, "Level99Aura", effectFuncLevel99Aura, "effect/ring_blue.tga"},
		{"EF_LEVEL99_6", effectLevel99AuraBottom, "GroundAura", effectFuncGroundAura, "effect/pikapika2.bmp"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 5*time.Minute {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != tc.adapter || component.textureFile != tc.texture || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserCombatEffectsThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	soul, ok := worldEffectSpecForID(effectSoulBreaker)
	if !ok || len(soul.components) != 1 || soul.duration != 500*time.Millisecond {
		t.Fatalf("EF_SOULBREAKER spec = %+v ok=%t", soul, ok)
	}
	if len(soul.sfx) != 1 || soul.sfx[0] != "effect\\기공포.wav" {
		t.Fatalf("EF_SOULBREAKER sfx = %#v", soul.sfx)
	}
	if component := soul.components[0]; component.kind != effectComponent3D || component.textureFile != "effect/purpleslash.tga" || component.alphaMax != 0.4 || !component.fadeIn || !component.fadeOut || !component.toSrc || !component.rotateWithCamera || !component.rotateToTarget || component.angleStart != 90 || component.posZ != 2 || component.sizeStart != effectTableSize(100) || component.sizeEnd != effectTableSize(200) || !component.attachedEntity {
		t.Fatalf("EF_SOULBREAKER component = %+v", component)
	}

	pressure, ok := worldEffectSpecForID(effectPressure)
	if !ok || len(pressure.components) != 3 || pressure.duration != 1001*time.Millisecond {
		t.Fatalf("EF_PRESSURE spec = %+v ok=%t", pressure, ok)
	}
	if len(pressure.sfx) != 1 || pressure.sfx[0] != "effect\\프레셔.wav" || pressure.cameraShakeDelay != 500*time.Millisecond || pressure.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_PRESSURE timing/sfx = sfx %#v delay %s shake %s", pressure.sfx, pressure.cameraShakeDelay, pressure.cameraShake)
	}
	first, second, quake := pressure.components[0], pressure.components[1], pressure.components[2]
	if first.kind != effectComponent3D || first.textureFile != "effect/cross_old.bmp" || first.duration != 500*time.Millisecond || first.alphaMax != 0.6 || first.blendMode != 2 || !first.blendAdditive || !first.rotate || first.angleEnd != -611 || first.posZ != 20 || first.posZEnd != 5 || first.sizeStart != effectTableSize(100) || !first.attachedEntity {
		t.Fatalf("EF_PRESSURE first cross = %+v", first)
	}
	if second.kind != effectComponent3D || second.delay != 501*time.Millisecond || !second.fadeOut || second.angleStart != -611 || second.posZ != 5 || second.sizeStart != effectTableSize(100) || !second.attachedEntity {
		t.Fatalf("EF_PRESSURE second cross = %+v", second)
	}
	if quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || !quake.attachedEntity {
		t.Fatalf("EF_PRESSURE quake = %+v", quake)
	}

	for _, tc := range []struct {
		name      string
		id        int
		funcName  string
		wav       string
		duration  time.Duration
		delay     time.Duration
		duplicate int
	}{
		{"EF_BASH3D", effectBash3D, "Bash3D", "effect\\bash3d.wav", 500 * time.Millisecond, 200 * time.Millisecond, 5},
		{"EF_BASH3D2", effectBash3D2, "Bash3D2", "effect\\mon_폭기.wav", 400 * time.Millisecond, 50 * time.Millisecond, 8},
		{"EF_BASH3D3", effectBash3D3, "Bash3D3", "effect\\헤드 크러쉬.wav", 675 * time.Millisecond, 500 * time.Millisecond, 6},
		{"EF_BASH3D4", effectBash3D4, "Bash3D4", "effect\\비트 조인트.wav", 675 * time.Millisecond, 500 * time.Millisecond, 6},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 3 || spec.duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		if component := spec.components[0]; component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
			t.Fatalf("%s body func = %+v", tc.name, component)
		}
		for i, component := range spec.components[1:] {
			wantTop := 4.5
			if i == 1 {
				wantTop = 7.2
			}
			if component.kind != effectComponentCylinder || component.textureName != "alpha_center" || component.duration != 175*time.Millisecond || component.delay != tc.delay || component.duplicate != tc.duplicate || component.alphaMax != 0.6 || !component.fade || component.angleX != -90 || component.angleZRandom != 360 || !component.fixedPerspective || component.posZ != 1.5 || component.height != 0 || component.bottomSize != 0.01 || component.topSize != wantTop || component.animation != 2 || component.totalCircleSides != 30 || component.circleSides != 1 || !component.attachedEntity {
				t.Fatalf("%s cylinder %d = %+v", tc.name, i, component)
			}
		}
	}
}

func TestRobrowserBasilicaDrainAndMagicEffectsThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	basilica, ok := worldEffectSpecForID(effectBottomBasilica)
	if !ok || len(basilica.components) != 4 || basilica.duration != 20*time.Second {
		t.Fatalf("EF_BOTTOM_BASILICA spec = %+v ok=%t", basilica, ok)
	}
	for i, want := range []struct {
		size   float64
		height float64
		alpha  float64
		angleY float64
	}{
		{2.45, 2.0, 32.0 / 255.0, 0},
		{2.52, 2.1, 32.0 / 255.0, 10},
		{2.6, 2.0, 15.0 / 255.0, 26.6},
		{2.6, 2.0, 15.0 / 255.0, 79.8},
	} {
		component := basilica.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "alpha_down" || component.duration != 20*time.Second || component.totalCircleSides != 4 || component.circleSides != 4 || component.bottomSize != want.size || component.topSize != want.size || component.height != want.height || math.Abs(component.alphaMax-want.alpha) > 0.0001 || component.blendMode != 2 || !component.blendAdditive || !component.rotateWithCamera || component.angleY != want.angleY || !component.attachedEntity {
			t.Fatalf("EF_BOTTOM_BASILICA component %d = %+v", i, component)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		color     color.RGBA
		sizeStart float64
		sizeEnd   float64
	}{
		{"EF_ENERGYDRAIN2", effectEnergyDrain2, color.RGBA{R: 204, G: 204, B: 255, A: 255}, 160, 190},
		{"EF_ENERGYDRAIN3", effectEnergyDrain3, color.RGBA{R: 178, G: 255, B: 178, A: 255}, 140, 170},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 600*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.spriteFile != "data/sprite/이팩트/particle1" || !component.spriteRepeat || component.duration != 600*time.Millisecond || component.duplicate != 5 || !component.fromSrc || !component.toSrc || !component.rotateToTarget || component.color != tc.color || component.sizeStart != effectTableSize(tc.sizeStart) || component.sizeEnd != effectTableSize(tc.sizeEnd) || component.posZ != 5 || component.arc != 3 || component.retreat != 3 {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	trans, ok := worldEffectSpecForID(effectTransBlueBody)
	if !ok || len(trans.components) != 1 || trans.duration != 900*time.Millisecond {
		t.Fatalf("EF_TRANSBLUEBODY spec = %+v ok=%t", trans, ok)
	}
	if component := trans.components[0]; component.kind != effectComponentFUNC || component.funcName != "TransBlueBody" || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
		t.Fatalf("EF_TRANSBLUEBODY component = %+v", component)
	}

	magic, ok := worldEffectSpecForID(effectMagicCrasher)
	if !ok || len(magic.components) != 2 || magic.duration != time.Second {
		t.Fatalf("EF_MAGICCRASHER spec = %+v ok=%t", magic, ok)
	}
	if len(magic.sfx) != 1 || magic.sfx[0] != "effect\\매직 크래쉬.wav" || magic.cameraShakeDelay != 300*time.Millisecond || magic.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_MAGICCRASHER timing/sfx = sfx %#v delay %s shake %s", magic.sfx, magic.cameraShakeDelay, magic.cameraShake)
	}
	if body, quake := magic.components[0], magic.components[1]; body.kind != effectComponentFUNC || body.funcName != "MagicCrasherBodyColor" || !body.attachedEntity || quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || quake.delay != 300*time.Millisecond || !quake.attachedEntity {
		t.Fatalf("EF_MAGICCRASHER components = %+v", magic.components)
	}

	falcon, ok := worldEffectSpecForID(effectFalconAssault)
	if !ok || len(falcon.components) != 1 || falcon.duration != 500*time.Millisecond {
		t.Fatalf("EF_FALCONASSAULT spec = %+v ok=%t", falcon, ok)
	}
	if len(falcon.sfx) != 1 || falcon.sfx[0] != "effect\\hunter_blitzbeat.wav" || falcon.cameraShakeDelay != 300*time.Millisecond || falcon.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_FALCONASSAULT timing/sfx = sfx %#v delay %s shake %s", falcon.sfx, falcon.cameraShakeDelay, falcon.cameraShake)
	}
	if component := falcon.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || component.delay != 300*time.Millisecond || !component.attachedEntity {
		t.Fatalf("EF_FALCONASSAULT component = %+v", component)
	}

	moonlit, ok := worldEffectSpecForID(effectMoonlit)
	if !ok || len(moonlit.components) != 1 || moonlit.duration != 20*time.Second {
		t.Fatalf("EF_SPHEREWIND2 spec = %+v ok=%t", moonlit, ok)
	}
	if len(moonlit.sfx) != 1 || moonlit.sfx[0] != "effect\\달빛세레나데.wav" {
		t.Fatalf("EF_SPHEREWIND2 sfx = %#v", moonlit.sfx)
	}
	if component := moonlit.components[0]; component.kind != effectComponentFUNC || component.funcName != "FlatColorTile" || component.funcAdapter != effectFuncFlatColorTile || component.color != (color.RGBA{R: 255, G: 138, B: 187, A: 153}) || component.sizeStart != 1 || component.attachedEntity {
		t.Fatalf("EF_SPHEREWIND2 component = %+v", component)
	}
}

func TestRobrowserEffectsFourHundredToFourFiftyMatchTableRows(t *testing.T) {
	portal, ok := worldEffectSpecForID(effectPortal5)
	if !ok || len(portal.components) != 1 || portal.duration != 800*time.Millisecond {
		t.Fatalf("EF_PORTAL5 spec = %+v ok=%t", portal, ok)
	}
	if component := portal.components[0]; component.kind != effectComponentFUNC || component.funcName != "EffectBodyColor" || component.funcAdapter != effectFuncBodyColor || component.duration != 800*time.Millisecond || !component.attachedEntity {
		t.Fatalf("EF_PORTAL5 component = %+v", component)
	}

	mindBreaker, ok := worldEffectSpecForID(effectMagicCrasher2)
	if !ok || len(mindBreaker.components) != 1 || mindBreaker.duration != time.Second {
		t.Fatalf("EF_MAGICCRASHER2 spec = %+v ok=%t", mindBreaker, ok)
	}
	if len(mindBreaker.sfx) != 1 || mindBreaker.sfx[0] != "effect\\swordman_provoke.wav" {
		t.Fatalf("EF_MAGICCRASHER2 sfx = %#v", mindBreaker.sfx)
	}
	if component := mindBreaker.components[0]; component.kind != effectComponentFUNC || component.funcName != "EffectBodyColor" || component.funcAdapter != effectFuncBodyColor || component.duration != time.Second || !component.attachedEntity {
		t.Fatalf("EF_MAGICCRASHER2 component = %+v", component)
	}

	spider, ok := worldEffectSpecForID(effectBottomSpider)
	if !ok || len(spider.components) != 1 || spider.duration != 5*time.Second {
		t.Fatalf("EF_BOTTOM_SPIDER spec = %+v ok=%t", spider, ok)
	}
	if component := spider.components[0]; component.kind != effectComponentFUNC || component.funcName != "SpiderWeb" || component.funcAdapter != effectFuncGroundTexture || component.textureFile != "effect/spiderweb.tga" || component.duration != 5*time.Second || math.Abs(component.alphaMax-0.7) > 0.0001 || component.sizeStart != 1.5 || component.sizeEnd != 1.5 || component.posZ != 0.05 || !component.renderBefore || component.attachedEntity {
		t.Fatalf("EF_BOTTOM_SPIDER component = %+v", component)
	}

	fogWall, ok := worldEffectSpecForID(effectFogWallGround)
	if !ok || len(fogWall.components) != 2 || fogWall.duration != 1500*time.Millisecond {
		t.Fatalf("PF_FOGWALL ground spec = %+v ok=%t", fogWall, ok)
	}
	if component := fogWall.components[0]; component.kind != effectComponentFUNC || component.funcName != "FlatColorTile" || component.funcAdapter != effectFuncFlatColorTile || component.color != (color.RGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 153}) || component.sizeStart != 1 || !component.renderBefore || component.attachedEntity {
		t.Fatalf("PF_FOGWALL flat tile component = %+v", component)
	}
	if component := fogWall.components[1]; component.kind != effectComponentFUNC || component.funcName != "GroundTexture" || component.funcAdapter != effectFuncGroundTexture || component.textureFile != "effect/lens_w.bmp" || component.duration != 1500*time.Millisecond || component.sizeStart != 0.5 || component.sizeEnd != 0.5 || math.Abs(component.alphaMax-0.7) > 0.0001 || component.posZ != 0.4 || !component.blendAdditive || !component.renderBefore || component.attachedEntity {
		t.Fatalf("PF_FOGWALL texture component = %+v", component)
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_SOULBURN", effectSoulBurn, "소울번"},
		{"EF_SOULCHANGE", effectSoulChange, "사랑효과"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
	}

	meteor, ok := worldEffectSpecForID(effectSoulBreaker2)
	if !ok || len(meteor.components) != 8 || meteor.duration != 500*time.Millisecond {
		t.Fatalf("EF_SOULBREAKER2 spec = %+v ok=%t", meteor, ok)
	}
	if len(meteor.sfx) != 1 || meteor.sfx[0] != "effect\\메테오 어썰트.wav" {
		t.Fatalf("EF_SOULBREAKER2 sfx = %#v", meteor.sfx)
	}
	for i, tc := range []struct {
		posX    float64
		posY    float64
		posXEnd float64
		posYEnd float64
		angle   float64
	}{
		{-1, 0, -5, 0, 0},
		{-0.7, -0.7, -3.53, -3.53, -45},
		{0, -1, 0, -5, -90},
		{0.7, -0.7, 3.53, -3.53, -135},
		{1, 0, 5, 0, -180},
		{0.7, 0.7, 3.53, 3.53, -225},
		{0, 1, 0, 5, -270},
		{-0.7, 0.7, -3.53, 3.53, -315},
	} {
		component := meteor.components[i]
		if component.kind != effectComponent3D || component.textureFile != "effect/purpleslash.tga" || component.duration != 500*time.Millisecond || math.Abs(component.alphaMax-0.6) > 0.0001 || !component.fadeOut || !component.rotateWithCamera || component.sizeStart != effectTableSize(100) || component.sizeEnd != effectTableSize(200) || component.posX != tc.posX || component.posY != tc.posY || component.posXEnd != tc.posXEnd || component.posYEnd != tc.posYEnd || component.angleStart != tc.angle {
			t.Fatalf("EF_SOULBREAKER2 slash %d = %+v", i, component)
		}
	}

	for _, tc := range []struct {
		name       string
		id         int
		funcName   string
		duration   time.Duration
		targetSize float64
	}{
		{"EF_BABYBODY", effectBabyBody, "EffectSmallTransition", 300 * time.Millisecond, 2.5},
		{"EF_BABYBODY2", effectBabyBody2, "EffectSmall", 5 * time.Minute, 2.5},
		{"EF_GIANTBODY", effectGiantBody, "EffectBigTransition", 300 * time.Millisecond, 7.5},
		{"EF_GIANTBODY2", effectGiantBody2, "EffectBig", 5 * time.Minute, 7.5},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != effectFuncUnknown || component.duration != tc.duration || component.sizeEnd != tc.targetSize || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_QUAKEBODY", effectQuakeBody, "effect\\복호격.wav"},
		{"EF_STOPEFFECT", effectStopEffect, "effect\\t_효과음1.wav"},
		{"EF_JUMPBODY", effectJumpBody, "effect\\t_회피2.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound only", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
	}

	assumptio, ok := worldEffectSpecForID(effectAssumptio2)
	if !ok || len(assumptio.components) != 1 {
		t.Fatalf("EF_ASSUMPTIO2 spec = %+v ok=%t", assumptio, ok)
	}
	if len(assumptio.sfx) != 1 || assumptio.sfx[0] != "effect\\아숨프티오.wav" {
		t.Fatalf("EF_ASSUMPTIO2 sfx = %#v", assumptio.sfx)
	}
	if component := assumptio.components[0]; component.kind != effectComponentSTR || component.strFile != "asum" || !component.attachedEntity {
		t.Fatalf("EF_ASSUMPTIO2 component = %+v", component)
	}
}

func TestRobrowserSimpleEffectsFourFiftyToFiveHundredMatchTableRows(t *testing.T) {
	darkCross, ok := worldEffectSpecForID(effectDarkGrandCross)
	if !ok || len(darkCross.components) != 0 || len(darkCross.sfx) != 0 {
		t.Fatalf("EF_GRANDCROSS2 spec = %+v ok=%t, want empty robr row", darkCross, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_NPC_STOP", effectNPCStop, "스톱"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached SPR %q", tc.name, component, tc.file)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_MOCHI", effectMochi, "찹쌀떡"},
		{"EF_LAMADAN", effectRamadan, "ramadan"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_AGIUP", effectNPCPowerUp, "effect\\mon_폭기.wav"},
		{"EF_JUMPKICK", effectJumpKick, "effect\\t_날라차기.wav"},
		{"EF_EDP", effectEDP, "effect\\assasin_cloaking.wav"},
		{"EF_GUARD2", effectPreserve, "effect\\black_maximize_power_sword_bic.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserDarkCombatEffectsFourFiftyToFiveHundredMatchTableRows(t *testing.T) {
	soul, ok := worldEffectSpecForID(effectDarkSoulStrike)
	if !ok || len(soul.components) != 2 || soul.duration != 450*time.Millisecond {
		t.Fatalf("EF_SOULSTRIKE2 spec = %+v ok=%t", soul, ok)
	}
	if len(soul.sfx) != 1 || soul.sfx[0] != "effect\\ef_soulstrike.wav" {
		t.Fatalf("EF_SOULSTRIKE2 sfx = %#v", soul.sfx)
	}
	spark, particle := soul.components[0], soul.components[1]
	if spark.kind != effectComponent3D || spark.textureFile != "effect/pok3.tga" || spark.duration != 200*time.Millisecond || spark.delay != 250*time.Millisecond || spark.duplicateDelay != 150*time.Millisecond || !spark.fadeIn || !spark.fadeOut || !spark.toSrc || spark.posZEnd != 1 || !spark.posZSmooth || spark.posZStartRand != 5 || spark.posZStartMiddle != 6 || spark.sizeStart != effectTableSize(50) || !spark.attachedEntity {
		t.Fatalf("EF_SOULSTRIKE2 spark = %+v", spark)
	}
	if particle.kind != effectComponent3D || particle.spriteFile != "data/sprite/이팩트/particle5" || !particle.spriteRepeat || particle.duration != 250*time.Millisecond || particle.duplicate != 5 || particle.duplicateDelay != 20*time.Millisecond || !particle.toSrc || !particle.rotateToTarget || particle.sizeStart != effectTableSize(100) || particle.sizeEnd != effectTableSize(500) || particle.posZ != 3 || particle.arc != 4 || particle.retreat != 4 {
		t.Fatalf("EF_SOULSTRIKE2 particle = %+v", particle)
	}

	jupitel, ok := worldEffectSpecForID(effectDarkJupitelHit)
	if !ok || len(jupitel.components) != 2 || jupitel.duration != 300*time.Millisecond {
		t.Fatalf("EF_YUFITEL2 spec = %+v ok=%t", jupitel, ok)
	}
	pang, blast := jupitel.components[0], jupitel.components[1]
	if pang.kind != effectComponent3D || pang.textureFile != "effect/pokjuk_d.bmp" || pang.duration != 100*time.Millisecond || pang.sizeStart != 0 || pang.sizeEnd != effectTableSize(25) || pang.blendMode != 2 || !pang.blendAdditive || !pang.rotateToTarget || !pang.fadeOut || !pang.overlay || !pang.attachedEntity {
		t.Fatalf("EF_YUFITEL2 pang = %+v", pang)
	}
	if blast.kind != effectComponent3D || len(blast.textureFiles) != 5 || blast.textureFiles[0] != "effect/twirl_soft.bmp" || blast.textureFiles[1] != "effect/thunder_ball_b.bmp" || blast.textureFiles[3] != "effect/thunder_ball_c.bmp" || blast.frameDelay != 10*time.Millisecond || blast.duration != 300*time.Millisecond || blast.sizeStart != effectTableSize(75) || blast.blendMode != 2 || !blast.blendAdditive || !blast.overlay || !blast.attachedEntity {
		t.Fatalf("EF_YUFITEL2 blast = %+v", blast)
	}

	casting, ok := worldEffectSpecForID(effectDarkCasting)
	if !ok || len(casting.components) != 1 || casting.duration != 900*time.Millisecond {
		t.Fatalf("EF_DARKCASTING spec = %+v ok=%t", casting, ok)
	}
	if len(casting.sfx) != 1 || casting.sfx[0] != "effect\\ef_beginspell.wav" {
		t.Fatalf("EF_DARKCASTING sfx = %#v", casting.sfx)
	}
	ring := casting.components[0]
	if ring.kind != effectComponentCylinder || ring.textureName != "ring_black" || ring.alphaMax != 0.8 || ring.animation != 2 || ring.blendMode != 2 || !ring.blendAdditive || ring.bottomSize != 1 || ring.topSize != 5 || ring.height != 4 || !ring.fade || !ring.rotate || !ring.attachedEntity {
		t.Fatalf("EF_DARKCASTING ring = %+v", ring)
	}
}

func TestRobrowserMildWindEffectsFourFiftyToFiveHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      int
		texture string
	}{
		{"EF_BEGINASURA1", effectBeginAsura1, "effect/hanmoon1.tga"},
		{"EF_BEGINASURA2", effectBeginAsura2, "effect/hanmoon2.tga"},
		{"EF_BEGINASURA3", effectBeginAsura3, "effect/hanmoon3.tga"},
		{"EF_BEGINASURA4", effectBeginAsura4, "effect/hanmoon4.tga"},
		{"EF_BEGINASURA5", effectBeginAsura5, "effect/hanmoon7.tga"},
		{"EF_BEGINASURA6", effectBeginAsura6, "effect/hanmoon5.tga"},
		{"EF_BEGINASURA7", effectBeginAsura7, "effect/hanmoon6.tga"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 5 || spec.duration != time.Second {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\t_바람방출.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		first, second, last := spec.components[0], spec.components[1], spec.components[4]
		if first.kind != effectComponent3D || first.textureFile != tc.texture || first.color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) || first.alphaMax != 1 || first.sizeStart != effectTableSize(300) || first.sizeEnd != effectTableSize(100) || !first.sizeSmooth || first.posZ != 4 || first.blendMode != 2 || !first.blendAdditive || !first.fadeIn || !first.fadeOut || !first.attachedEntity {
			t.Fatalf("%s first glyph = %+v", tc.name, first)
		}
		if second.color != (color.RGBA{R: 178, G: 178, B: 255, A: 255}) || second.alphaMax != 0.2 || second.sizeStart != effectTableSize(220) || second.sizeEnd != effectTableSize(20) {
			t.Fatalf("%s second glyph = %+v", tc.name, second)
		}
		if last.color != (color.RGBA{R: 25, G: 25, B: 255, A: 255}) || last.alphaMax != 0.2 || last.sizeStart != effectTableSize(450) || last.sizeEnd != effectTableSize(100) {
			t.Fatalf("%s last glyph = %+v", tc.name, last)
		}
	}
}

func TestRobrowserSimpleEffectsFiveHundredToFiveFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_MAPAE", effectMapae, "mapae", "effect\\mapae.wav"},
		{"EF_ITEMPOKJUK", effectItemPokJuk, "itempokjuk", "effect\\itempokjuk.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_05VAL", effectValentine05, "05vallentine", ""},
		{"EF_ITEMFAST", effectItemFastDown, "fast", "effect\\fast.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached SPR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	hermode, ok := worldEffectSpecForID(effectBottomHermode)
	if !ok || len(hermode.components) != 0 || len(hermode.sfx) != 0 {
		t.Fatalf("EF_BOTTOM_HERMODE spec = %+v ok=%t, want empty robr row", hermode, ok)
	}
	hermodeMusic, ok := worldEffectSpecForID(effectHermodeMusic)
	if !ok || len(hermodeMusic.components) != 0 || len(hermodeMusic.sfx) != 1 || hermodeMusic.sfx[0] != "effect\\헤르모드의 지팡이" {
		t.Fatalf("517_music spec = %+v ok=%t, want robr Hermode sound", hermodeMusic, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_HATED", effectHated, "effect\\t_보조마법.wav"},
		{"EF_STIN", effectStin, "effect\\t_에너지방출.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserFuncEffectsFiveHundredToFiveFiftyMatchTableRows(t *testing.T) {
	spin, ok := worldEffectSpecForID(effectCastSpin)
	if !ok || spin.duration != 500*time.Millisecond || len(spin.components) != 1 {
		t.Fatalf("EF_CASTSPIN spec = %+v ok=%t", spin, ok)
	}
	spinComponent := spin.components[0]
	if spinComponent.kind != effectComponentFUNC || spinComponent.funcName != "CastSpin" || spinComponent.funcAdapter != effectFuncUnknown || !spinComponent.attachedEntity {
		t.Fatalf("EF_CASTSPIN component = %+v, want attached unsupported CastSpin FUNC", spinComponent)
	}

	chookgi, ok := worldEffectSpecForID(effectChookgi2)
	if !ok || chookgi.duration != 5*time.Minute || len(chookgi.components) != 1 {
		t.Fatalf("EF_CHOOKGI2 spec = %+v ok=%t", chookgi, ok)
	}
	sphere := chookgi.components[0]
	if sphere.kind != effectComponentFUNC || sphere.funcName != "SpiritSphere" || sphere.funcAdapter != effectFuncSpiritSphere || sphere.textureFile != "effect/thunder_center.bmp" || sphere.duplicate != 5 || !sphere.attachedEntity {
		t.Fatalf("EF_CHOOKGI2 component = %+v", sphere)
	}

	chemical, ok := worldEffectSpecForID(effectChemical2Dash)
	if !ok || chemical.duration != 500*time.Millisecond || chemical.cameraShake != 200*time.Millisecond || chemical.cameraShakeDelay != 132*time.Millisecond || len(chemical.components) != 1 {
		t.Fatalf("EF_CHEMICAL2DASH spec = %+v ok=%t", chemical, ok)
	}
	if component := chemical.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || !component.attachedEntity {
		t.Fatalf("EF_CHEMICAL2DASH component = %+v", component)
	}

	acid, ok := worldEffectSpecForID(effectAcidDemon)
	if !ok || acid.duration != 500*time.Millisecond || acid.cameraShake != 200*time.Millisecond || acid.cameraShakeDelay != 200*time.Millisecond || len(acid.components) != 1 {
		t.Fatalf("EF_ACIDDEMON spec = %+v ok=%t", acid, ok)
	}
	if component := acid.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || component.delay != 200*time.Millisecond || !component.attachedEntity {
		t.Fatalf("EF_ACIDDEMON component = %+v", component)
	}
}

func TestRobrowserBeginAsuraElevenMatchesTableRow(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBeginAsura11)
	if !ok || spec.duration != 2100*time.Millisecond || len(spec.components) != 14 {
		t.Fatalf("EF_BEGINASURA11 spec = %+v ok=%t", spec, ok)
	}
	firstRing, secondRing := spec.components[0], spec.components[1]
	for i, ring := range []worldEffectComponent{firstRing, secondRing} {
		if ring.kind != effectComponentCylinder || ring.textureName != "ring_white" || ring.duration != 800*time.Millisecond || ring.animation != 2 || ring.bottomSize != 1 || ring.height != -4 || !ring.fade || ring.rotate || !ring.attachedEntity || ring.blendMode != 2 || !ring.blendAdditive {
			t.Fatalf("EF_BEGINASURA11 ring %d = %+v", i, ring)
		}
	}
	if firstRing.topSize != 4.5 || secondRing.topSize != 2.5 {
		t.Fatalf("EF_BEGINASURA11 ring top sizes = %.1f %.1f", firstRing.topSize, secondRing.topSize)
	}

	firstGlyph, firstOut, lastOut := spec.components[2], spec.components[3], spec.components[13]
	if firstGlyph.kind != effectComponent3D || firstGlyph.textureFile != "effect/asura11.tga" || firstGlyph.duration != 1200*time.Millisecond || firstGlyph.delay != 0 || firstGlyph.posX != -8 || firstGlyph.posZ != 4 || firstGlyph.alphaMax != 1 || firstGlyph.duplicate != 3 || firstGlyph.duplicateDelay != 150*time.Millisecond || firstGlyph.alphaMaxDelta != -0.25 || !firstGlyph.fadeIn || firstGlyph.fadeOut || firstGlyph.sizeStart != effectTableSize(300) || firstGlyph.sizeEnd != effectTableSize(150) || !firstGlyph.sizeSmooth || !firstGlyph.attachedEntity || !firstGlyph.overlay {
		t.Fatalf("EF_BEGINASURA11 first glyph = %+v", firstGlyph)
	}
	if firstOut.textureFile != "effect/asura11.tga" || firstOut.duration != 400*time.Millisecond || firstOut.delay != 1200*time.Millisecond || firstOut.posX != -8 || firstOut.sizeStart != effectTableSize(150) || firstOut.sizeEnd != effectTableSize(250) || firstOut.fadeIn || !firstOut.fadeOut || firstOut.duplicate != 0 {
		t.Fatalf("EF_BEGINASURA11 first fade-out glyph = %+v", firstOut)
	}
	if lastOut.textureFile != "effect/asura16.tga" || lastOut.delay != 1700*time.Millisecond || lastOut.posX != 8 || lastOut.sizeStart != effectTableSize(150) || lastOut.sizeEnd != effectTableSize(250) {
		t.Fatalf("EF_BEGINASURA11 last glyph = %+v", lastOut)
	}
}

func TestRobrowserTarotCardsFiveHundredToFiveFiftyMatchTableRows(t *testing.T) {
	ids := []int{
		effectTarotCard1,
		effectTarotCard2,
		effectTarotCard3,
		effectTarotCard4,
		effectTarotCard5,
		effectTarotCard6,
		effectTarotCard7,
		effectTarotCard8,
		effectTarotCard9,
		effectTarotCard10,
		effectTarotCard11,
		effectTarotCard12,
		effectTarotCard13,
		effectTarotCard14,
	}
	for i, id := range ids {
		name := fmt.Sprintf("EF_TAROTCARD%d", i+1)
		spec, ok := worldEffectSpecForID(id)
		if !ok || spec.duration != 3*time.Second || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\priest_slowpoison.wav" {
			t.Fatalf("%s sfx = %#v", name, spec.sfx)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != fmt.Sprintf("effect/tarot%02d.tga", i+1) || component.duration != 3*time.Second || component.alphaMax != 1 || !component.attachedEntity || !component.fadeIn || !component.fadeOut || component.posZ != 4 || component.sizeStart != effectTableSize(100) || component.sizeEnd != effectTableSize(70) || !component.sizeSmooth {
			t.Fatalf("%s component = %+v", name, component)
		}
	}
}

func TestRobrowserSimpleEffectsFiveFiftyToSixHundredMatchTableRows(t *testing.T) {
	stin2, ok := worldEffectSpecForID(effectStin2)
	if !ok || len(stin2.components) != 0 || len(stin2.sfx) != 5 || len(stin2.sfxDelays) != 5 {
		t.Fatalf("EF_STIN2 spec = %+v ok=%t", stin2, ok)
	}
	for i := range stin2.sfx {
		if stin2.sfx[i] != "effect\\t_날라차기.wav" || stin2.sfxDelays[i] != time.Duration(i)*200*time.Millisecond {
			t.Fatalf("EF_STIN2 sound %d = %q delay %s", i, stin2.sfx[i], stin2.sfxDelays[i])
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_STIN3", effectStin3, "effect\\t_에너지방출.wav"},
		{"EF_KAIZEL", effectKaizel, "effect\\priest_resurrection.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_HFLIMOON1", effectHfliMoon1, "moonlight_1", "effect\\h_moonlight_1.wav"},
		{"EF_HFLIMOON2", effectHfliMoon2, "moonlight_2", "effect\\h_moonlight_2.wav"},
		{"EF_HFLIMOON3", effectHfliMoon3, "moonlight_3", "effect\\h_moonlight_3.wav"},
		{"EF_HO_UP", effectHoUp, "h_levelup", ""},
		{"EF_HAMIDEFENCE", effectHamiDefence, "defense", ""},
		{"EF_FOOD01", effectStatFoodSTR, "food_str", ""},
		{"EF_FOOD02", effectStatFoodINT, "food_int", ""},
		{"EF_FOOD03", effectStatFoodVIT, "food_vit", ""},
		{"EF_FOOD04", effectStatFoodAGI, "food_agi", ""},
		{"EF_FOOD05", effectStatFoodDEX, "food_dex", ""},
		{"EF_FOOD06", effectStatFoodLUK, "food_luk", ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		file      string
		wav       string
		direction bool
	}{
		{"EF_HAMICASTLE", effectHamiCastle, "캐슬링", "", false},
		{"EF_HAMIBLOOD", effectHamiBlood, "블러드러스트", "", false},
		{"EF_ITEM_THUNDER", effectItemThunder, "item_thunder", "", false},
		{"EF_ITEM_CLOUD", effectItemCloud, "item_cloud", "", false},
		{"EF_ITEM_CURSE", effectItemCurse, "item_curse", "", false},
		{"EF_ITEM_ZZZ", effectItemZZZ, "item_zzz", "_snore.wav", false},
		{"EF_ITEM_RAIN", effectItemRain, "item_rain", "", false},
		{"EF_M01", effectM01, "m_ef01", "", false},
		{"EF_M02", effectM02, "m_ef02", "", true},
		{"EF_M03", effectM03, "m_ef03", "", false},
		{"EF_M04", effectM04, "m_ef04", "", false},
		{"EF_M05", effectM05, "m_ef05", "dragon_breath.wav", false},
		{"EF_M06", effectM06, "m_ef06", "", false},
		{"EF_M07", effectM07, "m_ef07", "effect\\t_보조마법.wav", false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || !component.attachedEntity || component.spriteDirection != tc.direction {
			t.Fatalf("%s component = %+v, want attached SPR %q direction=%t", tc.name, component, tc.file, tc.direction)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserFunctionAndProjectileEffectsFiveFiftyToSixHundredMatchTableRows(t *testing.T) {
	quake, ok := worldEffectSpecForID(effectScreenQuake)
	if !ok || quake.duration != 200*time.Millisecond || quake.cameraShake != 200*time.Millisecond || len(quake.components) != 1 {
		t.Fatalf("EF_SCREEN_QUAKE spec = %+v ok=%t", quake, ok)
	}
	if component := quake.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || component.duration != 200*time.Millisecond || !component.attachedEntity {
		t.Fatalf("EF_SCREEN_QUAKE component = %+v", component)
	}

	throw, ok := worldEffectSpecForID(effectThrowItem6)
	if !ok || throw.duration != 200*time.Millisecond || len(throw.components) != 1 {
		t.Fatalf("EF_THROWITEM6 spec = %+v ok=%t", throw, ok)
	}
	component := throw.components[0]
	if component.kind != effectComponent3D || component.textureFile != "유저인터페이스/item/베넘나이프.bmp" || component.duration != 200*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || component.angleStart != 180 || component.angleEnd != 540 || component.posZ != 1 || component.sizeStart != effectTableSize(30) || component.sizeEnd != effectTableSize(30) || !component.attachedEntity {
		t.Fatalf("EF_THROWITEM6 component = %+v", component)
	}
}

func TestRobrowserSimpleEffectsSixHundredToSixFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		rand     bool
	}{
		{"EF_FIREHIT2", effectFireHit2, "firehit%d", "", true, true},
		{"EF_COOKING_OK", effectCookingOK, "cook_suc", "_heal_effect.wav", true, false},
		{"EF_COOKING_FAIL", effectCookingFail, "cook_fail", "caramel_die.wav", true, false},
		{"EF_KOUENKA", effectKouenka, "firehit", "effect\\ef_firearrow%d.wav", true, true},
		{"EF_HYOUSENSOU", effectHyousensou, "freeze", "effect\\ef_icearrow%d.wav", true, true},
		{"EF_THUNDERSTORM2", effectThunderStorm2, "setsudan", "effect\\ef_thunderstorm.wav", true, false},
		{"EF_BAKU", effectBaku, "fire dragon", "effect\\폭염룡.wav", false, false},
		{"EF_HYOUSYOURAKU", effectHyousyouraku, "icy", "effect\\빙정락.wav", false, false},
		{"EF_TRACKCASTING", effectTrackCasting, "트랙킹", "", true, false},
		{"EF_BULLSEYE", effectBullseye, "불스아이", "", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.rand {
			if component.strRandMin != 1 || component.strRandMax != 3 {
				t.Fatalf("%s STR rand = %d..%d", tc.name, component.strRandMin, component.strRandMax)
			}
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
		if tc.rand && (spec.sfxRandMin != 1 || spec.sfxRandMax != 3) {
			t.Fatalf("%s sfx rand = %d..%d", tc.name, spec.sfxRandMin, spec.sfxRandMax)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		file      string
		wav       string
		attached  bool
		stop      bool
		repeat    bool
		direction bool
	}{
		{"EF_NPC_STOP2", effectNPCStop2, "cconfine", "effect\\ef_hit6.wav", true, true, false, false},
		{"EF_FVOICE", effectFVoice, "fvoice", "amon_ra_die01.wav", false, false, false, false},
		{"EF_WINK", effectWink, "wink", "", false, false, false, false},
		{"EF_KIRIKAGE", effectKirikage, "그림자베기", "effect\\그림자베기.wav", true, false, false, false},
		{"EF_TATAMI", effectTatami, "다다미 뒤집기", "effect\\다다미뒤집기.wav", true, false, false, false},
		{"EF_KASUMIKIRI", effectKasumikiri, "안개베기", "effect\\안개베기.wav", true, false, false, false},
		{"EF_ISSEN", effectIssen, "일섬", "effect\\일섬.wav", true, false, false, false},
		{"EF_KAEN", effectKaen, "화염진", "effect\\화염진.wav", true, false, true, false},
		{"EF_DESPERADO", effectDesperado, "데스페라도", "effect\\데스페라도.wav", true, false, false, false},
		{"EF_LIGHTNING_S", effectLightningS, "라이트닝스피어", "", false, false, false, false},
		{"EF_BLIND_S", effectBlindS, "블라인드스피어", "", false, false, false, false},
		{"EF_POISON_S", effectPoisonS, "포이즌스피어", "", false, false, false, false},
		{"EF_FREEZING_S", effectFreezingS, "프리징스피어", "", false, false, false, false},
		{"EF_FLARE_S", effectFlareS, "플레어스피어", "", false, false, false, false},
		{"EF_RAPIDSHOWER", effectRapidShower, "래피드샤워", "effect\\래피드샤워.wav", true, false, false, false},
		{"EF_MAGICALBULLET", effectMagicalBullet, "매지컬불릿", "effect\\매지컬블릿.wav", true, false, false, false},
		{"EF_SPREADATTACK", effectSpreadAttack, "스프레드", "", true, false, false, true},
		{"EF_TRACKING", effectTracking, "트래킹", "", true, false, false, false},
		{"EF_TRIPLEACTION", effectTripleAction, "트리플액션", "effect\\트리플액션.wav", true, false, false, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || component.attachedEntity != tc.attached || component.spriteStopAtEnd != tc.stop || component.repeat != tc.repeat || component.spriteDirection != tc.direction {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	hapgyeok, ok := worldEffectSpecForID(effectHapgyeok)
	if !ok || len(hapgyeok.components) != 2 {
		t.Fatalf("EF_HAPGYEOK spec = %+v ok=%t", hapgyeok, ok)
	}
	if len(hapgyeok.sfx) != 1 || hapgyeok.sfx[0] != "effect\\itempokjuk.wav" {
		t.Fatalf("EF_HAPGYEOK sfx = %#v", hapgyeok.sfx)
	}
	if spr, str := hapgyeok.components[0], hapgyeok.components[1]; spr.kind != effectComponentSPR || spr.spriteFile != "합격_" || !spr.attachedEntity || str.kind != effectComponentSTR || str.strFile != "itempokjuk" || !str.attachedEntity {
		t.Fatalf("EF_HAPGYEOK components = %+v", hapgyeok.components)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_STIN4", effectStin4, "effect\\풍인.wav"},
		{"EF_RG_COIN3", effectRGCoin3, "effect\\디스암.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserProjectileEffectsSixHundredToSixFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      int
		texture string
		size    float64
	}{
		{"EF_THROWITEM7", effectThrowItem7, "유저인터페이스/item/수리검.bmp", 30},
		{"EF_THROWITEM8", effectThrowItem8, "유저인터페이스/item/쿠나이_독.bmp", 30},
		{"EF_THROWITEM9", effectThrowItem9, "유저인터페이스/item/풍마_뇌우.bmp", 30},
		{"EF_THROWITEM10", effectThrowItem10, "effect/coin_a.bmp", 20},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 200*time.Millisecond || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\닌자_던지기.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != tc.texture || component.duration != 200*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || component.angleStart != 180 || component.angleEnd != 540 || component.posZ != 1 || component.sizeStart != effectTableSize(tc.size) || component.sizeEnd != effectTableSize(tc.size) || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserFuncEffectsSixHundredToSixFiftyMatchTableRows(t *testing.T) {
	dust, ok := worldEffectSpecForID(effectBash3D5)
	if !ok || dust.duration != 175*time.Millisecond || len(dust.components) != 3 {
		t.Fatalf("EF_BASH3D5 spec = %+v ok=%t", dust, ok)
	}
	if len(dust.sfx) != 1 || dust.sfx[0] != "effect\\bash3d5.wav" {
		t.Fatalf("EF_BASH3D5 sfx = %#v", dust.sfx)
	}
	body, first, second := dust.components[0], dust.components[1], dust.components[2]
	if body.kind != effectComponentFUNC || body.funcName != "Bash3D5" || !body.attachedEntity {
		t.Fatalf("EF_BASH3D5 body = %+v", body)
	}
	for i, component := range []worldEffectComponent{first, second} {
		if component.kind != effectComponentCylinder || component.textureName != "alpha_center" || component.duration != 175*time.Millisecond || component.duplicate != 6 || component.alphaMax != 0.6 || !component.fade || component.angleX != -90 || component.angleZRandom != 360 || !component.fixedPerspective || component.posZ != 1.5 || component.height != 0 || component.bottomSize != 0.01 || component.animation != 2 || !component.attachedEntity || component.totalCircleSides != 30 || component.circleSides != 1 {
			t.Fatalf("EF_BASH3D5 cylinder %d = %+v", i, component)
		}
	}
	if first.topSize != 4.5 || second.topSize != 7.2 {
		t.Fatalf("EF_BASH3D5 top sizes = %.1f %.1f", first.topSize, second.topSize)
	}

	chookgi, ok := worldEffectSpecForID(effectChookgi3)
	if !ok || chookgi.duration != 5*time.Minute || len(chookgi.components) != 1 {
		t.Fatalf("EF_CHOOKGI3 spec = %+v ok=%t", chookgi, ok)
	}
	if sphere := chookgi.components[0]; sphere.kind != effectComponentFUNC || sphere.funcName != "SpiritSphere" || sphere.funcAdapter != effectFuncSpiritSphere || sphere.textureFile != "effect/thunder_center.bmp" || sphere.duplicate != 5 || !sphere.attachedEntity {
		t.Fatalf("EF_CHOOKGI3 component = %+v", sphere)
	}
}

func TestRobrowserEffectsSixFiftyToSevenHundredMatchTableRows(t *testing.T) {
	earthquake, ok := worldEffectSpecForID(effectNPCEarthquake)
	if !ok || earthquake.cameraShake != 650*time.Millisecond || len(earthquake.components) != 2 {
		t.Fatalf("EF_NPC_EARTHQUAKE spec = %+v ok=%t", earthquake, ok)
	}
	if len(earthquake.sfx) != 1 || earthquake.sfx[0] != "effect\\earth_quake.wav" {
		t.Fatalf("EF_NPC_EARTHQUAKE sfx = %#v", earthquake.sfx)
	}
	if spr, quake := earthquake.components[0], earthquake.components[1]; spr.kind != effectComponentSPR || spr.spriteFile != "어스퀘이크" || !spr.attachedEntity || quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || quake.duplicate != 3 || quake.duplicateDelay != 35*time.Millisecond || !quake.attachedEntity {
		t.Fatalf("EF_NPC_EARTHQUAKE components = %+v", earthquake.components)
	}

	dragon, ok := worldEffectSpecForID(effectDragonFear)
	if !ok || dragon.cameraShake != 650*time.Millisecond || len(dragon.components) != 2 {
		t.Fatalf("EF_DRAGONFEAR spec = %+v ok=%t", dragon, ok)
	}
	if len(dragon.sfx) != 1 || dragon.sfx[0] != "effect\\dragonfear.wav" {
		t.Fatalf("EF_DRAGONFEAR sfx = %#v", dragon.sfx)
	}
	if str, quake := dragon.components[0], dragon.components[1]; str.kind != effectComponentSTR || str.strFile != "dragon_h" || !str.attachedEntity || quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || !quake.attachedEntity {
		t.Fatalf("EF_DRAGONFEAR components = %+v", dragon.components)
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_BLEEDING", effectWideBleeding, "wideb", "effect\\wideb.wav"},
		{"EF_WIDECONFUSE", effectWideConfuse, "dfear", "effect\\dragonfear.wav"},
		{"EF_CRITICALWOUND", effectCriticalWound, "cwound", ""},
		{"EF_FLOWERLEAF", effectFlowerLeaf, "flower_leaf", ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
	}{
		{"EF_BOTTOM_RUNNER", effectBottomRunner, "effect/hanmoon1.tga"},
		{"EF_BOTTOM_TRANSFER", effectBottomTransfer, "effect/hanmoon2.tga"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 1500*time.Millisecond || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one ground texture component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != "GroundTexture" || component.funcAdapter != effectFuncGroundTexture || component.textureFile != tc.texture || component.sizeStart != 1 || component.sizeEnd != 1 || component.posZ != 0.05 || !component.blendAdditive || component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	evil, ok := worldEffectSpecForID(effectBottomEvilLand)
	if !ok || evil.duration != 1500*time.Millisecond || len(evil.components) != 2 {
		t.Fatalf("EF_BOTTOM_EVILLAND spec = %+v ok=%t", evil, ok)
	}
	tile, curse := evil.components[0], evil.components[1]
	if tile.kind != effectComponentFUNC || tile.funcName != "FlatColorTile" || tile.funcAdapter != effectFuncFlatColorTile || tile.color != (color.RGBA{R: 160, G: 160, B: 160, A: 51}) || tile.sizeStart != 1 || tile.attachedEntity {
		t.Fatalf("EF_BOTTOM_EVILLAND tile = %+v", tile)
	}
	if curse.kind != effectComponentFUNC || curse.funcName != "GroundTexture" || curse.funcAdapter != effectFuncGroundTexture || curse.textureFile != "effect/curse.bmp" || curse.sizeStart != 1 || curse.sizeEnd != 1 || curse.alphaMax != 0.7 || curse.posZ != 0.4 || !curse.blendAdditive || curse.attachedEntity {
		t.Fatalf("EF_BOTTOM_EVILLAND curse = %+v", curse)
	}

	guard, ok := worldEffectSpecForID(effectGuard3)
	if !ok || len(guard.components) != 0 || guard.duration != 500*time.Millisecond {
		t.Fatalf("EF_GUARD3 spec = %+v ok=%t, want sound-only", guard, ok)
	}
	if len(guard.sfx) != 1 || guard.sfx[0] != "effect\\kyrie_guard.wav" {
		t.Fatalf("EF_GUARD3 sfx = %#v", guard.sfx)
	}
}

func TestHighWizardStringKeyEffectsMatchRobrowser(t *testing.T) {
	magicPower, ok := worldEffectSpecForID(effectMagicPower)
	if !ok || magicPower.duration != 500*time.Millisecond || len(magicPower.components) != 0 {
		t.Fatalf("ef_magicpower spec = %+v ok=%t, want sound-only", magicPower, ok)
	}
	if len(magicPower.sfx) != 1 || magicPower.sfx[0] != "effect\\마법력 증폭.wav" {
		t.Fatalf("ef_magicpower sfx = %#v", magicPower.sfx)
	}

	gravitation, ok := worldEffectSpecForID(effectGravitation)
	if !ok || gravitation.duration != 1500*time.Millisecond || gravitation.cameraShake != 200*time.Millisecond || len(gravitation.components) != 2 {
		t.Fatalf("522_ground spec = %+v ok=%t", gravitation, ok)
	}
	tile, lens := gravitation.components[0], gravitation.components[1]
	if tile.kind != effectComponentFUNC || tile.funcName != "FlatColorTile" || tile.funcAdapter != effectFuncFlatColorTile || tile.color != (color.RGBA{R: 255, G: 255, B: 255, A: 51}) || tile.sizeStart != 1 || !tile.attachedEntity {
		t.Fatalf("522_ground tile = %+v", tile)
	}
	if lens.kind != effectComponentFUNC || lens.funcName != "GroundTexture" || lens.funcAdapter != effectFuncGroundTexture || lens.textureFile != "effect/lens_w.bmp" || lens.sizeStart != 0.5 || lens.sizeEnd != 0.5 || lens.alphaMax != 0.7 || lens.posZ != 0.4 || !lens.blendAdditive || !lens.attachedEntity {
		t.Fatalf("522_ground lens = %+v", lens)
	}
}

func TestRobrowserFirecrackerBannersSixFiftyToSevenHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_POK_LOVE", effectFirecracker2, "폭죽_러브"},
		{"EF_POK_WHITE", effectFirecracker3, "폭죽_화이트데이"},
		{"EF_POK_VALEN", effectFirecracker4, "폭죽_발렌타인"},
		{"EF_POK_BIRTH", effectFirecracker5, "폭죽_생일"},
		{"EF_POK_CHRISTMAS", effectFirecracker6, "폭죽_크리스마스"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want SPR banner plus STR itempokjuk", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\itempokjuk.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		if spr, str := spec.components[0], spec.components[1]; spr.kind != effectComponentSPR || spr.spriteFile != tc.file || !spr.attachedEntity || str.kind != effectComponentSTR || str.strFile != "itempokjuk" || !str.attachedEntity {
			t.Fatalf("%s components = %+v", tc.name, spec.components)
		}
	}
}

func TestRobrowserSimpleEffectsSevenHundredToSevenFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
	}{
		{"EF_ITEM315", effectItem315, "mobile_ef02", "", true},
		{"EF_ITEM316", effectItem316, "mobile_ef01", "", true},
		{"EF_ITEM317", effectItem317, "mobile_ef03", "", true},
		{"EF_STORM_MIN", effectStormMin, "storm_min", "effect\\wizard_stormgust.wav", true},
		{"EF_POK_JAP", effectFirecracker7, "pokjuk_jap", "", false},
		{"EF_ADO_STR", effectAdoramus, "ado", "effect\\ab_adoramus.wav", true},
		{"EF_IGN_STR", effectIgnitionBreak, "이그니션브레이크", "effect\\wl_jackfrost.wav", true},
		{"EF_CRIMSON_STR", effectCrimsonRock, "crimson_r", "effect\\crimson_r.wav", true},
		{"EF_HELL_STR", effectHellInferno, "hell_in", "", true},
		{"EF_DHOWL_STR", effectDragonHowling, "dragon_h", "dragon_h.wav", true},
		{"EF_CHAINL_STR", effectChainLightning, "chainlight", "effect\\chainlight.wav", true},
		{"EF_AIMED_STR", effectAimedBolt, "aimed", "", true},
		{"EF_ARROWSTORM_STR", effectArrowStorm, "arrowstorm", "", true},
		{"EF_LAULAMUS_STR", effectLaulamus, "laulamus", "", true},
		{"EF_LAUAGNUS_STR", effectLauagnus, "lauagnus", "", true},
		{"EF_MILSHIELD_STR", effectMillenniumShield, "mil_shield", "", true},
		{"EF_CONCENTRATION2", effectConcentration2, "concentration", "", true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	wewish, ok := worldEffectSpecForID(effectChristmasCarol)
	if !ok || len(wewish.sfx) != 1 || wewish.sfx[0] != "effect\\wewish.wav" {
		t.Fatalf("EF_WEWISH spec = %+v ok=%t", wewish, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_FORESTLIGHT5", effectForestLight5, "effect\\ab_renovation.wav"},
		{"EF_FROSTMYSTY", effectFrostMisty, "effect\\t_에나지방출.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	marsh, ok := worldEffectSpecForID(effectMarshOfAbyss)
	if !ok || len(marsh.components) != 1 {
		t.Fatalf("EF_SPR_MASH spec = %+v ok=%t, want one SPR component", marsh, ok)
	}
	if component := marsh.components[0]; component.kind != effectComponentSPR || component.spriteFile != "mashofa" || component.attachedEntity {
		t.Fatalf("EF_SPR_MASH component = %+v", component)
	}
}

func TestRobrowserCylinderEffectsSevenHundredToSevenFiftyMatchTableRows(t *testing.T) {
	for _, id := range []int{effectBottomBlue, effectBottomBlue2} {
		spec, ok := worldEffectSpecForID(id)
		if !ok || spec.duration != 20*time.Second || len(spec.components) != 4 {
			t.Fatalf("bottom blue effect %d spec = %+v ok=%t", id, spec, ok)
		}
		for i, component := range spec.components {
			if component.kind != effectComponentCylinder || component.textureName != "alpha_down" || component.duration != 20*time.Second || component.totalCircleSides != 4 || component.circleSides != 4 || !component.rotateWithCamera || component.blendMode != 2 || !component.blendAdditive || !component.attachedEntity {
				t.Fatalf("bottom blue effect %d component %d = %+v", id, i, component)
			}
		}
		if first := spec.components[0]; first.bottomSize != 1.5 || first.topSize != 1.5 || first.height != 2 || first.alphaMax != 40.0/255.0 || first.angleY != 0 || first.color != (color.RGBA{R: 51, G: 153, B: 255, A: 255}) {
			t.Fatalf("bottom blue first component = %+v", first)
		}
		if second := spec.components[1]; second.bottomSize != 1.58 || second.height != 2.1 || second.alphaMax != 32.0/255.0 || second.angleY != 10 {
			t.Fatalf("bottom blue second component = %+v", second)
		}
		if third := spec.components[2]; third.bottomSize != 1.65 || third.alphaMax != 15.0/255.0 || third.angleY != 26.6 || third.color != (color.RGBA{R: 25, G: 102, B: 255, A: 255}) {
			t.Fatalf("bottom blue third component = %+v", third)
		}
		if fourth := spec.components[3]; fourth.bottomSize != 1.65 || fourth.alphaMax != 15.0/255.0 || fourth.angleY != 79.8 {
			t.Fatalf("bottom blue fourth component = %+v", fourth)
		}
	}

	judex, ok := worldEffectSpecForID(effectFirePillarOn2)
	if !ok || judex.duration != time.Second || len(judex.components) != 3 {
		t.Fatalf("EF_FIREPILLARON2 spec = %+v ok=%t", judex, ok)
	}
	if len(judex.sfx) != 1 || judex.sfx[0] != "effect\\ab_judex.wav" {
		t.Fatalf("EF_FIREPILLARON2 sfx = %#v", judex.sfx)
	}
	want := []struct {
		bottom float64
		top    float64
		height float64
	}{
		{0.4, 0.5, 3.5},
		{0.45, 0.75, 2.5},
		{0.5, 1, 1.5},
	}
	for i, component := range judex.components {
		if component.kind != effectComponentCylinder || component.textureName != "ring_white" || component.duration != time.Second || component.attachedEntity || !component.rotate || component.bottomSize != want[i].bottom || component.topSize != want[i].top || component.height != want[i].height {
			t.Fatalf("EF_FIREPILLARON2 component %d = %+v", i, component)
		}
	}
}

func TestRobrowserEarthWallEffectSevenHundredToSevenFiftyMatchesTableRow(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectEarthWall)
	if !ok || spec.duration != time.Second || spec.cameraShake != 200*time.Millisecond || len(spec.components) != 2 {
		t.Fatalf("EF_EARTHWALL spec = %+v ok=%t", spec, ok)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("EF_EARTHWALL sfx = %#v", spec.sfx)
	}
	horn, quake := spec.components[0], spec.components[1]
	if horn.kind != effectComponentQuadHorn || horn.textureFile != "effect/stone.bmp" || horn.duration != time.Second || horn.quadHornHeightMin != 0.75 || horn.quadHornHeightMax != 1.2 || horn.quadHornOffsetXMin != 0.2 || horn.quadHornOffsetXMax != 0.2 || horn.quadHornOffsetYMin != 0.2 || horn.quadHornOffsetYMax != 0.2 || horn.quadHornOffsetZ != -0.1 || horn.quadHornBottomMin != 0.4 || horn.quadHornBottomMax != 0.9 || horn.blendMode != 1 || horn.quadHornRotateYMin != 1 || horn.quadHornRotateYMax != 360 || horn.quadHornRotateZMin != -8 || horn.quadHornRotateZMax != 8 || horn.animation != 3 || horn.quadHornAnimSpeed != 250*time.Millisecond || !horn.quadHornAnimOut {
		t.Fatalf("EF_EARTHWALL horn = %+v", horn)
	}
	if quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || !quake.attachedEntity {
		t.Fatalf("EF_EARTHWALL quake = %+v", quake)
	}
}

func TestRobrowserEffectsSevenFiftyToEightHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_POTION_BERSERK2", effectBerserkPotion2, "버서크"},
		{"EF_CRASHAXE", effectCrashAxe, "powerswing"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want STR %q attached", tc.name, component, tc.file)
		}
		if len(spec.sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
		}
	}

	spin, ok := worldEffectSpecForID(effectCastSpin2)
	if !ok || spin.duration != 500*time.Millisecond || len(spin.components) != 1 {
		t.Fatalf("EF_CASTSPIN2 spec = %+v ok=%t", spin, ok)
	}
	if component := spin.components[0]; component.kind != effectComponentFUNC || component.funcName != "CastSpin2" || !component.attachedEntity {
		t.Fatalf("EF_CASTSPIN2 component = %+v", component)
	}

	stasis, ok := worldEffectSpecForID(effectStasis)
	if !ok || stasis.duration != 500*time.Millisecond || len(stasis.components) != 0 {
		t.Fatalf("EF_STASIS spec = %+v ok=%t", stasis, ok)
	}
	if len(stasis.sfx) != 1 || stasis.sfx[0] != "effect\\wl_stasis.wav" {
		t.Fatalf("EF_STASIS sfx = %#v", stasis.sfx)
	}
}

func TestRobrowserGlassWall3EffectMatchesTableRow(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectGlassWall3)
	if !ok || len(spec.components) != 4 {
		t.Fatalf("EF_GLASSWALL3 spec = %+v ok=%t", spec, ok)
	}
	tint := color.RGBA{R: 153, G: 255, B: 153, A: 255}
	first := spec.components[0]
	if first.kind != effectComponentCylinder || first.textureName != "magic_green" || first.color != tint {
		t.Fatalf("EF_GLASSWALL3 first identity = %+v", first)
	}
	if first.duration != 500*time.Millisecond || first.alphaMax != 0.4 || first.animation != 4 || first.bottomSize != 2.4 || first.topSize != 3.9 || first.height != 0.1 || first.posZ != 0.1 {
		t.Fatalf("EF_GLASSWALL3 first timing/geometry = %+v", first)
	}
	if first.duplicate != 150 || first.duplicateDelay != 200*time.Millisecond || !first.fadeOut || !first.rotate || first.blendMode != 2 || !first.blendAdditive || !first.attachedEntity {
		t.Fatalf("EF_GLASSWALL3 first flags = %+v", first)
	}
	if got := worldEffectComponentDuration(spec, first); got != 30300*time.Millisecond {
		t.Fatalf("EF_GLASSWALL3 first resolved duration = %s, want 30300ms", got)
	}

	for i, want := range []struct {
		bottom float64
		top    float64
		height float64
		alpha  float64
		posZ   float64
		sides  int
		circle int
	}{
		{0.6, 0.6, 7, 0.4, 0, 32, 32},
		{0.8, 0.8, 6, 0.4, 0, 32, 32},
		{1, 1, 1, 0.5, 2, 20, 10},
	} {
		component := spec.components[i+1]
		texture := "magic_green"
		if i == 2 {
			texture = "alpha1"
		}
		if component.kind != effectComponentCylinder || component.textureName != texture || component.color != tint {
			t.Fatalf("EF_GLASSWALL3 component %d identity = %+v", i+1, component)
		}
		if component.duration != 30*time.Second || component.alphaMax != want.alpha || component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height || component.posZ != want.posZ {
			t.Fatalf("EF_GLASSWALL3 component %d geometry = %+v", i+1, component)
		}
		if !component.fade || !component.rotate || component.blendMode != 2 || !component.blendAdditive || !component.attachedEntity || component.totalCircleSides != want.sides || component.circleSides != want.circle {
			t.Fatalf("EF_GLASSWALL3 component %d flags = %+v", i+1, component)
		}
	}
}

func TestRobrowserRollingCutterCounterEffectsMatchTableRows(t *testing.T) {
	ids := []int{
		effectRolling1,
		effectRolling2,
		effectRolling3,
		effectRolling4,
		effectRolling5,
		effectRolling6,
		effectRolling7,
		effectRolling8,
		effectRolling9,
		effectRolling10,
	}
	for i, id := range ids {
		spec, ok := worldEffectSpecForID(id)
		if !ok || spec.duration != time.Second || len(spec.components) != 5 {
			t.Fatalf("EF_ROLLING%d spec = %+v ok=%t", i+1, spec, ok)
		}
		texture := fmt.Sprintf("effect/회전카운터%d.tga", i+1)
		for j, want := range []struct {
			alpha     float64
			color     color.RGBA
			sizeStart float64
			blendMode int
			additive  bool
		}{
			{1, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 200, 1, false},
			{0.2, color.RGBA{R: 178, G: 178, B: 255, A: 255}, 220, 2, true},
			{0.2, color.RGBA{R: 127, G: 127, B: 255, A: 255}, 240, 2, true},
			{0.2, color.RGBA{R: 76, G: 76, B: 255, A: 255}, 260, 2, true},
			{0.2, color.RGBA{R: 25, G: 25, B: 255, A: 255}, 280, 2, true},
		} {
			component := spec.components[j]
			if component.kind != effectComponent3D || component.textureFile != texture || component.duration != time.Second {
				t.Fatalf("EF_ROLLING%d component %d identity = %+v", i+1, j, component)
			}
			if component.alphaMax != want.alpha || component.color != want.color || component.sizeStart != effectTableSize(want.sizeStart) || component.sizeEnd != effectTableSize(20) {
				t.Fatalf("EF_ROLLING%d component %d visual = %+v", i+1, j, component)
			}
			if !component.fadeIn || !component.fadeOut || component.posZ != 4 || !component.sizeSmooth || component.blendMode != want.blendMode || component.blendAdditive != want.additive || !component.attachedEntity {
				t.Fatalf("EF_ROLLING%d component %d flags = %+v", i+1, j, component)
			}
		}
	}
}

func TestRobrowserBottomBasilica2EffectMatchesTableRow(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBottomBasilica2)
	if !ok || spec.duration != 20*time.Second || len(spec.components) != 4 {
		t.Fatalf("EF_BOTTOM_BASILICA2 spec = %+v ok=%t", spec, ok)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\wl_whiteimprison.wav" {
		t.Fatalf("EF_BOTTOM_BASILICA2 sfx = %#v", spec.sfx)
	}
	for i, want := range []struct {
		size   float64
		height float64
		alpha  float64
		angleY float64
	}{
		{2.2, 3.0, 65.0 / 255.0, 0},
		{2.25, 3.1, 65.0 / 255.0, 10},
		{2.3, 3.0, 15.0 / 255.0, 0},
		{2.3, 3.0, 15.0 / 255.0, 53.2},
	} {
		component := spec.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "alpha_down" || component.duration != 20*time.Second {
			t.Fatalf("EF_BOTTOM_BASILICA2 component %d identity = %+v", i, component)
		}
		if component.totalCircleSides != 4 || component.circleSides != 4 || component.bottomSize != want.size || component.topSize != want.size || component.height != want.height || math.Abs(component.alphaMax-want.alpha) > 0.0001 || component.angleY != want.angleY {
			t.Fatalf("EF_BOTTOM_BASILICA2 component %d geometry = %+v", i, component)
		}
		if component.blendMode != 2 || !component.blendAdditive || !component.rotateWithCamera || !component.attachedEntity {
			t.Fatalf("EF_BOTTOM_BASILICA2 component %d flags = %+v", i, component)
		}
	}
}

func TestRobrowserEffectsEightHundredToEightFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_ENERVATION", effectEnervation, "enervation", ""},
		{"EF_ENERVATION2", effectEnervation2, "groomy", ""},
		{"EF_ENERVATION3", effectEnervation3, "ignorance", ""},
		{"EF_ENERVATION4", effectEnervation4, "laziness", "effect\\laziness.wav"},
		{"EF_ENERVATION5", effectEnervation5, "unlucky", ""},
		{"EF_ENERVATION6", effectEnervation6, "weakness", ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want STR %q attached", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_RECOGNIZED", effectRecognized, "effect\\wl_recognizedspell.wav"},
		{"EF_TETRA", effectTetra, "effect\\wl_tetravortex.wav"},
		{"EF_STRETCH", effectStretch, "effect\\bodypaint.wav"},
		{"EF_BOTTOM_MANHOLE", effectBottomManhole, "effect\\dimension.wav"},
		{"EF_MANHOLE", effectManhole, "effect\\manhole.wav"},
		{"EF_FORESTLIGHT6", effectForestLight6, "effect\\dimension.wav"},
		{"EF_BOTTOM_ANI", effectBottomAni, "effect\\chaospanic.wav"},
		{"EF_BOTTOM_MAELSTROM", effectBottomMaelstrom, "effect\\maelstrom.wav"},
		{"EF_BOTTOM_BLOODYLUST", effectBottomBloodyLust, "effect\\bloodylust.wav"},
		{"EF_HEAL_N", effectHealN, "effect\\기공포.wav"},
		{"EF_DANCE1", effectDance1, "effect\\수줍은하루의우울.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 500*time.Millisecond || len(spec.components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	tetraCasting, ok := worldEffectSpecForID(effectTetraCasting)
	if !ok || tetraCasting.duration != 500*time.Millisecond || len(tetraCasting.components) != 1 {
		t.Fatalf("EF_TETRACASTING spec = %+v ok=%t", tetraCasting, ok)
	}
	if component := tetraCasting.components[0]; component.kind != effectComponentFUNC || component.funcName != "TetraCasting" || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
		t.Fatalf("EF_TETRACASTING component = %+v", component)
	}

	chookgi, ok := worldEffectSpecForID(effectChookgiN)
	if !ok || chookgi.duration != 5*time.Minute || len(chookgi.components) != 1 {
		t.Fatalf("EF_CHOOKGI_N spec = %+v ok=%t", chookgi, ok)
	}
	if component := chookgi.components[0]; component.kind != effectComponentFUNC || component.funcName != "SpiritSphere" || component.funcAdapter != effectFuncSpiritSphere || !component.attachedEntity {
		t.Fatalf("EF_CHOOKGI_N component = %+v", component)
	}
}

func TestRobrowserEffectsEightFiftyToNineHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_RAIN_PARTICLE", effectRainParticle, "effect\\rainstorm.wav"},
		{"EF_CHEMICAL_V2", effectChemicalV2, "effect\\안식의자장가.wav"},
		{"EF_CIRCLEPOWER2", effectCirclePower2, "effect\\순환하는자연의소리.wav"},
		{"EF_SPR_PLANT2", effectSprPlant2, "effect\\워그와함께춤을.wav"},
		{"EF_SPR_PLANT3", effectSprPlant3, "effect\\마나의노래.wav"},
		{"EF_SPR_PLANT4", effectSprPlant4, "effect\\새터데이나이트피버.wav"},
		{"EF_SPR_PLANT5", effectSprPlant5, "effect\\레라드의이슬.wav"},
		{"EF_SPR_PLANT6", effectSprPlant6, "effect\\멜로디오브싱크.wav"},
		{"EF_SPR_PLANT7", effectSprPlant7, "effect\\비욘드오브워크라이.wav"},
		{"EF_SPR_PLANT8", effectSprPlant8, "effect\\언리미티드허밍보이스.wav"},
		{"EF_HEARTASURA", effectHeartAsura, "effect\\세이렌의목소리.wav"},
		{"EF_ELECTRIC4", effectElectric4, "effect\\sr_earthshaker.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 500*time.Millisecond || len(spec.components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
		wav     string
		tint    color.RGBA
	}{
		{"EF_BOT_REVERB", effectBotReverb, "effect/melody_b.bmp", "effect\\reverberation.wav", color.RGBA{R: 255, G: 153, B: 153, A: 255}},
		{"EF_BOT_REVERB2", effectBotReverb2, "effect/melody_a.bmp", "effect\\나락의노래.wav", color.RGBA{R: 153, G: 153, B: 255, A: 255}},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 100*time.Millisecond || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != tc.texture || component.color != tc.tint || component.duration != 100*time.Millisecond || component.alphaMax != 0.6 || !component.attachedEntity || !component.repeat || component.posZ != 0.5 || component.sizeStart != effectTableSize(50) || component.sizeEnd != effectTableSize(50) {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	secra, ok := worldEffectSpecForID(effectSecra2)
	if !ok || secra.duration != 1500*time.Millisecond || len(secra.components) != 10 {
		t.Fatalf("EF_SECRA2 spec = %+v ok=%t", secra, ok)
	}
	if len(secra.sfx) != 1 || secra.sfx[0] != "effect\\ab_ancilla.wav" {
		t.Fatalf("EF_SECRA2 sfx = %#v", secra.sfx)
	}
	firstSecra, lastSecra := secra.components[0], secra.components[len(secra.components)-1]
	if firstSecra.kind != effectComponent3D || firstSecra.textureFile != "effect/priest_spell.bmp" || firstSecra.color != (color.RGBA{R: 255, G: 140, B: 140, A: 255}) || firstSecra.duration != 1500*time.Millisecond || firstSecra.alphaMax != 0.3 || firstSecra.blendMode != 2 || !firstSecra.blendAdditive || !firstSecra.fadeIn || !firstSecra.fadeOut || !firstSecra.attachedEntity || firstSecra.posZ != 7 || firstSecra.sizeStart != effectTableSize(850) || firstSecra.sizeEnd != effectTableSize(100) || !firstSecra.sizeSmooth {
		t.Fatalf("EF_SECRA2 first component = %+v", firstSecra)
	}
	if lastSecra.color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) || lastSecra.sizeStart != effectTableSize(400) || lastSecra.sizeEnd != effectTableSize(100) {
		t.Fatalf("EF_SECRA2 last component = %+v", lastSecra)
	}

	glass, ok := worldEffectSpecForID(effectGlassWall4)
	if !ok || glass.duration != 30*time.Second || len(glass.components) != 3 {
		t.Fatalf("EF_GLASSWALL4 spec = %+v ok=%t", glass, ok)
	}
	if len(glass.sfx) != 1 || glass.sfx[0] != "effect\\ef_readyportal.wav" {
		t.Fatalf("EF_GLASSWALL4 sfx = %#v", glass.sfx)
	}
	tree, pulseA, pulseB := glass.components[0], glass.components[1], glass.components[2]
	if tree.kind != effectComponent3D || tree.textureFile != "effect/ef_epitree.tga" || tree.color != (color.RGBA{G: 255, A: 255}) || tree.duration != 30*time.Second || tree.alphaMax != 0.6 || tree.blendMode != 2 || !tree.blendAdditive || !tree.attachedEntity || tree.posZ != 7 || tree.sizeStart != effectTableSize(400) || tree.sizeEnd != effectTableSize(400) {
		t.Fatalf("EF_GLASSWALL4 tree = %+v", tree)
	}
	if pulseA.duration != 990*time.Millisecond || pulseA.duplicate != 15 || pulseA.duplicateDelay != 2*time.Second || pulseA.delay != 0 || pulseA.sizeStart != effectTableSize(380) || pulseA.sizeEnd != effectTableSize(420) {
		t.Fatalf("EF_GLASSWALL4 first pulse = %+v", pulseA)
	}
	if pulseB.delay != time.Second || pulseB.sizeStart != effectTableSize(420) || pulseB.sizeEnd != effectTableSize(380) {
		t.Fatalf("EF_GLASSWALL4 second pulse = %+v", pulseB)
	}

	bash, ok := worldEffectSpecForID(effectBash3D6)
	if !ok || bash.duration != 500*time.Millisecond || len(bash.components) != 3 {
		t.Fatalf("EF_BASH3D6 spec = %+v ok=%t", bash, ok)
	}
	if len(bash.sfx) != 1 || bash.sfx[0] != "effect\\bash3d.wav" {
		t.Fatalf("EF_BASH3D6 sfx = %#v", bash.sfx)
	}
	if body := bash.components[0]; body.kind != effectComponentFUNC || body.funcName != "Bash3D6" || body.funcAdapter != effectFuncUnknown || !body.attachedEntity {
		t.Fatalf("EF_BASH3D6 body = %+v", body)
	}
	for i, component := range bash.components[1:] {
		wantTop := 4.5
		if i == 1 {
			wantTop = 7.2
		}
		if component.kind != effectComponentCylinder || component.textureName != "alpha_center" || component.color != (color.RGBA{R: 76, G: 127, B: 255, A: 255}) || component.duration != 175*time.Millisecond || component.delay != 200*time.Millisecond || component.duplicate != 5 || component.alphaMax != 0.6 || !component.fade || component.angleX != -90 || component.angleZRandom != 360 || !component.fixedPerspective || component.posZ != 1.5 || component.bottomSize != 0.01 || component.topSize != wantTop || component.animation != 2 || !component.attachedEntity {
			t.Fatalf("EF_BASH3D6 cylinder %d = %+v", i, component)
		}
	}

	teiHit, ok := worldEffectSpecForID(effectTeiHit1T)
	if !ok || teiHit.duration != 350*time.Millisecond || len(teiHit.components) != 1 {
		t.Fatalf("EF_TEIHIT1T spec = %+v ok=%t", teiHit, ok)
	}
	if len(teiHit.sfx) != 1 || teiHit.sfx[0] != "effect\\mon_아수라 패황권.wav" {
		t.Fatalf("EF_TEIHIT1T sfx = %#v", teiHit.sfx)
	}
	component := teiHit.components[0]
	if component.kind != effectComponent3D || component.textureFile != "effect/lens1.tga" || component.color != (color.RGBA{R: 25, G: 25, B: 255, A: 255}) || component.duration != 250*time.Millisecond || component.delay != 100*time.Millisecond || component.duplicate != 24 || component.duplicateDelay != 0 || component.alphaMax != 0.8 || component.blendMode != 2 || !component.blendAdditive || !component.fadeIn || !component.fadeOut || !component.attachedEntity || component.posXEndRand != 40 || component.posYEndRand != 40 || component.sizeStartX != effectTableSize(10) || component.sizeStartY != effectTableSize(150) || component.sizeEndX != effectTableSize(10) || component.sizeEndY != effectTableSize(150) || !component.overlay || !component.rotateToTarget || !component.rotateWithCamera {
		t.Fatalf("EF_TEIHIT1T component = %+v", component)
	}
}

func TestRobrowserEffectsNineHundredToNineFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_PRIMECHARGE2", effectPrimeCharge2, "effect\\lg_prestige.wav"},
		{"EF_PRIMECHARGE3", effectPrimeCharge3, "effect\\lg_banding.wav"},
		{"EF_PRIMECHARGE4", effectPrimeCharge4, "effect\\lg_inspiration.wav"},
		{"EF_SPR_PLANT10", effectSprPlant10, "effect\\s사이킥웨이브.wav"},
		{"EF_COLDTHROW2", effectColdThrow2, "effect\\wl_jackfrost.wav"},
		{"EF_DEMONICFIRE4", effectDemonicFire4, "effect\\s워머.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 500*time.Millisecond || len(spec.components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_FIREWALL2", effectFireWall2, "firewall_per"},
		{"EF_SHOCKWAVE2", effectShockwave2, "hunter_shockwave_blue"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || len(spec.sfx) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component and no sound", tc.name, spec, ok)
		}
		if component := spec.components[0]; component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
		sfx     []string
		delays  []time.Duration
	}{
		{"EF_PRESSURE2", effectPressure2, "effect/shield.bmp", []string{"effect\\프레셔.wav", "effect\\lg_shieldpress.wav"}, []time.Duration{0, 500 * time.Millisecond}},
		{"EF_PRESSURE3", effectPressure3, "effect/cross1.bmp", []string{"effect\\프레셔.wav"}, nil},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 1001*time.Millisecond || len(spec.components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if spec.cameraShake != 0 || spec.cameraShakeDelay != 0 {
			t.Fatalf("%s camera shake = %s delay %s, want none", tc.name, spec.cameraShake, spec.cameraShakeDelay)
		}
		if !reflect.DeepEqual(spec.sfx, tc.sfx) || !reflect.DeepEqual(spec.sfxDelays, tc.delays) {
			t.Fatalf("%s sfx = %#v delays %#v", tc.name, spec.sfx, spec.sfxDelays)
		}

		first, second := spec.components[0], spec.components[1]
		if first.kind != effectComponent3D || first.textureFile != tc.texture || first.duration != 500*time.Millisecond || first.alphaMax != 0.6 || first.blendMode != 2 || !first.blendAdditive || !first.rotate || first.angleStart != 0 || first.angleEnd != -611 || first.posZ != 20 || first.posZEnd != 5 || first.sizeStart != effectTableSize(100) || first.sizeEnd != effectTableSize(100) || !first.attachedEntity {
			t.Fatalf("%s first component = %+v", tc.name, first)
		}
		if second.kind != effectComponent3D || second.textureFile != tc.texture || second.duration != 500*time.Millisecond || second.delay != 501*time.Millisecond || second.alphaMax != 0.6 || second.blendMode != 2 || !second.blendAdditive || !second.fadeOut || second.angleStart != -611 || second.angleEnd != -611 || second.posZ != 5 || second.sizeStart != effectTableSize(100) || second.sizeEnd != effectTableSize(100) || !second.attachedEntity {
			t.Fatalf("%s second component = %+v", tc.name, second)
		}
	}
}

func TestRobrowserEffectsNineFiftyToOneThousandMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_POISON_MIST", effectPoisonMist, "poison_mist"},
		{"EF_ERASER_CUTTER", effectEraserCutter, "eraser_cutter"},
		{"EF_LAVA_SLIDE", effectLavaSlide, "lava_slide"},
		{"EF_SONIC_CLAW", effectSonicClaw, "sonic_claw"},
		{"EF_TINDER_BREAKER", effectTinderBreaker, "tinder"},
		{"EF_MIDNIGHT_FRENZY", effectMidnightFrenzy, "mid_frenzy"},
		{"EF_VOLCANIC_ASH", effectVolcanicAsh, "vash00"},
		{"EF_2011RWC", effectRWC2011, "rwc2011"},
		{"EF_2011RWC2", effectRWC2011Two, "rwc2011_2"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || len(spec.sfx) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component and no sound", tc.name, spec, ok)
		}
		if component := spec.components[0]; component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserEffectsOneThousandToTenFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      int
		file    string
		wav     string
		randMin int
		randMax int
	}{
		{"EF_RUN_MAKE_OK", effectRunMakeOK, "rune_success", "", 0, 0},
		{"EF_RUN_MAKE_FAILURE", effectRunMakeFailure, "rune_fail", "", 0, 0},
		{"EF_MIRESULT_MAKE_OK", effectMIResultMakeOK, "changematerial_su", "", 0, 0},
		{"EF_MIRESULT_MAKE_FAIL", effectMIResultMakeFail, "changematerial_fa", "", 0, 0},
		{"EF_ALL_RAY_OF_PROTECTION", effectAllRayProtect, "guardian", "", 0, 0},
		{"EF_VENOMFOG", effectVenomFog, "bubble%d_1", "", 1, 4},
		{"EF_DUSTSTORM", effectDustStorm, "dust", "", 0, 0},
		{"EF_DANCE_BLADE_ATK", effectDanceBladeAtk, "dancingblade", "", 0, 0},
		{"EF_INVINCIBLEOFF2", effectInvincibleOff2, "invincibleoff2", "", 0, 0},
		{"EF_DEATHSUMMON", effectDeathSummon, "devil", "", 0, 0},
		{"EF_GC_DARKCROW", effectGCDarkCrow, "gc_darkcrow", "", 0, 0},
		{"EF_ALL_FULL_THROTTLE", effectAllFullThrottle, "all_full_throttle", "effect\\all_full_throttle.wav", 0, 0},
		{"EF_SR_FLASHCOMBO", effectSRFlashCombo, "sr_flashcombo", "effect\\sr_flashcombo.wav", 0, 0},
		{"EF_RK_LUXANIMA", effectRKLuxAnima, "rk_luxanima", "", 0, 0},
		{"EF_SO_ELEMENTAL_SHIELD", effectSOElemShield, "so_elemental_shield", "effect\\so_elemental_shield.wav", 0, 0},
		{"EF_AB_OFFERTORIUM", effectABOffertorium, "ab_offertorium", "effect\\ab_offertorium.wav", 0, 0},
		{"EF_WL_TELEKINESIS_INTENSE", effectWLTelekinesis, "wl_telekinesis_intense", "effect\\wl_telekinesis_intense.wav", 0, 0},
		{"EF_GN_ILLUSIONDOPING", effectGNIllusionDoping, "gn_illusiondoping", "effect\\gn_illusiondoping.wav", 0, 0},
		{"EF_NC_MAGMA_ERUPTION", effectNCMagmaEruption, "nc_magma_eruption", "effect\\nc_magma_eruption.wav", 0, 0},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
		} else if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.strRandMin != tc.randMin || component.strRandMax != tc.randMax || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserEffectsTenFiftyToElevenHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name        string
		id          int
		file        string
		texturePath string
		wav         string
	}{
		{"EF_NPC_CHILL", effectNPCChill, "chill", "", ""},
		{"EF_AB_OFFERTORIUM_RING", effectOffertoriumRing, "ab_offertorium_ring", "", ""},
		{"EF_HAMMER_OF_GOD", effectHammerOfGod, "stormgust", "", "effect\\RL_HAMMER_OF_GOD.wav"},
		{"EF_ACH_COMPLETE", effectAchComplete, "ach_complete/ppring3", "ach_complete/", ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
		} else if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.texturePath != tc.texturePath || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserEffectsPostElevenHundredMatchTableRows(t *testing.T) {
	body, ok := worldEffectSpecForID(effectBodyColor)
	if !ok || body.duration != 300*time.Millisecond || len(body.components) != 1 {
		t.Fatalf("EffectBodyColor spec = %+v ok=%t, want 300ms FUNC component", body, ok)
	}
	bodyComponent := body.components[0]
	if bodyComponent.kind != effectComponentFUNC || bodyComponent.funcName != "EffectBodyColor" || bodyComponent.funcAdapter != effectFuncBodyColor || !bodyComponent.attachedEntity {
		t.Fatalf("EffectBodyColor component = %+v", bodyComponent)
	}

	for _, tc := range []struct {
		name         string
		id           int
		file         string
		texturePath  string
		head         bool
		yOffset      float64
		renderBefore bool
	}{
		{"EF_BAKURETSU_HADOU", effectBakuretsuHadou, "bakuretsu_hadou/bakuretsu_hadou", "bakuretsu_hadou/", true, -50, false},
		{"EF_DIGITAL_SPACE", effectDigitalSpace, "digital_space/digital_space", "digital_space/", false, 0, true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 5*time.Minute || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want 5m SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || component.texturePath != tc.texturePath || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want SPR %q texturePath %q attached", tc.name, component, tc.file, tc.texturePath)
		}
		if !component.repeat || !component.spriteRepeat || component.spriteHead != tc.head || component.spriteYOffset != tc.yOffset || component.renderBefore != tc.renderBefore {
			t.Fatalf("%s component flags = %+v", tc.name, component)
		}
		if len(spec.sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
		}
	}

	for _, tc := range []struct {
		name         string
		id           int
		colorName    string
		renderBefore bool
	}{
		{"DROPEFFECT_PINK", dropEffectPink, "pink", true},
		{"DROPEFFECT_YELLOW", dropEffectYellow, "yellow", false},
		{"DROPEFFECT_PURPLE", dropEffectPurple, "purple", false},
		{"DROPEFFECT_BLUE", dropEffectBlue, "blue", false},
		{"DROPEFFECT_GREEN", dropEffectGreen, "green", false},
		{"DROPEFFECT_RED", dropEffectRed, "red", false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want two STR components", tc.name, spec, ok)
		}
		wantSFX := "effect\\drop_" + tc.colorName + ".wav"
		if len(spec.sfx) != 1 || spec.sfx[0] != wantSFX {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, wantSFX)
		}
		wantFiles := []string{
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName,
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "_bottom/dropitem_" + tc.colorName + "_bottom",
		}
		wantTexturePaths := []string{
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "/",
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "_bottom/",
		}
		for i, component := range spec.components {
			if component.kind != effectComponentSTR || component.strFile != wantFiles[i] || component.texturePath != wantTexturePaths[i] {
				t.Fatalf("%s component %d = %+v, want STR %q texturePath %q", tc.name, i, component, wantFiles[i], wantTexturePaths[i])
			}
			if component.attachedEntity || component.renderBefore != tc.renderBefore {
				t.Fatalf("%s component %d flags = %+v", tc.name, i, component)
			}
		}
	}

	for _, tc := range []struct {
		name        string
		id          int
		file        string
		texturePath string
	}{
		{"EF_NEW_SUCCESS", effectNewSuccess, "grade_enchant/new_success/new_success", "grade_enchant/new_success/"},
		{"EF_NEW_FAILURE", effectNewFailure, "grade_enchant/new_failed/new_failed", "grade_enchant/new_failed/"},
		{"EF_NEW_INTRO", effectNewIntro, "grade_enchant/new_intro/new_intro", "grade_enchant/new_intro/"},
		{"EF_UI_ENCHANT_INTRO_YELLOW", effectEnchantYellow, "ui_enchant/ui_intro_yellow/ui_intro_yellow", "ui_enchant/ui_intro_yellow/"},
		{"EF_UI_ENCHANT_SUCCESS", effectEnchantSuccess, "ui_enchant/ui_enchant_success/ui_enchant_success", "ui_enchant/ui_enchant_success/"},
		{"EF_UI_ENCHANT_FAIL", effectEnchantFail, "ui_enchant/ui_fail/ui_enchant_fail", "ui_enchant/ui_fail/"},
		{"EF_UI_ENCHANT_INTRO_BLUE", effectEnchantBlue, "ui_enchant/ui_intro_blue/ui_intro_blue", "ui_enchant/ui_intro_blue/"},
		{"EF_UI_ENCHANT_UP_SUCCESS", effectEnchantUpSuccess, "ui_enchant/ui_levelup_success/ui_levelup_success", "ui_enchant/ui_levelup_success/"},
		{"EF_UI_ENCHANT_UP_FAIL", effectEnchantUpFail, "ui_enchant/ui_fail/ui_levelup_fail", "ui_enchant/ui_fail/"},
		{"EF_UI_ENCHANT_INTRO_GREEN", effectEnchantGreen, "ui_enchant/ui_intro_green/ui_intro_green", "ui_enchant/ui_intro_green/"},
		{"EF_UI_ENCHANT_RESET_SUCCESS", effectEnchantResetOK, "ui_enchant/ui_reset_success/ui_reset_success", "ui_enchant/ui_reset_success/"},
		{"EF_UI_ENCHANT_RESET_FAIL", effectEnchantResetFail, "ui_enchant/ui_fail/ui_reset_fail", "ui_enchant/ui_fail/"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if len(spec.sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.texturePath != tc.texturePath || component.attachedEntity || component.renderBefore {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestEffectBodyColorTintFollowsRobrowserFlashWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		worldEffects: []worldEffect{{
			effectID: effectBodyColor,
			actorID:  42,
			starts:   starts,
			expires:  starts.Add(300 * time.Millisecond),
		}},
	}
	if got := mode.actorBodyColorTint(42, base, starts); got != base {
		t.Fatalf("initial tint = %+v, want base %+v", got, base)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(50*time.Millisecond)); got != (color.RGBA{R: 177, G: 75, B: 100, A: 255}) {
		t.Fatalf("half flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond)); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("full flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(300*time.Millisecond)); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("final flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(301*time.Millisecond)); got != base {
		t.Fatalf("expired tint = %+v, want base %+v", got, base)
	}
	if got := mode.actorBodyColorTint(99, base, starts.Add(100*time.Millisecond)); got != base {
		t.Fatalf("other actor tint = %+v, want base %+v", got, base)
	}
}

func TestPortal5BodyColorTintFollowsRobrowserFlashWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		worldEffects: []worldEffect{{
			effectID: effectPortal5,
			actorID:  42,
			starts:   starts,
			expires:  starts.Add(800 * time.Millisecond),
		}},
	}
	if got := mode.actorBodyColorTint(42, base, starts); got != (color.RGBA{R: 100, G: 150, B: 0, A: 25}) {
		t.Fatalf("initial tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(500*time.Millisecond)); got != (color.RGBA{R: 100, G: 150, B: 100, A: 140}) {
		t.Fatalf("mid tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(800*time.Millisecond)); got != base {
		t.Fatalf("final tint = %+v, want base %+v", got, base)
	}
}

func TestMagicCrasher2BodyColorTintUsesRobrowserRandomColorWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		worldEffects: []worldEffect{{
			effectID: effectMagicCrasher2,
			actorID:  42,
			starts:   starts,
			expires:  starts.Add(time.Second),
		}},
	}
	first := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond))
	second := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond))
	if first != second {
		t.Fatalf("random tint should be deterministic per frame: first %+v second %+v", first, second)
	}
	if first == base || first.A != base.A {
		t.Fatalf("active random tint = %+v, want color-channel change with unchanged alpha", first)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(time.Second)); got != base {
		t.Fatalf("expired random tint = %+v, want base %+v", got, base)
	}
}

func TestActorBodySizeMultiplierFollowsRobrowserEswooSizes(t *testing.T) {
	starts := time.Unix(10, 0)
	for _, tc := range []struct {
		name     string
		effectID int
		duration time.Duration
		at       time.Duration
		want     float64
	}{
		{"EF_BABYBODY start", effectBabyBody, 300 * time.Millisecond, 0, 1},
		{"EF_BABYBODY middle", effectBabyBody, 300 * time.Millisecond, 150 * time.Millisecond, 0.75},
		{"EF_BABYBODY end", effectBabyBody, 300 * time.Millisecond, 300 * time.Millisecond, 0.5},
		{"EF_BABYBODY2", effectBabyBody2, 5 * time.Minute, time.Second, 0.5},
		{"EF_GIANTBODY middle", effectGiantBody, 300 * time.Millisecond, 150 * time.Millisecond, 1.25},
		{"EF_GIANTBODY end", effectGiantBody, 300 * time.Millisecond, 300 * time.Millisecond, 1.5},
		{"EF_GIANTBODY2", effectGiantBody2, 5 * time.Minute, time.Second, 1.5},
	} {
		mode := WorldMode{
			worldEffects: []worldEffect{{
				effectID: tc.effectID,
				actorID:  42,
				starts:   starts,
				expires:  starts.Add(tc.duration),
				duration: tc.duration,
			}},
		}
		if got := mode.actorBodySizeMultiplier(42, starts.Add(tc.at)); got != tc.want {
			t.Fatalf("%s multiplier = %f, want %f", tc.name, got, tc.want)
		}
	}
}

func TestRobrowserRepairWeaponAndShockwaveSpecs(t *testing.T) {
	repair, ok := worldEffectSpecForID(effectRepairWeapon)
	if !ok || len(repair.components) != 1 || repair.duration != 1820*time.Millisecond {
		t.Fatalf("EF_REPAIRWEAPON spec = %+v ok=%t", repair, ok)
	}
	if len(repair.sfx) != 2 || repair.sfx[0] != "effect\\black_weapon_repair_a.wav" || repair.sfx[1] != "effect\\black_weapon_repair_a.wav" {
		t.Fatalf("EF_REPAIRWEAPON sfx = %v", repair.sfx)
	}
	if len(repair.sfxDelays) != 2 || repair.sfxDelays[0] != 480*time.Millisecond || repair.sfxDelays[1] != 1320*time.Millisecond {
		t.Fatalf("EF_REPAIRWEAPON sfx delays = %v", repair.sfxDelays)
	}
	if component := repair.components[0]; component.kind != effectComponentSTR || component.strFile != "repairweapon" || !component.attachedEntity {
		t.Fatalf("EF_REPAIRWEAPON component = %+v", component)
	}

	shockwave, ok := worldEffectSpecForID(effectShockwave)
	if !ok || len(shockwave.components) != 1 {
		t.Fatalf("EF_SHOCKWAVE spec = %+v ok=%t", shockwave, ok)
	}
	if len(shockwave.sfx) != 1 || shockwave.sfx[0] != "effect\\hunter_shockwavetrap.wav" {
		t.Fatalf("EF_SHOCKWAVE sfx = %v", shockwave.sfx)
	}
	if component := shockwave.components[0]; component.kind != effectComponentSPR || component.spriteFile != "shockwave" || !component.attachedEntity {
		t.Fatalf("EF_SHOCKWAVE component = %+v", component)
	}
}

func TestRobrowserWaterBallAndSonicBlowSpecs(t *testing.T) {
	water, ok := worldEffectSpecForID(effectWaterBall)
	if !ok || len(water.components) != 1 || water.duration != 500*time.Millisecond {
		t.Fatalf("EF_WATERBALL spec = %+v ok=%t", water, ok)
	}
	component := water.components[0]
	if component.kind != effectComponent3D || len(component.textureFiles) != 3 || component.textureFiles[0] != "effect/water_out_a.bmp" || component.textureFiles[2] != "effect/water_out_c.bmp" {
		t.Fatalf("EF_WATERBALL texture files = %+v", component)
	}
	if component.frameDelay != 10*time.Millisecond || component.duration != 500*time.Millisecond || !component.fadeOut || component.posXRand != 1.5 || component.posZRand != 1.5 || component.posYEnd != 3 || !component.posYSmooth || component.sizeStart != effectTableSize(30.5) || !component.rotateWithCamera || !component.blendAdditive || !component.attachedEntity {
		t.Fatalf("EF_WATERBALL component = %+v", component)
	}

	water2, ok := worldEffectSpecForID(effectWaterBall2)
	if !ok || len(water2.components) != 1 || water2.duration != 1450*time.Millisecond {
		t.Fatalf("EF_WATERBALL2 spec = %+v ok=%t", water2, ok)
	}
	projectile := water2.components[0]
	if projectile.kind != effectComponent3D || projectile.spriteFile != "data\\sprite\\이팩트\\waterball" || projectile.duration != 500*time.Millisecond || projectile.duplicate != 20 || projectile.duplicateDelay != 50*time.Millisecond {
		t.Fatalf("EF_WATERBALL2 projectile resource/timing = %+v", projectile)
	}
	if !projectile.fromSrc || !projectile.rotateToTarget || !projectile.fadeOut || projectile.sizeStart != effectTableSize(50) || projectile.posZ != 5 || projectile.posZEnd != 0.0001 || projectile.arc != 7.5 || projectile.retreat != 5 {
		t.Fatalf("EF_WATERBALL2 projectile motion = %+v", projectile)
	}

	sonic, ok := worldEffectSpecForID(effectSonicBlow)
	if !ok || len(sonic.components) != 1 || sonic.duration != 400*time.Millisecond {
		t.Fatalf("EF_SONICBLOW spec = %+v ok=%t", sonic, ok)
	}
	ring := sonic.components[0]
	if ring.kind != effectComponent3D || ring.textureFile != "effect/ring2.bmp" || ring.duration != 400*time.Millisecond || ring.alphaMax != 1 || !ring.fadeOut || ring.sizeStart != effectTableSize(100) || ring.sizeEnd != effectTableSize(300) || !ring.blendAdditive || !ring.attachedEntity {
		t.Fatalf("EF_SONICBLOW ring = %+v", ring)
	}
	spin, ok := worldEffectSpecForID(effectSonicBlowHit)
	if !ok || len(spin.components) != 1 || spin.components[0].kind != effectComponentFUNC || spin.components[0].funcName != "SonicBlowHitSpin" || !spin.components[0].attachedEntity {
		t.Fatalf("EF_SONICBLOWHIT spec = %+v ok=%t", spin, ok)
	}
}

func TestRobrowserCrashEarthFirePillarAndQuadHornOneHundredSpecs(t *testing.T) {
	crash, ok := worldEffectSpecForID(effectCrashEarth)
	if !ok || len(crash.components) != 1 {
		t.Fatalf("EF_CRASHEARTH spec = %+v ok=%t", crash, ok)
	}
	if crash.cameraShakeDelay != 350*time.Millisecond || crash.cameraShake != 650*time.Millisecond {
		t.Fatalf("EF_CRASHEARTH camera shake = delay %s duration %s", crash.cameraShakeDelay, crash.cameraShake)
	}
	if component := crash.components[0]; component.kind != effectComponentSTR || component.strFile != "crashearth" || component.attachedEntity {
		t.Fatalf("EF_CRASHEARTH component = %+v", component)
	}

	fire, ok := worldEffectSpecForID(effectFirePillarOn)
	if !ok || len(fire.components) != 3 || fire.duration != 6*time.Second {
		t.Fatalf("EF_FIREPILLARON spec = %+v ok=%t", fire, ok)
	}
	for i, component := range fire.components {
		if component.kind != effectComponentCylinder || component.textureName != "magic_red" || component.duration != 5*time.Second || component.delay != time.Second || !component.rotate || component.attachedEntity {
			t.Fatalf("EF_FIREPILLARON component %d = %+v", i, component)
		}
	}
	if fire.components[0].bottomSize != 1 || fire.components[0].topSize != 2 || fire.components[0].height != 3 || fire.components[2].bottomSize != 0.5 || fire.components[2].topSize != 1 || fire.components[2].height != 7 {
		t.Fatalf("EF_FIREPILLARON cylinder sizes = %+v", fire.components)
	}

	grim, ok := worldEffectSpecForID(effectGrimtoothAtk)
	if !ok || len(grim.components) != 3 || grim.duration != 15*time.Second {
		t.Fatalf("EF_GRIMTOOTHATK spec = %+v ok=%t", grim, ok)
	}
	first := grim.components[0]
	if first.kind != effectComponentQuadHorn || first.textureFile != "effect/stone.bmp" || first.duration != 15*time.Second || first.quadHornHeightMin != 2.5 || first.quadHornBottomMin != 0.15 || first.quadHornRotateXMin != -15 || first.quadHornOffsetYMin != 0.4 || first.quadHornOffsetZ != -0.2 || first.animation != 3 || first.quadHornAnimSpeed != 120*time.Millisecond || !first.quadHornAnimOut {
		t.Fatalf("EF_GRIMTOOTHATK first = %+v", first)
	}
	if grim.components[1].quadHornRotateYMin != 45 || grim.components[1].quadHornRotateZMin != -15 || grim.components[2].quadHornRotateZMin != 15 {
		t.Fatalf("EF_GRIMTOOTHATK rotations = %+v %+v", grim.components[1], grim.components[2])
	}

	heaven, ok := worldEffectSpecForID(effectHeavenDrive)
	if !ok || len(heaven.components) != 25 || heaven.duration != time.Second || heaven.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_HEAVENDRIVE spec = %+v ok=%t", heaven, ok)
	}
	if len(heaven.sfx) != 1 || heaven.sfx[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("EF_HEAVENDRIVE sfx = %v", heaven.sfx)
	}
	center := heaven.components[12]
	if center.kind != effectComponentQuadHorn || center.textureFile != "effect/stone.bmp" || center.duration != time.Second || center.posX != 0 || center.posY != 0 || center.quadHornHeightMin != 0.75 || center.quadHornHeightMax != 1.2 || center.quadHornBottomMin != 0.4 || center.quadHornBottomMax != 0.7 || center.quadHornAnimSpeed != 250*time.Millisecond || !center.quadHornAnimOut {
		t.Fatalf("EF_HEAVENDRIVE center = %+v", center)
	}
	if heaven.components[0].posX != -2 || heaven.components[0].posY != -2 || heaven.components[24].posX != 2 || heaven.components[24].posY != 2 {
		t.Fatalf("EF_HEAVENDRIVE grid edges = %+v %+v", heaven.components[0], heaven.components[24])
	}
}

func TestRobrowserQuadHornRuntimeDefaults(t *testing.T) {
	if got := quadHornDefaultOffset(0); got != 0.5 {
		t.Fatalf("quadHornDefaultOffset(0) = %v, want robr default 0.5", got)
	}
	if got := quadHornDefaultOffset(-0.2); got != -0.2 {
		t.Fatalf("quadHornDefaultOffset(-0.2) = %v, want -0.2", got)
	}
	effect := worldEffect{effectID: effectEarthSpike, actorID: 1, starts: time.Unix(10, 20)}
	if got := quadHornRange(effect, 1, 0, -0.2); got != -0.2 {
		t.Fatalf("quadHornRange max<min = %v, want -0.2", got)
	}
}

func TestRobrowserOldPortalEffectsZeroToFifty(t *testing.T) {
	entry, ok := worldEffectSpecForID(effectEntry)
	if !ok {
		t.Fatal("EF_ENTRY spec missing")
	}
	if entry.duration != 500*time.Millisecond || len(entry.sfx) != 1 || entry.sfx[0] != "effect\\ef_portal.wav" {
		t.Fatalf("EF_ENTRY timing/sfx = duration %s sfx %#v", entry.duration, entry.sfx)
	}
	if len(entry.components) != 3 {
		t.Fatalf("EF_ENTRY components = %d, want 3", len(entry.components))
	}
	for i, component := range entry.components {
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 500*time.Millisecond || component.animation != 1 || !component.fade || !component.rotate {
			t.Fatalf("EF_ENTRY component %d = %+v", i, component)
		}
	}
	if entry.components[0].attachedEntity || !entry.components[1].attachedEntity || !entry.components[2].attachedEntity {
		t.Fatalf("EF_ENTRY attachment flags = %t %t %t", entry.components[0].attachedEntity, entry.components[1].attachedEntity, entry.components[2].attachedEntity)
	}
	if entry.components[0].height != 7.5 || entry.components[1].height != 8 || entry.components[2].topSize != 1.5 {
		t.Fatalf("EF_ENTRY dimensions = %+v", entry.components)
	}

	warp, ok := worldEffectSpecForID(effectWarp)
	if !ok || len(warp.components) != 1 {
		t.Fatalf("EF_WARP spec = %+v ok=%t", warp, ok)
	}
	wave := warp.components[0]
	if wave.kind != effectComponentCylinder || wave.textureName != "ring_yellow" || wave.animation != 4 || wave.duplicate != 4 || wave.duplicateDelay != 300*time.Millisecond {
		t.Fatalf("EF_WARP wave = %+v", wave)
	}
	if wave.bottomSize != 10 || wave.topSize != 13 || wave.posZ != 0.1 || !wave.attachedEntity {
		t.Fatalf("EF_WARP dimensions = %+v", wave)
	}
	if got := worldEffectComponentDuration(warp, wave); got != 1900*time.Millisecond {
		t.Fatalf("EF_WARP resolved duration = %s, want 1900ms", got)
	}

	teleport, ok := worldEffectSpecForID(effectTeleportOld)
	if !ok || len(teleport.components) != 1 {
		t.Fatalf("EF_TELEPORTATION spec = %+v ok=%t", teleport, ok)
	}
	beam := teleport.components[0]
	if teleport.duration != time.Second || len(teleport.sfx) != 1 || teleport.sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("EF_TELEPORTATION timing/sfx = %s %#v", teleport.duration, teleport.sfx)
	}
	if beam.kind != effectComponentCylinder || beam.textureName != "ring_blue" || beam.animation != 5 || beam.height != 35 || beam.bottomSize != 0.8 || beam.topSize != 0.7 || !beam.rotate {
		t.Fatalf("EF_TELEPORTATION beam = %+v", beam)
	}

	ready, ok := worldEffectSpecForID(effectReadyPortalOld)
	if !ok || len(ready.components) != 1 {
		t.Fatalf("EF_READYPORTAL spec = %+v ok=%t", ready, ok)
	}
	portal := ready.components[0]
	if ready.duration != 25*time.Second || len(ready.sfx) != 1 || ready.sfx[0] != "effect\\ef_readyportal.wav" {
		t.Fatalf("EF_READYPORTAL timing/sfx = %s %#v", ready.duration, ready.sfx)
	}
	if portal.kind != effectComponentCylinder || portal.textureName != "alpha_down" || portal.color != (color.RGBA{R: 178, G: 178, B: 255, A: 255}) || portal.height != 15 || portal.alphaMax != 0.6 {
		t.Fatalf("EF_READYPORTAL cylinder = %+v", portal)
	}
}

func TestRobrowserOldRestoreEffectsZeroToFifty(t *testing.T) {
	exit, ok := worldEffectSpecForID(effectExit)
	if !ok {
		t.Fatal("EF_EXIT spec missing")
	}
	if exit.duration != 2*time.Second || len(exit.sfx) != 1 || exit.sfx[0] != "_heal_effect.wav" || len(exit.components) != 3 {
		t.Fatalf("EF_EXIT spec = %+v", exit)
	}
	if cylinder := exit.components[0]; cylinder.kind != effectComponentCylinder || cylinder.textureName != "alpha_down" || cylinder.duration != 2*time.Second || cylinder.animation != 1 || cylinder.alphaMax != 0.2 || !cylinder.blendAdditive {
		t.Fatalf("EF_EXIT cylinder = %+v", cylinder)
	}
	if particle := exit.components[1]; particle.kind != effectComponent3D || particle.textureFile != "effect/pok3.tga" || particle.delay != 400*time.Millisecond || particle.duplicate != 6 || particle.duplicateDelay != 80*time.Millisecond || !particle.sparkling {
		t.Fatalf("EF_EXIT first particle = %+v", particle)
	}
	if particle := exit.components[2]; particle.duration != 900*time.Millisecond || particle.delay != 200*time.Millisecond || particle.duplicate != 3 || particle.duplicateDelay != 200*time.Millisecond || particle.posZEnd != 6 {
		t.Fatalf("EF_EXIT second particle = %+v", particle)
	}

	enhance, ok := worldEffectSpecForID(effectEnhance)
	if !ok || len(enhance.components) != 3 {
		t.Fatalf("EF_ENHANCE spec = %+v ok=%t", enhance, ok)
	}
	if enhance.components[0].textureName != "alpha_down" || enhance.components[0].blendAdditive != true || enhance.components[0].duration != 2*time.Second {
		t.Fatalf("EF_ENHANCE cylinder = %+v", enhance.components[0])
	}
	for _, tc := range []struct {
		index     int
		delay     time.Duration
		duplicate int
	}{
		{index: 1, delay: 500 * time.Millisecond, duplicate: 7},
		{index: 2, delay: 400 * time.Millisecond, duplicate: 3},
	} {
		component := enhance.components[tc.index]
		if component.kind != effectComponent3D || component.textureFile != "effect/ac_center2.tga" || component.delay != tc.delay || component.duplicate != tc.duplicate || component.duplicateDelay != 200*time.Millisecond {
			t.Fatalf("EF_ENHANCE particle %d = %+v", tc.index, component)
		}
		if component.sizeStartX != 2.5*effectPixelRatio || component.sizeRandY != 15*effectPixelRatio || component.sizeRandYMiddle != 45*effectPixelRatio {
			t.Fatalf("EF_ENHANCE particle %d size = %+v", tc.index, component)
		}
	}

	healSP, ok := worldEffectSpecForID(effectHealSP)
	if !ok || len(healSP.components) != 3 {
		t.Fatalf("EF_HEALSP spec = %+v ok=%t", healSP, ok)
	}
	if len(healSP.sfx) != 1 || healSP.sfx[0] != "_heal_effect.wav" {
		t.Fatalf("EF_HEALSP sfx = %#v", healSP.sfx)
	}
	blue := color.RGBA{R: 25, G: 128, B: 255, A: 255}
	if cylinder := healSP.components[0]; cylinder.textureName != "ring_blue" || cylinder.color != blue || !cylinder.rotate || !cylinder.blendAdditive {
		t.Fatalf("EF_HEALSP cylinder = %+v", cylinder)
	}
	if healSP.components[1].color != blue || healSP.components[2].color != blue {
		t.Fatalf("EF_HEALSP particle tints = %+v %+v", healSP.components[1].color, healSP.components[2].color)
	}
}

func TestRobrowserOldBoltSoundAndStatusEffectsZeroToFifty(t *testing.T) {
	glass, ok := worldEffectSpecForID(effectGlassWall)
	if !ok || len(glass.components) != 1 {
		t.Fatalf("EF_GLASSWALL spec = %+v ok=%t", glass, ok)
	}
	if component := glass.components[0]; component.kind != effectComponentSTR || component.strFile != "effect/safetywall" || component.attachedEntity {
		t.Fatalf("EF_GLASSWALL component = %+v", component)
	}
	if len(glass.sfx) != 1 || glass.sfx[0] != "effect\\ef_glasswall.wav" {
		t.Fatalf("EF_GLASSWALL sfx = %#v", glass.sfx)
	}

	ice, ok := worldEffectSpecForID(effectIceArrow)
	if !ok {
		t.Fatal("EF_ICEARROW spec missing")
	}
	if len(ice.components) != 0 || len(ice.sfx) != 1 || ice.sfx[0] != "effect\\ef_icearrow%d.wav" || ice.sfxRandMin != 1 || ice.sfxRandMax != 3 {
		t.Fatalf("EF_ICEARROW spec = %+v", ice)
	}

	fire, ok := worldEffectSpecForID(effectFireArrow)
	if !ok {
		t.Fatal("EF_FIREARROW spec missing")
	}
	if len(fire.components) != 0 || len(fire.sfx) != 1 || fire.sfx[0] != "effect\\ef_firearrow1.wav" {
		t.Fatalf("EF_FIREARROW spec = %+v", fire)
	}

	incAgiDex, ok := worldEffectSpecForID(effectIncAgiDex)
	if !ok || len(incAgiDex.components) != 1 {
		t.Fatalf("EF_INCAGIDEX spec = %+v ok=%t", incAgiDex, ok)
	}
	if len(incAgiDex.sfx) != 1 || incAgiDex.sfx[0] != "effect\\ef_incagidex.wav" {
		t.Fatalf("EF_INCAGIDEX sfx = %#v", incAgiDex.sfx)
	}
	overlay := incAgiDex.components[0]
	if overlay.kind != effectComponent3D || overlay.textureFile != "effect/dex_agi_up.bmp" || overlay.duration != time.Second || !overlay.fadeIn || !overlay.fadeOut || !overlay.attachedEntity || !overlay.overlay {
		t.Fatalf("EF_INCAGIDEX overlay = %+v", overlay)
	}
	if overlay.posZ != 0.4 || overlay.posZEnd != 3 || overlay.sizeStart != 100*effectPixelRatio || overlay.sizeStartY != 45*effectPixelRatio || !overlay.sizeSmooth {
		t.Fatalf("EF_INCAGIDEX overlay geometry = %+v", overlay)
	}
}

func TestPneumaEffectSpecMatchesRoBrowserSTR(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectPneuma)
	if !ok {
		t.Fatal("Pneuma effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("Pneuma components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "pneuma%d" || component.strRandMin != 1 || component.strRandMax != 3 || component.attachedEntity {
		t.Fatalf("Pneuma component = %+v", component)
	}
}

func TestTorchEffectSpecMatchesRoBrowserShape(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectTorch)
	if !ok {
		t.Fatal("torch effect spec missing")
	}
	if spec.duration != 24*time.Hour {
		t.Fatalf("torch duration = %s", spec.duration)
	}
	if len(spec.components) != 1 {
		t.Fatalf("torch components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "torch_01" || !component.spriteRepeat {
		t.Fatalf("torch component = %+v", component)
	}
	if component.duration != 600*time.Millisecond || component.spriteDelay != 100*time.Millisecond {
		t.Fatalf("torch timing = duration %s delay %s", component.duration, component.spriteDelay)
	}
	if component.posX != 0.1 || component.posZ != 0.8 || component.sizeStart != effectTableSize(100) || component.angleStart != 270 || !component.rotateToTarget {
		t.Fatalf("torch placement = %+v", component)
	}
	if got := worldEffectSpriteAngle(component); got != 360 {
		t.Fatalf("torch effective angle = %.1f, want 360.0", got)
	}
}

func TestFireflyEffectSpecUsesFaintSpriteParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectFirefly)
	if !ok {
		t.Fatal("firefly effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("firefly components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "" || component.spriteFile == "" || !component.spriteRepeat {
		t.Fatalf("firefly component = %+v", component)
	}
	if component.alphaMax > 0.25 || component.sizeEnd > effectTableSize(120) {
		t.Fatalf("firefly should stay faint and moderately sized: %+v", component)
	}
}

func TestBubbleEffectSpecMatchesRoBrowserShape(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBubble)
	if !ok {
		t.Fatal("bubble effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("bubble components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "bubble%d" || component.strRandMin != 1 || component.strRandMax != 4 {
		t.Fatalf("bubble component = %+v", component)
	}
}

func TestWorldEffectSpecLookupReturnsCopy(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing")
	}
	spec.sfx[0] = "mutated.wav"
	spec.components[0].textureName = "mutated"

	again, ok := worldEffectSpecForID(effectBashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing after mutation")
	}
	if again.sfx[0] != "effect\\ef_bash.wav" || again.components[0].textureName != "alpha_down" {
		t.Fatalf("catalog mutated: %+v", again)
	}
}

func TestResolveEffectSTRFileUsesDeterministicRandRange(t *testing.T) {
	component := worldEffectComponent{
		strFile:    "firehit%d",
		strRandMin: 1,
		strRandMax: 3,
	}
	effect := worldEffect{effectID: effectFireHit, actorID: 100, starts: time.Unix(10, 20)}
	got := resolveEffectSTRFile(component, effect, false)
	if got != "firehit1" && got != "firehit2" && got != "firehit3" {
		t.Fatalf("resolved STR file = %q, want firehit1..3", got)
	}
	if again := resolveEffectSTRFile(component, effect, false); again != got {
		t.Fatalf("resolved STR file changed from %q to %q", got, again)
	}
}

func TestResolveEffectSTRFileUsesMinFileForLessEffects(t *testing.T) {
	component := worldEffectComponent{
		strFile:    "angelus",
		strMinFile: "jong_mini",
	}
	if got := resolveEffectSTRFile(component, worldEffect{}, true); got != "jong_mini" {
		t.Fatalf("resolved STR file = %q, want jong_mini", got)
	}
}

func expectEffectIDs(t *testing.T, label string, got []int, want ...int) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s effects = %v, want %v", label, got, want)
	}
}

func TestSwordmanSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "SM_BASH begin", skillBeginEffectIDs(5), effectBashBegin)
	expectEffectIDs(t, "SM_BASH hit", skillHitEffectIDs(5), effectBashHit)
	expectEffectIDs(t, "SM_PROVOKE success", skillSuccessEffectIDs(6), effectProvoke)
	expectEffectIDs(t, "SM_MAGNUM target", skillEffectIDs(7), effectQuakeMagnum)
	expectEffectIDs(t, "SM_MAGNUM caster", skillEffectOnCasterIDs(7), effectMagnumBreak)
	expectEffectIDs(t, "SM_ENDURE", skillEffectIDs(8), effectEndure)
}

func TestNoviceSkillEffectMappings(t *testing.T) {
	if action := skillAction(db.SkillNVBasic); !action.defined || action.action != skillActorActionNone {
		t.Fatalf("basic skill action = %+v, want no source action", action)
	}
	expectEffectIDs(t, "NV_FIRSTAID", skillEffectIDs(142), effectFirstAid)
	expectEffectIDs(t, "NV_TRICKDEAD", skillEffectIDs(143))
}

func TestMageSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "MG_SIGHT success", skillSuccessEffectIDs(10))
	expectEffectIDs(t, "MG_SIGHT immediate", skillEffectIDs(10))
	expectEffectIDs(t, "MG_NAPALMBEAT hit", skillHitEffectIDs(11), effectBashHit)
	expectEffectIDs(t, "MG_SAFETYWALL ground", skillGroundEffectIDs(12))
	expectEffectIDs(t, "MG_SOULSTRIKE before-hit", skillBeforeHitEffectIDs(13), effectSoulStrike)
	expectEffectIDs(t, "MG_SOULSTRIKE hit", skillHitEffectIDs(13), effectBashHit)
	expectEffectIDs(t, "MG_COLDBOLT before-hit", skillBeforeHitEffectIDs(14), effectColdBolt)
	expectEffectIDs(t, "MG_COLDBOLT hit", skillHitEffectIDs(14), effectColdHit)
	expectEffectIDs(t, "MG_FROSTDIVER", skillEffectIDs(15), effectFrostDiver)
	expectEffectIDs(t, "MG_FROSTDIVER before-hit", skillBeforeHitEffectIDs(15))
	expectEffectIDs(t, "MG_FROSTDIVER hit", skillHitEffectIDs(15), effectFrostDiverHit)
	expectEffectIDs(t, "MG_STONECURSE", skillEffectIDs(16), effectStoneCurse)
	expectEffectIDs(t, "MG_FIREBOLT before-hit", skillBeforeHitEffectIDs(19), effectFireBolt)
	expectEffectIDs(t, "MG_FIREBALL before-hit", skillBeforeHitEffectIDs(17), effectFireBall)
	expectEffectIDs(t, "MG_FIREWALL ground", skillGroundEffectIDs(18), effectFireWall)
	for _, skillID := range []uint16{17, 18, 19} {
		expectEffectIDs(t, "fire skill hit", skillHitEffectIDs(skillID), effectFireHit)
	}
	for _, skillID := range []uint16{20, 21} {
		expectEffectIDs(t, "wind skill hit", skillHitEffectIDs(skillID), effectWindHit)
	}
	expectEffectIDs(t, "MG_LIGHTNINGBOLT before-hit", skillBeforeHitEffectIDs(20))
	expectEffectIDs(t, "MG_LIGHTNINGBOLT", skillEffectIDs(20), effectLightningBolt)
	expectEffectIDs(t, "MG_THUNDERSTORM", skillEffectIDs(21), effectThunderStorm)
	expectEffectIDs(t, "MG_THUNDERSTORM ground", skillGroundEffectIDs(21))
	expectEffectIDs(t, "MG_ENERGYCOAT", skillEffectIDs(157), effectEnergyCoat)
	expectEffectIDs(t, "MG_THUNDERSTORM before-hit", skillBeforeHitEffectIDs(21))
	for _, skillID := range []uint16{20, 21} {
		expectEffectIDs(t, "wind skill begin", skillBeginEffectIDs(skillID))
	}
}

func TestFireBoltEffectSpecUsesFallingFrameList(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectFireBolt)
	if !ok {
		t.Fatal("fire bolt effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || len(component.textureFiles) != 6 || component.duration != 500*time.Millisecond {
		t.Fatalf("component = %+v", component)
	}
	if component.posZ != 20 || component.posZEnd != 0.0001 || component.posXStartMiddle != 5 || component.posYStartMiddle != 2 || component.angleStart != 112.5 || !component.blendAdditive {
		t.Fatalf("fire bolt trajectory = %+v", component)
	}
	if component.sizeStartX != 100*effectPixelRatio || component.sizeStartY != 50*effectPixelRatio {
		t.Fatalf("fire bolt size = %.3f x %.3f", component.sizeStartX, component.sizeStartY)
	}
}

func TestFireBallEffectSpecMatchesRobrowserProjectileAndHit(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectFireBall)
	if !ok {
		t.Fatal("fire ball effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_fireball.wav" {
		t.Fatalf("fire ball sfx = %#v", spec.sfx)
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "fireball" || !component.spriteRepeat {
		t.Fatalf("projectile sprite = %+v", component)
	}
	if component.duration != 250*time.Millisecond || component.delay != 160*time.Millisecond || component.delayOffsetDelta != -40*time.Millisecond {
		t.Fatalf("projectile timing = duration %s delay %s delta %s", component.duration, component.delay, component.delayOffsetDelta)
	}
	if component.duplicate != 5 || component.duplicateDelay != 0 {
		t.Fatalf("projectile duplicates = %d delay %s", component.duplicate, component.duplicateDelay)
	}
	if worldEffectComponentStartOffset(component, 0) != 160*time.Millisecond || worldEffectComponentStartOffset(component, 4) != 0 {
		t.Fatalf("projectile duplicate offsets = first %s last %s", worldEffectComponentStartOffset(component, 0), worldEffectComponentStartOffset(component, 4))
	}
	if worldEffectComponentDuration(spec, component) != 410*time.Millisecond {
		t.Fatalf("projectile resolved duration = %s, want 410ms", worldEffectComponentDuration(spec, component))
	}
	if !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || component.alphaMax != 0.2 || component.alphaMaxDelta != 0.2 {
		t.Fatalf("projectile orientation/alpha = %+v", component)
	}
	if component.posZ != 2 || component.sizeStart != 200*effectPixelRatio || component.sizeEnd != 200*effectPixelRatio {
		t.Fatalf("projectile position/size = %+v", component)
	}

	hitSpec, ok := worldEffectSpecForID(effectFireHit)
	if !ok || len(hitSpec.components) != 1 {
		t.Fatalf("fire hit effect missing or wrong component count: ok=%t components=%d", ok, len(hitSpec.components))
	}
	hit := hitSpec.components[0]
	if hit.kind != effectComponentSTR || hit.strFile != "firehit%d" || hit.strRandMin != 1 || hit.strRandMax != 3 || !hit.attachedEntity {
		t.Fatalf("fire hit STR = %+v", hit)
	}
	if len(hitSpec.sfx) != 1 || hitSpec.sfx[0] != "effect\\ef_firehit.wav" {
		t.Fatalf("fire hit sfx = %#v", hitSpec.sfx)
	}
}

func TestBashHitEffectSpecMatchesRobrowserLensCircle(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBashHit)
	if !ok {
		t.Fatal("bash hit effect missing")
	}
	if spec.duration != 350*time.Millisecond {
		t.Fatalf("duration = %s, want 350ms", spec.duration)
	}
	if len(spec.components) != 8 {
		t.Fatalf("components = %d, want 8 reference client lens slashes", len(spec.components))
	}
	for i, component := range spec.components {
		if component.kind != effectComponent2D {
			t.Fatalf("component %d kind = %d, want 2D", i, component.kind)
		}
		wantTexture := "effect/lens1.tga"
		if i%2 == 1 {
			wantTexture = "effect/lens2.tga"
		}
		if component.textureFile != wantTexture || component.duration != 250*time.Millisecond || !component.fadeOut || !component.overlay {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.durationRandMin != 200*time.Millisecond || component.durationRandMax != 350*time.Millisecond {
			t.Fatalf("component %d duration rand = %s..%s", i, component.durationRandMin, component.durationRandMax)
		}
		if component.sizeStartXRandMin != 25*effectPixelRatio || component.sizeStartXRandMax != 40*effectPixelRatio {
			t.Fatalf("component %d start x range = %.3f..%.3f", i, component.sizeStartXRandMin, component.sizeStartXRandMax)
		}
		if component.sizeStartY != 10*effectPixelRatio || component.sizeEndX != 1*effectPixelRatio {
			t.Fatalf("component %d fixed axis sizes = %.3f %.3f", i, component.sizeStartY, component.sizeEndX)
		}
		if component.sizeEndYRandMin != 250*effectPixelRatio || component.sizeEndYRandMax != 300*effectPixelRatio {
			t.Fatalf("component %d end y range = %.3f..%.3f", i, component.sizeEndYRandMin, component.sizeEndYRandMax)
		}
		if !component.circlePattern || component.circleInnerSize != 2.2 || component.circleOuterRandMin != 5 || component.circleOuterRandMax != 6 {
			t.Fatalf("component %d circle pattern = %+v", i, component)
		}
		if component.angleRandMax <= component.angleRandMin {
			t.Fatalf("component %d angle range = %.1f..%.1f", i, component.angleRandMin, component.angleRandMax)
		}
	}
	mode := &WorldMode{}
	effect := worldEffect{effectID: effectBashHit, actorID: 300, starts: time.Unix(10, 20)}
	startX, startY, _ := mode.effect3DOffset(client.Context{}, spec.components[0], effect, 0, 0, 0, 0, 0, 0)
	endX, endY, _ := mode.effect3DOffset(client.Context{}, spec.components[0], effect, 0, 0, 1, 0, 0, 0)
	if math.Hypot(endX, endY) <= math.Hypot(startX, startY) {
		t.Fatalf("circle pattern does not move outward: start=(%.2f,%.2f) end=(%.2f,%.2f)", startX, startY, endX, endY)
	}
}

func TestRegularHitEffectSpecMatchesRobrowserParticleBurst(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectHit1)
	if !ok || len(spec.components) != 1 {
		t.Fatalf("regular hit spec = %+v ok=%t, want one component", spec, ok)
	}
	if spec.duration != 300*time.Millisecond {
		t.Fatalf("duration = %s, want 300ms", spec.duration)
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "effect/pok3.tga" || component.duration != 300*time.Millisecond {
		t.Fatalf("component resource/timing = %+v", component)
	}
	if component.duplicate != 4 || component.alphaMax != 0.8 || !component.fadeIn || !component.fadeOut || !component.sparkling {
		t.Fatalf("component duplicate/fade = %+v", component)
	}
	if component.posZ != 1 || component.posXEndRand != 2 || component.posYEndRand != 2 || component.posZEndRand != 2 {
		t.Fatalf("component position = %+v", component)
	}
	if component.sizeStart != effectTableSize(10) || component.sizeEnd != effectTableSize(10) || component.sizeRand != effectTableSize(20) || !component.sizeSmooth {
		t.Fatalf("component size = %+v", component)
	}
}

func TestSkillHitEffectSpecsMatchRobrowserCylindersAndSlashes(t *testing.T) {
	hit3, ok := worldEffectSpecForID(effectHit3)
	if !ok || len(hit3.components) != 2 {
		t.Fatalf("hit3 spec = %+v ok=%t, want two cylinders", hit3, ok)
	}
	if len(hit3.sfx) != 1 || hit3.sfx[0] != "effect\\ef_hit3.wav" {
		t.Fatalf("hit3 sfx = %v", hit3.sfx)
	}
	if first, second := hit3.components[0], hit3.components[1]; first.kind != effectComponentCylinder || second.kind != effectComponentCylinder || first.textureName != "lens2" || second.textureName != "lens2" {
		t.Fatalf("hit3 cylinder resources = %+v %+v", first, second)
	}
	if hit3.components[0].bottomSize != 0.37 || hit3.components[0].topSize != 1 || hit3.components[1].bottomSize != 0.37 || hit3.components[1].topSize != 0.37 {
		t.Fatalf("hit3 cylinder sizes = %+v %+v", hit3.components[0], hit3.components[1])
	}
	for i, component := range hit3.components {
		if component.duration != 150*time.Millisecond || component.alphaMax != 0.8 || !component.fade || component.animation != 1 || component.posZ != 1 || component.height != 4 || component.angleX != -90 || !component.rotateWithCamera || !component.attachedEntity {
			t.Fatalf("hit3 component %d = %+v", i, component)
		}
	}

	hit4, ok := worldEffectSpecForID(effectHit4)
	if !ok || len(hit4.components) != 1 {
		t.Fatalf("hit4 spec = %+v ok=%t, want one cylinder", hit4, ok)
	}
	component := hit4.components[0]
	if component.kind != effectComponentCylinder || component.textureName != "lens2" || component.bottomSize != 0.15 || component.topSize != 1 || component.duration != 150*time.Millisecond || component.angleX != -90 || !component.attachedEntity {
		t.Fatalf("hit4 component = %+v", component)
	}
	if len(hit4.sfx) != 1 || hit4.sfx[0] != "effect\\ef_hit4.wav" {
		t.Fatalf("hit4 sfx = %v", hit4.sfx)
	}

	for _, tc := range []struct {
		name     string
		effectID int
		kind     effectComponentKind
		width    float64
		height   float64
		sfx      string
		overlay  bool
	}{
		{"hit5", effectHit5, effectComponent3D, effectTableSize(15), effectTableSize(200), "effect\\ef_hit5.wav", false},
		{"hit6", effectHit6, effectComponent2D, effectTableSize(10), effectTableSize(150), "effect\\ef_hit6.wav", true},
	} {
		spec, ok := worldEffectSpecForID(tc.effectID)
		if !ok || len(spec.components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want two slash components", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.sfx {
			t.Fatalf("%s sfx = %v", tc.name, spec.sfx)
		}
		for i, component := range spec.components {
			if component.kind != tc.kind || component.textureFile != "effect/lens2.tga" || component.duration != 400*time.Millisecond || component.alphaMax != 1 || !component.fadeOut || !component.rotate || component.overlay != tc.overlay {
				t.Fatalf("%s component %d = %+v", tc.name, i, component)
			}
			if component.posZ != 1 || component.sizeStartX != tc.width || component.sizeEndX != tc.width || component.sizeStartY != effectTableSize(10) || component.sizeEndY != tc.height {
				t.Fatalf("%s component %d size/position = %+v", tc.name, i, component)
			}
		}
		if spec.components[0].angleStart != 90 || spec.components[0].angleEnd != 0 || spec.components[1].angleStart != 180 || spec.components[1].angleEnd != 90 {
			t.Fatalf("%s slash angles = %+v %+v", tc.name, spec.components[0], spec.components[1])
		}
	}
}

func TestColdBoltEffectSpecMatchesRobrowserProjectileAndRing(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectColdBolt)
	if !ok {
		t.Fatal("cold bolt effect missing")
	}
	if len(spec.components) != 2 {
		t.Fatalf("components = %d, want 2", len(spec.components))
	}
	projectile := spec.components[0]
	if projectile.kind != effectComponent3D || projectile.textureFile != "effect/icearrow.tga" || projectile.duration != 500*time.Millisecond {
		t.Fatalf("projectile = %+v", projectile)
	}
	if projectile.posZ != 20 || projectile.posZEnd != 0.0001 || projectile.posXStartMiddle != 5 || projectile.posYStartMiddle != 2 || projectile.sizeStart != 50*effectPixelRatio {
		t.Fatalf("cold bolt projectile trajectory = %+v", projectile)
	}
	ring := spec.components[1]
	if ring.kind != effectComponentCylinder || ring.textureName != "ring_blue" || ring.delay != 500*time.Millisecond || ring.duration != 1000*time.Millisecond {
		t.Fatalf("ring = %+v", ring)
	}
	if ring.bottomSize != 3 || ring.topSize != 5 || ring.animation != 4 {
		t.Fatalf("cold bolt ring dimensions = %+v", ring)
	}
}

func TestSightEffectSpecOrbitsAroundActor(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectSight)
	if !ok {
		t.Fatal("sight effect missing")
	}
	if len(spec.components) != 2 {
		t.Fatalf("components = %d, want robr shadow + sight sprite", len(spec.components))
	}
	shadow := spec.components[0]
	if !shadow.shadowTexture || shadow.spriteFile != "data\\sprite\\shadow" || shadow.duplicate != 10 {
		t.Fatalf("sight shadow component = %+v", shadow)
	}
	if shadow.sizeStart != 30*effectPixelRatio || shadow.sizeDelta != 10*effectPixelRatio {
		t.Fatalf("sight shadow size = %.3f delta %.3f", shadow.sizeStart, shadow.sizeDelta)
	}
	component := spec.components[1]
	if component.spriteFile != "sight" || component.duplicate != 10 || component.orbitRadiusX != 3 || component.orbitRadiusY != 3 || component.orbitRotations != 10 {
		t.Fatalf("sight orbit component = %+v", component)
	}
	if component.sizeStart != 60*effectPixelRatio || component.sizeDelta != 20*effectPixelRatio || component.alphaMaxDelta != 3.0/255.0 {
		t.Fatalf("sight orbit size/alpha delta = %.3f delta %.3f alpha_delta %.3f", component.sizeStart, component.sizeDelta, component.alphaMaxDelta)
	}
	ctx := client.Context{}
	effect := worldEffect{effectID: effectSight, actorID: 2000000}
	mode := &WorldMode{}
	x0, y0, _ := mode.effect3DOffset(ctx, component, effect, 0, 0, 0, 0, 0, 0)
	x1, y1, _ := mode.effect3DOffset(ctx, component, effect, 0, 1, 0, 0, 0, 0)
	x2, y2, _ := mode.effect3DOffset(ctx, component, effect, 0, 0, 0.025, 0, 0, 0)
	if math.Hypot(x0-x1, y0-y1) < 0.1 {
		t.Fatalf("sight duplicates overlap: duplicate0=(%.3f,%.3f) duplicate1=(%.3f,%.3f)", x0, y0, x1, y1)
	}
	if math.Hypot(x0-x2, y0-y2) < 0.1 {
		t.Fatalf("sight orbit does not move over time: start=(%.3f,%.3f) later=(%.3f,%.3f)", x0, y0, x2, y2)
	}
}

func TestFireBallSpriteRotationUsesRobrowserWorldTrajectory(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectFireBall)
	if !ok || len(spec.components) == 0 {
		t.Fatal("fire ball effect missing")
	}
	component := spec.components[0]
	tests := []struct {
		name      string
		sourceX   int
		sourceY   int
		targetX   int
		targetY   int
		cameraYaw float64
		want      float64
	}{
		{name: "same row", sourceX: 10, sourceY: 20, targetX: 12, targetY: 20, want: -math.Pi / 2},
		{name: "same column", sourceX: 12, sourceY: 18, targetX: 12, targetY: 20, want: 0},
		{name: "diagonal", sourceX: 10, sourceY: 18, targetX: 12, targetY: 20, want: -math.Pi / 4},
		{name: "camera yaw", sourceX: 10, sourceY: 20, targetX: 12, targetY: 20, cameraYaw: 45, want: -3 * math.Pi / 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: tt.sourceX, Y: tt.sourceY}
			world.UpsertActor(worldstate.Actor{ID: 300, X: tt.targetX, Y: tt.targetY})
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   world,
			}
			effect := worldEffect{effectID: effectFireBall, actorID: 300, targetID: 2000000}
			startX, startY, _, endX, endY, _, ok := effectTrajectoryEndpoints(ctx, component, effect)
			if !ok {
				t.Fatal("trajectory endpoints missing")
			}
			if math.Hypot(endX-startX, endY-startY) <= 0.001 {
				t.Fatalf("trajectory did not span caster and target: %.2f,%.2f -> %.2f,%.2f", startX, startY, endX, endY)
			}
			projection := newSceneProjectionForTargetYaw(800, 600, float64(tt.targetX), float64(tt.targetY), 0, tt.cameraYaw)
			angle, ok := effectSpriteRobrowserRotation(ctx, projection, component, effect, 0)
			if !ok {
				t.Fatal("rotation missing")
			}
			wantFromFormula := -(90 - math.Atan2(endY-startY, endX-startX)*180/math.Pi + tt.cameraYaw) * math.Pi / 180
			if math.Abs(angle-wantFromFormula) > 0.001 {
				t.Fatalf("angle = %.3f, want robr formula %.3f", angle, wantFromFormula)
			}
			if math.Abs(angle-tt.want) > 0.001 {
				t.Fatalf("angle = %.3f, want %.3f", angle, tt.want)
			}
		})
	}
}

func TestGroundSampleEffectSpecUsesMagicTargetPlane(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectGroundSample)
	if !ok {
		t.Fatal("ground sample effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentFUNC || component.funcAdapter != effectFuncGroundSample || component.funcName != "MagicTarget" || component.textureFile != "effect/magic_target.tga" || component.sizeStart != 1 {
		t.Fatalf("component = %+v", component)
	}
	if component.duration != 0 {
		t.Fatalf("ground sample component duration = %s, want inherited cast duration", component.duration)
	}
}

func TestCastRingEffectSpecUsesMagicRingCylinder(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectCastRing)
	if !ok {
		t.Fatal("cast ring effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentFUNC || component.funcAdapter != effectFuncCastRing || component.funcName != "CastRing" || component.textureName != "ring_yellow" {
		t.Fatalf("component = %+v", component)
	}
	if component.bottomSize != 0.8 || component.topSize != 2.45 || component.height != 2.8 {
		t.Fatalf("magic ring dimensions = bottom %.2f top %.2f height %.2f", component.bottomSize, component.topSize, component.height)
	}
	if component.duration != 0 {
		t.Fatalf("cast ring component duration = %s, want inherited cast duration", component.duration)
	}
}

func TestLockOnTargetEffectSpecMatchesRobrowserCastTargetCircle(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectLockOnTarget)
	if !ok {
		t.Fatal("lock-on target effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentFUNC || component.funcAdapter != effectFuncLockOnTarget || component.funcName != "LockOnTarget" || component.textureFile != "effect/lockon128.tga" || !component.attachedEntity {
		t.Fatalf("component = %+v", component)
	}
	start := time.Unix(10, 0)
	if got := lockOnTargetSize(start, start); got != 15 {
		t.Fatalf("initial lock-on size = %.1f, want 15", got)
	}
	if got := lockOnTargetSize(start, start.Add(250*time.Millisecond)); got != 3 {
		t.Fatalf("settled lock-on size = %.1f, want 3", got)
	}
	if got := lockOnTargetTint(start, start); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("initial lock-on tint = %+v", got)
	}
	if got := lockOnTargetTint(start, start.Add(380*time.Millisecond)); got != (color.RGBA{R: 255, G: 12, B: 12, A: 255}) {
		t.Fatalf("low lock-on tint = %+v", got)
	}
}

func TestBeginSpellEffectsInheritCastDuration(t *testing.T) {
	for _, effectID := range []int{effectBeginSpell, effectBeginSpell2, effectBeginSpell3, effectBeginSpell4, effectBeginSpell5, effectBeginSpell6, effectBeginSpell7} {
		spec, ok := worldEffectSpecForID(effectID)
		if !ok {
			t.Fatalf("begin spell effect %d missing", effectID)
		}
		for i, component := range spec.components {
			if component.duration != 0 {
				t.Fatalf("effect %d component %d duration = %s, want inherited cast duration", effectID, i, component.duration)
			}
		}
	}
}

func TestApplyActorActionNotifyRepeatsFireBoltHits(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     19,
		SkillLevel:  4,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      1008,
		HitCount:    4,
		Action:      network.ActorActionSkill,
	})

	if len(mode.worldEffects) != 8 {
		t.Fatalf("world effects = %d, want 8", len(mode.worldEffects))
	}
	for i := 0; i < 4; i++ {
		effect := mode.worldEffects[i]
		if effect.effectID != effectFireBolt || effect.actorID != 300 || effect.targetID != 2000000 {
			t.Fatalf("before-hit effect %d = %+v", i, effect)
		}
		if i > 0 {
			if delay := effect.starts.Sub(mode.worldEffects[i-1].starts); delay != multiHitDelay {
				t.Fatalf("before-hit delay %d = %s, want %s", i, delay, multiHitDelay)
			}
		}
	}
	for i := 4; i < 8; i++ {
		effect := mode.worldEffects[i]
		if effect.effectID != effectFireHit || effect.actorID != 300 {
			t.Fatalf("hit effect %d = %+v", i, effect)
		}
		if i > 4 {
			if delay := effect.starts.Sub(mode.worldEffects[i-1].starts); delay != multiHitDelay {
				t.Fatalf("hit delay %d = %s, want %s", i, delay, multiHitDelay)
			}
		}
	}
	if len(mode.damageFloaters) != 8 {
		t.Fatalf("damage floaters = %d, want 8", len(mode.damageFloaters))
	}
	wantFloaters := []struct {
		text string
		kind damageFloaterKind
	}{
		{text: "252", kind: damageFloaterNormal},
		{text: "252", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "504", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "756", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "1008", kind: damageFloaterCombo},
	}
	for i, want := range wantFloaters {
		floater := mode.damageFloaters[i]
		if floater.text != want.text || floater.kind != want.kind {
			t.Fatalf("floater %d = %+v, want text=%q kind=%d", i, floater, want.text, want.kind)
		}
		if i > 1 {
			if delay := floater.starts.Sub(mode.damageFloaters[i-2].starts); delay != multiHitDelay {
				t.Fatalf("floater %d delay = %s, want %s", i, delay, multiHitDelay)
			}
		}
		if floater.kind == damageFloaterCombo {
			if floater.duration != damageFloaterDuration(damageFloaterCombo) {
				t.Fatalf("combo floater %d duration = %s, want %s", i, floater.duration, damageFloaterDuration(damageFloaterCombo))
			}
			wantVisible := damageFloaterComboTransientDuration()
			if i == len(wantFloaters)-1 {
				wantVisible = damageFloaterDuration(damageFloaterCombo)
			}
			if visible := floater.expires.Sub(floater.starts); visible != wantVisible {
				t.Fatalf("combo floater %d visible = %s, want %s", i, visible, wantVisible)
			}
		}
	}
}

func TestSkillCastAuraEffectMappings(t *testing.T) {
	tests := []struct {
		property uint32
		want     int
	}{
		{property: 0, want: effectBeginSpell},
		{property: 1, want: effectBeginSpell2},
		{property: 2, want: effectBeginSpell5},
		{property: 3, want: effectBeginSpell3},
		{property: 4, want: effectBeginSpell4},
		{property: 5, want: effectBeginSpell7},
		{property: 6, want: effectBeginSpell6},
		{property: 8, want: effectBeginSpell6},
		{property: 9, want: effectBeginSpell},
	}
	for _, tt := range tests {
		if got := skillCastAuraEffectID(tt.property); got != tt.want {
			t.Fatalf("cast aura property %d = %d, want %d", tt.property, got, tt.want)
		}
	}
}

func TestSkillVisualMetadataMappings(t *testing.T) {
	if skillAction(5).action != skillActorActionAttack || skillAction(7).action != skillActorActionAttack {
		t.Fatalf("swordman weapon-action skills = bash:%d magnum:%d", skillAction(5).action, skillAction(7).action)
	}
	if skillAction(8).action != skillActorActionReadyFight {
		t.Fatalf("endure action = %d, want ready fight", skillAction(8).action)
	}
	if skillAction(28).action != skillActorActionSkill {
		t.Fatalf("heal action = %d, want default skill action", skillAction(28).action)
	}
	defaultAction := skillAction(28)
	if !defaultAction.play || defaultAction.repeat || defaultAction.next == nil || defaultAction.next.action != skillActorActionIdle || !defaultAction.next.repeat {
		t.Fatalf("default skill action shape = %+v next=%+v, want robr-style skill action followed by repeating idle", defaultAction, defaultAction.next)
	}
	if size := skillCastGroundSampleSize(19); size != 1 {
		t.Fatalf("firebolt marker size = %.1f, want default 1", size)
	}
	if size := skillCastGroundSampleSize(db.SkillMGThunderstorm); size != 5 {
		t.Fatalf("thunderstorm marker size = %.1f, want roBrowser MagicTarget size 5", size)
	}
}

func TestWindHitEffectSpecMatchesRobrowserRandomSTRAndSFX(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectWindHit)
	if !ok {
		t.Fatal("wind hit effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "windhit%d" || component.strRandMin != 1 || component.strRandMax != 3 || !component.attachedEntity {
		t.Fatalf("wind hit component = %+v", component)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "_hit_fist%d.wav" || spec.sfxRandMin != 1 || spec.sfxRandMax != 3 {
		t.Fatalf("wind hit sfx = %+v rand=%d..%d", spec.sfx, spec.sfxRandMin, spec.sfxRandMax)
	}

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
	if !mode.addWorldEffectAt(ctx, effectWindHit, 2000000, time.Unix(10, 20)) {
		t.Fatal("wind hit effect was not added")
	}
	if len(mode.scheduledSounds) != 1 || len(mode.scheduledSounds[0].paths) != 1 {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
	path := mode.scheduledSounds[0].paths[0]
	if strings.Contains(path, "%d") || !strings.HasPrefix(path, "_hit_fist") || !strings.HasSuffix(path, ".wav") {
		t.Fatalf("scheduled wind hit sound path = %q", path)
	}
}

func TestSkillCastNotifyAddsDurationAura(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: 20, Property: 4, DelayTime: 2500})

	if len(mode.worldEffects) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.worldEffects))
	}
	circle := mode.worldEffects[0]
	if circle.effectID != effectCastRing || circle.actorID != 2000000 || circle.targetID != 0 || circle.duration != 2500*time.Millisecond {
		t.Fatalf("circle = %+v", circle)
	}
	lockon := mode.worldEffects[1]
	if lockon.effectID != effectLockOnTarget || lockon.actorID != 1100 || lockon.targetID != 0 || lockon.duration != 2500*time.Millisecond {
		t.Fatalf("lockon = %+v", lockon)
	}
	aura := mode.worldEffects[2]
	if aura.effectID != effectBeginSpell4 || aura.actorID != 2000000 || aura.targetID != 1100 || aura.duration != 2500*time.Millisecond {
		t.Fatalf("aura = %+v", aura)
	}
	bar, ok := mode.actorCastBars[150000]
	if !ok {
		t.Fatal("local cast bar missing")
	}
	if bar.duration != 2500*time.Millisecond || bar.color != (color.RGBA{R: 0, G: 255, B: 0, A: 255}) {
		t.Fatalf("cast bar = %+v", bar)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("cast animation missing")
	}
	if anim.actionFamily != spriteActionPCReadyFight || anim.duration != 2500*time.Millisecond || anim.hasFixedMotion {
		t.Fatalf("cast animation = %+v", anim)
	}
	if world.Dir != directionFromDelta(10, 20, 12, 20, 4) {
		t.Fatalf("cast dir = %d", world.Dir)
	}
}

func TestSkillCastNotifyHonorsHideCastAura(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: db.SkillACChargearrow, Property: 4, DelayTime: 1200})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %+v, want cast ring and target lock-on", mode.worldEffects)
	}
	if mode.worldEffects[0].effectID != effectCastRing {
		t.Fatalf("effect = %+v, want cast ring", mode.worldEffects[0])
	}
	if mode.worldEffects[1].effectID != effectLockOnTarget || mode.worldEffects[1].actorID != 1100 {
		t.Fatalf("effect = %+v, want target lock-on", mode.worldEffects[1])
	}
	if _, ok := mode.actorCastBars[150000]; !ok {
		t.Fatal("cast bar should remain visible when only hideCastAura is set")
	}
}

func TestSelfTargetSkillCastDoesNotAddLockOnTarget(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.addSkillCastEffects(ctx, db.SkillMGFireball, 3, 2000000, 2000000, 0, 0, 900*time.Millisecond, time.Now(), "self")

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %+v, want cast ring and aura", mode.worldEffects)
	}
	if mode.worldEffects[0].effectID != effectCastRing || mode.worldEffects[1].effectID != effectBeginSpell3 {
		t.Fatalf("effects = %+v", mode.worldEffects)
	}
}

func TestSkillCastEffectsDedupeServerAndLocalFallback(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	start := time.Now()
	mode.addSkillCastEffects(ctx, 19, 3, 2000000, 1100, 0, 0, 2800*time.Millisecond, start, "local")
	mode.addSkillCastEffects(ctx, 19, 3, 2000000, 1100, 0, 0, 2800*time.Millisecond, start.Add(20*time.Millisecond), "server")

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.worldEffects))
	}
	if mode.worldEffects[0].effectID != effectCastRing || mode.worldEffects[1].effectID != effectBeginSpell3 {
		t.Fatalf("effects = %+v", mode.worldEffects)
	}
}

func TestSkillResultAnimationReplacesCastStance(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 4, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: 28, DelayTime: 1200})
	if anim, ok := mode.actorAnims[150000]; !ok || anim.actionFamily != spriteActionPCReadyFight {
		t.Fatalf("cast stance animation = %+v ok=%t", anim, ok)
	}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 1100, SkillID: 28, Amount: 234, Result: 1})
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("skill result animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("skill result animation = %+v, want delivery skill action", anim)
	}
}

func TestGroundSkillCastEffectsAddGroundSampleMarker(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	start := time.Now()
	mode.addSkillCastEffects(ctx, 21, 4, 2000000, 0, 123, 456, 1800*time.Millisecond, start, "local-ground")

	if len(mode.worldEffects) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.worldEffects))
	}
	marker := mode.worldEffects[0]
	if marker.effectID != effectGroundSample || marker.actorID != 0 || marker.x != 123 || marker.y != 456 || marker.duration != 1800*time.Millisecond || marker.size != 5 {
		t.Fatalf("ground marker = %+v", marker)
	}
	if mode.worldEffects[1].effectID != effectCastRing || mode.worldEffects[2].effectID != effectBeginSpell4 {
		t.Fatalf("cast effects = %+v", mode.worldEffects)
	}
}

func TestAcolyteSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AL_DP passive", skillEffectIDs(22))
	expectEffectIDs(t, "AL_DEMONBANE passive", skillEffectIDs(23))
	expectEffectIDs(t, "AL_RUWACH hit", skillHitEffectIDs(24), effectBashHit)
	expectEffectIDs(t, "AL_PNEUMA ground", skillGroundEffectIDs(25), effectPneuma)
	expectEffectIDs(t, "AL_TELEPORT", skillEffectIDs(26))
	expectEffectIDs(t, "AL_WARP", skillEffectIDs(27))
	expectEffectIDs(t, "AL_HEAL", skillEffectIDs(28), effectHeal)
	expectEffectIDs(t, "AL_HEAL hit", skillHitEffectIDs(28), effectHealOffensive)
	expectEffectIDs(t, "AL_INCAGI", skillEffectIDs(29), effectIncAgility)
	expectEffectIDs(t, "AL_DECAGI", skillEffectIDs(30), effectDecAgility)
	expectEffectIDs(t, "AL_HOLYWATER", skillEffectIDs(31), effectAqua)
	expectEffectIDs(t, "AL_CRUCIS", skillEffectIDs(32), effectSignum)
	expectEffectIDs(t, "AL_ANGELUS", skillEffectIDs(33), effectAngelus)
	expectEffectIDs(t, "AL_BLESSING", skillEffectIDs(34), effectBlessing)
	expectEffectIDs(t, "AL_CURE", skillEffectIDs(35), effectCure)
}

func TestImportedSkillEffectFallback(t *testing.T) {
	expectEffectIDs(t, "PR_IMPOSITIO imported", skillEffectIDs(db.SkillPRImpositio), effectImpositio)
	expectEffectIDs(t, "ALL_RESURRECTION imported", skillEffectIDs(db.SkillALLResurrection), effectResurrection, 140)
	expectEffectIDs(t, "PR_SUFFRAGIUM imported", skillEffectIDs(db.SkillPRSuffragium), effectSuffragium)
	expectEffectIDs(t, "PR_ASPERSIO imported", skillEffectIDs(db.SkillPRAspersio), effectAspersio)
	expectEffectIDs(t, "PR_BENEDICTIO imported", skillEffectIDs(db.SkillPRBenedictio), effectBenedictio)
	expectEffectIDs(t, "PR_SANCTUARY imported", skillEffectIDs(db.SkillPRSanctuary), effectSanctuary)
	expectEffectIDs(t, "PR_KYRIE imported", skillEffectIDs(db.SkillPRKyrie), effectKyrie)
	expectEffectIDs(t, "PR_MAGNIFICAT imported", skillEffectIDs(db.SkillPRMagnificat), effectMagnificat)
	expectEffectIDs(t, "PR_GLORIA imported", skillEffectIDs(db.SkillPRGloria), effectGloria)
	expectEffectIDs(t, "PR_LEXDIVINA imported", skillEffectIDs(db.SkillPRLexdivina), effectLexDivina)
	expectEffectIDs(t, "PR_LEXAETERNA imported", skillEffectIDs(db.SkillPRLexaeterna), effectLexAeterna)
	expectEffectIDs(t, "PR_TURNUNDEAD imported hit", skillHitEffectIDs(db.SkillPRTurnundead), effectHolyLight)
	expectEffectIDs(t, "PR_MAGNUS imported", skillEffectIDs(db.SkillPRMagnus), effectMagnus)
	expectEffectIDs(t, "PR_MAGNUS imported ground", skillGroundEffectIDs(db.SkillPRMagnus), effectBottomMagnus)
	expectEffectIDs(t, "PR_SANCTUARY imported ground", skillGroundEffectIDs(db.SkillPRSanctuary), effectBottomSanc)
	expectEffectIDs(t, "PR_SLOWPOISON imported", skillEffectIDs(db.SkillPRSlowpoison), effectSlowPoison)
	expectEffectIDs(t, "PR_STRECOVERY imported", skillEffectIDs(db.SkillPRStrecovery), effectRecovery)
	expectEffectIDs(t, "PR_REDEMPTIO imported empty", skillEffectIDs(db.SkillPRRedemptio))
	expectEffectIDs(t, "WZ_FIREPILLAR imported", skillEffectIDs(db.SkillWZFirepillar), effectFirePillar)
	expectEffectIDs(t, "WZ_FIREPILLAR imported ground", skillGroundEffectIDs(db.SkillWZFirepillar), effectFirePillarOn)
	expectEffectIDs(t, "WZ_FIREPILLAR imported hit", skillHitEffectIDs(db.SkillWZFirepillar), effectFirePillarBomb)
	expectEffectIDs(t, "WZ_SIGHTRASHER imported", skillEffectIDs(db.SkillWZSightrasher), effectSightTrasher)
	expectEffectIDs(t, "WZ_SIGHTRASHER imported hit", skillHitEffectIDs(db.SkillWZSightrasher), effectFireHit)
	expectEffectIDs(t, "WZ_FIREIVY unused", skillEffectIDs(db.SkillWZFireivy))
	expectEffectIDs(t, "WZ_METEOR imported", skillEffectIDs(db.SkillWZMeteor), effectMeteorStorm)
	expectEffectIDs(t, "WZ_METEOR imported hit", skillHitEffectIDs(db.SkillWZMeteor), effectFireHit)
	expectEffectIDs(t, "WZ_JUPITEL imported", skillEffectIDs(db.SkillWZJupitel), effectJupitelThunder)
	expectEffectIDs(t, "WZ_JUPITEL imported before hit", skillBeforeHitEffectIDs(db.SkillWZJupitel), effectJupitelHit)
	expectEffectIDs(t, "WZ_WATERBALL imported self before hit", skillBeforeHitEffectSelfIDs(db.SkillWZWaterball), effectWaterBall)
	expectEffectIDs(t, "WZ_WATERBALL imported caster hit", skillHitEffectOnCasterIDs(db.SkillWZWaterball), effectWaterBall2)
	expectEffectIDs(t, "WZ_VERMILION imported", skillEffectIDs(db.SkillWZVermilion), effectLordVermilion)
	expectEffectIDs(t, "WZ_VERMILION imported hit", skillHitEffectIDs(db.SkillWZVermilion), effectWindHit)
	expectEffectIDs(t, "WZ_ICEWALL imported ground", skillGroundEffectIDs(db.SkillWZIcewall), effectIceWall)
	expectEffectIDs(t, "WZ_FROSTNOVA imported caster", skillEffectOnCasterIDs(db.SkillWZFrostnova), effectFrostDiverHit)
	expectEffectIDs(t, "WZ_FROSTNOVA imported hit", skillHitEffectIDs(db.SkillWZFrostnova), effectColdHit)
	expectEffectIDs(t, "WZ_EARTHSPIKE imported", skillEffectIDs(db.SkillWZEarthspike), effectEarthSpike)
	expectEffectIDs(t, "WZ_EARTHSPIKE imported hit", skillHitEffectIDs(db.SkillWZEarthspike), effectEarthHit)
	expectEffectIDs(t, "WZ_HEAVENDRIVE imported", skillEffectIDs(db.SkillWZHeavendrive), effectHeavenDrive)
	expectEffectIDs(t, "WZ_HEAVENDRIVE imported hit", skillHitEffectIDs(db.SkillWZHeavendrive), effectEarthHit)
	expectEffectIDs(t, "WZ_QUAGMIRE imported ground", skillGroundEffectIDs(db.SkillWZQuagmire), effectQuagmire)
	expectEffectIDs(t, "WZ_STORMGUST imported", skillEffectIDs(db.SkillWZStormgust), effectStormGust)
	expectEffectIDs(t, "WZ_STORMGUST imported hit", skillHitEffectIDs(db.SkillWZStormgust), effectColdHit)
	expectEffectIDs(t, "WZ_ESTIMATION imported empty", skillEffectIDs(db.SkillWZEstimation))
	expectEffectIDs(t, "WZ_SIGHTBLASTER imported", skillEffectIDs(db.SkillWZSightblaster), 601)
	expectEffectIDs(t, "MC_IDENTIFY imported empty", skillEffectIDs(db.SkillMCIdentify))
	expectEffectIDs(t, "MC_VENDING imported empty", skillEffectIDs(db.SkillMCVending))
	expectEffectIDs(t, "MC_CHANGECART imported empty", skillEffectIDs(db.SkillMCChangecart))
	expectEffectIDs(t, "MC_CARTDECORATE imported empty", skillEffectIDs(db.SkillMCCartdecorate))
	expectEffectIDs(t, "BS_REPAIRWEAPON imported", skillEffectIDs(db.SkillBSRepairweapon), effectRepairWeapon)
	expectEffectIDs(t, "BS_HAMMERFALL imported", skillEffectIDs(db.SkillBSHammerfall), effectCrashEarth)
	expectEffectIDs(t, "BS_ADRENALINE imported", skillEffectIDs(db.SkillBSAdrenaline), effectHasteUp)
	expectEffectIDs(t, "BS_ADRENALINE imported begin", skillBeginEffectIDs(db.SkillBSAdrenaline), effectAdrenalineCast)
	expectEffectIDs(t, "BS_WEAPONPERFECT imported", skillEffectIDs(db.SkillBSWeaponperfect), effectWeaponPerfect)
	expectEffectIDs(t, "BS_OVERTHRUST imported", skillEffectIDs(db.SkillBSOverthrust), effectOverthrust)
	expectEffectIDs(t, "BS_MAXIMIZE imported", skillEffectIDs(db.SkillBSMaximize), effectMaximizePower)
	expectEffectIDs(t, "BS_MAXIMIZE imported begin", skillBeginEffectIDs(db.SkillBSMaximize), effectMaximizeSounds)
	expectEffectIDs(t, "BS_ADRENALINE2 imported", skillEffectIDs(db.SkillBSAdrenaline2), effectHasteUp)
	expectEffectIDs(t, "BS_ADRENALINE2 imported begin", skillBeginEffectIDs(db.SkillBSAdrenaline2), effectAdrenalineCast)
	expectEffectIDs(t, "BS_GREED imported", skillEffectIDs(db.SkillBSGreed), effectGreedSound)
	expectEffectIDs(t, "KN_PIERCE imported caster", skillEffectOnCasterIDs(db.SkillKNPierce), effectPierceSelf)
	expectEffectIDs(t, "KN_PIERCE imported hit", skillHitEffectIDs(db.SkillKNPierce), effectEarthHit)
	expectEffectIDs(t, "KN_BRANDISHSPEAR imported", skillEffectIDs(db.SkillKNBrandishspear), effectBrandishSpear)
	expectEffectIDs(t, "KN_BRANDISHSPEAR imported caster", skillEffectOnCasterIDs(db.SkillKNBrandishspear), effectBrandishSpear2)
	expectEffectIDs(t, "KN_SPEARSTAB imported caster", skillEffectOnCasterIDs(db.SkillKNSpearstab), effectSpearStabSelf)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported caster", skillEffectOnCasterIDs(db.SkillKNSpearboomerang), effectSpearBmrSelf)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported before hit", skillBeforeHitEffectIDs(db.SkillKNSpearboomerang), effectSpearProjectile)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported hit", skillHitEffectIDs(db.SkillKNSpearboomerang), effectSpearBoomerang)
	expectEffectIDs(t, "KN_TWOHANDQUICKEN imported", skillEffectIDs(db.SkillKNTwohandquicken), effectTwoHandQuicken)
	expectEffectIDs(t, "KN_ONEHAND imported", skillEffectIDs(db.SkillKNOnehand), effectTwoHandQuicken)
	expectEffectIDs(t, "KN_CHARGEATK imported begin", skillBeginEffectIDs(db.SkillKNChargeatk), effectWhitePulse)
	expectEffectIDs(t, "KN_CHARGEATK imported hit", skillHitEffectIDs(db.SkillKNChargeatk), effectEnemyHitNormal1)
	expectEffectIDs(t, "KN_BOWLINGBASH imported caster", skillEffectOnCasterIDs(db.SkillKNBowlingbash), effectBowlingSelf)
	expectEffectIDs(t, "HT_SKIDTRAP imported", skillEffectIDs(db.SkillHTSkidtrap), effectSkidTrap)
	expectEffectIDs(t, "HT_LANDMINE imported empty", skillEffectIDs(db.SkillHTLandmine))
	expectEffectIDs(t, "HT_ANKLESNARE imported ground", skillGroundEffectIDs(db.SkillHTAnklesnare), effectAnkleSnareGround)
	expectEffectIDs(t, "HT_SHOCKWAVE imported", skillEffectIDs(db.SkillHTShockwave), effectShockwave)
	expectEffectIDs(t, "HT_SHOCKWAVE imported hit", skillHitEffectIDs(db.SkillHTShockwave), effectShockwaveHit)
	expectEffectIDs(t, "HT_SANDMAN imported hit", skillHitEffectIDs(db.SkillHTSandman), effectSandman)
	expectEffectIDs(t, "HT_FLASHER imported hit", skillHitEffectIDs(db.SkillHTFlasher), effectFlasher)
	expectEffectIDs(t, "HT_FREEZINGTRAP imported hit", skillHitEffectIDs(db.SkillHTFreezingtrap), effectFreezingTrap)
	expectEffectIDs(t, "HT_BLASTMINE imported hit", skillHitEffectIDs(db.SkillHTBlastmine), effectBlastMineBomb)
	expectEffectIDs(t, "HT_CLAYMORE imported hit", skillHitEffectIDs(db.SkillHTClaymoretrap), effectClaymore)
	expectEffectIDs(t, "HT_REMOVETRAP imported", skillEffectIDs(db.SkillHTRemovetrap), effectRemoveTrap)
	expectEffectIDs(t, "HT_BLITZBEAT imported", skillEffectIDs(db.SkillHTBlitzbeat), effectBlitzBeat)
	expectEffectIDs(t, "HT_DETECTING imported", skillEffectIDs(db.SkillHTDetecting), effectDetecting)
	expectEffectIDs(t, "HT_SPRINGTRAP imported", skillEffectIDs(db.SkillHTSpringtrap), effectSpringTrap)
	expectEffectIDs(t, "HT_TALKIEBOX imported empty", skillEffectIDs(db.SkillHTTalkiebox))
	expectEffectIDs(t, "HT_POWER imported empty", skillEffectIDs(db.SkillHTPower))
	expectEffectIDs(t, "AS_CLOAKING imported", skillEffectIDs(db.SkillASCloaking), effectCloaking)
	expectEffectIDs(t, "AS_SONICBLOW imported", skillEffectIDs(db.SkillASSonicblow), effectSonicBlow2)
	expectEffectIDs(t, "AS_SONICBLOW imported caster", skillEffectOnCasterIDs(db.SkillASSonicblow), effectSonicBlow)
	expectEffectIDs(t, "AS_SONICBLOW imported hit", skillHitEffectIDs(db.SkillASSonicblow), effectSonicBlowHit)
	expectEffectIDs(t, "AS_GRIMTOOTH imported", skillEffectIDs(db.SkillASGrimtooth), effectGrimtooth)
	expectEffectIDs(t, "AS_GRIMTOOTH imported hit", skillHitEffectIDs(db.SkillASGrimtooth), effectGrimtoothAtk)
	expectEffectIDs(t, "AS_POISONREACT imported", skillEffectIDs(db.SkillASPoisonreact), effectPoisonReact)
	expectEffectIDs(t, "AS_POISONREACT imported hit", skillHitEffectIDs(db.SkillASPoisonreact), effectPoisonReact2)
	expectEffectIDs(t, "AS_VENOMDUST imported", skillEffectIDs(db.SkillASVenomdust), effectVenomDust)
	expectEffectIDs(t, "AS_VENOMDUST imported ground", skillGroundEffectIDs(db.SkillASVenomdust), effectVenomDust2)
	expectEffectIDs(t, "AS_SPLASHER imported", skillEffectIDs(db.SkillASSplasher), effectVenomSplasher)
	expectEffectIDs(t, "NPC_DARKCROSS imported", skillEffectIDs(db.SkillNPCDarkcross), effectDarkGrandCross)
	expectEffectIDs(t, "NPC_DARKSTRIKE imported", skillEffectIDs(db.SkillNPCDarkstrike), effectDarkSoulStrike)
	expectEffectIDs(t, "NPC_STOP imported", skillEffectIDs(db.SkillNPCStop), effectNPCStop)
	expectEffectIDs(t, "NPC_POWERUP imported", skillEffectIDs(db.SkillNPCPowerup), effectNPCPowerUp)
	expectEffectIDs(t, "NPC_DARKBREATH imported", skillEffectIDs(db.SkillNPCDarkbreath), effectDarkBreath)
	expectEffectIDs(t, "NPC_DEFENDER imported", skillEffectIDs(db.SkillNPCDefender), effectDefender)
	expectEffectIDs(t, "NPC_KEEPING imported", skillEffectIDs(db.SkillNPCKeeping), effectKeeping)
	expectEffectIDs(t, "NPC_BLOODDRAIN imported caster", skillEffectOnCasterIDs(db.SkillNPCBlooddrain), effectBloodDrain)
	expectEffectIDs(t, "NPC_ENERGYDRAIN imported caster", skillEffectOnCasterIDs(db.SkillNPCEnergydrain), effectEnergyDrain)
	expectEffectIDs(t, "NPC_EARTHQUAKE imported caster", skillEffectOnCasterIDs(db.SkillNPCEarthquake), effectNPCEarthquake)
	expectEffectIDs(t, "NPC_DRAGONFEAR imported", skillEffectIDs(db.SkillNPCDragonfear), effectDragonFear)
	expectEffectIDs(t, "NPC_WIDEBLEEDING imported caster", skillEffectOnCasterIDs(db.SkillNPCWidebleeding), effectWideBleeding)
	expectEffectIDs(t, "NPC_EVILLAND imported ground", skillGroundEffectIDs(db.SkillNPCEvilland), effectBottomEvilLand)
	expectEffectIDs(t, "NPC_CRITICALWOUND imported hit", skillHitEffectIDs(db.SkillNPCCriticalwound), effectCriticalWound)
	expectEffectIDs(t, "RG_STEALCOIN imported success", skillSuccessEffectIDs(db.SkillRGStealcoin), effectStealCoin, effectRogueCoin)
	expectEffectIDs(t, "RG_BACKSTAP imported hit", skillHitEffectIDs(db.SkillRGBackstap), effectBackStab)
	expectEffectIDs(t, "RG_RAID imported caster", skillEffectOnCasterIDs(db.SkillRGRaid), effectTeiHit3)
	expectEffectIDs(t, "RG_STRIPWEAPON imported success", skillSuccessEffectIDs(db.SkillRGStripweapon), effectStripWeapon)
	expectEffectIDs(t, "RG_STRIPSHIELD imported success", skillSuccessEffectIDs(db.SkillRGStripshield), effectStripShield)
	expectEffectIDs(t, "RG_STRIPARMOR imported success", skillSuccessEffectIDs(db.SkillRGStriparmor), effectStripArmor)
	expectEffectIDs(t, "RG_STRIPHELM imported success", skillSuccessEffectIDs(db.SkillRGStriphelm), effectStripHelm)
	expectEffectIDs(t, "RG_INTIMIDATE imported", skillEffectIDs(db.SkillRGIntimidate), effectIntimidate)
	expectEffectIDs(t, "RG_GRAFFITI imported empty", skillEffectIDs(db.SkillRGGraffiti))
	expectEffectIDs(t, "RG_FLAGGRAFFITI imported empty", skillEffectIDs(db.SkillRGFlaggraffiti))
	expectEffectIDs(t, "RG_CLEANER imported empty", skillEffectIDs(db.SkillRGCleaner))
	expectEffectIDs(t, "RG_CLOSECONFINE imported", skillEffectIDs(db.SkillRGCloseconfine), 602)
	expectEffectIDs(t, "RG_CLOSECONFINE imported ground", skillGroundEffectIDs(db.SkillRGCloseconfine), effectNPCStop2)
	expectEffectIDs(t, "AM_PHARMACY imported empty", skillEffectIDs(db.SkillAMPharmacy))
	expectEffectIDs(t, "AM_DEMONSTRATION imported ground", skillGroundEffectIDs(db.SkillAMDemonstration), effectDemonstration)
	expectEffectIDs(t, "AM_ACIDTERROR imported before hit", skillBeforeHitEffectIDs(db.SkillAMAcidterror), effectThrowItem)
	expectEffectIDs(t, "AM_POTIONPITCHER imported", skillEffectIDs(db.SkillAMPotionpitcher), 299)
	expectEffectIDs(t, "AM_CANNIBALIZE imported empty", skillEffectIDs(db.SkillAMCannibalize))
	expectEffectIDs(t, "AM_SPHEREMINE imported empty", skillEffectIDs(db.SkillAMSpheremine))
	expectEffectIDs(t, "AM_CP_WEAPON imported", skillEffectIDs(db.SkillAMCpWeapon), effectChemicalProt)
	expectEffectIDs(t, "AM_CP_SHIELD imported", skillEffectIDs(db.SkillAMCpShield), effectChemicalProt)
	expectEffectIDs(t, "AM_CP_ARMOR imported", skillEffectIDs(db.SkillAMCpArmor), effectChemicalProt)
	expectEffectIDs(t, "AM_CP_HELM imported", skillEffectIDs(db.SkillAMCpHelm), effectChemicalProt)
	expectEffectIDs(t, "AM_CALLHOMUN imported empty", skillEffectIDs(db.SkillAMCallhomun))
	expectEffectIDs(t, "AM_REST imported empty", skillEffectIDs(db.SkillAMRest))
	expectEffectIDs(t, "AM_RESURRECTHOMUN imported empty", skillEffectIDs(db.SkillAMResurrecthomun))
	expectEffectIDs(t, "CR_AUTOGUARD imported", skillEffectIDs(db.SkillCRAutoguard), effectGuard)
	expectEffectIDs(t, "CR_SHIELDCHARGE imported", skillEffectIDs(db.SkillCRShieldcharge), effectShieldCharge)
	expectEffectIDs(t, "CR_SHIELDBOOMERANG imported", skillEffectIDs(db.SkillCRShieldboomerang), effectShieldBoomer)
	expectEffectIDs(t, "CR_SHIELDBOOMERANG imported before hit", skillBeforeHitEffectIDs(db.SkillCRShieldboomerang), effectShieldProjectile)
	expectEffectIDs(t, "CR_REFLECTSHIELD imported", skillEffectIDs(db.SkillCRReflectshield), effectReflectShield)
	expectEffectIDs(t, "CR_HOLYCROSS imported", skillEffectIDs(db.SkillCRHolycross), effectHolyCross)
	expectEffectIDs(t, "CR_GRANDCROSS imported", skillEffectIDs(db.SkillCRGrandcross), effectGrandCross)
	expectEffectIDs(t, "CR_DEVOTION imported", skillEffectIDs(db.SkillCRDevotion), effectDevotion)
	expectEffectIDs(t, "CR_PROVIDENCE imported", skillEffectIDs(db.SkillCRProvidence), effectProvidence)
	expectEffectIDs(t, "CR_DEFENDER imported", skillEffectIDs(db.SkillCRDefender), effectCrusaderDef)
	expectEffectIDs(t, "CR_SPEARQUICKEN imported", skillEffectIDs(db.SkillCRSpearquicken), effectSpearQuicken)
	expectEffectIDs(t, "MO_CALLSPIRITS imported empty", skillEffectIDs(db.SkillMOCallspirits))
	expectEffectIDs(t, "MO_ABSORBSPIRITS imported self success", skillSuccessEffectSelfIDs(db.SkillMOAbsorbspirits), effectAbsorbSpirits)
	expectEffectIDs(t, "MO_BODYRELOCATION imported empty", skillEffectIDs(db.SkillMOBodyrelocation))
	expectEffectIDs(t, "MO_INVESTIGATE imported", skillEffectIDs(db.SkillMOInvestigate), effectChimto)
	expectEffectIDs(t, "MO_FINGEROFFENSIVE imported", skillEffectIDs(db.SkillMOFingeroffensive), effectTanji)
	expectEffectIDs(t, "MO_STEELBODY imported", skillEffectIDs(db.SkillMOSteelbody), effectSteelBody, effectQuake)
	expectEffectIDs(t, "MO_BLADESTOP imported empty", skillEffectIDs(db.SkillMOBladestop))
	expectEffectIDs(t, "MO_EXPLOSIONSPIRITS imported caster", skillEffectOnCasterIDs(db.SkillMOExplosionspirits), effectGumgang2, effectQuake)
	expectEffectIDs(t, "MO_EXTREMITYFIST imported", skillEffectIDs(db.SkillMOExtremityfist), effectBeginAsura, effectQuake)
	expectEffectIDs(t, "MO_EXTREMITYFIST imported hit", skillHitEffectIDs(db.SkillMOExtremityfist), effectTeiHit1X)
	expectEffectIDs(t, "MO_TRIPLEATTACK imported", skillEffectIDs(db.SkillMOTripleattack), effectTripleAttack)
	expectEffectIDs(t, "MO_CHAINCOMBO imported", skillEffectIDs(db.SkillMOChaincombo), effectTeiHit1, effectChainCombo)
	expectEffectIDs(t, "MO_CHAINCOMBO imported caster", skillEffectOnCasterIDs(db.SkillMOChaincombo), effectGumgang3)
	expectEffectIDs(t, "MO_COMBOFINISH imported", skillEffectIDs(db.SkillMOCombofinish), 330, effectQuake)
	expectEffectIDs(t, "SA_CASTCANCEL imported empty", skillEffectIDs(db.SkillSACastcancel))
	expectEffectIDs(t, "SA_MAGICROD imported success", skillSuccessEffectIDs(db.SkillSAMagicrod), effectMagicRod)
	expectEffectIDs(t, "SA_SPELLBREAKER imported success", skillSuccessEffectIDs(db.SkillSASpellbreaker), effectSpellBreaker)
	expectEffectIDs(t, "SA_AUTOSPELL imported empty", skillEffectIDs(db.SkillSAAutospell))
	expectEffectIDs(t, "SA_FLAMELAUNCHER imported success", skillSuccessEffectIDs(db.SkillSAFlamelauncher), effectFlameLauncher)
	expectEffectIDs(t, "SA_FROSTWEAPON imported success", skillSuccessEffectIDs(db.SkillSAFrostweapon), effectFrostWeapon)
	expectEffectIDs(t, "SA_LIGHTNINGLOADER imported success", skillSuccessEffectIDs(db.SkillSALightningloader), effectLightningLoad)
	expectEffectIDs(t, "SA_SEISMICWEAPON imported success", skillSuccessEffectIDs(db.SkillSASeismicweapon), effectSeismicWeapon)
	expectEffectIDs(t, "SA_DISPELL imported success", skillSuccessEffectIDs(db.SkillSADispell), effectDispell)
	expectEffectIDs(t, "SA_VOLCANO imported caster", skillEffectOnCasterIDs(db.SkillSAVolcano), 225)
	expectEffectIDs(t, "SA_VOLCANO imported ground", skillGroundEffectIDs(db.SkillSAVolcano), effectBottomVolcano)
	expectEffectIDs(t, "SA_DELUGE imported caster", skillEffectOnCasterIDs(db.SkillSADeluge), 236)
	expectEffectIDs(t, "SA_DELUGE imported ground", skillGroundEffectIDs(db.SkillSADeluge), effectBottomDeluge)
	expectEffectIDs(t, "SA_VIOLENTGALE imported caster", skillEffectOnCasterIDs(db.SkillSAViolentgale), 237)
	expectEffectIDs(t, "SA_VIOLENTGALE imported ground", skillGroundEffectIDs(db.SkillSAViolentgale), effectBottomViolent)
	expectEffectIDs(t, "SA_LANDPROTECTOR imported caster", skillEffectOnCasterIDs(db.SkillSALandprotector), 238)
	expectEffectIDs(t, "SA_LANDPROTECTOR imported ground", skillGroundEffectIDs(db.SkillSALandprotector), effectBottomLand)
	expectEffectIDs(t, "SA_ABRACADABRA imported empty", skillEffectIDs(db.SkillSAAbracadabra))
	for _, skillID := range []uint16{
		db.SkillSAMonocell,
		db.SkillSAClasschange,
		db.SkillSASummonmonster,
		db.SkillSAReverseorcish,
		db.SkillSADeath,
		db.SkillSAFortune,
		db.SkillSATamingmonster,
		db.SkillSAQuestion,
		db.SkillSAGravity,
		db.SkillSALevelup,
		db.SkillSAInstantdeath,
		db.SkillSAFullrecovery,
		db.SkillSAComa,
	} {
		expectEffectIDs(t, "SA_ABRACADABRA result imported empty", skillEffectIDs(skillID))
	}
	expectEffectIDs(t, "BD_ADAPTATION imported empty", skillEffectIDs(db.SkillBDAdaptation))
	expectEffectIDs(t, "BD_ENCORE imported empty", skillEffectIDs(db.SkillBDEncore))
	expectEffectIDs(t, "BD_LULLABY imported", skillEffectIDs(db.SkillBDLullaby), effectBottomLullaby)
	expectEffectIDs(t, "BD_LULLABY imported ground", skillGroundEffectIDs(db.SkillBDLullaby), effectBottomLullaby)
	expectEffectIDs(t, "BD_RICHMANKIM imported", skillEffectIDs(db.SkillBDRichmankim), effectBottomRichKim)
	expectEffectIDs(t, "BD_RICHMANKIM imported ground", skillGroundEffectIDs(db.SkillBDRichmankim), effectBottomRichKim)
	expectEffectIDs(t, "BD_ETERNALCHAOS imported", skillEffectIDs(db.SkillBDEternalchaos), effectBottomChaos)
	expectEffectIDs(t, "BD_ETERNALCHAOS imported ground", skillGroundEffectIDs(db.SkillBDEternalchaos), effectBottomChaos)
	expectEffectIDs(t, "BD_DRUMBATTLEFIELD imported", skillEffectIDs(db.SkillBDDrumbattlefield), effectBottomDrum)
	expectEffectIDs(t, "BD_DRUMBATTLEFIELD imported ground", skillGroundEffectIDs(db.SkillBDDrumbattlefield), effectBottomDrum)
	expectEffectIDs(t, "BD_RINGNIBELUNGEN imported", skillEffectIDs(db.SkillBDRingnibelungen), effectBottomNibelung)
	expectEffectIDs(t, "BD_RINGNIBELUNGEN imported ground", skillGroundEffectIDs(db.SkillBDRingnibelungen), effectBottomNibelung)
	expectEffectIDs(t, "BD_ROKISWEIL imported", skillEffectIDs(db.SkillBDRokisweil), effectBottomRoki)
	expectEffectIDs(t, "BD_ROKISWEIL imported ground", skillGroundEffectIDs(db.SkillBDRokisweil), effectBottomRoki)
	expectEffectIDs(t, "BD_INTOABYSS imported", skillEffectIDs(db.SkillBDIntoabyss), effectBottomAbyss)
	expectEffectIDs(t, "BD_INTOABYSS imported ground", skillGroundEffectIDs(db.SkillBDIntoabyss), effectBottomAbyss)
	expectEffectIDs(t, "BD_SIEGFRIED imported", skillEffectIDs(db.SkillBDSiegfried), effectBottomSieg)
	expectEffectIDs(t, "BD_SIEGFRIED imported ground", skillGroundEffectIDs(db.SkillBDSiegfried), effectBottomSieg)
	expectEffectIDs(t, "BA_MUSICALLESSON imported empty", skillEffectIDs(db.SkillBaMusicallesson))
	expectEffectIDs(t, "BA_MUSICALSTRIKE imported before hit", skillBeforeHitEffectIDs(db.SkillBaMusicalstrike), effectArrowShot)
	if !skillHidesCastAura(db.SkillBaMusicalstrike) {
		t.Fatal("BA_MUSICALSTRIKE should hide cast aura like robr")
	}
	expectEffectIDs(t, "BA_DISSONANCE imported ground", skillGroundEffectIDs(db.SkillBaDissonance), effectBottomDissonance)
	expectEffectIDs(t, "BA_FROSTJOKE imported begin", skillBeginEffectIDs(db.SkillBaFrostjoke), effectTalkFrostJoke)
	expectEffectIDs(t, "BA_WHISTLE imported", skillEffectIDs(db.SkillBaWhistle), effectBottomWhistle)
	expectEffectIDs(t, "BA_WHISTLE imported ground", skillGroundEffectIDs(db.SkillBaWhistle), effectBottomWhistle)
	expectEffectIDs(t, "BA_ASSASSINCROSS imported", skillEffectIDs(db.SkillBaAssassincross), effectBottomSinX)
	expectEffectIDs(t, "BA_ASSASSINCROSS imported ground", skillGroundEffectIDs(db.SkillBaAssassincross), effectBottomSinX)
	expectEffectIDs(t, "BA_POEMBRAGI imported", skillEffectIDs(db.SkillBaPoembragi), effectBottomBragi)
	expectEffectIDs(t, "BA_POEMBRAGI imported ground", skillGroundEffectIDs(db.SkillBaPoembragi), effectBottomBragi)
	expectEffectIDs(t, "BA_APPLEIDUN imported", skillEffectIDs(db.SkillBaAppleidun), effectBottomApple)
	expectEffectIDs(t, "BA_APPLEIDUN imported ground", skillGroundEffectIDs(db.SkillBaAppleidun), effectBottomApple)
	expectEffectIDs(t, "BA_PANGVOICE imported success", skillSuccessEffectIDs(db.SkillBaPangvoice), effectFVoice)
	expectEffectIDs(t, "DC_DANCINGLESSON imported empty", skillEffectIDs(db.SkillDCDancinglesson))
	expectEffectIDs(t, "DC_THROWARROW imported before hit", skillBeforeHitEffectIDs(db.SkillDCThrowarrow), effectArrowShot)
	if !skillHidesCastAura(db.SkillDCThrowarrow) {
		t.Fatal("DC_THROWARROW should hide cast aura like robr")
	}
	expectEffectIDs(t, "DC_UGLYDANCE imported ground", skillGroundEffectIDs(db.SkillDCUglydance), effectBottomUglyDance)
	expectEffectIDs(t, "DC_SCREAM imported begin", skillBeginEffectIDs(db.SkillDCScream), effectTalkScream)
	expectEffectIDs(t, "DC_HUMMING imported", skillEffectIDs(db.SkillDCHumming), effectBottomHumming)
	expectEffectIDs(t, "DC_HUMMING imported ground", skillGroundEffectIDs(db.SkillDCHumming), effectBottomHumming)
	expectEffectIDs(t, "DC_DONTFORGETME imported", skillEffectIDs(db.SkillDCDontforgetme), effectBottomForget)
	expectEffectIDs(t, "DC_DONTFORGETME imported ground", skillGroundEffectIDs(db.SkillDCDontforgetme), effectBottomForget)
	expectEffectIDs(t, "DC_FORTUNEKISS imported", skillEffectIDs(db.SkillDCFortunekiss), effectBottomFortune)
	expectEffectIDs(t, "DC_FORTUNEKISS imported ground", skillGroundEffectIDs(db.SkillDCFortunekiss), effectBottomFortune)
	expectEffectIDs(t, "DC_SERVICEFORYOU imported", skillEffectIDs(db.SkillDCServiceforyou), effectBottomService)
	expectEffectIDs(t, "DC_SERVICEFORYOU imported ground", skillGroundEffectIDs(db.SkillDCServiceforyou), effectBottomService)
	expectEffectIDs(t, "DC_WINKCHARM imported success", skillSuccessEffectIDs(db.SkillDCWinkcharm), effectWink)
	expectEffectIDs(t, "SL_KAIZEL imported", skillEffectIDs(db.SkillSLKaizel), effectKaizel)
	expectEffectIDs(t, "SL_STUN imported", skillEffectIDs(db.SkillSLStun), effectStin3)
	expectEffectIDs(t, "SL_SMA imported", skillEffectIDs(db.SkillSLSma), effectStin2)
	expectEffectIDs(t, "SL_SWOO imported", skillEffectIDs(db.SkillSLSwoo), effectM07)
	expectEffectIDs(t, "SL_SKA imported", skillEffectIDs(db.SkillSLSka), effectSteelBody, effectGumgang2)
	expectEffectIDs(t, "AM_BERSERKPITCHER imported", skillEffectIDs(db.SkillAMBerserkpitcher), effectItemFast3)
	expectEffectIDs(t, "AM_BERSERKPITCHER imported before hit", skillBeforeHitEffectIDs(db.SkillAMBerserkpitcher), 541)
	expectEffectIDs(t, "AM_TWILIGHT1 imported", skillEffectIDs(db.SkillAMTwilight1), 497)
	expectEffectIDs(t, "AM_TWILIGHT2 imported", skillEffectIDs(db.SkillAMTwilight2), 498)
	expectEffectIDs(t, "AM_TWILIGHT3 imported", skillEffectIDs(db.SkillAMTwilight3), 499)
	expectEffectIDs(t, "CR_ALCHEMY imported empty", skillEffectIDs(db.SkillCRAlchemy))
	expectEffectIDs(t, "CR_SYNTHESISPOTION imported empty", skillEffectIDs(db.SkillCRSynthesispotion))
	expectEffectIDs(t, "CR_SLIMPITCHER imported empty", skillEffectIDs(db.SkillCRSlimpitcher))
	expectEffectIDs(t, "CR_FULLPROTECTION imported", skillEffectIDs(db.SkillCRFullprotection), effectChemicalProt, 500)
	expectEffectIDs(t, "CR_CULTIVATION imported empty", skillEffectIDs(db.SkillCRCultivation))
	expectEffectIDs(t, "SA_CREATECON imported empty", skillEffectIDs(db.SkillSACreatecon))
	expectEffectIDs(t, "SA_ELEMENTWATER imported", skillEffectIDs(db.SkillSAElementwater), effectFrostWeapon)
	expectEffectIDs(t, "SA_ELEMENTGROUND imported", skillEffectIDs(db.SkillSAElementground), effectSeismicWeapon)
	expectEffectIDs(t, "SA_ELEMENTFIRE imported", skillEffectIDs(db.SkillSAElementfire), effectFlameLauncher)
	expectEffectIDs(t, "SA_ELEMENTWIND imported", skillEffectIDs(db.SkillSAElementwind), effectLightningLoad)
	expectEffectIDs(t, "MO_KITRANSLATION imported empty", skillEffectIDs(db.SkillMOKitranslation))
	expectEffectIDs(t, "MO_BALKYOUNG imported", skillEffectIDs(db.SkillMOBalkyoung), 514)
	expectEffectIDs(t, "LK_PARRYING imported", skillEffectIDs(db.SkillLKParrying), effectGuard)
	expectEffectIDs(t, "LK_SPIRALPIERCE imported", skillEffectIDs(db.SkillLKSpiralpierce), effectMagnum2)
	expectEffectIDs(t, "LK_SPIRALPIERCE imported begin", skillBeginEffectIDs(db.SkillLKSpiralpierce), effectSpiralBeforeCast)
	expectEffectIDs(t, "LK_SPIRALPIERCE imported hit", skillHitEffectIDs(db.SkillLKSpiralpierce), effectSpearHitSound)
	expectEffectIDs(t, "LK_AURABLADE imported", skillEffectIDs(db.SkillLKAurablade), effectAuraBlade)
	expectEffectIDs(t, "LK_AURABLADE imported begin", skillBeginEffectIDs(db.SkillLKAurablade), effectWhitePulse)
	expectEffectIDs(t, "LK_CONCENTRATION imported", skillEffectIDs(db.SkillLKConcentration), effectLKConcentration)
	expectEffectIDs(t, "LK_TENSIONRELAX imported empty", skillEffectIDs(db.SkillLKTensionrelax))
	expectEffectIDs(t, "LK_BERSERK imported", skillEffectIDs(db.SkillLKBerserk), effectRedBody, effectQuake)
	expectEffectIDs(t, "LK_FURY imported", skillEffectIDs(db.SkillLKFury), effectRedBody, effectQuake)
	expectEffectIDs(t, "HP_ASSUMPTIO imported", skillEffectIDs(db.SkillHPAssumptio), effectAssumptio2)
	expectEffectIDs(t, "HP_BASILICA imported ground", skillGroundEffectIDs(db.SkillHPBasilica), effectBottomBasilica)
	expectEffectIDs(t, "HP_MEDITATIO has no robr effect row", skillEffectIDs(db.SkillHPMeditatio))
	expectEffectIDs(t, "HP_MANARECHARGE has no robr effect row", skillEffectIDs(db.SkillHPManarecharge))
	expectEffectIDs(t, "HW_MAGICCRASHER imported", skillEffectIDs(db.SkillHWMagiccrasher), effectMagicCrasher)
	expectEffectIDs(t, "HW_MAGICPOWER imported", skillEffectIDs(db.SkillHWMagicpower), effectMagicPower)
	expectEffectIDs(t, "HW_MAGICPOWER imported begin", skillBeginEffectIDs(db.SkillHWMagicpower), effectBashBegin)
	if !skillHidesCastAura(db.SkillHWMagicpower) {
		t.Fatal("HW_MAGICPOWER should hide cast aura like robr")
	}
	expectEffectIDs(t, "HW_SOULDRAIN imported caster", skillEffectOnCasterIDs(db.SkillHWSouldrain), effectEnergyDrain)
	expectEffectIDs(t, "HW_NAPALMVULCAN imported", skillEffectIDs(db.SkillHWNapalmvulcan), 401)
	expectEffectIDs(t, "HW_GANBANTEIN imported", skillEffectIDs(db.SkillHWGanbantein), 223)
	expectEffectIDs(t, "HW_GANBANTEIN imported ground", skillGroundEffectIDs(db.SkillHWGanbantein), 224)
	expectEffectIDs(t, "HW_GRAVITATION imported ground", skillGroundEffectIDs(db.SkillHWGravitation), effectGravitation)
	expectEffectIDs(t, "PA_PRESSURE imported before hit", skillBeforeHitEffectIDs(db.SkillPaPressure), effectPressure)
	expectEffectIDs(t, "PA_SACRIFICE imported", skillEffectIDs(db.SkillPaSacrifice), effectBash3D)
	expectEffectIDs(t, "PA_GOSPEL imported", skillEffectIDs(db.SkillPaGospel), effectBottomGospel)
	expectEffectIDs(t, "PA_GOSPEL imported ground", skillGroundEffectIDs(db.SkillPaGospel), effectGospelGround)
	expectEffectIDs(t, "PA_SHIELDCHAIN imported before hit", skillBeforeHitEffectIDs(db.SkillPaShieldchain), effectShieldProjectile)
	expectEffectIDs(t, "CH_PALMSTRIKE imported hit", skillHitEffectIDs(db.SkillChPalmstrike), effectHitLine2, effectQuake)
	expectEffectIDs(t, "CH_TIGERFIST imported", skillEffectIDs(db.SkillChTigerfist), effectBash3D2, effectQuake)
	expectEffectIDs(t, "CH_CHAINCRUSH imported", skillEffectIDs(db.SkillChChaincrush), effectChemical2Dash)
	expectEffectIDs(t, "CH_SOULCOLLECT imported begin", skillBeginEffectIDs(db.SkillChSoulcollect), effectPortal5, effectBeginSpell)
	expectEffectIDs(t, "PF_HPCONVERSION imported", skillEffectIDs(db.SkillPFHpconversion), effectEnergyDrain3)
	expectEffectIDs(t, "PF_HPCONVERSION imported caster", skillEffectOnCasterIDs(db.SkillPFHpconversion), effectEnergyDrain2)
	expectEffectIDs(t, "PF_HPCONVERSION imported self success", skillSuccessEffectSelfIDs(db.SkillPFHpconversion), effectTransBlueBody)
	expectEffectIDs(t, "PF_SOULCHANGE imported", skillEffectIDs(db.SkillPFSoulchange), effectLineLink2)
	expectEffectIDs(t, "PF_SOULCHANGE imported success", skillSuccessEffectIDs(db.SkillPFSoulchange), 385)
	expectEffectIDs(t, "PF_SOULBURN imported", skillEffectIDs(db.SkillPFSoulburn), effectSoulBurn)
	expectEffectIDs(t, "PF_MINDBREAKER imported success", skillSuccessEffectIDs(db.SkillPFMindbreaker), effectMagicCrasher2)
	expectEffectIDs(t, "PF_MEMORIZE imported", skillEffectIDs(db.SkillPFMemorize), 505)
	expectEffectIDs(t, "PF_FOGWALL imported ground", skillGroundEffectIDs(db.SkillPFFogwall), effectFogWallGround)
	expectEffectIDs(t, "PF_SPIDERWEB imported ground", skillGroundEffectIDs(db.SkillPFSpiderweb), effectBottomSpider)
	expectEffectIDs(t, "PF_DOUBLECASTING imported", skillEffectIDs(db.SkillPFDoublecasting), 521)
	expectEffectIDs(t, "ASC_BREAKER imported before hit", skillBeforeHitEffectIDs(db.SkillASCBreaker), effectSoulBreaker)
	expectEffectIDs(t, "ASC_METEORASSAULT imported caster", skillEffectOnCasterIDs(db.SkillASCMeteorassault), effectSoulBreaker2)
	if !skillHidesCastAura(db.SkillASCMeteorassault) {
		t.Fatal("ASC_METEORASSAULT should hide cast aura like robr")
	}
	expectEffectIDs(t, "ASC_CDP imported empty", skillEffectIDs(db.SkillASCCdp))
	expectEffectIDs(t, "SN_SIGHT imported", skillEffectIDs(db.SkillSNSight), effectTrueSight)
	expectEffectIDs(t, "SN_FALCONASSAULT imported", skillEffectIDs(db.SkillSNFalconassault), effectFalconAssault)
	expectEffectIDs(t, "HT_PHANTASMIC imported before hit", skillBeforeHitEffectIDs(db.SkillHTPhantasmic), effectArrowShot)
	expectEffectIDs(t, "HT_PHANTASMIC imported hit", skillHitEffectIDs(db.SkillHTPhantasmic), effectBashHit)
	expectEffectIDs(t, "SN_SHARPSHOOTING imported begin", skillBeginEffectIDs(db.SkillSNSharpshooting), effectSharpShootingCast)
	expectEffectIDs(t, "SN_SHARPSHOOTING imported before hit", skillBeforeHitEffectIDs(db.SkillSNSharpshooting), effectArrowShot)
	expectEffectIDs(t, "SN_SHARPSHOOTING imported hit", skillHitEffectIDs(db.SkillSNSharpshooting), effectTripleAttack2)
	expectEffectIDs(t, "SN_WINDWALK imported", skillEffectIDs(db.SkillSNWindwalk), effectPortal4)
	expectEffectIDs(t, "WS_MELTDOWN imported", skillEffectIDs(db.SkillWSMeltdown), effectMeltdown)
	expectEffectIDs(t, "WS_CREATECOIN imported empty", skillEffectIDs(db.SkillWSCreatecoin))
	expectEffectIDs(t, "WS_CREATENUGGET imported empty", skillEffectIDs(db.SkillWSCreatenugget))
	expectEffectIDs(t, "WS_CARTBOOST imported", skillEffectIDs(db.SkillWSCartboost), effectCartBoost)
	expectEffectIDs(t, "WS_SYSTEMCREATE imported empty", skillEffectIDs(db.SkillWSSystemcreate))
	expectEffectIDs(t, "WS_WEAPONREFINE imported empty", skillEffectIDs(db.SkillWSWeaponrefine))
	expectEffectIDs(t, "WS_CARTTERMINATION imported", skillEffectIDs(db.SkillWSCarttermination), 518)
	expectEffectIDs(t, "WS_OVERTHRUSTMAX imported", skillEffectIDs(db.SkillWSOverthrustmax), effectOverthrust)
	expectEffectIDs(t, "ASC_EDP imported", skillEffectIDs(db.SkillASCEdp), effectEDP)
	expectEffectIDs(t, "ST_CHASEWALK imported begin", skillBeginEffectIDs(db.SkillSTChasewalk), effectCastSpin)
	expectEffectIDs(t, "ST_REJECTSWORD imported", skillEffectIDs(db.SkillSTRejectsword), effectRejectSword)
	expectEffectIDs(t, "ST_PRESERVE imported", skillEffectIDs(db.SkillSTPreserve), effectPreserve)
	expectEffectIDs(t, "ST_PRESERVE imported begin", skillBeginEffectIDs(db.SkillSTPreserve), effectSharpShootingCast)
	expectEffectIDs(t, "ST_FULLSTRIP imported success", skillSuccessEffectIDs(db.SkillSTFullstrip), 495)
	expectEffectIDs(t, "CG_ARROWVULCAN imported", skillEffectIDs(db.SkillCGArrowvulcan), effectTripleAttack3)
	expectEffectIDs(t, "CG_ARROWVULCAN imported before hit", skillBeforeHitEffectIDs(db.SkillCGArrowvulcan), effectArrowShot)
	expectEffectIDs(t, "CG_MOONLIT imported", skillEffectIDs(db.SkillCGMoonlit), effectMoonlit)
	expectEffectIDs(t, "CG_MOONLIT imported ground", skillGroundEffectIDs(db.SkillCGMoonlit), effectMoonlit)
	expectEffectIDs(t, "CG_MARIONETTE imported", skillEffectIDs(db.SkillCGMarionette), 395)
	expectEffectIDs(t, "CG_MARIONETTE imported hit", skillHitEffectIDs(db.SkillCGMarionette), 396)
	expectEffectIDs(t, "CG_LONGINGFREEDOM imported", skillEffectIDs(db.SkillCGLongingfreedom), 500)
	expectEffectIDs(t, "CG_HERMODE imported music", skillEffectIDs(db.SkillCGHermode), effectHermodeMusic)
	expectEffectIDs(t, "CG_HERMODE imported ground", skillGroundEffectIDs(db.SkillCGHermode), effectBottomHermode)
	expectEffectIDs(t, "CG_TAROTCARD imported success", skillSuccessEffectIDs(db.SkillCGTarotcard), 500)
	expectEffectIDs(t, "CR_ACIDDEMONSTRATION imported", skillEffectIDs(db.SkillCRAciddemonstration), effectAcidDemon)
	expectEffectIDs(t, "SL_KAAHI imported", skillEffectIDs(db.SkillSLKaahi), effectHated)
	expectEffectIDs(t, "SL_STIN imported", skillEffectIDs(db.SkillSLStin), effectStin)
	expectEffectIDs(t, "LK_HEADCRUSH imported begin", skillBeginEffectIDs(db.SkillLKHeadcrush), effectBash3D3)
	expectEffectIDs(t, "LK_HEADCRUSH imported hit", skillHitEffectIDs(db.SkillLKHeadcrush), effectEnemyHitNormal1)
	expectEffectIDs(t, "LK_JOINTBEAT imported begin", skillBeginEffectIDs(db.SkillLKJointbeat), effectBash3D4)
	expectEffectIDs(t, "LK_JOINTBEAT imported hit", skillHitEffectIDs(db.SkillLKJointbeat), effectEnemyHitNormal1)
	expectEffectIDs(t, "TK_JUMPKICK imported hit", skillHitEffectIDs(db.SkillTKJumpkick), effectJumpKick)
	expectEffectIDs(t, "GS_INCREASING imported", skillEffectIDs(db.SkillGSIncreasing), effectNPCPowerUp)
	expectEffectIDs(t, "GS_TRIPLEACTION imported", skillEffectIDs(db.SkillGSTripleaction), effectTripleAction)
	expectEffectIDs(t, "GS_BULLSEYE imported", skillEffectIDs(db.SkillGSBullseye), effectBullseye)
	expectEffectIDs(t, "GS_MAGICALBULLET imported", skillEffectIDs(db.SkillGSMagicalbullet), effectMagicalBullet)
	expectEffectIDs(t, "GS_TRACKING imported", skillEffectIDs(db.SkillGSTracking), effectTrackCasting)
	expectEffectIDs(t, "GS_TRACKING imported hit", skillHitEffectIDs(db.SkillGSTracking), effectTracking)
	expectEffectIDs(t, "GS_DISARM imported", skillEffectIDs(db.SkillGSDisarm), effectRGCoin3)
	expectEffectIDs(t, "GS_RAPIDSHOWER imported", skillEffectIDs(db.SkillGSRapidshower), effectRapidShower)
	expectEffectIDs(t, "GS_DESPERADO imported", skillEffectIDs(db.SkillGSDesperado), effectDesperado)
	expectEffectIDs(t, "GS_DUST imported", skillEffectIDs(db.SkillGSDust), effectBash3D5)
	expectEffectIDs(t, "GS_FULLBUSTER imported", skillEffectIDs(db.SkillGSFullbuster), effectM02)
	expectEffectIDs(t, "GS_SPREADATTACK imported", skillEffectIDs(db.SkillGSSpreadattack), effectSpreadAttack)
	expectEffectIDs(t, "NJ_SYURIKEN imported before hit", skillBeforeHitEffectIDs(db.SkillNJSyuriken), effectThrowItem7)
	expectEffectIDs(t, "NJ_KUNAI imported before hit", skillBeforeHitEffectIDs(db.SkillNJKunai), effectThrowItem8)
	expectEffectIDs(t, "NJ_HUUMA imported before hit", skillBeforeHitEffectIDs(db.SkillNJHuuma), effectThrowItem9)
	expectEffectIDs(t, "NJ_ZENYNAGE imported before hit", skillBeforeHitEffectIDs(db.SkillNJZenynage), effectThrowItem10)
	expectEffectIDs(t, "NJ_TATAMIGAESHI imported ground", skillGroundEffectIDs(db.SkillNJTatamigaeshi), effectTatami)
	expectEffectIDs(t, "NJ_KASUMIKIRI imported", skillEffectIDs(db.SkillNJKasumikiri), effectKasumikiri)
	expectEffectIDs(t, "NJ_KIRIKAGE imported", skillEffectIDs(db.SkillNJKirikage), effectKirikage)
	expectEffectIDs(t, "NJ_KOUENKA imported", skillEffectIDs(db.SkillNJKouenka), effectKouenka)
	expectEffectIDs(t, "NJ_KAENSIN imported ground", skillGroundEffectIDs(db.SkillNJKaensin), effectKaen)
	expectEffectIDs(t, "NJ_BAKUENRYU imported", skillEffectIDs(db.SkillNJBakuenryu), effectBaku)
	expectEffectIDs(t, "NJ_HYOUSENSOU imported", skillEffectIDs(db.SkillNJHyousensou), effectHyousensou)
	expectEffectIDs(t, "NJ_HYOUSYOURAKU imported", skillEffectIDs(db.SkillNJHyousyouraku), effectHyousyouraku)
	expectEffectIDs(t, "NJ_HUUJIN imported", skillEffectIDs(db.SkillNJHuujin), effectStin4)
	expectEffectIDs(t, "NJ_RAIGEKISAI imported", skillEffectIDs(db.SkillNJRaigekisai), effectThunderStorm2)
	expectEffectIDs(t, "NJ_ISSEN imported", skillEffectIDs(db.SkillNJIssen), effectIssen)
	expectEffectIDs(t, "AS_VENOMKNIFE imported before hit", skillBeforeHitEffectIDs(db.SkillASVenomknife), effectThrowItem6)
	expectEffectIDs(t, "ALL_WEWISH imported", skillEffectIDs(db.SkillALLWewish), effectChristmasCarol)
	expectEffectIDs(t, "NPC_VENOMFOG imported", skillEffectIDs(db.SkillNPCVenomfog), effectVenomFog)
	expectEffectIDs(t, "RK_IGNITIONBREAK imported caster", skillEffectOnCasterIDs(db.SkillRKIgnitionbreak), effectIgnitionBreak)
	expectEffectIDs(t, "RK_DRAGONBREATH imported hit", skillHitEffectIDs(db.SkillRKDragonbreath), effectM05)
	expectEffectIDs(t, "RK_DRAGONHOWLING imported", skillEffectIDs(db.SkillRKDragonhowling), effectDragonHowling)
	expectEffectIDs(t, "RK_MILLENNIUMSHIELD imported", skillEffectIDs(db.SkillRKMillenniumshield), effectMillenniumShield)
	expectEffectIDs(t, "RK_ENCHANTBLADE imported", skillEffectIDs(db.SkillRKEnchantblade), effectBerserkPotion2)
	expectEffectIDs(t, "RK_SONICWAVE imported", skillEffectIDs(db.SkillRKSonicwave), effectHealN)
	expectEffectIDs(t, "WL_WHITEIMPRISON imported", skillEffectIDs(db.SkillWLWhiteimprison), effectBottomBasilica2)
	expectEffectIDs(t, "WL_FROSTMISTY imported", skillEffectIDs(db.SkillWLFrostmisty), effectFrostMisty)
	expectEffectIDs(t, "WL_MARSHOFABYSS imported", skillEffectIDs(db.SkillWLMarshofabyss), effectMarshOfAbyss)
	expectEffectIDs(t, "WL_RECOGNIZEDSPELL imported", skillEffectIDs(db.SkillWLRecognizedspell), effectRecognized)
	expectEffectIDs(t, "WL_STASIS imported", skillEffectIDs(db.SkillWLStasis), effectStasis)
	expectEffectIDs(t, "WL_CRIMSONROCK imported", skillEffectIDs(db.SkillWLCrimsonrock), effectCrimsonRock)
	expectEffectIDs(t, "WL_HELLINFERNO imported ground", skillGroundEffectIDs(db.SkillWLHellinferno), effectHellInferno)
	expectEffectIDs(t, "WL_CHAINLIGHTNING_ATK imported", skillEffectIDs(db.SkillWLChainlightningAtk), effectChainLightning)
	expectEffectIDs(t, "WL_EARTHSTRAIN imported ground", skillGroundEffectIDs(db.SkillWLEarthstrain), effectEarthWall)
	expectEffectIDs(t, "WL_TETRAVORTEX imported", skillEffectIDs(db.SkillWLTetravortex), effectTetra)
	expectEffectIDs(t, "WL_TETRAVORTEX imported begin", skillBeginEffectIDs(db.SkillWLTetravortex), effectTetraCasting)
	expectEffectIDs(t, "GC_ROLLINGCUTTER imported", skillEffectIDs(db.SkillGCRollingcutter), effectCastSpin2)
	expectEffectIDs(t, "AB_JUDEX imported", skillEffectIDs(db.SkillABJudex), effectFirePillarOn2)
	expectEffectIDs(t, "AB_JUDEX imported hit", skillHitEffectIDs(db.SkillABJudex), effectHolyLight)
	expectEffectIDs(t, "AB_ADORAMUS imported", skillEffectIDs(db.SkillABAdoramus), effectAdoramus)
	expectEffectIDs(t, "AB_EPICLESIS imported", skillEffectIDs(db.SkillABEpiclesis), effectGlassWall4)
	expectEffectIDs(t, "AB_EPICLESIS imported ground", skillGroundEffectIDs(db.SkillABEpiclesis), effectGlassWall3)
	expectEffectIDs(t, "RA_ARROWSTORM imported", skillEffectIDs(db.SkillRAArrowstorm), effectArrowStorm)
	expectEffectIDs(t, "RA_AIMEDBOLT imported", skillEffectIDs(db.SkillRAAimedbolt), effectAimedBolt)
	expectEffectIDs(t, "RA_AIMEDBOLT imported before hit", skillBeforeHitEffectIDs(db.SkillRAAimedbolt), effectArrowShot)
	expectEffectIDs(t, "RA_DETONATOR imported", skillEffectIDs(db.SkillRADetonator), effectConcentration2)
	expectEffectIDs(t, "NC_POWERSWING imported", skillEffectIDs(db.SkillNCPowerswing), effectCrashAxe)
	expectEffectIDs(t, "SR_EARTHSHAKER imported", skillEffectIDs(db.SkillSREarthshaker), effectElectric4)
	expectEffectIDs(t, "GC_DARKCROW imported", skillEffectIDs(db.SkillGCDarkcrow), effectGCDarkCrow)
	expectEffectIDs(t, "GN_ILLUSIONDOPING imported", skillEffectIDs(db.SkillGNIllusiondoping), effectGNIllusionDoping)
	expectEffectIDs(t, "RK_LUXANIMA imported", skillEffectIDs(db.SkillRKLuxanima), effectRKLuxAnima)
	expectEffectIDs(t, "NC_MAGMA_ERUPTION imported", skillEffectIDs(db.SkillNCMagmaEruption), effectNCMagmaEruption)
	expectEffectIDs(t, "SO_ELEMENTAL_SHIELD imported", skillEffectIDs(db.SkillSOElementalShield), effectSOElemShield)
	expectEffectIDs(t, "SR_FLASHCOMBO imported", skillEffectIDs(db.SkillSRFlashcombo), effectSRFlashCombo)
	expectEffectIDs(t, "AB_OFFERTORIUM imported", skillEffectIDs(db.SkillABOffertorium), effectABOffertorium)
	expectEffectIDs(t, "WL_TELEKINESIS_INTENSE imported", skillEffectIDs(db.SkillWLTelekinesisIntense), effectWLTelekinesis)
	expectEffectIDs(t, "ALL_FULL_THROTTLE imported", skillEffectIDs(db.SkillALLFullThrottle), effectAllFullThrottle)
	expectEffectIDs(t, "SC_BODYPAINT imported", skillEffectIDs(db.SkillSCBodypaint), effectStretch)
	expectEffectIDs(t, "SC_ENERVATION imported", skillEffectIDs(db.SkillSCEnervation), effectEnervation)
	expectEffectIDs(t, "SC_GROOMY imported", skillEffectIDs(db.SkillSCGroomy), effectEnervation2)
	expectEffectIDs(t, "SC_IGNORANCE imported", skillEffectIDs(db.SkillSCIgnorance), effectEnervation3)
	expectEffectIDs(t, "SC_LAZINESS imported", skillEffectIDs(db.SkillSCLaziness), effectEnervation4)
	expectEffectIDs(t, "SC_UNLUCKY imported", skillEffectIDs(db.SkillSCUnlucky), effectEnervation5)
	expectEffectIDs(t, "SC_WEAKNESS imported", skillEffectIDs(db.SkillSCWeakness), effectEnervation6)
	expectEffectIDs(t, "SC_MANHOLE imported ground", skillGroundEffectIDs(db.SkillSCManhole), effectBottomManhole)
	expectEffectIDs(t, "SC_MANHOLE imported success", skillSuccessEffectIDs(db.SkillSCManhole), effectManhole)
	expectEffectIDs(t, "SC_DIMENSIONDOOR imported ground", skillGroundEffectIDs(db.SkillSCDimensiondoor), effectForestLight6)
	expectEffectIDs(t, "SC_CHAOSPANIC imported ground", skillGroundEffectIDs(db.SkillSCChaospanic), effectBottomAni)
	expectEffectIDs(t, "SC_MAELSTROM imported ground", skillGroundEffectIDs(db.SkillSCMaelstrom), effectBottomMaelstrom)
	expectEffectIDs(t, "SC_BLOODYLUST imported ground", skillGroundEffectIDs(db.SkillSCBloodylust), effectBottomBloodyLust)
	expectEffectIDs(t, "LG_SHIELDPRESS imported before hit", skillBeforeHitEffectIDs(db.SkillLGShieldpress), effectPressure2)
	expectEffectIDs(t, "LG_PRESTIGE imported", skillEffectIDs(db.SkillLGPrestige), effectPrimeCharge2)
	expectEffectIDs(t, "LG_BANDING imported", skillEffectIDs(db.SkillLGBanding), effectPrimeCharge3)
	expectEffectIDs(t, "LG_INSPIRATION imported", skillEffectIDs(db.SkillLGInspiration), effectPrimeCharge4)
	expectEffectIDs(t, "SO_FIREWALK imported ground", skillGroundEffectIDs(db.SkillSOFirewalk), effectFireWall2)
	expectEffectIDs(t, "SO_ELECTRICWALK imported ground", skillGroundEffectIDs(db.SkillSOElectricwalk), effectShockwave2)
	expectEffectIDs(t, "SO_DIAMONDDUST imported", skillEffectIDs(db.SkillSODiamonddust), effectColdThrow2)
	expectEffectIDs(t, "SO_PSYCHIC_WAVE imported", skillEffectIDs(db.SkillSOPsychicWave), effectSprPlant10)
	expectEffectIDs(t, "SO_WARMER imported", skillEffectIDs(db.SkillSOWarmer), effectDemonicFire4)
	expectEffectIDs(t, "SO_VARETYR_SPEAR imported before hit", skillBeforeHitEffectIDs(db.SkillSOVaretyrSpear), effectPressure3)
	expectEffectIDs(t, "WM_REVERBERATION imported ground", skillGroundEffectIDs(db.SkillWmReverberation), effectBotReverb)
	expectEffectIDs(t, "WM_REVERBERATION_MELEE imported", skillEffectIDs(db.SkillWmReverberationMelee), effectBotReverb2)
	expectEffectIDs(t, "WM_SEVERE_RAINSTORM imported", skillEffectIDs(db.SkillWmSevereRainstorm), effectRainParticle)
	expectEffectIDs(t, "WM_POEMOFNETHERWORLD imported ground", skillGroundEffectIDs(db.SkillWmPoemofnetherworld), effectBotReverb2)
	expectEffectIDs(t, "WM_VOICEOFSIREN imported ground", skillGroundEffectIDs(db.SkillWmVoiceofsiren), effectHeartAsura)
	expectEffectIDs(t, "WM_LULLABY_DEEPSLEEP imported", skillEffectIDs(db.SkillWmLullabyDeepsleep), effectChemicalV2)
	expectEffectIDs(t, "WM_SIRCLEOFNATURE imported", skillEffectIDs(db.SkillWmSircleofnature), effectCirclePower2)
	expectEffectIDs(t, "WM_RANDOMIZESPELL imported", skillEffectIDs(db.SkillWmRandomizespell), effectSecra2)
	expectEffectIDs(t, "WM_GLOOMYDAY imported", skillEffectIDs(db.SkillWmGloomyday), effectDance1)
	expectEffectIDs(t, "WM_SONG_OF_MANA imported ground", skillGroundEffectIDs(db.SkillWmSongOfMana), effectSprPlant3)
	expectEffectIDs(t, "WM_DANCE_WITH_WUG imported ground", skillGroundEffectIDs(db.SkillWmDanceWithWug), effectSprPlant2)
	expectEffectIDs(t, "WM_SATURDAY_NIGHT_FEVER imported ground", skillGroundEffectIDs(db.SkillWmSaturdayNightFever), effectSprPlant4)
	expectEffectIDs(t, "WM_LERADS_DEW imported ground", skillGroundEffectIDs(db.SkillWmLeradsDew), effectSprPlant5)
	expectEffectIDs(t, "WM_MELODYOFSINK imported ground", skillGroundEffectIDs(db.SkillWmMelodyofsink), effectSprPlant6)
	expectEffectIDs(t, "WM_BEYOND_OF_WARCRY imported ground", skillGroundEffectIDs(db.SkillWmBeyondOfWarcry), effectSprPlant7)
	expectEffectIDs(t, "WM_UNLIMITED_HUMMING_VOICE imported ground", skillGroundEffectIDs(db.SkillWmUnlimitedHummingVoice), effectSprPlant8)
	expectEffectIDs(t, "HAMI_CASTLE imported", skillEffectIDs(db.SkillHamiCastle), effectHamiCastle)
	expectEffectIDs(t, "HAMI_DEFENCE imported", skillEffectIDs(db.SkillHamiDefence), effectHamiDefence)
	expectEffectIDs(t, "HAMI_BLOODLUST imported", skillEffectIDs(db.SkillHamiBloodlust), effectHamiBlood)
	expectEffectIDs(t, "MH_POISON_MIST imported", skillEffectIDs(db.SkillMhPoisonMist), effectPoisonMist)
	expectEffectIDs(t, "MH_ERASER_CUTTER imported", skillEffectIDs(db.SkillMhEraserCutter), effectEraserCutter)
	expectEffectIDs(t, "MH_SONIC_CRAW imported", skillEffectIDs(db.SkillMhSonicCraw), effectSonicClaw)
	expectEffectIDs(t, "MH_MIDNIGHT_FRENZY imported", skillEffectIDs(db.SkillMhMidnightFrenzy), effectMidnightFrenzy)
	expectEffectIDs(t, "MH_TINDER_BREAKER imported", skillEffectIDs(db.SkillMhTinderBreaker), effectTinderBreaker)
	expectEffectIDs(t, "MH_LAVA_SLIDE imported", skillEffectIDs(db.SkillMhLavaSlide), effectLavaSlide)
	expectEffectIDs(t, "MH_VOLCANIC_ASH imported", skillEffectIDs(db.SkillMhVolcanicAsh), effectVolcanicAsh)
	expectEffectIDs(t, "AB_CHEAL imported", skillEffectIDs(db.SkillABCheal), effectHeal2)
	expectEffectIDs(t, "AB_HIGHNESSHEAL imported", skillEffectIDs(db.SkillABHighnessheal), effectHeal4)
	expectEffectIDs(t, "AB_HIGHNESSHEAL imported hit", skillHitEffectIDs(db.SkillABHighnessheal), effectHealOffensive)
}

func TestRobrowserMiniSTREffectSpecs(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		min  string
	}{
		{"Mammonite", effectMammonite, "maemor", "memor_min"},
		{"Angelus", effectAngelus, "angelus", "jong_mini"},
		{"Cure", effectCure, "cure", "cure_min"},
		{"Gloria", effectGloria, "gloria", "gloria_min"},
		{"Magnificat", effectMagnificat, "magnificat", "magnificat_min"},
		{"Resurrection", effectResurrection, "resurrection", "resurrection_min"},
		{"Lex Aeterna", effectLexAeterna, "lexaeterna", "lexaeterna_min"},
		{"Suffragium", effectSuffragium, "suffragium", "suffragium_min"},
		{"Storm Gust", effectStormGust, "stormgust", "storm_min"},
		{"Weapon Perfection", effectWeaponPerfect, "weaponperfection", "weaponperfection_min"},
		{"Maximize Power", effectMaximizePower, "maximizepower", "maximize_min"},
		{"Kyrie Eleison", effectKyrie, "kyrie", "kyrie_min"},
		{"Christmas Carol", effectChristmasCarol, "angelus", "jong_mini"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.strMinFile != tc.min || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want STR %q min %q attached", tc.name, component, tc.file, tc.min)
		}
	}
}

func TestMVPEffectSpecMatchesRoBrowserSTR(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectMvp)
	if !ok {
		t.Fatal("MVP effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("MVP components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "mvp" || !component.attachedEntity {
		t.Fatalf("MVP component = %+v, want attached STR mvp", component)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\st_mvp.wav" {
		t.Fatalf("MVP sfx = %#v", spec.sfx)
	}
}

func TestImportedSkillActionFallback(t *testing.T) {
	archer := worldstate.Actor{Job: 3}
	if action := skillAction(db.SkillACDouble).actionFamilyForActor(archer); action != spriteActionPCAttack3 {
		t.Fatalf("AC_DOUBLE action = %d, want ATTACK3", action)
	}
	shower := skillAction(db.SkillACShower)
	if action := shower.actionFamilyForActor(archer); action != attackActionFamilyForActor(archer) {
		t.Fatalf("AC_SHOWER action = %d, want normal attack family", action)
	}
	if shower.speed != 50*time.Millisecond || shower.next == nil || shower.next.action != skillActorActionReadyFight {
		t.Fatalf("AC_SHOWER timing = %+v, want robr speed 50ms then READYFIGHT", shower)
	}
	merchant := worldstate.Actor{Job: 5}
	if action := skillAction(db.SkillMCCartrevolution).actionFamilyForActor(merchant); action != spriteActionPCAttack2 {
		t.Fatalf("MC_CARTREVOLUTION action = %d, want ATTACK2", action)
	}
	counter := skillAction(db.SkillKNAutocounter)
	if !counter.defined || counter.action != skillActorActionAttack || !counter.hasFrame || counter.frame != 0 || counter.play || counter.next != nil {
		t.Fatalf("KN_AUTOCOUNTER action = %+v, want robr attack frame 0 with play=false next=false", counter)
	}
	relax := skillAction(db.SkillLKTensionrelax)
	if !relax.defined || relax.action != skillActorActionNone {
		t.Fatalf("LK_TENSIONRELAX action = %+v, want robr false action", relax)
	}
	trap := skillAction(db.SkillHTLandmine)
	if !trap.defined || trap.action != skillActorActionPickup || !trap.play || trap.repeat || trap.next == nil || trap.next.action != skillActorActionIdle {
		t.Fatalf("HT_LANDMINE action = %+v, want robr PICKUP then IDLE", trap)
	}
	sight := skillAction(db.SkillSNSight)
	if !sight.defined || sight.action != skillActorActionSkill || !sight.play || sight.repeat || sight.next == nil || sight.next.action != skillActorActionIdle {
		t.Fatalf("SN_SIGHT action = %+v, want local skill action for robr ACTION then IDLE", sight)
	}
	for _, skillID := range []uint16{db.SkillDCWinkcharm, db.SkillBDRokisweil} {
		action := skillAction(skillID)
		if !action.defined || action.action != skillActorActionSkill || !action.play || !action.repeat || action.next != nil || !action.hasFrame || action.frame != 1 || action.length != 3 || action.speed != 250*time.Millisecond {
			t.Fatalf("dance/play action for skill %d = %+v, want robr repeating SKILL frame 1", skillID, action)
		}
	}
	sonic := skillAction(db.SkillASSonicblow)
	hits := 0
	sawReadyFight := false
	for spec := &sonic; spec != nil; spec = spec.next {
		if spec.action == skillActorActionReadyFight {
			if !spec.repeat || !spec.play || spec.next != nil {
				t.Fatalf("AS_SONICBLOW ready fight tail = %+v, want robr READYFIGHT loop", spec)
			}
			sawReadyFight = true
			break
		}
		if spec.action != skillActorActionAttack || spec.repeat || !spec.play {
			t.Fatalf("AS_SONICBLOW chain node %d = %+v, want robr ATTACK hit", hits+1, spec)
		}
		hits++
		if hits == 1 {
			if spec.speed != 0 {
				t.Fatalf("AS_SONICBLOW first hit speed = %s, want default", spec.speed)
			}
			continue
		}
		if spec.speed != 30*time.Millisecond {
			t.Fatalf("AS_SONICBLOW hit %d speed = %s, want 30ms", hits, spec.speed)
		}
	}
	if hits != 8 {
		t.Fatalf("AS_SONICBLOW hits = %d, want robr 8-hit chain", hits)
	}
	if !sawReadyFight {
		t.Fatal("AS_SONICBLOW chain missing robr READYFIGHT tail")
	}
}

func TestHunterStringKeyEffectsMatchRobrowser(t *testing.T) {
	cast, ok := worldEffectSpecForID(effectSharpShootingCast)
	if !ok || cast.duration != 10*time.Second || len(cast.components) != 1 {
		t.Fatalf("496_beforecast spec = %+v ok=%t, want one 10s CastRing component", cast, ok)
	}
	component := cast.components[0]
	if component.kind != effectComponentFUNC || component.funcName != "CastRing" || component.funcAdapter != effectFuncCastRing || component.textureName != "ring_jadu" {
		t.Fatalf("496_beforecast component identity = %+v", component)
	}
	if component.bottomSize != 0.8 || component.topSize != 2.45 || component.height != 2.8 || component.posZ != 0.08 {
		t.Fatalf("496_beforecast component shape = %+v", component)
	}
	if component.totalCircleSides != 20 || component.circleSides != 20 || component.alphaMax != 0.9 || !component.fade || !component.rotate || !component.attachedEntity {
		t.Fatalf("496_beforecast component flags = %+v", component)
	}
}

func TestBlacksmithStringKeyEffectsMatchRobrowser(t *testing.T) {
	adrenaline, ok := worldEffectSpecForID(effectAdrenalineCast)
	if !ok || adrenaline.duration != 500*time.Millisecond || len(adrenaline.components) != 0 || len(adrenaline.sfx) != 1 || adrenaline.sfx[0] != "effect\\black_adrenalinerush_a.wav" {
		t.Fatalf("98_beforecast spec = %+v ok=%t, want robr adrenaline pre-cast sound", adrenaline, ok)
	}

	maximize, ok := worldEffectSpecForID(effectMaximizeSounds)
	if !ok || maximize.duration != 950*time.Millisecond || len(maximize.components) != 0 {
		t.Fatalf("maximize_power_sounds spec = %+v ok=%t, want robr delayed sound row", maximize, ok)
	}
	wantSFX := []string{
		"effect\\black_maximize_power_circle.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword_bic.wav",
	}
	wantDelays := []time.Duration{time.Millisecond, 550 * time.Millisecond, 700 * time.Millisecond, 950 * time.Millisecond}
	if !reflect.DeepEqual(maximize.sfx, wantSFX) || !reflect.DeepEqual(maximize.sfxDelays, wantDelays) {
		t.Fatalf("maximize_power_sounds sfx = %#v delays %#v", maximize.sfx, maximize.sfxDelays)
	}

	greed, ok := worldEffectSpecForID(effectGreedSound)
	if !ok || greed.duration != 500*time.Millisecond || len(greed.components) != 0 || len(greed.sfx) != 1 || greed.sfx[0] != "effect\\ef_entry.wav" {
		t.Fatalf("ef_greed_sound spec = %+v ok=%t, want robr greed sound", greed, ok)
	}
}

func TestKnightStringKeyEffectsMatchRobrowser(t *testing.T) {
	white, ok := worldEffectSpecForID(effectWhitePulse)
	if !ok || white.duration != 500*time.Millisecond || len(white.components) != 0 || len(white.sfx) != 0 {
		t.Fatalf("white_pulse spec = %+v ok=%t, want robr no-draw 500ms row", white, ok)
	}

	projectile, ok := worldEffectSpecForID(effectSpearProjectile)
	if !ok || projectile.duration != 140*time.Millisecond || len(projectile.components) != 1 {
		t.Fatalf("ef_spear_projectile spec = %+v ok=%t, want one 140ms 3D component", projectile, ok)
	}
	component := projectile.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "창" || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.attachedEntity {
		t.Fatalf("ef_spear_projectile component flags = %+v", component)
	}
	if component.duration != 140*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || component.posZ != 1 || component.angleStart != 180 || component.angleEnd != 180 {
		t.Fatalf("ef_spear_projectile component timing/shape = %+v", component)
	}
	if component.sizeStart != 100*effectPixelRatio || component.sizeEnd != 100*effectPixelRatio {
		t.Fatalf("ef_spear_projectile size = %.3f/%.3f", component.sizeStart, component.sizeEnd)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"spear_hit_sound", effectSpearHitSound, "_hit_spear.wav"},
		{"enemy_hit_normal1", effectEnemyHitNormal1, "_enemy_hit_normal1.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 500*time.Millisecond || len(spec.components) != 0 || len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s spec = %+v ok=%t, want sound %q", tc.name, spec, ok, tc.wav)
		}
	}

	beforeCast, ok := worldEffectSpecForID(effectSpiralBeforeCast)
	if !ok || beforeCast.duration != 500*time.Millisecond || len(beforeCast.components) != 1 {
		t.Fatalf("339_beforecast spec = %+v ok=%t, want body color FUNC", beforeCast, ok)
	}
	beforeComponent := beforeCast.components[0]
	if beforeComponent.kind != effectComponentFUNC || beforeComponent.funcName != "EffectBodyColor" || beforeComponent.funcAdapter != effectFuncBodyColor || !beforeComponent.attachedEntity {
		t.Fatalf("339_beforecast component = %+v", beforeComponent)
	}

	quake, ok := worldEffectSpecForID(effectQuake)
	if !ok || quake.duration != 650*time.Millisecond || quake.cameraShake != 650*time.Millisecond || len(quake.components) != 1 {
		t.Fatalf("quake spec = %+v ok=%t, want 650ms CameraQuake", quake, ok)
	}
	quakeComponent := quake.components[0]
	if quakeComponent.kind != effectComponentFUNC || quakeComponent.funcName != "CameraQuake" || quakeComponent.duration != 650*time.Millisecond || !quakeComponent.attachedEntity {
		t.Fatalf("quake component = %+v", quakeComponent)
	}
}

func TestCrusaderStringKeyEffectsMatchRobrowser(t *testing.T) {
	projectile, ok := worldEffectSpecForID(effectShieldProjectile)
	if !ok || projectile.duration != 140*time.Millisecond || len(projectile.components) != 1 {
		t.Fatalf("ef_shield_projectile spec = %+v ok=%t, want one 140ms 3D component", projectile, ok)
	}
	component := projectile.components[0]
	if component.kind != effectComponent3D || component.textureFile != "effect/shield_boomerang.bmp" || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || !component.attachedEntity {
		t.Fatalf("ef_shield_projectile component flags = %+v", component)
	}
	if component.duration != 140*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || component.posZ != 1 || component.angleStart != 180 || component.angleEnd != 540 {
		t.Fatalf("ef_shield_projectile component timing/shape = %+v", component)
	}
	if component.sizeStart != 50*effectPixelRatio || component.sizeEnd != 50*effectPixelRatio {
		t.Fatalf("ef_shield_projectile size = %.3f/%.3f", component.sizeStart, component.sizeEnd)
	}

	gospel, ok := worldEffectSpecForID(effectGospelGround)
	if !ok || gospel.duration != 1500*time.Millisecond || len(gospel.components) != 2 {
		t.Fatalf("370_ground spec = %+v ok=%t, want two song ground components", gospel, ok)
	}
	tile, cross := gospel.components[0], gospel.components[1]
	if tile.kind != effectComponentFUNC || tile.funcName != "FlatColorTile" || tile.funcAdapter != effectFuncFlatColorTile || tile.color != (color.RGBA{R: 255, G: 255, B: 255, A: 13}) || tile.sizeStart != 1 || !tile.renderBefore || tile.attachedEntity {
		t.Fatalf("370_ground tile = %+v", tile)
	}
	if cross.kind != effectComponentFUNC || cross.funcName != "GroundTexture" || cross.funcAdapter != effectFuncGroundTexture || cross.textureFile != "effect/cross_old.bmp" || cross.duration != 1500*time.Millisecond || cross.sizeStart != 0.5 || cross.sizeEnd != 0.5 || cross.alphaMax != 0.7 || cross.posZ != 0.4 || !cross.blendAdditive || !cross.renderBefore || cross.attachedEntity {
		t.Fatalf("370_ground cross = %+v", cross)
	}
}

func TestArcherThiefMerchantSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AC_CONCENTRATION", skillEffectIDs(45), effectConcentration)
	expectEffectIDs(t, "AC_DOUBLE begin", skillBeginEffectIDs(46), effectBashBegin)
	expectEffectIDs(t, "AC_DOUBLE before-hit", skillBeforeHitEffectIDs(46), effectArrowShot)
	expectEffectIDs(t, "AC_DOUBLE hit", skillHitEffectIDs(46), effectBashHit)
	expectEffectIDs(t, "AC_SHOWER", skillEffectIDs(47), effectArrowShower)
	expectEffectIDs(t, "AC_SHOWER hit", skillHitEffectIDs(47), effectBashHit)
	expectEffectIDs(t, "AC_CHARGEARROW before-hit", skillBeforeHitEffectIDs(148), effectArrowShot)
	if !skillHidesCastAura(db.SkillACChargearrow) {
		t.Fatal("AC_CHARGEARROW should hide cast aura like roBrowser")
	}
	expectEffectIDs(t, "TF_DOUBLE passive", skillEffectIDs(48))
	expectEffectIDs(t, "TF_MISS passive", skillEffectIDs(49))
	expectEffectIDs(t, "TF_STEAL success", skillSuccessEffectIDs(50), effectSteal)
	expectEffectIDs(t, "TF_HIDING", skillEffectIDs(51))
	expectEffectIDs(t, "TF_POISON hit", skillHitEffectIDs(52), effectPoisonAttack)
	expectEffectIDs(t, "TF_DETOXIFY", skillEffectIDs(53), effectDetoxication)
	expectEffectIDs(t, "TF_SPRINKLESAND", skillEffectIDs(149), effectSprinkleSand)
	expectEffectIDs(t, "TF_BACKSLIDING", skillEffectIDs(150))
	expectEffectIDs(t, "TF_PICKSTONE", skillEffectIDs(151))
	if !skillHidesCastAura(db.SkillTFPickstone) {
		t.Fatal("TF_PICKSTONE should hide cast aura like roBrowser")
	}
	expectEffectIDs(t, "TF_THROWSTONE before-hit", skillBeforeHitEffectIDs(152), effectThrowItem3)
	expectEffectIDs(t, "MC_MAMMONITE", skillEffectIDs(42), effectMammonite)
	expectEffectIDs(t, "MC_CARTREVOLUTION begin", skillBeginEffectIDs(153), effectCartRevolution)
	expectEffectIDs(t, "MC_CARTREVOLUTION hit", skillHitEffectIDs(153), effectCartRevolution)
	expectEffectIDs(t, "MC_LOUD", skillEffectIDs(155), effectLoud)
	expectEffectIDs(t, "AL_HOLYLIGHT", skillEffectIDs(156), effectHolyLight)
}

func TestThiefThrowStoneEffectFollowsRoBrowserTable(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectThrowItem3)
	if !ok || len(spec.components) != 1 {
		t.Fatalf("throw stone spec = %#v ok=%t, want one component", spec, ok)
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "유저인터페이스/item/돌.bmp" {
		t.Fatalf("throw stone component = %#v, want stone texture 3D component", component)
	}
	if !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || component.posZ != 1 {
		t.Fatalf("throw stone trajectory flags = %#v", component)
	}
}

func TestArcherProjectileEffectsFollowRoBrowserTable(t *testing.T) {
	shot, ok := worldEffectSpecForID(effectArrowShot)
	if !ok || len(shot.components) != 1 {
		t.Fatalf("arrow shot spec = %#v ok=%t, want one component", shot, ok)
	}
	component := shot.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "data/sprite/npc/skel_archer_arrow" {
		t.Fatalf("arrow shot component = %#v, want skel_archer_arrow 3D sprite", component)
	}
	if !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || component.duration != 140*time.Millisecond {
		t.Fatalf("arrow shot robr flags = %#v", component)
	}

	shower, ok := worldEffectSpecForID(effectArrowShower)
	if !ok || len(shower.components) != 1 {
		t.Fatalf("arrow shower spec = %#v ok=%t, want one component", shower, ok)
	}
	component = shower.components[0]
	if component.duplicate != 10 || component.posXEndRand != 1.5 || component.posYEndRand != 1.5 {
		t.Fatalf("arrow shower scatter = %#v, want robr duplicate/scatter values", component)
	}
}

func TestBowNormalAttackAddsArrowProjectileEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 15, Y: 20, Job: 1002, ObjectType: actorObjectTypeMob, HasObjectType: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 200,
			Selected:  session.Character{ID: 200, Job: 3, Weapon: 11},
		},
		World: world,
	}
	mode := &WorldMode{}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    200,
		TargetID:    300,
		Damage:      12,
		HitCount:    1,
		Action:      0,
		SourceSpeed: 500,
		TargetSpeed: 500,
	})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want arrow projectile and regular hit", len(mode.worldEffects))
	}
	projectile := mode.worldEffects[0]
	if projectile.effectID != effectArrowShot || projectile.actorID != 300 || projectile.targetID != 200 {
		t.Fatalf("normal bow projectile = %+v", projectile)
	}
	hit := mode.worldEffects[1]
	if hit.effectID != effectHit1 || hit.actorID != 300 || !hit.starts.After(projectile.starts) {
		t.Fatalf("normal bow hit = %+v projectile=%+v", hit, projectile)
	}
}

func TestWarpEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AL_TELEPORT begin", skillBeginEffectIDs(26))
	expectEffectIDs(t, "Butterfly Wing item", itemUseEffectIDs(602), effectTeleportation)
	expectEffectIDs(t, "Fly Wing item", itemUseEffectIDs(601))
}

func TestSpeedPotionItemEffectMappingsMatchRobrowser(t *testing.T) {
	expectEffectIDs(t, "Concentration Potion item", itemUseEffectIDs(645), effectItemFast)
	expectEffectIDs(t, "Awakening Potion item", itemUseEffectIDs(656), effectItemFast2)
	expectEffectIDs(t, "Berserk Potion item", itemUseEffectIDs(657), effectItemFast3)
}

func TestTeleportModalRules(t *testing.T) {
	lv1 := session.Skill{ID: 26, Level: 1, Type: skillTargetEnemy, Range: 9}
	lv2 := session.Skill{ID: 26, Level: 2, Type: skillTargetEnemy, Range: 9}
	if !gameui.TeleportWarpListBypassesModal(lv1, network.WarpPointList{SkillID: 26, MapNames: []string{"Random"}}) {
		t.Fatal("Teleport level 1 should bypass the modal")
	}
	if gameui.TeleportWarpListBypassesModal(lv2, network.WarpPointList{SkillID: 26, MapNames: []string{"Random", "prontera"}}) {
		t.Fatal("Teleport level 2 with a save point should show the modal")
	}
}

func TestWarpPortalListOpensDestinationModal(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{Skills: session.Skills{List: []session.Skill{
		{ID: 27, Level: 4, Type: skillTargetPlace, Range: 9},
	}}}}
	mode.applyWarpPointList(ctx, network.WarpPointList{SkillID: 27, MapNames: []string{"prontera", "geffen", "payon"}})
	if !mode.ui.teleportModal.IsOpen() {
		t.Fatal("warp portal list should open the destination modal")
	}
	if mode.ui.teleportModal.Title() != "Warp Portal" {
		t.Fatalf("modal title = %q", mode.ui.teleportModal.Title())
	}
}

func TestSkillUnitEffectMappings(t *testing.T) {
	expectEffectIDs(t, "UNT_SAFETYWALL", skillUnitEffectIDs(126), effectSafetyWall)
	expectEffectIDs(t, "UNT_FIREWALL", skillUnitEffectIDs(127), effectFireWall)
	expectEffectIDs(t, "UNT_WARPPORTAL", skillUnitEffectIDs(128), effectPortal)
	expectEffectIDs(t, "rAthena UNT_WARP_ACTIVE", skillUnitEffectIDs(129), effectPortal)
	expectEffectIDs(t, "UNT_PNEUMA", skillUnitEffectIDs(133), effectPneuma)
	expectEffectIDs(t, "UNT_LULLABY", skillUnitEffectIDs(158), effectBottomLullaby)
	expectEffectIDs(t, "UNT_RICHMANKIM", skillUnitEffectIDs(159), effectBottomRichKim)
	expectEffectIDs(t, "UNT_ETERNALCHAOS", skillUnitEffectIDs(160), effectBottomChaos)
	expectEffectIDs(t, "UNT_DRUMBATTLEFIELD", skillUnitEffectIDs(161), effectBottomDrum)
	expectEffectIDs(t, "UNT_RINGNIBELUNGEN", skillUnitEffectIDs(162), effectBottomNibelung)
	expectEffectIDs(t, "UNT_ROKISWEIL", skillUnitEffectIDs(163), effectBottomRoki)
	expectEffectIDs(t, "UNT_INTOABYSS", skillUnitEffectIDs(164), effectBottomAbyss)
	expectEffectIDs(t, "UNT_SIEGFRIED", skillUnitEffectIDs(165), effectBottomSieg)
	expectEffectIDs(t, "UNT_DISSONANCE", skillUnitEffectIDs(166), effectBottomDissonance)
	expectEffectIDs(t, "UNT_WHISTLE", skillUnitEffectIDs(167), effectBottomWhistle)
	expectEffectIDs(t, "UNT_ASSASSINCROSS", skillUnitEffectIDs(168), effectBottomSinX)
	expectEffectIDs(t, "UNT_POEMBRAGI", skillUnitEffectIDs(169), effectBottomBragi)
	expectEffectIDs(t, "UNT_APPLEIDUN", skillUnitEffectIDs(170), effectBottomApple)
	expectEffectIDs(t, "UNT_UGLYDANCE", skillUnitEffectIDs(171), effectBottomUglyDance)
	expectEffectIDs(t, "UNT_HUMMING", skillUnitEffectIDs(172), effectBottomHumming)
	expectEffectIDs(t, "UNT_DONTFORGETME", skillUnitEffectIDs(173), effectBottomForget)
	expectEffectIDs(t, "UNT_FORTUNEKISS", skillUnitEffectIDs(174), effectBottomFortune)
	expectEffectIDs(t, "UNT_SERVICEFORYOU", skillUnitEffectIDs(175), effectBottomService)
	expectEffectIDs(t, "UNT_MOONLIT", skillUnitEffectIDs(181), effectMoonlit)
	expectEffectIDs(t, "UNT_FOGWALL", skillUnitEffectIDs(182), effectFogWallGround)
	expectEffectIDs(t, "UNT_GRAVITATION", skillUnitEffectIDs(184), effectGravitation)
	expectEffectIDs(t, "UNT_EVILLAND", skillUnitEffectIDs(199), effectBottomEvilLand)
	expectEffectIDs(t, "UNT_EPICLESIS", skillUnitEffectIDs(202), effectGlassWall3)
	expectEffectIDs(t, "UNT_EARTHSTRAIN", skillUnitEffectIDs(203), effectEarthWall)
	expectEffectIDs(t, "UNT_MANHOLE", skillUnitEffectIDs(204), effectBottomManhole)
	expectEffectIDs(t, "UNT_DIMENSIONDOOR", skillUnitEffectIDs(205), effectForestLight6)
	expectEffectIDs(t, "UNT_CHAOSPANIC", skillUnitEffectIDs(206), effectBottomAni)
	expectEffectIDs(t, "UNT_MAELSTROM", skillUnitEffectIDs(207), effectBottomMaelstrom)
	expectEffectIDs(t, "UNT_BLOODYLUST", skillUnitEffectIDs(208), effectBottomBloodyLust)
	expectEffectIDs(t, "UNT_REVERBERATION", skillUnitEffectIDs(218), effectBotReverb)
	expectEffectIDs(t, "UNT_FIREWALK", skillUnitEffectIDs(220), effectFireWall2)
	expectEffectIDs(t, "UNT_ELECTRICWALK", skillUnitEffectIDs(221), effectShockwave2)
	expectEffectIDs(t, "UNT_NETHERWORLD", skillUnitEffectIDs(222), effectBotReverb2)
}

func TestMagnumBreakEffectSpecUsesWorldCylinders(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectMagnumBreak)
	if !ok {
		t.Fatal("magnum break effect spec missing")
	}
	if spec.cameraShake != 0 {
		t.Fatalf("camera shake = %s, want none; quake_magnum carries shake", spec.cameraShake)
	}
	if len(spec.components) != 2 {
		t.Fatalf("components = %d, want 2", len(spec.components))
	}
	for i, component := range spec.components {
		if component.kind != effectComponentCylinder {
			t.Fatalf("component %d kind = %d, want cylinder", i, component.kind)
		}
		if component.fixedPerspective {
			t.Fatalf("component %d is fixed perspective, want world-space cylinder", i)
		}
		if component.animation != 4 || component.height <= 0 {
			t.Fatalf("component %d = %+v, want animation 4 with height", i, component)
		}
	}
}

func TestQuakeMagnumEffectStartsCameraShake(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectQuakeMagnum)
	if !ok {
		t.Fatal("quake magnum effect spec missing")
	}
	if spec.cameraShake != 50*time.Millisecond || len(spec.components) != 0 {
		t.Fatalf("quake magnum spec = %+v, want no-draw 50ms camera shake", spec)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	starts := time.Unix(100, 0)
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	if !mode.addWorldEffectAt(ctx, effectQuakeMagnum, 2000000, starts) {
		t.Fatal("add quake magnum effect failed")
	}
	if !mode.cameraShakeStart.Equal(starts) || !mode.cameraShakeEnd.Equal(starts.Add(50*time.Millisecond)) {
		t.Fatalf("camera shake = %s..%s", mode.cameraShakeStart, mode.cameraShakeEnd)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(10 * time.Millisecond)); x == 0 && y == 0 {
		t.Fatalf("camera shake offset = %.3f, %.3f, want non-zero", x, y)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(60 * time.Millisecond)); x != 0 || y != 0 {
		t.Fatalf("expired camera shake offset = %.3f, %.3f, want zero", x, y)
	}
}

func TestSpeedPotionEffectSpecsMatchRobrowserSTRRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		effectID int
		file     string
	}{
		{"Concentration Potion", effectItemFast, "집중"},
		{"Awakening Potion", effectItemFast2, "각성"},
		{"Berserk Potion", effectItemFast3, "버서크"},
	} {
		spec, ok := worldEffectSpecForID(tc.effectID)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ac_concentration.wav" {
			t.Fatalf("%s sfx = %v, want ac_concentration", tc.name, spec.sfx)
		}
	}
}

func TestBerserkPotionEffectStartsDelayedCameraShake(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectItemFast3)
	if !ok {
		t.Fatal("berserk potion effect spec missing")
	}
	if spec.cameraShake != 200*time.Millisecond || spec.cameraShakeDelay != 200*time.Millisecond {
		t.Fatalf("berserk potion shake = delay %s duration %s, want 200ms/200ms", spec.cameraShakeDelay, spec.cameraShake)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	starts := time.Unix(100, 0)
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	if !mode.addWorldEffectAt(ctx, effectItemFast3, 2000000, starts) {
		t.Fatal("add berserk potion effect failed")
	}
	shakeStart := starts.Add(200 * time.Millisecond)
	shakeEnd := starts.Add(400 * time.Millisecond)
	if !mode.cameraShakeStart.Equal(shakeStart) || !mode.cameraShakeEnd.Equal(shakeEnd) {
		t.Fatalf("camera shake = %s..%s, want %s..%s", mode.cameraShakeStart, mode.cameraShakeEnd, shakeStart, shakeEnd)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(100 * time.Millisecond)); x != 0 || y != 0 {
		t.Fatalf("early camera shake offset = %.3f, %.3f, want zero", x, y)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(250 * time.Millisecond)); x == 0 && y == 0 {
		t.Fatalf("active camera shake offset = %.3f, %.3f, want non-zero", x, y)
	}
}

func TestEndureEffectSpecMatchesRobrowser3DTexture(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectEndure)
	if !ok {
		t.Fatal("endure effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "effect/endure.tga" || component.duration != time.Second {
		t.Fatalf("component = %+v", component)
	}
	if !component.fadeIn || !component.fadeOut || !component.sizeSmooth {
		t.Fatalf("component fade/size flags = %+v", component)
	}
	if component.posZ != 2 || component.sizeStart != 200*effectPixelRatio || component.sizeEnd != 70*effectPixelRatio {
		t.Fatalf("component position/size = %+v", component)
	}
}

func TestTeleportationEffectSpecUsesRobrowserCylinderStack(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectTeleportation)
	if !ok {
		t.Fatal("teleportation effect spec missing")
	}
	if spec.duration != 1500*time.Millisecond {
		t.Fatalf("duration = %s, want 1500ms", spec.duration)
	}
	if !spec.detachLocalActor {
		t.Fatal("teleportation should detach from the local actor")
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	expected := []struct {
		bottom float64
		top    float64
		height float64
	}{
		{0.3, 0.3, 35},
		{0.6, 0.8, 25},
		{0.8, 1.0, 13},
		{1.0, 1.3, 5},
	}
	for i, want := range expected {
		component := spec.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond || component.blendMode != 2 || !component.blendAdditive {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height {
			t.Fatalf("component %d size = %.1f %.1f %.1f, want %.1f %.1f %.1f", i, component.bottomSize, component.topSize, component.height, want.bottom, want.top, want.height)
		}
		if component.fixedPerspective {
			t.Fatalf("component %d uses fixed perspective, want world-space cylinder", i)
		}
		if !component.attachedEntity {
			t.Fatalf("component %d is not attached to the entity", i)
		}
	}
}

func TestWarpPortalEffectSpecUsesPortal2Cylinders(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectPortal)
	if !ok {
		t.Fatal("portal effect spec missing")
	}
	if len(spec.sfx) != 2 || spec.sfx[0] != "effect\\ef_readyportal.wav" || spec.sfx[1] != "effect\\ef_portal.wav" {
		t.Fatalf("portal sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	first := spec.components[0]
	if first.kind != effectComponentCylinder || first.textureName != "ring_blue" || first.duration != 500*time.Millisecond || first.animation != 4 {
		t.Fatalf("first portal component = %+v", first)
	}
	if !first.repeat || first.repeatDelay != -300*time.Millisecond {
		t.Fatalf("first portal repeat = %t delay=%s, want reference client repeat -300ms", first.repeat, first.repeatDelay)
	}
	if spec.components[3].textureName != "alpha1" || spec.components[3].posZ != 2 || spec.components[3].height != 1 {
		t.Fatalf("portal cap component = %+v", spec.components[3])
	}
}

func TestHealEffectSpecUsesRobrowserCylindersAndParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectHeal)
	if !ok {
		t.Fatal("heal effect spec missing")
	}
	if spec.duration != 1840*time.Millisecond {
		t.Fatalf("duration = %s, want 1840ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "_heal_effect.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	for i, component := range spec.components[:2] {
		if component.kind != effectComponentCylinder || component.textureName != "ring_white" || component.animation != 1 {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.duration != 1500*time.Millisecond || component.height != 8 || component.alphaMax != 0.2 {
			t.Fatalf("component %d timing/shape = %+v", i, component)
		}
		if component.color != (color.RGBA{R: 178, G: 255, B: 178, A: 255}) || !component.blendAdditive {
			t.Fatalf("component %d tint/blend = %+v", i, component)
		}
	}
	firstParticle := spec.components[2]
	if firstParticle.kind != effectComponent3D || firstParticle.textureFile != "effect/pok3.tga" {
		t.Fatalf("first heal particle = %+v", firstParticle)
	}
	if firstParticle.duration != 1300*time.Millisecond || firstParticle.delay != 400*time.Millisecond || firstParticle.duplicateDelay != 10*time.Millisecond || firstParticle.duplicate != 15 {
		t.Fatalf("first heal particle timing = %+v", firstParticle)
	}
	if firstParticle.alphaMax != 0.6 || !firstParticle.fadeIn || !firstParticle.fadeOut || firstParticle.sparkling || firstParticle.sparkNumber != 0 {
		t.Fatalf("first heal particle fade = %+v", firstParticle)
	}
	if firstParticle.posXRand != 1.5 || firstParticle.posYRand != 1.5 || firstParticle.posZEndRand != 2 || firstParticle.posZEndMiddle != 6 {
		t.Fatalf("first heal particle position = %+v", firstParticle)
	}
	if firstParticle.sizeStart != 9*effectPixelRatio || firstParticle.sizeEnd != 9*effectPixelRatio || firstParticle.sizeRand != 2*effectPixelRatio {
		t.Fatalf("first heal particle size = %+v", firstParticle)
	}
	secondParticle := spec.components[3]
	if secondParticle.kind != effectComponent3D || secondParticle.textureFile != "effect/pok3.tga" {
		t.Fatalf("second heal particle = %+v", secondParticle)
	}
	if secondParticle.duration != 1100*time.Millisecond || secondParticle.delay != 200*time.Millisecond || secondParticle.duplicateDelay != 50*time.Millisecond || secondParticle.duplicate != 7 {
		t.Fatalf("second heal particle timing = %+v", secondParticle)
	}
	if secondParticle.alphaMax != 0.6 || !secondParticle.fadeIn || !secondParticle.fadeOut || secondParticle.sparkling || secondParticle.sparkNumber != 0 {
		t.Fatalf("second heal particle fade = %+v", secondParticle)
	}
	if secondParticle.posXRand != 1 || secondParticle.posYRand != 1 || secondParticle.posZEnd != 5 || secondParticle.posZStartRand != 1 {
		t.Fatalf("second heal particle position = %+v", secondParticle)
	}
	if secondParticle.sizeStart != 9*effectPixelRatio || secondParticle.sizeEnd != 9*effectPixelRatio || secondParticle.sizeRand != 2*effectPixelRatio {
		t.Fatalf("second heal particle size = %+v", secondParticle)
	}
}

func TestHealOffensiveEffectSpecUsesRobrowserCylindersAndParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectHealOffensive)
	if !ok {
		t.Fatal("offensive heal effect spec missing")
	}
	if spec.duration != 1490*time.Millisecond {
		t.Fatalf("duration = %s, want 1490ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "_heal_effect.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	for i, component := range spec.components[:2] {
		if component.kind != effectComponentCylinder || component.textureName != "ring_white" || component.animation != 1 {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.duration != time.Second || !component.blendAdditive || component.color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
			t.Fatalf("component %d timing/tint = %+v", i, component)
		}
	}
	if spec.components[0].height != 10 || spec.components[1].height != 9 {
		t.Fatalf("cylinder heights = %.1f %.1f", spec.components[0].height, spec.components[1].height)
	}
	firstParticle := spec.components[2]
	if firstParticle.kind != effectComponent3D || firstParticle.textureFile != "effect/pok3.tga" {
		t.Fatalf("first offensive heal particle = %+v", firstParticle)
	}
	if firstParticle.duration != time.Second || firstParticle.delay != 400*time.Millisecond || firstParticle.duplicateDelay != 10*time.Millisecond || firstParticle.duplicate != 10 {
		t.Fatalf("first offensive heal particle timing = %+v", firstParticle)
	}
	if firstParticle.alphaMax != 0.8 || !firstParticle.fadeIn || !firstParticle.fadeOut || !firstParticle.blendAdditive || !firstParticle.sparkling || firstParticle.sparkNumber != 2 {
		t.Fatalf("first offensive heal particle fade/blend = %+v", firstParticle)
	}
	if firstParticle.posXRand != 1.5 || firstParticle.posYRand != 1.5 || firstParticle.posZEndRand != 3 || firstParticle.posZEndMiddle != 6 {
		t.Fatalf("first offensive heal particle position = %+v", firstParticle)
	}
	secondParticle := spec.components[3]
	if secondParticle.kind != effectComponent3D || secondParticle.textureFile != "effect/pok3.tga" {
		t.Fatalf("second offensive heal particle = %+v", secondParticle)
	}
	if secondParticle.duration != 900*time.Millisecond || secondParticle.delay != 200*time.Millisecond || secondParticle.duplicateDelay != 50*time.Millisecond || secondParticle.duplicate != 5 {
		t.Fatalf("second offensive heal particle timing = %+v", secondParticle)
	}
	if secondParticle.alphaMax != 0.8 || !secondParticle.fadeIn || !secondParticle.fadeOut || !secondParticle.blendAdditive || !secondParticle.sparkling || secondParticle.sparkNumber != 2 {
		t.Fatalf("second offensive heal particle fade/blend = %+v", secondParticle)
	}
	if secondParticle.posXRand != 1 || secondParticle.posYRand != 1 || secondParticle.posZEnd != 6 || secondParticle.posZStartRand != 1 {
		t.Fatalf("second offensive heal particle position = %+v", secondParticle)
	}
	if secondParticle.sizeStart != 9*effectPixelRatio || secondParticle.sizeEnd != 9*effectPixelRatio || secondParticle.sizeRand != 2*effectPixelRatio {
		t.Fatalf("second offensive heal particle size = %+v", secondParticle)
	}
}

func TestIncreaseAgilityEffectSpecUsesRobrowserParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectIncAgility)
	if !ok {
		t.Fatal("increase agility effect spec missing")
	}
	if spec.duration != 1500*time.Millisecond {
		t.Fatalf("duration = %s, want 1500ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_incagility.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	particleCases := []struct {
		index     int
		alphaMax  float64
		delay     time.Duration
		duplicate int
	}{
		{index: 0, alphaMax: 1, delay: 500 * time.Millisecond, duplicate: 7},
		{index: 1, alphaMax: 0.75, delay: 400 * time.Millisecond, duplicate: 3},
		{index: 2, alphaMax: 1, delay: 0, duplicate: 10},
	}
	for _, tc := range particleCases {
		component := spec.components[tc.index]
		if component.kind != effectComponent3D || component.textureFile != "effect/ac_center2.tga" {
			t.Fatalf("particle %d resource = %+v", tc.index, component)
		}
		if component.duration != 1000*time.Millisecond || component.delay != tc.delay || component.duplicateDelay != 200*time.Millisecond || component.duplicate != tc.duplicate {
			t.Fatalf("particle %d timing = %+v", tc.index, component)
		}
		if component.alphaMax != tc.alphaMax || component.fadeIn || !component.fadeOut {
			t.Fatalf("particle %d fade = %+v", tc.index, component)
		}
		if component.posXRand != 1.5 || component.posYRand != 1 || component.posZStartRand != 1 || component.posZStartMiddle != 1 || component.posZEndRand != 1 || component.posZEndMiddle != 6 {
			t.Fatalf("particle %d position = %+v", tc.index, component)
		}
		if component.sizeStartX != 2.5*effectPixelRatio || component.sizeEndX != 2.5*effectPixelRatio {
			t.Fatalf("particle %d x size = %+v", tc.index, component)
		}
		if component.sizeStartY != 0 || component.sizeEndY != 0 || component.sizeRandY != 15*effectPixelRatio || component.sizeRandYMiddle != 45*effectPixelRatio {
			t.Fatalf("particle %d size = %+v", tc.index, component)
		}
		if component.blendAdditive {
			t.Fatalf("particle %d should use normal alpha blending", tc.index)
		}
	}
	overlay := spec.components[3]
	if overlay.kind != effectComponent3D || overlay.textureFile != "effect/agi_up.bmp" {
		t.Fatalf("overlay resource = %+v", overlay)
	}
	if overlay.duration != 1000*time.Millisecond || overlay.alphaMax != 1 || !overlay.fadeIn || !overlay.fadeOut {
		t.Fatalf("overlay timing/fade = %+v", overlay)
	}
	if overlay.posZ != 0.4 || overlay.posZEnd != 3 {
		t.Fatalf("overlay position = %+v", overlay)
	}
	if overlay.sizeStart != 100*effectPixelRatio || overlay.sizeEnd != 100*effectPixelRatio || overlay.sizeStartY != 45*effectPixelRatio || overlay.sizeEndY != 45*effectPixelRatio || !overlay.sizeSmooth {
		t.Fatalf("overlay size = %+v", overlay)
	}
	if !overlay.overlay {
		t.Fatal("overlay should use reference client overlay rendering")
	}
	if overlay.blendAdditive {
		t.Fatal("overlay should use normal alpha blending")
	}
}

func TestDecreaseAgilityEffectSpecUsesRobrowserParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectDecAgility)
	if !ok {
		t.Fatal("decrease agility effect spec missing")
	}
	if spec.duration != 1000*time.Millisecond {
		t.Fatalf("duration = %s, want 1000ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_decagility.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 2 {
		t.Fatalf("components = %d, want 2", len(spec.components))
	}
	particle := spec.components[0]
	if particle.kind != effectComponent3D || particle.textureFile != "effect/ac_center2.tga" {
		t.Fatalf("particle resource = %+v", particle)
	}
	if particle.duration != 1000*time.Millisecond || particle.duplicateDelay != 200*time.Millisecond || particle.duplicate != 20 {
		t.Fatalf("particle timing = %+v", particle)
	}
	if particle.alphaMax != 1 || particle.fadeIn || !particle.fadeOut {
		t.Fatalf("particle fade = %+v", particle)
	}
	if particle.posXRand != 1.5 || particle.posYRand != 1 || particle.posZStartRand != 1 || particle.posZStartMiddle != 6 || particle.posZEndRand != 1 || particle.posZEndMiddle != 1 {
		t.Fatalf("particle position = %+v", particle)
	}
	if particle.sizeStartX != effectTableSize(2.5) || particle.sizeEndX != effectTableSize(2.5) {
		t.Fatalf("particle x size = %+v", particle)
	}
	if particle.sizeStartY != 0 || particle.sizeEndY != 0 || particle.sizeRandY != effectTableSize(15) || particle.sizeRandYMiddle != effectTableSize(45) {
		t.Fatalf("particle size = %+v", particle)
	}
	if particle.blendAdditive {
		t.Fatal("particle should use normal alpha blending")
	}
	overlay := spec.components[1]
	if overlay.kind != effectComponent3D || overlay.textureFile != "effect/slow.bmp" {
		t.Fatalf("overlay resource = %+v", overlay)
	}
	if overlay.duration != 1000*time.Millisecond || overlay.alphaMax != 1 || !overlay.fadeIn || !overlay.fadeOut {
		t.Fatalf("overlay timing/fade = %+v", overlay)
	}
	if overlay.posZ != 2.8 || overlay.posZEnd != 0.4 {
		t.Fatalf("overlay position = %+v", overlay)
	}
	if overlay.sizeStart != effectTableSize(100) || overlay.sizeEnd != effectTableSize(100) || overlay.sizeStartY != effectTableSize(45) || overlay.sizeEndY != effectTableSize(45) || !overlay.sizeSmooth {
		t.Fatalf("overlay size = %+v", overlay)
	}
	if overlay.overlay {
		t.Fatal("overlay should use regular reference client 3D rendering")
	}
	if overlay.blendAdditive {
		t.Fatal("overlay should use normal alpha blending")
	}
}

func TestAngelusEffectSpecUsesRobrowserSTR(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectAngelus)
	if !ok {
		t.Fatal("angelus effect spec missing")
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_angelus.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "angelus" || component.strMinFile != "jong_mini" {
		t.Fatalf("STR resource = %+v", component)
	}
	if !component.attachedEntity || !component.spriteHead {
		t.Fatalf("STR attachment flags = %+v", component)
	}
}

func TestBlessingEffectSpecUsesRobrowserSpritesAndParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBlessing)
	if !ok {
		t.Fatal("blessing effect spec missing")
	}
	if spec.duration != 2500*time.Millisecond {
		t.Fatalf("duration = %s, want 2500ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_blessing.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	sprite := spec.components[0]
	if sprite.kind != effectComponentSPR || sprite.spriteFile != "축복" {
		t.Fatalf("sprite component = %+v", sprite)
	}
	if sprite.duration != 1500*time.Millisecond || sprite.spriteDelay != 30*time.Millisecond || !sprite.spriteRepeat || !sprite.spriteHead || sprite.spriteYOffset != -120 || !sprite.worldSizedSprite {
		t.Fatalf("sprite timing/placement = %+v", sprite)
	}

	particleCases := []struct {
		index       int
		delay       time.Duration
		posXRand    float64
		posYRand    float64
		sparkling   bool
		sparkNumber int
	}{
		{index: 1, delay: 300 * time.Millisecond, posXRand: 1.2, posYRand: 1, sparkling: true, sparkNumber: 2},
		{index: 2, delay: 400 * time.Millisecond, posXRand: 1.4, posYRand: 1.1},
	}
	for _, tc := range particleCases {
		component := spec.components[tc.index]
		if component.kind != effectComponent3D || component.spriteFile != "particle6" {
			t.Fatalf("particle %d resource = %+v", tc.index, component)
		}
		if component.duration != 1200*time.Millisecond || component.delay != tc.delay || component.duplicateDelay != 0 || component.duplicate != 6 {
			t.Fatalf("particle %d timing = %+v", tc.index, component)
		}
		if component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || component.sparkling != tc.sparkling || component.sparkNumber != tc.sparkNumber {
			t.Fatalf("particle %d fade/sparkle = %+v", tc.index, component)
		}
		if component.posXRand != tc.posXRand || component.posYRand != tc.posYRand || component.posZStartRand != 2 || component.posZStartMiddle != 5.5 || component.posZEndRand != 0.5 || component.posZEndMiddle != 1 {
			t.Fatalf("particle %d position = %+v", tc.index, component)
		}
		if component.sizeStart != 50*effectPixelRatio || component.sizeEnd != 50*effectPixelRatio {
			t.Fatalf("particle %d size = %+v", tc.index, component)
		}
	}

	aura := spec.components[3]
	if aura.kind != effectComponent3D || aura.textureFile != "effect/pok2.tga" {
		t.Fatalf("aura resource = %+v", aura)
	}
	if aura.duration != 2500*time.Millisecond || aura.alphaMax != 0.3 || !aura.fadeIn || !aura.fadeOut {
		t.Fatalf("aura timing/fade = %+v", aura)
	}
	if aura.color != (color.RGBA{R: 25, G: 191, B: 255, A: 255}) || !aura.blendAdditive {
		t.Fatalf("aura tint/blend = %+v", aura)
	}
	if aura.sizeStart != 140*effectPixelRatio || aura.sizeEnd != 140*effectPixelRatio {
		t.Fatalf("aura size = %+v", aura)
	}
}

func TestEmotionEffectSpecUsesEntityAttachmentOffset(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectEmotion)
	if !ok {
		t.Fatal("emotion effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSPR || component.spriteFile != "emotion" {
		t.Fatalf("emotion component = %+v", component)
	}
	if !component.attachedEntity || component.spriteYOffset != -100 || component.spriteHead {
		t.Fatalf("emotion placement = %+v", component)
	}
}

func TestWorldEffectDuplicateDeltasMatchRobrowserSemantics(t *testing.T) {
	component := worldEffectComponent{
		alphaMax:      0.2,
		alphaMaxDelta: 0.2,
		sizeStart:     100 * effectPixelRatio,
		sizeEnd:       100 * effectPixelRatio,
		sizeDelta:     -10,
	}
	if got := effectBillboardAlphaForDuplicate(0.5, component, 2); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("duplicate alpha = %.3f, want 0.6", got)
	}
	sizeX, sizeY := effect3DSize(component, worldEffect{}, 0, 0.5, 2)
	want := 80 * effectPixelRatio
	if math.Abs(sizeX-want) > 0.001 || math.Abs(sizeY-want) > 0.001 {
		t.Fatalf("duplicate size = %.3f x %.3f, want %.3f", sizeX, sizeY, want)
	}
}

func TestEffect3DSpriteScaleUsesRobrowserSpriteUnits(t *testing.T) {
	size := effectTableSize(200)
	got := effect3DSpriteScale(size)
	want := size / 100
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("sprite pixel scale = %.5f, want %.5f", got, want)
	}
	fireballWorldWidth := 64 * got
	wantWidth := 128 * effectPixelRatio
	if math.Abs(fireballWorldWidth-wantWidth) > 0.001 {
		t.Fatalf("64px fireball width = %.3f, want robr 128/35 %.3f", fireballWorldWidth, wantWidth)
	}
}

func TestEffect3DSpriteDrawOptionsHonorAdditiveBlend(t *testing.T) {
	defaultOptions := effect3DSpriteDrawOptions(worldEffectComponent{})
	if got := defaultOptions.Blend; got != render.BlendSourceOver {
		t.Fatalf("default sprite effect blend = %v, want source-over", got)
	}
	if got := defaultOptions.DepthBias; got != 0 {
		t.Fatalf("default sprite effect depth bias = %.3f, want 0", got)
	}
	if got := effect3DSpriteDrawOptions(worldEffectComponent{blendAdditive: true}).Blend; got != render.BlendLighter {
		t.Fatalf("additive sprite effect blend = %v, want lighter", got)
	}
	if got := effect3DSpriteDrawOptions(worldEffectComponent{worldSizedSprite: true}).DepthBias; got != strEffectDepthBias {
		t.Fatalf("world-sized sprite effect depth bias = %.3f, want %.3f", got, strEffectDepthBias)
	}
}

func TestWorldSizedSpriteBillboardUsesCenterDepthLikeRobrowser(t *testing.T) {
	options := render.DrawTrianglesOptions{DepthBias: 0.01}
	cmd := worldSpriteBillboardCommand(
		render.WhiteImage(),
		options,
		modelPoint3{x: 1, y: 2, z: 3},
		modelPoint3{x: 4, y: 5, z: 6},
		modelPoint3{x: 7, y: 8, z: 9},
		10,
		20,
		3,
		4,
		color.RGBA{R: 51, G: 102, B: 153, A: 204},
	)

	if cmd.UpAxis != [3]float32{7, 8, 9} {
		t.Fatalf("visible up axis = %+v, want rendered billboard up axis", cmd.UpAxis)
	}
	if cmd.DepthUpAxis != [3]float32{} {
		t.Fatalf("depth up axis = %+v, want center-depth axis", cmd.DepthUpAxis)
	}
	if cmd.DepthBias != options.DepthBias || cmd.Options.DepthBias != options.DepthBias {
		t.Fatalf("depth bias = command %.3f options %.3f, want %.3f", cmd.DepthBias, cmd.Options.DepthBias, options.DepthBias)
	}
}

func TestWorldEffectOrbitReplacesBasePositionLikeRobrowser(t *testing.T) {
	component := worldEffectComponent{
		posX:           -2,
		posY:           4,
		orbitRadiusX:   3,
		orbitRadiusY:   3,
		orbitRotations: 8,
		orbitPhase:     0.7,
		orbitClockwise: true,
	}
	x, y, _ := (&WorldMode{}).effect3DOffset(client.Context{}, component, worldEffect{}, 0, 0, 0, 0, 0, 0)
	angle := -0.7 * math.Pi / 2
	wantX := -math.Cos(angle) * 3
	wantY := math.Sin(angle) * 3
	if math.Abs(x-wantX) > 0.001 || math.Abs(y-wantY) > 0.001 {
		t.Fatalf("orbit offset = %.3f, %.3f; want %.3f, %.3f", x, y, wantX, wantY)
	}
	if math.Abs(x-(-2+wantX)) < 0.001 || math.Abs(y-(4+wantY)) < 0.001 {
		t.Fatalf("orbit offset incorrectly included base position: %.3f, %.3f", x, y)
	}
}

func TestWorldEffectBillboardSparklingAlphaMatchesRobrowser(t *testing.T) {
	component := worldEffectComponent{
		alphaMax:    1,
		sparkling:   true,
		sparkNumber: 2,
	}
	if got := effectBillboardAlphaForDuplicate(0, component, 0); math.Abs(got-1) > 0.001 {
		t.Fatalf("spark alpha at start = %.3f, want 1", got)
	}
	if got := effectBillboardAlphaForDuplicate(0.25, component, 0); math.Abs(got-0.008) > 0.001 {
		t.Fatalf("spark alpha at quarter = %.3f, want about 0.008", got)
	}
	if got := effectBillboardAlphaForDuplicate(0.75, component, 0); math.Abs(got-0.067) > 0.001 {
		t.Fatalf("spark alpha at three quarters = %.3f, want about 0.067", got)
	}
}

func TestWorldEffectBillboardAngleCanRotateWithCamera(t *testing.T) {
	projection := newSceneProjectionForTargetYaw(800, 600, 0, 0, 0, 45)
	component := worldEffectComponent{angleStart: 90, angleEnd: 180, rotateWithCamera: true}
	got := worldEffectBillboardAngle(component, projection, 0.5)
	want := degreesToRadians(180)
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("angle = %.3f, want %.3f", got, want)
	}
}

func TestEffectCylinderAngleXRotatesHeightAxisLikeRobrowser(t *testing.T) {
	got := rotateEffectCylinderVector(modelPoint3{y: 1}, -90, 0, 0)
	want := modelPoint3{z: -1}
	if !modelPointNear(got, want, 0.001) {
		t.Fatalf("rotated height axis = %+v, want %+v", got, want)
	}
}

func TestWorldCylinderBandAllowsRobrowserNegativeHeight(t *testing.T) {
	screen := render.NewFrame(320, 240)
	drawWorldCylinderBandWithBasis(screen, render.WhiteImage(), render.WhiteImage(), 0, 0, 0, 1, 2, -4, color.RGBA{A: 255}, 8, modelPoint3{x: 1}, modelPoint3{z: 1}, modelPoint3{y: 1})
	commands := reflect.ValueOf(screen).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatalf("world commands = %d, want one negative-height cylinder", commands.Len())
	}
	vertices := commands.Index(0).FieldByName("Vertices")
	if vertices.Len() < 2 {
		t.Fatalf("vertices = %d, want cylinder vertices", vertices.Len())
	}
	bottomY := vertices.Index(0).FieldByName("Y").Float()
	topY := vertices.Index(1).FieldByName("Y").Float()
	if topY >= bottomY {
		t.Fatalf("top Y = %.1f bottom Y = %.1f, want top below bottom", topY, bottomY)
	}
}

func TestSTRAnimationAttachedEntityUsesActorAnchor(t *testing.T) {
	anim := res.STRAnimation{Pos: [2]float32{320, 320}}

	_, groundY := strAnimationOffset(anim, false)
	_, attachedY := strAnimationOffset(anim, true)

	if groundY != -0.5 {
		t.Fatalf("ground STR y offset = %.3f, want -0.5", groundY)
	}
	if attachedY != 0 {
		t.Fatalf("attached STR y offset = %.3f, want 0", attachedY)
	}
}

func TestSTRAnimationLocalOffsetFlipsYBeforeRotationLikeRobrowser(t *testing.T) {
	anim := res.STRAnimation{
		Pos:   [2]float32{330, 340},
		Angle: 90,
	}
	gotX, gotY := strAnimationLocalOffset(anim, 10, 20, true)
	wantX, wantY := -10.0/35.0, -30.0/35.0
	if math.Abs(gotX-wantX) > 0.0001 || math.Abs(gotY-wantY) > 0.0001 {
		t.Fatalf("STR local offset = %.4f %.4f, want %.4f %.4f", gotX, gotY, wantX, wantY)
	}
}

func TestSTRAnimationBlendMatchesRobrowserD3DBlend(t *testing.T) {
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 7}); got != render.BlendSrcAlphaDstAlpha {
		t.Fatalf("SRC_ALPHA/DST_ALPHA blend = %v, want BlendSrcAlphaDstAlpha", got)
	}
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 2}); got != render.BlendLighter {
		t.Fatalf("SRC_ALPHA/ONE blend = %v, want BlendLighter", got)
	}
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6}); got != render.BlendSourceOver {
		t.Fatalf("regular STR blend = %v, want BlendSourceOver", got)
	}
}

func TestSTRAnimationDrawOptionsDisableFogToMatchRobrowser(t *testing.T) {
	options := strAnimationDrawOptions(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6})
	if !options.DisableFog {
		t.Fatal("STR draw options enabled map fog, want disabled")
	}
}

func TestSTRAnimationDrawOptionsUseDepthBiasToMatchRobrowser(t *testing.T) {
	options := strAnimationDrawOptions(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6})
	if options.DepthBias <= 0 {
		t.Fatalf("STR depth bias = %.3f, want positive robr-style camera bias", options.DepthBias)
	}
}

func TestSTRAnimationVertexUsesCenterDepthToMatchRobrowser(t *testing.T) {
	point := modelPoint3{x: 1, y: 2, z: 3}
	depthPoint := modelPoint3{x: 4, y: 5, z: 6}
	vertex := strAnimationVertex3D(point, texturePoint{u: 0.25, v: 0.5}, color.RGBA{R: 51, G: 102, B: 153, A: 204}, 80, 40, depthPoint)

	if vertex.X != 1 || vertex.Y != 2 || vertex.Z != 3 {
		t.Fatalf("vertex position = %.1f %.1f %.1f, want rendered point", vertex.X, vertex.Y, vertex.Z)
	}
	if vertex.DepthX != 4 || vertex.DepthY != 5 || vertex.DepthZ != 6 {
		t.Fatalf("vertex depth = %.1f %.1f %.1f, want STR center depth", vertex.DepthX, vertex.DepthY, vertex.DepthZ)
	}
	if vertex.SrcX != 20 || vertex.SrcY != 20 {
		t.Fatalf("vertex uv = %.1f %.1f, want texture pixel coords", vertex.SrcX, vertex.SrcY)
	}
}

func TestLevelUpEffectSpecsUseSTRResources(t *testing.T) {
	base, ok := worldEffectSpecForID(effectBaseLevelUp)
	if !ok {
		t.Fatal("base level-up effect spec missing")
	}
	if len(base.components) != 1 || base.components[0].kind != effectComponentSTR || base.components[0].strFile != "angel" || !base.components[0].attachedEntity {
		t.Fatalf("base level-up spec = %+v", base)
	}
	if len(base.sfx) != 1 || base.sfx[0] != "levelup.wav" {
		t.Fatalf("base level-up sfx = %#v", base.sfx)
	}
	job, ok := worldEffectSpecForID(effectJobLevelUp)
	if !ok {
		t.Fatal("job level-up effect spec missing")
	}
	if len(job.components) != 1 || job.components[0].kind != effectComponentSTR || job.components[0].strFile != "joblvup" {
		t.Fatalf("job level-up spec = %+v", job)
	}
	if len(job.sfx) != 0 {
		t.Fatalf("job level-up sfx = %#v", job.sfx)
	}
}

func TestSpecialEffectNotifyAddsLevelUpEffects(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectBaseLevelUp})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectJobLevelUp})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.worldEffects))
	}
	if mode.worldEffects[0].effectID != effectBaseLevelUp || mode.worldEffects[1].effectID != effectJobLevelUp {
		t.Fatalf("world effects = %+v", mode.worldEffects)
	}
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
	if mode.scheduledSounds[0].paths[0] != "levelup.wav" {
		t.Fatalf("base level-up sound = %+v", mode.scheduledSounds[0])
	}
}

func TestMVPNotifyAddsMVPBannerEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyMVPNotify(ctx, network.MVPNotify{AID: 2000000})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.effectID != effectMvp || effect.actorID != 2000000 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 1 || mode.scheduledSounds[0].paths[0] != "effect\\st_mvp.wav" {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
}

func TestParameterChangeLevelUpFallbackIsDeduped(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Progress:  session.Progress{BaseLevel: 10, JobLevel: 4},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseLevel, Value: 11})
	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseLevel, Value: 11})
	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobLevel, Value: 5})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.worldEffects))
	}
	if mode.worldEffects[0].effectID != effectBaseLevelUp || mode.worldEffects[1].effectID != effectJobLevelUp {
		t.Fatalf("world effects = %+v", mode.worldEffects)
	}
}

func TestSpecialEffectNotifyDedupesParameterLevelUpFallback(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Progress:  session.Progress{JobLevel: 21},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobLevel, Value: 22})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectJobLevelUp})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.effectID != effectJobLevelUp || effect.actorID != 2000000 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("scheduled sounds = %+v, want none for job level-up", mode.scheduledSounds)
	}
}

func TestWarpPortalActorEntryAddsPortalEffect(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}
	entry := network.ActorEntry{ID: 900, Job: 128, X: 30, Y: 40}

	upsertNetworkActor(ctx, entry)
	mode.applyWarpPortalEntry(ctx, entry)
	mode.applyWarpPortalEntry(ctx, entry)

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 900 || effect.effectID != effectPortal || effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect = %+v", effect)
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
	mode := &WorldMode{}
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

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000,
		Action:   network.ActionStandUp,
	})
	if world.Player.Sitting {
		t.Fatal("local player did not stand")
	}
}

func TestApplyActorActionNotifyUpdatesRemoteSitState(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, X: 10, Y: 20, Moving: true})
	mode := &WorldMode{}
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

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 300,
		Action:   network.ActionStandUp,
	})
	if actor := world.Actors[300]; actor.Sitting {
		t.Fatalf("remote actor stayed sitting: %+v", actor)
	}
}

func TestApplyItemPickupAckRemovesRequestedItemAndStartsPickupAnimation(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertItem(worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20, Amount: 1})
	mode := &WorldMode{
		pickupReqItemID: 9001,
		actorAnims:      make(map[uint32]actorAnimation),
	}
	mode.actorAnims[150000] = actorAnimation{actionFamily: spriteActionPCReadyFight, loop: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
		},
		World: world,
	}

	mode.applyItemPickupAck(ctx, network.ItemPickupAck{ItemID: 909, Amount: 1})

	if _, ok := world.Items[9001]; ok {
		t.Fatal("picked item should be removed locally after pickup ack")
	}
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
		t.Fatalf("pickup ack expired animation = %+v, want idle fallback", expired)
	}
	if mode.pickupReqItemID != 0 {
		t.Fatalf("pickup request item id = %d, want cleared", mode.pickupReqItemID)
	}
	if world.Dir != directionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
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

	if !mode.ui.deathModal.IsOpen() {
		t.Fatal("death modal should open for local death")
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

	if mode.ui.deathModal.IsOpen() {
		t.Fatal("death modal should clear when player is alive")
	}
	if _, ok := mode.actorAnims[150000]; ok {
		t.Fatal("character death animation should clear when player is alive")
	}
	if _, ok := mode.actorAnims[2000000]; ok {
		t.Fatal("account death animation should clear when player is alive")
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

func TestFollowCameraInitializesToRenderedPlayerPosition(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{
		X:            20,
		Y:            30,
		Moving:       true,
		FromX:        10,
		FromY:        20,
		ToX:          20,
		ToY:          30,
		MoveStarted:  now.Add(-750 * time.Millisecond),
		MoveDuration: 1500 * time.Millisecond,
	}
	ctx := client.Context{World: world}

	camera := followCamera{}
	camera.Update(ctx, now)

	if camera.x != 15.5 || camera.y != 25.5 {
		t.Fatalf("camera target = %.2f, %.2f, want rendered player center 15.5, 25.5", camera.x, camera.y)
	}
	if world.Camera.X != camera.x || world.Camera.Y != camera.y {
		t.Fatalf("world camera = %.2f, %.2f, want %.2f, %.2f", world.Camera.X, world.Camera.Y, camera.x, camera.y)
	}
}

func TestFollowCameraEasesTowardRenderedPlayerLikeReferenceView(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	ctx := client.Context{World: world}

	camera := followCamera{}
	now := time.Now()
	camera.Update(ctx, now)
	world.Player = worldstate.Actor{X: 14, Y: 20}
	camera.Update(ctx, now.Add(time.Second/60))

	if math.Abs(camera.x-10.9) > 0.001 || camera.y != 20.5 {
		t.Fatalf("camera target = %.3f, %.3f, want 10.900, 20.5", camera.x, camera.y)
	}
}

func TestCameraFollowLerpClampsLikeReferenceView(t *testing.T) {
	if got := cameraFollowLerp(100 * time.Millisecond); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("camera lerp = %.2f, want 0.60", got)
	}
	if got := cameraFollowLerp(time.Second); got != 1 {
		t.Fatalf("camera lerp = %.2f, want clamped 1.00", got)
	}
}

func TestCameraZoomLerpMatchesRobrowserZoomCurve(t *testing.T) {
	if got := cameraZoomLerp(cameraFollowLerp(time.Second / 60)); math.Abs(got-0.2) > 0.001 {
		t.Fatalf("zoom lerp = %.3f, want 0.200", got)
	}
	if got := cameraZoomLerp(cameraFollowLerp(100 * time.Millisecond)); got != 1 {
		t.Fatalf("large zoom lerp = %.2f, want clamped 1.00", got)
	}
}

func TestFollowCameraZoomEasesTowardTargetLikeRobrowser(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	ctx := client.Context{World: world}

	now := time.Now()
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, zoom: 125, zoomTarget: 125, lastUpdate: now}
	camera.ZoomByDelta(-15)

	if got := camera.currentZoom(); got != 125 {
		t.Fatalf("current zoom before update = %.1f, want unchanged 125.0", got)
	}
	if got := camera.targetZoom(); got != 110 {
		t.Fatalf("target zoom = %.1f, want 110.0", got)
	}

	camera.Update(ctx, now.Add(time.Second/60))
	if math.Abs(camera.currentZoom()-122) > 0.01 {
		t.Fatalf("smoothed zoom = %.2f, want 122.00", camera.currentZoom())
	}
}

func TestAppendActorDrawEntryUsesPathRenderDirection(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	actor := worldstate.Actor{
		X:            2,
		Y:            1,
		Dir:          4,
		Moving:       true,
		FromX:        0,
		FromY:        0,
		ToX:          2,
		ToY:          1,
		MoveStarted:  now.Add(-225 * time.Millisecond),
		MoveDuration: 450 * time.Millisecond,
		MovePath: []worldstate.WalkStep{
			{X: 0, Y: 0},
			{X: 0, Y: 1},
			{X: 1, Y: 1},
			{X: 2, Y: 1},
		},
	}
	projection := newSceneProjectionForTarget(800, 600, 0.5, 1.5, 0)

	entries := appendActorDrawEntry(nil, world, projection, actor, false, now, 800, 600)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if got, want := entries[0].actor.Dir, directionFromDelta(0, 1, 1, 1, 4); got != want {
		t.Fatalf("entry direction = %d, want %d", got, want)
	}
}

func TestActorBillboardSortDepthUsesTopInCameraProjection(t *testing.T) {
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 0)
	footDepth := projection.Depth(10.5, 20.5, 0)
	topDepth := projection.Depth(10.5, 20.5, actorBillboardWorldHeightUnit)
	got := actorBillboardSortDepth(projection, 10.5, 20.5, 0)
	want := math.Min(footDepth, topDepth)
	if got != want {
		t.Fatalf("billboard depth = %.4f, want closer of foot %.4f and top %.4f", got, footDepth, topDepth)
	}
	if got >= footDepth {
		t.Fatalf("billboard depth = %.4f, want closer than foot depth %.4f", got, footDepth)
	}
}

func TestCameraYawForIndoorMapIsLocked(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "geffen_in"},
	}
	if got := cameraYawForMap(ctx); got != -45 {
		t.Fatalf("indoor camera yaw = %.1f, want -45.0", got)
	}
	ctx.World.MapName = "prontera"
	if got := cameraYawForMap(ctx); got != defaultSceneCameraYaw {
		t.Fatalf("outdoor camera yaw = %.1f, want %.1f", got, defaultSceneCameraYaw)
	}
}

func TestCameraYawForFixedViewPointMapIsLocked(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "viewpointtable.txt"), []byte("fixed_view.rsw#150#50#170#30#30#30#60#30#45#\nfree_view.rsw#150#50#170#-360#360#0#60#30#45#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "fixed_view"},
	}
	if got := cameraYawForMap(ctx); got != -30 {
		t.Fatalf("fixed viewpoint yaw = %.1f, want -30.0", got)
	}
	if !cameraRotationLockedForMap(ctx) {
		t.Fatal("fixed viewpoint should lock camera rotation")
	}
	ctx.World.MapName = "free_view"
	if got := cameraYawForMap(ctx); got != defaultSceneCameraYaw {
		t.Fatalf("free viewpoint yaw = %.1f, want %.1f", got, defaultSceneCameraYaw)
	}
	if cameraRotationLockedForMap(ctx) {
		t.Fatal("free viewpoint should not lock camera rotation")
	}
}

func TestIndoorCameraZoomIsLockedWithoutLosingOutdoorZoom(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	world := worldstate.New()
	world.MapName = "geffen_in"
	ctx := client.Context{
		Resources: manager,
		World:     world,
	}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0, zoom: 150}

	indoorProjection := camera.Projection(ctx, 800, 600, time.Now())
	if got := indoorProjection.cameraZoom; got != sceneCameraZoom() {
		t.Fatalf("indoor projection zoom = %.1f, want %.1f", got, sceneCameraZoom())
	}

	ctx.World.MapName = "prontera"
	outdoorProjection := camera.Projection(ctx, 800, 600, time.Now())
	if got := outdoorProjection.cameraZoom; got != 150 {
		t.Fatalf("restored outdoor projection zoom = %.1f, want 150.0", got)
	}
}

func TestFollowCameraProjectionIncludesRuntimeYawOffset(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	ctx := client.Context{World: world}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.Rotate(90)
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraYaw; got != 90 {
		t.Fatalf("projection yaw = %.1f, want 90.0", got)
	}

	camera.ResetRotation()
	projection = camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraYaw; got != defaultSceneCameraYaw {
		t.Fatalf("reset projection yaw = %.1f, want %.1f", got, defaultSceneCameraYaw)
	}
}

func TestFollowCameraProjectionKeepsIndoorBaseYaw(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "geffen_in"},
	}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.Rotate(90)
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraYaw; got != -45 {
		t.Fatalf("indoor projection yaw = %.1f, want -45.0", got)
	}
	if camera.yawOffset != 0 {
		t.Fatalf("indoor projection left yaw offset = %.1f, want reset", camera.yawOffset)
	}
}

func TestCameraRotationIsDisabledOnIndoorMap(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	inputState := input.NewState()
	inputState.SetMousePosition(100, 100)
	inputState.SetMouseButton(input.MouseButtonRight, true)
	inputState.SetMousePosition(200, 100)
	mode := &WorldMode{}
	mode.camera.Rotate(90)
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "geffen_in"},
		Input:     inputState,
		ScreenW:   800,
		ScreenH:   600,
	}

	mode.updateCameraRotation(ctx)
	if mode.camera.yawOffset != 0 {
		t.Fatalf("indoor camera yaw offset = %.1f, want reset", mode.camera.yawOffset)
	}
}

func TestCameraDragYawDeltaMatchesRobrowserScale(t *testing.T) {
	if got := cameraDragYawDelta(100, 1000); got != -72 {
		t.Fatalf("drag yaw delta = %.1f, want -72.0", got)
	}
	if got := cameraDragYawDelta(-100, 1000); got != 72 {
		t.Fatalf("reverse drag yaw delta = %.1f, want 72.0", got)
	}
	if got := cameraDragYawDelta(100, 0); got != 0 {
		t.Fatalf("zero width drag yaw delta = %.1f, want 0", got)
	}
}

func TestCameraWheelZoomFactorZoomsInOnWheelUp(t *testing.T) {
	if got := cameraWheelZoomFactor(1); got >= 1 {
		t.Fatalf("wheel up factor = %.3f, want zoom-in factor below 1", got)
	}
	if got := cameraWheelZoomFactor(-1); got <= 1 {
		t.Fatalf("wheel down factor = %.3f, want zoom-out factor above 1", got)
	}
}

func TestCameraWheelZoomDeltaMatchesRobrowserStep(t *testing.T) {
	if got := cameraWheelZoomDelta(1); got != -15 {
		t.Fatalf("wheel up delta = %.1f, want -15", got)
	}
	if got := cameraWheelZoomDelta(-2); got != 15 {
		t.Fatalf("wheel down delta = %.1f, want 15", got)
	}
	if got := cameraWheelZoomDelta(0.25); got != -15 {
		t.Fatalf("trackpad wheel delta = %.1f, want one notch", got)
	}
}

func TestCameraZoomRangeMatchesRobrowserOutdoorDefaults(t *testing.T) {
	if got := sceneCameraZoom(); got != 125 {
		t.Fatalf("default zoom = %.1f, want reference client default 125", got)
	}
	if defaultCameraMinZoom != 65 || defaultCameraMaxZoom != 165 {
		t.Fatalf("zoom range = %.1f..%.1f, want goro outdoor 65..165", defaultCameraMinZoom, defaultCameraMaxZoom)
	}
}

func TestSceneFogDepthAtTargetMatchesRobrowserDefaultZoom(t *testing.T) {
	projection := newSceneProjectionForTargetYawZoom(1280, 720, 10.5, 20.5, 0, 0, defaultSceneCameraZoom)
	got := projection.FogDepth(10.5, 20.5, 0)
	const want = 1000 * ((defaultSceneCameraZoom * 0.5) - 1) / 999
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("target fog depth = %.3f, want %.3f", got, want)
	}
}

func TestCameraPinchZoomFactorZoomsInWhenFingersSpread(t *testing.T) {
	if got := cameraPinchZoomFactor(25); got >= 1 {
		t.Fatalf("pinch spread factor = %.3f, want zoom-in factor below 1", got)
	}
	if got := cameraPinchZoomFactor(-25); got <= 1 {
		t.Fatalf("pinch close factor = %.3f, want zoom-out factor above 1", got)
	}
}

func TestFollowCameraZoomIsClampedAndProjected(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{World: world}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.ZoomBy(0.1)
	if got := camera.currentZoom(); got != defaultCameraMinZoom {
		t.Fatalf("zoom in clamp = %.1f, want %.1f", got, defaultCameraMinZoom)
	}
	camera.ZoomBy(10)
	if got := camera.currentZoom(); got != defaultCameraMaxZoom {
		t.Fatalf("zoom out clamp = %.1f, want %.1f", got, defaultCameraMaxZoom)
	}
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraZoom; got != defaultCameraMaxZoom {
		t.Fatalf("projection zoom = %.1f, want %.1f", got, defaultCameraMaxZoom)
	}
}

func TestCursorRotateInfoMatchesRobrowser(t *testing.T) {
	info := cursorInfo(cursorActionRotate)
	if info.delayMult != 1 {
		t.Fatalf("rotate cursor info = %+v", info)
	}
}

func TestWorldSceneClearColorMatchesReferenceDefaults(t *testing.T) {
	if got := worldSceneClearColor("geffen_in"); got != (color.RGBA{A: 255}) {
		t.Fatalf("default map clear color = %#v, want black", got)
	}
	if got := worldSceneClearColor("data/yuno.gat"); got != (color.RGBA{R: 0x66, G: 0x99, B: 0xcc, A: 255}) {
		t.Fatalf("yuno clear color = %#v", got)
	}
	if got := worldSceneClearColor("airplane_01"); got != (color.RGBA{R: 0x66, G: 0x99, B: 0xcc, A: 255}) {
		t.Fatalf("airplane_01 clear color = %#v", got)
	}
	if got := worldSceneClearColor("sch_gld"); got != (color.RGBA{R: 0x66, G: 0x99, B: 0xcc, A: 255}) {
		t.Fatalf("sch_gld clear color = %#v", got)
	}
	if got := worldSceneClearColor("bat_fild02"); got != (color.RGBA{A: 255}) {
		t.Fatalf("bat_fild02 clear color = %#v, want black", got)
	}
	if got := worldSceneClearColor("5@tower.rsw"); got != (color.RGBA{R: 0x33, G: 0x00, B: 0x33, A: 255}) {
		t.Fatalf("tower clear color = %#v", got)
	}
	if got := worldSceneClearColor("thana_boss.rsw"); got != (color.RGBA{R: 0xe0, G: 0xd4, B: 0xc2, A: 255}) {
		t.Fatalf("thana_boss clear color = %#v", got)
	}
}

func TestSceneLightingFromRSWMatchesReferenceDirection(t *testing.T) {
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 0,
		Latitude:  45,
		Diffuse:   [3]float32{1, 1, 1},
		Opacity:   1,
	}})
	want := modelPoint3{x: -math.Sqrt2 / 2, y: -math.Sqrt2 / 2, z: 0}
	if math.Abs(lighting.direction.x-want.x) > 0.0001 ||
		math.Abs(lighting.direction.y-want.y) > 0.0001 ||
		math.Abs(lighting.direction.z-want.z) > 0.0001 {
		t.Fatalf("light direction = %+v, want %+v", lighting.direction, want)
	}
}

func TestSceneLightingModelScaleIgnoresOpacityLikeRobrowser(t *testing.T) {
	opaque := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 45,
		Latitude:  45,
		Diffuse:   [3]float32{0.3, 0.3, 0.3},
		Ambient:   [3]float32{0.4, 0.4, 0.4},
		Opacity:   1,
	}})
	half := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 45,
		Latitude:  45,
		Diffuse:   [3]float32{0.3, 0.3, 0.3},
		Ambient:   [3]float32{0.4, 0.4, 0.4},
		Opacity:   0.5,
	}})
	normal := modelPoint3{x: 0, y: 1, z: 0}
	if got, want := half.modelScale(normal), opaque.modelScale(normal); got != want {
		t.Fatalf("model scale changed with opacity: got %+v want %+v", got, want)
	}
}

func TestSceneLightingModelScaleUsesReferenceMinimumLightWeight(t *testing.T) {
	lighting := sceneLighting{
		direction: modelPoint3{y: -1},
		diffuse:   modelPoint3{x: 1, y: 1, z: 1},
		ambient:   modelPoint3{},
		env:       modelPoint3{x: 1, y: 1, z: 1},
	}
	got := lighting.modelScale(modelPoint3{y: 1})
	want := modelPoint3{x: 0.5, y: 0.5, z: 0.5}
	if got != want {
		t.Fatalf("model scale = %+v, want %+v", got, want)
	}
}

func TestSceneLightingScaleClampsLightBeforeEnvLikeRobrowser(t *testing.T) {
	lighting := sceneLighting{
		direction: modelPoint3{y: -1},
		diffuse:   modelPoint3{x: 0.8, y: 0.8, z: 0.8},
		ambient:   modelPoint3{x: 0.8, y: 0.8, z: 0.8},
		env:       modelPoint3{x: 0.5, y: 0.5, z: 0.5},
	}
	got := lighting.groundScale(modelPoint3{y: -1})
	want := modelPoint3{x: 0.5, y: 0.5, z: 0.5}
	if got != want {
		t.Fatalf("ground scale = %+v, want %+v", got, want)
	}
}

func TestSmoothGNDTopNormalsKeepsFlatTilesUniform(t *testing.T) {
	gnd := testGNDWithTopHeights(2, 2, func(_, _ int) [4]float32 {
		return [4]float32{0, 0, 0, 0}
	})
	normals := buildSmoothGNDTopNormals(gnd)
	if len(normals) != 4 {
		t.Fatalf("normal count = %d, want 4", len(normals))
	}
	center := normals[0]
	for i := 1; i < 4; i++ {
		if !modelPointNear(center[0], center[i], 0.0001) {
			t.Fatalf("flat tile normals differ: %v vs %v", center[0], center[i])
		}
	}
}

func TestSmoothGNDTopNormalsVariesSlopedTileCorners(t *testing.T) {
	gnd := testGNDWithTopHeights(3, 3, func(x, y int) [4]float32 {
		return [4]float32{
			float32(x*y + y),
			float32(x*3 + y*y + 2),
			float32(x*x + y*4 + 1),
			float32(x*x + y*y + x*y + 7),
		}
	})
	normals := buildSmoothGNDTopNormals(gnd)
	tile := normals[1+1*gnd.Width]
	same := true
	for i := 1; i < 4; i++ {
		if !modelPointNear(tile[0], tile[i], 0.0001) {
			same = false
		}
	}
	if same {
		t.Fatalf("sloped tile normals are all equal: %+v", tile)
	}
}

func TestSurfaceVertexTintsUsePerVertexNormals(t *testing.T) {
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 45,
		Latitude:  45,
		Diffuse:   [3]float32{1, 1, 1},
		Ambient:   [3]float32{0, 0, 0},
		Opacity:   1,
	}})
	tints := surfaceVertexTints(uniformGNDSurfaceBaseTints(color.RGBA{}), [4]float32{}, [4]modelPoint3{
		{x: -0.5, y: -math.Sqrt2 / 2, z: -0.5},
		{x: 0.5, y: -math.Sqrt2 / 2, z: -0.5},
		{x: 0.5, y: -math.Sqrt2 / 2, z: 0.5},
		{x: -0.5, y: -math.Sqrt2 / 2, z: 0.5},
	}, lighting)
	if tints[0] == tints[1] && tints[0] == tints[2] && tints[0] == tints[3] {
		t.Fatalf("vertex tints are uniform: %+v", tints)
	}
}

func TestPosterizeGNDLightmapColorUsesReferenceClientBuckets(t *testing.T) {
	got := posterizeGNDLightmapColor(color.RGBA{R: 15, G: 31, B: 255, A: 77})
	want := color.RGBA{R: 0, G: 16, B: 240, A: 77}
	if got != want {
		t.Fatalf("posterized lightmap color = %+v, want %+v", got, want)
	}
}

func TestTopGNDSurfaceBaseTintsUseNeighborTileColors(t *testing.T) {
	gnd := &res.GND{
		Width:  2,
		Height: 2,
		Surfaces: []res.GNDSurface{
			{Color: color.RGBA{R: 10, G: 20, B: 30, A: 255}},
			{Color: color.RGBA{R: 40, G: 50, B: 60, A: 255}},
			{Color: color.RGBA{R: 70, G: 80, B: 90, A: 255}},
			{Color: color.RGBA{R: 100, G: 110, B: 120, A: 255}},
		},
		Cells: []res.GNDCell{
			{Top: 0, Front: -1, Right: -1},
			{Top: 1, Front: -1, Right: -1},
			{Top: 2, Front: -1, Right: -1},
			{Top: 3, Front: -1, Right: -1},
		},
	}
	tints := topGNDSurfaceBaseTints(gnd, 0, 0, color.RGBA{})
	want := [4]color.RGBA{
		{R: 10, G: 20, B: 30, A: 255},
		{R: 40, G: 50, B: 60, A: 255},
		{R: 100, G: 110, B: 120, A: 255},
		{R: 70, G: 80, B: 90, A: 255},
	}
	if tints != want {
		t.Fatalf("top GND tints = %+v, want %+v", tints, want)
	}
}

func testGNDWithTopHeights(width, height int, fn func(x, y int) [4]float32) *res.GND {
	gnd := &res.GND{
		Width:    width,
		Height:   height,
		Cells:    make([]res.GNDCell, width*height),
		Surfaces: []res.GNDSurface{{TextureID: 0, LightmapID: -1}},
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gnd.Cells[x+y*width] = res.GNDCell{Top: 0, Front: -1, Right: -1, Heights: fn(x, y)}
		}
	}
	return gnd
}

func modelPointNear(a, b modelPoint3, epsilon float64) bool {
	return math.Abs(a.x-b.x) <= epsilon && math.Abs(a.y-b.y) <= epsilon && math.Abs(a.z-b.z) <= epsilon
}

func TestGNDDrawBoundsUseCameraFootprint(t *testing.T) {
	gnd := &res.GND{Width: 200, Height: 200}
	projection := newSceneProjectionForTargetYaw(1024, 768, 200.5, 260.5, 8, 0)
	startX, endX, startY, endY, ok := gndDrawBounds(gnd, projection, 1024, 768)
	if !ok {
		t.Fatal("missing GND bounds")
	}
	centerX := gndTileFromWorld(projection.playerX)
	centerY := gndTileFromWorld(projection.playerY)
	if startX > centerX || endX < centerX || startY > centerY || endY < centerY {
		t.Fatalf("bounds %d..%d,%d..%d do not include center %d,%d", startX, endX, startY, endY, centerX, centerY)
	}
	if endX-startX >= gnd.Width-1 || endY-startY >= gnd.Height-1 {
		t.Fatalf("camera bounds %d..%d,%d..%d should not cover the full map", startX, endX, startY, endY)
	}
}

func TestGNDShadowMapPointMatchesReferenceClientCellCenterMapping(t *testing.T) {
	x, y := gndShadowMapPoint(10, 20)
	if x != 42 || y != 82 {
		t.Fatalf("shadow map point for even cell = %d,%d, want 42,82", x, y)
	}
	x, y = gndShadowMapPoint(11, 21)
	if x != 46 || y != 86 {
		t.Fatalf("shadow map point for odd cell = %d,%d, want 46,86", x, y)
	}
}

func TestActorShadowFactorAveragesGNDLightmapAlpha(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 128
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}

	got := actorShadowFactor(&worldstate.World{GND: gnd}, 3, 3)
	want := float64(128) / 255
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("shadow factor = %.4f, want %.4f", got, want)
	}
}

func TestActorShadowFactorIgnoresGroundShadowBelowElevatedGAT(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 32
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}
	gat := &res.GAT{
		Width:  8,
		Height: 8,
		Cells:  make([]res.GATCell, 64),
	}
	for i := range gat.Cells {
		gat.Cells[i].Heights = [4]float32{2, 2, 2, 2}
	}

	got := actorShadowFactor(&worldstate.World{GAT: gat, GND: gnd}, 3, 3)
	if got != 1 {
		t.Fatalf("elevated GAT shadow factor = %.4f, want 1", got)
	}
}

func TestActorShadowFactorDefaultsToLitWithoutGroundLightmap(t *testing.T) {
	if got := actorShadowFactor(nil, 3, 3); got != 1 {
		t.Fatalf("nil world shadow = %.2f, want 1", got)
	}
	gnd := &res.GND{
		Width:    1,
		Height:   1,
		Surfaces: []res.GNDSurface{{LightmapID: 0}},
		Cells:    []res.GNDCell{{Top: -1}},
	}
	if got := actorShadowFactor(&worldstate.World{GND: gnd}, 0, 0); got != 1 {
		t.Fatalf("missing top surface shadow = %.2f, want 1", got)
	}
}

func TestQuadHasInvalidPointDetectsCameraSentinel(t *testing.T) {
	points := [4]screenPoint{{x: -1 << 20, y: -1 << 20}, {x: 1, y: 1}, {x: 2, y: 1}, {x: 1, y: 2}}
	if !quadHasInvalidPoint(points) {
		t.Fatal("expected camera sentinel point to invalidate GND quad")
	}
}

func TestMapWaterPrefersGNDOverride(t *testing.T) {
	gnd := &res.GND{Water: res.GNDWater{Present: true, Level: -4, Type: 3, WaveHeight: 2, WaveSpeed: 5, WavePitch: 20, AnimSpeed: 6}}
	rsw := &res.RSW{Water: res.RSWWater{Level: -1, Type: 1}}
	water, ok := mapWater(gnd, rsw)
	if !ok {
		t.Fatal("missing water")
	}
	if water.Level != -4 || water.Type != 3 || water.WaveHeight != 2 || water.AnimSpeed != 6 {
		t.Fatalf("water = %+v, want GND override", water)
	}
}

func TestWaterUVsUseFourTileRepeat(t *testing.T) {
	uvs := waterUVs(3, 4)
	if uvs[0] != (texturePoint{u: 0.75, v: 0}) || uvs[2] != (texturePoint{u: 1.0, v: 0.25}) {
		t.Fatalf("water uvs = %+v", uvs)
	}
}

func TestWaterVisibleForCellUsesInvertedHeightConvention(t *testing.T) {
	water := res.RSWWater{Level: -2, WaveHeight: 0.5}
	if !waterVisibleForCell(res.GNDCell{Heights: [4]float32{-3, -2, -2, -2}}, water) {
		t.Fatal("expected water where terrain is below the water threshold")
	}
	if waterVisibleForCell(res.GNDCell{Heights: [4]float32{-1, -1.2, -0.5, 0}}, water) {
		t.Fatal("unexpected water where all terrain vertices are above the water threshold")
	}
}

func TestRayWalkCellHitsProjectedGATCellCenter(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(180, 160)
	projection := newSceneProjectionForTargetYawZoom(1280, 720, cellCenter(107), cellCenter(90), 0, -45, sceneCameraZoom())
	point := projection.Project(cellCenter(107), cellCenter(87), 0)

	x, y, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("ray walk cell = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
}

func TestRayWalkCellSkipsBlockedGATCell(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(180, 160)
	world.GAT.Cells[87*world.GAT.Width+107] = res.GATCell{Type: res.GATTypeNone}
	projection := newSceneProjectionForTargetYawZoom(1280, 720, cellCenter(107), cellCenter(90), 0, -45, sceneCameraZoom())
	point := projection.Project(cellCenter(107), cellCenter(87), 0)

	_, _, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	if ok {
		t.Fatal("blocked ray walk cell should not be picked")
	}
}

func TestRayWalkCellWorksAtCloseZoom(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(180, 160)
	projection := newSceneProjectionForTargetYawZoom(1280, 720, cellCenter(107), cellCenter(90), 0, -45, defaultCameraMinZoom)
	point := projection.Project(cellCenter(107), cellCenter(87), 0)

	x, y, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("close zoom ray walk cell = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
}

func TestClickedWalkTargetUsesRayWalkCell(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(180, 160)
	projection := newSceneProjectionForTargetYawZoom(1280, 720, cellCenter(107), cellCenter(90), 0, -45, sceneCameraZoom())
	point := projection.Project(cellCenter(107), cellCenter(87), 0)

	x, y, ok := clickedWalkTarget(client.Context{World: world}, projection, int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("clicked walk target = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
}

func TestHoveredWalkCellUsesRayWalkCell(t *testing.T) {
	world := worldstate.New()
	world.Player.X = 5
	world.Player.Y = 5
	world.GAT = flatWalkableGAT(12, 12)
	projection := newSceneProjectionForTarget(800, 600, 5.5, 5.5, 0)
	point := projection.Project(6.5, 5.5, 0)

	x, y, ok := hoveredWalkCell(client.Context{World: world}, projection, int(point.x), int(point.y))
	if !ok || x != 6 || y != 5 {
		t.Fatalf("hovered cell = %d,%d ok=%t, want 6,5 true", x, y, ok)
	}
	if _, _, ok := hoveredWalkCell(client.Context{World: world}, projection, -10000, -10000); ok {
		t.Fatal("hover should not fall back to nearest cell outside the projected map")
	}
}

func flatWalkableGAT(width, height int) *res.GAT {
	gat := &res.GAT{
		Width:  width,
		Height: height,
		Cells:  make([]res.GATCell, width*height),
	}
	for i := range gat.Cells {
		gat.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
	return gat
}

func TestTileCursorCellVertsUseGATHeights(t *testing.T) {
	gat := &res.GAT{
		Width:  4,
		Height: 4,
		Cells:  make([]res.GATCell, 16),
	}
	gat.Cells[2*gat.Width+1] = res.GATCell{Heights: [4]float32{2, 2, 2, 2}, Type: res.GATTypeWalkable}
	verts, ok := tileCursorCellVerts(gat, 1, 2)
	if !ok {
		t.Fatal("missing cursor cell")
	}
	if math.Abs(verts[0].y-2) > 0.001 {
		t.Fatalf("cursor vertex y = %.4f, want 2", verts[0].y)
	}
}

func TestApplyActorNameAckUpdatesWorldActor(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 1002})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 300, Name: "Guide#prontera", GuildName: "Knights"})

	if got := world.Actors[300].Name; got != "Guide" {
		t.Fatalf("actor name = %q, want Guide", got)
	}
	if got := world.Actors[300].GuildName; got != "Knights" {
		t.Fatalf("actor guild = %q, want Knights", got)
	}
}

func TestApplyActorNameAckUpdatesLocalPlayer(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 200, Name: "Kivutar", GuildName: "Goro"})

	if got := world.Player.Name; got != "Kivutar" {
		t.Fatalf("player name = %q, want Kivutar", got)
	}
	if got := world.Player.GuildName; got != "Goro" {
		t.Fatalf("player guild = %q, want Goro", got)
	}
}

func TestApplyActorNameAckPreservesLocalGuildOnEmptyNameAck(t *testing.T) {
	world := worldstate.New()
	world.Player.GuildName = "Goro"
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200, GuildName: "Goro"},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 200, Name: "Kivutar"})

	if got := world.Player.GuildName; got != "Goro" {
		t.Fatalf("player guild = %q, want Goro", got)
	}
	if got := ctx.Session.GuildName; got != "Goro" {
		t.Fatalf("session guild = %q, want Goro", got)
	}
}

func TestHandleMapChangeSameServerUpdatesMapAndResetsActors(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.SetPlayerPosition(10, 20, 4)
	world.UpsertActor(worldstate.Actor{ID: 300, Name: "Remote", X: 11, Y: 20})
	sessionState := &session.Session{AccountID: 100, CharID: 200, PlayerDir: 4}
	uiManager := &worldModeTestUIManager{}
	ctx := client.Context{
		Session:   sessionState,
		World:     world,
		Input:     input.NewState(),
		UIManager: uiManager,
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 10, Message: "Warping..."})
	if !mode.ui.npcDialog.Update(ctx) {
		t.Fatal("npc dialog did not publish before map change")
	}
	if len(uiManager.overlays) == 0 {
		t.Fatal("npc dialog overlay was not published before map change")
	}

	next := mode.handleMapChange(ctx, network.MapChange{MapName: "geffen", X: 120, Y: 80})

	if next == nil || next.Name() != "world" {
		t.Fatalf("next mode = %#v, want world", next)
	}
	if len(uiManager.overlays) != 0 {
		t.Fatalf("npc dialog overlays after map change = %d, want 0", len(uiManager.overlays))
	}
	if world.MapName != "geffen" || sessionState.Zone.MapName != "geffen" {
		t.Fatalf("map = world %q session %q, want geffen", world.MapName, sessionState.Zone.MapName)
	}
	if world.Player.X != 120 || world.Player.Y != 80 || sessionState.PlayerX != 120 || sessionState.PlayerY != 80 {
		t.Fatalf("position = world %d,%d session %d,%d", world.Player.X, world.Player.Y, sessionState.PlayerX, sessionState.PlayerY)
	}
	if len(world.Actors) != 0 {
		t.Fatalf("actors were not cleared: %+v", world.Actors)
	}
}

func TestNextWorldModeReusesMinimapOverlay(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.SetPlayerPosition(10, 20, 4)
	ctx := client.Context{
		World:     world,
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.minimap.Update(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("minimap overlays before map change = %d, want 1", len(manager.overlays))
	}

	next := mode.nextWorldMode()
	world.MapName = "geffen"
	world.SetPlayerPosition(120, 80, 4)
	next.ui.minimap.Update(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("minimap overlays after map change = %d, want 1", len(manager.overlays))
	}
}

func TestNextWorldModeCarriesOpenInventoryWindow(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{Items: []session.InventoryItem{
			{Index: 2, ItemID: 501, Type: 0, Amount: 3, Identified: true},
		}},
	}
	ctx := client.Context{
		Session:   sessionState,
		World:     worldstate.New(),
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.inventoryBag.Toggle(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) == 0 {
		t.Fatal("inventory overlay was not published before map change")
	}

	next := mode.nextWorldMode()
	if len(manager.overlays) != 1 {
		t.Fatalf("inventory overlays after mode replacement = %d, want carried overlay", len(manager.overlays))
	}
	next.ui.inventoryBag.Update(ctx, &next.ui.shortcutBar, &next.ui.storageWindow, &next.ui.cartWindow, nil, &next.ui.equipmentWindow, &next.ui.itemInfoWindow)
	if len(manager.overlays) != 1 {
		t.Fatalf("inventory overlays after next mode update = %d, want 1", len(manager.overlays))
	}
}

func TestNextWorldModeCarriesAndRebindsWhisperWindow(t *testing.T) {
	ctx := client.Context{
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.whisperWindow.Open(ctx, "Alice")
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("whisper overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.whisperWindow.IsOpen() {
		t.Fatal("next world mode did not carry open whisper window")
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("whisper overlays after rebind = %d, want 1", len(manager.overlays))
	}
	if manager.overlays[0] == previousOverlay {
		t.Fatal("whisper overlay was not rebound")
	}

	next.ui.whisperWindow.AddError(ctx, "send failed")
	if len(manager.overlays) != 1 {
		t.Fatalf("whisper overlays after refresh = %d, want 1", len(manager.overlays))
	}
}

func TestNextWorldModeCarriesAndRebindsFriendSettingsWindow(t *testing.T) {
	ctx := client.Context{
		Session:   &session.Session{},
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.friendSettings.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("friend settings overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.friendSettings.IsOpen() {
		t.Fatal("next world mode did not carry open friend settings window")
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("friend settings overlays after rebind = %d, want 1", len(manager.overlays))
	}
	if manager.overlays[0] == previousOverlay {
		t.Fatal("friend settings overlay was not rebound")
	}
}

func TestNextWorldModeCarriesAndRebindsPartySettingsWindow(t *testing.T) {
	ctx := client.Context{
		Session: &session.Session{Party: session.Party{
			Name:    "Goro",
			Members: []session.PartyMember{{AccountID: 10, Role: 0}},
		}},
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.partySettings.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("party settings overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.partySettings.IsOpen() {
		t.Fatal("next world mode did not carry open party settings window")
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("party settings overlays after rebind = %d, want 1", len(manager.overlays))
	}
	if manager.overlays[0] == previousOverlay {
		t.Fatal("party settings overlay was not rebound")
	}
}

func TestNextWorldModeCarriesAndRebindsPartyHelperWindows(t *testing.T) {
	ctx := client.Context{
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.partyCreate.Open(ctx)
	mode.ui.partyInvite.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 2 {
		t.Fatalf("party helper overlays before map change = %d, want 2", len(manager.overlays))
	}
	previousCreate := manager.overlays[0]
	previousInvite := manager.overlays[1]

	next := mode.nextWorldMode()
	if !next.ui.partyCreate.IsOpen() || !next.ui.partyInvite.IsOpen() {
		t.Fatalf("next world mode did not carry helper windows create=%t invite=%t", next.ui.partyCreate.IsOpen(), next.ui.partyInvite.IsOpen())
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 2 {
		t.Fatalf("party helper overlays after rebind = %d, want 2", len(manager.overlays))
	}
	if manager.overlays[0] == previousCreate || manager.overlays[1] == previousInvite {
		t.Fatal("party helper overlay was not rebound")
	}
}

func TestHandleMapChangeSameLoadedMapReusesModeAndSnapsCamera(t *testing.T) {
	world := worldstate.New()
	world.MapName = "izlude"
	world.GND = &res.GND{}
	world.SetPlayerPosition(10, 20, 4)
	world.UpsertActor(worldstate.Actor{ID: 300, Name: "Remote", X: 11, Y: 20})
	sessionState := &session.Session{AccountID: 100, CharID: 200, PlayerDir: 4}
	ctx := client.Context{
		Session: sessionState,
		World:   world,
	}
	mode := &WorldMode{}

	next := mode.handleMapChange(ctx, network.MapChange{MapName: "izlude", X: 114, Y: 145})

	if next != nil {
		t.Fatalf("next mode = %#v, want nil same-mode reuse", next)
	}
	if world.Player.X != 114 || world.Player.Y != 145 || sessionState.PlayerX != 114 || sessionState.PlayerY != 145 {
		t.Fatalf("position = world %d,%d session %d,%d", world.Player.X, world.Player.Y, sessionState.PlayerX, sessionState.PlayerY)
	}
	if len(world.Actors) != 0 {
		t.Fatalf("actors were not cleared: %+v", world.Actors)
	}
	if !mode.camera.initialized || mode.camera.x != 114.5 || mode.camera.y != 145.5 {
		t.Fatalf("camera = initialized %t %.2f,%.2f, want 114.5,145.5", mode.camera.initialized, mode.camera.x, mode.camera.y)
	}
}

func TestActorDisplayNameUsesSelectedCharacterForPlayer(t *testing.T) {
	ctx := client.Context{Session: &session.Session{CharID: 200, Selected: session.Character{ID: 200, Name: "Kivutar"}}}

	if got := actorDisplayName(ctx, worldstate.Actor{Name: "Player"}, true); got != "Kivutar" {
		t.Fatalf("display name = %q, want Kivutar", got)
	}
}

func TestActorDisplayNameIncludesPartyName(t *testing.T) {
	ctx := client.Context{Session: &session.Session{
		CharID:   200,
		Selected: session.Character{ID: 200, Name: "Kivutar"},
		Party: session.Party{
			Name:    "Goro",
			Members: []session.PartyMember{{AccountID: 300, Name: "Alice"}},
		},
	}}

	if got := actorDisplayName(ctx, worldstate.Actor{Name: "Player"}, true); got != "Kivutar (Goro)" {
		t.Fatalf("local display name = %q, want Kivutar (Goro)", got)
	}
	if got := actorDisplayName(ctx, worldstate.Actor{ID: 300, Name: "Alice"}, false); got != "Alice (Goro)" {
		t.Fatalf("party member display name = %q, want Alice (Goro)", got)
	}
	if got := actorDisplayName(ctx, worldstate.Actor{ID: 400, Name: "Bob"}, false); got != "Bob" {
		t.Fatalf("non-party display name = %q, want Bob", got)
	}
}

func TestActorDisplayLabelsIncludeGuildNameOnSecondLine(t *testing.T) {
	ctx := client.Context{Session: &session.Session{CharID: 200, Selected: session.Character{ID: 200, Name: "Kivutar"}}}

	labels := actorDisplayLabels(ctx, worldstate.Actor{Name: "Player", GuildName: "Knights"}, true)
	if len(labels) != 2 || labels[0] != "Kivutar" || labels[1] != "Knights" {
		t.Fatalf("local labels = %#v, want Kivutar / Knights", labels)
	}

	labels = actorDisplayLabels(ctx, worldstate.Actor{ID: 300, Name: "Alice", GuildName: "Knights", HasObjectType: true, ObjectType: actorObjectTypePC}, false)
	if len(labels) != 2 || labels[0] != "Alice" || labels[1] != "Knights" {
		t.Fatalf("actor labels = %#v, want Alice / Knights", labels)
	}

	labels = actorDisplayLabels(ctx, worldstate.Actor{Name: "Poring", GuildName: "Knights", HasObjectType: true, ObjectType: actorObjectTypeMob}, false)
	if len(labels) != 1 || labels[0] != "Poring" {
		t.Fatalf("mob labels = %#v, want Poring only", labels)
	}
}

func TestGuildCreationResultAppliesPendingLocalGuildName(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{PendingGuildName: "Knights"},
		World:   world,
	}
	var mode WorldMode

	mode.handleGuildCreationResult(ctx, network.GuildCreationResult{Result: 0})

	if got := ctx.Session.GuildName; got != "Knights" {
		t.Fatalf("session guild = %q, want Knights", got)
	}
	if got := world.Player.GuildName; got != "Knights" {
		t.Fatalf("player guild = %q, want Knights", got)
	}
	if got := ctx.Session.PendingGuildName; got != "" {
		t.Fatalf("pending guild = %q, want empty", got)
	}
}

func TestHandleGuildNoticeUpdatesSessionAndAddsGuildConsoleMessages(t *testing.T) {
	sessionState := &session.Session{}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState}

	mode.handleGuildNotice(ctx, network.GuildNotice{
		Subject: " Maintenance ",
		Notice:  " Gather in Prontera. ",
	})

	if got := sessionState.Guild.NoticeSubject; got != "Maintenance" {
		t.Fatalf("notice subject = %q, want Maintenance", got)
	}
	if got := sessionState.Guild.Notice; got != "Gather in Prontera." {
		t.Fatalf("notice = %q, want Gather in Prontera.", got)
	}
	messages := mode.ui.console.Messages()
	if len(messages) != 2 {
		t.Fatalf("console messages = %+v, want 2 notice lines", messages)
	}
	if messages[0].Text != "[ Maintenance ]" || messages[1].Text != "[ Gather in Prontera. ]" {
		t.Fatalf("console messages = %+v", messages)
	}
	wantColor := color.RGBA{R: 255, G: 255, B: 99, A: 255}
	if messages[0].Color != wantColor || messages[1].Color != wantColor {
		t.Fatalf("console message colors = %+v, want %+v", messages, wantColor)
	}
}

func TestHandleGuildNoticeSkipsEmptyConsoleLines(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{}}

	mode.handleGuildNotice(ctx, network.GuildNotice{
		Subject: " ",
		Notice:  " Guild event tonight.\nMeet in Prontera. ",
	})

	messages := mode.ui.console.Messages()
	if len(messages) != 2 || messages[0].Text != "[ Guild event tonight. ]" || messages[1].Text != "[ Meet in Prontera. ]" {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestActorDisplayNameUsesServerNameBeforeFallback(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Name: "Kafra Employee#izlude", Job: 1002}

	if got := actorDisplayName(ctx, actor, false); got != "Kafra Employee" {
		t.Fatalf("display name = %q, want Kafra Employee", got)
	}
}

func TestActorDisplayNameUsesImportedMonsterFallback(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: 1002}

	if got := actorDisplayName(ctx, actor, false); got != "Poring" {
		t.Fatalf("display name = %q, want Poring from imported DB", got)
	}
}

func TestActorDisplayNameDoesNotLabelUnnamedPlayerJob(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: 0}

	if got := actorDisplayName(ctx, actor, false); got != "" {
		t.Fatalf("display name = %q, want empty", got)
	}
}

func TestHoverActorServerNameLookupIncludesCompanions(t *testing.T) {
	for _, actor := range []worldstate.Actor{
		{HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
		{HasObjectType: true, ObjectType: actorObjectTypeMercenary},
	} {
		if !shouldUseServerNameForHoverActor(actor) {
			t.Fatalf("companion actor should request server name: %+v", actor)
		}
	}
}

func TestHoveredActorDisplayNameUsesServerNameForNPC(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	mode := &WorldMode{}
	actor := worldstate.Actor{
		Job:           84,
		ObjectType:    actorObjectTypeNPC,
		HasObjectType: true,
	}

	if got := mode.hoveredActorDisplayName(ctx, actor, time.Now()); got != "4 M 02" {
		t.Fatalf("hovered NPC name = %q, want imported resource label", got)
	}
	actor.Name = "Kafra Employee#izlude"
	if got := mode.hoveredActorDisplayName(ctx, actor, time.Now()); got != "Kafra Employee" {
		t.Fatalf("hovered NPC server name = %q, want Kafra Employee", got)
	}
}

func TestHoveredActorDisplayNameUsesServerNameForMonster(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	mode := &WorldMode{}
	actor := worldstate.Actor{
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}

	if got := mode.hoveredActorDisplayName(ctx, actor, time.Now()); got != "Poring" {
		t.Fatalf("hovered monster name = %q, want imported monster label", got)
	}
	actor.Name = "Poring"
	if got := mode.hoveredActorDisplayName(ctx, actor, time.Now()); got != "Poring" {
		t.Fatalf("hovered monster server name = %q, want Poring", got)
	}
}

func TestFormatConsoleMessageUsesMsgStringTable(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "msgstringtable.txt"), []byte("ignored#\nYou got %d items.#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	got := formatConsoleMessage(manager, network.ChatMessage{MessageID: 1, Value: 3})
	if got != "You got 3 items." {
		t.Fatalf("message = %q", got)
	}
}

func TestColoredConsoleMessageUsesPacketColor(t *testing.T) {
	console := &gameui.ChatConsole{}
	addConsoleMessage(console, nil, network.ChatMessage{Text: "Experience Gained Base:1 (0.01%) Job:1 (0.01%)", Color: 0x00B5FFB5, HasColor: true})

	messages := console.Messages()
	wantColor := color.RGBA{R: 0xB5, G: 0xFF, B: 0xB5, A: 255}
	if len(messages) != 1 || messages[0].Text != "Experience Gained Base:1 (0.01%) Job:1 (0.01%)" || messages[0].Color != wantColor {
		t.Fatalf("messages = %+v, want color %+v", messages, wantColor)
	}
}

func TestFormatPickupConsoleMessageUsesMsgStringAndItemName(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	msgTable := strings.Repeat("ignored#\n", 153) + "You got %s %d.#\n"
	if err := os.WriteFile(filepath.Join(dataDir, "msgstringtable.txt"), []byte(msgTable), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "idnum2itemdisplaynametable.txt"), []byte("938#Apple#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	got := formatPickupConsoleMessage(manager, network.ItemPickupAck{ItemID: 938, Amount: 2, Identified: true})
	if got != "You got Apple 2." {
		t.Fatalf("pickup message = %q", got)
	}
}

func TestFormatPickupConsoleMessageFallback(t *testing.T) {
	got := formatPickupConsoleMessage(nil, network.ItemPickupAck{ItemID: 938, Amount: 0})
	if got != "You got item 938 1." {
		t.Fatalf("pickup message = %q", got)
	}
}

func TestActorDisplayNameDoesNotLabelWarpPortal(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: actorJobWarpPortal}

	if got := actorDisplayName(ctx, actor, false); got != "" {
		t.Fatalf("display name = %q, want empty", got)
	}
	if !isWarpActor(actor) {
		t.Fatal("expected warp actor classification")
	}
}

func TestMapFadeAlphaTransitionsThroughBlack(t *testing.T) {
	start := time.Unix(100, 0)
	mode := &WorldMode{}
	mode.startMapFadeOut(network.MapChange{MapName: "geffen"}, start)

	if got := mode.mapFadeAlpha(start); got != 0 {
		t.Fatalf("fade-out start alpha = %d, want 0", got)
	}
	if got := mode.mapFadeAlpha(start.Add(mapFadeOutDuration)); got != 255 {
		t.Fatalf("fade-out end alpha = %d, want 255", got)
	}

	mode.mapFade = mapFadeState{phase: mapFadeHold, started: start}
	if got := mode.mapFadeAlpha(start.Add(time.Second)); got != 255 {
		t.Fatalf("hold alpha = %d, want 255", got)
	}

	mode.startMapFadeIn(start)
	if got := mode.mapFadeAlpha(start); got != 255 {
		t.Fatalf("fade-in start alpha = %d, want 255", got)
	}
	if got := mode.mapFadeAlpha(start.Add(mapFadeInDuration)); got != 0 {
		t.Fatalf("fade-in end alpha = %d, want 0", got)
	}
}

func TestApplyInventoryItemListReplacesExistingAmount(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 938, Amount: 3}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryItemList(ctx, []network.InventoryItem{{
		Index:      7,
		ItemID:     938,
		Type:       3,
		Identified: true,
		Amount:     5,
	}})

	if len(sessionState.Inventory.Items) != 1 {
		t.Fatalf("inventory item count = %d, want 1", len(sessionState.Inventory.Items))
	}
	if got := sessionState.Inventory.Items[0]; got.Amount != 5 || !got.Identified || got.Type != 3 {
		t.Fatalf("inventory item = %+v, want replaced amount/type", got)
	}
}

func TestInventoryItemDeleteDecrementsAndRemoves(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 938, Amount: 3}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryItemDelete(ctx, network.InventoryItemDelete{Index: 7, Amount: 2})
	if got := sessionState.Inventory.Items[0].Amount; got != 1 {
		t.Fatalf("amount after partial delete = %d, want 1", got)
	}
	applyInventoryItemDelete(ctx, network.InventoryItemDelete{Index: 7, Amount: 1})
	if len(sessionState.Inventory.Items) != 0 {
		t.Fatalf("inventory item count = %d, want 0", len(sessionState.Inventory.Items))
	}
}

func TestUseItemAckSetsRemainingAmount(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 12, ItemID: 512, Amount: 4}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyUseItemAck(ctx, network.UseItemAck{Index: 12, ItemID: 512, Amount: 3, Result: 1})
	if got := sessionState.Inventory.Items[0].Amount; got != 3 {
		t.Fatalf("item amount = %d, want 3", got)
	}

	applyUseItemAck(ctx, network.UseItemAck{Index: 12, ItemID: 512, Amount: 0, Result: 1})
	if len(sessionState.Inventory.Items) != 0 {
		t.Fatalf("inventory item count = %d, want 0", len(sessionState.Inventory.Items))
	}
}

func TestItemIdentifyAckMarksInventoryItemIdentified(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 7, ItemID: 1201, Type: 5, Identified: false, Equip: true},
				{Index: 9, ItemID: 1202, Type: 5, Identified: false, Equip: true},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyItemIdentifyAck(ctx, network.ItemIdentifyAck{Index: 9, Success: true})

	if sessionState.Inventory.Items[0].Identified {
		t.Fatal("wrong item was identified")
	}
	if !sessionState.Inventory.Items[1].Identified {
		t.Fatal("target item was not identified")
	}
}

func TestItemIdentifyAckFailureDoesNotChangeInventory(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 1201, Type: 5, Identified: false, Equip: true}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyItemIdentifyAck(ctx, network.ItemIdentifyAck{Index: 7, Success: false})

	if sessionState.Inventory.Items[0].Identified {
		t.Fatal("failed identify ack changed item state")
	}
}

func TestUseItemAckFailureDoesNotChangeInventory(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 12, ItemID: 512, Amount: 4}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyUseItemAck(ctx, network.UseItemAck{Index: 12, ItemID: 512, Amount: 0, Result: 0})
	if got := sessionState.Inventory.Items[0].Amount; got != 4 {
		t.Fatalf("item amount = %d, want 4", got)
	}
}

func TestUseItemAckAddsItemUseEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 501, AID: 2000000, Amount: 2, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 2000000 || effect.effectID != effectPotionRed || effect.x != 10 || effect.y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestUseItemAckDispatchesAllMappedItemEffectArrays(t *testing.T) {
	const itemID uint16 = 65000
	itemEffectSpecs[itemID] = itemEffectSpec{
		effectIDs:         []int{effectPotionRed, effectBlessing},
		effectIDsOnCaster: []int{effectEndure},
	}
	defer delete(itemEffectSpecs, itemID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: itemID, AID: 1100, Amount: 2, Result: 1})

	want := []struct {
		effectID int
		actorID  uint32
		x        int
		y        int
	}{
		{effectPotionRed, 1100, 12, 22},
		{effectBlessing, 1100, 12, 22},
		{effectEndure, 1100, 12, 22},
	}
	if len(mode.worldEffects) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.worldEffects), len(want), mode.worldEffects)
	}
	for i, wantEffect := range want {
		got := mode.worldEffects[i]
		if got.effectID != wantEffect.effectID || got.actorID != wantEffect.actorID || got.x != wantEffect.x || got.y != wantEffect.y {
			t.Fatalf("effect %d = %+v, want %+v", i, got, wantEffect)
		}
	}
}

func TestButterflyWingEffectIsPinnedAtUsePosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 602, AID: 2000000, Amount: 1, Result: 1})
	world.Player.X = 30
	world.Player.Y = 40

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 0 || effect.effectID != effectTeleportation || effect.x != 10 || effect.y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestGroundSkillNotifyAddsCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 21, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 0 || effect.effectID != effectThunderStorm || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 21, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("deduped world effects = %d, want 1", len(mode.worldEffects))
	}
}

func TestPneumaGroundSkillNotifyAddsCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 25, SourceID: 2000000, Level: 1, X: 123, Y: 456})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 0 || effect.effectID != effectPneuma || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9001, CreatorID: 2000000, UnitID: 126, X: 123, Y: 456, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 9001 || effect.effectID != effectSafetyWall || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9001})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestFireWallSkillUnitEntryUsesPersistentEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9007, CreatorID: 2000000, UnitID: 127, X: 12, Y: 34, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	effect := mode.worldEffects[0]
	if effect.actorID != 9007 || effect.effectID != effectFireWall || effect.x != 12 || effect.y != 34 {
		t.Fatalf("effect = %+v", effect)
	}
	if !effect.persistent {
		t.Fatalf("fire wall skill unit effect is not persistent")
	}
	if effect.expires.Sub(effect.starts) < skillUnitEffectFallbackDuration {
		t.Fatalf("fire wall lifetime = %s, want skill unit fallback", effect.expires.Sub(effect.starts))
	}
	if effect.duration != 0 {
		t.Fatalf("fire wall animation override = %s, want native component timing", effect.duration)
	}
}

func TestMappedSkillUnitEntriesUsePersistentEffects(t *testing.T) {
	for unitID, spec := range skillUnitEffectSpecs {
		if len(spec.effectIDs) == 0 {
			continue
		}
		world := worldstate.New()
		world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
		mode := &WorldMode{}
		ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
		entryID := uint32(unitID) + 100000

		mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: entryID, CreatorID: 2000000, UnitID: unitID, X: 12, Y: 34, Visible: true})
		if len(mode.worldEffects) != len(spec.effectIDs) {
			t.Fatalf("unit %d world effects = %d, want %d", unitID, len(mode.worldEffects), len(spec.effectIDs))
		}
		for _, effect := range mode.worldEffects {
			if !effect.persistent {
				t.Fatalf("unit %d effect %d is not persistent", unitID, effect.effectID)
			}
		}
	}
}

func TestRepeatedSTRKeyIndexLoops(t *testing.T) {
	starts := time.Unix(100, 0)
	now := starts.Add(250 * time.Millisecond)
	if got := strEffectKeyIndex(starts, now, 60, 12, true); got != 3 {
		t.Fatalf("repeated key index = %.2f, want 3", got)
	}
	if got := strEffectKeyIndex(starts, now, 60, 12, false); got != 15 {
		t.Fatalf("one-shot key index = %.2f, want 15", got)
	}
}

func TestWarpPortalSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9003, CreatorID: 2000000, UnitID: 128, X: 30, Y: 40, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 9003 || effect.effectID != effectPortal || effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9003})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestRathenaActiveWarpPortalSkillUnitEntryPersistsUntilDisappear(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9005, CreatorID: 2000000, UnitID: 129, X: 30, Y: 40, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	effect := mode.worldEffects[0]
	if effect.actorID != 9005 || effect.effectID != effectPortal || effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect = %+v", effect)
	}
	if effect.expires.Sub(effect.starts) < skillUnitEffectFallbackDuration {
		t.Fatalf("portal lifetime = %s, want skill unit fallback", effect.expires.Sub(effect.starts))
	}
	if effect.duration != 0 {
		t.Fatalf("portal animation override = %s, want native component timing", effect.duration)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9005})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestWarpPortalSkillUnitLookChangeKeepsPortalAtSameCell(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9006, CreatorID: 2000000, UnitID: 129, X: 30, Y: 40, Visible: true})
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectPortal {
		t.Fatalf("world effects before look change = %+v, want portal", mode.worldEffects)
	}

	if !mode.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 9006, Type: 0, Value: 128}) {
		t.Fatal("skill unit look change was not handled")
	}
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1: %+v", len(mode.worldEffects), mode.worldEffects)
	}
	effect := mode.worldEffects[0]
	if effect.actorID != 9006 || effect.effectID != effectPortal || effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect after look change = %+v, want portal on same unit cell", effect)
	}
}

func TestSkillUnitEntryDispatchesAllMappedUnitEffectArrays(t *testing.T) {
	const unitID uint16 = 65000
	skillUnitEffectSpecs[unitID] = skillUnitEffectSpec{effectIDs: []int{effectPneuma, effectSafetyWall}}
	defer delete(skillUnitEffectSpecs, unitID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9004, CreatorID: 2000000, UnitID: unitID, X: 123, Y: 456, Visible: true})

	want := []int{effectPneuma, effectSafetyWall}
	if len(mode.worldEffects) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.worldEffects), len(want), mode.worldEffects)
	}
	for i, wantEffectID := range want {
		effect := mode.worldEffects[i]
		if effect.actorID != 9004 || effect.effectID != wantEffectID || effect.x != 123 || effect.y != 456 {
			t.Fatalf("effect %d = %+v, want effect %d on unit", i, effect, wantEffectID)
		}
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9004})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestPneumaSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9002, CreatorID: 2000000, UnitID: 133, X: 123, Y: 456, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 9002 || effect.effectID != effectPneuma || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9002})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestSkillNoDamageNotifyAddsProvokeEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 6, Amount: 2, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 1100 || effect.effectID != effectProvoke || effect.x != 12 || effect.y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestSkillNoDamageNotifyAddsStealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 50, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 1100 || effect.effectID != effectSteal || effect.x != 12 || effect.y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestSkillNoDamageNotifyEndureUsesReadyFightAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 1}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Hair: 1}}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 8, TargetID: 2000000, SourceID: 2000000, Result: 1})
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != spriteActionPCReadyFight || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client READYFIGHT action", anim)
	}
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectEndure {
		t.Fatalf("world effects = %+v, want Endure effect", mode.worldEffects)
	}
}

func TestSkillNoDamageNotifyAddsHealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 28, Amount: 234, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 1100 || effect.effectID != effectHeal || effect.x != 12 || effect.y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source cast animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
}

func TestActorActionNotifyHealUsesCastAndOffensiveHealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1015,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 4, Hair: 1}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     28,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      8,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
	found := false
	for _, effect := range mode.worldEffects {
		if effect.effectID == effectHealOffensive && effect.actorID == 300 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("world effects = %+v, want offensive heal effect on target", mode.worldEffects)
	}
}

func TestActorActionNotifyBashUsesRobrowserWeaponAttackOverride(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1015,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Hair: 1, Weapon: 3}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     5,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != spriteActionPCAttack2 {
		t.Fatalf("source animation = %+v, want reference client weapon attack override", anim)
	}
}

func TestActorActionNotifyHealDoesNotOverwriteLocalCastWithHurt(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 4, Hair: 1}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     28,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    2000000,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
	if len(mode.worldEffects) != 2 || mode.worldEffects[0].effectID != effectHeal || mode.worldEffects[1].effectID != effectHealOffensive {
		t.Fatalf("world effects = %+v, want reference client heal effect followed by hit effect", mode.worldEffects)
	}
}

func TestSkillFailAckAddsConsoleErrorWithoutEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 6, Result: 0, Cause: 0})

	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects = %d, want 0", len(mode.worldEffects))
	}
	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Action failed." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestStealFailAckUsesSkillSpecificMessage(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 50, Result: 0, Cause: 0})

	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Steal failed." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestPickedInventoryItemAddsToExistingStack(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 512, Type: 0, Amount: 3, Identified: true}},
		},
	}

	addPickedSessionInventoryItem(sessionState, session.InventoryItem{Index: 7, ItemID: 512, Type: 3, Amount: 2})

	if got := sessionState.Inventory.Items[0].Amount; got != 5 {
		t.Fatalf("picked stack amount = %d, want 5", got)
	}
	if got := sessionState.Inventory.Items[0].Type; got != 0 {
		t.Fatalf("picked stack type = %d, want preserved healing type", got)
	}
}

func TestSessionItemFromNetworkMarksEquipmentByType(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      7,
		ItemID:     1201,
		Type:       5,
		Location:   0x0002,
		Identified: true,
		Amount:     1,
	})
	if !item.Equip {
		t.Fatalf("item = %+v, want equipment item", item)
	}
}

func TestSessionItemFromNetworkDefaultsAmmoLocation(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      8,
		ItemID:     1750,
		Type:       10,
		Identified: true,
		Amount:     100,
	})
	if !item.Equip || item.Location != db.EquipAmmo {
		t.Fatalf("ammo item = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestInventoryItemListReplacesDifferentItemAtReusedIndex(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{
				Index:    11,
				ItemID:   1201,
				Type:     5,
				Location: 0x0002,
				Amount:   1,
				Equip:    true,
			}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryItemList(ctx, []network.InventoryItem{{
		Index:  11,
		ItemID: 938,
		Type:   3,
		Amount: 2,
	}})

	item := sessionState.Inventory.Items[0]
	if item.Equip || item.Location != 0 || item.Type != 3 || item.ItemID != 938 {
		t.Fatalf("item = %+v, want clean replacement", item)
	}
}

func TestPickedEquipmentKeepsEquipMetadata(t *testing.T) {
	sessionState := &session.Session{}
	addPickedSessionInventoryItem(sessionState, session.InventoryItem{
		Index:      11,
		ItemID:     1201,
		Type:       5,
		Location:   0x0002,
		Identified: true,
		Amount:     1,
		Equip:      inventoryItemTypeIsEquipment(5),
	})
	if len(sessionState.Inventory.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(sessionState.Inventory.Items))
	}
	item := sessionState.Inventory.Items[0]
	if !item.Equip || item.Location != 0x0002 {
		t.Fatalf("picked item = %+v, want equipment metadata", item)
	}
}

func TestApplyInventoryEquipAckUpdatesEquippedState(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 1, ItemID: 1201, Type: 4, Location: 0x0002, Equip: true},
				{Index: 2, ItemID: 1202, Type: 4, Location: 0x0002, Equip: true, Equipped: true},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 1, Location: 0x0002, Success: true})
	if !sessionState.Inventory.Items[0].Equipped {
		t.Fatal("equipped item was not marked equipped")
	}
	if sessionState.Inventory.Items[1].Equipped {
		t.Fatal("previous item in same location stayed equipped")
	}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 1, Location: 0x0002, Success: true, Unequip: true})
	if sessionState.Inventory.Items[0].Equipped {
		t.Fatal("unequipped item stayed equipped")
	}
}

func TestApplyInventoryEquipAckDefaultsAmmoLocation(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 1750, Type: 10, Amount: 100, Equip: true},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 3, Success: true})

	item := sessionState.Inventory.Items[0]
	if !item.Equipped || item.Location != db.EquipAmmo {
		t.Fatalf("ammo item after equip ack = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestApplyEquippedArrowMarksAmmoSlot(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 1750, Type: 10, Amount: 100, Location: db.EquipAmmo, Equip: true, Equipped: true},
				{Index: 9, ItemID: 1751, Type: 10, Amount: 50},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyEquippedArrow(ctx, network.EquippedArrow{Index: 9})

	if sessionState.Inventory.Items[0].Equipped {
		t.Fatal("previous arrow stayed equipped")
	}
	item := sessionState.Inventory.Items[1]
	if !item.Equip || !item.Equipped || item.Location != db.EquipAmmo {
		t.Fatalf("arrow item after ZC_EQUIP_ARROW = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestInventoryEquipmentRebuildsLocalWeaponAppearanceFromEquippedItem(t *testing.T) {
	sessionState := &session.Session{
		CharID:   150000,
		Selected: session.Character{ID: 150000, Job: 2, Weapon: 10},
		Characters: []session.Character{
			{ID: 150000, Job: 2, Weapon: 10},
		},
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, Job: 2, Weapon: 10}
	ctx := client.Context{Session: sessionState, World: world}

	applyInventoryItemList(ctx, []network.InventoryItem{
		{Index: 2, ItemID: 1607, Type: 5, Location: 0x0002, Equip: true, Equipped: true, Identified: true},
	})

	if sessionState.Selected.Weapon != 10 || sessionState.Selected.Shield != 0 {
		t.Fatalf("selected weapon = %d shield = %d, want 10/0", sessionState.Selected.Weapon, sessionState.Selected.Shield)
	}
	if sessionState.Characters[0].Weapon != 10 {
		t.Fatalf("character list weapon = %d, want 10", sessionState.Characters[0].Weapon)
	}
	if world.Player.Weapon != 10 || world.Player.Shield != 0 {
		t.Fatalf("world player weapon = %d shield = %d, want 10/0", world.Player.Weapon, world.Player.Shield)
	}
}

func TestApplyStoragePacketsUpdateSessionStorage(t *testing.T) {
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	applyStorageAmount(ctx, network.StorageAmount{Amount: 1, MaxAmount: 300})
	applyStorageItemList(ctx, []network.InventoryItem{{Index: 3, ItemID: 512, Type: 0, Amount: 4, Identified: true}})
	if !sessionState.Storage.Open {
		t.Fatal("storage was not marked open")
	}
	if sessionState.Storage.Amount != 1 || sessionState.Storage.MaxAmount != 300 {
		t.Fatalf("storage counts = %d/%d", sessionState.Storage.Amount, sessionState.Storage.MaxAmount)
	}
	if len(sessionState.Storage.Items) != 1 || sessionState.Storage.Items[0].ItemID != 512 || sessionState.Storage.Items[0].Amount != 4 {
		t.Fatalf("storage items = %+v", sessionState.Storage.Items)
	}

	applyStorageItemAdded(ctx, network.InventoryItem{Index: 3, ItemID: 512, Type: 0, Amount: 7, Identified: true})
	if got := sessionState.Storage.Items[0].Amount; got != 7 {
		t.Fatalf("storage amount after replace = %d, want 7", got)
	}
	applyStorageItemRemoved(ctx, network.StorageItemRemoved{Index: 3, Amount: 2})
	if got := sessionState.Storage.Items[0].Amount; got != 5 {
		t.Fatalf("storage amount after remove = %d, want 5", got)
	}
	applyStorageClosed(ctx)
	if sessionState.Storage.Open || len(sessionState.Storage.Items) != 0 {
		t.Fatalf("storage after close = %+v", sessionState.Storage)
	}
}

func TestApplyCartPacketsUpdateSessionCart(t *testing.T) {
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	applyCartAmount(ctx, network.CartAmount{Amount: 1, MaxAmount: 100, Weight: 450, MaxWeight: 80000})
	applyCartItemList(ctx, []network.InventoryItem{{Index: 3, ItemID: 512, Type: 0, Amount: 4, Identified: true}})
	if !sessionState.Cart.Open {
		t.Fatal("cart was not marked open")
	}
	if sessionState.Cart.Amount != 1 || sessionState.Cart.MaxAmount != 100 || sessionState.Cart.Weight != 450 || sessionState.Cart.MaxWeight != 80000 {
		t.Fatalf("cart counts = %+v", sessionState.Cart)
	}
	if len(sessionState.Cart.Items) != 1 || sessionState.Cart.Items[0].ItemID != 512 || sessionState.Cart.Items[0].Amount != 4 {
		t.Fatalf("cart items = %+v", sessionState.Cart.Items)
	}

	applyCartItemAdded(ctx, network.InventoryItem{Index: 3, ItemID: 512, Type: 0, Amount: 7, Identified: true})
	if got := sessionState.Cart.Items[0].Amount; got != 7 {
		t.Fatalf("cart amount after replace = %d, want 7", got)
	}
	applyCartItemRemoved(ctx, network.CartItemRemoved{Index: 3, Amount: 2})
	if got := sessionState.Cart.Items[0].Amount; got != 5 {
		t.Fatalf("cart amount after remove = %d, want 5", got)
	}
	applyCartClosed(ctx)
	if sessionState.Cart.Open {
		t.Fatalf("cart after close = %+v", sessionState.Cart)
	}
}

func TestAttackFocusTracksTargetAndAnimationStart(t *testing.T) {
	mode := &WorldMode{}
	first := time.Unix(10, 0)
	second := time.Unix(20, 0)

	mode.focusAttackTarget(100, first)
	if mode.attackFocusID != 100 || !mode.attackFocusStart.Equal(first) {
		t.Fatalf("first focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}

	mode.focusAttackTarget(100, second)
	if !mode.attackFocusStart.Equal(first) {
		t.Fatalf("same target reset animation start to %v", mode.attackFocusStart)
	}

	mode.focusAttackTarget(200, second)
	if mode.attackFocusID != 200 || !mode.attackFocusStart.Equal(second) {
		t.Fatalf("second focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}

	mode.clearAttackFocus()
	if mode.attackFocusID != 0 || !mode.attackFocusStart.IsZero() {
		t.Fatalf("clear focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}
}
