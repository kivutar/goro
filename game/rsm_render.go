package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	defaultRSMRenderRadius = 42
	rsmPlacementCellSize   = 64.0
)

type modelPoint3 struct {
	x float64
	y float64
	z float64
}

type modelWorldTriangle struct {
	verts       [3]modelPoint3
	uvs         [3]texturePoint
	color       color.RGBA
	textureName string
}

type rsmFaceMeta struct {
	uvs         [3]texturePoint
	textureName string
}

type modelBounds struct {
	min modelPoint3
	max modelPoint3
}

type rsmBounds struct {
	model modelBounds
	main  modelBounds
	nodes map[string]modelBounds
}

type rsmBoundsCacheKey struct {
	rsm  *res.RSM
	root string
}

type mat4 [16]float64

func (m *WorldMode) drawRSMModels(screen *render.Frame, manager *res.Manager, rsw *res.RSW, models map[string]*res.RSM, gnd *res.GND, projection sceneProjection, now time.Time) {
	for _, placement := range m.visibleRSMPlacements(rsw, gnd, projection) {
		rsm := m.rsmModelForPlacement(manager, models, placement.model)
		if rsm == nil {
			continue
		}
		frame, animated := rsmAnimationFrame(rsm, placement.model, now)
		if animated {
			m.drawAnimatedRSMPlacement(screen, manager, rsw, rsm, placement, frame)
			continue
		}
		meshes := m.rsmMeshesForPlacement(manager, rsm, rsw, placement)
		for _, mesh := range meshes {
			screen.DrawWorldMesh(mesh.mesh)
		}
	}
}

type visibleRSMPlacement struct {
	index int
	model res.RSWModel
	baseX float64
	baseY float64
}

type rsmPlacementGrid struct {
	rsw   *res.RSW
	gnd   *res.GND
	cells map[rsmPlacementGridCell][]visibleRSMPlacement
}

type rsmPlacementGridCell struct {
	x int
	y int
}

type animatedRSMNodeCache struct {
	frame    int
	matrices map[string]mat4
}

func (m *WorldMode) visibleRSMPlacements(rsw *res.RSW, gnd *res.GND, projection sceneProjection) []visibleRSMPlacement {
	if rsw == nil || gnd == nil {
		return nil
	}
	radius := rsmRenderRadius()
	grid := m.rsmPlacementGridFor(rsw, gnd)
	if grid == nil {
		return nil
	}
	queryRadius := radius * 2
	minCellX := rsmPlacementGridCoord(projection.playerX - queryRadius)
	maxCellX := rsmPlacementGridCoord(projection.playerX + queryRadius)
	minCellY := rsmPlacementGridCoord(projection.playerY - queryRadius)
	maxCellY := rsmPlacementGridCoord(projection.playerY + queryRadius)
	visible := make([]visibleRSMPlacement, 0, 32)
	for cellY := minCellY; cellY <= maxCellY; cellY++ {
		for cellX := minCellX; cellX <= maxCellX; cellX++ {
			for _, placement := range grid.cells[rsmPlacementGridCell{x: cellX, y: cellY}] {
				dx := placement.baseX - projection.playerX
				dy := placement.baseY - projection.playerY
				if math.Abs(dx) > queryRadius || math.Abs(dy) > queryRadius {
					continue
				}
				visible = append(visible, placement)
			}
		}
	}
	return visible
}

func (m *WorldMode) rsmPlacementGridFor(rsw *res.RSW, gnd *res.GND) *rsmPlacementGrid {
	if rsw == nil || gnd == nil {
		return nil
	}
	if m.rsmPlacementGrid != nil && m.rsmPlacementGrid.rsw == rsw && m.rsmPlacementGrid.gnd == gnd {
		return m.rsmPlacementGrid
	}
	grid := &rsmPlacementGrid{
		rsw:   rsw,
		gnd:   gnd,
		cells: make(map[rsmPlacementGridCell][]visibleRSMPlacement, len(rsw.Models)/4+1),
	}
	for index, model := range rsw.Models {
		if model.Filename == "" {
			continue
		}
		baseX := float64(model.Position.X) + float64(gnd.Width)
		baseY := float64(model.Position.Z) + float64(gnd.Height)
		cell := rsmPlacementGridCell{
			x: rsmPlacementGridCoord(baseX),
			y: rsmPlacementGridCoord(baseY),
		}
		grid.cells[cell] = append(grid.cells[cell], visibleRSMPlacement{
			index: index,
			model: model,
			baseX: baseX,
			baseY: baseY,
		})
	}
	m.rsmPlacementGrid = grid
	return grid
}

func rsmPlacementGridCoord(value float64) int {
	return int(math.Floor(value / rsmPlacementCellSize))
}

func (m *WorldMode) rsmModelForPlacement(manager *res.Manager, models map[string]*res.RSM, placement res.RSWModel) *res.RSM {
	if placement.Filename == "" {
		return nil
	}
	rsm, ok := models[placement.Filename]
	if !ok {
		loaded, err := loadRSMModel(manager, placement.Filename)
		if err == nil {
			rsm = loaded
		}
		models[placement.Filename] = rsm
	}
	return rsm
}

