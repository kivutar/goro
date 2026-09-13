package app

import (
	"context"
	"time"
)

// RunHeadless drives the same updates as the windowed client, without creating
// a renderer. Lua retains its own, slower tick interval.
func RunHeadless(ctx context.Context, g *Game) error {
	g.scriptContext = ctx
	defer g.Close()
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()
	for !g.quitting {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := g.Update(); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
	return nil
}
