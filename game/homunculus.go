package game

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) openHomunculusContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || ctx.Input.Pressed(input.KeyAlt) || uiPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedHomunculusTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok {
		return false
	}
	if ctx.Session != nil {
		ctx.Session.Homunculus.ID = actor.ID
		ctx.Session.Homunculus.Active = true
		ctx.Session.Homunculus.Job = actor.Job
	}
	glog.Debugf("homunculus context target id=%d name=%q job=%d", actor.ID, actor.Name, actor.Job)
	m.ui.homunculusContext.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY, homunculusAggressive(ctx))
	return true
}

func (m *WorldMode) handleHomunculusContextAction(ctx client.Context, action gameui.HomunculusContextAction) {
	switch action.Kind {
	case gameui.HomunculusContextActionInfo:
		m.openHomunculusInfo(ctx)
	case gameui.HomunculusContextActionFeed:
		m.openHomunculusFeedConfirm(ctx)
	case gameui.HomunculusContextActionToggleAssist:
		m.toggleHomunculusAssist(ctx)
	}
}

func (m *WorldMode) handleHomunculusInfoAction(ctx client.Context, action gameui.HomunculusInfoAction) {
	switch action.Kind {
	case gameui.HomunculusInfoActionSkill:
		m.ui.homunculusSkill.Toggle(ctx, m)
	case gameui.HomunculusInfoActionFeed:
		m.openHomunculusFeedConfirm(ctx)
	case gameui.HomunculusInfoActionDelete:
		m.openHomunculusDeleteConfirm(ctx)
	case gameui.HomunculusInfoActionRename:
		m.sendHomunculusRename(ctx, action.Name)
	}
}

func (m *WorldMode) openHomunculusInfo(ctx client.Context) {
	if ctx.Session == nil || (!ctx.Session.Homunculus.Active && ctx.Session.Homunculus.ID == 0) {
		return
	}
	if ctx.Network != nil {
		if err := ctx.Network.SendHomunculusCommand(network.HomunculusCommandInfo); err != nil {
			m.ui.console.AddErrorMessage("Homunculus info request failed.")
			glog.Warnf("homunculus info request failed: %v", err)
		}
	}
	m.ui.homunculusInfo.OpenInfo(ctx, ctx.Session.Homunculus)
}

func (m *WorldMode) openHomunculusFeedConfirm(ctx client.Context) {
	m.ui.homunculusConfirm.Open(ctx, "Feed Homunculus", "Are you sure you want to feed your homunculus?", func() {
		m.sendHomunculusFeed(ctx)
	}, nil)
}

func (m *WorldMode) openHomunculusDeleteConfirm(ctx client.Context) {
	m.ui.homunculusConfirm.Open(ctx, "Delete Homunculus", "Are you sure that you want to delete?", func() {
		m.sendHomunculusDelete(ctx)
	}, nil)
}

func (m *WorldMode) sendHomunculusFeed(ctx client.Context) {
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Homunculus feed failed: not connected.")
		return
	}
	if err := ctx.Network.SendHomunculusFeed(); err != nil {
		m.ui.console.AddErrorMessage("Homunculus feed failed.")
		glog.Warnf("homunculus feed failed: %v", err)
	}
}

func (m *WorldMode) sendHomunculusDelete(ctx client.Context) {
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Homunculus delete failed: not connected.")
		return
	}
	if err := ctx.Network.SendHomunculusDelete(); err != nil {
		m.ui.console.AddErrorMessage("Homunculus delete failed.")
		glog.Warnf("homunculus delete failed: %v", err)
		return
	}
	m.homDeleteID = currentHomunculusID(ctx)
}

func (m *WorldMode) sendHomunculusRename(ctx client.Context, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Homunculus rename failed: not connected.")
		return
	}
	if err := ctx.Network.SendHomunculusRename(name); err != nil {
		m.ui.console.AddErrorMessage("Homunculus rename failed.")
		glog.Warnf("homunculus rename failed name=%q: %v", name, err)
	}
}

