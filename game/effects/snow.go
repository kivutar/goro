package effects

import (
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

// CRagEffect::Snow follows the player and launches two PP_3DPARTICLE_NOMASTER
// ef_snow sprites every frame. Each flake starts up to 300 units from the
// player, 100 units above it, falls 0.5 per frame for 320 frames, fades in
// over 10 frames and out over its last 64. Positions are absolute client
// units; client Y points down.

const (
	snowFlakesPerFrame = 2
	snowFlakeLife      = 320
	snowFlakeFadeOut   = snowFlakeLife - snowFlakeLife/5
	snowFlakeFall      = 0.5
	snowFlakeLift      = 100
	snowFlakeSpread    = 300
	snowFlakeMaxAlpha  = 254
	snowFlakeFadeIn    = snowFlakeMaxAlpha / 10.0
	// snowMaxCatchUp bounds the frames stepped in one draw, so a long pause
	// skips ahead instead of stalling.
	snowMaxCatchUp = 120
)

type snowFlake struct {
	pos        scene.Point3
	size       float64
	alpha      float64
	alphaSpeed float64
	age        int
}

type snowState struct {
	frame  int
	rng    *rand.Rand
	flakes []snowFlake
}

func newSnowState(effect Instance) *snowState {
	seed := int64(deterministicUnit(effect, 1)*(1<<31)) ^ effect.Starts.UnixNano()
	return &snowState{rng: rand.New(rand.NewSource(seed))}
}

func (s *snowState) random(n int) int {
	if n <= 0 {
		return 0
	}
	return s.rng.Intn(n)
}

// advance steps the effect up to target. player is the player's position
// and ground the terrain height under it, both in client units.
func (s *snowState) advance(target int, player scene.Point3, ground float64) {
	if target-s.frame > snowMaxCatchUp {
		s.frame = target - snowMaxCatchUp
	}
	for ; s.frame <= target; s.frame++ {
		for i := 0; i < snowFlakesPerFrame; i++ {
			s.flakes = append(s.flakes, s.newFlake(player, ground))
		}
		alive := s.flakes[:0]
		for _, f := range s.flakes {
			if stepSnowFlake(&f) {
				alive = append(alive, f)
			}
		}
		s.flakes = alive
	}
}

// newFlake launches one flake: m_deltaPos2 is (0, ground-100, random(300)-300)
// turned by a random Y rotation and added to the player's position. The
// original adds the ground height to the player's own height, so flakes
// start higher over raised terrain; that is kept.
func (s *snowState) newFlake(player scene.Point3, ground float64) snowFlake {
	angle := scene.DegreesToRadians(float64(s.random(360)))
	reach := float64(s.random(snowFlakeSpread) - snowFlakeSpread)
	return snowFlake{
		pos: scene.Point3{
			X: player.X + reach*math.Sin(angle),
			Y: player.Y + ground - snowFlakeLift,
			Z: player.Z + reach*math.Cos(angle),
		},
		size:       float64(s.random(15)+25) / 50,
		alphaSpeed: snowFlakeFadeIn,
	}
}

// stepSnowFlake runs Prim3DParticleNoMaster and ProcessAlpha for one frame
// and reports whether the flake is still alive.
func stepSnowFlake(f *snowFlake) bool {
	if f.age >= snowFlakeLife {
		return false
	}
	f.pos.Y += snowFlakeFall
	if f.age == snowFlakeFadeOut {
		f.alphaSpeed = -f.alpha / (snowFlakeLife - snowFlakeFadeOut)
	}
	f.alpha = math.Max(0, math.Min(snowFlakeMaxAlpha, f.alpha+f.alphaSpeed))
	f.age++
	return true
}

// drawSnow renders each flake as an ef_snow sprite, its first action looping
// one motion every four frames.
func (m *System) drawSnow(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	s := effect.snow
	if s == nil || screen == nil || component.SpriteFile == "" {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if frame < 0 {
		return
	}
	// The player stands on the ground, so its height is the ground's too.
	player := scene.Point3{X: origin.X * 5, Y: -origin.Y * 5, Z: origin.Z * 5}
	s.advance(frame, player, player.Y)
	for _, f := range s.flakes {
		position := scene.Point3{X: f.pos.X / 5, Y: -f.pos.Y / 5, Z: f.pos.Z / 5}
		tint := color.RGBA{R: 255, G: 255, B: 255, A: uint8(f.alpha)}
		m.drawEffectParticleSpriteAction(screen, ctx, projection, component.SpriteFile, position, 0, (f.age-1)/4+1, f.size, 0, tint)
	}
}
