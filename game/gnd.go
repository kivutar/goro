package game

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func loadGND(manager *res.Manager, mapName string) (*res.GND, string, error) {
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

func (m *WorldMode) drawGNDWater(screen *render.Frame, manager *res.Manager, gnd *res.GND, rsw *res.RSW, projection sceneProjection, now time.Time, fog sceneFog) {
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
			if waterDraw, ok := newGNDWaterDraw(x, y, cell, water, waterFrame, waterTint, waterOffset); ok {
				if !textureLoaded {
					texture = m.waterTexture(manager, int(water.Type), waterFrame)
					textureLoaded = true
				}
				m.drawWaterSurface(screen, texture, waterDraw, projection, fog)
			}
		}
	}
}

func gndDrawBounds(gnd *res.GND, projection sceneProjection, screenWidth, screenHeight int) (int, int, int, int, bool) {
	return gndDrawBoundsWithMargin(gnd, projection, screenWidth, screenHeight, 24)
}

func gndDrawBoundsWithMargin(gnd *res.GND, projection sceneProjection, screenWidth, screenHeight, margin int) (int, int, int, int, bool) {
	if gnd == nil || gnd.Width <= 0 || gnd.Height <= 0 {
		return 0, 0, 0, 0, false
	}
	if margin < 0 {
		margin = 0
	}

	centerX := gndTileFromWorld(projection.playerX)
	centerY := gndTileFromWorld(projection.playerY)
	if minWorldX, maxWorldX, minWorldY, maxWorldY, ok := cameraGroundFootprint(projection, screenWidth, screenHeight); ok {
		startX := minInt(gndTileFromWorld(minWorldX), centerX) - margin
		endX := maxInt(gndTileFromWorld(maxWorldX), centerX) + margin
		startY := minInt(gndTileFromWorld(minWorldY), centerY) - margin
		endY := maxInt(gndTileFromWorld(maxWorldY), centerY) + margin
		return clampGNDRange(gnd, startX, endX, startY, endY)
	}
	const fallbackRadius = 96
	return clampGNDRange(gnd, centerX-fallbackRadius, centerX+fallbackRadius, centerY-fallbackRadius, centerY+fallbackRadius)
}

func gndTileFromWorld(coord float64) int {
	return int(math.Floor(coord * 0.5))
}

func clampGNDRange(gnd *res.GND, startX, endX, startY, endY int) (int, int, int, int, bool) {
	startX = maxInt(0, startX)
	endX = minInt(gnd.Width-1, endX)
	startY = maxInt(0, startY)
	endY = minInt(gnd.Height-1, endY)
	return startX, endX, startY, endY, startX <= endX && startY <= endY
}

func worldGNDHeightAt(world *worldstate.World, x, y float64) (float64, bool) {
	if world == nil || world.GND == nil {
		return 0, false
	}
	gridX := (x + 0.5) * 0.5
	gridY := (y + 0.5) * 0.5
	cellX := int(math.Floor(gridX))
	cellY := int(math.Floor(gridY))
	cell, ok := world.GND.Cell(cellX, cellY)
	if !ok {
		return 0, false
	}
	return bilinearHeight(cell.Heights, gridX-float64(cellX), gridY-float64(cellY)), true
}

func gndShadowMapPoint(x, y float64) (int, int) {
	x += 0.5
	y += 0.5
	shadowX := int(math.Floor(x/2)) * 8
	shadowY := int(math.Floor(y/2)) * 8
	localX := 0
	if int(x)&1 != 0 {
		localX = 4
	}
	localY := 0
	if int(y)&1 != 0 {
		localY = 4
	}
	localX += int(math.Floor((x - math.Floor(x)) * 4))
	localY += int(math.Floor((y - math.Floor(y)) * 4))
	shadowX += minInt(localX, 6)
	shadowY += minInt(localY, 6)
	return shadowX, shadowY
}

