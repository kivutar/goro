package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	mailWindowW     = 450
	mailWindowH     = 366
	mailRowsPerPage = 7
	mailReadW       = 410
	mailReadH       = 330
)

type MailActionKind uint8

const (
	MailActionNone MailActionKind = iota
	MailActionRefresh
	MailActionCompose
	MailActionRead
	MailActionTake
	MailActionDelete
	MailActionReturn
	MailActionAttach
	MailActionRemoveAttachment
	MailActionSend
	MailActionClose
)

// MailAction describes user intent only. The game owns packet sending and
// acknowledgement correlation; the window never changes inventory balances.
type MailAction struct {
	Kind                   MailActionKind
	ID                     uint32
	Item                   session.InventoryItem
	Recipient, Title, Body string
	Zeny                   uint32
}

type MailWindow struct {
	Window
	readBodyField                         *rotheme.TextAreaWidget
	readWindow                            Window
	ctx                                   Context
	itemInfo                              *ItemInfoWindow
	inbox                                 []network.MailEntry
	message                               *network.MailMessage
	page                                  int
	selected                              uint32
	compose                               bool
	busy                                  bool
	status                                string
	actions                               []MailAction
	attachment                            session.InventoryItem
	recipient, title, body, zeny          string
	recipientField, titleField, zenyField *textfield.Widget
	bodyField                             *rotheme.TextAreaWidget
	composeSlot                           *mailAttachmentSlot
	amount                                amountPrompt
	confirm                               ConfirmModal
}

func (w *MailWindow) Open(ctx Context) {
	w.EnsureWindow(mailWindowW, mailWindowH)
	w.ctx = ctx
	w.clearDraft()
	w.inbox = nil
	w.page = 0
	w.selected = 0
	w.message = nil
	w.closeRead()
	w.status = "Loading mail..."
	w.Window.Open(ctx, w.widgetTree())
	w.Publish(ctx)
}

func (w *MailWindow) CloseFromServer(ctx Context) {
	w.ctx = ctx
	w.Window.Close()
	w.closeRead()
	w.amount.Close(ctx)
	w.confirm.Close(ctx)
	w.clearDraft()
	w.inbox = nil
	w.message = nil
	w.actions = nil
}

func (w *MailWindow) requestClose() {
	ctx := w.ctx
	w.CloseFromServer(ctx)
	w.actions = append(w.actions, MailAction{Kind: MailActionClose})
}

func (w *MailWindow) Update(ctx Context, itemInfo *ItemInfoWindow) bool {
	w.ctx, w.itemInfo = ctx, itemInfo
	if w.UpdateModal(ctx) {
		return true
	}
	if w.readWindow.Update(ctx) {
		if !w.readWindow.IsOpen() {
			w.closeRead()
		}
		w.readWindow.Publish(ctx)
		return true
	}
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	if !w.IsOpen() {
		w.requestClose()
		return true
	}
	w.Publish(ctx)
	return consumed
}

func (w *MailWindow) UpdateModal(ctx Context) bool {
	return w.confirm.Update(ctx) || w.amount.Update(ctx)
}

// UI text events are dispatched before world input. Keep their Enter/arrows
// from also activating the console or navigating its history this frame.
func (w *MailWindow) UpdateKeyboardInput(ctx Context) bool {
	if !w.IsOpen() || !w.compose || ctx.Input == nil {
		return false
	}
	focused := w.recipientField != nil && w.recipientField.IsFocused() ||
		w.titleField != nil && w.titleField.IsFocused() ||
		w.zenyField != nil && w.zenyField.IsFocused() ||
		w.bodyField != nil && w.bodyField.IsFocused()
	if !focused {
		return false
	}
	if ctx.Input.JustPressed(input.KeyEscape) {
		w.requestClose()
		return true
	}
	return ctx.Input.JustPressed(input.KeyEnter) || ctx.Input.JustPressed(input.KeyArrowUp) || ctx.Input.JustPressed(input.KeyArrowDown)
}

func (w *MailWindow) ModalOpen() bool { return w.confirm.IsOpen() || w.amount.IsOpen() }

func (w *MailWindow) IsComposing() bool { return w.compose }

func (w *MailWindow) PopAction() MailAction {
	if len(w.actions) == 0 {
		return MailAction{}
	}
	action := w.actions[0]
	w.actions = w.actions[1:]
	return action
}

func (w *MailWindow) request(action MailAction) {
	if w.busy || !w.IsOpen() {
		return
	}
	w.actions = append(w.actions, action)
	w.SetBusy(true)
}

