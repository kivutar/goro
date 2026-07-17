package game

import (
	"fmt"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/session"
)

func (m *WorldMode) openSkillTextPrompt(ctx client.Context, skill session.Skill, x, y int, source string) {
	m.pendingSkillText = pendingSkillTextTarget{skill: skill, x: x, y: y, source: source}
	m.ui.skillTextPrompt.Open(ctx, skillTextPromptTitle(skill), "Message", "Message", skillGroundTextMaxBytes)
}

func (m *WorldMode) updateSkillTextPrompt(ctx client.Context) bool {
	consumed := m.ui.skillTextPrompt.Update(ctx)
	action := m.ui.skillTextPrompt.PopAction()
	if action.Submitted {
		m.sendPendingSkillText(ctx, action.Text)
		return true
	}
	if m.pendingSkillText.skill.ID != 0 && !m.ui.skillTextPrompt.IsOpen() {
		glog.Debugf("skill text prompt canceled skill=%d target=%d,%d", m.pendingSkillText.skill.ID, m.pendingSkillText.x, m.pendingSkillText.y)
		m.pendingSkillText = pendingSkillTextTarget{}
	}
	return consumed
}

func (m *WorldMode) sendPendingSkillText(ctx client.Context, text string) {
	pending := m.pendingSkillText
	m.pendingSkillText = pendingSkillTextTarget{}
	if pending.skill.ID == 0 {
		return
	}
	if err := m.skills().SendToGroundWithText(ctx, pending.skill, pending.x, pending.y, text, pending.source); err != nil {
		m.ui.console.AddErrorMessage("%s failed.", skillLabel(pending.skill))
		glog.Warnf("skill text send failed skill=%d target=%d,%d: %v", pending.skill.ID, pending.x, pending.y, err)
		return
	}
	glog.Debugf("skill text sent skill=%d target=%d,%d", pending.skill.ID, pending.x, pending.y)
}

func skillTextPromptTitle(skill session.Skill) string {
	return fmt.Sprintf("%s Message", skillLabel(skill))
}
