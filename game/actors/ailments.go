package actors

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/world"
)

type AilmentVisual struct {
	BodyStarted     time.Time
	SilenceStarted  time.Time
	CurseStarted    time.Time
	IceBreakStarted time.Time
	IceBreakOrigin  scene.Point3
}

// Actor snapshots restore persistent visuals; live state changes also play
// transition sounds. Repeated snapshots must not restart either one.
func (m *System) SyncActorAilments(ctx client.Context, id uint32, previous world.Actor, changedNow bool) {
	actor, ok, local := scene.ActorForCombatID(ctx, id)
	if !ok {
		return
	}
	if local {
		id = scene.LocalSkillTarget(ctx)
		m.SetActorStateEffect(ctx, id, effects.Blind, actor.HealthState&db.HealthStateBlind != 0)
	}
	visual := m.Ailments[id]
	if previous.BodyState != actor.BodyState || visual.BodyStarted.IsZero() {
		visual.BodyStarted = time.Time{}
		if actor.BodyState != 0 {
			visual.BodyStarted = time.Now()
		}
	}
	if actor.HealthState&db.HealthStateSilence == 0 {
		visual.SilenceStarted = time.Time{}
	} else if previous.HealthState&db.HealthStateSilence == 0 || visual.SilenceStarted.IsZero() {
		visual.SilenceStarted = time.Now()
	}
	if actor.HealthState&db.HealthStateCurse == 0 {
		visual.CurseStarted = time.Time{}
	} else if previous.HealthState&db.HealthStateCurse == 0 || visual.CurseStarted.IsZero() {
		visual.CurseStarted = time.Now()
	}
	if actor.BodyState == db.BodyStateFreeze {
		visual.IceBreakStarted = time.Time{}
		visual.IceBreakOrigin = scene.Point3{}
	} else if changedNow && previous.BodyState == db.BodyStateFreeze {
		visual.IceBreakStarted = time.Now()
		x, y := scene.ActorRenderPosition(actor, visual.IceBreakStarted)
		visual.IceBreakOrigin = scene.Point3{X: scene.CellCenter(x), Y: scene.TerrainHeightAt(ctx.World, x, y), Z: scene.CellCenter(y)}
	}
	if visual == (AilmentVisual{}) {
		delete(m.Ailments, id)
	} else {
		if m.Ailments == nil {
			m.Ailments = make(map[uint32]AilmentVisual)
		}
		m.Ailments[id] = visual
	}
	if changedNow && previous.BodyState != actor.BodyState && actor.BodyState == db.BodyStateStun {
		m.ailmentSound(id, "_stun.wav")
	}
	if changedNow && previous.BodyState != actor.BodyState {
		if actor.BodyState == db.BodyStateFreeze {
			m.ailmentSound(id, "_stonecurse.wav")
		}
		if previous.BodyState == db.BodyStateFreeze {
			m.ailmentSound(id, "_frozen_explosion.wav")
		}
		if previous.BodyState == db.BodyStateStone {
			m.ailmentSound(id, "_stone_explosion.wav")
		}
	}
	if changedNow && actor.HealthState&db.HealthStateBlind != 0 && previous.HealthState&db.HealthStateBlind == 0 {
		m.ailmentSound(id, "_blind.wav")
	}
	if changedNow && actor.HealthState&db.HealthStateSilence != 0 && previous.HealthState&db.HealthStateSilence == 0 {
		m.ailmentSound(id, "_silence.wav")
	}
	if changedNow && actor.HealthState&db.HealthStateCurse != 0 && previous.HealthState&db.HealthStateCurse == 0 {
		m.ailmentSound(id, "_curse.wav")
	}
	if changedNow && actor.HealthState&^previous.HealthState&(db.HealthStatePoison|db.HealthStateDpoison) != 0 {
		m.ailmentSound(id, "_poison.wav")
	}
	if changedNow && actor.HealthState&^previous.HealthState&db.HealthStateBleeding != 0 {
		m.ailmentSound(id, "_poison.wav")
	}
}

type ailmentSprite struct {
	file            string
	action          int
	started         time.Time
	height, alpha   float64
	repeat, overlay bool
	delay           time.Duration
}

