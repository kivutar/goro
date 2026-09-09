package game

import (
	"encoding/binary"
	"fmt"
	"net"
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
	"github.com/kivutar/goro/ui/rotheme"
	worldstate "github.com/kivutar/goro/world"
)

type mailTest struct {
	t      *testing.T
	m      *WorldMode
	ctx    client.Context
	server net.Conn
	now    time.Time
}

func newMailTest(t *testing.T) *mailTest {
	t.Helper()
	netClient, server := newBotTestConnection(t, 20080910)
	h := &mailTest{t: t, m: NewWorldMode(), server: server, now: time.Now(), ctx: client.Context{
		Network: netClient, Session: &session.Session{Inventory: session.Inventory{Zeny: 1000,
			Items: []session.InventoryItem{{Index: 7, ItemID: 501, Amount: 10, Identified: true}}}},
		ScreenW: 1024, ScreenH: 768,
	}}
	h.m.ui.mailWindow.Open(h.ctx, h.m.mailError)
	return h
}

func (h *mailTest) action(action gameui.MailAction) { h.m.handleMailAction(h.ctx, action, h.now) }
func (h *mailTest) expect(packets ...[]byte) {
	h.t.Helper()
	var want []byte
	for _, p := range packets {
		want = append(want, p...)
	}
	readBotTestPackets(h.t, h.server, want)
}
func (h *mailTest) quiet() {
	h.t.Helper()
	assertNoBotTestPacket(h.t, h.server, func() error { return nil })
}
func (h *mailTest) result(opcode uint16, result network.MailResult) {
	h.t.Helper()
	n := 3
	if opcode == network.PacketZCMailAddAttachment {
		n = 5
	}
	if opcode == network.PacketZCMailDelete || opcode == network.PacketZCMailReturn {
		n = 8
	}
	p := make([]byte, n)
	binary.LittleEndian.PutUint16(p, opcode)
	switch n {
	case 3:
		p[2] = byte(result.Result)
	case 5:
		binary.LittleEndian.PutUint16(p[2:], result.Index)
		p[4] = byte(result.Result)
	case 8:
		binary.LittleEndian.PutUint32(p[2:], result.ID)
		binary.LittleEndian.PutUint16(p[6:], result.Result)
	}
	if !h.m.handleMailPacket(h.ctx, network.Packet{ID: opcode, Data: p}, h.now) {
		h.t.Fatal("mail ACK was not handled")
	}
}
func (h *mailTest) compose() {
	h.action(gameui.MailAction{Kind: gameui.MailActionCompose, Recipient: "Erika", Title: "Hello"})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
}

func (h *mailTest) readReply(id, zeny uint32) {
	h.t.Helper()
	p := make([]byte, 101)
	binary.LittleEndian.PutUint16(p, network.PacketZCMailRead)
	binary.LittleEndian.PutUint16(p[2:], uint16(len(p)))
	binary.LittleEndian.PutUint32(p[4:], id)
	copy(p[8:48], "Gift")
	copy(p[48:72], "Erika")
	binary.LittleEndian.PutUint32(p[76:], zeny)
	h.m.handleMailPacket(h.ctx, network.Packet{ID: network.PacketZCMailRead, Data: p}, h.now)
}

func (h *mailTest) listReply(ids ...uint32) {
	p := make([]byte, 8+len(ids)*73)
	binary.LittleEndian.PutUint16(p, network.PacketZCMailList)
	binary.LittleEndian.PutUint16(p[2:], uint16(len(p)))
	binary.LittleEndian.PutUint32(p[4:], uint32(len(ids)))
	for i, id := range ids {
		o := 8 + i*73
		binary.LittleEndian.PutUint32(p[o:], id)
		copy(p[o+4:o+44], "Gift")
		copy(p[o+45:o+69], "Erika")
	}
	h.m.handleMailPacket(h.ctx, network.Packet{ID: network.PacketZCMailList, Data: p}, h.now)
}

