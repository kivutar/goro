package render

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/input"
)

type headlessTestGame struct {
	t       *testing.T
	quit    func()
	events  []string
	frames  int
	err     error
	texture *Image
}

func (g *headlessTestGame) Resize(w, h int) {
	if w != 800 || h != 600 {
		g.t.Fatalf("size = %dx%d", w, h)
	}
}
func (g *headlessTestGame) InputState() *input.State { return nil }
func (g *headlessTestGame) SetQuitFunc(quit func())  { g.quit = quit }
func (g *headlessTestGame) Update() error {
	g.events = append(g.events, "update")
	return g.err
}
func (g *headlessTestGame) Draw(f *Frame) {
	if len(f.commands) != 0 {
		g.t.Fatal("previous frame commands were retained")
	}
	f.DrawImage(g.texture, nil)
	g.events = append(g.events, "draw")
}
func (g *headlessTestGame) DrawUIOverlay(*Frame) {
	g.events = append(g.events, "ui overlay")
}
func (g *headlessTestGame) DrawOverlay(*Frame) {
	g.events = append(g.events, "overlay")
}
func (g *headlessTestGame) FrameSubmitted() {
	g.events = append(g.events, "submitted")
	g.frames++
	if g.frames == 3 {
		g.quit()
	}
}

func TestHeadlessFrameLifecycle(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	g := &headlessTestGame{t: t, texture: NewImage(1, 1)}
	if err := RunHeadless(ctx, g, config.WindowConfig{Width: 800, Height: 600}); err != nil {
		t.Fatal(err)
	}
	var want []string
	for i := 0; i < 3; i++ {
		want = append(want, "update", "draw", "ui overlay", "overlay", "submitted")
	}
	if !reflect.DeepEqual(g.events, want) {
		t.Fatalf("frame lifecycle = %v, want %v", g.events, want)
	}
}

func TestHeadlessStopsOnUpdateError(t *testing.T) {
	want := errors.New("update failed")
	g := &headlessTestGame{t: t, err: want}
	if err := RunHeadless(context.Background(), g, config.WindowConfig{Width: 800, Height: 600}); err != want {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if !reflect.DeepEqual(g.events, []string{"update"}) {
		t.Fatalf("drew a frame after an update error: %v", g.events)
	}
}
