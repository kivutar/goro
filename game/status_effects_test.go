package game

import (
	"slices"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestStatusIconsRemainRemovableAfterMapTransitions(t *testing.T) {
	for _, tt := range []struct {
		name             string
		updateBeforeStop bool
	}{
		{"status_ends_after_first_update", true},
		{"status_ends_before_first_update", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manager := &worldModeTestUIManager{}
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   worldstate.New(), UIManager: manager, ScreenW: 800, ScreenH: 600,
			}
			now := time.Now()
			mode := NewWorldMode()
			mode.handleNetworkPacket(ctx, testStatusEffectChangePacket(db.StatusIncAgi, ctx.Session.AccountID, true), now)
			mode.ui.statusIcons.Update(ctx, now)
			if len(manager.overlays) != 1 {
				t.Fatalf("status overlays = %d, want 1", len(manager.overlays))
			}
			original := manager.overlays[0]
			for range 3 {
				mode = mode.nextWorldMode()
				if tt.updateBeforeStop {
					mode.ui.statusIcons.Update(ctx, now)
					if len(manager.overlays) != 1 || manager.overlays[0] != original {
						t.Errorf("map transition left %d overlays, want the original status overlay only", len(manager.overlays))
					}
				}
			}

			mode.handleNetworkPacket(ctx, testStatusEffectChangePacket(db.StatusIncAgi, ctx.Session.AccountID, false), now)
			if _, active := ctx.Session.Statuses.Active[db.StatusIncAgi]; active {
				t.Fatal("server stop packet did not remove Increase Agility")
			}
			mode.ui.statusIcons.Update(ctx, now)
			if len(manager.overlays) != 0 {
				t.Fatalf("status ended but %d overlays remain", len(manager.overlays))
			}
		})
	}
}

func TestStatusIconsRefreshWhileWindowConsumesInput(t *testing.T) {
	for _, modal := range []bool{false, true} {
		name := "hovered_window"
		if modal {
			name = "escape_menu"
		}
		t.Run(name, func(t *testing.T) {
			manager := &worldModeTestUIManager{}
			netClient := network.NewClient(20080910, false)
			defer netClient.Close()
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   worldstate.New(), Input: input.NewState(), Network: netClient,
				UIManager: manager, ScreenW: 800, ScreenH: 600,
			}
			mode := NewWorldMode()
			mode.handleNetworkPacket(ctx, testStatusEffectChangePacket(db.StatusIncAgi, ctx.Session.AccountID, true), time.Now())
			mode.ui.statusIcons.Update(ctx, time.Now())
			if len(manager.overlays) != 1 {
				t.Fatalf("status overlays = %d, want 1", len(manager.overlays))
			}
			iconOverlay := manager.overlays[0]
			if modal {
				mode.ui.escapeMenu.Toggle(ctx)
			} else {
				mode.ui.statsWindow.OpenWindow(ctx)
				mode.ui.statsWindow.SetAutoPosition(100, 200)
				ctx.Input.SetMousePosition(110, 210)
			}

			mode.handleNetworkPacket(ctx, testStatusEffectChangePacket(db.StatusIncAgi, ctx.Session.AccountID, false), time.Now())
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if slices.Contains(manager.overlays, iconOverlay) {
				t.Fatal("status overlay remained visible while the window consumed input")
			}
		})
	}
}
