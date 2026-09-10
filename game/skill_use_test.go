package game

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestSkillTargetModes(t *testing.T) {
	if !isGroundTargetSkill(session.Skill{ID: 18, Type: 0x02}) {
		t.Fatal("ground skill type bit should request floor target")
	}
	if !isSelfTargetSkill(session.Skill{ID: 26, Type: 0x04}) {
		t.Fatal("self skill type bit should target the player")
	}
	if isSelfTargetSkill(session.Skill{ID: 21, Type: 0x06}) {
		t.Fatal("ground bit should win over self bit")
	}
}

func TestDeadPlayerCannotStartSkillUse(t *testing.T) {
	mode := &WorldMode{}
	err := mode.skills().Use(client.Context{
		Session: &session.Session{Dead: true},
	}, session.Skill{ID: 28, Type: skillTargetSelf, Level: 1}, "test")

	if err == nil {
		t.Fatal("dead player started a skill")
	}
}

func TestLevelOneTeleportQueuesRandomSelectionWithCast(t *testing.T) {
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

	mode := &WorldMode{}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 0x11223344, X: 10, Y: 20}
	ctx := client.Context{
		Network: netClient,
		Session: &session.Session{AccountID: 0x11223344},
		World:   world,
	}
	skill := session.Skill{ID: db.SkillALTeleport, Type: skillTargetSelf, Level: 1}
	if err := mode.skills().SendToID(ctx, skill, ctx.Session.AccountID, "test"); err != nil {
		t.Fatal(err)
	}

	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	packets := make([]byte, 30)
	if _, err := io.ReadFull(serverConn, packets); err != nil {
		t.Fatalf("reading Teleport requests: %v", err)
	}
	if opcode := binary.LittleEndian.Uint16(packets[0:2]); opcode != 0x0438 {
		t.Fatalf("cast opcode = 0x%04X, want 0x0438", opcode)
	}
	if level := binary.LittleEndian.Uint16(packets[2:4]); level != 1 {
		t.Fatalf("cast level = %d, want 1", level)
	}
	if opcode := binary.LittleEndian.Uint16(packets[10:12]); opcode != 0x011B {
		t.Fatalf("selection opcode = 0x%04X, want 0x011B", opcode)
	}
	if skillID := binary.LittleEndian.Uint16(packets[12:14]); skillID != db.SkillALTeleport {
		t.Fatalf("selection skill = %d, want %d", skillID, db.SkillALTeleport)
	}
	if mapName := string(packets[14:20]); mapName != "Random" {
		t.Fatalf("selection map = %q, want Random", mapName)
	}
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects before server reply = %+v, want none", mode.worldEffects)
	}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: skill.ID, Cause: 1})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after insufficient SP reply = %+v, want none", mode.worldEffects)
	}
	if messages := mode.ui.console.Messages(); len(messages) != 1 || messages[0].Text != "Not enough SP." {
		t.Fatalf("console messages = %+v, want insufficient SP error", messages)
	}
}

