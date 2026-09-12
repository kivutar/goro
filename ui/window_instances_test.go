package ui

import (
	"os"
	"path/filepath"
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

type instanceTestApp struct{ basicMenuTestApp }

func (a instanceTestApp) WidgetContext() widget.Context { return a.app.Window().Context() }

func newWindowInstanceTest() (Context, *Manager, *uiapp.App) {
	app := uiapp.New()
	bridge := instanceTestApp{basicMenuTestApp{app: app}}
	manager := NewManager()
	manager.SetUIApp(bridge)
	return Context{UIApp: bridge, UIManager: manager, Input: input.NewState(), ScreenW: 1024, ScreenH: 768}, manager, app
}

func TestItemDescriptionsAndCardArtworkAreIndependent(t *testing.T) {
	ctx, manager, app := newWindowInstanceTest()
	ctx.Resources = cardIllustrationTestManager(t)
	var windows ItemWindows
	item := session.InventoryItem{ItemID: 2607, Type: db.ItemTypeArmor, Location: db.EquipAccessory1, Equip: true, Identified: true, Refine: 3, Cards: [4]uint16{4001}}
	windows.openItem(ctx, item, 100, 100)
	item.Refine = 7
	windows.openItem(ctx, item, 100, 100)
	first, second := windows.descriptions[0], windows.descriptions[1]
	if first == second || first.item.Refine != 3 || second.item.Refine != 7 || len(manager.overlays) != 2 {
		t.Fatal("descriptions did not retain independent item snapshots")
	}
	if first.x == second.x && first.y == second.y {
		t.Fatal("new description completely covered the previous title bar")
	}

	first.Raise(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	slot := findInstanceCardSlot(first.content)
	if slot == nil {
		t.Fatal("slotted equipment has no card widget")
	}
	p := slot.ScreenBounds().Center()
	app.Window().HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonRight, event.ButtonStateRight, p, p, event.ModNone))
	app.Window().HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonRight, 0, p, p, event.ModNone))
	if consumed, err := windows.Update(ctx, nil); !consumed || err != nil {
		t.Fatalf("opening slotted card: consumed=%t err=%v", consumed, err)
	}
	if len(windows.descriptions) != 3 || first.item.Refine != 3 || first.item.ItemID != 2607 {
		t.Fatal("opening a slotted card replaced the equipment description")
	}
	card := windows.descriptions[2]
	if card.item.ItemID != 4001 {
		t.Fatalf("card description item = %d", card.item.ItemID)
	}
	for range 2 {
		card.Raise(ctx)
		app.Frame()
		app.Window().DrawTo(&uitest.MockCanvas{})
		x := float32(card.x + ROWindowFooterPadding + 20)
		y := float32(card.y + card.height - ROWindowFooterHeight/2)
		app.Window().HandleEvent(uitest.Click(x, y))
		app.Window().HandleEvent(uitest.Release(x, y))
		if consumed, err := windows.Update(ctx, nil); !consumed || err != nil {
			t.Fatalf("opening card artwork: consumed=%t err=%v", consumed, err)
		}
	}
	if len(windows.illustrations) != 2 || len(manager.overlays) != 5 {
		t.Fatalf("artwork instances=%d overlays=%d", len(windows.illustrations), len(manager.overlays))
	}
	art := windows.illustrations[0]
	art.Close()
	first.Close()
	windows.Update(ctx, nil)
	if len(windows.descriptions) != 2 || len(windows.illustrations) != 1 || len(manager.overlays) != 3 {
		t.Fatal("closing instances did not remove only those instances")
	}
	before := append([]widget.Widget(nil), manager.overlays...)
	windows.Rebind(ctx, nil)
	for i, root := range before {
		if manager.overlays[i] != root {
			t.Fatal("rebinding changed window identity or stacking order")
		}
	}
}

func findInstanceCardSlot(root widget.Widget) *itemInfoCardSlotWidget {
	if slot, ok := root.(*itemInfoCardSlotWidget); ok && slot.cardID != 0 {
		return slot
	}
	for _, child := range root.Children() {
		if slot := findInstanceCardSlot(child); slot != nil {
			return slot
		}
	}
	return nil
}

