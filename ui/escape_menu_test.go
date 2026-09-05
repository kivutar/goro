package ui

import (
	"testing"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

type escapeMenuTestUIManager struct {
	overlays []widget.Widget
}

func (m *escapeMenuTestUIManager) AddOverlay(root widget.Widget) {
	m.overlays = append(m.overlays, root)
}

func (m *escapeMenuTestUIManager) RemoveOverlay(root widget.Widget) {
	for i, overlay := range m.overlays {
		if overlay == root {
			m.overlays = append(m.overlays[:i], m.overlays[i+1:]...)
			return
		}
	}
}

func (m *escapeMenuTestUIManager) Clear() {
	m.overlays = nil
}

func TestEscapeMenuOpenPublishesGogpuWindow(t *testing.T) {
	inputState := input.NewState()
	manager := &escapeMenuTestUIManager{}
	menu := EscapeMenu{}
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}
	menu.Toggle(ctx)

	if !menu.Update(ctx) {
		t.Fatal("escape menu did not consume update while open")
	}
	if len(manager.overlays) != 1 {
		t.Fatalf("escape menu overlays = %d, want 1", len(manager.overlays))
	}
}

func TestEscapeMenuToggleClosesOpenWindow(t *testing.T) {
	manager := &escapeMenuTestUIManager{}
	menu := EscapeMenu{}
	ctx := client.Context{Input: input.NewState(), UIManager: manager, ScreenW: 800, ScreenH: 600}
	menu.Toggle(ctx)

	menu.Toggle(ctx)

	if menu.IsOpen() {
		t.Fatal("escape menu stayed open after toggle")
	}
	if len(manager.overlays) != 0 {
		t.Fatalf("escape menu overlays = %d, want 0", len(manager.overlays))
	}
}

func TestEscapeMenuEscapeKeyTogglesWindow(t *testing.T) {
	manager := &escapeMenuTestUIManager{}
	inputState := input.NewState()
	menu := EscapeMenu{}
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}

	inputState.SetKey(input.KeyEscape, true)
	if !menu.Update(ctx) {
		t.Fatal("escape key did not open menu")
	}
	if !menu.IsOpen() {
		t.Fatal("menu did not open from escape key")
	}

	inputState.EndFrame()
	inputState.SetKey(input.KeyEscape, false)
	inputState.EndFrame()
	inputState.SetKey(input.KeyEscape, true)
	if !menu.Update(ctx) {
		t.Fatal("escape key did not close menu")
	}
	if menu.IsOpen() {
		t.Fatal("menu did not close from escape key")
	}
}

func TestEscapeMenuDeathModeSurvivesEscapeToggle(t *testing.T) {
	manager := &escapeMenuTestUIManager{}
	inputState := input.NewState()
	menu := EscapeMenu{}
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}
	menu.OpenDeath(ctx)

	if !menu.IsOpen() || !menu.DeathMode() {
		t.Fatal("death menu did not open in death mode")
	}

	inputState.SetKey(input.KeyEscape, true)
	if !menu.Update(ctx) {
		t.Fatal("escape did not hide death menu")
	}
	if menu.IsOpen() || !menu.DeathMode() {
		t.Fatal("hiding death menu ended death mode")
	}

	inputState.EndFrame()
	inputState.SetKey(input.KeyEscape, false)
	inputState.EndFrame()
	inputState.SetKey(input.KeyEscape, true)
	if !menu.Update(ctx) {
		t.Fatal("escape did not reopen death menu")
	}
	if !menu.IsOpen() || !menu.DeathMode() {
		t.Fatal("death menu did not reopen in death mode")
	}
}

func TestEscapeMenuDeathModeDoesNotBlockPointerOutsideWindow(t *testing.T) {
	inputState := input.NewState()
	inputState.SetMousePosition(799, 599)
	menu := EscapeMenu{}
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	menu.OpenDeath(ctx)

	if menu.Update(ctx) {
		t.Fatal("death menu consumed pointer input outside its window")
	}
}

func TestEscapeMenuResetDeathRestoresRegularMode(t *testing.T) {
	menu := EscapeMenu{}
	ctx := client.Context{Input: input.NewState(), ScreenW: 800, ScreenH: 600}
	menu.OpenDeath(ctx)

	menu.ResetDeath(ctx)

	if menu.IsOpen() || menu.DeathMode() {
		t.Fatal("death reset left the menu open or in death mode")
	}
}

func TestEscapeMenuCharacterSelectAckRequestsModeSwitch(t *testing.T) {
	menu := EscapeMenu{pending: true, pendingAction: EscapeMenuActionCharacterSelect}

	if !menu.ApplyRestartAck(network.RestartAck{Allowed: true}) {
		t.Fatal("allowed restart ack should request character-select transition")
	}
}

