package maps

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	defaultRSMRenderRadius = 42
	rsmPlacementCellSize   = 64.0
)

type modelWorldTriangle struct {
	verts       [3]scene.Point3
	uvs         [3]scene.UV
	colors      [3]color.RGBA
	textureName string
}

type rsmFaceMeta struct {
	uvs         [3]scene.UV
	textureName string
}

type modelBounds struct {
	min scene.Point3
	max scene.Point3
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

func (m *System) DrawRSMModels(screen *render.Frame, manager *res.Manager, rsw *res.RSW, models map[string]*res.RSM, gnd *res.GND, projection scene.Projection, now time.Time) {
	for _, placement := range m.visibleRSMPlacements(rsw, gnd, projection) {
		rsm := m.rsmModelForPlacement(manager, models, placement.Model)
		if rsm == nil {
			continue
		}
		frame, animated := RSMAnimationFrame(rsm, placement.Model, now)
		if animated {
			m.DrawAnimatedRSMPlacement(screen, manager, rsw, gnd, rsm, placement, frame)
			continue
		}
		meshes := m.rsmMeshesForPlacement(manager, rsm, rsw, gnd, placement)
		for _, mesh := range meshes {
			screen.DrawWorldMesh(mesh.mesh)
		}
	}
}

type ModelPlacement struct {
	Index int
	Model res.RSWModel
	BaseX float64
	BaseY float64
}

type rsmPlacementGrid struct {
	rsw   *res.RSW
	gnd   *res.GND
	cells map[rsmPlacementGridCell][]ModelPlacement
}

type rsmPlacementGridCell struct {
	x int
	y int
}

type animatedRSMNodeCache struct {
	frame    int
	matrices map[string]scene.Mat4
}

func (m *System) visibleRSMPlacements(rsw *res.RSW, gnd *res.GND, projection scene.Projection) []ModelPlacement {
	if rsw == nil || gnd == nil {
		return nil
	}
	radius := RSMRenderRadius()
	grid := m.rsmPlacementGridFor(rsw, gnd)
	if grid == nil {
		return nil
	}
	queryRadius := radius * 2
	minCellX := rsmPlacementGridCoord(projection.PlayerX - queryRadius)
	maxCellX := rsmPlacementGridCoord(projection.PlayerX + queryRadius)
	minCellY := rsmPlacementGridCoord(projection.PlayerY - queryRadius)
	maxCellY := rsmPlacementGridCoord(projection.PlayerY + queryRadius)
	visible := make([]ModelPlacement, 0, 32)
	for cellY := minCellY; cellY <= maxCellY; cellY++ {
		for cellX := minCellX; cellX <= maxCellX; cellX++ {
			for _, placement := range grid.cells[rsmPlacementGridCell{x: cellX, y: cellY}] {
				dx := placement.BaseX - projection.PlayerX
				dy := placement.BaseY - projection.PlayerY
				if math.Abs(dx) > queryRadius || math.Abs(dy) > queryRadius {
					continue
				}
				visible = append(visible, placement)
			}
		}
	}
	return visible
}

func (m *System) rsmPlacementGridFor(rsw *res.RSW, gnd *res.GND) *rsmPlacementGrid {
	if rsw == nil || gnd == nil {
		return nil
	}
	if m.rsmPlacementGrid != nil && m.rsmPlacementGrid.rsw == rsw && m.rsmPlacementGrid.gnd == gnd {
		return m.rsmPlacementGrid
	}
	grid := &rsmPlacementGrid{
		rsw:   rsw,
		gnd:   gnd,
		cells: make(map[rsmPlacementGridCell][]ModelPlacement, len(rsw.Models)/4+1),
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
		grid.cells[cell] = append(grid.cells[cell], ModelPlacement{
			Index: index,
			Model: model,
			BaseX: baseX,
			BaseY: baseY,
		})
	}
	m.rsmPlacementGrid = grid
	return grid
}

func rsmPlacementGridCoord(value float64) int {
	return int(math.Floor(value / rsmPlacementCellSize))
}

func (m *System) rsmModelForPlacement(manager *res.Manager, models map[string]*res.RSM, placement res.RSWModel) *res.RSM {
	if placement.Filename == "" {
		return nil
	}
	rsm, ok := models[placement.Filename]
	if !ok {
		loaded, err := LoadRSMModel(manager, placement.Filename)
		if err == nil {
			rsm = loaded
		}
		models[placement.Filename] = rsm
	}
	return rsm
}

func (m *System) rsmMeshesForPlacement(manager *res.Manager, rsm *res.RSM, rsw *res.RSW, gnd *res.GND, visible ModelPlacement) []retainedWorldMesh {
	if m.rsmMeshCache == nil {
		m.rsmMeshCache = make(map[int][]retainedWorldMesh)
	}
	if rsm == nil {
		return nil
	}
	if meshes, ok := m.rsmMeshCache[visible.Index]; ok {
		return meshes
	}
	context, ok := m.rsmPlacementContext(rsm, rsw, visible)
	if !ok {
		return nil
	}
	if m.rsmNodeMatrices == nil {
		m.rsmNodeMatrices = make(map[*res.RSM]map[string]scene.Mat4)
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
		opts := *options
		opts.Environment = true
		key := retainedMeshKey{texture: texture, options: opts}
		builder := builders[key]
		if builder == nil {
			builder = &retainedMeshBuilder{texture: texture, options: opts}
			builders[key] = builder
			builderOrder = append(builderOrder, builder)
		}
		return builder
	}
	for _, nodeIndex := range context.nodeIndices {
		node := &rsm.Nodes[nodeIndex]
		faceLights := m.rsmPlacementFaceLights(rsm, node, context, visible, gnd)
		for _, worldTri := range buildRSMNodeWorldTriangles(rsm, node, nodeMatrices[node.Name], context.instance, m.rsmFaceMetas(rsm, node), faceLights) {
			texture := m.groundTexture(manager, worldTri.textureName)
			if texture != nil {
				bounds := texture.Bounds()
				w, h := float32(bounds.Dx()), float32(bounds.Dy())
				builder := builderFor(texture, rsmDrawOptionsForTriangle(render.FilterLinear, render.AddressClampToEdge, worldTri, visible.Index))
				if builder != nil {
					builder.addTriangle(
						scene.TexturedSurfaceVertex3D(worldTri.verts[0], worldTri.uvs[0], worldTri.colors[0], w, h),
						scene.TexturedSurfaceVertex3D(worldTri.verts[1], worldTri.uvs[1], worldTri.colors[1], w, h),
						scene.TexturedSurfaceVertex3D(worldTri.verts[2], worldTri.uvs[2], worldTri.colors[2], w, h),
					)
				}
				continue
			}
			if m.WhitePixel == nil {
				m.WhitePixel = render.NewImage(1, 1)
				m.WhitePixel.Fill(color.White)
			}
			builder := builderFor(m.WhitePixel, rsmDrawOptionsForTriangle(render.FilterNearest, render.AddressUnsafe, worldTri, visible.Index))
			if builder != nil {
				builder.addTriangle(
					scene.ColoredSurfaceVertex3D(worldTri.verts[0], 0, 0, worldTri.colors[0]),
					scene.ColoredSurfaceVertex3D(worldTri.verts[1], 1, 0, worldTri.colors[1]),
					scene.ColoredSurfaceVertex3D(worldTri.verts[2], 1, 1, worldTri.colors[2]),
				)
			}
		}
	}
	var meshes []retainedWorldMesh
	for _, builder := range builderOrder {
		builder.flush()
		meshes = append(meshes, builder.meshes...)
	}
	m.rsmMeshCache[visible.Index] = meshes
	return meshes
}

type rsmPlacementContext struct {
	nodeIndices []int
	instance    modelInstance
	lighting    Lighting
}

func (m *System) rsmPlacementContext(rsm *res.RSM, rsw *res.RSW, visible ModelPlacement) (rsmPlacementContext, bool) {
	placement := visible.Model
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
			matrix: buildRSMInstanceMatrix(rsm, placement, visible.BaseX, visible.BaseY, bounds.model),
		},
		lighting: m.Lighting(rsw),
	}, true
}

