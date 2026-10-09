package client

import (
	"strings"
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
	Config            config.Config
	Input             *input.State
	Resources         *res.Manager
	Session           *session.Session
	World             *world.World
	Network           *network.Client
	Audio             *audio.BGM
	Started           time.Time
	ScreenW           int
	ScreenH           int
	Runtime           RuntimeSettings
	RequestQuit       func()
	RequestScreenshot func() (string, error)
	UIApp             UIApp
	UIManager         UIManager
}

// PlayerHasEffectState reads the server's actor options, not status-icon timers.
func (c Context) PlayerHasEffectState(mask uint32) bool {
	return c.World != nil && c.World.Player.EffectState&mask != 0
}

type UIApp interface {
	SetUIRoot(widget.Widget)
	Frame()
	Invalidate()
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
	Anisotropy() int
	SetAnisotropy(int)
	SmoothSprites() bool
	SetSmoothSprites(bool)
	MSAA() bool
	SetMSAA(bool)
	Bloom() bool
	SetBloom(bool)
}

func (c Context) ScriptPath() string {
	if c.Session != nil && c.Session.ScriptPath != "" {
		return c.Session.ScriptPath
	}
	return strings.TrimSpace(c.Config.Script.Path)
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
