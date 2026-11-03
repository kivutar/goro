package effects

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

type ActorBlur struct {
	billboard *scene.Billboard
	born      time.Time
	x, y, z   float64
	scale     float64
	tint      color.RGBA
}

func (m *System) DrawActorBlurs(screen *render.Frame, projection scene.Projection, entry scene.ActorDrawEntry, billboard *scene.Billboard, scale, visibility, shadow float64, tint color.RGBA, now time.Time) {
	id := entry.Actor.ID
	if scene.ActorHasStealth(entry.Actor) {
		delete(m.ActorBlurs, id)
		return
	}
	blurs := m.ActorBlurs[id]
	live := blurs[:0]
	for _, blur := range blurs {
		if now.Sub(blur.born) < 23*db.EffectFrameDuration {
			live = append(live, blur)
		}
	}
	clear(blurs[len(live):])
	blurs = live
	// MakeBlur emits a frozen CBlurPC every five ticks, except on dyed bodies.
	active := (entry.IsPlayer || scene.ActorRepresentsPlayer(entry.Actor)) && entry.Actor.BodyPal == 0 && entry.Actor.Opt3State&(db.Opt3Quicken|db.Opt3Overthrust) != 0
	interval := 5 * db.EffectFrameDuration
	if !active && (entry.IsPlayer || scene.ActorRepresentsPlayer(entry.Actor)) && entry.Actor.Opt3State&db.Opt3Bunsin != 0 {
		active, interval = true, 20*db.EffectFrameDuration
	}
	if active && (len(blurs) == 0 || now.Sub(blurs[len(blurs)-1].born) >= interval) {
		blurs = append(blurs, ActorBlur{billboard: billboard, born: now, x: entry.WorldX, y: entry.WorldY, z: entry.WorldZ,
			scale: scale / scene.ActorBillboardScreenScale(projection, entry.WorldX, entry.WorldY, entry.WorldZ), tint: tint})
	}
	if len(blurs) == 0 {
		delete(m.ActorBlurs, id)
		return
	}
	if m.ActorBlurs == nil {
		m.ActorBlurs = make(map[uint32][]ActorBlur)
	}
	m.ActorBlurs[id] = blurs
	for _, blur := range blurs {
		alpha := float64(max(0, 180-8*int(now.Sub(blur.born)/db.EffectFrameDuration))) / 255
		m.drawActorBlur(screen, projection, blur, blur.billboard, visibility*alpha, shadow)
	}
}

func (m *System) drawActorBlur(screen *render.Frame, projection scene.Projection, blur ActorBlur, billboard *scene.Billboard, alpha, shadow float64) {
	if len(billboard.Parts) > 0 {
		for _, part := range billboard.Parts {
			m.drawActorBlur(screen, projection, blur, part, alpha, shadow)
		}
		return
	}
	options := scene.SpriteBillboardTriangleDrawOptions()
	options.Filter = render.FilterSprite
	scale := blur.scale * scene.ActorBillboardScreenScale(projection, blur.x, blur.y, blur.z)
	scene.DrawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, m.spriteTexture(billboard.Image), blur.x, blur.y, scene.ActorSpriteWorldZ(blur.z), scale, alpha, shadow, blur.tint, options)
}
