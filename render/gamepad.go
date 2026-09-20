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
}

func (r *runner) updateGamepad(now time.Time) {
	if r.gamepads == nil {
		return
	}
	state := r.game.InputState()
	if state == nil {
		return
	}
	pad := r.gamepads.Poll()
	if r.gamepadFocused {
		state.SetGamepad(pad)
	} else {
		state.ResetGamepad()
	}
	r.gamepadUI.update(state, r.gamepadEvents, now, r.width, r.height)
}

// Menu controls work even before a Lua script is loaded. Gameplay bindings
// (left stick, D-pad, West/North, shoulders) remain entirely in Lua.
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
	if x != 0 || y != 0 {
		c.x = max(0, min(float64(max(0, width-1)), c.x+x*800*dt))
		c.y = max(0, min(float64(max(0, height-1)), c.y+y*800*dt))
		for _, fn := range events.mouseMove {
			fn(c.x, c.y)
		}
	}
	c.lastX, c.lastY = state.MouseX, state.MouseY
	for i, button := range [...]input.GamepadButton{input.GamepadSouth, input.GamepadEast} {
		down := state.GamepadDown(button)
		if down == c.buttons[i] {
			continue
		}
		c.buttons[i] = down
		mouse := gpucontext.MouseButtonLeft
		if i == 1 {
			mouse = gpucontext.MouseButtonRight
		}
		events.setMouseButton(true, mouse, down, c.x, c.y)
	}
	if state.GamepadJustPressed(input.GamepadStart) && !state.KeyCodeDown(gpucontext.KeyEscape) {
		// A complete tap avoids sharing a held keyboard key with a controller.
		state.SetKeyCode(gpucontext.KeyEscape, true)
		if events.handleKeyPress != nil {
			events.handleKeyPress(gpucontext.KeyEscape)
		}
		if !events.keyConsumed(gpucontext.KeyEscape) {
			if events.prepareKeyInput != nil {
				events.prepareKeyInput(gpucontext.KeyEscape, 0)
			}
			if !events.keyConsumed(gpucontext.KeyEscape) {
				for _, fn := range events.keyPress {
					fn(gpucontext.KeyEscape, 0)
				}
			}
		}
		state.SetKeyCode(gpucontext.KeyEscape, false)
		for _, fn := range events.keyRelease {
			fn(gpucontext.KeyEscape, 0)
		}
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
