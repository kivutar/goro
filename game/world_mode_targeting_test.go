package game

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func cursorHoverTestContext(actor worldstate.Actor) (client.Context, scene.Projection) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(actor)
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	point := projection.Project(scene.CellCenter(float64(actor.X)), scene.CellCenter(float64(actor.Y)), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.X), int(point.Y))
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
		ObjectType:    scene.ActorObjectTypePC,
		HasObjectType: true,
	})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	playerPoint := projection.Project(scene.CellCenter(10), scene.CellCenter(20), 0)
	pcPoint := projection.Project(scene.CellCenter(11), scene.CellCenter(20), 0)

	if actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 6, Type: skills.SkillTargetEnemy}, int(playerPoint.X), int(playerPoint.Y), now, nil); ok {
		t.Fatalf("enemy skill should not self-target: %+v", actor)
	}

	actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 28, Type: skills.SkillTargetFriend}, int(playerPoint.X), int(playerPoint.Y), now, nil)
	if !ok {
		t.Fatal("expected friend skill to target local player")
	}
	if actor.ID != 100 {
		t.Fatalf("target id = %d, want local account id 100", actor.ID)
	}

	actor, ok = clickedSkillTarget(ctx, projection, session.Skill{ID: 29, Type: skills.SkillTargetFriend}, int(pcPoint.X), int(pcPoint.Y), now, nil)
	if !ok || actor.ID != 300 {
		t.Fatalf("friend skill target = %+v ok=%t, want pc 300", actor, ok)
	}
	if actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 13, Type: skills.SkillTargetEnemy}, int(pcPoint.X), int(pcPoint.Y), now, nil); ok {
		t.Fatalf("enemy skill should not target pc outside pvp: %+v", actor)
	}

	world = worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             12,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	ctx.World = world
	projection = testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	mobPoint := projection.Project(scene.CellCenter(12), scene.CellCenter(20), 0)

	if actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 29, Type: skills.SkillTargetFriend}, int(mobPoint.X), int(mobPoint.Y), now, nil); ok {
		t.Fatalf("friend skill should not target mob without noshift: %+v", actor)
	}

	actor, ok = clickedSkillTarget(ctx, projection, session.Skill{ID: 13, Type: skills.SkillTargetEnemy}, int(mobPoint.X), int(mobPoint.Y), now, nil)
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
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200, NoShift: true},
		World:   world,
	}
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	mobPoint := projection.Project(scene.CellCenter(12), scene.CellCenter(20), 0)

	actor, ok := clickedSkillTarget(ctx, projection, session.Skill{ID: 28, Type: skills.SkillTargetFriend}, int(mobPoint.X), int(mobPoint.Y), now, nil)
	if !ok || actor.ID != 400 {
		t.Fatalf("noshift friend skill target = %+v ok=%t, want mob 400", actor, ok)
	}
}

func TestResurrectionCanTargetDeadPlayer(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	deadPlayer := worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    scene.ActorObjectTypePC,
		HasObjectType: true,
	}
	world.UpsertActor(deadPlayer)
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	point := projection.Project(scene.CellCenter(11), scene.CellCenter(20), 0)
	deaths := map[uint32]time.Time{deadPlayer.ID: {}}

	actor, ok := clickedSkillTarget(ctx, projection, session.Skill{
		ID: db.SkillALLResurrection, Type: skills.SkillTargetFriend,
	}, int(point.X), int(point.Y), now, deaths)
	if !ok || actor.ID != deadPlayer.ID {
		t.Fatalf("Resurrection target = %+v ok=%t, want dead player %d", actor, ok, deadPlayer.ID)
	}
	if actor, ok := clickedSkillTarget(ctx, projection, session.Skill{
		ID: db.SkillALHeal, Type: skills.SkillTargetFriend,
	}, int(point.X), int(point.Y), now, deaths); ok && actor.ID == deadPlayer.ID {
		t.Fatalf("ordinary support skill targeted dead player: %+v", actor)
	}
}

