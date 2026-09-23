package game

import (
	"image"
	"image/color"
	"math"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	retainedWorldMeshMaxVertices = 65535
	gndRetainedChunkSize         = 16
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
	if len(b.vertices)+3 > retainedWorldMeshMaxVertices {
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
	if len(b.vertices)+len(vertices) > retainedWorldMeshMaxVertices {
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

func (m *WorldMode) drawGNDMeshes(screen *render.Frame, manager *res.Manager, gnd *res.GND, rsw *res.RSW, projection sceneProjection) {
	if gnd == nil {
		return
	}
	cache := m.gndMeshCache
	if cache == nil || cache.gnd != gnd || cache.rsw != rsw {
		cache = &gndRetainedMeshCache{gnd: gnd, rsw: rsw, lightmapAtlas: buildGNDLightmapAtlas(gnd), chunks: make(map[gndRetainedChunkKey][]retainedWorldMesh)}
		m.ownMapImage(cache.lightmapAtlas.image)
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
				x1 := minInt(gnd.Width-1, x0+gndRetainedChunkSize-1)
				y1 := minInt(gnd.Height-1, y0+gndRetainedChunkSize-1)
				meshes = m.buildGNDMeshChunk(manager, gnd, rsw, cache.lightmapAtlas, x0, x1, y0, y1)
				cache.chunks[key] = meshes
			}
			for _, mesh := range meshes {
				screen.DrawWorldMesh(mesh.mesh)
			}
		}
	}
}

func (m *WorldMode) buildGNDMeshChunk(manager *res.Manager, gnd *res.GND, rsw *res.RSW, lightmapAtlas gndLightmapAtlas, startX, endX, startY, endY int) []retainedWorldMesh {
	if gnd == nil {
		return nil
	}
	startX = maxInt(0, startX)
	startY = maxInt(0, startY)
	endX = minInt(gnd.Width-1, endX)
	endY = minInt(gnd.Height-1, endY)
	if startX > endX || startY > endY {
		return nil
	}
	if m.whitePixel == nil {
		m.whitePixel = render.NewImage(1, 1)
		m.whitePixel.Fill(color.White)
	}
	lighting := m.sceneLighting(rsw)
	topNormals := m.smoothGNDTopNormals(gnd)
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
	addTextured := func(texture *render.Image, verts [4]modelPoint3, uvs [4]texturePoint, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
		builder := builderFor(texture, nil, options)
		if builder == nil {
			return
		}
		bounds := texture.Bounds()
		w, h := float32(bounds.Dx()), float32(bounds.Dy())
		builder.addQuad([4]render.Vertex3D{
			texturedSurfaceVertex3D(verts[0], uvs[0], tints[0], w, h),
			texturedSurfaceVertex3D(verts[1], uvs[1], tints[1], w, h),
			texturedSurfaceVertex3D(verts[2], uvs[2], tints[2], w, h),
			texturedSurfaceVertex3D(verts[3], uvs[3], tints[3], w, h),
		}, indices)
	}
	addColored := func(verts [4]modelPoint3, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
		builder := builderFor(m.whitePixel, nil, options)
		if builder == nil {
			return
		}
		builder.addQuad([4]render.Vertex3D{
			coloredSurfaceVertex3D(verts[0], 0, 0, tints[0]),
			coloredSurfaceVertex3D(verts[1], 1, 0, tints[1]),
			coloredSurfaceVertex3D(verts[2], 1, 1, tints[2]),
			coloredSurfaceVertex3D(verts[3], 0, 1, tints[3]),
		}, indices)
	}
	addLightmapped := func(texture *render.Image, verts [4]modelPoint3, uvs [4]texturePoint, baseTints [4]color.RGBA, lightmapID int, lightmap res.GNDLightmap, lightScales [4]modelPoint3) {
		if texture == nil || lightmapAtlas.image == nil {
			addTextured(texture, verts, uvs, quadIndices012023, scaleSurfaceVertexTints(baseTints, lightScales), groundTextureDrawOptions())
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
			base := baseTints[i]
			lightScale := lightScales[i]
			tint := color.RGBA{
				R: clampColor(float64(base.R) * lightScale.x),
				G: clampColor(float64(base.G) * lightScale.y),
				B: clampColor(float64(base.B) * lightScale.z),
				A: base.A,
			}
			vertices[i] = lightmappedSurfaceVertex3D(verts[i], uvs[i], lightUVs[i], tint, textureWidth, textureHeight)
		}
		builder.addQuad(vertices, quadIndices012023)
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
					verts := [4]modelPoint3{
						{x: float64(x) * 2, y: float64(cell.Heights[0]), z: float64(y) * 2},
						{x: float64(x+1) * 2, y: float64(cell.Heights[1]), z: float64(y) * 2},
						{x: float64(x+1) * 2, y: float64(cell.Heights[3]), z: float64(y+1) * 2},
						{x: float64(x) * 2, y: float64(cell.Heights[2]), z: float64(y+1) * 2},
					}
					uvs := surfaceUVs(surface, vertexOrder)
					baseTints := topGNDSurfaceBaseTints(gnd, x, y, surface.Color)
					normals := gndTopNormalsAt(topNormals, gnd, x, y)
					if texture := m.groundTexture(manager, gndTextureName(gnd, surface.TextureID)); texture != nil {
						if lightmap, ok := gnd.Lightmap(surface.LightmapID); ok {
							addLightmapped(texture, verts, uvs, baseTints, surface.LightmapID, lightmap, vertexLightScales(lighting, normals))
						} else {
							addTextured(texture, verts, uvs, quadIndices012023, surfaceVertexTints(baseTints, cell.Heights, normals, lighting), groundTextureDrawOptions())
						}
					} else {
						addColored(verts, quadIndices012023, groundSurfaceVertexColors(gndTextureName(gnd, surface.TextureID), surface.Color, cell.Heights, normals, lighting), worldOpaqueTriangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
					}
				}
			}
			if cell.Front >= 0 && y+1 < gnd.Height {
				neighbor, neighborOK := gnd.Cell(x, y+1)
				surface, surfaceOK := gnd.Surface(cell.Front)
				if neighborOK && surfaceOK {
					vertexOrder := [4]int{0, 1, 3, 2}
					verts := [4]modelPoint3{
						{x: float64(x) * 2, y: float64(cell.Heights[2]), z: float64(y+1) * 2},
						{x: float64(x+1) * 2, y: float64(cell.Heights[3]), z: float64(y+1) * 2},
						{x: float64(x+1) * 2, y: float64(neighbor.Heights[1]), z: float64(y+1) * 2},
						{x: float64(x) * 2, y: float64(neighbor.Heights[0]), z: float64(y+1) * 2},
					}
					heights := [4]float32{cell.Heights[2], cell.Heights[3], neighbor.Heights[1], neighbor.Heights[0]}
					normals := uniformGNDNormals(modelPoint3{z: 1})
					addGNDRetainedSurface(m, manager, gnd, surface, vertexOrder, verts, heights, normals, lighting, addTextured, addColored)
				}
			}
			if cell.Right >= 0 && x+1 < gnd.Width {
				neighbor, neighborOK := gnd.Cell(x+1, y)
				surface, surfaceOK := gnd.Surface(cell.Right)
				if neighborOK && surfaceOK {
					vertexOrder := [4]int{0, 1, 3, 2}
					verts := [4]modelPoint3{
						{x: float64(x+1) * 2, y: float64(cell.Heights[3]), z: float64(y+1) * 2},
						{x: float64(x+1) * 2, y: float64(cell.Heights[1]), z: float64(y) * 2},
						{x: float64(x+1) * 2, y: float64(neighbor.Heights[0]), z: float64(y) * 2},
						{x: float64(x+1) * 2, y: float64(neighbor.Heights[2]), z: float64(y+1) * 2},
					}
					heights := [4]float32{cell.Heights[3], cell.Heights[1], neighbor.Heights[0], neighbor.Heights[2]}
					normals := uniformGNDNormals(modelPoint3{x: 1})
					addGNDRetainedSurface(m, manager, gnd, surface, vertexOrder, verts, heights, normals, lighting, addTextured, addColored)
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

func (a gndLightmapAtlas) uvs(index int) [4]texturePoint {
	if a.image == nil || a.perRow <= 0 || index < 0 {
		return [4]texturePoint{}
	}
	baseX := float32((index % a.perRow) * 8)
	baseY := float32((index / a.perRow) * 8)
	return [4]texturePoint{
		{u: baseX + 1, v: baseY + 1},
		{u: baseX + 7, v: baseY + 1},
		{u: baseX + 7, v: baseY + 7},
		{u: baseX + 1, v: baseY + 7},
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

func lightmappedSurfaceVertex3D(point modelPoint3, uv texturePoint, lightUV texturePoint, tint color.RGBA, textureWidth, textureHeight float32) render.Vertex3D {
	vertex := texturedSurfaceVertex3D(point, uv, tint, textureWidth, textureHeight)
	vertex.LightSrcX = lightUV.u
	vertex.LightSrcY = lightUV.v
	return vertex
}

func addGNDRetainedSurface(m *WorldMode, manager *res.Manager, gnd *res.GND, surface res.GNDSurface, vertexOrder [4]int, verts [4]modelPoint3, heights [4]float32, normals [4]modelPoint3, lighting sceneLighting, addTextured func(*render.Image, [4]modelPoint3, [4]texturePoint, []uint16, [4]color.RGBA, *render.DrawTrianglesOptions), addColored func([4]modelPoint3, []uint16, [4]color.RGBA, *render.DrawTrianglesOptions)) {
	baseTints := uniformGNDSurfaceBaseTints(surface.Color)
	uvs := surfaceUVs(surface, vertexOrder)
	textureName := gndTextureName(gnd, surface.TextureID)
	if texture := m.groundTexture(manager, textureName); texture != nil {
		addTextured(texture, verts, uvs, quadIndices012023, surfaceVertexTints(baseTints, heights, normals, lighting), groundTextureDrawOptions())
		return
	}
	addColored(verts, quadIndices012023, groundSurfaceVertexColors(textureName, surface.Color, heights, normals, lighting), worldOpaqueTriangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
}
