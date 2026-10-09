package render

import (
	"image"

	"golang.org/x/image/draw"
)

// NewImageWithMipmaps prepares a static texture for minification. Call it while
// loading assets, before uploading to the GPU. Drawing into it drops the mipmaps.
func NewImageWithMipmaps(src image.Image) *Image {
	out := NewImageFromStraightAlpha(src)
	for w, h := src.Bounds().Dx(), src.Bounds().Dy(); w > 1 || h > 1; {
		w, h = max(1, w/2), max(1, h/2)
		// Filter premultiplied colors so transparent pixels don't tint edges.
		level := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.BiLinear.Scale(level, level.Bounds(), src, src.Bounds(), draw.Src, nil)
		// The world shader uses straight-alpha blending, including at lower LODs.
		out.mipmaps = append(out.mipmaps, NewImageFromStraightAlpha(level).pix)
		src = level
	}
	return out
}

// ByteSize includes all pixels that must be uploaded, including mip levels.
func (i *Image) ByteSize() int {
	if i == nil || i.pix == nil {
		return 0
	}
	size := len(i.pix.Pix)
	for _, level := range i.mipmaps {
		size += len(level.Pix)
	}
	return size
}
