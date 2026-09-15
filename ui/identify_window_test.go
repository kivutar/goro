package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/gogpu/ui/uitest"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestIdentifySendsOneSelectionAndCloses(t *testing.T) {
	for _, count := range []int{1, 3} {
		t.Run(fmt.Sprintf("%d_items", count), func(t *testing.T) {
			ctx, manager, app := newWindowInstanceTest()
			inventoryContext, indexes := itemDialogBenchmarkContext(count)
			ctx.Session = inventoryContext.Session
			client, server := newIdentifyTestConnection(t)
			ctx.Network = client
			var w IdentifyWindow
			w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
			w.selected = ctx.Session.Inventory.Items[count-1]
			app.Frame()
			app.Window().DrawTo(&uitest.MockCanvas{})
			buttons := collectEscapeMenuButtons(w.content)
			if len(buttons) != 3 {
				t.Fatalf("buttons = %d, want Close, Cancel and OK", len(buttons))
			}
			point := buttons[2].ScreenBounds().Center()
			app.Window().HandleEvent(uitest.Click(point.X, point.Y))
			app.Window().HandleEvent(uitest.Release(point.X, point.Y))
			readIdentifyTestPacket(t, server, network.BuildItemIdentifyPacket(indexes[count-1]))
			if w.IsOpen() || len(manager.overlays) != 0 || w.content != nil {
				t.Fatal("successful submission left the picker open or published")
			}
			for _, item := range ctx.Session.Inventory.Items {
				if item.Identified {
					t.Fatal("submission identified an item before the server acknowledgement")
				}
			}

			// Neither another click nor retained callbacks may send from the closed picker.
			app.Window().HandleEvent(uitest.Click(point.X, point.Y))
			app.Window().HandleEvent(uitest.Release(point.X, point.Y))
			w.identifySelected(ctx)
			w.cancel(ctx)
			if w.Update(ctx) {
				t.Fatal("closed picker consumed input")
			}
			assertNoIdentifyTestPackets(t, client, server)
		})
	}
}

func TestIdentifyRequiresSelection(t *testing.T) {
	ctx, indexes := itemDialogBenchmarkContext(2)
	client, server := newIdentifyTestConnection(t)
	ctx.Network = client
	var w IdentifyWindow
	w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
	w.identifySelected(ctx)
	if !w.IsOpen() {
		t.Fatal("unselected picker closed")
	}
	assertNoIdentifyTestPackets(t, client, server)
}

func TestIdentifyRejectsMissingOrChangedSelection(t *testing.T) {
	for _, change := range []string{"removed", "replaced", "identified"} {
		for _, update := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_update_%t", change, update), func(t *testing.T) {
				ctx, indexes := itemDialogBenchmarkContext(2)
				client, server := newIdentifyTestConnection(t)
				ctx.Network = client
				var w IdentifyWindow
				w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
				w.selected = ctx.Session.Inventory.Items[0]
				if change == "removed" {
					ctx.Session.Inventory.Items = ctx.Session.Inventory.Items[1:]
				} else if change == "identified" {
					// A late ACK for an earlier request can identify an item shown here.
					ctx.Session.Inventory.Items[0].Identified = true
				} else {
					ctx.Session.Inventory.Items[0].Refine = 7
				}
				if update {
					w.Update(ctx)
				}
				w.identifySelected(ctx)
				if !w.IsOpen() || w.selected != (session.InventoryItem{}) {
					t.Fatal("missing or changed selection was not cleared without submitting")
				}
				assertNoIdentifyTestPackets(t, client, server)
			})
		}
	}
}

func TestIdentifySelectionFollowsItemAfterEarlierRemoval(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprintf("update_%t", update), func(t *testing.T) {
			ctx, indexes := itemDialogBenchmarkContext(2)
			client, server := newIdentifyTestConnection(t)
			ctx.Network = client
			var w IdentifyWindow
			w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
			w.selected = ctx.Session.Inventory.Items[1]
			ctx.Session.Inventory.Items = ctx.Session.Inventory.Items[1:]
			if update {
				w.Update(ctx)
			}
			if row := w.selectedRow(w.items(ctx.Session)); row != 0 {
				t.Fatalf("selected row after earlier removal = %d, want 0", row)
			}
			w.identifySelected(ctx)
			readIdentifyTestPacket(t, server, network.BuildItemIdentifyPacket(indexes[1]))
			if w.IsOpen() {
				t.Fatal("valid selection was lost when an earlier item disappeared")
			}
		})
	}
}

