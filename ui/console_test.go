package ui

import (
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
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

func TestConsoleDiscardTextInputRemovesShortcutRune(t *testing.T) {
	console := &ChatConsole{input: "draftg"}

	console.DiscardTextInput("g")

	if console.input != "draft" {
		t.Fatalf("console input = %q, want draft", console.input)
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
	if console.active || console.input != "" {
		t.Fatalf("console active=%t input=%q, want closed empty input", console.active, console.input)
	}
}

func TestConsoleBreakGuildCommandIsHandledLocally(t *testing.T) {
	console := &ChatConsole{input: `/breakguild "Mandala"`, active: true}

	if !console.SubmitCommand(client.Context{}, `/breakguild "Mandala"`) {
		t.Fatal("breakguild command was not handled")
	}
	if console.active || console.input != "" {
		t.Fatalf("console active=%t input=%q, want closed empty input", console.active, console.input)
	}
	if len(console.messages) != 1 || console.messages[0].Text != "send failed: not connected" {
		t.Fatalf("console messages = %+v", console.messages)
	}
}

func TestConsoleFameRankingCommandsWithoutNetwork(t *testing.T) {
	for _, command := range []string{"/blacksmith", "/alchemist", "/taekwon"} {
		console := &ChatConsole{input: command, active: true}

		if !console.SubmitCommand(client.Context{}, command) {
			t.Fatalf("%s command was not handled", command)
		}
		if console.active || console.input != "" {
			t.Fatalf("%s console active=%t input=%q, want closed empty input", command, console.active, console.input)
		}
		if len(console.messages) != 1 || console.messages[0].Text != "send failed: not connected" {
			t.Fatalf("%s console messages=%+v", command, console.messages)
		}
	}
}

func TestConsoleDoriDoriCommandWithoutNetwork(t *testing.T) {
	console := &ChatConsole{input: "/doridori", active: true}
	world := worldstate.New()

	if !console.SubmitCommand(client.Context{World: world}, "/doridori") {
		t.Fatal("doridori command was not handled")
	}
	if world.Player.HeadDir != 0 {
		t.Fatalf("head direction = %d after failed send, want unchanged", world.Player.HeadDir)
	}
	if len(console.messages) != 1 || console.messages[0].Text != "send failed: not connected" {
		t.Fatalf("console messages = %+v", console.messages)
	}
}

func TestDoriDoriHeadDirectionAlternates(t *testing.T) {
	if got := nextDoriDoriHeadDir(0); got != 1 {
		t.Fatalf("center head direction became %d, want 1", got)
	}
	if got := nextDoriDoriHeadDir(1); got != 2 {
		t.Fatalf("left head direction became %d, want 2", got)
	}
	if got := nextDoriDoriHeadDir(2); got != 1 {
		t.Fatalf("right head direction became %d, want 1", got)
	}
}

func TestDoriDoriRecoveryTiming(t *testing.T) {
	start := time.Unix(100, 0)
	var turns [doriDoriTurns]time.Time
	for i := 0; i < doriDoriTurns-1; i++ {
		if recordDoriDoriTurn(&turns, start.Add(time.Duration(i)*500*time.Millisecond)) {
			t.Fatalf("recovery triggered after %d turns", i+1)
		}
	}
	if !recordDoriDoriTurn(&turns, start.Add(2*time.Second)) {
		t.Fatal("five turns spanning two seconds did not trigger recovery")
	}

	for _, span := range []time.Duration{1500 * time.Millisecond, 3 * time.Second} {
		turns = [doriDoriTurns]time.Time{}
		for i := 0; i < doriDoriTurns; i++ {
			at := start.Add(time.Duration(i) * span / (doriDoriTurns - 1))
			if got := recordDoriDoriTurn(&turns, at); got {
				t.Fatalf("boundary span %s triggered recovery", span)
			}
		}
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

func TestConsoleSubmitConsumesClosingEnterFrame(t *testing.T) {
	console := &ChatConsole{input: "/ns", active: true}
	inputState := input.NewState()
	inputState.SetKey(input.KeyEnter, true)
	sessionState := &session.Session{}

	if !console.Update(client.Context{Input: inputState, ScreenW: 800, ScreenH: 600, UIManager: NewManager(), Session: sessionState}) {
		t.Fatal("closing submit enter frame was not consumed")
	}
	if !sessionState.NoShift {
		t.Fatal("command was not submitted")
	}
	if console.Active() || console.input != "" {
		t.Fatalf("console active=%t input=%q, want closed empty input", console.Active(), console.input)
	}
}

func TestConsoleTextFieldSubmitRemembersSubmittedWidgetText(t *testing.T) {
	console := &ChatConsole{input: "stale", active: true}
	field := console.inputWidget()
	field.SetText("  @jump 47 104  ")

	if !field.Event(nil, event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone)) {
		t.Fatal("enter key was not handled by text field")
	}
	console.UpdateInput(client.Context{})

	if len(console.history) != 1 || console.history[0] != "@jump 47 104" {
		t.Fatalf("history = %#v, want submitted @jump command", console.history)
	}
	if console.input != "  @jump 47 104  " {
		t.Fatalf("input = %q, want submitted text preserved after failed send", console.input)
	}
}

func TestConsoleFrameSubmitRemembersCurrentWidgetText(t *testing.T) {
	console := &ChatConsole{input: "stale", active: true}
	field := console.inputWidget()
	field.SetText("  @heal  ")
	inputState := input.NewState()
	inputState.SetKey(input.KeyEnter, true)

	if !console.Update(client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}) {
		t.Fatal("enter key was not handled by console")
	}

	if len(console.history) != 1 || console.history[0] != "@heal" {
		t.Fatalf("history = %#v, want submitted @heal command", console.history)
	}
	if console.input != "  @heal  " {
		t.Fatalf("input = %q, want submitted text preserved after failed send", console.input)
	}
}

func TestConsoleWidgetAndFrameSubmitOnce(t *testing.T) {
	sessionState := &session.Session{}
	ctx := client.Context{ScreenW: 800, ScreenH: 600, UIManager: NewManager(), Session: sessionState}
	console := &ChatConsole{active: true, ctx: ctx}
	field := console.inputWidget()
	field.SetText("/ns")

	if !field.Event(nil, event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone)) {
		t.Fatal("enter key was not handled by text field")
	}

	inputState := input.NewState()
	inputState.SetKey(input.KeyEnter, true)
	ctx.Input = inputState
	if !console.Update(ctx) {
		t.Fatal("submitted enter frame was not consumed")
	}
	if !sessionState.NoShift || console.Active() {
		t.Fatalf("after frame handling no_shift=%t active=%t, want one submit and closed console", sessionState.NoShift, console.Active())
	}

	inputState.EndFrame()
	inputState.SetKey(input.KeyEnter, false)
	console.Update(ctx)
	if !sessionState.NoShift || console.Active() {
		t.Fatalf("after following frame no_shift=%t active=%t, want no duplicate submit", sessionState.NoShift, console.Active())
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

func TestConsoleScrollToBottomDoesNotNotifyWhenAlreadyThere(t *testing.T) {
	console := &ChatConsole{}
	for i := 0; i < 20; i++ {
		console.AddMessage("line %d", i)
	}
	console.widgetTree(480, 176)
	scrollY := console.ensureScrollSignal()
	notifications := 0
	unsubscribe := scrollY.SubscribeForever(func(float32) {
		notifications++
	})
	defer unsubscribe()

	console.scrollToBottom()

	if notifications != 0 {
		t.Fatalf("unchanged bottom scroll sent %d notifications, want 0", notifications)
	}
}

func TestConsoleMessageRedrawDefersOneUpdate(t *testing.T) {
	console := &ChatConsole{}
	ctx := client.Context{
		Input:     input.NewState(),
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
	}

	console.Update(ctx)
	initialKey := console.cacheKey
	if initialKey == "" {
		t.Fatal("console did not cache its initial content")
	}

	console.AddBlueMessage("You got Jellopy (1).")
	if !console.pendingMessageRedraw || console.pendingMessageRedrawReady {
		t.Fatalf("pending redraw = %t ready = %t, want pending not ready", console.pendingMessageRedraw, console.pendingMessageRedrawReady)
	}

	console.Update(ctx)
	if console.cacheKey != initialKey {
		t.Fatal("message redraw happened in the packet frame")
	}
	if !console.pendingMessageRedraw || !console.pendingMessageRedrawReady {
		t.Fatalf("pending redraw = %t ready = %t, want armed for next update", console.pendingMessageRedraw, console.pendingMessageRedrawReady)
	}

	console.Update(ctx)
	if console.cacheKey == initialKey {
		t.Fatal("message redraw was not flushed on the following update")
	}
	if console.pendingMessageRedraw || console.pendingMessageRedrawReady {
		t.Fatalf("pending redraw = %t ready = %t, want cleared", console.pendingMessageRedraw, console.pendingMessageRedrawReady)
	}
}

func TestConsolePresentationFlushesMessagesWithoutInputUpdate(t *testing.T) {
	console := &ChatConsole{}
	ctx := client.Context{
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
	}

	console.UpdatePresentation(ctx)
	initialKey := console.cacheKey
	console.AddErrorMessage("You cannot exit the game right now.")

	console.UpdatePresentation(ctx)
	if console.cacheKey != initialKey {
		t.Fatal("message redraw did not preserve its one-update deferral")
	}

	console.UpdatePresentation(ctx)
	if console.cacheKey == initialKey {
		t.Fatal("presentation maintenance did not flush the message redraw")
	}
	if console.pendingMessageRedraw || console.pendingMessageRedrawReady {
		t.Fatalf("pending redraw = %t ready = %t, want cleared", console.pendingMessageRedraw, console.pendingMessageRedrawReady)
	}
}

func TestConsoleMessagesKeyCachesUntilMessagesChange(t *testing.T) {
	console := &ChatConsole{}

	console.AddSystemMessage("ready")
	first := console.messagesKey()
	if first == "" {
		t.Fatal("messages key was empty")
	}
	if console.messagesKeyDirty {
		t.Fatal("messages key stayed dirty after rebuild")
	}

	console.AddBlueMessage("updated")
	if !console.messagesKeyDirty {
		t.Fatal("message append did not invalidate messages key")
	}
	second := console.messagesKey()
	if second == first {
		t.Fatal("messages key did not change after message append")
	}
	if console.messagesKeyDirty {
		t.Fatal("messages key stayed dirty after second rebuild")
	}
}

func TestConsoleMessageRedrawWaitsForStablePlayerMarker(t *testing.T) {
	console := &ChatConsole{}
	world := worldstate.New()
	world.MapName = "prt_fild08"
	world.Player.X = 10
	world.Player.Y = 10
	world.Player.Dir = 0
	ctx := client.Context{
		Input:     input.NewState(),
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
		World:     world,
	}

	console.Update(ctx)
	console.Update(ctx)
	console.Update(ctx)
	initialKey := console.cacheKey

	world.Player.X = 11
	console.AddBlueMessage("You got Apple (1).")
	console.Update(ctx)
	if console.cacheKey != initialKey {
		t.Fatal("message redraw flushed on the marker-change update")
	}

	console.Update(ctx)
	if console.cacheKey != initialKey {
		t.Fatal("message redraw flushed before the minimap marker frame could drain")
	}

	console.Update(ctx)
	if console.cacheKey == initialKey {
		t.Fatal("message redraw did not flush after the marker became stable")
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
