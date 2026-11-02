package effects

import "github.com/kivutar/goro/client"

func Enabled(ctx client.Context) bool {
	if ctx.Session != nil {
		return !ctx.Session.NoEffects
	}
	return !ctx.Config.Gameplay.NoEffects
}

func effectEnabled(ctx client.Context, effectID int) bool {
	if Enabled(ctx) {
		return true
	}
	// The classic toggle keeps targeting, feedback and essential body effects.
	// Emotions are a separate actor overlay in the original client.
	switch effectID {
	case Emotion, LockOnTarget, ReadyPortalOld, ReadyPortal, Portal,
		RefineOK, RefineFail, FirePillarOn, Blind, TalkFrostJoke, TalkScream,
		BaseLevelUp, Angel2, BlueBody, RedLightBody, AsuraBody, TaeReady,
		RedHit, BlueHit, JumpBody, LandBody, QuakeBody, ReflectBody,
		Devil1, Devil2, Devil3, Devil4, Devil5, Devil6, Devil7, Devil8, Devil9, Devil10:
		return true
	}
	return false
}
