package actors

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func TestBlindFollowsLocalStateWithoutRestarting(t *testing.T) {
	w := world.New()
	w.Player = world.Actor{ID: 100, HasState: true, HealthState: db.HealthStateBlind}
	w.Actors[300] = world.Actor{ID: 300, HasState: true, HealthState: db.HealthStateBlind}
	ctx := client.Context{World: w, Session: &session.Session{AccountID: 100, CharID: 200}}
	fx := &effects.System{}
	var sounds []string
	m := &System{}
	m.Bind(Bindings{Effects: fx, ScheduleSoundAtActor: func(_ time.Time, _ uint32, paths ...string) { sounds = append(sounds, paths...) }})
	m.SyncActorAilments(ctx, 100, world.Actor{}, true)
	if len(fx.Active) != 1 || fx.Active[0].EffectID != effects.Blind || !fx.Active[0].Persistent || !fx.Active[0].Expires.IsZero() {
		t.Fatal("blindness did not start a persistent local screen mask")
	}
	started := fx.Active[0].Starts
	m.SyncActorAilments(ctx, 200, w.Player, true)
	if len(fx.Active) != 1 || !fx.Active[0].Starts.Equal(started) || len(sounds) != 1 || sounds[0] != "_blind.wav" {
		t.Fatal("duplicate state restarted the mask or sound")
	}
	m.SyncActorAilments(ctx, 300, world.Actor{}, true)
	if len(fx.Active) != 1 || len(sounds) != 2 {
		t.Fatal("remote blindness changed the local screen mask or lost its positional sound")
	}
	previous := w.Player
	w.Player.HealthState = 0
	m.SyncActorAilments(ctx, 200, previous, true)
	if len(fx.Active) != 0 {
		t.Fatal("screen mask survived curing blindness")
	}
	w.Player.HealthState = db.HealthStateBlind
	m.SyncActorAilments(ctx, 100, w.Player, false)
	if len(fx.Active) != 1 || len(sounds) != 2 {
		t.Fatal("map entry failed to restore the screen mask silently")
	}
}

func TestAilmentIndicatorsLoopUntilTheStateEnds(t *testing.T) {
	for _, tc := range []struct {
		name        string
		body        uint16
		health      uint16
		file, sound string
		action      int
	}{
		{"stun", db.BodyStateStun, 0, "Status-stun", "_stun.wav", 0},
		{"sleep", db.BodyStateSleep, 0, "Status-sleep", "", 0},
		{"silence", 0, db.HealthStateSilence, "emotion", "_silence.wav", 10},
		{"curse", 0, db.HealthStateCurse, "Status-curse", "_curse.wav", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := world.New()
			w.Player = world.Actor{ID: 100, Job: db.JobAcolyte, HasState: true, BodyState: tc.body, HealthState: tc.health}
			ctx := client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 100, CharID: 200, NoEffects: true}}
			first, second := render.NewImage(2, 2), render.NewImage(3, 3)
			view := &scene.SpriteView{
				Act: &res.ACT{Actions: make([]res.ACTAction, tc.action+1)},
				Billboards: map[scene.SpriteBillboardKey]*scene.Billboard{
					{ActionIndex: tc.action, Motion: 0}: {Image: first},
					{ActionIndex: tc.action, Motion: 1}: {Image: second},
				},
			}
			view.Act.Actions[tc.action] = res.ACTAction{DelayMS: 20, Animations: make([]res.ACTAnimation, 2)}
			fx := &effects.System{EffectViews: map[string]*scene.SpriteView{tc.file: view}}
			var sounds []string
			m := &System{}
			m.Bind(Bindings{Effects: fx, ScheduleSoundAtActor: func(_ time.Time, _ uint32, paths ...string) { sounds = append(sounds, paths...) }})
			m.SyncActorAilments(ctx, 100, world.Actor{}, true)
			visual := m.Ailments[100]
			started := visual.BodyStarted
			if tc.health != 0 {
				started = visual.SilenceStarted
			}
			if tc.health == db.HealthStateCurse {
				started = visual.CurseStarted
			}
			m.SyncActorAilments(ctx, 200, w.Player, true)
			wantSounds := 0
			if tc.sound != "" {
				wantSounds = 1
			}
			if len(sounds) != wantSounds || wantSounds > 0 && sounds[0] != tc.sound || m.Ailments[100] != visual {
				t.Fatal("duplicate ailment state restarted the sound or animation")
			}
			body := &scene.Billboard{Image: render.NewImage(40, 90)}
			entry := scene.ActorDrawEntry{Actor: w.Player, IsPlayer: true, WorldX: 10.5, WorldY: 20.5}
			for i, elapsed := range []time.Duration{0, 20 * time.Millisecond, 40 * time.Millisecond, time.Minute} {
				frame := render.NewFrame(800, 600)
				m.drawActorAilments(frame, ctx, testutil.Projection(800, 600, 10, 20, 0), entry, body, 1, started.Add(elapsed))
				draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
				if draws.Len() != 1 {
					t.Fatalf("ailment indicator missing at %s", elapsed)
				}
				if got := draws.Index(0).FieldByName("Width").Float(); got != float64(2+i%2) && i < 3 {
					t.Fatalf("wrong ailment animation frame at %s: width %v", elapsed, got)
				}
			}
			previous := w.Player
			w.Player.BodyState = 0
			w.Player.HealthState = 0
			m.SyncActorAilments(ctx, 100, previous, true)
			entry.Actor = w.Player
			frame := render.NewFrame(800, 600)
			m.drawActorAilments(frame, ctx, testutil.Projection(800, 600, 10, 20, 0), entry, body, 1, started.Add(time.Minute))
			if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() != 0 || len(m.Ailments) != 0 {
				t.Fatal("ailment indicator survived curing the status")
			}
		})
	}
}

