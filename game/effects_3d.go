package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *WorldMode) draw3DEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, componentIndex int, worldX, worldY, worldZ float64, now time.Time) {
	if component.textureFile == "" && len(component.textureFiles) == 0 && component.spriteFile == "" {
		return
	}
	duplicates := maxInt(component.duplicate, 1)
	componentDuration := component.duration
	if componentDuration <= 0 {
		componentDuration = 500 * time.Millisecond
	}
	for i := 0; i < duplicates; i++ {
		starts := effect.starts.Add(worldEffectComponentStartOffset(component, i))
		progress, active := worldEffectComponentDuplicateProgressForDraw(effect.starts, component, i, componentDuration, now)
		if !active {
			continue
		}
		alpha := effectBillboardAlphaForDuplicate(progress, component, i)
		if alpha <= 0 {
			continue
		}
		salt := componentIndex*1009 + i*37
		offsetX, offsetY, offsetZ := m.effect3DOffset(ctx, component, effect, salt, i, progress, worldX, worldY, worldZ)
		sizeX, sizeY := effect3DSize(component, effect, salt, progress, i)
		if sizeX <= 0 || sizeY <= 0 {
			continue
		}
		drawX := worldX + offsetX
		drawY := worldY + offsetY
		drawZ := worldZ + offsetZ
		if component.textureFile != "" || len(component.textureFiles) > 0 {
			texture := m.effectTextureFrame(ctx.Resources, component, progress)
			if texture == nil {
				continue
			}
			angle := worldEffectBillboardAngleForEffect(component, projection, effect, salt, progress)
			drawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, drawX, drawY, drawZ, sizeX, sizeY, angle, effectComponentTint(component, alpha), texturedEffectBillboardDrawOptions(component.blendAdditive, component.overlay))
			continue
		}
		size := (sizeX + sizeY) * 0.5
		m.draw3DSpriteEffect(screen, ctx, projection, effect, component, drawX, drawY, drawZ, size, alpha, progress, starts, now)
	}
}

func (m *WorldMode) effectTextureFrame(manager *res.Manager, component worldEffectComponent, progress float64) *render.Image {
	if len(component.textureFiles) == 0 {
		return m.effectFileTexture(manager, component.textureFile)
	}
	index := int(clampFloat(progress, 0, 0.999999) * float64(len(component.textureFiles)))
	if component.frameDelay > 0 {
		duration := component.duration
		if duration <= 0 {
			duration = 500 * time.Millisecond
		}
		elapsed := time.Duration(clampFloat(progress, 0, 0.999999) * float64(duration))
		index = int(elapsed/component.frameDelay) % len(component.textureFiles)
	}
	return m.effectFileTexture(manager, component.textureFiles[index])
}

func effectBillboardAlpha(progress float64, component worldEffectComponent) float64 {
	return effectBillboardAlphaForDuplicate(progress, component, 0)
}

func effectBillboardAlphaForDuplicate(progress float64, component worldEffectComponent, duplicateIndex int) float64 {
	alphaMax := component.alphaMax
	if alphaMax <= 0 {
		alphaMax = 1
	}
	alphaMax += component.alphaMaxDelta * float64(duplicateIndex)
	alphaMax = clampFloat(alphaMax, 0, 1)
	switch {
	case component.fadeIn && progress < 0.25:
		return progress / 0.25 * alphaMax
	case component.fadeOut && progress > 0.75:
		return (1 - progress) / 0.25 * alphaMax
	case component.sparkling:
		sparkNumber := component.sparkNumber
		if sparkNumber <= 0 {
			sparkNumber = 1
		}
		steps := progress * 100
		return alphaMax * ((math.Cos((steps*11*float64(sparkNumber)*math.Pi)/180) + 1) / 2)
	default:
		return alphaMax
	}
}

func effectBillboardSize(progress float64, component worldEffectComponent) float64 {
	start := component.sizeStart
	end := component.sizeEnd
	if start <= 0 {
		start = 1
	}
	if end <= 0 {
		end = start
	}
	if !component.sizeSmooth {
		return start + (end-start)*progress
	}
	return start + (end-start)*math.Log10(progress*9+1)
}

