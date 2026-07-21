package game

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const companionAITickInterval = 100 * time.Millisecond

type companionAIKind int

const (
	companionAIHomunculus companionAIKind = iota
	companionAIMercenary
)

type companionAISystem struct {
	homunculus *companionAI
	mercenary  *companionAI
	started    time.Time
	msg        map[uint32]string
	resMsg     map[uint32]string
}

type companionAI struct {
	kind     companionAIKind
	custom   bool
	state    *lua.LState
	source   string
	mainDir  string
	loaded   map[string]bool
	nextTick time.Time
	disabled bool

	mu                  sync.Mutex
	running             bool
	closeAfterRun       bool
	snapshot            companionAISnapshot
	trace               bool
	magicNumber2Checked bool
	magicNumber2Limit   int
	lastLoadErr         error
}

type companionAISnapshot struct {
	ctx         client.Context
	actorDeaths map[uint32]time.Time
}

func (s *companionAISystem) close() {
	if s == nil {
		return
	}
	if s.homunculus != nil {
		s.homunculus.close()
	}
	if s.mercenary != nil {
		s.mercenary.close()
	}
	*s = companionAISystem{}
}

func (ai *companionAI) close() {
	if ai == nil {
		return
	}
	ai.mu.Lock()
	if ai.running {
		ai.disabled = true
		ai.closeAfterRun = true
		ai.mu.Unlock()
		return
	}
	state := ai.state
	ai.state = nil
	ai.mu.Unlock()
	if state != nil {
		state.Close()
	}
}

func (m *WorldMode) updateCompanionAI(ctx client.Context, now time.Time) {
	if ctx.Session == nil || ctx.World == nil {
		return
	}
	if m.companionAI.started.IsZero() {
		m.companionAI.started = now
	}
	if m.companionAI.msg == nil {
		m.companionAI.msg = make(map[uint32]string)
	}
	if m.companionAI.resMsg == nil {
		m.companionAI.resMsg = make(map[uint32]string)
	}
	m.updateCompanionAIKind(ctx, companionAIHomunculus, ctx.Session.Homunculus.ID, now)
	m.updateCompanionAIKind(ctx, companionAIMercenary, ctx.Session.Mercenary.ID, now)
}

func (m *WorldMode) handleCompanionAICommandClick(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || ctx.World == nil || ctx.Session == nil || uiPointerBlocked(ctx) || !ctx.Input.Pressed(input.KeyAlt) {
		return false
	}
	kind := companionAIKind(-1)
	id := uint32(0)
	if ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		kind = companionAIMercenary
		id = ctx.Session.Mercenary.ID
	} else if ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		kind = companionAIHomunculus
		id = ctx.Session.Homunculus.ID
	} else {
		return false
	}
	if id == 0 {
		return true
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	if actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
		m.setCompanionAIMessage(id, fmt.Sprintf("3,%d", actor.ID))
		glog.Debugf("%s AI command attack companion=%d target=%d", companionAIKindName(kind), id, actor.ID)
		return true
	}
	if x, y, ok := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); ok {
		m.setCompanionAIMessage(id, fmt.Sprintf("1,%d,%d", x, y))
		glog.Debugf("%s AI command move companion=%d target=%d,%d", companionAIKindName(kind), id, x, y)
		return true
	}
	return true
}

func (m *WorldMode) updateCompanionAIKind(ctx client.Context, kind companionAIKind, id uint32, now time.Time) {
	if id == 0 || !m.companionActorPresent(ctx, kind, id) {
		return
	}
	ai := m.companionAIForKind(kind)
	useCustom := companionAIUseCustom(ctx.Session, kind)
	if ai != nil && ai.custom != useCustom {
		glog.Debugf("%s AI mode changed custom=%t, reloading", companionAIKindName(kind), useCustom)
		ai.close()
		m.setCompanionAIForKind(kind, nil)
		ai = nil
	}
	if ai == nil {
		loaded, err := newCompanionAI(ctx, m, kind, now)
		if err != nil {
			m.setCompanionAIForKind(kind, &companionAI{kind: kind, custom: useCustom, disabled: true, lastLoadErr: err})
			glog.Debugf("%s AI unavailable: %v", companionAIKindName(kind), err)
			return
		}
		ai = loaded
		m.setCompanionAIForKind(kind, ai)
		glog.Debugf("%s AI loaded source=%s", companionAIKindName(kind), ai.source)
	}
	if now.Before(ai.nextTick) {
		return
	}
	ai.nextTick = now.Add(companionAITickInterval)
	ai.startTick(id, newCompanionAISnapshot(ctx, m.actorDeaths))
}

