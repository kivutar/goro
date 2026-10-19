package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawKaizelEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	// EF_KAIZEL launches JobLvUp50_2(0) and (45) at tick 15.
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) - 15 + 1
	alpha := 5*min(age, 7) - 2*max(age-9, 0)
	if now.Before(effect.starts) || age <= 0 || alpha <= 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	inner, outer := float64(1+age)/5, float64(3+3*age)/5
	height := (1.5 + .4*float64(min(age, 7))) / 5
	var vertices []render.Vertex3D
	var indices []uint16
	for blade := 0; blade < 8; blade++ {
		for slice := 0; slice < 3; slice++ {
			sin, cos := math.Sincos(degreesToRadians(float64(blade*45 - 1 + slice)))
			low, high, a := height*.1, height*.4, alpha/3
			if slice == 1 {
				low, high, a = height*.2, height, alpha
			}
			appendSoulLinkerQuad(&vertices, &indices, texture, [4]modelPoint3{
				{x: x + cos*inner, y: z + low, z: y + sin*inner},
				{x: x + cos*outer, y: z + high, z: y + sin*outer},
				{x: x + cos*outer, y: z, z: y + sin*outer},
				{x: x + cos*inner, y: z, z: y + sin*inner},
			}, color.RGBA{255, 255, 255, uint8(a)})
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}

func blueBodyAlpha(bodyTime int) float64 {
	if bodyTime < 0 || bodyTime > 50 {
		return 0
	}
	if bodyTime <= 10 {
		return float64(bodyTime*15) / 255
	}
	if bodyTime <= 20 {
		return 160.0 / 255
	}
	return float64(155-(bodyTime-20)*5) / 255
}

// Body effects reuse the composed sprite, including its appearance and scale.
// They never mutate the cached image or the actor's authoritative position.
func (m *WorldMode) drawActorSpriteWithEffects3D(ctx client.Context, screen *render.Frame, projection sceneProjection, entry sceneActorDrawEntry, billboard *spriteBillboard, scale, visibility, shadow float64, tint color.RGBA, blend render.Blend, now time.Time) {
	blue, eskeAge, fourWayAge := 0.0, -1, -1
	// Monsters use the Energy Coat option bit for Eske. The initial burst
	// settles into a faint outline until the server clears this bit.
	if isMonsterLikeHoverActor(entry.actor) && entry.actor.Opt3State&db.Opt3Energycoat != 0 {
		eskeAge = 149
	}
	var shake [4]float64
	zoomScale := 160 / normalizeSceneCameraZoom(projection.cameraZoom)
	for _, effect := range m.worldEffects {
		owner := effect.actorID == entry.actor.ID || (isLocalActor(ctx, entry.actor.ID) && isLocalActor(ctx, effect.actorID))
		if !owner || now.Before(effect.starts) || effect.expired(now) {
			continue
		}
		frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
		switch effect.effectID {
		case effectBlueHit:
			if frame <= 16 {
				blue = max(blue, blueBodyAlpha(frame*3))
			}
		case effectBlueBody, effectReflectBody:
			if frame <= 150 {
				blue = max(blue, blueBodyAlpha(frame/3))
			}
		case effectHated, effectKaizel:
			bodyTime := min(frame, 25)
			if frame > 150 {
				bodyTime = max(175-frame, 0)
			}
			blue = max(blue, blueBodyAlpha(bodyTime))
		case effectAsuraBodyMonster:
			eskeAge = min(frame, 149)
		case effectFourWayBody:
			if frame >= 30 && frame < 60 {
				fourWayAge = frame - 30
			}
		case effectQuakeBody3:
			if frame > 30 && frame <= 50 {
				for i := range shake {
					shake[i] = float64(hitEffectRandom(effect, frame*4+i, 21)-10) * zoomScale
				}
			}
		}
	}
	reflectBody := entry.actor.Opt3State&db.Opt3Kaite != 0
	if blue == 0 && eskeAge < 0 && fourWayAge < 0 && shake == [4]float64{} && !reflectBody {
		m.drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, scale, visibility, shadow, tint, blend)
		return
	}
	x, y, z := entry.worldX, entry.worldY, actorSpriteWorldZ(entry.worldZ)
	right, up, units, ok := projection.BillboardBasis(x, y, z)
	if !ok {
		return
	}
	w, h := float64(billboard.image.Bounds().Dx()), float64(billboard.image.Bounds().Dy())
	left, top := -billboard.anchorX*scale+shake[2], -billboard.anchorY*scale+shake[0]
	width, height := w*scale+shake[3]-shake[2], h*scale+shake[1]-shake[0]
	if w == 0 || h == 0 || width == 0 || height == 0 {
		return
	}
	options := spriteBillboardTriangleDrawOptions()
	options.Filter, options.Blend = render.FilterSprite, blend
	alpha := visibility * float64(tint.A) / 255
	cmd := worldSpriteBillboardCommand(m.spriteTexture(billboard.image), *options,
		modelPoint3{x: x, y: z, z: y}, mul3(right, units*width/w), mul3(up, -units*height/h),
		float32(w), float32(h), float32(-left*w/width), float32(-top*h/height), multiplyTint(tint, shadow, shadow, shadow, visibility))
	cmd.DepthUpAxis = [3]float32{0, float32(-units * height / h), 0}
	// Native Kaite draws four expanding silhouettes behind a translucent body.
	if reflectBody && blend != render.BlendLighter {
		phase := float64(now.UnixNano()/int64(db.EffectFrameDuration)%200) * .1
		for i := 1; i <= 4; i++ {
			spread := math.Mod(float64(i*5)+phase, 20)
			drawBodyExpansion(screen, cmd, billboard, right, up, units, scale, spread*zoomScale, alpha*(100-spread*5)/255, blend)
		}
		cmd.ColorA *= 150.0 / 255
	}
	screen.DrawWorldBillboard(cmd)
	if blue > 0 {
		flash := cmd
		flash.Options.Blend = render.BlendLighter
		flash.ColorR, flash.ColorG, flash.ColorB, flash.ColorA = 5.0/255, 5.0/255, 1, float32(blue*alpha)
		screen.DrawWorldBillboard(flash)
		screen.DrawWorldBillboard(flash)
	}
	if eskeAge >= 0 {
		for i := 1; i <= 10; i++ {
			a := eskeAge - i*5
			if eskeAge > 80 {
				a = 95 - eskeAge + (10-i)*10
			}
			drawBodyExpansion(screen, cmd, billboard, right, up, units, scale, float64(i*3)*zoomScale, alpha*float64(clampInt(a, 0, 200))/255, render.BlendLighter)
		}
	}
	if fourWayAge >= 0 {
		for _, offset := range [...]modelPoint3{right, mul3(right, -1), up, mul3(up, -1)} {
			copy := cmd
			shift := mul3(offset, units*height*.05*float64(fourWayAge)*zoomScale)
			copy.Center[0] += float32(shift.x)
			copy.Center[1] += float32(shift.y)
			copy.Center[2] += float32(shift.z)
			copy.ColorR, copy.ColorG, copy.ColorB, copy.ColorA = 1, 1, 1, float32(alpha*float64(150-fourWayAge*5)/255)
			screen.DrawWorldBillboard(copy)
		}
	}
}

func drawBodyExpansion(screen *render.Frame, cmd render.WorldBillboardCommand, billboard *spriteBillboard, right, up modelPoint3, units, scale, spread, alpha float64, blend render.Blend) {
	if alpha <= 0 {
		return
	}
	w, h := float64(billboard.image.Bounds().Dx()), float64(billboard.image.Bounds().Dy())
	width, height := w*scale+2*spread, h*scale+2*spread
	r, u := mul3(right, units*width/w), mul3(up, -units*height/h)
	cmd.RightAxis = [3]float32{float32(r.x), float32(r.y), float32(r.z)}
	cmd.UpAxis = [3]float32{float32(u.x), float32(u.y), float32(u.z)}
	cmd.DepthUpAxis = [3]float32{0, float32(-units * height / h), 0}
	cmd.AnchorX = float32((billboard.anchorX*scale + spread) * w / width)
	cmd.AnchorY = float32((billboard.anchorY*scale + spread) * h / height)
	cmd.Options.Blend = blend
	cmd.ColorR, cmd.ColorG, cmd.ColorB, cmd.ColorA = 1, 1, 1, float32(alpha)
	screen.DrawWorldBillboard(cmd)
}
