package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

type effectFuncAdapter int

const (
	effectFuncUnknown effectFuncAdapter = iota
	effectFuncGroundSample
	effectFuncCastRing
	effectFuncLockOnTarget
	effectFuncLevel99Aura
	effectFuncGroundAura
	effectFuncLevel99Bubble
	effectFuncPropertyGround
	effectFuncLandProtectorGround
	effectFuncSpiritSphere
	effectFuncFlatColorTile
	effectFuncGroundTexture
	effectFuncBodyColor
	effectFuncMapPillar
	effectFuncHitRing
)

const (
	groundSampleFallbackRotationRadiansPerSecond = 40 * math.Pi / 180
	groundTextureHoverPeriodSeconds              = 10.8
)

func (m *WorldMode) drawFuncEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, componentIndex int, worldX, worldY, worldZ, progress float64, now time.Time) {
	switch component.funcAdapter {
	case effectFuncGroundSample:
		m.drawGroundPlaneEffect(screen, ctx, component, effect, worldX, worldY, progress, now)
	case effectFuncCastRing:
		m.drawCastRingEffect(screen, ctx, component, effect, componentIndex, worldX, worldY, worldZ, progress)
	case effectFuncLockOnTarget:
		m.drawLockOnTargetEffect(screen, ctx, component, effect, worldX, worldY, worldZ, progress, now)
	case effectFuncLevel99Aura:
		m.drawLevel99AuraEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGroundAura:
		m.drawGroundAuraEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncLevel99Bubble:
		m.drawLevel99BubbleEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncPropertyGround:
		m.drawPropertyGroundEffect(screen, ctx, component, effect, componentIndex, worldX, worldY, worldZ, now)
	case effectFuncLandProtectorGround:
		m.drawLandProtectorGroundEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncSpiritSphere:
		m.drawSpiritSphereEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncFlatColorTile:
		m.drawFlatColorTileEffect(screen, component, worldX, worldY, worldZ)
	case effectFuncGroundTexture:
		m.drawGroundTextureEffect(screen, ctx, component, effect, componentIndex, worldX, worldY, worldZ, progress, now)
	case effectFuncMapPillar:
		m.drawMapPillarEffect(screen, ctx, component, worldX, worldY, worldZ, progress)
	case effectFuncHitRing:
		m.drawHitRingEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	default:
	}
}

func (m *WorldMode) drawCastRingEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, componentIndex int, x, y, z, progress float64) {
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.textureName)
	tint := effectComponentTint(component, alpha)
	z += component.posZ
	angleOffset := 0.0
	if component.rotate {
		angleOffset = progress * 2 * math.Pi
		angleOffset += deterministicAngle(effect, componentIndex+17) * 0.08
	}
	drawWorldSoftRing(screen, m.whitePixel, x, y, z+0.015, component.bottomSize*1.15, 0.16, withAlpha(component.color, alpha*0.75), maxInt(component.circleSides, 20))
	if texture != nil {
		drawWorldCylinderBandRotated(screen, m.whitePixel, texture, x, y, z+0.035, component.bottomSize, component.topSize, component.height, tint, maxInt(component.circleSides, component.totalCircleSides), angleOffset)
		return
	}
	drawWorldCylinderBandRotated(screen, m.whitePixel, nil, x, y, z+0.035, component.bottomSize, component.topSize, component.height, tint, maxInt(component.circleSides, component.totalCircleSides), angleOffset)
}

func drawWorldCylinderBandRotated(screen *render.Frame, white, texture *render.Image, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int, angleOffset float64) {
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
		u := float32(i) / float32(segments)
		angle := angleOffset + float64(i)*2*math.Pi/float64(segments)
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		topAngle := angleOffset + (float64(i)+0.5)*2*math.Pi/float64(segments)
		topCosine := math.Cos(topAngle)
		topSine := math.Sin(topAngle)
		vertices = append(vertices,
			warpEffectTexturedVertex3D(x+cosine*bottomRadius, y+sine*bottomRadius, z, u*srcW, srcH, tint),
			warpEffectTexturedVertex3D(x+topCosine*topRadius, y+topSine*topRadius, z+height, u*srcW, 0, tint),
		)
		if i == segments {
			continue
		}
		base := uint16(i * 2)
		indices = append(indices, base, base+1, base+3, base, base+3, base+2)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, source, options)
}

