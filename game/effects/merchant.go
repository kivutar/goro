package effects

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func (m *System) initMammonite(ctx client.Context, effect *Instance) {
	if lessEffectsEnabled(ctx) {
		return
	}
	origin := EffectWorldAnchor(ctx, *effect, effect.Starts)
	effect.ActorOrigin, effect.HasActorTransform = origin, true
	effect.CoinTrails = new([5][]scene.Point3)
	// The native Coin effect spawns five PP_3DPARTICLEGRAVITY sprites at tick 20.
	// Compute their short paths once, so bounce audio and visuals share timing.
	for i := range effect.CoinTrails {
		salt := i * 80
		life := 40 + HitEffectRandom(*effect, salt, 30)
		yaw := scene.DegreesToRadians(float64(30 * HitEffectRandom(*effect, salt+1, 12)))
		axis := scene.Point3{X: math.Sin(yaw), Z: math.Cos(yaw)}
		position := scene.Add3(origin, scene.Mul3(axis, float64(2+HitEffectRandom(*effect, salt+2, 2))/5))
		position.Y += 2
		speed := float64(50+HitEffectRandom(*effect, salt+3, 50)) / 500
		accel := -speed / float64(life) / 3.5
		vertical := float64(50+HitEffectRandom(*effect, salt+4, 100)) / 125
		gravity := -vertical / float64(life) * 4
		trail := make([]scene.Point3, life)
		for age := range trail {
			speed = math.Max(0, speed+accel)
			vertical += gravity
			position = scene.Add3(position, scene.Mul3(axis, speed))
			position.Y += vertical
			if vertical < 0 && position.Y < scene.TerrainHeightAtRenderPoint(ctx.World, position.X, position.Z) {
				vertical /= -1.5
				path := fmt.Sprintf("effect\\ef_coin%d.wav", 1+HitEffectRandom(*effect, salt+5+age, 3))
				m.scheduleSoundVolumeAtPosition(effect.Starts.Add(time.Duration(20+age)*db.EffectFrameDuration), 1, position.X-.5, position.Z-.5, path)
			}
			trail[age] = position
		}
		effect.CoinTrails[i] = trail
	}
}

func (m *System) DrawMammoniteCoins(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if effect.CoinTrails == nil || lessEffectsEnabled(ctx) {
		return
	}
	age := int(now.Sub(effect.Starts)/db.EffectFrameDuration) - 20
	for _, trail := range effect.CoinTrails {
		if age < 0 || age >= len(trail) {
			continue
		}
		alpha := effectParticleAlpha(age, len(trail), 1, len(trail)-len(trail)/3, 254)
		// ShiftSegment keeps one smaller, half-opacity afterimage.
		for segment := 1; segment >= 0; segment-- {
			position := effect.ActorOrigin
			if age >= segment {
				position = trail[age-segment]
			}
			m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, position,
				age+1, 2-.5*float64(segment), 0, color.RGBA{255, 255, 255, alpha / uint8(segment+1)})
		}
	}
}

func (m *System) DrawCartRevolutionEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	for _, birth := range [...]int{7, 20} {
		age := frame - birth
		if age < 0 || age >= 20 {
			continue
		}
		n := float64(age + 1)
		radius := (n - n*(n+1)/80) / 5
		if texture := m.EffectFileTexture(ctx.Resources, component.TextureFile); texture != nil {
			alpha := effectParticleAlpha(age, 20, 15, 5, 180)
			vertices, indices := groundEffectRingMesh(texture, effect.ActorOrigin, math.Max(0, 1.75*radius-1), 1.75*radius, alpha)
			options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
			options.DepthTest = false
			screen.DrawTriangles3DOwned(vertices, indices, texture, options)
		}
		if texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0]); texture != nil {
			alpha := effectParticleAlpha(age, 20, 7, 5, 240)
			vertices, indices := effectSphereMesh(texture, effect.ActorOrigin, 1.35*radius, 3*n, alpha)
			screen.DrawTriangles3DOwned(vertices, indices, texture, scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat))
		}
	}
}
