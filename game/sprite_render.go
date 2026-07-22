package game

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	spriteActionIdle = iota
	spriteActionWalk
	spriteActionSit
	spriteActionPickup
)

const (
	spriteActionNonPCAttack  = 2
	spriteActionNonPCHurt    = 3
	spriteActionNonPCDeath   = 4
	spriteActionNonPCPerf1   = 5
	spriteActionNonPCPerf2   = 6
	spriteActionNonPCPerf3   = 7
	spriteActionNonPCSpecial = 8
	spriteActionPCReadyFight = 4
	spriteActionPCAttack1    = 5
	spriteActionPCHurt       = 6
	spriteActionPCDeath      = 8
	spriteActionPCFreeze2    = 9
	spriteActionPCAttack2    = 10
	spriteActionPCAttack3    = 11
	spriteActionPCSkill      = 12
)

const (
	humanoidBillboardWidth   = 224
	humanoidBillboardHeight  = 160
	humanoidBillboardAnchorX = 80
	humanoidBillboardAnchorY = 120
	// reference client lifts entity sprites by 0.2 map units before depth testing.
	// Keeping actor billboards exactly coplanar with terrain lets sloped GND
	// triangles clip the bottom pixels of tight NPC/mob sprite frames.
	actorSpriteTerrainLift = 0.2
	actorShadowTerrainLift = 0.05
	actorShadowDepthBias   = 0.001
)

const (
	humanoidCompositionWidth   = 512
	humanoidCompositionHeight  = 384
	humanoidCompositionAnchorX = 256
	humanoidCompositionAnchorY = 320
	humanoidCompositionPadding = 4
)

const (
	humanoidLayerBody = iota
	humanoidLayerHead
	humanoidLayerAccessoryBottom
	humanoidLayerAccessoryMid
	humanoidLayerAccessoryTop
	humanoidLayerWeapon
	humanoidLayerWeaponLight
	humanoidLayerShield
	humanoidLayerCount
)

type spriteView struct {
	spr           *res.SPR
	act           *res.ACT
	actSource     string
	source        string
	palette       *res.Palette
	paletteSource string
	images        map[spriteFrameKey]*render.Image
	billboards    map[singleSpriteBillboardKey]*spriteBillboard
	started       time.Time
}

type spriteFrameKey struct {
	index   int32
	sprType int32
}

type petAccessorySpriteKey struct {
	job         int
	accessoryID uint32
}

type humanoidSpriteView struct {
	body            *spriteView
	head            *spriteView
	accessoryBottom *spriteView
	accessoryMid    *spriteView
	accessoryTop    *spriteView
	weapon          *spriteView
	weaponLight     *spriteView
	shield          *spriteView
	imf             *res.IMF
	imfSource       string
	billboards      map[humanoidBillboardKey]*spriteBillboard
	started         time.Time
}

type humanoidAppearance struct {
	job         int
	head        int
	sex         byte
	bodyPalette int
	headPalette int
	weapon      int
	shield      int
	headTop     int
	headMid     int
	headLow     int
}

type humanoidBillboardKey struct {
	actionFamily int
	direction    int
	bodyMotion   int
	headMotion   int
}

type singleSpriteBillboardKey struct {
	actionIndex       int
	motion            int
	anchorX           float64
	anchorY           float64
	ignoreLayerAngles bool
}

type spriteBillboard struct {
	image   *render.Image
	anchorX float64
	anchorY float64
}

type spriteState struct {
	actionFamily   int
	direction      int
	headDir        uint8
	headTurn       bool
	cameraYaw      float64
	moving         bool
	started        time.Time
	loop           bool
	loopIdle       bool
	play           bool
	hasPlay        bool
	length         int
	hasLength      bool
	frameOffset    int
	hasFrameOffset bool
	fixedMotion    int
	hasFixedMotion bool
	speed          time.Duration
	hasSpeed       bool
	moveSpeedMS    int
	walkDistance   float64
}

func (m *WorldMode) drawPlayerSprite3D(ctx client.Context, screen *render.Frame, projection sceneProjection, entry sceneActorDrawEntry, direction int, cameraYaw float64, shadow float64, alpha float64) bool {
	now := time.Now()
	actor := entry.actor
	moving := actorIsMovingAt(actor, now)
	state := spriteState{
		actionFamily: spriteActionIdle,
		direction:    direction,
		headDir:      actor.HeadDir,
		headTurn:     true,
		cameraYaw:    cameraYaw,
		moving:       moving,
		moveSpeedMS:  actor.Speed,
	}
	if moving {
		state.actionFamily = spriteActionWalk
		state.loop = true
		state.walkDistance = actorRenderWalkDistance(actor, now)
	} else if actor.Sitting {
		state.actionFamily = spriteActionSit
	}
	if ctx.Session != nil {
		if anim, ok := m.actorAnimation(ctx.Session.CharID, now); ok {
			applyActorAnimationToSpriteState(&state, anim, true)
		} else if anim, ok := m.actorAnimation(ctx.Session.AccountID, now); ok {
			applyActorAnimationToSpriteState(&state, anim, true)
		}
	}
	if !isDeathActionFamily(state.actionFamily) {
		applyActorBodyState(actor, &state)
	}
	billboard, ok := humanoidBillboardForState(m.playerView, state, now)
	if !ok {
		return false
	}
	drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, m.playerRenderScale(ctx, actor, entry.scale, now), alpha, shadow, m.playerRenderTint(ctx, actor, now))
	return true
}

func applyActorAnimationToSpriteState(state *spriteState, anim actorAnimation, playerLike bool) bool {
	if state == nil {
		return false
	}
	if state.moving && !actorAnimationOverridesWalk(anim, playerLike) {
		return false
	}
	state.actionFamily = anim.actionFamily
	state.started = anim.started
	state.loop = anim.loop
	state.play = anim.play
	state.hasPlay = anim.hasPlay
	state.length = anim.length
	state.hasLength = anim.hasLength
	state.frameOffset = anim.frameOffset
	state.hasFrameOffset = anim.hasFrameOffset
	state.moving = false
	state.fixedMotion = anim.fixedMotion
	state.hasFixedMotion = anim.hasFixedMotion
	state.speed = anim.speed
	state.hasSpeed = anim.hasSpeed
	return true
}

func actorAnimationOverridesWalk(anim actorAnimation, playerLike bool) bool {
	if playerLike {
		return anim.actionFamily == spriteActionPCDeath
	}
	return anim.actionFamily == spriteActionNonPCDeath
}

