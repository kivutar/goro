package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	lua "github.com/yuin/gopher-lua"
)

func TestScriptLifetimeAcrossLoginAndWorld(t *testing.T) {
	ctx := wasdControlsTestContext(t)
	ctx.Config.Headless = true
	ctx.World.MapName = ""
	ctx.Network = network.NewClient(20080910, false)
	t.Cleanup(ctx.Network.Close)
	ctx.Config.Script.Path = filepath.Join(t.TempDir(), "controls.lua")
	if err := os.WriteFile(ctx.Config.Script.Path, []byte(`
inputs, ticks = 0, 0
function input() inputs = inputs + 1 end
function tick() ticks = ticks + 1 end
function gamepad()
    pressed = goro.gamepad.was_pressed("south")
    held = goro.gamepad.is_down("south")
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
`), 0600); err != nil {
		t.Fatal(err)
	}
	m := NewManager(ctx, NewLoginMode())
	t.Cleanup(m.Close)
	if err := m.Update(); err != nil {
		t.Fatal(err)
	}
	loginScript := m.script
	if loginScript == nil || loginScript.disabled {
		t.Fatal("script did not load safely in login")
	}
	assertLuaGlobalNumber(t, loginScript.state, "inputs", 1)
	assertLuaGlobalNumber(t, loginScript.state, "ticks", 0)
	// A new context must also reach the Lua bindings, rather than the input
	// pointer captured at registration time.
	ctx.Input = input.NewState()
	pad := input.GamepadFrame{ID: "test"}
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	m.UpdateContext(ctx)
	m.HandleGamepadInput(ctx, 0.02)
	if loginScript.state.GetGlobal("pressed") != lua.LTrue {
		t.Fatal("script kept its original input context")
	}
	world := NewWorldMode()
	m.enter(world)
	world.mapFade = mapFadeState{} // Finish map loading before exercising controls.
	if loginScript.state != nil {
		t.Fatal("login Lua state survived transition")
	}
	ctx.Input.EndFrame()
	capture := m.HandleGamepadInput(ctx, 0.02)
	if world.bot != m.script || m.script.disabled {
		t.Fatal("world did not bind a fresh script")
	}
	if !capture.Buttons[input.GamepadSouth] || m.script.state.GetGlobal("held") != lua.LFalse {
		t.Fatal("login confirmation leaked into world controls")
	}
	world.bot.nextTick = time.Time{}
	world.updateBot(ctx, time.Now())
	assertLuaGlobalNumber(t, world.bot.state, "ticks", 1)
	ctx.Input.SetGamepad(input.GamepadFrame{ID: "test"})
	m.HandleGamepadInput(ctx, 0.02)
	ctx.Input.EndFrame()
	ctx.Input.SetGamepad(pad)
	m.HandleGamepadInput(ctx, 0.02)
	if m.script.state.GetGlobal("pressed") != lua.LTrue {
		t.Fatal("new press remained blocked after release")
	}
	worldScript := m.script
	m.enter(NewLoginMode())
	if worldScript.state != nil || world.bot != nil {
		t.Fatal("world retained the old script")
	}
	m.syncScript(ctx)
	if m.script.disabled {
		t.Fatal("returning to login failed to load script")
	}
	last := m.script
	m.Close()
	if last.state != nil || m.script != nil {
		t.Fatal("closing manager leaked the script")
	}
}

func loginGamepadTestContext() client.Context {
	return client.Context{
		Input: input.NewState(), Session: session.New(),
		Resources: loginTestResources(res.Connection{Display: "Local"}, res.Connection{Display: "LAN"}),
		UIManager: &loginTestUIManager{}, ScreenW: 1280, ScreenH: 720,
	}
}