func (m *WorldMode) effect3DOffset(ctx client.Context, component worldEffectComponent, effect worldEffect, salt int, duplicateIndex int, progress float64, worldX, worldY, worldZ float64) (float64, float64, float64) {
	staticX := deterministicSigned(effect, salt+1) * component.posXRand
	staticY := deterministicSigned(effect, salt+2) * component.posYRand
	staticZ := deterministicSigned(effect, salt+10) * component.posZRand
	startX := component.posX + staticX + component.posXStartMiddle + deterministicSigned(effect, salt+11)*component.posXStartRand
	startY := component.posY + staticY + component.posYStartMiddle + deterministicSigned(effect, salt+12)*component.posYStartRand
	startZ := component.posZ + staticZ + component.posZStartMiddle + deterministicSigned(effect, salt+3)*component.posZStartRand
	endX := component.posXEnd + staticX + component.posXEndMiddle + deterministicSigned(effect, salt+4)*component.posXEndRand
	endY := component.posYEnd + staticY + component.posYEndMiddle + deterministicSigned(effect, salt+5)*component.posYEndRand
	endZ := component.posZEnd + staticZ + component.posZEndMiddle + deterministicSigned(effect, salt+6)*component.posZEndRand
	if component.circlePattern {
		angle := effectComponentAngleDegrees(component, effect, salt)
		radians := angle * math.Pi / 180
		outer := deterministicFloatRange(effect, salt+23, component.circleOuterRandMin, component.circleOuterRandMax)
		startX = math.Sin(radians)*component.circleInnerSize + staticX
		startY = math.Cos(radians)*component.circleInnerSize + staticY
		endX = math.Sin(radians)*outer + staticX
		endY = math.Cos(radians)*outer + staticY
	}
	if !component.circlePattern && component.posXEnd == 0 && component.posXEndRand == 0 && component.posXEndMiddle == 0 && component.posXStartRand == 0 && component.posXStartMiddle == 0 {
		endX = startX
	}
	if !component.circlePattern && component.posYEnd == 0 && component.posYEndRand == 0 && component.posYEndMiddle == 0 && component.posYStartRand == 0 && component.posYStartMiddle == 0 {
		endY = startY
	}
	if component.posZEnd == 0 && component.posZEndRand == 0 && component.posZEndMiddle == 0 {
		endZ = startZ
	}
	if component.fromSrc || component.toSrc {
		otherX, otherY, otherZ, ok := effectOtherEndpoint(ctx, effect, worldX, worldY, worldZ)
		if ok {
			dx := otherX - worldX
			dy := otherY - worldY
			dz := otherZ - worldZ
			if component.fromSrc {
				endX += dx
				endY += dy
				endZ += dz
			}
			if component.toSrc {
				startX += dx
				startY += dy
				startZ += dz
			}
		}
	}
	x := effectPositionAxis(progress, startX, endX, component.posXSmooth)
	y := effectPositionAxis(progress, startY, endY, component.posYSmooth)
	z := effectPositionAxis(progress, startZ, endZ, component.posZSmooth)
	if component.retreat != 0 {
		dx := endX - startX
		dy := endY - startY
		dist := math.Hypot(dx, dy)
		if dist > 0.001 {
			factor := math.Sin(progress*math.Pi) * component.retreat
			x -= dx / dist * factor
			y -= dy / dist * factor
		}
	}
	if component.arc != 0 {
		z += math.Sin(progress*math.Pi) * component.arc
	}
	if component.orbitRadiusX != 0 || component.orbitRadiusY != 0 || component.orbitRadiusZ != 0 {
		angle := progress*component.orbitRotations*350*math.Pi/180 - (component.orbitPhase+component.orbitPhaseDelta*float64(duplicateIndex))*math.Pi/2
		if component.orbitRadiusX != 0 {
			x = math.Cos(angle) * component.orbitRadiusX
			if component.orbitClockwise {
				x = -x
			}
		}
		if component.orbitRadiusY != 0 {
			y = math.Sin(angle) * component.orbitRadiusY
		}
	}
	return x, y, z
}