func drawActorSpriteBillboardTintAlpha3D(screen *render.Frame, projection sceneProjection, billboard *spriteBillboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64, tintColor color.RGBA) {
	drawSpriteBillboardTintAlpha3D(screen, projection, billboard, worldX, worldY, actorSpriteWorldZ(worldZ), scale, alpha, shadow, tintColor)
}

func actorSpriteWorldZ(terrainZ float64) float64 {
	return terrainZ + actorSpriteTerrainLift
}

func drawFixedSpriteShadowBillboard3D(screen *render.Frame, projection sceneProjection, view *spriteView, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64) bool {
	billboard, ok := fixedSpriteBillboard(view)
	if !ok {
		return false
	}
	options := spriteBillboardTriangleDrawOptions()
	options.DepthBias = actorShadowDepthBias
	drawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, shadow, color.RGBA{R: 255, G: 255, B: 255, A: 255}, options)
	return true
}

func drawSpriteBillboardAlpha3D(screen *render.Frame, projection sceneProjection, billboard *spriteBillboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64) {
	drawSpriteBillboardTintAlpha3D(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, shadow, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

func drawSpriteBillboardTintAlpha3D(screen *render.Frame, projection sceneProjection, billboard *spriteBillboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64, tintColor color.RGBA) {
	drawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, shadow, tintColor, spriteBillboardTriangleDrawOptions())
}

func drawSpriteBillboardTintAlphaOverlay3D(screen *render.Frame, projection sceneProjection, billboard *spriteBillboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64, tintColor color.RGBA) {
	drawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, shadow, tintColor, &render.DrawTrianglesOptions{
		Filter:  spriteDrawFilter(),
		Address: render.AddressClampToZero,
	})
}

func drawSpriteBillboardTintAlphaRotated3DWithOptions(screen *render.Frame, projection sceneProjection, billboard *spriteBillboard, worldX, worldY, worldZ, scale float64, angle float64, alpha float64, shadow float64, tintColor color.RGBA, options *render.DrawTrianglesOptions) {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	if alpha < 0 || math.IsNaN(alpha) {
		alpha = 0
	}
	if alpha > 1 || math.IsInf(alpha, 0) {
		alpha = 1
	}
	if shadow < 0 || math.IsNaN(shadow) {
		shadow = 0
	}
	if shadow > 1 || math.IsInf(shadow, 0) {
		shadow = 1
	}
	tintR := float64(tintColor.R) / 255 * shadow
	tintG := float64(tintColor.G) / 255 * shadow
	tintB := float64(tintColor.B) / 255 * shadow
	tintA := float64(tintColor.A) / 255 * alpha
	right, up, unitsPerPixel, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	bounds := billboard.image.Bounds()
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	center := modelPoint3{x: worldX, y: worldZ, z: worldY}
	tint := colorRGBAFromFloats(tintR, tintG, tintB, tintA)
	axisScale := scale * unitsPerPixel
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	rightAxis := add3(mul3(right, cosA*axisScale), mul3(up, -sinA*axisScale))
	upAxis := add3(mul3(right, -sinA*axisScale), mul3(up, -cosA*axisScale))
	screen.DrawWorldBillboard(render.WorldBillboardCommand{
		Texture:     billboard.image,
		Options:     *options,
		Center:      [3]float32{float32(center.x), float32(center.y), float32(center.z)},
		RightAxis:   [3]float32{float32(rightAxis.x), float32(rightAxis.y), float32(rightAxis.z)},
		UpAxis:      [3]float32{float32(upAxis.x), float32(upAxis.y), float32(upAxis.z)},
		DepthUpAxis: [3]float32{float32(upAxis.x), float32(upAxis.y), float32(upAxis.z)},
		Width:       float32(w),
		Height:      float32(h),
		AnchorX:     float32(billboard.anchorX),
		AnchorY:     float32(billboard.anchorY),
		ColorR:      float32(tint.R) / 255,
		ColorG:      float32(tint.G) / 255,
		ColorB:      float32(tint.B) / 255,
		ColorA:      float32(tint.A) / 255,
		DepthBias:   options.DepthBias,
	})
}

func drawSpriteBillboardTintAlpha3DWithOptions(screen *render.Frame, projection sceneProjection, billboard *spriteBillboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64, tintColor color.RGBA, options *render.DrawTrianglesOptions) {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	if alpha < 0 || math.IsNaN(alpha) {
		alpha = 0
	}
	if alpha > 1 || math.IsInf(alpha, 0) {
		alpha = 1
	}
	if shadow < 0 || math.IsNaN(shadow) {
		shadow = 0
	}
	if shadow > 1 || math.IsInf(shadow, 0) {
		shadow = 1
	}
	tintR := float64(tintColor.R) / 255 * shadow
	tintG := float64(tintColor.G) / 255 * shadow
	tintB := float64(tintColor.B) / 255 * shadow
	tintA := float64(tintColor.A) / 255 * alpha
	right, up, unitsPerPixel, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	bounds := billboard.image.Bounds()
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	center := modelPoint3{x: worldX, y: worldZ, z: worldY}
	tint := colorRGBAFromFloats(tintR, tintG, tintB, tintA)
	axisScale := scale * unitsPerPixel
	screen.DrawWorldBillboard(render.WorldBillboardCommand{
		Texture:     billboard.image,
		Options:     *options,
		Center:      [3]float32{float32(center.x), float32(center.y), float32(center.z)},
		RightAxis:   [3]float32{float32(right.x * axisScale), float32(right.y * axisScale), float32(right.z * axisScale)},
		UpAxis:      [3]float32{float32(-up.x * axisScale), float32(-up.y * axisScale), float32(-up.z * axisScale)},
		DepthUpAxis: [3]float32{0, float32(-axisScale), 0},
		Width:       float32(w),
		Height:      float32(h),
		AnchorX:     float32(billboard.anchorX),
		AnchorY:     float32(billboard.anchorY),
		ColorR:      float32(tint.R) / 255,
		ColorG:      float32(tint.G) / 255,
		ColorB:      float32(tint.B) / 255,
		ColorA:      float32(tint.A) / 255,
		DepthBias:   options.DepthBias,
	})
}

func drawSpriteBillboardTintAlphaWorld3D(screen *render.Frame, projection sceneProjection, billboard *spriteBillboard, worldX, worldY, worldZ, pixelScale, angle float64, alpha float64, shadow float64, tintColor color.RGBA) {
	drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, pixelScale, angle, alpha, shadow, tintColor, spriteBillboardTriangleDrawOptions())
}

