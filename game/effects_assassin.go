package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawSonicBlowEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	right, up, units, ok := projection.BillboardBasis(x, y, z)
	if texture == nil || !ok {
		return
	}
	// Sakexe's PP_2DCIRCLE is a ten-pixel band, twenty pixels above the caster.
	n := float64(now.Sub(effect.starts)/db.EffectFrameDuration + 1)
	radius := 7*n - .175*n*(n+1)/2
	alpha := clampFloat((40-n)/40, 0, 1) * 254 / 255
	center := add3(modelPoint3{x: x, y: z, z: y}, mul3(up, 20*units))
	tint := color.RGBA{255, 255, 255, uint8(alpha * 255)}
	bounds := texture.Bounds()
	vertices := make([]render.Vertex3D, 0, 74)
	indices := make([]uint16, 0, 216)
	for i := 0; i <= 36; i++ {
		angle := float64(i) * math.Pi / 18
		axis := add3(mul3(right, math.Cos(angle)), mul3(up, math.Sin(angle)))
		for j, r := range [2]float64{math.Max(0, radius-10), radius} {
			point := add3(center, mul3(axis, r*units))
			uv := texturePoint{u: float32(i) / 4, v: float32(1 - j)}
			vertices = append(vertices, texturedSurfaceVertex3D(point, uv, tint, float32(bounds.Dx()), float32(bounds.Dy())))
		}
		if i < 36 {
			base := uint16(i * 2)
			indices = append(indices, base, base+1, base+2, base+1, base+3, base+2)
		}
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.DepthTest = false
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}

func initAssassinProjectile(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	origin := destination
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		origin = effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, effect.starts)
		// The skill mapping passes victim, caster; these effects belong to the caster.
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	direction := sub3(destination, origin)
	if effect.effectID == effectGrimtooth {
		direction.y = 0
		// The original stops emitting immediately for a target within 2.5 native units.
		if distance := math.Hypot(direction.x, direction.z); distance > .5 {
			direction = mul3(direction, .4/distance)
		} else {
			direction = modelPoint3{}
		}
	} else {
		// ThrowItem keeps the entire displacement to calculate its flight time.
		origin.y += 1.6
	}
	effect.actorOrigin = origin
	effect.projectileDirection = direction
	effect.hasActorTransform = true
}

func venomKnifeAt(effect worldEffect, process int) (position modelPoint3, angle float64, alpha int) {
	if process <= 0 {
		return modelPoint3{}, 0, 0
	}
	delta := effect.projectileDirection
	distance := math.Hypot(delta.x, delta.z)
	travel := int(distance/.6) + 1
	stop := int(float64(travel) * 1.05)
	n := min(process, stop)
	position = effect.actorOrigin
	if distance > 0 {
		position = add3(position, mul3(delta, .6*float64(n)/distance))
	}
	phase := min(180, int(180*float64(n)/float64(travel)))
	position.y += math.Sin(degreesToRadians(float64(phase))) / 5
	return position, degreesToRadians(float64((330 + process*45) % 360)), max(0, 255-20*max(0, process-stop))
}

func (m *WorldMode) drawVenomKnifeEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	process := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	for trail := 0; trail < 4; trail++ {
		step := process
		if trail > 0 {
			// PrimThrowItem shifts the three afterimages before every odd update.
			step = (process-1)/2*2 - (trail-1)*2
		}
		position, angle, alpha := venomKnifeAt(effect, step)
		if trail > 0 {
			alpha = max(0, alpha-150-(trail-1)*25)
		}
		if alpha == 0 {
			continue
		}
		// RenderCloud's four corners lie on a circle of radius four native units.
		size := .8 * math.Sqrt2
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture,
			position.x, position.z, position.y, size, size, angle-3*math.Pi/4,
			color.RGBA{255, 200, 255, uint8(alpha)}, false)
	}
}

type grimtoothHorn struct {
	origin, right, up, axis modelPoint3
	size, height, alpha     float64
}

func grimtoothSpike(effect worldEffect, index, frame int) grimtoothHorn {
	pitch, yaw := 75.0, float64((6-index)*120)
	size, height, speed, accel := .18, 5.0, .7, .0002
	life, stop, age, fade := 1000, 11, frame, 990
	offset := [3]modelPoint3{{y: -8, z: -2.4}, {x: 2.4, y: -8, z: 1.2}, {x: -2.4, y: -8, z: 1.2}}[index%3]
	if effect.effectID == effectGrimtooth {
		birth := index * 3
		age = frame - birth
		life, stop, fade = min(40, 250-birth), 7, 30
		if life < 40 {
			fade = 0
		}
		speed, accel, height = .6, -.03, 2
		size = float64(60+hitEffectRandom(effect, index*3+1, 40)) / 500
		pitch = float64(75 + hitEffectRandom(effect, index*3+2, 30))
		yaw = float64(hitEffectRandom(effect, index*3+3, 360))
		offset = add3(modelPoint3{y: -4}, mul3(effect.projectileDirection, float64(birth+1)))
	}
	if age < 0 || age >= life {
		return grimtoothHorn{}
	}
	// Native horns have a square base in XY and their tip on Z, rotated X then Y.
	// Convert the native downward Y axis to our upward one after the rotation.
	p, q := degreesToRadians(pitch), degreesToRadians(yaw)
	right := modelPoint3{x: math.Cos(q), z: -math.Sin(q)}
	up := modelPoint3{x: math.Sin(p) * math.Sin(q), y: -math.Cos(p), z: math.Sin(p) * math.Cos(q)}
	axis := modelPoint3{x: math.Cos(p) * math.Sin(q), y: math.Sin(p), z: math.Cos(p) * math.Cos(q)}
	n := float64(min(age+1, stop))
	distance := n*speed + accel*n*(n+1)/2
	return grimtoothHorn{
		origin: add3(effect.actorOrigin, add3(offset, mul3(axis, distance))),
		right:  right, up: up, axis: axis, size: size, height: height,
		alpha: clampFloat(float64(life-age-1)/float64(life-fade), 0, 1) * 254 / 255,
	}
}

