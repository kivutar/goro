package game

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestLoginSubmissionRemembersAndForgetsID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	ctx := client.Context{Session: session.New(), Resources: &res.Manager{}, ScreenW: 800, ScreenH: 600}
	mode := NewLoginMode()
	mode.username, mode.password, mode.keepID = "remembered-id", "not-persisted", true
	mode.updateLoginWindow(ctx)
	submit := func() {
		t.Helper()
		if !mode.loginWindow.Widget().Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone)) {
			t.Fatal("login field did not handle Enter")
		}
	}
	submit()
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Login.KeepID || cfg.Login.SavedUsername != "remembered-id" || cfg.Login.Password != "" {
		t.Fatalf("persisted login = %+v", cfg.Login)
	}
	// Returning to login uses the latest preference, even though Config is the startup snapshot.
	returned := NewLoginMode()
	returned.Enter(ctx)
	if returned.username != "remembered-id" || !returned.keepID || returned.password != "" {
		t.Fatal("returning to login lost the saved ID or retained the password")
	}
	// A fresh process restores from disk; an explicit username still takes precedence.
	for _, explicit := range []string{"", "configured-id"} {
		cfg.Login.Username = explicit
		restarted := NewLoginMode()
		restarted.Enter(client.Context{Config: cfg, Resources: ctx.Resources})
		want := "remembered-id"
		if explicit != "" {
			want = explicit
		}
		if restarted.username != want || !restarted.keepID || restarted.password != "" {
			t.Fatal("restarting did not restore the saved ID or honor explicit credentials")
		}
	}
	mode.loginWindow.KeepID = false
	submit()
	cfg, err = config.LoadConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Login.KeepID || cfg.Login.SavedUsername != "" {
		t.Fatal("submitting with Keep off retained the saved ID on disk")
	}
	returned = NewLoginMode()
	returned.Enter(ctx)
	if returned.username != "" || returned.keepID || ctx.Session.SavedUsername != "" {
		t.Fatal("submitting with Keep off retained the saved ID in the session")
	}
}
