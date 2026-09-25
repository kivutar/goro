package game

import (
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	lua "github.com/yuin/gopher-lua"
)

func (m *WorldMode) botKeyPress(ctx client.Context, code input.KeyCode) {
	path := strings.TrimSpace(ctx.Config.Script.Path)
	if path == "" || m.bot == nil || m.bot.path != path || m.bot.disabled {
		return
	}
	m.bot.ctx = ctx
	if err := m.bot.keyPress(code); err != nil {
		glog.Warnf("lua script keypress failed path=%q: %v", m.bot.path, err)
		m.bot.close()
		m.bot.disabled = true
	}
}

func (b *luaBot) keyPress(code input.KeyCode) error {
	if b == nil || b.state == nil {
		return nil
	}
	fn := b.state.GetGlobal("keypress")
	if fn == lua.LNil {
		return nil
	}
	available := b.keyboardAvailable
	b.keyboardAvailable = true
	defer func() { b.keyboardAvailable = available }()
	return b.state.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, lua.LString(input.KeyCodeName(code)))
}

func registerLuaKeyboardAPI(state *lua.LState, api *lua.LTable, bot *luaBot) {
	available := func() bool {
		return bot != nil && bot.keyboardAvailable && bot.ctx.Input != nil
	}
	keyboard := state.NewTable()
	state.SetFuncs(keyboard, map[string]lua.LGFunction{
		"available": func(L *lua.LState) int {
			L.Push(lua.LBool(available()))
			return 1
		},
		"is_down": func(L *lua.LState) int {
			code, ok := input.KeyCodeFromName(L.CheckString(1))
			L.Push(lua.LBool(ok && available() && bot.ctx.Input.KeyCodeDown(code)))
			return 1
		},
		"was_pressed": func(L *lua.LState) int {
			code, ok := input.KeyCodeFromName(L.CheckString(1))
			L.Push(lua.LBool(ok && available() && bot.ctx.Input.KeyCodeJustPressed(code)))
			return 1
		},
		"was_released": func(L *lua.LState) int {
			code, ok := input.KeyCodeFromName(L.CheckString(1))
			L.Push(lua.LBool(ok && available() && bot.ctx.Input.KeyCodeJustReleased(code)))
			return 1
		},
		"consume_press": func(L *lua.LState) int {
			code, ok := input.KeyCodeFromName(L.CheckString(1))
			L.Push(lua.LBool(ok && available() && bot.ctx.Input.ConsumeKeyCodePress(code)))
			return 1
		},
		"text": func(L *lua.LState) int {
			if !available() {
				L.Push(lua.LString(""))
			} else {
				L.Push(lua.LString(bot.ctx.Input.TextInput()))
			}
			return 1
		},
	})
	api.RawSetString("keyboard", keyboard)
}
