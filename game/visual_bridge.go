package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) effects() *effects.System {
	if m.mapScene.Textures == nil {
		m.mapScene.Textures = make(map[string]*render.Image)
	}
	if m.mapScene.TextureMiss == nil {
		m.mapScene.TextureMiss = make(map[string]struct{})
	}
	m.effectSystem.Bind(m, m.mapScene.Textures, m.mapScene.TextureMiss)
	return &m.effectSystem
}

func (m *WorldMode) EffectSound(at time.Time, effect effects.Instance, paths ...string) {
	m.scheduleSoundAtWorldEffect(at, effect, paths...)
}

func (m *WorldMode) EffectSoundAtPosition(at time.Time, volume, x, y float64, paths ...string) {
	m.scheduleSoundVolumeAtPosition(at, volume, x, y, paths...)
}

func (m *WorldMode) EffectSoundGlobal(at time.Time, paths ...string) { m.scheduleSound(at, paths...) }

func (m *WorldMode) EffectShake(at time.Time, duration time.Duration) {
	m.startCameraShake(at, duration)
}

func (m *WorldMode) EffectStarted(ctx client.Context, effect effects.Instance, at time.Time) {
	m.applyWorldEffectSideEffects(ctx, effect, at)
}

func (m *WorldMode) EffectCameraYaw(ctx client.Context) float64 {
	yaw := scene.CameraYawForMap(ctx)
	if !scene.CameraRotationLockedForMap(ctx) {
		yaw += m.camera.yawOffset
	}
	return yaw
}

func (m *WorldMode) EffectSpriteTexture(image *render.Image) *render.Image {
	return m.spriteTexture(image)
}

func (m *WorldMode) spriteTexture(src *render.Image) *render.Image {
	if !m.mapScene.UpscaleTextures || src == nil {
		return src
	}
	if m.spriteUpscaler == nil {
		m.spriteUpscaler = sprites.NewUpscaler()
	}
	return m.spriteUpscaler.Texture(src, m.mapScene.Images.Own)
}

func (m *WorldMode) applySpecialEffectNotify(ctx client.Context, notify network.SpecialEffectNotify) {
	effectID := effects.SpecialEffectID(notify.EffectID)
	if effectID <= 0 {
		return
	}
	if scene.IsLocalActor(ctx, notify.AID) {
		switch notify.EffectID {
		case network.SpecialEffectBaseLevelUp,
			network.SpecialEffectSuperNoviceBaseLevelUp,
			network.SpecialEffectTaekwonBaseLevelUp:
			m.ui.levelUpNotifications.NotifyBase()
		case network.SpecialEffectJobLevelUp,
			network.SpecialEffectSuperNoviceJobLevelUp:
			m.ui.levelUpNotifications.NotifyJob()
		}
	}
	if effectID == effects.NPCStop2Del {
		m.effects().Remove(effects.NPCStop2, notify.AID)
		return
	}
	switch notify.EffectID {
	case network.SpecialEffectBaseLevelUp, network.SpecialEffectSuperNoviceBaseLevelUp,
		network.SpecialEffectJobLevelUp, network.SpecialEffectSuperNoviceJobLevelUp, network.SpecialEffectTaekwonBaseLevelUp:
		m.addLevelUpEffect(ctx, notify.AID, effectID)
		return
	}
	spec, _ := effects.Lookup(effectID)
	if effects.CraftingResultSpecialEffect(notify.EffectID) {
		m.effects().ReplaceCraftingResultEffect(ctx, notify, effectID, spec)
		return
	}
	// Each Kaupe dodge is a separate hit, even while an earlier dodge fades.
	if effectID != effects.StormKick4 && m.effects().HasActive(effectID, notify.AID, time.Now()) {
		return
	}
	if m.effects().Add(ctx, effectID, notify.AID) {
		glog.Debugf("special effect actor=%d special=%d effect=%d sfx=%v", notify.AID, notify.EffectID, effectID, spec.Sfx)
	}
}

func (m *WorldMode) addLevelUpEffect(ctx client.Context, actorID uint32, effectID int) {
	effectID = localLevelUpEffect(ctx, actorID, effectID)
	if scene.IsLocalActor(ctx, actorID) {
		actorID = scene.LocalSkillTarget(ctx)
	}
	ordinary, enhanced := effects.BaseLevelUp, effects.Angel2
	if effectID == effects.JobLevelUp || effectID == effects.JobLvUp50 {
		ordinary, enhanced = effects.JobLevelUp, effects.JobLvUp50
	}
	// Parameter updates and effect notifications may arrive in either order.
	if effectID == ordinary && m.effects().HasActive(enhanced, actorID, time.Now()) {
		return
	}
	if effectID == enhanced {
		m.effects().Remove(ordinary, actorID)
	}
	if m.effects().AddIfMissing(ctx, effectID, actorID) && effectID == effects.Angel2 && scene.IsLocalActor(ctx, actorID) {
		// The angel itself is silent; only the local level-up plays this sound.
		m.scheduleSoundAtActor(time.Now(), actorID, "levelup.wav")
	}
}

func localLevelUpEffect(ctx client.Context, actorID uint32, effectID int) int {
	if ctx.Session == nil || !scene.IsLocalActor(ctx, actorID) {
		return effectID
	}
	switch effectID {
	case effects.BaseLevelUp:
		if ctx.Session.Selected.Job == db.JobSuperNovice || ctx.Session.Selected.Job == db.JobSuperNoviceB {
			return effects.Angel2
		}
	case effects.JobLevelUp:
		if ctx.Session.Progress.JobLevel > 50 {
			return effects.JobLvUp50
		}
	}
	return effectID
}

func (m *WorldMode) applyMVPNotify(ctx client.Context, notify network.MVPNotify) {
	if m.effects().Add(ctx, effects.Mvp, notify.AID) {
		glog.Debugf("mvp effect actor=%d effect=%d", notify.AID, effects.Mvp)
	}
}
