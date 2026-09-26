package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

func botChatTestPacket(opcode uint16, sender uint32, text string) network.Packet {
	header := 8
	if opcode == network.PacketZCNotifyPlayerChat || opcode == network.PacketZCBroadcast {
		header = 4
	}
	data := make([]byte, header+len(text)+1)
	binary.LittleEndian.PutUint16(data, opcode)
	binary.LittleEndian.PutUint16(data[2:], uint16(len(data)))
	if header == 8 {
		binary.LittleEndian.PutUint32(data[4:], sender)
	}
	copy(data[header:], text)
	return network.Packet{ID: opcode, Data: data}
}

func newBotChatTestContext(t *testing.T, path string) (client.Context, *WorldMode) {
	t.Helper()
	sess := session.New()
	sess.AccountID, sess.CharID = 100, 101
	sess.Selected.Name = "Bot"
	sess.Vitals = session.Vitals{HP: 100, MaxHP: 100, SP: 20, MaxSP: 100}
	sess.Party.Members = []session.PartyMember{{AccountID: 301, Name: "Remote"}}
	world := worldstate.New()
	world.GAT = flatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 100, X: 10, Y: 10}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "Alice", X: 20, Y: 10, HasObjectType: true, ObjectType: actorObjectTypePC}
	world.Actors[400] = worldstate.Actor{ID: 400, Name: "Alice", HasObjectType: true, ObjectType: actorObjectTypeNPC}
	ctx := client.Context{Session: sess, World: world}
	ctx.Config.Script.Path = path
	mode := NewWorldMode()
	bot, err := newLuaBot(ctx, mode, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(bot.close)
	mode.bot = bot
	return ctx, mode
}