func (m *WorldMode) drawGroundPlaneEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, worldX, worldY, progress float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil || ctx.World == nil {
		return
	}
	alpha := effectBillboardAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	size := component.sizeStart
	if effect.size > 0 {
		size = effect.size
	}
	if size <= 0 {
		size = 1
	}
	half := size * 0.5
	angle := groundSampleRotationAngle(effect, now)
	uv := func(u, v float64) texturePoint {
		sinA, cosA := math.Sin(angle), math.Cos(angle)
		x, y := u-0.5, v-0.5
		return texturePoint{
			u: float32(x*cosA - y*sinA + 0.5),
			v: float32(x*sinA + y*cosA + 0.5),
		}
	}
	point := func(dx, dy float64) modelPoint3 {
		x := worldX + dx
		y := worldY + dy
		return modelPoint3{x: x, y: terrainHeightAt(ctx.World, x-0.5, y-0.5) + component.posZ, z: y}
	}
	verts := [4]modelPoint3{
		point(-half, -half),
		point(half, -half),
		point(half, half),
		point(-half, half),
	}
	uvs := [4]texturePoint{
		uv(0, 0),
		uv(1, 0),
		uv(1, 1),
		uv(0, 1),
	}
	tint := effectComponentTint(component, alpha)
	drawTexturedSurface3DWithOptions(screen, texture, verts, uvs, quadIndices012023, [4]color.RGBA{tint, tint, tint, tint}, groundSampleDrawOptions())
}

func groundSampleDrawOptions() *render.DrawTrianglesOptions {
	return triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
}

func groundSampleRotationAngle(effect worldEffect, now time.Time) float64 {
	elapsed := math.Max(0, now.Sub(effect.starts).Seconds())
	speed := effect.groundSampleRotationRadiansPerSecond
	if speed <= 0 {
		speed = groundSampleFallbackRotationRadiansPerSecond
	}
	return elapsed * speed
}

func (m *WorldMode) drawLockOnTargetEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, worldX, worldY, worldZ float64, progress float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	size := lockOnTargetSize(effect.starts, now)
	half := size * 0.5
	angle := float64(now.UnixMilli())*math.Pi/720 + math.Pi/4
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	point := func(dx, dy float64) modelPoint3 {
		x := dx*cosA - dy*sinA
		y := dx*sinA + dy*cosA
		return modelPoint3{x: worldX + x, y: worldZ + component.posZ, z: worldY + y}
	}
	verts := [4]modelPoint3{
		point(-half, -half),
		point(half, -half),
		point(half, half),
		point(-half, half),
	}
	uvs := [4]texturePoint{
		{u: 0, v: 0},
		{u: 1, v: 0},
		{u: 1, v: 1},
		{u: 0, v: 1},
	}
	tint := lockOnTargetTint(effect.starts, now)
	tint.A = uint8(clampFloat(alpha, 0, 1) * 255)
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	drawTexturedSurface3DWithOptions(screen, texture, verts, uvs, quadIndices012023, [4]color.RGBA{tint, tint, tint, tint}, options)
}

func lockOnTargetSize(starts, now time.Time) float64 {
	elapsed := now.Sub(starts).Seconds() * 1000 / 50
	elapsed = clampFloat(elapsed, 1, 5)
	return (6 - elapsed) * 3
}

func lockOnTargetTint(starts, now time.Time) color.RGBA {
	elapsed := int(now.Sub(starts) / (20 * time.Millisecond))
	if elapsed < 0 {
		elapsed = 0
	}
	factor := float64(20-(elapsed%20)) / 20
	gb := uint8(clampFloat(factor, 0, 1) * 255)
	return color.RGBA{R: 255, G: gb, B: gb, A: 255}
}

func (m *WorldMode) drawLevel99AuraEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	elapsedFrames := math.Max(0, now.Sub(effect.starts).Seconds()*60)
	build := math.Sin(math.Min(elapsedFrames, 90) * math.Pi / 180)
	if build <= 0 {
		return
	}
	tint := color.RGBA{R: 100, G: 100, B: 255, A: 120}
	for band := 0; band < 3; band++ {
		drawLevel99AuraBand(screen, m.whitePixel, texture, x, y, z+0.02, band, elapsedFrames, build, tint)
	}
}

