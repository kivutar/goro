package sprites

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	ActionNonPCAttack  = 2
	ActionNonPCHurt    = 3
	ActionNonPCDeath   = 4
	ActionNonPCPerf1   = 5
	ActionNonPCPerf2   = 6
	ActionNonPCPerf3   = 7
	ActionNonPCSpecial = 8
	ActionPCReadyFight = 4
	ActionPCAttack1    = 5
	ActionPCHurt       = 6
	ActionPCDeath      = 8
	ActionPCFreeze2    = 9
	ActionPCAttack2    = 10
	ActionPCAttack3    = 11
	ActionPCSkill      = 12
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

type Humanoid struct {
	Body            *scene.SpriteView
	head            *scene.SpriteView
	accessoryBottom *scene.SpriteView
	accessoryMid    *scene.SpriteView
	accessoryTop    *scene.SpriteView
	weapon          *scene.SpriteView
	weaponLight     *scene.SpriteView
	shield          *scene.SpriteView
	imf             *res.IMF
	imfSource       string
	billboards      map[humanoidBillboardKey]*scene.Billboard
	Started         time.Time
}

type Appearance struct {
	Job         int
	Head        int
	Sex         byte
	Admin       bool
	BodyPalette int
	HeadPalette int
	Weapon      int
	Shield      int
	HeadTop     int
	HeadMid     int
	HeadLow     int
}

type humanoidBillboardKey struct {
	actionFamily int
	direction    int
	bodyMotion   int
	headMotion   int

	separateEquipment bool
}

type State struct {
	SeparateEquipment bool

	ActionFamily   int
	Direction      int
	HeadDir        uint8
	HeadTurn       bool
	CameraYaw      float64
	Moving         bool
	Started        time.Time
	Loop           bool
	PingPong       bool
	LoopIdle       bool
	Play           bool
	HasPlay        bool
	Length         int
	HasLength      bool
	FrameOffset    int
	HasFrameOffset bool
	FixedMotion    int
	HasFixedMotion bool
	Speed          time.Duration
	HasSpeed       bool
	MoveSpeedMS    int
	WalkDistance   float64
}

func DrawFixedShadow3D(screen *render.Frame, projection scene.Projection, view *scene.SpriteView, worldX, worldY, worldZ, scale float64, alpha float64, shadow scene.LightTint) bool {
	billboard, ok := FixedBillboard(view)
	if !ok {
		return false
	}
	return DrawShadow3D(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, shadow)
}

func DrawShadow3D(screen *render.Frame, projection scene.Projection, billboard *scene.Billboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow scene.LightTint) bool {
	if billboard == nil {
		return false
	}
	options := scene.SpriteBillboardTriangleDrawOptions()
	options.DepthBias = scene.ActorShadowDepthBias
	scene.DrawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, billboard.Image, worldX, worldY, worldZ, scale, alpha, shadow, color.RGBA{R: 255, G: 255, B: 255, A: 255}, options)
	return true
}

func DrawOverlay3D(screen *render.Frame, projection scene.Projection, billboard *scene.Billboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow scene.LightTint, tintColor color.RGBA) {
	scene.DrawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, billboard.Image, worldX, worldY, worldZ, scale, alpha, shadow, tintColor, &render.DrawTrianglesOptions{
		Filter:  scene.SpriteDrawFilter(),
		Address: render.AddressClampToZero,
	})
}

func HumanoidBillboard(view *Humanoid, state State, now time.Time) (*scene.Billboard, bool) {
	if view == nil || view.Body == nil || view.Body.Act == nil || view.Body.Spr == nil {
		return nil, false
	}
	state.Direction = DirectionForCamera(state.Direction, state.CameraYaw)
	bodyActionIndex, bodyAction, ok := ResolveAction(view.Body.Act, state.ActionFamily, state.Direction)
	if !ok || len(bodyAction.Animations) == 0 {
		return nil, false
	}
	bodyMotion := BodyMotionForState(bodyAction, state, view.Started, now)
	headMotion := 0
	if view.head != nil {
		if _, headAction, headOK := ResolveAction(view.head.Act, state.ActionFamily, state.Direction); headOK && len(headAction.Animations) > 0 {
			headMotion = selectHeadMotion(state, bodyMotion, headAction)
		}
	}
	key := humanoidBillboardKey{
		separateEquipment: state.SeparateEquipment,
		actionFamily:      bodyActionIndex / 8,
		direction:         bodyActionIndex % 8,
		bodyMotion:        bodyMotion,
		headMotion:        headMotion,
	}
	if billboard, ok := view.billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := composeHumanoidBillboard(view, key.actionFamily, key.direction, bodyAction, bodyMotion, headMotion, key.separateEquipment)
	if !ok {
		return nil, false
	}
	if view.billboards == nil {
		view.billboards = make(map[humanoidBillboardKey]*scene.Billboard)
	}
	view.billboards[key] = billboard
	return billboard, true
}

