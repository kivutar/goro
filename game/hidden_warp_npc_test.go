package game

import (
	"fmt"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestHiddenWarpNPCIsNotVisibleOrSelectable(t *testing.T) {
	for _, name := range []string{"", "boom1#airplane"} {
		for _, hasObjectType := range []bool{false, true} {
			t.Run(fmt.Sprintf("name=%q/object-type=%t", name, hasObjectType), func(t *testing.T) {
				actor := worldstate.Actor{ID: 300, Job: 139, Name: name, X: 11, Y: 20, ObjectType: actorObjectTypeNPC, HasObjectType: hasObjectType}
				ctx, projection := cursorHoverTestContext(actor)
				ctx.Resources = &res.Manager{}
				mode := &WorldMode{}
				now := time.Now()
				for _, entry := range mode.collectSceneActorEntries(render.NewFrame(800, 600), ctx, projection) {
					if entry.actor.ID == actor.ID {
						t.Error("hidden warp NPC entered the actor draw list")
					}
				}
				if actorCastsShadow(actor) {
					t.Error("hidden warp NPC casts a shadow")
				}
				if got := actorDisplayName(ctx, actor, false); got != "" {
					t.Errorf("display name = %q, want empty", got)
				}
				if labels := mode.hoveredActorDisplayLabels(ctx, actor, now); len(labels) != 0 {
					t.Errorf("hover labels = %v, want none", labels)
				}
				mouseX, mouseY := ctx.Input.MouseX, ctx.Input.MouseY
				if got, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, nil); ok {
					t.Errorf("hovered hidden NPC: %+v", got)
				}
				if _, ok := clickedTalkTarget(ctx, projection, mouseX, mouseY, now, nil); ok || cursorActorCanTalk(actor) {
					t.Error("hidden warp NPC can be talked to")
				}
				ctx.Session.NoShift = true
				if actorCanBeSkillTargeted(ctx, session.Skill{ID: 13, Type: skillTargetEnemy}, actor) || actorCanBeAttackClicked(ctx, actor) {
					t.Error("hidden warp NPC can be targeted")
				}
				if mode.scriptHighlightActor(ctx, actor.ID) {
					t.Error("hidden warp NPC can be highlighted by a script")
				}
			})
		}
	}
}

func TestHiddenWarpNPCRetainsAirshipExplosionAnchor(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Session: &session.Session{AccountID: 100, CharID: 200}}
	entry := network.ActorEntry{ID: 300, Job: 139, X: 239, Y: 62, ObjectType: actorObjectTypeNPC, HasObjectType: true, Appearance: true}
	mode := &WorldMode{}
	mode.upsertNetworkActor(ctx, entry)
	actor, ok := ctx.World.Actors[entry.ID]
	if !ok || actor.Job != entry.Job || actor.X != entry.X || actor.Y != entry.Y {
		t.Fatalf("hidden effect anchor was not retained: %+v, exists=%t", actor, ok)
	}
	mode.applyWarpPortalEntry(ctx, entry)
	if len(mode.worldEffects) != 0 {
		t.Fatal("hidden warp NPC generated a visible warp portal")
	}
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: entry.ID, EffectID: effectSuiExplosion})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want one explosion", len(mode.worldEffects))
	}
	effect := mode.worldEffects[0]
	if effect.effectID != effectSuiExplosion || effect.actorID != entry.ID || effect.x != entry.X || effect.y != entry.Y {
		t.Fatalf("explosion lost its hidden NPC anchor: %+v", effect)
	}
}

func TestHiddenNPCRemainsInteractive(t *testing.T) {
	// Unlike class 139, the original client gives class 111 a pick rectangle.
	actor := worldstate.Actor{ID: 300, Job: 111, X: 11, Y: 20, ObjectType: actorObjectTypeNPC, HasObjectType: true}
	ctx, projection := cursorHoverTestContext(actor)
	got, ok := clickedTalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, time.Now(), nil)
	if !ok || got.ID != actor.ID {
		t.Fatalf("interactive hidden NPC was not selectable: %+v, ok=%t", got, ok)
	}
}

func TestHiddenWarpNPCDoesNotLoadPoringRealData(t *testing.T) {
	manager := realDataManager(t)
	if view, status := loadNonPCSpriteView(manager, 139, "hidden warp NPC"); view != nil {
		t.Fatalf("hidden warp NPC loaded a sprite: %s", status)
	}
	if view, status := loadNonPCSpriteView(manager, 1002, "Poring"); view == nil {
		t.Fatalf("normal Poring failed to load: %s", status)
	}
}
