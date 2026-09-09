package ui

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

func newMailWindowTest(t *testing.T) (*MailWindow, Context, *Manager) {
	t.Helper()
	manager := NewManager()
	ctx := Context{ScreenW: 1024, ScreenH: 768, UIManager: manager, Input: input.NewState(), Session: &session.Session{Inventory: session.Inventory{Zeny: 1000}}}
	w := &MailWindow{}
	w.Open(ctx)
	drawMailWindowTest(manager)
	return w, ctx, manager
}

func drawMailWindowTest(manager *Manager) {
	ctx := widget.NewContext()
	manager.root.Layout(ctx, geometry.Tight(geometry.Sz(1024, 768)))
	manager.root.Draw(ctx, &uitest.MockCanvas{})
}

func TestMailWindowTabsSitFlushBelowTitle(t *testing.T) {
	for _, compose := range []bool{false, true} {
		t.Run(fmt.Sprintf("compose=%t", compose), func(t *testing.T) {
			w, _, manager := newMailWindowTest(t)
			if compose {
				w.BeginCompose("Erika", "Hello")
			}
			drawMailWindowTest(manager)
			children := w.content.Children()
			title := children[0].(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
			frame := children[1].Children()[0]
			strip := frame.Children()[0]
			bounds := strip.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
			if bounds.Min.Y != title.Max.Y || bounds.Min.X != title.Min.X || bounds.Max.X != title.Max.X {
				t.Fatalf("tab strip %v must meet title bottom and sides %v", bounds, title)
			}
			inbox := strip.Children()[0].(*tabWidget).ScreenBounds()
			write := strip.Children()[1].(*tabWidget).ScreenBounds()
			if inbox.Min != bounds.Min || inbox.Max.Y != bounds.Max.Y || write.Min.Y != bounds.Min.Y || write.Max.Y != bounds.Max.Y || write.Min.X != inbox.Max.X-1 {
				t.Fatalf("tabs must fill the strip height and share their border: inbox=%v write=%v", inbox, write)
			}
			divider := frame.Children()[1]
			dividerBounds := divider.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
			if want := geometry.NewRect(bounds.Min.X, bounds.Max.Y, bounds.Width(), 1); dividerBounds != want {
				t.Fatalf("tab divider = %v, want %v", dividerBounds, want)
			}
			canvas := &uitest.MockCanvas{}
			divider.Draw(widget.NewContext(), canvas)
			if len(canvas.Rects) != 1 {
				t.Fatalf("tab divider draws = %d, want one solid line", len(canvas.Rects))
			}
			uitest.AssertColorEqual(t, canvas.Rects[0].Color, rotheme.Default.Colors.WindowBorder)
			body := frame.Children()[2].Children()[0]
			page := body.Children()[0].Children()[0]
			padding := float32(8)
			if !compose {
				padding = 0
				page = page.Children()[0].Children()[0] // Inbox table, including its header.
			}
			pageBounds := page.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
			if pageBounds.Min.Y != dividerBounds.Max.Y+padding || pageBounds.Min.X != bounds.Min.X+padding || pageBounds.Max.X != bounds.Max.X-padding {
				t.Fatalf("page %v must follow tab divider %v with %gpx padding", pageBounds, dividerBounds, padding)
			}
		})
	}
}

func TestMailWindowReservesStatusSpaceOnlyForMessages(t *testing.T) {
	for _, compose := range []bool{false, true} {
		t.Run(fmt.Sprintf("compose=%t", compose), func(t *testing.T) {
			w, _, manager := newMailWindowTest(t)
			if compose {
				w.BeginCompose("Erika", "Hello")
			}
			for _, status := range []string{"", "Waiting for the mail server...", "Mail was not sent.", ""} {
				w.ShowStatus(status)
				drawMailWindowTest(manager)
				children := w.content.Children()
				footer := children[2].(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
				frame := children[1].Children()[0]
				body := frame.Children()[2].Children()[0]
				wantChildren := 1
				wantGap := float32(0)
				if compose {
					wantGap = 8
				}
				if status != "" {
					wantChildren = 2
					wantGap += 24 + 4
				}
				if len(body.Children()) != wantChildren {
					t.Fatalf("status %q: body children = %d, want %d", status, len(body.Children()), wantChildren)
				}
				page := body.Children()[0].(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
				if gap := footer.Min.Y - page.Max.Y; gap != wantGap {
					t.Fatalf("status %q: space above footer = %g, want %g", status, gap, wantGap)
				}
			}
		})
	}
}

func TestMailWindowIdleUpdateKeepsPublishedContent(t *testing.T) {
	for _, mode := range []string{"inbox", "read", "compose", "closed"} {
		t.Run(mode, func(t *testing.T) {
			w, ctx, manager := newMailWindowTest(t)
			switch mode {
			case "read":
				w.ShowMail(network.MailMessage{ID: 1, Sender: "Erika", Title: "Hello", Body: "A message"})
			case "compose":
				w.BeginCompose("Erika", "Hello")
			case "closed":
				w.CloseFromServer(ctx)
			}
			drawMailWindowTest(manager)
			content, readContent := w.content, w.readWindow.content
			published, readPublished := w.published, w.readWindow.published
			allocs := testing.AllocsPerRun(100, func() { w.Update(ctx, nil) })
			if allocs != 0 {
				t.Fatalf("idle mail update allocated %g objects", allocs)
			}
			if w.content != content || w.readWindow.content != readContent || w.published != published || w.readWindow.published != readPublished {
				t.Fatal("idle update rebuilt mail content or its published windows")
			}
		})
	}
}

func TestMailWindowInboxPaginationAndReadIntent(t *testing.T) {
	for _, count := range []int{0, 1, 7, 8, 30} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			w, _, manager := newMailWindowTest(t)
			entries := make([]network.MailEntry, count)
			for i := range entries {
				entries[i] = network.MailEntry{ID: uint32(i + 1), Title: fmt.Sprintf("Subject %d", i+1), Sender: "Erika", Timestamp: 1150000000}
			}
			w.SetInbox(entries)
			if got := w.pageCount(); got != max(1, (count+6)/7) {
				t.Fatalf("page count = %d", got)
			}
			w.page = w.pageCount() - 1
			w.refresh()
			drawMailWindowTest(manager)
			if count == 0 {
				return
			}
			// Route through the published, positioned window and shared table.
			position := geometry.Pt(float32(w.x+28), float32(w.y+ROWindowTitleHeight+24+1+24+12))
			if !manager.root.Event(widget.NewContext(), event.NewMouseEvent(event.MouseDoubleClick, event.ButtonLeft, event.ButtonStateLeft, position, position, event.ModNone)) {
				t.Fatal("mail row click leaked through the window")
			}
			action := w.PopAction()
			wantID := uint32(w.page*7 + 1)
			if action.Kind != MailActionRead || action.ID != wantID {
				t.Fatalf("read action = %+v, want id %d", action, wantID)
			}
			w.request(MailAction{Kind: MailActionRead, ID: wantID})
			if next := w.PopAction(); next.Kind != MailActionNone {
				t.Fatalf("busy window queued a duplicate: %+v", next)
			}
			w.SetBusy(false)
			for _, entry := range entries {
				w.RemoveMail(entry.ID)
			}
			if w.page != 0 || w.pageCount() != 1 {
				t.Fatal("deleting the final page did not clamp pagination")
			}
			drawMailWindowTest(manager)
		})
	}
}

