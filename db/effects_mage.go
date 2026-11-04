package db

import "time"

func mageProjectileEffectSpec(id int) EffectSpec {
	spec := funcEffectSpec("MageProjectile", 52*EffectFrameDuration, true)
	switch id {
	case effectFireBall:
		spec.Components[0].SpriteFile = "fireball"
		for _, tick := range []int{20, 24, 28, 32} {
			spec.SFX = append(spec.SFX, "effect\\ef_fireball.wav")
			spec.SFXDelays = append(spec.SFXDelays, time.Duration(tick)*EffectFrameDuration)
		}
	case effectSoulStrike:
		spec.Duration = 46 * EffectFrameDuration
		spec.Components[0].SpriteFile = "particle1"
		spec.SFX = []string{"effect\\ef_soulstrike.wav"}
		spec.SFXDelays = []time.Duration{6 * EffectFrameDuration}
	case effectFrostDiver:
		spec.Components[0].TextureFile = "effect/ice.tga"
	}
	return spec
}

func mageFrostHitEffectSpec() EffectSpec {
	spec := funcEffectSpec("FrostDiverHit", 40*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/ice.tga"
	spec.SFX = []string{"effect\\ef_frostdiver2.wav"}
	return spec
}

func mageBoltEffectSpec(fire bool) EffectSpec {
	spec := funcEffectSpec("MageBolt", 82*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\ef_icearrow%d.wav"}
	spec.SFXRandMin, spec.SFXRandMax = 1, 3
	spec.SFXDelays = []time.Duration{12 * EffectFrameDuration}
	spec.Components[0].TextureFiles = []string{"effect/icearrow.tga"}
	spec.Components[0].TextureFile = "effect/ring_blue.tga"
	if fire {
		spec.Duration = 160 * EffectFrameDuration
		spec.Components[0].SpriteFile = "particle4"
		spec.SFX[0] = "effect\\ef_firearrow%d.wav"
		spec.Components[0].TextureFiles = []string{
			"effect/불화살1.tga", "effect/불화살2.tga", "effect/불화살3.tga",
			"effect/불화살4.tga", "effect/불화살5.tga", "effect/불화살6.tga",
		}
		spec.Components[0].TextureFile = "effect/ring_yellow.tga"
	}
	return spec
}

func mageSightEffectSpec() EffectSpec {
	spec := funcEffectSpec("Sight", 12*time.Second, true)
	spec.Components[0].SpriteFile = "sight"
	shadow := spec.Components[0]
	shadow.SpriteFile = "data/sprite/shadow"
	shadow.ShadowTexture = true
	spec.Components = append(spec.Components, shadow)
	spec.SFX = []string{"effect\\ef_sight.wav"}
	return spec
}

func safetyWallEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 9999 * EffectFrameDuration,
		SFX:      []string{"effect\\ef_glasswall.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSTR,
				STRFile:        "safetywall",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "SafetyWall",
				TextureFile:    "effect/alpha_down.tga",
				AttachedEntity: true,
			},
		},
	}
}

func glassWallEffectSpec() EffectSpec {
	return strEffectSpec("effect/safetywall", "effect\\ef_glasswall.wav")
}

func napalmBeatEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 700 * time.Millisecond,
		SFX:      []string{"effect\\ef_napalmbeat.wav"},
	}
}

func soulStrikeEffectSpec() EffectSpec {
	return mageProjectileEffectSpec(effectSoulStrike)
}

func stoneCurseEffectSpec() EffectSpec {
	return strEffectSpecAttached("stonecurse", "", false)
}

func coldBoltEffectSpec() EffectSpec {
	return mageBoltEffectSpec(false)
}

func fireBoltEffectSpec() EffectSpec {
	return mageBoltEffectSpec(true)
}

func fireBallEffectSpec() EffectSpec {
	return mageProjectileEffectSpec(effectFireBall)
}

func fireWallEffectSpec() EffectSpec {
	return strEffectSpecRandom("firewall%d", "effect\\ef_firewall.wav", 1, 2)
}

func frostDiverEffectSpec() EffectSpec {
	return mageProjectileEffectSpec(effectFrostDiver)
}

func lightningBoltEffectSpec() EffectSpec {
	return strEffectSpecAttached("lightning", "", false)
}

func thunderStormEffectSpec() EffectSpec {
	return strEffectSpec("thunderstorm", "effect\\magician_thunderstorm.wav")
}

func energyCoatEffectSpec() EffectSpec {
	return strEffectSpecAttached("energycoat", "", false)
}
