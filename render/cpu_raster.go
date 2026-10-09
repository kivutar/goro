package render

import (
	"image/color"
	"math"
)

func (i *Image) drawTriangle(v0, v1, v2 Vertex, texture *Image, opts DrawTrianglesOptions) {
	minX := int(math.Floor(float64(min3(v0.DstX, v1.DstX, v2.DstX))))
	minY := int(math.Floor(float64(min3(v0.DstY, v1.DstY, v2.DstY))))
	maxX := int(math.Ceil(float64(max3(v0.DstX, v1.DstX, v2.DstX))))
	maxY := int(math.Ceil(float64(max3(v0.DstY, v1.DstY, v2.DstY))))
	b := i.pix.Bounds()
	minX, minY = clampInt(minX, b.Min.X, b.Max.X), clampInt(minY, b.Min.Y, b.Max.Y)
	maxX, maxY = clampInt(maxX, b.Min.X, b.Max.X), clampInt(maxY, b.Min.Y, b.Max.Y)
	area := edge(v0.DstX, v0.DstY, v1.DstX, v1.DstY, v2.DstX, v2.DstY)
	if area == 0 {
		return
	}
	for y := minY; y < maxY; y++ {
		py := float32(y) + 0.5
		for x := minX; x < maxX; x++ {
			px := float32(x) + 0.5
			w0 := edge(v1.DstX, v1.DstY, v2.DstX, v2.DstY, px, py) / area
			w1 := edge(v2.DstX, v2.DstY, v0.DstX, v0.DstY, px, py) / area
			w2 := edge(v0.DstX, v0.DstY, v1.DstX, v1.DstY, px, py) / area
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			sx := w0*v0.SrcX + w1*v1.SrcX + w2*v2.SrcX
			sy := w0*v0.SrcY + w1*v1.SrcY + w2*v2.SrcY
			r := w0*v0.ColorR + w1*v1.ColorR + w2*v2.ColorR
			g := w0*v0.ColorG + w1*v1.ColorG + w2*v2.ColorG
			bb := w0*v0.ColorB + w1*v1.ColorB + w2*v2.ColorB
			a := w0*v0.ColorA + w1*v1.ColorA + w2*v2.ColorA
			sc := texture.sample(sx, sy, opts.Filter, opts.Address)
			sc.R, sc.G, sc.B, sc.A = scaleByte(sc.R, r), scaleByte(sc.G, g), scaleByte(sc.B, bb), scaleByte(sc.A, a)
			i.blendPixel(x, y, sc, opts.Blend)
		}
	}
}

func (i *Image) sample(x, y float32, filter Filter, address Address) color.RGBA {
	w, h := i.pix.Bounds().Dx(), i.pix.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return color.RGBA{}
	}
	x, y, ok := addressCoord(x, y, w, h, address)
	if !ok {
		return color.RGBA{}
	}
	if filter.resolveSprite(true) != FilterLinear {
		return i.rgbaAt(int(math.Floor(float64(x))), int(math.Floor(float64(y))))
	}
	x0, y0 := int(math.Floor(float64(x))), int(math.Floor(float64(y)))
	tx, ty := float64(x)-float64(x0), float64(y)-float64(y0)
	c00 := i.rgbaAt(x0, y0)
	c10 := i.rgbaAt(x0+1, y0)
	c01 := i.rgbaAt(x0, y0+1)
	c11 := i.rgbaAt(x0+1, y0+1)
	return color.RGBA{
		R: lerpByte(lerpByteF(c00.R, c10.R, tx), lerpByteF(c01.R, c11.R, tx), ty),
		G: lerpByte(lerpByteF(c00.G, c10.G, tx), lerpByteF(c01.G, c11.G, tx), ty),
		B: lerpByte(lerpByteF(c00.B, c10.B, tx), lerpByteF(c01.B, c11.B, tx), ty),
		A: lerpByte(lerpByteF(c00.A, c10.A, tx), lerpByteF(c01.A, c11.A, tx), ty),
	}
}

func (i *Image) rgbaAt(x, y int) color.RGBA {
	b := i.pix.Bounds()
	x = clampInt(x, 0, b.Dx()-1)
	y = clampInt(y, 0, b.Dy()-1)
	off := i.pix.PixOffset(x+b.Min.X, y+b.Min.Y)
	return color.RGBA{R: i.pix.Pix[off], G: i.pix.Pix[off+1], B: i.pix.Pix[off+2], A: i.pix.Pix[off+3]}
}

