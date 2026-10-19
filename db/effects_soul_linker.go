package db

import (
	"image/color"
	"time"
)

func estinEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:  200 * EffectFrameDuration,
		SFX:       []string{"effect\\t_에너지방출.wav"},
		SFXDelays: []time.Duration{10 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Estin",
			TextureFile: "effect/270바람.tga", Color: color.RGBA{80, 80, 255, 255},
		}},
	}
}

func estunEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 250 * EffectFrameDuration,
		SFX:      []string{"effect\\t_에너지방출.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Estun",
			TextureFiles: []string{"effect/thunder_ball_0002.bmp", "effect/thunder_ball_0003.bmp"},
			Color:        color.RGBA{150, 150, 250, 255},
		}},
	}
}

func soulLinkerHitLinesEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 200 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "SoulLinkerHitLines",
			TextureFile: "effect/white02.bmp", AttachedEntity: true,
		}},
	}
}

func esmaEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 200 * EffectFrameDuration,
		SFX: []string{
			"effect\\t_날라차기.wav", "effect\\t_날라차기.wav", "effect\\t_날라차기.wav",
			"effect\\t_날라차기.wav", "effect\\t_날라차기.wav",
		},
		SFXDelays: []time.Duration{5 * EffectFrameDuration, 20 * EffectFrameDuration, 35 * EffectFrameDuration, 50 * EffectFrameDuration, 65 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Esma",
			TextureFile: "effect/270바람.tga", Color: color.RGBA{80, 80, 255, 255},
		}},
	}
}

// Kaahi and Kaizel share EF_HATED's particles and body glow. Keeping the
// components together lets each skill play its own sound only once.
func soulLinkerBuffEffectSpec(kaizel bool) EffectSpec {
	spec := EffectSpec{
		Duration: 200 * EffectFrameDuration,
		SFX:      []string{"effect\\t_보조마법.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "SoulLinkerParticles",
			TextureFiles: []string{"effect/pok1.tga", "effect/pok3.tga"},
			Color:        color.RGBA{150, 150, 250, 255}, AttachedEntity: true,
		}},
	}
	if kaizel {
		spec.SFX = []string{"effect\\priest_resurrection.wav"}
		spec.Components = append(spec.Components, EffectComponent{
			Kind: EffectComponentFUNC, FuncName: "Kaizel",
			TextureFile: "effect/magic_blue.tga", AttachedEntity: true,
		})
	}
	return spec
}

func esmaReadyEffectSpec() EffectSpec {
	spec := soulLinkerBuffEffectSpec(false)
	spec.SFX = nil
	return spec
}

func soulLinkerBodyEffectSpec(name, sound string, frames int) EffectSpec {
	spec := funcEffectSpec(name, time.Duration(frames)*EffectFrameDuration, true)
	if sound != "" {
		spec.SFX = []string{sound}
	}
	return spec
}

func esmaHitTextureEffectSpec() EffectSpec {
	return EffectSpec{Duration: 200 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "EsmaHitTexture",
		TextureFile: "effect/freeze_a.bmp", AttachedEntity: true,
	}}}
}
