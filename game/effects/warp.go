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

// Warp portals (JT_WARPNPC) launch EF_WARPZONE2 (CNpc::SetState), drawn by
// CRagEffect::WarpZone2: a filled ground disc, two PP_3DCASTING_4 funnels of
// four rings and particle1 sprites orbiting around the portal. Every part
// depends only on the effect's age, so it is computed per frame. Distances
// are in client units, five per Goro unit; client Y points down.

// warpFunnelRing is one m_GI entry of a PP_3DCASTING_4 primitive.
type warpFunnelRing struct {
	distance float64
	rotStart int
}

type warpFunnel struct {
	launch int // effect frame (m_stateCnt) that launches the primitive
	rgb    color.RGBA
	rings  [4]warpFunnelRing
}

var warpFunnels = [2]warpFunnel{
	// m_size 11 and 4 select these colours in RenderTeiRect.
	{launch: 1, rgb: color.RGBA{R: 170, G: 170, B: 255}, rings: [4]warpFunnelRing{{2.5, 270}, {5, 0}, {7.5, 90}, {10, 180}}},
	{launch: 2, rgb: color.RGBA{R: 100, G: 100, B: 255}, rings: [4]warpFunnelRing{{2.7, 271}, {5.2, 1}, {7.7, 91}, {10.2, 181}}},
}

const (
	warpFunnelShrink    = 0.05 // distance lost per frame (Prim3DCasting4)
	warpFunnelRestart   = 10.0 // distance after reaching the centre
	warpFunnelMaxAlpha  = 70
	warpDiscRadius      = 15.0
	warpDiscLift        = 1.5
	warpDiscMaxAlpha    = 128.0
	warpDiscFadeFrames  = 10.0
	warpParticleEvery   = 10
	warpParticleLife    = 70
	warpParticleFadeOut = warpParticleLife - warpParticleLife/3
	warpParticleSize    = 0.7
)

func (m *System) drawWarpZone2(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if screen == nil || frame < 0 {
		return
	}
	if len(component.TextureFiles) > 0 {
		if disc := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0]); disc != nil {
			drawWarpDisc(screen, disc, origin, frame)
		}
	}
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFile); texture != nil {
		var vertices []render.Vertex3D
		var indices []uint16
		for _, funnel := range warpFunnels {
			age := frame - funnel.launch
			if age < 0 {
				continue
			}
			for _, ring := range funnel.rings {
				distance, alpha := warpFunnelRingState(ring.distance, age)
				if alpha <= 0 {
					continue
				}
				tint := funnel.rgb
				tint.A = uint8(alpha)
				appendWarpFunnelRing(&vertices, &indices, texture, ring, origin, distance, tint)
			}
		}
		if len(vertices) > 0 {
			options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
			options.Blend = render.BlendLighter
			screen.DrawTriangles3DOwned(vertices, indices, texture, options)
		}
	}
	if component.SpriteFile != "" {
		m.drawWarpParticles(screen, ctx, projection, component.SpriteFile, effect, origin, frame)
	}
}

// warpFunnelRingState steps Prim3DCasting4 for age frames: the ring shrinks
// 0.05 per frame, restarts at 10 with no alpha once it reaches the centre,
// and its alpha then climbs by one per frame up to 70.
func warpFunnelRingState(start float64, age int) (distance float64, alpha int) {
	steps := age + 1
	// Frames to reach the centre from the start, then every 200 frames.
	first := int(math.Ceil(start/warpFunnelShrink - 1e-9))
	if steps < first {
		return start - float64(steps)*warpFunnelShrink, min(steps, warpFunnelMaxAlpha)
	}
	period := int(math.Round(warpFunnelRestart / warpFunnelShrink))
	since := (steps - first) % period
	return warpFunnelRestart - float64(since)*warpFunnelShrink, min(since+1, warpFunnelMaxAlpha)
}

// appendWarpFunnelRing follows Render3DCasting for a 360-degree ring of
// E_DIVISION points whose height and outward reach both equal its distance.
func appendWarpFunnelRing(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, ring warpFunnelRing, origin scene.Point3, distance float64, tint color.RGBA) {
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	const fullAngle = 360
	step := fullAngle / (level99AuraDivisions - 1)
	// rise_angle is an integer: 90 - distance * 9, truncated.
	rise := scene.DegreesToRadians(float64(int(90 - distance*9)))
	radial, lift := math.Cos(rise)*distance, math.Sin(rise)*distance
	var prevBase, prevTop scene.Point3
	order := 0
	for count := 0; count <= fullAngle; count += step {
		angleDeg := (count + ring.rotStart) % 360
		if order == level99AuraDivisions-1 {
			angleDeg = ring.rotStart % 360
		}
		cosA, sinA := math.Cos(scene.DegreesToRadians(float64(angleDeg))), math.Sin(scene.DegreesToRadians(float64(angleDeg)))
		base := scene.Point3{X: origin.X + cosA*distance/5, Y: origin.Y, Z: origin.Z + sinA*distance/5}
		top := scene.Point3{X: origin.X + cosA*(distance+radial)/5, Y: origin.Y + lift/5, Z: origin.Z + sinA*(distance+radial)/5}
		if order > 0 {
			part := order - 1
			u0 := float32(part) / level99AuraDivisions
			u1 := float32(part+1) / level99AuraDivisions
			first := uint16(len(*vertices))
			*vertices = append(*vertices,
				scene.TexturedSurfaceVertex3D(prevBase, scene.UV{U: u0, V: 1}, tint, w, h),
				scene.TexturedSurfaceVertex3D(base, scene.UV{U: u1, V: 1}, tint, w, h),
				scene.TexturedSurfaceVertex3D(top, scene.UV{U: u1, V: 0}, tint, w, h),
				scene.TexturedSurfaceVertex3D(prevTop, scene.UV{U: u0, V: 0}, tint, w, h),
			)
			*indices = append(*indices, first, first+1, first+2, first, first+2, first+3)
		}
		order = min(order+1, level99AuraDivisions-1)
		prevBase, prevTop = base, top
	}
}

