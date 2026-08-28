package game

import (
	"testing"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

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

func TestResurrectionCanTargetDeadPlayer(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	deadPlayer := worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypePC,
		HasObjectType: true,
	}
	world.UpsertActor(deadPlayer)
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(11), cellCenter(20), 0)
	deaths := map[uint32]time.Time{deadPlayer.ID: {}}

	actor, ok := clickedSkillTarget(ctx, projection, session.Skill{
		ID: db.SkillALLResurrection, Type: skillTargetFriend,
	}, int(point.x), int(point.y), now, deaths)
	if !ok || actor.ID != deadPlayer.ID {
		t.Fatalf("Resurrection target = %+v ok=%t, want dead player %d", actor, ok, deadPlayer.ID)
	}
	if actor, ok := clickedSkillTarget(ctx, projection, session.Skill{
		ID: db.SkillALHeal, Type: skillTargetFriend,
	}, int(point.x), int(point.y), now, deaths); ok && actor.ID == deadPlayer.ID {
		t.Fatalf("ordinary support skill targeted dead player: %+v", actor)
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

func TestNormalAttackTargetWithinRangeRejectsRangedCorner(t *testing.T) {
	if !normalAttackTargetWithinRange(10, 20, 11, 21, 1) {
		t.Fatal("normal melee attack should keep diagonal adjacency")
	}
	if !attackTargetWithinRange(124, 70, 129, 75, 5) {
		t.Fatal("client skill-style range should still include the square corner")
	}
	if normalAttackTargetWithinRange(124, 70, 129, 75, 5) {
		t.Fatal("normal ranged attack should reject the server-failing square corner")
	}
	if !normalAttackTargetWithinRange(125, 72, 129, 75, 5) {
		t.Fatal("normal ranged attack should allow a non-corner cell at the same range")
	}
}

func TestNormalAttackRangeUsesMovingActorCurrentCell(t *testing.T) {
	now := time.Now()
	target := worldstate.Actor{
		ID:           300,
		X:            30,
		Y:            10,
		FromX:        20,
		FromY:        10,
		ToX:          30,
		ToY:          10,
		Moving:       true,
		MoveStarted:  now,
		MoveDuration: 10 * time.Second,
		MovePath: []worldstate.WalkStep{
			{X: 20, Y: 10},
			{X: 30, Y: 10},
		},
	}

	targetX, targetY := actorCurrentCell(target, now)
	if !normalAttackTargetWithinRange(11, 10, targetX, targetY, 9) {
		t.Fatal("moving target should be in range at its current rendered cell")
	}
	if attackTargetWithinRange(11, 10, target.X, target.Y, 9) {
		t.Fatal("test target final destination should be out of range")
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

func TestNormalAttackApproachCellAvoidsRejectedRangedCorner(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(200, 100)
	ctx := client.Context{World: world}

	x, y, ok := normalAttackApproachCellFromTarget(ctx, 123, 71, 129, 75, 5)
	if !ok {
		t.Fatal("expected ranged normal attack approach cell")
	}
	if x == 124 && y == 70 {
		t.Fatalf("normal attack approach = %d,%d, want to avoid server-rejected range corner", x, y)
	}
	if !normalAttackTargetWithinRange(x, y, 129, 75, 5) {
		t.Fatalf("normal attack approach = %d,%d, want server-compatible range", x, y)
	}
	if !walkTargetReachable(ctx, 123, 71, x, y) {
		t.Fatalf("normal attack approach = %d,%d, want reachable path", x, y)
	}
}

func TestRangedAttackApproachCellRequiresReachablePath(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(12, 5)
	for y := 0; y < world.GAT.Height; y++ {
		world.GAT.SetCellRawType(4, y, 1)
	}
	ctx := client.Context{World: world}

	if x, y, ok := rangedAttackApproachCellFromTarget(ctx, 1, 2, 8, 2, 3); ok {
		t.Fatalf("ranged approach = %d,%d, want no unreachable chase cell", x, y)
	}
}

func TestRangedAttackApproachCellSkipsUnreachablePreferredCell(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(12, 5)
	for _, cell := range [][2]int{
		{4, 1}, {4, 2}, {4, 3},
		{5, 1}, {5, 3},
		{6, 1}, {6, 2}, {6, 3},
	} {
		world.GAT.SetCellRawType(cell[0], cell[1], 1)
	}
	ctx := client.Context{World: world}

	x, y, ok := rangedAttackApproachCellFromTarget(ctx, 1, 2, 8, 2, 3)
	if !ok {
		t.Fatal("expected reachable fallback approach cell")
	}
	if x != 5 || y != 0 {
		t.Fatalf("ranged approach = %d,%d, want reachable fallback 5,0", x, y)
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

func TestClassSpecificBowViewIDAddsArrowProjectileEffect(t *testing.T) {
	actor := worldstate.Actor{Job: db.JobArcher, Weapon: 73}
	effectID, ok := normalAttackBeforeHitEffectID(nil, actor)
	if !ok || effectID != effectArrowShot {
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

func TestGroundClickCancelsPendingAttackChase(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             30,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})

	inputState := input.NewState()
	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	now := time.Now()
	mode := &WorldMode{
		tickCooldown: 2,
		pendingAttack: attackIntent{
			targetID:    300,
			expires:     now.Add(time.Second),
			lastChaseAt: now,
		},
		lockedAttackID:   300,
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
	point := projection.Project(cellCenter(12), cellCenter(20), 0)
	inputState.SetMousePosition(int(point.x), int(point.y))
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.pendingAttack.targetID != 0 {
		t.Fatalf("pending attack target = %d, want canceled", mode.pendingAttack.targetID)
	}
	if mode.lockedAttackID != 0 {
		t.Fatalf("locked attack target = %d, want canceled", mode.lockedAttackID)
	}
	if mode.attackFocusID != 0 {
		t.Fatalf("attack focus target = %d, want canceled", mode.attackFocusID)
	}
}

func TestNPCClickIgnoresWalkCooldown(t *testing.T) {
	networkClient, serverConn := newBotTestConnection(t, 20080910)
	defer networkClient.Close()
	defer serverConn.Close()

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	npc := worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    actorObjectTypeNPC,
		HasObjectType: true,
	}
	world.UpsertActor(npc)

	inputState := input.NewState()
	mode := &WorldMode{
		walkCooldownUntil: time.Now().Add(time.Hour),
		tickCooldown:      2,
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
	point := projection.Project(cellCenter(float64(npc.X)), cellCenter(float64(npc.Y)), 0)
	inputState.SetMousePosition(int(point.x), int(point.y))
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildNPCContactPacket(npc.ID, 0))
}

func TestGroundClickRespectsWalkCooldown(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)

	inputState := input.NewState()
	networkClient := network.NewClient(20080910, false)
	defer networkClient.Close()
	blockedUntil := time.Now().Add(time.Hour)
	mode := &WorldMode{walkCooldownUntil: blockedUntil, tickCooldown: 2}
	ctx := client.Context{
		Input:   inputState,
		Network: networkClient,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	projection := mode.sceneProjection(ctx, ctx.ScreenW, ctx.ScreenH, time.Now())
	point := projection.Project(cellCenter(12), cellCenter(20), 0)
	inputState.SetMousePosition(int(point.x), int(point.y))
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
