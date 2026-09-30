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
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestWASDGamepadMovementAttackAndLoot(t *testing.T) {
	for _, name := range []string{"stick", "dpad", "keyboard_and_stick", "attack", "nearest_attack", "loot"} {
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
			mode.HandleGamepadInput(ctx, 1.0/60)
			mode.updateBotInput(ctx, true)
			if err := mode.bot.tick(); err != nil {
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
				conn, server := newBotTestConnection(t, 20080910)
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
					mode.HandleGamepadInput(ctx, 1.0/60)
					mode.updateBotInput(ctx, true)
					if err := mode.bot.tick(); err != nil {
						t.Fatal(err)
					}
					ctx.Input.EndFrame()
					if mode.bot.disabled {
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

func TestWASDGamepadHealTargetsRespectNoShift(t *testing.T) {
	for _, override := range []string{"none", "shift", "noshift"} {
		t.Run(override, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			conn, server := newBotTestConnection(t, 20080910)
			ctx.Network = conn
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			ctx.Session.AccountID, ctx.Session.CharID = 100, 200
			ctx.World.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
			ctx.World.Actors[100] = ctx.World.Player // Local actor aliases must not duplicate self.
			ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
			ctx.World.Actors[301] = worldstate.Actor{ID: 301, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
			ctx.World.Actors[302] = worldstate.Actor{ID: 302, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true, EffectState: db.EffectStateHide}
			ctx.World.Actors[303] = worldstate.Actor{ID: 303, X: 11, Y: 20, Job: actorJobHiddenWarpNPC}
			skill := session.Skill{ID: db.SkillALHeal, Type: skillTargetFriend, Level: 3, Range: 9}
			ctx.Session.Skills.List = []session.Skill{skill}
			ctx.Session.Hotkeys = session.Hotkeys{Loaded: true, Version: 1, Slots: []session.HotkeySlot{
				{Type: network.HotkeyTypeSkill, ID: uint32(skill.ID), Level: 3},
			}}
			mode := NewWorldMode()
			mode.actorDeaths = map[uint32]time.Time{301: time.Now()}
			loadKeyboardTestBot(t, ctx, mode)
			pad := input.GamepadFrame{ID: "test"}
			pad.Axes[input.GamepadRightTrigger] = 1
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			mode.HandleGamepadInput(ctx, 1.0/60)
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
			mode.HandleGamepadInput(ctx, 1.0/60)
			want := ctx.Session.AccountID
			if override == "noshift" {
				want = 300
			}
			if mode.bot.disabled || mode.scriptHighlight.id != want {
				t.Fatalf("Heal target = %d, want %d", mode.scriptHighlight.id, want)
			}
			ctx.Input.EndFrame()
			pad.Buttons[input.GamepadRightShoulder] = false
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			mode.HandleGamepadInput(ctx, 1.0/60)
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
	ctx.World.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
	ctx.World.Actors[301] = worldstate.Actor{ID: 301, X: 11, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
	ctx.World.Actors[302] = worldstate.Actor{ID: 302, X: 10, Y: 21, ObjectType: actorObjectTypeMob, HasObjectType: true}
	ctx.World.Actors[303] = worldstate.Actor{ID: 303, X: 10, Y: 20, ObjectType: actorObjectTypeMob, HasObjectType: true}
	mode := NewWorldMode()
	mode.actorDeaths = map[uint32]time.Time{303: time.Now()}
	loadKeyboardTestBot(t, ctx, mode)
	pad := input.GamepadFrame{ID: "test"}
	press := func(button input.GamepadButton, want uint32) {
		t.Helper()
		pad.Buttons[button] = true
		ctx.Input.SetGamepad(pad)
		mode.HandleGamepadInput(ctx, 1.0/60)
		if mode.bot.disabled || mode.scriptHighlight.id != want {
			t.Fatalf("target = %d, want %d (script disabled: %v)", mode.scriptHighlight.id, want, mode.bot.disabled)
		}
		ctx.Input.EndFrame()
		pad.Buttons[button] = false
		ctx.Input.SetGamepad(pad)
		mode.HandleGamepadInput(ctx, 1.0/60)
		ctx.Input.EndFrame()
	}
	// Nearest first, ID breaks distance ties; both directions wrap.
	for _, want := range []uint32{301, 302, 300, 301} {
		press(input.GamepadRightShoulder, want)
	}
	press(input.GamepadLeftShoulder, 300)

	mode.pendingSkill = pendingSkillTarget{skill: session.Skill{ID: db.SkillALHeal, Type: skillTargetFriend, Level: 1}}
	press(input.GamepadRightShoulder, ctx.Session.AccountID)
	mode.pendingSkill = pendingSkillTarget{}
	mode.HandleGamepadInput(ctx, 1.0/60)
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
			pad.Buttons[input.GamepadRightShoulder] = name == "selected"
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			capture := mode.HandleGamepadInput(ctx, 1.0/60)
			if capture.Buttons[input.GamepadSouth] {
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

func TestWASDGamepadNPCDialogYieldsToDisconnect(t *testing.T) {
	for name, button := range map[string]input.GamepadButton{"south": input.GamepadSouth, "east": input.GamepadEast} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
			mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
			openDisconnectDialog(ctx, &mode.ui.disconnectDialog, "Disconnected")
			ctx.Input.EndFrame()
			pad := input.GamepadFrame{ID: "test"}
			pad.Buttons[button] = true
			ctx.Input.SetGamepad(pad)
			capture := mode.HandleGamepadInput(ctx, 1.0/60)
			if capture.Buttons[input.GamepadSouth] || capture.Buttons[input.GamepadEast] || capture.Pointer {
				t.Fatal("NPC captured pointer controls belonging to the disconnect alert")
			}
			if !mode.ui.npcDialog.IsOpen() || !mode.ui.disconnectDialog.IsOpen() {
				t.Fatal("controller acted on a dialog instead of leaving input to the pointer")
			}

			// Removing the alert restores normal NPC confirmation/cancellation.
			mode.ui.disconnectDialog.Close(ctx)
			ctx.Input.EndFrame()
			pad.Buttons[button] = false
			ctx.Input.SetGamepad(pad)
			mode.HandleGamepadInput(ctx, 1.0/60)
			ctx.Input.EndFrame()
			pad.Buttons[button] = true
			ctx.Input.SetGamepad(pad)
			capture = mode.HandleGamepadInput(ctx, 1.0/60)
			if !capture.Buttons[button] || mode.ui.npcDialog.IsOpen() {
				t.Fatal("NPC controls did not resume after the alert closed")
			}
			if mode.bot.disabled {
				t.Fatal("controller callback failed")
			}
		})
	}
}

func TestNPCDialogModalPriorityForKeyboardAndGamepad(t *testing.T) {
	for _, name := range []string{"friend", "trade", "party", "pet", "homunculus", "mercenary"} {
		t.Run(name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Network, _ = newBotTestConnection(t, 20080910)
			ctx.Config.Script.Path = "builtin:wasd"
			ctx.World = worldstate.New()
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
			mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
			mode.ui.npcDialog.Update(ctx) // Publish before testing keyboard dispatch.
			modal := map[string]*gameui.ConfirmModal{
				"friend": &mode.ui.friendRequest, "trade": &mode.ui.tradeRequest,
				"party": &mode.ui.partyInfo, "pet": &mode.ui.petConfirm,
				"homunculus": &mode.ui.homunculusConfirm, "mercenary": &mode.ui.mercenaryConfirm,
			}[name]
			modal.OpenAlert(ctx, name, "Confirm", nil)
			ctx.Input.EndFrame()
			pad := input.GamepadFrame{ID: "test"}
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			capture := mode.HandleGamepadInput(ctx, 1.0/60)
			if capture.Buttons[input.GamepadSouth] || capture.Buttons[input.GamepadEast] || !mode.ui.npcDialog.IsOpen() {
				t.Fatal("NPC stole controller input from the modal")
			}
			ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if modal.IsOpen() || !mode.ui.npcDialog.IsOpen() {
				t.Fatal("Enter did not confirm only the higher-priority modal")
			}
			// An ordinary window must not become a blanket blocker for NPCs.
			mode.ui.settingsWindow.OpenWindow(ctx)
			ctx.Input.EndFrame()
			ctx.Input.SetKeyCode(gpucontext.KeyEnter, false)
			pad.Buttons[input.GamepadSouth] = false
			ctx.Input.SetGamepad(pad)
			mode.HandleGamepadInput(ctx, 1.0/60)
			ctx.Input.EndFrame()
			pad.Buttons[input.GamepadSouth] = true
			ctx.Input.SetGamepad(pad)
			capture = mode.HandleGamepadInput(ctx, 1.0/60)
			if !capture.Buttons[input.GamepadSouth] || mode.ui.npcDialog.IsOpen() || mode.bot.disabled {
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
	mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
	mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
	mode.ui.npcDialog.Update(ctx)
	// This modal's Update runs after NPC dispatch, so ordering alone cannot
	// give it priority. The shared eligibility check must protect it too.
	mode.ui.guildMemberPrompt.Open(ctx, "Guild", "Reason", "", 40)
	ctx.Input.EndFrame()
	ctx.Input.SetKeyCode(gpucontext.KeyEscape, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.guildMemberPrompt.IsOpen() || !mode.ui.npcDialog.IsOpen() {
		t.Fatal("Escape did not cancel only the guild prompt above the NPC")
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
