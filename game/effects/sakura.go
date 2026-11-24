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

// CRagEffect::Sakura and CRagEffect::Maple launch one PP_SAKURA petal every
// second frame, for the first 300 (Maple: 200) frames. Petals live as long
// as the effect: each frame they fall, sway by a random walk
// (CEffectPrim::PrimSakura), and restart 200 units above the effect's
// master once they drop below it. Positions are absolute client units;
// client Y points down.

const (
	sakuraPetalSpawnEnd = 300
	maplePetalSpawnEnd  = 200
	sakuraPetalAlpha    = 200
	sakuraPetalSpread   = 300
	sakuraPetalRespawn  = 200
	// sakuraPetalMaxCatchUp bounds the frames stepped in one draw, so a
	// long pause skips ahead instead of stalling.
	sakuraPetalMaxCatchUp = 120
)

type sakuraPetal struct {
	pos    scene.Point3
	fall   float64
	size   float64
	walk   [4]float64 // m_GI.height[0..3]: two sway angles and their targets
	action int
	born   int
}

type sakuraPetalState struct {
	maple  bool
	frame  int
	rng    *rand.Rand
	petals []sakuraPetal
}

func newSakuraPetalState(effect Instance) *sakuraPetalState {
	seed := int64(deterministicUnit(effect, 1)*(1<<31)) ^ effect.Starts.UnixNano()
	return &sakuraPetalState{maple: effect.EffectID == Maple, rng: rand.New(rand.NewSource(seed))}
}

// random is the client's random(n): 0 when n is not positive.
func (s *sakuraPetalState) random(n int) int {
	if n <= 0 {
		return 0
	}
	return s.rng.Intn(n)
}

func (s *sakuraPetalState) advance(target int, master scene.Point3) {
	if target-s.frame > sakuraPetalMaxCatchUp {
		s.frame = target - sakuraPetalMaxCatchUp
	}
	for ; s.frame <= target; s.frame++ {
		for i := range s.petals {
			s.stepPetal(&s.petals[i], master)
		}
		spawnEnd := sakuraPetalSpawnEnd
		if s.maple {
			spawnEnd = maplePetalSpawnEnd
		}
		if s.frame%2 == 0 && s.frame < spawnEnd {
			s.petals = append(s.petals, s.newPetal(master))
		}
	}
}

func (s *sakuraPetalState) newPetal(master scene.Point3) sakuraPetal {
	p := sakuraPetal{born: s.frame}
	if s.maple {
		p.fall = float64(s.random(4)+2) * 0.03
		p.size = float64(s.random(6)+35) / 100
	} else {
		p.fall = float64(s.random(2)+2) * 0.1
		p.size = float64(s.random(6)+22) / 100
	}
	for i := range p.walk {
		p.walk[i] = float64(s.random(360))
	}
	s.respawn(&p, master)
	if !s.maple {
		p.action = s.random(3)
	}
	return p
}

func (s *sakuraPetalState) respawn(p *sakuraPetal, master scene.Point3) {
	p.pos = scene.Point3{
		X: master.X + float64(s.random(sakuraPetalSpread)-sakuraPetalSpread/2),
		Y: master.Y - sakuraPetalRespawn,
		Z: master.Z + float64(s.random(sakuraPetalSpread)-sakuraPetalSpread/2),
	}
}

func (s *sakuraPetalState) stepPetal(p *sakuraPetal, master scene.Point3) {
	for i := 0; i < 4; i += 2 {
		if p.walk[i+1] > p.walk[i] {
			p.walk[i] = math.Min(p.walk[i]+3, 359)
			if p.walk[i] > p.walk[i+1] {
				p.walk[i+1] = float64(s.random(int(p.walk[i])))
			}
		} else {
			p.walk[i] = math.Max(p.walk[i]-2, 0)
			if p.walk[i] <= p.walk[i+1] {
				p.walk[i+1] = float64(359 - s.random(int(p.walk[i])))
			}
		}
	}
	swayX, swayZ := 0.24, 0.30
	if s.maple {
		swayX, swayZ = 0.12, 0.15
	}
	p.pos.Y += p.fall
	p.pos.X += swayX * sinTableDegrees(p.walk[0])
	p.pos.Z += swayZ * sinTableDegrees(p.walk[2])
	if p.pos.Y > master.Y {
		s.respawn(p, master)
	}
}

// sinTableDegrees reads g_sinTable, indexed by whole degrees.
func sinTableDegrees(degrees float64) float64 {
	return math.Sin(scene.DegreesToRadians(float64(int(degrees))))
}

// drawSakuraPetals renders each petal as a PP_SAKURA sprite: alpha 200, its
// action looping one motion every four frames.
func (m *System) drawSakuraPetals(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	s := effect.sakuraPetals
	if s == nil || screen == nil || component.SpriteFile == "" {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if frame < 0 {
		return
	}
	master := scene.Point3{X: origin.X * 5, Y: -origin.Y * 5, Z: origin.Z * 5}
	s.advance(frame, master)
	tint := color.RGBA{R: 255, G: 255, B: 255, A: sakuraPetalAlpha}
	for _, p := range s.petals {
		position := scene.Point3{X: p.pos.X / 5, Y: -p.pos.Y / 5, Z: p.pos.Z / 5}
		motion := (s.frame-1-p.born)/4 + 1
		m.drawEffectParticleSpriteAction(screen, ctx, projection, component.SpriteFile, position, p.action, motion, p.size, 0, tint)
	}
}
