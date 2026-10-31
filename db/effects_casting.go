package db

import (
	"image/color"
	"time"
)

func blueCastingEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 60 * EffectFrameDuration, SFX: []string{"effect\\EF_BeginSpell.wav"},
		Components: []EffectComponent{{Kind: EffectComponentFUNC, FuncName: "BlueCasting", TextureFile: "effect/ring_blue.tga", AttachedEntity: true}},
	}
}

func coupleCastingEffectSpec() EffectSpec {
	spec := blueCastingEffectSpec()
	spec.Duration = 1000 * EffectFrameDuration
	spec.Components[0].TextureFile = "effect/ring_red.tga"
	return spec
}

func castAuraEffectSpec(texture string, tint color.RGBA, alphaMax, height, topSize float64, rotate bool) EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      texture,
			AlphaMax:         alphaMax,
			Fade:             true,
			Rotate:           rotate,
			Animation:        2,
			BottomSize:       1,
			TopSize:          topSize,
			Height:           height,
			TotalCircleSides: 32,
			CircleSides:      32,
			Color:            tint,
		}},
	}
}

func elementalCastAuraEffectSpec(texture string, tint color.RGBA, alphaMax float64) EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         0.3,
				Fade:             true,
				Rotate:           true,
				Animation:        1,
				BottomSize:       1,
				TopSize:          1,
				Height:           30,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         alphaMax,
				Fade:             true,
				Rotate:           true,
				Animation:        1,
				BottomSize:       1,
				TopSize:          1.3,
				Height:           2,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         alphaMax,
				Fade:             true,
				Rotate:           true,
				Animation:        2,
				BottomSize:       1,
				TopSize:          4,
				Height:           3,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
		},
	}
}

func darkCastingEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      "ring_black",
			Color:            color.RGBA{R: 255, G: 255, B: 255, A: 255},
			AlphaMax:         0.8,
			Animation:        2,
			AttachedEntity:   true,
			BlendMode:        2,
			BlendAdditive:    true,
			BottomSize:       1,
			TopSize:          5,
			Height:           4,
			Fade:             true,
			Rotate:           true,
			TotalCircleSides: 32,
			CircleSides:      32,
		}},
	}
}

func beginSpellEffectSpec() EffectSpec {
	return castAuraEffectSpec("ring_yellow", color.RGBA{R: 255, G: 245, B: 120, A: 255}, 0.8, 4, 5, false)
}

func beginSpell2EffectSpec() EffectSpec {
	return elementalCastAuraEffectSpec("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 0.6)
}

func beginSpell3EffectSpec() EffectSpec {
	return elementalCastAuraEffectSpec("ring_red", color.RGBA{R: 255, G: 100, B: 100, A: 255}, 0.7)
}

func beginSpell4EffectSpec() EffectSpec {
	return elementalCastAuraEffectSpec("ring_white", color.RGBA{R: 150, G: 255, B: 150, A: 255}, 0.6)
}

func beginSpell5EffectSpec() EffectSpec {
	return elementalCastAuraEffectSpec("ring_yellow", color.RGBA{R: 255, G: 245, B: 120, A: 255}, 0.8)
}

func beginSpell6EffectSpec() EffectSpec {
	return castAuraEffectSpec("ring_white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.8, 4, 5, true)
}

func beginSpell7EffectSpec() EffectSpec {
	return elementalCastAuraEffectSpec("ring_purple", color.RGBA{R: 200, G: 160, B: 255, A: 255}, 0.7)
}

func lockOnTargetEffectSpec() EffectSpec {
	return EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "LockOnTarget",
			TextureFile:    "effect/lockon128.tga",
			AttachedEntity: true,
		}},
	}
}

func castRingEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponentFUNC,
			FuncName:         "CastRing",
			TextureName:      "ring_yellow",
			AlphaMax:         0.9,
			Fade:             true,
			Rotate:           true,
			Animation:        1,
			BottomSize:       0.8,
			TopSize:          2.45,
			Height:           2.8,
			PosZ:             0.08,
			TotalCircleSides: 20,
			CircleSides:      20,
			Color:            color.RGBA{R: 255, G: 245, B: 150, A: 255},
		}},
	}
}

func groundSampleEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:        EffectComponentFUNC,
			FuncName:    "MagicTarget",
			TextureFile: "effect/magic_target.tga",
			AlphaMax:    0.9,
			PosZ:        0.08,
			SizeStart:   1,
			SizeEnd:     1,
		}},
	}
}

func groundCastEffectSpec(texture string, ticks int) EffectSpec {
	spec := funcEffectSpec("GroundCast", time.Duration(ticks)*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/" + texture + ".tga"
	return spec
}
