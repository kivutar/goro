package effects

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *System) draw3DSpriteEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, effect Instance, component Component, worldX, worldY, worldZ float64, size float64, alpha float64, progress float64, starts time.Time, now time.Time) {
	view := m.EffectSpriteView(ctx.Resources, component.SpriteFile)
	if view == nil || len(view.Act.Actions) == 0 {
		return
	}
	actionIndex := 0
	action := view.Act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return
	}
	delayMS := float64(action.DelayMS)
	if component.SpriteDelay > 0 {
		delayMS = float64(component.SpriteDelay) / float64(time.Millisecond)
	}
	motion := 0
	if component.SpriteRepeat {
		motion = scene.SpriteMotionIndexWithDelay(action, starts, now, true, delayMS)
	} else {
		motion = scene.SpriteMotionIndexWithDelay(action, starts, now, false, delayMS)
	}
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	ignoreLayerAngles := component.RotateToTarget
	key := scene.SpriteBillboardKey{ActionIndex: actionIndex, Motion: motion, IgnoreLayerAngles: ignoreLayerAngles}
	billboard, ok := view.Billboards[key]
	if !ok {
		var baseOK bool
		billboard, baseOK = scene.ComposeSingleSpriteBillboardWithOptions(view, action.Animations[motion], ignoreLayerAngles)
		if !baseOK {
			return
		}
		view.Billboards[key] = billboard
	}
	tint := effectComponentTint(component, 1)
	scale := Effect3DSpriteScale(size)
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = scene.PixelRatio
	}
	options := Effect3DSpriteDrawOptions(component)
	angle := 0.0
	if rotation, ok := EffectSpriteRobrowserRotation(ctx, projection, component, effect, progress); ok {
		angle = rotation
	}
	scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, scale, angle, alpha, 1, tint, options)
}

func Effect3DSpriteDrawOptions(component Component) *render.DrawTrianglesOptions {
	options := scene.SpriteBillboardTriangleDrawOptions()
	if component.WorldSizedSprite {
		options.DepthBias = StrEffectDepthBias
	}
	if component.BlendAdditive {
		options.Blend = render.BlendLighter
	}
	if component.Overlay {
		options.DepthTest = false
	}
	return options
}

func Effect3DSpriteScale(size float64) float64 {
	if size <= 0 || math.IsNaN(size) || math.IsInf(size, 0) {
		return scene.PixelRatio
	}
	// robr's ThreeDEffect multiplies frame pixels by (layer.scale / 100) * size,
	// then SpriteRenderer converts those pixels to world units through /35.
	// effectTableSize has already applied /35, and ACT layer composition keeps
	// the layer scale in the composed bitmap, so the remaining per-pixel scale is
	// the table size divided by 100.
	return size / 100
}

func EffectSpriteRobrowserRotation(ctx client.Context, projection scene.Projection, component Component, effect Instance, progress float64) (float64, bool) {
	angle := component.AngleStart
	if component.Rotate {
		angle += (component.AngleEnd - component.AngleStart) * progress
	}
	rotated := component.Rotate || component.AngleStart != 0 || component.AngleEnd != 0
	if component.RotateToTarget {
		startX, startY, _, endX, endY, _, ok := EffectTrajectoryEndpoints(ctx, component, effect)
		if ok {
			angle += 90 - math.Atan2(endY-startY, endX-startX)*180/math.Pi
			rotated = true
		}
	}
	if component.RotateWithCamera {
		angle += projection.CameraYaw
		rotated = true
	}
	if !rotated && math.Abs(angle) <= 0.0001 {
		return 0, false
	}
	return -angle * math.Pi / 180, true
}

func EffectTrajectoryEndpoints(ctx client.Context, component Component, effect Instance) (float64, float64, float64, float64, float64, float64, bool) {
	if ctx.World == nil {
		return 0, 0, 0, 0, 0, 0, false
	}
	x, y, ok := EffectAnchor(ctx, effect.ActorID)
	if !ok {
		return 0, 0, 0, 0, 0, 0, false
	}
	worldX := scene.CellCenter(float64(x))
	worldY := scene.CellCenter(float64(y))
	worldZ := scene.TerrainHeightAt(ctx.World, float64(x), float64(y)) + 0.07
	startX := component.PosX + component.posXStartMiddle
	startY := component.PosY + component.posYStartMiddle
	startZ := component.PosZ + component.PosZStartMiddle
	endX := component.PosXEnd + component.posXEndMiddle
	endY := component.PosYEnd + component.posYEndMiddle
	endZ := component.PosZEnd + component.PosZEndMiddle
	if component.PosXEnd == 0 && component.posXEndMiddle == 0 && component.posXStartMiddle == 0 {
		endX = startX
	}
	if component.PosYEnd == 0 && component.posYEndMiddle == 0 && component.posYStartMiddle == 0 {
		endY = startY
	}
	if component.PosZEnd == 0 && component.PosZEndMiddle == 0 {
		endZ = startZ
	}
	if component.fromSrc || component.ToSrc {
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
			if component.ToSrc {
				startX += dx
				startY += dy
				startZ += dz
			}
		}
	}
	return worldX + startX, worldY + startY, worldZ + startZ, worldX + endX, worldY + endY, worldZ + endZ, true
}

