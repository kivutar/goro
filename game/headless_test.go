package game

import (
	"context"
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

func headlessTestContext(t *testing.T, script string) client.Context {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "bot.lua")
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Config:    config.Config{Headless: true, Window: config.WindowConfig{Width: 800, Height: 600}, Script: config.ScriptConfig{Path: path}},
		Resources: &res.Manager{Root: root}, Session: session.New(), World: worldstate.New(),
		Input: input.NewState(), Network: network.NewClient(20080910, false),
	}
	ctx.World.MapName = ""
	t.Cleanup(ctx.Network.Close)
	return ctx
}

func TestHeadlessBotTicksWhileDeadAndRespectsServerProgress(t *testing.T) {
	ctx := headlessTestContext(t, "ticks = 0; function tick() ticks = ticks + 1 end")
	ctx.Session.CharID = 150000
	ctx.World.Player.ID = ctx.Session.CharID
	m := NewWorldMode()
	t.Cleanup(m.Close)
	if next := m.Enter(ctx); next != nil {
		t.Fatal("unexpected redirect")
	}
	if _, err := m.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if m.bot == nil || m.mapFade.phase != mapFadeNone {
		t.Fatal("headless world waited for a rendered frame")
	}
	for i, dead := range []bool{false, true} {
		if dead {
			m.startActorDeath(ctx, ctx.Session.CharID)
			if !ctx.Session.Dead {
				t.Fatal("death notification did not mark the player dead")
			}
		}
		m.bot.nextTick = time.Time{}
		if _, err := m.Update(ctx); err != nil {
			t.Fatal(err)
		}
		if got := m.bot.state.GetGlobal("ticks"); got != lua.LNumber(i+1) {
			t.Fatalf("ticks = %v, dead=%t", got, dead)
		}
	}
	m.startServerProgress(ctx, network.ProgressBar{Duration: time.Hour}, time.Now())
	m.bot.nextTick = time.Time{}
	if _, err := m.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if got := m.bot.state.GetGlobal("ticks"); got != lua.LNumber(2) {
		t.Fatalf("script ran during server progress: ticks=%v", got)
	}
	bot := m.bot
	m.Close()
	if bot.state != nil {
		t.Fatal("leaving the world retained the Lua state")
	}
}

func TestHeadlessReportsScriptErrors(t *testing.T) {
	for _, script := range []string{"invalid lua !!!", "error('initialization')", "function tick() error('tick') end", "function input() error('input') end"} {
		t.Run(script, func(t *testing.T) {
			ctx := headlessTestContext(t, script)
			m := NewWorldMode()
			t.Cleanup(m.Close)
			m.Enter(ctx)
			_, err := m.Update(ctx)
			if err == nil {
				m.bot.nextTick = time.Time{}
				_, err = m.Update(ctx)
			}
			if err == nil || !strings.Contains(err.Error(), "script") {
				t.Fatalf("script failure = %v", err)
			}
		})
	}
}

func TestHeadlessLuaCanBeCancelled(t *testing.T) {
	for _, script := range []string{"while true do end", "function tick() while true do end end"} {
		ctx := headlessTestContext(t, script)
		scriptCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		ctx.ScriptContext = scriptCtx
		m := NewWorldMode()
		m.Enter(ctx)
		_, err := m.Update(ctx)
		if err == nil {
			m.bot.nextTick = time.Time{}
			_, err = m.Update(ctx)
		}
		cancel()
		m.Close()
		if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Fatalf("cancellation = %v", err)
		}
	}
}

