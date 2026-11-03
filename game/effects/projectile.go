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

func initDirectedEffect(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	origin := destination
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		origin = EffectWorldAnchor(ctx, Instance{ActorID: effect.TargetID}, effect.Starts)
		// The skill mapping passes victim, caster; these effects belong to the caster.
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
	}
	direction := scene.Sub3(destination, origin)
	if effect.EffectID == Grimtooth {
		direction.Y = 0
		// The original stops emitting immediately for a target within 2.5 native units.
		if distance := math.Hypot(direction.X, direction.Z); distance > .5 {
			direction = scene.Mul3(direction, .4/distance)
		} else {
			direction = scene.Point3{}
		}
	} else {
		// ThrowItem keeps the entire displacement to calculate its flight time.
		origin.Y += 1.6
	}
	effect.ActorOrigin = origin
	effect.ProjectileDirection = direction
	effect.HasActorTransform = true
	if effect.EffectID == ThrowItem4 {
		effect.acidBottles = &acidBottleFlight{}
	}
}

func ThrowItemAt(effect Instance, process int) (position scene.Point3, angle float64, alpha int) {
	if process <= 0 {
		return scene.Point3{}, 0, 0
	}
	delta := effect.ProjectileDirection
	distance := math.Hypot(delta.X, delta.Z)
	speed, height, rotation, spin := .6, .2, 330, 45
	if effect.EffectID == ThrowItem3 {
		speed, height, rotation, spin = .4, 2, 0, 10
	}
	travel := int(distance/speed) + 1
	if isAlchemistBottle(effect.EffectID) {
		speed, height, rotation, spin = distance/25, 6, 0, 10
		travel = 26
	}
	stop := int(float64(travel) * 1.05)
	n := min(process, stop)
	position = effect.ActorOrigin
	if distance > 0 {
		position = scene.Add3(position, scene.Mul3(delta, speed*float64(n)/distance))
	}
	phase := min(180, int(180*float64(n)/float64(travel)))
	position.Y += math.Sin(scene.DegreesToRadians(float64(phase))) * height
	return position, scene.DegreesToRadians(float64((rotation + process*spin) % 360)), max(0, 255-20*max(0, process-stop))
}

func (m *System) drawThrowItemEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, componentIndex int, effect Instance, now time.Time) {
	texture := m.EffectFileTextureWithBlackKey(ctx.Resources, component.TextureFile, !isAlchemistBottle(effect.EffectID))
	if texture == nil {
		return
	}
	process := int((now.Sub(effect.Starts)-component.Delay)/db.EffectFrameDuration) + 1
	if now.Before(effect.Starts.Add(component.Delay)) {
		return
	}
	if effect.EffectID == ThrowItem4 {
		if effect.acidBottles == nil || componentIndex >= effect.acidBottles.count {
			return
		}
		// Each bottle starts at the caster's position on its launch tick, but
		// both aim at the victim's position from the cast notification.
		destination := scene.Add3(effect.ActorOrigin, effect.ProjectileDirection)
		destination.Y -= 1.6
		effect.ActorOrigin = effect.acidBottles.origins[componentIndex]
		effect.ProjectileDirection = scene.Sub3(destination, effect.ActorOrigin)
		effect.ActorOrigin.Y += 1.6
		effect.ItemLevel = 4
	}
	for trail := 0; trail < 4; trail++ {
		step := process
		if trail > 0 {
			// PrimThrowItem shifts the three afterimages before every odd update.
			step = (process-1)/2*2 - (trail-1)*2
		}
		position, angle, alpha := ThrowItemAt(effect, step)
		if trail > 0 {
			alpha = max(0, alpha-150-(trail-1)*25)
		}
		if alpha == 0 {
			continue
		}
		size := .8 * math.Sqrt2
		width, additive := size, false
		tint := color.RGBA{255, 200, 255, uint8(alpha)}
		if effect.EffectID == ThrowItem3 {
			size = .4 * math.Sqrt2
			width = size
			tint.G = 255
		}
		if isAlchemistBottle(effect.EffectID) {
			size = .6 * math.Sqrt2
			width = -size // RenderCloud reverses the bottle's texture corners.
			tint = color.RGBA{135, 255, 135, uint8(alpha)}
			if effect.EffectID != ThrowItem {
				colors := [...]color.RGBA{{255, 135, 135, 255}, {255, 205, 135, 255}, {255, 255, 135, 255}, {255, 255, 255, 255}, {135, 135, 255, 255}, {255, 255, 255, 255}}
				if effect.ItemLevel < 1 || effect.ItemLevel > len(colors) {
					continue
				}
				tint = colors[effect.ItemLevel-1]
				tint.A = uint8(alpha)
				additive = effect.ItemLevel == 6
			}
		}
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture,
			position.X, position.Z, position.Y, width, size, angle-3*math.Pi/4,
			tint, additive)
	}
}

