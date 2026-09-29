package input

import (
	"math"

	"github.com/kivutar/goro/glog"
)

// GamepadButton names describe positions, independent of the printed labels.
type GamepadButton uint8

const (
	GamepadSouth GamepadButton = iota
	GamepadEast
	GamepadWest
	GamepadNorth
	GamepadLeftShoulder
	GamepadRightShoulder
	GamepadBack
	GamepadStart
	GamepadLeftStick
	GamepadRightStick
	GamepadUp
	GamepadDown
	GamepadLeft
	GamepadRight
	GamepadButtonCount
)

type GamepadAxis uint8

const (
	GamepadLeftX GamepadAxis = iota
	GamepadLeftY
	GamepadRightX
	GamepadRightY
	GamepadLeftTrigger
	GamepadRightTrigger
	GamepadAxisCount
)

var gamepadButtonNames = [...]string{"south", "east", "west", "north", "left_shoulder", "right_shoulder", "back", "start", "left_stick", "right_stick", "dpad_up", "dpad_down", "dpad_left", "dpad_right"}
var gamepadAxisNames = [...]string{"left_x", "left_y", "right_x", "right_y", "left_trigger", "right_trigger"}

func GamepadButtonFromName(name string) (GamepadButton, bool) {
	for button, candidate := range gamepadButtonNames {
		if name == candidate {
			return GamepadButton(button), true
		}
	}
	return 0, false
}

func GamepadAxisFromName(name string) (GamepadAxis, bool) {
	for axis, candidate := range gamepadAxisNames {
		if name == candidate {
			return GamepadAxis(axis), true
		}
	}
	return 0, false
}

// GamepadFrame transfers the latest normalized controller state and the ordered
// button transitions collected since the previous drain. Empty ID means disconnected.
// Sticks use [-1,1], with negative Y pointing up; triggers use [0,1].
type GamepadFrame struct {
	ID      string
	Name    string
	Buttons [GamepadButtonCount]bool
	Axes    [GamepadAxisCount]float64
	// Changes belongs to the receiver; draining again cannot replay or mutate it.
	Changes []GamepadButtonChange
}

type GamepadButtonChange struct {
	Button GamepadButton
	Down   bool
}

// GamepadCapture lets gameplay claim controls before the pointer fallback.
// It applies to one frame; button captures last until release in the renderer.
type GamepadCapture struct {
	Buttons [GamepadButtonCount]bool
	Pointer bool
}

func (p *GamepadFrame) setButton(button GamepadButton, down bool) {
	if button < GamepadButtonCount && p.Buttons[button] != down {
		p.Buttons[button] = down
		p.Changes = append(p.Changes, GamepadButtonChange{button, down})
	}
}

type gamepadState struct {
	GamepadFrame
	pressed  [GamepadButtonCount]bool
	released [GamepadButtonCount]bool
	changes  []GamepadButtonChange
}

// SetGamepad runs on the game thread, like SetKeyCode. Sources must not mutate
// State from OS callback threads. Switching controllers clears the old edges.
func (s *State) SetGamepad(next GamepadFrame) {
	if next.ID == "" {
		next = GamepadFrame{}
	}
	if next.ID != s.gamepad.ID && next.ID != "" {
		s.ResetGamepad()
	}
	for _, change := range next.Changes {
		s.setGamepadButton(change.Button, change.Down)
	}
	for button, down := range next.Buttons {
		s.setGamepadButton(GamepadButton(button), down)
	}
	for axis, value := range next.Axes {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			value = 0
		}
		low := -1.0
		if GamepadAxis(axis) >= GamepadLeftTrigger {
			low = 0
		}
		next.Axes[axis] = max(low, min(1, value))
	}
	next.Changes = nil
	s.gamepad.GamepadFrame = next
}

func (s *State) setGamepadButton(button GamepadButton, down bool) {
	if button >= GamepadButtonCount || s.gamepad.Buttons[button] == down {
		return
	}
	s.gamepad.Buttons[button] = down
	if down {
		s.gamepad.pressed[button] = true
	} else {
		s.gamepad.released[button] = true
	}
	s.gamepad.changes = append(s.gamepad.changes, GamepadButtonChange{button, down})
}

// GamepadChanges reads this frame's transitions without consuming them.
// The returned slice is read-only and valid until EndFrame or ResetGamepad.
func (s *State) GamepadChanges() []GamepadButtonChange { return s.gamepad.changes }

func (s *State) ResetGamepad()          { s.gamepad = gamepadState{} }
func (s *State) GamepadConnected() bool { return s.gamepad.ID != "" }
func (s *State) GamepadName() string    { return s.gamepad.Name }
func (s *State) GamepadDown(button GamepadButton) bool {
	return button < GamepadButtonCount && s.gamepad.Buttons[button]
}
func (s *State) GamepadJustPressed(button GamepadButton) bool {
	return button < GamepadButtonCount && s.gamepad.pressed[button]
}
func (s *State) GamepadJustReleased(button GamepadButton) bool {
	return button < GamepadButtonCount && s.gamepad.released[button]
}
func (s *State) GamepadValue(axis GamepadAxis) float64 {
	if axis >= GamepadAxisCount {
		return 0
	}
	return s.gamepad.Axes[axis]
}

type gamepadBackend interface {
	// drain transfers pending changes for every device, together with its current state.
	drain() []GamepadFrame
	close()
}

// GamepadSource keeps the first controller selected until it disconnects.
// It has one consumer: the window loop. Create, drain and close on that thread
// (required by macOS). Other consumers read the shared State after SetGamepad.
type GamepadSource struct {
	backend  gamepadBackend
	selected string
}

func NewGamepadSource() (*GamepadSource, error) {
	backend, err := newGamepadBackend()
	if err != nil {
		return nil, err
	}
	return &GamepadSource{backend: backend}, nil
}

// DrainFrame consumes queued changes for all controllers and returns the selected
// controller's frame. Call once per update, then pass it to State.SetGamepad.
// Diagnostics and other readers must query State instead of draining the source.
func (s *GamepadSource) DrainFrame() GamepadFrame {
	pads := s.backend.drain()
	for _, pad := range pads {
		if pad.ID == s.selected {
			return pad
		}
	}
	if len(pads) > 0 {
		s.selected = pads[0].ID
		glog.Infof("gamepad selected name=%q id=%q", pads[0].Name, s.selected)
		return pads[0]
	}
	if s.selected != "" {
		glog.Infof("gamepad disconnected id=%q", s.selected)
	}
	s.selected = ""
	return GamepadFrame{}
}

func (s *GamepadSource) Close() { s.backend.close() }

// DiscardPending drops queued actions at a focus boundary, even if updates were
// suspended while unfocused. The next drain still reports current held state.
func (s *GamepadSource) DiscardPending() { s.DrainFrame() }
