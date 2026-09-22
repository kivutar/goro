package game

import (
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	lua "github.com/yuin/gopher-lua"
)

// Keep Lua locals and cached API functions across map changes. Only their
// backing world mode/context changes; scripts can reset map-local decisions.
func (m *WorldMode) botMapChanged(ctx client.Context) {
	b := m.bot
	if b == nil {
		return
	}
	if b.path != strings.TrimSpace(ctx.Config.Script.Path) {
		b.close()
		m.bot = nil
		return
	}
	b.ctx, b.mode = ctx, m
	b.nextTick = time.Time{}
	b.keyboardAvailable = false
	if b.disabled || b.state == nil {
		return
	}
	fn := b.state.GetGlobal("map_changed")
	if fn == lua.LNil {
		return
	}
	if err := b.state.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}); err != nil {
		glog.Warnf("lua script map_changed failed path=%q: %v", b.path, err)
		b.close()
		b.disabled = true
	}
}
