package game

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/bot"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/login"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestChatScriptSwitchSurvivesMapChangeAndStops(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	manager := &Manager{ctx: ctx, mode: mode, script: mode.bot}
	t.Cleanup(manager.Close)
	original := mode.bot
	mode.ui.console.SendText(ctx, "/script wasd")
	manager.syncScript(ctx)
	if mode.bot == nil || mode.bot.Disabled || mode.bot.Path != "builtin:wasd" || original.Active() {
		t.Fatal("chat command did not replace the configured script and close its Lua state")
	}
	t.Cleanup(mode.bot.Close)
	mode.updateBotInput(ctx, true)
	if !mode.bot.KeyboardAvailable {
		t.Fatal("frame input still used the configured script path")
	}
	ctx.Input.SetKeyCode(gpucontext.KeyW, true)
	mode.botKeyPress(ctx, gpucontext.KeyW)
	if !ctx.Input.KeyCodeConsumed(gpucontext.KeyW) {
		t.Fatal("keyboard input still used the configured script path")
	}
	ctx.Input.ResetKeyboard()

	next := mode.nextWorldMode()
	manager.closeScript()
	manager.mode = next
	manager.syncScript(ctx)
	if next.bot == nil || next.bot.Disabled || next.bot.Path != "builtin:wasd" {
		t.Fatal("map change lost the selected script")
	}
	selected := next.bot
	t.Cleanup(selected.Close)
	ctx.Input.SetKeyCode(gpucontext.KeyW, true)
	next.botKeyPress(ctx, gpucontext.KeyW)
	if !ctx.Input.KeyCodeConsumed(gpucontext.KeyW) {
		t.Fatal("selected script did not receive keyboard input")
	}
	ctx.Input.ResetKeyboard()
	next.ui.console.SendText(ctx, "/script none")
	ctx.Input.SetKeyCode(gpucontext.KeyW, true)
	next.botKeyPress(ctx, gpucontext.KeyW)
	if ctx.Input.KeyCodeConsumed(gpucontext.KeyW) {
		t.Fatal("stopped script still intercepted the keyboard")
	}
	manager.syncScript(ctx)
	if next.bot != nil || selected.Active() {
		t.Fatal("stop command did not close the Lua state")
	}
}

func wasdControlsTestContext(t *testing.T) client.Context {
	t.Helper()
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.Session.AccountID = 100
	ctx.World = worldstate.New()
	ctx.World.GAT = testutil.FlatWalkableGAT(64, 64)
	ctx.World.Player = worldstate.Actor{ID: 100, X: 10, Y: 20}
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, HasObjectType: true, ObjectType: scene.ActorObjectTypeMob}
	ctx.World.Actors[301] = worldstate.Actor{ID: 301, X: 11, Y: 21, HasObjectType: true, ObjectType: scene.ActorObjectTypeMob}
	skill := session.Skill{ID: db.SkillACDouble, Type: skills.SkillTargetEnemy, Level: 3, Range: 9}
	heal := session.Skill{ID: db.SkillALHeal, Type: skills.SkillTargetFriend, Level: 3, Range: 9}
	ctx.Session.Skills.List = []session.Skill{skill, heal}
	ctx.Session.Hotkeys = session.Hotkeys{Loaded: true, Version: 1, Slots: []session.HotkeySlot{
		{Type: network.HotkeyTypeSkill, ID: uint32(skill.ID), Level: 3},
		{Type: network.HotkeyTypeSkill, ID: uint32(heal.ID), Level: 3},
	}}
	return ctx
}

// Run a complete press/release through the gameplay callbacks. Keyboard
// shortcuts also reach the normal hotbar handler, to catch double activation.
func wasdControlPress(t *testing.T, ctx client.Context, mode *WorldMode, device, action string) {
	t.Helper()
	key := map[string]gpucontext.Key{
		"next": gpucontext.KeyTab, "skill": gpucontext.KeyF1, "heal": gpucontext.KeyF2,
		"confirm": gpucontext.KeyEnter, "cancel": gpucontext.KeyEscape, "attack": gpucontext.KeyF,
	}[action]
	pad := input.GamepadFrame{}
	if device == "gamepad" {
		pad.ID = "test"
		button := map[string]input.GamepadButton{
			"next": input.GamepadRightShoulder, "skill": input.GamepadSouth, "heal": input.GamepadEast,
			"confirm": input.GamepadSouth, "cancel": input.GamepadEast, "attack": input.GamepadSouth,
		}[action]
		pad.Buttons[button] = true
		if action == "skill" || action == "heal" {
			pad.Axes[input.GamepadRightTrigger] = 1
		}
	} else {
		ctx.Input.SetKeyCode(key, true)
		mode.HandleKeyPress(ctx, key)
		if !ctx.Input.KeyCodeConsumed(key) {
			t.Fatalf("keyboard action %s was not consumed", action)
		}
	}
	ctx.Input.SetGamepad(pad)
	mode.bot.HandleGamepad(ctx, 1.0/60)
	mode.updateBotInput(ctx, true)
	if mode.ui.shortcutBar.UpdateKeyboardInput(ctx, mode, false) {
		t.Fatal("script action also reached the native hotbar")
	}
	if err := mode.bot.Tick(); err != nil {
		t.Fatal(err)
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKeyCode(key, false)
	ctx.Input.SetGamepad(input.GamepadFrame{ID: pad.ID})
	mode.bot.HandleGamepad(ctx, 1.0/60)
	mode.updateBotInput(ctx, true)
	ctx.Input.EndFrame()
	if mode.bot.Disabled {
		t.Fatal("control script failed")
	}
}

func TestWASDKeyboardAndGamepadUseSameTargeting(t *testing.T) {
	for _, selector := range []string{"keyboard", "gamepad"} {
		for _, caster := range []string{"keyboard", "gamepad"} {
			for _, order := range []string{"target_first", "skill_first", "ineligible", "cancel", "repeat_skill", "attack_nearest", "attack_selected"} {
				t.Run(selector+"/"+caster+"/"+order, func(t *testing.T) {
					ctx := wasdControlsTestContext(t)
					conn, server := testutil.NewConnection(t, 20080910)
					ctx.Network = conn
					skill, heal := ctx.Session.Skills.List[0], ctx.Session.Skills.List[1]
					mode := NewWorldMode()
					loadKeyboardTestBot(t, ctx, mode)
					press := func(device, action string) { wasdControlPress(t, ctx, mode, device, action) }
					want := uint32(301)
					switch order {
					case "target_first", "ineligible", "cancel", "attack_selected":
						press(selector, "next")
						press(selector, "next")
						if mode.scriptHighlight.id != 301 {
							t.Fatal("target selection did not persist between presses")
						}
					}
					switch order {
					case "attack_nearest", "attack_selected":
						if order == "attack_nearest" {
							want = 300
						}
						press(caster, "attack")
						readLegacyBotTestActionPacket(t, server, want, network.ActionAttack)
						// A target acquired through attacking is usable by a skill too.
						press(selector, "skill")
					case "skill_first", "repeat_skill":
						assertNoBotTestPacket(t, server, func() error { press(caster, "skill"); return nil })
						if mode.scriptHighlight.id != 300 {
							t.Fatal("skill did not select nearest eligible target")
						}
						if order == "repeat_skill" {
							want = 300
							press(selector, "skill")
						} else {
							press(selector, "next")
							press(caster, "confirm")
						}
					case "ineligible", "cancel":
						if order == "cancel" {
							press(caster, "cancel")
							if mode.scriptHighlight.id != 0 {
								t.Fatal("cancel kept a selected enemy")
							}
							want = 300
							assertNoBotTestPacket(t, server, func() error { press(caster, "skill"); return nil })
						} else {
							skill, want = heal, ctx.Session.AccountID
							assertNoBotTestPacket(t, server, func() error { press(caster, "heal"); return nil })
						}
						if mode.scriptHighlight.id != want {
							t.Fatalf("fallback target = %d, want %d", mode.scriptHighlight.id, want)
						}
						press(selector, "confirm")
					default:
						press(caster, "skill")
					}
					if order == "attack_nearest" || order == "attack_selected" {
						testutil.ReadPackets(t, server, []byte{0x18, 0x01})
					}
					readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 3, want, 20080910))
					assertNoBotTestPacket(t, server, func() error { return nil })
				})
			}
		}
	}
}

func TestWASDArmingSkillPreservesMovementRelease(t *testing.T) {
	for _, device := range []string{"keyboard", "gamepad"} {
		for _, release := range []string{"same_frame", "next_frame"} {
			t.Run(device+"/"+release, func(t *testing.T) {
				ctx := wasdControlsTestContext(t)
				conn, server := testutil.NewConnection(t, 20080910)
				ctx.Network = conn
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				pad := input.GamepadFrame{}
				setMovement := func(down bool) {
					ctx.Input.SetKeyCode(gpucontext.KeyW, down && device == "keyboard")
					pad.Axes[input.GamepadLeftY] = 0
					if device == "gamepad" {
						pad.ID = "test"
						if down {
							pad.Axes[input.GamepadLeftY] = -1
						}
					}
				}
				frame := func() {
					ctx.Input.SetGamepad(pad)
					mode.bot.HandleGamepad(ctx, 1.0/60)
					mode.updateBotInput(ctx, true)
					mode.ui.shortcutBar.UpdateKeyboardInput(ctx, mode, false)
					ctx.Input.EndFrame()
					if mode.bot.Disabled {
						t.Fatal("control script failed")
					}
				}
				setMovement(true)
				frame()
				walk, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
				readBotTestPackets(t, server, walk)
				// The server has accepted the walk, but it has not finished yet.
				ctx.World.Player = worldstate.Actor{ID: 100, X: 10, Y: 28, FromX: 10, FromY: 20, ToX: 10, ToY: 28,
					Moving: true, MoveStarted: time.Now(), MoveDuration: 8 * time.Second}
				mode.walkCooldownUntil = time.Time{}
				mode.walkAwaitingAck = false
				if release == "same_frame" {
					setMovement(false)
				}
				if device == "keyboard" {
					ctx.Input.SetKeyCode(gpucontext.KeyF1, true)
					mode.HandleKeyPress(ctx, gpucontext.KeyF1)
				} else {
					pad.Axes[input.GamepadRightTrigger] = 1
					pad.Buttons[input.GamepadSouth] = true
				}
				frame()
				if mode.skillSystem.Pending.Skill.ID == 0 || mode.skillSystem.Pending.TargetID != 0 {
					t.Fatal("skill was not left armed")
				}
				setMovement(false)
				ctx.Input.SetKeyCode(gpucontext.KeyF1, false)
				pad.Axes[input.GamepadRightTrigger] = 0
				pad.Buttons[input.GamepadSouth] = false
				frame()
				stop, _ := network.BuildWalkToXYPacketForClientDate(10, 21, 20080910)
				readBotTestPackets(t, server, stop)
				assertNoBotTestPacket(t, server, func() error { return nil })
			})
		}
	}
}