func (w *MailWindow) SetBusy(busy bool) {
	w.busy = busy
	if w.bodyField != nil {
		w.bodyField.SetEnabled(!busy)
	}
	w.refresh()
}

func (w *MailWindow) ShowStatus(message string) { w.status = message; w.refresh() }

func (w *MailWindow) SetInbox(entries []network.MailEntry) {
	w.inbox = append(w.inbox[:0], entries...)
	w.page = min(w.page, w.pageCount()-1)
	selectedExists := false
	for _, entry := range w.inbox {
		selectedExists = selectedExists || entry.ID == w.selected
	}
	if !selectedExists {
		w.selected = 0
	}
	if !w.compose {
		w.status = ""
	}
	w.refresh()
}

func (w *MailWindow) ShowMail(message network.MailMessage) {
	w.setReadMessage(message)
	for i := range w.inbox {
		if w.inbox[i].ID == message.ID {
			w.inbox[i].Read = true
		}
	}
	w.readWindow.EnsureWindow(mailReadW, mailReadH)
	w.readWindow.Open(w.ctx, w.readTree())
	w.readWindow.Publish(w.ctx)
	w.refresh()
}

func (w *MailWindow) ReadMessage() (network.MailMessage, bool) {
	if w.message == nil {
		return network.MailMessage{}, false
	}
	return *w.message, true
}

// UpdateReadMessage refreshes attachments without reopening a dismissed reader.
func (w *MailWindow) UpdateReadMessage(message network.MailMessage) {
	if w.message != nil && w.message.ID == message.ID {
		w.setReadMessage(message)
		w.refresh()
	}
}

func (w *MailWindow) setReadMessage(message network.MailMessage) {
	if w.readBodyField == nil || w.message == nil || w.message.ID != message.ID || w.message.Body != message.Body {
		if w.readBodyField != nil {
			w.releaseFocus(w.readBodyField)
		}
		w.readBodyField = rotheme.TextArea(message.Body, 0, nil)
		w.readBodyField.SetReadOnly(true)
	}
	w.message = &message
}

func (w *MailWindow) closeRead() {
	if w.readBodyField != nil {
		w.releaseFocus(w.readBodyField)
		w.readBodyField = nil
	}
	w.readWindow.Close()
}

func (w *MailWindow) releaseFocus(field widget.Widget) {
	if ctx := windowWidgetContext(w.ctx); ctx != nil {
		ctx.ReleaseFocus(field)
	}
}

func (w *MailWindow) RemoveMail(id uint32) {
	for i, e := range w.inbox {
		if e.ID == id {
			w.inbox = append(w.inbox[:i], w.inbox[i+1:]...)
			break
		}
	}
	w.page = min(w.page, w.pageCount()-1)
	if w.message != nil && w.message.ID == id {
		w.message = nil
		w.closeRead()
	}
	if w.selected == id {
		w.selected = 0
	}
	w.refresh()
}

func (w *MailWindow) SetAttachment(item session.InventoryItem) {
	w.attachment = item
	w.refresh()
}

func (w *MailWindow) BeginCompose(recipient, title string) {
	w.clearDraft()
	w.compose = true
	w.recipient, w.title = recipient, mailTruncateText(title, network.MailTitleMax)
	w.closeRead()
	w.status = ""
	w.refresh()
	if w.recipientField != nil {
		if ctx := windowWidgetContext(w.ctx); ctx != nil {
			ctx.RequestFocus(w.recipientField)
		} else {
			w.recipientField.SetFocused(true)
		}
	}
}

func (w *MailWindow) ShowInbox() {
	w.clearDraft()
	w.closeRead()
	w.refresh()
}

func (w *MailWindow) clearDraft() {
	for _, field := range []*textfield.Widget{w.recipientField, w.titleField, w.zenyField} {
		if field != nil {
			w.releaseFocus(field)
		}
	}
	if w.bodyField != nil {
		w.releaseFocus(w.bodyField)
	}
	w.compose = false
	w.attachment = session.InventoryItem{}
	w.recipient, w.title, w.body, w.zeny = "", "", "", ""
	w.recipientField, w.titleField, w.zenyField = nil, nil, nil
	w.bodyField = nil
	w.composeSlot = nil
}

func (w *MailWindow) refresh() {
	if w.IsOpen() {
		w.SetContent(w.widgetTree())
		w.Publish(w.ctx)
	}
	if w.readWindow.IsOpen() && w.message != nil {
		w.readWindow.SetContent(w.readTree())
		w.readWindow.Publish(w.ctx)
	}
}

func (w *MailWindow) pageCount() int { return max(1, (len(w.inbox)+mailRowsPerPage-1)/mailRowsPerPage) }

