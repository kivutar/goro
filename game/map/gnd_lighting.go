package maps

import (
	"image/color"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
)

// gndCellLight holds the lit vertex colours of a GND cell's surfaces before
// the environment colour, in the classic client's vertex order:
// top (x,y) (x+1,y) (x,y+1) (x+1,y+1); front and right as built by AssignGnd.
type gndCellLight struct {
	top, front, right [4]color.RGBA
}

type gndCellNormals struct {
	top          [2]scene.Point3
	front, right scene.Point3
}

// goroQuadOrder maps Goro's quad vertex order to the classic client's.
var goroQuadOrder = [4]int{0, 1, 3, 2}

func (l gndCellLight) topGoro() [4]color.RGBA   { return reorderGoroQuad(l.top) }
func (l gndCellLight) frontGoro() [4]color.RGBA { return reorderGoroQuad(l.front) }
func (l gndCellLight) rightGoro() [4]color.RGBA { return reorderGoroQuad(l.right) }

func reorderGoroQuad(colors [4]color.RGBA) [4]color.RGBA {
	var out [4]color.RGBA
	for i, k := range goroQuadOrder {
		out[i] = colors[k]
	}
	return out
}

func (m *System) gndCellLights(gnd *res.GND, rsw *res.RSW) []gndCellLight {
	if gnd == nil {
		return nil
	}
	if m.gndLightSource == gnd && m.gndLightRSW == rsw && len(m.gndLights) == gnd.Width*gnd.Height {
		return m.gndLights
	}
	m.gndLightSource = gnd
	m.gndLightRSW = rsw
	// Night only changes Env, so the baked colours use the day lighting.
	m.gndLights = buildGNDCellLights(gnd, sceneLightingFromRSW(rsw))
	return m.gndLights
}

func gndCellLightAt(lights []gndCellLight, gnd *res.GND, x, y int) gndCellLight {
	if gnd == nil || x < 0 || y < 0 || x >= gnd.Width || y >= gnd.Height || x+y*gnd.Width >= len(lights) {
		white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		quad := [4]color.RGBA{white, white, white, white}
		return gndCellLight{top: quad, front: quad, right: quad}
	}
	return lights[x+y*gnd.Width]
}

