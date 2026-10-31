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

type castingRing struct {
	process, alpha, angle int
	radius                float64
}

type CastingState struct {
	Tick      int
	rings     [8]castingRing
	auraBlade bool
}

// BlueCasting launches two four-layer SAINTCASTING rings. PrimGI1 makes
// each ring flare and reset until the final half-second of the cast.
func (p *CastingState) Advance(frame, duration int) {
	if p.Tick == 0 {
		for i := range p.rings {
			r := &p.rings[i]
			r.radius, r.angle = 4.1, 80
			r.alpha = 45 - (i/4)*20 + (3-i%4)*45
			if duration <= 60 {
				r.alpha, r.process = 0, -(i%4)*5
			}
		}
	}
	for p.Tick < min(frame+1, duration) {
		p.Tick++
		for i := range p.rings {
			r := &p.rings[i]
			r.process++
			if r.process <= 0 {
				continue
			}
			r.radius += .07
			r.angle--
			if r.angle < 10 {
				r.angle, r.alpha = 10, 0
			}
			if r.radius >= 4 {
				r.alpha = max(r.alpha-3, 0)
				if r.alpha == 0 && r.process < duration-30 {
					r.radius, r.angle = 3.37, 74
					if p.auraBlade {
						r.angle = 64
					}
				}
			} else {
				if p.auraBlade {
					r.alpha += 5
				} else {
					r.alpha += 10
				}
			}
		}
	}
}

func (m *System) DrawBlueCastingEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	if now.Before(effect.Starts) || effect.BlueCasting == nil {
		return
	}
	duration := 60
	if effect.Duration > 0 {
		duration = max(1, int(effect.Duration/db.EffectFrameDuration))
	} else if !effect.Expires.IsZero() {
		duration = max(1, int(effect.Expires.Sub(effect.Starts)/db.EffectFrameDuration))
	}
	effect.BlueCasting.Advance(int(now.Sub(effect.Starts)/db.EffectFrameDuration), duration)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i, r := range effect.BlueCasting.rings {
		if i == 4 && effect.BlueCasting.auraBlade {
			screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
			vertices, indices = nil, nil
			texture = m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
			if texture == nil {
				return
			}
		}
		if r.alpha <= 0 {
			continue
		}
		layer := i % 4
		phase := r.process + layer*90
		if layer >= 2 {
			phase += r.process
		}
		riseSin, riseCos := math.Sincos(scene.DegreesToRadians(float64(r.angle)))
		base := uint16(len(vertices))
		for segment := 0; segment <= 20; segment++ {
			sin, cos := math.Sincos(scene.DegreesToRadians(float64(segment*18 + (layer+2)*90)))
			maxHeight := 20 - layer
			if effect.EffectID == BeginAsura || effect.EffectID == BeginAsura11 {
				maxHeight = 25 - layer
			}
			if effect.BlueCasting.auraBlade {
				maxHeight = 15 - layer
			}
			height := float64(maxHeight) * (1 + .3*math.Sin(scene.DegreesToRadians(float64(segment*9)))*math.Sin(scene.DegreesToRadians(float64(phase))))
			bottom := scene.Point3{X: x + cos*r.radius/5, Y: z, Z: y + sin*r.radius/5}
			top := scene.Add3(bottom, scene.Point3{X: cos * riseCos * height / 5, Y: riseSin * height / 5, Z: sin * riseCos * height / 5})
			u := float32(segment) / 21
			c := color.RGBA{100, 100, 255, uint8(scene.ClampInt(r.alpha, 0, 255))}
			if effect.BlueCasting.auraBlade || effect.EffectID == BeginAsura || effect.EffectID == BeginAsura11 {
				c.R, c.G = 255, 255
			}
			if effect.EffectID == CoupleCasting {
				c.R, c.G, c.B = 255, 89, 182
			}
			w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
			vertices = append(vertices, scene.TexturedSurfaceVertex3D(bottom, scene.UV{U: u, V: 1}, c, w, h), scene.TexturedSurfaceVertex3D(top, scene.UV{U: u, V: 0}, c, w, h))
			if segment < 20 {
				b := base + uint16(segment*2)
				indices = append(indices, b, b+1, b+2, b+1, b+3, b+2)
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}

func (m *System) drawCastRingEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, componentIndex int, x, y, z, progress float64) {
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.TextureName)
	tint := effectComponentTint(component, alpha)
	z += component.PosZ
	angleOffset := 0.0
	if component.Rotate {
		angleOffset = progress * 2 * math.Pi
		angleOffset += deterministicAngle(effect, componentIndex+17) * 0.08
	}
	drawWorldSoftRing(screen, m.whitePixel, x, y, z+0.015, component.BottomSize*1.15, 0.16, scene.WithAlpha(component.Color, alpha*0.75), scene.MaxInt(component.CircleSides, 20))
	if texture != nil {
		drawWorldCylinderBandRotated(screen, m.whitePixel, texture, x, y, z+0.035, component.BottomSize, component.TopSize, component.Height, tint, scene.MaxInt(component.CircleSides, component.TotalCircleSides), angleOffset)
		return
	}
	drawWorldCylinderBandRotated(screen, m.whitePixel, nil, x, y, z+0.035, component.BottomSize, component.TopSize, component.Height, tint, scene.MaxInt(component.CircleSides, component.TotalCircleSides), angleOffset)
}