func TestTeleportWaitsForMapChange(t *testing.T) {
	for _, tc := range []struct {
		name   string
		level  int
		maps   []string
		cancel bool
	}{
		{name: "level one", level: 1, maps: []string{"Random"}},
		{name: "level two random", level: 2, maps: []string{"Random", "prontera"}},
		{name: "cancel level two", level: 2, maps: []string{"Random", "prontera"}, cancel: true},
		{name: "server skips menu", level: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			netClient, serverConn := newBotTestConnection(t, 20080910)
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			mode := &WorldMode{}
			ctx := client.Context{
				Network: netClient,
				Session: &session.Session{AccountID: 2000000},
				World:   world,
				Input:   input.NewState(),
			}
			skill := session.Skill{ID: db.SkillALTeleport, Type: skillTargetSelf, Level: tc.level}
			if err := mode.skills().Use(ctx, skill, "test"); err != nil {
				t.Fatal(err)
			}
			want := network.BuildUseSkillToIDPacketForClientDate(skill.ID, uint16(tc.level), world.Player.ID, 20080910)
			if tc.level == 1 {
				want = append(want, network.BuildSelectWarpPointPacket(skill.ID, "Random")...)
			}
			readBotTestPackets(t, serverConn, want)
			if tc.maps != nil {
				mode.applyWarpPointList(ctx, network.WarpPointList{SkillID: skill.ID, MapNames: tc.maps})
				if tc.level == 2 {
					if !mode.ui.teleportModal.IsOpen() {
						t.Fatal("level-two Teleport did not open its destination dialog")
					}
					key := input.KeyEnter
					if tc.cancel {
						key = input.KeyEscape
					}
					ctx.Input.SetKey(key, true)
					mode.ui.teleportModal.Update(ctx)
				}
				if !tc.cancel {
					readBotTestPackets(t, serverConn, network.BuildSelectWarpPointPacket(skill.ID, "Random"))
				}
			}
			if mode.ui.teleportModal.IsOpen() {
				t.Fatal("Teleport destination dialog remained open")
			}
			if len(mode.worldEffects) != 0 || len(mode.scheduledSounds) != 0 || mode.mapFade.phase != mapFadeNone {
				t.Fatalf("Teleport played before map confirmation: effects=%+v sounds=%+v fade=%+v", mode.worldEffects, mode.scheduledSounds, mode.mapFade)
			}

			// Even after canceling, a later unrelated warp follows the same
			// server-driven transition without a leftover Teleport effect.
			_, stop := mode.handleNetworkPacket(ctx, testMapChangePacket("prontera", 100, 120), time.Now())
			if !stop || mode.mapFade.phase != mapFadeOut || !mode.mapFade.hasChange {
				t.Fatalf("confirmed map change did not start the fade: stop=%t fade=%+v", stop, mode.mapFade)
			}
			if len(mode.worldEffects) != 0 || len(mode.scheduledSounds) != 0 {
				t.Fatal("map transition fabricated a local Teleport effect or sound")
			}
		})
	}
}

func TestChangeCartSkillOpensSelector(t *testing.T) {
	mode := &WorldMode{}
	controller := skillController{mode: mode}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}}

	if err := controller.Use(ctx, session.Skill{ID: skillChangeCart, Level: 1, Type: skillTargetSelf}, "test"); err != nil {
		t.Fatalf("change cart use failed: %v", err)
	}
	if !mode.ui.changeCartWindow.IsOpen() {
		t.Fatal("change cart window was not opened")
	}
	if mode.pendingSkill.skill.ID != 0 {
		t.Fatalf("pending skill = %+v, want none", mode.pendingSkill.skill)
	}
}

func TestSessionSkillFromNetworkUsesDBMaxBeforeResourceMax(t *testing.T) {
	skill := sessionSkillFromNetwork(network.SkillInfo{
		ID:         db.SkillHTBlitzbeat,
		Level:      10,
		Upgradable: true,
	})
	if skill.Level != 5 || skill.MaxLevel != 5 {
		t.Fatalf("blitz beat skill = %+v, want level/max clamped to db max 5", skill)
	}
}

func TestTargetSkillPendingLevelUsesDBMaxBeforeResourceMax(t *testing.T) {
	mode := &WorldMode{}
	controller := skillController{mode: mode}
	skill := session.Skill{ID: db.SkillHTBlitzbeat, Level: 10, MaxLevel: 10, Type: skillTargetEnemy, Range: 9}

	if err := controller.Use(client.Context{}, skill, "test"); err != nil {
		t.Fatalf("blitz beat use failed: %v", err)
	}
	if mode.pendingSkill.skill.Level != 5 || mode.pendingSkill.maxLevel != 5 {
		t.Fatalf("pending blitz beat = %+v, want level/max capped to db max 5", mode.pendingSkill)
	}

	inputState := input.NewState()
	inputState.AddWheel(0, 20)
	if !mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("pending skill wheel was not consumed")
	}
	if mode.pendingSkill.skill.Level != 5 {
		t.Fatalf("pending blitz beat level = %d, want capped to 5", mode.pendingSkill.skill.Level)
	}
}

