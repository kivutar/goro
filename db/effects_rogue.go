package db

import "time"

func rogueBurstEffectSpec(effectID int) EffectSpec {
	spec := funcEffectSpec("RogueBurst", 300*EffectFrameDuration, false)
	switch effectID {
	case EffectRogueCoin:
		spec.Components[0].TextureFile = "effect/coin_a.bmp"
		spec.SFX = []string{"effect\\rog_steal coin.wav"}
	case EffectIntimidate:
		spec.Duration = 999 * EffectFrameDuration
		spec.Components[0].TextureFile = "effect/white01.bmp"
		spec.SFX = []string{"effect\\EF_Bash.wav", "effect\\rog_intimidate.wav"}
		spec.SFXDelays = []time.Duration{0, 35 * EffectFrameDuration}
	case EffectTeiHit3:
		spec.Components[0].TextureFile = "effect/lens2.tga"
	}
	return spec
}

func backStabEffectSpec() EffectSpec {
	spec := funcEffectSpec("ImpactParticles", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/thunder_center.bmp"
	spec.Components[0].Duplicate = 80
	spec.SFX = []string{"effect\\rog_back stap.wav"}
	spec.SFXDelays = []time.Duration{20 * EffectFrameDuration}
	return spec
}

func divestEffectSpec(file string) EffectSpec {
	// The 2008 executable plays this sound six ticks into the STR animation.
	spec := strEffectSpecAttached(file, "effect\\복호격.wav", false)
	spec.SFXDelays = []time.Duration{6 * EffectFrameDuration}
	return spec
}

func closeConfineHitEffectSpec() EffectSpec {
	spec := funcEffectSpec("QuakeBody4", 60*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\ef_hit6.wav"}
	spec.SFXDelays = []time.Duration{20 * EffectFrameDuration}
	return spec
}

func closeConfineRingEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 99999 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponent3D, SpriteFile: "cconfine",
			Duration:    99999 * EffectFrameDuration,
			SpriteDelay: 12 * EffectFrameDuration, SpriteStopAtEnd: true,
			AlphaMax: 100.0 / 255, SizeStart: 100 * EffectPixelRatio, SizeEnd: 100 * EffectPixelRatio,
			AttachedEntity: true, Overlay: true,
		}},
	}
}

func intimidateEffectSpec() EffectSpec {
	return rogueBurstEffectSpec(EffectIntimidate)
}

func stealCoinEffectSpec() EffectSpec {
	return strEffectSpecAttached("steal_coin", "", false)
}

func stripWeaponEffectSpec() EffectSpec {
	return divestEffectSpec("strip_weapon")
}

func stripShieldEffectSpec() EffectSpec {
	return divestEffectSpec("strip_shield")
}

func stripArmorEffectSpec() EffectSpec {
	return divestEffectSpec("strip_armor")
}

func stripHelmEffectSpec() EffectSpec {
	return divestEffectSpec("strip_helm")
}

func rogueCoinEffectSpec() EffectSpec {
	return rogueBurstEffectSpec(EffectRogueCoin)
}

func teiHit3EffectSpec() EffectSpec {
	return rogueBurstEffectSpec(EffectTeiHit3)
}
