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

type spiritSpheres struct {
	tick   int
	angles [5][7]int
}

func NewSpiritSpheres(effect Instance) *spiritSpheres {
	p := &spiritSpheres{}
	for i := range p.angles {
		for j := range p.angles[i] {
			p.angles[i][j] = HitEffectRandom(effect, i*7+j, 360)
		}
	}
	return p
}

func (p *spiritSpheres) Advance(effect Instance, now time.Time) {
	end := max(0, int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1)
	// Catch up cosmetic wandering for at most two seconds after a long pause.
	p.tick = max(p.tick, end-120)
	for p.tick < end {
		for i := range p.angles {
			a := &p.angles[i]
			for j := 0; j < 6; j += 2 {
				seed := (p.tick*5+i)*7 + j
				if a[j+1] > a[j] {
					a[j] = min(a[j]+1, 359)
					if a[j] > a[j+1] {
						a[j+1] = HitEffectRandom(effect, seed, max(a[j], 1))
					}
				} else {
					a[j] = max(a[j]-1, 0)
					if a[j] <= a[j+1] {
						a[j+1] = 359 - HitEffectRandom(effect, seed, max(a[j], 1))
					}
				}
			}
			a[6] = (a[6] + 5 + HitEffectRandom(effect, (p.tick*5+i)*7+6, 5)) % 360
		}
		p.tick++
	}
}

func (m *System) drawSpiritSphereEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, c.TextureFile)
	p := effect.SpiritSpheres
	if texture == nil || p == nil || now.Before(effect.Starts) {
		return
	}
	p.Advance(effect, now)
	// Native coordinates use five units per map cell.
	orbit, height, jitter, radius := 7.0/5, 15.0/5, 1.0/5, 2.0/5
	outerTint, innerTint := color.RGBA{0, 0, 255, 0}, color.RGBA{255, 255, 0, 0}
	switch effect.EffectID {
	case Chookgi2: // Baby Monk.
		orbit, height, jitter, radius = 5.5/5, 14.0/5, .7/5, 1.3/5
	case Chookgi3:
		jitter, radius = .5/5, 1.0/5
		outerTint, innerTint = color.RGBA{68, 42, 30, 0}, color.RGBA{255, 255, 255, 0}
	}
	outerTint.A, innerTint.A = uint8(min(p.tick, 200)), uint8(min(p.tick, 200))
	count := min(effect.HitCount, 5)
	for i := 0; i < count; i++ {
		a := p.angles[i]
		degrees := i*72 + p.tick
		size := radius
		if effect.EffectID == Chookgi3 {
			degrees = i*360/count + p.tick/2
			if i < effect.HitCount-5 {
				size *= 2
			}
		}
		angle := scene.DegreesToRadians(float64(degrees % 360))
		cx := x + orbit*math.Sin(angle) + jitter*math.Sin(scene.DegreesToRadians(float64(a[0])))
		cy := y + orbit*math.Cos(angle) + jitter*math.Sin(scene.DegreesToRadians(float64(a[2])))
		cz := z + height - jitter*math.Sin(scene.DegreesToRadians(float64(a[4])))
		pulse := size * .05 * math.Sin(scene.DegreesToRadians(float64(a[6])))
		outer, inner := math.Sqrt2*(size+pulse), math.Sqrt2*(size*.5+pulse)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, cx, cy, cz, outer, outer, math.Pi/4, outerTint, true)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, cx, cy, cz, inner, inner, math.Pi/4, innerTint, true)
	}
}
