package testutil

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/network"
)

func ReadPackets(t *testing.T, conn net.Conn, want []byte) {
	t.Helper()
	got := make([]byte, len(want))
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("packets = %x, want %x", got, want)
	}
}

func NewConnection(t *testing.T, clientDate int) (*network.Client, net.Conn) {
	t.Helper()
	// 2008 packets pad with clock-derived bytes; freeze them so expected
	// packets built after sending still match.
	t.Cleanup(network.FreezeLegacyPaddingClock(time.Unix(1700000000, 0)))
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- conn
	}()

	networkClient := network.NewClient(clientDate, false)
	port := listener.Addr().(*net.TCPAddr).Port
	if err := networkClient.Connect(context.Background(), "127.0.0.1", port); err != nil {
		listener.Close()
		t.Fatal(err)
	}

	var serverConn net.Conn
	select {
	case serverConn = <-accepted:
	case err := <-acceptErr:
		networkClient.Close()
		listener.Close()
		t.Fatal(err)
	case <-time.After(time.Second):
		networkClient.Close()
		listener.Close()
		t.Fatal("timed out accepting bot test connection")
	}
	if err := listener.Close(); err != nil {
		networkClient.Close()
		serverConn.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		networkClient.Close()
		serverConn.Close()
	})
	return networkClient, serverConn
}
