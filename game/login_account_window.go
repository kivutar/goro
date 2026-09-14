package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
)

func (m *LoginMode) updateAccountInput(ctx client.Context) {
	m.updateAccountWindow(ctx)
	if m.serviceWindow != nil {
		window := m.serviceWindow
		window.Update(ctx)
		if m.serviceWindow == window {
			window.Publish(ctx)
		}
		return
	}
	if m.loginWindow != nil {
		m.loginWindow.Update(ctx)
		m.loginWindow.Publish(ctx)
	}
}

func (m *LoginMode) updateAccountWindow(ctx client.Context) {
	switch m.accountStep {
	case loginAccountConnection:
		if len(loginConnections(ctx)) > 1 && !ctx.Config.Login.AutoLogin {
			m.updateLoginServerWindow(ctx)
			return
		}
	case loginAccountCharacterService:
		m.updateCharacterServiceWindow(ctx)
		return
	case loginAccountCharacterConnecting:
		return
	}
	if m.accountStep == loginAccountConnection {
		if len(loginConnections(ctx)) == 0 {
			m.status = "no login servers discovered"
		} else {
			m.status = "enter account credentials"
		}
	}
	m.accountStep = loginAccountCredentials
	m.updateLoginWindow(ctx)
}

func (m *LoginMode) updateLoginWindow(ctx client.Context) {
	if ctx.Config.Headless {
		return
	}
	if m.loginWindow == nil {
		m.loginWindow = gameui.NewLoginWindow(ctx, m.username, m.password, m.keepID, gameui.LoginWindowCallbacks{
			OnSubmit: func() {
				m.username = m.loginWindow.Username
				m.password = m.loginWindow.Password
				m.keepID = m.loginWindow.KeepID
				m.saveLoginID(ctx)
				if conn, ok := m.selectedLoginConnection(ctx); ok {
					m.connectAndMaybeLogin(ctx, conn, true)
				}
			},
		})
		m.loginWindow.Publish(ctx)
		return
	}
	m.loginWindow.SetContext(ctx)
	m.username = m.loginWindow.Username
	m.password = m.loginWindow.Password
	m.loginWindow.Publish(ctx)
}

func (m *LoginMode) saveLoginID(ctx client.Context) {
	if ctx.Session != nil {
		ctx.Session.KeepLoginID = m.keepID
		ctx.Session.SavedUsername = ""
		if m.keepID {
			ctx.Session.SavedUsername = m.username
		}
	}
	if _, err := config.SaveLoginID(m.username, m.keepID); err != nil {
		glog.Warnf("login ID save failed: %v", err)
	}
}

func (m *LoginMode) selectedLoginConnection(ctx client.Context) (res.Connection, bool) {
	connections := loginConnections(ctx)
	if len(connections) == 0 {
		return res.Connection{}, false
	}
	if m.selectedLoginServer < 0 || m.selectedLoginServer >= len(connections) {
		m.selectedLoginServer = 0
	}
	return connections[m.selectedLoginServer], true
}

func loginConnections(ctx client.Context) []res.Connection {
	if ctx.Resources == nil {
		return nil
	}
	return ctx.Resources.ClientInfo.Connections
}
