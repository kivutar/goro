package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestSpiritSphereRenderedGeometry(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = &res.Manager{}
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/thunder_center.bmp|blackkey=true": render.NewImage(8, 8),
	}}
	projection := testutil.Projection(800, 600, 10, 20, 0)
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		name            string
		id, count, tick int
		centers         [][3]float64
		sizes           []float64
	}{
		{"monk three fixed slots", Chookgi, 3, 360,
			[][3]float64{{10, 5, 21.4}, {11.331479123, 5, 20.432623792}, {10.822899353, 5, 18.867376208}},
			[]float64{.565685425, .565685425, .565685425}},
		{"monk quarter orbit", Chookgi, 1, 90, [][3]float64{{11.4, 5, 20}}, []float64{.565685425}},
		{"baby monk", Chookgi2, 1, 360, [][3]float64{{10, 4.8, 21.1}}, []float64{.367695526}},
		{"coin half speed", Chookgi3, 1, 180, [][3]float64{{11.4, 5, 20}}, []float64{.282842712}},
		{"coin double size", Chookgi3, 6, 720,
			[][3]float64{{10, 5, 21.4}, {11.331479123, 5, 20.432623792}, {10.822899353, 5, 18.867376208}, {9.177100647, 5, 18.867376208}, {8.668520877, 5, 20.432623792}},
			[]float64{.565685425, .282842712, .282842712, .282842712, .282842712}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A known animation snapshot isolates geometry from random wandering.
			e := Instance{EffectID: tc.id, HitCount: tc.count, Starts: start, SpiritSpheres: &spiritSpheres{tick: tc.tick}}
			spec, _ := Lookup(tc.id)
			frame := render.NewFrame(800, 600)
			m.drawSpiritSphereEffect(frame, ctx, projection, spec.Components[0], e, 10, 20, 2, start.Add(time.Duration(tc.tick-1)*db.EffectFrameDuration))
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if commands.Len() != len(tc.centers)*2 {
				t.Fatalf("got %d layers, want %d", commands.Len(), len(tc.centers)*2)
			}
			for i, center := range tc.centers {
				for layer := 0; layer < 2; layer++ {
					command := commands.Index(i*2 + layer)
					for axis, want := range center {
						if got := command.FieldByName("Center").Index(axis).Float(); math.Abs(got-want) > 1e-5 {
							t.Fatalf("sphere %d layer %d axis %d: %g, want %g", i, layer, axis, got, want)
						}
					}
					width := command.FieldByName("Width").Float()
					right := command.FieldByName("RightAxis")
					var size float64
					for axis := 0; axis < 3; axis++ {
						size += math.Pow(right.Index(axis).Float()*width, 2)
					}
					want := tc.sizes[i] / float64(layer+1)
					if math.Abs(math.Sqrt(size)-want) > 1e-5 {
						t.Fatalf("sphere %d layer %d size=%g, want %g", i, layer, math.Sqrt(size), want)
					}
					basisRight, _, _, _ := projection.BillboardBasis(center[0], center[2], center[1])
					dot := (right.Index(0).Float()*basisRight.X + right.Index(1).Float()*basisRight.Y + right.Index(2).Float()*basisRight.Z) * width / want
					if math.Abs(dot-math.Sqrt(.5)) > 1e-5 {
						t.Fatal("sphere texture must remain at 45 degrees while orbiting")
					}
					if render.Blend(command.FieldByName("Options").FieldByName("Blend").Int()) != render.BlendLighter {
						t.Fatal("sphere layers must use additive blending")
					}
				}
			}
		})
	}
}

func TestSpiritSphereFadeAndColors(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = &res.Manager{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	for _, id := range []int{Chookgi, Chookgi2} {
		m := &System{textures: map[string]*render.Image{
			"__effectfile_effect/thunder_center.bmp|blackkey=true": render.NewImage(8, 8),
		}}
		m.AddBetween(ctx, id, 300, 0, start)
		for _, tc := range []struct{ frame, alpha int }{{-1, 0}, {0, 1}, {99, 100}, {199, 200}, {400, 200}} {
			frame := render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration), false)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if tc.frame < 0 {
				if commands.Len() != 0 {
					t.Fatal("spheres drew before their start time")
				}
				continue
			}
			if commands.Len() != 10 {
				t.Fatalf("got %d layers, want five paired spheres", commands.Len())
			}
			for i := 0; i < commands.Len(); i++ {
				want := [4]float64{0, 0, 1, float64(tc.alpha) / 255}
				if i%2 == 1 {
					want[0], want[1], want[2] = 1, 1, 0
				}
				for channel, field := range []string{"ColorR", "ColorG", "ColorB", "ColorA"} {
					if got := commands.Index(i).FieldByName(field).Float(); math.Abs(got-want[channel]) > 1e-6 {
						t.Fatalf("effect %d frame %d layer %d %s=%g, want %g", id, tc.frame, i, field, got, want[channel])
					}
				}
			}
		}
	}
}
