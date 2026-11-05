package actors

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

func (m *System) StarFusionHeight(actor worldstate.Actor, now time.Time) float64 {
	active := false
	job := scene.ActorVisualJob(actor)
	if job == db.JobStar2 || job == db.JobStar2B {
		anim, ok := m.ActorAnimation(actor.ID, now)
		active = ok && anim.ActionFamily != scene.SpriteActionIdle && anim.ActionFamily != scene.SpriteActionWalk
	}
	return m.bindings.Effects.FusionHeight(actor, now, active)
}

// Body effects reuse the composed sprite, including its appearance and scale.
// They never mutate the cached image or the actor's authoritative position.
func (m *System) DrawActorSpriteWithEffects3D(ctx client.Context, screen *render.Frame, projection scene.Projection, entry scene.ActorDrawEntry, billboard *scene.Billboard, scale, visibility, shadow float64, tint color.RGBA, blend render.Blend, now time.Time) {
	m.bindings.Effects.DrawActor(ctx, screen, projection, entry, billboard, scale, visibility, shadow, tint, blend, now, m.StarFusionHeight(entry.Actor, now))
	m.drawActorAilments(screen, ctx, projection, entry, billboard, visibility, now)
}