// drawWarpDisc is the PP_3DCIRCLE with PT_FILLCIRCLE (Render3DCircle): a
// horizontal disc 1.5 units above the portal, ten 36-degree segments with
// alpha_down's gradient from the centre outwards, fading in to alpha 128
// over ten frames and alpha blended.
func drawWarpDisc(screen *render.Frame, texture *render.Image, origin scene.Point3, frame int) {
	alpha := math.Min(warpDiscMaxAlpha, warpDiscMaxAlpha/warpDiscFadeFrames*float64(frame+1))
	tint := color.RGBA{R: 255, G: 255, B: 255, A: uint8(alpha)}
	center := scene.Point3{X: origin.X, Y: origin.Y + warpDiscLift/5, Z: origin.Z}
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	var vertices []render.Vertex3D
	var indices []uint16
	u := float32(0)
	for angle := 0; angle < 360; angle += 36 {
		a0, a1 := scene.DegreesToRadians(float64(angle)), scene.DegreesToRadians(float64(angle+36))
		edge0 := scene.Point3{X: center.X + math.Cos(a0)*warpDiscRadius/5, Y: center.Y, Z: center.Z + math.Sin(a0)*warpDiscRadius/5}
		edge1 := scene.Point3{X: center.X + math.Cos(a1)*warpDiscRadius/5, Y: center.Y, Z: center.Z + math.Sin(a1)*warpDiscRadius/5}
		first := uint16(len(vertices))
		vertices = append(vertices,
			scene.TexturedSurfaceVertex3D(center, scene.UV{U: u, V: 1}, tint, w, h),
			scene.TexturedSurfaceVertex3D(center, scene.UV{U: u + .25, V: 1}, tint, w, h),
			scene.TexturedSurfaceVertex3D(edge1, scene.UV{U: u + .25, V: 0}, tint, w, h),
			scene.TexturedSurfaceVertex3D(edge0, scene.UV{U: u, V: 0}, tint, w, h),
		)
		indices = append(indices, first, first+3, first+1, first+1, first+3, first+2)
		u += .25
		if u >= 1 {
			u = 0
		}
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}

// drawWarpParticles draws the PP_3DPARTICLEORBIT sprites launched every ten
// frames. Each lives 70 frames, orbits at 4.5 or 5.25 units (the original
// divides random(200) by 100 as integers) around a centre already offset by
// that radius, turns 1.5 to 2.5 degrees per frame, rises under decelerating
// gravity and fades out over its last third.
func (m *System) drawWarpParticles(screen *render.Frame, ctx client.Context, projection scene.Projection, sprite string, effect Instance, origin scene.Point3, frame int) {
	for spawn := frame - frame%warpParticleEvery; spawn > frame-warpParticleLife && spawn >= 0; spawn -= warpParticleEvery {
		age := frame - spawn
		p, alpha := warpParticleAt(effect, spawn/warpParticleEvery, age)
		if alpha <= 0 {
			continue
		}
		position := scene.Point3{X: origin.X + p.X/5, Y: origin.Y - p.Y/5, Z: origin.Z + p.Z/5}
		// AnimMotion advances every four frames, starting from the second motion.
		m.drawEffectParticleSpriteAction(screen, ctx, projection, sprite, position, 0, age/4+1,
			warpParticleSize, 0, color.RGBA{R: 255, G: 255, B: 255, A: uint8(alpha)})
	}
}

// warpParticleAt returns a particle's client-unit offset and alpha after
// age processing steps.
func warpParticleAt(effect Instance, index, age int) (scene.Point3, float64) {
	salt := 9000 + index*8
	radius := 4.5 + float64(HitEffectRandom(effect, salt, 200)/100)*0.75
	longitude := float64(HitEffectRandom(effect, salt+1, 360))
	speed := 2.5 - float64(HitEffectRandom(effect, salt+3, 11))*0.1
	if HitEffectRandom(effect, salt+2, 2) != 0 {
		speed = -speed
	}
	const gravSpeed = -0.3
	gravAccel := -(gravSpeed / warpParticleLife) / 1.5
	steps := float64(age + 1)
	start := scene.Point3{X: radius * math.Sin(scene.DegreesToRadians(longitude)), Z: radius * math.Cos(scene.DegreesToRadians(longitude))}
	orbit := radius + 0.0005*steps
	angle := scene.DegreesToRadians(longitude + speed*steps)
	p := scene.Point3{
		X: start.X + orbit*math.Sin(angle),
		Y: gravSpeed*steps + gravAccel*steps*(steps+1)/2,
		Z: start.Z + orbit*math.Cos(angle),
	}
	alpha := 254.0
	if age >= warpParticleFadeOut {
		alpha -= 254.0 / (warpParticleLife - warpParticleFadeOut) * float64(age-warpParticleFadeOut+1)
	}
	return p, math.Max(0, alpha)
}
