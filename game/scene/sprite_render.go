package scene

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	SpriteActionIdle = iota
	SpriteActionWalk
	SpriteActionSit
	SpriteActionPickup
)

const (
	HumanoidBillboardWidth   = 224
	HumanoidBillboardHeight  = 160
	HumanoidBillboardAnchorX = 80
	HumanoidBillboardAnchorY = 120
	// reference client lifts entity sprites by 0.2 map units before depth testing.
	// Keeping actor billboards exactly coplanar with terrain lets sloped GND
	// triangles clip the bottom pixels of tight NPC/mob sprite frames.
	actorSpriteTerrainLift = 0.2
	ActorShadowTerrainLift = 0.05
	ActorShadowDepthBias   = GroundDecalDepthBias
)

type SpriteView struct {
	Spr           *res.SPR
	Act           *res.ACT
	ActSource     string
	Source        string
	Palette       *res.Palette
	PaletteSource string
	Images        map[SpriteFrameKey]*render.Image
	Billboards    map[SpriteBillboardKey]*Billboard
	Started       time.Time
}

type SpriteFrameKey struct {
	Index   int32
	sprType int32
}

type SpriteBillboardKey struct {
	ActionIndex       int
	Motion            int
	AnchorX           float64
	AnchorY           float64
	IgnoreLayerAngles bool
}

type Billboard struct {
	Image   *render.Image
	AnchorX float64
	AnchorY float64

	Parts      []*Billboard
	LayerGroup int
}

func ActorSpriteWorldZ(terrainZ float64) float64 {
	return terrainZ + actorSpriteTerrainLift
}

func DrawSpriteBillboardTintAlpha3D(screen *render.Frame, projection Projection, billboard *Billboard, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64, tintColor color.RGBA) {
	DrawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, billboard.Image, worldX, worldY, worldZ, scale, alpha, shadow, tintColor, SpriteBillboardTriangleDrawOptions())
}

func DrawSpriteBillboardTintAlpha3DWithOptions(screen *render.Frame, projection Projection, billboard *Billboard, texture *render.Image, worldX, worldY, worldZ, scale float64, alpha float64, shadow float64, tintColor color.RGBA, options *render.DrawTrianglesOptions) {
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
	// Keep geometry and depth in original pixels, independently of texture resolution.
	bounds := billboard.Image.Bounds()
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	center := Point3{X: worldX, Y: worldZ, Z: worldY}
	tint := colorRGBAFromFloats(tintR, tintG, tintB, tintA)
	axisScale := scale * unitsPerPixel
	screen.DrawWorldBillboard(render.WorldBillboardCommand{
		Texture:     texture,
		Options:     *options,
		Center:      [3]float32{float32(center.X), float32(center.Y), float32(center.Z)},
		RightAxis:   [3]float32{float32(right.X * axisScale), float32(right.Y * axisScale), float32(right.Z * axisScale)},
		UpAxis:      [3]float32{float32(-up.X * axisScale), float32(-up.Y * axisScale), float32(-up.Z * axisScale)},
		DepthUpAxis: [3]float32{0, float32(-axisScale), 0},
		Width:       float32(w),
		Height:      float32(h),
		AnchorX:     float32(billboard.AnchorX),
		AnchorY:     float32(billboard.AnchorY),
		ColorR:      float32(tint.R) / 255,
		ColorG:      float32(tint.G) / 255,
		ColorB:      float32(tint.B) / 255,
		ColorA:      float32(tint.A) / 255,
		DepthBias:   options.DepthBias,
	})
}

func DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen *render.Frame, projection Projection, billboard *Billboard, worldX, worldY, worldZ, pixelScale, angle float64, alpha float64, shadow float64, tintColor color.RGBA, options *render.DrawTrianglesOptions) {
	if screen == nil || billboard == nil || billboard.Image == nil {
		return
	}
	if options == nil {
		options = SpriteBillboardTriangleDrawOptions()
	}
	if pixelScale <= 0 || math.IsNaN(pixelScale) || math.IsInf(pixelScale, 0) {
		pixelScale = PixelRatio
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
	rightAxis := Mul3(right, pixelScale)
	upAxis := Mul3(up, -pixelScale)
	if angle != 0 {
		sinA, cosA := math.Sin(angle), math.Cos(angle)
		rightAxis = Add3(Mul3(right, cosA*pixelScale), Mul3(up, sinA*pixelScale))
		upAxis = Add3(Mul3(right, sinA*pixelScale), Mul3(up, -cosA*pixelScale))
	}
	tint := colorRGBAFromFloats(
		float64(tintColor.R)/255*shadow,
		float64(tintColor.G)/255*shadow,
		float64(tintColor.B)/255*shadow,
		float64(tintColor.A)/255*alpha,
	)
	bounds := billboard.Image.Bounds()
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	center := Point3{X: worldX, Y: worldZ, Z: worldY}
	cmd := WorldSpriteBillboardCommand(billboard.Image, *options, center, rightAxis, upAxis, float32(w), float32(h), float32(billboard.AnchorX), float32(billboard.AnchorY), tint)
	screen.DrawWorldBillboard(cmd)
}

func WorldSpriteBillboardCommand(texture *render.Image, options render.DrawTrianglesOptions, center, rightAxis, upAxis Point3, width, height, anchorX, anchorY float32, tint color.RGBA) render.WorldBillboardCommand {
	// robr's SpriteRenderer depth correction projects vertical billboards onto a
	// center-anchored plane, so the lower edge of map effect sprites does not
	// sink behind the floor.
	return render.WorldBillboardCommand{
		Texture:     texture,
		Options:     options,
		Center:      [3]float32{float32(center.X), float32(center.Y), float32(center.Z)},
		RightAxis:   [3]float32{float32(rightAxis.X), float32(rightAxis.Y), float32(rightAxis.Z)},
		UpAxis:      [3]float32{float32(upAxis.X), float32(upAxis.Y), float32(upAxis.Z)},
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

func SpriteBillboardTriangleDrawOptions() *render.DrawTrianglesOptions {
	options := TriangleDrawOptions(SpriteDrawFilter(), render.AddressClampToZero)
	options.Blend = render.BlendSourceOver
	return options
}

func colorRGBAFromFloats(r, g, b, a float64) color.RGBA {
	return color.RGBA{
		R: ClampColor(r * 255),
		G: ClampColor(g * 255),
		B: ClampColor(b * 255),
		A: ClampColor(a * 255),
	}
}

func SpriteDrawFilter() render.Filter {
	return render.FilterLinear
}

func SpriteCompositionFilter() render.Filter {
	return render.FilterNearest
}

func ComposeSingleSpriteBillboard(view *SpriteView, anim res.ACTAnimation) (*Billboard, bool) {
	return ComposeSingleSpriteBillboardWithOptions(view, anim, false)
}

func ComposeSingleSpriteBillboardWithOptions(view *SpriteView, anim res.ACTAnimation, ignoreLayerAngles bool) (*Billboard, bool) {
	minX, minY, maxX, maxY, ok := SpriteAnimationLayerBounds(view, anim, ignoreLayerAngles)
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
	if !DrawSpriteAnimationWithOptions(target, view, anim, anchorX, anchorY, 0, 0, ignoreLayerAngles) {
		return nil, false
	}
	return &Billboard{
		Image:   target,
		AnchorX: anchorX,
		AnchorY: anchorY,
	}, true
}

func SpriteAnimationLayerBounds(view *SpriteView, anim res.ACTAnimation, ignoreLayerAngles bool) (float64, float64, float64, float64, bool) {
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
		if layer.Angle != 0 && !ignoreLayerAngles {
			sin, cos := math.Sincos(float64(layer.Angle) * math.Pi / 180)
			sin, cos = math.Abs(sin), math.Abs(cos)
			width, height = width*cos+height*sin, width*sin+height*cos
		}
		centerX, centerY := SpriteLayerCenter(0, 0, layer)
		minX = math.Min(minX, centerX-width*0.5)
		maxX = math.Max(maxX, centerX+width*0.5)
		minY = math.Min(minY, centerY-height*0.5)
		maxY = math.Max(maxY, centerY+height*0.5)
		ok = true
	}
	return minX, minY, maxX, maxY, ok
}

func spriteLayerFrameSize(view *SpriteView, index int32, sprType int32) (float64, float64, bool) {
	if view == nil || view.Spr == nil {
		return 0, 0, false
	}
	frameIndex := int(index)
	if sprType == res.SPRFrameRGBA {
		frameIndex += view.Spr.RGBAIndex
	}
	if frameIndex < 0 || frameIndex >= len(view.Spr.Frames) {
		return 0, 0, false
	}
	frame := view.Spr.Frames[frameIndex]
	if frame.Width <= 0 || frame.Height <= 0 {
		return 0, 0, false
	}
	return float64(frame.Width), float64(frame.Height), true
}

func SpriteMotionIndexWithDelay(action res.ACTAction, started time.Time, now time.Time, loop bool, delayMS float64) int {
	return SpriteMotionIndexWithOptions(action, started, now, loop, delayMS, 0, 0)
}

func SpriteMotionIndexWithOptions(action res.ACTAction, started time.Time, now time.Time, loop bool, delayMS float64, frameLimit int, frameOffset int) int {
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
	index := int(float64(elapsed) / (delay * float64(time.Millisecond)))
	if loop || motionCount == 1 {
		return MinInt(frameOffset+index%motionCount, len(action.Animations)-1)
	}
	if index >= motionCount {
		return MinInt(frameOffset+motionCount-1, len(action.Animations)-1)
	}
	return MinInt(frameOffset+index, len(action.Animations)-1)
}

func DrawSpriteAnimationWithOptions(target *render.Image, view *SpriteView, anim res.ACTAnimation, anchorX, anchorY float64, posX, posY int32, ignoreLayerAngles bool) bool {
	rendered := false
	for _, layer := range anim.Layers {
		if layer.Index < 0 {
			continue
		}
		img, ok := SpriteViewImage(view, layer.Index, layer.SPRType)
		if !ok {
			continue
		}
		DrawSpriteLayerWithOptions(target, img, layer, anchorX+float64(posX), anchorY+float64(posY), ignoreLayerAngles)
		rendered = true
	}
	return rendered
}

func SpriteViewImage(view *SpriteView, index int32, sprType int32) (*render.Image, bool) {
	key := SpriteFrameKey{Index: index, sprType: sprType}
	if img, ok := view.Images[key]; ok {
		return img, true
	}
	frame, ok := view.Spr.FrameImageWithPalette(int(index), int(sprType), view.Palette)
	if !ok {
		return nil, false
	}
	img := render.NewImageFromStraightAlpha(frame)
	view.Images[key] = img
	return img, true
}

func DrawSpriteLayerWithOptions(target *render.Image, img *render.Image, layer res.ACTLayer, centerX, centerY float64, ignoreLayerAngle bool) {
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
		// Positive ACT angles rotate clockwise in screen coordinates.
		opts.GeoM.Rotate(float64(layer.Angle) * math.Pi / 180)
	}
	layerCenterX, layerCenterY := SpriteLayerCenter(centerX, centerY, layer)
	opts.GeoM.Translate(layerCenterX, layerCenterY)
	opts.Filter = SpriteCompositionFilter()
	opts.ColorScale.Scale(layer.Color[0], layer.Color[1], layer.Color[2], layer.Color[3])
	target.DrawImage(img, &opts)
}

func SpriteLayerCenter(centerX, centerY float64, layer res.ACTLayer) (float64, float64) {
	return centerX + float64(layer.X), centerY + float64(layer.Y)
}

func NormalizeDirectionIndex(direction int) int {
	direction %= 8
	if direction < 0 {
		direction += 8
	}
	return direction
}
