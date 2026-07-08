package render

import (
	"sync"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
)

type Key = input.Key

const (
	KeyEnter      = input.KeyEnter
	KeyEscape     = input.KeyEscape
	KeyTab        = input.KeyTab
	KeyArrowUp    = input.KeyArrowUp
	KeyArrowDown  = input.KeyArrowDown
	KeyArrowLeft  = input.KeyArrowLeft
	KeyArrowRight = input.KeyArrowRight
	KeyBackspace  = input.KeyBackspace
	KeyShift      = input.KeyShift
	KeyCtrl       = input.KeyCtrl
	KeyF1         = input.KeyF1
	KeyF2         = input.KeyF2
	KeyF3         = input.KeyF3
	KeyF4         = input.KeyF4
	KeyF5         = input.KeyF5
	KeyF6         = input.KeyF6
	KeyF7         = input.KeyF7
	KeyF8         = input.KeyF8
	KeyF9         = input.KeyF9
)

type MouseButton = input.MouseButton

const (
	MouseButtonLeft  = input.MouseButtonLeft
	MouseButtonRight = input.MouseButtonRight
)

const (
	CursorModeNormal = 0
	CursorModeHidden = 1
)

var cursorState = struct {
	sync.Mutex
	app  *gogpu.App
	mode int
}{
	mode: CursorModeNormal,
}

func setCursorApp(app *gogpu.App) {
	cursorState.Lock()
	cursorState.app = app
	mode := cursorState.mode
	cursorState.Unlock()
	applyCursorMode(app, mode)
}

func SetCursorMode(mode int) {
	cursorState.Lock()
	cursorState.mode = mode
	app := cursorState.app
	cursorState.Unlock()
	applyCursorMode(app, mode)
}

func applyCursorMode(app *gogpu.App, mode int) {
	if app == nil {
		return
	}
	if mode == CursorModeHidden {
		app.SetCursor(gpucontext.CursorNone)
		return
	}
	app.SetCursor(gpucontext.CursorDefault)
}
