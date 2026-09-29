package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	lua "github.com/yuin/gopher-lua"
)

func registerLuaGamepadAPI(state *lua.LState, api *lua.LTable, ctx client.Context, bot *luaBot) {
	// The same gameplay focus policy as keyboard input: chat, forms, modal
	// dialogs and death suspend controls. Device metadata remains queryable.
	available := func() bool {
		return bot != nil && bot.keyboardAvailable && ctx.Input != nil && ctx.Input.GamepadConnected()
	}
	// The early gamepad callback also sees dialog input. Gameplay input() keeps
	// the existing focus filtering; available() still reports gameplay focus.
	readable := func() bool {
		return bot != nil && (bot.keyboardAvailable || bot.gamepadInput) && ctx.Input != nil
	}
	gamepad := state.NewTable()
	buttonQuery := func(query func(input.GamepadButton) bool, needsConnection bool) lua.LGFunction {
		return func(L *lua.LState) int {
			button, valid := input.GamepadButtonFromName(L.CheckString(1))
			allowed := readable()
			if needsConnection {
				allowed = allowed && ctx.Input.GamepadConnected()
			}
			L.Push(lua.LBool(valid && allowed && query(button)))
			return 1
		}
	}
	state.SetFuncs(gamepad, map[string]lua.LGFunction{
		"available": func(L *lua.LState) int { L.Push(lua.LBool(available())); return 1 },
		"connected": func(L *lua.LState) int { L.Push(lua.LBool(ctx.Input != nil && ctx.Input.GamepadConnected())); return 1 },
		"name": func(L *lua.LState) int {
			name := ""
			if ctx.Input != nil {
				name = ctx.Input.GamepadName()
			}
			L.Push(lua.LString(name))
			return 1
		},
		"axis": func(L *lua.LState) int {
			axis, valid := input.GamepadAxisFromName(L.CheckString(1))
			value := 0.0
			if valid && readable() && ctx.Input.GamepadConnected() {
				value = ctx.Input.GamepadValue(axis)
			}
			L.Push(lua.LNumber(value))
			return 1
		},
		"is_down":      buttonQuery(func(b input.GamepadButton) bool { return ctx.Input.GamepadDown(b) }, true),
		"was_pressed":  buttonQuery(func(b input.GamepadButton) bool { return ctx.Input.GamepadJustPressed(b) }, true),
		"was_released": buttonQuery(func(b input.GamepadButton) bool { return ctx.Input.GamepadJustReleased(b) }, false),
		"consume": func(L *lua.LState) int {
			button, valid := input.GamepadButtonFromName(L.CheckString(1))
			if valid && bot.gamepadInput {
				bot.gamepadCapture.Buttons[button] = true
			}
			return 0
		},
		"consume_pointer": func(L *lua.LState) int {
			if bot.gamepadInput {
				bot.gamepadCapture.Pointer = true
			}
			return 0
		},
	})
	api.RawSetString("gamepad", gamepad)
}
