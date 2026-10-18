package game

import (
	"bytes"
	"context"
	"io"
	"math"
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestWarpClickWalksTowardWarpActor(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := ln.Accept()
		accepted <- conn
	}()

	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	addr := ln.Addr().(*net.TCPAddr)
	if err := netClient.Connect(context.Background(), addr.IP.String(), addr.Port); err != nil {
		t.Fatal(err)
	}
	serverConn := <-accepted
	if serverConn == nil {
		t.Fatal("server did not accept test client")
	}
	defer serverConn.Close()

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	warp := worldstate.Actor{
		ID:            900,
		X:             14,
		Y:             20,
		Job:           actorJobWarpPortal,
		ObjectType:    actorObjectTypeNPC,
		HasObjectType: true,
	}
	world.UpsertActor(warp)

	inputState := input.NewState()
	mode := &WorldMode{}
	ctx := client.Context{
		Input:   inputState,
		Network: netClient,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	now := time.Now()
	projection := mode.sceneProjection(ctx, ctx.ScreenW, ctx.ScreenH, now)
	point := projection.Project(cellCenter(float64(warp.X)), cellCenter(float64(warp.Y)), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 8)
	if _, err := io.ReadFull(serverConn, packet); err != nil {
		t.Fatalf("reading warp walk request: %v", err)
	}
	want, ok := network.BuildWalkToXYPacketForClientDate(13, 20, 20080910)
	if !ok {
		t.Fatal("building expected walk request")
	}
	if !bytes.Equal(packet, want) {
		t.Fatalf("warp walk packet = % x, want % x", packet, want)
	}
}

func TestWarpWalkTargetUsesWalkableNeighborWhenCenterIsBlocked(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	world.GAT = flatWalkableGAT(64, 64)
	world.GAT.SetCellRawType(14, 20, 1)
	ctx := client.Context{World: world}

	x, y, ok := warpWalkTarget(ctx, worldstate.Actor{X: 14, Y: 20, Job: actorJobWarpPortal}, time.Now())
	if !ok {
		t.Fatal("expected reachable warp approach cell")
	}
	if x != 13 || y != 20 {
		t.Fatalf("warp approach = %d,%d, want 13,20", x, y)
	}
}