func TestHeadlessFailsInsteadOfWaitingForWindows(t *testing.T) {
	ctx := headlessTestContext(t, "function tick() end")
	login := NewLoginMode()
	login.Enter(ctx)
	login.applyAccountRefuseLogin(ctx, network.AccountRefuseLogin{ErrorCode: 1})
	if _, err := login.Update(ctx); err == nil || !strings.Contains(err.Error(), "Incorrect Password") {
		t.Fatalf("login refusal = %v", err)
	}
	login = NewLoginMode()
	login.headlessDeadline = time.Now().Add(-time.Second)
	if _, err := login.Update(ctx); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("login timeout = %v", err)
	}
	m := NewWorldMode()
	m.Enter(ctx)
	t.Cleanup(m.Close)
	m.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 1, Message: "Choose something"})
	if _, err := m.Update(ctx); err == nil || !strings.Contains(err.Error(), "NPC dialog") {
		t.Fatalf("NPC dialog = %v", err)
	}
	ctx.World.MapName = "missing.gat"
	manager := NewManager(ctx, NewWorldMode())
	if err := manager.Update(); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing map = %v", err)
	}
}

func TestHeadlessCombatKeepsMovementTimingWithoutVisuals(t *testing.T) {
	ctx := headlessTestContext(t, "")
	ctx.Session.CharID = 150000
	ctx.World.Player = worldstate.Actor{
		ID: 150000, Job: 1, X: 15, Y: 20,
		Moving: true, FromX: 10, FromY: 20, ToX: 15, ToY: 20,
		MoveStarted: time.Now(), MoveDuration: 5 * time.Second,
		MovePath: []worldstate.WalkStep{{X: 10, Y: 20}, {X: 15, Y: 20}},
	}
	ctx.World.UpsertActor(worldstate.Actor{ID: 400, Job: 1002, X: 11, Y: 20})
	m := NewWorldMode()
	m.Enter(ctx)
	t.Cleanup(m.Close)
	m.lockedAttackID = 400
	before := time.Now()
	m.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 400, TargetID: 150000, SourceSpeed: 580, TargetSpeed: 480, Damage: 42,
	})
	after := time.Now()
	if len(m.scheduledStops) == 0 {
		t.Fatal("hit did not schedule a movement stop")
	}
	hit := m.scheduledStops[0]
	if hit.at.Before(before.Add(580*time.Millisecond)) || hit.at.After(after.Add(580*time.Millisecond)) {
		t.Fatalf("hit time %v did not use server attack duration", hit.at)
	}
	if !hit.resumeWalk || hit.resumeAt.Sub(hit.at) != 480*time.Millisecond {
		t.Fatalf("hurt stop = %+v, want resume after server hurt duration", hit)
	}
	m.processScheduledActorStops(ctx, hit.at)
	if ctx.World.Player.Moving {
		t.Fatal("hit did not pause walking")
	}
	m.processScheduledWalkResumes(ctx, hit.resumeAt)
	if !ctx.World.Player.Moving || ctx.World.Player.ToX != 15 || ctx.World.Player.ToY != 20 {
		t.Fatalf("walking did not resume toward the original destination: %+v", ctx.World.Player)
	}

	ctx.Session.Vitals.HP, ctx.Session.Vitals.MaxHP = 50, 100
	m.applyRecovery(ctx, network.Recovery{StatusID: network.StatusHP, Amount: 25})
	if ctx.Session.Vitals.HP != 75 {
		t.Fatalf("recovery HP = %d, want 75", ctx.Session.Vitals.HP)
	}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 150000, SkillID: db.SkillMGFirebolt, DelayTime: 1200})
	if ctx.World.Player.Moving {
		t.Fatal("casting did not stop walking")
	}
	m.applyEmotionNotify(ctx, network.EmotionNotify{GID: 150000, Type: 1})
	m.applySpeechBubble(ctx, network.ChatMessage{GID: 150000, Text: "hello"}, time.Now())
	m.addWorldEffectAtCellLifetime(ctx, effectHeal, 150000, 10, 20, time.Now(), time.Second, true)
	m.addWorldEffectAtCellDurationSize(ctx, effectHeal, 150000, 10, 20, time.Now(), time.Second, 1)
	if len(m.actorAnims)+len(m.damageFloaters)+len(m.worldEffects)+len(m.speechBubbles)+len(m.actorCastBars) != 0 {
		t.Fatal("headless combat allocated visual state")
	}
}