func (m *WorldMode) rsmMeshesForPlacement(manager *res.Manager, rsm *res.RSM, rsw *res.RSW, visible visibleRSMPlacement) []retainedWorldMesh {
	if m.rsmMeshCache == nil {
		m.rsmMeshCache = make(map[int][]retainedWorldMesh)
	}
	if rsm == nil {
		return nil
	}
	if meshes, ok := m.rsmMeshCache[visible.index]; ok {
		return meshes
	}
	context, ok := m.rsmPlacementContext(rsm, rsw, visible)
	if !ok {
		return nil
	}
	if m.rsmNodeMatrices == nil {
		m.rsmNodeMatrices = make(map[*res.RSM]map[string]mat4)
	}
	nodeMatrices, ok := m.rsmNodeMatrices[rsm]
	if !ok {
		nodeMatrices = buildRSMNodeMatrices(rsm, 0)
		m.rsmNodeMatrices[rsm] = nodeMatrices
	}
	builders := make(map[retainedMeshKey]*retainedMeshBuilder)
	builderOrder := make([]*retainedMeshBuilder, 0, 4)
	builderFor := func(texture *render.Image, options *render.DrawTrianglesOptions) *retainedMeshBuilder {
		if texture == nil || options == nil {
			return nil
		}
		key := retainedMeshKey{texture: texture, options: *options}
		builder := builders[key]
		if builder == nil {
			builder = &retainedMeshBuilder{texture: texture, options: *options}
			builders[key] = builder
			builderOrder = append(builderOrder, builder)
		}
		return builder
	}
	for _, nodeIndex := range context.nodeIndices {
		node := &rsm.Nodes[nodeIndex]
		for _, worldTri := range buildRSMNodeWorldTriangles(rsm, node, nodeMatrices[node.Name], context.instance, context.lighting, m.rsmFaceMetas(rsm, node)) {
			texture := m.groundTexture(manager, worldTri.textureName)
			if texture != nil {
				bounds := texture.Bounds()
				w, h := float32(bounds.Dx()), float32(bounds.Dy())
				builder := builderFor(texture, rsmDrawOptionsForTriangle(render.FilterLinear, render.AddressClampToEdge, worldTri, visible.index))
				if builder != nil {
					builder.addTriangle(
						texturedSurfaceVertex3D(worldTri.verts[0], worldTri.uvs[0], worldTri.color, w, h),
						texturedSurfaceVertex3D(worldTri.verts[1], worldTri.uvs[1], worldTri.color, w, h),
						texturedSurfaceVertex3D(worldTri.verts[2], worldTri.uvs[2], worldTri.color, w, h),
					)
				}
				continue
			}
			if m.whitePixel == nil {
				m.whitePixel = render.NewImage(1, 1)
				m.whitePixel.Fill(color.White)
			}
			builder := builderFor(m.whitePixel, rsmDrawOptionsForTriangle(render.FilterNearest, render.AddressUnsafe, worldTri, visible.index))
			if builder != nil {
				builder.addTriangle(
					coloredSurfaceVertex3D(worldTri.verts[0], 0, 0, worldTri.color),
					coloredSurfaceVertex3D(worldTri.verts[1], 1, 0, worldTri.color),
					coloredSurfaceVertex3D(worldTri.verts[2], 1, 1, worldTri.color),
				)
			}
		}
	}
	var meshes []retainedWorldMesh
	for _, builder := range builderOrder {
		builder.flush()
		meshes = append(meshes, builder.meshes...)
	}
	m.rsmMeshCache[visible.index] = meshes
	return meshes
}

type rsmPlacementContext struct {
	nodeIndices []int
	instance    modelInstance
	lighting    sceneLighting
}

func (m *WorldMode) rsmPlacementContext(rsm *res.RSM, rsw *res.RSW, visible visibleRSMPlacement) (rsmPlacementContext, bool) {
	placement := visible.model
	rootName := selectedRSMRootName(rsm, placement.NodeName)
	nodeIndices := selectedRSMNodeIndices(rsm, rootName)
	if len(nodeIndices) == 0 {
		return rsmPlacementContext{}, false
	}
	boundsKey := rsmBoundsCacheKey{rsm: rsm, root: rootName}
	if m.rsmBoundsCache == nil {
		m.rsmBoundsCache = make(map[rsmBoundsCacheKey]rsmBounds)
	}
	bounds, ok := m.rsmBoundsCache[boundsKey]
	if !ok {
		bounds = calculateRSMBoundsForNodes(rsm, nodeIndices)
		m.rsmBoundsCache[boundsKey] = bounds
	}
	return rsmPlacementContext{
		nodeIndices: nodeIndices,
		instance: modelInstance{
			bounds: bounds.model,
			matrix: buildRSMInstanceMatrix(rsm, placement, visible.baseX, visible.baseY, bounds.model),
		},
		lighting: m.sceneLighting(rsw),
	}, true
}

type animatedRSMDrawBatch struct {
	screen  *render.Frame
	texture *render.Image
	options render.DrawTrianglesOptions
	verts   []render.Vertex3D
	indices []uint16
}

type animatedRSMScratch struct {
	batches    []animatedRSMDrawBatch
	worldVerts []modelPoint3
}

func (s *animatedRSMScratch) reset() {
	for i := range s.batches {
		s.batches[i].clear()
	}
	s.batches = s.batches[:0]
	s.worldVerts = s.worldVerts[:0]
}

func (b *animatedRSMDrawBatch) addTriangle(a, c, d render.Vertex3D) {
	if b.texture == nil {
		return
	}
	if len(b.verts)+3 > retainedWorldMeshMaxVertices {
		b.flush()
	}
	base := uint16(len(b.verts))
	b.verts = append(b.verts, a, c, d)
	b.indices = append(b.indices, base, base+1, base+2)
}

func (b *animatedRSMDrawBatch) flush() {
	if len(b.verts) == 0 || len(b.indices) == 0 || b.texture == nil {
		b.verts = b.verts[:0]
		b.indices = b.indices[:0]
		return
	}
	b.screen.DrawTriangles3DOwned(b.verts, b.indices, b.texture, &b.options)
	b.verts = nil
	b.indices = nil
}

