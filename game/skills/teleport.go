package skills

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

func (m *System) ApplyWarpPointList(ctx client.Context, list network.WarpPointList) {
	if list.SkillID != gameui.TeleportSkillID && list.SkillID != gameui.WarpPortalSkillID {
		glog.Debugf("warp point list ignored skill=%d maps=%v", list.SkillID, list.MapNames)
		return
	}
	if gameui.TeleportWarpListBypassesModal(list) {
		m.autoSelectTeleportRandom(ctx, list)
		return
	}
	m.bindings.TeleportModal.OpenWarpPointList(list)
	glog.Debugf("warp point destination list skill=%d maps=%v", list.SkillID, list.MapNames)
}

func (m *System) ApplyRememberWarpPointAck(_ client.Context, ack network.RememberWarpPointAck) {
	switch ack.Result {
	case 0:
		m.bindings.Console.AddBlueMessage("Saved location as a Memo Point for Warp skill.")
	case 1:
		m.bindings.Console.AddErrorMessage("Skill Level is not high enough.")
	case 2:
		m.bindings.Console.AddErrorMessage("You haven't learned Warp.")
	default:
		m.bindings.Console.AddErrorMessage("Memo failed.")
	}
	glog.Debugf("remember warp point ack result=%d", ack.Result)
}

func (m *System) autoSelectTeleportRandom(ctx client.Context, list network.WarpPointList) {
	if ctx.Network == nil {
		return
	}
	if err := ctx.Network.SendSelectWarpPoint(list.SkillID, gameui.TeleportRandomMap); err != nil {
		return
	}
	// Like the destination dialog, only select here. ZC_CHANGEMAP drives
	// the local transition; observers get the departure effect from vanish.
	m.bindings.TeleportModal.Reset()
	glog.Debugf("teleport random selected automatically skill=%d maps=%v", list.SkillID, list.MapNames)
}
