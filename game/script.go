package game

import (
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/scripts"
	lua "github.com/yuin/gopher-lua"
)

const botTickInterval = 150 * time.Millisecond

// The manager owns one script for the active mode. WorldMode borrows it to run
// gameplay callbacks at the appropriate points in its update.
type luaScript struct {
	path              string
	state             *lua.LState
	ctx               client.Context
	mode              *WorldMode
	ui                scriptUI
	nextTick          time.Time
	disabled          bool
	keyboardAvailable bool
	gamepadInput      bool
	gamepadCapture    input.GamepadCapture
	blockedButtons    [input.GamepadButtonCount]bool
}

type scriptUI interface {
	controlScriptUI(client.Context, string) bool
}

func (m *Manager) syncScript(ctx client.Context) {
	path := ctx.ScriptPath()
	if m.script != nil && m.script.path != path {
		m.closeScript()
	}
	if path == "" || path == "none" {
		return
	}
	if m.script == nil {
		script, err := newLuaScript(ctx, m.mode, path)
		if err != nil {
			glog.Warnf("lua script load failed path=%q: %v", path, err)
			script = &luaScript{path: path, disabled: true}
		}
		m.script = script
		if world, ok := m.mode.(*WorldMode); ok {
			world.bot = script
		}
	}
	m.script.ctx = ctx
	m.script.blockedButtons = m.scriptBlocked
}

func (m *Manager) closeScript() {
	if m.script != nil {
		m.script.close()
		m.script = nil
	}
	if world, ok := m.mode.(*WorldMode); ok {
		world.bot = nil
	}
}

func newLuaScript(ctx client.Context, mode Mode, path string) (*luaScript, error) {
	world, _ := mode.(*WorldMode)
	ui, _ := mode.(scriptUI)
	b := &luaScript{
		path: path, state: lua.NewState(), ctx: ctx, mode: world, ui: ui,
		nextTick: time.Now().Add(botTickInterval),
	}
	b.registerAPI(world)
	var err error
	if name, builtin := strings.CutPrefix(path, "builtin:"); builtin {
		var source []byte
		source, err = scripts.Builtin.ReadFile(name + ".lua")
		if err == nil {
			err = b.state.DoString(string(source))
		}
	} else {
		err = b.state.DoFile(path)
	}
	if err != nil {
		b.close()
		return nil, err
	}
	return b, nil
}

func (b *luaScript) worldContext() client.Context {
	if b.mode == nil {
		return client.Context{}
	}
	return b.ctx
}

func (b *luaScript) close() {
	if b == nil {
		return
	}
	if b.mode != nil {
		b.mode.clearScriptHighlight()
	}
	if b.state != nil {
		b.state.Close()
		b.state = nil
	}
}

func (b *luaScript) fail(callback string, err error) {
	glog.Warnf("lua script %s failed path=%q: %v", callback, b.path, err)
	b.close()
	b.disabled = true
}

func (b *luaScript) invoke(name string, args ...lua.LValue) error {
	if b == nil || b.disabled || b.state == nil {
		return nil
	}
	fn := b.state.GetGlobal(name)
	if fn == lua.LNil {
		return nil
	}
	return b.state.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, args...)
}

func (b *luaScript) tick() error { return b.invoke("tick") }

func (b *luaScript) inputFrame(available bool) error {
	b.keyboardAvailable = available
	return b.invoke("input")
}
