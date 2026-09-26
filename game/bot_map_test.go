package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

func TestWASDHeldKeyContinuesAfterMapChange(t *testing.T) {
	for _, sameMap := range []bool{false, true} {
		t.Run(fmt.Sprintf("same_map=%t", sameMap), func(t *testing.T) {
			ctx, mode := newBotChatTestContext(t, filepath.Join("..", "scripts", "wasd.lua"))
			ctx.Input = input.NewState()
			key, _ := input.KeyCodeFromName("KeyD")
			ctx.Input.SetKeyCode(key, true)
			netClient, server := newBotTestConnection(t, 20080910)
			ctx.Network = netClient
			mode.updateBotInput(ctx, true)
			walk, _ := network.BuildWalkToXYPacketForClientDate(18, 10, 20080910)
			readBotTestPackets(t, server, walk)

			if !sameMap {
				mode = mode.nextWorldMode()
			}
			ctx.World.SetPlayerPosition(40, 40, 0)
			mode.botMapChanged(ctx)
			mode.walkCooldownUntil = time.Time{}
			mode.updateBotInput(ctx, true)
			walk, _ = network.BuildWalkToXYPacketForClientDate(48, 40, 20080910)
			readBotTestPackets(t, server, walk)
			if !ctx.Input.KeyCodeDown(key) {
				t.Fatal("map callback released the held physical key")
			}
		})
	}
}

func TestMapChangedActionsFollowLoadAcknowledgement(t *testing.T) {
	for _, headless := range []bool{false, true} {
		t.Run(fmt.Sprintf("headless=%t", headless), func(t *testing.T) {
			ctx := mapRecoveryTestContext(t)
			ctx.Config.Headless = headless
			ctx.Session.Selected = ctx.Session.Characters[0]
			netClient, server := newBotTestConnection(t, 20080910)
			ctx.Network = netClient
			writeTestGAT(t, ctx.Resources.Root, ctx.World.MapName)
			ctx.Config.Script.Path = filepath.Join(t.TempDir(), "map.lua")
			if err := os.WriteFile(ctx.Config.Script.Path, []byte(`
function map_changed() assert(goro.message("map ready")) end
`), 0o600); err != nil {
				t.Fatal(err)
			}
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			mode = mode.nextWorldMode()
			if next := mode.Enter(ctx); next != nil {
				t.Fatalf("map loading failed: %T", next)
			}
			want := append(network.BuildLoadEndAckPacket(), network.BuildGlobalChatPacketForClientDate("Novice", "map ready", 20080910)...)
			readBotTestPackets(t, server, want)
			if next := mode.handleMapChange(ctx, network.MapChange{MapName: ctx.World.MapName}); next != nil {
				t.Fatalf("same-map warp replaced the world mode: %T", next)
			}
			readBotTestPackets(t, server, want)
		})
	}
}

func TestLuaBotMapChangePreservesLocalsAndRebindsCachedAPI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
local maps = 0
local hp, players, walk = goro.hp, goro.players, goro.walk
local is_down = goro.keyboard.is_down
function map_changed()
    maps = maps + 1
    seen = { maps=maps, hp=hp(), dead=players()[1].dead }
end
function input() seen.key = is_down("KeyW") end
function tick() assert(walk(20, 10)) end
`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, old := newBotChatTestContext(t, path)
	old.actorDeaths = map[uint32]time.Time{300: time.Now()}
	old.walkCooldownUntil = time.Now().Add(time.Hour)
	bot := old.bot
	next := old.nextWorldMode()
	if next.bot != bot || old.bot != nil {
		t.Fatal("map transition did not transfer ownership of the Lua state")
	}
	ctx.Session = session.New()
	ctx.Session.Vitals.HP = 77
	ctx.Input = input.NewState()
	key, _ := input.KeyCodeFromName("KeyW")
	ctx.Input.SetKeyCode(key, true)
	netClient, server := newBotTestConnection(t, 20080910)
	ctx.Network = netClient
	next.botMapChanged(ctx)
	if bot.disabled {
		t.Fatal("map callback failed")
	}
	seen := bot.state.GetGlobal("seen").(*lua.LTable)
	assertLuaNumber(t, seen, "maps", 1)
	assertLuaNumber(t, seen, "hp", 77)
	assertLuaBool(t, seen, "dead", false)
	if err := bot.inputFrame(true); err != nil {
		t.Fatal(err)
	}
	assertLuaBool(t, seen, "key", true)
	if err := bot.tick(); err != nil {
		t.Fatal(err)
	}
	walk, _ := network.BuildWalkToXYPacketForClientDate(20, 10, 20080910)
	readBotTestPackets(t, server, walk)
	if next.walkCooldownUntil.IsZero() {
		t.Fatal("cached walk function still targeted the old world mode")
	}
	last := next.nextWorldMode()
	last.botMapChanged(ctx)
	assertLuaNumber(t, bot.state.GetGlobal("seen").(*lua.LTable), "maps", 2)
}

func TestLuaBotMapCallbackFailureAndCharacterSwitchCloseState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`function map_changed() error("test") end`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, mode := newBotChatTestContext(t, path)
	mode.botMapChanged(ctx)
	bot := mode.bot
	if !bot.disabled || bot.state != nil {
		t.Fatal("failing map callback did not disable and close Lua")
	}
	next := mode.nextWorldMode()
	next.botMapChanged(ctx)
	if next.bot != bot || !bot.disabled {
		t.Fatal("map transition restarted a failed script")
	}
	ctx, mode = newBotChatTestContext(t, path)
	bot = mode.bot
	mode.nextCharacterSelectMode(ctx)
	if mode.bot != nil || bot.state != nil {
		t.Fatal("character selection retained the previous character's Lua state")
	}
}

func TestLuaBotMapChangeRespectsChangedScriptPath(t *testing.T) {
	ctx, mode := newBotChatTestContext(t, companionScriptForTest(t))
	bot := mode.bot
	ctx.Config.Script.Path = ""
	next := mode.nextWorldMode()
	next.botMapChanged(ctx)
	if next.bot != nil || bot.state != nil {
		t.Fatal("map transition kept a removed script")
	}
}

func TestCompanionKeepsChatLeaderAcrossMapChange(t *testing.T) {
	ctx, mode := newBotChatTestContext(t, companionScriptForTest(t))
	netClient, server := newBotTestConnection(t, 20080910)
	ctx.Network = netClient
	mode.handleNetworkPacket(ctx, botChatTestPacket(network.PacketZCNotifyChat, 300, "Alice : follow"), time.Now())
	readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Bot", "Following you.", 20080910))
	bot := mode.bot
	next := mode.nextWorldMode()
	ctx.World.Actors = make(map[uint32]worldstate.Actor)
	ctx.World.Player.X, ctx.World.Player.Y = 1, 1
	next.botMapChanged(ctx)
	// Do not walk towards coordinates remembered on the previous map.
	assertNoBotTestPacket(t, server, bot.tick)
	// The leader reappears before its name arrives. The server ID is enough.
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 20, Y: 10, HasObjectType: true, ObjectType: actorObjectTypePC}
	if err := bot.tick(); err != nil {
		t.Fatal(err)
	}
	walk, _ := network.BuildWalkToXYPacketForClientDate(16, 8, 20080910)
	readBotTestPackets(t, server, walk)
}