func drawSpriteBillboardTintAlphaWorld3DWithOptions(screen *render.Frame, projection sceneProjection, billboard *spriteBillboard, worldX, worldY, worldZ, pixelScale, angle float64, alpha float64, shadow float64, tintColor color.RGBA, options *render.DrawTrianglesOptions) {
	if screen == nil || billboard == nil || billboard.image == nil {
		return
	}
	if options == nil {
		options = spriteBillboardTriangleDrawOptions()
	}
	if pixelScale <= 0 || math.IsNaN(pixelScale) || math.IsInf(pixelScale, 0) {
		pixelScale = effectPixelRatio
	}
	if alpha < 0 || math.IsNaN(alpha) {
		alpha = 0
	}
	if alpha > 1 || math.IsInf(alpha, 0) {
		alpha = 1
	}
	if shadow < 0 || math.IsNaN(shadow) {
		shadow = 0
	}
	if shadow > 1 || math.IsInf(shadow, 0) {
		shadow = 1
	}
	right, up, _, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	rightAxis := mul3(right, pixelScale)
	upAxis := mul3(up, -pixelScale)
	if angle != 0 {
		sinA, cosA := math.Sin(angle), math.Cos(angle)
		rightAxis = add3(mul3(right, cosA*pixelScale), mul3(up, sinA*pixelScale))
		upAxis = add3(mul3(right, sinA*pixelScale), mul3(up, -cosA*pixelScale))
	}
	tint := colorRGBAFromFloats(
		float64(tintColor.R)/255*shadow,
		float64(tintColor.G)/255*shadow,
		float64(tintColor.B)/255*shadow,
		float64(tintColor.A)/255*alpha,
	)
	bounds := billboard.image.Bounds()
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	center := modelPoint3{x: worldX, y: worldZ, z: worldY}
	cmd := worldSpriteBillboardCommand(billboard.image, *options, center, rightAxis, upAxis, float32(w), float32(h), float32(billboard.anchorX), float32(billboard.anchorY), tint)
	screen.DrawWorldBillboard(cmd)
}

func worldSpriteBillboardCommand(texture *render.Image, options render.DrawTrianglesOptions, center, rightAxis, upAxis modelPoint3, width, height, anchorX, anchorY float32, tint color.RGBA) render.WorldBillboardCommand {
	// robr's SpriteRenderer depth correction projects vertical billboards onto a
	// center-anchored plane, so the lower edge of map effect sprites does not
	// sink behind the floor.
	return render.WorldBillboardCommand{
		Texture:     texture,
		Options:     options,
		Center:      [3]float32{float32(center.x), float32(center.y), float32(center.z)},
		RightAxis:   [3]float32{float32(rightAxis.x), float32(rightAxis.y), float32(rightAxis.z)},
		UpAxis:      [3]float32{float32(upAxis.x), float32(upAxis.y), float32(upAxis.z)},
		DepthUpAxis: [3]float32{},
		Width:       width,
		Height:      height,
		AnchorX:     anchorX,
		AnchorY:     anchorY,
		ColorR:      float32(tint.R) / 255,
		ColorG:      float32(tint.G) / 255,
		ColorB:      float32(tint.B) / 255,
		ColorA:      float32(tint.A) / 255,
		DepthBias:   options.DepthBias,
	}
}

func spriteBillboardTriangleDrawOptions() *render.DrawTrianglesOptions {
	options := triangleDrawOptions(spriteDrawFilter(), render.AddressClampToZero)
	options.Blend = render.BlendSourceOver
	return options
}

func colorRGBAFromFloats(r, g, b, a float64) color.RGBA {
	return color.RGBA{
		R: clampColor(r * 255),
		G: clampColor(g * 255),
		B: clampColor(b * 255),
		A: clampColor(a * 255),
	}
}

func spriteDrawFilter() render.Filter {
	return render.FilterLinear
}

func spriteCompositionFilter() render.Filter {
	return render.FilterNearest
}

func humanoidBillboardForState(view *humanoidSpriteView, state spriteState, now time.Time) (*spriteBillboard, bool) {
	if view == nil || view.body == nil || view.body.act == nil || view.body.spr == nil {
		return nil, false
	}
	state.direction = spriteDirectionFromWorldDirForCamera(state.direction, state.cameraYaw)
	bodyActionIndex, bodyAction, ok := resolveSpriteAction(view.body.act, state.actionFamily, state.direction)
	if !ok || len(bodyAction.Animations) == 0 {
		return nil, false
	}
	bodyMotion := bodyMotionForState(bodyAction, state, view.started, now)
	headMotion := 0
	if view.head != nil {
		if _, headAction, headOK := resolveSpriteAction(view.head.act, state.actionFamily, state.direction); headOK && len(headAction.Animations) > 0 {
			headMotion = selectHeadMotion(state, bodyMotion, headAction)
		}
	}
	key := humanoidBillboardKey{
		actionFamily: bodyActionIndex / 8,
		direction:    bodyActionIndex % 8,
		bodyMotion:   bodyMotion,
		headMotion:   headMotion,
	}
	if billboard, ok := view.billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := composeHumanoidBillboard(view, key.actionFamily, key.direction, bodyAction, bodyMotion, headMotion)
	if !ok {
		return nil, false
	}
	view.billboards[key] = billboard
	return billboard, true
}

func singleSpriteBillboardForState(view *spriteView, state spriteState, now time.Time) (*spriteBillboard, bool) {
	if view == nil || view.act == nil || view.spr == nil {
		return nil, false
	}
	state.direction = spriteDirectionFromWorldDirForCamera(state.direction, state.cameraYaw)
	actionIndex, action, ok := resolveSpriteAction(view.act, state.actionFamily, state.direction)
	if !ok || len(action.Animations) == 0 {
		return nil, false
	}
	motion := bodyMotionForState(action, state, view.started, now)
	key := singleSpriteBillboardKey{actionIndex: actionIndex, motion: motion}
	if billboard, ok := view.billboards[key]; ok {
		return billboard, true
	}
	if motion < 0 || motion >= len(action.Animations) {
		return nil, false
	}
	billboard, ok := composeSingleSpriteBillboard(view, action.Animations[motion])
	if !ok {
		return nil, false
	}
	view.billboards[key] = billboard
	return billboard, true
}

func fixedSpriteBillboard(view *spriteView) (*spriteBillboard, bool) {
	if view == nil || view.act == nil || view.spr == nil || len(view.act.Actions) == 0 || len(view.act.Actions[0].Animations) == 0 {
		return nil, false
	}
	key := singleSpriteBillboardKey{actionIndex: 0, motion: 0}
	if billboard, ok := view.billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := composeSingleSpriteBillboard(view, view.act.Actions[0].Animations[0])
	if !ok {
		return nil, false
	}
	view.billboards[key] = billboard
	return billboard, true
}

