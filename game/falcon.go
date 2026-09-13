package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

const (
	actorPersistentEffectOptionMask = actorEffectCartMask | db.EffectStateFalcon | db.EffectStateWedding
	falconGlideHeight               = 5.0
	falconFollowRange               = 2
	falconStopRange                 = 1
	falconRetargetInterval          = time.Second
	falconAttackMoveSpeedMS         = 35
	falconAttackReturnDelay         = 432 * time.Millisecond
	falconAttackOvershoot           = 5
)

type falconRenderState struct {
	x            float64
	y            float64
	direction    int
	path         []worldstate.WalkStep
	moveStarted  time.Time
	moveDuration time.Duration
	moveSpeedMS  int
	moveStartX   float64
	moveStartY   float64
	moving       bool
	hasTarget    bool
	targetX      int
	targetY      int
	lastWalkTick time.Time
	attacking    bool
	returnAt     time.Time
}

func actorHasFalcon(actor worldstate.Actor) bool {
	if actor.EffectState&db.EffectStateFalcon == 0 {
		return false
	}
	return actorJobSupportsFalconSprite(int(actor.Job))
}

func actorJobSupportsFalconSprite(job int) bool {
	switch job {
	case db.JobHunter, db.JobHunterH, db.JobHunterB,
		db.JobRanger, db.JobRanger2, 4098,
		db.JobWindhawk, 4270, 4278:
		return true
	default:
		return false
	}
}

func falconMoveSpeedMS(ownerSpeed int) int {
	if ownerSpeed <= 0 {
		return 100
	}
	speed := ownerSpeed - 50
	if speed < 1 {
		return 1
	}
	return speed
}

func falconSkillAttacksTarget(skillID uint16) bool {
	switch skillID {
	case db.SkillHTBlitzbeat, db.SkillSNFalconassault:
		return true
	default:
		return false
	}
}

func falconSkillAttacksGround(skillID uint16) bool {
	return skillID == db.SkillHTDetecting
}

func (m *WorldMode) applyFalconStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusFalcon || ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = localSkillTarget(ctx)
	}
	actor, ok, local := actorForCombatID(ctx, id)
	if !ok {
		return
	}
	if change.Active {
		actor.EffectState |= db.EffectStateFalcon
	} else {
		actor.EffectState &^= db.EffectStateFalcon
	}
	actor.HasState = true
	if local {
		ctx.World.Player.EffectState = actor.EffectState
		ctx.World.Player.HasState = true
		setSelectedCharacterOptionBit(ctx, db.EffectStateFalcon, change.Active)
		if !change.Active {
			m.removeFalconState(id)
			if ctx.Session != nil {
				m.removeFalconState(ctx.Session.CharID)
			}
		}
		glog.Debugf("actor falcon status local actor=%d active=%t", id, change.Active)
		return
	}
	upsertActor(ctx, actor)
	if !change.Active {
		m.removeFalconState(id)
	}
	glog.Debugf("actor falcon status actor=%d active=%t", id, change.Active)
}

func (m *WorldMode) applyFalconSkillNoDamageNotify(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	if !falconSkillAttacksTarget(notify.SkillID) {
		return
	}
	target, ok, _ := actorForCombatID(ctx, notify.TargetID)
	if !ok {
		return
	}
	targetX, targetY := actorRenderPosition(target, now)
	m.startFalconAttackAt(ctx, notify.SourceID, int(targetX), int(targetY), now)
}

func (m *WorldMode) applyFalconActorActionNotify(ctx client.Context, action network.ActorActionNotify, now time.Time) {
	if !falconSkillAttacksTarget(action.SkillID) {
		return
	}
	target, ok, _ := actorForCombatID(ctx, action.TargetID)
	if !ok {
		return
	}
	targetX, targetY := actorRenderPosition(target, now)
	m.startFalconAttackAt(ctx, action.SourceID, int(targetX), int(targetY), now)
}

func (m *WorldMode) applyFalconSkillCastNotify(ctx client.Context, notify network.SkillCastNotify, now time.Time) {
	if !falconSkillAttacksGround(notify.SkillID) {
		return
	}
	m.startFalconAttackAt(ctx, notify.SourceID, int(notify.X), int(notify.Y), now)
}

