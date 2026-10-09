package render

import (
	"testing"

	"github.com/gogpu/wgpu"
)

func TestSpriteFilteringSwitchesSamplerWithoutAffectingExplicitFilters(t *testing.T) {
	// Cached sentinels exercise sampler selection without a GPU. A miss would
	// attempt to allocate on the nil device, rather than silently passing.
	linear, nearest := &wgpu.Sampler{}, &wgpu.Sampler{}
	r := &gpuRenderer{samplers: map[samplerKey]*wgpu.Sampler{
		{filter: FilterLinear, address: AddressClampToZero, anisotropy: 1}:  linear,
		{filter: FilterNearest, address: AddressClampToZero, anisotropy: 1}: nearest,
	}}
	for _, smooth := range []bool{true, false, true} {
		r.smoothSprites = smooth
		for _, filter := range []Filter{FilterSprite, FilterLinear, FilterNearest} {
			want := linear
			if filter == FilterNearest || (filter == FilterSprite && !smooth) {
				want = nearest
			}
			got, err := r.sampler(DrawTrianglesOptions{Filter: filter, Address: AddressClampToZero}, NewImage(4, 4))
			if err != nil || got != want {
				t.Fatalf("smooth=%t filter=%d: sampler=%p want=%p error=%v", smooth, filter, got, want, err)
			}
		}
	}
}
