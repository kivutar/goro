package game

import (
	"encoding/binary"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestRemoteActorUpdatePreservesExistingMovement(t *testing.T) {
	for _, elapsed := range []time.Duration{time.Second, 30 * time.Second} {
		t.Run(elapsed.String(), func(t *testing.T) {
			started := time.Unix(100, 0)
			w := worldstate.New()
			ctx := client.Context{World: w}
			actors.UpsertActorAt(ctx, worldstate.Actor{
				ID: 300, X: 30, Y: 20, FromX: 10, FromY: 20, ToX: 30, ToY: 20,
				Moving: true, Speed: 150,
			}, started)
			before := w.Actors[300]
			update := before
			update.Dir = 4
			update.EffectState = db.EffectStateFalcon
			update.HasState = true
			actors.UpsertActorAt(ctx, update, started.Add(elapsed))
			if got := w.Actors[300]; !reflect.DeepEqual(got, update) {
				t.Fatalf("state update changed movement: started=%v, want %v; actor=%+v", got.MoveStarted, before.MoveStarted, got)
			}

			// A subsequent movement packet must still start a new path.
			now := started.Add(elapsed)
			actors.UpsertNetworkActor(ctx, network.ActorEntry{
				ID: 300, X: 30, Y: 25, FromX: 30, FromY: 20, ToX: 30, ToY: 25,
				Moving: true, Speed: 150,
			})
			actor := w.Actors[300]
			if !actor.MoveStarted.After(now) || actor.FromX != 30 || actor.FromY != 20 || actor.ToY != 25 {
				t.Fatalf("new movement was not initialized: %+v", actor)
			}
		})
	}
}

func TestSkillPacketsDoNotReplayCompletedCasterWalk(t *testing.T) {
	for _, tc := range []struct {
		name    string
		skillID uint16
		casting bool
		ground  bool
	}{
		{name: "Heal", skillID: db.SkillALHeal},
		{name: "Blessing", skillID: db.SkillALBlessing},
		{name: "Increase AGI cast", skillID: db.SkillALIncagi, casting: true},
		{name: "ground cast", skillID: db.SkillMGFirewall, casting: true, ground: true},
	} {
		for _, synced := range []bool{false, true} {
			name := tc.name + "/without server clock"
			if synced {
				name = tc.name + "/with server clock"
			}
			t.Run(name, func(t *testing.T) {
				now := time.Now()
				sess := &session.Session{AccountID: 100, CharID: 101}
				if synced {
					sess.SyncServerTick(31000, now)
				}
				w := worldstate.New()
				w.Player = worldstate.Actor{ID: 100, X: 34, Y: 20}
				ctx := client.Context{World: w, Session: sess}
				actors.UpsertActorAt(ctx, worldstate.Actor{
					ID: 300, Job: db.JobAcolyte, Appearance: true,
					X: 30, Y: 20, FromX: 10, FromY: 20, ToX: 30, ToY: 20,
					Moving: true, Speed: 150, MoveStartTick: 1000, HasMoveStartTick: true,
				}, now.Add(-30*time.Second))
				before := w.Actors[300]
				if x, y := scene.ActorRenderPosition(before, now); x != 30 || y != 20 {
					t.Fatalf("setup: caster at %v,%v, want completed walk at 30,20", x, y)
				}
				packet := network.Packet{ID: 0x011A, Data: make([]byte, 15)}
				if tc.casting {
					packet = network.Packet{ID: 0x013E, Data: make([]byte, 24)}
					binary.LittleEndian.PutUint32(packet.Data[2:], 300)
					if tc.ground {
						binary.LittleEndian.PutUint16(packet.Data[10:], 34)
						binary.LittleEndian.PutUint16(packet.Data[12:], 20)
					} else {
						binary.LittleEndian.PutUint32(packet.Data[6:], 100)
					}
					binary.LittleEndian.PutUint16(packet.Data[14:], tc.skillID)
					binary.LittleEndian.PutUint32(packet.Data[20:], 1000)
				} else {
					binary.LittleEndian.PutUint16(packet.Data[2:], tc.skillID)
					binary.LittleEndian.PutUint16(packet.Data[4:], 100)
					binary.LittleEndian.PutUint32(packet.Data[6:], 100)
					binary.LittleEndian.PutUint32(packet.Data[10:], 300)
					packet.Data[14] = 1
				}
				binary.LittleEndian.PutUint16(packet.Data, packet.ID)
				mode := NewWorldMode()
				mode.handleNetworkPacket(ctx, packet, now)
				actor := w.Actors[300]
				if x, y := scene.ActorRenderPosition(actor, time.Now()); x != 30 || y != 20 {
					t.Fatalf("skill teleported caster to %v,%v; want 30,20", x, y)
				}
				if actor.Dir != actors.DirectionFromDelta(30, 20, 34, 20, before.Dir) {
					t.Fatal("caster did not turn toward the skill target")
				}
				if _, ok := mode.actorSystem.ActorAnims[300]; !ok {
					t.Fatal("skill animation did not start")
				}
			})
		}
	}
}

func TestWalkPathBuildsPathAroundObstacle(t *testing.T) {
	gat := testutil.PathGAT(3, 3, map[worldstate.WalkStep]bool{
		{X: 1, Y: 0}: true,
	})

	path := maps.WalkPath(gat, 0, 0, 2, 0)
	if len(path) < 4 {
		t.Fatalf("path too short: %+v", path)
	}
	for _, step := range path {
		if step.X == 1 && step.Y == 0 {
			t.Fatalf("path crosses blocked cell: %+v", path)
		}
	}
	w := worldstate.New()
	setPlayerMovementPathAt(client.Context{World: w}, path, 0, 0, 2, 0, 0, time.Now(), 0)
	if got := w.Player.MoveDuration; got <= 300*time.Millisecond {
		t.Fatalf("duration = %s, want longer than direct two-cell move", got)
	}
}

func TestWalkPathStoresFinalSegmentDirection(t *testing.T) {
	gat := testutil.PathGAT(3, 3, map[worldstate.WalkStep]bool{
		{X: 1, Y: 0}: true,
	})
	path := maps.WalkPath(gat, 0, 0, 2, 0)
	w := worldstate.New()
	setPlayerMovementPathAt(client.Context{World: w}, path, 0, 0, 2, 0, actors.DirectionFromDelta(0, 0, 2, 0, 0), time.Now(), 0)

	if got, want := w.Player.Dir, actors.DirectionFromDelta(2, 1, 2, 0, 0); got != want {
		t.Fatalf("player dir = %d, want final segment dir %d; path=%+v", got, want, w.Player.MovePath)
	}
	if w.Dir != w.Player.Dir {
		t.Fatalf("world dir = %d, want player dir %d", w.Dir, w.Player.Dir)
	}
}

func TestRemoteMoveUpdateUsesExistingActorSpeed(t *testing.T) {
	now := time.Unix(100, 0)
	w := worldstate.New()
	w.Actors[300] = worldstate.Actor{ID: 300, X: 0, Y: 0, Speed: 500}

	actors.UpsertActorAt(client.Context{World: w}, worldstate.Actor{
		ID:     300,
		X:      4,
		Y:      0,
		FromX:  0,
		FromY:  0,
		ToX:    4,
		ToY:    0,
		Moving: true,
	}, now)

	actor := w.Actors[300]
	if actor.Speed != 500 {
		t.Fatalf("speed = %d, want preserved 500", actor.Speed)
	}
	if got, want := actor.MoveDuration, 2*time.Second; got != want {
		t.Fatalf("move duration = %s, want %s", got, want)
	}
}

func TestRemoteMoveUpdateUsesServerMoveStartTick(t *testing.T) {
	s := &session.Session{}
	now := time.Unix(100, 0)
	s.SyncServerTick(1500, now)
	w := worldstate.New()
	w.Actors[300] = worldstate.Actor{ID: 300, X: 0, Y: 0, Speed: 500}

	actors.UpsertActorAt(client.Context{World: w, Session: s}, worldstate.Actor{
		ID:               300,
		X:                4,
		Y:                0,
		FromX:            0,
		FromY:            0,
		ToX:              4,
		ToY:              0,
		MoveStartTick:    500,
		HasMoveStartTick: true,
		Moving:           true,
	}, now)

	actor := w.Actors[300]
	if got, want := actor.MoveStarted, now.Add(-time.Second); !got.Equal(want) {
		t.Fatalf("move started = %s, want %s", got, want)
	}
	x, y := scene.ActorRenderPosition(actor, now)
	if x < 1.9 || x > 2.1 || y != 0 {
		t.Fatalf("render position = %.2f,%.2f, want around 2,0", x, y)
	}
}

func TestSpeedUpdateDoesNotRewriteActiveWalk(t *testing.T) {
	w := worldstate.New()
	w.Player = worldstate.Actor{ID: 100, X: 10, Y: 20, Speed: 150}
	w.GAT = testutil.FlatWalkableGAT(64, 64)
	ctx := client.Context{World: w, Session: &session.Session{}}
	now := time.Now()
	setPlayerMovementAt(ctx, 10, 20, 20, 20, 0, now, 0)
	beforeX, beforeY := scene.ActorRenderPosition(w.Player, now.Add(600*time.Millisecond))
	applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusSpeed, Value: 300})
	afterX, afterY := scene.ActorRenderPosition(w.Player, now.Add(600*time.Millisecond))
	if beforeX != afterX || beforeY != afterY || w.Player.MoveDuration != 1500*time.Millisecond {
		t.Fatal("speed update changed the position or finish time of an active path")
	}
	w.SetPlayerPosition(20, 20, 0)
	setPlayerMovementAt(ctx, 20, 20, 22, 20, 0, now.Add(2*time.Second), 0)
	if w.Player.MoveDuration != 600*time.Millisecond {
		t.Fatal("new walk did not use the updated speed")
	}
}

