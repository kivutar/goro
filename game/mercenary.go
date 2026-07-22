package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) openMercenaryContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || ctx.Input.Pressed(input.KeyAlt) || uiPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedMercenaryTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok {
		return false
	}
	if ctx.Session != nil {
		ctx.Session.Mercenary.ID = actor.ID
		ctx.Session.Mercenary.Active = true
		ctx.Session.Mercenary.Job = actor.Job
		if ctx.Session.Mercenary.Name == "" {
			ctx.Session.Mercenary.Name = actor.Name
		}
	}
	glog.Debugf("mercenary context target id=%d name=%q job=%d", actor.ID, actor.Name, actor.Job)
	m.ui.mercenaryContext.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY, mercenaryAggressive(ctx))
	return true
}

func (m *WorldMode) handleMercenaryContextAction(ctx client.Context, action gameui.MercenaryContextAction) {
	switch action.Kind {
	case gameui.MercenaryContextActionInfo:
		m.openMercenaryInfo(ctx)
	case gameui.MercenaryContextActionToggleAssist:
		m.toggleMercenaryAssist(ctx)
	}
}

func (m *WorldMode) handleMercenaryInfoAction(ctx client.Context, action gameui.MercenaryInfoAction) {
	switch action.Kind {
	case gameui.MercenaryInfoActionSkill:
		m.ui.mercenarySkill.Toggle(ctx, m)
	case gameui.MercenaryInfoActionDelete:
		m.openMercenaryDeleteConfirm(ctx)
	}
}

func (m *WorldMode) openMercenaryInfo(ctx client.Context) {
	if ctx.Session == nil || (!ctx.Session.Mercenary.Active && ctx.Session.Mercenary.ID == 0) {
		return
	}
	m.ui.mercenaryInfo.OpenInfo(ctx, ctx.Session.Mercenary)
}

func (m *WorldMode) openMercenaryDeleteConfirm(ctx client.Context) {
	m.ui.mercenaryConfirm.Open(ctx, "Dismiss Mercenary", "Are you sure that you want to delete?", func() {
		m.sendMercenaryDelete(ctx)
	}, nil)
}

func (m *WorldMode) sendMercenaryDelete(ctx client.Context) {
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Mercenary dismiss failed: not connected.")
		return
	}
	if err := ctx.Network.SendMercenaryCommand(network.MercenaryCommandDelete); err != nil {
		m.ui.console.AddErrorMessage("Mercenary dismiss failed.")
		glog.Warnf("mercenary dismiss failed: %v", err)
		return
	}
	m.mercDeleteID = currentMercenaryID(ctx)
}

func (m *WorldMode) toggleMercenaryAssist(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.MercenaryAggressive = !ctx.Session.MercenaryAggressive
	glog.Debugf("mercenary assist aggressive=%t", ctx.Session.MercenaryAggressive)
}

func mercenaryAggressive(ctx client.Context) bool {
	return ctx.Session != nil && ctx.Session.MercenaryAggressive
}

func currentMercenaryID(ctx client.Context) uint32 {
	if ctx.Session != nil && ctx.Session.Mercenary.ID != 0 {
		return ctx.Session.Mercenary.ID
	}
	return findCompanionActorID(ctx, actorObjectTypeMercenary)
}

func (m *WorldMode) clearDeletedMercenary(ctx client.Context) {
	id := m.mercDeleteID
	if id == 0 && ctx.Session != nil {
		id = ctx.Session.Mercenary.ID
	}
	m.mercDeleteID = 0
	if ctx.Session != nil {
		ctx.Session.Mercenary = session.Companion{}
	}
	if id != 0 {
		delete(m.companionAI.msg, id)
		delete(m.companionAI.resMsg, id)
	}
	m.ui.mercenaryInfo.Close()
	m.ui.mercenarySkill.Close()
	m.ui.mercenaryContext.Close()
}

func clickedMercenaryTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	expectedID := uint32(0)
	if ctx.Session != nil {
		expectedID = ctx.Session.Mercenary.ID
	}
	if expectedID == 0 {
		expectedID = findCompanionActorID(ctx, actorObjectTypeMercenary)
	}
	actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, deadActors)
	if !ok || actor.ID == 0 || !actor.HasObjectType || actor.ObjectType != actorObjectTypeMercenary {
		return worldstate.Actor{}, false
	}
	if expectedID != 0 && actor.ID != expectedID {
		return worldstate.Actor{}, false
	}
	return actor, true
}
