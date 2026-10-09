package app

import (
	"sync/atomic"

	"github.com/kivutar/goro/config"
)

type runtimeSettings struct {
	fullscreen    atomic.Bool
	vsync         atomic.Bool
	fps           atomic.Bool
	anisotropy    atomic.Int32
	smoothSprites atomic.Bool
	msaa          atomic.Bool
}

func newRuntimeSettings(cfg config.Config) *runtimeSettings {
	settings := &runtimeSettings{}
	settings.fullscreen.Store(cfg.Window.Fullscreen)
	settings.vsync.Store(cfg.Render.VSync)
	settings.fps.Store(cfg.Render.FPS)
	settings.SetAnisotropy(cfg.Render.Anisotropy)
	settings.SetSmoothSprites(cfg.Render.SmoothSprites)
	settings.SetMSAA(cfg.Render.MSAA)
	return settings
}

func (s *runtimeSettings) SmoothSprites() bool {
	return s == nil || s.smoothSprites.Load()
}

func (s *runtimeSettings) SetSmoothSprites(value bool) {
	if s != nil {
		s.smoothSprites.Store(value)
	}
}

func (s *runtimeSettings) Anisotropy() int {
	if s == nil {
		return 0
	}
	return int(s.anisotropy.Load())
}

func (s *runtimeSettings) SetAnisotropy(value int) {
	if s != nil {
		s.anisotropy.Store(int32(value))
	}
}

func (s *runtimeSettings) Fullscreen() bool {
	if s == nil {
		return false
	}
	return s.fullscreen.Load()
}

func (s *runtimeSettings) SetFullscreen(value bool) {
	if s != nil {
		s.fullscreen.Store(value)
	}
}

func (s *runtimeSettings) VSync() bool {
	if s == nil {
		return false
	}
	return s.vsync.Load()
}

func (s *runtimeSettings) SetVSync(value bool) {
	if s != nil {
		s.vsync.Store(value)
	}
}

func (s *runtimeSettings) FPS() bool {
	if s == nil {
		return false
	}
	return s.fps.Load()
}

func (s *runtimeSettings) SetFPS(value bool) {
	if s != nil {
		s.fps.Store(value)
	}
}

func (s *runtimeSettings) MSAA() bool {
	return s != nil && s.msaa.Load()
}

func (s *runtimeSettings) SetMSAA(value bool) {
	if s != nil {
		s.msaa.Store(value)
	}
}