func TestLuaChatReceivesPlayerPackets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chat.lua")
	if err := os.WriteFile(path, []byte(`function chat(message) seen = message end`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, headless := range []bool{false, true} {
		t.Run(fmt.Sprintf("headless=%t", headless), func(t *testing.T) {
			ctx, mode := newBotChatTestContext(t, path)
			ctx.Config.Headless = headless
			for _, tc := range []struct {
				opcode                   uint16
				id                       uint32
				raw, channel, name, text string
			}{
				{network.PacketZCNotifyChat, 300, "Alice : follow", "public", "Alice", "follow"},
				{network.PacketZCNotifyChat, 300, "follow", "public", "Alice", "follow"},
				{network.PacketZCNotifyChat, 300, "Alice : hello : there", "public", "Alice", "hello : there"},
				{network.PacketZCNotifyChat, 300, "Someone else : follow", "public", "Alice", "Someone else : follow"},
				{network.PacketZCNotifyChatParty, 301, "Remote : follow", "party", "Remote", "follow"},
			} {
				mode.handleNetworkPacket(ctx, botChatTestPacket(tc.opcode, tc.id, tc.raw), time.Now())
				seen, ok := mode.bot.state.GetGlobal("seen").(*lua.LTable)
				if !ok {
					t.Fatalf("chat callback not called for %q", tc.raw)
				}
				assertLuaNumber(t, seen, "sender_id", float64(tc.id))
				assertLuaString(t, seen, "sender_name", tc.name)
				assertLuaString(t, seen, "channel", tc.channel)
				assertLuaString(t, seen, "text", tc.text)
			}
			// Public chat can arrive before the actor-name response.
			actor := ctx.World.Actors[300]
			actor.Name = ""
			ctx.World.Actors[300] = actor
			mode.handleNetworkPacket(ctx, botChatTestPacket(network.PacketZCNotifyChat, 300, "Éloïse : follow"), time.Now())
			seen := mode.bot.state.GetGlobal("seen").(*lua.LTable)
			assertLuaNumber(t, seen, "sender_id", 300)
			assertLuaString(t, seen, "sender_name", "Éloïse")
			assertLuaString(t, seen, "text", "follow")
		})
	}
}

func TestLuaChatIgnoresNonPlayerMessagesAndSelf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chat.lua")
	if err := os.WriteFile(path, []byte(`function chat(message) error("unexpected chat") end`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, mode := newBotChatTestContext(t, path)
	for _, packet := range []network.Packet{
		botChatTestPacket(network.PacketZCNotifyChat, 400, "Alice : follow"), // NPC with a player's name.
		botChatTestPacket(network.PacketZCNotifyChat, 999, "Alice : follow"), // Unknown speaker.
		botChatTestPacket(network.PacketZCNotifyChat, 0, "Alice : follow"),
		botChatTestPacket(network.PacketZCNotifyChat, 100, "Bot : follow"),
		botChatTestPacket(network.PacketZCNotifyChatParty, 100, "Bot : follow"),
		botChatTestPacket(network.PacketZCNotifyPlayerChat, 0, "Bot : follow"),
		botChatTestPacket(network.PacketZCBroadcast, 0, "Alice : follow"),
	} {
		mode.handleNetworkPacket(ctx, packet, time.Now())
		if mode.bot.disabled {
			t.Fatalf("unexpected chat callback for packet 0x%04x", packet.ID)
		}
	}
	mode.ui.chatRoom.Open(ctx, "Room", 20, true, []string{"Alice"})
	mode.handleNetworkPacket(ctx, botChatTestPacket(network.PacketZCNotifyChat, 300, "Alice : follow"), time.Now())
	if mode.bot.disabled {
		t.Fatal("chat-room messages were treated as public chat")
	}
}

func TestLuaChatCallbackIsOptionalAndErrorsPreserveChat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chat.lua")
	if err := os.WriteFile(path, []byte(`function tick() end`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, mode := newBotChatTestContext(t, path)
	packet := botChatTestPacket(network.PacketZCNotifyChat, 300, "Alice : follow")
	mode.handleNetworkPacket(ctx, packet, time.Now())
	if mode.bot.disabled {
		t.Fatal("missing optional callback disabled the script")
	}
	if err := mode.bot.state.DoString(`function chat(message) error("test") end`); err != nil {
		t.Fatal(err)
	}
	packet = botChatTestPacket(network.PacketZCNotifyChat, 300, "Alice : still displayed")
	mode.handleNetworkPacket(ctx, packet, time.Now())
	if !mode.bot.disabled || mode.bot.state != nil {
		t.Fatal("failed chat callback did not close and disable the script")
	}
	if len(mode.ui.console.Messages()) != 2 {
		t.Fatal("Lua callback changed the normal chat transcript")
	}
	mode.handleNetworkPacket(ctx, packet, time.Now()) // A disabled callback is not retried.
}

func TestCompanionFollowsPublicSpeakerWithoutParty(t *testing.T) {
	ctx, mode := newBotChatTestContext(t, companionScriptForTest(t))
	ctx.Session.Party.Members = nil
	netClient, server := newBotTestConnection(t, 20080910)
	ctx.Network = netClient
	bot := mode.bot
	mode.handleNetworkPacket(ctx, botChatTestPacket(network.PacketZCNotifyChat, 300, "Alice : follow"), time.Now())
	readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Bot", "Following you.", 20080910))
	if err := bot.tick(); err != nil {
		t.Fatal(err)
	}
	walk, ok := network.BuildWalkToXYPacketForClientDate(16, 10, 20080910)
	if !ok {
		t.Fatal("invalid walk destination")
	}
	readBotTestPackets(t, server, walk)
}

func TestCompanionChatHealSendsSkillWithoutParty(t *testing.T) {
	ctx, mode := newBotChatTestContext(t, companionScriptForTest(t))
	ctx.Session.Party.Members = nil
	ctx.Session.Skills.List = []session.Skill{{ID: db.SkillALHeal, Type: skillTargetFriend, Level: 7, Range: 9}}
	actor := ctx.World.Actors[300]
	actor.X = 12
	ctx.World.Actors[300] = actor
	netClient, server := newBotTestConnection(t, 20080910)
	ctx.Network = netClient
	mode.handleNetworkPacket(ctx, botChatTestPacket(network.PacketZCNotifyChat, 300, "Alice : heal"), time.Now())
	if err := mode.bot.tick(); err != nil {
		t.Fatal(err)
	}
	want := network.BuildUseSkillToIDPacketForClientDate(db.SkillALHeal, 1, 300, 20080910)
	want = append(want, network.BuildGlobalChatPacketForClientDate("Bot", "Healing you.", 20080910)...)
	readBotTestPackets(t, server, want)
}

func TestCompanionAutoHealUsesReceivedPartyHP(t *testing.T) {
	ctx, mode := newBotChatTestContext(t, companionScriptForTest(t))
	ctx.Session.Party.Name = "Test party"
	ctx.Session.Party.Members = []session.PartyMember{{AccountID: 300, Name: "Alice"}}
	ctx.Session.Skills.List = []session.Skill{{ID: db.SkillALHeal, Type: skillTargetFriend, Level: 7, Range: 9}}
	actor := ctx.World.Actors[300]
	actor.X = 12
	ctx.World.Actors[300] = actor
	netClient, server := newBotTestConnection(t, 20080910)
	ctx.Network = netClient
	mode.handleNetworkPacket(ctx, botChatTestPacket(network.PacketZCNotifyChat, 300, "Alice : follow"), time.Now())
	readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Bot", "Following you.", 20080910))
	assertNoBotTestPacket(t, server, mode.bot.tick) // Unknown HP is not an injury.
	data := make([]byte, 10)
	binary.LittleEndian.PutUint16(data, network.PacketZCNotifyHPToGroup)
	binary.LittleEndian.PutUint32(data[2:], 300)
	binary.LittleEndian.PutUint16(data[6:], 50)
	binary.LittleEndian.PutUint16(data[8:], 100)
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCNotifyHPToGroup, Data: data}, time.Now())
	if err := mode.bot.tick(); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(db.SkillALHeal, 1, 300, 20080910))
}