func drawLevel99AuraBand(screen *render.Frame, white, texture *render.Image, x, y, z float64, band int, elapsedFrames, build float64, tint color.RGBA) {
	const (
		divisions        = 21
		fullDisplayAngle = 315.0
		gameToWorld      = 0.1 * 2.2
		innerCircleScale = 0.6
	)
	if screen == nil || white == nil {
		return
	}
	source := white
	srcW, srcH := float32(1), float32(1)
	if texture != nil {
		source = texture
		bounds := texture.Bounds()
		srcW = float32(bounds.Dx())
		srcH = float32(bounds.Dy())
	}
	maxHeight := float64(15-2*band) * gameToWorld
	distance := (3.9 + 0.2*float64(band)) * gameToWorld * innerCircleScale
	riseAngle := float64(55-5*band) * math.Pi / 180
	rotStart := float64(band)*90 + elapsedFrames*float64(band+3)
	basicAngle := fullDisplayAngle / float64(divisions-1)
	cosRise := math.Cos(riseAngle)
	sinRise := math.Sin(riseAngle)
	center := modelPoint3{x: x, y: z, z: y}
	vertices := make([]render.Vertex3D, 0, divisions*2)
	indices := make([]uint16, 0, (divisions-1)*6)
	for k := 0; k < divisions; k++ {
		angle := (rotStart + float64(k)*basicAngle) * math.Pi / 180
		cosAngle := math.Cos(angle)
		sinAngle := math.Sin(angle)
		sinLimit := math.Sin((90 + float64(k-10)*9) * math.Pi / 180)
		height := math.Max(0, maxHeight*sinLimit*build)
		radialRise := cosRise * height
		verticalRise := sinRise * height
		u := float32(k) / float32(divisions-1)
		base := modelPoint3{
			x: center.x + distance*cosAngle,
			y: center.y,
			z: center.z + distance*sinAngle,
		}
		top := modelPoint3{
			x: center.x + (distance+radialRise)*cosAngle,
			y: center.y + verticalRise,
			z: center.z + (distance+radialRise)*sinAngle,
		}
		vertices = append(vertices,
			texturedSurfaceVertex3D(base, texturePoint{u: u, v: 1}, tint, srcW, srcH),
			texturedSurfaceVertex3D(top, texturePoint{u: u, v: 0}, tint, srcW, srcH),
		)
		if k == divisions-1 {
			continue
		}
		baseIndex := uint16(k * 2)
		indices = append(indices, baseIndex, baseIndex+1, baseIndex+2, baseIndex+1, baseIndex+3, baseIndex+2)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, source, options)
}

func (m *WorldMode) drawGroundAuraEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	elapsed := now.Sub(effect.starts).Seconds()
	sizes := []float64{component.sizeStart, component.sizeEnd}
	for i, size := range sizes {
		if size <= 0 {
			continue
		}
		phase := elapsed*6 + float64(i)*23*math.Pi/180
		size += math.Sin(phase) * 0.08
		drawGroundTextureQuad(screen, texture, x, y, z+component.posZ+0.01*float64(i), size, size, float64(i)*23*math.Pi/180, color.RGBA{R: 255, G: 255, B: 255, A: 204}, true)
	}
}

func (m *WorldMode) drawLevel99BubbleEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	elapsedFrames := math.Max(0, now.Sub(effect.starts).Seconds()*60)
	size := component.sizeStart
	if size <= 0 {
		size = 2.4 * level99BubbleGameToWorld * 0.6 * 2
	}
	for column := 0; column < level99BubbleColumns; column++ {
		for anchor := 0; anchor < level99BubbleAnchors; anchor++ {
			sample := level99BubbleSample(effect, column, anchor, elapsedFrames)
			if !sample.visible || sample.alpha <= 0 {
				continue
			}
			tint := component.color
			if tint.A == 0 {
				tint = color.RGBA{R: 80, G: 80, B: 255, A: 250}
			}
			tint.A = uint8(sample.alpha * 255)
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, x+sample.offsetX, y+sample.offsetY, z+component.posZ+sample.height, size, size, 0, tint, true)
		}
	}
}

const (
	level99BubbleGameToWorld = 0.1 * 2.2
	level99BubbleColumns     = 4
	level99BubbleAnchors     = 4
	level99BubbleSeedMax     = 99.0
	level99BubbleResetY      = -30.0
	level99BubbleRiseSpeed   = 0.15
	level99BubbleDrift       = 0.15
	level99BubblePhaseXRate  = 0.045
	level99BubblePhaseYRate  = 0.052
)

type level99BubbleParticle struct {
	visible bool
	offsetX float64
	offsetY float64
	height  float64
	alpha   float64
}

