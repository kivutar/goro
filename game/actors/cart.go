package actors

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

const ActorEffectCartMask = db.EffectStateCart1 | db.EffectStateCart2 | db.EffectStateCart3 | db.EffectStateCart4 | db.EffectStateCart5

func applyActorCartStateFromEffect(actor *worldstate.Actor) {
	if actor == nil {
		return
	}
	cartNum, ok := cartNumFromEffectState(actor.EffectState, int(actor.Job))
	actor.HasCartState = true
	if !ok {
		actor.HasCart = false
		actor.CartNum = 0
		return
	}
	actor.HasCart = true
	actor.CartNum = cartNum
}

func cartNumFromEffectState(effectState uint32, job int) (int, bool) {
	if effectState&ActorEffectCartMask == 0 {
		return 0, false
	}
	if playerJobUsesNoviceCart(job) {
		return 0, true
	}
	switch {
	case effectState&db.EffectStateCart5 != 0:
		return 5, true
	case effectState&db.EffectStateCart4 != 0:
		return 4, true
	case effectState&db.EffectStateCart3 != 0:
		return 3, true
	case effectState&db.EffectStateCart2 != 0:
		return 2, true
	default:
		return 1, true
	}
}

func ActorCartState(actor worldstate.Actor) (bool, int) {
	if actor.HasCartState {
		return actor.HasCart, normalizeCartNumForActor(int(actor.Job), actor.CartNum)
	}
	cartNum, ok := cartNumFromEffectState(actor.EffectState, int(actor.Job))
	return ok, normalizeCartNumForActor(int(actor.Job), cartNum)
}

func SetActorPushCartStatus(actor *worldstate.Actor, active bool, cartNum int) {
	if actor == nil {
		return
	}
	actor.HasCart = active
	actor.HasCartState = true
	actor.EffectState &^= ActorEffectCartMask
	if !active {
		return
	}
	actor.CartNum = normalizeCartNumForActor(int(actor.Job), cartNum)
	actor.EffectState |= effectStateForCartNum(actor.CartNum)
}

func normalizeCartNumForActor(job int, cartNum int) int {
	if playerJobUsesNoviceCart(job) {
		return 0
	}
	if cartNum <= 0 {
		return 1
	}
	if cartNum >= 13 {
		return 13
	}
	return cartNum
}

func playerJobUsesNoviceCart(job int) bool {
	return job == 0 || job == 23 || job == 4001
}

func effectStateForCartNum(cartNum int) uint32 {
	switch cartNum {
	case 2:
		return db.EffectStateCart2
	case 3:
		return db.EffectStateCart3
	case 4:
		return db.EffectStateCart4
	case 5:
		return db.EffectStateCart5
	default:
		return db.EffectStateCart1
	}
}

func cartDrawAfterActor(actor worldstate.Actor, cameraYaw float64) bool {
	hasCart, _ := ActorCartState(actor)
	if !hasCart || !res.HasPlayerJobToken(int(actor.Job)) {
		return false
	}
	dir := cartSpriteDirection(actor, cameraYaw)
	return dir > 2 && dir < 6
}

func cartSpriteDirection(actor worldstate.Actor, cameraYaw float64) int {
	return sprites.DirectionForCamera(actor.Dir, cameraYaw)
}

func cartSpriteOffset(direction int) (float64, float64) {
	switch scene.NormalizeDirectionIndex(direction) {
	case 0:
		return 0, -30
	case 1:
		return 30, -10
	case 2:
		return 40, 0
	case 3:
		return 30, 10
	case 4:
		return 0, 20
	case 5:
		return -30, 10
	case 6:
		return -40, 0
	default:
		return -30, -10
	}
}

func CartOffsetBillboard(billboard *scene.Billboard, dx, dy float64) *scene.Billboard {
	if billboard == nil {
		return nil
	}
	out := *billboard
	out.AnchorX -= dx
	out.AnchorY -= dy
	return &out
}

