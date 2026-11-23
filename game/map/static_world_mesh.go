package maps

import (
	"image"
	"image/color"
	"math"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	gndRetainedChunkSize = 16
)

type retainedWorldMesh struct {
	mesh *render.WorldMesh
}

type gndRetainedMeshCache struct {
	gnd           *res.GND
	rsw           *res.RSW
	lightmapAtlas gndLightmapAtlas
	chunks        map[gndRetainedChunkKey][]retainedWorldMesh
}

type gndRetainedChunkKey struct {
	x int
	y int
}

type retainedMeshKey struct {
	texture      *render.Image
	lightTexture *render.Image
	options      render.DrawTrianglesOptions
}

type retainedMeshBuilder struct {
	texture      *render.Image
	lightTexture *render.Image
	options      render.DrawTrianglesOptions
	vertices     []render.Vertex3D
	indices      []uint16
	meshes       []retainedWorldMesh
}

func (b *retainedMeshBuilder) addTriangle(a, c, d render.Vertex3D) {
	if b.texture == nil {
		return
	}
	if len(b.vertices)+3 > scene.MaxWorldMeshVertices {
		b.flush()
	}
	base := uint16(len(b.vertices))
	b.vertices = append(b.vertices, a, c, d)
	b.indices = append(b.indices, base, base+1, base+2)
}

func (b *retainedMeshBuilder) addQuad(vertices [4]render.Vertex3D, indices []uint16) {
	if b.texture == nil {
		return
	}
	if len(b.vertices)+len(vertices) > scene.MaxWorldMeshVertices {
		b.flush()
	}
	base := uint16(len(b.vertices))
	b.vertices = append(b.vertices, vertices[:]...)
	for _, index := range indices {
		b.indices = append(b.indices, base+index)
	}
}

func (b *retainedMeshBuilder) flush() {
	if len(b.vertices) == 0 || len(b.indices) == 0 || b.texture == nil {
		b.vertices = b.vertices[:0]
		b.indices = b.indices[:0]
		return
	}
	mesh := render.NewWorldMeshWithLightmap(b.vertices, b.indices, b.texture, b.lightTexture, &b.options)
	b.meshes = append(b.meshes, retainedWorldMesh{mesh: mesh})
	b.vertices = nil
	b.indices = nil
}

func (m *System) DrawGNDMeshes(screen *render.Frame, manager *res.Manager, gnd *res.GND, rsw *res.RSW, projection scene.Projection) {
	if gnd == nil {
		return
	}
	cache := m.gndMeshCache
	if cache == nil || cache.gnd != gnd || cache.rsw != rsw {
		cache = &gndRetainedMeshCache{gnd: gnd, rsw: rsw, lightmapAtlas: buildGNDLightmapAtlas(gnd), chunks: make(map[gndRetainedChunkKey][]retainedWorldMesh)}
		m.Images.Own(cache.lightmapAtlas.image)
		m.gndMeshCache = cache
	}
	width := screen.Bounds().Dx()
	height := screen.Bounds().Dy()
	startX, endX, startY, endY, ok := gndDrawBounds(gnd, projection, width, height)
	if !ok {
		return
	}
	startChunkX := startX / gndRetainedChunkSize
	endChunkX := endX / gndRetainedChunkSize
	startChunkY := startY / gndRetainedChunkSize
	endChunkY := endY / gndRetainedChunkSize
	for chunkY := startChunkY; chunkY <= endChunkY; chunkY++ {
		for chunkX := startChunkX; chunkX <= endChunkX; chunkX++ {
			key := gndRetainedChunkKey{x: chunkX, y: chunkY}
			meshes, ok := cache.chunks[key]
			if !ok {
				x0 := chunkX * gndRetainedChunkSize
				y0 := chunkY * gndRetainedChunkSize
				x1 := scene.MinInt(gnd.Width-1, x0+gndRetainedChunkSize-1)
				y1 := scene.MinInt(gnd.Height-1, y0+gndRetainedChunkSize-1)
				meshes = m.buildGNDMeshChunk(manager, gnd, rsw, cache.lightmapAtlas, x0, x1, y0, y1)
				cache.chunks[key] = meshes
			}
			for _, mesh := range meshes {
				screen.DrawWorldMesh(mesh.mesh)
			}
		}
	}
}

