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

func isAlchemistBottle(effectID int) bool {
	return effectID == ThrowItem || effectID == ThrowItem2 || effectID == ThrowItem4 || effectID == ThrowItem5
}

func (m *System) drawChemicalProtection(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := max(0, 18*min(n, 10)-15*max(0, n-20))
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if alpha == 0 || texture == nil {
		return
	}
	center := effect.ActorOrigin
	// The rays capture the recipient's position before the parent effect's lift.
	right, up, _, ok := projection.BillboardBasis(center.X, center.Z, center.Y)
	if !ok {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 8; i++ {
		angle := (i%4)*90 + 30 + (i/4)*30
		for layer := 0; layer < 2; layer++ {
			spread := 3 + layer*2
			c := color.RGBA{255, 255, 255, uint8(alpha / 2)}
			if layer == 1 {
				c = color.RGBA{130, 130, 255, uint8(alpha)}
			}
			edge := func(degrees int) scene.Point3 {
				sin, cos := math.Sincos(scene.DegreesToRadians(float64(degrees)))
				return scene.Add3(center, scene.Add3(scene.Mul3(right, 20*cos), scene.Mul3(up, 20*sin)))
			}
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{center, edge(angle + spread), edge(angle - spread), center}, c)
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}

type twilightParticle struct {
	Position scene.Point3
	steering [4]int
	angle    int
	phase    int
	radius   float64
	texture  int
}

type twilightParticles struct {
	tick      int
	Particles [320]twilightParticle
}

func NewTwilightParticles(ctx client.Context, effect Instance) *twilightParticles {
	state := &twilightParticles{}
	// Both originals scatter the ingredients around the local viewer, even
	// when another player's cast launches the effect.
	center := EffectWorldAnchor(ctx, Instance{ActorID: scene.LocalSkillTarget(ctx)}, effect.Starts)
	for i := range state.Particles {
		p := &state.Particles[i]
		salt := i * 12
		p.Position.X = center.X + float64(HitEffectRandom(effect, salt, 160)-80)/5
		p.Position.Z = center.Z + float64(HitEffectRandom(effect, salt+1, 160)-80)/5
		p.Position.Y = scene.TerrainHeightAt(ctx.World, p.Position.X-.5, p.Position.Z-.5) + float64(5+HitEffectRandom(effect, salt+2, 66))/5
		for j := range p.steering {
			p.steering[j] = HitEffectRandom(effect, salt+3+j, 360)
		}
		p.angle, p.phase = HitEffectRandom(effect, salt+7, 360), HitEffectRandom(effect, salt+8, 360)
		p.radius = (3 + .1*float64(HitEffectRandom(effect, salt+9, 11))) / 5
		if effect.EffectID == Twilight3 {
			p.texture = HitEffectRandom(effect, (i/4)*48+10, 3)
		}
	}
	return state
}

func (state *twilightParticles) Advance(effect Instance, frame int) {
	for state.tick <= frame {
		state.tick++
		for i := range state.Particles {
			p := &state.Particles[i]
			for j := 0; j < 4; j += 2 {
				angle, target := &p.steering[j], &p.steering[j+1]
				salt := 4000 + state.tick*1280 + i*4 + j
				if *target > *angle {
					*angle = min(*angle+1, 359)
					if *angle > *target {
						*target = HitEffectRandom(effect, salt, max(1, *angle))
					}
				} else {
					*angle = max(*angle-1, 0)
					if *angle <= *target {
						*target = 359 - HitEffectRandom(effect, salt, max(1, *angle))
					}
				}
			}
			p.Position.X += .02 * math.Sin(scene.DegreesToRadians(float64(p.steering[0])))
			p.Position.Z += .02 * math.Sin(scene.DegreesToRadians(float64(p.steering[2])))
			p.angle, p.phase = (p.angle+1)%360, (p.phase+1)%360
		}
	}
}

func (m *System) drawTwilight(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if effect.Twilight == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	duration := int(effect.Expires.Sub(effect.Starts) / db.EffectFrameDuration)
	n := frame + 1
	alpha := max(0, 5*min(n, 49)-5*max(0, n-(duration-50)))
	if alpha == 0 {
		return
	}
	effect.Twilight.Advance(effect, frame)
	for index, file := range component.TextureFiles {
		texture := m.EffectFileTexture(ctx.Resources, file)
		if texture == nil {
			continue
		}
		var vertices []render.Vertex3D
		var indices []uint16
		for _, p := range effect.Twilight.Particles {
			if p.texture != index {
				continue
			}
			right, up, _, ok := projection.BillboardBasis(p.Position.X, p.Position.Z, p.Position.Y)
			if !ok {
				continue
			}
			radius := p.radius * (1 + .05*math.Sin(scene.DegreesToRadians(float64(p.phase))))
			var corners [4]scene.Point3
			for j := range corners {
				sin, cos := math.Sincos(scene.DegreesToRadians(float64(p.angle + 180 + j*90)))
				corners[j] = scene.Add3(p.Position, scene.Add3(scene.Mul3(right, radius*cos), scene.Mul3(up, radius*sin)))
			}
			appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{255, 255, 255, uint8(alpha)})
		}
		if len(indices) > 0 {
			screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
		}
	}
}
