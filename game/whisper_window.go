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
	consumed := m.ui.whisperWindow.Update(ctx)
	if action := m.ui.whisperWindow.PopAction(); action.Target != "" && action.Message != "" {
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
	if ctx.Network == nil {
		m.ui.whisperWindow.AddError(ctx, "send failed: not connected")
		m.ui.console.AddErrorMessage("send failed: not connected")
		return
	}
	if err := ctx.Network.SendWhisper(target, message); err != nil {
		m.ui.whisperWindow.AddError(ctx, "send failed: "+err.Error())
		m.ui.console.AddErrorMessage("send failed: %s", err)
		glog.Warnf("whisper window send failed target=%q: %v", target, err)
		return
	}
	m.ui.whisperWindow.AddOutgoing(ctx, message)
	m.ui.console.AddBlueMessage("[ To %s ] : %s", target, message)
}

func (m *WorldMode) addWhisperWindowIncoming(ctx client.Context, whisper network.WhisperMessage) {
	sender := strings.TrimSpace(whisper.Sender)
	message := strings.TrimSpace(whisper.Message)
	if sender == "" || message == "" {
		return
	}
	if !m.ui.whisperWindow.IsOpen() && !shouldOpenWhisperWindow(ctx.Session, sender) {
		return
	}
	m.ui.whisperWindow.Open(ctx, sender)
	m.ui.whisperWindow.AddIncoming(ctx, sender, message)
}

func (m *WorldMode) addWhisperWindowAck(ctx client.Context, ack network.WhisperAck) {
	if ack.Result == 0 || !m.ui.whisperWindow.IsOpen() {
		return
	}
	m.ui.whisperWindow.AddError(ctx, whisperAckMessage(ctx.Resources, ack))
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