type animatedRSMScratch struct {
	batches    []scene.ModelDrawBatch
	worldVerts []scene.Point3
}

func (s *animatedRSMScratch) reset() {
	for i := range s.batches {
		s.batches[i].Clear()
	}
	s.batches = s.batches[:0]
	s.worldVerts = s.worldVerts[:0]
}

func (m *System) DrawAnimatedRSMPlacement(screen *render.Frame, manager *res.Manager, rsw *res.RSW, gnd *res.GND, rsm *res.RSM, visible ModelPlacement, frame int) {
	context, ok := m.rsmPlacementContext(rsm, rsw, visible)
	if !ok {
		return
	}
	nodeMatrices := m.animatedRSMNodeMatrices(rsm, frame)
	batchOrder := m.rsmAnimScratch.batches[:0]
	for _, nodeIndex := range context.nodeIndices {
		node := &rsm.Nodes[nodeIndex]
		faceLights := m.rsmPlacementFaceLights(rsm, node, context, visible, gnd)
		m.drawAnimatedRSMNodeTriangles(screen, manager, rsm, node, nodeMatrices[node.Name], context, visible, faceLights, &batchOrder)
	}
	for i := range batchOrder {
		batchOrder[i].Flush()
		batchOrder[i].Clear()
	}
	m.rsmAnimScratch.batches = batchOrder[:0]
}