func TestCurrentNormalAttackRangeUsesEquippedBowAndVultureEye(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{Items: []session.InventoryItem{
			{Index: 1, ItemID: 1701, Location: db.EquipWeapon, WearLocation: db.EquipWeapon, Equip: true, Equipped: true},
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

func TestServerAttackRangeSurvivesEquipmentUpdates(t *testing.T) {
	ctx := client.Context{
		Session: &session.Session{Skills: session.Skills{List: []session.Skill{{ID: 44, Level: 3}}}},
		World:   worldstate.New(),
	}
	mode := NewWorldMode()
	for _, want := range []int{12, 3, 1} {
		data := make([]byte, 4)
		binary.LittleEndian.PutUint16(data, network.PacketZCAttackRange)
		binary.LittleEndian.PutUint16(data[2:], uint16(want))
		mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCAttackRange, Data: data}, time.Now())
		checkRange := func() {
			t.Helper()
			if got := currentNormalAttackRange(ctx); got != want || ctx.Session.AttackRange != want || ctx.World.Player.AttackRange != want {
				t.Fatalf("attack range = %d, session=%d player=%d; want server range %d", got, ctx.Session.AttackRange, ctx.World.Player.AttackRange, want)
			}
		}
		checkRange()
		applyInventoryItemList(ctx, []network.InventoryItem{
			{Index: 1, ItemID: 1701, Location: db.EquipWeapon, WearLocation: db.EquipWeapon, Equip: true, Equipped: true},
		})
		checkRange()
		applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 1, Location: db.EquipWeapon, Success: true, Unequip: true})
		checkRange()
		applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 1, Location: db.EquipWeapon, Success: true})
		checkRange()
	}
}

func TestClassSpecificBowViewIDAddsArrowProjectileEffect(t *testing.T) {
	actor := worldstate.Actor{Job: db.JobArcher, Weapon: 73}
	effectID, ok := normalAttackBeforeHitEffectID(nil, actor)
	if !ok || effectID != effects.ArrowShot {
		t.Fatalf("class-specific bow effect = %d ok=%t, want arrow projectile", effectID, ok)
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
	if world.Player.AttackRange != 7 {
		t.Fatalf("player attack range = %d, want server range 7", world.Player.AttackRange)
	}
}

func TestGroundClickCancelsPendingAttackChase(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             30,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})

	inputState := input.NewState()
	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	now := time.Now()
	mode := &WorldMode{
		pendingAttack: attackIntent{
			targetID:    300,
			lastChaseAt: now,
		},
		attackFocusID:    300,
		attackFocusStart: now,
	}
	ctx := client.Context{
		Input:   inputState,
		Network: netClient,
		Session: &session.Session{AccountID: 2000000, CharID: 150000, NoCtrl: true},
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	projection := mode.sceneProjection(ctx, ctx.ScreenW, ctx.ScreenH, now)
	point := projection.Project(scene.CellCenter(12), scene.CellCenter(20), 0)
	inputState.SetMousePosition(int(point.X), int(point.Y))
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.pendingAttack.targetID != 0 {
		t.Fatalf("pending attack target = %d, want canceled", mode.pendingAttack.targetID)
	}
	if mode.attackFocusID != 0 {
		t.Fatalf("attack focus target = %d, want canceled", mode.attackFocusID)
	}
}

func TestNPCClickIgnoresWalkCooldown(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	defer networkClient.Close()
	defer serverConn.Close()

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	npc := worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeNPC,
		HasObjectType: true,
	}
	world.UpsertActor(npc)

	inputState := input.NewState()
	mode := &WorldMode{
		walkCooldownUntil: time.Now().Add(time.Hour),
	}
	ctx := client.Context{
		Input:   inputState,
		Network: networkClient,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	projection := mode.sceneProjection(ctx, ctx.ScreenW, ctx.ScreenH, time.Now())
	point := projection.Project(scene.CellCenter(float64(npc.X)), scene.CellCenter(float64(npc.Y)), 0)
	inputState.SetMousePosition(int(point.X), int(point.Y))
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildNPCContactPacket(npc.ID, 0))
}

