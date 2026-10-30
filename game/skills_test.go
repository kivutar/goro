package game

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func groundSkillTestContext(t *testing.T) (client.Context, *WorldMode, net.Conn) {
	t.Helper()
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 200, X: 20, Y: 20}
	return client.Context{
		World: world, Network: networkClient,
		Session: &session.Session{AccountID: 200},
		Input:   input.NewState(), ScreenW: 1280, ScreenH: 720,
	}, &WorldMode{}, serverConn
}

func TestGroundSkillWalksBeforeCasting(t *testing.T) {
	for _, tc := range []struct {
		name               string
		x, y, walkX, walkY int
		click              bool
	}{
		{"diagonal", 31, 31, 24, 24, false},
		{"mouse click", 31, 31, 24, 24, true},
		{"map origin", 0, 0, 7, 7, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, mode, serverConn := groundSkillTestContext(t)
			skill := session.Skill{ID: db.SkillMGFirewall, Level: 5, Type: skills.SkillTargetPlace, Range: 9}
			if tc.click {
				if err := mode.skills().Use(ctx, skill, "test"); err != nil {
					t.Fatal(err)
				}
				projection := testutil.Projection(ctx.ScreenW, ctx.ScreenH, scene.CellCenter(20), scene.CellCenter(20), 0)
				point := projection.Project(scene.CellCenter(float64(tc.x)), scene.CellCenter(float64(tc.y)), 0)
				ctx.Input.SetMousePosition(int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
				mode.skills().HandleClick(ctx, projection, time.Now())
			} else if err := mode.skills().UseGround(ctx, skill, tc.x, tc.y, "", "test"); err != nil {
				t.Fatal(err)
			}
			walk, ok := network.BuildWalkToXYPacketForClientDate(tc.walkX, tc.walkY, 20080910)
			if !ok {
				t.Fatal("invalid expected walk destination")
			}
			readBotTestPackets(t, serverConn, walk)
			if pending := mode.skillSystem.Pending; !pending.Ground || pending.X != tc.x || pending.Y != tc.y || pending.Skill.Level != 5 {
				t.Fatalf("pending ground skill = %+v", pending)
			}
			applySelfMoveAck(ctx, network.SelfMoveAck{FromX: 20, FromY: 20, ToX: tc.walkX, ToY: tc.walkY})
			assertNoBotTestPacket(t, serverConn, func() error {
				mode.skills().UpdatePendingTarget(ctx, "test", false)
				mode.skills().ProcessPendingTarget(ctx)
				return nil
			})
			ctx.World.Player.MoveStarted = time.Now().Add(-ctx.World.Player.MoveDuration - time.Millisecond)
			mode.skills().UpdatePendingTarget(ctx, "test", false)
			if mode.skillSystem.Pending.ReadyAt.IsZero() {
				t.Fatal("ground skill was not scheduled after arrival")
			}
			mode.skillSystem.Pending.ReadyAt = time.Now().Add(-time.Millisecond)
			mode.skills().ProcessPendingTarget(ctx)
			readBotTestPackets(t, serverConn, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 5, tc.x, tc.y, 20080910))
			if mode.skillSystem.Pending.Skill.ID != 0 {
				t.Fatal("ground skill remained pending after casting")
			}
			if len(mode.effects().Active) != 0 || len(mode.scheduledSounds) != 0 {
				t.Fatal("ground skill played effects before server confirmation")
			}
		})
	}
}

func TestGroundSkillInRangeCastsWithoutWalking(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	skill := session.Skill{ID: db.SkillMGFirewall, Level: 1, Type: skills.SkillTargetPlace, Range: 9}
	if err := mode.skills().UseGround(ctx, skill, 30, 20, "", "test"); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 1, 30, 20, 20080910))
	if mode.skillSystem.Pending.Skill.ID != 0 {
		t.Fatal("in-range ground skill was queued")
	}
}

func TestGroundSkillChaseCancels(t *testing.T) {
	for _, action := range []string{"escape", "right click", "new skill", "timeout", "map change"} {
		t.Run(action, func(t *testing.T) {
			ctx, mode, serverConn := groundSkillTestContext(t)
			skill := session.Skill{ID: db.SkillMGFirewall, Level: 1, Type: skills.SkillTargetPlace, Range: 9}
			if err := mode.skills().UseGround(ctx, skill, 31, 31, "", "test"); err != nil {
				t.Fatal(err)
			}
			walk, _ := network.BuildWalkToXYPacketForClientDate(24, 24, 20080910)
			readBotTestPackets(t, serverConn, walk)
			assertNoBotTestPacket(t, serverConn, func() error {
				ctx.World.Player.X, ctx.World.Player.Y = 24, 24
				mode.skillSystem.Pending.ReadyAt = time.Now().Add(-time.Millisecond)
				switch action {
				case "escape", "right click":
					if action == "escape" {
						ctx.Input.SetKey(input.KeyEscape, true)
					} else {
						ctx.Input.SetMouseButton(input.MouseButtonRight, true)
					}
					if !mode.skills().CancelFromInput(ctx) {
						t.Fatal("ground skill cancellation was ignored")
					}
				case "new skill":
					if err := mode.skills().Use(ctx, session.Skill{ID: db.SkillALHeal, Level: 1, Type: skills.SkillTargetFriend, Range: 9}, "test"); err != nil {
						return err
					}
				case "timeout":
					mode.skillSystem.Pending.Expires = time.Now().Add(-time.Millisecond)
				case "map change":
					mode.handleNetworkPacket(ctx, testMapChangePacket("prontera", 24, 24), time.Now())
				}
				mode.skills().UpdatePendingTarget(ctx, "test", false)
				mode.skills().ProcessPendingTarget(ctx)
				return nil
			})
			if mode.skillSystem.Pending.Ground || mode.skillSystem.Pending.TargetID != 0 {
				t.Fatal("canceled ground target remained queued")
			}
		})
	}
}

func TestGroundSkillRetryKeepsDeadlineAndText(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	skill := session.Skill{ID: db.SkillHTTalkiebox, Level: 1, Type: skills.SkillTargetPlace, Range: 9}
	mode.skillSystem.PendingText = skills.PendingTextTarget{Skill: skill, X: 31, Y: 31, Source: "test"}
	mode.skills().SendPendingSkillText(ctx, "Hello from the trap!")
	walk, _ := network.BuildWalkToXYPacketForClientDate(24, 24, 20080910)
	readBotTestPackets(t, serverConn, walk)
	deadline := mode.skillSystem.Pending.Expires
	mode.skillSystem.Pending.LastChaseAt = time.Now().Add(-scene.AttackRetryInterval)
	mode.skills().UpdatePendingTarget(ctx, "test", false)
	readBotTestPackets(t, serverConn, walk)
	if !mode.skillSystem.Pending.Expires.Equal(deadline) {
		t.Fatal("retry extended the skill timeout")
	}
	ctx.World.Player.X, ctx.World.Player.Y = 24, 24
	mode.skills().UpdatePendingTarget(ctx, "test", false)
	mode.skillSystem.Pending.ReadyAt = time.Now().Add(-time.Millisecond)
	mode.skills().ProcessPendingTarget(ctx)
	readBotTestPackets(t, serverConn, network.BuildUseSkillToGroundWithTextPacketForClientDate(skill.ID, 1, 31, 31, "Hello from the trap!", 20080910))
}

func TestGroundSkillUsesCompanionPositionAndMovePacket(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	ctx.Session.Mercenary = session.Companion{ID: 400, Active: true}
	mercenary := worldstate.Actor{ID: 400, X: 10, Y: 10, ObjectType: scene.ActorObjectTypeMercenary, HasObjectType: true}
	ctx.World.UpsertActor(mercenary)
	skill := session.Skill{ID: db.SkillMaLandmine, Level: 1, Type: skills.SkillTargetPlace, Range: 3}
	if err := mode.skills().UseGround(ctx, skill, 14, 14, "", "test"); err != nil {
		t.Fatal(err)
	}
	walk, ok := network.BuildCompanionMovePacket(400, 11, 11)
	if !ok {
		t.Fatal("invalid expected companion destination")
	}
	readBotTestPackets(t, serverConn, walk)
	mercenary.X, mercenary.Y = 11, 11
	ctx.World.UpsertActor(mercenary)
	mode.skills().UpdatePendingTarget(ctx, "test", false)
	if mode.skillSystem.Pending.ReadyAt.IsZero() {
		t.Fatal("ground skill did not use the companion's square range")
	}
	mode.skillSystem.Pending.ReadyAt = time.Now().Add(-time.Millisecond)
	mode.skills().ProcessPendingTarget(ctx)
	readBotTestPackets(t, serverConn, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 1, 14, 14, 20080910))
}

func TestMapChangeCancelsSkillTextPrompt(t *testing.T) {
	for _, targetMap := range []string{"prontera", "geffen"} {
		t.Run(targetMap, func(t *testing.T) {
			ctx, mode, serverConn := groundSkillTestContext(t)
			ctx.World.MapName = "prontera"
			manager := &worldModeTestUIManager{}
			ctx.UIManager = manager
			skill := session.Skill{ID: db.SkillHTTalkiebox, Level: 1, Type: skills.SkillTargetPlace, Range: 9}
			mode.skills().OpenSkillTextPrompt(ctx, skill, 31, 31, "test")
			if !mode.ui.skillTextPrompt.IsOpen() || len(manager.overlays) != 1 {
				t.Fatal("skill text prompt was not published")
			}
			mode.handleNetworkPacket(ctx, testMapChangePacket(targetMap, 24, 24), time.Now())
			if mode.ui.skillTextPrompt.IsOpen() || mode.skillSystem.PendingText.Skill.ID != 0 || len(manager.overlays) != 0 {
				t.Fatal("map change retained the skill text prompt or its overlay")
			}
			assertNoBotTestPacket(t, serverConn, func() error {
				mode.skills().SendPendingSkillText(ctx, "Too late")
				return nil
			})

			// Complete the normal fade/handoff, including reuse of the same map.
			mode.mapFade.Started = time.Now().Add(-maps.FadeOutDuration)
			if next, err := mode.Update(ctx); err != nil || next != nil {
				t.Fatalf("fade-out handoff = %T, err=%v", next, err)
			}
			for range maps.FadeHandoffFrames {
				mode.mapFade.RecordCoveredFrame()
			}
			next, err := mode.Update(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if targetMap == "prontera" {
				if next != nil {
					t.Fatalf("same-map warp replaced the world mode: %T", next)
				}
				readBotTestPackets(t, serverConn, network.BuildLoadEndAckPacket())
			} else {
				var ok bool
				mode, ok = next.(*WorldMode)
				if !ok {
					t.Fatalf("map handoff = %T, want *WorldMode", next)
				}
			}
			if mode.skills().UpdateSkillTextPrompt(ctx) || mode.ui.skillTextPrompt.IsOpen() || len(manager.overlays) != 0 {
				t.Fatal("completed map transition restored the canceled prompt")
			}
			assertNoBotTestPacket(t, serverConn, func() error {
				mode.skills().SendPendingSkillText(ctx, "Still too late")
				return nil
			})
		})
	}
}

func TestUnreachableGroundTargetReplacesPreviousCast(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	ctx.World.GAT = testutil.FlatWalkableGAT(12, 5)
	ctx.World.Player.X, ctx.World.Player.Y = 10, 2
	skill := session.Skill{ID: db.SkillMGFirewall, Level: 1, Type: skills.SkillTargetPlace, Range: 3}
	if err := mode.skills().UseGround(ctx, skill, 0, 2, "", "test"); err != nil {
		t.Fatal(err)
	}
	walk, _ := network.BuildWalkToXYPacketForClientDate(4, 2, 20080910)
	readBotTestPackets(t, serverConn, walk)
	for y := 0; y < ctx.World.GAT.Height; y++ {
		ctx.World.GAT.SetCellRawType(5, y, 1)
	}
	assertNoBotTestPacket(t, serverConn, func() error {
		if err := mode.skills().UseGround(ctx, skill, 0, 3, "", "test"); err != nil {
			return err
		}
		ctx.World.Player.X, ctx.World.Player.Y = 4, 2
		mode.skills().UpdatePendingTarget(ctx, "test", false)
		mode.skillSystem.Pending.ReadyAt = time.Now().Add(-time.Millisecond)
		mode.skills().ProcessPendingTarget(ctx)
		return nil
	})
	if mode.skillSystem.Pending.Ground || mode.skillSystem.Pending.TargetID != 0 {
		t.Fatal("unreachable selection left a previous target queued")
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
	skill := session.Skill{ID: db.SkillALTeleport, Type: skills.SkillTargetSelf, Level: 1}
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
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects before server reply = %+v, want none", mode.effects().Active)
	}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: skill.ID, Cause: 1})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after insufficient SP reply = %+v, want none", mode.effects().Active)
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
			netClient, serverConn := testutil.NewConnection(t, 20080910)
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			mode := &WorldMode{}
			ctx := client.Context{
				Network: netClient,
				Session: &session.Session{AccountID: 2000000},
				World:   world,
				Input:   input.NewState(),
			}
			skill := session.Skill{ID: db.SkillALTeleport, Type: skills.SkillTargetSelf, Level: tc.level}
			if err := mode.skills().Use(ctx, skill, "test"); err != nil {
				t.Fatal(err)
			}
			want := network.BuildUseSkillToIDPacketForClientDate(skill.ID, uint16(tc.level), world.Player.ID, 20080910)
			if tc.level == 1 {
				want = append(want, network.BuildSelectWarpPointPacket(skill.ID, "Random")...)
			}
			readBotTestPackets(t, serverConn, want)
			if tc.maps != nil {
				mode.skills().ApplyWarpPointList(ctx, network.WarpPointList{SkillID: skill.ID, MapNames: tc.maps})
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
			if len(mode.effects().Active) != 0 || len(mode.scheduledSounds) != 0 || mode.mapFade.Phase != maps.FadeNone {
				t.Fatalf("Teleport played before map confirmation: effects=%+v sounds=%+v fade=%+v", mode.effects().Active, mode.scheduledSounds, mode.mapFade)
			}

			// Even after canceling, a later unrelated warp follows the same
			// server-driven transition without a leftover Teleport effect.
			_, stop := mode.handleNetworkPacket(ctx, testMapChangePacket("prontera", 100, 120), time.Now())
			if !stop || mode.mapFade.Phase != maps.FadeOut || !mode.mapFade.HasChange {
				t.Fatalf("confirmed map change did not start the fade: stop=%t fade=%+v", stop, mode.mapFade)
			}
			if len(mode.effects().Active) != 0 || len(mode.scheduledSounds) != 0 {
				t.Fatal("map transition fabricated a local Teleport effect or sound")
			}
		})
	}
}

func TestChangeCartSkillOpensSelector(t *testing.T) {
	mode := &WorldMode{}
	controller := mode.skills()
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}}

	if err := controller.Use(ctx, session.Skill{ID: skills.SkillChangeCart, Level: 1, Type: skills.SkillTargetSelf}, "test"); err != nil {
		t.Fatalf("change cart use failed: %v", err)
	}
	if !mode.ui.changeCartWindow.IsOpen() {
		t.Fatal("change cart window was not opened")
	}
	if mode.skillSystem.Pending.Skill.ID != 0 {
		t.Fatalf("pending skill = %+v, want none", mode.skillSystem.Pending.Skill)
	}
}

func TestSessionSkillFromNetworkUsesDBMaxBeforeResourceMax(t *testing.T) {
	skill := scene.SessionSkillFromNetwork(network.SkillInfo{
		ID:         db.SkillHTBlitzbeat,
		Level:      10,
		Upgradable: true,
	})
	if skill.Level != 5 || skill.MaxLevel != 5 {
		t.Fatalf("blitz beat skill = %+v, want level/max clamped to db max 5", skill)
	}
}

func TestTextGroundSkillClickOpensPrompt(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(32, 32)
	inputState := input.NewState()
	projection := testutil.Projection(1280, 720, scene.CellCenter(10), scene.CellCenter(20), 0)
	point := projection.Project(scene.CellCenter(12), scene.CellCenter(20), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{Skill: session.Skill{ID: db.SkillHTTalkiebox, Name: "Talkie Box", Level: 1, Type: skills.SkillTargetPlace, Range: 9}}}}
	ctx := client.Context{
		Input:     inputState,
		Session:   &session.Session{AccountID: 2000000, CharID: 150000},
		World:     world,
		ScreenW:   1280,
		ScreenH:   720,
		UIManager: &worldModeTestUIManager{},
	}

	mode.skills().HandleClick(ctx, projection, time.Now())

	if mode.skillSystem.Pending.Skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want cleared while prompt is open", mode.skillSystem.Pending.Skill.ID)
	}
	if mode.skillSystem.PendingText.Skill.ID != db.SkillHTTalkiebox || mode.skillSystem.PendingText.X != 12 || mode.skillSystem.PendingText.Y != 20 {
		t.Fatalf("pending text skill = %+v", mode.skillSystem.PendingText)
	}
	if !mode.ui.skillTextPrompt.IsOpen() {
		t.Fatal("skill text prompt was not opened")
	}
}

func TestMercenaryTargetSkillChasesFromMercenaryPosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 24, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMercenary,
		HasObjectType: true,
	})
	target := worldstate.Actor{
		ID:            300,
		X:             25,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(target)
	skill := session.Skill{ID: db.SkillMsBash, Level: 1, Type: skills.SkillTargetEnemy, Range: 1}
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

	if mode.skills().UseTarget(ctx, skill, target, "test") != nil {
		t.Fatal("mercenary skill did not chase even though only the player was in range")
	}
	if mode.skillSystem.Pending.TargetID != target.ID || mode.skillSystem.Pending.Skill.ID != db.SkillMsBash {
		t.Fatalf("pending mercenary skill = %+v", mode.skillSystem.Pending)
	}
}

func TestHealApproachesZombieWithoutWalkingAwayOnOtherAxis(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 200, Job: db.JobAcolyte, X: 20, Y: 20}
	zombie := worldstate.Actor{
		ID: 300, Job: 1015, X: 31, Y: 22,
		ObjectType: scene.ActorObjectTypeMob, HasObjectType: true,
	}
	world.UpsertActor(zombie)
	ctx := client.Context{
		World: world, Network: networkClient,
		Session: &session.Session{AccountID: 200},
	}
	mode := &WorldMode{}
	skill := session.Skill{ID: db.SkillALHeal, Level: 1, Type: skills.SkillTargetFriend, Range: 9}
	if err := mode.skills().UseTarget(ctx, skill, zombie, "test"); err != nil {
		t.Fatal(err)
	}
	want, ok := network.BuildWalkToXYPacketForClientDate(22, 20, 20080910)
	if !ok {
		t.Fatal("could not build expected approach packet")
	}
	readBotTestPackets(t, serverConn, want)
	if mode.skillSystem.Pending.TargetID != zombie.ID || mode.skillSystem.Pending.Skill.ID != skill.ID {
		t.Fatalf("pending skill = %+v, want Heal on the zombie", mode.skillSystem.Pending)
	}

	// Once the server has moved us into range, cast on the original target.
	world.Player.X = 22
	mode.skills().UpdatePendingTarget(ctx, "test", false)
	if mode.skillSystem.Pending.ReadyAt.IsZero() {
		t.Fatal("Heal was not scheduled after reaching range")
	}
	mode.skillSystem.Pending.ReadyAt = time.Now().Add(-time.Millisecond)
	mode.skills().ProcessPendingTarget(ctx)
	readBotTestPackets(t, serverConn, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 1, zombie.ID, 20080910))
	if mode.skillSystem.Pending.Skill.ID != 0 {
		t.Fatal("Heal remained pending after casting")
	}
}

func TestHealApproachesDiagonalZombiesWithinServerRange(t *testing.T) {
	for _, offset := range [][2]int{{11, 11}, {-11, 11}, {11, -11}, {-11, -11}, {9, 9}} {
		t.Run(fmt.Sprintf("offset_%d_%d", offset[0], offset[1]), func(t *testing.T) {
			networkClient, serverConn := testutil.NewConnection(t, 20080910)
			world := worldstate.New()
			world.GAT = testutil.FlatWalkableGAT(64, 64)
			world.Player = worldstate.Actor{ID: 200, Job: db.JobAcolyte, X: 20, Y: 20}
			zombie := worldstate.Actor{
				ID: 300, Job: 1015, X: 20 + offset[0], Y: 20 + offset[1],
				ObjectType: scene.ActorObjectTypeMob, HasObjectType: true,
			}
			world.UpsertActor(zombie)
			ctx := client.Context{
				World: world, Network: networkClient,
				Session: &session.Session{AccountID: 200},
			}
			mode := &WorldMode{}
			skill := session.Skill{ID: db.SkillALHeal, Level: 1, Type: skills.SkillTargetFriend, Range: 9}
			if err := mode.skills().UseTarget(ctx, skill, zombie, "test"); err != nil {
				t.Fatal(err)
			}
			// At range 9 (+1 client allowance), the nearest diagonal cell
			// is seven cells from the target on each axis, not nine.
			x, y := zombie.X-7, zombie.Y-7
			if offset[0] < 0 {
				x = zombie.X + 7
			}
			if offset[1] < 0 {
				y = zombie.Y + 7
			}
			want, ok := network.BuildWalkToXYPacketForClientDate(x, y, 20080910)
			if !ok {
				t.Fatal("could not build expected approach packet")
			}
			readBotTestPackets(t, serverConn, want)
			applySelfMoveAck(ctx, network.SelfMoveAck{FromX: 20, FromY: 20, ToX: x, ToY: y})
			world.Player.MoveStarted = time.Now().Add(-world.Player.MoveDuration / 2)
			mode.skills().UpdatePendingTarget(ctx, "test", false)
			if !mode.skillSystem.Pending.ReadyAt.IsZero() {
				t.Fatal("Heal scheduled while still outside circular range")
			}

			world.Player.MoveStarted = time.Now().Add(-world.Player.MoveDuration - time.Millisecond)
			mode.skills().UpdatePendingTarget(ctx, "test", false)
			if mode.skillSystem.Pending.ReadyAt.IsZero() {
				t.Fatal("Heal not scheduled after reaching circular range")
			}
			// rAthena's player distance is int(hypot(dx,dy)-0.1).
			if distance := int(math.Hypot(float64(x-zombie.X), float64(y-zombie.Y)) - 0.1); distance > skill.Range {
				t.Fatalf("approach cell is still outside server range: distance=%d range=%d", distance, skill.Range)
			}
			mode.skillSystem.Pending.ReadyAt = time.Now().Add(-time.Millisecond)
			mode.skills().ProcessPendingTarget(ctx)
			readBotTestPackets(t, serverConn, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 1, zombie.ID, 20080910))
			if mode.skillSystem.Pending.Skill.ID != 0 {
				t.Fatal("Heal remained pending after casting")
			}
		})
	}
}

func TestPendingHealRechecksCircularRangeBeforeCasting(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 200, X: 20, Y: 20}
	// The zombie moved into a square corner after Heal was scheduled.
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 1015, X: 29, Y: 29})
	ctx := client.Context{World: world, Network: networkClient, Session: &session.Session{AccountID: 200}}
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: db.SkillALHeal, Level: 1, Type: skills.SkillTargetFriend, Range: 9},
		TargetID: 300,
		ReadyAt:  time.Now().Add(-time.Second),
		Expires:  time.Now().Add(time.Second),
	}},
	}
	mode.skills().ProcessPendingTarget(ctx)
	want, ok := network.BuildWalkToXYPacketForClientDate(22, 22, 20080910)
	if !ok {
		t.Fatal("could not build expected approach packet")
	}
	readBotTestPackets(t, serverConn, want)
	if mode.skillSystem.Pending.TargetID != 300 || !mode.skillSystem.Pending.ReadyAt.IsZero() {
		t.Fatalf("pending Heal = %+v, want to keep chasing the zombie", mode.skillSystem.Pending)
	}
}

func TestMercenaryTargetSkillUsesRawServerRange(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 15, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMercenary,
		HasObjectType: true,
	})
	target := worldstate.Actor{
		ID:            300,
		X:             20,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(target)
	skill := session.Skill{ID: db.SkillMaDouble, Level: 2, Type: skills.SkillTargetEnemy, Range: 9}
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

	if mode.skills().UseTarget(ctx, skill, target, "test") != nil {
		t.Fatal("mercenary skill at distance 10 should chase for raw range 9")
	}
	if mode.skillSystem.Pending.TargetID != target.ID || mode.skillSystem.Pending.Skill.ID != db.SkillMaDouble {
		t.Fatalf("pending mercenary skill = %+v", mode.skillSystem.Pending)
	}
}

func TestMercenaryPendingTargetSkillSchedulesFromMercenaryPosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             24,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMercenary,
		HasObjectType: true,
	})
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             25,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	skill := session.Skill{ID: db.SkillMsBash, Level: 1, Type: skills.SkillTargetEnemy, Range: 1}
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    skill,
		TargetID: 300,
		Expires:  time.Now().Add(time.Second),
	}},
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

	if mode.skillSystem.Pending.TargetID != 300 {
		t.Fatal("pending mercenary skill was cleared")
	}
	if mode.skillSystem.Pending.ReadyAt.IsZero() {
		t.Fatal("pending mercenary skill was not scheduled from the mercenary position")
	}
}

func TestAcolyteCastAuraAndPoses(t *testing.T) {
	for _, skill := range []uint16{
		db.SkillALRuwach, db.SkillALPneuma, db.SkillALTeleport, db.SkillALWarp,
		db.SkillALHeal, db.SkillALIncagi, db.SkillALDecagi, db.SkillALHolywater,
		db.SkillALCrucis, db.SkillALAngelus, db.SkillALBlessing, db.SkillALCure, db.SkillALHolylight,
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{
			SkillID: skill, SourceID: 2000000, TargetID: 300, Property: 6, DelayTime: 1000,
		})
		if !m.effects().HasActive(effects.BeginSpell6, 2000000, time.Now()) || m.effects().HasActive(effects.BeginSpell, 2000000, time.Now()) {
			t.Fatalf("skill %d: native cast aura must follow the holy property", skill)
		}
		pose := sprites.ActionPCReadyFight
		if skill == db.SkillALIncagi {
			pose = sprites.ActionPCSkill
		} else if skill == db.SkillALBlessing {
			pose = scene.SpriteActionIdle
		}
		if m.actorSystem.ActorAnims[2000000].ActionFamily != pose {
			t.Fatalf("skill %d: wrong casting pose", skill)
		}
	}
}

func TestAlchemistBerserkPotionAndCoupleCasting(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20.5, 20.5, 0)
	m := &WorldMode{}
	m.effects().AddAt(ctx, effects.ItemFast3, 300, start)
	if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start.Add(18*db.EffectFrameDuration)) {
		t.Fatal("Berserk Potion sound must coincide with the tick-18 shake")
	}
	spec, _ := effects.Lookup(effects.ItemFast3)
	if spec.Components[0].StrFile != "버서크" {
		t.Fatal("Berserk Potion lost its STR")
	}
	for _, tick := range []int{0, 36, 148} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawBerserkPotion(frame, ctx, spec.Components[1], m.effects().Active[0], start.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		want := 1
		if tick == 36 {
			want = 2
		}
		if tick == 148 {
			want = 0
		}
		if commands.Len() != want {
			t.Fatalf("wrong pillar/streak lifetime at tick %d", tick)
		}
	}
	texture := render.NewImage(12, 24)
	texture.Fill(color.White)
	billboard := &scene.Billboard{Image: texture, AnchorX: 6, AnchorY: 24}
	entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
	for _, tick := range []int{0, 150, 151} {
		frame := render.NewFrame(800, 600)
		m.actors().DrawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
		green := 120.0 / 255
		if tick == 151 {
			green = 1
		}
		if math.Abs(draw.FieldByName("ColorG").Float()-green) > .001 {
			t.Fatal("Berserk Potion tint must end at tick 151")
		}
	}
	m = &WorldMode{}
	m.effects().AddAt(ctx, effects.CoupleCasting, 300, start)
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, start.Add(180*db.EffectFrameDuration), false)
	if m.effects().Active[0].BlueCasting.Tick != 181 {
		t.Fatal("standalone Couple Casting must use its declared lifetime")
	}
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("Couple Casting stopped after one second")
	}
	v := commands.Index(0).FieldByName("Vertices").Index(0)
	if math.Abs(v.FieldByName("ColorG").Float()-89.0/255) > .001 || math.Abs(v.FieldByName("ColorB").Float()-182.0/255) > .001 {
		t.Fatal("Couple Casting must use its native pink tint")
	}
}