// Shield Boomerang and Monk Tanji share the native spinning-projectile primitive.
type shieldImage struct {
	Position scene.Point3
	angle    float64
	Alpha    int
}

type shieldFlight struct {
	Images         [4]shieldImage
	tick, Phase    int
	heading, slope float64
	distance       float64
	oneWay         bool
	deflection     float64
}

func initShieldBoomerang(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
	}
	origin := EffectWorldAnchor(ctx, *effect, effect.Starts)
	delta := scene.Sub3(destination, origin)
	distance := math.Hypot(delta.X, delta.Z)
	origin.Y += 1.6
	effect.Shield = &shieldFlight{
		Images:  [4]shieldImage{{Position: origin, Alpha: 255, angle: -3 * math.Pi / 4}},
		heading: math.Atan2(delta.Z, delta.X), distance: distance,
	}
	if distance > 0 {
		effect.Shield.slope = .4 * delta.Y / distance
	}
	// TANJI uses the same native primitive, with a delayed, one-way flight.
	if effect.EffectID == Tanji {
		effect.Shield.tick = -15
		effect.Shield.oneWay = true
		effect.Shield.deflection = scene.DegreesToRadians(float64(HitEffectRandom(*effect, 0, 45)))
		if HitEffectRandom(*effect, 1, 2) == 0 {
			effect.Shield.deflection *= -1
		}
		effect.Shield.Images[0].Alpha = 5
		effect.Shield.Images[0].angle = math.Pi / 4
	}
}

func (flight *shieldFlight) Advance(tick int, caster scene.Point3) {
	for flight.tick < min(tick, 499) {
		flight.tick++
		if flight.tick <= 0 {
			continue
		}
		current := &flight.Images[0]
		if flight.oneWay && flight.tick <= 10 {
			current.Alpha = min(255, current.Alpha+25)
		}
		if flight.oneWay || flight.tick%3 == 0 {
			flight.Images[3] = flight.Images[2]
			flight.Images[3].Alpha = max(0, flight.Images[3].Alpha-25)
			flight.Images[2] = flight.Images[1]
			flight.Images[2].Alpha = max(0, flight.Images[2].Alpha-25)
			flight.Images[1] = *current
			flight.Images[1].Alpha = max(0, current.Alpha-150)
		}
		current.angle -= math.Pi / 12
		switch flight.Phase {
		case 0:
			if .4*float64(flight.tick) > flight.distance {
				flight.Phase = 1
				if flight.oneWay {
					flight.Phase = 2
					flight.heading += flight.deflection
				}
			}
			current.Position.Y += flight.slope
		case 1:
			want := math.Atan2(caster.Z-current.Position.Z, caster.X-current.Position.X)
			want = math.Mod(want+2*math.Pi, 2*math.Pi)
			flight.heading = math.Mod(flight.heading+2*math.Pi, 2*math.Pi)
			if math.Abs(want-flight.heading) > scene.DegreesToRadians(4) {
				flight.heading = math.Mod(flight.heading+scene.DegreesToRadians(5), 2*math.Pi)
			} else {
				flight.Phase = 2
			}
		case 2:
			current.Alpha = max(0, current.Alpha-12)
			current.Position.Y -= flight.slope
		}
		speed := .4
		if flight.oneWay && flight.Phase == 2 {
			speed *= 3
		}
		current.Position.X += speed * math.Cos(flight.heading)
		current.Position.Z += speed * math.Sin(flight.heading)
	}
}

func drawShieldImage(screen *render.Frame, projection scene.Projection, texture *render.Image, particle shieldImage, size float64, additive bool) {
	if particle.Alpha <= 0 {
		return
	}
	p := particle.Position
	drawTexturedEffectBillboardRotatedXY(screen, projection, texture, p.X, p.Z, p.Y,
		size, size, particle.angle, color.RGBA{255, 255, 255, uint8(min(255, particle.Alpha))}, additive)
}

