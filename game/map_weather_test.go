package game

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

func TestMapWeatherForComodoMatchesReferenceWeatherTable(t *testing.T) {
	for _, name := range []string{"comodo", "comodo.gat", "data\\comodo.rsw", "DATA/COMODO.RSW"} {
		if got := mapWeatherForMap(name); got != mapWeatherFireworks {
			t.Fatalf("mapWeatherForMap(%q) = %d, want fireworks", name, got)
		}
	}
	if got := mapWeatherForMap("prontera"); got != mapWeatherNone {
		t.Fatalf("mapWeatherForMap(prontera) = %d, want none", got)
	}
}

func TestMapWeatherEffectIDForReferenceWeatherTable(t *testing.T) {
	tests := map[string]int{
		"xmas":                 effectSnow,
		"xmas.gat":             effectSnow,
		"data\\yuno.gat":       effectCloud2,
		"einbroch":             effectCloud4,
		"airplane":             effectCloud5,
		"airplane.rsw":         effectCloud5,
		"airplane_01.gat":      effectCloud5,
		"DATA/AIRPLANE_01.RSW": effectCloud5,
	}
	for name, want := range tests {
		if got := mapWeatherEffectIDForMap(name); got != want {
			t.Fatalf("mapWeatherEffectIDForMap(%q) = %d, want %d", name, got, want)
		}
	}
	if got := mapWeatherEffectIDForMap("prontera"); got != 0 {
		t.Fatalf("mapWeatherEffectIDForMap(prontera) = %d, want 0", got)
	}
	if got := mapWeatherEffectIDForMap("payon"); got != 0 {
		t.Fatalf("mapWeatherEffectIDForMap(payon) = %d, want disabled rain", got)
	}
}

func TestMapWeatherReferenceEffectsHaveSpecs(t *testing.T) {
	for _, effectID := range []int{
		effectRain,
		effectSnow,
		effectSakura,
		effectMaple,
		effectCloud,
		effectCloud2,
		effectCloud3,
		effectCloud4,
		effectCloud5,
		effectCloud6,
		effectCloud7,
		effectCloud8,
	} {
		if _, ok := worldEffectSpecForID(effectID); !ok {
			t.Fatalf("missing weather effect spec %d", effectID)
		}
	}
}

func TestYunoCloudWeatherParamsMatchClassicProfile(t *testing.T) {
	params, ok := weatherCloudParamsForEffect(effectCloud2)
	if !ok {
		t.Fatal("EF_CLOUD2 weather cloud params missing")
	}
	if len(params.textureFiles) != 3 || params.textureFiles[0] != "effect/cloud4.tga" || params.textureFiles[1] != "effect/cloud1.tga" || params.textureFiles[2] != "effect/cloud2.tga" {
		t.Fatalf("EF_CLOUD2 weather cloud resources = %+v", params)
	}
	if params.tint != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) || params.alphaMax != 240.0/255.0 {
		t.Fatalf("EF_CLOUD2 tint/alpha = %+v", params)
	}
	if params.count != 240 || params.offsetMin != 5 || params.radius != 40 || params.zOffset != -10 || params.zRand != 2 {
		t.Fatalf("EF_CLOUD2 placement = %+v", params)
	}
	if params.sizeBase != 30*math.Sqrt2*0.2 || params.sizeRand != 20*math.Sqrt2*0.2 {
		t.Fatalf("EF_CLOUD2 size = %+v", params)
	}
	if params.ramp != 80*time.Second/60 || params.fadeOut != 240*time.Second/60 || params.rotStartMin != 5*time.Second || params.rotStartRand != 200*time.Second/60 {
		t.Fatalf("EF_CLOUD2 timing = %+v", params)
	}
	if params.useGround || params.overlay || params.additive || params.blackKey || !params.disableFog || params.screenHaze.A != 0 {
		t.Fatalf("EF_CLOUD2 render options = %+v", params)
	}
}

