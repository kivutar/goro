package actors

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

// applyActorStateSnapshot updates state shared by the login and world modes.
// Rendering transitions remain the responsibility of WorldMode.
func ApplyActorStateSnapshot(ctx client.Context, change network.ActorStateChange) (oldState uint32, local, ok bool) {
	if change.ID == 0 || ctx.World == nil {
		return 0, false, false
	}
	if scene.IsLocalActor(ctx, change.ID) {
		oldState = ctx.World.Player.EffectState
		setActorRenderState(&ctx.World.Player, change.BodyState, change.HealthState, change.EffectState)
		setSelectedCharacterOptionBit(ctx, db.EffectStateWedding, change.EffectState&db.EffectStateWedding != 0)
		return oldState, true, true
	}
	actor, ok := ctx.World.Actors[change.ID]
	if !ok {
		return 0, false, false
	}
	oldState = actor.EffectState
	setActorRenderState(&actor, change.BodyState, change.HealthState, change.EffectState)
	UpsertActor(ctx, actor)
	return oldState, false, true
}

func setActorRenderState(actor *worldstate.Actor, bodyState, healthState uint16, effectState uint32) {
	actor.BodyState = bodyState
	actor.HealthState = healthState
	actor.EffectState = effectState
	actor.HasState = true
	applyActorCartStateFromEffect(actor)
}

func applyActorBodyState(actor worldstate.Actor, state *sprites.State) {
	switch actor.BodyState {
	case db.BodyStateFreeze:
		state.ActionFamily = freezeActionFamily(actor)
		state.Moving = false
		state.Loop = false
		state.LoopIdle = false
		state.Play = false
		state.HasPlay = true
		state.FixedMotion = 0
		state.HasFixedMotion = true
	case db.BodyStateStone:
		state.Moving = false
		state.Loop = false
		state.LoopIdle = false
		state.Play = false
		state.HasPlay = true
	}
}

func ActorStateTint(actor worldstate.Actor) color.RGBA {
	r, g, b, a := 1.0, 1.0, 1.0, 1.0
	switch actor.BodyState {
	case db.BodyStateStone:
		r, g, b = 0.1, 0.1, 0.1
	case db.BodyStateStonewait:
		r, g, b = 0.3, 0.3, 0.3
	case db.BodyStateFreeze:
		r, g, b = 0.0, 0.4, 0.8
	}
	if actor.HealthState&db.HealthStateCurse != 0 {
		r *= 0.5
		g *= 0.15
		b *= 0.1
	}
	if actor.HealthState&db.HealthStatePoison != 0 {
		r *= 0.9
		g *= 0.4
		b *= 0.8
	}
	if actor.HealthState&db.HealthStateBlind != 0 {
		r *= 0.2
		g *= 0.2
		b *= 0.2
	}
	opt3 := actor.Opt3State
	if scene.IsMonsterLikeHoverActor(actor) {
		opt3 &^= db.Opt3Energycoat | db.Opt3Overthrust // Monsters use these bits for Eske and Eswoo.
	}
	vr, vg, vb, va := actorOpt3StateTint(opt3)
	r *= vr
	g *= vg
	b *= vb
	a *= va
	return color.RGBA{R: byte(scene.ClampUnit(r) * 255), G: byte(scene.ClampUnit(g) * 255), B: byte(scene.ClampUnit(b) * 255), A: byte(scene.ClampUnit(a) * 255)}
}

func actorOpt3StateTint(opt3State uint32) (float64, float64, float64, float64) {
	r, g, b, a := 1.0, 1.0, 1.0, 1.0
	if opt3State&db.Opt3Quicken != 0 {
		b = 0
	}
	if opt3State&db.Opt3Energycoat != 0 {
		r, g, b = 0.5, 0.5, 0.85
	}
	if opt3State&db.Opt3Bunsin != 0 {
		r, g, b = 155.0/255, 155.0/255, 1
	}
	if opt3State&db.Opt3Overthrust != 0 {
		r, g, b = 250.0/255, 150.0/255, 150.0/255
	}
	if opt3State&db.Opt3Bladestop != 0 {
		r, g, b = 64.0/255, 64.0/255, 64.0/255
	}
	return r, g, b, a
}

func (m *System) actorRenderTint(actor worldstate.Actor, now time.Time) color.RGBA {
	return m.ActorBodyColorTint(actor.ID, ActorStateTint(actor), now)
}