func TestArcherConcentrationCast(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{
		SkillID: db.SkillACConcentration, SourceID: 2000000, TargetID: 2000000, DelayTime: 4000, Property: 3,
	})
	if len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != effects.IncAgiDex || m.effects().Active[1].Duration != 4*time.Second || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\ef_incagidex.wav" {
		t.Fatal("Concentration must use its AGI/DEX casting label and sound, regardless of element")
	}
	label := effects.AgilityLabel(false, 119, 240)
	if math.Abs(label.Y+125.55) > .001 || label.Alpha != 200 {
		t.Fatal("Concentration's label must follow the cast duration instead of ending after 100 ticks")
	}
	ctx.Resources = testutil.RealDataManager(t)
	frame := render.NewFrame(800, 600)
	effect := m.effects().Active[1]
	spec, _ := effects.Lookup(effects.IncAgiDex)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	m.effects().DrawAgilityEffect(frame, ctx, projection, spec.Components[0], effect, 10.5, 20.5, 0, effect.Starts.Add(120*db.EffectFrameDuration))
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
		t.Fatal("Concentration's rising streaks must remain visible during a longer cast")
	}
}

func TestArcherArrowShowerSpreadAroundMovingTarget(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	now := time.Unix(100, 0)
	target := ctx.World.Actors[300]
	target.Moving = true
	target.FromX, target.FromY, target.ToX, target.ToY = 20, 20, 21, 19
	target.MoveStarted, target.MoveDuration = now.Add(-750*time.Millisecond), time.Second
	ctx.World.Actors[300] = target
	m := &WorldMode{}
	m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillACShower}, now, now)
	arrows := m.effects().Active[0].ArrowTargets
	if arrows[0].X != 21.25 || arrows[0].Z != 19.75 {
		t.Fatalf("center arrow must aim at the interpolated position: %+v", arrows[0])
	}
	var centerX, centerZ float64
	for _, arrow := range arrows[1:] {
		centerX += arrow.X / 8
		centerZ += arrow.Z / 8
	}
	if centerX != 21.5 || centerZ != 19.5 {
		t.Fatalf("spread center = %g,%g, want center of containing cell 21,19", centerX, centerZ)
	}
}

func TestArcherEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.effects().AddArrowShower(ctx, 2000000, 20, 20, false, start)
	effect := m.effects().Active[0]
	spec, _ := effects.Lookup(effects.ArrowShower)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for _, tc := range []struct {
		delay time.Duration
		count int
	}{{-1, 0}, {48 * time.Millisecond, 9}, {192 * time.Millisecond, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawArrowShower(frame, ctx, projection, spec.Components[0], effect, effect.Starts.Add(tc.delay))
		if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != tc.count {
			t.Fatalf("delay %s: drew %d arrows, want %d", tc.delay, got, tc.count)
		}
	}
	for _, id := range []int{effects.IncAgiDex, effects.Concentration} {
		if !m.effects().AddAt(ctx, id, 2000000, start) {
			t.Fatalf("could not add effect %d", id)
		}
	}
	for _, sound := range m.scheduledSounds {
		for _, path := range sound.paths {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Error(err)
			}
		}
	}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, start.Add(30*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
		t.Fatal("Concentration's rising streaks did not render")
	}
}

func TestSonicBlowCasterEffectsAndEightImpacts(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	hitAt := starts.Add(400 * time.Millisecond)
	action := network.ActorActionNotify{SkillID: db.SkillASSonicblow, SourceID: 2000000, TargetID: 300, Damage: 80, HitCount: 8}
	m.addSkillEffect(ctx, action, starts, hitAt)
	m.addSkillHitEffect(ctx, action, hitAt)
	if len(m.effects().Active) != 10 {
		t.Fatalf("effects = %+v", m.effects().Active)
	}
	for i, effect := range m.effects().Active {
		if i < 2 {
			if effect.EffectID != []int{effects.SonicBlow, effects.SonicBlow2}[i] || effect.ActorID != 2000000 || !effect.Starts.Equal(starts) {
				t.Fatalf("Sonic Blow cast effect on wrong actor or delayed: %+v", effect)
			}
		} else if effect.EffectID != effects.SonicBlowHit || effect.ActorID != 300 || !effect.Starts.Equal(hitAt.Add(time.Duration(i-2)*scene.MultiHitDelay)) {
			t.Fatalf("wrong Sonic Blow impact: %+v", effect)
		}
	}
}

func TestVenomKnifeFlightAndAfterimages(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillASVenomknife, SourceID: 2000000, TargetID: 300}, starts)
	if len(m.effects().Active) != 1 {
		t.Fatal("Venom Knife must launch exactly one projectile")
	}
	effect := m.effects().Active[0]
	first, angle, alpha := effects.ThrowItemAt(effect, 1)
	if math.Abs(first.X-11.1) > 1e-9 || angle != scene.DegreesToRadians(15) || alpha != 255 {
		t.Fatalf("knife should immediately advance three native units: %+v, %g, %d", first, angle, alpha)
	}
	near := effect
	near.ProjectileDirection.X = 1.2
	_, _, nearAlpha := effects.ThrowItemAt(near, 10)
	_, _, farAlpha := effects.ThrowItemAt(effect, 10)
	if nearAlpha >= farAlpha {
		t.Fatal("flight time must depend on distance")
	}
	if _, _, alpha := effects.ThrowItemAt(effect, 30); alpha != 0 {
		t.Fatal("knife must fade after arriving")
	}
	m.mapScene.Textures = map[string]*render.Image{"__effectfile_유저인터페이스/item/베넘나이프.bmp|blackkey=true": render.NewImage(8, 8)}
	frame := render.NewFrame(800, 600)
	projection := testutil.Projection(800, 600, 14, 20, 0)
	m.effects().Draw(frame, ctx, projection, starts.Add(6*db.EffectFrameDuration), false)
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if commands.Len() != 4 {
		t.Fatalf("knife and three afterimages: got %d commands", commands.Len())
	}
	for i, process := range []int{7, 6, 4, 2} {
		want, _, _ := effects.ThrowItemAt(effect, process)
		billboard := commands.Index(i)
		if math.Abs(billboard.FieldByName("Center").Index(0).Float()-want.X) > 1e-5 {
			t.Fatalf("afterimage %d did not use the previous two-tick sample", i)
		}
	}
}

func TestGrimtoothCastPackets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		damage int32
	}{
		{"normal", -30000},
		{"hidden WoE damage", 1},
		{"hallucination", 1234},
		{"hallucination zero", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.AssassinEffectContext()
			ctx.World.Actors[301] = worldstate.Actor{ID: 301, X: 16, Y: 21, Job: 1002}
			ctx.World.Actors[2000001] = worldstate.Actor{ID: 2000001, X: 10, Y: 21, Job: db.JobAssassin}
			m := &WorldMode{}
			starts := time.Unix(100, 0)
			action := network.ActorActionNotify{SkillID: db.SkillASGrimtooth, SkillLevel: 5,
				SourceID: 2000000, TargetID: 300, ServerTick: 1000, SourceSpeed: 500,
				Damage: tc.damage, HitCount: 1, Action: network.ActorActionSkill}
			m.addSkillEffect(ctx, action, starts, starts.Add(500*time.Millisecond))
			if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 1 {
				t.Fatalf("cast must start a trail and sound regardless of damage: %+v", m.effects().Active)
			}
			// The preamble may refer to a splash victim, not the primary target.
			action.TargetID, action.Damage, action.TargetSpeed = 301, 30, 200
			m.addSkillEffect(ctx, action, starts.Add(time.Millisecond), starts.Add(500*time.Millisecond))
			action.Action = 5 // splash damage must never launch a trail
			m.addSkillEffect(ctx, action, starts.Add(2*time.Millisecond), starts.Add(500*time.Millisecond))
			if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 1 {
				t.Fatal("damage packets duplicated the trail or sound")
			}
			action.Action = network.ActorActionSkill
			action.SourceID = 2000001
			m.addSkillEffect(ctx, action, starts, starts)
			action.SourceID, action.ServerTick = 2000000, 1500
			m.addSkillEffect(ctx, action, starts.Add(500*time.Millisecond), starts.Add(time.Second))
			if len(m.effects().Active) != 3 || len(m.scheduledSounds) != 3 {
				t.Fatal("another caster or subsequent cast was suppressed while the first trail was active")
			}
			m = &WorldMode{}
			action.Action = 5
			m.addSkillEffect(ctx, action, starts, starts)
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("a splash-only notification started a trail")
			}
			action.Action, action.SourceID = network.ActorActionSkill, 999
			m.addSkillEffect(ctx, action, starts, starts)
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("a missing caster started a trail at the victim")
			}
		})
	}
}

func TestGrimtoothPreambleAndFixedTrail(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	action := network.ActorActionNotify{SkillID: db.SkillASGrimtooth, SourceID: 2000000, TargetID: 300, Damage: -30000, Action: network.ActorActionSkill}
	m.addSkillEffect(ctx, action, starts, starts.Add(time.Second))
	action.Damage = 30
	m.addSkillEffect(ctx, action, starts, starts)
	m.addSkillHitEffect(ctx, action, starts)
	action.Action = 5 // secondary victim's damage-only packet
	m.addSkillEffect(ctx, action, starts, starts)
	if len(m.effects().Active) != 2 || m.effects().Active[0].EffectID != effects.Grimtooth || m.effects().Active[1].EffectID != effects.GrimtoothAtk {
		t.Fatalf("one cast must produce one trail and one impact per victim: %+v", m.effects().Active)
	}
	trail := m.effects().Active[0]
	if trail.ActorID != 2000000 || m.scheduledSounds[0].actorID != 2000000 || trail.ActorOrigin.X != 10.5 || trail.ActorOrigin.Z != 20.5 || math.Abs(trail.ProjectileDirection.X-.4) > 1e-9 || !trail.Starts.Equal(starts) {
		t.Fatalf("trail does not start at caster toward target: %+v", trail)
	}
	m.mapScene.Textures = map[string]*render.Image{"__effectfile_effect/stone.bmp|blackkey=true": render.NewImage(8, 8)}
	m.effects().Active = m.effects().Active[:1]
	projection := testutil.Projection(800, 600, 14, 20, 0)
	draw := func() reflect.Value {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, starts.Add(30*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 {
			t.Fatalf("expected one batched trail, got %d commands", commands.Len())
		}
		return commands.Index(0).FieldByName("Vertices")
	}
	before := draw()
	if before.Len() != 11*12 {
		t.Fatalf("tick 30 should have eleven horns: %d vertices", before.Len())
	}
	ctx.World.Player.X = 80
	delete(ctx.World.Actors, 300)
	after := draw()
	for i := 0; i < before.Len(); i++ {
		for _, axis := range []string{"X", "Y", "Z"} {
			if before.Index(i).FieldByName(axis).Float() != after.Index(i).FieldByName(axis).Float() {
				t.Fatal("moving actors pulled an already emitted Grimtooth trail")
			}
		}
	}
	spike := effects.GrimtoothSpike(trail, 0, 6)
	if spike.Origin != effects.GrimtoothSpike(trail, 0, 20).Origin {
		t.Fatal("trail spike kept moving after the native seven updates")
	}
	if effects.GrimtoothSpike(trail, 0, 39).Alpha != 0 {
		t.Fatal("trail spike did not finish fading")
	}
	impact := effects.Instance{EffectID: effects.GrimtoothAtk}
	if effects.GrimtoothSpike(impact, 0, 10).Origin != effects.GrimtoothSpike(impact, 0, 989).Origin || effects.GrimtoothSpike(impact, 0, 999).Alpha != 0 {
		t.Fatal("impact spikes should stop after eleven ticks and fade at the end")
	}
}

func TestBlacksmithHammerFallAndCastPoses(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.World.Player.Job, ctx.World.Player.Weapon = db.JobBlacksmith, db.WeaponAxe
	ctx.Session.Selected.Job, ctx.Session.Selected.Weapon = db.JobBlacksmith, db.WeaponAxe
	ctx.Session.Selected.ID = 100
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.startSkillSourceCastAnimation(ctx, 2000000, db.SkillBSHammerfall, time.Second, start)
	if m.actorSystem.ActorAnims[2000000].ActionFamily != sprites.AttackActionFamily(ctx.Resources, ctx.World.Player) {
		t.Fatal("Hammer Fall must cast with the equipped weapon's attack pose")
	}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillBSHammerfall, SourceID: 2000000, X: 15, Y: 20})
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillBSHammerfall, SourceID: 2000000, X: 15, Y: 20})
	if len(m.effects().Active) != 2 || m.actorSystem.ActorAnims[2000000].ActionFamily != sprites.ActionPCSkill {
		t.Fatal("each Hammer Fall must play its release pose and create a fresh ground impact")
	}
	for _, e := range m.effects().Active {
		if e.EffectID != effects.CrashEarth || e.ActorID != 0 || e.X != 15 || e.Y != 20 {
			t.Fatal("Hammer Fall must stay at its target cell")
		}
	}
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{
		{db.SkillBSHammerfall, effects.BashBegin}, {db.SkillBSAdrenaline, effects.BashBegin},
		{db.SkillBSMaximize, effects.BashBegin}, {db.SkillBSRepairweapon, effects.BeginSpell3},
	} {
		m := &WorldMode{}
		m.effects().AddSkillCastEffects(ctx, tc.skill, 3, 2000000, 300, 0, 0, time.Second, start, "test")
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != tc.aura {
			t.Fatalf("skill %d has the wrong casting aura", tc.skill)
		}
		for _, s := range m.scheduledSounds {
			if s.paths[0] == "effect\\black_adrenalinerush_a.wav" || s.paths[0] == "effect\\black_maximize_power_circle.wav" {
				t.Fatal("buff audio must wait for the success packet")
			}
		}
	}
}

func TestBlacksmithEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, tc := range []struct {
		id    int
		ticks []int
	}{
		{effects.HasteUp, []int{0, 30}}, {effects.RepairWeapon, []int{20, 55}},
		{effects.CrashEarth, []int{30}}, {effects.WeaponPerfect, []int{0}},
		{effects.MaximizePower, []int{0, 40, 47, 65}}, {effects.Overthrust, []int{0, 0}},
	} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, tc.id, 2000000, start)
		if len(m.scheduledSounds) != len(tc.ticks) {
			t.Fatalf("effect %d: incorrect sound count", tc.id)
		}
		for i, sound := range m.scheduledSounds {
			if sound.at.Sub(start) != time.Duration(tc.ticks[i])*db.EffectFrameDuration {
				t.Fatalf("effect %d sound %d has incorrect timing", tc.id, i)
			}
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		if value.FieldByName("commands").Len()+value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d drew nothing with client assets", tc.id)
		}
		if tc.id == effects.HasteUp {
			if got := value.FieldByName("worldBillboards").Len(); got != 4 {
				t.Fatalf("Adrenaline Rush needs four orbiting particles, got %d", got)
			}
			center := value.FieldByName("worldBillboards").Index(0).FieldByName("Center")
			if math.Abs(center.Index(1).Float()-(.07+(2.1+.462)/5)) > .001 {
				t.Fatal("Adrenaline particles have the wrong rising trajectory")
			}
		}
		if tc.id == effects.CrashEarth {
			spec, _ := effects.Lookup(tc.id)
			if spec.CameraShakeDelay != 35*db.EffectFrameDuration || m.scheduledSounds[0].paths[0] != "effect\\wizard_fire_pillar_b.wav" {
				t.Fatal("Hammer Fall must use the native impact sound and quake tick")
			}
		}
	}
}

func TestBlueCastingFlaresDuringCastAndEndsWithIt(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	start := time.Unix(100, 0)
	for _, duration := range []int{6, 60, 180} {
		m.effects().Active = nil
		m.effects().AddBetweenDuration(ctx, effects.BlueCasting, 300, 0, start, time.Duration(duration)*db.EffectFrameDuration)
		e := m.effects().Active[0]
		duration = max(duration, 56)
		if e.Expires.Sub(e.Starts) != time.Duration(duration)*db.EffectFrameDuration {
			t.Fatal("short casts need the native minimum flare duration")
		}
		spec, _ := effects.Lookup(effects.BlueCasting)
		frame := render.NewFrame(800, 600)
		m.effects().DrawBlueCastingEffect(frame, ctx, spec.Components[0], e, 18.5, 20.5, 0, start.Add(15*db.EffectFrameDuration))
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 1 {
			t.Fatal("missing cast rings")
		}
		one, many := &effects.CastingState{}, &effects.CastingState{}
		one.Advance(duration, duration)
		for i := 0; i <= duration; i++ {
			many.Advance(i, duration)
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatal("cast rings depend on render rate")
		}
		frame = render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 18.5, 20.5, 0), start.Add(time.Duration(duration+1)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 || len(m.effects().Active) != 0 {
			t.Fatal("rings survived the cast")
		}
	}
}

func TestAsuraNativeGlyphTimingAndPalette(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12, 20, 0)
	for _, id := range []int{328, 510} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, id, 300, start)
		effect := m.effects().Active[0]
		spec, _ := effects.Lookup(id)
		if len(spec.Components) != 2 || spec.Duration != 350*db.EffectFrameDuration || len(m.scheduledSounds) != 1 {
			t.Fatal("Asura needs native rings, glyphs, and one casting sound")
		}
		for _, tick := range []int{0, 1, 21, 291} {
			frame := render.NewFrame(800, 600)
			m.effects().DrawEffectGlyphs(frame, ctx, projection, spec.Components[1], effect, start.Add(time.Duration(tick)*db.EffectFrameDuration))
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			count := 0
			if tick == 1 {
				count = 10
			}
			if tick == 21 {
				count = 20
			}
			shade := 0.0
			if id == 510 {
				count += count / 10
				shade = 10.0 / 255
			}
			if draws.Len() != count {
				t.Fatalf("Asura %d tick %d: %d glyph layers, want %d", id, tick, draws.Len(), count)
			}
			for i := 0; i < draws.Len(); i++ {
				draw := draws.Index(i)
				if math.Abs(draw.FieldByName("ColorR").Float()-shade) > .00001 || draw.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendSourceOver) || math.Abs(draw.FieldByName("Center").Index(1).Float()-6.07) > .0001 {
					t.Fatal("Asura must use the 2008 dark glyph palette, normal alpha, and native height")
				}
			}
		}
	}
}

func TestChampionEffectsRenderWithNativeAssets(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12, 20, 0)
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Champion sprite missing: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			_, action, ok := sprites.ResolveAction(view.Body.Act, sprites.ActionPCSkill, dir)
			if !ok || len(action.Animations) <= 4 {
				t.Fatal("Champion is missing the fixed skill poses")
			}
		}
	}
	for _, tc := range []struct{ id, tick, end int }{{376, 35, 100}, {377, 10, 50}, {402, 10, 80}, {512, 30, 125}, {263, 10, 65}} {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, tc.id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
		}
		for _, tick := range []int{tc.tick, tc.end} {
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
			if (commands.Len() > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d: commands %d", tc.id, tick, commands.Len())
			}
			if tc.id == 512 && tick == tc.tick {
				vertices := commands.Index(0).FieldByName("Vertices")
				if vertices.Len() != 32*4 {
					t.Fatal("Chain Crush needs thirty-two flat streaks")
				}
				for i := 0; i < vertices.Len(); i++ {
					if math.Abs(vertices.Index(i).FieldByName("Y").Float()-1.27) > .0001 {
						t.Fatal("Chain Crush streaks must be six native units above ground")
					}
				}
			}
		}
	}
}

func TestClownGypsyEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	ids := []int{effects.TripleAttack3, effects.Moonlit, effects.MarionetteLink, effects.BottomHermode}
	for id := effects.TarotCard1; id <= effects.TarotCard14; id++ {
		ids = append(ids, id)
	}
	for _, id := range ids {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
			if id == effects.TripleAttack3 && sound.at.Sub(start) != 30*db.EffectFrameDuration {
				t.Fatal("Arrow Vulcan sound must wait 30 ticks")
			}
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(65*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len()+billboards.Len() == 0 {
			t.Fatalf("Clown effect %d drew nothing with actual client assets", id)
		}
		if id >= effects.TarotCard1 && id <= effects.TarotCard14 {
			cmd := billboards.Index(0)
			if math.Abs(cmd.FieldByName("Center").Index(1).Float()-6.07) > .001 || len(m.scheduledSounds) != 0 {
				t.Fatal("Tarot card must hover at native height and remain silent")
			}
		}
	}
	for _, skill := range []uint16{db.SkillCGMoonlit, db.SkillCGHermode, db.SkillCGLongingfreedom, db.SkillCGTarotcard} {
		for _, sound := range db.SkillEffects[skill].SFX {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestCreatorEffectRenderingWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	m := &WorldMode{}
	m.effects().AddBetween(ctx, effects.ThrowItem4, 300, 2000000, start)
	for _, tc := range []struct{ tick, billboards int }{{4, 0}, {5, 1}, {9, 3}, {10, 4}, {20, 8}, {60, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if billboards.Len() != tc.billboards {
			t.Fatalf("bottle frame %d: got %d billboards, want %d", tc.tick, billboards.Len(), tc.billboards)
		}
		for i := 0; i < billboards.Len(); i++ {
			if billboards.Index(i).FieldByName("ColorG").Float() != 1 {
				t.Fatal("Acid Demonstration bottles must be untinted")
			}
		}
	}
	for variant, id := range []int{effects.Slim, effects.Slim2, effects.Slim3} {
		m := &WorldMode{}
		m.effects().AddCellLifetime(ctx, id, 0, 22, 21, start, 0, false)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(25*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if billboards.Len() != 4 || commands.Len() != 1 {
			t.Fatal("potion burst needs its falling icon, three afterimages, and a ground wave")
		}
		wantVertices := 4
		if variant == 0 {
			wantVertices = 8
		}
		if commands.Index(0).FieldByName("Vertices").Len() != wantVertices {
			t.Fatal("red potion must draw its ground wave twice")
		}
		m.effects().Draw(render.NewFrame(800, 600), ctx, projection, start.Add(5*time.Second), false)
		if len(m.effects().Active) != 0 {
			t.Fatal("potion burst failed to expire")
		}
	}
	m = &WorldMode{}
	m.effects().AddAt(ctx, effects.AcidDemon, 300, start)
	actor := ctx.World.Actors[300]
	actor.X = 60
	ctx.World.Actors[300] = actor
	for _, tick := range []int{19, 43, 44} {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick == 44 {
			if commands.Len() != 0 {
				t.Fatal("Acid Demonstration rings survived their fade")
			}
			continue
		}
		if commands.Len() != 1 || commands.Index(0).FieldByName("Vertices").Len() != 168 {
			t.Fatal("Acid Demonstration must batch four rings using the original 20-segment meshes")
		}
		v := commands.Index(0).FieldByName("Vertices").Index(0)
		if math.Abs(v.FieldByName("X").Float()-20.5) > 1 || math.Abs(v.FieldByName("ColorG").Float()-50.0/255) > .001 {
			t.Fatal("impact rings must remain at the original victim position with their native magenta tint")
		}
	}
	texture := m.mapScene.Textures["__effectfile_effect/ring_black.tga|blackkey=false"]
	if texture == nil {
		t.Fatal("Acid Demonstration must retain the TGA's black pixels")
	}
	source, _, err := res.LoadImageExact(ctx.Resources, []string{"data/texture/effect/ring_black.tga"})
	if err != nil {
		t.Fatal(err)
	}
	partialAlpha := 0
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			want := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			got := texture.RGBA().RGBAAt(x, y)
			if got.R != want.R || got.G != want.G || got.B != want.B || got.A != want.A {
				t.Fatalf("impact texture changed straight-alpha pixel at %d,%d: %v != %v", x, y, got, want)
			}
			if want.A > 0 && want.A < 255 {
				partialAlpha++
			}
		}
	}
	if partialAlpha == 0 {
		t.Fatal("fixture lacks the TGA's translucent edges")
	}
	spec, _ := effects.Lookup(effects.AcidDemon)
	if spec.CameraShakeDelay != 5*db.EffectFrameDuration || len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start) {
		t.Fatal("impact sound must start immediately and quake at tick five")
	}
}

func TestGunslingerNativePosesAndCasting(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job = db.JobGunslinger
	for _, tc := range []struct {
		skill       uint16
		first, last int
		family      int
		duration    time.Duration
	}{
		{500, 4, 5, sprites.ActionPCSkill, 500 * time.Millisecond},
		{501, 0, 3, sprites.ActionPCSkill, time.Second}, {504, 0, 3, sprites.ActionPCSkill, time.Second},
		{505, 0, 3, sprites.ActionPCSkill, time.Second}, {506, 0, 3, sprites.ActionPCSkill, time.Second},
		{516, 0, 3, sprites.ActionPCSkill, time.Second}, {517, 0, 3, sprites.ActionPCSkill, time.Second},
		{508, 0, 2, sprites.ActionPCAttack1, 500 * time.Millisecond}, {521, 0, 2, sprites.ActionPCAttack1, 500 * time.Millisecond},
	} {
		for _, seed := range []int64{0, 1} {
			anim := effects.SkillAction(tc.skill).ActorAnimationForActorWithResources(nil, ctx.World.Player, time.Unix(100, seed), time.Second)
			var total time.Duration
			for i := tc.first; i <= tc.last; i++ {
				frame := i
				if tc.last == 3 && seed == 1 {
					frame = 3 - i
				}
				if !anim.HasFixedMotion || anim.FixedMotion != frame || anim.ActionFamily != tc.family {
					t.Fatalf("skill %d pose %d = %+v", tc.skill, frame, anim)
				}
				total += anim.Duration
				if i < tc.last {
					if anim.Next == nil {
						t.Fatal("missing next pose")
					}
					anim = *anim.Next
				}
			}
			if total != tc.duration || anim.Next != nil {
				t.Fatalf("skill %d lifetime=%s want %s", tc.skill, total, tc.duration)
			}
		}
	}
	for _, id := range []uint16{502, 503, 507, 512, 513, 514, 515, 518, 519, 520} {
		if effects.SkillAction(id).Action != effects.SkillActorActionAttack {
			t.Fatalf("skill %d must use weapon attack", id)
		}
	}
	for _, id := range []uint16{503, 504, 505, 512, 513, 514, 515, 517, 518, 519, 520, 521} {
		m := &WorldMode{}
		now := time.Unix(100, 0)
		m.startSkillSourceCastAnimation(ctx, 2000000, id, 2*time.Second, now)
		a := m.actorSystem.ActorAnims[2000000]
		family, frame := sprites.ActionPCSkill, 1
		if id >= 518 {
			family, frame = sprites.ActionPCAttack1, 0
		}
		if a.ActionFamily != family || !a.HasFixedMotion || a.FixedMotion != frame || a.Duration != 500*time.Millisecond {
			t.Fatalf("skill %d cast pose %+v", id, a)
		}
		m.effects().AddSkillCastEffects(ctx, id, 0, 2000000, 300, 0, 0, time.Second, now, "test")
		found := false
		for _, e := range m.effects().Active {
			found = found || e.EffectID == effects.BlueCasting
		}
		if !found {
			t.Fatalf("skill %d missing blue casting rings", id)
		}
	}
}

func TestGunslingerAssetsAndTiming(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, tc := range []struct{ id, tick int }{{456, 30}, {584, 3}, {623, 30}, {627, 40}, {628, 30}, {629, 30}, {637, 3}, {638, 3}, {639, 3}, {640, 3}, {641, 3}, {642, 3}, {643, 3}, {644, 3}, {645, 3}, {647, 3}, {648, 32}} {
		m.effects().Active = nil
		m.scheduledSounds = nil
		if !m.effects().AddBetween(ctx, tc.id, 300, 2000000, start) {
			t.Fatalf("missing effect %d", tc.id)
		}
		e := m.effects().Active[0]
		e.HitCount = 8
		spec, _ := effects.Lookup(tc.id)
		frame := render.NewFrame(800, 600)
		for _, c := range spec.Components {
			files := append([]string{}, c.TextureFiles...)
			if c.TextureFile != "" {
				files = append(files, c.TextureFile)
			}
			for _, file := range files {
				if m.effects().EffectFileTexture(ctx.Resources, file) == nil {
					t.Fatalf("effect %d missing texture %s", tc.id, file)
				}
			}
			if c.SpriteFile != "" && m.effects().EffectSpriteView(ctx.Resources, c.SpriteFile) == nil {
				t.Fatalf("missing sprite %s", c.SpriteFile)
			}
			m.effects().DrawFuncEffect(frame, ctx, projection, e, c, 0, 20.5, 20.5, 0, 0, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		}
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d drew nothing", tc.id)
		}
	}
	for id, tick := range map[int]int{584: 5, 623: 28, 627: 21, 637: 10, 643: 20, 644: 0, 648: 25} {
		m.scheduledSounds = nil
		m.effects().AddBetween(ctx, id, 300, 2000000, start)
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].at.Sub(start) != time.Duration(tick)*db.EffectFrameDuration {
			t.Fatalf("effect %d sound timing %+v", id, m.scheduledSounds)
		}
		if id == 648 && m.scheduledSounds[0].positioned {
			t.Fatal("Triple Action sound should be centered")
		}
	}
	// Body overlays are visible for both buff colours, then disappear.
	for _, id := range []int{548, 625, 626} {
		m.effects().Active = nil
		m.effects().AddAt(ctx, id, 300, start)
		billboard := &scene.Billboard{Image: render.NewImage(10, 20)}
		entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
		frame := render.NewFrame(800, 600)
		m.effects().DrawActorSpriteBodyEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, 0, start.Add(5*db.EffectFrameDuration))
		if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() < 2 {
			t.Fatalf("body flash %d missing", id)
		}
	}
}

func TestHunterEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	start := time.Unix(100, 0)
	for _, tc := range []struct{ id, frame int }{{effects.BlitzBeat, 10}, {effects.BlitzBeat, 40}, {effects.Flasher, 30}, {effects.Detecting, 20}, {effects.RemoveTrap, 20}, {effects.Shockwave, 20}} {
		m := &WorldMode{mapScene: maps.System{WhitePixel: render.NewImage(1, 1)}}
		m.effects().AddAt(ctx, tc.id, 300, start)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d frame %d drew nothing", tc.id, tc.frame)
		}
		for _, sound := range m.scheduledSounds {
			for _, path := range sound.paths {
				if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
					t.Error(err)
				}
			}
		}
	}
}

func TestKnightAttackEffects(t *testing.T) {
	starts := time.Unix(100, 0)
	hitAt := starts.Add(300 * time.Millisecond)
	for _, tc := range []struct {
		skill uint16
		begin int
		hits  []int
	}{
		{db.SkillKNPierce, effects.PierceSelf, []int{effects.Pierce}},
		{db.SkillKNSpearstab, effects.SpearStabSelf, []int{effects.Pierce}},
		{db.SkillKNBowlingbash, effects.BowlingSelf, []int{effects.Hit1, effects.BowlingBash, effects.Hit1, effects.Hit1}},
		{db.SkillKNSpearboomerang, effects.SpearBmrSelf, []int{effects.Hit4}},
		{db.SkillKNChargeatk, -1, []int{effects.BashHit}},
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 90, HitCount: 3, Action: 8}
		m.addSkillBeginEffect(ctx, action, starts)
		m.addSkillEffect(ctx, action, starts, hitAt)
		if tc.begin >= 0 {
			if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.begin || m.effects().Active[0].ActorID != 2000000 || !m.effects().Active[0].Starts.Equal(starts) {
				t.Fatalf("skill %d must start its caster STR before impact: %+v", tc.skill, m.effects().Active)
			}
		} else if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d invented a startup flash", tc.skill)
		}
		before := len(m.effects().Active)
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != before+len(tc.hits) {
			t.Fatalf("skill %d replayed its impact for each damage number: %+v", tc.skill, m.effects().Active)
		}
		for i, id := range tc.hits {
			effect := m.effects().Active[before+i]
			wantAt := hitAt
			if tc.skill == db.SkillKNBowlingbash && i > 1 {
				wantAt = wantAt.Add(time.Duration(i-1) * scene.MultiHitDelay)
			}
			if effect.EffectID != id || effect.ActorID != 300 || !effect.Starts.Equal(wantAt) {
				t.Fatalf("skill %d impact has wrong owner, visual or timing: %+v", tc.skill, effect)
			}
		}
		action.Damage = 0
		m = &WorldMode{}
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d played a hit on a miss", tc.skill)
		}
	}
}

func TestKnightBrandishStartsBothSTRsImmediately(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &WorldMode{}
	now := time.Now()
	action := network.ActorActionNotify{SkillID: db.SkillKNBrandishspear, SourceID: 2000000, TargetID: 300, Damage: 50}
	m.addSkillBeginEffect(ctx, action, now)
	m.addSkillEffect(ctx, action, now, now.Add(time.Second))
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 0 {
		t.Fatalf("Brandish effects or extra audio: %+v, %+v", m.effects().Active, m.scheduledSounds)
	}
	for i, want := range []struct {
		id    int
		actor uint32
	}{{effects.BrandishSpear2, 2000000}, {effects.BrandishSpear, 300}} {
		effect := m.effects().Active[i]
		if effect.EffectID != want.id || effect.ActorID != want.actor || !effect.Starts.Equal(now) {
			t.Fatalf("Brandish phase %d is delayed or misplaced: %+v", i, effect)
		}
	}
}

func TestKnightChargeAttackCast(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: db.SkillKNChargeatk, SourceID: 2000000, TargetID: 300, DelayTime: 1000})
	if m.actorSystem.ActorAnims[2000000].ActionFamily != scene.SpriteActionIdle {
		t.Fatal("Charge Attack must charge while standing")
	}
	found := false
	for _, effect := range m.effects().Active {
		if effect.EffectID == effects.BeginSpell6 {
			found = true
		}
	}
	if !found {
		t.Fatal("Charge Attack is missing its native casting aura")
	}
}

func TestKnightSpearFlightAndSounds(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillKNSpearboomerang, SourceID: 2000000, TargetID: 300, HitCount: 3}, starts)
	if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 4 {
		t.Fatalf("Spear Boomerang must launch one four-image trail: %+v", m.effects().Active)
	}
	effect := m.effects().Active[0]
	if effect.ActorID != 2000000 || effect.TargetID != 300 {
		t.Fatal("spear travels in the wrong direction")
	}
	for i, sound := range m.scheduledSounds {
		if !sound.at.Equal(starts.Add(time.Duration(i*4) * db.EffectFrameDuration)) {
			t.Fatal("spear sounds are not spaced four native ticks apart")
		}
	}
	for _, age := range []int{0, 6, 13} {
		point, ok := effects.SpearBoomerangPosition(effect, age)
		if !ok || math.Abs(point.X-(10.5+20*float64(age+1)/15)) > .001 || math.Abs(point.Y-3.07) > .001 {
			t.Fatalf("unexpected spear position at tick %d: %+v", age, point)
		}
	}
	for _, age := range []int{-1, 14, 15, 16} {
		if _, ok := effects.SpearBoomerangPosition(effect, age); ok {
			t.Fatalf("spear visible before launch or after reaching target: %d", age)
		}
	}
}

func TestMageCastsUseElementalAura(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, skill := range []uint16{db.SkillMGNapalmbeat, db.SkillMGSafetywall, db.SkillMGSoulstrike, db.SkillMGColdbolt,
		db.SkillMGFrostdiver, db.SkillMGStonecurse, db.SkillMGFireball, db.SkillMGFirewall, db.SkillMGFirebolt,
		db.SkillMGLightningbolt, db.SkillMGThunderstorm, db.SkillMGEnergycoat} {
		// Native PROPERTY_WATER through PROPERTY_SAINT select colored auras;
		// neutral, dark, ghost, and undead retain EF_BEGINSPELL.
		for property, aura := range []int{12, 54, 56, 55, 57, 59, 58, 12, 12, 12} {
			m := &WorldMode{}
			m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: uint32(property)})
			if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != aura || m.effects().Active[2].Duration != time.Second {
				t.Fatalf("skill %d property %d: expected casting aura %d for one second", skill, property, aura)
			}
		}
	}
}

func TestMageProjectileSequencesAndSounds(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill uint16
		ticks []int
	}{
		{db.SkillMGFireball, []int{20, 24, 28, 32}},
		{db.SkillMGSoulstrike, []int{6, 17, 28, 39}},
		{db.SkillMGColdbolt, []int{12}},
		{db.SkillMGFirebolt, []int{12}},
	} {
		m := &WorldMode{}
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, HitCount: 4}, start)
		if len(m.effects().Active) != 1 || len(m.scheduledSounds) != len(tc.ticks) {
			t.Fatalf("skill %d: expected a single sequence and %d sounds", tc.skill, len(tc.ticks))
		}
		for i, tick := range tc.ticks {
			sound := m.scheduledSounds[i]
			if !sound.at.Equal(start.Add(time.Duration(tick)*db.EffectFrameDuration)) || len(sound.paths) != 1 || strings.Contains(sound.paths[0], "%d") {
				t.Fatalf("skill %d: wrong sound or timing at tick %d", tc.skill, tick)
			}
		}
		if tc.skill == db.SkillMGColdbolt || tc.skill == db.SkillMGFirebolt {
			duration := 112 * db.EffectFrameDuration
			if tc.skill == db.SkillMGFirebolt {
				duration = 160 * db.EffectFrameDuration
			}
			if effect := m.effects().Active[0]; effect.HitCount != 4 || effect.Duration != duration {
				t.Fatal("bolt sequence must preserve all projectiles")
			}
		}
	}
}

func TestMageFrostDiverTrailAndBurst(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{mapScene: maps.System{Textures: map[string]*render.Image{"__effectfile_effect/ice.tga|blackkey=true": render.NewImage(32, 32)}}}
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillMGFrostdiver, SourceID: 2000000, TargetID: 300}, start)
	effect := m.effects().Active[0]
	if effect.ActorID != 2000000 || effect.TargetID != 300 || !effect.Starts.Equal(start) || effects.FrostDiverTrailCount(effect) != 48 {
		t.Fatal("Frost Diver must travel from the caster to the target, starting at action time")
	}
	spec, _ := effects.Lookup(effects.FrostDiver)
	for _, tick := range []int{0, 10, 38, 48, 86, 87} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawFrostDiverSpikes(frame, ctx, spec.Components[0], effect, effect.ActorOrigin, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick >= 87 {
			if cmds.Len() != 0 {
				t.Fatal("ice trail survived its last spike")
			}
			continue
		}
		want := min(48, tick+1) - max(0, tick-39)
		if cmds.Len() != 1 || cmds.Index(0).FieldByName("Vertices").Len() != want*12 {
			t.Fatalf("tick %d: incorrect ice spike count", tick)
		}
	}
	spec, _ = effects.Lookup(effects.FrostDiverHit)
	frame := render.NewFrame(800, 600)
	m.effects().DrawFrostDiverSpikes(frame, ctx, spec.Components[0], effects.Instance{EffectID: effects.FrostDiverHit, Starts: start}, effect.ActorOrigin, start.Add(20*db.EffectFrameDuration))
	cmd := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Index(0)
	vertices := cmd.FieldByName("Vertices")
	if vertices.Len() != 8*12 || render.Blend(cmd.FieldByName("Options").FieldByName("Blend").Int()) != render.BlendSourceOver {
		t.Fatal("Frost Diver impact must be eight translucent ice spikes")
	}
	for i := 0; i < vertices.Len(); i += 3 {
		base, tip := vertices.Index(i), vertices.Index(i+1)
		if tip.FieldByName("Y").Float() <= base.FieldByName("Y").Float() {
			t.Fatal("ice spike points into the ground")
		}
	}
}

func TestMageRealAssets(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{effects.SoulStrike, effects.FireBall, effects.FrostDiver, effects.FrostDiverHit, effects.FireBolt, effects.ColdBolt, effects.Sight, effects.StoneCurse, effects.EnergyCoat, effects.SafetyWall, effects.FireWall, effects.ThunderStorm} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.effects().AddBetween(ctx, id, 300, 2000000, start) {
				t.Fatal("effect unavailable")
			}
			for _, sound := range m.scheduledSounds {
				for _, path := range sound.paths {
					if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
						t.Fatal(err)
					}
				}
			}
			draws := 0
			for _, tick := range []int{0, 12, 20, 30, 45} {
				frame := render.NewFrame(800, 600)
				m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 20, 20, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
				v := reflect.ValueOf(frame).Elem()
				draws += v.FieldByName("worldCommands").Len() + v.FieldByName("worldBillboards").Len() + v.FieldByName("commands").Len()
			}
			if draws == 0 {
				t.Fatal("effect produced no geometry with real assets")
			}
		})
	}
	if _, err := ctx.Resources.ReadFile("data/wav/_stonecurse.wav"); err != nil {
		t.Fatal(err)
	}
}

func TestMerchantAttackEffects(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	start := time.Unix(100, 0)
	hit := start.Add(500 * time.Millisecond)
	for _, skill := range []uint16{db.SkillMCMammonite, db.SkillMCCartrevolution} {
		for _, damage := range []int32{0, 100} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: damage, Action: network.ActorActionSkill}
			m.addSkillBeginEffect(ctx, action, start)
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			m.addSkillHitEffect(ctx, action, hit)
			begin, impact := effects.BashBegin, effects.Mammonite
			if skill == db.SkillMCCartrevolution {
				begin, impact = effects.CartRevolution, effects.CartRevolution
			}
			want := 1
			if damage > 0 {
				want += 2
			}
			if len(m.effects().Active) != want || m.effects().Active[0].EffectID != begin || m.effects().Active[0].ActorID != 2000000 || !m.effects().Active[0].Starts.Equal(start) {
				t.Fatalf("skill %d damage %d: wrong caster effects: %+v", skill, damage, m.effects().Active)
			}
			if damage > 0 {
				if e := m.effects().Active[2]; e.EffectID != impact || e.ActorID != 300 || !e.Starts.Equal(hit) {
					t.Fatalf("skill %d: impact must appear on the victim at hit time: %+v", skill, e)
				}
			}
			if skill == db.SkillMCCartrevolution {
				for i, effect := range []effects.Instance{m.effects().Active[0], m.effects().Active[len(m.effects().Active)-1]} {
					if damage == 0 && i == 1 {
						break
					}
					for j, tick := range []int{7, 20} {
						if sound := m.scheduledSounds[i*2+j]; !sound.at.Equal(effect.Starts.Add(time.Duration(tick) * db.EffectFrameDuration)) {
							t.Fatalf("Cart Revolution sound must coincide with burst %d: %+v", tick, sound)
						}
					}
				}
			}
			before, sounds := len(m.effects().Active), len(m.scheduledSounds)
			action.Action = 5 // Secondary splash notification.
			m.addSkillBeginEffect(ctx, action, start)
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			m.addSkillHitEffect(ctx, action, hit)
			if len(m.effects().Active) != before || len(m.scheduledSounds) != sounds {
				t.Fatal("splash victims must not replay the skill")
			}
		}
	}
	for _, weapon := range []int16{0, 1, 6} {
		actor := ctx.World.Player
		actor.Weapon = weapon
		if got := effects.SkillAction(db.SkillMCMammonite).ActionFamilyForActorWithResources(nil, actor); got != sprites.AttackActionFamily(nil, actor) {
			t.Fatal("Mammonite must use the equipped weapon's normal attack animation")
		}
	}
}

func TestMammoniteCoinBounces(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.World.GAT = &res.GAT{Width: 30, Height: 30, Cells: make([]res.GATCell, 900)}
	for i := range ctx.World.GAT.Cells {
		ctx.World.GAT.Cells[i].Heights = [4]float32{4, 4, 4, 4}
	}
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.effects().AddAt(ctx, effects.Mammonite, 300, start)
	effect := m.effects().Active[0]
	if effect.CoinTrails == nil {
		t.Fatal("missing five coin trajectories")
	}
	bounces := 0
	for _, trail := range effect.CoinTrails {
		if len(trail) < 40 || len(trail) > 69 || trail[0].Y <= 6 || trail[5].Y <= trail[0].Y {
			t.Fatal("coins must rise from ten native units above the victim's ground height")
		}
		for age := 1; age < len(trail); age++ {
			if trail[age].Y < 4 && trail[age].Y < trail[age-1].Y {
				bounces++
				if age+1 < len(trail) && trail[age+1].Y <= trail[age].Y {
					t.Fatal("coin failed to bounce off terrain")
				}
			}
		}
	}
	if bounces < 5 || len(m.scheduledSounds) != bounces {
		t.Fatalf("one sound per bounce: bounces=%d sounds=%d", bounces, len(m.scheduledSounds))
	}
	for _, sound := range m.scheduledSounds {
		if !sound.at.After(start.Add(20 * db.EffectFrameDuration)) {
			t.Fatal("coin audio must wait until a coin hits the ground")
		}
	}
	ctx.Session.LessEffects = true
	m = &WorldMode{}
	m.effects().AddAt(ctx, effects.Mammonite, 300, start)
	if m.effects().Active[0].CoinTrails != nil || len(m.scheduledSounds) != 0 {
		t.Fatal("minimal Mammonite must omit coins and bounce sounds")
	}
}

func TestMerchantEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	for _, id := range []int{effects.Mammonite, effects.CartRevolution, effects.Loud} {
		m.effects().AddAt(ctx, id, 300, start)
		spec, _ := effects.Lookup(id)
		for _, component := range spec.Components {
			if component.StrFile != "" && m.effects().LoadWorldEffectSTR(ctx.Resources, component.StrFile, component.TexturePath) == nil {
				t.Fatalf("missing STR %q", component.StrFile)
			}
		}
	}
	for _, sound := range m.scheduledSounds {
		for _, path := range sound.paths {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Error(err)
			}
		}
	}
	projection := testutil.Projection(800, 600, 11, 20, 0)
	coin, _ := effects.Lookup(effects.Mammonite)
	for _, tc := range []struct{ tick, count int }{{19, 0}, {20, 5}, {21, 10}, {35, 10}, {90, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawMammoniteCoins(frame, ctx, projection, coin.Components[1], m.effects().Active[0], start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); count != tc.count {
			t.Fatalf("tick %d: got %d coins/afterimages, want %d", tc.tick, count, tc.count)
		}
	}
	cart, _ := effects.Lookup(effects.CartRevolution)
	for _, tc := range []struct{ tick, count int }{{6, 0}, {8, 2}, {21, 4}, {30, 2}, {40, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawCartRevolutionEffect(frame, ctx, cart.Components[1], m.effects().Active[1], start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len(); count != tc.count {
			t.Fatalf("tick %d: got %d burst meshes, want %d", tc.tick, count, tc.count)
		}
	}
}

func TestMonkBuffsFollowActorEntrySnapshots(t *testing.T) {
	for _, job := range []int16{db.JobMonk, db.JobSuperNovice} {
		ctx := testutil.MonkEffectContext()
		delete(ctx.World.Actors, 300)
		m := &WorldMode{}
		data := make([]byte, 60)
		binary.LittleEndian.PutUint32(data[2:], 300)
		binary.LittleEndian.PutUint16(data[16:], uint16(job))
		applyEntry := func(state uint32) {
			t.Helper()
			binary.LittleEndian.PutUint32(data[44:], state)
			entry, ok, err := network.ParseActorEntry(network.Packet{ID: 0x02EE, Data: data})
			if err != nil || !ok {
				t.Fatalf("parse entry: ok=%t err=%v", ok, err)
			}
			m.upsertNetworkActor(ctx, entry)
		}
		assertAuras := func(active bool) {
			t.Helper()
			for _, id := range []int{effects.Gumgang, effects.SteelBody} {
				if m.effects().HasActive(id, 300, time.Now().Add(time.Hour)) != active {
					t.Fatalf("job %d effect %d active != %t", job, id, active)
				}
			}
		}
		applyEntry(db.Opt3Explosionspirits | db.Opt3Steelbody)
		assertAuras(true)
		original := append([]effects.Instance(nil), m.effects().Active...)
		applyEntry(db.Opt3Explosionspirits | db.Opt3Steelbody)
		m.upsertNetworkActor(ctx, network.ActorEntry{ID: 300, Moving: true, ToX: 15, ToY: 20})
		assertAuras(true)
		if !reflect.DeepEqual(m.effects().Active, original) {
			t.Fatal("actor refresh restarted the auras")
		}
		applyEntry(0)
		assertAuras(false)
	}
}

func TestThrowSpiritSphereSizeUsesSphereCount(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	starts := time.Now()
	for _, count := range []uint16{1, 5} {
		m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillMOFingeroffensive, SkillLevel: 5, HitCount: count}, starts, starts)
	}
	small, large := m.effects().Active[0].Size, m.effects().Active[1].Size
	if math.Abs(large/small-4.0/2.4) > 1e-6 {
		t.Fatalf("sphere sizes = %g, %g; want scaling by the number of spheres used", small, large)
	}
}

func TestMonkAttackEffectsStartAtRelease(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		target []int
		caster []int
	}{
		{db.SkillMOTripleattack, nil, []int{effects.TripleAttack}},
		{db.SkillMOInvestigate, []int{effects.Chimto}, []int{effects.TeiHit2}},
		{db.SkillMOFingeroffensive, nil, []int{effects.Tanji}},
		{db.SkillMOExtremityfist, []int{effects.TeiHit1X}, []int{effects.Quake}},
		{db.SkillMOChaincombo, []int{effects.TeiHit1, effects.ChainCombo}, []int{effects.Gumgang3}},
		{db.SkillMOCombofinish, nil, []int{effects.Gumgang3, effects.HitLine, effects.Quake}},
	} {
		ctx := testutil.MonkEffectContext()
		m := &WorldMode{}
		release := time.Unix(100, 0)
		hit := release.Add(800 * time.Millisecond)
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, SkillLevel: 5, Damage: 100, HitCount: 3}
		m.addSkillEffect(ctx, action, release, hit)
		if len(m.effects().Active) != len(tc.target)+len(tc.caster) {
			t.Fatalf("skill %d effects = %+v", tc.skill, m.effects().Active)
		}
		for _, effect := range m.effects().Active {
			wantActor := uint32(0)
			for _, id := range tc.target {
				if effect.EffectID == id {
					wantActor = 300
				}
			}
			for _, id := range tc.caster {
				if effect.EffectID == id {
					wantActor = 2000000
				}
			}
			if wantActor == 0 || effect.ActorID != wantActor || !effect.Starts.Equal(release) {
				t.Fatalf("skill %d misplaced or delayed effect: %+v", tc.skill, effect)
			}
		}
		if tc.skill == db.SkillMOChaincombo {
			if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\mon_연환.wav" {
				t.Fatalf("Chain Combo sounds = %+v; must not play Fury", m.scheduledSounds)
			}
		}
		if tc.skill == db.SkillMOTripleattack {
			if len(m.scheduledSounds) != 3 {
				t.Fatalf("Triple Attack sounds = %+v", m.scheduledSounds)
			}
			for i, sound := range m.scheduledSounds {
				want := "effect\\ef_hit2.wav"
				if i == 1 {
					want = "effect\\ef_hit4.wav"
				}
				if sound.paths[0] != want || !sound.at.Equal(release.Add(time.Duration(10*(i+1))*db.EffectFrameDuration)) {
					t.Fatalf("Triple Attack sound %d: %+v", i, sound)
				}
			}
		}
	}
}

func TestMonkImpactMarkersAndSplash(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	starts := time.Unix(100, 0)
	for _, tc := range []struct {
		skill uint16
		want  map[int]int
	}{
		{db.SkillMOTripleattack, map[int]int{effects.Hit1: 3}},
		{db.SkillMOInvestigate, map[int]int{effects.Hit1: 3}},
		{db.SkillMOFingeroffensive, map[int]int{effects.Hit1: 3, effects.BashHit: 1}},
		{db.SkillMOExtremityfist, map[int]int{effects.Hit1: 3}},
		{db.SkillMOChaincombo, map[int]int{effects.Hit1: 3, effects.SonicBlowHit: 3}},
		{db.SkillMOCombofinish, map[int]int{effects.Hit1: 3, effects.BashHit: 1}},
		{db.SkillMOBalkyoung, map[int]int{effects.Hit3: 3}},
	} {
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 3}
		m := &WorldMode{}
		m.addSkillHitEffect(ctx, action, starts)
		got := make(map[int]int)
		for _, effect := range m.effects().Active {
			if effect.ActorID != 300 {
				t.Fatalf("skill %d impact on wrong actor", tc.skill)
			}
			got[effect.EffectID]++
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("skill %d hit markers = %v, want %v", tc.skill, got, tc.want)
		}
		action.Action = 5
		m = &WorldMode{}
		m.addSkillEffect(ctx, action, starts, starts)
		m.addSkillHitEffect(ctx, action, starts)
		// The original explicitly exempts Combo Finish from ACTION_SPLASH suppression.
		if (len(m.effects().Active) > 0) != (tc.skill == db.SkillMOCombofinish) {
			t.Fatalf("skill %d splash effects = %+v", tc.skill, m.effects().Active)
		}
	}
}

func TestBladeStopCapturesVictimsPlayingFrame(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	actions := make([]res.ACTAction, 13*8)
	for i := range actions {
		actions[i] = res.ACTAction{DelayMS: 100, Animations: make([]res.ACTAnimation, 4)}
	}
	started := time.Now().Add(-150 * time.Millisecond)
	m := &WorldMode{actorSystem: actors.System{PlayerView: &sprites.Humanoid{Body: &scene.SpriteView{Act: &res.ACT{Actions: actions}}, Started: started}}}
	m.setCombatActorAction(ctx, 2000000, scene.ActorAnimation{ActionFamily: sprites.ActionPCAttack1, Started: started, Play: true, HasPlay: true, Duration: time.Second})
	m.applyActorBladeStop(ctx, network.ActorBladeStop{SourceID: 300, TargetID: 2000000, Active: true})
	anim := m.actorSystem.ActorAnims[150000]
	want := scene.MinInt(3, int(anim.Started.Sub(started)/(100*time.Millisecond)))
	if want == 0 || anim.FixedMotion != want || anim.Play {
		t.Fatalf("victim pose = %+v, want frozen frame %d", anim, want)
	}
	if later, ok := m.actors().ActorAnimation(150000, time.Now().Add(time.Minute)); !ok || later.FixedMotion != want {
		t.Fatal("Blade Stop victim resumed animation before release")
	}
}

