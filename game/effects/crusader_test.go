package effects

import (
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
