package game

import (
	"encoding/binary"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/ui/rotheme"
	worldstate "github.com/kivutar/goro/world"
)

func TestPartyCreationWaitsForServerMembership(t *testing.T) {
	for _, command := range []bool{false, true} {
		name := "window"
		if command {
			name = "organize command"
		}
		t.Run(name, func(t *testing.T) {
			netClient, conn := testutil.NewConnection(t, 20080910)
			s := &session.Session{AccountID: 123, Party: session.Party{RefuseInvites: true}}
			ctx := client.Context{Session: s, Network: netClient}
			mode := &WorldMode{}
			request := func() {
				t.Helper()
				if command {
					mode.ui.console.SendText(ctx, `/organize "Friends"`)
					readBotTestPackets(t, conn, network.BuildMakePartyPacket("Friends"))
				} else {
					mode.social().CreatePartyFromWindow(ctx, gameui.PartyCreateWindowAction{
						Name: "Friends", ItemPickup: 1, ItemDivision: 1,
					})
					readBotTestPackets(t, conn, network.BuildMakeParty2Packet("Friends", 1, 1))
				}
				if s.Party.Active() {
					t.Fatalf("request created unconfirmed party: %+v", s.Party)
				}
			}
			request()
			// Basic Skill below level 7 is rejected through ZC_ACK_TOUSESKILL,
			// without a party creation response.
			mode.applySkillFailAck(network.SkillFailAck{SkillID: 1, Number: 4})
			if s.Party.Active() {
				t.Fatalf("rejected party is active: %+v", s.Party)
			}
			if messages := mode.ui.console.Messages(); len(messages) != 1 || messages[0].Text != "Cannot form a party." {
				t.Fatalf("rejection messages = %+v", messages)
			}
			request() // A refused request must not block another attempt.
			mode.social().HandlePartyCreateResult(ctx, network.PartyCreateResult{Result: 0})
			social.ApplyPartyList(ctx, network.PartyList{
				Name: "Friends", Members: []network.PartyMember{{AccountID: 123, Name: "Alice"}},
			})
			if !s.Party.Active() || s.Party.Name != "Friends" || len(s.Party.Members) != 1 || s.Party.Members[0].AccountID != 123 {
				t.Fatalf("confirmed party = %+v", s.Party)
			}
			if !s.Party.RefuseInvites {
				t.Fatal("creation changed the invitation preference")
			}
		})
	}
}

