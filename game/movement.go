package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

const (
	walkRequestCooldown    = 225 * time.Millisecond
	walkErrorCooldown      = 500 * time.Millisecond
	turnDirectionCooldown  = 100 * time.Millisecond
	heldWalkRepeatInterval = 600 * time.Millisecond
)

func (m *WorldMode) setWalkCooldown(duration time.Duration) {
	m.walkCooldownUntil = time.Now().Add(duration)
}

func (m *WorldMode) walkReady(now time.Time) bool {
	return (m.walkCooldownUntil.IsZero() || !now.Before(m.walkCooldownUntil)) &&
		(!m.walkAwaitingAck || now.Sub(m.walkRequestAt) >= time.Second)
}

type hoveredWalkCellCache struct {
	valid bool
	key   hoveredWalkCellKey
	x     int
	y     int
	ok    bool
}

type hoveredWalkCellKey struct {
	gat        *res.GAT
	mouseX     int
	mouseY     int
	playerX    int
	playerY    int
	targetX    float64
	targetY    float64
	targetZ    float64
	cameraYaw  float64
	cameraZoom float64
	screenW    float64
	screenH    float64
}

type scheduledActorStop struct {
	id         uint32
	at         time.Time
	resumeWalk bool
	resumeAt   time.Time
}

type scheduledWalkResume struct {
	id  uint32
	at  time.Time
	toX int
	toY int
}

func (m *WorldMode) updateHeldWalk(ctx client.Context, pointerBlocked bool, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MousePressed(input.MouseButtonLeft) || pointerBlocked {
		m.nextHeldWalkAt = time.Time{}
		return false
	}
	if ctx.Input.MouseJustPressed(input.MouseButtonLeft) || m.pendingAttack.targetID != 0 || m.pendingPickup.itemID != 0 || m.queuedWalk != nil || m.skillSystem.Pending.Skill.ID != 0 || m.petSystem.PendingCapture.Active || shouldUseTurnOnlyGroundClick(ctx) {
		return false
	}
	if !m.walkReady(now) || (!m.nextHeldWalkAt.IsZero() && now.Before(m.nextHeldWalkAt)) {
		return false
	}
	m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)

	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	targetX, targetY, ok := maps.ClickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok || playerAtWalkTarget(ctx.World.Player, targetX, targetY, now) {
		return false
	}
	m.requestMove(ctx, targetX, targetY, "held click")
	return true
}

func playerAtWalkTarget(player worldstate.Actor, targetX, targetY int, now time.Time) bool {
	x, y := scene.ActorRenderPosition(player, now)
	return int(math.Round(x)) == targetX && int(math.Round(y)) == targetY
}

func warpWalkTarget(ctx client.Context, actor worldstate.Actor, now time.Time) (int, int, bool) {
	if ctx.World == nil {
		return 0, 0, false
	}
	targetX, targetY := scene.ActorCurrentCell(actor, now)
	if ctx.World.GAT == nil {
		return targetX, targetY, scene.WalkTargetInBounds(ctx, targetX, targetY)
	}

	playerX, playerY := scene.CurrentPlayerCell(ctx, now)
	bestX, bestY := 0, 0
	bestCost := int(^uint(0) >> 1)
	found := false
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			// ROBr pathfinds to within one Euclidean tile of a warp actor.
			if dx*dx+dy*dy > 1 {
				continue
			}
			x, y := targetX+dx, targetY+dy
			if !ctx.World.GAT.InBounds(x, y) || !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			path, ok := maps.FindWalkPath(ctx.World.GAT, playerX, playerY, x, y)
			if !ok {
				continue
			}
			cost := 0
			for i := 1; i < len(path); i++ {
				cost += maps.PathStepCost(
					maps.PathPoint{X: path[i-1].X, Y: path[i-1].Y},
					maps.PathPoint{X: path[i].X, Y: path[i].Y},
				)
			}
			if cost < bestCost {
				bestX, bestY = x, y
				bestCost = cost
				found = true
			}
		}
	}
	return bestX, bestY, found
}

type walkIntent struct {
	x, y    int
	source  string
	chasing bool
}

func (m *WorldMode) playerChasing() bool {
	return m.pendingAttack.targetID != 0 || m.pendingPickup.itemID != 0 || m.skillSystem.PlayerChasing()
}

func (m *WorldMode) cancelPlayerAction(ctx client.Context) {
	m.stopAttack(ctx)
	m.pendingPickup = pickupIntent{}
	m.skillSystem.CancelPlayerChase()
	m.queuedWalk = nil
	m.scheduledResumes = nil
}

