package db

import "time"

func shieldBoomerangEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 499 * EffectFrameDuration,
		SFX:      []string{"effect\\cru_shield boomerang.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "ShieldBoomerang", TextureFile: "effect/shield_boomerang.bmp",
		}},
	}
}

func grandCrossEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 100 * EffectFrameDuration,
		SFX:      []string{"effect\\cru_grand cross.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "GrandCross",
			TextureFile: "effect/ring_white.tga", TextureFiles: []string{"effect/ring_yellow.tga"},
		}},
	}
}

func crusaderDefenseEffectSpec(texture string) EffectSpec {
	return EffectSpec{
		Duration: 200 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "CrusaderDefense", TextureFile: "effect/" + texture + ".tga",
		}},
	}
}

func providenceEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("providence", "", false)
	spec.Duration = 90 * EffectFrameDuration
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "Providence", TextureFile: "effect/ring_yellow.tga", Duration: spec.Duration,
	})
	return spec
}

func shrinkEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 26 * EffectFrameDuration,
		SFX:      []string{"effect\\EF_BeginSpell.wav"}, SFXDelays: []time.Duration{5 * EffectFrameDuration},
	}
}

func devotionLinkEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: time.Second,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "DevotionLink", TextureFile: "effect/cloud11.tga", Repeat: true,
		}},
	}
}

func devotionEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("devotion", "", false)
	// ProcessEZ2STR uses -20 pixels for Devotion, versus the usual -80.
	spec.Components[0].SpriteYOffset = 60
	return spec
}

func guardEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 44 * EffectFrameDuration,
		SFX:      []string{"effect\\kyrie_guard.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Guard",
			TextureFile: "effect/guardk.tga", TextureFiles: []string{"effect/guardk2.tga"},
		}},
	}
}

func crusaderDefEffectSpec() EffectSpec {
	return crusaderDefenseEffectSpec("ring_black")
}

func holyCrossEffectSpec() EffectSpec {
	return strEffectSpecAttached("holy_cross", "effect\\cru_holy cross.wav", false)
}

func shieldChargeEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("shield_charge", "", false)
	// Shield Charge uses a zero screen offset instead of the usual -80 pixels.
	spec.Components[0].SpriteYOffset = 80
	return spec
}

func spearQuickenEffectSpec() EffectSpec {
	return strEffectSpecAttached("twohand", "effect\\knight_twohandquicken.wav", false)
}

func reflectShieldEffectSpec() EffectSpec {
	return crusaderDefenseEffectSpec("ring_yellow")
}
