package render

import "math"

type Filter int

const (
	FilterNearest Filter = iota
	FilterLinear
	// FilterSprite follows the world sprite smoothing setting on the GPU.
	// CPU image composition uses linear sampling, independent of this setting.
	FilterSprite
)

func (f Filter) resolveSprite(smooth bool) Filter {
	if f != FilterSprite {
		return f
	}
	if smooth {
		return FilterLinear
	}
	return FilterNearest
}

type Address int

const (
	AddressUnsafe Address = iota
	AddressClampToZero
	AddressClampToEdge
	AddressRepeat
)

type Blend int

const (
	BlendSourceOver Blend = iota
	BlendLighter
	BlendSrcAlphaDstAlpha
)

type Vertex struct {
	DstX, DstY     float32
	SrcX, SrcY     float32
	ColorR, ColorG float32
	ColorB, ColorA float32
}

type Vertex3D struct {
	X, Y, Z                float32
	SrcX, SrcY             float32
	ColorR, ColorG         float32
	ColorB, ColorA         float32
	DepthX, DepthY, DepthZ float32
	LightSrcX, LightSrcY   float32
}

type Camera3D struct {
	Enabled        bool
	ViewProjection [16]float32
	Fog            Fog3D
}

type Fog3D struct {
	Enabled  bool
	Near     float32
	Far      float32
	Strength float32
	ColorR   float32
	ColorG   float32
	ColorB   float32
}

type DrawTrianglesOptions struct {
	Filter     Filter
	Address    Address
	Blend      Blend
	DepthTest  bool
	DepthWrite bool
	DepthBias  float32
	DisableFog bool
}

type DrawImageOptions struct {
	GeoM       GeoM
	ColorScale ColorScale
	Filter     Filter
	Blend      Blend
}

type GeoM struct {
	a, b, c float64
	d, e, f float64
	set     bool
}

func (g *GeoM) ensure() {
	if !g.set {
		g.a, g.e = 1, 1
		g.set = true
	}
}

func (g *GeoM) Translate(tx, ty float64) {
	g.ensure()
	g.c += tx
	g.f += ty
}

func (g *GeoM) Scale(sx, sy float64) {
	g.ensure()
	g.a *= sx
	g.b *= sx
	g.c *= sx
	g.d *= sy
	g.e *= sy
	g.f *= sy
}

func (g *GeoM) Rotate(theta float64) {
	g.ensure()
	s, c := math.Sin(theta), math.Cos(theta)
	a, b, cc := g.a, g.b, g.c
	d, e, f := g.d, g.e, g.f
	g.a = c*a - s*d
	g.b = c*b - s*e
	g.c = c*cc - s*f
	g.d = s*a + c*d
	g.e = s*b + c*e
	g.f = s*cc + c*f
}

func (g GeoM) apply(x, y float64) (float64, float64) {
	if !g.set {
		return x, y
	}
	return g.a*x + g.b*y + g.c, g.d*x + g.e*y + g.f
}

func (g GeoM) invert() (GeoM, bool) {
	if !g.set {
		return GeoM{a: 1, e: 1, set: true}, true
	}
	det := g.a*g.e - g.b*g.d
	if math.Abs(det) < 1e-12 {
		return GeoM{}, false
	}
	inv := 1 / det
	return GeoM{
		a:   g.e * inv,
		b:   -g.b * inv,
		c:   (g.b*g.f - g.e*g.c) * inv,
		d:   -g.d * inv,
		e:   g.a * inv,
		f:   (g.d*g.c - g.a*g.f) * inv,
		set: true,
	}, true
}

type ColorScale struct {
	r, g, b, a float32
	set        bool
}

func (c *ColorScale) ensure() {
	if !c.set {
		c.r, c.g, c.b, c.a = 1, 1, 1, 1
		c.set = true
	}
}

func (c *ColorScale) Scale(r, g, b, a float32) {
	c.ensure()
	c.r *= r
	c.g *= g
	c.b *= b
	c.a *= a
}

func (c *ColorScale) ScaleAlpha(a float32) {
	c.ensure()
	c.a *= a
}

func (c ColorScale) rgba() (float32, float32, float32, float32) {
	if !c.set {
		return 1, 1, 1, 1
	}
	return c.r, c.g, c.b, c.a
}
