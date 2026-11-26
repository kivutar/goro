package maps

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func LoadGND(manager *res.Manager, mapName string) (*res.GND, string, error) {
	base := strings.TrimSuffix(strings.TrimSuffix(mapName, ".gat"), ".rsw")
	candidates := []string{
		"data\\" + base + ".gnd",
		"data/" + base + ".gnd",
		base + ".gnd",
	}
	data, source, err := manager.ReadFileCandidates(candidates)
	if err != nil {
		return nil, source, fmt.Errorf("cannot load GND for map %s: %w", mapName, err)
	}
	gnd, err := res.ParseGND(data)
	return gnd, source, err
}

func (m *System) DrawGNDWater(screen *render.Frame, manager *res.Manager, gnd *res.GND, rsw *res.RSW, projection scene.Projection, now time.Time, fog scene.SceneFog) {
	water, ok := mapWater(gnd, rsw)
	if !ok {
		return
	}
	width := screen.Bounds().Dx()
	height := screen.Bounds().Dy()
	startX, endX, startY, endY, ok := gndDrawBoundsWithMargin(gnd, projection, width, height, 4)
	if !ok {
		return
	}
	waterFrame := waterFrameForTime(water, now)
	waterTint := waterTint(water, rsw)
	waterOffset := waterOffsetForTime(water, now)
	var texture *render.Image
	textureLoaded := false
	for y := startY; y <= endY; y++ {
		for x := startX; x <= endX; x++ {
			cell, ok := gnd.Cell(x, y)
			if !ok || cell.Top < 0 {
				continue
			}
			if waterDraw, ok := newGNDWaterDraw(x, y, cell, water, waterTint, waterOffset); ok {
				if !textureLoaded {
					texture = m.waterTexture(manager, int(water.Type), waterFrame)
					textureLoaded = true
				}
				m.drawWaterSurface(screen, texture, waterDraw, projection, fog)
			}
		}
	}
}

func gndDrawBounds(gnd *res.GND, projection scene.Projection, screenWidth, screenHeight int) (int, int, int, int, bool) {
	return gndDrawBoundsWithMargin(gnd, projection, screenWidth, screenHeight, 24)
}

func gndDrawBoundsWithMargin(gnd *res.GND, projection scene.Projection, screenWidth, screenHeight, margin int) (int, int, int, int, bool) {
	if gnd == nil || gnd.Width <= 0 || gnd.Height <= 0 {
		return 0, 0, 0, 0, false
	}
	if margin < 0 {
		margin = 0
	}

	centerX := gndTileFromWorld(projection.PlayerX)
	centerY := gndTileFromWorld(projection.PlayerY)
	if minWorldX, maxWorldX, minWorldY, maxWorldY, ok := cameraGroundFootprint(projection, screenWidth, screenHeight); ok {
		startX := scene.MinInt(gndTileFromWorld(minWorldX), centerX) - margin
		endX := scene.MaxInt(gndTileFromWorld(maxWorldX), centerX) + margin
		startY := scene.MinInt(gndTileFromWorld(minWorldY), centerY) - margin
		endY := scene.MaxInt(gndTileFromWorld(maxWorldY), centerY) + margin
		return clampGNDRange(gnd, startX, endX, startY, endY)
	}
	const fallbackRadius = 96
	return clampGNDRange(gnd, centerX-fallbackRadius, centerX+fallbackRadius, centerY-fallbackRadius, centerY+fallbackRadius)
}

func gndTileFromWorld(coord float64) int {
	return int(math.Floor(coord * 0.5))
}

func clampGNDRange(gnd *res.GND, startX, endX, startY, endY int) (int, int, int, int, bool) {
	startX = scene.MaxInt(0, startX)
	endX = scene.MinInt(gnd.Width-1, endX)
	startY = scene.MaxInt(0, startY)
	endY = scene.MinInt(gnd.Height-1, endY)
	return startX, endX, startY, endY, startX <= endX && startY <= endY
}

