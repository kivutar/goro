package game

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"time"

	uiwidget "github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

const (
	cursorActionDefault = 0
	cursorActionTalk    = 1
	cursorActionClick   = 2
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
	view      *playerSpriteView
	viewMiss  bool
	fallback  *render.Image
	action    int
	started   time.Time
	loadTried bool
}

var cursorActionInfos = map[int]cursorActionInfo{
	cursorActionDefault: {drawX: 1, drawY: 19, delayMult: 2.0},
	cursorActionTalk:    {drawX: 20, drawY: 40, delayMult: 1.0},
	cursorActionRotate:  {drawX: 18, drawY: 26, delayMult: 1.0},
	cursorActionWarp:    {drawX: 10, drawY: 32, delayMult: 1.0},
	cursorActionPick:    {drawX: 20, drawY: 40, delayMult: 1.0},
	cursorActionTarget:  {drawX: 20, drawY: 50, delayMult: 0.5},
	cursorActionTarget2: {drawX: 20, drawY: 50, delayMult: 0.5},
	cursorActionNoWalk:  {drawX: 13, drawY: 25, delayMult: 1.0},
}

func (m *WorldMode) drawROCursor(screen *render.Image, ctx client.Context, projection sceneProjection, now time.Time) {
	if ctx.Input == nil {
		return
	}
	render.SetCursorMode(render.CursorModeHidden)
	action := m.cursorDesiredAction(ctx, projection, now)
	state := m.cursorState()
	state.draw(screen, ctx, action, now)
	m.storeCursorState(state)
	drawPendingSkillCursorLevel(screen, ctx, m.pendingSkill.skill)
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

func (s *roCursorState) ensureLoaded(ctx client.Context) {
	if s == nil || s.loadTried || s.view != nil || s.viewMiss {
		return
	}
	s.loadTried = true
	if view, status := loadCursorSpriteView(ctx.Resources); view != nil {
		s.view = view
	} else {
		s.viewMiss = true
		log.Printf("cursor resources unavailable: %s", status)
	}
}

func (s *roCursorState) draw(screen *render.Image, ctx client.Context, action int, now time.Time) {
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
	opts.GeoM.Translate(float64(ctx.Input.MouseX)-frame.anchorX, float64(ctx.Input.MouseY)-frame.anchorY)
	opts.Filter = spriteDrawFilter()
	screen.DrawImage(frame.image, &opts)
}

func (s *roCursorState) frame(action int, info cursorActionInfo, now time.Time) (*spriteBillboard, bool) {
	if s == nil || s.view == nil || s.viewMiss || s.view.act == nil {
		return nil, false
	}
	if action < 0 || action >= len(s.view.act.Actions) || len(s.view.act.Actions[action].Animations) == 0 {
		action = cursorActionDefault
		info = cursorInfo(action)
	}
	actionDef := s.view.act.Actions[action]
	delay := float64(actionDef.DelayMS) * info.delayMult
	motion := spriteMotionIndexWithDelay(actionDef, s.started, now, true, delay)
	return cursorFrameBillboard(s.view, action, motion, info.drawX, info.drawY)
}

func (m *WorldMode) cursorDesiredAction(ctx client.Context, projection sceneProjection, now time.Time) int {
	mouseX, mouseY := ctx.Input.MouseX, ctx.Input.MouseY
	if action, ok := uiCursorAction(ctx); ok {
		return action
	}
	if ctx.Input.MousePressed(render.MouseButtonRight) {
		return cursorActionRotate
	}
	if m.pendingSkill.skill.ID != 0 {
		if _, ok := clickedSkillTarget(ctx, projection, m.pendingSkill.skill, mouseX, mouseY, now, m.actorDeaths); ok {
			return cursorActionTarget2
		}
		return cursorActionTarget
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
	if ctx.Input.MousePressed(render.MouseButtonLeft) {
		return cursorActionClick
	}
	if ctx.World != nil && ctx.World.GAT != nil {
		if _, _, ok := hoveredWalkCell(ctx, projection, mouseX, mouseY); !ok {
			return cursorActionNoWalk
		}
	}
	return cursorActionDefault
}

func uiCursorAction(ctx client.Context) (int, bool) {
	if ctx.UIApp == nil {
		return 0, false
	}
	if ctx.UIApp.Cursor() == uiwidget.CursorPointer {
		return cursorActionClick, true
	}
	return 0, false
}

func hoveredCursorActor(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
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
		actorX, actorY := actor.RenderPosition(now)
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

func cursorActorCanTalk(actor worldstate.Actor) bool {
	if actor.ID == 0 || !actor.HasObjectType {
		return false
	}
	return actor.ObjectType != actorObjectTypeMob && !isWarpActor(actor)
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

func drawFallbackROCursor(screen, img *render.Image, mouseX, mouseY int) {
	if img == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(mouseX), float64(mouseY))
	opts.Filter = spriteDrawFilter()
	screen.DrawImage(img, &opts)
}

func drawPendingSkillCursorLevel(screen *render.Image, ctx client.Context, skill session.Skill) {
	if screen == nil || ctx.Input == nil {
		return
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return
	}
	label := fmt.Sprintf("Lv%d", skill.Level)
	x := ctx.Input.MouseX + 18
	y := ctx.Input.MouseY + 16
	width := len([]rune(label))*7 + 8
	gameui.DrawSurface(screen, x, y, width, 15, gameui.PanelBodyColor, gameui.WindowBorderColor)
	render.DebugPrintAtColor(screen, label, x+4, y+1, gameui.TitleTextColor)
}