func TestMailServerOpenListReadAndNewMail(t *testing.T) {
	h := newMailTest(t)
	h.m.ui.mailWindow.CloseFromServer(h.ctx)
	open := network.Packet{ID: network.PacketZCMailWindow, Data: []byte{0x60, 2, 0, 0, 0, 0}}
	h.m.handleNetworkPacket(h.ctx, open, h.now)
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
	h.m.handleNetworkPacket(h.ctx, open, h.now)
	h.quiet()
	h.listReply(42, 43)
	if h.m.mail.pending.Kind != gameui.MailActionNone || h.m.mail.inboxDirty || !h.m.ui.mailWindow.IsOpen() {
		t.Fatal("opening/listing the inbox did not finish")
	}
	h.action(gameui.MailAction{Kind: gameui.MailActionRead, ID: 42})
	h.expect(network.BuildMailReadPacket(42))
	h.readReply(43, 0)
	if h.m.mail.pending.ID != 42 {
		t.Fatal("read reply for another mail consumed request")
	}
	h.readReply(42, 0)
	if msg, ok := h.m.ui.mailWindow.ReadMessage(); !ok || msg.ID != 42 {
		t.Fatal("read reply did not open requested mail")
	}
	h.compose()
	newMail := network.Packet{ID: network.PacketZCMailNew, Data: make([]byte, 70)}
	binary.LittleEndian.PutUint16(newMail.Data, newMail.ID)
	h.m.handleMailPacket(h.ctx, newMail, h.now)
	if h.m.mail.inboxDirty {
		t.Fatal("unread-count notification treated as new mail")
	}
	binary.LittleEndian.PutUint32(newMail.Data[2:], 44)
	h.m.handleMailPacket(h.ctx, newMail, h.now)
	h.m.updateMail(h.ctx, h.now)
	h.quiet()
	if !h.m.mail.inboxDirty || !h.m.ui.mailWindow.IsComposing() {
		t.Fatal("incoming mail changed the draft")
	}
}

func TestMailRefreshReplyAfterReopenDoesNotReplaceNewInbox(t *testing.T) {
	h := newMailTest(t)
	h.action(gameui.MailAction{Kind: gameui.MailActionRefresh})
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
	h.action(gameui.MailAction{Kind: gameui.MailActionClose})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	h.m.handleMailPacket(h.ctx, network.Packet{ID: network.PacketZCMailWindow, Data: []byte{0x60, 2, 0, 0, 0, 0}}, h.now)
	h.listReply(42)
	if !h.m.mail.inboxDirty {
		t.Fatal("cancelled refresh consumed the new inbox request")
	}
	h.m.updateMail(h.ctx, h.now)
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
	h.listReply(43)
	if h.m.mail.inboxDirty || h.m.mail.pending.Kind != gameui.MailActionNone {
		t.Fatal("new refresh did not finish")
	}
}

func TestMailMissingReadTargetRefreshesInbox(t *testing.T) {
	h := newMailTest(t)
	h.action(gameui.MailAction{Kind: gameui.MailActionRead, ID: 42})
	h.expect(network.BuildMailReadPacket(42))
	h.result(network.PacketZCMailReturn, network.MailResult{ID: 42, Result: 1})
	if h.m.mail.pending.Kind != gameui.MailActionNone || !h.m.mail.inboxDirty {
		t.Fatal("missing read target left mailbox stuck")
	}
	h.m.updateMail(h.ctx, h.now)
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
}
func (h *mailTest) attach() session.InventoryItem {
	h.t.Helper()
	item := h.ctx.Session.Inventory.Items[0]
	item.Amount = 3
	h.action(gameui.MailAction{Kind: gameui.MailActionAttach, Item: item})
	h.expect(network.BuildMailAddAttachmentPacket(7, 3))
	return item
}