func (m *WorldMode) applyHomunculusFeedResultMessage(ctx client.Context, result network.HomunculusFeedResult) {
	if result.Result {
		return
	}
	name := fmt.Sprintf("item %d", result.ItemID)
	if ctx.Resources != nil {
		if resolved, ok := ctx.Resources.ItemDisplayName(int(result.ItemID), true); ok && strings.TrimSpace(resolved) != "" {
			name = resolved
		}
	}
	m.ui.console.AddErrorMessage("Failed to feed homunculus with %s.", name)
}

func (m *WorldMode) toggleHomunculusAssist(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.HomunculusAggressive = !ctx.Session.HomunculusAggressive
	glog.Debugf("homunculus assist aggressive=%t", ctx.Session.HomunculusAggressive)
}

func homunculusAggressive(ctx client.Context) bool {
	return ctx.Session != nil && ctx.Session.HomunculusAggressive
}

func currentHomunculusID(ctx client.Context) uint32 {
	if ctx.Session != nil && ctx.Session.Homunculus.ID != 0 {
		return ctx.Session.Homunculus.ID
	}
	return findCompanionActorID(ctx, actorObjectTypeHomunculus)
}

func (m *WorldMode) clearDeletedHomunculus(ctx client.Context) {
	id := m.homDeleteID
	if id == 0 && ctx.Session != nil {
		id = ctx.Session.Homunculus.ID
	}
	m.homDeleteID = 0
	if ctx.Session != nil {
		ctx.Session.Homunculus = session.Companion{}
	}
	if id != 0 {
		delete(m.companionAI.msg, id)
		delete(m.companionAI.resMsg, id)
	}
	m.ui.homunculusInfo.Close()
	m.ui.homunculusSkill.Close()
	m.ui.homunculusContext.Close()
}

func clickedHomunculusTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	expectedID := uint32(0)
	if ctx.Session != nil {
		expectedID = ctx.Session.Homunculus.ID
	}
	if expectedID == 0 {
		expectedID = findCompanionActorID(ctx, actorObjectTypeHomunculus)
	}
	actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, deadActors)
	if !ok || actor.ID == 0 || !actor.HasObjectType || actor.ObjectType != actorObjectTypeHomunculus {
		return worldstate.Actor{}, false
	}
	if expectedID != 0 && actor.ID != expectedID {
		return worldstate.Actor{}, false
	}
	return actor, true
}

func (m *WorldMode) maybeQueueAggressiveCompanionTarget(ctx client.Context, kind companionAIKind, id uint32, actorDeaths map[uint32]time.Time) {
	if ctx.Session == nil || ctx.World == nil || id == 0 || !companionAIAggressive(ctx, kind) {
		return
	}
	source, ok := companionActorByID(ctx, id)
	if !ok {
		return
	}
	targetID := uint32(0)
	bestDistance := 32.0
	for _, actor := range ctx.World.Actors {
		if actor.ID == 0 || actor.ID == id || isLocalActor(ctx, actor.ID) {
			continue
		}
		if kind == companionAIHomunculus && actor.ID == ctx.Session.Mercenary.ID {
			continue
		}
		if kind == companionAIMercenary && actor.ID == ctx.Session.Homunculus.ID {
			continue
		}
		if !m.luaCompanionIsMonster(ctx, actor.ID, actorDeaths) {
			continue
		}
		distance := companionCellDistance(source.X, source.Y, actor.X, actor.Y)
		if distance < bestDistance {
			bestDistance = distance
			targetID = actor.ID
		}
	}
	if targetID != 0 {
		m.setCompanionAIMessage(id, fmt.Sprintf("3,%d", targetID))
	}
}

func companionAIAggressive(ctx client.Context, kind companionAIKind) bool {
	if ctx.Session == nil {
		return false
	}
	switch kind {
	case companionAIHomunculus:
		return ctx.Session.HomunculusAggressive
	case companionAIMercenary:
		return ctx.Session.MercenaryAggressive
	default:
		return false
	}
}

func companionIDForAIKind(ctx client.Context, kind companionAIKind) uint32 {
	if ctx.Session == nil {
		return 0
	}
	switch kind {
	case companionAIHomunculus:
		return ctx.Session.Homunculus.ID
	case companionAIMercenary:
		return ctx.Session.Mercenary.ID
	default:
		return 0
	}
}

func companionCellDistance(x1, y1, x2, y2 int) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return math.Sqrt(dx*dx + dy*dy)
}