func cartDirectionOffsetBillboard(billboard *scene.Billboard, direction int) *scene.Billboard {
	dx, dy := cartSpriteOffset(direction)
	return CartOffsetBillboard(billboard, dx, dy)
}

func (m *System) cartSpriteView(ctx client.Context, cartNum int) *scene.SpriteView {
	if ctx.Resources == nil {
		return nil
	}
	if cartNum < 0 {
		cartNum = 0
	}
	if cartNum > 13 {
		cartNum = 13
	}
	if m.CartViews == nil {
		m.CartViews = make(map[int]*scene.SpriteView)
	}
	if m.CartViewMiss == nil {
		m.CartViewMiss = make(map[int]struct{})
	}
	if view, ok := m.CartViews[cartNum]; ok {
		return view
	}
	if _, miss := m.CartViewMiss[cartNum]; miss {
		return nil
	}
	view, status := sprites.LoadCart(ctx.Resources, cartNum)
	if view == nil {
		m.CartViewMiss[cartNum] = struct{}{}
		glog.Warnf("cart sprite unavailable cart=%d: %s", cartNum, status)
		return nil
	}
	m.CartViews[cartNum] = view
	if status != "" {
		glog.Debugf("cart sprite resources cart=%d %s", cartNum, status)
	}
	return view
}

func (m *System) DrawActorCart3D(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry, cameraYaw float64, shadow scene.LightTint, alpha float64) bool {
	actor := entry.Actor
	if scene.ActorHasStealth(actor) || !res.HasPlayerJobToken(int(actor.Job)) {
		return false
	}
	hasCart, cartNum := ActorCartState(actor)
	if !hasCart {
		return false
	}
	view := m.cartSpriteView(ctx, cartNum)
	if view == nil {
		return false
	}
	now := time.Now()
	state := sprites.State{
		ActionFamily:   scene.SpriteActionIdle,
		Direction:      actor.Dir,
		CameraYaw:      cameraYaw,
		Moving:         scene.ActorIsMovingAt(actor, now),
		MoveSpeedMS:    actor.Speed,
		HasPlay:        true,
		Play:           false,
		FixedMotion:    0,
		HasFixedMotion: true,
	}
	if state.Moving {
		state.ActionFamily = scene.SpriteActionWalk
		state.Loop = true
		state.HasPlay = false
		state.HasFixedMotion = false
		state.WalkDistance = actorRenderWalkDistance(actor, now)
	}
	billboard, ok := sprites.BillboardForState(view, state, now)
	if !ok {
		return false
	}
	billboard = cartDirectionOffsetBillboard(billboard, cartSpriteDirection(actor, cameraYaw))
	m.DrawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.WorldX, entry.WorldY, entry.WorldZ, entry.Scale, alpha, shadow, color.RGBA{R: 255, G: 255, B: 255, A: 255}, render.BlendSourceOver)
	return true
}

func (m *System) DrawActorCartShadow3D(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry, cameraYaw float64, now time.Time) bool {
	actor := entry.Actor
	if !entry.CastShadow || scene.ActorHasStealth(actor) || m.ShadowView == nil || m.ShadowViewMiss || !res.HasPlayerJobToken(int(actor.Job)) {
		return false
	}
	if hasCart, _ := ActorCartState(actor); !hasCart {
		return false
	}
	billboard, ok := sprites.FixedBillboard(m.ShadowView)
	if !ok {
		return false
	}
	billboard = cartDirectionOffsetBillboard(billboard, cartSpriteDirection(actor, cameraYaw))
	scale := m.actorShadowRenderScale(ctx, entry, now)
	if scale == 0 {
		return false
	}
	return sprites.DrawShadow3D(screen, projection, billboard, entry.WorldX, entry.WorldY, entry.WorldZ+scene.ActorShadowTerrainLift, scale, m.actorVisualAlpha(actor.ID, now), entry.Shadow)
}
