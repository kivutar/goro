package game

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
	lua "github.com/yuin/gopher-lua"
)

// Exercise the shipped logic with fixed settings, independently of changes to
// the user's leader name or learned skill levels made for live play testing.
func companionScriptForTest(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "scripts", "companion.lua"))
	if err != nil {
		t.Fatal(err)
	}
	prefix, rest, ok := strings.Cut(string(data), "local config = {")
	if !ok {
		t.Fatal("companion configuration block is missing")
	}
	_, suffix, ok := strings.Cut(rest, "\n}")
	if !ok {
		t.Fatal("companion configuration block is unterminated")
	}
	const settings = `local config = {
    leader = "Malki",
    heal_level = 1, blessing_level = 1, increase_agi_level = 1,
    follow_start = 6, follow_stop = 4,
    catch_up_seconds = 10,
    heal_below = 0.70, critical_hp = 0.35,
    rest_below_sp = 0.15, resume_at_sp = 0.70,
}`
	path := filepath.Join(t.TempDir(), "companion.lua")
	if err := os.WriteFile(path, []byte(prefix+settings+suffix), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Run the shipped script against observations and a clock we control. The
// actions deliberately do not mutate HP/SP: only a later server update does.
func newCompanionScriptTest(t *testing.T) func(float64, string, ...string) {
	t.Helper()
	state := lua.NewState()
	t.Cleanup(state.Close)
	if err := state.DoString(`
test_now = 0
test_me = {id=1, x=10, y=10, hp=100, max_hp=100, sp=20, max_sp=100, dead=false}
test_leader = {id=2, name="Malki", x=12, y=10, distance=2, hp=100, max_hp=100, party_member=true, dead=false}
test_players = {test_leader}
test_enemies, test_inventory, test_reject = {}, {}, {}
test_calls = {}
os.clock = function() return test_now end
local function action(text, key)
    table.insert(test_calls, text)
    return not test_reject[key or text]
end
goro = {
    player = function() return test_me end,
    players = function() return test_players end,
    enemies = function() return test_enemies end,
    inventory = function() return test_inventory end,
    walk = function(x, y) return action(string.format("walk:%d,%d", x, y), "walk") end,
    stop = function() return action("stop") end,
    message = function(text) return action(text) end,
    use_item = function(index) return action("item:" .. index, "item") end,
    skill = function(id, name, level)
        return action(string.format("skill:%s:%d:%d", name, id, level), name)
    end,
}
`); err != nil {
		t.Fatal(err)
	}
	if err := state.DoFile(companionScriptForTest(t)); err != nil {
		t.Fatal(err)
	}
	return func(now float64, setup string, want ...string) {
		t.Helper()
		state.SetGlobal("test_now", lua.LNumber(now))
		state.SetGlobal("test_calls", state.NewTable())
		if err := state.DoString(setup); err != nil {
			t.Fatal(err)
		}
		if err := state.CallByParam(lua.P{Fn: state.GetGlobal("tick"), Protect: true}); err != nil {
			t.Fatal(err)
		}
		var got []string
		state.GetGlobal("test_calls").(*lua.LTable).ForEach(func(_, value lua.LValue) {
			got = append(got, value.String())
		})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("at %.2fs: actions=%v, want %v", now, got, want)
		}
	}
}

func TestCompanionFollowsConfiguredPlayerWithoutParty(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_leader.x=20; test_leader.distance=10; test_leader.party_member=false; test_leader.name="Someone else"`)
	step(2, `test_leader.name="Malki"`, "walk:16,10")
	step(2.15, ``) // No walking request every tick.
	step(3, `test_leader.x=23; test_leader.distance=13`, "walk:19,10")
	step(3.15, `test_leader.distance=3`, "stop")
	step(4, `test_leader.distance=5`) // Hysteresis avoids constant start/stop.
	step(5, `test_leader.distance=8`, "walk:19,10")
	step(5.15, `test_players={}`)
	step(6, ``, "walk:23,10") // Finish approaching a disappearing leader.
	step(15.1, ``, "stop")    // Stop once the last observation is too old.
}

func TestCompanionFollowCommandSwitchesToSpeakerID(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_leader.x=20; test_leader.distance=10`, "walk:16,10")
	step(0.2, `
new_leader={id=3,name="Alice",x=30,y=10,distance=20,hp=0,max_hp=0,party_member=false,dead=false}
test_players={test_leader,new_leader}
chat({channel="public",sender_id=3,sender_name="Malki",text="  FoLLoW  "})
`, "Following you.", "stop")
	step(0.6, ``, "walk:26,10")
	step(1.7, `new_leader.name="Changed name"; new_leader.x=31`, "walk:27,10")
	step(2, `test_players={test_leader}`)
	step(2.8, ``, "walk:31,10") // Do not resume following the old configured leader.
	step(3.9, `test_players={test_leader,new_leader}`, "walk:27,10")
	step(4, `chat({channel="party",sender_id=2,text="follow"})`, "%Following you.", "stop")
	step(4.4, ``, "walk:16,10")
}