// buildGNDCellLights ports the vertex lighting of C3dGround15::AssignGnd.
// Heights are compared and normals computed in raw file units.
func buildGNDCellLights(gnd *res.GND, lighting Lighting) []gndCellLight {
	width, height := gnd.Width, gnd.Height
	if width <= 0 || height <= 0 {
		return nil
	}
	zoom := float64(gnd.Zoom)
	if zoom == 0 {
		zoom = 10
	}
	cellAt := func(x, y int) (res.GNDCell, bool) {
		if x < 0 || y < 0 || x >= width || y >= height {
			return res.GNDCell{}, false
		}
		return gnd.Cell(x, y)
	}
	hasSurface := func(id int) bool {
		_, ok := gnd.Surface(id)
		return id >= 0 && ok
	}
	raw := func(cell res.GNDCell, k int) float64 { return -float64(cell.Heights[k]) * 5 }

	normals := make([]gndCellNormals, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell, _ := cellAt(x, y)
			normal := &normals[x+y*width]
			if hasSurface(cell.Top) {
				v0 := scene.Point3{Y: raw(cell, 0)}
				v1 := scene.Point3{X: zoom, Y: raw(cell, 1)}
				v2 := scene.Point3{Y: raw(cell, 2), Z: zoom}
				v3 := scene.Point3{X: zoom, Y: raw(cell, 3), Z: zoom}
				normal.top[0] = scene.Normalize3(originalCross(scene.Sub3(v2, v1), scene.Sub3(v2, v0)))
				normal.top[1] = scene.Normalize3(originalCross(scene.Sub3(v3, v1), scene.Sub3(v3, v2)))
			}
			if below, ok := cellAt(x, y+1); ok && hasSurface(cell.Front) {
				normal.front = scene.Point3{Z: 1}
				if raw(cell, 2) < raw(below, 0) {
					normal.front = scene.Point3{Z: -1}
				}
			}
			if right, ok := cellAt(x+1, y); ok && hasSurface(cell.Right) {
				// The classic client compares height 2 with the right cell's
				// height 0 here, not the shared edge.
				normal.right = scene.Point3{X: 1}
				if raw(cell, 2) < raw(right, 0) {
					normal.right = scene.Point3{X: -1}
				}
			}
		}
	}

	type neighbour struct {
		cell   res.GNDCell
		normal gndCellNormals
		ok     bool
	}
	neighbourAt := func(x, y int) neighbour {
		cell, ok := cellAt(x, y)
		if !ok {
			return neighbour{}
		}
		return neighbour{cell: cell, normal: normals[x+y*width], ok: true}
	}
	topColor := func(n neighbour) color.RGBA {
		if !n.ok {
			return color.RGBA{}
		}
		surface, ok := gnd.Surface(n.cell.Top)
		if n.cell.Top < 0 || !ok {
			return color.RGBA{}
		}
		return surface.Color
	}
	shade := func(base color.RGBA, light scene.Point3) color.RGBA {
		return color.RGBA{
			R: uint8(float64(base.R) * light.X),
			G: uint8(float64(base.G) * light.Y),
			B: uint8(float64(base.B) * light.Z),
			A: 255,
		}
	}

	lights := make([]gndCellLight, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var n [9]neighbour
			n[0] = neighbourAt(x, y)
			n[1] = neighbourAt(x-1, y-1)
			n[2] = neighbourAt(x, y-1)
			n[3] = neighbourAt(x+1, y-1)
			n[4] = neighbourAt(x-1, y)
			n[5] = neighbourAt(x+1, y)
			n[6] = neighbourAt(x-1, y+1)
			n[7] = neighbourAt(x, y+1)
			n[8] = neighbourAt(x+1, y+1)
			self := n[0]
			h := self.cell.Heights
			var vn [4]scene.Point3

			vn[0] = self.normal.top[0]
			if n[2].ok && n[2].cell.Heights[2] == h[0] {
				vn[0] = scene.Add3(vn[0], scene.Add3(n[2].normal.top[0], n[2].normal.top[1]))
			}
			if n[4].ok && n[4].cell.Heights[1] == h[0] {
				vn[0] = scene.Add3(vn[0], scene.Add3(n[4].normal.top[0], n[4].normal.top[1]))
			}
			if n[1].ok && n[1].cell.Heights[3] == h[0] {
				vn[0] = scene.Add3(vn[0], n[1].normal.top[1])
			}

			vn[1] = scene.Add3(self.normal.top[0], self.normal.top[1])
			if n[5].ok && n[5].cell.Heights[0] == h[1] {
				vn[1] = scene.Add3(vn[1], n[5].normal.top[0])
			}
			if n[2].ok && n[2].cell.Heights[3] == h[1] {
				vn[1] = scene.Add3(vn[1], n[2].normal.top[1])
			}
			if n[3].ok && n[3].cell.Heights[2] == h[1] {
				vn[1] = scene.Add3(vn[1], scene.Add3(n[3].normal.top[0], n[3].normal.top[1]))
			}

			vn[2] = scene.Add3(self.normal.top[0], self.normal.top[1])
			if n[4].ok && n[4].cell.Heights[3] == h[2] {
				vn[2] = scene.Add3(vn[2], n[4].normal.top[1])
			}
			if n[7].ok && n[7].cell.Heights[0] == h[2] {
				vn[2] = scene.Add3(vn[2], n[7].normal.top[0])
			}
			if n[6].ok && n[6].cell.Heights[1] == h[2] {
				vn[2] = scene.Add3(vn[2], scene.Add3(n[6].normal.top[0], n[6].normal.top[1]))
			}

			// Vertex 3 starts from the first triangle's normal, as in the original.
			vn[3] = self.normal.top[0]
			if n[5].ok && n[5].cell.Heights[2] == h[3] {
				vn[3] = scene.Add3(vn[3], scene.Add3(n[5].normal.top[0], n[5].normal.top[1]))
			}
			if n[7].ok && n[7].cell.Heights[1] == h[3] {
				vn[3] = scene.Add3(vn[3], scene.Add3(n[7].normal.top[0], n[7].normal.top[1]))
			}
			if n[8].ok && n[8].cell.Heights[0] == h[3] {
				vn[3] = scene.Add3(vn[3], n[8].normal.top[0])
			}

			light := &lights[x+y*width]
			// Each top vertex takes the surface colour of the cell it starts;
			// missing neighbours leave it black.
			topColors := [4]color.RGBA{topColor(n[0]), topColor(n[5]), topColor(n[7]), topColor(n[8])}
			for k := range light.top {
				light.top[k] = shade(topColors[k], lighting.litOriginal(scene.Normalize3(vn[k])))
			}
			frontLight := lighting.litOriginal(self.normal.front)
			frontColors := [4]color.RGBA{topColor(n[7]), topColor(n[8]), topColor(n[7]), topColor(n[8])}
			for k := range light.front {
				light.front[k] = shade(frontColors[k], frontLight)
			}
			rightLight := lighting.litOriginal(self.normal.right)
			rightColors := [4]color.RGBA{topColor(n[8]), topColor(n[5]), topColor(n[8]), topColor(n[5])}
			for k := range light.right {
				light.right[k] = shade(rightColors[k], rightLight)
			}
		}
	}
	return lights
}

// originalCross is vector3d::CrossProduct.
func originalCross(a, b scene.Point3) scene.Point3 {
	return scene.Point3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}
