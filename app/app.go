package app

import (
	"fmt"
	"log"
	"time"

	gameaudio "github.com/kivutar/goro/audio"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/world"
)

type Game struct {
	cfg      config.Config
	input    *input.State
	resource *res.Manager
	session  *session.Session
	world    *world.World
	network  *network.Client
	audio    *gameaudio.BGM
	modes    *game.Manager
	runtime  *runtimeSettings
	uiApp    client.UIApp
	ui       *gameui.Manager
	started  time.Time
	screenW  int
	screenH  int
	quit     func()
	quitting bool
}

func New(cfg config.Config) (*Game, error) {
	resource, err := res.NewManager(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resource manager: %w", err)
	}
	loadClientUIFont(resource)

	g := &Game{
		cfg:      cfg,
		input:    input.NewState(),
		resource: resource,
		session:  session.New(),
		world:    world.New(),
		network:  network.NewClient(cfg.Packet.ClientDate, cfg.Network.Trace),
		audio:    gameaudio.NewBGM(resource, cfg.Audio.BGM, cfg.Audio.BGMVolume, cfg.Audio.SFXVolume),
		runtime:  newRuntimeSettings(cfg.Window.Fullscreen, cfg.Render.VSync, cfg.Render.FPS),
		ui:       gameui.NewManager(),
		started:  time.Now(),
		screenW:  cfg.Window.Width,
		screenH:  cfg.Window.Height,
	}

	ctx := g.modeContext()
	g.modes = game.NewManager(ctx, game.NewLoginMode())
	return g, nil
}

func (g *Game) Update() error {
	defer g.input.EndFrame()
	g.network.Pump()
	g.modes.UpdateContext(g.modeContext())
	return g.modes.Update()
}

func (g *Game) Draw(screen *render.Image) {
	g.modes.Draw(screen)
}

func (g *Game) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		g.screenW = g.cfg.Window.Width
		g.screenH = g.cfg.Window.Height
		return
	}
	g.screenW = width
	g.screenH = height
}

func (g *Game) InputState() *input.State {
	return g.input
}

func (g *Game) SetQuitFunc(quit func()) {
	g.quit = quit
}

func (g *Game) SetUIApp(uiApp client.UIApp) {
	g.uiApp = uiApp
	if g.ui != nil {
		g.ui.SetUIApp(uiApp)
	}
}

func (g *Game) RequestQuit() {
	if g.quitting {
		return
	}
	g.quitting = true
	if g.network != nil {
		g.network.Close()
	}
	if g.audio != nil {
		g.audio.Stop()
	}
	if g.quit != nil {
		g.quit()
	}
}

func (g *Game) RuntimeFullscreen() bool {
	return g.runtime.Fullscreen()
}

func (g *Game) RuntimeVSync() bool {
	return g.runtime.VSync()
}

func (g *Game) RuntimeFPS() bool {
	return g.runtime.FPS()
}

func loadClientUIFont(resource *res.Manager) {
	regular, err := resource.ReadFileExact("System/Font/SCDream4.otf")
	if err != nil {
		return
	}
	bold, err := resource.ReadFileExact("System/Font/SCDream6.otf")
	if err != nil {
		log.Printf("ui font regular loaded but bold missing: %v", err)
	}
	if err := render.SetUIFont(regular, bold); err != nil {
		log.Printf("ui font load failed: %v", err)
		return
	}
	if len(bold) > 0 {
		log.Printf("ui font loaded path=System/Font/SCDream4.otf bold=System/Font/SCDream6.otf")
	} else {
		log.Printf("ui font loaded path=System/Font/SCDream4.otf")
	}
}

func (g *Game) modeContext() client.Context {
	return client.Context{
		Config:      g.cfg,
		Input:       g.input,
		Resources:   g.resource,
		Session:     g.session,
		World:       g.world,
		Network:     g.network,
		Audio:       g.audio,
		Started:     g.started,
		ScreenW:     g.screenW,
		ScreenH:     g.screenH,
		Runtime:     g.runtime,
		RequestQuit: g.RequestQuit,
		UIApp:       g.uiApp,
		UIManager:   g.ui,
	}
}