func (m *WorldMode) requestMove(ctx client.Context, x, y int, source string) bool {
	if !playerCanWalkTo(ctx, x, y) {
		return false
	}
	m.cancelPlayerAction(ctx)
	return m.requestWalk(ctx, x, y, source)
}

func (m *WorldMode) playerAnimation(ctx client.Context, now time.Time) (scene.ActorAnimation, bool) {
	if ctx.World == nil {
		return scene.ActorAnimation{}, false
	}
	ids := []uint32{ctx.World.Player.ID}
	if ctx.Session != nil {
		ids = append(ids, ctx.Session.AccountID, ctx.Session.CharID)
	}
	for _, id := range ids {
		if anim, ok := m.actorSystem.ActorAnimation(id, now); ok && !anim.AutoCounter && !anim.KeepMovement &&
			(anim.Casting || now.Before(anim.Started.Add(anim.Duration))) {
			return anim, true
		}
	}
	return scene.ActorAnimation{}, false
}

func (m *WorldMode) playerMoveLocked(ctx client.Context, now time.Time) bool {
	anim, ok := m.playerAnimation(ctx, now)
	return ok && (anim.ActionFamily == scene.SpriteActionPickup ||
		anim.ActionFamily == sprites.ActionPCAttack1 || anim.ActionFamily == sprites.ActionPCAttack2 || anim.ActionFamily == sprites.ActionPCAttack3)
}

func (m *WorldMode) playerCanAct(ctx client.Context, now time.Time) bool {
	if scene.PlayerIsDead(ctx) || ctx.World == nil || ctx.World.Player.Sitting {
		return false
	}
	anim, ok := m.playerAnimation(ctx, now)
	return !ok || (!anim.Casting && anim.ActionFamily != sprites.ActionPCSkill && !m.playerMoveLocked(ctx, now))
}

func (m *WorldMode) playerCanUseSkill(ctx client.Context, now time.Time) bool {
	anim, ok := m.playerAnimation(ctx, now)
	return (ok && anim.Combo && !scene.PlayerIsDead(ctx) && !ctx.World.Player.Sitting) || m.playerCanAct(ctx, now)
}

func (m *WorldMode) processQueuedWalk(ctx client.Context, now time.Time) bool {
	if m.queuedWalk == nil || !m.walkReady(now) || m.playerMoveLocked(ctx, now) {
		return true
	}
	walk := *m.queuedWalk
	m.queuedWalk = nil
	if (walk.chasing && !m.playerChasing()) || !playerCanWalkTo(ctx, walk.x, walk.y) {
		return false
	}
	if err := ctx.Network.SendWalkToXY(walk.x, walk.y); err != nil {
		glog.Warnf("%s walk request failed to=%d,%d: %v", walk.source, walk.x, walk.y, err)
		m.setWalkCooldown(walkErrorCooldown)
		return false
	}
	m.walkAwaitingAck = true
	m.walkRequestAt = now
	m.setWalkCooldown(walkRequestCooldown)
	return true
}

func playerCanWalkTo(ctx client.Context, x, y int) bool {
	return ctx.World != nil && ctx.Network != nil && !scene.PlayerIsDead(ctx) &&
		scene.WalkTargetInBounds(ctx, x, y) &&
		(!ctx.PlayerHasEffectState(db.EffectStateHide) || learnedSkillLevel(ctx.Session, db.SkillRGTunneldrive) > 0)
}

func (m *WorldMode) requestWalk(ctx client.Context, targetX, targetY int, source string) bool {
	if !playerCanWalkTo(ctx, targetX, targetY) {
		return false
	}
	m.queuedWalk = &walkIntent{x: targetX, y: targetY, source: source, chasing: m.playerChasing()}
	// Ownership changes immediately, even if an animation or acknowledgement
	// delays the packet. A previous input controller must not stop this walk.
	m.walkSequence++
	return m.processQueuedWalk(ctx, time.Now())
}

// requestWalkStop shortens the current walk to the end of its active path
// segment. Following the server-approved path avoids a visible reversal when
// that path bends around an obstacle.
func (m *WorldMode) requestWalkStop(ctx client.Context, source string) bool {
	now := time.Now()
	if scene.PlayerIsDead(ctx) || ctx.World == nil || ctx.Network == nil {
		return false
	}
	player := ctx.World.Player
	if !scene.ActorIsMovingAt(player, now) {
		return true
	}
	if !m.walkReady(now) {
		return false
	}
	targetX, targetY := nextPlayerPathCell(player, now)
	return m.requestWalk(ctx, targetX, targetY, source+" stop")
}

