package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

func TestCompanionAILoadsGravityGlobals(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "AI"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := `
function AI(id)
	local owner = GetV(0, id)
	local x, y = GetV(1, id)
	local range = GetV(4, id)
	local hp = GetV(8, id)
	local maxhp = GetV(10, id)
	local msg = GetMsg(id)
	local actors = GetActors()
	seen_id = id
	seen_owner = owner
	seen_x = x
	seen_y = y
	seen_range = range
	seen_hp = hp
	seen_maxhp = maxhp
	seen_msg_cmd = msg[1]
	seen_msg_target = msg[2]
	seen_actor_count = #actors
	seen_monster = IsMonster(400)
end
`
	if err := os.WriteFile(filepath.Join(root, "AI", "AI.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	sess := session.New()
	sess.AccountID = 100
	sess.CharID = 200
	sess.Homunculus = session.Companion{
		ID:          300,
		Active:      true,
		HP:          123,
		MaxHP:       456,
		SP:          12,
		MaxSP:       34,
		AttackRange: 7,
	}
	world := worldstate.New()
	world.Actors[300] = worldstate.Actor{ID: 300, X: 10, Y: 20, Job: 6001, HasObjectType: true, ObjectType: actorObjectTypeHomunculus, AttackRange: 7}
	world.Actors[400] = worldstate.Actor{ID: 400, X: 11, Y: 20, Job: 1002, HasObjectType: true, ObjectType: actorObjectTypeMob}
	ctx := client.Context{Resources: manager, Session: sess, World: world, Started: time.Now()}
	mode := NewWorldMode()
	mode.actorLife = make(map[uint32]actorLife)
	mode.setCompanionAIMessage(300, "3,400")

	ai, err := newCompanionAI(ctx, mode, companionAIHomunculus, time.Now().Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer ai.close()
	if err := ai.tick(300); err != nil {
		t.Fatal(err)
	}

	assertLuaGlobalNumber(t, ai.state, "seen_id", 300)
	assertLuaGlobalNumber(t, ai.state, "seen_owner", 100)
	assertLuaGlobalNumber(t, ai.state, "seen_x", 10)
	assertLuaGlobalNumber(t, ai.state, "seen_y", 20)
	assertLuaGlobalNumber(t, ai.state, "seen_range", 7)
	assertLuaGlobalNumber(t, ai.state, "seen_hp", 123)
	assertLuaGlobalNumber(t, ai.state, "seen_maxhp", 456)
	assertLuaGlobalNumber(t, ai.state, "seen_msg_cmd", 3)
	assertLuaGlobalNumber(t, ai.state, "seen_msg_target", 400)
	assertLuaGlobalNumber(t, ai.state, "seen_monster", 1)
	if got := int(ai.state.GetGlobal("seen_actor_count").(lua.LNumber)); got < 3 {
		t.Fatalf("seen_actor_count = %d, want at least owner/homunculus/monster", got)
	}
}

func assertLuaGlobalNumber(t *testing.T, L *lua.LState, name string, want int) {
	t.Helper()
	got, ok := L.GetGlobal(name).(lua.LNumber)
	if !ok {
		t.Fatalf("%s = %s, want number %d", name, L.GetGlobal(name).String(), want)
	}
	if int(got) != want {
		t.Fatalf("%s = %d, want %d", name, int(got), want)
	}
}