func (w *MailWindow) widgetTree() widget.Widget {
	title := "Mail"
	if w.compose {
		title = "Write Mail"
	}
	inboxTab := newTabWidget(tabWidgetConfig{label: "Inbox", width: 72, height: 24, active: !w.compose, disabled: w.busy, onClick: func() {
		if w.compose {
			w.request(MailAction{Kind: MailActionRefresh})
		}
	}})
	writeTab := newTabWidget(tabWidgetConfig{label: "Write", width: 72, height: 24, active: w.compose, disabled: w.busy, onClick: func() {
		if !w.compose {
			w.request(MailAction{Kind: MailActionCompose})
		}
	}})
	var content widget.Widget
	var footer []widget.Widget
	if w.compose {
		content = w.composeTree()
		footer = []widget.Widget{primitives.Expanded(primitives.Box()), rotheme.ButtonDisabled("Send", w.busy, w.send), rotheme.ButtonDisabled("Cancel", w.busy, func() { w.request(MailAction{Kind: MailActionRefresh}) })}
	} else {
		content = w.inboxTree()
		footer = []widget.Widget{rotheme.ButtonDisabled("Refresh", w.busy, func() { w.request(MailAction{Kind: MailActionRefresh}) }), primitives.Expanded(primitives.Box()), rotheme.ButtonDisabled("Read", w.busy || w.selected == 0, func() { w.request(MailAction{Kind: MailActionRead, ID: w.selected}) }), rotheme.Button("Close", w.requestClose)}
	}
	return Win(Title(title), CloseButton(true), OnClose(w.requestClose), Size(mailWindowW, mailWindowH),
		Content(primitives.Box(primitives.HBox(inboxTab, writeTab).Gap(0), primitives.Expanded(content), primitives.Box(rotheme.Text(w.status)).Height(24)).Padding(8).Gap(4).CrossAlign(primitives.CrossAxisStretch)), Footer(footer...))
}

func (w *MailWindow) inboxTree() widget.Widget {
	start := w.page * mailRowsPerPage
	end := min(len(w.inbox), start+mailRowsPerPage)
	rows := append([]network.MailEntry(nil), w.inbox[start:end]...)
	selected := -1
	for i, e := range rows {
		if e.ID == w.selected {
			selected = i
		}
	}
	table := rotheme.TableView(
		rotheme.TableViewColumns([]rotheme.TableViewColumn{{Key: "subject", Title: "Subject", Flex: 1, MinWidth: 150}, {Key: "sender", Title: "From", Width: 112}, {Key: "date", Title: "Date", Width: 85}}),
		rotheme.TableViewRowCount(len(rows)), rotheme.TableViewRowHeight(24), rotheme.TableViewHeaderHeight(24),
		rotheme.TableViewEmptyText("Your mailbox is empty."), rotheme.TableViewSelectedRow(state.NewSignal(selected)),
		rotheme.TableViewBuildSimpleCell(func(cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
			e := rows[cell.Row]
			s := e.Title
			if cell.Column.Key == "sender" {
				s = e.Sender
			} else if cell.Column.Key == "date" {
				s = time.Unix(int64(e.Timestamp), 0).Format("02/01/06")
			}
			color := rotheme.Default.Colors.Text
			if !e.Read {
				color = rotheme.Default.Colors.LabelText
			}
			return rotheme.TableViewSimpleCell{Text: s, Color: color}
		}), rotheme.TableViewOnRowClick(func(row int) {
			if row >= 0 && row < len(rows) && !w.busy {
				w.selected = rows[row].ID
				w.refresh()
			}
		}),
		rotheme.TableViewOnRowEvent(func(row int, e event.Event) bool {
			mouse, ok := e.(*event.MouseEvent)
			if ok && mouse.MouseType == event.MouseDoubleClick && mouse.Button == event.ButtonLeft && row >= 0 && row < len(rows) {
				w.request(MailAction{Kind: MailActionRead, ID: rows[row].ID})
				return true
			}
			return false
		}),
	)
	page := primitives.HBox(rotheme.IconButtonDisabled(rotheme.IconButtonLeft, w.busy || w.page == 0, func() { w.page--; w.refresh() }),
		rotheme.Text(fmt.Sprintf("%d / %d", w.page+1, w.pageCount())), rotheme.IconButtonDisabled(rotheme.IconButtonRight, w.busy || w.page+1 >= w.pageCount(), func() { w.page++; w.refresh() }),
		primitives.Expanded(primitives.Box()), rotheme.Text(fmt.Sprintf("%d / %d", len(w.inbox), network.MailInboxCapacity))).Gap(8).CrossAlign(primitives.CrossAxisCenter)
	return primitives.Box(primitives.Expanded(table), page).Gap(6).CrossAlign(primitives.CrossAxisStretch)
}

