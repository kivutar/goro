package input

import (
	"math"
	"testing"
)

func TestGamepadEdgesDisconnectAndFocusLoss(t *testing.T) {
	s := NewState()
	pad := GamepadSnapshot{ID: "one", Name: "Controller"}
	pad.Buttons[GamepadWest] = true
	pad.Axes[GamepadLeftX] = 2
	pad.Axes[GamepadLeftY] = math.NaN()
	s.SetGamepad(pad)
	if !s.GamepadConnected() || !s.GamepadDown(GamepadWest) || !s.GamepadJustPressed(GamepadWest) || s.GamepadValue(GamepadLeftX) != 1 || s.GamepadValue(GamepadLeftY) != 0 {
		t.Fatal("invalid initial controller snapshot")
	}
	s.EndFrame()
	s.SetGamepad(pad)
	if s.GamepadJustPressed(GamepadWest) || !s.GamepadDown(GamepadWest) {
		t.Fatal("held button generated another edge")
	}
	s.SetGamepad(GamepadSnapshot{})
	if s.GamepadConnected() || s.GamepadDown(GamepadWest) || !s.GamepadJustReleased(GamepadWest) || s.GamepadValue(GamepadLeftX) != 0 {
		t.Fatal("disconnect left a held input")
	}
	s.EndFrame()
	if s.GamepadJustReleased(GamepadWest) {
		t.Fatal("release survived the frame")
	}
	s.SetGamepad(pad)
	s.ResetGamepad()
	if s.GamepadDown(GamepadWest) || s.GamepadJustPressed(GamepadWest) || s.GamepadJustReleased(GamepadWest) {
		t.Fatal("focus cancellation generated input")
	}
}

func TestAndroidGamepadKeepsKeysHatsAndTriggersIndependent(t *testing.T) {
	AndroidResetGamepads()
	t.Cleanup(AndroidResetGamepads)
	AndroidGamepadDevice(4, "Test controller", true)
	AndroidGamepadKey(4, 19, true)  // D-pad key held while stick events arrive.
	AndroidGamepadKey(4, 104, true) // Digital left trigger.
	AndroidGamepadMotion(4, [GamepadAxisCount]float64{0.75, -0.5}, 1, 0)
	backend := &androidGamepadBackend{}
	pad := backend.poll()[0]
	if !pad.Buttons[GamepadUp] || !pad.Buttons[GamepadRight] || pad.Axes[GamepadLeftTrigger] != 1 || pad.Axes[GamepadLeftX] != 0.75 {
		t.Fatal("motion erased a held key or trigger")
	}
	AndroidGamepadKey(4, 22, true)
	AndroidGamepadMotion(4, [GamepadAxisCount]float64{}, 0, 0)
	AndroidGamepadKey(4, 19, false)
	AndroidGamepadKey(4, 104, false)
	pad = backend.poll()[0]
	if pad.Buttons[GamepadUp] || !pad.Buttons[GamepadRight] || pad.Axes[GamepadLeftTrigger] != 0 {
		t.Fatal("key/hat release corrupted another source")
	}
	AndroidGamepadDevice(4, "", false)
	AndroidGamepadKey(4, 99, true)
	if len(backend.poll()) != 0 {
		t.Fatal("late input recreated a disconnected device")
	}
}

type testGamepads struct{ pads []GamepadSnapshot }

func (b *testGamepads) poll() []GamepadSnapshot { return b.pads }
func (*testGamepads) close()                    {}

func TestGamepadSelectionSurvivesEnumerationChanges(t *testing.T) {
	backend := &testGamepads{pads: []GamepadSnapshot{{ID: "one"}, {ID: "two"}}}
	source := &GamepadSource{backend: backend}
	if source.Poll().ID != "one" {
		t.Fatal("did not select the first controller")
	}
	backend.pads[0], backend.pads[1] = backend.pads[1], backend.pads[0]
	if source.Poll().ID != "one" {
		t.Fatal("enumeration reordered the active controller")
	}
	backend.pads = backend.pads[:1]
	if source.Poll().ID != "two" {
		t.Fatal("did not switch after disconnect")
	}
	backend.pads = nil
	if source.Poll().ID != "" {
		t.Fatal("last controller remained connected")
	}
}
