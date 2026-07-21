package ui

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestConsoleNoShiftCommandTogglesSessionPreference(t *testing.T) {
	console := &ChatConsole{input: "/ns", active: true}
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	if !console.SubmitCommand(ctx, "/ns") {
		t.Fatal("noshift command was not handled")
	}
	if !sessionState.NoShift {
		t.Fatal("noshift was not enabled")
	}
	if console.active || console.input != "" {
		t.Fatalf("console active=%t input=%q, want closed empty input", console.active, console.input)
	}

	if !console.SubmitCommand(ctx, "/noshift") {
		t.Fatal("noshift command was not handled")
	}
	if sessionState.NoShift {
		t.Fatal("noshift was not disabled")
	}
}

func TestConsoleNoCtrlCommandTogglesSessionPreference(t *testing.T) {
	console := &ChatConsole{input: "/nc", active: true}
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	if !console.SubmitCommand(ctx, "/nc") {
		t.Fatal("noctrl command was not handled")
	}
	if !sessionState.NoCtrl {
		t.Fatal("noctrl was not enabled")
	}

	if !console.SubmitCommand(ctx, "/noctrl") {
		t.Fatal("noctrl command was not handled")
	}
	if sessionState.NoCtrl {
		t.Fatal("noctrl was not disabled")
	}
}

func TestConsoleMineffectCommandTogglesSessionPreference(t *testing.T) {
	console := &ChatConsole{input: "/mineffect", active: true}
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	if !console.SubmitCommand(ctx, "/mineffect") {
		t.Fatal("mineffect command was not handled")
	}
	if !sessionState.LessEffects {
		t.Fatal("less effects was not enabled")
	}
	if console.active || console.input != "" {
		t.Fatalf("console active=%t input=%q, want closed empty input", console.active, console.input)
	}

	if !console.SubmitCommand(ctx, "/mineffect") {
		t.Fatal("mineffect command was not handled")
	}
	if sessionState.LessEffects {
		t.Fatal("less effects was not disabled")
	}
}

func TestConsoleCompanionAICommandsToggleSessionPreference(t *testing.T) {
	console := &ChatConsole{active: true}
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	if !console.SubmitCommand(ctx, "/hoai") {
		t.Fatal("hoai command was not handled")
	}
	if !sessionState.HomunculusCustomAI {
		t.Fatal("homunculus custom AI was not enabled")
	}
	if console.active || console.input != "" {
		t.Fatalf("console active=%t input=%q, want closed empty input", console.active, console.input)
	}

	if !console.SubmitCommand(ctx, "/merai") {
		t.Fatal("merai command was not handled")
	}
	if !sessionState.MercenaryCustomAI {
		t.Fatal("mercenary custom AI was not enabled")
	}

	if !console.SubmitCommand(ctx, "/hoai") {
		t.Fatal("hoai command was not handled")
	}
	if sessionState.HomunculusCustomAI {
		t.Fatal("homunculus custom AI was not disabled")
	}
}

func TestConsoleMemoCommandWithoutNetwork(t *testing.T) {
	console := &ChatConsole{input: "/memo", active: true}

	if !console.SubmitCommand(client.Context{}, "/memo") {
		t.Fatal("memo command was not handled")
	}
	if console.active || console.input != "" {
		t.Fatalf("console active=%t input=%q, want closed empty input", console.active, console.input)
	}
	if len(console.messages) != 1 || console.messages[0].Text != "send failed: not connected" {
		t.Fatalf("console messages = %+v", console.messages)
	}
}

func TestConsoleScreenshotCommandRequestsCapture(t *testing.T) {
	console := &ChatConsole{input: "/screenshot", active: true}
	requested := false
	ctx := client.Context{
		RequestScreenshot: func() (string, error) {
			requested = true
			return "/tmp/goro-test.png", nil
		},
	}

	if !console.SubmitCommand(ctx, "/screenshot") {
		t.Fatal("screenshot command was not handled")
	}
	if !requested {
		t.Fatal("screenshot was not requested")
	}
	if console.active || console.input != "" {
		t.Fatalf("console active=%t input=%q, want closed empty input", console.active, console.input)
	}
	if len(console.messages) != 1 || console.messages[0].Text != "Screenshot: /tmp/goro-test.png" {
		t.Fatalf("console messages = %+v", console.messages)
	}
}