func TestHeadlessLoginRejectsEmptyServerAndCharacterLists(t *testing.T) {
	for _, tc := range []struct {
		id   uint16
		size int
		want string
	}{
		{0x0069, 47, "no character servers"},
		{0x006b, 24, "slot 0 is empty"},
	} {
		ctx := headlessTestContext(t, "")
		ctx.Resources.ClientInfo.Connections = []res.Connection{{Address: "127.0.0.1"}}
		var server net.Conn
		ctx.Network, server = newBotTestConnection(t, 20080910)
		packet := make([]byte, tc.size)
		binary.LittleEndian.PutUint16(packet, tc.id)
		binary.LittleEndian.PutUint16(packet[2:4], uint16(len(packet)))
		if _, err := server.Write(packet); err != nil {
			t.Fatal(err)
		}
		m := NewLoginMode()
		m.Enter(ctx)
		var err error
		for deadline := time.Now().Add(time.Second); err == nil && time.Now().Before(deadline); {
			_, err = m.Update(ctx)
			if err == nil {
				time.Sleep(time.Millisecond)
			}
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("packet %04x: error = %v, want %q", tc.id, err, tc.want)
		}
	}
}

func TestHeadlessSkipsSpriteResourcesRealData(t *testing.T) {
	ctx := headlessTestContext(t, "")
	ctx.Resources = realDataManager(t)
	m := NewWorldMode()
	m.Enter(ctx)
	t.Cleanup(m.Close)
	m.reloadPlayerSpriteView(ctx, "test look change")
	for _, job := range []int16{0, 1002, 1288} {
		actor := worldstate.Actor{ID: 123, Job: job, Sex: 1, Head: 1}
		if _, ok := m.actorResolvedAction(ctx, actor, spriteActionPCAttack1); ok {
			t.Fatal("headless timing resolved a sprite animation")
		}
		if got := m.actorActionDuration(ctx, actor, deathActionFamilyForActor(actor), defaultDeathAnimationDuration); got != defaultDeathAnimationDuration {
			t.Fatalf("death duration = %v, want fixed fallback", got)
		}
		m.nonPCSpriteView(ctx, actor)
	}
	m.applyEmotionNotify(ctx, network.EmotionNotify{GID: ctx.World.Player.ID, Type: 1})
	if m.playerView != nil || m.shadowView != nil || m.cursorView != nil ||
		len(m.actorViews)+len(m.nonPCViews)+len(m.gr2Models)+len(m.effectViews) != 0 ||
		len(m.actorViewMiss)+len(m.nonPCViewMiss)+len(m.gr2ModelMiss)+len(m.effectViewMiss) != 0 {
		t.Fatal("headless mode tried to load sprite or model resources")
	}
}

func TestHeadlessSongSkillStillSendsChat(t *testing.T) {
	ctx := headlessTestContext(t, "")
	ctx.Session.CharID = 150000
	ctx.Session.Selected = session.Character{ID: 150000, Name: "Bard"}
	ctx.World.Player.ID = 150000
	data := filepath.Join(ctx.Resources.Root, "data")
	if err := os.MkdirAll(data, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "ba_frostjoke.txt"), []byte("FROST JOKE\r\n\tBard line\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var conn net.Conn
	ctx.Network, conn = newBotTestConnection(t, 20080910)
	m := NewWorldMode()
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SourceID: 150000, TargetID: 150000, SkillID: db.SkillBaFrostjoke, Result: 1,
	})
	readBotTestPackets(t, conn, network.BuildGlobalChatPacketForClientDate("Bard", "Bard line", 20080910))
	if len(m.worldEffects)+len(m.speechBubbles) != 0 {
		t.Fatal("song skill created visual state")
	}
}
