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

const (
	mapPillarBands          = 4
	mapPillarSegments       = 20
	mapPillarCycleFrames    = 999.0
	mapPillarGrowStartFrame = 200.0
	mapPillarGrowEndFrame   = 290.0
	mapPillarShrinkFrame    = 800.0
	mapPillarWorldScale     = 0.2
	mapPillarBandRadiusStep = 0.5 * mapPillarWorldScale
	mapPillarRiseRadians    = 89.0 * math.Pi / 180.0
)

func (m *System) drawMapPillarEffect(screen *render.Frame, ctx client.Context, component Component, x, y, z, progress float64) {
	texture := m.effectTexture(ctx.Resources, component.TextureName)
	if texture == nil || component.Height <= 0 || component.BottomSize <= 0 {
		return
	}
	tint := effectComponentTint(component, component.AlphaMax)
	frames := math.Floor(scene.ClampFloat(progress, 0, 0.999999)*mapPillarCycleFrames) + 1
	for band := 0; band < mapPillarBands; band++ {
		process := frames + float64(band)*30
		height := MapPillarBandHeight(process, component.Height)
		if height <= 0 {
			continue
		}
		radius := component.BottomSize + float64(band)*mapPillarBandRadiusStep
		rotation := float64(band)*90 + frames
		drawMapPillarBand(screen, texture, x, y, z, radius, height, rotation, tint)
	}
}

func MapPillarBandHeight(process, maxHeight float64) float64 {
	if process < mapPillarGrowStartFrame || maxHeight <= 0 {
		return 0
	}
	if process >= mapPillarShrinkFrame {
		maxHeight -= (process - (mapPillarShrinkFrame - 1)) * mapPillarWorldScale
		if maxHeight <= 0 {
			return 0
		}
	}
	if process <= mapPillarGrowEndFrame {
		return maxHeight * math.Sin((process-mapPillarGrowStartFrame)*math.Pi/180)
	}
	return maxHeight
}

func drawMapPillarBand(screen *render.Frame, texture *render.Image, x, y, z, radius, height, rotationDegrees float64, tint color.RGBA) {
	if screen == nil || texture == nil || radius <= 0 || height <= 0 || tint.A == 0 {
		return
	}
	bounds := texture.Bounds()
	srcW, srcH := float32(bounds.Dx()), float32(bounds.Dy())
	center := scene.Point3{X: x, Y: z, Z: y}
	radialRise := math.Cos(mapPillarRiseRadians) * height
	verticalRise := math.Sin(mapPillarRiseRadians) * height
	vertices := make([]render.Vertex3D, 0, (mapPillarSegments+1)*2)
	indices := make([]uint16, 0, mapPillarSegments*6)
	for i := 0; i <= mapPillarSegments; i++ {
		u := float32(i) / float32(mapPillarSegments+1)
		angle := (rotationDegrees + float64(i)*360/float64(mapPillarSegments)) * math.Pi / 180
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		base := scene.Point3{
			X: center.X + radius*cosine,
			Y: center.Y,
			Z: center.Z + radius*sine,
		}
		top := scene.Point3{
			X: center.X + (radius+radialRise)*cosine,
			Y: center.Y + verticalRise,
			Z: center.Z + (radius+radialRise)*sine,
		}
		vertices = append(vertices,
			scene.TexturedSurfaceVertex3D(base, scene.UV{U: u, V: 1}, tint, srcW, srcH),
			scene.TexturedSurfaceVertex3D(top, scene.UV{U: u, V: 0}, tint, srcW, srcH),
		)
		if i == mapPillarSegments {
			continue
		}
		baseIndex := uint16(i * 2)
		indices = append(indices, baseIndex, baseIndex+1, baseIndex+2, baseIndex+1, baseIndex+3, baseIndex+2)
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, texture, options)
}

type fireflyPath [140]effectParticle

// PrepareEnvironmentEffect captures randomized motion once per emission.
func (m *System) PrepareEnvironmentEffect(effect *Instance, projection scene.Projection) {
	switch effect.EffectID {
	case Firefly:
		effect.firefly = newFireflyPath(*effect)
	case Glow4, MapGhost:
		effect.mapParticles = newMapParticleState(*effect)
	case Rainbow:
		_, _, right, _ := projection.CameraBasis()
		effect.ProjectileDirection = right
	}
}

