package weather

import (
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

// CWeather::LaunchPokJuk launches EF_POKJUK_SOUND and 160 EF_POKJUK effects
// on the player. Each EF_POKJUK holds one PP_POKJUK primitive of four
// sparks (CEffectPrim::PrimPokJuk): a spark waits 300-349 frames, rises
// unseen 0.25 per frame from 15 units beside the player, bursts after 155
// frames into a coloured, fading, slowing spark, and starts again every 2001
// frames from wherever the player then stands. Positions are absolute client
// units; client Y points down.

const (
	pokJukEffects        = 160
	pokJukSparksPerPrim  = 4
	pokJukSparks         = pokJukEffects * pokJukSparksPerPrim
	pokJukStartDelay     = 300
	pokJukStartJitter    = 50
	pokJukOffsetX        = 15
	pokJukRise           = 0.25
	pokJukBurstAt        = 154
	pokJukBurstAlpha     = 250
	pokJukGravity        = 0.02
	pokJukDrag           = 0.98
	pokJukCycle          = 2000
	pokJukTwinkleStart   = 160
	pokJukTwinkleFrames  = 20
	pokJukSpin           = 5
	pokJukHeadRadius     = 2.0
	pokJukSoundFirst     = 300
	pokJukSoundPeriod    = 2002
	pokJukMaxCatchUp     = 120
	pokJukTrailEvery     = 3
	pokJukTrailRadiusCut = 0.5
)

// pokJukColors are RenderPokJuk's colours for flag1[0].
var pokJukColors = [5]color.RGBA{
	{R: 70, G: 70, B: 255, A: 255},
	{R: 255, G: 70, B: 70, A: 255},
	{R: 70, G: 255, B: 70, A: 255},
	{R: 255, G: 255, B: 90, A: 255},
	{R: 255, G: 70, B: 255, A: 255},
}

// pokJukTextures are m_texture[0..2], selected by flag1[3].
var pokJukTextures = [3]string{"effect/pok1.tga", "effect/pok3.tga", "effect/pok2.tga"}

type pokJukSpark struct {
	process     int
	bursting    bool
	twinkle     int
	color       int
	twinkleAt   int
	rotation    int
	elevation   int // full_display_angle
	heading     int // rise_angle
	speed       float64
	alpha       int
	head, trail scene.Point3 // vecB_now and vecB_pre
	tNow, tPre  scene.Point3 // vecT_now and vecT_pre, only feeding the trail
}

type mapWeatherFireworkState struct {
	key    string
	starts time.Time
	frame  int
	rng    *rand.Rand
	sparks []pokJukSpark
}

func mapWeatherFireworkCenter(world *worldstate.World, projection scene.Projection, now time.Time) (float64, float64) {
	if world == nil {
		return projection.PlayerX, projection.PlayerY
	}
	return scene.ActorRenderPosition(world.Player, now)
}

func (s *mapWeatherFireworkState) reset() {
	*s = mapWeatherFireworkState{}
}

func (s *mapWeatherFireworkState) random(n int) int {
	if n <= 0 {
		return 0
	}
	return s.rng.Intn(n)
}

// ensure launches the sparks when a map is entered, as CRagEffect::PokJuk
// does on its first frame.
func (s *mapWeatherFireworkState) ensure(key string, player scene.Point3, now time.Time) {
	if s.sparks != nil && s.key == key {
		return
	}
	s.reset()
	s.key, s.starts = key, now
	s.rng = rand.New(rand.NewSource(now.UnixNano()))
	origin := player
	origin.X -= pokJukOffsetX
	s.sparks = make([]pokJukSpark, pokJukSparks)
	for i := range s.sparks {
		s.sparks[i] = pokJukSpark{
			process:   -pokJukStartDelay - s.random(pokJukStartJitter),
			rotation:  s.random(360),
			head:      origin,
			trail:     origin,
			tNow:      origin,
			tPre:      origin,
			color:     s.random(len(pokJukColors)),
			twinkleAt: pokJukTwinkleStart + s.random(4)*pokJukTwinkleFrames,
			speed:     0.4 + float64(s.random(100))*0.004,
			elevation: s.random(360),
			heading:   s.random(360),
		}
	}
}

// update steps every spark to the current effect frame.
func (s *mapWeatherFireworkState) update(player scene.Point3, now time.Time) {
	target := int(now.Sub(s.starts) / db.EffectFrameDuration)
	if target-s.frame > pokJukMaxCatchUp {
		s.frame = target - pokJukMaxCatchUp
	}
	for ; s.frame < target; s.frame++ {
		for i := range s.sparks {
			s.step(&s.sparks[i], player)
		}
	}
}

// step is one frame of CEffectPrim::PrimPokJuk for one spark.
func (s *mapWeatherFireworkState) step(p *pokJukSpark, player scene.Point3) {
	p.process++
	if p.process <= 0 {
		return
	}
	if p.process%pokJukTrailEvery == 0 {
		p.tPre, p.tNow, p.trail = p.tNow, p.trail, p.head
	}
	p.twinkle = 0
	if p.process >= p.twinkleAt && p.process <= p.twinkleAt+pokJukTwinkleFrames {
		switch p.process % pokJukTwinkleFrames {
		case 15, 19:
			p.twinkle = 1
		case 17:
			p.twinkle = 2
		}
	}
	p.rotation = (p.rotation + pokJukSpin) % 360
	if !p.bursting {
		p.head.Y -= pokJukRise
		if p.process > pokJukBurstAt {
			p.bursting, p.alpha = true, pokJukBurstAlpha
		}
	} else {
		reach := cosDegrees(p.elevation) * p.speed
		p.head.Y += sinDegrees(p.elevation)*p.speed + pokJukGravity
		p.head.X += cosDegrees(p.heading) * reach
		p.head.Z += sinDegrees(p.heading) * reach
		p.speed *= pokJukDrag
		p.alpha--
		if s.random(2) == 0 {
			p.alpha--
		}
		p.alpha = max(p.alpha, 0)
	}
	if p.process > pokJukCycle {
		p.process, p.bursting = 0, false
		p.heading = s.random(360)
		p.head = player
		p.head.X -= pokJukOffsetX
		p.speed = 0.4 + float64(s.random(100))*0.004
	}
}

func sinDegrees(degrees int) float64 { return math.Sin(scene.DegreesToRadians(float64(degrees))) }
func cosDegrees(degrees int) float64 { return math.Cos(scene.DegreesToRadians(float64(degrees))) }

// scheduleSounds plays EF_POKJUK_SOUND's 폭죽.wav on frame 300 and then
// every 2002 frames.
func (s *mapWeatherFireworkState) scheduleSounds(m *System, now time.Time, scheduleSound func(time.Time, ...string)) {
	frame := int(now.Sub(s.starts) / db.EffectFrameDuration)
	if frame < pokJukSoundFirst {
		return
	}
	cycle := (frame - pokJukSoundFirst) / pokJukSoundPeriod
	starts := s.starts.Add(time.Duration(pokJukSoundFirst+cycle*pokJukSoundPeriod) * db.EffectFrameDuration)
	m.scheduleMapWeatherSound(0, starts, now, scheduleSound, pokJukLaunchSFX)
}

// draw is RenderPokJuk: each visible spark is a screen-facing square spun
// by its rotation, at its head with radius 2 and on its trail with radius
// 1.5 and half the alpha, drawn additively. While twinkling it switches to
// a white pok3 or pok2 square at a quarter of the alpha.
func (s *mapWeatherFireworkState) draw(screen *render.Frame, fx *effects.System, ctx client.Context, projection scene.Projection) {
	var textures [3]*render.Image
	for i, file := range pokJukTextures {
		textures[i] = fx.EffectFileTexture(ctx.Resources, file)
	}
	_, _, right, up := projection.CameraBasis()
	var batches [3]struct {
		vertices []render.Vertex3D
		indices  []uint16
	}
	for _, p := range s.sparks {
		if p.process <= 0 || p.alpha <= 0 {
			continue
		}
		texture := textures[p.twinkle]
		if texture == nil {
			continue
		}
		tint := pokJukColors[p.color]
		alpha := p.alpha
		if p.twinkle != 0 {
			tint = color.RGBA{R: 255, G: 255, B: 255, A: 255}
			alpha /= 4
		}
		b := &batches[p.twinkle]
		radius := pokJukHeadRadius
		for segment, center := range [2]scene.Point3{p.head, p.trail} {
			tint.A = uint8(alpha / (segment + 1))
			appendPokJukSquare(&b.vertices, &b.indices, texture, center, right, up, radius, p.rotation, tint)
			radius -= pokJukTrailRadiusCut
		}
	}
	options := effects.TexturedEffectBillboardDrawOptions(true, false)
	options.DisableFog = true
	for i, b := range batches {
		if len(b.vertices) > 0 {
			screen.DrawTriangles3DOwned(b.vertices, b.indices, textures[i], options)
		}
	}
}

// appendPokJukSquare appends the quad whose corners lie radius client units
// from center at rotation, +90, +180 and +270 degrees in the screen plane.
func appendPokJukSquare(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, center, right, up scene.Point3, radius float64, rotation int, tint color.RGBA) {
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	world := scene.Point3{X: center.X / 5, Y: -center.Y / 5, Z: center.Z / 5}
	uvs := [4]scene.UV{{U: 0, V: 1}, {U: 1, V: 1}, {U: 1, V: 0}, {U: 0, V: 0}}
	first := uint16(len(*vertices))
	for corner := range 4 {
		angle := rotation + corner*90
		offset := scene.Add3(scene.Mul3(right, cosDegrees(angle)*radius/5), scene.Mul3(up, sinDegrees(angle)*radius/5))
		*vertices = append(*vertices, scene.TexturedSurfaceVertex3D(scene.Add3(world, offset), uvs[corner], tint, w, h))
	}
	*indices = append(*indices, first, first+1, first+2, first+3, first+2, first)
}

// pokJukPlayerPoint is the player's position in client units.
func pokJukPlayerPoint(world *worldstate.World, x, y float64) scene.Point3 {
	return scene.Point3{X: x * 5, Y: -scene.TerrainHeightAt(world, x-0.5, y-0.5) * 5, Z: y * 5}
}
