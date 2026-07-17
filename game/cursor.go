package game

import (
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/world"
)

const (
	cursorActionDefault = 0
	cursorActionTalk    = 1
	cursorActionClick   = 2
	cursorActionLock    = 3
	cursorActionRotate  = 4
	cursorActionAttack  = 5
	cursorActionWarp    = 7
	cursorActionPick    = 9
	cursorActionTarget  = 10
	cursorActionTarget2 = 11
	cursorActionNoWalk  = 13
)

type cursorActionInfo struct {
	drawX     float64
	drawY     float64
	delayMult float64
}

type roCursorState struct {
	view      *spriteView
	viewMiss  bool
	fallback  *render.Image
	action    int
	started   time.Time
	loadTried bool
}

var cursorActionInfos = map[int]cursorActionInfo{
	cursorActionDefault: {drawX: 1, drawY: 19, delayMult: 2.0},
	cursorActionTalk:    {drawX: 20, drawY: 40, delayMult: 1.0},
	cursorActionLock:    {drawX: 20, drawY: 40, delayMult: 1.0},
	cursorActionRotate:  {drawX: 18, drawY: 26, delayMult: 1.0},
	cursorActionWarp:    {drawX: 10, drawY: 32, delayMult: 1.0},
	cursorActionPick:    {drawX: 20, drawY: 40, delayMult: 1.0},
	cursorActionTarget:  {drawX: 20, drawY: 50, delayMult: 0.5},
	cursorActionTarget2: {drawX: 20, drawY: 50, delayMult: 0.5},
	cursorActionNoWalk:  {drawX: 13, drawY: 25, delayMult: 1.0},
}

func (m *WorldMode) drawROCursor(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
	if ctx.Input == nil {
		return
	}
	render.SetCursorMode(render.CursorModeHidden)
	action := m.cursorDesiredAction(ctx, projection, now)
	magnetX, magnetY := m.cursorMagnetOffset(ctx, projection, action, now)
	state := m.cursorState()
	state.draw(screen, ctx, action, now, magnetX, magnetY)
	m.storeCursorState(state)
	ui.DrawPendingSkillCursorLevel(screen, ctx, m.pendingSkill.skill)
}

func (m *WorldMode) cursorState() *roCursorState {
	return &roCursorState{
		view:     m.cursorView,
		viewMiss: m.cursorViewMiss,
		fallback: m.cursorFallback,
		action:   m.cursorAction,
		started:  m.cursorStarted,
	}
}

func (m *WorldMode) storeCursorState(state *roCursorState) {
	if state == nil {
		return
	}
	m.cursorView = state.view
	m.cursorViewMiss = state.viewMiss
	m.cursorFallback = state.fallback
	m.cursorAction = state.action
	m.cursorStarted = state.started
}

func (m *WorldMode) loadedCursorState(ctx client.Context) *roCursorState {
	state := m.cursorState()
	state.ensureLoaded(ctx)
	m.storeCursorState(state)
	return state
}

func (s *roCursorState) ensureLoaded(ctx client.Context) {
	if s == nil || s.loadTried || s.view != nil || s.viewMiss {
		return
	}
	s.loadTried = true
	if view, status := loadCursorSpriteView(ctx.Resources); view != nil {
		s.view = view
	} else {
		s.viewMiss = true
		glog.Warnf("cursor resources unavailable: %s", status)
	}
}

func (s *roCursorState) draw(screen *render.Frame, ctx client.Context, action int, now time.Time, magnetX, magnetY float64) {
	if s == nil || screen == nil || ctx.Input == nil {
		return
	}
	s.ensureLoaded(ctx)
	if action != s.action {
		s.action = action
		s.started = now
	}
	if s.started.IsZero() {
		s.started = now
	}
	frame, ok := s.frame(action, cursorInfo(action), now)
	if !ok {
		drawFallbackROCursor(screen, s.fallbackTexture(), ctx.Input.MouseX, ctx.Input.MouseY)
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(ctx.Input.MouseX)-frame.anchorX-magnetX, float64(ctx.Input.MouseY)-frame.anchorY-magnetY)
	opts.Filter = spriteDrawFilter()
	screen.DrawImage(frame.image, &opts)
}

func (s *roCursorState) frame(action int, info cursorActionInfo, now time.Time) (*spriteBillboard, bool) {
	return s.frameAt(action, info, s.started, now)
}

func (s *roCursorState) frameAt(action int, info cursorActionInfo, start, now time.Time) (*spriteBillboard, bool) {
	if s == nil || s.view == nil || s.viewMiss || s.view.act == nil {
		return nil, false
	}
	if action < 0 || action >= len(s.view.act.Actions) || len(s.view.act.Actions[action].Animations) == 0 {
		action = cursorActionDefault
		info = cursorInfo(action)
	}
	actionDef := s.view.act.Actions[action]
	delay := float64(actionDef.DelayMS) * info.delayMult
	motion := spriteMotionIndexWithDelay(actionDef, start, now, true, delay)
	return cursorFrameBillboard(s.view, action, motion, info.drawX, info.drawY)
}

