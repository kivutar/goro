package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

// Exercise headless login, bot updates, and map transitions without drawing.
func TestHeadlessLoginBotAndWarps(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := listener.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	port := uint16(listener.Addr().(*net.TCPAddr).Port)
	root := t.TempDir()
	writeHeadlessFixture(t, root, "clientinfo.xml", []byte(fmt.Sprintf(`<clientinfo><connection><address>127.0.0.1</address><port>%d</port><version>55</version></connection></clientinfo>`, port)))
	gat := make([]byte, 14+3*3*20)
	copy(gat, "GRAT")
	gat[4], gat[5] = 1, 2
	binary.LittleEndian.PutUint32(gat[6:10], 3)
	binary.LittleEndian.PutUint32(gat[10:14], 3)
	writeHeadlessFixture(t, root, "first.gat", gat)
	writeHeadlessFixture(t, root, "second.gat", gat)
	writeHeadlessFixture(t, root, "bot.lua", []byte(`
local sent = false
function tick()
    if not sent then
        assert(goro.walk(1, 0))
        assert(goro.message("headless"))
        sent = true
    end
end
`))
	cfg := config.Config{
		Headless: true, DataDir: root,
		Audio:  config.AudioConfig{Disabled: true},
		Window: config.WindowConfig{Width: 800, Height: 600},
		Packet: config.PacketConfig{ClientDate: 20080910},
		Login:  config.LoginConfig{AutoLogin: true, Username: "tester", Password: "test-password", CharSlot: 0},
		Script: config.ScriptConfig{Path: filepath.Join(root, "bot.lua")},
	}
	g, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var runErr error
	go func() {
		runErr = render.RunHeadless(ctx, g, cfg.Window)
		g.RequestQuit()
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("headless runner did not stop")
		}
	})
	accept := func() net.Conn {
		t.Helper()
		conn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	login := accept()
	expectHeadlessPacket(t, login, network.BuildAccountLoginPacket(network.AccountLogin{Username: "tester", Password: "test-password", Version: 55}))
	accepted := headlessPacket(0x0069, 79)
	binary.LittleEndian.PutUint16(accepted[2:4], uint16(len(accepted)))
	binary.LittleEndian.PutUint32(accepted[4:8], 100)
	binary.LittleEndian.PutUint32(accepted[8:12], 2000000)
	copy(accepted[47:51], []byte{127, 0, 0, 1})
	binary.LittleEndian.PutUint16(accepted[51:53], port)
	writeHeadlessPacket(t, login, accepted)
	character := accept()
	expectHeadlessPacket(t, character, network.BuildCharServerEnterPacket(network.CharServerEnter{AccountID: 2000000, AuthCode: 100}))
	list := headlessPacket(0x006b, 24+108)
	binary.LittleEndian.PutUint16(list[2:4], uint16(len(list)))
	binary.LittleEndian.PutUint32(list[24:28], 150000)
	binary.LittleEndian.PutUint16(list[24+42:24+44], 100)
	binary.LittleEndian.PutUint16(list[24+44:24+46], 100)
	copy(list[24+74:24+98], "Tester")
	writeHeadlessPacket(t, character, list)
	expectHeadlessPacket(t, character, network.BuildSelectCharacterPacket(0))
	zone := headlessPacket(0x0071, 28)
	binary.LittleEndian.PutUint32(zone[2:6], 150000)
	copy(zone[6:22], "first.gat")
	copy(zone[22:26], []byte{127, 0, 0, 1})
	binary.LittleEndian.PutUint16(zone[26:28], port)
	writeHeadlessPacket(t, character, zone)
	enterMap := func(conn net.Conn) {
		t.Helper()
		want := network.BuildMapServerEnterPacketForClientDate(network.MapServerEnter{}, cfg.Packet.ClientDate)
		got := readHeadlessPacket(t, conn, len(want))
		if network.ID(got) != network.ID(want) {
			t.Fatalf("map enter opcode = %04x, want %04x", network.ID(got), network.ID(want))
		}
		writeHeadlessPacket(t, conn, headlessPacket(0x0073, 11))
		expectHeadlessPacket(t, conn, network.BuildLoadEndAckPacket())
	}
	botAction := func(conn net.Conn) {
		t.Helper()
		move, ok := network.BuildWalkToXYPacketForClientDate(1, 0, cfg.Packet.ClientDate)
		if !ok {
			t.Fatal("invalid test destination")
		}
		expectHeadlessPacket(t, conn, move)
		expectHeadlessPacket(t, conn, network.BuildGlobalChatPacketForClientDate("Tester", "headless", cfg.Packet.ClientDate))
	}
	mapConn := accept()
	enterMap(mapConn)
	botAction(mapConn)
	warp := headlessPacket(0x0091, 22)
	copy(warp[2:18], "first.gat")
	writeHeadlessPacket(t, mapConn, warp)
	expectHeadlessPacket(t, mapConn, network.BuildLoadEndAckPacket())
	copy(warp[2:18], "second.gat")
	writeHeadlessPacket(t, mapConn, warp)
	expectHeadlessPacket(t, mapConn, network.BuildLoadEndAckPacket())
	botAction(mapConn)
	serverWarp := headlessPacket(0x0092, 28)
	copy(serverWarp[2:18], "first.gat")
	copy(serverWarp[22:26], []byte{127, 0, 0, 1})
	binary.LittleEndian.PutUint16(serverWarp[26:28], port)
	writeHeadlessPacket(t, mapConn, serverWarp)
	mapConn = accept()
	enterMap(mapConn)
	botAction(mapConn)
	cancel()
	select {
	case <-done:
		if runErr != nil {
			t.Fatalf("headless renderer result = %v", runErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("headless renderer did not stop")
	}
	if !g.quitting || g.network.Status() != "offline" || g.uiApp != nil || !g.cfg.Audio.Disabled {
		t.Fatal("headless session used presentation or was not closed")
	}
}

func TestHeadlessDrawRealMap(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to run against real RO assets")
	}
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	for _, headless := range []bool{false, true} {
		t.Run(fmt.Sprintf("headless=%t", headless), func(t *testing.T) {
			cfg := config.Config{
				Headless: headless, DataDir: root,
				Window: config.WindowConfig{Width: 800, Height: 600},
				Audio:  config.AudioConfig{Disabled: true},
			}
			g, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer g.RequestQuit()
			g.world.MapName = "prontera.gat"
			g.modes = game.NewManager(g.modeContext(), game.NewWorldMode())
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			if err := render.RunHeadless(ctx, g, cfg.Window); err != nil {
				t.Fatal(err)
			}
			if g.world.GAT == nil {
				t.Fatal("Prontera collision grid did not load")
			}
			if headless {
				if g.world.GND != nil || g.world.RSW != nil || len(g.world.RSM) != 0 {
					t.Fatal("headless mode loaded scenery")
				}
			} else if g.world.GND == nil || g.world.RSW == nil {
				t.Fatal("Prontera scenery did not load")
			}
		})
	}
}

func writeHeadlessFixture(t *testing.T, root, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func headlessPacket(id uint16, size int) []byte {
	packet := make([]byte, size)
	binary.LittleEndian.PutUint16(packet, id)
	return packet
}

func writeHeadlessPacket(t *testing.T, conn net.Conn, data []byte) {
	t.Helper()
	if err := conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(data); err != nil {
		t.Fatal(err)
	}
}

func readHeadlessPacket(t *testing.T, conn net.Conn, size int) []byte {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(conn, data); err != nil {
		t.Fatal(err)
	}
	return data
}

func expectHeadlessPacket(t *testing.T, conn net.Conn, want []byte) {
	t.Helper()
	if got := readHeadlessPacket(t, conn, len(want)); !bytes.Equal(got, want) {
		t.Fatalf("packet = %x, want %x", got, want)
	}
}
