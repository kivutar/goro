package game

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	lua "github.com/yuin/gopher-lua"
)

const botTickInterval = 150 * time.Millisecond

type luaBot struct {
	path     string
	state    *lua.LState
	nextTick time.Time
	disabled bool
}

func (m *WorldMode) updateBot(ctx client.Context, now time.Time) {
	path := strings.TrimSpace(ctx.Config.Script.Path)
	if path == "" {
		if m.bot != nil {
			m.bot.close()
			m.bot = nil
		}
		return
	}
	if m.bot == nil || m.bot.path != path {
		if m.bot != nil {
			m.bot.close()
		}
		bot, err := newLuaBot(ctx, m, path)
		if err != nil {
			glog.Warnf("lua script load failed path=%q: %v", path, err)
			m.bot = &luaBot{path: path, disabled: true}
			return
		}
		m.bot = bot
		glog.Debugf("lua script loaded path=%q", path)
	}
	if m.bot.disabled || now.Before(m.bot.nextTick) {
		return
	}
	m.bot.nextTick = now.Add(botTickInterval)
	if err := m.bot.tick(); err != nil {
		glog.Warnf("lua script tick failed path=%q: %v", m.bot.path, err)
		m.bot.close()
		m.bot.disabled = true
	}
}

func newLuaBot(ctx client.Context, mode *WorldMode, path string) (*luaBot, error) {
	bot := &luaBot{
		path:     path,
		state:    lua.NewState(),
		nextTick: time.Now().Add(botTickInterval),
	}
	bot.registerAPI(ctx, mode)
	if err := bot.state.DoFile(path); err != nil {
		bot.close()
		return nil, err
	}
	return bot, nil
}

func (b *luaBot) close() {
	if b == nil || b.state == nil {
		return
	}
	b.state.Close()
	b.state = nil
}

func (b *luaBot) tick() error {
	if b == nil || b.state == nil {
		return nil
	}
	fn := b.state.GetGlobal("tick")
	if fn == lua.LNil {
		return nil
	}
	return b.state.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true})
}

func (b *luaBot) registerAPI(ctx client.Context, mode *WorldMode) {
	api := b.state.NewTable()
	b.state.SetFuncs(api, map[string]lua.LGFunction{
		"enemies": func(L *lua.LState) int {
			L.Push(luaEnemyList(L, ctx, mode.actorDeaths))
			return 1
		},
		"items": func(L *lua.LState) int {
			L.Push(luaItemList(L, ctx))
			return 1
		},
		"attack": func(L *lua.LState) int {
			id := uint32(L.CheckInt(1))
			L.Push(lua.LBool(mode.scriptAttack(ctx, id)))
			return 1
		},
		"target": func(L *lua.LState) int {
			id := uint32(L.CheckInt(1))
			L.Push(lua.LBool(mode.scriptAttack(ctx, id)))
			return 1
		},
		"loot": func(L *lua.LState) int {
			id := uint32(L.CheckInt(1))
			L.Push(lua.LBool(mode.scriptLoot(ctx, id)))
			return 1
		},
		"hp": func(L *lua.LState) int {
			hp, maxHP := scriptHP(ctx)
			L.Push(lua.LNumber(hp))
			L.Push(lua.LNumber(maxHP))
			return 2
		},
		"sp": func(L *lua.LState) int {
			sp, maxSP := scriptSP(ctx)
			L.Push(lua.LNumber(sp))
			L.Push(lua.LNumber(maxSP))
			return 2
		},
		"player": func(L *lua.LState) int {
			L.Push(luaPlayerTable(L, ctx))
			return 1
		},
	})
	b.state.SetGlobal("goro", api)
}

func (m *WorldMode) scriptAttack(ctx client.Context, id uint32) bool {
	if ctx.World == nil {
		return false
	}
	actor, ok := ctx.World.Actors[id]
	if !ok || !actorCanBeAttackClicked(ctx, actor) {
		return false
	}
	m.requestAttack(ctx, actor, "script")
	return true
}

func (m *WorldMode) scriptLoot(ctx client.Context, id uint32) bool {
	if ctx.World == nil {
		return false
	}
	item, ok := ctx.World.Items[id]
	if !ok {
		return false
	}
	m.clearLockedAttack()
	m.clearAttackFocus()
	m.requestPickup(ctx, item, "script")
	return true
}

func luaEnemyList(L *lua.LState, ctx client.Context, actorDeaths map[uint32]time.Time) *lua.LTable {
	result := L.NewTable()
	if ctx.World == nil {
		return result
	}
	now := time.Now()
	playerX, playerY := currentPlayerCell(ctx, now)
	ids := make([]int, 0, len(ctx.World.Actors))
	for id, actor := range ctx.World.Actors {
		if _, dead := actorDeaths[id]; dead {
			continue
		}
		if actorCanBeAttackClicked(ctx, actor) {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		actor := ctx.World.Actors[uint32(id)]
		row := L.NewTable()
		row.RawSetString("id", lua.LNumber(actor.ID))
		row.RawSetString("name", lua.LString(actor.Name))
		row.RawSetString("x", lua.LNumber(actor.X))
		row.RawSetString("y", lua.LNumber(actor.Y))
		row.RawSetString("job", lua.LNumber(actor.Job))
		row.RawSetString("object_type", lua.LNumber(actor.ObjectType))
		row.RawSetString("distance", lua.LNumber(cellDistance(playerX, playerY, actor.X, actor.Y)))
		result.Append(row)
	}
	return result
}

func luaItemList(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	if ctx.World == nil {
		return result
	}
	now := time.Now()
	playerX, playerY := currentPlayerCell(ctx, now)
	ids := make([]int, 0, len(ctx.World.Items))
	for id := range ctx.World.Items {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	for _, id := range ids {
		item := ctx.World.Items[uint32(id)]
		row := L.NewTable()
		row.RawSetString("id", lua.LNumber(item.ID))
		row.RawSetString("item_id", lua.LNumber(item.ItemID))
		row.RawSetString("amount", lua.LNumber(item.Amount))
		row.RawSetString("x", lua.LNumber(item.X))
		row.RawSetString("y", lua.LNumber(item.Y))
		row.RawSetString("identified", lua.LBool(item.Identified))
		row.RawSetString("distance", lua.LNumber(cellDistance(playerX, playerY, item.X, item.Y)))
		result.Append(row)
	}
	return result
}

func luaPlayerTable(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	now := time.Now()
	x, y := currentPlayerCell(ctx, now)
	hp, maxHP := scriptHP(ctx)
	sp, maxSP := scriptSP(ctx)
	if ctx.Session != nil {
		result.RawSetString("id", lua.LNumber(ctx.Session.CharID))
	}
	result.RawSetString("x", lua.LNumber(x))
	result.RawSetString("y", lua.LNumber(y))
	result.RawSetString("hp", lua.LNumber(hp))
	result.RawSetString("max_hp", lua.LNumber(maxHP))
	result.RawSetString("sp", lua.LNumber(sp))
	result.RawSetString("max_sp", lua.LNumber(maxSP))
	return result
}

func scriptHP(ctx client.Context) (int, int) {
	if ctx.Session == nil {
		return 0, 0
	}
	return ctx.Session.Vitals.HP, ctx.Session.Vitals.MaxHP
}

func scriptSP(ctx client.Context) (int, int) {
	if ctx.Session == nil {
		return 0, 0
	}
	return ctx.Session.Vitals.SP, ctx.Session.Vitals.MaxSP
}

func cellDistance(ax, ay, bx, by int) float64 {
	return math.Hypot(float64(bx-ax), float64(by-ay))
}