func TestWASDInvalidSkillTargetReleasesConfirmation(t *testing.T) {
	for _, device := range []string{"keyboard", "gamepad"} {
		for _, invalidation := range []string{"removed", "dead", "hidden"} {
			t.Run(device+"/"+invalidation, func(t *testing.T) {
				ctx := wasdControlsTestContext(t)
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				wasdControlPress(t, ctx, mode, device, "skill")
				if mode.scriptHighlight.id != 300 {
					t.Fatal("skill did not select the nearest target")
				}
				switch invalidation {
				case "removed":
					mode.removeActorNow(ctx, 300)
				case "dead":
					mode.actorSystem.ActorDeaths = map[uint32]time.Time{300: time.Now()}
				case "hidden":
					actor := ctx.World.Actors[300]
					actor.EffectState = db.EffectStateHide
					ctx.World.Actors[300] = actor
				}
				wasdControlPress(t, ctx, mode, device, "confirm")
				if mode.scriptHighlight.id != 0 || mode.skillSystem.Pending.Skill.ID == 0 {
					t.Fatal("failed confirmation should clear the target and keep the skill armed")
				}
				if device == "keyboard" {
					ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
					mode.HandleKeyPress(ctx, gpucontext.KeyEnter)
					if ctx.Input.KeyCodeConsumed(gpucontext.KeyEnter) {
						t.Fatal("Enter was still consumed for an invalid target")
					}
				} else {
					pad := input.GamepadFrame{ID: "test"}
					pad.Buttons[input.GamepadSouth] = true
					ctx.Input.SetGamepad(pad)
					if capture := mode.bot.HandleGamepad(ctx, 1.0/60); capture.Buttons[input.GamepadSouth] {
						t.Fatal("South did not return to pointer targeting")
					}
				}
			})
		}
	}
}

func TestWASDSkillChaseTakesOverMovement(t *testing.T) {
	for _, device := range []string{"keyboard", "gamepad"} {
		for _, order := range []string{"target_first", "skill_first"} {
			t.Run(device+"/"+order, func(t *testing.T) {
				ctx := wasdControlsTestContext(t)
				conn, server := testutil.NewConnection(t, 20080910)
				ctx.Network = conn
				clear(ctx.World.Actors)
				ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 30, Y: 20, HasObjectType: true, ObjectType: scene.ActorObjectTypeMob}
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				selectAction, castAction := "next", "skill"
				if order == "skill_first" {
					selectAction, castAction = "skill", "confirm"
				}
				wasdControlPress(t, ctx, mode, device, selectAction)
				ctx.Input.SetKeyCode(gpucontext.KeyW, true)
				mode.updateBotInput(ctx, true)
				walk, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
				readBotTestPackets(t, server, walk)
				ctx.World.Player = worldstate.Actor{ID: 100, X: 10, Y: 28, FromX: 10, FromY: 20, ToX: 10, ToY: 28,
					Moving: true, MoveStarted: time.Now(), MoveDuration: 8 * time.Second}
				mode.walkCooldownUntil = time.Time{}
				mode.walkAwaitingAck = false
				ctx.Input.SetKeyCode(gpucontext.KeyW, false)
				wasdControlPress(t, ctx, mode, device, castAction)
				if mode.skillSystem.Pending.TargetID != 300 {
					t.Fatal("skill did not start chasing its target")
				}
				chase, _ := network.BuildWalkToXYPacketForClientDate(20, 20, 20080910)
				readBotTestPackets(t, server, chase)
				// Releasing the old movement must not stop the new skill chase,
				// even before the server acknowledges that new destination.
				mode.walkCooldownUntil = time.Time{}
				mode.walkAwaitingAck = false
				assertNoBotTestPacket(t, server, func() error {
					mode.updateBotInput(ctx, true)
					return nil
				})
			})
		}
	}
}

// Keep movement held across individual frames, including shortcut presses.
func wasdMovementFrame(t *testing.T, ctx client.Context, mode *WorldMode, device string, move, shortcut bool) {
	t.Helper()
	pad := input.GamepadFrame{}
	if device == "gamepad" {
		pad.ID = "test"
		if move {
			pad.Axes[input.GamepadLeftY] = -1
		}
		if shortcut {
			pad.Axes[input.GamepadRightTrigger] = 1
			pad.Buttons[input.GamepadSouth] = true
		}
	} else {
		ctx.Input.SetKeyCode(gpucontext.KeyW, move)
		pressed := shortcut && !ctx.Input.KeyCodeDown(gpucontext.KeyF1)
		ctx.Input.SetKeyCode(gpucontext.KeyF1, shortcut)
		if pressed {
			mode.HandleKeyPress(ctx, gpucontext.KeyF1)
		}
	}
	ctx.Input.SetGamepad(pad)
	mode.bot.HandleGamepad(ctx, 1.0/60)
	mode.updateBotInput(ctx, true)
	if mode.ui.shortcutBar.UpdateKeyboardInput(ctx, mode, false) {
		t.Fatal("shortcut also reached the native hotbar")
	}
	ctx.Input.EndFrame()
	if mode.bot.Disabled {
		t.Fatal("control script failed")
	}
}

func TestWASDSkillsResumeHeldMovementAfterServerStop(t *testing.T) {
	for _, device := range []string{"keyboard", "gamepad"} {
		for _, skill := range []session.Skill{
			{ID: db.SkillALAngelus, Type: skills.SkillTargetSelf, Level: 3},
			{ID: db.SkillALHeal, Type: skills.SkillTargetFriend, Level: 3, Range: 9},
			{ID: db.SkillMGFirewall, Type: skills.SkillTargetPlace, Level: 3, Range: 9},
		} {
			t.Run(device+"/"+skills.SkillLabel(skill), func(t *testing.T) {
				ctx := wasdControlsTestContext(t)
				conn, server := testutil.NewConnection(t, 20080910)
				ctx.Network = conn
				ctx.Session.Skills.List = []session.Skill{skill}
				ctx.Session.Hotkeys.Slots[0] = session.HotkeySlot{Type: network.HotkeyTypeSkill, ID: uint32(skill.ID), Level: 3}
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				frame := func(shortcut bool) { wasdMovementFrame(t, ctx, mode, device, true, shortcut) }
				frame(false)
				walk, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
				readBotTestPackets(t, server, walk)
				ctx.World.Player = worldstate.Actor{ID: 100, X: 10, Y: 28, FromX: 10, FromY: 20, ToX: 10, ToY: 28,
					Moving: true, MoveStarted: time.Now(), MoveDuration: 8 * time.Second}
				frame(true)
				if !skills.IsSelfTargetSkill(skill) {
					frame(false)
					if skills.IsGroundTargetSkill(skill) {
						if err := mode.skills().UseGround(ctx, skill, 12, 20, "", "target"); err != nil {
							t.Fatal(err)
						}
					} else {
						frame(true) // Confirm the already highlighted self target.
					}
				}
				if skills.IsGroundTargetSkill(skill) {
					readBotTestPackets(t, server, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 3, 12, 20, 20080910))
				} else {
					readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 3, 100, 20080910))
				}
				// The reply can arrive after another input frame; keep the old
				// walk until the server actually interrupts it for the cast.
				mode.walkCooldownUntil = time.Time{}
				mode.walkAwaitingAck = false
				assertNoBotTestPacket(t, server, func() error { frame(false); return nil })
				mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 100, TargetID: 100, SkillID: skill.ID, DelayTime: 1})
				if ctx.World.Player.Moving {
					t.Fatal("cast did not stop the previous walk")
				}
				frame(false)
				readBotTestPackets(t, server, walk)
				// While awaiting the new walk acknowledgement, the normal
				// movement cooldown still prevents duplicate requests.
				assertNoBotTestPacket(t, server, func() error { frame(false); return nil })
			})
		}
	}
}

func TestWASDGroundSkillChaseSurvivesMovementRelease(t *testing.T) {
	for _, device := range []string{"keyboard", "gamepad"} {
		t.Run(device, func(t *testing.T) {
			ctx := wasdControlsTestContext(t)
			conn, server := testutil.NewConnection(t, 20080910)
			ctx.Network = conn
			skill := session.Skill{ID: db.SkillMGFirewall, Type: skills.SkillTargetPlace, Level: 3, Range: 9}
			ctx.Session.Skills.List = []session.Skill{skill}
			ctx.Session.Hotkeys.Slots[0] = session.HotkeySlot{Type: network.HotkeyTypeSkill, ID: uint32(skill.ID), Level: 3}
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			frame := func(move, shortcut bool) { wasdMovementFrame(t, ctx, mode, device, move, shortcut) }
			frame(true, false)
			walk, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
			readBotTestPackets(t, server, walk)
			ctx.World.Player = worldstate.Actor{ID: 100, X: 10, Y: 28, FromX: 10, FromY: 20, ToX: 10, ToY: 28,
				Moving: true, MoveStarted: time.Now(), MoveDuration: 8 * time.Second}
			frame(true, true)
			frame(true, false)
			projection := testutil.Projection(ctx.ScreenW, ctx.ScreenH, scene.CellCenter(10), scene.CellCenter(20), 0)
			point := projection.Project(scene.CellCenter(30), scene.CellCenter(20), 0)
			ctx.Input.SetMousePosition(int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
			if device == "gamepad" {
				pad := input.GamepadFrame{ID: "test"}
				pad.Axes[input.GamepadLeftY] = -1
				pad.Buttons[input.GamepadSouth] = true
				ctx.Input.SetGamepad(pad)
				if capture := mode.bot.HandleGamepad(ctx, 1.0/60); capture.Buttons[input.GamepadSouth] {
					t.Fatal("ground targeting did not leave South to the pointer")
				}
			}
			mode.updateBotInput(ctx, true)
			mode.skills().HandleClick(ctx, projection, time.Now())
			ctx.Input.EndFrame()
			if mode.queuedWalk == nil {
				t.Fatal("skill chase was not queued behind the previous walk")
			}
			assertNoBotTestPacket(t, server, func() error { frame(false, false); return nil })
			if !mode.skillSystem.Pending.Ground {
				t.Fatal("releasing movement canceled the queued skill chase")
			}
			mode.walkAwaitingAck = false
			mode.walkCooldownUntil = time.Time{}
			mode.processQueuedWalk(ctx, time.Now())
			chase, _ := network.BuildWalkToXYPacketForClientDate(20, 20, 20080910)
			readBotTestPackets(t, server, chase)
			applySelfMoveAck(ctx, network.SelfMoveAck{FromX: 10, FromY: 20, ToX: 20, ToY: 20})
			mode.walkCooldownUntil = time.Time{}
			mode.walkAwaitingAck = false
			assertNoBotTestPacket(t, server, func() error { frame(false, false); return nil })
			if !mode.skillSystem.Pending.Ground {
				t.Fatal("ground skill chase was canceled")
			}
			ctx.World.Player.MoveStarted = time.Now().Add(-ctx.World.Player.MoveDuration - time.Millisecond)
			mode.skills().UpdatePendingTarget(ctx, "test", false)
			mode.skillSystem.Pending.ReadyAt = time.Now().Add(-time.Millisecond)
			mode.skills().ProcessPendingTarget(ctx)
			readBotTestPackets(t, server, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 3, 30, 20, 20080910))
		})
	}
}

func TestWASDGamepadMovementAttackAndLoot(t *testing.T) {
	for _, name := range []string{"stick", "dpad", "keyboard_and_stick", "attack", "nearest_attack", "loot"} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			conn, server := testutil.NewConnection(t, 20080910)
			ctx.Network = conn
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			ctx.World.GAT = testutil.FlatWalkableGAT(64, 64)
			ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			ctx.World.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501, X: 11, Y: 20}
			ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
			pad := input.GamepadFrame{ID: "test"}
			switch name {
			case "stick", "keyboard_and_stick":
				pad.Axes[input.GamepadLeftX], pad.Axes[input.GamepadLeftY] = 0.8, -0.8
			case "dpad":
				pad.Buttons[input.GamepadRight], pad.Buttons[input.GamepadUp] = true, true
			case "attack":
				pad.Buttons[input.GamepadSouth] = true
				pad.Buttons[input.GamepadRightShoulder] = true
			case "nearest_attack":
				pad.Buttons[input.GamepadSouth] = true
			case "loot":
				pad.Buttons[input.GamepadWest] = true
			}
			ctx.Input.SetGamepad(pad)
			if name == "keyboard_and_stick" {
				ctx.Input.SetKeyCode(gpucontext.KeyW, true)
				ctx.Input.SetKeyCode(gpucontext.KeyD, true)
			}
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			mode.bot.HandleGamepad(ctx, 1.0/60)
			mode.updateBotInput(ctx, true)
			if err := mode.bot.Tick(); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "attack", "nearest_attack":
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

