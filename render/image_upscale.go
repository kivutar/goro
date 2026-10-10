package render

import (
	"image"
	"image/color"
	"runtime"
	"sync"

	"github.com/virtualparadox/xbrscaler"
)

// UpscaleTexture2x reconstructs low-resolution texture edges with xBR. Run it
// during asset loading, before mipmap generation. Textures larger than 1024 on
// either axis stay unchanged to bound the memory and loading cost on mobile.
func UpscaleTexture2x(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 || w > 1024 || h > 1024 {
		return src
	}
	pixels := make([]uint32, w*h)
	for y := range h {
		for x := range w {
			c := color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			if c.A != 0 {
				// xBR's channel masks expect R in the low byte and A in the high byte.
				pixels[y*w+x] = uint32(c.R) | uint32(c.G)<<8 | uint32(c.B)<<16 | uint32(c.A)<<24
			}
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, w*2, h*2))
	scaleRows := func(first, last int) {
		// xBR reads two neighboring rows. Overlap the bands so dividing work
		// between cores produces exactly the same pixels as a single pass.
		top, bottom := max(0, first-2), min(h, last+2)
		rows := pixels[top*w : bottom*w]
		scaled, sw, _ := xbrscaler.NewXbrScaler(false).Xbr2x(&rows, w, bottom-top, true, true)
		for y := first * 2; y < last*2; y++ {
			for x := range sw {
				p := (*scaled)[(y-top*2)*sw+x]
				if p>>24 != 0 {
					i := y*out.Stride + x*4
					out.Pix[i+0] = byte(p)
					out.Pix[i+1] = byte(p >> 8)
					out.Pix[i+2] = byte(p >> 16)
					out.Pix[i+3] = byte(p >> 24)
				}
			}
		}
	}
	workers := min(4, runtime.GOMAXPROCS(0), max(1, h/32))
	if workers == 1 || w*h < 128*128 {
		scaleRows(0, h)
	} else {
		var work sync.WaitGroup
		for worker := range workers {
			work.Go(func() { scaleRows(h*worker/workers, h*(worker+1)/workers) })
		}
		work.Wait()
	}
	return out
}
