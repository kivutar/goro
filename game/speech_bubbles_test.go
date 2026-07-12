package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestSpeechBubbleTextUsesMessagePart(t *testing.T) {
	if got := speechBubbleText("Kivutar : hello there"); got != "hello there" {
		t.Fatalf("speechBubbleText = %q, want hello there", got)
	}
}

func TestApplySpeechBubbleUsesLocalActorIDsForLocalEcho(t *testing.T) {
	mode := NewWorldMode()
	ctx := client.Context{Session: &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Name: "Kivutar"},
	}}
	mode.applySpeechBubble(ctx, network.ChatMessage{Text: "Kivutar : hello"}, time.Now())
	if mode.speechBubbles[150000].text != "hello" {
		t.Fatalf("char bubble = %+v, want hello", mode.speechBubbles[150000])
	}
	if mode.speechBubbles[2000000].text != "hello" {
		t.Fatalf("account bubble = %+v, want hello", mode.speechBubbles[2000000])
	}
}
