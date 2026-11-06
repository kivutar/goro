package weather

import (
	"image/color"

	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
)

// FogFromMap combines the map fog table with weather-specific colors.
func FogFromMap(manager *res.Manager, mapName string, cfg config.FogConfig) scene.SceneFog {
	if manager == nil || !cfg.Enabled {
		return scene.SceneFog{}
	}
	parameter, ok := manager.FogParameter(mapName)
	if !ok || parameter.Far <= parameter.Near {
		return scene.SceneFog{}
	}
	// The original uses linear fog, so the table's exponential density is unused.
	fogColor := parameter.Color
	if weatherColor, ok := sceneFogColorForMapWeather(mapName); ok {
		fogColor = weatherColor
	}
	// Table distances span the original 10..1500 view planes, with five source
	// units per Goro cell. They are independent of our camera's clipping planes.
	const nearPlane, farPlane = 10.0 / 5, 1500.0 / 5
	return scene.SceneFog{
		Enabled: true,
		Near:    nearPlane + parameter.Near*(farPlane-nearPlane),
		Far:     nearPlane + parameter.Far*(farPlane-nearPlane),
		Color:   fogColor,
	}
}

func sceneFogColorForMapWeather(mapName string) (color.RGBA, bool) {
	effectID := mapWeatherEffectIDForMap(mapName)
	switch effectID {
	case effects.Cloud4:
		params, ok := weatherCloudParamsForEffect(effectID)
		return params.tint, ok
	default:
		return color.RGBA{}, false
	}
}