func level99BubbleSample(effect worldEffect, column, anchor int, elapsedFrames float64) level99BubbleParticle {
	elapsedFrames = math.Max(0, elapsedFrames)
	salt := column*97 + anchor*31
	seed := deterministicUnit(effect, salt+501) * level99BubbleSeedMax
	cycleSpan := level99BubbleSeedMax - level99BubbleResetY
	cycle := math.Mod(elapsedFrames*level99BubbleRiseSpeed+seed, cycleSpan)
	clientY := level99BubbleSeedMax - cycle
	if clientY > 0 {
		return level99BubbleParticle{}
	}
	heightUnits := -clientY
	visibleFrames := heightUnits / level99BubbleRiseSpeed
	visibleStartFrame := elapsedFrames - visibleFrames
	phaseX := visibleStartFrame*level99BubblePhaseXRate + deterministicUnit(effect, salt+601)*2*math.Pi
	phaseY := visibleStartFrame*level99BubblePhaseYRate + deterministicUnit(effect, salt+701)*2*math.Pi
	signX, signY := level99BubbleDriftSigns(anchor)
	return level99BubbleParticle{
		visible: true,
		offsetX: signX * level99BubbleIntegratedDrift(phaseX, level99BubblePhaseXRate, visibleFrames) *
			level99BubbleGameToWorld,
		offsetY: signY * level99BubbleIntegratedDrift(phaseY, level99BubblePhaseYRate, visibleFrames) *
			level99BubbleGameToWorld,
		height: heightUnits * level99BubbleGameToWorld,
		alpha:  level99BubbleAlpha(clientY),
	}
}

func level99BubbleDriftSigns(anchor int) (float64, float64) {
	switch anchor % level99BubbleAnchors {
	case 1:
		return -1, -1
	case 2:
		return 1, -1
	case 3:
		return -1, 1
	default:
		return 1, 1
	}
}

func level99BubbleIntegratedDrift(phase, rate, frames float64) float64 {
	if frames <= 0 || rate == 0 {
		return 0
	}
	return level99BubbleDrift / rate * (math.Cos(phase) - math.Cos(phase+rate*frames))
}

func level99BubbleAlpha(clientY float64) float64 {
	alpha := 250.0
	if clientY < -20 {
		alpha = 250 + (clientY+20)*25
	}
	return clampFloat(alpha/255, 0, 250.0/255.0)
}

func (m *WorldMode) drawPropertyGroundEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, componentIndex int, x, y, z float64, now time.Time) {
	texture := m.effectTexture(ctx.Resources, component.textureName)
	if texture == nil {
		return
	}
	elapsed := now.Sub(effect.starts).Seconds()
	phase := elapsed*2 + deterministicAngle(effect, componentIndex+811)
	sizeMult := math.Sin(phase)
	if sizeMult < 0.5 {
		sizeMult = 0.5
	}
	tint := effectComponentTint(component, 1)
	drawWorldCylinderBandRotated(screen, m.whitePixel, texture, x, y, z+0.05, component.bottomSize*sizeMult, component.topSize*sizeMult, component.height, tint, maxInt(component.circleSides, component.totalCircleSides), phase)
}

const (
	mapPillarBands          = 4
	mapPillarSegments       = 20
	mapPillarCycleFrames    = 960.0
	mapPillarGrowStartFrame = 200.0
	mapPillarGrowEndFrame   = 290.0
	mapPillarShrinkFrame    = 800.0
	mapPillarWorldScale     = 0.2
	mapPillarBandRadiusStep = 0.5 * mapPillarWorldScale
	mapPillarRiseRadians    = 89.0 * math.Pi / 180.0
)

func (m *WorldMode) drawMapPillarEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, x, y, z, progress float64) {
	texture := m.effectTexture(ctx.Resources, component.textureName)
	if texture == nil || component.height <= 0 || component.bottomSize <= 0 {
		return
	}
	tint := effectComponentTint(component, component.alphaMax)
	frames := clampFloat(progress, 0, 0.999999) * mapPillarCycleFrames
	for band := 0; band < mapPillarBands; band++ {
		process := frames + float64(band)*30
		height := mapPillarBandHeight(process, component.height)
		if height <= 0 {
			continue
		}
		radius := component.bottomSize + float64(band)*mapPillarBandRadiusStep
		rotation := float64(band)*90 + frames
		drawMapPillarBand(screen, texture, x, y, z, radius, height, rotation, tint)
	}
}

