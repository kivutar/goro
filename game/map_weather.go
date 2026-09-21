package game

import (
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

type mapWeatherKind int

const (
	mapWeatherNone mapWeatherKind = iota
	mapWeatherLoopingEffect
	mapWeatherFireworks
)

const (
	pokJukLaunchSFX        = "effect\\폭죽.wav"
	pokJukExplosionSFX     = "effect\\itempokjuk.wav"
	pokJukWeatherFireworks = 4
)

func mapWeatherForMap(name string) mapWeatherKind {
	if mapWeatherEffectIDForMap(name) != 0 {
		return mapWeatherLoopingEffect
	}
	if normalizeMapNameForWeather(name) == "comodo.rsw" {
		return mapWeatherFireworks
	}
	return mapWeatherNone
}

func mapWeatherEffectIDForMap(name string) int {
	switch normalizeMapNameForWeather(name) {
	case "xmas.rsw":
		return effectSnow
	case "yuno.rsw":
		return effectCloud2
	case "airplane.rsw", "airplane_01.rsw":
		return effectCloud5
	case "einbroch.rsw":
		return effectCloud4
	default:
		return 0
	}
}

func normalizeMapNameForWeather(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if index := strings.LastIndexAny(name, `\/`); index >= 0 {
		name = name[index+1:]
	}
	if strings.HasSuffix(name, ".gat") {
		name = strings.TrimSuffix(name, ".gat") + ".rsw"
	}
	if name != "" && !strings.Contains(name, ".") {
		name += ".rsw"
	}
	return name
}

func (m *WorldMode) drawMapWeatherEffects(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
	if screen == nil || ctx.World == nil || ctx.World.GND == nil {
		return
	}
	kind := mapWeatherForMap(ctx.World.MapName)
	effectID := mapWeatherEffectIDForMap(ctx.World.MapName)
	switch kind {
	case mapWeatherFireworks:
		m.mapWeatherCloud.reset()
		m.drawFireworksWeather(screen, ctx, projection, now)
	case mapWeatherLoopingEffect:
		m.mapWeatherPokJuk.reset()
		if _, ok := weatherCloudParamsForEffect(effectID); ok {
			m.drawMapWeatherCloudEffect(screen, ctx, projection, effectID, now)
			return
		}
		m.mapWeatherCloud.reset()
		m.drawLoopingMapWeatherEffect(screen, ctx, projection, effectID, now)
	default:
		m.mapWeatherCloud.reset()
		m.mapWeatherPokJuk.reset()
	}
}

func (m *WorldMode) drawLoopingMapWeatherEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effectID int, now time.Time) {
	spec, ok := worldEffectSpecForID(effectID)
	if !ok {
		return
	}
	starts := loopingMapWeatherEffectStart(effectID, spec.duration, now)
	effect := worldEffect{
		effectID: effectID,
		actorID:  uint32(effectID),
		starts:   starts,
		expires:  now.Add(24 * time.Hour),
		duration: spec.duration,
		x:        int(math.Round(projection.playerX)),
		y:        int(math.Round(projection.playerY)),
	}
	worldX := projection.playerX
	worldY := projection.playerY
	worldZ := terrainHeightAt(ctx.World, worldX-0.5, worldY-0.5) + 1
	for componentIndex, component := range spec.components {
		duration := m.worldEffectResolvedComponentDuration(ctx, spec, component)
		if duration <= 0 {
			duration = spec.duration
		}
		componentStarts := loopingMapWeatherEffectStart(effectID+componentIndex*17, duration, now)
		effect.starts = componentStarts
		progress := worldEffectComponentProgress(componentStarts, duration, now)
		m.drawWorldEffectComponent(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, progress, duration, now)
	}
}

func (m *WorldMode) drawFireworksWeather(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
	key := normalizeMapNameForWeather(ctx.World.MapName)
	centerX, centerY := mapWeatherFireworkCenter(ctx.World, projection, now)
	m.mapWeatherPokJuk.ensure(key, ctx.World, centerX, centerY, now)
	m.mapWeatherPokJuk.update(ctx.World, centerX, centerY, now)
	m.mapWeatherPokJuk.scheduleSounds(m, now)
	m.mapWeatherPokJuk.draw(screen, m, ctx, projection, now)
}

func (m *WorldMode) scheduleMapWeatherSound(index int, starts time.Time, now time.Time, paths ...string) {
	elapsed := now.Sub(starts)
	if elapsed < 0 {
		return
	}
	if m.mapWeatherSounds == nil {
		m.mapWeatherSounds = make(map[int]time.Time)
	}
	if m.mapWeatherSounds[index].Equal(starts) {
		return
	}
	m.mapWeatherSounds[index] = starts
	if elapsed > 120*time.Millisecond {
		return
	}
	m.scheduleSound(now, paths...)
}

func loopingMapWeatherEffectStart(index int, duration time.Duration, now time.Time) time.Time {
	if duration <= 0 {
		return now
	}
	phase := time.Duration(index) * duration / 3
	origin := time.Unix(0, 0).Add(phase)
	elapsed := now.Sub(origin)
	if elapsed < 0 {
		return origin
	}
	return now.Add(-(elapsed % duration))
}
