package weather

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestDisabledEffectsStopWeather(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.MapName = "comodo"
	ctx.World.GND = &res.GND{Width: 1, Height: 1}
	ctx.Session.NoEffects = true
	m := &System{sounds: map[int]time.Time{effects.Firecracker: time.Now()}}
	frame := render.NewFrame(800, 600)
	m.Draw(frame, ctx, testutil.Projection(800, 600, .5, .5, 0), time.Now(), &effects.System{}, func(time.Time, ...string) {
		t.Fatal("disabled weather played a sound")
	})
	if !reflect.DeepEqual(*m, System{}) {
		t.Fatal("disabled weather retained its particle or sound cycle")
	}
}

func TestMapWeatherForComodoMatchesReferenceWeatherTable(t *testing.T) {
	for _, name := range []string{"comodo", "comodo.gat", "data\\comodo.rsw", "DATA/COMODO.RSW"} {
		if got, _ := mapWeather(name, "korea"); got != mapWeatherFireworks {
			t.Fatalf("mapWeather(%q) = %d, want fireworks", name, got)
		}
	}
	if got, _ := mapWeather("geffen", "korea"); got != mapWeatherNone {
		t.Fatalf("mapWeather(geffen) = %d, want none", got)
	}
}

func TestServiceGatedMapWeather(t *testing.T) {
	tests := []struct {
		name, service string
		kind          mapWeatherKind
		effectID      int
	}{
		{"prontera", "korea", mapWeatherLoopingEffect, effects.Sakura},
		{"pay_arche", "singapore", mapWeatherLoopingEffect, effects.Sakura},
		{"comodo", "malaysia", mapWeatherFireworks, 0},
		{"prontera", "america", mapWeatherNone, 0},
		{"comodo", "america", mapWeatherNone, 0},
		{"xmas_fild01", "america", mapWeatherLoopingEffect, effects.Snow},
	}
	for _, tc := range tests {
		kind, effectID := mapWeather(tc.name, tc.service)
		if kind != tc.kind || effectID != tc.effectID {
			t.Fatalf("mapWeather(%q, %q) = %d, %d; want %d, %d", tc.name, tc.service, kind, effectID, tc.kind, tc.effectID)
		}
	}
}

func TestMapWeatherEffectIDForReferenceWeatherTable(t *testing.T) {
	tests := map[string]int{
		"xmas":                 effects.Snow,
		"xmas.gat":             effects.Snow,
		"xmas_fild01":          effects.Snow,
		"data\\yuno.gat":       effects.Cloud2,
		"einbroch":             effects.Cloud4,
		"airplane":             effects.Cloud5,
		"airplane.rsw":         effects.Cloud5,
		"airplane_01.gat":      effects.Cloud5,
		"DATA/AIRPLANE_01.RSW": effects.Cloud5,
	}
	for name, want := range tests {
		if got := mapWeatherEffectIDForMap(name); got != want {
			t.Fatalf("mapWeatherEffectIDForMap(%q) = %d, want %d", name, got, want)
		}
	}
	// Prontera's cherry blossoms depend on the service type (mapWeather).
	if got := mapWeatherEffectIDForMap("prontera"); got != 0 {
		t.Fatalf("mapWeatherEffectIDForMap(prontera) = %d, want 0", got)
	}
	if got := mapWeatherEffectIDForMap("payon"); got != 0 {
		t.Fatalf("mapWeatherEffectIDForMap(payon) = %d, want disabled rain", got)
	}
}

func TestMapWeatherReferenceEffectsHaveSpecs(t *testing.T) {
	for _, effectID := range []int{
		effects.Rain,
		effects.Snow,
		effects.Sakura,
		effects.Maple,
		effects.Cloud,
		effects.Cloud2,
		effects.Cloud3,
		effects.Cloud4,
		effects.Cloud5,
		effects.Cloud6,
		effects.Cloud7,
		effects.Cloud8,
	} {
		if _, ok := effects.Lookup(effectID); !ok {
			t.Fatalf("missing weather effect spec %d", effectID)
		}
	}
}

