package game

import (
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func (m *WorldMode) updateWhisperWindow(ctx client.Context) bool {
	consumed, action := m.ui.whisperWindows.Update(ctx)
	if action.Target != "" && action.Message != "" {
		m.sendWhisperWindowMessage(ctx, action)
		return true
	}
	return consumed
}

func (m *WorldMode) sendWhisperWindowMessage(ctx client.Context, action gameui.WhisperWindowAction) {
	target := strings.TrimSpace(action.Target)
	message := strings.TrimSpace(action.Message)
	if target == "" || message == "" {
		return
	}
	conversation := m.ui.whisperWindows.Find(target)
	if conversation == nil {
		return
	}
	if ctx.Network == nil {
		conversation.AddError(ctx, "send failed: not connected")
		m.ui.console.AddErrorMessage("send failed: not connected")
		return
	}
	if err := ctx.Network.SendWhisper(target, message); err != nil {
		conversation.AddError(ctx, "send failed: "+err.Error())
		m.ui.console.AddErrorMessage("send failed: %s", err)
		glog.Warnf("whisper window send failed target=%q: %v", target, err)
		return
	}
	conversation.AddOutgoing(ctx, message)
	m.ui.console.AddBlueMessage("[ To %s ] : %s", target, message)
}

func (m *WorldMode) addWhisperWindowIncoming(ctx client.Context, whisper network.WhisperMessage) {
	sender := strings.TrimSpace(whisper.Sender)
	message := strings.TrimSpace(whisper.Message)
	if sender == "" || message == "" {
		return
	}
	conversation := m.ui.whisperWindows.Find(sender)
	if (conversation == nil || !conversation.IsOpen()) && !shouldOpenWhisperWindow(ctx.Session, sender) {
		return
	}
	m.ui.whisperWindows.AddIncoming(ctx, sender, message)
}

func shouldOpenWhisperWindow(s *session.Session, sender string) bool {
	settings := session.DefaultWhisperSettings()
	if s != nil && s.Whisper.Configured {
		settings = s.Whisper
	}
	if friendNameInSession(s, sender) {
		return settings.OpenFriends
	}
	return settings.OpenStrangers
}
