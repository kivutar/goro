package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

func TestCompanionAILoadsDefaultBeforeCustomUntilCommandTogglesCustom(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "AI", "USER_AI"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AI", "AI.lua"), []byte(`function AI(id) source = 1 end`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AI", "USER_AI", "AI.lua"), []byte(`function AI(id) source = 2 end`), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{Resources: manager, Session: session.New()}
	mode := NewWorldMode()

	ai, err := newCompanionAI(ctx, mode, companionAIHomunculus, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer ai.close()
	if ai.source != "AI/AI.lua" || ai.custom {
		t.Fatalf("ai source=%q custom=%t, want default AI", ai.source, ai.custom)
	}
	if err := ai.tick(300); err != nil {
		t.Fatal(err)
	}
	assertLuaGlobalNumber(t, ai.state, "source", 1)

	ctx.Session.HomunculusCustomAI = true
	customAI, err := newCompanionAI(ctx, mode, companionAIHomunculus, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer customAI.close()
	if customAI.source != "AI/USER_AI/AI.lua" || !customAI.custom {
		t.Fatalf("ai source=%q custom=%t, want custom AI", customAI.source, customAI.custom)
	}
	if err := customAI.tick(300); err != nil {
		t.Fatal(err)
	}
	assertLuaGlobalNumber(t, customAI.state, "source", 2)
}

func TestCompanionAICustomModeDoesNotFallbackToDefault(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "AI"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AI", "AI.lua"), []byte(`function AI(id) source = 1 end`), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{Resources: manager, Session: session.New()}
	ctx.Session.HomunculusCustomAI = true
	if ai, err := newCompanionAI(ctx, NewWorldMode(), companionAIHomunculus, time.Now()); err == nil {
		ai.close()
		t.Fatalf("custom AI loaded source=%q, want missing custom AI error", ai.source)
	}
}

func TestCompanionAIIOOpenUsesResourceRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "AI", "USER_AI", "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := `
local f, err = io.open("./AI/USER_AI/data/test.txt", "w")
if not f then error(err) end
f:write("ok")
f:close()

local r, read_err = io.open("./AI/USER_AI/data/test.txt", "r")
if not r then error(read_err) end
loaded = r:read("*a")
r:close()

function AI(id) end
`
	if err := os.WriteFile(filepath.Join(root, "AI", "USER_AI", "AI.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{Resources: manager, Session: session.New()}
	ctx.Session.HomunculusCustomAI = true

	ai, err := newCompanionAI(ctx, NewWorldMode(), companionAIHomunculus, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer ai.close()

	assertLuaGlobalString(t, ai.state, "loaded", "ok")
	data, err := os.ReadFile(filepath.Join(root, "AI", "USER_AI", "data", "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ok" {
		t.Fatalf("test.txt = %q, want ok", string(data))
	}
}

func TestCompanionAIStringGfindSupportsLegacyDirectIterator(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "AI", "USER_AI"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := `
matched = string.gfind("1.56", "%d.%d%d")()
function AI(id) end
`
	if err := os.WriteFile(filepath.Join(root, "AI", "USER_AI", "AI.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{Resources: manager, Session: session.New()}
	ctx.Session.HomunculusCustomAI = true

	ai, err := newCompanionAI(ctx, NewWorldMode(), companionAIHomunculus, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer ai.close()

	assertLuaGlobalString(t, ai.state, "matched", "1.56")
}

func TestCompanionAILegacyTableForLoopUsesPairs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "AI", "USER_AI"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := strings.Join([]string{
		"LegacyList = {10, 20, 30}",
		"legacy_count = 0",
		"legacy_total = 0",
		"for i,v in LegacyList do",
		"\tlegacy_count = legacy_count + 1",
		"\tlegacy_total = legacy_total + v",
		"end",
		"function AI(id) end",
	}, "\r\n")
	if err := os.WriteFile(filepath.Join(root, "AI", "USER_AI", "AI.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{Resources: manager, Session: session.New()}
	ctx.Session.HomunculusCustomAI = true

	ai, err := newCompanionAI(ctx, NewWorldMode(), companionAIHomunculus, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer ai.close()

	assertLuaGlobalNumber(t, ai.state, "legacy_count", 3)
	assertLuaGlobalNumber(t, ai.state, "legacy_total", 60)
}

func TestCompanionAIReloadsWhenCustomToggleChanges(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "AI", "USER_AI"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AI", "AI.lua"), []byte(`function AI(id) end`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AI", "USER_AI", "AI.lua"), []byte(`function AI(id) end`), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	sessionState := session.New()
	sessionState.Homunculus = session.Companion{ID: 300, Active: true}
	world := worldstate.New()
	world.Actors[300] = worldstate.Actor{ID: 300, HasObjectType: true, ObjectType: actorObjectTypeHomunculus}
	ctx := client.Context{Resources: manager, Session: sessionState, World: world}
	mode := NewWorldMode()

	now := time.Now()
	mode.updateCompanionAIKind(ctx, companionAIHomunculus, 300, now)
	if mode.companionAI.homunculus == nil || mode.companionAI.homunculus.source != "AI/AI.lua" {
		t.Fatalf("homunculus AI = %+v, want default source", mode.companionAI.homunculus)
	}

	sessionState.HomunculusCustomAI = true
	mode.updateCompanionAIKind(ctx, companionAIHomunculus, 300, now.Add(time.Millisecond))
	if mode.companionAI.homunculus == nil || mode.companionAI.homunculus.source != "AI/USER_AI/AI.lua" {
		t.Fatalf("homunculus AI = %+v, want custom source", mode.companionAI.homunculus)
	}
}

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

func TestCompanionActorPositionTruncatesMovingCellsLikeRobrowser(t *testing.T) {
	now := time.Now()
	sess := session.New()
	sess.AccountID = 100
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           100,
		X:            4,
		Y:            0,
		FromX:        0,
		FromY:        0,
		ToX:          4,
		ToY:          0,
		Moving:       true,
		MoveStarted:  now.Add(-600 * time.Millisecond),
		MoveDuration: 4 * time.Second,
	}
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             14,
		Y:             0,
		FromX:         10,
		FromY:         0,
		ToX:           14,
		ToY:           0,
		Moving:        true,
		MoveStarted:   now.Add(-600 * time.Millisecond),
		MoveDuration:  4 * time.Second,
		HasObjectType: true,
		ObjectType:    actorObjectTypeHomunculus,
	}
	ctx := client.Context{Session: sess, World: world}

	ownerX, ownerY := companionActorPosition(ctx, 100)
	if ownerX != 0 || ownerY != 0 {
		t.Fatalf("owner AI position = %d,%d, want truncated 0,0", ownerX, ownerY)
	}
	actorX, actorY := companionActorPosition(ctx, 300)
	if actorX != 10 || actorY != 0 {
		t.Fatalf("companion AI position = %d,%d, want truncated 10,0", actorX, actorY)
	}
}

func TestCompanionAICellDistanceMatchesDefaultLua(t *testing.T) {
	if got, want := companionAICellDistance(0, 0, 3, 4), 5; got != want {
		t.Fatalf("distance = %d, want %d", got, want)
	}
	if got := companionAICellDistance(-1, 0, 3, 4); got != -1 {
		t.Fatalf("missing distance = %d, want -1", got)
	}
}

func TestCompanionSkillRangeFallsBackToRobrowserSkillInfo(t *testing.T) {
	sess := session.New()
	sess.Homunculus = session.Companion{ID: 300, Active: true, AttackRange: 1}
	world := worldstate.New()
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             10,
		Y:             20,
		AttackRange:   1,
		HasObjectType: true,
		ObjectType:    actorObjectTypeHomunculus,
	}
	ctx := client.Context{Session: sess, World: world}
	mode := NewWorldMode()

	if got := mode.companionSkillRange(ctx, companionAIHomunculus, 300, db.SkillHvanCaprice, 3); got != 10 {
		t.Fatalf("caprice AI range = %d, want roBrowser attack range + 1", got)
	}
}

func TestCompanionSkillRangePrefersServerSkillInfo(t *testing.T) {
	sess := session.New()
	sess.Homunculus = session.Companion{
		ID:     300,
		Active: true,
		Skills: session.Skills{
			List: []session.Skill{{ID: db.SkillHvanCaprice, Level: 3, Range: 4}},
		},
	}
	world := worldstate.New()
	world.Actors[300] = worldstate.Actor{ID: 300, AttackRange: 1, HasObjectType: true, ObjectType: actorObjectTypeHomunculus}
	ctx := client.Context{Session: sess, World: world}
	mode := NewWorldMode()

	if got := mode.companionSkillRange(ctx, companionAIHomunculus, 300, db.SkillHvanCaprice, 3); got != 5 {
		t.Fatalf("caprice server AI range = %d, want server range + 1", got)
	}
}

func TestCompanionAISkillTargetRangeUsesRobrowserEuclideanDistance(t *testing.T) {
	source := worldstate.Actor{X: 10, Y: 20}
	target := worldstate.Actor{X: 11, Y: 21}

	if companionAISkillTargetInRange(source, target, 1) {
		t.Fatal("diagonal target at sqrt(2) cells was in range 1, want roBrowser Euclidean range check")
	}
	if !companionAISkillTargetInRange(source, target, 2) {
		t.Fatal("diagonal target at sqrt(2) cells was out of range 2")
	}
}

func TestCompanionAIGetVSkillAttackRangeUsesLearnedSkillInfo(t *testing.T) {
	sess := session.New()
	sess.Homunculus = session.Companion{
		ID:          300,
		Active:      true,
		AttackRange: 1,
		Skills: session.Skills{
			List: []session.Skill{
				{ID: db.SkillHvanCaprice, Level: 5},
				{ID: db.SkillHvanChaotic, Level: 5, Range: 4},
				{ID: db.SkillMhStahlHorn, Level: 10},
			},
		},
	}
	ctx := client.Context{Session: sess}
	mode := NewWorldMode()

	if got := callCompanionGetV(t, mode, ctx, companionAIHomunculus, 6, 300, int(db.SkillHvanCaprice)); got != 9 {
		t.Fatalf("V_SKILLATTACKRANGE caprice = %d, want roBrowser range 9", got)
	}
	if got := callCompanionGetV(t, mode, ctx, companionAIHomunculus, 6, 300, int(db.SkillHvanChaotic)); got != 4 {
		t.Fatalf("V_SKILLATTACKRANGE chaotic = %d, want server range 4", got)
	}
	if got := callCompanionGetV(t, mode, ctx, companionAIHomunculus, 14, 300, int(db.SkillMhStahlHorn), 4); got != 6 {
		t.Fatalf("V_SKILLATTACKRANGE_LEVEL stahl horn = %d, want level 4 range 6", got)
	}
	if got := callCompanionGetV(t, mode, ctx, companionAIHomunculus, 6, 300, int(db.SkillMhEraserCutter)); got != 1 {
		t.Fatalf("V_SKILLATTACKRANGE unlearned = %d, want 1", got)
	}
}

func TestCompanionAIStartTickSkipsOverlapAndDefersClose(t *testing.T) {
	ai := &companionAI{kind: companionAIHomunculus, state: lua.NewState()}
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	ai.state.SetGlobal("AI", ai.state.NewFunction(func(L *lua.LState) int {
		started <- struct{}{}
		<-release
		return 0
	}))

	ai.startTick(300, companionAISnapshot{ctx: client.Context{Session: session.New()}})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first AI tick did not start")
	}

	ai.startTick(300, companionAISnapshot{ctx: client.Context{Session: session.New()}})
	select {
	case <-started:
		t.Fatal("overlapping AI tick started")
	case <-time.After(20 * time.Millisecond):
	}

	ai.close()
	close(release)
	deadline := time.After(time.Second)
	for {
		ai.mu.Lock()
		done := !ai.running && ai.state == nil
		ai.mu.Unlock()
		if done {
			return
		}
		select {
		case <-deadline:
			t.Fatal("AI close was not deferred until the running tick completed")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestCompanionAIAdjustsMagicNumber2ForHighMonsterIDs(t *testing.T) {
	ai := &companionAI{kind: companionAIHomunculus, state: lua.NewState()}
	defer ai.close()
	ai.state.SetGlobal("MagicNumber2", lua.LNumber(100000))

	ai.syncActorIDCompatibilityLimit(110013315)

	if got := int(ai.state.GetGlobal("MagicNumber2").(lua.LNumber)); got != 110013316 {
		t.Fatalf("MagicNumber2 = %d, want one past high monster gid", got)
	}
}

func TestCompanionActorMotionReturnsStandAfterWalkExpires(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             4,
		Y:             0,
		FromX:         0,
		FromY:         0,
		ToX:           4,
		ToY:           0,
		Moving:        true,
		MoveStarted:   now.Add(-time.Second),
		MoveDuration:  100 * time.Millisecond,
		AIMotion:      aiMotionMove,
		HasAIMotion:   true,
		HasObjectType: true,
		ObjectType:    actorObjectTypeHomunculus,
	}
	ctx := client.Context{World: world}
	mode := NewWorldMode()

	if got := mode.companionActorMotion(ctx, 300, nil); got != aiMotionStand {
		t.Fatalf("expired walk motion = %d, want stand", got)
	}

	actor := world.Actors[300]
	actor.MoveStarted = now
	world.Actors[300] = actor
	if got := mode.companionActorMotion(ctx, 300, nil); got != aiMotionMove {
		t.Fatalf("active walk motion = %d, want move", got)
	}
}

func TestCompanionActorMotionReturnsStandAfterActionExpires(t *testing.T) {
	world := worldstate.New()
	world.Actors[300] = worldstate.Actor{
		ID:              300,
		X:               4,
		Y:               0,
		AIMotion:        aiMotionAttack,
		HasAIMotion:     true,
		AIMotionExpires: time.Now().Add(-time.Millisecond),
		HasObjectType:   true,
		ObjectType:      actorObjectTypeHomunculus,
	}
	ctx := client.Context{World: world}
	mode := NewWorldMode()

	if got := mode.companionActorMotion(ctx, 300, nil); got != aiMotionStand {
		t.Fatalf("expired attack motion = %d, want stand", got)
	}

	actor := world.Actors[300]
	actor.AIMotionExpires = time.Now().Add(time.Second)
	world.Actors[300] = actor
	if got := mode.companionActorMotion(ctx, 300, nil); got != aiMotionAttack {
		t.Fatalf("active attack motion = %d, want attack", got)
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

func assertLuaGlobalString(t *testing.T, L *lua.LState, name, want string) {
	t.Helper()
	got, ok := L.GetGlobal(name).(lua.LString)
	if !ok {
		t.Fatalf("%s = %s, want string %q", name, L.GetGlobal(name).String(), want)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", name, string(got), want)
	}
}

func callCompanionGetV(t *testing.T, mode *WorldMode, ctx client.Context, kind companionAIKind, args ...int) int {
	t.Helper()
	L := lua.NewState()
	defer L.Close()
	for _, arg := range args {
		L.Push(lua.LNumber(arg))
	}
	if got := mode.luaCompanionGetV(L, ctx, kind, nil); got != 1 {
		t.Fatalf("luaCompanionGetV returned %d values, want 1", got)
	}
	value, ok := L.Get(-1).(lua.LNumber)
	if !ok {
		t.Fatalf("luaCompanionGetV result = %s, want number", L.Get(-1).String())
	}
	return int(value)
}
