package game

import (
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	worldstate "github.com/kivutar/goro/world"
)

func TestWASDGamepadMovementAttackAndLoot(t *testing.T) {
	for _, name := range []string{"stick", "dpad", "keyboard_and_stick", "attack", "loot"} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			conn, server := newBotTestConnection(t, 20080910)
			ctx.Network = conn
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			ctx.World.GAT = flatWalkableGAT(64, 64)
			ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			ctx.World.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501, X: 11, Y: 20}
			ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
			pad := input.GamepadSnapshot{ID: "test"}
			switch name {
			case "stick", "keyboard_and_stick":
				pad.Axes[input.GamepadLeftX], pad.Axes[input.GamepadLeftY] = 0.8, -0.8
			case "dpad":
				pad.Buttons[input.GamepadRight], pad.Buttons[input.GamepadUp] = true, true
			case "attack":
				pad.Buttons[input.GamepadWest] = true
			case "loot":
				pad.Buttons[input.GamepadNorth] = true
			}
			ctx.Input.SetGamepad(pad)
			if name == "keyboard_and_stick" {
				ctx.Input.SetKeyCode(gpucontext.KeyW, true)
				ctx.Input.SetKeyCode(gpucontext.KeyD, true)
			}
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			mode.updateBotInput(ctx, true)
			if err := mode.bot.tick(); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "attack":
				readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
			case "loot":
				readBotTestPackets(t, server, network.BuildItemPickupPacketForClientDate(400, 20080910))
			default:
				want, _ := network.BuildWalkToXYPacketForClientDate(18, 28, 20080910)
				readBotTestPackets(t, server, want)
			}
		})
	}
}

func TestWASDGamepadDeadzoneAndFocusedChat(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	conn, server := newBotTestConnection(t, 20080910)
	ctx.Network = conn
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.GAT = flatWalkableGAT(64, 64)
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadSnapshot{ID: "test"}
	pad.Axes[input.GamepadLeftX], pad.Axes[input.GamepadLeftY] = 0.2, -0.2
	ctx.Input.SetGamepad(pad)
	assertNoBotTestPacket(t, server, func() error { return mode.bot.inputFrame(true) })
	pad.Axes[input.GamepadLeftY] = -1
	pad.Buttons[input.GamepadWest] = true
	ctx.Input.SetGamepad(pad)
	ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
	mode.ui.console.UpdateInput(ctx)
	if !mode.ui.keyboardInputBlocked(ctx) {
		t.Fatal("test chat did not take focus")
	}
	assertNoBotTestPacket(t, server, func() error {
		mode.updateBotInput(ctx, !mode.ui.keyboardInputBlocked(ctx))
		return mode.bot.tick()
	})
}

func TestWASDGamepadDisconnectStopsWalking(t *testing.T) {
	conn, server := newBotTestConnection(t, 20080910)
	state := input.NewState()
	world := worldstate.New()
	world.GAT = flatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	bot, err := newLuaBot(client.Context{Input: state, Network: conn, World: world}, mode, "builtin:wasd")
	if err != nil {
		t.Fatal(err)
	}
	defer bot.close()
	pad := input.GamepadSnapshot{ID: "test"}
	pad.Axes[input.GamepadLeftY] = -1
	state.SetGamepad(pad)
	if err := bot.inputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
	readBotTestPackets(t, server, want)
	state.EndFrame()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 28, FromX: 10, FromY: 20, ToX: 10, ToY: 28, Moving: true, MoveStarted: time.Now(), MoveDuration: 8 * time.Second, MovePath: []worldstate.WalkStep{{X: 10, Y: 20}, {X: 10, Y: 21}, {X: 10, Y: 28}}}
	mode.walkCooldownUntil = time.Time{}
	state.SetGamepad(input.GamepadSnapshot{})
	if err := bot.inputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, _ = network.BuildWalkToXYPacketForClientDate(10, 21, 20080910)
	readBotTestPackets(t, server, want)
}
