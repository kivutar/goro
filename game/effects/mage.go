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

func initMageProjectile(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
	}
	effect.ActorOrigin = EffectWorldAnchor(ctx, *effect, effect.Starts)
	effect.ProjectileDirection = scene.Sub3(destination, effect.ActorOrigin)
	effect.ProjectileDirection.Y = 0
	effect.HasActorTransform = true
	if effect.EffectID == FrostDiver {
		effect.Duration = time.Duration(FrostDiverTrailCount(*effect)-1+40) * db.EffectFrameDuration
		effect.Expires = effect.Starts.Add(effect.Duration)
	}
}

func FrostDiverTrailCount(effect Instance) int {
	distance := math.Hypot(effect.ProjectileDirection.X, effect.ProjectileDirection.Z)
	// One spike per tick, starting five native units ahead, advancing two,
	// and stopping within 2.5 units of the destination.
	return 1 + max(0, int(math.Ceil((distance*5-7.5)/2)))
}

func (m *System) drawSafetyWall(screen *render.Frame, ctx client.Context, component Component, effect Instance, origin scene.Point3, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 5*min(n, 15) - 2*max(0, n-3079)
	if alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 3; i++ {
		height := float64(34+3*i) / 5 * math.Sin(scene.DegreesToRadians(float64(min(n, 90))))
		appendEffectBand(&vertices, &indices, texture, origin, (3.6-.3*float64(i))/5,
			scene.DegreesToRadians(float64(HitEffectRandom(effect, i, 360))), 2*math.Pi, 0, 20,
			color.RGBA{255, 89, 182, uint8(alpha)}, func(j int) float64 {
				return height * (.75 + .25*math.Sin(scene.DegreesToRadians(float64(j*34+n*(4-i)))))
			})
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}

func (m *System) drawMageProjectile(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	if effect.EffectID == FrostDiver {
		m.DrawFrostDiverSpikes(screen, ctx, component, effect, effect.ActorOrigin, now)
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	delta := effect.ProjectileDirection
	if effect.EffectID == FireBall {
		angle := math.Atan2(delta.Z, delta.X) - math.Pi/2 + scene.DegreesToRadians(projection.CameraYaw)
		for i, alpha := range []uint8{255, 180, 130, 80} {
			age := frame - 20 - i*4
			if age < 0 || age >= 20 {
				continue
			}
			fraction := float64(age+1) / 20
			if math.Hypot(delta.X, delta.Z)*(1-fraction) <= .6 {
				continue
			}
			p := scene.Add3(effect.ActorOrigin, scene.Mul3(delta, fraction))
			p.Y += 3
			tint := color.RGBA{255, 255, 255, alpha}
			if i > 0 {
				tint.R, tint.G, tint.B = 246, 199, 76
			}
			m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, p, age+1, 1.3, angle, tint)
		}
		return
	}
	count := max(1, effect.HitCount)
	for i := 0; i < count; i++ {
		age := frame - 6 - i*11
		if age < 0 || age >= 40 {
			continue
		}
		roll := 1.0
		if count > 1 {
			roll = -90 + float64((i+1)*180)/float64(count-1)
		}
		for segment := 11; segment >= 0; segment-- {
			p := splineProjectilePosition(effect, max(0, age+1-segment), roll)
			scale := 2.75 * (1 - float64(segment)/24)
			alpha := uint8(255 * (1 - float64(segment)/12))
			m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, p, age/4+1, scale, 0, color.RGBA{255, 255, 255, alpha})
		}
	}
}

func (m *System) DrawFrostDiverSpikes(screen *render.Frame, ctx client.Context, component Component, effect Instance, origin scene.Point3, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	count := 8
	direction := scene.Normalize3(effect.ProjectileDirection)
	trail := effect.EffectID == FrostDiver
	if trail {
		count = FrostDiverTrailCount(effect)
	}
	vertices := make([]render.Vertex3D, 0, 8*12)
	indices := make([]uint16, 0, 8*12)
	first := 0
	if trail {
		first, count = max(0, frame-39), min(count, frame+1)
	}
	for i := first; i < count; i++ {
		age, center := frame, origin
		if trail {
			age -= i
			center = scene.Add3(center, scene.Mul3(direction, float64(5+2*i)/5))
		}
		if age < 0 || age >= 40 {
			continue
		}
		salt := i * 6
		radius := float64(1+HitEffectRandom(effect, salt, 4)) / 10
		size := float64(100+HitEffectRandom(effect, salt+1, 250)) / 100
		height := float64(200+HitEffectRandom(effect, salt+2, 100)) / 10
		pitch := float64(80 + HitEffectRandom(effect, salt+3, 20))
		alpha := uint8(200)
		if trail {
			radius = float64(5+HitEffectRandom(effect, salt, 10)) / 10
			size = float64(60+HitEffectRandom(effect, salt+1, 40)) / 100
			height = float64(150+HitEffectRandom(effect, salt+2, 30)) / 10
			pitch = float64(75 + HitEffectRandom(effect, salt+3, 30))
			alpha = 255
		}
		placement := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+4, 360)))
		yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+5, 360)))
		center.X += radius * math.Sin(placement) / 5
		center.Z -= radius * math.Cos(placement) / 5
		center.Y -= 6
		n := float64(min(age+1, 21))
		if age >= 30 {
			alpha = uint8(float64(alpha) * float64(39-age) / 10)
		}
		appendEffectSpike(&vertices, &indices, texture, center, size, height, scene.DegreesToRadians(pitch), yaw,
			3*n-.15*n*(n+1)/2, color.RGBA{255, 255, 255, alpha})
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
	}
}