func TestMailAttachmentAcknowledgementAndReset(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	item := h.attach()
	if got := h.ctx.Session.Inventory.Items[0].Amount; got != 10 {
		t.Fatalf("inventory changed before ACK: %d", got)
	}
	// A duplicate user action and an unrelated ACK cannot complete the request.
	h.action(gameui.MailAction{Kind: gameui.MailActionAttach, Item: item})
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 8})
	if h.m.mail.pending.Kind != gameui.MailActionAttach {
		t.Fatal("unrelated ACK consumed pending attach")
	}
	h.quiet()
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	if got := h.ctx.Session.Inventory.Items[0].Amount; got != 7 {
		t.Fatalf("inventory after ACK = %d, want 7", got)
	}
	h.action(gameui.MailAction{Kind: gameui.MailActionRemoveAttachment})
	h.expect(network.BuildMailResetPacket(network.MailResetItem))
	if got := h.ctx.Session.Inventory.Items[0].Amount; got != 7 {
		t.Fatalf("reset fabricated returned items: %d", got)
	}
	// Legacy servers return reservations through the ordinary item-add path.
	addPickedSessionInventoryItem(h.ctx.Session, item)
	if got := h.ctx.Session.Inventory.Items[0].Amount; got != 10 {
		t.Fatalf("returned inventory = %d", got)
	}
}

func TestMailComposerLayoutIsStableDuringRequestsAndErrors(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	var editor *rotheme.TextAreaWidget
	layout := func() geometry.Rect {
		t.Helper()
		root := h.m.ui.mailWindow.Widget()
		ctx := widget.NewContext()
		root.Layout(ctx, geometry.Tight(geometry.Sz(1024, 768)))
		root.Draw(ctx, &uitest.MockCanvas{})
		editor = nil
		var findEditor func(widget.Widget)
		findEditor = func(w widget.Widget) {
			if field, ok := w.(*rotheme.TextAreaWidget); ok {
				editor = field
			}
			for _, child := range w.Children() {
				findEditor(child)
			}
		}
		findEditor(root)
		if editor == nil {
			t.Fatal("composer has no body editor")
		}
		return editor.ScreenBounds()
	}
	want := layout()
	original := editor
	h.attach()
	if got := layout(); got != want || editor != original || editor.IsEnabled() {
		t.Fatalf("pending attachment changed editor layout: got %v, want %v (enabled=%t)", got, want, editor.IsEnabled())
	}
	h.m.updateMail(h.ctx, h.now.Add(time.Second))
	if got := layout(); got != want {
		t.Fatalf("waiting for attachment changed editor layout: got %v, want %v", got, want)
	}
	h.m.updateMail(h.ctx, h.now.Add(time.Minute))
	if got := layout(); got != want || editor != original || editor.IsEnabled() {
		t.Fatalf("timeout changed editor layout: got %v, want %v (enabled=%t)", got, want, editor.IsEnabled())
	}
	if messages := h.m.ui.console.Messages(); len(messages) != 1 || messages[0].Text != "No mail reply from the server. Reconnect before retrying." {
		t.Fatalf("timeout console messages = %+v", messages)
	}
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	if got := layout(); got != want || editor != original || !editor.IsEnabled() {
		t.Fatalf("attachment reply changed editor layout: got %v, want %v (enabled=%t)", got, want, editor.IsEnabled())
	}
	action := gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "Missing", Title: "Hello"}
	h.action(action)
	packet, _ := network.BuildMailSendPacket(action.Recipient, action.Title, action.Body)
	h.expect(network.BuildMailResetPacket(network.MailResetZeny), packet)
	h.result(network.PacketZCMailSend, network.MailResult{Result: 1})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	if got := layout(); got != want || editor != original || !editor.IsEnabled() {
		t.Fatalf("send failure changed editor layout: got %v, want %v (enabled=%t)", got, want, editor.IsEnabled())
	}
	if messages := h.m.ui.console.Messages(); len(messages) != 2 || messages[1].Text != "Mail was not sent. Check the recipient and try again." {
		t.Fatalf("send failure console messages = %+v", messages)
	}
}