func TestGuildNoticeEditorEnterDoesNotActivateConsole(t *testing.T) {
	mode := NewWorldMode()
	ctx := client.Context{
		Session: &session.Session{GuildID: 99, Guild: session.Guild{
			ID: 99, IsMaster: true, MenuAccess: 0x80, Notice: "Hello",
		}},
		World: worldstate.New(), Input: input.NewState(), Network: network.NewClient(20080910, false), ScreenW: 800, ScreenH: 600,
	}
	t.Cleanup(func() { ctx.Network.Close() })
	mode.ui.social.GuildWindow.OpenWindow(ctx)
	root := mode.ui.social.GuildWindow.Widget()
	wc := widget.NewContext()
	root.Layout(wc, geometry.Tight(geometry.Sz(800, 600)))
	root.Draw(wc, &uitest.MockCanvas{})
	// Open the final tab through the positioned widget tree, as a click would.
	tabs := root.Children()[0].Children()[1].Children()[0].Children()[0]
	notice := tabs.Children()[5].(interface{ ScreenBounds() geometry.Rect })
	p := notice.ScreenBounds().Center()
	root.Event(wc, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, p, p, event.ModNone))
	root.Event(wc, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, p, p, event.ModNone))
	root.Layout(wc, geometry.Tight(geometry.Sz(800, 600)))
	root.Draw(wc, &uitest.MockCanvas{})
	var field *rotheme.TextAreaWidget
	var findEditor func(widget.Widget)
	findEditor = func(w widget.Widget) {
		if editor, ok := w.(*rotheme.TextAreaWidget); ok {
			field = editor
		}
		for _, child := range w.Children() {
			findEditor(child)
		}
	}
	findEditor(mode.ui.social.GuildWindow.Widget())
	if field == nil {
		t.Fatalf("Notice tab has no multiline editor: tab=%T bounds=%v root=%v", notice, notice.ScreenBounds(), root.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds())
	}
	field.SetFocused(true)
	field.Event(wc, event.NewKeyEvent(event.KeyPress, event.KeyEnd, 0, event.ModCtrl))
	field.Event(wc, event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	ctx.Input.SetKey(input.KeyEnter, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.console.Active() || field.Text() != "Hello\n" || !mode.ui.keyboardInputBlocked() {
		t.Fatal("Enter in the guild notice did not stay in the multiline editor")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKey(input.KeyEnter, false)
	ctx.Input.SetKey(input.KeyEscape, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.social.GuildWindow.IsOpen() || mode.ui.escapeMenu.IsOpen() || mode.ui.console.Active() || field.IsFocused() {
		t.Fatal("Escape should close only the guild window and release editor focus")
	}
}

func TestSiegeGuildEmblemEligibilityAndPosition(t *testing.T) {
	entry := scene.ActorDrawEntry{
		Actor:   worldstate.Actor{GuildID: 10, EmblemVersion: 2},
		ScreenX: 100,
		ScreenY: 200,
		Scale:   1,
	}
	if !siegeActorShowsGuildEmblem(entry) {
		t.Fatal("visible guild actor did not qualify for a siege emblem")
	}
	entry.Actor.EffectState = db.EffectStateHide
	if siegeActorShowsGuildEmblem(entry) {
		t.Fatal("hidden actor qualified for a siege emblem")
	}
	entry.Actor.EffectState = db.EffectStateCloak
	if siegeActorShowsGuildEmblem(entry) {
		t.Fatal("cloaked actor qualified for a siege emblem")
	}
	entry.Actor.EffectState = 0
	x, y := siegeGuildEmblemPosition(entry.ScreenX, actors.ActorSpriteTopY(entry.ScreenY, entry.Scale), 24)
	if x != 88 || y != actors.ActorSpriteTopY(200, 1)-28 {
		t.Fatalf("siege emblem position = %.0f,%.0f", x, y)
	}
}

func TestAdoptionRequestPacketOpensConfirmation(t *testing.T) {
	data := make([]byte, 34)
	binary.LittleEndian.PutUint16(data[0:2], network.PacketZCReqBaby)
	binary.LittleEndian.PutUint32(data[2:6], 2000001)
	binary.LittleEndian.PutUint32(data[6:10], 2000000)
	copy(data[10:34], "Zambla")

	mode := &WorldMode{}
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCReqBaby, Data: data}, time.Now())

	if !mode.ui.social.AdoptionRequest.IsOpen() {
		t.Fatal("adoption request did not open a confirmation")
	}
}

func TestWhisperWindowsRouteMessagesAndRespectRecipientSettings(t *testing.T) {
	ctx := client.Context{Input: input.NewState(), UIManager: gameui.NewManager(), Session: &session.Session{
		Whisper: session.WhisperSettings{Configured: true},
	}}
	mode := NewWorldMode()
	alice := mode.ui.social.WhisperWindows.Open(ctx, "Alice")
	mode.social().AddWhisperWindowIncoming(ctx, network.WhisperMessage{Sender: "Bob", Message: "uninvited"})
	if mode.ui.social.WhisperWindows.Find("Bob") != nil {
		t.Fatal("Alice's open window bypassed Bob's popup preference")
	}
	mode.social().AddWhisperWindowIncoming(ctx, network.WhisperMessage{Sender: "alice", Message: "for Alice"})
	if !strings.Contains(testutil.DrawnWidgetText(alice.Widget()), "for Alice") {
		t.Fatal("message to the existing conversation was dropped")
	}
	ctx.Session.Whisper.OpenStrangers = true
	mode.social().AddWhisperWindowIncoming(ctx, network.WhisperMessage{Sender: "Bob", Message: "for Bob"})
	bob := mode.ui.social.WhisperWindows.Find("Bob")
	if bob == nil || !bob.IsOpen() || strings.Contains(testutil.DrawnWidgetText(alice.Widget()), "for Bob") {
		t.Fatal("incoming conversation replaced or polluted Alice's window")
	}
	next := mode.nextWorldMode()
	next.rebindPersistentUI(ctx)
	if next.ui.social.WhisperWindows.Find("Alice") != alice || next.ui.social.WhisperWindows.Find("Bob") != bob {
		t.Fatal("map change lost conversation instances")
	}
	if NewWorldMode().ui.social.WhisperWindows.IsOpen() {
		t.Fatal("fresh session inherited previous conversations")
	}
}

func TestWhisperSendsAndUnattributedAcknowledgements(t *testing.T) {
	netClient, server := testutil.NewConnection(t, 20080910)
	ctx := client.Context{Network: netClient, Input: input.NewState(), UIManager: gameui.NewManager()}
	mode := NewWorldMode()
	alice := mode.ui.social.WhisperWindows.Open(ctx, "Alice")
	bob := mode.ui.social.WhisperWindows.Open(ctx, "Bob")
	mode.social().SendWhisperWindowMessage(ctx, gameui.WhisperWindowAction{Target: "Alice", Message: "first"})
	if err := client.SendChat(ctx, "/w Carol console"); err != nil {
		t.Fatal(err)
	}
	mode.social().SendWhisperWindowMessage(ctx, gameui.WhisperWindowAction{Target: "Bob", Message: "second"})
	readBotTestPackets(t, server, network.BuildWhisperPacket("Alice", "first"))
	readBotTestPackets(t, server, network.BuildWhisperPacket("Carol", "console"))
	readBotTestPackets(t, server, network.BuildWhisperPacket("Bob", "second"))
	aliceBefore := testutil.DrawnWidgetText(alice.Widget())
	bobBefore := testutil.DrawnWidgetText(bob.Widget())
	// Replies carry no recipient and need not follow send order. In particular,
	// a reply to a console send must never be assigned to an open conversation.
	for _, result := range []byte{1, 0, 2} {
		mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCAckWhisper, Data: []byte{0x98, 0x00, result}}, time.Now())
	}
	if testutil.DrawnWidgetText(alice.Widget()) != aliceBefore || testutil.DrawnWidgetText(bob.Widget()) != bobBefore {
		t.Fatal("unnamed acknowledgement was assigned to a conversation")
	}
	netClient.Close()
	mode.social().SendWhisperWindowMessage(ctx, gameui.WhisperWindowAction{Target: "Alice", Message: "offline"})
	if !strings.Contains(testutil.DrawnWidgetText(alice.Widget()), "send failed") || testutil.DrawnWidgetText(bob.Widget()) != bobBefore {
		t.Fatal("local send failure did not stay with its originating conversation")
	}
}

