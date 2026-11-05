package actors

import (
	"image/color"
	"math"
	"reflect"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestApplyLocalActorLookChangeUpdatesSelectedCharacter(t *testing.T) {
	sessionState := &session.Session{
		AccountID: 100,
		CharID:    200,
		Selected:  session.Character{ID: 200, Job: 0, Hair: 1},
		Characters: []session.Character{
			{ID: 200, Job: 0, Hair: 1},
		},
	}
	ctx := client.Context{
		Session: sessionState,
		World:   worldstate.New(),
	}
	look := network.ActorLookChange{
		ID:    200,
		Type:  2,
		Value: uint32(2101)<<16 | 1201,
	}

	if !ApplyActorLookChange(ctx, look) {
		t.Fatal("local look change should request player sprite reload")
	}
	if sessionState.Selected.Weapon != 1201 || sessionState.Selected.Shield != 2101 {
		t.Fatalf("selected appearance = weapon %d shield %d", sessionState.Selected.Weapon, sessionState.Selected.Shield)
	}
	if sessionState.Characters[0].Weapon != 1201 || sessionState.Characters[0].Shield != 2101 {
		t.Fatalf("character appearance = weapon %d shield %d", sessionState.Characters[0].Weapon, sessionState.Characters[0].Shield)
	}
	if ctx.World.Player.Weapon != 1201 || ctx.World.Player.Shield != 2101 {
		t.Fatalf("world player appearance = weapon %d shield %d", ctx.World.Player.Weapon, ctx.World.Player.Shield)
	}
}

func TestApplyActorJumpPositionMovesLocalPlayer(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4, Moving: true}
	sessionState := &session.Session{AccountID: 2000000, PlayerX: 10, PlayerY: 20}
	ctx := client.Context{Session: sessionState, World: world}

	ApplyActorJumpPosition(ctx, network.ActorJumpPosition{ID: 2000000, X: 7, Y: 20})

	if world.Player.X != 7 || world.Player.Y != 20 || world.Player.Moving {
		t.Fatalf("player after jump = %+v, want stopped at 7,20", world.Player)
	}
	if sessionState.PlayerX != 7 || sessionState.PlayerY != 20 {
		t.Fatalf("session position = %d,%d, want 7,20", sessionState.PlayerX, sessionState.PlayerY)
	}
}

func TestApplyMapAcceptEnterMarksAdminPlayer(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000, AdminList: []uint32{2000000}}
	ctx := client.Context{Session: sessionState, World: world}

	ApplyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 10, Y: 20, Dir: 4})

	if !world.Player.IsAdmin {
		t.Fatalf("world player admin state = false")
	}
}

func TestActorCartStateFromEffectUsesReferenceCartNumbers(t *testing.T) {
	actor := worldstate.Actor{Job: 5, EffectState: db.EffectStateCart3}
	hasCart, cartNum := ActorCartState(actor)
	if !hasCart || cartNum != 3 {
		t.Fatalf("cart from effect = %t, %d", hasCart, cartNum)
	}
	actor = worldstate.Actor{Job: 23, EffectState: db.EffectStateCart5}
	hasCart, cartNum = ActorCartState(actor)
	if !hasCart || cartNum != 0 {
		t.Fatalf("super novice cart from effect = %t, %d", hasCart, cartNum)
	}
}

func TestActorHasFalconUsesReferenceEffectBitAndJobs(t *testing.T) {
	if !actorHasFalcon(worldstate.Actor{Job: db.JobHunter, EffectState: db.EffectStateFalcon}) {
		t.Fatal("hunter falcon bit should create a falcon")
	}
	if !actorHasFalcon(worldstate.Actor{Job: db.JobHunterH, EffectState: db.EffectStateFalcon}) {
		t.Fatal("sniper falcon bit should create a falcon")
	}
	if actorHasFalcon(worldstate.Actor{Job: db.JobKnight, EffectState: db.EffectStateFalcon}) {
		t.Fatal("non-hunter falcon bit should not create a falcon sprite")
	}
	if actorHasFalcon(worldstate.Actor{Job: db.JobHunter}) {
		t.Fatal("hunter without falcon bit should not create a falcon")
	}
}