func (i *Image) blendPixel(x, y int, src color.RGBA, blend Blend) {
	if src.A == 0 {
		return
	}
	off := i.pix.PixOffset(x, y)
	if blend == BlendLighter {
		i.pix.Pix[off] = addByte(i.pix.Pix[off], alphaByte(src.R, src.A))
		i.pix.Pix[off+1] = addByte(i.pix.Pix[off+1], alphaByte(src.G, src.A))
		i.pix.Pix[off+2] = addByte(i.pix.Pix[off+2], alphaByte(src.B, src.A))
		i.pix.Pix[off+3] = addByte(i.pix.Pix[off+3], src.A)
		return
	}
	if blend == BlendSrcAlphaDstAlpha {
		dstA := i.pix.Pix[off+3]
		i.pix.Pix[off] = srcAlphaDstAlphaByte(src.R, src.A, i.pix.Pix[off], dstA)
		i.pix.Pix[off+1] = srcAlphaDstAlphaByte(src.G, src.A, i.pix.Pix[off+1], dstA)
		i.pix.Pix[off+2] = srcAlphaDstAlphaByte(src.B, src.A, i.pix.Pix[off+2], dstA)
		i.pix.Pix[off+3] = srcAlphaDstAlphaByte(src.A, src.A, dstA, dstA)
		return
	}
	a := uint32(src.A)
	ia := uint32(255 - src.A)
	dstA := uint32(i.pix.Pix[off+3])
	outA := a + dstA*ia/255
	if outA == 0 {
		i.pix.Pix[off] = 0
		i.pix.Pix[off+1] = 0
		i.pix.Pix[off+2] = 0
		i.pix.Pix[off+3] = 0
		return
	}
	i.pix.Pix[off] = straightSourceOverByte(src.R, src.A, i.pix.Pix[off], uint8(dstA), outA)
	i.pix.Pix[off+1] = straightSourceOverByte(src.G, src.A, i.pix.Pix[off+1], uint8(dstA), outA)
	i.pix.Pix[off+2] = straightSourceOverByte(src.B, src.A, i.pix.Pix[off+2], uint8(dstA), outA)
	i.pix.Pix[off+3] = byte(outA)
}

func addressCoord(x, y float32, w, h int, address Address) (float32, float32, bool) {
	if address == AddressRepeat {
		x = float32(math.Mod(float64(x), float64(w)))
		y = float32(math.Mod(float64(y), float64(h)))
		if x < 0 {
			x += float32(w)
		}
		if y < 0 {
			y += float32(h)
		}
		return x, y, true
	}
	if x < 0 || y < 0 || x >= float32(w) || y >= float32(h) {
		if address == AddressClampToZero {
			return 0, 0, false
		}
		x = float32(clampInt(int(x), 0, w-1))
		y = float32(clampInt(int(y), 0, h-1))
	}
	return x, y, true
}

func edge(ax, ay, bx, by, cx, cy float32) float32 {
	return (cx-ax)*(by-ay) - (cy-ay)*(bx-ax)
}

func scaleByte(v uint8, scale float32) uint8 {
	if scale <= 0 {
		return 0
	}
	out := int(float32(v)*scale + 0.5)
	if out > 255 {
		return 255
	}
	return uint8(out)
}

func addByte(a, b uint8) uint8 {
	sum := int(a) + int(b)
	if sum > 255 {
		return 255
	}
	return byte(sum)
}

func alphaByte(c, a uint8) uint8 {
	return uint8((uint32(c)*uint32(a) + 127) / 255)
}

func straightSourceOverByte(src, srcA, dst, dstA uint8, outA uint32) uint8 {
	srcPremul := uint32(src) * uint32(srcA)
	dstPremul := uint32(dst) * uint32(dstA) * uint32(255-srcA) / 255
	out := (srcPremul + dstPremul) / outA
	if out > 255 {
		return 255
	}
	return uint8(out)
}

func srcAlphaDstAlphaByte(src, srcA, dst, dstA uint8) uint8 {
	out := (uint32(src)*uint32(srcA) + uint32(dst)*uint32(dstA) + 127) / 255
	if out > 255 {
		return 255
	}
	return uint8(out)
}

func lerpByteF(a, b uint8, t float64) float64 {
	return float64(a) + (float64(b)-float64(a))*t
}

func lerpByte(a, b, t float64) uint8 {
	v := a + (b-a)*t
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func min3(a, b, c float32) float32 {
	return float32(math.Min(float64(a), math.Min(float64(b), float64(c))))
}

func max3(a, b, c float32) float32 {
	return float32(math.Max(float64(a), math.Max(float64(b), float64(c))))
}