func TestHandleChatRoomCreateAckKeepsSuccessOutOfRoomTranscript(t *testing.T) {
	mode := &WorldMode{
		socialSystem: social.System{PendingChatRoom: network.ChatRoomCreate{Title: "Room", Limit: 20, Public: true}},
	}
	ctx := client.Context{
		Session:   &session.Session{CharID: 1, Selected: session.Character{ID: 1, Name: "Kivutar"}},
		UIManager: &worldModeTestUIManager{},
	}

	mode.social().HandleChatRoomCreateAck(ctx, network.ChatRoomCreateAck{Result: 0})

	if !mode.ui.social.ChatRoom.IsOpen() {
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
	mode.ui.social.ChatRoom.Open(ctx, "Room", 20, true, []string{"Kivutar"})

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
	if len(mode.actorSystem.SpeechBubbles) != 0 {
		t.Fatalf("speech bubbles = %d, want 0", len(mode.actorSystem.SpeechBubbles))
	}
}

func TestHandleAnnouncementBypassesOpenChatRoom(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{UIManager: &worldModeTestUIManager{}}
	mode.ui.social.ChatRoom.Open(ctx, "Room", 20, true, []string{"Kivutar"})
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
	lines := reflect.ValueOf(&mode.ui.social.ChatRoom).Elem().FieldByName("lines")
	out := make([]string, 0, lines.Len())
	for i := 0; i < lines.Len(); i++ {
		out = append(out, lines.Index(i).FieldByName("text").String())
	}
	return out
}

func newWorldMailTest(t *testing.T) (*WorldMode, client.Context, net.Conn, time.Time) {
	t.Helper()
	netClient, server := testutil.NewConnection(t, 20080910)
	mode := NewWorldMode()
	ctx := client.Context{
		Network: netClient, Session: &session.Session{Inventory: session.Inventory{Zeny: 1000,
			Items: []session.InventoryItem{{Index: 7, ItemID: 501, Amount: 10, Identified: true}}}},
		ScreenW: 1024, ScreenH: 768,
	}
	mode.ui.social.MailWindow.Open(ctx, mode.social().MailError)
	return mode, ctx, server, time.Now()
}

func startWorldMailAttachment(t *testing.T, mode *WorldMode, ctx client.Context, server net.Conn, now time.Time) session.InventoryItem {
	t.Helper()
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionCompose, Recipient: "Erika", Title: "Hello"}, now)
	testutil.ReadPackets(t, server, network.BuildMailResetPacket(network.MailResetAll))
	item := ctx.Session.Inventory.Items[0]
	item.Amount = 3
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionAttach, Item: item}, now)
	testutil.ReadPackets(t, server, network.BuildMailAddAttachmentPacket(7, 3))
	return item
}

