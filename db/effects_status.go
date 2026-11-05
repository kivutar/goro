package db

import "time"

func freezeEffectSpec() EffectSpec {
	return strEffectSpecAttached("freeze", "", false)
}

func freezedEffectSpec() EffectSpec {
	return strEffectSpecAttached("freezed", "", false)
}

func iceCrashEffectSpec() EffectSpec {
	return strEffectSpecAttached("icecrash", "", false)
}

func blindEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 99999999 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "Blind",
			TextureFiles:   []string{"effect/fullb.tga", "effect/white02.bmp"},
			AttachedEntity: true,
		}},
	}
}

func poisonStatusEffectSpec() EffectSpec {
	return funcEffectSpec("Poison", 500*time.Millisecond, false)
}

func itemThunderEffectSpec() EffectSpec {
	return sprEffectSpec("item_thunder", "", true, false)
}

func itemCloudEffectSpec() EffectSpec {
	return sprEffectSpec("item_cloud", "", true, false)
}

func itemCurseEffectSpec() EffectSpec {
	return sprEffectSpec("item_curse", "", true, false)
}

func itemZZZEffectSpec() EffectSpec {
	return sprEffectSpec("item_zzz", "_snore.wav", true, false)
}

func itemRainEffectSpec() EffectSpec {
	return sprEffectSpec("item_rain", "", true, false)
}

func m01EffectSpec() EffectSpec {
	return sprEffectSpec("m_ef01", "", true, false)
}

func m03EffectSpec() EffectSpec {
	return sprEffectSpec("m_ef03", "", true, false)
}

func m04EffectSpec() EffectSpec {
	return sprEffectSpec("m_ef04", "", true, false)
}

func m06EffectSpec() EffectSpec {
	return sprEffectSpec("m_ef06", "", true, false)
}

func npcStop2DelEffectSpec() EffectSpec {
	return funcEffectSpec("RemoveCloseConfine", 2*EffectFrameDuration, true)
}
