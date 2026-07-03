package ui

import (
	"testing"

	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
)

type escapeMenuTestUIManager struct {
	root widget.Widget
}

func (m *escapeMenuTestUIManager) SetRoot(root widget.Widget) {
	m.root = root
}

func (m *escapeMenuTestUIManager) Clear() {
	m.root = nil
}

func TestEscapeMenuOpenPublishesGogpuWindow(t *testing.T) {
	inputState := input.NewState()
	manager := &escapeMenuTestUIManager{}
	menu := EscapeMenu{}
	menu.OpenMenu()
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}

	if !menu.Update(ctx) {
		t.Fatal("escape menu did not consume update while open")
	}
	if manager.root == nil {
		t.Fatal("escape menu did not publish its gogpu/ui root")
	}
}

func TestEscapeMenuCharacterSelectAckRequestsModeSwitch(t *testing.T) {
	menu := EscapeMenu{open: true, pending: true}

	if !menu.ApplyRestartAck(network.RestartAck{Allowed: true}) {
		t.Fatal("allowed restart ack should request character-select transition")
	}
}

func TestEscapeMenuCharacterSelectAckDeniedKeepsMenuOpen(t *testing.T) {
	menu := EscapeMenu{open: true, pending: true}

	if menu.ApplyRestartAck(network.RestartAck{Allowed: false}) {
		t.Fatal("denied restart ack should not request transition")
	}
	if !menu.open || menu.pending {
		t.Fatalf("menu = %+v, want open without pending request", menu)
	}
}

func TestEscapeMenuCharacterSelectWithoutNetworkShowsError(t *testing.T) {
	menu := EscapeMenu{open: true}
	menu.RequestCharacterSelect(client.Context{})

	if menu.pending {
		t.Fatal("menu stayed pending without a network connection")
	}
}
