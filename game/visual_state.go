package game

import (
	"slices"
	"time"
)

// Expiration must also run when updates are not followed by a draw.
func (m *WorldMode) pruneVisualState(now time.Time) {
	m.damageFloaters = slices.DeleteFunc(m.damageFloaters, func(f damageFloater) bool {
		return now.After(f.expires)
	})
	m.worldEffects = slices.DeleteFunc(m.worldEffects, func(e worldEffect) bool {
		return now.After(e.expires)
	})
	for id, bubble := range m.speechBubbles {
		if now.After(bubble.expires) {
			delete(m.speechBubbles, id)
		}
	}
	for id, bar := range m.actorCastBars {
		if _, active := actorCastBarProgress(bar, now); !active {
			delete(m.actorCastBars, id)
		}
	}
}
