package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

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

func (m *System) DrawActorSpriteBodyEffects3D(ctx client.Context, screen *render.Frame, projection scene.Projection, entry scene.ActorDrawEntry, billboard *scene.Billboard, scale, visibility, shadow float64, tint color.RGBA, blend render.Blend, jump float64, now time.Time) {
	if len(billboard.Parts) > 0 {
		for _, part := range billboard.Parts {
			m.DrawActorSpriteBodyEffects3D(ctx, screen, projection, entry, part, scale, visibility, shadow, tint, blend, jump, now)
		}
		return
	}
	if billboard.LayerGroup == 1 && entry.Actor.Opt3State&db.Opt3Aurablade != 0 {
		blend = render.BlendLighter
	}
	doubleBody := false
	weaponSpark := false
	blue, red, eskeAge, fourWayAge := 0.0, 0.0, -1, -1
	soulAlpha := 0.0
	// Monsters use the Energy Coat option bit for Eske. The initial burst
	// settles into a faint outline until the server clears this bit.
	if scene.IsMonsterLikeHoverActor(entry.Actor) && entry.Actor.Opt3State&db.Opt3Energycoat != 0 {
		eskeAge = 149
	}
	var shake [4]float64
	squash, lift, turn := 1.0, 0.0, 0.0
	zoomScale := 160 / scene.NormalizeSceneCameraZoom(projection.CameraZoom)
	local := scene.IsLocalActor(ctx, entry.Actor.ID) || ctx.World != nil && entry.Actor.ID != 0 && entry.Actor.ID == ctx.World.Player.ID
	for _, effect := range m.Active {
		owner := effect.ActorID == entry.Actor.ID || local && scene.IsLocalActor(ctx, effect.ActorID)
		if !owner || now.Before(effect.Starts) || effect.Expired(now) || !effectEnabled(ctx, effect.EffectID) {
			continue
		}
		frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
		switch effect.EffectID {
		case HitLine2:
			if frame >= 10 && frame <= 20 {
				doubleBody = doubleBody || billboard.LayerGroup == 0
				tint.R, tint.G, tint.B = 255, 120, 50
			}
		case Chemical2Dash:
			if frame >= 20 && frame <= 100 {
				doubleBody = doubleBody || billboard.LayerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 150
			}
		case Bash3D2:
			if frame >= 5 && frame <= 35 && billboard.LayerGroup == 0 {
				blend = render.BlendLighter
			}
		case Bash3D:
			if frame >= 20 && frame <= 40 {
				doubleBody = doubleBody || billboard.LayerGroup == 0
				tint.R, tint.G, tint.B = 255, 200, 200
			}
		case Shrink:
			if frame >= 5 && frame < 26 {
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G, tint.B = 250, 250, 100
					if billboard.LayerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case Portal5:
			if frame >= 5 && frame <= 65 {
				tint.R, tint.G, tint.B = 250, 250, 200
				if billboard.LayerGroup == 0 {
					blend = render.BlendLighter
				}
			}
		case RejectSword:
			if frame < 30 {
				doubleBody = doubleBody || billboard.LayerGroup == 0
				shade := uint8(100)
				if frame%6 >= 3 {
					shade = 250
				}
				tint.R, tint.G, tint.B = shade, shade, shade
			}
		case Preserve:
			if frame >= 10 && frame < 31 {
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G, tint.B = 250, 250, 250
					if billboard.LayerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case SoulBreaker, SoulBreaker2, EDP:
			end := 10
			if effect.EffectID == EDP {
				end = 80
			}
			if frame <= end {
				tint.R, tint.G, tint.B = 255, 255, 255
				if effect.EffectID != EDP || frame%2 == 0 {
					tint.G = 0
					if billboard.LayerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case MagicCrasher:
			if billboard.LayerGroup == 0 && entry.Stealth == scene.StealthVisible && MagicCrasherFlashActive(now.Sub(effect.Starts)) {
				blend = render.BlendLighter
			}
		case LightBlade:
			weaponSpark = billboard.LayerGroup == 1 && frame%2 == 0
		case EnergyDrain2:
			if frame >= 55 && frame <= 65 {
				tint.R, tint.G, tint.B = 100, 100, 255
				if billboard.LayerGroup == 0 {
					blend = render.BlendLighter
				}
			}
		case EnergyDrain3:
			if frame >= 50 && frame <= 80 {
				tint.R, tint.G, tint.B = uint8(250-2*frame), uint8(250-2*frame), 250
				doubleBody = billboard.LayerGroup == 0
			}
		case LinkLight:
			if frame >= 40 && frame <= 70 {
				tint.R, tint.G, tint.B = 200, 150, 50
				if billboard.LayerGroup == 0 {
					blend = render.BlendLighter
				}
			}
		case Memorize, DoubleCastBody:
			end := 36
			if effect.EffectID == DoubleCastBody {
				end = 51
			}
			if frame >= 10 && frame < end {
				doubleBody = doubleBody || effect.EffectID == DoubleCastBody && billboard.LayerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G, tint.B = 250, 250, 100
					if effect.EffectID == DoubleCastBody {
						tint.R, tint.G, tint.B = 255, 0, 0
					}
					if billboard.LayerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case ItemFast3:
			if frame <= 150 {
				tint.R, tint.G, tint.B = 250, 120, 120
			}
		case Assumptio:
			doubleBody = billboard.LayerGroup == 0
		case PinkBody:
			tint.R, tint.G, tint.B = 255, 89, 182
			if billboard.LayerGroup == 0 {
				doubleBody, blend = true, render.BlendLighter
			}
		case ChemicalBody:
			fixedStatusColor := entry.Actor.BodyState == db.BodyStateFreeze ||
				entry.Actor.BodyState == db.BodyStateStone || entry.Actor.BodyState == db.BodyStateStonewait ||
				entry.Actor.HealthState&(db.HealthStatePoison|db.HealthStateDpoison|db.HealthStateBleeding|db.HealthStateCurse) != 0
			if frame < 31 {
				doubleBody = billboard.LayerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G = 0, 0
					if billboard.LayerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			} else if frame < 62 && !fixedStatusColor {
				// Recover by eight color levels per tick unless a status fixes the color.
				tint.R, tint.G, tint.B = uint8(8*(frame-30)), uint8(8*(frame-30)), 255
			}
		case RedBody:
			tint.R, tint.G, tint.B = 255, 100, 100
			if billboard.LayerGroup == 0 {
				blend = render.BlendLighter
			}
		case LKConcentration:
			tint.R, tint.G, tint.B = 255, 255, 160
		case PierceBody:
			if frame < 141 {
				doubleBody = billboard.LayerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G, tint.B = 250, 250, 100
					if billboard.LayerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case Bash3D3, Bash3D4:
			if frame >= 20 && frame <= 50 {
				doubleBody = billboard.LayerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 255
				if effect.EffectID == Bash3D3 {
					tint.B = 200
				}
			}
		case AsuraBody:
			soulAlpha = max(soulAlpha, float64(min(frame, 200))/255)
		case FlowerCast:
			soulAlpha = max(soulAlpha, float64(max(0, min(frame, 400-frame)))/255)
		case RedLightBody:
			red = max(red, blueBodyAlpha(min(frame/8, 25)))
		case TaeReady:
			blue = max(blue, blueBodyAlpha(frame))
		case MadnessBlue:
			blue = max(blue, blueBodyAlpha(20+frame%32))
		case MadnessRed:
			red = max(red, blueBodyAlpha(20+frame%32))
		case RedHit:
			if frame <= 16 {
				red = max(red, blueBodyAlpha(frame*3))
			}
		case BlueHit:
			if frame <= 16 {
				blue = max(blue, blueBodyAlpha(frame*3))
			}
		case BlueBody, ReflectBody:
			if frame <= 150 {
				blue = max(blue, blueBodyAlpha(frame/3))
			}
		case Hated, Kaizel:
			bodyTime := min(frame, 25)
			if frame > 150 {
				bodyTime = max(175-frame, 0)
			}
			blue = max(blue, blueBodyAlpha(bodyTime))
		case AsuraBodyMonster:
			eskeAge = min(frame, 149)
		case FourWayBody:
			if frame >= 30 && frame < 60 {
				fourWayAge = frame - 30
			}
		case PressedBody:
			if frame >= 17 && frame < 50 {
				age := frame - 17
				squash = .4 - float64(HitEffectRandom(effect, frame, 6))*.01
				if age <= 3 {
					squash = 1 - .2*float64(age)
				}
			}
		case KickedBody:
			age := frame - 30
			if age >= 0 && age <= 23 {
				lift = .3 * math.Sin(scene.DegreesToRadians(float64(age*5)))
				if age > 18 {
					lift = .3
				}
				if age > 20 {
					lift = float64(23-age) * .1
				}
			}
		case SpinedBody, SpinedBody2:
			age := frame * 2
			if effect.EffectID == SpinedBody2 {
				age = frame - 14
			}
			if age >= 0 && age < 36 {
				turn = scene.DegreesToRadians(180 - 180*math.Cos(scene.DegreesToRadians(float64(age*5))))
			}
		case JumpBody:
			if frame > 35 {
				jump = float64((frame - 35) * 20)
				visibility *= float64(max(0, 255-(frame-35)*20)) / 255
			}
		case LandBody:
			jump = float64(max(0, 25-frame) * 20)
			visibility *= float64(scene.ClampInt((frame-10)*17, 0, 255)) / 255
			if frame < 25 {
				turn = scene.DegreesToRadians(180 - 180*math.Cos(scene.DegreesToRadians(float64((frame+11)*5))))
			}
		case QuakeBody, QuakeBody2, QuakeBody3, QuakeBody4:
			start, end := 30, 50
			randomRange, randomOffset := 21, 10
			if effect.EffectID == QuakeBody || effect.EffectID == QuakeBody2 {
				start, end = -1, 35
				if res.HasPlayerJobToken(int(entry.Actor.Job)) {
					randomRange, randomOffset = 11, 5
				}
			}
			if effect.EffectID == QuakeBody4 {
				start, end = 20, 59
				if frame > start && frame <= end && frame%2 == 0 {
					tint = scene.MultiplyTint(tint, 250.0/255, 50.0/255, 50.0/255, 1)
				}
			}
			if frame > start && frame <= end {
				for i := range shake {
					shake[i] = float64(HitEffectRandom(effect, frame*4+i, randomRange)-randomOffset) * zoomScale
				}
			}
		}
	}
	reflectBody := entry.Actor.Opt3State&db.Opt3Kaite != 0
	if blue == 0 && red == 0 && soulAlpha == 0 && eskeAge < 0 && fourWayAge < 0 && shake == [4]float64{} && !reflectBody && !doubleBody && !weaponSpark && squash == 1 && lift == 0 && turn == 0 && jump == 0 {
		m.DrawActorSpriteBillboard(screen, projection, billboard, entry.WorldX, entry.WorldY, entry.WorldZ, scale, visibility, shadow, tint, blend)
		return
	}
	x, y, z := entry.WorldX, entry.WorldY, scene.ActorSpriteWorldZ(entry.WorldZ)
	right, up, units, ok := projection.BillboardBasis(x, y, z)
	if !ok {
		return
	}
	w, h := float64(billboard.Image.Bounds().Dx()), float64(billboard.Image.Bounds().Dy())
	left, top := -billboard.AnchorX*scale+shake[2], -billboard.AnchorY*scale+shake[0]
	width, height := w*scale+shake[3]-shake[2], h*scale+shake[1]-shake[0]
	top += height*(1-squash-lift) - jump*zoomScale
	height *= squash
	if w == 0 || h == 0 || width == 0 || height == 0 {
		return
	}
	options := scene.SpriteBillboardTriangleDrawOptions()
	options.Filter, options.Blend = render.FilterSprite, blend
	alpha := visibility * float64(tint.A) / 255
	center := scene.Point3{X: x, Y: z, Z: y}
	if turn != 0 {
		s, c := math.Sincos(turn)
		// Rotate around the native body's pivot, 40 pixels above its feet.
		center = scene.Add3(center, scene.Mul3(scene.Add3(scene.Mul3(right, -s), scene.Mul3(up, 1-c)), 40*zoomScale*units))
		right, up = scene.Sub3(scene.Mul3(right, c), scene.Mul3(up, s)), scene.Add3(scene.Mul3(right, s), scene.Mul3(up, c))
	}
	cmd := scene.WorldSpriteBillboardCommand(m.spriteTexture(billboard.Image), *options,
		center, scene.Mul3(right, units*width/w), scene.Mul3(up, -units*height/h),
		float32(w), float32(h), float32(-left*w/width), float32(-top*h/height), scene.MultiplyTint(tint, shadow, shadow, shadow, visibility))
	cmd.DepthUpAxis = [3]float32{0, float32(-units * height / h), 0}
	if weaponSpark {
		spark := cmd
		spark.Options.Blend = render.BlendLighter
		spark.Options.DepthWrite = false
		screen.DrawWorldBillboard(spark)
	}
	if doubleBody {
		phase := float64(now.UnixNano() / int64(db.EffectFrameDuration) % 181)
		drawBodyExpansion(screen, cmd, billboard, right, up, units, scale,
			(5+1.5*math.Sin(scene.DegreesToRadians(phase)))*zoomScale, alpha, render.BlendLighter, tint)
	}
	// Native Kaite draws four expanding silhouettes behind a translucent body.
	if reflectBody && blend != render.BlendLighter {
		phase := float64(now.UnixNano()/int64(db.EffectFrameDuration)%200) * .1
		for i := 1; i <= 4; i++ {
			spread := math.Mod(float64(i*5)+phase, 20)
			drawBodyExpansion(screen, cmd, billboard, right, up, units, scale, spread*zoomScale, alpha*(100-spread*5)/255, blend, color.RGBA{255, 255, 255, 255})
		}
		cmd.ColorA *= 150.0 / 255
	}
	cmd.Options.DepthWrite = true
	screen.DrawWorldBillboard(cmd)
	cmd.Options.DepthWrite = false
	if blue > 0 {
		flash := cmd
		flash.Options.Blend = render.BlendLighter
		flash.ColorR, flash.ColorG, flash.ColorB, flash.ColorA = 5.0/255, 5.0/255, 1, float32(blue*alpha)
		screen.DrawWorldBillboard(flash)
		screen.DrawWorldBillboard(flash)
	}
	if red > 0 {
		flash := cmd
		flash.Options.Blend = render.BlendLighter
		flash.ColorR, flash.ColorG, flash.ColorB, flash.ColorA = 1, 5.0/255, 5.0/255, float32(red*alpha)
		screen.DrawWorldBillboard(flash)
		screen.DrawWorldBillboard(flash)
	}
	if soulAlpha > 0 {
		// Pc::RenderBodyLayer scales two blue copies around the feet by 1% and 2%.
		for i := 1; i <= 2; i++ {
			copy := cmd
			factor := float32(1 + float64(i)*.01)
			for axis := range copy.RightAxis {
				copy.RightAxis[axis] *= factor
				copy.UpAxis[axis] *= factor
				copy.DepthUpAxis[axis] *= factor
			}
			copy.Options.Blend = render.BlendLighter
			copy.ColorR, copy.ColorG, copy.ColorB, copy.ColorA = 55.0/255, 55.0/255, 1, float32(soulAlpha*alpha)
			screen.DrawWorldBillboard(copy)
		}
	}
	if eskeAge >= 0 {
		for i := 1; i <= 10; i++ {
			a := eskeAge - i*5
			if eskeAge > 80 {
				a = 95 - eskeAge + (10-i)*10
			}
			drawBodyExpansion(screen, cmd, billboard, right, up, units, scale, float64(i*3)*zoomScale, alpha*float64(scene.ClampInt(a, 0, 200))/255, render.BlendLighter, color.RGBA{255, 255, 255, 255})
		}
	}
	if fourWayAge >= 0 {
		for _, offset := range [...]scene.Point3{right, scene.Mul3(right, -1), up, scene.Mul3(up, -1)} {
			copy := cmd
			shift := scene.Mul3(offset, units*height*.05*float64(fourWayAge)*zoomScale)
			copy.Center[0] += float32(shift.X)
			copy.Center[1] += float32(shift.Y)
			copy.Center[2] += float32(shift.Z)
			copy.ColorR, copy.ColorG, copy.ColorB, copy.ColorA = 1, 1, 1, float32(alpha*float64(150-fourWayAge*5)/255)
			screen.DrawWorldBillboard(copy)
		}
	}
}

func drawBodyExpansion(screen *render.Frame, cmd render.WorldBillboardCommand, billboard *scene.Billboard, right, up scene.Point3, units, scale, spread, alpha float64, blend render.Blend, tint color.RGBA) {
	if alpha <= 0 {
		return
	}
	w, h := float64(billboard.Image.Bounds().Dx()), float64(billboard.Image.Bounds().Dy())
	width, height := w*scale+2*spread, h*scale+2*spread
	r, u := scene.Mul3(right, units*width/w), scene.Mul3(up, -units*height/h)
	cmd.RightAxis = [3]float32{float32(r.X), float32(r.Y), float32(r.Z)}
	cmd.UpAxis = [3]float32{float32(u.X), float32(u.Y), float32(u.Z)}
	cmd.DepthUpAxis = [3]float32{0, float32(-units * height / h), 0}
	cmd.AnchorX = float32((billboard.AnchorX*scale + spread) * w / width)
	cmd.AnchorY = float32((billboard.AnchorY*scale + spread) * h / height)
	cmd.Options.Blend = blend
	cmd.ColorR, cmd.ColorG, cmd.ColorB, cmd.ColorA = float32(tint.R)/255, float32(tint.G)/255, float32(tint.B)/255, float32(alpha)
	screen.DrawWorldBillboard(cmd)
}

// Auto Guard and Preserve share the native shield geometry.
func (m *System) drawGuardEffect(screen *render.Frame, ctx client.Context, _ scene.Projection, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := min(100, n*20) - 6*max(0, n-26)
	if alpha <= 0 {
		return
	}
	origin := EffectWorldAnchor(ctx, effect, now)
	radiusBase := 7.8
	facing := scene.DegreesToRadians(float64(90 + 45*EffectActorDirection(ctx, effect.ActorID, now)))
	if effect.EffectID == Preserve {
		radiusBase = 10
		facing = scene.DegreesToRadians(float64(90 + 45*effect.ActorDirection + 10*(n-1)))
	}
	for layer := 0; layer < 3; layer++ {
		path := component.TextureFile
		if layer == 2 {
			path = component.TextureFiles[0]
		}
		texture := m.EffectFileTexture(ctx.Resources, path)
		if texture == nil {
			continue
		}
		vertices, indices := []render.Vertex3D{}, []uint16{}
		radius, centerHeight := radiusBase+.2*float64(layer), radiusBase
		if layer == 0 {
			centerHeight += 2
		}
		if layer == 2 {
			radius += .1 * float64(max(0, n-26))
		}
		for row := 0; row < 3; row++ {
			pitch := scene.DegreesToRadians(float64(row*45 - 45))
			halfSize := float64(3 - scene.AbsInt(row-1))
			for col := 0; col < 5; col++ {
				yaw := facing + scene.DegreesToRadians(float64(col*45-90))
				center := scene.Point3{X: radius * math.Cos(pitch) * math.Cos(yaw), Y: radius*math.Sin(pitch) - centerHeight, Z: radius * math.Cos(pitch) * math.Sin(yaw)}
				var corners [4]scene.Point3
				for i, offset := range [4][2]float64{{halfSize, halfSize}, {-halfSize, halfSize}, {-halfSize, -halfSize}, {halfSize, -halfSize}} {
					x, y := offset[0], offset[1]
					z := y * math.Cos(math.Pi/2+pitch)
					point := scene.Add3(center, scene.Point3{
						X: x*math.Cos(yaw-math.Pi/2) - z*math.Sin(yaw-math.Pi/2),
						Y: y * math.Sin(math.Pi/2+pitch),
						Z: x*math.Sin(yaw-math.Pi/2) + z*math.Cos(yaw-math.Pi/2),
					})
					corners[3-i] = scene.Add3(origin, scene.Point3{X: point.X / 5, Y: -point.Y / 5, Z: point.Z / 5})
				}
				green := []uint8{205, 155, 225}[layer]
				appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{green, 255, green, uint8(alpha)})
			}
		}
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(layer < 2, false))
	}
}

func (m *System) DrawActor(ctx client.Context, screen *render.Frame, projection scene.Projection, entry scene.ActorDrawEntry, billboard *scene.Billboard, scale, visibility, shadow float64, tint color.RGBA, blend render.Blend, now time.Time, jump float64) {
	m.DrawActorBlurs(screen, projection, entry, billboard, scale, visibility, shadow, tint, now)
	m.DrawActorSpriteBodyEffects3D(ctx, screen, projection, entry, billboard, scale, visibility, shadow, tint, blend, jump, now)
}

func (m *System) DrawActorSpriteBillboard(screen *render.Frame, projection scene.Projection, billboard *scene.Billboard, worldX, worldY, worldZ, scale, alpha, shadow float64, tintColor color.RGBA, blend render.Blend) {
	options := scene.SpriteBillboardTriangleDrawOptions()
	options.DepthWrite = true
	options.Filter = render.FilterSprite
	options.Blend = blend
	scene.DrawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, m.spriteTexture(billboard.Image), worldX, worldY, scene.ActorSpriteWorldZ(worldZ), scale, alpha, shadow, tintColor, options)
}
