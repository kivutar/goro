package render

import (
	"image"
	"image/color"
	"math"
)

// Frame is the per-frame screen target. It owns draw command buffers and
// frame-only state; Image is only CPU texture data.
type Frame struct {
	width  int
	height int

	screenScaleX float32
	screenScaleY float32

	commands        []DrawCommand
	worldCommands   []WorldCommand
	worldMeshes     []WorldMeshCommand
	worldBillboards []WorldBillboardCommand
	uiTextBoxes     []UITextBoxCommand
	uiTextLabels    []UITextLabelCommand
	uiActorLabels   []UIActorLabelCommand
	imageUploads    []imageUpload

	clear  color.RGBA
	camera Camera3D
}

func NewFrame(width, height int) *Frame {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return &Frame{
		width:        width,
		height:       height,
		screenScaleX: 1,
		screenScaleY: 1,
	}
}

func (f *Frame) Bounds() image.Rectangle {
	if f == nil {
		return image.Rectangle{}
	}
	return image.Rect(0, 0, f.width, f.height)
}

func (f *Frame) BeginFrame() {
	if f == nil {
		return
	}
	f.commands = f.commands[:0]
	f.worldCommands = f.worldCommands[:0]
	f.worldMeshes = f.worldMeshes[:0]
	f.worldBillboards = f.worldBillboards[:0]
	f.uiTextBoxes = f.uiTextBoxes[:0]
	f.uiTextLabels = f.uiTextLabels[:0]
	f.uiActorLabels = f.uiActorLabels[:0]
	clear(f.imageUploads)
	f.imageUploads = f.imageUploads[:0]
	f.camera = Camera3D{}
}

type imageUpload struct {
	image   *Image
	options DrawTrianglesOptions
}

// PrepareImage uploads a texture with this frame without drawing it. Callers
// wait for FrameSubmitted before considering the upload complete.
func (f *Frame) PrepareImage(img *Image, options *DrawTrianglesOptions) {
	if f == nil || img == nil || img.pix == nil {
		return
	}
	var opts DrawTrianglesOptions
	if options != nil {
		opts = *options
	}
	f.imageUploads = append(f.imageUploads, imageUpload{image: img, options: opts})
}

func (f *Frame) clearUIOverlayCommands() {
	if f == nil {
		return
	}
	f.uiTextBoxes = f.uiTextBoxes[:0]
	f.uiTextLabels = f.uiTextLabels[:0]
	f.uiActorLabels = f.uiActorLabels[:0]
}

func (f *Frame) SetScreenScale(x, y float32) {
	if f == nil {
		return
	}
	if x <= 0 || math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
		x = 1
	}
	if y <= 0 || math.IsNaN(float64(y)) || math.IsInf(float64(y), 0) {
		y = 1
	}
	f.screenScaleX = x
	f.screenScaleY = y
}

func (f *Frame) SetCamera3D(camera Camera3D) {
	if f == nil {
		return
	}
	f.camera = camera
}

func (f *Frame) Fill(c color.Color) {
	if f == nil {
		return
	}
	f.clear = color.RGBAModel.Convert(c).(color.RGBA)
}

func (f *Frame) DrawImage(src *Image, opts *DrawImageOptions) {
	if f == nil || src == nil || src.pix == nil {
		return
	}
	var o DrawImageOptions
	if opts != nil {
		o = *opts
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
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
	f.DrawTrianglesOwned(vertices, quadIndices012213, src, &DrawTrianglesOptions{Filter: o.Filter, Address: AddressClampToZero, Blend: o.Blend})
}

func (f *Frame) DrawTriangles(vertices []Vertex, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if f == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	f.DrawTrianglesOwned(append([]Vertex(nil), vertices...), append([]uint16(nil), indices...), texture, &o)
}

func (f *Frame) DrawTrianglesOwned(vertices []Vertex, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if f == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	f.commands = append(f.commands, DrawCommand{
		Vertices: vertices,
		Indices:  indices,
		Texture:  texture,
		Options:  o,
	})
}

func (f *Frame) DrawTriangles3D(vertices []Vertex3D, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if f == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	f.DrawTriangles3DOwned(append([]Vertex3D(nil), vertices...), append([]uint16(nil), indices...), texture, &o)
}

func (f *Frame) DrawTriangles3DOwned(vertices []Vertex3D, indices []uint16, texture *Image, opts *DrawTrianglesOptions) {
	if f == nil || texture == nil || texture.pix == nil {
		return
	}
	var o DrawTrianglesOptions
	if opts != nil {
		o = *opts
	}
	f.worldCommands = append(f.worldCommands, WorldCommand{
		Vertices: vertices,
		Indices:  indices,
		Texture:  texture,
		Options:  o,
	})
}

func (f *Frame) DrawWorldMesh(mesh *WorldMesh) {
	if f == nil || mesh == nil || mesh.texture == nil || mesh.texture.pix == nil || len(mesh.vertices) == 0 || len(mesh.indices) == 0 {
		return
	}
	f.worldMeshes = append(f.worldMeshes, WorldMeshCommand{Mesh: mesh})
}

func (f *Frame) DrawWorldBillboard(cmd WorldBillboardCommand) {
	if f == nil || cmd.Texture == nil || cmd.Texture.pix == nil || cmd.Width <= 0 || cmd.Height <= 0 {
		return
	}
	f.worldBillboards = append(f.worldBillboards, cmd)
}
