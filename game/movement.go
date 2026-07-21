package game

import (
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

const (
	walkRequestCooldown    = 200 * time.Millisecond
	walkErrorCooldown      = 500 * time.Millisecond
	turnDirectionCooldown  = 100 * time.Millisecond
	heldWalkRepeatInterval = 500 * time.Millisecond
)

func (m *WorldMode) setWalkCooldown(duration time.Duration) {
	m.walkCooldownUntil = time.Now().Add(duration)
}

func (m *WorldMode) walkReady(now time.Time) bool {
	return m.walkCooldownUntil.IsZero() || !now.Before(m.walkCooldownUntil)
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
	if ctx.Input.MouseJustPressed(input.MouseButtonLeft) || m.pendingSkill.skill.ID != 0 || m.pendingPetCapture.active || shouldUseTurnOnlyGroundClick(ctx) {
		return false
	}
	if !m.walkReady(now) || (!m.nextHeldWalkAt.IsZero() && now.Before(m.nextHeldWalkAt)) {
		return false
	}
	m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)

	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	targetX, targetY, ok := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok || playerAtWalkTarget(ctx.World.Player, targetX, targetY, now) {
		return false
	}
	m.clearLockedAttack()
	m.clearAttackFocus()
	m.requestWalk(ctx, targetX, targetY, "held click")
	return true
}

func playerAtWalkTarget(player worldstate.Actor, targetX, targetY int, now time.Time) bool {
	x, y := actorRenderPosition(player, now)
	return int(math.Round(x)) == targetX && int(math.Round(y)) == targetY
}

func currentPlayerCell(ctx client.Context, now time.Time) (int, int) {
	if ctx.World == nil {
		return 0, 0
	}
	x, y := actorRenderPosition(ctx.World.Player, now)
	return int(math.Round(x)), int(math.Round(y))
}

