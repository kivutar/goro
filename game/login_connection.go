package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
)

func (m *LoginMode) showConnectionFailed(ctx client.Context) {
	glog.Warnf("login connection failed: %s", m.status)
	m.loginPending = false
	m.autoAttempted = true
	m.disableLoginServerPing()
	m.disableCharServerPing()
	// A failed character/map connection must not leave the error covered by
	// a transition or hand off to the world without a connection.
	m.fade = loginFadeState{}
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	if m.disconnectDialog.IsOpen() {
		return
	}
	m.status = disconnectMessageText(ctx.Resources, disconnectMessage{1, "Failed to Connect to Server."})
	m.disconnectDialog.OpenAlert(ctx, "Connection failed", m.status, func() {
		m.phase = loginPhaseAccount
		m.accountStep = loginAccountCredentials
		m.autoCharAttempted = false
		m.status = "enter account credentials"
		m.publishPhaseWindow(ctx)
	})
}
