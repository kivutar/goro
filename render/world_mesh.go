package render

type WorldMesh struct {
	vertices     []Vertex3D
	indices      []uint16
	texture      *Image
	lightTexture *Image
	options      DrawTrianglesOptions
	version      uint64
}

type WorldMeshCommand struct {
	Mesh *WorldMesh
}

func NewWorldMeshWithLightmap(vertices []Vertex3D, indices []uint16, texture, lightTexture *Image, opts *DrawTrianglesOptions) *WorldMesh {
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	return &WorldMesh{
		vertices:     append([]Vertex3D(nil), vertices...),
		indices:      append([]uint16(nil), indices...),
		texture:      texture,
		lightTexture: lightTexture,
		options:      o,
		version:      1,
	}
}

func (m *WorldMesh) VertexCount() int {
	if m == nil {
		return 0
	}
	return len(m.vertices)
}

func (m *WorldMesh) IndexCount() int {
	if m == nil {
		return 0
	}
	return len(m.indices)
}
