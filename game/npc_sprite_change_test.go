package game

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func npcSpriteChangeTestPacket(id, job uint32, packetType byte) network.Packet {
	data := make([]byte, 11)
	binary.LittleEndian.PutUint16(data, 0x01B0)
	binary.LittleEndian.PutUint32(data[2:], id)
	data[6] = packetType
	binary.LittleEndian.PutUint32(data[7:], job)
	return network.Packet{ID: 0x01B0, Data: data}
}

func TestNPCSpriteChangePreservesActorAndMovement(t *testing.T) {
	now := time.Now()
	before := worldstate.Actor{
		ID: 300, Job: 1008, Name: "Pupa", Appearance: true, X: 15, Y: 20, Dir: 6,
		ObjectType: actorObjectTypeMob, HasObjectType: true, Speed: 200,
		Moving: true, FromX: 10, FromY: 20, ToX: 15, ToY: 20,
		MoveStarted: now.Add(-time.Second), MoveDuration: 2 * time.Second,
		MoveStartX: 10.5, MoveStartY: 20.5, HasMoveStart: true,
		MovePath: []worldstate.WalkStep{{X: 10, Y: 20}, {X: 15, Y: 20}},
	}
	for _, login := range []bool{false, true} {
		for _, packetType := range []byte{0, 1, 2} {
			ctx := client.Context{World: worldstate.New(), Session: &session.Session{}}
			ctx.World.UpsertActor(before)
			want := ctx.World.Actors[before.ID]
			want.Job = 1018
			packet := npcSpriteChangeTestPacket(before.ID, 1018, packetType)
			if login {
				if !(&LoginMode{}).applyLoginActorBootstrapPacket(ctx, packet) {
					t.Fatal("login dropped transformation")
				}
			} else {
				(&WorldMode{}).handleNetworkPacket(ctx, packet, now)
			}
			if got := ctx.World.Actors[before.ID]; !reflect.DeepEqual(got, want) {
				t.Fatalf("login=%t type=%d: sprite change modified movement or failed: %+v", login, packetType, got)
			}
			// Later movement-only entries must retain the transformed class.
			upsertNetworkActor(ctx, network.ActorEntry{ID: before.ID, Moving: true, FromX: 15, FromY: 20, ToX: 16, ToY: 20, X: 16, Y: 20})
			if got := ctx.World.Actors[before.ID]; got.Job != 1018 || !got.Moving {
				t.Fatalf("move lost transformation: %+v", got)
			}
		}
	}
}

func TestNPCSpriteChangeIgnoresMissingActorsAndPlayers(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Session: &session.Session{AccountID: 100, CharID: 200, Selected: session.Character{ID: 200, Job: 1}}}
	ctx.World.Player = worldstate.Actor{ID: 200, Job: 1, Appearance: true}
	ctx.World.UpsertActor(worldstate.Actor{ID: 300, Job: 1, Appearance: true, ObjectType: actorObjectTypePC, HasObjectType: true})
	ctx.World.UpsertActor(worldstate.Actor{ID: 301, Job: 1, Appearance: true})
	mode := &WorldMode{}
	for _, id := range []uint32{0, 100, 200, 300, 301, 999} {
		mode.handleNetworkPacket(ctx, npcSpriteChangeTestPacket(id, 1018, 0), time.Now())
	}
	if len(ctx.World.Actors) != 2 || ctx.World.Player.Job != 1 || ctx.Session.Selected.Job != 1 || ctx.World.Actors[300].Job != 1 || ctx.World.Actors[301].Job != 1 {
		t.Fatal("NPC sprite change created an actor or changed a player")
	}
	// A class cannot wrap around the signed 16-bit job stored in the world.
	ctx.World.UpsertActor(worldstate.Actor{ID: 400, Job: 1008, Appearance: true})
	for _, job := range []uint32{0, 1, 32768, 0x10000 + 1018} {
		mode.handleNetworkPacket(ctx, npcSpriteChangeTestPacket(400, job, 0), time.Now())
		if ctx.World.Actors[400].Job != 1008 {
			t.Fatalf("accepted invalid NPC class %d", job)
		}
	}
}

func TestNPCSpriteChangeHandlesHiddenClasses(t *testing.T) {
	ctx := client.Context{World: worldstate.New()}
	ctx.World.UpsertActor(worldstate.Actor{ID: 300, Job: 1002, Appearance: true, ObjectType: actorObjectTypeNPC, HasObjectType: true})
	mode := &WorldMode{}
	for _, job := range []uint32{139, 111, 1002} {
		mode.handleNetworkPacket(ctx, npcSpriteChangeTestPacket(300, job, 1), time.Now())
		actor := ctx.World.Actors[300]
		if uint32(actor.Job) != job || actorJobHasNoSprite(int(actor.Job)) != (job != 1002) {
			t.Fatalf("class %d: %+v", job, actor)
		}
	}
}

func TestPupaTransformationsSwitchSpritesRealData(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Session: &session.Session{}, Resources: realDataManager(t)}
	mode := &WorldMode{}
	now := time.Now()
	// Keep one unchanged Pupa to catch accidental mutation of the shared view.
	for id := uint32(300); id <= 364; id++ {
		ctx.World.UpsertActor(worldstate.Actor{ID: id, Job: 1008, Appearance: true, X: 10, Y: 20, Speed: 200})
	}
	pupa := mode.nonPCSpriteView(ctx, ctx.World.Actors[300])
	if pupa == nil || !strings.Contains(strings.ToLower(pupa.source), "pupa.spr") {
		t.Fatal("Pupa resource did not load")
	}
	var stream []byte
	for id := uint32(301); id <= 364; id++ {
		stream = append(stream, npcSpriteChangeTestPacket(id, 1018, byte(id%2)).Data...)
		name := make([]byte, 30)
		binary.LittleEndian.PutUint16(name, 0x0095)
		binary.LittleEndian.PutUint32(name[2:], id)
		copy(name[6:], "Creamy")
		stream = append(stream, name...)
	}
	framer := network.NewFramer(network.PacketLengths2008())
	for len(stream) > 0 {
		n := min(7, len(stream))
		packets, err := framer.Push(stream[:n])
		stream = stream[n:]
		if err != nil {
			t.Fatal(err)
		}
		for _, packet := range packets {
			mode.handleNetworkPacket(ctx, packet, now)
		}
	}
	for id := uint32(301); id <= 364; id++ {
		upsertNetworkActor(ctx, network.ActorEntry{ID: id, Moving: true, FromX: 10, FromY: 20, ToX: 15, ToY: 20, X: 15, Y: 20})
		actor := ctx.World.Actors[id]
		view := mode.nonPCSpriteView(ctx, actor)
		if actor.Job != 1018 || actor.Name != "Creamy" || view == nil || view == pupa || !strings.Contains(strings.ToLower(view.source), "creamy.spr") {
			t.Fatalf("actor %d did not become Creamy: %+v", id, actor)
		}
		state := mode.nonPCSpriteState(actor, time.Now())
		if state.actionFamily != spriteActionWalk || !state.moving {
			t.Fatalf("Creamy lost movement: %+v", state)
		}
	}
	if mode.nonPCSpriteView(ctx, ctx.World.Actors[300]) != pupa {
		t.Fatal("transformation changed other Pupas' shared sprite")
	}
	t.Log("64 Pupas switched to Creamy sprites; unchanged Pupa retained its sprite")
}
