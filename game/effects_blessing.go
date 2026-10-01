package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

func blessingCircleAlpha(component worldEffectComponent, elapsed time.Duration) float64 {
	if elapsed < 0 || elapsed >= component.duration || component.frameDelay <= 0 {
		return 0
	}
	n := float64(elapsed/component.frameDelay + 1)
	frames := float64(component.duration / component.frameDelay)
	// Blessing's Prim3DCircle ramps for 30 ticks at both ends.
	return component.alphaMax * clampFloat(math.Min(n/30, (frames-n)/30), 0, 1)
}

func (m *WorldMode) drawBlessingCircleEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	alpha := blessingCircleAlpha(component, now.Sub(effect.starts)-component.delay)
	if alpha <= 0 || component.circleSides < 3 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.textureName)
	if texture == nil {
		return
	}
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	tint := effectComponentTint(component, alpha)
	vertices := make([]render.Vertex3D, 0, component.circleSides*3)
	indices := make([]uint16, 0, component.circleSides*3)
	center := modelPoint3{x: x, y: z + component.posZ, z: y}
	point := func(i int) modelPoint3 {
		angle := float64(i) * 2 * math.Pi / float64(component.circleSides)
		return add3(center, modelPoint3{x: component.sizeStart * math.Cos(angle), z: component.sizeStart * math.Sin(angle)})
	}
	// Render3DCircle with PT_FILLCIRCLE: a fan on the world X/Z plane,
	// alpha_down from centre (V=1) to rim (V=0), U advancing 1/4 per wedge.
	for i := 0; i < component.circleSides; i++ {
		u := float32(i) / 4
		base := uint16(len(vertices))
		vertices = append(vertices,
			texturedSurfaceVertex3D(center, texturePoint{u: u + 0.25, v: 1}, tint, w, h),
			texturedSurfaceVertex3D(point(i), texturePoint{u: u, v: 0}, tint, w, h),
			texturedSurfaceVertex3D(point(i+1), texturePoint{u: u + 0.25, v: 0}, tint, w, h))
		indices = append(indices, base, base+1, base+2)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.DepthTest = !component.overlay // The original explicitly uses RF_NODEPTHCHECK.
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}