func TestNinjaWindAndPetalsAreFrameRateIndependent(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: 540, HitCount: 3, SourceID: 2000000, TargetID: 300}, start)
	e := m.effects().Active[0]
	var stepped, jumped effects.NinjaWind
	for tick := 0; tick < 16; tick++ {
		stepped.Advance(ctx, e, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	jumped.Advance(ctx, e, start.Add(15*db.EffectFrameDuration))
	if !reflect.DeepEqual(stepped, jumped) {
		t.Fatal("wind path depends on frame rate")
	}
	if stepped.Heads[0].Samples[0].Position.Z == stepped.Heads[1].Samples[0].Position.Z {
		t.Fatal("alternating wind blades must curve on different sides")
	}
	delete(ctx.World.Actors, 300)
	stepped.Advance(ctx, e, start.Add(100*db.EffectFrameDuration))
	for _, h := range stepped.Heads {
		for _, s := range h.Samples {
			if s.Alpha != 0 {
				t.Fatal("wind did not fade after losing its target")
			}
		}
	}
	ctx = testutil.ArcherEffectContext()
	m = &WorldMode{}
	m.effects().AddAt(ctx, effects.Kouenka, 300, start)
	e = m.effects().Active[0]
	var a, b effects.NinjaPetals
	for tick := 0; tick < 30; tick++ {
		a.Advance(ctx, e, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	b.Advance(ctx, e, start.Add(29*db.EffectFrameDuration))
	if !reflect.DeepEqual(a, b) {
		t.Fatal("petals depend on frame rate")
	}
	if a.Particles[0].Position.Y <= e.ActorOrigin.Y || a.Particles[19].Age != 11 {
		t.Fatal("petals must rise and appear one per tick")
	}
}

func TestMagicCrasherBlendReachesActorDraws(t *testing.T) {
	sprite := &scene.SpriteView{
		Spr:    &res.SPR{Frames: []res.SPRFrame{{Type: res.SPRFrameRGBA, Width: 8, Height: 8, Data: testutil.SolidRGBAFrame(8, 8)}}},
		Act:    &res.ACT{Actions: []res.ACTAction{{DelayMS: 100, Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{Index: 0, SPRType: res.SPRFrameRGBA, ScaleX: 1, ScaleY: 1, Color: [4]float32{1, 1, 1, 1}}}}}}}},
		Images: make(map[scene.SpriteFrameKey]*render.Image), Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}
	mode := &WorldMode{actorSystem: actors.System{PlayerView: &sprites.Humanoid{Body: sprite}, NonPCViews: map[int]*scene.SpriteView{1002: sprite}}}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 42, X: 10, Y: 20}
	ctx := client.Context{World: world, Session: &session.Session{AccountID: 43, CharID: 44}}
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	for _, target := range []uint32{42, 43, 44, 99} {
		for _, expired := range []bool{false, true} {
			now := time.Now()
			effect := effects.Instance{EffectID: effects.MagicCrasher, ActorID: target, Starts: now.Add(-700 * time.Millisecond), Expires: now.Add(time.Second)}
			if expired {
				effect.Expires = now.Add(-time.Millisecond)
			}
			mode.effects().Active = []effects.Instance{effect}
			for _, local := range []bool{false, true} {
				for _, stealth := range []scene.StealthView{scene.StealthVisible, scene.StealthSilhouette} {
					entry := scene.ActorDrawEntry{Actor: worldstate.Actor{ID: 99, Job: 1002}, WorldX: 10.5, WorldY: 20.5, Scale: 1, Stealth: stealth}
					frame := render.NewFrame(800, 600)
					var drawn bool
					if local {
						entry.Actor = world.Player
						drawn = mode.actors().DrawPlayerSprite3D(ctx, frame, projection, entry, 0, 0, 1, 1)
					} else {
						drawn = mode.actors().DrawNonPCSprite3D(frame, ctx, projection, entry, 0, 1)
					}
					commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
					if !drawn || commands.Len() != 1 {
						t.Fatalf("local=%t: missing actor billboard", local)
					}
					want := render.BlendSourceOver
					if !expired && stealth == scene.StealthVisible && (local == (target != 99)) {
						want = render.BlendLighter
					}
					options := commands.Index(0).FieldByName("Options")
					if got := render.Blend(options.FieldByName("Blend").Int()); got != want {
						t.Fatalf("target=%d local=%t expired=%t stealth=%d: blend %v, want %v", target, local, expired, stealth, got, want)
					}
					if !options.FieldByName("DepthTest").Bool() || !options.FieldByName("DepthWrite").Bool() {
						t.Fatal("flash changed actor depth options")
					}
				}
			}
		}
	}
}

func TestPriestCasting(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, skill := range []uint16{
		db.SkillMGSafetywall, db.SkillALLResurrection, db.SkillPRImpositio, db.SkillPRSuffragium,
		db.SkillPRAspersio, db.SkillPRBenedictio, db.SkillPRSanctuary, db.SkillPRStrecovery,
		db.SkillPRKyrie, db.SkillPRMagnificat, db.SkillPRGloria, db.SkillPRLexdivina,
		db.SkillPRTurnundead, db.SkillPRLexaeterna, db.SkillPRMagnus,
		db.SkillHPAssumptio, db.SkillHPBasilica, db.SkillPRRedemptio,
	} {
		m := &WorldMode{}
		// Ordinary auras follow the property; High Priest and Redemptio stay white.
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: 3})
		aura := effects.BeginSpell3
		if skill == db.SkillHPAssumptio || skill == db.SkillHPBasilica || skill == db.SkillPRRedemptio {
			aura = effects.BeginSpell6
		}
		if !m.effects().HasActive(aura, 2000000, time.Now()) {
			t.Fatalf("skill %d: wrong cast aura", skill)
		}
		pose := sprites.ActionPCReadyFight
		if skill == db.SkillPRRedemptio {
			pose = scene.SpriteActionIdle
		}
		if m.actorSystem.ActorAnims[2000000].ActionFamily != pose {
			t.Fatalf("skill %d: wrong cast pose", skill)
		}
	}
}

func TestProfessorClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobSageH, db.JobSageH
	start := time.Unix(100, 0)
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.Body.ActSource, res.PlayerJobToken(db.JobSageH)) {
			t.Fatalf("must load the Professor body without fallback: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			for _, family := range []int{sprites.ActionPCReadyFight, sprites.ActionPCSkill} {
				if _, a, ok := sprites.ResolveAction(view.Body.Act, family, dir); !ok || len(a.Animations) == 0 {
					t.Fatal("Professor casting/release ACT is missing")
				}
			}
		}
	}
	for _, skill := range []uint16{373, 374, 375, 402, 403, 404, 405, 482} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Property: 3, DelayTime: 1000})
		aura := 55
		if skill == 404 {
			aura = 16
		}
		if m.actorSystem.ActorAnims[2000000].ActionFamily != sprites.ActionPCReadyFight || len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != aura {
			t.Fatalf("skill %d needs casting aura %d and the ready pose", skill, aura)
		}
	}
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct{ effect, tick, billboards, vertices int }{
		{383, 0, 21, 0}, {383, 4, 105, 0}, {383, 60, 84, 0}, {383, 64, 0, 0},
		{384, 19, 0, 80}, {384, 77, 0, 0}, {404, 90, 0, 4}, {405, 90, 0, 24},
	} {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, tc.effect, 2000000, 300, start)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		commands := v.FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			vertices += commands.Index(i).FieldByName("Vertices").Len()
		}
		if v.FieldByName("worldBillboards").Len() != tc.billboards || vertices != tc.vertices {
			t.Fatalf("effect %d tick %d: billboards=%d vertices=%d", tc.effect, tc.tick, v.FieldByName("worldBillboards").Len(), vertices)
		}
		if tc.effect == 405 {
			cmd := commands.Index(0)
			if cmd.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendSourceOver) {
				t.Fatal("Wall of Fog needs translucent grey walls, not additive light")
			}
		}
	}
	m := &WorldMode{}
	for _, file := range []string{"소울번", "사람효과"} {
		if m.effects().LoadWorldEffectSTR(ctx.Resources, file, "") == nil {
			t.Fatalf("missing native STR %s", file)
		}
	}
	for _, sound := range []string{"소울 체인지", "매직 크래쉬", "마법력 증폭", "swordman_provoke", "priest_suffragium"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + sound + ".wav"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWorldDrawPlacesEffectsBeforeAndAfterEntities(t *testing.T) {
	const effectID = -1
	db.EffectSpecs[effectID] = db.EffectSpec{Components: []db.EffectComponent{
		{Kind: db.EffectComponentSPR, SpriteFile: "after", Duration: time.Hour},
		{Kind: db.EffectComponentSPR, SpriteFile: "before", Duration: time.Hour, RenderBefore: true},
	}}
	t.Cleanup(func() { delete(db.EffectSpecs, effectID) })
	for _, terrain := range []string{"gat", "gnd", "models"} {
		t.Run(terrain, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 42, X: 1, Y: 1}
			world.Actors[100] = worldstate.Actor{ID: 100, Job: 1002, X: 1, Y: 1}
			world.GAT = &res.GAT{Width: 4, Height: 4, Cells: make([]res.GATCell, 16)}
			if terrain != "gat" {
				world.GND = &res.GND{Width: 2, Height: 2, Cells: make([]res.GNDCell, 4)}
			}
			if terrain == "models" {
				world.RSW = &res.RSW{}
				world.RSM = map[string]*res.RSM{"empty": {}}
			}
			ctx := client.Context{World: world, Resources: &res.Manager{}, Session: &session.Session{CharID: 42}}
			ctx.Config.Render.NoUI = true
			before, actor, after := testutil.EffectRenderTestSprite(), testutil.EffectRenderTestSprite(), testutil.EffectRenderTestSprite()
			starts := time.Now().Add(-time.Second)
			mode := &WorldMode{effectSystem: effects.System{EffectViews: map[string]*scene.SpriteView{"before": before, "after": after}, Active: []effects.Instance{
				{EffectID: effectID, X: 1, Y: 1, Starts: starts, Expires: starts.Add(time.Hour)},
				{EffectID: effectID, X: 1, Y: 1, Starts: starts, Expires: starts},
				{EffectID: effectID, X: 1, Y: 1, Starts: starts.Add(time.Hour)},
			}}, actorSystem: actors.System{NonPCViews: map[int]*scene.SpriteView{1002: actor}}}
			frame := render.NewFrame(800, 600)
			mode.Draw(ctx, frame)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if commands.Len() != 3 {
				t.Fatalf("draws = %d, want one effect behind, one actor, one effect in front", commands.Len())
			}
			for i, view := range []*scene.SpriteView{before, actor, after} {
				want := reflect.ValueOf(view.Billboards[scene.SpriteBillboardKey{}].Image).Pointer()
				if commands.Index(i).FieldByName("Texture").Pointer() != want {
					t.Fatalf("draw %d has wrong texture/order", i)
				}
			}
			if len(mode.effects().Active) != 2 {
				t.Fatal("expired effects should be removed and scheduled effects retained")
			}
		})
	}
}

func TestRogueAttackOwnersAndTiming(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	starts := time.Unix(100, 0)
	for _, skill := range []uint16{db.SkillRGBackstap, db.SkillRGIntimidate} {
		m := &WorldMode{}
		// The action effect also plays on a miss.
		action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 0}
		m.addSkillEffect(ctx, action, starts, starts.Add(time.Second))
		m.addSkillHitEffect(ctx, action, starts.Add(time.Second))
		if len(m.effects().Active) != 1 {
			t.Fatalf("skill %d: effects = %+v", skill, m.effects().Active)
		}
		effect := m.effects().Active[0]
		if effect.ActorID != action.SourceID || !effect.Starts.Equal(starts) {
			t.Fatalf("skill %d must start on the caster at action time: %+v", skill, effect)
		}
		if skill == db.SkillRGBackstap {
			if effect.ActorOrigin.X != 16.5 || effect.ProjectileDirection.X != 6 || effect.ProjectileDirection.Z != 0 {
				t.Fatalf("Back Stab must start at the victim and continue away from the caster: %+v", effect)
			}
			if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(starts.Add(20*db.EffectFrameDuration)) {
				t.Fatalf("Back Stab sound = %+v", m.scheduledSounds)
			}
		} else {
			if effect.ActorOrigin.X != 10.5 || len(m.scheduledSounds) != 2 ||
				!m.scheduledSounds[0].at.Equal(starts) || !m.scheduledSounds[1].at.Equal(starts.Add(35*db.EffectFrameDuration)) {
				t.Fatalf("Snatch origin or sounds: %+v, %+v", effect, m.scheduledSounds)
			}
		}
	}
}

func TestSageCastColors(t *testing.T) {
	for _, tc := range []struct {
		skill    uint16
		property uint32
		effect   int
	}{
		{db.SkillSAVolcano, 3, effects.BeginSpell3},
		{db.SkillSADeluge, 1, effects.BeginSpell2},
		{db.SkillSAViolentgale, 4, effects.BeginSpell5},
		{db.SkillSALandprotector, 2, effects.BeginSpell4},
		{db.SkillSAAutospell, 0, effects.BeginSpell},
		{db.SkillSASpellbreaker, 0, effects.BeginSpell},
		{db.SkillSADispell, 0, effects.BeginSpell},
		{db.SkillSAAbracadabra, 0, effects.BeginSpell},
		{db.SkillSAElementwater, 0, effects.BeginSpell2},
		{db.SkillSAElementfire, 0, effects.BeginSpell3},
		{db.SkillSAElementground, 0, effects.BeginSpell4},
		{db.SkillSAElementwind, 0, effects.BeginSpell5},
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{
			SkillID: tc.skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: tc.property,
		})
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != tc.effect {
			t.Fatalf("skill %d: wrong casting color for property %d", tc.skill, tc.property)
		}
	}
}

func TestSageRealAssets(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{
		effects.Volcano, effects.Deluge, effects.ViolentGale, effects.LandProtector,
		effects.BottomVolcano, effects.BottomDeluge, effects.BottomViolent, effects.BottomLand,
		effects.MagicRod, effects.SpellBreaker, effects.Dispell,
		effects.FlameLauncher, effects.FrostWeapon, effects.LightningLoad, effects.SeismicWeapon,
	} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.effects().AddAt(ctx, id, 300, start) {
				t.Fatal("missing effect")
			}
			for _, sound := range m.scheduledSounds {
				for _, path := range sound.paths {
					if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
						t.Fatal(err)
					}
				}
			}
			draws := 0
			for _, tick := range []int{0, 15, 60, 150} {
				frame := render.NewFrame(800, 600)
				m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 20, 20, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
				v := reflect.ValueOf(frame).Elem()
				draws += v.FieldByName("worldCommands").Len() + v.FieldByName("worldBillboards").Len() + v.FieldByName("commands").Len()
			}
			if draws == 0 {
				t.Fatal("effect produced no geometry with real assets")
			}
		})
	}
}

func TestSniperEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for _, id := range []int{effects.TrueSight, effects.Portal4, effects.TripleAttack2} {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
			if id == effects.TripleAttack2 && sound.at.Sub(start) != 30*db.EffectFrameDuration {
				t.Fatal("Sharp Shooting sound should wait 30 ticks")
			}
		}
		at := 10
		if id == effects.TripleAttack2 {
			at = 60
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(at)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing", id)
		}
		frame = render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(90*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 {
			t.Fatalf("effect %d still drew after particles faded", id)
		}
	}
	effect := effects.Instance{ActorOrigin: scene.Point3{X: 10}, ProjectileDirection: scene.Point3{Z: 8}, Starts: start}
	for i := 0; i < 16; i++ {
		from, to, _ := effects.SharpShootingLine(effect, i, 60)
		if math.Abs(to.X-from.X) > 1e-6 || to.Z-from.Z < 8 || from.Y < 1.6 || from.Y > 2.4 {
			t.Fatal("Sharp Shooting rays have incorrect direction, length, or height")
		}
		_, _, alpha := effects.SharpShootingLine(effect, i, 40)
		if alpha != 0 {
			t.Fatal("Sharp Shooting ray appeared before native delay")
		}
	}
}

func TestSoulLinkerSkillSequenceMatchesOriginalClient(t *testing.T) {
	for _, tc := range []struct {
		skill              uint16
		projectile, impact int
	}{
		{db.SkillSLStin, effects.Stin, effects.QuakeBody3},
		{db.SkillSLStun, effects.Stin3, effects.HitLine4},
	} {
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		starts := time.Unix(100, 0)
		hitAt := starts.Add(500 * time.Millisecond)
		action := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: tc.skill, Damage: 10}
		m.addSkillEffect(ctx, action, starts, hitAt)
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != 3 {
			t.Fatalf("skill %d: effects = %+v", tc.skill, m.effects().Active)
		}
		for i, id := range []int{tc.projectile, tc.impact, effects.BlueHit} {
			effect := m.effects().Active[i]
			want := starts
			if i == 2 {
				want = hitAt
			}
			if effect.EffectID != id || effect.ActorID != 300 || !effect.Starts.Equal(want) {
				t.Fatalf("skill %d: wrong effect order, owner or timing: %+v", tc.skill, effect)
			}
		}
		projectile := m.effects().Active[0]
		if projectile.ActorOrigin.X != 10.5 || projectile.ActorOrigin.Z != 20.5 || projectile.ProjectileDirection != (scene.Point3{X: 1}) {
			t.Fatalf("projectile starts at target instead of caster: %+v", projectile)
		}
		if len(m.scheduledSounds) != 1 {
			t.Fatalf("skill %d: sounds = %+v", tc.skill, m.scheduledSounds)
		}
		m.effects().Active = nil
		action.Damage = 0
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != 0 {
			t.Fatal("a miss must not flash the target blue")
		}
	}
}

func TestEstinNativeMotionAndTrail(t *testing.T) {
	distance, alpha := effects.EstinHead(10)
	wantDistance, speed := .8, .8
	for i := 2; i <= 10; i++ {
		speed *= 1.02
		wantDistance += speed
	}
	if math.Abs(distance-wantDistance/5) > 1e-9 || alpha != 225.0/255 {
		t.Fatalf("end of rise = %g/%g", distance, alpha)
	}
	if _, alpha := effects.EstinHead(55); alpha != 0 {
		t.Fatal("Estin head should finish fading at tick 55")
	}
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	starts := time.Unix(100, 0)
	m.effects().AddBetween(ctx, effects.Stin, 300, 2000000, starts)
	projection := testutil.Projection(800, 600, 14.5, 20.5, 0)
	draw := func() *render.Frame {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, starts.Add(16*db.EffectFrameDuration), false)
		return frame
	}
	frame := draw()
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("Estin should draw one batched wind trail")
	}
	vertices := commands.Index(0).FieldByName("Vertices")
	if vertices.Len() != 16 {
		t.Fatalf("trail vertices=%d, want four squares", vertices.Len())
	}
	for i := 0; i < 16; i++ {
		if math.Abs(vertices.Index(i).FieldByName("Y").Float()-m.effects().Active[0].ActorOrigin.Y) > 1e-5 {
			t.Fatal("Estin must use world-horizontal quads as in RenderSTIN")
		}
	}
	ctx.World.Player.X, ctx.World.Player.Y = 50, 50
	delete(ctx.World.Actors, 300)
	again := reflect.ValueOf(draw()).Elem().FieldByName("worldCommands").Index(0).FieldByName("Vertices")
	for i := 0; i < vertices.Len(); i++ {
		for _, axis := range []string{"X", "Y", "Z"} {
			if vertices.Index(i).FieldByName(axis).Float() != again.Index(i).FieldByName(axis).Float() {
				t.Fatal("movement/removal bent an already launched projectile")
			}
		}
	}
}

func TestEstunEmitsParticlesAndStops(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	starts := time.Unix(100, 0)
	m.effects().AddBetween(ctx, effects.Stin3, 300, 2000000, starts)
	effect := m.effects().Active[0]
	spec, _ := effects.Lookup(effects.Stin3)
	projection := testutil.Projection(800, 600, 14.5, 20.5, 0)
	for _, tc := range []struct{ frame, count int }{{-1, 0}, {0, 0}, {1, 4}, {10, 40}, {199, 196}, {230, 76}, {250, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawRisingBuffParticles(frame, ctx, projection, spec.Components[0], effect, starts.Add(time.Duration(tc.frame)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.count {
			t.Fatalf("tick %d: %d particles, want %d", tc.frame, draws.Len(), tc.count)
		}
		for i := 0; i < draws.Len(); i++ {
			if draws.Index(i).FieldByName("Center").Index(1).Float() <= effect.ActorOrigin.Y {
				t.Fatal("particles must rise above the emitter")
			}
		}
	}
}

func TestEstunImpactStreaksHaveBoundedLifetime(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	spec, _ := effects.Lookup(effects.HitLine4)
	starts := time.Unix(100, 0)
	effect := effects.Instance{EffectID: effects.HitLine4, ActorID: 300, Starts: starts}
	for _, tick := range []int{5, 30, 80, 200} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawHitLineBurst(frame, ctx, spec.Components[0], effect, 18.5, 20.5, 0, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if (commands.Len() > 0) != (tick == 30) {
			t.Fatalf("tick %d: impact draws=%d", tick, commands.Len())
		}
		if tick == 30 {
			vertices := commands.Index(0).FieldByName("Vertices")
			indices := commands.Index(0).FieldByName("Indices")
			for i := 0; i < indices.Len(); i++ {
				if int(indices.Index(i).Uint()) >= vertices.Len() {
					t.Fatal("invalid streak mesh index")
				}
			}
		}
	}
}

func TestEsmaSequenceAndHoming(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillSLSma, Damage: 100}, starts, starts.Add(500*time.Millisecond))
	if len(m.effects().Active) != 4 {
		t.Fatalf("Esma effects = %+v", m.effects().Active)
	}
	for i, id := range []int{effects.FourWayBody, effects.Stin2, effects.HitLine6, effects.HitTexture} {
		if e := m.effects().Active[i]; e.EffectID != id || e.ActorID != 300 || !e.Starts.Equal(starts) {
			t.Fatalf("wrong Esma component/timing: %+v", e)
		}
	}
	if len(m.scheduledSounds) != 5 {
		t.Fatalf("Esma sounds = %+v", m.scheduledSounds)
	}
	for i, sound := range m.scheduledSounds {
		if !sound.at.Equal(starts.Add(time.Duration(5+15*i) * db.EffectFrameDuration)) {
			t.Fatalf("wrong Esma sound timing: %+v", sound)
		}
	}
	e := m.effects().Active[1]
	e.Esma.Advance(ctx, e, starts.Add(10*db.EffectFrameDuration))
	left, right := e.Esma.Heads[0].Samples[0], e.Esma.Heads[1].Samples[0]
	if left.Position.Z <= e.ActorOrigin.Z || right.Position.Z >= e.ActorOrigin.Z || math.Abs(left.Position.X-right.Position.X) > 1e-8 {
		t.Fatalf("Esma must leave the caster in opposite arcs: %v %v", left, right)
	}
	// A second render pass in the same tick must not advance the simulation.
	before := *e.Esma
	e.Esma.Advance(ctx, e, starts.Add(10*db.EffectFrameDuration))
	if *e.Esma != before {
		t.Fatal("drawing twice advanced Esma")
	}
	actor := ctx.World.Actors[300]
	actor.X, actor.Y = 24, 30
	ctx.World.Actors[300] = actor
	e.Esma.Advance(ctx, e, starts.Add(11*db.EffectFrameDuration))
	wantHistory := before.Heads[0].Samples[1]
	wantHistory.Alpha *= .8
	if e.Esma.Heads[0].Samples[2] != wantHistory {
		t.Fatal("moving target rewrote an old trail position")
	}
	for i := range e.Esma.Heads {
		step := scene.Sub3(e.Esma.Heads[i].Samples[0].Position, before.Heads[i].Samples[0].Position)
		if math.Abs(math.Hypot(step.X, step.Z)-.4) > 1e-8 {
			t.Fatal("moving target teleported an Esma head")
		}
	}
	delete(ctx.World.Actors, 300)
	e.Esma.Advance(ctx, e, starts.Add(100*db.EffectFrameDuration))
	for _, head := range e.Esma.Heads {
		for _, sample := range head.Samples {
			if sample.Alpha > 0 {
				t.Fatal("Esma must fade when its target leaves")
			}
		}
	}
}

func TestSoulLinkerParticleAndKaizelTiming(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
	for _, tc := range []struct{ id, tick, count int }{
		{effects.Hated, 0, 0}, {effects.Hated, 1, 4}, {effects.Hated, 5, 8}, {effects.Hated, 135, 0},
		{effects.SmaReady, 39, 0}, {effects.SmaReady, 41, 4}, {effects.SmaReady, 175, 0},
	} {
		frame := render.NewFrame(800, 600)
		spec, _ := effects.Lookup(tc.id)
		m.effects().AddAt(ctx, tc.id, 300, starts)
		effect := m.effects().Active[len(m.effects().Active)-1]
		m.effects().DrawRisingBuffParticles(frame, ctx, projection, spec.Components[0], effect, starts.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); n != tc.count {
			t.Fatalf("effect %d tick %d: got %d particles, want %d", tc.id, tc.tick, n, tc.count)
		}
	}
	spec, _ := effects.Lookup(effects.Kaizel)
	for _, tick := range []int{14, 15, 25, 45} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawKaizelEffect(frame, ctx, spec.Components[1], effects.Instance{Starts: starts}, 18.5, 20.5, 0, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if (commands.Len() > 0) != (tick == 15 || tick == 25) {
			t.Fatalf("Kaizel burst at wrong tick %d", tick)
		}
		if commands.Len() > 0 && commands.Index(0).FieldByName("Vertices").Len() != 8*3*4 {
			t.Fatal("Kaizel needs eight blades of three quads each")
		}
	}
}

func TestSoulLinkerBuffParticlesKeepEmissionPositions(t *testing.T) {
	for _, id := range []int{effects.Hated, effects.Kaizel, effects.SmaReady} {
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		soulLinkerTestTextures(m)
		starts := time.Unix(100, 0)
		m.effects().AddAt(ctx, id, 300, starts)
		projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
		first := 0
		if id == effects.SmaReady {
			first = 40
		}
		draw := func(tick int) reflect.Value {
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, projection, starts.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			return reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		}
		before := draw(first + 1)
		actor := ctx.World.Actors[300]
		actor.X += 10
		ctx.World.Actors[300] = actor
		after := draw(first + 5)
		if before.Len() != 4 || after.Len() != 8 {
			t.Fatalf("effect %d: particles before/after movement = %d/%d", id, before.Len(), after.Len())
		}
		for i := 0; i < 4; i++ {
			for _, axis := range []int{0, 2} {
				if before.Index(i).FieldByName("Center").Index(axis).Float() != after.Index(i).FieldByName("Center").Index(axis).Float() {
					t.Fatalf("effect %d: movement relocated an emitted particle", id)
				}
			}
			if after.Index(i+4).FieldByName("Center").Index(0).Float() < 27 {
				t.Fatalf("effect %d: new particles did not follow the actor", id)
			}
		}
	}
}

func TestSoulLinkerMonsterSkillSuccessOwnsActivation(t *testing.T) {
	for _, skill := range []uint16{db.SkillSLSwoo, db.SkillSLSke, db.SkillSLSka} {
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		m.handleNetworkPacket(ctx, testutil.SoulLinkerSkillPacket(skill, 0), time.Now())
		if ctx.World.Actors[300].Opt3State != 0 || len(m.effects().Active) != 0 {
			t.Fatalf("failed skill %d changed monster effects", skill)
		}
		m.handleNetworkPacket(ctx, testutil.SoulLinkerSkillPacket(skill, 1), time.Now())
		if ctx.World.Actors[300].Opt3State == 0 {
			t.Fatalf("skill %d still requires a status-icon packet", skill)
		}
		m.upsertNetworkActor(ctx, network.ActorEntry{ID: 300, Job: 1002, X: 18, Y: 20, HasState: true, HasOpt3State: true})
		if m.effects().HasActive(effects.BabyBody, 300, time.Now()) || m.effects().HasActive(effects.SteelBody, 300, time.Now()) || m.effects().HasActive(effects.AsuraBodyMonster, 300, time.Now()) {
			t.Fatalf("skill %d ignored a newer actor snapshot", skill)
		}
	}
}

func TestKaupeRendersEveryDodge(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	var w network.Writer
	w.Uint16(network.PacketZCNotifyEffect2)
	w.Uint32(300)
	w.Uint32(effects.StormKick4)
	packet := network.Packet{ID: network.PacketZCNotifyEffect2, Data: w.Bytes()}
	m.handleNetworkPacket(ctx, packet, time.Now())
	first := m.effects().Active[0]
	m.handleNetworkPacket(ctx, packet, time.Now())
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 2 || m.effects().Active[0] != first || m.effects().Active[0].SoulLights == m.effects().Active[1].SoulLights {
		t.Fatal("successive dodges must have independent effects and sounds")
	}
}

func TestSoulLinkAndKaupeParticleTimingIndependentOfFrameRate(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	for _, id := range []int{effects.SoulLink, effects.StormKick4} {
		m := &WorldMode{}
		soulLinkerTestTextures(m)
		start := time.Unix(100, 0)
		m.effects().AddAt(ctx, id, 300, start)
		e := m.effects().Active[0]
		one, many := &effects.SoulLights{}, &effects.SoulLights{}
		right := scene.Point3{X: 1}
		one.Advance(ctx, e, right, start.Add(100*db.EffectFrameDuration))
		for i := 0; i <= 100; i++ {
			many.Advance(ctx, e, right, start.Add(time.Duration(i)*db.EffectFrameDuration))
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatalf("effect %d depends on render rate", id)
		}
		projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(30*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing", id)
		}
		e.SoulLights.Advance(ctx, e, right, start.Add(300*db.EffectFrameDuration))
		if len(e.SoulLights.Particles) != 0 {
			t.Fatalf("effect %d particles never die", id)
		}
		if id == effects.StormKick4 && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].at.Sub(start) != 15*db.EffectFrameDuration) {
			t.Fatal("Kaupe dodge sound timing")
		}
	}
	for _, tc := range []struct {
		age, layer int
		alpha      float64
	}{{0, 10, 0}, {19, 10, 240.0 / 255}, {100, 10, 240.0 / 255}, {148, 10, 0}, {129, 8, 0}} {
		_, a := effects.SoulLinkGlyph(tc.age, tc.layer)
		if math.Abs(a-tc.alpha) > 1e-8 {
			t.Fatalf("glyph age=%d layer=%d alpha=%v", tc.age, tc.layer, a)
		}
	}
}

