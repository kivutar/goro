package effects

import (
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/game/scene"
)

func TestSnowFollowsClassicParticleLife(t *testing.T) {
	s := newSnowState(Instance{EffectID: Snow, Starts: time.Unix(0, 0)})
	player := scene.Point3{X: 1000, Y: -20, Z: 500}
	for frame := 0; frame <= snowFlakeLife+50; frame += 10 {
		s.advance(frame, player, player.Y)
	}
	// Two flakes per frame, each alive for 320 frames.
	if got, want := len(s.flakes), snowFlakesPerFrame*snowFlakeLife; got != want {
		t.Fatalf("flakes = %d, want %d", got, want)
	}
	for _, f := range s.flakes {
		if f.size < 0.5 || f.size > 0.78 {
			t.Fatalf("size = %v, want 0.5..0.78", f.size)
		}
		reach := math.Hypot(f.pos.X-player.X, f.pos.Z-player.Z)
		if reach > snowFlakeSpread+1e-9 {
			t.Fatalf("reach = %v, want <= %d", reach, snowFlakeSpread)
		}
		start := player.Y + player.Y - snowFlakeLift
		if want := start + snowFlakeFall*float64(f.age); math.Abs(f.pos.Y-want) > 1e-9 {
			t.Fatalf("y at age %d = %v, want %v", f.age, f.pos.Y, want)
		}
	}
}

func TestSnowFlakeAlphaFadesInAndOut(t *testing.T) {
	f := snowFlake{alphaSpeed: snowFlakeFadeIn}
	alphas := make([]float64, 0, snowFlakeLife)
	for stepSnowFlake(&f) {
		alphas = append(alphas, f.alpha)
	}
	if len(alphas) != snowFlakeLife {
		t.Fatalf("life = %d, want %d", len(alphas), snowFlakeLife)
	}
	if alphas[0] != snowFlakeFadeIn || alphas[9] != snowFlakeMaxAlpha || alphas[snowFlakeFadeOut-1] != snowFlakeMaxAlpha {
		t.Fatalf("fade in = %v, %v, %v", alphas[0], alphas[9], alphas[snowFlakeFadeOut-1])
	}
	if last := alphas[len(alphas)-1]; math.Abs(last) > 1e-6 {
		t.Fatalf("final alpha = %v, want 0", last)
	}
}