func (m *WorldMode) cursorDesiredAction(ctx client.Context, projection sceneProjection, now time.Time) int {
	mouseX, mouseY := ctx.Input.MouseX, ctx.Input.MouseY
	if action, ok := uiCursorAction(ctx); ok {
		return action
	}
	if ctx.Input.MousePressed(input.MouseButtonRight) {
		return cursorActionRotate
	}
	if m.pendingSkill.skill.ID != 0 {
		if _, ok := clickedSkillTarget(ctx, projection, m.pendingSkill.skill, mouseX, mouseY, now, m.actorDeaths); ok {
			return cursorActionTarget2
		}
		return cursorActionTarget
	}
	if m.pendingPetCapture.active {
		if _, ok := clickedSkillTarget(ctx, projection, petCaptureTargetSkill(), mouseX, mouseY, now, m.actorDeaths); ok {
			return cursorActionTarget2
		}
		return cursorActionTarget
	}
	if _, ok := m.hoveredVendingBoard(ctx, projection, mouseX, mouseY, now); ok {
		return cursorActionClick
	}
	if _, ok := m.hoveredChatRoomBoard(ctx, projection, mouseX, mouseY, now); ok {
		return cursorActionClick
	}
	if _, ok := clickedGroundItem(ctx, projection, mouseX, mouseY, now); ok {
		return cursorActionPick
	}
	if actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, m.actorDeaths); ok {
		switch {
		case isWarpActor(actor):
			return cursorActionWarp
		case actorCanBeAttackClicked(ctx, actor):
			return cursorActionAttack
		case cursorActorCanTalk(actor):
			return cursorActionTalk
		}
	}
	if ctx.Input.MousePressed(input.MouseButtonLeft) {
		return cursorActionClick
	}
	if ctx.World != nil && ctx.World.GAT != nil {
		if _, _, ok := m.hoveredWalkCell(ctx, projection, mouseX, mouseY); !ok {
			return cursorActionNoWalk
		}
	}
	return cursorActionDefault
}

func (m *WorldMode) cursorMagnetOffset(ctx client.Context, projection sceneProjection, action int, now time.Time) (float64, float64) {
	if ctx.Input == nil {
		return 0, 0
	}
	switch action {
	case cursorActionPick:
		if !cursorSnapItems(ctx) {
			return 0, 0
		}
		item, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now)
		if !ok {
			return 0, 0
		}
		x, y := floorItemWorldPosition(item)
		z := floorItemRenderHeight(ctx.World, item, now)
		point := projection.Project(cellCenter(x), cellCenter(y), z)
		scale := actorBillboardScreenScale(projection, cellCenter(x), cellCenter(y), z) * 0.42
		targetX, targetY := groundItemPickBoundsCenter(float64(point.x), float64(point.y), scale)
		return float64(ctx.Input.MouseX) - targetX, float64(ctx.Input.MouseY) - targetY
	case cursorActionAttack:
		if !cursorSnapTargets(ctx) {
			return 0, 0
		}
		actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
		if !ok {
			return 0, 0
		}
		return m.cursorActorMagnetOffset(ctx, projection, actor, now)
	case cursorActionTarget2:
		if !cursorSnapTargets(ctx) {
			return 0, 0
		}
		skill, ok := m.pendingTargetCursorSkill()
		if !ok {
			return 0, 0
		}
		actor, ok := clickedSkillTarget(ctx, projection, skill, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
		if !ok || !cursorActorCanSnap(actor) {
			return 0, 0
		}
		return m.cursorActorMagnetOffset(ctx, projection, actor, now)
	default:
		return 0, 0
	}
}

func (m *WorldMode) pendingTargetCursorSkill() (session.Skill, bool) {
	if m.pendingSkill.skill.ID != 0 {
		return m.pendingSkill.skill, true
	}
	if m.pendingPetCapture.active {
		return petCaptureTargetSkill(), true
	}
	return session.Skill{}, false
}

func (m *WorldMode) cursorActorMagnetOffset(ctx client.Context, projection sceneProjection, actor world.Actor, now time.Time) (float64, float64) {
	if ctx.Input == nil || ctx.World == nil {
		return 0, 0
	}
	actorX, actorY := actorRenderPosition(actor, now)
	terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
	scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
	if targetX, targetY, ok := m.cursorActorSpriteCenter(ctx, projection, actor, point, scale, now); ok {
		return float64(ctx.Input.MouseX) - targetX, float64(ctx.Input.MouseY) - targetY
	}
	targetX, targetY := actorPickBoundsCenter(float64(point.x), float64(point.y), scale)
	return float64(ctx.Input.MouseX) - targetX, float64(ctx.Input.MouseY) - targetY
}