func TestStarGladiatorNativeAssetsAndGeometry(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Now()
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, tc := range []struct{ id, tick, end int }{{418, 50, -1}, {572, 20, 301}, {488, 90, 205}, {487, 20, 151}, {475, 30, -1}, {484, 30, -1}} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, tc.id, 2000000, start)
		for _, tick := range []int{tc.tick, tc.end} {
			if tick < 0 {
				continue
			}
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			v := reflect.ValueOf(frame).Elem()
			commands := v.FieldByName("worldCommands")
			count := commands.Len() + v.FieldByName("worldBillboards").Len()
			if (count > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d draws=%d", tc.id, tick, count)
			}
			if tc.id == 488 && tick == 90 {
				mesh := commands.Index(0).FieldByName("Vertices")
				if mesh.Len() != 8*21*4 {
					t.Fatalf("Comfort must have eight 315-degree rings: %d vertices", mesh.Len())
				}
			}
		}
		for _, sound := range m.scheduledSounds {
			for _, path := range sound.paths {
				if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for _, path := range []string{"T_변신.wav", "T_등록.wav", "_blind.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + path); err != nil {
			t.Fatal(err)
		}
	}
	ctx.Session.Selected.Job = db.JobStar2
	for _, sex := range []uint8{0, 1} {
		if view, message := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false); view == nil {
			t.Fatal(message)
		}
	}
}

func TestBashImpactOnlyOnHit(t *testing.T) {
	ctx := testutil.SwordmanContext()
	starts := time.Unix(100, 0)
	for _, damage := range []int32{0, 50} {
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: db.SkillSMBash, SourceID: 2000000, TargetID: 300, Damage: damage}
		m.addSkillBeginEffect(ctx, action, starts)
		m.addSkillHitEffect(ctx, action, starts.Add(200*time.Millisecond))
		want := 1
		if damage > 0 {
			want = 2
		}
		if len(m.effects().Active) != want || m.effects().Active[0].EffectID != effects.BashBegin || m.effects().Active[0].ActorID != 2000000 {
			t.Fatalf("damage %d: effects=%+v", damage, m.effects().Active)
		}
		if damage > 0 && (m.effects().Active[1].ActorID != 300 || m.effects().Active[1].EffectID != effects.BashHit || !m.effects().Active[1].Starts.Equal(starts.Add(200*time.Millisecond))) {
			t.Fatal("Bash impact did not follow the victim/hit time")
		}
	}
}

func TestTaekwonKickEffectsHaveNativeOwnershipAndTiming(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill   uint16
		effects map[int]uint32
	}{
		{413, map[int]uint32{}}, // The no-damage packet owns Tornado Kick.
		{415, map[int]uint32{effects.PressedBody: 300, effects.HitLine6: 300}},
		{417, map[int]uint32{effects.SpinedBody2: 300, effects.HitLine4: 300}},
		{419, map[int]uint32{effects.KickedBody: 300, effects.HitLine5: 300}},
		{421, map[int]uint32{effects.QuakeBody2: 300, effects.Chemical3: 2000000, effects.JumpKick: 2000000}},
	} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 10, HitCount: 1, Action: network.ActorActionSkill}
		m.addSkillEffect(ctx, a, start, start.Add(time.Second))
		got := map[int]uint32{}
		for _, e := range m.effects().Active {
			got[e.EffectID] = e.ActorID
			if e.Starts != start {
				t.Fatal("kick effects must start when the attack begins")
			}
			if e.EffectID == effects.Chemical3 && (e.TargetID != 300 || e.ProjectileDirection.X <= 0) {
				t.Fatal("Flying Kick streaks must point from caster to victim")
			}
		}
		if !reflect.DeepEqual(got, tc.effects) {
			t.Fatalf("skill %d effects: %v want %v", tc.skill, got, tc.effects)
		}
		m.effects().Active = nil
		m.addSkillHitEffect(ctx, a, start.Add(time.Second))
		if len(m.effects().Active) != 0 {
			t.Fatal("Taekwon skills must not add the ordinary hit flash")
		}
	}
}

func TestStoneFlingFlight(t *testing.T) {
	ctx := testutil.ThiefEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillTFThrowstone, SourceID: 2000000, TargetID: 300}, starts)
	effect := m.effects().Active[0]
	first, angle, alpha := effects.ThrowItemAt(effect, 1)
	if math.Abs(first.X-10.9) > 1e-9 || angle != scene.DegreesToRadians(10) || alpha != 255 {
		t.Fatalf("stone must advance two native units and rotate ten degrees per tick: %+v, %g, %d", first, angle, alpha)
	}
	middle, _, _ := effects.ThrowItemAt(effect, 8)
	if math.Abs(middle.Y-effect.ActorOrigin.Y-2) > 1e-9 {
		t.Fatal("stone's arc must rise ten native units above its launch height")
	}
	near := effect
	near.ProjectileDirection.X = 1.2
	_, _, nearAlpha := effects.ThrowItemAt(near, 10)
	_, _, farAlpha := effects.ThrowItemAt(effect, 10)
	if nearAlpha >= farAlpha {
		t.Fatal("stone flight duration must depend on distance")
	}
	m.mapScene.Textures = map[string]*render.Image{"__effectfile_유저인터페이스/item/돌.bmp|blackkey=true": render.NewImage(8, 8)}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 14, 20, 0), starts.Add(6*db.EffectFrameDuration), false)
	if n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); n != 4 {
		t.Fatalf("stone and three afterimages: got %d", n)
	}
}

func TestWhitesmithEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct {
		effect int
		sound  string
	}{
		{effects.Meltdown, "effect\\black_overthrust.wav"}, {effects.CartBoost, "effect\\ef_incagility.wav"},
	} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, tc.effect, 300, start)
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != tc.sound || !m.scheduledSounds[0].at.Equal(start) {
			t.Fatal("buff must play its native sound on activation")
		}
		if _, err := ctx.Resources.ReadFile("data/wav/" + tc.sound); err != nil {
			t.Fatal(err)
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("commands").Len()+v.FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing with client STR assets", tc.effect)
		}
	}
	m := &WorldMode{}
	m.effects().AddAt(ctx, effects.CartTermination, 300, start)
	if _, err := ctx.Resources.ReadFile("data/wav/" + m.scheduledSounds[0].paths[0]); err != nil {
		t.Fatal(err)
	}
	for _, tick := range []int{29, 55, 93} {
		frame := render.NewFrame(800, 600)
		at := start.Add(time.Duration(tick) * db.EffectFrameDuration)
		m.effects().Draw(frame, ctx, projection, at, false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick != 55 {
			if commands.Len() != 0 {
				t.Fatalf("unexpected Cart Termination particles at tick %d", tick)
			}
			continue
		}
		if commands.Len() != 1 {
			t.Fatal("Cart Termination did not draw its particle burst")
		}
		vertices := commands.Index(0).FieldByName("Vertices")
		if vertices.Len() < 24*4 || vertices.Len() > 48*4 {
			t.Fatal("incorrect size of Cart Termination burst")
		}
		center := effects.EffectWorldAnchor(ctx, m.effects().Active[0], at)
		for i := 0; i < vertices.Len(); i += 4 {
			var x, y, z float64
			for j := 0; j < 4; j++ {
				v := vertices.Index(i + j)
				x += v.FieldByName("X").Float() / 4
				y += v.FieldByName("Y").Float() / 4
				z += v.FieldByName("Z").Float() / 4
				if math.Abs(v.FieldByName("ColorA").Float()-224.0/255) > .001 {
					t.Fatal("incorrect particle fade")
				}
			}
			radius := math.Sqrt((x-center.X)*(x-center.X) + (y-center.Y)*(y-center.Y) + (z-center.Z)*(z-center.Z))
			if math.Abs(radius-5.2) > .001 {
				t.Fatal("particles must travel 26 native units before slowing")
			}
		}
	}
}

func TestWizardSoundTiming(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.effects().AddAt(ctx, effects.FirePillar, 300, start)
	if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start.Add(75*db.EffectFrameDuration)) {
		t.Fatal("Fire Pillar must delay its sound until the pillar erupts")
	}
	m.scheduledSounds = nil
	spec, _ := effects.Lookup(effects.Quagmire)
	for id := uint32(9000); id < 10200; id++ {
		m.effects().ScheduleWorldEffectSound(start, spec, effects.Instance{EffectID: effects.Quagmire, ActorID: id, Starts: start, X: 15, Y: 20})
	}
	if n := len(m.scheduledSounds); n < 60 || n > 140 {
		t.Fatalf("Quagmire played %d sounds for 1200 cells, expected roughly one in twelve", n)
	}
	for _, sound := range m.scheduledSounds {
		x, y, ok := scheduledSoundPosition(ctx, sound, start)
		if !ok || x != 15 || y != 20 {
			t.Fatal("Quagmire sound must stay at its cell")
		}
	}
}

func TestWizardMeteorKeepsReleasePose(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.World.Player.Job = db.JobWizard
	ctx.World.Player.Dir = 3
	actions := make([]res.ACTAction, 13*8)
	for i := range actions {
		actions[i].Animations = make([]res.ACTAnimation, 4)
	}
	m := &WorldMode{actorSystem: actors.System{PlayerView: &sprites.Humanoid{Body: &scene.SpriteView{Act: &res.ACT{Actions: actions}}}}}
	for _, x := range []uint16{5, 30} {
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillWZMeteor, SourceID: 2000000, X: x, Y: 20})
		anim := m.actorSystem.ActorAnims[100]
		if ctx.World.Player.Dir != 3 || anim.ActionFamily != sprites.ActionPCSkill || anim.Play || !anim.HasFixedMotion || anim.FixedMotion != 3 {
			t.Fatal("Meteor must hold the last skill frame without turning toward each impact")
		}
	}
}

func TestWizardMeteorPreservesMovement(t *testing.T) {
	for _, sourceID := range []uint32{2000000, 100, 300} {
		ctx := testutil.KnightEffectContext()
		actor, _, local := scene.ActorForCombatID(ctx, sourceID)
		actor.Job, actor.Dir = db.JobWizard, 3
		actor.Moving = true
		actor.FromX, actor.FromY = 10, 20
		actor.ToX, actor.ToY = 20, 20
		actor.MovePath = []worldstate.WalkStep{{X: 10, Y: 20}, {X: 20, Y: 20}}
		actor.MoveStarted = time.Now()
		actor.MoveDuration = 10 * time.Second
		if local {
			ctx.World.Player = actor
		} else {
			ctx.World.Actors[actor.ID] = actor
		}
		m := &WorldMode{}
		for _, x := range []uint16{5, 30} {
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillWZMeteor, SourceID: sourceID, X: x, Y: 20})
			got := ctx.World.Player
			if !local {
				got = ctx.World.Actors[actor.ID]
			}
			if !reflect.DeepEqual(got, actor) || len(m.actorSystem.ActorAnims) != 0 {
				t.Fatalf("meteor interrupted actor %d's movement or walking animation", sourceID)
			}
		}
		if len(m.effects().Active) != 2 {
			t.Fatal("moving caster lost the meteor effects")
		}
	}
}

func TestWizardRealAssets(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{effects.SightTrasher, effects.SightBlaster, effects.IceWall, effects.EarthSpike, effects.HeavenDrive, effects.StormGust, effects.LordVermilion, effects.MeteorStorm, effects.JupitelThunder, effects.JupitelHit, effects.WaterBall2, effects.Quagmire, effects.FirePillar, effects.FirePillarBomb, effects.FirePillarOn} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.effects().AddBetween(ctx, id, 300, 2000000, start) {
				t.Fatal("missing effect")
			}
			for _, sound := range m.scheduledSounds {
				for _, path := range sound.paths {
					if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
						t.Fatal(err)
					}
				}
			}
			draws := 0
			for _, tick := range []int{0, 15, 60, 150} {
				frame := render.NewFrame(800, 600)
				m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 20, 20, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
				v := reflect.ValueOf(frame).Elem()
				draws += v.FieldByName("worldCommands").Len() + v.FieldByName("worldBillboards").Len() + v.FieldByName("commands").Len()
			}
			if draws == 0 {
				t.Fatal("effect produced no geometry with real assets")
			}
		})
	}
}

func TestEffectBodyColorTintFollowsRobrowserFlashWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		effectSystem: effects.System{Active: []effects.Instance{{
			EffectID: effects.BodyColor,
			ActorID:  42,
			Starts:   starts,
			Expires:  starts.Add(300 * time.Millisecond),
		}}},
	}
	if got := mode.actors().ActorBodyColorTint(42, base, starts); got != base {
		t.Fatalf("initial tint = %+v, want base %+v", got, base)
	}
	if got := mode.actors().ActorBodyColorTint(42, base, starts.Add(50*time.Millisecond)); got != (color.RGBA{R: 177, G: 75, B: 100, A: 255}) {
		t.Fatalf("half flash tint = %+v", got)
	}
	if got := mode.actors().ActorBodyColorTint(42, base, starts.Add(100*time.Millisecond)); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("full flash tint = %+v", got)
	}
	if got := mode.actors().ActorBodyColorTint(42, base, starts.Add(300*time.Millisecond)); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("final flash tint = %+v", got)
	}
	if got := mode.actors().ActorBodyColorTint(42, base, starts.Add(301*time.Millisecond)); got != base {
		t.Fatalf("expired tint = %+v, want base %+v", got, base)
	}
	if got := mode.actors().ActorBodyColorTint(99, base, starts.Add(100*time.Millisecond)); got != base {
		t.Fatalf("other actor tint = %+v, want base %+v", got, base)
	}
}

func TestMagicCrasher2BodyColorTintUsesOriginalColorWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		effectSystem: effects.System{Active: []effects.Instance{{
			EffectID: effects.MagicCrasher2,
			ActorID:  42,
			Starts:   starts,
			Expires:  starts.Add(100 * db.EffectFrameDuration),
		}}},
	}
	first := mode.actors().ActorBodyColorTint(42, base, starts.Add(100*time.Millisecond))
	second := mode.actors().ActorBodyColorTint(42, base, starts.Add(100*time.Millisecond))
	if first != second {
		t.Fatalf("random tint should be deterministic per frame: first %+v second %+v", first, second)
	}
	if first == base || first.A != base.A {
		t.Fatalf("active random tint = %+v, want color-channel change with unchanged alpha", first)
	}
	if got := mode.actors().ActorBodyColorTint(42, base, starts.Add(61*db.EffectFrameDuration)); got != base {
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
		{"EF_BABYBODY start", effects.BabyBody, 300 * time.Millisecond, 0, 1},
		{"EF_BABYBODY middle", effects.BabyBody, 300 * time.Millisecond, 150 * time.Millisecond, 0.75},
		{"EF_BABYBODY end", effects.BabyBody, 300 * time.Millisecond, 300 * time.Millisecond, 0.5},
		{"EF_BABYBODY2", effects.BabyBody2, 5 * time.Minute, time.Second, 0.5},
		{"EF_GIANTBODY middle", effects.GiantBody, 300 * time.Millisecond, 150 * time.Millisecond, 1.25},
		{"EF_GIANTBODY end", effects.GiantBody, 300 * time.Millisecond, 300 * time.Millisecond, 1.5},
		{"EF_GIANTBODY2", effects.GiantBody2, 5 * time.Minute, time.Second, 1.5},
	} {
		mode := WorldMode{
			effectSystem: effects.System{Active: []effects.Instance{{
				EffectID: tc.effectID,
				ActorID:  42,
				Starts:   starts,
				Expires:  starts.Add(tc.duration),
				Duration: tc.duration,
			}}},
		}
		if got := mode.actors().ActorBodySizeMultiplier(42, starts.Add(tc.at)); got != tc.want {
			t.Fatalf("%s multiplier = %f, want %f", tc.name, got, tc.want)
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
		ObjectType:    scene.ActorObjectTypeMob,
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

	if len(mode.effects().Active) != 5 {
		t.Fatalf("world effects = %d, want one bolt sequence and four impacts", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.FireBolt || effect.ActorID != 300 || effect.HitCount != 4 {
		t.Fatalf("wrong bolt sequence: %+v", effect)
	}
	for i := 1; i < 5; i++ {
		effect := mode.effects().Active[i]
		if effect.EffectID != effects.FireHit || effect.ActorID != 300 {
			t.Fatalf("hit effect %d = %+v", i, effect)
		}
		if i > 1 {
			if delay := effect.Starts.Sub(mode.effects().Active[i-1].Starts); delay != scene.MultiHitDelay {
				t.Fatalf("hit delay %d = %s, want %s", i, delay, scene.MultiHitDelay)
			}
		}
	}
	if len(mode.battleSystem.DamageFloaters) != 8 {
		t.Fatalf("damage floaters = %d, want 8", len(mode.battleSystem.DamageFloaters))
	}
	wantFloaters := []struct {
		text string
		kind battle.FloaterKind
	}{
		{text: "252", kind: battle.FloaterNormal},
		{text: "252", kind: battle.FloaterCombo},
		{text: "252", kind: battle.FloaterNormal},
		{text: "504", kind: battle.FloaterCombo},
		{text: "252", kind: battle.FloaterNormal},
		{text: "756", kind: battle.FloaterCombo},
		{text: "252", kind: battle.FloaterNormal},
		{text: "1008", kind: battle.FloaterCombo},
	}
	for i, want := range wantFloaters {
		floater := mode.battleSystem.DamageFloaters[i]
		if floater.Text != want.text || floater.Kind != want.kind {
			t.Fatalf("floater %d = %+v, want text=%q kind=%d", i, floater, want.text, want.kind)
		}
		if i > 1 {
			if delay := floater.Starts.Sub(mode.battleSystem.DamageFloaters[i-2].Starts); delay != scene.MultiHitDelay {
				t.Fatalf("floater %d delay = %s, want %s", i, delay, scene.MultiHitDelay)
			}
		}
		if floater.Kind == battle.FloaterCombo {
			if floater.Duration != battle.DamageFloaterDuration(battle.FloaterCombo) {
				t.Fatalf("combo floater %d duration = %s, want %s", i, floater.Duration, battle.DamageFloaterDuration(battle.FloaterCombo))
			}
			wantVisible := battle.DamageFloaterComboTransientDuration()
			if i == len(wantFloaters)-1 {
				wantVisible = battle.DamageFloaterDuration(battle.FloaterCombo)
			}
			if visible := floater.Expires.Sub(floater.Starts); visible != wantVisible {
				t.Fatalf("combo floater %d visible = %s, want %s", i, visible, wantVisible)
			}
		}
	}
}

func TestLightningBoltStartsWithSkillAction(t *testing.T) {
	for _, tt := range []struct {
		name   string
		hits   uint16
		damage int32
	}{
		{"single hit", 1, 100},
		{"four hits", 4, 400},
		{"ten hits", 10, 1000},
		{"zero damage", 4, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			world.UpsertActor(worldstate.Actor{ID: 300, X: 11, Y: 20, Job: 1002, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true})
			mode := &WorldMode{}
			ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
			mode.applyActorActionNotify(ctx, network.ActorActionNotify{
				SkillID: db.SkillMGLightningbolt, SkillLevel: tt.hits,
				SourceID: 2000000, TargetID: 300, SourceSpeed: 580, TargetSpeed: 480,
				Damage: tt.damage, HitCount: tt.hits, Action: network.ActorActionSkill,
			})

			hitCount := int(tt.hits)
			if tt.damage == 0 {
				hitCount = 0
			}
			if len(mode.effects().Active) != 1+hitCount {
				t.Fatalf("effects = %d, want one lightning animation and %d impacts", len(mode.effects().Active), hitCount)
			}
			started := mode.actorSystem.ActorAnims[2000000].Started
			bolt := mode.effects().Active[0]
			if started.IsZero() || bolt.EffectID != effects.LightningBolt || bolt.ActorID != 300 || !bolt.Starts.Equal(started) {
				t.Fatalf("lightning = %+v, want one animation on target starting with caster at %s", bolt, started)
			}
			for i, hit := range mode.effects().Active[1:] {
				want := started.Add(580*time.Millisecond + time.Duration(i)*200*time.Millisecond)
				if hit.EffectID != effects.WindHit || hit.ActorID != 300 || !hit.Starts.Equal(want) {
					t.Fatalf("impact %d = %+v, want wind hit on target at %s", i, hit, want)
				}
			}
			var damageTimes []time.Time
			for _, floater := range mode.battleSystem.DamageFloaters {
				if floater.Kind == battle.FloaterNormal {
					damageTimes = append(damageTimes, floater.Starts)
				}
			}
			if len(damageTimes) != hitCount {
				t.Fatalf("damage numbers = %d, want %d", len(damageTimes), hitCount)
			}
			for i, hitAt := range damageTimes {
				if !hitAt.Equal(mode.effects().Active[i+1].Starts) {
					t.Fatalf("damage %d does not coincide with its impact", i)
				}
			}
		})
	}
}

func TestZeroDamageSkillKeepsExecutionEffectsWithoutHitReaction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1063,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     db.SkillMGSoulstrike,
		SkillLevel:  8,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		HitCount:    4,
		Action:      8,
	})

	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.SoulStrike || mode.effects().Active[0].HitCount != 4 || mode.effects().Active[0].ActorID != 2000000 || mode.effects().Active[0].TargetID != 300 {
		t.Fatalf("world effects = %+v, want one four-soul sequence from caster to target", mode.effects().Active)
	}
	if _, ok := mode.actorSystem.ActorAnims[300]; ok {
		t.Fatal("zero-damage target received a hurt animation")
	}
	if len(mode.scheduledSounds) != 4 {
		t.Fatalf("scheduled sounds = %+v, want four Soul Strike effect sounds", mode.scheduledSounds)
	}
	for i, sound := range mode.scheduledSounds {
		if len(sound.paths) != 1 || sound.paths[0] != "effect\\ef_soulstrike.wav" {
			t.Fatalf("scheduled sound %d = %+v, want only the Soul Strike effect sound", i, sound)
		}
	}
	if len(mode.battleSystem.DamageFloaters) != 1 || mode.battleSystem.DamageFloaters[0].Text != "miss" || mode.battleSystem.DamageFloaters[0].Kind != battle.FloaterMiss {
		t.Fatalf("damage floaters = %+v, want one miss", mode.battleSystem.DamageFloaters)
	}
}

func TestZeroDamageSkillKeepsImmediateEffectWithoutHitEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1063,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     db.SkillMGFrostdiver,
		SkillLevel:  1,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.FrostDiver {
		t.Fatalf("world effects = %+v, want Frost Diver execution effect without hit effect", mode.effects().Active)
	}
	wantStart := mode.actorSystem.ActorAnims[2000000].Started
	if !mode.effects().Active[0].Starts.Equal(wantStart) {
		t.Fatalf("Frost Diver start = %s, want action time %s", mode.effects().Active[0].Starts, wantStart)
	}
}

func TestWindHitEffectSpecMatchesRobrowserRandomSTRAndSFX(t *testing.T) {
	spec, ok := effects.Lookup(effects.WindHit)
	if !ok {
		t.Fatal("wind hit effect missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentSTR || component.StrFile != "windhit%d" || component.StrRandMin != 1 || component.StrRandMax != 3 || !component.AttachedEntity {
		t.Fatalf("wind hit component = %+v", component)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "_hit_fist%d.wav" || spec.SfxRandMin != 1 || spec.SfxRandMax != 3 {
		t.Fatalf("wind hit sfx = %+v rand=%d..%d", spec.Sfx, spec.SfxRandMin, spec.SfxRandMax)
	}

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
	if !mode.effects().AddAt(ctx, effects.WindHit, 2000000, time.Unix(10, 20)) {
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

	if len(mode.effects().Active) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.effects().Active))
	}
	circle := mode.effects().Active[0]
	if circle.EffectID != effects.CastRing || circle.ActorID != 2000000 || circle.TargetID != 0 || circle.Duration != 2500*time.Millisecond {
		t.Fatalf("circle = %+v", circle)
	}
	lockon := mode.effects().Active[1]
	if lockon.EffectID != effects.LockOnTarget || lockon.ActorID != 1100 || lockon.TargetID != 0 || lockon.Duration != 2500*time.Millisecond {
		t.Fatalf("lockon = %+v", lockon)
	}
	aura := mode.effects().Active[2]
	if aura.EffectID != effects.BeginSpell5 || aura.ActorID != 2000000 || aura.TargetID != 1100 || aura.Duration != 2500*time.Millisecond {
		t.Fatalf("aura = %+v", aura)
	}
	bar, ok := mode.battleSystem.ActorCastBars[150000]
	if !ok {
		t.Fatal("local cast bar missing")
	}
	if bar.Duration != 2500*time.Millisecond || bar.Color != (color.RGBA{R: 0, G: 255, B: 0, A: 255}) {
		t.Fatalf("cast bar = %+v", bar)
	}
	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("cast animation missing")
	}
	if anim.ActionFamily != sprites.ActionPCReadyFight || anim.Duration != 2500*time.Millisecond || anim.HasFixedMotion {
		t.Fatalf("cast animation = %+v", anim)
	}
	if world.Dir != actors.DirectionFromDelta(10, 20, 12, 20, 4) {
		t.Fatalf("cast dir = %d", world.Dir)
	}
}

func TestSkillCastCancelClearsOnlyTheCasterPresentation(t *testing.T) {
	for _, ground := range []bool{false, true} {
		for _, cancelID := range []uint32{2000000, 150000, 300} {
			t.Run(fmt.Sprintf("ground=%t/cancel=%d", ground, cancelID), func(t *testing.T) {
				world := worldstate.New()
				world.Player = worldstate.Actor{ID: 150000, X: 10, Y: 20, Job: db.JobMagician}
				world.UpsertActor(worldstate.Actor{ID: 300, X: 11, Y: 20, Job: db.JobMagician})
				world.UpsertActor(worldstate.Actor{ID: 400, X: 12, Y: 20, Job: 1002})
				ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}
				mode := NewWorldMode()
				sourceID, otherID := uint32(2000000), uint32(300)
				if cancelID == 300 {
					sourceID, otherID = otherID, sourceID
				}
				cast := network.SkillCastNotify{SourceID: sourceID, TargetID: 400, SkillID: db.SkillMGFirebolt, Property: 3, DelayTime: 5000}
				if ground {
					cast.TargetID, cast.X, cast.Y, cast.SkillID = 0, 12, 20, db.SkillMGThunderstorm
				}
				mode.applySkillCastNotify(ctx, cast)
				cast.SourceID = otherID
				mode.applySkillCastNotify(ctx, cast)
				if len(mode.effects().Active) != 6 {
					t.Fatalf("two casts must each own three effects: %+v", mode.effects().Active)
				}
				mode.effects().AddAt(ctx, effects.IncAgility, sourceID, time.Now())
				wantEffects := append([]effects.Instance(nil), mode.effects().Active[3:]...)
				otherAnim := mode.actorSystem.ActorAnims[otherID]

				data := []byte{0xB9, 0x01, 0, 0, 0, 0}
				binary.LittleEndian.PutUint32(data[2:], cancelID)
				packet := network.Packet{ID: network.PacketZCSkillCastCancel, Data: data}
				mode.handleNetworkPacket(ctx, packet, time.Now())
				mode.handleNetworkPacket(ctx, packet, time.Now())
				if !reflect.DeepEqual(mode.effects().Active, wantEffects) {
					t.Fatal("cancellation removed another cast/buff or left the cancelled cast's effects")
				}
				for _, id := range []uint32{sourceID, cancelID} {
					if _, ok := mode.battleSystem.ActorCastBars[id]; ok {
						t.Fatalf("cast bar still present for %d", id)
					}
					if _, ok := mode.actorSystem.ActorAnims[id]; ok {
						t.Fatalf("cast animation still present for %d", id)
					}
				}
				if _, ok := mode.battleSystem.ActorCastBars[otherID]; !ok || !reflect.DeepEqual(mode.actorSystem.ActorAnims[otherID], otherAnim) {
					t.Fatal("another caster lost its bar or animation")
				}
			})
		}
	}
}

