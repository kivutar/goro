package game

import (
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func TestWhisperWindowsRouteMessagesAndRespectRecipientSettings(t *testing.T) {
	ctx := client.Context{Input: input.NewState(), UIManager: gameui.NewManager(), Session: &session.Session{
		Whisper: session.WhisperSettings{Configured: true},
	}}
	mode := NewWorldMode()
	alice := mode.ui.whisperWindows.Open(ctx, "Alice")
	mode.addWhisperWindowIncoming(ctx, network.WhisperMessage{Sender: "Bob", Message: "uninvited"})
	if mode.ui.whisperWindows.Find("Bob") != nil {
		t.Fatal("Alice's open window bypassed Bob's popup preference")
	}
	mode.addWhisperWindowIncoming(ctx, network.WhisperMessage{Sender: "alice", Message: "for Alice"})
	if !strings.Contains(whisperWindowText(alice.Widget()), "for Alice") {
		t.Fatal("message to the existing conversation was dropped")
	}
	ctx.Session.Whisper.OpenStrangers = true
	mode.addWhisperWindowIncoming(ctx, network.WhisperMessage{Sender: "Bob", Message: "for Bob"})
	bob := mode.ui.whisperWindows.Find("Bob")
	if bob == nil || !bob.IsOpen() || strings.Contains(whisperWindowText(alice.Widget()), "for Bob") {
		t.Fatal("incoming conversation replaced or polluted Alice's window")
	}
	next := mode.nextWorldMode()
	next.rebindPersistentUI(ctx)
	if next.ui.whisperWindows.Find("Alice") != alice || next.ui.whisperWindows.Find("Bob") != bob {
		t.Fatal("map change lost conversation instances")
	}
	if NewWorldMode().ui.whisperWindows.IsOpen() {
		t.Fatal("fresh session inherited previous conversations")
	}
}

func TestWhisperSendsAndUnattributedAcknowledgements(t *testing.T) {
	netClient, server := newBotTestConnection(t, 20080910)
	ctx := client.Context{Network: netClient, Input: input.NewState(), UIManager: gameui.NewManager()}
	mode := NewWorldMode()
	alice := mode.ui.whisperWindows.Open(ctx, "Alice")
	bob := mode.ui.whisperWindows.Open(ctx, "Bob")
	mode.sendWhisperWindowMessage(ctx, gameui.WhisperWindowAction{Target: "Alice", Message: "first"})
	if err := client.SendChat(ctx, "/w Carol console"); err != nil {
		t.Fatal(err)
	}
	mode.sendWhisperWindowMessage(ctx, gameui.WhisperWindowAction{Target: "Bob", Message: "second"})
	readBotTestPackets(t, server, network.BuildWhisperPacket("Alice", "first"))
	readBotTestPackets(t, server, network.BuildWhisperPacket("Carol", "console"))
	readBotTestPackets(t, server, network.BuildWhisperPacket("Bob", "second"))
	aliceBefore := whisperWindowText(alice.Widget())
	bobBefore := whisperWindowText(bob.Widget())
	// Replies carry no recipient and need not follow send order. In particular,
	// a reply to a console send must never be assigned to an open conversation.
	for _, result := range []byte{1, 0, 2} {
		mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCAckWhisper, Data: []byte{0x98, 0x00, result}}, time.Now())
	}
	if whisperWindowText(alice.Widget()) != aliceBefore || whisperWindowText(bob.Widget()) != bobBefore {
		t.Fatal("unnamed acknowledgement was assigned to a conversation")
	}
	netClient.Close()
	mode.sendWhisperWindowMessage(ctx, gameui.WhisperWindowAction{Target: "Alice", Message: "offline"})
	if !strings.Contains(whisperWindowText(alice.Widget()), "send failed") || whisperWindowText(bob.Widget()) != bobBefore {
		t.Fatal("local send failure did not stay with its originating conversation")
	}
}

func whisperWindowText(root widget.Widget) string {
	if root == nil {
		return ""
	}
	// Window overlays expose their updated children after drawing the damaged
	// frame. Assert the text the player would see, after that render step.
	ctx := widget.NewContext()
	root.Layout(ctx, geometry.Loose(geometry.Sz(1024, 768)))
	root.Draw(ctx, &uitest.MockCanvas{})
	return whisperWidgetText(root)
}

func whisperWidgetText(root widget.Widget) string {
	text := ""
	if label, ok := root.(interface{ Content() string }); ok {
		text = label.Content() + "\n"
	}
	for _, child := range root.Children() {
		text += whisperWidgetText(child)
	}
	return text
}