func animatedRSMBatchFor(screen *render.Frame, batches *[]scene.ModelDrawBatch, texture *render.Image, options render.DrawTrianglesOptions) *scene.ModelDrawBatch {
	if texture == nil {
		return nil
	}
	// The environment colour scales model vertices at draw time (C3dNode::Render).
	options.Environment = true
	order := *batches
	for i := range order {
		if order[i].Texture == texture && order[i].Options == options {
			return &order[i]
		}
	}
	order = append(order, scene.ModelDrawBatch{Screen: screen, Texture: texture, Options: options})
	*batches = order
	return &order[len(order)-1]
}

func (m *System) drawAnimatedRSMNodeTriangles(screen *render.Frame, manager *res.Manager, rsm *res.RSM, node *res.RSMNode, nodeMatrix scene.Mat4, context rsmPlacementContext, visible ModelPlacement, faceLights [][3]color.RGBA, batchOrder *[]scene.ModelDrawBatch) {
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
		faceColors := rsmFaceColors(faceMeta.textureName, faceLights, faceIndex)
		texture := m.groundTexture(manager, faceMeta.textureName)
		if texture != nil {
			bounds := texture.Bounds()
			w, h := float32(bounds.Dx()), float32(bounds.Dy())
			options := rsmDrawOptionsForVertices(render.FilterLinear, render.AddressClampToEdge, a, b, c, visible.Index)
			batch := animatedRSMBatchFor(screen, batchOrder, texture, options)
			if batch != nil {
				batch.AddTriangle(
					scene.TexturedSurfaceVertex3D(a, faceMeta.uvs[0], faceColors[0], w, h),
					scene.TexturedSurfaceVertex3D(b, faceMeta.uvs[1], faceColors[1], w, h),
					scene.TexturedSurfaceVertex3D(c, faceMeta.uvs[2], faceColors[2], w, h),
				)
			}
			continue
		}
		if m.WhitePixel == nil {
			m.WhitePixel = render.NewImage(1, 1)
			m.WhitePixel.Fill(color.White)
		}
		options := rsmDrawOptionsForVertices(render.FilterNearest, render.AddressUnsafe, a, b, c, visible.Index)
		batch := animatedRSMBatchFor(screen, batchOrder, m.WhitePixel, options)
		if batch != nil {
			batch.AddTriangle(
				scene.ColoredSurfaceVertex3D(a, 0, 0, faceColors[0]),
				scene.ColoredSurfaceVertex3D(b, 1, 0, faceColors[1]),
				scene.ColoredSurfaceVertex3D(c, 1, 1, faceColors[2]),
			)
		}
	}
}