func TestAutoRunTargetSkillStartsTargetSelection(t *testing.T) {
	mode := &WorldMode{}
	mode.skills().ApplyAutoRun(client.Context{}, network.AutoRunSkill{Skill: network.SkillInfo{
		ID:    db.SkillALLResurrection,
		Type:  skillTargetFriend,
		Level: 1,
		Range: 9,
		Name:  "Resurrection",
	}})

	if mode.pendingSkill.skill.ID != db.SkillALLResurrection {
		t.Fatalf("pending skill = %+v, want Resurrection", mode.pendingSkill.skill)
	}
	if mode.pendingSkill.skill.Level != 1 || mode.pendingSkill.skill.Type != skillTargetFriend {
		t.Fatalf("pending Resurrection = %+v", mode.pendingSkill.skill)
	}
}

func TestTextGroundSkillClickOpensPrompt(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(32, 32)
	inputState := input.NewState()
	projection := newSceneProjectionForTarget(1280, 720, cellCenter(10), cellCenter(20), 0)
	point := projection.Project(cellCenter(12), cellCenter(20), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{skill: session.Skill{ID: db.SkillHTTalkiebox, Name: "Talkie Box", Level: 1, Type: skillTargetPlace, Range: 9}},
	}
	ctx := client.Context{
		Input:     inputState,
		Session:   &session.Session{AccountID: 2000000, CharID: 150000},
		World:     world,
		ScreenW:   1280,
		ScreenH:   720,
		UIManager: &worldModeTestUIManager{},
	}

	mode.skills().HandleClick(ctx, projection, time.Now())

	if mode.pendingSkill.skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want cleared while prompt is open", mode.pendingSkill.skill.ID)
	}
	if mode.pendingSkillText.skill.ID != db.SkillHTTalkiebox || mode.pendingSkillText.x != 12 || mode.pendingSkillText.y != 20 {
		t.Fatalf("pending text skill = %+v", mode.pendingSkillText)
	}
	if !mode.ui.skillTextPrompt.IsOpen() {
		t.Fatal("skill text prompt was not opened")
	}
}

func TestMercenaryTargetSkillChasesFromMercenaryPosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 24, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		ObjectType:    actorObjectTypeMercenary,
		HasObjectType: true,
	})
	target := worldstate.Actor{
		ID:            300,
		X:             25,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(target)
	skill := session.Skill{ID: db.SkillMsBash, Level: 1, Type: skillTargetEnemy, Range: 1}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{
			Mercenary: session.Companion{
				ID:     400,
				Active: true,
				Skills: session.Skills{List: []session.Skill{skill}},
			},
		},
		World: world,
	}

	if !mode.skills().chaseTargetIfNeeded(ctx, skill, target, "test") {
		t.Fatal("mercenary skill did not chase even though only the player was in range")
	}
	if mode.pendingSkill.targetID != target.ID || mode.pendingSkill.skill.ID != db.SkillMsBash {
		t.Fatalf("pending mercenary skill = %+v", mode.pendingSkill)
	}
}