func TestFalconFollowUsesRobrowserRangeFloor(t *testing.T) {
	if got := falconFollowDistance(11.9, 20, 10, 20); got != 1 {
		t.Fatalf("falcon follow distance = %d, want floored euclidean distance", got)
	}
	path := falconFollowPath(10, 20, 12, 21, falconStopRange)
	end := path[len(path)-1]
	if end.X != 11 || end.Y != 21 {
		t.Fatalf("diagonal-ish falcon endpoint = %d,%d, want floor(distance) <= 1 stop", end.X, end.Y)
	}
}

func TestFalconSpriteStateUsesIdleGlideAction(t *testing.T) {
	falcon := &FalconRenderState{Direction: 3, Moving: true}
	state := falcon.spriteState(45)
	if state.ActionFamily != scene.SpriteActionIdle || !state.LoopIdle {
		t.Fatalf("falcon sprite state = %+v, want looping idle glide action", state)
	}
	if state.ActionFamily == scene.SpriteActionWalk {
		t.Fatal("falcon follow should not use character walk action")
	}

	falcon.Attacking = true
	state = falcon.spriteState(45)
	if state.ActionFamily != scene.SpriteActionWalk {
		t.Fatalf("falcon attack sprite state = %+v, want walk action like robr", state)
	}
}

func TestCartOffsetBillboardAppliesReferencePixelOffset(t *testing.T) {
	base := &scene.Billboard{AnchorX: 100, AnchorY: 200}
	dx, dy := cartSpriteOffset(2)
	got := CartOffsetBillboard(base, dx, dy)
	if got == base {
		t.Fatal("cart offset should copy billboard")
	}
	if got.AnchorX != 60 || got.AnchorY != 200 {
		t.Fatalf("offset billboard anchor = %.0f, %.0f", got.AnchorX, got.AnchorY)
	}
	if base.AnchorX != 100 || base.AnchorY != 200 {
		t.Fatalf("base billboard mutated = %.0f, %.0f", base.AnchorX, base.AnchorY)
	}
}

func TestCartDirectionOffsetBillboardPlacesShadowAtCart(t *testing.T) {
	base := &scene.Billboard{AnchorX: 100, AnchorY: 200}
	got := cartDirectionOffsetBillboard(base, 5)
	if got == base {
		t.Fatal("cart shadow offset should copy billboard")
	}
	if got.AnchorX != 130 || got.AnchorY != 190 {
		t.Fatalf("cart shadow anchor = %.0f, %.0f, want 130, 190", got.AnchorX, got.AnchorY)
	}
	if base.AnchorX != 100 || base.AnchorY != 200 {
		t.Fatalf("base shadow billboard mutated = %.0f, %.0f", base.AnchorX, base.AnchorY)
	}
}

func TestActorStateTintMatchesReferenceBodyAndHealthTints(t *testing.T) {
	tint := ActorStateTint(worldstate.Actor{
		BodyState:   db.BodyStateFreeze,
		HealthState: db.HealthStateBlind,
		HasState:    true,
	})
	if tint.R != 0 || tint.G != 128 || tint.B != 255 || tint.A != 255 {
		t.Fatalf("tint = %+v, want frozen blue without a blind sprite tint", tint)
	}
}

func TestHealthAilmentsUseOriginalStatusColors(t *testing.T) {
	for _, tc := range []struct {
		health uint16
		want   color.RGBA
	}{
		{db.HealthStatePoison, color.RGBA{0, 192, 64, 255}},
		{db.HealthStateDpoison, color.RGBA{0, 192, 64, 255}},
		{db.HealthStateBleeding, color.RGBA{200, 32, 0, 255}},
		{db.HealthStatePoison | db.HealthStateBleeding, color.RGBA{200, 32, 0, 255}},
		{db.HealthStateCurse, color.RGBA{200, 50, 50, 255}},
		{db.HealthStatePoison | db.HealthStateCurse, color.RGBA{200, 50, 50, 255}},
		{db.HealthStatePoison | db.HealthStateBleeding | db.HealthStateCurse, color.RGBA{200, 50, 50, 255}},
	} {
		if got := ActorStateTint(worldstate.Actor{BodyState: db.BodyStateFreeze, HealthState: tc.health, Opt3State: db.Opt3Energycoat}); got != tc.want {
			t.Fatalf("status 0x%04x color = %v, want %v", tc.health, got, tc.want)
		}
	}
}

