package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func (m *LoginMode) updateLoginServerWindow(ctx client.Context) {
	m.updateServiceWindow(ctx, "Server", loginServerNames(loginConnections(ctx)), m.selectedLoginServer, gameui.ServiceWindowCallbacks{
		OnSelect: func(index int) {
			m.selectLoginServer(ctx, index)
		},
		OnCancel: func() {
			m.openQuitConfirm(ctx)
		},
	})
}

func (m *LoginMode) updateCharacterServiceWindow(ctx client.Context) {
	selected := 0
	servers := []session.CharServer(nil)
	if ctx.Session != nil {
		selected = ctx.Session.CharServerIndex
		servers = ctx.Session.CharServers
	}
	m.updateServiceWindow(ctx, "Service", characterServiceNames(ctx.Resources, servers), selected, gameui.ServiceWindowCallbacks{
		OnSelect: func(index int) {
			m.selectCharacterService(ctx, index, true)
		},
		OnCancel: func() {
			m.cancelCharacterServiceSelection(ctx)
		},
	})
}

func (m *LoginMode) updateServiceWindow(ctx client.Context, title string, names []string, selected int, callbacks gameui.ServiceWindowCallbacks) {
	if ctx.Config.Headless {
		return
	}
	if m.serviceWindow == nil {
		m.serviceWindow = gameui.NewServiceWindow(ctx, names, gameui.ServiceWindowOptions{
			Title:    title,
			Selected: selected,
		}, callbacks)
		m.serviceWindow.Publish(ctx)
		return
	}
	m.serviceWindow.SetContext(ctx)
	m.serviceWindow.Publish(ctx)
}

func (m *LoginMode) selectLoginServer(ctx client.Context, index int) {
	connections := loginConnections(ctx)
	if index < 0 || index >= len(connections) {
		return
	}
	m.selectedLoginServer = index
	m.hideServiceWindow(ctx)
	m.accountStep = loginAccountCredentials
	m.status = "enter account credentials"
	m.updateLoginWindow(ctx)
}

func (m *LoginMode) showLoginServerSelection(ctx client.Context) {
	if len(loginConnections(ctx)) <= 1 {
		return
	}
	m.hideLoginWindow(ctx)
	m.hideServiceWindow(ctx)
	m.accountStep = loginAccountConnection
	m.status = "select a server"
	m.updateLoginServerWindow(ctx)
}

func (m *LoginMode) applyAccountAcceptLogin(ctx client.Context, login network.AccountAcceptLogin) {
	m.loginPending = false
	ctx.Session.AccountID = login.AccountID
	ctx.Session.AuthCode = login.AuthCode
	ctx.Session.UserLevel = login.UserLevel
	ctx.Session.Sex = login.Sex
	ctx.Session.CharServers = convertCharServers(login.CharServer)
	ctx.Session.CharServerIndex = 0
	m.status = fmt.Sprintf("account accepted: aid=%d char_servers=%d", login.AccountID, len(login.CharServer))
	glog.Infof("account accepted aid=%d sex=%d admin=%t char_servers=%d", login.AccountID, login.Sex, ctx.Session.IsAdminID(login.AccountID), len(login.CharServer))
	for _, server := range login.CharServer {
		m.packets = append(m.packets, fmt.Sprintf("char %s %s:%d users=%d", server.Name, server.Address, server.Port, server.UserCount))
	}
	m.enableLoginServerPing(time.Now())
	if ctx.Config.Login.AutoLogin && len(login.CharServer) > 0 {
		m.selectCharacterService(ctx, 0, false)
		return
	}
	m.showCharacterServiceSelection(ctx)
}

func (m *LoginMode) showCharacterServiceSelection(ctx client.Context) {
	m.hideLoginWindow(ctx)
	m.hideServiceWindow(ctx)
	m.accountStep = loginAccountCharacterService
	m.status = "select a character server"
	m.updateCharacterServiceWindow(ctx)
}

func (m *LoginMode) selectCharacterService(ctx client.Context, index int, userConfirmed bool) {
	server, ok := selectedCharacterServer(ctx.Session, index)
	if !ok {
		return
	}
	ctx.Session.CharServerIndex = index
	m.hideLoginWindow(ctx)
	m.hideServiceWindow(ctx)
	m.accountStep = loginAccountCharacterConnecting
	if userConfirmed {
		m.playConfirmSFX(ctx)
	}
	if !m.connectCharServer(ctx, server) {
		m.accountStep = loginAccountCharacterService
	}
}

func (m *LoginMode) cancelCharacterServiceSelection(ctx client.Context) {
	m.disableLoginServerPing()
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	m.hideServiceWindow(ctx)
	m.accountStep = loginAccountCredentials
	m.status = "character server selection cancelled"
	m.updateLoginWindow(ctx)
}

func (m *LoginMode) hideLoginWindow(ctx client.Context) {
	if m.loginWindow == nil {
		return
	}
	m.username = m.loginWindow.Username
	m.password = m.loginWindow.Password
	m.loginWindow.Unpublish(ctx)
	m.loginWindow = nil
}

func (m *LoginMode) hideServiceWindow(ctx client.Context) {
	if m.serviceWindow == nil {
		return
	}
	m.serviceWindow.Unpublish(ctx)
	m.serviceWindow = nil
}

func loginServerNames(connections []res.Connection) []string {
	names := make([]string, len(connections))
	for i, connection := range connections {
		name := strings.TrimSpace(connection.Display)
		if name == "" {
			name = fmt.Sprintf("%s:%d", connection.Address, connection.Port)
		}
		names[i] = name
	}
	return names
}

func characterServiceNames(manager *res.Manager, servers []session.CharServer) []string {
	names := make([]string, len(servers))
	for i, server := range servers {
		names[i] = characterServiceName(manager, server)
	}
	return names
}

func characterServiceName(manager *res.Manager, server session.CharServer) string {
	name := strings.TrimSpace(server.Name)
	if name == "" {
		name = fmt.Sprintf("%s:%d", server.Address, server.Port)
	}
	if server.Property != 0 {
		prefix := loginMessage(manager, 482, "New")
		name = strings.TrimSpace(prefix + " " + name)
	}
	if server.State != 0 {
		return strings.TrimSpace(name + " " + loginMessage(manager, 484, "(On maintenance)"))
	}
	count := fmt.Sprintf("(%d)", server.UserCount)
	if pattern := loginMessage(manager, 483, ""); strings.Contains(pattern, "%d") {
		count = strings.Replace(pattern, "%d", fmt.Sprintf("%d", server.UserCount), 1)
	}
	return strings.TrimSpace(name + " " + count)
}

func loginMessage(manager *res.Manager, id int, fallback string) string {
	if manager != nil {
		if message, ok := manager.MsgString(id); ok && strings.TrimSpace(message) != "" {
			return strings.TrimSpace(message)
		}
	}
	return fallback
}

func selectedCharacterServer(state *session.Session, index int) (network.CharServer, bool) {
	if state == nil || index < 0 || index >= len(state.CharServers) {
		return network.CharServer{}, false
	}
	server := state.CharServers[index]
	return network.CharServer{
		Address:   server.Address,
		Port:      server.Port,
		Name:      server.Name,
		UserCount: server.UserCount,
		State:     server.State,
		Property:  server.Property,
	}, true
}
