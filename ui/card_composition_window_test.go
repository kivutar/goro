package ui

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func cardCompositionSelectionFixture() (*CardCompositionWindow, *session.Session) {
	s := &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
		{Index: 2, ItemID: 2301}, {Index: 3, ItemID: 2302}, {Index: 4, ItemID: 2303},
	}}}
	w := &CardCompositionWindow{indexes: []uint16{4, 2, 3}, cardIndex: 5, selectedIndex: 3}
	return w, s
}

func TestCardCompositionSelectionFollowsInventoryIndex(t *testing.T) {
	w, s := cardCompositionSelectionFixture()
	if row := w.selectedRow(w.items(s)); row != 1 {
		t.Fatalf("initial selected row = %d, want 1", row)
	}

	s.Inventory.Items = s.Inventory.Items[1:]
	w.ClampScroll(s)
	if w.selectedIndex != 3 || w.selectedRow(w.items(s)) != 0 {
		t.Fatalf("selection after earlier removal = index %d, row %d; want index 3, row 0", w.selectedIndex, w.selectedRow(w.items(s)))
	}
}

func TestCardCompositionSelectionClearsWhenItemDisappears(t *testing.T) {
	for _, beforeUpdate := range []bool{false, true} {
		t.Run(map[bool]string{false: "update", true: "confirm_before_update"}[beforeUpdate], func(t *testing.T) {
			w, s := cardCompositionSelectionFixture()
			s.Inventory.Items = append(s.Inventory.Items[:1], s.Inventory.Items[2:]...)
			if beforeUpdate {
				w.composeSelected(Context{Session: s})
			} else {
				w.ClampScroll(s)
			}
			if w.selectedIndex != 0 || w.selectedRow(w.items(s)) != -1 {
				t.Fatalf("removed item left selection = index %d, row %d", w.selectedIndex, w.selectedRow(w.items(s)))
			}
		})
	}
}

func TestCardCompositionConfirmsOriginalItemAfterEarlierRemoval(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	client := network.NewClient(20080827, false)
	t.Cleanup(client.Close)
	if err := client.Connect(context.Background(), "127.0.0.1", listener.Addr().(*net.TCPAddr).Port); err != nil {
		t.Fatal(err)
	}
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	w, s := cardCompositionSelectionFixture()
	s.Inventory.Items = s.Inventory.Items[1:]
	// Confirm can run before the next frame has refreshed the table.
	w.composeSelected(Context{Session: s, Network: client})
	want := network.BuildItemCompositionPacket(5, 3)
	got := make([]byte, len(want))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("composition packet = %x, want %x (equipment index 3)", got, want)
	}
}