func TestWASDGamepadSkillChordUsesSelectedEnemyWithoutAttackOrLoot(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	conn, server := testutil.NewConnection(t, 20080910)
	ctx.Network = conn
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.Session.AccountID = ctx.World.Player.ID
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	skill := session.Skill{ID: db.SkillACDouble, Type: skills.SkillTargetEnemy, Level: 3, Range: 9}
	selfSkill := session.Skill{ID: db.SkillALAngelus, Type: skills.SkillTargetSelf, Level: 3}
	ctx.Session.Skills.List = []session.Skill{skill, selfSkill}
	ctx.Session.Hotkeys = session.Hotkeys{Loaded: true, Version: 1, Slots: []session.HotkeySlot{
		{Type: network.HotkeyTypeSkill, ID: uint32(skill.ID), Level: 3},
		{}, {}, {Type: network.HotkeyTypeSkill, ID: uint32(selfSkill.ID), Level: 3},
	}}
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadFrame{ID: "test"}
	pad.Buttons[input.GamepadRightShoulder] = true
	pad.Axes[input.GamepadRightTrigger] = 1 // Keep the skill modifier held while selecting.
	ctx.Input.SetGamepad(pad)
	assertNoBotTestPacket(t, server, func() error {
		mode.bot.HandleGamepad(ctx, 1.0/60)
		return nil
	})
	if mode.scriptHighlight.id != 300 {
		t.Fatal("shoulder did not select enemy")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadRightShoulder] = false
	pad.Axes[input.GamepadRightTrigger] = 1
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	capture := mode.bot.HandleGamepad(ctx, 1.0/60)
	for _, button := range []input.GamepadButton{input.GamepadSouth, input.GamepadEast, input.GamepadWest, input.GamepadNorth} {
		if !capture.Buttons[button] {
			t.Fatalf("skill modifier failed to capture button %d", button)
		}
	}
	if mode.bot.Disabled {
		t.Fatal("controller callback failed")
	}
	readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 3, 300, 20080910))
	ctx.Input.EndFrame()
	pad.Axes[input.GamepadRightTrigger] = 0
	ctx.Input.SetGamepad(pad)
	assertNoBotTestPacket(t, server, func() error {
		capture = mode.bot.HandleGamepad(ctx, 1.0/60)
		mode.updateBotInput(ctx, true)
		return mode.bot.Tick()
	})
	if !capture.Buttons[input.GamepadSouth] {
		t.Fatal("releasing R2 let a held skill button become a click")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadSouth] = false
	pad.Buttons[input.GamepadEast] = true
	pad.Axes[input.GamepadRightTrigger] = 1
	ctx.Input.SetGamepad(pad)
	mode.skillSystem.Pending = skills.PendingTarget{Skill: skill}
	assertNoBotTestPacket(t, server, func() error {
		mode.bot.HandleGamepad(ctx, 1.0/60)
		return nil
	})
	if mode.skillSystem.Pending.Skill.ID != skill.ID {
		t.Fatal("empty hotbar slot changed the armed skill")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadEast] = false
	pad.Buttons[input.GamepadNorth] = true
	ctx.Input.SetGamepad(pad)
	mode.bot.HandleGamepad(ctx, 1.0/60)
	readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(selfSkill.ID, 3, ctx.Session.AccountID, 20080910))
	assertNoBotTestPacket(t, server, func() error { return nil })
	if mode.skillSystem.Pending.Skill.ID != skill.ID {
		t.Fatal("self skill also cast the previously armed targeted skill")
	}
}

func TestWASDGamepadDpadSkillChordsDoNotBecomeMovement(t *testing.T) {
	for _, tc := range []struct {
		name       string
		button     input.GamepadButton
		slot, x, y int
	}{
		{"up", input.GamepadUp, 5, 10, 28},
		{"right", input.GamepadRight, 6, 18, 20},
		{"down", input.GamepadDown, 7, 10, 12},
		{"left", input.GamepadLeft, 8, 2, 20},
	} {
		for _, contents := range []string{"skill", "empty"} {
			t.Run(tc.name+"/"+contents, func(t *testing.T) {
				ctx := wasdControlsTestContext(t)
				conn, server := testutil.NewConnection(t, 20080910)
				ctx.Network = conn
				skill := ctx.Session.Skills.List[0]
				ctx.Session.Hotkeys.Slots = make([]session.HotkeySlot, 9)
				if contents == "skill" {
					ctx.Session.Hotkeys.Slots[tc.slot-1] = session.HotkeySlot{Type: network.HotkeyTypeSkill, ID: uint32(skill.ID), Level: 3}
				}
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				wasdControlPress(t, ctx, mode, "gamepad", "next")
				pad := input.GamepadFrame{ID: "test"}
				frame := func() {
					ctx.Input.SetGamepad(pad)
					mode.bot.HandleGamepad(ctx, 1.0/60)
					mode.updateBotInput(ctx, true)
					if err := mode.bot.Tick(); err != nil {
						t.Fatal(err)
					}
					ctx.Input.EndFrame()
					if mode.bot.Disabled {
						t.Fatal("control script failed")
					}
				}
				pad.Axes[input.GamepadRightTrigger] = 1
				pad.Buttons[tc.button] = true
				frame()
				if contents == "skill" {
					readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 3, 300, 20080910))
				}
				// Holding a chord must neither repeat the skill nor start walking,
				// including when R2 is released before the direction.
				for _, trigger := range []float64{1, 0} {
					pad.Axes[input.GamepadRightTrigger] = trigger
					assertNoBotTestPacket(t, server, func() error { frame(); return nil })
				}
				pad.Buttons[tc.button] = false
				frame()
				pad.Buttons[tc.button] = true
				frame()
				walk, _ := network.BuildWalkToXYPacketForClientDate(tc.x, tc.y, 20080910)
				readBotTestPackets(t, server, walk)
			})
		}
	}
}

func TestWASDKeyboardSkillTargetSurvivesWithoutGamepad(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	mode := NewWorldMode()
	mode.skillSystem.Pending = skills.PendingTarget{Skill: session.Skill{ID: db.SkillACDouble, Type: skills.SkillTargetEnemy, Level: 3, Range: 9}}
	loadKeyboardTestBot(t, ctx, mode)
	ctx.Input.SetKeyCode(gpucontext.KeyTab, true)
	botKeyPressForTest(t, mode.bot, gpucontext.KeyTab)
	mode.bot.HandleGamepad(ctx, 1.0/60)
	if mode.scriptHighlight.id != 300 {
		t.Fatal("idle controller callback cleared keyboard skill selection")
	}
}

func TestWASDGamepadHealTargetsRespectNoShift(t *testing.T) {
	for _, override := range []string{"none", "shift", "noshift"} {
		t.Run(override, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			conn, server := testutil.NewConnection(t, 20080910)
			ctx.Network = conn
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			ctx.Session.AccountID, ctx.Session.CharID = 100, 200
			ctx.World.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
			ctx.World.Actors[100] = ctx.World.Player // Local actor aliases must not duplicate self.
			ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
			ctx.World.Actors[301] = worldstate.Actor{ID: 301, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
			ctx.World.Actors[302] = worldstate.Actor{ID: 302, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true, EffectState: db.EffectStateHide}
			ctx.World.Actors[303] = worldstate.Actor{ID: 303, X: 11, Y: 20, Job: scene.ActorJobHiddenWarpNPC}
			skill := session.Skill{ID: db.SkillALHeal, Type: skills.SkillTargetFriend, Level: 3, Range: 9}
			ctx.Session.Skills.List = []session.Skill{skill}
			ctx.Session.Hotkeys = session.Hotkeys{Loaded: true, Version: 1, Slots: []session.HotkeySlot{
				{Type: network.HotkeyTypeSkill, ID: uint32(skill.ID), Level: 3},
			}}
			mode := NewWorldMode()
			mode.actorSystem.ActorDeaths = map[uint32]time.Time{301: time.Now()}
			loadKeyboardTestBot(t, ctx, mode)
			pad := input.GamepadFrame{ID: "test"}
			pad.Axes[input.GamepadRightTrigger] = 1
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			mode.bot.HandleGamepad(ctx, 1.0/60)
			if mode.scriptHighlight.id != ctx.Session.AccountID {
				t.Fatalf("Heal initially selected %d, want self", mode.scriptHighlight.id)
			}
			ctx.Input.EndFrame()
			pad.Axes[input.GamepadRightTrigger] = 0
			pad.Buttons[input.GamepadSouth] = false
			pad.Buttons[input.GamepadRightShoulder] = true
			ctx.Input.SetGamepad(pad)
			// Changing the override after arming the skill must affect cycling.
			ctx.Session.NoShift = override == "noshift"
			shift, _ := input.KeyCodeFromName("ShiftLeft")
			ctx.Input.SetKeyCode(shift, override == "shift")
			mode.bot.HandleGamepad(ctx, 1.0/60)
			want := ctx.Session.AccountID
			if override == "noshift" {
				want = 300
			}
			if mode.bot.Disabled || mode.scriptHighlight.id != want {
				t.Fatalf("Heal target = %d, want %d", mode.scriptHighlight.id, want)
			}
			ctx.Input.EndFrame()
			pad.Buttons[input.GamepadRightShoulder] = false
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			mode.bot.HandleGamepad(ctx, 1.0/60)
			readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 3, want, 20080910))
		})
	}
}

