//go:build android && cgo

// Goro's Android shared library. Java owns the Activity and ANativeWindow;
// Go owns the game and GPU loop. Stop waits for GPU teardown before Java
// returns its Surface to Android.
package main

/*
#cgo LDFLAGS: -landroid -llog
#include <stdlib.h>
#include <stdint.h>
*/
import "C"

import (
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/app"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
)

var host struct {
	sync.Mutex
	done   chan struct{}
	status string
}
var epoch = time.Now()

//export GoroStart
func GoroStart(window C.uintptr_t, width, height C.int, directory *C.char) {
	host.Lock()
	defer host.Unlock()
	if host.done != nil {
		return
	}
	dir := C.GoString(directory)
	host.status = ""
	done := make(chan struct{})
	host.done = done
	gogpu.AndroidSetWindow(uintptr(window), int(width), int(height))
	go func() {
		defer close(done)
		if err := run(dir, int(width), int(height)); err != nil {
			glog.Errorf("Android: %v", err)
			host.Lock()
			host.status = err.Error()
			host.Unlock()
		}
	}()
}

func run(dir string, width, height int) error {
	// Config, screenshots and resource paths must all be writable app storage.
	if err := os.Chdir(dir); err != nil {
		return err
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		return err
	}
	cfg, err := config.LoadConfig([]string{"--data-dir", dir, "--width", strconv.Itoa(width), "--height", strconv.Itoa(height), "--graphics-api", "vulkan", "--fullscreen"})
	if err != nil {
		return err
	}
	closeLog, err := glog.Configure(cfg.Log)
	if err != nil {
		return err
	}
	defer closeLog()
	game, err := app.New(cfg)
	if err != nil {
		return err
	}
	defer game.Close()
	glog.Infof("Android starting Vulkan renderer at %dx%d", width, height)
	return render.Run(game, cfg.Window, cfg.Render)
}

//export GoroStop
func GoroStop() {
	host.Lock()
	done := host.done
	host.Unlock()
	if done == nil {
		return
	}
	gogpu.AndroidClose()
	<-done
	host.Lock()
	host.done = nil
	host.Unlock()
}

//export GoroStatus
func GoroStatus() *C.char {
	host.Lock()
	defer host.Unlock()
	if host.status == "" && host.done != nil {
		select {
		case <-host.done:
			return C.CString("@closed")
		default:
		}
	}
	return C.CString(host.status)
}

//export GoroPointer
func GoroPointer(kind, button, buttons C.int, x, y C.float) {
	gogpu.AndroidPointer(gpucontext.PointerEvent{
		Type: gpucontext.PointerEventType(kind), PointerID: 1,
		X: float64(x), Y: float64(y), Button: gpucontext.Button(button),
		Buttons: gpucontext.Buttons(buttons), PointerType: gpucontext.PointerTypeMouse,
		IsPrimary: true, Timestamp: time.Since(epoch),
	})
}

//export GoroScroll
func GoroScroll(x, y, delta C.float) {
	gogpu.AndroidScroll(gpucontext.ScrollEvent{X: float64(x), Y: float64(y), DeltaY: float64(delta), DeltaMode: gpucontext.ScrollDeltaLine})
}

//export GoroKey
func GoroKey(code, mods, down C.int) {
	key := input.AndroidKey(int(code))
	if key != gpucontext.KeyUnknown {
		gogpu.AndroidKey(key, gpucontext.Modifiers(mods), down != 0)
	}
}

//export GoroText
func GoroText(codepoint C.int) { gogpu.AndroidChar(rune(codepoint)) }

func main() {}
