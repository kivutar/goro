package game

import (
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func (m *WorldMode) handleQuestPacket(ctx client.Context, pkt network.Packet) bool {
	var err error
	switch pkt.ID {
	case network.PacketZCQuestList:
		var states []network.QuestState
		states, _, err = network.ParseQuestList(pkt)
		if err == nil {
			entries := make([]session.Quest, len(states))
			for i, state := range states {
				entries[i] = session.Quest{ID: state.ID, Active: state.Active}
			}
			ctx.Session.Quests.Replace(entries)
		}
	case network.PacketZCQuestMissions:
		var missions []network.QuestMission
		missions, _, err = network.ParseQuestMissions(pkt)
		if err == nil {
			for _, mission := range missions {
				ctx.Session.Quests.SetMission(mission.ID, mission.ExpiresAt, questObjectives(mission.Objectives))
			}
		}
	case network.PacketZCQuestAdd:
		var quest network.QuestAdd
		quest, _, err = network.ParseQuestAdd(pkt)
		if err == nil {
			ctx.Session.Quests.Set(session.Quest{ID: quest.ID, Active: quest.Active, ExpiresAt: quest.ExpiresAt, Objectives: questObjectives(quest.Objectives)})
		}
	case network.PacketZCQuestDelete:
		var id uint32
		id, _, err = network.ParseQuestDelete(pkt)
		if err == nil {
			ctx.Session.Quests.Remove(id)
		}
	case network.PacketZCQuestHunt:
		var hunts []network.QuestHunt
		hunts, _, err = network.ParseQuestHunt(pkt)
		if err == nil {
			for _, hunt := range hunts {
				ctx.Session.Quests.UpdateHunt(hunt.QuestID, hunt.MonsterID, hunt.Current, hunt.Required)
			}
		}
	case network.PacketZCQuestActive:
		var state network.QuestState
		state, _, err = network.ParseQuestActive(pkt)
		if err == nil {
			ctx.Session.Quests.SetActive(state.ID, state.Active)
		}
	default:
		return false
	}
	if err != nil {
		glog.Errorf("parse quest packet 0x%04X: %v", pkt.ID, err)
	}
	return true
}

func questObjectives(objectives []network.QuestObjective) []session.QuestObjective {
	result := make([]session.QuestObjective, len(objectives))
	for i, objective := range objectives {
		result[i] = session.QuestObjective{MonsterID: objective.MonsterID, Name: objective.Name, Current: objective.Current}
	}
	return result
}

func (m *WorldMode) toggleQuestWindowFromInput(ctx client.Context) bool {
	if !plainAltDown(ctx.Input) || m.ui.nonConsoleKeyboardInputBlocked() || !ctx.Input.KeyCodeJustPressed(gpucontext.KeyU) {
		return false
	}
	ctx.Input.ConsumeKeyCodePress(gpucontext.KeyU)
	m.ui.questWindow.Toggle(ctx)
	return true
}

func (m *WorldMode) setQuestActive(ctx client.Context, id uint32, active bool) {
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Quest update failed: not connected.")
		return
	}
	if err := ctx.Network.SendQuestActive(id, active); err != nil {
		m.ui.console.AddErrorMessage("Quest update failed.")
		glog.Warnf("quest update failed: %v", err)
	}
	// The journal only changes once ZC_ACTIVE_QUEST acknowledges the request.
}
