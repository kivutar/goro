package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

type Filter int

const (
	FilterNearest Filter = iota
	FilterLinear
)

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
	Enabled bool
	Near    float32
	Far     float32
	ColorR  float32
	ColorG  float32
	ColorB  float32
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

type Image struct {
	pix             *image.RGBA
	screen          bool
	screenScaleX    float32
	screenScaleY    float32
	version         uint64
	commands        []DrawCommand
	worldCommands   []WorldCommand
	worldMeshes     []WorldMeshCommand
	worldBillboards []WorldBillboardCommand
	uiRects         []UIRectCommand
	uiTextBoxes     []UITextBoxCommand
	uiTextLabels    []UITextLabelCommand
	clear           color.RGBA
	camera          Camera3D
}

var whiteImage *Image

var quadIndices012213 = []uint16{0, 1, 2, 2, 1, 3}

func WhiteImage() *Image {
	if whiteImage == nil {
		whiteImage = NewImage(1, 1)
		whiteImage.Fill(color.White)
	}
	return whiteImage
}

type DrawCommand struct {
	Vertices []Vertex
	Indices  []uint16
	Texture  *Image
	Options  DrawTrianglesOptions
}

type WorldCommand struct {
	Vertices     []Vertex3D
	Indices      []uint16
	Texture      *Image
	LightTexture *Image
	Options      DrawTrianglesOptions
}

type WorldMesh struct {
	vertices     []Vertex3D
	indices      []uint16
	texture      *Image
	lightTexture *Image
	options      DrawTrianglesOptions
	version      uint64
}

type WorldMeshCommand struct {
	Mesh *WorldMesh
}

type WorldBillboardCommand struct {
	Texture     *Image
	Options     DrawTrianglesOptions
	Center      [3]float32
	RightAxis   [3]float32
	UpAxis      [3]float32
	DepthUpAxis [3]float32
	Width       float32
	Height      float32
	AnchorX     float32
	AnchorY     float32
	ColorR      float32
	ColorG      float32
	ColorB      float32
	ColorA      float32
	DepthBias   float32
}

type UITextLabelCommand struct {
	Text       string
	X          float64
	Y          float64
	Foreground color.RGBA
	Outline    color.RGBA
	Centered   bool
	Bold       bool
	Size       float32
}

type UIRectCommand struct {
	X, Y, W, H float64
	Color      color.RGBA
}

type UITextBoxAnchor int

const (
	UITextBoxAnchorTopLeft UITextBoxAnchor = iota
	UITextBoxAnchorBottomCenter
	UITextBoxAnchorTooltipCenter
)

type UITextBoxCommand struct {
	Text     string
	X        float64
	Y        float64
	AltY     float64
	Anchor   UITextBoxAnchor
	MaxWidth float64
	MaxLines int
}

func NewScreenImage(width, height int) *Image {
	img := NewImage(width, height)
	img.screen = true
	img.screenScaleX = 1
	img.screenScaleY = 1
	return img
}

func (i *Image) BeginFrame() {
	if i == nil {
		return
	}
	i.commands = i.commands[:0]
	i.worldCommands = i.worldCommands[:0]
	i.worldMeshes = i.worldMeshes[:0]
	i.worldBillboards = i.worldBillboards[:0]
	i.uiRects = i.uiRects[:0]
	i.uiTextBoxes = i.uiTextBoxes[:0]
	i.uiTextLabels = i.uiTextLabels[:0]
	i.camera = Camera3D{}
}

func (i *Image) clearUIOverlayCommands() {
	if i == nil {
		return
	}
	i.uiRects = i.uiRects[:0]
	i.uiTextBoxes = i.uiTextBoxes[:0]
	i.uiTextLabels = i.uiTextLabels[:0]
}

func (i *Image) SetScreenScale(x, y float32) {
	if i == nil || !i.screen {
		return
	}
	if x <= 0 || math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
		x = 1
	}
	if y <= 0 || math.IsNaN(float64(y)) || math.IsInf(float64(y), 0) {
		y = 1
	}
	i.screenScaleX = x
	i.screenScaleY = y
}

func (i *Image) SetCamera3D(camera Camera3D) {
	if i == nil {
		return
	}
	i.camera = camera
}

