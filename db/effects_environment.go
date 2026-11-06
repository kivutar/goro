package db

import (
	"fmt"
	"image/color"
	"math"
	"time"
)

func sakuraEffectSpec() EffectSpec {
	return weatherLeafEffectSpec("data/sprite/이팩트/sakura01", color.RGBA{R: 255, G: 210, B: 225, A: 255})
}

func mapleEffectSpec() EffectSpec {
	return weatherLeafEffectSpec("data/sprite/이팩트/단풍", color.RGBA{R: 255, G: 170, B: 80, A: 255})
}

func cloudEffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.18, 35, -10, 20*time.Second)
}

func cloud2EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.58, 35, 4, 20*time.Second)
}

func cloud3EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/fog1.tga", "effect/fog2.tga", "effect/fog3.tga"}, color.RGBA{R: 120, G: 110, B: 100, A: 255}, 0.78, 45, -10, 34*time.Second)
}

func cloud5EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 225, G: 212, B: 194, A: 255}, 0.70, 50, 4, 8*time.Second)
}

func cloud6EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.58, 35, 4, 28*time.Second)
}

func cloud7EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 51, G: 79, B: 161, A: 255}, 0.55, 35, 4, 20*time.Second)
}

func cloud8EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 140, B: 51, A: 255}, 0.62, 35, 4, 20*time.Second)
}

func smokeEffectSpec() EffectSpec {
	return ambientSpriteEffectSpec("굴뚝연기", 50)
}

func fireflyEffectSpec() EffectSpec {
	return ambientSpriteEffectSpec("particle1", 140)
}

func torchEffectSpec() EffectSpec {
	return ambientSpriteEffectSpec("torch_01", 250)
}

func torchRedEffectSpec() EffectSpec {
	return coloredTorchEffectSpec("red")
}

func torchGreenEffectSpec() EffectSpec {
	return coloredTorchEffectSpec("green")
}

func torchPurpleEffectSpec() EffectSpec {
	return coloredTorchEffectSpec("violet")
}

func bubbleEffectSpec() EffectSpec {
	return strEffectSpecRandom("bubble%d", "", 1, 4)
}

func dragonSmokeEffectSpec() EffectSpec {
	return ambientSpriteEffectSpec("굴뚝연기", 50)
}

func mapPillarBlueEffectSpec() EffectSpec {
	return mapPillarEffectSpec("ring_blue", color.RGBA{R: 100, G: 100, B: 255, A: 255}, 2, 50.0/255.0)
}

func mapPillar2EffectSpec() EffectSpec {
	return mapPillarEffectSpec("ring_blue", color.RGBA{R: 100, G: 100, B: 255, A: 255}, 11, 70.0/255.0)
}

func mapPillar3EffectSpec() EffectSpec {
	return mapPillarEffectSpec("magic_green", color.RGBA{R: 100, G: 255, B: 100, A: 255}, 2, 50.0/255.0)
}

func mapPillar4EffectSpec() EffectSpec {
	return mapPillarEffectSpec("ring_red", color.RGBA{R: 255, G: 100, B: 100, A: 255}, 2, 50.0/255.0)
}

func mapGhostEffectSpec() EffectSpec {
	spec := funcEffectSpec("MapGhost", 99999999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/ghost.bmp"
	return spec
}

func glow1EffectSpec() EffectSpec {
	return mapGlowEffectSpec("effect/glow01.bmp")
}

func glow2EffectSpec() EffectSpec {
	return mapGlowEffectSpec("effect/glow02.bmp")
}

func glow4EffectSpec() EffectSpec {
	spec := funcEffectSpec("MapGlow", 99999999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/glow04.bmp"
	spec.Components[0].TextureFiles = []string{"effect/mpa3.tga"}
	return spec
}

func bubbleDropEffectSpec() EffectSpec {
	spec := funcEffectSpec("BubbleDrop", 200*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/썬더스톰파티클.tga"
	return spec
}

func rainbowEffectSpec() EffectSpec {
	spec := funcEffectSpec("Rainbow", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	return spec
}

func banjjakiiEffectSpec() EffectSpec {
	return ambientSpriteEffectSpec("크리스마스", 125)
}

func bottomBlueEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 20 * time.Second,
		Components: []EffectComponent{
			bottomBlueCylinder(1.5, 2.0, 40.0/255.0, 0, color.RGBA{R: 51, G: 153, B: 255, A: 255}),
			bottomBlueCylinder(1.58, 2.1, 32.0/255.0, 10, color.RGBA{R: 51, G: 153, B: 255, A: 255}),
			bottomBlueCylinder(1.65, 2.0, 15.0/255.0, 26.6, color.RGBA{R: 25, G: 102, B: 255, A: 255}),
			bottomBlueCylinder(1.65, 2.0, 15.0/255.0, 79.8, color.RGBA{R: 25, G: 102, B: 255, A: 255}),
		},
	}
}

func bottomBlueCylinder(size, height, alpha, angleY float64, tint color.RGBA) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "alpha_down",
		Duration:         20 * time.Second,
		AlphaMax:         alpha,
		BottomSize:       size,
		TopSize:          size,
		Height:           height,
		AngleY:           angleY,
		RotateWithCamera: true,
		TotalCircleSides: 4,
		CircleSides:      4,
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		Color:            tint,
	}
}