func (m *System) drawShieldBoomerangEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil || effect.Shield == nil {
		return
	}
	tick := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	size := 7 * math.Sqrt2 / 5
	if effect.Shield.oneWay {
		tick -= 15
		size = effect.Size
		if size == 0 {
			size = 2 * math.Sqrt2 / 5
		}
	}
	effect.Shield.Advance(tick, EffectWorldAnchor(ctx, effect, now))
	for _, particle := range effect.Shield.Images {
		drawShieldImage(screen, projection, texture, particle, size, effect.Shield.oneWay)
	}
}

func (m *System) drawChemicalDash(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	age := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 12*min(age, 10) - 4*max(0, age-20)
	piercing := effect.EffectID == Chemical4
	if piercing {
		alpha = 20*min(age, 10) - 2*max(0, age-20)
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	angle := math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X)
	direction := scene.Point3{X: math.Cos(angle), Z: math.Sin(angle)}
	side := scene.Point3{X: math.Sin(angle), Z: -math.Cos(angle)}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 32; i++ {
		seed := i * 6
		offset := float64(15 + HitEffectRandom(effect, seed, 11))
		if i/4%2 == 0 {
			offset = -offset
		}
		width := .3 + .1*float64(HitEffectRandom(effect, seed+1, 4))
		length := float64(20 + HitEffectRandom(effect, seed+2, 31))
		speed := 1.5 + .1*float64(HitEffectRandom(effect, seed+3, 21))
		travel := float64(HitEffectRandom(effect, seed+4, 200)) + float64(age)*speed
		if piercing {
			width += .1
			length += 10
			travel += .025 * float64(age*(age+1))
		}
		travel = math.Mod(travel, 200) - 100
		point := func(along, across float64) scene.Point3 {
			p := scene.Add3(effect.ActorOrigin, scene.Add3(scene.Mul3(direction, along/5), scene.Mul3(side, across/5)))
			p.Y += 1.2
			return p
		}
		shade := uint8(225 - 20*HitEffectRandom(effect, seed+5, 3))
		tint := color.RGBA{shade, shade, 255, uint8(alpha)}
		if piercing {
			tint.G, tint.B = 255, shade
		}
		appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
			point(travel+length, offset+width), point(travel, offset+width),
			point(travel, offset-width), point(travel+length, offset-width),
		}, tint)
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
}

func appendEffectTube(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, from, to scene.Point3, radius float64, sides int, tint color.RGBA) {
	delta := scene.Sub3(to, from)
	if delta.X == 0 && delta.Z == 0 {
		return
	}
	direction := math.Atan2(delta.Z, delta.X) + math.Pi/2
	var previous scene.Point3
	for i := 0; i <= sides; i++ {
		s, c := math.Sincos(float64(i) * 2 * math.Pi / float64(sides))
		offset := scene.Point3{X: radius * c * math.Cos(direction), Z: radius * c * math.Sin(direction), Y: -radius * s}
		if i > 0 {
			appendEffectQuad(vertices, indices, texture, [4]scene.Point3{
				scene.Add3(to, previous), scene.Add3(to, offset), scene.Add3(from, offset), scene.Add3(from, previous),
			}, tint)
		}
		previous = offset
	}
}

func splineProjectilePosition(effect Instance, tick int, roll float64) scene.Point3 {
	if tick == 0 {
		return effect.ActorOrigin
	}
	n := float64(tick)
	direction := scene.Normalize3(effect.ProjectileDirection)
	// Prim3DParticleSpline first retreats, then accelerates forward. Its
	// transverse arc is rotated around the flight axis for each soul.
	forward := scene.Sub3(scene.Mul3(effect.ProjectileDirection, n/40), scene.Mul3(direction, (3.5*n-.175*n*(n+1)/2)/5))
	arc := (2*n - .1*n*(n-1)/2) / 5
	s, c := math.Sincos(scene.DegreesToRadians(roll))
	p := scene.Add3(effect.ActorOrigin, forward)
	p.Y += 2 + arc*c
	p.X -= direction.Z * arc * s
	p.Z += direction.X * arc * s
	return p
}