func mapPillarBandHeight(process, maxHeight float64) float64 {
	if process < mapPillarGrowStartFrame || maxHeight <= 0 {
		return 0
	}
	if process >= mapPillarShrinkFrame {
		maxHeight -= (process - (mapPillarShrinkFrame - 1)) * mapPillarWorldScale
		if maxHeight <= 0 {
			return 0
		}
	}
	if process <= mapPillarGrowEndFrame {
		return maxHeight * math.Sin((process-mapPillarGrowStartFrame)*math.Pi/180)
	}
	return maxHeight
}

func drawMapPillarBand(screen *render.Frame, texture *render.Image, x, y, z, radius, height, rotationDegrees float64, tint color.RGBA) {
	if screen == nil || texture == nil || radius <= 0 || height <= 0 || tint.A == 0 {
		return
	}
	bounds := texture.Bounds()
	srcW, srcH := float32(bounds.Dx()), float32(bounds.Dy())
	center := modelPoint3{x: x, y: z, z: y}
	radialRise := math.Cos(mapPillarRiseRadians) * height
	verticalRise := math.Sin(mapPillarRiseRadians) * height
	vertices := make([]render.Vertex3D, 0, (mapPillarSegments+1)*2)
	indices := make([]uint16, 0, mapPillarSegments*6)
	for i := 0; i <= mapPillarSegments; i++ {
		u := float32(i) / float32(mapPillarSegments)
		angle := (rotationDegrees + float64(i)*360/float64(mapPillarSegments)) * math.Pi / 180
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		base := modelPoint3{
			x: center.x + radius*cosine,
			y: center.y,
			z: center.z + radius*sine,
		}
		top := modelPoint3{
			x: center.x + (radius+radialRise)*cosine,
			y: center.y + verticalRise,
			z: center.z + (radius+radialRise)*sine,
		}
		vertices = append(vertices,
			texturedSurfaceVertex3D(base, texturePoint{u: u, v: 1}, tint, srcW, srcH),
			texturedSurfaceVertex3D(top, texturePoint{u: u, v: 0}, tint, srcW, srcH),
		)
		if i == mapPillarSegments {
			continue
		}
		baseIndex := uint16(i * 2)
		indices = append(indices, baseIndex, baseIndex+1, baseIndex+2, baseIndex+1, baseIndex+3, baseIndex+2)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, texture, options)
}

func (m *WorldMode) drawLandProtectorGroundEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	elapsed := now.Sub(effect.starts).Seconds()
	size := component.sizeStart + (component.sizeEnd-component.sizeStart)*(0.5+0.5*math.Sin(elapsed*4))
	if size <= 0 {
		size = 0.8
	}
	drawGroundTextureQuad(screen, texture, x, y, z+component.posZ, size, size, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255}, true)
}

func (m *WorldMode) drawFlatColorTileEffect(screen *render.Frame, component worldEffectComponent, x, y, z float64) {
	size := component.sizeStart
	if size <= 0 {
		size = 1
	}
	tint := component.color
	if tint.A == 0 {
		tint.A = 128
	}
	drawGroundTextureQuad(screen, m.whitePixel, x, y, z+component.posZ+0.02, size, size, 0, tint, false)
}

func (m *WorldMode) drawGroundTextureEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, componentIndex int, x, y, z, progress float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	size := component.sizeStart
	if size <= 0 {
		size = 1
	}
	if component.sizeEnd > 0 && component.sizeEnd != size {
		size += (component.sizeEnd - size) * progress
	}
	alpha := effectComponentAlpha(progress, component)
	tint := effectComponentTint(component, alpha)
	zOffset := groundTextureZOffset(component, effect, componentIndex, now)
	drawGroundTextureQuad(screen, texture, x, y, z+zOffset, size, size, component.angleStart, tint, component.blendAdditive)
}

func groundTextureZOffset(component worldEffectComponent, effect worldEffect, componentIndex int, now time.Time) float64 {
	if component.posZEnd == 0 || component.posZEnd == component.posZ || now.IsZero() {
		return component.posZ
	}
	elapsed := math.Max(0, now.Sub(effect.starts).Seconds())
	phase := deterministicAngle(effect, componentIndex+733) + elapsed*2*math.Pi/groundTextureHoverPeriodSeconds
	return component.posZ + (component.posZEnd-component.posZ)*(0.5+0.5*math.Sin(phase))
}