func TestWASDGamepadTargetCyclingKeepsEnemyAndSkillSelectionsSeparate(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.Session.AccountID = ctx.World.Player.ID
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	ctx.World.Actors[301] = worldstate.Actor{ID: 301, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	ctx.World.Actors[302] = worldstate.Actor{ID: 302, X: 10, Y: 21, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	ctx.World.Actors[303] = worldstate.Actor{ID: 303, X: 10, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	mode := NewWorldMode()
	mode.actorSystem.ActorDeaths = map[uint32]time.Time{303: time.Now()}
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadFrame{ID: "test"}
	press := func(button input.GamepadButton, want uint32) {
		t.Helper()
		pad.Buttons[button] = true
		ctx.Input.SetGamepad(pad)
		mode.bot.HandleGamepad(ctx, 1.0/60)
		if mode.bot.Disabled || mode.scriptHighlight.id != want {
			t.Fatalf("target = %d, want %d (script disabled: %v)", mode.scriptHighlight.id, want, mode.bot.Disabled)
		}
		ctx.Input.EndFrame()
		pad.Buttons[button] = false
		ctx.Input.SetGamepad(pad)
		mode.bot.HandleGamepad(ctx, 1.0/60)
		ctx.Input.EndFrame()
	}
	// Nearest first, ID breaks distance ties; both directions wrap.
	for _, want := range []uint32{301, 302, 300, 301} {
		press(input.GamepadRightShoulder, want)
	}
	press(input.GamepadLeftShoulder, 300)

	mode.skillSystem.Pending = skills.PendingTarget{Skill: session.Skill{ID: db.SkillALHeal, Type: skills.SkillTargetFriend, Level: 1}}
	press(input.GamepadRightShoulder, ctx.Session.AccountID)
	mode.skillSystem.Pending = skills.PendingTarget{}
	mode.bot.HandleGamepad(ctx, 1.0/60)
	if mode.scriptHighlight.id != 300 {
		t.Fatal("ending skill targeting lost the selected enemy")
	}
	press(input.GamepadLeftShoulder, 302)
	delete(ctx.World.Actors, 302)
	press(input.GamepadRightShoulder, 301)
	clear(ctx.World.Actors)
	press(input.GamepadRightShoulder, 0)
	press(input.GamepadLeftShoulder, 0)
}

type gamepadHoverTestUI struct {
	client.UIManager
	blocked bool
}

func (u *gamepadHoverTestUI) PointerBlocked(int, int) bool { return u.blocked }

func TestWASDGamepadUIClickDoesNotBecomeAnAttackWhenPointerLeavesWindow(t *testing.T) {
	for _, name := range []string{"selected", "nearest"} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			conn, server := testutil.NewConnection(t, 20080910)
			ctx.Network = conn
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
			ui := &gamepadHoverTestUI{blocked: true}
			ctx.UIManager = ui
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			pad := input.GamepadFrame{ID: "test"}
			pad.Buttons[input.GamepadRightShoulder] = name == "selected"
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			capture := mode.bot.HandleGamepad(ctx, 1.0/60)
			if capture.Buttons[input.GamepadSouth] {
				t.Fatal("press over UI was not left to the pointer")
			}
			// Pointer motion follows the early callback; the held press still belongs
			// to the UI, both in this frame and after dragging outside the window.
			ui.blocked = false
			for i := 0; i < 2; i++ {
				assertNoBotTestPacket(t, server, func() error {
					mode.updateBotInput(ctx, true)
					return mode.bot.Tick()
				})
				ctx.Input.EndFrame()
				pad.Buttons[input.GamepadRightShoulder] = false
				ctx.Input.SetGamepad(pad)
				mode.bot.HandleGamepad(ctx, 1.0/60)
			}
		})
	}
}

func TestWASDGamepadCameraMovementAndDialogFocus(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadFrame{ID: "test"}
	pad.Axes[input.GamepadLeftTrigger] = 1
	pad.Axes[input.GamepadRightX] = 1
	pad.Axes[input.GamepadRightY] = 1
	initialZoom, initialPitch := mode.camera.targetZoom(), mode.camera.currentPitch()
	ctx.Input.SetGamepad(pad)
	capture := mode.bot.HandleGamepad(ctx, 0.02)
	if !capture.Pointer || mode.camera.yawOffset != 2 {
		t.Fatalf("camera modifier did not claim stick and rotate: %+v yaw=%v", capture, mode.camera.yawOffset)
	}
	if mode.camera.targetZoom() != initialZoom || mode.camera.currentPitch() <= initialPitch {
		t.Fatal("L2 + stick down did not tilt without zooming")
	}
	pitch := mode.camera.currentPitch()
	pad.Axes[input.GamepadRightTrigger] = 1
	for _, leftTrigger := range []float64{0, 1} {
		ctx.Input.EndFrame()
		pad.Axes[input.GamepadLeftTrigger] = leftTrigger
		ctx.Input.SetGamepad(pad)
		previousZoom := mode.camera.targetZoom()
		capture = mode.bot.HandleGamepad(ctx, 0.02)
		if !capture.Pointer || mode.camera.targetZoom() <= previousZoom || mode.camera.currentPitch() != pitch || mode.camera.yawOffset != 2 {
			t.Fatal("R2 + stick down did not exclusively zoom out, including with both triggers held")
		}
	}
	zoom := mode.camera.targetZoom()
	ctx.Input.EndFrame()
	mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
	mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	capture = mode.bot.HandleGamepad(ctx, 0.02)
	if mode.ui.npc.Dialog.IsOpen() || !capture.Buttons[input.GamepadSouth] || mode.camera.yawOffset != 2 || mode.camera.targetZoom() != zoom {
		t.Fatal("NPC confirmation did not take priority over camera/gameplay")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadSouth] = false
	pad.Axes[input.GamepadRightY] = -1
	ctx.Input.SetGamepad(pad)
	mode.bot.HandleGamepad(ctx, 0.02)
	if mode.camera.targetZoom() >= zoom {
		t.Fatal("R2 + stick up did not zoom in")
	}
	if mode.bot.Disabled {
		t.Fatal("controller callback failed")
	}
}

func TestWASDGamepadNPCDialogYieldsToDisconnect(t *testing.T) {
	for name, button := range map[string]input.GamepadButton{"south": input.GamepadSouth, "east": input.GamepadEast} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
			mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
			login.OpenDisconnectDialog(ctx, &mode.ui.disconnectDialog, "Disconnected")
			ctx.Input.EndFrame()
			pad := input.GamepadFrame{ID: "test"}
			pad.Buttons[button] = true
			ctx.Input.SetGamepad(pad)
			capture := mode.bot.HandleGamepad(ctx, 1.0/60)
			if capture.Buttons[input.GamepadSouth] || capture.Buttons[input.GamepadEast] || capture.Pointer {
				t.Fatal("NPC captured pointer controls belonging to the disconnect alert")
			}
			if !mode.ui.npc.Dialog.IsOpen() || !mode.ui.disconnectDialog.IsOpen() {
				t.Fatal("controller acted on a dialog instead of leaving input to the pointer")
			}

			// Removing the alert restores normal NPC confirmation/cancellation.
			mode.ui.disconnectDialog.Close(ctx)
			ctx.Input.EndFrame()
			pad.Buttons[button] = false
			ctx.Input.SetGamepad(pad)
			mode.bot.HandleGamepad(ctx, 1.0/60)
			ctx.Input.EndFrame()
			pad.Buttons[button] = true
			ctx.Input.SetGamepad(pad)
			capture = mode.bot.HandleGamepad(ctx, 1.0/60)
			if !capture.Buttons[button] || mode.ui.npc.Dialog.IsOpen() {
				t.Fatal("NPC controls did not resume after the alert closed")
			}
			if mode.bot.Disabled {
				t.Fatal("controller callback failed")
			}
		})
	}
}

func TestNPCDialogModalPriorityForKeyboardAndGamepad(t *testing.T) {
	for _, name := range []string{"friend", "trade", "party", "pet", "homunculus", "mercenary"} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Network, _ = testutil.NewConnection(t, 20080910)
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
			mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
			mode.ui.npc.Dialog.Update(ctx) // Publish before testing keyboard dispatch.
			modal := map[string]*gameui.ConfirmModal{
				"friend": &mode.ui.social.FriendRequest, "trade": &mode.ui.social.TradeRequest,
				"party": &mode.ui.social.PartyInfo, "pet": &mode.ui.pet.Confirm,
				"homunculus": &mode.ui.companions.HomunculusConfirm, "mercenary": &mode.ui.companions.MercenaryConfirm,
			}[name]
			modal.OpenAlert(ctx, name, "Confirm", nil)
			ctx.Input.EndFrame()
			pad := input.GamepadFrame{ID: "test"}
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			capture := mode.bot.HandleGamepad(ctx, 1.0/60)
			if capture.Buttons[input.GamepadSouth] || capture.Buttons[input.GamepadEast] || !mode.ui.npc.Dialog.IsOpen() {
				t.Fatal("NPC stole controller input from the modal")
			}
			ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if modal.IsOpen() || !mode.ui.npc.Dialog.IsOpen() {
				t.Fatal("Enter did not confirm only the higher-priority modal")
			}
			// An ordinary window must not become a blanket blocker for NPCs.
			mode.ui.settingsWindow.OpenWindow(ctx)
			ctx.Input.EndFrame()
			ctx.Input.SetKeyCode(gpucontext.KeyEnter, false)
			pad.Buttons[input.GamepadSouth] = false
			ctx.Input.SetGamepad(pad)
			mode.bot.HandleGamepad(ctx, 1.0/60)
			ctx.Input.EndFrame()
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			capture = mode.bot.HandleGamepad(ctx, 1.0/60)
			if !capture.Buttons[input.GamepadSouth] || mode.ui.npc.Dialog.IsOpen() || mode.bot.Disabled {
				t.Fatal("NPC controller input did not resume after confirming the modal")
			}
		})
	}
}

func TestNPCDialogKeyboardYieldsToGuildPrompt(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Network = network.NewClient(20080910, false)
	t.Cleanup(func() { ctx.Network.Close() })
	ctx.World = worldstate.New()
	mode := NewWorldMode()
	mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
	mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
	mode.ui.npc.Dialog.Update(ctx)
	// This modal's Update runs after NPC dispatch, so ordering alone cannot
	// give it priority. The shared eligibility check must protect it too.
	mode.ui.social.GuildMemberPrompt.Open(ctx, "Guild", "Reason", "", 40)
	ctx.Input.EndFrame()
	ctx.Input.SetKeyCode(gpucontext.KeyEscape, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.social.GuildMemberPrompt.IsOpen() || !mode.ui.npc.Dialog.IsOpen() {
		t.Fatal("Escape did not cancel only the guild prompt above the NPC")
	}
}

func TestWASDGamepadMovementFollowsRotatedCamera(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	conn, server := testutil.NewConnection(t, 20080910)
	ctx.Network = conn
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.GAT = testutil.FlatWalkableGAT(64, 64)
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 20, Y: 20}
	mode := NewWorldMode()
	mode.camera.Rotate(90)
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadFrame{ID: "test"}
	pad.Axes[input.GamepadLeftY] = -1
	ctx.Input.SetGamepad(pad)
	mode.updateBotInput(ctx, true)
	want, _ := network.BuildWalkToXYPacketForClientDate(12, 20, 20080910)
	readBotTestPackets(t, server, want)
}

func TestWASDGamepadDeadzoneAndFocusedChat(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	conn, server := testutil.NewConnection(t, 20080910)
	ctx.Network = conn
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.GAT = testutil.FlatWalkableGAT(64, 64)
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadFrame{ID: "test"}
	pad.Axes[input.GamepadLeftX], pad.Axes[input.GamepadLeftY] = 0.2, -0.2
	ctx.Input.SetGamepad(pad)
	assertNoBotTestPacket(t, server, func() error { return mode.bot.InputFrame(true) })
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
		return mode.bot.Tick()
	})
}

func TestWASDGamepadDisconnectStopsWalking(t *testing.T) {
	conn, server := testutil.NewConnection(t, 20080910)
	state := input.NewState()
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	bot, err := newLuaScript(client.Context{Input: state, Network: conn, World: world}, mode, "builtin:wasd")
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	pad := input.GamepadFrame{ID: "test"}
	pad.Axes[input.GamepadLeftY] = -1
	state.SetGamepad(pad)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
	readBotTestPackets(t, server, want)
	state.EndFrame()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 28, FromX: 10, FromY: 20, ToX: 10, ToY: 28, Moving: true, MoveStarted: time.Now(), MoveDuration: 8 * time.Second, MovePath: []worldstate.WalkStep{{X: 10, Y: 20}, {X: 10, Y: 21}, {X: 10, Y: 28}}}
	mode.walkCooldownUntil = time.Time{}
	mode.walkAwaitingAck = false
	state.SetGamepad(input.GamepadFrame{})
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, _ = network.BuildWalkToXYPacketForClientDate(10, 21, 20080910)
	readBotTestPackets(t, server, want)
}