func gndShadowMapAlpha(gnd *res.GND, shadowX, shadowY int) uint8 {
	if gnd == nil || shadowX < 0 || shadowY < 0 || shadowX >= gnd.Width*8 || shadowY >= gnd.Height*8 {
		return 255
	}
	cellX := shadowX / 8
	cellY := shadowY / 8
	localX := shadowX % 8
	localY := shadowY % 8
	cell, ok := gnd.Cell(cellX, cellY)
	if !ok || cell.Top < 0 {
		return 255
	}
	surface, ok := gnd.Surface(cell.Top)
	if !ok {
		return 255
	}
	lightmap, ok := gnd.Lightmap(surface.LightmapID)
	if !ok {
		return 255
	}
	return lightmap.Alpha[localY][localX]
}

func cameraGroundFootprint(projection sceneProjection, screenWidth, screenHeight int) (float64, float64, float64, float64, bool) {
	aspect := 1.0
	if screenHeight > 0 {
		aspect = float64(screenWidth) / float64(screenHeight)
	}

	eye, forward, right, up := projection.cameraBasis()
	tanHalfFOV := math.Tan(degreesToRadians(sceneCameraFOV()) * 0.5)

	samples := [][2]float64{
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
		{0, 0}, {-1, 0}, {1, 0}, {0, -1}, {0, 1},
	}
	minX, maxX := 0.0, 0.0
	minY, maxY := 0.0, 0.0
	found := false
	for _, sample := range samples {
		dir := normalize3(add3(add3(forward, mul3(right, sample[0]*tanHalfFOV*aspect)), mul3(up, sample[1]*tanHalfFOV)))
		if math.Abs(dir.y) < 0.000001 {
			continue
		}
		t := (projection.playerZ - eye.y) / dir.y
		if t <= 0 || !isFinite(t) {
			continue
		}
		hit := add3(eye, mul3(dir, t))
		if !isFinite(hit.x) || !isFinite(hit.z) {
			continue
		}
		if !found {
			minX, maxX = hit.x, hit.x
			minY, maxY = hit.z, hit.z
			found = true
			continue
		}
		minX = math.Min(minX, hit.x)
		maxX = math.Max(maxX, hit.x)
		minY = math.Min(minY, hit.z)
		maxY = math.Max(maxY, hit.z)
	}
	return minX, maxX, minY, maxY, found
}

type gndSurfaceDraw struct {
	verts      [4]modelPoint3
	uvs        [4]texturePoint
	indices    []uint16
	heights    [4]float32
	water      bool
	waterType  int
	waterFrame int
	tint       color.RGBA
}

func (m *WorldMode) smoothGNDTopNormals(gnd *res.GND) [][4]modelPoint3 {
	if gnd == nil {
		return nil
	}
	if m.gndNormalSource == gnd && len(m.gndTopNormals) == gnd.Width*gnd.Height {
		return m.gndTopNormals
	}
	m.gndNormalSource = gnd
	m.gndTopNormals = buildSmoothGNDTopNormals(gnd)
	return m.gndTopNormals
}

