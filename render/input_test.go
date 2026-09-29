package render

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
)

func TestFanoutPreparesEditingKeysBeforeDispatch(t *testing.T) {
	source := &fanoutEventSource{}
	filtered := newFanoutEventSource(source)
	state := input.NewState()
	var order []string
	filtered.prepareKeyInput = func(code input.KeyCode, mods gpucontext.Modifiers) {
		if code != gpucontext.KeyDelete || mods != gpucontext.ModControl {
			t.Fatal("key preparation lost the physical key or modifiers")
		}
		order = append(order, "prepare")
		if !state.KeyCodeJustPressed(code) {
			t.Fatal("key preparation ran before physical input was recorded")
		}
	}
	filtered.OnKeyPress(func(gpucontext.Key, gpucontext.Modifiers) {
		order = append(order, "ui")
	})
	wireInput(filtered, state)
	for _, fn := range source.keyPress {
		fn(gpucontext.KeyDelete, gpucontext.ModControl)
	}
	if !reflect.DeepEqual(order, []string{"prepare", "ui"}) || !state.KeyCodeJustPressed(gpucontext.KeyDelete) {
		t.Fatalf("editing key dispatch order=%v; physical press must remain available", order)
	}
	for _, fn := range source.keyRelease {
		fn(gpucontext.KeyDelete, 0)
	}
	if len(order) != 2 || state.KeyCodeDown(gpucontext.KeyDelete) {
		t.Fatal("release prepared focus or failed to release the key")
	}
}

func TestFanoutPreservesTextWithoutShortcutFilter(t *testing.T) {
	source := &fanoutEventSource{}
	filtered := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(filtered, state)
	var uiText string
	filtered.OnTextInput(func(text string) { uiText += text })
	for _, fn := range source.keyPress {
		fn(gpucontext.KeyLeftAlt, gpucontext.ModAlt)
		fn(gpucontext.Key2, gpucontext.ModAlt)
	}
	for _, fn := range source.textInput {
		fn("™") // Option+2 on macOS, including on login screens.
	}
	if uiText != "™" || state.TextInput() != uiText {
		t.Fatalf("Option text was lost: %q / %q", uiText, state.TextInput())
	}
}

func TestFanoutFiltersOnlyActiveShortcutText(t *testing.T) {
	source := &fanoutEventSource{}
	filtered := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(filtered, state)
	var uiText string
	filtered.OnTextInput(func(text string) { uiText += text })
	active := true
	filtered.prepareTextInput = func(code input.KeyCode) bool {
		return active && state.Pressed(input.KeyAlt) && code == gpucontext.KeyM
	}
	press := func(key gpucontext.Key, mods gpucontext.Modifiers) {
		for _, fn := range source.keyPress {
			fn(key, mods)
		}
	}
	typeText := func(text string) {
		for _, fn := range source.textInput {
			fn(text)
		}
	}
	press(gpucontext.KeyLeftAlt, gpucontext.ModAlt)
	press(gpucontext.KeyM, gpucontext.ModAlt)
	typeText(",") // AZERTY physical M; glyph does not affect the binding.
	if uiText != "" || state.TextInput() != "," {
		t.Fatal("Alt shortcut inserted text into an editor")
	}
	if !state.KeyCodeJustPressed(gpucontext.KeyM) {
		t.Fatal("filter lost physical key events")
	}
	state.ConsumeKeyCodePress(gpucontext.KeyM)
	state.EndFrame()
	press(gpucontext.KeyM, gpucontext.ModAlt)
	typeText(",")
	if uiText != "" || state.TextInput() != "," || state.KeyCodeJustPressed(gpucontext.KeyM) {
		t.Fatal("held shortcut leaked text or retriggered its press")
	}
	active = false // The active mode or focused form does not handle this key.
	// The consumed press remains owned until release. A new press in the
	// focused form must be delivered normally.
	for _, fn := range source.keyRelease {
		fn(gpucontext.KeyM, gpucontext.ModAlt)
	}
	press(gpucontext.KeyM, gpucontext.ModAlt)
	typeText("µ")
	active = true
	press(gpucontext.Key2, gpucontext.ModAlt)
	typeText("™") // An unbound physical key is not filtered either.
	press(gpucontext.KeyM, gpucontext.ModAlt)
	for _, fn := range source.keyRelease {
		fn(gpucontext.KeyM, gpucontext.ModAlt)
	}
	typeText("released")
	if uiText != "µ™released" || state.TextInput() != ",µ™released" {
		t.Fatalf("non-shortcut text was lost: %q / %q", uiText, state.TextInput())
	}
	press(gpucontext.KeyM, gpucontext.ModAlt)
	for _, fn := range source.focus {
		fn(false)
		fn(true)
	}
	typeText("returned")
	if uiText != "µ™releasedreturned" || state.TextInput() != "returned" {
		t.Fatalf("text after focus loss = %q / %q", uiText, state.TextInput())
	}
}

