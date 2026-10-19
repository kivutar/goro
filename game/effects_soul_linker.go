package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

type soulLinkerParticles struct {
	origins [21]modelPoint3
	count   int
}

func initSoulLinkerProjectile(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	origin := destination
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		origin = effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, effect.starts)
	}
	direction := sub3(destination, origin)
	direction.y = 0
	if distance := math.Hypot(direction.x, direction.z); distance > 0 {
		direction = mul3(direction, 1/distance)
	}
	// Estin/Estun keep their cast positions; Esma keeps the launch position
	// while homing toward its target.
	if effect.effectID == effectStin2 {
		origin.y += 2
		effect.esma = &esmaProjectile{}
	} else {
		origin.y += float64(10-hitEffectRandom(*effect, 1, 5)) / 5
	}
	effect.actorOrigin = origin
	effect.projectileDirection = direction
	effect.hasActorTransform = true
}

// dhxj PrimSTIN: three startup ticks, nine accelerating fade-in ticks,
// then faster acceleration while fading. Distances are converted to cells.
func estinHead(process int) (distance, alpha float64) {
	if process <= 0 {
		return 0, 0
	}
	rise := min(process-1, 9)
	fall := max(process-10, 0)
	distance = .8 * (1 + 1.02*(math.Pow(1.02, float64(rise))-1)/.02)
	distance += .8 * math.Pow(1.02, 9) * 1.04 * (math.Pow(1.04, float64(fall)) - 1) / .04
	return distance / 5, float64(max(25*rise-5*fall, 0)) / 255
}