func buildSmoothGNDTopNormals(gnd *res.GND) [][4]modelPoint3 {
	if gnd == nil || gnd.Width <= 0 || gnd.Height <= 0 {
		return nil
	}
	count := gnd.Width * gnd.Height
	cellNormals := make([]modelPoint3, count)
	for y := 0; y < gnd.Height; y++ {
		for x := 0; x < gnd.Width; x++ {
			cell, ok := gnd.Cell(x, y)
			if !ok || cell.Top < 0 {
				continue
			}
			verts := [4]modelPoint3{
				{x: float64(x) * 2, y: float64(cell.Heights[0]), z: float64(y) * 2},
				{x: float64(x+1) * 2, y: float64(cell.Heights[1]), z: float64(y) * 2},
				{x: float64(x+1) * 2, y: float64(cell.Heights[3]), z: float64(y+1) * 2},
				{x: float64(x) * 2, y: float64(cell.Heights[2]), z: float64(y+1) * 2},
			}
			cellNormals[x+y*gnd.Width] = quadNormal(verts)
		}
	}

	normals := make([][4]modelPoint3, count)
	for y := 0; y < gnd.Height; y++ {
		for x := 0; x < gnd.Width; x++ {
			cellNormal := gndCellNormalAt(cellNormals, gnd.Width, gnd.Height, x, y)
			if cellNormal == (modelPoint3{}) {
				cellNormal = modelPoint3{y: -1}
			}
			normals[x+y*gnd.Width] = [4]modelPoint3{
				smoothGNDNormalAt(cellNormals, gnd.Width, gnd.Height, cellNormal, [][2]int{{x, y}, {x - 1, y}, {x - 1, y - 1}, {x, y - 1}}),
				smoothGNDNormalAt(cellNormals, gnd.Width, gnd.Height, cellNormal, [][2]int{{x, y}, {x + 1, y}, {x + 1, y - 1}, {x, y - 1}}),
				smoothGNDNormalAt(cellNormals, gnd.Width, gnd.Height, cellNormal, [][2]int{{x, y}, {x + 1, y}, {x + 1, y + 1}, {x, y + 1}}),
				smoothGNDNormalAt(cellNormals, gnd.Width, gnd.Height, cellNormal, [][2]int{{x, y}, {x - 1, y}, {x - 1, y + 1}, {x, y + 1}}),
			}
		}
	}
	return normals
}

func gndCellNormalAt(normals []modelPoint3, width, height, x, y int) modelPoint3 {
	if x < 0 || y < 0 || x >= width || y >= height {
		return modelPoint3{}
	}
	return normals[x+y*width]
}

func smoothGNDNormalAt(normals []modelPoint3, width, height int, fallback modelPoint3, samples [][2]int) modelPoint3 {
	var sum modelPoint3
	for _, sample := range samples {
		normal := gndCellNormalAt(normals, width, height, sample[0], sample[1])
		if normal == (modelPoint3{}) {
			continue
		}
		sum = add3(sum, normal)
	}
	if sum == (modelPoint3{}) {
		return fallback
	}
	return normalize3(sum)
}

func gndTopNormalsAt(normals [][4]modelPoint3, gnd *res.GND, x, y int) [4]modelPoint3 {
	if gnd == nil || x < 0 || y < 0 || x >= gnd.Width || y >= gnd.Height {
		return [4]modelPoint3{}
	}
	index := x + y*gnd.Width
	if index < 0 || index >= len(normals) {
		return [4]modelPoint3{}
	}
	return normals[index]
}

func topGNDSurfaceBaseTints(gnd *res.GND, x, y int, fallback color.RGBA) [4]color.RGBA {
	return [4]color.RGBA{
		gndTopTileBaseTintAt(gnd, x, y, fallback),
		gndTopTileBaseTintAt(gnd, x+1, y, fallback),
		gndTopTileBaseTintAt(gnd, x+1, y+1, fallback),
		gndTopTileBaseTintAt(gnd, x, y+1, fallback),
	}
}

func gndTopTileBaseTintAt(gnd *res.GND, x, y int, fallback color.RGBA) color.RGBA {
	if gnd != nil {
		if cell, ok := gnd.Cell(x, y); ok && cell.Top >= 0 {
			if surface, ok := gnd.Surface(cell.Top); ok {
				return gndSurfaceBaseTint(surface.Color)
			}
		}
	}
	return gndSurfaceBaseTint(fallback)
}

func uniformGNDSurfaceBaseTints(surfaceColor color.RGBA) [4]color.RGBA {
	tint := gndSurfaceBaseTint(surfaceColor)
	return [4]color.RGBA{tint, tint, tint, tint}
}

func gndSurfaceBaseTint(surfaceColor color.RGBA) color.RGBA {
	if surfaceColor.A == 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return color.RGBA{R: surfaceColor.R, G: surfaceColor.G, B: surfaceColor.B, A: 255}
}

