package game

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *WorldMode) draw3DSpriteEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, worldX, worldY, worldZ float64, size float64, alpha float64, starts time.Time, now time.Time) {
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 {
		return
	}
	actionIndex := 0
	action := view.act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return
	}
	delayMS := float64(action.DelayMS)
	if component.spriteDelay > 0 {
		delayMS = float64(component.spriteDelay / time.Millisecond)
	}
	motion := 0
	if component.spriteRepeat {
		motion = spriteMotionIndexWithDelay(action, starts, now, true, delayMS)
	} else {
		motion = spriteMotionIndexWithDelay(action, starts, now, false, delayMS)
	}
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	ignoreLayerAngles := component.rotateToTarget
	key := singleSpriteBillboardKey{actionIndex: actionIndex, motion: motion, ignoreLayerAngles: ignoreLayerAngles}
	billboard, ok := view.billboards[key]
	if !ok {
		var baseOK bool
		billboard, baseOK = composeSingleSpriteBillboardWithOptions(view, action.Animations[motion], ignoreLayerAngles)
		if !baseOK {
			return
		}
		view.billboards[key] = billboard
	}
	tint := effectComponentTint(component, 1)
	if component.worldSizedSprite {
		scale := size / 100
		angle := -worldEffectSpriteAngle(component) * math.Pi / 180
		options := spriteBillboardTriangleDrawOptions()
		if component.blendAdditive {
			options.Blend = render.BlendLighter
		}
		drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, scale, angle, alpha, 1, tint, options)
		return
	}
	scale := effect3DSpriteScale(size)
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	options := effect3DSpriteDrawOptions(component)
	if angle, ok := effectSpriteScreenRotation(ctx, projection, component, effect); ok {
		drawSpriteBillboardTintAlphaRotated3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, scale, angle, alpha, 1, tint, options)
		return
	}
	drawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, 1, tint, options)
}

func effect3DSpriteDrawOptions(component worldEffectComponent) *render.DrawTrianglesOptions {
	options := spriteBillboardTriangleDrawOptions()
	if component.blendAdditive {
		options.Blend = render.BlendLighter
	}
	return options
}

func effect3DSpriteScale(size float64) float64 {
	if size <= 0 || math.IsNaN(size) || math.IsInf(size, 0) {
		return 1
	}
	// reference client's SpriteRenderer applies size as (size / 175) * xSize, with
	// xSize defaulting to 5. effectTableSize already stores that scale.
	return size
}

func effectSpriteScreenRotation(ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect) (float64, bool) {
	if component.rotateToTarget {
		startX, startY, startZ, endX, endY, endZ, ok := effectTrajectoryEndpoints(ctx, component, effect)
		if ok {
			start := projection.Project(startX, startY, startZ)
			end := projection.Project(endX, endY, endZ)
			dx := float64(end.x - start.x)
			dy := float64(end.y - start.y)
			if math.Hypot(dx, dy) > 0.001 {
				return math.Atan2(dy, dx) - math.Pi/2, true
			}
		}
	}
	angle := worldEffectBillboardAngle(component, projection, 0)
	if math.Abs(angle) > 0.0001 {
		return angle, true
	}
	return 0, false
}