func (b *animatedRSMDrawBatch) clear() {
	b.screen = nil
	b.texture = nil
	b.options = render.DrawTrianglesOptions{}
	b.verts = nil
	b.indices = nil
}

func (m *WorldMode) drawAnimatedRSMPlacement(screen *render.Frame, manager *res.Manager, rsw *res.RSW, rsm *res.RSM, visible visibleRSMPlacement, frame int) {
	context, ok := m.rsmPlacementContext(rsm, rsw, visible)
	if !ok {
		return
	}
	nodeMatrices := m.animatedRSMNodeMatrices(rsm, frame)
	batchOrder := m.rsmAnimScratch.batches[:0]
	for _, nodeIndex := range context.nodeIndices {
		node := &rsm.Nodes[nodeIndex]
		m.drawAnimatedRSMNodeTriangles(screen, manager, rsm, node, nodeMatrices[node.Name], context, visible, &batchOrder)
	}
	for i := range batchOrder {
		batchOrder[i].flush()
		batchOrder[i].clear()
	}
	m.rsmAnimScratch.batches = batchOrder[:0]
}

func animatedRSMBatchFor(screen *render.Frame, batches *[]animatedRSMDrawBatch, texture *render.Image, options render.DrawTrianglesOptions) *animatedRSMDrawBatch {
	if texture == nil {
		return nil
	}
	order := *batches
	for i := range order {
		if order[i].texture == texture && order[i].options == options {
			return &order[i]
		}
	}
	order = append(order, animatedRSMDrawBatch{screen: screen, texture: texture, options: options})
	*batches = order
	return &order[len(order)-1]
}

func (m *WorldMode) drawAnimatedRSMNodeTriangles(screen *render.Frame, manager *res.Manager, rsm *res.RSM, node *res.RSMNode, nodeMatrix mat4, context rsmPlacementContext, visible visibleRSMPlacement, batchOrder *[]animatedRSMDrawBatch) {
	worldVerts := m.animatedRSMNodeWorldVerts(rsm, node, nodeMatrix, context.instance)
	if len(worldVerts) == 0 {
		return
	}
	faceMetas := m.rsmFaceMetas(rsm, node)
	for faceIndex, face := range node.Faces {
		if int(face.VertexIndices[0]) >= len(worldVerts) || int(face.VertexIndices[1]) >= len(worldVerts) || int(face.VertexIndices[2]) >= len(worldVerts) {
			continue
		}
		a := worldVerts[face.VertexIndices[0]]
		b := worldVerts[face.VertexIndices[1]]
		c := worldVerts[face.VertexIndices[2]]
		var faceMeta rsmFaceMeta
		if faceIndex < len(faceMetas) {
			faceMeta = faceMetas[faceIndex]
		} else {
			faceMeta.textureName, faceMeta.uvs = rsmFaceTexture(rsm, node, face)
		}
		faceColor := rsmFaceColor(rsm, faceMeta.textureName, a, b, c, context.lighting)
		texture := m.groundTexture(manager, faceMeta.textureName)
		if texture != nil {
			bounds := texture.Bounds()
			w, h := float32(bounds.Dx()), float32(bounds.Dy())
			options := rsmDrawOptionsForVertices(render.FilterLinear, render.AddressClampToEdge, a, b, c, visible.index)
			batch := animatedRSMBatchFor(screen, batchOrder, texture, options)
			if batch != nil {
				batch.addTriangle(
					texturedSurfaceVertex3D(a, faceMeta.uvs[0], faceColor, w, h),
					texturedSurfaceVertex3D(b, faceMeta.uvs[1], faceColor, w, h),
					texturedSurfaceVertex3D(c, faceMeta.uvs[2], faceColor, w, h),
				)
			}
			continue
		}
		if m.whitePixel == nil {
			m.whitePixel = render.NewImage(1, 1)
			m.whitePixel.Fill(color.White)
		}
		options := rsmDrawOptionsForVertices(render.FilterNearest, render.AddressUnsafe, a, b, c, visible.index)
		batch := animatedRSMBatchFor(screen, batchOrder, m.whitePixel, options)
		if batch != nil {
			batch.addTriangle(
				coloredSurfaceVertex3D(a, 0, 0, faceColor),
				coloredSurfaceVertex3D(b, 1, 0, faceColor),
				coloredSurfaceVertex3D(c, 1, 1, faceColor),
			)
		}
	}
}

func (m *WorldMode) animatedRSMNodeWorldVerts(rsm *res.RSM, node *res.RSMNode, nodeMatrix mat4, instance modelInstance) []modelPoint3 {
	if node == nil || len(node.Vertices) == 0 {
		return nil
	}
	if cap(m.rsmAnimScratch.worldVerts) < len(node.Vertices) {
		m.rsmAnimScratch.worldVerts = make([]modelPoint3, len(node.Vertices))
	}
	worldVerts := m.rsmAnimScratch.worldVerts[:len(node.Vertices)]
	modelMatrix := rsmNodeModelMatrix(rsm, node, nodeMatrix, instance)
	for i, vertex := range node.Vertices {
		worldVerts[i] = mat4TransformPoint(modelMatrix, vectorFromRSM(vertex))
	}
	return worldVerts
}