func effectOtherEndpoint(ctx client.Context, effect worldEffect, fallbackX, fallbackY, fallbackZ float64) (float64, float64, float64, bool) {
	if effect.targetID == 0 || ctx.World == nil {
		return fallbackX, fallbackY, fallbackZ, false
	}
	if actor, ok := ctx.World.Actors[effect.targetID]; ok {
		x, y := actorRenderPosition(actor, time.Now())
		return cellCenter(x), cellCenter(y), terrainHeightAt(ctx.World, x, y) + 0.07, true
	}
	if isLocalActor(ctx, effect.targetID) {
		x, y := actorRenderPosition(ctx.World.Player, time.Now())
		return cellCenter(x), cellCenter(y), terrainHeightAt(ctx.World, x, y) + 0.07, true
	}
	return fallbackX, fallbackY, fallbackZ, false
}

func effectPositionAxis(progress, start, end float64, smooth bool) float64 {
	if smooth {
		return start + (end-start)*math.Log10(progress*9+1)
	}
	return start + (end-start)*progress
}

func worldEffectBillboardAngleForEffect(component worldEffectComponent, projection sceneProjection, effect worldEffect, salt int, progress float64) float64 {
	angle := effectComponentAngleDegrees(component, effect, salt)
	if component.angleRandMax <= component.angleRandMin {
		angle = component.angleStart + (component.angleEnd-component.angleStart)*progress
	}
	angle *= math.Pi / 180
	if component.rotateWithCamera {
		angle += degreesToRadians(projection.cameraYaw)
	}
	return angle
}

func effectComponentAngleDegrees(component worldEffectComponent, effect worldEffect, salt int) float64 {
	if component.angleRandMax > component.angleRandMin {
		return deterministicFloatRange(effect, salt+21, component.angleRandMin, component.angleRandMax)
	}
	return component.angleStart
}

func effect3DSize(component worldEffectComponent, effect worldEffect, salt int, progress float64, duplicateIndex int) (float64, float64) {
	size := effectBillboardSize(progress, component)
	sizeX := size
	sizeY := size
	if component.sizeStartX > 0 || component.sizeEndX > 0 {
		sizeX = effectAxisSize(progress, component.sizeStartX, component.sizeEndX, component.sizeSmooth)
	}
	if component.sizeStartY > 0 || component.sizeEndY > 0 {
		sizeY = effectAxisSize(progress, component.sizeStartY, component.sizeEndY, component.sizeSmooth)
	}
	if component.sizeStartXRandMax > component.sizeStartXRandMin || component.sizeEndXRandMax > component.sizeEndXRandMin {
		start := component.sizeStartX
		if component.sizeStartXRandMax > component.sizeStartXRandMin {
			start = deterministicFloatRange(effect, salt+31, component.sizeStartXRandMin, component.sizeStartXRandMax)
		}
		end := component.sizeEndX
		if component.sizeEndXRandMax > component.sizeEndXRandMin {
			end = deterministicFloatRange(effect, salt+32, component.sizeEndXRandMin, component.sizeEndXRandMax)
		}
		sizeX = effectAxisSize(progress, start, end, component.sizeSmooth)
	}
	if component.sizeStartYRandMax > component.sizeStartYRandMin || component.sizeEndYRandMax > component.sizeEndYRandMin {
		start := component.sizeStartY
		if component.sizeStartYRandMax > component.sizeStartYRandMin {
			start = deterministicFloatRange(effect, salt+33, component.sizeStartYRandMin, component.sizeStartYRandMax)
		}
		end := component.sizeEndY
		if component.sizeEndYRandMax > component.sizeEndYRandMin {
			end = deterministicFloatRange(effect, salt+34, component.sizeEndYRandMin, component.sizeEndYRandMax)
		}
		sizeY = effectAxisSize(progress, start, end, component.sizeSmooth)
	}
	if component.sizeDelta != 0 {
		delta := component.sizeDelta * float64(duplicateIndex) * effectPixelRatio
		sizeX += delta
		sizeY += delta
	}
	if component.sizeRand != 0 {
		sizeX += deterministicSigned(effect, salt+7) * component.sizeRand
		sizeY = sizeX
	}
	if component.sizeRandX != 0 {
		if component.sizeRandXMiddle != 0 {
			sizeX = component.sizeRandXMiddle + deterministicSigned(effect, salt+8)*component.sizeRandX
		} else {
			sizeX += deterministicSigned(effect, salt+8) * component.sizeRandX
		}
	}
	if component.sizeRandY != 0 {
		if component.sizeRandYMiddle != 0 {
			sizeY = component.sizeRandYMiddle + deterministicSigned(effect, salt+9)*component.sizeRandY
		} else {
			sizeY += deterministicSigned(effect, salt+9) * component.sizeRandY
		}
	}
	return sizeX, sizeY
}

