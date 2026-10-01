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
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/app"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

var host struct {
	sync.Mutex
	done   chan struct{}
	status string
	game   *app.Game
}
var epoch = time.Now()

//export GoroStart
func GoroStart(window C.uintptr_t, width, height C.int, directory, source *C.char) {
	host.Lock()
	defer host.Unlock()
	if host.done != nil {
		return
	}
	dir := C.GoString(directory)
	dataSource := C.GoString(source)
	host.status = ""
	done := make(chan struct{})
	host.done = done
	gogpu.AndroidSetWindow(uintptr(window), int(width), int(height))
	input.AndroidResetGamepads()
	go func() {
		defer close(done)
		if err := run(dir, dataSource, int(width), int(height)); err != nil {
			glog.Errorf("Android: %v", err)
			host.Lock()
			host.status = err.Error()
			host.Unlock()
		}
	}()
}

func run(dir, dataSource string, width, height int) error {
	// Settings and screenshots stay in writable app storage. Assets may instead
	// come from a read-only document tree granted by Android's folder picker.
	if err := os.Chdir(dir); err != nil {
		return err
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		return err
	}
	if err := os.Setenv("XDG_CONFIG_HOME", dir); err != nil {
		return err
	}
	cfg, err := config.LoadConfig([]string{"--data-dir", dir, "--width", strconv.Itoa(width), "--height", strconv.Itoa(height), "--graphics-api", "vulkan", "--fullscreen"})
	if err != nil {
		return err
	}
	if cfg.Script.Path == "" {
		cfg.Script.Path = "builtin:wasd"
	}
	closeLog, err := glog.Configure(cfg.Log)
	if err != nil {
		return err
	}
	defer closeLog()
	var game *app.Game
	if strings.HasPrefix(dataSource, "content://") {
		files := newDocumentFS(dataSource)
		// Check the grant before loading: a revoked or moved folder must return
		// to the picker instead of starting with an empty resource manager.
		entries, err := files.ReadDir(".")
		if err != nil {
			return err
		}
		hasData := false
		for _, entry := range entries {
			switch strings.ToLower(entry.Name()) {
			case "data":
				hasData = hasData || entry.IsDir()
			case "data.ini", "data.grf", "fdata.grf", "rdata.grf", "sdata.grf":
				hasData = hasData || !entry.IsDir()
			}
		}
		if !hasData {
			return fmt.Errorf("no Ragnarok client data found: choose the folder containing DATA.INI, data.grf, or data/")
		}
		resource, err := res.NewManagerFS(files)
		if err != nil {
			return err
		}
		// Keep saved AI files separate for each selected client folder.
		cfg.AIStateDir = filepath.Join(dir, "ai-state", fmt.Sprintf("%x", sha256.Sum256([]byte(dataSource))))
		game = app.NewWithResources(cfg, resource)
	} else {
		cfg.DataDir = dataSource
		game, err = app.New(cfg)
		if err != nil {
			return err
		}
	}
	host.Lock()
	host.game = game
	host.Unlock()
	defer func() {
		host.Lock()
		host.game = nil
		host.Unlock()
		game.Close()
	}()
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
	input.AndroidResetGamepads()
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

//export GoroCanChooseFolder
func GoroCanChooseFolder() C.int {
	host.Lock()
	defer host.Unlock()
	if host.game == nil || host.game.InLogin() {
		return 1
	}
	return 0
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

//export GoroFocus
func GoroFocus(focused C.int) { gogpu.AndroidFocus(focused != 0) }

//export GoroGamepadDevice
func GoroGamepadDevice(id C.int, name *C.char, connected C.int) {
	input.AndroidGamepadDevice(int(id), C.GoString(name), connected != 0)
}

//export GoroGamepadKey
func GoroGamepadKey(id, key, down C.int) {
	input.AndroidGamepadKey(int(id), int(key), down != 0)
}

//export GoroGamepadMotion
func GoroGamepadMotion(id C.int, lx, ly, rx, ry, lt, rt, hx, hy C.float) {
	input.AndroidGamepadMotion(int(id), [input.GamepadAxisCount]float64{float64(lx), float64(ly), float64(rx), float64(ry), float64(lt), float64(rt)}, float64(hx), float64(hy))
}

func main() {}
