package maps

import (
	"image/color"
	"math"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
)

// RSM shade types (C3dModelRes::m_shadeType).
const (
	rsmShadeNone   = 0
	rsmShadeFlat   = 1
	rsmShadeSmooth = 2
)

type rsmFaceLightKey struct {
	rsm   *res.RSM
	node  *res.RSMNode
	index int
	baseX float64
	baseY float64
}

// rsmPlacementFaceLights returns the per-corner colours of a node's faces,
// before the environment colour. The classic client computes them once, when
// the model is placed, from its first animation frame
// (C3dActor::UpdateVertexColor), so animated models keep them.
func (m *System) rsmPlacementFaceLights(rsm *res.RSM, node *res.RSMNode, context rsmPlacementContext, visible ModelPlacement, gnd *res.GND) [][3]color.RGBA {
	key := rsmFaceLightKey{rsm: rsm, node: node, index: visible.Index, baseX: visible.BaseX, baseY: visible.BaseY}
	if lights, ok := m.rsmFaceLights[key]; ok {
		return lights
	}
	if m.rsmFaceLights == nil {
		m.rsmFaceLights = make(map[rsmFaceLightKey][][3]color.RGBA)
	}
	if m.rsmNodeMatrices == nil {
		m.rsmNodeMatrices = make(map[*res.RSM]map[string]scene.Mat4)
	}
	nodeMatrices, ok := m.rsmNodeMatrices[rsm]
	if !ok {
		nodeMatrices = buildRSMNodeMatrices(rsm, 0)
		m.rsmNodeMatrices[rsm] = nodeMatrices
	}
	lights := buildRSMNodeFaceLights(rsm, node, nodeMatrices[node.Name], context, visible, gnd)
	m.rsmFaceLights[key] = lights
	return lights
}

// buildRSMNodeFaceLights ports C3dNode::UpdateVertexColor and
// C3dNode::UpdateDestVertColor.
func buildRSMNodeFaceLights(rsm *res.RSM, node *res.RSMNode, nodeMatrix scene.Mat4, context rsmPlacementContext, visible ModelPlacement, gnd *res.GND) [][3]color.RGBA {
	if node == nil || len(node.Faces) == 0 {
		return nil
	}
	lighting := context.lighting
	// The classic client bakes the node's offset matrix into its vertices and
	// derives normals from them; the remaining node, parent and placement
	// transforms then rotate those normals.
	offset := scene.Mat4FromMat3(node.Matrix)
	outer := rsmNodeOuterMatrix(rsm, node, nodeMatrix, context.instance)
	model := rsmNodeModelMatrix(rsm, node, nodeMatrix, context.instance)
	baked := make([]scene.Point3, len(node.Vertices))
	for i, vertex := range node.Vertices {
		baked[i] = scene.Mat4TransformVector(offset, vectorFromRSM(vertex))
	}
	validFace := func(face res.RSMFace) bool {
		return int(face.VertexIndices[0]) < len(baked) && int(face.VertexIndices[1]) < len(baked) && int(face.VertexIndices[2]) < len(baked)
	}

	faceNormals := make([]scene.Point3, len(node.Faces))
	for i, face := range node.Faces {
		if !validFace(face) {
			continue
		}
		v0, v1, v2 := baked[face.VertexIndices[0]], baked[face.VertexIndices[1]], baked[face.VertexIndices[2]]
		faceNormals[i] = scene.Normalize3(originalCross(scene.Sub3(v2, v1), scene.Sub3(v2, v0)))
	}
	var vertexNormals map[int32][]scene.Point3
	if rsm.ShadeType == rsmShadeSmooth {
		vertexNormals = rsmSmoothVertexNormals(node, faceNormals, len(baked), validFace)
	}
	worldNormal := func(normal scene.Point3) scene.Point3 {
		return scene.Normalize3(scene.Mat4TransformVector(outer, normal))
	}

	alpha := scene.ClampColor(float64(rsmAlpha(rsm)) * 255)
	lights := make([][3]color.RGBA, len(node.Faces))
	for i, face := range node.Faces {
		if !validFace(face) {
			continue
		}
		var corner [3]scene.Point3
		switch rsm.ShadeType {
		case rsmShadeFlat:
			lit := lighting.lit(worldNormal(faceNormals[i]))
			corner = [3]scene.Point3{lit, lit, lit}
		case rsmShadeSmooth:
			normals := vertexNormals[face.SmoothGroup]
			for k := range corner {
				corner[k] = lighting.lit(worldNormal(normals[face.VertexIndices[k]]))
			}
		default:
			corner = [3]scene.Point3{{X: 1, Y: 1, Z: 1}, {X: 1, Y: 1, Z: 1}, {X: 1, Y: 1, Z: 1}}
		}
		for k := range corner {
			scale := 255.0
			if rsm.ShadeType == rsmShadeFlat || rsm.ShadeType == rsmShadeSmooth {
				if rsmTextureVertexHasColor(node, face.TextureVertexIndices[k]) {
					scale = 128
				}
			}
			world := scene.Mat4TransformPoint(model, vectorFromRSM(node.Vertices[face.VertexIndices[k]]))
			bright := gndBrightObjColor(gnd, world, visible)
			lights[i][k] = color.RGBA{
				R: uint8(math.Min(float64(uint8(corner[k].X*scale))+float64(bright.R), 255)),
				G: uint8(math.Min(float64(uint8(corner[k].Y*scale))+float64(bright.G), 255)),
				B: uint8(math.Min(float64(uint8(corner[k].Z*scale))+float64(bright.B), 255)),
				A: alpha,
			}
		}
	}
	return lights
}

