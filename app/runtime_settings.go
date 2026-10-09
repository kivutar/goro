package app

import "sync/atomic"

type runtimeSettings struct {
	fullscreen atomic.Bool
	vsync      atomic.Bool
	fps        atomic.Bool
	anisotropy atomic.Int32
}

func newRuntimeSettings(fullscreen, vsync, fps bool, anisotropy int) *runtimeSettings {
	settings := &runtimeSettings{}
	settings.fullscreen.Store(fullscreen)
	settings.vsync.Store(vsync)
	settings.fps.Store(fps)
	settings.SetAnisotropy(anisotropy)
	return settings
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