func (m *System) buildGNDMeshChunk(manager *res.Manager, gnd *res.GND, rsw *res.RSW, lightmapAtlas gndLightmapAtlas, startX, endX, startY, endY int) []retainedWorldMesh {
	if gnd == nil {
		return nil
	}
	startX = scene.MaxInt(0, startX)
	startY = scene.MaxInt(0, startY)
	endX = scene.MinInt(gnd.Width-1, endX)
	endY = scene.MinInt(gnd.Height-1, endY)
	if startX > endX || startY > endY {
		return nil
	}
	if m.WhitePixel == nil {
		m.WhitePixel = render.NewImage(1, 1)
		m.WhitePixel.Fill(color.White)
	}
	lighting := m.Lighting(rsw)
	cellLights := m.gndCellLights(gnd, rsw)
	builders := make(map[retainedMeshKey]*retainedMeshBuilder)
	builderFor := func(texture, lightTexture *render.Image, options *render.DrawTrianglesOptions) *retainedMeshBuilder {
		if texture == nil || options == nil {
			return nil
		}
		key := retainedMeshKey{texture: texture, lightTexture: lightTexture, options: *options}
		builder := builders[key]
		if builder == nil {
			builder = &retainedMeshBuilder{texture: texture, lightTexture: lightTexture, options: *options}
			builders[key] = builder
		}
		return builder
	}
	addTextured := func(texture *render.Image, verts [4]scene.Point3, uvs [4]scene.UV, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
		builder := builderFor(texture, nil, options)
		if builder == nil {
			return
		}
		bounds := texture.Bounds()
		w, h := float32(bounds.Dx()), float32(bounds.Dy())
		builder.addQuad([4]render.Vertex3D{
			scene.TexturedSurfaceVertex3D(verts[0], uvs[0], tints[0], w, h),
			scene.TexturedSurfaceVertex3D(verts[1], uvs[1], tints[1], w, h),
			scene.TexturedSurfaceVertex3D(verts[2], uvs[2], tints[2], w, h),
			scene.TexturedSurfaceVertex3D(verts[3], uvs[3], tints[3], w, h),
		}, indices)
	}
	addColored := func(verts [4]scene.Point3, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
		builder := builderFor(m.WhitePixel, nil, options)
		if builder == nil {
			return
		}
		builder.addQuad([4]render.Vertex3D{
			scene.ColoredSurfaceVertex3D(verts[0], 0, 0, tints[0]),
			scene.ColoredSurfaceVertex3D(verts[1], 1, 0, tints[1]),
			scene.ColoredSurfaceVertex3D(verts[2], 1, 1, tints[2]),
			scene.ColoredSurfaceVertex3D(verts[3], 0, 1, tints[3]),
		}, indices)
	}
	addLightmapped := func(texture *render.Image, verts [4]scene.Point3, uvs [4]scene.UV, tints [4]color.RGBA, lightmapID int) {
		if texture == nil || lightmapAtlas.image == nil {
			addTextured(texture, verts, uvs, scene.QuadIndices012023, tints, groundTextureDrawOptions())
			return
		}
		textureBounds := texture.Bounds()
		textureWidth := float32(textureBounds.Dx())
		textureHeight := float32(textureBounds.Dy())
		options := groundTextureDrawOptions()
		builder := builderFor(texture, lightmapAtlas.image, options)
		if builder == nil {
			return
		}
		lightUVs := lightmapAtlas.uvs(lightmapID)
		var vertices [4]render.Vertex3D
		for i := range vertices {
			vertices[i] = lightmappedSurfaceVertex3D(verts[i], uvs[i], lightUVs[i], tints[i], textureWidth, textureHeight)
		}
		builder.addQuad(vertices, scene.QuadIndices012023)
	}

	// Side surfaces are lightmapped like top surfaces (C3dGround15::AssignGnd).
	addSideSurface := func(surface res.GNDSurface, vertexOrder [4]int, verts [4]scene.Point3, tints [4]color.RGBA) {
		texture := m.groundTexture(manager, gndTextureName(gnd, surface.TextureID))
		if _, ok := gnd.Lightmap(surface.LightmapID); ok && texture != nil {
			addLightmapped(texture, verts, surfaceUVs(surface, vertexOrder), tints, surface.LightmapID)
			return
		}
		addGNDRetainedSurface(m, manager, gnd, surface, vertexOrder, verts, tints, addTextured, addColored)
	}

	for y := startY; y <= endY; y++ {
		for x := startX; x <= endX; x++ {
			cell, ok := gnd.Cell(x, y)
			if !ok {
				continue
			}
			if cell.Top >= 0 {
				if surface, ok := gnd.Surface(cell.Top); ok {
					vertexOrder := [4]int{0, 1, 3, 2}
					verts := [4]scene.Point3{
						{X: float64(x) * 2, Y: float64(cell.Heights[0]), Z: float64(y) * 2},
						{X: float64(x+1) * 2, Y: float64(cell.Heights[1]), Z: float64(y) * 2},
						{X: float64(x+1) * 2, Y: float64(cell.Heights[3]), Z: float64(y+1) * 2},
						{X: float64(x) * 2, Y: float64(cell.Heights[2]), Z: float64(y+1) * 2},
					}
					uvs := surfaceUVs(surface, vertexOrder)
					tints := gndEnvTints(lighting, gndCellLightAt(cellLights, gnd, x, y).topGoro())
					if texture := m.groundTexture(manager, gndTextureName(gnd, surface.TextureID)); texture != nil {
						if _, ok := gnd.Lightmap(surface.LightmapID); ok {
							addLightmapped(texture, verts, uvs, tints, surface.LightmapID)
						} else {
							addTextured(texture, verts, uvs, scene.QuadIndices012023, tints, groundTextureDrawOptions())
						}
					} else {
						addColored(verts, scene.QuadIndices012023, untexturedGroundColors(gndTextureName(gnd, surface.TextureID), tints), WorldOpaqueTriangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
					}
				}
			}
			if cell.Front >= 0 && y+1 < gnd.Height {
				neighbor, neighborOK := gnd.Cell(x, y+1)
				surface, surfaceOK := gnd.Surface(cell.Front)
				if neighborOK && surfaceOK {
					vertexOrder := [4]int{0, 1, 3, 2}
					verts := [4]scene.Point3{
						{X: float64(x) * 2, Y: float64(cell.Heights[2]), Z: float64(y+1) * 2},
						{X: float64(x+1) * 2, Y: float64(cell.Heights[3]), Z: float64(y+1) * 2},
						{X: float64(x+1) * 2, Y: float64(neighbor.Heights[1]), Z: float64(y+1) * 2},
						{X: float64(x) * 2, Y: float64(neighbor.Heights[0]), Z: float64(y+1) * 2},
					}
					tints := gndEnvTints(lighting, gndCellLightAt(cellLights, gnd, x, y).frontGoro())
					addSideSurface(surface, vertexOrder, verts, tints)
				}
			}
			if cell.Right >= 0 && x+1 < gnd.Width {
				neighbor, neighborOK := gnd.Cell(x+1, y)
				surface, surfaceOK := gnd.Surface(cell.Right)
				if neighborOK && surfaceOK {
					vertexOrder := [4]int{0, 1, 3, 2}
					verts := [4]scene.Point3{
						{X: float64(x+1) * 2, Y: float64(cell.Heights[3]), Z: float64(y+1) * 2},
						{X: float64(x+1) * 2, Y: float64(cell.Heights[1]), Z: float64(y) * 2},
						{X: float64(x+1) * 2, Y: float64(neighbor.Heights[0]), Z: float64(y) * 2},
						{X: float64(x+1) * 2, Y: float64(neighbor.Heights[2]), Z: float64(y+1) * 2},
					}
					tints := gndEnvTints(lighting, gndCellLightAt(cellLights, gnd, x, y).rightGoro())
					addSideSurface(surface, vertexOrder, verts, tints)
				}
			}
		}
	}
	var meshes []retainedWorldMesh
	for _, builder := range builders {
		builder.flush()
		meshes = append(meshes, builder.meshes...)
	}
	return meshes
}

type gndLightmapAtlas struct {
	image  *render.Image
	perRow int
}

func buildGNDLightmapAtlas(gnd *res.GND) gndLightmapAtlas {
	if gnd == nil || len(gnd.Lightmaps) == 0 {
		return gndLightmapAtlas{}
	}
	count := len(gnd.Lightmaps)
	perRow := int(math.Round(math.Sqrt(float64(count))))
	if perRow < 1 {
		perRow = 1
	}
	rows := (count + perRow - 1) / perRow
	width := nextPowerOfTwoInt(perRow * 8)
	height := nextPowerOfTwoInt(rows * 8)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for index, lightmap := range gnd.Lightmaps {
		baseX := (index % perRow) * 8
		baseY := (index / perRow) * 8
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				pixel := img.PixOffset(baseX+x, baseY+y)
				lm := posterizeGNDLightmapColor(lightmap.Color[y][x])
				img.Pix[pixel+0] = lm.R
				img.Pix[pixel+1] = lm.G
				img.Pix[pixel+2] = lm.B
				img.Pix[pixel+3] = lightmap.Alpha[y][x]
			}
		}
	}
	return gndLightmapAtlas{
		image:  render.NewImageFromImage(img),
		perRow: perRow,
	}
}

