package effects

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
)

func (m *System) initGunslingerShot(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
	}
	effect.ActorOrigin = EffectWorldAnchor(ctx, *effect, effect.Starts)
	effect.ProjectileDirection = scene.Sub3(destination, effect.ActorOrigin)
	effect.HasActorTransform = true
	if effect.EffectID == Chemical4 {
		return
	}
	yaw := m.cameraYaw(ctx)
	effect.SpriteFrameOverride = directedSpriteAction(scene.Mul3(effect.ProjectileDirection, -1), yaw)
	if effect.EffectID == SpreadAttack {
		effect.SpriteFrameOverride *= 2
	}
	effect.HasSpriteFrame = true
}