func cameraGroundFootprint(projection scene.Projection, screenWidth, screenHeight int) (float64, float64, float64, float64, bool) {
	aspect := 1.0
	if screenHeight > 0 {
		aspect = float64(screenWidth) / float64(screenHeight)
	}

	eye, forward, right, up := projection.CameraBasis()
	tanHalfFOV := math.Tan(scene.DegreesToRadians(scene.SceneCameraFOV()) * 0.5)

	samples := [][2]float64{
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
		{0, 0}, {-1, 0}, {1, 0}, {0, -1}, {0, 1},
	}
	minX, maxX := 0.0, 0.0
	minY, maxY := 0.0, 0.0
	found := false
	for _, sample := range samples {
		dir := scene.Normalize3(scene.Add3(scene.Add3(forward, scene.Mul3(right, sample[0]*tanHalfFOV*aspect)), scene.Mul3(up, sample[1]*tanHalfFOV)))
		if math.Abs(dir.Y) < 0.000001 {
			continue
		}
		t := (projection.PlayerZ - eye.Y) / dir.Y
		if t <= 0 || !scene.IsFinite(t) {
			continue
		}
		hit := scene.Add3(eye, scene.Mul3(dir, t))
		if !scene.IsFinite(hit.X) || !scene.IsFinite(hit.Z) {
			continue
		}
		if !found {
			minX, maxX = hit.X, hit.X
			minY, maxY = hit.Z, hit.Z
			found = true
			continue
		}
		minX = math.Min(minX, hit.X)
		maxX = math.Max(maxX, hit.X)
		minY = math.Min(minY, hit.Z)
		maxY = math.Max(maxY, hit.Z)
	}
	return minX, maxX, minY, maxY, found
}

type gndSurfaceDraw struct {
	verts   [4]scene.Point3
	uvs     [4]scene.UV
	indices []uint16
	tint    color.RGBA
}

func newGNDWaterDraw(x, y int, cell res.GNDCell, water res.RSWWater, tint color.RGBA, waterOffset float64) (gndSurfaceDraw, bool) {
	if !waterVisibleForCell(cell, water) {
		return gndSurfaceDraw{}, false
	}
	heights := waterHeightsForCell(water, x, y, waterOffset)
	verts := [4]scene.Point3{
		{X: float64(x) * 2, Y: float64(heights[0]), Z: float64(y) * 2},
		{X: float64(x+1) * 2, Y: float64(heights[1]), Z: float64(y) * 2},
		{X: float64(x+1) * 2, Y: float64(heights[3]), Z: float64(y+1) * 2},
		{X: float64(x) * 2, Y: float64(heights[2]), Z: float64(y+1) * 2},
	}
	return gndSurfaceDraw{
		verts:   verts,
		uvs:     waterUVs(x, y),
		indices: scene.QuadIndices012023,
		tint:    tint,
	}, true
}

func mapWater(gnd *res.GND, rsw *res.RSW) (res.RSWWater, bool) {
	if gnd != nil && gnd.Water.Present {
		return res.RSWWater{
			Level:      gnd.Water.Level,
			Type:       gnd.Water.Type,
			WaveHeight: gnd.Water.WaveHeight,
			WaveSpeed:  gnd.Water.WaveSpeed,
			WavePitch:  gnd.Water.WavePitch,
			AnimSpeed:  gnd.Water.AnimSpeed,
		}, true
	}
	if rsw != nil {
		return rsw.Water, true
	}
	return res.RSWWater{}, false
}

func (m *System) drawWaterSurface(screen *render.Frame, texture *render.Image, draw gndSurfaceDraw, projection scene.Projection, fog scene.SceneFog) {
	tints := fog.MixVertexTints(projection, draw.verts, [4]color.RGBA{draw.tint, draw.tint, draw.tint, draw.tint})
	if texture == nil {
		scene.DrawColoredSurfaceTints3DAlpha(screen, m.WhitePixel, draw.verts, draw.indices, tints)
		return
	}
	scene.DrawTexturedSurface3DAlpha(screen, texture, draw.verts, draw.uvs, draw.indices, tints)
}

func (m *System) waterTexture(manager *res.Manager, waterType, frame int) *render.Image {
	frame = ((frame % 32) + 32) % 32
	key := fmt.Sprintf("__water_%d_%02d", waterType, frame)
	if texture, ok := m.Textures[key]; ok {
		return texture
	}
	if _, ok := m.TextureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.WaterTextureCandidates(waterType, frame))
	if err != nil {
		m.TextureMiss[key] = struct{}{}
		return nil
	}
	texture := m.Images.Own(render.NewImageFromImage(img))
	m.Textures[key] = texture
	return texture
}