func (m *WorldMode) companionAIForKind(kind companionAIKind) *companionAI {
	switch kind {
	case companionAIHomunculus:
		return m.companionAI.homunculus
	case companionAIMercenary:
		return m.companionAI.mercenary
	default:
		return nil
	}
}

func (m *WorldMode) setCompanionAIForKind(kind companionAIKind, ai *companionAI) {
	switch kind {
	case companionAIHomunculus:
		m.companionAI.homunculus = ai
	case companionAIMercenary:
		m.companionAI.mercenary = ai
	}
}

func (m *WorldMode) companionActorPresent(ctx client.Context, kind companionAIKind, id uint32) bool {
	if ctx.World == nil || id == 0 {
		return false
	}
	actor, ok := ctx.World.Actors[id]
	if !ok || !actor.HasObjectType {
		return false
	}
	switch kind {
	case companionAIHomunculus:
		return actor.ObjectType == actorObjectTypeHomunculus
	case companionAIMercenary:
		return actor.ObjectType == actorObjectTypeMercenary
	default:
		return false
	}
}

func newCompanionAI(ctx client.Context, mode *WorldMode, kind companionAIKind, now time.Time) (*companionAI, error) {
	if ctx.Resources == nil {
		return nil, fmt.Errorf("resources unavailable")
	}
	useCustom := companionAIUseCustom(ctx.Session, kind)
	candidates := companionAICandidates(kind, useCustom)
	var lastErr error
	for _, candidate := range candidates {
		ai := &companionAI{
			kind:     kind,
			custom:   useCustom,
			state:    lua.NewState(),
			source:   candidate.main,
			mainDir:  candidate.dir,
			loaded:   make(map[string]bool),
			nextTick: now.Add(companionAITickInterval),
			trace:    companionAITraceEnabled(),
		}
		ai.registerAPI(ctx, mode)
		if err := ai.doAIFile(ctx.Resources, candidate.main); err != nil {
			ai.close()
			lastErr = err
			continue
		}
		if _, ok := ai.state.GetGlobal("AI").(*lua.LFunction); !ok {
			ai.close()
			lastErr = fmt.Errorf("%s does not define AI", candidate.main)
			continue
		}
		return ai, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no AI candidates")
	}
	return nil, lastErr
}

type companionAICandidate struct {
	main string
	dir  string
}

func companionAIUseCustom(sessionState *session.Session, kind companionAIKind) bool {
	if sessionState == nil {
		return false
	}
	switch kind {
	case companionAIHomunculus:
		return sessionState.HomunculusCustomAI
	case companionAIMercenary:
		return sessionState.MercenaryCustomAI
	default:
		return false
	}
}

func companionAICandidates(kind companionAIKind, useCustom bool) []companionAICandidate {
	main := "AI.lua"
	if kind == companionAIMercenary {
		main = "AI_M.lua"
	}
	defaultCandidate := companionAICandidate{main: "AI/" + main, dir: "AI"}
	customCandidate := companionAICandidate{main: "AI/USER_AI/" + main, dir: "AI/USER_AI"}
	if useCustom {
		return []companionAICandidate{customCandidate}
	}
	return []companionAICandidate{defaultCandidate}
}

