package db

import (
	"image/color"
	"strconv"
	"time"
)

func teiHitEffectSpec(asura bool) EffectSpec {
	spec := funcEffectSpec("MonkRays", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.Components[0].Duplicate = 80
	if asura {
		spec.Components[0].TextureFile = "effect/lens1.tga"
		spec.Components[0].Duplicate = 96
	}
	return spec
}

func teiHit2EffectSpec() EffectSpec {
	spec := funcEffectSpec("ImpactParticles", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/thunder_center.bmp"
	spec.Components[0].Duplicate = 20
	return spec
}

func tanjiEffectSpec() EffectSpec {
	spec := funcEffectSpec("Tanji", 500*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/blue_ivy.bmp"
	spec.SFX = []string{"effect\\mon_탄지신통.wav"}
	return spec
}

func tripleAttackEffectSpec() EffectSpec {
	spec := funcEffectSpec("TripleAttack", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/cloud11.tga"
	spec.SFX = []string{"effect\\ef_hit2.wav", "effect\\ef_hit4.wav", "effect\\ef_hit2.wav"}
	spec.SFXDelays = []time.Duration{10 * EffectFrameDuration, 20 * EffectFrameDuration, 30 * EffectFrameDuration}
	return spec
}

func hitLineEffectSpec() EffectSpec {
	spec := funcEffectSpec("HitLine", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/white02.bmp"
	return spec
}

func gumgangEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 99999 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "Gumgang",
			TextureFiles:   []string{"effect/super1.bmp", "effect/super2.bmp", "effect/super3.bmp", "effect/super4.bmp", "effect/super5.bmp"},
			Color:          color.RGBA{R: 100, G: 100, B: 255, A: 255},
			AttachedEntity: true,
		}},
	}
}

func chimtoEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 16 * EffectFrameDuration,
		SFX:      []string{"effect\\mon_침투경.wav"},
		Components: []EffectComponent{{
			Kind:        EffectComponent3D,
			SpriteFile:  "침투경_", // misc/chimto in the original source's resource aliases
			SpriteDelay: 2 * EffectFrameDuration,
			Duration:    16 * EffectFrameDuration,
			AlphaMax:    200.0 / 255,
			SizeStart:   effectTableSize(250),
			SizeEnd:     effectTableSize(250),
		}},
	}
}

func absorbSpiritsEffectSpec() EffectSpec {
	spec := funcEffectSpec("AbsorbSpirits", 100*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/ring_blue.tga"
	spec.SFX = []string{"effect\\흡기.wav"}
	return spec
}

func beginAsuraEffectSpec(champion bool) EffectSpec {
	spec := blueCastingEffectSpec()
	spec.Duration = 350 * EffectFrameDuration
	spec.Components[0].TextureFile = "effect/ring_white.tga"
	glyphs := EffectComponent{Kind: EffectComponentFUNC, FuncName: "AsuraGlyphs"}
	for i := 1; i <= 6; i++ {
		index := i
		if champion {
			index += 10
		}
		glyphs.TextureFiles = append(glyphs.TextureFiles, "effect/asura"+strconv.Itoa(index)+".tga")
	}
	spec.Components = append(spec.Components, glyphs)
	return spec
}

func steelBodyEffectSpec() EffectSpec {
	spec := gumgangEffectSpec()
	// A packet effect is finite; the Steel Body status keeps its aura alive.
	spec.Duration = 2 * time.Second
	spec.Components[0].Duplicate = 4
	spec.SFX = []string{"effect\\mon_금강불괴.wav"}
	return spec
}

func steelBodyRingEffectSpec() EffectSpec {
	return gumgang2EffectSpec()
}

func gumgang2EffectSpec() EffectSpec {
	return groundCastEffectSpec("ring_yellow", 300)
}

func teiHit1EffectSpec() EffectSpec {
	return teiHitEffectSpec(false)
}

func gumgang3EffectSpec() EffectSpec {
	return groundCastEffectSpec("ring_yellow", 300)
}

func teiHit1XEffectSpec() EffectSpec {
	return teiHitEffectSpec(true)
}

func chainComboEffectSpec() EffectSpec {
	return strEffectSpecAttached("연환", "effect\\mon_연환.wav", false)
}

func beginAsuraCastEffectSpec() EffectSpec {
	return beginAsuraEffectSpec(false)
}

func teiHit1TEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 350 * time.Millisecond,
		SFX:      []string{"effect\\mon_아수라 패황권.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "effect/lens1.tga",
			Color:            effectTableColor(0.1, 0.1, 1),
			Duration:         250 * time.Millisecond,
			Delay:            100 * time.Millisecond,
			Duplicate:        24,
			DuplicateDelay:   0,
			AlphaMax:         0.8,
			BlendMode:        2,
			BlendAdditive:    true,
			FadeIn:           true,
			FadeOut:          true,
			AttachedEntity:   true,
			PosXEndRand:      40,
			PosYEndRand:      40,
			SizeStartX:       effectTableSize(10),
			SizeStartY:       effectTableSize(150),
			SizeEndX:         effectTableSize(10),
			SizeEndY:         effectTableSize(150),
			Overlay:          true,
			RotateToTarget:   true,
			RotateWithCamera: true,
		}},
	}
}
