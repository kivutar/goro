//go:build android

// This file is overlaid into gogpu/internal/platform by build.sh.
package platform

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/gogpu/gpucontext"
)

// The host retains the ANativeWindow until Run has returned and all GPU
// resources have been released. Input is queued from Android's UI thread.
var androidHost struct {
	sync.Mutex
	window *androidWindow
	events []Event
}

var androidWake = make(chan struct{}, 1)

func AndroidSetWindow(handle uintptr, width, height int) {
	androidHost.Lock()
	defer androidHost.Unlock()
	androidHost.window = &androidWindow{id: NewWindowID(), handle: handle, width: width, height: height}
	androidHost.events = nil
}

func AndroidEvent(ev Event) {
	androidHost.Lock()
	if w := androidHost.window; w != nil {
		ev.WindowID = w.id
		n := len(androidHost.events)
		if ev.Type == EventPointerMove && n > 0 && androidHost.events[n-1].Type == EventPointerMove {
			androidHost.events[n-1] = ev
		} else {
			androidHost.events = append(androidHost.events, ev)
		}
	}
	androidHost.Unlock()
	select {
	case androidWake <- struct{}{}:
	default:
	}
}

type androidPlatform struct{}

func newPlatformManager() PlatformManager { return &androidPlatform{} }
func (*androidPlatform) Init() error      { return nil }
func (*androidPlatform) CreateWindow(Config) (PlatformWindow, error) {
	androidHost.Lock()
	defer androidHost.Unlock()
	w := androidHost.window
	if w == nil || w.handle == 0 {
		return nil, errors.New("Android surface is not ready")
	}
	return w, nil
}
func (*androidPlatform) PollEvents() Event {
	androidHost.Lock()
	defer androidHost.Unlock()
	if len(androidHost.events) == 0 {
		return Event{}
	}
	ev := androidHost.events[0]
	androidHost.events[0] = Event{}
	androidHost.events = androidHost.events[1:]
	return ev
}
func (*androidPlatform) WaitEvents() { <-androidWake }
func (*androidPlatform) WakeUp() {
	select {
	case androidWake <- struct{}{}:
	default:
	}
}
func (*androidPlatform) ClipboardRead() (string, error) {
	return "", errors.New("Android clipboard is unavailable")
}
func (*androidPlatform) ClipboardWrite(string) error {
	return errors.New("Android clipboard is unavailable")
}
func (*androidPlatform) DarkMode() bool                            { return false }
func (*androidPlatform) ReduceMotion() bool                        { return false }
func (*androidPlatform) HighContrast() bool                        { return false }
func (*androidPlatform) FontScale() float32                        { return 1 }
func (*androidPlatform) SubpixelLayout() gpucontext.SubpixelLayout { return gpucontext.SubpixelNone }
func (*androidPlatform) FontSmoothing() gpucontext.FontSmoothing {
	return gpucontext.FontSmoothingGrayscale
}
func (*androidPlatform) SetAppName(string) {}
func (*androidPlatform) ShowOpenFileDialog(FileDialogOptions) ([]string, error) {
	return nil, errors.New("Android file dialogs are unavailable")
}
func (*androidPlatform) ShowSaveFileDialog(FileDialogOptions) (string, error) {
	return "", errors.New("Android file dialogs are unavailable")
}
func (*androidPlatform) Destroy() {
	androidHost.Lock()
	androidHost.window = nil
	androidHost.events = nil
	androidHost.Unlock()
}

type androidWindow struct {
	id            WindowID
	handle        uintptr
	width, height int
	closed        atomic.Bool
}

func (w *androidWindow) ID() WindowID                  { return w.id }
func (w *androidWindow) GetHandle() (uintptr, uintptr) { return 0, w.handle }
func (w *androidWindow) LogicalSize() (int, int)       { return w.width, w.height }
func (w *androidWindow) PhysicalSize() (int, int)      { return w.width, w.height }
func (*androidWindow) ScaleFactor() float64            { return 1 }
func (w *androidWindow) PrepareFrame() PrepareFrameResult {
	return PrepareFrameResult{ScaleFactor: 1, PhysicalWidth: uint32(w.width), PhysicalHeight: uint32(w.height)}
}
func (*androidWindow) InSizeMove() bool                                                   { return false }
func (w *androidWindow) ShouldClose() bool                                                { return w.closed.Load() }
func (*androidWindow) SetTitle(string)                                                    {}
func (*androidWindow) SetMinSize(int, int)                                                {}
func (*androidWindow) SetMaxSize(int, int)                                                {}
func (*androidWindow) RequestSize(int, int)                                               {}
func (*androidWindow) SetCursor(int)                                                      {}
func (*androidWindow) SetFrameless(bool)                                                  {}
func (*androidWindow) IsFrameless() bool                                                  { return true }
func (*androidWindow) SetFullscreen(bool)                                                 {}
func (*androidWindow) IsFullscreen() bool                                                 { return true }
func (*androidWindow) SetHitTestCallback(func(float64, float64) gpucontext.HitTestResult) {}
func (*androidWindow) Minimize()                                                          {}
func (*androidWindow) Maximize()                                                          {}
func (*androidWindow) IsMaximized() bool                                                  { return true }
func (w *androidWindow) Close()                                                           { w.closed.Store(true) }
func (*androidWindow) Show()                                                              {}
func (*androidWindow) Hide()                                                              {}
func (*androidWindow) SetPosition(int, int)                                               {}
func (*androidWindow) SyncFrame()                                                         {}
func (*androidWindow) SetCursorMode(int)                                                  {}
func (*androidWindow) CursorMode() int                                                    { return 0 }
func (*androidWindow) SetModalFrameCallback(func())                                       {}
func (*androidWindow) StartDrag(_ []string, done func(DragResult)) {
	if done != nil {
		done(DragCancelled)
	}
}
func (*androidWindow) Destroy() {}

var _ PlatformManager = (*androidPlatform)(nil)
var _ PlatformWindow = (*androidWindow)(nil)
