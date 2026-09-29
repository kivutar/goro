package render

import (
	"slices"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
)

func TestGamepadDeliversShortTapsToUIInOrder(t *testing.T) {
	events := &fanoutEventSource{}
	state := input.NewState()
	wireInput(events, state)
	var got []string
	events.OnMousePress(func(button gpucontext.MouseButton, _, _ float64) {
		if button == gpucontext.MouseButtonLeft {
			got = append(got, "left down")
		} else {
			got = append(got, "right down")
		}
	})
	events.OnMouseRelease(func(button gpucontext.MouseButton, _, _ float64) {
		if button == gpucontext.MouseButtonLeft {
			got = append(got, "left up")
		} else {
			got = append(got, "right up")
		}
	})
	events.OnKeyPress(func(key gpucontext.Key, _ gpucontext.Modifiers) {
		if key == gpucontext.KeyEscape {
			got = append(got, "escape")
		}
	})
	pad := input.GamepadFrame{ID: "test", Changes: []input.GamepadButtonChange{
		{Button: input.GamepadSouth, Down: true},
		{Button: input.GamepadEast, Down: true},
		{Button: input.GamepadSouth, Down: false},
		{Button: input.GamepadStart, Down: true},
		{Button: input.GamepadStart, Down: false},
		{Button: input.GamepadEast, Down: false},
		{Button: input.GamepadStart, Down: true},
		{Button: input.GamepadStart, Down: false},
	}}
	state.SetGamepad(pad)
	ui := gamepadUIState{}
	ui.update(state, events, time.Now(), 300, 200)
	want := []string{"left down", "right down", "left up", "escape", "right up", "escape"}
	if !slices.Equal(got, want) {
		t.Fatalf("UI events = %v, want %v", got, want)
	}
	if state.MousePressed(input.MouseButtonLeft) || state.MousePressed(input.MouseButtonRight) || state.KeyCodeDown(gpucontext.KeyEscape) {
		t.Fatal("a short tap left a held UI input")
	}
	state.EndFrame()
	pad.Changes = nil
	state.SetGamepad(pad)
	ui.update(state, events, time.Now(), 300, 200)
	if !slices.Equal(got, want) {
		t.Fatal("consumed UI taps replayed on the next frame")
	}
}

func TestGamepadCursorAndMouseShareHeldButtons(t *testing.T) {
	source := &fanoutEventSource{}
	events := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(events, state)
	state.SetMousePosition(100, 100)
	ui := gamepadUIState{}
	now := time.Now()
	ui.update(state, events, now, 300, 200)
	pad := input.GamepadFrame{ID: "test"}
	pad.Axes[input.GamepadRightX] = 1
	pad.Buttons[input.GamepadSouth] = true
	state.SetGamepad(pad)
	ui.update(state, events, now.Add(time.Second/60), 300, 200)
	if state.MouseX <= 100 || state.MouseY != 100 || !state.MouseJustPressed(input.MouseButtonLeft) {
		t.Fatal("controller did not move/click the pointer")
	}
	for _, fn := range source.mousePress {
		fn(gpucontext.MouseButtonLeft, 110, 100)
	}
	state.EndFrame()
	state.SetGamepad(input.GamepadFrame{})
	ui.update(state, events, now.Add(time.Second/30), 300, 200)
	if !state.MousePressed(input.MouseButtonLeft) || state.MouseJustReleased(input.MouseButtonLeft) {
		t.Fatal("controller disconnect released a physical mouse button")
	}
	for _, fn := range source.mouseRelease {
		fn(gpucontext.MouseButtonLeft, 110, 100)
	}
	if state.MousePressed(input.MouseButtonLeft) || !state.MouseJustReleased(input.MouseButtonLeft) {
		t.Fatal("physical mouse release was lost")
	}
}

func TestGamepadStartTapsEscapeOnce(t *testing.T) {
	for _, consumeAt := range []string{"none", "handler", "prepare"} {
		t.Run(consumeAt, func(t *testing.T) {
			events := &fanoutEventSource{}
			state := input.NewState()
			wireInput(events, state)
			wantPresses := 0
			switch consumeAt {
			case "handler":
				events.handleKeyPress = func(key input.KeyCode) { state.ConsumeKeyCodePress(key) }
			case "prepare":
				events.prepareKeyInput = func(key input.KeyCode, _ gpucontext.Modifiers) { state.ConsumeKeyCodePress(key) }
			default:
				wantPresses = 1
			}
			presses := 0
			events.OnKeyPress(func(key gpucontext.Key, _ gpucontext.Modifiers) {
				if key == gpucontext.KeyEscape {
					presses++
				}
			})
			pad := input.GamepadFrame{ID: "test"}
			pad.Buttons[input.GamepadStart] = true
			state.SetGamepad(pad)
			ui := gamepadUIState{}
			ui.update(state, events, time.Now(), 300, 200)
			if presses != wantPresses || state.JustPressed(input.KeyEscape) != (wantPresses == 1) || state.KeyCodeDown(gpucontext.KeyEscape) {
				t.Fatal("Start did not deliver one complete Escape tap")
			}
			state.EndFrame()
			state.SetGamepad(pad)
			ui.update(state, events, time.Now(), 300, 200)
			if presses != wantPresses {
				t.Fatal("held Start repeated")
			}
		})
	}
}

