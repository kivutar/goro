package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawCylinderEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, componentIndex int, x, y, z float64, componentDuration time.Duration, now time.Time) {
	texture := m.effectTexture(ctx.Resources, component.textureName)
	if texture == nil {
		return
	}
	duplicates := maxInt(component.duplicate, 1)
	baseDuration := componentDuration - worldEffectComponentMaxStartOffset(component)
	if baseDuration <= 0 {
		baseDuration = component.duration
	}
	if baseDuration <= 0 {
		baseDuration = 500 * time.Millisecond
	}
	for i := 0; i < duplicates; i++ {
		instanceProgress, active := worldEffectComponentDuplicateProgressForDraw(effect.starts, component, i, baseDuration, now)
		if !active {
			continue
		}
		m.drawCylinderEffectInstance(screen, projection, texture, effect, component, componentIndex, x, y, z, instanceProgress, i)
	}
}

func (m *WorldMode) drawCylinderEffectInstance(screen *render.Frame, projection sceneProjection, texture *render.Image, effect worldEffect, component worldEffectComponent, componentIndex int, x, y, z, progress float64, duplicateIndex int) {
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	bottomSize, topSize, height := effectCylinderAnimatedDimensions(component, progress)
	if !component.fixedPerspective {
		drawWorldCylinderBandOriented(screen, m.whitePixel, texture, projection, component, x, y, z+component.posZ, bottomSize, topSize, height, effectComponentTint(component, alpha), maxInt(component.circleSides, component.totalCircleSides), progress)
		return
	}
	angle := 0.0
	if component.rotate {
		angle += progress * 2 * math.Pi
		angle += deterministicAngle(effect, componentIndex*101+duplicateIndex+31) * 0.08
	}
	if component.angleZRandom != 0 {
		angle += deterministicAngle(effect, componentIndex*101+duplicateIndex) * component.angleZRandom / 360
	}
	drawTexturedEffectCylinder(screen, projection, texture, x, y, z+component.posZ, effectCylinderDraw{
		bottomSize:       bottomSize,
		topSize:          topSize,
		totalCircleSides: component.totalCircleSides,
		circleSides:      component.circleSides,
		alpha:            alpha,
		angle:            angle,
	})
}

func effectCylinderAnimatedDimensions(component worldEffectComponent, progress float64) (float64, float64, float64) {
	bottomSize := component.bottomSize
	topSize := component.topSize
	height := component.height
	switch component.animation {
	case 1:
		height *= progress
	case 2:
		topSize *= progress
	case 3:
		scale := 1 - progress
		bottomSize *= scale
		topSize *= scale
		if progress < 0.5 {
			height *= progress * 2
		} else if progress > 0.5 {
			height *= (1 - progress) * 2
		}
	case 4:
		bottomSize *= progress
		topSize *= progress
	case 5:
		if progress < 0.5 {
			height *= progress * 2
		} else {
			height *= (1 - progress) * 2
		}
	}
	return bottomSize, topSize, height
}

type effectCylinderDraw struct {
	bottomSize       float64
	topSize          float64
	totalCircleSides int
	circleSides      int
	alpha            float64
	angle            float64
}

