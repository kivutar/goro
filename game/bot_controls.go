package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	lua "github.com/yuin/gopher-lua"
)

// The script claims controls before the renderer's pointer fallback.
func (b *luaScript) handleGamepad(ctx client.Context, dt float64) input.GamepadCapture {
	if b == nil || b.disabled || b.state == nil || b.path != ctx.ScriptPath() {
		return input.GamepadCapture{}
	}
	if b.mode != nil && b.mode.uiInputSuspended() {
		return input.GamepadCapture{Buttons: b.blockedButtons}
	}
	b.ctx = ctx
	previous := b.keyboardAvailable
	b.keyboardAvailable = b.mode != nil && !b.mode.ui.KeyboardShortcutsBlocked(ctx)
	b.gamepadInput = true
	b.gamepadCapture = input.GamepadCapture{Buttons: b.blockedButtons}
	if err := b.invoke("gamepad", lua.LNumber(dt)); err != nil {
		b.fail("gamepad", err)
	}
	b.gamepadInput = false
	b.keyboardAvailable = previous
	return b.gamepadCapture
}

func registerLuaControlsAPI(state *lua.LState, api *lua.LTable, bot *luaScript) {
	m := bot.mode
	allowed := func() bool { return m != nil && !m.uiInputSuspended() && !m.ui.KeyboardShortcutsBlocked(bot.ctx) }
	state.SetFuncs(api, map[string]lua.LGFunction{
		"use_shortcut": func(L *lua.LState) int {
			ctx := bot.ctx
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
			ctx := bot.ctx
			yaw, pitch := float64(L.CheckNumber(1)), float64(L.CheckNumber(2))
			if allowed() && !cameraRotationLockedForMap(ctx) && isFinite(yaw) && isFinite(pitch) {
				m.camera.Rotate(yaw)
				m.camera.Tilt(pitch)
			}
			return 0
		},
		"zoom_camera": func(L *lua.LState) int {
			ctx := bot.ctx
			delta := float64(L.CheckNumber(1))
			if allowed() && !cameraZoomLockedForMap(ctx) && isFinite(delta) {
				m.camera.ZoomByDelta(delta)
			}
			return 0
		},
		"camera_yaw": func(L *lua.LState) int {
			ctx := bot.ctx
			if m == nil {
				L.Push(lua.LNumber(0))
				return 1
			}
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
			ctx := bot.ctx
			open := m != nil && m.ui.npcDialog.IsOpen() && !m.uiInputSuspended() && m.ui.npcInputAvailable()
			if open && L.GetTop() != 0 {
				m.ui.npcDialog.Control(ctx, L.CheckString(1))
			}
			L.Push(lua.LBool(open))
			return 1
		},
		"pointer_over_ui": func(L *lua.LState) int {
			ctx := bot.ctx
			blocked := false
			if ui, ok := ctx.UIManager.(interface{ PointerBlocked(int, int) bool }); ok && ctx.Input != nil {
				blocked = ui.PointerBlocked(ctx.Input.MouseX, ctx.Input.MouseY)
			}
			L.Push(lua.LBool(blocked))
			return 1
		},
	})
}
