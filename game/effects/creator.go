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

type acidBottleFlight struct {
	origins [2]scene.Point3
	count   int
}

func (flight *acidBottleFlight) advance(ctx client.Context, effect Instance, spec Spec, now time.Time) {
	for flight.count < len(flight.origins) {
		at := effect.Starts.Add(spec.Components[flight.count].Delay)
		if now.Before(at) {
			break
		}
		flight.origins[flight.count] = EffectWorldAnchor(ctx, effect, at)
		flight.count++
	}
}

func (m *System) drawSlimPitcher(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	origin := EffectWorldAnchor(ctx, effect, now)
	variant := effect.EffectID - Slim
	colors := [...]color.RGBA{{255, 135, 135, 255}, {255, 255, 135, 255}, {255, 255, 255, 255}}
	if texture := m.EffectFileTextureWithBlackKey(ctx.Resources, component.TextureFile, false); texture != nil {
		for trail := 0; trail < 4; trail++ {
			process := n - trail
			if process <= 0 {
				continue
			}
			alpha := min(process, 10)*20 - max(0, process-32)*5
			if trail > 0 {
				alpha -= 100 + 25*(trail-1)
			}
			if alpha <= 0 {
				continue
			}
			p := origin
			p.Y += 16 - .6*float64(min(process, 24))
			angle := deterministicAngle(effect, 0) - scene.DegreesToRadians(float64(15*min(process, 24)))
			tint := colors[variant]
			tint.A = uint8(alpha)
			size := 8 * math.Sqrt2 / 5
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, p.X, p.Z, p.Y, -size, size, angle-3*math.Pi/4, tint, false)
		}
	}
	age := n - 16
	alpha := min(age, 10)*2 - max(0, age-10)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	if age <= 0 || alpha <= 0 || texture == nil {
		return
	}
	origin.Y++
	radius := 2.7 * math.Pow(1.07, float64(age))
	tint := [...]color.RGBA{{255, 0, 0, uint8(alpha)}, {255, 255, 0, uint8(alpha)}, {255, 255, 255, uint8(alpha)}}[variant]
	var vertices []render.Vertex3D
	var indices []uint16
	layers := 1
	if variant == 0 {
		layers = 2
	}
	for i := 0; i < layers; i++ {
		appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
			scene.Add3(origin, scene.Point3{X: radius}), scene.Add3(origin, scene.Point3{Z: radius}),
			scene.Add3(origin, scene.Point3{X: -radius}), scene.Add3(origin, scene.Point3{Z: -radius}),
		}, tint)
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}

func (m *System) drawAcidDemon(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := min(80, n*5) - 15*max(0, n-39)
	if n <= 0 || alpha <= 0 {
		return
	}
	texture := m.EffectFileTextureWithBlackKey(ctx.Resources, component.TextureFile, false)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for layer, r := range [...]struct{ radius, height, rise, rotation float64 }{
		{2.4, 25, 70, 0}, {2.7, 22, 57, 90}, {3, 19, 45, 180}, {2.2, 30, 90, 0},
	} {
		// Render3DCasting divides 315 degrees by 20 using integer arithmetic.
		growth, step := 1.005, 15
		if layer == 3 {
			growth, step = 1.025, 18
		}
		for tick := 0; tick < n; tick++ {
			if layer == 3 {
				r.rotation += float64(2 + HitEffectRandom(effect, tick*4+layer, 2))
			} else {
				r.rotation += float64(4 + HitEffectRandom(effect, tick*4+layer, 3))
			}
		}
		r.height *= math.Pow(growth, float64(n)) * math.Sin(scene.DegreesToRadians(float64(n)))
		riseSin, riseCos := math.Sincos(scene.DegreesToRadians(r.rise))
		base := uint16(len(vertices))
		for segment := 0; segment <= 20; segment++ {
			sin, cos := math.Sincos(scene.DegreesToRadians(r.rotation + float64(segment*step)))
			height := r.height * math.Sin(scene.DegreesToRadians(float64(segment*9))) / 5
			bottom := scene.Add3(effect.ActorOrigin, scene.Point3{X: cos * r.radius / 5, Z: sin * r.radius / 5})
			top := scene.Add3(bottom, scene.Point3{X: cos * riseCos * height, Y: riseSin * height, Z: sin * riseCos * height})
			u := float32(segment) / 21
			tint := color.RGBA{255, 50, 255, uint8(alpha)}
			w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
			vertices = append(vertices, scene.TexturedSurfaceVertex3D(bottom, scene.UV{U: u, V: 1}, tint, w, h), scene.TexturedSurfaceVertex3D(top, scene.UV{U: u}, tint, w, h))
			if segment < 20 {
				b := base + uint16(segment*2)
				indices = append(indices, b, b+1, b+2, b+1, b+3, b+2)
			}
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
}
