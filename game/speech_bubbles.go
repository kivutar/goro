package game

import (
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

const (
	speechBubbleBaseDuration = 3500 * time.Millisecond
	speechBubblePerRune      = 40 * time.Millisecond
	speechBubbleMaxDuration  = 8500 * time.Millisecond
	speechBubbleMaxWidth     = 220
)

type speechBubble struct {
	text    string
	expires time.Time
}

func (m *WorldMode) applySpeechBubble(ctx client.Context, chat network.ChatMessage, now time.Time) {
	if strings.TrimSpace(chat.Text) == "" {
		return
	}
	text := speechBubbleText(chat.Text)
	if text == "" {
		return
	}
	actorIDs := speechBubbleActorIDs(ctx, chat)
	if len(actorIDs) == 0 {
		return
	}
	if m.speechBubbles == nil {
		m.speechBubbles = make(map[uint32]speechBubble)
	}
	duration := speechBubbleBaseDuration + time.Duration(len([]rune(text)))*speechBubblePerRune
	if duration > speechBubbleMaxDuration {
		duration = speechBubbleMaxDuration
	}
	for _, actorID := range actorIDs {
		if actorID == 0 {
			continue
		}
		m.speechBubbles[actorID] = speechBubble{
			text:    text,
			expires: now.Add(duration),
		}
	}
}

func speechBubbleActorIDs(ctx client.Context, chat network.ChatMessage) []uint32 {
	if chat.GID != 0 {
		ids := []uint32{chat.GID}
		if ctx.Session != nil {
			if chat.GID == ctx.Session.AccountID && ctx.Session.CharID != 0 {
				ids = append(ids, ctx.Session.CharID)
			} else if chat.GID == ctx.Session.CharID && ctx.Session.AccountID != 0 {
				ids = append(ids, ctx.Session.AccountID)
			}
		}
		return ids
	}
	name, _, ok := strings.Cut(strings.TrimSpace(chat.Text), " : ")
	if !ok || ctx.Session == nil {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(name), selectedCharacterName(ctx.Session)) {
		return nil
	}
	ids := make([]uint32, 0, 2)
	if ctx.Session.CharID != 0 {
		ids = append(ids, ctx.Session.CharID)
	}
	if ctx.Session.AccountID != 0 {
		ids = append(ids, ctx.Session.AccountID)
	}
	return ids
}

func speechBubbleText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if _, message, ok := strings.Cut(text, " : "); ok {
		return strings.TrimSpace(message)
	}
	return text
}

const actorSpeechBubbleGap = 6.0

func actorSpeechBubbleBottomY(baseY, scale float64) float64 {
	return actorSpriteTopY(baseY, scale) - actorSpeechBubbleGap
}

func (m *WorldMode) drawSpeechBubbles(screen *render.Frame, entries []sceneActorDrawEntry, now time.Time) {
	if len(m.speechBubbles) == 0 {
		return
	}
	for id, bubble := range m.speechBubbles {
		if now.After(bubble.expires) {
			delete(m.speechBubbles, id)
		}
	}
	if len(m.speechBubbles) == 0 {
		return
	}
	for _, entry := range entries {
		bubble, ok := m.speechBubbles[entry.actor.ID]
		if !ok || strings.TrimSpace(bubble.text) == "" {
			continue
		}
		bottomY := actorSpeechBubbleBottomY(entry.screenY, entry.scale)
		render.DrawUISpeechBubble(screen, bubble.text, entry.screenX, bottomY, speechBubbleMaxWidth)
	}
}
