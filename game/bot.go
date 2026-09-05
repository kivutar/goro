package game

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

const botTickInterval = 150 * time.Millisecond

type luaBot struct {
	path              string
	state             *lua.LState
	mode              *WorldMode
	nextTick          time.Time
	disabled          bool
	keyboardAvailable bool
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

func (m *WorldMode) updateBotInput(ctx client.Context, keyboardAvailable bool) {
	path := strings.TrimSpace(ctx.Config.Script.Path)
	if path == "" || m.bot == nil || m.bot.path != path || m.bot.disabled {
		return
	}
	if err := m.bot.inputFrame(keyboardAvailable); err != nil {
		glog.Warnf("lua script input failed path=%q: %v", m.bot.path, err)
		m.bot.close()
		m.bot.disabled = true
	}
}

func newLuaBot(ctx client.Context, mode *WorldMode, path string) (*luaBot, error) {
	bot := &luaBot{
		path:     path,
		state:    lua.NewState(),
		mode:     mode,
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

func (b *luaBot) inputFrame(keyboardAvailable bool) error {
	if b == nil || b.state == nil {
		return nil
	}
	b.keyboardAvailable = keyboardAvailable
	fn := b.state.GetGlobal("input")
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
		"players": func(L *lua.LState) int {
			L.Push(luaPlayerList(L, ctx, mode))
			return 1
		},
		"companions": func(L *lua.LState) int {
			L.Push(luaCompanionList(L, ctx, mode))
			return 1
		},
		"items": func(L *lua.LState) int {
			L.Push(luaItemList(L, ctx))
			return 1
		},
		"inventory": func(L *lua.LState) int {
			L.Push(luaInventoryList(L, ctx))
			return 1
		},
		"use_item": func(L *lua.LState) int {
			L.Push(lua.LBool(scriptUseItem(ctx, L.CheckInt(1))))
			return 1
		},
		"revive": func(L *lua.LState) int {
			L.Push(lua.LBool(scriptAutoRevive(ctx)))
			return 1
		},
		"message": func(L *lua.LState) int {
			L.Push(lua.LBool(scriptMessage(ctx, L.CheckString(1))))
			return 1
		},
		"walk": func(L *lua.LState) int {
			L.Push(lua.LBool(mode.scriptWalk(ctx, L.CheckInt(1), L.CheckInt(2))))
			return 1
		},
		"stop": func(L *lua.LState) int {
			L.Push(lua.LBool(mode.scriptStop(ctx)))
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
		"skill": func(L *lua.LState) int {
			id := uint32(L.CheckInt(1))
			skillArg := L.Get(2)
			if skillArg == lua.LNil {
				L.ArgError(2, "skill id or name expected")
				return 0
			}
			level := -1
			if L.GetTop() >= 3 && L.Get(3) != lua.LNil {
				level = L.CheckInt(3)
			}
			L.Push(lua.LBool(mode.scriptSkill(ctx, id, skillArg, level)))
			return 1
		},
		"pending_skill": func(L *lua.LState) int {
			L.Push(luaPendingSkill(L, ctx, mode))
			return 1
		},
		"use_pending_skill": func(L *lua.LState) int {
			id := luaOptionalActorID(L, 1)
			L.Push(lua.LBool(mode.scriptUsePendingSkill(ctx, id)))
			return 1
		},
		"highlight_actor": func(L *lua.LState) int {
			id := luaOptionalActorID(L, 1)
			L.Push(lua.LBool(mode.scriptHighlightActor(ctx, id)))
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
	registerLuaKeyboardAPI(b.state, api, ctx, b)
	b.state.SetGlobal("goro", api)
}

func (m *WorldMode) scriptAttack(ctx client.Context, id uint32) bool {
	if playerIsDead(ctx) || ctx.World == nil {
		return false
	}
	actor, ok := ctx.World.Actors[id]
	if !ok || !actorCanBeAttackClicked(ctx, actor) {
		return false
	}
	m.requestAttack(ctx, actor, "script")
	return true
}

func (m *WorldMode) scriptSkill(ctx client.Context, id uint32, skillArg lua.LValue, requestedLevel int) bool {
	if ctx.World == nil {
		return false
	}
	skill, ok := luaSkill(ctx, skillArg)
	if !ok {
		glog.Debugf("script skill unavailable target=%d skill=%s", id, skillArg.String())
		return false
	}
	skill = normalizeSessionSkillLevelCap(skill)
	if requestedLevel != -1 {
		if requestedLevel < 1 || requestedLevel > skill.Level {
			glog.Debugf("script skill invalid level skill=%d requested=%d learned=%d", skill.ID, requestedLevel, skill.Level)
			return false
		}
		if selectable, known := db.SkillLevelSelectable(skill.ID); known && !selectable && requestedLevel != skill.Level {
			glog.Debugf("script skill level not selectable skill=%d requested=%d learned=%d", skill.ID, requestedLevel, skill.Level)
			return false
		}
		skill.Level = requestedLevel
	}
	actor, ok, _ := actorForCombatID(ctx, id)
	_, dead := m.actorDeaths[id]
	if !ok || dead || !actorCanBeSkillTargeted(ctx, skill, actor) {
		glog.Debugf("script skill invalid target skill=%d target=%d", skill.ID, id)
		return false
	}
	if err := m.skills().UseTarget(ctx, skill, actor, "script"); err != nil {
		glog.Debugf("script skill failed skill=%d target=%d: %v", skill.ID, id, err)
		return false
	}
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
	return m.requestPickup(ctx, item, "script")
}

func scriptUseItem(ctx client.Context, index int) bool {
	if ctx.Session == nil || index <= 0 || index > int(^uint16(0)) {
		return false
	}
	item, ok := findSessionInventoryItem(ctx.Session, uint16(index))
	if !ok {
		return false
	}
	if err := gameui.UseInventoryItem(ctx, item); err != nil {
		glog.Debugf("script item use failed index=%d item=%d: %v", item.Index, item.ItemID, err)
		return false
	}
	return true
}

func scriptAutoRevive(ctx client.Context) bool {
	if err := client.RequestAutoRevive(ctx); err != nil {
		glog.Debugf("script auto-revive failed: %v", err)
		return false
	}
	return true
}

func scriptMessage(ctx client.Context, message string) bool {
	if err := client.SendChat(ctx, message); err != nil {
		glog.Debugf("script message failed: %v", err)
		return false
	}
	return true
}

func luaSkill(ctx client.Context, skillArg lua.LValue) (session.Skill, bool) {
	switch value := skillArg.(type) {
	case lua.LNumber:
		if value <= 0 || value > 65535 {
			return session.Skill{}, false
		}
		return skillByID(ctx.Session, uint16(value))
	case lua.LString:
		return luaSkillByName(ctx, string(value))
	default:
		return session.Skill{}, false
	}
}

func luaSkillByName(ctx client.Context, name string) (session.Skill, bool) {
	if ctx.Session == nil {
		return session.Skill{}, false
	}
	needle := normalizeLuaSkillName(name)
	if needle == "" {
		return session.Skill{}, false
	}
	for _, skill := range ctx.Session.Skills.List {
		if normalizeLuaSkillName(skill.Name) == needle || normalizeLuaSkillName(db.SkillResourceName[skill.ID]) == needle {
			return skill, true
		}
	}
	return session.Skill{}, false
}

func normalizeLuaSkillName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", " ")
	return strings.Join(strings.Fields(name), " ")
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
		row := luaActorTable(L, actor, playerX, playerY)
		row.RawSetString("object_type", lua.LNumber(actor.ObjectType))
		result.Append(row)
	}
	return result
}

func luaPlayerList(L *lua.LState, ctx client.Context, mode *WorldMode) *lua.LTable {
	result := L.NewTable()
	if ctx.World == nil {
		return result
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	ids := make([]int, 0, len(ctx.World.Actors))
	for id, actor := range ctx.World.Actors {
		if actorRepresentsPlayer(actor) && !isLocalActor(ctx, actor.ID) {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		actor := ctx.World.Actors[uint32(id)]
		row := luaActorTable(L, actor, playerX, playerY)
		member := luaPartyMember(ctx.Session, actor.ID)
		row.RawSetString("party_member", lua.LBool(member != nil))
		dead := false
		if mode != nil {
			_, dead = mode.actorDeaths[actor.ID]
		}
		if member != nil {
			row.RawSetString("hp", lua.LNumber(member.HP))
			row.RawSetString("max_hp", lua.LNumber(member.MaxHP))
			dead = dead || member.Dead
		} else {
			row.RawSetString("hp", lua.LNumber(0))
			row.RawSetString("max_hp", lua.LNumber(0))
		}
		row.RawSetString("dead", lua.LBool(dead))
		result.Append(row)
	}
	return result
}

func luaCompanionList(L *lua.LState, ctx client.Context, mode *WorldMode) *lua.LTable {
	result := L.NewTable()
	if ctx.World == nil {
		return result
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	ids := make([]int, 0, 2)
	for id, actor := range ctx.World.Actors {
		if actor.HasObjectType && (actor.ObjectType == actorObjectTypeHomunculus || actor.ObjectType == actorObjectTypeMercenary) {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		actor := ctx.World.Actors[uint32(id)]
		row := luaActorTable(L, actor, playerX, playerY)
		kind := "homunculus"
		if actor.ObjectType == actorObjectTypeMercenary {
			kind = "mercenary"
		}
		row.RawSetString("kind", lua.LString(kind))
		row.RawSetString("own", lua.LBool(luaCompanionOwnedByPlayer(ctx.Session, actor)))
		hp, maxHP, sp, maxSP := 0, 0, 0, 0
		if mode != nil {
			hp, maxHP, sp, maxSP, _ = mode.companionLife(ctx, actor.ID)
		}
		row.RawSetString("hp", lua.LNumber(hp))
		row.RawSetString("max_hp", lua.LNumber(maxHP))
		row.RawSetString("sp", lua.LNumber(sp))
		row.RawSetString("max_sp", lua.LNumber(maxSP))
		dead := false
		if mode != nil {
			_, dead = mode.actorDeaths[actor.ID]
		}
		row.RawSetString("dead", lua.LBool(dead))
		result.Append(row)
	}
	return result
}

func luaActorTable(L *lua.LState, actor worldstate.Actor, playerX, playerY int) *lua.LTable {
	row := L.NewTable()
	row.RawSetString("id", lua.LNumber(actor.ID))
	row.RawSetString("name", lua.LString(actor.Name))
	row.RawSetString("x", lua.LNumber(actor.X))
	row.RawSetString("y", lua.LNumber(actor.Y))
	row.RawSetString("job", lua.LNumber(actor.Job))
	row.RawSetString("distance", lua.LNumber(cellDistance(playerX, playerY, actor.X, actor.Y)))
	return row
}

func luaCompanionOwnedByPlayer(s *session.Session, actor worldstate.Actor) bool {
	if s == nil {
		return false
	}
	if actor.ObjectType == actorObjectTypeHomunculus {
		return actor.ID == s.Homunculus.ID
	}
	return actor.ObjectType == actorObjectTypeMercenary && actor.ID == s.Mercenary.ID
}

func luaPartyMember(s *session.Session, actorID uint32) *session.PartyMember {
	if s == nil || actorID == 0 {
		return nil
	}
	return findPartyMember(&s.Party, actorID)
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

func luaInventoryList(L *lua.LState, ctx client.Context) *lua.LTable {
	result := L.NewTable()
	if ctx.Session == nil {
		return result
	}
	items := append([]session.InventoryItem(nil), ctx.Session.Inventory.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	for _, item := range items {
		row := L.NewTable()
		row.RawSetString("index", lua.LNumber(item.Index))
		row.RawSetString("item_id", lua.LNumber(item.ItemID))
		row.RawSetString("amount", lua.LNumber(item.Amount))
		row.RawSetString("identified", lua.LBool(item.Identified))
		row.RawSetString("usable", lua.LBool(db.ItemTypeIsUsable(item.Type)))
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
		result.RawSetString("id", lua.LNumber(localSkillTarget(ctx)))
	}
	result.RawSetString("x", lua.LNumber(x))
	result.RawSetString("y", lua.LNumber(y))
	result.RawSetString("hp", lua.LNumber(hp))
	result.RawSetString("max_hp", lua.LNumber(maxHP))
	result.RawSetString("sp", lua.LNumber(sp))
	result.RawSetString("max_sp", lua.LNumber(maxSP))
	result.RawSetString("dead", lua.LBool(playerIsDead(ctx)))
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