func TestMailWindowDraftSurvivesStatusAndIncomingMail(t *testing.T) {
	w, _, manager := newMailWindowTest(t)
	w.BeginCompose("Erika", "Hello")
	w.body = "Some text\nNext line"
	w.bodyField = rotheme.TextArea(w.body, network.MailBodyMax, func(v string) { w.body = v })
	w.zeny = "123"
	w.refresh()
	field := w.bodyField
	w.SetBusy(true)
	w.ShowStatus("Waiting...")
	w.SetInbox([]network.MailEntry{{ID: 2, Title: "Incoming", Sender: "Zambla"}})
	w.SetBusy(false)
	drawMailWindowTest(manager)
	if w.bodyField != field || w.body != "Some text\nNext line" || w.recipient != "Erika" || w.title != "Hello" || w.zeny != "123" {
		t.Fatal("inbox/status update reset the draft editor")
	}
	w.send()
	got := w.PopAction()
	if got.Kind != MailActionSend || got.Body != w.body || got.Zeny != 123 || got.Recipient != "Erika" {
		t.Fatalf("send = %+v", got)
	}
}

func TestMailWindowValidatesTextAndZeny(t *testing.T) {
	for _, tc := range []struct{ name, recipient, title, zeny string }{
		{"empty recipient", "", "Subject", "0"},
		{"empty title", "Erika", "", "0"},
		{"too many bytes", "Erika", strings.Repeat("é", 20), "0"},
		{"too much Zeny", "Erika", "Subject", "1001"},
		{"negative Zeny", "Erika", "Subject", "-1"},
		{"overflow Zeny", "Erika", "Subject", "4294967296"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, _, _ := newMailWindowTest(t)
			w.BeginCompose(tc.recipient, "")
			w.title, w.zeny = tc.title, tc.zeny
			w.send()
			if w.PopAction().Kind != MailActionNone || w.status == "" || w.busy {
				t.Fatal("invalid draft queued or failed silently")
			}
		})
	}
}

