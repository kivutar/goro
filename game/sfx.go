package game

import (
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

type scheduledSound struct {
	at          time.Time
	paths       []string
	volume      float64
	positioned  bool
	actorID     uint32
	hasPosition bool
	x           float64
	y           float64
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
	m.scheduleSoundOptions(scheduledSound{at: at, volume: volume}, paths...)
}

func (m *WorldMode) scheduleSoundAtActor(at time.Time, actorID uint32, paths ...string) {
	m.scheduleSoundVolumeAtActor(at, 1, actorID, paths...)
}

func (m *WorldMode) scheduleSoundVolumeAtActor(at time.Time, volume float64, actorID uint32, paths ...string) {
	m.scheduleSoundOptions(scheduledSound{at: at, volume: volume, positioned: true, actorID: actorID}, paths...)
}

func (m *WorldMode) scheduleSoundVolumeAtPosition(at time.Time, volume float64, x, y float64, paths ...string) {
	m.scheduleSoundOptions(scheduledSound{at: at, volume: volume, positioned: true, hasPosition: true, x: x, y: y}, paths...)
}

func (m *WorldMode) scheduleSoundAtWorldEffect(at time.Time, effect worldEffect, paths ...string) {
	sound := scheduledSound{
		at:          at,
		volume:      1,
		positioned:  true,
		actorID:     effect.actorID,
		hasPosition: true,
		x:           float64(effect.x),
		y:           float64(effect.y),
	}
	m.scheduleSoundOptions(sound, paths...)
}

func (m *WorldMode) scheduleSoundOptions(sound scheduledSound, paths ...string) {
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
	sound.paths = append([]string(nil), clean...)
	m.scheduledSounds = append(m.scheduledSounds, sound)
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
		m.playScheduledSound(ctx, sound, now)
	}
	m.scheduledSounds = active
}

func (m *WorldMode) playScheduledSound(ctx client.Context, sound scheduledSound, now time.Time) {
	volume := sound.volume
	if sound.positioned {
		x, y, ok := scheduledSoundPosition(ctx, sound, now)
		if !ok {
			return
		}
		volume *= spatialSoundGain(ctx, x, y, now)
	}
	m.playSFXFirstVolume(ctx, volume, sound.paths...)
}

func (m *WorldMode) preloadMapSounds(ctx client.Context) {
	if ctx.Audio == nil || ctx.World == nil || ctx.World.RSW == nil {
		return
	}
	for _, sound := range ctx.World.RSW.Sounds {
		if sound.Volume <= 0 {
			continue
		}
		if err := ctx.Audio.PreloadSFX(sound.File); err != nil {
			glog.Debugf("preload map sound %s: %v", sound.File, err)
		}
	}
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
		m.scheduleSoundVolumeAtPosition(now, float64(sound.Volume), soundX, soundY, sound.File)
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
		m.scheduleSoundAtActor(now, actor.ID, sound)
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
				glog.Debugf("sfx playing path=%s source=%s volume=%.2f", path, source, volume)
			}
			return
		}
		lastErr = err
	}
	if lastErr != nil {
		glog.Warnf("sfx failed paths=%v: %v", paths, lastErr)
	}
}

const (
	// Reference clients use 40/250 world-unit distances; Goro keeps
	// actor and RSW sound positions in cells, so those become 4/25 cells.
	referenceSoundMinDistanceCells = 4.0
	referenceSoundMaxDistanceCells = 25.0
)

func scheduledSoundPosition(ctx client.Context, sound scheduledSound, now time.Time) (float64, float64, bool) {
	if ctx.World == nil {
		return 0, 0, false
	}
	if sound.actorID != 0 {
		if actor, ok := ctx.World.Actors[sound.actorID]; ok {
			x, y := actorRenderPosition(actor, now)
			return x, y, true
		}
		if isLocalActor(ctx, sound.actorID) {
			x, y := actorRenderPosition(ctx.World.Player, now)
			return x, y, true
		}
		if sound.hasPosition {
			return sound.x, sound.y, true
		}
		return 0, 0, false
	}
	if !sound.hasPosition {
		return 0, 0, false
	}
	return sound.x, sound.y, true
}

func spatialSoundGain(ctx client.Context, sourceX, sourceY float64, now time.Time) float64 {
	if ctx.World == nil {
		return 1
	}
	listenerX, listenerY := actorRenderPosition(ctx.World.Player, now)
	return referenceSpatialSoundGain(listenerX, listenerY, sourceX, sourceY)
}

func referenceSpatialSoundGain(listenerX, listenerY, sourceX, sourceY float64) float64 {
	dist := math.Hypot(sourceX-listenerX, sourceY-listenerY)
	if !isFinite(dist) {
		return 1
	}
	dist = clampFloat(dist, referenceSoundMinDistanceCells, referenceSoundMaxDistanceCells)
	return referenceSoundMinDistanceCells / dist
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

func actionAttackMarkerMotion(act *res.ACT, action res.ACTAction) int {
	if act != nil {
		for motion, animation := range action.Animations {
			if animation.Sound < 0 || animation.Sound >= len(act.Sounds) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(act.Sounds[animation.Sound]), "atk") {
				return motion
			}
		}
	}
	return firstActionSoundMotion(action)
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

func combatHitSFXCandidates(action network.ActorActionNotify, source worldstate.Actor, sourceOK bool, target worldstate.Actor, targetOK bool) []string {
	if action.SkillID != 0 {
		return rotatedSFXCandidates(db.EnemyHitNormalSounds(), combatHitSFXRoll(action))
	}
	if targetOK && res.HasPlayerJobToken(int(target.Job)) {
		return db.JobHitSounds(int(target.Job))
	}
	if sourceOK && actorUsesWeaponSounds(source) {
		return db.WeaponHitSounds(db.PlayerWeaponType(actorWeaponForSounds(source)))
	}
	return nil
}

func rotatedSFXCandidates(paths []string, roll uint32) []string {
	if len(paths) <= 1 {
		return paths
	}
	start := int(roll % uint32(len(paths)))
	out := make([]string, 0, len(paths))
	out = append(out, paths[start:]...)
	out = append(out, paths[:start]...)
	return out
}

func combatHitSFXRoll(action network.ActorActionNotify) uint32 {
	roll := action.SourceID*1103515245 + action.TargetID*2654435761
	roll ^= uint32(action.ServerTick) * 2246822519
	roll ^= uint32(action.SkillID)<<16 | uint32(action.SkillLevel)
	roll ^= uint32(action.Damage) + uint32(action.LeftDamage)*16777619
	roll ^= uint32(action.HitCount)<<8 | uint32(action.Action)
	roll ^= roll >> 16
	return roll
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