func TestHealApproachesZombieWithoutWalkingAwayOnOtherAxis(t *testing.T) {
	networkClient, serverConn := newBotTestConnection(t, 20080910)
	world := worldstate.New()
	world.GAT = flatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 200, Job: db.JobAcolyte, X: 20, Y: 20}
	zombie := worldstate.Actor{
		ID: 300, Job: 1015, X: 31, Y: 22,
		ObjectType: actorObjectTypeMob, HasObjectType: true,
	}
	world.UpsertActor(zombie)
	ctx := client.Context{
		World: world, Network: networkClient,
		Session: &session.Session{AccountID: 200},
	}
	mode := &WorldMode{}
	skill := session.Skill{ID: db.SkillALHeal, Level: 1, Type: skillTargetFriend, Range: 9}
	if err := mode.skills().UseTarget(ctx, skill, zombie, "test"); err != nil {
		t.Fatal(err)
	}
	want, ok := network.BuildWalkToXYPacketForClientDate(22, 20, 20080910)
	if !ok {
		t.Fatal("could not build expected approach packet")
	}
	readBotTestPackets(t, serverConn, want)
	if mode.pendingSkill.targetID != zombie.ID || mode.pendingSkill.skill.ID != skill.ID {
		t.Fatalf("pending skill = %+v, want Heal on the zombie", mode.pendingSkill)
	}

	// Once the server has moved us into range, cast on the original target.
	world.Player.X = 22
	mode.skills().UpdatePendingTarget(ctx, "test", false)
	if mode.pendingSkill.readyAt.IsZero() {
		t.Fatal("Heal was not scheduled after reaching range")
	}
	mode.pendingSkill.readyAt = time.Now().Add(-time.Millisecond)
	mode.skills().ProcessPendingTarget(ctx)
	readBotTestPackets(t, serverConn, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 1, zombie.ID, 20080910))
	if mode.pendingSkill.skill.ID != 0 {
		t.Fatal("Heal remained pending after casting")
	}
}

func TestMercenaryTargetSkillUsesRawServerRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 15, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		ObjectType:    actorObjectTypeMercenary,
		HasObjectType: true,
	})
	target := worldstate.Actor{
		ID:            300,
		X:             20,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(target)
	skill := session.Skill{ID: db.SkillMaDouble, Level: 2, Type: skillTargetEnemy, Range: 9}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{
			Mercenary: session.Companion{
				ID:     400,
				Active: true,
				Skills: session.Skills{List: []session.Skill{skill}},
			},
		},
		World: world,
	}

	if !mode.skills().chaseTargetIfNeeded(ctx, skill, target, "test") {
		t.Fatal("mercenary skill at distance 10 should chase for raw range 9")
	}
	if mode.pendingSkill.targetID != target.ID || mode.pendingSkill.skill.ID != db.SkillMaDouble {
		t.Fatalf("pending mercenary skill = %+v", mode.pendingSkill)
	}
}

func TestMercenaryPendingTargetSkillSchedulesFromMercenaryPosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             24,
		Y:             20,
		ObjectType:    actorObjectTypeMercenary,
		HasObjectType: true,
	})
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             25,
		Y:             20,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	skill := session.Skill{ID: db.SkillMsBash, Level: 1, Type: skillTargetEnemy, Range: 1}
	mode := &WorldMode{
		pendingSkill: pendingSkillTarget{
			skill:    skill,
			targetID: 300,
			expires:  time.Now().Add(time.Second),
		},
	}
	ctx := client.Context{
		Session: &session.Session{
			Mercenary: session.Companion{
				ID:     400,
				Active: true,
				Skills: session.Skills{List: []session.Skill{skill}},
			},
		},
		World: world,
	}

	mode.skills().UpdatePendingTarget(ctx, "test", false)

	if mode.pendingSkill.targetID != 300 {
		t.Fatal("pending mercenary skill was cleared")
	}
	if mode.pendingSkill.readyAt.IsZero() {
		t.Fatal("pending mercenary skill was not scheduled from the mercenary position")
	}
}

func TestTargetSkillRangeUsesMovingActorCurrentCell(t *testing.T) {
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

	if !targetSkillWithinRangeFrom(11, 10, 9, target) {
		t.Fatal("moving target should be in range at its current rendered cell")
	}
	if targetSkillWithinRangeCells(11, 10, target.X, target.Y, 9) {
		t.Fatal("test target final destination should be out of range")
	}
}