func coloredTorchEffectSpec(variant string) EffectSpec {
	spec := funcEffectSpec("MapTexture", 99999999*EffectFrameDuration, false)
	for frame := 1; frame <= 13; frame++ {
		spec.Components[0].TextureFiles = append(spec.Components[0].TextureFiles, fmt.Sprintf("effect/torch_%s%02d.bmp", variant, frame))
	}
	return spec
}

func mapGlowEffectSpec(texture string) EffectSpec {
	spec := funcEffectSpec("MapTexture", 99999999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = texture
	return spec
}

func mapPillarEffectSpec(texture string, tint color.RGBA, radiusStart, alphaMax float64) EffectSpec {
	const (
		duration   = 999 * EffectFrameDuration
		worldScale = 0.2
	)
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponentFUNC,
			FuncName:      "MapPillar",
			TextureName:   texture,
			Duration:      duration,
			AlphaMax:      alphaMax,
			BottomSize:    radiusStart * worldScale,
			Height:        120 * worldScale,
			BlendMode:     2,
			BlendAdditive: true,
			Color:         tint,
		}},
	}
}

func ambientSpriteEffectSpec(sprite string, frames int) EffectSpec {
	spec := funcEffectSpec("AmbientSprite", time.Duration(frames)*EffectFrameDuration, false)
	spec.Components[0].SpriteFile = sprite
	return spec
}

func weatherRainEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFile:   "effect/alpha_center.tga",
			Duration:      900 * time.Millisecond,
			Duplicate:     85,
			AlphaMax:      0.5,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: true,
			PosXRand:      28,
			PosYRand:      28,
			PosZ:          18,
			PosZEnd:       0,
			PosXEnd:       -3.5,
			PosYEnd:       2.0,
			PosZSmooth:    false,
			SizeStartX:    effectTableSize(2),
			SizeStartY:    effectTableSize(52),
			SizeEndX:      effectTableSize(2),
			SizeEndY:      effectTableSize(52),
			AngleStart:    -25,
			Color:         color.RGBA{R: 185, G: 215, B: 255, A: 255},
		}},
	}
}

func weatherSnowEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 8 * time.Second,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFile:   "effect/pok3.tga",
			Duration:      8 * time.Second,
			Duplicate:     180,
			AlphaMax:      0.85,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: true,
			PosXRand:      60,
			PosYRand:      60,
			PosZStartRand: 2,
			PosZ:          20,
			PosZEnd:       -4,
			PosZSmooth:    false,
			SizeStart:     effectTableSize(7),
			SizeEnd:       effectTableSize(7),
			Color:         color.RGBA{R: 245, G: 250, B: 255, A: 255},
		}},
	}
}

func weatherLeafEffectSpec(spriteFile string, tint color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 5200 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       spriteFile,
			SpriteRepeat:     true,
			Duration:         5200 * time.Millisecond,
			Duplicate:        36,
			AlphaMax:         0.92,
			FadeIn:           true,
			FadeOut:          true,
			PosXRand:         30,
			PosYRand:         30,
			PosZ:             20,
			PosZEnd:          0,
			PosXEnd:          -4,
			PosYEnd:          2,
			PosXEndRand:      3,
			PosYEndRand:      3,
			PosXSmooth:       true,
			PosYSmooth:       true,
			SizeStart:        effectTableSize(80),
			SizeEnd:          effectTableSize(80),
			AngleRandMin:     0,
			AngleRandMax:     360,
			RotateWithCamera: true,
			Color:            tint,
		}},
	}
}

func weatherCloudEffectSpec(textures []string, tint color.RGBA, alpha, radius, z float64, duration time.Duration) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFiles:  textures,
			Duration:      duration,
			Duplicate:     28,
			AlphaMax:      alpha,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: false,
			PosXRand:      radius,
			PosYRand:      radius,
			PosXEnd:       2.4,
			PosYEnd:       0.8,
			PosZ:          z,
			SizeStartX:    effectTableSize(1500),
			SizeStartY:    effectTableSize(900),
			SizeEndX:      effectTableSize(1500),
			SizeEndY:      effectTableSize(900),
			AngleRandMin:  -10,
			AngleRandMax:  10,
			Color:         tint,
		}},
	}
}

func einbrochWeatherCloudEffectSpec() EffectSpec {
	const duration = 34 * time.Second
	size := 35 * math.Sqrt2 * 0.1
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFiles:  []string{"effect/fog1.tga", "effect/fog2.tga", "effect/fog3.tga"},
			Duration:      duration,
			Duplicate:     320,
			AlphaMax:      170.0 / 255.0,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: false,
			Overlay:       true,
			PosXRand:      15,
			PosYRand:      15,
			PosZRand:      0.5,
			PosXEnd:       1.5,
			PosYEnd:       1.5,
			PosZ:          -2,
			SizeStartX:    size,
			SizeStartY:    size,
			SizeEndX:      size,
			SizeEndY:      size,
			SizeRand:      10 * math.Sqrt2 * 0.1,
			Color:         color.RGBA{R: 252, G: 171, B: 143, A: 255},
		}},
	}
}