func TestPetrificationUsesOriginalGreys(t *testing.T) {
	for _, tc := range []struct {
		body uint16
		grey uint8
	}{
		{db.BodyStateStonewait, 128}, {db.BodyStateStone, 64},
	} {
		want := color.RGBA{tc.grey, tc.grey, tc.grey, 255}
		if got := ActorStateTint(worldstate.Actor{BodyState: tc.body, Opt3State: db.Opt3Quicken}); got != want {
			t.Fatalf("petrification color = %v, want %v", got, want)
		}
	}
}

func TestVisibleStatusIconIDsAreKnownAndSorted(t *testing.T) {
	active := map[uint16]session.StatusEffect{
		99:           {ID: 99},
		12:           {ID: 12},
		10:           {ID: 10},
		db.StatusSit: {ID: db.StatusSit},
	}
	ids := gameui.VisibleStatusIconIDs(active)
	if !reflect.DeepEqual(ids, []uint16{10, 12}) {
		t.Fatalf("ids = %+v", ids)
	}
}

func TestApplyRemoteActorLookChangeUpdatesWorldActor(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 0, Head: 1, Appearance: true})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	if ApplyActorLookChange(ctx, network.ActorLookChange{ID: 300, Type: 4, Value: 7}) {
		t.Fatal("remote look change should not request local player sprite reload")
	}
	actor := world.Actors[300]
	if actor.HeadTop != 7 {
		t.Fatalf("remote head top = %d, want 7", actor.HeadTop)
	}
}

func TestDirectionFromDeltaUsesRathenaDirectionOrder(t *testing.T) {
	cases := []struct {
		name string
		toX  int
		toY  int
		want int
	}{
		{name: "north", toX: 10, toY: 11, want: 0},
		{name: "northwest", toX: 9, toY: 11, want: 1},
		{name: "west", toX: 9, toY: 10, want: 2},
		{name: "southwest", toX: 9, toY: 9, want: 3},
		{name: "south", toX: 10, toY: 9, want: 4},
		{name: "southeast", toX: 11, toY: 9, want: 5},
		{name: "east", toX: 11, toY: 10, want: 6},
		{name: "northeast", toX: 11, toY: 11, want: 7},
		{name: "long mostly north", toX: 11, toY: 20, want: 0},
		{name: "long mostly west", toX: 0, toY: 11, want: 2},
		{name: "long mostly south", toX: 11, toY: 0, want: 4},
		{name: "long mostly east", toX: 20, toY: 11, want: 6},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := DirectionFromDelta(10, 10, tt.toX, tt.toY, 6); got != tt.want {
				t.Fatalf("directionFromDelta = %d, want %d", got, tt.want)
			}
		})
	}
	if got := DirectionFromDelta(10, 10, 10, 10, -1); got != 7 {
		t.Fatalf("stationary fallback = %d, want 7", got)
	}
}

func TestActorBillboardScreenScaleUsesProjectedReferenceHeight(t *testing.T) {
	if scene.ActorBillboardWorldHeightUnit != 5 {
		t.Fatalf("actor billboard world height = %.1f, want 5.0", scene.ActorBillboardWorldHeightUnit)
	}

	projection := testutil.Projection(800, 600, 10.5, 20.5, 5)

	scale := scene.ActorBillboardScreenScale(projection, 10.5, 20.5, 5)
	if math.Abs(scale-1.04) > 0.01 {
		t.Fatalf("camera billboard scale = %.3f, want about 1.04 at reference client default zoom", scale)
	}
}

