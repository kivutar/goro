package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	mapTextureUploadBytes = 8 << 20
	mapTextureUploadCount = 32
)

func (m *WorldMode) ownMapImage(img *render.Image) *render.Image {
	if img == nil {
		return nil
	}
	if m.mapImages == nil {
		m.mapImages = &render.ImageGroup{}
	}
	return m.mapImages.Add(img)
}

func (m *WorldMode) releaseMapTextures() {
	m.mapImages.Release()
	m.mapImages = nil
	m.mapTextureUploads = nil
	m.mapUploadBatch = 0
}

// Leave also covers disconnects, character selection, and failed map loads.
func (m *WorldMode) Leave() {
	m.releaseMapTextures()
}

// Use the draw-time loaders, preserving resource lookup and transparency rules.
// Only the current map retains these images; no cross-map texture cache grows.
func (m *WorldMode) preloadMapTextures(ctx client.Context) {
	if ctx.Config.Headless || ctx.Resources == nil || ctx.World == nil {
		return
	}
	seen := make(map[*render.Image]bool)
	add := func(img *render.Image) {
		if img != nil && !seen[img] {
			seen[img] = true
			m.mapTextureUploads = append(m.mapTextureUploads, img)
		}
	}
	gnd, rsw := ctx.World.GND, ctx.World.RSW
	if gnd != nil {
		for _, name := range gnd.Textures {
			add(m.groundTexture(ctx.Resources, name))
		}
		atlas := buildGNDLightmapAtlas(gnd)
		add(m.ownMapImage(atlas.image))
		m.gndMeshCache = &gndRetainedMeshCache{gnd: gnd, rsw: rsw, lightmapAtlas: atlas, chunks: make(map[gndRetainedChunkKey][]retainedWorldMesh)}
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
				add(m.groundTexture(ctx.Resources, name))
			}
		}
	}
	if water, ok := mapWater(gnd, rsw); ok && gnd != nil {
		for _, cell := range gnd.Cells {
			if cell.Top >= 0 && waterVisibleForCell(cell, water) {
				for frame := 0; frame < 32; frame++ {
					add(m.waterTexture(ctx.Resources, int(water.Type), frame))
				}
				break
			}
		}
	}
}

// Keep the map covered until all requested uploads have been submitted, then
// render the usual two warmup frames. A skipped GPU frame retries the same batch.
func (m *WorldMode) prepareMapTextureUploads(screen *render.Frame) bool {
	m.mapUploadBatch = 0
	bytes := 0
	for _, img := range m.mapTextureUploads {
		size := len(img.RGBA().Pix)
		if m.mapUploadBatch > 0 && (bytes+size > mapTextureUploadBytes || m.mapUploadBatch >= mapTextureUploadCount) {
			break
		}
		screen.PrepareImage(img, groundTextureDrawOptions())
		m.mapUploadBatch++
		bytes += size
	}
	return m.mapUploadBatch > 0
}
