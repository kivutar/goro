package game

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

type groundGraffiti struct {
	x, y    int
	angle   float64
	message string
	mesh    *render.WorldMesh
	images  render.ImageGroup
}

func (m *WorldMode) applyGraffitiEntry(ctx client.Context, entry network.SkillUnitEntry) {
	m.removeGraffiti(entry.ID)
	if ctx.Config.Headless || entry.ID == 0 || !entry.Visible || entry.Message == "" {
		return
	}
	if m.graffiti == nil {
		m.graffiti = make(map[uint32]*groundGraffiti)
	}
	// CGraffiSkill uses the viewer's facing when the writing first appears.
	angle := degreesToRadians(float64(effectActorDirection(ctx, localSkillTarget(ctx), time.Now())*45 + 180))
	m.graffiti[entry.ID] = &groundGraffiti{x: int(entry.X), y: int(entry.Y), angle: angle, message: entry.Message}
}

func (m *WorldMode) removeGraffiti(id uint32) {
	if writing := m.graffiti[id]; writing != nil {
		writing.images.Release()
		delete(m.graffiti, id)
	}
}

func graffitiTexture(atlas image.Image, message string) *render.Image {
	// Sakexe's bitmap alphabet is laid out in 32-pixel cells, eight per row.
	// Its 384-pixel texture request is rounded to a 512-pixel allocation.
	out := image.NewRGBA(image.Rect(0, 0, 512, 512))
	x, y, gap := 0, (144-len(message))/24*36, 16
	for i := 0; i < len(message); i++ {
		c, index := message[i], -1
		switch {
		case c >= 'A' && c <= 'Z':
			index, gap = int(c-'A'), 32
		case c >= 'a' && c <= 'z':
			index, gap = 26+int(c-'a'), 32
		case c >= '0' && c <= '9':
			index, gap = 54+int(c-'0'), 32
		case c == '?':
			index, gap = 53, 16
		case c == '!':
			index, gap = 54, 16
		case c == ' ':
			end := x
			for j := i + 1; j < len(message) && message[j] != ' '; j++ {
				end += 32
			}
			if end >= 512 {
				x, y = 0, y+36
			}
		default:
			gap = 16
		}
		if x+gap >= 512 {
			x, y = 0, y+36
		}
		if index >= 0 {
			for dy := 0; dy < 32; dy++ {
				for dx := 0; dx < gap; dx++ {
					out.Set(x+dx, y+dy, atlas.At(index%8*32+dx*32/gap, index/8*32+dy))
				}
			}
		}
		x += gap
	}
	return render.NewImageFromImage(out)
}

func (m *WorldMode) drawGraffiti(screen *render.Frame, ctx client.Context, projection sceneProjection) {
	for _, writing := range m.graffiti {
		if math.Abs(float64(writing.x)-projection.playerX) > rsmRenderRadius()*2 ||
			math.Abs(float64(writing.y)-projection.playerY) > rsmRenderRadius()*2 {
			continue
		}
		if writing.mesh == nil {
			atlas := m.effectFileTexture(ctx.Resources, "effect/alpabet.bmp")
			if atlas == nil {
				continue
			}
			texture := writing.images.Add(graffitiTexture(atlas.RGBA(), writing.message))
			writing.mesh = graffitiMesh(ctx, writing, texture)
		}
		screen.DrawWorldMesh(writing.mesh)
	}
}

func graffitiMesh(ctx client.Context, writing *groundGraffiti, texture *render.Image) *render.WorldMesh {
	vertices := make([]render.Vertex3D, 0, 18*18*4)
	indices := make([]uint16, 0, 18*18*6)
	sin, cos := math.Sincos(writing.angle)
	// Follow the terrain one cell at a time, as CGraffiSkill::Render does.
	for y := 0; y < 18; y++ {
		for x := 0; x < 18; x++ {
			cx, cy := writing.x+x-9, writing.y+y-9
			if cx < 0 || cy < 0 {
				continue
			}
			cell, ok := ctx.World.GND.Cell(cx/2, cy/2)
			if !ok || cell.Top < 0 {
				continue
			}
			base := uint16(len(vertices))
			for _, offset := range [4][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
				dx, dy := float64(x+offset[0]-9), float64(y+offset[1]-9)
				wx, wy := float64(writing.x)+dx, float64(writing.y)+dy
				// Match the GND mesh's diagonal from corner 0 to corner 3.
				tx, ty := float64(cx%2+offset[0])/2, float64(cy%2+offset[1])/2
				height := float64(cell.Heights[0])*(1-max(tx, ty)) + float64(cell.Heights[3])*min(tx, ty)
				if tx > ty {
					height += float64(cell.Heights[1]) * (tx - ty)
				} else {
					height += float64(cell.Heights[2]) * (ty - tx)
				}
				uv := texturePoint{u: float32(1 - (dx*cos+dy*sin+9)/18), v: float32((-dx*sin + dy*cos + 9) / 18)}
				vertices = append(vertices, texturedSurfaceVertex3D(modelPoint3{x: wx, y: height + .01, z: wy}, uv, color.RGBA{255, 255, 255, 255}, 512, 512))
			}
			indices = append(indices, base, base+1, base+2, base, base+2, base+3)
		}
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	options.DepthBias = tileCursorDepthBias
	return render.NewWorldMeshWithLightmap(vertices, indices, texture, nil, options)
}
