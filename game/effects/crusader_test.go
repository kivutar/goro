package effects

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func TestShieldBoomerangFollowsMovingCasterAcrossSkippedFrames(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	caster := &ctx.World.Player
	caster.Moving, caster.MoveStarted, caster.MoveDuration = true, start, 3*time.Second
	caster.FromX, caster.FromY, caster.ToX, caster.ToY = 20, 20, 20, 40
	projection := testutil.Projection(800, 600, 20, 20, 0)
	var want shieldFlight
	for _, interval := range []int{1, 10, 70} {
		for _, hidden := range []bool{false, true} {
			t.Run(fmt.Sprintf("interval=%d/hidden=%t", interval, hidden), func(t *testing.T) {
				ctx.Session.NoEffects = false
				m := &System{textures: map[string]*render.Image{
					"__effectfile_effect/shield_boomerang.bmp|blackkey=true": render.NewImage(1, 1),
				}}
				m.AddBetween(ctx, ShieldBoomer, 300, 2000000, start)
				ctx.Session.NoEffects = hidden
				for tick := 0; tick <= 70; tick += interval {
					m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
				}
				flight := *m.Active[0].Shield
				if interval == 1 && !hidden {
					want = flight
					if flight.tick != 71 || flight.Phase == 0 {
						t.Fatal("shield must have begun its return flight")
					}
				} else if flight != want {
					t.Fatalf("return flight or trail depends on drawing frequency or visibility: got %+v, want %+v", flight, want)
				}
				ctx.Session.NoEffects = false
				m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(70*db.EffectFrameDuration), true)
				if *m.Active[0].Shield != flight {
					t.Fatal("drawing the other render pass advanced the shield twice")
				}
			})
		}
	}
}

func TestCrusaderDefenseRingsKeepTheirOpenArc(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	for _, id := range []int{CrusaderDef, ReflectShield} {
		spec, _ := Lookup(id)
		texture := render.NewImage(21, 21)
		m := &System{textures: map[string]*render.Image{
			"__effectfile_" + spec.Components[0].TextureFile + "|blackkey=true": texture,
		}}
		m.AddAt(ctx, id, 300, start)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, scene.Projection{}, start.Add(30*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 {
			t.Fatalf("effect %d: got %d draws", id, commands.Len())
		}
		vertices := commands.Index(0).FieldByName("Vertices")
		if vertices.Len() != 4*8*4 {
			t.Fatalf("effect %d: each of the four rings must have eight faces", id)
		}
		first, last := vertices.Index(0), vertices.Index(7*4+1)
		angle := math.Atan2(last.FieldByName("Z").Float()-20.5, last.FieldByName("X").Float()-30.5) -
			math.Atan2(first.FieldByName("Z").Float()-20.5, first.FieldByName("X").Float()-30.5)
		if math.Abs(math.Mod(angle+2*math.Pi, 2*math.Pi)-scene.DegreesToRadians(320)) > .0001 ||
			math.Abs(last.FieldByName("SrcX").Float()-8) > .0001 {
			t.Fatalf("effect %d: ring must leave a 40-degree gap and use eight of 21 texture slices", id)
		}
	}
}

func TestDevotionSnapshotAndRemoval(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &System{}
	m.AddAt(ctx, Devotion, 2000000, time.Now())
	notify := network.ActorDevotion{SourceID: 2000000, Targets: [5]uint32{300, 300}}
	m.ApplyActorDevotion(ctx, notify)
	m.ApplyActorDevotion(ctx, notify)
	if len(m.Active) != 2 || !m.Active[1].Expires.IsZero() || m.Active[1].TargetID != 300 {
		t.Fatalf("snapshots must replace links without duplicating them: %+v", m.Active)
	}
	m.RemoveActor(300)
	if len(m.Active) != 1 || m.Active[0].EffectID != Devotion {
		t.Fatal("vanishing target left a link or removed the cast animation")
	}
	m.ApplyActorDevotion(ctx, notify)
	m.ApplyActorDevotion(ctx, network.ActorDevotion{SourceID: 2000000})
	if len(m.Active) != 1 {
		t.Fatal("empty snapshot did not clear the caster's links")
	}
	m.ApplyActorDevotion(ctx, notify)
	m.RemoveActor(2000000)
	if len(m.Active) != 0 {
		t.Fatal("vanishing caster left Devotion effects")
	}
}