func TestNPCClickAfterKeyboardFocusLoss(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	npc := worldstate.Actor{
		ID: 300, X: 11, Y: 20,
		ObjectType: scene.ActorObjectTypeNPC, HasObjectType: true,
	}
	world.UpsertActor(npc)
	ctx := client.Context{
		Input: input.NewState(), Network: networkClient,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world, ScreenW: 800, ScreenH: 600,
	}
	mode := NewWorldMode()
	projection := mode.sceneProjection(ctx, ctx.ScreenW, ctx.ScreenH, time.Now())
	point := projection.Project(scene.CellCenter(float64(npc.X)), scene.CellCenter(float64(npc.Y)), 0)
	ctx.Input.SetMousePosition(int(point.X), int(point.Y))
	ctx.Input.SetKey(input.KeyAlt, true)
	ctx.Input.EndFrame()
	ctx.Input.SetMouseButton(input.MouseButtonLeft, true)

	// A stuck Alt used to route this city NPC click to companion commands,
	// even when the player had no companion. The cursor still said "talk".
	if got := mode.cursorDesiredAction(ctx, projection, time.Now()); got != cursorActionTalk {
		t.Fatalf("cursor = %d, want talk", got)
	}
	if !mode.handleCompanionAICommandClick(ctx, time.Now()) {
		t.Fatal("test did not reproduce the Alt-click interception")
	}
	ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
	ctx.Input.EndFrame()
	ctx.Input.ResetKeyboard() // The render backend does this on focus loss.
	ctx.Input.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildNPCContactPacket(npc.ID, 0))
}

func TestGroundClickRespectsWalkCooldown(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(64, 64)

	inputState := input.NewState()
	networkClient := network.NewClient(20080910, false)
	defer networkClient.Close()
	blockedUntil := time.Now().Add(time.Hour)
	mode := &WorldMode{walkCooldownUntil: blockedUntil}
	ctx := client.Context{
		Input:   inputState,
		Network: networkClient,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	projection := mode.sceneProjection(ctx, ctx.ScreenW, ctx.ScreenH, time.Now())
	point := projection.Project(scene.CellCenter(12), scene.CellCenter(20), 0)
	inputState.SetMousePosition(int(point.X), int(point.Y))
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !mode.walkCooldownUntil.Equal(blockedUntil) {
		t.Fatalf("walk cooldown changed to %s, want blocked until %s", mode.walkCooldownUntil, blockedUntil)
	}
	if mode.nextHeldWalkAt.IsZero() {
		t.Fatal("initial throttled click did not delay held-walk repeat")
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
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: 50, Level: 1, Type: skills.SkillTargetEnemy, Range: 1},
		TargetID: 300,
		Expires:  time.Now().Add(time.Second),
	}},
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	mode.skills().ContinuePendingTarget(ctx, "test")

	if mode.skillSystem.Pending.TargetID != 300 {
		t.Fatalf("pending target cleared")
	}
	if mode.skillSystem.Pending.ReadyAt.IsZero() {
		t.Fatal("pending skill was not scheduled")
	}
	if time.Until(mode.skillSystem.Pending.ReadyAt) > 100*time.Millisecond {
		t.Fatalf("readyAt too far in future: %s", time.Until(mode.skillSystem.Pending.ReadyAt))
	}
}

func TestUpdatePendingTargetSkillDoesNotKeepDelayingScheduledAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	readyAt := time.Now().Add(10 * time.Millisecond)
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: 50, Level: 1, Type: skills.SkillTargetEnemy, Range: 1},
		TargetID: 300,
		Expires:  time.Now().Add(time.Second),
		ReadyAt:  readyAt,
	}},
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	mode.skills().UpdatePendingTarget(ctx, "test", false)

	if !mode.skillSystem.Pending.ReadyAt.Equal(readyAt) {
		t.Fatalf("readyAt moved from %s to %s", readyAt, mode.skillSystem.Pending.ReadyAt)
	}
}

func TestContinuePendingTargetSkillWaitsWhileOutOfRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             15,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: 50, Level: 1, Type: skills.SkillTargetEnemy, Range: 1},
		TargetID: 300,
		Expires:  time.Now().Add(time.Second),
	}},
	}
	ctx := client.Context{Session: &session.Session{AccountID: 100, CharID: 200}, World: world}

	mode.skills().ContinuePendingTarget(ctx, "test")

	if mode.skillSystem.Pending.TargetID != 300 {
		t.Fatalf("pending target cleared")
	}
	if !mode.skillSystem.Pending.ReadyAt.IsZero() {
		t.Fatalf("pending skill readyAt = %s, want zero while out of range", mode.skillSystem.Pending.ReadyAt)
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

func TestLocalPlayerMoveSpeedUsesServerCartPenalty(t *testing.T) {
	for _, tc := range []struct {
		level int
		speed int
	}{
		{level: 1, speed: 217},
		{level: 5, speed: 187},
		{level: 10, speed: 150},
	} {
		t.Run(fmt.Sprintf("level_%d", tc.level), func(t *testing.T) {
			ctx := client.Context{
				World: worldstate.New(),
				Session: &session.Session{
					Selected: session.Character{ID: 150004, Option: db.EffectStateCart1},
					Skills: session.Skills{List: []session.Skill{
						{ID: db.SkillMCPushcart, Level: tc.level},
					}},
				},
			}
			applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusSpeed, Value: int64(tc.speed)})
			if ctx.World.Player.Speed != tc.speed {
				t.Fatalf("player speed = %d, want server speed %d", ctx.World.Player.Speed, tc.speed)
			}
		})
	}
}