func (ai *companionAI) registerAPI(ctx client.Context, mode *WorldMode) {
	L := ai.state
	ai.registerLua51Compatibility()
	ai.registerFileIO(ctx)
	L.SetGlobal("MoveToOwner", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		if ai.trace {
			glog.Debugf("%s AI MoveToOwner id=%d", companionAIKindName(ai.kind), id)
		}
		if ctx.Network != nil {
			if err := ctx.Network.SendCompanionMoveToOwner(id); err != nil {
				glog.Warnf("%s AI MoveToOwner failed id=%d: %v", companionAIKindName(ai.kind), id, err)
			}
		}
		return 0
	}))
	L.SetGlobal("Move", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		x := L.CheckInt(2)
		y := L.CheckInt(3)
		if ai.trace {
			glog.Debugf("%s AI Move id=%d dst=%d,%d", companionAIKindName(ai.kind), id, x, y)
		}
		if ctx.Network != nil {
			if err := ctx.Network.SendCompanionMove(id, x, y); err != nil {
				glog.Warnf("%s AI Move failed id=%d dst=%d,%d: %v", companionAIKindName(ai.kind), id, x, y, err)
			}
		}
		return 0
	}))
	L.SetGlobal("Attack", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		targetID := uint32(L.CheckInt(2))
		if ai.trace {
			glog.Debugf("%s AI Attack id=%d target=%d", companionAIKindName(ai.kind), id, targetID)
		}
		if ctx.Network != nil {
			if err := ctx.Network.SendCompanionAttack(id, targetID); err != nil {
				glog.Warnf("%s AI Attack failed id=%d target=%d: %v", companionAIKindName(ai.kind), id, targetID, err)
			}
		}
		return 0
	}))
	L.SetGlobal("GetV", L.NewFunction(func(L *lua.LState) int {
		return mode.luaCompanionGetV(L, ai.apiContext(ctx), ai.kind, ai.apiActorDeaths(mode))
	}))
	L.SetGlobal("GetActors", L.NewFunction(func(L *lua.LState) int {
		apiCtx := ai.apiContext(ctx)
		deaths := ai.apiActorDeaths(mode)
		mode.maybeQueueAggressiveCompanionTarget(apiCtx, ai.kind, companionIDForAIKind(apiCtx, ai.kind), deaths)
		L.Push(mode.luaCompanionActors(L, apiCtx, ai))
		return 1
	}))
	L.SetGlobal("GetTick", L.NewFunction(func(L *lua.LState) int {
		started := mode.companionAI.started
		if started.IsZero() {
			started = time.Now()
		}
		L.Push(lua.LNumber(time.Since(started).Milliseconds()))
		return 1
	}))
	L.SetGlobal("GetMsg", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		L.Push(mode.luaCompanionMessageTable(L, id, false))
		return 1
	}))
	L.SetGlobal("GetResMsg", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		L.Push(mode.luaCompanionMessageTable(L, id, true))
		return 1
	}))
	L.SetGlobal("chaiGetMsg", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		values := mode.consumeCompanionAIMessage(id, false)
		for i := 0; i < 3; i++ {
			value := 0
			if i < len(values) {
				value = values[i]
			}
			L.Push(lua.LNumber(value))
		}
		return 3
	}))
	L.SetGlobal("SkillObject", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		level := uint16(maxInt(1, L.CheckInt(2)))
		skillID := uint16(L.CheckInt(3))
		targetID := uint32(L.CheckInt(4))
		mode.luaCompanionSkillObject(ai.apiContext(ctx), ai.kind, id, level, skillID, targetID, ai.apiActorDeaths(mode))
		L.Push(lua.LNumber(0))
		return 1
	}))
	L.SetGlobal("SkillGround", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		level := uint16(maxInt(1, L.CheckInt(2)))
		skillID := uint16(L.CheckInt(3))
		x := L.CheckInt(4)
		y := L.CheckInt(5)
		mode.luaCompanionSkillGround(ai.apiContext(ctx), ai.kind, id, level, skillID, x, y, ai.apiActorDeaths(mode))
		L.Push(lua.LNumber(0))
		return 1
	}))
	L.SetGlobal("IsMonster", L.NewFunction(func(L *lua.LState) int {
		id := uint32(L.CheckInt(1))
		if mode.luaCompanionIsMonster(ai.apiContext(ctx), id, ai.apiActorDeaths(mode)) {
			L.Push(lua.LNumber(1))
		} else {
			L.Push(lua.LNumber(0))
		}
		return 1
	}))
	L.SetGlobal("TraceAI", L.NewFunction(func(L *lua.LState) int {
		if ai.trace {
			glog.Debugf("%s AI TraceAI: %s", companionAIKindName(ai.kind), luaValueString(L.Get(1)))
		}
		return 0
	}))
	L.SetGlobal("Trace", L.NewFunction(func(L *lua.LState) int {
		if ai.trace {
			glog.Debugf("%s AI Trace: %s", companionAIKindName(ai.kind), luaValueString(L.Get(1)))
		}
		return 0
	}))
	L.SetGlobal("TraceValue", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(luaValueString(L.Get(1))))
		return 1
	}))
	L.SetGlobal("chaiDevLog", L.NewFunction(func(L *lua.LState) int {
		if ai.trace {
			glog.Debugf("%s AI %s: %s", companionAIKindName(ai.kind), luaValueString(L.Get(1)), luaValueString(L.Get(2)))
		}
		return 0
	}))
	L.SetGlobal("require", L.NewFunction(func(L *lua.LState) int {
		module := L.CheckString(1)
		if err := ai.requireAIFile(ctx.Resources, module); err != nil {
			L.RaiseError("require %q failed: %v", module, err)
			return 0
		}
		L.Push(lua.LBool(true))
		return 1
	}))
	L.SetGlobal("dofile", L.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)
		if err := ai.doAIFile(ctx.Resources, path); err != nil {
			L.RaiseError("dofile %q failed: %v", path, err)
			return 0
		}
		return 0
	}))
}

func (ai *companionAI) registerLua51Compatibility() {
	const source = `
if string and string.gmatch then
	local gmatch = string.gmatch
	string.gfind = function(s, pattern)
		local iter, state = gmatch(s, pattern)
		return function()
			return iter(state)
		end
	end
end
`
	if err := ai.state.DoString(source); err != nil {
		glog.Warnf("%s AI Lua 5.1 compatibility failed: %v", companionAIKindName(ai.kind), err)
	}
}