func TestCompanionFollowCommandWakesFromRest(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_me.sp=5`, "/sit")
	step(1, `chat({channel="public",sender_id=2,text="follow"})`, "Following you.", "/stand")
	step(1.6, `test_leader.x=20; test_leader.distance=10`, "walk:16,10")
}

func TestCompanionReachesLostLeaderCellInsideFollowDistance(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, ``)
	step(1, `test_players={}`, "walk:12,10")
	step(1.2, `test_me.x=12`, "stop")
	step(2, ``) // Reached the last observed cell; wait for the leader.
}

func TestCompanionFollowKeepsGapAndRecognizesArrival(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_leader.x=20; test_leader.y=20; test_leader.distance=math.sqrt(200)`, "walk:17,17")
	// The diagonal destination is over four cells away. Reaching it must
	// finish following, so support actions are not blocked forever.
	step(1.1, `test_me.x=17; test_me.y=17; test_me.sp=100; test_leader.distance=math.sqrt(18)`, "skill:AL_BLESSING:2:1")
}

func TestCompanionFollowTriesFartherCellWhenBlocked(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `
test_leader.x=20; test_leader.distance=10
local walk=goro.walk
goro.walk=function(x,y) local ok=walk(x,y); return ok and x~=16 end
`, "walk:16,10", "walk:15,10")
}

func TestCompanionFindsAlternativeFollowCell(t *testing.T) {
	for _, tc := range []struct {
		x, y    int
		blocked [2]worldstate.WalkStep
	}{
		{10, 10, [2]worldstate.WalkStep{{X: 17, Y: 17}, {X: 16, Y: 16}}},
		{30, 10, [2]worldstate.WalkStep{{X: 23, Y: 17}, {X: 24, Y: 16}}},
		{10, 30, [2]worldstate.WalkStep{{X: 17, Y: 23}, {X: 16, Y: 24}}},
		{30, 30, [2]worldstate.WalkStep{{X: 23, Y: 23}, {X: 24, Y: 24}}},
	} {
		t.Run(fmt.Sprintf("from_%d_%d", tc.x, tc.y), func(t *testing.T) {
			ctx, mode := newBotChatTestContext(t, companionScriptForTest(t))
			netClient, server := newBotTestConnection(t, 20080910)
			ctx.Network = netClient
			mode.bot.ctx = ctx
			ctx.World.SetPlayerPosition(tc.x, tc.y, 0)
			actor := ctx.World.Actors[300]
			actor.Name, actor.X, actor.Y = "Malki", 20, 20
			ctx.World.Actors[300] = actor
			for _, cell := range tc.blocked {
				ctx.World.GAT.Cells[cell.Y*ctx.World.GAT.Width+cell.X].Type = res.GATTypeNone
			}
			// Observe the accepted destination while using the real walk API.
			if err := mode.bot.state.DoString(`
local walk = goro.walk
function goro.walk(x, y)
    if not walk(x, y) then return false end
    chosen = {x=x, y=y}
    return true
end
`); err != nil {
				t.Fatal(err)
			}
			if err := mode.bot.tick(); err != nil {
				t.Fatal(err)
			}
			chosen, ok := mode.bot.state.GetGlobal("chosen").(*lua.LTable)
			if !ok {
				t.Fatal("both direct cells were blocked and the companion failed to try another destination")
			}
			x := int(chosen.RawGetString("x").(lua.LNumber))
			y := int(chosen.RawGetString("y").(lua.LNumber))
			if gap := math.Hypot(float64(x-20), float64(y-20)); gap < 4 || gap > 6 {
				t.Fatalf("alternative cell %d,%d violates follow distance: %v", x, y, gap)
			}
			if _, reachable := findWalkPath(ctx.World.GAT, tc.x, tc.y, x, y); !reachable {
				t.Fatalf("alternative cell %d,%d is unreachable", x, y)
			}
			walk, _ := network.BuildWalkToXYPacketForClientDate(x, y, 20080910)
			readBotTestPackets(t, server, walk)
			assertNoBotTestPacket(t, server, mode.bot.tick)
		})
	}
}

