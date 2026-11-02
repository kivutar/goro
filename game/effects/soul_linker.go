package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func initSoulLinkerProjectile(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	origin := destination
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		origin = EffectWorldAnchor(ctx, Instance{ActorID: effect.TargetID}, effect.Starts)
	}
	direction := scene.Sub3(destination, origin)
	direction.Y = 0
	if distance := math.Hypot(direction.X, direction.Z); distance > 0 {
		direction = scene.Mul3(direction, 1/distance)
	}
	// Estin/Estun keep their cast positions; Esma keeps the launch position
	// while homing toward its target.
	if effect.EffectID == Stin2 {
		origin.Y += 2
		effect.Esma = &esmaProjectile{}
	} else {
		origin.Y += float64(10-HitEffectRandom(*effect, 1, 5)) / 5
	}
	effect.ActorOrigin = origin
	effect.ProjectileDirection = direction
	effect.HasActorTransform = true
}

// Estin uses three startup ticks, nine accelerating fade-in ticks,
// then faster acceleration while fading. Distances are converted to cells.
func EstinHead(process int) (distance, alpha float64) {
	if process <= 0 {
		return 0, 0
	}
	rise := min(process-1, 9)
	fall := max(process-10, 0)
	distance = .8 * (1 + 1.02*(math.Pow(1.02, float64(rise))-1)/.02)
	distance += .8 * math.Pow(1.02, 9) * 1.04 * (math.Pow(1.04, float64(fall)) - 1) / .04
	return distance / 5, float64(max(25*rise-5*fall, 0)) / 255
}

func (m *System) drawEstinEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) || frame >= 200 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
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
		distance, alpha := EstinHead(sample)
		if alpha <= 0 {
			continue
		}
		alpha *= math.Pow(.8, float64(trail))
		center := scene.Add3(effect.ActorOrigin, scene.Mul3(effect.ProjectileDirection, distance))
		var corners [4]scene.Point3
		for i := range corners {
			angle := scene.DegreesToRadians(float64(sample*30 + (i+1)*90))
			corners[i] = scene.Add3(center, scene.Point3{X: 4 * math.Cos(angle), Z: 4 * math.Sin(angle)})
		}
		appendEffectQuad(&vertices, &indices, texture, corners, effectComponentTint(component, alpha))
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}

type esmaSample struct {
	Position scene.Point3
	Alpha    float64
	spin     float64
}

type esmaHead struct {
	Samples [4]esmaSample
	heading float64
	fading  bool
}

type esmaProjectile struct {
	tick  int
	Heads [2]esmaHead
}

// PrimSTIN2 turns the two heads in opposite directions at 60 Hz. Retaining
// their small histories avoids bending an old trail when the target moves.
func (p *esmaProjectile) Advance(ctx client.Context, effect Instance, now time.Time) {
	end := min(int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1, 200)
	for p.tick < end {
		p.tick++
		at := effect.Starts.Add(time.Duration(p.tick-1) * db.EffectFrameDuration)
		target := EffectWorldAnchor(ctx, effect, at)
		_, targetExists, _ := scene.ActorForCombatID(ctx, effect.ActorID)
		for side := range p.Heads {
			head := &p.Heads[side]
			turn := 1.0
			if side == 0 {
				turn = -1
			}
			if p.tick == 1 {
				for i := range head.Samples {
					head.Samples[i].Position = effect.ActorOrigin
				}
				head.heading = math.Atan2(target.Z-effect.ActorOrigin.Z, target.X-effect.ActorOrigin.X)*180/math.Pi - turn*90
			}
			if !targetExists || p.tick > 140 {
				head.fading = true
			}
			sample := &head.Samples[0]
			if head.fading && sample.Alpha <= 0 {
				continue
			}
			sample.spin = float64(p.tick * 30)
			if p.tick > 1 && p.tick <= 20 {
				sample.Alpha = min(sample.Alpha+15, 250)
			}
			if p.tick > 5 {
				if head.fading {
					for i := range head.Samples {
						head.Samples[i].Alpha = max(head.Samples[i].Alpha-5, 0)
					}
					head.heading += turn
				} else {
					toward := math.Atan2(target.Z-sample.Position.Z, target.X-sample.Position.X) * 180 / math.Pi
					toward = math.Mod(toward+360, 360)
					if math.Abs(head.heading-toward) > 3 {
						head.heading += turn * 5
					}
				}
			}
			head.heading = math.Mod(head.heading+360, 360)
			sin, cos := math.Sincos(scene.DegreesToRadians(head.heading))
			sample.Position.X += .4 * cos
			sample.Position.Z += .4 * sin
			if p.tick%3 == 0 {
				for i := 3; i > 0; i-- {
					head.Samples[i] = head.Samples[i-1]
					head.Samples[i].Alpha *= .8
				}
			}
		}
	}
}