func mageBoltFlight(ctx client.Context, effect Instance, origin scene.Point3, index int) (start, direction scene.Point3, landing int) {
	start = scene.Point3{X: float64(25+HitEffectRandom(effect, index*4, 10)) / 5, Y: 12, Z: float64(15+HitEffectRandom(effect, index*4+1, 10)) / 5}
	end := scene.Point3{X: float64(HitEffectRandom(effect, index*4+2, 4)-2) / 5, Z: float64(HitEffectRandom(effect, index*4+3, 4)-2) / 5}
	direction = scene.Normalize3(scene.Sub3(end, start))
	for landing = 0; landing < 70; landing++ {
		p := scene.Add3(origin, scene.Add3(start, scene.Mul3(direction, .55*float64(landing+1))))
		if p.Y < scene.TerrainHeightAtRenderPoint(ctx.World, p.X, p.Z) {
			break
		}
	}
	return
}

func (m *System) drawMageBolt(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	fire := effect.EffectID == FireBolt || effect.EffectID == FireArrow
	count := max(1, effect.HitCount)
	for i := max(0, (frame-82)/10); i < min(count, (frame-12)/10+1); i++ {
		age := frame - 12 - i*10
		if age < 0 || age >= 70 {
			continue
		}
		start, direction, landing := mageBoltFlight(ctx, effect, origin, i)
		center := scene.Add3(origin, scene.Add3(start, scene.Mul3(direction, .55*float64(min(age, landing)+1))))
		alpha := 255.0
		if age > landing {
			alpha *= scene.ClampFloat(float64(landing+25-age)/20, 0, 1)
		}
		texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[(age+1)%len(component.TextureFiles)])
		if texture != nil && alpha > 0 {
			width, length := 3.8/5, 11.5/5
			if fire {
				width, length = 3.5/5, 14.0/5
			}
			along := scene.Mul3(direction, length)
			right := scene.Mul3(scene.Normalize3(scene.Cross3(direction, scene.Point3{Y: 1})), width)
			up := scene.Mul3(scene.Normalize3(scene.Cross3(direction, right)), width)
			vertices := make([]render.Vertex3D, 0, 8)
			indices := make([]uint16, 0, 12)
			for _, side := range [2]scene.Point3{right, up} {
				appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
					scene.Add3(scene.Sub3(center, along), side), scene.Add3(scene.Add3(center, along), side),
					scene.Sub3(scene.Add3(center, along), side), scene.Sub3(scene.Sub3(center, along), side),
				}, color.RGBA{255, 255, 255, uint8(alpha)})
			}
			screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
		}
		ringAge := age - landing - 1
		if ringAge >= 0 && ringAge < 30 {
			texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
			if texture == nil {
				continue
			}
			n := float64(ringAge + 1)
			radius := (1.2*n - (2.4/70)*n*(n+1)/2) / 5
			vertices, indices := groundEffectRingMesh(texture, origin, max(0, radius-2), radius, effectParticleAlpha(ringAge, 30, 10, 20, 254))
			screen.DrawTriangles3DOwned(vertices, indices, texture, scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat))
		}
	}
	if fire {
		m.drawFireBoltEmbers(screen, ctx, projection, component, effect, origin, frame)
	}
}

func (m *System) drawFireBoltEmbers(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, frame int) {
	for birth := max(4, (frame-28)/4*4); birth <= min(frame, 156); birth += 4 {
		age, salt := frame-birth, 1000+birth*5
		life := 6 + HitEffectRandom(effect, salt, 24)
		if age < 0 || age >= life {
			continue
		}
		position, pitch, yaw := origin, 0.0, 0.0
		if birth > 12 {
			index := min((birth-13)/10, max(1, effect.HitCount)-1)
			start, direction, landing := mageBoltFlight(ctx, effect, origin, index)
			tick := min(birth-12-index*10, landing+1)
			position = scene.Add3(origin, scene.Add3(start, scene.Mul3(direction, .55*float64(tick))))
			pitch, yaw = math.Asin(-direction.Y), math.Atan2(-direction.X, -direction.Z)
		}
		pitch += scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+1, 20) - 10))
		yaw += scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+2, 40) - 20))
		speed := float64(60+HitEffectRandom(effect, salt+3, 90)) / 100
		n := float64(age + 1)
		distance := (speed*n - (speed/float64(life)/1.5)*n*(n+1)/2) / 5
		delta := rotateModelPointY(rotateModelPointX(scene.Point3{Z: distance}, pitch), yaw)
		delta.Y = -delta.Y
		position = scene.Add3(position, delta)
		scale := float64(20+HitEffectRandom(effect, salt+4, 30)) / 100 * (1 - n/float64(life))
		alpha := uint8(254 * max(0, 1-n/float64(min(life, 160-birth))))
		m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, position, age/4+1, scale, 0, color.RGBA{255, 255, 255, alpha})
	}
}

func (m *System) drawSightEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	for birth := max(0, (frame-19+1)/2*2); birth <= frame; birth += 2 {
		age := frame - birth
		n := float64(age + 1)
		angle := scene.DegreesToRadians(float64(birth) * -5)
		p := scene.Add3(origin, scene.Point3{X: 3 * math.Sin(angle), Z: -3 * math.Cos(angle)})
		size, alpha, motion := 2.5-.1*n, 150-3*n, age/3+1
		if effect.EffectID == SightBlaster {
			alpha = max(0, 50-3*n)
		}
		p.Y += 4
		if component.ShadowTexture {
			size, alpha, motion = 1.5-.06*n, 120-n, age/2+1
			p.Y = scene.TerrainHeightAtRenderPoint(ctx.World, p.X, p.Z) + scene.ActorShadowTerrainLift
		}
		m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, p, motion, size, 0,
			color.RGBA{255, 255, 255, uint8(alpha)})
	}
}