func TestEinbrochCloudWeatherSpecUsesClassicResources(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectCloud4)
	if !ok || len(spec.components) != 1 {
		t.Fatalf("EF_CLOUD4 spec = %+v ok=%t", spec, ok)
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || len(component.textureFiles) != 3 || component.textureFiles[0] != "effect/fog1.tga" || component.textureFiles[2] != "effect/fog3.tga" {
		t.Fatalf("EF_CLOUD4 textures = %+v", component)
	}
	if component.color != (color.RGBA{R: 252, G: 171, B: 143, A: 255}) || component.alphaMax != 170.0/255.0 {
		t.Fatalf("EF_CLOUD4 tint/alpha = %+v", component)
	}
}

func TestEinbrochCloudWeatherParamsMatchClassicProfile(t *testing.T) {
	params, ok := weatherCloudParamsForEffect(effectCloud4)
	if !ok {
		t.Fatal("EF_CLOUD4 weather cloud params missing")
	}
	if len(params.textureFiles) != 3 || params.textureFiles[0] != "effect/fog1.tga" || params.tint != (color.RGBA{R: 252, G: 171, B: 143, A: 255}) {
		t.Fatalf("EF_CLOUD4 weather cloud resources = %+v", params)
	}
	if params.count != 320 || params.radius != 30 || params.zOffset != 4 || params.zRand != 1 {
		t.Fatalf("EF_CLOUD4 weather cloud placement = %+v", params)
	}
	if !params.useGround || params.alphaMax != weatherCloudClassicAlphaMax {
		t.Fatalf("EF_CLOUD4 ground-relative alpha = %+v", params)
	}
	if params.blackKey {
		t.Fatalf("EF_CLOUD4 weather cloud should preserve black-key pixels: %+v", params)
	}
	if !params.disableFog {
		t.Fatalf("EF_CLOUD4 weather cloud should not be darkened by map fog: %+v", params)
	}
	if params.screenHaze != (color.RGBA{R: 252, G: 171, B: 143, A: 70}) {
		t.Fatalf("EF_CLOUD4 screen haze = %+v, want peach weather tint", params.screenHaze)
	}
	if params.sizeBase != 35*math.Sqrt2*0.2 || params.sizeRand != 10*math.Sqrt2*0.2 {
		t.Fatalf("EF_CLOUD4 weather cloud size = %+v", params)
	}
	if params.ramp != 170*time.Second/60 || params.rotStartMin != 5*time.Second || params.rotStartRand != 200*time.Second/60 {
		t.Fatalf("EF_CLOUD4 weather cloud timing = %+v", params)
	}
}

func TestYunoCloudWeatherSpawnsInOuterSkyRing(t *testing.T) {
	params, _ := weatherCloudParamsForEffect(effectCloud2)
	state := mapWeatherCloudState{}
	now := time.Unix(10, 0)
	const centerX = 37.5
	const centerY = 98.5
	world := &worldstate.World{GND: testGNDWithTopHeights(64, 64, func(_, _ int) [4]float32 {
		return [4]float32{12, 12, 12, 12}
	})}
	state.ensure("yuno.rsw", params, world, centerX, centerY, now)
	if len(state.clouds) == 0 {
		t.Fatal("weather cloud state empty")
	}
	first := state.clouds[0]
	dx := math.Abs(first.x - centerX)
	dy := math.Abs(first.y - centerY)
	if dx < 5 || dx > 45 || dy < 5 || dy > 45 {
		t.Fatalf("Yuno cloud offset = %.2f,%.2f, want signed 5..45 from center", dx, dy)
	}
	if first.z < -10 || first.z > -8 {
		t.Fatalf("Yuno cloud z = %.2f, want fixed weather height -10..-8", first.z)
	}
}

func TestMapWeatherCloudKeepsExistingParticlesWhenCenterMoves(t *testing.T) {
	params, _ := weatherCloudParamsForEffect(effectCloud4)
	state := mapWeatherCloudState{}
	now := time.Unix(10, 0)
	state.ensure("einbroch.rsw", params, nil, 37.5, 98.5, now)
	if len(state.clouds) == 0 {
		t.Fatal("weather cloud state empty")
	}
	firstX, firstY := state.clouds[0].x, state.clouds[0].y
	state.update(params, nil, 57.5, 118.5, now.Add(time.Second))
	if math.Hypot(state.clouds[0].x-firstX, state.clouds[0].y-firstY) > 1 {
		t.Fatalf("weather cloud snapped to moved center: before %.2f,%.2f after %.2f,%.2f", firstX, firstY, state.clouds[0].x, state.clouds[0].y)
	}
}

