package game

import (
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
			pad := input.GamepadFrame{ID: "test"}
			switch name {
			case "stick", "keyboard_and_stick":
				pad.Axes[input.GamepadLeftX], pad.Axes[input.GamepadLeftY] = 0.8, -0.8
			case "dpad":
				pad.Buttons[input.GamepadRight], pad.Buttons[input.GamepadUp] = true, true
			case "attack":
				pad.Buttons[input.GamepadSouth] = true
				pad.Buttons[input.GamepadRightShoulder] = true
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
			mode.HandleGamepadInput(ctx, 1.0/60)
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

func TestWASDGamepadSkillChordUsesSelectedEnemyWithoutAttackOrLoot(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	conn, server := newBotTestConnection(t, 20080910)
	ctx.Network = conn
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.Session.AccountID = ctx.World.Player.ID
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
	skill := session.Skill{ID: db.SkillACDouble, Type: skillTargetEnemy, Level: 3, Range: 9}
	selfSkill := session.Skill{ID: db.SkillALAngelus, Type: skillTargetSelf, Level: 3}
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
		mode.HandleGamepadInput(ctx, 1.0/60)
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
	capture := mode.HandleGamepadInput(ctx, 1.0/60)
	for _, button := range []input.GamepadButton{input.GamepadSouth, input.GamepadEast, input.GamepadWest, input.GamepadNorth} {
		if !capture.Buttons[button] {
			t.Fatalf("skill modifier failed to capture button %d", button)
		}
	}
	if mode.bot.disabled {
		t.Fatal("controller callback failed")
	}
	readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 3, 300, 20080910))
	ctx.Input.EndFrame()
	pad.Axes[input.GamepadRightTrigger] = 0
	ctx.Input.SetGamepad(pad)
	assertNoBotTestPacket(t, server, func() error {
		capture = mode.HandleGamepadInput(ctx, 1.0/60)
		mode.updateBotInput(ctx, true)
		return mode.bot.tick()
	})
	if !capture.Buttons[input.GamepadSouth] {
		t.Fatal("releasing R2 let a held skill button become a click")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadSouth] = false
	pad.Buttons[input.GamepadEast] = true
	pad.Axes[input.GamepadRightTrigger] = 1
	ctx.Input.SetGamepad(pad)
	mode.pendingSkill = pendingSkillTarget{skill: skill}
	assertNoBotTestPacket(t, server, func() error {
		mode.HandleGamepadInput(ctx, 1.0/60)
		return nil
	})
	if mode.pendingSkill.skill.ID != skill.ID {
		t.Fatal("empty hotbar slot changed the armed skill")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadEast] = false
	pad.Buttons[input.GamepadNorth] = true
	ctx.Input.SetGamepad(pad)
	mode.HandleGamepadInput(ctx, 1.0/60)
	readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(selfSkill.ID, 3, ctx.Session.AccountID, 20080910))
	assertNoBotTestPacket(t, server, func() error { return nil })
	if mode.pendingSkill.skill.ID != skill.ID {
		t.Fatal("self skill also cast the previously armed targeted skill")
	}
}

func TestWASDKeyboardSkillTargetSurvivesWithoutGamepad(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
	mode := NewWorldMode()
	mode.pendingSkill = pendingSkillTarget{skill: session.Skill{ID: db.SkillACDouble, Type: skillTargetEnemy, Level: 3, Range: 9}}
	loadKeyboardTestBot(t, ctx, mode)
	ctx.Input.SetKeyCode(gpucontext.KeyTab, true)
	botKeyPressForTest(t, mode.bot, gpucontext.KeyTab)
	mode.HandleGamepadInput(ctx, 1.0/60)
	if mode.scriptHighlight.id != 300 {
		t.Fatal("idle controller callback cleared keyboard skill selection")
	}
}

type gamepadHoverTestUI struct {
	client.UIManager
	blocked bool
}

func (u *gamepadHoverTestUI) PointerBlocked(int, int) bool { return u.blocked }

func TestWASDGamepadUIClickDoesNotBecomeAnAttackWhenPointerLeavesWindow(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	conn, server := newBotTestConnection(t, 20080910)
	ctx.Network = conn
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
	ui := &gamepadHoverTestUI{blocked: true}
	ctx.UIManager = ui
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadFrame{ID: "test"}
	pad.Buttons[input.GamepadRightShoulder] = true
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	capture := mode.HandleGamepadInput(ctx, 1.0/60)
	if capture.Buttons[input.GamepadSouth] || mode.scriptHighlight.id != 300 {
		t.Fatal("press over UI was not left to the pointer")
	}
	// Pointer motion follows the early callback; the held press still belongs
	// to the UI, both in this frame and after dragging outside the window.
	ui.blocked = false
	for i := 0; i < 2; i++ {
		assertNoBotTestPacket(t, server, func() error {
			mode.updateBotInput(ctx, true)
			return mode.bot.tick()
		})
		ctx.Input.EndFrame()
		pad.Buttons[input.GamepadRightShoulder] = false
		ctx.Input.SetGamepad(pad)
		mode.HandleGamepadInput(ctx, 1.0/60)
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
	capture := mode.HandleGamepadInput(ctx, 0.02)
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
		capture = mode.HandleGamepadInput(ctx, 0.02)
		if !capture.Pointer || mode.camera.targetZoom() <= previousZoom || mode.camera.currentPitch() != pitch || mode.camera.yawOffset != 2 {
			t.Fatal("R2 + stick down did not exclusively zoom out, including with both triggers held")
		}
	}
	zoom := mode.camera.targetZoom()
	ctx.Input.EndFrame()
	mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
	mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	capture = mode.HandleGamepadInput(ctx, 0.02)
	if mode.ui.npcDialog.IsOpen() || !capture.Buttons[input.GamepadSouth] || mode.camera.yawOffset != 2 || mode.camera.targetZoom() != zoom {
		t.Fatal("NPC confirmation did not take priority over camera/gameplay")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadSouth] = false
	pad.Axes[input.GamepadRightY] = -1
	ctx.Input.SetGamepad(pad)
	mode.HandleGamepadInput(ctx, 0.02)
	if mode.camera.targetZoom() >= zoom {
		t.Fatal("R2 + stick up did not zoom in")
	}
	if mode.bot.disabled {
		t.Fatal("controller callback failed")
	}
}

func TestWASDGamepadMovementFollowsRotatedCamera(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	conn, server := newBotTestConnection(t, 20080910)
	ctx.Network = conn
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.GAT = flatWalkableGAT(64, 64)
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
	conn, server := newBotTestConnection(t, 20080910)
	ctx.Network = conn
	ctx.Config.Script.Path = "builtin:wasd"
	ctx.World = worldstate.New()
	ctx.World.GAT = flatWalkableGAT(64, 64)
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadFrame{ID: "test"}
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
	pad := input.GamepadFrame{ID: "test"}
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
	state.SetGamepad(input.GamepadFrame{})
	if err := bot.inputFrame(true); err != nil {
		t.Fatal(err)
	}
	want, _ = network.BuildWalkToXYPacketForClientDate(10, 21, 20080910)
	readBotTestPackets(t, server, want)
}
