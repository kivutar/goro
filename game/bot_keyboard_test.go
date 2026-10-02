package game

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

func loadKeyboardTestBot(t *testing.T, ctx client.Context, mode *WorldMode) {
	t.Helper()
	bot, err := newLuaScript(ctx, mode, ctx.Config.Script.Path)
	if err != nil {
		t.Fatal(err)
	}
	mode.bot = bot
	t.Cleanup(bot.close)
}

func botKeyPressForTest(t *testing.T, bot *luaScript, code gpucontext.Key) {
	t.Helper()
	if err := bot.keyPress(code); err != nil {
		t.Fatal(err)
	}
}

// Match native ordering: snapshot, early handler, default key/text routing.
// Repeats retain consumption without invoking the handler again.
func prepareKeyboardTestText(mode *WorldMode, ctx client.Context, code gpucontext.Key, text string) bool {
	repeated := ctx.Input.KeyCodeDown(code)
	ctx.Input.SetKeyCode(code, true)
	if !repeated {
		mode.HandleKeyPress(ctx, code)
	}
	if !ctx.Input.KeyCodeConsumed(code) {
		mode.PrepareKeyInput(ctx, code, 0)
	}
	ctx.Input.AddTextInput(text)
	if ctx.Input.KeyCodeConsumed(code) {
		return true
	}
	return mode.PrepareTextInput(ctx, code)
}

func TestWASDControlsBeforeChatPreparation(t *testing.T) {
	for _, battle := range []bool{false, true} {
		for _, tc := range []struct {
			name string
			code gpucontext.Key
			text string
		}{
			{"move", gpucontext.KeyW, "z"}, // Physical W on AZERTY.
			{"loot", gpucontext.KeySpace, " "},
			{"fight", gpucontext.KeyF, "f"},
		} {
			t.Run(fmt.Sprintf("battle_%t/%s", battle, tc.name), func(t *testing.T) {
				ctx := chatShortcutTestContext(t)
				conn, server := newBotTestConnection(t, 20080910)
				ctx.Network = conn
				ctx.Session.BattleMode = battle
				ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
				ctx.World = worldstate.New()
				ctx.World.GAT = flatWalkableGAT(64, 64)
				ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
				ctx.World.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501, X: 11, Y: 20}
				ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				if !prepareKeyboardTestText(mode, ctx, tc.code, tc.text) || mode.ui.console.Active() {
					t.Fatal("chat took the keyboard control before Lua input")
				}
				mode.updateBotInput(ctx, !mode.ui.keyboardInputBlocked(ctx))
				if err := mode.bot.tick(); err != nil {
					t.Fatal(err)
				}
				switch tc.name {
				case "move":
					want, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
					readBotTestPackets(t, server, want)
				case "loot":
					readBotTestPackets(t, server, network.BuildItemPickupPacketForClientDate(400, 20080910))
				case "fight":
					readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
				}
				if ctx.Input.KeyCodeJustPressed(tc.code) || !ctx.Input.KeyCodeDown(tc.code) {
					t.Fatal("script did not consume the press while preserving held input")
				}
				ctx.Input.EndFrame()
				if !prepareKeyboardTestText(mode, ctx, tc.code, tc.text) || mode.ui.console.Active() {
					t.Fatal("key repeat opened chat")
				}
			})
		}
	}
}

func TestKeyboardScriptDoesNotStealFocusedInput(t *testing.T) {
	for _, battle := range []bool{false, true} {
		for _, form := range []bool{false, true} {
			t.Run(fmt.Sprintf("battle_%t/form_%t", battle, form), func(t *testing.T) {
				ctx := chatShortcutTestContext(t)
				ctx.Session.BattleMode = battle
				ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				if form {
					mode.ui.settingsWindow.OpenWindow(ctx)
				} else {
					ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
					mode.ui.console.UpdateInput(ctx)
					ctx.Input.EndFrame()
					ctx.Input.SetKeyCode(gpucontext.KeyEnter, false)
				}
				if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "z") {
					t.Fatal("script filtered text intended for the focused UI")
				}
				mode.updateBotInput(ctx, !mode.ui.keyboardInputBlocked(ctx))
				if mode.bot.keyboardAvailable || !ctx.Input.KeyCodeJustPressed(gpucontext.KeyW) {
					t.Fatal("script consumed a focused UI's keyboard input")
				}
				if mode.ui.console.Active() == form {
					t.Fatal("text preparation changed focus between the form and chat")
				}
			})
		}
	}
}