func TestPokJukWeatherEffectSpecUsesAdditiveParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectPokJuk)
	if !ok {
		t.Fatal("pokjuk effect spec missing")
	}
	if spec.duration != 4*time.Second || len(spec.components) != 4 {
		t.Fatalf("pokjuk spec = %+v", spec)
	}
	launch := spec.components[0]
	if launch.kind != effectComponent3D || launch.textureFile != "effect/pok3.tga" || !launch.blendAdditive {
		t.Fatalf("pokjuk launch = %+v", launch)
	}
	if launch.posZ != 0 || launch.posZEnd != 8 {
		t.Fatalf("pokjuk launch z = %.1f..%.1f", launch.posZ, launch.posZEnd)
	}
	for i, component := range spec.components[1:] {
		if component.kind != effectComponent3D || component.textureFile != "effect/pok3.tga" || !component.blendAdditive {
			t.Fatalf("pokjuk burst component %d = %+v", i+1, component)
		}
		if component.duplicate == 0 || component.posXEndRand <= 0 || component.posYEndRand <= 0 || component.posZEndRand <= 0 {
			t.Fatalf("pokjuk burst spread %d = %+v", i+1, component)
		}
	}
}

func TestLoopingMapWeatherEffectStartStaggersFireworks(t *testing.T) {
	now := time.Unix(10, 500*int64(time.Millisecond))
	duration := 4 * time.Second
	first := loopingMapWeatherEffectStart(0, duration, now)
	second := loopingMapWeatherEffectStart(1, duration, now)
	if first.Equal(second) {
		t.Fatalf("weather effect starts were not staggered: %s", first)
	}
	if now.Sub(first) < 0 || now.Sub(first) >= duration {
		t.Fatalf("first start = %s for now %s", first, now)
	}
	if now.Sub(second) < 0 || now.Sub(second) >= duration {
		t.Fatalf("second start = %s for now %s", second, now)
	}
}

func TestPokJukWeatherUsesFourStatefulRockets(t *testing.T) {
	if pokJukWeatherFireworks != 4 {
		t.Fatalf("PokJuk weather rockets = %d, want 4 like the reference client", pokJukWeatherFireworks)
	}
	state := mapWeatherFireworkState{}
	now := time.Unix(40, 0)
	state.ensure("comodo.rsw", nil, 100, 200, now)
	for i, rocket := range state.rockets {
		if rocket.launchTime.IsZero() {
			t.Fatalf("rocket %d was not initialized", i)
		}
	}
}

func TestPokJukWeatherRiseMatchesROClassic(t *testing.T) {
	classicRise := 154 * time.Second / 60
	if pokJukWeatherRiseDuration != classicRise {
		t.Fatalf("PokJuk weather rise duration = %s, want ro-classic 154-frame timing", pokJukWeatherRiseDuration)
	}
}

func TestPokJukWeatherHasMultipleActiveRockets(t *testing.T) {
	state := mapWeatherFireworkState{}
	now := time.Unix(40, 0)
	state.ensure("comodo.rsw", nil, 100, 200, now)
	active := 0
	for _, rocket := range state.rockets {
		elapsed := now.Sub(rocket.launchTime)
		if elapsed >= 0 && elapsed < pokJukWeatherRocketLifetime() {
			active++
		}
	}
	if active < 2 {
		t.Fatalf("active PokJuk rockets = %d, want multiple visible weather rockets", active)
	}
}

func TestPokJukWeatherRocketDoesNotFollowMovedCenterWhileActive(t *testing.T) {
	state := mapWeatherFireworkState{}
	now := time.Unix(40, 0)
	state.ensure("comodo.rsw", nil, 100, 200, now)
	first := state.rockets[0]
	state.update(nil, 300, 400, now.Add(time.Second))
	second := state.rockets[0]
	if first.x != second.x || first.y != second.y || first.z != second.z {
		t.Fatalf("active rocket followed moved center: before %.2f,%.2f,%.2f after %.2f,%.2f,%.2f", first.x, first.y, first.z, second.x, second.y, second.z)
	}
}

