package game

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

const actorEffectCartMask = db.EffectStateCart1 | db.EffectStateCart2 | db.EffectStateCart3 | db.EffectStateCart4 | db.EffectStateCart5

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
	if effectState&actorEffectCartMask == 0 {
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

func actorCartState(actor worldstate.Actor) (bool, int) {
	if actor.HasCartState {
		return actor.HasCart, normalizeCartNumForActor(int(actor.Job), actor.CartNum)
	}
	cartNum, ok := cartNumFromEffectState(actor.EffectState, int(actor.Job))
	return ok, normalizeCartNumForActor(int(actor.Job), cartNum)
}

func setActorPushCartStatus(actor *worldstate.Actor, active bool, cartNum int) {
	if actor == nil {
		return
	}
	actor.HasCart = active
	actor.HasCartState = true
	actor.EffectState &^= actorEffectCartMask
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
	hasCart, _ := actorCartState(actor)
	if !hasCart || !res.HasPlayerJobToken(int(actor.Job)) {
		return false
	}
	dir := cartSpriteDirection(actor, cameraYaw)
	return dir > 2 && dir < 6
}

func cartSpriteDirection(actor worldstate.Actor, cameraYaw float64) int {
	return spriteDirectionFromWorldDirForCamera(actor.Dir, cameraYaw)
}

func cartSpriteOffset(direction int) (float64, float64) {
	switch normalizeDirectionIndex(direction) {
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

func cartOffsetBillboard(billboard *spriteBillboard, dx, dy float64) *spriteBillboard {
	if billboard == nil {
		return nil
	}
	out := *billboard
	out.anchorX -= dx
	out.anchorY -= dy
	return &out
}

func (m *WorldMode) applyPushCartStatus(ctx client.Context, change network.StatusEffectChange) bool {
	if change.StatusID != db.StatusOnPushCart || ctx.World == nil {
		return false
	}
	cartNum := 1
	if change.HasValues {
		cartNum = int(change.Values[0])
	}
	if change.ActorID == 0 || isLocalActor(ctx, change.ActorID) {
		setActorPushCartStatus(&ctx.World.Player, change.Active, cartNum)
		refreshLocalPlayerMoveSpeed(ctx)
		glog.Debugf("actor cart status local actor=%d active=%t cart=%d", change.ActorID, change.Active, ctx.World.Player.CartNum)
		return true
	}
	actor, ok := ctx.World.Actors[change.ActorID]
	if !ok {
		return true
	}
	setActorPushCartStatus(&actor, change.Active, cartNum)
	upsertActor(ctx, actor)
	glog.Debugf("actor cart status actor=%d active=%t cart=%d", change.ActorID, change.Active, actor.CartNum)
	return true
}

func (m *WorldMode) cartSpriteView(ctx client.Context, cartNum int) *spriteView {
	if ctx.Resources == nil {
		return nil
	}
	if cartNum < 0 {
		cartNum = 0
	}
	if cartNum > 13 {
		cartNum = 13
	}
	if m.cartViews == nil {
		m.cartViews = make(map[int]*spriteView)
	}
	if m.cartViewMiss == nil {
		m.cartViewMiss = make(map[int]struct{})
	}
	if view, ok := m.cartViews[cartNum]; ok {
		return view
	}
	if _, miss := m.cartViewMiss[cartNum]; miss {
		return nil
	}
	view, status := loadCartSpriteView(ctx.Resources, cartNum)
	if view == nil {
		m.cartViewMiss[cartNum] = struct{}{}
		glog.Warnf("cart sprite unavailable cart=%d: %s", cartNum, status)
		return nil
	}
	m.cartViews[cartNum] = view
	if status != "" {
		glog.Debugf("cart sprite resources cart=%d %s", cartNum, status)
	}
	return view
}

func (m *WorldMode) drawActorCart3D(screen *render.Frame, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry, cameraYaw float64, shadow float64, alpha float64) bool {
	actor := entry.actor
	if !res.HasPlayerJobToken(int(actor.Job)) {
		return false
	}
	hasCart, cartNum := actorCartState(actor)
	if !hasCart {
		return false
	}
	view := m.cartSpriteView(ctx, cartNum)
	if view == nil {
		return false
	}
	now := time.Now()
	state := spriteState{
		actionFamily:   spriteActionIdle,
		direction:      actor.Dir,
		cameraYaw:      cameraYaw,
		moving:         actorIsMovingAt(actor, now),
		moveSpeedMS:    actor.Speed,
		hasPlay:        true,
		play:           false,
		fixedMotion:    0,
		hasFixedMotion: true,
	}
	if state.moving {
		state.actionFamily = spriteActionWalk
		state.loop = true
		state.hasPlay = false
		state.hasFixedMotion = false
		state.walkDistance = actorRenderWalkDistance(actor, now)
	}
	billboard, ok := singleSpriteBillboardForState(view, state, now)
	if !ok {
		return false
	}
	dx, dy := cartSpriteOffset(cartSpriteDirection(actor, cameraYaw))
	billboard = cartOffsetBillboard(billboard, dx, dy)
	drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, entry.scale, alpha, shadow, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	return true
}