func TestCompanionBlockedFollowSearchIsBoundedAndThrottled(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `
test_leader.x=20; test_leader.y=20; test_leader.distance=math.sqrt(200)
attempts=0
goro.walk=function(x,y)
    assert((x-20)^2+(y-20)^2 >= 16, "fallback got too close")
    attempts=attempts+1
    return false
end
`)
	step(0.15, `assert(attempts > 2 and attempts < 100); initial_attempts=attempts`)
	step(0.3, `assert(attempts == initial_attempts, "retried before cooldown")`)
	step(1.1, ``)
	step(1.2, `assert(attempts == 2*initial_attempts, "search did not retry after cooldown")`)
}

func TestCompanionHealsOnRequestWithoutPartyOrHP(t *testing.T) {
	for _, channel := range []string{"public", "party"} {
		t.Run(channel, func(t *testing.T) {
			step := newCompanionScriptTest(t)
			reply := "Healing you."
			if channel == "party" {
				reply = "%" + reply
			}
			step(0, `test_leader.name="Alice"; test_leader.party_member=false; test_leader.hp=0; test_leader.max_hp=0;
chat({channel="`+channel+`",sender_id=2,text=" HEAL "})`, "skill:AL_HEAL:2:1", reply)
			step(3, ``)                                          // One request produces one heal.
			step(5, `test_leader.x=20; test_leader.distance=10`) // Healing does not change leader.
		})
	}
}

func TestCompanionStandsToHealNonCriticalPartyMember(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_me.sp=5`, "/sit")
	step(1, `test_me.sp=13; test_leader.hp=50`, "/stand")
	step(1.6, ``, "skill:AL_HEAL:2:1")
}

func TestCompanionRequestedHealWaitsForStandStopAndCast(t *testing.T) {
	t.Run("resting", func(t *testing.T) {
		step := newCompanionScriptTest(t)
		step(0, `test_me.sp=5`, "/sit")
		step(1, `test_me.sp=20; chat({channel="public",sender_id=2,text="heal"})`, "/stand")
		step(1.6, ``, "skill:AL_HEAL:2:1", "Healing you.")
	})
	t.Run("walking", func(t *testing.T) {
		step := newCompanionScriptTest(t)
		step(0, `test_leader.x=20; test_leader.distance=10`, "walk:16,10")
		step(0.2, `test_leader.x=17; test_leader.distance=7; chat({channel="public",sender_id=2,text="heal"})`, "stop")
		step(0.6, ``, "skill:AL_HEAL:2:1", "Healing you.")
	})
	t.Run("casting", func(t *testing.T) {
		step := newCompanionScriptTest(t)
		step(0, `test_me.hp=50`, "skill:AL_HEAL:1:1")
		step(0.1, `test_me.hp=100; chat({channel="public",sender_id=2,text="heal"})`)
		step(2.3, ``, "skill:AL_HEAL:2:1", "Healing you.")
	})
	t.Run("critical self", func(t *testing.T) {
		step := newCompanionScriptTest(t)
		step(0, `test_me.hp=20; chat({channel="public",sender_id=2,text="heal"})`, "skill:AL_HEAL:1:1")
		step(2.3, `test_me.hp=100`, "skill:AL_HEAL:2:1", "Healing you.")
	})
}

func TestCompanionReportsHealRequestProblems(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_me.sp=5; chat({channel="public",sender_id=2,text="heal"})`, "I need more SP to heal.", "/sit")
	step = newCompanionScriptTest(t)
	step(0, `test_leader.name="Alice"; test_leader.x=19; test_leader.distance=9;
chat({channel="public",sender_id=2,text="heal"})`, "Come a little closer for Heal.")
	step = newCompanionScriptTest(t)
	step(0, `test_reject.AL_HEAL=true; chat({channel="public",sender_id=2,text="heal"})`,
		"skill:AL_HEAL:2:1", "I couldn't cast Heal. Check my learned skill level.")
	step(3, `chat({channel="public",sender_id=2,text="heal"})`, "I can't cast Heal right now.")
}

