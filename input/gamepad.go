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

// GamepadSnapshot is a normalized controller sample. Empty ID means disconnected.
// Sticks use [-1,1], with negative Y pointing up; triggers use [0,1].
type GamepadSnapshot struct {
	ID      string
	Name    string
	Buttons [GamepadButtonCount]bool
	Axes    [GamepadAxisCount]float64
}

type gamepadState struct {
	GamepadSnapshot
	pressed  [GamepadButtonCount]bool
	released [GamepadButtonCount]bool
}

// SetGamepad runs on the game thread, like SetKeyCode. Sources must not mutate
// State from OS callback threads. Switching controllers clears the old edges.
func (s *State) SetGamepad(next GamepadSnapshot) {
	if next.ID == "" {
		next = GamepadSnapshot{}
	}
	if next.ID != s.gamepad.ID && next.ID != "" {
		s.ResetGamepad()
	}
	for button, down := range next.Buttons {
		if down && !s.gamepad.Buttons[button] {
			s.gamepad.pressed[button] = true
		}
		if !down && s.gamepad.Buttons[button] {
			s.gamepad.released[button] = true
		}
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
	s.gamepad.GamepadSnapshot = next
}

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
	poll() []GamepadSnapshot
	close()
}

// GamepadSource keeps the first controller selected until it disconnects.
// Create, poll and close on the window's thread (required by macOS).
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

func (s *GamepadSource) Poll() GamepadSnapshot {
	pads := s.backend.poll()
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
	return GamepadSnapshot{}
}

func (s *GamepadSource) Close() { s.backend.close() }
