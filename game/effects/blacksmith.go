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

func (m *System) drawBlacksmithEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if frame < 0 {
		return
	}
	origin := EffectWorldAnchor(ctx, effect, now)
	if effect.EffectID == HasteUp && frame < 100 {
		// The orbit stays at the casting position; the screen-space rays follow the actor.
		particleOrigin := origin
		if effect.HasActorTransform {
			particleOrigin = effect.ActorOrigin
		}
		n := float64(frame + 1)
		for i := 0; i < 4; i++ {
			longitude := float64(i*90) + .3*n + .1*n*(n+1)
			angle := scene.DegreesToRadians(math.Trunc(longitude))
			position := scene.Add3(particleOrigin, scene.Point3{
				X: 1.4 * math.Sin(angle), Z: 1.4 * math.Cos(angle),
				Y: (.1*n + .001*n*(n+1)) / 5,
			})
			m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, position, frame/4+1, 1.2, 0,
				color.RGBA{255, 255, 255, effectParticleAlpha(frame, 100, 1, 90, 254)})
		}
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	center, ok := projection.ProjectPoint(origin.X, origin.Z, origin.Y)
	if texture == nil || !ok {
		return
	}
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	vertex := func(angle, radius float64, u, v, alpha float32) render.Vertex {
		return render.Vertex{
			DstX: center.X + float32(math.Cos(angle)*radius), DstY: center.Y - float32(math.Sin(angle)*radius),
			SrcX: u, SrcY: v, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha,
		}
	}
	var vertices []render.Vertex
	var indices []uint16
	switch effect.EffectID {
	case HasteUp:
		if frame >= 80 {
			return
		}
		center.Y -= 40
		n := float64(min(frame+1, 40))
		for i := 0; i < 20; i++ {
			salt := i * 5
			speed := float64(10+HitEffectRandom(effect, salt+1, 60)) / 10
			angle := float64(HitEffectRandom(effect, salt, 360)) + speed*n - speed/120*n*(n+1)/2
			angle += float64(max(0, frame-39)) * float64(15+HitEffectRandom(effect, salt+4, 10)) / 10
			halfAngle := float64(5+HitEffectRandom(effect, salt+2, 25)) / 10
			radius := float64(50+HitEffectRandom(effect, salt+3, 20)) / 13 * n
			base := uint16(len(vertices))
			vertices = append(vertices, vertex(0, 0, w/2, 0, 254.0/255),
				vertex(scene.DegreesToRadians(float64(int(angle-halfAngle))), radius, w, h, 254.0/255),
				vertex(scene.DegreesToRadians(float64(int(angle+halfAngle))), radius, 0, h, 254.0/255))
			indices = append(indices, base, base+1, base+2)
		}
	case Overthrust:
		if frame >= 40 {
			return
		}
		center.Y -= 20
		n := float64(frame + 1)
		radius := 7*n - .175*n*(n+1)/2
		inner := math.Max(0, radius-10)
		alpha := float32(254 * (40 - n) / 40 / 255)
		for i := 0; i < 36; i++ {
			angle, next := scene.DegreesToRadians(float64(i*10)), scene.DegreesToRadians(float64((i+1)*10))
			u := float32(i%4) * w / 4
			base := uint16(len(vertices))
			vertices = append(vertices, vertex(angle, inner, u, h, alpha), vertex(next, inner, u+w/4, h, alpha),
				vertex(next, radius, u+w/4, 0, alpha), vertex(angle, radius, u, 0, alpha))
			indices = append(indices, base, base+2, base+3, base, base+1, base+2)
		}
	}
	if len(indices) > 0 {
		screen.DrawTrianglesOwned(vertices, indices, texture, &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat})
	}
}