func (m *System) playerRenderTint(ctx client.Context, actor worldstate.Actor, now time.Time) color.RGBA {
	tint := m.actorRenderTint(actor, now)
	if ctx.Session == nil {
		return tint
	}
	if ctx.Session.AccountID != actor.ID {
		tint = m.ActorBodyColorTint(ctx.Session.AccountID, tint, now)
	}
	if ctx.Session.CharID != actor.ID && ctx.Session.CharID != ctx.Session.AccountID {
		tint = m.ActorBodyColorTint(ctx.Session.CharID, tint, now)
	}
	return tint
}

func (m *System) ActorRenderBlend(actorID uint32, now time.Time) render.Blend {
	if actorID != 0 {
		for _, effect := range m.bindings.Effects.Active {
			if effect.ActorID != actorID || now.Before(effect.Starts) || effect.Expired(now) {
				continue
			}
			elapsed := now.Sub(effect.Starts)
			if effects.SniperBodyFlashActive(effect.EffectID, int(elapsed/db.EffectFrameDuration)) {
				return render.BlendLighter // BL_LIGHT_BODY selects RF_EFFECT in the original client.
			}
		}
	}
	return render.BlendSourceOver
}

func (m *System) playerRenderBlend(ctx client.Context, actor worldstate.Actor, now time.Time) render.Blend {
	if ctx.Session != nil {
		for _, id := range []uint32{ctx.Session.AccountID, ctx.Session.CharID} {
			if blend := m.ActorRenderBlend(id, now); blend != render.BlendSourceOver {
				return blend
			}
		}
	}
	return m.ActorRenderBlend(actor.ID, now)
}

func (m *System) ActorBodyColorTint(actorID uint32, tint color.RGBA, now time.Time) color.RGBA {
	if actorID == 0 {
		return tint
	}
	base := tint
	for _, effect := range m.bindings.Effects.Active {
		if effect.ActorID != actorID || now.Before(effect.Starts) || effect.Expired(now) {
			continue
		}
		switch effect.EffectID {
		case effects.Gumgang:
			if effect.Persistent {
				tint = effects.FuryBodyTint(base, effect, now)
			}
		case effects.Portal4:
			if effects.SniperBodyFlashActive(effect.EffectID, int(now.Sub(effect.Starts)/db.EffectFrameDuration)) {
				tint = scene.MultiplyTint(base, 220.0/255, 250.0/255, 220.0/255, 1)
			}
		case effects.Shrink:
			frame := now.Sub(effect.Starts) / db.EffectFrameDuration
			if frame >= 5 && frame < 26 && frame%2 == 0 {
				tint = scene.MultiplyTint(base, 250.0/255, 250.0/255, 100.0/255, 1)
			}
		case effects.Bash3D:
			frame := now.Sub(effect.Starts) / db.EffectFrameDuration
			if frame >= 20 && frame <= 40 {
				tint = scene.MultiplyTint(base, 1, 200.0/255, 200.0/255, 1)
			}
		case effects.BodyColor:
			tint = effectBodyColorTint(tint, now.Sub(effect.Starts))
		case effects.MagicCrasher2:
			tint = magicCrasher2BodyColorTint(tint, actorID, effect.Starts, now.Sub(effect.Starts))
		case effects.MagicCrasher:
			frame := now.Sub(effect.Starts) / db.EffectFrameDuration
			if effects.MagicCrasherFlashActive(now.Sub(effect.Starts)) {
				seed := uint64(actorID)*0x9e3779b185ebca87 ^ uint64(effect.Starts.UnixNano()) ^ uint64(frame)*0xc2b2ae3d27d4eb4f
				tint = scene.MultiplyTint(base, randomBodyColorChannel(seed), randomBodyColorChannel(seed+1), randomBodyColorChannel(seed+2), 1)
			}
		case effects.TransBlueBody:
			frame := now.Sub(effect.Starts) / db.EffectFrameDuration
			if frame < 200 {
				channel := float64(205-frame) / 255
				tint = scene.MultiplyTint(base, channel, channel, 1, 1)
			}
		}
	}
	return tint
}

func effectBodyColorTint(tint color.RGBA, elapsed time.Duration) color.RGBA {
	alpha := scene.ClampFloat(float64(elapsed)/float64(100*time.Millisecond), 0, 1)
	return color.RGBA{
		R: mixTintChannel(tint.R, 255, alpha),
		G: mixTintChannel(tint.G, 0, alpha),
		B: mixTintChannel(tint.B, 0, alpha),
		A: tint.A,
	}
}

