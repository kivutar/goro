package effects

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func TestBlindDrawsPersistentLocalMaskWithEffectsDisabled(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Session.NoEffects = true
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/fullb.tga|blackkey=true":   render.NewImage(32, 32),
		"__effectfile_effect/white02.bmp|blackkey=true": render.NewImage(1, 1),
	}}
	starts := time.Unix(100, 0)
	if !m.AddPersistentBetween(ctx, Blind, ctx.Session.AccountID, 0, starts) {
		t.Fatal("blindness was disabled by /effect")
	}
	frame := render.NewFrame(1920, 1080)
	m.Draw(frame, ctx, testutil.Projection(1920, 1080, 10, 20, 0), starts.Add(time.Minute), false)
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 5 {
		t.Fatalf("mask must draw four gradient quadrants and the opaque surround; got %d draws", commands.Len())
	}
	for i := 0; i < commands.Len(); i++ {
		options := commands.Index(i).FieldByName("Options")
		if !options.FieldByName("AfterBillboards").Bool() || options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() {
			t.Fatal("the screen mask must cover sprites and models without writing depth")
		}
		if options.FieldByName("Blend").Int() != int64(render.BlendSourceOver) {
			t.Fatal("every part of the mask must darken the world, not brighten a quadrant")
		}
		if !options.FieldByName("DisableFog").Bool() {
			t.Fatal("map fog must not recolor the blindness mask")
		}
	}
	m.Remove(Blind, ctx.Session.AccountID)
	frame = render.NewFrame(1920, 1080)
	m.Draw(frame, ctx, testutil.Projection(1920, 1080, 10, 20, 0), starts.Add(time.Minute), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 {
		t.Fatal("cured blindness still darkened the world")
	}
}

func TestBlindMaskStaysCenteredOnMovingActor(t *testing.T) {
	for _, id := range []int{Blind, Devil1, Devil10} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			ctx := testutil.KnightEffectContext()
			m := &System{textures: map[string]*render.Image{
				"__effectfile_effect/fullb.tga|blackkey=true":   render.NewImage(32, 32),
				"__effectfile_effect/white02.bmp|blackkey=true": render.NewImage(1, 1),
			}}
			starts := time.Unix(100, 0)
			if !m.AddPersistentBetween(ctx, id, ctx.Session.AccountID, 0, starts) {
				t.Fatal("could not start blindness")
			}
			for _, yaw := range []float64{0, 90, 225} {
				ctx.World.Player.X++
				ctx.World.Player.Y++
				projection := testutil.ProjectionYaw(1920, 1080, 10, 20, 0, yaw)
				frame := render.NewFrame(1920, 1080)
				m.Draw(frame, ctx, projection, starts.Add(time.Minute), false)
				commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
				if commands.Len() != 5 {
					t.Fatalf("mask draws = %d, want 5", commands.Len())
				}
				want := projection.Project(scene.CellCenter(float64(ctx.World.Player.X)), scene.CellCenter(float64(ctx.World.Player.Y)), 0.07)
				var edges [4]scene.Point2
				for i := 0; i < 4; i++ {
					vertices := commands.Index(i).FieldByName("Vertices")
					project := func(index int) scene.Point2 {
						v := vertices.Index(index)
						return projection.Project(v.FieldByName("X").Float(), v.FieldByName("Z").Float(), v.FieldByName("Y").Float())
					}
					// The transparent corner of fullb.tga is at (0, height).
					center := vertices.Index(3)
					if center.FieldByName("SrcX").Float() != 0 || center.FieldByName("SrcY").Float() != 32 {
						t.Fatal("the clear texture corner must meet the other quadrants at the actor")
					}
					if got := project(3); math.Abs(float64(got.X-want.X)) > .001 || math.Abs(float64(got.Y-want.Y)) > .001 {
						t.Fatalf("mask center = %v, actor anchor = %v", got, want)
					}
					edges[i] = project(1)
				}
				for i := 0; i < 2; i++ {
					if math.Abs(float64(edges[i].X+edges[3-i].X-2*want.X)) > .001 || math.Abs(float64(edges[i].Y+edges[3-i].Y-2*want.Y)) > .001 {
						t.Fatal("opposite quadrants must stay symmetric as the camera rotates")
					}
				}
			}
		})
	}
}
