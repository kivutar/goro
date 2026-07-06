package game

import (
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
		StatusID: statusEffectHiding,
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
		StatusID: statusEffectHiding,
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
		t.Fatalf("camera billboard scale = %.3f, want about 1.04 at roBrowser default zoom", scale)
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
			{Index: 1, ItemID: 1701, Location: equipLocationWeapon, Equip: true, Equipped: true},
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

func TestWorldModeParameterChangeRecoveryFeedback(t *testing.T) {
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

	if sessionState.Vitals.HP != 85 {
		t.Fatalf("hp = %d, want 85", sessionState.Vitals.HP)
	}
	if len(mode.damageFloaters) != 1 {
		t.Fatalf("floaters = %d, want 1", len(mode.damageFloaters))
	}
	if mode.damageFloaters[0].text != "15" || mode.damageFloaters[0].kind != damageFloaterRecoveryHP {
		t.Fatalf("floater = %+v", mode.damageFloaters[0])
	}
	if len(mode.scheduledSounds) != 1 || mode.scheduledSounds[0].paths[0] != recoveryHPSFX {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
}

func TestApplyRecoveryUpdatesHPAndAddsBlueFloater(t *testing.T) {
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
	if got := mode.scheduledSounds[0].paths; len(got) < 2 || got[0] != recoveryHPSFX || got[1] != recoverySFXFallbacks[0] {
		t.Fatalf("scheduled sound paths = %v, want %q then fallback %q", got, recoveryHPSFX, recoverySFXFallbacks[0])
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

func TestSetSelectedCharacterSeedsInventoryZeny(t *testing.T) {
	sessionState := &session.Session{}

	setSelectedCharacter(sessionState, session.Character{ID: 1234, Money: 95000})

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
	progress := sessionProgressFromCharacter(session.Character{Level: 12, JobLevel: 7})
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
	if world.Dir != worldstate.DirectionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
	}
	if len(mode.damageFloaters) != 1 || !mode.damageFloaters[0].starts.Equal(targetAnim.started) {
		t.Fatalf("damage floater = %+v targetStarted=%s", mode.damageFloaters, targetAnim.started)
	}
	life, ok := mode.actorLife[300]
	if !ok {
		t.Fatal("target estimated life fallback missing")
	}
	if life.hp != 8 || life.maxHP != 50 || !life.estimated {
		t.Fatalf("target estimated life = %+v, want 8/50 estimated", life)
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
	if coverage.Implemented != 73 {
		t.Fatalf("implemented effects = %d, want 73", coverage.Implemented)
	}
	if coverage.RobrowserActive != 607 || coverage.RobrowserAll != 1147 {
		t.Fatalf("roBrowser totals = active %d all %d", coverage.RobrowserActive, coverage.RobrowserAll)
	}
	if coverage.ActivePercent < 12.0 || coverage.ActivePercent > 12.1 {
		t.Fatalf("active coverage = %.3f, want about 12.0", coverage.ActivePercent)
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
	if component.posX != 0.1 || component.posZ != 0.8 || component.sizeStart != roBrowserEffectSize(100) || component.angleStart != 270 || !component.rotateToTarget {
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
	if component.alphaMax > 0.25 || component.sizeEnd > roBrowserEffectSize(120) {
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
	got := resolveEffectSTRFile(component, effect)
	if got != "firehit1" && got != "firehit2" && got != "firehit3" {
		t.Fatalf("resolved STR file = %q, want firehit1..3", got)
	}
	if again := resolveEffectSTRFile(component, effect); again != got {
		t.Fatalf("resolved STR file changed from %q to %q", got, again)
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

func TestMageSkillEffectMappings(t *testing.T) {
	if !skillForcesPassive(9) {
		t.Fatal("MG_SRECOVERY should be passive")
	}
	expectEffectIDs(t, "MG_SIGHT success", skillSuccessEffectIDs(10))
	expectEffectIDs(t, "MG_SIGHT immediate", skillEffectIDs(10))
	if !skillForcesSelfTarget(10) {
		t.Fatal("MG_SIGHT should force self-targeting")
	}
	expectEffectIDs(t, "MG_NAPALMBEAT hit", skillHitEffectIDs(11), effectBashHit)
	expectEffectIDs(t, "MG_SAFETYWALL ground", skillGroundEffectIDs(12))
	if !skillForcesGroundTarget(12) {
		t.Fatal("MG_SAFETYWALL should force ground targeting")
	}
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
	if !skillForcesGroundTarget(18) {
		t.Fatal("MG_FIREWALL should force ground targeting")
	}
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
	if component.sizeStartX != 100*roBrowserEffectPixelRatio || component.sizeStartY != 50*roBrowserEffectPixelRatio {
		t.Fatalf("fire bolt size = %.3f x %.3f", component.sizeStartX, component.sizeStartY)
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
		t.Fatalf("components = %d, want 8 roBrowser lens slashes", len(spec.components))
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
		if component.sizeStartXRandMin != 25*roBrowserEffectPixelRatio || component.sizeStartXRandMax != 40*roBrowserEffectPixelRatio {
			t.Fatalf("component %d start x range = %.3f..%.3f", i, component.sizeStartXRandMin, component.sizeStartXRandMax)
		}
		if component.sizeStartY != 10*roBrowserEffectPixelRatio || component.sizeEndX != 1*roBrowserEffectPixelRatio {
			t.Fatalf("component %d fixed axis sizes = %.3f %.3f", i, component.sizeStartY, component.sizeEndX)
		}
		if component.sizeEndYRandMin != 250*roBrowserEffectPixelRatio || component.sizeEndYRandMax != 300*roBrowserEffectPixelRatio {
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
	if projectile.posZ != 20 || projectile.posZEnd != 0.0001 || projectile.posXStartMiddle != 5 || projectile.posYStartMiddle != 2 || projectile.sizeStart != 50*roBrowserEffectPixelRatio {
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
	if len(spec.components) < 2 {
		t.Fatalf("components = %d, want orbit sprite", len(spec.components))
	}
	component := spec.components[1]
	if component.spriteFile != "sight" || component.duplicate != 10 || component.orbitRadiusX != 3 || component.orbitRadiusY != 3 || component.orbitRotations != 10 {
		t.Fatalf("sight orbit component = %+v", component)
	}
	if component.sizeStart != 60*roBrowserEffectPixelRatio || component.sizeEnd != 80*roBrowserEffectPixelRatio {
		t.Fatalf("sight orbit size = %.3f -> %.3f", component.sizeStart, component.sizeEnd)
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

func TestFireBallSpriteRotationUsesProjectedTrajectory(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{ID: 300, X: 12, Y: 20})
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}
	spec, ok := worldEffectSpecForID(effectFireBall)
	if !ok || len(spec.components) == 0 {
		t.Fatal("fire ball effect missing")
	}
	component := spec.components[0]
	effect := worldEffect{effectID: effectFireBall, actorID: 300, targetID: 2000000}

	startX, _, _, endX, _, _, ok := effectTrajectoryEndpoints(ctx, component, effect)
	if !ok {
		t.Fatal("trajectory endpoints missing")
	}
	if startX >= endX {
		t.Fatalf("trajectory start/end = %.2f -> %.2f, want caster-to-target direction", startX, endX)
	}

	projection := newSceneProjectionForTargetYaw(800, 600, 11, 20, 0, 0)
	angle, ok := effectSpriteScreenRotation(ctx, projection, component, effect)
	if !ok {
		t.Fatal("rotation missing")
	}
	start := projection.Project(startX, 20.5, terrainHeightAt(world, 10, 20)+0.07+component.posZ)
	end := projection.Project(endX, 20.5, terrainHeightAt(world, 12, 20)+0.07+component.posZ)
	want := math.Atan2(float64(end.y-start.y), float64(end.x-start.x)) - math.Pi/2
	if math.Abs(angle-want) > 0.001 {
		t.Fatalf("angle = %.3f, want %.3f", angle, want)
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
	if len(mode.damageFloaters) != 4 {
		t.Fatalf("damage floaters = %d, want 4", len(mode.damageFloaters))
	}
	for i, floater := range mode.damageFloaters {
		if floater.text != "252" {
			t.Fatalf("floater %d text = %q, want 252", i, floater.text)
		}
	}
}

func TestActorActionNotifyDispatchesAllMappedCombatEffectArrays(t *testing.T) {
	const skillID uint16 = 65001
	roBrowserSkillEffects[skillID] = roBrowserSkillEffect{
		effectIDs:              []int{effectHeal, effectBlessing},
		effectIDsOnCaster:      []int{effectEndure},
		beforeHitEffectIDs:     []int{effectSoulStrike, effectFireBolt},
		beforeHitEffectIDsSelf: []int{effectBashBegin},
		hitEffectIDs:           []int{effectFireHit, effectWindHit},
		hitEffectIDsOnCaster:   []int{effectIncAgility},
	}
	defer delete(roBrowserSkillEffects, skillID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 300, X: 11, Y: 20, Job: 1002, ObjectType: actorObjectTypeMob, HasObjectType: true})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     skillID,
		SkillLevel:  1,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      100,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	want := []struct {
		effectID int
		actorID  uint32
	}{
		{effectSoulStrike, 300},
		{effectFireBolt, 300},
		{effectBashBegin, 2000000},
		{effectHeal, 300},
		{effectBlessing, 300},
		{effectEndure, 2000000},
		{effectFireHit, 300},
		{effectWindHit, 300},
		{effectIncAgility, 2000000},
	}
	if len(mode.worldEffects) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.worldEffects), len(want), mode.worldEffects)
	}
	for i, wantEffect := range want {
		got := mode.worldEffects[i]
		if got.effectID != wantEffect.effectID || got.actorID != wantEffect.actorID {
			t.Fatalf("effect %d = %+v, want %+v", i, got, wantEffect)
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

func TestSkillCastFallbackMappings(t *testing.T) {
	tests := []struct {
		name         string
		skillID      uint16
		level        uint16
		wantProperty uint32
		wantDuration time.Duration
	}{
		{name: "soul strike", skillID: 13, level: 5, wantProperty: 8, wantDuration: 2500 * time.Millisecond},
		{name: "cold bolt", skillID: 14, level: 4, wantProperty: 1, wantDuration: 2800 * time.Millisecond},
		{name: "fire ball", skillID: 17, level: 3, wantProperty: 3, wantDuration: time.Second},
		{name: "fire bolt", skillID: 19, level: 4, wantProperty: 3, wantDuration: 2800 * time.Millisecond},
		{name: "lightning bolt", skillID: 20, level: 4, wantProperty: 4, wantDuration: 2800 * time.Millisecond},
		{name: "thunder storm", skillID: 21, level: 4, wantProperty: 4, wantDuration: 1800 * time.Millisecond},
		{name: "warp portal", skillID: 27, level: 4, wantProperty: 0, wantDuration: time.Second},
		{name: "increase agi", skillID: 29, level: 10, wantProperty: 0, wantDuration: time.Second},
		{name: "decrease agi", skillID: 30, level: 10, wantProperty: 0, wantDuration: time.Second},
		{name: "aqua benedicta", skillID: 31, level: 1, wantProperty: 0, wantDuration: time.Second},
		{name: "signum crucis", skillID: 32, level: 10, wantProperty: 0, wantDuration: 500 * time.Millisecond},
		{name: "angelus", skillID: 33, level: 10, wantProperty: 0, wantDuration: 500 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			property, duration := skillCastFallback(tt.skillID, tt.level)
			if property != tt.wantProperty || duration != tt.wantDuration {
				t.Fatalf("skillCastFallback = property %d duration %s, want property %d duration %s", property, duration, tt.wantProperty, tt.wantDuration)
			}
		})
	}

	if property, duration := skillCastFallback(5, 10); property != 0 || duration != 0 {
		t.Fatalf("non-cast skill fallback = property %d duration %s, want zero", property, duration)
	}
}

func TestSkillVisualMetadataMappings(t *testing.T) {
	if skillAction(5) != roBrowserSkillActionAttack || skillAction(7) != roBrowserSkillActionAttack {
		t.Fatalf("swordman weapon-action skills = bash:%d magnum:%d", skillAction(5), skillAction(7))
	}
	if skillAction(8) != roBrowserSkillActionReadyFight {
		t.Fatalf("endure action = %d, want ready fight", skillAction(8))
	}
	if skillAction(28) != roBrowserSkillActionDefault {
		t.Fatalf("heal action = %d, want default skill action", skillAction(28))
	}
	if !skillForcesGroundTarget(21) || !skillForcesGroundTarget(25) {
		t.Fatalf("ground target overrides = thunderstorm:%t pneuma:%t", skillForcesGroundTarget(21), skillForcesGroundTarget(25))
	}
	if size := skillCastGroundSampleSize(21); size != 5 {
		t.Fatalf("thunderstorm marker size = %.1f, want 5", size)
	}
	if size := skillCastGroundSampleSize(19); size != 1 {
		t.Fatalf("firebolt marker size = %.1f, want default 1", size)
	}
	recovery := skillRecoveryFloater(28)
	if !recovery.enabled || recovery.kind != damageFloaterRecoveryHP || recovery.color != recoveryHPColor {
		t.Fatalf("heal recovery floater = %+v", recovery)
	}
}

func TestSkillCastNotifyAddsDurationAura(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: 20, Property: 4, DelayTime: 2500})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.worldEffects))
	}
	circle := mode.worldEffects[0]
	if circle.effectID != effectCastRing || circle.actorID != 2000000 || circle.targetID != 0 || circle.duration != 2500*time.Millisecond {
		t.Fatalf("circle = %+v", circle)
	}
	aura := mode.worldEffects[1]
	if aura.effectID != effectBeginSpell4 || aura.actorID != 2000000 || aura.targetID != 1100 || aura.duration != 2500*time.Millisecond {
		t.Fatalf("aura = %+v", aura)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("cast animation missing")
	}
	if anim.actionFamily != spriteActionPCReadyFight || anim.duration != 2500*time.Millisecond || anim.hasFixedMotion {
		t.Fatalf("cast animation = %+v", anim)
	}
	if world.Dir != worldstate.DirectionFromDelta(10, 20, 12, 20, 4) {
		t.Fatalf("cast dir = %d", world.Dir)
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

func TestLocalGroundSkillCastFallbackFacesCellAndStartsCastAnimation(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, X: 10, Y: 20, Dir: 4}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 4, Hair: 1}},
		World:   world,
	}

	start := time.Now()
	mode.addLocalSkillCastFallback(ctx, 21, 4, 2000000, 0, 12, 20, 1800*time.Millisecond, start, "local-ground")

	if want := directionFromDelta(10, 20, 12, 20, 4); world.Dir != want || world.Player.Dir != want {
		t.Fatalf("local cast dir = world:%d player:%d, want %d", world.Dir, world.Player.Dir, want)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("local ground cast animation missing")
	}
	if anim.actionFamily != spriteActionPCReadyFight || anim.duration != 1800*time.Millisecond || anim.hasFixedMotion {
		t.Fatalf("local ground cast animation = %+v", anim)
	}
	if len(mode.worldEffects) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.worldEffects))
	}
	if mode.worldEffects[0].effectID != effectGroundSample || mode.worldEffects[0].x != 12 || mode.worldEffects[0].y != 20 {
		t.Fatalf("ground marker = %+v", mode.worldEffects[0])
	}
	if mode.worldEffects[1].effectID != effectCastRing || mode.worldEffects[1].actorID != 2000000 {
		t.Fatalf("cast circle = %+v", mode.worldEffects[1])
	}
	if mode.worldEffects[2].effectID != effectBeginSpell4 || mode.worldEffects[2].actorID != 2000000 {
		t.Fatalf("cast aura = %+v", mode.worldEffects[2])
	}
}

func TestAcolyteSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AL_DP passive", skillEffectIDs(22))
	expectEffectIDs(t, "AL_DEMONBANE passive", skillEffectIDs(23))
	expectEffectIDs(t, "AL_RUWACH", skillEffectIDs(24), effectRuwach)
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

func TestArcherThiefMerchantSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AC_CONCENTRATION", skillEffectIDs(45), effectConcentration)
	expectEffectIDs(t, "AC_DOUBLE begin", skillBeginEffectIDs(46), effectBashBegin)
	expectEffectIDs(t, "AC_DOUBLE before-hit", skillBeforeHitEffectIDs(46), effectArrowShot)
	expectEffectIDs(t, "AC_DOUBLE hit", skillHitEffectIDs(46), effectBashHit)
	expectEffectIDs(t, "AC_SHOWER", skillEffectIDs(47), effectArrowShower)
	expectEffectIDs(t, "AC_SHOWER hit", skillHitEffectIDs(47), effectBashHit)
	expectEffectIDs(t, "TF_DOUBLE passive", skillEffectIDs(48))
	expectEffectIDs(t, "TF_MISS passive", skillEffectIDs(49))
	expectEffectIDs(t, "TF_STEAL success", skillSuccessEffectIDs(50), effectSteal)
	expectEffectIDs(t, "TF_HIDING", skillEffectIDs(51))
	expectEffectIDs(t, "TF_POISON hit", skillHitEffectIDs(52), effectPoisonAttack)
	expectEffectIDs(t, "TF_DETOXIFY", skillEffectIDs(53), effectDetoxication)
	expectEffectIDs(t, "TF_SPRINKLESAND", skillEffectIDs(149), effectSprinkleSand)
	expectEffectIDs(t, "TF_BACKSLIDING", skillEffectIDs(150))
	expectEffectIDs(t, "TF_PICKSTONE", skillEffectIDs(151))
	expectEffectIDs(t, "TF_THROWSTONE before-hit", skillBeforeHitEffectIDs(152), effectThrowItem3)
	expectEffectIDs(t, "MC_MAMMONITE", skillEffectIDs(42), effectMammonite)
}