func (ai *companionAI) registerFileIO(ctx client.Context) {
	if ctx.Resources == nil || ctx.Resources.Root == "" {
		return
	}
	L := ai.state
	ioTable, ok := L.GetGlobal("io").(*lua.LTable)
	if !ok {
		return
	}
	openFn, ok := ioTable.RawGetString("open").(*lua.LFunction)
	if !ok {
		return
	}
	root := ctx.Resources.Root
	ioTable.RawSetString("open", L.NewFunction(func(L *lua.LState) int {
		path := companionAIIOPath(root, L.CheckString(1))
		mode := ""
		nargs := 1
		if L.GetTop() >= 2 {
			mode = L.CheckString(2)
			nargs = 2
		}
		if companionAIIOWriteMode(mode) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
		}
		baseTop := L.GetTop()
		L.Push(openFn)
		L.Push(lua.LString(path))
		if nargs == 2 {
			L.Push(lua.LString(mode))
		}
		if err := L.PCall(nargs, lua.MultRet, nil); err != nil {
			L.SetTop(baseTop)
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		return L.GetTop() - baseTop
	}))
}

func companionAIIOPath(root, path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"'`)
	path = strings.ReplaceAll(path, "\\", string(filepath.Separator))
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	path = strings.TrimPrefix(path, "."+string(filepath.Separator))
	path = strings.TrimPrefix(path, "./")
	return filepath.Join(root, path)
}

func companionAIIOWriteMode(mode string) bool {
	return strings.HasPrefix(mode, "w") || strings.HasPrefix(mode, "a")
}

func companionAITraceEnabled() bool {
	return os.Getenv("GORO_DEBUG_AI_TRACE") == "1"
}

func (ai *companionAI) syncActorIDCompatibilityLimit(maxMonsterID uint32) {
	if ai == nil || ai.state == nil || maxMonsterID == 0 {
		return
	}
	if !ai.magicNumber2Checked {
		ai.magicNumber2Checked = true
		current, ok := ai.state.GetGlobal("MagicNumber2").(lua.LNumber)
		if !ok || current <= 0 {
			ai.magicNumber2Limit = -1
			return
		}
		ai.magicNumber2Limit = int(current)
	}
	if ai.magicNumber2Limit <= 0 {
		return
	}
	nextLimit := int(maxMonsterID) + 1
	if nextLimit <= ai.magicNumber2Limit {
		return
	}
	oldLimit := ai.magicNumber2Limit
	ai.magicNumber2Limit = nextLimit
	ai.state.SetGlobal("MagicNumber2", lua.LNumber(nextLimit))
	if ai.trace {
		glog.Debugf("%s AI MagicNumber2 adjusted from %d to %d for high monster gids", companionAIKindName(ai.kind), oldLimit, nextLimit)
	}
}

func (ai *companionAI) startTick(id uint32, snapshot companionAISnapshot) {
	ai.mu.Lock()
	if ai.disabled || ai.running || ai.state == nil {
		ai.mu.Unlock()
		return
	}
	ai.running = true
	ai.snapshot = snapshot
	source := ai.source
	kind := ai.kind
	ai.mu.Unlock()

	go func() {
		err := ai.tick(id)

		ai.mu.Lock()
		ai.running = false
		closeAfterRun := ai.closeAfterRun
		if err != nil {
			ai.disabled = true
			closeAfterRun = true
		}
		var state *lua.LState
		if closeAfterRun {
			state = ai.state
			ai.state = nil
			ai.closeAfterRun = false
		}
		ai.snapshot = companionAISnapshot{}
		ai.mu.Unlock()

		if err != nil {
			glog.Warnf("%s AI tick failed source=%s id=%d: %v", companionAIKindName(kind), source, id, err)
		}
		if state != nil {
			state.Close()
		}
	}()
}

func (ai *companionAI) tick(id uint32) error {
	if ai == nil || ai.state == nil {
		return nil
	}
	fn := ai.state.GetGlobal("AI")
	if fn == lua.LNil {
		return nil
	}
	return ai.state.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	}, lua.LNumber(id))
}

func (ai *companionAI) apiContext(defaultCtx client.Context) client.Context {
	if ai == nil {
		return defaultCtx
	}
	if ai.hasSnapshot() {
		return ai.snapshot.ctx
	}
	return defaultCtx
}

func (ai *companionAI) apiActorDeaths(mode *WorldMode) map[uint32]time.Time {
	if ai != nil && ai.hasSnapshot() {
		return ai.snapshot.actorDeaths
	}
	if mode == nil {
		return nil
	}
	return mode.actorDeaths
}

