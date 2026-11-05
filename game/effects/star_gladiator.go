package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

func (m *System) drawStarRings(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil || now.Before(effect.Starts) {
		return
	}
	age := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	black := effect.EffectID == BlackDevil
	count := 8
	if black {
		count, age = 4, age-1 // BlackDevil creates its primitive at tick one.
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for ring := 0; ring < count; ring++ {
		radius, tilt, height, alpha := starRingState(black, ring, age)
		if alpha <= 0 {
			continue
		}
		span, rotation := 315, (ring%4)*90
		tint := color.RGBA{100, 100, 255, uint8(alpha)}
		if ring >= 4 {
			rotation = (rotation + 90) % 360
		}
		if black {
			span, rotation = 360, ((ring+3)%4)*90
			tint.R, tint.G, tint.B = 50, 50, 50
		}
		point := func(angle, order int, top bool) scene.Point3 {
			s, c := math.Sincos(scene.DegreesToRadians(float64(angle + rotation)))
			r, h := radius, 0.0
			if top {
				h = height
				if !black {
					h *= math.Sin(scene.DegreesToRadians(float64(min(order, 20) * 9)))
				}
				r += math.Cos(scene.DegreesToRadians(tilt)) * h
				h *= math.Sin(scene.DegreesToRadians(tilt))
			}
			return scene.Point3{X: x + c*r/5, Y: z + h/5, Z: y + s*r/5}
		}
		step := span / 20
		for angle, order := step, 1; angle <= span; angle, order = angle+step, order+1 {
			u := float32(min(order, 20)-1) / 21
			start := len(vertices)
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				point(angle-step, order-1, true), point(angle, order, true),
				point(angle, order, false), point(angle-step, order-1, false),
			}, tint)
			for i := 0; i < 4; i++ {
				tx := u
				if i == 1 || i == 2 {
					tx += 1.0 / 21
				}
				vertices[start+i].SrcX = tx * float32(texture.Bounds().Dx())
			}
		}
	}
	if len(vertices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(!black, false))
	}
}

func starRingState(black bool, ring, age int) (radius, tilt, height float64, alpha int) {
	if age <= 0 {
		return
	}
	if !black {
		radius = [...]float64{4.5, 5, 5.5, 5, 5.5, 6, 6.5, 6}[ring]
		tilt = [...]float64{85, 80, 70, 75, 50, 55, 65, 60}[ring]
		height = ([...]float64{27, 28, 29, 30, 30, 29, 28, 27}[ring] + 2.5*math.Sin(scene.DegreesToRadians(float64(age%90)*4))) * math.Sin(scene.DegreesToRadians(float64(min(age, 90))))
		alpha = max(0, min(age, 19)*5-max(0, age-110))
		return
	}
	// Prim3DCasting5: four rings expand independently, then disappear on the
	// first wrap after tick fifty (at most 150 ticks of simulation).
	radius = float64(ring+1) * 2.5
	for tick := 1; tick <= age; tick++ {
		radius = float64(float32(radius) + .1)
		if radius >= 13 {
			radius, alpha = 3, 0
			if tick > 50 {
				return radius, 0, 0, 0
			}
		}
		height = radius - 3
		if height > 5 {
			height = 10 - height
			alpha = max(0, alpha-2)
		} else {
			alpha += 2
		}
	}
	tilt = float64(int(90 - (radius-3)*5))
	height = max(0, height*2)
	return
}

type StarFlight struct {
	updated       time.Time
	ticks, height int
}

func (m *System) FusionHeight(actor worldstate.Actor, now time.Time, active bool) float64 {
	job := scene.ActorVisualJob(actor)
	flying := job == db.JobStar2 || job == db.JobStar2B
	flight := m.StarFlights[actor.ID]
	if flight == nil {
		if !flying {
			return 0
		}
		flight = &StarFlight{updated: now}
		if m.StarFlights == nil {
			m.StarFlights = make(map[uint32]*StarFlight)
		}
		m.StarFlights[actor.ID] = flight
	}
	ticks := int(now.Sub(flight.updated) / db.EffectFrameDuration)
	flight.updated = flight.updated.Add(time.Duration(ticks) * db.EffectFrameDuration)
	steps := (flight.ticks+ticks)/15 - flight.ticks/15
	flight.ticks = (flight.ticks + ticks) % 180
	if !flying {
		flight.height = max(0, flight.height-ticks)
		if flight.height == 0 {
			delete(m.StarFlights, actor.ID)
		}
		return float64(flight.height)
	}
	target := 20
	if actor.Sitting && actor.Opt3State&db.Opt3Warm == 0 {
		target = 100
	}
	if active && actor.Opt3State&db.Opt3Warm == 0 {
		target, steps = 0, ticks
	}
	if flight.height < target {
		flight.height = min(target, flight.height+steps)
	} else {
		flight.height = max(target, flight.height-steps)
	}
	return max(0, float64(flight.height)+4*math.Sin(scene.DegreesToRadians(float64(flight.ticks*2))))
}