func (m *WorldMode) animatedRSMNodeMatrices(rsm *res.RSM, frame int) map[string]mat4 {
	if rsm == nil {
		return nil
	}
	if m.rsmAnimNodes == nil {
		m.rsmAnimNodes = make(map[*res.RSM]animatedRSMNodeCache)
	}
	if cached, ok := m.rsmAnimNodes[rsm]; ok && cached.frame == frame {
		return cached.matrices
	}
	// Frame values are milliseconds, so retaining every sampled pose makes an
	// otherwise idle map grow continuously. Only the current pose is reusable.
	matrices := buildRSMNodeMatrices(rsm, frame)
	m.rsmAnimNodes[rsm] = animatedRSMNodeCache{frame: frame, matrices: matrices}
	return matrices
}

func rsmDrawOptionsForTriangle(filter render.Filter, address render.Address, tri modelWorldTriangle, placementIndex int) *render.DrawTrianglesOptions {
	if rsmTriangleVertsAreHorizontal(tri.verts[0], tri.verts[1], tri.verts[2]) {
		return rsmHorizontalSurfaceDrawOptions(filter, address, placementIndex)
	}
	return rsmModelDrawOptions(filter, address)
}

func rsmDrawOptionsForVertices(filter render.Filter, address render.Address, a, b, c modelPoint3, placementIndex int) render.DrawTrianglesOptions {
	options := render.DrawTrianglesOptions{
		Filter:     filter,
		Address:    address,
		DepthTest:  true,
		DepthWrite: true,
		DepthBias:  rsmModelDepthBias,
	}
	if rsmTriangleVertsAreHorizontal(a, b, c) && placementIndex >= 0 {
		layer := placementIndex % rsmHorizontalDepthBiasLayers
		options.DepthBias += float32(layer) * rsmHorizontalDepthBiasStep
	}
	return options
}

func rsmTriangleVertsAreHorizontal(a, b, c modelPoint3) bool {
	normal := normalize3(cross3(sub3(b, a), sub3(c, a)))
	return math.Abs(normal.y) >= 0.98
}

func selectedRSMRootName(rsm *res.RSM, rootName string) string {
	if rsm == nil || rootName == "" {
		return ""
	}
	for i := range rsm.Nodes {
		if rsm.Nodes[i].Name == rootName {
			return rootName
		}
	}
	return ""
}

func selectedRSMNodeIndices(rsm *res.RSM, rootName string) []int {
	if rsm == nil || len(rsm.Nodes) == 0 {
		return nil
	}
	if rootName == "" {
		return allRSMNodeIndices(rsm)
	}
	nodes := make(map[string]*res.RSMNode, len(rsm.Nodes))
	for i := range rsm.Nodes {
		nodes[rsm.Nodes[i].Name] = &rsm.Nodes[i]
	}
	if nodes[rootName] == nil {
		return allRSMNodeIndices(rsm)
	}
	selected := map[string]struct{}{rootName: {}}
	descendantRoots := map[string]struct{}{rootName: {}}
	for node := nodes[rootName]; node != nil; {
		parentName := node.ParentName
		if parentName == "" || parentName == node.Name {
			break
		}
		parent := nodes[parentName]
		if parent == nil {
			break
		}
		selected[parent.Name] = struct{}{}
		node = parent
	}

	changed := true
	for changed {
		changed = false
		for i := range rsm.Nodes {
			node := &rsm.Nodes[i]
			if _, ok := descendantRoots[node.Name]; ok {
				continue
			}
			if _, ok := descendantRoots[node.ParentName]; !ok {
				continue
			}
			selected[node.Name] = struct{}{}
			descendantRoots[node.Name] = struct{}{}
			changed = true
		}
	}

	indices := make([]int, 0, len(selected))
	for i := range rsm.Nodes {
		if _, ok := selected[rsm.Nodes[i].Name]; ok {
			indices = append(indices, i)
		}
	}
	return indices
}

func allRSMNodeIndices(rsm *res.RSM) []int {
	if rsm == nil || len(rsm.Nodes) == 0 {
		return nil
	}
	indices := make([]int, len(rsm.Nodes))
	for i := range indices {
		indices[i] = i
	}
	return indices
}

type modelInstance struct {
	bounds modelBounds
	matrix mat4
}

func (m *WorldMode) rsmFaceMetas(rsm *res.RSM, node *res.RSMNode) []rsmFaceMeta {
	if rsm == nil || node == nil || len(node.Faces) == 0 {
		return nil
	}
	if m.rsmFaceMetaCache == nil {
		m.rsmFaceMetaCache = make(map[*res.RSM]map[*res.RSMNode][]rsmFaceMeta)
	}
	nodeCache := m.rsmFaceMetaCache[rsm]
	if nodeCache == nil {
		nodeCache = make(map[*res.RSMNode][]rsmFaceMeta)
		m.rsmFaceMetaCache[rsm] = nodeCache
	}
	if metas := nodeCache[node]; metas != nil {
		return metas
	}
	metas := make([]rsmFaceMeta, len(node.Faces))
	for i, face := range node.Faces {
		textureName, uvs := rsmFaceTexture(rsm, node, face)
		metas[i] = rsmFaceMeta{textureName: textureName, uvs: uvs}
	}
	nodeCache[node] = metas
	return metas
}

