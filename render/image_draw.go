package render

import "math"

func (i *Image) DrawImage(src *Image, opts *DrawImageOptions) {
	if i == nil || i.pix == nil || src == nil || src.pix == nil {
		return
	}
	var o DrawImageOptions
	if opts != nil {
		o = *opts
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	corners := [4][2]float64{{0, 0}, {float64(w), 0}, {0, float64(h)}, {float64(w), float64(h)}}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, corner := range corners {
		x, y := o.GeoM.apply(corner[0], corner[1])
		minX, minY = math.Min(minX, x), math.Min(minY, y)
		maxX, maxY = math.Max(maxX, x), math.Max(maxY, y)
	}
	inv, ok := o.GeoM.invert()
	if !ok {
		return
	}
	r, g, b, a := o.ColorScale.rgba()
	dstBounds := i.pix.Bounds()
	x0 := clampInt(int(math.Floor(minX)), dstBounds.Min.X, dstBounds.Max.X)
	y0 := clampInt(int(math.Floor(minY)), dstBounds.Min.Y, dstBounds.Max.Y)
	x1 := clampInt(int(math.Ceil(maxX)), dstBounds.Min.X, dstBounds.Max.X)
	y1 := clampInt(int(math.Ceil(maxY)), dstBounds.Min.Y, dstBounds.Max.Y)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			sx, sy := inv.apply(float64(x)+0.5, float64(y)+0.5)
			if sx < 0 || sy < 0 || sx >= float64(w) || sy >= float64(h) {
				continue
			}
			sc := src.sample(float32(sx), float32(sy), o.Filter, AddressClampToZero)
			sc.R, sc.G, sc.B, sc.A = scaleByte(sc.R, r), scaleByte(sc.G, g), scaleByte(sc.B, b), scaleByte(sc.A, a)
			i.blendPixel(x, y, sc, o.Blend)
		}
	}
	i.modified()
}

func (i *Image) DrawTriangles(vertices []Vertex, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if i == nil || i.pix == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	for n := 0; n+2 < len(indices); n += 3 {
		i.drawTriangle(vertices[indices[n]], vertices[indices[n+1]], vertices[indices[n+2]], texture, o)
	}
	i.modified()
}

func (i *Image) DrawTrianglesOwned(vertices []Vertex, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if i == nil || i.pix == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	i.DrawTriangles(vertices, indices, texture, &o)
}

func (i *Image) DrawTriangles3D(vertices []Vertex3D, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if i == nil || i.pix == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	vertices2D := make([]Vertex, len(vertices))
	for n, vertex := range vertices {
		vertices2D[n] = Vertex{
			DstX:   vertex.X,
			DstY:   vertex.Y,
			SrcX:   vertex.SrcX,
			SrcY:   vertex.SrcY,
			ColorR: vertex.ColorR,
			ColorG: vertex.ColorG,
			ColorB: vertex.ColorB,
			ColorA: vertex.ColorA,
		}
	}
	i.DrawTriangles(vertices2D, indices, texture, &o)
}

func (i *Image) DrawTriangles3DOwned(vertices []Vertex3D, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if i == nil || i.pix == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	i.DrawTriangles3D(vertices, indices, texture, &o)
}
