package client

import (
	"time"

	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/audio"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

type Context struct {
	Config      config.Config
	Input       *input.State
	Resources   *res.Manager
	Session     *session.Session
	World       *world.World
	Network     *network.Client
	Audio       *audio.BGM
	Started     time.Time
	ScreenW     int
	ScreenH     int
	Runtime     RuntimeSettings
	RequestQuit func()
	UIApp       UIApp
	UIManager   UIManager
}

type UIApp interface {
	SetUIRoot(widget.Widget)
	Frame()
	Cursor() widget.CursorType
	HoveredWidget() widget.Widget
}

type UIManager interface {
	AddOverlay(widget.Widget)
	RemoveOverlay(widget.Widget)
	Clear()
}

type RuntimeSettings interface {
	Fullscreen() bool
	SetFullscreen(bool)
	VSync() bool
	SetVSync(bool)
	FPS() bool
	SetFPS(bool)
}

func (c Context) ScreenSize() (int, int) {
	width, height := c.ScreenW, c.ScreenH
	if width <= 0 {
		width = c.Config.Window.Width
	}
	if height <= 0 {
		height = c.Config.Window.Height
	}
	return width, height
}