func (m *System) drawEsmaEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	if now.Before(effect.Starts) || effect.Esma == nil {
		return
	}
	effect.Esma.Advance(ctx, effect, now)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for _, head := range effect.Esma.Heads {
		for _, sample := range head.Samples {
			if sample.Alpha <= 0 {
				continue
			}
			var corners [4]scene.Point3
			for i := range corners {
				sin, cos := math.Sincos(scene.DegreesToRadians(sample.spin + float64((i+1)*90)))
				corners[i] = scene.Add3(sample.Position, scene.Point3{X: 5 * cos, Z: 5 * sin})
			}
			appendEffectQuad(&vertices, &indices, texture, corners, effectComponentTint(component, sample.Alpha/255))
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}

func (m *System) drawEsmaHitTexture(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	age := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 8 * min(age-40, 82-age)
	if now.Before(effect.Starts) || alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	size := math.Sqrt2 * (10 + .7*float64(age-40)) / 5
	size *= 1 + .05*math.Sin(scene.DegreesToRadians(float64(age*2)))
	// RenderEffectTexture's hit branch falls through the white and grey passes.
	for _, tint := range [...]color.RGBA{{130, 130, 250, uint8(alpha)}, {255, 255, 255, uint8(alpha)}, {150, 150, 150, uint8(alpha)}, {50, 50, 50, uint8(alpha)}} {
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, x, y, effect.ActorOrigin.Y+2, size, size, 0, tint, true)
	}
}

// PrimSoulParticle and PrimPeongUp share the same wandering motion. Keep
// their history so neither frame rate nor camera movement resets a trail.
type SoulLights struct {
	tick         int
	Particles    []wanderingEffectParticle
	glyphOrigins [2]scene.Point3
}

func soulLinkLightPosition(tick int, right scene.Point3) scene.Point3 {
	angle := min(tick*(tick+1)/60, 359)
	p := scene.Mul3(right, (-30+float64(angle)/6)/5)
	p.Y = (40 + 20*math.Sin(scene.DegreesToRadians(float64(angle)))) / 5
	return p
}

func (p *SoulLights) Advance(ctx client.Context, effect Instance, right scene.Point3, now time.Time) {
	end := min(int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1, 300)
	dodge := effect.EffectID == StormKick4
	for p.tick < end {
		p.tick++
		if !dodge && (p.tick == 2 || p.tick == 22) {
			at := effect.Starts.Add(time.Duration(p.tick-1) * db.EffectFrameDuration)
			p.glyphOrigins[(p.tick-2)/20] = EffectWorldAnchor(ctx, effect, at)
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
			particle := wanderingEffectParticle{seed: seed, kind: i % 3, speed: .07 / 5,
				Position: soulLinkLightPosition(p.tick, right),
				radius:   (1 + float64(HitEffectRandom(effect, (p.tick*2+i/3)*16, 11))*.1) / 5}
			for j := range particle.angles {
				particle.angles[j] = HitEffectRandom(effect, seed+j, 360)
			}
			if dodge {
				angle := deterministicAngle(effect, seed+4)
				particle.Position = scene.Point3{X: .7 * math.Cos(angle), Z: .7 * math.Sin(angle), Y: float64(HitEffectRandom(effect, seed+5, 8)) / 5}
				particle.Age = -HitEffectRandom(effect, seed+6, 26)
				particle.radius = (2 + float64(HitEffectRandom(effect, seed+7, 11))*.05) / 5
				particle.speed = (.08 + float64(HitEffectRandom(effect, seed+8, 11))*.005) / 5
			}
			p.Particles = append(p.Particles, particle)
		}
		live := p.Particles[:0]
		for _, particle := range p.Particles {
			particle.Age++
			if particle.Age > 0 {
				if (dodge && particle.Age >= 102) || (!dodge && particle.Age >= 90) {
					continue
				}
				advanceWanderingAngles(&particle, effect, p.tick)
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
				if particle.Position.Y > 0 {
					particle.Position.X += x * math.Sin(scene.DegreesToRadians(float64(particle.angles[0])))
					particle.Position.Z += z * math.Sin(scene.DegreesToRadians(float64(particle.angles[2])))
				}
				particle.Position.Y += y
			}
			live = append(live, particle)
		}
		p.Particles = live
	}
}

