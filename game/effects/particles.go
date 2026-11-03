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

func (m *System) drawEffectParticleSprite(screen *render.Frame, ctx client.Context, projection scene.Projection, file string, position scene.Point3, motion int, scale, angle float64, tint color.RGBA) {
	m.drawEffectParticleSpriteAction(screen, ctx, projection, file, position, 0, motion, scale, angle, tint)
}

func (m *System) drawEffectParticleSpriteAction(screen *render.Frame, ctx client.Context, projection scene.Projection, file string, position scene.Point3, action, motion int, scale, angle float64, tint color.RGBA) {
	if tint.A == 0 || scale <= 0 {
		return
	}
	view := m.EffectSpriteView(ctx.Resources, file)
	if view == nil || action < 0 || action >= len(view.Act.Actions) || len(view.Act.Actions[action].Animations) == 0 {
		return
	}
	motion %= len(view.Act.Actions[action].Animations)
	key := scene.SpriteBillboardKey{ActionIndex: action, Motion: motion}
	billboard, ok := view.Billboards[key]
	if !ok {
		billboard, ok = scene.ComposeSingleSpriteBillboard(view, view.Act.Actions[action].Animations[motion])
		if !ok {
			return
		}
		view.Billboards[key] = billboard
	}
	alpha := float64(tint.A) / 255
	tint.A = 255
	scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, position.X, position.Z, position.Y,
		scale*scene.PixelRatio, angle, alpha, 1, tint, scene.SpriteBillboardTriangleDrawOptions())
}

type effectParticle struct {
	Position           scene.Point3
	size, Angle, Alpha float64
}

func RisingParticleAt(effect Instance, index, age int) effectParticle {
	life, fade, initial := 40, 32, 0.0
	speed := float64(30+HitEffectRandom(effect, index*4+1, 30)) / 500
	size := float64(40+HitEffectRandom(effect, index*4+2, 50)) / 100
	yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, index*4+3, 360)))
	radius := float64(2+HitEffectRandom(effect, index*4+4, 6)) / 5
	if effect.EffectID == EDP {
		speed = float64(30+HitEffectRandom(effect, index*4+1, 50)) / 500
		size = float64(20+HitEffectRandom(effect, index*4+2, 20)) / 100
	}
	if effect.EffectID == VenomDust2 {
		life, fade, initial = 10, 8, 70
		speed = float64(20+HitEffectRandom(effect, index*4+1, 20)) / 400
		size, radius = .8, 0
	}
	if age < 0 || age >= life {
		return effectParticle{}
	}
	n := float64(age + 1)
	alpha := math.Min(254, initial+25.4*n)
	if age >= fade {
		alpha = math.Min(254, initial+25.4*float64(fade)) * float64(life-age-1) / float64(life-fade)
	}
	return effectParticle{
		Position: scene.Point3{X: math.Sin(yaw) * radius, Y: n*speed - speed/(2*float64(life))*n*(n+1)/2, Z: math.Cos(yaw) * radius},
		size:     size, Angle: scene.DegreesToRadians(n * 3), Alpha: alpha / 255,
	}
}

func (m *System) drawRisingParticlesEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	view := m.EffectSpriteView(ctx.Resources, component.SpriteFile)
	if view == nil || len(view.Act.Actions) == 0 || len(view.Act.Actions[0].Animations) == 0 {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	life, last := 40, min(frame/5, 42)
	interval := 5
	origin := effect.ActorOrigin
	if effect.EffectID == VenomDust2 {
		life, last = 10, frame/5
	} else if effect.EffectID == Detoxication {
		if lessEffectsEnabled(ctx) {
			interval = 15
		}
		// ProcessAlpha puts particles that would outlive the emitter straight
		// into their final fade, starting from zero alpha.
		last = min(frame/interval, 60/interval)
	} else if effect.EffectID == EDP {
		interval, last = 3, min(frame/3, 26)
		origin = EffectWorldAnchor(ctx, effect, now)
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	for i := max(0, (frame-life+interval)/interval); i <= last; i++ {
		age := frame - i*interval
		particle := RisingParticleAt(effect, i, age)
		if particle.Alpha <= 0 {
			continue
		}
		motion := (age/4 + 1) % len(view.Act.Actions[0].Animations)
		key := scene.SpriteBillboardKey{Motion: motion}
		billboard, ok := view.Billboards[key]
		if !ok {
			billboard, ok = scene.ComposeSingleSpriteBillboard(view, view.Act.Actions[0].Animations[motion])
			if !ok {
				continue
			}
			view.Billboards[key] = billboard
		}
		position := scene.Add3(origin, particle.Position)
		scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard,
			position.X, position.Z, position.Y, particle.size*scene.PixelRatio, -particle.Angle, particle.Alpha, 1, color.RGBA{255, 255, 255, 255}, options)
	}
}

