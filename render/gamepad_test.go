package render

import (
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
)

func TestGamepadCursorAndMouseShareHeldButtons(t *testing.T) {
	source := &fanoutEventSource{}
	events := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(events, state)
	state.SetMousePosition(100, 100)
	ui := gamepadUIState{}
	now := time.Now()
	ui.update(state, events, now, 300, 200)
	pad := input.GamepadSnapshot{ID: "test"}
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
	state.SetGamepad(input.GamepadSnapshot{})
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
	events := &fanoutEventSource{}
	state := input.NewState()
	wireInput(events, state)
	presses := 0
	events.OnKeyPress(func(key gpucontext.Key, _ gpucontext.Modifiers) {
		if key == gpucontext.KeyEscape {
			presses++
		}
	})
	pad := input.GamepadSnapshot{ID: "test"}
	pad.Buttons[input.GamepadStart] = true
	state.SetGamepad(pad)
	ui := gamepadUIState{}
	ui.update(state, events, time.Now(), 300, 200)
	if presses != 1 || !state.JustPressed(input.KeyEscape) || state.KeyCodeDown(gpucontext.KeyEscape) {
		t.Fatal("Start did not deliver one complete Escape tap")
	}
	state.EndFrame()
	state.SetGamepad(pad)
	ui.update(state, events, time.Now(), 300, 200)
	if presses != 1 {
		t.Fatal("held Start repeated")
	}
}

func TestFocusLossCancelsControllerAndMouseWithoutClicks(t *testing.T) {
	source := &fanoutEventSource{}
	events := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(events, state)
	pad := input.GamepadSnapshot{ID: "test"}
	pad.Buttons[input.GamepadSouth] = true
	state.SetGamepad(pad)
	events.setMouseButton(true, gpucontext.MouseButtonLeft, true, 10, 20)
	for _, fn := range source.mousePress {
		fn(gpucontext.MouseButtonLeft, 10, 20)
	}
	for _, fn := range source.focus {
		fn(false)
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