func buildRSMNodeWorldTriangles(rsm *res.RSM, node *res.RSMNode, nodeMatrix mat4, instance modelInstance, lighting sceneLighting, faceMetas []rsmFaceMeta) []modelWorldTriangle {
	if len(node.Vertices) == 0 || len(node.Faces) == 0 {
		return nil
	}

	modelMatrix := rsmNodeModelMatrix(rsm, node, nodeMatrix, instance)
	worldVerts := make([]modelPoint3, len(node.Vertices))
	for i, vertex := range node.Vertices {
		world := mat4TransformPoint(modelMatrix, vectorFromRSM(vertex))
		worldVerts[i] = world
	}

	triangles := make([]modelWorldTriangle, 0, len(node.Faces))
	for faceIndex, face := range node.Faces {
		if int(face.VertexIndices[0]) >= len(worldVerts) || int(face.VertexIndices[1]) >= len(worldVerts) || int(face.VertexIndices[2]) >= len(worldVerts) {
			continue
		}
		a := worldVerts[face.VertexIndices[0]]
		b := worldVerts[face.VertexIndices[1]]
		c := worldVerts[face.VertexIndices[2]]
		var faceMeta rsmFaceMeta
		if faceIndex < len(faceMetas) {
			faceMeta = faceMetas[faceIndex]
		} else {
			faceMeta.textureName, faceMeta.uvs = rsmFaceTexture(rsm, node, face)
		}
		faceColor := rsmFaceColor(rsm, faceMeta.textureName, a, b, c, lighting)
		triangles = append(triangles, modelWorldTriangle{
			verts:       [3]modelPoint3{a, b, c},
			uvs:         faceMeta.uvs,
			color:       faceColor,
			textureName: faceMeta.textureName,
		})
	}
	return triangles
}

func rsmNodeModelMatrix(rsm *res.RSM, node *res.RSMNode, nodeMatrix mat4, instance modelInstance) mat4 {
	localMatrix := mat4Identity()
	localMatrix = mat4Translate(localMatrix, modelPoint3{
		x: -(instance.bounds.min.x + instance.bounds.max.x) * 0.5,
		y: instance.bounds.max.y,
		z: -(instance.bounds.min.z + instance.bounds.max.z) * 0.5,
	})
	localMatrix = mat4Scale(localMatrix, modelPoint3{x: 1, y: -1, z: 1})
	localMatrix = mat4Multiply(localMatrix, nodeMatrix)
	if len(rsm.Nodes) != 1 {
		localMatrix = mat4Translate(localMatrix, vectorFromRSM(node.Offset))
	}
	localMatrix = mat4Multiply(localMatrix, mat4FromMat3(node.Matrix))
	return mat4Multiply(instance.matrix, localMatrix)
}

func buildRSMInstanceMatrix(rsm *res.RSM, placement res.RSWModel, baseX, baseY float64, bounds modelBounds) mat4 {
	matrix := mat4Identity()
	matrix = mat4Translate(matrix, modelPoint3{x: baseX, y: float64(placement.Position.Y), z: baseY})
	matrix = mat4RotateZ(matrix, degreesToRadians(float64(-placement.Rotation.Z)))
	matrix = mat4RotateX(matrix, degreesToRadians(float64(-placement.Rotation.X)))
	matrix = mat4RotateY(matrix, degreesToRadians(float64(placement.Rotation.Y)))
	matrix = mat4Scale(matrix, vectorFromRSW(placement.Scale))

	if main := rsmMainNode(rsm); main != nil && rsmVersionAtLeast(rsm, 2, 2) {
		matrix = mat4Scale(matrix, modelPoint3{x: 1, y: -1, z: 1})
		matrix = mat4Translate(matrix, vectorFromRSM(main.Offset))
		matrix = mat4Translate(matrix, modelPoint3{y: (bounds.max.y - bounds.min.y) * 0.5})
		matrix = mat4Translate(matrix, modelPoint3{
			x: (bounds.max.x + bounds.min.x) * 0.5,
			y: (bounds.max.y + bounds.min.y) * 0.5,
			z: (bounds.max.z + bounds.min.z) * 0.5,
		})
	}
	return matrix
}

func buildRSMNodeMatrices(rsm *res.RSM, frame int) map[string]mat4 {
	nodes := make(map[string]*res.RSMNode, len(rsm.Nodes))
	for i := range rsm.Nodes {
		nodes[rsm.Nodes[i].Name] = &rsm.Nodes[i]
	}
	out := make(map[string]mat4, len(rsm.Nodes))
	visiting := make(map[string]bool, len(rsm.Nodes))
	var build func(*res.RSMNode) mat4
	build = func(node *res.RSMNode) mat4 {
		if matrix, ok := out[node.Name]; ok {
			return matrix
		}
		if visiting[node.Name] {
			return mat4Identity()
		}
		visiting[node.Name] = true
		matrix := mat4Identity()
		if parent := nodes[node.ParentName]; parent != nil && parent != node && node.ParentName != node.Name {
			matrix = build(parent)
		}
		matrix = mat4Translate(matrix, sampleRSMPosition(node, frame))
		if rotation, ok := sampleRSMRotation(node, frame); ok {
			matrix = mat4RotateQuat(matrix, rotation)
		} else {
			matrix = mat4RotateAxis(matrix, vectorFromRSM(node.RotationAxis), float64(node.RotationAngle))
		}
		matrix = mat4Scale(matrix, sampleRSMScale(node, frame))
		out[node.Name] = matrix
		visiting[node.Name] = false
		return matrix
	}
	for i := range rsm.Nodes {
		build(&rsm.Nodes[i])
	}
	return out
}

func rsmAnimationFrame(rsm *res.RSM, _ res.RSWModel, now time.Time) (int, bool) {
	if !rsmHasNodeAnimation(rsm) {
		return 0, false
	}
	// robr drives animated map models from the global millisecond tick. The RSW
	// animation speed is not applied here; the keyframe times are already in ms.
	frame := int(now.UnixMilli())
	animLen := rsmAnimationLength(rsm)
	if animLen > 0 {
		frame %= animLen
		if frame < 0 {
			frame += animLen
		}
	}
	return frame, true
}

