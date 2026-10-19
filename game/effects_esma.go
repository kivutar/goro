package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

type esmaSample struct {
	position modelPoint3
	alpha    float64
	spin     float64
}

type esmaHead struct {
	samples [4]esmaSample
	heading float64
	fading  bool
}

type esmaProjectile struct {
	tick  int
	heads [2]esmaHead
}

// PrimSTIN2 turns the two heads in opposite directions at 60 Hz. Retaining
// their small histories avoids bending an old trail when the target moves.
func (p *esmaProjectile) advance(ctx client.Context, effect worldEffect, now time.Time) {
	end := min(int(now.Sub(effect.starts)/db.EffectFrameDuration)+1, 200)
	for p.tick < end {
		p.tick++
		at := effect.starts.Add(time.Duration(p.tick-1) * db.EffectFrameDuration)
		target := effectWorldAnchor(ctx, effect, at)
		_, targetExists, _ := actorForCombatID(ctx, effect.actorID)
		for side := range p.heads {
			head := &p.heads[side]
			turn := 1.0
			if side == 0 {
				turn = -1
			}
			if p.tick == 1 {
				for i := range head.samples {
					head.samples[i].position = effect.actorOrigin
				}
				head.heading = math.Atan2(target.z-effect.actorOrigin.z, target.x-effect.actorOrigin.x)*180/math.Pi - turn*90
			}
			if !targetExists || p.tick > 140 {
				head.fading = true
			}
			sample := &head.samples[0]
			if head.fading && sample.alpha <= 0 {
				continue
			}
			sample.spin = float64(p.tick * 30)
			if p.tick > 1 && p.tick <= 20 {
				sample.alpha = min(sample.alpha+15, 250)
			}
			if p.tick > 5 {
				if head.fading {
					for i := range head.samples {
						head.samples[i].alpha = max(head.samples[i].alpha-5, 0)
					}
					head.heading += turn
				} else {
					toward := math.Atan2(target.z-sample.position.z, target.x-sample.position.x) * 180 / math.Pi
					toward = math.Mod(toward+360, 360)
					if math.Abs(head.heading-toward) > 3 {
						head.heading += turn * 5
					}
				}
			}
			head.heading = math.Mod(head.heading+360, 360)
			sin, cos := math.Sincos(degreesToRadians(head.heading))
			sample.position.x += .4 * cos
			sample.position.z += .4 * sin
			if p.tick%3 == 0 {
				for i := 3; i > 0; i-- {
					head.samples[i] = head.samples[i-1]
					head.samples[i].alpha *= .8
				}
			}
		}
	}
}

func (m *WorldMode) drawEsmaEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	if now.Before(effect.starts) || effect.esma == nil {
		return
	}
	effect.esma.advance(ctx, effect, now)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for _, head := range effect.esma.heads {
		for _, sample := range head.samples {
			if sample.alpha <= 0 {
				continue
			}
			var corners [4]modelPoint3
			for i := range corners {
				sin, cos := math.Sincos(degreesToRadians(sample.spin + float64((i+1)*90)))
				corners[i] = add3(sample.position, modelPoint3{x: 5 * cos, z: 5 * sin})
			}
			appendSoulLinkerQuad(&vertices, &indices, texture, corners, effectComponentTint(component, sample.alpha/255))
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func (m *WorldMode) drawEsmaHitTexture(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := 8 * min(age-40, 82-age)
	if now.Before(effect.starts) || alpha <= 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	size := math.Sqrt2 * (10 + .7*float64(age-40)) / 5
	size *= 1 + .05*math.Sin(degreesToRadians(float64(age*2)))
	// RenderEffectTexture's hit branch falls through the white and grey passes.
	for _, tint := range [...]color.RGBA{{130, 130, 250, uint8(alpha)}, {255, 255, 255, uint8(alpha)}, {150, 150, 150, uint8(alpha)}, {50, 50, 50, uint8(alpha)}} {
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, x, y, effect.actorOrigin.y+2, size, size, 0, tint, true)
	}
}
