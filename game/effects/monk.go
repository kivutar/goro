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

func FuryBodyTint(tint color.RGBA, effect Instance, now time.Time) color.RGBA {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	gb := 200.0
	if HitEffectRandom(effect, frame, 14) == 0 {
		gb = 250
	}
	return scene.MultiplyTint(tint, 250.0/255, gb/255, gb/255, 1)
}

func monkRay(effect Instance, index, frame int) ([4]scene.Point3, color.RGBA) {
	salt := index * 3
	age := frame + 1 - 10 - HitEffectRandom(effect, salt, 25)
	alpha := 15*scene.ClampInt(age-10, 0, 10) - 5*max(0, age-20)
	tint := color.RGBA{225, 225, 80, 0}
	if effect.EffectID == TeiHit1X {
		tint = color.RGBA{255, 255, 255, 0}
	}
	var corners [4]scene.Point3
	if alpha <= 0 {
		return corners, tint
	}
	tint.A = uint8(alpha)
	pitch := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+1, 360)))
	yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+2, 360)))
	distance := float64(-10 + age*3)
	// TEIHIT1 uses a .7-unit half-width and a length multiplier of 24.
	for i, p := range [4][2]float64{{16.8, -.7}, {16.8, .7}, {-16.8, .7}, {-16.8, -.7}} {
		corners[i] = scene.Add3(effect.ActorOrigin, effectParticlePoint(distance+p[0], p[1], pitch, yaw))
		corners[i].Y += 1.8
	}
	return corners, tint
}

func (m *System) drawMonkRays(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < component.Duplicate; i++ {
		corners, tint := monkRay(effect, i, frame)
		if tint.A > 0 {
			appendEffectQuad(&vertices, &indices, texture,
				[4]scene.Point3{corners[3], corners[2], corners[1], corners[0]}, tint)
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}

func initTripleAttack(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
	}
	effect.ActorOrigin = EffectWorldAnchor(ctx, *effect, effect.Starts)
	effect.ProjectileDirection = scene.Sub3(destination, effect.ActorOrigin)
	effect.HasActorTransform = true
}

func tripleAttackLine(effect Instance, index, frame int) (from, to scene.Point3, alpha int) {
	salt := index * 6
	age := frame + 1 - index/4 - HitEffectRandom(effect, salt, 20)
	alpha = max(0, 8*scene.ClampInt(age, 0, 5)-4*max(0, age-5))
	angle := math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X)
	s, c := math.Sincos(angle)
	back := .15 * float64(HitEffectRandom(effect, salt+1, 100))
	from = scene.Add3(effect.ActorOrigin, scene.Point3{
		X: (.06*float64(HitEffectRandom(effect, salt+2, 100)) - 3 - back*c) / 5,
		Z: (.06*float64(HitEffectRandom(effect, salt+3, 100)) - 3 - back*s) / 5,
		Y: (8 + .04*float64(HitEffectRandom(effect, salt+4, 100))) / 5,
	})
	length := (20 + .2*float64(HitEffectRandom(effect, salt+5, 100))) / 5
	to = scene.Add3(from, scene.Point3{X: length * c, Z: length * s})
	return
}

func (m *System) drawTripleAttack(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	for layer := 0; layer < 2; layer++ {
		var vertices []render.Vertex3D
		var indices []uint16
		for i := 0; i < 8; i++ {
			from, to, alpha := tripleAttackLine(effect, i, frame)
			if alpha == 0 {
				continue
			}
			tint := color.RGBA{255, 255, uint8(55 + 145*layer), uint8(alpha)}
			appendEffectTube(&vertices, &indices, texture, from, to, float64(layer+1)*.05, 5, tint)
		}
		if len(indices) > 0 {
			screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(layer == 1, false))
		}
	}
}

// HITLINE keeps twenty samples of a bouncing ray, reducing each bounce by 30%.
func hitLinePoint(age int, speed, direction float64) scene.Point3 {
	if age <= 0 {
		return scene.Point3{}
	}
	phase := age * 8
	bounces := (phase - 1) / 180
	phase -= bounces * 180
	radius := math.Pow(.7, float64(bounces))
	distance := speed * (2*(1-radius)/.3 + float64(phase)/90*radius) / 5
	return scene.Point3{
		X: distance * math.Cos(direction), Z: distance * math.Sin(direction),
		Y: speed * math.Sin(scene.DegreesToRadians(float64(phase))) * 1.5 * radius / 5,
	}
}

func (m *System) drawHitLine(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 16; i++ {
		salt := i * 4
		age := frame + 1 - i/4 - HitEffectRandom(effect, salt, 26)
		alpha := 125 + HitEffectRandom(effect, salt+1, 51) - 5*max(0, age-35)
		if age <= 0 || alpha <= 0 {
			continue
		}
		direction := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+2, 360)))
		speed := float64(6 + HitEffectRandom(effect, salt+3, 8))
		for j := 0; j < 20 && alpha-j*5 > 0; j++ {
			// The native shift copies the new head into slot 1 as well.
			from := scene.Add3(effect.ActorOrigin, hitLinePoint(age-max(0, j-1), speed, direction))
			to := scene.Add3(effect.ActorOrigin, hitLinePoint(age-j, speed, direction))
			tint := color.RGBA{255, 200, uint8(10 + j*7), uint8(alpha - j*5)}
			if effect.EffectID == HitLine2 {
				tint.G, tint.B = uint8(60+j*7), uint8(60+j*7)
			}
			appendEffectTube(&vertices, &indices, texture, from, to, .02, 3, tint)
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}

// Heal2's absorption variant has four blue, wavy bands and no rising sparks.
func (m *System) drawAbsorbSpiritsEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 5*min(n, 15) - 2*max(0, n-39)
	if n <= 0 || alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	origin := scene.Point3{X: x, Y: z, Z: y}
	var vertices []render.Vertex3D
	var indices []uint16
	for _, band := range [...]struct{ radius, height, rise, rotation, speed float64 }{
		{4.6, 60, 90, 0, 4},
		{5, 60, 90, 180, 3},
		{5, 18, 50, float64(HitEffectRandom(effect, 0, 360)), 2},
		{5.2, 18, 48, float64(HitEffectRandom(effect, 1, 360)), 2},
	} {
		rise := scene.DegreesToRadians(band.rise)
		height := band.height / 5 * math.Sin(rise) * math.Sin(scene.DegreesToRadians(float64(min(n, 90))))
		appendEffectBand(&vertices, &indices, texture, origin, band.radius/5,
			scene.DegreesToRadians(band.rotation), 2*math.Pi, 1/math.Tan(rise), 20,
			color.RGBA{100, 100, 255, uint8(alpha)}, func(j int) float64 {
				return height * (.75 + .25*math.Sin(scene.DegreesToRadians(float64(j*34)+float64(n)*band.speed)))
			})
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}