func BillboardForState(view *scene.SpriteView, state State, now time.Time) (*scene.Billboard, bool) {
	if view == nil || view.Act == nil || view.Spr == nil {
		return nil, false
	}
	state.Direction = DirectionForCamera(state.Direction, state.CameraYaw)
	actionIndex, action, ok := ResolveAction(view.Act, state.ActionFamily, state.Direction)
	if !ok || len(action.Animations) == 0 {
		return nil, false
	}
	motion := BodyMotionForState(action, state, view.Started, now)
	key := scene.SpriteBillboardKey{ActionIndex: actionIndex, Motion: motion}
	if billboard, ok := view.Billboards[key]; ok {
		return billboard, true
	}
	if motion < 0 || motion >= len(action.Animations) {
		return nil, false
	}
	billboard, ok := scene.ComposeSingleSpriteBillboard(view, action.Animations[motion])
	if !ok {
		return nil, false
	}
	view.Billboards[key] = billboard
	return billboard, true
}

func FixedBillboard(view *scene.SpriteView) (*scene.Billboard, bool) {
	if view == nil || view.Act == nil || view.Spr == nil || len(view.Act.Actions) == 0 || len(view.Act.Actions[0].Animations) == 0 {
		return nil, false
	}
	key := scene.SpriteBillboardKey{ActionIndex: 0, Motion: 0}
	if billboard, ok := view.Billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := scene.ComposeSingleSpriteBillboard(view, view.Act.Actions[0].Animations[0])
	if !ok {
		return nil, false
	}
	view.Billboards[key] = billboard
	return billboard, true
}

func CursorFrameBillboard(view *scene.SpriteView, actionIndex, motion int, anchorX, anchorY float64) (*scene.Billboard, bool) {
	if view == nil || view.Act == nil || view.Spr == nil || len(view.Act.Actions) == 0 {
		return nil, false
	}
	if actionIndex < 0 || actionIndex >= len(view.Act.Actions) || len(view.Act.Actions[actionIndex].Animations) == 0 {
		actionIndex = 0
	}
	action := view.Act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return nil, false
	}
	if motion < 0 {
		motion = 0
	}
	motion %= len(action.Animations)
	key := scene.SpriteBillboardKey{ActionIndex: actionIndex, Motion: motion, AnchorX: anchorX, AnchorY: anchorY}
	if billboard, ok := view.Billboards[key]; ok {
		return billboard, true
	}
	anim := action.Animations[motion]
	minX, minY, maxX, maxY, ok := scene.SpriteAnimationLayerBounds(view, anim, false)
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
	if !DrawAnimation(target, view, anim, compositionAnchorX, compositionAnchorY, 0, 0) {
		return nil, false
	}
	billboard := &scene.Billboard{
		Image:   target,
		AnchorX: compositionAnchorX,
		AnchorY: compositionAnchorY,
	}
	view.Billboards[key] = billboard
	return billboard, true
}

func composeHumanoidBillboard(view *Humanoid, actionFamily, direction int, bodyAction res.ACTAction, bodyMotion, headMotion int, separateEquipment bool) (*scene.Billboard, bool) {
	if bodyMotion < 0 || bodyMotion >= len(bodyAction.Animations) {
		return nil, false
	}
	target := render.NewImage(humanoidCompositionWidth, humanoidCompositionHeight)
	bodyActionIndex := actionFamily*8 + direction
	order := playerRenderLayerOrder(view.imf, bodyActionIndex, bodyMotion)
	if separateEquipment {
		billboard := &scene.Billboard{}
		// Keep IMF order, including equipment drawn behind the body.
		for first := 0; first < len(order); {
			group := humanoidLayerGroup(order[first])
			last := first + 1
			for last < len(order) && humanoidLayerGroup(order[last]) == group {
				last++
			}
			target.Fill(color.RGBA{})
			if view.imf != nil {
				drawPlayerIMFLayers(target, view, bodyActionIndex, bodyMotion, headMotion, order[first:last])
			} else {
				drawFallbackHumanoidLayers(target, view, actionFamily, direction, bodyAction, bodyMotion, headMotion, order[first:last])
			}
			if part, ok := cropHumanoidBillboard(target); ok {
				part.LayerGroup = group
				billboard.Parts = append(billboard.Parts, part)
			}
			first = last
		}
		return billboard, len(billboard.Parts) > 0
	}
	drawn := false
	if view.imf != nil {
		drawn = drawPlayerIMFLayers(target, view, bodyActionIndex, bodyMotion, headMotion, order[:])
	} else {
		drawn = drawFallbackHumanoidLayers(target, view, actionFamily, direction, bodyAction, bodyMotion, headMotion, order[:])
	}
	if !drawn {
		return nil, false
	}
	return cropHumanoidBillboard(target)
}

