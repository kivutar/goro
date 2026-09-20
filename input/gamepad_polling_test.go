package input

import (
	"sync/atomic"
	"testing"
	"time"
)

type stalledGamepads struct {
	entered chan struct{}
	samples chan []GamepadSnapshot
	unblock chan struct{}
	closed  atomic.Bool
}

func (b *stalledGamepads) poll() []GamepadSnapshot {
	select {
	case b.entered <- struct{}{}:
	case <-b.unblock:
		return nil
	}
	select {
	case pads := <-b.samples:
		return pads
	case <-b.unblock:
		return nil
	}
}
func (b *stalledGamepads) close() { b.closed.Store(true) }

func TestGamepadPollDoesNotWaitForDeviceIO(t *testing.T) {
	backend := &stalledGamepads{entered: make(chan struct{}), samples: make(chan []GamepadSnapshot), unblock: make(chan struct{})}
	poller := newPollingGamepads(backend)
	t.Cleanup(func() { close(backend.unblock); poller.close() })
	awaitIO := func() {
		t.Helper()
		select {
		case <-backend.entered:
		case <-time.After(time.Second):
			t.Fatal("device worker did not enter poll")
		}
	}
	read := func() []GamepadSnapshot {
		t.Helper()
		result := make(chan []GamepadSnapshot, 1)
		go func() { result <- poller.poll() }()
		select {
		case pads := <-result:
			return pads
		case <-time.After(time.Second):
			t.Fatal("frame thread blocked on device I/O")
			return nil
		}
	}
	awaitIO() // The underlying device call is stalled, as discovery was on Linux.
	if len(read()) != 0 {
		t.Fatal("unexpected initial sample")
	}
	pad := GamepadSnapshot{ID: "test"}
	pad.Buttons[GamepadWest] = true
	sample := []GamepadSnapshot{pad}
	backend.samples <- sample
	awaitIO() // Publishing completed; the next device call is now stalled.
	if pads := read(); len(pads) != 1 || !pads[0].Buttons[GamepadWest] {
		t.Fatal("cached controller sample was not available during device I/O")
	}
	sample[0].Buttons[GamepadWest] = false
	if !read()[0].Buttons[GamepadWest] {
		t.Fatal("backend buffer reuse mutated a published sample")
	}
	backend.samples <- nil // Disconnect.
	awaitIO()
	if len(read()) != 0 {
		t.Fatal("disconnect was not published")
	}
	done := make(chan struct{})
	go func() { poller.close(); close(done) }()
	<-poller.stop
	backend.samples <- nil
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	if !backend.closed.Load() {
		t.Fatal("device handles were not closed")
	}
	poller.close() // Idempotent close must not panic or block.
}
