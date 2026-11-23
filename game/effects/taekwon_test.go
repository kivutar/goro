package effects

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func TestTaekwonBodyTransformsOnlyChangeRendering(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	entry := scene.ActorDrawEntry{Actor: actor, WorldX: 20.5, WorldY: 20.5}
	billboard := &scene.Billboard{Image: render.NewImage(40, 80), AnchorX: 20, AnchorY: 80}
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct{ id, tick int }{{413, 22}, {414, 7}, {415, 45}, {426, 5}, {445, 40}, {446, 20}, {458, 15}, {466, 25}} {
		m := &System{}
		m.AddAt(ctx, tc.id, actor.ID, start)
		frame := render.NewFrame(800, 600)
		m.DrawActor(ctx, frame, projection, entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), 0)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != 1 {
			t.Fatalf("effect %d should transform the single body: %d draws", tc.id, draws.Len())
		}
		for _, name := range []string{"Center", "RightAxis", "UpAxis", "DepthUpAxis"} {
			for axis := 0; axis < 3; axis++ {
				v := draws.Index(0).FieldByName(name).Index(axis).Float()
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Fatalf("effect %d produced invalid %s", tc.id, name)
				}
			}
		}
		if tc.id == 413 {
			// Keeping the feet fixed requires anchorY/height to remain one.
			if math.Abs(draws.Index(0).FieldByName("AnchorY").Float()-80) > .0001 {
				t.Fatal("Heel Drop squash lifted the victim off the floor")
			}
		}
		if !reflect.DeepEqual(ctx.World.Actors[300], actor) || billboard.AnchorY != 80 {
			t.Fatal("body effect mutated the actor or shared sprite")
		}
	}
}

func TestTaekwonParticlesDoNotDependOnFrameRate(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	ctx.World.Player.Moving = true
	ctx.World.Player.MoveStarted, ctx.World.Player.MoveDuration = start, 2*time.Second
	ctx.World.Player.FromX, ctx.World.Player.FromY = 10, 20
	ctx.World.Player.ToX, ctx.World.Player.ToY = 20, 20
	var wantFootprints []Instance
	var wantJump taekwonJumpParticles
	for _, step := range []int{1, 5, 60} {
		m := &System{}
		m.AddAt(ctx, Run, 100, start)
		jump := taekwonJumpParticles{tick: 35}
		for tick := step; tick <= 60; tick += step {
			at := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.AdvanceTaekwonFootprints(ctx, at)
			jump.advance(ctx, Instance{EffectID: Peong, ActorID: 100, Starts: start}, at)
		}
		if step == 1 {
			wantFootprints, wantJump = m.Active[1:], jump
			continue
		}
		if !reflect.DeepEqual(m.Active[1:], wantFootprints) || jump != wantJump {
			t.Fatalf("particle history changed with an update every %d ticks", step)
		}
	}
}

func TestTaekwonFootprintsRemainAfterRunStops(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	m := &System{}
	m.AddAt(ctx, Run, 100, start)
	m.AdvanceTaekwonFootprints(ctx, start.Add(7*db.EffectFrameDuration))
	if len(m.Active) != 1 {
		t.Fatal("stationary Run left footprints")
	}
	for tick := 14; tick <= 21; tick += 7 {
		ctx.World.Player.X++
		m.AdvanceTaekwonFootprints(ctx, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	if len(m.Active) != 3 || m.Active[1].EffectID != Foot4 || m.Active[2].EffectID != Foot3 {
		t.Fatalf("Run must alternate left/right footprints: %+v", m.Active)
	}
	m.Remove(Run, 100)
	ctx.World.Player.X = 50
	m.AdvanceTaekwonFootprints(ctx, start.Add(28*db.EffectFrameDuration))
	if len(m.Active) != 2 || m.Active[0].ActorID != 0 || m.Active[0].ActorOrigin.X != 11.5 || m.Active[1].ActorOrigin.X != 12.5 {
		t.Fatal("stopping Run must leave its footprints where they were emitted")
	}
}
