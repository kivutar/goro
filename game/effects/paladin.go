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

type shieldChainFlight struct {
	Origins [5]scene.Point3
	Count   int
}

func (chain *shieldChainFlight) advance(ctx client.Context, effect Instance, now time.Time) {
	for chain.Count < len(chain.Origins) {
		at := effect.Starts.Add(time.Duration(3*chain.Count) * db.EffectFrameDuration)
		if now.Before(at) {
			break
		}
		chain.Origins[chain.Count] = EffectWorldAnchor(ctx, effect, at)
		chain.Count++
	}
}

func shieldChainImage(effect Instance, origin scene.Point3, index, process int) shieldImage {
	alpha := min(250, 25+45*process) - 10*max(0, process-14)
	direction := deterministicAngle(effect, index)
	position := origin
	position.X += .5 * float64(process-15) * math.Cos(direction)
	position.Z += .5 * float64(process-15) * math.Sin(direction)
	position.Y += 1.6
	return shieldImage{Position: position, Alpha: max(0, alpha), angle: -float64(process)*math.Pi/12 - 3*math.Pi/4}
}

func (m *System) drawShieldChainEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil || effect.ShieldChain == nil {
		return
	}
	effect.ShieldChain.advance(ctx, effect, now)
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	for i := 0; i < 5; i++ {
		process := frame - 3*i + 1
		if process <= 0 {
			continue
		}
		origin := effect.ShieldChain.Origins[i]
		drawShieldImage(screen, projection, texture, shieldChainImage(effect, origin, i, process), math.Sqrt2, false)
		for trail := 0; trail < 3; trail++ {
			previous := process/3*3 - 1 - 3*trail
			if previous < 2 {
				continue
			}
			particle := shieldChainImage(effect, origin, i, previous)
			particle.Alpha -= 150 + 25*trail
			drawShieldImage(screen, projection, texture, particle, math.Sqrt2, false)
		}
	}
}

func PressureImage(effect Instance, process int) shieldImage {
	position := effect.ActorOrigin
	position.Y += 23 - .6*float64(min(process, 32))
	return shieldImage{
		Position: position,
		Alpha:    max(0, min(process, 10)*20-max(0, process-32)*5),
		angle:    deterministicAngle(effect, 0) - float64(min(process, 32))*math.Pi/12,
	}
}

type pressureFlight struct {
	tick   int
	origin scene.Point3
	images [4]shieldImage
}

func (flight *pressureFlight) advance(ctx client.Context, effect Instance, now time.Time) {
	// The main cross fades at tick 72, followed by its three afterimages.
	process := min(int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1, 75)
	for flight.tick < process {
		copy(flight.images[1:], flight.images[:3])
		// The cross follows its victim for 32 ticks; each afterimage keeps
		// the position it had on its own tick, including after landing.
		if flight.tick < 32 {
			at := effect.Starts.Add(time.Duration(flight.tick) * db.EffectFrameDuration)
			flight.origin = EffectWorldAnchor(ctx, effect, at)
		}
		flight.tick++
		effect.ActorOrigin = flight.origin
		flight.images[0] = PressureImage(effect, flight.tick)
	}
}

func (m *System) drawPressureEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil || effect.pressure == nil {
		return
	}
	process := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	for i, particle := range effect.pressure.images {
		if i > 0 {
			particle.Alpha -= 100 + 25*(i-1)
		}
		drawShieldImage(screen, projection, texture, particle, 12*math.Sqrt2/5, true)
	}
	age := process - 28
	alpha := min(age, 10)*2 - max(0, age-10)
	if age > 0 && alpha > 0 {
		if ground := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0]); ground != nil {
			origin := EffectWorldAnchor(ctx, effect, now)
			origin.Y++
			radius := 2.7 * math.Pow(1.07, float64(age))
			vertices, indices := []render.Vertex3D{}, []uint16{}
			appendEffectQuad(&vertices, &indices, ground, [4]scene.Point3{
				scene.Add3(origin, scene.Point3{X: radius}), scene.Add3(origin, scene.Point3{Z: radius}),
				scene.Add3(origin, scene.Point3{X: -radius}), scene.Add3(origin, scene.Point3{Z: -radius}),
			}, color.RGBA{255, 255, 255, uint8(alpha)})
			screen.DrawTriangles3DOwned(vertices, indices, ground, TexturedEffectBillboardDrawOptions(true, false))
		}
	}
}

func (m *System) drawGospelCrossEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	n := float64(now.Sub(effect.Starts)/db.EffectFrameDuration + 1)
	origin := EffectWorldAnchor(ctx, effect, now)
	origin.Y += 1.2 - .6*math.Sin(deterministicAngle(effect, effect.X*31+effect.Y)+scene.DegreesToRadians(n))
	size := 1.5 * math.Sqrt2 / 5 * (1 + .05*math.Sin(scene.DegreesToRadians(n*5)))
	drawShieldImage(screen, projection, texture, shieldImage{Position: origin, Alpha: 200}, size, true)
}