func rsmAnimationLength(rsm *res.RSM) int {
	if rsm == nil {
		return 1
	}
	if rsm.AnimLength > 0 {
		return int(rsm.AnimLength)
	}
	maxFrame := 0
	for i := range rsm.Nodes {
		node := &rsm.Nodes[i]
		for _, key := range node.PositionKeyframes {
			frame := int(key.Frame)
			if frame > maxFrame {
				maxFrame = frame
			}
		}
		for _, key := range node.RotationKeyframes {
			frame := int(key.Frame)
			if frame > maxFrame {
				maxFrame = frame
			}
		}
		for _, key := range node.ScaleKeyframes {
			frame := int(key.Frame)
			if frame > maxFrame {
				maxFrame = frame
			}
		}
	}
	return maxFrame + 1
}

func rsmHasNodeAnimation(rsm *res.RSM) bool {
	if rsm == nil {
		return false
	}
	for i := range rsm.Nodes {
		node := &rsm.Nodes[i]
		if len(node.PositionKeyframes) > 0 || len(node.RotationKeyframes) > 1 || len(node.ScaleKeyframes) > 0 {
			return true
		}
	}
	return false
}

func sampleRSMPosition(node *res.RSMNode, frame int) modelPoint3 {
	keys := node.PositionKeyframes
	if len(keys) == 0 {
		return vectorFromRSM(node.Position)
	}
	if len(keys) == 1 || frame <= int(keys[0].Frame) {
		return vectorFromRSM(keys[0].Pos)
	}
	if frame >= int(keys[len(keys)-1].Frame) {
		return vectorFromRSM(keys[len(keys)-1].Pos)
	}
	for index := 1; index < len(keys); index++ {
		if frame > int(keys[index].Frame) {
			continue
		}
		prev, next := keys[index-1], keys[index]
		t := rsmKeyframeT(frame, prev.Frame, next.Frame)
		return lerpPoint3(vectorFromRSM(prev.Pos), vectorFromRSM(next.Pos), t)
	}
	return vectorFromRSM(node.Position)
}

func sampleRSMScale(node *res.RSMNode, frame int) modelPoint3 {
	keys := node.ScaleKeyframes
	if len(keys) == 0 {
		return vectorFromRSM(node.Scale)
	}
	if len(keys) == 1 || frame <= int(keys[0].Frame) {
		return vectorFromRSM(keys[0].Scale)
	}
	if frame >= int(keys[len(keys)-1].Frame) {
		return vectorFromRSM(keys[len(keys)-1].Scale)
	}
	for index := 1; index < len(keys); index++ {
		if frame > int(keys[index].Frame) {
			continue
		}
		prev, next := keys[index-1], keys[index]
		t := rsmKeyframeT(frame, prev.Frame, next.Frame)
		return lerpPoint3(vectorFromRSM(prev.Scale), vectorFromRSM(next.Scale), t)
	}
	return vectorFromRSM(node.Scale)
}

func sampleRSMRotation(node *res.RSMNode, frame int) ([4]float32, bool) {
	keys := node.RotationKeyframes
	if len(keys) == 0 {
		return [4]float32{}, false
	}
	if len(keys) == 1 || frame <= int(keys[0].Frame) {
		return keys[0].Quaternion, true
	}
	if frame >= int(keys[len(keys)-1].Frame) {
		return keys[len(keys)-1].Quaternion, true
	}
	for index := 1; index < len(keys); index++ {
		if frame > int(keys[index].Frame) {
			continue
		}
		prev, next := keys[index-1], keys[index]
		t := rsmKeyframeT(frame, prev.Frame, next.Frame)
		return slerpRSMQuaternion(prev.Quaternion, next.Quaternion, t), true
	}
	return keys[0].Quaternion, true
}

func rsmKeyframeT(frame int, from int32, to int32) float64 {
	delta := int(to - from)
	if delta <= 0 {
		return 0
	}
	return float64(frame-int(from)) / float64(delta)
}

func lerpPoint3(a, b modelPoint3, t float64) modelPoint3 {
	return modelPoint3{
		x: a.x + (b.x-a.x)*t,
		y: a.y + (b.y-a.y)*t,
		z: a.z + (b.z-a.z)*t,
	}
}

func slerpRSMQuaternion(a, b [4]float32, t float64) [4]float32 {
	from := [4]float64{float64(a[0]), float64(a[1]), float64(a[2]), float64(a[3])}
	to := [4]float64{float64(b[0]), float64(b[1]), float64(b[2]), float64(b[3])}
	dot := from[0]*to[0] + from[1]*to[1] + from[2]*to[2] + from[3]*to[3]
	if dot < 0 {
		dot = -dot
		for i := range to {
			to[i] = -to[i]
		}
	}
	var out [4]float64
	if dot > 0.9995 {
		for i := range out {
			out[i] = from[i] + (to[i]-from[i])*t
		}
	} else {
		theta0 := math.Acos(math.Max(-1, math.Min(1, dot)))
		sinTheta0 := math.Sin(theta0)
		if math.Abs(sinTheta0) <= 1e-6 {
			return a
		}
		theta := theta0 * t
		sinTheta := math.Sin(theta)
		s0 := math.Cos(theta) - dot*sinTheta/sinTheta0
		s1 := sinTheta / sinTheta0
		for i := range out {
			out[i] = from[i]*s0 + to[i]*s1
		}
	}
	length := math.Sqrt(out[0]*out[0] + out[1]*out[1] + out[2]*out[2] + out[3]*out[3])
	if length == 0 {
		return [4]float32{0, 0, 0, 1}
	}
	return [4]float32{
		float32(out[0] / length),
		float32(out[1] / length),
		float32(out[2] / length),
		float32(out[3] / length),
	}
}