func effectTrajectoryEndpoints(ctx client.Context, component worldEffectComponent, effect worldEffect) (float64, float64, float64, float64, float64, float64, bool) {
	if ctx.World == nil {
		return 0, 0, 0, 0, 0, 0, false
	}
	x, y, ok := effectAnchor(ctx, effect.actorID)
	if !ok {
		return 0, 0, 0, 0, 0, 0, false
	}
	worldX := cellCenter(float64(x))
	worldY := cellCenter(float64(y))
	worldZ := terrainHeightAt(ctx.World, float64(x), float64(y)) + 0.07
	startX := component.posX + component.posXStartMiddle
	startY := component.posY + component.posYStartMiddle
	startZ := component.posZ + component.posZStartMiddle
	endX := component.posXEnd + component.posXEndMiddle
	endY := component.posYEnd + component.posYEndMiddle
	endZ := component.posZEnd + component.posZEndMiddle
	if component.posXEnd == 0 && component.posXEndMiddle == 0 && component.posXStartMiddle == 0 {
		endX = startX
	}
	if component.posYEnd == 0 && component.posYEndMiddle == 0 && component.posYStartMiddle == 0 {
		endY = startY
	}
	if component.posZEnd == 0 && component.posZEndMiddle == 0 {
		endZ = startZ
	}
	if component.fromSrc || component.toSrc {
		otherX, otherY, otherZ, otherOK := effectOtherEndpoint(ctx, effect, worldX, worldY, worldZ)
		if otherOK {
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
	return worldX + startX, worldY + startY, worldZ + startZ, worldX + endX, worldY + endY, worldZ + endZ, true
}

func worldEffectSpriteAngle(component worldEffectComponent) float64 {
	angle := component.angleStart
	if !component.rotateToTarget {
		return angle
	}
	startX, startY := component.posX, component.posY
	endX, endY := component.posXEnd, component.posYEnd
	if component.posXEnd == 0 && component.posXEndRand == 0 {
		endX = startX
	}
	if component.posYEnd == 0 && component.posYEndRand == 0 {
		endY = startY
	}
	return angle + 90 - math.Atan2(endY-startY, endX-startX)*180/math.Pi
}

func (m *WorldMode) drawSPREffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, worldX, worldY, worldZ float64, now time.Time) {
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 {
		return
	}
	actionIndex := component.spriteFrame
	if effect.hasSpriteFrame {
		actionIndex = effect.spriteFrameOverride
	}
	if component.spriteDirection {
		if actor, ok := ctx.World.Actors[effect.actorID]; ok {
			actionIndex = actorRenderDirection(actor, now) % len(view.act.Actions)
		} else if isLocalActor(ctx, effect.actorID) {
			actionIndex = actorRenderDirection(ctx.World.Player, now) % len(view.act.Actions)
		}
	}
	if actionIndex < 0 || actionIndex >= len(view.act.Actions) {
		actionIndex = 0
	}
	action := view.act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return
	}
	delayMS := float64(action.DelayMS)
	if component.spriteDelay > 0 {
		delayMS = float64(component.spriteDelay / time.Millisecond)
	}
	motion := 0
	if component.spriteRepeat {
		motion = spriteMotionIndexWithDelay(action, effect.starts, now, true, delayMS)
	} else {
		motion = spriteMotionIndexWithDelay(action, effect.starts, now, false, delayMS)
		if motion >= len(action.Animations)-1 && !component.spriteStopAtEnd && component.duration <= 0 {
			return
		}
	}
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	key := singleSpriteBillboardKey{
		actionIndex: actionIndex,
		motion:      motion,
		anchorX:     component.spriteXOffset,
		anchorY:     component.spriteYOffset,
	}
	billboard, ok := view.billboards[key]
	if !ok {
		base, baseOK := composeSingleSpriteBillboard(view, action.Animations[motion])
		if !baseOK {
			return
		}
		copy := *base
		copy.anchorX -= component.spriteXOffset
		copy.anchorY -= component.spriteYOffset
		billboard = &copy
		view.billboards[key] = billboard
	}
	z := worldZ + component.posZ
	if component.spriteHead {
		z += 2.0
	}
	if component.worldSizedSprite {
		drawSpriteBillboardTintAlphaWorld3D(screen, projection, billboard, worldX, worldY, z, effectPixelRatio, 0, 1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		return
	}
	scale := 1.0
	if component.attachedEntity {
		scale = actorBillboardScreenScale(projection, worldX, worldY, worldZ)
	}
	drawSpriteBillboardTintAlpha3D(screen, projection, billboard, worldX, worldY, z, scale, 1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

const effectSpriteRoot = "data\\sprite\\이팩트\\"

func (m *WorldMode) effectSpriteView(manager *res.Manager, file string) *spriteView {
	file = strings.TrimSpace(file)
	if manager == nil || file == "" {
		return nil
	}
	if m.effectViews == nil {
		m.effectViews = make(map[string]*spriteView)
	}
	if m.effectViewMiss == nil {
		m.effectViewMiss = make(map[string]struct{})
	}
	key := strings.ReplaceAll(file, "/", "\\")
	if view, ok := m.effectViews[key]; ok {
		return view
	}
	if _, ok := m.effectViewMiss[key]; ok {
		return nil
	}
	actCandidates := effectSpriteResourceCandidates(file, "act")
	sprCandidates := effectSpriteResourceCandidates(file, "spr")
	view, status := loadSpriteView(manager, actCandidates, sprCandidates, nil, "effect sprite "+file)
	if view == nil {
		m.effectViewMiss[key] = struct{}{}
		glog.Warnf("effect sprite unavailable file=%q: %s", file, status)
		return nil
	}
	m.effectViews[key] = view
	glog.Debugf("effect sprite resources file=%q %s", file, status)
	return view
}

func effectSpriteResourceCandidates(file, ext string) []string {
	normalized := strings.TrimSpace(strings.ReplaceAll(file, "/", "\\"))
	normalized = strings.TrimSuffix(normalized, ".spr")
	normalized = strings.TrimSuffix(normalized, ".act")
	if normalized == "" {
		return nil
	}
	base := normalized
	if !strings.HasPrefix(strings.ToLower(base), "data\\sprite\\") {
		base = effectSpriteRoot + base
	}
	path := base + "." + ext
	return []string{path, strings.ReplaceAll(path, "\\", "/")}
}