// rsmSmoothVertexNormals averages face normals per smoothing group and vertex
// (C3dMesh::UpdateNormal).
func rsmSmoothVertexNormals(node *res.RSMNode, faceNormals []scene.Point3, vertexCount int, validFace func(res.RSMFace) bool) map[int32][]scene.Point3 {
	groups := make(map[int32][]scene.Point3)
	for i, face := range node.Faces {
		if !validFace(face) {
			continue
		}
		normals := groups[face.SmoothGroup]
		if normals == nil {
			normals = make([]scene.Point3, vertexCount)
			groups[face.SmoothGroup] = normals
		}
		for _, index := range face.VertexIndices {
			normals[index] = scene.Add3(normals[index], faceNormals[i])
		}
	}
	for _, normals := range groups {
		for i := range normals {
			normals[i] = scene.Normalize3(normals[i])
		}
	}
	return groups
}

func rsmTextureVertexHasColor(node *res.RSMNode, index uint16) bool {
	if int(index) >= len(node.TextureVertices) {
		return false
	}
	return node.TextureVertices[index].Color != [4]float32{}
}

// gndBrightObjColor is the ground cell colour the classic client adds to model
// vertices (C3dGround15::GetBrightObjVal): twice the centre lightmap colour of
// the cell's top surface, or white outside the map or without a top surface.
// The original samples at the vertex's world position plus the model's
// position again; that offset is reproduced.
func gndBrightObjColor(gnd *res.GND, world scene.Point3, visible ModelPlacement) color.RGBA {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if gnd == nil || len(gnd.Lightmaps) == 0 {
		return color.RGBA{}
	}
	halfWidth, halfHeight := gnd.Width/2, gnd.Height/2
	// Goro world units are a fifth of the original's and cells are ten
	// original units wide; RSW positions are already divided by five.
	x := int(math.Floor((world.X-float64(2*halfWidth))/2+float64(visible.Model.Position.X)/2)) + halfWidth
	y := int(math.Floor((world.Z-float64(2*halfHeight))/2+float64(visible.Model.Position.Z)/2)) + halfHeight
	if x < 0 || y < 0 || x >= gnd.Width || y >= gnd.Height {
		return white
	}
	cell, ok := gnd.Cell(x, y)
	if !ok || cell.Top < 0 {
		return white
	}
	surface, ok := gnd.Surface(cell.Top)
	if !ok {
		return white
	}
	lightmap, ok := gnd.Lightmap(surface.LightmapID)
	if !ok {
		return white
	}
	c := lightmap.Color[4][4]
	double := func(v uint8) uint8 { return uint8(math.Min(float64(v)*2, 255)) }
	return color.RGBA{R: double(c.R), G: double(c.G), B: double(c.B), A: 255}
}