func TestReadingClosesOnlyOriginatingDescriptionAndRaisesReader(t *testing.T) {
	ctx, manager, app := newWindowInstanceTest()
	root := t.TempDir()
	path := filepath.Join(root, "data", "book", "7277.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("Book contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx.Resources = &res.Manager{Root: root}
	var windows ItemWindows
	if err := windows.book.Open(ctx, 7277, "Already open book"); err != nil {
		t.Fatal(err)
	}
	reader := windows.book.published
	windows.openItem(ctx, session.InventoryItem{ItemID: 501, Identified: true}, 100, 100)
	other := windows.descriptions[0]
	windows.openItem(ctx, session.InventoryItem{ItemID: 7277, Identified: true}, 100, 100)
	description := windows.descriptions[1]
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	x := float32(description.x + ROWindowFooterPadding + 20)
	y := float32(description.y + description.height - ROWindowFooterHeight/2)
	app.Window().HandleEvent(uitest.Click(x, y))
	app.Window().HandleEvent(uitest.Release(x, y))
	if consumed, err := windows.Update(ctx, nil); !consumed || err != nil {
		t.Fatalf("reading book: consumed=%t err=%v", consumed, err)
	}
	if description.IsOpen() || !other.IsOpen() || !windows.book.IsOpen() || windows.book.published != reader || manager.TopEscapeOverlay() != reader || len(manager.overlays) != 2 {
		t.Fatal("reading replaced an unrelated description or failed to raise the existing reader")
	}
}

func TestWindowPollingRespectsStackingAndEscape(t *testing.T) {
	ctx, manager, app := newWindowInstanceTest()
	lower, upper := NewWindow(200, 150), NewWindow(200, 150)
	lower.OpenAt(100, 100, primitives.Box())
	upper.OpenAt(130, 110, primitives.Box())
	lower.Publish(ctx)
	upper.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	ctx.Input.SetMousePosition(140, 115)
	ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
	if lower.Update(ctx) || lower.dragging {
		t.Fatal("lower window intercepted a drag on the visible window")
	}
	if !upper.Update(ctx) || !upper.dragging {
		t.Fatal("visible window did not start dragging")
	}
	ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
	ctx.Input.EndFrame()
	upper.Update(ctx)

	lower.Raise(ctx)
	ctx.Input.SetMousePosition(150, 140) // Over both windows while closing the raised one.
	ctx.Input.SetKey(input.KeyEscape, true)
	var menu EscapeMenu
	if menu.Update(ctx) || upper.Update(ctx) {
		t.Fatal("Escape was intercepted before reaching the top window")
	}
	if !lower.Update(ctx) || lower.IsOpen() || !upper.IsOpen() {
		t.Fatal("Escape did not close only the top window")
	}
	if manager.TopEscapeOverlay() != upper.published {
		t.Fatal("closed window remained in the Escape stack")
	}
}

func TestHUDDoesNotPreventOpeningEscapeMenu(t *testing.T) {
	ctx, _, _ := newWindowInstanceTest()
	ctx.Session = &session.Session{}
	var character CharacterWindow
	var basic BasicMenu
	var console ChatConsole
	character.Update(ctx)
	basic.Update(ctx, BasicMenuCallbacks{})
	console.Publish(ctx)
	ctx.Input.SetKey(input.KeyEscape, true)
	var menu EscapeMenu
	if !menu.Update(ctx) || !menu.IsOpen() || !character.IsOpen() || !basic.IsOpen() {
		t.Fatal("HUD windows consumed Escape before the menu could open")
	}
}

func TestEscapeRunsWindowCleanupOnce(t *testing.T) {
	ctx, manager, _ := newWindowInstanceTest()
	ctx.Session = &session.Session{Selected: session.Character{ID: 150004, Option: inventoryBagCartOptionMask}}
	var cart CartWindow
	cart.OpenWindow(ctx)
	ctx.Input.SetKey(input.KeyEscape, true)
	if !cart.Update(ctx, nil, nil, nil) || cart.IsOpen() || ctx.Session.Cart.Open {
		t.Fatal("Escape bypassed the cart's close action")
	}
	ctx.Input.ResetKeyboard()
	var mail MailWindow
	mail.Open(ctx, nil)
	ctx.Input.SetKey(input.KeyEscape, true)
	if !mail.Update(ctx, nil) || mail.IsOpen() {
		t.Fatal("Escape did not close the mailbox")
	}
	if action := mail.PopAction(); action.Kind != MailActionClose {
		t.Fatalf("mail close action = %+v", action)
	}
	if action := mail.PopAction(); action.Kind != MailActionNone {
		t.Fatalf("mail close action ran more than once: %+v", action)
	}
	if manager.TopEscapeOverlay() != nil {
		t.Fatal("closed transaction window remained in the Escape stack")
	}
}

func TestWhisperInstancesKeepDraftsFocusAndRecipients(t *testing.T) {
	ctx, manager, app := newWindowInstanceTest()
	var windows WhisperWindows
	alice := windows.Open(ctx, "Alice")
	app.Window().HandleEvent(uitest.KeyType(event.KeyA, 'a', event.ModNone))
	bob := windows.Open(ctx, "Bob")
	app.Window().HandleEvent(uitest.KeyType(event.KeyB, 'b', event.ModNone))
	if alice.input != "a" || bob.input != "b" || alice.inputField.IsFocused() || !bob.inputField.IsFocused() {
		t.Fatalf("independent drafts/focus: Alice=%q Bob=%q", alice.input, bob.input)
	}
	if windows.Open(ctx, " alice ") != alice || len(manager.overlays) != 2 {
		t.Fatal("opening the same recipient duplicated the conversation")
	}
	app.Window().HandleEvent(uitest.KeyType(event.KeyA, 'a', event.ModNone))
	windows.AddIncoming(ctx, "Bob", "hello")
	windows.AddIncoming(ctx, "Alice", "message while typing")
	windows.AddIncoming(ctx, "Carol", "new conversation")
	if !alice.inputField.IsFocused() || bob.inputField.IsFocused() || windows.Find("Carol").inputField.IsFocused() {
		t.Fatal("incoming message stole keyboard focus")
	}
	app.Window().HandleEvent(uitest.KeyType(event.KeyC, 'c', event.ModNone))
	if alice.input != "aac" || bob.input != "b" {
		t.Fatal("incoming message redirected typing to a different recipient")
	}
	// The queued submit must be drained even while the pointer is over another window.
	windows.Open(ctx, "Bob")
	ctx.Input.SetMousePosition(alice.x+5, alice.y+40)
	app.Window().HandleEvent(uitest.KeyPress(event.KeyEnter, event.ModNone))
	ctx.Input.SetKey(input.KeyEnter, true)
	consumed, action := windows.Update(ctx)
	if !consumed || action.Target != "Bob" || action.Message != "b" {
		t.Fatalf("submit went to the wrong conversation: %+v", action)
	}
	if _, duplicate := windows.Update(ctx); duplicate.Target != "" {
		t.Fatalf("Enter submitted twice: %+v", duplicate)
	}
	ctx.Input.ResetKeyboard()
	alice.Close()
	if windows.Open(ctx, "Alice") != alice || alice.input != "aac" || len(bob.lines) != 1 {
		t.Fatal("closing/reopening a conversation lost a draft or another conversation")
	}
	before := append([]widget.Widget(nil), manager.overlays...)
	windows.Rebind(ctx)
	for i, root := range before {
		if manager.overlays[i] != root {
			t.Fatal("whisper rebind changed stacking order")
		}
	}
	if !alice.inputField.IsFocused() || bob.inputField.IsFocused() {
		t.Fatal("whisper rebind changed the focused conversation")
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	windows.Update(ctx)
	if alice.IsOpen() || alice.inputField.IsFocused() || !bob.IsOpen() {
		t.Fatal("Escape did not close and blur only the top conversation")
	}
}

func TestNewWindowPlacementStaysOnScreen(t *testing.T) {
	ctx, _, _ := newWindowInstanceTest()
	for range 40 {
		w := NewWindow(300, 428)
		w.Open(ctx, primitives.Box())
		w.Publish(ctx)
		bounds := geometry.NewRect(float32(w.x), float32(w.y), float32(w.width), float32(w.height))
		if bounds.Min.X < windowScreenMargin || bounds.Min.Y < windowScreenMargin || bounds.Max.X > float32(ctx.ScreenW-windowScreenMargin) || bounds.Max.Y > float32(ctx.ScreenH-windowScreenMargin) {
			t.Fatalf("window escaped screen bounds: %v", bounds)
		}
	}
}