func (m *System) DrawSPREffect(screen *render.Frame, ctx client.Context, projection scene.Projection, effect Instance, component Component, worldX, worldY, worldZ float64, now time.Time) {
	view := m.EffectSpriteView(ctx.Resources, component.SpriteFile)
	if view == nil || len(view.Act.Actions) == 0 {
		return
	}
	actionIndex := component.spriteFrame
	if effect.HasSpriteFrame {
		actionIndex = effect.SpriteFrameOverride
	}
	if component.SpriteDirection {
		if actor, ok := ctx.World.Actors[effect.ActorID]; ok {
			actionIndex = scene.ActorRenderDirection(actor, now) % len(view.Act.Actions)
		} else if scene.IsLocalActor(ctx, effect.ActorID) {
			actionIndex = scene.ActorRenderDirection(ctx.World.Player, now) % len(view.Act.Actions)
		}
	}
	if actionIndex < 0 || actionIndex >= len(view.Act.Actions) {
		actionIndex = 0
	}
	action := view.Act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return
	}
	delayMS := float64(action.DelayMS)
	if component.SpriteDelay > 0 {
		delayMS = float64(component.SpriteDelay) / float64(time.Millisecond)
	}
	motion := 0
	if component.SpriteRepeat {
		motion = scene.SpriteMotionIndexWithDelay(action, effect.Starts, now, true, delayMS)
	} else {
		if delayMS <= 0 {
			delayMS = 150
		}
		motion = scene.SpriteMotionIndexWithDelay(action, effect.Starts, now, false, delayMS)
		// The last frame remains visible until its own delay has elapsed.
		if !component.SpriteStopAtEnd && component.Duration <= 0 && float64(now.Sub(effect.Starts))/float64(time.Millisecond) >= float64(len(action.Animations))*delayMS {
			return
		}
	}
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	key := scene.SpriteBillboardKey{
		ActionIndex: actionIndex,
		Motion:      motion,
		AnchorX:     component.spriteXOffset,
		AnchorY:     component.SpriteYOffset,
	}
	billboard, ok := view.Billboards[key]
	if !ok {
		base, baseOK := scene.ComposeSingleSpriteBillboard(view, action.Animations[motion])
		if !baseOK {
			return
		}
		copy := *base
		copy.AnchorX -= component.spriteXOffset
		copy.AnchorY -= component.SpriteYOffset
		billboard = &copy
		view.Billboards[key] = billboard
	}
	z := worldZ + component.PosZ
	if component.SpriteHead {
		z += 2.0
	}
	if component.WorldSizedSprite {
		options := Effect3DSpriteDrawOptions(component)
		scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, worldX, worldY, z, scene.PixelRatio, 0, 1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255}, options)
		return
	}
	scale := 1.0
	if component.AttachedEntity {
		scale = scene.ActorBillboardScreenScale(projection, worldX, worldY, worldZ)
	}
	scene.DrawSpriteBillboardTintAlpha3D(screen, projection, billboard, worldX, worldY, z, scale, 1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

const effectSpriteRoot = "data\\sprite\\이팩트\\"

func (m *System) EffectSpriteView(manager *res.Manager, file string) *scene.SpriteView {
	file = strings.TrimSpace(file)
	if manager == nil || file == "" {
		return nil
	}
	if m.EffectViews == nil {
		m.EffectViews = make(map[string]*scene.SpriteView)
	}
	if m.EffectViewMiss == nil {
		m.EffectViewMiss = make(map[string]struct{})
	}
	key := strings.ReplaceAll(file, "/", "\\")
	if view, ok := m.EffectViews[key]; ok {
		return view
	}
	if _, ok := m.EffectViewMiss[key]; ok {
		return nil
	}
	actCandidates := effectSpriteResourceCandidates(file, "act")
	sprCandidates := effectSpriteResourceCandidates(file, "spr")
	view, status := scene.LoadSpriteView(manager, actCandidates, sprCandidates, nil, "effect sprite "+file)
	if view == nil {
		m.EffectViewMiss[key] = struct{}{}
		glog.Warnf("effect sprite unavailable file=%q: %s", file, status)
		return nil
	}
	m.EffectViews[key] = view
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

func (m *System) initDirectedSpriteEffect(ctx client.Context, effect *Instance) {
	effect.ActorOrigin = EffectWorldAnchor(ctx, *effect, effect.Starts)
	caster := *effect
	caster.ActorID = effect.TargetID
	effect.ProjectileDirection = scene.Sub3(EffectWorldAnchor(ctx, caster, effect.Starts), effect.ActorOrigin)
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
	}
	effect.HasActorTransform = true
	yaw := m.cameraYaw(ctx)
	effect.SpriteFrameOverride = directedSpriteAction(effect.ProjectileDirection, yaw)
	effect.HasSpriteFrame = true
}

func directedSpriteAction(direction scene.Point3, yaw float64) int {
	angle := math.Atan2(direction.Z, direction.X)*180/math.Pi - yaw
	quadrant := int(math.Floor((math.Mod(angle, 360)+360)/90)) % 4
	return (6 - quadrant) % 4
}

func (m *System) drawDirectedSpriteEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	component.Duration = 0 // The non-repeating ACT owns its visible lifetime.
	p := effect.ActorOrigin
	m.DrawSPREffect(screen, ctx, projection, effect, component, p.X, p.Z, p.Y, now)
}