func cursorFrameBillboard(view *spriteView, actionIndex, motion int, anchorX, anchorY float64) (*spriteBillboard, bool) {
	if view == nil || view.act == nil || view.spr == nil || len(view.act.Actions) == 0 {
		return nil, false
	}
	if actionIndex < 0 || actionIndex >= len(view.act.Actions) || len(view.act.Actions[actionIndex].Animations) == 0 {
		actionIndex = 0
	}
	action := view.act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return nil, false
	}
	if motion < 0 {
		motion = 0
	}
	motion %= len(action.Animations)
	key := singleSpriteBillboardKey{actionIndex: actionIndex, motion: motion, anchorX: anchorX, anchorY: anchorY}
	if billboard, ok := view.billboards[key]; ok {
		return billboard, true
	}
	anim := action.Animations[motion]
	minX, minY, maxX, maxY, ok := spriteAnimationLayerBounds(view, anim)
	if !ok {
		return nil, false
	}
	const padding = 4.0
	minX = minX + anchorX - padding
	minY = minY + anchorY - padding
	maxX = maxX + anchorX + padding
	maxY = maxY + anchorY + padding
	width := int(math.Ceil(maxX - minX))
	height := int(math.Ceil(maxY - minY))
	if width <= 0 || height <= 0 {
		return nil, false
	}
	target := render.NewImage(width, height)
	compositionAnchorX := anchorX - minX
	compositionAnchorY := anchorY - minY
	if !drawSpriteAnimation(target, view, anim, compositionAnchorX, compositionAnchorY, 0, 0) {
		return nil, false
	}
	billboard := &spriteBillboard{
		image:   target,
		anchorX: compositionAnchorX,
		anchorY: compositionAnchorY,
	}
	view.billboards[key] = billboard
	return billboard, true
}

func composeHumanoidBillboard(view *humanoidSpriteView, actionFamily, direction int, bodyAction res.ACTAction, bodyMotion, headMotion int) (*spriteBillboard, bool) {
	if bodyMotion < 0 || bodyMotion >= len(bodyAction.Animations) {
		return nil, false
	}
	target := render.NewImage(humanoidCompositionWidth, humanoidCompositionHeight)
	bodyActionIndex := actionFamily*8 + direction
	drawn := false
	if view.imf != nil {
		drawn = drawPlayerIMFLayers(target, view, bodyActionIndex, bodyMotion, headMotion)
	} else {
		drawn = drawFallbackHumanoidLayers(target, view, actionFamily, direction, bodyAction, bodyMotion, headMotion)
	}
	if !drawn {
		return nil, false
	}
	return cropHumanoidBillboard(target)
}

func cropHumanoidBillboard(source *render.Image) (*spriteBillboard, bool) {
	minX, minY, maxX, maxY, ok := renderImageOpaqueBounds(source)
	if !ok {
		return nil, false
	}
	bounds := source.Bounds()
	minX = max(0, minX-humanoidCompositionPadding)
	minY = max(0, minY-humanoidCompositionPadding)
	maxX = min(bounds.Dx()-1, maxX+humanoidCompositionPadding)
	maxY = min(bounds.Dy()-1, maxY+humanoidCompositionPadding)
	cropRect := image.Rect(minX, minY, maxX+1, maxY+1)
	cropped := render.NewImageFromImage(source.RGBA().SubImage(cropRect))
	return &spriteBillboard{
		image:   cropped,
		anchorX: humanoidCompositionAnchorX - float64(minX),
		anchorY: humanoidCompositionAnchorY - float64(minY),
	}, true
}

func renderImageOpaqueBounds(source *render.Image) (int, int, int, int, bool) {
	rgba := source.RGBA()
	if rgba == nil {
		return 0, 0, 0, 0, false
	}
	bounds := rgba.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	ok := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if rgba.RGBAAt(x, y).A == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
			ok = true
		}
	}
	if !ok {
		return 0, 0, 0, 0, false
	}
	return minX - bounds.Min.X, minY - bounds.Min.Y, maxX - bounds.Min.X, maxY - bounds.Min.Y, true
}

func composeSingleSpriteBillboard(view *spriteView, anim res.ACTAnimation) (*spriteBillboard, bool) {
	return composeSingleSpriteBillboardWithOptions(view, anim, false)
}

func composeSingleSpriteBillboardWithOptions(view *spriteView, anim res.ACTAnimation, ignoreLayerAngles bool) (*spriteBillboard, bool) {
	minX, minY, maxX, maxY, ok := spriteAnimationLayerBounds(view, anim)
	if !ok {
		return nil, false
	}
	const padding = 4.0
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding
	width := int(math.Ceil(maxX - minX))
	height := int(math.Ceil(maxY - minY))
	if width <= 0 || height <= 0 {
		return nil, false
	}
	target := render.NewImage(width, height)
	anchorX := -minX
	anchorY := -minY
	if !drawSpriteAnimationWithOptions(target, view, anim, anchorX, anchorY, 0, 0, ignoreLayerAngles) {
		return nil, false
	}
	return &spriteBillboard{
		image:   target,
		anchorX: anchorX,
		anchorY: anchorY,
	}, true
}

func spriteAnimationLayerBounds(view *spriteView, anim res.ACTAnimation) (float64, float64, float64, float64, bool) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	ok := false
	for _, layer := range anim.Layers {
		if layer.Index < 0 {
			continue
		}
		width, height, frameOK := spriteLayerFrameSize(view, layer.Index, layer.SPRType)
		if !frameOK {
			continue
		}
		scaleX := math.Abs(float64(layer.ScaleX))
		scaleY := math.Abs(float64(layer.ScaleY))
		if scaleX == 0 {
			scaleX = 1
		}
		if scaleY == 0 {
			scaleY = 1
		}
		width *= scaleX
		height *= scaleY
		centerX, centerY := spriteLayerCenter(0, 0, layer)
		minX = math.Min(minX, centerX-width*0.5)
		maxX = math.Max(maxX, centerX+width*0.5)
		minY = math.Min(minY, centerY-height*0.5)
		maxY = math.Max(maxY, centerY+height*0.5)
		ok = true
	}
	return minX, minY, maxX, maxY, ok
}

