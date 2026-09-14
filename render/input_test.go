package render

import (
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
)

func TestFanoutAltShortcutsDoNotEmitText(t *testing.T) {
	source := &fanoutEventSource{}
	filtered := newFanoutEventSource(source)
	state := input.NewState()
	wireInput(filtered, state)
	var uiText string
	filtered.OnTextInput(func(text string) { uiText += text })
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
	press(gpucontext.Key1, gpucontext.ModAlt)
	typeText("&") // AZERTY physical Digit1
	press(gpucontext.KeyM, gpucontext.ModAlt)
	typeText("m")
	if uiText != "" || state.TextInput() != "" {
		t.Fatal("Alt shortcut inserted text into an editor")
	}
	if !state.KeyCodeJustPressed(gpucontext.Key1) || !state.KeyCodeJustPressed(gpucontext.KeyM) {
		t.Fatal("filter lost physical key events")
	}
	for _, fn := range source.keyRelease {
		fn(gpucontext.KeyLeftAlt, 0)
	}
	typeText("ordinary")
	press(gpucontext.KeyRightAlt, gpucontext.ModAlt)
	press(gpucontext.Key0, gpucontext.ModAlt)
	typeText("@")
	for _, fn := range source.keyRelease {
		fn(gpucontext.KeyRightAlt, 0)
	}
	press(gpucontext.Key0, gpucontext.ModAlt|gpucontext.ModControl)
	typeText("#")
	if uiText != "ordinary@#" || state.TextInput() != uiText {
		t.Fatalf("ordinary/AltGr text was lost: %q / %q", uiText, state.TextInput())
	}
	press(gpucontext.KeyM, gpucontext.ModAlt)
	for _, fn := range source.focus {
		fn(false)
		fn(true)
	}
	typeText("returned")
	if uiText != "ordinary@#returned" || state.TextInput() != "returned" {
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