func TestMailCloseReleasesAcknowledgedAttachment(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	h.attach()
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	// The UI hides first, then hands the close intent to the game next frame.
	h.m.ui.mailWindow.CloseFromServer(h.ctx)
	h.action(gameui.MailAction{Kind: gameui.MailActionClose})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	if h.m.ui.mailWindow.IsOpen() {
		t.Fatal("closed mailbox still open")
	}
}

func TestMailLateAttachmentAcknowledgementAfterClose(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	item := h.attach()
	h.action(gameui.MailAction{Kind: gameui.MailActionClose})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	h.m.ui.mailWindow.Open(h.ctx, h.m.mailError)
	h.m.mail.inboxDirty = true
	h.action(gameui.MailAction{Kind: gameui.MailActionCompose})
	h.quiet()
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	if h.m.ui.mailWindow.IsComposing() {
		t.Fatal("old operation changed the new mailbox")
	}
	addPickedSessionInventoryItem(h.ctx.Session, item)
	if got := h.ctx.Session.Inventory.Items[0].Amount; got != 10 {
		t.Fatalf("late ACK/reset duplicated items: %d", got)
	}
	h.m.updateMail(h.ctx, h.now)
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
}

func TestMailSendWaitsForZenyAcknowledgement(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "send", true: "cancel"}[cancel], func(t *testing.T) {
			h := newMailTest(t)
			h.compose()
			action := gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "Erika", Title: "Hello", Body: "First line\nSecond line", Zeny: 500}
			h.action(action)
			h.expect(network.BuildMailResetPacket(network.MailResetZeny), network.BuildMailAddAttachmentPacket(0, 500))
			h.action(action)
			h.quiet()
			if cancel {
				h.action(gameui.MailAction{Kind: gameui.MailActionClose})
				h.expect(network.BuildMailResetPacket(network.MailResetAll))
			}
			h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 0})
			if cancel {
				h.quiet()
				return
			}
			want, _ := network.BuildMailSendPacket(action.Recipient, action.Title, action.Body)
			h.expect(want)
			h.result(network.PacketZCMailSend, network.MailResult{})
			if h.m.ui.mailWindow.IsComposing() || h.m.mail.pending.Kind != gameui.MailActionNone {
				t.Fatal("successful send did not finish")
			}
			if h.ctx.Session.Inventory.Zeny != 1000 {
				t.Fatal("send fabricated a Zeny status update")
			}
		})
	}
}

func TestMailSendWithoutZenyAndFailure(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	h.attach()
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	action := gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "Missing", Title: "Hello", Body: "Keep this draft"}
	h.action(action)
	want, _ := network.BuildMailSendPacket(action.Recipient, action.Title, action.Body)
	h.expect(network.BuildMailResetPacket(network.MailResetZeny), want)
	h.result(network.PacketZCMailSend, network.MailResult{Result: 1})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	if !h.m.ui.mailWindow.IsComposing() || h.m.mail.pending.Kind != gameui.MailActionNone {
		t.Fatal("failed send discarded composer or stayed pending")
	}
	if h.ctx.Session.Inventory.Items[0].Amount != 7 {
		t.Fatal("send result applied a second inventory change")
	}
}

