package render

import "image/color"

var quadIndices012213 = []uint16{0, 1, 2, 2, 1, 3}

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

type UIActorLabelCommand struct {
	Labels     []string
	Emblem     *Image
	CenterX    float64
	Y          float64
	Foreground color.RGBA
	Outline    color.RGBA
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
	UITextBoxAnchorTopCenter
)

type UITextBoxCommand struct {
	Text     string
	X        float64
	Y        float64
	AltY     float64
	Anchor   UITextBoxAnchor
	MaxWidth float64
	MaxLines int
	style    overlayTextBoxStyle
}
