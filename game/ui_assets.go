package game

import (
	"fmt"
	"image"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

type previewRenderTarget interface {
	DrawImage(*render.Image, *render.DrawImageOptions)
	DrawTrianglesOwned([]render.Vertex, []uint16, *render.Image, *render.DrawTrianglesOptions)
}

const (
	monsterInfoPreviewWidth  = 110
	monsterInfoPreviewHeight = 160
)

func (m *WorldMode) DrawInventoryItemIcon(screen *render.Frame, manager *res.Manager, item session.InventoryItem, x, y int) {
	m.drawInventoryItemIcon(screen, manager, item, x, y)
}

func (m *WorldMode) DrawSkillIcon(screen *render.Frame, manager *res.Manager, skill session.Skill, x, y, size int) {
	m.drawSkillIcon(screen, manager, skill, x, y, size)
}

func (m *WorldMode) SkillIconImage(manager *res.Manager, skill session.Skill) image.Image {
	if manager == nil || skill.ID == 0 {
		return nil
	}
	resourceName, ok := manager.SkillResourceName(int(skill.ID))
	if !ok {
		resourceName = strings.ToUpper(strings.ReplaceAll(skillLabel(skill), " ", "_"))
	}
	key := fmt.Sprintf("__skill_icon_image_%d_%s", skill.ID, resourceName)
	if m.imageCache == nil {
		m.imageCache = make(map[string]image.Image)
	}
	if m.imageMiss == nil {
		m.imageMiss = make(map[string]struct{})
	}
	if img := m.imageCache[key]; img != nil {
		return img
	}
	if _, ok := m.imageMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.SkillIconTextureCandidates(resourceName, int(skill.ID)))
	if err != nil {
		m.imageMiss[key] = struct{}{}
		return nil
	}
	m.imageCache[key] = img
	return img
}

func (m *WorldMode) drawItemInfoIllustration(screen previewRenderTarget, manager *res.Manager, item session.InventoryItem, x, y, width, height int) {
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
}

func (m *WorldMode) ItemInfoIllustrationImage(manager *res.Manager, item session.InventoryItem, width, height int) image.Image {
	if width <= 0 || height <= 0 {
		return nil
	}
	img := render.NewImage(width, height)
	m.drawItemInfoIllustration(img, manager, item, 0, 0, width, height)
	return img.RGBA()
}

func (m *WorldMode) drawEquipmentPreview(screen previewRenderTarget, ctx client.Context, x, y, width, height int) {
	if screen == nil || width <= 0 || height <= 0 {
		return
	}
	view := m.playerView
	if view == nil && ctx.Resources != nil && ctx.Session != nil {
		loaded, _ := loadPlayerHumanoidSpriteView(ctx.Resources, localPlayerVisualCharacter(ctx), ctx.Session.Sex, localPlayerIsAdmin(ctx))
		view = loaded
		if loaded != nil {
			m.playerView = loaded
		}
	}
	m.drawHumanoidPreview(screen, view, x, y, width, height)
}

func (m *WorldMode) drawEquipmentPreviewForCharacter(screen previewRenderTarget, ctx client.Context, character session.Character, sex byte, x, y, width, height int) {
	if screen == nil || width <= 0 || height <= 0 || ctx.Resources == nil {
		return
	}
	view, _ := loadPlayerHumanoidSpriteView(ctx.Resources, character, sex, false)
	m.drawHumanoidPreview(screen, view, x, y, width, height)
}

func (m *WorldMode) drawHumanoidPreview(screen previewRenderTarget, view *humanoidSpriteView, x, y, width, height int) {
	state := spriteState{
		actionFamily: spriteActionIdle,
		direction:    4,
	}
	billboard, ok := humanoidBillboardForState(view, state, time.Now())
	if !ok || billboard == nil || billboard.image == nil {
		return
	}
	drawBillboardPreview(screen, billboard, x, y, width, height)
}

func drawBillboardPreview(screen previewRenderTarget, billboard *spriteBillboard, x, y, width, height int) {
	if screen == nil || billboard == nil || billboard.image == nil || width <= 0 || height <= 0 {
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

func (m *WorldMode) monsterInfoPreviewImage(ctx client.Context, class uint16, width, height int) image.Image {
	if width <= 0 || height <= 0 || ctx.Resources == nil {
		return nil
	}
	key := fmt.Sprintf("__monster_info_preview_%d_%dx%d", class, width, height)
	if m.imageCache == nil {
		m.imageCache = make(map[string]image.Image)
	}
	if cached := m.imageCache[key]; cached != nil {
		return cached
	}
	actor := worldstate.Actor{
		Job:           int16(class),
		HasObjectType: true,
		ObjectType:    actorObjectTypeMob,
	}
	view := m.nonPCSpriteView(ctx, actor)
	state := spriteState{
		actionFamily:   spriteActionIdle,
		direction:      4,
		fixedMotion:    0,
		hasFixedMotion: true,
	}
	billboard, ok := singleSpriteBillboardForState(view, state, time.Now())
	if !ok {
		return nil
	}
	target := render.NewImage(width, height)
	drawBillboardPreview(target, billboard, 0, 0, width, height)
	preview := target.RGBA()
	m.imageCache[key] = preview
	return preview
}

func (m *WorldMode) EquipmentPreviewImage(ctx client.Context, width, height int) image.Image {
	if width <= 0 || height <= 0 {
		return nil
	}
	img := render.NewImage(width, height)
	m.drawEquipmentPreview(img, ctx, 0, 0, width, height)
	return img.RGBA()
}

func (m *WorldMode) EquipmentPreviewImageForCharacter(ctx client.Context, character session.Character, sex byte, width, height int) image.Image {
	if width <= 0 || height <= 0 {
		return nil
	}
	img := render.NewImage(width, height)
	m.drawEquipmentPreviewForCharacter(img, ctx, character, sex, 0, 0, width, height)
	return img.RGBA()
}