func TestIdentifySendFailureKeepsSelectionForRetry(t *testing.T) {
	for _, disconnected := range []bool{false, true} {
		t.Run(fmt.Sprintf("disconnected_client_%t", disconnected), func(t *testing.T) {
			ctx, indexes := itemDialogBenchmarkContext(2)
			if disconnected {
				ctx.Network = network.NewClient(20080910, false)
			}
			var w IdentifyWindow
			w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
			w.selected = ctx.Session.Inventory.Items[1]
			content := w.content
			w.identifySelected(ctx)
			if !w.IsOpen() || w.content != content || w.selected != ctx.Session.Inventory.Items[1] {
				t.Fatal("send failure closed or reset the picker")
			}
			client, server := newIdentifyTestConnection(t)
			ctx.Network = client
			w.identifySelected(ctx)
			readIdentifyTestPacket(t, server, network.BuildItemIdentifyPacket(indexes[1]))
			if w.IsOpen() {
				t.Fatal("successful retry left the picker open")
			}
		})
	}
}

func TestIdentifyCancelClosesWithoutAppraising(t *testing.T) {
	for _, action := range []string{"Cancel", "Close", "Escape"} {
		t.Run(action, func(t *testing.T) {
			ctx, manager, app := newWindowInstanceTest()
			inventoryContext, indexes := itemDialogBenchmarkContext(2)
			ctx.Session = inventoryContext.Session
			client, server := newIdentifyTestConnection(t)
			ctx.Network = client
			var w IdentifyWindow
			w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
			w.selected = ctx.Session.Inventory.Items[1]
			if action == "Escape" {
				ctx.Input.SetKey(input.KeyEscape, true)
				w.Update(ctx)
			} else {
				app.Frame()
				app.Window().DrawTo(&uitest.MockCanvas{})
				buttons := collectEscapeMenuButtons(w.content)
				if len(buttons) != 3 {
					t.Fatalf("buttons = %d, want 3", len(buttons))
				}
				buttonIndex := 0
				if action == "Cancel" {
					buttonIndex = 1
				}
				point := buttons[buttonIndex].ScreenBounds().Center()
				app.Window().HandleEvent(uitest.Click(point.X, point.Y))
				app.Window().HandleEvent(uitest.Release(point.X, point.Y))
			}
			readIdentifyTestPacket(t, server, network.BuildItemIdentifyPacket(identifyCancelIndex))
			if w.IsOpen() || len(manager.overlays) != 0 {
				t.Fatal("cancel left the picker open or published")
			}
			w.cancel(ctx)
			w.identifySelected(ctx)
			assertNoIdentifyTestPackets(t, client, server)
		})
	}
}

func TestIdentifyCancelSendFailureAllowsRetry(t *testing.T) {
	ctx, indexes := itemDialogBenchmarkContext(2)
	ctx.Network = network.NewClient(20080910, false)
	var w IdentifyWindow
	w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
	w.cancel(ctx)
	if !w.IsOpen() {
		t.Fatal("failed cancellation closed the picker")
	}
	client, server := newIdentifyTestConnection(t)
	ctx.Network = client
	w.cancel(ctx)
	readIdentifyTestPacket(t, server, network.BuildItemIdentifyPacket(identifyCancelIndex))
	if w.IsOpen() {
		t.Fatal("successful cancellation left the picker open")
	}
}

func TestIdentifyCancelWithoutNetworkClosesLocally(t *testing.T) {
	ctx, indexes := itemDialogBenchmarkContext(2)
	var w IdentifyWindow
	w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
	w.cancel(ctx)
	if w.IsOpen() {
		t.Fatal("cancel without a connection left the picker open")
	}
}

func newIdentifyTestConnection(t *testing.T) (*network.Client, net.Conn) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	client := network.NewClient(20080910, false)
	t.Cleanup(client.Close)
	if err := client.Connect(context.Background(), "127.0.0.1", listener.Addr().(*net.TCPAddr).Port); err != nil {
		t.Fatal(err)
	}
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return client, conn
}

func readIdentifyTestPacket(t *testing.T, conn net.Conn, want []byte) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("packet = %x, want %x", got, want)
	}
}

func assertNoIdentifyTestPackets(t *testing.T, client *network.Client, conn net.Conn) {
	t.Helper()
	// An ordered marker detects unexpected requests without a timeout-based wait.
	marker := []byte{0xAA, 0xBB}
	if err := client.Send(marker); err != nil {
		t.Fatal(err)
	}
	readIdentifyTestPacket(t, conn, marker)
}