func (ai *companionAI) hasSnapshot() bool {
	return ai != nil && (ai.snapshot.ctx.Session != nil || ai.snapshot.ctx.World != nil)
}

func newCompanionAISnapshot(ctx client.Context, actorDeaths map[uint32]time.Time) companionAISnapshot {
	snapshot := ctx
	snapshot.Input = nil
	snapshot.Audio = nil
	snapshot.Runtime = nil
	snapshot.RequestQuit = nil
	snapshot.RequestScreenshot = nil
	snapshot.UIApp = nil
	snapshot.UIManager = nil
	snapshot.Session = copySessionForCompanionAI(ctx.Session)
	snapshot.World = copyWorldForCompanionAI(ctx.World)
	return companionAISnapshot{
		ctx:         snapshot,
		actorDeaths: copyActorDeathsForCompanionAI(actorDeaths),
	}
}

func copySessionForCompanionAI(src *session.Session) *session.Session {
	if src == nil {
		return nil
	}
	dst := *src
	dst.CharServers = append([]session.CharServer(nil), src.CharServers...)
	dst.Characters = append([]session.Character(nil), src.Characters...)
	dst.Homunculus.Skills.List = append([]session.Skill(nil), src.Homunculus.Skills.List...)
	dst.Mercenary.Skills.List = append([]session.Skill(nil), src.Mercenary.Skills.List...)
	return &dst
}

func copyWorldForCompanionAI(src *worldstate.World) *worldstate.World {
	if src == nil {
		return nil
	}
	dst := *src
	dst.Player = copyActorForCompanionAI(src.Player)
	dst.Actors = make(map[uint32]worldstate.Actor, len(src.Actors))
	for id, actor := range src.Actors {
		dst.Actors[id] = copyActorForCompanionAI(actor)
	}
	dst.Items = nil
	dst.RSM = nil
	return &dst
}

func copyActorForCompanionAI(actor worldstate.Actor) worldstate.Actor {
	actor.MovePath = append([]worldstate.WalkStep(nil), actor.MovePath...)
	return actor
}

func copyActorDeathsForCompanionAI(src map[uint32]time.Time) map[uint32]time.Time {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[uint32]time.Time, len(src))
	for id, removeAt := range src {
		dst[id] = removeAt
	}
	return dst
}

func (ai *companionAI) requireAIFile(manager *res.Manager, module string) error {
	path := ai.normalizeAIPath(module)
	if ai.loaded[path] {
		return nil
	}
	return ai.doAIFile(manager, path)
}

func (ai *companionAI) doAIFile(manager *res.Manager, path string) error {
	path = ai.normalizeAIPath(path)
	data, err := manager.ReadFile(path)
	if err != nil {
		return err
	}
	text := decodeAILua(data)
	ai.loaded[path] = true
	if err := ai.state.DoString(text); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func (ai *companionAI) normalizeAIPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"'`)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimSuffix(path, ".lua")
	if !strings.Contains(path, "/") && ai.mainDir != "" {
		path = ai.mainDir + "/" + path
	}
	return path + ".lua"
}

func decodeAILua(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), data)
	if err != nil {
		return string(data)
	}
	return string(decoded)
}

func (m *WorldMode) luaCompanionGetV(L *lua.LState, ctx client.Context, kind companionAIKind, actorDeaths map[uint32]time.Time) int {
	variable := L.CheckInt(1)
	id := uint32(L.CheckInt(2))
	switch variable {
	case 0:
		L.Push(lua.LNumber(localSkillTarget(ctx)))
		return 1
	case 1, 13:
		x, y := companionActorPosition(ctx, id)
		L.Push(lua.LNumber(x))
		L.Push(lua.LNumber(y))
		return 2
	case 2:
		L.Push(lua.LNumber(1))
		return 1
	case 3:
		L.Push(lua.LNumber(m.companionActorMotion(ctx, id, actorDeaths)))
		return 1
	case 4:
		L.Push(lua.LNumber(m.companionAttackRange(ctx, kind, id)))
		return 1
	case 6, 14:
		skillID := uint16(0)
		if L.GetTop() >= 3 {
			skillID = uint16(L.CheckInt(3))
		}
		level := uint16(0)
		if variable == 14 && L.GetTop() >= 4 {
			level = uint16(maxInt(1, L.CheckInt(4)))
		}
		L.Push(lua.LNumber(m.companionSkillAttackRange(ctx, kind, id, skillID, level)))
		return 1
	case 5:
		targetID := companionActorTarget(ctx, id)
		if targetID == 0 {
			L.Push(lua.LNumber(-1))
		} else {
			L.Push(lua.LNumber(targetID))
		}
		return 1
	case 7:
		actor, ok := companionActorByID(ctx, id)
		if !ok {
			L.Push(lua.LNumber(-1))
			return 1
		}
		L.Push(lua.LNumber(int(actor.Job) % 6000))
		return 1
	case 8:
		hp, _, _, _, ok := m.companionLife(ctx, id)
		pushCompanionOptionalNumber(L, hp, ok)
		return 1
	case 9:
		_, _, sp, _, ok := m.companionLife(ctx, id)
		pushCompanionOptionalNumber(L, sp, ok)
		return 1
	case 10:
		_, maxHP, _, _, ok := m.companionLife(ctx, id)
		pushCompanionOptionalNumber(L, maxHP, ok)
		return 1
	case 11:
		_, _, _, maxSP, ok := m.companionLife(ctx, id)
		pushCompanionOptionalNumber(L, maxSP, ok)
		return 1
	case 12:
		actor, ok := companionActorByID(ctx, id)
		if !ok {
			L.Push(lua.LNumber(0))
			return 1
		}
		L.Push(lua.LNumber(mercenaryTypeFromJob(actor.Job)))
		return 1
	default:
		L.Push(lua.LNumber(0))
		return 1
	}
}