func spriteLayerFrameSize(view *spriteView, index int32, sprType int32) (float64, float64, bool) {
	if view == nil || view.spr == nil {
		return 0, 0, false
	}
	frameIndex := int(index)
	if sprType == res.SPRFrameRGBA {
		frameIndex += view.spr.RGBAIndex
	}
	if frameIndex < 0 || frameIndex >= len(view.spr.Frames) {
		return 0, 0, false
	}
	frame := view.spr.Frames[frameIndex]
	if frame.Width <= 0 || frame.Height <= 0 {
		return 0, 0, false
	}
	return float64(frame.Width), float64(frame.Height), true
}

func drawFallbackHumanoidLayers(target *render.Image, view *humanoidSpriteView, actionFamily, direction int, bodyAction res.ACTAction, bodyMotion, headMotion int) bool {
	bodyAnim := bodyAction.Animations[bodyMotion]
	actionIndex := actionFamily*8 + direction
	drawn := false
	for _, layer := range playerRenderLayerOrder(nil, actionIndex, bodyMotion) {
		switch layer {
		case humanoidLayerBody:
			drawn = drawSpriteAnimation(target, view.body, bodyAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, 0, 0) || drawn
		case humanoidLayerHead:
			drawn = drawFallbackHeadMotion(target, view, bodyAnim, actionIndex, headMotion) || drawn
		case humanoidLayerAccessoryBottom:
			drawn = drawAttachedAccessoryMotion(target, view.accessoryBottom, view.head, bodyAnim, actionIndex, headMotion) || drawn
		case humanoidLayerAccessoryMid:
			drawn = drawAttachedAccessoryMotion(target, view.accessoryMid, view.head, bodyAnim, actionIndex, headMotion) || drawn
		case humanoidLayerAccessoryTop:
			drawn = drawAttachedAccessoryMotion(target, view.accessoryTop, view.head, bodyAnim, actionIndex, headMotion) || drawn
		case humanoidLayerWeapon:
			drawn = drawFallbackOverlayMotion(target, view.weapon, view.body, actionIndex, bodyMotion) || drawn
		case humanoidLayerWeaponLight:
			drawn = drawFallbackOverlayMotion(target, view.weaponLight, view.body, actionIndex, bodyMotion) || drawn
		case humanoidLayerShield:
			drawn = drawFallbackOverlayMotion(target, view.shield, view.body, actionIndex, bodyMotion) || drawn
		}
	}
	return drawn
}

func drawFallbackHeadMotion(target *render.Image, view *humanoidSpriteView, bodyAnim res.ACTAnimation, actionIndex, headMotion int) bool {
	if view.head == nil || view.head.act == nil || view.head.spr == nil {
		return false
	}
	headAnim, ok := actionAnimation(view.head.act, actionIndex, headMotion)
	if !ok {
		headAnim, ok = actionAnimation(view.head.act, actionIndex, 0)
	}
	if !ok {
		return false
	}
	headPosX, headPosY := attachmentDelta(bodyAnim, headAnim)
	return drawSpriteAnimation(target, view.head, headAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, headPosX, headPosY)
}

func drawPlayerIMFLayers(target *render.Image, view *humanoidSpriteView, actionIndex, bodyMotion, headMotion int) bool {
	order := playerRenderLayerOrder(view.imf, actionIndex, bodyMotion)
	bodyAnim, bodyAnimOK := actionAnimation(view.body.act, actionIndex, bodyMotion)
	drawn := false
	for _, layer := range order {
		switch layer {
		case humanoidLayerBody:
			drawn = drawPlayerIMFLayer(target, view.body, view.imf, humanoidLayerBody, actionIndex, bodyMotion, nil) || drawn
		case humanoidLayerHead:
			if view.head == nil || view.head.act == nil || view.head.spr == nil {
				continue
			}
			var attachBase *res.ACTAnimation
			if bodyAnimOK {
				attachBase = &bodyAnim
			}
			drawn = drawPlayerIMFLayer(target, view.head, view.imf, humanoidLayerHead, actionIndex, headMotion, attachBase) || drawn
		case humanoidLayerAccessoryBottom:
			if bodyAnimOK {
				drawn = drawAttachedAccessoryMotion(target, view.accessoryBottom, view.head, bodyAnim, actionIndex, headMotion) || drawn
			}
		case humanoidLayerAccessoryMid:
			if bodyAnimOK {
				drawn = drawAttachedAccessoryMotion(target, view.accessoryMid, view.head, bodyAnim, actionIndex, headMotion) || drawn
			}
		case humanoidLayerAccessoryTop:
			if bodyAnimOK {
				drawn = drawAttachedAccessoryMotion(target, view.accessoryTop, view.head, bodyAnim, actionIndex, headMotion) || drawn
			}
		case humanoidLayerWeapon:
			drawn = drawPlayerOverlayMotion(target, view.weapon, view.body, view.imf, humanoidLayerWeapon, actionIndex, bodyMotion) || drawn
		case humanoidLayerWeaponLight:
			drawn = drawPlayerOverlayMotion(target, view.weaponLight, view.body, view.imf, humanoidLayerWeaponLight, actionIndex, bodyMotion) || drawn
		case humanoidLayerShield:
			drawn = drawPlayerOverlayMotion(target, view.shield, view.body, view.imf, humanoidLayerShield, actionIndex, bodyMotion) || drawn
		}
	}
	return drawn
}

func drawPlayerIMFLayer(target *render.Image, sprite *spriteView, imf *res.IMF, layerPriority, actionIndex, motionIndex int, attachBase *res.ACTAnimation) bool {
	if sprite == nil || sprite.act == nil || sprite.spr == nil {
		return false
	}
	if actionIndex < 0 || actionIndex >= len(sprite.act.Actions) {
		return false
	}
	action := sprite.act.Actions[actionIndex]
	if motionIndex < 0 || motionIndex >= len(action.Animations) {
		return false
	}
	anim := action.Animations[motionIndex]
	resolvedLayer := layerPriority
	if imf != nil {
		if layer := imf.LayerForPriority(layerPriority, actionIndex, motionIndex); layer >= 0 {
			resolvedLayer = layer
		}
	}
	drawLayerIndex := visibleACTLayerIndex(anim, resolvedLayer, layerPriority)
	if drawLayerIndex < 0 {
		return false
	}
	pointX, pointY := int32(0), int32(0)
	if imf != nil {
		pointX, pointY = imf.Point(drawLayerIndex, actionIndex, motionIndex)
	}
	layer := anim.Layers[drawLayerIndex]
	if attachBase != nil {
		dx, dy := attachmentDelta(*attachBase, anim)
		pointX += dx
		pointY += dy
	}
	return drawSpriteLayerByValue(target, sprite, layer, humanoidCompositionAnchorX+float64(pointX), humanoidCompositionAnchorY+float64(pointY))
}

