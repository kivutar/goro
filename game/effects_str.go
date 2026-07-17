package game

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *WorldMode) drawSTREffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, worldX, worldY, worldZ float64, now time.Time) bool {
	str := m.loadWorldEffectSTR(ctx.Resources, resolveEffectSTRFile(component, effect, lessEffectsEnabled(ctx)), component.texturePath)
	if str == nil {
		return false
	}
	fps := str.FPS
	if fps <= 0 {
		fps = 60
	}
	keyIndex := float64(now.Sub(effect.starts)) / float64(time.Second) * float64(fps)
	drawn := false
	for _, layer := range str.Layers {
		anim, ok := calculateSTRAnimation(layer, keyIndex)
		if !ok {
			continue
		}
		if math.IsNaN(float64(anim.AniFrame)) || math.IsInf(float64(anim.AniFrame), 0) {
			continue
		}
		textureIndex := int(anim.AniFrame)
		if textureIndex < 0 || textureIndex >= len(layer.Textures) {
			continue
		}
		texture := m.strEffectTexture(ctx.Resources, layer.Textures[textureIndex])
		if texture == nil {
			continue
		}
		drawSTRAnimation(screen, projection, texture, worldX, worldY, worldZ, anim, component.attachedEntity)
		drawn = true
	}
	return drawn
}

func resolveEffectSTRFile(component worldEffectComponent, effect worldEffect, lessEffects bool) string {
	if lessEffects && component.strMinFile != "" {
		return component.strMinFile
	}
	if component.strFile == "" || !strings.Contains(component.strFile, "%d") || component.strRandMax < component.strRandMin || component.strRandMin <= 0 {
		return component.strFile
	}
	span := component.strRandMax - component.strRandMin + 1
	index := component.strRandMin
	if span > 1 {
		value := uint32(effect.effectID*1103515245) ^ effect.actorID ^ uint32(effect.starts.UnixNano())
		value ^= value >> 16
		index += int(value % uint32(span))
	}
	return fmt.Sprintf(component.strFile, index)
}