func (m *WorldMode) drawGrimtoothEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	first, last := 0, 2
	if effect.effectID == effectGrimtooth {
		first, last = max(0, (frame-39+2)/3), min(frame/3, 83)
		if effect.projectileDirection == (modelPoint3{}) {
			last = 0
		}
	}
	bounds := texture.Bounds()
	vertices := make([]render.Vertex3D, 0, 168)
	indices := make([]uint16, 0, 168)
	for i := first; i <= last; i++ {
		horn := grimtoothSpike(effect, i, frame)
		if horn.alpha <= 0 {
			continue
		}
		tint := color.RGBA{255, 255, 255, uint8(horn.alpha * 255)}
		corners := [4]modelPoint3{
			mul3(add3(horn.right, horn.up), -horn.size),
			mul3(sub3(horn.right, horn.up), horn.size),
			mul3(add3(horn.right, horn.up), horn.size),
			mul3(sub3(horn.up, horn.right), horn.size),
		}
		for side, edge := range [4][2]int{{0, 1}, {0, 3}, {3, 2}, {2, 1}} {
			for j, local := range [3]modelPoint3{mul3(horn.axis, horn.height), corners[edge[0]], corners[edge[1]]} {
				indices = append(indices, uint16(len(vertices)))
				vertices = append(vertices, texturedSurfaceVertex3D(add3(horn.origin, local), quadHornUVs[side*3+j], tint, float32(bounds.Dx()), float32(bounds.Dy())))
			}
		}
	}
	if len(vertices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
	}
}

type poisonParticle struct {
	position           modelPoint3
	size, angle, alpha float64
}

func poisonParticleAt(effect worldEffect, index, age int) poisonParticle {
	life, fade, initial := 40, 32, 0.0
	speed := float64(30+hitEffectRandom(effect, index*4+1, 30)) / 500
	size := float64(40+hitEffectRandom(effect, index*4+2, 50)) / 100
	yaw := degreesToRadians(float64(hitEffectRandom(effect, index*4+3, 360)))
	radius := float64(2+hitEffectRandom(effect, index*4+4, 6)) / 5
	if effect.effectID == effectVenomDust2 {
		life, fade, initial = 10, 8, 70
		speed = float64(20+hitEffectRandom(effect, index*4+1, 20)) / 400
		size, radius = .8, 0
	}
	if age < 0 || age >= life {
		return poisonParticle{}
	}
	n := float64(age + 1)
	alpha := math.Min(254, initial+25.4*n)
	if age >= fade {
		alpha = math.Min(254, initial+25.4*float64(fade)) * float64(life-age-1) / float64(life-fade)
	}
	return poisonParticle{
		position: modelPoint3{x: math.Sin(yaw) * radius, y: n*speed - speed/(2*float64(life))*n*(n+1)/2, z: math.Cos(yaw) * radius},
		size:     size, angle: degreesToRadians(n * 3), alpha: alpha / 255,
	}
}

func (m *WorldMode) drawPoisonParticlesEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 || len(view.act.Actions[0].Animations) == 0 {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	life, last := 40, min(frame/5, 42)
	if effect.effectID == effectVenomDust2 {
		life, last = 10, frame/5
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	for i := max(0, (frame-life+5)/5); i <= last; i++ {
		age := frame - i*5
		particle := poisonParticleAt(effect, i, age)
		if particle.alpha <= 0 {
			continue
		}
		motion := (age/4 + 1) % len(view.act.Actions[0].Animations)
		key := singleSpriteBillboardKey{motion: motion}
		billboard, ok := view.billboards[key]
		if !ok {
			billboard, ok = composeSingleSpriteBillboard(view, view.act.Actions[0].Animations[motion])
			if !ok {
				continue
			}
			view.billboards[key] = billboard
		}
		position := add3(effect.actorOrigin, particle.position)
		drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard,
			position.x, position.z, position.y, particle.size*effectPixelRatio, -particle.angle, particle.alpha, 1, color.RGBA{255, 255, 255, 255}, options)
	}
}
