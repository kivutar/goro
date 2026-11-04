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

type arrowVulcanRays struct {
	origins [9]scene.Point3
	count   int
}

func (rays *arrowVulcanRays) advance(ctx client.Context, effect Instance, now time.Time) {
	for rays.count < len(rays.origins) {
		at := effect.Starts.Add(time.Duration(rays.count) * db.EffectFrameDuration)
		if now.Before(at) {
			break
		}
		rays.origins[rays.count] = EffectWorldAnchor(ctx, effect, at)
		rays.count++
	}
}

func (m *System) RemoveMarionetteLinks(actorID uint32) {
	active := m.Active[:0]
	for _, effect := range m.Active {
		if effect.EffectID != MarionetteLink || effect.ActorID != actorID && effect.TargetID != actorID {
			active = append(active, effect)
		}
	}
	m.Active = active
}

func arrowVulcanLine(effect Instance, index, frame int) (from, to scene.Point3, alpha int, radius float64) {
	if effect.arrowVulcan != nil {
		destination := scene.Add3(effect.ActorOrigin, effect.ProjectileDirection)
		effect.ActorOrigin = effect.arrowVulcan.origins[index/4]
		effect.ProjectileDirection = scene.Sub3(destination, effect.ActorOrigin)
	}
	salt := index * 7
	age := frame + 1 - index/4 - 30 - HitEffectRandom(effect, salt, 110)
	alpha = max(0, 8*scene.ClampInt(age, 0, 5)-4*max(0, age-5))
	angle := math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X)
	s, c := math.Sincos(angle)
	back := .15 * float64(HitEffectRandom(effect, salt+1, 100))
	from = scene.Add3(effect.ActorOrigin, scene.Point3{
		X: (.08*float64(HitEffectRandom(effect, salt+2, 100)) - 4 - back*c) / 5,
		Z: (.08*float64(HitEffectRandom(effect, salt+3, 100)) - 4 - back*s) / 5,
		Y: (4 + .06*float64(HitEffectRandom(effect, salt+4, 100))) / 5,
	})
	length := math.Hypot(effect.ProjectileDirection.X, effect.ProjectileDirection.Z) + .06*float64(HitEffectRandom(effect, salt+5, 100))
	to = scene.Add3(from, scene.Point3{X: length * c, Z: length * s})
	radius = (.02 + .008*float64(HitEffectRandom(effect, salt+6, 11))) / 5
	return
}

func TarotCardAlpha(frame int) int {
	n := frame + 1
	return max(0, min(220, 10*n)-5*max(0, n-199))
}

func (m *System) drawClownGypsyEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	n := frame + 1
	origin := EffectWorldAnchor(ctx, effect, now)
	var vertices []render.Vertex3D
	var indices []uint16
	switch effect.EffectID {
	case TripleAttack3:
		for layer := 0; layer < 2; layer++ {
			vertices, indices = nil, nil
			for i := 0; i < 36; i++ {
				from, to, alpha, radius := arrowVulcanLine(effect, i, frame)
				if alpha > 0 {
					appendEffectTube(&vertices, &indices, texture, from, to, radius*float64(layer+1), 5, color.RGBA{255, 89, 182, uint8(alpha)})
				}
			}
			if len(indices) > 0 {
				screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(layer == 1, false))
			}
		}
		return
	case Moonlit:
		origin.Y += 2.8
		for i := 0; i < 20; i++ {
			salt := i * 4
			halfWidth := (4 + .2*float64(HitEffectRandom(effect, salt, 11))) / 5
			angle := HitEffectRandom(effect, salt+1, 360) + n*(3+HitEffectRandom(effect, salt+2, 4))
			arc := 180 + HitEffectRandom(effect, salt+3, 91)
			step := arc / 20
			tilt := scene.DegreesToRadians(float64(i%4*45 + i/4*9))
			s, c := math.Sincos(tilt)
			edge := scene.Point3{X: -halfWidth * s, Y: -halfWidth * c}
			point := func(a int) scene.Point3 {
				sin, cos := math.Sincos(scene.DegreesToRadians(float64(a)))
				return scene.Add3(origin, scene.Point3{X: 2.8 * cos * c, Y: -2.8 * cos * s, Z: 2.8 * sin})
			}
			for j := 0; j < arc/step; j++ {
				from, to := point(angle+j*step), point(angle+(j+1)*step)
				start := len(vertices)
				appendEffectQuad(&vertices, &indices, texture,
					[4]scene.Point3{scene.Sub3(from, edge), scene.Sub3(to, edge), scene.Add3(to, edge), scene.Add3(from, edge)}, color.RGBA{255, 89, 182, 100})
				for k, u := range []int{min(j, 19), min(j+1, 20), min(j+1, 20), min(j, 19)} {
					vertices[start+k].SrcX = float32(u) / 21 * float32(texture.Bounds().Dx())
				}
			}
		}
	case MarionetteLink:
		if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); !ok {
			return
		}
		target := effect
		target.ActorID = effect.TargetID
		to := EffectWorldAnchor(ctx, target, now)
		origin.Y, to.Y = origin.Y+1.6, to.Y+1.6
		if n < 15 {
			to = scene.Add3(origin, scene.Mul3(scene.Sub3(to, origin), math.Sin(scene.DegreesToRadians(float64(n*6)))))
		}
		phase := scene.DegreesToRadians(float64(n % 360))
		halfStep := math.Pi / 360
		radius := (.7 + .005*math.Sin(phase/2)*math.Sin(phase/2+halfStep)/math.Sin(halfStep)) / 5
		for layer := 1; layer <= 2; layer++ {
			appendEffectTube(&vertices, &indices, texture, origin, to, radius*float64(layer), 10, color.RGBA{255, 89, 182, uint8(min(80, n) * layer)})
		}
	case BottomHermode:
		yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, 0, 360) + n))
		phase := scene.DegreesToRadians(float64(HitEffectRandom(effect, 1, 360) + n))
		radius := (1.44 + .16*math.Sin(phase)) / 5
		var bottom, top [4]scene.Point3
		for i := range bottom {
			s, c := math.Sincos(yaw + float64(i)*math.Pi/2)
			bottom[i] = scene.Add3(origin, scene.Point3{X: radius * c, Y: (12 - 2*math.Sin(yaw)) / 5, Z: radius * s})
			top[i] = scene.Add3(bottom[i], scene.Point3{Y: 1.6 / 5})
		}
		grey := uint8(HitEffectRandom(effect, 2, 31))
		alpha := uint8(min(140, 2*n))
		appendEffectQuad(&vertices, &indices, texture, bottom, color.RGBA{grey, grey, 250, alpha})
		appendEffectQuad(&vertices, &indices, texture, top, color.RGBA{grey, grey, 250, alpha})
		for i, blue := range []uint8{220, 190, 130, 160} {
			j := (i + 1) % 4
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{top[i], top[j], bottom[j], bottom[i]}, color.RGBA{grey, grey, blue, alpha})
		}
	default: // The server selects and positions the Tarot card through ZC_NOTIFY_EFFECT.
		alpha := TarotCardAlpha(frame)
		if alpha == 0 {
			return
		}
		size := 1.6 * (1 + .05*math.Sin(scene.DegreesToRadians(float64(2*n))))
		origin.Y = effect.ActorOrigin.Y + 6
		DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, origin.X, origin.Z, origin.Y,
			size, math.Sqrt(3)*size, 0, color.RGBA{255, 255, 255, uint8(alpha)}, TexturedEffectBillboardDrawOptions(false, false))
		return
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}