func (m *WorldMode) applyFalconGroundSkillNotify(ctx client.Context, notify network.GroundSkillNotify, now time.Time) {
	if !falconSkillAttacksGround(notify.SkillID) {
		return
	}
	m.startFalconAttackAt(ctx, notify.SourceID, int(notify.X), int(notify.Y), now)
}

func (m *WorldMode) startFalconAttackAt(ctx client.Context, sourceID uint32, targetX, targetY int, now time.Time) bool {
	source, ok, local := actorForCombatID(ctx, sourceID)
	if !ok {
		return false
	}
	if local {
		source = actorWithSelectedPersistentEffectOptions(ctx, source)
		source.ID = localFalconOwnerID(ctx)
	}
	if source.ID == 0 || !actorHasFalcon(source) {
		return false
	}
	falcon := m.updateFalconState(source, now)
	if falcon == nil {
		return false
	}
	falcon.startAttack(targetX, targetY, now)
	return true
}

func actorWithSelectedPersistentEffectOptions(ctx client.Context, actor worldstate.Actor) worldstate.Actor {
	if ctx.Session == nil {
		return actor
	}
	character := ctx.Session.SelectedCharacter()
	actor.Job = character.Job
	if actor.HasState {
		actor.EffectState = (actor.EffectState &^ actorPersistentEffectOptionMask) | (character.Option & actorPersistentEffectOptionMask)
	} else {
		actor.EffectState = character.Option
	}
	return actor
}

func localFalconOwnerID(ctx client.Context) uint32 {
	if ctx.Session == nil {
		return 0
	}
	if ctx.Session.CharID != 0 {
		return ctx.Session.CharID
	}
	return ctx.Session.AccountID
}

func setSelectedCharacterOptionBit(ctx client.Context, bit uint32, active bool) {
	if ctx.Session == nil || bit == 0 {
		return
	}
	if active {
		ctx.Session.Selected.Option |= bit
	} else {
		ctx.Session.Selected.Option &^= bit
	}
	id := ctx.Session.Selected.ID
	if id == 0 {
		id = ctx.Session.CharID
	}
	for i := range ctx.Session.Characters {
		if ctx.Session.Characters[i].ID != id {
			continue
		}
		if active {
			ctx.Session.Characters[i].Option |= bit
		} else {
			ctx.Session.Characters[i].Option &^= bit
		}
		return
	}
}

func (m *WorldMode) removeFalconState(ownerID uint32) {
	if m.falcons == nil || ownerID == 0 {
		return
	}
	delete(m.falcons, ownerID)
}

func (m *WorldMode) falconSpriteView(ctx client.Context, job int) *spriteView {
	if ctx.Resources == nil {
		return nil
	}
	if m.falconViews == nil {
		m.falconViews = make(map[int]*spriteView)
	}
	if m.falconViewMiss == nil {
		m.falconViewMiss = make(map[int]struct{})
	}
	if view, ok := m.falconViews[job]; ok {
		return view
	}
	if _, miss := m.falconViewMiss[job]; miss {
		return nil
	}
	view, status := loadFalconSpriteView(ctx.Resources, job)
	if view == nil {
		m.falconViewMiss[job] = struct{}{}
		glog.Warnf("falcon sprite unavailable job=%d: %s", job, status)
		return nil
	}
	m.falconViews[job] = view
	if status != "" {
		glog.Debugf("falcon sprite resources job=%d %s", job, status)
	}
	return view
}

func (m *WorldMode) drawSceneActorFalcons(screen *render.Frame, ctx client.Context, projection sceneProjection, entries []sceneActorDrawEntry) {
	if len(entries) == 0 {
		m.pruneFalconStates(nil)
		return
	}
	now := time.Now()
	activeOwners := make(map[uint32]struct{})
	for _, entry := range entries {
		if actorHasStealth(entry.actor) || !actorHasFalcon(entry.actor) {
			continue
		}
		activeOwners[entry.actor.ID] = struct{}{}
		alpha := m.actorVisualAlpha(entry.actor.ID, now)
		m.drawActorFalcon3D(screen, ctx, projection, entry.actor, projection.cameraYaw, alpha, now)
	}
	m.pruneFalconStates(activeOwners)
}

