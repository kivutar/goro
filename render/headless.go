package render

import (
	"context"
	"time"

	"github.com/kivutar/goro/config"
)

// RunHeadless runs the normal frame lifecycle without a window or GPU.
// Frame records CPU-side commands; the headless renderer discards them.
func RunHeadless(ctx context.Context, game Game, cfg config.WindowConfig) error {
	running := true
	if receiver, ok := game.(quitReceiver); ok {
		receiver.SetQuitFunc(func() { running = false })
	}
	game.Resize(cfg.Width, cfg.Height)
	screen := NewFrame(cfg.Width, cfg.Height)
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()
	for running {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := game.Update(); err != nil {
				return err
			}
			screen.BeginFrame()
			game.Draw(screen)
			if drawer, ok := game.(uiOverlayDrawer); ok {
				drawer.DrawUIOverlay(screen)
			}
			if drawer, ok := game.(overlayDrawer); ok {
				drawer.DrawOverlay(screen)
			}
			if receiver, ok := game.(frameSubmittedReceiver); ok {
				receiver.FrameSubmitted()
			}
		}
	}
	return nil
}
