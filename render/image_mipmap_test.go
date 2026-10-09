package render

import (
	"image"
	"image/color"
	"testing"
)

func TestMipmapsDimensionsAndSize(t *testing.T) {
	for _, size := range []image.Point{{8, 8}, {7, 3}, {1, 9}, {9, 1}, {1, 1}} {
		t.Run(size.String(), func(t *testing.T) {
			// Also exercise sources with a nonzero origin.
			src := image.NewNRGBA(image.Rectangle{Min: image.Pt(3, 5), Max: image.Pt(3+size.X, 5+size.Y)})
			img := NewImageWithMipmaps(src)
			w, h, bytes := size.X, size.Y, size.X*size.Y*4
			for n, mip := range img.mipmaps {
				if w == 1 && h == 1 {
					t.Fatal("extra mip level after 1x1")
				}
				w, h = max(1, w/2), max(1, h/2)
				if mip.Bounds() != image.Rect(0, 0, w, h) {
					t.Fatalf("mip %d: bounds %v, want %dx%d", n+1, mip.Bounds(), w, h)
				}
				bytes += w * h * 4
			}
			if w != 1 || h != 1 || img.ByteSize() != bytes {
				t.Fatalf("incomplete chain or wrong upload size: last=%dx%d bytes=%d want=%d", w, h, img.ByteSize(), bytes)
			}
		})
	}
}

func TestMipmapsFilterDetailAndTransparentEdges(t *testing.T) {
	for _, transparent := range []bool{false, true} {
		src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				c := color.NRGBA{G: 255, A: 255}
				if (x+y)%2 == 0 {
					c = color.NRGBA{A: 255}
					if transparent {
						c = color.NRGBA{R: 255, B: 255} // Invisible color must not bleed.
					}
				}
				src.SetNRGBA(x, y, c)
			}
		}
		img := NewImageWithMipmaps(src)
		last := img.mipmaps[len(img.mipmaps)-1].RGBAAt(0, 0)
		if last.R != 0 || last.B != 0 {
			t.Fatalf("transparent=%t: edge color bled into mip: %v", transparent, last)
		}
		if transparent {
			if last.G < 253 || last.A < 126 || last.A > 129 {
				t.Fatalf("straight-alpha green edge became dark or lost coverage: %v", last)
			}
		} else if last.G < 126 || last.G > 129 || last.A != 255 {
			t.Fatalf("fine checkerboard did not average to half brightness: %v", last)
		}
	}
}

func TestModifyingImageDropsStaleMipmaps(t *testing.T) {
	for name, modify := range map[string]func(*Image){
		"fill": func(img *Image) { img.Fill(color.White) },
		"draw": func(img *Image) { img.DrawImage(WhiteImage(), nil) },
		"triangles": func(img *Image) {
			img.DrawTriangles([]Vertex{{}, {DstX: 2}, {DstY: 2}}, []uint16{0, 1, 2}, WhiteImage(), nil)
		},
		"update": func(img *Image) { updateRGBAImage(image.NewRGBA(image.Rect(0, 0, 4, 4)), img) },
	} {
		t.Run(name, func(t *testing.T) {
			img := NewImageWithMipmaps(image.NewRGBA(image.Rect(0, 0, 4, 4)))
			version := img.version
			modify(img)
			if img.version == version || len(img.mipmaps) != 0 || img.ByteSize() != 4*4*4 {
				t.Fatal("modified image retained stale GPU pixels or mip levels")
			}
		})
	}
}