func TestPokJukWeatherRocketRecyclesNearCurrentPlayer(t *testing.T) {
	state := mapWeatherFireworkState{}
	now := time.Unix(40, 0)
	state.ensure("comodo.rsw", nil, 100, 200, now)
	first := state.rockets[0]
	state.update(nil, 300, 400, now.Add(pokJukWeatherCycle()))
	second := state.rockets[0]
	if math.Abs(second.x-300) > pokJukWeatherSpread || math.Abs(second.y-400) > pokJukWeatherSpread {
		t.Fatalf("recycled rocket origin = %.2f,%.2f, want within %.2f of current player", second.x, second.y, float64(pokJukWeatherSpread))
	}
	if math.Hypot(second.x-first.x, second.y-first.y) < 100 {
		t.Fatalf("recycled rocket did not move to new player area: before %.2f,%.2f after %.2f,%.2f", first.x, first.y, second.x, second.y)
	}
}

func TestPokJukWeatherResetsWithMapKey(t *testing.T) {
	state := mapWeatherFireworkState{}
	now := time.Unix(40, 0)
	state.ensure("comodo.rsw", nil, 100, 200, now)
	first := state.rockets[0]
	state.ensure("other.rsw", nil, 300, 400, now.Add(time.Second))
	second := state.rockets[0]
	if math.Abs(second.x-300) > pokJukWeatherSpread || math.Abs(second.y-400) > pokJukWeatherSpread {
		t.Fatalf("reset rocket origin = %.2f,%.2f, want within %.2f of new player", second.x, second.y, float64(pokJukWeatherSpread))
	}
	if math.Hypot(second.x-first.x, second.y-first.y) < 100 {
		t.Fatalf("reset rocket did not move to new map area: before %.2f,%.2f after %.2f,%.2f", first.x, first.y, second.x, second.y)
	}
}

func TestPokJukWeatherCenterUsesPlayerPosition(t *testing.T) {
	now := time.Unix(10, 0)
	world := &worldstate.World{Player: worldstate.Actor{X: 12, Y: 34}}
	projection := sceneProjection{playerX: 1000, playerY: 2000}
	x, y := mapWeatherFireworkCenter(world, projection, now)
	if x != 12 || y != 34 {
		t.Fatalf("firework center = %.2f,%.2f, want player position", x, y)
	}
}

func TestPokJukWeatherTextureFallsBackToAvailableTexture(t *testing.T) {
	texture := render.NewImage(1, 1)
	textures := [3]*render.Image{nil, nil, texture}
	if got := pokJukWeatherTexture(textures, 0); got != texture {
		t.Fatalf("fallback texture = %p, want %p", got, texture)
	}
}

func TestScheduleMapWeatherSoundOnlyNearNewCycle(t *testing.T) {
	now := time.Unix(10, 0)
	mode := NewWorldMode()
	mode.scheduleMapWeatherSound(0, now.Add(-time.Second), now, pokJukLaunchSFX)
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("scheduled stale weather sounds = %+v", mode.scheduledSounds)
	}
	mode.scheduleMapWeatherSound(0, now.Add(-50*time.Millisecond), now, pokJukLaunchSFX)
	if len(mode.scheduledSounds) != 1 || mode.scheduledSounds[0].paths[0] != pokJukLaunchSFX {
		t.Fatalf("scheduled weather sounds = %+v", mode.scheduledSounds)
	}
	mode.scheduleMapWeatherSound(0, now.Add(-50*time.Millisecond), now, pokJukLaunchSFX)
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("weather sound scheduled twice = %+v", mode.scheduledSounds)
	}
}

func TestScheduleMapWeatherExplosionSoundUsesItemPokjukFallback(t *testing.T) {
	now := time.Unix(10, 0)
	mode := NewWorldMode()
	mode.scheduleMapWeatherSound(1, now.Add(-50*time.Millisecond), now, pokJukExplosionSFX, pokJukLaunchSFX)
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled weather sounds = %+v", mode.scheduledSounds)
	}
	got := mode.scheduledSounds[0].paths
	if len(got) != 2 || got[0] != pokJukExplosionSFX || got[1] != pokJukLaunchSFX {
		t.Fatalf("explosion sound paths = %#v", got)
	}
}
