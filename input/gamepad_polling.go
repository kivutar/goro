package input

import (
	"slices"
	"sync"
	"time"
)

// pollingGamepads keeps device discovery and driver calls off the frame thread.
// The lock only protects cached samples; it is never held during device I/O.
// Only wrap backends whose OS APIs permit polling on a background goroutine.
type pollingGamepads struct {
	mu   sync.Mutex
	pads []GamepadSnapshot
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

func newPollingGamepads(backend gamepadBackend) *pollingGamepads {
	p := &pollingGamepads{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		defer backend.close()
		ticker := time.NewTicker(4 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-p.stop:
				return
			default:
			}
			pads := backend.poll()
			p.mu.Lock()
			// Disconnects discard pending actions from that controller.
			p.pads = slices.DeleteFunc(p.pads, func(old GamepadSnapshot) bool {
				return !slices.ContainsFunc(pads, func(next GamepadSnapshot) bool { return next.ID == old.ID })
			})
			for _, next := range pads {
				i := slices.IndexFunc(p.pads, func(old GamepadSnapshot) bool { return next.ID == old.ID })
				if i < 0 {
					i = len(p.pads)
					p.pads = append(p.pads, GamepadSnapshot{ID: next.ID})
				}
				pad := &p.pads[i]
				for _, change := range next.Changes {
					pad.setButton(change.Button, change.Down)
				}
				for button, down := range next.Buttons {
					pad.setButton(GamepadButton(button), down)
				}
				pad.Name, pad.Axes = next.Name, next.Axes
			}
			p.mu.Unlock()
			select {
			case <-p.stop:
				return
			case <-ticker.C:
			}
		}
	}()
	return p
}

func (p *pollingGamepads) poll() []GamepadSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	pads := slices.Clone(p.pads)
	for i := range p.pads {
		p.pads[i].Changes = nil // Transfer ownership; later polls cannot replay edges.
	}
	return pads
}

func (p *pollingGamepads) close() {
	p.once.Do(func() { close(p.stop) })
	<-p.done
}