func ambientSpriteAt(effect Instance, index, frame int) (effectParticle, int, int) {
	p := effectParticle{Position: scene.Point3{Y: 2}, size: 1, Alpha: 1}
	motion, action := frame+1, 0
	switch effect.EffectID {
	case Torch:
		p.size = positiveMapParam(effect.Params[0], 1)
		motion = frame/max(1, int(effect.Params[1])) + 1
	case Banjjakii:
		choice := HitEffectRandom(effect, 0, 10)
		action = choice / 2
		if choice%3 == 1 {
			p.size = .01
		}
		motion = frame/max(1, int(effect.Params[1])) + 1
	case Firefly:
		path := effect.firefly
		if path == nil {
			path = newFireflyPath(effect)
		}
		if frame >= len(path) {
			return effectParticle{}, 0, 0
		}
		p, motion = path[frame], frame/4+1
	default: // Smoke and DragonSmoke share the same puff motion.
		pitch := scene.DegreesToRadians(float64(75 + HitEffectRandom(effect, index*3+1, 30)))
		yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, index*3+2, 360)))
		distance := float64(3+HitEffectRandom(effect, index*3+3, 3)) / 10 * float64(frame+1) / 5
		p.Position = scene.Point3{X: math.Cos(pitch) * math.Sin(yaw) * distance, Y: 1.8 + math.Sin(pitch)*distance, Z: math.Cos(pitch) * math.Cos(yaw) * distance}
		p.size, p.Angle = 1.5, scene.DegreesToRadians(float64(frame+1)*.75)
		p.Alpha = float64(effectParticleAlpha(frame, 50, 0, 34, 254)) / 255
		motion = frame/4 + 1
	}
	return p, action, motion
}

func (m *System) drawAmbientSprite(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, origin scene.Point3, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if frame < 0 {
		return
	}
	count := 1
	if effect.EffectID == Smoke {
		count += HitEffectRandom(effect, 100, 4)
	}
	for i := 0; i < count; i++ {
		p, action, motion := ambientSpriteAt(effect, i, frame)
		m.drawEffectParticleSpriteAction(screen, ctx, projection, c.SpriteFile, scene.Add3(origin, p.Position), action, motion,
			p.size, p.Angle, color.RGBA{255, 255, 255, uint8(scene.ClampFloat(p.Alpha*255, 0, 255))})
	}
}

func newFireflyPath(effect Instance) *fireflyPath {
	path := new(fireflyPath)
	salt := 0
	random := func(count int) int { salt++; return HitEffectRandom(effect, salt, max(1, count)) }
	speed := float64(15+random(35)) / 100
	yaw, pitch := float64(random(360)), float64(random(360))
	direction := func() scene.Point3 {
		y, p := scene.DegreesToRadians(yaw), scene.DegreesToRadians(pitch)
		// Native Y rotation followed by X rotation; Y is inverted in Goro.
		return scene.Point3{X: math.Sin(y), Y: math.Sin(p) * math.Cos(y), Z: math.Cos(p) * math.Cos(y)}
	}
	position := scene.Add3(scene.Point3{Y: 2}, scene.Mul3(direction(), float64(2+random(13))/5))
	count, size := 5+random(25), float64(30+random(70))/100
	alpha, maximum, alphaSpeed := 0.0, 80.0, 80.0/1.5
	for frame := range path {
		position = scene.Add3(position, scene.Mul3(direction(), speed/5))
		if frame%count == 0 {
			speed = float64(15+random(35)) / 100
			yaw += float64(random(80) - 40)
			pitch += float64(random(80) - 40)
			count = 5 + random(10)
		}
		switch frame {
		case 40:
			alpha, maximum = 120, 200
		case 100:
			alpha, maximum = 0, 80
		}
		if (alphaSpeed < 0 && alpha <= 0) || (alphaSpeed >= 0 && alpha >= maximum) {
			alphaSpeed = -alphaSpeed
		}
		alpha = scene.ClampFloat(alpha+alphaSpeed, 0, maximum)
		path[frame] = effectParticle{Position: position, size: size, Alpha: alpha / 255}
	}
	return path
}

func positiveMapParam(value float32, fallback float64) float64 {
	if value <= 0 || !scene.IsFinite(float64(value)) {
		return fallback
	}
	return float64(value)
}

func mapTextureFrame(effect Instance, frame int) (size, height, angle float64, alpha uint8, texture int) {
	n := frame + 1
	if effect.EffectID == Glow1 || effect.EffectID == Glow2 {
		radius := positiveMapParam(effect.Params[0], 30)
		radius *= 1 + .05*math.Sin(scene.DegreesToRadians(float64(n*2%360)))
		return radius * math.Sqrt2 / 5, 0, scene.DegreesToRadians(float64(HitEffectRandom(effect, 0, 360) + n - 135)), 50, 0
	}
	return positiveMapParam(effect.Params[0], 20) * math.Sqrt2 / 5, 2, 0, 130, n / 6 % 13
}