func (m *WorldMode) luaCompanionActors(L *lua.LState, ctx client.Context, ai *companionAI) *lua.LTable {
	result := L.NewTable()
	result.Append(lua.LNumber(0))
	if ctx.World == nil {
		return result
	}
	if owner := localSkillTarget(ctx); owner != 0 {
		result.Append(lua.LNumber(owner))
	}
	ids := make([]int, 0, len(ctx.World.Actors))
	maxMonsterID := uint32(0)
	for id, actor := range ctx.World.Actors {
		ids = append(ids, int(id))
		if actor.HasObjectType && companionAIObjectTypeIsMonster(actor.ObjectType) && id > maxMonsterID {
			maxMonsterID = id
		}
	}
	ai.syncActorIDCompatibilityLimit(maxMonsterID)
	sort.Ints(ids)
	for _, id := range ids {
		result.Append(lua.LNumber(id))
	}
	return result
}

func companionAIObjectTypeIsMonster(objectType uint8) bool {
	return objectType == actorObjectTypeMob || objectType == actorObjectTypeNPCABR || objectType == actorObjectTypeNPCBionic
}

func (m *WorldMode) luaCompanionMessageTable(L *lua.LState, id uint32, response bool) *lua.LTable {
	result := L.NewTable()
	for _, value := range m.consumeCompanionAIMessage(id, response) {
		result.Append(lua.LNumber(value))
	}
	return result
}

func (m *WorldMode) consumeCompanionAIMessage(id uint32, response bool) []int {
	raw := "0"
	queue := m.companionAI.msg
	if response {
		queue = m.companionAI.resMsg
	}
	if queue != nil {
		if value, ok := queue[id]; ok {
			raw = value
			delete(queue, id)
		}
	}
	parts := strings.Split(raw, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			value = 0
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		values = append(values, 0)
	}
	return values
}

func (m *WorldMode) setCompanionAIMessage(id uint32, raw string) {
	if id == 0 {
		return
	}
	if m.companionAI.msg == nil {
		m.companionAI.msg = make(map[uint32]string)
	}
	if m.companionAI.resMsg == nil {
		m.companionAI.resMsg = make(map[uint32]string)
	}
	if _, exists := m.companionAI.msg[id]; !exists {
		m.companionAI.msg[id] = raw
		return
	}
	m.companionAI.resMsg[id] = raw
}

func (m *WorldMode) luaCompanionSkillObject(ctx client.Context, kind companionAIKind, id uint32, level, skillID uint16, targetID uint32, actorDeaths map[uint32]time.Time) {
	if ctx.Network == nil || !m.companionIDMatches(ctx, kind, id) {
		return
	}
	source, ok := companionActorByID(ctx, id)
	if !ok || !m.companionCanUseSkill(ctx, id, actorDeaths) {
		return
	}
	target, ok := companionActorByID(ctx, targetID)
	if !ok {
		return
	}
	rangeCells := m.companionSkillRange(ctx, kind, id, skillID, level)
	if rangeCells <= 0 {
		rangeCells = m.companionAttackRange(ctx, kind, id)
	}
	if !companionAISkillTargetInRange(source, target, rangeCells) {
		return
	}
	if err := ctx.Network.SendUseSkillToID(skillID, level, targetID); err != nil {
		glog.Warnf("%s AI SkillObject failed id=%d skill=%d level=%d target=%d: %v", companionAIKindName(kind), id, skillID, level, targetID, err)
	}
}