func (a gndLightmapAtlas) uvs(index int) [4]scene.UV {
	if a.image == nil || a.perRow <= 0 || index < 0 {
		return [4]scene.UV{}
	}
	baseX := float32((index % a.perRow) * 8)
	baseY := float32((index / a.perRow) * 8)
	return [4]scene.UV{
		{U: baseX + 1, V: baseY + 1},
		{U: baseX + 7, V: baseY + 1},
		{U: baseX + 7, V: baseY + 7},
		{U: baseX + 1, V: baseY + 7},
	}
}

func nextPowerOfTwoInt(value int) int {
	if value <= 1 {
		return 1
	}
	out := 1
	for out < value {
		out *= 2
	}
	return out
}

func lightmappedSurfaceVertex3D(point scene.Point3, uv scene.UV, lightUV scene.UV, tint color.RGBA, textureWidth, textureHeight float32) render.Vertex3D {
	vertex := scene.TexturedSurfaceVertex3D(point, uv, tint, textureWidth, textureHeight)
	vertex.LightSrcX = lightUV.U
	vertex.LightSrcY = lightUV.V
	return vertex
}

func addGNDRetainedSurface(m *System, manager *res.Manager, gnd *res.GND, surface res.GNDSurface, vertexOrder [4]int, verts [4]scene.Point3, tints [4]color.RGBA, addTextured func(*render.Image, [4]scene.Point3, [4]scene.UV, []uint16, [4]color.RGBA, *render.DrawTrianglesOptions), addColored func([4]scene.Point3, []uint16, [4]color.RGBA, *render.DrawTrianglesOptions)) {
	uvs := surfaceUVs(surface, vertexOrder)
	textureName := gndTextureName(gnd, surface.TextureID)
	if texture := m.groundTexture(manager, textureName); texture != nil {
		addTextured(texture, verts, uvs, scene.QuadIndices012023, tints, groundTextureDrawOptions())
		return
	}
	addColored(verts, scene.QuadIndices012023, untexturedGroundColors(textureName, tints), WorldOpaqueTriangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
}

// gndEnvTints applies the environment colour at draw time (C3dGround15::RenderNew).
func gndEnvTints(lighting Lighting, colors [4]color.RGBA) [4]color.RGBA {
	for i := range colors {
		colors[i] = lighting.applyEnv(colors[i])
	}
	return colors
}

// untexturedGroundColors shades a placeholder colour for surfaces whose
// texture failed to load.
func untexturedGroundColors(textureName string, tints [4]color.RGBA) [4]color.RGBA {
	base := textureColor(textureName)
	var colors [4]color.RGBA
	for i, tint := range tints {
		colors[i] = color.RGBA{
			R: uint8(uint16(base.R) * uint16(tint.R) / 255),
			G: uint8(uint16(base.G) * uint16(tint.G) / 255),
			B: uint8(uint16(base.B) * uint16(tint.B) / 255),
			A: 255,
		}
	}
	return colors
}

// Use the draw-time loaders, preserving resource lookup and transparency rules.
// Only the current map retains these images; no cross-map texture cache grows.
func (m *System) TextureLoadTasks(ctx client.Context) []func() {
	if ctx.Config.Headless || ctx.Resources == nil || ctx.World == nil {
		return nil
	}
	m.UpscaleTextures = ctx.Config.Render.TextureUpscaling
	if ctx.Runtime != nil {
		m.UpscaleTextures = ctx.Runtime.TextureUpscaling()
	}
	seen := make(map[*render.Image]bool)
	add := func(img *render.Image) {
		if img != nil && !seen[img] {
			seen[img] = true
			m.Images.Uploads = append(m.Images.Uploads, img)
		}
	}
	var tasks []func()
	gnd, rsw := ctx.World.GND, ctx.World.RSW
	if gnd != nil {
		for _, name := range gnd.Textures {
			tasks = append(tasks, func() { add(m.groundTexture(ctx.Resources, name)) })
		}
		tasks = append(tasks, func() {
			atlas := buildGNDLightmapAtlas(gnd)
			add(m.Images.Own(atlas.image))
			m.gndMeshCache = &gndRetainedMeshCache{gnd: gnd, rsw: rsw, lightmapAtlas: atlas, chunks: make(map[gndRetainedChunkKey][]retainedWorldMesh)}
		})
	}
	if rsw != nil {
		models := make(map[*res.RSM]bool)
		for _, placement := range rsw.Models {
			model := ctx.World.RSM[placement.Filename]
			if model == nil || models[model] {
				continue
			}
			models[model] = true
			for _, name := range model.Textures {
				tasks = append(tasks, func() { add(m.groundTexture(ctx.Resources, name)) })
			}
		}
	}
	if water, ok := mapWater(gnd, rsw); ok && gnd != nil {
		for _, cell := range gnd.Cells {
			if cell.Top >= 0 && waterVisibleForCell(cell, water) {
				for frame := 0; frame < 32; frame++ {
					tasks = append(tasks, func() { add(m.waterTexture(ctx.Resources, int(water.Type), frame)) })
				}
				break
			}
		}
	}
	return tasks
}