func loadKeyboardTestBot(t *testing.T, ctx client.Context, mode *WorldMode) {
	t.Helper()
	bot, err := newLuaScript(ctx, mode, ctx.Config.Script.Path)
	if err != nil {
		t.Fatal(err)
	}
	mode.bot = bot
	t.Cleanup(bot.Close)
}

func botKeyPressForTest(t *testing.T, bot *bot.Script, code gpucontext.Key) {
	t.Helper()
	if err := bot.KeyPress(code); err != nil {
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
				conn, server := testutil.NewConnection(t, 20080910)
				ctx.Network = conn
				ctx.Session.BattleMode = battle
				ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
				ctx.World = worldstate.New()
				ctx.World.GAT = testutil.FlatWalkableGAT(64, 64)
				ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
				ctx.World.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501, X: 11, Y: 20}
				ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				if !prepareKeyboardTestText(mode, ctx, tc.code, tc.text) || mode.ui.console.Active() {
					t.Fatal("chat took the keyboard control before Lua input")
				}
				mode.updateBotInput(ctx, !mode.ui.keyboardInputBlocked(ctx))
				if err := mode.bot.Tick(); err != nil {
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
				if mode.bot.KeyboardAvailable || !ctx.Input.KeyCodeJustPressed(gpucontext.KeyW) {
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
				ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
				wantStops := 1
				if code == gpucontext.KeyF || code == gpucontext.KeySpace {
					wantStops = 0
				}
				wasd, err := os.ReadFile(ctx.Config.Script.Path)
				if err != nil {
					t.Fatal(err)
				}
				ctx.Config.Script.Path = scriptTestFile(t, string(wasd)+fmt.Sprintf(`
local actions, stops, frame = 0, 0, 0
local function action() actions = actions + 1; return true end
goro.walk, goro.attack, goro.loot = action, action, action
function goro.stop() stops = stops + 1; return true end
local expected = {{0, 0}, {1, 0}, {1, %[1]d}, {1, %[1]d}, {2, %[1]d}}
local wasd_tick = tick
function tick()
	wasd_tick()
	frame = frame + 1
	assert(actions == expected[frame][1], "action requests")
	assert(stops == expected[frame][2], "stop requests")
end
`, wantStops))
				mode := NewWorldMode()
				loadKeyboardTestBot(t, ctx, mode)
				frame := func() {
					t.Helper()
					if err := mode.bot.InputFrame(true); err != nil {
						t.Fatal(err)
					}
					if err := mode.bot.Tick(); err != nil {
						t.Fatal(err)
					}
					ctx.Input.EndFrame()
				}
				ctx.Input.SetKeyCode(modifier, true)
				ctx.Input.SetKeyCode(code, true)
				mode.HandleKeyPress(ctx, code)
				if ctx.Input.KeyCodeConsumed(code) {
					t.Fatal("WASD consumed a modified shortcut")
				}
				frame()
				ctx.Input.SetKeyCode(modifier, false)
				frame() // Releasing the modifier restores the held control.
				ctx.Input.SetKeyCode(modifier, true)
				frame() // Pressing it during movement stops once.
				frame()
				ctx.Input.SetKeyCode(modifier, false)
				frame()
			})
		}
	}
}

func TestKeyboardObserverPreservesClassicChat(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = scriptTestFile(t, `seen = false
function keypress(code) assert(code == "KeyW"); seen = true end
function tick() assert(seen, "keypress was not invoked") end`)
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "z") || !mode.ui.console.Active() {
		t.Fatal("a keyboard observer changed classic chat activation")
	}
	if err := mode.bot.Tick(); err != nil {
		t.Fatal(err)
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
	ctx.Config.Script.Path = scriptTestFile(t, `function keypress(code) end`)
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
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
	ctx.Config.Script.Path = scriptTestFile(t, `function keypress(code) error("test") end`)
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "z") || !mode.ui.console.Active() || !mode.bot.Disabled {
		t.Fatal("failed callback swallowed input or remained active")
	}
}

func TestKeyboardConsumedPressKeepsTranslatedTextForScript(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = scriptTestFile(t, `typed = false
function keypress(code) goro.keyboard.consume_press(code) end
function input() assert(goro.keyboard.text() == "é"); typed = true end
function tick() assert(typed, "input was not invoked") end`)
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	if !prepareKeyboardTestText(mode, ctx, gpucontext.KeyE, "é") || mode.ui.console.Active() {
		t.Fatal("consumed key activated the console")
	}
	mode.updateBotInput(ctx, !mode.ui.keyboardInputBlocked(ctx))
	if err := mode.bot.Tick(); err != nil {
		t.Fatal(err)
	}
}

func TestInactiveKeyboardScriptPreservesClassicChat(t *testing.T) {
	for _, name := range []string{"tick only", "disabled", "closed", "removed", "replaced"} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
			if name == "tick only" {
				ctx.Config.Script.Path = scriptTestFile(t, `function tick() end`)
			}
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			switch name {
			case "disabled":
				mode.bot.Disabled = true
			case "closed":
				mode.bot.Close()
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

func TestLuaBotExposesWorldState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	local hp, max_hp = goro.hp()
	local sp, max_sp = goro.sp()
	local player = goro.player()
	local enemies = goro.enemies()
	local players = goro.players()
	local companions = goro.companions()
	local items = goro.items()
	local inventory = goro.inventory()
	seen = {
		hp = hp,
		max_hp = max_hp,
		sp = sp,
		max_sp = max_sp,
		player_x = player.x,
		player_y = player.y,
		player_dead = player.dead,
		enemies = #enemies,
		enemy_id = enemies[1].id,
		players = #players,
		player_id = players[1].id,
		player_party_member = players[1].party_member,
		player_hp = players[1].hp,
		player_max_hp = players[1].max_hp,
		companions = #companions,
		homunculus_id = companions[1].id,
		homunculus_kind = companions[1].kind,
		homunculus_own = companions[1].own,
		homunculus_hp = companions[1].hp,
		mercenary_id = companions[2].id,
		mercenary_kind = companions[2].kind,
		items = #items,
		item_id = items[1].item_id,
		inventory = #inventory,
		inventory_index = inventory[1].index,
		inventory_item_id = inventory[1].item_id,
		inventory_amount = inventory[1].amount,
		inventory_identified = inventory[1].identified,
		inventory_usable = inventory[1].usable,
		second_inventory_usable = inventory[2].usable,
	}
	assert(seen.hp == 42, "hp")
	assert(seen.max_hp == 100, "max_hp")
	assert(seen.sp == 7, "sp")
	assert(seen.max_sp == 20, "max_sp")
	assert(seen.player_x == 10, "player_x")
	assert(seen.player_y == 20, "player_y")
	assert(seen.player_dead == false, "player_dead")
	assert(seen.enemies == 1, "enemies")
	assert(seen.enemy_id == 300, "enemy_id")
	assert(seen.players == 1, "players")
	assert(seen.player_id == 301, "player_id")
	assert(seen.player_party_member == true, "player_party_member")
	assert(seen.player_hp == 30, "player_hp")
	assert(seen.player_max_hp == 80, "player_max_hp")
	assert(seen.companions == 2, "companions")
	assert(seen.homunculus_id == 302, "homunculus_id")
	assert(seen.homunculus_kind == "homunculus", "homunculus_kind")
	assert(seen.homunculus_own == true, "homunculus_own")
	assert(seen.homunculus_hp == 45, "homunculus_hp")
	assert(seen.mercenary_id == 303, "mercenary_id")
	assert(seen.mercenary_kind == "mercenary", "mercenary_kind")
	assert(seen.items == 1, "items")
	assert(seen.item_id == 501, "item_id")
	assert(seen.inventory == 2, "inventory")
	assert(seen.inventory_index == 4, "inventory_index")
	assert(seen.inventory_item_id == 501, "inventory_item_id")
	assert(seen.inventory_amount == 5, "inventory_amount")
	assert(seen.inventory_identified == true, "inventory_identified")
	assert(seen.inventory_usable == true, "inventory_usable")
	assert(seen.second_inventory_usable == false, "second_inventory_usable")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	sess := session.New()
	sess.CharID = 2000000
	sess.Vitals = session.Vitals{HP: 42, MaxHP: 100, SP: 7, MaxSP: 20}
	sess.Party.Members = []session.PartyMember{{AccountID: 301, Name: "Alice", HP: 30, MaxHP: 80}}
	sess.Homunculus = session.Companion{ID: 302, Active: true, HP: 45, MaxHP: 60, SP: 12, MaxSP: 20}
	sess.Mercenary = session.Companion{ID: 303, Active: true, HP: 90, MaxHP: 100, SP: 8, MaxSP: 10}
	sess.Inventory.Items = []session.InventoryItem{
		{Index: 9, ItemID: 909, Type: db.ItemTypeEtc, Amount: 2, Identified: true},
		{Index: 4, ItemID: 501, Type: db.ItemTypeHealing, Amount: 5, Identified: true},
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.CharID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "Poring", X: 12, Y: 21, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	world.Actors[301] = worldstate.Actor{ID: 301, Name: "Alice", X: 9, Y: 22, Job: db.JobAcolyte, ObjectType: scene.ActorObjectTypePC, HasObjectType: true}
	world.Actors[302] = worldstate.Actor{ID: 302, Name: "Lif", X: 11, Y: 20, ObjectType: scene.ActorObjectTypeHomunculus, HasObjectType: true}
	world.Actors[303] = worldstate.Actor{ID: 303, Name: "David", X: 10, Y: 22, ObjectType: scene.ActorObjectTypeMercenary, HasObjectType: true}
	world.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501, X: 11, Y: 20, Amount: 2, Identified: true}

	mode := &WorldMode{}
	bot, err := newLuaScript(client.Context{Session: sess, World: world}, mode, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}

}

func TestLuaPlayersIncludeUnnamedPlayers(t *testing.T) {
	sess := session.New()
	sess.AccountID = 2000000
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID, X: 10, Y: 20}
	world.Actors[2000001] = worldstate.Actor{
		ID:         2000001,
		X:          12,
		Y:          20,
		Job:        db.JobKnight,
		Appearance: true,
	}

	path := scriptTestFile(t, `
function tick()
	local players = goro.players()
	assert(#players == 1)
	assert(players[1].id == 2000001)
	assert(players[1].name == "")
end
`)
	script, err := newLuaScript(client.Context{Session: sess, World: world}, &WorldMode{}, path)
	if err != nil {
		t.Fatal(err)
	}
	defer script.Close()
	if err := script.Tick(); err != nil {
		t.Fatal(err)
	}
}

func TestLuaBotCanUseInventoryItem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	local inventory = goro.inventory()
	used = goro.use_item(inventory[1].index)
	assert(used, "goro.use_item returned false")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	networkClient, serverConn := testutil.NewConnection(t, 20080910)

	sess := session.New()
	sess.AccountID = 0x11223344
	sess.Inventory.Items = []session.InventoryItem{{
		Index:      7,
		ItemID:     501,
		Type:       db.ItemTypeHealing,
		Amount:     3,
		Identified: true,
	}}
	bot, err := newLuaScript(client.Context{Session: sess, Network: networkClient}, &WorldMode{}, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}

	want := network.BuildUseInventoryItemPacketForClientDate(7, sess.AccountID, 20080910)
	readBotTestPackets(t, serverConn, want)
}

