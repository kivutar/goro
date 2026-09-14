package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestHeadlessCombatKeepsStateWithoutVisuals(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		root = t.TempDir()
	}
	resources, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	sess := session.New()
	sess.AccountID, sess.CharID = 2000000, 150000
	sess.Vitals.HP, sess.Vitals.MaxHP = 50, 100
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 1002, X: 11, Y: 20})
	ctx := client.Context{Config: config.Config{Headless: true}, Resources: resources, Session: sess, World: world}
	mode := NewWorldMode()

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: sess.AccountID, TargetID: 300, SkillID: db.SkillMGFirebolt, DelayTime: 500})
	if anim := mode.actorAnims[sess.AccountID]; anim.duration != 500*time.Millisecond {
		t.Fatalf("cast duration = %v, want server's 500 ms", anim.duration)
	}
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 300, TargetID: sess.AccountID, SourceSpeed: 600, TargetSpeed: 250, Damage: 10,
	})
	if actor := world.Actors[300]; actor.AITargetID != sess.AccountID || actor.AIMotion != aiMotionAttack {
		t.Fatalf("monster combat state was lost: %+v", actor)
	}
	if anim := mode.actorAnims[300]; anim.duration != 600*time.Millisecond {
		t.Fatalf("attack duration = %v, want server's 600 ms", anim.duration)
	}
	mode.applyRecovery(ctx, network.Recovery{StatusID: network.StatusHP, Amount: 20})
	if sess.Vitals.HP != 70 {
		t.Fatalf("HP = %d, want 70 after recovery", sess.Vitals.HP)
	}
	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: sess.AccountID, TargetID: sess.AccountID, SkillID: db.SkillALAngelus, Result: 1})
	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: sess.AccountID, SkillID: db.SkillMGFirewall, X: 10, Y: 20})
	mode.applyEmotionNotify(ctx, network.EmotionNotify{GID: 300, Type: 0})
	mode.applySpeechBubble(ctx, network.ChatMessage{GID: 300, Text: "hello"}, time.Now())
	if err := mode.applyNPCCutin(ctx, network.NPCCutin{Image: "headless-unused", Position: network.NPCCutinLeft}); err != nil {
		t.Fatal(err)
	}
	mode.reloadPlayerSpriteView(ctx, "test")
	mode.mercenaryHumanoidSpriteView(ctx, worldstate.Actor{ID: 400, Job: 6017})
	mode.nonPCGR2ModelView(ctx, worldstate.Actor{ID: 500, Job: 1288})
	mode.startActorDeath(ctx, 300)
	if _, dead := mode.actorDeaths[300]; !dead {
		t.Fatal("dead monster was not excluded from bot targets")
	}
	if len(mode.worldEffects) != 0 || len(mode.damageFloaters) != 0 || len(mode.speechBubbles) != 0 || len(mode.actorCastBars) != 0 {
		t.Fatal("headless combat retained visuals that require drawing to expire")
	}
	if mode.playerView != nil || len(mode.nonPCViews) != 0 || len(mode.nonPCViewMiss) != 0 || len(mode.mercenaryViews) != 0 || len(mode.mercenaryViewMiss) != 0 {
		t.Fatal("headless combat loaded or attempted to load actor sprites")
	}
	if len(mode.gr2Models) != 0 || len(mode.gr2ModelMiss) != 0 {
		t.Fatal("headless combat loaded or attempted to load actor models")
	}
	if len(mode.effectViews) != 0 || len(mode.effectViewMiss) != 0 || len(mode.strEffects) != 0 || len(mode.strEffectMiss) != 0 {
		t.Fatal("headless combat loaded or attempted to load effect assets")
	}
	if len(mode.textures) != 0 || len(mode.textureMiss) != 0 {
		t.Fatal("headless mode loaded or attempted to load textures")
	}
}

func TestHeadlessSongSkillsStillSendChat(t *testing.T) {
	root := t.TempDir()
	for file, text := range map[string]string{
		"dc_scream.txt":    "SCREAM\r\n\tDancer line\r\n",
		"ba_frostjoke.txt": "FROST JOKE\r\n\tBard line\r\n",
	} {
		if err := os.WriteFile(filepath.Join(root, file), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	resources, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	networkClient, conn := newBotTestConnection(t, 20080910)
	sess := session.New()
	sess.AccountID, sess.CharID = 2000000, 150000
	sess.Selected = session.Character{ID: sess.CharID, Name: "Tester", Job: db.JobDancer}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID, X: 10, Y: 20}
	ctx := client.Context{Config: config.Config{Headless: true}, Resources: resources, Session: sess, World: world, Network: networkClient}
	mode := NewWorldMode()
	for _, tc := range []struct {
		skill uint16
		text  string
	}{{db.SkillDCScream, "Dancer line"}, {db.SkillBaFrostjoke, "Bard line"}} {
		mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: sess.AccountID, TargetID: sess.AccountID, SkillID: tc.skill, Result: 1})
		readBotTestPackets(t, conn, network.BuildGlobalChatPacketForClientDate("Tester", tc.text, 20080910))
	}
	if len(mode.worldEffects) != 0 || len(mode.speechBubbles) != 0 {
		t.Fatal("song skills created headless visuals")
	}
}

func TestHeadlessBotWaitsForProgressAndCanRevive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
ticks = 0
function tick()
    ticks = ticks + 1
    if goro.player().dead then assert(goro.revive()) end
end
`), 0600); err != nil {
		t.Fatal(err)
	}
	networkClient, conn := newBotTestConnection(t, 20080910)
	sess := session.New()
	sess.AccountID, sess.CharID = 2000000, 150000
	sess.Vitals.HP, sess.Vitals.MaxHP = 100, 100
	sess.Inventory.Items = []session.InventoryItem{{ItemID: client.TokenOfSiegfriedItemID, Amount: 1}}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID}
	ctx := client.Context{
		Config:  config.Config{Headless: true, Script: config.ScriptConfig{Path: path}},
		Session: sess, World: world, Network: networkClient,
	}
	mode := NewWorldMode()
	var err error
	mode.bot, err = newLuaBot(ctx, mode, path)
	if err != nil {
		t.Fatal(err)
	}
	defer mode.bot.close()
	update := func(want string) {
		t.Helper()
		mode.bot.nextTick = time.Time{}
		if next, err := mode.Update(ctx); next != nil || err != nil {
			t.Fatalf("headless update = %v, %v", next, err)
		}
		if mode.bot.disabled {
			t.Fatal("bot script failed")
		}
		if got := mode.bot.state.GetGlobal("ticks").String(); got != want {
			t.Fatalf("bot ticks = %s, want %s", got, want)
		}
	}
	mode.startServerProgress(ctx, network.ProgressBar{Duration: time.Hour}, time.Now())
	update("0")
	mode.serverProgress.started = time.Now().Add(-2 * time.Hour)
	update("1")
	readBotTestPackets(t, conn, network.BuildProgressBarDonePacket())
	mode.startActorDeath(ctx, sess.AccountID)
	update("2")
	readBotTestPackets(t, conn, network.BuildAutoRevivePacket())
}