func drawTexturedEffectCylinder(screen *render.Frame, projection sceneProjection, texture *render.Image, worldX, worldY, worldZ float64, draw effectCylinderDraw) {
	if screen == nil || texture == nil || draw.alpha <= 0 || draw.topSize <= 0 || draw.totalCircleSides <= 0 || draw.circleSides <= 0 {
		return
	}
	right, up, _, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	center := modelPoint3{x: worldX, y: worldZ, z: worldY}
	tint := color.RGBA{R: 255, G: 255, B: 255, A: uint8(clampFloat(draw.alpha, 0, 1) * 255)}
	vertices := make([]render.Vertex3D, 0, (draw.circleSides+1)*2)
	indices := make([]uint16, 0, draw.circleSides*6)
	point := func(radius, angle float64) modelPoint3 {
		return add3(add3(center, mul3(right, math.Sin(angle)*radius)), mul3(up, math.Cos(angle)*radius))
	}
	for i := 0; i <= draw.circleSides; i++ {
		a := float64(i) / float64(draw.totalCircleSides)
		angle := draw.angle + a*2*math.Pi
		u := float32(a * float64(draw.totalCircleSides) / float64(draw.circleSides))
		vertices = append(vertices,
			texturedSurfaceVertex3D(point(draw.bottomSize, angle), texturePoint{u: u, v: 1}, tint, w, h),
			texturedSurfaceVertex3D(point(draw.topSize, angle), texturePoint{u: u, v: 0}, tint, w, h),
		)
		if i < draw.circleSides {
			base := uint16(i * 2)
			indices = append(indices, base, base+1, base+2, base+1, base+3, base+2)
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
}

func drawWorldCylinderBandOriented(screen *render.Frame, white, texture *render.Image, projection sceneProjection, component worldEffectComponent, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int, progress float64) {
	right := modelPoint3{x: 1}
	depth := modelPoint3{z: 1}
	up := modelPoint3{y: 1}
	if component.angleX != 0 || component.angleY != 0 || component.angleZ != 0 || component.rotateWithCamera || component.rotate {
		angleY := component.angleY
		if component.rotate {
			angleY += progress * 360
		}
		if component.rotateWithCamera {
			angleY += projection.cameraYaw
		}
		right = rotateEffectCylinderVector(right, component.angleX, angleY, component.angleZ)
		depth = rotateEffectCylinderVector(depth, component.angleX, angleY, component.angleZ)
		up = rotateEffectCylinderVector(up, component.angleX, angleY, component.angleZ)
	}
	drawWorldCylinderBandWithBasis(screen, white, texture, x, y, z, bottomRadius, topRadius, height, c, segments, right, depth, up)
}

func rotateEffectCylinderVector(v modelPoint3, angleX, angleY, angleZ float64) modelPoint3 {
	v = rotateModelPointX(v, degreesToRadians(angleX))
	v = rotateModelPointY(v, degreesToRadians(angleY))
	v = rotateModelPointZ(v, degreesToRadians(angleZ))
	return v
}

func rotateModelPointX(v modelPoint3, angle float64) modelPoint3 {
	if angle == 0 {
		return v
	}
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	return modelPoint3{x: v.x, y: v.y*cosA - v.z*sinA, z: v.y*sinA + v.z*cosA}
}

func rotateModelPointY(v modelPoint3, angle float64) modelPoint3 {
	if angle == 0 {
		return v
	}
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	return modelPoint3{x: v.x*cosA + v.z*sinA, y: v.y, z: -v.x*sinA + v.z*cosA}
}

func rotateModelPointZ(v modelPoint3, angle float64) modelPoint3 {
	if angle == 0 {
		return v
	}
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	return modelPoint3{x: v.x*cosA - v.y*sinA, y: v.x*sinA + v.y*cosA, z: v.z}
}

func drawWorldCylinderBandWithBasis(screen *render.Frame, white, texture *render.Image, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int, right, depth, up modelPoint3) {
	drawWorldCylinderBandWithOptions(screen, white, texture, x, y, z, bottomRadius, topRadius, height, c, segments, right, depth, up, 1, render.BlendLighter)
}

func drawWorldCylinderBandWithOptions(screen *render.Frame, white, texture *render.Image, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int, right, depth, up modelPoint3, textureRepeats float64, blend render.Blend) {
	if segments < 3 || bottomRadius <= 0.01 || topRadius <= 0.01 || math.Abs(height) <= 0.01 || c.A == 0 {
		return
	}
	vertices := make([]render.Vertex3D, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)
	tint := c
	srcW, srcH := float32(1), float32(1)
	source := white
	if texture != nil {
		source = texture
		bounds := texture.Bounds()
		srcW = float32(bounds.Dx())
		srcH = float32(bounds.Dy())
	}
	for i := 0; i <= segments; i++ {
		u := float32(float64(i) / float64(segments) * textureRepeats)
		angle := float64(i) * 2 * math.Pi / float64(segments)
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		center := modelPoint3{x: x, y: z, z: y}
		bottom := add3(center, add3(mul3(right, cosine*bottomRadius), mul3(depth, sine*bottomRadius)))
		top := add3(center, add3(add3(mul3(right, cosine*topRadius), mul3(depth, sine*topRadius)), mul3(up, height)))
		vertices = append(vertices,
			texturedSurfaceVertex3D(bottom, texturePoint{u: u, v: 1}, tint, srcW, srcH),
			texturedSurfaceVertex3D(top, texturePoint{u: u, v: 0}, tint, srcW, srcH),
		)
		if i == segments {
			continue
		}
		base := uint16(i * 2)
		indices = append(indices, base, base+1, base+3, base, base+3, base+2)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = blend
	screen.DrawTriangles3DOwned(vertices, indices, source, options)
}
