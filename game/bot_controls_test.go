package game

import (
	"math"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func wasdControlsTestContext(t *testing.T) client.Context {
	t.Helper()
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.Session.AccountID = 100
	ctx.World = worldstate.New()
	ctx.World.GAT = flatWalkableGAT(64, 64)
	ctx.World.Player = worldstate.Actor{ID: 100, X: 10, Y: 20}
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, HasObjectType: true, ObjectType: actorObjectTypeMob}
	ctx.World.Actors[301] = worldstate.Actor{ID: 301, X: 11, Y: 21, HasObjectType: true, ObjectType: actorObjectTypeMob}
	skill := session.Skill{ID: db.SkillACDouble, Type: skillTargetEnemy, Level: 3, Range: 9}
	heal := session.Skill{ID: db.SkillALHeal, Type: skillTargetFriend, Level: 3, Range: 9}
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
	mode.HandleGamepadInput(ctx, 1.0/60)
	mode.updateBotInput(ctx, true)
	if mode.ui.shortcutBar.UpdateKeyboardInput(ctx, mode, false) {
		t.Fatal("script action also reached the native hotbar")
	}
	if err := mode.bot.tick(); err != nil {
		t.Fatal(err)
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKeyCode(key, false)
	ctx.Input.SetGamepad(input.GamepadFrame{ID: pad.ID})
	mode.HandleGamepadInput(ctx, 1.0/60)
	mode.updateBotInput(ctx, true)
	ctx.Input.EndFrame()
	if mode.bot.disabled {
		t.Fatal("control script failed")
	}
}

func TestWASDKeyboardAndGamepadUseSameTargeting(t *testing.T) {
	for _, selector := range []string{"keyboard", "gamepad"} {
		for _, caster := range []string{"keyboard", "gamepad"} {
			for _, order := range []string{"target_first", "skill_first", "ineligible", "cancel", "repeat_skill", "attack_nearest", "attack_selected"} {
				t.Run(selector+"/"+caster+"/"+order, func(t *testing.T) {
					ctx := wasdControlsTestContext(t)
					conn, server := newBotTestConnection(t, 20080910)
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
				conn, server := newBotTestConnection(t, 20080910)
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
					mode.HandleGamepadInput(ctx, 1.0/60)
					mode.updateBotInput(ctx, true)
					mode.ui.shortcutBar.UpdateKeyboardInput(ctx, mode, false)
					ctx.Input.EndFrame()
					if mode.bot.disabled {
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
				if mode.pendingSkill.skill.ID == 0 || mode.pendingSkill.targetID != 0 {
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
					mode.actorDeaths = map[uint32]time.Time{300: time.Now()}
				case "hidden":
					actor := ctx.World.Actors[300]
					actor.EffectState = db.EffectStateHide
					ctx.World.Actors[300] = actor
				}
				wasdControlPress(t, ctx, mode, device, "confirm")
				if mode.scriptHighlight.id != 0 || mode.pendingSkill.skill.ID == 0 {
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
					if capture := mode.HandleGamepadInput(ctx, 1.0/60); capture.Buttons[input.GamepadSouth] {
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
				conn, server := newBotTestConnection(t, 20080910)
				ctx.Network = conn
				clear(ctx.World.Actors)
				ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 30, Y: 20, HasObjectType: true, ObjectType: actorObjectTypeMob}
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
				ctx.Input.SetKeyCode(gpucontext.KeyW, false)
				wasdControlPress(t, ctx, mode, device, castAction)
				if mode.pendingSkill.targetID != 300 {
					t.Fatal("skill did not start chasing its target")
				}
				chase, _ := network.BuildWalkToXYPacketForClientDate(20, 20, 20080910)
				readBotTestPackets(t, server, chase)
				// Releasing the old movement must not stop the new skill chase,
				// even before the server acknowledges that new destination.
				mode.walkCooldownUntil = time.Time{}
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
	mode.HandleGamepadInput(ctx, 1.0/60)
	mode.updateBotInput(ctx, true)
	if mode.ui.shortcutBar.UpdateKeyboardInput(ctx, mode, false) {
		t.Fatal("shortcut also reached the native hotbar")
	}
	ctx.Input.EndFrame()
	if mode.bot.disabled {
		t.Fatal("control script failed")
	}
}

func TestWASDSkillsResumeHeldMovementAfterServerStop(t *testing.T) {
	for _, device := range []string{"keyboard", "gamepad"} {
		for _, skill := range []session.Skill{
			{ID: db.SkillALAngelus, Type: skillTargetSelf, Level: 3},
			{ID: db.SkillALHeal, Type: skillTargetFriend, Level: 3, Range: 9},
			{ID: db.SkillMGFirewall, Type: skillTargetPlace, Level: 3, Range: 9},
		} {
			t.Run(device+"/"+skillLabel(skill), func(t *testing.T) {
				ctx := wasdControlsTestContext(t)
				conn, server := newBotTestConnection(t, 20080910)
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
				if !isSelfTargetSkill(skill) {
					frame(false)
					if isGroundTargetSkill(skill) {
						if err := mode.skills().UseGround(ctx, skill, 12, 20, "", "target"); err != nil {
							t.Fatal(err)
						}
					} else {
						frame(true) // Confirm the already highlighted self target.
					}
				}
				if isGroundTargetSkill(skill) {
					readBotTestPackets(t, server, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 3, 12, 20, 20080910))
				} else {
					readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 3, 100, 20080910))
				}
				// The reply can arrive after another input frame; keep the old
				// walk until the server actually interrupts it for the cast.
				mode.walkCooldownUntil = time.Time{}
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
			conn, server := newBotTestConnection(t, 20080910)
			ctx.Network = conn
			skill := session.Skill{ID: db.SkillMGFirewall, Type: skillTargetPlace, Level: 3, Range: 9}
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
			projection := newSceneProjectionForTarget(ctx.ScreenW, ctx.ScreenH, cellCenter(10), cellCenter(20), 0)
			point := projection.Project(cellCenter(30), cellCenter(20), 0)
			ctx.Input.SetMousePosition(int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
			if device == "gamepad" {
				pad := input.GamepadFrame{ID: "test"}
				pad.Axes[input.GamepadLeftY] = -1
				pad.Buttons[input.GamepadSouth] = true
				ctx.Input.SetGamepad(pad)
				if capture := mode.HandleGamepadInput(ctx, 1.0/60); capture.Buttons[input.GamepadSouth] {
					t.Fatal("ground targeting did not leave South to the pointer")
				}
			}
			mode.updateBotInput(ctx, true)
			mode.skills().HandleClick(ctx, projection, time.Now())
			ctx.Input.EndFrame()
			chase, _ := network.BuildWalkToXYPacketForClientDate(20, 20, 20080910)
			readBotTestPackets(t, server, chase)
			applySelfMoveAck(ctx, network.SelfMoveAck{FromX: 10, FromY: 20, ToX: 20, ToY: 20})
			mode.walkCooldownUntil = time.Time{}
			assertNoBotTestPacket(t, server, func() error { frame(false, false); return nil })
			if !mode.pendingSkill.ground {
				t.Fatal("ground skill chase was canceled")
			}
			ctx.World.Player.MoveStarted = time.Now().Add(-ctx.World.Player.MoveDuration - time.Millisecond)
			mode.skills().UpdatePendingTarget(ctx, "test", false)
			mode.pendingSkill.readyAt = time.Now().Add(-time.Millisecond)
			mode.skills().ProcessPendingTarget(ctx)
			readBotTestPackets(t, server, network.BuildUseSkillToGroundPacketForClientDate(skill.ID, 3, 30, 20, 20080910))
		})
	}
}
