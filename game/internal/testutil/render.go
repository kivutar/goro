package testutil

import (
	"image/color"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func EffectRenderTestSprite() *scene.SpriteView {
	return &scene.SpriteView{
		Spr: &res.SPR{},
		Act: &res.ACT{Actions: []res.ACTAction{{DelayMS: 100, Animations: []res.ACTAnimation{{}}}}},
		Billboards: map[scene.SpriteBillboardKey]*scene.Billboard{
			{}: {Image: render.NewImage(8, 8), AnchorX: 4, AnchorY: 8},
		},
	}
}

func GNDWithTopHeights(width, height int, fn func(x, y int) [4]float32) *res.GND {
	gnd := &res.GND{
		Width:    width,
		Height:   height,
		Cells:    make([]res.GNDCell, width*height),
		Surfaces: []res.GNDSurface{{TextureID: 0, LightmapID: -1, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}}},
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gnd.Cells[x+y*width] = res.GNDCell{Top: 0, Front: -1, Right: -1, Heights: fn(x, y)}
		}
	}
	return gnd
}

func SolidRGBAFrame(width, height int) []byte {
	data := make([]byte, width*height*4)
	for i := 0; i < len(data); i += 4 {
		data[i+0] = 255
		data[i+3] = 255
	}
	return data
}
