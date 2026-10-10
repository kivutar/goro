package render

import (
	"bytes"
	"image"
	"image/color"
	"runtime"
	"testing"
)

func TestUpscaleTexturePreservesSolidColorsAndSubimageBounds(t *testing.T) {
	for _, c := range []color.NRGBA{{R: 211, G: 89, B: 37, A: 255}, {G: 187, A: 128}, {}} {
		for _, size := range []image.Point{{1, 1}, {1, 5}, {7, 1}, {7, 5}} {
			parent := image.NewNRGBA(image.Rect(0, 0, 20, 20))
			src := parent.SubImage(image.Rect(3, 4, 3+size.X, 4+size.Y)).(*image.NRGBA)
			for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
				for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
					src.SetNRGBA(x, y, c)
				}
			}
			out := UpscaleTexture2x(src)
			if out.Bounds() != image.Rect(0, 0, size.X*2, size.Y*2) {
				t.Fatalf("wrong bounds for %v: %v", size, out.Bounds())
			}
			for y := range size.Y * 2 {
				for x := range size.X * 2 {
					if got := color.NRGBAModel.Convert(out.At(x, y)).(color.NRGBA); got != c {
						t.Fatalf("solid color changed at %d,%d: %v, want %v", x, y, got, c)
					}
				}
			}
		}
	}
}

func TestUpscaleTextureSmoothsDiagonalWithoutTransparentColorBleed(t *testing.T) {
	for _, transparent := range []bool{false, true} {
		src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
		for y := range 8 {
			for x := range 8 {
				c := color.NRGBA{G: 255, A: 255}
				if x > y {
					c = color.NRGBA{A: 255}
					if transparent {
						c = color.NRGBA{R: 255, B: 255} // Invisible magenta must not bleed.
					}
				}
				src.SetNRGBA(x, y, c)
			}
		}
		out := UpscaleTexture2x(src).(*image.NRGBA)
		blended := false
		for y := range 16 {
			for x := range 16 {
				c := out.NRGBAAt(x, y)
				if c.R != 0 || c.B != 0 || transparent && c.A != 0 && c.G != 255 {
					t.Fatalf("edge acquired a color fringe: %v", c)
				}
				if transparent {
					blended = blended || c.A > 0 && c.A < 255
				} else {
					blended = blended || c.G > 0 && c.G < 255
				}
			}
		}
		if !blended {
			t.Fatalf("transparent=%t: diagonal is still nearest-neighbor scaled", transparent)
		}
		texture := NewImageWithMipmaps(out)
		if texture.Bounds().Dx() != 16 || len(texture.mipmaps) != 4 || texture.ByteSize() != 1364 {
			t.Fatal("upscaled image did not produce a complete, accounted-for mip chain")
		}
	}
}

func TestUpscaleTextureSkipsEmptyAndLargeImages(t *testing.T) {
	for _, size := range []image.Point{{0, 1}, {1, 0}, {1025, 1}, {1, 1025}} {
		src := image.NewNRGBA(image.Rectangle{Max: size})
		if UpscaleTexture2x(src) != src {
			t.Fatalf("unexpected upscale for %v", size)
		}
	}
}

func TestUpscaleTextureParallelBandsMatchSinglePass(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 129, 131))
	for y := range 131 {
		for x := range 129 {
			src.SetNRGBA(x, y, color.NRGBA{R: byte(x * 7), G: byte(y * 11), B: byte((x + y) * 13), A: byte(x * y)})
		}
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	serial := UpscaleTexture2x(src).(*image.NRGBA)
	runtime.GOMAXPROCS(4)
	parallel := UpscaleTexture2x(src).(*image.NRGBA)
	if !bytes.Equal(serial.Pix, parallel.Pix) {
		t.Fatal("parallel upscaling changed pixels at band boundaries")
	}
}