func (m *WorldMode) drawEstinEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame >= 200 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	process := frame + 1 - 3
	var vertices []render.Vertex3D
	var indices []uint16
	for trail := 0; trail < 4; trail++ {
		sample := process
		if trail > 0 {
			// PrimSTIN copies the head into a three-slot history every third tick.
			sample = min(process, 56)/3*3 - (trail-1)*3
		}
		distance, alpha := estinHead(sample)
		if alpha <= 0 {
			continue
		}
		alpha *= math.Pow(.8, float64(trail))
		center := add3(effect.actorOrigin, mul3(effect.projectileDirection, distance))
		var corners [4]modelPoint3
		for i := range corners {
			angle := degreesToRadians(float64(sample*30 + (i+1)*90))
			corners[i] = add3(center, modelPoint3{x: 4 * math.Cos(angle), z: 4 * math.Sin(angle)})
		}
		appendSoulLinkerQuad(&vertices, &indices, texture, corners, effectComponentTint(component, alpha))
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func (m *WorldMode) drawSoulLinkerParticles(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || len(component.textureFiles) != 2 {
		return
	}
	textures := [2]*render.Image{
		m.effectFileTexture(ctx.Resources, component.textureFiles[0]),
		m.effectFileTexture(ctx.Resources, component.textureFiles[1]),
	}
	start, end, step, life := 0, 80, 4, 55
	jitter, radius, radiusStep := 11, 2.0, .1
	if effect.effectID == effectSmaReady {
		start, end = 40, 120
	} else if effect.effectID == effectStin3 {
		end, step, life = 199, 1, 50
		jitter, radius, radiusStep = 9, 1.2, .08
	}
	if effect.effectID != effectStin3 {
		if effect.buffParticles == nil {
			return
		}
		// Keep each emission's position when the actor replaces its movement path.
		p := effect.buffParticles
		for p.count < len(p.origins) && start+p.count*step <= min(frame, end) {
			at := effect.starts.Add(time.Duration(start+p.count*step) * db.EffectFrameDuration)
			p.origins[p.count] = effectWorldAnchor(ctx, effect, at)
			p.count++
		}
	}
	// ParticleTime and SMA both emit four independently rising particles.
	first := max(start, (frame-life+step)/step*step)
	for emitted := first; emitted <= min(frame, end); emitted += step {
		age := frame - emitted + 1
		alpha := float64(max(15*(min(age, 10)-1)-3*max(age-10, 0), 0)) / 255
		if alpha <= 0 {
			continue
		}
		var center modelPoint3
		if effect.effectID == effectStin3 {
			center = add3(effect.actorOrigin, mul3(effect.projectileDirection, .8*float64(max(emitted-2, 0))/5))
		} else {
			center = effect.buffParticles.origins[(emitted-start)/step]
		}
		for i := 0; i < 4; i++ {
			salt := (emitted*4 + i) * 5
			position := add3(center, modelPoint3{
				x: float64(hitEffectRandom(effect, salt+2, jitter)-jitter/2) / 5,
				z: float64(hitEffectRandom(effect, salt+3, jitter)-jitter/2) / 5,
				y: (6 + float64(age)*(.1+.01*float64(hitEffectRandom(effect, salt+4, 20)))) / 5,
			})
			size := math.Sqrt2 * (radius + radiusStep*float64(hitEffectRandom(effect, salt+5, 10))) / 5
			index := 0
			if age == 35-i*7 || age == 36-i*7 {
				index = 1
			}
			drawTexturedEffectBillboardRotatedXY(screen, projection, textures[index],
				position.x, position.z, position.y, size, size, degreesToRadians(float64(-5*age+45)), effectComponentTint(component, alpha), true)
		}
	}
}

func (m *WorldMode) drawSoulLinkerHitLines(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame >= 200 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	// HitLine4(3) for Estun; HitLine3(3) for Esma.
	count, start, randomDelay, fixedDelay, baseRadius, heightBase := 16, 6, 31, 0, .05/5, .8
	if effect.effectID == effectHitLine6 {
		count, start, randomDelay, fixedDelay, baseRadius, heightBase = 20, 0, 41, 30, .07/5, .6
	}
	for i := 0; i < count; i++ {
		salt := i * 5
		age := frame - (start + i/4) - fixedDelay - hitEffectRandom(effect, salt, randomDelay) + 1
		if age <= 0 || age >= 34 {
			continue
		}
		angle := deterministicAngle(effect, salt+1)
		sin, cos := math.Sincos(angle)
		distance := float64(6+hitEffectRandom(effect, salt+2, 6)) / 5
		height := heightBase + .1*float64(hitEffectRandom(effect, salt+3, 7))
		alpha := float64(125 + hitEffectRandom(effect, salt+4, 121))
		point := func(tick int) modelPoint3 {
			phase := float64(clampInt(tick, 0, 30) * 6)
			travel := phase / 90 * distance
			return modelPoint3{x: x + cos*travel, y: z + 1 + distance*math.Sin(degreesToRadians(phase))*height, z: y + sin*travel}
		}
		for trail := 0; trail < 4; trail++ {
			// PrimHitLine3 updates history after advancing its head, so the
			// first history slot has the same position as the head.
			a, b := point(age-max(0, trail-1)), point(age-trail)
			if a == b {
				continue
			}
			for layer := 0; layer < 2; layer++ {
				radius := baseRadius
				brightness := uint8(255 - (5-trail)*30)
				tint := color.RGBA{brightness, brightness, brightness, uint8(alpha - float64(trail*10))}
				if layer == 1 {
					radius *= 4
					tint = color.RGBA{0, 0, 255, tint.A / 2}
				}
				offset := func(side int) modelPoint3 {
					s, c := math.Sincos(degreesToRadians(float64(side * 120)))
					return modelPoint3{x: -sin * radius * c, y: -radius * s, z: cos * radius * c}
				}
				for side := 0; side < 3; side++ {
					p, q := offset(side), offset(side+1)
					appendSoulLinkerQuad(&vertices, &indices, texture, [4]modelPoint3{add3(a, q), add3(b, q), add3(b, p), add3(a, p)}, tint)
				}
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func appendSoulLinkerQuad(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, corners [4]modelPoint3, tint color.RGBA) {
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	base := uint16(len(*vertices))
	for i, uv := range [...]texturePoint{{u: 0, v: 0}, {u: 1, v: 0}, {u: 1, v: 1}, {u: 0, v: 1}} {
		*vertices = append(*vertices, texturedSurfaceVertex3D(corners[i], uv, tint, w, h))
	}
	*indices = append(*indices, base, base+1, base+2, base, base+2, base+3)
}