func (w *MailWindow) composeTree() widget.Widget {
	disabled := textfield.DisabledFn(func() bool { return w.busy })
	if w.recipientField == nil {
		w.recipientField = rotheme.TextField(w.recipient, textfield.TypeText, func(v string) { w.recipient = v }, nil, textfield.MaxLength(network.MailRecipientMax), disabled)
	}
	if w.titleField == nil {
		w.titleField = rotheme.TextField(w.title, textfield.TypeText, func(v string) { w.title = v }, nil, textfield.MaxLength(network.MailTitleMax), disabled)
	}
	if w.zenyField == nil {
		w.zenyField = rotheme.TextField(w.zeny, textfield.TypeNumber, func(v string) { w.zeny = v }, nil, textfield.MaxLength(10), disabled)
	}
	if w.bodyField == nil {
		w.bodyField = rotheme.TextArea(w.body, network.MailBodyMax, func(v string) { w.body = v })
	}
	w.bodyField.SetEnabled(!w.busy)
	if w.composeSlot == nil || w.composeSlot.item != w.attachment {
		w.composeSlot = w.newAttachmentSlot(w.attachment)
	}
	itemName := "Drag an item here"
	if w.attachment.ItemID != 0 {
		itemName = inventoryItemDisplayName(w.ctx.Resources, w.attachment)
	}
	attachment := primitives.HBox(w.composeSlot, primitives.Expanded(rotheme.Text(itemName)), rotheme.ButtonDisabled("Remove", w.busy || w.attachment.ItemID == 0, func() { w.request(MailAction{Kind: MailActionRemoveAttachment}) })).Gap(8).CrossAlign(primitives.CrossAxisCenter).Height(40)
	return primitives.Box(mailFieldRow("To", w.recipientField), mailFieldRow("Subject", w.titleField), primitives.Expanded(w.bodyField), attachment, mailFieldRow("Zeny", w.zenyField)).Gap(5).CrossAlign(primitives.CrossAxisStretch)
}

func mailFieldRow(label string, field widget.Widget) widget.Widget {
	return primitives.HBox(primitives.Box(rotheme.Label(label)).Width(65), primitives.Expanded(field)).Gap(6).Height(24).CrossAlign(primitives.CrossAxisCenter)
}

func (w *MailWindow) send() {
	if w.busy {
		return
	}
	if _, err := network.BuildMailSendPacket(w.recipient, w.title, w.body); err != nil {
		w.ShowStatus(err.Error())
		return
	}
	zeny := uint64(0)
	if strings.TrimSpace(w.zeny) != "" {
		var err error
		zeny, err = strconv.ParseUint(strings.TrimSpace(w.zeny), 10, 32)
		if err != nil {
			w.ShowStatus("Enter a valid Zeny amount.")
			return
		}
	}
	if w.ctx.Session == nil || zeny > uint64(max(0, w.ctx.Session.Inventory.Zeny)) {
		w.ShowStatus("Not enough Zeny.")
		return
	}
	w.request(MailAction{Kind: MailActionSend, Recipient: w.recipient, Title: w.title, Body: w.body, Zeny: uint32(zeny)})
}

func (w *MailWindow) readTree() widget.Widget {
	m := *w.message
	hasAttachment := m.HasAttachment()
	item := mailInventoryItem(m.Attachment)
	itemName := "No attached item"
	if item.ItemID != 0 {
		itemName = inventoryItemDisplayName(w.ctx.Resources, item)
	}
	return Win(Title("Read Mail"), CloseButton(true), OnClose(w.closeRead), Size(mailReadW, mailReadH),
		Content(primitives.Box(rotheme.Label(m.Title), rotheme.Text("From: "+m.Sender), primitives.Expanded(w.readBodyField),
			primitives.HBox(w.newAttachmentSlot(item), primitives.Expanded(rotheme.Text(itemName))).Gap(8).Height(40).CrossAlign(primitives.CrossAxisCenter),
			rotheme.Text(fmt.Sprintf("Zeny: %s", formatHUDNumber(int64(m.Zeny)))), rotheme.Text(w.status)).Padding(10).Gap(6).CrossAlign(primitives.CrossAxisStretch)),
		Footer(rotheme.ButtonDisabled("Get", w.busy || !hasAttachment, func() { w.request(MailAction{Kind: MailActionTake, ID: m.ID}) }),
			rotheme.ButtonDisabled("Reply", w.busy, func() {
				title := m.Title
				if !strings.HasPrefix(strings.ToLower(title), "re:") {
					title = "Re: " + title
				}
				w.request(MailAction{Kind: MailActionCompose, Recipient: m.Sender, Title: title})
			}),
			primitives.Expanded(primitives.Box()),
			rotheme.ButtonDisabled("Return", w.busy, func() {
				w.confirm.Open(w.ctx, "Return Mail", "Return this mail and its attachments to the sender?", func() { w.request(MailAction{Kind: MailActionReturn, ID: m.ID}) }, nil)
			}),
			rotheme.ButtonDisabled("Delete", w.busy || hasAttachment, func() {
				w.confirm.Open(w.ctx, "Delete Mail", "Delete this message permanently?", func() { w.request(MailAction{Kind: MailActionDelete, ID: m.ID}) }, nil)
			})))
}

