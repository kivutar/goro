package game

import (
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

type scheduledSound struct {
	at     time.Time
	paths  []string
	volume float64
}

type actorSoundFrame struct {
	actionFamily int
	motion       int
	soundIndex   int
}

func (m *WorldMode) scheduleSound(at time.Time, paths ...string) {
	m.scheduleSoundVolume(at, 1, paths...)
}

func (m *WorldMode) scheduleSoundVolume(at time.Time, volume float64, paths ...string) {
	clean := paths[:0]
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			clean = append(clean, path)
		}
	}
	if len(clean) == 0 {
		return
	}
	m.scheduledSounds = append(m.scheduledSounds, scheduledSound{
		at:     at,
		paths:  append([]string(nil), clean...),
		volume: volume,
	})
}

func (m *WorldMode) playDueScheduledSounds(ctx client.Context, now time.Time) {
	if len(m.scheduledSounds) == 0 {
		return
	}
	active := m.scheduledSounds[:0]
	for _, sound := range m.scheduledSounds {
		if now.Before(sound.at) {
			active = append(active, sound)
			continue
		}
		m.playSFXFirstVolume(ctx, sound.volume, sound.paths...)
	}
	m.scheduledSounds = active
}

func (m *WorldMode) processMapSounds(ctx client.Context, now time.Time) {
	if ctx.World == nil || ctx.World.RSW == nil || ctx.World.GND == nil || len(ctx.World.RSW.Sounds) == 0 {
		return
	}
	playerX, playerY := actorRenderPosition(ctx.World.Player, now)
	width := float64(ctx.World.GND.Width)
	height := float64(ctx.World.GND.Height)
	if m.mapSoundNext == nil {
		m.mapSoundNext = make(map[int]time.Time)
	}
	for index, sound := range ctx.World.RSW.Sounds {
		if strings.TrimSpace(sound.File) == "" {
			continue
		}
		if sound.Volume <= 0 {
			continue
		}
		if next := m.mapSoundNext[index]; !next.IsZero() && now.Before(next) {
			continue
		}
		soundX := float64(sound.Position.X) + width
		soundY := float64(sound.Position.Z) + height
		maxDistance := float64(sound.Range)*0.2 + float64(sound.Height)
		if math.Hypot(soundX-playerX, soundY-playerY) > maxDistance {
			continue
		}
		m.scheduleSoundVolume(now, float64(sound.Volume), sound.File)
		delay := time.Duration(float64(time.Second) * float64(sound.Cycle))
		if delay < 100*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		m.mapSoundNext[index] = now.Add(delay)
	}
}

func (m *WorldMode) processActorMotionSounds(ctx client.Context, now time.Time) {
	if ctx.World == nil || len(ctx.World.Actors) == 0 {
		return
	}
	for _, actor := range ctx.World.Actors {
		m.processNonPCMotionSound(ctx, actor, now)
	}
}

func (m *WorldMode) processNonPCMotionSound(ctx client.Context, actor worldstate.Actor, now time.Time) {
	if actor.ID == 0 || res.HasPlayerJobToken(int(actor.Job)) || !actorWithinSoundRange(ctx, actor, now) {
		return
	}
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil || view.act == nil {
		return
	}
	state := m.nonPCSpriteState(actor, now)
	switch state.actionFamily {
	case spriteActionNonPCAttack, spriteActionNonPCHurt:
		return
	}
	_, action, ok := resolveSpriteAction(view.act, state.actionFamily, state.direction)
	if !ok || len(action.Animations) == 0 {
		return
	}
	motion := bodyMotionForState(action, state, view.started, now)
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	soundIndex := action.Animations[motion].Sound
	current := actorSoundFrame{actionFamily: state.actionFamily, motion: motion, soundIndex: soundIndex}
	if m.actorSoundFrames == nil {
		m.actorSoundFrames = make(map[uint32]actorSoundFrame)
	}
	if previous, ok := m.actorSoundFrames[actor.ID]; ok && previous == current {
		return
	}
	m.actorSoundFrames[actor.ID] = current
	if soundIndex < 0 {
		return
	}
	if sound := actionSoundName(view.act, action, motion); sound != "" {
		m.scheduleSound(now, sound)
	}
}

func actorWithinSoundRange(ctx client.Context, actor worldstate.Actor, now time.Time) bool {
	if ctx.World == nil {
		return false
	}
	actorX, actorY := actorRenderPosition(actor, now)
	playerX, playerY := actorRenderPosition(ctx.World.Player, now)
	const soundRangeCells = 25
	return math.Hypot(actorX-playerX, actorY-playerY) <= soundRangeCells
}

func (m *WorldMode) playSFXFirstVolume(ctx client.Context, volume float64, paths ...string) {
	if ctx.Audio == nil {
		return
	}
	var lastErr error
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		source, err := ctx.Audio.PlaySFXVolume(path, volume)
		if err == nil {
			if source != "" {
				glog.Debugf("sfx playing path=%s source=%s", path, source)
			}
			return
		}
		lastErr = err
	}
	if lastErr != nil {
		glog.Warnf("sfx failed paths=%v: %v", paths, lastErr)
	}
}

func actionSoundName(act *res.ACT, action res.ACTAction, motion int) string {
	if act == nil || motion < 0 || motion >= len(action.Animations) {
		return ""
	}
	soundIndex := action.Animations[motion].Sound
	if soundIndex < 0 || soundIndex >= len(act.Sounds) {
		return ""
	}
	sound := strings.TrimSpace(act.Sounds[soundIndex])
	if strings.EqualFold(sound, "atk") {
		return ""
	}
	return sound
}

func actionSFXCandidatesForActor(actor worldstate.Actor, act *res.ACT, action res.ACTAction, motion int) []string {
	if act == nil || motion < 0 || motion >= len(action.Animations) {
		return nil
	}
	soundIndex := action.Animations[motion].Sound
	if soundIndex < 0 || soundIndex >= len(act.Sounds) {
		return nil
	}
	sound := strings.TrimSpace(act.Sounds[soundIndex])
	if strings.EqualFold(sound, "atk") {
		return weaponAttackSFXCandidates(actor)
	}
	if sound == "" {
		return nil
	}
	return []string{sound}
}

func weaponAttackSFXCandidates(actor worldstate.Actor) []string {
	if !actorUsesWeaponSounds(actor) {
		return nil
	}
	return db.WeaponAttackSounds(db.PlayerWeaponType(actorWeaponForSounds(actor)))
}

func combatHitSFXCandidates(source worldstate.Actor, sourceOK bool, target worldstate.Actor, targetOK bool) []string {
	if targetOK && res.HasPlayerJobToken(int(target.Job)) {
		return db.JobHitSounds(int(target.Job))
	}
	if sourceOK && actorUsesWeaponSounds(source) {
		return db.WeaponHitSounds(db.PlayerWeaponType(actorWeaponForSounds(source)))
	}
	return nil
}

func actorUsesWeaponSounds(actor worldstate.Actor) bool {
	return res.HasPlayerJobToken(int(actor.Job)) || actorIsMercenary(actor)
}

func actorWeaponForSounds(actor worldstate.Actor) int {
	if actorIsMercenary(actor) {
		return mercenaryWeaponForAppearance(int(actor.Job), int(actor.Weapon))
	}
	return int(actor.Weapon)
}
