package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	lua "github.com/yuin/gopher-lua"
)

// HandleGamepadInput runs before the renderer turns unclaimed controls into
// mouse input, so a skill chord cannot also click a window or walk on the map.
func (m *WorldMode) HandleGamepadInput(ctx client.Context, dt float64) input.GamepadCapture {
	b := m.bot
	if b == nil || b.disabled || b.state == nil || b.path != ctx.ScriptPath() || m.uiInputSuspended() {
		return input.GamepadCapture{}
	}
	fn := b.state.GetGlobal("gamepad")
	if fn == lua.LNil {
		return input.GamepadCapture{}
	}
	previous := b.keyboardAvailable
	b.keyboardAvailable = !m.ui.KeyboardShortcutsBlocked(ctx)
	b.gamepadInput = true
	b.gamepadCapture = input.GamepadCapture{}
	err := b.state.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, lua.LNumber(dt))
	b.gamepadInput = false
	b.keyboardAvailable = previous
	if err != nil {
		glog.Warnf("lua script gamepad failed path=%q: %v", b.path, err)
		b.close()
		b.disabled = true
	}
	return b.gamepadCapture
}

func registerLuaControlsAPI(state *lua.LState, api *lua.LTable, ctx client.Context, bot *luaBot) {
	m := bot.mode
	allowed := func() bool { return m != nil && !m.uiInputSuspended() && !m.ui.KeyboardShortcutsBlocked(ctx) }
	state.SetFuncs(api, map[string]lua.LGFunction{
		"use_shortcut": func(L *lua.LState) int {
			slot := L.CheckInt(1)
			var skillID uint16
			var used bool
			if allowed() {
				skillID, used = m.ui.shortcutBar.ActivateSlot(ctx, m, slot)
			}
			L.Push(lua.LBool(used))
			L.Push(lua.LNumber(skillID))
			return 2
		},
		"rotate_camera": func(L *lua.LState) int {
			yaw, pitch := float64(L.CheckNumber(1)), float64(L.CheckNumber(2))
			if allowed() && !cameraRotationLockedForMap(ctx) && isFinite(yaw) && isFinite(pitch) {
				m.camera.Rotate(yaw)
				m.camera.Tilt(pitch)
			}
			return 0
		},
		"zoom_camera": func(L *lua.LState) int {
			delta := float64(L.CheckNumber(1))
			if allowed() && !cameraZoomLockedForMap(ctx) && isFinite(delta) {
				m.camera.ZoomByDelta(delta)
			}
			return 0
		},
		"camera_yaw": func(L *lua.LState) int {
			yaw := cameraYawForMap(ctx)
			if !cameraRotationLockedForMap(ctx) {
				yaw += m.camera.yawOffset
			}
			L.Push(lua.LNumber(yaw))
			return 1
		},
		"cancel_skill": func(L *lua.LState) int {
			if allowed() {
				m.skills().Cancel("script")
			}
			return 0
		},
		"npc_dialog": func(L *lua.LState) int {
			open := m.ui.npcDialog.IsOpen() && !m.uiInputSuspended() && !m.ui.interactionModalOpen() && !m.ui.escapeMenu.IsOpen()
			if open && L.GetTop() != 0 {
				m.ui.npcDialog.Control(ctx, L.CheckString(1))
			}
			L.Push(lua.LBool(open))
			return 1
		},
		"pointer_over_ui": func(L *lua.LState) int {
			blocked := false
			if ui, ok := ctx.UIManager.(interface{ PointerBlocked(int, int) bool }); ok && ctx.Input != nil {
				blocked = ui.PointerBlocked(ctx.Input.MouseX, ctx.Input.MouseY)
			}
			L.Push(lua.LBool(blocked))
			return 1
		},
	})
}
