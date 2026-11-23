package scene

import (
	"image/color"
	"math"
)

// LightTint is the per-channel brightness the classic client applies to an
// actor's sprite from the ground lightmap under it (CRenderObject::SetLmIntensity).
type LightTint struct {
	R, G, B float64
}

// FullLight leaves sprite colors unchanged.
var FullLight = LightTint{R: 1, G: 1, B: 1}

// GrayLight scales every channel by the same factor.
func GrayLight(v float64) LightTint {
	return LightTint{R: v, G: v, B: v}
}

// Clamped limits every channel to [0, 1], treating NaN as dark and infinities as lit.
func (l LightTint) Clamped() LightTint {
	return LightTint{R: clampLightChannel(l.R), G: clampLightChannel(l.G), B: clampLightChannel(l.B)}
}

// Apply scales a tint's color channels, leaving alpha unchanged.
func (l LightTint) Apply(tint color.RGBA) color.RGBA {
	l = l.Clamped()
	return MultiplyTint(tint, l.R, l.G, l.B, 1)
}

func clampLightChannel(v float64) float64 {
	if v < 0 || math.IsNaN(v) {
		return 0
	}
	if v > 1 || math.IsInf(v, 0) {
		return 1
	}
	return v
}