func TestConsoleInputHistoryUsesArrowKeys(t *testing.T) {
	console := &ChatConsole{input: "draft", active: true}
	console.rememberInput("/sit")
	console.rememberInput("hello")

	inputState := input.NewState()
	inputState.SetKey(input.KeyArrowUp, true)
	if !console.Update(client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}) {
		t.Fatal("up key was not handled")
	}
	if console.input != "hello" {
		t.Fatalf("first history input = %q, want hello", console.input)
	}

	inputState = input.NewState()
	inputState.SetKey(input.KeyArrowUp, true)
	console.Update(client.Context{Input: inputState, ScreenW: 800, ScreenH: 600})
	if console.input != "/sit" {
		t.Fatalf("second history input = %q, want /sit", console.input)
	}

	inputState = input.NewState()
	inputState.SetKey(input.KeyArrowUp, true)
	console.Update(client.Context{Input: inputState, ScreenW: 800, ScreenH: 600})
	if console.input != "/sit" {
		t.Fatalf("oldest history input = %q, want /sit", console.input)
	}

	inputState = input.NewState()
	inputState.SetKey(input.KeyArrowDown, true)
	console.Update(client.Context{Input: inputState, ScreenW: 800, ScreenH: 600})
	if console.input != "hello" {
		t.Fatalf("newer history input = %q, want hello", console.input)
	}

	inputState = input.NewState()
	inputState.SetKey(input.KeyArrowDown, true)
	console.Update(client.Context{Input: inputState, ScreenW: 800, ScreenH: 600})
	if console.input != "draft" {
		t.Fatalf("restored draft input = %q, want draft", console.input)
	}
}

func TestConsoleOutsideClickBlursAndPassesThrough(t *testing.T) {
	console := &ChatConsole{input: "hello", active: true}
	inputState := input.NewState()
	inputState.MouseX = 700
	inputState.MouseY = 100
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if console.Update(client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}) {
		t.Fatal("outside click was consumed")
	}
	if console.active {
		t.Fatal("console stayed active after outside click")
	}
	if console.input != "hello" {
		t.Fatalf("input = %q, want preserved draft", console.input)
	}
}

func TestConsoleTypingAndRefocusScrollToBottom(t *testing.T) {
	console := &ChatConsole{}
	for i := 0; i < 20; i++ {
		console.AddMessage("line %d", i)
	}
	console.widgetTree(480, 176)
	bottom := console.ensureScrollSignal().Get()
	if bottom <= 0 {
		t.Fatalf("bottom scroll = %f, want positive", bottom)
	}
	console.ensureScrollSignal().Set(0)
	console.setInput("hello")
	if got := console.ensureScrollSignal().Get(); got != bottom {
		t.Fatalf("typing scroll = %f, want %f", got, bottom)
	}
	console.ensureScrollSignal().Set(0)
	console.setActive(true)
	if got := console.ensureScrollSignal().Get(); got != bottom {
		t.Fatalf("refocus scroll = %f, want %f", got, bottom)
	}
}

func TestConsoleBottomScrollUsesRenderedLineHeight(t *testing.T) {
	lines := 80
	viewportHeight := 132

	got := consoleBottomScrollY(lines, viewportHeight)
	want := float32(lines*consoleLineH + (lines - 1) - viewportHeight)
	if got != want {
		t.Fatalf("bottom scroll = %f, want %f", got, want)
	}

	oldEstimate := float32(lines*11 + (lines - 1) - viewportHeight)
	if oldEstimate/got > 0.85 {
		t.Fatalf("old estimate ratio = %.2f, want visibly below real bottom", oldEstimate/got)
	}
}