// Full Strip and Disarm share the equipment-particle burst.
func (m *System) drawEquipmentBurst(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) || frame > 108 {
		return
	}
	for kind, path := range component.TextureFiles {
		texture := m.EffectFileTexture(ctx.Resources, path)
		if texture == nil {
			continue
		}
		count, size := 16, float64(5+kind*4)*math.Sqrt2/5
		if effect.EffectID == RGCoin3 {
			count, size = 4, math.Sqrt2
		}
		for i := 0; i < count; i++ {
			seed := (kind*16 + i) * 4
			n := frame + 1 - 18 - HitEffectRandom(effect, seed, 30)
			alpha := min(n, 10)*25 - max(0, n-25)*7
			if n <= 0 || alpha <= 0 {
				continue
			}
			pitch := scene.DegreesToRadians(float64(HitEffectRandom(effect, seed+1, 360)))
			yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, seed+2, 360)))
			center := scene.Add3(effect.ActorOrigin, effectParticlePoint(float64(12+n), 0, pitch, yaw))
			roll := scene.DegreesToRadians(float64(HitEffectRandom(effect, seed+3, 360) + 3*(frame+1)))
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, center.X, center.Z, center.Y, size, size, roll, color.RGBA{250, 100, 100, uint8(alpha)}, true)
		}
	}
}

type wanderingEffectParticle struct {
	Position        scene.Point3
	angles          [4]int
	seed, Age, kind int
	radius, speed   float64
}

func advanceWanderingAngles(particle *wanderingEffectParticle, effect Instance, tick int) {
	for axis := 0; axis < 4; axis += 2 {
		a, target := &particle.angles[axis], &particle.angles[axis+1]
		if *target > *a {
			*a = min(*a+3, 359)
			if *a > *target {
				*target = HitEffectRandom(effect, particle.seed+tick*4+axis, max(*a, 1))
			}
		} else {
			*a = max(*a-2, 0)
			if *a <= *target {
				*target = 359 - HitEffectRandom(effect, particle.seed+tick*4+axis, max(*a, 1))
			}
		}
	}
}

type risingBuffParticles struct {
	origins [21]scene.Point3
	count   int
}

func (m *System) DrawRisingBuffParticles(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) || len(component.TextureFiles) != 2 {
		return
	}
	textures := [2]*render.Image{
		m.EffectFileTexture(ctx.Resources, component.TextureFiles[0]),
		m.EffectFileTexture(ctx.Resources, component.TextureFiles[1]),
	}
	start, end, step, life := 0, 80, 4, 55
	jitter, radius, radiusStep := 11, 2.0, .1
	if effect.EffectID == SmaReady {
		start, end = 40, 120
	} else if effect.EffectID == Stin3 {
		end, step, life = 199, 1, 50
		jitter, radius, radiusStep = 9, 1.2, .08
	}
	if effect.EffectID != Stin3 {
		if effect.BuffParticles == nil {
			return
		}
		// Keep each emission's position when the actor replaces its movement path.
		p := effect.BuffParticles
		for p.count < len(p.origins) && start+p.count*step <= min(frame, end) {
			at := effect.Starts.Add(time.Duration(start+p.count*step) * db.EffectFrameDuration)
			p.origins[p.count] = EffectWorldAnchor(ctx, effect, at)
			p.count++
		}
	}
	// ParticleTime and SMA both emit four independently rising particles.
	first := max(start, (frame-life+step)/step*step)
	for emitted := first; emitted <= min(frame, end); emitted += step {
		age := frame - emitted + 1
		alpha := float64(max(15*(min(age, 10)-1)-3*max(age-10, 0), 0)) / 255
		if alpha <= 0 {
			continue
		}
		var center scene.Point3
		if effect.EffectID == Stin3 {
			center = scene.Add3(effect.ActorOrigin, scene.Mul3(effect.ProjectileDirection, .8*float64(max(emitted-2, 0))/5))
		} else {
			center = effect.BuffParticles.origins[(emitted-start)/step]
		}
		for i := 0; i < 4; i++ {
			salt := (emitted*4 + i) * 5
			position := scene.Add3(center, scene.Point3{
				X: float64(HitEffectRandom(effect, salt+2, jitter)-jitter/2) / 5,
				Z: float64(HitEffectRandom(effect, salt+3, jitter)-jitter/2) / 5,
				Y: (6 + float64(age)*(.1+.01*float64(HitEffectRandom(effect, salt+4, 20)))) / 5,
			})
			size := math.Sqrt2 * (radius + radiusStep*float64(HitEffectRandom(effect, salt+5, 10))) / 5
			index := 0
			if age == 35-i*7 || age == 36-i*7 {
				index = 1
			}
			drawTexturedEffectBillboardRotatedXY(screen, projection, textures[index],
				position.X, position.Z, position.Y, size, size, scene.DegreesToRadians(float64(-5*age+45)), effectComponentTint(component, alpha), true)
		}
	}
}
