package render

import (
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
)

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
	filtered.suppressShortcutText = func(code input.KeyCode) bool {
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
	if uiText != "" || state.TextInput() != "" {
		t.Fatal("Alt shortcut inserted text into an editor")
	}
	if !state.KeyCodeJustPressed(gpucontext.KeyM) {
		t.Fatal("filter lost physical key events")
	}
	state.ConsumeKeyCodePress(gpucontext.KeyM)
	state.EndFrame()
	press(gpucontext.KeyM, gpucontext.ModAlt)
	typeText(",")
	if uiText != "" || state.TextInput() != "" || state.KeyCodeJustPressed(gpucontext.KeyM) {
		t.Fatal("held shortcut leaked text or retriggered its press")
	}
	active = false // The active mode or focused form does not handle this key.
	typeText("µ")
	active = true
	press(gpucontext.Key2, gpucontext.ModAlt)
	typeText("™") // An unbound physical key is not filtered either.
	press(gpucontext.KeyM, gpucontext.ModAlt)
	for _, fn := range source.keyRelease {
		fn(gpucontext.KeyM, gpucontext.ModAlt)
	}
	typeText("released")
	if uiText != "µ™released" || state.TextInput() != uiText {
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

func TestWireInputAltTabWithoutKeyRelease(t *testing.T) {
	for _, name := range []string{"AltLeft", "AltRight"} {
		t.Run(name, func(t *testing.T) {
			events := &fanoutEventSource{}
			state := input.NewState()
			wireInput(events, state)
			alt, ok := input.KeyCodeFromName(name)
			if !ok {
				t.Fatalf("unknown key code %q", name)
			}
			for _, fn := range events.keyPress {
				fn(alt, gpucontext.ModAlt)
				fn(gpucontext.KeyTab, gpucontext.ModAlt)
			}
			state.EndFrame()

			// Alt is released in the other application, so Goro receives no
			// key release between losing and regaining keyboard focus.
			for _, fn := range events.focus {
				fn(false)
			}
			if state.Pressed(input.KeyAlt) || state.KeyCodeDown(alt) || state.Pressed(input.KeyTab) {
				t.Fatal("keys remained held after focus loss")
			}
			for _, fn := range events.focus {
				fn(true)
			}
			for _, fn := range events.mousePress {
				fn(gpucontext.MouseButtonLeft, 400, 300)
			}
			if state.Pressed(input.KeyAlt) || !state.MouseJustPressed(input.MouseButtonLeft) {
				t.Fatal("first click after Alt+Tab was not an ordinary left click")
			}

			// Genuine Alt shortcuts must still work after returning.
			for _, fn := range events.keyPress {
				fn(alt, gpucontext.ModAlt)
			}
			if !state.JustPressed(input.KeyAlt) || !state.KeyCodeJustPressed(alt) {
				t.Fatal("fresh Alt press was not recognized")
			}
			for _, fn := range events.keyRelease {
				fn(alt, 0)
			}
			if state.Pressed(input.KeyAlt) || !state.KeyCodeJustReleased(alt) {
				t.Fatal("fresh Alt release was not recognized")
			}
		})
	}
}