func uniformGNDNormals(normal modelPoint3) [4]modelPoint3 {
	normal = normalize3(normal)
	return [4]modelPoint3{normal, normal, normal, normal}
}

func newGNDWaterDraw(x, y int, cell res.GNDCell, water res.RSWWater, waterFrame int, tint color.RGBA, waterOffset float64) (gndSurfaceDraw, bool) {
	if !waterVisibleForCell(cell, water) {
		return gndSurfaceDraw{}, false
	}
	heights := waterHeightsForCell(water, x, y, waterOffset)
	verts := [4]modelPoint3{
		{x: float64(x) * 2, y: float64(heights[0]), z: float64(y) * 2},
		{x: float64(x+1) * 2, y: float64(heights[1]), z: float64(y) * 2},
		{x: float64(x+1) * 2, y: float64(heights[3]), z: float64(y+1) * 2},
		{x: float64(x) * 2, y: float64(heights[2]), z: float64(y+1) * 2},
	}
	return gndSurfaceDraw{
		verts:      verts,
		uvs:        waterUVs(x, y),
		indices:    quadIndices012023,
		heights:    heights,
		water:      true,
		waterType:  int(water.Type),
		waterFrame: waterFrame,
		tint:       tint,
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

func (m *WorldMode) drawWaterSurface(screen *render.Frame, texture *render.Image, draw gndSurfaceDraw, projection sceneProjection, fog sceneFog) {
	tints := fog.mixVertexTints(projection, draw.verts, [4]color.RGBA{draw.tint, draw.tint, draw.tint, draw.tint})
	if texture == nil {
		drawColoredSurfaceTints3DAlpha(screen, m.whitePixel, draw.verts, draw.indices, tints)
		return
	}
	drawTexturedSurface3DAlpha(screen, texture, draw.verts, draw.uvs, draw.indices, tints)
}

func (m *WorldMode) waterTexture(manager *res.Manager, waterType, frame int) *render.Image {
	frame = ((frame % 32) + 32) % 32
	key := fmt.Sprintf("__water_%d_%02d", waterType, frame)
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.WaterTextureCandidates(waterType, frame))
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil
	}
	texture := m.ownMapImage(render.NewImageFromImage(img))
	m.textures[key] = texture
	return texture
}

func (m *WorldMode) groundTexture(manager *res.Manager, name string) *render.Image {
	if name == "" {
		return nil
	}
	if texture, ok := m.textures[name]; ok {
		return texture
	}
	if _, ok := m.textureMiss[name]; ok {
		return nil
	}

	img, _, err := res.LoadImage(manager, res.GroundTextureCandidates(name))
	if err != nil {
		m.textureMiss[name] = struct{}{}
		return nil
	}
	texture := m.ownMapImage(render.NewImageFromImage(img))
	m.textures[name] = texture
	return texture
}

func gndTextureName(gnd *res.GND, textureID int) string {
	if gnd == nil || textureID < 0 || textureID >= len(gnd.Textures) {
		return ""
	}
	return gnd.Textures[textureID]
}

func surfaceUVs(surface res.GNDSurface, order [4]int) [4]texturePoint {
	return [4]texturePoint{
		{u: surface.U[order[0]], v: surface.V[order[0]]},
		{u: surface.U[order[1]], v: surface.V[order[1]]},
		{u: surface.U[order[2]], v: surface.V[order[2]]},
		{u: surface.U[order[3]], v: surface.V[order[3]]},
	}
}

