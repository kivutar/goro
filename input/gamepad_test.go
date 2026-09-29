package input

import (
	"math"
	"testing"
)

func TestGamepadEdgesDisconnectAndFocusLoss(t *testing.T) {
	s := NewState()
	pad := GamepadFrame{ID: "one", Name: "Controller"}
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
	s.SetGamepad(GamepadFrame{})
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
	pad := backend.drain()[0]
	if !pad.Buttons[GamepadUp] || !pad.Buttons[GamepadRight] || pad.Axes[GamepadLeftTrigger] != 1 || pad.Axes[GamepadLeftX] != 0.75 {
		t.Fatal("motion erased a held key or trigger")
	}
	AndroidGamepadKey(4, 22, true)
	AndroidGamepadMotion(4, [GamepadAxisCount]float64{}, 0, 0)
	AndroidGamepadKey(4, 19, false)
	AndroidGamepadKey(4, 104, false)
	pad = backend.drain()[0]
	if pad.Buttons[GamepadUp] || !pad.Buttons[GamepadRight] || pad.Axes[GamepadLeftTrigger] != 0 {
		t.Fatal("key/hat release corrupted another source")
	}
	AndroidGamepadDevice(4, "", false)
	AndroidGamepadKey(4, 99, true)
	if len(backend.drain()) != 0 {
		t.Fatal("late input recreated a disconnected device")
	}
}

type testGamepads struct{ pads []GamepadFrame }

func (b *testGamepads) drain() []GamepadFrame { return b.pads }
func (*testGamepads) close()                  {}

func TestGamepadSelectionSurvivesEnumerationChanges(t *testing.T) {
	backend := &testGamepads{pads: []GamepadFrame{{ID: "one"}, {ID: "two"}}}
	source := &GamepadSource{backend: backend}
	if source.DrainFrame().ID != "one" {
		t.Fatal("did not select the first controller")
	}
	backend.pads[0], backend.pads[1] = backend.pads[1], backend.pads[0]
	if source.DrainFrame().ID != "one" {
		t.Fatal("enumeration reordered the active controller")
	}
	backend.pads = backend.pads[:1]
	if source.DrainFrame().ID != "two" {
		t.Fatal("did not switch after disconnect")
	}
	backend.pads = nil
	if source.DrainFrame().ID != "" {
		t.Fatal("last controller remained connected")
	}
}

func TestGamepadFrameReadersDoNotConsumeChanges(t *testing.T) {
	AndroidResetGamepads()
	t.Cleanup(AndroidResetGamepads)
	AndroidGamepadDevice(1, "controller", true)
	source := &GamepadSource{backend: &androidGamepadBackend{}}
	source.DrainFrame()
	AndroidGamepadKey(1, 108, true)
	AndroidGamepadKey(1, 108, false)
	frame := source.DrainFrame()
	state := NewState()
	state.SetGamepad(frame)
	// UI, Lua and diagnostics can all read the same frame's edges.
	for range 3 {
		if !state.GamepadJustPressed(GamepadStart) || !state.GamepadJustReleased(GamepadStart) || len(state.GamepadChanges()) != 2 {
			t.Fatal("a state reader consumed frame transitions")
		}
	}
	if next := source.DrainFrame(); len(next.Changes) != 0 || next.ID != frame.ID {
		t.Fatal("draining replayed changes or lost current controller state")
	}
	AndroidGamepadKey(1, 108, true)
	source.DiscardPending()
	next := source.DrainFrame()
	if len(next.Changes) != 0 || !next.Buttons[GamepadStart] {
		t.Fatal("discard must drop actions while retaining current held state")
	}
	if len(frame.Changes) != 2 || frame.Changes[1].Down {
		t.Fatal("later input mutated an already drained frame")
	}
}
