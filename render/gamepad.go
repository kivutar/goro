package render

import (
	"math"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
)

type gamepadUIState struct {
	lastUpdate   time.Time
	x, y         float64
	lastX, lastY int
	buttons      [2]bool
	suppressed   [2]bool
	capture      input.GamepadCapture
}

func (r *runner) updateGamepad(now time.Time) {
	if r.gamepads == nil {
		return
	}
	state := r.game.InputState()
	if state == nil {
		return
	}
	pad := r.gamepads.DrainFrame()
	if r.gamepadFocused {
		state.SetGamepad(pad)
	} else {
		state.ResetGamepad()
	}
	r.gamepadUI.capture = input.GamepadCapture{}
	if handler, ok := r.game.(interface {
		HandleGamepadInput(float64) input.GamepadCapture
	}); ok {
		dt := 0.0
		if !r.gamepadUI.lastUpdate.IsZero() {
			dt = max(0, min(now.Sub(r.gamepadUI.lastUpdate).Seconds(), 0.05))
		}
		r.gamepadUI.capture = handler.HandleGamepadInput(dt)
	}
	r.gamepadUI.update(state, r.gamepadEvents, now, r.width, r.height)
}

// Unclaimed controls provide pointer navigation even without a Lua script.
func (c *gamepadUIState) update(state *input.State, events *fanoutEventSource, now time.Time, width, height int) {
	dt := 0.0
	if !c.lastUpdate.IsZero() {
		dt = min(now.Sub(c.lastUpdate).Seconds(), 0.05)
	}
	if c.lastUpdate.IsZero() || state.MouseX != c.lastX || state.MouseY != c.lastY {
		c.x, c.y = float64(state.MouseX), float64(state.MouseY)
	}
	c.lastUpdate = now
	axis := func(axis input.GamepadAxis) float64 {
		v := state.GamepadValue(axis)
		if math.Abs(v) <= 0.2 {
			return 0
		}
		return math.Copysign((math.Abs(v)-0.2)/0.8, v)
	}
	x, y := axis(input.GamepadRightX), axis(input.GamepadRightY)
	if !c.capture.Pointer && (x != 0 || y != 0) {
		c.x = max(0, min(float64(max(0, width-1)), c.x+x*800*dt))
		c.y = max(0, min(float64(max(0, height-1)), c.y+y*800*dt))
		for _, fn := range events.mouseMove {
			fn(c.x, c.y)
		}
	}
	c.lastX, c.lastY = state.MouseX, state.MouseY
	setMouseButton := func(i int, down bool) {
		if down == c.buttons[i] {
			return
		}
		c.buttons[i] = down
		mouse := gpucontext.MouseButtonLeft
		if i == 1 {
			mouse = gpucontext.MouseButtonRight
		}
		events.setMouseButton(true, mouse, down, c.x, c.y)
	}
	mouseButtons := [...]input.GamepadButton{input.GamepadSouth, input.GamepadEast}
	applyMouseButton := func(i int, down bool) {
		if down && c.capture.Buttons[mouseButtons[i]] && !c.buttons[i] {
			c.suppressed[i] = true
		}
		setMouseButton(i, down && !c.suppressed[i])
		if !down {
			c.suppressed[i] = false
		}
	}
	for _, change := range state.GamepadChanges() {
		switch change.Button {
		case input.GamepadSouth:
			applyMouseButton(0, change.Down)
		case input.GamepadEast:
			applyMouseButton(1, change.Down)
		case input.GamepadStart:
			if change.Down && !c.capture.Buttons[change.Button] {
				events.tapKey(gpucontext.KeyEscape)
			}
		}
	}
	// Reconcile held buttons after a controller switch or input reset.
	for i, button := range mouseButtons {
		applyMouseButton(i, state.GamepadDown(button))
	}
}

// A controller release must not release a button still held on a real mouse.
func (f *fanoutEventSource) setMouseButton(controller bool, button gpucontext.MouseButton, down bool, x, y float64) {
	if f.physicalMouse == nil {
		f.physicalMouse = make(map[gpucontext.MouseButton]bool)
	}
	if f.controllerMouse == nil {
		f.controllerMouse = make(map[gpucontext.MouseButton]bool)
	}
	wasDown := f.physicalMouse[button] || f.controllerMouse[button]
	if controller {
		f.controllerMouse[button] = down
	} else {
		f.physicalMouse[button] = down
	}
	isDown := f.physicalMouse[button] || f.controllerMouse[button]
	if isDown == wasDown {
		return
	}
	listeners := f.mouseRelease
	if isDown {
		listeners = f.mousePress
	}
	for _, fn := range listeners {
		fn(button, x, y)
	}
}
