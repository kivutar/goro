package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

func (m *WorldMode) applyEmotionNotify(ctx client.Context, notify network.EmotionNotify) {
	frame, ok := db.EmotionSpriteFrame(notify.Type)
	if !ok {
		return
	}
	now := time.Now()
	duration := m.emotionDuration(ctx, frame)
	m.removeWorldEffect(effectEmotion, notify.GID)
	x, y, ok := effectAnchor(ctx, notify.GID)
	if !ok {
		return
	}
	m.worldEffects = append(m.worldEffects, worldEffect{
		effectID:            effectEmotion,
		actorID:             notify.GID,
		x:                   x,
		y:                   y,
		starts:              now,
		expires:             now.Add(duration),
		duration:            duration,
		spriteFrameOverride: frame,
		hasSpriteFrame:      true,
	})
	glog.Debugf("emotion actor=%d type=%d frame=%d", notify.GID, notify.Type, frame)
}

func (m *WorldMode) emotionDuration(ctx client.Context, frame int) time.Duration {
	view := m.effectSpriteView(ctx.Resources, "emotion")
	if view == nil || frame < 0 || frame >= len(view.act.Actions) {
		return time.Second
	}
	action := view.act.Actions[frame]
	delayMS := action.DelayMS
	if delayMS <= 0 {
		delayMS = 100
	}
	frames := len(action.Animations)
	if frames <= 0 {
		frames = 1
	}
	return time.Duration(float64(delayMS)*float64(frames)) * time.Millisecond
}
