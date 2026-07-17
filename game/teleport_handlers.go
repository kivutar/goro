package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func (m *WorldMode) applyWarpPointList(ctx client.Context, list network.WarpPointList) {
	if list.SkillID != gameui.TeleportSkillID && list.SkillID != gameui.WarpPortalSkillID {
		glog.Debugf("warp point list ignored skill=%d maps=%v", list.SkillID, list.MapNames)
		return
	}
	skill, ok := skillByID(ctx.Session, list.SkillID)
	if !ok {
		skill = session.Skill{ID: list.SkillID, Level: 1}
		if len(list.MapNames) > 1 {
			skill.Level = 2
		}
	}
	if gameui.TeleportWarpListBypassesModal(skill, list) {
		m.autoSelectTeleportRandom(ctx, list)
		return
	}
	m.ui.teleportModal.OpenWarpPointList(list, skill)
	glog.Debugf("warp point destination list skill=%d maps=%v", list.SkillID, list.MapNames)
}

func (m *WorldMode) applyRememberWarpPointAck(_ client.Context, ack network.RememberWarpPointAck) {
	switch ack.Result {
	case 0:
		m.ui.console.AddBlueMessage("Saved location as a Memo Point for Warp skill.")
	case 1:
		m.ui.console.AddErrorMessage("Skill Level is not high enough.")
	case 2:
		m.ui.console.AddErrorMessage("You haven't learned Warp.")
	default:
		m.ui.console.AddErrorMessage("Memo failed.")
	}
	glog.Debugf("remember warp point ack result=%d", ack.Result)
}

func (m *WorldMode) autoSelectTeleportRandom(ctx client.Context, list network.WarpPointList) {
	mapName := gameui.TeleportRandomMap
	for _, name := range list.MapNames {
		if name == gameui.TeleportRandomMap {
			mapName = name
			break
		}
	}
	if ctx.Network == nil {
		return
	}
	if err := ctx.Network.SendSelectWarpPoint(list.SkillID, mapName); err != nil {
		return
	}
	m.ui.teleportModal.Reset()
	glog.Debugf("teleport random selected automatically skill=%d maps=%v", list.SkillID, list.MapNames)
}