func TestLocalPlayerMoveSpeedDefaultsWithoutServerSpeed(t *testing.T) {
	ctx := client.Context{
		World: worldstate.New(),
		Session: &session.Session{
			Selected: session.Character{ID: 150004, Option: db.EffectStateCart1},
			Skills: session.Skills{List: []session.Skill{
				{ID: db.SkillMCPushcart, Level: 5},
			}},
		},
	}
	refreshLocalPlayerMoveSpeed(ctx)
	if ctx.World.Player.Speed != defaultPlayerMoveSpeedMS {
		t.Fatalf("player speed = %d, want default %d", ctx.World.Player.Speed, defaultPlayerMoveSpeedMS)
	}
}

func TestLocalPlayerMoveSpeedWaitsForServerAfterCartChanges(t *testing.T) {
	ctx := client.Context{
		World: worldstate.New(),
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150004,
			Selected:  session.Character{ID: 150004, Option: db.EffectStateCart1},
		},
	}
	mode := &WorldMode{}
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusSpeed, Value: 187})
	applySkillInfoList(ctx, network.SkillInfoList{Skills: []network.SkillInfo{{ID: db.SkillMCPushcart, Level: 5}}})
	applyCartAmount(ctx, network.CartAmount{MaxAmount: 100})
	mode.applyPushCartStatus(ctx, network.StatusEffectChange{StatusID: db.StatusOnPushCart, Active: true})
	if ctx.World.Player.Speed != 187 {
		t.Fatalf("cart initialization changed server speed to %d", ctx.World.Player.Speed)
	}

	applySkillInfoUpdate(ctx, network.SkillInfoUpdate{Skill: network.SkillInfo{ID: db.SkillMCPushcart, Level: 10}})
	mode.applyPushCartStatus(ctx, network.StatusEffectChange{StatusID: db.StatusOnPushCart, Active: false})
	if ctx.World.Player.Speed != 187 {
		t.Fatalf("cart changes predicted speed %d before server update", ctx.World.Player.Speed)
	}

	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusSpeed, Value: 150})
	if ctx.World.Player.Speed != 150 {
		t.Fatalf("player speed after cart removal = %d, want server speed 150", ctx.World.Player.Speed)
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
	if len(mode.battleSystem.DamageFloaters) != 0 {
		t.Fatalf("parameter changes should not add recovery floaters: %+v", mode.battleSystem.DamageFloaters)
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
	if len(mode.battleSystem.DamageFloaters) != 1 {
		t.Fatalf("floaters = %d, want 1", len(mode.battleSystem.DamageFloaters))
	}
	floater := mode.battleSystem.DamageFloaters[0]
	if floater.ActorID != 2000000 || floater.Text != "12" || floater.Color != recoveryHPColor {
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
	if len(mode.battleSystem.DamageFloaters) != 1 {
		t.Fatalf("floaters = %d, want 1", len(mode.battleSystem.DamageFloaters))
	}
	floater := mode.battleSystem.DamageFloaters[0]
	if floater.ActorID != 2000000 || floater.Text != "7" || floater.Color != recoverySPColor || floater.Kind != battle.FloaterRecoverySP {
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

func TestApplyActorActionNotifySchedulesAttackAndHitAnimations(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
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

	sourceAnim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("local character animation missing")
	}
	if sourceAnim.ActionFamily != sprites.ActionPCAttack3 {
		t.Fatalf("source action = %d, want %d", sourceAnim.ActionFamily, sprites.ActionPCAttack3)
	}
	earlyAnim, ok := mode.actors().ActorAnimation(150000, sourceAnim.Started.Add(sourceAnim.Duration-time.Millisecond))
	if !ok || earlyAnim.ActionFamily != sprites.ActionPCAttack3 {
		t.Fatalf("early source animation = %+v ok=%t, want attack until it finishes", earlyAnim, ok)
	}
	readyAnim, ok := mode.actors().ActorAnimation(150000, sourceAnim.Started.Add(sourceAnim.Duration))
	if !ok || readyAnim.ActionFamily != sprites.ActionPCReadyFight || !readyAnim.Loop {
		t.Fatalf("post-attack animation = %+v ok=%t, want looping READYFIGHT after attack", readyAnim, ok)
	}
	targetAnim, ok := mode.actorSystem.ActorAnims[300]
	if !ok {
		t.Fatal("target animation missing")
	}
	if targetAnim.ActionFamily != sprites.ActionNonPCHurt {
		t.Fatalf("target action = %d, want %d", targetAnim.ActionFamily, sprites.ActionNonPCHurt)
	}
	if targetAnim.Started.Sub(sourceAnim.Started) != 580*time.Millisecond {
		t.Fatalf("hit delay = %s, want 580ms", targetAnim.Started.Sub(sourceAnim.Started))
	}
	if world.Dir != actors.DirectionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
	}
	if len(mode.battleSystem.DamageFloaters) != 1 || !mode.battleSystem.DamageFloaters[0].Starts.Equal(targetAnim.Started) {
		t.Fatalf("damage floater = %+v targetStarted=%s", mode.battleSystem.DamageFloaters, targetAnim.Started)
	}
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want regular hit effect", len(mode.effects().Active))
	}
	hitEffect := mode.effects().Active[0]
	if hitEffect.EffectID != effects.Hit1 || hitEffect.ActorID != 300 || !hitEffect.Starts.Equal(targetAnim.Started) {
		t.Fatalf("regular hit effect = %+v targetStarted=%s", hitEffect, targetAnim.Started)
	}
	if _, ok := mode.battleSystem.ActorLife[300]; ok {
		t.Fatal("target life should not be estimated from damage")
	}

	applySelfMoveAck(ctx, network.SelfMoveAck{FromX: 10, FromY: 20, ToX: 12, ToY: 20})
	mode.clearLocalActorAction(ctx)
	if anim, ok := mode.actors().ActorAnimation(150000, time.Now()); ok {
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
		ObjectType:    scene.ActorObjectTypeMob,
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

	targetAnim, ok := mode.actorSystem.ActorAnims[300]
	if !ok {
		t.Fatal("target animation missing")
	}
	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %+v, want regular and critical hit effects", mode.effects().Active)
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.Hit1 || effect.ActorID != 300 || !effect.Starts.Equal(targetAnim.Started) {
		t.Fatalf("regular hit effect = %+v targetStarted=%s", effect, targetAnim.Started)
	}
	if effect := mode.effects().Active[1]; effect.EffectID != effects.BashHit || effect.ActorID != 300 || !effect.Starts.Equal(targetAnim.Started) {
		t.Fatalf("critical hit effect = %+v targetStarted=%s", effect, targetAnim.Started)
	}
	if len(mode.battleSystem.DamageFloaters) != 1 || mode.battleSystem.DamageFloaters[0].Kind != battle.FloaterCritical {
		t.Fatalf("damage floaters = %+v, want critical", mode.battleSystem.DamageFloaters)
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
		ObjectType:    scene.ActorObjectTypeMob,
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

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want begin and hit effects", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.BashBegin || effect.ActorID != 2000000 {
		t.Fatalf("begin effect = %+v", effect)
	}
	effect := mode.effects().Active[1]
	if effect.EffectID != effects.BashHit || effect.ActorID != 300 || effect.X != 11 || effect.Y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
	if delay := effect.Starts.Sub(mode.actorSystem.ActorAnims[150000].Started); delay != 580*time.Millisecond {
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
		ObjectType:    scene.ActorObjectTypeMob,
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

	hurtAnim, ok := mode.actorSystem.ActorAnims[300]
	if !ok {
		t.Fatal("target hurt animation missing")
	}
	if hurtAnim.ActionFamily != sprites.ActionPCHurt {
		t.Fatalf("target action = %d, want %d", hurtAnim.ActionFamily, sprites.ActionPCHurt)
	}
	if actor := world.Actors[300]; !actor.Moving {
		t.Fatal("target should keep moving before scheduled hit time")
	}
	mode.processScheduledActorStops(ctx, hurtAnim.Started)
	actor := world.Actors[300]
	if actor.Moving || actor.MovePath != nil || actor.FromX != actor.X || actor.ToX != actor.X {
		t.Fatalf("target movement after hit = %+v, want stopped at hit position", actor)
	}
	if actor.X == 15 {
		t.Fatalf("target stopped at destination, want interpolated hit position: %+v", actor)
	}
	readyAnim, ok := mode.actors().ActorAnimation(300, hurtAnim.Started.Add(hurtAnim.Duration))
	if !ok || readyAnim.ActionFamily != sprites.ActionPCReadyFight || !readyAnim.Loop {
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
		ObjectType:    scene.ActorObjectTypeMob,
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

	hurtAnim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("local hurt animation missing")
	}
	if world.Player.Moving == false {
		t.Fatal("local player should keep moving before scheduled hit time")
	}
	mode.processScheduledActorStops(ctx, hurtAnim.Started)
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

func TestApplyActorActionNotifyPreservesLocalCastBarOnHit(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 15, Y: 20, Job: 2}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}
	mode.battle().StartActorCastBar(ctx, 2000000, 2*time.Second, time.Now())
	if len(mode.battleSystem.ActorCastBars) != 2 {
		t.Fatalf("cast bars = %d, want account and char aliases", len(mode.battleSystem.ActorCastBars))
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    2000000,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	if len(mode.battleSystem.ActorCastBars) != 2 {
		t.Fatalf("cast bars after hit = %+v, want preserved until server cancellation", mode.battleSystem.ActorCastBars)
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
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{pendingAttack: attackIntent{targetID: 400}}
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

	hurtAnim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("local hurt animation missing")
	}
	mode.processScheduledActorStops(ctx, hurtAnim.Started)
	if world.Player.Moving {
		t.Fatalf("local player moving during hurt = %+v, want paused", world.Player)
	}
	if len(mode.scheduledResumes) == 0 {
		t.Fatal("scheduled walk resume missing")
	}
	mode.processScheduledWalkResumes(ctx, hurtAnim.Started.Add(hurtAnim.Duration))
	if !world.Player.Moving {
		t.Fatalf("local player after hurt = %+v, want resumed walk", world.Player)
	}
	if world.Player.ToX != 15 || world.Player.ToY != 20 {
		t.Fatalf("local player resumed target = %d,%d, want 15,20", world.Player.ToX, world.Player.ToY)
	}
	afterWalk := world.Player.MoveStarted.Add(world.Player.MoveDuration).Add(time.Millisecond)
	if anim, ok := mode.actors().ActorAnimation(150000, afterWalk); ok {
		t.Fatalf("local player animation after resumed walk = %+v, want idle/no stale READYFIGHT", anim)
	}
	if anim, ok := mode.actors().ActorAnimation(2000000, afterWalk); ok {
		t.Fatalf("local account animation after resumed walk = %+v, want idle/no stale READYFIGHT", anim)
	}
}

func TestMovingActorPacketClearsReadyFightAction(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, X: 10, Y: 20, Job: 1, Speed: 150})
	mode := &WorldMode{actorSystem: actors.System{ActorAnims: map[uint32]scene.ActorAnimation{
		300: {
			ActionFamily: sprites.ActionPCReadyFight,
			Loop:         true,
			Play:         true,
			HasPlay:      true,
		},
	}}}
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

	if anim, ok := mode.actorSystem.ActorAnims[300]; ok {
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

	mode.startCombatAnimation(ctx, 300, sprites.ActionPCAttack2, time.Now(), scene.DefaultAttackAnimationDuration)

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
	mode := &WorldMode{actorSystem: actors.System{ActorAnims: map[uint32]scene.ActorAnimation{
		300: {
			ActionFamily: sprites.ActionPCSkill,
			Started:      started,
			Duration:     100 * time.Millisecond,
			Next: &scene.ActorAnimation{
				ActionFamily: sprites.ActionPCReadyFight,
				StartDelay:   50 * time.Millisecond,
				Loop:         true,
				Play:         true,
				HasPlay:      true,
			},
		},
	}}}

	if anim, ok := mode.actors().ActorAnimation(300, started.Add(125*time.Millisecond)); ok {
		t.Fatalf("scene.ActorAnimation before delayed next = %+v, want inactive gap", anim)
	}
	anim, ok := mode.actors().ActorAnimation(300, started.Add(150*time.Millisecond))
	if !ok || anim.ActionFamily != sprites.ActionPCReadyFight || !anim.Loop {
		t.Fatalf("scene.ActorAnimation delayed next = %+v ok=%t, want ready fight loop", anim, ok)
	}
	if got := anim.Started.Sub(started); got != 150*time.Millisecond {
		t.Fatalf("delayed next start = %s, want 150ms", got)
	}
}