func calculateRSMBoundsForNodes(rsm *res.RSM, nodeIndices []int) rsmBounds {
	bounds := rsmBounds{
		model: emptyModelBounds(),
		main:  modelBounds{},
		nodes: make(map[string]modelBounds, len(rsm.Nodes)),
	}
	nodeMatrices := buildRSMNodeMatrices(rsm, 0)
	for _, nodeIndex := range nodeIndices {
		if nodeIndex < 0 || nodeIndex >= len(rsm.Nodes) {
			continue
		}
		node := &rsm.Nodes[nodeIndex]
		matrix := nodeMatrices[node.Name]
		if len(rsm.Nodes) != 1 {
			matrix = mat4Translate(matrix, vectorFromRSM(node.Offset))
		}
		matrix = mat4Multiply(matrix, mat4FromMat3(node.Matrix))

		nodeBounds := transformedNodeBounds(node, matrix)
		bounds.nodes[node.Name] = nodeBounds
		if !nodeBounds.empty() {
			bounds.model.include(nodeBounds)
		}
	}
	if bounds.model.empty() {
		bounds.model = modelBounds{}
	}
	if len(nodeIndices) > 0 && nodeIndices[0] >= 0 && nodeIndices[0] < len(rsm.Nodes) {
		main := &rsm.Nodes[nodeIndices[0]]
		bounds.main = bounds.nodes[main.Name]
	}
	if bounds.main.empty() {
		bounds.main = bounds.model
	}
	return bounds
}

func transformedNodeBounds(node *res.RSMNode, matrix mat4) modelBounds {
	bounds := emptyModelBounds()
	for _, vertex := range node.Vertices {
		point := mat4TransformPoint(matrix, vectorFromRSM(vertex))
		bounds.min.x = math.Min(bounds.min.x, point.x)
		bounds.min.y = math.Min(bounds.min.y, point.y)
		bounds.min.z = math.Min(bounds.min.z, point.z)
		bounds.max.x = math.Max(bounds.max.x, point.x)
		bounds.max.y = math.Max(bounds.max.y, point.y)
		bounds.max.z = math.Max(bounds.max.z, point.z)
	}
	return bounds
}

func emptyModelBounds() modelBounds {
	return modelBounds{
		min: modelPoint3{x: math.Inf(1), y: math.Inf(1), z: math.Inf(1)},
		max: modelPoint3{x: math.Inf(-1), y: math.Inf(-1), z: math.Inf(-1)},
	}
}

func (b modelBounds) empty() bool {
	return math.IsInf(b.min.x, 0)
}

func (b *modelBounds) include(other modelBounds) {
	if other.empty() {
		return
	}
	if b.empty() {
		*b = other
		return
	}
	b.min.x = math.Min(b.min.x, other.min.x)
	b.min.y = math.Min(b.min.y, other.min.y)
	b.min.z = math.Min(b.min.z, other.min.z)
	b.max.x = math.Max(b.max.x, other.max.x)
	b.max.y = math.Max(b.max.y, other.max.y)
	b.max.z = math.Max(b.max.z, other.max.z)
}

func rsmMainNode(rsm *res.RSM) *res.RSMNode {
	if rsm == nil || len(rsm.Nodes) == 0 {
		return nil
	}
	if rsm.MainNodeName != "" {
		for i := range rsm.Nodes {
			if rsm.Nodes[i].Name == rsm.MainNodeName {
				return &rsm.Nodes[i]
			}
		}
	}
	return &rsm.Nodes[0]
}

func vectorFromRSM(v res.RSMVector3) modelPoint3 {
	return modelPoint3{x: float64(v.X), y: float64(v.Y), z: float64(v.Z)}
}

func vectorFromRSW(v res.RSWVector3) modelPoint3 {
	return modelPoint3{x: float64(v.X), y: float64(v.Y), z: float64(v.Z)}
}

func rsmVersionAtLeast(rsm *res.RSM, major, minor byte) bool {
	return rsm != nil && (rsm.VersionMajor > major || rsm.VersionMajor == major && rsm.VersionMinor >= minor)
}

func mat4Identity() mat4 {
	return mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func mat4Multiply(a, b mat4) mat4 {
	var out mat4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			out[col*4+row] =
				a[0*4+row]*b[col*4+0] +
					a[1*4+row]*b[col*4+1] +
					a[2*4+row]*b[col*4+2] +
					a[3*4+row]*b[col*4+3]
		}
	}
	return out
}

func mat4Translate(matrix mat4, v modelPoint3) mat4 {
	t := mat4Identity()
	t[12] = v.x
	t[13] = v.y
	t[14] = v.z
	return mat4Multiply(matrix, t)
}

func mat4Scale(matrix mat4, v modelPoint3) mat4 {
	s := mat4Identity()
	s[0] = v.x
	s[5] = v.y
	s[10] = v.z
	return mat4Multiply(matrix, s)
}

func mat4RotateX(matrix mat4, angle float64) mat4 {
	s, c := math.Sin(angle), math.Cos(angle)
	r := mat4Identity()
	r[5] = c
	r[6] = s
	r[9] = -s
	r[10] = c
	return mat4Multiply(matrix, r)
}

func mat4RotateY(matrix mat4, angle float64) mat4 {
	s, c := math.Sin(angle), math.Cos(angle)
	r := mat4Identity()
	r[0] = c
	r[2] = -s
	r[8] = s
	r[10] = c
	return mat4Multiply(matrix, r)
}

func mat4RotateZ(matrix mat4, angle float64) mat4 {
	s, c := math.Sin(angle), math.Cos(angle)
	r := mat4Identity()
	r[0] = c
	r[1] = s
	r[4] = -s
	r[5] = c
	return mat4Multiply(matrix, r)
}

