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

func TestStarGladiatorBodyOverlays(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	billboard := &scene.Billboard{Image: render.NewImage(40, 80), AnchorX: 20, AnchorY: 80}
	entry := scene.ActorDrawEntry{Actor: ctx.World.Player, WorldX: 10.5, WorldY: 20.5}
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, id := range []int{544, 488} {
		m := &System{}
		m.AddAt(ctx, id, 2000000, start)
		frame := render.NewFrame(800, 600)
		m.DrawActor(ctx, frame, projection, entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(200*db.EffectFrameDuration), 0)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != 3 {
			t.Fatal("native body effects need two additive copies over the ordinary sprite")
		}
		wantR, wantB := 1.0, 5.0/255
		if id == 488 {
			wantR, wantB = 55.0/255, 1
		}
		for i := 1; i < 3; i++ {
			if math.Abs(draws.Index(i).FieldByName("ColorR").Float()-wantR) > .00001 || math.Abs(draws.Index(i).FieldByName("ColorB").Float()-wantB) > .00001 {
				t.Fatal("body flash palette differs from native")
			}
		}
	}
}

func TestStarFusionFlightUsesServerOptionAndElapsedTicks(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	actor := ctx.World.Player
	actor.Job, actor.EffectState = db.JobStar, db.EffectStateFlying
	if scene.VisualJobForEffectState(db.JobStar2, 0) != db.JobStar || scene.VisualJobForEffectState(db.JobStar2B, 0) != db.JobStarB {
		t.Fatal("server flying class IDs must normalize when the option ends")
	}
	if scene.ActorVisualJob(actor) != db.JobStar2 {
		t.Fatal("Fusion did not select its flying ACT")
	}
	if scene.VisualJobForEffectState(db.JobStarB, db.EffectStateFlying) != db.JobStar2B {
		t.Fatal("baby Fusion visual job")
	}
	start := time.Unix(100, 0)
	for _, step := range []int{1, 15, 300} {
		m := &System{}
		m.FusionHeight(actor, start, false)
		for tick := step; tick <= 300; tick += step {
			m.FusionHeight(actor, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		}
		if m.StarFlights[actor.ID].height != 20 {
			t.Fatal("Fusion flight height depends on render rate")
		}
		actor.EffectState = 0
		if scene.ActorVisualJob(actor) != db.JobStar {
			t.Fatal("cleared Fusion retained its flying job")
		}
		m.FusionHeight(actor, start.Add(320*db.EffectFrameDuration), false)
		if len(m.StarFlights) != 0 {
			t.Fatal("Fusion did not land and release its render state")
		}
		actor.EffectState = db.EffectStateFlying
	}
	m := &System{}
	actor.Sitting = true
	m.FusionHeight(actor, start, false)
	m.FusionHeight(actor, start.Add(1500*db.EffectFrameDuration), false)
	if m.StarFlights[actor.ID].height != 100 {
		t.Fatal("sitting Fusion must rise higher")
	}
	actor.Opt3State = db.Opt3Warm
	m.FusionHeight(actor, start.Add(2700*db.EffectFrameDuration), false)
	if m.StarFlights[actor.ID].height != 20 {
		t.Fatal("Warmth must keep Fusion at its ordinary height")
	}
}
