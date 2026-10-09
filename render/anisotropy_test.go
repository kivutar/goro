package render

import (
	"image"
	"testing"
)

func TestAnisotropyOnlyAffectsLinearMipmappedTextures(t *testing.T) {
	mipmapped := NewImageWithMipmaps(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	for _, tt := range []struct {
		name   string
		image  *Image
		filter Filter
		level  int
		want   uint16
	}{
		{"terrain", mipmapped, FilterLinear, 8, 8},
		{"off", mipmapped, FilterLinear, 0, 1},
		{"unsupported", mipmapped, FilterLinear, 1, 1},
		{"nearest", mipmapped, FilterNearest, 8, 1},
		{"sprite", NewImage(4, 4), FilterLinear, 8, 1},
		{"lightmap sampler", nil, FilterLinear, 16, 1},
		{"clamp high", mipmapped, FilterLinear, 32, 16},
		{"clamp low", mipmapped, FilterLinear, -1, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key := imageSamplerKey(DrawTrianglesOptions{Filter: tt.filter, Address: AddressRepeat}, tt.image, tt.level)
			if key.anisotropy != tt.want || key.filter != tt.filter || key.address != AddressRepeat {
				t.Fatalf("unexpected sampler: %+v; want anisotropy %d", key, tt.want)
			}
		})
	}
}