func TestWASDLeavesControlAForClassicChat(t *testing.T) {
	for _, modifier := range []input.KeyCode{gpucontext.KeyLeftControl, gpucontext.KeyRightControl} {
		t.Run(input.KeyCodeName(modifier), func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			ctx.Input.SetKeyCode(modifier, true)
			ctx.Input.SetKeyCode(gpucontext.KeyA, true)
			mode.HandleKeyPress(ctx, gpucontext.KeyA)
			if ctx.Input.KeyCodeConsumed(gpucontext.KeyA) {
				t.Fatal("WASD stole Ctrl+A before chat could restore selection editing")
			}
			mode.PrepareKeyInput(ctx, gpucontext.KeyA, gpucontext.ModControl)
			if !mode.ui.console.Active() {
				t.Fatal("Ctrl+A did not restore classic chat focus")
			}
		})
	}
}

func TestWASDModifiersSuspendHeldControls(t *testing.T) {
	for _, modifier := range []input.KeyCode{
		gpucontext.KeyLeftControl, gpucontext.KeyRightControl,
		gpucontext.KeyLeftAlt, gpucontext.KeyRightAlt,
		gpucontext.KeyLeftSuper, gpucontext.KeyRightSuper,
	} {
		for _, code := range []input.KeyCode{
			gpucontext.KeyW, gpucontext.KeyA, gpucontext.KeyS, gpucontext.KeyD,
			gpucontext.KeyF, gpucontext.KeySpace,
		} {
			t.Run(input.KeyCodeName(modifier)+"/"+input.KeyCodeName(code), func(t *testing.T) {
				ctx := chatShortcutTestContext(t)
				ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
				ctx.World = worldstate.New()
				ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
				ctx.World.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501, X: 11, Y: 20}
				ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				// Observe Lua's action requests without involving the network or
				// simulating the movement/attack acknowledgements from a server.
				if err := mode.bot.state.DoString(`
actions, stops = 0, 0
local function action() actions = actions + 1; return true end
goro.walk, goro.attack, goro.loot = action, action, action
function goro.stop() stops = stops + 1; return true end
`); err != nil {
					t.Fatal(err)
				}
				frame := func(wantActions, wantStops int) {
					t.Helper()
					if err := mode.bot.inputFrame(true); err != nil {
						t.Fatal(err)
					}
					if err := mode.bot.tick(); err != nil {
						t.Fatal(err)
					}
					if got := mode.bot.state.GetGlobal("actions"); got != lua.LNumber(wantActions) {
						t.Fatalf("action requests = %v, want %d", got, wantActions)
					}
					if got := mode.bot.state.GetGlobal("stops"); got != lua.LNumber(wantStops) {
						t.Fatalf("stop requests = %v, want %d", got, wantStops)
					}
					ctx.Input.EndFrame()
				}
				ctx.Input.SetKeyCode(modifier, true)
				ctx.Input.SetKeyCode(code, true)
				mode.HandleKeyPress(ctx, code)
				if ctx.Input.KeyCodeConsumed(code) {
					t.Fatal("WASD consumed a modified shortcut")
				}
				frame(0, 0)
				ctx.Input.SetKeyCode(modifier, false)
				frame(1, 0) // Releasing the modifier restores the held control.
				ctx.Input.SetKeyCode(modifier, true)
				wantStops := 1
				if code == gpucontext.KeyF || code == gpucontext.KeySpace {
					wantStops = 0
				}
				frame(1, wantStops) // Pressing it during movement stops once.
				frame(1, wantStops)
				ctx.Input.SetKeyCode(modifier, false)
				frame(2, wantStops)
			})
		}
	}
}

