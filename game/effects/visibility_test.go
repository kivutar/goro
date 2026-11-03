package effects

import (
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestDisabledEffectsDoNotHideActorsOrEssentialBodyFeedback(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Session.NoEffects = true
	now := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	billboard := testutil.EffectRenderTestSprite().Billboards[scene.SpriteBillboardKey{}]
	entry := scene.ActorDrawEntry{Actor: ctx.World.Player, IsPlayer: true, WorldX: 10.5, WorldY: 20.5}
	for _, id := range []int{RedBody, AsuraBody} {
		m := &System{Active: []Instance{{EffectID: id, ActorID: entry.Actor.ID, Starts: now.Add(-time.Second)}}}
		frame := render.NewFrame(800, 600)
		m.DrawActor(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, now, 0)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() == 0 {
			t.Fatal("effect toggle hid the actor")
		}
		if id == RedBody && draws.Index(0).FieldByName("ColorG").Float() != 1 {
			t.Fatal("hidden Berserk effect still tinted the actor")
		}
		if id == AsuraBody && draws.Len() < 2 {
			t.Fatal("essential Soul Link body effect disappeared")
		}
	}
}

func TestEffectTogglePreservesLifetimesAndEssentialFeedback(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Resources: &res.Manager{}, Session: session.New()}
	now := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, .5, .5, 0)
	spec := Spec{Components: []Component{{Kind: Component3D, SpriteFile: "test", SizeStart: 1, SizeEnd: 1, Duration: time.Second}}}
	m := &System{
		EffectViews: map[string]*scene.SpriteView{"test": testutil.EffectRenderTestSprite()},
		specs:       map[int]Spec{Heal: spec, Sight: spec, LockOnTarget: spec, Emotion: spec},
		Active: []Instance{
			{EffectID: Heal, Starts: now, Expires: now.Add(time.Second)},
			{EffectID: Sight, Starts: now, Persistent: true},
			{EffectID: LockOnTarget, Starts: now, Persistent: true},
			{EffectID: Emotion, Starts: now, Persistent: true},
		},
	}
	for _, step := range []struct {
		hidden        bool
		age           time.Duration
		draws, active int
	}{
		{false, 0, 4, 4},
		{true, 0, 2, 4},
		{true, 2 * time.Second, 2, 3},
		{false, 2 * time.Second, 3, 3},
	} {
		ctx.Session.NoEffects = step.hidden
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, now.Add(step.age), false)
		if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != step.draws || len(m.Active) != step.active {
			t.Fatalf("hidden=%t age=%s: draws=%d active=%d, want %d/%d", step.hidden, step.age, got, len(m.Active), step.draws, step.active)
		}
	}
}

func TestEffectVisibilityUsesSessionOverrideAndRetainsBodyFeedback(t *testing.T) {
	ctx := client.Context{}
	if !Enabled(ctx) {
		t.Fatal("effects should be enabled by default")
	}
	ctx.Config.Gameplay.NoEffects = true
	if effectEnabled(ctx, Heal) || !effectEnabled(ctx, JumpBody) || !effectEnabled(ctx, RefineFail) {
		t.Fatal("disabled effects must retain essential feedback")
	}
	ctx.Session = session.New()
	ctx.Session.LessEffects = true
	if !effectEnabled(ctx, Heal) {
		t.Fatal("runtime preference must override configuration independently of /mineffect")
	}
	if effectEnabled(ctx, BashBegin) || !effectEnabled(ctx, BashHit) || !effectEnabled(ctx, Detoxication) {
		t.Fatal("/mineffect must suppress the Bash flare without hiding impacts or Detoxify")
	}
	ctx.Config.Gameplay.LessEffects = true
	ctx.Session.LessEffects = false
	if !effectEnabled(ctx, BashBegin) {
		t.Fatal("the runtime /mineffect preference must override configuration")
	}
	ctx.Session = nil
	ctx.Config.Gameplay.NoEffects = false
	if effectEnabled(ctx, BashBegin) {
		t.Fatal("/mineffect must use configuration when there is no session")
	}
}