func (m *System) drawActorAilments(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry, body *scene.Billboard, visibility float64, now time.Time) {
	if m.bindings.Effects == nil || visibility <= 0 {
		return
	}
	id := entry.Actor.ID
	if entry.IsPlayer {
		id = scene.LocalSkillTarget(ctx)
	}
	visual := m.Ailments[id]
	height := 3.0 // The original uses a 90-pixel head height for player characters.
	if body != nil && body.Image != nil && !res.HasPlayerJobToken(scene.ActorVisualJob(entry.Actor)) {
		height = float64(body.Image.Bounds().Dy()) / 30
	}
	file := ""
	switch entry.Actor.BodyState {
	case db.BodyStateStun:
		file = "Status-stun"
	case db.BodyStateSleep:
		file = "Status-sleep"
	}
	if file != "" {
		m.drawAilmentSprite(screen, ctx, projection, entry, ailmentSprite{file: file, started: visual.BodyStarted, height: height, alpha: visibility, repeat: true, overlay: true}, now)
	}
	if entry.Actor.HealthState&db.HealthStateSilence != 0 {
		m.drawAilmentSprite(screen, ctx, projection, entry, ailmentSprite{file: "emotion", action: 10, started: visual.SilenceStarted, height: height, alpha: visibility, repeat: true, overlay: true}, now)
	}
	if entry.Actor.HealthState&db.HealthStateCurse != 0 {
		direction := sprites.DirectionForCamera(entry.Actor.Dir, projection.CameraYaw)
		m.drawAilmentSprite(screen, ctx, projection, entry, ailmentSprite{file: "Status-curse", action: direction, started: visual.CurseStarted, height: 3, alpha: visibility * 127 / 255, repeat: true}, now)
	}
	if entry.Actor.BodyState == db.BodyStateFreeze {
		m.drawAilmentSprite(screen, ctx, projection, entry, ailmentSprite{file: "얼음땡", started: visual.BodyStarted, alpha: visibility, repeat: true}, now)
	} else if !visual.IceBreakStarted.IsZero() {
		entry.WorldX, entry.WorldY, entry.WorldZ = visual.IceBreakOrigin.X, visual.IceBreakOrigin.Z, visual.IceBreakOrigin.Y
		if !m.drawAilmentSprite(screen, ctx, projection, entry, ailmentSprite{file: "얼음땡", action: 1, started: visual.IceBreakStarted, alpha: visibility, delay: time.Second / 60}, now) {
			visual.IceBreakStarted = time.Time{}
			visual.IceBreakOrigin = scene.Point3{}
			if visual == (AilmentVisual{}) {
				delete(m.Ailments, id)
			} else {
				m.Ailments[id] = visual
			}
		}
	}
}

func (m *System) drawAilmentSprite(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry, sprite ailmentSprite, now time.Time) bool {
	view := m.bindings.Effects.EffectSpriteView(ctx.Resources, sprite.file)
	if view == nil || view.Act == nil || sprite.action < 0 || len(view.Act.Actions) == 0 {
		return false
	}
	sprite.action %= len(view.Act.Actions)
	action := view.Act.Actions[sprite.action]
	if len(action.Animations) == 0 {
		return false
	}
	if sprite.started.IsZero() {
		sprite.started = view.Started
	}
	delay := float64(action.DelayMS)
	if sprite.delay > 0 {
		delay = float64(sprite.delay) / float64(time.Millisecond)
	}
	if delay <= 0 {
		delay = 150
	}
	if !sprite.repeat && now.Sub(sprite.started) >= time.Duration(float64(len(action.Animations))*delay*float64(time.Millisecond)) {
		return false
	}
	motion := scene.SpriteMotionIndexWithDelay(action, sprite.started, now, sprite.repeat, delay)
	if motion < 0 || motion >= len(action.Animations) {
		return false
	}
	key := scene.SpriteBillboardKey{ActionIndex: sprite.action, Motion: motion}
	billboard, ok := view.Billboards[key]
	if !ok {
		billboard, ok = scene.ComposeSingleSpriteBillboard(view, action.Animations[motion])
		if !ok {
			return false
		}
		view.Billboards[key] = billboard
	}
	opts := scene.SpriteBillboardTriangleDrawOptions()
	opts.DepthTest = !sprite.overlay
	opts.DepthWrite = false
	scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, entry.WorldX, entry.WorldY, entry.WorldZ+sprite.height, scene.PixelRatio, 0, sprite.alpha, 1, color.RGBA{255, 255, 255, 255}, opts)
	return true
}

func (m *System) ailmentSound(id uint32, path string) {
	if m.bindings.ScheduleSoundAtActor != nil {
		m.bindings.ScheduleSoundAtActor(time.Now(), id, path)
	}
}
