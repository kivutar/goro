package game

import (
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
)

// Use the physical key left of 1, including on AZERTY. Do not steal AltGr.
func (m *WorldMode) toggleWorldMapFromInput(ctx client.Context) bool {
	if !plainCtrlDown(ctx.Input) || !ctx.Input.KeyCodeJustPressed(gpucontext.KeyGrave) {
		return false
	}
	if !m.ui.worldMap.IsOpen() && m.ui.nonConsoleKeyboardInputBlocked() {
		return false
	}
	ctx.Input.ConsumeKeyCodePress(gpucontext.KeyGrave)
	if err := m.ui.worldMap.Toggle(ctx); err != nil {
		m.ui.console.AddErrorMessage("World map unavailable: %v", err)
		glog.Warnf("world map: %v", err)
	}
	return true
}