func TestLuaBotCanAutoRevive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	revived = goro.revive()
	assert(revived, "goro.revive returned false")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	sess := session.New()
	sess.Dead = true
	sess.Inventory.Items = []session.InventoryItem{{
		ItemID: client.TokenOfSiegfriedItemID,
		Amount: 1,
	}}
	bot, err := newLuaScript(client.Context{
		Session: sess,
		World:   worldstate.New(),
		Network: networkClient,
	}, &WorldMode{}, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}

	readBotTestPackets(t, serverConn, network.BuildAutoRevivePacket())
}

func TestLuaBotCanSendMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	sent = goro.message("hello public")
		and goro.message("@autoloot 100")
		and goro.message("%hello party")
		and goro.message("$hello guild")
		and goro.message("/w Alice hello whisper")
		and goro.message("/sit")
		and goro.message("/stand")
	assert(sent, "goro.message returned false")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	sess := session.New()
	sess.AccountID = 0x11223344
	sess.Selected.Name = "Kivutar"
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID, Moving: true}
	bot, err := newLuaScript(client.Context{Session: sess, Network: networkClient, World: world}, &WorldMode{}, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}

	want := make([]byte, 0)
	want = append(want, network.BuildGlobalChatPacketForClientDate("Kivutar", "hello public", 20080910)...)
	want = append(want, network.BuildGlobalChatPacketForClientDate("Kivutar", "@autoloot 100", 20080910)...)
	want = append(want, network.BuildPartyMessagePacket("Kivutar : hello party")...)
	want = append(want, network.BuildGuildMessagePacket("Kivutar : hello guild")...)
	want = append(want, network.BuildWhisperPacket("Alice", "hello whisper")...)
	want = append(want, network.BuildActionRequestPacketForClientDate(sess.AccountID, network.ActionSitDown, 20080910)...)
	want = append(want, network.BuildActionRequestPacketForClientDate(sess.AccountID, network.ActionStandUp, 20080910)...)
	readBotTestPackets(t, serverConn, want)
	if world.Player.Sitting {
		t.Fatal("player remained sitting after /stand")
	}
	if world.Player.Moving {
		t.Fatal("player movement was not cleared by /sit")
	}
}

