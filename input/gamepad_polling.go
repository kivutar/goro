package input

import (
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type gamepadSamples struct{ pads []GamepadSnapshot }

// pollingGamepads keeps device discovery and driver calls off the frame thread.
// Published snapshots are immutable; poll never waits for device I/O or a lock.
// Only wrap backends whose OS APIs permit polling on a background goroutine.
type pollingGamepads struct {
	latest atomic.Pointer[gamepadSamples]
	stop   chan struct{}
	done   chan struct{}
	once   sync.Once
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
			// The backend owns its returned slice. Copy it before publishing so
			// later samples cannot mutate data the frame thread is still reading.
			pads := backend.poll()
			if previous := p.latest.Load(); previous == nil || !slices.Equal(previous.pads, pads) {
				p.latest.Store(&gamepadSamples{pads: append([]GamepadSnapshot(nil), pads...)})
			}
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
	if sample := p.latest.Load(); sample != nil {
		return sample.pads
	}
	return nil
}

func (p *pollingGamepads) close() {
	p.once.Do(func() { close(p.stop) })
	<-p.done
}
