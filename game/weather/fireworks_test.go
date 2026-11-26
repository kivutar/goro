package weather

import (
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
)

func TestPokJukLaunchesClassicSparks(t *testing.T) {
	var s mapWeatherFireworkState
	player := scene.Point3{X: 500, Y: -10, Z: 700}
	s.ensure("comodo.rsw", player, time.Unix(0, 0))
	if len(s.sparks) != 640 {
		t.Fatalf("sparks = %d, want 160 effects of 4", len(s.sparks))
	}
	for _, p := range s.sparks {
		if p.process > -300 || p.process <= -350 {
			t.Fatalf("start delay = %d, want -300..-349", p.process)
		}
		if p.head != (scene.Point3{X: 485, Y: -10, Z: 700}) {
			t.Fatalf("launch point = %+v, want 15 units beside the player", p.head)
		}
	}
}

func TestPokJukSparkRisesUnseenThenBursts(t *testing.T) {
	var s mapWeatherFireworkState
	player := scene.Point3{}
	s.ensure("comodo.rsw", player, time.Unix(0, 0))
	p := s.sparks[0]
	p.process = 0
	for p.process < pokJukBurstAt {
		s.step(&p, player)
		if p.alpha != 0 {
			t.Fatalf("alpha while rising at %d = %d", p.process, p.alpha)
		}
	}
	s.step(&p, player)
	if !p.bursting || p.alpha != pokJukBurstAlpha {
		t.Fatalf("burst at %d: bursting=%v alpha=%d", p.process, p.bursting, p.alpha)
	}
	if want := -0.25 * 155; math.Abs(p.head.Y-want) > 1e-9 {
		t.Fatalf("burst height = %v, want %v", p.head.Y, want)
	}
	for p.process != 0 {
		s.step(&p, player)
	}
	if p.process != 0 || p.bursting || p.alpha != 0 {
		t.Fatalf("after cycle: process=%d bursting=%v alpha=%d", p.process, p.bursting, p.alpha)
	}
}

func TestPokJukSoundFollowsEffectFrames(t *testing.T) {
	start := time.Unix(0, 0)
	s := mapWeatherFireworkState{starts: start}
	m := &System{}
	var played []time.Time
	schedule := func(at time.Time, _ ...string) { played = append(played, at) }
	for _, frame := range []int{299, 300, 301, 300 + pokJukSoundPeriod} {
		s.scheduleSounds(m, start.Add(time.Duration(frame)*db.EffectFrameDuration), schedule)
	}
	if len(played) != 2 {
		t.Fatalf("sounds played = %d, want 2", len(played))
	}
}