func TestImpossibleMovementDoesNotCrossWall(t *testing.T) {
	w := worldstate.New()
	w.Player = worldstate.Actor{ID: 100, X: 0, Y: 1, Speed: 150}
	w.GAT = testutil.PathGAT(3, 3, map[worldstate.WalkStep]bool{{X: 1, Y: 0}: true, {X: 1, Y: 1}: true, {X: 1, Y: 2}: true})
	ctx := client.Context{World: w, Session: &session.Session{}}
	setPlayerMovementAt(ctx, 0, 1, 2, 1, 0, time.Now(), 0)
	if w.Player.Moving || w.Player.X != 0 {
		t.Fatal("failed pathfinding invented a route through the wall")
	}
	actors.UpsertActor(ctx, worldstate.Actor{ID: 300, Moving: true, FromX: 0, FromY: 1, ToX: 2, ToY: 1, X: 2, Y: 1})
	if a := w.Actors[300]; a.Moving || a.X != 0 {
		t.Fatal("remote actor crossed the wall")
	}
}

func TestLongAcknowledgedWalkMovesLocalAndRemoteActors(t *testing.T) {
	w := worldstate.New()
	w.GAT = testutil.FlatWalkableGAT(64, 64)
	w.Player = worldstate.Actor{ID: 100, X: 10, Y: 10, Speed: 150}
	ctx := client.Context{World: w, Session: &session.Session{AccountID: 100}}
	applySelfMoveAck(ctx, network.SelfMoveAck{FromX: 10, FromY: 10, ToX: 42, ToY: 42})
	actors.UpsertActor(ctx, worldstate.Actor{ID: 300, Moving: true, FromX: 10, FromY: 10, ToX: 42, ToY: 42, X: 42, Y: 42, Speed: 150})
	for _, actor := range []worldstate.Actor{w.Player, w.Actors[300]} {
		if !actor.Moving || len(actor.MovePath) != 33 {
			t.Fatalf("actor %d did not follow the acknowledged 32-step route: %+v", actor.ID, actor)
		}
		x, y := scene.ActorRenderPosition(actor, actor.MoveStarted.Add(actor.MoveDuration/2))
		if x < 25.9 || x > 26.1 || y < 25.9 || y > 26.1 {
			t.Fatalf("actor %d halfway position = %f,%f, want 26,26", actor.ID, x, y)
		}
	}
}