func nextPlayerPathCell(player worldstate.Actor, now time.Time) (int, int) {
	if len(player.MovePath) >= 2 {
		elapsed := now.Sub(player.MoveStarted)
		_, _, x, y := scene.RenderPathSegmentWithSpeed(
			player.MovePath,
			elapsed,
			scene.ActorWalkSpeed(player),
			player.MoveStartX,
			player.MoveStartY,
			player.HasMoveStart,
		)
		return int(math.Round(x)), int(math.Round(y))
	}

	x, y := scene.ActorRenderPosition(player, now)
	targetX := int(math.Round(x)) + actors.SignInt(player.ToX-int(math.Round(x)))
	targetY := int(math.Round(y)) + actors.SignInt(player.ToY-int(math.Round(y)))
	return targetX, targetY
}

func shouldUseTurnOnlyGroundClick(ctx client.Context) bool {
	if ctx.World == nil || ctx.Input == nil {
		return false
	}
	return ctx.World.Player.Sitting || ctx.Input.Pressed(input.KeyShift)
}

func (m *WorldMode) requestChangeDirection(ctx client.Context, targetX, targetY int, source string) {
	if ctx.Network == nil {
		m.setWalkCooldown(walkErrorCooldown)
		return
	}
	if ctx.World == nil {
		return
	}
	playerX, playerY := scene.CurrentPlayerCell(ctx, time.Now())
	targetDir := actors.DirectionFromDelta(playerX, playerY, targetX, targetY, ctx.World.Player.Dir)
	headDir, bodyDir, ok := resolveTurnOnlyDirection(ctx.World.Player.Dir, int(ctx.World.Player.HeadDir), targetDir)
	if !ok {
		return
	}
	glog.Debugf("%s change direction request player=%d,%d target=%d,%d head_dir=%d dir=%d", source, playerX, playerY, targetX, targetY, headDir, bodyDir)
	if err := ctx.Network.SendChangeDirection(headDir, bodyDir); err != nil {
		glog.Warnf("%s change direction failed target=%d,%d head_dir=%d dir=%d: %v", source, targetX, targetY, headDir, bodyDir, err)
		m.setWalkCooldown(walkErrorCooldown)
		return
	}
	m.applyLocalDirection(ctx, headDir, bodyDir)
	m.setWalkCooldown(turnDirectionCooldown)
}

func resolveTurnOnlyDirection(currentBodyDir int, currentHeadDir int, targetDir int) (uint8, uint8, bool) {
	bodyDir := scene.NormalizeDirectionIndex(currentBodyDir)
	headDir := normalizeHeadDir(currentHeadDir)
	targetDir = scene.NormalizeDirectionIndex(targetDir)
	delta := scene.NormalizeDirectionIndex(bodyDir - targetDir)

	resolvedBodyDir := bodyDir
	resolvedHeadDir := 0
	switch delta {
	case 0, 4:
		resolvedBodyDir = targetDir
	case 1:
		if headDir != 1 {
			resolvedHeadDir = 1
		} else {
			resolvedBodyDir = targetDir
		}
	case 2, 3:
		resolvedBodyDir = scene.NormalizeDirectionIndex(targetDir + 1)
		resolvedHeadDir = 1
	case 7:
		if headDir != 2 {
			resolvedHeadDir = 2
		} else {
			resolvedBodyDir = targetDir
		}
	case 5, 6:
		resolvedBodyDir = scene.NormalizeDirectionIndex(targetDir - 1)
		resolvedHeadDir = 2
	default:
		return 0, 0, false
	}
	return uint8(normalizeHeadDir(resolvedHeadDir)), uint8(resolvedBodyDir), true
}

func normalizeHeadDir(headDir int) int {
	if headDir < 0 {
		return 0
	}
	if headDir > 2 {
		return 2
	}
	return headDir
}

func (m *WorldMode) applyLocalDirection(ctx client.Context, headDir, dir uint8) {
	if ctx.World == nil {
		return
	}
	ctx.World.Player.Dir = int(dir & 7)
	ctx.World.Player.HeadDir = uint8(normalizeHeadDir(int(headDir)))
	ctx.World.Dir = ctx.World.Player.Dir
	if ctx.Session != nil {
		ctx.Session.PlayerDir = ctx.World.Player.Dir
	}
}