func TestLuaBotCanStartLongWalkFromPhysicalKey(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	inputState := input.NewState()
	keyW, ok := input.KeyCodeFromName("KeyW")
	if !ok {
		t.Fatal("KeyW was not recognized")
	}
	inputState.SetKeyCode(keyW, true)

	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	bot, err := newLuaScript(
		client.Context{Input: inputState, Network: networkClient, World: world},
		mode,
		filepath.Join("..", "scripts", "wasd.lua"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	botKeyPressForTest(t, bot, keyW)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, ok := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
	if !ok {
		t.Fatal("failed to build expected walk packet")
	}
	readBotTestPackets(t, serverConn, want)
	if inputState.KeyCodeJustPressed(keyW) {
		t.Fatal("physical W press was not consumed")
	}
	if inputState.JustPressed(input.KeyW) {
		t.Fatal("W shortcut press was not consumed")
	}
	if !inputState.KeyCodeDown(keyW) {
		t.Fatal("physical W held state was consumed")
	}

	// Model an acknowledged walk, one cell along the eight-cell path.
	world.Player = worldstate.Actor{
		ID: 2000000, X: 10, Y: 28, FromX: 10, FromY: 20, ToX: 10, ToY: 28,
		Moving: true, MoveStarted: time.Now().Add(-time.Second), MoveDuration: 8 * time.Second,
	}
	mode.walkCooldownUntil = time.Time{}
	mode.walkAwaitingAck = false
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if err := serverConn.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if _, err := serverConn.Read(make([]byte, 1)); err == nil {
		t.Fatal("held key unexpectedly refreshed the active walk")
	}

	world.Player.MoveStarted = time.Now().Add(-5 * time.Second)
	mode.walkCooldownUntil = time.Time{}
	mode.walkAwaitingAck = false
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, ok = network.BuildWalkToXYPacketForClientDate(10, 33, 20080910)
	if !ok {
		t.Fatal("failed to build expected rolling walk packet")
	}
	readBotTestPackets(t, serverConn, want)

	inputState.EndFrame()
	inputState.SetKeyCode(keyW, false)
	now := time.Now()
	world.Player = worldstate.Actor{
		ID:           2000000,
		X:            10,
		Y:            33,
		FromX:        10,
		FromY:        25,
		ToX:          10,
		ToY:          33,
		Moving:       true,
		MoveStarted:  now,
		MoveDuration: 8 * time.Second,
		MovePath: []worldstate.WalkStep{
			{X: 10, Y: 25},
			{X: 11, Y: 25},
			{X: 10, Y: 33},
		},
	}
	mode.walkCooldownUntil = time.Time{}
	mode.walkAwaitingAck = false
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, ok = network.BuildWalkToXYPacketForClientDate(11, 25, 20080910)
	if !ok {
		t.Fatal("failed to build expected release retarget packet")
	}
	readBotTestPackets(t, serverConn, want)
}

func TestLuaBotCanWalkDiagonally(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	inputState := input.NewState()
	for _, name := range []string{"KeyW", "KeyD"} {
		code, ok := input.KeyCodeFromName(name)
		if !ok {
			t.Fatalf("%s was not recognized", name)
		}
		inputState.SetKeyCode(code, true)
	}

	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	bot, err := newLuaScript(
		client.Context{Input: inputState, Network: networkClient, World: world},
		&WorldMode{},
		filepath.Join("..", "scripts", "wasd.lua"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, ok := network.BuildWalkToXYPacketForClientDate(18, 28, 20080910)
	if !ok {
		t.Fatal("failed to build expected diagonal walk packet")
	}
	readBotTestPackets(t, serverConn, want)
}

func TestLuaBotHeldSpaceLootsNearbyItemsInDistanceOrder(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	inputState := input.NewState()
	space, ok := input.KeyCodeFromName("Space")
	if !ok {
		t.Fatal("Space was not recognized")
	}
	inputState.SetKeyCode(space, true)

	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501, X: 12, Y: 20}
	world.Items[401] = worldstate.FloorItem{ID: 401, ItemID: 501, X: 11, Y: 20}
	world.Items[402] = worldstate.FloorItem{ID: 402, ItemID: 501, X: 30, Y: 20}
	bot, err := newLuaScript(
		client.Context{Input: inputState, Network: networkClient, World: world},
		&WorldMode{},
		filepath.Join("..", "scripts", "wasd.lua"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	botKeyPressForTest(t, bot, space)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildItemPickupPacketForClientDate(401, 20080910))
	assertNoBotTestPacket(t, serverConn, func() error { return bot.Tick() })

	if err := bot.InputFrame(false); err != nil {
		t.Fatal(err)
	}
	delete(world.Items, 401)
	world.Player.X = 11
	assertNoBotTestPacket(t, serverConn, func() error { return bot.Tick() })

	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildItemPickupPacketForClientDate(400, 20080910))
	if inputState.KeyCodeJustPressed(space) {
		t.Fatal("Space press was not consumed")
	}
	if !inputState.KeyCodeDown(space) {
		t.Fatal("Space held state was consumed")
	}
}

func TestLuaBotHeldFightsNearbyEnemiesInDistanceOrder(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	inputState := input.NewState()
	keyF, ok := input.KeyCodeFromName("KeyF")
	if !ok {
		t.Fatal("KeyF was not recognized")
	}
	inputState.SetKeyCode(keyF, true)

	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "Poring", X: 12, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	world.Actors[301] = worldstate.Actor{ID: 301, Name: "Drops", X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	world.Actors[302] = worldstate.Actor{ID: 302, Name: "Lunatic", X: 30, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	bot, err := newLuaScript(
		client.Context{Input: inputState, Network: networkClient, World: world},
		&WorldMode{},
		filepath.Join("..", "scripts", "wasd.lua"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	botKeyPressForTest(t, bot, keyF)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
	readLegacyBotTestActionPacket(t, serverConn, 301, network.ActionAttack)
	assertNoBotTestPacket(t, serverConn, func() error { return bot.Tick() })

	delete(world.Actors, 301)
	world.Player.X = 11
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
	readLegacyBotTestActionPacket(t, serverConn, 300, network.ActionAttack)
	if inputState.KeyCodeJustPressed(keyF) {
		t.Fatal("physical F press was not consumed")
	}
	if !inputState.KeyCodeDown(keyF) {
		t.Fatal("physical F held state was consumed")
	}
}

func TestLuaBotExposesPendingSkillAndActorHighlight(t *testing.T) {
	path := filepath.Join(t.TempDir(), "targeting.lua")
	if err := os.WriteFile(path, []byte(`
function input()
	pending = goro.pending_skill()
	highlighted = goro.highlight_actor(300)
	assert(pending ~= nil)
	assert(pending.id == 46, "id")
	assert(pending.name == "Double Strafe", "name")
	assert(pending.level == 3, "level")
	assert(pending.max_level == 10, "max_level")
	assert(pending.type == 1, "type")
	assert(pending.target == "actor", "target")
	assert(pending.caster_id == 200, "caster_id")
	assert(pending.caster_kind == "player", "caster_kind")
	assert(pending.caster_x == 10, "caster_x")
	assert(pending.caster_y == 20, "caster_y")
	assert(highlighted, "goro.highlight_actor returned false")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	sess := session.New()
	sess.AccountID = 100
	sess.CharID = 200
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.CharID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: db.SkillACDouble, Name: "Double Strafe", Type: skills.SkillTargetEnemy, Level: 3, Range: 9},
		MaxLevel: 10,
	}},
	}
	bot, err := newLuaScript(client.Context{Session: sess, World: world}, mode, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := bot.InputFrame(true); err != nil {
		bot.Close()
		t.Fatal(err)
	}

	if mode.scriptHighlight.id != 300 || mode.scriptHighlight.started.IsZero() {
		bot.Close()
		t.Fatalf("script highlight = %+v, want actor 300", mode.scriptHighlight)
	}

	bot.Close()
	if mode.scriptHighlight.id != 0 {
		t.Fatalf("script highlight survived bot close: %+v", mode.scriptHighlight)
	}
}

func TestWASDLuaCyclesAndUsesPendingSkillTargets(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	inputState := input.NewState()
	tab, ok := input.KeyCodeFromName("Tab")
	if !ok {
		t.Fatal("Tab was not recognized")
	}
	enter, ok := input.KeyCodeFromName("Enter")
	if !ok {
		t.Fatal("Enter was not recognized")
	}
	shift, ok := input.KeyCodeFromName("ShiftLeft")
	if !ok {
		t.Fatal("ShiftLeft was not recognized")
	}

	sess := session.New()
	sess.AccountID = 100
	sess.CharID = 200
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.CharID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	world.Actors[301] = worldstate.Actor{ID: 301, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	world.Actors[302] = worldstate.Actor{ID: 302, X: 10, Y: 21, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	world.Actors[400] = worldstate.Actor{ID: 400, X: 10, Y: 19, ObjectType: scene.ActorObjectTypePC, HasObjectType: true}
	skill := session.Skill{ID: db.SkillACDouble, Type: skills.SkillTargetEnemy, Level: 3, Range: 9}
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{Skill: skill, MaxLevel: 10}}, actorSystem: actors.System{ActorDeaths: map[uint32]time.Time{302: time.Now().Add(time.Second)}}}
	bot, err := newLuaScript(
		client.Context{Input: inputState, Network: networkClient, Session: sess, World: world},
		mode,
		filepath.Join("..", "scripts", "wasd.lua"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	inputState.SetKeyCode(tab, true)
	botKeyPressForTest(t, bot, tab)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if got := mode.scriptHighlight.id; got != 301 {
		t.Fatalf("first Tab target = %d, want nearest living enemy 301", got)
	}
	if inputState.KeyCodeJustPressed(tab) || inputState.JustPressed(input.KeyTab) {
		t.Fatal("Tab press was not consumed")
	}

	inputState.SetKeyCode(tab, false)
	inputState.EndFrame()
	for _, name := range []string{"ControlLeft", "ControlRight", "AltLeft", "AltRight", "MetaLeft", "MetaRight"} {
		modifier, ok := input.KeyCodeFromName(name)
		if !ok {
			t.Fatalf("%s was not recognized", name)
		}
		inputState.SetKeyCode(modifier, true)
		for _, code := range []input.KeyCode{tab, enter} {
			inputState.SetKeyCode(code, true)
			botKeyPressForTest(t, bot, code)
			if !inputState.KeyCodeJustPressed(code) || mode.scriptHighlight.id != 301 || mode.skillSystem.Pending.Skill.ID != skill.ID {
				t.Fatalf("%s+%s changed skill targeting or consumed the shortcut", name, input.KeyCodeName(code))
			}
			inputState.SetKeyCode(code, false)
			inputState.EndFrame()
		}
		inputState.SetKeyCode(modifier, false)
	}
	inputState.SetKeyCode(tab, true)
	botKeyPressForTest(t, bot, tab)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if got := mode.scriptHighlight.id; got != 300 {
		t.Fatalf("second Tab target = %d, want enemy 300", got)
	}

	inputState.SetKeyCode(tab, false)
	inputState.EndFrame()
	inputState.SetKeyCode(shift, true)
	inputState.SetKeyCode(tab, true)
	botKeyPressForTest(t, bot, tab)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if got := mode.scriptHighlight.id; got != 301 {
		t.Fatalf("reverse Tab target = %d, want enemy 301", got)
	}

	inputState.SetKeyCode(tab, false)
	inputState.SetKeyCode(shift, false)
	inputState.EndFrame()
	inputState.SetKeyCode(enter, true)
	botKeyPressForTest(t, bot, enter)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildUseSkillToIDPacketForClientDate(skill.ID, uint16(skill.Level), 301, 20080910))
	if mode.skillSystem.Pending.Skill.ID != 0 {
		t.Fatalf("pending skill remained after Enter: %+v", mode.skillSystem.Pending)
	}
	if mode.scriptHighlight.id != 0 {
		t.Fatalf("script highlight remained after Enter: %+v", mode.scriptHighlight)
	}
	if inputState.KeyCodeJustPressed(enter) || inputState.JustPressed(input.KeyEnter) {
		t.Fatal("Enter press was not consumed for a focused target")
	}
}

func TestWASDLuaCyclesFriendlySkillTargets(t *testing.T) {
	inputState := input.NewState()
	tab, ok := input.KeyCodeFromName("Tab")
	if !ok {
		t.Fatal("Tab was not recognized")
	}
	sess := session.New()
	sess.AccountID = 100
	sess.CharID = 200
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.CharID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "Alice", X: 11, Y: 20, ObjectType: scene.ActorObjectTypePC, HasObjectType: true}
	world.Actors[301] = worldstate.Actor{ID: 301, Name: "Lif", X: 12, Y: 20, ObjectType: scene.ActorObjectTypeHomunculus, HasObjectType: true}
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill: session.Skill{ID: db.SkillALHeal, Type: skills.SkillTargetFriend, Level: 7, Range: 9},
	}},
	}
	bot, err := newLuaScript(
		client.Context{Input: inputState, Session: sess, World: world},
		mode,
		filepath.Join("..", "scripts", "wasd.lua"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	pressTab := func() {
		t.Helper()
		inputState.SetKeyCode(tab, true)
		botKeyPressForTest(t, bot, tab)
		if err := bot.InputFrame(true); err != nil {
			t.Fatal(err)
		}
		inputState.SetKeyCode(tab, false)
		inputState.EndFrame()
	}
	for index, want := range []uint32{sess.AccountID, 300, 301} {
		pressTab()
		if got := mode.scriptHighlight.id; got != want {
			t.Fatalf("friendly Tab target %d = %d, want %d", index+1, got, want)
		}
	}
}

func TestWASDHealReverseCycleThenConfirm(t *testing.T) {
	for _, tc := range []struct {
		name, shift string
		noShift     bool
	}{
		{"left_shift", "ShiftLeft", false},
		{"right_shift", "ShiftRight", false},
		{"noshift_left", "ShiftLeft", true},
		{"noshift_right", "ShiftRight", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			conn, server := testutil.NewConnection(t, 20080910)
			ctx.Network = conn
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.Session.AccountID, ctx.Session.CharID = 100, 200
			ctx.Session.NoShift = tc.noShift
			ctx.World = worldstate.New()
			ctx.World.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
			ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, HasObjectType: true, ObjectType: scene.ActorObjectTypeMob}
			mode := NewWorldMode()
			skill := session.Skill{ID: db.SkillALHeal, Type: skills.SkillTargetFriend, Level: 3, Range: 9}
			mode.skillSystem.Pending = skills.PendingTarget{Skill: skill}
			loadKeyboardTestBot(t, ctx, mode)
			tab, _ := input.KeyCodeFromName("Tab")
			shift, _ := input.KeyCodeFromName(tc.shift)
			enter, _ := input.KeyCodeFromName("Enter")
			ctx.Input.SetKeyCode(tab, true)
			botKeyPressForTest(t, mode.bot, tab)
			if mode.scriptHighlight.id != ctx.Session.AccountID {
				t.Fatal("initial Tab should select self")
			}
			ctx.Input.SetKeyCode(tab, false)
			ctx.Input.EndFrame()
			ctx.Input.SetKeyCode(shift, true)
			ctx.Input.SetKeyCode(tab, true)
			botKeyPressForTest(t, mode.bot, tab)
			want := ctx.Session.AccountID
			if tc.noShift {
				want = 300
			}
			if mode.scriptHighlight.id != want {
				t.Fatalf("Shift+Tab selected %d, want %d", mode.scriptHighlight.id, want)
			}
			ctx.Input.SetKeyCode(tab, false)
			ctx.Input.SetKeyCode(shift, false)
			ctx.Input.EndFrame()
			ctx.Input.SetKeyCode(enter, true)
			botKeyPressForTest(t, mode.bot, enter)
			readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 3, want, 20080910))
		})
	}
}

func TestWASDLuaLeavesUnrelatedTargetingKeysAlone(t *testing.T) {
	inputState := input.NewState()
	enter, ok := input.KeyCodeFromName("Enter")
	if !ok {
		t.Fatal("Enter was not recognized")
	}
	tab, ok := input.KeyCodeFromName("Tab")
	if !ok {
		t.Fatal("Tab was not recognized")
	}
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill: session.Skill{ID: db.SkillACDouble, Type: skills.SkillTargetEnemy, Level: 3, Range: 9},
	}},
	}
	bot, err := newLuaScript(
		client.Context{Input: inputState},
		mode,
		filepath.Join("..", "scripts", "wasd.lua"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	inputState.SetKeyCode(enter, true)
	botKeyPressForTest(t, bot, enter)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if !inputState.KeyCodeJustPressed(enter) || !inputState.JustPressed(input.KeyEnter) {
		t.Fatal("Enter without a focused target was consumed")
	}

	inputState.SetKeyCode(enter, false)
	inputState.EndFrame()
	mode.skillSystem.Pending = skills.PendingTarget{
		Skill: session.Skill{ID: 18, Type: skills.SkillTargetPlace, Level: 1, Range: 9},
	}
	inputState.SetKeyCode(tab, true)
	botKeyPressForTest(t, bot, tab)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if !inputState.KeyCodeJustPressed(tab) || !inputState.JustPressed(input.KeyTab) {
		t.Fatal("Tab for a ground-targeted skill was consumed")
	}
}

func TestExampleLuaBotUsesPotionRestsAndResumes(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	sess := session.New()
	sess.AccountID = 0x11223344
	sess.Selected.Name = "Kivutar"
	sess.Vitals = session.Vitals{HP: 50, MaxHP: 100, SP: 20, MaxSP: 20}
	sess.Inventory.Items = []session.InventoryItem{{
		Index:      7,
		ItemID:     501,
		Type:       db.ItemTypeHealing,
		Amount:     3,
		Identified: true,
	}}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID}
	bot, err := newLuaScript(
		client.Context{Session: sess, Network: networkClient, World: world},
		&WorldMode{},
		filepath.Join("..", "scripts", "loot-and-attack.lua"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, serverConn, network.BuildUseInventoryItemPacketForClientDate(7, sess.AccountID, 20080910))
	if world.Player.Sitting {
		t.Fatal("bot rested while a healing potion was available above its critical threshold")
	}

	sess.Inventory.Items = nil
	sess.Vitals = session.Vitals{HP: 20, MaxHP: 100, SP: 2, MaxSP: 20}
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
	readLegacyBotTestActionPacket(t, serverConn, sess.AccountID, network.ActionSitDown)
	readBotTestPackets(t, serverConn, network.BuildGlobalChatPacketForClientDate("Kivutar", "I'm tired. Resting for a moment.", 20080910))
	if !world.Player.Sitting {
		t.Fatal("bot did not enter the resting state")
	}

	sess.Vitals = session.Vitals{HP: 90, MaxHP: 100, SP: 16, MaxSP: 20}
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
	readLegacyBotTestActionPacket(t, serverConn, sess.AccountID, network.ActionStandUp)
	readBotTestPackets(t, serverConn, network.BuildGlobalChatPacketForClientDate("Kivutar", "Ready to go again.", 20080910))
	if world.Player.Sitting {
		t.Fatal("bot did not leave the resting state")
	}
}

func TestScriptUseItemRejectsInvalidInventoryEntries(t *testing.T) {
	sess := session.New()
	sess.Inventory.Items = []session.InventoryItem{{Index: 7, ItemID: 909, Type: db.ItemTypeEtc, Amount: 1}}
	ctx := client.Context{Session: sess}

	for _, index := range []int{-1, 0, 7, 8, 1 << 16} {
		if scriptUseItem(ctx, index) {
			t.Fatalf("scriptUseItem(%d) = true, want false", index)
		}
	}
}

func TestLuaBotCanRequestTargetSkillChase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	ok = goro.skill(300, 46)
	assert(ok, "goro.skill returned false")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	sess := session.New()
	sess.CharID = 2000000
	sess.Skills.List = []session.Skill{{
		ID:    db.SkillACDouble,
		Type:  skills.SkillTargetEnemy,
		Level: 10,
		Range: 9,
		Name:  "Double Strafe",
	}}
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	world.Player = worldstate.Actor{ID: sess.CharID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "Poring", X: 30, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}

	mode := &WorldMode{}
	bot, err := newLuaScript(client.Context{Session: sess, World: world}, mode, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}

	if mode.skillSystem.Pending.TargetID != 300 || mode.skillSystem.Pending.Skill.ID != db.SkillACDouble {
		t.Fatalf("pending skill = %+v, want AC_DOUBLE target 300", mode.skillSystem.Pending)
	}
}

func TestLuaBotCanUseSelfSkill(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	assert(goro.skill(goro.player().id, "AL_ANGELUS", 3))
	assert(goro.skill(goro.player().id, "AL_ANGELUS"))
	assert(not goro.skill(300, "AL_ANGELUS"))
	assert(not goro.skill(goro.player().id, "AL_ANGELUS", 6))
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	sess := session.New()
	sess.AccountID = 2000000
	sess.CharID = 150000
	sess.Skills.List = []session.Skill{{ID: db.SkillALAngelus, Type: skills.SkillTargetSelf, Level: 5, Name: "Angelus"}}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID, X: 10, Y: 20}
	ctx := client.Context{Session: sess, Network: networkClient, World: world}
	mode := &WorldMode{}
	bot, err := newLuaScript(ctx, mode, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
	want := network.BuildUseSkillToIDPacketForClientDate(db.SkillALAngelus, 3, sess.AccountID, 20080910)
	want = append(want, network.BuildUseSkillToIDPacketForClientDate(db.SkillALAngelus, 5, sess.AccountID, 20080910)...)
	readBotTestPackets(t, serverConn, want)

	sess.Dead = true
	if mode.scriptSkill(ctx, sess.AccountID, sess.Skills.List[0], -1) {
		t.Fatal("scriptSkill allowed a dead player to cast Angelus")
	}
	assertNoBotTestPacket(t, serverConn, func() error { return nil })
}

func TestLuaBotCanHealNearbyPlayer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	local players = goro.players()
	healed = goro.skill(players[1].id, "AL_HEAL")
	assert(healed, "goro.skill returned false for Heal on a nearby player")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	sess := session.New()
	sess.AccountID = 2000000
	sess.Skills.List = []session.Skill{{
		ID:    db.SkillALHeal,
		Type:  skills.SkillTargetFriend,
		Level: 7,
		Range: 9,
		Name:  "Heal",
	}}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		Name:          "Alice",
		X:             12,
		Y:             21,
		Job:           db.JobAcolyte,
		ObjectType:    scene.ActorObjectTypePC,
		HasObjectType: true,
	}

	bot, err := newLuaScript(client.Context{Session: sess, Network: networkClient, World: world}, &WorldMode{}, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}

	want := network.BuildUseSkillToIDPacketForClientDate(db.SkillALHeal, 7, 300, 20080910)
	readBotTestPackets(t, serverConn, want)
}

func TestLuaBotCanTargetCompanionsWithFriendlySkills(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	for _, companion in ipairs(goro.companions()) do
		if companion.kind == "homunculus" then
			pitched = goro.skill(companion.id, "AM_POTIONPITCHER", 3)
		elseif companion.kind == "mercenary" then
			blessed = goro.skill(companion.id, "AL_BLESSING")
		end
	end
	assert(pitched, "goro.skill returned false for Potion Pitcher on a homunculus")
	assert(blessed, "goro.skill returned false for Blessing on a mercenary")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	sess := session.New()
	sess.AccountID = 2000000
	sess.Skills.List = []session.Skill{
		{ID: db.SkillAMPotionpitcher, Type: skills.SkillTargetFriend, Level: 5, Range: 9, Name: "Potion Pitcher"},
		{ID: db.SkillALBlessing, Type: skills.SkillTargetFriend, Level: 10, Range: 9, Name: "Blessing"},
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "Lif", X: 11, Y: 20, ObjectType: scene.ActorObjectTypeHomunculus, HasObjectType: true}
	world.Actors[301] = worldstate.Actor{ID: 301, Name: "David", X: 12, Y: 20, ObjectType: scene.ActorObjectTypeMercenary, HasObjectType: true}

	bot, err := newLuaScript(client.Context{Session: sess, Network: networkClient, World: world}, &WorldMode{}, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}

	want := make([]byte, 0)
	want = append(want, network.BuildUseSkillToIDPacketForClientDate(db.SkillAMPotionpitcher, 3, 300, 20080910)...)
	want = append(want, network.BuildUseSkillToIDPacketForClientDate(db.SkillALBlessing, 10, 301, 20080910)...)
	readBotTestPackets(t, serverConn, want)
}

func TestLuaBotRejectsSkillTargetMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.lua")
	if err := os.WriteFile(path, []byte(`
function tick()
	healed_enemy = goro.skill(300, "AL_HEAL")
	attacked_player = goro.skill(301, "SM_BASH")
	attacked_companion = goro.skill(302, "SM_BASH")
	invalid_level = goro.skill(301, "AL_HEAL", 8)
	assert(not healed_enemy, "goro.skill allowed Heal on an enemy")
	assert(not attacked_player, "goro.skill allowed an enemy skill on a player")
	assert(not attacked_companion, "goro.skill allowed an enemy skill on a companion")
	assert(not invalid_level, "goro.skill allowed a level above the learned skill level")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	sess := session.New()
	sess.AccountID = 2000000
	sess.Skills.List = []session.Skill{
		{ID: db.SkillALHeal, Type: skills.SkillTargetFriend, Level: 7, Range: 9, Name: "Heal"},
		{ID: db.SkillSMBash, Type: skills.SkillTargetEnemy, Level: 10, Range: 1, Name: "Bash"},
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.AccountID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "Poring", X: 11, Y: 20, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	world.Actors[301] = worldstate.Actor{ID: 301, Name: "Alice", X: 11, Y: 21, Job: db.JobAcolyte, ObjectType: scene.ActorObjectTypePC, HasObjectType: true}
	world.Actors[302] = worldstate.Actor{ID: 302, Name: "Lif", X: 10, Y: 21, ObjectType: scene.ActorObjectTypeHomunculus, HasObjectType: true}

	bot, err := newLuaScript(client.Context{Session: sess, World: world}, &WorldMode{}, path)
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()
	if err := bot.Tick(); err != nil {
		t.Fatal(err)
	}
}

func TestLuaBotDoesNotExposeDyingEnemies(t *testing.T) {
	sess := session.New()
	sess.CharID = 2000000
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: sess.CharID, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "Poring", X: 12, Y: 21, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}

	mode := &WorldMode{actorSystem: actors.System{ActorDeaths: map[uint32]time.Time{300: time.Now().Add(time.Second)}}}
	path := scriptTestFile(t, `function tick() assert(#goro.enemies() == 0) end`)
	script, err := newLuaScript(client.Context{Session: sess, World: world}, mode, path)
	if err != nil {
		t.Fatal(err)
	}
	defer script.Close()
	if err := script.Tick(); err != nil {
		t.Fatal(err)
	}
}

var readBotTestPackets = testutil.ReadPackets

func assertNoBotTestPacket(t *testing.T, conn net.Conn, action func() error) {
	t.Helper()
	if err := action(); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	n, err := conn.Read(make([]byte, 1))
	if n != 0 || err == nil {
		t.Fatalf("unexpected packet: bytes=%d err=%v", n, err)
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("checking for packet: %v", err)
	}
}

func readLegacyBotTestActionPacket(t *testing.T, conn net.Conn, targetID uint32, action uint8) {
	t.Helper()
	packet := make([]byte, 19)
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(conn, packet); err != nil {
		t.Fatal(err)
	}
	if got := network.ID(packet); got != network.PacketCZRequestAct {
		t.Fatalf("action packet ID = 0x%04x, want 0x%04x", got, network.PacketCZRequestAct)
	}
	if got := binary.LittleEndian.Uint32(packet[5:9]); got != targetID {
		t.Fatalf("action target = %d, want %d", got, targetID)
	}
	if got := packet[18]; got != action {
		t.Fatalf("action = %d, want %d", got, action)
	}
}

func TestScriptLifetimeAcrossLoginAndWorld(t *testing.T) {
	ctx := wasdControlsTestContext(t)
	ctx.Config.Headless = true
	ctx.World.MapName = ""
	ctx.Network = network.NewClient(20080910, false)
	t.Cleanup(ctx.Network.Close)
	ctx.Config.Script.Path = filepath.Join(t.TempDir(), "controls.lua")
	output := filepath.Join(t.TempDir(), "callbacks.txt")
	if err := os.WriteFile(ctx.Config.Script.Path, []byte(fmt.Sprintf(`
inputs, ticks = 0, 0
local function report()
	local file = assert(io.open(%q, "w"))
	file:write(inputs, " ", ticks, " ", tostring(pressed), " ", tostring(held))
	file:close()
end
function input() inputs = inputs + 1; report() end
function tick() ticks = ticks + 1; report() end
function gamepad()
    pressed = goro.gamepad.was_pressed("south")
    held = goro.gamepad.is_down("south")
    report()
end
if not goro.in_game() then
    assert(#goro.enemies() == 0 and #goro.players() == 0 and #goro.companions() == 0)
    assert(#goro.items() == 0 and #goro.inventory() == 0 and #goro.skill_targets() == 0)
    assert(goro.pending_skill() == nil and goro.player().hp == 0)
    assert(not goro.walk(1, 1) and not goro.stop() and not goro.attack(1))
    assert(not goro.target(1) and not goro.skill(1, 28) and not goro.loot(1))
    assert(not goro.use_item(1) and not goro.revive() and not goro.message("hello"))
    assert(not goro.use_pending_skill(1) and not goro.highlight_actor(1))
    assert(not goro.use_shortcut(1) and not goro.npc_dialog())
    assert(goro.camera_yaw() == 0)
    goro.rotate_camera(1, 1); goro.zoom_camera(1); goro.cancel_skill()
end
`, output)), 0600); err != nil {
		t.Fatal(err)
	}
	m := NewManager(ctx, NewLoginMode())
	t.Cleanup(m.Close)
	if err := m.Update(); err != nil {
		t.Fatal(err)
	}
	loginScript := m.script
	if loginScript == nil || loginScript.Disabled {
		t.Fatal("script did not load safely in login")
	}
	assertScriptOutput(t, output, "1 0 nil nil")
	// A new context must also reach the Lua bindings, rather than the input
	// pointer captured at registration time.
	ctx.Input = input.NewState()
	pad := input.GamepadFrame{ID: "test"}
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	m.UpdateContext(ctx)
	m.HandleGamepadInput(ctx, 0.02)
	assertScriptOutput(t, output, "1 0 true true")
	world := NewWorldMode()
	m.enter(world)
	world.mapFade = maps.Fade{} // Finish map loading before exercising controls.
	if loginScript.Active() {
		t.Fatal("login Lua state survived transition")
	}
	ctx.Input.EndFrame()
	capture := m.HandleGamepadInput(ctx, 0.02)
	if world.bot != m.script || m.script.Disabled {
		t.Fatal("world did not bind a fresh script")
	}
	if !capture.Buttons[input.GamepadSouth] {
		t.Fatal("login confirmation leaked into world controls")
	}
	assertScriptOutput(t, output, "0 0 false false")
	world.bot.NextTick = time.Time{}
	world.updateBot(ctx, time.Now())
	assertScriptOutput(t, output, "0 1 false false")
	ctx.Input.SetGamepad(input.GamepadFrame{ID: "test"})
	m.HandleGamepadInput(ctx, 0.02)
	ctx.Input.EndFrame()
	ctx.Input.SetGamepad(pad)
	m.HandleGamepadInput(ctx, 0.02)
	assertScriptOutput(t, output, "0 1 true true")
	worldScript := m.script
	m.enter(NewLoginMode())
	if worldScript.Active() || world.bot != nil {
		t.Fatal("world retained the old script")
	}
	m.syncScript(ctx)
	if m.script.Disabled {
		t.Fatal("returning to login failed to load script")
	}
	last := m.script
	m.Close()
	if last.Active() || m.script != nil {
		t.Fatal("closing manager leaked the script")
	}
}

func scriptTestFile(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.lua")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertScriptOutput(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("script output = %q, want %q", got, want)
	}
}