func (m *System) drawSoulLights(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if now.Before(effect.Starts) || effect.SoulLights == nil {
		return
	}
	center := EffectWorldAnchor(ctx, effect, now)
	if effect.EffectID == StormKick4 {
		center = effect.ActorOrigin
	}
	right, up, _, ok := projection.BillboardBasis(center.X, center.Z, center.Y)
	if !ok {
		return
	}
	effect.SoulLights.Advance(ctx, effect, right, now)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for _, particle := range effect.SoulLights.Particles {
		if particle.Age <= 0 {
			continue
		}
		a := max(20*min(particle.Age, 9)-3*max(particle.Age-30, 0), 0)
		c := color.RGBA{80, 80, 255, uint8(a)}
		if effect.EffectID == StormKick4 {
			a = max(5*min(particle.Age, 29)-2*max(particle.Age-29, 0), 0)
			c = color.RGBA{105, 105, 225, uint8(a)}
		}
		half := particle.radius / math.Sqrt2
		position := scene.Add3(center, particle.Position)
		if a == 0 || particle.Position.Y < half*up.Y {
			continue
		}
		r, u := scene.Mul3(right, half), scene.Mul3(up, half)
		appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
			scene.Add3(scene.Sub3(position, r), u), scene.Add3(scene.Add3(position, r), u), scene.Sub3(scene.Add3(position, r), u), scene.Sub3(scene.Sub3(position, r), u),
		}, c)
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}

func SoulLinkGlyph(age, layer int) (radius, alpha float64) {
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

func (m *System) drawSoulLinkEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if now.Before(effect.Starts) || effect.SoulLights == nil || len(component.TextureFiles) != 8 {
		return
	}
	m.drawSoulLights(screen, ctx, projection, component, effect, now)
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	center := EffectWorldAnchor(ctx, effect, now)
	right, _, _, ok := projection.BillboardBasis(center.X, center.Z, center.Y)
	if !ok {
		return
	}
	for i, file := range component.TextureFiles {
		age := frame - i*20
		texture := m.EffectFileTexture(ctx.Resources, file)
		position := scene.Add3(effect.SoulLights.glyphOrigins[i/4], scene.Mul3(right, float64(-22+i*6+(i/4)*2)/5))
		position.Y += 8
		for layer := 10; layer >= 1; layer-- {
			radius, alpha := SoulLinkGlyph(age, layer)
			if alpha <= 0 {
				continue
			}
			size := math.Sqrt2 * radius
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, position.X, position.Z, position.Y, size, size, 0, color.RGBA{255, 255, 255, uint8(alpha * 255)}, false)
		}
	}
	tick := frame + 1
	a := max(20*min(tick, 10)-5*max(tick-170, 0), 0)
	if a > 0 {
		p := scene.Add3(center, soulLinkLightPosition(tick, right))
		drawTexturedEffectBillboardRotatedXY(screen, projection, m.EffectFileTexture(ctx.Resources, component.TextureFile), p.X, p.Z, p.Y, math.Sqrt2/2, math.Sqrt2/2, 0, color.RGBA{125, 125, 255, uint8(a)}, false)
	}
}

func (m *System) DrawKaizelEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	// EF_KAIZEL launches JobLvUp50_2(0) and (45) at tick 15.
	age := int(now.Sub(effect.Starts)/db.EffectFrameDuration) - 15 + 1
	if now.Before(effect.Starts) {
		return
	}
	m.drawRadialBlades(screen, ctx, component.TextureFile, scene.Point3{X: x, Y: z, Z: y}, age, 1.5, 0, 8)
}