func humanoidLayerGroup(layer int) int {
	switch layer {
	case humanoidLayerWeapon, humanoidLayerWeaponLight:
		return 1
	case humanoidLayerShield:
		return 2
	default:
		return 0
	}
}

func cropHumanoidBillboard(source *render.Image) (*scene.Billboard, bool) {
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
	return &scene.Billboard{
		Image:   cropped,
		AnchorX: humanoidCompositionAnchorX - float64(minX),
		AnchorY: humanoidCompositionAnchorY - float64(minY),
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

func drawFallbackHumanoidLayers(target *render.Image, view *Humanoid, actionFamily, direction int, bodyAction res.ACTAction, bodyMotion, headMotion int, order []int) bool {
	bodyAnim := bodyAction.Animations[bodyMotion]
	actionIndex := actionFamily*8 + direction
	drawn := false
	for _, layer := range order {
		switch layer {
		case humanoidLayerBody:
			drawn = DrawAnimation(target, view.Body, bodyAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, 0, 0) || drawn
		case humanoidLayerHead:
			drawn = drawFallbackHeadMotion(target, view, bodyAnim, actionIndex, headMotion) || drawn
		case humanoidLayerAccessoryBottom:
			drawn = drawAttachedAccessoryMotion(target, view.accessoryBottom, view.head, bodyAnim, actionIndex, headMotion) || drawn
		case humanoidLayerAccessoryMid:
			drawn = drawAttachedAccessoryMotion(target, view.accessoryMid, view.head, bodyAnim, actionIndex, headMotion) || drawn
		case humanoidLayerAccessoryTop:
			drawn = drawAttachedAccessoryMotion(target, view.accessoryTop, view.head, bodyAnim, actionIndex, headMotion) || drawn
		case humanoidLayerWeapon:
			drawn = drawFallbackOverlayMotion(target, view.weapon, view.Body, actionIndex, bodyMotion) || drawn
		case humanoidLayerWeaponLight:
			drawn = drawFallbackOverlayMotion(target, view.weaponLight, view.Body, actionIndex, bodyMotion) || drawn
		case humanoidLayerShield:
			drawn = drawFallbackOverlayMotion(target, view.shield, view.Body, actionIndex, bodyMotion) || drawn
		}
	}
	return drawn
}

func drawFallbackHeadMotion(target *render.Image, view *Humanoid, bodyAnim res.ACTAnimation, actionIndex, headMotion int) bool {
	if view.head == nil || view.head.Act == nil || view.head.Spr == nil {
		return false
	}
	headAnim, ok := actionAnimation(view.head.Act, actionIndex, headMotion)
	if !ok {
		headAnim, ok = actionAnimation(view.head.Act, actionIndex, 0)
	}
	if !ok {
		return false
	}
	headPosX, headPosY := attachmentDelta(bodyAnim, headAnim)
	return DrawAnimation(target, view.head, headAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, headPosX, headPosY)
}

func drawPlayerIMFLayers(target *render.Image, view *Humanoid, actionIndex, bodyMotion, headMotion int, order []int) bool {
	bodyAnim, bodyAnimOK := actionAnimation(view.Body.Act, actionIndex, bodyMotion)
	drawn := false
	for _, layer := range order {
		switch layer {
		case humanoidLayerBody:
			drawn = drawPlayerIMFLayer(target, view.Body, view.imf, humanoidLayerBody, actionIndex, bodyMotion, nil) || drawn
		case humanoidLayerHead:
			if view.head == nil || view.head.Act == nil || view.head.Spr == nil {
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
			drawn = drawPlayerOverlayMotion(target, view.weapon, view.Body, view.imf, humanoidLayerWeapon, actionIndex, bodyMotion) || drawn
		case humanoidLayerWeaponLight:
			drawn = drawPlayerOverlayMotion(target, view.weaponLight, view.Body, view.imf, humanoidLayerWeaponLight, actionIndex, bodyMotion) || drawn
		case humanoidLayerShield:
			drawn = drawPlayerOverlayMotion(target, view.shield, view.Body, view.imf, humanoidLayerShield, actionIndex, bodyMotion) || drawn
		}
	}
	return drawn
}

func drawPlayerIMFLayer(target *render.Image, sprite *scene.SpriteView, imf *res.IMF, layerPriority, actionIndex, motionIndex int, attachBase *res.ACTAnimation) bool {
	if sprite == nil || sprite.Act == nil || sprite.Spr == nil {
		return false
	}
	if actionIndex < 0 || actionIndex >= len(sprite.Act.Actions) {
		return false
	}
	action := sprite.Act.Actions[actionIndex]
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

func drawAttachedAccessoryMotion(target *render.Image, accessory *scene.SpriteView, head *scene.SpriteView, bodyAnim res.ACTAnimation, actionIndex, headMotionIndex int) bool {
	if accessory == nil || accessory.Act == nil || accessory.Spr == nil || head == nil || head.Act == nil || head.Spr == nil {
		return false
	}
	headAnim, ok := actionAnimation(head.Act, actionIndex, headMotionIndex)
	if !ok {
		return false
	}
	accessoryMotion := headMotionIndex
	accessoryAnim, ok := actionAnimation(accessory.Act, actionIndex, accessoryMotion)
	if !ok {
		accessoryAnim, ok = actionAnimation(accessory.Act, actionIndex, 0)
	}
	if !ok {
		return false
	}
	headDX, headDY := attachmentDelta(bodyAnim, headAnim)
	accessoryDX, accessoryDY := attachmentDelta(headAnim, accessoryAnim)
	return DrawAnimation(target, accessory, accessoryAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, headDX+accessoryDX, headDY+accessoryDY)
}

func drawPlayerOverlayMotion(target *render.Image, overlay *scene.SpriteView, body *scene.SpriteView, imf *res.IMF, layerPriority, actionIndex, bodyMotion int) bool {
	if overlay == nil || overlay.Act == nil || overlay.Spr == nil || body == nil || body.Act == nil || imf == nil {
		return false
	}
	motionIndex := resolveOverlayMotionIndex(overlay.Act, body.Act, actionIndex, bodyMotion)
	overlayAnim, ok := actionAnimation(overlay.Act, actionIndex, motionIndex)
	if !ok {
		overlayAnim, ok = actionAnimation(overlay.Act, actionIndex, 0)
	}
	if !ok {
		return false
	}
	pointX, pointY := imf.Point(layerPriority, actionIndex, bodyMotion)
	return DrawAnimation(target, overlay, overlayAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, pointX, pointY)
}

func drawFallbackOverlayMotion(target *render.Image, overlay *scene.SpriteView, body *scene.SpriteView, actionIndex, bodyMotion int) bool {
	if overlay == nil || overlay.Act == nil || overlay.Spr == nil || body == nil || body.Act == nil {
		return false
	}
	if len(overlay.Act.Actions) == 0 {
		return false
	}
	resolvedActionIndex := actionIndex % len(overlay.Act.Actions)
	if resolvedActionIndex < 0 {
		resolvedActionIndex += len(overlay.Act.Actions)
	}
	motionIndex := resolveOverlayMotionIndex(overlay.Act, body.Act, resolvedActionIndex, bodyMotion)
	overlayAnim, ok := actionAnimation(overlay.Act, resolvedActionIndex, motionIndex)
	if !ok {
		overlayAnim, ok = actionAnimation(overlay.Act, resolvedActionIndex, 0)
	}
	if !ok {
		return false
	}
	return DrawAnimation(target, overlay, overlayAnim, humanoidCompositionAnchorX, humanoidCompositionAnchorY, 0, 0)
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
		var layer int
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

func ResolveAction(act *res.ACT, actionFamily, direction int) (int, res.ACTAction, bool) {
	if act == nil || len(act.Actions) == 0 {
		return 0, res.ACTAction{}, false
	}
	direction = scene.NormalizeDirectionIndex(direction)
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

func MotionIndex(action res.ACTAction, started time.Time, now time.Time, loop bool) int {
	return scene.SpriteMotionIndexWithDelay(action, started, now, loop, float64(action.DelayMS))
}

func BodyMotionForState(action res.ACTAction, state State, started time.Time, now time.Time) int {
	if state.HasFixedMotion && len(action.Animations) > 0 {
		if state.FixedMotion >= len(action.Animations) {
			return len(action.Animations) - 1
		}
		return state.FixedMotion
	}
	if !state.Started.IsZero() {
		started = state.Started
	}
	delayMS := float64(action.DelayMS)
	if state.HasSpeed && state.Speed > 0 {
		delayMS = float64(state.Speed) / float64(time.Millisecond)
	}
	if state.HasPlay && !state.Play {
		if state.HasFrameOffset {
			return scene.MinInt(scene.MaxInt(0, state.FrameOffset), len(action.Animations)-1)
		}
		return 0
	}
	if state.ActionFamily == scene.SpriteActionWalk && state.MoveSpeedMS > 0 {
		if delayMS <= 0 {
			delayMS = 150
		}
		delayMS = delayMS * float64(state.MoveSpeedMS) / 150
	}
	if state.ActionFamily == scene.SpriteActionWalk && state.WalkDistance > 0 {
		return walkMotionIndex(action, state.WalkDistance)
	}
	if state.HeadTurn && (state.ActionFamily == scene.SpriteActionIdle || state.ActionFamily == scene.SpriteActionSit) && int(state.HeadDir) < len(action.Animations) {
		return int(state.HeadDir)
	}
	frameLimit := 0
	if state.HasLength {
		frameLimit = state.Length
	}
	frameOffset := 0
	if state.HasFrameOffset {
		frameOffset = state.FrameOffset
	}
	if state.PingPong && frameLimit > 1 {
		index := int(max(0, float64(now.Sub(started))/float64(time.Millisecond))/max(1, delayMS)) % (2*frameLimit - 2)
		if index >= frameLimit {
			index = 2*frameLimit - 2 - index
		}
		return scene.MinInt(frameOffset+index, len(action.Animations)-1)
	}
	if state.ActionFamily == scene.SpriteActionWalk || state.Loop {
		return scene.SpriteMotionIndexWithOptions(action, started, now, true, delayMS, frameLimit, frameOffset)
	}
	if state.ActionFamily == scene.SpriteActionIdle && state.LoopIdle {
		return scene.SpriteMotionIndexWithOptions(action, started, now, true, delayMS, frameLimit, frameOffset)
	}
	if !state.Started.IsZero() {
		return scene.SpriteMotionIndexWithOptions(action, started, now, false, delayMS, frameLimit, frameOffset)
	}
	if state.ActionFamily != scene.SpriteActionWalk {
		return 0
	}
	return scene.SpriteMotionIndexWithOptions(action, started, now, true, delayMS, frameLimit, frameOffset)
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

func selectHeadMotion(state State, bodyMotion int, headAction res.ACTAction) int {
	if len(headAction.Animations) == 0 {
		return 0
	}
	if state.HeadTurn && (state.ActionFamily == scene.SpriteActionIdle || state.ActionFamily == scene.SpriteActionSit) && int(state.HeadDir) < len(headAction.Animations) {
		return int(state.HeadDir)
	}
	if state.ActionFamily == scene.SpriteActionWalk && bodyMotion >= 0 && bodyMotion < len(headAction.Animations) {
		return bodyMotion
	}
	if isTransientPCAction(state.ActionFamily) && bodyMotion >= 0 && bodyMotion < len(headAction.Animations) {
		return bodyMotion
	}
	return 0
}

func isTransientPCAction(actionFamily int) bool {
	switch actionFamily {
	case scene.SpriteActionPickup, ActionPCReadyFight, ActionPCAttack1, ActionPCHurt, ActionPCDeath, ActionPCAttack2, ActionPCAttack3, ActionPCSkill:
		return true
	default:
		return false
	}
}

func DrawAnimation(target *render.Image, view *scene.SpriteView, anim res.ACTAnimation, anchorX, anchorY float64, posX, posY int32) bool {
	return scene.DrawSpriteAnimationWithOptions(target, view, anim, anchorX, anchorY, posX, posY, false)
}

func drawSpriteLayerByValue(target *render.Image, view *scene.SpriteView, layer res.ACTLayer, centerX, centerY float64) bool {
	if layer.Index < 0 {
		return false
	}
	img, ok := scene.SpriteViewImage(view, layer.Index, layer.SPRType)
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

func drawSpriteLayer(target *render.Image, img *render.Image, layer res.ACTLayer, centerX, centerY float64) {
	scene.DrawSpriteLayerWithOptions(target, img, layer, centerX, centerY, false)
}

func DirectionForCamera(direction int, cameraYaw float64) int {
	yawSteps := int(math.Round(cameraYaw / 45))
	return (4 - scene.NormalizeDirectionIndex(direction) + yawSteps) & 7
}
