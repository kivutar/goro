package db

import "time"

func pangVoiceEffectSpec() EffectSpec {
	return directedSpriteEffectSpec("fvoice")
}

func bottomWhistleEffectSpec() EffectSpec {
	return songMusicEffectSpec("effect/melody_b.bmp", 0)
}

func bottomSinXEffectSpec() EffectSpec {
	return songVerticalEffectSpec("effect/ring_red.tga", 1)
}

func bottomBragiEffectSpec() EffectSpec {
	return songMusicEffectSpec("", 8)
}

func bottomAppleEffectSpec() EffectSpec {
	return songMusicEffectSpec("effect/idun_apple.bmp", 6)
}

func talkFrostJokeEffectSpec() EffectSpec {
	return funcEffectSpec("FrostJokeTalk", 500*time.Millisecond, true)
}

func bottomDissonanceEffectSpec() EffectSpec {
	return songVerticalEffectSpec("effect/ring_blue.tga", 0)
}
