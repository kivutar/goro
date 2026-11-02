package db

import "time"

func mvpEffectSpec() EffectSpec {
	return strEffectSpecAttached("mvp", "effect\\st_mvp.wav", false)
}

func baseLevelUpEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1300 * time.Millisecond,
		SFX:      []string{"levelup.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "angel",
			AttachedEntity: true,
		}},
	}
}

func jobLevelUpEffectSpec() EffectSpec {
	return EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "joblvup",
			AttachedEntity: true,
		}},
	}
}

func jobLvUp50EffectSpec() EffectSpec {
	spec := strEffectSpecAttached("joblvup", "", false)
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "LevelUpBurst", TextureFile: "effect/magic_green.tga",
		Duration: 33 * EffectFrameDuration, AttachedEntity: true,
	})
	return spec
}

func superAngelEffectSpec() EffectSpec {
	spec := EffectSpec{Duration: 200 * EffectFrameDuration}
	for layer, sprite := range []string{"천사", "천사날개", "깃털1", "깃털2"} {
		spec.Components = append(spec.Components, EffectComponent{
			Kind: EffectComponentFUNC, FuncName: "SuperAngel", SpriteFile: sprite, SpriteFrame: layer,
		})
	}
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "LevelUpBurst", TextureFile: "effect/ring_blue.tga",
		AttachedEntity: true,
	})
	return spec
}

func achCompleteEffectSpec() EffectSpec {
	return strEffectSpecAttachedTexturePath("ach_complete/ppring3", "ach_complete/", "", false)
}
