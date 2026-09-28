package input

import (
	"slices"
	"testing"
	"time"
)

func TestAndroidGamepadTapBetweenFrames(t *testing.T) {
	AndroidResetGamepads()
	t.Cleanup(AndroidResetGamepads)
	AndroidGamepadDevice(7, "controller", true)
	source := &GamepadSource{backend: &androidGamepadBackend{}}
	state := NewState()
	state.SetGamepad(source.Poll())
	state.EndFrame()

	// Both taps occur while the game is drawing one slow frame.
	for range 2 {
		AndroidGamepadKey(7, 108, true)
		AndroidGamepadKey(7, 108, true) // Key repeat must not add another press.
		AndroidGamepadKey(7, 108, false)
	}
	// A D-pad hat tap must survive too, even with newer analog samples.
	AndroidGamepadMotion(7, [GamepadAxisCount]float64{}, 1, 0)
	AndroidGamepadMotion(7, [GamepadAxisCount]float64{0.75}, 0, 0)
	state.SetGamepad(source.Poll())
	want := []GamepadButtonChange{
		{GamepadStart, true}, {GamepadStart, false},
		{GamepadStart, true}, {GamepadStart, false},
		{GamepadRight, true}, {GamepadRight, false},
	}
	if !slices.Equal(state.GamepadChanges(), want) {
		t.Fatalf("button order = %v, want %v", state.GamepadChanges(), want)
	}
	if !state.GamepadJustPressed(GamepadStart) || !state.GamepadJustReleased(GamepadStart) || state.GamepadDown(GamepadStart) || state.GamepadValue(GamepadLeftX) != 0.75 {
		t.Fatal("tap edges or latest analog position lost")
	}
	state.EndFrame()
	state.SetGamepad(source.Poll())
	if len(state.GamepadChanges()) != 0 || state.GamepadJustPressed(GamepadStart) || state.GamepadJustReleased(GamepadStart) {
		t.Fatal("consumed edges replayed on the next frame")
	}

	AndroidGamepadKey(7, 108, true)
	AndroidGamepadDevice(7, "", false)
	AndroidGamepadDevice(7, "controller", true)
	state.SetGamepad(source.Poll())
	if len(state.GamepadChanges()) != 0 {
		t.Fatal("disconnect retained pending input")
	}
	AndroidGamepadKey(7, 108, true)
	AndroidResetGamepads() // Android focus loss.
	AndroidGamepadDevice(7, "controller", true)
	state.ResetGamepad()
	state.SetGamepad(source.Poll())
	if len(state.GamepadChanges()) != 0 {
		t.Fatal("focus loss retained pending input")
	}
}

func TestGamepadWorkerRetainsTransitionsUntilConsumed(t *testing.T) {
	backend := &stalledGamepads{entered: make(chan struct{}), samples: make(chan []GamepadSnapshot), unblock: make(chan struct{})}
	poller := newPollingGamepads(backend)
	t.Cleanup(func() { close(backend.unblock); poller.close() })
	awaitPoll := func() {
		t.Helper()
		select {
		case <-backend.entered:
		case <-time.After(time.Second):
			t.Fatal("worker stalled")
		}
	}
	sample := func(pads ...GamepadSnapshot) {
		t.Helper()
		backend.samples <- pads
		awaitPoll()
	}
	source := &GamepadSource{backend: poller}
	state := NewState()
	pad := GamepadSnapshot{ID: "controller"}
	awaitPoll()
	sample(pad)
	state.SetGamepad(source.Poll())
	state.EndFrame()

	// The worker samples both edges between game updates.
	pad.Buttons[GamepadRightShoulder] = true
	sample(pad)
	pad.Buttons[GamepadRightShoulder] = false
	pad.Axes[GamepadLeftX] = 0.5
	sample(pad)
	state.SetGamepad(source.Poll())
	if !state.GamepadJustPressed(GamepadRightShoulder) || !state.GamepadJustReleased(GamepadRightShoulder) || state.GamepadDown(GamepadRightShoulder) || state.GamepadValue(GamepadLeftX) != 0.5 {
		t.Fatal("sampled tap or latest stick position lost")
	}
	saved := slices.Clone(state.GamepadChanges())
	if !slices.Equal(saved, []GamepadButtonChange{{GamepadRightShoulder, true}, {GamepadRightShoulder, false}}) {
		t.Fatal("sampled edges out of order")
	}
	state.EndFrame()
	state.SetGamepad(source.Poll())
	if len(state.GamepadChanges()) != 0 {
		t.Fatal("sampled edges replayed")
	}

	pad.Buttons[GamepadRightShoulder] = true
	sample(pad)
	source.Poll() // The focus callback drains pending transitions.
	state.ResetGamepad()
	pad.Buttons[GamepadRightShoulder] = false
	sample(pad)
	state.SetGamepad(source.Poll())
	if state.GamepadJustPressed(GamepadRightShoulder) {
		t.Fatal("tap from before focus change replayed")
	}
	state.EndFrame()

	pad.Buttons[GamepadRightShoulder] = true
	sample(pad)
	sample() // Disconnect before this queued press reaches the game.
	state.SetGamepad(source.Poll())
	if state.GamepadConnected() || len(state.GamepadChanges()) != 0 {
		t.Fatal("disconnected controller delivered queued actions")
	}
	pad.Buttons[GamepadRightShoulder] = false
	sample(pad)
	state.SetGamepad(source.Poll())
	if len(state.GamepadChanges()) != 0 {
		t.Fatal("reconnected controller replayed queued actions")
	}
}
