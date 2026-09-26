package game

import (
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	lua "github.com/yuin/gopher-lua"
)

// Called from world packet handling, on the same update thread as Lua ticks.
// The packet's sender ID determines identity; message text never selects it.
func (m *WorldMode) botChat(ctx client.Context, channel string, senderID uint32, text string) {
	path := strings.TrimSpace(ctx.Config.Script.Path)
	if path == "" || m.bot == nil || m.bot.path != path || m.bot.disabled || senderID == 0 || isLocalActor(ctx, senderID) {
		return
	}
	name, player := "", false
	if ctx.World != nil {
		if actor, ok := ctx.World.Actors[senderID]; ok && actorRepresentsPlayer(actor) {
			name, player = actor.Name, true
		}
	}
	if channel == "party" {
		if member := luaPartyMember(ctx.Session, senderID); member != nil {
			player = true
			if name == "" {
				name = member.Name
			}
		}
	}
	if !player {
		return
	}
	text = strings.TrimSpace(text)
	if name != "" {
		text = strings.TrimPrefix(text, name+" : ")
	} else if prefix, body, ok := strings.Cut(text, " : "); ok {
		// A visible actor can speak before its name response arrives.
		name, text = strings.TrimSpace(prefix), body
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	m.bot.ctx = ctx
	if err := m.bot.chat(channel, senderID, name, text); err != nil {
		glog.Warnf("lua script chat failed path=%q: %v", m.bot.path, err)
		m.bot.close()
		m.bot.disabled = true
	}
}

func (b *luaBot) chat(channel string, senderID uint32, name, text string) error {
	if b == nil || b.state == nil {
		return nil
	}
	fn := b.state.GetGlobal("chat")
	if fn == lua.LNil {
		return nil
	}
	message := b.state.NewTable()
	message.RawSetString("channel", lua.LString(channel))
	message.RawSetString("sender_id", lua.LNumber(senderID))
	message.RawSetString("sender_name", lua.LString(name))
	message.RawSetString("text", lua.LString(text))
	return b.state.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, message)
}
