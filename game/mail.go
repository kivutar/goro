package game

import (
	"fmt"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

type mailState struct {
	pending          gameui.MailAction
	waitingZeny      bool
	readingAfterTake bool
	cancelled        bool
	inboxDirty       bool
	since            time.Time
	warned           bool
}

func (m *WorldMode) updateMail(ctx client.Context, now time.Time) {
	for {
		action := m.ui.mailWindow.PopAction()
		if action.Kind == gameui.MailActionNone {
			break
		}
		m.handleMailAction(ctx, action, now)
	}
	if m.mail.pending.Kind != gameui.MailActionNone {
		// Anonymous legacy ACKs cannot safely be forgotten on timeout: a late
		// reply could otherwise complete the next send/take operation instead.
		if !m.mail.warned && now.Sub(m.mail.since) > 10*time.Second {
			m.mail.warned = true
			m.mailError("No mail reply from the server. Reconnect before retrying.")
		}
		return
	}
	if m.mail.inboxDirty && m.ui.mailWindow.IsOpen() && !m.ui.mailWindow.IsComposing() {
		m.handleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionRefresh}, now)
	}
}

func (m *WorldMode) handleMailAction(ctx client.Context, action gameui.MailAction, now time.Time) {
	if action.Kind == gameui.MailActionClose {
		m.closeMail(ctx)
		return
	}
	if !m.ui.mailWindow.IsOpen() || m.mail.pending.Kind != gameui.MailActionNone {
		return
	}
	if ctx.Network == nil {
		m.ui.mailWindow.SetBusy(false)
		m.mailError("Unable to contact the mail server.")
		return
	}
	var err error
	switch action.Kind {
	case gameui.MailActionCompose:
		err = ctx.Network.SendMailReset(network.MailResetAll)
		if err == nil {
			m.ui.mailWindow.BeginCompose(action.Recipient, action.Title)
		}
	case gameui.MailActionRemoveAttachment:
		err = ctx.Network.SendMailReset(network.MailResetItem)
		if err == nil {
			m.ui.mailWindow.SetAttachment(session.InventoryItem{})
		}
	case gameui.MailActionRefresh:
		err = ctx.Network.SendMailReset(network.MailResetAll)
		if err == nil {
			err = ctx.Network.SendMailRefresh()
		}
		if err == nil {
			m.ui.mailWindow.ShowInbox()
		}
	case gameui.MailActionRead:
		if action.ID == 0 {
			m.ui.mailWindow.SetBusy(false)
			return
		}
		err = ctx.Network.SendMailRead(action.ID)
	case gameui.MailActionTake, gameui.MailActionDelete, gameui.MailActionReturn:
		message, ok := m.ui.mailWindow.ReadMessage()
		if !ok || message.ID != action.ID || action.ID == 0 {
			m.ui.mailWindow.SetBusy(false)
			return
		}
		switch action.Kind {
		case gameui.MailActionTake:
			if !message.HasAttachment() {
				m.ui.mailWindow.SetBusy(false)
				return
			}
			err = ctx.Network.SendMailGetAttachment(action.ID)
		case gameui.MailActionDelete:
			if message.HasAttachment() {
				m.ui.mailWindow.SetBusy(false)
				m.mailError("Take the attachments before deleting this mail.")
				return
			}
			err = ctx.Network.SendMailDelete(action.ID)
		case gameui.MailActionReturn:
			err = ctx.Network.SendMailReturn(action.ID, message.Sender)
		}
	case gameui.MailActionAttach:
		item, ok := findSessionInventoryItem(ctx.Session, action.Item.Index)
		if !ok || item.Index < 2 || item.ItemID != action.Item.ItemID || item.Equipped || action.Item.Amount <= 0 || action.Item.Amount > min(item.Amount, 32767) {
			m.ui.mailWindow.SetBusy(false)
			m.mailError("This item is no longer available to attach.")
			return
		}
		err = ctx.Network.SendMailAddAttachment(item.Index, uint32(action.Item.Amount))
	case gameui.MailActionSend:
		// Validate before reserving Zeny or transmitting any part of a send.
		_, err = network.BuildMailSendPacket(action.Recipient, action.Title, action.Body)
		if err == nil && (ctx.Session == nil || uint64(action.Zeny) > uint64(max(0, ctx.Session.Inventory.Zeny))) {
			err = fmt.Errorf("not enough Zeny")
		}
		if err == nil {
			err = ctx.Network.SendMailReset(network.MailResetZeny)
		}
		if err == nil {
			if action.Zeny > 0 {
				err = ctx.Network.SendMailAddAttachment(0, action.Zeny)
			} else {
				err = ctx.Network.SendMail(action.Recipient, action.Title, action.Body)
			}
		}
	default:
		return
	}
	if err != nil {
		m.ui.mailWindow.SetBusy(false)
		m.mailError("Mail request failed: " + err.Error())
		return
	}
	if action.Kind == gameui.MailActionCompose || action.Kind == gameui.MailActionRemoveAttachment {
		m.ui.mailWindow.SetBusy(false)
		m.ui.mailWindow.ShowStatus("")
		return
	}
	m.mail.pending = action
	m.mail.cancelled = false
	m.mail.waitingZeny = action.Kind == gameui.MailActionSend && action.Zeny > 0
	m.mail.since = now
	m.mail.warned = false
	m.ui.mailWindow.SetBusy(true)
	m.ui.mailWindow.ShowStatus("Waiting for the mail server...")
}

