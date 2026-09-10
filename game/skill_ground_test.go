package game

import (
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

func groundSkillTestContext(t *testing.T) (client.Context, *WorldMode, net.Conn) {
	t.Helper()
	networkClient, serverConn := newBotTestConnection(t, 20080910)
	world := worldstate.New()
	world.GAT = flatWalkableGAT(64, 64)
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
			skill := session.Skill{ID: db.SkillMGFirewall, Level: 5, Type: skillTargetPlace, Range: 9}
			if tc.click {
				if err := mode.skills().Use(ctx, skill, "test"); err != nil {
					t.Fatal(err)
				}
				projection := newSceneProjectionForTarget(ctx.ScreenW, ctx.ScreenH, cellCenter(20), cellCenter(20), 0)
				point := projection.Project(cellCenter(float64(tc.x)), cellCenter(float64(tc.y)), 0)
				ctx.Input.SetMousePosition(int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
				mode.skills().HandleClick(ctx, projection, time.Now())
			} else if err := mode.skills().UseGround(ctx, skill, tc.x, tc.y, "", "test"); err != nil {
				t.Fatal(err)
			}
			walk, ok := network.BuildWalkToXYPacketForClientDate(tc.walkX, tc.walkY, 20080910)
			if !ok {
				t.Fatal("invalid expected walk destination")
			}
			readBotTestPackets(t, serverConn, walk)
			if pending := mode.pendingSkill; !pending.ground || pending.x != tc.x || pending.y != tc.y || pending.skill.Level != 5 {
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
			if mode.pendingSkill.readyAt.IsZero() {
				t.Fatal("ground skill was not scheduled after arrival")
			}
			mode.pendingSkill.readyAt = time.Now().Add(-time.Millisecond)
			mode.skills().ProcessPendingTarget(ctx)
			readBotTestPackets(t, serverConn, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 5, tc.x, tc.y, 20080910))
			if mode.pendingSkill.skill.ID != 0 {
				t.Fatal("ground skill remained pending after casting")
			}
			if len(mode.worldEffects) != 0 || len(mode.scheduledSounds) != 0 {
				t.Fatal("ground skill played effects before server confirmation")
			}
		})
	}
}

func TestGroundSkillInRangeCastsWithoutWalking(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	skill := session.Skill{ID: db.SkillMGFirewall, Level: 1, Type: skillTargetPlace, Range: 9}
	if err := mode.skills().UseGround(ctx, skill, 30, 20, "", "test"); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 1, 30, 20, 20080910))
	if mode.pendingSkill.skill.ID != 0 {
		t.Fatal("in-range ground skill was queued")
	}
}

func TestGroundSkillChaseCancels(t *testing.T) {
	for _, action := range []string{"escape", "right click", "new skill", "timeout", "map change"} {
		t.Run(action, func(t *testing.T) {
			ctx, mode, serverConn := groundSkillTestContext(t)
			skill := session.Skill{ID: db.SkillMGFirewall, Level: 1, Type: skillTargetPlace, Range: 9}
			if err := mode.skills().UseGround(ctx, skill, 31, 31, "", "test"); err != nil {
				t.Fatal(err)
			}
			walk, _ := network.BuildWalkToXYPacketForClientDate(24, 24, 20080910)
			readBotTestPackets(t, serverConn, walk)
			assertNoBotTestPacket(t, serverConn, func() error {
				ctx.World.Player.X, ctx.World.Player.Y = 24, 24
				mode.pendingSkill.readyAt = time.Now().Add(-time.Millisecond)
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
					if err := mode.skills().Use(ctx, session.Skill{ID: db.SkillALHeal, Level: 1, Type: skillTargetFriend, Range: 9}, "test"); err != nil {
						return err
					}
				case "timeout":
					mode.pendingSkill.expires = time.Now().Add(-time.Millisecond)
				case "map change":
					mode.handleNetworkPacket(ctx, testMapChangePacket("prontera", 24, 24), time.Now())
				}
				mode.skills().UpdatePendingTarget(ctx, "test", false)
				mode.skills().ProcessPendingTarget(ctx)
				return nil
			})
			if mode.pendingSkill.hasTarget() {
				t.Fatal("canceled ground target remained queued")
			}
		})
	}
}