func applyActorDirectionChange(ctx client.Context, direction network.ActorDirectionChange) {
	if ctx.World == nil || direction.ID == 0 {
		return
	}
	dir := int(direction.Dir & 7)
	headDir := uint8(normalizeHeadDir(int(direction.HeadDir)))
	if scene.IsLocalActor(ctx, direction.ID) {
		ctx.World.Player.Dir = dir
		ctx.World.Player.HeadDir = headDir
		ctx.World.Dir = dir
		if ctx.Session != nil {
			ctx.Session.PlayerDir = dir
		}
		return
	}
	actor, ok := ctx.World.Actors[direction.ID]
	if !ok {
		return
	}
	actor.Dir = dir
	actor.HeadDir = headDir
	ctx.World.Actors[direction.ID] = actor
}

func applySelfMoveAck(ctx client.Context, ack network.SelfMoveAck) {
	now := time.Now()
	fastForward := time.Duration(0)
	if ctx.Session != nil {
		if elapsed, ok := ctx.Session.ElapsedSinceServerTick(ack.ServerTick, now); ok {
			fastForward = elapsed
		}
	}
	dir := actors.DirectionFromDelta(ack.FromX, ack.FromY, ack.ToX, ack.ToY, ctx.World.Dir)
	setPlayerMovementAt(ctx, ack.FromX, ack.FromY, ack.ToX, ack.ToY, dir, now, fastForward)
	ctx.Session.PlayerX = ack.ToX
	ctx.Session.PlayerY = ack.ToY
}

func setPlayerMovementAt(ctx client.Context, fromX, fromY, toX, toY, dir int, now time.Time, fastForward time.Duration) {
	if ctx.World == nil {
		return
	}
	path := maps.WalkPath(ctx.World.GAT, fromX, fromY, toX, toY)
	if len(path) == 0 {
		fromX, fromY = scene.CurrentPlayerCell(ctx, now)
		path = maps.WalkPath(ctx.World.GAT, fromX, fromY, toX, toY)
		if len(path) == 0 {
			return
		}
	}
	setPlayerMovementPathAt(ctx, path, fromX, fromY, toX, toY, dir, now, fastForward)
}