func TestFreezeBlockPersistsAndBreakAnimationPlaysOnce(t *testing.T) {
	w := world.New()
	w.Player = world.Actor{ID: 100, X: 10, Y: 20, HasState: true, BodyState: db.BodyStateFreeze}
	ctx := client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 100, CharID: 200}}
	view := &scene.SpriteView{
		Act: &res.ACT{Actions: []res.ACTAction{
			{DelayMS: 25, Animations: make([]res.ACTAnimation, 1)},
			{DelayMS: 25, Animations: make([]res.ACTAnimation, 2)},
		}},
		Billboards: map[scene.SpriteBillboardKey]*scene.Billboard{
			{ActionIndex: 0, Motion: 0}: {Image: render.NewImage(4, 4)},
			{ActionIndex: 1, Motion: 0}: {Image: render.NewImage(5, 5)},
			{ActionIndex: 1, Motion: 1}: {Image: render.NewImage(6, 6)},
		},
	}
	fx := &effects.System{EffectViews: map[string]*scene.SpriteView{"얼음땡": view}}
	var sounds []string
	m := &System{}
	m.Bind(Bindings{Effects: fx, ScheduleSoundAtActor: func(_ time.Time, _ uint32, paths ...string) { sounds = append(sounds, paths...) }})
	m.SyncActorAilments(ctx, 100, world.Actor{}, true)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	body := &scene.Billboard{Image: render.NewImage(40, 90)}
	entry := scene.ActorDrawEntry{IsPlayer: true, WorldX: 10.5, WorldY: 20.5}
	draw := func(at time.Time, width float64) {
		t.Helper()
		frame := render.NewFrame(800, 600)
		entry.Actor = w.Player
		m.drawActorAilments(frame, ctx, projection, entry, body, 1, at)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if width == 0 {
			if draws.Len() != 0 {
				t.Fatal("finished shatter animation was still drawn")
			}
			return
		}
		if draws.Len() != 1 || draws.Index(0).FieldByName("Width").Float() != width {
			t.Fatalf("wrong ice animation frame: %v draws, want width %v", draws.Len(), width)
		}
		if !draws.Index(0).FieldByName("Options").FieldByName("DepthTest").Bool() {
			t.Fatal("ice block must be occluded by walls")
		}
		if w.Player.BodyState != db.BodyStateFreeze {
			center := draws.Index(0).FieldByName("Center")
			for axis, want := range [3]float64{10.5, 0, 20.5} {
				if got := center.Index(axis).Float(); got != want {
					t.Fatalf("ice shatter moved with the actor: center[%d] = %v, want %v", axis, got, want)
				}
			}
		}
	}
	draw(m.Ailments[100].BodyStarted.Add(time.Minute), 4)
	previous := w.Player
	w.Player.BodyState = 0
	m.SyncActorAilments(ctx, 200, previous, true)
	started := m.Ailments[100].IceBreakStarted
	m.SyncActorAilments(ctx, 100, w.Player, true)
	if len(sounds) != 2 || sounds[0] != "_stonecurse.wav" || sounds[1] != "_frozen_explosion.wav" || m.Ailments[100].IceBreakStarted != started {
		t.Fatal("freezing or shattering sound/animation was missing or repeated")
	}
	draw(started, 5)
	w.Player.X, w.Player.Y = 12, 24
	entry.WorldX, entry.WorldY, entry.WorldZ = 12.5, 24.5, 8
	m.SyncActorAilments(ctx, 100, w.Player, true)
	draw(started.Add(time.Second/60), 6)
	draw(started.Add(time.Second), 0)
	if len(m.Ailments) != 0 {
		t.Fatal("completed ice shatter retained presentation state")
	}
}

func TestHealthAilmentSoundsOnlyPlayOnNewBits(t *testing.T) {
	for _, bit := range []uint16{db.HealthStatePoison, db.HealthStateDpoison, db.HealthStateBleeding} {
		w := world.New()
		w.Actors[300] = world.Actor{ID: 300, HasState: true, HealthState: bit}
		ctx := client.Context{World: w}
		var sounds []string
		m := &System{}
		m.Bind(Bindings{Effects: &effects.System{}, ScheduleSoundAtActor: func(_ time.Time, id uint32, paths ...string) {
			if id != 300 {
				t.Fatal("ailment sound lost its actor")
			}
			sounds = append(sounds, paths...)
		}})
		m.SyncActorAilments(ctx, 300, world.Actor{}, false)
		if len(sounds) != 0 {
			t.Fatal("initial actor snapshot replayed a status sound")
		}
		m.SyncActorAilments(ctx, 300, world.Actor{}, true)
		m.SyncActorAilments(ctx, 300, w.Actors[300], true)
		if len(sounds) != 1 || sounds[0] != "_poison.wav" {
			t.Fatal("new ailment did not play exactly one positional sound")
		}
	}
}