func TestCompanionClearsHealRequestOnMapChangeOrExpiry(t *testing.T) {
	for _, tc := range []struct {
		name, reset string
		time        float64
	}{
		{"map change", `map_changed()`, 1},
		{"expiry", ``, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			step := newCompanionScriptTest(t)
			step(0, `test_me.hp=50`, "skill:AL_HEAL:1:1")
			step(0.1, `test_me.hp=100; chat({channel="public",sender_id=2,text="heal"})`)
			step(tc.time, tc.reset)
		})
	}
}

func TestCompanionDoesNotPursueDeadLeader(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_leader.x=20; test_leader.distance=10`, "walk:16,10")
	step(1, `test_leader.dead=true`, "stop")
	step(2, `test_players={}`)
}

func TestCompanionMapChangeClearsRestAndOldDestination(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_me.sp=5`, "/sit")
	step(1, `test_me.sp=20; test_players={}; map_changed()`)
	step(2, `test_players={test_leader}; test_leader.x=20; test_leader.distance=10`, "walk:16,10")
	step(3, `test_players={}; test_me.x=1; map_changed()`)
	step(4, ``)
}

func TestCompanionFollowCommandDoesNotInterruptCast(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_leader.hp=40`, "skill:AL_HEAL:2:1")
	step(0.2, `test_leader.x=20; test_leader.distance=10; chat({channel="public",sender_id=2,text="follow"})`, "Following you.")
	step(0.3, `chat({channel="public",sender_id=2,text="follow"})`) // Throttle acknowledgements.
	step(2.3, ``, "walk:16,10")
}

func TestCompanionIgnoresOtherCommandsAndUnavailableSpeakers(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_leader.name="Alice"; test_leader.x=20; test_leader.distance=10`)
	step(1, `chat({channel="public",sender_id=2,text="please follow"})`)
	step(2, `chat({channel="guild",sender_id=2,text="follow"})`)
	step(3, `chat({channel="public",sender_id=999,text="follow"})`)
	step(4, `chat({channel="public",sender_id=1,text="follow"})`)
	step(5, `test_leader.dead=true; chat({channel="public",sender_id=2,text="follow"})`)
	step(6, `test_leader.dead=false; test_me.dead=true; chat({channel="public",sender_id=2,text="follow"})`)
	step(7, `test_me.dead=false`)
}

func TestCompanionHealsByPriority(t *testing.T) {
	for _, tc := range []struct {
		name, setup, action string
	}{
		{"leader", `test_leader.hp=40`, "skill:AL_HEAL:2:1"},
		{"self", `test_me.hp=50`, "skill:AL_HEAL:1:1"},
		{"critical self", `test_me.hp=20; test_leader.hp=10`, "skill:AL_HEAL:1:1"},
		{"more injured leader", `test_me.hp=60; test_leader.hp=40`, "skill:AL_HEAL:2:1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			step := newCompanionScriptTest(t)
			step(0, tc.setup, tc.action)
			step(0.15, ``)
			step(2, ``) // Wait for cast/action delay and server HP updates.
			step(2.3, ``, tc.action)
			step(5, `test_me.hp=100; test_leader.hp=100`)
		})
	}
}

func TestCompanionDoesNotHealUnknownOrDeadPartyMembers(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_leader.hp=0; test_leader.max_hp=0`)
	step(0.5, `test_leader.max_hp=100`)
	step(1, `test_leader.dead=true`)
	step(2, `test_leader.dead=false; test_leader.hp=40; test_leader.distance=12; test_leader.x=22`, "walk:18,10")
	step(2.15, `test_leader.distance=7`, "stop")
	step(2.5, ``, "skill:AL_HEAL:2:1")
	step(2.65, `test_leader.distance=10`) // Following cannot interrupt this cast.
}

func TestCompanionUsesPotionBeforeEmergencyHeal(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_me.hp=20; test_inventory={{index=7,item_id=501,amount=2,usable=true}}`, "item:7")
	step(0.15, ``, "skill:AL_HEAL:1:1")
	step(0.3, ``)
	step(1, ``, "item:7")
	step(2.4, `test_me.hp=100`)
}