func TestMailWindowAmountEscapeAndCloseLifecycle(t *testing.T) {
	w, ctx, manager := newMailWindowTest(t)
	w.BeginCompose("Erika", "Hi")
	drawMailWindowTest(manager)
	center := w.composeSlot.ScreenBounds().Center()
	item := session.InventoryItem{Index: 7, ItemID: 501, Amount: 10, Identified: true}
	if !w.AcceptInventoryDrop(ctx, item, int(center.X), int(center.Y)) || !w.amount.IsOpen() {
		t.Fatal("stack drop did not open amount prompt")
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	if !w.UpdateModal(ctx) || w.ModalOpen() || !w.IsComposing() || !w.IsOpen() {
		t.Fatal("Escape did not cancel only the amount prompt")
	}
	if w.PopAction().Kind != MailActionNone {
		t.Fatal("Escape attached the item")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKey(input.KeyEscape, false)
	drawMailWindowTest(manager)
	if !w.AcceptInventoryDrop(ctx, item, int(center.X), int(center.Y)) {
		t.Fatal("second drop not accepted")
	}
	w.amount.value = "3"
	w.amount.submit(ctx)
	got := w.PopAction()
	if got.Kind != MailActionAttach || got.Item.Amount != 3 || got.Item.Index != 7 {
		t.Fatalf("attach amount = %+v", got)
	}
	w.requestClose()
	if w.IsOpen() || w.readWindow.IsOpen() || w.ModalOpen() || len(manager.overlays) != 0 {
		t.Fatal("close left mail overlays alive")
	}
	if w.PopAction().Kind != MailActionClose {
		t.Fatal("close lost reset intent")
	}
}

func TestMailWindowDropsNeverDiscardItems(t *testing.T) {
	w, ctx, manager := newMailWindowTest(t)
	item := session.InventoryItem{Index: 7, ItemID: 501, Amount: 10}
	// Inbox, footer, and busy composer must swallow invalid drops.
	for _, compose := range []bool{false, true} {
		if compose {
			w.BeginCompose("", "")
			w.SetBusy(true)
		}
		drawMailWindowTest(manager)
		for _, p := range []geometry.Point{geometry.Pt(float32(w.x+20), float32(w.y+90)), geometry.Pt(float32(w.x+20), float32(w.y+mailWindowH-15))} {
			ctx.Input.SetMousePosition(int(p.X), int(p.Y))
			inventory := &InventoryBagWindow{dragActive: true, dragItem: item}
			if !inventory.UpdateDrag(ctx, nil, nil, nil, nil, nil, w) {
				t.Fatal("mail drop did not consume drag release")
			}
			if inventory.amountPrompt.IsOpen() || w.PopAction().Kind != MailActionNone {
				t.Fatal("mail drop became a ground drop or attachment")
			}
		}
	}
}

func TestMailWindowDoesNotAcceptDropsThroughOtherWindows(t *testing.T) {
	w, ctx, manager := newMailWindowTest(t)
	w.BeginCompose("", "")
	drawMailWindowTest(manager)
	p := w.composeSlot.ScreenBounds().Center()
	cover := NewWindow(90, 70)
	cover.OpenAt(int(p.X)-40, int(p.Y)-30, newInertOverlay())
	cover.Publish(ctx)
	drawMailWindowTest(manager)
	if manager.OverlayAt(int(p.X), int(p.Y)) != cover.published {
		t.Fatal("wrong drop z-order")
	}
	if w.AcceptInventoryDrop(ctx, session.InventoryItem{Index: 7, ItemID: 501, Amount: 1}, int(p.X), int(p.Y)) {
		t.Fatal("mail accepted item through foreground window")
	}
	if w.amount.IsOpen() || w.PopAction().Kind != MailActionNone {
		t.Fatal("occluded drop changed draft")
	}
}

func TestMailWindowReadStateAndReplyTitle(t *testing.T) {
	w, ctx, manager := newMailWindowTest(t)
	w.SetInbox([]network.MailEntry{{ID: 42, Title: "Hello", Sender: "Erika"}})
	w.ShowMail(network.MailMessage{ID: 42, Title: "Hello", Sender: "Erika", Body: "First\nSecond", Zeny: 5, Attachment: network.MailAttachment{ItemID: 501, Amount: 3}})
	drawMailWindowTest(manager)
	if !w.inbox[0].Read || !w.readWindow.IsOpen() {
		t.Fatal("read message did not update read state")
	}
	w.UpdateReadMessage(network.MailMessage{ID: 43})
	if w.message.Zeny != 5 {
		t.Fatal("cleared unrelated mail")
	}
	w.UpdateReadMessage(network.MailMessage{ID: 42, Title: "Hello", Sender: "Erika", Body: "First\nSecond"})
	if w.message.Zeny != 0 || w.message.Attachment.ItemID != 0 {
		t.Fatal("received attachment remains in reader")
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	if !w.Update(ctx, nil) || w.readWindow.IsOpen() || !w.IsOpen() {
		t.Fatal("Escape should close the reader before the inbox")
	}
	w.BeginCompose("Erika", "Re: "+strings.Repeat("é", 25))
	if len(w.title) > network.MailTitleMax || !utf8.ValidString(w.title) || !strings.HasPrefix(w.title, "Re: ") {
		t.Fatalf("invalid reply title %q", w.title)
	}
}

func TestMailComposerFocusAndMultilineThroughPublishedUI(t *testing.T) {
	w, ctx, manager := newMailWindowTest(t)
	a := uiapp.New()
	bridge := mailWindowTestApp{basicMenuTestApp{app: a}}
	manager.SetUIApp(bridge)
	ctx.UIApp = bridge
	w.ctx = ctx
	w.BeginCompose("Erika", "Hi")
	a.Frame()
	a.Window().DrawTo(&uitest.MockCanvas{})
	p := w.bodyField.ScreenBounds().Min.Add(geometry.Pt(12, 12))
	a.Window().HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, p, p, event.ModNone))
	a.Window().HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, p, p, event.ModNone))
	a.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'x', event.ModNone))
	a.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	a.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'y', event.ModNone))
	if w.body != "x\ny" || w.recipient != "Erika" || !w.bodyField.IsFocused() || w.recipientField.IsFocused() {
		t.Fatalf("wrong editor focus/body: body=%q recipient=%q focused=%t/%t", w.body, w.recipient, w.bodyField.IsFocused(), w.recipientField.IsFocused())
	}
	a.Frame()
	a.Window().DrawTo(&uitest.MockCanvas{})
	from := w.bodyField.ScreenBounds().Min.Add(geometry.Pt(6, 10))
	to := w.bodyField.ScreenBounds().Min.Add(geometry.Pt(20, 30))
	a.Window().HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, from, from, event.ModNone))
	a.Window().HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonLeft, event.ButtonStateLeft, to, to, event.ModNone))
	a.Window().HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, to, to, event.ModNone))
	a.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'z', event.ModNone))
	if w.body != "z" {
		t.Fatalf("captured selection used wrong coordinates: %q", w.body)
	}
	a.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModNone))
	if !w.zenyField.IsFocused() || w.bodyField.IsFocused() {
		t.Fatal("Tab did not move from body to Zeny field")
	}
	w.requestClose()
	if a.Window().Context().FocusedWidget() != nil {
		t.Fatal("closing mail retained a detached editor as the focused widget")
	}
}

