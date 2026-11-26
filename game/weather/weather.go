package weather

import (
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

// System owns the weather particles and sound cycles for the current map.
// Its zero value is ready to use; replace it when entering a new map.
type System struct {
	cloud     mapWeatherCloudState
	fireworks mapWeatherFireworkState
	sounds    map[int]time.Time
	// petals is the map's Sakura, Maple or Snow effect, kept across frames
	// because its particles carry state; petalsMap is the map it belongs to.
	petals    *effects.Instance
	petalsMap string
}

type mapWeatherKind int

const (
	mapWeatherNone mapWeatherKind = iota
	mapWeatherLoopingEffect
	mapWeatherFireworks
)

const pokJukLaunchSFX = "effect\\폭죽.wav"

// mapWeather picks a map's weather as CGameMode::OnInit does. Prontera and
// pay_arche cherry blossoms and Comodo fireworks only run for the Korean,
// Malaysian and Singaporean services.
func mapWeather(name, service string) (mapWeatherKind, int) {
	if effectID := mapWeatherEffectIDForMap(name); effectID != 0 {
		return mapWeatherLoopingEffect, effectID
	}
	switch service {
	case "korea", "malaysia", "singapore":
	default:
		return mapWeatherNone, 0
	}
	switch normalizeMapNameForWeather(name) {
	case "prontera.rsw", "pay_arche.rsw":
		return mapWeatherLoopingEffect, effects.Sakura
	case "comodo.rsw":
		return mapWeatherFireworks, 0
	}
	return mapWeatherNone, 0
}

func serviceType(ctx client.Context) string {
	if ctx.Resources == nil {
		return "korea"
	}
	return ctx.Resources.ClientInfo.Service()
}

func mapWeatherEffectIDForMap(name string) int {
	name = normalizeMapNameForWeather(name)
	// The 2008 client also recognizes the tower template after a three-byte
	// instance prefix. Keep the full name as the particle state's map key.
	if len(name) == len("0005@tower.rsw") && (name[3:] == "5@tower.rsw" || name[3:] == "6@tower.rsw") {
		name = name[3:]
	}
	switch name {
	case "xmas.rsw", "xmas_fild01.rsw":
		return effects.Snow
	case "gef_fild07.rsw", "mjolnir_01.rsw":
		return effects.Cloud
	case "yuno.rsw", "gonryun.rsw", "gon_dun02.rsw", "ra_temsky.rsw", "que_temsky.rsw",
		"sch_gld.rsw", "bat_fild02.rsw", "bat_b01.rsw", "bat_b02.rsw":
		return effects.Cloud2
	case "valkyrie.rsw", "rwc01.rsw", "himinn.rsw",
		"que_qsch01.rsw", "que_qsch02.rsw", "que_qsch03.rsw", "que_qsch04.rsw", "que_qsch05.rsw",
		"que_qaru01.rsw", "que_qaru02.rsw", "que_qaru03.rsw", "que_qaru04.rsw", "que_qaru05.rsw":
		return effects.Cloud3
	case "airplane.rsw", "airplane_01.rsw":
		return effects.Cloud5
	case "einbroch.rsw":
		return effects.Cloud4
	case "thana_boss.rsw", "moc_fild22.rsw", "moc_fild22b.rsw":
		return effects.Cloud6
	case "6@tower.rsw":
		return effects.Cloud7
	case "5@tower.rsw":
		return effects.Cloud8
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

// Draw renders map weather using the shared effects renderer and sound queue.
func (m *System) Draw(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, fx *effects.System, scheduleSound func(time.Time, ...string)) {
	if !effects.Enabled(ctx) {
		*m = System{}
		return
	}
	if screen == nil || ctx.World == nil || ctx.World.GND == nil {
		return
	}
	kind, effectID := mapWeather(ctx.World.MapName, serviceType(ctx))
	switch kind {
	case mapWeatherFireworks:
		m.cloud.reset()
		m.petals = nil
		m.drawFireworksWeather(screen, ctx, projection, now, fx, scheduleSound)
	case mapWeatherLoopingEffect:
		m.fireworks.reset()
		if _, ok := weatherCloudParamsForEffect(effectID); ok {
			m.drawMapWeatherCloudEffect(screen, ctx, projection, effectID, now, fx)
			return
		}
		m.cloud.reset()
		if effectID == effects.Sakura || effectID == effects.Maple || effectID == effects.Snow {
			m.drawStatefulWeather(screen, ctx, projection, effectID, now, fx)
			return
		}
		m.petals = nil
		m.drawLoopingMapWeatherEffect(screen, ctx, projection, effectID, now, fx)
	default:
		m.cloud.reset()
		m.fireworks.reset()
		m.petals = nil
	}
}

// drawStatefulWeather runs the map's petal or snow effect from when the map
// is entered (CWeather::LaunchSakura or LaunchSnow on the player), at the
// player's feet.
func (m *System) drawStatefulWeather(screen *render.Frame, ctx client.Context, projection scene.Projection, effectID int, now time.Time, fx *effects.System) {
	spec, ok := effects.Lookup(effectID)
	if !ok {
		return
	}
	key := normalizeMapNameForWeather(ctx.World.MapName)
	if m.petals == nil || m.petalsMap != key || m.petals.EffectID != effectID {
		effect := effects.Instance{
			EffectID: effectID,
			ActorID:  uint32(effectID),
			Starts:   now,
			Expires:  now.Add(spec.Duration),
			Duration: spec.Duration,
		}
		fx.PrepareEnvironmentEffect(&effect, projection)
		m.petals, m.petalsMap = &effect, key
	}
	worldX, worldY := projection.PlayerX, projection.PlayerY
	worldZ := scene.TerrainHeightAt(ctx.World, worldX-0.5, worldY-0.5)
	for componentIndex, component := range spec.Components {
		fx.DrawWorldEffectComponent(screen, ctx, projection, *m.petals, component, componentIndex, worldX, worldY, worldZ, 0, spec.Duration, now)
	}
}

func (m *System) drawLoopingMapWeatherEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, effectID int, now time.Time, fx *effects.System) {
	spec, ok := effects.Lookup(effectID)
	if !ok {
		return
	}
	starts := loopingMapWeatherEffectStart(effectID, spec.Duration, now)
	effect := effects.Instance{
		EffectID: effectID,
		ActorID:  uint32(effectID),
		Starts:   starts,
		Expires:  now.Add(24 * time.Hour),
		Duration: spec.Duration,
		X:        int(math.Round(projection.PlayerX)),
		Y:        int(math.Round(projection.PlayerY)),
	}
	worldX := projection.PlayerX
	worldY := projection.PlayerY
	worldZ := scene.TerrainHeightAt(ctx.World, worldX-0.5, worldY-0.5) + 1
	for componentIndex, component := range spec.Components {
		duration := fx.WorldEffectResolvedComponentDuration(ctx, spec, component, effect)
		if duration <= 0 {
			duration = spec.Duration
		}
		componentStarts := loopingMapWeatherEffectStart(effectID+componentIndex*17, duration, now)
		effect.Starts = componentStarts
		progress := effects.WorldEffectComponentProgress(componentStarts, duration, now)
		fx.DrawWorldEffectComponent(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, progress, duration, now)
	}
}

func (m *System) drawFireworksWeather(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, fx *effects.System, scheduleSound func(time.Time, ...string)) {
	key := normalizeMapNameForWeather(ctx.World.MapName)
	centerX, centerY := mapWeatherFireworkCenter(ctx.World, projection, now)
	player := pokJukPlayerPoint(ctx.World, centerX, centerY)
	m.fireworks.ensure(key, player, now)
	m.fireworks.update(player, now)
	m.fireworks.scheduleSounds(m, now, scheduleSound)
	m.fireworks.draw(screen, fx, ctx, projection)
}

func (m *System) scheduleMapWeatherSound(index int, starts time.Time, now time.Time, scheduleSound func(time.Time, ...string), paths ...string) {
	elapsed := now.Sub(starts)
	if elapsed < 0 {
		return
	}
	if m.sounds == nil {
		m.sounds = make(map[int]time.Time)
	}
	if m.sounds[index].Equal(starts) {
		return
	}
	m.sounds[index] = starts
	if elapsed > 120*time.Millisecond {
		return
	}
	if scheduleSound != nil {
		scheduleSound(now, paths...)
	}
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
