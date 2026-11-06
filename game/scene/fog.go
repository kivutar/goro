package scene

import (
	"image/color"
	"math"
)

type SceneFog struct {
	Enabled bool
	Near    float64
	Far     float64
	Color   color.RGBA
}

func (f SceneFog) MixColor(c color.RGBA, depth float64) color.RGBA {
	if !f.Enabled || f.Far <= f.Near || !IsFinite(depth) {
		return c
	}
	visibility := math.Max(0, math.Min(1, (f.Far-depth)/(f.Far-f.Near)))
	// The original stores visibility in the vertex's 8-bit specular alpha.
	visibility = math.Floor(visibility*255) / 255
	amount := 1 - visibility
	if amount <= 0 {
		return c
	}
	if amount >= 1 {
		return color.RGBA{R: f.Color.R, G: f.Color.G, B: f.Color.B, A: c.A}
	}
	inverse := 1 - amount
	return color.RGBA{
		R: ClampColor(float64(c.R)*inverse + float64(f.Color.R)*amount),
		G: ClampColor(float64(c.G)*inverse + float64(f.Color.G)*amount),
		B: ClampColor(float64(c.B)*inverse + float64(f.Color.B)*amount),
		A: c.A,
	}
}

func (f SceneFog) MixVertexTints(projection Projection, verts [4]Point3, tints [4]color.RGBA) [4]color.RGBA {
	if !f.Enabled {
		return tints
	}
	for i, vert := range verts {
		tints[i] = f.MixColor(tints[i], projection.Depth(vert.X, vert.Z, vert.Y))
	}
	return tints
}
