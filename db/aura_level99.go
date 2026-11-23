package db

import (
	"time"
)

func level99EffectSpec() EffectSpec {
	spec := funcEffectSpec("Level99Aura", 5*time.Minute, true)
	spec.Components[0].TextureFile = "effect/ring_blue.tga"
	return spec
}

func level99GroundEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "GroundAura",
			TextureFile:    "effect/pikapika2.bmp",
			AttachedEntity: true,
		}},
	}
}

// level99BubbleEffectSpec is EF_LEVEL99_3, CRagEffect::Level99_3(0): white
// freezing_circle particles shown with /aura off.
func level99BubbleEffectSpec() EffectSpec {
	return level99ParticleEffectSpec("effect/freezing_circle.bmp")
}

// level99BlueBubbleEffectSpec is EF_LEVEL99_4, CRagEffect::Level99_3(1): the
// blue whitelight bubbles of the full aura.
func level99BlueBubbleEffectSpec() EffectSpec {
	return level99ParticleEffectSpec("effect/whitelight.tga")
}

func level99ParticleEffectSpec(texture string) EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "Level99Bubble",
			TextureFile:    texture,
			AttachedEntity: true,
		}},
	}
}