func TestThiefSkillTargetRules(t *testing.T) {
	if !skillForcesPassive(48) {
		t.Fatal("TF_DOUBLE should be passive")
	}
	if !skillForcesPassive(49) {
		t.Fatal("TF_MISS should be passive")
	}
	if !isSelfTargetSkill(session.Skill{ID: 51, Level: 1, Type: skillTargetEnemy, Range: 1}) {
		t.Fatal("TF_HIDING should self-cast even when the skill list reports a range")
	}
	if skillAction(149) != roBrowserSkillActionAttack {
		t.Fatal("TF_SPRINKLESAND should use attack action")
	}
	if skillAction(152) != roBrowserSkillActionAttack {
		t.Fatal("TF_THROWSTONE should use attack action")
	}
}

func TestThiefThrowStoneEffectFollowsRoBrowserTable(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectThrowItem3)
	if !ok || len(spec.components) != 1 {
		t.Fatalf("throw stone spec = %#v ok=%t, want one component", spec, ok)
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "\xc0\xaf\xc0\xfa\xc0\xce\xc5\xcd\xc6\xe4\xc0\xcc\xbd\xba/item/\xb5\xb9.bmp" {
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
	world.Actors[300] = worldstate.Actor{ID: 300, X: 15, Y: 20, Job: 1002, ObjectType: 5}
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

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want arrow projectile", len(mode.worldEffects))
	}
	effect := mode.worldEffects[0]
	if effect.effectID != effectArrowShot || effect.actorID != 300 || effect.targetID != 200 {
		t.Fatalf("normal bow projectile = %+v", effect)
	}
}

func TestWarpEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AL_TELEPORT begin", skillBeginEffectIDs(26))
	expectEffectIDs(t, "Butterfly Wing item", itemUseEffectIDs(602), effectTeleportation)
	expectEffectIDs(t, "Fly Wing item", itemUseEffectIDs(601))
}

func TestTeleportSkillIsSelfTargetDespiteAttackRange(t *testing.T) {
	if !isSelfTargetSkill(session.Skill{ID: 26, Level: 2, Type: skillTargetEnemy, Range: 1}) {
		t.Fatal("AL_TELEPORT should self-cast even when the skill list reports attack range")
	}
	if !isGroundTargetSkill(session.Skill{ID: 27, Level: 4, Type: skillTargetEnemy, Range: 9}) {
		t.Fatal("AL_WARP should ground-target even when the skill list reports attack range")
	}
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
	if !mode.teleportModal.IsOpen() {
		t.Fatal("warp portal list should open the destination modal")
	}
	if mode.teleportModal.Title() != "Warp Portal" {
		t.Fatalf("modal title = %q", mode.teleportModal.Title())
	}
}

func TestSkillUnitEffectMappings(t *testing.T) {
	expectEffectIDs(t, "UNT_SAFETYWALL", skillUnitEffectIDs(126), effectSafetyWall)
	expectEffectIDs(t, "UNT_FIREWALL", skillUnitEffectIDs(127), effectFireWall)
	expectEffectIDs(t, "UNT_WARPPORTAL", skillUnitEffectIDs(128), effectPortal)
	expectEffectIDs(t, "rAthena UNT_WARP_ACTIVE", skillUnitEffectIDs(129), effectPortal)
	expectEffectIDs(t, "UNT_PNEUMA", skillUnitEffectIDs(133), effectPneuma)
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
	if component.posZ != 2 || component.sizeStart != 200*roBrowserEffectPixelRatio || component.sizeEnd != 70*roBrowserEffectPixelRatio {
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
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height {
			t.Fatalf("component %d size = %.1f %.1f %.1f, want %.1f %.1f %.1f", i, component.bottomSize, component.topSize, component.height, want.bottom, want.top, want.height)
		}
		if component.fixedPerspective {
			t.Fatalf("component %d uses fixed perspective, want world-space cylinder", i)
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
		t.Fatalf("first portal repeat = %t delay=%s, want roBrowser repeat -300ms", first.repeat, first.repeatDelay)
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
	if firstParticle.sizeStart != 9*roBrowserEffectPixelRatio || firstParticle.sizeEnd != 9*roBrowserEffectPixelRatio || firstParticle.sizeRand != 2*roBrowserEffectPixelRatio {
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
	if secondParticle.sizeStart != 9*roBrowserEffectPixelRatio || secondParticle.sizeEnd != 9*roBrowserEffectPixelRatio || secondParticle.sizeRand != 2*roBrowserEffectPixelRatio {
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
	if secondParticle.sizeStart != 9*roBrowserEffectPixelRatio || secondParticle.sizeEnd != 9*roBrowserEffectPixelRatio || secondParticle.sizeRand != 2*roBrowserEffectPixelRatio {
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
		if component.sizeStartX != 2.5*roBrowserEffectPixelRatio || component.sizeEndX != 2.5*roBrowserEffectPixelRatio {
			t.Fatalf("particle %d x size = %+v", tc.index, component)
		}
		if component.sizeStartY != 0 || component.sizeEndY != 0 || component.sizeRandY != 15*roBrowserEffectPixelRatio || component.sizeRandYMiddle != 45*roBrowserEffectPixelRatio {
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
	if overlay.sizeStart != 100*roBrowserEffectPixelRatio || overlay.sizeEnd != 100*roBrowserEffectPixelRatio || overlay.sizeStartY != 45*roBrowserEffectPixelRatio || overlay.sizeEndY != 45*roBrowserEffectPixelRatio || !overlay.sizeSmooth {
		t.Fatalf("overlay size = %+v", overlay)
	}
	if !overlay.overlay {
		t.Fatal("overlay should use roBrowser overlay rendering")
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
	if spec.duration != 4800*time.Millisecond {
		t.Fatalf("duration = %s, want 4800ms", spec.duration)
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
	if particle.sizeStartX != roBrowserEffectSize(2.5) || particle.sizeEndX != roBrowserEffectSize(2.5) {
		t.Fatalf("particle x size = %+v", particle)
	}
	if particle.sizeStartY != 0 || particle.sizeEndY != 0 || particle.sizeRandY != roBrowserEffectSize(15) || particle.sizeRandYMiddle != roBrowserEffectSize(45) {
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
	if overlay.sizeStart != roBrowserEffectSize(100) || overlay.sizeEnd != roBrowserEffectSize(100) || overlay.sizeStartY != roBrowserEffectSize(45) || overlay.sizeEndY != roBrowserEffectSize(45) || !overlay.sizeSmooth {
		t.Fatalf("overlay size = %+v", overlay)
	}
	if overlay.overlay {
		t.Fatal("overlay should use regular roBrowser 3D rendering")
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
	if sprite.kind != effectComponentSPR || sprite.spriteFile != "\xC3\xE0\xBA\xB9" {
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
		if component.sizeStart != 50*roBrowserEffectPixelRatio || component.sizeEnd != 50*roBrowserEffectPixelRatio {
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
	if aura.sizeStart != 140*roBrowserEffectPixelRatio || aura.sizeEnd != 140*roBrowserEffectPixelRatio {
		t.Fatalf("aura size = %+v", aura)
	}
}

func TestWorldEffectDuplicateDeltasMatchRobrowserSemantics(t *testing.T) {
	component := worldEffectComponent{
		alphaMax:      0.2,
		alphaMaxDelta: 0.2,
		sizeStart:     100 * roBrowserEffectPixelRatio,
		sizeEnd:       100 * roBrowserEffectPixelRatio,
		sizeDelta:     -10,
	}
	if got := effectBillboardAlphaForDuplicate(0.5, component, 2); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("duplicate alpha = %.3f, want 0.6", got)
	}
	sizeX, sizeY := effect3DSize(component, worldEffect{}, 0, 0.5, 2)
	want := 80 * roBrowserEffectPixelRatio
	if math.Abs(sizeX-want) > 0.001 || math.Abs(sizeY-want) > 0.001 {
		t.Fatalf("duplicate size = %.3f x %.3f, want %.3f", sizeX, sizeY, want)
	}
}

func TestEffect3DSpriteScaleUsesRobrowserSpriteUnits(t *testing.T) {
	size := roBrowserEffectSize(80)
	if got := effect3DSpriteScale(size); math.Abs(got-size) > 0.001 {
		t.Fatalf("sprite scale = %.3f, want %.3f", got, size)
	}
}

func TestEffect3DSpriteDrawOptionsHonorAdditiveBlend(t *testing.T) {
	if got := effect3DSpriteDrawOptions(worldEffectComponent{}).Blend; got != render.BlendSourceOver {
		t.Fatalf("default sprite effect blend = %v, want source-over", got)
	}
	if got := effect3DSpriteDrawOptions(worldEffectComponent{blendAdditive: true}).Blend; got != render.BlendLighter {
		t.Fatalf("additive sprite effect blend = %v, want lighter", got)
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

func TestSTRAnimationBlendMatchesRobrowserD3DBlend(t *testing.T) {
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 7}); got != render.BlendLighter {
		t.Fatalf("SRC_ALPHA/DST_ALPHA blend = %v, want BlendLighter", got)
	}
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 2}); got != render.BlendLighter {
		t.Fatalf("SRC_ALPHA/ONE blend = %v, want BlendLighter", got)
	}
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6}); got != render.BlendSourceOver {
		t.Fatalf("regular STR blend = %v, want BlendSourceOver", got)
	}
}

func TestWorldEffectSpecsMatchRobrowserRenderableSubset(t *testing.T) {
	source, err := os.ReadFile("/home/kivutar/src/robr/src/DB/Effects/EffectTable.js")
	if os.IsNotExist(err) {
		t.Skip("roBrowser checkout not available")
	}
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseRobrowserEffectTableSubset(string(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, effectID := range []int{
		effectMammonite,
		effectSoulStrike,
		effectSteal,
		effectPoisonAttack,
		effectDetoxication,
		effectStoneCurse,
		effectFireBall,
		effectFireWall,
		effectFrostDiverHit,
		effectLightningBolt,
		effectThunderStorm,
		effectRuwach,
		effectDecAgility,
		effectAqua,
		effectSignum,
		effectAngelus,
		effectBlessing,
		effectFireHit,
		effectFireSplashHit,
		effectConcentration,
		effectCure,
		effectRefineOK,
		effectRefineFail,
		effectJobLevelUp,
		effectTeleportation,
		effectPharmacyOK,
		effectPharmacyFail,
		effectHeal,
		effectHealOffensive,
		effectPortal,
	} {
		got, ok := worldEffectSpecForID(effectID)
		if !ok {
			t.Fatalf("world effect %d missing", effectID)
		}
		want, ok := parsed[effectID]
		if !ok {
			t.Fatalf("roBrowser effect %d missing", effectID)
		}
		if !reflect.DeepEqual(roBrowserRenderableWorldEffectSpec(got), want) {
			t.Fatalf("effect %d\n got: %#v\nwant: %#v", effectID, got, want)
		}
	}
}

func roBrowserRenderableWorldEffectSpec(spec worldEffectSpec) worldEffectSpec {
	spec.cameraShake = 0
	spec.detachLocalActor = false
	return spec
}

func TestLevelUpEffectSpecsUseSTRResources(t *testing.T) {
	base, ok := worldEffectSpecForID(effectBaseLevelUp)
	if !ok {
		t.Fatal("base level-up effect spec missing")
	}
	if len(base.components) != 1 || base.components[0].kind != effectComponentSTR || base.components[0].strFile != "angel" {
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
	if mode.pickupReqItemID != 0 {
		t.Fatalf("pickup request item id = %d, want cleared", mode.pickupReqItemID)
	}
	if world.Dir != worldstate.DirectionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
	}
}

func TestApplyActorPickupActionNotifyStartsPickupInsteadOfAttack(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertItem(worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20, Amount: 1})
	mode := &WorldMode{actorAnims: make(map[uint32]actorAnimation)}
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
	if len(mode.damageFloaters) != 0 {
		t.Fatalf("pickup notify should not create damage floaters: %+v", mode.damageFloaters)
	}
	if world.Dir != worldstate.DirectionFromDelta(10, 20, 11, 20, 4) {
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
	if life.hp != 12 || life.maxHP != 48 || life.fromTiny {
		t.Fatalf("life = %+v, want exact 12/48", life)
	}
}

func TestCombatLifeFallbackDoesNotSubtractRawDamageFromTinyHPGauge(t *testing.T) {
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

	mode.applyActorHPUpdate(network.ActorHPUpdate{ID: 300, HP: 95, MaxHP: 100, Tiny: true})
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	life, ok := mode.actorLife[300]
	if !ok {
		t.Fatal("life missing")
	}
	if life.hp != 95 || life.maxHP != 100 || !life.fromTiny {
		t.Fatalf("tiny life = %+v, want unchanged 95/100", life)
	}
}

func TestCombatLifeFallbackUsesEstimatedRedPlantMaxHPWithoutHPPacket(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		Job:           1078,
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
		Damage:      1,
		Action:      0,
	})

	life, ok := mode.monsterLifeForSense(world.Actors[300].ID)
	if !ok {
		t.Fatal("red plant estimated life missing")
	}
	if life.hp != 9 || life.maxHP != 10 || !life.estimated {
		t.Fatalf("red plant estimated life = %+v, want 9/10 estimated", life)
	}
}

func TestCombatLifeFallbackSubtractsRawDamageFromExactHPGauge(t *testing.T) {
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
	if life.hp != 38 || life.maxHP != 100 || life.fromTiny {
		t.Fatalf("exact life = %+v, want 38/100", life)
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
		nonPCViews: map[int]*playerSpriteView{
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
		nonPCViews: map[int]*playerSpriteView{
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
	if mode.status != "skill canceled" {
		t.Fatalf("status = %q, want skill canceled", mode.status)
	}
}

func TestBasicMenuOptionOpensEscapeMenu(t *testing.T) {
	mode := &WorldMode{}

	mode.handleBasicMenuAction(client.Context{}, "option")

	if !mode.escapeMenu.IsOpen() {
		t.Fatal("escape menu did not open")
	}
	if mode.escapeMenu.Action() != gameui.EscapeMenuActionNone {
		t.Fatalf("escape menu action = %d, want none", mode.escapeMenu.Action())
	}
	if mode.escapeMenu.Pending() {
		t.Fatal("escape menu kept stale pending state")
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
	if mode.status != "skill canceled" {
		t.Fatalf("status = %q, want skill canceled", mode.status)
	}
}

func TestPendingGroundSkillDoesNotCancelWhenClickingGround(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
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
	if mode.status == "skill canceled" {
		t.Fatal("ground skill was canceled instead of treated as a ground target")
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

	if !mode.deathModal.IsOpen() {
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

	if mode.deathModal.IsOpen() {
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
		nonPCViews: map[int]*playerSpriteView{
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

func TestFollowCameraInterpolatesTowardPlayerLikeReferenceView(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	ctx := client.Context{World: world}

	camera := followCamera{}
	now := time.Now()
	camera.Update(ctx, now)
	world.Player = worldstate.Actor{X: 14, Y: 20}
	camera.Update(ctx, now.Add(time.Second/60))

	if math.Abs(camera.x-10.9) > 0.001 || camera.y != 20.5 {
		t.Fatalf("camera target = %.2f, %.2f, want 10.9, 20.5", camera.x, camera.y)
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
	if got, want := entries[0].actor.Dir, worldstate.DirectionFromDelta(0, 1, 1, 1, 4); got != want {
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

func TestCameraFollowFactorUsesReferenceDefault(t *testing.T) {
	if got := cameraFollowFactor(); got != defaultCameraFollowFactor {
		t.Fatalf("camera follow factor = %.2f, want %.2f", got, defaultCameraFollowFactor)
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
	if got := cameraWheelZoomDelta(-2); got != 30 {
		t.Fatalf("wheel down delta = %.1f, want 30", got)
	}
}

func TestCameraZoomRangeMatchesRobrowserOutdoorDefaults(t *testing.T) {
	if got := sceneCameraZoom(); got != 125 {
		t.Fatalf("default zoom = %.1f, want roBrowser default 125", got)
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
	if got := worldSceneClearColor("data/yuno.gat"); got != (color.RGBA{R: 0x99, G: 0xcc, B: 0xff, A: 255}) {
		t.Fatalf("yuno clear color = %#v", got)
	}
	if got := worldSceneClearColor("5@tower.rsw"); got != (color.RGBA{R: 0x33, G: 0x00, B: 0x33, A: 255}) {
		t.Fatalf("tower clear color = %#v", got)
	}
}

func TestSceneLightingFromRSWMatchesReferenceDirection(t *testing.T) {
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 45,
		Latitude:  45,
		Diffuse:   [3]float32{1, 1, 1},
		Opacity:   1,
	}})
	want := modelPoint3{x: -0.5, y: -math.Sqrt2 / 2, z: -0.5}
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

func TestPosterizeGNDLightmapColorUsesRObrowserBuckets(t *testing.T) {
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

func TestGNDShadowMapPointMatchesROBrowserCellCenterMapping(t *testing.T) {
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

func TestClickedWalkCellByProjectedPolygonUsesWalkableGATCell(t *testing.T) {
	world := worldstate.New()
	world.Player.X = 5
	world.Player.Y = 5
	world.GAT = &res.GAT{
		Width:  12,
		Height: 12,
		Cells:  make([]res.GATCell, 12*12),
	}
	for i := range world.GAT.Cells {
		world.GAT.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
	projection := newSceneProjectionForTarget(800, 600, 5.5, 5.5, 0)
	point := projection.Project(6.5, 5.5, 0)

	x, y, ok := clickedWalkCellByProjectedPolygon(client.Context{World: world}, projection, int(point.x), int(point.y), 0, 11, 0, 11)
	if !ok || x != 6 || y != 5 {
		t.Fatalf("clicked cell = %d,%d ok=%t, want 6,5 true", x, y, ok)
	}
}

func TestClickedWalkCellByProjectedPolygonSkipsBlockedGATCell(t *testing.T) {
	world := worldstate.New()
	world.Player.X = 5
	world.Player.Y = 5
	world.GAT = &res.GAT{
		Width:  12,
		Height: 12,
		Cells:  make([]res.GATCell, 12*12),
	}
	world.GAT.Cells[5*world.GAT.Width+6] = res.GATCell{Type: res.GATTypeNone}
	projection := newSceneProjectionForTarget(800, 600, 5.5, 5.5, 0)
	point := projection.Project(6.5, 5.5, 0)

	_, _, ok := clickedWalkCellByProjectedPolygon(client.Context{World: world}, projection, int(point.x), int(point.y), 0, 11, 0, 11)
	if ok {
		t.Fatal("blocked cell should not be picked")
	}
}

func TestHoveredWalkCellRequiresProjectedWalkableCell(t *testing.T) {
	world := worldstate.New()
	world.Player.X = 5
	world.Player.Y = 5
	world.GAT = &res.GAT{
		Width:  12,
		Height: 12,
		Cells:  make([]res.GATCell, 12*12),
	}
	for i := range world.GAT.Cells {
		world.GAT.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
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

func TestTileCursorCellVertsUseGATHeightsWithLift(t *testing.T) {
	gat := &res.GAT{
		Width:  4,
		Height: 4,
		Cells:  make([]res.GATCell, 16),
	}
	gat.Cells[2*gat.Width+1] = res.GATCell{Heights: [4]float32{2, 2, 2, 2}, Type: res.GATTypeWalkable}
	now := time.Unix(0, 0)
	verts, ok := tileCursorCellVerts(gat, 1, 2, now)
	if !ok {
		t.Fatal("missing cursor cell")
	}
	wantY := 2 + tileCursorLift(now)
	if math.Abs(verts[0].y-wantY) > 0.001 {
		t.Fatalf("cursor vertex y = %.4f, want %.4f", verts[0].y, wantY)
	}
}

func TestApplyActorNameAckUpdatesWorldActor(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 1002})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 300, Name: "Guide#prontera"})

	if got := world.Actors[300].Name; got != "Guide" {
		t.Fatalf("actor name = %q, want Guide", got)
	}
}

func TestApplyActorNameAckUpdatesLocalPlayer(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 200, Name: "Kivutar"})

	if got := world.Player.Name; got != "Kivutar" {
		t.Fatalf("player name = %q, want Kivutar", got)
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
	mode.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 10, Message: "Warping..."})
	if !mode.npcDialog.Update(ctx) {
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

func TestActorDisplayNameUsesServerNameBeforeFallback(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Name: "Kafra Employee#izlude", Job: 1002}

	if got := actorDisplayName(ctx, actor, false); got != "Kafra Employee" {
		t.Fatalf("display name = %q, want Kafra Employee", got)
	}
}

func TestActorDisplayNameFallsBackToNonPCResource(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: 1002}

	if got := actorDisplayName(ctx, actor, false); got != "Poring" {
		t.Fatalf("display name = %q, want Poring", got)
	}
}

func TestActorDisplayNameDoesNotLabelUnnamedPlayerJob(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: 0}

	if got := actorDisplayName(ctx, actor, false); got != "" {
		t.Fatalf("display name = %q, want empty", got)
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
		t.Fatalf("hovered NPC name = %q, want resource fallback", got)
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
		t.Fatalf("hovered monster name = %q, want resource fallback", got)
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
	roBrowserItemEffects[itemID] = roBrowserItemEffect{
		effectIDs:         []int{effectPotionRed, effectBlessing},
		effectIDsOnCaster: []int{effectEndure},
	}
	defer delete(roBrowserItemEffects, itemID)

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
	roBrowserSkillUnitEffects[unitID] = roBrowserSkillUnitEffect{effectIDs: []int{effectPneuma, effectSafetyWall}}
	defer delete(roBrowserSkillUnitEffects, unitID)

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

func TestSkillNoDamageNotifyAddsRuwachAuraOnCaster(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 24, Amount: 1, TargetID: 2000000, SourceID: 2000000, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 2000000 || effect.effectID != effectRuwach || effect.x != 10 || effect.y != 20 {
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
		t.Fatalf("source animation = %+v, want roBrowser READYFIGHT action", anim)
	}
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectEndure {
		t.Fatalf("world effects = %+v, want Endure effect", mode.worldEffects)
	}
}

func TestSkillNoDamageNotifyDispatchesAllMappedEffectArrays(t *testing.T) {
	const skillID uint16 = 65000
	roBrowserSkillEffects[skillID] = roBrowserSkillEffect{
		effectIDs:            []int{effectHeal, effectBlessing},
		effectIDsOnCaster:    []int{effectEndure},
		successEffectIDs:     []int{effectProvoke},
		successEffectIDsSelf: []int{effectIncAgility},
	}
	defer delete(roBrowserSkillEffects, skillID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skillID, Amount: 2, TargetID: 1100, SourceID: 2000000, Result: 1})

	if len(mode.worldEffects) != 5 {
		t.Fatalf("world effects = %d, want 5: %+v", len(mode.worldEffects), mode.worldEffects)
	}
	want := []struct {
		effectID int
		actorID  uint32
		x        int
		y        int
	}{
		{effectHeal, 1100, 12, 22},
		{effectBlessing, 1100, 12, 22},
		{effectEndure, 2000000, 10, 20},
		{effectProvoke, 1100, 12, 22},
		{effectIncAgility, 2000000, 10, 20},
	}
	for i, wantEffect := range want {
		got := mode.worldEffects[i]
		if got.effectID != wantEffect.effectID || got.actorID != wantEffect.actorID || got.x != wantEffect.x || got.y != wantEffect.y {
			t.Fatalf("effect %d = %+v, want %+v", i, got, wantEffect)
		}
	}
}

func TestSkillNoDamageNotifyAddsHealEffectAndFloater(t *testing.T) {
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
	if len(mode.damageFloaters) != 1 {
		t.Fatalf("damage floaters = %d, want 1", len(mode.damageFloaters))
	}
	floater := mode.damageFloaters[0]
	if floater.actorID != 1100 || floater.text != "234" || floater.kind != damageFloaterRecoveryHP {
		t.Fatalf("floater = %+v", floater)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source cast animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want roBrowser DEFAULT skill action", anim)
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
		t.Fatalf("source animation = %+v, want roBrowser DEFAULT skill action", anim)
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
		t.Fatalf("source animation = %+v, want roBrowser weapon attack override", anim)
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
		t.Fatalf("source animation = %+v, want roBrowser DEFAULT skill action", anim)
	}
	if len(mode.worldEffects) != 2 || mode.worldEffects[0].effectID != effectHeal || mode.worldEffects[1].effectID != effectHealOffensive {
		t.Fatalf("world effects = %+v, want roBrowser heal effect followed by hit effect", mode.worldEffects)
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
	messages := mode.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Action failed." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestStealFailAckUsesSkillSpecificMessage(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 50, Result: 0, Cause: 0})

	messages := mode.console.Messages()
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
	if !item.Equip || item.Location != equipLocationAmmo {
		t.Fatalf("ammo item = %+v, want equipped ammo location 0x%04X", item, equipLocationAmmo)
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
	if !item.Equipped || item.Location != equipLocationAmmo {
		t.Fatalf("ammo item after equip ack = %+v, want equipped ammo location 0x%04X", item, equipLocationAmmo)
	}
}

func TestApplyEquippedArrowMarksAmmoSlot(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 1750, Type: 10, Amount: 100, Location: equipLocationAmmo, Equip: true, Equipped: true},
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
	if !item.Equip || !item.Equipped || item.Location != equipLocationAmmo {
		t.Fatalf("arrow item after ZC_EQUIP_ARROW = %+v, want equipped ammo location 0x%04X", item, equipLocationAmmo)
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