func TestWASDLoginMenusAndModalPriority(t *testing.T) {
	ctx := loginGamepadTestContext()
	ctx.Config.Script.Path = "builtin:wasd"
	mode := NewLoginMode()
	mode.updateAccountWindow(ctx)
	m := &Manager{ctx: ctx, mode: mode}
	t.Cleanup(m.Close)
	press := func(button input.GamepadButton) {
		t.Helper()
		ctx.Input.SetGamepad(input.GamepadFrame{ID: "test"})
		m.HandleGamepadInput(ctx, 0.02)
		ctx.Input.EndFrame()
		pad := input.GamepadFrame{ID: "test"}
		pad.Buttons[button] = true
		ctx.Input.SetGamepad(pad)
		capture := m.HandleGamepadInput(ctx, 0.02)
		if m.script.disabled {
			t.Fatal("login script failed")
		}
		if !capture.Buttons[input.GamepadSouth] || !capture.Buttons[input.GamepadEast] {
			t.Fatal("menu buttons leaked into pointer fallback")
		}
		ctx.Input.EndFrame()
	}
	press(input.GamepadDown)
	if mode.serviceWindow.SelectedIndex() != 1 {
		t.Fatal("D-pad did not select LAN")
	}
	press(input.GamepadSouth)
	if mode.selectedLoginServer != 1 || mode.loginWindow == nil || mode.serviceWindow != nil {
		t.Fatal("South did not open credentials for LAN")
	}
	press(input.GamepadEast)
	if mode.serviceWindow == nil || mode.loginWindow != nil {
		t.Fatal("East did not return to servers")
	}
	press(input.GamepadEast)
	if !mode.quitConfirm.IsOpen() {
		t.Fatal("East did not open exit confirmation")
	}
	selected := mode.serviceWindow.SelectedIndex()
	press(input.GamepadDown)
	if mode.serviceWindow.SelectedIndex() != selected {
		t.Fatal("modal navigation changed server behind it")
	}
	press(input.GamepadEast)
	if mode.quitConfirm.IsOpen() || mode.serviceWindow == nil {
		t.Fatal("cancel leaked through exit dialog")
	}
	mode.hideServiceWindow(ctx)
	mode.phase = loginPhaseCharacter
	mode.selectedSlot = 0
	press(input.GamepadRight)
	if mode.selectedSlot != 1 {
		t.Fatal("D-pad did not navigate character slots")
	}
	press(input.GamepadSouth)
	if mode.fade.target != loginPhaseCreate || mode.create.slot != 1 {
		t.Fatal("South did not activate empty character slot")
	}
	press(input.GamepadRight)
	if mode.selectedSlot != 1 {
		t.Fatal("input changed character selection during fade")
	}
}

func TestWASDLoginRetainsPointerControls(t *testing.T) {
	ctx := loginGamepadTestContext()
	ctx.Config.Script.Path = "builtin:wasd"
	mode := NewLoginMode()
	mode.updateAccountWindow(ctx)
	m := &Manager{ctx: ctx, mode: mode}
	t.Cleanup(m.Close)
	pad := input.GamepadFrame{ID: "test"}
	pad.Axes[input.GamepadRightX] = 1
	ctx.Input.SetGamepad(pad)
	m.HandleGamepadInput(ctx, 0.02)
	ctx.Input.EndFrame()
	pad.Axes[input.GamepadRightX] = 0
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	capture := m.HandleGamepadInput(ctx, 0.02)
	if capture.Pointer || capture.Buttons[input.GamepadSouth] || mode.serviceWindow == nil {
		t.Fatal("moving the pointer did not restore normal UI clicks")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadSouth] = false
	pad.Buttons[input.GamepadDown] = true
	ctx.Input.SetGamepad(pad)
	capture = m.HandleGamepadInput(ctx, 0.02)
	if !capture.Buttons[input.GamepadSouth] || mode.serviceWindow.SelectedIndex() != 1 {
		t.Fatal("D-pad did not restore menu navigation")
	}
	mode.hideServiceWindow(ctx)
	mode.phase = loginPhaseCreate
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadDown] = false
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	capture = m.HandleGamepadInput(ctx, 0.02)
	if capture.Buttons[input.GamepadSouth] || capture.Pointer || m.script.disabled {
		t.Fatal("character creation lost its pointer controls")
	}
}
