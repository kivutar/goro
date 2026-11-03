package db

import "time"

func ninjaFirePetalEffectSpec() EffectSpec {
	spec := funcEffectSpec("NinjaPetals", 110*EffectFrameDuration, true)
	spec.Components[0].SpriteFile = "sakura01"
	spec.Components = append(spec.Components, EffectComponent{Kind: EffectComponentFUNC,
		FuncName: "Sacrifice", TextureFile: "effect/alpha_center.tga", AttachedEntity: true})
	spec.SFX = []string{"effect\\EF_FireArrow%d.wav", "effect\\EF_FireArrow%d.wav", "effect\\EF_FireArrow%d.wav", "effect\\EF_FireArrow%d.wav"}
	spec.SFXRandMin, spec.SFXRandMax = 1, 3
	spec.SFXDelays = []time.Duration{12 * EffectFrameDuration, 13 * EffectFrameDuration, 14 * EffectFrameDuration, 15 * EffectFrameDuration}
	return spec
}

func ninjaIceSpearEffectSpec() EffectSpec {
	spec := earthSpikeEffectSpec()
	spec.Duration = 250 * EffectFrameDuration
	spec.Components[0].TextureFile = "effect/ice.tga"
	spec.Components[0].AttachedEntity = false
	spec.SFX = []string{"effect\\EF_IceArrow%d.wav"}
	spec.SFXRandMin, spec.SFXRandMax = 1, 3
	return spec
}

func ninjaWindEffectSpec(vertical bool) EffectSpec {
	spec := funcEffectSpec("NinjaWind", 200*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/270바람.tga"
	spec.SFX = []string{"effect\\풍인.wav"}
	if vertical {
		spec.SFX = []string{"effect\\t_바람방출.wav"}
		spec.SFXDelays = []time.Duration{EffectFrameDuration}
	}
	return spec
}

func ninjaSpriteEffectSpec(id int) EffectSpec {
	spec := funcEffectSpec("NativeSprite", 100*EffectFrameDuration, true)
	c := &spec.Components[0]
	c.WorldSizedSprite, c.Overlay = true, true
	c.SpriteDelay = 4 * EffectFrameDuration
	switch id {
	case effectKirikage:
		c.SpriteFile, spec.SFX = "그림자베기", []string{"effect\\그림자베기.wav"}
	case effectKasumikiri:
		c.SpriteFile, spec.SFX = "안개베기", []string{"effect\\안개베기.wav"}
	case effectIssen:
		c.SpriteFile, spec.SFX = "일섬", []string{"effect\\일섬.wav"}
		c.SpriteDelay, c.PosZ = 2*EffectFrameDuration, 1.2
	case effectTatami:
		c.SpriteFile, c.SpriteDelay, c.PosZ = "다다미 뒤집기", 6*EffectFrameDuration, 1.2
		c.Overlay, c.AttachedEntity = false, false
	case effectKaen:
		c.SpriteFile, c.PosZ, c.SpriteRepeat = "화염진", .2, true
		c.Overlay, c.AttachedEntity = false, false
		spec.Duration = 23 * EffectFrameDuration
	case effectThunderStorm2:
		spec.Duration = 200 * EffectFrameDuration
		c.SpriteFile, c.SpriteDelay, c.PosZ = "썬더스톰", 2*EffectFrameDuration, .9
		c.SizeStart = 250 * EffectPixelRatio
		c.SpriteRepeat = true
		spec.SFX = []string{"effect\\EF_ThunderStorm.wav"}
	}
	return spec
}

func throwItem7EffectSpec() EffectSpec {
	return throwItemFlightEffectSpec("유저인터페이스/item/수리검.bmp")
}

func throwItem8EffectSpec() EffectSpec {
	return throwItemFlightEffectSpec("유저인터페이스/item/쿠나이_독.bmp")
}

func throwItem9EffectSpec() EffectSpec {
	return throwItemFlightEffectSpec("유저인터페이스/item/풍마_열화.bmp")
}

func throwItem10EffectSpec() EffectSpec {
	return throwItemFlightEffectSpec("effect/coin_a.bmp")
}

func bottomSuitonEffectSpec() EffectSpec {
	return propertyGroundEffectSpec("ring_blue")
}

func stin4EffectSpec() EffectSpec {
	return ninjaWindEffectSpec(false)
}

func stin5EffectSpec() EffectSpec {
	return ninjaWindEffectSpec(true)
}

func thunderStorm2EffectSpec() EffectSpec {
	return ninjaSpriteEffectSpec(effectThunderStorm2)
}

func kirikageEffectSpec() EffectSpec {
	return ninjaSpriteEffectSpec(effectKirikage)
}

func tatamiEffectSpec() EffectSpec {
	return ninjaSpriteEffectSpec(effectTatami)
}

func kasumikiriEffectSpec() EffectSpec {
	return ninjaSpriteEffectSpec(effectKasumikiri)
}

func issenEffectSpec() EffectSpec {
	return ninjaSpriteEffectSpec(effectIssen)
}

func kaenEffectSpec() EffectSpec {
	spec := ninjaSpriteEffectSpec(effectKaen)
	spec.SFX = []string{"effect\\화염진.wav"}
	return spec
}

func bakuEffectSpec() EffectSpec {
	return strEffectSpec("fire dragon", "effect\\폭염룡.wav")
}

func hyousyourakuEffectSpec() EffectSpec {
	return strEffectSpec("icy", "effect\\빙정락.wav")
}