func TestCapturedGamepadControlsDoNotMoveOrClickPointer(t *testing.T) {
	events := &fanoutEventSource{}
	state := input.NewState()
	wireInput(events, state)
	state.SetMousePosition(100, 100)
	clicks := 0
	events.OnMousePress(func(gpucontext.MouseButton, float64, float64) { clicks++ })
	now := time.Now()
	ui := gamepadUIState{}
	ui.update(state, events, now, 300, 200)
	pad := input.GamepadFrame{ID: "test"}
	pad.Axes[input.GamepadRightX] = 1
	pad.Buttons[input.GamepadSouth] = true
	state.SetGamepad(pad)
	ui.capture.Pointer = true
	ui.capture.Buttons[input.GamepadSouth] = true
	ui.update(state, events, now.Add(time.Second/60), 300, 200)
	if state.MouseX != 100 || clicks != 0 {
		t.Fatal("captured controls reached the pointer")
	}
	state.EndFrame()
	ui.capture = input.GamepadCapture{}
	ui.update(state, events, now.Add(time.Second/30), 300, 200)
	if clicks != 0 || state.MousePressed(input.MouseButtonLeft) {
		t.Fatal("releasing modifier turned a held face button into a click")
	}
	pad.Buttons[input.GamepadSouth] = false
	state.SetGamepad(pad)
	ui.update(state, events, now.Add(time.Second/20), 300, 200)
	state.EndFrame()
	pad.Buttons[input.GamepadSouth] = true
	state.SetGamepad(pad)
	ui.update(state, events, now.Add(time.Second/15), 300, 200)
	if clicks != 1 {
		t.Fatal("pointer click did not return after releasing the button")
	}
	// A release and new press may share one frame. The new press must follow
	// the current capture, even when the previous press belonged to the UI.
	state.EndFrame()
	pad.Changes = []input.GamepadButtonChange{
		{Button: input.GamepadSouth, Down: false},
		{Button: input.GamepadSouth, Down: true},
	}
	state.SetGamepad(pad)
	ui.capture.Buttons[input.GamepadSouth] = true
	ui.update(state, events, now.Add(time.Second/10), 300, 200)
	if clicks != 1 || state.MousePressed(input.MouseButtonLeft) {
		t.Fatal("captured re-press leaked after releasing a UI press in the same frame")
	}
	// Conversely, releasing a captured press allows a fresh unclaimed press.
	state.EndFrame()
	state.SetGamepad(pad)
	ui.capture = input.GamepadCapture{}
	ui.update(state, events, now.Add(time.Second/5), 300, 200)
	if clicks != 2 || !state.MousePressed(input.MouseButtonLeft) {
		t.Fatal("a fresh pointer press inherited the previous press's capture")
	}
}

func TestFocusLossCancelsControllerAndMouseWithoutClicks(t *testing.T) {
	source := &fanoutEventSource{}
	events := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(events, state)
	pad := input.GamepadFrame{ID: "test"}
	pad.Buttons[input.GamepadSouth] = true
	state.SetGamepad(pad)
	events.setMouseButton(true, gpucontext.MouseButtonLeft, true, 10, 20)
	for _, fn := range source.mousePress {
		fn(gpucontext.MouseButtonLeft, 10, 20)
	}
	for _, fn := range source.focus {
		fn(false)
	}
	if len(state.GamepadChanges()) != 0 {
		t.Fatal("focus loss retained queued controller input")
	}
	if state.GamepadConnected() || state.GamepadJustReleased(input.GamepadSouth) || state.MousePressed(input.MouseButtonLeft) || state.MouseJustReleased(input.MouseButtonLeft) {
		t.Fatal("focus loss retained held input or generated a click")
	}
	for _, fn := range source.focus {
		fn(true)
	}
	for _, fn := range source.mousePress {
		fn(gpucontext.MouseButtonLeft, 10, 20)
	}
	if !state.MouseJustPressed(input.MouseButtonLeft) {
		t.Fatal("stale mouse state blocked input after regaining focus")
	}
}
