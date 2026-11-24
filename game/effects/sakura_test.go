package effects

import (
	"testing"
	"time"

	"github.com/kivutar/goro/game/scene"
)

func TestSakuraPetalsFollowClassicSpawnAndRespawn(t *testing.T) {
	for _, tc := range []struct {
		effectID, petals int
		minSize, maxSize float64
	}{
		{Sakura, 150, .22, .27},
		{Maple, 100, .35, .40},
	} {
		s := newSakuraPetalState(Instance{EffectID: tc.effectID, Starts: time.Unix(5, 0)})
		master := scene.Point3{X: 100, Y: -20, Z: 50}
		for frame := 0; frame <= 2000; frame += 60 {
			s.advance(frame, master)
			for _, p := range s.petals {
				if p.pos.Y > master.Y || p.pos.Y < master.Y-sakuraPetalRespawn {
					t.Fatalf("effect %d petal height %v outside the 200 units above the master", tc.effectID, p.pos.Y)
				}
			}
		}
		if len(s.petals) != tc.petals {
			t.Fatalf("effect %d petals = %d, want %d", tc.effectID, len(s.petals), tc.petals)
		}
		for _, p := range s.petals {
			if p.size < tc.minSize || p.size > tc.maxSize {
				t.Fatalf("effect %d petal size %v, want %v..%v", tc.effectID, p.size, tc.minSize, tc.maxSize)
			}
		}
	}
}