func (m *System) drawMapTexture(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, origin scene.Point3, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if frame < 0 {
		return
	}
	size, height, angle, alpha, textureIndex := mapTextureFrame(effect, frame)
	file := c.TextureFile
	if len(c.TextureFiles) > 0 {
		file = c.TextureFiles[textureIndex]
	}
	texture := m.EffectFileTexture(ctx.Resources, file)
	// The native texture renderer draws the white, grey and dark passes.
	for _, value := range []uint8{255, 150, 50} {
		DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, origin.X, origin.Z, origin.Y+height,
			size, size, angle, color.RGBA{value, value, value, alpha}, TexturedEffectBillboardDrawOptions(true, false))
	}
}

var rainbowColors = [...]color.RGBA{{255, 0, 0, 255}, {255, 126, 0, 255}, {255, 255, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {115, 50, 200, 255}, {255, 0, 255, 255}}

func rainbowPoint(origin, axis scene.Point3, angle float64, band int) scene.Point3 {
	radius := 10 * (1 - float64(band)*.05)
	p := scene.Add3(origin, scene.Mul3(axis, math.Cos(angle)*radius))
	p.Y -= math.Sin(angle) * radius * 2
	return p
}

func (m *System) drawRainbow(screen *render.Frame, ctx client.Context, c Component, effect Instance, origin scene.Point3, now time.Time) {
	frame := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	if frame <= 0 || frame > 300 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, c.TextureFile)
	if texture == nil {
		return
	}
	axis := effect.ProjectileDirection
	if axis == (scene.Point3{}) {
		axis = scene.Point3{X: 1}
	}
	alpha := min(frame, 360-frame)
	vertices, indices := make([]render.Vertex3D, 0, 7*21*4), make([]uint16, 0, 7*21*6)
	for band, tint := range rainbowColors {
		tint.A = uint8(alpha)
		for degrees := 0; degrees <= 180; degrees += 9 {
			base := len(vertices)
			a, b := scene.DegreesToRadians(float64(degrees+frame)), scene.DegreesToRadians(float64(degrees+frame+9))
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				rainbowPoint(origin, axis, b, band), rainbowPoint(origin, axis, b, band+1), rainbowPoint(origin, axis, a, band+1), rainbowPoint(origin, axis, a, band),
			}, tint)
			// The native strip uses successive slices of the texture across the arc.
			v0, v1 := float32(degrees/9)/21, float32(degrees/9+1)/21
			for i, uv := range [4]scene.UV{{U: 1, V: v0}, {U: 1, V: v1}, {U: 0, V: v1}, {U: 0, V: v0}} {
				vertices[base+i].SrcX = uv.U * float32(texture.Bounds().Dx())
				vertices[base+i].SrcY = uv.V * float32(texture.Bounds().Dy())
			}
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}

func bubbleDropAt(effect Instance, frame int) effectParticle {
	n := float64(frame + 1)
	if n <= 0 {
		return effectParticle{}
	}
	height := positiveMapParam(effect.Params[0], 50)
	if height <= 1 {
		height = 50
	}
	stop := math.Trunc(height)
	fall := math.Min(n, math.Max(0, stop-1))
	alpha := math.Min(n, 10) * 20
	if n >= stop {
		alpha -= 10 * (n - stop + 1)
	}
	return effectParticle{Position: scene.Point3{Y: (height - fall) / 5}, size: math.Sqrt2 / 5,
		Angle: scene.DegreesToRadians(float64(HitEffectRandom(effect, 0, 360)) - 15*fall - 135), Alpha: math.Max(0, alpha) / 255}
}

func (m *System) drawBubbleDrop(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, origin scene.Point3, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	texture := m.EffectFileTexture(ctx.Resources, c.TextureFile)
	for trail, loss := range []float64{0, 100, 125, 150} {
		p := bubbleDropAt(effect, frame-trail)
		alpha := uint8(scene.ClampFloat(p.Alpha*255-loss, 0, 255))
		DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, origin.X, origin.Z, origin.Y+p.Position.Y,
			p.size, p.size, p.Angle, color.RGBA{130, 130, 250, alpha}, TexturedEffectBillboardDrawOptions(true, false))
	}
}

type mapParticleGroup struct {
	angles   [4]int // Current/target angle for X and Z wandering.
	orbits   [4]int
	position scene.Point3 // Native units, with Y down.
}

type mapParticleState struct {
	frame, randomIndex int
	groups             []mapParticleGroup
}

func newMapParticleState(effect Instance) *mapParticleState {
	count := 1
	if effect.EffectID == Glow4 {
		count = 40
	}
	s := &mapParticleState{frame: -1, groups: make([]mapParticleGroup, count)}
	for i := range s.groups {
		g := &s.groups[i]
		for j := range g.angles {
			g.angles[j] = s.random(effect, 360)
		}
		for j := range g.orbits {
			g.orbits[j] = s.random(effect, 360)
		}
		if effect.EffectID == MapGhost {
			g.position = scene.Point3{X: float64(s.random(effect, 15) - 8), Y: float64(s.random(effect, 100)), Z: float64(s.random(effect, 15) - 8)}
		}
	}
	return s
}

