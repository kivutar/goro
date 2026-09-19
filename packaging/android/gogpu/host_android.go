//go:build android

// This file is overlaid into gogpu by build.sh.
package gogpu

import (
	"fmt"
	"log/slog"

	"github.com/gogpu/gogpu/internal/platform"
	"github.com/gogpu/gpucontext"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

// Android implementations commonly expose RGBA but not desktop's default
// BGRA swapchain format. Set both formats before configuring the surface or
// creating pipelines so the compositor and every render target agree.
func (r *Renderer) selectAndroidSurfaceFormat() error {
	caps := r.adapter.GetSurfaceCapabilities(r.primary.surface)
	if caps == nil || len(caps.Formats) == 0 {
		return fmt.Errorf("gogpu: Android surface has no supported formats")
	}
	format := caps.Formats[0]
preferred:
	for _, candidate := range []gputypes.TextureFormat{gputypes.TextureFormatRGBA8Unorm, gputypes.TextureFormatBGRA8Unorm} {
		for _, supported := range caps.Formats {
			if candidate == supported {
				format = candidate
				break preferred
			}
		}
	}
	r.surfaceFormat = format
	r.primary.format = format
	slog.Info("Android surface capabilities", "format", format, "alphaModes", caps.AlphaModes, "presentModes", caps.PresentModes)
	return nil
}

func resolveAndroidAlphaMode(caps *wgpu.SurfaceCapabilities) gputypes.CompositeAlphaMode {
	if caps == nil || len(caps.AlphaModes) == 0 {
		return gputypes.CompositeAlphaModeAuto
	}
	for _, preferred := range []gputypes.CompositeAlphaMode{gputypes.CompositeAlphaModeOpaque, gputypes.CompositeAlphaModeInherit} {
		for _, supported := range caps.AlphaModes {
			if preferred == supported {
				return supported
			}
		}
	}
	return caps.AlphaModes[0]
}

func AndroidSetWindow(handle uintptr, width, height int) {
	platform.AndroidSetWindow(handle, width, height)
}
func AndroidClose() { platform.AndroidEvent(platform.Event{Type: platform.EventClose}) }
func AndroidPointer(ev gpucontext.PointerEvent) {
	kind := platform.EventPointerMove
	if ev.Type == gpucontext.PointerDown {
		kind = platform.EventPointerDown
	}
	if ev.Type == gpucontext.PointerUp {
		kind = platform.EventPointerUp
	}
	platform.AndroidEvent(platform.Event{Type: kind, Pointer: ev})
}
func AndroidKey(key gpucontext.Key, mods gpucontext.Modifiers, down bool) {
	kind := platform.EventKeyUp
	if down {
		kind = platform.EventKeyDown
	}
	platform.AndroidEvent(platform.Event{Type: kind, Key: key, Mods: mods})
}
func AndroidChar(char rune) {
	platform.AndroidEvent(platform.Event{Type: platform.EventChar, Char: char})
}
func AndroidScroll(ev gpucontext.ScrollEvent) {
	platform.AndroidEvent(platform.Event{Type: platform.EventScroll, Scroll: ev})
}