func (m *WorldMode) drawSpiritSphereEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	count := maxInt(component.duplicate, 5)
	elapsedFrames := math.Max(0, now.Sub(effect.starts).Seconds()*60)
	alpha := clampFloat(elapsedFrames*0.005, 0, 1)
	for i := 0; i < count; i++ {
		angle := elapsedFrames*math.Pi/180 + float64(i)*2*math.Pi/float64(count)
		radius := 0.55
		worldX := x + math.Cos(angle)*radius
		worldY := y + math.Sin(angle)*radius
		worldZ := z + 1.2 + math.Sin(angle*0.5)*0.12
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, worldX, worldY, worldZ, 0.35, 0.35, angle, color.RGBA{R: 0, G: 0, B: 255, A: uint8(153 * alpha)}, true)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, worldX, worldY, worldZ+0.02, 0.21, 0.21, -angle, color.RGBA{R: 204, G: 204, B: 255, A: uint8(255 * alpha)}, true)
	}
}

func drawGroundTextureQuad(screen *render.Frame, texture *render.Image, x, y, z, sizeX, sizeY, angle float64, tint color.RGBA, additive bool) {
	if screen == nil || texture == nil || sizeX <= 0 || sizeY <= 0 || tint.A == 0 {
		return
	}
	halfX := sizeX * 0.5
	halfY := sizeY * 0.5
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	point := func(dx, dy float64) modelPoint3 {
		px := dx*cosA - dy*sinA
		py := dx*sinA + dy*cosA
		return modelPoint3{x: x + px, y: z, z: y + py}
	}
	verts := [4]modelPoint3{
		point(-halfX, -halfY),
		point(halfX, -halfY),
		point(halfX, halfY),
		point(-halfX, halfY),
	}
	uvs := [4]texturePoint{{u: 0, v: 0}, {u: 1, v: 0}, {u: 1, v: 1}, {u: 0, v: 1}}
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	if additive {
		options.Blend = render.BlendLighter
	}
	drawTexturedSurface3DWithOptions(screen, texture, verts, uvs, quadIndices012023, [4]color.RGBA{tint, tint, tint, tint}, options)
}

func warpEffectTexturedVertex3D(x, y, z float64, srcX, srcY float32, c color.RGBA) render.Vertex3D {
	point := modelPoint3{x: x, y: z, z: y}
	return render.Vertex3D{
		X:      float32(point.x),
		Y:      float32(point.y),
		Z:      float32(point.z),
		SrcX:   srcX,
		SrcY:   srcY,
		ColorR: float32(c.R) / 255,
		ColorG: float32(c.G) / 255,
		ColorB: float32(c.B) / 255,
		ColorA: float32(c.A) / 255,
		DepthX: float32(point.x),
		DepthY: float32(point.y),
		DepthZ: float32(point.z),
	}
}

func warpEffectVertex3D(x, y, z float64, c color.RGBA) render.Vertex3D {
	return warpEffectTexturedVertex3D(x, y, z, 0, 0, c)
}

func drawWorldSoftRing(screen *render.Frame, white *render.Image, x, y, z, radius, width float64, c color.RGBA, segments int) {
	inner := math.Max(0, radius-width*0.5)
	mid := math.Max(inner+0.01, radius)
	outer := math.Max(mid+0.01, radius+width*0.5)
	drawWorldRingBand(screen, white, x, y, z, inner, mid, 0, c.A, c, segments)
	drawWorldRingBand(screen, white, x, y, z, mid, outer, c.A, 0, c, segments)
}

func drawWorldRingBand(screen *render.Frame, white *render.Image, x, y, z, innerRadius, outerRadius float64, innerAlpha, outerAlpha uint8, c color.RGBA, segments int) {
	if segments < 3 || outerRadius <= innerRadius {
		return
	}
	vertices := make([]render.Vertex3D, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)
	innerColor := c
	outerColor := c
	innerColor.A = innerAlpha
	outerColor.A = outerAlpha
	for i := 0; i <= segments; i++ {
		angle := float64(i) * 2 * math.Pi / float64(segments)
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		vertices = append(vertices,
			warpEffectVertex3D(x+cosine*innerRadius, y+sine*innerRadius, z, innerColor),
			warpEffectVertex3D(x+cosine*outerRadius, y+sine*outerRadius, z, outerColor),
		)
		if i == segments {
			continue
		}
		base := uint16(i * 2)
		indices = append(indices, base, base+1, base+3, base, base+3, base+2)
	}
	options := triangleDrawOptions(render.FilterNearest, render.AddressUnsafe)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, white, options)
}