const groundSampleFallbackRotationRadiansPerSecond = 40 * math.Pi / 180

func (m *System) drawGroundPlaneEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, worldX, worldY, progress float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil || ctx.World == nil {
		return
	}
	alpha := effectBillboardAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	size := component.SizeStart
	if effect.Size > 0 {
		size = effect.Size
	}
	if size <= 0 {
		size = 1
	}
	half := size * 0.5
	angle := GroundSampleRotationAngle(effect, now)
	uv := func(u, v float64) scene.UV {
		sinA, cosA := math.Sin(angle), math.Cos(angle)
		x, y := u-0.5, v-0.5
		return scene.UV{
			U: float32(x*cosA - y*sinA + 0.5),
			V: float32(x*sinA + y*cosA + 0.5),
		}
	}
	point := func(dx, dy float64) scene.Point3 {
		x := worldX + dx
		y := worldY + dy
		return scene.Point3{X: x, Y: scene.TerrainHeightAt(ctx.World, x-0.5, y-0.5) + component.PosZ, Z: y}
	}
	verts := [4]scene.Point3{
		point(-half, -half),
		point(half, -half),
		point(half, half),
		point(-half, half),
	}
	uvs := [4]scene.UV{
		uv(0, 0),
		uv(1, 0),
		uv(1, 1),
		uv(0, 1),
	}
	tint := effectComponentTint(component, alpha)
	scene.DrawTexturedSurface3DWithOptions(screen, texture, verts, uvs, scene.QuadIndices012023, [4]color.RGBA{tint, tint, tint, tint}, GroundSampleDrawOptions())
}

func GroundSampleDrawOptions() *render.DrawTrianglesOptions {
	return scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
}

func GroundSampleRotationAngle(effect Instance, now time.Time) float64 {
	elapsed := math.Max(0, now.Sub(effect.Starts).Seconds())
	speed := effect.GroundSampleRotationRadiansPerSecond
	if speed <= 0 {
		speed = groundSampleFallbackRotationRadiansPerSecond
	}
	return elapsed * speed
}

func (m *System) drawLockOnTargetEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, worldX, worldY, worldZ float64, progress float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	size := LockOnTargetSize(effect.Starts, now)
	half := size * 0.5
	angle := float64(now.UnixMilli())*math.Pi/720 + math.Pi/4
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	point := func(dx, dy float64) scene.Point3 {
		x := dx*cosA - dy*sinA
		y := dx*sinA + dy*cosA
		return scene.Point3{X: worldX + x, Y: worldZ + component.PosZ, Z: worldY + y}
	}
	verts := [4]scene.Point3{
		point(-half, -half),
		point(half, -half),
		point(half, half),
		point(-half, half),
	}
	uvs := [4]scene.UV{
		{U: 0, V: 0},
		{U: 1, V: 0},
		{U: 1, V: 1},
		{U: 0, V: 1},
	}
	tint := LockOnTargetTint(effect.Starts, now)
	tint.A = uint8(scene.ClampFloat(alpha, 0, 1) * 255)
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	scene.DrawTexturedSurface3DWithOptions(screen, texture, verts, uvs, scene.QuadIndices012023, [4]color.RGBA{tint, tint, tint, tint}, options)
}

func LockOnTargetSize(starts, now time.Time) float64 {
	elapsed := now.Sub(starts).Seconds() * 1000 / 50
	elapsed = scene.ClampFloat(elapsed, 1, 5)
	return (6 - elapsed) * 3
}

func LockOnTargetTint(starts, now time.Time) color.RGBA {
	elapsed := int(now.Sub(starts) / (20 * time.Millisecond))
	if elapsed < 0 {
		elapsed = 0
	}
	factor := float64(20-(elapsed%20)) / 20
	gb := uint8(scene.ClampFloat(factor, 0, 1) * 255)
	return color.RGBA{R: 255, G: gb, B: gb, A: 255}
}

// VOLCANO / VOLCANO2: four expanding bands share the PrimGI2 animation.
func (m *System) drawGroundCastEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	heightScale, initialRise, fade := 5.0, 80, 2
	if effect.EffectID == Ganbantein {
		heightScale, initialRise, fade = 3, 70, 8
	}
	expansion, rotation, minRise := .3, 5, 40
	if effect.EffectID == Gumgang2 || effect.EffectID == db.SkillEffectSteelBodyRing {
		expansion, rotation, minRise = .1, 3, 30
	}
	fadeIn := 20
	if effect.EffectID == Gumgang3 {
		fadeIn = 10
	}
	alpha := fadeIn*min(n, 10) - fade*max(0, n-10)
	if n <= 0 || alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	rise := scene.DegreesToRadians(float64(max(minRise, initialRise-n)))
	origin := scene.Point3{X: x, Y: z, Z: y}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for i := 0; i < 4; i++ {
		height := heightScale * math.Sin(rise) * (1 + .15*math.Sin(scene.DegreesToRadians(float64(n+i*90))))
		appendEffectBand(&vertices, &indices, texture, origin, (float64(i+1)+expansion*float64(n))/5,
			scene.DegreesToRadians(float64(i*90+n*rotation)), 2*math.Pi, 1/math.Tan(rise), 20,
			color.RGBA{255, 255, 255, uint8(alpha)},
			func(j int) float64 { return height * math.Sin(scene.DegreesToRadians(float64(j*9))) })
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
}
