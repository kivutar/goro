package ui

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestDeathModalOpenAndClearWhenAlive(t *testing.T) {
	var modal DeathModal
	modal.OpenDeath()
	if !modal.open {
		t.Fatal("expected modal open")
	}
	if modal.pending != DeathModalActionNone {
		t.Fatalf("pending = %d, want none", modal.pending)
	}

	ctx := client.Context{Session: &session.Session{}}
	ctx.Session.Vitals.HP = 0
	ctx.Session.Selected.HP = 0
	modal.ClearIfAlive(ctx)
	if !modal.open {
		t.Fatal("modal cleared while character is still dead")
	}

	ctx.Session.Vitals.HP = 1
	modal.ClearIfAlive(ctx)
	if modal.open {
		t.Fatal("modal stayed open after positive HP")
	}
}

func TestDeathModalCharacterSelectAckRequestsModeSwitch(t *testing.T) {
	var modal DeathModal
	modal.OpenDeath()
	modal.pending = DeathModalActionCharSelect

	if !modal.ApplyRestartAck(network.RestartAck{Allowed: true}) {
		t.Fatal("allowed restart ack should request character-select transition")
	}
}

func TestDeathModalCharacterSelectAckDeniedKeepsModalOpen(t *testing.T) {
	var modal DeathModal
	modal.OpenDeath()
	modal.pending = DeathModalActionCharSelect

	if modal.ApplyRestartAck(network.RestartAck{Allowed: false}) {
		t.Fatal("denied restart ack should not request transition")
	}
	if !modal.open || modal.pending != DeathModalActionNone {
		t.Fatalf("modal = %+v, want open and no pending action", modal)
	}
}
