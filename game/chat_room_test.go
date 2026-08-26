package game

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestHandleChatRoomCreateAckKeepsSuccessOutOfRoomTranscript(t *testing.T) {
	mode := &WorldMode{
		pendingChatRoom: network.ChatRoomCreate{Title: "Room", Limit: 20, Public: true},
	}
	ctx := client.Context{
		Session:   &session.Session{CharID: 1, Selected: session.Character{ID: 1, Name: "Kivutar"}},
		UIManager: &worldModeTestUIManager{},
	}

	mode.handleChatRoomCreateAck(ctx, network.ChatRoomCreateAck{Result: 0})

	if !mode.ui.chatRoom.IsOpen() {
		t.Fatal("chat room was not opened")
	}
	if lines := chatRoomLineTexts(t, mode); len(lines) != 0 {
		t.Fatalf("chat room lines = %q, want an empty transcript", lines)
	}
	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Chat room has been created." {
		t.Fatalf("console messages = %#v", messages)
	}
}

func TestHandleChatMessageRoutesToOpenChatRoomOnly(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{
		UIManager: &worldModeTestUIManager{},
	}
	mode.ui.chatRoom.Open(ctx, "Room", 20, true, []string{"Kivutar"})

	mode.handleChatMessage(ctx, network.ChatMessage{
		Text: "Kivutar : hello",
	}, time.Now())

	lines := chatRoomLineTexts(t, mode)
	if len(lines) != 1 {
		t.Fatalf("chat room lines = %d, want 1", len(lines))
	}
	if lines[0] != "Kivutar : hello" {
		t.Fatalf("chat room line = %q", lines[0])
	}
	if len(mode.speechBubbles) != 0 {
		t.Fatalf("speech bubbles = %d, want 0", len(mode.speechBubbles))
	}
}

func TestHandleAnnouncementBypassesOpenChatRoom(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{UIManager: &worldModeTestUIManager{}}
	mode.ui.chatRoom.Open(ctx, "Room", 20, true, []string{"Kivutar"})
	now := time.Now()
	mode.handleChatMessage(ctx, network.ChatMessage{
		Text:         "The Emperium has been destroyed.",
		Color:        0x0000FFFF,
		HasColor:     true,
		Announcement: true,
		FontType:     700,
		FontSize:     20,
	}, now)

	if lines := chatRoomLineTexts(t, mode); len(lines) != 0 {
		t.Fatalf("announcement entered chat-room transcript: %q", lines)
	}
	if !mode.ui.announcement.Visible(now) {
		t.Fatal("announcement overlay was not shown")
	}
	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "The Emperium has been destroyed." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func chatRoomLineTexts(t *testing.T, mode *WorldMode) []string {
	t.Helper()
	lines := reflect.ValueOf(&mode.ui.chatRoom).Elem().FieldByName("lines")
	out := make([]string, 0, lines.Len())
	for i := 0; i < lines.Len(); i++ {
		out = append(out, lines.Index(i).FieldByName("text").String())
	}
	return out
}