func (m *WorldMode) pruneFalconStates(activeOwners map[uint32]struct{}) {
	if m.falcons == nil {
		return
	}
	for ownerID := range m.falcons {
		if _, ok := activeOwners[ownerID]; ok {
			continue
		}
		delete(m.falcons, ownerID)
	}
}

func (m *WorldMode) drawActorFalcon3D(screen *render.Frame, ctx client.Context, projection sceneProjection, actor worldstate.Actor, cameraYaw float64, alpha float64, now time.Time) bool {
	if ctx.World == nil || !actorHasFalcon(actor) {
		return false
	}
	view := m.falconSpriteView(ctx, int(actor.Job))
	if view == nil {
		return false
	}
	falcon := m.updateFalconState(actor, now)
	if falcon == nil {
		return false
	}
	state := falcon.spriteState(cameraYaw)
	billboard, ok := singleSpriteBillboardForState(view, state, now)
	if !ok {
		return false
	}
	terrainZ := terrainHeightAt(ctx.World, falcon.x, falcon.y)
	worldX, worldY := cellWorldAnchor(falcon.x, falcon.y)
	worldZ := terrainZ + falconGlideHeight
	scale := actorBillboardScreenScale(projection, worldX, worldY, worldZ)
	shadow := actorShadowFactor(ctx.World, falcon.x, falcon.y)
	drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, shadow, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	return true
}

func (m *WorldMode) updateFalconState(actor worldstate.Actor, now time.Time) *falconRenderState {
	if actor.ID == 0 {
		return nil
	}
	if m.falcons == nil {
		m.falcons = make(map[uint32]*falconRenderState)
	}
	ownerX, ownerY := actorRenderPosition(actor, now)
	state := m.falcons[actor.ID]
	if state == nil {
		targetX, targetY := falconFollowTargetCell(ownerX, ownerY)
		state = &falconRenderState{
			x:           ownerX,
			y:           ownerY,
			direction:   actor.Dir,
			moveSpeedMS: falconMoveSpeedMS(actor.Speed),
			hasTarget:   true,
			targetX:     targetX,
			targetY:     targetY,
		}
		m.falcons[actor.ID] = state
		return state
	}
	state.advance(now)
	if state.attacking {
		if !state.returnAt.IsZero() && !now.Before(state.returnAt) {
			state.attacking = false
			if state.hasTarget {
				state.startPathTo(state.targetX, state.targetY, 0, now)
			}
		}
		return state
	}
	if falconFollowDistance(ownerX, ownerY, state.x, state.y) < falconFollowRange {
		return state
	}
	targetX, targetY := falconFollowTargetCell(ownerX, ownerY)
	targetChanged := !state.hasTarget || state.targetX != targetX || state.targetY != targetY
	if !targetChanged {
		return state
	}
	if !state.lastWalkTick.IsZero() && state.lastWalkTick.Add(falconRetargetInterval).After(now) {
		return state
	}
	state.moveSpeedMS = falconMoveSpeedMS(actor.Speed)
	state.startFollow(targetX, targetY, now)
	return state
}

func (state *falconRenderState) advance(now time.Time) {
	if state == nil || !state.moving || len(state.path) < 2 {
		return
	}
	elapsed := now.Sub(state.moveStarted)
	if state.moveDuration <= 0 || elapsed >= state.moveDuration {
		last := state.path[len(state.path)-1]
		state.x = float64(last.X)
		state.y = float64(last.Y)
		state.moving = false
		state.path = nil
		return
	}
	state.x, state.y = renderPathPositionWithSpeed(state.path, elapsed, state.moveSpeedMS, state.moveStartX, state.moveStartY, true)
	fromX, fromY, toX, toY := renderPathSegmentWithSpeed(state.path, elapsed, state.moveSpeedMS, state.moveStartX, state.moveStartY, true)
	state.direction = directionFromFloatDelta(fromX, fromY, toX, toY, state.direction)
}