func (m *System) animatedRSMNodeWorldVerts(rsm *res.RSM, node *res.RSMNode, nodeMatrix scene.Mat4, instance modelInstance) []scene.Point3 {
	if node == nil || len(node.Vertices) == 0 {
		return nil
	}
	if cap(m.rsmAnimScratch.worldVerts) < len(node.Vertices) {
		m.rsmAnimScratch.worldVerts = make([]scene.Point3, len(node.Vertices))
	}
	worldVerts := m.rsmAnimScratch.worldVerts[:len(node.Vertices)]
	modelMatrix := rsmNodeModelMatrix(rsm, node, nodeMatrix, instance)
	for i, vertex := range node.Vertices {
		worldVerts[i] = scene.Mat4TransformPoint(modelMatrix, vectorFromRSM(vertex))
	}
	return worldVerts
}

func (m *System) animatedRSMNodeMatrices(rsm *res.RSM, frame int) map[string]scene.Mat4 {
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

func rsmDrawOptionsForVertices(filter render.Filter, address render.Address, a, b, c scene.Point3, placementIndex int) render.DrawTrianglesOptions {
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

func rsmTriangleVertsAreHorizontal(a, b, c scene.Point3) bool {
	normal := scene.Normalize3(scene.Cross3(scene.Sub3(b, a), scene.Sub3(c, a)))
	return math.Abs(normal.Y) >= 0.98
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
	matrix scene.Mat4
}

func (m *System) rsmFaceMetas(rsm *res.RSM, node *res.RSMNode) []rsmFaceMeta {
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

func buildRSMNodeWorldTriangles(rsm *res.RSM, node *res.RSMNode, nodeMatrix scene.Mat4, instance modelInstance, faceMetas []rsmFaceMeta, faceLights [][3]color.RGBA) []modelWorldTriangle {
	if len(node.Vertices) == 0 || len(node.Faces) == 0 {
		return nil
	}

	modelMatrix := rsmNodeModelMatrix(rsm, node, nodeMatrix, instance)
	worldVerts := make([]scene.Point3, len(node.Vertices))
	for i, vertex := range node.Vertices {
		world := scene.Mat4TransformPoint(modelMatrix, vectorFromRSM(vertex))
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
		triangles = append(triangles, modelWorldTriangle{
			verts:       [3]scene.Point3{a, b, c},
			uvs:         faceMeta.uvs,
			colors:      rsmFaceColors(faceMeta.textureName, faceLights, faceIndex),
			textureName: faceMeta.textureName,
		})
	}
	return triangles
}

func rsmNodeModelMatrix(rsm *res.RSM, node *res.RSMNode, nodeMatrix scene.Mat4, instance modelInstance) scene.Mat4 {
	localMatrix := rsmNodeOuterMatrix(nodeMatrix, instance)
	if len(rsm.Nodes) != 1 {
		localMatrix = scene.Mat4Translate(localMatrix, vectorFromRSM(node.Offset))
	}
	return scene.Mat4Multiply(localMatrix, scene.Mat4FromMat3(node.Matrix))
}

// rsmNodeOuterMatrix is the node's transform without its offset matrix, which
// the classic client bakes into the vertices.
func rsmNodeOuterMatrix(nodeMatrix scene.Mat4, instance modelInstance) scene.Mat4 {
	localMatrix := scene.Mat4Identity()
	localMatrix = scene.Mat4Translate(localMatrix, scene.Point3{
		X: -(instance.bounds.min.X + instance.bounds.max.X) * 0.5,
		Y: instance.bounds.max.Y,
		Z: -(instance.bounds.min.Z + instance.bounds.max.Z) * 0.5,
	})
	localMatrix = scene.Mat4Scale(localMatrix, scene.Point3{X: 1, Y: -1, Z: 1})
	localMatrix = scene.Mat4Multiply(localMatrix, nodeMatrix)
	return scene.Mat4Multiply(instance.matrix, localMatrix)
}

func buildRSMInstanceMatrix(rsm *res.RSM, placement res.RSWModel, baseX, baseY float64, bounds modelBounds) scene.Mat4 {
	matrix := scene.Mat4Identity()
	matrix = scene.Mat4Translate(matrix, scene.Point3{X: baseX, Y: float64(placement.Position.Y), Z: baseY})
	matrix = scene.Mat4RotateZ(matrix, scene.DegreesToRadians(float64(-placement.Rotation.Z)))
	matrix = scene.Mat4RotateX(matrix, scene.DegreesToRadians(float64(-placement.Rotation.X)))
	matrix = scene.Mat4RotateY(matrix, scene.DegreesToRadians(float64(placement.Rotation.Y)))
	matrix = scene.Mat4Scale(matrix, vectorFromRSW(placement.Scale))

	if main := rsmMainNode(rsm); main != nil && rsmVersionAtLeast(rsm, 2, 2) {
		matrix = scene.Mat4Scale(matrix, scene.Point3{X: 1, Y: -1, Z: 1})
		matrix = scene.Mat4Translate(matrix, vectorFromRSM(main.Offset))
		matrix = scene.Mat4Translate(matrix, scene.Point3{Y: (bounds.max.Y - bounds.min.Y) * 0.5})
		matrix = scene.Mat4Translate(matrix, scene.Point3{
			X: (bounds.max.X + bounds.min.X) * 0.5,
			Y: (bounds.max.Y + bounds.min.Y) * 0.5,
			Z: (bounds.max.Z + bounds.min.Z) * 0.5,
		})
	}
	return matrix
}

func buildRSMNodeMatrices(rsm *res.RSM, frame int) map[string]scene.Mat4 {
	nodes := make(map[string]*res.RSMNode, len(rsm.Nodes))
	for i := range rsm.Nodes {
		nodes[rsm.Nodes[i].Name] = &rsm.Nodes[i]
	}
	out := make(map[string]scene.Mat4, len(rsm.Nodes))
	visiting := make(map[string]bool, len(rsm.Nodes))
	var build func(*res.RSMNode) scene.Mat4
	build = func(node *res.RSMNode) scene.Mat4 {
		if matrix, ok := out[node.Name]; ok {
			return matrix
		}
		if visiting[node.Name] {
			return scene.Mat4Identity()
		}
		visiting[node.Name] = true
		matrix := scene.Mat4Identity()
		if parent := nodes[node.ParentName]; parent != nil && parent != node && node.ParentName != node.Name {
			matrix = build(parent)
		}
		matrix = scene.Mat4Translate(matrix, sampleRSMPosition(node, frame))
		if rotation, ok := sampleRSMRotation(node, frame); ok {
			matrix = scene.Mat4RotateQuat(matrix, rotation)
		} else {
			matrix = scene.Mat4RotateAxis(matrix, vectorFromRSM(node.RotationAxis), float64(node.RotationAngle))
		}
		matrix = scene.Mat4Scale(matrix, sampleRSMScale(node, frame))
		out[node.Name] = matrix
		visiting[node.Name] = false
		return matrix
	}
	for i := range rsm.Nodes {
		build(&rsm.Nodes[i])
	}
	return out
}

func RSMAnimationFrame(rsm *res.RSM, _ res.RSWModel, now time.Time) (int, bool) {
	if !rsmHasNodeAnimation(rsm) {
		return 0, false
	}
	// robr drives animated map models from the global millisecond tick. The RSW
	// animation speed is not applied here; the keyframe times are already in ms.
	frame := int(now.UnixMilli())
	animLen := RSMAnimationLength(rsm)
	if animLen > 0 {
		frame %= animLen
		if frame < 0 {
			frame += animLen
		}
	}
	return frame, true
}

func RSMAnimationLength(rsm *res.RSM) int {
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

func sampleRSMPosition(node *res.RSMNode, frame int) scene.Point3 {
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

func sampleRSMScale(node *res.RSMNode, frame int) scene.Point3 {
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

func lerpPoint3(a, b scene.Point3, t float64) scene.Point3 {
	return scene.Point3{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
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
			matrix = scene.Mat4Translate(matrix, vectorFromRSM(node.Offset))
		}
		matrix = scene.Mat4Multiply(matrix, scene.Mat4FromMat3(node.Matrix))

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

func transformedNodeBounds(node *res.RSMNode, matrix scene.Mat4) modelBounds {
	bounds := emptyModelBounds()
	for _, vertex := range node.Vertices {
		point := scene.Mat4TransformPoint(matrix, vectorFromRSM(vertex))
		bounds.min.X = math.Min(bounds.min.X, point.X)
		bounds.min.Y = math.Min(bounds.min.Y, point.Y)
		bounds.min.Z = math.Min(bounds.min.Z, point.Z)
		bounds.max.X = math.Max(bounds.max.X, point.X)
		bounds.max.Y = math.Max(bounds.max.Y, point.Y)
		bounds.max.Z = math.Max(bounds.max.Z, point.Z)
	}
	return bounds
}

func emptyModelBounds() modelBounds {
	return modelBounds{
		min: scene.Point3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)},
		max: scene.Point3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)},
	}
}

func (b modelBounds) empty() bool {
	return math.IsInf(b.min.X, 0)
}

func (b *modelBounds) include(other modelBounds) {
	if other.empty() {
		return
	}
	if b.empty() {
		*b = other
		return
	}
	b.min.X = math.Min(b.min.X, other.min.X)
	b.min.Y = math.Min(b.min.Y, other.min.Y)
	b.min.Z = math.Min(b.min.Z, other.min.Z)
	b.max.X = math.Max(b.max.X, other.max.X)
	b.max.Y = math.Max(b.max.Y, other.max.Y)
	b.max.Z = math.Max(b.max.Z, other.max.Z)
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

func vectorFromRSM(v res.RSMVector3) scene.Point3 {
	return scene.Point3{X: float64(v.X), Y: float64(v.Y), Z: float64(v.Z)}
}

func vectorFromRSW(v res.RSWVector3) scene.Point3 {
	return scene.Point3{X: float64(v.X), Y: float64(v.Y), Z: float64(v.Z)}
}

func rsmVersionAtLeast(rsm *res.RSM, major, minor byte) bool {
	return rsm != nil && (rsm.VersionMajor > major || rsm.VersionMajor == major && rsm.VersionMinor >= minor)
}

func rsmFaceTexture(rsm *res.RSM, node *res.RSMNode, face res.RSMFace) (string, [3]scene.UV) {
	textureName := ""
	if int(face.TextureID) < len(node.TextureRefs) {
		textureIndex := node.TextureRefs[face.TextureID].Index
		if textureIndex >= 0 && int(textureIndex) < len(rsm.Textures) {
			textureName = rsm.Textures[textureIndex]
		}
	}
	var uvs [3]scene.UV
	for i, tvertIndex := range face.TextureVertexIndices {
		if int(tvertIndex) >= len(node.TextureVertices) {
			continue
		}
		tvert := node.TextureVertices[tvertIndex]
		uvs[i] = scene.UV{U: tvert.U, V: tvert.V}
	}
	return textureName, uvs
}

// rsmFaceColors returns a face's baked corner colours; the environment colour
// is applied at draw time.
func rsmFaceColors(textureName string, faceLights [][3]color.RGBA, faceIndex int) [3]color.RGBA {
	var colors [3]color.RGBA
	if faceIndex >= len(faceLights) {
		return colors
	}
	for k, light := range faceLights[faceIndex] {
		if textureName == "" {
			// Placeholder colour for faces without a texture.
			base := textureColor(textureName)
			light.R = uint8(uint16(light.R) * uint16(base.R) / 255)
			light.G = uint8(uint16(light.G) * uint16(base.G) / 255)
			light.B = uint8(uint16(light.B) * uint16(base.B) / 255)
		}
		colors[k] = light
	}
	return colors
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

func RSMRenderRadius() float64 {
	return defaultRSMRenderRadius
}