func TestMailGetDeleteReturnAndErrors(t *testing.T) {
	for _, kind := range []gameui.MailActionKind{gameui.MailActionTake, gameui.MailActionDelete, gameui.MailActionReturn} {
		for _, code := range []uint16{0, 1, 2} {
			t.Run(fmt.Sprintf("%d/result%d", kind, code), func(t *testing.T) {
				h := newMailTest(t)
				msg := network.MailMessage{ID: 42, Sender: "Erika", Title: "Gift"}
				if kind != gameui.MailActionDelete {
					msg.Zeny = 100
				}
				h.m.ui.mailWindow.ShowMail(msg)
				h.action(gameui.MailAction{Kind: kind, ID: 42})
				opcode := network.PacketZCMailGetAttachment
				switch kind {
				case gameui.MailActionTake:
					h.expect(network.BuildMailGetAttachmentPacket(42))
				case gameui.MailActionDelete:
					opcode = network.PacketZCMailDelete
					h.expect(network.BuildMailDeletePacket(42))
				case gameui.MailActionReturn:
					opcode = network.PacketZCMailReturn
					h.expect(network.BuildMailReturnPacket(42, "Erika"))
				}
				h.result(opcode, network.MailResult{ID: 42, Result: code})
				if kind == gameui.MailActionTake && code == 0 {
					h.expect(network.BuildMailReadPacket(42))
					h.readReply(42, 0)
				}
				got, exists := h.m.ui.mailWindow.ReadMessage()
				if code != 0 && (!exists || got != msg) {
					t.Fatalf("error modified message: %+v", got)
				}
				if code == 0 && kind == gameui.MailActionTake && got.Zeny != 0 {
					t.Fatal("attachment not cleared")
				}
				if code == 0 && kind != gameui.MailActionTake && exists {
					t.Fatal("removed mail still readable")
				}
			})
		}
	}
}

func TestMailDoesNotDeleteAttachmentsOrSendInvalidDraft(t *testing.T) {
	h := newMailTest(t)
	h.m.ui.mailWindow.ShowMail(network.MailMessage{ID: 42, Zeny: 100})
	h.action(gameui.MailAction{Kind: gameui.MailActionDelete, ID: 42})
	h.action(gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "Erika", Title: "Hi", Zeny: 1001})
	h.action(gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "", Title: "Hi"})
	h.quiet()
	if h.m.mail.pending.Kind != gameui.MailActionNone {
		t.Fatal("invalid request became pending")
	}
}

func TestMailTimeoutKeepsAnonymousAcknowledgementLedger(t *testing.T) {
	h := newMailTest(t)
	h.m.ui.mailWindow.ShowMail(network.MailMessage{ID: 42, Zeny: 100})
	h.action(gameui.MailAction{Kind: gameui.MailActionTake, ID: 42})
	h.expect(network.BuildMailGetAttachmentPacket(42))
	h.m.updateMail(h.ctx, h.now.Add(time.Minute))
	if !h.m.mail.warned || h.m.mail.pending.ID != 42 {
		t.Fatal("timeout forgot anonymous ACK correlation")
	}
	h.action(gameui.MailAction{Kind: gameui.MailActionTake, ID: 42})
	h.quiet()
	h.result(network.PacketZCMailGetAttachment, network.MailResult{})
	h.expect(network.BuildMailReadPacket(42))
	h.readReply(42, 0)
	if h.m.mail.pending.Kind != gameui.MailActionNone {
		t.Fatal("late result did not release pending request")
	}
}

func TestMailGetItemAndZenyRepliesCannotCompleteNextRetrieval(t *testing.T) {
	h := newMailTest(t)
	h.m.ui.mailWindow.ShowMail(network.MailMessage{ID: 42, Zeny: 100, Attachment: network.MailAttachment{ItemID: 501, Amount: 3}})
	h.action(gameui.MailAction{Kind: gameui.MailActionTake, ID: 42})
	h.expect(network.BuildMailGetAttachmentPacket(42))
	h.result(network.PacketZCMailGetAttachment, network.MailResult{})
	h.expect(network.BuildMailReadPacket(42))
	h.action(gameui.MailAction{Kind: gameui.MailActionRead, ID: 43})
	h.result(network.PacketZCMailGetAttachment, network.MailResult{})
	if !h.m.mail.readingAfterTake || h.m.mail.pending.ID != 42 {
		t.Fatal("duplicate retrieval ACK completed the read barrier")
	}
	h.quiet()
	h.readReply(42, 0)
	msg, _ := h.m.ui.mailWindow.ReadMessage()
	if msg.Zeny != 0 || msg.Attachment.ItemID != 0 || h.m.mail.pending.Kind != gameui.MailActionNone {
		t.Fatal("read did not finish retrieval")
	}
}