func (w *MailWindow) AcceptInventoryDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	if !w.IsOpen() {
		return false
	}
	if manager, ok := ctx.UIManager.(interface{ OverlayAt(int, int) widget.Widget }); ok {
		top := manager.OverlayAt(mx, my)
		if top != w.published && top != w.readWindow.published {
			return false
		}
	}
	if w.readWindow.IsOpen() && pointInRect(mx, my, w.readWindow.x, w.readWindow.y, mailReadW, mailReadH) {
		return true
	}
	if !pointInRect(mx, my, w.x, w.y, mailWindowW, mailWindowH) {
		return false
	}
	// Dropping on the inbox, footer or a busy composer is still a UI drop,
	// never a request to discard the item on the ground behind the window.
	if !w.compose || w.composeSlot == nil || !w.composeSlot.ScreenBounds().Contains(geometry.Pt(float32(mx), float32(my))) {
		return true
	}
	if w.busy {
		return true
	}
	if w.attachment.ItemID != 0 {
		w.ShowStatus("Remove the attached item first.")
		return true
	}
	if item.Index < 2 || item.Amount < 1 || item.Equipped {
		w.ShowStatus("This item cannot be attached.")
		return true
	}
	request := func(amount uint16) {
		item.Amount = int(amount)
		w.request(MailAction{Kind: MailActionAttach, Item: item})
	}
	if item.Amount > 1 {
		w.amount.Open(ctx, "Amount to attach", 1, uint16(min(item.Amount, 32767)), request)
	} else {
		request(1)
	}
	return true
}

func mailInventoryItem(item network.MailAttachment) session.InventoryItem {
	return session.InventoryItem{ItemID: item.ItemID, Amount: int(item.Amount), Type: uint8(item.Type), Identified: item.Identified, Damaged: item.Damaged, Refine: item.Refine, Cards: item.Cards}
}

func mailTruncateText(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.RuneStart(value[maxBytes]) {
		maxBytes--
	}
	return value[:maxBytes]
}

type mailAttachmentSlot struct {
	widget.WidgetBase
	owner *MailWindow
	item  session.InventoryItem
	icon  image.Image
}

func (w *MailWindow) newAttachmentSlot(item session.InventoryItem) *mailAttachmentSlot {
	slot := &mailAttachmentSlot{owner: w, item: item, icon: tradeItemIconImage(w.ctx.Resources, item)}
	slot.SetVisible(true)
	slot.SetEnabled(true)
	return slot
}
func (s *mailAttachmentSlot) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	size := c.Constrain(geometry.Sz(40, 40))
	s.SetBounds(geometry.FromPointSize(s.Position(), size))
	return size
}
func (s *mailAttachmentSlot) Draw(_ widget.Context, canvas widget.Canvas) {
	b := s.Bounds()
	drawInventoryGridCellShadow(canvas, b, false)
	if s.icon != nil {
		ib := s.icon.Bounds()
		canvas.DrawImage(s.icon, geometry.Pt(b.Min.X+(b.Width()-float32(ib.Dx()))/2, b.Min.Y+(b.Height()-float32(ib.Dy()))/2))
	}
	if s.item.Amount > 1 {
		rotheme.DrawText(canvas, strconv.Itoa(s.item.Amount), geometry.NewRect(b.Min.X, b.Max.Y-14, b.Width()-2, 14), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignRight)
	}
}
func (s *mailAttachmentSlot) Event(_ widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if mouse.MouseType == event.MousePress && mouse.Button == event.ButtonRight && s.item.ItemID != 0 && s.owner.itemInfo != nil && s.owner.ctx.Input != nil {
		s.owner.itemInfo.openItem(s.owner.ctx, s.item, s.owner.ctx.Input.MouseX, s.owner.ctx.Input.MouseY)
	}
	return true
}