func TestFanoutConsumedPressSuppressesUIAndTextUntilRelease(t *testing.T) {
	source := &fanoutEventSource{}
	filtered := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(filtered, state)
	claim := true
	calls, prepared, uiKeys := 0, 0, 0
	var uiText string
	filtered.handleKeyPress = func(code input.KeyCode) {
		calls++
		if !state.KeyCodeDown(code) || !state.KeyCodeJustPressed(code) {
			t.Fatal("handler did not receive the fresh physical press")
		}
		if claim && code == gpucontext.KeyW {
			state.ConsumeKeyCodePress(code)
		}
	}
	filtered.prepareKeyInput = func(input.KeyCode, gpucontext.Modifiers) { prepared++ }
	filtered.prepareTextInput = func(input.KeyCode) bool { prepared++; return false }
	filtered.OnKeyPress(func(gpucontext.Key, gpucontext.Modifiers) { uiKeys++ })
	filtered.OnTextInput(func(text string) { uiText += text })
	press := func(code gpucontext.Key, text string) {
		for _, fn := range source.keyPress {
			fn(code, 0)
		}
		for _, fn := range source.textInput {
			fn(text)
		}
	}
	press(gpucontext.KeyW, "z") // AZERTY: consume by position, not glyph.
	press(gpucontext.KeyW, "z") // Native repeat within the same frame.
	state.EndFrame()
	press(gpucontext.KeyW, "z") // Repeat in a later frame.
	if calls != 1 || prepared != 0 || uiKeys != 0 || uiText != "" || state.TextInput() != "z" {
		t.Fatalf("consumed key leaked: calls=%d prepared=%d keys=%d text=%q", calls, prepared, uiKeys, uiText)
	}
	if !state.KeyCodeDown(gpucontext.KeyW) || state.JustPressed(input.KeyW) {
		t.Fatal("consumption lost held state or left a native shortcut press")
	}
	press(gpucontext.KeyE, "é")
	if calls != 2 || prepared != 2 || uiKeys != 1 || uiText != "é" {
		t.Fatal("unconsumed key did not reach default key/text handling")
	}
	for _, fn := range source.keyRelease {
		fn(gpucontext.KeyW, 0)
	}
	if state.KeyCodeDown(gpucontext.KeyW) || !state.KeyCodeJustReleased(gpucontext.KeyW) {
		t.Fatal("consumed key did not release")
	}
	claim = false
	press(gpucontext.KeyW, "z")
	if calls != 3 || uiText != "éz" || !state.KeyCodeJustPressed(gpucontext.KeyW) {
		t.Fatal("new unclaimed press inherited the old consumption")
	}
	for _, fn := range source.keyRelease {
		fn(gpucontext.KeyW, 0)
	}
	claim = true
	press(gpucontext.KeyW, "z")
	for _, fn := range source.focus {
		fn(false)
		fn(true)
	}
	for _, fn := range source.textInput {
		fn("漢字") // Committed text with no associated physical key.
	}
	if state.KeyCodeConsumed(gpucontext.KeyW) || uiText != "éz漢字" {
		t.Fatal("focus loss retained consumption or lost unassociated text")
	}
}

func TestFanoutPreparationCanConsumeOpeningEnter(t *testing.T) {
	source := &fanoutEventSource{}
	filtered := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(filtered, state)
	prepared := 0
	filtered.prepareKeyInput = func(code input.KeyCode, _ gpucontext.Modifiers) {
		prepared++
		state.ConsumeKeyCodePress(code)
	}
	filtered.OnKeyPress(func(gpucontext.Key, gpucontext.Modifiers) {
		t.Fatal("opening Enter also reached the newly focused field")
	})
	for _, fn := range source.keyPress {
		fn(gpucontext.KeyEnter, 0)
		fn(gpucontext.KeyEnter, 0)
	}
	if prepared != 1 || state.JustPressed(input.KeyEnter) {
		t.Fatal("opening Enter was prepared twice or remained a game action")
	}
}