func TestSkillCastCancelPreservesReplacementAnimation(t *testing.T) {
	for _, action := range []int{sprites.ActionPCHurt, sprites.ActionPCDeath, sprites.ActionPCSkill} {
		ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: worldstate.New()}
		ctx.World.Player = worldstate.Actor{ID: 150000, Job: db.JobMagician}
		mode := NewWorldMode()
		mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, SkillID: db.SkillMGFirebolt, Property: 3, DelayTime: 5000})
		replacement := scene.ActorAnimation{ActionFamily: action, Started: time.Now(), Duration: time.Second}
		mode.setCombatActorAction(ctx, 2000000, replacement)
		mode.applySkillCastCancel(ctx, 2000000)
		for _, id := range []uint32{2000000, 150000} {
			if got := mode.actorSystem.ActorAnims[id]; !reflect.DeepEqual(got, replacement) {
				t.Fatalf("replacement animation %d for %d changed: %+v", action, id, got)
			}
		}
	}
}

func TestArrowRepelCastUsesNativeFlash(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: db.SkillACChargearrow, Property: 4, DelayTime: 1200})

	if len(mode.effects().Active) != 3 {
		t.Fatalf("world effects = %+v, want cast ring, target lock-on, and Bash flash", mode.effects().Active)
	}
	if mode.effects().Active[0].EffectID != effects.CastRing {
		t.Fatalf("effect = %+v, want cast ring", mode.effects().Active[0])
	}
	if mode.effects().Active[1].EffectID != effects.LockOnTarget || mode.effects().Active[1].ActorID != 1100 {
		t.Fatalf("effect = %+v, want target lock-on", mode.effects().Active[1])
	}
	if e := mode.effects().Active[2]; e.EffectID != effects.BashBegin || e.Duration != 0 {
		t.Fatalf("Arrow Repel needs a single flash with its own duration: %+v", e)
	}
	if _, ok := mode.battleSystem.ActorCastBars[150000]; !ok {
		t.Fatal("Arrow Repel cast bar should remain visible")
	}
}

func TestSkillResultAnimationReplacesCastStance(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 4, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: 28, DelayTime: 1200})
	if anim, ok := mode.actorSystem.ActorAnims[150000]; !ok || anim.ActionFamily != sprites.ActionPCReadyFight {
		t.Fatalf("cast stance animation = %+v ok=%t", anim, ok)
	}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 1100, SkillID: 28, Amount: 234, Result: 1})
	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("skill result animation missing")
	}
	if anim.ActionFamily != sprites.ActionPCSkill || anim.HasFixedMotion {
		t.Fatalf("skill result animation = %+v, want delivery skill action", anim)
	}
}

func TestImportedSkillActionFallback(t *testing.T) {
	archer := worldstate.Actor{Job: 3}
	if action := effects.SkillAction(db.SkillACDouble).ActionFamilyForActorWithResources(nil, archer); action != sprites.AttackActionFamily(nil, archer) {
		t.Fatalf("AC_DOUBLE action = %d, want normal weapon attack", action)
	}
	shower := effects.SkillAction(db.SkillACShower)
	if action := shower.ActionFamilyForActorWithResources(nil, archer); action != sprites.AttackActionFamily(nil, archer) {
		t.Fatalf("AC_SHOWER action = %d, want normal attack family", action)
	}
	if shower.Speed != 0 {
		t.Fatalf("AC_SHOWER must use the supplied attack duration: %+v", shower)
	}
	merchant := worldstate.Actor{Job: 5}
	if action := effects.SkillAction(db.SkillMCCartrevolution).ActionFamilyForActorWithResources(nil, merchant); action != sprites.ActionPCSkill {
		t.Fatalf("MC_CARTREVOLUTION action = %d, want SKILL", action)
	}
	counter := effects.SkillAction(db.SkillKNAutocounter)
	if !counter.Defined || counter.Action != effects.SkillActorActionAttack || !counter.HasFrame || counter.Frame != 0 || counter.Play || counter.Next != nil {
		t.Fatalf("KN_AUTOCOUNTER action = %+v, want robr attack frame 0 with play=false next=false", counter)
	}
	relax := effects.SkillAction(db.SkillLKTensionrelax)
	if !relax.Defined || relax.Action != effects.SkillActorActionNone {
		t.Fatalf("LK_TENSIONRELAX action = %+v, want robr false action", relax)
	}
	trap := effects.SkillAction(db.SkillHTLandmine)
	if !trap.Defined || trap.Action != effects.SkillActorActionPickup || !trap.Play || trap.Repeat || trap.Next == nil || trap.Next.Action != effects.SkillActorActionIdle {
		t.Fatalf("HT_LANDMINE action = %+v, want robr PICKUP then IDLE", trap)
	}
	sight := effects.SkillAction(db.SkillSNSight)
	if !sight.Defined || sight.Action != effects.SkillActorActionIdle || !sight.Play || !sight.Repeat || sight.Next != nil {
		t.Fatalf("SN_SIGHT action = %+v, want native standing pose", sight)
	}
	for _, skillID := range []uint16{db.SkillDCWinkcharm, db.SkillBDRokisweil} {
		action := effects.SkillAction(skillID)
		if !action.Defined || action.Action != effects.SkillActorActionSkill || !action.Play || !action.Repeat || action.Next != nil || !action.HasFrame || action.Frame != 1 || action.Length != 3 || action.Speed != 10*db.EffectFrameDuration {
			t.Fatalf("dance/play action for skill %d = %+v, want native repeating SKILL frames 1–3", skillID, action)
		}
	}
	sonic := effects.SkillAction(db.SkillASSonicblow)
	hits := 0
	sawReadyFight := false
	for spec := &sonic; spec != nil; spec = spec.Next {
		if spec.Action == effects.SkillActorActionReadyFight {
			if !spec.Repeat || !spec.Play || spec.Next != nil {
				t.Fatalf("AS_SONICBLOW ready fight tail = %+v, want robr READYFIGHT loop", spec)
			}
			sawReadyFight = true
			break
		}
		if spec.Action != effects.SkillActorActionAttack || spec.Repeat || !spec.Play {
			t.Fatalf("AS_SONICBLOW chain node %d = %+v, want robr ATTACK hit", hits+1, spec)
		}
		hits++
		if hits == 1 {
			if spec.Speed != 0 {
				t.Fatalf("AS_SONICBLOW first hit speed = %s, want default", spec.Speed)
			}
			continue
		}
		if spec.Speed != 30*time.Millisecond {
			t.Fatalf("AS_SONICBLOW hit %d speed = %s, want 30ms", hits, spec.Speed)
		}
	}
	if hits != 8 {
		t.Fatalf("AS_SONICBLOW hits = %d, want robr 8-hit chain", hits)
	}
	if !sawReadyFight {
		t.Fatal("AS_SONICBLOW chain missing robr READYFIGHT tail")
	}
}

func TestBowNormalAttackAddsArrowProjectileEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 15, Y: 20, Job: 1002, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
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

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want arrow projectile and regular hit", len(mode.effects().Active))
	}
	projectile := mode.effects().Active[0]
	if projectile.EffectID != effects.ArrowShot || projectile.ActorID != 300 || projectile.TargetID != 200 {
		t.Fatalf("normal bow projectile = %+v", projectile)
	}
	if projectile.Duration != scene.ReferenceBowArrowFlightDuration || projectile.Expires.Sub(projectile.Starts) != scene.ReferenceBowArrowFlightDuration {
		t.Fatalf("normal bow projectile flight = duration %s lifetime %s, want %s", projectile.Duration, projectile.Expires.Sub(projectile.Starts), scene.ReferenceBowArrowFlightDuration)
	}
	hit := mode.effects().Active[1]
	if hit.EffectID != effects.Hit1 || hit.ActorID != 300 || !hit.Starts.Equal(projectile.Expires) {
		t.Fatalf("normal bow hit = %+v projectile=%+v", hit, projectile)
	}
}

func TestArcherMercenaryNormalAttackAddsArrowProjectileEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		Job:           6017,
		ObjectType:    scene.ActorObjectTypeMercenary,
		HasObjectType: true,
	})
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             15,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    300,
		Damage:      12,
		HitCount:    1,
		Action:      0,
		SourceSpeed: 500,
		TargetSpeed: 500,
	})

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want arrow projectile and regular hit", len(mode.effects().Active))
	}
	projectile := mode.effects().Active[0]
	if projectile.EffectID != effects.ArrowShot || projectile.ActorID != 300 || projectile.TargetID != 400 {
		t.Fatalf("mercenary arrow projectile = %+v", projectile)
	}
	hit := mode.effects().Active[1]
	if hit.EffectID != effects.Hit1 || hit.ActorID != 300 || !hit.Starts.After(projectile.Starts) {
		t.Fatalf("mercenary bow hit = %+v projectile=%+v", hit, projectile)
	}
}

func TestWarpPortalListOpensDestinationModal(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{Skills: session.Skills{List: []session.Skill{
		{ID: 27, Level: 4, Type: skills.SkillTargetPlace, Range: 9},
	}}}}
	mode.skills().ApplyWarpPointList(ctx, network.WarpPointList{SkillID: 27, MapNames: []string{"prontera", "geffen", "payon"}})
	if !mode.ui.teleportModal.IsOpen() {
		t.Fatal("warp portal list should open the destination modal")
	}
	if mode.ui.teleportModal.Title() != "Warp Portal" {
		t.Fatalf("modal title = %q", mode.ui.teleportModal.Title())
	}
}

func TestQuakeMagnumEffectStartsCameraShake(t *testing.T) {
	spec, ok := effects.Lookup(effects.QuakeMagnum)
	if !ok {
		t.Fatal("quake magnum effect spec missing")
	}
	if spec.CameraShake != 50*time.Millisecond || len(spec.Components) != 0 {
		t.Fatalf("quake magnum spec = %+v, want no-draw 50ms camera shake", spec)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	starts := time.Unix(100, 0)
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	if !mode.effects().AddAt(ctx, effects.QuakeMagnum, 2000000, starts) {
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

func TestBerserkPotionEffectStartsDelayedCameraShake(t *testing.T) {
	spec, ok := effects.Lookup(effects.ItemFast3)
	if !ok {
		t.Fatal("berserk potion effect spec missing")
	}
	if spec.CameraShake != 200*time.Millisecond || spec.CameraShakeDelay != 18*db.EffectFrameDuration {
		t.Fatalf("berserk potion shake = delay %s duration %s, want 18 ticks/200ms", spec.CameraShakeDelay, spec.CameraShake)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	starts := time.Unix(100, 0)
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	if !mode.effects().AddAt(ctx, effects.ItemFast3, 2000000, starts) {
		t.Fatal("add berserk potion effect failed")
	}
	shakeStart := starts.Add(18 * db.EffectFrameDuration)
	shakeEnd := shakeStart.Add(200 * time.Millisecond)
	if !mode.cameraShakeStart.Equal(shakeStart) || !mode.cameraShakeEnd.Equal(shakeEnd) {
		t.Fatalf("camera shake = %s..%s, want %s..%s", mode.cameraShakeStart, mode.cameraShakeEnd, shakeStart, shakeEnd)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(100 * time.Millisecond)); x != 0 || y != 0 {
		t.Fatalf("early camera shake offset = %.3f, %.3f, want zero", x, y)
	}
	if x, y := mode.cameraShakeOffset(shakeStart.Add(50 * time.Millisecond)); x == 0 && y == 0 {
		t.Fatalf("active camera shake offset = %.3f, %.3f, want non-zero", x, y)
	}
}

func TestSPREffectShowsEveryFrameBeforeExpiring(t *testing.T) {
	image := render.NewImage(4, 4)
	image.Fill(color.White)
	animation := res.ACTAnimation{Layers: []res.ACTLayer{{
		ScaleX: 1, ScaleY: 1, Color: [4]float32{1, 1, 1, 1},
	}}}
	act := &res.ACT{Actions: make([]res.ACTAction, 20)}
	act.Actions[0] = res.ACTAction{Animations: []res.ACTAnimation{animation}, DelayMS: 50}
	// Pupa's sweating emotion uses action 19: 24 frames at 50 ms each.
	act.Actions[19] = res.ACTAction{Animations: make([]res.ACTAnimation, 24), DelayMS: 50}
	for i := range act.Actions[19].Animations {
		act.Actions[19].Animations[i] = animation
	}
	view := &scene.SpriteView{
		Act:        act,
		Spr:        &res.SPR{Frames: []res.SPRFrame{{Width: 4, Height: 4}}},
		Images:     map[scene.SpriteFrameKey]*render.Image{{}: image},
		Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}
	mode := WorldMode{effectSystem: effects.System{EffectViews: map[string]*scene.SpriteView{"emotion": view}}}
	ctx := client.Context{Resources: &res.Manager{}}
	projection := testutil.Projection(800, 600, 0, 0, 0)
	starts := time.Unix(10, 0)
	for _, tc := range []struct {
		name                  string
		action, ms            int
		stop, repeat, visible bool
	}{
		{"first", 19, 0, false, false, true},
		{"last begins", 19, 1150, false, false, true},
		{"last ends", 19, 1199, false, false, true},
		{"expired", 19, 1200, false, false, false},
		{"single frame", 0, 0, false, false, true},
		{"single expired", 0, 50, false, false, false},
		{"hold last", 19, 1200, true, false, true},
		{"repeat", 19, 1200, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			screen := render.NewFrame(800, 600)
			effect := effects.Instance{Starts: starts, HasSpriteFrame: true, SpriteFrameOverride: tc.action}
			component := effects.Component{SpriteFile: "emotion", SpriteStopAtEnd: tc.stop, SpriteRepeat: tc.repeat}
			mode.effects().DrawSPREffect(screen, ctx, projection, effect, component, 0, 0, 0, starts.Add(time.Duration(tc.ms)*time.Millisecond))
			visible := !reflect.DeepEqual(screen, render.NewFrame(800, 600))
			if visible != tc.visible {
				t.Fatalf("visible=%t, want %t", visible, tc.visible)
			}
		})
	}
}

func TestWorldSizedSpriteBillboardUsesCenterDepthLikeRobrowser(t *testing.T) {
	options := render.DrawTrianglesOptions{DepthBias: 0.01}
	cmd := scene.WorldSpriteBillboardCommand(
		render.WhiteImage(),
		options,
		scene.Point3{X: 1, Y: 2, Z: 3},
		scene.Point3{X: 4, Y: 5, Z: 6},
		scene.Point3{X: 7, Y: 8, Z: 9},
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
	component := effects.Component{
		PosX:           -2,
		PosY:           4,
		OrbitRadiusX:   3,
		OrbitRadiusY:   3,
		OrbitRotations: 8,
		OrbitPhase:     0.7,
		OrbitClockwise: true,
	}
	x, y, _ := (&WorldMode{}).effects().Effect3DOffset(client.Context{}, component, effects.Instance{}, 0, 0, 0, 0, 0, 0)
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

func TestSpecialEffectNotifyAddsLevelUpEffects(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectBaseLevelUp})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectJobLevelUp})

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.effects().Active))
	}
	if mode.effects().Active[0].EffectID != effects.BaseLevelUp || mode.effects().Active[1].EffectID != effects.JobLevelUp {
		t.Fatalf("world effects = %+v", mode.effects().Active)
	}
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
	if mode.scheduledSounds[0].paths[0] != "levelup.wav" {
		t.Fatalf("base level-up sound = %+v", mode.scheduledSounds[0])
	}
	if !mode.ui.levelUpNotifications.BaseVisible() || !mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("local level-up effects did not show both availability notifications")
	}
}

func TestRemoteLevelUpEffectDoesNotShowAvailabilityNotification(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 3000000, EffectID: network.SpecialEffectBaseLevelUp})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 3000000, EffectID: network.SpecialEffectJobLevelUp})

	if mode.ui.levelUpNotifications.BaseVisible() || mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("remote level-up effect showed a local availability notification")
	}
}

func TestLevelUpNotificationActionsOpenStatusAndSkillWindows(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{
		Session:   &session.Session{},
		UIManager: gameui.NewManager(),
		ScreenW:   800,
		ScreenH:   600,
	}

	if !mode.handleLevelUpNotificationAction(ctx, gameui.LevelUpNotificationBase|gameui.LevelUpNotificationJob) {
		t.Fatal("level-up notification action was not handled")
	}
	if !mode.ui.statsWindow.IsOpen() {
		t.Fatal("base notification did not open the status window")
	}
	if !mode.ui.skillWindow.IsOpen() {
		t.Fatal("job notification did not open the skill window")
	}
}

func TestSpecialEffectNotifyMapsServerResultEffects(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	for _, effectID := range []uint32{
		network.SpecialEffectRefineFailure,
		network.SpecialEffectRefineSuccess,
		network.SpecialEffectPharmacySuccess,
		network.SpecialEffectPharmacyFailure,
		network.SpecialEffectGameOver,
	} {
		mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: effectID})
	}

	want := []int{effects.RefineOK, effects.PharmacyFail}
	wantSFX := []string{
		"effect\\bs_refinefailed.wav",
		"effect\\bs_refinesuccess.wav",
		"effect\\p_success.wav",
		"effect\\p_failed.wav",
	}
	if len(mode.effects().Active) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.effects().Active), len(want), mode.effects().Active)
	}
	for i, effect := range mode.effects().Active {
		if effect.EffectID != want[i] {
			t.Fatalf("world effect %d = %+v, want effect %d", i, effect, want[i])
		}
	}
	if len(mode.scheduledSounds) != len(wantSFX) {
		t.Fatalf("scheduled sounds = %d, want %d: %+v", len(mode.scheduledSounds), len(wantSFX), mode.scheduledSounds)
	}
	for i, sound := range mode.scheduledSounds {
		if len(sound.paths) != 1 || sound.paths[0] != wantSFX[i] {
			t.Fatalf("scheduled sound %d = %+v, want %q", i, sound, wantSFX[i])
		}
	}
}

func TestCookingListSelectionAndCancellation(t *testing.T) {
	netClient, server := testutil.NewConnection(t, 20080910)
	mode := NewWorldMode()
	ctx := client.Context{Input: input.NewState(), Network: netClient, ScreenW: 800, ScreenH: 600}
	press := func(key input.Key) {
		ctx.Input.SetKey(key, true)
		if !mode.ui.makingItem.Update(ctx) {
			t.Fatalf("key %v was not consumed", key)
		}
		ctx.Input.EndFrame()
		ctx.Input.SetKey(key, false)
		ctx.Input.EndFrame()
	}
	for _, makeType := range []byte{1, 2, 3} {
		data := []byte{0x5a, 0x02, 10, 0, makeType, 0, 0xe1, 0x2e, 0xe2, 0x2e}
		packet := network.Packet{ID: network.PacketZCMakingItemList, Data: data}
		mode.handleNetworkPacket(ctx, packet, time.Now())
		if !mode.ui.makingItem.IsOpen() {
			t.Fatal("cooking packet did not open the item selection window")
		}
		press(input.KeyArrowDown)
		press(input.KeyEnter)
		testutil.ReadPackets(t, server, []byte{0x5b, 0x02, makeType, 0, 0xe2, 0x2e})
		if mode.ui.makingItem.IsOpen() {
			t.Fatal("selection left the cooking window open")
		}
		mode.handleNetworkPacket(ctx, packet, time.Now())
		press(input.KeyEscape)
		testutil.ReadPackets(t, server, []byte{0x5b, 0x02, makeType, 0, 0xff, 0xff})
		if mode.ui.makingItem.IsOpen() {
			t.Fatal("cancellation left the cooking window open")
		}
	}
	// Reopening the shared window for forging must restore its original reply.
	data := []byte{0x8d, 0x01, 12, 0, 0xf5, 1, 0xe8, 3, 0xde, 3, 5, 0}
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMakableItemList, Data: data}, time.Now())
	press(input.KeyEnter)
	testutil.ReadPackets(t, server, []byte{0x8e, 0x01, 0xf5, 1, 0xe8, 3, 0xde, 3, 5, 0})
	// An empty recipe list still releases the server's crafting menu.
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMakingItemList, Data: []byte{0x5a, 0x02, 6, 0, 1, 0}}, time.Now())
	testutil.ReadPackets(t, server, []byte{0x5b, 0x02, 1, 0, 0xff, 0xff})
	if mode.ui.makingItem.IsOpen() {
		t.Fatal("empty cooking list left a window open")
	}
}

func TestMakingItemWindowEnterConsumesBeforeConsole(t *testing.T) {
	inputState := input.NewState()
	netClient := network.NewClient(20080910, false)
	mode := &WorldMode{}
	ctx := client.Context{
		Input:   inputState,
		Network: netClient,
		ScreenW: 800,
		ScreenH: 600,
	}
	mode.ui.makingItem.OpenList(ctx, network.MakingItemList{
		Items: []network.MakingItemOption{{ItemID: 501}},
	})

	inputState.SetKey(input.KeyEnter, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}

	if mode.ui.console.Active() {
		t.Fatal("console became active before making item window consumed Enter")
	}
	if !mode.ui.makingItem.IsOpen() {
		t.Fatal("making item window closed even though the disconnected test client could not send")
	}
}

func TestMVPNotifyAddsMVPBannerEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyMVPNotify(ctx, network.MVPNotify{AID: 2000000})

	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.Mvp || effect.ActorID != 2000000 {
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

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.effects().Active))
	}
	if mode.effects().Active[0].EffectID != effects.BaseLevelUp || mode.effects().Active[1].EffectID != effects.JobLevelUp {
		t.Fatalf("world effects = %+v", mode.effects().Active)
	}
	if !mode.ui.levelUpNotifications.BaseVisible() || !mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("parameter level-up fallback did not show both availability notifications")
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

	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.JobLevelUp || effect.ActorID != 2000000 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("scheduled sounds = %+v, want none for job level-up", mode.scheduledSounds)
	}
}

func soulLinkerTestTextures(m *WorldMode) {
	m.mapScene.Textures = make(map[string]*render.Image)
	for _, id := range []int{effects.Stin, effects.Stin3, effects.HitLine4, effects.Stin2, effects.HitLine6, effects.HitTexture, effects.AsuraBodyMonster, effects.Hated, effects.Kaizel, effects.BlueBody, effects.ReflectBody, effects.SmaReady, effects.SoulLink, effects.AsuraBody, effects.BlueCasting, effects.TaeReady, effects.StormKick4, effects.BabyBodyBack, effects.M07} {
		for _, c := range db.EffectSpecs[id].Components {
			for _, file := range append([]string{c.TextureFile}, c.TextureFiles...) {
				if file != "" {
					m.mapScene.Textures["__effectfile_"+file+"|blackkey=true"] = render.NewImage(8, 8)
				}
			}
		}
	}
}

func TestAlchemistCastAndGroundOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemist, db.JobAlchemist
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill        uint16
		pose, effect int
	}{
		{db.SkillAMPharmacy, sprites.ActionPCReadyFight, effects.BashBegin},
		{db.SkillAMDemonstration, sprites.ActionPCReadyFight, effects.BashBegin},
		{db.SkillAMSpheremine, sprites.ActionPCReadyFight, effects.BashBegin},
		{db.SkillAMAcidterror, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMPotionpitcher, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCannibalize, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCpWeapon, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCpShield, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCpArmor, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCpHelm, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCallhomun, scene.SpriteActionIdle, effects.CoupleCasting},
		{db.SkillAMRest, scene.SpriteActionIdle, effects.CoupleCasting},
		{db.SkillAMResurrecthomun, scene.SpriteActionIdle, effects.CoupleCasting},
		{db.SkillAMBerserkpitcher, sprites.ActionPCAttack1, effects.BeginSpell2},
		{db.SkillAMTwilight1, sprites.ActionPCSkill, effects.Twilight1},
		{db.SkillAMTwilight2, sprites.ActionPCSkill, effects.Twilight2},
		{db.SkillAMTwilight3, sprites.ActionPCSkill, effects.Twilight3},
	} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, tc.skill, 3*time.Second, start)
		m.effects().AddSkillCastEffects(ctx, tc.skill, 1, 2000000, 0, 0, 0, 3*time.Second, start, "test")
		if m.actorSystem.ActorAnims[2000000].ActionFamily != tc.pose || len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != tc.effect {
			t.Fatalf("incorrect cast pose or aura for skill %d", tc.skill)
		}
	}
	for _, skill := range []uint16{db.SkillAMDemonstration, db.SkillAMCannibalize, db.SkillAMSpheremine} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: skill, X: 20, Y: 20})
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 || m.actorSystem.ActorAnims[2000000].Started.IsZero() {
			t.Fatal("ground notification must animate the caster and leave the visual to its unit/entity")
		}
		if skill != db.SkillAMDemonstration {
			continue
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 177, X: 20, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Demonstration || !m.effects().Active[0].Persistent {
			t.Fatal("Demonstration must belong to its persistent ground unit")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
		if len(m.effects().Active) != 0 {
			t.Fatal("Demonstration survived unit removal")
		}
	}
	for _, skill := range []uint16{db.SkillAMPharmacy, db.SkillAMTwilight1, db.SkillAMTwilight2, db.SkillAMTwilight3} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: skill, Result: 1})
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 || m.actorSystem.ActorAnims[2000000].Started.IsZero() {
			t.Fatal("crafting completion must animate the caster without a release flare or sound")
		}
	}
}

func TestAssassinPoisonParticleMotionAndFieldLifetime(t *testing.T) {
	for _, id := range []int{effects.EnchantPoison, effects.VenomDust2} {
		effect := effects.Instance{EffectID: id, Starts: time.Unix(100, 0)}
		life := 40
		if id == effects.VenomDust2 {
			life = 10
		}
		first := effects.RisingParticleAt(effect, 0, 0)
		later := effects.RisingParticleAt(effect, 0, life-3)
		if first.Alpha <= 0 || later.Position.Y <= first.Position.Y || later.Angle <= first.Angle || effects.RisingParticleAt(effect, 0, life-1).Alpha != 0 {
			t.Fatalf("effect %d has wrong rise, rotation or fade: %+v / %+v", id, first, later)
		}
		if id == effects.VenomDust2 && (first.Position.X != 0 || first.Position.Z != 0) {
			t.Fatal("Venom Dust particles must stay inside their field cell")
		}
	}
	ctx := testutil.AssassinEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, CreatorID: 2000000, UnitID: 146, X: 12, Y: 20, Visible: true})
	if len(m.effects().Active) != 1 || !m.effects().Active[0].Persistent || m.effects().Active[0].EffectID != effects.VenomDust2 {
		t.Fatal("Venom Dust must be owned by the server's ground unit")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
	if len(m.effects().Active) != 0 {
		t.Fatal("Venom Dust survived its field's removal")
	}
}

func TestBardPerformanceOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, skill := range []uint16{db.SkillBDLullaby, db.SkillBDRichmankim, db.SkillBDEternalchaos, db.SkillBDDrumbattlefield, db.SkillBDRingnibelungen, db.SkillBDRokisweil, db.SkillBDIntoabyss, db.SkillBDSiegfried, db.SkillBaDissonance, db.SkillBaWhistle, db.SkillBaAssassincross, db.SkillBaPoembragi, db.SkillBaAppleidun} {
		for _, ground := range []bool{false, true} {
			m := &WorldMode{}
			if ground {
				m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000})
			} else {
				m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
			}
			sounds := 1
			if skill == db.SkillBaDissonance || skill == db.SkillBaWhistle {
				sounds = 0
			}
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != sounds || !m.actorSystem.ActorAnims[2000000].Loop {
				t.Fatalf("skill %d ground=%t: activation must only start performance and its one sound", skill, ground)
			}
			m.stopPerformance(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
			if len(m.actorSystem.ActorAnims) != 0 {
				t.Fatal("performance end must clear both local actor aliases")
			}
		}
	}
	m := &WorldMode{}
	for unit := uint16(158); unit <= 170; unit++ {
		entry := network.SkillUnitEntry{ID: uint32(unit), UnitID: unit, X: 15, Y: 20, Visible: true}
		m.applySkillUnitEntry(ctx, entry)
		started := m.effects().Active[len(m.effects().Active)-1].Starts
		m.applySkillUnitEntry(ctx, entry)
		if !m.effects().Active[len(m.effects().Active)-1].Starts.Equal(started) {
			t.Fatal("ground-unit refresh restarted the song animation")
		}
	}
	if len(m.effects().Active) != 13 || len(m.scheduledSounds) != 0 {
		t.Fatal("each ground unit must own one silent visual")
	}
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 169, Value: 166}) || m.effects().Active[len(m.effects().Active)-1].EffectID != effects.BottomDissonanceGround {
		t.Fatal("overlapping songs must change to Dissonance")
	}
	for unit := uint32(158); unit <= 170; unit++ {
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit})
	}
	if len(m.effects().Active) != 0 {
		t.Fatal("song symbols survived removal of their ground units")
	}
}

func TestBardPangVoiceAndSongAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for unit := uint16(158); unit <= 170; unit++ {
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: unit, X: 15, Y: 20, Visible: true})
		effect := m.effects().Active[0]
		spec, _ := effects.Lookup(effect.EffectID)
		for _, file := range spec.Components[0].TextureFiles {
			if m.effects().EffectFileTexture(ctx.Resources, file) == nil {
				t.Fatalf("missing texture %s", file)
			}
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, effect.Starts.Add(20*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("song unit %d drew nothing", unit)
		}
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if unit == 160 || unit == 165 {
			vertices := commands.Index(0).FieldByName("Vertices")
			for i := 0; i < vertices.Len(); i++ {
				v := vertices.Index(i)
				radius := math.Hypot(v.FieldByName("X").Float()-15.5, v.FieldByName("Z").Float()-20.5)
				if radius < .399 || radius > .801 {
					t.Fatal("song light must rise vertically at its native radius")
				}
			}
		}
		if unit == 166 || unit == 168 {
			vertices := commands.Index(0).FieldByName("Vertices")
			if vertices.Index(0).FieldByName("Y").Float() <= vertices.Index(3).FieldByName("Y").Float() {
				t.Fatal("song ribbon texture is vertically inverted")
			}
		}
	}
	for skill, spec := range db.SkillEffects {
		if effects.IsBardPerformance(skill) {
			for _, sound := range spec.SFX {
				if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	m := &WorldMode{}
	m.effects().AddBetween(ctx, effects.FVoice, 300, 2000000, start)
	effect := m.effects().Active[0]
	if !effect.HasSpriteFrame || effect.ActorOrigin.X != 20.5 || effect.ProjectileDirection.X != -10 {
		t.Fatal("Pang Voice must capture its target and caster direction")
	}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, start.Add(6*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
		t.Fatal("Pang Voice drew nothing")
	}
	for initial := 0; initial <= 10; initial++ {
		process, alpha, height := initial, 0, 0.0
		for n := 1; n < 200; n++ {
			process++
			if process <= 10 {
				alpha += 25
			} else {
				alpha -= 5
				if alpha <= 0 {
					alpha, process, height = 0, 0, 0
				}
				height += .04
			}
			a, h := effects.SongOutParticle(n, initial)
			if a != alpha || math.Abs(h-height) > 1e-9 {
				t.Fatalf("Roki's veil diverged at initial %d tick %d", initial, n)
			}
		}
	}
}

func TestClownGypsyMoonlitStatusOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMoon, ActorID: 100, Active: true})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Moonlit || !m.effects().Active[0].Expires.IsZero() {
		t.Fatal("Moonlit status must create a persistent sphere, including on entry into view")
	}
	start := m.effects().Active[0].Starts
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillCGMoonlit, SourceID: 2000000, TargetID: 2000000, Result: 1})
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 181, X: 10, Y: 20, Visible: true})
	if len(m.effects().Active) != 1 || !m.effects().Active[0].Starts.Equal(start) || len(m.scheduledSounds) != 1 {
		t.Fatal("activation and ground units must not duplicate or restart the sphere")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMoon, ActorID: 2000000})
	if len(m.effects().Active) != 0 || ctx.World.Player.Opt3State&db.Opt3Moonlit != 0 {
		t.Fatal("Moonlit sphere survived its status ending")
	}
}

func TestClownGypsyHermodeGroundUnitOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillCGHermode, SourceID: 2000000})
	if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 1 || !m.actorSystem.ActorAnims[2000000].Loop {
		t.Fatal("Hermode ground activation must only begin the dance and sound")
	}
	unit := network.SkillUnitEntry{ID: 500, UnitID: 185, X: 15, Y: 20, Visible: true}
	m.applySkillUnitEntry(ctx, unit)
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.BottomHermode || len(m.scheduledSounds) != 1 {
		t.Fatal("Hermode's unit must own one silent cube")
	}
	start := m.effects().Active[0].Starts
	m.applySkillUnitEntry(ctx, unit)
	if !m.effects().Active[0].Starts.Equal(start) {
		t.Fatal("refresh must not restart the cube")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit.ID})
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
	if len(m.effects().Active)+len(m.actorSystem.ActorAnims) != 0 {
		t.Fatal("Hermode ground removal and status ending must stop the cube and dance")
	}
}

func TestDancerPerformanceOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, skill := range []uint16{db.SkillDCUglydance, db.SkillDCHumming, db.SkillDCDontforgetme, db.SkillDCFortunekiss, db.SkillDCServiceforyou} {
		for _, ground := range []bool{false, true} {
			m := &WorldMode{}
			if ground {
				m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000})
			} else {
				m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
			}
			sounds := 1
			if skill == db.SkillDCUglydance {
				sounds = 0
			}
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != sounds || !m.actorSystem.ActorAnims[2000000].PingPong {
				t.Fatalf("dance %d ground=%t must start only its pose and sound", skill, ground)
			}
			m.stopPerformance(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
			if len(m.actorSystem.ActorAnims) != 0 {
				t.Fatal("dance continued after status end")
			}
		}
	}
	m := &WorldMode{}
	for unit := uint16(171); unit <= 175; unit++ {
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: uint32(unit), UnitID: unit, X: 15, Y: 20, Visible: true})
	}
	if len(m.effects().Active) != 5 || len(m.scheduledSounds) != 0 {
		t.Fatal("five dance units must create five silent visuals")
	}
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 172, Value: 171}) || m.effects().Active[len(m.effects().Active)-1].EffectID != effects.BottomUglyDanceGround {
		t.Fatal("overlapping dances must change to Ugly Dance")
	}
	for unit := uint32(171); unit <= 175; unit++ {
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit})
	}
	if len(m.effects().Active) != 0 {
		t.Fatal("dance visual survived ground unit removal")
	}
}

func TestDancerWinkAndClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for unit := uint16(171); unit <= 175; unit++ {
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: unit, X: 15, Y: 20, Visible: true})
		effect := m.effects().Active[0]
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, effect.Starts.Add(20*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len()+billboards.Len() == 0 {
			t.Fatalf("dance unit %d drew nothing", unit)
		}
		if unit == 173 || unit == 175 {
			vertices := commands.Index(0).FieldByName("Vertices")
			count, red, green := 16, 155.0/255, 1.0
			if unit == 175 {
				count, red, green = 8, 1, 150.0/255
			}
			if vertices.Len() != count || math.Abs(vertices.Index(0).FieldByName("ColorR").Float()-red) > .001 || math.Abs(vertices.Index(0).FieldByName("ColorG").Float()-green) > .001 {
				t.Fatalf("dance unit %d has incorrect ribbon count or color", unit)
			}
		}
	}
	for skill, spec := range db.SkillEffects {
		if effects.IsDancerPerformance(skill) || skill == db.SkillDCWinkcharm {
			for _, sound := range spec.SFX {
				if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	m := &WorldMode{}
	n := network.SkillNoDamageNotify{SkillID: db.SkillDCWinkcharm, SourceID: 2000000, TargetID: 300, Result: 1}
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 1 {
		t.Fatal("Wink needs its kiss emote, directional sprite, and one sound")
	}
	if e := m.effects().Active[0]; e.EffectID != effects.Emotion || e.ActorID != n.SourceID || e.SpriteFrameOverride != 30 {
		t.Fatal("Wink must display the kiss emote on the caster")
	}
	e := m.effects().Active[1]
	if e.EffectID != effects.Wink || e.ActorID != n.SourceID || e.TargetID != n.TargetID || e.ActorOrigin.X != 20.5 || e.ProjectileDirection.X != -10 {
		t.Fatal("Wink must capture the target position and caster-to-target orientation")
	}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, e.Starts.Add(6*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
		t.Fatal("Wink sprite did not draw")
	}
	m.effects().RemoveActor(n.SourceID)
	if len(m.effects().Active) != 0 {
		t.Fatal("Wink must disappear when its caster leaves view")
	}
	m = &WorldMode{}
	start := time.Now()
	m.startSkillSourceCastAnimation(ctx, n.SourceID, n.SkillID, time.Second, start)
	m.effects().AddSkillCastEffects(ctx, n.SkillID, 0, n.SourceID, n.TargetID, 0, 0, time.Second, start, "test")
	if anim := m.actorSystem.ActorAnims[n.SourceID]; !anim.Loop || !anim.PingPong || anim.ActionFamily != sprites.ActionPCSkill {
		t.Fatal("Wink must dance during its cast")
	}
	if len(m.effects().Active) != 2 {
		t.Fatal("Wink cast needs only the normal circle and lock-on, without an aura")
	}
}

func TestGunslingerGroundAndCoinLifetime(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	for unit := uint16(190); unit <= 194; unit++ {
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 777, UnitID: unit, X: 12, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != 638+int(unit-190) || !m.effects().Active[0].Persistent {
			t.Fatalf("grenade %d: %+v", unit, m.effects().Active)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 777})
		if len(m.effects().Active) != 0 {
			t.Fatal("grenade survived unit removal")
		}
	}
	ctx.World.Player.Job = db.JobGunslinger
	ctx.Session.Selected.Job = db.JobGunslinger
	for _, count := range []uint16{1, 5, 6, 10, 0} {
		if !actors.ApplySpiritSphereCount(ctx, network.SpiritSphereCount{ActorID: 2000000, Count: count}) {
			t.Fatal("missing local actor")
		}
		m.actors().SyncActorSpiritEffects(ctx, 2000000)
		if count == 0 {
			if len(m.effects().Active) != 0 {
				t.Fatal("coins survived consumption")
			}
			continue
		}
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Chookgi3 || m.effects().Active[0].HitCount != int(count) || m.effects().Active[0].SpiritSpheres == nil {
			t.Fatalf("count %d: %+v", count, m.effects().Active)
		}
		e := m.effects().Active[0]
		p := e.SpiritSpheres
		m.actors().SyncActorSpiritEffects(ctx, 2000000)
		if m.effects().Active[0].SpiritSpheres != p {
			t.Fatal("unchanged count restarted coins")
		}
		stepped, jumped := effects.NewSpiritSpheres(e), effects.NewSpiritSpheres(e)
		for tick := 0; tick < 100; tick++ {
			stepped.Advance(e, e.Starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		}
		jumped.Advance(e, e.Starts.Add(99*db.EffectFrameDuration))
		if !reflect.DeepEqual(stepped, jumped) {
			t.Fatal("coins depend on render frame rate")
		}
	}
}

func TestHighWizardDamageAndGroundPackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobWizardH, db.JobWizardH
	for _, tc := range []struct {
		skill                 uint16
		effect, sounds, begin int
	}{
		{db.SkillHWMagiccrasher, 380, 1, 0}, {db.SkillHWNapalmvulcan, 401, 5, 1},
	} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: tc.skill, SourceSpeed: 500, Damage: 100, HitCount: 3, Action: network.ActorActionSkill}
		m.applyActorActionNotify(ctx, a)
		if len(m.effects().Active) != 4+tc.begin {
			t.Fatalf("skill %d needs its release effect, optional caster flare, and three hits", tc.skill)
		}
		release := m.effects().Active[tc.begin]
		if release.EffectID != tc.effect || !release.Starts.Equal(m.actorSystem.ActorAnims[2000000].Started) || !release.Starts.Before(m.effects().Active[tc.begin+1].Starts) {
			t.Fatal("release effect must precede the hit flashes")
		}
		if len(m.scheduledSounds) != tc.sounds+1+tc.begin {
			t.Fatalf("skill %d scheduled %d sounds, want %d", tc.skill, len(m.scheduledSounds), tc.sounds+1+tc.begin)
		}
		if tc.skill == db.SkillHWNapalmvulcan {
			for i, sound := range m.scheduledSounds[1+tc.begin:] {
				if sound.at.Sub(release.Starts) != time.Duration(20+10*i)*db.EffectFrameDuration {
					t.Fatal("Napalm Vulcan sound must play at ticks 20, 30, 40, 50, 60 regardless of hit count")
				}
			}
		}
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		if len(m.effects().Active) != 4+tc.begin {
			t.Fatal("splash packet replayed caster/impact effects")
		}
		a.Action, a.Damage = network.ActorActionSkill, 0
		m.effects().Active = nil
		m.scheduledSounds = nil
		m.applyActorActionNotify(ctx, a)
		if len(m.effects().Active) != 1+tc.begin || len(m.scheduledSounds) != tc.sounds+tc.begin {
			t.Fatal("a miss must retain release effects and sounds, without hit flashes")
		}
	}
	m := &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillHWGanbantein, X: 22, Y: 21})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != 223 || m.effects().Active[0].X != 22 || m.effects().Active[0].ActorID != 0 || m.actorSystem.ActorAnims[2000000].ActionFamily != sprites.ActionPCSkill || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].positioned {
		t.Fatal("Ganbantein needs its ground rings, skill pose and centered sound")
	}
	m = &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillHWGravitation, X: 22, Y: 21})
	if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\wizard_earthspike.wav" {
		t.Fatal("Gravitation ground notification must leave the field to unit entries")
	}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 184, X: 22, Y: 21, Visible: true})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != 522 || !m.effects().Active[0].Persistent {
		t.Fatal("Gravitation unit must own its stone and ice spikes")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
	if len(m.effects().Active) != 0 {
		t.Fatal("Gravitation field survived its unit")
	}
	m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: db.SkillHWGravitation, SourceID: 500, TargetID: 300, Action: network.ActorActionSkill, Damage: 100, HitCount: 1})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Hit1 || m.effects().Active[0].ActorID != 300 {
		t.Fatal("Gravitation damage must flash on the victim without duplicating the field")
	}
}

func TestHunterTrapPlacementAndTrigger(t *testing.T) {
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{
		{db.SkillHTLandmine, 147, effects.FirePillarBomb},
		{db.SkillHTAnklesnare, 145, 0}, {db.SkillHTShockwave, 148, 0},
		{db.SkillHTSandman, 149, effects.Sandman}, {db.SkillHTFlasher, 150, effects.Flasher},
		{db.SkillHTFreezingtrap, 151, effects.FreezingTrap}, {db.SkillHTBlastmine, 143, effects.BlastMineBomb},
		{db.SkillHTClaymoretrap, 152, effects.Claymore}, {db.SkillHTTalkiebox, 153, 0},
	} {
		t.Run(db.SkillResourceName[tc.skill], func(t *testing.T) {
			ctx := testutil.ArcherEffectContext()
			m := &WorldMode{}
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: tc.skill, X: 15, Y: 20})
			if len(m.effects().Active) != 0 || m.actorSystem.ActorAnims[2000000].ActionFamily != scene.SpriteActionPickup {
				t.Fatal("placing a trap must play pickup without detonating it")
			}
			m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
			if tc.unit == 148 && (len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Shockwave || !m.effects().Active[0].Persistent) {
				t.Fatal("Shock Wave must loop on the trap")
			}
			before := len(m.effects().Active)
			m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 500, TargetID: 300, SkillID: tc.skill, Damage: 100, HitCount: 3, Action: 5})
			hits := 0
			for _, e := range m.effects().Active[before:] {
				if e.EffectID != effects.Hit1 || e.ActorID != 300 {
					t.Fatal("victim damage must only create the small impact, not the trap explosion")
				}
				hits++
			}
			if (tc.unit == 147 || tc.unit == 151 || tc.unit == 143 || tc.unit == 152) && hits != 3 {
				t.Fatal("missing trap damage impacts")
			}
			look := uint32(db.SkillUnitUsedTraps)
			if tc.unit == 147 {
				look = 141
			}
			if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: look}) {
				t.Fatal("unhandled trap look")
			}
			m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
			if tc.effect == 0 {
				if len(m.effects().Active) != 0 {
					t.Fatal("expired trap retained its aura")
				}
				return
			}
			if len(m.effects().Active) != hits+1 {
				t.Fatalf("got %d explosions", len(m.effects().Active)-hits)
			}
			e := m.effects().Active[hits]
			if e.EffectID != tc.effect || e.ActorID != 0 || e.Persistent || e.X != 15 || e.Y != 20 {
				t.Fatalf("incorrect trap burst: %+v", e)
			}
			if tc.unit == 143 {
				found := false
				for _, sound := range m.scheduledSounds {
					if sound.paths[0] == "effect\\hunter_blastmine.wav" {
						found = true
						if !sound.at.Equal(e.Starts.Add(20 * db.EffectFrameDuration)) {
							t.Fatal("Blast Mine sound must wait twenty ticks")
						}
					}
				}
				if !found {
					t.Fatal("missing Blast Mine sound")
				}
			}
		})
	}
}

func TestHunterTrapUtilityTargetsTrapAndSurvivesRemoval(t *testing.T) {
	for _, skill := range []uint16{db.SkillHTRemovetrap, db.SkillHTSpringtrap} {
		ctx := testutil.ArcherEffectContext()
		ctx.World.Player.Job = db.JobHunter
		ctx.Session.Selected.Job = db.JobHunter
		ctx.Session.Selected.Option = db.EffectStateFalcon
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 149, X: 15, Y: 25, Visible: true})
		n := network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 500, SkillID: skill}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != 0 {
			t.Fatal("failed trap skill emitted effect")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
		if len(m.effects().Active) != 1 || m.effects().Active[0].ActorID != 0 || m.effects().Active[0].X != 15 || m.effects().Active[0].Y != 25 {
			t.Fatal("effect must stay at removed trap")
		}
		if skill == db.SkillHTSpringtrap && (m.actorSystem.Falcons[100] == nil || !m.actorSystem.Falcons[100].Attacking) {
			t.Fatal("Spring Trap did not send the falcon")
		}
	}
}

func TestHunterAnkleSnareClosesOnce(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 145, X: 15, Y: 20, Visible: false})
	m.skills().ApplySkillUnitUpdate(network.SkillUnitUpdate{ID: 500})
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: 145}) {
		t.Fatal("Ankle Snare close was ignored")
	}
	u := m.skillSystem.Units[500]
	rsm := &res.RSM{AnimLength: 1000}
	for _, tc := range []struct {
		age    time.Duration
		frame  int
		moving bool
	}{{0, 3, true}, {db.EffectFrameDuration, 103, true}, {time.Second, 903, false}} {
		frame, moving := skills.SkillUnitRSMFrame(rsm, res.RSWModel{}, u, u.AnimationStarted.Add(tc.age))
		if frame != tc.frame || moving != tc.moving {
			t.Fatalf("age %s: frame=%d moving=%t", tc.age, frame, moving)
		}
	}
}

func TestHunterTrapTargetAndTalkieBox(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 153, X: 15, Y: 20, Visible: true})
	projection := testutil.Projection(800, 600, 15, 20, 0)
	p := projection.Project(15.5, 20.5, 0)
	skill := session.Skill{ID: db.SkillHTRemovetrap, Type: skills.SkillTargetTrap, Level: 1, Range: 2}
	actor, ok := m.skills().ClickedSkillTarget(ctx, projection, skill, int(p.X), int(p.Y), time.Now())
	if !ok || actor.ID != 500 {
		t.Fatal("trap could not be selected")
	}

	m.actors().ApplySpeechBubble(ctx, network.ChatMessage{GID: 500, Text: "Beware : trap!"}, time.Now())
	m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: 140})
	b := m.actorSystem.SpeechBubbles[500]
	if b.Ground == nil || *b.Ground != [2]int{15, 20} || b.Text != "Beware : trap!" {
		t.Fatalf("wrong Talkie Box bubble: %+v", b)
	}

	if _, ok := m.skills().ClickedSkillTarget(ctx, projection, skill, int(p.X), int(p.Y), time.Now()); ok {
		t.Fatal("used trap still selectable")
	}
	m.effects().AddAt(ctx, effects.BlitzBeat, 300, time.Now())
	m.actors().ApplySpeechBubble(ctx, network.ChatMessage{GID: 300, Text: "Actor : hello"}, time.Now())
	if b := m.actorSystem.SpeechBubbles[300]; b.Ground != nil || b.Text != "hello" {
		t.Fatal("an actor's visual effects must not turn its speech into a ground bubble")
	}
}

func TestNinjaReleaseAndGroundOwnership(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		id     int
		owner  uint32
		sounds int
	}{{527, 0, 0, 1}, {536, 635, 300, 1}, {539, 636, 2000000, 1}, {541, 622, 2000000, 1}, {531, 0, 0, 0}, {532, 0, 0, 0}, {543, 0, 0, 0}, {529, 0, 0, 0}} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d sounds=%d", tc.skill, len(m.scheduledSounds))
		}
		if tc.id == 0 {
			if len(m.effects().Active) != 0 {
				t.Fatalf("skill %d invented an effect", tc.skill)
			}
		} else if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id || m.effects().Active[0].ActorID != tc.owner {
			t.Fatalf("skill %d wrong effect owner: %+v", tc.skill, m.effects().Active)
		}
		if tc.skill == 527 && m.scheduledSounds[0].positioned {
			t.Fatal("Tatami's sound is centered")
		}
		m.effects().Active, m.scheduledSounds = nil, nil
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now(), time.Now())
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d replayed its release on damage", tc.skill)
		}
	}
	for _, tc := range []struct {
		skill, unit uint16
		id          int
	}{{527, 188, 631}, {535, 189, 634}, {538, 187, 620}} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		testutil.ExpectEffectIDs(t, "ground is unit-owned", effects.SkillGroundEffectIDs(tc.skill))
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 777, UnitID: tc.unit, X: 12, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || !m.effects().Active[0].Persistent || m.effects().Active[0].EffectID != tc.id || (tc.unit != 189 && len(m.scheduledSounds) != 0) {
			t.Fatalf("unit %d ownership: %+v", tc.unit, m.effects().Active)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 777})
		if len(m.effects().Active) != 0 {
			t.Fatal("ground effect outlived its server unit")
		}
	}
}

func TestPriestGroundSpellsUseUnitLifetime(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{
		{db.SkillMGSafetywall, 126, effects.SafetyWall},
		{db.SkillMGFirewall, 127, effects.FireWall},
		{db.SkillPRSanctuary, 131, effects.BottomSanc},
		{db.SkillHPBasilica, 180, effects.BottomBasilica},
	} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d created an unowned ground effect", tc.skill)
		}
		if tc.skill == db.SkillHPBasilica && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\바실리카.wav") {
			t.Fatal("Basilica must play its sound once at release")
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != 900 {
			t.Fatalf("skill %d: missing unit-owned effect", tc.skill)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d left an orphan effect", tc.skill)
		}
	}
	for skill, id := range map[uint16]int{db.SkillPRBenedictio: effects.Benedictio, db.SkillPRMagnus: effects.Magnus} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != id || m.effects().Active[0].ActorID != 0 || m.effects().Active[0].X != 15 {
			t.Fatal("ground spell must launch its STR at the selected cell")
		}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 2 {
			t.Fatal("a second cast at the same cell must play its own activation")
		}
	}
	for skill, kind := range map[uint16]battle.FloaterKind{db.SkillPRSanctuary: battle.FloaterRecoveryHP, db.SkillMGSrecovery: battle.FloaterRecoverySP} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1, Amount: 777})
		if len(m.effects().Active)+len(m.scheduledSounds)+len(m.actorSystem.ActorAnims) != 0 || len(m.battleSystem.DamageFloaters) != 1 || m.battleSystem.DamageFloaters[0].Text != "777" || m.battleSystem.DamageFloaters[0].Kind != kind {
			t.Fatalf("skill %d: recovery should show the amount without a spell animation or sound", skill)
		}
	}
}

func TestProfessorGroundUnits(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{{404, 182, 405}, {405, 183, 404}} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 14, Y: 20, Level: 1})
		if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 || m.actorSystem.ActorAnims[2000000].ActionFamily != sprites.ActionPCSkill {
			t.Fatal("ground notification must animate the caster without duplicating the unit's effect")
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 14, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || !m.effects().Active[0].Persistent {
			t.Fatal("ground unit must own the native persistent effect")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.effects().Active) != 0 {
			t.Fatal("ground visual survived unit deletion")
		}
	}
}

func TestSageGroundSpellOwnership(t *testing.T) {
	for _, tc := range []struct {
		skill, unit uint16
		cast, field int
	}{
		{db.SkillSAVolcano, 154, effects.Volcano, effects.BottomVolcano},
		{db.SkillSADeluge, 155, effects.Deluge, effects.BottomDeluge},
		{db.SkillSAViolentgale, 156, effects.ViolentGale, effects.BottomViolent},
		{db.SkillSALandprotector, 157, effects.LandProtector, effects.BottomLand},
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		notify := network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20}
		m.applyGroundSkillNotify(ctx, notify)
		if len(m.effects().Active) != 1 {
			t.Fatalf("skill %d: missing activation", tc.skill)
		}
		cast := m.effects().Active[0]
		if cast.EffectID != tc.cast || cast.ActorID != 0 || cast.X != 15 || cast.Y != 20 || cast.Persistent {
			t.Fatalf("skill %d: activation must stay at the selected cell", tc.skill)
		}
		if len(m.actorSystem.ActorAnims) == 0 {
			t.Fatalf("skill %d: caster has no release animation", tc.skill)
		}
		// A recast at the same location must replay the brief activation.
		m.applyGroundSkillNotify(ctx, notify)
		if len(m.effects().Active) != 2 {
			t.Fatalf("skill %d: recast was suppressed", tc.skill)
		}
		for i := uint32(0); i < 3; i++ {
			entry := network.SkillUnitEntry{ID: 900 + i, UnitID: tc.unit, X: 15 + uint16(i), Y: 20, Visible: true}
			m.applySkillUnitEntry(ctx, entry)
			m.applySkillUnitEntry(ctx, entry)
		}
		if len(m.effects().Active) != 5 {
			t.Fatalf("skill %d: duplicated or missing field cells", tc.skill)
		}
		for i, field := range m.effects().Active[2:] {
			if field.EffectID != tc.field || !field.Persistent || field.X != 15+i || field.ActorID != uint32(900+i) {
				t.Fatalf("skill %d: incorrect field cell: %+v", tc.skill, field)
			}
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 901})
		if len(m.effects().Active) != 4 {
			t.Fatal("removing one field cell removed its neighbors or left an orphan")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 902})
		if len(m.effects().Active) != 2 {
			t.Fatal("field deletion left persistent visuals")
		}
	}
}

func TestWizardGroundSpellOwnership(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		id          int
	}{
		{db.SkillWZFirepillar, 135, effects.FirePillarOn},
		{db.SkillWZFirepillar, 136, effects.FirePillarBomb},
		{db.SkillWZIcewall, 141, effects.IceWall},
		{db.SkillWZQuagmire, 142, effects.Quagmire},
	} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d created an effect outside its unit lifetime", tc.skill)
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id {
			t.Fatalf("unit %d has no visual", tc.unit)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.effects().Active) != 0 {
			t.Fatalf("unit %d left an orphan visual", tc.unit)
		}
	}
	for _, skill := range []uint16{db.SkillWZStormgust, db.SkillWZVermilion, db.SkillWZMeteor, db.SkillWZHeavendrive} {
		m := &WorldMode{}
		for i := 0; i < 2; i++ {
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		}
		if len(m.effects().Active) != 2 || m.effects().Active[0].ActorID != 0 || m.effects().Active[0].X != 15 {
			t.Fatalf("skill %d must play every ground notification at the notified cell", skill)
		}
	}
}