func (m *System) drawNativeSpriteEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, x, y, z float64, now time.Time) {
	if now.Before(effect.Starts.Add(c.Delay)) {
		return
	}
	if effect.EffectID == Kaen || effect.EffectID >= LightningS && effect.EffectID <= FlareS {
		period := 16 * db.EffectFrameDuration
		cycle := now.Sub(effect.Starts) / period
		for previous := time.Duration(1); previous >= 0; previous-- {
			birth := (cycle - previous) * period
			if birth < 0 || (!effect.Persistent && birth > 0) || now.Sub(effect.Starts)-birth >= 23*db.EffectFrameDuration {
				continue
			}
			copy := effect
			copy.Starts = effect.Starts.Add(birth)
			if effect.EffectID == Kaen {
				c.SpriteDelay = time.Duration(3+HitEffectRandom(effect, int(cycle-previous), 5)) * db.EffectFrameDuration
			}
			copy.Starts = copy.Starts.Add(-c.SpriteDelay)
			m.DrawSPREffect(screen, ctx, projection, copy, c, x, y, z, now)
		}
		return
	}
	// Native sprite primitives advance their first motion on tick zero.
	effect.Starts = effect.Starts.Add(c.Delay - c.SpriteDelay)
	c.Delay = 0
	if c.SizeStart > 0 {
		m.draw3DSpriteEffect(screen, ctx, projection, effect, c, x, y, z+c.PosZ, c.SizeStart, effectComponentAlpha(0, c), 0, effect.Starts, now)
		return
	}
	m.DrawSPREffect(screen, ctx, projection, effect, c, x, y, z, now)
}

type asuraGlyphs struct {
	origins [2]scene.Point3
	count   int
}

func (m *System) DrawEffectGlyphs(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	mildWind := effect.EffectID >= BeginAsura1 && effect.EffectID <= BeginAsura7
	if !mildWind && effect.asuraGlyphs == nil {
		return
	}
	for !mildWind && effect.asuraGlyphs.count < 2 {
		i := effect.asuraGlyphs.count
		at := effect.Starts.Add(time.Duration(1+20*i) * db.EffectFrameDuration)
		if now.Before(at) {
			break
		}
		effect.asuraGlyphs.origins[i] = EffectWorldAnchor(ctx, effect, at)
		effect.asuraGlyphs.count++
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	champion := effect.EffectID == BeginAsura11
	spacing, radiusBase, shade := 12.0, 18.0, uint8(0)
	if champion {
		spacing, radiusBase, shade = 16, 24, 10
	}
	origin := effect.ActorOrigin
	if mildWind {
		origin = EffectWorldAnchor(ctx, effect, now)
		shade = 255
	}
	right, _, _, ok := projection.BillboardBasis(origin.X, origin.Z, origin.Y)
	if !ok {
		return
	}
	for i, file := range component.TextureFiles {
		// The left and right primitives start at ticks 1 and 21; their
		// internal delays make the six characters appear 20 ticks apart.
		age := frame - 20*i
		if mildWind {
			age = frame + 1
		}
		if age <= 0 {
			continue
		}
		texture := m.EffectFileTexture(ctx.Resources, file)
		position := origin
		if !mildWind {
			origin = effect.asuraGlyphs.origins[i/3]
			position = scene.Add3(origin, scene.Mul3(right, (float64(i)-2.5)*spacing/5))
		}
		position.Y += 6
		radius := (radiusBase - .1*float64(min(age, 49))) * math.Pow(1.01, float64(scene.ClampInt(age-140, 0, 49)))
		for layer := 10; layer >= 1; layer-- {
			alpha := min(13*min(age, 19), 50+layer*20)
			if layer < 10 {
				alpha -= 5 * max(0, age-19)
			} else {
				alpha -= 5 * max(0, age-140)
			}
			if alpha <= 0 {
				continue
			}
			size := math.Sqrt2 * (radius + float64(10-layer)*3) / 5
			tint := color.RGBA{shade, shade, shade, uint8(alpha)}
			if champion && layer == 10 {
				tint.A = uint8(alpha / 2)
				drawTexturedEffectBillboardRotatedXY(screen, projection, texture, position.X, position.Z, position.Y, size, size, 0, tint, false)
				tint.A = uint8(alpha)
			}
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, position.X, position.Z, position.Y, size, size, 0, tint, false)
		}
	}
}
