package effects

import (
	"image/color"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestSoulLinkerTargetFlashAndShake(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &System{Active: []Instance{{EffectID: BlueHit, ActorID: 300, Starts: starts}}}
	billboard := &scene.Billboard{Image: render.NewImage(64, 64), AnchorX: 32, AnchorY: 60}
	projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
	entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 18.5, WorldY: 20.5, Scale: 1, Shadow: scene.FullLight}
	for _, tc := range []struct{ frame, count int }{{-1, 1}, {0, 1}, {4, 3}, {16, 3}, {17, 1}} {
		frame := render.NewFrame(800, 600)
		m.DrawActor(ctx, frame, projection, entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, starts.Add(time.Duration(tc.frame)*db.EffectFrameDuration), 0)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.count {
			t.Fatalf("tick %d: body draws %d, want %d", tc.frame, draws.Len(), tc.count)
		}
		if tc.count > 0 && draws.Index(0).FieldByName("ColorB").Float() != 1 {
			t.Fatal("base body must remain visible under the overlay")
		}
		if tc.count > 0 {
			normal := render.NewFrame(800, 600)
			m.DrawActorSpriteBillboard(normal, projection, billboard, entry.WorldX, entry.WorldY, entry.WorldZ, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver)
			base := reflect.ValueOf(normal).Elem().FieldByName("worldBillboards").Index(0)
			for _, name := range []string{"Center", "RightAxis", "UpAxis", "DepthUpAxis"} {
				for axis := 0; axis < 3; axis++ {
					if math.Abs(base.FieldByName(name).Index(axis).Float()-draws.Index(0).FieldByName(name).Index(axis).Float()) > 1e-5 {
						t.Fatalf("flash changed normal sprite geometry: %s", name)
					}
				}
			}
			for i := 1; i < tc.count; i++ {
				if draws.Index(i).FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendLighter) {
					t.Fatal("blue body overlays must use additive blending")
				}
			}
		}
	}
	m.Active[0].EffectID = QuakeBody3
	for _, tick := range []int{30, 31, 50, 51} {
		frame := render.NewFrame(800, 600)
		m.DrawActor(ctx, frame, projection, entry, billboard, 1, 1, scene.FullLight, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, starts.Add(time.Duration(tick)*db.EffectFrameDuration), 0)
		draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
		active := math.Abs(draw.FieldByName("AnchorX").Float()-32) > 1e-5 || math.Abs(draw.FieldByName("AnchorY").Float()-60) > 1e-5
		if active != (tick > 30 && tick <= 50) {
			t.Fatalf("shake active at tick %d: %v", tick, active)
		}
	}
	if billboard.AnchorX != 32 || billboard.AnchorY != 60 || ctx.World.Actors[300].X != 18 {
		t.Fatal("shake mutated sprite or actor")
	}
}

func TestSoulLinkerRealEffectAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to check real effect assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	m := &System{}
	for _, id := range []int{Stin, Stin3, HitLine4, Stin2, HitLine6, HitTexture, AsuraBodyMonster, Hated, Kaizel, BlueBody, ReflectBody, SmaReady, SoulLink, AsuraBody, BlueCasting, TaeReady, StormKick4, BabyBodyBack, M07} {
		spec, _ := Lookup(id)
		for _, c := range spec.Components {
			if c.SpriteFile != "" && m.EffectSpriteView(manager, c.SpriteFile) == nil {
				t.Errorf("effect %d missing sprite %s", id, c.SpriteFile)
			}
			paths := append([]string{c.TextureFile}, c.TextureFiles...)
			for _, p := range paths {
				if p != "" && m.EffectFileTexture(manager, p) == nil {
					t.Errorf("effect %d missing %s", id, p)
				}
			}
		}
		for _, s := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + s); err != nil {
				t.Error(err)
			}
		}
	}
	for _, skill := range []uint16{db.SkillSLSka, db.SkillSLSwoo, db.SkillMOExplosionspirits} {
		for _, sound := range db.SkillEffects[skill].SFX {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
}