func (m *WorldMode) loadWorldEffectSTR(manager *res.Manager, strFile, texturePath string) *res.STR {
	if manager == nil || strFile == "" {
		return nil
	}
	if m.strEffects == nil {
		m.strEffects = make(map[string]*res.STR)
	}
	if m.strEffectMiss == nil {
		m.strEffectMiss = make(map[string]struct{})
	}
	normalized := strings.ReplaceAll(strFile, "/", "\\")
	paths := []string{"data\\texture\\effect\\" + normalized + ".str"}
	if strings.ContainsAny(strFile, `/\`) {
		paths = append([]string{"data\\texture\\" + normalized + ".str"}, paths...)
	}
	attempted := false
	for _, path := range paths {
		key := "__str_" + path + "|" + texturePath
		if str, ok := m.strEffects[key]; ok {
			return str
		}
		if _, ok := m.strEffectMiss[key]; ok {
			continue
		}
		attempted = true
		data, err := manager.ReadFileExact(path)
		if err != nil {
			m.strEffectMiss[key] = struct{}{}
			continue
		}
		str, err := res.ParseSTR(data, texturePath)
		if err != nil {
			m.strEffectMiss[key] = struct{}{}
			glog.Warnf("str effect parse failed path=%s: %v", path, err)
			return nil
		}
		m.strEffects[key] = str
		return str
	}
	if attempted {
		glog.Warnf("str effect missing file=%s", strFile)
	}
	return nil
}

func (m *WorldMode) strEffectTexture(manager *res.Manager, path string) *render.Image {
	path = strings.TrimSpace(path)
	if manager == nil || path == "" {
		return nil
	}
	key := "__strtex_" + path
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	candidates := []string{path, strings.ReplaceAll(path, "\\", "/")}
	img, _, err := res.LoadImageExact(manager, candidates)
	if err != nil {
		m.textureMiss[key] = struct{}{}
		glog.Warnf("str effect texture missing path=%s: %v", path, err)
		return nil
	}
	texture := render.NewImageFromImage(res.ApplyEffectTransparency(img))
	m.textures[key] = texture
	return texture
}

func strEffectDuration(str *res.STR, fallback time.Duration) time.Duration {
	if str == nil || str.FPS <= 0 || str.MaxKey <= 0 {
		return fallback
	}
	duration := time.Duration(float64(str.MaxKey) / float64(str.FPS) * float64(time.Second))
	if duration <= 0 {
		return fallback
	}
	return duration + 100*time.Millisecond
}

func calculateSTRAnimation(layer res.STRLayer, keyIndex float64) (res.STRAnimation, bool) {
	animations := layer.Animations
	lastFrame := 0
	lastSource := 0
	fromID := -1
	toID := -1
	for i, anim := range animations {
		if float64(anim.Frame) <= keyIndex {
			if anim.Type == 0 {
				fromID = i
			}
			if anim.Type == 1 {
				toID = i
			}
		}
		if anim.Frame > lastFrame {
			lastFrame = anim.Frame
		}
		if anim.Type == 0 && anim.Frame > lastSource {
			lastSource = anim.Frame
		}
	}
	if fromID < 0 || (toID < 0 && float64(lastFrame) < keyIndex) {
		return res.STRAnimation{}, false
	}
	from := animations[fromID]
	var to res.STRAnimation
	hasTo := toID >= 0 && toID < len(animations)
	if hasTo {
		to = animations[toID]
	}
	delta := float32(keyIndex - float64(from.Frame))
	out := res.STRAnimation{
		SrcAlpha:  from.SrcAlpha,
		DestAlpha: from.DestAlpha,
	}
	if !hasTo || toID != fromID+1 || to.Frame != from.Frame {
		if hasTo && lastSource <= from.Frame {
			return res.STRAnimation{}, false
		}
		return from, true
	}
	out.Angle = from.Angle + to.Angle*delta
	out.AniFrame = strAnimFrame(from, to, delta, len(layer.Textures))
	for i := range out.Color {
		out.Color[i] = from.Color[i] + to.Color[i]*delta
	}
	for i := range out.Pos {
		out.Pos[i] = from.Pos[i] + to.Pos[i]*delta
	}
	for i := range out.UV {
		out.UV[i] = from.UV[i] + to.UV[i]*delta
	}
	for i := range out.XY {
		out.XY[i] = from.XY[i] + to.XY[i]*delta
	}
	return out, true
}

func strAnimFrame(from, to res.STRAnimation, delta float32, texCount int) float32 {
	switch to.AniType {
	case 1:
		return from.AniFrame + to.AniFrame*delta
	case 2:
		return minFloat32(from.AniFrame+to.Delay*delta, float32(texCount-1))
	case 3:
		count := float32(maxInt(texCount, 1))
		return float32(math.Mod(float64(from.AniFrame+to.Delay*delta), float64(count)))
	case 4:
		count := float32(maxInt(texCount, 1))
		value := float32(math.Mod(float64(from.AniFrame-to.Delay*delta), float64(count)))
		if value < 0 {
			value += count
		}
		return value
	default:
		return 0
	}
}

func drawSTRAnimation(screen *render.Frame, projection sceneProjection, texture *render.Image, worldX, worldY, worldZ float64, anim res.STRAnimation, attached bool) {
	right, up, _, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	const pixelRatio = 1.0 / 35.0
	offsetX, offsetY := strAnimationOffset(anim, attached)
	center := modelPoint3{x: worldX, y: worldZ, z: worldY}
	angle := -float64(anim.Angle) * math.Pi / 180
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	vertexPoint := func(ix, iy int) modelPoint3 {
		x := float64(anim.XY[ix])
		y := float64(anim.XY[iy])
		rotX := x*cosA - y*sinA
		rotY := x*sinA + y*cosA
		dx := rotX*pixelRatio + offsetX
		dy := -rotY*pixelRatio + offsetY
		return add3(add3(center, mul3(right, dx)), mul3(up, dy))
	}
	tint := strAnimationTint(anim)
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	vertices := []render.Vertex3D{
		texturedSurfaceVertex3D(vertexPoint(0, 4), texturePoint{u: 0, v: 0}, tint, w, h),
		texturedSurfaceVertex3D(vertexPoint(1, 5), texturePoint{u: 1, v: 0}, tint, w, h),
		texturedSurfaceVertex3D(vertexPoint(3, 7), texturePoint{u: 0, v: 1}, tint, w, h),
		texturedSurfaceVertex3D(vertexPoint(2, 6), texturePoint{u: 1, v: 1}, tint, w, h),
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	options.Blend = strAnimationBlend(anim)
	screen.DrawTriangles3DOwned(vertices, quadIndices012213, texture, options)
}

func strAnimationBlend(anim res.STRAnimation) render.Blend {
	// reference client applies the Direct3D blend factors stored in each STR layer.
	// D3DBLEND_SRCALPHA + D3DBLEND_DESTALPHA is used by bright fog-like
	// effects such as Pneuma; on our opaque world target it matches src-alpha
	// additive blending.
	if anim.SrcAlpha == 5 && (anim.DestAlpha == 2 || anim.DestAlpha == 7) {
		return render.BlendLighter
	}
	if anim.DestAlpha == 2 {
		return render.BlendLighter
	}
	return render.BlendSourceOver
}

func strAnimationOffset(anim res.STRAnimation, attached bool) (float64, float64) {
	const pixelRatio = 1.0 / 35.0
	verticalBase := -0.5
	if attached {
		verticalBase = 0
	}
	return float64(anim.Pos[0]-320) * pixelRatio, -float64(anim.Pos[1]-320)*pixelRatio + verticalBase
}

func strAnimationTint(anim res.STRAnimation) color.RGBA {
	return color.RGBA{
		R: uint8(clampFloat(float64(anim.Color[0]), 0, 1) * 255),
		G: uint8(clampFloat(float64(anim.Color[1]), 0, 1) * 255),
		B: uint8(clampFloat(float64(anim.Color[2]), 0, 1) * 255),
		A: uint8(clampFloat(float64(anim.Color[3]), 0, 1) * 255),
	}
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
