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
		ctx.Session.Dead = dead
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

func TestHeadlessUpdatesExpireVisualState(t *testing.T) {
	ctx := headlessTestContext(t, "function tick() end")
	m := NewWorldMode()
	m.Enter(ctx)
	t.Cleanup(m.Close)
	expired := time.Now().Add(-time.Second)
	future := time.Now().Add(time.Hour)
	m.damageFloaters = []damageFloater{{expires: future}}
	for i := 0; i < 500; i++ {
		m.damageFloaters = append(m.damageFloaters, damageFloater{expires: expired, text: "damage"})
		m.worldEffects = append(m.worldEffects, worldEffect{expires: expired})
		m.speechBubbles[uint32(i)] = speechBubble{expires: expired, text: "speech"}
		if _, err := m.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if len(m.damageFloaters) != 1 || len(m.worldEffects) != 0 || len(m.speechBubbles) != 0 {
		t.Fatal("visual state accumulated without drawing, or an active effect was discarded")
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

func TestHeadlessActorTimingLoadsWithoutDrawingRealData(t *testing.T) {
	ctx := client.Context{Resources: realDataManager(t), Config: config.Config{Headless: true}}
	m := NewWorldMode()
	actor := worldstate.Actor{ID: 123, Job: 0, Sex: 1, Head: 1}
	action, ok := m.actorResolvedAction(ctx, actor, spriteActionPCAttack1)
	if !ok || len(action.Animations) == 0 || action.DelayMS <= 0 {
		t.Fatal("actor animation timing was unavailable before drawing")
	}
}
