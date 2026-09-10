package input

import (
	"testing"

	"github.com/gogpu/gpucontext"
)

func TestFrameIDAdvancesOnlyAtEndFrame(t *testing.T) {
	state := NewState()
	frame := state.FrameID()
	state.SetKey(KeyEnter, true)
	state.ResetKeyboard()
	if state.FrameID() != frame {
		t.Fatal("input events changed the frame ID")
	}
	state.EndFrame()
	if state.FrameID() != frame+1 {
		t.Fatal("EndFrame did not advance the frame ID")
	}
}

func TestResetKeyboard(t *testing.T) {
	for _, heldAcrossFrame := range []bool{false, true} {
		name := "pending presses"
		if heldAcrossFrame {
			name = "held keys"
		}
		t.Run(name, func(t *testing.T) {
			state := NewState()
			codes := []KeyCode{
				gpucontext.KeyLeftAlt, gpucontext.KeyRightAlt,
				gpucontext.KeyLeftControl, gpucontext.KeyLeftShift,
				gpucontext.KeyTab, gpucontext.KeyW, gpucontext.KeyG,
			}
			for _, code := range codes {
				state.SetKeyCode(code, true)
			}
			state.SetKey(KeyL, true)
			if heldAcrossFrame {
				state.EndFrame()
			}
			state.SetKeyCode(gpucontext.KeyG, false)
			state.AddTextInput("g")
			state.SetMousePosition(10, 20)
			state.SetMouseButton(MouseButtonLeft, true)

			state.ResetKeyboard()

			for _, code := range codes {
				if state.KeyCodeDown(code) || state.KeyCodeJustPressed(code) || state.KeyCodeJustReleased(code) {
					t.Errorf("key %v retained held state or an edge", code)
				}
			}
			for _, key := range []Key{KeyAlt, KeyCtrl, KeyShift, KeyTab, KeyW, KeyG, KeyL} {
				if state.Pressed(key) || state.JustPressed(key) {
					t.Errorf("legacy key %v retained held state or a press", key)
				}
			}
			if state.TextInput() != "" {
				t.Fatal("pending text input survived keyboard reset")
			}
			if state.MouseX != 10 || state.MouseY != 20 || !state.MouseJustPressed(MouseButtonLeft) || !state.MousePressed(MouseButtonLeft) {
				t.Fatal("keyboard reset modified pointer state")
			}
			state.EndFrame()
			state.SetKeyCode(gpucontext.KeyLeftAlt, false)
			if state.KeyCodeJustReleased(gpucontext.KeyLeftAlt) {
				t.Fatal("late key release created an edge for a canceled press")
			}
		})
	}
}

func TestTouchDistance(t *testing.T) {
	got := touchDistance(TouchPoint{X: 10, Y: 20}, TouchPoint{X: 13, Y: 24})
	if got != 5 {
		t.Fatalf("touch distance = %.1f, want 5.0", got)
	}
}

func TestMouseJustReleased(t *testing.T) {
	state := NewState()
	state.SetMouseButton(MouseButtonLeft, true)
	state.EndFrame()

	state.SetMouseButton(MouseButtonLeft, false)
	if !state.MouseJustReleased(MouseButtonLeft) {
		t.Fatal("MouseJustReleased = false, want true")
	}

	state.EndFrame()
	if state.MouseJustReleased(MouseButtonLeft) {
		t.Fatal("MouseJustReleased persisted after EndFrame")
	}
}

func TestConsumeKeyCodePressPreservesHeldStateAndClaimsLegacyShortcut(t *testing.T) {
	state := NewState()
	code, ok := KeyCodeFromName("KeyW")
	if !ok {
		t.Fatal("KeyW was not recognized")
	}
	state.SetKeyCode(code, true)

	if !state.ConsumeKeyCodePress(code) {
		t.Fatal("ConsumeKeyCodePress = false, want true")
	}
	if state.KeyCodeJustPressed(code) {
		t.Fatal("physical press remained visible after consumption")
	}
	if state.JustPressed(KeyW) {
		t.Fatal("legacy shortcut press remained visible after consumption")
	}
	if !state.KeyCodeDown(code) {
		t.Fatal("held state was cleared after consuming key edge")
	}
	if state.ConsumeKeyCodePress(code) {
		t.Fatal("second ConsumeKeyCodePress = true, want false")
	}

	state.EndFrame()
	state.SetKeyCode(code, false)
	if !state.KeyCodeJustReleased(code) {
		t.Fatal("physical key release was not reported")
	}
}

func TestKeyCodeNamesCoverPhysicalAndNonTextKeys(t *testing.T) {
	for _, name := range []string{"KeyA", "KeyZ", "Digit0", "F12", "ArrowUp", "ControlRight", "NumpadEnter"} {
		if _, ok := KeyCodeFromName(name); !ok {
			t.Fatalf("KeyCodeFromName(%q) = false", name)
		}
	}
	if _, ok := KeyCodeFromName("a"); ok {
		t.Fatal("layout-dependent glyph unexpectedly resolved as a key code")
	}
}
