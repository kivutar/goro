package db

func firstAidEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 299 * EffectFrameDuration,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{{
			Kind:        EffectComponentFUNC,
			FuncName:    "FirstAid",
			TextureFile: "effect/pikapika2.bmp",
		}},
	}
}