func waterUVs(x, y int) [4]texturePoint {
	const scale = 0.25
	baseU := float32(x&3) * scale
	baseV := float32(y&3) * scale
	return [4]texturePoint{
		{u: baseU, v: baseV},
		{u: baseU + scale, v: baseV},
		{u: baseU + scale, v: baseV + scale},
		{u: baseU, v: baseV + scale},
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
	return float32(math.Sin(degreesToRadians(degrees)))
}

func waterTint(water res.RSWWater, rsw *res.RSW) color.RGBA {
	alpha := uint8(204)
	if water.Type == 4 || water.Type == 6 {
		alpha = 255
	}
	if rsw != nil && water.Type == 4 {
		return color.RGBA{
			R: clampColor(float64(rsw.Light.Ambient[0]) * 255),
			G: clampColor(float64(rsw.Light.Ambient[1]) * 255),
			B: clampColor(float64(rsw.Light.Ambient[2]) * 255),
			A: alpha,
		}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: alpha}
}

func surfaceTint(base color.RGBA, height float32, normal modelPoint3, lighting sceneLighting) color.RGBA {
	scale := lighting.groundScale(normal)
	heightShade := groundHeightShadeAt(height)
	return color.RGBA{
		R: clampColor(float64(base.R) * scale.x * heightShade),
		G: clampColor(float64(base.G) * scale.y * heightShade),
		B: clampColor(float64(base.B) * scale.z * heightShade),
		A: 255,
	}
}

func surfaceVertexTints(baseTints [4]color.RGBA, heights [4]float32, normals [4]modelPoint3, lighting sceneLighting) [4]color.RGBA {
	return [4]color.RGBA{
		surfaceTint(baseTints[0], heights[0], normals[0], lighting),
		surfaceTint(baseTints[1], heights[1], normals[1], lighting),
		surfaceTint(baseTints[2], heights[2], normals[2], lighting),
		surfaceTint(baseTints[3], heights[3], normals[3], lighting),
	}
}

func scaleSurfaceVertexTints(baseTints [4]color.RGBA, lightScales [4]modelPoint3) [4]color.RGBA {
	var tints [4]color.RGBA
	for i := range tints {
		lightScale := lightScales[i]
		base := baseTints[i]
		tints[i] = color.RGBA{
			R: clampColor(float64(base.R) * lightScale.x),
			G: clampColor(float64(base.G) * lightScale.y),
			B: clampColor(float64(base.B) * lightScale.z),
			A: base.A,
		}
	}
	return tints
}

func vertexLightScales(lighting sceneLighting, normals [4]modelPoint3) [4]modelPoint3 {
	return [4]modelPoint3{
		lighting.groundScale(normals[0]),
		lighting.groundScale(normals[1]),
		lighting.groundScale(normals[2]),
		lighting.groundScale(normals[3]),
	}
}

func groundSurfaceVertexColors(textureName string, surfaceColor color.RGBA, heights [4]float32, normals [4]modelPoint3, lighting sceneLighting) [4]color.RGBA {
	base := textureColor(textureName)
	if surfaceColor.A != 0 && !(surfaceColor.R == 255 && surfaceColor.G == 255 && surfaceColor.B == 255) {
		base.R = uint8((uint16(base.R)*2 + uint16(surfaceColor.R)) / 3)
		base.G = uint8((uint16(base.G)*2 + uint16(surfaceColor.G)) / 3)
		base.B = uint8((uint16(base.B)*2 + uint16(surfaceColor.B)) / 3)
	}
	var colors [4]color.RGBA
	for i := range colors {
		scale := lighting.groundScale(normals[i])
		heightShade := groundHeightShadeAt(heights[i])
		colors[i] = color.RGBA{
			R: clampColor(float64(base.R) * scale.x * heightShade),
			G: clampColor(float64(base.G) * scale.y * heightShade),
			B: clampColor(float64(base.B) * scale.z * heightShade),
			A: 255,
		}
	}
	return colors
}

func groundHeightShadeAt(height float32) float64 {
	return 0.88 + math.Sin(float64(height)*0.08)*0.06
}

func posterizeGNDLightmapColor(c color.RGBA) color.RGBA {
	return color.RGBA{
		R: posterizeGNDLightmapChannel(c.R),
		G: posterizeGNDLightmapChannel(c.G),
		B: posterizeGNDLightmapChannel(c.B),
		A: c.A,
	}
}

func posterizeGNDLightmapChannel(v uint8) uint8 {
	return (v / 16) * 16
}
