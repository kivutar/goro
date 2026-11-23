package maps

import (
	"hash/fnv"
	"image/color"
	"math"
	"strings"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
)

// Lighting holds a map's directional light (C3dWorldRes light fields).
// direction is in Goro world space, which mirrors the classic client's Y axis.
type Lighting struct {
	direction scene.Point3
	Diffuse   scene.Point3
	ambient   scene.Point3
	// Env is CWeather's environment colour, multiplied into ground and model
	// vertex colours at draw time; night lowers it.
	Env scene.Point3
}

func sceneLightingFromRSW(rsw *res.RSW) Lighting {
	// C3dWorldRes::Load's defaults; res.ParseRSW fills them in for old files.
	longitude := 45.0
	latitude := 45.0
	diffuse := scene.Point3{X: 1, Y: 1, Z: 1}
	ambient := scene.Point3{X: 0.3, Y: 0.3, Z: 0.3}
	if rsw != nil {
		longitude = float64(rsw.Light.Longitude)
		latitude = float64(rsw.Light.Latitude)
		diffuse = scene.Point3{X: float64(rsw.Light.Diffuse[0]), Y: float64(rsw.Light.Diffuse[1]), Z: float64(rsw.Light.Diffuse[2])}
		ambient = scene.Point3{X: float64(rsw.Light.Ambient[0]), Y: float64(rsw.Light.Ambient[1]), Z: float64(rsw.Light.Ambient[2])}
	}
	return Lighting{
		direction: lightDirection(longitude, latitude),
		Diffuse:   diffuse,
		ambient:   ambient,
		Env:       environmentColor(diffuse, ambient),
	}
}

// lightDirection rotates (0, 1, 0) about X by the latitude and then about Y by
// the longitude, as C3dWorldRes::Load does, and mirrors Y into Goro space.
func lightDirection(longitude, latitude float64) scene.Point3 {
	lon := scene.DegreesToRadians(longitude)
	lat := scene.DegreesToRadians(latitude)
	return scene.Point3{
		X: math.Sin(lat) * math.Sin(lon),
		Y: -math.Cos(lat),
		Z: math.Sin(lat) * math.Cos(lon),
	}
}

// originalDirection is the light direction in the classic client's world space.
func (l Lighting) originalDirection() scene.Point3 {
	return scene.Point3{X: l.direction.X, Y: -l.direction.Y, Z: l.direction.Z}
}

// environmentColor is CWeather::Process's m_EnvDiff.
func environmentColor(diffuse, ambient scene.Point3) scene.Point3 {
	return scene.Point3{
		X: 1 - (1-diffuse.X)*(1-ambient.X),
		Y: 1 - (1-diffuse.Y)*(1-ambient.Y),
		Z: 1 - (1-diffuse.Z)*(1-ambient.Z),
	}
}

func (m *System) Lighting(rsw *res.RSW) Lighting {
	lighting := sceneLightingFromRSW(rsw)
	if !m.Night {
		return lighting
	}
	// CWeather lowers only its own red and green diffuse at night. The baked
	// ground and model lighting keeps the map's diffuse; only Env changes.
	night := lighting.Diffuse
	night.X = math.Min(night.X, 0.5)
	night.Y = math.Min(night.Y, 0.5)
	lighting.Env = environmentColor(night, lighting.ambient)
	return lighting
}

// lit is min(ambient + max(normal·light, 0) * diffuse, 1) per channel, the
// vertex lighting of C3dNode::UpdateVertexColor and C3dGround15::AssignGnd.
// The normal must be in Goro world space.
func (l Lighting) lit(normal scene.Point3) scene.Point3 {
	return l.litProduct(scene.Dot3(normal, l.direction))
}

// litOriginal is lit for a normal in the classic client's world space.
func (l Lighting) litOriginal(normal scene.Point3) scene.Point3 {
	return l.litProduct(scene.Dot3(normal, l.originalDirection()))
}

func (l Lighting) litProduct(product float64) scene.Point3 {
	product = math.Max(product, 0)
	return scene.Point3{
		X: math.Min(l.ambient.X+product*l.Diffuse.X, 1),
		Y: math.Min(l.ambient.Y+product*l.Diffuse.Y, 1),
		Z: math.Min(l.ambient.Z+product*l.Diffuse.Z, 1),
	}
}

// applyEnv multiplies a vertex colour by Env, truncating like the classic
// client's byte arithmetic.
func (l Lighting) applyEnv(c color.RGBA) color.RGBA {
	env := clampUnitPoint(l.Env)
	return color.RGBA{
		R: uint8(float64(c.R) * env.X),
		G: uint8(float64(c.G) * env.Y),
		B: uint8(float64(c.B) * env.Z),
		A: c.A,
	}
}

// ModelScaleNormalized approximates lighting for GR2 actor models.
func (l Lighting) ModelScaleNormalized(normal scene.Point3) scene.Point3 {
	weight := math.Max(scene.Dot3(normal, l.direction), 0.5)
	return scene.Point3{
		X: scene.ClampUnit(l.ambient.X+l.Diffuse.X*weight) * scene.ClampUnit(l.Env.X),
		Y: scene.ClampUnit(l.ambient.Y+l.Diffuse.Y*weight) * scene.ClampUnit(l.Env.Y),
		Z: scene.ClampUnit(l.ambient.Z+l.Diffuse.Z*weight) * scene.ClampUnit(l.Env.Z),
	}
}

func clampUnitPoint(point scene.Point3) scene.Point3 {
	return scene.Point3{X: scene.ClampUnit(point.X), Y: scene.ClampUnit(point.Y), Z: scene.ClampUnit(point.Z)}
}

func textureColor(name string) color.RGBA {
	if name == "" {
		return color.RGBA{R: 78, G: 86, B: 78, A: 255}
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.ToLower(name)))
	value := hash.Sum32()
	return color.RGBA{
		R: 70 + uint8(value&0x3f),
		G: 80 + uint8((value>>8)&0x4f),
		B: 68 + uint8((value>>16)&0x4f),
		A: 255,
	}
}
