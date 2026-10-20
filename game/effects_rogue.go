package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

// The native PP_INTIMIDATE and PP_TEIHIT1 rotate in XY, then XZ.
// Native Y points down; our Y points up. Distances here use native units.
func rogueParticlePoint(x, y, pitch, yaw float64) modelPoint3 {
	s, c := math.Sincos(pitch)
	rx, ry := c*x-s*y, s*x+c*y
	s, c = math.Sincos(yaw)
	return modelPoint3{x: c * rx / 5, y: -ry / 5, z: s * rx / 5}
}

func rogueBurstParticle(effect worldEffect, index, frame int) ([4]modelPoint3, color.RGBA) {
	salt := index * 5
	delay, rise, tint := 70, 15, color.RGBA{100, 150, 255, 0}
	if effect.effectID == effectRogueCoin {
		delay, rise, tint = 18, 25, color.RGBA{250, 250, 155, 0}
	}
	pitch := degreesToRadians(float64(hitEffectRandom(effect, salt+1, 360)))
	yaw := degreesToRadians(float64(hitEffectRandom(effect, salt+2, 360)))
	var corners [4]modelPoint3
	if effect.effectID == effectTeiHit3 {
		age := frame + 1 - 20 - hitEffectRandom(effect, salt, 10)
		alpha := 15*clampInt(age-10, 0, 10) - 5*max(0, age-20)
		if alpha <= 0 {
			return corners, tint
		}
		tint = color.RGBA{200, 200, 255, uint8(alpha)}
		distance := float64(-10 + age*2)
		for j, p := range [4][2]float64{{16, -1}, {16, 1}, {-16, 1}, {-16, -1}} {
			corners[j] = rogueParticlePoint(distance+p[0], p[1], pitch, yaw)
		}
		return corners, tint
	}
	age := frame + 1 - delay - hitEffectRandom(effect, salt, 30)
	alpha := rise*clampInt(age, 0, 10) - 7*max(0, age-25)
	if alpha <= 0 {
		return corners, tint
	}
	tint.A = uint8(alpha)
	center := rogueParticlePoint(12+1.5*float64(age), 0, pitch, yaw)
	roll := degreesToRadians(float64(hitEffectRandom(effect, salt+3, 360)))
	spin := degreesToRadians(float64(hitEffectRandom(effect, salt+4, 360) + 5*(frame+1)))
	for j := range corners {
		corners[j] = add3(center, rogueParticlePoint(3.5, 0, roll+float64(j)*math.Pi/2, spin))
	}
	return corners, tint
}

func (m *WorldMode) drawRogueBurst(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	count := 80
	if effect.effectID == effectRogueCoin {
		count = 120
	}
	vertices := make([]render.Vertex3D, 0, count*4)
	indices := make([]uint16, 0, count*6)
	for i := 0; i < count; i++ {
		corners, tint := rogueBurstParticle(effect, i, frame)
		if tint.A == 0 {
			continue
		}
		// RenderTeiRect maps its first edge to the bottom of the texture.
		for j := range corners {
			corners[j] = add3(corners[j], effect.actorOrigin)
		}
		appendSoulLinkerQuad(&vertices, &indices, texture,
			[4]modelPoint3{corners[3], corners[2], corners[1], corners[0]}, tint)
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(effect.effectID != effectTeiHit3, false))
	}
}

func initBackStabEffect(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	origin := destination
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		origin = effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, effect.starts)
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	effect.actorOrigin = destination
	effect.actorOrigin.y += 1.6
	effect.projectileDirection = sub3(destination, origin)
	effect.hasActorTransform = true
}

func backStabParticle(effect worldEffect, index, frame int) (position modelPoint3, size, angle float64, alpha int) {
	salt := index * 5
	age := frame + 1 - 40 + hitEffectRandom(effect, salt, 10)
	alpha = max(0, 15*clampInt(age-1, 0, 9)-3*max(0, age-10))
	if alpha == 0 {
		return
	}
	direction := math.Atan2(effect.projectileDirection.z, effect.projectileDirection.x)
	direction += degreesToRadians(float64(hitEffectRandom(effect, salt+1, 60) - 30))
	speed := (.75 + .01*float64(hitEffectRandom(effect, salt+2, 50))) / 5
	position = add3(effect.actorOrigin, modelPoint3{
		x: math.Cos(direction) * speed * float64(age),
		z: math.Sin(direction) * speed * float64(age),
		y: (1 - .01*float64(hitEffectRandom(effect, salt+3, 200))) * float64(age) / 5,
	})
	size = (.1 + .05*float64(hitEffectRandom(effect, salt+4, 16))) / 5 * math.Sqrt2
	angle = degreesToRadians(float64(-5*age + 45))
	return
}

func (m *WorldMode) drawBackStabEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for i := 0; i < 80; i++ {
		position, size, angle, alpha := backStabParticle(effect, i, frame)
		if alpha > 0 {
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture,
				position.x, position.z, position.y, size, size, angle, color.RGBA{130, 130, 250, uint8(alpha)}, true)
		}
	}
}

func (m *WorldMode) applyCloseConfineStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusRgCconfineM && change.StatusID != db.StatusRgCconfineS {
		return
	}
	id := change.ActorID
	if id == 0 || isLocalActor(ctx, id) {
		id = localSkillTarget(ctx)
	}
	m.setActorStateEffect(ctx, id, effectNPCStop2, change.Active)
	if change.Active && change.HasDuration {
		now := time.Now()
		for i := range m.worldEffects {
			effect := &m.worldEffects[i]
			if effect.effectID == effectNPCStop2 && effect.actorID == id && !effect.expired(now) {
				effect.expires = now.Add(change.Duration)
			}
		}
	}
}