func visibleACTLayerIndex(anim res.ACTAnimation, candidates ...int) int {
	for _, index := range candidates {
		if index >= 0 && index < len(anim.Layers) && anim.Layers[index].Index >= 0 {
			return index
		}
	}
	for index, layer := range anim.Layers {
		if layer.Index >= 0 {
			return index
		}
	}
	return -1
}

func actionAnimation(act *res.ACT, actionIndex, motionIndex int) (res.ACTAnimation, bool) {
	if act == nil || actionIndex < 0 || actionIndex >= len(act.Actions) {
		return res.ACTAnimation{}, false
	}
	action := act.Actions[actionIndex]
	if motionIndex < 0 || motionIndex >= len(action.Animations) {
		return res.ACTAnimation{}, false
	}
	return action.Animations[motionIndex], true
}

func drawAttachedAccessoryMotion(target *render.Image, accessory *spriteView, head *spriteView, bodyAnim res.ACTAnimation, actionIndex, headMotionIndex int) bool {
	if accessory == nil || accessory.act == nil || accessory.spr == nil || head == nil || head.act == nil || head.spr == nil {
		return false
	}
	headAnim, ok := actionAnimation(head.act, actionIndex, headMotionIndex)
	if !ok {
		return false
	}
	accessoryMotion := headMotionIndex
	accessoryAnim, ok := actionAnimation(accessory.act, actionIndex, accessoryMotion)
	if !ok {
		accessoryAnim, ok = actionAnimation(accessory.act, actionIndex, 0)
	}
	if !ok {
		return false
	}
	headDX, headDY := attachmentDelta(bodyAnim, headAnim)
	accessoryDX, accessoryDY := attachmentDelta(headAnim, accessoryAnim)
	return drawSpriteAnimation(target, accessory, accessoryAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, headDX+accessoryDX, headDY+accessoryDY)
}

func drawPlayerOverlayMotion(target *render.Image, overlay *spriteView, body *spriteView, imf *res.IMF, layerPriority, actionIndex, bodyMotion int) bool {
	if overlay == nil || overlay.act == nil || overlay.spr == nil || body == nil || body.act == nil || imf == nil {
		return false
	}
	motionIndex := resolveOverlayMotionIndex(overlay.act, body.act, actionIndex, bodyMotion)
	overlayAnim, ok := actionAnimation(overlay.act, actionIndex, motionIndex)
	if !ok {
		overlayAnim, ok = actionAnimation(overlay.act, actionIndex, 0)
	}
	if !ok {
		return false
	}
	pointX, pointY := imf.Point(layerPriority, actionIndex, bodyMotion)
	return drawSpriteAnimation(target, overlay, overlayAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, pointX, pointY)
}

func drawFallbackOverlayMotion(target *render.Image, overlay *spriteView, body *spriteView, actionIndex, bodyMotion int) bool {
	if overlay == nil || overlay.act == nil || overlay.spr == nil || body == nil || body.act == nil {
		return false
	}
	if len(overlay.act.Actions) == 0 {
		return false
	}
	resolvedActionIndex := actionIndex % len(overlay.act.Actions)
	if resolvedActionIndex < 0 {
		resolvedActionIndex += len(overlay.act.Actions)
	}
	motionIndex := resolveOverlayMotionIndex(overlay.act, body.act, resolvedActionIndex, bodyMotion)
	overlayAnim, ok := actionAnimation(overlay.act, resolvedActionIndex, motionIndex)
	if !ok {
		overlayAnim, ok = actionAnimation(overlay.act, resolvedActionIndex, 0)
	}
	if !ok {
		return false
	}
	return drawSpriteAnimation(target, overlay, overlayAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, 0, 0)
}

func resolveOverlayMotionIndex(overlayAct *res.ACT, bodyAct *res.ACT, actionIndex, bodyMotion int) int {
	if overlayAct == nil || actionIndex < 0 || actionIndex >= len(overlayAct.Actions) {
		return bodyMotion
	}
	overlayMotionCount := len(overlayAct.Actions[actionIndex].Animations)
	if overlayMotionCount <= 0 {
		return 0
	}
	motionIndex := bodyMotion
	if bodyAct != nil && actionIndex >= 0 && actionIndex < len(bodyAct.Actions) {
		bodyMotionCount := len(bodyAct.Actions[actionIndex].Animations)
		if bodyMotionCount > 0 && overlayMotionCount > bodyMotionCount && overlayMotionCount%bodyMotionCount == 0 {
			motionIndex = bodyMotion * (overlayMotionCount / bodyMotionCount)
		}
	}
	if motionIndex < 0 {
		return 0
	}
	if motionIndex >= overlayMotionCount {
		return overlayMotionCount - 1
	}
	return motionIndex
}