type mailWindowTestApp struct{ basicMenuTestApp }

func (a mailWindowTestApp) WidgetContext() widget.Context { return a.app.Window().Context() }

func TestMailDeleteConfirmationDoesNotDeleteBeforeAcknowledgement(t *testing.T) {
	w, ctx, manager := newMailWindowTest(t)
	w.ShowMail(network.MailMessage{ID: 42, Sender: "Erika", Title: "Hello"})
	drawMailWindowTest(manager)
	clickDelete := func() {
		p := geometry.Pt(float32(w.readWindow.x+mailReadW-45), float32(w.readWindow.y+mailReadH-21))
		wc := widget.NewContext()
		manager.root.Event(wc, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, p, p, event.ModNone))
		manager.root.Event(wc, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, p, p, event.ModNone))
	}
	clickDelete()
	if !w.confirm.IsOpen() || w.PopAction().Kind != MailActionNone {
		t.Fatal("Delete did not ask for confirmation")
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	w.UpdateModal(ctx)
	if w.confirm.IsOpen() || w.message == nil || w.PopAction().Kind != MailActionNone {
		t.Fatal("Escape confirmed deletion")
	}
	drawMailWindowTest(manager)
	clickDelete()
	w.confirm.Confirm(ctx)
	action := w.PopAction()
	if action.Kind != MailActionDelete || action.ID != 42 || w.message == nil || !w.readWindow.IsOpen() {
		t.Fatalf("delete intent/state = %+v", action)
	}
}