func TestActorBillboardScreenScaleDoesNotChangeWithCameraPitch(t *testing.T) {
	const (
		width  = 800
		height = 600
		x      = 10.5
		y      = 20.5
		z      = 5
	)
	want := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, scene.CameraPitch(), 125), x, y, z)
	for _, pitch := range []float64{scene.DefaultCameraMinPitch, 215, 235, scene.DefaultCameraMaxPitch} {
		projection := scene.NewProjection(width, height, x, y, z, 0, pitch, 125)
		if got := scene.ActorBillboardScreenScale(projection, x, y, z); math.Abs(got-want) > 0.000001 {
			t.Fatalf("actor scale at pitch %.1f = %.6f, want %.6f", pitch, got, want)
		}
	}
}

func TestActorBillboardScreenScaleFollowsZoomAndPerspectiveDepth(t *testing.T) {
	const (
		width  = 800
		height = 600
		x      = 10.5
		y      = 20.5
		z      = 5
	)
	zoomedIn := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, 235, scene.DefaultCameraMinZoom), x, y, z)
	zoomedOut := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, 235, scene.DefaultCameraMaxZoom), x, y, z)
	if zoomedIn <= zoomedOut {
		t.Fatalf("actor scale did not follow zoom: zoomed in %.6f, zoomed out %.6f", zoomedIn, zoomedOut)
	}

	nearer := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, 235, 125), x, y-5, z)
	farther := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, 235, 125), x, y+5, z)
	if nearer <= farther {
		t.Fatalf("actor scale did not follow perspective depth: nearer %.6f, farther %.6f", nearer, farther)
	}
}

func TestActorAnchorOutsideViewportKeepsBodyVisibleBelowScreen(t *testing.T) {
	if actorAnchorOutsideViewport(400, 600+150, 800, 600, 1) {
		t.Fatal("actor should remain visible while its body can still overlap the bottom edge")
	}
	if !actorAnchorOutsideViewport(400, 600+260, 800, 600, 1) {
		t.Fatal("actor should be culled after the whole billboard is beyond the bottom edge")
	}
}

func TestActorViewportCullMarginsScaleWithZoom(t *testing.T) {
	_, _, _, bottom1 := actorViewportCullMargins(1)
	_, _, _, bottom2 := actorViewportCullMargins(2)
	if bottom2 <= bottom1 {
		t.Fatalf("bottom margin did not scale: %.1f <= %.1f", bottom2, bottom1)
	}
	left, right, top, bottom := actorViewportCullMargins(0)
	if left <= 0 || right <= 0 || top <= 0 || bottom <= 0 {
		t.Fatalf("invalid fallback margins: %.1f %.1f %.1f %.1f", left, right, top, bottom)
	}
}

func TestWalkingSpriteStateIgnoresReadyFightAnimation(t *testing.T) {
	state := sprites.State{
		ActionFamily: scene.SpriteActionWalk,
		Moving:       true,
		Loop:         true,
	}

	if ApplyActorAnimationToSpriteState(&state, scene.ActorAnimation{ActionFamily: sprites.ActionPCReadyFight, Loop: true}, true) {
		t.Fatal("ready fight should not override walking state")
	}
	if state.ActionFamily != scene.SpriteActionWalk || !state.Moving {
		t.Fatalf("state = %+v, want still walking", state)
	}

	if !ApplyActorAnimationToSpriteState(&state, scene.ActorAnimation{ActionFamily: sprites.ActionPCDeath}, true) {
		t.Fatal("death should override walking state")
	}
	if state.ActionFamily != sprites.ActionPCDeath || state.Moving {
		t.Fatalf("death state = %+v", state)
	}

	nonPC := sprites.State{ActionFamily: scene.SpriteActionWalk, Moving: true, Loop: true}
	if !ApplyActorAnimationToSpriteState(&nonPC, scene.ActorAnimation{ActionFamily: sprites.ActionNonPCDeath}, false) {
		t.Fatal("non-pc death should override walking state")
	}
	if nonPC.ActionFamily != sprites.ActionNonPCDeath || nonPC.Moving {
		t.Fatalf("non-pc death state = %+v", nonPC)
	}
}