func TestMailAttachmentAcknowledgementAndReset(t *testing.T) {
	mode, ctx, server, now := newWorldMailTest(t)
	item := startWorldMailAttachment(t, mode, ctx, server, now)
	if got := ctx.Session.Inventory.Items[0].Amount; got != 10 {
		t.Fatalf("inventory changed before ACK: %d", got)
	}
	// A duplicate user action and an unrelated ACK cannot complete the request.
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionAttach, Item: item}, now)
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMailAddAttachment, Data: []byte{0x55, 2, 8, 0, 0}}, now)
	if mode.socialSystem.Mail.Pending.Kind != gameui.MailActionAttach {
		t.Fatal("unrelated ACK consumed pending attach")
	}
	assertNoBotTestPacket(t, server, func() error { return nil })
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMailAddAttachment, Data: []byte{0x55, 2, 7, 0, 0}}, now)
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMailAddAttachment, Data: []byte{0x55, 2, 7, 0, 0}}, now)
	if got := ctx.Session.Inventory.Items[0].Amount; got != 7 {
		t.Fatalf("inventory after ACK = %d, want 7", got)
	}
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionRemoveAttachment}, now)
	testutil.ReadPackets(t, server, network.BuildMailResetPacket(network.MailResetItem))
	if got := ctx.Session.Inventory.Items[0].Amount; got != 7 {
		t.Fatalf("reset fabricated returned items: %d", got)
	}
	// Legacy servers return reservations through the ordinary item-add path.
	addPickedSessionInventoryItem(ctx.Session, item)
	if got := ctx.Session.Inventory.Items[0].Amount; got != 10 {
		t.Fatalf("returned inventory = %d", got)
	}
}

func TestMailLateAttachmentAcknowledgementAfterClose(t *testing.T) {
	mode, ctx, server, now := newWorldMailTest(t)
	item := startWorldMailAttachment(t, mode, ctx, server, now)
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionClose}, now)
	testutil.ReadPackets(t, server, network.BuildMailResetPacket(network.MailResetAll))
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMailWindow, Data: []byte{0x60, 2, 0, 0, 0, 0}}, now)
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionCompose}, now)
	assertNoBotTestPacket(t, server, func() error { return nil })
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMailAddAttachment, Data: []byte{0x55, 2, 7, 0, 0}}, now)
	if mode.ui.social.MailWindow.IsComposing() {
		t.Fatal("old operation changed the new mailbox")
	}
	addPickedSessionInventoryItem(ctx.Session, item)
	if got := ctx.Session.Inventory.Items[0].Amount; got != 10 {
		t.Fatalf("late ACK/reset duplicated items: %d", got)
	}
	mode.social().UpdateMail(ctx, now)
	testutil.ReadPackets(t, server, append(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket()...))
}

