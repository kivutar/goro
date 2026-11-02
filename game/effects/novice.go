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

func (m *System) drawFirstAidEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 4*min(n, 20) - max(0, n-100)
	if alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	phase := scene.DegreesToRadians(float64(HitEffectRandom(effect, 1, 360) + 3*n))
	size := math.Sqrt2 * 10 / 5 * (1 + .05*math.Sin(phase))
	angle := scene.DegreesToRadians(float64(HitEffectRandom(effect, 0, 360)))
	p := effect.ActorOrigin
	DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, p.X, p.Z, p.Y+12.0/5,
		size, size, angle, color.RGBA{150, 150, 250, uint8(alpha)}, TexturedEffectBillboardDrawOptions(true, false))
}