func mat4RotateAxis(matrix mat4, axis modelPoint3, angle float64) mat4 {
	axis = normalize3(axis)
	if axis == (modelPoint3{}) || angle == 0 {
		return matrix
	}
	x, y, z := axis.x, axis.y, axis.z
	s, c := math.Sin(angle), math.Cos(angle)
	t := 1 - c
	r := mat4Identity()
	r[0] = x*x*t + c
	r[1] = y*x*t + z*s
	r[2] = z*x*t - y*s
	r[4] = x*y*t - z*s
	r[5] = y*y*t + c
	r[6] = z*y*t + x*s
	r[8] = x*z*t + y*s
	r[9] = y*z*t - x*s
	r[10] = z*z*t + c
	return mat4Multiply(matrix, r)
}

func mat4RotateQuat(matrix mat4, q [4]float32) mat4 {
	x, y, z, w := float64(q[0]), float64(q[1]), float64(q[2]), float64(q[3])
	length := math.Sqrt(x*x + y*y + z*z + w*w)
	if length == 0 {
		return matrix
	}
	x /= length
	y /= length
	z /= length
	w /= length

	x2, y2, z2 := x+x, y+y, z+z
	xx, xy, xz := x*x2, x*y2, x*z2
	yy, yz, zz := y*y2, y*z2, z*z2
	wx, wy, wz := w*x2, w*y2, w*z2

	r := mat4Identity()
	r[0] = 1 - yy - zz
	r[1] = xy + wz
	r[2] = xz - wy
	r[4] = xy - wz
	r[5] = 1 - xx - zz
	r[6] = yz + wx
	r[8] = xz + wy
	r[9] = yz - wx
	r[10] = 1 - xx - yy
	return mat4Multiply(matrix, r)
}

func mat4FromMat3(matrix [9]float32) mat4 {
	out := mat4Identity()
	out[0] = float64(matrix[0])
	out[1] = float64(matrix[1])
	out[2] = float64(matrix[2])
	out[4] = float64(matrix[3])
	out[5] = float64(matrix[4])
	out[6] = float64(matrix[5])
	out[8] = float64(matrix[6])
	out[9] = float64(matrix[7])
	out[10] = float64(matrix[8])
	return out
}

func mat4TransformPoint(matrix mat4, point modelPoint3) modelPoint3 {
	return modelPoint3{
		x: matrix[0]*point.x + matrix[4]*point.y + matrix[8]*point.z + matrix[12],
		y: matrix[1]*point.x + matrix[5]*point.y + matrix[9]*point.z + matrix[13],
		z: matrix[2]*point.x + matrix[6]*point.y + matrix[10]*point.z + matrix[14],
	}
}

func rsmFaceTexture(rsm *res.RSM, node *res.RSMNode, face res.RSMFace) (string, [3]texturePoint) {
	textureName := ""
	if int(face.TextureID) < len(node.TextureRefs) {
		textureIndex := node.TextureRefs[face.TextureID].Index
		if textureIndex >= 0 && int(textureIndex) < len(rsm.Textures) {
			textureName = rsm.Textures[textureIndex]
		}
	}
	var uvs [3]texturePoint
	for i, tvertIndex := range face.TextureVertexIndices {
		if int(tvertIndex) >= len(node.TextureVertices) {
			continue
		}
		tvert := node.TextureVertices[tvertIndex]
		uvs[i] = texturePoint{u: tvert.U, v: tvert.V}
	}
	return textureName, uvs
}

func rsmFaceColor(rsm *res.RSM, textureName string, a, b, c modelPoint3, lighting sceneLighting) color.RGBA {
	base := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if textureName == "" {
		base = textureColor(textureName)
	}
	normal := modelPoint3{y: -1}
	if rsm == nil || rsm.ShadeType != 0 {
		normal = normalize3(cross3(sub3(b, a), sub3(c, a)))
	}
	scale := lighting.modelScaleNormalized(normal)
	return color.RGBA{
		R: clampColor(float64(base.R) * scale.x),
		G: clampColor(float64(base.G) * scale.y),
		B: clampColor(float64(base.B) * scale.z),
		A: clampColor(float64(rsmAlpha(rsm)) * 255),
	}
}

func rsmAlpha(rsm *res.RSM) float32 {
	if rsm == nil || rsm.Alpha <= 0 {
		return 1
	}
	if rsm.Alpha > 1 {
		return 1
	}
	return rsm.Alpha
}

func sub3(a, b modelPoint3) modelPoint3 {
	return modelPoint3{x: a.x - b.x, y: a.y - b.y, z: a.z - b.z}
}

func add3(a, b modelPoint3) modelPoint3 {
	return modelPoint3{x: a.x + b.x, y: a.y + b.y, z: a.z + b.z}
}

func mul3(v modelPoint3, scalar float64) modelPoint3 {
	return modelPoint3{x: v.x * scalar, y: v.y * scalar, z: v.z * scalar}
}

func cross3(a, b modelPoint3) modelPoint3 {
	return modelPoint3{
		x: a.y*b.z - a.z*b.y,
		y: a.z*b.x - a.x*b.z,
		z: a.x*b.y - a.y*b.x,
	}
}

func normalize3(v modelPoint3) modelPoint3 {
	length := math.Sqrt(v.x*v.x + v.y*v.y + v.z*v.z)
	if length == 0 {
		return modelPoint3{}
	}
	return modelPoint3{x: v.x / length, y: v.y / length, z: v.z / length}
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func rsmRenderRadius() float64 {
	return defaultRSMRenderRadius
}