func NewImage(width, height int) *Image {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return &Image{pix: image.NewRGBA(image.Rect(0, 0, width, height))}
}

func NewImageFromImage(src image.Image) *Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return &Image{pix: dst}
}

func NewWorldMesh(vertices []Vertex3D, indices []uint16, texture *Image, opts *DrawTrianglesOptions) *WorldMesh {
	return NewWorldMeshWithLightmap(vertices, indices, texture, nil, opts)
}

func NewWorldMeshWithLightmap(vertices []Vertex3D, indices []uint16, texture, lightTexture *Image, opts *DrawTrianglesOptions) *WorldMesh {
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	return &WorldMesh{
		vertices:     append([]Vertex3D(nil), vertices...),
		indices:      append([]uint16(nil), indices...),
		texture:      texture,
		lightTexture: lightTexture,
		options:      o,
		version:      1,
	}
}

func (m *WorldMesh) VertexCount() int {
	if m == nil {
		return 0
	}
	return len(m.vertices)
}

func (m *WorldMesh) IndexCount() int {
	if m == nil {
		return 0
	}
	return len(m.indices)
}

func (i *Image) Bounds() image.Rectangle {
	if i == nil || i.pix == nil {
		return image.Rectangle{}
	}
	return i.pix.Bounds()
}

func (i *Image) RGBA() *image.RGBA {
	if i == nil {
		return nil
	}
	return i.pix
}

