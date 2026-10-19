package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

type soulLightParticle struct {
	position        modelPoint3
	angles          [4]int
	seed, age, kind int
	radius, speed   float64
}

// PrimSoulParticle and PrimPeongUp share the same wandering motion. Keep
// their history so neither frame rate nor camera movement resets a trail.
type soulLights struct {
	tick         int
	particles    []soulLightParticle
	glyphOrigins [2]modelPoint3
}

func soulLinkLightPosition(tick int, right modelPoint3) modelPoint3 {
	angle := min(tick*(tick+1)/60, 359)
	p := mul3(right, (-30+float64(angle)/6)/5)
	p.y = (40 + 20*math.Sin(degreesToRadians(float64(angle)))) / 5
	return p
}

func (p *soulLights) advance(ctx client.Context, effect worldEffect, right modelPoint3, now time.Time) {
	end := min(int(now.Sub(effect.starts)/db.EffectFrameDuration)+1, 300)
	dodge := effect.effectID == effectStormKick4
	for p.tick < end {
		p.tick++
		if !dodge && (p.tick == 2 || p.tick == 22) {
			at := effect.starts.Add(time.Duration(p.tick-1) * db.EffectFrameDuration)
			p.glyphOrigins[(p.tick-2)/20] = effectWorldAnchor(ctx, effect, at)
		}
		count := 0
		if dodge && p.tick == 1 {
			count = 120
		}
		if !dodge && p.tick < 180 {
			count = 6
		}
		for i := 0; i < count; i++ {
			seed := (p.tick*120 + i) * 16
			particle := soulLightParticle{seed: seed, kind: i % 3, speed: .07 / 5,
				position: soulLinkLightPosition(p.tick, right),
				radius:   (1 + float64(hitEffectRandom(effect, (p.tick*2+i/3)*16, 11))*.1) / 5}
			for j := range particle.angles {
				particle.angles[j] = hitEffectRandom(effect, seed+j, 360)
			}
			if dodge {
				angle := deterministicAngle(effect, seed+4)
				particle.position = modelPoint3{x: .7 * math.Cos(angle), z: .7 * math.Sin(angle), y: float64(hitEffectRandom(effect, seed+5, 8)) / 5}
				particle.age = -hitEffectRandom(effect, seed+6, 26)
				particle.radius = (2 + float64(hitEffectRandom(effect, seed+7, 11))*.05) / 5
				particle.speed = (.08 + float64(hitEffectRandom(effect, seed+8, 11))*.005) / 5
			}
			p.particles = append(p.particles, particle)
		}
		live := p.particles[:0]
		for _, particle := range p.particles {
			particle.age++
			if particle.age > 0 {
				if (dodge && particle.age >= 102) || (!dodge && particle.age >= 90) {
					continue
				}
				for axis := 0; axis < 4; axis += 2 {
					a, target := &particle.angles[axis], &particle.angles[axis+1]
					if *target > *a {
						*a = min(*a+3, 359)
						if *a > *target {
							*target = hitEffectRandom(effect, particle.seed+p.tick*4+axis, max(*a, 1))
						}
					} else {
						*a = max(*a-2, 0)
						if *a <= *target {
							*target = 359 - hitEffectRandom(effect, particle.seed+p.tick*4+axis, max(*a, 1))
						}
					}
				}
				x, z, y := particle.speed, particle.speed, particle.speed
				if dodge {
					x, z = .03/5, .03/5
				} else {
					switch particle.kind {
					case 1:
						x, z, y = -x, -z, -y
					case 2:
						x, z, y = x*1.2, -z*1.2, 0
					}
				}
				if particle.position.y > 0 {
					particle.position.x += x * math.Sin(degreesToRadians(float64(particle.angles[0])))
					particle.position.z += z * math.Sin(degreesToRadians(float64(particle.angles[2])))
				}
				particle.position.y += y
			}
			live = append(live, particle)
		}
		p.particles = live
	}
}

func (m *WorldMode) drawSoulLights(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if now.Before(effect.starts) || effect.soulLights == nil {
		return
	}
	center := effectWorldAnchor(ctx, effect, now)
	if effect.effectID == effectStormKick4 {
		center = effect.actorOrigin
	}
	right, up, _, ok := projection.BillboardBasis(center.x, center.z, center.y)
	if !ok {
		return
	}
	effect.soulLights.advance(ctx, effect, right, now)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for _, particle := range effect.soulLights.particles {
		if particle.age <= 0 {
			continue
		}
		a := max(20*min(particle.age, 9)-3*max(particle.age-30, 0), 0)
		c := color.RGBA{80, 80, 255, uint8(a)}
		if effect.effectID == effectStormKick4 {
			a = max(5*min(particle.age, 29)-2*max(particle.age-29, 0), 0)
			c = color.RGBA{105, 105, 225, uint8(a)}
		}
		half := particle.radius / math.Sqrt2
		position := add3(center, particle.position)
		if a == 0 || particle.position.y < half*up.y {
			continue
		}
		r, u := mul3(right, half), mul3(up, half)
		appendSoulLinkerQuad(&vertices, &indices, texture, [4]modelPoint3{
			add3(sub3(position, r), u), add3(add3(position, r), u), sub3(add3(position, r), u), sub3(sub3(position, r), u),
		}, c)
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func soulLinkGlyph(age, layer int) (radius, alpha float64) {
	if age <= 0 {
		return 0, 0
	}
	a := min(13*min(age, 19), 10+layer*23)
	if layer <= 8 {
		a -= 2 * max(age-19, 0)
	} else {
		a -= 5 * max(age-100, 0)
	}
	growth := min(max(age-100, 0), 43) + min(max(age-100, 0), 47)
	return (5*math.Pow(1.005, float64(growth)) + float64(10-layer)) / 5, float64(max(a, 0)) / 255
}

func (m *WorldMode) drawSoulLinkEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if now.Before(effect.starts) || effect.soulLights == nil || len(component.textureFiles) != 8 {
		return
	}
	m.drawSoulLights(screen, ctx, projection, component, effect, now)
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	center := effectWorldAnchor(ctx, effect, now)
	right, _, _, ok := projection.BillboardBasis(center.x, center.z, center.y)
	if !ok {
		return
	}
	for i, file := range component.textureFiles {
		age := frame - i*20
		texture := m.effectFileTexture(ctx.Resources, file)
		position := add3(effect.soulLights.glyphOrigins[i/4], mul3(right, float64(-22+i*6+(i/4)*2)/5))
		position.y += 8
		for layer := 10; layer >= 1; layer-- {
			radius, alpha := soulLinkGlyph(age, layer)
			if alpha <= 0 {
				continue
			}
			size := math.Sqrt2 * radius
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, position.x, position.z, position.y, size, size, 0, color.RGBA{255, 255, 255, uint8(alpha * 255)}, false)
		}
	}
	tick := frame + 1
	a := max(20*min(tick, 10)-5*max(tick-170, 0), 0)
	if a > 0 {
		p := add3(center, soulLinkLightPosition(tick, right))
		drawTexturedEffectBillboardRotatedXY(screen, projection, m.effectFileTexture(ctx.Resources, component.textureFile), p.x, p.z, p.y, math.Sqrt2/2, math.Sqrt2/2, 0, color.RGBA{125, 125, 255, uint8(a)}, false)
	}
}