func TestGroundSkillRetryKeepsDeadlineAndText(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	skill := session.Skill{ID: db.SkillHTTalkiebox, Level: 1, Type: skillTargetPlace, Range: 9}
	mode.pendingSkillText = pendingSkillTextTarget{skill: skill, x: 31, y: 31, source: "test"}
	mode.sendPendingSkillText(ctx, "Hello from the trap!")
	walk, _ := network.BuildWalkToXYPacketForClientDate(24, 24, 20080910)
	readBotTestPackets(t, serverConn, walk)
	deadline := mode.pendingSkill.expires
	mode.pendingSkill.lastChaseAt = time.Now().Add(-attackRetryInterval)
	mode.skills().UpdatePendingTarget(ctx, "test", false)
	readBotTestPackets(t, serverConn, walk)
	if !mode.pendingSkill.expires.Equal(deadline) {
		t.Fatal("retry extended the skill timeout")
	}
	ctx.World.Player.X, ctx.World.Player.Y = 24, 24
	mode.skills().UpdatePendingTarget(ctx, "test", false)
	mode.pendingSkill.readyAt = time.Now().Add(-time.Millisecond)
	mode.skills().ProcessPendingTarget(ctx)
	readBotTestPackets(t, serverConn, network.BuildUseSkillToGroundWithTextPacketForClientDate(skill.ID, 1, 31, 31, "Hello from the trap!", 20080910))
}

func TestGroundSkillUsesCompanionPositionAndMovePacket(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	ctx.Session.Mercenary = session.Companion{ID: 400, Active: true}
	mercenary := worldstate.Actor{ID: 400, X: 10, Y: 10, ObjectType: actorObjectTypeMercenary, HasObjectType: true}
	ctx.World.UpsertActor(mercenary)
	skill := session.Skill{ID: db.SkillMaLandmine, Level: 1, Type: skillTargetPlace, Range: 3}
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
	if mode.pendingSkill.readyAt.IsZero() {
		t.Fatal("ground skill did not use the companion's square range")
	}
	mode.pendingSkill.readyAt = time.Now().Add(-time.Millisecond)
	mode.skills().ProcessPendingTarget(ctx)
	readBotTestPackets(t, serverConn, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 1, 14, 14, 20080910))
}

func TestMapChangeCancelsSkillTextPrompt(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	ctx.UIManager = &worldModeTestUIManager{}
	skill := session.Skill{ID: db.SkillHTTalkiebox, Level: 1, Type: skillTargetPlace, Range: 9}
	mode.openSkillTextPrompt(ctx, skill, 31, 31, "test")
	mode.handleNetworkPacket(ctx, testMapChangePacket("prontera", 24, 24), time.Now())
	if mode.ui.skillTextPrompt.IsOpen() || mode.pendingSkillText.skill.ID != 0 {
		t.Fatal("map change retained the skill text prompt")
	}
	assertNoBotTestPacket(t, serverConn, func() error {
		mode.sendPendingSkillText(ctx, "Too late")
		return nil
	})
}

func TestUnreachableGroundTargetReplacesPreviousCast(t *testing.T) {
	ctx, mode, serverConn := groundSkillTestContext(t)
	ctx.World.GAT = flatWalkableGAT(12, 5)
	ctx.World.Player.X, ctx.World.Player.Y = 10, 2
	skill := session.Skill{ID: db.SkillMGFirewall, Level: 1, Type: skillTargetPlace, Range: 3}
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
		mode.pendingSkill.readyAt = time.Now().Add(-time.Millisecond)
		mode.skills().ProcessPendingTarget(ctx)
		return nil
	})
	if mode.pendingSkill.hasTarget() {
		t.Fatal("unreachable selection left a previous target queued")
	}
}