func magicCrasher2BodyColorTint(tint color.RGBA, actorID uint32, starts time.Time, elapsed time.Duration) color.RGBA {
	if elapsed < 0 || elapsed/db.EffectFrameDuration > 60 {
		return tint
	}
	bucket := uint64(elapsed / db.EffectFrameDuration)
	seed := uint64(actorID)*0x9e3779b185ebca87 ^ uint64(starts.UnixNano()) ^ bucket*0xc2b2ae3d27d4eb4f
	return scene.MultiplyTint(tint, randomBodyColorChannel(seed), randomBodyColorChannel(seed+1), randomBodyColorChannel(seed+2), 1)
}

func randomBodyColorChannel(seed uint64) float64 {
	seed ^= seed >> 33
	seed *= 0xff51afd7ed558ccd
	seed ^= seed >> 33
	seed *= 0xc4ceb9fe1a85ec53
	seed ^= seed >> 33
	return float64(byte(seed>>56)) / 255
}

func mixTintChannel(from, to uint8, alpha float64) uint8 {
	value := float64(from)*(1-alpha) + float64(to)*alpha
	return uint8(scene.ClampFloat(value, 0, 255))
}

func (m *System) actorRenderScale(actorID uint32, baseScale float64, now time.Time) float64 {
	return baseScale * m.ActorBodySizeMultiplier(actorID, now)
}

func (m *System) PlayerRenderScale(ctx client.Context, actor worldstate.Actor, baseScale float64, now time.Time) float64 {
	if scale := m.playerBodyRenderScale(actor.ID, actor.Job, baseScale, now); scale != baseScale {
		return scale
	}
	if ctx.Session == nil {
		return baseScale
	}
	if ctx.Session.AccountID != actor.ID {
		if scale := m.playerBodyRenderScale(ctx.Session.AccountID, actor.Job, baseScale, now); scale != baseScale {
			return scale
		}
	}
	if ctx.Session.CharID != actor.ID && ctx.Session.CharID != ctx.Session.AccountID {
		if scale := m.playerBodyRenderScale(ctx.Session.CharID, actor.Job, baseScale, now); scale != baseScale {
			return scale
		}
	}
	return baseScale
}

func (m *System) playerBodyRenderScale(actorID uint32, job int16, baseScale float64, now time.Time) float64 {
	bodyScale := playerBodyScaleForJob(job)
	multiplier := m.actorBodySizeMultiplierFrom(actorID, now, bodyScale)
	if multiplier == bodyScale {
		return baseScale
	}
	return baseScale / bodyScale * multiplier
}

func (m *System) ActorBodySizeMultiplier(actorID uint32, now time.Time) float64 {
	return m.actorBodySizeMultiplierFrom(actorID, now, 1)
}

func (m *System) actorBodySizeMultiplierFrom(actorID uint32, now time.Time, baseMultiplier float64) float64 {
	if actorID == 0 {
		return baseMultiplier
	}
	multiplier := baseMultiplier
	var latest time.Time
	for _, effect := range m.bindings.Effects.Active {
		if effect.ActorID != actorID || now.Before(effect.Starts) || effect.Expired(now) {
			continue
		}
		if effect.Starts.Before(latest) {
			continue
		}
		if effect.EffectID == effects.BabyBodyBack || (effect.EffectID == effects.BabyBody && effect.Persistent) {
			frame := float64(now.Sub(effect.Starts) / db.EffectFrameDuration)
			if effect.EffectID == effects.BabyBodyBack {
				frame = 45 - frame*3
			}
			multiplier = baseMultiplier * (.75 + .25*math.Cos(scene.DegreesToRadians(scene.ClampFloat(frame, 0, 45)*4)))
			latest = effect.Starts
			continue
		}
		targetSize, transition, ok := actorBodySizeTarget(effect.EffectID)
		if !ok {
			continue
		}
		targetMultiplier := targetSize / 5
		if transition {
			duration := effect.Duration
			if duration <= 0 {
				duration = 300 * time.Millisecond
			}
			progress := scene.ClampFloat(float64(now.Sub(effect.Starts))/float64(duration), 0, 1)
			targetMultiplier = baseMultiplier + (targetMultiplier-baseMultiplier)*progress
		}
		multiplier = targetMultiplier
		latest = effect.Starts
	}
	return multiplier
}

func actorBodySizeTarget(effectID int) (targetSize float64, transition bool, ok bool) {
	switch effectID {
	case effects.BabyBody:
		return 2.5, true, true
	case effects.BabyBody2:
		return 2.5, false, true
	case effects.GiantBody:
		return 7.5, true, true
	case effects.GiantBody2:
		return 7.5, false, true
	default:
		return 0, false, false
	}
}

func freezeActionFamily(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return sprites.ActionPCFreeze2
	}
	return scene.SpriteActionIdle
}