func (m *System) groundTexture(manager *res.Manager, name string) *render.Image {
	if name == "" {
		return nil
	}
	if texture, ok := m.Textures[name]; ok {
		return texture
	}
	if _, ok := m.TextureMiss[name]; ok {
		return nil
	}

	img, _, err := res.LoadImage(manager, res.GroundTextureCandidates(name))
	if err != nil {
		m.TextureMiss[name] = struct{}{}
		return nil
	}
	if m.UpscaleTextures {
		img = render.UpscaleTexture2x(img)
	}
	texture := m.Images.Own(render.NewImageWithMipmaps(img))
	m.Textures[name] = texture
	return texture
}

func gndTextureName(gnd *res.GND, textureID int) string {
	if gnd == nil || textureID < 0 || textureID >= len(gnd.Textures) {
		return ""
	}
	return gnd.Textures[textureID]
}

func surfaceUVs(surface res.GNDSurface, order [4]int) [4]scene.UV {
	return [4]scene.UV{
		{U: surface.U[order[0]], V: surface.V[order[0]]},
		{U: surface.U[order[1]], V: surface.V[order[1]]},
		{U: surface.U[order[2]], V: surface.V[order[2]]},
		{U: surface.U[order[3]], V: surface.V[order[3]]},
	}
}

func waterUVs(x, y int) [4]scene.UV {
	const scale = 0.25
	baseU := float32(x&3) * scale
	baseV := float32(y&3) * scale
	return [4]scene.UV{
		{U: baseU, V: baseV},
		{U: baseU + scale, V: baseV},
		{U: baseU + scale, V: baseV + scale},
		{U: baseU, V: baseV + scale},
	}
}

func waterHeightsForCell(water res.RSWWater, x, y int, offset float64) [4]float32 {
	level := water.Level
	if water.WaveHeight == 0 {
		return [4]float32{level, level, level, level}
	}
	pitch := float64(water.WavePitch)
	diagonal := float64(x + y)
	h1 := waterSin(offset+pitch*diagonal)*water.WaveHeight + level
	h0 := waterSin(offset+pitch*(diagonal-1))*water.WaveHeight + level
	h3 := waterSin(offset+pitch*(diagonal+1))*water.WaveHeight + level
	return [4]float32{h0, h1, h1, h3}
}

func waterVisibleForCell(cell res.GNDCell, water res.RSWWater) bool {
	threshold := water.Level + water.WaveHeight
	return cell.Heights[0] < threshold ||
		cell.Heights[1] < threshold ||
		cell.Heights[2] < threshold ||
		cell.Heights[3] < threshold
}

func waterFrameForTime(water res.RSWWater, now time.Time) int {
	animSpeed := int(water.AnimSpeed)
	if animSpeed <= 0 {
		animSpeed = 1
	}
	frame := int(now.UnixMilli()*60/1000) / animSpeed
	return frame % 32
}

func waterOffsetForTime(water res.RSWWater, now time.Time) float64 {
	offset := math.Mod(float64(now.UnixMilli()*60/1000)*float64(water.WaveSpeed), 360)
	if offset > 180 {
		offset -= 360
	}
	return offset
}

func waterSin(degrees float64) float32 {
	return float32(math.Sin(scene.DegreesToRadians(degrees)))
}

func waterTint(water res.RSWWater, rsw *res.RSW) color.RGBA {
	alpha := uint8(204)
	if water.Type == 4 || water.Type == 6 {
		alpha = 255
	}
	if rsw != nil && water.Type == 4 {
		return color.RGBA{
			R: scene.ClampColor(float64(rsw.Light.Ambient[0]) * 255),
			G: scene.ClampColor(float64(rsw.Light.Ambient[1]) * 255),
			B: scene.ClampColor(float64(rsw.Light.Ambient[2]) * 255),
			A: alpha,
		}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: alpha}
}

// posterizeGNDLightmapColor reproduces the classic lightmap texture
// (CLightmapMgr::Create): each channel, shadow intensity included, keeps its
// top four bits in an A4R4G4B4 surface (CTexture), which Direct3D expands
// back to eight bits as n * 17.
func posterizeGNDLightmapColor(c color.RGBA) color.RGBA {
	return color.RGBA{
		R: posterizeGNDLightmapChannel(c.R),
		G: posterizeGNDLightmapChannel(c.G),
		B: posterizeGNDLightmapChannel(c.B),
		A: posterizeGNDLightmapChannel(c.A),
	}
}

func posterizeGNDLightmapChannel(v uint8) uint8 {
	return (v >> 4) * 17
}
