package game

import (
	"fmt"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func newMapErrorMode(ctx client.Context, loadErr error, console gameui.ChatConsole) *LoginMode {
	glog.Errorf("map load failed map=%s: %v", ctx.World.MapName, loadErr)
	if ctx.Network != nil {
		ctx.Network.Close()
		// Discard packets and errors belonging to the failed map connection.
		ctx.Network.DrainPackets()
		ctx.Network.DrainErrors()
	}
	if ctx.Session != nil {
		ctx.Session.Playing = false
		ctx.Session.Storage = session.Storage{}
		ctx.Session.Cart = session.Cart{}
	}
	m := NewLoginMode()
	m.console = console
	m.console.AddErrorMessage("%s", loadErr.Error())
	// Let the player fix their game data or explicitly log in again.
	m.autoAttempted = true
	m.autoCharAttempted = true
	m.mapError = fmt.Sprintf("Cannot load map:\n%s\nCheck your game data.\nPlease log in again.", ctx.World.MapName)
	return m
}
