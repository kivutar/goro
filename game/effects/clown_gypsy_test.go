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

func TestClownGypsyBodyEffectRendering(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		id, frame, copies int
		tint              color.RGBA
	}{
		{PinkBody, 1000, 2, color.RGBA{255, 89, 182, 255}},
		{ChemicalBody, 0, 2, color.RGBA{0, 0, 255, 255}},
		{ChemicalBody, 1, 2, color.RGBA{255, 255, 255, 255}},
		{ChemicalBody, 30, 2, color.RGBA{0, 0, 255, 255}},
		{ChemicalBody, 31, 1, color.RGBA{8, 8, 255, 255}},
		{ChemicalBody, 40, 1, color.RGBA{80, 80, 255, 255}},
		{ChemicalBody, 61, 1, color.RGBA{248, 248, 255, 255}},
		{ChemicalBody, 62, 1, color.RGBA{255, 255, 255, 255}},
	} {
		m := &System{}
		m.AddAt(ctx, tc.id, 300, start)
		frame := render.NewFrame(800, 600)
		billboard := &scene.Billboard{Image: render.NewImage(10, 20), AnchorX: 5, AnchorY: 20}
		entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
		m.DrawActor(ctx, frame, testutil.Projection(800, 600, 15, 20, 0), entry, billboard, 1, 1, 1,
			color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration), 0)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.copies {
			t.Fatalf("body effect %d frame %d: %d layers, want %d", tc.id, tc.frame, draws.Len(), tc.copies)
		}
		body := draws.Index(draws.Len() - 1)
		if math.Abs(body.FieldByName("ColorR").Float()-float64(tc.tint.R)/255) > .0001 ||
			math.Abs(body.FieldByName("ColorG").Float()-float64(tc.tint.G)/255) > .0001 ||
			math.Abs(body.FieldByName("ColorB").Float()-float64(tc.tint.B)/255) > .0001 {
			t.Fatalf("body effect %d frame %d: wrong tint, want %v", tc.id, tc.frame, tc.tint)
		}
	}
}

func TestClownGypsyRecoveryPreservesStatusTint(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	base := color.RGBA{64, 128, 192, 255}
	recovering := color.RGBA{80, 80, 255, 255}
	for _, tc := range []struct {
		name         string
		body, health uint16
		want         color.RGBA
	}{
		{"freeze", db.BodyStateFreeze, 0, base},
		{"stone", db.BodyStateStone, 0, base},
		{"petrifying", db.BodyStateStonewait, 0, base},
		{"poison", 0, db.HealthStatePoison, base},
		{"deadly poison", 0, db.HealthStateDpoison, base},
		{"curse", 0, db.HealthStateCurse, base},
		{"stun", db.BodyStateStun, 0, recovering},
		{"silence", 0, db.HealthStateSilence, recovering},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &System{}
			m.AddAt(ctx, ChemicalBody, 300, start)
			entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
			// A status received after the flashing phase must own the body's color.
			entry.Actor.BodyState, entry.Actor.HealthState = tc.body, tc.health
			frame := render.NewFrame(800, 600)
			billboard := &scene.Billboard{Image: render.NewImage(10, 20), AnchorX: 5, AnchorY: 20}
			m.DrawActor(ctx, frame, testutil.Projection(800, 600, 15, 20, 0), entry, billboard, 1, 1, 1,
				base, render.BlendSourceOver, start.Add(40*db.EffectFrameDuration), 0)
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			body := draws.Index(draws.Len() - 1)
			if math.Abs(body.FieldByName("ColorR").Float()-float64(tc.want.R)/255) > .0001 ||
				math.Abs(body.FieldByName("ColorG").Float()-float64(tc.want.G)/255) > .0001 ||
				math.Abs(body.FieldByName("ColorB").Float()-float64(tc.want.B)/255) > .0001 {
				t.Fatalf("incorrect recovery tint, want %v", tc.want)
			}
		})
	}
}

func TestClownGypsyArrowVulcanSpawnPositions(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	actor := ctx.World.Player
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 10, 20, 10, 40
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for _, ticks := range [][]int{{0, 1, 2, 3, 4, 5, 6, 7, 8}, {8}} {
		ctx.World.Player = actor
		ctx.Session.NoEffects = false
		m := &System{}
		if !m.AddBetween(ctx, TripleAttack3, 300, 2000000, start) {
			t.Fatal("failed to create Arrow Vulcan rays")
		}
		ctx.Session.NoEffects = true
		for _, tick := range ticks {
			m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		}
		effect := m.Active[0]
		if effect.arrowVulcan.count != 9 {
			t.Fatalf("frames %v: captured %d batches, want nine", ticks, effect.arrowVulcan.count)
		}
		for i, origin := range effect.arrowVulcan.origins {
			wantZ := scene.CellCenter(20 + 10*(time.Duration(i)*db.EffectFrameDuration).Seconds())
			if origin.X != 10.5 || math.Abs(origin.Z-wantZ) > 1e-6 {
				t.Fatalf("frames %v: batch %d origin %v, want (10.5, %f)", ticks, i, origin, wantZ)
			}
			from, to, _, _ := arrowVulcanLine(effect, i*4, 100)
			if math.Abs(math.Atan2(to.Z-from.Z, to.X-from.X)-math.Atan2(20.5-wantZ, 10)) > 1e-6 {
				t.Fatalf("batch %d does not aim from its spawn position toward the original target", i)
			}
		}
		from, to, _, _ := arrowVulcanLine(effect, 32, 100)
		ctx.World.Player.Moving, ctx.World.Player.X, ctx.World.Player.Y = false, 60, 60
		target := ctx.World.Actors[300]
		target.X, target.Y = 50, 50
		ctx.World.Actors[300] = target
		m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(100*db.EffectFrameDuration), false)
		gotFrom, gotTo, _, _ := arrowVulcanLine(m.Active[0], 32, 100)
		if gotFrom != from || gotTo != to {
			t.Fatal("existing rays must not move with either actor")
		}
		target.X, target.Y = 20, 20
		ctx.World.Actors[300] = target
	}
}