func TestSkillUnitPositionUpdatesDoNotCreateActors(t *testing.T) {
	for _, packetID := range []uint16{0x0088, 0x01FF} {
		t.Run(fmt.Sprintf("0x%04X", packetID), func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Moving: true}
			ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
			mode := &WorldMode{}
			mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{
				ID: 9001, CreatorID: world.Player.ID, UnitID: 141, X: 12, Y: 20, Visible: true,
			})
			data := make([]byte, 10)
			binary.LittleEndian.PutUint16(data, packetID)
			binary.LittleEndian.PutUint32(data[2:], 9001)
			binary.LittleEndian.PutUint16(data[6:], 12)
			binary.LittleEndian.PutUint16(data[8:], 20)
			packet := network.Packet{ID: packetID, Data: data}
			mode.handleNetworkPacket(ctx, packet, time.Now())
			if len(world.Actors) != 0 {
				t.Fatalf("Ice Wall position update created a phantom actor: %+v", world.Actors)
			}
			if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.IceWall {
				t.Fatal("position update removed the Ice Wall visual")
			}
			mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9001})
			mode.handleNetworkPacket(ctx, packet, time.Now())
			if len(world.Actors) != 0 || len(mode.effects().Active) != 0 {
				t.Fatal("late position update resurrected the expired skill unit")
			}

			// Real Novices and the local player must still receive position fixes.
			world.UpsertActor(worldstate.Actor{ID: 300, Job: db.JobNovice, Name: "Alice", Appearance: true, X: 5, Y: 6, Moving: true})
			binary.LittleEndian.PutUint32(data[2:], 300)
			mode.handleNetworkPacket(ctx, packet, time.Now())
			actor := world.Actors[300]
			if actor.X != 12 || actor.Y != 20 || actor.Moving || actor.Name != "Alice" || !actor.Appearance {
				t.Fatalf("known actor was not updated correctly: %+v", actor)
			}
			binary.LittleEndian.PutUint32(data[2:], world.Player.ID)
			mode.handleNetworkPacket(ctx, packet, time.Now())
			if world.Player.X != 12 || world.Player.Y != 20 || world.Player.Moving || ctx.Session.PlayerX != 12 || ctx.Session.PlayerY != 20 {
				t.Fatal("local position update stopped working")
			}
		})
	}
}

func TestGroundSkillNotifyAddsCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 21, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 0 || effect.EffectID != effects.ThunderStorm || effect.X != 123 || effect.Y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 21, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.effects().Active) != 2 {
		t.Fatalf("repeated Thunderstorm casts = %d, want 2", len(mode.effects().Active))
	}
}

func TestPneumaGroundSkillAndUnitDoNotDuplicate(t *testing.T) {
	for _, unitFirst := range []bool{false, true} {
		ctx := testutil.KnightEffectContext()
		mode := &WorldMode{}
		cast := network.GroundSkillNotify{SkillID: db.SkillALPneuma, SourceID: 2000000, Level: 1, X: 12, Y: 23}
		unit := network.SkillUnitEntry{ID: 9001, CreatorID: 2000000, UnitID: 133, X: 12, Y: 23, Visible: true}
		if unitFirst {
			mode.applySkillUnitEntry(ctx, unit)
		}
		mode.applyGroundSkillNotify(ctx, cast)
		mode.applySkillUnitEntry(ctx, unit)
		if len(mode.effects().Active) != 1 {
			t.Fatalf("Pneuma must render once: %+v", mode.effects().Active)
		}
		effect := mode.effects().Active[0]
		if effect.EffectID != effects.Pneuma || effect.ActorID != unit.ID || !effect.Persistent || effect.X != 12 || effect.Y != 23 {
			t.Fatal("Pneuma must belong to its ground unit")
		}
		if mode.actorSystem.ActorAnims[2000000].ActionFamily != sprites.ActionPCSkill {
			t.Fatal("Pneuma release must animate its caster")
		}
		mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit.ID})
		if len(mode.effects().Active) != 0 {
			t.Fatal("Pneuma remained visible after its unit disappeared")
		}
	}
}

func TestGospelGroundSkillNotifyPlaysOneCastSoundOnCaster(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
	notify := network.GroundSkillNotify{SkillID: db.SkillPaGospel, SourceID: 2000000, Level: 10}

	mode.applyGroundSkillNotify(ctx, notify)
	mode.applyGroundSkillNotify(ctx, notify)

	if len(mode.effects().Active) != 1 {
		t.Fatalf("Gospel cast effects = %+v, want one deduplicated caster effect", mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.EffectID != effects.BottomGospel || effect.ActorID != 2000000 {
		t.Fatalf("Gospel cast effect = %+v, want EF_BOTTOM_GOSPEL on caster", effect)
	}
	if len(mode.scheduledSounds) != 1 || len(mode.scheduledSounds[0].paths) != 1 || mode.scheduledSounds[0].paths[0] != "effect\\가스펠.wav" {
		t.Fatalf("Gospel cast sounds = %+v, want one Gospel sound", mode.scheduledSounds)
	}
}

func TestGospelSkillUnitUsesPersistentGroundVisualWithoutSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9179, CreatorID: 2000000, UnitID: 179, X: 12, Y: 34, Visible: true})

	if len(mode.effects().Active) != 1 {
		t.Fatalf("Gospel unit effects = %+v, want one", mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.EffectID != effects.GospelGround || effect.ActorID != 9179 || effect.X != 12 || effect.Y != 34 || !effect.Persistent {
		t.Fatalf("Gospel unit effect = %+v, want persistent 370_ground at unit cell", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("Gospel unit sounds = %+v, want cast sound owned by poseffect only", mode.scheduledSounds)
	}
}

func TestSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9001, CreatorID: 2000000, UnitID: 126, X: 123, Y: 456, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 9001 || effect.EffectID != effects.SafetyWall || effect.X != 123 || effect.Y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9001})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
	}
}

func TestFireWallSkillUnitEntryUsesPersistentEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9007, CreatorID: 2000000, UnitID: 127, X: 12, Y: 34, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	effect := mode.effects().Active[0]
	if effect.ActorID != 9007 || effect.EffectID != effects.FireWall || effect.X != 12 || effect.Y != 34 {
		t.Fatalf("effect = %+v", effect)
	}
	if !effect.Persistent {
		t.Fatalf("fire wall skill unit effect is not persistent")
	}
	if effect.Expires.Sub(effect.Starts) < effects.SkillUnitEffectFallbackDuration {
		t.Fatalf("fire wall lifetime = %s, want skill unit fallback", effect.Expires.Sub(effect.Starts))
	}
	if effect.Duration != 0 {
		t.Fatalf("fire wall animation override = %s, want native component timing", effect.Duration)
	}
}

func TestSkillUnitEntryRefreshesExistingPersistentCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9008, CreatorID: 2000000, UnitID: 172, X: 12, Y: 34, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	first := mode.effects().Active[0]

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9008, CreatorID: 2000000, UnitID: 172, X: 13, Y: 35, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects after refresh = %d, want 1: %+v", len(mode.effects().Active), mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.ActorID != 9008 || effect.EffectID != effects.BottomHummingGround || effect.X != 13 || effect.Y != 35 {
		t.Fatalf("effect after refresh = %+v, want moved humming ground effect", effect)
	}
	if !effect.Starts.Equal(first.Starts) {
		t.Fatalf("effect start changed from %s to %s, want loop phase preserved", first.Starts, effect.Starts)
	}
	if effect.Expires.Before(first.Expires) {
		t.Fatalf("effect expiry moved backward from %s to %s", first.Expires, effect.Expires)
	}
}

func TestWizardSkillUnitEntrySchedulesCellSound(t *testing.T) {
	for _, tc := range []struct {
		name   string
		unitID uint16
		path   string
	}{
		{name: "ice wall", unitID: 141, path: "effect\\wizard_icewall.wav"},
		{name: "fire pillar", unitID: 136, path: "effect\\wizard_fire_pillar_b.wav"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			mode := &WorldMode{}
			ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

			mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9101, CreatorID: 2000000, UnitID: tc.unitID, X: 12, Y: 34, Visible: true})

			if len(mode.scheduledSounds) != 1 {
				t.Fatalf("scheduled sounds = %+v, want one", mode.scheduledSounds)
			}
			sound := mode.scheduledSounds[0]
			if len(sound.paths) != 1 || sound.paths[0] != tc.path {
				t.Fatalf("sound paths = %+v, want %s", sound.paths, tc.path)
			}
			if !sound.positioned || sound.actorID != 9101 || !sound.hasPosition {
				t.Fatalf("sound position metadata = %+v", sound)
			}
			x, y, ok := scheduledSoundPosition(ctx, sound, time.Now())
			if !ok || x != 12 || y != 34 {
				t.Fatalf("sound position = %.0f,%.0f ok=%t, want 12,34 true", x, y, ok)
			}
		})
	}
}

func TestStormGustGroundNotifySchedulesSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 34}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillWZStormgust, SourceID: 2000000, X: 12, Y: 34})

	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v, want one", mode.scheduledSounds)
	}
	sound := mode.scheduledSounds[0]
	if len(sound.paths) != 1 || sound.paths[0] != "effect\\wizard_stormgust.wav" {
		t.Fatalf("sound paths = %+v, want storm gust sfx", sound.paths)
	}
	x, y, ok := scheduledSoundPosition(ctx, sound, time.Now())
	if !ok || x != 12 || y != 34 {
		t.Fatalf("sound position = %.0f,%.0f ok=%t, want target position 12,34 true", x, y, ok)
	}
}

func TestQuagmireSkillUnitEntriesBuildFiveByFiveZone(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	const centerX, centerY = 50, 60
	entryID := uint32(9100)
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: entryID, CreatorID: 2000000, UnitID: 142, X: uint16(centerX + dx), Y: uint16(centerY + dy), Visible: true})
			entryID++
		}
	}

	if len(mode.effects().Active) != 25 {
		t.Fatalf("quagmire world effects = %d, want 25", len(mode.effects().Active))
	}
	cells := make(map[worldstate.WalkStep]bool, len(mode.effects().Active))
	for _, effect := range mode.effects().Active {
		if effect.EffectID != effects.Quagmire {
			t.Fatalf("effect id = %d, want quagmire: %+v", effect.EffectID, effect)
		}
		if !effect.Persistent {
			t.Fatalf("quagmire unit effect is not persistent: %+v", effect)
		}
		cells[worldstate.WalkStep{X: effect.X, Y: effect.Y}] = true
	}
	for y := centerY - 2; y <= centerY+2; y++ {
		for x := centerX - 2; x <= centerX+2; x++ {
			if !cells[worldstate.WalkStep{X: x, Y: y}] {
				t.Fatalf("missing quagmire cell %d,%d in %+v", x, y, cells)
			}
		}
	}
}

func TestTrapSkillUnitEntryAddsRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9201, CreatorID: 2000000, UnitID: 145, X: 12, Y: 34, Visible: true})

	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects = %+v, want none for trap placement model", mode.effects().Active)
	}
	unit, ok := mode.skillSystem.Units[9201]
	if !ok {
		t.Fatal("trap model was not tracked")
	}
	if unit.UnitID != 145 || unit.X != 12 || unit.Y != 34 || unit.ModelPath != db.SkillUnitModels[145].ModelPath {
		t.Fatalf("trap model = %+v", unit)
	}
	if len(unit.ModelFallbacks) != 1 || unit.ModelFallbacks[0] != "effect\\trap03.rsm" {
		t.Fatalf("trap fallbacks = %+v", unit.ModelFallbacks)
	}
	if unit.Scale != 0.15 || !unit.HasFixedFrame || unit.FixedFrame != 3 {
		t.Fatalf("trap render controls = scale %.3f fixed=%t frame=%d", unit.Scale, unit.HasFixedFrame, unit.FixedFrame)
	}
}

func TestRobrowserRSMTrapSkillUnitModelsUseTrapRenderControls(t *testing.T) {
	expected := map[uint16]string{
		143: "외부소품\\트랩03_3.rsm",
		144: "외부소품\\트랩02.rsm",
		145: "외부소품\\트랩01.rsm",
		147: "외부소품\\트랩03.rsm",
		148: "외부소품\\트랩03_6.rsm",
		149: "외부소품\\트랩03_4.rsm",
		150: "외부소품\\트랩03_5.rsm",
		151: "외부소품\\트랩03_2.rsm",
		152: "외부소품\\트랩04.rsm",
		153: "외부소품\\트랩05.rsm",
		210: "event\\3차트랩_변화01.rsm",
		211: "event\\3차트랩_변수01.rsm",
		212: "event\\3차트랩_변지01.rsm",
		213: "event\\3차트랩_변풍01.rsm",
		214: "event\\3차트랩_화01.rsm",
		215: "event\\3차트랩_수01.rsm",
		216: "event\\3차트랩_풍01.rsm",
		217: "event\\3차트랩_지01.rsm",
		229: "event\\3차트랩_가시01.rsm",
	}
	for unitID, wantPath := range expected {
		spec, ok := db.SkillUnitModels[unitID]
		if !ok {
			t.Fatalf("trap unit %d is not mapped", unitID)
		}
		if spec.ModelPath != wantPath {
			t.Fatalf("trap unit %d model = %q, want %q", unitID, spec.ModelPath, wantPath)
		}
		if spec.Scale != 0.15 || !spec.HasFixedFrame || spec.FixedFrame != 3 {
			t.Fatalf("trap unit %d render controls = scale %.3f fixed=%t frame=%d", unitID, spec.Scale, spec.HasFixedFrame, spec.FixedFrame)
		}
	}
}

func TestHiddenTrapSkillUnitUpdateRevealsRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9202, CreatorID: 3000000, UnitID: 151, X: 12, Y: 34, Visible: false})
	if len(mode.skillSystem.Units) != 0 {
		t.Fatalf("visible trap models = %+v, want none", mode.skillSystem.Units)
	}
	if _, ok := mode.skillSystem.HiddenUnits[9202]; !ok {
		t.Fatal("hidden trap model was not tracked")
	}

	mode.skills().ApplySkillUnitUpdate(network.SkillUnitUpdate{ID: 9202})

	if _, ok := mode.skillSystem.HiddenUnits[9202]; ok {
		t.Fatalf("hidden trap still tracked after reveal: %+v", mode.skillSystem.HiddenUnits[9202])
	}
	if unit, ok := mode.skillSystem.Units[9202]; !ok || unit.UnitID != 151 || unit.X != 12 || unit.Y != 34 {
		t.Fatalf("visible trap after reveal = %+v ok=%t", unit, ok)
	}
}

func TestUsedTrapLookChangeRemovesModelAndSpawnsTriggerBurst(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9203, CreatorID: 2000000, UnitID: 151, X: 12, Y: 34, Visible: true})
	if !mode.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 9203, Type: 0, Value: uint32(db.SkillUnitUsedTraps)}) {
		t.Fatal("used trap look change was not handled")
	}

	if len(mode.skillSystem.Units) != 0 {
		t.Fatalf("trap models after trigger = %+v, want none", mode.skillSystem.Units)
	}
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %+v, want freezing trap burst", mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.EffectID != effects.FreezingTrap || effect.ActorID != 0 || effect.X != 12 || effect.Y != 34 || effect.Persistent {
		t.Fatalf("trigger burst = %+v", effect)
	}
}

func TestSkillUnitDisappearRemovesRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9204, CreatorID: 2000000, UnitID: 145, X: 12, Y: 34, Visible: true})
	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9204})

	if len(mode.skillSystem.Units) != 0 || len(mode.skillSystem.HiddenUnits) != 0 || len(mode.effects().Active) != 0 {
		t.Fatalf("skill unit state after disappear models=%+v hidden=%+v effects=%+v", mode.skillSystem.Units, mode.skillSystem.HiddenUnits, mode.effects().Active)
	}
}

func TestWarpPortalSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9003, CreatorID: 2000000, UnitID: 128, X: 30, Y: 40, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 9003 || effect.EffectID != effects.Portal || effect.X != 30 || effect.Y != 40 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9003})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
	}
}

func TestPreWarpPortalSkillUnitEntryUsesReadyPortalEffectUntilDisappear(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9005, CreatorID: 2000000, UnitID: 129, X: 30, Y: 40, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	effect := mode.effects().Active[0]
	if effect.ActorID != 9005 || effect.EffectID != effects.ReadyPortal || effect.X != 30 || effect.Y != 40 {
		t.Fatalf("effect = %+v", effect)
	}
	if effect.Expires.Sub(effect.Starts) < effects.SkillUnitEffectFallbackDuration {
		t.Fatalf("portal lifetime = %s, want skill unit fallback", effect.Expires.Sub(effect.Starts))
	}
	if effect.Duration != 0 {
		t.Fatalf("portal animation override = %s, want native component timing", effect.Duration)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9005})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
	}
}

func TestWarpPortalSkillUnitLookChangeKeepsPortalAtSameCell(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9006, CreatorID: 2000000, UnitID: 129, X: 30, Y: 40, Visible: true})
	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.ReadyPortal {
		t.Fatalf("world effects before look change = %+v, want ready portal", mode.effects().Active)
	}

	if !mode.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 9006, Type: 0, Value: 128}) {
		t.Fatal("skill unit look change was not handled")
	}
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1: %+v", len(mode.effects().Active), mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.ActorID != 9006 || effect.EffectID != effects.Portal || effect.X != 30 || effect.Y != 40 {
		t.Fatalf("effect after look change = %+v, want portal on same unit cell", effect)
	}
}

func TestPneumaSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9002, CreatorID: 2000000, UnitID: 133, X: 123, Y: 456, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 9002 || effect.EffectID != effects.Pneuma || effect.X != 123 || effect.Y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9002})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
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
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 1100 || effect.EffectID != effects.Provoke || effect.X != 12 || effect.Y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestSkillNoDamageNotifyAddsSongTalkBeginEffects(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "dc_scream.txt"), []byte("SCREAM\r\n\tDancer line\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "ba_frostjoke.txt"), []byte("FROST JOKE\r\n\tBard line\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		skillID  uint16
		effectID int
		text     string
	}{
		{name: "scream", skillID: db.SkillDCScream, effectID: effects.TalkScream, text: "Dancer line"},
		{name: "frost joke", skillID: db.SkillBaFrostjoke, effectID: effects.TalkFrostJoke, text: "Bard line"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 150000, X: 10, Y: 20}
			sessionState := &session.Session{
				AccountID: 2000000,
				CharID:    150000,
				Selected:  session.Character{ID: 150000, Name: "Kivutar", Job: db.JobDancer},
			}
			mode := &WorldMode{}
			ctx := client.Context{Session: sessionState, World: world, Resources: manager}

			mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skillID, TargetID: 2000000, SourceID: 2000000, Result: 1})
			if len(mode.effects().Active) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
			}
			if effect := mode.effects().Active[0]; effect.ActorID != 2000000 || effect.EffectID != tc.effectID {
				t.Fatalf("effect = %+v, want actor 2000000 effect %d", effect, tc.effectID)
			}
			if bubble, ok := mode.actorSystem.SpeechBubbles[2000000]; !ok || bubble.Text != tc.text {
				t.Fatalf("account bubble = %+v ok=%t, want %q", bubble, ok, tc.text)
			}
			if bubble, ok := mode.actorSystem.SpeechBubbles[150000]; !ok || bubble.Text != tc.text {
				t.Fatalf("char bubble = %+v ok=%t, want %q", bubble, ok, tc.text)
			}
		})
	}
}

func TestSkillNoDamageNotifyAddsStealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 50, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 1100 || effect.EffectID != effects.Steal || effect.X != 12 || effect.Y != 22 {
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
	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.ActionFamily != sprites.ActionPCReadyFight || anim.HasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client READYFIGHT action", anim)
	}
	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.Endure {
		t.Fatalf("world effects = %+v, want Endure effect", mode.effects().Active)
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
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 1100 || effect.EffectID != effects.Heal || effect.X != 12 || effect.Y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("source cast animation missing")
	}
	if anim.ActionFamily != sprites.ActionPCSkill || anim.HasFixedMotion {
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
		ObjectType:    scene.ActorObjectTypeMob,
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

	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.ActionFamily != sprites.ActionPCSkill || anim.HasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
	found := false
	for _, effect := range mode.effects().Active {
		if effect.EffectID == effects.HealOffensive && effect.ActorID == 300 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("world effects = %+v, want offensive heal effect on target", mode.effects().Active)
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
		ObjectType:    scene.ActorObjectTypeMob,
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

	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.ActionFamily != sprites.ActionPCAttack2 {
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

	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.ActionFamily != sprites.ActionPCSkill || anim.HasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
	if len(mode.effects().Active) != 3 || mode.effects().Active[0].EffectID != effects.HealOffensive || mode.effects().Active[1].EffectID != effects.Hit1 || mode.effects().Active[2].EffectID != effects.BashHit {
		t.Fatalf("world effects = %+v, want offensive Heal followed by the native impact pair", mode.effects().Active)
	}
}

func TestSkillFailAckAddsConsoleErrorWithoutEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 6, Result: 0, Cause: 0})

	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects = %d, want 0", len(mode.effects().Active))
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

func TestSessionItemFromNetworkDoesNotMarkCardsAsEquipment(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      9,
		ItemID:     4001,
		Type:       db.ItemTypeCard,
		Location:   db.EquipAccessory1,
		Identified: true,
		Amount:     1,
		Equip:      true,
		Equipped:   true,
	})
	if item.Equip || item.Equipped || item.Location != db.EquipAccessory1 {
		t.Fatalf("card item = %+v, want non-equipment card with original location", item)
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
				{Index: 2, ItemID: 1202, Type: 4, Location: 0x0002, WearLocation: 0x0002, Equip: true, Equipped: true},
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
	if !item.Equipped || item.Location != db.EquipAmmo || item.WearLocation != db.EquipAmmo {
		t.Fatalf("ammo item after equip ack = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestApplyEquippedArrowMarksAmmoSlot(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 1750, Type: 10, Amount: 100, Location: db.EquipAmmo, WearLocation: db.EquipAmmo, Equip: true, Equipped: true},
				{Index: 9, ItemID: 1751, Type: 10, Amount: 50},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyEquippedArrow(ctx, network.EquippedArrow{Index: 9})

	if sessionState.Inventory.Items[0].Equipped || sessionState.Inventory.Items[0].WearLocation != 0 {
		t.Fatal("previous arrow stayed equipped")
	}
	item := sessionState.Inventory.Items[1]
	if !item.Equip || !item.Equipped || item.Location != db.EquipAmmo || item.WearLocation != db.EquipAmmo {
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
		{Index: 2, ItemID: 1607, Type: 5, Location: 0x0002, WearLocation: 0x0002, Equip: true, Equipped: true, Identified: true},
	})

	if sessionState.Selected.Weapon != 1607 || sessionState.Selected.Shield != 0 {
		t.Fatalf("selected weapon = %d shield = %d, want 1607/0", sessionState.Selected.Weapon, sessionState.Selected.Shield)
	}
	if sessionState.Characters[0].Weapon != 1607 {
		t.Fatalf("character list weapon = %d, want 1607", sessionState.Characters[0].Weapon)
	}
	if world.Player.Weapon != 1607 || world.Player.Shield != 0 {
		t.Fatalf("world player weapon = %d shield = %d, want 1607/0", world.Player.Weapon, world.Player.Shield)
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
	session.CloseStorage(ctx.Session)
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

func TestGraffitiLifetimeAndRetainedTerrainMesh(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	ctx.World.GND = &res.GND{Width: 20, Height: 20, Cells: make([]res.GNDCell, 400)}
	for i := range ctx.World.GND.Cells {
		ctx.World.GND.Cells[i].Heights = [4]float32{3, 3, 3, 3}
	}
	m := &WorldMode{mapScene: maps.System{Textures: map[string]*render.Image{
		"__effectfile_effect/alpabet.bmp|blackkey=true": render.NewImage(256, 256),
	}}}
	entry := network.SkillUnitEntry{ID: 900, CreatorID: 2000000, X: 20, Y: 20, UnitID: 176, Visible: true, HasMessage: true, Message: "Hello"}
	m.applySkillUnitEntry(ctx, entry)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	frame := render.NewFrame(800, 600)
	m.skills().DrawGraffiti(frame, ctx, projection)
	writing := m.skillSystem.Graffiti[900]
	if writing == nil || writing.Mesh == nil || writing.Mesh.VertexCount() != 18*18*4 || writing.Mesh.IndexCount() != 18*18*6 {
		t.Fatalf("missing 18-cell ground writing: %+v", writing)
	}
	if math.Abs(writing.Angle-math.Pi) > 1e-9 {
		t.Fatalf("native facing conversion = %g, want pi", writing.Angle)
	}
	vertices := reflect.ValueOf(writing.Mesh).Elem().FieldByName("vertices")
	for i := 0; i < vertices.Len(); i++ {
		if math.Abs(vertices.Index(i).FieldByName("Y").Float()-3.01) > .0001 {
			t.Fatal("graffiti did not follow the ground")
		}
	}
	cached := writing.Mesh
	m.skills().DrawGraffiti(render.NewFrame(800, 600), ctx, projection)
	if m.skillSystem.Graffiti[900].Mesh != cached {
		t.Fatal("rebuilt unchanged graffiti")
	}
	entry.Message = "Replaced"
	m.applySkillUnitEntry(ctx, entry)
	if len(m.skillSystem.Graffiti) != 1 || m.skillSystem.Graffiti[900].Message != "Replaced" || m.skillSystem.Graffiti[900].Mesh != nil {
		t.Fatal("entry refresh did not replace its text and geometry")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
	if len(m.skillSystem.Graffiti) != 0 {
		t.Fatal("Remover/expiry left the writing visible")
	}
	entry.Visible = false
	m.applySkillUnitEntry(ctx, entry)
	if len(m.skillSystem.Graffiti) != 0 {
		t.Fatal("hidden writing became visible")
	}
}

func TestGraffitiReleasesImages(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	ctx.World.GND = &res.GND{Width: 1, Height: 1, Cells: []res.GNDCell{{}}}
	for _, event := range []string{"replacement", "removal", "hidden", "ordinary unit", "map change", "leave"} {
		t.Run(event, func(t *testing.T) {
			m := &WorldMode{mapScene: maps.System{Textures: map[string]*render.Image{
				"__effectfile_effect/alpabet.bmp|blackkey=true": render.NewImage(256, 256),
			}}}
			entry := network.SkillUnitEntry{ID: 900, Visible: true, HasMessage: true, Message: "Hello"}
			m.applySkillUnitEntry(ctx, entry)
			m.skills().DrawGraffiti(render.NewFrame(800, 600), ctx, testutil.Projection(800, 600, 0, 0, 0))
			pixels := reflect.ValueOf(m.skillSystem.Graffiti[900].Mesh).Elem().FieldByName("texture").Elem().FieldByName("pix")
			if pixels.IsNil() {
				t.Fatal("writing has no pixels before removal")
			}
			switch event {
			case "replacement":
				entry.Message = "Replaced"
				m.applySkillUnitEntry(ctx, entry)
			case "removal":
				m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: entry.ID})
			case "hidden":
				entry.Visible = false
				m.applySkillUnitEntry(ctx, entry)
			case "ordinary unit":
				entry.HasMessage = false
				m.applySkillUnitEntry(ctx, entry)
			case "map change":
				m.releaseMapTextures()
			case "leave":
				m.Leave()
			}
			if !pixels.IsNil() {
				t.Fatal("writing retained its pixels instead of releasing its image group")
			}
			m.Leave()
		})
	}
}

func TestStarPlaceRequestOpensConfirmationAndAccepts(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := listener.Accept()
		accepted <- conn
	}()

	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	address := listener.Addr().(*net.TCPAddr)
	if err := netClient.Connect(context.Background(), address.IP.String(), address.Port); err != nil {
		t.Fatal(err)
	}
	serverConn := <-accepted
	if serverConn == nil {
		t.Fatal("server did not accept test client")
	}
	defer serverConn.Close()

	ctx := client.Context{Network: netClient, ScreenW: 800, ScreenH: 600}
	mode := &WorldMode{}
	packet := network.Packet{ID: network.PacketZCStarPlace, Data: []byte{0x53, 0x02, 0x01}}
	if next, stop := mode.handleNetworkPacket(ctx, packet, time.Now()); next != nil || stop {
		t.Fatalf("star place request changed mode: next=%T stop=%t", next, stop)
	}
	if !mode.ui.starPlaceConfirm.IsOpen() {
		t.Fatal("star place request did not open confirmation")
	}
	if !mode.ui.interactionModalOpen() {
		t.Fatal("star place confirmation did not block world interactions")
	}

	mode.ui.starPlaceConfirm.Confirm(ctx)

	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	want := network.BuildAgreeStarPlacePacket(1)
	got := make([]byte, len(want))
	if _, err := io.ReadFull(serverConn, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("star place reply = %x, want %x", got, want)
	}
}