func (m *WorldMode) requestWalk(ctx client.Context, targetX, targetY int, source string) {
	if !walkTargetInBounds(ctx, targetX, targetY) {
		m.setWalkCooldown(walkRequestCooldown)
		return
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	glog.Debugf("%s walk request from=%d,%d to=%d,%d", source, playerX, playerY, targetX, targetY)
	if err := ctx.Network.SendWalkToXY(targetX, targetY); err == nil {
		m.setWalkCooldown(walkRequestCooldown)
	} else {
		glog.Warnf("%s walk request failed from=%d,%d to=%d,%d: %v", source, playerX, playerY, targetX, targetY, err)
		m.setWalkCooldown(walkErrorCooldown)
	}
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
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	targetDir := directionFromDelta(playerX, playerY, targetX, targetY, ctx.World.Player.Dir)
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
	bodyDir := normalizeDirectionIndex(currentBodyDir)
	headDir := normalizeHeadDir(currentHeadDir)
	targetDir = normalizeDirectionIndex(targetDir)
	delta := normalizeDirectionIndex(bodyDir - targetDir)

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
		resolvedBodyDir = normalizeDirectionIndex(targetDir + 1)
		resolvedHeadDir = 1
	case 7:
		if headDir != 2 {
			resolvedHeadDir = 2
		} else {
			resolvedBodyDir = targetDir
		}
	case 5, 6:
		resolvedBodyDir = normalizeDirectionIndex(targetDir - 1)
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
	if isLocalActor(ctx, direction.ID) {
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
	dir := directionFromDelta(ack.FromX, ack.FromY, ack.ToX, ack.ToY, ctx.World.Dir)
	setPlayerMovementAt(ctx, ack.FromX, ack.FromY, ack.ToX, ack.ToY, dir, now, fastForward)
	ctx.Session.PlayerX = ack.ToX
	ctx.Session.PlayerY = ack.ToY
}

func setPlayerMovementAt(ctx client.Context, fromX, fromY, toX, toY, dir int, now time.Time, fastForward time.Duration) {
	if ctx.World == nil {
		return
	}
	path := walkPath(ctx.World.GAT, fromX, fromY, toX, toY)
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
		path = directMovementPath(fromX, fromY, toX, toY)
	}
	finalDir := movementFinalDirection(path, fromX, fromY, toX, toY, dir)
	speed := actorMoveSpeed(ctx.World.Player)
	startX, startY, hasMoveStart, offset := movementStart(path, oldPlayer, now, fastForward, speed, fromX, fromY)
	duration := actorMovementDurationFromWithSpeed(path, fromX, fromY, toX, toY, speed, startX, startY, hasMoveStart)
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
	ctx.World.Player.MoveStartX = startX
	ctx.World.Player.MoveStartY = startY
	ctx.World.Player.HasMoveStart = hasMoveStart
	ctx.World.Player.WalkDistance = movementWalkDistanceOffset(path, oldPlayer, now, offset, speed, startX, startY, hasMoveStart)
	ctx.World.Dir = finalDir
}

func upsertActor(ctx client.Context, actor worldstate.Actor) {
	upsertActorAt(ctx, actor, time.Now())
}

func upsertActorAt(ctx client.Context, actor worldstate.Actor, now time.Time) {
	if ctx.World == nil {
		return
	}
	existing, hasExisting := ctx.World.Actors[actor.ID]
	if actor.Moving {
		if actor.FromX == 0 && actor.FromY == 0 {
			if hasExisting {
				actor.FromX = existing.X
				actor.FromY = existing.Y
			}
		}
		if actor.ToX == 0 && actor.ToY == 0 {
			actor.ToX = actor.X
			actor.ToY = actor.Y
		}
		if actor.Speed <= 0 && hasExisting && existing.Speed > 0 {
			actor.Speed = existing.Speed
		}
		if len(actor.MovePath) == 0 {
			actor.MovePath = walkPath(ctx.World.GAT, actor.FromX, actor.FromY, actor.ToX, actor.ToY)
			if len(actor.MovePath) == 0 {
				actor.MovePath = directMovementPath(actor.FromX, actor.FromY, actor.ToX, actor.ToY)
			}
		}
		fastForward := time.Duration(0)
		if actor.HasMoveStartTick && ctx.Session != nil {
			if elapsed, ok := ctx.Session.ElapsedSinceServerTick(actor.MoveStartTick, now); ok {
				fastForward = elapsed
			}
		}
		speed := actorMoveSpeed(actor)
		startX, startY, hasMoveStart, offset := movementStart(actor.MovePath, existing, now, fastForward, speed, actor.FromX, actor.FromY)
		actor.MoveStarted = now.Add(-offset)
		actor.MoveDuration = actorMovementDurationFromWithSpeed(actor.MovePath, actor.FromX, actor.FromY, actor.ToX, actor.ToY, speed, startX, startY, hasMoveStart)
		actor.MoveStartX = startX
		actor.MoveStartY = startY
		actor.HasMoveStart = hasMoveStart
		if hasExisting && actorIsMovingAt(existing, now) {
			actor.WalkDistance = movementWalkDistanceOffset(actor.MovePath, existing, now, offset, speed, startX, startY, hasMoveStart)
		}
	}
	ctx.World.UpsertActor(actor)
}

func applyWarpPosition(ctx client.Context, x, y int) {
	dir := ctx.World.Dir
	if ctx.Session.PlayerDir != 0 {
		dir = ctx.Session.PlayerDir
	}
	ctx.Session.PlayerX = x
	ctx.Session.PlayerY = y
	ctx.World.SetPlayerPosition(x, y, dir)
}

func applyActorSetPosition(ctx client.Context, position network.ActorSetPosition) {
	if isLocalActor(ctx, position.ID) {
		ctx.World.SetPlayerPosition(position.X, position.Y, ctx.World.Dir)
		ctx.Session.PlayerX = position.X
		ctx.Session.PlayerY = position.Y
		return
	}
	upsertActor(ctx, worldstate.Actor{
		ID: position.ID,
		X:  position.X,
		Y:  position.Y,
	})
}

func applyActorJumpPosition(ctx client.Context, position network.ActorJumpPosition) {
	if isLocalActor(ctx, position.ID) {
		ctx.World.SetPlayerPosition(position.X, position.Y, ctx.World.Dir)
		ctx.Session.PlayerX = position.X
		ctx.Session.PlayerY = position.Y
		return
	}
	upsertActor(ctx, worldstate.Actor{
		ID: position.ID,
		X:  position.X,
		Y:  position.Y,
	})
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
	if ctx.World == nil || !isLocalActor(ctx, id) {
		m.stopActorMovementAt(ctx, id, at)
		return
	}
	toX := ctx.World.Player.ToX
	toY := ctx.World.Player.ToY
	stopLocalPlayerMovementAt(ctx, at)
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
	if ctx.World == nil || !isLocalActor(ctx, id) || !m.hasCombatFocus() {
		return false
	}
	actor := ctx.World.Player
	if !actorIsMovingAt(actor, at) || len(actor.MovePath) < 2 {
		return false
	}
	x, y := actorRenderPosition(actor, at)
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
	if ctx.World == nil || !isLocalActor(ctx, resume.id) || !m.hasCombatFocus() {
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
	if isLocalActor(ctx, id) {
		stopLocalPlayerMovementAt(ctx, at)
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok || !actorIsMovingAt(actor, at) {
		return
	}
	x, y := actorRenderPosition(actor, at)
	actor.X = int(math.Round(x))
	actor.Y = int(math.Round(y))
	clearActorMovement(&actor)
	ctx.World.Actors[id] = actor
}

func stopLocalPlayerMovementAt(ctx client.Context, at time.Time) {
	if ctx.World == nil || !actorIsMovingAt(ctx.World.Player, at) {
		return
	}
	x, y := actorRenderPosition(ctx.World.Player, at)
	ctx.World.Player.X = int(math.Round(x))
	ctx.World.Player.Y = int(math.Round(y))
	clearActorMovement(&ctx.World.Player)
	if ctx.Session != nil {
		ctx.Session.PlayerX = ctx.World.Player.X
		ctx.Session.PlayerY = ctx.World.Player.Y
	}
}

func clearActorMovement(actor *worldstate.Actor) {
	actor.Moving = false
	actor.FromX = actor.X
	actor.FromY = actor.Y
	actor.ToX = actor.X
	actor.ToY = actor.Y
	actor.MovePath = nil
	actor.HasMoveStart = false
	actor.MoveStartX = 0
	actor.MoveStartY = 0
	actor.WalkDistance = 0
}

func actorIsMovingAt(actor worldstate.Actor, now time.Time) bool {
	return actor.Moving && !actor.MoveStarted.IsZero() && now.Before(actor.MoveStarted.Add(actor.MoveDuration))
}

func actorRenderPosition(actor worldstate.Actor, now time.Time) (float64, float64) {
	if !actorIsMovingAt(actor, now) || actor.MoveDuration <= 0 {
		return float64(actor.X), float64(actor.Y)
	}
	elapsed := now.Sub(actor.MoveStarted)
	if len(actor.MovePath) >= 2 {
		return renderPathPositionWithSpeed(actor.MovePath, elapsed, actorMoveSpeed(actor), actor.MoveStartX, actor.MoveStartY, actor.HasMoveStart)
	}
	t := float64(elapsed) / float64(actor.MoveDuration)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	x := float64(actor.FromX) + float64(actor.ToX-actor.FromX)*t
	y := float64(actor.FromY) + float64(actor.ToY-actor.FromY)*t
	return x, y
}

func actorRenderDirection(actor worldstate.Actor, now time.Time) int {
	if !actorIsMovingAt(actor, now) || len(actor.MovePath) < 2 {
		return actor.Dir
	}
	elapsed := now.Sub(actor.MoveStarted)
	fromX, fromY, toX, toY := renderPathSegmentWithSpeed(actor.MovePath, elapsed, actorMoveSpeed(actor), actor.MoveStartX, actor.MoveStartY, actor.HasMoveStart)
	if math.Abs(fromX-toX) < 0.001 && math.Abs(fromY-toY) < 0.001 {
		return actor.Dir
	}
	return directionFromFloatDelta(fromX, fromY, toX, toY, actor.Dir)
}

func actorRenderWalkDistance(actor worldstate.Actor, now time.Time) float64 {
	if !actorIsMovingAt(actor, now) || actor.MoveDuration <= 0 {
		return 0
	}
	base := actor.WalkDistance
	elapsed := now.Sub(actor.MoveStarted)
	if elapsed < 0 {
		elapsed = 0
	}
	if len(actor.MovePath) >= 2 {
		return base + pathWalkDistanceWithSpeed(actor.MovePath, elapsed, actorMoveSpeed(actor), actor.MoveStartX, actor.MoveStartY, actor.HasMoveStart)
	}
	t := float64(elapsed) / float64(actor.MoveDuration)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return base + math.Hypot(float64(actor.ToX-actor.FromX), float64(actor.ToY-actor.FromY))*t
}

func movementFinalDirection(path []worldstate.WalkStep, fromX, fromY, toX, toY int, fallback int) int {
	if len(path) >= 2 {
		from := path[len(path)-2]
		to := path[len(path)-1]
		return directionFromDelta(from.X, from.Y, to.X, to.Y, fallback)
	}
	return directionFromDelta(fromX, fromY, toX, toY, fallback)
}

func movementStart(path []worldstate.WalkStep, oldPlayer worldstate.Actor, now time.Time, fastForward time.Duration, speedMS, fromX, fromY int) (float64, float64, bool, time.Duration) {
	startX := float64(fromX)
	startY := float64(fromY)
	duration := time.Duration(0)
	if len(path) > 0 {
		duration = actorMovementDurationWithSpeed(path, path[0].X, path[0].Y, path[len(path)-1].X, path[len(path)-1].Y, speedMS)
	}
	if duration > 0 && fastForward > duration*4 {
		fastForward = 0
	}
	offset := clampMovementOffset(fastForward, duration)
	if actorIsMovingAt(oldPlayer, now) && len(path) >= 2 {
		x, y := actorRenderPosition(oldPlayer, now)
		if currentOffset, ok := pathElapsedAtPositionWithSpeed(path, x, y, speedMS); ok {
			if currentOffset > offset {
				offset = clampMovementOffset(currentOffset, duration)
			}
			return startX, startY, false, offset
		}
		if math.Hypot(x-float64(fromX), y-float64(fromY)) <= 1.25 {
			return x, y, true, 0
		}
		if nearMovementStartLine(path, x, y, 3.0, 0.25) {
			return x, y, true, 0
		}
	}
	return startX, startY, false, offset
}

func nearMovementStartLine(path []worldstate.WalkStep, x, y float64, maxDistance, maxLateral float64) bool {
	if len(path) < 2 {
		return false
	}
	from := path[0]
	to := path[1]
	fromX := float64(from.X)
	fromY := float64(from.Y)
	dx := float64(to.X - from.X)
	dy := float64(to.Y - from.Y)
	lengthSq := dx*dx + dy*dy
	if lengthSq == 0 {
		return false
	}
	if math.Hypot(x-fromX, y-fromY) > maxDistance {
		return false
	}
	t := ((x-fromX)*dx + (y-fromY)*dy) / lengthSq
	if t > 0.25 {
		return false
	}
	px := fromX + dx*t
	py := fromY + dy*t
	return math.Hypot(x-px, y-py) <= maxLateral
}

func movementWalkDistanceOffset(path []worldstate.WalkStep, oldPlayer worldstate.Actor, now time.Time, offset time.Duration, speedMS int, startX, startY float64, hasMoveStart bool) float64 {
	if !actorIsMovingAt(oldPlayer, now) {
		return 0
	}
	previous := actorRenderWalkDistance(oldPlayer, now)
	current := pathWalkDistanceWithSpeed(path, offset, speedMS, startX, startY, hasMoveStart)
	return previous - current
}

func clampMovementOffset(offset, duration time.Duration) time.Duration {
	if offset < 0 {
		return 0
	}
	if duration > 0 && offset > duration {
		return duration
	}
	return offset
}

func actorMovementDurationWithSpeed(path []worldstate.WalkStep, fromX, fromY, toX, toY int, speedMS int) time.Duration {
	return actorMovementDurationFromWithSpeed(path, fromX, fromY, toX, toY, speedMS, 0, 0, false)
}

func actorMovementDurationFromWithSpeed(path []worldstate.WalkStep, fromX, fromY, toX, toY int, speedMS int, startX, startY float64, hasMoveStart bool) time.Duration {
	if len(path) >= 2 {
		total := time.Duration(0)
		for i := 1; i < len(path); i++ {
			segmentStartX, segmentStartY := pathSegmentStart(path, i, startX, startY, hasMoveStart)
			total += movementSegmentDurationFloatWithSpeed(float64(path[i].X)-segmentStartX, float64(path[i].Y)-segmentStartY, speedMS)
		}
		if total > 0 {
			return total
		}
	}
	dx := absInt(toX - fromX)
	dy := absInt(toY - fromY)
	if dx == 0 && dy == 0 {
		return movementSegmentDurationWithSpeed(0, 0, speedMS)
	}
	return movementSegmentDurationWithSpeed(dx, dy, speedMS)
}

const defaultMoveSpeedMS = 150

func actorMoveSpeed(actor worldstate.Actor) int {
	if actor.Speed > 0 {
		return actor.Speed
	}
	return defaultMoveSpeedMS
}

func movementSegmentDurationWithSpeed(dx, dy int, speedMS int) time.Duration {
	if speedMS <= 0 {
		speedMS = defaultMoveSpeedMS
	}
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx == 0 && dy == 0 {
		return time.Duration(speedMS) * time.Millisecond
	}
	if dx != 0 && dy != 0 && dx == dy {
		return time.Duration(math.Round(float64(dx)*float64(speedMS)*math.Sqrt2)) * time.Millisecond
	}
	steps := dx
	if dy > steps {
		steps = dy
	}
	return time.Duration(steps*speedMS) * time.Millisecond
}

func movementSegmentDurationFloatWithSpeed(dx, dy float64, speedMS int) time.Duration {
	if speedMS <= 0 {
		speedMS = defaultMoveSpeedMS
	}
	dist := math.Hypot(dx, dy)
	if dist <= 0 {
		return 0
	}
	return time.Duration(math.Round(dist*float64(speedMS))) * time.Millisecond
}

func renderPathPositionWithSpeed(path []worldstate.WalkStep, elapsed time.Duration, speedMS int, startX, startY float64, hasMoveStart bool) (float64, float64) {
	if len(path) == 0 {
		return 0, 0
	}
	if len(path) == 1 {
		return float64(path[0].X), float64(path[0].Y)
	}
	if elapsed < 0 {
		elapsed = 0
	}
	remaining := elapsed
	for i := 1; i < len(path); i++ {
		fromX, fromY := pathSegmentStart(path, i, startX, startY, hasMoveStart)
		toX := float64(path[i].X)
		toY := float64(path[i].Y)
		duration := movementSegmentDurationFloatWithSpeed(toX-fromX, toY-fromY, speedMS)
		if duration <= 0 {
			continue
		}
		if remaining <= duration || i == len(path)-1 {
			t := float64(remaining) / float64(duration)
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			return fromX + (toX-fromX)*t, fromY + (toY-fromY)*t
		}
		remaining -= duration
	}
	last := path[len(path)-1]
	return float64(last.X), float64(last.Y)
}

func renderPathSegmentWithSpeed(path []worldstate.WalkStep, elapsed time.Duration, speedMS int, startX, startY float64, hasMoveStart bool) (float64, float64, float64, float64) {
	if len(path) == 0 {
		return 0, 0, 0, 0
	}
	if len(path) == 1 {
		x := float64(path[0].X)
		y := float64(path[0].Y)
		return x, y, x, y
	}
	if elapsed <= 0 {
		fromX, fromY := pathSegmentStart(path, 1, startX, startY, hasMoveStart)
		return fromX, fromY, float64(path[1].X), float64(path[1].Y)
	}
	remaining := elapsed
	for i := 1; i < len(path); i++ {
		fromX, fromY := pathSegmentStart(path, i, startX, startY, hasMoveStart)
		toX := float64(path[i].X)
		toY := float64(path[i].Y)
		duration := movementSegmentDurationFloatWithSpeed(toX-fromX, toY-fromY, speedMS)
		if remaining < duration || i == len(path)-1 {
			return fromX, fromY, toX, toY
		}
		remaining -= duration
	}
	last := path[len(path)-1]
	x := float64(last.X)
	y := float64(last.Y)
	return x, y, x, y
}

func pathElapsedAtPositionWithSpeed(path []worldstate.WalkStep, x, y float64, speedMS int) (time.Duration, bool) {
	if len(path) < 2 {
		return 0, false
	}
	bestDistance := math.Inf(1)
	bestElapsed := time.Duration(0)
	elapsedBefore := time.Duration(0)
	for i := 1; i < len(path); i++ {
		from := path[i-1]
		to := path[i]
		dx := float64(to.X - from.X)
		dy := float64(to.Y - from.Y)
		segmentDuration := movementSegmentDurationWithSpeed(to.X-from.X, to.Y-from.Y, speedMS)
		segmentLengthSq := dx*dx + dy*dy
		if segmentLengthSq == 0 {
			elapsedBefore += segmentDuration
			continue
		}
		t := ((x-float64(from.X))*dx + (y-float64(from.Y))*dy) / segmentLengthSq
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		px := float64(from.X) + dx*t
		py := float64(from.Y) + dy*t
		distance := math.Hypot(x-px, y-py)
		if distance < bestDistance {
			bestDistance = distance
			bestElapsed = elapsedBefore + time.Duration(float64(segmentDuration)*t)
		}
		elapsedBefore += segmentDuration
	}
	if bestDistance > 0.05 {
		return 0, false
	}
	return bestElapsed, true
}

func pathWalkDistanceWithSpeed(path []worldstate.WalkStep, elapsed time.Duration, speedMS int, startX, startY float64, hasMoveStart bool) float64 {
	if len(path) < 2 || elapsed <= 0 {
		return 0
	}
	remaining := elapsed
	total := 0.0
	for i := 1; i < len(path); i++ {
		fromX, fromY := pathSegmentStart(path, i, startX, startY, hasMoveStart)
		toX := float64(path[i].X)
		toY := float64(path[i].Y)
		segmentDistance := math.Hypot(toX-fromX, toY-fromY)
		duration := movementSegmentDurationFloatWithSpeed(toX-fromX, toY-fromY, speedMS)
		if duration <= 0 {
			total += segmentDistance
			continue
		}
		if remaining >= duration {
			total += segmentDistance
			remaining -= duration
			continue
		}
		t := float64(remaining) / float64(duration)
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		total += segmentDistance * t
		break
	}
	return total
}

func pathSegmentStart(path []worldstate.WalkStep, segmentIndex int, startX, startY float64, hasMoveStart bool) (float64, float64) {
	if segmentIndex == 1 && hasMoveStart {
		return startX, startY
	}
	from := path[segmentIndex-1]
	return float64(from.X), float64(from.Y)
}

func directMovementPath(fromX, fromY, toX, toY int) []worldstate.WalkStep {
	if fromX == toX && fromY == toY {
		return []worldstate.WalkStep{{X: fromX, Y: fromY}}
	}
	return []worldstate.WalkStep{{X: fromX, Y: fromY}, {X: toX, Y: toY}}
}

func walkTargetInBounds(ctx client.Context, x, y int) bool {
	if x < 0 || y < 0 {
		return false
	}
	if ctx.World.GAT != nil {
		return x < ctx.World.GAT.Width && y < ctx.World.GAT.Height
	}
	if ctx.World.GND != nil {
		return x < ctx.World.GND.Width && y < ctx.World.GND.Height
	}
	return x <= 1023 && y <= 1023
}

func directionFromDelta(fromX, fromY, toX, toY int, fallback int) int {
	return directionFromFloatDelta(float64(fromX), float64(fromY), float64(toX), float64(toY), normalizeDirectionIndex(fallback))
}

func directionFromFloatDelta(fromX, fromY, toX, toY float64, fallback int) int {
	dx := toX - fromX
	dy := toY - fromY
	if math.Abs(dx) < 0.001 && math.Abs(dy) < 0.001 {
		if fallback < 0 {
			return 0
		}
		return fallback & 7
	}
	actionDir := int(math.Round(-math.Atan2(dy, dx)/(math.Pi/4)+6)) & 7
	return (4 - actionDir) & 7
}

func clickedWalkTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int) (int, int, bool) {
	return rayWalkCell(ctx, projection, mouseX, mouseY)
}

func hoveredWalkCell(ctx client.Context, projection sceneProjection, mouseX, mouseY int) (int, int, bool) {
	return rayWalkCell(ctx, projection, mouseX, mouseY)
}

func (m *WorldMode) hoveredWalkCell(ctx client.Context, projection sceneProjection, mouseX, mouseY int) (int, int, bool) {
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
		targetX:    projection.playerX,
		targetY:    projection.playerY,
		targetZ:    projection.playerZ,
		cameraYaw:  projection.cameraYaw,
		cameraZoom: projection.cameraZoom,
		screenW:    projection.screenW,
		screenH:    projection.screenH,
	}
	if m.hoveredWalk.valid && m.hoveredWalk.key == key {
		return m.hoveredWalk.x, m.hoveredWalk.y, m.hoveredWalk.ok
	}
	x, y, ok := hoveredWalkCell(ctx, projection, mouseX, mouseY)
	m.hoveredWalk = hoveredWalkCellCache{
		valid: true,
		key:   key,
		x:     x,
		y:     y,
		ok:    ok,
	}
	return x, y, ok
}

const (
	walkRayStep        = 0.5
	walkRayMaxDistance = 300.0
	walkRayHitEpsilon  = 0.5
)

func rayWalkCell(ctx client.Context, projection sceneProjection, mouseX, mouseY int) (int, int, bool) {
	if ctx.World == nil || ctx.World.GAT == nil {
		return 0, 0, false
	}
	eye, dir, ok := projection.ScreenRay(mouseX, mouseY)
	if !ok || dir.y >= 0 {
		return 0, 0, false
	}
	previousDelta := math.Inf(1)
	for distance := 1.0; distance <= walkRayMaxDistance; distance += walkRayStep {
		point := add3(eye, mul3(dir, distance))
		x, y := rayPointGATCell(point)
		if !ctx.World.GAT.InBounds(x, y) {
			previousDelta = math.Inf(1)
			continue
		}
		height := terrainHeightAtRenderPoint(ctx.World, point.x, point.z)
		delta := point.y - height
		if math.Abs(delta) <= walkRayHitEpsilon || (previousDelta > 0 && delta < 0) {
			if ctx.World.GAT.Walkable(x, y) {
				return x, y, true
			}
			return 0, 0, false
		}
		previousDelta = delta
	}
	return 0, 0, false
}

func rayPointGATCell(point modelPoint3) (int, int) {
	return int(math.Floor(point.x)), int(math.Floor(point.z))
}

func terrainHeightAtRenderPoint(world *worldstate.World, x, y float64) float64 {
	return terrainHeightAt(world, x-0.5, y-0.5)
}