func (i *Image) Fill(c color.Color) {
	if i == nil || i.pix == nil {
		return
	}
	if i.screen {
		i.clear = color.RGBAModel.Convert(c).(color.RGBA)
		return
	}
	draw.Draw(i.pix, i.pix.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	i.version++
}

func (i *Image) DrawImage(src *Image, opts *DrawImageOptions) {
	if i == nil || i.pix == nil || src == nil || src.pix == nil {
		return
	}
	var o DrawImageOptions
	if opts != nil {
		o = *opts
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if i.screen {
		r, g, b, a := o.ColorScale.rgba()
		p0x, p0y := o.GeoM.apply(0, 0)
		p1x, p1y := o.GeoM.apply(float64(w), 0)
		p2x, p2y := o.GeoM.apply(0, float64(h))
		p3x, p3y := o.GeoM.apply(float64(w), float64(h))
		vertices := []Vertex{
			{DstX: float32(p0x), DstY: float32(p0y), SrcX: 0, SrcY: 0, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
			{DstX: float32(p1x), DstY: float32(p1y), SrcX: float32(w), SrcY: 0, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
			{DstX: float32(p2x), DstY: float32(p2y), SrcX: 0, SrcY: float32(h), ColorR: r, ColorG: g, ColorB: b, ColorA: a},
			{DstX: float32(p3x), DstY: float32(p3y), SrcX: float32(w), SrcY: float32(h), ColorR: r, ColorG: g, ColorB: b, ColorA: a},
		}
		i.DrawTrianglesOwned(vertices, quadIndices012213, src, &DrawTrianglesOptions{Filter: o.Filter, Address: AddressClampToZero, Blend: o.Blend})
		return
	}
	corners := [4][2]float64{{0, 0}, {float64(w), 0}, {0, float64(h)}, {float64(w), float64(h)}}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, corner := range corners {
		x, y := o.GeoM.apply(corner[0], corner[1])
		minX, minY = math.Min(minX, x), math.Min(minY, y)
		maxX, maxY = math.Max(maxX, x), math.Max(maxY, y)
	}
	inv, ok := o.GeoM.invert()
	if !ok {
		return
	}
	r, g, b, a := o.ColorScale.rgba()
	dstBounds := i.pix.Bounds()
	x0 := clampInt(int(math.Floor(minX)), dstBounds.Min.X, dstBounds.Max.X)
	y0 := clampInt(int(math.Floor(minY)), dstBounds.Min.Y, dstBounds.Max.Y)
	x1 := clampInt(int(math.Ceil(maxX)), dstBounds.Min.X, dstBounds.Max.X)
	y1 := clampInt(int(math.Ceil(maxY)), dstBounds.Min.Y, dstBounds.Max.Y)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			sx, sy := inv.apply(float64(x)+0.5, float64(y)+0.5)
			if sx < 0 || sy < 0 || sx >= float64(w) || sy >= float64(h) {
				continue
			}
			sc := src.sample(float32(sx), float32(sy), o.Filter, AddressClampToZero)
			sc.R, sc.G, sc.B, sc.A = scaleByte(sc.R, r), scaleByte(sc.G, g), scaleByte(sc.B, b), scaleByte(sc.A, a)
			i.blendPixel(x, y, sc, o.Blend)
		}
	}
}

func (i *Image) DrawTriangles(vertices []Vertex, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if i == nil || i.pix == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	if i.screen {
		i.DrawTrianglesOwned(append([]Vertex(nil), vertices...), append([]uint16(nil), indices...), texture, &o)
		return
	}
	for n := 0; n+2 < len(indices); n += 3 {
		i.drawTriangle(vertices[indices[n]], vertices[indices[n+1]], vertices[indices[n+2]], texture, o)
	}
	i.version++
}

func (i *Image) DrawTrianglesOwned(vertices []Vertex, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if i == nil || i.pix == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	if i.screen {
		i.commands = append(i.commands, DrawCommand{
			Vertices: vertices,
			Indices:  indices,
			Texture:  texture,
			Options:  o,
		})
		return
	}
	i.DrawTriangles(vertices, indices, texture, &o)
}

func (i *Image) DrawTriangles3D(vertices []Vertex3D, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if i == nil || i.pix == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	if i.screen {
		i.DrawTriangles3DOwned(append([]Vertex3D(nil), vertices...), append([]uint16(nil), indices...), texture, &o)
		return
	}
	vertices2D := make([]Vertex, len(vertices))
	for n, vertex := range vertices {
		vertices2D[n] = Vertex{
			DstX:   vertex.X,
			DstY:   vertex.Y,
			SrcX:   vertex.SrcX,
			SrcY:   vertex.SrcY,
			ColorR: vertex.ColorR,
			ColorG: vertex.ColorG,
			ColorB: vertex.ColorB,
			ColorA: vertex.ColorA,
		}
	}
	i.DrawTriangles(vertices2D, indices, texture, opts)
}

func (i *Image) DrawTriangles3DOwned(vertices []Vertex3D, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if i == nil || i.pix == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	if i.screen {
		i.worldCommands = append(i.worldCommands, WorldCommand{
			Vertices: vertices,
			Indices:  indices,
			Texture:  texture,
			Options:  o,
		})
		return
	}
	i.DrawTriangles3D(vertices, indices, texture, &o)
}

func (i *Image) DrawWorldMesh(mesh *WorldMesh) {
	if i == nil || i.pix == nil || mesh == nil || mesh.texture == nil || mesh.texture.pix == nil || len(mesh.vertices) == 0 || len(mesh.indices) == 0 {
		return
	}
	if i.screen {
		i.worldMeshes = append(i.worldMeshes, WorldMeshCommand{Mesh: mesh})
		return
	}
	i.DrawTriangles3D(mesh.vertices, mesh.indices, mesh.texture, &mesh.options)
}

func (i *Image) DrawWorldBillboard(cmd WorldBillboardCommand) {
	if i == nil || i.pix == nil || cmd.Texture == nil || cmd.Texture.pix == nil || cmd.Width <= 0 || cmd.Height <= 0 {
		return
	}
	if i.screen {
		i.worldBillboards = append(i.worldBillboards, cmd)
	}
}

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
	if filter != FilterLinear {
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
	a := uint32(src.A)
	ia := uint32(255 - src.A)
	i.pix.Pix[off] = byte((uint32(src.R)*a + uint32(i.pix.Pix[off])*ia) / 255)
	i.pix.Pix[off+1] = byte((uint32(src.G)*a + uint32(i.pix.Pix[off+1])*ia) / 255)
	i.pix.Pix[off+2] = byte((uint32(src.B)*a + uint32(i.pix.Pix[off+2])*ia) / 255)
	i.pix.Pix[off+3] = byte(a + uint32(i.pix.Pix[off+3])*ia/255)
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