func playerRenderLayerOrder(imf *res.IMF, actionIndex, motionIndex int) [humanoidLayerCount]int {
	if imf == nil {
		return [humanoidLayerCount]int{
			humanoidLayerShield,
			humanoidLayerBody,
			humanoidLayerHead,
			humanoidLayerAccessoryTop,
			humanoidLayerAccessoryMid,
			humanoidLayerAccessoryBottom,
			humanoidLayerWeapon,
			humanoidLayerWeaponLight,
		}
	}
	resolveLayerPriority := func(priority int) int {
		layer := imf.LayerForPriority(priority, actionIndex, motionIndex)
		if layer < 0 {
			layer = priority
		}
		return layer
	}

	dir := actionIndex & 7
	headLayerPassed := false
	bodyAndAccessoryExchanged := 0
	var order [humanoidLayerCount]int
	outIndex := 0
	for pass := humanoidLayerCount - 1; pass >= 0; pass-- {
		layer := humanoidLayerBody
		if dir >= 2 && dir <= 5 {
			if pass == humanoidLayerShield {
				layer = humanoidLayerShield
			} else if pass >= 5 && pass <= 6 {
				layer = resolveLayerPriority(pass - 5)
			} else {
				layer = 6 - pass
			}
		} else if pass >= 6 && pass <= 7 {
			layer = resolveLayerPriority(pass - 6)
		} else {
			layer = 7 - pass
		}

		originalLayer := layer
		if (headLayerPassed || layer == humanoidLayerHead) && layer == humanoidLayerBody {
			headLayerPassed = true
			layer = humanoidLayerAccessoryBottom
			bodyAndAccessoryExchanged++
		}
		if !headLayerPassed && layer == humanoidLayerHead {
			headLayerPassed = true
		}
		if bodyAndAccessoryExchanged == 1 && originalLayer == humanoidLayerAccessoryBottom {
			bodyAndAccessoryExchanged = 2
			layer = humanoidLayerBody
		}
		if layer >= humanoidLayerCount {
			layer = humanoidLayerBody
		}
		if layer == humanoidLayerAccessoryBottom {
			layer = humanoidLayerAccessoryTop
		} else if layer == humanoidLayerAccessoryTop {
			layer = humanoidLayerAccessoryBottom
		}
		order[outIndex] = layer
		outIndex++
	}

	bodyIndex, headIndex := -1, -1
	for index, layer := range order {
		if layer == humanoidLayerBody && bodyIndex < 0 {
			bodyIndex = index
		} else if layer == humanoidLayerHead && headIndex < 0 {
			headIndex = index
		}
	}
	if bodyIndex >= 0 && headIndex >= 0 && headIndex < bodyIndex {
		order[bodyIndex], order[headIndex] = order[headIndex], order[bodyIndex]
	}

	var reordered [humanoidLayerCount]int
	var delayed [humanoidLayerCount]int
	delayedCount := 0
	reorderedCount := 0
	headDrawn := false
	for _, layer := range order {
		if !headDrawn && isHeadAccessoryLayer(layer) {
			delayed[delayedCount] = layer
			delayedCount++
			continue
		}
		reordered[reorderedCount] = layer
		reorderedCount++
		if layer == humanoidLayerHead {
			headDrawn = true
			for index := 0; index < delayedCount; index++ {
				reordered[reorderedCount] = delayed[index]
				reorderedCount++
			}
			delayedCount = 0
		}
	}
	for index := 0; index < delayedCount; index++ {
		reordered[reorderedCount] = delayed[index]
		reorderedCount++
	}
	reordered = ensureRenderLayerPresent(reordered, humanoidLayerBody)
	reordered = ensureRenderLayerPresent(reordered, humanoidLayerHead)
	return reordered
}

func isHeadAccessoryLayer(layer int) bool {
	return layer == humanoidLayerAccessoryBottom || layer == humanoidLayerAccessoryMid || layer == humanoidLayerAccessoryTop
}

func ensureRenderLayerPresent(order [humanoidLayerCount]int, required int) [humanoidLayerCount]int {
	counts := make(map[int]int, len(order))
	for _, layer := range order {
		counts[layer]++
	}
	if counts[required] > 0 {
		return order
	}
	for index := len(order) - 1; index >= 0; index-- {
		layer := order[index]
		if counts[layer] > 1 {
			counts[layer]--
			order[index] = required
			return order
		}
	}
	order[len(order)-1] = required
	return order
}

func resolveSpriteAction(act *res.ACT, actionFamily, direction int) (int, res.ACTAction, bool) {
	if act == nil || len(act.Actions) == 0 {
		return 0, res.ACTAction{}, false
	}
	direction = normalizeDirectionIndex(direction)
	preferred := actionFamily*8 + direction
	if preferred >= 0 && preferred < len(act.Actions) && len(act.Actions[preferred].Animations) > 0 {
		return preferred, act.Actions[preferred], true
	}
	if len(act.Actions) == 8 && direction < len(act.Actions) && len(act.Actions[direction].Animations) > 0 {
		return direction, act.Actions[direction], true
	}
	if actionFamily >= 0 && actionFamily < len(act.Actions) && len(act.Actions[actionFamily].Animations) > 0 {
		return actionFamily, act.Actions[actionFamily], true
	}
	base := actionFamily * 8
	if base >= 0 && base < len(act.Actions) && len(act.Actions[base].Animations) > 0 {
		return base, act.Actions[base], true
	}
	for index, action := range act.Actions {
		if len(action.Animations) > 0 {
			return index, action, true
		}
	}
	return 0, res.ACTAction{}, false
}

func spriteMotionIndex(action res.ACTAction, started time.Time, now time.Time, loop bool) int {
	return spriteMotionIndexWithDelay(action, started, now, loop, float64(action.DelayMS))
}

func spriteMotionIndexWithDelay(action res.ACTAction, started time.Time, now time.Time, loop bool, delayMS float64) int {
	return spriteMotionIndexWithOptions(action, started, now, loop, delayMS, 0, 0)
}

func spriteMotionIndexWithOptions(action res.ACTAction, started time.Time, now time.Time, loop bool, delayMS float64, frameLimit int, frameOffset int) int {
	if len(action.Animations) == 0 {
		return 0
	}
	motionCount := len(action.Animations)
	if frameLimit > 0 && frameLimit < motionCount {
		motionCount = frameLimit
	}
	delay := delayMS
	if delay <= 0 {
		delay = 150
	}
	elapsed := now.Sub(started)
	if elapsed < 0 {
		elapsed = 0
	}
	index := int(float64(elapsed.Milliseconds()) / delay)
	if loop || motionCount == 1 {
		return (index + frameOffset) % motionCount
	}
	if index >= motionCount {
		return minInt(frameOffset+motionCount-1, len(action.Animations)-1)
	}
	return minInt(frameOffset+index, len(action.Animations)-1)
}

func bodyMotionForState(action res.ACTAction, state spriteState, started time.Time, now time.Time) int {
	if state.hasFixedMotion && len(action.Animations) > 0 {
		if state.fixedMotion >= len(action.Animations) {
			return len(action.Animations) - 1
		}
		return state.fixedMotion
	}
	if !state.started.IsZero() {
		started = state.started
	}
	delayMS := float64(action.DelayMS)
	if state.hasSpeed && state.speed > 0 {
		delayMS = float64(state.speed / time.Millisecond)
	}
	if state.hasPlay && !state.play {
		if state.hasFrameOffset {
			return minInt(maxInt(0, state.frameOffset), len(action.Animations)-1)
		}
		return 0
	}
	if state.actionFamily == spriteActionWalk && state.moveSpeedMS > 0 {
		if delayMS <= 0 {
			delayMS = 150
		}
		delayMS = delayMS * float64(state.moveSpeedMS) / 150
	}
	if state.actionFamily == spriteActionWalk && state.walkDistance > 0 {
		return walkMotionIndex(action, state.walkDistance)
	}
	if state.headTurn && (state.actionFamily == spriteActionIdle || state.actionFamily == spriteActionSit) && int(state.headDir) < len(action.Animations) {
		return int(state.headDir)
	}
	frameLimit := 0
	if state.hasLength {
		frameLimit = state.length
	}
	frameOffset := 0
	if state.hasFrameOffset {
		frameOffset = state.frameOffset
	}
	if state.actionFamily == spriteActionWalk || state.loop {
		return spriteMotionIndexWithOptions(action, started, now, true, delayMS, frameLimit, frameOffset)
	}
	if state.actionFamily == spriteActionIdle && state.loopIdle {
		return spriteMotionIndexWithOptions(action, started, now, true, delayMS, frameLimit, frameOffset)
	}
	if !state.started.IsZero() {
		return spriteMotionIndexWithOptions(action, started, now, false, delayMS, frameLimit, frameOffset)
	}
	if state.actionFamily != spriteActionWalk {
		return 0
	}
	return spriteMotionIndexWithOptions(action, started, now, true, delayMS, frameLimit, frameOffset)
}

