package game

import (
	"encoding/binary"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
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
			upsertActorAt(ctx, worldstate.Actor{
				ID: 300, X: 30, Y: 20, FromX: 10, FromY: 20, ToX: 30, ToY: 20,
				Moving: true, Speed: 150,
			}, started)
			before := w.Actors[300]
			update := before
			update.Dir = 4
			update.EffectState = db.EffectStateFalcon
			update.HasState = true
			upsertActorAt(ctx, update, started.Add(elapsed))
			if got := w.Actors[300]; !reflect.DeepEqual(got, update) {
				t.Fatalf("state update changed movement: started=%v, want %v; actor=%+v", got.MoveStarted, before.MoveStarted, got)
			}

			// A subsequent movement packet must still start a new path.
			now := started.Add(elapsed)
			upsertNetworkActor(ctx, network.ActorEntry{
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
				upsertActorAt(ctx, worldstate.Actor{
					ID: 300, Job: db.JobAcolyte, Appearance: true,
					X: 30, Y: 20, FromX: 10, FromY: 20, ToX: 30, ToY: 20,
					Moving: true, Speed: 150, MoveStartTick: 1000, HasMoveStartTick: true,
				}, now.Add(-30*time.Second))
				before := w.Actors[300]
				if x, y := actorRenderPosition(before, now); x != 30 || y != 20 {
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
				if x, y := actorRenderPosition(actor, time.Now()); x != 30 || y != 20 {
					t.Fatalf("skill teleported caster to %v,%v; want 30,20", x, y)
				}
				if actor.Dir != directionFromDelta(30, 20, 34, 20, before.Dir) {
					t.Fatal("caster did not turn toward the skill target")
				}
				if _, ok := mode.actorAnims[300]; !ok {
					t.Fatal("skill animation did not start")
				}
			})
		}
	}
}