func (m *WorldMode) cursorActorSpriteCenter(ctx client.Context, projection sceneProjection, actor world.Actor, point screenPoint, scale float64, now time.Time) (float64, float64, bool) {
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return 0, 0, false
	}
	state := m.nonPCSpriteState(actor, now)
	state.cameraYaw = projection.cameraYaw
	billboard, ok := singleSpriteBillboardForState(view, state, now)
	if !ok {
		return 0, 0, false
	}
	return spriteBillboardScreenCenter(billboard, point, scale)
}

func spriteBillboardScreenCenter(billboard *spriteBillboard, point screenPoint, scale float64) (float64, float64, bool) {
	if billboard == nil || billboard.image == nil {
		return 0, 0, false
	}
	bounds := billboard.image.Bounds()
	if bounds.Empty() {
		return 0, 0, false
	}
	scale = normalizePickScale(scale)
	x := float64(point.x) + (float64(bounds.Dx())/2-billboard.anchorX)*scale
	y := float64(point.y) + (float64(bounds.Dy())/2-billboard.anchorY)*scale
	return x, y, true
}

func cursorActorCanSnap(actor world.Actor) bool {
	if !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case actorObjectTypeMob, actorObjectTypeNPCABR, actorObjectTypeNPCBionic:
		return true
	default:
		return false
	}
}

func cursorSnapTargets(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.SnapTargets
	}
	return ctx.Config.Gameplay.SnapTargets
}

func cursorSnapItems(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.SnapItems
	}
	return ctx.Config.Gameplay.SnapItems
}

func uiCursorAction(ctx client.Context) (int, bool) {
	if ctx.UIApp != nil && ctx.UIApp.Cursor() == widget.CursorPointer {
		return cursorActionClick, true
	}
	if uiPointerBlocked(ctx) {
		return cursorActionDefault, true
	}
	return 0, false
}

func uiPointerBlocked(ctx client.Context) bool {
	if ctx.Input == nil || ctx.UIManager == nil {
		return false
	}
	blocker, ok := ctx.UIManager.(interface {
		PointerBlocked(x, y int) bool
	})
	return ok && blocker.PointerBlocked(ctx.Input.MouseX, ctx.Input.MouseY)
}

func hoveredCursorActor(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (world.Actor, bool) {
	if ctx.World == nil {
		return world.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best world.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if isLocalActor(ctx, actor.ID) || actor.ID == 0 {
			continue
		}
		if int(actor.Job) == actorJobClearNPC {
			continue
		}
		actorX, actorY := actorRenderPosition(actor, now)
		terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
		scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
		if !pointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.x), float64(point.y), scale) {
			continue
		}
		dx := float64(point.x) - float64(mouseX)
		dy := float64(point.y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func cursorActorCanTalk(actor world.Actor) bool {
	if actor.ID == 0 || !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case actorObjectTypeNPC, actorObjectTypeNPC2:
		return !isWarpActor(actor)
	default:
		return false
	}
}

func cursorInfo(action int) cursorActionInfo {
	if info, ok := cursorActionInfos[action]; ok {
		return info
	}
	return cursorActionInfos[cursorActionDefault]
}

func (s *roCursorState) fallbackTexture() *render.Image {
	if s.fallback != nil {
		return s.fallback
	}
	const width = 18
	const height = 24
	mask := []string{
		"X.................",
		"XX................",
		"XOX...............",
		"XOOX..............",
		"XOOOX.............",
		"XOOOOX............",
		"XOOOOOX...........",
		"XOOOOOOX..........",
		"XOOOOOOOX.........",
		"XOOOOOOOOX........",
		"XOOOOOOOOOX.......",
		"XOOOOOOOOOOX......",
		"XOOOOXXXXXXXX.....",
		"XOOXOX............",
		"XOX..OX...........",
		"XX....OX..........",
		"X......OX.........",
		"........X.........",
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y, row := range mask {
		for x, ch := range row {
			switch ch {
			case 'X':
				img.SetRGBA(x, y, color.RGBA{A: 255})
			case 'O':
				img.SetRGBA(x, y, color.RGBA{R: 246, G: 246, B: 246, A: 255})
			}
		}
	}
	s.fallback = render.NewImageFromImage(img)
	return s.fallback
}

func drawFallbackROCursor(screen *render.Frame, img *render.Image, mouseX, mouseY int) {
	if img == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(mouseX), float64(mouseY))
	opts.Filter = spriteDrawFilter()
	screen.DrawImage(img, &opts)
}