func TestCompanionRestsAndStandsForMovementOrDanger(t *testing.T) {
	for _, tc := range []struct{ name, wake string }{
		{"leader moving", `test_leader.distance=9`},
		{"nearby enemy", `test_enemies={{distance=4}}`},
		{"damage", `test_me.hp=90`},
		{"recovered", `test_me.sp=70`},
		{"emergency heal", `test_me.sp=13; test_leader.hp=20`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			step := newCompanionScriptTest(t)
			step(0, `test_me.sp=5`, "/sit")
			step(1, ``)
			step(2, tc.wake, "/stand")
		})
	}
	step := newCompanionScriptTest(t)
	step(0, `test_me.sp=5; test_enemies={{distance=4}}`)
	step(1, `test_enemies={}; test_me.hp=90`)
	step(5, ``) // Recent damage also makes sitting unsafe.
	step(6.1, ``, "/sit")
}

func TestCompanionUnavailableHealDoesNotBlockFollowingOrRecovery(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_reject.AL_HEAL=true; test_leader.hp=40`, "skill:AL_HEAL:2:1")
	step(0.15, ``)
	step(1, ``)
	step(2, `test_leader.distance=10; test_leader.x=20`, "walk:16,10")
	step(3, `test_leader.distance=3; test_me.hp=50`, "stop")
	step(3.5, ``) // Wait until it is safe to rest after the HP loss.
	step(8.1, ``, "/sit")
}

func TestCompanionBuffTimersAndSPReserve(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_me.sp=100`, "skill:AL_BLESSING:2:1")
	step(0.15, ``)
	step(2.3, ``, "skill:AL_INCAGI:2:1")
	step(4.6, ``, "skill:AL_BLESSING:1:1")
	step(6.9, ``, "skill:AL_INCAGI:1:1")
	step(10, ``)
	step(54, ``)
	step(55.1, ``, "skill:AL_BLESSING:2:1")

	step = newCompanionScriptTest(t)
	step(0, `test_me.sp=30; test_me.max_sp=40`) // Neither buff leaves SP for Heal.
	step(1, `test_me.sp=41`, "skill:AL_BLESSING:2:1")
}

func TestCompanionBuffsAvoidDangerAndAgiHPCost(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_me.sp=100; test_me.hp=20; test_me.max_hp=20; test_enemies={{distance=4}}`)
	step(1, `test_enemies={}`, "skill:AL_BLESSING:2:1")
	step(3.3, ``, "skill:AL_BLESSING:1:1")
	step(5.6, ``) // Increase AGI would consume most of our HP.
}

func TestCompanionDeathClearsBuffTimers(t *testing.T) {
	step := newCompanionScriptTest(t)
	step(0, `test_me.sp=100`, "skill:AL_BLESSING:2:1")
	step(0.2, `test_me.dead=true; test_me.hp=0`)
	step(5, ``)
	step(6, `test_me.dead=false; test_me.hp=100`, "skill:AL_BLESSING:2:1")
}

func TestCompanionScriptUsesExistingLuaAPI(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		selfHP, selfSP, allyHP int
		skill                  uint16
		target                 uint32
	}{
		{"heal party member", 100, 20, 40, db.SkillALHeal, 300},
		{"heal self first", 20, 20, 10, db.SkillALHeal, 2000000},
		{"buff party member", 100, 100, 100, db.SkillALBlessing, 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			networkClient, server := newBotTestConnection(t, 20080910)
			sess := session.New()
			sess.AccountID = 2000000
			sess.Vitals = session.Vitals{HP: tc.selfHP, MaxHP: 100, SP: tc.selfSP, MaxSP: 100}
			sess.Skills.List = []session.Skill{
				{ID: db.SkillALHeal, Type: skillTargetFriend, Level: 7, Range: 9},
				{ID: db.SkillALBlessing, Type: skillTargetFriend, Level: 10, Range: 9},
			}
			sess.Party.Members = []session.PartyMember{{AccountID: 300, Name: "Malki", HP: tc.allyHP, MaxHP: 100}}
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: sess.AccountID, X: 10, Y: 10, Job: db.JobAcolyte}
			world.Actors[300] = worldstate.Actor{
				ID: 300, Name: "Malki", X: 12, Y: 10, Job: db.JobAcolyte,
				ObjectType: actorObjectTypePC, HasObjectType: true,
			}
			bot, err := newLuaBot(client.Context{Session: sess, Network: networkClient, World: world},
				&WorldMode{}, companionScriptForTest(t))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(bot.close)
			if err := bot.tick(); err != nil {
				t.Fatal(err)
			}
			readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(tc.skill, 1, tc.target, 20080910))
			assertNoBotTestPacket(t, server, bot.tick)
		})
	}
}
