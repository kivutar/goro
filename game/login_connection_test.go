package game

import (
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestLoginConnectionFailureStaysVisibleAndAllowsRetry(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().(*net.TCPAddr)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	connection := res.Connection{Address: address.IP.String(), Port: address.Port}
	for _, stage := range []string{"login", "character", "map"} {
		t.Run(stage, func(t *testing.T) {
			state := input.NewState()
			netClient := network.NewClient(20080910, false)
			defer netClient.Close()
			quit := false
			ctx := client.Context{
				Input: state, Network: netClient, Session: &session.Session{},
				Resources: loginTestResources(connection),
				UIManager: &loginTestUIManager{}, ScreenW: 1280, ScreenH: 720,
				RequestQuit: func() { quit = true },
			}
			mode := NewLoginMode()
			mode.username, mode.password = "tester", "test-password"
			ctx.Config.Login.AutoLogin = true
			switch stage {
			case "login":
				mode.accountStep = loginAccountCredentials
				mode.connectAndMaybeLogin(ctx, connection, true)
			case "character":
				mode.accountStep = loginAccountCharacterConnecting
				mode.connectCharServer(ctx, network.CharServer{Address: connection.Address, Port: uint16(connection.Port)})
			case "map":
				mode.phase = loginPhaseCharacter
				mode.fade = loginFadeState{phase: loginFadeHold, started: time.Now()}
				mode.connectMapServer(ctx, network.ZoneServerNotify{Address: connection.Address, Port: uint16(connection.Port)})
			}
			// The UI submit callback runs before the same key event reaches
			// input.State. Its newly opened dialog must not accept this Enter.
			state.SetKey(input.KeyEnter, true)
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if quit || !mode.disconnectDialog.IsOpen() {
				t.Fatal("connection error was acknowledged by the login key press")
			}
			if mode.fade.phase != loginFadeNone {
				t.Fatal("connection error is hidden behind a login transition")
			}
			state.EndFrame()
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if quit || !mode.disconnectDialog.IsOpen() {
				t.Fatal("held login key dismissed the connection error")
			}
			state.EndFrame()
			state.SetKey(input.KeyEnter, false)
			state.SetKey(input.KeyEnter, true)
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if quit || mode.disconnectDialog.IsOpen() {
				t.Fatal("acknowledging the error did not stay in the client")
			}
			if mode.phase != loginPhaseAccount || mode.accountStep != loginAccountCredentials || mode.loginWindow == nil {
				t.Fatal("connection failure did not restore the login form")
			}
			if mode.loginPending || mode.loginPingActive || mode.charPingActive {
				t.Fatal("failed connection retained login activity")
			}
			if mode.loginWindow.Username != "tester" || mode.loginWindow.Password != "test-password" {
				t.Fatal("connection failure discarded credentials")
			}
			state.EndFrame()
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if mode.disconnectDialog.IsOpen() || mode.loginPending {
				t.Fatal("autologin retried the failed connection without confirmation")
			}

			// A subsequent explicit retry must still send the login packet.
			retryListener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			defer retryListener.Close()
			retry := res.Connection{Address: "127.0.0.1", Port: retryListener.Addr().(*net.TCPAddr).Port}
			mode.connectAndMaybeLogin(ctx, retry, true)
			if err := retryListener.SetDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			serverConn, err := retryListener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			defer serverConn.Close()
			readBotTestPackets(t, serverConn, network.BuildAccountLoginPacket(network.AccountLogin{
				Username: "tester", Password: "test-password",
			}))
			if !mode.loginPending || mode.disconnectDialog.IsOpen() || quit {
				t.Fatal("explicit retry did not start a new login attempt")
			}
		})
	}
}