func (s *mapParticleState) random(effect Instance, n int) int {
	s.randomIndex++
	return HitEffectRandom(effect, s.randomIndex, max(1, n))
}

func (s *mapParticleState) advance(effect Instance, frame int) {
	// Discard suspended time instead of simulating minutes of invisible particles.
	if frame-s.frame > 600 {
		s.frame = frame - 600
	}
	for s.frame < frame {
		s.frame++
		for i := range s.groups {
			g := &s.groups[i]
			up, down := 1, 1
			if effect.EffectID == MapGhost {
				up, down = 3, 2
			}
			for axis := 0; axis < 4; axis += 2 {
				angle, target := g.angles[axis], g.angles[axis+1]
				if target > angle {
					angle = min(359, angle+up)
					if angle > target {
						target = s.random(effect, angle)
					}
				} else {
					angle = max(0, angle-down)
					if angle <= target {
						target = 359 - s.random(effect, angle)
					}
				}
				g.angles[axis], g.angles[axis+1] = angle, target
			}
			if effect.EffectID == MapGhost {
				if g.position.Y < 0 {
					g.position.X += .15 * math.Sin(scene.DegreesToRadians(float64(g.angles[0])))
					g.position.Z += .15 * math.Sin(scene.DegreesToRadians(float64(g.angles[2])))
					if g.position.Y < -50 {
						g.position = scene.Point3{X: float64(s.random(effect, 15) - 8), Y: float64(s.random(effect, 100)), Z: float64(s.random(effect, 15) - 8)}
					}
				}
				g.position.Y -= .6
			} else {
				for j := range g.orbits {
					g.orbits[j] = (g.orbits[j] + 359) % 360
				}
			}
		}
	}
}

func (m *System) drawMapGhost(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, origin scene.Point3, now time.Time) {
	s := effect.mapParticles
	if s == nil {
		return
	}
	s.advance(effect, int(now.Sub(effect.Starts)/db.EffectFrameDuration))
	p := s.groups[0].position
	size := 3.2 * math.Sqrt2 / 5
	_, up, _, ok := projection.BillboardBasis(origin.X, origin.Z, origin.Y)
	if !ok {
		return
	}
	cornerY := p.Y - up.Y*3.2/math.Sqrt2
	if cornerY > 0 {
		return
	}
	alpha := uint8(scene.ClampFloat(250+math.Min(0, cornerY+40)*25, 0, 250))
	texture := m.EffectFileTexture(ctx.Resources, c.TextureFile)
	DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, origin.X+p.X/5, origin.Z+p.Z/5, origin.Y-p.Y/5,
		size, size, 0, color.RGBA{155, 155, 155, alpha}, TexturedEffectBillboardDrawOptions(true, false))
}

func (m *System) drawMapGlow(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, origin scene.Point3, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if frame < 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, c.TextureFile)
	if texture != nil {
		corners := [4]scene.Point3{}
		for i := range corners {
			angle := scene.DegreesToRadians(float64(frame + 1 + i*90))
			corners[i] = scene.Add3(origin, scene.Point3{X: math.Cos(angle) * 5.4, Y: .1, Z: math.Sin(angle) * 5.4})
		}
		vertices, indices := []render.Vertex3D{}, []uint16{}
		appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{105, 225, 105, 250})
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
	s := effect.mapParticles
	if s == nil || len(c.TextureFiles) == 0 {
		return
	}
	s.advance(effect, frame)
	texture = m.EffectFileTexture(ctx.Resources, c.TextureFiles[0])
	_, up, _, ok := projection.BillboardBasis(origin.X, origin.Z, origin.Y)
	if !ok {
		return
	}
	for i, g := range s.groups {
		height := float64((i/4 + 1) * 4)
		alpha := uint8(scene.ClampFloat(130+(-height-up.Y/math.Sqrt2+1)*4, 0, 255))
		dx := .5 * math.Sin(scene.DegreesToRadians(float64(g.angles[0])))
		dz := .5 * math.Sin(scene.DegreesToRadians(float64(g.angles[2])))
		for anchor, phase := range g.orbits {
			sx, sz := level99BubbleDriftSigns(anchor)
			angle := scene.DegreesToRadians(float64(phase))
			DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture,
				origin.X+4*math.Sin(angle)+sx*dx, origin.Z+4*math.Cos(angle)+sz*dz, origin.Y+height/5,
				math.Sqrt2/5, math.Sqrt2/5, 0, color.RGBA{155, 255, 155, alpha}, TexturedEffectBillboardDrawOptions(true, false))
		}
	}
}
