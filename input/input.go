package input

import (
	"math"
	"sort"
)

type Key int

const (
	KeyEnter Key = iota
	KeyEscape
	KeyTab
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyBackspace
	KeyShift
	KeyCtrl
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
)

type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
)

type TouchID int64

type State struct {
	keys              map[Key]bool
	prev              map[Key]bool
	justKeys          map[Key]bool
	buttons           map[MouseButton]bool
	prevMouse         map[MouseButton]bool
	justMouse         map[MouseButton]bool
	justMouseReleased map[MouseButton]bool

	MouseX   int
	MouseY   int
	MouseDX  int
	MouseDY  int
	hasMouse bool

	WheelX        float64
	WheelY        float64
	textInput     []rune
	TouchPoints   []TouchPoint
	PinchDelta    float64
	touchIDs      []TouchID
	touches       map[TouchID]TouchPoint
	hasPinch      bool
	pinchDistance float64
}

type TouchPoint struct {
	ID TouchID
	X  int
	Y  int
}

func NewState() *State {
	return &State{
		keys:              make(map[Key]bool),
		prev:              make(map[Key]bool),
		justKeys:          make(map[Key]bool),
		buttons:           make(map[MouseButton]bool),
		prevMouse:         make(map[MouseButton]bool),
		justMouse:         make(map[MouseButton]bool),
		justMouseReleased: make(map[MouseButton]bool),
		touches:           make(map[TouchID]TouchPoint),
	}
}

func (s *State) Update() {
	s.EndFrame()
}

func (s *State) EndFrame() {
	for key, down := range s.keys {
		s.prev[key] = down
	}
	for key := range s.justKeys {
		delete(s.justKeys, key)
	}
	for button, down := range s.buttons {
		s.prevMouse[button] = down
	}
	for button := range s.justMouse {
		delete(s.justMouse, button)
	}
	for button := range s.justMouseReleased {
		delete(s.justMouseReleased, button)
	}
	s.MouseDX = 0
	s.MouseDY = 0
	s.WheelX = 0
	s.WheelY = 0
	s.textInput = s.textInput[:0]
	s.updateTouches()
}

func (s *State) SetKey(key Key, pressed bool) {
	if pressed && !s.keys[key] {
		s.justKeys[key] = true
	}
	s.keys[key] = pressed
}

func (s *State) SetMouseButton(button MouseButton, pressed bool) {
	if pressed && !s.buttons[button] {
		s.justMouse[button] = true
	}
	if !pressed && s.buttons[button] {
		s.justMouseReleased[button] = true
	}
	s.buttons[button] = pressed
}

func (s *State) SetMousePosition(x, y int) {
	if s.hasMouse {
		s.MouseDX += x - s.MouseX
		s.MouseDY += y - s.MouseY
	} else {
		s.hasMouse = true
	}
	s.MouseX, s.MouseY = x, y
}

func (s *State) AddWheel(x, y float64) {
	s.WheelX += x
	s.WheelY += y
}

func (s *State) AddTextInput(text string) {
	for _, r := range text {
		if r >= 0x20 && r != 0x7f {
			s.textInput = append(s.textInput, r)
		}
	}
}

func (s *State) TextInput() string {
	return string(s.textInput)
}

func (s *State) SetTouch(id TouchID, x, y int, pressed bool) {
	if pressed {
		s.touches[id] = TouchPoint{ID: id, X: x, Y: y}
		return
	}
	delete(s.touches, id)
}

func (s *State) Pressed(key Key) bool {
	return s.keys[key]
}

func (s *State) JustPressed(key Key) bool {
	return s.justKeys[key] || (s.keys[key] && !s.prev[key])
}

func (s *State) MousePressed(button MouseButton) bool {
	return s.buttons[button]
}

func (s *State) MouseJustPressed(button MouseButton) bool {
	return s.justMouse[button] || (s.buttons[button] && !s.prevMouse[button])
}

func (s *State) MouseJustReleased(button MouseButton) bool {
	return s.justMouseReleased[button] || (!s.buttons[button] && s.prevMouse[button])
}

func (s *State) updateTouches() {
	s.touchIDs = s.touchIDs[:0]
	for id := range s.touches {
		s.touchIDs = append(s.touchIDs, id)
	}
	sort.Slice(s.touchIDs, func(i, j int) bool {
		return s.touchIDs[i] < s.touchIDs[j]
	})
	s.TouchPoints = s.TouchPoints[:0]
	for _, id := range s.touchIDs {
		s.TouchPoints = append(s.TouchPoints, s.touches[id])
	}
	s.PinchDelta = 0
	if len(s.TouchPoints) < 2 {
		s.hasPinch = false
		s.pinchDistance = 0
		return
	}
	distance := touchDistance(s.TouchPoints[0], s.TouchPoints[1])
	if s.hasPinch {
		s.PinchDelta = distance - s.pinchDistance
	} else {
		s.hasPinch = true
	}
	s.pinchDistance = distance
}

func touchDistance(a, b TouchPoint) float64 {
	return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))
}
