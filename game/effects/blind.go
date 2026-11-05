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

func (m *System) drawBlind(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	if !scene.IsLocalActor(ctx, effect.ActorID) || now.Before(effect.Starts) || len(component.TextureFiles) != 2 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	white := m.EffectFileTexture(ctx.Resources, component.TextureFiles[1])
	right, up, units, ok := projection.BillboardBasis(x, y, z)
	if texture == nil || white == nil || !ok {
		return
	}
	radius := float64(140-10*(effect.EffectID-Devil1+1)) / 5
	shade := uint8(30)
	if effect.EffectID == Blind {
		radius, shade = 10, 10
	}
	tint := color.RGBA{shade, shade, shade, uint8(min(255, int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1))}
	point := func(dx, dy float64) scene.Point3 {
		return scene.Add3(scene.Point3{X: x, Y: z, Z: y}, scene.Add3(scene.Mul3(right, dx), scene.Mul3(up, dy)))
	}
	opts := TexturedEffectBillboardDrawOptions(false, true)
	opts.AfterBillboards = true
	opts.DisableFog = true
	for _, signs := range [][2]float64{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		var vertices []render.Vertex3D
		var indices []uint16
		appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{point(signs[0]*radius, 0), point(signs[0]*radius, signs[1]*radius), point(0, signs[1]*radius), point(0, 0)}, tint)
		screen.DrawTriangles3DOwned(vertices, indices, texture, opts)
	}
	// Extend the original opaque surround to cover modern widescreen viewports.
	outer := radius + math.Max(projection.ScreenW, projection.ScreenH)*units*2
	var vertices []render.Vertex3D
	var indices []uint16
	for _, rect := range [][4]float64{{-outer, radius, outer, outer}, {-outer, -outer, outer, -radius}, {-outer, -radius, -radius, radius}, {radius, -radius, outer, radius}} {
		appendEffectQuad(&vertices, &indices, white, [4]scene.Point3{point(rect[0], rect[3]), point(rect[2], rect[3]), point(rect[2], rect[1]), point(rect[0], rect[1])}, tint)
	}
	screen.DrawTriangles3DOwned(vertices, indices, white, opts)
}