func (m *WorldMode) closeMail(ctx client.Context) {
	// A user close has already hidden the window; its reservation still needs
	// releasing even when there is no outstanding acknowledgement.
	if ctx.Network != nil {
		if err := ctx.Network.SendMailReset(network.MailResetAll); err != nil {
			glog.Warnf("mail reset on close failed: %v", err)
		}
	}
	m.ui.mailWindow.CloseFromServer(ctx)
	m.mail.cancelled = true
	m.mail.inboxDirty = false
	// Preserve any outstanding ACK until it arrives, even if another mailbox
	// is opened on this same connection. Character selection creates a new
	// WorldMode; map changes on the same connection carry this ledger forward.
}

func (m *WorldMode) finishMailOperation() bool {
	visible := !m.mail.cancelled && m.ui.mailWindow.IsOpen()
	m.mail.pending = gameui.MailAction{}
	m.mail.waitingZeny = false
	m.mail.readingAfterTake = false
	m.mail.cancelled = false
	m.mail.warned = false
	m.ui.mailWindow.SetBusy(false)
	if visible {
		m.ui.mailWindow.ShowStatus("")
	}
	return visible
}

func (m *WorldMode) mailError(message string) {
	m.ui.mailWindow.ShowStatus(message)
	m.ui.console.AddErrorMessage("%s", message)
}

func (m *WorldMode) handleMailPacket(ctx client.Context, pkt network.Packet, now time.Time) bool {
	var err error
	switch pkt.ID {
	case network.PacketZCMailWindow:
		if m.ui.disconnectDialog.IsOpen() {
			return true
		}
		var open bool
		open, _, err = network.ParseMailWindow(pkt)
		if err != nil {
			break
		}
		if !open {
			m.closeMail(ctx)
			return true
		}
		if m.ui.mailWindow.IsOpen() {
			return true
		}
		m.ui.mailWindow.Open(ctx)
		m.mail.inboxDirty = true
		m.ui.mailWindow.SetBusy(m.mail.pending.Kind != gameui.MailActionNone)
		m.updateMail(ctx, now)
	case network.PacketZCMailList:
		var entries []network.MailEntry
		entries, _, err = network.ParseMailList(pkt)
		if err != nil {
			break
		}
		if m.ui.mailWindow.IsOpen() && !m.mail.cancelled {
			m.ui.mailWindow.SetInbox(entries)
			m.mail.inboxDirty = false
		}
		if m.mail.pending.Kind == gameui.MailActionRefresh {
			m.finishMailOperation()
		}
	case network.PacketZCMailRead:
		var message network.MailMessage
		message, _, err = network.ParseMailRead(pkt)
		if err != nil {
			break
		}
		if (m.mail.pending.Kind == gameui.MailActionRead || m.mail.readingAfterTake) && message.ID == m.mail.pending.ID {
			refresh := m.mail.readingAfterTake
			if m.finishMailOperation() {
				if refresh {
					m.ui.mailWindow.UpdateReadMessage(message)
				} else {
					m.ui.mailWindow.ShowMail(message)
				}
			}
		}
	case network.PacketZCMailNew:
		var entry network.MailEntry
		entry, _, err = network.ParseMailNew(pkt)
		if err != nil {
			break
		}
		// ID zero is a legacy unread-status refresh, not a new message.
		if entry.ID != 0 {
			m.ui.console.AddBlueMessage("You have new mail.")
			m.mail.inboxDirty = true
		}
	case network.PacketZCMailAddAttachment, network.PacketZCMailGetAttachment, network.PacketZCMailSend, network.PacketZCMailDelete, network.PacketZCMailReturn:
		var result network.MailResult
		result, _, err = network.ParseMailResult(pkt)
		if err == nil {
			m.handleMailResult(ctx, pkt.ID, result, now)
		}
	default:
		return false
	}
	if err != nil {
		glog.Errorf("parse legacy mail 0x%04X: %v", pkt.ID, err)
	}
	return true
}

