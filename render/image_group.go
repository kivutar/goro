package render

// ImageGroup owns images with a common lifetime, such as a map's textures.
// Like Image and Frame, it is used on the game/render thread.
type ImageGroup struct {
	images   []*Image
	released bool
}

// Add takes ownership of a newly created image. An image belongs to at most
// one group and must not be used after that group is released.
func (g *ImageGroup) Add(img *Image) *Image {
	if img == nil {
		return nil
	}
	if g.released || img.group != nil {
		panic("render: image group is released or image already owned")
	}
	img.group = g
	g.images = append(g.images, img)
	return img
}

// Release drops CPU pixels immediately. The renderer destroys uploaded
// textures and their bind groups before its next frame, on the GPU thread.
func (g *ImageGroup) Release() {
	if g == nil || g.released {
		return
	}
	g.released = true
	for _, img := range g.images {
		img.pix = nil
	}
}

func (r *gpuRenderer) releaseImageGroups() {
	for group := range r.imageGroups {
		if !group.released {
			continue
		}
		for _, img := range group.images {
			r.releaseImageTexture(img)
		}
		// Non-batched retained meshes also hold GPU buffers and image pointers.
		for mesh, cached := range r.worldMeshes {
			if mesh.texture.group == group || (mesh.lightTexture != nil && mesh.lightTexture.group == group) {
				if cached.vertexBuf != nil {
					cached.vertexBuf.Release()
				}
				if cached.indexBuf != nil {
					cached.indexBuf.Release()
				}
				delete(r.worldMeshes, mesh)
			}
		}
		delete(r.imageGroups, group)
	}
}