func TestKeyboardObserverPreservesClassicChat(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	if err := mode.bot.state.DoString(`function keypress(code) seen = code end`); err != nil {
		t.Fatal(err)
	}
	if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "z") || !mode.ui.console.Active() {
		t.Fatal("a keyboard observer changed classic chat activation")
	}
	if got := mode.bot.state.GetGlobal("seen"); got != lua.LString("KeyW") {
		t.Fatalf("observed physical key = %q, want KeyW", got)
	}
	if ctx.Input.TextInput() != "z" {
		t.Fatal("observing a key lost its translated text")
	}
}

func TestWASDUnclaimedKeysKeepNormalChatBehavior(t *testing.T) {
	for _, battle := range []bool{false, true} {
		ctx := chatShortcutTestContext(t)
		ctx.Session.BattleMode = battle
		ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
		mode := NewWorldMode()
		loadKeyboardTestBot(t, ctx, mode)
		if prepareKeyboardTestText(mode, ctx, gpucontext.KeyH, "h") || mode.ui.console.Active() == battle {
			t.Fatalf("unbound H did not keep its normal behavior (Battle Mode %t)", battle)
		}
		if !ctx.Input.KeyCodeJustPressed(gpucontext.KeyH) {
			t.Fatal("an unclaimed key was lost to native shortcuts")
		}
	}
	ctx := chatShortcutTestContext(t)
	ctx.Session.BattleMode = true
	ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	if err := mode.bot.state.DoString(`function keypress(code) end`); err != nil {
		t.Fatal(err)
	}
	prepareKeyboardTestText(mode, ctx, gpucontext.KeySpace, " ")
	if !mode.ui.console.Active() || ctx.Input.KeyCodeConsumed(gpucontext.KeySpace) {
		t.Fatal("observing Space prevented Battle Mode chat activation")
	}
}

func TestKeyboardEnterThenMovementLetterInSameBatch(t *testing.T) {
	for _, battle := range []bool{false, true} {
		ctx := chatShortcutTestContext(t)
		ctx.Session.BattleMode = battle
		ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
		mode := NewWorldMode()
		loadKeyboardTestBot(t, ctx, mode)
		ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
		mode.HandleKeyPress(ctx, gpucontext.KeyEnter)
		mode.PrepareKeyInput(ctx, gpucontext.KeyEnter, 0)
		if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "z") || !mode.ui.console.Active() {
			t.Fatal("a movement binding stole text following Enter in the same batch")
		}
	}
}

func TestKeyboardCallbackErrorsRestoreDefaultHandling(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	if err := mode.bot.state.DoString(`function keypress(code) error("test") end`); err != nil {
		t.Fatal(err)
	}
	if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "z") || !mode.ui.console.Active() || !mode.bot.disabled {
		t.Fatal("failed callback swallowed input or remained active")
	}
}

func TestKeyboardConsumedPressKeepsTranslatedTextForScript(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	if err := mode.bot.state.DoString(`
function keypress(code) goro.keyboard.consume_press(code) end
function input() typed = goro.keyboard.text() end
`); err != nil {
		t.Fatal(err)
	}
	if !prepareKeyboardTestText(mode, ctx, gpucontext.KeyE, "é") || mode.ui.console.Active() {
		t.Fatal("consumed key activated the console")
	}
	mode.updateBotInput(ctx, !mode.ui.keyboardInputBlocked(ctx))
	if got := mode.bot.state.GetGlobal("typed"); got != lua.LString("é") {
		t.Fatalf("script lost raw text for its own editor: %q", got)
	}
}

func TestInactiveKeyboardScriptPreservesClassicChat(t *testing.T) {
	for _, name := range []string{"tick only", "disabled", "closed", "removed", "replaced"} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			switch name {
			case "tick only":
				mode.bot.state.SetGlobal("keypress", lua.LNil)
			case "disabled":
				mode.bot.disabled = true
			case "closed":
				mode.bot.close()
			case "removed":
				ctx.Config.Script.Path = ""
			case "replaced":
				ctx.Config.Script.Path = "another.lua"
			}
			if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "z") || !mode.ui.console.Active() {
				t.Fatal("inactive keyboard script changed classic chat activation")
			}
		})
	}
}