func (m *WorldMode) luaCompanionSkillGround(ctx client.Context, kind companionAIKind, id uint32, level, skillID uint16, x, y int, actorDeaths map[uint32]time.Time) {
	if ctx.Network == nil || !m.companionIDMatches(ctx, kind, id) || !m.companionCanUseSkill(ctx, id, actorDeaths) {
		return
	}
	if !walkTargetInBounds(ctx, x, y) {
		return
	}
	if err := ctx.Network.SendUseSkillToGround(skillID, level, x, y); err != nil {
		glog.Warnf("%s AI SkillGround failed id=%d skill=%d level=%d target=%d,%d: %v", companionAIKindName(kind), id, skillID, level, x, y, err)
	}
}

func (m *WorldMode) companionIDMatches(ctx client.Context, kind companionAIKind, id uint32) bool {
	if ctx.Session == nil || id == 0 {
		return false
	}
	switch kind {
	case companionAIHomunculus:
		return id == ctx.Session.Homunculus.ID
	case companionAIMercenary:
		return id == ctx.Session.Mercenary.ID
	default:
		return false
	}
}

func (m *WorldMode) companionCanUseSkill(ctx client.Context, id uint32, actorDeaths map[uint32]time.Time) bool {
	switch m.companionActorMotion(ctx, id, actorDeaths) {
	case aiMotionStand, aiMotionMove, aiMotionAttack:
		return true
	default:
		return false
	}
}

func companionAISkillTargetInRange(source, target worldstate.Actor, rangeCells int) bool {
	return companionCellDistance(source.X, source.Y, target.X, target.Y) <= float64(rangeCells)
}

func (m *WorldMode) companionSkillAttackRange(ctx client.Context, kind companionAIKind, id uint32, skillID, level uint16) int {
	if skillID == 0 {
		return m.companionAttackRange(ctx, kind, id)
	}
	if !m.companionIDMatches(ctx, kind, id) {
		return 1
	}
	skill, ok := companionAISkill(ctx.Session, kind, skillID)
	if !ok || skill.Level <= 0 {
		return 1
	}
	if skill.Range > 0 {
		return skill.Range
	}
	lookupLevel := int(level)
	if lookupLevel <= 0 || lookupLevel > skill.Level {
		lookupLevel = skill.Level
	}
	if attackRange, ok := db.SkillAttackRange(skillID, lookupLevel); ok {
		return attackRange
	}
	return 1
}

func (m *WorldMode) companionSkillRange(ctx client.Context, kind companionAIKind, id uint32, skillID, level uint16) int {
	if ctx.Session == nil {
		return 1
	}
	if skill, ok := companionAISkill(ctx.Session, kind, skillID); ok && skill.Range > 0 {
		return skill.Range + 1
	}
	if attackRange, ok := db.SkillAttackRange(skillID, int(level)); ok {
		return attackRange + 1
	}
	return m.companionAttackRange(ctx, kind, id)
}

func companionAISkill(sessionState *session.Session, kind companionAIKind, skillID uint16) (session.Skill, bool) {
	if sessionState == nil {
		return session.Skill{}, false
	}
	skills := sessionState.Homunculus.Skills.List
	if kind == companionAIMercenary {
		skills = sessionState.Mercenary.Skills.List
	}
	for _, skill := range skills {
		if skill.ID == skillID {
			return skill, true
		}
	}
	return session.Skill{}, false
}

func (m *WorldMode) luaCompanionIsMonster(ctx client.Context, id uint32, actorDeaths map[uint32]time.Time) bool {
	if id == 0 || ctx.World == nil {
		return false
	}
	if _, dead := actorDeaths[id]; dead {
		return false
	}
	actor, ok := ctx.World.Actors[id]
	if !ok || !actor.HasObjectType {
		return false
	}
	return companionAIObjectTypeIsMonster(actor.ObjectType)
}

func companionActorByID(ctx client.Context, id uint32) (worldstate.Actor, bool) {
	if ctx.World == nil || id == 0 {
		return worldstate.Actor{}, false
	}
	if isLocalActor(ctx, id) {
		actor := ctx.World.Player
		actor.ID = id
		if ctx.Session != nil {
			character := ctx.Session.SelectedCharacter()
			actor.Job = character.Job
			actor.Weapon = character.Weapon
			actor.Shield = character.Shield
			actor.Sex = ctx.Session.Sex
			actor.Appearance = true
		}
		return actor, true
	}
	actor, ok := ctx.World.Actors[id]
	return actor, ok
}

func companionActorPosition(ctx client.Context, id uint32) (int, int) {
	if ctx.World == nil || id == 0 {
		return -1, -1
	}
	now := time.Now()
	if isLocalActor(ctx, id) {
		return companionActorRenderCell(ctx.World.Player, now)
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return -1, -1
	}
	return companionActorRenderCell(actor, now)
}