func TestMailDisconnectClearsWindowsAndPendingRequests(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	h.attach()
	h.m.handleNetworkPacket(h.ctx, network.Packet{ID: network.PacketSCNotifyBan, Data: []byte{0x81, 0, 15}}, h.now)
	if !h.m.ui.disconnectDialog.IsOpen() || h.m.ui.mailWindow.IsOpen() || h.m.mail.pending.Kind != gameui.MailActionNone {
		t.Fatal("disconnect left mail interaction alive")
	}
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	if h.ctx.Session.Inventory.Items[0].Amount != 10 {
		t.Fatal("discarded connection changed inventory")
	}
	h.m.handleMailPacket(h.ctx, network.Packet{ID: network.PacketZCMailWindow, Data: []byte{0x60, 2, 0, 0, 0, 0}}, h.now)
	if h.m.ui.mailWindow.IsOpen() {
		t.Fatal("old server packet reopened mail over the disconnect alert")
	}
	if next := NewWorldMode(); next.mail.pending.Kind != gameui.MailActionNone {
		t.Fatal("new login inherited pending mail")
	}
}

func TestMailComposerEnterDoesNotActivateConsole(t *testing.T) {
	h := newMailTest(t)
	h.ctx.Input = input.NewState()
	h.ctx.World = worldstate.New()
	h.compose()
	h.ctx.Input.SetKey(input.KeyEnter, true)
	if _, err := h.m.Update(h.ctx); err != nil {
		t.Fatal(err)
	}
	if h.m.ui.console.Active() {
		t.Fatal("Enter in the composer activated map chat")
	}
	h.ctx.Input.EndFrame()
	h.ctx.Input.SetKey(input.KeyEnter, false)
	h.ctx.Input.SetKey(input.KeyEscape, true)
	if _, err := h.m.Update(h.ctx); err != nil {
		t.Fatal(err)
	}
	if h.m.ui.mailWindow.IsOpen() || h.m.ui.escapeMenu.IsOpen() {
		t.Fatal("Escape should dismiss only the composer")
	}
	h.m.updateMail(h.ctx, h.now)
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
}

func TestMailAcknowledgementInMapChangeBatchIsNotDiscarded(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	h.attach()
	change := network.Packet{ID: 0x0091, Data: make([]byte, 22)}
	binary.LittleEndian.PutUint16(change.Data, change.ID)
	copy(change.Data[2:18], "prontera.gat")
	h.m.deferredPackets = []network.Packet{change, {ID: network.PacketZCMailAddAttachment, Data: []byte{0x55, 2, 7, 0, 0}}}
	if _, stop := h.m.handleNetworkPackets(h.ctx, h.now); !stop {
		t.Fatal("map transition did not stop the frame")
	}
	if len(h.m.deferredPackets) != 1 || h.ctx.Session.Inventory.Items[0].Amount != 10 {
		t.Fatal("transition discarded or prematurely processed its remaining packets")
	}
	h.action(gameui.MailAction{Kind: gameui.MailActionClose})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	h.m = h.m.nextWorldMode()
	if _, stop := h.m.handleNetworkPackets(h.ctx, h.now); stop {
		t.Fatal("deferred mail ACK interrupted next map")
	}
	if len(h.m.deferredPackets) != 0 || h.ctx.Session.Inventory.Items[0].Amount != 7 || h.m.mail.pending.Kind != gameui.MailActionNone {
		t.Fatal("next map did not process the deferred reservation ACK")
	}
}

func TestMailMapTransitionPreservesPendingCorrelation(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	h.attach()
	h.action(gameui.MailAction{Kind: gameui.MailActionClose})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	h.m = h.m.nextWorldMode()
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	if h.ctx.Session.Inventory.Items[0].Amount != 7 {
		t.Fatal("map transition lost inventory reservation ACK")
	}
	if h.m.ui.mailWindow.IsOpen() {
		t.Fatal("late ACK reopened mail on a different map")
	}
}