func TestYunoCloudWeatherParamsMatchClassicProfile(t *testing.T) {
	params, ok := weatherCloudParamsForEffect(effects.Cloud2)
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
	spec, ok := effects.Lookup(effects.Cloud4)
	if !ok || len(spec.Components) != 1 {
		t.Fatalf("EF_CLOUD4 spec = %+v ok=%t", spec, ok)
	}
	component := spec.Components[0]
	if component.Kind != effects.Component3D || len(component.TextureFiles) != 3 || component.TextureFiles[0] != "effect/fog1.tga" || component.TextureFiles[2] != "effect/fog3.tga" {
		t.Fatalf("EF_CLOUD4 textures = %+v", component)
	}
	if component.Color != (color.RGBA{R: 252, G: 171, B: 143, A: 255}) || component.AlphaMax != 170.0/255.0 {
		t.Fatalf("EF_CLOUD4 tint/alpha = %+v", component)
	}
}

func TestEinbrochCloudWeatherParamsMatchClassicProfile(t *testing.T) {
	params, ok := weatherCloudParamsForEffect(effects.Cloud4)
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
	params, _ := weatherCloudParamsForEffect(effects.Cloud2)
	state := mapWeatherCloudState{}
	now := time.Unix(10, 0)
	const centerX = 37.5
	const centerY = 98.5
	world := &worldstate.World{GND: testutil.GNDWithTopHeights(64, 64, func(_, _ int) [4]float32 {
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
	params, _ := weatherCloudParamsForEffect(effects.Cloud4)
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
	spec, ok := effects.Lookup(effects.PokJuk)
	if !ok {
		t.Fatal("pokjuk effect spec missing")
	}
	if spec.Duration != 4*time.Second || len(spec.Components) != 4 {
		t.Fatalf("pokjuk spec = %+v", spec)
	}
	launch := spec.Components[0]
	if launch.Kind != effects.Component3D || launch.TextureFile != "effect/pok3.tga" || !launch.BlendAdditive {
		t.Fatalf("pokjuk launch = %+v", launch)
	}
	if launch.PosZ != 0 || launch.PosZEnd != 8 {
		t.Fatalf("pokjuk launch z = %.1f..%.1f", launch.PosZ, launch.PosZEnd)
	}
	for i, component := range spec.Components[1:] {
		if component.Kind != effects.Component3D || component.TextureFile != "effect/pok3.tga" || !component.BlendAdditive {
			t.Fatalf("pokjuk burst component %d = %+v", i+1, component)
		}
		if component.Duplicate == 0 || component.PosXEndRand <= 0 || component.PosYEndRand <= 0 || component.PosZEndRand <= 0 {
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

func TestScheduleMapWeatherSoundOnlyNearNewCycle(t *testing.T) {
	now := time.Unix(10, 0)
	mode := &System{}
	var scheduled [][]string
	schedule := func(at time.Time, paths ...string) {
		if !at.Equal(now) {
			t.Fatalf("sound scheduled at %v, want %v", at, now)
		}
		scheduled = append(scheduled, paths)
	}
	mode.scheduleMapWeatherSound(0, now.Add(-time.Second), now, schedule, pokJukLaunchSFX)
	if len(scheduled) != 0 {
		t.Fatalf("scheduled stale weather sounds = %+v", scheduled)
	}
	mode.scheduleMapWeatherSound(0, now.Add(-50*time.Millisecond), now, schedule, pokJukLaunchSFX)
	if len(scheduled) != 1 || scheduled[0][0] != pokJukLaunchSFX {
		t.Fatalf("scheduled weather sounds = %+v", scheduled)
	}
	mode.scheduleMapWeatherSound(0, now.Add(-50*time.Millisecond), now, schedule, pokJukLaunchSFX)
	if len(scheduled) != 1 {
		t.Fatalf("weather sound scheduled twice = %+v", scheduled)
	}
}