func companionActorRenderCell(actor worldstate.Actor, now time.Time) (int, int) {
	if actor.Moving {
		x, y := actorRenderPosition(actor, now)
		return int(x), int(y)
	}
	return actor.X, actor.Y
}

func companionAICellDistance(x1, y1, x2, y2 int) int {
	if x1 == -1 || x2 == -1 {
		return -1
	}
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return int(math.Floor(math.Sqrt(dx*dx + dy*dy)))
}

func (m *WorldMode) companionActorMotion(ctx client.Context, id uint32, actorDeaths map[uint32]time.Time) int {
	if ctx.World == nil || id == 0 {
		return aiMotionStand
	}
	now := time.Now()
	if _, dead := actorDeaths[id]; dead {
		return aiMotionDead
	}
	if isLocalActor(ctx, id) {
		if actorIsMovingAt(ctx.World.Player, now) {
			return aiMotionMove
		}
		if ctx.World.Player.HasAIMotion {
			if ctx.World.Player.AIMotion == aiMotionMove && ctx.World.Player.Moving {
				return aiMotionStand
			}
			if companionAIMotionExpired(ctx.World.Player, now) {
				return aiMotionStand
			}
			return ctx.World.Player.AIMotion
		}
		return aiMotionStand
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return aiMotionStand
	}
	if actorIsMovingAt(actor, now) {
		return aiMotionMove
	}
	if actor.HasAIMotion {
		if actor.AIMotion == aiMotionMove && actor.Moving {
			return aiMotionStand
		}
		if companionAIMotionExpired(actor, now) {
			return aiMotionStand
		}
		return actor.AIMotion
	}
	return aiMotionStand
}

func companionAIMotionExpired(actor worldstate.Actor, now time.Time) bool {
	return actor.AIMotion != aiMotionDead && !actor.AIMotionExpires.IsZero() && !now.Before(actor.AIMotionExpires)
}

func companionActorTarget(ctx client.Context, id uint32) uint32 {
	if ctx.World == nil || id == 0 {
		return 0
	}
	if isLocalActor(ctx, id) {
		return ctx.World.Player.AITargetID
	}
	if actor, ok := ctx.World.Actors[id]; ok {
		return actor.AITargetID
	}
	return 0
}

func (m *WorldMode) companionAttackRange(ctx client.Context, kind companionAIKind, id uint32) int {
	if actor, ok := companionActorByID(ctx, id); ok && actor.AttackRange > 0 {
		return actor.AttackRange
	}
	if ctx.Session != nil {
		switch kind {
		case companionAIHomunculus:
			if id == ctx.Session.Homunculus.ID && ctx.Session.Homunculus.AttackRange > 0 {
				return ctx.Session.Homunculus.AttackRange
			}
		case companionAIMercenary:
			if id == ctx.Session.Mercenary.ID && ctx.Session.Mercenary.AttackRange > 0 {
				return ctx.Session.Mercenary.AttackRange
			}
		}
	}
	return 1
}

func (m *WorldMode) companionLife(ctx client.Context, id uint32) (hp, maxHP, sp, maxSP int, ok bool) {
	if ctx.Session != nil {
		if id == ctx.Session.Homunculus.ID && ctx.Session.Homunculus.Active {
			hom := ctx.Session.Homunculus
			return hom.HP, hom.MaxHP, hom.SP, hom.MaxSP, true
		}
		if id == ctx.Session.Mercenary.ID && ctx.Session.Mercenary.Active {
			merc := ctx.Session.Mercenary
			return merc.HP, merc.MaxHP, merc.SP, merc.MaxSP, true
		}
	}
	if m.actorLife != nil {
		if life, found := m.actorLife[id]; found {
			return life.hp, life.maxHP, life.sp, life.maxSP, true
		}
	}
	return 0, 0, 0, 0, false
}

func pushCompanionOptionalNumber(L *lua.LState, value int, ok bool) {
	if !ok {
		L.Push(lua.LNumber(-1))
		return
	}
	L.Push(lua.LNumber(value))
}

func mercenaryTypeFromJob(job int16) int {
	text := strconv.Itoa(int(job))
	if strings.HasPrefix(text, "-") {
		text = strings.TrimPrefix(text, "-")
	}
	if len(text) <= 1 {
		return 0
	}
	value, err := strconv.Atoi(text[1:])
	if err != nil {
		return 0
	}
	return value
}

func companionAIKindName(kind companionAIKind) string {
	switch kind {
	case companionAIHomunculus:
		return "homunculus"
	case companionAIMercenary:
		return "mercenary"
	default:
		return "companion"
	}
}

func luaValueString(value lua.LValue) string {
	if value == nil || value == lua.LNil {
		return ""
	}
	return value.String()
}
