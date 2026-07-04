package ui

import (
	"testing"

	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
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
	menu.OpenMenu()
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}

	if !menu.Update(ctx) {
		t.Fatal("escape menu did not consume update while open")
	}
	if len(manager.overlays) != 1 {
		t.Fatalf("escape menu overlays = %d, want 1", len(manager.overlays))
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