func effectAxisSize(progress, start, end float64, smooth bool) float64 {
	if start <= 0 && end > 0 {
		start = end
	}
	if end <= 0 && start > 0 {
		end = start
	}
	if smooth {
		factor := math.Log10(progress*9 + 1)
		return start + (end-start)*factor
	}
	return start + (end-start)*progress
}

func drawTexturedEffectBillboardRotatedXY(screen *render.Frame, projection sceneProjection, texture *render.Image, worldX, worldY, worldZ, sizeX, sizeY, angle float64, tint color.RGBA, additive bool) {
	drawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, worldX, worldY, worldZ, sizeX, sizeY, angle, tint, texturedEffectBillboardDrawOptions(additive, false))
}

func texturedEffectBillboardDrawOptions(additive, overlay bool) *render.DrawTrianglesOptions {
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	if additive {
		options.Blend = render.BlendLighter
	}
	if overlay {
		options.DepthTest = false
	}
	return options
}

func drawTexturedEffectBillboardRotatedXYWithOptions(screen *render.Frame, projection sceneProjection, texture *render.Image, worldX, worldY, worldZ, sizeX, sizeY, angle float64, tint color.RGBA, options *render.DrawTrianglesOptions) {
	if screen == nil || texture == nil || tint.A == 0 {
		return
	}
	right, up, _, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	center := modelPoint3{x: worldX, y: worldZ, z: worldY}
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	axisScaleX := sizeX / float64(w)
	axisScaleY := sizeY / float64(h)
	rightAxis := mul3(right, axisScaleX)
	upAxis := mul3(up, -axisScaleY)
	if angle != 0 {
		sinA, cosA := math.Sin(angle), math.Cos(angle)
		rightAxis = add3(mul3(right, cosA*axisScaleX), mul3(up, -sinA*axisScaleX))
		upAxis = add3(mul3(right, -sinA*axisScaleY), mul3(up, -cosA*axisScaleY))
	}
	if options == nil {
		options = texturedEffectBillboardDrawOptions(false, false)
	}
	screen.DrawWorldBillboard(render.WorldBillboardCommand{
		Texture:     texture,
		Options:     *options,
		Center:      [3]float32{float32(center.x), float32(center.y), float32(center.z)},
		RightAxis:   [3]float32{float32(rightAxis.x), float32(rightAxis.y), float32(rightAxis.z)},
		UpAxis:      [3]float32{float32(upAxis.x), float32(upAxis.y), float32(upAxis.z)},
		DepthUpAxis: [3]float32{float32(upAxis.x), float32(upAxis.y), float32(upAxis.z)},
		Width:       w,
		Height:      h,
		AnchorX:     w * 0.5,
		AnchorY:     h * 0.5,
		ColorR:      float32(tint.R) / 255,
		ColorG:      float32(tint.G) / 255,
		ColorB:      float32(tint.B) / 255,
		ColorA:      float32(tint.A) / 255,
	})
}
