package db

import "time"

func gunslingerSpriteEffectSpec(id int) EffectSpec {
	spec := funcEffectSpec("NativeSprite", 100*EffectFrameDuration, true)
	c := &spec.Components[0]
	c.WorldSizedSprite, c.Overlay = true, true
	c.SpriteDelay = 2 * EffectFrameDuration
	switch id {
	case EffectM02:
		c.SpriteFile, c.SpriteDelay = "m_ef02", 4*EffectFrameDuration
		spec.SFX, spec.SFXDelays = []string{"effect\\풀버스터.wav"}, []time.Duration{5 * EffectFrameDuration}
	case EffectDesperado:
		c.SpriteFile, c.SpriteDelay = "데스페라도", 4*EffectFrameDuration
		spec.SFX, spec.SFXDelays = []string{"effect\\데스페라도.wav"}, []time.Duration{10 * EffectFrameDuration}
	case EffectLightningS, EffectBlindS, EffectPoisonS, EffectFreezingS, EffectFlareS:
		c.SpriteFile = [...]string{"라이트닝스피어", "블라인드스피어", "포이즌스피어", "프리징스피어", "플레어스피어"}[id-EffectLightningS]
		c.PosZ, c.SpriteRepeat = .2, true
		c.Overlay, c.AttachedEntity = false, false
		spec.Duration = 23 * EffectFrameDuration
	case EffectRapidShower:
		c.SpriteFile, c.SpriteDelay = "래피드샤워", 4*EffectFrameDuration
		spec.SFX, spec.SFXDelays = []string{"effect\\래피드샤워.wav"}, []time.Duration{20 * EffectFrameDuration}
	case EffectMagicalBullet:
		c.SpriteFile, spec.SFX = "매지컬불릿", []string{"effect\\매지컬블릿.wav"}
	case EffectSpreadAttack:
		c.SpriteFile, c.SpriteDelay = "스프레드", 4*EffectFrameDuration
	case EffectTracking:
		c.SpriteFile = "트래킹"
	case EffectTripleAction:
		c.SpriteFile, c.Delay = "트리플액션", 30*EffectFrameDuration
		spec.SFX, spec.SFXDelays = []string{"effect\\트리플액션.wav"}, []time.Duration{25 * EffectFrameDuration}
	}
	return spec
}

func gunslingerAgilityEffectSpec() EffectSpec {
	spec := agilityEffectSpec(false)
	spec.Components[0].TextureFile = ""
	return spec
}

func piercingShotEffectSpec() EffectSpec {
	spec := taekwonDashEffectSpec()
	spec.Duration = 120 * EffectFrameDuration
	spec.Components[0].TextureFile = "effect/line3y.tga"
	spec.SFX, spec.SFXDelays = []string{"effect\\피어싱샷.wav"}, []time.Duration{28 * EffectFrameDuration}
	return spec
}

func disarmEffectSpec() EffectSpec {
	spec := funcEffectSpec("FullStrip", 110*EffectFrameDuration, false)
	spec.Components[0].TextureFiles = []string{
		"유저인터페이스/item/골드룩스.bmp", "유저인터페이스/item/더개리슨.bmp",
		"유저인터페이스/item/더블샷건.bmp", "유저인터페이스/item/더스크.bmp",
	}
	spec.SFX, spec.SFXDelays = []string{"effect\\디스암.wav"}, []time.Duration{21 * EffectFrameDuration}
	return spec
}

func gunslingerCoinSoundEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\플립.wav")
}

func redHitEffectSpec() EffectSpec {
	return funcEffectSpec("RedHit", 60*EffectFrameDuration, true)
}

func madnessBlueEffectSpec() EffectSpec {
	return funcEffectSpec("BlueHit", 64*EffectFrameDuration, true)
}

func madnessRedEffectSpec() EffectSpec {
	return funcEffectSpec("RedHit", 64*EffectFrameDuration, true)
}

func m02EffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectM02)
}

func bash3D5EffectSpec() EffectSpec {
	return bashRayEffectSpec("effect\\더스트샷.wav", 0)
}

func desperadoEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectDesperado)
}

func lightningSEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectLightningS)
}

func blindSEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectBlindS)
}

func poisonSEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectPoisonS)
}

func freezingSEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectFreezingS)
}

func flareSEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectFlareS)
}

func rapidShowerEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectRapidShower)
}

func magicalBulletEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectMagicalBullet)
}

func spreadAttackEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectSpreadAttack)
}

func trackCastingEffectSpec() EffectSpec {
	return strEffectSpecAttached("트랙킹", "", false)
}

func trackingEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectTracking)
}

func tripleActionEffectSpec() EffectSpec {
	return gunslingerSpriteEffectSpec(EffectTripleAction)
}

func bullseyeEffectSpec() EffectSpec {
	return strEffectSpecAttached("불스아이", "", false)
}