func TestWireInputAltTabWithoutKeyRelease(t *testing.T) {
	for _, name := range []string{"AltLeft", "AltRight"} {
		t.Run(name, func(t *testing.T) {
			source := &fanoutEventSource{}
			events := newFanoutEventSource(source)
			state := input.NewState()
			wireInput(events, state)
			alt, ok := input.KeyCodeFromName(name)
			if !ok {
				t.Fatalf("unknown key code %q", name)
			}
			for _, fn := range source.keyPress {
				fn(alt, gpucontext.ModAlt)
				fn(gpucontext.KeyTab, gpucontext.ModAlt)
			}
			state.EndFrame()

			// Alt is released in the other application, so Goro receives no
			// key release between losing and regaining keyboard focus.
			for _, fn := range source.focus {
				fn(false)
			}
			if state.Pressed(input.KeyAlt) || state.KeyCodeDown(alt) || state.Pressed(input.KeyTab) {
				t.Fatal("keys remained held after focus loss")
			}
			for _, fn := range source.focus {
				fn(true)
			}
			for _, fn := range source.mousePress {
				fn(gpucontext.MouseButtonLeft, 400, 300)
			}
			if state.Pressed(input.KeyAlt) || !state.MouseJustPressed(input.MouseButtonLeft) {
				t.Fatal("first click after Alt+Tab was not an ordinary left click")
			}

			// Genuine Alt shortcuts must still work after returning.
			for _, fn := range source.keyPress {
				fn(alt, gpucontext.ModAlt)
			}
			if !state.JustPressed(input.KeyAlt) || !state.KeyCodeJustPressed(alt) {
				t.Fatal("fresh Alt press was not recognized")
			}
			for _, fn := range source.keyRelease {
				fn(alt, 0)
			}
			if state.Pressed(input.KeyAlt) || !state.KeyCodeJustReleased(alt) {
				t.Fatal("fresh Alt release was not recognized")
			}
		})
	}
}

func TestPhysicalAndControllerEscapeShareDispatch(t *testing.T) {
	for _, synthetic := range []bool{false, true} {
		for _, consumeAt := range []string{"", "handler", "prepare"} {
			t.Run(fmt.Sprintf("synthetic=%v/consume=%s", synthetic, consumeAt), func(t *testing.T) {
				source := &fanoutEventSource{}
				events := newFanoutEventSource(source)
				state := input.NewState()
				wireInput(events, state)
				var order []string
				events.handleKeyPress = func(key input.KeyCode) {
					if !state.KeyCodeDown(key) || !state.KeyCodeJustPressed(key) {
						t.Fatal("handler preceded state update")
					}
					order = append(order, "handler")
					if consumeAt == "handler" {
						state.ConsumeKeyCodePress(key)
					}
				}
				events.prepareKeyInput = func(key input.KeyCode, _ gpucontext.Modifiers) {
					order = append(order, "prepare")
					if consumeAt == "prepare" {
						state.ConsumeKeyCodePress(key)
					}
				}
				events.OnKeyPress(func(gpucontext.Key, gpucontext.Modifiers) { order = append(order, "ui") })
				events.OnKeyRelease(func(key gpucontext.Key, _ gpucontext.Modifiers) {
					if state.KeyCodeDown(key) {
						t.Fatal("release preceded state update")
					}
					order = append(order, "release")
				})
				if synthetic {
					pad := input.GamepadFrame{ID: "controller"}
					pad.Buttons[input.GamepadStart] = true
					state.SetGamepad(pad)
					(&gamepadUIState{}).update(state, events, time.Now(), 300, 200)
				} else {
					for _, fn := range source.keyPress {
						fn(gpucontext.KeyEscape, 0)
					}
					for _, fn := range source.keyRelease {
						fn(gpucontext.KeyEscape, 0)
					}
				}
				want := []string{"handler", "prepare", "ui", "release"}
				if consumeAt == "handler" {
					want = []string{"handler", "release"}
				}
				if consumeAt == "prepare" {
					want = []string{"handler", "prepare", "release"}
				}
				if !reflect.DeepEqual(order, want) {
					t.Fatalf("dispatch = %v, want %v", order, want)
				}
			})
		}
	}
}

func TestControllerTapPreservesPhysicalKeysAndTextAssociation(t *testing.T) {
	source := &fanoutEventSource{}
	events := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(events, state)
	events.handleKeyPress = func(key input.KeyCode) {
		if key == gpucontext.KeyM {
			state.ConsumeKeyCodePress(key)
		}
	}
	for _, fn := range source.keyPress {
		fn(gpucontext.KeyM, 0)
	}
	events.tapKey(gpucontext.KeyEscape)
	events.OnTextInput(func(string) { t.Fatal("synthetic key changed physical shortcut/text association") })
	for _, fn := range source.textInput {
		fn("m")
	}
	for _, fn := range source.keyPress {
		fn(gpucontext.KeyEscape, 0)
	}
	state.EndFrame()
	events.tapKey(gpucontext.KeyEscape)
	if !state.KeyCodeDown(gpucontext.KeyEscape) || state.KeyCodeJustReleased(gpucontext.KeyEscape) || state.KeyCodeJustPressed(gpucontext.KeyEscape) {
		t.Fatal("controller tap changed a physically held Escape key")
	}
}