func setPlayerMovementPathAt(ctx client.Context, path []worldstate.WalkStep, fromX, fromY, toX, toY, dir int, now time.Time, fastForward time.Duration) {
	if ctx.World == nil {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	oldPlayer := ctx.World.Player
	if len(path) == 0 {
		path = actors.DirectMovementPath(fromX, fromY, toX, toY)
	}
	finalDir := movementFinalDirection(path, fromX, fromY, toX, toY, dir)
	speed := scene.ActorMoveSpeed(ctx.World.Player)
	startX, startY, hasMoveStart, offset := actors.MovementStart(path, oldPlayer, now, fastForward, speed, fromX, fromY)
	duration := actors.ActorMovementDurationFromWithSpeed(path, fromX, fromY, toX, toY, speed, startX, startY, hasMoveStart)
	ctx.World.Player.X = toX
	ctx.World.Player.Y = toY
	ctx.World.Player.Dir = finalDir
	ctx.World.Player.HeadDir = 0
	ctx.World.Player.FromX = fromX
	ctx.World.Player.FromY = fromY
	ctx.World.Player.ToX = toX
	ctx.World.Player.ToY = toY
	ctx.World.Player.Moving = true
	ctx.World.Player.Sitting = false
	ctx.World.Player.MoveStarted = now.Add(-offset)
	ctx.World.Player.MovePath = path
	ctx.World.Player.MoveDuration = duration
	ctx.World.Player.MoveSpeed = speed
	ctx.World.Player.MoveStartX = startX
	ctx.World.Player.MoveStartY = startY
	ctx.World.Player.HasMoveStart = hasMoveStart
	ctx.World.Player.WalkDistance = actors.MovementWalkDistanceOffset(path, oldPlayer, now, offset, speed, startX, startY, hasMoveStart)
	ctx.World.Dir = finalDir
}

func (m *WorldMode) scheduleActorStop(id uint32, at time.Time, resumeWalk bool, resumeAt time.Time) {
	if id == 0 || at.IsZero() {
		return
	}
	m.scheduledStops = append(m.scheduledStops, scheduledActorStop{id: id, at: at, resumeWalk: resumeWalk, resumeAt: resumeAt})
}

func (m *WorldMode) processScheduledActorStops(ctx client.Context, now time.Time) {
	if len(m.scheduledStops) == 0 {
		return
	}
	active := m.scheduledStops[:0]
	for _, stop := range m.scheduledStops {
		if now.Before(stop.at) {
			active = append(active, stop)
			continue
		}
		if stop.resumeWalk {
			m.pauseActorMovementForResume(ctx, stop.id, stop.at, stop.resumeAt)
		} else {
			m.stopActorMovementAt(ctx, stop.id, stop.at)
		}
	}
	m.scheduledStops = active
}

func (m *WorldMode) pauseActorMovementForResume(ctx client.Context, id uint32, at time.Time, resumeAt time.Time) {
	if !m.canResumeActorWalk(ctx, id, at) {
		m.stopActorMovementAt(ctx, id, at)
		return
	}
	if ctx.World == nil || !scene.IsLocalActor(ctx, id) {
		m.stopActorMovementAt(ctx, id, at)
		return
	}
	toX := ctx.World.Player.ToX
	toY := ctx.World.Player.ToY
	actors.StopLocalPlayerMovementAt(ctx, at)
	if ctx.World.Player.X == toX && ctx.World.Player.Y == toY {
		return
	}
	m.scheduledResumes = append(m.scheduledResumes, scheduledWalkResume{
		id:  id,
		at:  resumeAt,
		toX: toX,
		toY: toY,
	})
}

func (m *WorldMode) canResumeActorWalk(ctx client.Context, id uint32, at time.Time) bool {
	if ctx.World == nil || !scene.IsLocalActor(ctx, id) || !m.hasCombatFocus() {
		return false
	}
	actor := ctx.World.Player
	if !scene.ActorIsMovingAt(actor, at) || len(actor.MovePath) < 2 {
		return false
	}
	x, y := scene.ActorRenderPosition(actor, at)
	return math.Hypot(float64(actor.ToX)-x, float64(actor.ToY)-y) > 0.25
}

func (m *WorldMode) processScheduledWalkResumes(ctx client.Context, now time.Time) {
	if len(m.scheduledResumes) == 0 {
		return
	}
	active := m.scheduledResumes[:0]
	for _, resume := range m.scheduledResumes {
		if now.Before(resume.at) {
			active = append(active, resume)
			continue
		}
		m.resumeActorWalk(ctx, resume)
	}
	m.scheduledResumes = active
}

func (m *WorldMode) resumeActorWalk(ctx client.Context, resume scheduledWalkResume) {
	if ctx.World == nil || !scene.IsLocalActor(ctx, resume.id) || !m.hasCombatFocus() {
		return
	}
	player := ctx.World.Player
	if player.X == resume.toX && player.Y == resume.toY {
		return
	}
	setPlayerMovementAt(ctx, player.X, player.Y, resume.toX, resume.toY, player.Dir, resume.at, 0)
	m.clearLocalActorAction(ctx)
}

func (m *WorldMode) stopActorMovementAt(ctx client.Context, id uint32, at time.Time) {
	if ctx.World == nil || id == 0 {
		return
	}
	if scene.IsLocalActor(ctx, id) {
		actors.StopLocalPlayerMovementAt(ctx, at)
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok || !scene.ActorIsMovingAt(actor, at) {
		return
	}
	x, y := scene.ActorRenderPosition(actor, at)
	actor.X = int(math.Round(x))
	actor.Y = int(math.Round(y))
	actors.ClearActorMovement(&actor)
	ctx.World.Actors[id] = actor
}

func movementFinalDirection(path []worldstate.WalkStep, fromX, fromY, toX, toY int, fallback int) int {
	if len(path) >= 2 {
		from := path[len(path)-2]
		to := path[len(path)-1]
		return actors.DirectionFromDelta(from.X, from.Y, to.X, to.Y, fallback)
	}
	return actors.DirectionFromDelta(fromX, fromY, toX, toY, fallback)
}

func (m *WorldMode) hoveredWalkCell(ctx client.Context, projection scene.Projection, mouseX, mouseY int) (int, int, bool) {
	if ctx.World == nil || ctx.World.GAT == nil {
		m.hoveredWalk.valid = false
		return 0, 0, false
	}
	key := hoveredWalkCellKey{
		gat:        ctx.World.GAT,
		mouseX:     mouseX,
		mouseY:     mouseY,
		playerX:    ctx.World.Player.X,
		playerY:    ctx.World.Player.Y,
		targetX:    projection.PlayerX,
		targetY:    projection.PlayerY,
		targetZ:    projection.PlayerZ,
		cameraYaw:  projection.CameraYaw,
		cameraZoom: projection.CameraZoom,
		screenW:    projection.ScreenW,
		screenH:    projection.ScreenH,
	}
	if m.hoveredWalk.valid && m.hoveredWalk.key == key {
		return m.hoveredWalk.x, m.hoveredWalk.y, m.hoveredWalk.ok
	}
	x, y, ok := maps.HoveredWalkCell(ctx, projection, mouseX, mouseY)
	m.hoveredWalk = hoveredWalkCellCache{
		valid: true,
		key:   key,
		x:     x,
		y:     y,
		ok:    ok,
	}
	return x, y, ok
}
