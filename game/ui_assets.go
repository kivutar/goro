package game

import (
	"github.com/kivutar/goro/client"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func (m *WorldMode) DrawInventoryItemIcon(screen *render.Image, manager *res.Manager, item session.InventoryItem, x, y int) {
	m.drawInventoryItemIcon(screen, manager, item, x, y)
}

func (m *WorldMode) DrawSkillIcon(screen *render.Image, manager *res.Manager, skill session.Skill, x, y, size int) {
	m.drawSkillIcon(screen, manager, skill, x, y, size)
}

func (m *WorldMode) SkillIconImage(manager *res.Manager, skill session.Skill, size int) image.Image {
	if size <= 0 {
		return nil
	}
	img := render.NewImage(size, size)
	m.DrawSkillIcon(img, manager, skill, 0, 0, size)
	return img.RGBA()
}

func (m *WorldMode) drawItemInfoIllustration(screen *render.Image, manager *res.Manager, item session.InventoryItem, x, y, width, height int) {
	if screen == nil || width <= 0 || height <= 0 {
		return
	}
	if image := m.itemCollectionTexture(manager, item.ItemID, item.Identified); image != nil {
		bounds := image.Bounds()
		srcW, srcH := float64(bounds.Dx()), float64(bounds.Dy())
		if srcW > 0 && srcH > 0 {
			scale := math.Min(float64(width)/srcW, float64(height)/srcH)
			dstW, dstH := srcW*scale, srcH*scale
			var opts render.DrawImageOptions
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate(float64(x)+(float64(width)-dstW)/2, float64(y)+(float64(height)-dstH)/2)
			opts.Filter = spriteDrawFilter()
			screen.DrawImage(image, &opts)
			return
		}
	}
	render.DebugPrintAtColor(screen, "No image", x+width/2-24, y+height/2-7, color.RGBA{R: 98, G: 112, B: 126, A: 255})
}

func (m *WorldMode) ItemInfoIllustrationImage(manager *res.Manager, item session.InventoryItem, width, height int) image.Image {
	if width <= 0 || height <= 0 {
		return nil
	}
	img := render.NewImage(width, height)
	m.drawItemInfoIllustration(img, manager, item, 0, 0, width, height)
	return img.RGBA()
}

func (m *WorldMode) drawEquipmentPreview(screen *render.Image, ctx client.Context, x, y, width, height int) {
	if screen == nil || width <= 0 || height <= 0 {
		return
	}
	view := m.playerView
	if view == nil && ctx.Resources != nil && ctx.Session != nil {
		loaded, _ := loadPlayerHumanoidSpriteView(ctx.Resources, selectedCharacter(ctx.Session), ctx.Session.Sex)
		view = loaded
		if loaded != nil {
			m.playerView = loaded
		}
	}
	state := spriteState{
		actionFamily: spriteActionIdle,
		direction:    4,
	}
	billboard, ok := humanoidBillboardForState(view, state, time.Now())
	if !ok || billboard == nil || billboard.image == nil {
		render.DrawRect(screen, float64(x+width/2-14), float64(y+height/2-24), 28, 48, render.ColorPanel)
		render.DrawRect(screen, float64(x+width/2-14), float64(y+height/2-24), 28, 2, render.ColorAccent)
		return
	}
	bounds := visibleImageBounds(billboard.image)
	if bounds.Empty() {
		return
	}
	srcW, srcH := float64(bounds.Dx()), float64(bounds.Dy())
	if srcW <= 0 || srcH <= 0 {
		return
	}
	scale := math.Min(float64(width-4)/srcW, float64(height-4)/srcH)
	scale = math.Min(scale, 1.6)
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	dstW, dstH := srcW*scale, srcH*scale
	dstX := float64(x) + (float64(width)-dstW)/2
	dstY := float64(y) + (float64(height)-dstH)/2
	vertices := []render.Vertex{
		{DstX: float32(dstX), DstY: float32(dstY), SrcX: float32(bounds.Min.X), SrcY: float32(bounds.Min.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX + dstW), DstY: float32(dstY), SrcX: float32(bounds.Max.X), SrcY: float32(bounds.Min.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX), DstY: float32(dstY + dstH), SrcX: float32(bounds.Min.X), SrcY: float32(bounds.Max.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX + dstW), DstY: float32(dstY + dstH), SrcX: float32(bounds.Max.X), SrcY: float32(bounds.Max.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	screen.DrawTrianglesOwned(vertices, quadIndices012213, billboard.image, &render.DrawTrianglesOptions{Filter: spriteDrawFilter(), Address: render.AddressClampToZero})
}

func (m *WorldMode) EquipmentPreviewImage(ctx client.Context, width, height int) image.Image {
	if width <= 0 || height <= 0 {
		return nil
	}
	img := render.NewImage(width, height)
	m.drawEquipmentPreview(img, ctx, 0, 0, width, height)
	return img.RGBA()
}