func TestEscapeMenuCharacterSelectAckDeniedKeepsMenuOpen(t *testing.T) {
	menu := EscapeMenu{}
	menu.Toggle(client.Context{ScreenW: 800, ScreenH: 600})
	menu.pending = true
	menu.pendingAction = EscapeMenuActionCharacterSelect

	if menu.ApplyRestartAck(network.RestartAck{Allowed: false}) {
		t.Fatal("denied restart ack should not request transition")
	}
	if !menu.Window.IsOpen() || menu.pending {
		t.Fatalf("menu = %+v, want open without pending request", menu)
	}
}

func TestEscapeMenuCharacterSelectWithoutNetworkShowsError(t *testing.T) {
	menu := EscapeMenu{}
	menu.RequestCharacterSelect(client.Context{})

	if menu.pending {
		t.Fatal("menu stayed pending without a network connection")
	}
}

func TestEscapeMenuQuitWithoutNetworkRequestsQuit(t *testing.T) {
	quit := false
	menu := EscapeMenu{}
	menu.Toggle(client.Context{ScreenW: 800, ScreenH: 600})
	menu.RequestQuitGame(client.Context{
		ScreenW:     800,
		ScreenH:     600,
		RequestQuit: func() { quit = true },
	})

	if !quit {
		t.Fatal("quit callback was not called")
	}
	if menu.pending {
		t.Fatal("menu stayed pending without a network connection")
	}
}

func TestEscapeMenuQuitAckRefusedKeepsMenuOpen(t *testing.T) {
	menu := EscapeMenu{pending: true, pendingAction: EscapeMenuActionExit}
	menu.Toggle(client.Context{ScreenW: 800, ScreenH: 600})
	menu.pending = true
	menu.pendingAction = EscapeMenuActionExit

	if !menu.ApplyQuitGameAck(network.QuitGameAck{Allowed: false, Result: 1}) {
		t.Fatal("quit ack was not handled")
	}
	if !menu.Window.IsOpen() || menu.pending {
		t.Fatalf("menu = %+v, want open without pending request", menu)
	}
}

func TestEscapeMenuRefusalDoesNotReopenCanceledMenu(t *testing.T) {
	menu := EscapeMenu{}
	menu.Toggle(client.Context{ScreenW: 800, ScreenH: 600})
	menu.pending = true
	menu.pendingAction = EscapeMenuActionExit
	menu.Window.Close()

	if !menu.ApplyQuitGameAck(network.QuitGameAck{Allowed: false, Result: 1}) {
		t.Fatal("quit ack was not handled")
	}
	if menu.Window.IsOpen() {
		t.Fatal("refused quit request reopened the canceled menu")
	}
}

func TestEscapeMenuPendingOnlyDisablesServerActions(t *testing.T) {
	menu := EscapeMenu{pending: true, pendingAction: EscapeMenuActionExit}
	buttons := collectEscapeMenuButtons(menu.widgetTree(client.Context{}))
	if len(buttons) != 4 {
		t.Fatalf("buttons = %d, want 4", len(buttons))
	}

	wantFocusable := []bool{false, true, true, false}
	for i, want := range wantFocusable {
		if got := buttons[i].IsFocusable(); got != want {
			t.Fatalf("button %d focusable = %t, want %t", i, got, want)
		}
	}
}

func TestEscapeMenuDeathModeUsesDeathActions(t *testing.T) {
	menu := EscapeMenu{deathMode: true}
	buttons := collectEscapeMenuButtons(menu.widgetTree(client.Context{}))
	if len(buttons) != 4 {
		t.Fatalf("buttons = %d, want 4", len(buttons))
	}
}

func TestEscapeMenuDeathModeOffersTokenOfSiegfried(t *testing.T) {
	sessionState := &session.Session{Dead: true, Inventory: session.Inventory{Items: []session.InventoryItem{{ItemID: client.TokenOfSiegfriedItemID, Amount: 1}}}}
	world := worldstate.New()
	ctx := client.Context{Session: sessionState, World: world}
	menu := EscapeMenu{}
	menu.OpenDeath(ctx)

	buttons := collectEscapeMenuButtons(menu.widgetTree(ctx))
	if len(buttons) != 5 {
		t.Fatalf("death buttons = %d, want 5 with Token of Siegfried", len(buttons))
	}
	if menu.height != escapeMenuReviveHeight {
		t.Fatalf("death menu height = %d, want %d", menu.height, escapeMenuReviveHeight)
	}
}

func collectEscapeMenuButtons(root widget.Widget) []*button.Widget {
	if root == nil {
		return nil
	}
	buttons := make([]*button.Widget, 0, 4)
	var visit func(widget.Widget)
	visit = func(current widget.Widget) {
		if current == nil {
			return
		}
		if currentButton, ok := current.(*button.Widget); ok {
			buttons = append(buttons, currentButton)
		}
		if parent, ok := current.(interface{ Children() []widget.Widget }); ok {
			for _, child := range parent.Children() {
				visit(child)
			}
		}
	}
	visit(root)
	return buttons
}