const walkDistanceToMotion = 4.6 * 0.37 * 4 * 25

func walkMotionIndex(action res.ACTAction, distance float64) int {
	if len(action.Animations) == 0 {
		return 0
	}
	delay := float64(action.DelayMS)
	if delay <= 0 {
		delay = 150
	}
	motion := int(math.Floor(distance * walkDistanceToMotion / delay))
	if motion < 0 {
		return 0
	}
	return motion % len(action.Animations)
}

func selectHeadMotion(state spriteState, bodyMotion int, headAction res.ACTAction) int {
	if len(headAction.Animations) == 0 {
		return 0
	}
	if state.headTurn && (state.actionFamily == spriteActionIdle || state.actionFamily == spriteActionSit) && int(state.headDir) < len(headAction.Animations) {
		return int(state.headDir)
	}
	if state.actionFamily == spriteActionWalk && bodyMotion >= 0 && bodyMotion < len(headAction.Animations) {
		return bodyMotion
	}
	if isTransientPCAction(state.actionFamily) && bodyMotion >= 0 && bodyMotion < len(headAction.Animations) {
		return bodyMotion
	}
	return 0
}

func isTransientPCAction(actionFamily int) bool {
	switch actionFamily {
	case spriteActionPickup, spriteActionPCReadyFight, spriteActionPCAttack1, spriteActionPCHurt, spriteActionPCDeath, spriteActionPCAttack2, spriteActionPCAttack3, spriteActionPCSkill:
		return true
	default:
		return false
	}
}

func drawSpriteAnimation(target *render.Image, view *spriteView, anim res.ACTAnimation, anchorX, anchorY float64, posX, posY int32) bool {
	return drawSpriteAnimationWithOptions(target, view, anim, anchorX, anchorY, posX, posY, false)
}

func drawSpriteAnimationWithOptions(target *render.Image, view *spriteView, anim res.ACTAnimation, anchorX, anchorY float64, posX, posY int32, ignoreLayerAngles bool) bool {
	rendered := false
	for _, layer := range anim.Layers {
		if layer.Index < 0 {
			continue
		}
		img, ok := spriteViewImage(view, layer.Index, layer.SPRType)
		if !ok {
			continue
		}
		drawSpriteLayerWithOptions(target, img, layer, anchorX+float64(posX), anchorY+float64(posY), ignoreLayerAngles)
		rendered = true
	}
	return rendered
}

func drawSpriteLayerByValue(target *render.Image, view *spriteView, layer res.ACTLayer, centerX, centerY float64) bool {
	if layer.Index < 0 {
		return false
	}
	img, ok := spriteViewImage(view, layer.Index, layer.SPRType)
	if !ok {
		return false
	}
	drawSpriteLayer(target, img, layer, centerX, centerY)
	return true
}

func attachmentDelta(baseAnim, attachedAnim res.ACTAnimation) (int32, int32) {
	if len(baseAnim.Pos) == 0 || len(attachedAnim.Pos) == 0 {
		return 0, 0
	}
	attached := attachedAnim.Pos[0]
	for _, base := range baseAnim.Pos {
		if base.Attr == attached.Attr {
			return base.X - attached.X, base.Y - attached.Y
		}
	}
	base := baseAnim.Pos[0]
	return base.X - attached.X, base.Y - attached.Y
}

func spriteViewImage(view *spriteView, index int32, sprType int32) (*render.Image, bool) {
	key := spriteFrameKey{index: index, sprType: sprType}
	if img, ok := view.images[key]; ok {
		return img, true
	}
	frame, ok := view.spr.FrameImageWithPalette(int(index), int(sprType), view.palette)
	if !ok {
		return nil, false
	}
	img := render.NewImageFromImage(frame)
	view.images[key] = img
	return img, true
}

func drawSpriteLayer(target *render.Image, img *render.Image, layer res.ACTLayer, centerX, centerY float64) {
	drawSpriteLayerWithOptions(target, img, layer, centerX, centerY, false)
}

func drawSpriteLayerWithOptions(target *render.Image, img *render.Image, layer res.ACTLayer, centerX, centerY float64, ignoreLayerAngle bool) {
	bounds := img.Bounds()
	width := float64(bounds.Dx())
	height := float64(bounds.Dy())
	scaleX := float64(layer.ScaleX)
	scaleY := float64(layer.ScaleY)
	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}
	if layer.Mirror {
		scaleX = -scaleX
	}

	var opts render.DrawImageOptions
	opts.GeoM.Translate(-width/2, -height/2)
	opts.GeoM.Scale(scaleX, scaleY)
	if layer.Angle != 0 && !ignoreLayerAngle {
		opts.GeoM.Rotate(float64(-layer.Angle) * math.Pi / 180)
	}
	layerCenterX, layerCenterY := spriteLayerCenter(centerX, centerY, layer)
	opts.GeoM.Translate(layerCenterX, layerCenterY)
	opts.Filter = spriteCompositionFilter()
	opts.ColorScale.Scale(layer.Color[0], layer.Color[1], layer.Color[2], layer.Color[3])
	target.DrawImage(img, &opts)
}

func spriteLayerCenter(centerX, centerY float64, layer res.ACTLayer) (float64, float64) {
	return centerX + float64(layer.X), centerY + float64(layer.Y)
}

func normalizeDirectionIndex(direction int) int {
	direction %= 8
	if direction < 0 {
		direction += 8
	}
	return direction
}

func spriteDirectionFromWorldDir(direction int) int {
	return spriteDirectionFromWorldDirForCamera(direction, 0)
}

func spriteDirectionFromWorldDirForCamera(direction int, cameraYaw float64) int {
	yawSteps := int(math.Round(cameraYaw / 45))
	return (4 - normalizeDirectionIndex(direction) + yawSteps) & 7
}
