package effects

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/render"
)

func TestThiefEnvenomFlarePatterns(t *testing.T) {
	ctx := testutil.ThiefEffectContext()
	start := time.Unix(100, 0)
	m := &System{}
	for _, id := range SkillBeginEffectIDs(db.SkillTFPoison) {
		m.AddAt(ctx, id, 2000000, start)
	}
	if len(m.Active) != 2 || !m.Active[0].Starts.Equal(m.Active[1].Starts) {
		t.Fatal("Envenom must launch two simultaneous flares")
	}
	for i := 0; i < 20; i++ {
		a, b, c := bashFlash(m.Active[0], i, 5)
		x, y, z := bashFlash(m.Active[1], i, 5)
		if a != x || b != y || c != z {
			return
		}
	}
	t.Fatal("Envenom's flares must have independent ray patterns")
}

func TestDetoxifyReducedParticles(t *testing.T) {
	ctx := testutil.ThiefEffectContext()
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 14, 20, 0)
	for _, reduced := range []bool{false, true} {
		ctx.Session.LessEffects = reduced
		m := &System{EffectViews: map[string]*scene.SpriteView{"particle2": testutil.EffectRenderTestSprite()}}
		m.AddAt(ctx, Detoxication, 300, start)
		for _, tc := range []struct{ tick, normal, reduced int }{{20, 5, 2}, {40, 8, 2}, {70, 6, 2}, {98, 1, 1}, {100, 0, 0}} {
			frame := render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
			want := tc.normal
			if reduced {
				want = tc.reduced
			}
			if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != want {
				t.Fatalf("tick %d reduced=%t: got %d particles, want %d", tc.tick, reduced, got, want)
			}
		}
	}
}

func TestThiefEffectsWithClientAssets(t *testing.T) {
	manager := testutil.RealDataManager(t)
	ctx := testutil.ThiefEffectContext()
	ctx.Resources = manager
	ctx.Session.Selected.Job = db.JobThief
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(manager, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Thief sprite missing: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			if _, action, ok := sprites.ResolveAction(view.Body.Act, sprites.ActionPCSkill, dir); !ok || len(action.Animations) == 0 {
				t.Fatal("Thief skill ACT missing")
			}
		}
	}
	m := &System{}
	starts := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 14, 20, 0)
	for _, tc := range []struct{ id, frame, count int }{
		{Steal, 20, 10}, {Detoxication, 20, 5}, {Detoxication, 98, 1},
		{SprinkleSand, 20, 80}, {ThrowItem3, 6, 4},
	} {
		m.Active = nil
		m.AddBetween(ctx, tc.id, 300, 2000000, starts)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts.Add(time.Duration(tc.frame)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != tc.count {
			t.Fatalf("effect %d tick %d: want %d particles, got %d", tc.id, tc.frame, tc.count, commands.Len())
		}
		spec, _ := Lookup(tc.id)
		for _, sound := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestThiefParticleMotion(t *testing.T) {
	effect := Instance{EffectID: Steal, Starts: time.Unix(100, 0)}
	for i := 0; i < 10; i++ {
		first, middle, last := stealParticleAt(effect, i, 0), stealParticleAt(effect, i, 24), stealParticleAt(effect, i, 48)
		if middle.Position.Y <= first.Position.Y || last.Position.Y >= middle.Position.Y || last.size >= first.size || last.Alpha >= first.Alpha {
			t.Fatalf("Steal particle %d must rise, fall, shrink and fade: %+v / %+v / %+v", i, first, middle, last)
		}
		if p := stealParticleAt(effect, i, 49); p.Alpha != 0 || p.size != 0 {
			t.Fatal("Steal must finish at fifty ticks")
		}
	}
	effect.EffectID = Detoxication
	first, last := RisingParticleAt(effect, 0, 0), RisingParticleAt(effect, 0, 38)
	if first.Alpha <= 0 || last.Position.Y <= first.Position.Y || last.Angle <= first.Angle || RisingParticleAt(effect, 0, 39).Alpha != 0 {
		t.Fatal("Detoxify particles must rise, rotate and fade over forty ticks")
	}
	effect.EffectID = SprinkleSand
	for i := 0; i < 80; i++ {
		if sprinkleSandParticleAt(effect, i, 0).Alpha != 0 || sprinkleSandParticleAt(effect, i, 20).Alpha <= 0 || sprinkleSandParticleAt(effect, i, 65).Alpha != 0 {
			t.Fatalf("Sand Attack particle %d has wrong delay or fade", i)
		}
	}
}