func TestMailDisconnectClearsWindowsAndPendingRequests(t *testing.T) {
	mode, ctx, server, now := newWorldMailTest(t)
	startWorldMailAttachment(t, mode, ctx, server, now)
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketSCNotifyBan, Data: []byte{0x81, 0, 15}}, now)
	if !mode.ui.disconnectDialog.IsOpen() || mode.ui.social.MailWindow.IsOpen() || mode.socialSystem.Mail.Pending.Kind != gameui.MailActionNone {
		t.Fatal("disconnect left mail interaction alive")
	}
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMailAddAttachment, Data: []byte{0x55, 2, 7, 0, 0}}, now)
	if ctx.Session.Inventory.Items[0].Amount != 10 {
		t.Fatal("discarded connection changed inventory")
	}
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMailWindow, Data: []byte{0x60, 2, 0, 0, 0, 0}}, now)
	if mode.ui.social.MailWindow.IsOpen() {
		t.Fatal("old server packet reopened mail over the disconnect alert")
	}
	if next := NewWorldMode(); next.socialSystem.Mail.Pending.Kind != gameui.MailActionNone {
		t.Fatal("new login inherited pending mail")
	}
}

func TestMailComposerEnterDoesNotActivateConsole(t *testing.T) {
	mode, ctx, server, now := newWorldMailTest(t)
	ctx.Input = input.NewState()
	ctx.World = worldstate.New()
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionCompose, Recipient: "Erika", Title: "Hello"}, now)
	testutil.ReadPackets(t, server, network.BuildMailResetPacket(network.MailResetAll))
	ctx.Input.SetKey(input.KeyEnter, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.console.Active() {
		t.Fatal("Enter in the composer activated map chat")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKey(input.KeyEnter, false)
	ctx.Input.SetKey(input.KeyEscape, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.social.MailWindow.IsOpen() || mode.ui.escapeMenu.IsOpen() {
		t.Fatal("Escape should dismiss only the composer")
	}
	mode.social().UpdateMail(ctx, now)
	testutil.ReadPackets(t, server, network.BuildMailResetPacket(network.MailResetAll))
}

func TestMailAcknowledgementInMapChangeBatchIsNotDiscarded(t *testing.T) {
	mode, ctx, server, now := newWorldMailTest(t)
	startWorldMailAttachment(t, mode, ctx, server, now)
	change := network.Packet{ID: 0x0091, Data: make([]byte, 22)}
	binary.LittleEndian.PutUint16(change.Data, change.ID)
	copy(change.Data[2:18], "prontera.gat")
	mode.deferredPackets = []network.Packet{change, {ID: network.PacketZCMailAddAttachment, Data: []byte{0x55, 2, 7, 0, 0}}}
	if _, stop := mode.handleNetworkPackets(ctx, now); !stop {
		t.Fatal("map transition did not stop the frame")
	}
	if len(mode.deferredPackets) != 1 || ctx.Session.Inventory.Items[0].Amount != 10 {
		t.Fatal("transition discarded or prematurely processed its remaining packets")
	}
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionClose}, now)
	testutil.ReadPackets(t, server, network.BuildMailResetPacket(network.MailResetAll))
	mode = mode.nextWorldMode()
	if _, stop := mode.handleNetworkPackets(ctx, now); stop {
		t.Fatal("deferred mail ACK interrupted next map")
	}
	if len(mode.deferredPackets) != 0 || ctx.Session.Inventory.Items[0].Amount != 7 || mode.socialSystem.Mail.Pending.Kind != gameui.MailActionNone {
		t.Fatal("next map did not process the deferred reservation ACK")
	}
}

func TestMailMapTransitionPreservesPendingCorrelation(t *testing.T) {
	mode, ctx, server, now := newWorldMailTest(t)
	startWorldMailAttachment(t, mode, ctx, server, now)
	mode.social().HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionClose}, now)
	testutil.ReadPackets(t, server, network.BuildMailResetPacket(network.MailResetAll))
	mode = mode.nextWorldMode()
	mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMailAddAttachment, Data: []byte{0x55, 2, 7, 0, 0}}, now)
	if ctx.Session.Inventory.Items[0].Amount != 7 {
		t.Fatal("map transition lost inventory reservation ACK")
	}
	if mode.ui.social.MailWindow.IsOpen() {
		t.Fatal("late ACK reopened mail on a different map")
	}
}