func (m *WorldMode) handleMailResult(ctx client.Context, opcode uint16, result network.MailResult, now time.Time) {
	pending := m.mail.pending
	switch opcode {
	case network.PacketZCMailAddAttachment:
		if pending.Kind == gameui.MailActionSend && m.mail.waitingZeny && result.Index == 0 {
			if result.Result != 0 {
				m.finishMailOperation()
				m.mailError("The Zeny could not be attached.")
				return
			}
			if m.mail.cancelled {
				m.finishMailOperation()
				return
			}
			m.mail.waitingZeny = false
			m.mail.since = now
			if err := ctx.Network.SendMail(pending.Recipient, pending.Title, pending.Body); err != nil {
				m.finishMailOperation()
				m.mailError("Sending mail failed: " + err.Error())
			}
			return
		}
		if pending.Kind != gameui.MailActionAttach || result.Index != pending.Item.Index {
			return
		}
		if result.Result == 0 {
			// Legacy ACK_MAIL_ADD_ITEM removes the reserved stack from the
			// client inventory. Reset/failure returns it via normal add-item
			// packets; successful delivery does not send a second deletion.
			removeSessionInventoryItem(ctx.Session, pending.Item.Index, pending.Item.Amount)
			if m.finishMailOperation() {
				m.ui.mailWindow.SetAttachment(pending.Item)
			}
		} else {
			m.finishMailOperation()
			m.mailError("This item cannot be attached.")
		}
	case network.PacketZCMailGetAttachment:
		if pending.Kind != gameui.MailActionTake || m.mail.readingAfterTake {
			return
		}
		if result.Result == 0 {
			// Some legacy servers send a success for the item and another for
			// Zeny. Re-read before accepting another operation: the ID-bearing
			// reply refreshes attachments and separates those anonymous ACKs
			// from the next retrieval, even if TCP splits them across frames.
			m.mail.readingAfterTake = true
			m.mail.since = now
			if err := ctx.Network.SendMailRead(pending.ID); err != nil {
				m.mailError("Unable to refresh the attachment: " + err.Error())
			}
			return
		}
		m.finishMailOperation()
		if result.Result == 1 {
			m.mailError("Cannot receive the attachment: the Zeny limit would be exceeded.")
		} else {
			m.mailError("Cannot receive the attachment: not enough inventory space or weight capacity.")
		}
	case network.PacketZCMailSend:
		if pending.Kind != gameui.MailActionSend || m.mail.waitingZeny {
			return
		}
		visible := m.finishMailOperation()
		if result.Result == 0 {
			m.ui.console.AddBlueMessage("Mail sent.")
			if visible {
				m.ui.mailWindow.ShowInbox()
				m.mail.inboxDirty = true
			}
		} else {
			// The server restores attachments using normal inventory/status
			// packets. Preserve the message text, but do not reuse reservations.
			if ctx.Network != nil {
				_ = ctx.Network.SendMailReset(network.MailResetAll)
			}
			m.ui.mailWindow.SetAttachment(session.InventoryItem{})
			m.mailError("Mail was not sent. Check the recipient and try again.")
		}
	case network.PacketZCMailDelete, network.PacketZCMailReturn:
		if (pending.Kind == gameui.MailActionRead || m.mail.readingAfterTake) && opcode == network.PacketZCMailReturn && result.ID == pending.ID && result.Result != 0 {
			// rAthena uses ACK_MAIL_RETURN to report a missing read target.
			m.finishMailOperation()
			m.mailError("This mail is no longer available.")
			m.mail.inboxDirty = true
			return
		}
		kind := gameui.MailActionDelete
		if opcode == network.PacketZCMailReturn {
			kind = gameui.MailActionReturn
		}
		if pending.Kind != kind || result.ID != pending.ID {
			return
		}
		m.finishMailOperation()
		if result.Result == 0 {
			m.ui.mailWindow.RemoveMail(result.ID)
			m.mail.inboxDirty = true
		} else if kind == gameui.MailActionDelete {
			m.mailError("Mail could not be deleted. Take its attachments first.")
		} else {
			m.mailError("This mail could not be returned to its sender.")
		}
	}
}
