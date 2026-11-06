package maps

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

type rswEffectEmitter struct {
	next, initialized time.Time
	interval          time.Duration
	active            []effects.Instance
	spec              effects.Spec
}

func (e *rswEffectEmitter) emit(source res.RSWEffect, index int, now time.Time, visible bool) time.Time {
	if e.initialized.IsZero() {
		e.initialized = now
		seed := effects.Instance{EffectID: int(source.ID), ActorID: uint32(index + 1), Starts: now}
		frames := float64(source.Delay)
		if math.IsNaN(frames) || math.IsInf(frames, 0) || frames < 1 {
			frames = 1
		}
		// RSW intervals are tick counts, not milliseconds. Fireflies also jitter
		// their period and initial phase; other emitters start within 24 ticks.
		frames = math.Min(frames, math.MaxInt32-1)
		initial := effects.HitEffectRandom(seed, 0, 24)
		if int(source.ID) == effects.Firefly {
			process := effects.HitEffectRandom(seed, 1, int(frames)+1)
			frames += float64(effects.HitEffectRandom(seed, 2, max(1, int(frames/10)))) - frames/20
			initial = max(0, int(frames)-process)
		}
		e.interval = time.Duration(max(1, int64(frames))) * db.EffectFrameDuration
		e.next = now.Add(time.Duration(initial) * db.EffectFrameDuration)
	}
	if !visible || now.Before(e.next) {
		return time.Time{}
	}
	// Resume with one emission after being out of range or a stalled frame.
	// Never replay a backlog of particles the player could not see.
	starts := e.next
	if now.Sub(starts) >= e.interval {
		starts = now
	}
	e.next = starts.Add(e.interval)
	return starts
}

func (m *System) DrawRSWEffects(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, fx func() *effects.System) {
	if !effects.Enabled(ctx) {
		m.rswEffectSource, m.rswEffects = nil, nil
		return
	}
	if screen == nil || ctx.World == nil || ctx.World.RSW == nil || ctx.World.GND == nil {
		return
	}
	if m.rswEffectSource != ctx.World.RSW {
		m.rswEffectSource = ctx.World.RSW
		m.rswEffects = make([]rswEffectEmitter, len(ctx.World.RSW.Effects))
		specs := make(map[int]effects.Spec)
		for i, source := range ctx.World.RSW.Effects {
			id := int(source.ID)
			spec, cached := specs[id]
			if !cached {
				spec, _ = effects.Lookup(id)
				specs[id] = spec
			}
			m.rswEffects[i].spec = spec
		}
	}
	for index, source := range ctx.World.RSW.Effects {
		emitter := &m.rswEffects[index]
		spec := emitter.spec
		if len(spec.Components) == 0 {
			continue
		}
		x, y, z := rswEffectWorldPosition(ctx.World.GND, source)
		visible := math.Hypot(x-projection.PlayerX, y-projection.PlayerY) < 26
		if starts := emitter.emit(source, index, now, visible); !starts.IsZero() {
			effect := effects.Instance{EffectID: int(source.ID), Params: source.Param, ActorID: uint32(index + 1),
				X: int(math.Round(x)), Y: int(math.Round(y)), Starts: starts}
			duration := spec.Duration
			for _, component := range spec.Components {
				duration = max(duration, fx().WorldEffectResolvedComponentDuration(ctx, spec, component, effect))
			}
			effect.Expires, effect.Duration = starts.Add(duration), duration
			fx().PrepareEnvironmentEffect(&effect, projection)
			emitter.active = append(emitter.active, effect)
		}
		active := emitter.active[:0]
		for _, effect := range emitter.active {
			if !now.Before(effect.Expires) {
				continue
			}
			active = append(active, effect)
			if !visible {
				continue
			}
			for componentIndex, component := range spec.Components {
				component = MapEffectComponent(component)
				duration := fx().WorldEffectResolvedComponentDuration(ctx, spec, component, effect)
				progress := effects.WorldEffectComponentProgress(effect.Starts, duration, now)
				if progress < 1 {
					fx().DrawWorldEffectComponent(screen, ctx, projection, effect, component, componentIndex, x, y, z, progress, duration, now)
				}
			}
		}
		clear(emitter.active[len(active):])
		emitter.active = active
	}
}

func MapEffectComponent(component effects.Component) effects.Component {
	if component.Kind == effects.Component3D && component.SpriteFile != "" {
		component.WorldSizedSprite = true
	}
	return component
}

func rswEffectWorldPosition(gnd *res.GND, effect res.RSWEffect) (float64, float64, float64) {
	if gnd == nil {
		return float64(effect.Position.X), float64(effect.Position.Z), float64(effect.Position.Y)
	}
	return float64(effect.Position.X) + float64(gnd.Width), float64(effect.Position.Z) + float64(gnd.Height), float64(effect.Position.Y)
}