func (state *falconRenderState) startFollow(targetX, targetY int, now time.Time) {
	if state == nil {
		return
	}
	state.hasTarget = true
	state.targetX = targetX
	state.targetY = targetY
	state.lastWalkTick = now
	state.startPathTo(targetX, targetY, falconStopRange, now)
}

func (state *falconRenderState) startAttack(targetX, targetY int, now time.Time) {
	if state == nil {
		return
	}
	state.advance(now)
	overX, overY := falconOvershootTarget(state.x, state.y, targetX, targetY)
	state.attacking = true
	state.returnAt = now.Add(falconAttackReturnDelay)
	state.moveSpeedMS = falconAttackMoveSpeedMS
	state.startPathTo(overX, overY, 0, now)
}

func (state *falconRenderState) startPathTo(targetX, targetY int, stopRange int, now time.Time) {
	if state == nil {
		return
	}
	fromX, fromY := falconFollowTargetCell(state.x, state.y)
	path := falconFollowPath(fromX, fromY, targetX, targetY, stopRange)
	if len(path) < 2 {
		state.moving = false
		state.path = nil
		return
	}
	state.path = path
	state.moveStarted = now
	state.moveStartX = state.x
	state.moveStartY = state.y
	state.moveDuration = actorMovementDurationFromWithSpeed(path, fromX, fromY, path[len(path)-1].X, path[len(path)-1].Y, state.moveSpeedMS, state.moveStartX, state.moveStartY, true)
	state.moving = state.moveDuration > 0
	if state.moving {
		state.direction = directionFromDelta(fromX, fromY, path[1].X, path[1].Y, state.direction)
	}
}

func (state *falconRenderState) spriteState(cameraYaw float64) spriteState {
	if state == nil {
		return spriteState{}
	}
	sprite := spriteState{
		actionFamily: spriteActionIdle,
		direction:    state.direction,
		cameraYaw:    cameraYaw,
		loopIdle:     true,
		moving:       state.moving,
		moveSpeedMS:  state.moveSpeedMS,
		loop:         true,
	}
	if state.attacking {
		sprite.actionFamily = spriteActionWalk
	}
	return sprite
}

func falconFollowDistance(ownerX, ownerY, falconX, falconY float64) int {
	return int(math.Floor(math.Hypot(ownerX-falconX, ownerY-falconY)))
}

func falconFollowTargetCell(x, y float64) (int, int) {
	return int(x), int(y)
}

func falconOvershootTarget(fromX, fromY float64, targetX, targetY int) (int, int) {
	overX := targetX
	overY := targetY
	toX := float64(targetX)
	toY := float64(targetY)
	switch {
	case fromX > toX && fromY > toY:
		overX -= falconAttackOvershoot
		overY -= falconAttackOvershoot
	case fromX < toX && fromY > toY:
		overX += falconAttackOvershoot
		overY -= falconAttackOvershoot
	case fromX < toX && fromY < toY:
		overX += falconAttackOvershoot
		overY += falconAttackOvershoot
	case fromX > toX && fromY < toY:
		overX -= falconAttackOvershoot
		overY += falconAttackOvershoot
	case fromY < toY:
		overY += falconAttackOvershoot
	case fromY > toY:
		overY -= falconAttackOvershoot
	case fromX < toX:
		overX += falconAttackOvershoot
	case fromX > toX:
		overX -= falconAttackOvershoot
	}
	return overX, overY
}

func falconFollowPath(fromX, fromY, targetX, targetY, stopRange int) []worldstate.WalkStep {
	path := []worldstate.WalkStep{{X: fromX, Y: fromY}}
	x, y := fromX, fromY
	for steps := 0; steps < 256 && !falconWithinRange(x, y, targetX, targetY, stopRange); steps++ {
		dx := signInt(targetX - x)
		dy := signInt(targetY - y)
		if dx == 0 && dy == 0 {
			break
		}
		x += dx
		y += dy
		path = append(path, worldstate.WalkStep{X: x, Y: y})
	}
	return path
}

func falconWithinRange(x, y, targetX, targetY, stopRange int) bool {
	return int(math.Floor(math.Hypot(float64(targetX-x), float64(targetY-y)))) <= stopRange
}

func signInt(value int) int {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	default:
		return 0
	}
}
